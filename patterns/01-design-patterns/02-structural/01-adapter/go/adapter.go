package adapter

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model         string
	Messages      []Message
	RequiresTools bool
}

type ChatResponse struct {
	Content      string `json:"content"`
	FinishReason string `json:"finish_reason"`
}

type LegacyRequest struct {
	Deployment string `json:"deployment"`
	Prompt     string `json:"prompt"`
}

type LegacyResponse struct {
	Output   string `json:"output"`
	StopCode string `json:"stop_code"`
}

type LegacyClient interface {
	Generate(context.Context, LegacyRequest) (LegacyResponse, error)
}

type ProviderError struct {
	Code    string
	Message string
}

func (e *ProviderError) Error() string {
	return e.Message
}

type NormalizedError struct {
	Code      string
	Retryable bool
	Message   string
}

func (e *NormalizedError) Error() string {
	return e.Message
}

type LegacyProviderAdapter struct {
	client LegacyClient
}

func NewLegacyProviderAdapter(client LegacyClient) *LegacyProviderAdapter {
	return &LegacyProviderAdapter{client: client}
}

func (a *LegacyProviderAdapter) Complete(ctx context.Context, request ChatRequest) (ChatResponse, error) {
	if request.RequiresTools {
		return ChatResponse{}, &NormalizedError{
			Code:      "unsupported_capability",
			Retryable: false,
			Message:   "legacy provider does not support tools",
		}
	}

	deployment, ok := map[string]string{"chat-pro": "legacy-chat-pro"}[request.Model]
	if !ok {
		return ChatResponse{}, &NormalizedError{
			Code:      "invalid_request",
			Retryable: false,
			Message:   fmt.Sprintf("unknown model: %s", request.Model),
		}
	}

	parts := make([]string, 0, len(request.Messages))
	for _, message := range request.Messages {
		parts = append(parts, message.Role+": "+message.Content)
	}

	legacyResponse, err := a.client.Generate(ctx, LegacyRequest{
		Deployment: deployment,
		Prompt:     strings.Join(parts, "\n"),
	})
	if err != nil {
		return ChatResponse{}, normalizeError(err)
	}

	finishReason := map[string]string{"stop": "stop", "length": "length"}[legacyResponse.StopCode]
	if finishReason == "" {
		finishReason = "unknown"
	}

	return ChatResponse{
		Content:      legacyResponse.Output,
		FinishReason: finishReason,
	}, nil
}

func normalizeError(err error) error {
	var providerError *ProviderError
	if !errors.As(err, &providerError) {
		return &NormalizedError{Code: "upstream_failure", Retryable: true, Message: err.Error()}
	}

	switch providerError.Code {
	case "OVER_QUOTA":
		return &NormalizedError{Code: "rate_limited", Retryable: true, Message: providerError.Message}
	case "BAD_REQUEST":
		return &NormalizedError{Code: "invalid_request", Retryable: false, Message: providerError.Message}
	default:
		return &NormalizedError{Code: "upstream_failure", Retryable: true, Message: providerError.Message}
	}
}
