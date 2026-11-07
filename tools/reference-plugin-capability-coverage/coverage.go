// Package plugincoverage reports which declared plugin capabilities are actually
// selected by an environment, and which are not.
//
// The question is deliberately narrow: a plugin may declare a capability that
// no Environment binds. That is not automatically a defect (a bundle may ship
// candidates for another environment), but a reader cannot currently tell the
// difference between "not needed here" and "declared and silently unused".
//
// This reference reads the accepted World only. It does not resolve providers,
// does not select an implementation, does not grant authority, and does not
// re-derive the plugin lock.
package plugincoverage

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/ir"
)

const (
	// Schema identifies the report. It is a reference output, not a product
	// contract.
	Schema = "tei.reference.plugin-capability-coverage/v0"
	// Authority states what the report is allowed to claim.
	Authority = "derived-read-only"

	RuleID      = "tei.reference.plugin-capability-coverage"
	RuleVersion = "v0"
)

// rulePolicy is covered by the fingerprint. Changing what counts as "selected"
// without changing this text must fail review.
const rulePolicy = `plugin-capability-coverage/v0
declared: plugin.Provides[] from every Plugin definition in the World
selected: an Environment binding names the exact Capability ID the declaration
          offers, for a binding form this rule counts
binding forms counted: deployment executor, observation provider,
                      evidence provider
not counted: database instance provider (it names a provider technology such as
             postgres/oracle, not the Capability ID database.postgres)
not counted: lease policy, permission grant, external requirement, artifact
not checked: whether the selected implementation exists, conforms or is trusted
`

// Status values for one declared capability.
const (
	// Selected means an Environment's binding names this Kind+ID.
	Selected = "selected"
	// Unselected means the declaration exists but no Environment binds it.
	Unselected = "unselected"
)

// Capability is one declared plugin capability.
type Capability struct {
	Plugin  string `json:"plugin"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
	ID      string `json:"id"`
	Status  string `json:"status"`
	// Environments lists every environment that binds it.
	Environments []string `json:"environments,omitempty"`
}

// Application groups the report. It is flat per plugin, so a reader can sort by
// plugin without a second index.
type Report struct {
	Schema          string       `json:"schema"`
	Authority       string       `json:"authority"`
	RuleID          string       `json:"ruleId"`
	RuleVersion     string       `json:"ruleVersion"`
	RuleFingerprint string       `json:"ruleFingerprint"`
	Environments    []string     `json:"environments"`
	Capabilities    []Capability `json:"capabilities"`
	// Unselected counts declarations no environment binds. It is a count, not
	// a list, because the list is already in Capabilities.
	Unselected int      `json:"unselectedCount"`
	Notes      []string `json:"notes"`
}

// Derive reads the accepted World and reports the coverage.
func Derive(world ir.World) Report {
	report := Report{
		Schema:          Schema,
		Authority:       Authority,
		RuleID:          RuleID,
		RuleVersion:     RuleVersion,
		RuleFingerprint: fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(rulePolicy))),
		Capabilities:    []Capability{},
		Notes: []string{
			"unselected means no Environment binding names this Kind+ID; it is not proof that the plugin is unused",
			"a selection is a declaration, not Provider conformance, artifact trust or Host authority",
			"database-provider capabilities always appear unselected: an Environment instance names a provider technology (postgres/oracle), not the Capability ID (database.postgres)",
			"lease policies, grants and external requirements are not counted as selection",
		},
	}

	bound := bindings(world)
	envs := append([]environment.Environment(nil), world.Environments...)
	sort.Slice(envs, func(i, j int) bool { return envs[i].ID < envs[j].ID })
	for _, env := range envs {
		report.Environments = append(report.Environments, string(env.ID))
	}

	plugins := append([]pluginDefinition(nil), definitions(world)...)
	sort.Slice(plugins, func(i, j int) bool { return plugins[i].id < plugins[j].id })
	for _, p := range plugins {
		for _, capability := range p.provides {
			key := string(capability.kind) + "\x00" + string(capability.id)
			entry := Capability{
				Plugin: p.id, Version: p.version,
				Kind: string(capability.kind), ID: string(capability.id),
				Status: Unselected,
			}
			if names, ok := bound[key]; ok && len(names) > 0 {
				entry.Status = Selected
				entry.Environments = names
			} else {
				report.Unselected++
			}
			report.Capabilities = append(report.Capabilities, entry)
		}
	}
	sort.Slice(report.Capabilities, func(i, j int) bool {
		a, b := report.Capabilities[i], report.Capabilities[j]
		if a.Plugin != b.Plugin {
			return a.Plugin < b.Plugin
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.ID < b.ID
	})
	return report
}

// pluginDefinition is the small part of a plugin definition this package needs.
// Keeping it here avoids exporting a second structural view of the plugin.
type pluginDefinition struct {
	id       string
	version  string
	provides []pluginCapability
}

type pluginCapability struct {
	kind core.CanonicalID
	id   core.CanonicalID
}

func definitions(world ir.World) []pluginDefinition {
	out := make([]pluginDefinition, 0, len(world.Plugins))
	for _, def := range world.Plugins {
		entry := pluginDefinition{id: string(def.ID), version: def.Version}
		for _, capability := range def.Provides {
			entry.provides = append(entry.provides, pluginCapability{
				kind: core.CanonicalID(capability.Kind), id: capability.ID,
			})
		}
		out = append(out, entry)
	}
	return out
}

// bindings collects, per environment, the Kind+ID pairs an environment binds.
// Only the binding forms the rule policy lists are counted.
func bindings(world ir.World) map[string][]string {
	out := map[string][]string{}
	record := func(kind, id, env string) {
		if id == "" {
			return
		}
		key := kind + "\x00" + id
		for _, existing := range out[key] {
			if existing == env {
				return
			}
		}
		out[key] = append(out[key], env)
	}
	for _, env := range world.Environments {
		name := string(env.ID)
		// Database instances are NOT counted. An instance names a provider
		// technology ("postgres"), not the Capability ID ("database.postgres"),
		// so counting it would report a false selection.
		for _, binding := range env.DeploymentExecutors {
			record(string("deployment-executor"), string(binding.Executor), name)
		}
		for _, binding := range env.ObservationProviders {
			record(string("observation-provider"), string(binding.Provider), name)
		}
		for _, binding := range env.EvidenceProviders {
			record(string("evidence-provider"), string(binding.Provider), name)
		}
	}
	for key := range out {
		sort.Strings(out[key])
	}
	return out
}

// Marshal renders the report as stable, indented JSON.
func (r Report) Marshal() ([]byte, error) { return json.MarshalIndent(r, "", "  ") }
