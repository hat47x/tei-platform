// Package capabilitydegradation is a read-only reference for #2096.
//
// It compares the guarantees each generated operation requires with the
// capabilities an existing database or provider was observed to have. When a
// capability is missing, the operation is withheld or explicitly degraded; it
// is never generated with a silently weaker guarantee and never certified as
// equivalent. It does not modify a database or generate code.
package capabilitydegradation

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

const (
	InputSchema  = "tei.reference.capability-degradation-input/v0"
	ReportSchema = "tei.reference.capability-degradation-report/v0"

	StatusPassed     = "passed"
	StatusDegraded   = "degraded"
	StatusUnresolved = "unresolved"
	StatusBlocked    = "blocked"

	OperationGenerated  = "generated"
	OperationWithheld   = "withheld"
	OperationDegraded   = "degraded"
	OperationRejected   = "rejected"
	OperationUnresolved = "unresolved"

	ActionRead   = "read"
	ActionInsert = "insert"
	ActionUpdate = "update"
	ActionDelete = "delete"

	// CapabilityConflictDetection is the column that lets an update compare the
	// version it read with the version it writes.
	CapabilityConflictDetection = "optimistic-lock-column"
)

// Table is one existing table and the capabilities observed on it. Observed is
// false when the table was not inspected; a missing capability then stays
// unknown instead of being recorded as a confirmed absence.
type Table struct {
	ID           string   `json:"id"`
	Observed     bool     `json:"observed"`
	Capabilities []string `json:"capabilities"`
}

// Operation is one operation a generator would produce.
type Operation struct {
	ID               string   `json:"id"`
	Table            string   `json:"table"`
	Action           string   `json:"action"`
	Requires         []string `json:"requires"`
	AllowDegradation bool     `json:"allowDegradation,omitempty"`
	WeakerGuarantee  string   `json:"weakerGuarantee,omitempty"`
}

// Input is the model the check reads.
type Input struct {
	Schema     string      `json:"schema"`
	TargetID   string      `json:"targetId"`
	Tables     []Table     `json:"tables"`
	Operations []Operation `json:"operations"`
}

// OperationResult is the disposition of one operation.
type OperationResult struct {
	ID        string   `json:"id"`
	Table     string   `json:"table"`
	Action    string   `json:"action"`
	Status    string   `json:"status"`
	Missing   []string `json:"missing,omitempty"`
	Guarantee string   `json:"weakerGuarantee,omitempty"`
	Reason    string   `json:"reason,omitempty"`
}

// Degradation records a reduced operation and the confirmed missing
// capabilities behind it.
type Degradation struct {
	Operation       string   `json:"operation"`
	Table           string   `json:"table"`
	Missing         []string `json:"missing"`
	Disposition     string   `json:"disposition"`
	WeakerGuarantee string   `json:"weakerGuarantee,omitempty"`
	Confirmed       bool     `json:"confirmed"`
}

// Unresolved records an operation whose table was not observed.
type Unresolved struct {
	Operation string `json:"operation"`
	Table     string `json:"table"`
	Reason    string `json:"reason"`
}

// Report is the check result.
type Report struct {
	Schema              string            `json:"schema"`
	Status              string            `json:"status"`
	TargetID            string            `json:"targetId"`
	Operations          []OperationResult `json:"operations"`
	Degradations        []Degradation     `json:"degradations,omitempty"`
	Unresolved          []Unresolved      `json:"unresolved,omitempty"`
	Problems            []string          `json:"problems,omitempty"`
	Equivalence         string            `json:"equivalence"`
	CertifiedEquivalent bool              `json:"certifiedEquivalent"`
	Note                string            `json:"note"`
}

// LoadInput reads an input file.
func LoadInput(path string) (Input, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Input{}, err
	}
	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		return Input{}, fmt.Errorf("decode %s: %w", path, err)
	}
	return input, nil
}

// Check classifies each operation and records the reduced ones. A missing
// capability on an observed table is a confirmed fact; an unobserved table is
// unresolved and is not recorded as a degradation.
func Check(input Input) Report {
	report := Report{
		Schema:              ReportSchema,
		TargetID:            input.TargetID,
		Equivalence:         "not-certified",
		CertifiedEquivalent: false,
		Note:                "縮退は同等の認証ではありません。不足する能力は、観測済みの表では確認済みの事実、未観測の表では未確認として扱います。",
	}

	var problems []string
	if input.Schema != InputSchema {
		problems = append(problems, fmt.Sprintf("schema is %q, want %q", input.Schema, InputSchema))
	}

	tables := make(map[string]Table, len(input.Tables))
	for _, table := range input.Tables {
		if table.ID == "" {
			problems = append(problems, "table with empty id")
			continue
		}
		if _, exists := tables[table.ID]; exists {
			problems = append(problems, fmt.Sprintf("duplicate table id %q", table.ID))
			continue
		}
		tables[table.ID] = table
	}

	var results []OperationResult
	var degradations []Degradation
	var unresolved []Unresolved
	for _, operation := range input.Operations {
		result := OperationResult{ID: operation.ID, Table: operation.Table, Action: operation.Action}

		if operation.ID == "" {
			problems = append(problems, "operation with empty id")
			continue
		}
		if !validAction(operation.Action) {
			result.Status = OperationRejected
			result.Reason = fmt.Sprintf("unknown action %q", operation.Action)
			problems = append(problems, fmt.Sprintf("operation %q: %s", operation.ID, result.Reason))
			results = append(results, result)
			continue
		}
		if len(operation.Requires) == 0 {
			result.Status = OperationRejected
			result.Reason = "operation must declare the guarantees it requires"
			problems = append(problems, fmt.Sprintf("operation %q: %s", operation.ID, result.Reason))
			results = append(results, result)
			continue
		}
		if needsConflictDetection(operation.Action) && !contains(operation.Requires, CapabilityConflictDetection) {
			result.Status = OperationRejected
			result.Reason = fmt.Sprintf("%s must require %s", operation.Action, CapabilityConflictDetection)
			problems = append(problems, fmt.Sprintf("operation %q: %s", operation.ID, result.Reason))
			results = append(results, result)
			continue
		}

		table, exists := tables[operation.Table]
		if !exists {
			result.Status = OperationRejected
			result.Reason = fmt.Sprintf("table %q is not declared", operation.Table)
			problems = append(problems, fmt.Sprintf("operation %q: %s", operation.ID, result.Reason))
			results = append(results, result)
			continue
		}

		if !table.Observed {
			result.Status = OperationUnresolved
			result.Reason = "table capabilities were not observed"
			unresolved = append(unresolved, Unresolved{Operation: operation.ID, Table: table.ID, Reason: result.Reason})
			results = append(results, result)
			continue
		}

		missing := missingCapabilities(operation.Requires, table.Capabilities)
		result.Missing = missing
		if len(missing) == 0 {
			if operation.AllowDegradation || operation.WeakerGuarantee != "" {
				result.Status = OperationRejected
				result.Reason = "degradation declared although no capability is missing"
				problems = append(problems, fmt.Sprintf("operation %q: %s", operation.ID, result.Reason))
				results = append(results, result)
				continue
			}
			result.Status = OperationGenerated
			results = append(results, result)
			continue
		}

		if operation.AllowDegradation && operation.WeakerGuarantee != "" {
			result.Status = OperationDegraded
			result.Guarantee = operation.WeakerGuarantee
			result.Reason = "explicit weaker guarantee accepted by the definition"
			degradations = append(degradations, Degradation{
				Operation:       operation.ID,
				Table:           table.ID,
				Missing:         missing,
				Disposition:     OperationDegraded,
				WeakerGuarantee: operation.WeakerGuarantee,
				Confirmed:       true,
			})
			results = append(results, result)
			continue
		}

		result.Status = OperationWithheld
		result.Reason = "required capability is missing; the operation is not generated"
		degradations = append(degradations, Degradation{
			Operation:   operation.ID,
			Table:       table.ID,
			Missing:     missing,
			Disposition: OperationWithheld,
			Confirmed:   true,
		})
		results = append(results, result)
	}

	sort.Slice(results, func(i, j int) bool { return results[i].ID < results[j].ID })
	sort.Slice(degradations, func(i, j int) bool { return degradations[i].Operation < degradations[j].Operation })
	sort.Slice(unresolved, func(i, j int) bool { return unresolved[i].Operation < unresolved[j].Operation })
	report.Operations = results
	report.Degradations = degradations
	report.Unresolved = unresolved
	report.Problems = problems

	switch {
	case len(problems) > 0:
		report.Status = StatusBlocked
	case len(unresolved) > 0:
		report.Status = StatusUnresolved
	case len(degradations) > 0:
		report.Status = StatusDegraded
	default:
		report.Status = StatusPassed
	}
	return report
}

// GeneratedOperations returns the identifiers of the operations that would be
// generated. Withheld, degraded, rejected, and unresolved operations are not
// included.
func GeneratedOperations(report Report) []string {
	var ids []string
	for _, operation := range report.Operations {
		if operation.Status == OperationGenerated {
			ids = append(ids, operation.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

func missingCapabilities(requires, capabilities []string) []string {
	have := make(map[string]bool, len(capabilities))
	for _, capability := range capabilities {
		have[capability] = true
	}
	var missing []string
	for _, requirement := range requires {
		if !have[requirement] {
			missing = append(missing, requirement)
		}
	}
	sort.Strings(missing)
	return missing
}

func needsConflictDetection(action string) bool {
	return action == ActionUpdate || action == ActionDelete
}

func validAction(action string) bool {
	switch action {
	case ActionRead, ActionInsert, ActionUpdate, ActionDelete:
		return true
	default:
		return false
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
