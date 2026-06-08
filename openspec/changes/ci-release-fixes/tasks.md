# Tasks: ci-release-fixes

## Phase 1: Fix CI Go Version

- [x] **1.1** Update `security` job Go version: change `go-version: '1.24'` → `'1.25'` in `.github/workflows/ci.yml` line ~111
- [x] **1.2** Update `goreleaser` job Go version: change `go-version: '1.24'` → `'1.25'` in `.github/workflows/ci.yml` line ~140
- [x] **1.3** Update `build` job Go version: change `go-version: '1.24'` → `'1.25'` in `.github/workflows/ci.yml` line ~164

## Phase 2: Fix Build Verification Steps

- [x] **2.1** Fix Unix verify step: change `./bak.exe version` → `./bak version` in `.github/workflows/ci.yml` line ~176
- [x] **2.2** Verify Windows step is correct: `.\bak.exe version` (no change needed, confirm only)

## Phase 3: Fix Taskfile Binary Name

- [x] **3.1** Replace `BINARY: bak.exe` with OS-conditional expression in `Taskfile.yml` line 5:
  ```yaml
  BINARY:
    sh: echo '{{if eq .OS "windows"}}bak.exe{{else}}bak{{end}}'
  ```

## Phase 4: Verification

- [x] **4.1** Grep `ci.yml` for `1.24` — must return zero matches
- [x] **4.2** Grep `ci.yml` for `bak.exe` in Unix context — must only appear in Windows-conditional steps
- [x] **4.3** Run `task build` locally — verify binary name matches host OS

---

## Workload Forecast

| Metric | Value |
|--------|-------|
| Estimated changed lines | ~10-15 (3 version bumps + 1 verify fix + 1 Taskfile var) |
| Files changed | 2 (`ci.yml`, `Taskfile.yml`) |
| Decision needed before apply | No |
| Chained PRs recommended | No |
| 400-line budget risk | Low |

**Delivery**: Single PR. Well under 400-line budget. No splitting needed.
