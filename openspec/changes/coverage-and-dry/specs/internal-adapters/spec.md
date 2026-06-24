# Delta for internal/adapters

> NEW spec for the `internal/adapters` package. Extends the existing
> `generic-adapter` spec with multi-category root-file support, and
> converts `internal/adapters/opencode` into a kilocode-style delegating
> wrapper. Folds in four latent bugfixes discovered during DRY review.

## ADDED Requirements

### Requirement: GenericAdapter multi-category root-file scan

`GenericAdapter` MUST support an optional `RootConfigFiles map[string]string`
field mapping file name → category. When non-nil, `ListItems` MUST call
a single `scanRootFiles(configDir, catSet, opts)` once (not per category)
and include each discovered file iff its mapped category is in `catSet`.
Each returned `Item` MUST have `Category` set to the mapped category
(not a fixed default).

#### Scenario: file included when its category is requested

- GIVEN a `GenericAdapter` with `RootConfigFiles = {"mcp.json": "mcp", "opencode.json": "config"}`
- AND a file `mcp.json` exists at the config root
- WHEN `ListItems(home, ["mcp"])` is called
- THEN the returned slice contains exactly one `Item` with `Category="mcp"` and `RelPath="mcp.json"`

#### Scenario: file excluded when its category is not requested

- GIVEN the same adapter and files as above
- WHEN `ListItems(home, ["config"])` is called
- THEN `mcp.json` is NOT present in the returned slice

#### Scenario: root scan runs once for multiple matching categories

- GIVEN `RootConfigFiles` with entries in both "config" and "mcp"
- WHEN `ListItems(home, ["config", "mcp"])` is called
- THEN `scanRootFiles` is invoked exactly once (not twice)

### Requirement: MaxFileSize applies to root files

`scanRootFiles` MUST honor `opts.MaxFileSize` for every root file. Files
exceeding the limit MUST be skipped and a warning written to stderr.

#### Scenario: oversized root file skipped with warning

- GIVEN `opts.MaxFileSize = 100` and a root file of 200 bytes
- WHEN `ListItems` is called
- THEN the file is absent from results AND a warning is written to stderr

### Requirement: opencode delegates to GenericAdapter

`internal/adapters/opencode.Adapter` MUST become a thin delegating
wrapper around a package-level `adapters.GenericAdapter` (kilocode
pattern). All interface methods (`Name`, `Detect`, `ListItems`, `Backup`,
`Restore`, `SetScanOptions`) MUST forward to the base. The package MUST
NOT contain its own `scanDir`/`scanRootFiles` implementations.

#### Scenario: opencode ListItems produces identical output before and after refactor

- GIVEN an opencode config tree with skills/, commands/, agent/, plugins/, and root files (opencode.jsonc, mcp.json, AGENTS.md)
- WHEN `ListItems(home, ["config","mcp","skills","commands","agents","plugins"])` is called
- THEN the returned items match the pre-refactor output (same set of RelPaths, Categories, hashes, and sizes)

### Requirement: mcp.json preserved as distinct category

`mcp.json` at the config root MUST always be reported with `Category="mcp"`
(never merged into "config"). This is a hard preservation requirement —
refactors MUST NOT change the category assignment.

#### Scenario: mcp.json backed up with Category=mcp

- GIVEN an `mcp.json` at the config root
- WHEN `ListItems(home, ["mcp"])` is called
- THEN the manifest contains an `Item` with `Category="mcp"` and `RelPath="mcp.json"`

### Requirement: bare errors wrapped with %w

All errors returned from internal helpers MUST be wrapped with
`fmt.Errorf("context: %w", err)` — bare `errors.New` or unwrapped
returns are forbidden. Context strings MUST start with lowercase.

#### Scenario: scanDir wraps stat errors with context

- GIVEN a directory path that cannot be stat'd
- WHEN `scanDir` is invoked
- THEN the returned error wraps the underlying error and starts with a lowercase context prefix

### Requirement: scanDir continues on stderr write failure

When `os.Stderr.WriteString` fails while emitting a MaxFileSize warning,
`scanDir` MUST log the write error to verbose output and continue
scanning — it MUST NOT abort the walk or return the write error.

#### Scenario: stderr write failure does not abort scan

- GIVEN a file exceeding `MaxFileSize` AND a stderr writer that returns an error
- WHEN `scanDir` walks past the file
- THEN the file is skipped AND the walk continues AND subsequent files are still reported

### Requirement: unused homeDir parameter removed from scanDir

`scanDir` signature MUST NOT include an unused `homeDir` parameter.

#### Scenario: scanDir compiles without homeDir

- GIVEN the refactored signature
- WHEN the package builds
- THEN `scanDir` accepts `(dir, category, configDir string, opts ScanOptions)` (or equivalent without `homeDir`)

## Verification Requirements

### Requirement: coverage target for internal/adapters

The `internal/adapters` package (including all sub-packages) MUST achieve
≥85% statement coverage after the change.

#### Scenario: coverage gate passes

- WHEN `go test -cover ./internal/adapters/...` is run
- THEN total coverage is ≥85%
