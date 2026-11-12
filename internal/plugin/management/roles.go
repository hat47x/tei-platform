package management

import (
	"sort"

	"tei/internal/environment"
	"tei/internal/plugin"
)

// Role representation. A management target the plugin contract cannot express
// today is reported as unrepresented instead of being invented as a new
// CapabilityKind. The set is derived from plugin.CapabilityKinds, so it cannot
// drift from the resolver's accepted vocabulary.
const (
	RoleRepresented = "represented"
	RolePartial     = "partial"
	RoleNotAdopted  = "role-contract-not-adopted"
	RoleBuildInput  = "build-input-axis"
	RoleExternal    = "external-requirement"
	RoleUIKit       = "ui-kit-axis"
)

// RoleStatus is one management target and how the plugin contract currently
// expresses it.
type RoleStatus struct {
	Target          string   `json:"target"`
	Status          string   `json:"status"`
	CapabilityKinds []string `json:"capabilityKinds,omitempty"`
	Note            string   `json:"note,omitempty"`
}

// roleStatuses is the closed table of management targets. Aralger
// CapabilityKind added to plugin.CapabilityKinds is not silently presented as
// an adopted role: the table names the kinds it actually observed, so the
// report shows what exists rather than what is planned.
func roleStatuses() []RoleStatus {
	available := map[string]bool{}
	for _, kind := range plugin.CapabilityKinds {
		available[string(kind)] = true
	}

	var out []RoleStatus
	add := func(target, status string, kinds []string, note string) {
		var observed []string
		for _, kind := range kinds {
			if available[kind] {
				observed = append(observed, kind)
			}
		}
		entry := RoleStatus{Target: target, Status: status, CapabilityKinds: observed, Note: note}
		if len(observed) == 0 && status == RoleRepresented {
			// A represented target with no observed kind is not represented.
			entry.Status = RoleNotAdopted
		}
		out = append(out, entry)
	}

	add("language-plugin", RoleNotAdopted, []string{}, "Language Plugin の契約は未採用（#988/#1018）")
	add("operator", RoleNotAdopted, []string{}, "Operator の契約は未採用（#988）")
	add("data-provider", RoleRepresented, []string{string(plugin.CapabilityDatabaseProvider), string(plugin.CapabilityContractQueryExecutor)}, "")
	add("target-generator", RoleBuildInput, []string{}, "Build 入力の軸でありプラグイン行とは別（#943/#1121）")
	add("ui-studio-extension", RolePartial, []string{string(plugin.CapabilityEditorIntelligence)}, "UI Kit は別軸（#997）")
	add("external-integration", RoleExternal, []string{}, "ExternalRequirement{scope,kind,name} として宣言できる")
	add("test-diagnostic", RoleNotAdopted, []string{}, "プラグイン契約に種別が無い（#4225/#3267/#1180）")
	add("deployment-runtime", RoleRepresented, []string{
		string(plugin.CapabilityDeploymentExecutor),
		string(plugin.CapabilityObservationProvider),
		string(plugin.CapabilityEvidenceProvider),
		string(plugin.CapabilityMigrationProvider),
	}, "配備・観測・根拠・移行は既存の実行能力")

	// A kind that exists but is not named by any target is reported too, so a
	// new kind never disappears from this table.
	named := map[string]bool{}
	for _, entry := range out {
		for _, kind := range entry.CapabilityKinds {
			named[kind] = true
		}
	}
	var unlisted []string
	for _, kind := range plugin.CapabilityKinds {
		if !named[string(kind)] {
			unlisted = append(unlisted, string(kind))
		}
	}
	if len(unlisted) > 0 {
		sort.Strings(unlisted)
		out = append(out, RoleStatus{
			Target: "capability-kind-not-in-management-table", Status: RoleRepresented,
			CapabilityKinds: unlisted,
			Note:            "管理対象の分類表に無い種別。新しい運用対象として扱わない",
		})
	}
	return out
}

// roleCoverage maps an external requirement scope to the phase it belongs to.
// It is a projection of plugin.ExternalScopes so the report cannot claim a
// scope the contract does not accept.
func roleCoverage() []string {
	return append([]string(nil), plugin.ExternalScopes...)
}

// featureCoverage reports the declared product features. A feature the
// environment selects is not a plugin: it is a capability flag the contract
// already names.
func featureCoverage(env environment.Environment) []string {
	out := make([]string, 0, len(env.Features))
	for _, f := range env.Features {
		out = append(out, string(f))
	}
	sort.Strings(out)
	return out
}
