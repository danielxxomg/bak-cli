// Package cloud implements the GitHub Gist sync client and token
// management for bak's cloud backup feature.
//
// The Gist client uses net/http and encoding/json (no external
// dependencies) to interact with the GitHub Gist API v3. All gists
// are created as private.
package cloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GistFile represents a single file inside a Gist.
type GistFile struct {
	Filename string // display name (e.g. "backup.tar.gz")
	Content  string // raw text content
}

// Gist represents a GitHub Gist as returned by the API.
type Gist struct {
	ID          string     `json:"id"`
	Description string     `json:"description"`
	Public      bool       `json:"public"`
	HTMLURL     string     `json:"html_url"`
	Files       []GistFile `json:"-"`
}

// gistFileAPI is the JSON shape GitHub expects/returns for a file.
type gistFileAPI struct {
	Content string `json:"content"`
}

// gistCreateRequest is the JSON body for POST /gists.
type gistCreateRequest struct {
	Description string                  `json:"description"`
	Public      bool                    `json:"public"`
	Files       map[string]gistFileAPI  `json:"files"`
}

// gistUpdateRequest is the JSON body for PATCH /gists/{id}.
type gistUpdateRequest struct {
	Description string                  `json:"description,omitempty"`
	Files       map[string]gistFileAPI  `json:"files,omitempty"`
}

// gistResponse is the JSON shape returned by the GitHub Gist API.
type gistResponse struct {
	ID          string                  `json:"id"`
	Description string                  `json:"description"`
	Public      bool                    `json:"public"`
	HTMLURL     string                  `json:"html_url"`
	Files       map[string]gistFileAPI  `json:"files"`
}

// defaultTimeout is the HTTP client timeout for Gist API calls.
const defaultTimeout = 30 * time.Second

// GistAPIBase is the base URL for the GitHub Gist API. Exported for
// test override.
var GistAPIBase = "https://api.github.com"

// httpClient is the shared HTTP client used by all Gist operations.
var httpClient = &http.Client{Timeout: defaultTimeout}

// CreateGist creates a new private GitHub Gist containing the given
// files. Returns the gist ID on success.
func CreateGist(token, description string, files []GistFile) (string, error) {
	if token == "" {
		return "", fmt.Errorf("create gist: token is required")
	}
	if len(files) == 0 {
		return "", fmt.Errorf("create gist: at least one file is required")
	}

	filesMap := make(map[string]gistFileAPI, len(files))
	for _, f := range files {
		filesMap[f.Filename] = gistFileAPI{Content: f.Content}
	}

	body := gistCreateRequest{
		Description: description,
		Public:      false, // force private
		Files:       filesMap,
	}

	resp, err := gistAPI(token, http.MethodPost, GistAPIBase+"/gists", body)
	if err != nil {
		return "", fmt.Errorf("create gist: %w", err)
	}

	var gist gistResponse
	if err := json.Unmarshal(resp, &gist); err != nil {
		return "", fmt.Errorf("create gist: parse response: %w", err)
	}

	return gist.ID, nil
}

// UpdateGist replaces the contents of an existing Gist with the given
// files. Returns an error if the gist does not exist or the token is
// invalid.
func UpdateGist(token, gistID, description string, files []GistFile) error {
	if token == "" {
		return fmt.Errorf("update gist: token is required")
	}
	if gistID == "" {
		return fmt.Errorf("update gist: gist ID is required")
	}

	filesMap := make(map[string]gistFileAPI, len(files))
	for _, f := range files {
		filesMap[f.Filename] = gistFileAPI{Content: f.Content}
	}

	body := gistUpdateRequest{
		Description: description,
		Files:       filesMap,
	}

	url := GistAPIBase + "/gists/" + gistID
	_, err := gistAPI(token, http.MethodPatch, url, body)
	if err != nil {
		return fmt.Errorf("update gist %q: %w", gistID, err)
	}

	return nil
}

// GetGist fetches a Gist by its ID and returns the files contained
// within it.
func GetGist(token, gistID string) ([]GistFile, error) {
	if token == "" {
		return nil, fmt.Errorf("get gist: token is required")
	}
	if gistID == "" {
		return nil, fmt.Errorf("get gist: gist ID is required")
	}

	url := GistAPIBase + "/gists/" + gistID
	resp, err := gistAPI(token, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("get gist %q: %w", gistID, err)
	}

	var gist gistResponse
	if err := json.Unmarshal(resp, &gist); err != nil {
		return nil, fmt.Errorf("get gist: parse response: %w", err)
	}

	files := make([]GistFile, 0, len(gist.Files))
	for name, f := range gist.Files {
		files = append(files, GistFile{
			Filename: name,
			Content:  f.Content,
		})
	}

	return files, nil
}

// DeleteGist permanently removes a Gist by its ID.
func DeleteGist(token, gistID string) error {
	if token == "" {
		return fmt.Errorf("delete gist: token is required")
	}
	if gistID == "" {
		return fmt.Errorf("delete gist: gist ID is required")
	}

	url := GistAPIBase + "/gists/" + gistID
	_, err := gistAPI(token, http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("delete gist %q: %w", gistID, err)
	}

	return nil
}

// apiError represents a GitHub API error response.
type apiError struct {
	StatusCode int
	Message    string
	Body       string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("GitHub API error %d: %s", e.StatusCode, e.Message)
}

// gistAPI performs an authenticated request against the GitHub Gist API.
func gistAPI(token, method, url string, body interface{}) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "bak-cli")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respData))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return nil, &apiError{
			StatusCode: resp.StatusCode,
			Message:    msg,
			Body:       string(respData),
		}
	}

	return respData, nil
}
