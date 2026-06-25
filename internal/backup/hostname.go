package backup

import (
	"fmt"
	"io"
	"os"
)

// osHostname is the os.Hostname fallback used when the injected hostname
// function is nil. It is a package-level variable so tests can inject a
// failing implementation (AGENTS.md: inject OS calls via variables for
// testability).
var osHostname = os.Hostname

// ResolveHostname returns the hostname via fn, falling back to os.Hostname
// when fn is nil. On error it returns "unknown" and, when verbose is set,
// writes a warning to errOut. It is the canonical hostname resolver shared
// by the backup workflow (internal/backup) and the out-of-workflow push
// caller (internal/actions), consolidating the previously duplicated
// resolution logic.
//
// ResolveHostname lives in internal/backup (the leaf package) because
// internal/backup cannot import internal/actions without a circular
// dependency.
func ResolveHostname(fn func() (string, error), verbose bool, errOut io.Writer) string {
	hostnameFn := fn
	if hostnameFn == nil {
		hostnameFn = osHostname
	}
	hostname, err := hostnameFn()
	if err != nil {
		if verbose && errOut != nil {
			fmt.Fprintf(errOut, "warning: could not get hostname: %v\n", err) //nolint:errcheck // non-critical diagnostic
		}
		return "unknown"
	}
	return hostname
}
