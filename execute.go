package semaphore

import "context"

// Execute acquires weight, invokes operation, and releases on success, error,
// or panic. It preserves the returned value and error or the original panic.
// An acquisition failure returns T's zero value and does not invoke operation.
// ctx, semaphore and operation must be non-nil. operation runs synchronously in
// the caller's goroutine with ctx; the caller must make it honor cancellation.
// Close cannot interrupt it. Callbacks may reenter the Semaphore, but must not
// synchronously wait for weight they themselves hold or otherwise cannot drain.
func Execute[T any](ctx context.Context, semaphore *Semaphore, weight int64, operation func(context.Context) (T, error)) (result T, err error) {
	permit, err := semaphore.Acquire(ctx, weight)
	if err != nil {
		return result, err
	}
	defer func() {
		panicValue := recover()
		_ = permit.Release()
		if panicValue != nil {
			panic(panicValue)
		}
	}()
	return operation(ctx)
}

// Run is the error-only convenience form of Execute.
func (semaphore *Semaphore) Run(ctx context.Context, weight int64, operation func(context.Context) error) error {
	_, err := Execute(ctx, semaphore, weight, func(ctx context.Context) (struct{}, error) {
		return struct{}{}, operation(ctx)
	})
	return err
}
