# Security policy

## Supported versions

The latest stable v1 release receives security fixes. Older releases and the
`main` branch are unsupported; upgrade before reporting unless the issue is a
regression under active development.

| Version | Supported |
| --- | --- |
| Latest stable v1 release | Yes |
| Older releases | No |
| `main` | No |

## Reporting a vulnerability

Do not disclose a suspected vulnerability in a public issue. Use the
repository's private security reporting facility when available. If private
reporting is unavailable, ask a maintainer for a private contact channel
without disclosing the vulnerability.

Do not include credentials, production identifiers, customer data, or exploit
payloads in a public report, initial contact request, fixture, event, snapshot,
or benchmark artifact.

## Model

The package processes only numeric configuration, weights, contexts, and a
caller-supplied observer. Configuration and queues are bounded. It performs no
network, filesystem, process, environment, reflection, or unsafe operation.
Observer callbacks are untrusted: they run outside accounting locks and panics
are recovered. Callers remain responsible for releasing permits and bounding
the work protected by each permit.
