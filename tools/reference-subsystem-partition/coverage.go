package partition

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// SubjectKind classifies the entity that was examined.
const (
	SubjectFunction = "Function"
	SubjectData     = "Data"
	SubjectRelation = "Relation"
	SubjectSource   = "Source"
)

// CoverageEntry records the provenance state of one subject.
type CoverageEntry struct {
	SubjectID     string         `json:"subjectId"`
	Kind          string         `json:"kind"`
	Provenance    ProvenanceKind `json:"provenance"`
	SourceLocator string         `json:"sourceLocator"`
	Staleness     string         `json:"staleness"`
	Notes         string         `json:"notes"`
}

// CoverageReport summarizes the examination status for all supplied subjects.
type CoverageReport struct {
	SchemaVersion  string          `json:"schemaVersion"`
	GeneratedAt    string          `json:"generatedAt"`
	BaseCommit     string          `json:"baseCommit"`
	AuthorityLimit string          `json:"authorityLimit"`
	Subjects       []CoverageEntry `json:"subjects"`
	Summary        map[string]int  `json:"summary"`
}

// CoverageOptions tunes coverage reporting.
type CoverageOptions struct {
	SchemaVersion string
	GeneratedAt   string
	BaseCommit    string
}

// CoverageLedgerOption is a functional option for BuildCoverageReport.
type CoverageLedgerOption func(*CoverageOptions)

// WithSchemaVersion sets the schema version on the report.
func WithSchemaVersion(v string) CoverageLedgerOption {
	return func(o *CoverageOptions) { o.SchemaVersion = v }
}

// WithReportTimestamp sets the generated-at timestamp on the report.
func WithReportTimestamp(v string) CoverageLedgerOption {
	return func(o *CoverageOptions) { o.GeneratedAt = v }
}

// WithBaseCommit sets the base commit on the report.
func WithBaseCommit(v string) CoverageLedgerOption {
	return func(o *CoverageOptions) { o.BaseCommit = v }
}

// BuildCoverageReport deterministically summarizes supplied subjects and entries.
// The output is sorted by SubjectID then provenance. Empty subjects are allowed;
// callers should not equate empty subjects with "no unresolved items".
func BuildCoverageReport(subjects []string, entries []CoverageEntry, opts ...CoverageLedgerOption) (CoverageReport, error) {
	options := CoverageOptions{
		SchemaVersion: "tei.subsystem-partition.coverage/v0",
		GeneratedAt:   "",
		BaseCommit:    "",
	}
	for _, apply := range opts {
		apply(&options)
	}

	seen := map[string]bool{}
	for _, id := range subjects {
		if id == "" || strings.TrimSpace(id) != id {
			return CoverageReport{}, fmt.Errorf("invalid subject id %q", id)
		}
		seen[strings.TrimSpace(id)] = true
	}
	for i := range entries {
		entries[i].SubjectID = strings.TrimSpace(entries[i].SubjectID)
		entries[i].Kind = strings.TrimSpace(entries[i].Kind)
		entries[i].Provenance = ProvenanceKind(strings.TrimSpace(string(entries[i].Provenance)))
		if entries[i].SubjectID == "" || entries[i].Provenance == "" {
			return CoverageReport{}, fmt.Errorf("coverage entry %d has empty subjectId or provenance", i)
		}
		seen[entries[i].SubjectID] = true
	}
	// include subjects that only appear in the subject list and not in entries.
	subjectKeys := make([]string, 0, len(seen))
	for id := range seen {
		subjectKeys = append(subjectKeys, id)
	}
	sort.Strings(subjectKeys)

	subjectsOut := make([]CoverageEntry, 0)
	for _, id := range subjectKeys {
		found := false
		for _, e := range entries {
			if e.SubjectID == id {
				found = true
				break
			}
		}
		if !found {
			subjectsOut = append(subjectsOut, CoverageEntry{
				SubjectID:  id,
				Kind:       SubjectSource,
				Provenance: ProvNotExamined,
				Notes:      "subject listed without any explicit entry",
			})
		}
	}
	subjectsOut = append(subjectsOut, entries...)
	sort.Slice(subjectsOut, func(i, j int) bool {
		if subjectsOut[i].SubjectID != subjectsOut[j].SubjectID {
			return subjectsOut[i].SubjectID < subjectsOut[j].SubjectID
		}
		return subjectsOut[i].Provenance < subjectsOut[j].Provenance
	})

	summary := map[string]int{"subjects": len(subjectsOut)}
	for _, e := range subjectsOut {
		summary[string(e.Provenance)]++
	}

	return CoverageReport{
		SchemaVersion:  options.SchemaVersion,
		GeneratedAt:    options.GeneratedAt,
		BaseCommit:     options.BaseCommit,
		AuthorityLimit: AuthorityLimit,
		Subjects:       subjectsOut,
		Summary:        summary,
	}, nil
}

// MustMarshalCoverageReport is a deterministic JSON encoder for coverage reports.
func MustMarshalCoverageReport(report CoverageReport) []byte {
	b, _ := json.Marshal(report)
	return b
}
