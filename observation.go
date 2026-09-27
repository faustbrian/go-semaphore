package semaphore

// EventKind identifies one bounded semaphore state transition.
type EventKind string

const (
	// EventAdmitted reports successful acquisition.
	EventAdmitted EventKind = "admitted"
	// EventQueued reports entry into the bounded FIFO queue.
	EventQueued EventKind = "queued"
	// EventCanceled reports removal from the queue by caller cancellation.
	EventCanceled EventKind = "canceled"
	// EventRejected reports work that did not enter or acquire from the queue.
	EventRejected EventKind = "rejected"
	// EventReleased reports successful exactly-once permit release.
	EventReleased EventKind = "released"
	// EventClosed reports the first shutdown transition.
	EventClosed EventKind = "closed"
)

// Reason is a bounded, low-cardinality transition reason.
type Reason string

const (
	// ReasonImmediate identifies acquisition without waiting.
	ReasonImmediate Reason = "immediate"
	// ReasonFIFO identifies admission from the FIFO queue.
	ReasonFIFO Reason = "fifo"
	// ReasonUnavailable identifies immediate capacity rejection.
	ReasonUnavailable Reason = "unavailable"
	// ReasonInvalidWeight identifies a non-positive weight.
	ReasonInvalidWeight Reason = "invalid_weight"
	// ReasonOversize identifies a weight above total capacity.
	ReasonOversize Reason = "oversize"
	// ReasonQueueFull identifies bounded queue saturation.
	ReasonQueueFull Reason = "queue_full"
	// ReasonContextCanceled identifies caller cancellation.
	ReasonContextCanceled Reason = "context_canceled"
	// ReasonDeadline identifies caller deadline expiry.
	ReasonDeadline Reason = "deadline"
	// ReasonClosed identifies work rejected by shutdown.
	ReasonClosed Reason = "closed"
	// ReasonReleased identifies successful permit release.
	ReasonReleased Reason = "released"
	// ReasonShutdown identifies the first close operation.
	ReasonShutdown Reason = "shutdown"
)

// Event is an immutable, bounded, secret-safe transition snapshot.
type Event struct {
	Kind     EventKind
	Reason   Reason
	PermitID PermitID
	Weight   int64
	Snapshot Snapshot
}

// EventBatch is an owned, bounded observation batch. Dropped reports events
// overwritten since the previous drain because the configured buffer was full.
type EventBatch struct {
	Events  []Event
	Dropped uint64
}

// DrainEvents returns and removes buffered events in transition order. The
// returned slice is caller-owned and contains at most Config.EventBuffer items.
func (semaphore *Semaphore) DrainEvents() EventBatch {
	semaphore.mu.Lock()
	defer semaphore.mu.Unlock()

	batch := EventBatch{Events: make([]Event, semaphore.eventCount), Dropped: semaphore.droppedEvents}
	for index := range semaphore.eventCount {
		batch.Events[index] = semaphore.events[(semaphore.eventHead+index)%len(semaphore.events)]
	}
	clear(semaphore.events)
	semaphore.eventHead = 0
	semaphore.eventCount = 0
	semaphore.droppedEvents = 0
	return batch
}
