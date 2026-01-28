package llm

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// mockRoundTripper implements http.RoundTripper for testing HTTP requests.
type mockRoundTripper struct {
	handler func(*http.Request) *http.Response
}

func (m *mockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return m.handler(req), nil
}

// helper to patch http.DefaultClient.Transport for test isolation
func withMockHTTPClient(handler func(*http.Request) *http.Response, testFunc func()) {
	origTransport := http.DefaultClient.Transport
	http.DefaultClient.Transport = &mockRoundTripper{handler: handler}
	defer func() { http.DefaultClient.Transport = origTransport }()
	testFunc()
}

func TestSendReviewPrompt_ModelSelection(t *testing.T) {
	client := &Client{
		Provider: "openai",
		APIKey:   "dummy",
		Endpoint: "http://example.com",
		Model:    "arcee-ai/trinity-large-preview:free",
	}

	// Intercept the outgoing request and check the model field
	withMockHTTPClient(func(req *http.Request) *http.Response {
		body, _ := io.ReadAll(req.Body)
		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("Failed to unmarshal request body: %v", err)
		}
		model, ok := reqBody["model"].(string)
		if !ok || model != "arcee-ai/trinity-large-preview:free" {
			t.Errorf("Expected model 'arcee-ai/trinity-large-preview:free', got '%v'", reqBody["model"])
		}
		// Return a minimal valid OpenAI response
		resp := `{"choices":[{"message":{"content":"Test response"}}]}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(resp)),
			Header:     make(http.Header),
		}
	}, func() {
		resp, err := client.SendReviewPrompt("test prompt")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if resp != "Test response" {
			t.Errorf("Expected 'Test response', got '%s'", resp)
		}
	})
}

func TestSendReviewPrompt_DefaultModel(t *testing.T) {
	client := &Client{
		Provider: "openai",
		APIKey:   "dummy",
		Endpoint: "http://example.com",
		// Model is empty, should default to gpt-3.5-turbo
	}

	withMockHTTPClient(func(req *http.Request) *http.Response {
		body, _ := io.ReadAll(req.Body)
		var reqBody map[string]interface{}
		_ = json.Unmarshal(body, &reqBody)
		model, ok := reqBody["model"].(string)
		if !ok || model != "gpt-3.5-turbo" {
			t.Errorf("Expected default model 'gpt-3.5-turbo', got '%v'", reqBody["model"])
		}
		resp := `{"choices":[{"message":{"content":"Default model response"}}]}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(resp)),
			Header:     make(http.Header),
		}
	}, func() {
		resp, err := client.SendReviewPrompt("test prompt")
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if resp != "Default model response" {
			t.Errorf("Expected 'Default model response', got '%s'", resp)
		}
	})
}

func TestSendReviewPrompt_MissingAPIKey(t *testing.T) {
	client := &Client{
		Provider: "openai",
		APIKey:   "",
		Endpoint: "http://example.com",
		Model:    "gpt-3.5-turbo",
	}
	_, err := client.SendReviewPrompt("test prompt")
	if err == nil || !strings.Contains(err.Error(), "missing OpenAI API key") {
		t.Errorf("Expected missing API key error, got: %v", err)
	}
}

func TestSendReviewPrompt_MissingEndpoint(t *testing.T) {
	client := &Client{
		Provider: "openai",
		APIKey:   "dummy",
		Endpoint: "",
		Model:    "gpt-3.5-turbo",
	}
	_, err := client.SendReviewPrompt("test prompt")
	if err == nil || !strings.Contains(err.Error(), "missing OpenAI API endpoint") {
		t.Errorf("Expected missing endpoint error, got: %v", err)
	}
}

func TestSendReviewPrompt_UnsupportedProvider(t *testing.T) {
	client := &Client{
		Provider: "anthropic",
		APIKey:   "dummy",
		Endpoint: "http://example.com",
		Model:    "claude-2",
	}
	_, err := client.SendReviewPrompt("test prompt")
	if err == nil || !strings.Contains(err.Error(), "unsupported LLM provider") {
		t.Errorf("Expected unsupported provider error, got: %v", err)
	}
}

func TestSendReviewPrompt_OpenAIErrorResponse(t *testing.T) {
	client := &Client{
		Provider: "openai",
		APIKey:   "dummy",
		Endpoint: "http://example.com",
		Model:    "gpt-3.5-turbo",
	}
	withMockHTTPClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 400,
			Body:       io.NopCloser(bytes.NewBufferString("bad request")),
			Header:     make(http.Header),
		}
	}, func() {
		_, err := client.SendReviewPrompt("test prompt")
		if err == nil || !strings.Contains(err.Error(), "OpenAI API error") {
			t.Errorf("Expected OpenAI API error, got: %v", err)
		}
	})
}

func TestSendReviewPrompt_InvalidJSONResponse(t *testing.T) {
	client := &Client{
		Provider: "openai",
		APIKey:   "dummy",
		Endpoint: "http://example.com",
		Model:    "gpt-3.5-turbo",
	}
	withMockHTTPClient(func(req *http.Request) *http.Response {
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString("{not json")),
			Header:     make(http.Header),
		}
	}, func() {
		_, err := client.SendReviewPrompt("test prompt")
		if err == nil || !strings.Contains(err.Error(), "failed to parse OpenAI response") {
			t.Errorf("Expected JSON parse error, got: %v", err)
		}
	})
}

func TestSendReviewPrompt_NoChoicesInResponse(t *testing.T) {
	client := &Client{
		Provider: "openai",
		APIKey:   "dummy",
		Endpoint: "http://example.com",
		Model:    "gpt-3.5-turbo",
	}
	withMockHTTPClient(func(req *http.Request) *http.Response {
		resp := `{"choices":[]}`
		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(bytes.NewBufferString(resp)),
			Header:     make(http.Header),
		}
	}, func() {
		_, err := client.SendReviewPrompt("test prompt")
		if err == nil || !strings.Contains(err.Error(), "no choices returned from OpenAI API") {
			t.Errorf("Expected no choices error, got: %v", err)
		}
	})
}
