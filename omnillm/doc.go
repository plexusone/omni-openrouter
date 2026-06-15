// Package openrouter provides a thick omnillm provider for OpenRouter.
//
// This package implements the omnillm-core Provider interface using the official
// OpenRouter Go SDK. It registers itself as a thick provider on import, allowing
// applications to use OpenRouter models through the unified omnillm interface.
//
// # Usage
//
// Import this package for side effects to register the OpenRouter provider:
//
//	import _ "github.com/plexusone/omni-openrouter/omnillm"
//
// Then create a chat client with the OpenRouter provider:
//
//	client, err := omnillm.NewChatClient(omnillm.Config{
//	    Provider: "openrouter",
//	    APIKey:   os.Getenv("OPENROUTER_API_KEY"),
//	    Model:    "anthropic/claude-3.5-sonnet",
//	})
//
// # Supported Models
//
// OpenRouter provides access to 300+ models from various providers including:
//   - Anthropic (Claude models)
//   - OpenAI (GPT models)
//   - Google (Gemini models)
//   - Meta (Llama models)
//   - And many more
//
// Use model identifiers in the format "provider/model-name", for example:
//   - "anthropic/claude-3.5-sonnet"
//   - "openai/gpt-4-turbo"
//   - "google/gemini-pro"
//
// # Authentication
//
// The provider supports API key authentication. You can obtain an API key from
// https://openrouter.ai/keys or use the OAuth PKCE flow via the auth package.
//
// # OAuth PKCE Flow
//
// For interactive applications, you can use the auth package to authenticate:
//
//	import "github.com/plexusone/omni-openrouter/auth"
//
//	apiKey, err := auth.Login(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
package openrouter
