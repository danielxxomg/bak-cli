# Delta: oauth-clipboard

## MODIFIED Requirements

### Requirement: Auto-copy user code
The system MUST copy the OAuth user code to the clipboard when `atotto/clipboard` is available. Clipboard failure MUST NOT block the login flow.

(Previously: `cmd/login.go:77-81` created a `DeviceClient` without injecting the `Clipboard` function. `atotto/clipboard` was never imported. The clipboard field remained nil, so copy was silently skipped.)

#### Scenario: Clipboard function injected in cmd/login.go

- GIVEN `BAK_GITHUB_OAUTH_CLIENT_ID` is set
- WHEN `runLoginWithDeps` creates the `DeviceClient`
- THEN `Clipboard` MUST be set to `clipboard.WriteAll` from `atotto/clipboard`
- AND the `atotto/clipboard` package MUST be imported in `cmd/login.go`

#### Scenario: User code auto-copied to clipboard

- GIVEN the OAuth device flow receives a user code
- WHEN `RequestToken` displays the code
- THEN the code MUST be copied to the system clipboard via the injected `Clipboard` function

#### Scenario: Clipboard failure is graceful

- GIVEN the clipboard copy fails (e.g., no display server, Wayland restriction)
- WHEN `RequestToken` attempts to copy
- THEN the error MUST be logged to stderr
- AND the flow MUST continue (user code still displayed on screen)
- AND the login MUST NOT be blocked
