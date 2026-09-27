# Semaphore threat model

Scope: root v2 contract; release preparation dated 2026-09-27.

## Scope and trust boundaries

The module owns process-local capacity accounting, permit ownership, FIFO
waiter links, shutdown state, aggregate counters, and an optional bounded event
ring. Capacity, queue limits, event limits, acquisition weights, contexts, and
operation callbacks enter from callers. Events contain only package-defined
enums, numeric permit metadata, and aggregate snapshots.

The module performs no network, filesystem, database, queue, cache, process,
environment, reflection, cryptographic, serialization, or unsafe operation. It
does not authenticate or authorize work. Callers own goroutines, context
deadlines, protected operations, event export, and process termination.

## Security properties

- Capacity arithmetic and permit release linearize under one mutex.
- Waiter and event storage have construction-time count bounds.
- Invalid weights and saturated queues fail before allocation.
- Cancellation is rechecked after accounting-lock acquisition and removes a
  waiting caller before its grant; a linearized grant always returns its permit
  so capacity ownership is not lost.
- Duplicate and concurrent releases cannot return capacity twice.
- Closing rejects queued and future admission while preserving existing permit
  ownership and bounded draining.
- Observation invokes no caller callback. A full ring overwrites its oldest
  event and reports a saturating dropped count.
- Errors and events retain no callback, context value, credential, label, or
  arbitrary caller string.
- Production owns no goroutine, timer, finalizer, registry, or implicit I/O.

## Threat analysis

| Threat | Disposition |
| --- | --- |
| Excess queue or event allocation | Mitigated by `MaxWaiters`, `MaxEventBuffer`, and validation before allocation. |
| Weighted overflow or capacity corruption | Mitigated by positive weights, capacity comparison before addition, and mutex-owned accounting. |
| Cancellation, close, grant, or duplicate-release races | Mitigated by one linearization lock and race/model regressions. |
| Telemetry callback blocks after admission and strands capacity | Removed in v2 by replacing callbacks with bounded pull-based observation. |
| Secret disclosure | Avoided by accepting no arbitrary diagnostic strings and exposing only bounded numeric/package-defined metadata. |
| SSRF, traversal, injection, parser, decompression, or cryptographic attack | Not applicable because the module owns none of those boundaries. |
| Abandoned permit or cancellation-ignoring operation | Accepted caller lifecycle risk below; automatic expiry cannot prove protected work stopped. |
| Process crash loses local exclusion | Accepted deployment-scope risk below; this module is not a distributed lease or fence. |

## Accepted risks

| Risk | Owner | Rationale and mitigation | Review condition |
| --- | --- | --- | --- |
| A caller can abandon a permit or run work that ignores cancellation indefinitely. | Integrating application | Only the caller can establish that protected work stopped. Use `Execute`, operation deadlines, shutdown `Close` plus bounded `Wait`, and a process supervisor; never infer that timeout released capacity. | Revisit if Go gains safe preemption or the module adopts an explicit external lease/fencing boundary. |
| A crash discards permits and waiter state, and replica-local limits do not form global exclusion. | Deployment owner | The primitive is intentionally process-local. Use a durable distributed lease with fencing for cross-process safety and calculate replica aggregate capacity explicitly. | Revisit when a consumer requires cross-process ownership. |
| A full event ring loses the oldest telemetry event. | Observability owner | Admission safety takes priority over telemetry delivery. Size `EventBuffer`, drain it regularly, and alert on `EventBatch.Dropped`; zero explicitly disables retention. | Revisit if loss-free audit delivery becomes a requirement. |
| Monotonic diagnostic counters and permit IDs can saturate or wrap only after process-lifetime operation counts approaching `uint64` exhaustion. | Package maintainers | IDs are neither authorization nor fencing tokens, and normal process recycling occurs many orders of magnitude earlier. Consumers must not assign security identity to them. | Revisit if counters become durable, externally trusted, or reachable at accelerated rates. |

## Release boundary

The callback removal is behaviorally and API incompatible with released v1.
Active source uses `/v2` at the repository root and is eligible for release
when its required source checks pass. Eligibility does not prove publication.
Consumer migration requires the actual public v2 tag and proxy artifacts;
legacy v1 API evidence and frozen compatibility cohorts remain separate.
