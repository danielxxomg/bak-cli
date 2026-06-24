# Delta for internal/config

> NEW spec for the `internal/config` package. No behavior change — this
> spec captures the test surface that must be protected/verified to
> raise coverage from 70.6% to ≥85%. Requirements describe behavior
> that MUST be exercised by tests.

## ADDED Requirements

### Requirement: settings-key helpers are tested

The package MUST have table-driven tests covering `getSettingsField`,
`setSettingsField`, and `parseBool` for all documented key aliases and
edge cases (unknown key, non-bool string, empty value).

#### Scenario: getSettingsField resolves all known aliases

- GIVEN each documented settings key alias
- WHEN `getSettingsField` is called with that alias
- THEN it returns the canonical JSON key

#### Scenario: setSettingsField writes through to the JSON blob

- GIVEN a settings JSON blob and a valid key
- WHEN `setSettingsField` is called with a new value
- THEN the blob contains the new value at the canonical key

#### Scenario: parseBool accepts yes/no and rejects non-boolean strings

- GIVEN the strings `"yes"` and `"no"` (case-insensitive)
- WHEN `parseBool` is called
- THEN `"yes"` returns `(true, nil)` and `"no"` returns `(false, nil)`
- AND strings like `"2"` or `""` return an error

### Requirement: splitWildcard is tested

`splitWildcard` MUST have table-driven tests for: no wildcard, trailing
`*`, leading `*`, embedded `*`, empty input.

#### Scenario: trailing wildcard splits correctly

- GIVEN `"skills/*"`
- WHEN `splitWildcard` is called
- THEN it returns `["skills/", ""]` (splits AT `*`, dropping the star)

#### Scenario: no wildcard returns single-element slice

- GIVEN `"agents"`
- WHEN `splitWildcard` is called
- THEN it returns `["agents"]` (single element, no split)

### Requirement: matchSegment wildcard branches are tested

`matchSegment` MUST have table-driven tests covering: exact match,
wildcard match-all, mismatch, empty segment.

#### Scenario: wildcard matches any segment

- GIVEN pattern segment `"*"` and any input segment
- WHEN `matchSegment` is called
- THEN it returns true

#### Scenario: mismatch returns false

- GIVEN pattern segment `"foo"` and input `"bar"`
- WHEN `matchSegment` is called
- THEN it returns false

### Requirement: Load error paths are tested

`Load` MUST be tested for: missing file (returns zero value, no error),
malformed JSON (returns error), unreadable file (returns error).

#### Scenario: missing config file returns zero value

- GIVEN a config home with no config file
- WHEN `Load` is called
- THEN it returns the zero `Config` and no error

#### Scenario: malformed JSON returns error

- GIVEN a config file containing invalid JSON
- WHEN `Load` is called
- THEN it returns a non-nil error

### Requirement: Save error paths are tested

`Save` MUST be tested for: unwritable directory (returns error), happy
path (writes file atomically).

#### Scenario: unwritable directory returns error

- GIVEN a config home pointing at a read-only directory
- WHEN `Save` is called
- THEN it returns a non-nil error

### Requirement: Get/Set error paths are tested

`Get` and `Set` MUST be tested for: unknown key (returns error), valid
key round-trip.

#### Scenario: unknown key returns error

- GIVEN a valid config
- WHEN `Get("nonexistent.key")` is called
- THEN it returns an error

### Requirement: coverage target for internal/config

The `internal/config` package MUST achieve ≥85% statement coverage
after the change.

#### Scenario: coverage gate passes

- WHEN `go test -cover ./internal/config/...` is run
- THEN total coverage is ≥85%
