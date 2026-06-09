package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// contentRequest is the JSON body for repository Contents API
// create/update operations (PUT/POST /repos/{owner}/{repo}/contents/{path}).
// Used by GitHubRepoProvider and GiteaProvider.
type contentRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
	Branch  string `json:"branch"`
	SHA     string `json:"sha,omitempty"`
}

// contentResponse is the JSON response from a Contents API
// single-file endpoint. Used by GitHubRepoProvider and GiteaProvider.
type contentResponse struct {
	Name    string      `json:"name"`
	Path    string      `json:"path"`
	SHA     string      `json:"sha"`
	Size    int64       `json:"size"`
	Content contentFile `json:"content"`
}

// contentFile holds the file metadata and decoded/encoded content
// from a Contents API response (nested inside contentResponse).
type contentFile struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	SHA      string `json:"sha"`
	Size     int64  `json:"size"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
}

// getFileSHA fetches file metadata from a Contents API endpoint
// and returns the file's SHA. Returns empty string and nil error
// if the file does not exist (HTTP 404). For any other non-2xx
// status, returns a descriptive error.
func getFileSHA(client *http.Client, token, url string) (string, error) {
	req, err := newRequest(http.MethodGet, url, token, "application/json", "", nil)
	if err != nil {
		return "", fmt.Errorf("check file: %w", err)
	}

	body, status, err := doRequest(client, req)
	if err != nil {
		return "", fmt.Errorf("check file: %w", err)
	}

	if status == http.StatusNotFound {
		return "", nil
	}

	if status < 200 || status >= 300 {
		return "", fmt.Errorf("check file: %w", formatAPIError(body, status))
	}

	var cr contentResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	return cr.Content.SHA, nil
}

// writeContentFile sends a create/update request to a Contents API
// endpoint. It marshals the contentRequest, sends it via newRequest
// and doRequest, and returns an error on failure.
func writeContentFile(client *http.Client, token, method, accept, url string, req contentRequest) error {
	bodyBytes, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := newRequest(method, url, token, accept, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}

	body, status, err := doRequest(client, httpReq)
	if err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	if status < 200 || status >= 300 {
		return fmt.Errorf("write file: %w", formatAPIError(body, status))
	}

	return nil
}
