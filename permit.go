package semaphore

import (
	"context"
	"errors"
	"strconv"
)

// PermitID is stable process-local identity metadata for one admission.
// Values increment per Semaphore and wrap on uint64 overflow. They are not
// globally unique identifiers, authorization credentials or fencing tokens.
type PermitID uint64

// IsZero reports whether the numeric identifier is zero, including the default
// value or an admission whose sequence wrapped to zero.
func (id PermitID) IsZero() bool { return id == 0 }

// String returns a bounded non-secret diagnostic representation.
func (id PermitID) String() string { return strconv.FormatUint(uint64(id), 10) }

// Permit owns acquired weight until Release succeeds exactly once.
// Only Acquire and TryAcquire create usable permits. Keep the returned pointer:
// copying a Permit would duplicate release state and violate capacity accounting.
// ID and Weight may run concurrently with Release. The zero value is unusable.
type Permit struct {
	semaphore *Semaphore
	id        PermitID
	weight    int64
	released  bool
}

// ID returns stable process-local identity metadata.
func (permit *Permit) ID() PermitID { return permit.id }

// Weight returns the acquired weight.
func (permit *Permit) Weight() int64 { return permit.weight }

// Release returns the permit's weight exactly once. It remains valid after the
// acquisition context is canceled and is safe for concurrent callers.
// It also remains valid after Close. The first call returns nil; subsequent
// calls return *DuplicateReleaseError matching ErrDuplicateRelease only after
// the successful release has completed capacity accounting.
func (permit *Permit) Release() error {
	permit.semaphore.mu.Lock()
	if permit.released {
		permit.semaphore.mu.Unlock()
		return &DuplicateReleaseError{ID: permit.id}
	}
	permit.released = true
	permit.semaphore.acquired -= permit.weight
	permit.semaphore.recordEventLocked(EventReleased, ReasonReleased, permit.id, permit.weight)
	permit.semaphore.grantWaitersLocked()
	if permit.semaphore.acquired == 0 && permit.semaphore.drained != nil {
		close(permit.semaphore.drained)
		permit.semaphore.drained = nil
	}
	permit.semaphore.mu.Unlock()
	return nil
}

// Acquire acquires positive weight immediately when capacity is available or
// waits in strict FIFO order. The context must be non-nil.
// It returns a non-nil caller-owned permit only on success; errors return nil.
// Nonpositive and oversized weights produce *WeightError before context or
// closed-state checks. An already-canceled context produces *CanceledError
// before closed-state checks. Closed admission produces *ClosedError; a full
// waiter queue produces *QueueFullError, including when MaxWaiters is zero.
//
// Once queued, a grant or Close already committed under the accounting lock
// takes precedence over cancellation. A successful grant therefore still owns
// weight even if ctx is now done. Always release the returned permit; canceling
// ctx does not release it or stop admitted work.
func (semaphore *Semaphore) Acquire(ctx context.Context, weight int64) (*Permit, error) {
	if reason, err := semaphore.validateWeight(weight); err != nil {
		semaphore.reject(reason, weight)
		return nil, err
	}
	done := ctx.Done()
	if err := ctx.Err(); err != nil {
		semaphore.recordCancellation(err, weight)
		return nil, canceledError(err)
	}

	semaphore.mu.Lock()
	select {
	case <-done:
		semaphore.mu.Unlock()
		err := ctx.Err()
		semaphore.recordCancellation(err, weight)
		return nil, canceledError(err)
	default:
	}
	if semaphore.closed {
		semaphore.rejections++
		semaphore.recordEventLocked(EventRejected, ReasonClosed, PermitID(0), weight)
		semaphore.mu.Unlock()
		return nil, &ClosedError{}
	}
	if semaphore.waiterCount == 0 && semaphore.capacity-semaphore.acquired >= weight {
		permit := semaphore.grantLocked(weight)
		semaphore.recordEventLocked(EventAdmitted, ReasonImmediate, permit.id, weight)
		semaphore.mu.Unlock()
		return permit, nil
	}
	if semaphore.waiterCount >= semaphore.maxWaiters {
		semaphore.rejections++
		semaphore.recordEventLocked(EventRejected, ReasonQueueFull, PermitID(0), weight)
		semaphore.mu.Unlock()
		return nil, &QueueFullError{MaxWaiters: semaphore.maxWaiters}
	}

	waiter := &waiter{weight: weight, ready: make(chan struct{})}
	semaphore.enqueueWaiterLocked(waiter)
	semaphore.recordEventLocked(EventQueued, ReasonFIFO, PermitID(0), weight)
	semaphore.mu.Unlock()

	var cause error
	select {
	case <-waiter.ready:
	case <-done:
		cause = ctx.Err()
	}

	semaphore.mu.Lock()
	// Resolve the state transition, not the select winner: returning cancellation
	// after a committed grant would strand acquired weight without an owner.
	if waiter.permit != nil {
		permit := waiter.permit
		semaphore.mu.Unlock()
		return permit, nil
	}
	if waiter.err != nil {
		err := waiter.err
		semaphore.mu.Unlock()
		return nil, err
	}
	semaphore.removeWaiterLocked(waiter)
	semaphore.cancellations++
	semaphore.recordEventLocked(EventCanceled, cancellationReason(cause), PermitID(0), weight)
	semaphore.grantWaitersLocked()
	semaphore.mu.Unlock()
	return nil, canceledError(cause)
}

// TryAcquire attempts immediate admission without bypassing queued callers.
// Success returns a permit, true and nil. Temporarily unavailable capacity or
// a queued predecessor returns nil, false and nil; invalid/oversized weights or
// closed admission return nil, false and the same typed errors as Acquire.
func (semaphore *Semaphore) TryAcquire(weight int64) (*Permit, bool, error) {
	if reason, err := semaphore.validateWeight(weight); err != nil {
		semaphore.reject(reason, weight)
		return nil, false, err
	}

	semaphore.mu.Lock()
	if semaphore.closed {
		semaphore.rejections++
		semaphore.recordEventLocked(EventRejected, ReasonClosed, PermitID(0), weight)
		semaphore.mu.Unlock()
		return nil, false, &ClosedError{}
	}
	if semaphore.waiterCount != 0 || semaphore.capacity-semaphore.acquired < weight {
		semaphore.rejections++
		semaphore.recordEventLocked(EventRejected, ReasonUnavailable, PermitID(0), weight)
		semaphore.mu.Unlock()
		return nil, false, nil
	}

	permit := semaphore.grantLocked(weight)
	semaphore.recordEventLocked(EventAdmitted, ReasonImmediate, permit.id, weight)
	semaphore.mu.Unlock()
	return permit, true, nil
}

func (semaphore *Semaphore) grantLocked(weight int64) *Permit {
	semaphore.nextID++
	semaphore.acquired += weight
	semaphore.admissions++
	return &Permit{semaphore: semaphore, id: PermitID(semaphore.nextID), weight: weight}
}

func (semaphore *Semaphore) grantWaitersLocked() {
	for {
		waiter := semaphore.waiterHead
		if waiter == nil {
			return
		}
		if semaphore.capacity-semaphore.acquired < waiter.weight {
			// Skipping a heavy head admits a stream of small followers that can
			// starve it. TestWeightedHeadEventuallyRunsWithoutFollowerBypass
			// protects this intentional utilization-versus-fairness tradeoff.
			return
		}
		semaphore.removeWaiterLocked(waiter)
		waiter.permit = semaphore.grantLocked(waiter.weight)
		semaphore.recordEventLocked(EventAdmitted, ReasonFIFO, waiter.permit.id, waiter.weight)
		close(waiter.ready)
	}
}

func (semaphore *Semaphore) validateWeight(weight int64) (Reason, error) {
	if weight <= 0 {
		return ReasonInvalidWeight, &WeightError{Weight: weight, Capacity: semaphore.capacity}
	}
	if weight > semaphore.capacity {
		return ReasonOversize, &WeightError{Weight: weight, Capacity: semaphore.capacity, oversize: true}
	}
	return "", nil
}

func (semaphore *Semaphore) reject(reason Reason, weight int64) {
	semaphore.mu.Lock()
	semaphore.rejections++
	semaphore.recordEventLocked(EventRejected, reason, PermitID(0), weight)
	semaphore.mu.Unlock()
}

func (semaphore *Semaphore) recordCancellation(cause error, weight int64) {
	semaphore.mu.Lock()
	semaphore.cancellations++
	semaphore.recordEventLocked(EventCanceled, cancellationReason(cause), PermitID(0), weight)
	semaphore.mu.Unlock()
}

func cancellationReason(err error) Reason {
	if errors.Is(err, context.DeadlineExceeded) {
		return ReasonDeadline
	}
	return ReasonContextCanceled
}
