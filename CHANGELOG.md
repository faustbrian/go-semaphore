# Changelog

## Unreleased

### Changed

- Replace copied repository tooling with the pinned `go-library-tools` v1.0.13
  contract while retaining package-owned policy and verification evidence.

### Documentation

- Replace archived monorepo links and completed execution artifacts with a
  standalone, human-oriented documentation structure.

## 1.0.0 - 2026-08-25

### Changed

- Exclude intentional nested modules from root local-proxy archives so local,
  bootstrap, CI, and public module checksums describe the same source
  boundary.

- Track the pinned documentation-tool lockfile so clean CI checkouts install
  the exact validated cspell dependency.

- Reconcile standalone dependency checksums against deterministic current
  module archives so CI, local verification, and release consumers resolve
  identical content.

- Harden standalone documentation validation with deterministic spelling and
  link checks, package-specific documentation gates, and repository-local
  contributor guidance.

### Changed

- Publish the module from its standalone `github.com/faustbrian/go-semaphore` identity while preserving its documented API and behavior.

### Documentation

- Link the package README to package-owned documentation.

### Added

- Process-local positive weighted capacity with strict FIFO admission and
  explicitly bounded waiting.
- Context-aware acquisition, fair immediate attempts, typed saturation and
  lifecycle errors, and deterministic cancellation removal.
- Stable owned permits with concurrent duplicate-release protection.
- Idempotent close, queued-waiter rejection, and context-bounded drain.
- Immutable snapshots and bounded observer events outside synchronization.
- Result-, error-, and panic-preserving execution helpers.
- Kubernetes scope, migration, operations, security, API, FAQ, and performance
  guidance.
- Race, fuzz, conservation, lifecycle, benchmark, coverage, mutation, API,
  documentation, and clean-consumer gate definitions.
- Generated concurrent reference histories, deterministic cancellation and
  shutdown races, queue-node and permit-retention checks, and source-owned
  goroutine/timer/finalizer guards.
- Equivalent benchmark dimensions for strict FIFO handoff, cancellation queue
  depth, mixed weights, observer overhead, x/sync v0.22.0, and the actively
  released kit4go v0.9.0 semaphore, with semantic differences disclosed.
