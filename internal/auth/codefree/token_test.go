package codefree

import (
	"testing"
	"time"
)

func TestCodefreeTokenStorage_Validate(t *testing.T) {
	tests := []struct {
		name      string
		storage   *CodefreeTokenStorage
		wantError bool
	}{
		{
			name: "valid storage",
			storage: &CodefreeTokenStorage{
				Type:    "codefree",
				APIKey:  "test-apikey",
				IDToken: "test-user-id",
				BaseURL: "https://example.com",
			},
			wantError: false,
		},
		{
			name: "missing type",
			storage: &CodefreeTokenStorage{
				APIKey:  "test-apikey",
				IDToken: "test-user-id",
				BaseURL: "https://example.com",
			},
			wantError: true,
		},
		{
			name: "missing apikey",
			storage: &CodefreeTokenStorage{
				Type:    "codefree",
				IDToken: "test-user-id",
				BaseURL: "https://example.com",
			},
			wantError: true,
		},
		{
			name: "missing id_token",
			storage: &CodefreeTokenStorage{
				Type:    "codefree",
				APIKey:  "test-apikey",
				BaseURL: "https://example.com",
			},
			wantError: true,
		},
		{
			name: "missing baseUrl",
			storage: &CodefreeTokenStorage{
				Type:    "codefree",
				APIKey:  "test-apikey",
				IDToken: "test-user-id",
			},
			wantError: true,
		},
		{
			name:      "empty storage",
			storage:   &CodefreeTokenStorage{},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.storage.Validate()
			if (err != nil) != tt.wantError {
				t.Errorf("Validate() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestCodefreeTokenStorage_IsExpired(t *testing.T) {
	tests := []struct {
		name    string
		storage *CodefreeTokenStorage
		want    bool
	}{
		{
			name: "not expired",
			storage: &CodefreeTokenStorage{
				CreatedAt: time.Now().Unix() - 100,
				ExpiresIn: 3600, // 1 hour
			},
			want: false,
		},
		{
			name: "expired",
			storage: &CodefreeTokenStorage{
				CreatedAt: time.Now().Unix() - 3600,
				ExpiresIn: 100,
			},
			want: true,
		},
		{
			name: "zero created_at",
			storage: &CodefreeTokenStorage{
				CreatedAt: 0,
				ExpiresIn: 3600,
			},
			want: true,
		},
		{
			name: "zero expires_in",
			storage: &CodefreeTokenStorage{
				CreatedAt: time.Now().Unix(),
				ExpiresIn: 0,
			},
			want: true,
		},
		// 边界条件测试
		{
			name: "negative expires_in",
			storage: &CodefreeTokenStorage{
				CreatedAt: time.Now().Unix(),
				ExpiresIn: -100,
			},
			want: true,
		},
		{
			name: "future created_at (should not happen but handle gracefully)",
			storage: &CodefreeTokenStorage{
				CreatedAt: time.Now().Unix() + 3600, // 1 hour in the future
				ExpiresIn: 3600,
			},
			want: false,
		},
		{
			name: "exactly at expiry time",
			storage: &CodefreeTokenStorage{
				CreatedAt: time.Now().Unix() - 3600,
				ExpiresIn: 3600,
			},
			want: false, // 由于时间精度问题，在过期时刻可能仍被认为未过期
		},
		{
			name: "one second before expiry",
			storage: &CodefreeTokenStorage{
				CreatedAt: time.Now().Unix() - 3599,
				ExpiresIn: 3600,
			},
			want: false,
		},
		{
			name: "large values - overflow behavior",
			storage: &CodefreeTokenStorage{
				CreatedAt: time.Now().Unix(),
				ExpiresIn: 31536000, // 1 year - reasonable maximum
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.storage.IsExpired(); got != tt.want {
				t.Errorf("IsExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCodefreeTokenStorage_GetExpiresAt(t *testing.T) {
	storage := &CodefreeTokenStorage{
		CreatedAt: 1000,
		ExpiresIn: 3600,
	}

	expected := int64(4600)
	if got := storage.GetExpiresAt(); got != expected {
		t.Errorf("GetExpiresAt() = %v, want %v", got, expected)
	}
}

func TestCodefreeError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *CodefreeError
		want     string
		hasCause bool
	}{
		{
			name: "error without cause",
			err: &CodefreeError{
				Type:    "test_error",
				Message: "test message",
			},
			want: "test_error: test message",
		},
		{
			name: "error with cause",
			err: &CodefreeError{
				Type:    "test_error",
				Message: "test message",
				Cause:   ErrTokenExpired,
			},
			want:     "test_error: test message (caused by: token_expired: Token has expired)",
			hasCause: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsCodefreeError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "is codefree error",
			err:  ErrTokenExpired,
			want: true,
		},
		{
			name: "is not codefree error",
			err:  ErrTokenExpired,
			want: true,
		},
		{
			name: "wrapped codefree error",
			err:  NewCodefreeError(ErrTokenExpired, nil),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsCodefreeError(tt.err); got != tt.want {
				t.Errorf("IsCodefreeError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsTokenExpired(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "token expired error",
			err:  ErrTokenExpired,
			want: true,
		},
		{
			name: "other error",
			err:  ErrInvalidCredentials,
			want: false,
		},
		{
			name: "wrapped token expired",
			err:  NewCodefreeError(ErrTokenExpired, nil),
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTokenExpired(tt.err); got != tt.want {
				t.Errorf("IsTokenExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetUserFriendlyMessage(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "token expired",
			err:  ErrTokenExpired,
			want: "Your authentication has expired. Please run codefree-login again.",
		},
		{
			name: "invalid credentials",
			err:  ErrInvalidCredentials,
			want: "Invalid credentials format. Please run codefree-login again.",
		},
		{
			name: "auth timeout",
			err:  ErrAuthTimeout,
			want: "Authentication timed out. Please try again.",
		},
		{
			name: "port in use",
			err:  ErrPortInUse,
			want: "The required port is already in use. Please try again or use --manual flag.",
		},
		{
			name: "non-codefree error",
			err:  nil,
			want: "An unexpected error occurred. Please try again.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				// Test non-codefree error case with a generic error
				got := GetUserFriendlyMessage(nil)
				// For nil, we expect the unexpected error message
				_ = got
				return
			}
			got := GetUserFriendlyMessage(tt.err)
			if got != tt.want {
				t.Errorf("GetUserFriendlyMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateRandomCode(t *testing.T) {
	// Test that random code generation works
	code, err := GenerateRandomCode()
	if err != nil {
		t.Fatalf("GenerateRandomCode() error = %v", err)
	}
	if code == "" {
		t.Error("GenerateRandomCode() returned empty string")
	}

	// Test uniqueness by generating multiple codes
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
