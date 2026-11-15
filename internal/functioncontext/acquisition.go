package functioncontext

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/driver"
	"tei/internal/semantic"
)

// RelationObservation is what the Provider reported about one relation the plan reads or the
// Workspace declares. RowSecurityActive is the Provider's answer for THIS session, not the
// catalog flag (a catalog read can be stale inside a snapshot).
type RelationObservation struct {
	Name              string `json:"name"`
	Found             bool   `json:"found"`
	RelKind           string `json:"relKind,omitempty"`
	RowSecurityActive bool   `json:"rowSecurityActive"`
}

// AcquisitionObservation is what the host observed about the read it is about to do, on the
// same transaction. It holds facts only. The meaning is decided by assess below.
//
// LockedTables are the declared relations locked ACCESS SHARE before these observations, so
// the observed state cannot change under the read. LockNote is non-empty when that failed.
type AcquisitionObservation struct {
	SystemIdentifier string                `json:"systemIdentifier,omitempty"`
	LockedTables     []string              `json:"lockedTables,omitempty"`
	LockNote         string                `json:"lockNote,omitempty"`
	PlanRelations    []string              `json:"planRelations,omitempty"`
	PlanNodes        []string              `json:"planNodes,omitempty"`
	Relations        []RelationObservation `json:"relations,omitempty"`
	// PlanNote is non-empty when the plan could not be observed. Nothing is inferred then.
	PlanNote string `json:"planNote,omitempty"`
}

const (
	VisibilityRLSActive   = "rls-active"
	VisibilityRLSInactive = "rls-inactive"

	ScanSingle     = "single-relation"
	ScanMulti      = "multi-relation"
	ScanUnobserved = "unobserved"

	KeyUnique       = "unique"
	KeyDuplicate    = "duplicate"
	KeyNotCheckable = "not-checkable"

	// OpenNativePredicate is always open for a native SQL Query: what rows the SQL keeps is
	// decided by its text and by objects outside it (functions, session settings), which the
	// plan does not show. It is why coverage is never complete here.
	OpenNativePredicate = "native-sql-predicate-not-closed"
)

// planNodes whose presence means rows may be dropped or reshaped in a way a closed relation
// observation does not describe. Each is a reason to say "partial", not "unknown".
var partialNodeReasons = map[string]string{
	"Limit":         "limit-in-plan",
	"Function Scan": "function-scan",
	"Foreign Scan":  "foreign-scan",
	"Append":        "append-or-partitioned",
	"Merge Append":  "append-or-partitioned",
	"Custom Scan":   "custom-scan",
}

var joinNodes = map[string]bool{"Nested Loop": true, "Hash Join": true, "Merge Join": true}

// assess turns the observation into: coverage (partial when a restriction is SEEN, otherwise
// unestablished), the visibility word, the open list, and the scan shape. It can lower
// coverage to partial; it can never raise it to complete.
func assess(declared driver.DeclaredBrowseRelations, obs *AcquisitionObservation) (semantic.Coverage, []string, []string, string, string) {
	open := []string{OpenNativePredicate}
	if obs == nil {
		return semantic.CoverageUnestablished, nil, append(open, "acquisition-not-observed"), NotObserved, ScanUnobserved
	}
	var reasons []string
	visibility := NotObserved
	if len(obs.LockedTables) == 0 || obs.LockNote != "" {
		open = append(open, "not-locked-before-observation")
	}
	if obs.SystemIdentifier == "" {
		open = append(open, "system-identifier-unobserved")
	}
	scan := ScanUnobserved
	if obs.PlanNote != "" {
		open = append(open, "plan-unobserved")
	} else {
		scan = ScanSingle
		if len(obs.PlanRelations) != 1 {
			scan = ScanMulti
		}
		for _, node := range obs.PlanNodes {
			if reason, ok := partialNodeReasons[node]; ok {
				reasons = append(reasons, reason)
			}
			if joinNodes[node] {
				scan = ScanMulti
			}
		}
		if len(obs.PlanRelations) == 0 {
			// No relation read at all: nothing ties the rows to a declared relation.
			reasons = append(reasons, "no-relation-in-plan")
		}
	}

	rls := map[string]bool{}
	anyRLS, sawRelation := false, false
	for _, rel := range obs.Relations {
		sawRelation = true
		rls[rel.Name] = rel.RowSecurityActive
		if rel.RowSecurityActive {
			anyRLS = true
			reasons = append(reasons, "row-security-active:"+rel.Name)
		}
		if rel.Found && rel.RelKind != "r" {
			reasons = append(reasons, fmt.Sprintf("relation-kind:%s=%s", rel.Name, rel.RelKind))
		}
	}
	if sawRelation {
		visibility = VisibilityRLSInactive
		if anyRLS {
			visibility = VisibilityRLSActive
		}
	}

	switch declared.Status {
	case driver.DeclaredRelationsResolved:
		planned := map[string]bool{}
		for _, name := range obs.PlanRelations {
			planned[name] = true
		}
		declaredSet := map[string]bool{}
		for _, name := range declared.Tables {
			declaredSet[name] = true
			if obs.PlanNote == "" && !planned[name] {
				reasons = append(reasons, "declared-relation-not-read:"+name)
			}
		}
		for _, name := range obs.PlanRelations {
			if !declaredSet[name] {
				reasons = append(reasons, "relation-not-declared:"+name)
			}
		}
	default:
		status := string(declared.Status)
		if status == "" {
			status = string(driver.DeclaredRelationsNone)
		}
		open = append(open, "declared-relations-"+status)
	}

	reasons = uniqueSorted(reasons)
	if len(reasons) > 0 {
		return semantic.CoveragePartial, reasons, uniqueSorted(open), visibility, scan
	}
	return semantic.CoverageUnestablished, nil, uniqueSorted(open), visibility, scan
}

// declaredKey reports whether the Owner-declared primary key columns are present in the
// mapped result and, when they are, whether their values are unique among the rows read. It
// says nothing about a column the SQL aliased or computed.
func declaredKey(declared driver.DeclaredBrowseRelations, columns []Column, rows [][]Cell) string {
	if declared.Status != driver.DeclaredRelationsResolved || len(declared.PrimaryKey) == 0 {
		return KeyNotCheckable
	}
	index := make([]int, 0, len(declared.PrimaryKey))
	for _, attribute := range declared.PrimaryKey {
		found := -1
		for i, column := range columns {
			if column.Attribute == attribute {
				found = i
				break
			}
		}
		if found < 0 {
			return KeyNotCheckable
		}
		index = append(index, found)
	}
	seen := map[string]bool{}
	for _, row := range rows {
		var parts []string
		for _, i := range index {
			parts = append(parts, fmt.Sprintf("%s:%v", row[i].State, row[i].Value))
		}
		key := strings.Join(parts, "\x1f")
		if seen[key] {
			return KeyDuplicate
		}
		seen[key] = true
	}
	return KeyUnique
}

func uniqueSorted(in []string) []string {
	set := map[string]bool{}
	for _, s := range in {
		set[s] = true
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}
