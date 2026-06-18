# oauth-device-flow Specification

## Purpose

GitHub OAuth Device Flow (RFC 8628) for `bak login`. Replaces manual PAT paste with browser-based authorization. Hand-rolled HTTP using `net/http` + `encoding/json` (no new deps), consistent with existing `cloud/github_gist.go` pattern. `atotto/clipboard` already a transitive dep (`go.mod:21`).

## Requirements

### Requirement: Device code request

The system MUST POST to `https://github.com/login/device/code` with `client_id` and `scope: "gist"` to obtain a device code, user code, verification URI, polling interval, and expiration.

#### Scenario: Successful device code request

- GIVEN `BAK_GITHUB_OAUTH_CLIENT_ID` is set to a valid OAuth App client ID
- WHEN `bak login` is invoked
- THEN the system SHALL POST to the device code endpoint and receive `device_code`, `user_code`, `verification_uri`, `interval`, `expires_in`

#### Scenario: Missing client ID

- GIVEN `BAK_GITHUB_OAUTH_CLIENT_ID` is not set
- WHEN `bak login` is invoked
- THEN the system SHALL fall back to manual PAT paste (existing flow)

### Requirement: User code display and browser open

The system MUST display the user code and verification URI to the user. The system SHOULD auto-open the browser to the verification URI.

#### Scenario: Browser opens automatically

- GIVEN a display environment is available (`DISPLAY` is set on Linux, or macOS/Windows)
- WHEN the device code is received
- THEN the system SHALL open the browser to `verification_uri` via `xdg-open` / `open` / `cmd /c start`
- AND display "Enter code: XXXX-XXXX" on the terminal

#### Scenario: Auto-copy user code

- GIVEN a clipboard mechanism is available
- WHEN the user code is received
- THEN the user code SHALL be copied to the clipboard via `atotto/clipboard`

#### Scenario: Headless fallback

- GIVEN no display environment (`DISPLAY=""` on Linux)
- WHEN the device code is received
- THEN the system SHALL print the verification URI and user code to stdout
- AND skip the browser open step without error

### Requirement: Token polling

The system MUST poll `https://github.com/login/oauth/access_token` at the server-specified interval until the user authorizes, the code expires, or an unrecoverable error occurs.

#### Scenario: User authorizes in browser

- GIVEN the system is polling at the specified interval
- WHEN the user completes authorization in the browser
- THEN the poll SHALL receive `access_token` and the login SHALL succeed

#### Scenario: Slow authorization

- GIVEN the polling interval is 5 seconds and expiration is 900 seconds
- WHEN the user takes 60 seconds to authorize
- THEN the system SHALL continue polling and display "Waiting for authorization..."

#### Scenario: Code expires

- GIVEN the expiration time has elapsed
- WHEN the next poll returns `error: expired_token`
- THEN the system SHALL display "Code expired. Run 'bak login' again." and exit with error

#### Scenario: Authorization denied

- WHEN the poll returns `error: access_denied`
- THEN the system SHALL display "Authorization denied." and exit with error

### Requirement: Token storage

The system MUST store the obtained OAuth token in the same location as the current PAT (`~/.config/bak/config.json` under `github.token`) and validate it via `cloud.ValidateToken`.

#### Scenario: Token saved after OAuth

- GIVEN the OAuth flow completes successfully
- WHEN the access token is received
- THEN the token SHALL be validated via `cloud.ValidateToken`
- AND saved to `config.json` under `github.token`
