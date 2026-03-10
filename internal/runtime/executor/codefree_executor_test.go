package executor

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	cliproxyauth "github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/auth"
)

// createTestRequest creates a new HTTP request for testing
func createTestRequest(t *testing.T, rawURL, body string) *http.Request {
	t.Helper()
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("Failed to parse URL: %v", err)
	}
	return &http.Request{
		Method: http.MethodPost,
		URL:    parsedURL,
		Header: http.Header{},
	}
}

func TestCodefreeExecutor_Identifier(t *testing.T) {
	cfg := &config.Config{}
	executor := NewCodefreeExecutor(cfg)

	if got := executor.Identifier(); got != "codefree" {
		t.Errorf("Identifier() = %q, want %q", got, "codefree")
	}
}

func TestCodefreeExecutor_PrepareRequest_NilRequest(t *testing.T) {
	cfg := &config.Config{}
	executor := NewCodefreeExecutor(cfg)

	err := executor.PrepareRequest(nil, nil, nil)
	if err != nil {
		t.Errorf("PrepareRequest(nil, nil, nil) should return nil, got %v", err)
	}
}

func TestCodefreeExecutor_PrepareRequest_InjectsHeaders(t *testing.T) {
	cfg := &config.Config{}
	executor := NewCodefreeExecutor(cfg)

	req := createTestRequest(t, "https://example.com/v1/chat/completions", `{"model":"test-model"}`)
	auth := &cliproxyauth.Auth{
		Attributes: map[string]string{
			"api_key": "test-api-key",
			"user_id": "test-user-id",
		},
	}

	err := executor.PrepareRequest(req, auth, []byte(`{"model":"test-model"}`))
	if err != nil {
		t.Fatalf("PrepareRequest() error = %v", err)
	}

	// Check injected headers
	if got := req.Header.Get("apiKey"); got != "test-api-key" {
		t.Errorf("apiKey header = %q, want %q", got, "test-api-key")
	}
	if got := req.Header.Get("userId"); got != "test-user-id" {
		t.Errorf("userId header = %q, want %q", got, "test-user-id")
	}
	if got := req.Header.Get("modelName"); got != "test-model" {
		t.Errorf("modelName header = %q, want %q", got, "test-model")
	}
	if got := req.Header.Get("clientType"); got != "codefree-cli" {
		t.Errorf("clientType header = %q, want %q", got, "codefree-cli")
	}
}

func TestCodefreeExecutor_PrepareRequest_ExtractsModelName(t *testing.T) {
	cfg := &config.Config{}
	executor := NewCodefreeExecutor(cfg)

	tests := []struct {
		name          string
		requestBody   string
		expectedModel string
	}{
		{
			name:          "simple model",
			requestBody:   `{"model":"gpt-4"}`,
			expectedModel: "gpt-4",
		},
		{
			name:        "model with thinking suffix",
			requestBody: `{"model":"deepseek-v3:thinking-16384"}`,
			// Colon-format suffix is NOT parsed by thinking.ParseSuffix (uses parenthesis format)
			expectedModel: "deepseek-v3:thinking-16384",
		},
		{
			name:        "model with parenthesis thinking suffix",
			requestBody: `{"model":"deepseek-v3(thinking-16384)"}`,
			// Parenthesis-format suffix IS parsed by thinking.ParseSuffix
			expectedModel: "deepseek-v3",
		},
		{
			name:          "empty model",
			requestBody:   `{"model":""}`,
			expectedModel: "",
		},
		{
			name:          "no model field",
			requestBody:   `{"prompt":"hello"}`,
			expectedModel: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createTestRequest(t, "https://example.com/v1/chat/completions", tt.requestBody)
			auth := &cliproxyauth.Auth{
				Attributes: map[string]string{
					"api_key": "key",
					"user_id": "user",
				},
			}

			err := executor.PrepareRequest(req, auth, []byte(tt.requestBody))
			if err != nil {
				t.Fatalf("PrepareRequest() error = %v", err)
			}

			got := req.Header.Get("modelName")
			if got != tt.expectedModel {
				t.Errorf("modelName header = %q, want %q", got, tt.expectedModel)
			}
		})
	}
}

func TestCodefreeExecutor_PrepareRequest_MissingAuth(t *testing.T) {
	cfg := &config.Config{}
	executor := NewCodefreeExecutor(cfg)

	req := createTestRequest(t, "https://example.com/v1/chat/completions", `{"model":"test"}`)

	err := executor.PrepareRequest(req, nil, []byte(`{"model":"test"}`))
	if err != nil {
		t.Fatalf("PrepareRequest() error = %v", err)
	}

	// Headers should be empty when auth is nil
	if got := req.Header.Get("apiKey"); got != "" {
		t.Errorf("apiKey header should be empty, got %q", got)
	}
	if got := req.Header.Get("userId"); got != "" {
		t.Errorf("userId header should be empty, got %q", got)
	}
	// modelName should still be extracted from body
	if got := req.Header.Get("modelName"); got != "test" {
		t.Errorf("modelName header = %q, want %q", got, "test")
	}
	// clientType is always set
	if got := req.Header.Get("clientType"); got != "codefree-cli" {
		t.Errorf("clientType header = %q, want %q", got, "codefree-cli")
	}
}

func TestCodefreeExecutor_Refresh_ReturnsError(t *testing.T) {
	cfg := &config.Config{}
	executor := NewCodefreeExecutor(cfg)

	_, err := executor.Refresh(nil, nil)
	if err == nil {
		t.Error("Refresh() should return error for Codefree provider")
	}
}
