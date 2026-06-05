package cloud

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/danielxxomg/bak-cli/internal/config"
)

func TestGitHubGistProvider_Name(t *testing.T) {
	p := NewGitHubGistProvider(nil, "test-token")
	if p.Name() != "github-gist" {
		t.Errorf("Name() = %q, want github-gist", p.Name())
	}
}

func TestGitHubGistProvider_Push_Create(t *testing.T) {
	_, cleanup := setupMockGistAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/gists" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(gistResponse{
			ID:      "gist-create-1",
			HTMLURL: "https://gist.github.com/gist-create-1",
		})
	})
	defer cleanup()

	cfg := &config.Config{}
	_ = cfg.Set("github.gist_id", "") // No existing gist ID — forces create.

	p := NewGitHubGistProvider(cfg, "valid-token")
	id, err := p.Push([]byte("test-archive-data"), PushMeta{
		BackupID:  "20260605-120000",
		CreatedAt: time.Now(),
		Hostname:  "testbox",
	})
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if id != "gist-create-1" {
		t.Errorf("Push id = %q, want gist-create-1", id)
	}
}

func TestGitHubGistProvider_Push_Update(t *testing.T) {
	_, cleanup := setupMockGistAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/gists/") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(gistResponse{ID: "existing-gist"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	cfg := &config.Config{}
	_ = cfg.Set("github.gist_id", "existing-gist")

	p := NewGitHubGistProvider(cfg, "valid-token")
	id, err := p.Push([]byte("updated-archive"), PushMeta{
		BackupID: "20260605-130000",
	})
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if id != "existing-gist" {
		t.Errorf("Push id = %q, want existing-gist", id)
	}
}

func TestGitHubGistProvider_Push_NoToken(t *testing.T) {
	p := NewGitHubGistProvider(nil, "") // empty token
	_, err := p.Push([]byte("data"), PushMeta{BackupID: "test"})
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if !strings.Contains(err.Error(), "token is required") {
		t.Errorf("error = %v, want mention of token", err)
	}
}

func TestGitHubGistProvider_Pull(t *testing.T) {
	_, cleanup := setupMockGistAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/gists/") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(gistResponse{
				ID: "pull-gist",
				Files: map[string]gistFileAPI{
					"backup.tar.gz": {Content: "YmFja3VwLWRhdGE="},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	p := NewGitHubGistProvider(nil, "valid-token")
	data, err := p.Pull("pull-gist")
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if string(data) != "YmFja3VwLWRhdGE=" {
		t.Errorf("Pull data = %q, want YmFja3VwLWRhdGE=", string(data))
	}
}

func TestGitHubGistProvider_Pull_NoBackupFile(t *testing.T) {
	_, cleanup := setupMockGistAPI(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/gists/") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(gistResponse{
				ID:    "no-backup-gist",
				Files: map[string]gistFileAPI{
					// No backup.tar.gz file.
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	p := NewGitHubGistProvider(nil, "valid-token")
	_, err := p.Pull("no-backup-gist")
	if err == nil {
		t.Fatal("expected error when backup.tar.gz not found")
	}
	if !strings.Contains(err.Error(), "backup.tar.gz") {
		t.Errorf("error = %v, want mention of backup.tar.gz", err)
	}
}

func TestGitHubGistProvider_Pull_NoToken(t *testing.T) {
	p := NewGitHubGistProvider(nil, "")
	_, err := p.Pull("some-id")
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestGitHubGistProvider_List(t *testing.T) {
	_, cleanup := setupMockGistAPI(t, func(w http.ResponseWriter, r *http.Request) {
		// GitHub Gist API: GET /gists lists user gists.
		if r.Method == http.MethodGet && r.URL.Path == "/gists" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]gistResponse{
				{
					ID:          "list-1",
					Description: "bak backup 20260605-120000",
					HTMLURL:     "https://gist.github.com/list-1",
					Files: map[string]gistFileAPI{
						"backup.tar.gz": {Content: "data1"},
					},
				},
				{
					ID:          "list-2",
					Description: "bak backup 20260604-100000",
					HTMLURL:     "https://gist.github.com/list-2",
					Files: map[string]gistFileAPI{
						"backup.tar.gz": {Content: "data2"},
					},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	p := NewGitHubGistProvider(nil, "valid-token")
	metas, err := p.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("List length = %d, want 2", len(metas))
	}
	if metas[0].ID != "list-1" {
		t.Errorf("metas[0].ID = %q, want list-1", metas[0].ID)
	}
	if metas[0].URL != "https://gist.github.com/list-1" {
		t.Errorf("metas[0].URL = %q", metas[0].URL)
	}
	if metas[1].ID != "list-2" {
		t.Errorf("metas[1].ID = %q, want list-2", metas[1].ID)
	}
}

func TestGitHubGistProvider_TokenResolution(t *testing.T) {
	// Token passed to constructor should be used directly.
	p := NewGitHubGistProvider(nil, "direct-token")

	_, cleanup := setupMockGistAPI(t, func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != "Bearer direct-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodGet && r.URL.Path == "/gists/test-id" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(gistResponse{
				ID: "test-id",
				Files: map[string]gistFileAPI{
					"backup.tar.gz": {Content: "token-test-data"},
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})
	defer cleanup()

	data, err := p.Pull("test-id")
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if string(data) != "token-test-data" {
		t.Errorf("data = %q, want token-test-data", string(data))
	}
}

func TestGitHubGistProvider_PushIntegration(t *testing.T) {
	// Full integration: create, then update via Push.
	var storedFiles map[string]gistFileAPI
	gistID := ""
	created := false

	_, cleanup := setupMockGistAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/gists":
			var req gistCreateRequest
			json.NewDecoder(r.Body).Decode(&req)
			storedFiles = req.Files
			gistID = "integration-gist"
			created = true
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(gistResponse{
				ID:      gistID,
				HTMLURL: "https://gist.github.com/" + gistID,
				Files:   req.Files,
			})

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/gists/") && created:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(gistResponse{
				ID:    gistID,
				Files: storedFiles,
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer cleanup()

	cfg := &config.Config{}
	_ = cfg.Set("github.gist_id", "") // No gist yet.

	p := NewGitHubGistProvider(cfg, "valid-token")

	// First push: creates.
	id, err := p.Push([]byte("first-push"), PushMeta{BackupID: "test-1"})
	if err != nil {
		t.Fatalf("first Push: %v", err)
	}
	if id != "integration-gist" {
		t.Errorf("first id = %q, want integration-gist", id)
	}

	// Pull back what we pushed.
	data, err := p.Pull(id)
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	// Push base64-encodes the archive; Pull returns the encoded content.
	if string(data) != "Zmlyc3QtcHVzaA==" {
		t.Errorf("Pull data = %q, want Zmlyc3QtcHVzaA==", string(data))
	}
}

func TestNewGitHubGistProvider_NilConfig(t *testing.T) {
	// Should handle nil config gracefully.
	p := NewGitHubGistProvider(nil, "token")
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Name() != "github-gist" {
		t.Errorf("Name() = %q, want github-gist", p.Name())
	}
}
