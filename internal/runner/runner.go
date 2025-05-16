package runner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/lazarkap/buzzbench.io/internal/api"
)

// Runner handles test execution
type Runner struct {
	Verbose bool
	Logger  *log.Logger
}

// NewRunner creates a new test runner
func NewRunner(verbose bool, logger *log.Logger) *Runner {
	return &Runner{
		Verbose: verbose,
		Logger:  logger,
	}
}

// Variable represents a test variable definition
type Variable struct {
	Name       string `json:"name"`
	Type       string `json:"type"`       // string, integer, float, boolean, uuid, timestamp
	Strategy   string `json:"strategy"`   // static, sequential, random, uuid, timestamp, template
	Value      string `json:"value"`      // for static
	StartValue int    `json:"startValue"` // for sequential
	EndValue   int    `json:"endValue"`   // for sequential
	Increment  int    `json:"increment"`  // for sequential
	MinValue   int    `json:"minValue"`   // for random
	MaxValue   int    `json:"maxValue"`   // for random
	Template   string `json:"template"`   // for template
	current    int    // internal counter for sequential
}

// VariableContext holds the current state for variable generation
type VariableContext struct {
	Variables    map[string]*Variable
	RequestIndex int
	Rand         *rand.Rand // Pre-seeded random generator
	Mutex        sync.Mutex // For thread-safe updates
}

// RunTest executes a performance test based on the provided configuration
func (r *Runner) RunTest(config api.TestConfiguration) (api.TestResult, error) {
	r.logInfo("Starting test: %s", config.Name)
	r.logInfo("URL: %s", config.URL)
	r.logInfo("Method: %s", config.Method)
	r.logInfo("Requests: %d", config.Requests)
	r.logInfo("Concurrency: %d", config.Concurrency)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(config.Requests/config.Concurrency+10)*time.Second,
	)
	defer cancel()

	// Initialize variable context if needed
	var varCtx *VariableContext
	if config.UseVariables {
		r.logInfo("Using variables for this test")
		varCtx = r.setupVariableContext(config)
	}

	result := api.TestResult{
		TestConfigurationID: config.ID,
		URL:                 config.URL,
		Method:              config.Method,
		Requests:            config.Requests,
		Concurrency:         config.Concurrency,
		StatusCodes:         make(map[string]int),
		Errors:              []api.ErrorData{},
		Timeline:            []api.TimelinePoint{},
	}

	// Buffered channels to prevent blocking
	resultChan := make(chan api.RequestResult, config.Requests)
	requestChan := make(chan int, config.Requests)

	// Prepare request indices
	go func() {
		defer close(requestChan)
		for i := 0; i < config.Requests; i++ {
			select {
			case requestChan <- i:
			case <-ctx.Done():
				return
			}
		}
	}()

	// Worker pool with proper synchronization
	var wg sync.WaitGroup
	for i := 0; i < config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case reqIdx, ok := <-requestChan:
					if !ok {
						return // Channel closed
					}
					r.executeRequest(ctx, config, reqIdx, varCtx, resultChan)
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	// Close result channel when all workers are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	startTime := time.Now()
	var totalDuration time.Duration
	minDuration := time.Hour // Start with a very large value
	maxDuration := time.Duration(0)
	successCount := 0
	totalCount := 0
	timelinePoints := make(map[int64][]float64)

	// Process results
	for res := range resultChan {
		totalCount++

		if res.Error != nil {
			result.Errors = append(result.Errors, api.ErrorData{
				Message: res.Error.Error(),
			})
			continue
		}

		statusKey := fmt.Sprintf("%d", res.Status)
		result.StatusCodes[statusKey]++

		// Consider 2xx and 3xx as success
		if res.Status >= 200 && res.Status < 400 {
			successCount++
		} else {
			result.Errors = append(result.Errors, api.ErrorData{
				Status:  statusKey,
				Message: http.StatusText(res.Status),
			})
		}

		totalDuration += res.Duration

		if res.Duration < minDuration {
			minDuration = res.Duration
		}

		if res.Duration > maxDuration {
			maxDuration = res.Duration
		}

		second := res.Timestamp.Unix()
		timelinePoints[second] = append(timelinePoints[second], float64(res.Duration.Milliseconds()))
	}

	totalTestDuration := time.Since(startTime)

	if totalCount > 0 {
		result.SuccessRate = float64(successCount) / float64(totalCount) * 100
		result.AvgResponseTime = float64(totalDuration.Milliseconds()) / float64(totalCount)
		result.RequestsPerSecond = float64(totalCount) / totalTestDuration.Seconds()

		if successCount > 0 {
			result.MinResponseTime = float64(minDuration.Milliseconds())
			result.MaxResponseTime = float64(maxDuration.Milliseconds())
		}
	}

	// Process timeline data
	for second, durations := range timelinePoints {
		var sum float64
		for _, d := range durations {
			sum += d
		}
		avg := sum / float64(len(durations))

		// Add to timeline
		result.Timeline = append(result.Timeline, api.TimelinePoint{
			Timestamp:    float64(second),
			ResponseTime: avg,
			ActiveUsers:  float64(len(durations)),
		})
	}

	r.logInfo("Test completed successfully")
	r.logInfo("Success Rate: %.2f%%", result.SuccessRate)
	r.logInfo("Avg Response Time: %.2f ms", result.AvgResponseTime)
	r.logInfo("Min Response Time: %.2f ms", result.MinResponseTime)
	r.logInfo("Max Response Time: %.2f ms", result.MaxResponseTime)
	r.logInfo("Requests Per Second: %.2f", result.RequestsPerSecond)

	return result, nil
}

// executeRequest handles the execution of a single request
func (r *Runner) executeRequest(
	ctx context.Context,
	config api.TestConfiguration,
	reqIdx int,
	varCtx *VariableContext,
	resultChan chan<- api.RequestResult,
) {
	select {
	case <-ctx.Done():
		return
	default:
		// Create HTTP client with the configured timeout
		client := &http.Client{
			Timeout: time.Duration(config.TimeoutSecs) * time.Second,
		}

		// Apply variables to URL and body if needed
		reqURL := config.URL
		reqBody := config.Body

		if config.UseVariables && varCtx != nil {
			// Process URL with variables
			var err error
			reqURL, err = r.processVariables(reqURL, varCtx, reqIdx)
			if err != nil {
				resultChan <- api.RequestResult{
					Duration:  0,
					Status:    0,
					Error:     err,
					Timestamp: time.Now(),
				}
				return
			}

			// Process body with variables if applicable
			if config.Method == "POST" || config.Method == "PUT" || config.Method == "PATCH" {
				reqBody, err = r.processVariables(reqBody, varCtx, reqIdx)
				if err != nil {
					resultChan <- api.RequestResult{
						Duration:  0,
						Status:    0,
						Error:     err,
						Timestamp: time.Now(),
					}
					return
				}
			}
		}

		var req *http.Request
		var err error

		if config.Method == "GET" || config.Method == "DELETE" {
			req, err = http.NewRequestWithContext(ctx, config.Method, reqURL, nil)
		} else {
			var body *bytes.Buffer

			if reqBody != "" {
				body = bytes.NewBufferString(reqBody)
			} else {
				body = bytes.NewBufferString("{}")
			}

			req, err = http.NewRequestWithContext(ctx, config.Method, reqURL, body)
			req.Header.Set("Content-Type", "application/json")
		}

		if err != nil {
			resultChan <- api.RequestResult{
				Duration:  0,
				Status:    0,
				Error:     err,
				Timestamp: time.Now(),
			}
			return
		}

		if config.AuthToken != "" {
			req.Header.Set("Authorization", config.AuthToken)
		}

		reqStart := time.Now()
		resp, err := client.Do(req)
		reqDuration := time.Since(reqStart)

		result := api.RequestResult{
			Duration:  reqDuration,
			Timestamp: reqStart,
		}

		if err != nil {
			result.Error = err
		} else {
			result.Status = resp.StatusCode
			resp.Body.Close()
		}

		resultChan <- result
	}
}

// setupVariableContext initializes the variable context for the test
func (r *Runner) setupVariableContext(config api.TestConfiguration) *VariableContext {
	// Create a new random source with a seed based on current time
	source := rand.NewSource(time.Now().UnixNano())
	rnd := rand.New(source)

	ctx := &VariableContext{
		Variables: make(map[string]*Variable),
		Rand:      rnd,
		Mutex:     sync.Mutex{},
	}

	// Parse variables JSON
	var variables []*Variable
	if err := json.Unmarshal([]byte(config.Variables), &variables); err != nil {
		r.logInfo("Error parsing variables: %v", err)
		return ctx
	}

	// Initialize variables
	for _, v := range variables {
		// Set defaults if needed
		if v.Strategy == "sequential" && v.Increment <= 0 {
			v.Increment = 1
		}
		if v.Strategy == "sequential" {
			v.current = v.StartValue
		}
		ctx.Variables[v.Name] = v
	}

	return ctx
}

// processVariables replaces variables in a string with their values
func (r *Runner) processVariables(input string, ctx *VariableContext, requestIndex int) (string, error) {
	if input == "" {
		return input, nil
	}

	// Find all variable placeholders
	re := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	result := re.ReplaceAllStringFunc(input, func(match string) string {
		// Extract variable name from {{name}}
		varName := match[2 : len(match)-2]
		varValue, err := r.getVariableValue(varName, ctx, requestIndex)
		if err != nil {
			r.logInfo("Variable error: %v", err)
			return match // Return original if error
		}
		return varValue
	})

	return result, nil
}

// getVariableValue generates a value for a variable based on its definition
func (r *Runner) getVariableValue(name string, ctx *VariableContext, requestIndex int) (string, error) {
	// Special built-in variables
	if name == "$index" {
		return strconv.Itoa(requestIndex), nil
	} else if name == "$random" {
		return strconv.Itoa(ctx.Rand.Intn(10000)), nil
	}

	// Look up the variable definition
	v, exists := ctx.Variables[name]
	if !exists {
		return "", fmt.Errorf("undefined variable: %s", name)
	}

	// Process based on strategy
	switch v.Strategy {
	case "static":
		return v.Value, nil

	case "sequential":
		// Safely get and increment the value
		ctx.Mutex.Lock()
		current := v.current

		// Increment for next use
		v.current += v.Increment

		// Handle wrapping around if we exceed end value
		if v.EndValue > v.StartValue && v.current > v.EndValue {
			v.current = v.StartValue
		}
		ctx.Mutex.Unlock()

		return strconv.Itoa(current), nil

	case "random":
		if v.Type == "integer" {
			return strconv.Itoa(ctx.Rand.Intn(v.MaxValue-v.MinValue+1) + v.MinValue), nil
		} else if v.Type == "float" {
			val := float64(v.MinValue) + ctx.Rand.Float64()*float64(v.MaxValue-v.MinValue)
			return fmt.Sprintf("%.2f", val), nil
		} else {
			// For non-numeric, generate a random string
			return fmt.Sprintf("random-%d", ctx.Rand.Intn(10000)), nil
		}

	case "uuid":
		return uuid.New().String(), nil

	case "timestamp":
		return time.Now().Format(time.RFC3339), nil

	case "template":
		// Process template
		template := v.Template
		template = strings.ReplaceAll(template, "{{$index}}", strconv.Itoa(requestIndex))
		template = strings.ReplaceAll(template, "{{$random}}", strconv.Itoa(ctx.Rand.Intn(10000)))
		return template, nil

	default:
		return "", fmt.Errorf("unsupported variable strategy: %s", v.Strategy)
	}
}

// logInfo logs information if verbose mode is enabled
func (r *Runner) logInfo(format string, v ...interface{}) {
	r.Logger.Printf(format, v...)
}

// logDebug logs debug information if verbose mode is enabled
func (r *Runner) logDebug(format string, v ...interface{}) {
	if r.Verbose {
		r.Logger.Printf("[DEBUG] "+format, v...)
	}
}
