package semaphore_test

import (
	"context"
	"errors"
	"testing"

	"github.com/faustbrian/go-semaphore/v2"
)

func TestBufferedEventsReportQueuedCancellationAndClosedRejection(t *testing.T) {
	t.Parallel()

	sem, err := semaphore.New(semaphore.Config{
		Capacity:    1,
		MaxWaiters:  1,
		EventBuffer: 16,
	})
	if err != nil {
		t.Fatal(err)
	}
	held, err := sem.Acquire(testContext(t), 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() {
		_, acquireErr := sem.Acquire(ctx, 1)
		result <- acquireErr
	}()
	waitForSnapshot(t, sem, func(snapshot semaphore.Snapshot) bool { return snapshot.Waiters == 1 })
	cancel()
	if err := receive(t, result); !errors.Is(err, semaphore.ErrCanceled) {
		t.Fatalf("Acquire() error = %v", err)
	}
	if err := sem.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := sem.Acquire(testContext(t), 1); !errors.Is(err, semaphore.ErrClosed) {
		t.Fatalf("closed Acquire() error = %v", err)
	}
	if err := held.Release(); err != nil {
		t.Fatal(err)
	}

	events := sem.DrainEvents().Events
	want := map[[2]string]bool{
		{string(semaphore.EventQueued), string(semaphore.ReasonFIFO)}:              false,
		{string(semaphore.EventCanceled), string(semaphore.ReasonContextCanceled)}: false,
		{string(semaphore.EventClosed), string(semaphore.ReasonShutdown)}:          false,
		{string(semaphore.EventRejected), string(semaphore.ReasonClosed)}:          false,
	}
	for _, event := range events {
		key := [2]string{string(event.Kind), string(event.Reason)}
		if _, exists := want[key]; exists {
			want[key] = true
		}
	}
	for transition, observed := range want {
		if !observed {
			t.Errorf("transition %v missing from %+v", transition, events)
		}
	}
}

func TestObservationIsPullBasedAndBounded(t *testing.T) {
	t.Parallel()

	sem, err := semaphore.New(semaphore.Config{Capacity: 1, EventBuffer: 2})
	if err != nil {
		t.Fatal(err)
	}
	permit, err := sem.Acquire(testContext(t), 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, acquired, err := sem.TryAcquire(1); err != nil || acquired {
		t.Fatalf("TryAcquire() = %t, %v", acquired, err)
	}
	if err := permit.Release(); err != nil {
		t.Fatal(err)
	}

	batch := sem.DrainEvents()
	if batch.Dropped != 1 || len(batch.Events) != 2 {
		t.Fatalf("DrainEvents() = %+v", batch)
	}
	if batch.Events[0].Kind != semaphore.EventRejected || batch.Events[1].Kind != semaphore.EventReleased {
		t.Fatalf("event kinds = %q, %q", batch.Events[0].Kind, batch.Events[1].Kind)
	}
	if next := sem.DrainEvents(); next.Dropped != 0 || len(next.Events) != 0 {
		t.Fatalf("second DrainEvents() = %+v", next)
	}
}
