# Delta for ci-consistency

## ADDED Requirements

### REQ-CI-004: GGA PR Review job
**Priority**: should

The CI pipeline MUST include a non-blocking GGA review job that runs on pull requests using `--pr-mode --diff-only` flags.

**Scenario**: PR opened or updated
- GIVEN a pull request targets `main`
- WHEN the PR is opened or new commits are pushed
- THEN CI runs `gga run --pr-mode --diff-only` as a dedicated job
- AND the job result MUST NOT block merge (warn-only via `continue-on-error: true`)

**Acceptance criteria**:
- [ ] `.github/workflows/ci.yml` contains a `gga-review` job
- [ ] Job triggers on `pull_request` event
- [ ] Job invokes `gga run --pr-mode --diff-only`
- [ ] Job is configured with `continue-on-error: true`
- [ ] Job uses the same Go version as other CI jobs (per REQ-CI-001)

**Scenario**: GGA provider unavailable in CI
- GIVEN the `gga-review` job is running
- WHEN the AI provider times out or returns an error
- THEN the job MUST NOT fail the CI pipeline (non-blocking)

**Acceptance criteria**:
- [ ] Provider timeout does not block merge
- [ ] CI logs show the GGA failure reason for debugging
