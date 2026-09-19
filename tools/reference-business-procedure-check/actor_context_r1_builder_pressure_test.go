package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

const (
	actorContextR1ApplicationSubject = "order_entry"
	actorContextR1PatternSubject     = "order_entry.customer_reference"
	actorContextR1ContractSubject    = "master_data.customer"
)

type actorContextR1ConstraintProbe struct {
	ChangeRequest             string                           `json:"changeRequest"`
	ReferenceFingerprint      string                           `json:"referenceFingerprint"`
	RequiredInputFingerprints []actorContextR0InputFingerprint `json:"requiredInputFingerprints"`
	BusinessStep              string                           `json:"businessStep"`
	BusinessAction            string                           `json:"businessAction"`
	BusinessRealization       string                           `json:"businessRealization"`
	Pattern                   string                           `json:"pattern"`
	PatternKind               string                           `json:"patternKind"`
	Relation                  string                           `json:"relation"`
	Datasource                string                           `json:"datasource"`
	ApplicationAccess         string                           `json:"applicationAccess"`
	Contract                  string                           `json:"contract"`
	ContractQuery             string                           `json:"contractQuery"`
	ContractReads             []string                         `json:"contractReads,omitempty"`
	ContractWrites            []string                         `json:"contractWrites,omitempty"`
	ConflictCodes             []string                         `json:"conflictCodes,omitempty"`
	UnresolvedDimensions      []string                         `json:"unresolvedDimensions,omitempty"`
	AuthorityLimits           []string                         `json:"authorityLimits"`
}

func TestActorContextR1FindsCustomerEditConflictBeforeInventingWriteAuthority(t *testing.T) {
	root := repoPath()
	workspacePath := filepath.Join(root, "reference", "workspace")
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	conflictPath := filepath.Join(root, "reference", "business-system-design", "order-entry-customer-edit-conflict.json")

	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	sourceIdx, err := workspace.BuildSourceIndex(workspacePath, loaded)
	if err != nil {
		t.Fatal(err)
	}

	got, err := deriveActorContextR1ConstraintProbe(loaded.World, sourceIdx, f, workspacePath, fixturePath, conflictPath)
	if err != nil {
		t.Fatal(err)
	}

	if got.ReferenceFingerprint == "" {
		t.Fatal("R1 lost the Business Procedure Reference fingerprint")
	}
	for _, subject := range []struct {
		kind string
		id   string
	}{
		{kind: "application", id: actorContextR1ApplicationSubject},
		{kind: "application-pattern", id: actorContextR1PatternSubject},
		{kind: "database-contract", id: actorContextR1ContractSubject},
	} {
		if !actorContextR0HasInputSubject(got.RequiredInputFingerprints, subject.kind, subject.id) {
			t.Fatalf("R1 freshness set does not include decisive subject %s:%s: %+v", subject.kind, subject.id, got.RequiredInputFingerprints)
		}
	}
	for _, artifact := range []string{actorContextR0ProcedureArtifact, actorContextR0ConflictArtifact} {
		if !actorContextR1HasInputArtifact(got.RequiredInputFingerprints, artifact) {
			t.Fatalf("R1 freshness set does not include decisive Reference input %q: %+v", artifact, got.RequiredInputFingerprints)
		}
	}

	if got.BusinessStep != "BP-02" ||
		got.BusinessAction != "confirm the customer associated with the order" ||
		got.BusinessRealization != "application-use" {
		t.Fatalf("R1 lost the current Business Procedure Reference meaning: %+v", got)
	}
	if got.Pattern != "order_entry.customer_reference" || got.PatternKind != string(application.PatternReference) {
		t.Fatalf("R1 did not preserve customer_reference as a reference Pattern: %+v", got)
	}
	if got.Relation != "sales.order.customer" || got.Datasource != "master" {
		t.Fatalf("R1 lost the exact relation/datasource path: %+v", got)
	}
	if got.ApplicationAccess != string(database.AccessRead) {
		t.Fatalf("R1 expanded accepted master access beyond read: %+v", got)
	}
	if got.Contract != "master_data.customer" || got.ContractQuery != "find_by_id" {
		t.Fatalf("R1 lost the exact Contract Query path: %+v", got)
	}
	if !actorContextR0Contains(got.ContractReads, "customer.customer") {
		t.Fatalf("R1 lost the customer read Contract meaning: %+v", got)
	}
	if len(got.ContractWrites) != 0 {
		t.Fatalf("R1 invented a customer write Contract: %+v", got.ContractWrites)
	}
	if !actorContextR0Contains(got.ConflictCodes, "BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT") {
		t.Fatalf("R1 failed to expose the existing read/write conflict: %+v", got)
	}
	if len(got.UnresolvedDimensions) == 0 {
		t.Fatalf("R1 collapsed design alternatives instead of retaining unresolved dimensions: %+v", got)
	}

	assertActorContextR1NoResolutionShortcut(t, reflect.TypeOf(actorContextR1ConstraintProbe{}))
}

func TestActorContextR1DoesNotGeneralizeCurrentPathConflictToFeatureWideImpossibility(t *testing.T) {
	typ := reflect.TypeOf(actorContextR1ConstraintProbe{})
	for _, forbidden := range []string{
		"FeatureImpossible",
		"AcceptedResolution",
		"Winner",
		"Preferred",
		"AutoPatch",
		"WriteCapability",
		"CustomerOwner",
		"CanExecute",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R1 probe introduced an unsupported resolution shortcut field %q", forbidden)
		}
	}
}

func deriveActorContextR1ConstraintProbe(
	world ir.World,
	sourceIdx sourceindex.Index,
	f fixture,
	workspacePath,
	fixturePath,
	conflictPath string,
) (actorContextR1ConstraintProbe, error) {
	if f.Canonical {
		return actorContextR1ConstraintProbe{}, errProcedureMeaning("R1 Business Procedure fixture must remain non-canonical")
	}
	var businessStep *step
	for i := range f.Steps {
		if f.Steps[i].ID == "BP-02" {
			businessStep = &f.Steps[i]
			break
		}
	}
	if businessStep == nil {
		return actorContextR1ConstraintProbe{}, errProcedureMeaning("BP-02 not found")
	}
	if len(businessStep.ApplicationPatterns) != 1 || businessStep.ApplicationPatterns[0] != "order_entry.customer_reference" {
		return actorContextR1ConstraintProbe{}, errProcedureMeaning("BP-02 customer_reference realization changed")
	}

	app, pattern, ok := findPatternOwner(world, "order_entry.customer_reference")
	if !ok {
		return actorContextR1ConstraintProbe{}, errProcedureMeaning("order_entry.customer_reference did not resolve exactly once")
	}
	req, ok := app.DatabaseByAliasExact(pattern.Datasource)
	if !ok {
		return actorContextR1ConstraintProbe{}, errProcedureMeaning("customer_reference datasource did not resolve exactly once")
	}
	contract, queryName, err := actorContextR1ResolveContractQuery(world, req, pattern)
	if err != nil {
		return actorContextR1ConstraintProbe{}, err
	}

	result, err := run(workspacePath, fixturePath, conflictPath)
	if err != nil {
		return actorContextR1ConstraintProbe{}, err
	}
	var codes []string
	for _, item := range result.Diagnostics {
		codes = append(codes, item.Code)
	}
	sort.Strings(codes)

	reads := make([]string, 0, len(contract.Reads))
	for _, ref := range contract.Reads {
		reads = append(reads, string(ref))
	}
	writes := make([]string, 0, len(contract.Writes))
	for _, ref := range contract.Writes {
		writes = append(writes, string(ref))
	}
	sort.Strings(reads)
	sort.Strings(writes)

	referenceFingerprint, err := actorContextR0Fingerprint(fixturePath)
	if err != nil {
		return actorContextR1ConstraintProbe{}, err
	}
	inputs, err := actorContextR1InputFingerprints(sourceIdx, fixturePath, conflictPath)
	if err != nil {
		return actorContextR1ConstraintProbe{}, err
	}

	return actorContextR1ConstraintProbe{
		ChangeRequest:             "allow editing and updating the customer referenced from order_entry",
		ReferenceFingerprint:      referenceFingerprint,
		RequiredInputFingerprints: inputs,
		BusinessStep:              businessStep.ID,
		BusinessAction:            businessStep.BusinessAction,
		BusinessRealization:       businessStep.Realization,
		Pattern:                   string(pattern.ID),
		PatternKind:               string(pattern.Kind),
		Relation:                  string(pattern.Relation),
		Datasource:                string(pattern.Datasource),
		ApplicationAccess:         string(req.Access),
		Contract:                  string(pattern.ContractQuery.Contract),
		ContractQuery:             queryName,
		ContractReads:             reads,
		ContractWrites:            writes,
		ConflictCodes:             codes,
		UnresolvedDimensions: []string{
			"customer-update-ownership",
			"write-authority-path",
			"business-responsibility-expansion",
			"capability-or-handoff-design",
		},
		AuthorityLimits: []string{
			"current-path-conflict-does-not-prove-feature-impossibility",
			"change-request-does-not-create-write-authority",
			"design-alternatives-remain-proposals-until-accepted",
		},
	}, nil
}

func actorContextR1InputFingerprints(sourceIdx sourceindex.Index, fixturePath, conflictPath string) ([]actorContextR0InputFingerprint, error) {
	specs := []struct {
		kind string
		id   string
	}{
		{kind: "application", id: actorContextR1ApplicationSubject},
		{kind: "application-pattern", id: actorContextR1PatternSubject},
		{kind: "database-contract", id: actorContextR1ContractSubject},
	}

	var inputs []actorContextR0InputFingerprint
	for _, spec := range specs {
		input, err := actorContextR1SourceIndexInputFingerprint(sourceIdx, spec.kind, spec.id)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input)
	}
	for _, spec := range []struct {
		scope    string
		artifact string
		path     string
	}{
		{scope: "business-procedure-reference", artifact: actorContextR0ProcedureArtifact, path: fixturePath},
		{scope: "review-evidence", artifact: actorContextR0ConflictArtifact, path: conflictPath},
	} {
		input, err := actorContextR0FileInputFingerprint(spec.scope, spec.artifact, spec.path)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input)
	}
	return actorContextR0NormalizeInputFingerprints(inputs), nil
}

func actorContextR1SourceIndexInputFingerprint(idx sourceindex.Index, kind, id string) (actorContextR0InputFingerprint, error) {
	refs := idx.ReferencesFor(sourceindex.Subject{Kind: kind, ID: id})
	var definitionRefs []sourceindex.Reference
	for _, ref := range refs {
		if ref.Role == sourceindex.RoleDefinition {
			definitionRefs = append(definitionRefs, ref)
		}
	}
	if len(definitionRefs) != 1 {
		return actorContextR0InputFingerprint{}, errProcedureMeaning(
			fmt.Sprintf("R1 source subject %s:%s requires exactly one definition reference; got %d", kind, id, len(definitionRefs)),
		)
	}
	ref := definitionRefs[0]
	artifact, ok := actorContextR0ArtifactByPath(idx, ref.ArtifactPath)
	if !ok {
		return actorContextR0InputFingerprint{}, errProcedureMeaning(
			fmt.Sprintf("R1 source subject %s:%s points to missing artifact %s", kind, id, ref.ArtifactPath),
		)
	}
	return actorContextR0InputFingerprint{
		Scope:       "source-definition",
		SubjectKind: kind,
		SubjectID:   id,
		Artifact:    "reference/workspace/" + artifact.Path,
		Fingerprint: artifact.Fingerprint,
	}, nil
}

func actorContextR1HasInputArtifact(values []actorContextR0InputFingerprint, artifact string) bool {
	for _, value := range values {
		if value.Artifact == artifact {
			return true
		}
	}
	return false
}

func actorContextR1ResolveContractQuery(
	world ir.World,
	req application.DatabaseRequirement,
	pattern application.Pattern,
) (database.Contract, string, error) {
	if pattern.ContractQuery == nil {
		return database.Contract{}, "", errProcedureMeaning("customer_reference has no Contract Query")
	}

	var matchedDB *database.Database
	for i := range world.Databases {
		if world.Databases[i].ID != req.TargetDatabase {
			continue
		}
		if matchedDB != nil {
			return database.Contract{}, "", errProcedureMeaning("target database is ambiguous")
		}
		matchedDB = &world.Databases[i]
	}
	if matchedDB == nil {
		return database.Contract{}, "", errProcedureMeaning("target database not found")
	}

	var matchedContract *database.Contract
	for i := range matchedDB.Contracts {
		if matchedDB.Contracts[i].ID != pattern.ContractQuery.Contract {
			continue
		}
		if matchedContract != nil {
			return database.Contract{}, "", errProcedureMeaning("database Contract is ambiguous")
		}
		matchedContract = &matchedDB.Contracts[i]
	}
	if matchedContract == nil {
		return database.Contract{}, "", errProcedureMeaning("database Contract not found")
	}

	queryName := string(pattern.ContractQuery.Query)
	matches := 0
	for _, query := range matchedContract.Queries {
		if query.Name == queryName {
			matches++
		}
	}
	if matches != 1 {
		return database.Contract{}, "", errProcedureMeaning("Contract Query did not resolve exactly once")
	}
	return *matchedContract, queryName, nil
}

func assertActorContextR1NoResolutionShortcut(t *testing.T, typ reflect.Type) {
	t.Helper()
	for _, forbidden := range []string{
		"AcceptedResolution",
		"RecommendedResolution",
		"Winner",
		"Preferred",
		"FeatureImpossible",
		"CustomerOwner",
		"WriteCapability",
		"AutoPatch",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R1 constraint probe introduced unsupported resolution shortcut field %q", forbidden)
		}
	}
}
