package cloud

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
)

// execCommand is the function used to run external commands.
// Overridable for tests.
var execCommand = exec.Command

// openBrowserOS opens the given URL in the user's default browser.
// Platform detection uses runtime.GOOS:
//
//   - linux: xdg-open (requires DISPLAY env var)
//   - darwin: open
//   - windows: rundll32 url.dll,FileProtocolHandler
//
// When no display is available on Linux (DISPLAY is empty), an error
// is returned so callers can fall back to printing the URL.
//
// This is a package-level variable so callers (e.g., DeviceClient)
// can override it for injection.
var openBrowserOS = func(url string) error {
	switch runtime.GOOS {
	case "linux":
		if os.Getenv("DISPLAY") == "" {
			return fmt.Errorf("no display available — set DISPLAY or open this URL manually")
		}
		return execCommand("xdg-open", url).Start()
	case "darwin":
		return execCommand("open", url).Start()
	case "windows":
		return execCommand("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
}

// OpenBrowser opens the given URL in the user's default browser.
// This is a thin wrapper around openBrowserOS for users who want a
// clean function reference without the var indirection.
func OpenBrowser(url string) error {
	return openBrowserOS(url)
}
