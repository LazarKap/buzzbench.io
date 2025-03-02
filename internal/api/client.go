package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// Client provides methods to interact with the BuzzBench API
type Client struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchPipelineTests retrieves all tests configured to run in the pipeline
func (c *Client) FetchPipelineTests() ([]TestConfiguration, error) {
	url := fmt.Sprintf("%s/tests/pipeline", c.BaseURL)
	req, err := c.newRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	var response APIResponse
	if err := c.do(req, &response); err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	return response.Tests, nil
}

// FetchTestByID retrieves a specific test configuration by ID
func (c *Client) FetchTestByID(testID string) (*TestConfiguration, error) {
	url := fmt.Sprintf("%s/tests/%s", c.BaseURL, testID)
	req, err := c.newRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	var test TestConfiguration
	if err := c.do(req, &test); err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}

	return &test, nil
}

// SubmitTestResult sends test results back to the API
func (c *Client) SubmitTestResult(result TestResult) error {
	url := fmt.Sprintf("%s/test-results", c.BaseURL)

	req, err := c.newRequest("POST", url, result)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if err := c.do(req, nil); err != nil {
		return fmt.Errorf("execute request: %w", err)
	}

	return nil
}

// newRequest creates a new HTTP request with common headers
func (c *Client) newRequest(method, url string, body interface{}) (*http.Request, error) {
	var buf bytes.Buffer

	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			return nil, fmt.Errorf("encode request body: %w", err)
		}
	}

	req, err := http.NewRequest(method, url, &buf)
	if err != nil {
		return nil, fmt.Errorf("create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.APIKey))

	return req, nil
}

// do executes an HTTP request and decodes the response
func (c *Client) do(req *http.Request, v interface{}) error {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response body: %w", err)
	}

	// Check for error status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	// If there's no response structure to decode into, we're done
	if v == nil {
		return nil
	}

	// Decode the JSON response
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	return nil
}
