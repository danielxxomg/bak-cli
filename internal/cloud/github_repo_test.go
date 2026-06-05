package cloud

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/danielxxomg/bak-cli/internal/config"
)

func TestGitHubRepoProvider_Name(t *testing.T) {
	p := NewGitHubRepoProvider(nil, "test-token", "user/repo")
	if p.Name() != "github-repo" {
		t.Errorf("Name() = %q, want github-repo", p.Name())
	}
}

func TestGitHubRepoProvider_Push_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GET to check existence → 404 (doesn't exist yet)
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/contents/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// PUT to create new file (GitHub uses PUT for both create and update)
		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/contents/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(githubContentResponse{
				Content: githubContentFile{
					Name: "20260605-120000.tar.gz",
					Path: "bak-backups/20260605-120000.tar.gz",
					SHA:  "sha-new-001",
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := &GitHubRepoProvider{
		cfg:     nil,
		token:   "valid-token",
		repo:    "user/backups",
		branch:  "main",
		client:  srv.Client(),
		apiBase: srv.URL,
	}

	id, err := p.Push([]byte("test-archive-data"), PushMeta{
		BackupID:  "20260605-120000",
		CreatedAt: time.Now(),
		Hostname:  "testbox",
	})
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if id != "20260605-120000" {
		t.Errorf("Push id = %q, want 20260605-120000", id)
	}
}

func TestGitHubRepoProvider_Push_Update(t *testing.T) {
	getCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GET returns existing file with SHA
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/contents/") {
			getCalled = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(githubContentResponse{
				Content: githubContentFile{
					Name: "20260605-120000.tar.gz",
					Path: "bak-backups/20260605-120000.tar.gz",
					SHA:  "sha-existing-001",
				},
			})
			return
		}
		// PUT to update with SHA
		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/contents/") {
			var req githubContentRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.SHA != "sha-existing-001" {
				w.WriteHeader(http.StatusConflict)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(githubContentResponse{
				Content: githubContentFile{
					Name: "20260605-120000.tar.gz",
					Path: "bak-backups/20260605-120000.tar.gz",
					SHA:  "sha-new-002",
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := &GitHubRepoProvider{
		cfg:     nil,
		token:   "valid-token",
		repo:    "user/backups",
		branch:  "main",
		client:  srv.Client(),
		apiBase: srv.URL,
	}

	id, err := p.Push([]byte("updated-archive"), PushMeta{
		BackupID: "20260605-120000",
	})
	if err != nil {
		t.Fatalf("Push: %v", err)
	}
	if id != "20260605-120000" {
		t.Errorf("Push id = %q, want 20260605-120000", id)
	}
	if !getCalled {
		t.Error("GET was not called to check file existence")
	}
}

func TestGitHubRepoProvider_Push_NoToken(t *testing.T) {
	p := &GitHubRepoProvider{
		token:   "",
		repo:    "user/repo",
		branch:  "main",
		apiBase: "https://api.github.com",
	}
	_, err := p.Push([]byte("data"), PushMeta{BackupID: "test"})
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if !strings.Contains(err.Error(), "token is required") {
		t.Errorf("error = %v, want mention of token", err)
	}
}

func TestGitHubRepoProvider_Push_NoRepo(t *testing.T) {
	p := &GitHubRepoProvider{
		token:   "token",
		repo:    "",
		branch:  "main",
		apiBase: "https://api.github.com",
	}
	_, err := p.Push([]byte("data"), PushMeta{BackupID: "test"})
	if err == nil {
		t.Fatal("expected error for missing repo")
	}
}

func TestGitHubRepoProvider_Pull(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("backup-data-here"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/contents/") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(githubContentResponse{
				Content: githubContentFile{
					Name:     "20260605-120000.tar.gz",
					Path:     "bak-backups/20260605-120000.tar.gz",
					SHA:      "sha-pull-001",
					Encoding: "base64",
					Content:  encoded,
				},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := &GitHubRepoProvider{
		token:   "valid-token",
		repo:    "user/backups",
		branch:  "main",
		client:  srv.Client(),
		apiBase: srv.URL,
	}

	data, err := p.Pull("20260605-120000")
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if string(data) != "backup-data-here" {
		t.Errorf("Pull data = %q, want backup-data-here", string(data))
	}
}

func TestGitHubRepoProvider_Pull_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := &GitHubRepoProvider{
		token:   "valid-token",
		repo:    "user/backups",
		branch:  "main",
		client:  srv.Client(),
		apiBase: srv.URL,
	}

	_, err := p.Pull("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent backup")
	}
}

func TestGitHubRepoProvider_Pull_NoToken(t *testing.T) {
	p := &GitHubRepoProvider{
		token:   "",
		repo:    "user/repo",
		branch:  "main",
		apiBase: "https://api.github.com",
	}
	_, err := p.Pull("some-id")
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestGitHubRepoProvider_Pull_EmptyID(t *testing.T) {
	p := &GitHubRepoProvider{
		token:   "token",
		repo:    "user/repo",
		branch:  "main",
		apiBase: "https://api.github.com",
	}
	_, err := p.Pull("")
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestGitHubRepoProvider_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]githubContentResponse{
			{
				Name: "20260605-120000.tar.gz",
				Path: "bak-backups/20260605-120000.tar.gz",
				SHA:  "sha-list-1",
				Size: 102400,
			},
			{
				Name: "20260604-100000.tar.gz",
				Path: "bak-backups/20260604-100000.tar.gz",
				SHA:  "sha-list-2",
				Size: 51200,
			},
		})
	}))
	defer srv.Close()

	p := &GitHubRepoProvider{
		token:   "valid-token",
		repo:    "user/backups",
		branch:  "main",
		client:  srv.Client(),
		apiBase: srv.URL,
	}

	metas, err := p.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 2 {
		t.Fatalf("List length = %d, want 2", len(metas))
	}
	if metas[0].ID != "20260605-120000" {
		t.Errorf("metas[0].ID = %q, want 20260605-120000", metas[0].ID)
	}
	if metas[0].Size != 102400 {
		t.Errorf("metas[0].Size = %d, want 102400", metas[0].Size)
	}
}

func TestGitHubRepoProvider_List_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]githubContentResponse{})
	}))
	defer srv.Close()

	p := &GitHubRepoProvider{
		token:   "valid-token",
		repo:    "user/backups",
		branch:  "main",
		client:  srv.Client(),
		apiBase: srv.URL,
	}

	metas, err := p.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 0 {
		t.Errorf("List length = %d, want 0", len(metas))
	}
}

func TestGitHubRepoProvider_List_NoToken(t *testing.T) {
	p := &GitHubRepoProvider{
		token:   "",
		repo:    "user/repo",
		branch:  "main",
		apiBase: "https://api.github.com",
	}
	_, err := p.List()
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestGitHubRepoProvider_TokenResolution(t *testing.T) {
	// Token passed directly should be used.
	p := NewGitHubRepoProvider(nil, "direct-token", "user/repo")
	if p.token != "direct-token" {
		t.Errorf("token = %q, want direct-token", p.token)
	}
}

func TestGitHubRepoProvider_PushIntegration(t *testing.T) {
	// Full integration: push then pull round-trip.
	var storedContent string
	var storedSHA string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		auth := r.Header.Get("Authorization")
		if auth != "Bearer valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/contents/"):
			if storedSHA == "" {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			json.NewEncoder(w).Encode(githubContentResponse{
				Content: githubContentFile{
					Name:     "20260605-120000.tar.gz",
					Path:     "bak-backups/20260605-120000.tar.gz",
					SHA:      storedSHA,
					Encoding: "base64",
					Content:  storedContent,
				},
			})

		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/contents/"):
			var req githubContentRequest
			json.NewDecoder(r.Body).Decode(&req)
			storedContent = req.Content
			if storedSHA == "" {
				storedSHA = "sha-created-001"
				w.WriteHeader(http.StatusCreated)
			} else {
				storedSHA = "sha-updated-002"
				w.WriteHeader(http.StatusOK)
			}
			json.NewEncoder(w).Encode(githubContentResponse{
				Content: githubContentFile{
					Path: "bak-backups/20260605-120000.tar.gz",
					SHA:  storedSHA,
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p := &GitHubRepoProvider{
		token:   "valid-token",
		repo:    "user/backups",
		branch:  "main",
		client:  srv.Client(),
		apiBase: srv.URL,
	}

	// First push: creates new file.
	id, err := p.Push([]byte("first-push"), PushMeta{BackupID: "20260605-120000"})
	if err != nil {
		t.Fatalf("first Push: %v", err)
	}
	if id != "20260605-120000" {
		t.Errorf("id = %q, want 20260605-120000", id)
	}

	// Pull it back.
	data, err := p.Pull(id)
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if string(data) != "first-push" {
		t.Errorf("Pull data = %q, want first-push", string(data))
	}

	// Second push: updates existing file.
	_, err = p.Push([]byte("second-push"), PushMeta{BackupID: "20260605-120000"})
	if err != nil {
		t.Fatalf("second Push: %v", err)
	}

	data, err = p.Pull("20260605-120000")
	if err != nil {
		t.Fatalf("Pull after update: %v", err)
	}
	if string(data) != "second-push" {
		t.Errorf("Pull data after update = %q, want second-push", string(data))
	}
}

func TestGitHubRepoProvider_ConfigTokenResolution(t *testing.T) {
	cfg := &config.Config{}
	_ = cfg.Set("providers.github-repo.token", "config-token")
	p := NewGitHubRepoProvider(cfg, "", "user/repo")
	if p.token != "config-token" {
		t.Errorf("token = %q, want config-token", p.token)
	}
}

func TestNewGitHubRepoProvider_NilConfig(t *testing.T) {
	p := NewGitHubRepoProvider(nil, "token", "user/repo")
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Name() != "github-repo" {
		t.Errorf("Name() = %q, want github-repo", p.Name())
	}
}

func TestGitHubRepoProvider_DefaultAPIBase(t *testing.T) {
	p := NewGitHubRepoProvider(nil, "token", "user/repo")
	if p.apiBase != "https://api.github.com" {
		t.Errorf("apiBase = %q, want https://api.github.com", p.apiBase)
	}
}
