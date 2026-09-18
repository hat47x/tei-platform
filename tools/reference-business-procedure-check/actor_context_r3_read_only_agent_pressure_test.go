package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/workspace"
)

type actorContextR3ReadIntentProbe struct {
	ReferenceFingerprint      string                           `json:"referenceFingerprint"`
	RequiredInputFingerprints []actorContextR0InputFingerprint `json:"requiredInputFingerprints"`
	Procedure                 string                           `json:"procedure"`
	Step                   string   `json:"step"`
	BusinessAction         string   `json:"businessAction"`
	Pattern                string   `json:"pattern"`
	PatternKind            string   `json:"patternKind"`
	Relation               string   `json:"relation"`
	Datasource             string   `json:"datasource"`
	ApplicationAccess      string   `json:"applicationAccess"`
	Contract               string   `json:"contract"`
	Query                  string   `json:"query"`
	InputSemanticAttribute string   `json:"inputSemanticAttribute"`
	QueryInput             string   `json:"queryInput"`
	QueryInputType         string   `json:"queryInputType"`
	Result                 string   `json:"result"`
	ResultCardinality      string   `json:"resultCardinality"`
	RuntimeGaps            []string `json:"runtimeGaps"`
	InvocationAttempted    bool     `json:"invocationAttempted"`
	ObservationReceived    bool     `json:"observationReceived"`
	CompletionUnresolved   bool     `json:"completionUnresolved"`
	SourceRefs             []string `json:"sourceRefs"`
	AuthorityLimits        []string `json:"authorityLimits"`
}

func TestActorContextR3DerivesReadIntentWithoutInventingRuntimeAuthority(t *testing.T) {
	root := repoPath()
	workspacePath := filepath.Join(root, "reference", "workspace")
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}

	got, err := deriveActorContextR3ReadIntentProbe(loaded.World, f, workspacePath, fixturePath)
	if err != nil {
		t.Fatal(err)
	}

	if got.ReferenceFingerprint == "" {
		t.Fatal("R3 lost the Business Procedure Reference fingerprint")
	}
	for _, artifact := range []string{
		actorContextR1ApplicationArtifact,
		actorContextR1PatternArtifact,
		actorContextR1ContractArtifact,
		actorContextR0ProcedureArtifact,
	} {
		if !actorContextR1HasInputArtifact(got.RequiredInputFingerprints, artifact) {
			t.Fatalf("R3 freshness set does not include decisive read-route input %q: %+v", artifact, got.RequiredInputFingerprints)
		}
	}
	if actorContextR1HasInputArtifact(got.RequiredInputFingerprints, actorContextR0ConflictArtifact) {
		t.Fatalf("R3 must not depend on R1 conflict Evidence that is irrelevant to the read route: %+v", got.RequiredInputFingerprints)
	}

	if got.Procedure != "order-entry.order-handling" ||
		got.Step != "BP-02" ||
		got.BusinessAction != "confirm the customer associated with the order" {
		t.Fatalf("R3 lost the Business Action context: %+v", got)
	}
	if got.Pattern != "order_entry.customer_reference" ||
		got.PatternKind != string(application.PatternReference) ||
		got.Relation != "sales.order.customer" {
		t.Fatalf("R3 lost the exact customer reference route: %+v", got)
	}
	if got.Datasource != "master" || got.ApplicationAccess != string(database.AccessRead) {
		t.Fatalf("R3 expanded or lost accepted read-only datasource access: %+v", got)
	}
	if got.Contract != "master_data.customer" || got.Query != "find_by_id" {
		t.Fatalf("R3 lost the exact Contract Query: %+v", got)
	}
	if got.InputSemanticAttribute != "customer.customer.customer_id" ||
		got.QueryInput != "id" ||
		got.QueryInputType != "customer.customer_id" {
		t.Fatalf("R3 lost the semantic-to-query input mapping: %+v", got)
	}
	if got.Result != "customer.customer" ||
		got.ResultCardinality != string(database.QueryResultZeroOrOne) {
		t.Fatalf("R3 lost the expected query result contract: %+v", got)
	}

	for _, gap := range []string{
		"actor-principal-unresolved",
		"input-instance-value-unresolved",
		"authorization-not-evaluated",
		"resource-provider-not-evaluated",
	} {
		if !actorContextR0Contains(got.RuntimeGaps, gap) {
			t.Fatalf("R3 hid runtime gap %q: %+v", gap, got.RuntimeGaps)
		}
	}
	if got.InvocationAttempted {
		t.Fatal("R3 static source must not invoke the Contract Query")
	}
	if got.ObservationReceived {
		t.Fatal("R3 static source must not fabricate an Observation")
	}
	if !got.CompletionUnresolved {
		t.Fatal("R3 route resolution must not imply Business Action completion")
	}

	assertActorContextR3NoRuntimeAuthorityShortcut(t, reflect.TypeOf(actorContextR3ReadIntentProbe{}))
}

func deriveActorContextR3ReadIntentProbe(world ir.World, f fixture, workspacePath, fixturePath string) (actorContextR3ReadIntentProbe, error) {
	if f.Canonical {
		return actorContextR3ReadIntentProbe{}, errProcedureMeaning("R3 Business Procedure fixture must remain non-canonical")
	}

	var bp02 *step
	for i := range f.Steps {
		if f.Steps[i].ID == "BP-02" {
			bp02 = &f.Steps[i]
			break
		}
	}
	if bp02 == nil {
		return actorContextR3ReadIntentProbe{}, errProcedureMeaning("BP-02 not found")
	}
	if len(bp02.ApplicationPatterns) != 1 || bp02.ApplicationPatterns[0] != "order_entry.customer_reference" {
		return actorContextR3ReadIntentProbe{}, errProcedureMeaning("BP-02 customer_reference realization changed")
	}

	app, pattern, ok := findPatternOwner(world, bp02.ApplicationPatterns[0])
	if !ok {
		return actorContextR3ReadIntentProbe{}, errProcedureMeaning("customer_reference did not resolve exactly once")
	}
	if pattern.ContractQuery == nil {
		return actorContextR3ReadIntentProbe{}, errProcedureMeaning("customer_reference Contract Query missing")
	}
	if len(pattern.ContractQuery.Inputs) != 1 {
		return actorContextR3ReadIntentProbe{}, errProcedureMeaning("R3 requires exactly one explicit Contract Query input binding")
	}

	req, ok := app.DatabaseByAliasExact(pattern.Datasource)
	if !ok {
		return actorContextR3ReadIntentProbe{}, errProcedureMeaning("customer_reference datasource did not resolve exactly once")
	}
	contract, queryName, err := actorContextR1ResolveContractQuery(world, req, pattern)
	if err != nil {
		return actorContextR3ReadIntentProbe{}, err
	}

	var query *database.QueryContract
	for i := range contract.Queries {
		if contract.Queries[i].Name != queryName {
			continue
		}
		if query != nil {
			return actorContextR3ReadIntentProbe{}, errProcedureMeaning("Contract Query is ambiguous")
		}
		query = &contract.Queries[i]
	}
	if query == nil {
		return actorContextR3ReadIntentProbe{}, errProcedureMeaning("Contract Query not found")
	}
	if len(query.Inputs) != 1 {
		return actorContextR3ReadIntentProbe{}, errProcedureMeaning("R3 requires exactly one Contract Query input")
	}

	binding := pattern.ContractQuery.Inputs[0]
	input := query.Inputs[0]
	if binding.Input != input.Name {
		return actorContextR3ReadIntentProbe{}, errProcedureMeaning("Pattern input binding does not match Contract Query input")
	}

	referenceFingerprint, err := actorContextR0Fingerprint(fixturePath)
	if err != nil {
		return actorContextR3ReadIntentProbe{}, err
	}
	inputs, err := actorContextR3InputFingerprints(workspacePath, fixturePath)
	if err != nil {
		return actorContextR3ReadIntentProbe{}, err
	}

	return actorContextR3ReadIntentProbe{
		ReferenceFingerprint:      referenceFingerprint,
		RequiredInputFingerprints: inputs,
		Procedure:              f.Procedure.ID,
		Step:                   bp02.ID,
		BusinessAction:         bp02.BusinessAction,
		Pattern:                string(pattern.ID),
		PatternKind:            string(pattern.Kind),
		Relation:               string(pattern.Relation),
		Datasource:             string(pattern.Datasource),
		ApplicationAccess:      string(req.Access),
		Contract:               string(pattern.ContractQuery.Contract),
		Query:                  queryName,
		InputSemanticAttribute: string(binding.TargetAttribute),
		QueryInput:             string(input.Name),
		QueryInputType:         string(input.Type),
		Result:                 string(query.Result),
		ResultCardinality:      string(query.ResultCardinality),
		RuntimeGaps: []string{
			"actor-principal-unresolved",
			"input-instance-value-unresolved",
			"authorization-not-evaluated",
			"resource-provider-not-evaluated",
		},
		InvocationAttempted:  false,
		ObservationReceived:  false,
		CompletionUnresolved: true,
		SourceRefs: actorContextR3InputArtifacts(inputs),
		AuthorityLimits: []string{
			"resolved-read-route-does-not-grant-agent-authority",
			"application-access-does-not-equal-agent-authorization",
			"static-input-binding-does-not-provide-instance-value",
			"query-route-does-not-prove-business-completion",
		},
	}, nil
}

func actorContextR3InputFingerprints(workspacePath, fixturePath string) ([]actorContextR0InputFingerprint, error) {
	specs := []struct {
		scope    string
		artifact string
		path     string
	}{
		{scope: "source-definition", artifact: actorContextR1ApplicationArtifact, path: filepath.Join(workspacePath, "applications", "order-entry", "application.tei")},
		{scope: "source-definition", artifact: actorContextR1PatternArtifact, path: filepath.Join(workspacePath, "applications", "order-entry", "patterns", "customer-reference.tei")},
		{scope: "source-definition", artifact: actorContextR1ContractArtifact, path: filepath.Join(workspacePath, "databases", "master-data", "contracts", "customer.tei")},
		{scope: "business-procedure-reference", artifact: actorContextR0ProcedureArtifact, path: fixturePath},
	}
	inputs := make([]actorContextR0InputFingerprint, 0, len(specs))
	for _, spec := range specs {
		input, err := actorContextR0FileInputFingerprint(spec.scope, spec.artifact, spec.path)
		if err != nil {
			return nil, err
		}
		inputs = append(inputs, input)
	}
	return actorContextR0NormalizeInputFingerprints(inputs), nil
}

func actorContextR3InputArtifacts(values []actorContextR0InputFingerprint) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value.Artifact != "" {
			out = append(out, value.Artifact)
		}
	}
	return actorContextR0SortedUnique(out)
}

func assertActorContextR3NoRuntimeAuthorityShortcut(t *testing.T, typ reflect.Type) {
	t.Helper()
	for _, forbidden := range []string{
		"Authorized",
		"CanExecute",
		"Principal",
		"CustomerIDValue",
		"Credential",
		"QuerySucceeded",
		"BusinessCompleted",
		"AgentTrust",
		"Confidence",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R3 read intent probe introduced forbidden runtime authority shortcut field %q", forbidden)
		}
	}
}
