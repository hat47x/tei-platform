package migration

import (
	"fmt"
	"strings"

	"tei/internal/core"
	"tei/internal/database/schemadiff"
	"tei/internal/environment"
)

func Plan(id string, database core.CanonicalID, actualFingerprint string, diff schemadiff.Diff, provider environment.DatabaseProvider) (Definition, error) {
	if strings.TrimSpace(id) == "" {
		return Definition{}, fmt.Errorf("migration id is required")
	}
	if diff.Database != string(database) {
		return Definition{}, fmt.Errorf("migration database %s differs from diff database %s", database, diff.Database)
	}
	m := Definition{ID: id, Database: database, BasedOnActualFingerprint: actualFingerprint, Recovery: RecoveryForwardFix}
	for i := range diff.Changes {
		c := diff.Changes[i]
		m.Steps = append(m.Steps, Step{Kind: StepSchema, Change: &c})
	}
	stmts, err := Render(provider, diff)
	if err != nil {
		return Definition{}, err
	}
	m.Realizations = []Realization{{Provider: provider, Statements: stmts}}
	m.Steps = append(m.Steps, Step{Kind: StepVerify, Description: "re-introspect database and compare with desired physical schema"})
	return m, nil
}
