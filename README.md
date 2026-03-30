# BuzzBench

A powerful, flexible API performance testing tool.

Works standalone from the command line — no account or dashboard required.
Optionally integrates with the BuzzBench.io dashboard for storing and visualizing results.

## Features

- **High Concurrency Testing** — simulate hundreds or thousands of concurrent users
- **Flexible Configuration** — CLI flags for quick tests, JSON files for multi-test suites
- **Dynamic Variables** — vary URLs and request bodies across requests with sequential, random, UUID, timestamp, and template strategies
- **Detailed Analytics** — response time (avg / min / max), throughput, success rate, status code breakdown
- **JSON Output** — save results to a file for custom processing or CI assertions
- **Pipeline Integration** — run tests automatically as part of your CI/CD workflow

---

## Installation

```bash
git clone https://github.com/lazarkap/buzzbench.io.git
cd BuzzBenchService
go build -o buzzbench ./cmd/buzzbench
```

Then run it with:

```bash
./buzzbench -h
```

---

## Quick Start — Local Demo

No account needed. Requires Go and Python 3.

```bash
git clone https://github.com/lazarkap/buzzbench.io.git
cd BuzzBenchService
./demo.sh
```

This builds the binary, starts a local FastAPI server, and runs several example tests against it — covering flag mode, config-file mode, POST requests, dynamic variables, slow endpoints, and error rate reporting.

---

## Usage

### Mode 1 — Local flag mode (no API key required)

Test any URL directly from the command line.

```bash
# Basic GET
buzzbench -url http://localhost:8000/health

# Custom concurrency and request count
buzzbench -url http://api.example.com/users -requests 500 -concurrency 50

# POST with a JSON body
buzzbench -url http://api.example.com/orders \
          -method POST \
          -body '{"product_id": 1, "quantity": 2}' \
          -requests 100 -concurrency 10

# With an auth header
buzzbench -url http://api.example.com/protected \
          -auth "Bearer my-token"

# Save results to a file
buzzbench -url http://api.example.com/health -out results.json
```

### Mode 2 — Config file mode (no API key required)

Define one or more tests in a JSON file and run them all in sequence.

```bash
buzzbench -config tests.json
buzzbench -config tests.json -out results.json
```

### Mode 3 — API mode (requires BUZZBENCH_API_KEY)

Fetch test configurations from the BuzzBench.io dashboard and submit results back.

```bash
export BUZZBENCH_API_KEY=your_api_key_here

# Run all pipeline-enabled tests
buzzbench

# Run a specific test by ID
buzzbench -test -id test-123
```

---

## Config File Format

A config file is a JSON array of test objects. Save it as `tests.json` and run with `-config tests.json`.

### Minimal example

```json
[
  {
    "name": "Health check",
    "url": "http://localhost:8000/health",
    "method": "GET",
    "requests": 100,
    "concurrency": 10,
    "timeout_seconds": 5
  }
]
```

### All fields

```json
[
  {
    "id": "optional-identifier",
    "name": "Human-readable test name",
    "description": "What this test is checking",
    "url": "http://api.example.com/endpoint",
    "method": "GET",
    "requests": 200,
    "concurrency": 20,
    "timeout_seconds": 10,
    "auth_token": "Bearer my-token",
    "body": "{\"key\": \"value\"}",
    "variables": []
  }
]
```

| Field | Type | Required | Description |
|---|---|---|---|
| `name` | string | yes | Label shown in the test summary |
| `url` | string | yes | Target URL. Can contain `{{variableName}}` placeholders |
| `method` | string | yes | HTTP method: `GET` `POST` `PUT` `DELETE` `PATCH` |
| `requests` | int | yes | Total number of requests to send |
| `concurrency` | int | yes | Number of concurrent workers |
| `timeout_seconds` | int | yes | Per-request timeout |
| `body` | string | no | Request body. Can contain `{{variableName}}` placeholders |
| `auth_token` | string | no | Passed as the `Authorization` header verbatim |
| `variables` | array | no | Variable definitions (see below) |
| `id` | string | no | Optional identifier |
| `description` | string | no | Optional note, not used at runtime |

---

## Dynamic Variables

Variables let you send different data with each request — different user IDs, random product IDs, unique request identifiers, and more.

Use `{{variableName}}` as a placeholder anywhere in the `url` or `body`. When variables are defined in the config file, `use_variables` is set automatically — you don't need to add it.

### Built-in variables

These work without any variable definition:

| Placeholder | Description | Example value |
|---|---|---|
| `{{$index}}` | Request index, starting at 0 | `0`, `1`, `2` ... |
| `{{$random}}` | Random integer between 0 and 9999 | `4821` |

```json
{
  "name": "Index in URL",
  "url": "http://api.example.com/items/{{$index}}",
  "method": "GET",
  "requests": 50,
  "concurrency": 5,
  "timeout_seconds": 5
}
```

### Variable strategies

Define variables in the `variables` array. Each variable has a `name`, `type`, and `strategy`.

---

#### `static` — same value for every request

```json
{
  "name": "Static token test",
  "url": "http://api.example.com/data?env={{env}}",
  "method": "GET",
  "requests": 100,
  "concurrency": 10,
  "timeout_seconds": 5,
  "variables": [
    {
      "name": "env",
      "type": "string",
      "strategy": "static",
      "value": "staging"
    }
  ]
}
```

| Field | Required | Description |
|---|---|---|
| `value` | yes | The fixed value to use |

---

#### `sequential` — counts up from start to end, then wraps

```json
{
  "name": "User lookup",
  "url": "http://api.example.com/users/{{userId}}",
  "method": "GET",
  "requests": 200,
  "concurrency": 10,
  "timeout_seconds": 5,
  "variables": [
    {
      "name": "userId",
      "type": "integer",
      "strategy": "sequential",
      "startValue": 1,
      "endValue": 100,
      "increment": 1
    }
  ]
}
```

| Field | Required | Description |
|---|---|---|
| `startValue` | yes | Starting value |
| `endValue` | yes | Wraps back to `startValue` after this |
| `increment` | no | Step size (default: 1) |

---

#### `random` — random number in a range, per request

```json
{
  "name": "Create order with random product",
  "url": "http://api.example.com/orders",
  "method": "POST",
  "requests": 100,
  "concurrency": 10,
  "timeout_seconds": 5,
  "body": "{\"product_id\": {{productId}}, \"quantity\": {{qty}}}",
  "variables": [
    {
      "name": "productId",
      "type": "integer",
      "strategy": "random",
      "minValue": 1,
      "maxValue": 9999
    },
    {
      "name": "qty",
      "type": "integer",
      "strategy": "random",
      "minValue": 1,
      "maxValue": 10
    }
  ]
}
```

| Field | Required | Description |
|---|---|---|
| `type` | yes | `integer` or `float` |
| `minValue` | yes | Lower bound (inclusive) |
| `maxValue` | yes | Upper bound (inclusive) |

---

#### `uuid` — a new UUID v4 for every request

Useful for idempotency keys, correlation IDs, or unique resource creation.

```json
{
  "name": "Create resource with unique ID",
  "url": "http://api.example.com/resources",
  "method": "POST",
  "requests": 50,
  "concurrency": 5,
  "timeout_seconds": 5,
  "body": "{\"id\": \"{{requestId}}\", \"name\": \"test\"}",
  "variables": [
    {
      "name": "requestId",
      "type": "string",
      "strategy": "uuid"
    }
  ]
}
```

No extra fields needed — a new UUID is generated for each request automatically.

---

#### `timestamp` — current time in RFC3339 format

```json
{
  "name": "Ingest event with timestamp",
  "url": "http://api.example.com/events",
  "method": "POST",
  "requests": 100,
  "concurrency": 10,
  "timeout_seconds": 5,
  "body": "{\"event\": \"login\", \"time\": \"{{ts}}\"}",
  "variables": [
    {
      "name": "ts",
      "type": "string",
      "strategy": "timestamp"
    }
  ]
}
```

Produces values like `2024-03-15T14:32:01Z` (RFC3339).

---

#### `template` — compose a string using built-in placeholders

Use `{{$index}}` and `{{$random}}` inside the template value to build dynamic strings.

```json
{
  "name": "Create user with generated name",
  "url": "http://api.example.com/users",
  "method": "POST",
  "requests": 50,
  "concurrency": 5,
  "timeout_seconds": 5,
  "body": "{\"username\": \"{{username}}\", \"email\": \"{{email}}\"}",
  "variables": [
    {
      "name": "username",
      "type": "string",
      "strategy": "template",
      "template": "user_{{$index}}"
    },
    {
      "name": "email",
      "type": "string",
      "strategy": "template",
      "template": "user{{$random}}@example.com"
    }
  ]
}
```

| Field | Required | Description |
|---|---|---|
| `template` | yes | String with `{{$index}}` and/or `{{$random}}` inside |

---

### Combining multiple variables

Variables can be mixed freely in the same test. All are resolved independently per request.

```json
{
  "name": "Full example — multiple variables",
  "url": "http://api.example.com/orders/{{orderId}}",
  "method": "PUT",
  "requests": 100,
  "concurrency": 10,
  "timeout_seconds": 5,
  "body": "{\"request_id\": \"{{reqId}}\", \"status\": \"{{status}}\", \"updated_by\": {{agentId}}}",
  "variables": [
    { "name": "orderId",  "type": "integer", "strategy": "sequential", "startValue": 1000, "endValue": 1099 },
    { "name": "reqId",   "type": "string",  "strategy": "uuid" },
    { "name": "status",  "type": "string",  "strategy": "static", "value": "processing" },
    { "name": "agentId", "type": "integer", "strategy": "random", "minValue": 1, "maxValue": 50 }
  ]
}
```

---

## Command Line Reference

```
buzzbench [FLAGS]

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
```

---

## Output Example

```
[1/2] Health check

=== TEST SUMMARY ===
URL: http://api.example.com/health
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

=== ERRORS ===
  [2 occurrences] 503: Service Unavailable
```

---

## License

MIT License — see the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request
