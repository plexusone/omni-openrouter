# omni-openrouter

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
- Automatic error classification for intelligent fallback

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
