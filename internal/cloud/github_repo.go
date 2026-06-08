package cloud

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/danielxxomg/bak-cli/internal/config"
)

const (
	githubAPIDefault = "https://api.github.com"
	githubRepoDir    = "bak-backups"
	githubTimeout    = 30 * time.Second
)

// GitHubRepoProvider implements Provider using a private GitHub
// repository as the storage backend. Backups are stored as files
// under the "bak-backups/" directory in the configured repository
// via the GitHub Contents API.
type GitHubRepoProvider struct {
	cfg     *config.Config
	token   string
	repo    string
	branch  string
	client  *http.Client
	apiBase string
}

// NewGitHubRepoProvider creates a provider for github-repo.
// Token resolution: uses the provided token if non-empty, otherwise
// falls back to GITHUB_TOKEN env var or config providers.github-repo.token.
func NewGitHubRepoProvider(cfg *config.Config, token, repo string) *GitHubRepoProvider {
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" && cfg != nil {
		if t, err := cfg.Get("providers.github-repo.token"); err == nil && t != "" {
			token = t
		}
	}
	return &GitHubRepoProvider{
		cfg:     cfg,
		token:   token,
		repo:    repo,
		branch:  defaultBranch,
		client:  &http.Client{Timeout: githubTimeout},
		apiBase: githubAPIDefault,
	}
}

// Name returns "github-repo".
func (p *GitHubRepoProvider) Name() string {
	return "github-repo"
}

// Push uploads a backup archive to the GitHub repository.
// The archive is stored as bak-backups/{backupID}.tar.gz via the
// GitHub Contents API (PUT /repos/{owner}/{repo}/contents/{path}).
func (p *GitHubRepoProvider) Push(archive []byte, meta PushMeta) (string, error) {
	if p.token == "" {
		return "", fmt.Errorf("push github-repo: token is required")
	}
	if p.repo == "" {
		return "", fmt.Errorf("push github-repo: repo is required")
	}

	filePath := fmt.Sprintf("%s/%s.tar.gz", githubRepoDir, meta.BackupID)
	content := base64.StdEncoding.EncodeToString(archive)
	message := fmt.Sprintf("bak backup %s", meta.BackupID)

	// Check if the file already exists.
	sha, err := p.getFileSHA(filePath)
	if err != nil {
		return "", fmt.Errorf("push github-repo: %w", err)
	}

	// GitHub uses PUT for both create and update.
	if err := p.putFile(filePath, content, message, sha); err != nil {
		return "", fmt.Errorf("push github-repo: %w", err)
	}

	return meta.BackupID, nil
}

// Pull downloads a backup archive from the GitHub repository by its
// backup ID via the GitHub Contents API.
func (p *GitHubRepoProvider) Pull(id string) ([]byte, error) {
	if p.token == "" {
		return nil, fmt.Errorf("pull github-repo: token is required")
	}
	if id == "" {
		return nil, fmt.Errorf("pull github-repo: backup ID is required")
	}
	if p.repo == "" {
		return nil, fmt.Errorf("pull github-repo: repo is required")
	}

	filePath := fmt.Sprintf("%s/%s.tar.gz", githubRepoDir, id)
	url := fmt.Sprintf("%s/repos/%s/contents/%s", p.apiBase, p.repo, filePath)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("pull github-repo: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "bak-cli")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("pull github-repo: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("pull github-repo: read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return nil, fmt.Errorf("pull github-repo: API error %d: %s", resp.StatusCode, msg)
	}

	var cr githubContentResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return nil, fmt.Errorf("pull github-repo: parse response: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(cr.Content.Content)
	if err != nil {
		return nil, fmt.Errorf("pull github-repo: decode content: %w", err)
	}

	return decoded, nil
}

// List returns metadata for all bak backups stored in the GitHub repo.
func (p *GitHubRepoProvider) List() ([]BackupMeta, error) {
	if p.token == "" {
		return nil, fmt.Errorf("list github-repo: token is required")
	}
	if p.repo == "" {
		return nil, fmt.Errorf("list github-repo: repo is required")
	}

	url := fmt.Sprintf("%s/repos/%s/contents/%s", p.apiBase, p.repo, githubRepoDir)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("list github-repo: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "bak-cli")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list github-repo: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("list github-repo: read response: %w", err)
	}

	// Directory listing returns 404 if the directory doesn't exist yet.
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return nil, fmt.Errorf("list github-repo: API error %d: %s", resp.StatusCode, msg)
	}

	var items []githubContentResponse
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, fmt.Errorf("list github-repo: parse response: %w", err)
	}

	metas := make([]BackupMeta, 0, len(items))
	for _, item := range items {
		backupID := strings.TrimSuffix(item.Name, ".tar.gz")

		metas = append(metas, BackupMeta{
			ID:       backupID,
			BackupID: backupID,
			Size:     item.Size,
			URL:      fmt.Sprintf("https://github.com/%s/blob/%s/%s/%s", p.repo, p.branch, githubRepoDir, item.Name),
		})
	}

	return metas, nil
}

// getFileSHA checks if a file exists by fetching its metadata from the
// GitHub API. Returns the file SHA if it exists, or empty string if not found.
func (p *GitHubRepoProvider) getFileSHA(filePath string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/contents/%s", p.apiBase, p.repo, filePath)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("check file: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "bak-cli")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("check file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return "", nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("check file: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return "", fmt.Errorf("check file: api error %d: %s", resp.StatusCode, msg)
	}

	var cr githubContentResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	return cr.Content.SHA, nil
}

// putFile creates or updates a file in the GitHub repository via the
// Contents API. GitHub uses PUT for both create (without SHA) and
// update (with SHA).
func (p *GitHubRepoProvider) putFile(filePath, content, message, sha string) error {
	url := fmt.Sprintf("%s/repos/%s/contents/%s", p.apiBase, p.repo, filePath)

	reqBody := githubContentRequest{
		Message: message,
		Content: content,
		Branch:  p.branch,
		SHA:     sha,
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest(http.MethodPut, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "bak-cli")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(respBody))
		if msg == "" {
			msg = http.StatusText(resp.StatusCode)
		}
		return fmt.Errorf("write file: api error %d: %s", resp.StatusCode, msg)
	}

	return nil
}

// githubContentRequest is the JSON body for the GitHub Contents API
// PUT /repos/{owner}/{repo}/contents/{path}.
type githubContentRequest struct {
	Message string `json:"message"`
	Content string `json:"content"`
	Branch  string `json:"branch"`
	SHA     string `json:"sha,omitempty"`
}

// githubContentResponse is the JSON response from the GitHub Contents API.
type githubContentResponse struct {
	Name    string            `json:"name"`
	Path    string            `json:"path"`
	SHA     string            `json:"sha"`
	Size    int64             `json:"size"`
	Content githubContentFile `json:"content"`
}

// githubContentFile holds the file metadata and content from a
// GitHub Contents API response.
type githubContentFile struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	SHA      string `json:"sha"`
	Size     int64  `json:"size"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
}
