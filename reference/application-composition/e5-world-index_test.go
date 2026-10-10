package main

import (
    "encoding/json"
    "path/filepath"
    "reflect"
    "testing"

    "tei/internal/application"
    "tei/internal/core"
    "tei/internal/database"
    "tei/internal/ir"
    "tei/internal/query"
    "tei/internal/semantic"
    "tei/internal/sourceindex"
    "tei/internal/workspace"
)

func e5Fixture() (ir.World, sourceindex.Index) {
    world := ir.World{
        Applications: []application.Application{{
            ID: core.CanonicalID("order_entry"),
            SemanticUses: []core.CanonicalID{"sales"},
            Patterns: []application.Pattern{{ID:"order_entry.submit_order",Kind:application.PatternCommand}},
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
            {Path:"applications/order-entry/patterns/submit-order.tei",Fingerprint:"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
            {Path:"semantic/sales/order.tei",Fingerprint:"sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
            {Path:"databases/sales-core/contracts/order.tei",Fingerprint:"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
        },
        References: []sourceindex.Reference{
            {Subject:sourceindex.Subject{Kind:"application",ID:"order_entry"},ArtifactPath:"applications/order-entry/application.tei",Role:sourceindex.RoleDefinition},
            {Subject:sourceindex.Subject{Kind:"semantic-validation",ID:"sales.order_line.quantity_positive"},ArtifactPath:"semantic/sales/validation.tei",Role:sourceindex.RoleDefinition},
            {Subject:sourceindex.Subject{Kind:"database",ID:"sales_core"},ArtifactPath:"databases/sales-core/database.tei",Role:sourceindex.RoleDefinition},
            {Subject:sourceindex.Subject{Kind:"application-pattern",ID:"order_entry.submit_order"},ArtifactPath:"applications/order-entry/patterns/submit-order.tei",Role:sourceindex.RoleDefinition},
            {Subject:sourceindex.Subject{Kind:"semantic",ID:"sales"},ArtifactPath:"semantic/sales/order.tei",Role:sourceindex.RoleDefinition},
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
    if got.Status!="observed" || len(got.Dependencies)!=6 || len(got.Issues)!=0 {
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

func TestE5RejectsDuplicateContractOwner(t *testing.T) {
    world,index:=e5Fixture()
    world.Databases=append(world.Databases,database.Database{
        ID:"other_database",
        Contracts:[]database.Contract{{ID:"sales_core.order",Version:database.ContractVersion{Major:4,Minor:2}}},
    })
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"target_contract_missing_or_ambiguous"){t.Fatal("global duplicate contract identity accepted")}
}

func TestE5DoesNotUseAuthoringAsDefinition(t *testing.T) {
    world,index:=e5Fixture()
    index.References[len(index.References)-1].Role=sourceindex.RoleAuthoring
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"source_definition_missing_or_ambiguous"){t.Fatal("authoring reference used as definition")}
}

func TestE5RejectsMissingSemanticNamespace(t *testing.T) {
    world,index:=e5Fixture()
    world.Semantics[0].ID="other"
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"semantic_namespace_missing_or_ambiguous") {
        t.Fatal("missing semantic namespace accepted")
    }
}

func TestE5RejectsDuplicateApplicationSemanticUse(t *testing.T) {
    world,index:=e5Fixture()
    world.Applications[0].SemanticUses=append(world.Applications[0].SemanticUses,world.Applications[0].SemanticUses[0])
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"duplicate_semantic_use") {
        t.Fatal("duplicate semantic use accepted")
    }
}

func TestE5RejectsMissingPatternDefinitionReference(t *testing.T) {
    world,index:=e5Fixture()
    // The fourth reference is the Application Pattern Source definition.
    index.References=append(index.References[:3],index.References[4:]...)
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"source_definition_missing_or_ambiguous") {
        t.Fatal("pattern source without definition accepted")
    }
}

func TestE5RejectsDuplicateApplicationPattern(t *testing.T) {
    world,index:=e5Fixture()
    world.Applications[0].Patterns=append(world.Applications[0].Patterns,world.Applications[0].Patterns[0])
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"duplicate_or_empty_pattern") {
        t.Fatal("duplicate pattern accepted")
    }
}

func TestE5RejectsMissingSemanticNamespaceSourceReference(t *testing.T) {
    world,index:=e5Fixture()
    // Namespace declaration and its Source Index reference are separate checks.
    index.References=append(index.References[:4],index.References[5:]...)
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"source_definition_missing_or_ambiguous") {
        t.Fatal("semantic source without a definition reference accepted")
    }
}

func TestE5PairedEvidenceIncludesMatchedReportAndIndex(t *testing.T) {
    world, index := e5Fixture()
    report := inspectE5(world, index, "order_entry")
    envelope := e5PairedEvidence(report, index)
    if envelope.Format != "tei.e5-paired-source-observation.v1" ||
        envelope.Qualification != "read-only-paired-input-not-an-authority" {
        t.Fatalf("unexpected evidence contract: %+v", envelope)
    }
    if !reflect.DeepEqual(envelope.Report, report) || !reflect.DeepEqual(envelope.Index, index) {
        t.Fatal("paired evidence changed the provided report or source index")
    }
    raw, err := json.Marshal(envelope)
    if err != nil { t.Fatal(err) }
    var decoded struct {
        Format string `json:"format"`
        Report e5Report `json:"report"`
        Index sourceindex.Index `json:"index"`
        Qualification string `json:"qualification"`
    }
    if err := json.Unmarshal(raw, &decoded); err != nil { t.Fatal(err) }
    if decoded.Format != envelope.Format || !reflect.DeepEqual(decoded.Report, report) ||
        !reflect.DeepEqual(decoded.Index, index) {
        t.Fatalf("serialized transport does not preserve evidence fields: %+v", decoded)
    }
}

func TestE5PairedEvidenceCannotPromoteAnUnresolvedReport(t *testing.T) {
    world, index := e5Fixture()
    world.Applications = nil
    report := inspectE5(world, index, "order_entry")
    paired := e5PairedEvidence(report, index)
    if paired.Report.Status != "unresolved" || len(paired.Report.Issues) == 0 ||
        paired.Qualification != "read-only-paired-input-not-an-authority" {
        t.Fatalf("diagnostic was promoted by pairing: %+v", paired)
    }
}

func TestE5TwoAliasesMayShareOneDatabaseContract(t *testing.T) {
    world, index := e5Fixture()
    another := world.Applications[0].Databases[0]
    another.Alias = "sales_reporting"
    world.Applications[0].Databases = append(world.Applications[0].Databases, another)
    report := inspectE5(world, index, "order_entry")
    if report.Status != "observed" || len(report.Issues) != 0 {
        t.Fatalf("distinct aliases to the same contract must be allowed: %+v", report)
    }
    // E6 treats dependency Subject identity as a set, not an alias multiset.
    if len(report.Dependencies) != 6 {
        t.Fatalf("expected six unique definition subjects, got %+v", report.Dependencies)
    }
    countDB, countContract := 0, 0
    for _, dep := range report.Dependencies {
        if dep.Relation == "database-target" { countDB++ }
        if dep.Relation == "requires-contract" { countContract++ }
    }
    if countDB != 1 || countContract != 1 {
        t.Fatalf("duplicate logical Source subjects were emitted: %+v", report.Dependencies)
    }
}

func TestE5DuplicateContractWithinOneAliasStillFails(t *testing.T) {
    world, index := e5Fixture()
    req := world.Applications[0].Databases[0].RequiredContracts[0]
    world.Applications[0].Databases[0].RequiredContracts =
        append(world.Applications[0].Databases[0].RequiredContracts, req)
    if !e5HasIssue(inspectE5(world, index, "order_entry"), "duplicate_contract_requirement") {
        t.Fatal("duplicate contract inside one alias was accepted")
    }
}

func TestE5OtherAliasMajorMismatchCannotHideBehindValidAlias(t *testing.T) {
    world, index := e5Fixture()
    another := world.Applications[0].Databases[0]
    another.Alias = "sales_reporting"
    another.RequiredContracts = []database.ContractRequirement{{
        ContractID: "sales_core.order", CompatibleMajor: 5,
    }}
    world.Applications[0].Databases = append(world.Applications[0].Databases, another)
    report := inspectE5(world, index, "order_entry")
    if report.Status != "unresolved" || !e5HasIssue(report, "declared_major_mismatch") {
        t.Fatalf("second alias masked a major mismatch: %+v", report)
    }
}

func e5PatternTargetsFixture() (ir.World, sourceindex.Index) {
    world, index := e5Fixture()
    world.Semantics[0].Data = []semantic.Data{{
        ID: "sales.order", Relations: []semantic.Relation{{ID: "sales.order.lines"}},
    }}
    world.Semantics[0].Capabilities = []semantic.Capability{{ID: "sales.order.submit"}}
    world.Queries = []query.EmbeddedSQL{{ID: "order_entry.orders", Application: "order_entry"}}
    pattern := &world.Applications[0].Patterns[0]
    pattern.Data = "sales.order"
    pattern.Relation = "sales.order.lines"
    pattern.Capability = "sales.order.submit"
    pattern.Query = "order_entry.orders"
    refs := []struct {
        subject sourceindex.Subject
        path string
    }{
        {sourceindex.Subject{Kind:"semantic-data",ID:"sales.order"}, "semantic/sales/order.tei"},
        {sourceindex.Subject{Kind:"semantic-relation",ID:"sales.order.lines"}, "semantic/sales/relation.tei"},
        {sourceindex.Subject{Kind:"semantic-capability",ID:"sales.order.submit"}, "semantic/sales/capability.tei"},
        {sourceindex.Subject{Kind:"query",ID:"order_entry.orders"}, "applications/order-entry/queries/orders.tei"},
    }
    for _, r := range refs {
        // A Namespace and its Data can be declared in the same physical file.
        // The Index stores one Artifact per path but multiple Subject references.
        found := false
        for _, artifact := range index.Artifacts {
            if artifact.Path == r.path {
                found = true
                break
            }
        }
        if !found {
            index.Artifacts = append(index.Artifacts,
                sourceindex.Artifact{Path:r.path,Fingerprint:"sha256:abababababababababababababababababababababababababababababababab"})
        }
        index.References = append(index.References,
            sourceindex.Reference{Subject:r.subject,ArtifactPath:r.path,Role:sourceindex.RoleDefinition})
    }
    return world,index
}

func TestE5FollowsTypedPatternTargets(t *testing.T) {
    world, index := e5PatternTargetsFixture()
    report := inspectE5(world,index,"order_entry")
    if report.Status!="observed" || len(report.Dependencies)!=10 {
        t.Fatalf("expected six direct plus four Pattern target definitions: %+v",report)
    }
    seen := map[string]bool{}
    for _, dep := range report.Dependencies {
        seen[dep.Relation+":"+dep.Subject.ID] = true
    }
    for _, claim := range []string{
        "pattern-data:sales.order", "pattern-relation:sales.order.lines",
        "pattern-capability:sales.order.submit", "pattern-query:order_entry.orders",
    } {
        if !seen[claim] {t.Fatalf("missing Pattern target dependency: %s; %+v",claim,report)}
    }
}

func TestE5MissingPatternCapabilityFailsClosed(t *testing.T) {
    world,index := e5PatternTargetsFixture()
    world.Semantics[0].Capabilities=nil
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"pattern_target_missing_or_ambiguous") {
        t.Fatal("missing Semantic Capability accepted")
    }
}

func TestE5PatternQueryCannotCrossApplication(t *testing.T) {
    world,index := e5PatternTargetsFixture()
    world.Queries[0].Application="another_app"
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"pattern_query_wrong_application") {
        t.Fatal("foreign application query accepted")
    }
}

func TestE5PatternReferentMissingIndexDefinitionFailsClosed(t *testing.T) {
    world,index := e5PatternTargetsFixture()
    // The last reference is the Pattern's Query definition.
    index.References=index.References[:len(index.References)-1]
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"source_definition_missing_or_ambiguous") {
        t.Fatal("missing Query Source definition accepted")
    }
}

func TestE5ReusedPatternTargetEmitsOneSubject(t *testing.T) {
    world,index := e5PatternTargetsFixture()
    world.Applications[0].Patterns = append(world.Applications[0].Patterns,
        application.Pattern{ID:"order_entry.second", Kind:application.PatternReference,
            Relation:"sales.order.lines"})
    index.Artifacts=append(index.Artifacts,sourceindex.Artifact{
        Path:"applications/order-entry/patterns/second.tei",
        Fingerprint:"sha256:cdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd",
    })
    index.References=append(index.References,sourceindex.Reference{
        Subject:sourceindex.Subject{Kind:"application-pattern",ID:"order_entry.second"},
        ArtifactPath:"applications/order-entry/patterns/second.tei",Role:sourceindex.RoleDefinition,
    })
    report:=inspectE5(world,index,"order_entry")
    if report.Status!="observed" || len(report.Dependencies)!=11 {
        t.Fatalf("reused relation must be observed once: %+v",report)
    }
    count:=0
    for _, dep:=range report.Dependencies {
        if dep.Relation=="pattern-relation" && dep.Subject.ID=="sales.order.lines" {count++}
    }
    if count!=1 {t.Fatalf("relation source duplicated: %+v",report.Dependencies)}
}

func TestE5DuplicatePatternTargetIdentityFailsClosed(t *testing.T) {
    world,index := e5PatternTargetsFixture()
    world.Semantics[0].Data[0].Relations = append(world.Semantics[0].Data[0].Relations,
        world.Semantics[0].Data[0].Relations[0])
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"pattern_target_missing_or_ambiguous") {
        t.Fatal("duplicate Semantic Relation accepted")
    }
}

func e5ContractQueryFixture() (ir.World, sourceindex.Index) {
    world, index := e5Fixture()
    world.Applications[0].Patterns = append(world.Applications[0].Patterns, application.Pattern{
        ID: "order_entry.customer_reference",
        Kind: application.PatternReference,
        Datasource: "sales",
        ContractQuery: &application.ContractQueryInvocation{
            Contract: "sales_core.order",
            Query: "find_by_id",
        },
    })
    world.Databases[0].Contracts[0].Queries = []database.QueryContract{{Name:"find_by_id"}}
    index.Artifacts = append(index.Artifacts, sourceindex.Artifact{
        Path:"applications/order-entry/patterns/customer-reference.tei",
        Fingerprint:"sha256:abababababababababababababababababababababababababababababababab",
    })
    index.References = append(index.References, sourceindex.Reference{
        Subject:sourceindex.Subject{Kind:"application-pattern",ID:"order_entry.customer_reference"},
        ArtifactPath:"applications/order-entry/patterns/customer-reference.tei",
        Role:sourceindex.RoleDefinition,
    },sourceindex.Reference{
        Subject:sourceindex.Subject{Kind:"database-contract-query",ID:"sales_core.order/find_by_id"},
        ArtifactPath:"databases/sales-core/contracts/order.tei",
        Role:sourceindex.RoleDefinition,
    })
    return world,index
}

func TestE5PatternContractQueryDirectReference(t *testing.T) {
    world,index := e5ContractQueryFixture()
    report := inspectE5(world,index,"order_entry")
    if report.Status != "observed" {
        t.Fatalf("typed ContractQuery not observed: %+v", report.Issues)
    }
    found := 0
    for _, dep := range report.Dependencies {
        if dep.Relation == "pattern-contract-query" && dep.Subject.Kind == "database-contract-query" &&
            dep.Subject.ID == "sales_core.order/find_by_id" &&
            dep.Source.Path == "databases/sales-core/contracts/order.tei" { found++ }
    }
    if found != 1 {t.Fatalf("expected exact ContractQuery definition once: %+v",report.Dependencies)}
}

func TestE5PatternContractQueryMissingDatasource(t *testing.T) {
    world,index := e5ContractQueryFixture()
    world.Applications[0].Patterns[1].Datasource = "undeclared"
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"contract_query_datasource_missing_or_ambiguous") {
        t.Fatal("unknown datasource alias accepted")
    }
}

func TestE5PatternContractQueryWrongRequiredContract(t *testing.T) {
    world,index := e5ContractQueryFixture()
    world.Applications[0].Databases[0].RequiredContracts = nil
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"contract_query_contract_unavailable") {
        t.Fatal("unrequired ContractQuery contract accepted")
    }
}

func TestE5PatternContractQueryMissingOrDuplicateQuery(t *testing.T) {
    world,index := e5ContractQueryFixture()
    world.Databases[0].Contracts[0].Queries = nil
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"contract_query_definition_missing_or_ambiguous") {
        t.Fatal("missing contract Query accepted")
    }
    world,index = e5ContractQueryFixture()
    world.Databases[0].Contracts[0].Queries = append(world.Databases[0].Contracts[0].Queries,
        world.Databases[0].Contracts[0].Queries[0])
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"contract_query_definition_missing_or_ambiguous") {
        t.Fatal("duplicate contract Query accepted")
    }
}

func TestE5PatternContractQueryMissingSourceDefinition(t *testing.T) {
    world,index := e5ContractQueryFixture()
    for i, ref := range index.References {
        if ref.Subject == (sourceindex.Subject{Kind:"database-contract-query",ID:"sales_core.order/find_by_id"}) {
            index.References = append(index.References[:i],index.References[i+1:]...)
            break
        }
    }
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"source_definition_missing_or_ambiguous") {
        t.Fatal("missing ContractQuery source reference accepted")
    }
}

func TestE5PatternContractQueryWrongDatabaseOwner(t *testing.T) {
    world,index := e5ContractQueryFixture()
    world.Databases[0].Contracts = nil
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"contract_query_contract_unavailable") {
        t.Fatal("ContractQuery not in chosen Database accepted")
    }
}

func TestE5SharedArtifactSupportsMultipleDefinitionSubjects(t *testing.T) {
    world,index := e5PatternTargetsFixture()
    path := "semantic/sales/order.tei"
    artifacts, definitions := 0, 0
    for _, artifact := range index.Artifacts {
        if artifact.Path == path { artifacts++ }
    }
    for _, ref := range index.References {
        if ref.ArtifactPath == path && ref.Role == sourceindex.RoleDefinition {
            definitions++
        }
    }
    if artifacts != 1 || definitions != 2 {
        t.Fatalf("one Artifact with Namespace and Data Source references expected; artifacts=%d refs=%d",
            artifacts, definitions)
    }
    report := inspectE5(world,index,"order_entry")
    if report.Status != "observed" {
        t.Fatalf("valid multi-subject Source must resolve: %+v", report)
    }
    names := map[string]bool{}
    for _, dep := range report.Dependencies {
        if dep.Source.Path == path { names[dep.Subject.Kind] = true }
    }
    if !names["semantic"] || !names["semantic-data"] {
        t.Fatalf("one physical Source must retain two semantic Subject identities: %+v",report.Dependencies)
    }
}

func TestE5DuplicateArtifactPathRemainsUnresolved(t *testing.T) {
    world,index := e5PatternTargetsFixture()
    for _, artifact := range index.Artifacts {
        if artifact.Path == "semantic/sales/order.tei" {
            index.Artifacts = append(index.Artifacts,artifact)
            break
        }
    }
    report := inspectE5(world,index,"order_entry")
    if report.Status != "unresolved" ||
        !e5HasIssue(report,"source_definition_missing_or_ambiguous") {
        t.Fatalf("duplicate Artifact path accepted as unique evidence: %+v",report)
    }
}

func TestE5OrderEntryWorkspaceContractQueryReference(t *testing.T) {
    // This is an actual checked-in Workspace consumer, not the synthetic E5
    // fixture. It must not be counted as PASS until run on a trusted checkout.
    root := filepath.Join("..", "..", "reference", "workspace")
    loaded, err := workspace.Load(root)
    if err != nil { t.Fatalf("load reference Workspace: %v", err) }
    index, err := workspace.BuildSourceIndex(root, loaded)
    if err != nil { t.Fatalf("index reference Workspace: %v", err) }
    report := inspectE5(loaded.World, index, "order_entry")
    if report.Status != "observed" {
        t.Fatalf("actual order_entry direct references unresolved: %+v",report.Issues)
    }
    found, inputSource := 0, 0
    for _, dep := range report.Dependencies {
        if dep.Relation == "pattern-contract-query" &&
            dep.Subject.Kind == "database-contract-query" &&
            dep.Subject.ID == "master_data.customer/find_by_id" &&
            dep.Source.Path == "databases/master-data/contracts/customer.tei" {
            found++
        }
        if dep.Relation == "pattern-contract-query-input-attribute" &&
            dep.Subject.Kind == "semantic-attribute" &&
            dep.Subject.ID == "customer.customer.customer_id" &&
            dep.Source.Path == "semantic/customer/customer.tei" {
            inputSource++
        }
    }
    if found != 1 || inputSource != 1 {
        t.Fatalf("actual ContractQuery definition=%d, input attribute=%d dependencies=%+v",found,inputSource,report.Dependencies)
    }
}

func TestE5SUIStage0IsNotAnApplicationConsumer(t *testing.T) {
    root := filepath.Join("..", "..", "reference", "sui-adoption", "workspace")
    loaded, err := workspace.Load(root)
    if err != nil { t.Fatalf("load SUI Stage 0 Workspace: %v", err) }
    index, err := workspace.BuildSourceIndex(root, loaded)
    if err != nil { t.Fatalf("index SUI Stage 0 Workspace: %v", err) }
    report := inspectE5(loaded.World,index,"sui_reference_adoption_stage0")
    if report.Status != "unresolved" || !e5HasIssue(report,"application_missing_or_ambiguous") {
        t.Fatalf("SUI Stage 0 was incorrectly promoted to Application: %+v",report)
    }
}

func e5ContractQueryAttributeFixture() (ir.World, sourceindex.Index) {
    world, index := e5ContractQueryFixture()
    world.Semantics[0].Data = []semantic.Data{{
        ID:"sales.customer",
        Attributes:[]semantic.Attribute{{ID:"sales.customer.customer_id"}},
    }}
    world.Applications[0].Patterns[1].ContractQuery.Inputs = []application.ContractQueryInputBinding{{
        TargetAttribute:"sales.customer.customer_id",Input:"id",
    }}
    index.Artifacts=append(index.Artifacts,sourceindex.Artifact{
        Path:"semantic/sales/customer.tei",
        Fingerprint:"sha256:cdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcdcd",
    })
    index.References=append(index.References,sourceindex.Reference{
        Subject:sourceindex.Subject{Kind:"semantic-attribute",ID:"sales.customer.customer_id"},
        ArtifactPath:"semantic/sales/customer.tei",Role:sourceindex.RoleDefinition,
    })
    return world,index
}

func TestE5ContractQueryInputAttributeSource(t *testing.T) {
    world,index:=e5ContractQueryAttributeFixture()
    report:=inspectE5(world,index,"order_entry")
    if report.Status!="observed" {t.Fatalf("unexpected source problem: %+v",report.Issues)}
    count:=0
    for _,dep:=range report.Dependencies {
        if dep.Relation=="pattern-contract-query-input-attribute" &&
            dep.Subject.Kind=="semantic-attribute" &&
            dep.Subject.ID=="sales.customer.customer_id" &&
            dep.Source.Path=="semantic/sales/customer.tei" {count++}
    }
    if count!=1 {t.Fatalf("input attribute source not observed once: %+v",report.Dependencies)}
}

func TestE5ContractQueryInputAttributeNotInWorld(t *testing.T) {
    world,index:=e5ContractQueryAttributeFixture()
    world.Semantics[0].Data=nil
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"contract_query_input_attribute_missing_or_ambiguous") {
        t.Fatal("undefined semantic input attribute accepted")
    }
}

func TestE5ContractQueryInputAttributeAmbiguous(t *testing.T) {
    world,index:=e5ContractQueryAttributeFixture()
    world.Semantics[0].Data[0].Attributes=append(world.Semantics[0].Data[0].Attributes,
        world.Semantics[0].Data[0].Attributes[0])
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"contract_query_input_attribute_missing_or_ambiguous") {
        t.Fatal("ambiguous semantic input attribute accepted")
    }
}

func TestE5ContractQueryInputAttributeMissingSource(t *testing.T) {
    world,index:=e5ContractQueryAttributeFixture()
    for i,ref:=range index.References {
        if ref.Subject.Kind=="semantic-attribute" && ref.Subject.ID=="sales.customer.customer_id" {
            index.References=append(index.References[:i],index.References[i+1:]...)
            break
        }
    }
    if !e5HasIssue(inspectE5(world,index,"order_entry"),"source_definition_missing_or_ambiguous") {
        t.Fatal("semantic input attribute without Source Index definition accepted")
    }
}

func TestE5ContractQueryInputAttributeNotPromotedWithoutQuery(t *testing.T) {
    world,index:=e5ContractQueryAttributeFixture()
    world.Databases[0].Contracts[0].Queries=nil
    report:=inspectE5(world,index,"order_entry")
    if !e5HasIssue(report,"contract_query_definition_missing_or_ambiguous") {
        t.Fatal("missing Query accepted")
    }
    for _,dep:=range report.Dependencies {
        if dep.Relation=="pattern-contract-query-input-attribute" {
            t.Fatalf("unresolved Query promoted its input source: %+v",report.Dependencies)
        }
    }
}
