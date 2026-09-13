---
---

# ADR-023: Single Integration Pipeline

## Status
Accepted

Related: [ADR-004](ADR-004-auto-release-and-versioning.md) (the auto-tag
release path this pipeline follows).

## Date
2026-09-13

## Context

Integration tests were split across eight workflows: seven per-distro files
(`test-integration-<distro>.yml`), each triggered by `workflow_run` on
`Auto Tag & Release` and downloading the published release binary, plus a
dispatch-only `test-integration-branch.yml` that built from an arbitrary ref.

This had three problems:

- **Maintenance duplication** — seven near-identical files, one per distro.
- **Release coupling and a race** — the per-distro runs downloaded the release
  that `Auto Tag & Release` had just created via goreleaser; the download could
  race the release publication.
- **No PR coverage** — the suite only ran after a release, so integration
  regressions were caught after merge to `main`, not on the pull request.

## Decision

Replace all eight workflows with one matrixed workflow,
`.github/workflows/integration.yml` (`Integration Tests`).

- **Triggers:** `workflow_run` on `["CI", "Auto Tag & Release"]`
  (`types: [completed]`) and a manual `workflow_dispatch` with a `ref` input.
- **Routing:**
  - `Auto Tag & Release` -> build `main`.
  - `CI` whose upstream event is `pull_request` -> build
    `workflow_run.head_sha` (the pull-request head).
  - `workflow_dispatch` -> build `inputs.ref`.
- **Gating:** the job runs only when the upstream workflow concluded
  `success`, the upstream is a same-repository run
  (`head_repository.full_name == github.repository`, i.e. no forks), and the
  upstream is either `Auto Tag & Release` or a `pull_request`-event `CI` run.
  Gating on the whole `CI` workflow means lint, unit tests, commit-lint,
  security, and SonarCloud have all passed.
- **Coverage:** one matrix over
  `[ubuntu, debian, fedora, centos, rocky, arch, opensuse]`, each building the
  same `test/Dockerfile.<distro>` and running the corresponding
  `test/integration/<distro>.sh`.
- **Build source:** `go build` from the routed ref, not a release artifact.

Because `workflow_run` workflows execute in the default-branch context and can
access secrets, the job pins the checked-out ref, disables persisted checkout
credentials (`persist-credentials: false`), and disables the Go module cache
(`cache: false`) to avoid cache poisoning from untrusted refs.

## Alternatives Considered

### Keep per-distro workflows, add a PR trigger to each
- **Cons:** seven files to keep in sync, seven `workflow_run` entries, and the
  release-download race remains. Rejected.

### Put integration into `ci.yml` as a job with `needs: [sonarcloud]`
- **Pros:** native job dependency, no `workflow_run`.
- **Cons:** the integration suite must also run after `Auto Tag & Release`,
  which is a different workflow; mixing both trigger shapes and a conditional
  `needs` in one file is brittle. Rejected.

### Continue downloading the release binary after `Auto Tag & Release`
- **Cons:** does not test current `main` code and keeps the publication race.
  Rejected.

### Distinguish release-chained vs PR-chained runs in the README badge
- **Cons:** not possible — every `workflow_run` run lives on the default
  branch, so branch/event badge filters cannot separate them. Rejected; the
  badge reflects the latest integration result.

## Consequences

- **Positive:** one file, one matrix, full parity with the seven per-distro
  scripts; PR heads are integration-tested before merge; the release-download
  race is gone.
- **Positive:** the manual `workflow_dispatch` + `ref` capability of
  `test-integration-branch.yml` is preserved.
- **Positive:** the concurrency group is scoped by workflow, upstream
  workflow, source repository, and branch. A fork whose branch name matches a
  trusted branch cannot cancel an in-flight `main` run, and the sibling
  `CI`-push and `Auto Tag & Release` `workflow_run` chains on the same merge
  occupy separate lanes instead of cancelling each other.
- **Residual:** the published release tarball is no longer integration-tested
  directly; the pipeline tests the `main` commit that produced it.
- **Residual:** the full seven-distro Docker matrix runs on every qualifying
  pull request, which is network- and CPU-heavy.
- **Residual:** `workflow_run` only fires for workflows present on the default
  branch, so the pull-request integration path activates only after this
  workflow is merged to `main`.
