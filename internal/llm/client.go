package llm

import (
	"context"
	"fmt"

	"github.com/jake/coder/internal/config"
	openai "github.com/sashabaranov/go-openai"
)

type Client struct {
	client *openai.Client
	config *config.LLMConfig
}

func NewClient(cfg *config.LLMConfig) *Client {
	clientConfig := openai.DefaultConfig(cfg.APIKey)
	clientConfig.BaseURL = cfg.Endpoint

	return &Client{
		client: openai.NewClientWithConfig(clientConfig),
		config: cfg,
	}
}

type Message struct {
	Role    string      `json:"role"`
	Content string      `json:"content"`
	Tool    *ToolCall   `json:"tool_call,omitempty"`
	ToolID  string      `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type CompletionRequest struct {
	Messages    []Message
	Tools       []Tool
	Temperature float32
	MaxTokens   int
}

type Tool struct {
	Type     string   `json:"type"`
	Function Function `json:"function"`
}

type Function struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type CompletionResponse struct {
	Content   string
	ToolCalls []ToolCall
	FinishReason string
}

func (c *Client) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	messages := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, msg := range req.Messages {
		messages[i] = openai.ChatCompletionMessage{
			Role:    msg.Role,
			Content: msg.Content,
		}
		if msg.ToolID != "" {
			messages[i].ToolCallID = msg.ToolID
		}
	}

	tools := make([]openai.Tool, len(req.Tools))
	for i, tool := range req.Tools {
		tools[i] = openai.Tool{
			Type: openai.ToolType(tool.Type),
			Function: &openai.FunctionDefinition{
				Name:        tool.Function.Name,
				Description: tool.Function.Description,
				Parameters:  tool.Function.Parameters,
			},
		}
	}

	temperature := req.Temperature
	if temperature == 0 {
		temperature = c.config.Temperature
	}
	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.config.MaxTokens
	}

	chatReq := openai.ChatCompletionRequest{
		Model:       c.config.Model,
		Messages:    messages,
		Tools:       tools,
		Temperature: temperature,
		MaxTokens:   maxTokens,
	}

	resp, err := c.client.CreateChatCompletion(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned")
	}

	choice := resp.Choices[0]
	result := &CompletionResponse{
		Content:      choice.Message.Content,
		FinishReason: string(choice.FinishReason),
	}

	if len(choice.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]ToolCall, len(choice.Message.ToolCalls))
		for i, tc := range choice.Message.ToolCalls {
			result.ToolCalls[i] = ToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
			}
			result.ToolCalls[i].Function.Name = tc.Function.Name
			result.ToolCalls[i].Function.Arguments = tc.Function.Arguments
		}
	}

	return result, nil
}
