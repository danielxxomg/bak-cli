package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	configtest "github.com/danielxxomg/bak-cli/internal/config/testutil"
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

func TestGenerateEnvExample_HomeRelativeHeaderAndNoAbsolutePaths(t *testing.T) { //nolint:paralleltest // uses t.Setenv
	homeDir := t.TempDir()
	configtest.SetConfigHome(t, homeDir)

	configDir := filepath.Join(homeDir, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	secretFile := filepath.Join(configDir, "secrets.env")
	secretContent := "# Configuration\nGITHUB_TOKEN=ghp_abc123def456ghi789jkl012mno345pqr678stu\nLOG_LEVEL=info\n"
	if err := os.WriteFile(secretFile, []byte(secretContent), 0644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	patterns := DefaultPatterns()

	if err := GenerateEnvExample([]string{secretFile}, patterns, outputDir); err != nil {
		t.Fatalf("GenerateEnvExample: %v", err)
	}

	examplePath := filepath.Join(outputDir, ".env.example")
	data, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("read .env.example: %v", err)
	}
	content := string(data)

	// Must use home-relative path in header
	wantHeader := "# ---- ~/.config/opencode/secrets.env ----"
	if !strings.Contains(content, wantHeader) {
		t.Errorf(".env.example missing home-relative header %q, got:\n%s", wantHeader, content)
	}

	// Must not leak absolute path or homeDir
	if strings.Contains(content, homeDir) {
		t.Errorf(".env.example leaked absolute homeDir %q:\n%s", homeDir, content)
	}

	// Must contain placeholder
	if !strings.Contains(content, "<YOUR_") {
		t.Errorf(".env.example missing placeholder:\n%s", content)
	}

	// Must not contain raw secret
	if strings.Contains(content, "ghp_abc123") {
		t.Errorf(".env.example contains raw secret value:\n%s", content)
	}

	// Must preserve non-secret lines
	if !strings.Contains(content, "LOG_LEVEL=info") {
		t.Errorf(".env.example missing non-secret line:\n%s", content)
	}
}

func TestGenerateEnvExample_UnreadableSourceCleanNote(t *testing.T) { //nolint:paralleltest // uses t.Setenv
	homeDir := t.TempDir()
	configtest.SetConfigHome(t, homeDir)

	missingFile := filepath.Join(homeDir, ".config", "opencode", "missing.env")
	outputDir := t.TempDir()
	patterns := DefaultPatterns()

	if err := GenerateEnvExample([]string{missingFile}, patterns, outputDir); err != nil {
		t.Fatalf("GenerateEnvExample: %v", err)
	}

	examplePath := filepath.Join(outputDir, ".env.example")
	data, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("read .env.example: %v", err)
	}
	content := string(data)

	// Must emit clean human-readable note, not raw Go error string
	if !strings.Contains(content, "# [could not read source file:") {
		t.Errorf(".env.example missing clean human-readable note, got:\n%s", content)
	}

	// Must not contain raw Go error substrings or absolute paths
	if strings.Contains(content, "open ") || strings.Contains(content, "no such file or directory") {
		t.Errorf(".env.example leaked raw Go error string:\n%s", content)
	}
	if strings.Contains(content, homeDir) {
		t.Errorf(".env.example leaked absolute path %q:\n%s", homeDir, content)
	}
}

func TestScanFile_ExtendedSecretFamilies(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	patterns := DefaultPatterns()

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "AWS access key ID (AKIA)",
			content: "key = AKIAIOSFODNN7EXAMPLE\n",
			want:    1,
		},
		{
			name:    "AWS temporary key ID (ASIA)",
			content: "ASIAY34FZKBOKMUTVV7A\n",
			want:    1,
		},
		{
			name:    "Google Cloud API key",
			content: "AIzaSyD-1234567890abcdefghijklmnopqrstu\n",
			want:    1,
		},
		{
			name:    "Stripe secret live key (sk_live)",
			content: "sk_live_abcdefghij1234\n",
			want:    1,
		},
		{
			name:    "Stripe restricted test key (rk_test)",
			content: "rk_test_abcdefghij1234\n",
			want:    1,
		},
		{
			name:    "PostgreSQL connection string with credentials in JSON",
			content: `{"database_url": "postgres://myuser:mypassword@localhost:5432/mydb"}` + "\n",
			want:    1,
		},
		{
			name:    "Redis connection string with auth",
			content: "redis://admin:hunter2@cache.internal:6379/0\n",
			want:    1,
		},
		{
			name:    "MongoDB SRV connection string with auth",
			content: "mongodb+srv://u:pw@cluster0.example.net/db\n",
			want:    1,
		},
		{
			name:    "Authorization header Bearer token",
			content: "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9\n",
			want:    1,
		},
		{
			name:    "Raw Bearer token standalone",
			content: "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9\n",
			want:    1,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			dir := t.TempDir()
			fp := filepath.Join(dir, "secrets.env")
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

func TestScanFile_ExtendedSecretFamilies_FalsePositiveGuards(t *testing.T) { //nolint:paralleltest // not yet parallelized — shared state (os.Stderr/execCommand/config-file/struct) isolation pending
	patterns := DefaultPatterns()

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "Stripe publishable live key ignored",
			content: "pk_live_abcdefghij1234\n",
			want:    0,
		},
		{
			name:    "Stripe publishable test key ignored",
			content: "pk_test_abcdefghij1234\n",
			want:    0,
		},
		{
			name:    "PostgreSQL connection string without credentials ignored",
			content: "postgres://localhost:5432/mydb\n",
			want:    0,
		},
		{
			name:    "Redis connection string without credentials ignored",
			content: "redis://cache.internal:6379/0\n",
			want:    0,
		},
		{
			name:    "HTTPS URL ignored",
			content: "https://example.com/path?a=b\n",
			want:    0,
		},
		{
			name:    "Short Bearer token placeholder ignored",
			content: "Bearer token\n",
			want:    0,
		},
	}

	for _, tt := range tests { //nolint:paralleltest // subtests share table/struct state
		t.Run(tt.name, func(t *testing.T) { //nolint:paralleltest // subtests share table/struct state
			dir := t.TempDir()
			fp := filepath.Join(dir, "clean.env")
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

func TestGenerateEnvExample_DSNRedaction(t *testing.T) { //nolint:paralleltest // uses t.Setenv via configtest.SetConfigHome
	homeDir := t.TempDir()
	configtest.SetConfigHome(t, homeDir)

	configDir := filepath.Join(homeDir, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}

	configFile := filepath.Join(configDir, "config.json")
	content := `{"database_url": "postgres://myuser:mypassword@localhost:5432/mydb"}` + "\n"
	if err := os.WriteFile(configFile, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	outputDir := t.TempDir()
	patterns := DefaultPatterns()

	if err := GenerateEnvExample([]string{configFile}, patterns, outputDir); err != nil {
		t.Fatalf("GenerateEnvExample: %v", err)
	}

	examplePath := filepath.Join(outputDir, ".env.example")
	data, err := os.ReadFile(examplePath)
	if err != nil {
		t.Fatalf("read .env.example: %v", err)
	}
	outStr := string(data)

	if strings.Contains(outStr, "myuser") {
		t.Errorf(".env.example leaked username %q", "myuser")
	}
	if strings.Contains(outStr, "mypassword") {
		t.Errorf(".env.example leaked password %q", "mypassword")
	}
	if !strings.Contains(outStr, "<YOUR_SECRET>") {
		t.Errorf(".env.example missing <YOUR_SECRET> placeholder:\n%s", outStr)
	}

	found := false
	for _, line := range strings.Split(outStr, "\n") {
		if strings.Contains(line, "<YOUR_SECRET>") {
			found = true
			jsonPart := strings.TrimSpace(strings.Split(line, "  #")[0])
			var parsed map[string]string
			if err := json.Unmarshal([]byte(jsonPart), &parsed); err != nil {
				t.Fatalf("surrounding JSON is not parseable: %v (line: %s)", err, jsonPart)
			}
			if parsed["database_url"] != "<YOUR_SECRET>" {
				t.Errorf("database_url = %q, want <YOUR_SECRET>", parsed["database_url"])
			}
		}
	}
	if !found {
		t.Fatal("redacted JSON line not found in .env.example")
	}
}

// TestRedactLine_MultipleFamiliesOnOneLine guards a real leak: a single JSON
// line routinely carries more than one credential family, and redacting only
// the first match left the rest of the line in the clear.
func TestRedactLine_MultipleFamiliesOnOneLine(t *testing.T) { //nolint:paralleltest // pure function
	line := `{"db":"postgres://realuser:realsecret@db.internal:5432/app","aws":"AKIAIOSFODNN7EXAMPLE","token":"ghp_abc123def456ghi789jkl012mno345pqr678stu"}`

	got, matched := redactLine(line, DefaultPatterns())
	if !matched {
		t.Fatal("expected the line to match a secret pattern")
	}
	for _, leak := range []string{"realuser", "realsecret", "AKIAIOSFODNN7EXAMPLE", "ghp_abc123def456ghi789jkl012mno345pqr678stu"} {
		if strings.Contains(got, leak) {
			t.Errorf("redacted output still leaks %q:\n%s", leak, got)
		}
	}
	if !strings.Contains(got, "<YOUR_SECRET>") {
		t.Errorf("expected a placeholder in redacted output:\n%s", got)
	}
	// Sibling keys must survive so the user can still tell which value to
	// re-enter. The generic key/value pattern intentionally redacts from the
	// key name onward, so the redacted key itself is not expected to survive;
	// what matters is that one secret no longer eats the rest of the line.
	for _, key := range []string{`"db":`, `"aws":`} {
		if !strings.Contains(got, key) {
			t.Errorf("redaction destroyed surrounding JSON, missing %s:\n%s", key, got)
		}
	}
	if !strings.HasSuffix(strings.TrimSpace(got), "}") {
		t.Errorf("redaction destroyed the JSON object:\n%s", got)
	}
}

// TestRedactLine_NoMatchLeavesLineUnchanged verifies the no-match path.
func TestRedactLine_NoMatchLeavesLineUnchanged(t *testing.T) { //nolint:paralleltest // pure function
	line := `{"model":"lab","theme":"rose-pine"}`
	got, matched := redactLine(line, DefaultPatterns())
	if matched {
		t.Error("expected no match on a secret-free line")
	}
	if got != line {
		t.Errorf("line was modified: got %q, want %q", got, line)
	}
}

// TestRedactLine_OverlappingSpansMerged verifies overlapping matches from
// different patterns collapse into one placeholder without corrupting output.
func TestRedactLine_OverlappingSpansMerged(t *testing.T) { //nolint:paralleltest // pure function
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`SECRET`),
		regexp.MustCompile(`SECRET-[0-9]+`),
	}
	got, matched := redactLine("value=SECRET-1234 end", patterns)
	if !matched {
		t.Fatal("expected a match")
	}
	if strings.Count(got, "<YOUR_SECRET>") != 1 {
		t.Errorf("expected exactly one merged placeholder, got %q", got)
	}
	if got != "value=<YOUR_SECRET> end" {
		t.Errorf("merged redaction = %q", got)
	}
}
