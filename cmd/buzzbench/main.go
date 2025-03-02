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
	// Initialize logger
	logger := log.New(os.Stdout, "", log.LstdFlags)

	// Initialize configuration
	cfg := config.New()
	cfg.ParseFlags()

	// Create API client
	client := api.NewClient(cfg.BaseURL, cfg.APIKey)

	// Create test runner
	testRunner := runner.NewRunner(cfg.Verbose, logger)

	// Show banner
	fmt.Println("BuzzBench - API Performance Testing Tool")
	fmt.Println("----------------------------------------")

	var tests []api.TestConfiguration
	var err error

	// Fetch tests
	if cfg.SingleTest {
		// Run a single test by ID
		var test *api.TestConfiguration
		logger.Printf("Fetching test with ID: %s", cfg.TestID)

		test, err = client.FetchTestByID(cfg.TestID)
		if err != nil {
			logger.Fatalf("Error fetching test: %v", err)
		}

		tests = []api.TestConfiguration{*test}
	} else {
		// Run all pipeline tests
		logger.Printf("Fetching pipeline tests from %s", cfg.BaseURL)

		tests, err = client.FetchPipelineTests()
		if err != nil {
			logger.Fatalf("Error fetching pipeline tests: %v", err)
		}
	}

	logger.Printf("Found %d tests to run", len(tests))

	if len(tests) == 0 {
		logger.Println("No tests to run. Exiting.")
		os.Exit(0)
	}

	// Process results for JSON output if needed
	var allResults []api.TestResult

	// Run each test
	for i, test := range tests {
		logger.Printf("\n[%d/%d] Running test: %s", i+1, len(tests), test.Name)

		result, err := testRunner.RunTest(test)
		if err != nil {
			logger.Printf("Error running test: %v", err)
			continue
		}

		// Create analyzer for results
		analyzer := results.NewAnalyzer(result)

		// Print test summary
		if !cfg.OutputJSON {
			analyzer.PrintSummary()
		} else {
			// Store for JSON output
			allResults = append(allResults, result)
		}

		// Submit test results to API if not running in JSON-only mode
		if !cfg.OutputJSON {
			logger.Printf("Submitting test results to %s", cfg.BaseURL)
			if err := client.SubmitTestResult(result); err != nil {
				logger.Printf("Error submitting results: %v", err)
			} else {
				logger.Printf("Test results submitted successfully")
			}
		}

		// Wait a short time between tests
		if i < len(tests)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	// Handle JSON output if requested
	if cfg.OutputJSON {
		var output []byte
		var err error

		if len(allResults) == 1 {
			// Single test result
			output, err = json.MarshalIndent(allResults[0], "", "  ")
		} else {
			// Multiple test results
			output, err = json.MarshalIndent(allResults, "", "  ")
		}

		if err != nil {
			logger.Fatalf("Error encoding JSON results: %v", err)
		}

		fmt.Println(string(output))
	}
}
