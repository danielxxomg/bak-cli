# Delta: lint-fixes

## MODIFIED Requirements

### Requirement: All-lint-green
The system MUST pass `golangci-lint run` with exit code 0 and zero warnings on all platforms.

(Previously: `golangci-lint run` exited 1 with 4 issues: 2 `gocritic ifElseChain` and 2 `goimports` violations.)

#### Scenario: ifElseChain in profiles.go resolved

- GIVEN `internal/tui/screens/profiles.go:193` contains an if-else chain
- WHEN the code is refactored
- THEN the if-else chain MUST be replaced with a `switch` statement
- AND `gocritic` MUST NOT flag the file

#### Scenario: ifElseChain in restore.go resolved

- GIVEN `internal/tui/screens/restore.go:226` contains an if-else chain
- WHEN the code is refactored
- THEN the if-else chain MUST be replaced with a `switch` statement
- AND `gocritic` MUST NOT flag the file

#### Scenario: goimports passes on login_test.go

- GIVEN `internal/actions/login_test.go:61` has import formatting issues
- WHEN `goimports` runs
- THEN imports MUST be properly grouped and sorted
- AND the file MUST pass `goimports` check

#### Scenario: goimports passes on oauth_device_test.go

- GIVEN `internal/cloud/oauth_device_test.go:136` has import formatting issues
- WHEN `goimports` runs
- THEN imports MUST be properly grouped and sorted
- AND the file MUST pass `goimports` check

#### Scenario: Full lint suite green

- GIVEN all 4 lint issues are fixed
- WHEN `golangci-lint run` executes
- THEN it MUST exit 0 with zero issues reported
