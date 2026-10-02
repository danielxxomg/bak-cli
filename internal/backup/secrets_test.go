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

func TestScanFile_DocumentedTokenFamilies(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	patterns := DefaultPatterns()

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "GitHub OAuth token (gho_*)",
			content: "VAR_OACC=gho_0123456789abcdefghijklmnopqrstuvE0A1B2C3D4\n",
			want:    1,
		},
		{
			name:    "GitHub user-to-server token (ghu_*)",
			content: "VAR_USER=ghu_0123456789abcdefghijklmnopqrstuvE0A1B2C3D4\n",
			want:    1,
		},
		{
			name:    "GitHub server-to-server token (ghs_*)",
			content: "VAR_SRV=ghs_0123456789abcdefghijklmnopqrstuvE0A1B2C3D4\n",
			want:    1,
		},
		{
			name:    "GitHub refresh token (ghr_*)",
			content: "VAR_REF=ghr_0123456789abcdefghijklmnopqrstuvE0A1B2C3D4\n",
			want:    1,
		},
		{
			name:    "Slack bot token (xoxb-*)",
			content: "VAR_BOT=xoxb-123456789012-1234567890123-abcdefghij!?#klmnop\n",
			want:    1,
		},
		{
			name:    "Slack user token (xoxp-*)",
			content: "VAR_USR=xoxp-123456789012-1234567890123-abcdefghij!?#klmnop\n",
			want:    1,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			dir := t.TempDir()
			fp := filepath.Join(dir, "tokens.env")
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

func TestGenerateEnvExample_DocumentedTokenFamilies(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	dir := t.TempDir()
	envFile := filepath.Join(dir, "documented.env")
	content := strings.Join([]string{
		"VAR_OACC=gho_0123456789abcdefghijklmnopqrstuvE0A1B2C3D4",
		"VAR_USER=ghu_0123456789abcdefghijklmnopqrstuvE0A1B2C3D4",
		"VAR_SRV=ghs_0123456789abcdefghijklmnopqrstuvE0A1B2C3D4",
		"VAR_REF=ghr_0123456789abcdefghijklmnopqrstuvE0A1B2C3D4",
		"VAR_BOT=xoxb-123456789012-1234567890123-abcdefghij!?#klmnop",
		"VAR_USR=xoxp-123456789012-1234567890123-abcdefghij!?#klmnop",
		"APP_PORT=8080",
	}, "\n") + "\n"

	if err := os.WriteFile(envFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	patterns := DefaultPatterns()

	if err := GenerateEnvExample([]string{envFile}, patterns, outputDir); err != nil {
		t.Fatalf("GenerateEnvExample: %v", err)
	}

	examplePath := filepath.Join(outputDir, ".env.example")
	data, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("read .env.example: %v", err)
	}

	example := string(data)

	rawTokens := []string{
		"gho_0123456789abcdefghijklmnopqrstuvE0A1B2C3D4",
		"ghu_0123456789abcdefghijklmnopqrstuvE0A1B2C3D4",
		"ghs_0123456789abcdefghijklmnopqrstuvE0A1B2C3D4",
		"ghr_0123456789abcdefghijklmnopqrstuvE0A1B2C3D4",
		"xoxb-123456789012-1234567890123-abcdefghij!?#klmnop",
		"xoxp-123456789012-1234567890123-abcdefghij!?#klmnop",
	}

	for _, token := range rawTokens {
		if strings.Contains(example, token) {
			t.Errorf(".env.example still contains raw token: %s", token)
		}
	}

	if !strings.Contains(example, "APP_PORT=8080") {
		t.Errorf(".env.example missing preserved non-secret line APP_PORT=8080")
	}
}

func TestScanFile_DocumentedTokenFamilies_Triangulation(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	patterns := DefaultPatterns()

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "Too-short GitHub token rejected",
			content: "VAR_OACC=gho_tooshort123\n",
			want:    0,
		},
		{
			name:    "Too-short Slack token rejected",
			content: "VAR_BOT=xoxb-short\n",
			want:    0,
		},
		{
			name:    "Unrecognized gh prefix rejected",
			content: "VAR_OTHER=ghx_0123456789abcdefghijklmnopqrstuvE0A1B2C3D4\n",
			want:    0,
		},
		{
			name:    "Unrecognized xox prefix rejected",
			content: "VAR_OTHER=xoxa-123456789012-1234567890123-abcdefghij!?#klmnop\n",
			want:    0,
		},
		{
			name:    "Case-insensitive GitHub OAuth token",
			content: "VAR_OACC=GHO_0123456789ABCDEFGHIJKLMNOPQRSTUVe0a1b2c3d4\n",
			want:    1,
		},
		{
			name:    "Case-insensitive Slack bot token",
			content: "VAR_BOT=XOXB-123456789012-1234567890123-ABCDEFGHIJ!?#KLMNOP\n",
			want:    1,
		},
		{
			name:    "JSON embedded GitHub token",
			content: `{"custom_field": "gho_0123456789abcdefghijklmnopqrstuvE0A1B2C3D4"}` + "\n",
			want:    1,
		},
		{
			name:    "JSON embedded Slack token",
			content: `{"custom_field": "xoxp-123456789012-1234567890123-abcdefghij!?#klmnop"}` + "\n",
			want:    1,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			dir := t.TempDir()
			fp := filepath.Join(dir, "triangulate.env")
			if err := os.WriteFile(fp, []byte(tt.content), 0644); err != nil {
				t.Fatal(err)
			}

			results, err := ScanFile(fp, patterns)
			if err != nil {
				t.Fatalf("ScanFile: %v", err)
			}
			if len(results) != tt.want {
				t.Errorf("got %d matches, want %d. Results: %+v", len(results), tt.want, results)
			}
		})
	}
}
