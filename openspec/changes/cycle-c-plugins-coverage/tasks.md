# Tasks: Cycle C — Config-Driven Plugins & Test Coverage

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 1500-2000 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | PR 1 (Foundation) → PR 2 (Actions) → PR 3 (Integration) |
| Delivery strategy | ask-on-risk |
| Chain strategy | stacked-to-main |

Decision needed before apply: Yes
Chained PRs recommended: Yes
Chain strategy: stacked-to-main
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Notes |
|------|------|-----------|-------|
| 1 | YAML schemas, loaders, DI interfaces, mocks | PR 1 | main; ~400 lines; additive only |
| 2 | Extract 4 action structs with tests | PR 2 | main; ~600 lines; depends on PR 1 |
| 3 | Wire cmd/ + YAML integration + coverage | PR 3 | main; ~500 lines; depends on PR 2 |

## Phase 1: Foundation (Additive Infrastructure)

- [x] 1.1 Add `gopkg.in/yaml.v3` to `go.mod` and run `go mod tidy`
- [x] 1.2 Create `internal/presets/schema.go` with `YAMLPreset`, `YAMLMetadata` types
- [x] 1.3 Create `internal/presets/loader.go` with `LoadFromDir(dir string) ([]YAMLPreset, error)` — scan directory, parse YAML, validate required fields
- [x] 1.4 Create `internal/presets/loader_test.go` — table-driven tests: valid YAML, missing categories, missing directory, invalid syntax, empty dir, multiple files, non-yaml ignored (8 test functions)
- [x] 1.5 Create `internal/adapters/schema.go` with `YAMLAdapter`, `YAMLCategoryPattern` types
- [x] 1.6 Create `internal/adapters/yaml.go` with `ConfigAdapter` struct implementing `Adapter` interface (Name, Detect, ListItems, Backup, Restore methods)
- [x] 1.7 Add `LoadYAMLAdapters(dir string) ([]*ConfigAdapter, error)` to `yaml.go` — scan, parse, validate
- [x] 1.8 Create `internal/adapters/yaml_test.go` — table-driven tests: valid adapter, missing name, missing config_path, invalid syntax, empty dir, multiple files, non-yaml ignored, Detect, ListItems (12 test functions)
- [x] 1.9 Add `RegisterOrReplace(adapter Adapter, override bool)` to `internal/adapters/registry.go` — handle conflict resolution with warning
- [x] 1.10 Create `internal/actions/interfaces.go` with `FileSystem` interface (9 methods: UserHomeDir, Stat, ReadDir, ReadFile, MkdirAll, CopyFile, RemoveAll, WalkDir, WriteFile) and `ConfigLoader` interface
- [x] 1.11 Create `internal/actions/os_impl.go` with `OSFileSystem` and `RealConfigLoader` — real OS implementations wrapping stdlib calls
- [x] 1.12 Create `internal/actions/mock_impl.go` with `MockFileSystem` and `MockConfigLoader` — configurable mock implementations for tests

## Phase 2: Action Extraction (Core Implementation)

- [x] 2.1 Create `internal/actions/backup.go` with `BackupAction` struct (FS, Config, Registry, Presets, flags) and `Run(cmd, args) error` method — copy logic from `cmd/backup.go`, replace OS calls with `a.FS.*` and `a.Config.*`
- [x] 2.2 Create `internal/actions/backup_test.go` — table-driven tests with mocks: happy path, unknown preset, stat error, mkdir error, copy error, secret detected (10+ scenarios)
- [x] 2.3 Create `internal/actions/restore.go` with `RestoreAction` struct and `Run()` method — copy logic from `cmd/restore.go`, use injected dependencies
- [x] 2.4 Create `internal/actions/restore_test.go` — table-driven tests: happy path, missing manifest, checksum mismatch, dry-run diff, git error (10+ scenarios)
- [x] 2.5 Create `internal/actions/push.go` with `PushAction` struct and `Run()` method — copy logic from `cmd/push.go`, use injected dependencies
- [x] 2.6 Create `internal/actions/push_test.go` — table-driven tests: happy path, config load error, auth error, pack error, provider error (8+ scenarios)
- [x] 2.7 Create `internal/actions/pull.go` with `PullAction` struct and `Run()` method — copy logic from `cmd/pull.go`, use injected dependencies
- [x] 2.8 Create `internal/actions/pull_test.go` — table-driven tests: happy path, auth error, unpack error, restore error (8+ scenarios)

## Phase 3: Wire Actions (Integration)

- [x] 3.1 Modify `cmd/backup.go` — replace `RunE` body with: parse flags → create `OSFileSystem`, `RealConfigLoader` → build `BackupAction` → call `action.Run(cmd, args)`
- [x] 3.2 Modify `cmd/restore.go` — thin wire to `RestoreAction`
- [x] 3.3 Modify `cmd/push.go` — thin wire to `PushAction`
- [x] 3.4 Modify `cmd/pull.go` — thin wire to `PullAction`
- [ ] 3.5 Run `go test ./cmd/... -cover` — verify cmd/ coverage ≥80%
- [ ] 3.6 Run `go test ./internal/presets/... -cover` — verify presets/ coverage ≥95%
- [ ] 3.7 Run `go test ./internal/adapters/... -cover` — verify adapters/ coverage ≥90%
- [ ] 3.8 Run `go test ./...` — verify all existing tests pass
- [ ] 3.9 Run `go vet ./...` and `golangci-lint run` — verify clean

## Phase 4: YAML Integration (Feature Wiring)

- [x] 4.1 Modify `internal/presets/presets.go` — add `ResolveAll(presetName string, override bool) ([]string, error)` that calls `LoadYAMLPresets()`, merges with built-ins, then calls `Resolve()`
- [x] 4.2 Update `cmd/backup.go` and `cmd/restore.go` — call `presets.ResolveAll(preset, override)` instead of `presets.Resolve(preset)`
- [x] 4.3 Add `--override` flag to `bak backup` and `bak restore` commands in `cmd/backup.go` and `cmd/restore.go`
- [x] 4.4 Modify `internal/adapters/register/register.go` — add `LoadYAMLAdapters(reg *Registry, override bool)` call after built-in registration
- [x] 4.5 Update `cmd/backup.go` and `cmd/restore.go` — call `register.LoadYAMLAdapters(reg, override)` after `register.All(reg)`
- [x] 4.6 Create `examples/presets/custom.yaml` — example YAML preset with comments
- [x] 4.7 Create `examples/adapters/myapp.yaml` — example YAML adapter with comments
- [x] 4.8 Update `README.md` — add "Custom Presets" section explaining YAML schema, directory location, merge behavior, `--override` flag
- [x] 4.9 Update `README.md` — add "Custom Adapters" section explaining YAML schema, categories, patterns
- [ ] 4.10 Run full test suite — verify all scenarios from specs pass (14 scenarios across 3 domains)

## Phase 5: Cleanup & Verification

- [ ] 5.1 Remove any temporary debug code or TODO comments
- [ ] 5.2 Verify godoc comments on all exported types and functions (per AGENTS.md rules)
- [ ] 5.3 Run `go test -coverprofile=coverage.out ./...` and `go tool cover -func=coverage.out` — generate final coverage report
- [ ] 5.4 Verify success criteria: `bak backup --preset my-custom` works, YAML adapter detected, coverage targets met, all tests pass, linting clean
