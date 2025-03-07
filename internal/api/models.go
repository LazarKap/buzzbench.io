package api

import (
	"time"
)

// TestConfiguration holds the configuration for a performance test
type TestConfiguration struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	URL           string `json:"url"`
	Method        string `json:"method"`
	Requests      int    `json:"requests"`
	Concurrency   int    `json:"concurrency"`
	TimeoutSecs   int    `json:"timeout_seconds"`
	AuthToken     string `json:"auth_token,omitempty"`
	Body          string `json:"body,omitempty"`
	RunInPipeline bool   `json:"run_in_pipeline"`
	UseVariables  bool   `json:"use_variables"`       // Whether to use dynamic variables
	Variables     string `json:"variables,omitempty"` // JSON string for variable definitions
	Description   string `json:"description,omitempty"`
}

// Variable represents a definition of a dynamic variable
type Variable struct {
	Name       string `json:"name"`
	Type       string `json:"type"`                 // string, integer, float, boolean, uuid, timestamp
	Strategy   string `json:"strategy"`             // static, sequential, random, uuid, timestamp, template
	Value      string `json:"value,omitempty"`      // for static
	StartValue int    `json:"startValue,omitempty"` // for sequential
	EndValue   int    `json:"endValue,omitempty"`   // for sequential
	Increment  int    `json:"increment,omitempty"`  // for sequential
	MinValue   int    `json:"minValue,omitempty"`   // for random
	MaxValue   int    `json:"maxValue,omitempty"`   // for random
	Template   string `json:"template,omitempty"`   // for template
}

// TestResult contains the outcome of a performance test
type TestResult struct {
	TestConfigurationID string          `json:"test_configuration_id"`
	URL                 string          `json:"url"`
	Method              string          `json:"method"`
	Requests            int             `json:"requests"`
	Concurrency         int             `json:"concurrency"`
	SuccessRate         float64         `json:"success_rate"`
	AvgResponseTime     float64         `json:"avg_response_time"`
	MinResponseTime     float64         `json:"min_response_time"`
	MaxResponseTime     float64         `json:"max_response_time"`
	RequestsPerSecond   float64         `json:"requests_per_second"`
	StatusCodes         map[string]int  `json:"status_codes"`
	Errors              []ErrorData     `json:"errors,omitempty"`
	Timeline            []TimelinePoint `json:"timeline,omitempty"`
}

// RequestResult represents the result of a single HTTP request
type RequestResult struct {
	Duration  time.Duration
	Status    int
	Error     error
	Timestamp time.Time
}

// ErrorData represents error information
type ErrorData struct {
	Status  string `json:"status,omitempty"`
	Message string `json:"message"`
}

// TimelinePoint represents a data point in the test timeline
type TimelinePoint struct {
	Timestamp    float64 `json:"timestamp"`
	ResponseTime float64 `json:"response_time"`
	ActiveUsers  float64 `json:"active_users"`
}

// APIResponse is a generic API response structure
type APIResponse struct {
	Tests []TestConfiguration `json:"tests"`
}
