package main

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
	authoritytrust "tei/tools/reference-authority-evidence-trust/trust"
)

func serviceSelectionPluginIdentity(t *testing.T, provider Provider) plugin.Identity {
	t.Helper()
	return plugin.Identity{
		ID:      core.MustCanonicalID(provider.PluginID),
		Version: provider.PluginVersion,
	}
}

type testHostCurrentAcceptanceSource struct {
	snapshot HostCurrentAcceptanceSnapshot
	reads    int
	err      error
}

func (s *testHostCurrentAcceptanceSource) ReadCurrentHostAcceptance(
	_ context.Context, _ pruntime.ServiceProviderSelection,
) (HostCurrentAcceptanceSnapshot, error) {
	s.reads++
	if s.err != nil {
		return HostCurrentAcceptanceSnapshot{}, s.err
	}
	// A Host-owned source returns an isolated observation, not mutable
	// aliases of its authoritative grant collection.
	snapshot := s.snapshot
	snapshot.CurrentTrustGrants = append([]authoritytrust.Grant(nil), s.snapshot.CurrentTrustGrants...)
	return snapshot, nil
}

func testHostAcceptanceSnapshot(
	t *testing.T,
	contract Contract,
	provider Provider,
	selected pruntime.ServiceProviderSelection,
) *testHostCurrentAcceptanceSource {
	t.Helper()
	publication := authoritytrust.Statement{
		AuthorityClass:       serviceContractPublicationAuthorityClass,
		Scope:                "contract:" + selected.ContractID,
		Issuer:               authoritytrust.Issuer{ID: "reference.host.contract-publisher", Version: "1.0.0"},
		StatementFingerprint: selected.ContractFingerprint,
	}
	evidence := evidenceFor(provider, contract, quoteLegacyFingerprint("quote-conformance-evidence"))
	conformance := authoritytrust.Statement{
		AuthorityClass:       serviceProviderConformanceAuthorityClass,
		Scope:                "contract:" + selected.ContractID + "/provider:" + provider.PluginID + "@" + provider.PluginVersion,
		Issuer:               authoritytrust.Issuer{ID: "reference.host.provider-conformance-issuer", Version: "1.0.0"},
		StatementFingerprint: fingerprint(evidence),
	}
	publicationGrant, err := authoritytrust.NewGrant(
		publication.AuthorityClass, publication.Scope, publication.Issuer,
		quoteLegacyFingerprint("host-contract-trust-revision"), true,
	)
	if err != nil {
		t.Fatal(err)
	}
	conformanceGrant, err := authoritytrust.NewGrant(
		conformance.AuthorityClass, conformance.Scope, conformance.Issuer,
		quoteLegacyFingerprint("host-provider-trust-revision"), true,
	)
	if err != nil {
		t.Fatal(err)
	}
	return &testHostCurrentAcceptanceSource{
		snapshot: HostCurrentAcceptanceSnapshot{
			Contract:               contract,
			Provider:               provider,
			ProviderEvidence:       evidence,
			ContractPublication:    publication,
			ProviderConformance:    conformance,
			CurrentTrustGrants:     []authoritytrust.Grant{publicationGrant, conformanceGrant},
			ContractAccepted:       true,
			ProviderAccepted:       true,
			ContractAcceptanceRef:  "synthetic-quote-contract-acceptance",
			ProviderConformanceRef: "synthetic-quote-provider-conformance",
			AuthorityRevision:      "synthetic-test-current-v1",
		},
	}
}

func TestReferenceHostAcceptedSnapshotReusesCurrentPublisherAndConformanceTrust(t *testing.T) {
	calls := 0
	contract, provider, _ := quoteCandidateRegistryFixture(t, &calls)
	selected := pruntime.ServiceProviderSelection{
		ContractID:          contract.ID,
		ContractVersion:     contract.Version,
		ContractFingerprint: fingerprint(contract),
		Scope:               string(provider.Scope),
		Provider:            serviceSelectionPluginIdentity(t, provider),
		ImplementationID:    provider.ImplementationID,
		ArtifactDigest:      provider.ArtifactDigest,
	}
	source := testHostAcceptanceSnapshot(t, contract, provider, selected)
	reader := &ReferenceHostCurrentAcceptanceReader{Source: source}

	accepted, err := reader.ReadCurrentAcceptedService(context.Background(), selected)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Selection != selected ||
		accepted.ContractAcceptanceRef != source.snapshot.ContractAcceptanceRef ||
		accepted.ProviderConformanceRef != source.snapshot.ProviderConformanceRef ||
		accepted.AuthorityRevision != source.snapshot.AuthorityRevision ||
		source.reads != 1 {
		t.Fatalf("Host read did not preserve exact accepted origin: %+v reads=%d", accepted, source.reads)
	}

	// A previously accepted selection and unchanged Runtime Plan must not
	// cache an issuer or Provider approval after the Host revokes either.
	for _, n := range []int{0, 1} {
		grant := source.snapshot.CurrentTrustGrants[n]
		revoked, err := authoritytrust.NewGrant(
			grant.AuthorityClass, grant.Scope, grant.Issuer,
			grant.PolicyFingerprint, false,
		)
		if err != nil {
			t.Fatal(err)
		}
		source.snapshot.CurrentTrustGrants[n] = revoked
		before := source.reads
		_, err = reader.ReadCurrentAcceptedService(context.Background(), selected)
		if err == nil || !strings.Contains(err.Error(), "trust denied") {
			t.Fatalf("current issuer revocation %d was not rejected: %v", n, err)
		}
		if source.reads != before+1 {
			t.Fatalf("revocation reused a stale Host decision: reads=%d", source.reads)
		}
		source.snapshot.CurrentTrustGrants[n] = grant
	}
}

func TestReferenceHostAcceptanceRejectsDriftedOrSelfAssertedAuthority(t *testing.T) {
	calls := 0
	contract, provider, _ := quoteCandidateRegistryFixture(t, &calls)
	selected := pruntime.ServiceProviderSelection{
		ContractID:          contract.ID,
		ContractVersion:     contract.Version,
		ContractFingerprint: fingerprint(contract),
		Scope:               string(provider.Scope),
		Provider:            serviceSelectionPluginIdentity(t, provider),
		ImplementationID:    provider.ImplementationID,
		ArtifactDigest:      provider.ArtifactDigest,
	}
	for _, tc := range []struct {
		name   string
		mutate func(*HostCurrentAcceptanceSnapshot)
		want   string
	}{
		{
			name: "inactive Contract",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.ContractAccepted = false
			},
			want: "acceptance is inactive",
		},
		{
			name: "inactive Provider",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.ProviderAccepted = false
			},
			want: "acceptance is inactive",
		},
		{
			name: "changed Contract failure semantics",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.Contract.FailureSemantics += "-changed"
			},
			want: "differs from selected exact Contract",
		},
		{
			name: "different Contract version",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.Contract.Version = "1.1.0"
			},
			want: "differs from selected exact Contract",
		},
		{
			name: "different Provider artifact",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.Provider.ArtifactDigest = quoteLegacyFingerprint("other-image")
			},
			want: "differs from selected exact implementation",
		},
		{
			name: "different Provider implementation",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.Provider.ImplementationID = "other-quote-implementation"
			},
			want: "differs from selected exact implementation",
		},
		{
			name: "Provider conformance evidence from other artifact",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.ProviderEvidence.ArtifactDigest = quoteLegacyFingerprint("unrelated")
			},
			want: "conformance evidence differs",
		},
		{
			name: "Provider conformance evidence substituted after issuer statement",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.ProviderEvidence.EvidenceFingerprint = "forged-evidence"
			},
			want: "conformance statement has wrong authority or evidence",
		},
		{
			name: "publisher tries to claim wrong namespace",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.ContractPublication.Scope = "contract:std.unrelated"
			},
			want: "publisher statement has wrong authority",
		},
		{
			name: "Provider issuer cannot act as Contract publisher",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.ContractPublication.AuthorityClass = serviceProviderConformanceAuthorityClass
			},
			want: "publisher statement has wrong authority",
		},
		{
			name: "Provider issuer tries another plugin",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.ProviderConformance.Scope = "contract:" + selected.ContractID + "/provider:other.plugin@1.0.0"
			},
			want: "conformance statement has wrong authority",
		},
		{
			name: "Contract publisher grant missing",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.CurrentTrustGrants = s.CurrentTrustGrants[1:]
			},
			want: "current Contract publisher trust denied",
		},
		{
			name: "Provider issuer grant missing",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.CurrentTrustGrants = s.CurrentTrustGrants[:1]
			},
			want: "current Provider conformance issuer trust denied",
		},
		{
			name: "ambiguous Contract grants",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.CurrentTrustGrants = append(s.CurrentTrustGrants, s.CurrentTrustGrants[0])
			},
			want: "current Contract publisher trust denied",
		},
		{
			name: "Host acceptance record missing",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.ContractAcceptanceRef = ""
			},
			want: "acceptance references or authority revision are missing",
		},
		{
			name: "Host issuer revision missing",
			mutate: func(s *HostCurrentAcceptanceSnapshot) {
				s.AuthorityRevision = ""
			},
			want: "acceptance references or authority revision are missing",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			source := testHostAcceptanceSnapshot(t, contract, provider, selected)
			tc.mutate(&source.snapshot)
			reader := &ReferenceHostCurrentAcceptanceReader{Source: source}
			_, err := reader.ReadCurrentAcceptedService(context.Background(), selected)
			if err == nil || !strings.Contains(err.Error(), tc.want) || source.reads != 1 {
				t.Fatalf("unaccepted authority was admitted: err=%v reads=%d", err, source.reads)
			}
		})
	}
}

func TestReferenceHostAcceptanceRejectsMissingOrBrokenSource(t *testing.T) {
	var absent *ReferenceHostCurrentAcceptanceReader
	_, err := absent.ReadCurrentAcceptedService(context.Background(), pruntime.ServiceProviderSelection{})
	if err == nil || !strings.Contains(err.Error(), "source is missing") {
		t.Fatalf("missing reader source not rejected: %v", err)
	}
	reader := &ReferenceHostCurrentAcceptanceReader{Source: &testHostCurrentAcceptanceSource{
		err: fmt.Errorf("Host records unavailable"),
	}}
	_, err = reader.ReadCurrentAcceptedService(context.Background(), pruntime.ServiceProviderSelection{})
	if err == nil || !strings.Contains(err.Error(), "Host current acceptance source denied") {
		t.Fatalf("unavailable Host source not rejected: %v", err)
	}
}
