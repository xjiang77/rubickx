package adapter

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
)

type contractCase struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Input    struct {
		Request struct {
			Model         string    `json:"model"`
			Messages      []Message `json:"messages"`
			RequiresTools bool      `json:"requires_tools"`
		} `json:"request"`
		ProviderResponse *struct {
			Output   string `json:"output"`
			StopCode string `json:"stop_code"`
		} `json:"provider_response"`
		ProviderError *ProviderError `json:"provider_error"`
	} `json:"input"`
	Expected struct {
		MappedRequest *LegacyRequest `json:"mapped_request"`
		Response      *struct {
			Content      string `json:"content"`
			FinishReason string `json:"finish_reason"`
		} `json:"response"`
		ProviderCalls int `json:"provider_calls"`
	} `json:"expected"`
	ExpectedError *struct {
		Code      string `json:"code"`
		Retryable bool   `json:"retryable"`
	} `json:"expected_error"`
}

type contractFixture struct {
	PatternID string         `json:"pattern_id"`
	Cases     []contractCase `json:"cases"`
}

type fakeLegacyClient struct {
	response    LegacyResponse
	err         error
	lastRequest *LegacyRequest
	calls       int
}

func (f *fakeLegacyClient) Generate(_ context.Context, request LegacyRequest) (LegacyResponse, error) {
	f.calls++
	f.lastRequest = &request
	return f.response, f.err
}

func loadFixture(t *testing.T) contractFixture {
	t.Helper()
	body, err := os.ReadFile("../fixtures/contract.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture contractFixture
	if err := json.Unmarshal(body, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func findCase(t *testing.T, fixture contractFixture, id string) contractCase {
	t.Helper()
	for _, candidate := range fixture.Cases {
		if candidate.ID == id {
			return candidate
		}
	}
	t.Fatalf("missing contract case %q", id)
	return contractCase{}
}

func requestFrom(testCase contractCase) ChatRequest {
	return ChatRequest{
		Model:         testCase.Input.Request.Model,
		Messages:      testCase.Input.Request.Messages,
		RequiresTools: testCase.Input.Request.RequiresTools,
	}
}

func TestMapsSharedContractFixture(t *testing.T) {
	testCase := findCase(t, loadFixture(t), "maps_request_and_response")
	provider := testCase.Input.ProviderResponse
	client := &fakeLegacyClient{response: LegacyResponse{Output: provider.Output, StopCode: provider.StopCode}}
	adapter := NewLegacyProviderAdapter(client)

	response, err := adapter.Complete(context.Background(), requestFrom(testCase))
	if err != nil {
		t.Fatal(err)
	}

	if *client.lastRequest != *testCase.Expected.MappedRequest {
		t.Fatalf("mapped request = %#v, want %#v", *client.lastRequest, *testCase.Expected.MappedRequest)
	}
	if response.Content != testCase.Expected.Response.Content || response.FinishReason != testCase.Expected.Response.FinishReason {
		t.Fatalf("response = %#v, want %#v", response, *testCase.Expected.Response)
	}
}

func TestFailsExplicitlyForUnknownModel(t *testing.T) {
	testCase := findCase(t, loadFixture(t), "rejects_unknown_model")
	_, err := NewLegacyProviderAdapter(&fakeLegacyClient{}).Complete(context.Background(), requestFrom(testCase))
	assertNormalizedError(t, err, testCase.ExpectedError.Code, testCase.ExpectedError.Retryable)
}

func TestNormalizesProviderError(t *testing.T) {
	testCase := findCase(t, loadFixture(t), "normalizes_provider_error")
	client := &fakeLegacyClient{err: testCase.Input.ProviderError}
	_, err := NewLegacyProviderAdapter(client).Complete(context.Background(), requestFrom(testCase))
	assertNormalizedError(t, err, testCase.ExpectedError.Code, testCase.ExpectedError.Retryable)
}

func TestRejectsUnsupportedBeforeProviderCall(t *testing.T) {
	testCase := findCase(t, loadFixture(t), "rejects_unsupported_before_provider_call")
	client := &fakeLegacyClient{}
	_, err := NewLegacyProviderAdapter(client).Complete(context.Background(), requestFrom(testCase))
	assertNormalizedError(t, err, testCase.ExpectedError.Code, testCase.ExpectedError.Retryable)
	if client.calls != testCase.Expected.ProviderCalls {
		t.Fatalf("provider calls = %d, want %d", client.calls, testCase.Expected.ProviderCalls)
	}
}

func assertNormalizedError(t *testing.T, err error, code string, retryable bool) {
	t.Helper()
	var normalized *NormalizedError
	if !errors.As(err, &normalized) {
		t.Fatalf("error = %T, want *NormalizedError", err)
	}
	if normalized.Code != code || normalized.Retryable != retryable {
		t.Fatalf("normalized error = %#v, want code=%q retryable=%v", normalized, code, retryable)
	}
}
