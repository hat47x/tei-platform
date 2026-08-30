package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
)

// AppliedState is the normalized outcome of one migration tool history entry.
// Tool-specific history tables/files are adapters; TEI consumes this stable IR.
type AppliedState string

const (
	AppliedSucceeded AppliedState = "applied"
	AppliedFailed    AppliedState = "failed"
	AppliedUnknown   AppliedState = "unknown"
)

type AppliedMigration struct {
	Database    core.CanonicalID `json:"database"`
	ID          string           `json:"id"`
	State       AppliedState     `json:"state"`
	Checksum    string           `json:"checksum,omitempty"`
	AppliedAt   string           `json:"appliedAt,omitempty"`
	Rank        int              `json:"rank,omitempty"`
	Tool        Tool             `json:"tool"`
	Description string           `json:"description,omitempty"`
}

type HistorySnapshot struct {
	Database   core.CanonicalID   `json:"database"`
	Tool       Tool               `json:"tool"`
	CapturedAt string             `json:"capturedAt,omitempty"`
	Entries    []AppliedMigration `json:"entries,omitempty"`
}

func (h HistorySnapshot) Normalize() HistorySnapshot {
	out := h
	out.Entries = append([]AppliedMigration(nil), h.Entries...)
	sort.SliceStable(out.Entries, func(i, j int) bool {
		if out.Entries[i].Rank != 0 && out.Entries[j].Rank != 0 && out.Entries[i].Rank != out.Entries[j].Rank {
			return out.Entries[i].Rank < out.Entries[j].Rank
		}
		return out.Entries[i].ID < out.Entries[j].ID
	})
	return out
}

// Fingerprint is a stable checksum of the canonical migration definition used
// to detect repository mutation after a migration has been shared/applied.
// It intentionally excludes provider execution history and source locations.
func Fingerprint(m Definition) (string, error) {
	normalized := m
	normalized.Parents = append([]string(nil), m.Parents...)
	sort.Strings(normalized.Parents)
	// Realizations are derived for model-first migrations; authored SQL steps
	// already live in Steps. Excluding derived realizations prevents a renderer
	// upgrade from pretending the canonical migration itself was edited.
	normalized.Realizations = nil
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

type RepositoryState string

const (
	RepoApplied       RepositoryState = "applied"
	RepoPending       RepositoryState = "pending"
	RepoFailed        RepositoryState = "failed"
	RepoDiverged      RepositoryState = "diverged"
	RepoOrphanApplied RepositoryState = "orphan-applied"
)

type HistoryComparisonEntry struct {
	Database           core.CanonicalID `json:"database"`
	ID                 string           `json:"id"`
	State              RepositoryState  `json:"state"`
	RepositoryChecksum string           `json:"repositoryChecksum,omitempty"`
	AppliedChecksum    string           `json:"appliedChecksum,omitempty"`
	AppliedAt          string           `json:"appliedAt,omitempty"`
	Tool               Tool             `json:"tool,omitempty"`
}

type HistoryComparison struct {
	Database core.CanonicalID         `json:"database"`
	Entries  []HistoryComparisonEntry `json:"entries"`
}

// CompareHistory compares append-only repository definitions with a normalized
// migration-tool history snapshot. It does not infer schema correctness; that
// is a separate Actual Schema comparison.
func CompareHistory(database core.CanonicalID, defs []Definition, history HistorySnapshot) (HistoryComparison, error) {
	if history.Database != "" && history.Database != database {
		return HistoryComparison{}, fmt.Errorf("history database %s differs from %s", history.Database, database)
	}
	repo := map[string]Definition{}
	checksums := map[string]string{}
	for _, m := range defs {
		if m.Database != database {
			continue
		}
		if _, dup := repo[m.ID]; dup {
			return HistoryComparison{}, fmt.Errorf("duplicate migration %s for database %s", m.ID, database)
		}
		fp, err := Fingerprint(m)
		if err != nil {
			return HistoryComparison{}, err
		}
		repo[m.ID] = m
		checksums[m.ID] = fp
	}
	applied := map[string]AppliedMigration{}
	for _, e := range history.Entries {
		if e.Database != "" && e.Database != database {
			return HistoryComparison{}, fmt.Errorf("history entry %s targets database %s, expected %s", e.ID, e.Database, database)
		}
		if strings.TrimSpace(e.ID) == "" {
			return HistoryComparison{}, fmt.Errorf("history entry has empty migration id")
		}
		if _, dup := applied[e.ID]; dup {
			return HistoryComparison{}, fmt.Errorf("history contains migration %s more than once", e.ID)
		}
		applied[e.ID] = e
	}

	ids := make([]string, 0, len(repo)+len(applied))
	seen := map[string]bool{}
	for id := range repo {
		ids = append(ids, id)
		seen[id] = true
	}
	for id := range applied {
		if !seen[id] {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)

	out := HistoryComparison{Database: database}
	for _, id := range ids {
		_, inRepo := repo[id]
		a, inHistory := applied[id]
		e := HistoryComparisonEntry{Database: database, ID: id}
		if inRepo {
			e.RepositoryChecksum = checksums[id]
		}
		if inHistory {
			e.AppliedChecksum = a.Checksum
			e.AppliedAt = a.AppliedAt
			e.Tool = a.Tool
		}
		switch {
		case !inRepo && inHistory:
			e.State = RepoOrphanApplied
		case inRepo && !inHistory:
			e.State = RepoPending
		case a.State == AppliedFailed:
			e.State = RepoFailed
		case a.Checksum != "" && e.RepositoryChecksum != a.Checksum:
			e.State = RepoDiverged
		default:
			e.State = RepoApplied
		}
		out.Entries = append(out.Entries, e)
	}
	return out, nil
}
