package main

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type proofAuthorityVersionSnapshotProbe struct {
	SemanticFingerprint string
	Locator             string
	ResolvedCount       int
}

type proofAcceptanceSnapshotProbe map[string]proofAuthorityVersionSnapshotProbe

type proofEvaluationStampSnapshotProbe struct {
	ProofID               string
	AuthorityFingerprints map[string]string
}

type proofFreshnessSnapshotProbe struct {
	Fresh                bool
	ChangedAuthorities   []string
	UnresolvedAuthorities []string
}

func TestProofSnapshotInvalidationIsSelectiveByRequiredAuthority(t *testing.T) {
	base := proofAcceptanceSnapshotProbe{
		"business:order-entry":          {SemanticFingerprint: "business-v1", Locator: "business/order-entry", ResolvedCount: 1},
		"application:order-entry":       {SemanticFingerprint: "application-v1", Locator: "applications/order-entry/application.tei", ResolvedCount: 1},
		"semantic:sales.order.submit":   {SemanticFingerprint: "capability-v1", Locator: "semantic/sales/order.tei", ResolvedCount: 1},
		"database:sales_core.order":     {SemanticFingerprint: "database-v1", Locator: "databases/sales-core/contracts/order.tei", ResolvedCount: 1},
		"security:legacy-authorization": {SemanticFingerprint: "security-v1", Locator: "legacy-auth/authorization.json", ResolvedCount: 1},
		"semantic:customer":             {SemanticFingerprint: "customer-v1", Locator: "semantic/customer/customer.tei", ResolvedCount: 1},
	}

	p3, err := stampProofEvaluationSnapshotProbe("P3-07", []string{
		"business:order-entry",
		"application:order-entry",
		"semantic:sales.order.submit",
		"database:sales_core.order",
	}, base)
	if err != nil {
		t.Fatal(err)
	}
	p4, err := stampProofEvaluationSnapshotProbe("P4-06b", []string{
		"business:order-entry",
		"security:legacy-authorization",
	}, base)
	if err != nil {
		t.Fatal(err)
	}

	securityChanged := cloneProofAcceptanceSnapshotProbe(base)
	securityChanged["security:legacy-authorization"] = proofAuthorityVersionSnapshotProbe{
		SemanticFingerprint: "security-v2",
		Locator:             "legacy-auth/authorization.json",
		ResolvedCount:       1,
	}
	assertProofFreshnessSnapshotProbe(t, p3, securityChanged, true, nil, nil)
	assertProofFreshnessSnapshotProbe(t, p4, securityChanged, false, []string{"security:legacy-authorization"}, nil)

	databaseChanged := cloneProofAcceptanceSnapshotProbe(base)
	databaseChanged["database:sales_core.order"] = proofAuthorityVersionSnapshotProbe{
		SemanticFingerprint: "database-v2",
		Locator:             "databases/sales-core/contracts/order.tei",
		ResolvedCount:       1,
	}
	assertProofFreshnessSnapshotProbe(t, p3, databaseChanged, false, []string{"database:sales_core.order"}, nil)
	assertProofFreshnessSnapshotProbe(t, p4, databaseChanged, true, nil, nil)

	unrelatedChanged := cloneProofAcceptanceSnapshotProbe(base)
	unrelatedChanged["semantic:customer"] = proofAuthorityVersionSnapshotProbe{
		SemanticFingerprint: "customer-v2",
		Locator:             "semantic/customer/customer.tei",
		ResolvedCount:       1,
	}
	assertProofFreshnessSnapshotProbe(t, p3, unrelatedChanged, true, nil, nil)
	assertProofFreshnessSnapshotProbe(t, p4, unrelatedChanged, true, nil, nil)
}

func TestProofSnapshotInvalidationIgnoresLocatorOnlyChangeWhenSemanticFingerprintIsStable(t *testing.T) {
	base := proofAcceptanceSnapshotProbe{
		"application:order-entry": {
			SemanticFingerprint: "application-v1",
			Locator:             "applications/order-entry/application.tei",
			ResolvedCount:       1,
		},
	}
	stamp, err := stampProofEvaluationSnapshotProbe("P1-00", []string{"application:order-entry"}, base)
	if err != nil {
		t.Fatal(err)
	}

	moved := cloneProofAcceptanceSnapshotProbe(base)
	moved["application:order-entry"] = proofAuthorityVersionSnapshotProbe{
		SemanticFingerprint: "application-v1",
		Locator:             "applications/order-entry/current/application.tei",
		ResolvedCount:       1,
	}
	assertProofFreshnessSnapshotProbe(t, stamp, moved, true, nil, nil)
}

func TestProofSnapshotInvalidationFailsClosedForMissingOrAmbiguousRequiredAuthority(t *testing.T) {
	base := proofAcceptanceSnapshotProbe{
		"database:sales_core.order": {
			SemanticFingerprint: "database-v1",
			Locator:             "databases/sales-core/contracts/order.tei",
			ResolvedCount:       1,
		},
	}
	stamp, err := stampProofEvaluationSnapshotProbe("P3-07", []string{"database:sales_core.order"}, base)
	if err != nil {
		t.Fatal(err)
	}

	missing := cloneProofAcceptanceSnapshotProbe(base)
	delete(missing, "database:sales_core.order")
	assertProofFreshnessSnapshotProbe(t, stamp, missing, false, nil, []string{"database:sales_core.order"})

	ambiguous := cloneProofAcceptanceSnapshotProbe(base)
	ambiguous["database:sales_core.order"] = proofAuthorityVersionSnapshotProbe{
		SemanticFingerprint: "database-v1",
		Locator:             "databases/sales-core/contracts/order.tei",
		ResolvedCount:       2,
	}
	assertProofFreshnessSnapshotProbe(t, stamp, ambiguous, false, nil, []string{"database:sales_core.order"})
}

func TestStaleContradictionCannotEliminateCandidate(t *testing.T) {
	results := loadOrderEntryProofResultsPacketProbe(t)
	byID := crossViewProofResultsByIDProbe(results)
	if byID["P3-02"].Result != proofContradictionProbe {
		t.Fatalf("test setup: P3-02 is not contradiction: %+v", byID["P3-02"])
	}

	base := proofAcceptanceSnapshotProbe{
		"application:order-entry/master-access": {
			SemanticFingerprint: "master-read-v1",
			Locator:             "applications/order-entry/application.tei",
			ResolvedCount:       1,
		},
	}
	stamp, err := stampProofEvaluationSnapshotProbe("P3-02", []string{"application:order-entry/master-access"}, base)
	if err != nil {
		t.Fatal(err)
	}
	stamps := map[string]proofEvaluationStampSnapshotProbe{"P3-02": stamp}

	freshResults, staleIDs, err := currentProofResultsForCandidateSnapshotProbe(results, []string{"P3-02"}, stamps, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(staleIDs) != 0 {
		t.Fatalf("unchanged contradiction was marked stale: %v", staleIDs)
	}
	freshPacket, err := buildCandidateLocalProofPacketProbe(freshResults, candidateProofScopePacketProbe{
		CandidateID:      "customer-master-mutation",
		RequiredProofIDs: []string{"P3-02"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if freshPacket.Elimination == nil {
		t.Fatalf("fresh contradiction did not eliminate candidate: %+v", freshPacket)
	}

	changed := cloneProofAcceptanceSnapshotProbe(base)
	changed["application:order-entry/master-access"] = proofAuthorityVersionSnapshotProbe{
		SemanticFingerprint: "master-access-v2",
		Locator:             "applications/order-entry/application.tei",
		ResolvedCount:       1,
	}
	currentResults, staleIDs, err := currentProofResultsForCandidateSnapshotProbe(results, []string{"P3-02"}, stamps, changed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(staleIDs, []string{"P3-02"}) {
		t.Fatalf("changed contradiction was not marked stale: %v", staleIDs)
	}
	if len(currentResults) != 0 {
		t.Fatalf("stale contradiction leaked into current proof set: %+v", currentResults)
	}

	stalePacket, err := buildCandidateLocalProofPacketProbe(currentResults, candidateProofScopePacketProbe{
		CandidateID:      "customer-master-mutation",
		RequiredProofIDs: []string{"P3-02"},
	})
	if err == nil {
		t.Fatalf("stale contradiction still produced a candidate packet: %+v", stalePacket)
	}
	if !strings.Contains(err.Error(), "required proof") {
		t.Fatalf("stale proof did not fail closed as missing current proof: %v", err)
	}
}

func TestProofEvaluationStampIsDerivedFreshnessMetadataNotAuthority(t *testing.T) {
	stampType := reflect.TypeOf(proofEvaluationStampSnapshotProbe{})
	want := []string{"ProofID", "AuthorityFingerprints"}
	if got := fieldNames(stampType); !reflect.DeepEqual(got, want) {
		t.Fatalf("proof stamp shape changed: got=%v want=%v", got, want)
	}
	for _, forbidden := range []string{
		"Accepted", "Canonical", "Authoritative", "Confidence", "Score", "Complete", "WriteBack", "SourceOfTruth",
	} {
		if _, found := stampType.FieldByName(forbidden); found {
			t.Fatalf("proof freshness stamp gained authority field %q", forbidden)
		}
	}
}

func stampProofEvaluationSnapshotProbe(proofID string, requiredAuthorities []string, snapshot proofAcceptanceSnapshotProbe) (proofEvaluationStampSnapshotProbe, error) {
	if strings.TrimSpace(proofID) == "" {
		return proofEvaluationStampSnapshotProbe{}, fmt.Errorf("proof ID is required")
	}
	if len(requiredAuthorities) == 0 {
		return proofEvaluationStampSnapshotProbe{}, fmt.Errorf("proof %s has no declared authority dependency", proofID)
	}

	fingerprints := make(map[string]string, len(requiredAuthorities))
	seen := map[string]struct{}{}
	for _, authorityKey := range requiredAuthorities {
		authorityKey = strings.TrimSpace(authorityKey)
		if authorityKey == "" {
			return proofEvaluationStampSnapshotProbe{}, fmt.Errorf("proof %s has blank authority dependency", proofID)
		}
		if _, duplicate := seen[authorityKey]; duplicate {
			return proofEvaluationStampSnapshotProbe{}, fmt.Errorf("proof %s duplicates authority dependency %s", proofID, authorityKey)
		}
		seen[authorityKey] = struct{}{}

		authority, ok := snapshot[authorityKey]
		if !ok || authority.ResolvedCount != 1 || strings.TrimSpace(authority.SemanticFingerprint) == "" {
			return proofEvaluationStampSnapshotProbe{}, fmt.Errorf("proof %s authority %s is not exact-one fingerprinted", proofID, authorityKey)
		}
		fingerprints[authorityKey] = authority.SemanticFingerprint
	}
	return proofEvaluationStampSnapshotProbe{ProofID: proofID, AuthorityFingerprints: fingerprints}, nil
}

func evaluateProofFreshnessSnapshotProbe(stamp proofEvaluationStampSnapshotProbe, current proofAcceptanceSnapshotProbe) proofFreshnessSnapshotProbe {
	changed := make([]string, 0)
	unresolved := make([]string, 0)
	for authorityKey, evaluatedFingerprint := range stamp.AuthorityFingerprints {
		authority, ok := current[authorityKey]
		if !ok || authority.ResolvedCount != 1 || strings.TrimSpace(authority.SemanticFingerprint) == "" {
			unresolved = append(unresolved, authorityKey)
			continue
		}
		if authority.SemanticFingerprint != evaluatedFingerprint {
			changed = append(changed, authorityKey)
		}
	}
	sort.Strings(changed)
	sort.Strings(unresolved)
	return proofFreshnessSnapshotProbe{
		Fresh:                 len(changed) == 0 && len(unresolved) == 0,
		ChangedAuthorities:    changed,
		UnresolvedAuthorities: unresolved,
	}
}

func currentProofResultsForCandidateSnapshotProbe(
	results []crossViewProofResultProbe,
	requiredProofIDs []string,
	stamps map[string]proofEvaluationStampSnapshotProbe,
	current proofAcceptanceSnapshotProbe,
) ([]crossViewProofResultProbe, []string, error) {
	byID := crossViewProofResultsByIDProbe(results)
	fresh := make([]crossViewProofResultProbe, 0, len(requiredProofIDs))
	stale := make([]string, 0)
	seen := map[string]struct{}{}

	for _, proofID := range requiredProofIDs {
		if _, duplicate := seen[proofID]; duplicate {
			return nil, nil, fmt.Errorf("required proof %s is duplicated", proofID)
		}
		seen[proofID] = struct{}{}

		result, ok := byID[proofID]
		if !ok {
			return nil, nil, fmt.Errorf("required proof result %s is missing", proofID)
		}
		stamp, ok := stamps[proofID]
		if !ok {
			stale = append(stale, proofID)
			continue
		}
		if stamp.ProofID != proofID {
			return nil, nil, fmt.Errorf("proof stamp %s is bound to %s", proofID, stamp.ProofID)
		}
		freshness := evaluateProofFreshnessSnapshotProbe(stamp, current)
		if !freshness.Fresh {
			stale = append(stale, proofID)
			continue
		}
		fresh = append(fresh, result)
	}
	sort.Slice(fresh, func(i, j int) bool { return fresh[i].ID < fresh[j].ID })
	sort.Strings(stale)
	return fresh, stale, nil
}

func cloneProofAcceptanceSnapshotProbe(source proofAcceptanceSnapshotProbe) proofAcceptanceSnapshotProbe {
	clone := make(proofAcceptanceSnapshotProbe, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func assertProofFreshnessSnapshotProbe(
	t *testing.T,
	stamp proofEvaluationStampSnapshotProbe,
	current proofAcceptanceSnapshotProbe,
	wantFresh bool,
	wantChanged []string,
	wantUnresolved []string,
) {
	t.Helper()
	got := evaluateProofFreshnessSnapshotProbe(stamp, current)
	if got.Fresh != wantFresh {
		t.Fatalf("fresh=%v want=%v: %+v", got.Fresh, wantFresh, got)
	}
	if !reflect.DeepEqual(got.ChangedAuthorities, wantChanged) {
		t.Fatalf("changed authorities=%v want=%v", got.ChangedAuthorities, wantChanged)
	}
	if !reflect.DeepEqual(got.UnresolvedAuthorities, wantUnresolved) {
		t.Fatalf("unresolved authorities=%v want=%v", got.UnresolvedAuthorities, wantUnresolved)
	}
}
