package migration

import (
	"context"
	"fmt"

	"tei/internal/core"
)

// HistoryReader is the integration boundary for Atlas/Flyway/Liquibase/etc.
// Tool-specific schemas and CLI output stay behind adapters; TEI consumes only
// HistorySnapshot.
type HistoryReader interface {
	Tool() Tool
	ReadHistory(context.Context, core.CanonicalID) (HistorySnapshot, error)
}

func ReadHistory(ctx context.Context, r HistoryReader, database core.CanonicalID) (HistorySnapshot, error) {
	if r == nil {
		return HistorySnapshot{}, fmt.Errorf("migration history reader is nil")
	}
	h, err := r.ReadHistory(ctx, database)
	if err != nil {
		return HistorySnapshot{}, err
	}
	if h.Database != database {
		return HistorySnapshot{}, fmt.Errorf("migration history reader returned database %s, expected %s", h.Database, database)
	}
	if h.Tool == "" {
		h.Tool = r.Tool()
	}
	if h.Tool != r.Tool() {
		return HistorySnapshot{}, fmt.Errorf("migration history reader tool %s differs from snapshot tool %s", r.Tool(), h.Tool)
	}
	for i := range h.Entries {
		if h.Entries[i].Database == "" {
			h.Entries[i].Database = database
		}
		if h.Entries[i].Tool == "" {
			h.Entries[i].Tool = h.Tool
		}
	}
	return h.Normalize(), nil
}
