// Package management derives the read-only plugin management inventory (#4376).
//
// It answers "what does TEI know about this plugin, and what does it not know"
// from what the Workspace, the resolver and the Plugin Lock already say. It is
// deliberately not a second resolver, registry or authority: every selection
// and every refusal reason comes from plugin.Catalog and
// plugin.ResolveEnvironment, and every pin comes from plugin.Lock.
//
// The package never reads a file, never writes a file, never starts a plugin
// and never grants permission. Installation, the enabled setting, initialisation
// failure and optional dependencies are NOT modelled in TEI today; a row names
// those gaps instead of reporting them as false.
package management

import (
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/permission"
	"tei/internal/plugin"
	"tei/internal/plugin/runtime"
)

// Evidence is a reference to a fact. It carries no Source text, no data value
// and no permission token: only the kind of fact, a stable reference and a
// fingerprint a reader can recompute.
type Evidence struct {
	Kind        string `json:"kind"`
	Ref         string `json:"ref"`
	Fingerprint string `json:"fingerprint,omitempty"`
}

// Capability is a projection of plugin.Capability for the inventory. It is not
// a second declaration.
type Capability struct {
	Kind              string   `json:"kind"`
	ID                string   `json:"id"`
	Actions           []string `json:"actions,omitempty"`
	Observations      []string `json:"observations,omitempty"`
	Evidence          []string `json:"evidence,omitempty"`
	DatabaseProviders []string `json:"databaseProviders,omitempty"`
	Features          []string `json:"features,omitempty"`
	Permissions       []string `json:"permissions,omitempty"`
}

// Requirement is one declared dependency and the compatible major it demands.
type Requirement struct {
	Plugin          string `json:"plugin"`
	CompatibleMajor int    `json:"compatibleMajor"`
}

// External is one declared external requirement.
type External struct {
	Scope string `json:"scope"`
	Kind  string `json:"kind"`
	Name  string `json:"name"`
}

// Row is one plugin as the inventory sees it. Every field is a separate fact:
// Declared, EnvSelected and Resolved are never folded into one "enabled" value.
type Row struct {
	PluginID string `json:"pluginId"`
	Version  string `json:"version,omitempty"`

	// Declared is true when the Workspace declares this plugin.
	Declared bool `json:"declared"`
	// EnvSelected is true when the environment names this plugin directly, or
	// the resolver selected it. It is a composition choice, not an "enabled"
	// setting.
	EnvSelected bool `json:"envSelected"`
	// Resolved is true when the resolver selected this exact plugin version.
	Resolved bool `json:"resolved"`
	// LockPinned is nil when no lock was supplied. A nil value is "unknown",
	// never false.
	LockPinned           *bool  `json:"lockPinned,omitempty"`
	ArtifactDigest       string `json:"artifactDigest,omitempty"`
	BuildPlanFingerprint string `json:"buildPlanFingerprint,omitempty"`
	RuntimeKind          string `json:"runtimeKind,omitempty"`
	RuntimeBoundary      string `json:"runtimeBoundary,omitempty"`
	RuntimeTrust         string `json:"runtimeTrust,omitempty"`

	Roles        []string      `json:"roles,omitempty"`
	Capabilities []Capability  `json:"capabilities,omitempty"`
	Requires     []Requirement `json:"requires,omitempty"`
	External     []External    `json:"external,omitempty"`

	ReasonCodes []string   `json:"reasonCodes"`
	Remediation []string   `json:"remediation,omitempty"`
	Impact      string     `json:"impact,omitempty"`
	Recheck     []string   `json:"recheck,omitempty"`
	Remaining   []string   `json:"remaining,omitempty"`
	Evidence    []Evidence `json:"evidence,omitempty"`

	// NotModelled names the facts TEI cannot answer for this row, so a reader
	// does not mistake a missing fact for a false one.
	NotModelled []string `json:"notModelled,omitempty"`
}

// ResolverReason is one refusal the resolver produced, kept in the resolver's
// own vocabulary beside the management code it maps to. Both codes are
// reported; neither is rewritten into the other.
type ResolverReason struct {
	ManagementCode string `json:"managementCode"`
	ResolverCode   string `json:"resolverCode,omitempty"`
	Subject        string `json:"subject,omitempty"`
	Message        string `json:"message,omitempty"`
}

// ResolverOutcome is the single resolution the inventory reports. The
// inventory never resolves twice, never resolves in another order and never
// falls back to another provider.
type ResolverOutcome struct {
	Resolved        bool             `json:"resolved"`
	ReasonCodes     []string         `json:"reasonCodes,omitempty"`
	Reasons         []ResolverReason `json:"reasons,omitempty"`
	LockFormat      int              `json:"lockFormatVersion,omitempty"`
	LockFingerprint string           `json:"lockFingerprint,omitempty"`
	// PlanBuilt is nil when no lock was supplied: a runtime plan cannot be
	// built without one, so the runtime boundary is unknown rather than false.
	PlanBuilt       *bool    `json:"planBuilt,omitempty"`
	PlanReasonCodes []string `json:"planReasonCodes,omitempty"`
}

// Level statuses. "no-provider" is distinct from "multiple-providers", and an
// unknown provider is never reported as covered.
const (
	LevelProvided          = "provided"
	LevelNoProvider        = "no-provider"
	LevelMultipleProviders = "multiple-providers"
	LevelUnsupported       = "unsupported"
)

// Level is one data level the inventory reports coverage for: a database
// instance the environment declares, or a product feature it selects.
// Coverage comes from the declared capabilities, never from a name.
type Level struct {
	Kind        string   `json:"kind"`
	ID          string   `json:"id"`
	Provider    string   `json:"provider,omitempty"`
	Providers   []string `json:"providers,omitempty"`
	Status      string   `json:"status"`
	Requires    int      `json:"requires,omitempty"`
	Covered     int      `json:"covered,omitempty"`
	ReasonCodes []string `json:"reasonCodes,omitempty"`
}

// World is the part of ir.World the inventory reads. Naming it here keeps the
// package free of the whole IR and makes the input explicit.
type World struct {
	Plugins      []plugin.Definition
	Bundles      []plugin.Bundle
	Environments []environment.Environment
}

// Input is the read-only observation the inventory derives from.
type Input struct {
	World       World
	Environment core.CanonicalID
	// Lock is the resolved plugin lock, when the caller supplied one.
	Lock *plugin.Lock
	// ScanState is complete, partial or not-run. An empty value means complete
	// for a caller that has the whole World in hand.
	ScanState  string
	ObservedAt string
}

// Counts separates what was observed from the total. A nil total is unknown.
// It is never encoded as zero.
type Counts struct {
	PluginsDeclared int  `json:"pluginsDeclared"`
	PluginsSelected int  `json:"pluginsSelected"`
	PluginsResolved int  `json:"pluginsResolved"`
	PluginsWithGaps int  `json:"pluginsWithGaps"`
	PluginsTotal    *int `json:"pluginsTotal,omitempty"`
}

// Report is the whole read-only inventory.
type Report struct {
	SchemaVersion  string `json:"schemaVersion"`
	Environment    string `json:"environment"`
	ScanState      string `json:"scanState"`
	ScanObservedAt string `json:"scanObservedAt,omitempty"`
	// SettingsVersion is nil: TEI has no source for it yet, and a fabricated
	// value would make every recheck look current.
	SettingsVersion *string           `json:"settingsVersion"`
	Confirmed       bool              `json:"confirmed"`
	ScanReasons     []string          `json:"scanReasons,omitempty"`
	Resolver        ResolverOutcome   `json:"resolver"`
	Rows            []Row             `json:"rows"`
	Levels          []Level           `json:"levels,omitempty"`
	Totals          Counts            `json:"totals"`
	NotModelled     []string          `json:"notModelled"`
	Messages        map[string]string `json:"messages"`
	AuthorityLimits []string          `json:"authorityLimits"`
}

// Derive builds the inventory. It reads; it never writes, starts a plugin,
// grants permission or resolves a second time.
func Derive(in Input) (Report, error) {
	state := in.ScanState
	if state == "" {
		state = ScanComplete
	}
	switch state {
	case ScanComplete, ScanPartial, ScanNotRun:
	default:
		return Report{}, fmt.Errorf("unknown scan state %q", state)
	}
	if in.Environment == "" {
		return Report{}, fmt.Errorf("environment is required")
	}
	env, ok := findEnvironment(in.World.Environments, in.Environment)
	if !ok {
		return Report{}, fmt.Errorf("unknown environment %s", in.Environment)
	}

	catalog := plugin.Catalog{Plugins: in.World.Plugins, Bundles: in.World.Bundles}
	report := Report{
		SchemaVersion:   SchemaVersion,
		Environment:     string(env.ID),
		ScanState:       state,
		ScanObservedAt:  in.ObservedAt,
		NotModelled:     append([]string(nil), allNotModelled...),
		Messages:        messages(),
		AuthorityLimits: authorityLimits(),
	}
	switch state {
	case ScanPartial:
		report.ScanReasons = append(report.ScanReasons, ReasonScanIncomplete)
	case ScanNotRun:
		report.ScanReasons = append(report.ScanReasons, ReasonScanNotRun)
	}

	// One resolution, through the existing resolver.
	var resolved plugin.ResolvedSet
	var resolveErr error
	var plan runtime.EnvironmentPlan
	var planErr error
	planBuilt := false
	if in.Lock != nil {
		report.Resolver.LockFormat = in.Lock.FormatVersion
		if fp, err := plugin.EnvironmentLockFingerprint(*in.Lock, env.ID); err == nil {
			report.Resolver.LockFingerprint = fp
		}
		resolved, resolveErr = catalog.ResolveEnvironmentWithLock(env, *in.Lock)
		if resolveErr == nil {
			if p, err := runtime.BuildPlan(catalog, env, *in.Lock); err == nil {
				plan = p
				planBuilt = true
			} else {
				// A plan refusal is a lock or grant problem, not a second
				// resolution: report it beside the resolution the resolver
				// already made.
				planErr = err
			}
		}
	} else {
		resolved, resolveErr = catalog.ResolveEnvironment(env)
	}
	if resolveErr != nil {
		report.Resolver.Reasons = resolverReasons(resolveErr)
	} else {
		report.Resolver.Resolved = true
	}
	if planBuilt {
		built := true
		report.Resolver.PlanBuilt = &built
	} else if in.Lock != nil {
		built := false
		report.Resolver.PlanBuilt = &built
		if planErr != nil {
			for _, reason := range resolverReasons(planErr) {
				report.Resolver.Reasons = append(report.Resolver.Reasons, reason)
				report.Resolver.PlanReasonCodes = append(report.Resolver.PlanReasonCodes, reason.ManagementCode)
			}
			report.Resolver.PlanReasonCodes = sortedUnique(report.Resolver.PlanReasonCodes)
		}
	}
	if len(report.Resolver.Reasons) > 0 {
		report.Resolver.ReasonCodes = nil
		for _, reason := range report.Resolver.Reasons {
			report.Resolver.ReasonCodes = append(report.Resolver.ReasonCodes, reason.ManagementCode)
		}
		report.Resolver.ReasonCodes = sortedUnique(report.Resolver.ReasonCodes)
	}

	resolvedIDs := map[string]plugin.Identity{}
	for _, ident := range resolved.Plugins {
		resolvedIDs[string(ident.ID)] = ident
	}
	direct := map[string]bool{}
	for _, req := range env.Plugins {
		direct[string(req.ID)] = true
	}

	// Rows are the union of declared plugins and plugins the environment names
	// (directly or through the resolver). A plugin the environment selects but
	// the Workspace does not declare still gets a row: the reader has to see
	// that the declaration is missing.
	rowIDs := map[string]bool{}
	for _, p := range in.World.Plugins {
		rowIDs[string(p.ID)] = true
	}
	for id := range direct {
		rowIDs[id] = true
	}
	for id := range resolvedIDs {
		rowIDs[id] = true
	}

	refusalBySubject := map[string][]ResolverReason{}
	for _, reason := range report.Resolver.Reasons {
		refusalBySubject[reason.Subject] = append(refusalBySubject[reason.Subject], reason)
	}

	rows := make([]Row, 0, len(rowIDs))
	runtimeFacts := map[string]runtimeFact{}
	grantGapsByPlugin := map[string][]permission.Spec{}
	if planBuilt {
		runtimeFacts = runtimeSummary(plan)
		grantGapsByPlugin = grantGaps(plan)
	}
	for id := range rowIDs {
		row := Row{PluginID: id}
		definition, declared := exactDefinition(in.World.Plugins, id)
		if declared {
			row.Declared = true
			row.Version = definition.Version
			row.RuntimeKind = string(definition.Runtime.Kind)
			row.ArtifactDigest = definition.Artifact.Digest
			row.Roles, row.Capabilities = rolesOf(definition)
			row.Requires = requirementsOf(definition)
			row.External = externalsOf(definition)
			row.Evidence = append(row.Evidence, Evidence{
				Kind:        "declaration",
				Ref:         id,
				Fingerprint: plugin.DefinitionFingerprint(definition),
			})
			row.LockPinned = pinnedFor(in.Lock, env.ID, plugin.Identity{ID: definition.ID, Version: definition.Version}, &row)
			if fact, ok := runtimeFacts[id]; ok {
				row.RuntimeBoundary = fact.Boundary
				row.RuntimeTrust = fact.Trust
			}
			if in.Lock != nil && planBuilt {
				// The grant check reuses the plan's own capability entries, so
				// it cannot report a gap BuildPlan did not compute. Without a
				// plan the runtime policy is unknown, not empty.
				for _, spec := range grantGapsByPlugin[id] {
					row.ReasonCodes = append(row.ReasonCodes, ReasonPermissionMissing)
					row.Evidence = append(row.Evidence, Evidence{Kind: "grant", Ref: spec.Key()})
				}
			}
		}
		if ident, ok := resolvedIDs[id]; ok {
			row.Resolved = true
			row.Version = ident.Version
			if fact, ok := runtimeFacts[id]; ok {
				row.RuntimeBoundary = fact.Boundary
				row.RuntimeTrust = fact.Trust
			}
			row.Evidence = append(row.Evidence, Evidence{
				Kind: "resolution", Ref: id, Fingerprint: ident.Version,
			})
		}
		row.EnvSelected = direct[id] || row.Resolved
		for _, reason := range refusalBySubject[id] {
			row.ReasonCodes = append(row.ReasonCodes, reason.ManagementCode)
		}
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].PluginID < rows[j].PluginID })
	for i := range rows {
		finishRow(&rows[i], state, in.Lock != nil)
	}
	report.Rows = rows
	report.Levels = deriveLevels(env, in.World.Plugins)
	report.Totals = countsOf(rows, state)
	report.Confirmed = state == ScanComplete && len(report.ScanReasons) == 0 &&
		report.Resolver.Resolved && !anyGap(rows) && allLevelsCovered(report.Levels)

	for code := range messageCodes(report) {
		if _, ok := report.Messages[code]; !ok {
			report.Messages[code] = code
		}
	}
	return report, nil
}

func findEnvironment(envs []environment.Environment, id core.CanonicalID) (environment.Environment, bool) {
	for _, e := range envs {
		if e.ID == id {
			return e, true
		}
	}
	return environment.Environment{}, false
}

// exactDefinition returns the single definition of id. A repeated id is not
// chosen: the resolver reports it as a duplicate, and the inventory must not
// pick one of them.
func exactDefinition(defs []plugin.Definition, id string) (plugin.Definition, bool) {
	var found plugin.Definition
	count := 0
	for _, d := range defs {
		if string(d.ID) == id {
			found = d
			count++
		}
	}
	return found, count == 1
}

// pinnedFor reports whether the lock pins this exact plugin identity, and fills
// the lock-derived fields. A lock that has no environment entry or no matching
// plugin yields false, which is a fact about the lock, not about the plugin.
func pinnedFor(lock *plugin.Lock, envID core.CanonicalID, ident plugin.Identity, row *Row) *bool {
	if lock == nil {
		return nil
	}
	pinned := false
	for _, el := range lock.Environments {
		if el.Environment != envID {
			continue
		}
		for _, lp := range el.Plugins {
			if lp.ID != ident.ID {
				continue
			}
			if lp.Version != ident.Version {
				continue
			}
			pinned = true
			row.ArtifactDigest = lp.ArtifactDigest
			row.BuildPlanFingerprint = lp.BuildPlanFingerprint
		}
	}
	return &pinned
}

func requirementsOf(d plugin.Definition) []Requirement {
	out := make([]Requirement, 0, len(d.Requires))
	for _, r := range d.Requires {
		out = append(out, Requirement{Plugin: string(r.Plugin), CompatibleMajor: r.CompatibleMajor})
	}
	return out
}

func externalsOf(d plugin.Definition) []External {
	out := make([]External, 0, len(d.External))
	for _, e := range d.External {
		out = append(out, External{Scope: e.Scope, Kind: e.Kind, Name: e.Name})
	}
	return out
}

func rolesOf(d plugin.Definition) ([]string, []Capability) {
	kinds := map[string]bool{}
	caps := make([]Capability, 0, len(d.Provides))
	for _, c := range d.Provides {
		kinds[string(c.Kind)] = true
		caps = append(caps, Capability{
			Kind:              string(c.Kind),
			ID:                string(c.ID),
			Actions:           stringsOf(c.Actions),
			Observations:      stringsOf(c.Observations),
			Evidence:          stringsOf(c.Evidence),
			DatabaseProviders: stringsOf(c.DatabaseProviders),
			Features:          stringsOf(c.Features),
			Permissions:       permissionKeys(c.Permissions),
		})
	}
	roles := make([]string, 0, len(kinds))
	for kind := range kinds {
		roles = append(roles, kind)
	}
	sort.Strings(roles)
	return roles, caps
}

func stringsOf[T ~string](values []T) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = string(v)
	}
	sort.Strings(out)
	return out
}

func permissionKeys(specs []permission.Spec) []string {
	if len(specs) == 0 {
		return nil
	}
	out := make([]string, len(specs))
	for i, spec := range specs {
		out[i] = spec.Key()
	}
	sort.Strings(out)
	return out
}

func recordReasons(row *Row, reasons []ResolverReason) {
	for _, reason := range reasons {
		row.ReasonCodes = append(row.ReasonCodes, reason.ManagementCode)
	}
}

func finishRow(row *Row, state string, lockProvided bool) {
	row.ReasonCodes = sortedUnique(row.ReasonCodes)

	switch {
	case !row.Declared:
		row.ReasonCodes = append(row.ReasonCodes, ReasonPluginNotDeclared)
	case row.EnvSelected && !row.Resolved && len(row.ReasonCodes) == 0:
		row.ReasonCodes = append(row.ReasonCodes, ReasonDeclaredButUnresolvable)
	case !row.EnvSelected && !row.Resolved && len(row.ReasonCodes) == 0:
		row.ReasonCodes = append(row.ReasonCodes, ReasonEnvNotSelected)
	}

	if !lockProvided {
		row.ReasonCodes = append(row.ReasonCodes, ReasonRuntimeUnverified, ReasonGrantUnverified)
	} else if row.LockPinned != nil && !*row.LockPinned {
		row.ReasonCodes = append(row.ReasonCodes, ReasonRuntimeUnverified)
	}
	if row.RuntimeKind != "" && !runtimeKindSupported(row.RuntimeKind) {
		row.ReasonCodes = append(row.ReasonCodes, ReasonUnsupportedTarget)
	}
	if state != ScanComplete {
		row.Remaining = append(row.Remaining, "rescan-required")
	}
	row.NotModelled = append(row.NotModelled,
		NotModelledInstall, NotModelledEnable, NotModelledInitFailure,
		NotModelledInvocation, NotModelledEffect)

	row.ReasonCodes = sortedUnique(row.ReasonCodes)
	row.Remediation = sortedUnique(remediationFor(row))
	row.Impact = impactOf(row)
	row.Recheck = sortedUnique(recheckFor(row))
	row.Remaining = sortedUnique(row.Remaining)
	sort.Slice(row.Evidence, func(i, j int) bool {
		if row.Evidence[i].Kind != row.Evidence[j].Kind {
			return row.Evidence[i].Kind < row.Evidence[j].Kind
		}
		return row.Evidence[i].Ref < row.Evidence[j].Ref
	})
}

func runtimeKindSupported(kind string) bool {
	for _, valid := range plugin.RuntimeKinds {
		if string(valid) == kind {
			return true
		}
	}
	return false
}

func remediationFor(row *Row) []string {
	var out []string
	for _, code := range row.ReasonCodes {
		switch code {
		case ReasonPluginNotDeclared:
			out = append(out, "declare-plugin")
		case ReasonEnvNotSelected:
			out = append(out, "select-for-environment")
		case ReasonVersionUnsupported:
			out = append(out, "update-plugin-version")
		case ReasonDependencyCycle, ReasonDependencyUnresolved, ReasonContractIncompatible:
			out = append(out, "fix-dependency-declaration")
		case ReasonPermissionMissing:
			out = append(out, "grant-permission")
		case ReasonPermissionUnknown:
			out = append(out, "confirm-permission")
		case ReasonGrantUnverified:
			out = append(out, "supply-plugin-lock")
		case ReasonUnsupportedTarget:
			out = append(out, "choose-supported-runtime")
		case ReasonRuntimeUnverified:
			out = append(out, "supply-plugin-lock")
		case ReasonNotObserved:
			out = append(out, "read-invocation-evidence")
		}
	}
	return out
}

func impactOf(row *Row) string {
	switch {
	case row.Resolved && row.Declared && len(row.ReasonCodes) == 0:
		return "resolved"
	case !row.Resolved:
		return "no-data-access"
	default:
		return "unknown"
	}
}

func recheckFor(row *Row) []string {
	var out []string
	for _, code := range row.ReasonCodes {
		switch code {
		case ReasonEnvNotSelected, ReasonPermissionMissing, ReasonGrantUnverified:
			out = append(out, "after-settings-change")
		case ReasonVersionUnsupported, ReasonUnsupportedTarget:
			out = append(out, "after-plugin-update")
		case ReasonLockNotProvided, ReasonRuntimeUnverified:
			out = append(out, "after-lock-provided")
		case ReasonScanIncomplete, ReasonScanNotRun:
			out = append(out, "after-scan-complete")
		}
	}
	return out
}

func countsOf(rows []Row, state string) Counts {
	c := Counts{}
	for _, row := range rows {
		if row.Declared {
			c.PluginsDeclared++
		}
		if row.EnvSelected {
			c.PluginsSelected++
		}
		if row.Resolved {
			c.PluginsResolved++
		}
		if len(row.ReasonCodes) > 0 {
			c.PluginsWithGaps++
		}
	}
	if state == ScanComplete {
		total := len(rows)
		c.PluginsTotal = &total
	}
	return c
}

func anyGap(rows []Row) bool {
	for _, row := range rows {
		if len(row.ReasonCodes) > 0 {
			return true
		}
	}
	return false
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

func authorityLimits() []string {
	return []string{
		"diagnosis grants no permission and no acceptance",
		"installation and the enabled setting are not modelled; a row does not mean enabled",
		"real invocation, current grant and real effect are outside this inventory",
		"resolved does not mean conformance-observed or owner-accepted",
		"the parser rejects unknown Source fragments, so this inventory reports no byte retention",
	}
}
