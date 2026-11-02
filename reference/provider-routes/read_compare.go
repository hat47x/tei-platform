package providerroutes

import (
	"fmt"
	"sort"
)

// ReadSlice is a synthetic observation of one read from a declared route.
// All fields are fixture facts, not a proof that a Provider enforced them.
// Rows are a bag: duplicate occurrences and NULL/absent states are preserved.
type ReadSlice struct {
	Route          Path
	Scope          string
	Grain          string
	QueryContract  string
	SnapshotIssuer string
	SnapshotToken  string
	Complete       bool
	Rows           []ReadRow
}

type ReadRow struct {
	Identity   string
	Occurrence string
	Presence   string // "value", "null", "absent"; fixture vocabulary
	Value      string // exact fixture value bytes, only interpreted if Presence="value"
}

type ReadComparison struct {
	Status string // "fixture_consistent", "contradicted", "not_established"
	Reason string
}

// AssertedCommonCohort is an INDEPENDENT fixture assumption. It MUST NOT be
// constructed by merely comparing SnapshotToken strings across Providers.
// Production validation must live with the existing read/Host contract.
type AssertedCommonCohort struct {
	EstablishedByFixture bool
	EvidenceLabel        string
}

// CompareReadSlices checks bag equality under an externally established common
// read cohort. Equality is *fixture consistency*, never actual Provider or
// business equivalence. Two equal snapshot tokens are intentionally ignored.
func CompareReadSlices(a, b ReadSlice, cohort AssertedCommonCohort) ReadComparison {
	unknown := func(why string) ReadComparison { return ReadComparison{Status: "not_established", Reason: why} }
	if a.Route.SemanticData == "" || a.Route.SemanticData != b.Route.SemanticData ||
		a.Scope == "" || b.Scope == "" || a.Scope != b.Scope || a.Route.Scope != a.Scope || b.Route.Scope != b.Scope ||
		a.Grain == "" || a.Grain != b.Grain ||
		a.QueryContract == "" || a.QueryContract != b.QueryContract {
		return unknown("scope/grain/query-contract-unqualified")
	}
	if !cohort.EstablishedByFixture || cohort.EvidenceLabel == "" {
		return unknown("shared-cohort-not-established")
	}
	if !a.Complete || !b.Complete {
		return unknown("read-incomplete")
	}
	if !validRows(a.Rows) || !validRows(b.Rows) {
		return unknown("row-identity-or-presence-unqualified")
	}
	if hasUnobservedValue(a.Rows) || hasUnobservedValue(b.Rows) {
		return unknown("field-value-not-observed")
	}
	aRows, bRows := bagSignature(a.Rows), bagSignature(b.Rows)
	if len(aRows) != len(bRows) {
		return ReadComparison{Status: "contradicted", Reason: "complete-bags-differ"}
	}
	for i := range aRows {
		if aRows[i] != bRows[i] {
			return ReadComparison{Status: "contradicted", Reason: "complete-bags-differ"}
		}
	}
	return ReadComparison{Status: "fixture_consistent", Reason: "complete-bags-equal-under-independent-fixture-premise"}
}

func validRows(rows []ReadRow) bool {
	for _, r := range rows {
		if r.Identity == "" || r.Occurrence == "" {
			return false
		}
		if r.Presence != "value" && r.Presence != "null" && r.Presence != "absent" {
			return false
		}
	}
	return true
}

// A complete rowset does not guarantee every field was observed. SQL NULL is
// an observed presence state; "absent" means the field is not known.
func hasUnobservedValue(rows []ReadRow) bool {
	for _, row := range rows {
		if row.Presence == "absent" {
			return true
		}
	}
	return false
}

// Use length-prefixing rather than separators because fixture values and IDs
// may contain tabs, newlines, or delimiters. Count duplicated rows (bag).
func bagSignature(rows []ReadRow) []string {
	keys := make([]string, 0, len(rows))
	for _, row := range rows {
		v := row.Value
		if row.Presence != "value" {
			v = ""
		}
		keys = append(keys, fmt.Sprintf("%d:%s%d:%s%d:%s%d:%s", len(row.Identity), row.Identity,
			len(row.Occurrence), row.Occurrence, len(row.Presence), row.Presence, len(v), v))
	}
	sort.Strings(keys)
	return keys
}
