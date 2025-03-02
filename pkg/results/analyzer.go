package results

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"sort"
	"strings"

	"github.com/lazarkap/buzzbench.io/internal/api"
)

// Analyzer provides methods for analyzing test results
type Analyzer struct {
	Result api.TestResult
}

// NewAnalyzer creates a new results analyzer
func NewAnalyzer(result api.TestResult) *Analyzer {
	return &Analyzer{
		Result: result,
	}
}

// PrintSummary prints a summary of the test results to stdout
func (a *Analyzer) PrintSummary() {
	fmt.Println("\n=== TEST SUMMARY ===")
	fmt.Printf("URL: %s\n", a.Result.URL)
	fmt.Printf("Method: %s\n", a.Result.Method)
	fmt.Printf("Requests: %d\n", a.Result.Requests)
	fmt.Printf("Concurrency: %d\n", a.Result.Concurrency)
	fmt.Printf("Success Rate: %.2f%%\n", a.Result.SuccessRate)
	fmt.Printf("Avg Response Time: %.2f ms\n", a.Result.AvgResponseTime)
	fmt.Printf("Min Response Time: %.2f ms\n", a.Result.MinResponseTime)
	fmt.Printf("Max Response Time: %.2f ms\n", a.Result.MaxResponseTime)
	fmt.Printf("Requests Per Second: %.2f\n", a.Result.RequestsPerSecond)

	fmt.Println("\n=== STATUS CODES ===")
	a.printStatusCodes()

	if len(a.Result.Errors) > 0 {
		fmt.Println("\n=== ERRORS ===")
		a.printErrors()
	}
}

// SaveJSON saves the test results to a JSON file
func (a *Analyzer) SaveJSON(filePath string) error {
	data, err := json.MarshalIndent(a.Result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// printStatusCodes prints the status code distribution
func (a *Analyzer) printStatusCodes() {
	if len(a.Result.StatusCodes) == 0 {
		fmt.Println("No status codes recorded")
		return
	}

	// Sort status codes for consistent output
	var codes []string
	for code := range a.Result.StatusCodes {
		codes = append(codes, code)
	}
	sort.Strings(codes)

	totalRequests := a.Result.Requests

	for _, code := range codes {
		count := a.Result.StatusCodes[code]
		percentage := float64(count) / float64(totalRequests) * 100

		codeType := getStatusCodeType(code)
		fmt.Printf("  %s: %d (%.1f%%) - %s\n", code, count, percentage, codeType)
	}
}

// printErrors prints error information
func (a *Analyzer) printErrors() {
	if len(a.Result.Errors) == 0 {
		return
	}

	// Group errors by message
	errorCounts := make(map[string]int)
	for _, err := range a.Result.Errors {
		key := fmt.Sprintf("%s: %s", err.Status, err.Message)
		errorCounts[key]++
	}

	// Sort error messages for consistent output
	var errorMessages []string
	for msg := range errorCounts {
		errorMessages = append(errorMessages, msg)
	}
	sort.Strings(errorMessages)

	// Print errors with counts
	for _, msg := range errorMessages {
		count := errorCounts[msg]
		fmt.Printf("  [%d occurrences] %s\n", count, msg)
	}
}

// GetStatusCodeCounts returns counts grouped by status code type
func (a *Analyzer) GetStatusCodeCounts() map[string]int {
	result := map[string]int{
		"success":     0,
		"redirection": 0,
		"clientError": 0,
		"serverError": 0,
		"unknown":     0,
	}

	for code, count := range a.Result.StatusCodes {
		codeType := getStatusCodeType(code)
		switch {
		case strings.Contains(codeType, "Success"):
			result["success"] += count
		case strings.Contains(codeType, "Redirection"):
			result["redirection"] += count
		case strings.Contains(codeType, "Client Error"):
			result["clientError"] += count
		case strings.Contains(codeType, "Server Error"):
			result["serverError"] += count
		default:
			result["unknown"] += count
		}
	}

	return result
}

// GetPerformanceGrade evaluates the test performance
func (a *Analyzer) GetPerformanceGrade() string {
	// Calculate score based on success rate and response time
	successScore := a.Result.SuccessRate / 100 * 50 // 50% of score from success rate

	// Response time score (lower is better)
	// Assuming < 100ms is excellent, > 1000ms is poor
	var responseTimeScore float64
	if a.Result.AvgResponseTime <= 100 {
		responseTimeScore = 50 // 50% of score from response time
	} else if a.Result.AvgResponseTime >= 1000 {
		responseTimeScore = 0
	} else {
		// Linear scale between 100ms and 1000ms
		responseTimeScore = 50 * (1 - (a.Result.AvgResponseTime-100)/900)
	}

	totalScore := successScore + responseTimeScore

	// Assign grade based on total score
	switch {
	case totalScore >= 90:
		return "A"
	case totalScore >= 80:
		return "B"
	case totalScore >= 70:
		return "C"
	case totalScore >= 60:
		return "D"
	default:
		return "F"
	}
}

// getStatusCodeType returns a human-readable status code type
func getStatusCodeType(code string) string {
	// Remove all non-digit characters to handle format variations
	codeDigits := code
	if len(codeDigits) > 0 {
		firstChar := codeDigits[0]
		switch firstChar {
		case '1':
			return "Informational"
		case '2':
			return "Success"
		case '3':
			return "Redirection"
		case '4':
			return "Client Error"
		case '5':
			return "Server Error"
		}
	}
	return "Unknown"
}
