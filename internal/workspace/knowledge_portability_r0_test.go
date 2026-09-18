package workspace_test

import (
	"reflect"
	"sort"
	"testing"

	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

type knowledgePortabilityR0DestinationProbe struct {
	Label              string
	CustomerSemanticID string
	DatabaseTarget     string
	ContractID         string
	ContractMajor      int
	Access             database.Access
	DatasourceAlias    string
}

type knowledgePortabilityR0CandidateProbe struct {
	ObservedIn          string
	ObservedFrom        string
	AuthoritySubject    string
	ReuseRelation       string
	Qualification       string
	Reasons             []string
	DestinationSubject  string
	DestinationAlias    string
	DependencySubjects  []string
	ProvenanceSubjects  []string
	AuthoritySource     string
	AuthorityFingerprint string
	ProvenanceSource    string
	ProvenanceFingerprint string
	AuthorityLimits     []string
}

type knowledgePortabilityR0EnvironmentProbe struct {
	Environment       string
	OriginApplication string
	OriginAlias       string
	Instance          string
	Database          string
	Provider          string
	ReuseRelation     string
	Qualification     string
	DestinationBinding string
	AuthorityLimits   []string
}

func TestKnowledgePortabilityR0SeparatesDiscoveryProvenanceFromSemanticAuthority(t *testing.T) {
	root := repoRoot(t)
	ws := root + "/reference/workspace"
	loaded, err := workspace.Load(ws)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := workspace.BuildSourceIndex(ws, loaded)
	if err != nil {
		t.Fatal(err)
	}

	destination := knowledgePortabilityR0DestinationProbe{
		Label:              "repair_intake-test-only",
		CustomerSemanticID: "customer.customer",
		DatabaseTarget:     "master_data",
		ContractID:         "master_data.customer",
		ContractMajor:      2,
		Access:             database.AccessRead,
		DatasourceAlias:    "customer_master",
	}
	got, err := deriveKnowledgePortabilityR0SemanticCandidate(idx, destination)
	if err != nil {
		t.Fatal(err)
	}

	if got.ObservedFrom != "application-pattern:order_entry.customer_reference" {
		t.Fatalf("discovery provenance changed: %+v", got)
	}
	if got.AuthoritySubject != "semantic-data:customer.customer" {
		t.Fatalf("order_entry was incorrectly treated as owner of customer meaning: %+v", got)
	}
	if got.ReuseRelation != "linked-reuse" || got.Qualification != "applicable" {
		t.Fatalf("shared Customer Semantic was not kept as linked reuse candidate: %+v", got)
	}
	if !containsPortabilityR0(got.DependencySubjects, "semantic-data:customer.customer") {
		t.Fatalf("destination dependency does not point at shared Semantic authority: %+v", got.DependencySubjects)
	}
	if containsPortabilityR0(got.DependencySubjects, "application-pattern:order_entry.customer_reference") {
		t.Fatalf("discovery provenance leaked into dependency authority: %+v", got.DependencySubjects)
	}
	if !containsPortabilityR0(got.ProvenanceSubjects, "application-pattern:order_entry.customer_reference") {
		t.Fatalf("origin Pattern provenance was lost: %+v", got.ProvenanceSubjects)
	}
	if got.AuthoritySource == "" || got.AuthorityFingerprint == "" || got.ProvenanceSource == "" || got.ProvenanceFingerprint == "" {
		t.Fatalf("source/fingerprint provenance is incomplete: %+v", got)
	}
}

func TestKnowledgePortabilityR0ContractReuseRequiresDestinationChoice(t *testing.T) {
	root := repoRoot(t)
	ws := root + "/reference/workspace"
	loaded, err := workspace.Load(ws)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := workspace.BuildSourceIndex(ws, loaded)
	if err != nil {
		t.Fatal(err)
	}

	applicable := knowledgePortabilityR0DestinationProbe{
		Label:              "repair_intake-test-only",
		CustomerSemanticID: "customer.customer",
		DatabaseTarget:     "master_data",
		ContractID:         "master_data.customer",
		ContractMajor:      2,
		Access:             database.AccessRead,
		DatasourceAlias:    "customer_master",
	}
	got, err := deriveKnowledgePortabilityR0ContractCandidate(loaded.World, idx, applicable)
	if err != nil {
		t.Fatal(err)
	}
	if got.ReuseRelation != "conditional-linked-reuse" || got.Qualification != "applicable" {
		t.Fatalf("qualified destination Contract reuse was not accepted as conditional linked reuse: %+v", got)
	}
	if got.DestinationAlias != "customer_master" {
		t.Fatalf("destination alias was not kept destination-owned: %+v", got)
	}
	if got.DestinationAlias == "master" {
		t.Fatalf("origin datasource alias was copied into destination: %+v", got)
	}
	if !containsPortabilityR0(got.DependencySubjects, "database-contract:master_data.customer") {
		t.Fatalf("Contract linked dependency is missing: %+v", got.DependencySubjects)
	}

	unknown := applicable
	unknown.DatabaseTarget = ""
	unknown.ContractID = ""
	unknown.ContractMajor = 0
	unknown.DatasourceAlias = ""
	gotUnknown, err := deriveKnowledgePortabilityR0ContractCandidate(loaded.World, idx, unknown)
	if err != nil {
		t.Fatal(err)
	}
	if gotUnknown.Qualification != "unknown" {
		t.Fatalf("missing destination DB/Contract choice was silently inherited from origin: %+v", gotUnknown)
	}
}

func TestKnowledgePortabilityR0ApplicationPatternBecomesDestinationOwnedVariation(t *testing.T) {
	root := repoRoot(t)
	ws := root + "/reference/workspace"
	loaded, err := workspace.Load(ws)
	if err != nil {
		t.Fatal(err)
	}
	idx, err := workspace.BuildSourceIndex(ws, loaded)
	if err != nil {
		t.Fatal(err)
	}

	got, err := deriveKnowledgePortabilityR0PatternCandidate(idx, "repair_intake.customer_reference")
	if err != nil {
		t.Fatal(err)
	}
	if got.ReuseRelation != "adopted-local-variation" {
		t.Fatalf("Application-owned Pattern was treated as cross-Application shared authority: %+v", got)
	}
	if got.AuthoritySubject != "application-pattern:order_entry.customer_reference" {
		t.Fatalf("origin Pattern authority was lost: %+v", got)
	}
	if got.DestinationSubject != "application-pattern:repair_intake.customer_reference" {
		t.Fatalf("destination Pattern was not given destination-owned identity: %+v", got)
	}
	if containsPortabilityR0(got.DependencySubjects, "application-pattern:order_entry.customer_reference") {
		t.Fatalf("local variation accidentally became linked dependency on origin Pattern: %+v", got.DependencySubjects)
	}
	if !containsPortabilityR0(got.ProvenanceSubjects, "application-pattern:order_entry.customer_reference") {
		t.Fatalf("origin Pattern provenance was not retained: %+v", got.ProvenanceSubjects)
	}
	for _, forbidden := range []string{"semantic-relation:sales.order.customer", "datasource-alias:master"} {
		if containsPortabilityR0(got.DependencySubjects, forbidden) {
			t.Fatalf("origin Application-specific detail %q was silently copied into destination: %+v", forbidden, got)
		}
	}
}

func TestKnowledgePortabilityR0EnvironmentBindingRequiresDestinationRebind(t *testing.T) {
	root := repoRoot(t)
	ws := root + "/reference/workspace"
	loaded, err := workspace.Load(ws)
	if err != nil {
		t.Fatal(err)
	}

	got, err := deriveKnowledgePortabilityR0EnvironmentProbe(loaded.World, "development")
	if err != nil {
		t.Fatal(err)
	}
	if got.OriginApplication != "order_entry" || got.OriginAlias != "master" || got.Instance != "master_data.dev" {
		t.Fatalf("current origin Environment binding was not resolved exactly: %+v", got)
	}
	if got.Database != "master_data" || got.Provider != "oracle" {
		t.Fatalf("provider/database Evidence was lost: %+v", got)
	}
	if got.ReuseRelation != "rebind-required" || got.Qualification != "destination-owned" {
		t.Fatalf("Environment binding was treated as portable shared authority: %+v", got)
	}
	if got.DestinationBinding != "" {
		t.Fatalf("destination Environment binding was auto-created from origin: %+v", got)
	}
}

func TestKnowledgePortabilityR0SeparatesDependencyStalenessFromProvenanceDrift(t *testing.T) {
	linked := knowledgePortabilityR0CandidateProbe{
		DependencySubjects: []string{"semantic-data:customer.customer", "database-contract:master_data.customer"},
		ProvenanceSubjects: []string{"application-pattern:order_entry.customer_reference"},
	}
	localVariation := knowledgePortabilityR0CandidateProbe{
		DependencySubjects: nil,
		ProvenanceSubjects: []string{"application-pattern:order_entry.customer_reference"},
	}

	if got := knowledgePortabilityR0ChangeEffect(linked, "semantic-data:customer.customer"); got != "dependency-requalification" {
		t.Fatalf("linked dependency change effect=%q", got)
	}
	if got := knowledgePortabilityR0ChangeEffect(localVariation, "application-pattern:order_entry.customer_reference"); got != "provenance-drift" {
		t.Fatalf("origin Pattern change should be provenance drift for local variation, got=%q", got)
	}
	if got := knowledgePortabilityR0ChangeEffect(linked, "application-pattern:order_entry.order_record"); got != "unaffected" {
		t.Fatalf("unrelated origin change stale-all'ed portability candidate: got=%q", got)
	}
}

func TestKnowledgePortabilityR0DoesNotCollapsePortabilityToAuthorityShortcut(t *testing.T) {
	typ := reflect.TypeOf(knowledgePortabilityR0CandidateProbe{})
	for _, forbidden := range []string{
		"Portable",
		"AutoApply",
		"AutoSync",
		"Inherited",
		"OverrideWins",
		"OriginOwnsDestination",
		"DestinationUpdatesOrigin",
		"CopyAllDependencies",
		"SameNameMeansSameMeaning",
		"PreferredCandidate",
		"Winner",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("Knowledge Portability R0 introduced unsupported shortcut field %q", forbidden)
		}
	}
}

func deriveKnowledgePortabilityR0SemanticCandidate(
	idx sourceindex.Index,
	destination knowledgePortabilityR0DestinationProbe,
) (knowledgePortabilityR0CandidateProbe, error) {
	authority := sourceindex.Subject{Kind: "semantic-data", ID: "customer.customer"}
	provenance := sourceindex.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"}
	authoritySource, authorityFingerprint, err := knowledgePortabilityR0Source(idx, authority)
	if err != nil {
		return knowledgePortabilityR0CandidateProbe{}, err
	}
	provenanceSource, provenanceFingerprint, err := knowledgePortabilityR0Source(idx, provenance)
	if err != nil {
		return knowledgePortabilityR0CandidateProbe{}, err
	}
	qualification := "unknown"
	if destination.CustomerSemanticID == authority.ID {
		qualification = "applicable"
	}
	return knowledgePortabilityR0CandidateProbe{
		ObservedIn:            "order_entry",
		ObservedFrom:          portabilityR0SubjectKey(provenance),
		AuthoritySubject:      portabilityR0SubjectKey(authority),
		ReuseRelation:         "linked-reuse",
		Qualification:         qualification,
		Reasons:               []string{"destination explicitly selects the same Customer Semantic identity"},
		DependencySubjects:    []string{portabilityR0SubjectKey(authority)},
		ProvenanceSubjects:    []string{portabilityR0SubjectKey(provenance)},
		AuthoritySource:       authoritySource,
		AuthorityFingerprint:  authorityFingerprint,
		ProvenanceSource:      provenanceSource,
		ProvenanceFingerprint: provenanceFingerprint,
		AuthorityLimits: []string{
			"discovery-provenance-is-not-authority-owner",
			"same-name-without-qualified-identity-is-insufficient",
		},
	}, nil
}

func deriveKnowledgePortabilityR0ContractCandidate(
	world ir.World,
	idx sourceindex.Index,
	destination knowledgePortabilityR0DestinationProbe,
) (knowledgePortabilityR0CandidateProbe, error) {
	authority := sourceindex.Subject{Kind: "database-contract", ID: "master_data.customer"}
	authoritySource, authorityFingerprint, err := knowledgePortabilityR0Source(idx, authority)
	if err != nil {
		return knowledgePortabilityR0CandidateProbe{}, err
	}
	contract, ok := knowledgePortabilityR0FindContract(world, "master_data", "master_data.customer")
	if !ok {
		return knowledgePortabilityR0CandidateProbe{}, errPortabilityR0("master_data.customer Contract did not resolve exactly once")
	}

	qualification := "unknown"
	reasons := []string{"destination Database / Contract choice is unresolved"}
	if destination.DatabaseTarget != "" || destination.ContractID != "" || destination.ContractMajor != 0 {
		switch {
		case destination.DatabaseTarget != "master_data":
			qualification = "incompatible"
			reasons = []string{"destination selected a different Database target"}
		case destination.ContractID != "master_data.customer":
			qualification = "incompatible"
			reasons = []string{"destination selected a different Contract"}
		case destination.ContractMajor != contract.Version.Major:
			qualification = "incompatible"
			reasons = []string{"destination Contract major is incompatible"}
		case destination.Access != database.AccessRead && destination.Access != database.AccessReadWrite:
			qualification = "incompatible"
			reasons = []string{"destination access does not permit the required read use"}
		default:
			qualification = "applicable"
			reasons = []string{"destination explicitly selected master_data.customer compatible major and read access"}
		}
	}

	return knowledgePortabilityR0CandidateProbe{
		ObservedIn:           "order_entry",
		ObservedFrom:         "application-pattern:order_entry.customer_reference",
		AuthoritySubject:     portabilityR0SubjectKey(authority),
		ReuseRelation:        "conditional-linked-reuse",
		Qualification:        qualification,
		Reasons:              reasons,
		DestinationAlias:     destination.DatasourceAlias,
		DependencySubjects:   []string{portabilityR0SubjectKey(authority)},
		ProvenanceSubjects:   []string{"application-pattern:order_entry.customer_reference"},
		AuthoritySource:      authoritySource,
		AuthorityFingerprint: authorityFingerprint,
		AuthorityLimits: []string{
			"origin-datasource-alias-is-not-inherited",
			"origin-environment-binding-is-not-inherited",
			"contract-applicability-requires-destination-choice",
		},
	}, nil
}

func deriveKnowledgePortabilityR0PatternCandidate(
	idx sourceindex.Index,
	destinationPatternID string,
) (knowledgePortabilityR0CandidateProbe, error) {
	origin := sourceindex.Subject{Kind: "application-pattern", ID: "order_entry.customer_reference"}
	source, fingerprint, err := knowledgePortabilityR0Source(idx, origin)
	if err != nil {
		return knowledgePortabilityR0CandidateProbe{}, err
	}
	return knowledgePortabilityR0CandidateProbe{
		ObservedIn:             "order_entry",
		ObservedFrom:           portabilityR0SubjectKey(origin),
		AuthoritySubject:       portabilityR0SubjectKey(origin),
		ReuseRelation:          "adopted-local-variation",
		Qualification:          "applicable-as-provenance",
		DestinationSubject:     "application-pattern:" + destinationPatternID,
		DependencySubjects:     nil,
		ProvenanceSubjects:     []string{portabilityR0SubjectKey(origin)},
		AuthoritySource:        source,
		AuthorityFingerprint:   fingerprint,
		ProvenanceSource:       source,
		ProvenanceFingerprint:  fingerprint,
		Reasons: []string{
			"Application Pattern is Application-owned",
			"destination must author and accept its own Pattern identity",
		},
		AuthorityLimits: []string{
			"origin-pattern-is-provenance-not-destination-authority",
			"origin-relation-and-datasource-are-not-auto-copied",
			"origin-change-does-not-auto-patch-destination",
		},
	}, nil
}

func deriveKnowledgePortabilityR0EnvironmentProbe(world ir.World, environmentID string) (knowledgePortabilityR0EnvironmentProbe, error) {
	for _, env := range world.Environments {
		if string(env.ID) != environmentID {
			continue
		}
		instanceByID := map[string]struct {
			database string
			provider string
		}{}
		for _, instance := range env.Instances {
			instanceByID[string(instance.ID)] = struct {
				database string
				provider string
			}{database: string(instance.Definition), provider: string(instance.Provider)}
		}
		for _, binding := range env.Bindings {
			if binding.Application != "order_entry" || binding.Alias != "master" {
				continue
			}
			instance, ok := instanceByID[string(binding.Instance)]
			if !ok {
				return knowledgePortabilityR0EnvironmentProbe{}, errPortabilityR0("origin binding points to missing instance")
			}
			return knowledgePortabilityR0EnvironmentProbe{
				Environment:       environmentID,
				OriginApplication: string(binding.Application),
				OriginAlias:       string(binding.Alias),
				Instance:          string(binding.Instance),
				Database:          instance.database,
				Provider:          instance.provider,
				ReuseRelation:     "rebind-required",
				Qualification:     "destination-owned",
				DestinationBinding: "",
				AuthorityLimits: []string{
					"application-and-alias-are-origin-local",
					"environment-instance-is-not-auto-copied",
					"provider-evidence-does-not-author-destination-binding",
				},
			}, nil
		}
		return knowledgePortabilityR0EnvironmentProbe{}, errPortabilityR0("order_entry.master binding not found")
	}
	return knowledgePortabilityR0EnvironmentProbe{}, errPortabilityR0("environment not found")
}

func knowledgePortabilityR0FindContract(world ir.World, databaseID, contractID string) (database.Contract, bool) {
	var found database.Contract
	count := 0
	for _, db := range world.Databases {
		if string(db.ID) != databaseID {
			continue
		}
		for _, contract := range db.Contracts {
			if string(contract.ID) == contractID {
				found = contract
				count++
			}
		}
	}
	return found, count == 1
}

func knowledgePortabilityR0Source(idx sourceindex.Index, subject sourceindex.Subject) (string, string, error) {
	refs := idx.ReferencesFor(subject)
	if len(refs) != 1 {
		return "", "", errPortabilityR0("subject " + portabilityR0SubjectKey(subject) + " does not have exactly one Source Index reference")
	}
	for _, artifact := range idx.Artifacts {
		if artifact.Path == refs[0].ArtifactPath {
			return artifact.Path, artifact.Fingerprint, nil
		}
	}
	return "", "", errPortabilityR0("Source Index artifact not found for " + portabilityR0SubjectKey(subject))
}

func knowledgePortabilityR0ChangeEffect(candidate knowledgePortabilityR0CandidateProbe, changedSubject string) string {
	if containsPortabilityR0(candidate.DependencySubjects, changedSubject) {
		return "dependency-requalification"
	}
	if containsPortabilityR0(candidate.ProvenanceSubjects, changedSubject) {
		return "provenance-drift"
	}
	return "unaffected"
}

func portabilityR0SubjectKey(subject sourceindex.Subject) string {
	return subject.Kind + ":" + subject.ID
}

func containsPortabilityR0(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

type portabilityR0Error string

func (e portabilityR0Error) Error() string { return string(e) }

func errPortabilityR0(message string) error { return portabilityR0Error(message) }

func normalizePortabilityR0(values []string) []string {
	out := append([]string(nil), values...)
	sort.Strings(out)
	return out
}
