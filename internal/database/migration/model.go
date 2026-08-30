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

func (r RecoveryStrategy) Valid() bool {
	return r == RecoveryForwardFix || r == RecoveryExplicitRollback || r == RecoveryNotReversible
}

type StepKind string

const (
	StepSchema StepKind = "schema"
	StepData   StepKind = "data"
	StepVerify StepKind = "verify"
)

func (k StepKind) Valid() bool { return k == StepSchema || k == StepData || k == StepVerify }

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
