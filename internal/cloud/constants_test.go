package cloud

// Shared test constants extracted to satisfy goconst (min-occurrences 3).
// These strings appeared 3+ times across test files and were flagged by
// golangci-lint goconst. Centralising them avoids repetition and keeps
// test data consistent.

const (
	testBackupID    = "20260605-120000"
	testBackupName  = "20260605-120000.tar.gz"
	testBackupPath  = "bak-backups/20260605-120000.tar.gz"
	testEncoding    = "base64"
	testBearerToken = "Bearer valid-token"
	testOldID       = "20260101-000000"
	testDelegatedID = "delegated-id"
	testContentsDir = "/contents/bak-backups"
)
