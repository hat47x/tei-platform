// Package batchcompleteness is a read-only, synthetic reference for Data
// Derivation D13 (#1724). It distinguishes a normal NULL, an input that could
// not be read, and an evaluation failure at the row level, and it decides
// whether a query over a derived column can claim a complete result set.
//
// It is not a Query planner, a SQL executor, an authorization decision, or an
// accepted API. It does not write a Source. A normal NULL is not a failure and
// is never turned into zero; a row whose input could not be read is never
// silently dropped or counted as complete. The pure rule and input state come
// from Data Derivation D1 (#1637) in internal/semantic/derivationref.
package batchcompleteness

import (
	"fmt"
	"strings"

	"tei/internal/semantic/derivationref"
)

// Operation names. A set operation uses the derived column to decide the
// result set, so an unevaluable row cannot be excluded or replaced.
const (
	OpDisplay      = "display"
	OpFilter       = "filter"
	OpSort         = "sort"
	OpAggregate    = "aggregate"
	OpCountRows    = "count-rows"
	OpCountDerived = "count-derived"
)

// Result codes.
const (
	CodeOK                     = "OK"
	CodePartialProjection      = "PARTIAL_PROJECTION"
	CodeIncompleteSetOperation = "INCOMPLETE_SET_OPERATION"
	CodeIncompleteDisplay      = "INCOMPLETE_DISPLAY"
	CodeUnsupported            = "UNSUPPORTED_OPERATION"
)

// Row states in the report.
const (
	RowPresent          = "present"
	RowNull             = "null"
	RowInputUnavailable = "input-unavailable"
)

// Operand is one factor of the derived amount.
type Operand struct {
	State   string `json:"state"`
	Decimal string `json:"decimal,omitempty"`
}

// Row is one source row.
type Row struct {
	ID       string  `json:"id"`
	Quantity Operand `json:"quantity"`
	Price    Operand `json:"price"`
}

// Query is the operation that consumes the derived amount column.
type Query struct {
	Op string `json:"op"`
	// PartialProjectionAllowed is only meaningful for the display operation: the
	// case or API must explicitly allow returning successful rows plus per-row
	// diagnostics for an optional column that no set operation uses.
	PartialProjectionAllowed bool `json:"partialProjectionAllowed,omitempty"`
}

// Input is the whole synthetic request.
type Input struct {
	Rows  []Row `json:"rows"`
	Query Query `json:"query"`
}

// RowResult is the per-row outcome of the derivation.
type RowResult struct {
	ID     string `json:"id"`
	State  string `json:"state"`
	Amount string `json:"amount,omitempty"`
}

// Report is the read-only result. Complete is true only when a full result set
// can be claimed. Partial is true only for an explicitly allowed partial
// projection of an optional display column.
type Report struct {
	Op         string      `json:"op"`
	Complete   bool        `json:"complete"`
	Partial    bool        `json:"partial"`
	Code       string      `json:"code"`
	Message    string      `json:"message,omitempty"`
	Rows       []RowResult `json:"rows"`
	Unresolved []string    `json:"unresolved,omitempty"`
	Count      *int        `json:"count,omitempty"`
}

// Evaluate derives the amount of every row and decides whether the query can
// claim a complete result. It never writes anything.
func Evaluate(in Input) Report {
	report := Report{Op: strings.TrimSpace(in.Query.Op), Rows: make([]RowResult, 0, len(in.Rows))}
	for _, row := range in.Rows {
		result := RowResult{ID: row.ID}
		product, err := derivationref.Multiply(
			derivationref.Input{State: derivationref.InputState(strings.TrimSpace(row.Quantity.State)), Decimal: row.Quantity.Decimal},
			derivationref.Input{State: derivationref.InputState(strings.TrimSpace(row.Price.State)), Decimal: row.Price.Decimal},
		)
		if err != nil {
			// An unevaluable row is not a value and not a normal NULL.
			result.State = RowInputUnavailable
			report.Unresolved = append(report.Unresolved, row.ID)
		} else {
			result.State = string(product.State)
			result.Amount = product.Decimal
		}
		report.Rows = append(report.Rows, result)
	}

	switch report.Op {
	case OpCountRows:
		// A plain row count does not depend on the derived column.
		count := len(in.Rows)
		report.Count = &count
		report.Complete = true
		report.Code = CodeOK
	case OpDisplay:
		switch {
		case len(report.Unresolved) == 0:
			report.Complete = true
			report.Code = CodeOK
		case in.Query.PartialProjectionAllowed:
			report.Partial = true
			report.Code = CodePartialProjection
			report.Message = "任意表示列について、成功した行の値と評価不能の診断だけを返す。完全な計算済み表として扱わない。"
		default:
			report.Code = CodeIncompleteDisplay
			report.Message = "任意表示列でも部分Projectionが明示的に許可されていないため、全体を不成立とする。"
		}
	case OpFilter, OpSort, OpAggregate, OpCountDerived:
		if len(report.Unresolved) == 0 {
			report.Complete = true
			report.Code = CodeOK
		} else {
			report.Code = CodeIncompleteSetOperation
			report.Message = "集合演算に用いる導出列に評価不能行があるため、完全な結果・件数を返せない。除外や0/NULL置換をしない。"
		}
	default:
		report.Code = CodeUnsupported
		report.Message = fmt.Sprintf("unsupported operation %q", in.Query.Op)
	}
	return report
}
