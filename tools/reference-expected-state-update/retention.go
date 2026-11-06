package expectedstateupdate

import (
	"fmt"
	"sort"
	"time"
)

// RetentionPolicy is the reference retention policy for observation records
// (#2845). It keeps at most MaxRuns runs and drops runs older than MaxAge.
// Both limits are required: an unset limit is a configuration error, not an
// unlimited policy.
type RetentionPolicy struct {
	MaxRuns int
	MaxAge  time.Duration
}

// RetainedRun is one stored run with the time it was recorded. The timestamp
// is supplied by the store; the record itself carries no clock.
type RetainedRun struct {
	RunID      string
	RecordedAt time.Time
	Record     Record
}

// Validate checks the policy before any run is retained.
func (p RetentionPolicy) Validate() error {
	if p.MaxRuns < 1 {
		return fmt.Errorf("retention: max runs must be at least 1, got %d", p.MaxRuns)
	}
	if p.MaxAge <= 0 {
		return fmt.Errorf("retention: max age must be positive, got %s", p.MaxAge)
	}
	return nil
}

// Retain returns the runs to keep, newest first. A run without a recorded time
// cannot be aged and fails closed. The input order is not trusted.
func (p RetentionPolicy) Retain(runs []RetainedRun, now time.Time) ([]RetainedRun, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	for i, run := range runs {
		if run.RecordedAt.IsZero() {
			return nil, fmt.Errorf("retention: run %d has no recorded time", i)
		}
	}
	ordered := append([]RetainedRun(nil), runs...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].RecordedAt.After(ordered[j].RecordedAt)
	})
	kept := make([]RetainedRun, 0, len(ordered))
	for _, run := range ordered {
		if now.Sub(run.RecordedAt) > p.MaxAge {
			continue
		}
		if len(kept) == p.MaxRuns {
			break
		}
		kept = append(kept, run)
	}
	return kept, nil
}
