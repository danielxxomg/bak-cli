package backup

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestDefaultPatterns(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	patterns := DefaultPatterns()
	if len(patterns) == 0 {
		t.Fatal("DefaultPatterns returned empty slice")
	}
	// All patterns should compile (already enforced by regexp.MustCompile).
	for i, p := range patterns {
		if p == nil {
			t.Errorf("pattern[%d] is nil", i)
		}
	}
}

func TestScanFile_NoSecrets(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cleanFile := filepath.Join(dir, "clean.txt")
	if err := os.WriteFile(cleanFile, []byte("hello world\nno secrets here\n"), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := ScanFile(cleanFile, DefaultPatterns())
	if err != nil {
		t.Fatalf("ScanFile: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d: %+v", len(results), results)
	}
}

func TestScanFile_DetectsSecrets(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	patterns := DefaultPatterns()

	tests := []struct {
		name    string
		content string
		want    int // minimum number of matches
	}{
		{
			name:    "GitHub token in env style",
			content: "GITHUB_TOKEN=ghp_abc123def456ghi789jkl012mno345pqr678stu\n",
			want:    1,
		},
		{
			name:    "API key assignment",
			content: "export OPENAI_API_KEY=sk-proj-abcdef1234567890abcdef1234567890\n",
			want:    1,
		},
		{
			name:    "Anthropic key",
			content: `ANTHROPIC_API_KEY: "sk-ant-api03-abcdef1234567890abcdef1234567890"` + "\n",
			want:    1,
		},
		{
			name:    "Generic token",
			content: "AUTH_TOKEN=abcdef1234567890abcdef1234567890abc\n",
			want:    1,
		},
		{
			name:    "Password assignment",
			content: "DB_PASSWORD=super_secret_password_123!\n",
			want:    1,
		},
		{
			name:    "Mixed content — secrets and non-secrets",
			content: "# Config\nPORT=8080\nDATABASE_URL=postgres://localhost/db\nGITHUB_TOKEN=ghp_abc123def456ghi789jkl012mno345pqr678stu\nLOG_LEVEL=info\n",
			want:    1,
		},
		{
			name:    "Ghps token inline",
			content: "token: ghps_abcdefghijklmnopqrstuvwxyz123456789012\n",
			want:    1,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			dir := t.TempDir()
			fp := filepath.Join(dir, "test.env")
			if err := os.WriteFile(fp, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			results, err := ScanFile(fp, patterns)
			if err != nil {
				t.Fatalf("ScanFile: %v", err)
			}
			if len(results) < tt.want {
				t.Errorf("got %d matches, want at least %d. Results: %+v", len(results), tt.want, results)
			}
		})
	}
}

func TestScanFile_CustomPatterns(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	custom := []*regexp.Regexp{
		regexp.MustCompile(`(?i)custom_secret\s*=\s*\w+`),
	}

	dir := t.TempDir()
	fp := filepath.Join(dir, "custom.cfg")
	if err := os.WriteFile(fp, []byte("custom_secret = hunter2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := ScanFile(fp, custom)
	if err != nil {
		t.Fatalf("ScanFile: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 match with custom pattern, got %d", len(results))
	}
	if results[0].Line != 1 {
		t.Errorf("line = %d, want 1", results[0].Line)
	}
}

func TestGenerateEnvExample(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()

	// Create test files.
	secretFile := filepath.Join(dir, "secrets.env")
	if err := os.WriteFile(secretFile, []byte(
		"# Config\nGITHUB_TOKEN=ghp_abc123def456ghi789jkl012mno345pqr678stu\nAPI_KEY=my-secret-key\nLOG_LEVEL=info\n",
	), 0644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	patterns := DefaultPatterns()

	err := GenerateEnvExample([]string{secretFile}, patterns, outputDir)
	if err != nil {
		t.Fatalf("GenerateEnvExample: %v", err)
	}

	// Read the generated file.
	examplePath := filepath.Join(outputDir, ".env.example")
	data, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("read .env.example: %v", err)
	}

	content := string(data)

	// The file should exist and contain placeholder substitutions.
	if !strings.Contains(content, "<YOUR_") {
		t.Errorf(".env.example doesn't contain any placeholder:\n%s", content)
	}

	// It should NOT contain the actual tokens.
	if strings.Contains(content, "ghp_abc123") {
		t.Errorf(".env.example still contains GitHub token:\n%s", content)
	}

	// It should preserve non-secret lines.
	if !strings.Contains(content, "LOG_LEVEL=info") {
		t.Errorf(".env.example is missing non-secret line:\n%s", content)
	}
}

func TestGenerateEnvExample_NoSecrets(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	cleanFile := filepath.Join(dir, "clean.env")
	if err := os.WriteFile(cleanFile, []byte("PORT=8080\nHOST=localhost\n"), 0644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	err := GenerateEnvExample([]string{cleanFile}, DefaultPatterns(), outputDir)
	if err != nil {
		t.Fatalf("GenerateEnvExample: %v", err)
	}

	examplePath := filepath.Join(outputDir, ".env.example")
	data, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("read .env.example: %v", err)
	}

	if strings.Contains(string(data), "<YOUR_") {
		t.Errorf(".env.example contains placeholder when no secrets present")
	}
}

func TestScanResult_Fields(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	fp := filepath.Join(dir, "test.env")
	if err := os.WriteFile(fp, []byte("GITHUB_TOKEN=ghp_abcdef1234567890123456789012345678901234\n"), 0644); err != nil {
		t.Fatal(err)
	}

	results, err := ScanFile(fp, DefaultPatterns())
	if err != nil {
		t.Fatalf("ScanFile: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	r := results[0]
	if r.FilePath != fp {
		t.Errorf("FilePath = %q, want %q", r.FilePath, fp)
	}
	if r.Line != 1 {
		t.Errorf("Line = %d, want 1", r.Line)
	}
	if r.Pattern == "" {
		t.Error("Pattern should not be empty")
	}
	if r.content == "" {
		t.Error("Content should not be empty")
	}
}

func TestScanFile_Nonexistent(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	_, err := ScanFile("/nonexistent/path/foo.env", DefaultPatterns())
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
