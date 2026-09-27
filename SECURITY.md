# Security policy

## Supported versions

The latest stable release in the v2 line receives security fixes. Release
preparation on `main` is not publication or a supported artifact. Legacy v1
consumers must evaluate their own observation and lifecycle exposure; security
support must not be inferred from the continued availability of old tags.

## Reporting a vulnerability

Do not disclose a suspected vulnerability in a public issue. Use the
repository's private security reporting facility when available. If private
reporting is unavailable, ask a maintainer for a private contact channel
without disclosing the vulnerability.

Do not include credentials, production identifiers, customer data, or exploit
payloads in a public report, initial contact request, fixture, event, snapshot,
or benchmark artifact.

## Model

The package processes numeric configuration, weights, contexts, and
caller-owned protected operations. Configuration, queues, and event retention
are bounded. Observation invokes no caller callback; operation callbacks and
custom contexts are trusted in-process collaborators, not a sandbox boundary.
The primitive owns no network, filesystem, process, environment, reflection,
or unsafe operation. Callers must release permits and bound protected work.
See the [threat model](docs/security/threat-model.md) for residual risks.
