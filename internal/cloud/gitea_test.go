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

func TestGiteaProvider_Name(t *testing.T) {
	p := NewGiteaProvider(nil, "test-token", "https://git.example.com", "user/repo")
	if p.Name() != "gitea" {
		t.Errorf("Name() = %q, want gitea", p.Name())
	}
}

func TestCodebergProvider_Name(t *testing.T) {
	p := NewCodebergProvider(nil, "test-token", "user/repo")
	if p.Name() != "codeberg" {
		t.Errorf("Name() = %q, want codeberg", p.Name())
	}
}

func TestCodebergProvider_DefaultBaseURL(t *testing.T) {
	p := NewCodebergProvider(nil, "test-token", "user/repo")
	if p.baseURL != "https://codeberg.org" {
		t.Errorf("baseURL = %q, want https://codeberg.org", p.baseURL)
	}
}

func TestGiteaProvider_Push_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GET to check existence → 404 (doesn't exist yet)
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/contents/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		// POST to create new file
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/contents/") {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(contentResponse{
				Content: contentFile{
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

	p := &GiteaProvider{
		cfg:     nil,
		token:   "valid-token",
		baseURL: srv.URL,
		repo:    "user/backups",
		branch:  "main",
		client:  srv.Client(),
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

func TestGiteaProvider_Push_Update(t *testing.T) {
	getCalled := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// GET returns existing file with SHA
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/contents/") {
			getCalled = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(contentResponse{
				Content: contentFile{
					Name: "20260605-120000.tar.gz",
					Path: "bak-backups/20260605-120000.tar.gz",
					SHA:  "sha-existing-001",
				},
			})
			return
		}
		// PUT to update with SHA
		if r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/contents/") {
			var req contentRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.SHA != "sha-existing-001" {
				w.WriteHeader(http.StatusConflict)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(contentResponse{
				Content: contentFile{
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

	p := &GiteaProvider{
		cfg:     nil,
		token:   "valid-token",
		baseURL: srv.URL,
		repo:    "user/backups",
		branch:  "main",
		client:  srv.Client(),
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

func TestGiteaProvider_Push_NoToken(t *testing.T) {
	p := &GiteaProvider{
		token:   "",
		baseURL: "https://git.example.com",
		repo:    "user/repo",
		branch:  "main",
	}
	_, err := p.Push([]byte("data"), PushMeta{BackupID: "test"})
	if err == nil {
		t.Fatal("expected error for missing token")
	}
	if !strings.Contains(err.Error(), "token is required") {
		t.Errorf("error = %v, want mention of token", err)
	}
}

func TestGiteaProvider_Push_NoRepo(t *testing.T) {
	p := &GiteaProvider{
		token:   "token",
		baseURL: "https://git.example.com",
		repo:    "",
		branch:  "main",
	}
	_, err := p.Push([]byte("data"), PushMeta{BackupID: "test"})
	if err == nil {
		t.Fatal("expected error for missing repo")
	}
}

func TestGiteaProvider_Pull(t *testing.T) {
	encoded := base64.StdEncoding.EncodeToString([]byte("backup-data-here"))
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/contents/") {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(contentResponse{
				Content: contentFile{
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

	p := &GiteaProvider{
		token:   "valid-token",
		baseURL: srv.URL,
		repo:    "user/backups",
		branch:  "main",
		client:  srv.Client(),
	}

	data, err := p.Pull("20260605-120000")
	if err != nil {
		t.Fatalf("Pull: %v", err)
	}
	if string(data) != "backup-data-here" {
		t.Errorf("Pull data = %q, want backup-data-here", string(data))
	}
}

func TestGiteaProvider_Pull_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p := &GiteaProvider{
		token:   "valid-token",
		baseURL: srv.URL,
		repo:    "user/backups",
		branch:  "main",
		client:  srv.Client(),
	}

	_, err := p.Pull("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent backup")
	}
}

func TestGiteaProvider_Pull_NoToken(t *testing.T) {
	p := &GiteaProvider{
		token:   "",
		baseURL: "https://git.example.com",
		repo:    "user/repo",
		branch:  "main",
	}
	_, err := p.Pull("some-id")
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestGiteaProvider_Pull_EmptyID(t *testing.T) {
	p := &GiteaProvider{
		token:   "token",
		baseURL: "https://git.example.com",
		repo:    "user/repo",
		branch:  "main",
	}
	_, err := p.Pull("")
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestGiteaProvider_List(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]contentResponse{
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

	p := &GiteaProvider{
		token:   "valid-token",
		baseURL: srv.URL,
		repo:    "user/backups",
		branch:  "main",
		client:  srv.Client(),
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
	if metas[1].ID != "20260604-100000" {
		t.Errorf("metas[1].ID = %q, want 20260604-100000", metas[1].ID)
	}
}

func TestGiteaProvider_List_Empty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode([]contentResponse{})
	}))
	defer srv.Close()

	p := &GiteaProvider{
		token:   "valid-token",
		baseURL: srv.URL,
		repo:    "user/backups",
		branch:  "main",
		client:  srv.Client(),
	}

	metas, err := p.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(metas) != 0 {
		t.Errorf("List length = %d, want 0", len(metas))
	}
}

func TestGiteaProvider_List_NoToken(t *testing.T) {
	p := &GiteaProvider{
		token:   "",
		baseURL: "https://git.example.com",
		repo:    "user/repo",
		branch:  "main",
	}
	_, err := p.List()
	if err == nil {
		t.Fatal("expected error for missing token")
	}
}

func TestGiteaProvider_TokenResolution(t *testing.T) {
	// Token passed directly should be used.
	p := NewGiteaProvider(nil, "direct-token", "https://git.example.com", "user/repo")
	if p.token != "direct-token" {
		t.Errorf("token = %q, want direct-token", p.token)
	}
}

func TestGiteaProvider_BaseURLDefault(t *testing.T) {
	p := NewGiteaProvider(nil, "token", "", "user/repo")
	if p.baseURL != "https://codeberg.org" {
		t.Errorf("baseURL = %q, want https://codeberg.org", p.baseURL)
	}
}

func TestGiteaProvider_PushIntegration(t *testing.T) {
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
			json.NewEncoder(w).Encode(contentResponse{
				Content: contentFile{
					Name:     "20260605-120000.tar.gz",
					Path:     "bak-backups/20260605-120000.tar.gz",
					SHA:      storedSHA,
					Encoding: "base64",
					Content:  storedContent,
				},
			})

		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/contents/"):
			var req contentRequest
			json.NewDecoder(r.Body).Decode(&req)
			storedContent = req.Content
			storedSHA = "sha-created-001"
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(contentResponse{
				Content: contentFile{
					Path: "bak-backups/20260605-120000.tar.gz",
					SHA:  storedSHA,
				},
			})

		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "/contents/"):
			var req contentRequest
			json.NewDecoder(r.Body).Decode(&req)
			storedContent = req.Content
			storedSHA = "sha-updated-002"
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(contentResponse{
				Content: contentFile{
					Path: "bak-backups/20260605-120000.tar.gz",
					SHA:  storedSHA,
				},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p := &GiteaProvider{
		token:   "valid-token",
		baseURL: srv.URL,
		repo:    "user/backups",
		branch:  "main",
		client:  srv.Client(),
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
	// Content was stored as base64; Pull decodes it.
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

func TestGiteaProvider_ConfigTokenResolution(t *testing.T) {
	// Token from config when none passed directly.
	cfg := &config.Config{}
	_ = cfg.Set("providers.gitea.token", "config-token")
	p := NewGiteaProvider(cfg, "", "https://git.example.com", "user/repo")
	if p.token != "config-token" {
		t.Errorf("token = %q, want config-token", p.token)
	}
}

func TestCodebergProvider_ConfigTokenResolution(t *testing.T) {
	cfg := &config.Config{}
	_ = cfg.Set("providers.codeberg.token", "codeberg-config-token")
	p := NewCodebergProvider(cfg, "", "user/repo")
	if p.token != "codeberg-config-token" {
		t.Errorf("token = %q, want codeberg-config-token", p.token)
	}
}

func TestNewGiteaProvider_NilConfig(t *testing.T) {
	p := NewGiteaProvider(nil, "token", "https://git.example.com", "user/repo")
	if p == nil {
		t.Fatal("expected non-nil provider")
	}
	if p.Name() != "gitea" {
		t.Errorf("Name() = %q, want gitea", p.Name())
	}
}
