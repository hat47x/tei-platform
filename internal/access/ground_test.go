package access_test

import (
	"strings"
	"testing"

	"tei/internal/access"
	"tei/internal/core"
)

const lineage = core.CanonicalID("tei.frozen_world.lineage")

// declaredWorkdir is the environment's declaration: a logical name, not a path.
func declaredWorkdir() access.Document {
	return access.Document{Application: "order_entry", Grants: []access.Entry{{
		IssuedBy:     "environment:development",
		Kind:         "filesystem",
		Access:       "write",
		Scope:        "migration-workdir",
		CapabilityID: lineage,
	}}}
}

func issuedFor(scope string) access.Document {
	return access.Document{Grants: []access.Entry{{
		IssuedBy:     "script:build-frozen-api",
		Kind:         "filesystem",
		Access:       "write",
		Scope:        scope,
		CapabilityID: lineage,
	}}}
}

func TestGroundRejectsAnIssuanceTheDeclarationDoesNotAllow(t *testing.T) {
	// The declaration names a logical scope, so a concrete path is not covered by
	// it as written. Without a resolution nothing can tie the two together.
	err := access.Ground(issuedFor("/tmp/run/observed.json"), declaredWorkdir(), nil, lineage)
	if err == nil {
		t.Fatal("an issuance outside the declarations must be refused")
	}
	if !strings.Contains(err.Error(), "not inside anything the declaration allows") {
		t.Fatalf("the refusal must say what is missing: %v", err)
	}
}

func TestGroundAcceptsAnIssuanceInsideAResolvedDeclaration(t *testing.T) {
	resolutions := []access.Resolution{{
		ResolvedBy: "host:build-frozen-api",
		Declared:   "migration-workdir",
		Root:       "/tmp/run",
	}}
	if err := access.Ground(issuedFor("/tmp/run/observed.json"), declaredWorkdir(), resolutions, lineage); err != nil {
		t.Fatalf("an issuance inside the resolved declaration must be accepted: %v", err)
	}
}

func TestGroundRejectsAnIssuanceOutsideAResolvedDeclaration(t *testing.T) {
	resolutions := []access.Resolution{{
		ResolvedBy: "host:build-frozen-api",
		Declared:   "migration-workdir",
		Root:       "/tmp/run",
	}}
	if err := access.Ground(issuedFor("/tmp/other/observed.json"), declaredWorkdir(), resolutions, lineage); err == nil {
		t.Fatal("a resolution must not widen the declared scope: a sibling directory stays outside")
	}
}

func TestGroundRejectsAnIssuanceWithoutADeclaration(t *testing.T) {
	err := access.Ground(issuedFor("/tmp/run/observed.json"), access.Document{Grants: []access.Entry{}}, nil, lineage)
	if err == nil {
		t.Fatal("an issuance with nothing to stay inside must be refused")
	}
}

func TestGroundAcceptsAnEmptyIssuance(t *testing.T) {
	// Asking for nothing needs nothing to stay inside of. Otherwise "no grants"
	// would be refused while it grants no access at all.
	if err := access.Ground(access.Document{Grants: []access.Entry{}}, access.Document{Grants: []access.Entry{}}, nil, lineage); err != nil {
		t.Fatalf("an empty issuance must be accepted: %v", err)
	}
}

func TestGroundRejectsAResolutionOfAnUndeclaredScope(t *testing.T) {
	resolutions := []access.Resolution{{ResolvedBy: "host:build", Declared: "somewhere-else", Root: "/tmp/run"}}
	err := access.Ground(issuedFor("/tmp/run/observed.json"), declaredWorkdir(), resolutions, lineage)
	if err == nil || !strings.Contains(err.Error(), "which no declaration names") {
		t.Fatalf("a resolution must map a declared scope: %v", err)
	}
}

func TestGroundRejectsAResolutionTheCapabilityMadeForItself(t *testing.T) {
	resolutions := []access.Resolution{{ResolvedBy: string(lineage), Declared: "migration-workdir", Root: "/tmp/run"}}
	err := access.Ground(issuedFor("/tmp/run/observed.json"), declaredWorkdir(), resolutions, lineage)
	if err == nil || !strings.Contains(err.Error(), "which is the capability that wants to use it") {
		t.Fatalf("a capability must not resolve its own declaration: %v", err)
	}
}

func TestGroundRejectsAResolutionThatNamesNoResolver(t *testing.T) {
	resolutions := []access.Resolution{{Declared: "migration-workdir", Root: "/tmp/run"}}
	err := access.Ground(issuedFor("/tmp/run/observed.json"), declaredWorkdir(), resolutions, lineage)
	if err == nil || !strings.Contains(err.Error(), "does not name who resolved it") {
		t.Fatalf("a resolution must name who made it: %v", err)
	}
}

func TestGroundStillRefusesASelfIssuedGrant(t *testing.T) {
	document := issuedFor("/tmp/run/observed.json")
	document.Grants[0].IssuedBy = string(lineage)
	if err := access.Ground(document, declaredWorkdir(), nil, lineage); err == nil {
		t.Fatal("grounding must not replace the self-issuance refusal")
	}
}

func TestGroundRefusesADeclarationForAnotherCapability(t *testing.T) {
	// The environment declared what migration.atlas may touch. That says nothing
	// about what the lineage exporter may touch, so it grounds nothing here.
	declared := access.Document{Grants: []access.Entry{{
		IssuedBy:     "environment:development",
		Kind:         "filesystem",
		Access:       "write",
		Scope:        "/tmp/run",
		CapabilityID: core.MustCanonicalID("migration.atlas"),
	}}}
	err := access.Ground(issuedFor("/tmp/run/observed.json"), declared, nil, lineage)
	if err == nil || !strings.Contains(err.Error(), "no declaration names tei.frozen_world.lineage") {
		t.Fatalf("a declaration for another capability must not ground this one: %v", err)
	}
}

func TestDeclaresReportsWhatTheDocumentCovers(t *testing.T) {
	if !access.Declares(declaredWorkdir(), lineage) {
		t.Fatal("the declaration names the lineage exporter")
	}
	if access.Declares(declaredWorkdir(), core.MustCanonicalID("migration.atlas")) {
		t.Fatal("the declaration does not name migration.atlas")
	}
	if access.Declares(access.Document{}, lineage) {
		t.Fatal("an empty document names nothing")
	}
}
