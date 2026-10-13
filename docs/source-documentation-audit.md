# Source documentation audit

[Documentation map](index.md) · [Repository README](../README.md)
[Next: API reference](api.md)

Source baseline: `c0971755e021f868c0e5b7b69092c6d4f3803736` (remote main).
Scope: root module `github.com/faustbrian/go-semaphore/v2`, root package
`semaphore` and its internal/external test files. Reproduce the file inventory
with `git ls-files '*.go'`; inspect the named baseline with `git show`.
This is a source/API audit, not runtime requalification or adoption guidance.

## Acceptance, ownership and checks

Preserve behavior. Document process-local weighted FIFO admission, caller-owned
permits, queue/event bounds, cancellation linearization and drain generations.
Callers are all public methods, Execute/Run, examples and external consumers;
no public shape or executable tokens change. Named risks are misleading shutdown,
copying, fairness and observer guarantees. Tier A checks: parsed Go/token
comparison against baseline, gofmt, rendered `go doc -all`, useful final-diff
inspection, local links/marker/secret scan. Unchanged runtime/race/fuzz/benchmark
results are not claimed freshly run. Shared objective-check CI rollout is pending.

## Complete file inventory and initial findings

| Tracked Go file | Audit and disposition |
| --- | --- |
| semaphore.go | Canonical package scope too terse; Config/Snapshot fields missing; zero-value/concurrency/shutdown/Wait drain-generation contracts missing. |
| permit.go | Permit non-copyability and admission/cancellation precedence need prose; ID wraps and is not global/fencing identity; duplicate release needs typed contract. |
| errors.go | Structured diagnostic fields lack documentation; cancellation sentinels also classify Wait, not just Acquire. |
| execute.go | Caller-owned callback, no shutdown/cancellation enforcement, nil collaborator and panic behavior need clarification. |
| observation.go | Event/Batch fields missing; caller-owned copies are mutable, not immutable; loss saturates while ordinary counters wrap. |
| architecture_internal_test.go | AST guard protects absence of hidden goroutines/timers/finalizers; terminal waiter cases protect detached queue nodes. |
| benchmark_test.go | Workloads and non-equivalent reference semantics need comments; churn benchmarks include setup in measurement. |
| event_ownership_internal_test.go | Owned drain and disabled retention tests are self-describing; overflow case protects saturating loss accounting. |
| example_test.go | Both unchanged runnable examples demonstrate permit release/Execute ownership. |
| fuzz_test.go | Numeric trust boundary and accounting invariant need a concise fuzz-target comment. |
| hardening_test.go | Deterministic concurrent histories/cancellation matrix, FIFO starvation, deadlines and weak permit retention protect lifecycle. |
| model_test.go | Seeded reference history protects weighted conservation; close/release race and integer extremes protect ownership/overflow. |
| observation_test.go | Pull-based bounded events protect queue/cancel/close snapshots and no callback path. |
| release_linearization_internal_test.go | Custom contexts model reentrant caller code; duplicate Release waits for accounting under one lock. |
| semaphore_test.go | Weighted ownership, duplicate release, context/error precedence, FIFO, bounded saturation, shutdown and execution outcomes. |

No build tags, platform-only files, cgo, generated source or source regeneration
commands occur in these tracked Go files. No unfinished-debt, workaround, warning or deprecation markers were found. Existing design, operations, security and
performance pages own broad rationale; source comments avoid duplicating them.
No specification conformance claim is introduced.

## Questions and delivery

No rationale decision needs guessing. PermitID and snapshot counters can wrap;
the comments must describe this rather than invent overflow rejection. Wait
observes a drain generation, not a permanent quiescence guarantee; callers need
Close first when admission must stop. There is no runtime defect or separately
authorized behavioral repair in this batch. Documentation-only changes do not
require a module release. Remote-main delivery and applicable CI remain pending.

## Local completed evidence

- `gofmt -l` affected source: clean; `git diff --check`: passed.
- Go 1.27.0 disposable scanner/parser inspected all 15 tracked Go files:
  identical executable tokens against baseline, 101 exported declarations and
  fields with idiomatic comments, exactly one canonical package comment.
- `go doc -all .`: passed and rendered package/API documentation inspected,
  including nil-context, FIFO cancellation precedence, non-copyable permits,
  counter wrap, event ownership, and Close versus drain-generation semantics.
- Existing design reference resolves locally; changed-content secret scan found
  no private-key/access-token patterns. No unfinished-debt/deprecation/generator
  or platform exceptions introduced.
- Ordinary complete-final-diff review: no unresolved documentation finding.
- Runtime/example tests, race, fuzz, mutation, benchmarks and release qualification
  were not selected: executable source and all examples remain unchanged.

The exact task-owned build/module/temporary caches, scanner and rendered output
were removed after the bounded verification session. Shared checker adoption,
remote-main delivery and exact-source CI remain pending, not claimed complete.

## Objective enforcement adoption

The local `source-docs-check` target is required by ordinary/full checks and CI.
Shared CI uses `source_documentation: check` with the immutable Tools v2.4.0
workflow and matching authenticated binary/checksum declaration. Structural
checks do not replace this maintained semantic audit or native docs/examples.
