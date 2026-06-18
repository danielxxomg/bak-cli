package cloud

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// DeviceLoginBase is the base URL for GitHub's OAuth Device Flow endpoints.
// Overridable for tests (e.g., set to httptest.Server.URL).
var DeviceLoginBase = "https://github.com"

// DeviceClient performs GitHub OAuth Device Flow (RFC 8628) to obtain an
// access token without requiring a web server on the client side.
//
// All fields are optional with sensible defaults:
//   - HTTPClient: defaults to http.DefaultClient
//   - BaseURL:    defaults to DeviceLoginBase ("https://github.com")
//   - Out:        defaults to io.Discard (no output)
//   - OpenBrowser: when nil, OpenBrowser is never called
//   - Clipboard:   when nil, clipboard copy is skipped
type DeviceClient struct {
	// ClientID is the OAuth App client ID.
	ClientID string

	// HTTPClient is the HTTP client for API calls.
	HTTPClient *http.Client

	// BaseURL overrides the GitHub base URL for testing.
	BaseURL string

	// Out receives human-readable progress messages.
	Out io.Writer

	// OpenBrowser is called with the verification URI to open the
	// user's browser. When nil, the browser-open step is skipped.
	OpenBrowser func(url string) error

	// Clipboard is called with the user code to copy it to the
	// system clipboard. When nil, clipboard copy is skipped.
	Clipboard func(code string) error

	// sleepFn is the sleep function for polling intervals.
	// Defaults to time.Sleep; overridable for tests.
	sleepFn func(time.Duration)
}

// deviceCodeResponse is the JSON returned by POST /login/device/code.
type deviceCodeResponse struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// tokenPollResponse is the JSON returned by POST /login/oauth/access_token.
type tokenPollResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error,omitempty"`
	ErrorDesc   string `json:"error_description,omitempty"`
}

// RequestToken performs the full RFC 8628 Device Flow:
//  1. POST /login/device/code to obtain device_code + user_code
//  2. Display user_code and open browser to verification_uri
//  3. Poll POST /login/oauth/access_token until the user authorizes
//     or the code expires
func (c *DeviceClient) RequestToken() (string, error) {
	// Apply defaults.
	baseURL := c.BaseURL
	if baseURL == "" {
		baseURL = DeviceLoginBase
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	out := c.Out
	if out == nil {
		out = io.Discard
	}

	// 1. Request device code.
	deviceResp, err := requestDeviceCode(httpClient, baseURL, c.ClientID)
	if err != nil {
		return "", fmt.Errorf("device code: %w", err)
	}

	// 2. Show user code and verification URI.
	_, _ = fmt.Fprintf(out, "\nOpen this URL in your browser: %s\n", deviceResp.VerificationURI)
	_, _ = fmt.Fprintf(out, "Enter code: %s\n\n", deviceResp.UserCode)

	// Copy user code to clipboard (best-effort).
	if c.Clipboard != nil {
		if err := c.Clipboard(deviceResp.UserCode); err != nil {
			_, _ = fmt.Fprintf(out, "Could not copy code to clipboard: %v\n", err)
		}
	}

	// Open browser to verification URI (best-effort).
	if c.OpenBrowser != nil {
		if err := c.OpenBrowser(deviceResp.VerificationURI); err != nil {
			_, _ = fmt.Fprintf(out, "Could not open browser: %v\n", err)
		}
	}

	// 3. Poll for access token.
	interval := deviceResp.Interval
	if interval < 1 {
		interval = 5 // default per RFC 8628
	}
	deadline := time.Now().Add(time.Duration(deviceResp.ExpiresIn) * time.Second)

	for {
		if time.Now().After(deadline) {
			return "", fmt.Errorf("login: timed out waiting for authorization")
		}

		resp, err := pollAccessToken(httpClient, baseURL, c.ClientID, deviceResp.DeviceCode)
		if err != nil {
			return "", err
		}

		switch resp.Error {
		case "":
			// Success — access token received.
			if resp.AccessToken == "" {
				// Empty response with no error — treat as pending.
				_, _ = fmt.Fprint(out, "Waiting for authorization...\n")
			} else {
				return resp.AccessToken, nil
			}

		case "authorization_pending":
			// Continue polling at the current interval.
			_, _ = fmt.Fprint(out, "Waiting for authorization...\n")

		case "slow_down":
			// Server requests slower polling — increase interval.
			interval += 5
			_, _ = fmt.Fprintf(out, "Slow down requested — polling every %ds\n", interval)

		case "expired_token":
			return "", fmt.Errorf("login: code expired; run 'bak login' again")

		case "access_denied":
			return "", fmt.Errorf("login: authorization denied")

		default:
			_, _ = fmt.Fprintf(out, "Unexpected response: %s\n", resp.Error)
		}

		sleepFn := c.sleepFn
		if sleepFn == nil {
			sleepFn = time.Sleep
		}
		sleepFn(time.Duration(interval) * time.Second)
	}
}

// requestDeviceCode POSTs to /login/device/code to start the flow.
func requestDeviceCode(httpClient *http.Client, baseURL, clientID string) (*deviceCodeResponse, error) {
	if clientID == "" {
		return nil, fmt.Errorf("request device code: client_id is required")
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", "gist")

	var dr deviceCodeResponse
	if err := postOAuthForm(httpClient, baseURL, "/login/device/code", form, &dr); err != nil {
		return nil, fmt.Errorf("device code: %w", err)
	}

	// Check for API-level error in the JSON body (re-read needed since
	// postOAuthForm already consumed the body — but error responses
	// won't unmarshal into deviceCodeResponse, so we check for empty fields).
	if dr.DeviceCode == "" {
		return nil, fmt.Errorf("device code: empty device_code in response")
	}

	return &dr, nil
}

// pollAccessToken POSTs to /login/oauth/access_token to check for completion.
func pollAccessToken(httpClient *http.Client, baseURL, clientID, deviceCode string) (*tokenPollResponse, error) {
	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("device_code", deviceCode)
	form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

	var tr tokenPollResponse
	if err := postOAuthForm(httpClient, baseURL, "/login/oauth/access_token", form, &tr); err != nil {
		return nil, fmt.Errorf("poll token: %w", err)
	}

	return &tr, nil
}

// postOAuthForm POSTs form-encoded data to an OAuth endpoint and unmarshals
// the JSON response into target. Returns an error on transport, status, or
// unmarshal failure.
func postOAuthForm(httpClient *http.Client, baseURL, path string, form url.Values, target any) error {
	urlStr := strings.TrimRight(baseURL, "/") + path
	req, err := newRequest(http.MethodPost, urlStr, "",
		"application/json", "application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	body, status, err := doRequest(httpClient, req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}

	if status < 200 || status >= 300 {
		return formatAPIError(body, status)
	}

	if err := json.Unmarshal(body, target); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	return nil
}
