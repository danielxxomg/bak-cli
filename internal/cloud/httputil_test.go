package cloud

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPullContentFromAPI(t *testing.T) {
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
			id:    "20260101-000000",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(contentResponse{
					Content: contentFile{
						Name:     "20260101-000000.tar.gz",
						Encoding: "base64",
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
			id:    "20260101-000000",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				t.Error("server must not be called when token empty")
				w.WriteHeader(500)
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
				w.WriteHeader(500)
			},
			wantErr:    true,
			wantErrSub: "backup ID is required",
		},
		{
			name:  "empty repo returns validation error before request",
			token: "token",
			repo:  "",
			id:    "20260101-000000",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				t.Error("server must not be called when repo empty")
				w.WriteHeader(500)
			},
			wantErr:    true,
			wantErrSub: "repo is required",
		},
		{
			name:  "4xx status returns wrapped error",
			token: "token",
			repo:  "user/repo",
			id:    "20260101-000000",
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
			id:    "20260101-000000",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(contentResponse{
					Content: contentFile{
						Name:     "bad.tar.gz",
						Encoding: "base64",
						Content:  "!!!not-valid-base64!!!",
					},
				})
			},
			wantErr:    true,
			wantErrSub: "decode content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(tt.handler)
			defer srv.Close()

			url := srv.URL + "/contents/" + tt.id
			data, err := pullContentFromAPI(srv.Client(), tt.token, tt.repo, tt.id, url, "application/json", "pull")

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
