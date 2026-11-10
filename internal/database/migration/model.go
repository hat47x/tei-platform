package migration

import (
	"tei/internal/core"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

type RecoveryStrategy string

const (
	RecoveryForwardFix       RecoveryStrategy = "forward-fix"
	RecoveryExplicitRollback RecoveryStrategy = "explicit-rollback"
	RecoveryNotReversible    RecoveryStrategy = "not-reversible"
)

// RecoveryStrategies lists every accepted strategy in report order, so a
// diagnostic can quote the same set Valid accepts (#4094).
var RecoveryStrategies = []RecoveryStrategy{RecoveryForwardFix, RecoveryExplicitRollback, RecoveryNotReversible}

func (r RecoveryStrategy) Valid() bool {
	for _, valid := range RecoveryStrategies {
		if r == valid {
			return true
		}
	}
	return false
}

type StepKind string

const (
	StepSchema StepKind = "schema"
	StepData   StepKind = "data"
	StepVerify StepKind = "verify"
)

// StepKinds lists every accepted step kind in report order (#4094).
var StepKinds = []StepKind{StepSchema, StepData, StepVerify}

func (k StepKind) Valid() bool {
	for _, valid := range StepKinds {
		if k == valid {
			return true
		}
	}
	return false
}

type Definition struct {
	ID                       string           `json:"id"`
	Database                 core.CanonicalID `json:"database"`
	Parents                  []string         `json:"parents,omitempty"`
	BasedOnActualFingerprint string           `json:"basedOnActualFingerprint,omitempty"`
	Recovery                 RecoveryStrategy `json:"recovery"`
	Steps                    []Step           `json:"steps,omitempty"`
	Realizations             []Realization    `json:"realizations,omitempty"`
}

type Step struct {
	Kind        StepKind           `json:"kind"`
	Change      *schemadiff.Change `json:"change,omitempty"`
	SQL         *SQLStep           `json:"sql,omitempty"`
	Description string             `json:"description,omitempty"`
}

type SQLStep struct {
	Provider environment.DatabaseProvider `json:"provider"`
	Source   string                       `json:"source"`
}

type Realization struct {
	Provider   environment.DatabaseProvider `json:"provider"`
	Statements []string                     `json:"statements,omitempty"`
}
