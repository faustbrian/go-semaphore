# Compatibility Policy

This repository currently has one releasable module at its root. Root releases
use `v<version>` tags. If an independently releasable nested module is added,
its tags use `<module-directory>/v<version>`.

Before `v1`, minor releases MAY contain reviewed breaking changes, but every
break MUST be documented with migration guidance. Patch releases MUST remain
backward compatible. At and after `v1`, incompatible exported API or documented
behavior changes require a new major version.

Compatibility includes exported Go APIs, error classification, serialization,
protocol behavior, persistence schemas, environment variables, command output,
resource ownership, ordering, retry/idempotency semantics, and documented
defaults. A compile-compatible change can still be behaviorally breaking.

Specification-backed modules MUST NOT diverge from their declared standards.
Ambiguities require documented decisions and stable tests. Deprecated APIs
follow [`DEPRECATION.md`](DEPRECATION.md).

## Current major-version boundary

The root v2 contract replaces v1 `Config.Observer`, `Observer`, and
`ObserverFunc` with bounded `Config.EventBuffer` and caller-pulled
`DrainEvents`. The immutable v1 API baseline is retained separately.

Adoption requires a public v2 tag and module proxy artifacts, then `/v2`
imports and an application-owned event draining/loss policy. Until those
artifacts exist, consumers must retain published v1 dependencies without
local replacements. The `go-service/integration/adoption` lifecycle consumer
needs an explicit migration from callback signaling to event draining.
The tools `release/compatibility-consumer` v1 cohort remains historical;
current v2 selection must be verified separately after publication.
