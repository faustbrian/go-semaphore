# Operations and security

Size capacity from the protected local resource, not request rate. Weighted
units must represent one documented, stable resource quantity. `MaxWaiters`
bounds memory and waiting demand; a full queue is overload evidence and should
normally fail fast rather than trigger unbounded retries.

Monitor snapshot utilization (`Acquired / Capacity`), queue depth, rejection
reasons, cancellation rate, and drain deadlines. Admissions are not downstream
successes, and queue-full or closed rejections must not be recorded as
dependency failures by an outer circuit breaker.

Observation is pull-based. `EventBuffer` reserves a fixed event ring and zero
disables event retention. `DrainEvents` returns transition-ordered owned data;
when producers overrun the ring, the oldest event is overwritten and the batch
reports how many events were dropped. Recording invokes no caller callback, so
telemetry cannot delay returning an admitted permit or releasing capacity.

The API accepts no keys, credentials, arbitrary labels, or error text. Permit
IDs are process-local monotonic diagnostics and must not be treated as secrets,
authorization, distributed fencing tokens, or globally unique identifiers.
Snapshots expose aggregate local state only.

On overload, investigate held permit duration and abandonment before raising
capacity. A permanently held permit is caller-owned lifecycle failure. The
package intentionally does not expire permits because expiration cannot stop
still-running work safely.
