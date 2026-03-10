package codefree

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCodefreeOAuthServer_BuildStateParam(t *testing.T) {
	server := NewCodefreeOAuthServer()

	// Start server first to get a port
	if err := server.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer func() { _ = server.Stop(context.Background()) }()

	randomCode := "test-random-code"
	state := server.BuildStateParam(randomCode)

	// State should be valid base64 and contain the callback URL
	decoded := decodeBase64(t, state)

	// Decoded should contain the random code
	if !strings.Contains(decoded, randomCode) {
		t.Errorf("BuildStateParam() decoded = %q, should contain randomCode %q", decoded, randomCode)
	}

	// Should contain localhost
	if !strings.Contains(decoded, "127.0.0.1") {
		t.Errorf("BuildStateParam() decoded = %q, should contain 127.0.0.1", decoded)
	}
}

func TestCodefreeOAuthServer_StartAndStop(t *testing.T) {
	server := NewCodefreeOAuthServer()

	// Start server
	err := server.Start()
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}

	// Verify server is running
	if !server.IsRunning() {
		t.Error("IsRunning() should return true after Start()")
	}

	// Verify port was assigned
	port := server.GetPort()
	if port <= 0 {
		t.Errorf("GetPort() = %d, want > 0", port)
	}

	// Stop server
	err = server.Stop(context.Background())
	if err != nil {
		t.Fatalf("Stop() error = %v", err)
	}

	// Verify server is not running
	if server.IsRunning() {
		t.Error("IsRunning() should return false after Stop()")
	}
}

func TestCodefreeOAuthServer_MultipleStartAttempts(t *testing.T) {
	server := NewCodefreeOAuthServer()

	// First start
	err := server.Start()
	if err != nil {
		t.Fatalf("First Start() error = %v", err)
	}
	port1 := server.GetPort()

	// Stop
	_ = server.Stop(context.Background())

	// Second start should work
	err = server.Start()
	if err != nil {
		t.Fatalf("Second Start() error = %v", err)
	}
	port2 := server.GetPort()

	// Ports should be assigned
	if port1 <= 0 || port2 <= 0 {
		t.Errorf("Ports should be > 0, got port1=%d, port2=%d", port1, port2)
	}

	_ = server.Stop(context.Background())
}

func TestCodefreeOAuthServer_GetPort(t *testing.T) {
	server := NewCodefreeOAuthServer()

	// Port should be 0 before start
	if port := server.GetPort(); port != 0 {
		t.Errorf("GetPort() before start = %d, want 0", port)
	}

	// Start and check port
	if err := server.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer func() { _ = server.Stop(context.Background()) }()

	port := server.GetPort()
	if port <= 0 || port > 65535 {
		t.Errorf("GetPort() = %d, want valid port number", port)
	}
}

func TestCodefreeOAuthServer_SetExpectedRandomCode(t *testing.T) {
	server := NewCodefreeOAuthServer()
	code := "test-code"

	server.SetExpectedRandomCode(code)

	// This is an internal state, we can't verify directly without callback
	// Just ensure it doesn't panic
}

func TestCodefreeOAuthServer_SetManualMode(t *testing.T) {
	server := NewCodefreeOAuthServer()

	// Default should be false
	if server.IsManualMode() {
		t.Error("IsManualMode() should return false by default")
	}

	// Set to true
	server.SetManualMode(true)
	if !server.IsManualMode() {
		t.Error("IsManualMode() should return true after SetManualMode(true)")
	}

	// Set back to false
	server.SetManualMode(false)
	if server.IsManualMode() {
		t.Error("IsManualMode() should return false after SetManualMode(false)")
	}
}

func TestCodefreeOAuthServer_CallbackHandler(t *testing.T) {
	server := NewCodefreeOAuthServer()
	randomCode := "callback-test-code"
	server.SetExpectedRandomCode(randomCode)

	// Create test server using the handler directly
	ts := httptest.NewServer(http.HandlerFunc(server.handleCallback))
	defer ts.Close()

	// Test successful callback
	resp, err := http.Get(ts.URL + "?code=test-code&randomCode=" + randomCode)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Callback status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}

func TestCodefreeOAuthServer_CallbackHandler_Error(t *testing.T) {
	server := NewCodefreeOAuthServer()

	ts := httptest.NewServer(http.HandlerFunc(server.handleCallback))
	defer ts.Close()

	// Test error callback
	resp, err := http.Get(ts.URL + "?error=access_denied&error_description=User%20denied")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should return 400 (error page)
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Callback status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestCodefreeOAuthServer_CallbackHandler_InvalidRandomCode(t *testing.T) {
	server := NewCodefreeOAuthServer()
	server.SetExpectedRandomCode("expected-code")

	ts := httptest.NewServer(http.HandlerFunc(server.handleCallback))
	defer ts.Close()

	// Test with wrong randomCode
	resp, err := http.Get(ts.URL + "?code=test-code&randomCode=wrong-code")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should return error
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Callback status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestCodefreeOAuthServer_CallbackHandler_MissingCode(t *testing.T) {
	server := NewCodefreeOAuthServer()

	ts := httptest.NewServer(http.HandlerFunc(server.handleCallback))
	defer ts.Close()

	// Test without code parameter
	resp, err := http.Get(ts.URL + "?state=test-state")
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should return error
	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Callback status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestCodefreeOAuthServer_CallbackHandler_WrongMethod(t *testing.T) {
	server := NewCodefreeOAuthServer()

	ts := httptest.NewServer(http.HandlerFunc(server.handleCallback))
	defer ts.Close()

	// Test POST request (should be rejected)
	resp, err := http.Post(ts.URL, "application/json", strings.NewReader("{}"))
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Should return 405 Method Not Allowed
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("Callback status = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
	}
}

func TestGenerateRandomCode_Uniqueness(t *testing.T) {
	code, err := GenerateRandomCode()
	if err != nil {
		t.Fatalf("GenerateRandomCode() error = %v", err)
	}
	if code == "" {
		t.Error("GenerateRandomCode() returned empty string")
	}

	// Generate multiple codes and verify uniqueness
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code, err := GenerateRandomCode()
		if err != nil {
			t.Fatalf("GenerateRandomCode() error = %v", err)
		}
		if codes[code] {
			t.Errorf("GenerateRandomCode() generated duplicate code: %s", code)
		}
		codes[code] = true
	}
}

func TestExtractRandomCodeFromState(t *testing.T) {
	// Create a valid state with randomCode
	stateData := "http://127.0.0.1:8080/oauth2callback?from=cli&randomCode=test123"
	state := encodeBase64String(stateData)

	code, err := ExtractRandomCodeFromState(state)
	if err != nil {
		t.Fatalf("ExtractRandomCodeFromState() error = %v", err)
	}

	if code != "test123" {
		t.Errorf("ExtractRandomCodeFromState() = %q, want %q", code, "test123")
	}
}

func TestExtractRandomCodeFromState_Invalid(t *testing.T) {
	// Test invalid base64
	_, err := ExtractRandomCodeFromState("not-valid-base64!!!")
	if err == nil {
		t.Error("ExtractRandomCodeFromState() should return error for invalid base64")
	}

	// Test missing randomCode
	state := encodeBase64String("http://127.0.0.1:8080/oauth2callback?from=cli")
	_, err = ExtractRandomCodeFromState(state)
	if err == nil {
		t.Error("ExtractRandomCodeFromState() should return error when randomCode is missing")
	}
}

func TestExtractCodeFromURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantCode  string
		wantError bool
	}{
		{
			name:      "valid code",
			url:       "http://example.com/callback?code=abc123&state=test",
			wantCode:  "abc123",
			wantError: false,
		},
		{
			name:      "no query params",
			url:       "http://example.com/callback",
			wantCode:  "",
			wantError: true,
		},
		{
			name:      "no code param",
			url:       "http://example.com/callback?state=test",
			wantCode:  "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := ExtractCodeFromURL(tt.url)
			if (err != nil) != tt.wantError {
				t.Errorf("ExtractCodeFromURL() error = %v, wantError %v", err, tt.wantError)
				return
			}
			if code != tt.wantCode {
				t.Errorf("ExtractCodeFromURL() = %q, want %q", code, tt.wantCode)
			}
		})
	}
}

func TestBuildStateParamJSON(t *testing.T) {
	state := BuildStateParamJSON(8080, "test-code")

	// Should be valid base64
	parsed, err := ParseStateParamJSON(state)
	if err != nil {
		t.Fatalf("ParseStateParamJSON() error = %v", err)
	}

	if parsed.RandomCode != "test-code" {
		t.Errorf("RandomCode = %q, want %q", parsed.RandomCode, "test-code")
	}
	if !strings.Contains(parsed.CallbackURL, "8080") {
		t.Errorf("CallbackURL = %q, should contain port 8080", parsed.CallbackURL)
	}
}

func TestGetRandomPortString(t *testing.T) {
	server := NewCodefreeOAuthServer()

	if err := server.Start(); err != nil {
		t.Fatalf("Start() error: %v", err)
	}
	defer func() { _ = server.Stop(context.Background()) }()

	portStr := server.GetRandomPortString()
	if portStr == "" {
		t.Error("GetRandomPortString() returned empty string")
	}
}

// Helper functions

func decodeBase64(t *testing.T, s string) string {
	t.Helper()
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}
	return string(decoded)
}

func encodeBase64String(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
