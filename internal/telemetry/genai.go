package telemetry

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// GenAI Semantic Conventions
// Based on OpenTelemetry GenAI experimental semantic conventions

const (
	// System attributes
	AttrGenAISystem          = "gen_ai.system"           // e.g., "openai", "anthropic"
	AttrGenAIRequestModel    = "gen_ai.request.model"    // Model name
	AttrGenAIResponseModel   = "gen_ai.response.model"   // Actual model used

	// Request attributes
	AttrGenAIRequestMaxTokens     = "gen_ai.request.max_tokens"
	AttrGenAIRequestTemperature   = "gen_ai.request.temperature"
	AttrGenAIRequestTopP          = "gen_ai.request.top_p"
	AttrGenAIRequestFrequencyPenalty = "gen_ai.request.frequency_penalty"
	AttrGenAIRequestPresencePenalty  = "gen_ai.request.presence_penalty"

	// Response attributes
	AttrGenAIResponseID          = "gen_ai.response.id"
	AttrGenAIResponseFinishReason = "gen_ai.response.finish_reason"

	// Usage attributes
	AttrGenAIUsagePromptTokens     = "gen_ai.usage.prompt_tokens"
	AttrGenAIUsageCompletionTokens = "gen_ai.usage.completion_tokens"
	AttrGenAIUsageTotalTokens      = "gen_ai.usage.total_tokens"

	// Tool/Function calling
	AttrGenAIToolCallsCount = "gen_ai.tool_calls.count"
	AttrGenAIToolName       = "gen_ai.tool.name"

	// Custom attributes
	AttrPromptLength    = "prompt.length"
	AttrResponseLength  = "response.length"
	AttrContextWindow   = "context.window"
)

// LLMSpan creates a span for an LLM call with GenAI conventions
type LLMSpan struct {
	span trace.Span
	ctx  context.Context
}

// StartLLMSpan starts a new LLM span
func StartLLMSpan(ctx context.Context, tracer trace.Tracer, operation string, model string) (*LLMSpan, context.Context) {
	ctx, span := tracer.Start(ctx, operation,
		trace.WithSpanKind(trace.SpanKindClient),
		trace.WithAttributes(
			attribute.String(AttrGenAISystem, "openai-compatible"),
			attribute.String(AttrGenAIRequestModel, model),
		),
	)

	return &LLMSpan{
		span: span,
		ctx:  ctx,
	}, ctx
}

// SetRequestParams sets request parameters
func (ls *LLMSpan) SetRequestParams(maxTokens int, temperature float32) {
	ls.span.SetAttributes(
		attribute.Int(AttrGenAIRequestMaxTokens, maxTokens),
		attribute.Float64(AttrGenAIRequestTemperature, float64(temperature)),
	)
}

// SetPrompt sets the prompt (as event to avoid large attributes)
func (ls *LLMSpan) SetPrompt(prompt string) {
	ls.span.AddEvent("prompt",
		trace.WithAttributes(
			attribute.Int(AttrPromptLength, len(prompt)),
			attribute.String("prompt.content", truncate(prompt, 1000)),
		),
	)
}

// SetResponse sets the response
func (ls *LLMSpan) SetResponse(response string, finishReason string) {
	ls.span.SetAttributes(
		attribute.String(AttrGenAIResponseFinishReason, finishReason),
		attribute.Int(AttrResponseLength, len(response)),
	)

	ls.span.AddEvent("response",
		trace.WithAttributes(
			attribute.String("response.content", truncate(response, 1000)),
		),
	)
}

// SetTokenUsage sets token usage statistics
func (ls *LLMSpan) SetTokenUsage(promptTokens, completionTokens, totalTokens int) {
	ls.span.SetAttributes(
		attribute.Int(AttrGenAIUsagePromptTokens, promptTokens),
		attribute.Int(AttrGenAIUsageCompletionTokens, completionTokens),
		attribute.Int(AttrGenAIUsageTotalTokens, totalTokens),
	)
}

// SetToolCalls records tool calls
func (ls *LLMSpan) SetToolCalls(toolCalls []string) {
	ls.span.SetAttributes(
		attribute.Int(AttrGenAIToolCallsCount, len(toolCalls)),
	)

	for i, toolName := range toolCalls {
		ls.span.AddEvent("tool_call",
			trace.WithAttributes(
				attribute.Int("tool_call.index", i),
				attribute.String(AttrGenAIToolName, toolName),
			),
		)
	}
}

// SetError records an error
func (ls *LLMSpan) SetError(err error) {
	ls.span.RecordError(err)
	ls.span.SetStatus(codes.Error, err.Error())
}

// End ends the span
func (ls *LLMSpan) End() {
	ls.span.SetStatus(codes.Ok, "")
	ls.span.End()
}

// EndWithError ends the span with an error
func (ls *LLMSpan) EndWithError(err error) {
	ls.SetError(err)
	ls.span.End()
}

// ToolSpan creates a span for tool execution
type ToolSpan struct {
	span     trace.Span
	ctx      context.Context
	hasError bool
}

// StartToolSpan starts a new tool execution span
func StartToolSpan(ctx context.Context, tracer trace.Tracer, toolName string) (*ToolSpan, context.Context) {
	ctx, span := tracer.Start(ctx, "tool."+toolName,
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("tool.name", toolName),
		),
	)

	return &ToolSpan{
		span: span,
		ctx:  ctx,
	}, ctx
}

// SetParameters sets tool parameters
func (ts *ToolSpan) SetParameters(params string) {
	ts.span.AddEvent("parameters",
		trace.WithAttributes(
			attribute.String("tool.parameters", truncate(params, 500)),
		),
	)
}

// SetResult sets tool execution result
func (ts *ToolSpan) SetResult(result string, success bool) {
	ts.span.SetAttributes(
		attribute.Bool("tool.success", success),
		attribute.Int("tool.result_length", len(result)),
	)

	ts.span.AddEvent("result",
		trace.WithAttributes(
			attribute.String("tool.result", truncate(result, 500)),
		),
	)
}

// SetError records a tool error
func (ts *ToolSpan) SetError(err error) {
	ts.hasError = true
	ts.span.RecordError(err)
	ts.span.SetStatus(codes.Error, err.Error())
}

// End ends the span
func (ts *ToolSpan) End() {
	if !ts.hasError {
		ts.span.SetStatus(codes.Ok, "")
	}
	ts.span.End()
}

// RetrievalSpan creates a span for retrieval operations
type RetrievalSpan struct {
	span trace.Span
	ctx  context.Context
}

// StartRetrievalSpan starts a new retrieval span
func StartRetrievalSpan(ctx context.Context, tracer trace.Tracer, query string) (*RetrievalSpan, context.Context) {
	ctx, span := tracer.Start(ctx, "retrieval.search",
		trace.WithSpanKind(trace.SpanKindInternal),
		trace.WithAttributes(
			attribute.String("retrieval.query", truncate(query, 200)),
		),
	)

	return &RetrievalSpan{
		span: span,
		ctx:  ctx,
	}, ctx
}

// SetResults sets retrieval results
func (rs *RetrievalSpan) SetResults(count int, topScore float32) {
	rs.span.SetAttributes(
		attribute.Int("retrieval.results_count", count),
		attribute.Float64("retrieval.top_score", float64(topScore)),
	)
}

// SetMethod sets the retrieval method used
func (rs *RetrievalSpan) SetMethod(method string) {
	rs.span.SetAttributes(
		attribute.String("retrieval.method", method),
	)
}

// End ends the span
func (rs *RetrievalSpan) End() {
	rs.span.SetStatus(codes.Ok, "")
	rs.span.End()
}

// Helper functions

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// AddCustomAttribute adds a custom attribute to a span
func AddCustomAttribute(span trace.Span, key string, value interface{}) {
	switch v := value.(type) {
	case string:
		span.SetAttributes(attribute.String(key, v))
	case int:
		span.SetAttributes(attribute.Int(key, v))
	case int64:
		span.SetAttributes(attribute.Int64(key, v))
	case float64:
		span.SetAttributes(attribute.Float64(key, v))
	case bool:
		span.SetAttributes(attribute.Bool(key, v))
	}
}
