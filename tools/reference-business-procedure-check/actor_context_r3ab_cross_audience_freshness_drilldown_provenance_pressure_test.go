package main

import (
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

const (
	actorContextR3ABRoleReferenceEvidence  = "reference-evidence"
	actorContextR3ABRoleRuntimeObservation = "runtime-observation"
	actorContextR3ABRoleTestOnlyContext    = "test-only-context"
	actorContextR3ABRoleCorroboration      = "corroboration"

	actorContextR3ABDerivedOperationalStop = "r3aa-operational-stop"
	actorContextR3ABDerivedContextIdentity = "r3x-context-identity"
	actorContextR3ABDerivedFreshnessAxes   = "r3y-freshness-authority-axes"
	actorContextR3ABDerivedPrecedenceGate  = "r3z-precedence-evidence-gate"
)

type actorContextR3ABProvenanceRootProbe struct {
	Identity    string
	Role        string
	ClaimKind   string
	Locator     string
	Fingerprint string
}

type actorContextR3ABDerivedNodeProbe struct {
	Identity string
	Kind     string
	Locator  string
}

type actorContextR3ABProvenanceLinkProbe struct {
	From     string
	To       string
	Relation string
}

type actorContextR3ABDrilldownProbe struct {
	Audience       string
	ReasonIdentity string
	DerivedNodes   []actorContextR3ABDerivedNodeProbe
	Roots          []actorContextR3ABProvenanceRootProbe
	Links          []actorContextR3ABProvenanceLinkProbe
	InlineRootIDs  []string
	AuthorityLimits []string
}

func TestActorContextR3ABHumanAndAgentShareFreshnessReasonAndProvenanceRoots(t *testing.T) {
	human := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceHuman)
	agent := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)

	if human.ReasonIdentity == "" || human.ReasonIdentity != agent.ReasonIdentity {
		t.Fatalf("Human / Agent freshness reason identity diverged: human=%q agent=%q",
			human.ReasonIdentity, agent.ReasonIdentity)
	}
	if !reflect.DeepEqual(human.DerivedNodes, agent.DerivedNodes) {
		t.Fatalf("Human / Agent drill-down reached different derived analysis nodes: human=%+v agent=%+v",
			human.DerivedNodes, agent.DerivedNodes)
	}
	if !reflect.DeepEqual(human.Roots, agent.Roots) {
		t.Fatalf("Human / Agent drill-down reached different provenance roots: human=%+v agent=%+v",
			human.Roots, agent.Roots)
	}
	if !reflect.DeepEqual(human.Links, agent.Links) {
		t.Fatalf("Human / Agent drill-down provenance links diverged: human=%+v agent=%+v",
			human.Links, agent.Links)
	}
}

func TestActorContextR3ABInlineDisclosureMayDifferWithoutChangingSharedRoots(t *testing.T) {
	human := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceHuman)
	agent := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)

	if reflect.DeepEqual(human.InlineRootIDs, agent.InlineRootIDs) {
		t.Fatal("test setup requires purpose-bound Human / Agent inline disclosure to differ")
	}
	if len(human.InlineRootIDs) >= len(agent.InlineRootIDs) {
		t.Fatalf("Human compact drill-down unexpectedly disclosed at least as much as Agent: human=%v agent=%v",
			human.InlineRootIDs, agent.InlineRootIDs)
	}
	if !reflect.DeepEqual(human.Roots, agent.Roots) {
		t.Fatal("inline disclosure difference changed shared provenance root set")
	}
}

func TestActorContextR3ABPriorObservationRemainsRuntimeEvidenceNotCurrentOutcome(t *testing.T) {
	drilldown := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)

	var observation *actorContextR3ABProvenanceRootProbe
	for i := range drilldown.Roots {
		if drilldown.Roots[i].Role == actorContextR3ABRoleRuntimeObservation {
			observation = &drilldown.Roots[i]
			break
		}
	}
	if observation == nil {
		t.Fatalf("R3AB lost prior Authorization Observation root: %+v", drilldown.Roots)
	}
	if observation.ClaimKind != "prior-authorization-observation:allowed" {
		t.Fatalf("R3AB changed prior Observation meaning: %+v", observation)
	}

	core := actorContextR3ABFreshnessCore(t)
	if core.RuntimeOutcome != "" {
		t.Fatalf("R3AB promoted prior Observation outcome to current RuntimeOutcome: %+v", core)
	}
}

func TestActorContextR3ABLegacyPolicyRemainsCorroborationOnly(t *testing.T) {
	drilldown := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)

	var policy *actorContextR3ABProvenanceRootProbe
	for i := range drilldown.Roots {
		if drilldown.Roots[i].Identity == "legacy-policy:execute:order_entry.submit_order:allow" {
			policy = &drilldown.Roots[i]
			break
		}
	}
	if policy == nil {
		t.Fatalf("R3AB lost Legacy policy corroboration root: %+v", drilldown.Roots)
	}
	if policy.Role != actorContextR3ABRoleCorroboration {
		t.Fatalf("Legacy policy was promoted beyond corroboration: %+v", policy)
	}
	for _, forbidden := range []string{"freshness-authority", "action-authority", "current-runtime-outcome"} {
		if strings.Contains(policy.ClaimKind, forbidden) {
			t.Fatalf("Legacy policy corroboration acquired forbidden claim %q: %+v", forbidden, policy)
		}
	}
}

func TestActorContextR3ABExplicitActionContextRemainsTestOnlyInput(t *testing.T) {
	drilldown := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)

	for _, root := range drilldown.Roots {
		if root.Identity != "test-only-context:r3x:order-entry.order-handling/BP-06:subject-order-viewer:execute:order_entry.submit_order" {
			continue
		}
		if root.Role != actorContextR3ABRoleTestOnlyContext {
			t.Fatalf("R3X explicit action context was promoted to Repository authority: %+v", root)
		}
		if root.ClaimKind != "synthetic-authorization-context-input" {
			t.Fatalf("R3X explicit action context lost test-only status: %+v", root)
		}
		return
	}
	t.Fatal("R3AB lost explicit test-only Authorization context root")
}

func TestActorContextR3ABLocatorMoveDoesNotChangeRootIdentity(t *testing.T) {
	drilldown := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceHuman)
	if len(drilldown.Roots) == 0 {
		t.Fatal("R3AB test setup has no provenance roots")
	}

	before := drilldown.Roots[0]
	after := before
	after.Locator = "reference/business-system-design/moved/order-entry-business-procedure.json"

	if actorContextR3ABRootIdentity(before) != actorContextR3ABRootIdentity(after) {
		t.Fatalf("locator-only move changed provenance root identity: before=%+v after=%+v", before, after)
	}
	if before.Locator == after.Locator {
		t.Fatal("test setup did not change locator")
	}
}

func TestActorContextR3ABDerivedPlanNodesAreNotProvenanceAuthorityRoots(t *testing.T) {
	drilldown := actorContextR3ABBuildFreshnessDrilldown(t, actorContextR3PAudienceAgent)

	for _, root := range drilldown.Roots {
		if strings.HasPrefix(root.Locator, "plan/ACTOR_CONTEXT_R3") {
			t.Fatalf("derived R3 plan artifact leaked into authority/root set: %+v", root)
		}
	}
	for _, node := range drilldown.DerivedNodes {
		if !strings.HasPrefix(node.Locator, "plan/ACTOR_CONTEXT_R3") {
			t.Fatalf("derived analysis node lost explicit design-artifact locator: %+v", node)
		}
	}
}

func TestActorContextR3ABDrilldownHasNoTruthWinnerOrActionShortcut(t *testing.T) {
	typ := reflect.TypeOf(actorContextR3ABDrilldownProbe{})
	for _, forbidden := range []string{
		"TrustedRoot",
		"CanonicalTruth",
		"PreferredSource",
		"Winner",
		"StrongerAuthority",
		"ResolvedByAudience",
		"NextAction",
		"HandoffOwner",
		"Approved",
		"Authorized",
		"CanExecute",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3AB drill-down introduced forbidden truth/authority shortcut %q", forbidden)
		}
	}
}

func actorContextR3ABBuildFreshnessDrilldown(
	t *testing.T,
	audience string,
) actorContextR3ABDrilldownProbe {
	t.Helper()

	core := actorContextR3ABFreshnessCore(t)
	view, err := actorContextR3PSelectBoundedStopView(
		core,
		audience,
		map[string]string{
			actorContextR3PAudienceHuman: actorContextR3PPurposeOrientation,
			actorContextR3PAudienceAgent: actorContextR3PPurposeResolutionInspection,
		}[audience],
	)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(view.Payload.MachineReasons, []string{actorContextR3AAFreshnessReason}) {
		t.Fatalf("R3AB test setup lost freshness stop reason for %s: %+v", audience, view)
	}

	root := repoPath()
	procedurePath := "reference/business-system-design/order-entry-business-procedure.json"
	overlayPath := "reference/business-system-design/order-entry-protected-target-overlay.json"
	policyPath := "reference/legacy-auth/authorization.json"

	procedureFingerprint, err := actorContextR0Fingerprint(filepath.Join(root, procedurePath))
	if err != nil {
		t.Fatal(err)
	}
	overlayFingerprint, err := actorContextR0Fingerprint(filepath.Join(root, overlayPath))
	if err != nil {
		t.Fatal(err)
	}
	policyFingerprint, err := actorContextR0Fingerprint(filepath.Join(root, policyPath))
	if err != nil {
		t.Fatal(err)
	}

	observation := actorContextR3XBaseObservation("allowed")
	observationFingerprint, err := actorContextR3QHash(observation)
	if err != nil {
		t.Fatal(err)
	}
	context := actorContextR3XBaseContext()
	contextFingerprint, err := actorContextR3QHash(context)
	if err != nil {
		t.Fatal(err)
	}

	policyRaw, err := osReadFileForR3AB(filepath.Join(root, policyPath))
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{
		`"action": "execute"`,
		`"resource": "order_entry.submit_order"`,
		`"effect": "allow"`,
	} {
		if !strings.Contains(policyRaw, required) {
			t.Fatalf("R3AB test setup changed: Legacy policy lost %s", required)
		}
	}

	roots := []actorContextR3ABProvenanceRootProbe{
		{
			Identity:    "reference:business-procedure:order-entry.order-handling/BP-06:authorization-requirement",
			Role:        actorContextR3ABRoleReferenceEvidence,
			ClaimKind:   "authorization-requirement-source",
			Locator:     procedurePath,
			Fingerprint: procedureFingerprint,
		},
		{
			Identity:    "reference:protected-resource:order-entry.order-handling/BP-06:order_entry.submit_order",
			Role:        actorContextR3ABRoleReferenceEvidence,
			ClaimKind:   "protected-resource-source",
			Locator:     overlayPath,
			Fingerprint: overlayFingerprint,
		},
		{
			Identity:    "runtime-observation:prior-authorization:order-entry.order-handling/BP-06:subject-order-viewer:execute:order_entry.submit_order",
			Role:        actorContextR3ABRoleRuntimeObservation,
			ClaimKind:   "prior-authorization-observation:allowed",
			Locator:     "runtime:test-only/prior-authorization-observation",
			Fingerprint: observationFingerprint,
		},
		{
			Identity:    "test-only-context:r3x:order-entry.order-handling/BP-06:subject-order-viewer:execute:order_entry.submit_order",
			Role:        actorContextR3ABRoleTestOnlyContext,
			ClaimKind:   "synthetic-authorization-context-input",
			Locator:     "tools/reference-business-procedure-check/actor_context_r3x_authorization_context_identity_pressure_test.go",
			Fingerprint: contextFingerprint,
		},
		{
			Identity:    "legacy-policy:execute:order_entry.submit_order:allow",
			Role:        actorContextR3ABRoleCorroboration,
			ClaimKind:   "authorization-policy-rule-corroboration",
			Locator:     policyPath,
			Fingerprint: policyFingerprint,
		},
	}
	actorContextR3ABSortRoots(roots)

	nodes := []actorContextR3ABDerivedNodeProbe{
		{
			Identity: actorContextR3ABDerivedOperationalStop,
			Kind:     "derived-operational-stop",
			Locator:  "plan/ACTOR_CONTEXT_R3AA_AUTHORIZATION_FRESHNESS_STOP_HANDOFF_2026-09.md",
		},
		{
			Identity: actorContextR3ABDerivedContextIdentity,
			Kind:     "derived-context-identity-analysis",
			Locator:  "plan/ACTOR_CONTEXT_R3X_AUTHORIZATION_CONTEXT_IDENTITY_BOUNDARY_2026-09.md",
		},
		{
			Identity: actorContextR3ABDerivedFreshnessAxes,
			Kind:     "derived-freshness-authority-analysis",
			Locator:  "plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md",
		},
		{
			Identity: actorContextR3ABDerivedPrecedenceGate,
			Kind:     "derived-precedence-evidence-gate",
			Locator:  "plan/ACTOR_CONTEXT_R3Z_AUTHORIZATION_FRESHNESS_PRECEDENCE_EVIDENCE_GATE_2026-09.md",
		},
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].Identity < nodes[j].Identity })

	reasonIdentity := actorContextR3ABFreshnessReasonIdentity(core)
	links := []actorContextR3ABProvenanceLinkProbe{
		{From: reasonIdentity, To: actorContextR3ABDerivedOperationalStop, Relation: "explained-by"},
		{From: actorContextR3ABDerivedOperationalStop, To: actorContextR3ABDerivedContextIdentity, Relation: "requires-context-identity"},
		{From: actorContextR3ABDerivedOperationalStop, To: actorContextR3ABDerivedFreshnessAxes, Relation: "requires-freshness-authority-analysis"},
		{From: actorContextR3ABDerivedOperationalStop, To: actorContextR3ABDerivedPrecedenceGate, Relation: "bounded-by-precedence-evidence-gate"},
		{From: actorContextR3ABDerivedContextIdentity, To: roots[actorContextR3ABRootIndex(roots, actorContextR3ABRoleRuntimeObservation)].Identity, Relation: "observes-prior-tuple"},
		{From: actorContextR3ABDerivedContextIdentity, To: actorContextR3ABFindRoot(roots, actorContextR3ABRoleTestOnlyContext).Identity, Relation: "compares-current-test-context"},
		{From: actorContextR3ABDerivedFreshnessAxes, To: actorContextR3ABFindRootByClaim(roots, "authorization-requirement-source").Identity, Relation: "pressured-by-reference-meaning"},
		{From: actorContextR3ABDerivedFreshnessAxes, To: actorContextR3ABFindRootByClaim(roots, "protected-resource-source").Identity, Relation: "pressured-by-reference-meaning"},
		{From: actorContextR3ABDerivedPrecedenceGate, To: actorContextR3ABFindRoot(roots, actorContextR3ABRoleCorroboration).Identity, Relation: "audits-without-promoting"},
	}
	sort.Slice(links, func(i, j int) bool {
		if links[i].From != links[j].From {
			return links[i].From < links[j].From
		}
		if links[i].To != links[j].To {
			return links[i].To < links[j].To
		}
		return links[i].Relation < links[j].Relation
	})

	inline := []string{
		actorContextR3ABFindRootByClaim(roots, "authorization-requirement-source").Identity,
		actorContextR3ABFindRootByClaim(roots, "protected-resource-source").Identity,
	}
	if audience == actorContextR3PAudienceAgent {
		inline = append(inline,
			actorContextR3ABFindRoot(roots, actorContextR3ABRoleRuntimeObservation).Identity,
			actorContextR3ABFindRoot(roots, actorContextR3ABRoleTestOnlyContext).Identity,
			actorContextR3ABFindRoot(roots, actorContextR3ABRoleCorroboration).Identity,
		)
	}
	sort.Strings(inline)

	return actorContextR3ABDrilldownProbe{
		Audience:       audience,
		ReasonIdentity: reasonIdentity,
		DerivedNodes:   nodes,
		Roots:          roots,
		Links:          links,
		InlineRootIDs:  inline,
		AuthorityLimits: []string{
			"audience-specific-disclosure-does-not-create-audience-specific-truth",
			"derived-plan-artifact-is-not-provenance-authority-root",
			"reference-pressure-input-is-not-promoted-to-production-canonical-authority",
			"test-only-action-context-is-not-current-business-application-action-authority",
			"legacy-policy-is-corroboration-not-freshness-authority",
			"prior-observation-is-not-current-outcome-when-freshness-authority-unresolved",
			"locator-change-does-not-change-root-identity",
		},
	}
}

func actorContextR3ABFreshnessCore(t *testing.T) actorContextR3OStopProjectionCoreProbe {
	t.Helper()

	root := repoPath()
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	fixture, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	requirement, err := deriveActorContextR3BAuthorizationRequirementProbe(fixture, "BP-06")
	if err != nil {
		t.Fatal(err)
	}
	overlay, err := readApplicationPatternTargetOverlayProbe(
		filepath.Join(root, "reference", "business-system-design", "order-entry-protected-target-overlay.json"),
	)
	if err != nil {
		t.Fatal(err)
	}
	targets := actorContextR3BTargetsForStep(overlay, "BP-06")
	if requirement.Status != actorContextR3BAuthorizationRequired ||
		!reflect.DeepEqual(targets, []string{"order_entry.submit_order"}) {
		t.Fatalf("R3AB setup changed: requirement=%+v targets=%v", requirement, targets)
	}

	authority := actorContextR3YAnalyzeFreshnessAuthority(
		actorContextR3YTupleMatchImpact(),
		actorContextR3YFreshnessPressureProbe{StaticSemanticChanged: true},
		nil,
	)
	if authority.ReevaluationAuthorityStatus != actorContextR3YReevaluationUnresolved ||
		authority.ReuseAuthorityStatus != actorContextR3YReuseUnresolved {
		t.Fatalf("R3AB setup changed: freshness authority=%+v", authority)
	}

	explanation := actorContextR3MBuildStopExplanation(
		"order-entry.order-handling/BP-06",
		"authorization-evaluation-freshness-authority",
		[]string{actorContextR3AAFreshnessReason},
		[]actorContextR3MAuthorityFactProbe{
			{
				Kind:      "authorization-requirement",
				Value:     requirement.Status,
				SourceRef: requirement.RequirementSource,
			},
			{
				Kind:      "protected-resource",
				Value:     targets[0],
				SourceRef: "reference-pressure-overlay:BP-06",
			},
			{
				Kind:      "prior-observation-reuse-authority",
				Value:     authority.ReuseAuthorityStatus,
				SourceRef: "derived:r3y",
			},
		},
		[]string{
			"reference/business-system-design/order-entry-business-procedure.json",
			"reference/business-system-design/order-entry-protected-target-overlay.json",
			"runtime:test-only/prior-authorization-observation",
		},
	)
	taxonomy := actorContextR3NClassifyStop(actorContextR3NStopInputProbe{
		BusinessAddress: "order-entry.order-handling/BP-06",
		BlockingReasons: []string{actorContextR3AAFreshnessReason},
	})
	core, err := actorContextR3OBuildCommonStopCore(explanation, taxonomy)
	if err != nil {
		t.Fatal(err)
	}
	return core
}

func actorContextR3ABFreshnessReasonIdentity(core actorContextR3OStopProjectionCoreProbe) string {
	return strings.Join([]string{
		"stop-reason",
		core.StopStage,
		core.StopClass,
		strings.Join(actorContextR0SortedUnique(core.MachineReasons), ","),
		strings.Join(actorContextR0SortedUnique(core.ResolutionNeedClasses), ","),
	}, "|")
}

func actorContextR3ABRootIdentity(root actorContextR3ABProvenanceRootProbe) string {
	return root.Identity + "|" + root.Role + "|" + root.ClaimKind
}

func actorContextR3ABSortRoots(values []actorContextR3ABProvenanceRootProbe) {
	sort.Slice(values, func(i, j int) bool {
		return actorContextR3ABRootIdentity(values[i]) < actorContextR3ABRootIdentity(values[j])
	})
}

func actorContextR3ABFindRoot(
	values []actorContextR3ABProvenanceRootProbe,
	role string,
) actorContextR3ABProvenanceRootProbe {
	for _, value := range values {
		if value.Role == role {
			return value
		}
	}
	return actorContextR3ABProvenanceRootProbe{}
}

func actorContextR3ABFindRootByClaim(
	values []actorContextR3ABProvenanceRootProbe,
	claim string,
) actorContextR3ABProvenanceRootProbe {
	for _, value := range values {
		if value.ClaimKind == claim {
			return value
		}
	}
	return actorContextR3ABProvenanceRootProbe{}
}

func actorContextR3ABRootIndex(
	values []actorContextR3ABProvenanceRootProbe,
	role string,
) int {
	for i := range values {
		if values[i].Role == role {
			return i
		}
	}
	return -1
}

func osReadFileForR3AB(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}
