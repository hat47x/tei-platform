package main

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/ir"
	"tei/internal/validation"
)

const explicitContractSchema = "tei.explicit-pattern-contracts/v0"

// ExplicitContractReport records only references resolved from declared Source.
// It never claims runtime permission, provider access or observed effects.
type ExplicitContractReport struct {
	SchemaVersion   string                  `json:"schemaVersion"`
	Authority       string                  `json:"authority"`
	Application     core.CanonicalID        `json:"application"`
	Pattern         core.CanonicalID        `json:"pattern"`
	Kind            application.PatternKind `json:"kind"`
	DeclaredData    core.CanonicalID        `json:"declaredData,omitempty"`
	Query           core.CanonicalID        `json:"query,omitempty"`
	Datasource      core.LocalName          `json:"datasource,omitempty"`
	TargetDatabase  core.CanonicalID        `json:"targetDatabase,omitempty"`
	Capability      core.CanonicalID        `json:"capability,omitempty"`
	BindingIdentity string                  `json:"bindingIdentity,omitempty"`
	DataOperation   string                  `json:"dataOperation,omitempty"`
	// These are matched Source declarations, not an authorized Provider route.
	WriteDatasource    core.LocalName   `json:"writeDatasource,omitempty"`
	WriteDatabase      core.CanonicalID `json:"writeDatabase,omitempty"`
	PersistenceMapping core.CanonicalID `json:"persistenceMapping,omitempty"`
	Resolved        []string                `json:"resolved"`
	Unverified      []string                `json:"unverified"`
	Executable      bool                    `json:"executable"`
}

// InspectExplicitPattern reuses the Driver's authoritative declared-reference
// resolution, without calling any Executor. A resolved Contract is not a
// provider connection, write approval, or binding from a command to Data.
func InspectExplicitPattern(w ir.World, applicationID, patternID core.CanonicalID) (ExplicitContractReport, error) {
	resolver := driver.NewResolver(w)
	pattern, err := resolver.ResolvePattern(applicationID, patternID)
	if err != nil {
		return ExplicitContractReport{}, err
	}

	report := ExplicitContractReport{
		SchemaVersion: explicitContractSchema,
		Authority:     "declared-references-only",
		Application:   applicationID,
		Pattern:       patternID,
		Kind:          pattern.Pattern.Kind,
		Resolved:      []string{},
		Unverified:    []string{},
		Executable:    false,
	}

	switch pattern.Pattern.Kind {
	case application.PatternBrowse:
		// Resolver requires an Application-owned Query and one readable
		// Datasource. A Query declaration is not a live Provider session.
		if pattern.Data == nil || pattern.Query == nil || pattern.Datasource == nil {
			return ExplicitContractReport{}, fmt.Errorf("browse Pattern %s has incomplete declared references", patternID)
		}
		report.DeclaredData = pattern.Data.ID
		report.Query = pattern.Query.ID
		report.Datasource = pattern.Datasource.Alias
		report.TargetDatabase = pattern.Datasource.TargetDatabase
		report.Resolved = []string{"data", "query", "datasource"}
		report.Unverified = []string{"query-executor", "provider-session", "current-authority", "target-read", "independent-observation"}

	case application.PatternRecord:
		if pattern.Data == nil {
			return ExplicitContractReport{}, fmt.Errorf("record Pattern %s has no declared Data", patternID)
		}
		report.DeclaredData = pattern.Data.ID
		report.Resolved = []string{"data"}
		report.Unverified = []string{"read-query", "datasource", "provider-session", "current-authority", "target-read"}

	case application.PatternCommand:
		// The command Pattern does not associate a particular Data with an
		// operation (create/update). Never infer that link from its name or
		// the Capability's output type.
		command, err := resolver.ResolveCommandBinding(applicationID, patternID)
		if err != nil {
			return ExplicitContractReport{}, err
		}
		report.Capability = command.Binding.Capability.ID
		report.BindingIdentity = command.Binding.Binding.Identity()
		report.Resolved = []string{"capability", "application-scoped-binding"}
		report.Unverified = []string{"data-operation-link", "persistence", "provider-session", "current-authority", "target-command", "independent-database-read"}
		if effect := command.Binding.Capability.Effect; effect != nil {
			// Canonical effect declarations are validated with the existing
			// World validator; neither names nor output type establish effects.
			if diagnostics := validation.Validate(w); validation.HasErrors(diagnostics) {
				return ExplicitContractReport{}, fmt.Errorf("invalid Capability data effect in Canonical World: %v", diagnostics)
			}
			report.DeclaredData = effect.Data
			report.DataOperation = string(effect.Kind)
			report.Resolved = append(report.Resolved, "data-operation-link")
			report.Unverified = report.Unverified[1:]
			mapping, err := MatchDeclaredWriteMapping(w, pattern.Application, effect.Data)
			if err != nil {
				return ExplicitContractReport{}, err
			}
			if mapping != nil {
				report.PersistenceMapping = mapping.Mapping.ID
				report.WriteDatabase = mapping.Mapping.Database
				report.WriteDatasource = mapping.Datasource.Alias
				report.Resolved = append(report.Resolved, "writable-persistence-declarations")
			}
		}

	default:
		return ExplicitContractReport{}, fmt.Errorf("Pattern %s kind %s is outside standard browse/record/command contract inspection", patternID, pattern.Pattern.Kind)
	}
	return report, nil
}
