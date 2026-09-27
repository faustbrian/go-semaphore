package semaphore

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestValidateWeightBoundaries(t *testing.T) {
	sem := &Semaphore{capacity: 2}
	for _, test := range []struct {
		weight int64
		valid  bool
	}{
		{weight: -1},
		{weight: 0},
		{weight: 1, valid: true},
		{weight: 2, valid: true},
		{weight: 3},
	} {
		_, err := sem.validateWeight(test.weight)
		if (err == nil) != test.valid {
			t.Errorf("validateWeight(%d) error = %v", test.weight, err)
		}
	}
}

func TestAcquireDoesNotAdmitCancellationWhileWaitingForAccountingLock(t *testing.T) {
	t.Parallel()

	sem, err := New(Config{Capacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	base, cancel := context.WithCancel(context.Background())
	ctx := &errSignalingContext{Context: base, checked: make(chan struct{})}
	sem.mu.Lock()
	result := make(chan acquireResultInternal, 1)
	go func() {
		permit, acquireErr := sem.Acquire(ctx, 1)
		result <- acquireResultInternal{permit: permit, err: acquireErr}
	}()
	waitForContextCheck(t, ctx.checked)
	cancel()
	sem.mu.Unlock()

	outcome := receiveInternal(t, result)
	if outcome.permit != nil || !errors.Is(outcome.err, ErrCanceled) {
		t.Fatalf("Acquire() after lock-wait cancellation = %v, %v", outcome.permit, outcome.err)
	}
	if snapshot := sem.Snapshot(); snapshot.Acquired != 0 || snapshot.Cancellations != 1 {
		t.Fatalf("canceled acquisition consumed capacity: %+v", snapshot)
	}
}

type errSignalingContext struct {
	context.Context
	checked chan struct{}
	once    sync.Once
}

func (ctx *errSignalingContext) Err() error {
	err := ctx.Context.Err()
	ctx.once.Do(func() { close(ctx.checked) })
	return err
}

func TestAcquireDoesNotCallContextWhileHoldingAccountingLock(t *testing.T) {
	sem, err := New(Config{Capacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	base, cancel := context.WithCancel(context.Background())
	ctx := &snapshotErrContext{
		Context: base,
		sem:     sem,
		checked: make(chan struct{}),
	}
	sem.mu.Lock()
	result := make(chan acquireResultInternal, 1)
	go func() {
		permit, acquireErr := sem.Acquire(ctx, 1)
		result <- acquireResultInternal{permit: permit, err: acquireErr}
	}()
	waitForContextCheck(t, ctx.checked)
	cancel()
	sem.mu.Unlock()

	outcome := receiveInternal(t, result)
	if outcome.permit != nil || !errors.Is(outcome.err, ErrCanceled) {
		t.Fatalf("Acquire() after reentrant context cancellation = %v, %v", outcome.permit, outcome.err)
	}
	if snapshot := sem.Snapshot(); snapshot.Acquired != 0 || snapshot.Cancellations != 1 {
		t.Fatalf("canceled acquisition consumed capacity: %+v", snapshot)
	}
}

type snapshotErrContext struct {
	context.Context
	sem     *Semaphore
	checked chan struct{}
	calls   int
}

func TestQueuedCancellationCanReenterContextSnapshot(t *testing.T) {
	sem, err := New(Config{Capacity: 1, MaxWaiters: 1})
	if err != nil {
		t.Fatal(err)
	}
	held, err := sem.Acquire(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if releaseErr := held.Release(); releaseErr != nil {
			t.Errorf("release held permit: %v", releaseErr)
		}
	}()
	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &snapshotErrContext{Context: base, sem: sem, checked: make(chan struct{})}
	result := make(chan acquireResultInternal, 1)
	go func() {
		permit, acquireErr := sem.Acquire(ctx, 1)
		result <- acquireResultInternal{permit: permit, err: acquireErr}
	}()
	waitForSnapshotInternal(t, sem, func(snapshot Snapshot) bool { return snapshot.Waiters == 1 })
	cancel()
	outcome := receiveInternal(t, result)
	if outcome.permit != nil || !errors.Is(outcome.err, ErrCanceled) {
		t.Fatalf("queued reentrant cancellation = %v, %v", outcome.permit, outcome.err)
	}
	if snapshot := sem.Snapshot(); snapshot.Acquired != 1 || snapshot.Waiters != 0 || snapshot.Cancellations != 1 {
		t.Fatalf("queued cancellation accounting = %+v", snapshot)
	}
}

func (ctx *snapshotErrContext) Err() error {
	err := ctx.Context.Err()
	ctx.calls++
	if ctx.calls == 1 {
		close(ctx.checked)
	}
	if ctx.calls == 2 {
		ctx.sem.Snapshot()
	}
	return err
}

func waitForContextCheck(t *testing.T, checked <-chan struct{}) {
	t.Helper()
	select {
	case <-checked:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for initial context check")
	}
}

func TestConcurrentDuplicateReleaseWaitsForAccounting(t *testing.T) {
	t.Parallel()

	sem, err := New(Config{Capacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	permit, acquired, err := sem.TryAcquire(1)
	if err != nil || !acquired || permit == nil {
		t.Fatalf("TryAcquire() = %v, %t, %v", permit, acquired, err)
	}

	sem.mu.Lock()
	started := make(chan struct{}, 2)
	results := make(chan error, 2)
	for range 2 {
		go func() {
			started <- struct{}{}
			results <- permit.Release()
		}()
	}
	<-started
	<-started
	select {
	case result := <-results:
		sem.mu.Unlock()
		t.Fatalf("Release() completed before capacity accounting: %v", result)
	case <-time.After(20 * time.Millisecond):
	}
	sem.mu.Unlock()

	first := <-results
	second := <-results
	valid := first == nil && errors.Is(second, ErrDuplicateRelease)
	if errors.Is(first, ErrDuplicateRelease) && second == nil {
		valid = true
	}
	if !valid {
		t.Fatalf("release results = %v, %v", first, second)
	}
	if snapshot := sem.Snapshot(); snapshot.Acquired != 0 || snapshot.Available != 1 {
		t.Fatalf("snapshot = %+v", snapshot)
	}
}
