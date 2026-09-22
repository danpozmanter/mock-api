# mock-api

Mock an API from an OpenAPI spec, simulating low and high latency along with a
mocked error response. Written in [Gossamer][gossamer].

[![Tests](https://github.com/danpozmanter/mock-api/actions/workflows/test.yml/badge.svg)](https://github.com/danpozmanter/mock-api/actions)

[gossamer]: https://github.com/gossamer-lang/gossamer

## Features

- Mock every path of an OpenAPI spec (YAML or JSON), loaded from a file or an
  `http(s)://` URL, such as the OpenAI spec.
- Uniform-random latency within a configured range for each response.
- Simulated error responses at a target frequency, with feedback that steers
  the observed rate toward the target.
- Per-path JSON response overrides, written as YAML structure or as a JSON
  string.
- Server-Sent Events streaming via `?stream=true`.

## Prebuilt binaries

Each tagged release publishes archives that need no Gossamer toolchain, for
Linux (x86_64, aarch64), macOS (Apple silicon, Intel), and Windows (x86_64),
plus a `SHA256SUMS` file. Download the one for your platform from the
[releases page](https://github.com/danpozmanter/mock-api/releases), extract it,
and run the binary from the extracted directory, where `config.yaml` and
`spec.yaml` sit beside it:

```
tar -xzf mock-api-0.1.0-linux-x86_64.tar.gz
cd mock-api-0.1.0-linux-x86_64
./mock-api --port 9000
```

On macOS, a zip downloaded through a browser is quarantined; clear it with
`xattr -d com.apple.quarantine mock-api` before the first run. Releases are
built by `.github/workflows/release.yml` when a `v*` tag is pushed.

## Project layout

```
project.toml        # project manifest
config.yaml         # server config (--config, defaults to ./config.yaml)
spec.yaml           # example OpenAPI spec named by config.yaml's api_spec
src/
  main.gos          # entry point: flags, startup logging, server
  config.gos        # config parsing and validation
  apispec.gos       # OpenAPI spec loading (file or URL)
  handler.gos       # routing, latency, overrides, errors, SSE streaming
  simulator.gos     # adaptive error simulator
  random.gos        # lock-free SplitMix64 generator shared across requests
scripts/
  smoke.sh          # boots a built binary and checks each kind of answer
  smoke.yaml        # deterministic config used by smoke.sh
```

## Configuration

`config.yaml`:

```yaml
# URL or file path to your API spec (OpenAPI YAML or JSON).
api_spec: "https://raw.githubusercontent.com/openai/openai-openapi/master/openapi.yaml"

latency:
  low: 50           # Low latency in ms.
  high: 5000        # High latency in ms.

prefix: "v1"

responses:
  # Direct JSON string override
  "/v1/chat/completions": |
    {
      "id": "chatcmpl-123",
      "object": "chat.completion",
      "choices": [
        { "index": 0, "message": { "role": "assistant", "content": "Hello there!" } }
      ]
    }

  # Structure override (will be converted to JSON)
  "/v1/models":
    object: "list"
    data:
      - id: "gpt-3.5-turbo"
        object: "model"
        created: 1677610602

error_response:
  code: 500
  body:
    error: "simulated error occurred"
  frequency: 0.1
```

| Field | Meaning |
|------|---------|
| `api_spec` | File path or `http(s)://` URL of an OpenAPI document (YAML or JSON). |
| `latency.low` / `latency.high` | Inclusive range, in ms, for the uniform-random latency of each response. |
| `prefix` | Routing prefix prepended to every spec path. |
| `responses` | Optional map from a full route path (prefix included, as written in the spec, e.g. `/v1/models/{model}`) to its response body. A string value is parsed as JSON; any other value is used as is. |
| `error_response.code` | HTTP status returned when the error simulator fires. |
| `error_response.body` | Body returned with that status. |
| `error_response.frequency` | Target proportion of requests that fail, from `0` (never) to `1` (always). |

Every field except `responses` is required. Startup fails with one message
naming every missing or invalid value: a non-numeric latency or frequency,
`low` above `high`, a frequency outside `[0, 1]`, a code outside `100-599`, or
an override string that is not valid JSON. A `0` is a valid latency and a
valid frequency.

### Spec

Only `paths` and each path's operation keys (`get`, `put`, `post`, `delete`,
`options`, `head`, `patch`, `trace`) are read; other path-item keys such as
`parameters` or `summary` are ignored. Path templates like `/models/{model}`
match any single segment.

```yaml
paths:
  /chat/completions:
    post: {}
  /models:
    get: {}
  /models/{model}:
    get: {}
```

## Running

```
gos run src/main.gos                                  # uses ./config.yaml on port 8080
gos run src/main.gos --config config.yaml --port 9000
gos build --release && target/release/mock-api --port 9000
```

The `makefile` wraps the common commands: `make build`, `make run`,
`make test`, `make check`, `make fmt`, `make lint`, `make smoke`.

Startup output:

```
Loaded config: api_spec=spec.yaml prefix=v1 latency=50-5000 ms error_response=500 at frequency 0.1
Loaded API spec with 3 paths
Registered endpoint: POST /v1/chat/completions
Registered endpoint: GET /v1/models
Registered endpoint: GET /v1/models/{model}
Loaded responses: /v1/chat/completions, /v1/models
Starting server on 0.0.0.0:8080
```

Per-request log lines:

```
Path /v1/models: Sleeping for 3471.27 ms
Simulating error for request          # when the simulator fires
405 DELETE /v1/models                 # known path, undeclared method
404 GET /v1/unknown                   # unknown path
```

## Endpoints

For each path and method in the spec, the server registers
`/<prefix><path>` and, after sleeping a random latency, answers with:

- The **override** from `responses` for that route (status `200`).
- Otherwise the **default** body `{"message": "Response for <route path>"}`.
- The **error response** instead, at the configured frequency. Each path keeps
  its own error counters, shared by all of its methods.

Requests with a trailing slash match the same route. A known path called with
an undeclared method answers `405 {"error": "Method not allowed"}` and an
unknown path answers `404 {"error": "Not found"}`, both without latency or
simulated errors.

Append `?stream=true` to receive the body as Server-Sent Events: the compact
JSON is split into about three `data:` frames, each followed by a random
latency, and the stream ends with `data: [DONE]`.

## Tests

```
gos test                  # unit and end-to-end tests for every module
make smoke                # release build plus a live check of the native binary
```

The tests cover config parsing and validation, spec loading from files and
URLs, the error simulator's convergence and its behavior under concurrent
callers, routing (templates, trailing slashes, 404, 405), overrides, and a
server started on a free port that answers JSON and streams SSE frames.
