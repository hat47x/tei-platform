package pluginbuildsupply

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

func fixtureProvider(id, digestChar string) Provider {
	return Provider{
		ID: core.MustCanonicalID(id), Version: "1.0.0", Implementation: "impl", Contract: "tei.plugin-build",
		ContractMajor: 1, AuthorityScope: authorityBuildTool, ArtifactDigest: "sha256:" + strings.Repeat(digestChar, 64),
		RecipeKinds: []string{"go-package"}, Targets: []string{"host-process/linux/amd64"}, Profiles: []string{"reference"},
	}
}

var fixtureRequest = BuilderRequest{Contract: "tei.plugin-build", CompatibleMajor: 1, RecipeKind: "go-package", Target: "host-process/linux/amd64", Profile: "reference"}

func TestResolveBuilderStopsOnAmbiguityWhateverTheOrder(t *testing.T) {
	a, b := fixtureProvider("tei.builder.a", "a"), fixtureProvider("tei.builder.b", "b")
	_, errAB := ResolveBuilder(fixtureRequest, []Provider{a, b})
	_, errBA := ResolveBuilder(fixtureRequest, []Provider{b, a})
	if !errors.Is(errAB, ErrAmbiguousProvider) || errAB.Error() != errBA.Error() {
		t.Fatalf("ambiguity must stop the same way in both orders: %v / %v", errAB, errBA)
	}
	req := fixtureRequest
	req.ExplicitProvider = b.ID
	got, err := ResolveBuilder(req, []Provider{a, b})
	if err != nil || got.ID != b.ID {
		t.Fatalf("explicit provider: %v %v", got.ID, err)
	}
	wrongScope := a
	wrongScope.AuthorityScope = "runtime"
	if _, err := ResolveBuilder(fixtureRequest, []Provider{wrongScope}); !errors.Is(err, ErrNoProvider) {
		t.Fatalf("a provider outside build-tool scope must not be eligible: %v", err)
	}
}

func TestLockIsStableAndRevocationSeparatesReproductionFromUse(t *testing.T) {
	a, b := fixtureProvider("tei.builder.a", "a"), fixtureProvider("tei.builder.b", "b")
	lock, err := NewResolutionLock("intent", a)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := fingerprintJSON(lock)
	if _, err := ResolveBuilder(fixtureRequest, []Provider{b, a}); !errors.Is(err, ErrAmbiguousProvider) {
		t.Fatalf("fresh resolution after a provider was added: %v", err)
	}
	replayed, err := ReplayLock(lock, fixtureRequest, []Provider{b, a})
	if err != nil {
		t.Fatal(err)
	}
	if after, _ := fingerprintJSON(replayed); after != before {
		t.Fatalf("the added provider changed the fixed record")
	}
	drifted := a
	drifted.ArtifactDigest = "sha256:" + strings.Repeat("c", 64)
	if _, err := ReplayLock(lock, fixtureRequest, []Provider{drifted, b}); !errors.Is(err, ErrLockDrift) {
		t.Fatalf("a drifted provider must stop the replay: %v", err)
	}
	trust := TrustStore{Revoked: []Revocation{{ProviderID: a.ID, ArtifactDigest: a.ArtifactDigest, Reason: "key compromised"}}}
	if _, err := ReplayLock(lock, fixtureRequest, []Provider{a}); err != nil {
		t.Fatalf("reproduction must stay possible after revocation: %v", err)
	}
	if err := trust.Admit(lock); !errors.Is(err, ErrBuilderRevoked) {
		t.Fatalf("use after revocation must be refused: %v", err)
	}
	host := NewHost("unused", t.TempDir())
	plan := plugin.BuildPlan{Builder: lock.Builder}
	if _, err := host.Run(context.Background(), AttemptSpec{Plan: plan, Lock: lock, Trust: trust}); !errors.Is(err, ErrBuilderRevoked) {
		t.Fatalf("a revoked builder must not be launched: %v", err)
	}
	if host.IssuedCount() != 0 {
		t.Fatalf("a refused launch must not issue an occurrence")
	}
}

func fabricate(t *testing.T, host *Host, plan, outcome, digest string) AttemptRecord {
	t.Helper()
	id, ordinal, err := host.issue(plan)
	if err != nil {
		t.Fatal(err)
	}
	r := AttemptRecord{Schema: attemptSchema, PlanFingerprint: plan, OccurrenceID: id, Ordinal: ordinal, Mode: ModeBuild, Outcome: outcome, ArtifactDigest: digest}
	if outcome == OutcomeSucceeded {
		r.Termination = Termination{Exited: true}
	} else {
		r.Termination = Termination{Exited: true, ExitCode: 2}
	}
	if r.Fingerprint, err = attemptFingerprint(r); err != nil {
		t.Fatal(err)
	}
	host.finish(id, outcome)
	return r
}

func TestAttemptHistoryRefusesFailuresAndDoubleCounting(t *testing.T) {
	host := NewHost("unused", t.TempDir())
	plan := "plan-1"
	d := sha256Digest([]byte("artifact"))
	failed := fabricate(t, host, plan, OutcomeFailed, "")
	one := fabricate(t, host, plan, OutcomeSucceeded, d)
	h := History{PlanFingerprint: plan}
	for _, r := range []AttemptRecord{failed, one} {
		if err := h.Append(host, r); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := DeriveReproducibility(host, h); !errors.Is(err, ErrInsufficientOccurrences) {
		t.Fatalf("one success and one failure must not give reproducibility: %v", err)
	}
	if err := h.Append(host, one); !errors.Is(err, ErrOccurrenceCounted) {
		t.Fatalf("the same attempt must not be counted twice: %v", err)
	}
	renumbered := one
	renumbered.Ordinal = 99
	renumbered.Fingerprint, _ = attemptFingerprint(renumbered)
	if err := h.Append(host, renumbered); !errors.Is(err, ErrNotIssued) {
		t.Fatalf("a copied attempt under a new ordinal must be refused: %v", err)
	}
	forged := one
	forged.OccurrenceID = strings.Repeat("0", 32)
	forged.Fingerprint, _ = attemptFingerprint(forged)
	if err := h.Append(host, forged); !errors.Is(err, ErrNotIssued) {
		t.Fatalf("an occurrence the host did not issue must be refused: %v", err)
	}
	altered := one
	altered.ArtifactDigest = sha256Digest([]byte("other"))
	if err := h.Append(host, altered); !errors.Is(err, ErrRecordAltered) {
		t.Fatalf("an altered record must be refused: %v", err)
	}
	if _, err := DeriveReproducibility(host, History{PlanFingerprint: plan, Attempts: []AttemptRecord{one, one}}); !errors.Is(err, ErrOccurrenceCounted) {
		t.Fatalf("a history that lists one attempt twice must not give reproducibility: %v", err)
	}
	other := fabricate(t, host, "plan-2", OutcomeSucceeded, d)
	if err := h.Append(host, other); !errors.Is(err, ErrPlanMismatch) {
		t.Fatalf("the same bytes from another plan are not an attempt of this plan: %v", err)
	}
	two := fabricate(t, host, plan, OutcomeSucceeded, d)
	if err := h.Append(host, two); err != nil {
		t.Fatal(err)
	}
	rep, err := DeriveReproducibility(host, h)
	if err != nil || !rep.Reproducible || rep.Occurrences[0] == rep.Occurrences[1] {
		t.Fatalf("two distinct successful occurrences: %+v %v", rep, err)
	}
	h2 := History{PlanFingerprint: plan}
	for _, r := range []AttemptRecord{fabricate(t, host, plan, OutcomeSucceeded, d), fabricate(t, host, plan, OutcomeSucceeded, sha256Digest([]byte("x")))} {
		if err := h2.Append(host, r); err != nil {
			t.Fatal(err)
		}
	}
	mismatch, err := DeriveReproducibility(host, h2)
	if !errors.Is(err, ErrNotReproducible) || mismatch.Reproducible || mismatch.ArtifactDigest != "" {
		t.Fatalf("different artifacts must be recorded as a mismatch: %+v %v", mismatch, err)
	}
	store := Store{Root: t.TempDir()}
	if _, err := store.PutReproducible(mismatch, []byte("artifact")); !errors.Is(err, ErrNotStorable) {
		t.Fatalf("a mismatch must not be stored: %v", err)
	}
}

func TestStoreRefusesCorruptionWithoutRepair(t *testing.T) {
	store := Store{Root: t.TempDir()}
	data := []byte("artifact bytes")
	rep := Reproducibility{Reproducible: true, ArtifactDigest: sha256Digest(data)}
	put, err := store.PutReproducible(rep, data)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(put.Locator, "sha256/") || strings.Contains(put.Locator, store.Root) {
		t.Fatalf("locator must be store-relative: %s", put.Locator)
	}
	got, _, err := store.Get(put.Digest)
	if err != nil || string(got) != string(data) {
		t.Fatalf("get: %v", err)
	}
	if _, err := store.PutReproducible(rep, []byte("other bytes")); !errors.Is(err, ErrNotStorable) {
		t.Fatalf("bytes that are not the established artifact: %v", err)
	}
	path := filepath.Join(store.Root, filepath.FromSlash(put.Locator))
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("tampered bytes"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Get(put.Digest); !errors.Is(err, ErrStoreCorrupt) {
		t.Fatalf("a corrupt object must be refused: %v", err)
	}
	if _, err := store.PutReproducible(rep, data); !errors.Is(err, ErrStoreCorrupt) {
		t.Fatalf("a corrupt object must not be repaired by a put: %v", err)
	}
	if raw, _ := os.ReadFile(path); string(raw) != "tampered bytes" {
		t.Fatalf("the store repaired the object silently")
	}
	if _, _, err := store.Get(sha256Digest([]byte("absent"))); !errors.Is(err, ErrStoreNotFound) {
		t.Fatalf("unknown digest: %v", err)
	}
}

type stageInput struct {
	stage     int
	plan      string
	artifacts map[string]string
	engines   map[string]string
}

func buildChain(t *testing.T, inputs []stageInput) Chain {
	t.Helper()
	c := NewChain()
	for _, in := range inputs {
		if err := c.Append(in.stage, in.plan, in.artifacts, in.engines, map[string]string{"note": "synthetic"}); err != nil {
			t.Fatal(err)
		}
	}
	return *c
}

func syntheticStages(plan, artifact, engine string) []stageInput {
	return []stageInput{
		{1, plan, nil, nil},
		{2, plan, map[string]string{"first-success": artifact, "second-success": artifact, "stored": artifact}, nil},
		{3, plan, map[string]string{"first-build": artifact, "second-build": artifact, "registered": artifact}, map[string]string{"transform": "t", "static-response": engine}},
		{4, plan, map[string]string{"retrieved": artifact, "invoked": artifact}, map[string]string{"static-response": engine}},
	}
}

func TestVerifyThroughNeedsEveryStageAndOneIdentity(t *testing.T) {
	a, x := sha256Digest([]byte("a")), sha256Digest([]byte("x"))
	if res := VerifyThrough(buildChain(t, syntheticStages("p", a, "e"))); !res.Complete || res.ArtifactDigest != a {
		t.Fatalf("complete chain: %+v", res)
	}
	stages := syntheticStages("p", a, "e")
	missing := VerifyThrough(buildChain(t, append(append([]stageInput(nil), stages[:1]...), stages[2:]...)))
	if missing.Complete || len(missing.MissingStages) != 1 || missing.MissingStages[0] != 2 {
		t.Fatalf("a missing stage must be named: %+v", missing)
	}
	swapped := syntheticStages("p", a, "e")
	swapped[2].artifacts = map[string]string{"first-build": x, "second-build": x, "registered": x}
	res := VerifyThrough(buildChain(t, swapped))
	if res.Complete || len(res.SelfChecksPassed) != 4 || !strings.Contains(strings.Join(res.Mismatches, "\n"), "stage 3") {
		t.Fatalf("a swapped identity must fail only the through check: %+v", res)
	}
	plan := syntheticStages("p", a, "e")
	plan[1].plan = "q"
	if res := VerifyThrough(buildChain(t, plan)); res.Complete || len(res.SelfChecksPassed) != 4 {
		t.Fatalf("a swapped plan fingerprint must fail the through check: %+v", res)
	}
	broken := buildChain(t, syntheticStages("p", a, "e"))
	broken.Records[1].Prev = "other"
	if res := VerifyThrough(broken); res.Complete {
		t.Fatalf("a broken link must fail: %+v", res)
	}
}
