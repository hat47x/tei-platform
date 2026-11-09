package execution_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/effectlog"
	"tei/internal/execution"
)

type fakeRow struct {
	count int64
	err   error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	pointer, ok := dest[0].(*int64)
	if !ok {
		return errors.New("unexpected scan destination")
	}
	*pointer = r.count
	return nil
}

type fakeQueryer struct {
	row   fakeRow
	query string
}

func (q *fakeQueryer) QueryRowContext(_ context.Context, query string, _ ...any) execution.RowScanner {
	q.query = query
	return q.row
}

func TestSQLEffectObserverConfirmsVisibleRows(t *testing.T) {
	queryer := &fakeQueryer{row: fakeRow{count: 2}}
	observed, detail, err := execution.SQLEffectObserver{Queryer: queryer, Query: "SELECT count(*) FROM stock"}.ObserveEffect(context.Background(), effectlog.Record{RuleID: "sales.order.stock"})
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if !observed || detail == "" {
		t.Fatalf("two rows must confirm the effect: observed=%v detail=%q", observed, detail)
	}
	if queryer.query != "SELECT count(*) FROM stock" {
		t.Fatalf("the configured query must be used: %q", queryer.query)
	}
}

func TestSQLEffectObserverDoesNotConfirmZeroRows(t *testing.T) {
	observed, _, err := execution.SQLEffectObserver{Queryer: &fakeQueryer{row: fakeRow{}}, Query: "SELECT count(*) FROM stock"}.ObserveEffect(context.Background(), effectlog.Record{RuleID: "sales.order.stock"})
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if observed {
		t.Fatal("zero rows is not an observation")
	}
}

func TestSQLEffectObserverReportsAReadError(t *testing.T) {
	observer := execution.SQLEffectObserver{Queryer: &fakeQueryer{row: fakeRow{err: errors.New("connection closed")}}, Query: "SELECT count(*) FROM stock"}
	if _, _, err := observer.ObserveEffect(context.Background(), effectlog.Record{RuleID: "sales.order.stock"}); err == nil {
		t.Fatal("a failed read must be reported, not treated as unobserved")
	}
}

func TestSQLEffectObserverWithoutAQueryDoesNotConfirm(t *testing.T) {
	observed, detail, err := execution.SQLEffectObserver{}.ObserveEffect(context.Background(), effectlog.Record{RuleID: "sales.order.stock"})
	if err != nil || observed || detail == "" {
		t.Fatalf("an unconfigured observer must say so: observed=%v detail=%q err=%v", observed, detail, err)
	}
}
