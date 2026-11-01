// Package studioplugindisable is a read-only reference for Studio plugin
// enable/disable/remove planning (#2046). It resolves required dependencies,
// refuses a disable that would strand a dependent plugin, presents the
// dependents as an explicit candidate set, rechecks the settings version,
// declarations, source fingerprints and permission before applying, and keeps
// design sources byte-identical.
//
// It is not a Studio UI, a plugin loader, a resolver, an execution engine, or
// an accepted API. It never writes a source, installs or starts a plugin, and
// never turns a settings update into an extension-start guarantee. The real
// screen and the Host binding are outside this reference.
package studioplugindisable

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Schema identifies the plan and apply records.
const Schema = "tei.reference.studio-plugin-disable/v0"

// Actions.
const (
	ActionDisable = "disable"
	ActionRemove  = "remove"
	ActionEnable  = "enable"
)

// Plugin is one plugin and its dependency declarations. RequiredDeps are
// necessary for the plugin to work; OptionalDeps only add a feature.
type Plugin struct {
	ID           string   `json:"id"`
	Version      string   `json:"version,omitempty"`
	Installed    bool     `json:"installed"`
	Enabled      bool     `json:"enabled"`
	RequiredDeps []string `json:"requiredDeps,omitempty"`
	OptionalDeps []string `json:"optionalDeps,omitempty"`
	// RetainedData records that data remains. It never causes an automatic
	// reinstall or restart.
	RetainedData bool `json:"retainedData,omitempty"`
}

// Source is one design source file. The reference never edits it. Unknown
// attributes and child elements are carried through unchanged.
type Source struct {
	Path              string `json:"path"`
	Fingerprint       string `json:"fingerprint"`
	UnknownAttributes bool   `json:"unknownAttributes,omitempty"`
	UnknownElements   bool   `json:"unknownElements,omitempty"`
}

// State is the current Studio settings and the sources beside them.
type State struct {
	SettingsVersion string   `json:"settingsVersion"`
	GitHead         string   `json:"gitHead,omitempty"`
	Permission      string   `json:"permission"` // allowed, denied, unknown
	Plugins         []Plugin `json:"plugins"`
	Sources         []Source `json:"sources,omitempty"`
	// StandardEditorAvailable says the normal editor entry stays usable even
	// when a plugin-specific feature is lost.
	StandardEditorAvailable bool `json:"standardEditorAvailable"`
}

// Request is one user operation. It never names other plugins: the dependents
// are derived from the declarations, not from the row selection.
type Request struct {
	Action   string `json:"action"`
	PluginID string `json:"pluginId"`
}

// Snapshot is the fact set a plan was made from. Apply compares it again.
type Snapshot struct {
	SettingsVersion        string            `json:"settingsVersion"`
	GitHead                string            `json:"gitHead,omitempty"`
	DeclarationFingerprint string            `json:"declarationFingerprint"`
	SourceFingerprints     map[string]string `json:"sourceFingerprints"`
	Permission             string            `json:"permission"`
}

// Plan is the read-only proposal. Nothing is changed until Apply accepts it.
type Plan struct {
	Schema                  string   `json:"schema"`
	Action                  string   `json:"action"`
	Requested               string   `json:"requested"`
	Candidates              []string `json:"candidates"`
	Additional              []string `json:"additional,omitempty"`
	Unchanged               []string `json:"unchanged"`
	OptionalFeaturesLost    []string `json:"optionalFeaturesLost,omitempty"`
	Unresolved              []string `json:"unresolved,omitempty"`
	Cycles                  []string `json:"cycles,omitempty"`
	Reasons                 []string `json:"reasons,omitempty"`
	Refused                 bool     `json:"refused"`
	ConfirmationRequired    bool     `json:"confirmationRequired"`
	RestartRequired         []string `json:"restartRequired,omitempty"`
	SourcesPreserved        bool     `json:"sourcesPreserved"`
	StandardEditorAvailable bool     `json:"standardEditorAvailable"`
	SelectionChanged        bool     `json:"selectionChanged"`
	Snapshot                Snapshot `json:"snapshot"`
	Digest                  string   `json:"digest"`
}

// ApplyRequest carries the explicit confirmation of the expanded candidate
// set. Without it, a plan that needs additional dependents does not apply.
type ApplyRequest struct {
	Confirm bool `json:"confirm"`
}

// ApplyResult separates the settings update from the extension start.
type ApplyResult struct {
	Schema                  string          `json:"schema"`
	PlanDigest              string          `json:"planDigest"`
	SettingsApplied         bool            `json:"settingsApplied"`
	ExtensionsStarted       bool            `json:"extensionsStarted"`
	AllSucceeded            bool            `json:"allSucceeded"`
	SettingsAtomic          bool            `json:"settingsAtomic"`
	ExternalEffectAtomic    bool            `json:"externalEffectAtomic"`
	Reasons                 []string        `json:"reasons,omitempty"`
	ExtensionErrors         []string        `json:"extensionErrors,omitempty"`
	NewSettingsVersion      string          `json:"newSettingsVersion,omitempty"`
	EnabledAfter            map[string]bool `json:"enabledAfter,omitempty"`
	RemovedAfter            []string        `json:"removedAfter,omitempty"`
	SourcePreserved         bool            `json:"sourcePreserved"`
	SourcesAfter            []Source        `json:"sourcesAfter,omitempty"`
	StandardEditorAvailable bool            `json:"standardEditorAvailable"`
	PluginSpecificFeature   bool            `json:"pluginSpecificFeature"`
	AutoInstalled           bool            `json:"autoInstalled"`
	AutoStarted             bool            `json:"autoStarted"`
}

func sortedUnique(values []string) []string {
	sort.Strings(values)
	out := values[:0]
	for _, value := range values {
		if value == "" {
			continue
		}
		if len(out) == 0 || out[len(out)-1] != value {
			out = append(out, value)
		}
	}
	return out
}

func hashParts(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func validateState(state State) error {
	if strings.TrimSpace(state.SettingsVersion) == "" {
		return fmt.Errorf("settingsVersion is required")
	}
	switch state.Permission {
	case "allowed", "denied", "unknown":
	default:
		return fmt.Errorf("permission %q must be allowed, denied, or unknown", state.Permission)
	}
	seen := map[string]bool{}
	for i, p := range state.Plugins {
		if strings.TrimSpace(p.ID) == "" {
			return fmt.Errorf("plugins[%d].id is required", i)
		}
		if seen[p.ID] {
			return fmt.Errorf("plugin %q is declared twice", p.ID)
		}
		seen[p.ID] = true
	}
	seenSources := map[string]bool{}
	for i, s := range state.Sources {
		if strings.TrimSpace(s.Path) == "" || strings.TrimSpace(s.Fingerprint) == "" {
			return fmt.Errorf("sources[%d] needs path and fingerprint", i)
		}
		if seenSources[s.Path] {
			return fmt.Errorf("source %q is declared twice", s.Path)
		}
		seenSources[s.Path] = true
	}
	return nil
}

func validateRequest(req Request) error {
	switch req.Action {
	case ActionDisable, ActionRemove, ActionEnable:
	default:
		return fmt.Errorf("action %q must be disable, remove, or enable", req.Action)
	}
	if strings.TrimSpace(req.PluginID) == "" {
		return fmt.Errorf("pluginId is required")
	}
	return nil
}

func snapshotOf(state State) Snapshot {
	declarations := make([]string, 0, len(state.Plugins))
	for _, p := range state.Plugins {
		declarations = append(declarations,
			"plugin="+p.ID+"|version="+p.Version+"|installed="+boolString(p.Installed)+
				"|required="+strings.Join(sortedUnique(append([]string(nil), p.RequiredDeps...)), ",")+
				"|optional="+strings.Join(sortedUnique(append([]string(nil), p.OptionalDeps...)), ","))
	}
	sort.Strings(declarations)
	sources := map[string]string{}
	for _, s := range state.Sources {
		sources[s.Path] = s.Fingerprint
	}
	return Snapshot{
		SettingsVersion: state.SettingsVersion, GitHead: state.GitHead,
		DeclarationFingerprint: hashParts(declarations...),
		SourceFingerprints:     sources, Permission: state.Permission,
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func pluginMap(plugins []Plugin) map[string]Plugin {
	out := map[string]Plugin{}
	for _, p := range plugins {
		out[p.ID] = p
	}
	return out
}

func dependentsOf(plugins []Plugin) map[string][]string {
	out := map[string][]string{}
	for _, p := range plugins {
		for _, dep := range p.RequiredDeps {
			out[dep] = append(out[dep], p.ID)
		}
	}
	for key := range out {
		sort.Strings(out[key])
	}
	return out
}

// findCycle returns a cycle in the required-dependency graph restricted to the
// subset. A cycle is a different fact from a missing dependency.
func findCycle(byID map[string]Plugin, subset map[string]bool) []string {
	state := map[string]int{}
	var stack, cycle []string
	var dfs func(id string) bool
	dfs = func(id string) bool {
		state[id] = 1
		stack = append(stack, id)
		if p, ok := byID[id]; ok {
			for _, dep := range p.RequiredDeps {
				if !subset[dep] {
					continue
				}
				switch state[dep] {
				case 0:
					if dfs(dep) {
						return true
					}
				case 1:
					for i, value := range stack {
						if value == dep {
							cycle = append([]string(nil), stack[i:]...)
							cycle = append(cycle, dep)
							return true
						}
					}
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[id] = 2
		return false
	}
	ids := make([]string, 0, len(subset))
	for id := range subset {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if state[id] == 0 && dfs(id) {
			return cycle
		}
	}
	return nil
}

// PlanAction builds the candidate set and the diagnosis. It never mutates the
// state. A request that would strand a dependent is not refused outright: the
// dependents are shown as Additional candidates, and ConfirmationRequired asks
// the caller to accept that expanded set explicitly.
func PlanAction(state State, req Request) (Plan, error) {
	if err := validateState(state); err != nil {
		return Plan{}, err
	}
	if err := validateRequest(req); err != nil {
		return Plan{}, err
	}
	byID := pluginMap(state.Plugins)
	plan := Plan{
		Schema: Schema, Action: req.Action, Requested: req.PluginID,
		SourcesPreserved: true, StandardEditorAvailable: state.StandardEditorAvailable,
		Snapshot: snapshotOf(state),
	}
	refuse := func(reason string) (Plan, error) {
		plan.Reasons = append(plan.Reasons, reason)
		plan.Refused = true
		return finishPlan(plan)
	}
	p, ok := byID[req.PluginID]
	if !ok {
		return refuse("unknown-plugin")
	}
	if state.Permission != "allowed" {
		return refuse("permission-denied")
	}
	switch req.Action {
	case ActionEnable:
		if !p.Installed {
			return refuse("not-installed")
		}
		if p.Enabled {
			return refuse("already-enabled")
		}
		subset := map[string]bool{p.ID: true}
		for _, dep := range p.RequiredDeps {
			subset[dep] = true
		}
		if cycle := findCycle(byID, subset); cycle != nil {
			plan.Cycles = []string{strings.Join(cycle, "->")}
			return refuse("dependency-cycle")
		}
		for _, dep := range sortedUnique(append([]string(nil), p.RequiredDeps...)) {
			required, ok := byID[dep]
			if !ok {
				plan.Unresolved = append(plan.Unresolved, p.ID+":"+dep)
				return refuse("unresolved-dependency")
			}
			if !required.Installed || !required.Enabled {
				return refuse("dependency-not-enabled:" + dep)
			}
		}
		for _, dep := range sortedUnique(append([]string(nil), p.OptionalDeps...)) {
			if required, ok := byID[dep]; !ok || !required.Installed || !required.Enabled {
				plan.Reasons = append(plan.Reasons, "optional-not-available:"+dep)
			}
		}
		plan.Candidates = []string{p.ID}
		plan.RestartRequired = []string{p.ID}
		plan.Unchanged = otherIDs(state.Plugins, plan.Candidates)
		return finishPlan(plan)

	case ActionDisable, ActionRemove:
		if !p.Installed {
			return refuse("not-installed")
		}
		if !p.Enabled {
			return refuse("already-disabled")
		}
		dependents := dependentsOf(state.Plugins)
		closure := map[string]bool{p.ID: true}
		queue := []string{p.ID}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, dependent := range dependents[current] {
				if !closure[dependent] {
					closure[dependent] = true
					queue = append(queue, dependent)
				}
			}
		}
		if cycle := findCycle(byID, closure); cycle != nil {
			plan.Cycles = []string{strings.Join(cycle, "->")}
			return refuse("dependency-cycle")
		}
		enabled := map[string]bool{}
		for _, candidate := range state.Plugins {
			enabled[candidate.ID] = candidate.Enabled
		}
		var candidates, additional []string
		for id := range closure {
			if id == p.ID || enabled[id] {
				candidates = append(candidates, id)
			}
		}
		sort.Strings(candidates)
		for _, id := range candidates {
			if id != p.ID {
				additional = append(additional, id)
			}
		}
		// Missing required declarations are a different fact from a cycle.
		affected := map[string]bool{}
		for _, id := range candidates {
			affected[id] = true
		}
		for _, candidate := range state.Plugins {
			for _, dep := range candidate.RequiredDeps {
				if _, ok := byID[dep]; !ok {
					plan.Unresolved = append(plan.Unresolved, candidate.ID+":"+dep)
				}
			}
		}
		sort.Strings(plan.Unresolved)
		for _, item := range plan.Unresolved {
			id := strings.SplitN(item, ":", 2)[0]
			if affected[id] {
				return refuse("unresolved-dependency")
			}
		}
		for _, candidate := range state.Plugins {
			if affected[candidate.ID] || !candidate.Enabled {
				continue
			}
			for _, dep := range candidate.OptionalDeps {
				if closure[dep] {
					plan.OptionalFeaturesLost = append(plan.OptionalFeaturesLost, candidate.ID+":"+dep)
				}
			}
		}
		plan.OptionalFeaturesLost = sortedUnique(plan.OptionalFeaturesLost)
		plan.Candidates = candidates
		plan.Additional = additional
		plan.Unchanged = otherIDs(state.Plugins, candidates)
		plan.RestartRequired = append([]string(nil), candidates...)
		plan.ConfirmationRequired = len(additional) > 0
		for _, id := range additional {
			plan.Reasons = append(plan.Reasons, "dependent-must-be-disabled:"+id)
		}
		return finishPlan(plan)
	}
	return refuse("unknown-action")
}

// finishPlan sorts the output and computes the digest, so the same input
// yields byte-identical JSON.
func finishPlan(plan Plan) (Plan, error) {
	plan.Candidates = sortedUnique(plan.Candidates)
	plan.Additional = sortedUnique(plan.Additional)
	plan.Unchanged = sortedUnique(plan.Unchanged)
	plan.OptionalFeaturesLost = sortedUnique(plan.OptionalFeaturesLost)
	plan.Unresolved = sortedUnique(plan.Unresolved)
	plan.Cycles = sortedUnique(plan.Cycles)
	plan.Reasons = sortedUnique(plan.Reasons)
	plan.RestartRequired = sortedUnique(plan.RestartRequired)
	plan.SelectionChanged = false
	plan.Digest = planDigest(plan)
	return plan, nil
}

func planDigest(plan Plan) string {
	plan.Digest = ""
	raw, err := json.Marshal(plan)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func otherIDs(plugins []Plugin, selected []string) []string {
	inSelected := map[string]bool{}
	for _, id := range selected {
		inSelected[id] = true
	}
	var out []string
	for _, p := range plugins {
		if !inSelected[p.ID] {
			out = append(out, p.ID)
		}
	}
	sort.Strings(out)
	return out
}

// staleReasons compares the plan's snapshot with the current state. A changed
// source fingerprint is detected even when the Git HEAD and the settings
// version are unchanged.
func staleReasons(want, got Snapshot) []string {
	var reasons []string
	if want.SettingsVersion != got.SettingsVersion {
		reasons = append(reasons, "stale-settings:"+want.SettingsVersion+"->"+got.SettingsVersion)
	}
	if want.GitHead != got.GitHead {
		reasons = append(reasons, "stale-git-head:"+want.GitHead+"->"+got.GitHead)
	}
	if want.DeclarationFingerprint != got.DeclarationFingerprint {
		reasons = append(reasons, "stale-declaration")
	}
	paths := map[string]bool{}
	for path := range want.SourceFingerprints {
		paths[path] = true
	}
	for path := range got.SourceFingerprints {
		paths[path] = true
	}
	sortedPaths := make([]string, 0, len(paths))
	for path := range paths {
		sortedPaths = append(sortedPaths, path)
	}
	sort.Strings(sortedPaths)
	for _, path := range sortedPaths {
		if want.SourceFingerprints[path] != got.SourceFingerprints[path] {
			reasons = append(reasons, "stale-source:"+path)
		}
	}
	if want.Permission == "allowed" && got.Permission != "allowed" {
		reasons = append(reasons, "permission-revoked:"+got.Permission)
	}
	return sortedUnique(reasons)
}

func sameSources(before, after []Source) bool {
	if len(before) != len(after) {
		return false
	}
	for i := range before {
		if before[i] != after[i] {
			return false
		}
	}
	return true
}

// Apply rechecks the snapshot, computes the new settings, and reports the
// settings update and the extension start separately. It never edits a source
// and never installs or starts a plugin by itself.
func Apply(state State, plan Plan, req ApplyRequest, starter func([]string) error) (ApplyResult, error) {
	result := ApplyResult{
		Schema: Schema, PlanDigest: plan.Digest,
		SettingsAtomic: true, ExternalEffectAtomic: false,
		AutoInstalled: false, AutoStarted: false,
		SourcePreserved: true, StandardEditorAvailable: state.StandardEditorAvailable,
	}
	if err := validateState(state); err != nil {
		return result, err
	}
	if plan.Schema != Schema || plan.Digest == "" || planDigest(plan) != plan.Digest {
		result.Reasons = append(result.Reasons, "plan-tampered")
		return finishApply(result), nil
	}
	if stale := staleReasons(plan.Snapshot, snapshotOf(state)); len(stale) > 0 {
		result.Reasons = append(result.Reasons, stale...)
		return finishApply(result), nil
	}
	if plan.Refused {
		result.Reasons = append(result.Reasons, "plan-refused")
		return finishApply(result), nil
	}
	if plan.ConfirmationRequired && !req.Confirm {
		result.Reasons = append(result.Reasons, "confirmation-required")
		return finishApply(result), nil
	}

	enabledAfter := map[string]bool{}
	for _, p := range state.Plugins {
		enabledAfter[p.ID] = p.Enabled
	}
	var removed []string
	for _, id := range plan.Candidates {
		switch plan.Action {
		case ActionEnable:
			enabledAfter[id] = true
		default:
			enabledAfter[id] = false
			if plan.Action == ActionRemove {
				removed = append(removed, id)
			}
		}
	}
	sourcesAfter := append([]Source(nil), state.Sources...)
	result.SettingsApplied = true
	result.NewSettingsVersion = state.SettingsVersion + "+" + shortDigest(plan.Digest)
	result.EnabledAfter = enabledAfter
	result.RemovedAfter = sortedUnique(removed)
	result.SourcesAfter = sourcesAfter
	result.SourcePreserved = sameSources(state.Sources, sourcesAfter)
	result.StandardEditorAvailable = state.StandardEditorAvailable
	result.PluginSpecificFeature = plan.Action == ActionEnable

	if starter == nil {
		result.Reasons = append(result.Reasons, "extension-start-not-requested")
	} else {
		if err := starter(plan.RestartRequired); err != nil {
			result.ExtensionErrors = append(result.ExtensionErrors, err.Error())
			result.Reasons = append(result.Reasons, "extension-start-failed")
		} else {
			result.ExtensionsStarted = true
		}
	}
	result.AllSucceeded = result.SettingsApplied && result.ExtensionsStarted
	return finishApply(result), nil
}

func shortDigest(digest string) string {
	digest = strings.TrimPrefix(digest, "sha256:")
	if len(digest) > 12 {
		return digest[:12]
	}
	return digest
}

func finishApply(result ApplyResult) ApplyResult {
	result.Reasons = sortedUnique(result.Reasons)
	result.ExtensionErrors = sortedUnique(result.ExtensionErrors)
	result.RemovedAfter = sortedUnique(result.RemovedAfter)
	return result
}
