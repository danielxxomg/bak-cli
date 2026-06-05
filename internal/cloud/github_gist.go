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

// GitHubGistProvider implements Provider using GitHub Gist as the
// storage backend. It wraps the low-level gist.go API functions.
type GitHubGistProvider struct {
	cfg   *config.Config
	token string
}

// NewGitHubGistProvider creates a provider for github-gist.
// Token resolution: uses the provided token if non-empty, otherwise
// falls back to GITHUB_TOKEN env var or config providers.github.token.
func NewGitHubGistProvider(cfg *config.Config, token string) *GitHubGistProvider {
	if token == "" {
		token = os.Getenv("GITHUB_TOKEN")
	}
	if token == "" && cfg != nil {
		if t, err := cfg.Get("github.token"); err == nil && t != "" {
			token = t
		}
	}
	return &GitHubGistProvider{
		cfg:   cfg,
		token: token,
	}
}

// Name returns "github-gist".
func (p *GitHubGistProvider) Name() string {
	return "github-gist"
}

// Push uploads a backup archive to a GitHub Gist.
// If a gist_id is configured, it updates the existing gist;
// otherwise it creates a new private gist.
func (p *GitHubGistProvider) Push(archive []byte, meta PushMeta) (string, error) {
	if p.token == "" {
		return "", fmt.Errorf("push gist: token is required")
	}

	content := base64.StdEncoding.EncodeToString(archive)
	desc := fmt.Sprintf("bak backup %s — %s", meta.BackupID, time.Now().UTC().Format(time.RFC3339))

	files := []GistFile{
		{Filename: "backup.tar.gz", Content: content},
	}

	// Check for existing gist ID.
	gistID := ""
	if p.cfg != nil {
		if id, err := p.cfg.Get("github.gist_id"); err == nil && id != "" {
			gistID = id
		}
	}

	if gistID != "" {
		if err := UpdateGist(p.token, gistID, desc, files); err != nil {
			return "", fmt.Errorf("push gist: %w", err)
		}
		return gistID, nil
	}

	id, err := CreateGist(p.token, desc, files)
	if err != nil {
		return "", fmt.Errorf("push gist: %w", err)
	}

	// Save gist ID to config for future updates.
	if p.cfg != nil {
		if err := p.cfg.Set("github.gist_id", id); err != nil {
			fmt.Fprintf(os.Stderr, "warning: save gist id: %v\n", err)
		}
	}

	return id, nil
}

// Pull downloads a backup archive from a GitHub Gist by its gist ID.
func (p *GitHubGistProvider) Pull(id string) ([]byte, error) {
	if p.token == "" {
		return nil, fmt.Errorf("pull gist: token is required")
	}
	if id == "" {
		return nil, fmt.Errorf("pull gist: gist ID is required")
	}

	files, err := GetGist(p.token, id)
	if err != nil {
		return nil, fmt.Errorf("pull gist: %w", err)
	}

	for _, f := range files {
		if f.Filename == "backup.tar.gz" {
			return []byte(f.Content), nil
		}
	}

	return nil, fmt.Errorf("pull gist: no backup.tar.gz found in gist %s", id)
}

// List returns metadata for all bak backup gists owned by the user.
func (p *GitHubGistProvider) List() ([]BackupMeta, error) {
	if p.token == "" {
		return nil, fmt.Errorf("list gists: token is required")
	}

	data, err := gistAPI(p.token, http.MethodGet, GistAPIBase+"/gists", nil)
	if err != nil {
		return nil, fmt.Errorf("list gists: %w", err)
	}

	var gists []gistResponse
	if err := json.Unmarshal(data, &gists); err != nil {
		return nil, fmt.Errorf("list gists: parse response: %w", err)
	}

	metas := make([]BackupMeta, 0, len(gists))
	for _, g := range gists {
		var size int64
		for _, f := range g.Files {
			size += int64(len(f.Content))
		}

		// Extract backup ID from description "bak backup YYYYMMDD-HHMMSS"
		backupID := ""
		if strings.HasPrefix(g.Description, "bak backup ") && len(g.Description) >= 27 {
			backupID = g.Description[11:26] // YYYYMMDD-HHMMSS
		}

		metas = append(metas, BackupMeta{
			ID:        g.ID,
			BackupID:  backupID,
			Hostname:  "",
			Size:      size,
			URL:       g.HTMLURL,
		})
	}

	return metas, nil
}
