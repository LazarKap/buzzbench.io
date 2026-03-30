package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/lazarkap/buzzbench.io/internal/api"
	"github.com/lazarkap/buzzbench.io/internal/config"
	"github.com/lazarkap/buzzbench.io/internal/runner"
	"github.com/lazarkap/buzzbench.io/pkg/results"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	cfg := config.New()
	cfg.ParseFlags()

	client := api.NewClient(cfg.BaseURL, cfg.APIKey)
	testRunner := runner.NewRunner(cfg.Verbose, logger)

	fmt.Println("BuzzBench - API Performance Testing Tool")
	fmt.Println("----------------------------------------")

	var (
		tests []api.TestConfiguration
		err   error
	)

	switch {
	case cfg.LocalURL != "":
		// Mode 1: single test from CLI flags
		tests = []api.TestConfiguration{{
			ID:          "local",
			Name:        cfg.LocalName,
			URL:         cfg.LocalURL,
			Method:      cfg.LocalMethod,
			Requests:    cfg.LocalReqs,
			Concurrency: cfg.LocalConc,
			TimeoutSecs: cfg.LocalTO,
			Body:        cfg.LocalBody,
			AuthToken:   cfg.LocalAuth,
		}}

	case cfg.ConfigFile != "":
		// Mode 2: tests from a local JSON file
		tests, err = loadConfigFile(cfg.ConfigFile)
		if err != nil {
			logger.Fatalf("Error loading config file: %v", err)
		}

	case cfg.SingleTest:
		// Mode 3a: fetch a single test from the API by ID
		var test *api.TestConfiguration
		test, err = client.FetchTestByID(cfg.TestID)
		if err != nil {
			logger.Fatalf("Error fetching test: %v", err)
		}
		tests = []api.TestConfiguration{*test}

	default:
		// Mode 3b: fetch all pipeline tests from the API
		tests, err = client.FetchPipelineTests()
		if err != nil {
			logger.Fatalf("Error fetching pipeline tests: %v", err)
		}
	}

	if len(tests) == 0 {
		logger.Println("No tests to run. Exiting.")
		os.Exit(0)
	}

	var allResults []api.TestResult

	for i, test := range tests {
		fmt.Printf("\n[%d/%d] %s\n", i+1, len(tests), test.Name)

		result, err := testRunner.RunTest(test)
		if err != nil {
			logger.Printf("Error running test: %v", err)
			continue
		}

		analyzer := results.NewAnalyzer(result)

		if cfg.OutputJSON {
			allResults = append(allResults, result)
		} else {
			analyzer.PrintSummary()
		}

		// Only submit results to the API when in API mode and not doing JSON-only output
		if !cfg.IsLocalMode() && !cfg.OutputJSON {
			logger.Printf("Submitting test results to %s", cfg.BaseURL)
			if err := client.SubmitTestResult(result); err != nil {
				logger.Printf("Error submitting results: %v", err)
			} else {
				logger.Printf("Test results submitted successfully")
			}
		}

		if i < len(tests)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	// Handle JSON output
	if cfg.OutputJSON && len(allResults) > 0 {
		var output []byte

		if len(allResults) == 1 {
			output, err = json.MarshalIndent(allResults[0], "", "  ")
		} else {
			output, err = json.MarshalIndent(allResults, "", "  ")
		}
		if err != nil {
			logger.Fatalf("Error encoding JSON: %v", err)
		}

		if cfg.JSONOutFile != "" {
			if err := os.WriteFile(cfg.JSONOutFile, output, 0644); err != nil {
				logger.Fatalf("Error writing output file: %v", err)
			}
			logger.Printf("Results saved to %s", cfg.JSONOutFile)
		} else {
			fmt.Println(string(output))
		}
	}
}

// localVariable mirrors api.Variable but is used only for local config file parsing,
// where variables are a proper JSON array instead of an escaped JSON string.
type localVariable struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Strategy   string `json:"strategy"`
	Value      string `json:"value,omitempty"`
	StartValue int    `json:"startValue,omitempty"`
	EndValue   int    `json:"endValue,omitempty"`
	Increment  int    `json:"increment,omitempty"`
	MinValue   int    `json:"minValue,omitempty"`
	MaxValue   int    `json:"maxValue,omitempty"`
	Template   string `json:"template,omitempty"`
}

// localTest is the schema for entries in a local config file.
// Variables is a proper array — no escaped JSON string needed.
// use_variables is inferred automatically when variables are present.
type localTest struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	URL         string          `json:"url"`
	Method      string          `json:"method"`
	Requests    int             `json:"requests"`
	Concurrency int             `json:"concurrency"`
	TimeoutSecs int             `json:"timeout_seconds"`
	AuthToken   string          `json:"auth_token,omitempty"`
	Body        string          `json:"body,omitempty"`
	Variables   []localVariable `json:"variables,omitempty"`
	Description string          `json:"description,omitempty"`
}

func (lt localTest) toTestConfiguration() (api.TestConfiguration, error) {
	tc := api.TestConfiguration{
		ID:          lt.ID,
		Name:        lt.Name,
		URL:         lt.URL,
		Method:      lt.Method,
		Requests:    lt.Requests,
		Concurrency: lt.Concurrency,
		TimeoutSecs: lt.TimeoutSecs,
		AuthToken:   lt.AuthToken,
		Body:        lt.Body,
		Description: lt.Description,
	}

	if len(lt.Variables) > 0 {
		tc.UseVariables = true
		varJSON, err := json.Marshal(lt.Variables)
		if err != nil {
			return tc, fmt.Errorf("marshal variables: %w", err)
		}
		tc.Variables = string(varJSON)
	}

	return tc, nil
}

// loadConfigFile reads a JSON file containing an array of localTest definitions
// and converts them to api.TestConfiguration values the runner understands.
func loadConfigFile(path string) ([]api.TestConfiguration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", path, err)
	}

	var raw []localTest
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse %q: %w", path, err)
	}

	tests := make([]api.TestConfiguration, 0, len(raw))
	for _, lt := range raw {
		tc, err := lt.toTestConfiguration()
		if err != nil {
			return nil, fmt.Errorf("test %q: %w", lt.Name, err)
		}
		tests = append(tests, tc)
	}
	return tests, nil
}
