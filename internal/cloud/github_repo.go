package cloud

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
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
		token = os.Getenv(githubTokenEnv)
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
	return providerGithubRepo
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
	filePath := fmt.Sprintf("%s/%s.tar.gz", githubRepoDir, id)
	url := fmt.Sprintf("%s/repos/%s/contents/%s", p.apiBase, p.repo, filePath)

	return pullContentFromAPI(p.client, p.token, p.repo, id, url, acceptGitHub, "pull github-repo")
}

// List returns metadata for all bak backups stored in the GitHub repo.
// It delegates the HTTP listing logic to the shared listContentsDir helper,
// parameterizing the GitHub Contents API URL, accept header, error prefix,
// and per-item blob URL. Token and repo guards stay here.
func (p *GitHubRepoProvider) List() ([]BackupMeta, error) {
	if p.token == "" {
		return nil, fmt.Errorf("list github-repo: token is required")
	}
	if p.repo == "" {
		return nil, fmt.Errorf("list github-repo: repo is required")
	}

	url := fmt.Sprintf("%s/repos/%s/contents/%s", p.apiBase, p.repo, githubRepoDir)
	return listContentsDir(p.client, url, p.token, acceptGitHub, "list github-repo",
		func(item contentResponse) string {
			return fmt.Sprintf("https://github.com/%s/blob/%s/%s/%s", p.repo, p.branch, githubRepoDir, item.Name)
		})
}

// getFileSHA checks if a file exists by fetching its metadata from the
// GitHub API. Returns the file SHA if it exists, or empty string if not found.
func (p *GitHubRepoProvider) getFileSHA(filePath string) (string, error) {
	url := fmt.Sprintf("%s/repos/%s/contents/%s", p.apiBase, p.repo, filePath)
	return getFileSHA(p.client, p.token, url)
}

// putFile creates or updates a file in the GitHub repository via the
// Contents API. GitHub uses PUT for both create (without SHA) and
// update (with SHA).
func (p *GitHubRepoProvider) putFile(filePath, content, message, sha string) error {
	url := fmt.Sprintf("%s/repos/%s/contents/%s", p.apiBase, p.repo, filePath)
	req := contentRequest{
		Message: message,
		Content: content,
		Branch:  p.branch,
		SHA:     sha,
	}
	return writeContentFile(p.client, p.token, http.MethodPut, acceptGitHub, url, req)
}
