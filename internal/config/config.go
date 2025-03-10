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
	APIKey      string
	BaseURL     string
	Verbose     bool
	SingleTest  bool
	TestID      string
	OutputJSON  bool
	JSONOutFile string
}

// DefaultBaseURL is the default API endpoint
const DefaultBaseURL = "https://buzzbench.io/api"

// EmbeddedApiKey can be set at compile time using -ldflags
var EmbeddedApiKey string

// New creates a new configuration with defaults and overrides from .env file, environment variables, and flags
func New() *Config {

	// If EmbeddedApiKey is set, overwrite everything and use this value
	if EmbeddedApiKey != "" {
		return &Config{
			APIKey:  EmbeddedApiKey,
			BaseURL: DefaultBaseURL,
		}
	}

	// Load .env file if it exists
	loadEnvFile(".env")

	cfg := &Config{
		BaseURL: getEnv("BUZZBENCH_API_URL", DefaultBaseURL),
		APIKey:  getEnv("BUZZBENCH_API_KEY", ""),
	}

	// Clean the BaseURL
	cfg.BaseURL = strings.TrimRight(cfg.BaseURL, "/")

	return cfg
}

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(filename string) {
	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return // File doesn't exist, just return
	}

	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		return // Couldn't open file, just return
	}
	defer file.Close()

	// Read line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || len(strings.TrimSpace(line)) == 0 {
			continue
		}

		// Parse KEY=VALUE format
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Don't override existing environment variables
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, value)
		}
	}
}

// ParseFlags parses command line flags and updates the configuration
func (c *Config) ParseFlags() {
	flag.StringVar(&c.APIKey, "api-key", c.APIKey, "API key for BuzzBench (env: BUZZBENCH_API_KEY)")
	flag.StringVar(&c.BaseURL, "base-url", c.BaseURL, "Base URL for the BuzzBench API (env: BUZZBENCH_API_URL)")
	flag.BoolVar(&c.Verbose, "verbose", false, "Enable verbose output")
	flag.BoolVar(&c.SingleTest, "test", false, "Run a single test by ID")
	flag.StringVar(&c.TestID, "id", "", "Test ID to run (requires -test flag)")
	flag.BoolVar(&c.OutputJSON, "json", false, "Output results as JSON")
	flag.StringVar(&c.JSONOutFile, "out", "", "Output JSON results to file")

	flag.Parse()

	// Validate configuration
	if c.APIKey == "" {
		fmt.Println("Warning: No API key provided. Set BUZZBENCH_API_KEY environment variable or use --api-key flag.")
	}

	if c.SingleTest && c.TestID == "" {
		fmt.Println("Error: -test flag requires -id parameter")
		flag.Usage()
		os.Exit(1)
	}
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	if c.APIKey == "" {
		return fmt.Errorf("API key is required")
	}
	return nil
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
