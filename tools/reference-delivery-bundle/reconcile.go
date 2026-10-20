package deliverybundle

import (
	"fmt"
	"os"
	"time"
)

type ReconcileReceipt struct {
	Action  string          `json:"action"`
	Healthy bool            `json:"healthy"`
	Service *ServiceReceipt `json:"service,omitempty"`
	Health  *HealthReceipt  `json:"health,omitempty"`
}

// Reconcile performs one supervision pass: start if absent, leave a healthy
// service alone, and restart an unhealthy service. It is intended to be invoked
// periodically by an external supervisor; it does not loop or monitor.
func Reconcile(statePath, serviceCommand, healthCommand, logPath string, timeout time.Duration) (ReconcileReceipt, error) {
	if _, err := Status(statePath); err != nil {
		return ReconcileReceipt{}, err
	}
	if healthCommand == "" {
		return ReconcileReceipt{}, fmt.Errorf("health command is required")
	}
	record, recordErr := readingRecord(statePath)
	if recordErr != nil && !os.IsNotExist(recordErr) {
		return ReconcileReceipt{}, recordErr
	}
	if recordErr == nil && processAlive(record.PID) {
		health, err := Health(statePath, healthCommand, timeout)
		if err != nil {
			return ReconcileReceipt{}, err
		}
		if health.Healthy {
			return ReconcileReceipt{Action: "noop", Healthy: true, Health: &health}, nil
		}
		service, err := Restart(statePath, serviceCommand, logPath, timeout)
		if err != nil {
			return ReconcileReceipt{}, err
		}
		return ReconcileReceipt{Action: "restarted", Healthy: false, Service: &service, Health: &health}, nil
	}
	if recordErr == nil {
		if err := os.Remove(pidFile(statePath)); err != nil {
			return ReconcileReceipt{}, err
		}
	}
	service, err := Start(statePath, serviceCommand, logPath)
	if err != nil {
		return ReconcileReceipt{}, err
	}
	return ReconcileReceipt{Action: "started", Service: &service}, nil
}
