# BuzzBench

A powerful, flexible API performance testing tool designed to work with the BuzzBench dashboard.

## Features

- **High Concurrency Testing** - Simulate hundreds or thousands of concurrent users
- **Flexible Configuration** - Customizable requests, methods, headers, and bodies
- **Detailed Analytics** - Comprehensive metrics including response time, throughput, and error rates
- **Pipeline Integration** - Run tests automatically as part of your CI/CD workflow
- **JSON Output** - Export results for custom processing

## Installation

### Using Go Install

```bash
go install github.com/lazarkap/buzzbench.io/cmd/buzzbench@latest
```

### From Source

```bash
git clone https://github.com/lazarkap/buzzbench.io.git
cd buzzbench
go build -o buzzbench ./cmd/buzzbench
```

## Usage

### Environment Setup

Set your API key as an environment variable:

```bash
export BUZZBENCH_API_KEY=your_api_key_here
```

Optionally set a custom API URL:

```bash
export BUZZBENCH_API_URL=https://buzzbench.io/api
```

### Running Tests

Run all pipeline-enabled tests:

```bash
buzzbench
```

Run a specific test by ID:

```bash
buzzbench -test -id test-123
```

Enable verbose logging:

```bash
buzzbench -verbose
```

### JSON Output

Output results as JSON:

```bash
buzzbench -json
```

Save JSON results to a file:

```bash
buzzbench -json -out results.json
```

## Command Line Options

```
Usage of buzzbench:
  -api-key string
        API key for BuzzBench (env: BUZZBENCH_API_KEY)
  -base-url string
        Base URL for the BuzzBench API (env: BUZZBENCH_API_URL)
  -id string
        Test ID to run (requires -test flag)
  -json
        Output results as JSON
  -out string
        Output JSON results to file
  -test
        Run a single test by ID
  -verbose
        Enable verbose output
```

## Example

```bash
# Run a performance test and analyze the results
buzzbench -test -id perf-test-001 -verbose
```

## Output Example

```
BuzzBench - API Performance Testing Tool
----------------------------------------
[1/1] Running test: API Health Check

=== TEST SUMMARY ===
URL: https://api.example.com/health
Method: GET
Requests: 1000
Concurrency: 10
Success Rate: 99.80%
Avg Response Time: 32.54 ms
Min Response Time: 12.30 ms
Max Response Time: 187.60 ms
Requests Per Second: 289.45

=== STATUS CODES ===
  200: 998 (99.8%) - Success
  503: 2 (0.2%) - Server Error
```

## License

MIT License - see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request