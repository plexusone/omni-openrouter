# Omni-OpenRouter

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go SAST][go-sast-svg]][go-sast-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![Docs][docs-mkdoc-svg]][docs-mkdoc-url]
[![Visualization][viz-svg]][viz-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/omni-openrouter/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/omni-openrouter/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/omni-openrouter/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/omni-openrouter/actions/workflows/go-lint.yaml
 [go-sast-svg]: https://github.com/plexusone/omni-openrouter/actions/workflows/go-sast-codeql.yaml/badge.svg?branch=main
 [go-sast-url]: https://github.com/plexusone/omni-openrouter/actions/workflows/go-sast-codeql.yaml
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omni-openrouter
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omni-openrouter
 [docs-mkdoc-svg]: https://img.shields.io/badge/Go-dev%20guide-blue.svg
 [docs-mkdoc-url]: https://plexusone.dev/omni-openrouter
 [viz-svg]: https://img.shields.io/badge/Go-visualizaton-blue.svg
 [viz-url]: https://mango-dune-07a8b7110.1.azurestaticapps.net/?repo=plexusone%2Fomni-openrouter
 [loc-svg]: https://tokei.rs/b1/github/plexusone/omni-openrouter
 [repo-url]: https://github.com/plexusone/omni-openrouter
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omni-openrouter/blob/main/LICENSE

OpenRouter thick provider for omnillm, providing access to 300+ AI models through the unified omnillm interface.

## Installation

```bash
go get github.com/plexusone/omni-openrouter
```

## Usage

### As an omnillm Provider

Import the package for side effects to register the OpenRouter provider:

```go
package main

import (
    "context"
    "os"

    _ "github.com/plexusone/omni-openrouter/omnillm"
    "github.com/plexusone/omnillm"
)

func main() {
    client, err := omnillm.NewChatClient(omnillm.Config{
        Provider: "openrouter",
        APIKey:   os.Getenv("OPENROUTER_API_KEY"),
        Model:    "anthropic/claude-3.5-sonnet",
    })
    if err != nil {
        panic(err)
    }
    defer client.Close()

    // Use the client as usual
    resp, err := client.CreateChatCompletion(context.Background(), &omnillm.ChatCompletionRequest{
        Model: "anthropic/claude-3.5-sonnet",
        Messages: []omnillm.Message{
            {Role: omnillm.RoleUser, Content: "Hello!"},
        },
    })
    // ...
}
```

### OAuth PKCE Authentication

For interactive applications, you can use OAuth to authenticate:

```go
package main

import (
    "context"
    "fmt"

    "github.com/plexusone/omni-openrouter/auth"
)

func main() {
    ctx := context.Background()

    // Perform OAuth login (opens browser)
    apiKey, err := auth.Login(ctx)
    if err != nil {
        panic(err)
    }

    fmt.Printf("API key obtained: %s...\n", apiKey[:10])

    // Later, retrieve the stored key
    storedKey, err := auth.LoadAPIKey(ctx)
    if err != nil {
        panic(err)
    }
    // ...
}
```

## Supported Models

OpenRouter provides access to models from:

- **Anthropic**: Claude 4 Opus, Claude 4 Sonnet, Claude 3.5 Sonnet, etc.
- **OpenAI**: GPT-4 Turbo, GPT-4, GPT-3.5 Turbo, etc.
- **Google**: Gemini Pro, Gemini Ultra, etc.
- **Meta**: Llama 3, Llama 2, etc.
- **Mistral**: Mixtral, Mistral Large, etc.
- And 300+ more models

Model identifiers follow the format `provider/model-name`, for example:

- `anthropic/claude-opus-4-20250514`
- `openai/gpt-4-turbo`
- `google/gemini-pro`
- `meta-llama/llama-3-70b-instruct`

## Features

- Full tool/function calling support
- Streaming completions
- Vision support (for compatible models)
- JSON mode (for compatible models)
- Reasoning effort control
- Automatic error classification for intelligent fallback

## Reasoning

OpenRouter supports the `reasoning_effort` parameter with extended values:

```go
import omnillm "github.com/plexusone/omnillm-core"

effort := omnillm.ReasoningEffortHigh
response, err := client.CreateChatCompletion(ctx, &omnillm.ChatCompletionRequest{
    Model:           "anthropic/claude-3.5-sonnet",
    ReasoningEffort: &effort,
    Messages:        messages,
})
```

### Supported Values

OpenRouter supports extended `reasoning_effort` values beyond the standard four:

| Value | Description |
|-------|-------------|
| `"none"` | Disable reasoning |
| `"minimal"` | Minimal reasoning |
| `"low"` | Light reasoning |
| `"medium"` | Moderate reasoning |
| `"high"` | Deep reasoning |
| `"xhigh"` | Extra-high reasoning |
| `"max"` | Maximum reasoning |

The standard omnillm constants (`ReasoningEffortNone`, `ReasoningEffortLow`, `ReasoningEffortMedium`, `ReasoningEffortHigh`) work directly. For extended values like `"xhigh"` or `"max"`, set the string value directly.

See [Reasoning Feature Guide](https://github.com/plexusone/omnillm-core/blob/main/docs/features/reasoning.md) for cross-provider usage.

## Configuration

The provider accepts the following configuration:

| Field | Description |
|-------|-------------|
| `APIKey` | OpenRouter API key (required) |
| `BaseURL` | Custom API endpoint (optional) |
| `SiteURL` | Your application URL for attribution (optional) |
| `SiteName` | Your application name for attribution (optional) |

## License

MIT
