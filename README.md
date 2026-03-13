# vision-mcp

An MCP (Model Context Protocol) server providing image vision analysis via OpenAI-compatible APIs.

## Features

- **Multiple image sources**: Accepts local file paths, HTTP(S) URLs, or data URLs
- **Region cropping**: Crop to specific regions using fractional coordinates before analysis
- **OpenAI-compatible**: Works with any OpenAI-compatible API endpoint (local models like Ollama, or cloud providers)
- **Configurable detail levels**: Control analysis fidelity with `low`, `high`, or `auto` detail settings

## Installation

```bash
go install github.com/SecKatie/vision-mcp@latest
```

## Configuration

Required environment variables:

- `VISION_API_BASE_URL` — Base URL of your OpenAI-compatible API (e.g., `http://localhost:11434/v1`)
- `VISION_API_KEY` — API key or bearer token

Optional environment variables:

- `VISION_API_MODEL` — Model name to use (default: `gpt-4.1-mini`)
- `VISION_API_MAX_TOKENS` — Default max response tokens (default: `1024`)

## Usage

The server exposes a single `see` tool with the following schema:

### Input

| Field | Type | Description |
|-------|------|-------------|
| `source` | string | **Required.** Image source: local file path, HTTP(S) URL, or data URL |
| `question` | string | What to ask about the image (defaults to "Describe this image in detail.") |
| `detail` | string | API image detail level: `low`, `high`, or `auto` (default: `auto`) |
| `max_tokens` | int | Maximum tokens in the response for this request |
| `crop` | object | Crop region as fractional coordinates (0.0–1.0, see below) |

### Crop Region

When cropping, all coordinates are fractional (0.0–1.0) relative to image dimensions:

| Field | Type | Description |
|-------|------|-------------|
| `x` | number | **Required.** Left edge fraction 0.0-1.0 |
| `y` | number | **Required.** Top edge fraction 0.0-1.0 |
| `width` | number | **Required.** Width fraction 0.0-1.0 |
| `height` | number | **Required.** Height fraction 0.0-1.0 |

### Output

```json
{
  "text": "...",
  "model": "...",
  "prompt_tokens": 1234,
  "completion_tokens": 567
}
```

## Example MCP Client Configuration

Add to your Claude Desktop MCP configuration:

```json
{
  "mcpServers": {
    "vision": {
      "command": "vision-mcp",
      "env": {
        "VISION_API_BASE_URL": "http://localhost:11434/v1",
        "VISION_API_KEY": "your-api-key",
        "VISION_API_MODEL": "llava"
      }
    }
  }
}
```

## Development

Build:

```bash
go build ./...
```

Test:

```bash
go test ./...
```

Lint:

```bash
golangci-lint run
```

Format:

```bash
gofumpt -w .
goimports -w .
```

## License

AGPL-3.0 — See [LICENSE](LICENSE) for details.
