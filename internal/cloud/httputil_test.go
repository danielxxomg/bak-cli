package cloud

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPullContentFromAPI(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	validEncoded := base64.StdEncoding.EncodeToString([]byte("archive-bytes"))

	tests := []struct {
		name       string
		token      string
		repo       string
		id         string
		handler    http.HandlerFunc
		wantData   string
		wantErr    bool
		wantErrSub string
	}{
		{
			name:  "success returns decoded content",
			token: "token",
			repo:  "user/repo",
			id:    testOldID,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", acceptJSON)
				_ = json.NewEncoder(w).Encode(contentResponse{
					Content: contentFile{
						Name:     "20260101-000000.tar.gz",
						Encoding: testEncoding,
						Content:  validEncoded,
					},
				})
			},
			wantData: "archive-bytes",
			wantErr:  false,
		},
		{
			name:  "empty token returns validation error before request",
			token: "",
			repo:  "user/repo",
			id:    testOldID,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				t.Error("server must not be called when token empty")
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr:    true,
			wantErrSub: "token is required",
		},
		{
			name:  "empty id returns validation error before request",
			token: "token",
			repo:  "user/repo",
			id:    "",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				t.Error("server must not be called when id empty")
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr:    true,
			wantErrSub: "backup ID is required",
		},
		{
			name:  "empty repo returns validation error before request",
			token: "token",
			repo:  "",
			id:    testOldID,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				t.Error("server must not be called when repo empty")
				w.WriteHeader(http.StatusInternalServerError)
			},
			wantErr:    true,
			wantErrSub: "repo is required",
		},
		{
			name:  "4xx status returns wrapped error",
			token: "token",
			repo:  "user/repo",
			id:    testOldID,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"message":"forbidden"}`))
			},
			wantErr:    true,
			wantErrSub: "api error 403",
		},
		{
			name:  "invalid base64 content returns decode error",
			token: "token",
			repo:  "user/repo",
			id:    testOldID,
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", acceptJSON)
				_ = json.NewEncoder(w).Encode(contentResponse{
					Content: contentFile{
						Name:     "bad.tar.gz",
						Encoding: testEncoding,
						Content:  "!!!not-valid-base64!!!",
					},
				})
			},
			wantErr:    true,
			wantErrSub: "decode content",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			url := srv.URL + "/contents/" + tt.id
			data, err := pullContentFromAPI(srv.Client(), tt.token, tt.repo, tt.id, url, acceptJSON, "pull")

			if tt.wantErr {
				if err == nil {
					t.Fatalf("pullContentFromAPI() error = nil, want error containing %q", tt.wantErrSub)
				}
				if !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Errorf("pullContentFromAPI() error = %q, want substring %q", err.Error(), tt.wantErrSub)
				}
				// Error must be wrapped with the provider prefix from testWrap.
				if !strings.HasPrefix(err.Error(), "pull: ") {
					t.Errorf("pullContentFromAPI() error = %q, want prefix %q", err.Error(), "pull: ")
				}
				return
			}

			if err != nil {
				t.Fatalf("pullContentFromAPI() unexpected error = %v", err)
			}
			if string(data) != tt.wantData {
				t.Errorf("pullContentFromAPI() data = %q, want %q", string(data), tt.wantData)
			}
		})
	}
}

// TestListContentsDir verifies the shared cloud list helper parameterized by
// URL, accept header, error prefix, and per-item URL builder. Covers the spec
// scenarios: shared logic parameterized by URL/headers/prefix, 404 returns
// empty, and HTTP error propagated with correct prefix (task 2.1, RED).
func TestListContentsDir(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dirItems := []contentResponse{
		{Name: "20260101-000000.tar.gz", Size: 100},
		{Name: "20260102-000000.tar.gz", Size: 200},
		{Name: "20260103-000000.tar.gz", Size: 300},
	}

	tests := []struct {
		name        string
		token       string
		accept      string
		errPrefix   string
		wantPath    string // expected request path
		handler     http.HandlerFunc
		urlBuilder  func(item contentResponse) string
		wantCount   int
		wantURL     string // expected BackupMeta.URL for the first item
		wantErr     bool
		wantErrSub  string
		wantErrPref string
	}{
		{
			name:      "gitea-like success returns metas with gitea URLs and json accept",
			token:     "gitea-token",
			accept:    acceptJSON,
			errPrefix: "gitea: list",
			wantPath:  "/api/v1/repos/user/repo/contents/bak-backups",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Accept"); got != acceptJSON {
					t.Errorf("Accept header = %q, want %q", got, acceptJSON)
				}
				if got := r.Header.Get("Authorization"); got != "Bearer gitea-token" {
					t.Errorf("Authorization = %q, want %q", got, "Bearer gitea-token")
				}
				_ = json.NewEncoder(w).Encode(dirItems)
			},
			urlBuilder: func(item contentResponse) string {
				return "https://codeberg.org/user/repo/src/branch/main/bak-backups/" + item.Name
			},
			wantCount: 3,
			wantURL:   "https://codeberg.org/user/repo/src/branch/main/bak-backups/20260101-000000.tar.gz",
		},
		{
			name:      "github-like success uses different accept and github URL builder",
			token:     "gh-token",
			accept:    acceptGitHub,
			errPrefix: "list github-repo",
			wantPath:  "/repos/user/repo/contents/bak-backups",
			handler: func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Accept"); got != acceptGitHub {
					t.Errorf("Accept header = %q, want %q", got, acceptGitHub)
				}
				_ = json.NewEncoder(w).Encode(dirItems)
			},
			urlBuilder: func(item contentResponse) string {
				return "https://github.com/user/repo/blob/main/bak-backups/" + item.Name
			},
			wantCount: 3,
			wantURL:   "https://github.com/user/repo/blob/main/bak-backups/20260101-000000.tar.gz",
		},
		{
			name:       "404 returns empty slice and nil error",
			token:      "tok",
			accept:     acceptJSON,
			errPrefix:  "gitea: list",
			wantPath:   testContentsDir,
			handler:    func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
			urlBuilder: func(contentResponse) string { return "" },
			wantCount:  0,
		},
		{
			name:        "non-2xx error wrapped with provider prefix",
			token:       "tok",
			accept:      acceptJSON,
			errPrefix:   "gitea: list",
			wantPath:    testContentsDir,
			handler:     func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusInternalServerError) },
			urlBuilder:  func(contentResponse) string { return "" },
			wantErr:     true,
			wantErrSub:  "api error 500",
			wantErrPref: "gitea: list",
		},
		{
			name:        "github error uses github prefix",
			token:       "tok",
			accept:      acceptGitHub,
			errPrefix:   "list github-repo",
			wantPath:    testContentsDir,
			handler:     func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) },
			urlBuilder:  func(contentResponse) string { return "" },
			wantErr:     true,
			wantErrSub:  "api error 403",
			wantErrPref: "list github-repo",
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			var gotPath string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				tt.handler(w, r)
			}))
			defer srv.Close()

			url := srv.URL + tt.wantPath
			metas, err := listContentsDir(srv.Client(), url, tt.token, tt.accept, tt.errPrefix, tt.urlBuilder)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("listContentsDir() error = nil, want error")
				}
				if !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Errorf("listContentsDir() error = %q, want substring %q", err.Error(), tt.wantErrSub)
				}
				if !strings.HasPrefix(err.Error(), tt.wantErrPref) {
					t.Errorf("listContentsDir() error = %q, want prefix %q", err.Error(), tt.wantErrPref)
				}
				return
			}
			if err != nil {
				t.Fatalf("listContentsDir() unexpected error = %v", err)
			}
			if gotPath != tt.wantPath {
				t.Errorf("request path = %q, want %q", gotPath, tt.wantPath)
			}
			if len(metas) != tt.wantCount {
				t.Fatalf("metas count = %d, want %d", len(metas), tt.wantCount)
			}
			if tt.wantCount > 0 {
				if metas[0].URL != tt.wantURL {
					t.Errorf("metas[0].URL = %q, want %q", metas[0].URL, tt.wantURL)
				}
				if metas[0].ID != testOldID {
					t.Errorf("metas[0].ID = %q, want %q", metas[0].ID, testOldID)
				}
				if metas[0].BackupID != testOldID {
					t.Errorf("metas[0].BackupID = %q, want %q", metas[0].BackupID, testOldID)
				}
				if metas[0].Size != 100 {
					t.Errorf("metas[0].Size = %d, want %d", metas[0].Size, 100)
				}
			}
			if tt.wantCount == 0 && metas != nil && len(metas) != 0 {
				t.Errorf("expected empty result, got %v", metas)
			}
		})
	}
}
