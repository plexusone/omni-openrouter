package openrouter

import (
	"context"
	"io"

	openrouter "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/OpenRouterTeam/go-sdk/models/operations"
	"github.com/OpenRouterTeam/go-sdk/optionalnullable"
	"github.com/OpenRouterTeam/go-sdk/types/stream"
	core "github.com/plexusone/omnillm-core"
)

// ProviderNameOpenRouter is the provider name for OpenRouter.
const ProviderNameOpenRouter core.ProviderName = "openrouter"

func init() {
	// Register OpenRouter as a thick provider (priority 10, overrides thin providers)
	core.RegisterProvider(ProviderNameOpenRouter, newProviderFromConfig, core.PriorityThick)
}

// newProviderFromConfig creates a new OpenRouter provider from omnillm config (for registry).
func newProviderFromConfig(config core.ProviderConfig) (core.Provider, error) {
	return New(Config{
		APIKey:  config.APIKey,
		BaseURL: config.BaseURL,
	})
}

// Config holds configuration for the OpenRouter provider.
type Config struct {
	// APIKey is the OpenRouter API key (required).
	APIKey string

	// BaseURL is an optional custom API endpoint.
	BaseURL string

	// SiteURL is the URL of your application (for OpenRouter attribution).
	SiteURL string

	// SiteName is the name of your application (for OpenRouter attribution).
	SiteName string
}

// Provider implements core.Provider using the official OpenRouter SDK.
type Provider struct {
	client   *openrouter.OpenRouter
	config   Config
	siteURL  string
	siteName string
}

// Ensure Provider implements core.Provider at compile time.
var _ core.Provider = (*Provider)(nil)

// New creates a new OpenRouter provider with the given configuration.
func New(cfg Config) (*Provider, error) {
	if cfg.APIKey == "" {
		return nil, core.ErrInvalidAPIKey
	}

	opts := []openrouter.SDKOption{
		openrouter.WithSecurity(cfg.APIKey),
	}

	if cfg.BaseURL != "" {
		opts = append(opts, openrouter.WithServerURL(cfg.BaseURL))
	}

	if cfg.SiteName != "" {
		opts = append(opts, openrouter.WithXTitle(cfg.SiteName))
	}

	if cfg.SiteURL != "" {
		opts = append(opts, openrouter.WithHTTPReferer(cfg.SiteURL))
	}

	client := openrouter.New(opts...)

	return &Provider{
		client:   client,
		config:   cfg,
		siteURL:  cfg.SiteURL,
		siteName: cfg.SiteName,
	}, nil
}

// Name returns the provider identifier.
func (p *Provider) Name() string {
	return "openrouter"
}

// Capabilities returns the provider's supported features.
func (p *Provider) Capabilities() core.Capabilities {
	return core.Capabilities{
		Tools:             true,
		Streaming:         true,
		Vision:            true,
		JSON:              true, // OpenRouter supports JSON mode for compatible models
		SystemRole:        true,
		MaxContextWindow:  200000, // Varies by model, this is max available
		SupportsMaxTokens: true,
	}
}

// Close releases resources held by the provider.
func (p *Provider) Close() error {
	// The OpenRouter SDK doesn't require explicit cleanup
	return nil
}

// CreateChatCompletion sends a chat completion request and returns the response.
func (p *Provider) CreateChatCompletion(ctx context.Context, req *core.ChatCompletionRequest) (*core.ChatCompletionResponse, error) {
	chatReq := p.buildRequest(req)

	resp, err := p.client.Chat.Send(ctx, chatReq)
	if err != nil {
		return nil, p.wrapError(err)
	}

	if resp.ChatResult == nil {
		return nil, core.NewAPIError("openrouter", 0, "", "empty response from OpenRouter")
	}

	return p.convertResponse(resp.ChatResult), nil
}

// CreateChatCompletionStream creates a streaming chat completion.
func (p *Provider) CreateChatCompletionStream(ctx context.Context, req *core.ChatCompletionRequest) (core.ChatCompletionStream, error) {
	chatReq := p.buildRequest(req)
	chatReq.Stream = openrouter.Pointer(true)

	resp, err := p.client.Chat.Send(ctx, chatReq)
	if err != nil {
		return nil, p.wrapError(err)
	}

	if resp.EventStream == nil {
		return nil, core.NewAPIError("openrouter", 0, "", "streaming not enabled in response")
	}

	return &streamAdapter{stream: resp.EventStream}, nil
}

// buildRequest converts a core request to OpenRouter SDK ChatRequest.
func (p *Provider) buildRequest(req *core.ChatCompletionRequest) components.ChatRequest {
	chatReq := components.ChatRequest{
		Model:    openrouter.Pointer(req.Model),
		Messages: p.convertMessages(req.Messages),
	}

	if req.MaxTokens != nil {
		maxTokens := int64(*req.MaxTokens)
		chatReq.MaxTokens = optionalnullable.From(&maxTokens)
	}

	if req.Temperature != nil {
		chatReq.Temperature = optionalnullable.From(req.Temperature)
	}

	if req.TopP != nil {
		chatReq.TopP = optionalnullable.From(req.TopP)
	}

	if len(req.Stop) > 0 {
		stop := components.CreateStopArrayOfStr(req.Stop)
		chatReq.Stop = optionalnullable.From(&stop)
	}

	if len(req.Tools) > 0 {
		chatReq.Tools = p.convertTools(req.Tools)
	}

	if req.ToolChoice != nil {
		chatReq.ToolChoice = p.convertToolChoice(req.ToolChoice)
	}

	return chatReq
}

// convertMessages converts core messages to OpenRouter message format.
func (p *Provider) convertMessages(messages []core.Message) []components.ChatMessages {
	result := make([]components.ChatMessages, 0, len(messages))

	for _, msg := range messages {
		switch msg.Role {
		case core.RoleSystem:
			result = append(result, components.CreateChatMessagesSystem(
				components.ChatSystemMessage{
					Content: components.CreateChatSystemMessageContentStr(msg.Content),
					Role:    components.ChatSystemMessageRoleSystem,
				},
			))

		case core.RoleUser:
			result = append(result, components.CreateChatMessagesUser(
				components.ChatUserMessage{
					Content: components.CreateChatUserMessageContentStr(msg.Content),
					Role:    components.ChatUserMessageRoleUser,
				},
			))

		case core.RoleAssistant:
			assistantMsg := components.ChatAssistantMessage{
				Role: components.ChatAssistantMessageRoleAssistant,
			}

			// Set content using OptionalNullable
			content := components.CreateChatAssistantMessageContentStr(msg.Content)
			assistantMsg.Content = optionalnullable.From(&content)

			// Handle tool calls
			if len(msg.ToolCalls) > 0 {
				toolCalls := make([]components.ChatToolCall, len(msg.ToolCalls))
				for i, tc := range msg.ToolCalls {
					toolCalls[i] = components.ChatToolCall{
						ID:   tc.ID,
						Type: components.ChatToolCallTypeFunction,
						Function: components.ChatToolCallFunction{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				}
				assistantMsg.ToolCalls = toolCalls
			}

			result = append(result, components.CreateChatMessagesAssistant(assistantMsg))

		case core.RoleTool:
			toolCallID := ""
			if msg.ToolCallID != nil {
				toolCallID = *msg.ToolCallID
			}
			result = append(result, components.CreateChatMessagesTool(
				components.ChatToolMessage{
					Content:    components.CreateChatToolMessageContentStr(msg.Content),
					Role:       components.ChatToolMessageRoleTool,
					ToolCallID: toolCallID,
				},
			))
		}
	}

	return result
}

// convertTools converts core tools to OpenRouter tool format.
func (p *Provider) convertTools(tools []core.Tool) []components.ChatFunctionTool {
	result := make([]components.ChatFunctionTool, len(tools))

	for i, tool := range tools {
		var parameters map[string]any
		if tool.Function.Parameters != nil {
			if params, ok := tool.Function.Parameters.(map[string]any); ok {
				parameters = params
			}
		}

		result[i] = components.CreateChatFunctionToolChatFunctionToolFunction(
			components.ChatFunctionToolFunction{
				Type: components.ChatFunctionToolTypeFunction,
				Function: components.ChatFunctionToolFunctionFunction{
					Name:        tool.Function.Name,
					Description: openrouter.Pointer(tool.Function.Description),
					Parameters:  parameters,
				},
			},
		)
	}

	return result
}

// convertToolChoice converts core tool choice to OpenRouter format.
func (p *Provider) convertToolChoice(choice any) *components.ChatToolChoice {
	switch v := choice.(type) {
	case string:
		switch v {
		case "auto":
			tc := components.CreateChatToolChoiceChatToolChoiceAuto(components.ChatToolChoiceAutoAuto)
			return &tc
		case "none":
			tc := components.CreateChatToolChoiceChatToolChoiceNone(components.ChatToolChoiceNoneNone)
			return &tc
		case "required":
			tc := components.CreateChatToolChoiceChatToolChoiceRequired(components.ChatToolChoiceRequiredRequired)
			return &tc
		}
	case map[string]any:
		// Specific function choice
		if fn, ok := v["function"].(map[string]any); ok {
			if name, ok := fn["name"].(string); ok {
				tc := components.CreateChatToolChoiceChatNamedToolChoice(components.ChatNamedToolChoice{
					Type: components.ChatNamedToolChoiceTypeFunction,
					Function: components.ChatNamedToolChoiceFunction{
						Name: name,
					},
				})
				return &tc
			}
		}
	}
	tc := components.CreateChatToolChoiceChatToolChoiceAuto(components.ChatToolChoiceAutoAuto)
	return &tc
}

// convertResponse converts an OpenRouter response to core format.
func (p *Provider) convertResponse(resp *components.ChatResult) *core.ChatCompletionResponse {
	result := &core.ChatCompletionResponse{
		ID:      resp.ID,
		Object:  "chat.completion",
		Created: resp.Created,
		Model:   resp.Model,
	}

	if resp.Usage != nil {
		result.Usage = core.Usage{
			PromptTokens:     int(resp.Usage.PromptTokens),
			CompletionTokens: int(resp.Usage.CompletionTokens),
			TotalTokens:      int(resp.Usage.TotalTokens),
		}
	}

	if len(resp.Choices) > 0 {
		result.Choices = make([]core.ChatCompletionChoice, len(resp.Choices))
		for i, choice := range resp.Choices {
			var content string
			var toolCalls []core.ToolCall

			// Handle content - get via accessor method
			contentOpt := choice.Message.GetContent()
			if val, ok := contentOpt.Get(); ok && val.Str != nil {
				content = *val.Str
			}

			// Handle tool calls
			msgToolCalls := choice.Message.GetToolCalls()
			if len(msgToolCalls) > 0 {
				toolCalls = make([]core.ToolCall, len(msgToolCalls))
				for j, tc := range msgToolCalls {
					toolCalls[j] = core.ToolCall{
						ID:   tc.ID,
						Type: "function",
						Function: core.ToolFunction{
							Name:      tc.Function.Name,
							Arguments: tc.Function.Arguments,
						},
					}
				}
			}

			var finishReason *string
			if choice.FinishReason != nil {
				reason := string(*choice.FinishReason)
				finishReason = &reason
			}

			result.Choices[i] = core.ChatCompletionChoice{
				Index: int(choice.Index),
				Message: core.Message{
					Role:      core.RoleAssistant,
					Content:   content,
					ToolCalls: toolCalls,
				},
				FinishReason: finishReason,
			}
		}
	}

	// Preserve OpenRouter-specific metadata
	result.ProviderMetadata = map[string]any{
		"openrouter_id": result.ID,
	}

	return result
}

// wrapError converts OpenRouter SDK errors to core errors.
func (p *Provider) wrapError(err error) error {
	if err == nil {
		return nil
	}

	return core.NewAPIError("openrouter", 0, "", err.Error())
}

// ptrValueOrDefault returns the value of a pointer or a default if nil.
func ptrValueOrDefault[T any](ptr *T, def T) T {
	if ptr == nil {
		return def
	}
	return *ptr
}

// streamAdapter wraps an OpenRouter EventStream to implement core.ChatCompletionStream.
type streamAdapter struct {
	stream *stream.EventStream[operations.SendChatCompletionRequestResponseBody]
	done   bool
}

// Recv receives the next chunk from the stream.
func (s *streamAdapter) Recv() (*core.ChatCompletionChunk, error) {
	if s.done {
		return nil, io.EOF
	}

	if !s.stream.Next() {
		s.done = true
		if err := s.stream.Err(); err != nil {
			return nil, err
		}
		return nil, io.EOF
	}

	event := s.stream.Value()
	if event == nil {
		return nil, io.EOF
	}

	chunk := event.Data
	return s.convertChunk(&chunk), nil
}

// Close closes the stream.
func (s *streamAdapter) Close() error {
	s.done = true
	return s.stream.Close()
}

// convertChunk converts an OpenRouter streaming chunk to core format.
func (s *streamAdapter) convertChunk(chunk *components.ChatStreamChunk) *core.ChatCompletionChunk {
	result := &core.ChatCompletionChunk{
		ID:      chunk.ID,
		Object:  "chat.completion.chunk",
		Created: chunk.Created,
		Model:   chunk.Model,
	}

	if len(chunk.Choices) > 0 {
		result.Choices = make([]core.ChatCompletionChoice, len(chunk.Choices))
		for i, choice := range chunk.Choices {
			delta := &core.Message{}

			// Handle delta content via accessor
			contentOpt := choice.Delta.GetContent()
			if val, ok := contentOpt.Get(); ok {
				delta.Content = *val
			}

			if choice.Delta.Role != nil {
				delta.Role = core.Role(*choice.Delta.Role)
			}

			// Handle tool calls in delta
			deltaToolCalls := choice.Delta.GetToolCalls()
			if len(deltaToolCalls) > 0 {
				delta.ToolCalls = make([]core.ToolCall, len(deltaToolCalls))
				for j, tc := range deltaToolCalls {
					delta.ToolCalls[j] = core.ToolCall{
						ID:   ptrValueOrDefault(tc.ID, ""),
						Type: "function",
						Function: core.ToolFunction{
							Name:      ptrValueOrDefault(tc.Function.Name, ""),
							Arguments: ptrValueOrDefault(tc.Function.Arguments, ""),
						},
					}
				}
			}

			var finishReason *string
			if choice.FinishReason != nil {
				reason := string(*choice.FinishReason)
				finishReason = &reason
			}

			result.Choices[i] = core.ChatCompletionChoice{
				Index:        int(choice.Index),
				Delta:        delta,
				FinishReason: finishReason,
			}
		}
	}

	if chunk.Usage != nil {
		result.Usage = &core.Usage{
			PromptTokens:     int(chunk.Usage.PromptTokens),
			CompletionTokens: int(chunk.Usage.CompletionTokens),
			TotalTokens:      int(chunk.Usage.TotalTokens),
		}
	}

	result.ProviderMetadata = map[string]any{
		"openrouter_event": true,
	}

	return result
}

// Ensure streamAdapter implements core.ChatCompletionStream at compile time.
var _ core.ChatCompletionStream = (*streamAdapter)(nil)
