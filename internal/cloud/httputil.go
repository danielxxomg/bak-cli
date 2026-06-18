package cloud

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// newRequest builds an HTTP request with common headers.
// accept and contentType may be empty strings to omit those headers.
// When token is empty, the Authorization header is omitted (for
// endpoints that don't require authentication, e.g., OAuth Device Flow).
func newRequest(method, url, token, accept, contentType string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("User-Agent", "bak-cli")
	return req, nil
}

// doRequest executes an HTTP request and reads the response body.
// It returns the body bytes, HTTP status code, and any execution error.
// The caller is responsible for checking the status code.
func doRequest(client *http.Client, req *http.Request) (body []byte, status int, err error) {
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }() //nolint:errcheck // standard response body close in defer

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return body, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	return body, resp.StatusCode, nil
}

// formatAPIError formats an API error from a response body and status code.
// It attempts to parse JSON error responses (e.g., GitHub OAuth) and
// surfaces the error_description or error field. Falls back to the raw body
// or the HTTP status text.
func formatAPIError(body []byte, status int) error {
	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = http.StatusText(status)
		return fmt.Errorf("api error %d: %s", status, msg)
	}

	// Try to parse JSON error response.
	var apiErr struct {
		Error            string `json:"error"`
		ErrorDescription string `json:"error_description"`
		Message          string `json:"message"`
	}
	if err := json.Unmarshal(body, &apiErr); err == nil {
		if apiErr.ErrorDescription != "" {
			return fmt.Errorf("api error %d: %s", status, apiErr.ErrorDescription)
		}
		if apiErr.Error != "" {
			return fmt.Errorf("api error %d: %s", status, apiErr.Error)
		}
	}

	return fmt.Errorf("api error %d: %s", status, msg)
}
