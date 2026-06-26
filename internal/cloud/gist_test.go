package cloud

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// setupMockGistAPI starts a test HTTP server and configures the
// cloud package to use it. Returns the server and a cleanup function.
func setupMockGistAPI(t *testing.T, handler http.HandlerFunc) (*httptest.Server, func()) {
	t.Helper()

	srv := httptest.NewServer(handler)

	origBase := GistAPIBase
	origClient := httpClient

	GistAPIBase = srv.URL
	httpClient = srv.Client()

	return srv, func() {
		GistAPIBase = origBase
		httpClient = origClient
		srv.Close()
	}
}

func TestCreateGist_Validation(t *testing.T) {
	t.Run("empty token", func(t *testing.T) {
		_, err := CreateGist("", "desc", []GistFile{{Filename: "f", Content: "c"}})
		if err == nil {
			t.Fatal("expected error for empty token")
		}
		if !strings.Contains(err.Error(), "token is required") {
			t.Errorf("error = %v, want 'token is required'", err)
		}
	})

	t.Run("no files", func(t *testing.T) {
		_, err := CreateGist("token", "desc", nil)
		if err == nil {
			t.Fatal("expected error for empty files")
		}
		if !strings.Contains(err.Error(), "at least one file") {
			t.Errorf("error = %v, want 'at least one file'", err)
		}
	})
}

func TestGistCRUD_RoundTrip(t *testing.T) {
	// In-memory gist store for the mock server.
	var storedGist gistResponse

	_, cleanup := setupMockGistAPI(t, func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth != testBearerToken {
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"message": "Bad credentials"})
			return
		}

		w.Header().Set("Content-Type", acceptJSON)

		switch {
		case r.Method == http.MethodPost && r.URL.Path == gistsEndpoint:
			var req gistCreateRequest
			json.NewDecoder(r.Body).Decode(&req)
			storedGist = gistResponse{
				ID:          "gist-test-1",
				Description: req.Description,
				Public:      false,
				HTMLURL:     "https://gist.github.com/gist-test-1",
				Files:       req.Files,
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(storedGist)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/gists/"):
			if storedGist.ID == "" {
				w.WriteHeader(http.StatusNotFound)
				json.NewEncoder(w).Encode(map[string]string{"message": "Not Found"})
				return
			}
			json.NewEncoder(w).Encode(storedGist)

		case r.Method == http.MethodPatch && strings.HasPrefix(r.URL.Path, "/gists/"):
			var req gistUpdateRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.Description != "" {
				storedGist.Description = req.Description
			}
			if req.Files != nil {
				storedGist.Files = req.Files
			}
			json.NewEncoder(w).Encode(storedGist)

		case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, "/gists/"):
			storedGist = gistResponse{}
			w.WriteHeader(http.StatusNoContent)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
	defer cleanup()

	// --- Test: Create ---
	id, err := CreateGist("valid-token", "round-trip test", []GistFile{
		{Filename: "hello.txt", Content: "Hello, Gist!"},
	})
	if err != nil {
		t.Fatalf("CreateGist: %v", err)
	}
	if id != "gist-test-1" {
		t.Errorf("id = %q, want gist-test-1", id)
	}

	// --- Test: Get ---
	files, err := GetGist("valid-token", id)
	if err != nil {
		t.Fatalf("GetGist: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("got %d files, want 1", len(files))
	}
	if files[0].Filename != "hello.txt" {
		t.Errorf("filename = %q, want hello.txt", files[0].Filename)
	}
	if files[0].Content != "Hello, Gist!" {
		t.Errorf("content = %q, want Hello, Gist!", files[0].Content)
	}

	// --- Test: Update ---
	err = UpdateGist("valid-token", id, "updated desc", []GistFile{
		{Filename: "hello.txt", Content: "Updated content"},
		{Filename: "new.txt", Content: "New file"},
	})
	if err != nil {
		t.Fatalf("UpdateGist: %v", err)
	}

	// Verify update via Get.
	files, err = GetGist("valid-token", id)
	if err != nil {
		t.Fatalf("GetGist after update: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("after update: got %d files, want 2", len(files))
	}

	// --- Test: Delete ---
	err = DeleteGist("valid-token", id)
	if err != nil {
		t.Fatalf("DeleteGist: %v", err)
	}

	// Get after delete should fail.
	_, err = GetGist("valid-token", id)
	if err == nil {
		t.Fatal("GetGist after delete should fail")
	}
}

func TestGist_InvalidToken(t *testing.T) {
	_, cleanup := setupMockGistAPI(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Bad credentials"})
	})
	defer cleanup()

	_, err := CreateGist("bad-token", "test", []GistFile{
		{Filename: "f.txt", Content: "c"},
	})
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("error should mention 401: %v", err)
	}
}

func TestGist_NetworkError(t *testing.T) {
	// Point at an unreachable address.
	origBase := GistAPIBase
	origClient := httpClient
	GistAPIBase = "http://127.0.0.1:1"
	httpClient = &http.Client{}
	defer func() {
		GistAPIBase = origBase
		httpClient = origClient
	}()

	_, err := CreateGist("token", "test", []GistFile{
		{Filename: "f.txt", Content: "c"},
	})
	if err == nil {
		t.Fatal("expected error for network failure")
	}
}

func TestGist_NullInputs(t *testing.T) {
	// GetGist with empty token.
	_, err := GetGist("", "gist123")
	if err == nil {
		t.Fatal("GetGist: expected error for empty token")
	}

	// GetGist with empty gist ID.
	_, err = GetGist("token", "")
	if err == nil {
		t.Fatal("GetGist: expected error for empty gist ID")
	}

	// DeleteGist with empty token.
	err = DeleteGist("", "gist123")
	if err == nil {
		t.Fatal("DeleteGist: expected error for empty token")
	}

	// UpdateGist with empty token.
	err = UpdateGist("", "gist123", "desc", []GistFile{{Filename: "f", Content: "c"}})
	if err == nil {
		t.Fatal("UpdateGist: expected error for empty token")
	}
}
