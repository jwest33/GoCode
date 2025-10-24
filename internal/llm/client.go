package llm

import (
	"context"
	"fmt"
	"strings"

	"github.com/jake/gocode/internal/config"
	openai "github.com/sashabaranov/go-openai"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type Client struct {
	client *openai.Client
	config *config.LLMConfig
	tracer trace.Tracer
}

func NewClient(cfg *config.LLMConfig) *Client {
	clientConfig := openai.DefaultConfig(cfg.APIKey)
	clientConfig.BaseURL = cfg.Endpoint

	return &Client{
		client: openai.NewClientWithConfig(clientConfig),
		config: cfg,
		tracer: trace.NewNoopTracerProvider().Tracer("noop"),
	}
}

// SetTracer sets the tracer for this client
func (c *Client) SetTracer(tracer trace.Tracer) {
	c.tracer = tracer
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
	// Start telemetry span
	ctx, span := c.tracer.Start(ctx, "llm.completion",
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	// Set GenAI attributes
	span.SetAttributes(
		attribute.String("gen_ai.system", "openai-compatible"),
		attribute.String("gen_ai.request.model", c.config.Model),
		attribute.Float64("gen_ai.request.temperature", float64(req.Temperature)),
		attribute.Int("gen_ai.request.max_tokens", req.MaxTokens),
	)

	// Build prompt for logging
	promptBuilder := strings.Builder{}
	for _, msg := range req.Messages {
		promptBuilder.WriteString(fmt.Sprintf("[%s] %s\n", msg.Role, msg.Content))
	}

	// Log prompt as event (truncated)
	span.AddEvent("prompt",
		trace.WithAttributes(attribute.Int("prompt.length", promptBuilder.Len())),
	)

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
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, fmt.Errorf("completion failed: %w", err)
	}

	if len(resp.Choices) == 0 {
		err := fmt.Errorf("no completion choices returned")
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	choice := resp.Choices[0]
	result := &CompletionResponse{
		Content:      choice.Message.Content,
		FinishReason: string(choice.FinishReason),
	}

	// Set response attributes
	span.SetAttributes(
		attribute.String("gen_ai.response.finish_reason", string(choice.FinishReason)),
		attribute.Int("gen_ai.usage.prompt_tokens", resp.Usage.PromptTokens),
		attribute.Int("gen_ai.usage.completion_tokens", resp.Usage.CompletionTokens),
		attribute.Int("gen_ai.usage.total_tokens", resp.Usage.TotalTokens),
	)

	if len(choice.Message.ToolCalls) > 0 {
		result.ToolCalls = make([]ToolCall, len(choice.Message.ToolCalls))
		toolNames := make([]string, len(choice.Message.ToolCalls))

		for i, tc := range choice.Message.ToolCalls {
			result.ToolCalls[i] = ToolCall{
				ID:   tc.ID,
				Type: string(tc.Type),
			}
			result.ToolCalls[i].Function.Name = tc.Function.Name
			result.ToolCalls[i].Function.Arguments = tc.Function.Arguments
			toolNames[i] = tc.Function.Name
		}

		// Log tool calls
		span.SetAttributes(
			attribute.Int("gen_ai.tool_calls.count", len(toolNames)),
		)
		span.AddEvent("tool_calls",
			trace.WithAttributes(attribute.String("tools", strings.Join(toolNames, ", "))),
		)
	}

	// Log response
	span.AddEvent("response",
		trace.WithAttributes(attribute.Int("response.length", len(result.Content))),
	)

	span.SetStatus(codes.Ok, "")
	return result, nil
}
