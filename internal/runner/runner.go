package runner

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

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

// RunTest executes a performance test based on the provided configuration
func (r *Runner) RunTest(config api.TestConfiguration) (api.TestResult, error) {
	r.logInfo("Starting test: %s", config.Name)
	r.logInfo("URL: %s", config.URL)
	r.logInfo("Method: %s", config.Method)
	r.logInfo("Requests: %d", config.Requests)
	r.logInfo("Concurrency: %d", config.Concurrency)

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

	// Create HTTP client with the configured timeout
	client := &http.Client{
		Timeout: time.Duration(config.TimeoutSecs) * time.Second,
	}

	resultChan := make(chan api.RequestResult, config.Requests)
	sem := make(chan bool, config.Concurrency)

	startTime := time.Now()

	var wg sync.WaitGroup

	for i := 0; i < config.Requests; i++ {
		wg.Add(1)
		sem <- true

		go func(reqNum int) {
			defer wg.Done()
			defer func() { <-sem }()

			r.logDebug("Executing request %d", reqNum)

			var req *http.Request
			var err error

			if config.Method == "GET" || config.Method == "DELETE" {
				req, err = http.NewRequest(config.Method, config.URL, nil)
			} else {
				var body *bytes.Buffer

				if config.Body != "" {
					body = bytes.NewBufferString(config.Body)
				} else {
					body = bytes.NewBufferString("{}")
				}

				req, err = http.NewRequest(config.Method, config.URL, body)
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
				req.Header.Set("Authorization", "Bearer "+config.AuthToken)
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
		}(i)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var totalDuration time.Duration
	minDuration := time.Hour // Start with a very large value
	maxDuration := time.Duration(0)
	successCount := 0
	totalCount := 0
	timelinePoints := make(map[int64][]float64) // Map of timestamp to durations for that second

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
