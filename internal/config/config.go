package config

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"
)

// Config holds the application configuration
type Config struct {
	// API mode
	APIKey     string
	BaseURL    string
	SingleTest bool
	TestID     string

	// Output
	Verbose     bool
	OutputJSON  bool
	JSONOutFile string

	// Local flag mode (-url ...)
	LocalURL    string
	LocalMethod string
	LocalReqs   int
	LocalConc   int
	LocalTO     int
	LocalBody   string
	LocalAuth   string
	LocalName   string

	// Local config-file mode (-config ...)
	ConfigFile string
}

// IsLocalMode returns true when no BuzzBench API calls should be made.
func (c *Config) IsLocalMode() bool {
	return c.LocalURL != "" || c.ConfigFile != ""
}

// DefaultBaseURL is the default API endpoint
const DefaultBaseURL = "https://buzzbench.io/api"

// EmbeddedApiKey can be set at compile time using -ldflags
var EmbeddedApiKey string

// New creates a new configuration with defaults loaded from .env / environment.
func New() *Config {
	if EmbeddedApiKey != "" {
		return &Config{
			APIKey:  EmbeddedApiKey,
			BaseURL: DefaultBaseURL,
		}
	}

	loadEnvFile(".env")

	cfg := &Config{
		BaseURL: getEnv("BUZZBENCH_API_URL", DefaultBaseURL),
		APIKey:  getEnv("BUZZBENCH_API_KEY", ""),
	}
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	return cfg
}

// ParseFlags parses command line flags and updates the configuration.
func (c *Config) ParseFlags() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, `BuzzBench - API Performance Testing Tool

USAGE:
  buzzbench [FLAGS]

MODES:

  1. Local flag mode  (no API key required)
       Test a single URL directly from the command line.

       buzzbench -url http://localhost:8000/health
       buzzbench -url http://localhost:8000/users -requests 500 -concurrency 50
       buzzbench -url http://localhost:8000/orders -method POST -body '{"qty":1}' -requests 100

  2. Local config-file mode  (no API key required)
       Run multiple tests defined in a JSON file.

       buzzbench -config tests.json
       buzzbench -config tests.json -out results.json

  3. API mode  (requires BUZZBENCH_API_KEY)
       Fetch and run tests from the BuzzBench.io dashboard.

       buzzbench
       buzzbench -test -id <test-id>

FLAGS:

  Local test flags:
    -url string        Target URL to test (enables local flag mode)
    -name string       Label shown in the test summary  (default "Quick Test")
    -method string     HTTP method: GET POST PUT DELETE PATCH  (default "GET")
    -requests int      Total number of requests to send  (default 100)
    -concurrency int   Number of concurrent workers  (default 10)
    -timeout int       Per-request timeout in seconds  (default 30)
    -body string       Request body (for POST / PUT / PATCH)
    -auth string       Authorization header value

  Config-file flag:
    -config string     Path to a JSON test config file

  Output flags:
    -out string        Save results as JSON to this file
    -json              Print results as JSON to stdout
    -verbose           Enable verbose logging

  API flags:
    -api-key string    BuzzBench API key  (env: BUZZBENCH_API_KEY)
    -base-url string   BuzzBench API base URL  (env: BUZZBENCH_API_URL)
    -test              Run a single API test by ID (requires -id)
    -id string         Test ID to run

`)
	}

	// API flags
	flag.StringVar(&c.APIKey,     "api-key",  c.APIKey,  "API key for BuzzBench (env: BUZZBENCH_API_KEY)")
	flag.StringVar(&c.BaseURL,    "base-url", c.BaseURL, "Base URL for the BuzzBench API (env: BUZZBENCH_API_URL)")
	flag.BoolVar  (&c.SingleTest, "test",     false,     "Run a single test by ID (API mode)")
	flag.StringVar(&c.TestID,     "id",       "",        "Test ID to run (requires -test)")

	// Output flags
	flag.BoolVar  (&c.Verbose,     "verbose", false, "Enable verbose output")
	flag.BoolVar  (&c.OutputJSON,  "json",    false, "Print results as JSON to stdout")
	flag.StringVar(&c.JSONOutFile, "out",     "",    "Save results as JSON to file")

	// Local flag mode
	flag.StringVar(&c.LocalURL,    "url",         "",           "Target URL (enables local flag mode)")
	flag.StringVar(&c.LocalName,   "name",        "Quick Test", "Test label")
	flag.StringVar(&c.LocalMethod, "method",      "GET",        "HTTP method")
	flag.IntVar   (&c.LocalReqs,   "requests",    100,          "Number of requests")
	flag.IntVar   (&c.LocalConc,   "concurrency", 10,           "Concurrent workers")
	flag.IntVar   (&c.LocalTO,     "timeout",     30,           "Per-request timeout (seconds)")
	flag.StringVar(&c.LocalBody,   "body",        "",           "Request body JSON")
	flag.StringVar(&c.LocalAuth,   "auth",        "",           "Authorization header value")

	// Local config-file mode
	flag.StringVar(&c.ConfigFile, "config", "", "Path to JSON test config file")

	flag.Parse()

	// -out implies -json
	if c.JSONOutFile != "" {
		c.OutputJSON = true
	}

	// API key is only required in API mode
	if !c.IsLocalMode() && c.APIKey == "" {
		fmt.Fprintln(os.Stderr, "Warning: No API key provided. Set BUZZBENCH_API_KEY or use -api-key flag.")
	}

	if c.SingleTest && c.TestID == "" {
		fmt.Fprintln(os.Stderr, "Error: -test flag requires -id parameter")
		flag.Usage()
		os.Exit(1)
	}
}

// Validate checks if the configuration is valid for API mode.
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

// loadEnvFile loads environment variables from a .env file (does not override existing env vars).
func loadEnvFile(filename string) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return
	}
	file, err := os.Open(filename)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || len(strings.TrimSpace(line)) == 0 {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}

// getEnv retrieves an environment variable or returns a default value.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
