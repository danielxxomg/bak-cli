package cloud

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// withOAuthServer creates an httptest.Server that mocks GitHub OAuth
// Device Flow endpoints. The handler is a state machine that returns
// different responses on successive POSTs.
func withOAuthServer(deviceResp any, tokenResps []any) (*httptest.Server, *DeviceClient) {
	tokenIdx := 0

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		body, _ := io.ReadAll(r.Body)

		switch {
		case strings.HasSuffix(r.URL.Path, "/login/device/code"):
			vals, _ := url.ParseQuery(string(body))
			if vals.Get("client_id") == "" {
				json.NewEncoder(w).Encode(map[string]string{
					"error":             "invalid_request",
					"error_description": "client_id is required",
				})
				return
			}
			if vals.Get("scope") != "gist" {
				json.NewEncoder(w).Encode(map[string]string{
					"error":             "invalid_scope",
					"error_description": "scope must be gist",
				})
				return
			}
			json.NewEncoder(w).Encode(deviceResp)

		case strings.HasSuffix(r.URL.Path, "/login/oauth/access_token"):
			vals, _ := url.ParseQuery(string(body))
			if vals.Get("grant_type") != "urn:ietf:params:oauth:grant-type:device_code" {
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(map[string]string{
					"error": "unsupported_grant_type",
				})
				return
			}
			if tokenIdx >= len(tokenResps) {
				json.NewEncoder(w).Encode(map[string]string{
					"error": "expired_token",
				})
				return
			}
			json.NewEncoder(w).Encode(tokenResps[tokenIdx])
			tokenIdx++

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	client := &DeviceClient{
		ClientID:   "test-client-id",
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		Out:        io.Discard,
		sleepFn:    func(d time.Duration) {}, // no-op for fast tests
	}

	return srv, client
}

// deviceBaseResp is a reusable device code success response for tests.
func deviceBaseResp() map[string]any {
	return map[string]any{
		"device_code":      "dev-abc123",
		"user_code":        "ABCD-1234",
		"verification_uri": "https://github.com/login/device",
		"expires_in":       900,
		"interval":         0,
	}
}

// tokenSuccess returns a token success response.
func tokenSuccess(tok string) map[string]string {
	return map[string]string{"access_token": tok, "token_type": "bearer", "scope": "gist"}
}

// tokenError returns a token error response.
func tokenError(code string) map[string]string {
	return map[string]string{"error": code}
}

// --- Table-Driven Polling Tests ---

func TestDeviceClient_PollingStates(t *testing.T) {
	tests := []struct {
		name       string
		tokenResps []any
		wantToken  string
		wantErr    string
	}{
		{
			name:       "success_on_first_poll",
			tokenResps: []any{tokenSuccess("gho_immediate")},
			wantToken:  "gho_immediate",
		},
		{
			name:       "success_after_pending",
			tokenResps: []any{tokenError("authorization_pending"), tokenSuccess("gho_after_pending")},
			wantToken:  "gho_after_pending",
		},
		{
			name:       "success_after_slow_down",
			tokenResps: []any{tokenError("slow_down"), tokenSuccess("gho_after_slow")},
			wantToken:  "gho_after_slow",
		},
		{
			name:       "expired_token",
			tokenResps: []any{tokenError("expired_token")},
			wantErr:    "expired",
		},
		{
			name:       "access_denied",
			tokenResps: []any{tokenError("access_denied")},
			wantErr:    "denied",
		},
		{
			name:       "empty_response_triggers_pending_loop",
			tokenResps: []any{map[string]string{}}, // neither token nor error — treated as pending
			wantErr:    "expired",                  // subsequent poll after empty response exhausts queue → expired_token
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, client := withOAuthServer(deviceBaseResp(), tt.tokenResps)
			defer srv.Close()

			token, err := client.RequestToken()

			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
				}
				return
			}

			if tt.wantToken != "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if token != tt.wantToken {
					t.Errorf("token = %q, want %q", token, tt.wantToken)
				}
			}
		})
	}
}

// --- Table-Driven Device Code Request Tests ---

func TestDeviceClient_DeviceCodeErrors(t *testing.T) {
	tests := []struct {
		name     string
		clientID string
		baseURL  string
		wantErr  string
	}{
		{
			name:     "missing_client_id",
			clientID: "",
			wantErr:  "client_id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, client := withOAuthServer(deviceBaseResp(), []any{tokenSuccess("x")})
			defer srv.Close()

			if tt.clientID != "test-client-id" {
				client.ClientID = tt.clientID
			}

			_, err := client.RequestToken()
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %q, want to contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}

func TestDeviceClient_HttpError(t *testing.T) {
	client := &DeviceClient{
		ClientID:   "test-id",
		HTTPClient: &http.Client{Timeout: 1 * time.Millisecond},
		BaseURL:    "http://127.0.0.1:1", // nothing listening
		Out:        io.Discard,
		sleepFn:    func(d time.Duration) {},
	}
	_, err := client.RequestToken()
	if err == nil {
		t.Fatal("expected connection error, got nil")
	}
}

func TestDeviceClient_HTTP500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/login/device/code") {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	client := &DeviceClient{
		ClientID:   "test-id",
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		Out:        io.Discard,
		sleepFn:    func(d time.Duration) {},
	}

	_, err := client.RequestToken()
	if err == nil {
		t.Fatal("expected error for 500 status, got nil")
	}
}

// --- Browser & Clipboard Tests ---

func TestDeviceClient_OpenBrowserCalled(t *testing.T) {
	var openedURL string
	browserCalled := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/login/device/code"):
			json.NewEncoder(w).Encode(deviceBaseResp())
		case strings.HasSuffix(r.URL.Path, "/login/oauth/access_token"):
			json.NewEncoder(w).Encode(tokenSuccess("gho_token"))
		}
	}))
	defer srv.Close()

	client := &DeviceClient{
		ClientID:   "test-id",
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		Out:        io.Discard,
		sleepFn:    func(d time.Duration) {},
		OpenBrowser: func(url string) error {
			openedURL = url
			browserCalled <- struct{}{}
			return nil
		},
	}

	_, err := client.RequestToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case <-browserCalled:
		if !strings.Contains(openedURL, "github.com/login/device") {
			t.Errorf("expected browser URL to contain verification URI, got %q", openedURL)
		}
	default:
		t.Error("OpenBrowser was not called")
	}
}

func TestDeviceClient_ClipboardCalled(t *testing.T) {
	var copiedText string
	clipCalled := make(chan struct{}, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/login/device/code"):
			json.NewEncoder(w).Encode(deviceBaseResp())
		case strings.HasSuffix(r.URL.Path, "/login/oauth/access_token"):
			json.NewEncoder(w).Encode(tokenSuccess("gho_clip_token"))
		}
	}))
	defer srv.Close()

	client := &DeviceClient{
		ClientID:   "test-id",
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		Out:        io.Discard,
		sleepFn:    func(d time.Duration) {},
		Clipboard: func(s string) error {
			copiedText = s
			clipCalled <- struct{}{}
			return nil
		},
	}

	token, err := client.RequestToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "gho_clip_token" {
		t.Errorf("expected gho_clip_token, got %q", token)
	}

	select {
	case <-clipCalled:
		if copiedText != "ABCD-1234" {
			t.Errorf("expected clipboard to receive 'ABCD-1234', got %q", copiedText)
		}
	default:
		t.Error("Clipboard was not called")
	}
}

func TestDeviceClient_ClipboardErrorNonFatal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/login/device/code"):
			json.NewEncoder(w).Encode(deviceBaseResp())
		case strings.HasSuffix(r.URL.Path, "/login/oauth/access_token"):
			json.NewEncoder(w).Encode(tokenSuccess("gho_cliperr_token"))
		}
	}))
	defer srv.Close()

	client := &DeviceClient{
		ClientID:   "test-id",
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		Out:        io.Discard,
		sleepFn:    func(d time.Duration) {},
		Clipboard: func(s string) error {
			return fmt.Errorf("clipboard unavailable")
		},
	}

	token, err := client.RequestToken()
	if err != nil {
		t.Fatalf("clipboard error should be non-fatal: %v", err)
	}
	if token != "gho_cliperr_token" {
		t.Errorf("expected gho_cliperr_token, got %q", token)
	}
}

// --- Edge Cases ---

func TestDeviceClient_UserFriendlyOutput(t *testing.T) {
	var buf strings.Builder

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/login/device/code"):
			json.NewEncoder(w).Encode(deviceBaseResp())
		case strings.HasSuffix(r.URL.Path, "/login/oauth/access_token"):
			json.NewEncoder(w).Encode(tokenSuccess("gho_out_token"))
		}
	}))
	defer srv.Close()

	client := &DeviceClient{
		ClientID:   "test-id",
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		Out:        &buf,
		sleepFn:    func(d time.Duration) {},
	}

	_, err := client.RequestToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "ABCD-1234") {
		t.Errorf("output should contain user code, got: %s", output)
	}
	if !strings.Contains(output, "github.com/login/device") {
		t.Errorf("output should contain verification URI, got: %s", output)
	}
}

func TestDeviceClient_DefaultsApplied(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/login/device/code"):
			json.NewEncoder(w).Encode(deviceBaseResp())
		case strings.HasSuffix(r.URL.Path, "/login/oauth/access_token"):
			json.NewEncoder(w).Encode(tokenSuccess("gho_defaults"))
		}
	}))
	defer srv.Close()

	// Zero-value client with only required fields (Out defaults to Discard).
	client := &DeviceClient{
		ClientID:   "test-id",
		HTTPClient: srv.Client(),
		BaseURL:    srv.URL,
		sleepFn:    func(d time.Duration) {},
	}

	token, err := client.RequestToken()
	if err != nil {
		t.Fatalf("zero-value defaults should work: %v", err)
	}
	if token != "gho_defaults" {
		t.Errorf("expected gho_defaults, got %q", token)
	}
}
