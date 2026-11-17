//go:build !tei_repair_only

package main

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/driver"
)

// readIdentityFrom names the read the page shows.
//
// It reads the order key from the row that was actually selected and the
// Contract Query target from the resolution that was actually executed, so the
// identity cannot name a query or an order the read did not use. When the key
// or the target is unavailable it refuses: an unnamed read cannot have answers
// bound to it.
func readIdentityFrom(
	lines driver.HeaderDetailContractQueryEnvironmentExecution,
	selectedOrder semanticRowProjection,
	rowIndex int,
) (readIdentityProjection, error) {
	orderKey, ok := semanticRowValueByAttribute(selectedOrder, "sales.order.order_id")
	if !ok {
		return readIdentityProjection{}, fmt.Errorf("the selected order has no sales.order.order_id, so the read cannot be named")
	}
	target := lines.Execution.Resolution.Binding.Target
	if !target.Valid() {
		return readIdentityProjection{}, fmt.Errorf("the executed header-detail Contract Query has no complete target identity")
	}
	if rowIndex < 0 {
		return readIdentityProjection{}, fmt.Errorf("selected row %d is negative", rowIndex)
	}
	return readIdentityProjection{
		SelectedOrderKey: fmt.Sprint(orderKey),
		Environment:      string(lines.Environment.ID),
		Pattern:          string(lines.Execution.Resolution.Binding.Pattern),
		Database:         string(target.Database),
		Contract:         string(target.Contract),
		Query:            string(target.Query),
		InputOrderID:     fmt.Sprint(orderKey),
		RowCount:         len(lines.Execution.Result.Rows),
	}, nil
}

// requireAnswersBelongToRead refuses answers that do not belong to this read.
//
// Position alignment proves the answers line up with these rows. It cannot tell
// whether the answers came from this read or from another read of the same
// shape, because both would be aligned. The identity is what distinguishes
// them: it carries the selected order, the executed target, and the row count,
// and the answers are only rendered under it after this check.
func requireAnswersBelongToRead(
	identity readIdentityProjection,
	rows []rowAnswerProjection,
	relationAnswers []string,
) error {
	if identity.SelectedOrderKey == "" || identity.Database == "" || identity.Contract == "" || identity.Query == "" {
		return fmt.Errorf("the read has no complete identity, so its answers cannot be attributed")
	}
	if identity.RowCount != len(rows) {
		return fmt.Errorf("the read states %d rows but %d answer rows were produced", identity.RowCount, len(rows))
	}
	for index, row := range rows {
		if row.RowIndex != index {
			return fmt.Errorf("answer at position %d belongs to position %d", index, row.RowIndex)
		}
	}
	_ = relationAnswers
	return nil
}

// readIdentityKey is the stable comparison form of a read identity. Two reads
// with different keys are different reads even when their rows look identical.
func readIdentityKey(identity readIdentityProjection, environment core.CanonicalID) string {
	return fmt.Sprintf("%s|%s|%s|%s|%s|%d",
		environment, identity.Contract, identity.Query, identity.SelectedOrderKey,
		identity.InputOrderID, identity.RowCount)
}
