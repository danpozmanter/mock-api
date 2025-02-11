# mock-api

Mock an API using an OpenAPI spec, simulating low and high latency, as well as a mocked error response.

[![Tests Passing](https://github.com/danpozmanter/mock-api/actions/workflows/test.yml/badge.svg)](https://github.com/danpozmanter/mock-api/actions)

## Features

* Mock a full api spec, such as OpenAI.
* Set a range for the latency of the response.
* Set a frequency for error responses.
* Custom JSON response overrides.

## Example Configuration & Explanation

```yaml
# URL or file path to your API spec YAML.
api_spec: "https://raw.githubusercontent.com/openai/openai-openapi/master/openapi.yaml"

latency:
  low: 50           # Low latency in ms.
  high: 5000        # High latency in ms.

prefix: "v1"

responses:
  # Direct JSON string override
  "/v1/chat/completions":  |
    {
      "id": "chatcmpl-123",
      "object": "chat.completion",
      "created": 1677652288,
      "model": "gpt-3.5-turbo",
      "choices": [
        {
          "index": 0,
          "message": {
            "role": "assistant",
            "content": "Hello there!"
          }
        }
      ]
    }
  
  # Structure override (will be converted to JSON)
  "/v1/models": |
    {
      "object": "list",
      "data": [
        {
          "id": "gpt-3.5-turbo",
          "object": "model",
          "created": 1677610602
        },
        {
          "id": "gpt-4",
          "object": "model",
          "created": 1677610602
        }
      ]
    }

error_response:
  code: 500
  body:
    error: "simulated error occurred"
  frequency: 0.1

```

### Latency

Configures how quickly to respond with the mock data, in miliseconds. Specify the range for the latency of the response (randomly selected).

### Responses

Per matching request path, override any default response given in the api spec.

### Error Response

What response to return on error, along with the frequency.