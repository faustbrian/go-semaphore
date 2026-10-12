// Package semaphore provides a process-local FIFO weighted semaphore with
// explicit permit ownership and deterministic shutdown.
// Construct a Semaphore with New, then acquire and release a Permit or use
// Execute to release automatically when an operation returns or panics.
//
// Capacity limits simultaneous resource use, not request rate. Strict FIFO can
// leave capacity idle behind a large waiter; smaller requests cannot bypass it.
// Callers own operation deadlines and permit release. The package starts no
// goroutines or timers, introduces no infrastructure or cgo dependency, and does
// not provide distributed leases, fencing or automatic recovery of lost permits.
// Close stops admission; Wait observes held weight draining without stopping
// the protected operations themselves. See the repository's docs/design.md for
// the cancellation and shutdown model.
package semaphore

import (
	"context"
	"sync"
)

// MaxWaiters is the largest supported bounded FIFO queue.
const MaxWaiters = 1_000_000

// MaxEventBuffer is the largest supported bounded observation queue.
const MaxEventBuffer = 65_536

// Config defines capacity and queue bounds copied by New.
// Its zero value is invalid because Capacity must be positive.
type Config struct {
	// Capacity is the total available positive weight.
	Capacity int64
	// MaxWaiters bounds queued acquisitions; zero disables waiting.
	// Valid values range from zero through the package constant MaxWaiters.
	MaxWaiters int
	// EventBuffer bounds retained transition events; zero disables retention.
	// Valid values range from zero through MaxEventBuffer.
	EventBuffer int
}

// Snapshot is a value copy of observable semaphore state; modifying it cannot
// change the Semaphore. Counters wrap on uint64 overflow.
type Snapshot struct {
	// Capacity is the configured total weight.
	Capacity int64
	// Acquired is weight held by unreleased permits.
	Acquired int64
	// Available is Capacity minus Acquired, even after Close.
	Available int64
	// Waiters counts queued acquisitions not yet granted or removed.
	Waiters int
	// Admissions counts successful acquisitions, not completed operations.
	Admissions uint64
	// Rejections counts invalid, oversized, full-queue, unavailable and closed attempts.
	Rejections uint64
	// Cancellations counts acquisition attempts resolved as canceled or expired.
	// Canceled Wait calls do not increment it.
	Cancellations uint64
	// Closed reports that admission has stopped, not that held weight has drained.
	Closed bool
}

// Semaphore is a process-local weighted counting semaphore.
// All methods may be called concurrently. Construct it with New; its zero value
// is not a usable substitute. Do not copy a Semaphore after first use.
// It never expires permits or cancels admitted operations.
type Semaphore struct {
	mu            sync.Mutex
	capacity      int64
	acquired      int64
	maxWaiters    int
	admissions    uint64
	rejections    uint64
	cancellations uint64
	nextID        uint64
	closed        bool
	drained       chan struct{}
	waiterCount   int
	waiterHead    *waiter
	waiterTail    *waiter
	events        []Event
	eventHead     int
	eventCount    int
	droppedEvents uint64
}

type waiter struct {
	weight   int64
	ready    chan struct{}
	permit   *Permit
	err      error
	previous *waiter
	next     *waiter
}

// New constructs a semaphore after validating all configuration.
// Invalid capacity or queue/event bounds return nil and a *ConfigError matching
// ErrInvalidConfig. Construction starts no background work.
func New(config Config) (*Semaphore, error) {
	if config.Capacity <= 0 {
		return nil, &ConfigError{field: FieldCapacity, problem: ProblemMustBePositive}
	}
	if config.MaxWaiters < 0 {
		return nil, &ConfigError{field: FieldMaxWaiters, problem: ProblemMustNotBeNegative}
	}
	if config.MaxWaiters > MaxWaiters {
		return nil, &ConfigError{field: FieldMaxWaiters, problem: ProblemExceedsBound}
	}
	if config.EventBuffer < 0 {
		return nil, &ConfigError{field: FieldEventBuffer, problem: ProblemMustNotBeNegative}
	}
	if config.EventBuffer > MaxEventBuffer {
		return nil, &ConfigError{field: FieldEventBuffer, problem: ProblemExceedsBound}
	}
	return &Semaphore{
		capacity:   config.Capacity,
		maxWaiters: config.MaxWaiters,
		events:     make([]Event, config.EventBuffer),
	}, nil
}

// Snapshot returns a consistent value copy of current state.
func (semaphore *Semaphore) Snapshot() Snapshot {
	semaphore.mu.Lock()
	defer semaphore.mu.Unlock()
	return semaphore.snapshotLocked()
}

func (semaphore *Semaphore) snapshotLocked() Snapshot {
	return Snapshot{
		Capacity:      semaphore.capacity,
		Acquired:      semaphore.acquired,
		Available:     semaphore.capacity - semaphore.acquired,
		Waiters:       semaphore.waiterCount,
		Admissions:    semaphore.admissions,
		Rejections:    semaphore.rejections,
		Cancellations: semaphore.cancellations,
		Closed:        semaphore.closed,
	}
}

// Close idempotently stops new admission and rejects every queued waiter.
// Existing permits remain valid and releasable.
// It returns nil without waiting for those permits; use Wait to observe drain.
// Snapshot and DrainEvents remain usable after Close.
func (semaphore *Semaphore) Close() error {
	semaphore.mu.Lock()
	if semaphore.closed {
		semaphore.mu.Unlock()
		return nil
	}

	semaphore.closed = true
	for semaphore.waiterHead != nil {
		waiter := semaphore.waiterHead
		semaphore.removeWaiterLocked(waiter)
		waiter.err = &ClosedError{}
		semaphore.rejections++
		semaphore.recordEventLocked(EventRejected, ReasonClosed, PermitID(0), waiter.weight)
		close(waiter.ready)
	}
	semaphore.recordEventLocked(EventClosed, ReasonShutdown, PermitID(0), 0)
	semaphore.mu.Unlock()
	return nil
}

func (semaphore *Semaphore) enqueueWaiterLocked(waiter *waiter) {
	waiter.previous = semaphore.waiterTail
	if semaphore.waiterTail == nil {
		semaphore.waiterHead = waiter
	} else {
		semaphore.waiterTail.next = waiter
	}
	semaphore.waiterTail = waiter
	semaphore.waiterCount++
}

func (semaphore *Semaphore) removeWaiterLocked(waiter *waiter) {
	if waiter.previous == nil {
		semaphore.waiterHead = waiter.next
	} else {
		waiter.previous.next = waiter.next
	}
	if waiter.next == nil {
		semaphore.waiterTail = waiter.previous
	} else {
		waiter.next.previous = waiter.previous
	}
	waiter.previous = nil
	waiter.next = nil
	semaphore.waiterCount--
}

// Wait blocks until all acquired weight is returned or ctx is done. It does
// not close the semaphore or wait for queued callers unless they are admitted.
// Close first when shutdown must prevent further admission. Without Close,
// reaching zero is only a drain-generation boundary: new work may already hold
// weight by the time Wait returns. If no weight is held, Wait returns nil even
// for an expired context. Otherwise ctx must be non-nil; cancellation returns a
// *CanceledError matching the package and corresponding context sentinel.
// If drain and cancellation race, either result may be observed.
func (semaphore *Semaphore) Wait(ctx context.Context) error {
	semaphore.mu.Lock()
	if semaphore.acquired == 0 {
		semaphore.mu.Unlock()
		return nil
	}
	if semaphore.drained == nil {
		// Each channel represents one nonzero-to-zero generation. Release closes
		// it only after granting FIFO waiters, so a handoff is not a false drain.
		semaphore.drained = make(chan struct{})
	}
	drained := semaphore.drained
	semaphore.mu.Unlock()

	select {
	case <-drained:
		return nil
	case <-ctx.Done():
		return canceledError(ctx.Err())
	}
}

func (semaphore *Semaphore) recordEventLocked(kind EventKind, reason Reason, id PermitID, weight int64) {
	if len(semaphore.events) == 0 {
		return
	}
	event := Event{Kind: kind, Reason: reason, PermitID: id, Weight: weight, Snapshot: semaphore.snapshotLocked()}
	if semaphore.eventCount < len(semaphore.events) {
		index := (semaphore.eventHead + semaphore.eventCount) % len(semaphore.events)
		semaphore.events[index] = event
		semaphore.eventCount++
		return
	}
	semaphore.events[semaphore.eventHead] = event
	semaphore.eventHead = (semaphore.eventHead + 1) % len(semaphore.events)
	if semaphore.droppedEvents != ^uint64(0) {
		semaphore.droppedEvents++
	}
}
