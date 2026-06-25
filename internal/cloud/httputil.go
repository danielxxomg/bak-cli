package cloud

import (
	"encoding/base64"
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

// pullContentFromAPI validates the pull preconditions, fetches a single file
// from a Contents API endpoint, and returns its decoded (base64) content. It
// validates that token, id, and repo are non-empty before issuing the request,
// then performs the GET, checks the HTTP status, unmarshals the
// contentResponse, and decodes the content. errPrefix is prepended to every
// returned error (e.g., "gitea: pull", "pull github-repo"). The caller builds
// url because the Contents API path differs between providers.
func pullContentFromAPI(client *http.Client, token, repo, id, url, accept, errPrefix string) ([]byte, error) {
	wrap := func(format string, args ...any) error {
		return fmt.Errorf(errPrefix+": "+format, args...)
	}

	if token == "" {
		return nil, wrap("token is required")
	}
	if id == "" {
		return nil, wrap("backup ID is required")
	}
	if repo == "" {
		return nil, wrap("repo is required")
	}

	req, err := newRequest(http.MethodGet, url, token, accept, "", nil)
	if err != nil {
		return nil, wrap("build request: %w", err)
	}

	body, status, err := doRequest(client, req)
	if err != nil {
		return nil, wrap("%w", err)
	}

	if status < 200 || status >= 300 {
		return nil, wrap("%w", formatAPIError(body, status))
	}

	var cr contentResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return nil, wrap("parse response: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(cr.Content.Content)
	if err != nil {
		return nil, wrap("decode content: %w", err)
	}

	return decoded, nil
}

// listContentsDir fetches a Contents API directory listing and maps the
// entries to BackupMeta. It is the shared backing for GiteaProvider.List and
// GitHubRepoProvider.List, parameterized by the list URL, accept header,
// provider error prefix (errPrefix), and a urlBuilder that maps each
// contentResponse item to the provider-specific human-readable backup URL.
//
// The caller retains the token/repo guards; this helper only performs the
// HTTP fetch and mapping. A 404 response means "directory does not exist yet"
// and is reported as an empty slice with a nil error. Any other non-2xx
// status is returned as an error wrapped with errPrefix.
func listContentsDir(
	client *http.Client,
	url, token, accept, errPrefix string,
	urlBuilder func(item contentResponse) string,
) ([]BackupMeta, error) {
	wrap := func(format string, args ...any) error {
		return fmt.Errorf(errPrefix+": "+format, args...)
	}

	req, err := newRequest(http.MethodGet, url, token, accept, "", nil)
	if err != nil {
		return nil, wrap("build request: %w", err)
	}

	body, status, err := doRequest(client, req)
	if err != nil {
		return nil, wrap("%w", err)
	}

	// Directory listing may return 404 if the directory doesn't exist yet.
	if status == http.StatusNotFound {
		return nil, nil
	}

	if status < 200 || status >= 300 {
		return nil, wrap("%w", formatAPIError(body, status))
	}

	var items []contentResponse
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, wrap("parse response: %w", err)
	}

	metas := make([]BackupMeta, 0, len(items))
	for _, item := range items {
		backupID := strings.TrimSuffix(item.Name, ".tar.gz")
		metas = append(metas, BackupMeta{
			ID:       backupID,
			BackupID: backupID,
			Size:     item.Size,
			URL:      urlBuilder(item),
		})
	}
	return metas, nil
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
