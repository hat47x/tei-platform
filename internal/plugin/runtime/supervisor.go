package runtime

import (
	"context"
	"errors"
	"fmt"
	"time"

	"tei/internal/resource"
)

// LeaseRefreshTrigger is a host-side signal source (for example a secret-manager
// rotation notification). It never supplies a concrete credential. When it
// returns nil, the supervisor asks the existing lease controller to Refresh the
// resource view under the already-pinned runtime policy.
type LeaseRefreshTrigger interface {
	WaitForRefresh(context.Context, resource.Lease) error
}

type leaseSupervisor struct {
	controller ResourceLeaseController
	recorder   *leaseController
	policy     resource.LeasePolicy
	trigger    LeaseRefreshTrigger
	now        func() time.Time
}

func (s *leaseSupervisor) currentTime() time.Time {
	if s != nil && s.now != nil {
		return s.now()
	}
	return time.Now()
}

func (s *leaseSupervisor) record(state resource.SupervisionState, message string) {
	if s != nil && s.recorder != nil {
		s.recorder.recordSupervision(state, message)
	}
}

func parseLeaseExpiry(l resource.Lease) (time.Time, error) {
	if l.ExpiresAt == "" {
		return time.Time{}, fmt.Errorf("resource lease %s has no expiry", l.ID)
	}
	t, err := time.Parse(time.RFC3339, l.ExpiresAt)
	if err != nil {
		return time.Time{}, fmt.Errorf("resource lease %s has invalid expiry %q: %w", l.ID, l.ExpiresAt, err)
	}
	return t, nil
}

// run supervises one invocation lease until the invocation context ends. It
// renews automatically when the configured renew-before window opens. A host
// refresh trigger may request an explicit Refresh, but refresh never extends
// the lease lifetime. Under the M0.20 fail-closed policy, any supervision error
// terminates the invocation rather than leaving authority unmanaged.
func (s *leaseSupervisor) run(ctx context.Context) error {
	if s == nil || s.controller == nil {
		return nil
	}
	policy := s.policy.WithDefaults()
	if err := policy.Validate(); err != nil {
		return err
	}
	for {
		current := s.controller.Current()
		expiry, err := parseLeaseExpiry(current)
		if err != nil {
			s.record(resource.SupervisionRenewalDeadlineMissed, err.Error())
			return err
		}
		now := s.currentTime()
		if !now.Before(expiry) {
			err := fmt.Errorf("resource lease %s renewal deadline missed at %s", current.ID, expiry.Format(time.RFC3339Nano))
			s.record(resource.SupervisionRenewalDeadlineMissed, err.Error())
			return err
		}
		renewAt := expiry.Add(-policy.RenewBeforeDuration())
		wait := renewAt.Sub(now)
		if wait < 0 {
			wait = 0
		}
		timer := time.NewTimer(wait)

		var refreshCh <-chan error
		var refreshCancel context.CancelFunc
		if s.trigger != nil {
			triggerCtx, cancel := context.WithCancel(ctx)
			refreshCancel = cancel
			ch := make(chan error, 1)
			refreshCh = ch
			go func(lease resource.Lease) {
				ch <- s.trigger.WaitForRefresh(triggerCtx, lease)
			}(current)
		}

		select {
		case <-ctx.Done():
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			if refreshCancel != nil {
				refreshCancel()
			}
			return nil
		case err := <-refreshCh:
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			if refreshCancel != nil {
				refreshCancel()
			}
			if err != nil {
				if errors.Is(err, context.Canceled) && ctx.Err() != nil {
					return nil
				}
				s.record(resource.SupervisionRefreshFailed, err.Error())
				return fmt.Errorf("resource refresh trigger failed: %w", err)
			}
			s.record(resource.SupervisionRefreshDue, "host refresh trigger fired")
			refreshed, err := s.controller.Refresh(ctx)
			if err != nil {
				s.record(resource.SupervisionRefreshFailed, err.Error())
				return fmt.Errorf("supervised resource refresh failed: %w", err)
			}
			s.record(resource.SupervisionRefreshed, fmt.Sprintf("lease refreshed to revision %d", normalizedLeaseRevision(refreshed)))
			continue
		case <-timer.C:
			if refreshCancel != nil {
				refreshCancel()
			}
			s.record(resource.SupervisionRenewalDue, "renew-before window opened")
			renewed, err := s.controller.Renew(ctx)
			if err != nil {
				s.record(resource.SupervisionRenewalFailed, err.Error())
				now := s.currentTime()
				if now.After(expiry) || now.Equal(expiry) {
					s.record(resource.SupervisionRenewalDeadlineMissed, "renewal did not complete before lease expiry")
				}
				return fmt.Errorf("supervised resource lease renewal failed: %w", err)
			}
			s.record(resource.SupervisionRenewed, fmt.Sprintf("lease renewed to revision %d", normalizedLeaseRevision(renewed)))
		}
	}
}
