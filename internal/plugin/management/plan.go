package management

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
)

// PlanSchemaVersion identifies the read-only management plan (#4377).
const PlanSchemaVersion = "tei.plugin-management.plan/v0"

// Plan actions. They are separate operations, never one flag with a mode:
// disabling a plugin, removing it, deleting its data and converting its data
// change different things and have different irreversibility.
const (
	ActionDisable     = "disable-plugin"
	ActionRemove      = "remove-plugin"
	ActionDeleteData  = "delete-data"
	ActionConvertData = "convert-data"
	ActionUpdate      = "update-plugin"
)

// PlanActions lists every accepted action in report order.
var PlanActions = []string{ActionDisable, ActionRemove, ActionDeleteData, ActionConvertData, ActionUpdate}

// PlanRequest is one user operation. It never names other plugins: the
// dependents are derived from the declarations, not from a row selection.
type PlanRequest struct {
	Action   string `json:"action"`
	PluginID string `json:"pluginId"`
}

// PlanSnapshot is the fact set a plan was made from. The applying operation
// compares it again and changes nothing when it differs.
type PlanSnapshot struct {
	SettingsVersion        *string `json:"settingsVersion"`
	DeclarationFingerprint string  `json:"declarationFingerprint"`
	EnvironmentFingerprint string  `json:"environmentFingerprint"`
	LockFingerprint        string  `json:"lockFingerprint,omitempty"`
	ScanState              string  `json:"scanState"`
}

// UpdateChange is one difference between the current and the target lock.
type UpdateChange struct {
	Kind     string `json:"kind"`
	PluginID string `json:"pluginId"`
	From     string `json:"from,omitempty"`
	To       string `json:"to,omitempty"`
}

// Plan is the read-only proposal. Nothing changes until an applying operation
// rechecks the snapshot and the explicit confirmation.
type Plan struct {
	SchemaVersion string   `json:"schemaVersion"`
	Action        string   `json:"action"`
	Requested     string   `json:"requested"`
	Environment   string   `json:"environment"`
	Candidates    []string `json:"candidates"`
	Additional    []string `json:"additional,omitempty"`
	Unchanged     []string `json:"unchanged"`
	Conflicts     []string `json:"conflicts,omitempty"`
	Cycles        []string `json:"cycles,omitempty"`
	Unresolved    []string `json:"unresolved,omitempty"`
	Reasons       []string `json:"reasons,omitempty"`
	Refused       bool     `json:"refused"`

	OptionalDependencyNotModelled bool     `json:"optionalDependencyNotModelled"`
	OptionalFeaturesLost          []string `json:"optionalFeaturesLost,omitempty"`

	ConfirmationRequired bool `json:"confirmationRequired"`
	// RestartRequired names the targets that need a restart. A static Go
	// single binary cannot apply a settings change in place, so this is not
	// a hot-apply claim.
	RestartRequired      []string `json:"restartRequired,omitempty"`
	RegenerationRequired []string `json:"regenerationRequired,omitempty"`

	SourcesPreserved        bool `json:"sourcesPreserved"`
	DataPreserved           bool `json:"dataPreserved"`
	StandardEditorAvailable bool `json:"standardEditorAvailable"`
	SelectionChanged        bool `json:"selectionChanged"`

	// Snapshot is what the plan was made from; an applying operation must see
	// the same values or refuse.
	Snapshot PlanSnapshot `json:"snapshot"`
	// Digest identifies the plan content. An applying operation refuses a plan
	// whose digest does not match what it was given.
	Digest string `json:"digest"`

	UpdateChanges []UpdateChange `json:"updateChanges,omitempty"`

	UnattendedUnsafe []string `json:"unattendedUnsafe"`
	AuthorityLimits  []string `json:"authorityLimits"`
	NotModelled      []string `json:"notModelled,omitempty"`
}

// PlanInput is the read-only observation a plan derives from. It reuses the
// same World, environment and lock the inventory read, so a plan cannot see a
// different resolution than the list it explains.
type PlanInput struct {
	World       World
	Environment core.CanonicalID
	Lock        *plugin.Lock
	// TargetLock is the lock after an update, for ActionUpdate.
	TargetLock *plugin.Lock
	ScanState  string
	Request    PlanRequest
}

// UnsupportedPlanError reports an action the plan does not implement.
type UnsupportedPlanError struct {
	Action string
}

func (e *UnsupportedPlanError) Error() string {
	return fmt.Sprintf("unsupported plan action %q; valid actions are %s", e.Action, strings.Join(PlanActions, ", "))
}

// RefusalError reports a plan that cannot be made for this request, for example
// an unknown plugin or an environment the World does not declare.
type RefusalError struct {
	Code    string
	Message string
}

func (e *RefusalError) Error() string { return e.Message }

// DerivePlan builds the read-only plan. It reads; it never writes, starts a
// plugin, removes one or deletes anything. A plan that would change more than
// the requested plugin is presented with confirmationRequired=true and is not
// applied by this package at all.
func DerivePlan(in PlanInput) (Plan, error) {
	switch in.Request.Action {
	case ActionDisable, ActionRemove, ActionDeleteData, ActionConvertData, ActionUpdate:
	default:
		return Plan{}, &UnsupportedPlanError{Action: in.Request.Action}
	}

	state := in.ScanState
	if state == "" {
		state = ScanComplete
	}
	env, ok := findEnvironment(in.World.Environments, in.Environment)
	if !ok {
		return Plan{}, &UnknownEnvironmentError{ID: in.Environment}
	}
	if in.Request.PluginID == "" {
		return Plan{}, &RefusalError{Code: "invalid-argument", Message: "plan requires a plugin id"}
	}

	report, err := Derive(Input{
		World:       in.World,
		Environment: in.Environment,
		Lock:        in.Lock,
		ScanState:   state,
	})
	if err != nil {
		return Plan{}, err
	}

	plan := Plan{
		SchemaVersion:                 PlanSchemaVersion,
		Action:                        in.Request.Action,
		Requested:                     in.Request.PluginID,
		Environment:                   string(env.ID),
		OptionalDependencyNotModelled: true,
		SelectionChanged:              false,
		// A management operation on the settings never writes a Source, so
		// sourcesPreserved is a property of the operation, not of its success.
		SourcesPreserved:        true,
		StandardEditorAvailable: true,
		UnattendedUnsafe:        unattendedUnsafe(),
		AuthorityLimits:         planAuthorityLimits(),
		Snapshot: PlanSnapshot{
			SettingsVersion:        report.SettingsVersion,
			DeclarationFingerprint: declarationFingerprint(in.World),
			EnvironmentFingerprint: environmentFingerprint(env),
			LockFingerprint:        report.Resolver.LockFingerprint,
			ScanState:              state,
		},
	}
	if in.Lock != nil {
		plan.Snapshot.LockFingerprint = report.Resolver.LockFingerprint
	}

	selected := map[string]bool{}
	for _, ident := range resolvedIdentities(in, env) {
		selected[string(ident.ID)] = true
	}

	switch in.Request.Action {
	case ActionDisable, ActionRemove:
		deriveClosurePlan(&plan, in, selected)
	case ActionUpdate:
		deriveUpdatePlan(&plan, in, selected)
	case ActionDeleteData, ActionConvertData:
		deriveLifecyclePlan(&plan, in, selected)
	}

	if len(plan.Reasons) > 0 {
		plan.Refused = true
	}
	plan.ConfirmationRequired = len(plan.Additional) > 0 && !plan.Refused
	plan.Digest = planDigest(plan)
	return plan, nil
}

func resolvedIdentities(in PlanInput, env environment.Environment) []plugin.Identity {
	catalog := plugin.Catalog{Plugins: in.World.Plugins, Bundles: in.World.Bundles}
	var resolved plugin.ResolvedSet
	var err error
	if in.Lock != nil {
		resolved, err = catalog.ResolveEnvironmentWithLock(env, *in.Lock)
	} else {
		resolved, err = catalog.ResolveEnvironment(env)
	}
	if err != nil {
		return nil
	}
	return resolved.Plugins
}

// deriveClosurePlan computes the candidate set for disable/remove. The
// requested plugin and every plugin that requires it, transitively, become
// candidates. Unrelated plugins never become candidates: they are listed as
// unchanged, which is the fact a reader needs to trust the plan.
func deriveClosurePlan(plan *Plan, in PlanInput, selected map[string]bool) {
	requested := in.Request.PluginID
	if !selected[requested] {
		plan.Reasons = append(plan.Reasons, "not-selected")
		plan.Unresolved = append(plan.Unresolved, requested)
		plan.Candidates = []string{}
		plan.Unchanged = sortedKeys(selected)
		plan.DataPreserved = in.Request.Action == ActionDisable
		return
	}
	if in.Request.Action == ActionRemove {
		// Removing a plugin that is still required by a selected plugin is
		// refused: the dependent would lose its provider silently.
		dependents := dependentsOf(in.World.Plugins, requested)
		if len(dependents) > 0 {
			plan.Reasons = append(plan.Reasons, "still-required")
			plan.Conflicts = append(plan.Conflicts, dependents...)
			plan.Candidates = []string{}
			plan.Unchanged = sortedKeys(selected)
			plan.DataPreserved = true
			return
		}
	}

	dependents := dependentsOf(in.World.Plugins, requested)
	candidates := append([]string{requested}, dependents...)
	sort.Strings(candidates)

	additional := make([]string, 0, len(dependents))
	for _, id := range dependents {
		additional = append(additional, id)
	}
	plan.Candidates = candidates
	plan.Additional = sortedUnique(additional)

	unchanged := []string{}
	for id := range selected {
		if !containsString(candidates, id) {
			unchanged = append(unchanged, id)
		}
	}
	plan.Unchanged = sortedUnique(unchanged)

	// Data is preserved by both disable and remove: neither operation deletes
	// data, and the plan says so before an applying operation runs.
	plan.DataPreserved = true
	plan.OptionalFeaturesLost = optionalFeaturesLost(in.World.Plugins, candidates)

	// A settings change cannot be applied in place to a static binary. The
	// plan says so instead of claiming a hot apply.
	plan.RestartRequired = []string{"runtime"}
	if in.Request.Action == ActionRemove {
		plan.RegenerationRequired = []string{"build"}
	}
	plan.NotModelled = []string{NotModelledEnable, NotModelledOptionalDependency}
}

func deriveUpdatePlan(plan *Plan, in PlanInput, selected map[string]bool) {
	requested := in.Request.PluginID
	if in.TargetLock == nil {
		plan.Reasons = append(plan.Reasons, "target-lock-required")
		plan.Candidates = []string{}
		plan.Unchanged = sortedKeys(selected)
		plan.DataPreserved = true
		return
	}
	plan.UpdateChanges = diffLocks(in.Lock, in.TargetLock)
	if len(plan.UpdateChanges) == 0 {
		plan.Reasons = append(plan.Reasons, "no-change")
		plan.Candidates = []string{}
		plan.Unchanged = sortedKeys(selected)
		plan.DataPreserved = true
		return
	}
	if !selected[requested] {
		plan.Reasons = append(plan.Reasons, "not-selected")
		plan.Candidates = []string{}
		plan.Unchanged = sortedKeys(selected)
		plan.DataPreserved = true
		return
	}
	dependents := dependentsOf(in.World.Plugins, requested)
	plan.Candidates = append([]string{requested}, dependents...)
	sort.Strings(plan.Candidates)
	plan.Additional = sortedUnique(dependents)
	unchanged := []string{}
	for id := range selected {
		if !containsString(plan.Candidates, id) {
			unchanged = append(unchanged, id)
		}
	}
	plan.Unchanged = sortedUnique(unchanged)
	plan.DataPreserved = true
	plan.RestartRequired = []string{"runtime"}
	plan.RegenerationRequired = []string{"build"}
	plan.NotModelled = []string{NotModelledEnable}
}

// deriveLifecyclePlan plans data deletion or conversion. It never returns a
// plan that may run unattended: both operations change bytes and need an
// explicit confirmation, a permission check and a recovery path (#4380).
func deriveLifecyclePlan(plan *Plan, in PlanInput, selected map[string]bool) {
	_ = selected
	plan.Candidates = []string{in.Request.PluginID}
	plan.ConfirmationRequired = false
	plan.DataPreserved = false
	plan.Reasons = append(plan.Reasons,
		"requires-owner-approval",
		"recovery-path-not-established",
	)
	plan.UnattendedUnsafe = append(plan.UnattendedUnsafe, in.Request.Action)
	plan.NotModelled = []string{NotModelledRetainedData, NotModelledSettingsVersion}
}

// dependentsOf lists the plugins that require id, directly or transitively.
// It follows only the declared required dependencies: TEI has no optional
// dependency declaration, so nothing can be lost "optionally".
func dependentsOf(definitions []plugin.Definition, id string) []string {
	direct := map[string][]string{}
	for _, d := range definitions {
		for _, req := range d.Requires {
			direct[string(req.Plugin)] = append(direct[string(req.Plugin)], string(d.ID))
		}
	}
	seen := map[string]bool{}
	var out []string
	var walk func(string)
	walk = func(target string) {
		for _, dependent := range direct[target] {
			if seen[dependent] || dependent == id {
				continue
			}
			seen[dependent] = true
			out = append(out, dependent)
			walk(dependent)
		}
	}
	walk(id)
	sort.Strings(out)
	return out
}

// optionalFeaturesLost names the capability ids a candidate provides that no
// other selected plugin provides. TEI has no optional dependency, so a "lost
// feature" is a capability the environment still selects through another
// declaration; an empty list is the honest answer here.
func optionalFeaturesLost(definitions []plugin.Definition, candidates []string) []string {
	return []string{}
}

func diffLocks(current, target *plugin.Lock) []UpdateChange {
	var changes []UpdateChange
	if current == nil || target == nil {
		return changes
	}
	type key struct {
		environment string
		plugin      string
	}
	pinned := func(lock *plugin.Lock) map[key]plugin.LockedPlugin {
		out := map[key]plugin.LockedPlugin{}
		for _, el := range lock.Environments {
			for _, lp := range el.Plugins {
				out[key{string(el.Environment), string(lp.ID)}] = lp
			}
		}
		return out
	}
	from, to := pinned(current), pinned(target)
	for k, before := range from {
		after, ok := to[k]
		if !ok {
			changes = append(changes, UpdateChange{Kind: "removed", PluginID: k.plugin})
			continue
		}
		if before.Version != after.Version {
			changes = append(changes, UpdateChange{Kind: "version", PluginID: k.plugin, From: before.Version, To: after.Version})
		}
		if before.Fingerprint != after.Fingerprint {
			changes = append(changes, UpdateChange{Kind: "contract", PluginID: k.plugin})
		}
		if before.ArtifactDigest != after.ArtifactDigest {
			changes = append(changes, UpdateChange{Kind: "artifact", PluginID: k.plugin, From: before.ArtifactDigest, To: after.ArtifactDigest})
		}
		if before.BuildPlanFingerprint != after.BuildPlanFingerprint {
			changes = append(changes, UpdateChange{Kind: "build", PluginID: k.plugin})
		}
	}
	for k := range to {
		if _, ok := from[k]; !ok {
			changes = append(changes, UpdateChange{Kind: "added", PluginID: k.plugin})
		}
	}
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].PluginID != changes[j].PluginID {
			return changes[i].PluginID < changes[j].PluginID
		}
		return changes[i].Kind < changes[j].Kind
	})
	return changes
}

func declarationFingerprint(w World) string {
	type entry struct {
		ID      string `json:"id"`
		Version string `json:"version"`
		Hash    string `json:"hash"`
	}
	entries := make([]entry, 0, len(w.Plugins))
	for _, d := range w.Plugins {
		entries = append(entries, entry{ID: string(d.ID), Version: d.Version, Hash: plugin.DefinitionFingerprint(d)})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].ID != entries[j].ID {
			return entries[i].ID < entries[j].ID
		}
		return entries[i].Version < entries[j].Version
	})
	return hashJSON(entries)
}

func environmentFingerprint(env environment.Environment) string {
	return hashJSON(env)
}

func hashJSON(value any) string {
	raw, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// planDigest identifies the plan content. An applying operation refuses a plan
// whose digest does not match: a plan edited between planning and applying is
// not the plan the user confirmed.
func planDigest(plan Plan) string {
	copy := plan
	copy.Digest = ""
	return hashJSON(copy)
}

func planAuthorityLimits() []string {
	return []string{
		"a plan grants no permission and changes nothing",
		"the candidate set is derived from declared required dependencies only",
		"optional dependencies are not modelled; no feature is treated as optional",
		"restartRequired means the static target cannot apply the change in place",
		"data deletion and conversion are listed as unsafe to run unattended until #4380 settles recovery",
	}
}

func unattendedUnsafe() []string {
	return []string{ActionDeleteData, ActionConvertData}
}

func sortedKeys(values map[string]bool) []string {
	out := make([]string, 0, len(values))
	for value := range values {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
