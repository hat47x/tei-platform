package deliverybundle

import (
	"context"
	"fmt"
	"time"
)

// Supervise repeats Reconcile until iterations is reached or ctx is cancelled.
// A zero iterations value means no fixed limit. It does not add backoff or
// leader election; callers that need those should not use this helper.
func Supervise(ctx context.Context, statePath, serviceCommand, healthCommand, logPath string, timeout, interval time.Duration, iterations int, report func(ReconcileReceipt)) error {
	if iterations < 0 {
		return fmt.Errorf("iterations must not be negative")
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}
	for count := 0; iterations == 0 || count < iterations; count++ {
		receipt, err := Reconcile(statePath, serviceCommand, healthCommand, logPath, timeout)
		if err != nil {
			return err
		}
		if report != nil {
			report(receipt)
		}
		if iterations > 0 && count+1 >= iterations {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(interval):
		}
	}
	return nil
}
