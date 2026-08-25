# Changelog

## Unreleased

### Changed

- Track the pinned documentation-tool lockfile so clean CI checkouts install
  the exact validated cspell dependency.

- Reconcile standalone dependency checksums against deterministic current
  module archives so CI, local verification, and release consumers resolve
  identical content.

- Harden standalone documentation validation with deterministic spelling and
  link checks, package-specific documentation gates, and repository-local
  contributor guidance.

## 1.0.0 - 2026-08-25

### Changed

- Publish the module from its standalone `github.com/faustbrian/go-semaphore` identity while preserving its documented API and behavior.

### Documentation

- Link the package README to the repository-wide Golib documentation portal.

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
