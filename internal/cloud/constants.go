package cloud

// Shared string constants extracted to satisfy goconst (min-occurrences 3).
// These strings appeared 3+ times across production code and were flagged
// by golangci-lint goconst.

const (
	codebergName    = "codeberg"
	gistBackupFile  = "backup.tar.gz"
	gistsEndpoint   = "/gists"
	acceptJSON      = "application/json"
	acceptGitHub    = "application/vnd.github+json"
	errorKey        = "error"
	errExpiredToken = "expired_token"

	githubTokenEnv     = "GITHUB_TOKEN"
	githubTokenKey     = "github.token"
	providerGitea      = "gitea"
	providerGithubGist = "github-gist"
	providerGithubRepo = "github-repo"
)
