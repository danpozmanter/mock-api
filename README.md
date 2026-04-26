# mock-api (Gossamer port)

Mock an API using a JSON spec, simulating low and high latency along
with a mocked error response. This is a [Gossamer][gossamer] port of
the original Go implementation, which still lives untouched under
[`go_implementation/`](./go_implementation/).

[gossamer]: https://github.com/gossamer-lang/gossamer

## Features

- Mock a full API spec from a JSON file (or `http(s)://` URL).
- Set a uniform-random latency range for each response.
- Set a frequency for simulated error responses (with adjusting
  feedback so the observed rate converges to the target).
- Per-path JSON response overrides.
- Server-Sent-Events streaming via `?stream=true`.

## Project layout

```
project.toml             # Gossamer manifest
config.json              # mock-api config (loaded by --config, defaults to ./config.json)
spec.json                # API spec (referenced from config.json's "api_spec")
src/
├── main.gos             # entry point — flag parsing, App + http::Handler impl
├── random/mod.gos       # LCG PRNG + tests
├── simulator/mod.gos    # ErrorSimulator + tests (embeds its own PRNG)
├── config/mod.gos       # config parse / required-field validation + tests
├── apispec/mod.gos      # API spec loader + tests
└── handler/mod.gos      # routing / latency / overrides / SSE shaping + tests
go_implementation/        # the original Go port, kept for reference
```

Each subdirectory module is independently testable via
`gos test src/<name>/mod.gos`. `gos test src/` walks every `.gos`
under `src/` and runs all tests in one shot.

## Configuration

`config.json`:

```json
{
  "api_spec": "spec.json",
  "latency": { "low_ms": 50, "high_ms": 5000 },
  "prefix":  "v1",
  "responses": [
    {
      "path": "/v1/chat/completions",
      "body": {
        "id": "chatcmpl-123",
        "object": "chat.completion",
        "choices": [
          { "index": 0,
            "message": { "role": "assistant", "content": "Hello there!" } }
        ]
      }
    }
  ],
  "error_response": {
    "code": 500,
    "body": { "error": "simulated error occurred" },
    "frequency_per_10k": 1000
  }
}
```

`spec.json`:

```json
{
  "paths": [
    { "path": "/chat/completions", "methods": ["POST"] },
    { "path": "/models",           "methods": ["GET"] }
  ]
}
```

### Field reference

| Field | Type | Meaning |
|------|------|---------|
| `api_spec` | string | File path or HTTP(S) URL of the spec JSON. |
| `latency.low_ms` / `latency.high_ms` | int | Inclusive uniform-random latency range, in ms. |
| `prefix` | string | Routing prefix prepended to every spec path (no leading / trailing slashes). |
| `responses` | array | Per-path overrides: `{ "path": "<full-prefixed-path>", "body": <any JSON> }`. The body is rendered verbatim. |
| `error_response.code` | int | HTTP status returned when the error simulator fires. |
| `error_response.body` | any | JSON body returned with the error. |
| `error_response.frequency_per_10k` | int | Target error rate, expressed as numerator out of 10 000 (so `1000` → 10 %). |

### Schema differences vs the Go port

- **JSON, not YAML.** Gossamer's standard library has `std::encoding::json`
  but no YAML parser, so both files moved to JSON.
- **`responses` is an array, not a map.** Gossamer's `json::Value` does not
  yet expose key iteration or dynamic field lookup over `Object`s; an array
  of `{path, body}` records is the idiomatic shape.
- **`paths` is an array.** Same reason.
- **`frequency_per_10k: i64` instead of `frequency: f64`.** The interpreter's
  `as i64` cast is currently a bit-reinterpret (it doesn't perform the value
  conversion), so storing the rate as an integer numerator avoids any
  cross-kind float/int arithmetic at runtime.
- **`low_ms` / `high_ms` instead of `low` / `high`.** Names made explicit
  while we're touching the schema.

## Running

```
gos check src/main.gos       # parse + resolve + typecheck the binary entry
gos test  src/                # all 54 unit tests, every module
gos lint  src/main.gos        # style lints (warnings only)
gos run   src/main.gos        # boot the server (see runtime gotchas below)
```

Each module's tests can also be run on their own:

```
gos test src/random/mod.gos      #  6 PRNG tests
gos test src/simulator/mod.gos   # 14 simulator tests
gos test src/config/mod.gos      #  5 config tests
gos test src/apispec/mod.gos     #  7 apispec tests
gos test src/handler/mod.gos     # 21 handler tests
gos test src/main.gos            #  1 build_app_config test
```

The server listens on `0.0.0.0:8080` by default. Override with flags
(passed *after* `--`, since `gos run` interprets unknown flags itself):

```
gos run src/main.gos -- --config config.json --port 9000
```

## Endpoints

For each `(path, method)` in `spec.json`, the server registers a route
at `"/<prefix><path>"` and replies with:

- **Override response** (status `200`) when `responses[]` has a matching
  `path`, after sleeping a uniform-random latency.
- **Default response** `{"message": "Response for <full-path>"}` if no
  override matches.
- **Error response** (`error_response.code`) chosen at the configured
  frequency, regardless of which route matched.
- **`405`** for known paths called with an undeclared method.
- **`404`** for unknown paths.

Append `?stream=true` to wrap the body as a Server-Sent-Events frame
ending with `data: [DONE]`.

## Tests

```
gos test src/
```

54 unit tests across six modules:

- **`random_tests`** (6) — PRNG range, replay, edge cases.
- **`simulator_tests`** (14) — error simulator init, observed rate,
  convergence toward the target, adaptive adjustment when above / below
  target.
- **`config_tests`** (5) — JSON parse, required-field validation,
  partial-fill detection, helper coverage.
- **`apispec_tests`** (7) — spec parse, missing-field rejection, route
  count, byte-to-string round trip.
- **`handler_tests`** (21) — path joining, query-string-aware match,
  override lookup (with and without `?query=`), default response,
  latency range, SSE body shape, error body rendering, route matching
  (Found / 404 / 405).
- **`main_tests`** (1) — `build_app_config` snapshots scalars off the
  parsed JSON.

## Architecture

```
                    ┌──────────────┐
                    │  config.json │
                    └───────┬──────┘
                            │ config::load_config / parse_config
                            ▼
┌────────────┐      ┌────────────────┐      ┌──────────────┐
│ spec.json  │ ───▶ │ build_app_cfg  │ ───▶ │   AppConfig  │
└──────┬─────┘      └────────────────┘      └──────┬───────┘
       │ apispec::load_api_spec / parse_spec     │
       ▼                                           ▼
                          ┌──────────────────────────────────┐
                          │  App  impl http::Handler         │
                          │  serve(req)                      │
                          │   ├─ handler::match_route        │
                          │   ├─ handler::get_latency_ms     │
                          │   ├─ handler::response_body_for  │
                          │   └─ json_response | sse_response│
                          └──────────────────────────────────┘
```

Each module is self-contained (no cross-module method calls at runtime
— see *Gossamer-runtime gotchas* below). Mutating helpers (`Random::next_i64`,
`Random::below`, `Random::chance`, `ErrorSimulator::should_error`) take
`self` by value and return `(NewSelf, Result)` so the caller can rebind.

## Gossamer-runtime gotchas

This port works around several pre-1.0.0 limitations of the current
Gossamer interpreter (release 0.x at the time of writing). They are
called out in source comments next to the workaround, but in summary:

1. **Cross-module function and method calls don't dispatch at runtime.**
   `gos check` resolves cross-module references just fine, but at
   runtime calls into another module return `()` (the call body is
   never entered). The project is laid out as one module per
   subdirectory anyway — every module is independently testable via
   `gos test src/<name>/mod.gos` — and each module that needs a
   helper from elsewhere keeps a private copy. `src/main.gos` wires
   the modules together for the binary; the wiring typechecks but
   the running server has the gaps until cross-module dispatch
   lands.
2. **Method names are dispatched name-globally.** Two `impl` blocks
   each defining `fn new(...)` with different arities collide and one
   poisons the dispatch table; we use distinct names (`Random::seeded`,
   `ErrorSimulator::for_rate`).
3. **`&mut self` mutations don't persist back to the caller.** The
   runtime treats `self` as a local copy. We work around this by
   making mutating methods take `self` by value and return a new
   instance; the caller rebinds the name.
4. **`[T]::push` silently no-ops on typed arrays.** Static array
   literals work fine, so we build vectors as fixed-size literals and
   iterate parsed `json::Value` arrays directly via `.iter()` /
   `.len()`.
5. **`as f64` and `as i64` are bit-reinterpret casts at runtime, not
   value conversions.** Cross-kind arithmetic is therefore unsafe; the
   simulator and PRNG stay in pure `i64`. The error-rate threshold is
   stored as an integer numerator out of 10 000 instead of a float.
6. **`std::math::rand::Rng` methods are not yet bridged.** We
   implement a minimal LCG (Knuth's MMIX constants) directly in
   Gossamer.
7. **`std::sync::Mutex` and `std::sync::AtomicU64` methods are not yet
   bridged.** And `http::Handler::serve` is `&self`, so we cannot
   thread mutating state across requests via shared atomics. The
   request handler seeds a fresh PRNG per call from the configured
   seed plus a `time::now()` tick. Statistically equivalent to the Go
   simulator's "shared atomic counter" approach for our use case.
8. **`std::strings::split` / `String::trim_*` / `String::replace` /
   `strings::find` are regex-backed and the regex bridge is not yet
   wired.** The handler avoids stripping the query string off the
   request path; instead `path_matches` checks for both exact equality
   and the `route_path + "?"` prefix.
9. **`errors::Error::message()` returns `()`** in the current interp.
   Fallible APIs in this port use `Result<T, String>` instead of
   `Result<T, errors::Error>` so the message text is recoverable in
   tests.
10. **`fs::read_to_string` and `os::read_file` do not currently return
    a usable string** (the upstream `examples/file_io.gos` shows the
    same `read 0 bytes` behaviour). `gos test` still exercises the
    full validation / matching pipeline against inline JSON literals;
    the `--config <path>` flag will start to work as soon as the
    filesystem bridge does.

Each of these has a marked workaround in `src/main.gos`. As the
interpreter matures, the affected helpers can collapse back to the
idiomatic `&mut self` / `Result<_, errors::Error>` / `[T]::push`
shapes.

## Original Go implementation

The original Go port lives untouched under
[`go_implementation/`](./go_implementation/). All Go tests still pass
under `go test ./...` from that directory.
