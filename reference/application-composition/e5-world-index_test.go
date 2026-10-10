package main

import (
    "reflect"
    "testing"

    "tei/internal/application"
    "tei/internal/core"
    "tei/internal/database"
    "tei/internal/ir"
    "tei/internal/semantic"
    "tei/internal/sourceindex"
)

func e5Fixture() (ir.World, sourceindex.Index) {
    world := ir.World{
        Applications: []application.Application{{
            ID: core.CanonicalID("order_entry"),
            Validations: []core.CanonicalID{"sales.order_line.quantity_positive"},
            Databases: []application.DatabaseRequirement{{
                Alias: "sales",
                TargetDatabase: "sales_core",
                RequiredContracts: []database.ContractRequirement{{ContractID:"sales_core.order",CompatibleMajor:4}},
            }},
        }},
        Semantics: []semantic.Namespace{{ID:"sales", Validations: []semantic.Validation{{ID:"sales.order_line.quantity_positive"}}}},
        Databases: []database.Database{{ID:"sales_core",Contracts: []database.Contract{{
            ID:"sales_core.order",Version:database.ContractVersion{Major:4,Minor:2},
        }}}},
    }
    index := sourceindex.Index{
        Artifacts: []sourceindex.Artifact{
            {Path:"applications/order-entry/application.tei",Fingerprint:"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
            {Path:"semantic/sales/validation.tei",Fingerprint:"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
            {Path:"databases/sales-core/database.tei",Fingerprint:"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
            {Path:"databases/sales-core/contracts/order.tei",Fingerprint:"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
        },
        References: []sourceindex.Reference{
            {Subject:sourceindex.Subject{Kind:"application",ID:"order_entry"},ArtifactPath:"applications/order-entry/application.tei",Role:sourceindex.RoleDefinition},
            {Subject:sourceindex.Subject{Kind:"semantic-validation",ID:"sales.order_line.quantity_positive"},ArtifactPath:"semantic/sales/validation.tei",Role:sourceindex.RoleDefinition},
            {Subject:sourceindex.Subject{Kind:"database",ID:"sales_core"},ArtifactPath:"databases/sales-core/database.tei",Role:sourceindex.RoleDefinition},
            {Subject:sourceindex.Subject{Kind:"database-contract",ID:"sales_core.order"},ArtifactPath:"databases/sales-core/contracts/order.tei",Role:sourceindex.RoleDefinition},
        },
    }
    return world,index
}

func e5HasIssue(report e5Report, reason string) bool {
    for _, issue := range report.Issues {
        if issue.Reason == reason { return true }
    }
    return false
}

func TestE5ResolvesTypedWorldToSourceDefinitions(t *testing.T) {
    world,index:=e5Fixture()
    got:=inspectE5(world,index,"order_entry")
    if got.Status!="observed" || len(got.Dependencies)!=4 || len(got.Issues)!=0 {
        t.Fatalf("unexpected report: %+v",got)
    }
    hasContract:=false
    for _, d := range got.Dependencies {
        if d.Relation=="requires-contract" && d.Subject.ID=="sales_core.order" &&
            d.Source.Path=="databases/sales-core/contracts/order.tei" { hasContract=true }
    }
    if !hasContract {t.Fatalf("contract source missing: %+v",got)}
}

func TestE5FailsClosedOnMissingValidationDefinition(t *testing.T) {
    world,index:=e5Fixture()
    world.Semantics[0].Validations=nil
    report:=inspectE5(world,index,"order_entry")
    if report.Status!="unresolved" || !e5HasIssue(report,"validation_definition_missing_or_ambiguous") {t.Fatalf("%+v",report)}
}

func TestE5FailsClosedOnDuplicateValidationDefinition(t *testing.T) {
    world,index:=e5Fixture()
    world.Semantics[0].Validations=append(world.Semantics[0].Validations,world.Semantics[0].Validations[0])
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"validation_definition_missing_or_ambiguous"){t.Fatal("duplicate accepted")}
}

func TestE5FailsClosedOnDuplicateApplicationValidationUse(t *testing.T) {
    world,index:=e5Fixture()
    world.Applications[0].Validations=append(world.Applications[0].Validations,world.Applications[0].Validations[0])
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"duplicate_application_reference"){t.Fatal("duplicate use accepted")}
}

func TestE5FailsClosedOnMissingIndexedContractDefinition(t *testing.T) {
    world,index:=e5Fixture()
    index.References=index.References[:len(index.References)-1]
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"source_definition_missing_or_ambiguous"){t.Fatal("missing index reference accepted")}
}

func TestE5RejectsDuplicateIndexedDefinition(t *testing.T) {
    world,index:=e5Fixture()
    index.References=append(index.References,index.References[len(index.References)-1])
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"source_definition_missing_or_ambiguous"){t.Fatal("ambiguous source accepted")}
}

func TestE5DoesNotSubstituteAnotherDatabaseContract(t *testing.T) {
    world,index:=e5Fixture()
    world.Applications[0].Databases[0].RequiredContracts[0].ContractID="sales_core.missing"
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"target_contract_missing_or_ambiguous"){t.Fatal("wrong contract accepted")}
}

func TestE5RejectsDeclaredMajorMismatch(t *testing.T) {
    world,index:=e5Fixture()
    world.Databases[0].Contracts[0].Version.Major=5
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"declared_major_mismatch"){t.Fatal("mismatched major accepted")}
}

func TestE5RejectsAmbiguousDatabaseTarget(t *testing.T) {
    world,index:=e5Fixture()
    world.Databases=append(world.Databases,world.Databases[0])
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"database_definition_missing_or_ambiguous"){t.Fatal("duplicate database accepted")}
}

func TestE5RejectsDuplicateDatabaseAlias(t *testing.T) {
    world,index:=e5Fixture()
    world.Applications[0].Databases=append(world.Applications[0].Databases,world.Applications[0].Databases[0])
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"duplicate_or_empty_datasource_alias"){t.Fatal("duplicate alias accepted")}
}

func TestE5RejectsAmbiguousApplication(t *testing.T) {
    world,index:=e5Fixture()
    world.Applications=append(world.Applications,world.Applications[0])
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"application_missing_or_ambiguous"){t.Fatal("duplicate app accepted")}
}

func TestE5UnrelatedIndexedSourceDoesNotChangeReport(t *testing.T) {
    world,index:=e5Fixture()
    baseline:=inspectE5(world,index,"order_entry")
    index.Artifacts=append(index.Artifacts,sourceindex.Artifact{Path:"semantic/unrelated/other.tei",Fingerprint:"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"})
    next:=inspectE5(world,index,"order_entry")
    if !reflect.DeepEqual(baseline,next){t.Fatalf("unrelated source affected report: %+v",next)}
}

func TestE5RejectsMissingArtifactForReference(t *testing.T) {
    world,index:=e5Fixture()
    index.Artifacts=index.Artifacts[:len(index.Artifacts)-1]
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"source_definition_missing_or_ambiguous"){t.Fatal("missing artifact accepted")}
}

func TestE5MissingApplicationDoesNotPickFallback(t *testing.T) {
    world,index:=e5Fixture()
    if !e5HasIssue(inspectE5(world,index,"missing"),"application_missing_or_ambiguous"){t.Fatal("unknown app accepted")}
}
