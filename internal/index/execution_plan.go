package index

import (
	"tei/internal/core"
	"tei/internal/database/plan"
)

type ExecutionPlanEntry struct {
	Application core.CanonicalID      `json:"application"`
	Plan        plan.IR               `json:"plan"`
	Mappings    []plan.LogicalMapping `json:"logicalMappings,omitempty"`
}
