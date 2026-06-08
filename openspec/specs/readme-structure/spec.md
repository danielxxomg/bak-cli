# Spec: README Structure

## ADDED Requirements

### Requirement: Badge Completeness

The README badges section MUST display at minimum: Release version, License, Go Report Card, Go version (1.25+), Platform support (macOS | Linux | Windows), and Test status.

#### Scenario: Badge section rendered

- GIVEN a user views README.md on GitHub
- WHEN the badge row is rendered
- THEN it MUST contain ≥6 badge images
- AND MUST include a Go version badge showing `go 1.25+`
- AND MUST include a platform badge showing supported OS targets
- AND MUST include a test/CI status badge

#### Scenario: Badge links are valid

- GIVEN any badge in the README
- WHEN the badge image link is followed
- THEN it MUST resolve to a valid URL (no 404s)

### Requirement: Installation Hierarchy

The Installation section MUST present a recommended method prominently, with alternative methods in a collapsible `<details>` block.

#### Scenario: Default install visibility

- GIVEN a user reads the Installation section
- WHEN scanning for the quickest install path
- THEN the recommended method (Homebrew for macOS/Linux, Scoop for Windows) MUST be visible without expanding any collapsible section
- AND MUST be clearly labeled as "Recommended"

#### Scenario: Alternative methods collapsed

- GIVEN the Installation section
- WHEN alternative install methods are listed (From Source, Go install, deb, rpm)
- THEN they MUST be inside a `<details>` block with summary "Alternative install methods"

### Requirement: Platform Support Table

The README MUST include a table showing supported platforms and their install methods.

#### Scenario: Platform table present

- GIVEN a user views the README
- WHEN they look for platform compatibility info
- THEN a table MUST be present with columns: Platform, Install Method, Package Format
- AND MUST cover macOS (Homebrew), Linux (Homebrew, deb, rpm), and Windows (Scoop)

### Requirement: Contributing Link

The Contributing section MUST be condensed to a brief invitation (≤5 lines) with a link to CONTRIBUTING.md for the full guide.

#### Scenario: Contributing section brevity

- GIVEN a user reads the Contributing section
- WHEN they want to contribute
- THEN the section MUST contain ≤5 lines of inline text
- AND MUST include a markdown link to `CONTRIBUTING.md`
- AND the adapter interface example MUST move to CONTRIBUTING.md or remain as a brief snippet

### Requirement: Next Steps Section

The README MUST include a "Next Steps" section after Contributing with actionable items for users who completed their first backup.

#### Scenario: Next Steps section exists

- GIVEN a user finished reading the README
- WHEN they want to explore further
- THEN a "Next Steps" section MUST exist with ≥3 items
- AND items MUST link to relevant docs or commands (e.g., `bak wizard`, `bak schedule`, custom presets)

### Requirement: Collapsible Brand Assets

The Brand Assets section MUST be wrapped in a `<details>` block to reduce visual noise.

#### Scenario: Brand Assets collapsed by default

- GIVEN a user views the README
- WHEN the Brand Assets section is rendered
- THEN it MUST be inside `<details>` (collapsed by default)
- AND the summary text MUST read "Brand Assets"
