package execution

import (
	"context"
	"fmt"

	"tei/internal/effectlog"
)

// RowScanner is the part of a database row this package needs.
type RowScanner interface {
	Scan(dest ...any) error
}

// Queryer runs one read. It is satisfied by a database handle and by a test
// double, so the observer can be exercised without a database.
type Queryer interface {
	QueryRowContext(ctx context.Context, query string, args ...any) RowScanner
}

// SQLEffectObserver confirms an effect with an independent read: the query must
// return a count of rows that the effect produced. Zero rows means the effect is
// not visible, which is not a success.
type SQLEffectObserver struct {
	Queryer Queryer
	Query   string
	Args    func(effectlog.Record) []any
}

// ObserveEffect implements EffectObserver.
func (o SQLEffectObserver) ObserveEffect(ctx context.Context, record effectlog.Record) (bool, string, error) {
	if o.Queryer == nil || o.Query == "" {
		return false, "no observation is configured", nil
	}
	var args []any
	if o.Args != nil {
		args = o.Args(record)
	}
	var count int64
	if err := o.Queryer.QueryRowContext(ctx, o.Query, args...).Scan(&count); err != nil {
		return false, "", fmt.Errorf("observe %s: %w", record.RuleID, err)
	}
	if count == 0 {
		return false, "the effect is not visible yet", nil
	}
	return true, fmt.Sprintf("%d row(s) confirm the effect", count), nil
}
