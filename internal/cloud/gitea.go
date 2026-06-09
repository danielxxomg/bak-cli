package cloud

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/danielxxomg/bak-cli/internal/config"
)

const (
	defaultGiteaBaseURL = "https://codeberg.org"
	defaultBranch       = "main"
	giteaBackupDir      = "bak-backups"
	giteaTimeout        = 30 * time.Second
)

// GiteaProvider implements Provider using a Gitea/Forgejo repository as
// the storage backend. Backups are stored as files under the
// "bak-backups/" directory in the configured repository.
type GiteaProvider struct {
	cfg     *config.Config
	token   string
	baseURL string
	repo    string
	branch  string
	name    string // "gitea" or "codeberg" — used in error messages
	client  *http.Client
}

// NewGiteaProvider creates a provider for Gitea/Forgejo.
// Token resolution: uses the provided token if non-empty, otherwise
// falls back to GITEA_TOKEN env var or config providers.gitea.token.
// If baseURL is empty, defaults to "https://codeberg.org".
func NewGiteaProvider(cfg *config.Config, token, baseURL, repo string) *GiteaProvider {
	if token == "" {
		token = os.Getenv("GITEA_TOKEN")
	}
	if token == "" && cfg != nil {
		if t, err := cfg.Get("providers.gitea.token"); err == nil && t != "" {
			token = t
		}
	}
	if baseURL == "" {
		baseURL = defaultGiteaBaseURL
	}
	return &GiteaProvider{
		cfg:     cfg,
		token:   token,
		baseURL: baseURL,
		repo:    repo,
		branch:  defaultBranch,
		name:    "gitea",
		client:  &http.Client{Timeout: giteaTimeout},
	}
}

// CodebergProvider wraps GiteaProvider with a fixed Codeberg base URL.
type CodebergProvider struct {
	*GiteaProvider
}

// NewCodebergProvider creates a provider for Codeberg.org.
// Token resolution: uses the provided token if non-empty, otherwise
// falls back to CODEBERG_TOKEN env var or config providers.codeberg.token.
func NewCodebergProvider(cfg *config.Config, token, repo string) *CodebergProvider {
	if token == "" {
		token = os.Getenv("CODEBERG_TOKEN")
	}
	if token == "" && cfg != nil {
		if t, err := cfg.Get("providers.codeberg.token"); err == nil && t != "" {
			token = t
		}
	}
	gp := NewGiteaProvider(cfg, token, "https://codeberg.org", repo)
	gp.name = "codeberg"
	return &CodebergProvider{
		GiteaProvider: gp,
	}
}

// Name returns "codeberg".
func (p *CodebergProvider) Name() string {
	return "codeberg"
}

// Name returns the provider name.
func (p *GiteaProvider) Name() string {
	return p.name
}

// errf formats an error message with the provider name prefix.
func (p *GiteaProvider) errf(format string, args ...interface{}) error {
	return fmt.Errorf(p.name+": "+format, args...)
}

// Push uploads a backup archive to the Gitea repository.
// The archive is stored as bak-backups/{backupID}.tar.gz.
func (p *GiteaProvider) Push(archive []byte, meta PushMeta) (string, error) {
	if p.token == "" {
		return "", p.errf("push: token is required")
	}
	if p.repo == "" {
		return "", p.errf("push: repo is required")
	}

	filePath := fmt.Sprintf("%s/%s.tar.gz", giteaBackupDir, meta.BackupID)
	content := base64.StdEncoding.EncodeToString(archive)
	message := fmt.Sprintf("bak backup %s", meta.BackupID)

	// Check if the file already exists.
	sha, err := p.getFileSHA(filePath)
	if err != nil {
		return "", p.errf("push: %w", err)
	}

	if sha != "" {
		if err := p.putFile(filePath, content, message, sha); err != nil {
			return "", p.errf("push: %w", err)
		}
	} else {
		if err := p.postFile(filePath, content, message); err != nil {
			return "", p.errf("push: %w", err)
		}
	}

	return meta.BackupID, nil
}

// Pull downloads a backup archive from the Gitea repository by its backup ID.
func (p *GiteaProvider) Pull(id string) ([]byte, error) {
	if p.token == "" {
		return nil, p.errf("pull: token is required")
	}
	if id == "" {
		return nil, p.errf("pull: backup ID is required")
	}
	if p.repo == "" {
		return nil, p.errf("pull: repo is required")
	}

	filePath := fmt.Sprintf("%s/%s.tar.gz", giteaBackupDir, id)
	url := fmt.Sprintf("%s/api/v1/repos/%s/contents/%s", p.baseURL, p.repo, filePath)

	req, err := newRequest(http.MethodGet, url, p.token, "application/json", "", nil)
	if err != nil {
		return nil, p.errf("pull: build request: %w", err)
	}

	body, status, err := doRequest(p.client, req)
	if err != nil {
		return nil, p.errf("pull: %w", err)
	}

	if status < 200 || status >= 300 {
		return nil, p.errf("pull: %w", formatAPIError(body, status))
	}

	var cr contentResponse
	if err := json.Unmarshal(body, &cr); err != nil {
		return nil, p.errf("pull: parse response: %w", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(cr.Content.Content)
	if err != nil {
		return nil, p.errf("pull: decode content: %w", err)
	}

	return decoded, nil
}

// List returns metadata for all bak backups stored in the Gitea repo.
func (p *GiteaProvider) List() ([]BackupMeta, error) {
	if p.token == "" {
		return nil, p.errf("list: token is required")
	}
	if p.repo == "" {
		return nil, p.errf("list: repo is required")
	}

	url := fmt.Sprintf("%s/api/v1/repos/%s/contents/%s", p.baseURL, p.repo, giteaBackupDir)

	req, err := newRequest(http.MethodGet, url, p.token, "application/json", "", nil)
	if err != nil {
		return nil, p.errf("list: build request: %w", err)
	}

	body, status, err := doRequest(p.client, req)
	if err != nil {
		return nil, p.errf("list: %w", err)
	}

	// Directory listing may return 404 if the directory doesn't exist yet.
	if status == http.StatusNotFound {
		return nil, nil
	}

	if status < 200 || status >= 300 {
		return nil, p.errf("list: %w", formatAPIError(body, status))
	}

	var items []contentResponse
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, p.errf("list: parse response: %w", err)
	}

	metas := make([]BackupMeta, 0, len(items))
	for _, item := range items {
		// Extract backup ID from filename: "YYYYMMDD-HHMMSS.tar.gz".
		backupID := strings.TrimSuffix(item.Name, ".tar.gz")

		metas = append(metas, BackupMeta{
			ID:       backupID,
			BackupID: backupID,
			Size:     item.Size,
			URL:      fmt.Sprintf("%s/%s/src/branch/%s/%s/%s", p.baseURL, p.repo, p.branch, giteaBackupDir, item.Name),
		})
	}

	return metas, nil
}

// getFileSHA checks if a file exists by fetching its metadata from the Gitea API.
// Returns the file SHA if it exists, or empty string if not found.
func (p *GiteaProvider) getFileSHA(filePath string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/repos/%s/contents/%s", p.baseURL, p.repo, filePath)
	return getFileSHA(p.client, p.token, url)
}

// postFile creates a new file in the Gitea repository.
func (p *GiteaProvider) postFile(filePath, content, message string) error {
	return p.writeFile(http.MethodPost, filePath, content, message, "")
}

// putFile updates an existing file in the Gitea repository.
func (p *GiteaProvider) putFile(filePath, content, message, sha string) error {
	return p.writeFile(http.MethodPut, filePath, content, message, sha)
}

// writeFile creates or updates a file in the Gitea repository via the
// contents API.
func (p *GiteaProvider) writeFile(method, filePath, content, message, sha string) error {
	url := fmt.Sprintf("%s/api/v1/repos/%s/contents/%s", p.baseURL, p.repo, filePath)
	req := contentRequest{
		Content: content,
		Message: message,
		Branch:  p.branch,
		SHA:     sha,
	}
	return writeContentFile(p.client, p.token, method, "application/json", url, req)
}
