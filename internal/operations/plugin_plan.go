package operations

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

// PluginPlanSchemaVersion identifies a read-only plugin management plan (#4377).
// It is a plan, not an application: nothing here changes a Source, a setting, a
// data byte or an Artifact.
const PluginPlanSchemaVersion = "tei.plugin-management.plan/v0"

// The closed action set. There is deliberately no "apply": applying a plan is a
// separate, qualified operation owned by #2550/#2553, and a plan that could
// apply itself would let a read collapse into a write.
const (
	PlanActionDisable     = "disable"
	PlanActionRemove      = "remove"
	PlanActionDeleteData  = "delete-data"
	PlanActionConvertData = "convert-data"
	PlanActionUpdate      = "update"
)

// PlanActions returns the closed set in report order. `tei commands` advertises
// the same set, so a caller learns it before running anything.
func PlanActions() []string {
	return []string{
		PlanActionDisable, PlanActionRemove, PlanActionDeleteData,
		PlanActionConvertData, PlanActionUpdate,
	}
}

// IsPlanAction reports whether action is one the planner accepts.
func IsPlanAction(action string) bool {
	for _, a := range PlanActions() {
		if a == action {
			return true
		}
	}
	return false
}

// Plan reason codes. They are the management vocabulary (#4376) referenced by
// value, not re-declared as a second table: internal/operations cannot import
// internal/plugin/management because that package transitively imports net,
// which the operation boundary forbids (dependency_test.go). A cmd/tei test
// pins the overlapping codes byte-equal to management so the two vocabularies
// cannot drift. The codes that are genuinely new to planning (contract
// mismatch, a dependent that must be stopped, an irreversible data change and
// a tampered plan) have no resolver counterpart and therefore no
// PLUGIN_RESOLVE_* mapping.
const (
	PlanReasonPluginNotDeclared         = "plugin-not-declared"
	PlanReasonEnvNotSelected            = "env-not-selected"
	PlanReasonVersionUnsupported        = "version-unsupported"
	PlanReasonDependencyCycle           = "dependency-cycle"
	PlanReasonDependencyUnresolved      = "dependency-unresolved"
	PlanReasonContractIncompatible      = "contract-incompatible"
	PlanReasonContractMismatch          = "contract-mismatch"
	PlanReasonLockNotProvided           = "lock-not-provided"
	PlanReasonSourceFingerprintMismatch = "source-fingerprint-mismatch"
	PlanReasonDependentMustBeStopped    = "dependent-must-be-stopped"
	PlanReasonIrreversibleOperation     = "irreversible-operation"
	PlanReasonPlanTampered              = "plan-tampered"
)

// planResolverCodeByReason is the single table mapping a plan reason code to
// the PLUGIN_RESOLVE_* resolver code (one of the nine in
// internal/plugin/diagnostics.go) it corresponds to. It mirrors the management
// mapping and lives here, once, so the two vocabularies cannot both claim to be
// the reason. A plan reason with no resolver counterpart (a data change, a
// contract mismatch found by comparing a lock to its declaration, a tampered
// plan) maps to the empty string and carries no resolver code.
var planResolverCodeByReason = map[string]string{
	PlanReasonVersionUnsupported:   plugin.DiagnosticNoCompatiblePlugin,
	PlanReasonDependencyCycle:      plugin.DiagnosticDependencyCycle,
	PlanReasonDependencyUnresolved: plugin.DiagnosticInvalidRequirement,
	PlanReasonContractIncompatible: plugin.DiagnosticIncompatibleTransitive,
}

// planReason builds one reason, filling the resolver code from the single
// mapping table above so a plan code and its PLUGIN_RESOLVE_* counterpart are
// reported together and never drift apart.
func planReason(code, subject, message string) PlanReason {
	return PlanReason{
		Code:         code,
		ResolverCode: planResolverCodeByReason[code],
		Subject:      subject,
		Message:      message,
	}
}

// PlanReason is one reason a plan carries. The plan code and the resolver code
// ride beside each other; neither is rewritten into the other.
type PlanReason struct {
	Code         string `json:"code"`
	ResolverCode string `json:"resolverCode,omitempty"`
	Subject      string `json:"subject,omitempty"`
	Message      string `json:"message,omitempty"`
}

// PlanConflict is one composition conflict. A duplicate declaration (the same
// plugin id and version declared twice) is a different fact from two providers
// of one contract, and both are named here rather than folded into a refusal.
type PlanConflict struct {
	Kind         string   `json:"kind"`
	Subject      string   `json:"subject"`
	IDs          []string `json:"ids,omitempty"`
	Code         string   `json:"code"`
	ResolverCode string   `json:"resolverCode,omitempty"`
	Message      string   `json:"message,omitempty"`
}

// Conflict kinds.
const (
	ConflictDuplicateDeclaration = "duplicate-declaration"
	ConflictMultipleProviders    = "multiple-providers"
)

// PlanRequested is what the caller selected. The plan never invents a wider
// target than this: the dependents are derived from declarations, not added to
// the request.
type PlanRequested struct {
	Action      string `json:"action"`
	Plugin      string `json:"plugin"`
	Environment string `json:"environment"`
}

// PlanSnapshot is the fact set the plan was made from. An applier rechecks it
// before applying and changes nothing when one field has moved (#2550).
type PlanSnapshot struct {
	// SettingsVersion is nil: TEI has no settings store yet (#4378), and a
	// fabricated value would make every recheck look current.
	SettingsVersion        *string `json:"settingsVersion"`
	DeclarationFingerprint string  `json:"declarationFingerprint"`
	SourceFingerprint      string  `json:"sourceFingerprint,omitempty"`
	PluginLockFingerprint  string  `json:"pluginLockFingerprint,omitempty"`
	// Permission is "unknown": a plan reads declarations, not a live grant.
	Permission string `json:"permission"`
}

// PlanUpdateDiff is the before/after separation an update plan returns. Each
// field is a separate fact so a caller can tell a contract change from a mere
// version bump.
type PlanUpdateDiff struct {
	Plugin            string `json:"plugin"`
	CurrentVersion    string `json:"currentVersion,omitempty"`
	TargetVersion     string `json:"targetVersion,omitempty"`
	VersionChanged    bool   `json:"versionChanged"`
	ContractChanged   bool   `json:"contractChanged"`
	ProviderChanged   bool   `json:"providerChanged"`
	ArtifactChanged   bool   `json:"artifactChanged"`
	BuildPlanChanged  bool   `json:"buildPlanChanged"`
	DependencyChanged bool   `json:"dependencyChanged"`
}

// Plan is the whole read-only proposal. Nothing is changed until a separate,
// qualified applier accepts it.
type Plan struct {
	SchemaVersion string `json:"schemaVersion"`
	Action        string `json:"action"`
	Environment   string `json:"environment"`

	Requested PlanRequested `json:"requested"`

	Candidates                    []string       `json:"candidates"`
	Additional                    []string       `json:"additional"`
	Unchanged                     []string       `json:"unchanged"`
	OptionalFeaturesLost          []string       `json:"optionalFeaturesLost"`
	OptionalDependencyNotModelled bool           `json:"optionalDependencyNotModelled"`
	Unresolved                    []string       `json:"unresolved"`
	Cycles                        []string       `json:"cycles"`
	Conflicts                     []PlanConflict `json:"conflicts"`
	Reasons                       []PlanReason   `json:"reasons"`

	ConfirmationRequired bool     `json:"confirmationRequired"`
	RestartRequired      []string `json:"restartRequired"`
	RegenerationRequired []string `json:"regenerationRequired"`

	SourcesPreserved        bool `json:"sourcesPreserved"`
	DataPreserved           bool `json:"dataPreserved"`
	SelectionChanged        bool `json:"selectionChanged"`
	StandardEditorAvailable bool `json:"standardEditorAvailable"`

	Snapshot PlanSnapshot `json:"snapshot"`

	// Diff is present only for an update plan.
	Diff *PlanUpdateDiff `json:"diff,omitempty"`

	Digest          string   `json:"digest"`
	AuthorityLimits []string `json:"authorityLimits"`
	NotModelled     []string `json:"notModelled"`
}

// PluginPlanInput is the read-only observation the plan derives from. It names
// the part of the World the planner reads, so the input is explicit and the
// planner never reaches for the whole IR.
type PluginPlanInput struct {
	Plugins      []plugin.Definition
	Bundles      []plugin.Bundle
	Environments []environment.Environment
	Environment  core.CanonicalID
	Action       string
	Plugin       core.CanonicalID
	// Lock is the current resolved plugin lock, when the caller supplied one.
	Lock *plugin.Lock
	// TargetLock is the target resolved plugin lock; required by update.
	TargetLock *plugin.Lock
	// SourceFingerprint is the Workspace/Source fingerprint the plan was read
	// at, when the caller has one. It rides in the snapshot so an applier can
	// detect a Source that moved after the plan.
	SourceFingerprint string
}

// UnknownEnvironmentError reports an environment id the World does not declare.
type UnknownEnvironmentError struct {
	ID core.CanonicalID
}

func (e *UnknownEnvironmentError) Error() string {
	return fmt.Sprintf("unknown environment %s", e.ID)
}

// InvalidPlanActionError reports an action outside the closed set. It is a typed
// input error so the CLI classifies it as INVALID_ARGUMENT without matching
// message text.
type InvalidPlanActionError struct {
	Action string
}

func (e *InvalidPlanActionError) Error() string {
	return fmt.Sprintf("action %q is not one of %s", e.Action, strings.Join(PlanActions(), ", "))
}

// declaredGraph is the reverse required-dependency view of the declared
// plugins. It is a dependency analysis over declarations, not a resolver: it
// selects no version and builds no runtime set. The resolver is reused
// separately for the active set.
type declaredGraph struct {
	dependents    map[string][]string // required dep -> plugins that require it
	requires      map[string][]plugin.Requirement
	declared      map[string]bool
	duplicates    []string // "id@version" declared more than once
	hasCompatible func(id string, major int) bool
}

func buildGraph(plugins []plugin.Definition) declaredGraph {
	g := declaredGraph{
		dependents:    map[string][]string{},
		requires:      map[string][]plugin.Requirement{},
		declared:      map[string]bool{},
		hasCompatible: func(string, int) bool { return false },
	}
	seen := map[string]int{}
	for _, p := range plugins {
		id := string(p.ID)
		g.declared[id] = true
		seen[id+"@"+p.Version]++
		if _, ok := g.requires[id]; !ok {
			g.requires[id] = append([]plugin.Requirement(nil), p.Requires...)
		}
		for _, req := range p.Requires {
			g.dependents[string(req.Plugin)] = appendUnique(g.dependents[string(req.Plugin)], id)
		}
	}
	var dups []string
	for key, n := range seen {
		if n > 1 {
			dups = append(dups, key)
		}
	}
	sort.Strings(dups)
	g.duplicates = dups
	g.hasCompatible = func(id string, major int) bool {
		for _, p := range plugins {
			if string(p.ID) == id && plugin.CompatibleMajor(p.Version, major) {
				return true
			}
		}
		return false
	}
	return g
}

func appendUnique(values []string, v string) []string {
	for _, existing := range values {
		if existing == v {
			return values
		}
	}
	return append(values, v)
}

// closure returns the required-dependency closure of seed under reverse edges:
// the seed plus every plugin that (transitively) requires it.
func (g declaredGraph) closure(seed string) map[string]bool {
	out := map[string]bool{seed: true}
	queue := []string{seed}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, dependent := range g.dependents[cur] {
			if !out[dependent] {
				out[dependent] = true
				queue = append(queue, dependent)
			}
		}
	}
	return out
}

// findCycle returns one cycle in the required-dependency graph restricted to
// subset, as "a->b->a". A cycle is a different fact from a missing dependency.
func (g declaredGraph) findCycle(subset map[string]bool) []string {
	const (
		white = 0
		gray  = 1
		black = 2
	)
	state := map[string]int{}
	var stack, cycle []string
	var dfs func(id string) bool
	dfs = func(id string) bool {
		state[id] = gray
		stack = append(stack, id)
		for _, req := range g.requires[id] {
			dep := string(req.Plugin)
			if !subset[dep] {
				continue
			}
			switch state[dep] {
			case white:
				if dfs(dep) {
					return true
				}
			case gray:
				for i, value := range stack {
					if value == dep {
						cycle = append([]string(nil), stack[i:]...)
						cycle = append(cycle, dep)
						return true
					}
				}
			}
		}
		stack = stack[:len(stack)-1]
		state[id] = black
		return false
	}
	ids := make([]string, 0, len(subset))
	for id := range subset {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if state[id] == white && dfs(id) {
			return cycle
		}
	}
	return nil
}

// PluginPlan derives a read-only plan. It never writes, never starts a plugin,
// never grants permission and never mutates its input: every definition is
// worked on through a deep copy, so the caller's World is byte-identical after
// the call.
func PluginPlan(in PluginPlanInput) (Plan, error) {
	if !IsPlanAction(in.Action) {
		return Plan{}, &InvalidPlanActionError{Action: in.Action}
	}
	env, ok := findPlanEnvironment(in.Environments, in.Environment)
	if !ok {
		return Plan{}, &UnknownEnvironmentError{ID: in.Environment}
	}

	// Work on deep copies so normalization inside plugin.DefinitionFingerprint
	// (which sorts capability slices in place) cannot reorder the caller's data.
	plugins := clonePlugins(in.Plugins)

	plan := Plan{
		SchemaVersion:                 PluginPlanSchemaVersion,
		Action:                        in.Action,
		Environment:                   string(env.ID),
		Requested:                     PlanRequested{Action: in.Action, Plugin: string(in.Plugin), Environment: string(env.ID)},
		Candidates:                    []string{},
		Additional:                    []string{},
		Unchanged:                     []string{},
		OptionalFeaturesLost:          []string{},
		OptionalDependencyNotModelled: true,
		Unresolved:                    []string{},
		Cycles:                        []string{},
		Conflicts:                     []PlanConflict{},
		Reasons:                       []PlanReason{},
		RestartRequired:               []string{},
		RegenerationRequired:          []string{},
		// disable/remove/update never touch a data byte; delete-data and
		// convert-data do, and are marked below.
		SourcesPreserved:        true,
		DataPreserved:           in.Action != PlanActionDeleteData && in.Action != PlanActionConvertData,
		SelectionChanged:        false,
		StandardEditorAvailable: true,
		AuthorityLimits:         planAuthorityLimits(),
		NotModelled:             planNotModelled(in.Action),
	}
	plan.Snapshot = PlanSnapshot{
		SettingsVersion:        nil,
		DeclarationFingerprint: declarationFingerprint(plugins),
		SourceFingerprint:      in.SourceFingerprint,
		Permission:             "unknown",
	}

	catalog := plugin.Catalog{Plugins: plugins, Bundles: in.Bundles}
	g := buildGraph(plugins)
	active := activePluginIDs(catalog, env, plugins)
	runtimeKind := runtimeKindByID(plugins)

	// Duplicate declarations are a conflict the resolver also refuses; name it
	// here so the plan distinguishes it from a cycle or a missing dependency.
	for _, dup := range g.duplicates {
		id := dup
		if at := strings.IndexByte(dup, '@'); at >= 0 {
			id = dup[:at]
		}
		plan.Conflicts = append(plan.Conflicts, PlanConflict{
			Kind:         ConflictDuplicateDeclaration,
			Subject:      id,
			IDs:          []string{dup},
			Code:         PlanReasonDependencyUnresolved,
			ResolverCode: plugin.DiagnosticDuplicatePluginVersion,
			Message:      "plugin " + dup + " is declared more than once",
		})
	}
	plan.Conflicts = append(plan.Conflicts, multipleProviderConflicts(catalog, env, plugins)...)

	switch in.Action {
	case PlanActionDisable, PlanActionRemove:
		planStopClosure(&plan, g, active, runtimeKind, in.Plugin)
	case PlanActionDeleteData, PlanActionConvertData:
		planDataChange(&plan, g, active, runtimeKind, in.Plugin, in.Action)
	case PlanActionUpdate:
		planUpdate(&plan, g, active, runtimeKind, plugins, in)
	}

	if in.Lock != nil {
		if fp, err := plugin.EnvironmentLockFingerprint(*in.Lock, env.ID); err == nil {
			plan.Snapshot.PluginLockFingerprint = fp
		}
	}

	finishPlan(&plan)
	return plan, nil
}

// planStopClosure computes the candidate set for disable/remove: the target
// plus every plugin that transitively requires it. Unrelated plugins and pure
// dependencies of the target are left unchanged.
func planStopClosure(plan *Plan, g declaredGraph, active map[string]bool, runtimeKind map[string]string, target core.CanonicalID) {
	tid := string(target)
	refuse := func(reason PlanReason) {
		plan.Reasons = append(plan.Reasons, reason)
	}
	if !g.declared[tid] {
		refuse(planReason(PlanReasonPluginNotDeclared, tid, "plugin "+tid+" is not declared"))
		return
	}
	if !active[tid] {
		refuse(planReason(PlanReasonEnvNotSelected, tid, "plugin "+tid+" is not active in this environment"))
		return
	}
	closure := g.closure(tid)
	if cycle := g.findCycle(closure); cycle != nil {
		plan.Cycles = []string{strings.Join(cycle, "->")}
		refuse(planReason(PlanReasonDependencyCycle, tid, "dependency cycle: "+strings.Join(cycle, "->")))
		return
	}
	// A missing declaration and a version-incompatible declaration are two
	// different facts from a cycle, and from each other: a version mismatch is
	// a version reason, never an unresolved-declaration or a conflict reason.
	var unresolved []string
	versionIssue := false
	for id := range closure {
		for _, req := range g.requires[id] {
			dep := string(req.Plugin)
			if !g.declared[dep] {
				unresolved = append(unresolved, id+":"+dep)
				continue
			}
			if !g.hasCompatible(dep, req.CompatibleMajor) {
				versionIssue = true
				refuse(planReason(PlanReasonVersionUnsupported, dep, "no "+dep+" compatible with major "+itoa(req.CompatibleMajor)))
			}
		}
	}
	if versionIssue {
		return
	}
	if len(unresolved) > 0 {
		plan.Unresolved = sortedUnique(unresolved)
		refuse(planReason(PlanReasonDependencyUnresolved, tid, "unresolved required dependency"))
		return
	}

	candidates := make([]string, 0, len(closure))
	for id := range closure {
		if active[id] {
			candidates = append(candidates, id)
		}
	}
	sort.Strings(candidates)
	var additional []string
	for _, id := range candidates {
		if id != tid {
			additional = append(additional, id)
			refuse(planReason(PlanReasonDependentMustBeStopped, id, "plugin "+id+" requires "+tid+" and must be stopped with it"))
		}
	}
	plan.Candidates = candidates
	plan.Additional = additional
	plan.Unchanged = unchangedFrom(active, candidates)
	plan.RestartRequired = builtinCandidates(candidates, runtimeKind)
	plan.ConfirmationRequired = len(additional) > 0
}

// planDataChange plans an irreversible data operation. TEI has no plugin-owned
// data domain model yet, so the plan names the action and its irreversibility
// without inventing the bytes it would touch.
func planDataChange(plan *Plan, g declaredGraph, active map[string]bool, runtimeKind map[string]string, target core.CanonicalID, action string) {
	tid := string(target)
	if !g.declared[tid] {
		plan.Reasons = append(plan.Reasons, planReason(PlanReasonPluginNotDeclared, tid, "plugin "+tid+" is not declared"))
		return
	}
	if !active[tid] {
		plan.Reasons = append(plan.Reasons, planReason(PlanReasonEnvNotSelected, tid, "plugin "+tid+" is not active in this environment"))
		return
	}
	plan.Candidates = []string{tid}
	plan.Unchanged = unchangedFrom(active, plan.Candidates)
	plan.RestartRequired = builtinCandidates(plan.Candidates, runtimeKind)
	// A byte change is irreversible and needs its own explicit confirmation.
	plan.ConfirmationRequired = true
	plan.Reasons = append(plan.Reasons, planReason(PlanReasonIrreversibleOperation, tid, action+" changes data bytes and cannot be undone by this plan"))
}

// planUpdate compares the current and target locks for one plugin and separates
// the change kinds. It never claims a hot apply.
func planUpdate(plan *Plan, g declaredGraph, active map[string]bool, runtimeKind map[string]string, plugins []plugin.Definition, in PluginPlanInput) {
	tid := string(in.Plugin)
	if !g.declared[tid] {
		plan.Reasons = append(plan.Reasons, planReason(PlanReasonPluginNotDeclared, tid, "plugin "+tid+" is not declared"))
		return
	}
	if in.Lock == nil {
		plan.Reasons = append(plan.Reasons, planReason(PlanReasonLockNotProvided, tid, "update requires -plugin-lock"))
		return
	}
	if in.TargetLock == nil {
		plan.Reasons = append(plan.Reasons, planReason(PlanReasonLockNotProvided, tid, "update requires -target-lock"))
		return
	}
	cur, curOK := lockedPlugin(*in.Lock, in.Environment, in.Plugin)
	tgt, tgtOK := lockedPlugin(*in.TargetLock, in.Environment, in.Plugin)
	if !curOK {
		plan.Reasons = append(plan.Reasons, planReason(PlanReasonSourceFingerprintMismatch, tid, "current lock does not describe environment "+string(in.Environment)))
		return
	}
	if !tgtOK {
		plan.Reasons = append(plan.Reasons, planReason(PlanReasonSourceFingerprintMismatch, tid, "target lock does not describe environment "+string(in.Environment)))
		return
	}

	diff := PlanUpdateDiff{
		Plugin:         tid,
		CurrentVersion: cur.Version,
		TargetVersion:  tgt.Version,
	}
	diff.VersionChanged = cur.Version != tgt.Version
	diff.ContractChanged = cur.Fingerprint != tgt.Fingerprint
	diff.ArtifactChanged = cur.ArtifactDigest != tgt.ArtifactDigest
	diff.BuildPlanChanged = cur.BuildPlanFingerprint != tgt.BuildPlanFingerprint

	curDef, curDefOK := exactPlugin(plugins, in.Plugin, cur.Version)
	tgtDef, tgtDefOK := exactPlugin(plugins, in.Plugin, tgt.Version)
	if curDefOK && tgtDefOK {
		diff.ProviderChanged = !sameProviders(curDef, tgtDef)
		diff.DependencyChanged = !sameRequires(curDef, tgtDef)
		// A lock whose contract fingerprint no longer matches the declaration
		// it pins is stale: the update cannot be shown as compatible.
		if plugin.DefinitionFingerprint(curDef) != cur.Fingerprint {
			plan.Reasons = append(plan.Reasons, planReason(PlanReasonContractMismatch, tid, "current lock fingerprint diverges from the declared "+tid+"@"+cur.Version))
		}
		if plugin.DefinitionFingerprint(tgtDef) != tgt.Fingerprint {
			plan.Reasons = append(plan.Reasons, planReason(PlanReasonContractMismatch, tid, "target lock fingerprint diverges from the declared "+tid+"@"+tgt.Version))
		}
	}
	plan.Diff = &diff

	changed := diff.VersionChanged || diff.ContractChanged || diff.ProviderChanged ||
		diff.ArtifactChanged || diff.BuildPlanChanged || diff.DependencyChanged
	if changed {
		plan.Candidates = []string{tid}
		// A static Go single binary loads a plugin in-process: a changed plugin
		// is not live until the host restarts. Never claim a hot apply.
		if runtimeKind[tid] == string(plugin.RuntimeBuiltin) {
			plan.RestartRequired = []string{tid}
		}
		// Regeneration is required only when the build inputs moved.
		if diff.BuildPlanChanged || diff.ArtifactChanged {
			plan.RegenerationRequired = []string{tid}
		}
	}
	plan.Unchanged = unchangedFrom(active, plan.Candidates)
}

// VerifyPlan rechecks a plan's digest. It returns plan-tampered when the stored
// digest does not match the plan content, so an applier refuses a plan whose
// bytes moved after it was produced. It is read-only.
func VerifyPlan(plan Plan) []string {
	if plan.Digest == "" {
		return []string{PlanReasonPlanTampered}
	}
	if planDigest(plan) != plan.Digest {
		return []string{PlanReasonPlanTampered}
	}
	return nil
}

// finishPlan sorts every list and computes the digest, so the same input yields
// byte-identical JSON and a tampered plan is detectable.
func finishPlan(plan *Plan) {
	plan.Candidates = sortedUnique(plan.Candidates)
	plan.Additional = sortedUnique(plan.Additional)
	plan.Unchanged = sortedUnique(plan.Unchanged)
	plan.OptionalFeaturesLost = sortedUnique(plan.OptionalFeaturesLost)
	plan.Unresolved = sortedUnique(plan.Unresolved)
	plan.Cycles = sortedUnique(plan.Cycles)
	plan.RestartRequired = sortedUnique(plan.RestartRequired)
	plan.RegenerationRequired = sortedUnique(plan.RegenerationRequired)
	plan.AuthorityLimits = sortedUnique(plan.AuthorityLimits)
	plan.NotModelled = sortedUnique(plan.NotModelled)
	sort.Slice(plan.Conflicts, func(i, j int) bool {
		if plan.Conflicts[i].Kind != plan.Conflicts[j].Kind {
			return plan.Conflicts[i].Kind < plan.Conflicts[j].Kind
		}
		return plan.Conflicts[i].Subject < plan.Conflicts[j].Subject
	})
	sort.Slice(plan.Reasons, func(i, j int) bool {
		a, b := plan.Reasons[i], plan.Reasons[j]
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.ResolverCode != b.ResolverCode {
			return a.ResolverCode < b.ResolverCode
		}
		return a.Subject < b.Subject
	})
	plan.SelectionChanged = false
	// A nil list marshals as null, which a reader can mistake for "unknown".
	// Every list a plan carries is present, so normalize nil to empty before
	// the digest is taken.
	plan.Candidates = nonNilStrings(plan.Candidates)
	plan.Additional = nonNilStrings(plan.Additional)
	plan.Unchanged = nonNilStrings(plan.Unchanged)
	plan.OptionalFeaturesLost = nonNilStrings(plan.OptionalFeaturesLost)
	plan.Unresolved = nonNilStrings(plan.Unresolved)
	plan.Cycles = nonNilStrings(plan.Cycles)
	plan.RestartRequired = nonNilStrings(plan.RestartRequired)
	plan.RegenerationRequired = nonNilStrings(plan.RegenerationRequired)
	plan.AuthorityLimits = nonNilStrings(plan.AuthorityLimits)
	plan.NotModelled = nonNilStrings(plan.NotModelled)
	plan.Digest = planDigest(*plan)
}

// planDigest hashes the plan content with the digest field cleared, so a change
// to any candidate changes the digest and a mismatch names a tampered plan.
func planDigest(plan Plan) string {
	plan.Digest = ""
	raw, err := json.Marshal(plan)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func planAuthorityLimits() []string {
	return []string{
		"plugin-plan is read-only; it applies nothing and grants no permission",
		"a restart is required to load an updated plugin; hot-apply is not claimed",
		"confirmation does not apply the plan; applying is a separate qualified operation",
		"optional dependencies are not modelled; optional feature loss is not captured",
	}
}

func planNotModelled(action string) []string {
	base := []string{
		"install", "enable", "settings-version", "invocation", "effect", "optional-dependency",
	}
	if action == PlanActionDeleteData || action == PlanActionConvertData {
		base = append(base, "retained-data")
	}
	return base
}

// activePluginIDs returns the plugin ids resolved for the environment. It reuses
// the existing resolver exactly once; when the resolver refuses (a cycle or a
// version conflict elsewhere) it falls back to the declared selections so the
// plan can still name the target, and the graph analysis reports the refusal.
func activePluginIDs(catalog plugin.Catalog, env environment.Environment, plugins []plugin.Definition) map[string]bool {
	active := map[string]bool{}
	if resolved, err := catalog.ResolveEnvironment(env); err == nil {
		for _, ident := range resolved.Plugins {
			active[string(ident.ID)] = true
		}
		return active
	}
	selected := map[string]bool{}
	for _, req := range env.Plugins {
		selected[string(req.ID)] = true
	}
	for _, b := range env.Bundles {
		for _, bundle := range catalog.Bundles {
			if bundle.ID == b.ID && plugin.CompatibleMajor(bundle.Version, b.CompatibleMajor) {
				for _, p := range bundle.Plugins {
					selected[string(p.Plugin)] = true
				}
			}
		}
	}
	// Pull the transitive requires of the selections so a dependency is not
	// mistaken for an unrelated plugin.
	queue := make([]string, 0, len(selected))
	for id := range selected {
		queue = append(queue, id)
	}
	requires := map[string][]plugin.Requirement{}
	for _, p := range plugins {
		requires[string(p.ID)] = p.Requires
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if active[cur] {
			continue
		}
		active[cur] = true
		for _, req := range requires[cur] {
			if !active[string(req.Plugin)] {
				queue = append(queue, string(req.Plugin))
			}
		}
	}
	return active
}

func multipleProviderConflicts(catalog plugin.Catalog, env environment.Environment, plugins []plugin.Definition) []PlanConflict {
	resolved, err := catalog.ResolveEnvironment(env)
	if err != nil {
		return nil
	}
	type key struct{ kind, id string }
	providers := map[key][]string{}
	for _, rc := range resolved.Capabilities {
		k := key{string(rc.Capability.Kind), string(rc.Capability.ID)}
		providers[k] = appendUnique(providers[k], string(rc.Plugin.ID))
	}
	var keys []key
	for k := range providers {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].kind != keys[j].kind {
			return keys[i].kind < keys[j].kind
		}
		return keys[i].id < keys[j].id
	})
	var out []PlanConflict
	for _, k := range keys {
		ids := sortedUnique(providers[k])
		if len(ids) > 1 {
			out = append(out, PlanConflict{
				Kind:    ConflictMultipleProviders,
				Subject: k.kind + " " + k.id,
				IDs:     ids,
				Code:    PlanReasonContractIncompatible,
				Message: "capability " + k.kind + " " + k.id + " is provided by more than one plugin",
			})
		}
	}
	return out
}

func lockedPlugin(lock plugin.Lock, envID core.CanonicalID, id core.CanonicalID) (plugin.LockedPlugin, bool) {
	for _, el := range lock.Environments {
		if el.Environment != envID {
			continue
		}
		for _, lp := range el.Plugins {
			if lp.ID == id {
				return lp, true
			}
		}
	}
	return plugin.LockedPlugin{}, false
}

func exactPlugin(plugins []plugin.Definition, id core.CanonicalID, version string) (plugin.Definition, bool) {
	for _, p := range plugins {
		if p.ID == id && p.Version == version {
			return p, true
		}
	}
	return plugin.Definition{}, false
}

func sameProviders(a, b plugin.Definition) bool {
	return capabilityKey(a) == capabilityKey(b)
}

func sameRequires(a, b plugin.Definition) bool {
	ar := requirementKey(a)
	br := requirementKey(b)
	if len(ar) != len(br) {
		return false
	}
	for i := range ar {
		if ar[i] != br[i] {
			return false
		}
	}
	return true
}

func capabilityKey(d plugin.Definition) string {
	var parts []string
	for _, c := range d.Provides {
		parts = append(parts, string(c.Kind)+" "+string(c.ID)+" "+strings.Join(databaseProvidersOf(c), ","))
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

func databaseProvidersOf(c plugin.Capability) []string {
	out := make([]string, 0, len(c.DatabaseProviders))
	for _, p := range c.DatabaseProviders {
		out = append(out, string(p))
	}
	sort.Strings(out)
	return out
}

func requirementKey(d plugin.Definition) []string {
	out := make([]string, 0, len(d.Requires))
	for _, r := range d.Requires {
		out = append(out, string(r.Plugin)+"@"+itoa(r.CompatibleMajor))
	}
	sort.Strings(out)
	return out
}

func builtinCandidates(candidates []string, runtimeKind map[string]string) []string {
	var out []string
	for _, id := range candidates {
		if runtimeKind[id] == string(plugin.RuntimeBuiltin) {
			out = append(out, id)
		}
	}
	return out
}

func runtimeKindByID(plugins []plugin.Definition) map[string]string {
	out := map[string]string{}
	for _, p := range plugins {
		out[string(p.ID)] = string(p.Runtime.Kind)
	}
	return out
}

func unchangedFrom(active map[string]bool, candidates []string) []string {
	inCandidates := map[string]bool{}
	for _, id := range candidates {
		inCandidates[id] = true
	}
	var out []string
	for id := range active {
		if !inCandidates[id] {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}

func findPlanEnvironment(envs []environment.Environment, id core.CanonicalID) (environment.Environment, bool) {
	for _, e := range envs {
		if e.ID == id {
			return e, true
		}
	}
	return environment.Environment{}, false
}

func declarationFingerprint(plugins []plugin.Definition) string {
	fps := make([]string, 0, len(plugins))
	for _, p := range plugins {
		fps = append(fps, string(p.ID)+"@"+p.Version+"="+plugin.DefinitionFingerprint(p))
	}
	sort.Strings(fps)
	sum := sha256.Sum256([]byte(strings.Join(fps, "\n")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// clonePlugins deep-copies the definitions through JSON so the caller's World
// is never reordered by normalization. A marshal failure falls back to a shallow
// copy, which is still safe for the read-only paths that do not fingerprint.
func clonePlugins(src []plugin.Definition) []plugin.Definition {
	if src == nil {
		return nil
	}
	raw, err := json.Marshal(src)
	if err != nil {
		return append([]plugin.Definition(nil), src...)
	}
	var out []plugin.Definition
	if err := json.Unmarshal(raw, &out); err != nil {
		return append([]plugin.Definition(nil), src...)
	}
	return out
}

func sortedUnique(values []string) []string {
	if len(values) == 0 {
		return values
	}
	sorted := append([]string(nil), values...)
	sort.Strings(sorted)
	out := sorted[:0]
	for _, v := range sorted {
		if v == "" {
			continue
		}
		if len(out) == 0 || out[len(out)-1] != v {
			out = append(out, v)
		}
	}
	return out
}

// nonNilStrings returns an empty slice instead of nil, so a list marshals as []
// rather than null and a reader never mistakes "no entries" for "unknown".
func nonNilStrings(values []string) []string {
	if values == nil {
		return []string{}
	}
	return values
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
