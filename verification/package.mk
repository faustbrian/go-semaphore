.PHONY: docs leak stress

docs:
	./scripts/check-docs.sh

stress:
	go test -race ./... -run 'Concurrent|Generated|FIFO|WeightedHead|Cancellation|Close' -count=20

leak:
	go test -race ./... -run 'Cancellation|Close|Panic|Wait|TerminalWaiters|ReleasedPermit|HiddenGoroutine' -count=20
