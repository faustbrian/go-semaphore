package semaphore

import "testing"

func TestDrainedEventsAreOwnedAndDisabledBufferRetainsNothing(t *testing.T) {
	sem, err := New(Config{Capacity: 1, EventBuffer: 1})
	if err != nil {
		t.Fatal(err)
	}
	permit, acquired, err := sem.TryAcquire(1)
	if err != nil || !acquired {
		t.Fatalf("TryAcquire = %t, %v", acquired, err)
	}
	first := sem.DrainEvents()
	if len(first.Events) != 1 || first.Events[0].Kind != EventAdmitted {
		t.Fatalf("first batch = %+v", first)
	}
	first.Events[0].Kind = EventCanceled
	if err := permit.Release(); err != nil {
		t.Fatal(err)
	}
	second := sem.DrainEvents()
	if len(second.Events) != 1 || second.Events[0].Kind != EventReleased {
		t.Fatalf("second batch = %+v", second)
	}
	if first.Events[0].Kind != EventCanceled {
		t.Fatal("later recording changed caller-owned batch")
	}
	disabled, err := New(Config{Capacity: 1})
	if err != nil {
		t.Fatal(err)
	}
	permit, _, err = disabled.TryAcquire(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := permit.Release(); err != nil {
		t.Fatal(err)
	}
	if batch := disabled.DrainEvents(); len(batch.Events) != 0 || batch.Dropped != 0 {
		t.Fatalf("disabled batch = %+v", batch)
	}
}

func TestDroppedEventCountSaturates(t *testing.T) {
	sem, err := New(Config{Capacity: 1, EventBuffer: 1})
	if err != nil {
		t.Fatal(err)
	}
	sem.droppedEvents = ^uint64(0)
	permit, _, err := sem.TryAcquire(1)
	if err != nil {
		t.Fatal(err)
	}
	if err := permit.Release(); err != nil {
		t.Fatal(err)
	}
	if batch := sem.DrainEvents(); batch.Dropped != ^uint64(0) || len(batch.Events) != 1 || batch.Events[0].Kind != EventReleased {
		t.Fatalf("saturated batch = %+v", batch)
	}
	if batch := sem.DrainEvents(); batch.Dropped != 0 {
		t.Fatalf("drain did not reset loss = %+v", batch)
	}
}
