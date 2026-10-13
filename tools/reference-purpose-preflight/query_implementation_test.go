package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"tei/internal/database"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/realization"
)

func TestActualOracleContractSignatureDoesNotEstablishProvider(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	for _, env := range []string{"development", "production"} {
		t.Run(env, func(t *testing.T) {
			assessment, err := InspectWorkspaceForEnvironment(root, "order_entry", "application-pattern", "order_entry.customer_reference", env)
			if err != nil {
				t.Fatal(err)
			}
			observed := assessment.QueryImplementation
			if observed == nil || observed.Status != "contract-signature-declared" ||
				observed.Contract != "master_data.customer" || observed.Query != "find_by_id" ||
				observed.Database != "master_data" || observed.Provider != "oracle" ||
				observed.ContractMajor != 2 || observed.ContractMinor != 4 {
				t.Fatalf("incorrect Canonical Query signature or false SQL authority: %+v", assessment)
			}
			projected, err := ProjectForPurpose(assessment, "execute-preflight")
			if err != nil {
				t.Fatal(err)
			}
			obligation, ok := purposeRequirementByName(projected, "query-implementation")
			if !ok || obligation.Status != "conditional" || projected.ExecutionAllowed {
				t.Fatalf("missing or overstated Query implementation evidence: %+v", projected)
			}
		})
	}
}

func queryImplementationWorld() (use DeclaredContractQuery, binding EnvironmentBindingObservation, world ir.World) {
	world = syntheticEnvironmentBindingWorld()
	world.Databases = []database.Database{{
		ID: "master_data",
		Contracts: []database.Contract{{
			ID:      "master_data.customer",
			Version: database.ContractVersion{Major: 2, Minor: 4},
			Queries: []database.QueryContract{{
				Name:              "find_by_id",
				Inputs:            []database.QueryInput{{Name: "id", Type: "customer.customer_id"}},
				Result:            "customer.customer",
				ResultCardinality: database.QueryResultZeroOrOne,
			}},
		}},
	}}
	use = ObserveDeclaredContractQuery(world, "example", "example.customer_reference")
	binding = ObserveEnvironmentBinding(world, "development", "example", &use)
	return use, binding, world
}

func TestExactSQLRealizationDeclarationIsStillNotProviderConformance(t *testing.T) {
	use, binding, world := queryImplementationWorld()
	world.ContractQuerySQLRealizations = []realization.ContractQuerySQLRealization{{
		Target: realization.ContractQueryTarget{
			Database: "master_data", Contract: "master_data.customer", Query: "find_by_id",
		},
		Provider:      environment.ProviderOracle,
		Source:        "select customer_id from customer where customer_id = :id",
		Inputs:        []realization.ContractQueryInputParameterMapping{{Input: "id", Parameter: "id"}},
		ResultColumns: []query.ResultColumnMapping{{Column: "customer_id", Attribute: "customer.customer_id"}},
	}}
	observed := ObserveQueryImplementation(world, &use, &binding)
	if observed.Status != "sql-realization-declared" {
		t.Fatalf("valid explicit native SQL declaration not observed: %+v", observed)
	}
	assessment := WorkspaceAssessment{
		SubjectKind: "application-pattern", SubjectID: "example.customer_reference",
		QueryImplementation: &observed,
	}
	projected, err := ProjectForPurpose(assessment, "execute-preflight")
	if err != nil {
		t.Fatal(err)
	}
	ob, ok := purposeRequirementByName(projected, "query-implementation")
	if !ok || ob.Status != "not-evaluated" || projected.ExecutionAllowed {
		t.Fatalf("authored SQL was improperly elevated to runtime authority: %+v", projected)
	}
	world.ContractQuerySQLRealizations = append(world.ContractQuerySQLRealizations, world.ContractQuerySQLRealizations[0])
	if got := ObserveQueryImplementation(world, &use, &binding); got.Status != "unresolved" ||
		got.Reason != "ambiguous-sql-realization" {
		t.Fatalf("duplicate SQL realization silently selected: %+v", got)
	}
	world.ContractQuerySQLRealizations = world.ContractQuerySQLRealizations[:1]
	world.ContractQuerySQLRealizations[0].Inputs = nil
	if got := ObserveQueryImplementation(world, &use, &binding); got.Status != "unresolved" ||
		got.Reason != "sql-realization-contract-shape-invalid" {
		t.Fatalf("invalid SQL input mapping treated as qualified: %+v", got)
	}
}

func TestQuerySignatureVersionAndPrerequisitesAreExact(t *testing.T) {
	use, binding, world := queryImplementationWorld()
	if got := ObserveQueryImplementation(world, &use, &binding); got.Status != "contract-signature-declared" {
		t.Fatalf("signature-only case not represented: %+v", got)
	}
	world.Databases[0].Contracts[0].Version.Major = 3
	if got := ObserveQueryImplementation(world, &use, &binding); got.Status != "incompatible" {
		t.Fatalf("wrong Contract major was considered compatible: %+v", got)
	}
	_, _, world = queryImplementationWorld()
	world.Databases[0].Contracts[0].Queries = nil
	if got := ObserveQueryImplementation(world, &use, &binding); got.Status != "unresolved" {
		t.Fatalf("missing Query signature was accepted: %+v", got)
	}
	_, _, world = queryImplementationWorld()
	world.Databases = append(world.Databases, world.Databases[0])
	if got := ObserveQueryImplementation(world, &use, &binding); got.Status != "unresolved" {
		t.Fatalf("ambiguous Database selected: %+v", got)
	}
	if got := ObserveQueryImplementation(world, &use, nil); got.Status != "unresolved" {
		t.Fatalf("missing Environment binding promoted to Contract implementation: %+v", got)
	}
}

func TestQueryImplementationCLIIsOptInAndReadOnly(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	base := []string{
		"-workspace", root, "-application", "order_entry",
		"-subject-kind", "application-pattern", "-subject-id", "order_entry.customer_reference",
	}
	var legacy bytes.Buffer
	if err := runCLI(base, &legacy); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(legacy.Bytes(), []byte("queryImplementation")) {
		t.Fatalf("new implementation metadata leaked into previous flat output: %s", legacy.String())
	}
	var scoped bytes.Buffer
	args := append(append([]string{}, base...), "-purpose", "execute-preflight", "-environment", "development")
	if err := runCLI(args, &scoped); err != nil {
		t.Fatal(err)
	}
	var value struct {
		Assessment WorkspaceAssessment `json:"assessment"`
		Readiness  PurposeProjection   `json:"readiness"`
	}
	if err := json.Unmarshal(scoped.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	if value.Assessment.QueryImplementation == nil ||
		value.Assessment.QueryImplementation.Status != "contract-signature-declared" ||
		value.Readiness.ExecutionAllowed || value.Assessment.ExecutionAllowed {
		t.Fatalf("incorrect scoped Query implementation evidence: %+v", value)
	}
}
