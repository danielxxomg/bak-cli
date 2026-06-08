# Tasks: README Refresh

## Forecast

- **Estimated lines changed**: ~120 (additions + deletions in README.md only)
- **Decision needed before apply**: No
- **Chained PRs recommended**: No
- **400-line budget risk**: Low

---

## Phase 1: Header + Badges

- [x] **1.1** Expand badges from 3 to 6+: add Go version (`go 1.25+`), Platform (`macOS | Linux | Windows`), and CI/Tests badge
- [x] **1.2** Verify all badge URLs resolve correctly

## Phase 2: Platform Table

- [x] **2.1** Add "Supported Platforms" section after badges with table: Platform | Install Method | Package
- [x] **2.2** Cover macOS (Homebrew), Linux (Homebrew, deb, rpm), Windows (Scoop)

## Phase 3: Installation Restructure

- [x] **3.1** Split Installation into recommended (Homebrew for macOS/Linux, Scoop for Windows) with "Recommended" labels
- [x] **3.2** Move Debian/Ubuntu, RHEL/Fedora, Go install, and From Source into `<details>` block with summary "Alternative install methods"

## Phase 4: Contributing Condensation

- [x] **4.1** Replace 5-step Contributing list with ≤3 lines + link to CONTRIBUTING.md
- [x] **4.2** Move adapter interface code example to CONTRIBUTING.md (already present — CONTRIBUTING.md lines 186-301 contain complete adapter implementation guide)

## Phase 5: Next Steps Section

- [x] **5.1** Add "Next Steps" section after Contributing with 4 items: `bak wizard`, `bak schedule`, custom presets, custom adapters
- [x] **5.2** Each item links to relevant docs or shows command example

## Phase 6: Brand Assets Collapsible

- [x] **6.1** Wrap Brand Assets section in `<details><summary>Brand Assets</summary>...</details>`

## Phase 7: Verification

- [x] **7.1** Visual check: confirm all sections render correctly on GitHub (badges, tables, collapsibles)
- [x] **7.2** Confirm all existing content preserved — diff against original shows no deletions, only moves
- [x] **7.3** Confirm CONTRIBUTING.md link resolves
- [x] **7.4** Confirm no broken markdown (tables aligned, details tags closed)
