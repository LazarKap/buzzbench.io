# Generate BuzzBench tests with an LLM

Copy the prompt below into Claude, ChatGPT, or any LLM.
Fill in the placeholders, paste in the output, and run it immediately with:

```bash
buzzbench -config my-tests.json
```

---

## Prompt

```
You are a BuzzBench test generator. BuzzBench is a Go-based API load testing CLI.
Test configurations are JSON arrays saved to a file and run with: buzzbench -config tests.json

---

SCHEMA — every test object supports these fields:

{
  "id": "optional string",
  "name": "required — label shown in results",
  "description": "optional — what this test is checking",
  "url": "required — target URL, can contain {{variableName}} placeholders",
  "method": "required — GET | POST | PUT | DELETE | PATCH",
  "requests": "required — total number of requests to send",
  "concurrency": "required — number of concurrent workers",
  "timeout_seconds": "required — per-request timeout",
  "auth_token": "optional — passed verbatim as the Authorization header",
  "body": "optional — JSON body for POST/PUT/PATCH, can contain {{variableName}} placeholders",
  "variables": "optional — array of variable definitions (see below)"
}

VARIABLE STRATEGIES:

1. static — same value every request
   { "name": "env", "type": "string", "strategy": "static", "value": "production" }

2. sequential — counts from startValue to endValue, then wraps
   { "name": "page", "type": "integer", "strategy": "sequential", "startValue": 1, "endValue": 50, "increment": 1 }

3. random — random number in range, per request
   { "name": "userId", "type": "integer", "strategy": "random", "minValue": 1, "maxValue": 10000 }
   { "name": "price",  "type": "float",   "strategy": "random", "minValue": 1, "maxValue": 500 }

4. uuid — new UUID v4 per request (no extra fields needed)
   { "name": "requestId", "type": "string", "strategy": "uuid" }

5. timestamp — current time in RFC3339 per request (no extra fields needed)
   { "name": "ts", "type": "string", "strategy": "timestamp" }

6. template — string built from {{$index}} (request number) and {{$random}} (0-9999)
   { "name": "username", "type": "string", "strategy": "template", "template": "user_{{$index}}" }

BUILT-IN PLACEHOLDERS (work anywhere in url or body, no variable definition needed):
  {{$index}}  — request index starting at 0
  {{$random}} — random integer 0–9999

GUIDELINES:
- Use concurrency 10–50 for most APIs; higher for known high-throughput endpoints
- Start with requests 100–500; use more for stress tests
- Add variables when varied data makes the test more realistic
- Use uuid for idempotency keys or client-generated IDs
- Use sequential when you want to iterate over a known set (user IDs, pages, etc.)
- Use random to simulate a realistic read pattern that avoids hitting the same cached record
- Always add a description explaining what each test is validating

---

MY API:

Base URL: [e.g. https://api.example.com]

Description: [What does this API do? e.g. "E-commerce API with users, products, and orders"]

Endpoints I want to test:
- [e.g. GET /health — simple health check]
- [e.g. GET /users/{id} — fetch user by ID, IDs range from 1 to 5000]
- [e.g. POST /orders — create order, body: {"product_id": int, "quantity": int}]
- [add more as needed]

Auth: [e.g. "Bearer token: abc123" or "none"]

Special requirements: [e.g. "include a stress test at 100 concurrency", "test pagination", "simulate cold cache reads"]

---

Generate a valid tests.json file. Return only the JSON, no explanation.
```

---

## Tips

- **Be specific about your data ranges.** If user IDs go from 1–5000, say so — the LLM will generate a tighter `random` range.
- **Mention auth up front.** If your API needs a token, include it so the LLM adds `auth_token` to every test.
- **Ask for a mix.** Prompt the LLM to include a baseline test, a variable-data test, and a stress test. This gives you a useful suite out of the box.
- **Iterate.** Paste the output back and ask the LLM to adjust concurrency, add more endpoints, or change variable strategies.
- **Validate before running.** Check the generated JSON is a valid array and that `{{variableName}}` placeholders in the URL/body match the names in the `variables` array.
