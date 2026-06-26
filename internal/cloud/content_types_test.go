package cloud

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetFileSHA_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", acceptJSON)
		json.NewEncoder(w).Encode(contentResponse{
			Content: contentFile{
				SHA: "abc123def456",
			},
		})
	}))
	defer srv.Close()

	sha, err := getFileSHA(srv.Client(), "token", srv.URL+"/path/to/file")
	if err != nil {
		t.Fatalf("getFileSHA: %v", err)
	}
	if sha != "abc123def456" {
		t.Errorf("sha = %q, want abc123def456", sha)
	}
}

func TestGetFileSHA_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	sha, err := getFileSHA(srv.Client(), "token", srv.URL+"/path/to/file")
	if err != nil {
		t.Fatalf("getFileSHA: unexpected error: %v", err)
	}
	if sha != "" {
		t.Errorf("sha = %q, want empty string for not found", sha)
	}
}

func TestGetFileSHA_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer srv.Close()

	_, err := getFileSHA(srv.Client(), "token", srv.URL+"/path/to/file")
	if err == nil {
		t.Fatal("expected error for 500 status")
	}
	if !strings.Contains(err.Error(), "api error 500") {
		t.Errorf("error = %v, want api error 500", err)
	}
}

func TestWriteContentFile_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	req := contentRequest{
		Message: "commit message",
		Content: "base64content",
		Branch:  "main",
	}

	err := writeContentFile(srv.Client(), "token", http.MethodPut, acceptJSON, srv.URL+"/path", req)
	if err != nil {
		t.Fatalf("writeContentFile: %v", err)
	}
}

func TestWriteContentFile_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		w.Write([]byte("conflict"))
	}))
	defer srv.Close()

	req := contentRequest{
		Message: "commit",
		Content: "data",
		Branch:  "main",
		SHA:     "wrong-sha",
	}

	err := writeContentFile(srv.Client(), "token", http.MethodPut, acceptJSON, srv.URL+"/path", req)
	if err == nil {
		t.Fatal("expected error for 409 status")
	}
	if !strings.Contains(err.Error(), "api error 409") {
		t.Errorf("error = %v, want api error 409", err)
	}
}

func TestWriteContentFile_WithSHA(t *testing.T) {
	var receivedSHA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var cr contentRequest
		json.NewDecoder(r.Body).Decode(&cr)
		receivedSHA = cr.SHA
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	req := contentRequest{
		Message: "update",
		Content: "data",
		Branch:  "main",
		SHA:     "existing-sha",
	}

	err := writeContentFile(srv.Client(), "token", http.MethodPut, acceptJSON, srv.URL+"/path", req)
	if err != nil {
		t.Fatalf("writeContentFile: %v", err)
	}
	if receivedSHA != "existing-sha" {
		t.Errorf("SHA = %q, want existing-sha", receivedSHA)
	}
}
