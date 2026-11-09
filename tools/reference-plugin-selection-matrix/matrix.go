// Package pluginselectionmatrix reports, for every binding an environment
// declares, which declared plugin capability can serve it, and whether that is
// missing or ambiguous.
//
// The workspace validation already answers this for database instances and
// migration providers (DATABASE_PROVIDER_PLUGIN_MISSING / _AMBIGUOUS etc.). This
// reference asks the same question over every binding form at once, so a reader
// can see the whole selection surface instead of one diagnostic at a time. It
// reads the accepted World only and grants nothing.
package selectionmatrix

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/plugin"
)

const (
	// Schema identifies the report. It is a reference output, not a product
	// contract.
	Schema = "tei.reference.plugin-selection-matrix/v0"
	// Authority states what the report is allowed to claim.
	Authority = "derived-read-only"

	RuleID      = "tei.reference.plugin-selection-matrix"
	RuleVersion = "v0"

	// Per-binding verdicts. They mirror the vocabulary of the existing
	// validation diagnostics without replacing them.
	VerdictSelected  = "selected"
	VerdictMissing   = "missing"
	VerdictAmbiguous = "ambiguous"
)

// rulePolicy is covered by the fingerprint. It records exactly what is
// compared, so a change in the matching rule without a policy change fails
// review.
const rulePolicy = `plugin-selection-matrix/v0
binding forms:
  deployment executor   for action        -> Capability.SupportsAction
  observation provider  for observation   -> Capability.SupportsObservation
  evidence provider     for evidence      -> Capability.SupportsEvidence
  database instance     for technology    -> Capability.SupportsDatabaseProvider
  migration provider    for tool          -> Capability.SupportsMigrationTool
  environment feature                     -> Capability.SupportsFeature
matching uses the declared support sets, never the binding name as text
missing: no declared capability supports the binding
ambiguous: more than one declared capability supports the binding
feature gap: the workspace validation only checks editor.sql_analysis today;
             editor.database_workbench is reported here but no product
             diagnostic is emitted for it (an observation, not a diagnosis)
not checked: provider conformance, artifact trust, host authority, runtime
`

// Capability names one declared capability that supports a binding.
type Capability struct {
	Plugin string `json:"plugin"`
	ID     string `json:"id"`
}

// Selection is one binding an environment declares, and what can serve it.
type Selection struct {
	Environment  string       `json:"environment"`
	Form         string       `json:"form"`
	Kind         string       `json:"kind"`
	Binding      string       `json:"binding"`
	Verdict      string       `json:"verdict"`
	Capabilities []Capability `json:"capabilities"`
	Detail       string       `json:"detail,omitempty"`
}

// Report is the whole matrix.
type Report struct {
	Schema          string      `json:"schema"`
	Authority       string      `json:"authority"`
	RuleID          string      `json:"ruleId"`
	RuleVersion     string      `json:"ruleVersion"`
	RuleFingerprint string      `json:"ruleFingerprint"`
	Environments    []string    `json:"environments"`
	Selections      []Selection `json:"selections"`
	Missing         int         `json:"missingCount"`
	Ambiguous       int         `json:"ambiguousCount"`
	Notes           []string    `json:"notes"`
}

// Derive reads the accepted World and builds the matrix.
func Derive(world ir.World) Report {
	report := Report{
		Schema:          Schema,
		Authority:       Authority,
		RuleID:          RuleID,
		RuleVersion:     RuleVersion,
		RuleFingerprint: fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(rulePolicy))),
		Selections:      []Selection{},
		Notes: []string{
			"a selection is a declaration match, not provider conformance, artifact trust or host authority",
			"ambiguous is reported, never resolved by load order or name similarity",
			"the workspace validation owns the authority to reject these; this report only lists them",
			"editor.database_workbench is listed as missing here, but the workspace validation checks only editor.sql_analysis today; do not read this report as a product error",
		},
	}

	declared := declaredCapabilities(world)
	for _, name := range environmentNames(world) {
		report.Environments = append(report.Environments, name)
	}
	envs := append([]environment.Environment(nil), world.Environments...)
	sort.Slice(envs, func(i, j int) bool { return envs[i].ID < envs[j].ID })

	for _, env := range envs {
		name := string(env.ID)
		for _, binding := range env.DeploymentExecutors {
			report.add(name, "deployment executor", "action", string(binding.Action),
				declared.find(func(c plugin.Capability) bool {
					return c.Kind == plugin.CapabilityDeploymentExecutor && c.SupportsAction(binding.Action)
				}))
		}
		for _, binding := range env.ObservationProviders {
			report.add(name, "observation provider", "observation", string(binding.Kind),
				declared.find(func(c plugin.Capability) bool {
					return c.Kind == plugin.CapabilityObservationProvider && c.SupportsObservation(binding.Kind)
				}))
		}
		for _, binding := range env.EvidenceProviders {
			report.add(name, "evidence provider", "evidence", string(binding.Kind),
				declared.find(func(c plugin.Capability) bool {
					return c.Kind == plugin.CapabilityEvidenceProvider && c.SupportsEvidence(binding.Kind)
				}))
		}
		for _, instance := range env.Instances {
			report.add(name, "database instance", "database-provider", string(instance.Provider),
				declared.find(func(c plugin.Capability) bool {
					return c.Kind == plugin.CapabilityDatabaseProvider && c.SupportsDatabaseProvider(instance.Provider)
				}))
		}
		for _, feature := range env.Features {
			report.add(name, "environment feature", "feature", string(feature),
				declared.find(func(c plugin.Capability) bool {
					return c.Kind == plugin.CapabilityEditorIntelligence && c.SupportsFeature(feature)
				}))
		}
	}
	for _, provider := range world.MigrationProviders {
		// A migration provider declares a tool for a database. The migration
		// capability that supports the tool serves it.
		if provider.Tool == "" {
			continue
		}
		report.add("(workspace)", "migration provider", string(provider.Tool), string(provider.Database),
			declared.find(func(c plugin.Capability) bool {
				return c.Kind == plugin.CapabilityMigrationProvider && c.SupportsMigrationTool(provider.Tool)
			}))
	}

	sort.Slice(report.Selections, func(i, j int) bool {
		a, b := report.Selections[i], report.Selections[j]
		if a.Environment != b.Environment {
			return a.Environment < b.Environment
		}
		if a.Form != b.Form {
			return a.Form < b.Form
		}
		if a.Binding != b.Binding {
			return a.Binding < b.Binding
		}
		return a.Kind < b.Kind
	})
	return report
}

func (r *Report) add(env, form, kind, binding string, matches []Capability) {
	if matches == nil {
		// Encode [] rather than null so a reader never distinguishes "no
		// capability" from "the field was absent".
		matches = []Capability{}
	}
	entry := Selection{
		Environment: env, Form: form, Kind: kind, Binding: binding,
		Capabilities: matches,
	}
	switch len(matches) {
	case 0:
		entry.Verdict = VerdictMissing
		entry.Detail = "no declared capability supports this binding"
		r.Missing++
	case 1:
		entry.Verdict = VerdictSelected
	default:
		entry.Verdict = VerdictAmbiguous
		entry.Detail = "more than one declared capability supports this binding"
		r.Ambiguous++
	}
	r.Selections = append(r.Selections, entry)
}

type declaredSet []declaredCapability

type declaredCapability struct {
	plugin string
	value  plugin.Capability
}

func (d declaredSet) find(predicate func(plugin.Capability) bool) []Capability {
	var out []Capability
	for _, entry := range d {
		if predicate(entry.value) {
			out = append(out, Capability{Plugin: entry.plugin, ID: string(entry.value.ID)})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Plugin != out[j].Plugin {
			return out[i].Plugin < out[j].Plugin
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func declaredCapabilities(world ir.World) declaredSet {
	out := declaredSet{}
	defs := append([]plugin.Definition(nil), world.Plugins...)
	sort.Slice(defs, func(i, j int) bool { return defs[i].ID < defs[j].ID })
	for _, def := range defs {
		for _, capability := range def.Provides {
			out = append(out, declaredCapability{plugin: string(def.ID), value: capability})
		}
	}
	return out
}

func environmentNames(world ir.World) []string {
	out := make([]string, 0, len(world.Environments))
	for _, env := range world.Environments {
		out = append(out, string(env.ID))
	}
	sort.Strings(out)
	return out
}

// Marshal renders the report as stable, indented JSON.
func (r Report) Marshal() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
