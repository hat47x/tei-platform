# Second Application Reference — S5 Executable Pressure Implementation Packet

- Status: Branchless executable-pressure candidate negative-controls pinned / focused and health execution pending
- Date: 2026-09-20
- Primary workstream: ISSUE-000040
- Proposal SHA-256: `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`
- Source-inspection base: `c185937ab27a3a31332fefcd63ba5940b97b021e`
- Authoritative destination source created: No
- Executable Evidence obtained: No
- Shared Acceptance established: No

## 1. 目的

revised three-source Proposalを、次のhealth-capable cloneで迷いなくactual loader / full Validationへ通すための実装契約を固定する。

このpacketはコード実装やtest PASSそのものではない。

```text
implementation contract pinned
  != source changed
  != executable pressure passed
  != accepted
```

## 1.1 Branchless code candidate

current `main` を変更せず、health-capable laneで検証するためのunreferenced Git commitを作成した。

```text
candidate commit:
  8c570c746f78a3346348d3c9bb1f6a3bd6e3debb

parent:
  d71fa6305592ea4a0ae0b98a880808711ebac640

tree:
  1edb6f400fce178dd1ac67e1a8816c1680e65829

supersedes positive-only candidate:
  f30cab65b98505c6b5d71b2c08aa14483e72a54f

prior byte-identical rebases:
  9f0dd71dbd181e4798f7b8efefb09759bdcc256e
  eda2e367bc1a0f12d83a41fc55e253b96bf76e9d

supersedes unformatted candidate:
  e007c67b478feed958cd84ae0b040251a1dc8225

remote branch / PR:
  none
```

candidateの変更pathはexactly次の6件である。

```text
tools/reference-business-procedure-check/second_application_s3_destination_decision_pressure_test.go
tools/reference-business-procedure-check/second_application_s4_composition_candidate_pressure_test.go
tools/reference-business-procedure-check/second_application_s4_full_validation_pressure_test.go
tools/reference-business-procedure-check/second_application_s5_freshness_capture_pressure_test.go
tools/reference-business-procedure-check/second_application_s5_shared_acceptance_review_pressure_test.go
tools/run-knowledge-portability-reference-evidence.sh
```

このcandidateでは:

- S3 destination Semantic owner / attribute / Relation key decisionをmachine-readable化した
- S4をSemantic + Application + Patternのthree-source Proposalへ更新した
- temporary Reference Workspaceをactual `workspace.Load` + `validation.Validate`へ通すpressureを追加した
- S5 fingerprintを `c18ae0f...` へ更新した
- S5 freshness base constantをinspected revision `c185937a...` へ更新した
- reference-evidence runnerのgofmt集合をactual Second Application test集合へ追随させた

Git diff inspectionではcurrent `main` に既存 `repair_intake` source pathがなく、candidateは上記6 path以外を変更していないことを確認した。

first candidate `e007c67b...` をローカルへ再現してactual `gofmt -d` を実行したところ、S3 field alignmentとfull-validation testのmap alignmentにdriftを検出した。このdriftを修正し、`eda2e367...` として再構成した。その後、並行R7Bレーンによる計画文書commitだけで`main`が進んだため、6 source blobを一切変更せずcurrent `main` `b12962ee...` 上へ積み直し、`9f0dd71d...` をcurrent branchless candidateとした。

修正版candidateの5 Go filesについて、ローカルで `gofmt -d` が全件差分ゼロであることを確認し、さらにlocal formatted fileのGit blob SHAとcandidate内blob SHAをexact matchさせた。`f30cab65...` までは6 filesのblob SHAが従来候補と全件同一だった。current candidate `8c570c74...` ではfull-validation testだけを意図的に強化し、残る5 filesのblob SHAは従来候補と同一のまま維持した。強化後full-validation testはlocal `gofmt -d` 差分ゼロで、local `git hash-object` とcandidate blobが `0f1e69c1a62077ef31c6f599f567286209c4f0d5` に一致した。

```text
S3 pressure:
  3e840e24e8c3418b8aa601e2fa3a3407d57b4197

S4 composition:
  98ebd58b988cfb5118b3ef81a70dd2ddb1ed150b

S4 full validation:
  67f9045a8ca7628319dceb70944c2b75688d7319

S5 freshness:
  70de17b0c5901c378fcd51d181c14432a8692367

S5 review:
  e2b6dde5749cb525628d457c3060227ccca5bcd3
```

したがってcurrent Evidenceは:

```text
candidate Go gofmt execution     = PASS
candidate Go blob identity match  = PASS
negative-control source pressure  = pinned
candidate runner bash -n          = PASS
candidate runner blob match       = PASS
current API/static source audit   = complete
focused go test                  = not obtained
reference evidence execution     = not obtained
repository health check          = not obtained
Executable Evidence              = absent
```

である。gofmt PASSだけをexecutable-pressure PASSへ昇格しない。health-capable laneはcurrent main driftを先に確認し、必要ならcandidate内容をlatest mainへrebase/reconstructしたうえでfocused test / reference evidence / binding health checkを実行する。


### Static compile-risk / runner preflight

6件目の変更pathである `tools/run-knowledge-portability-reference-evidence.sh` もcandidateからローカルへexactに再現し、actual `bash -n` を実行した。

```text
runner syntax:
  PASS

candidate runner blob:
  42db7c4e789c36253b7ae995518eaef3c738b242

local reproduced runner blob:
  42db7c4e789c36253b7ae995518eaef3c738b242
```

またcurrent `main` の実装を直接照合し、candidateが参照するcompile surfaceを確認した。

- `semantic.Relation` に `ID` / `Target` / `Cardinality` / `Keys` が存在する
- `semantic.RelationKey` に `SourceAttribute` / `TargetAttribute` が存在する
- `semantic.CardinalityOne` が存在する
- `validation.Diagnostic` / `validation.Error` / `validation.Validate(ir.World)` が存在する
- same-package helper `copyDirectory(t, sourceRoot, targetRoot)` が存在する
- `ValidateReferenceContractQueryCrossLayer` はRelation key、typed input、result Data、cardinalityをcandidate設計どおり検査する
- `ValidatePersistenceRelationStorageMappings` は存在するPersistence Mappingだけを検査し、read-only destination SemanticにPersistence Mappingが存在しないこと自体をerrorにしない

したがって、現時点のsource inspectionではcandidateに新しいcompile/API blockerや「先にPersistenceを作る必要」は見つかっていない。

これはstatic/source Evidenceであり、`go test` / `workspace.Load` / `validation.Validate` のactual execution結果ではない。

## 2. Current main drift review

実際にrelevant sourceを再inspectionしたbaseは:

```text
c185937ab27a3a31332fefcd63ba5940b97b021e
docs(r7b): repair convergence audit and verify exact runner syntax
```

である。

その後mainは `b12962eebe75eea907241cf6cf2b831da7e99474` まで進んだ。`c185937a...` からcurrent mainまでの変更pathは、Second Applicationの計画文書4件と `plan/issue/000069-first-web-target-r7b-exact-t-replay.md` に限定されている。Second Application Proposal component、Customer shared authority、Application Pattern validation、reference Contract Query cross validation、Workspace loader、candidate対象test/scriptは不変なので、計画文書commitだけを理由にsource-inspection baseを動かさない。

## 2.1 Hosted runner availability

`.github/workflows/knowledge-portability-reference-evidence.yml` 自体を再確認した。

Repository sourceには2026-09-19の運用記録として:

```text
GitHub-hosted runner allocation is currently unavailable
PR run 35385856132 failed before any step was created
runner_id = 0
workflow remains manual-only until successful allocation is observed
```

と明記されている。

2026-09-20のActions履歴でもcurrent `main` に対応する利用可能なrunはなく、直近のKnowledge portability runsは2026-09-18のfailureである。したがってGitHub Actionsのfailureをcandidate test failureとして扱わず、またActionsをbinding health Evidenceの代替にもしない。

current execution gateは引き続き:

```text
Go-enabled full checkout
  -> focused TestSecondApplication...
  -> reference-evidence runner
  -> tools/repository-health-check.sh
```

である。

current shared/provenance blob identity:

```text
reference/workspace/semantic/customer/customer.tei
  c348122b33581e450974d23393d98fb8aa838388

reference/workspace/databases/master-data/contracts/customer.tei
  c1eea63b053fb936ae7aebbac82180af015b42e1

reference/workspace/applications/order-entry/patterns/customer-reference.tei
  252f6396cbc2394d1f26e748b01d54f49a2e1ecf
```

関連implementation blob identity:

```text
internal/validation/validation.go
  cff6c9f95c5829a6ea326a41b00c6f93c78f05a5

internal/validation/application_pattern.go
  e137189a5945efaec0c14d82f92c62ba98fa16fd

internal/validation/reference_contract_query_cross.go
  8c2a4b5bd89baf3c4562d62bfed009ce3094bf40

internal/workspace/loader.go
  948c3558b0f8ed8dd48d4a108ab2da6bffb379b0

internal/syntax/parser.go
  94b73b2d8cecc6d0e6cfe4e8ae4c1a3696a8d079
```

このrecheckはfreshness / implementation source inspectionであり、Acceptance witnessではない。

## 3. Exact code delta

### S3 destination decision pressure

`second_application_s3_destination_decision_pressure_test.go` へ、少なくとも次のdestination-owned decisionをmachine-readableに追加する。

```text
semantic namespace = repair_intake
semantic data = repair_intake.intake
customer id attribute = repair_intake.intake.customer_id
attribute type = customer.customer_id
relation = repair_intake.intake.customer
relation target = customer.customer
relation cardinality = one
relation key source = repair_intake.intake.customer_id
relation key target = customer.customer.customer_id
```

これらはorigin `sales.order.customer` からのcopyではなく、S3 destination decisionとしてassertする。

実装時のfield名は少なくとも次へ固定する。

```text
SemanticNamespaceCandidate
SemanticDataCandidate
CustomerIDAttributeCandidate
CustomerIDAttributeType
BusinessRelationTarget
BusinessRelationCardinality
BusinessRelationKeySource
BusinessRelationKeyTarget
```

既存の `BusinessRelationCandidate` はRelation identityとして維持し、target/cardinality/keyを別fieldで明示する。

### Compile-ready S3 shape

struct追加は次の形へ固定する。

```go
SemanticNamespaceCandidate   string
SemanticDataCandidate        string
CustomerIDAttributeCandidate string
CustomerIDAttributeType      string
BusinessRelationTarget       string
BusinessRelationCardinality  string
BusinessRelationKeySource    string
BusinessRelationKeyTarget    string
```

`secondApplicationS3Decision()` では:

```go
SemanticNamespaceCandidate:   "repair_intake",
SemanticDataCandidate:        "repair_intake.intake",
CustomerIDAttributeCandidate: "repair_intake.intake.customer_id",
CustomerIDAttributeType:      "customer.customer_id",
BusinessRelationTarget:       "customer.customer",
BusinessRelationCardinality:  "one",
BusinessRelationKeySource:    "repair_intake.intake.customer_id",
BusinessRelationKeyTarget:    "customer.customer.customer_id",
```

を返す。testではこれらを一括assertし、origin側 `sales.order.*` identityが混入しないことも確認する。

### S4 three-source pressure

`secondApplicationS4CompositionCandidate`へ`SemanticSource`を追加する。

Semantic candidateはexact revised Proposalと同じ:

```text
namespace repair_intake

data Intake {
    attribute customerId type customer.customer_id
    relation customer to customer.customer one
        key repair_intake.intake.customer_id -> customer.customer.customer_id
}
```

parser pressureは既存公開入口を使う。

```go
syntax.NewParser(got.SemanticSource).ParseSemanticFile()
syntax.NewParser(got.ApplicationSource).ParseApplication()
syntax.NewParser(got.PatternSource).ParseApplicationPattern()
```

Applicationは`uses semantic repair_intake`と`uses semantic customer`の2件を要求する。

Semantic parse resultではNamespace / Data / Attribute / Relation / keyのCanonical identityをassertする。

### Compile-ready S4 parser assertions

`secondApplicationS4CompositionCandidate` へ:

```go
SemanticSource string
```

を追加する。Application sourceは:

```text
application repair_intake {
    uses semantic repair_intake
    uses semantic customer

    database customer_master {
        target master_data
        access read
        requires {
            contract master_data.customer ^2
        }
    }
}
```

とする。

Semantic parse assertionはcurrent model shapeに合わせて少なくとも次を要求する。

```go
ns, err := syntax.NewParser(got.SemanticSource).ParseSemanticFile()
if err != nil {
    t.Fatalf("S4 Semantic candidate did not parse with existing grammar: %v", err)
}
if ns.ID != "repair_intake" || len(ns.Data) != 1 {
    t.Fatalf("destination Semantic drifted: %+v", ns)
}
data := ns.Data[0]
if data.ID != "repair_intake.intake" ||
    len(data.Attributes) != 1 ||
    data.Attributes[0].ID != "repair_intake.intake.customer_id" ||
    data.Attributes[0].TypeRef != "customer.customer_id" {
    t.Fatalf("destination Data/attribute drifted: %+v", data)
}
if len(data.Relations) != 1 {
    t.Fatalf("destination Relation count drifted: %+v", data.Relations)
}
relation := data.Relations[0]
if relation.ID != "repair_intake.intake.customer" ||
    relation.Target != "customer.customer" ||
    relation.Cardinality != semantic.CardinalityOne ||
    len(relation.Keys) != 1 ||
    relation.Keys[0].SourceAttribute != "repair_intake.intake.customer_id" ||
    relation.Keys[0].TargetAttribute != "customer.customer.customer_id" {
    t.Fatalf("destination Relation/key drifted: %+v", relation)
}
```

このassertionのためS4 test importへ `tei/internal/semantic` を追加する。

Applicationの `SemanticUses` はexact source orderに従い:

```go
len(app.SemanticUses) == 2
app.SemanticUses[0] == "repair_intake"
app.SemanticUses[1] == "customer"
```

を要求する。

## 4. Actual temporary Workspace pressure

parser-only successではなく、Repositoryのactual Reference Workspaceをbaselineにする。

既存`accepted_source_pressure_test.go`のhelperを再利用する。

```text
root := repoPath()
sourceRoot := <root>/reference/workspace
probeRoot := <t.TempDir()>/workspace
copyDirectory(t, sourceRoot, probeRoot)
```

probeRootにだけ次を作成する。

```text
semantic/repair-intake/intake.tei
applications/repair-intake/application.tei
applications/repair-intake/patterns/customer-reference.tei
```

Repositoryのauthoritative `reference/workspace`へはこの段階で書かない。

その後:

```go
loaded, err := workspace.Load(probeRoot)
diagnostics := validation.Validate(loaded.World)
```

を実行する。

current sourceでは `validation.Validate()` から:

```text
Validate
  -> validateQueryResultMappings
  -> validateApplicationDataAccessContracts
     -> ValidateSemanticRelationKeys
     -> ValidatePersistenceRelationStorageMappings
     -> ValidateDatabaseContractQuerySignatures
     -> ValidateReferenceContractQueryInvocations
     -> ValidateReferenceContractQueryCrossLayer
     -> ValidateHeaderDetailContractQueryBindings
     -> ValidateContractQuerySQLRealizations
```

へ到達することを確認した。したがって、このtestでRelation keyとContract Query cross-layerを含む通常のdata-access Validation経路を通せる。

Acceptance prerequisiteとしては、少なくとも`validation.Error`が0件であることを要求する。特に次をsilent skipしない。

- `UNKNOWN_APPLICATION_PATTERN_RELATION`
- Relation key source / target不整合
- Relation key type mismatch
- Contract Query input binding mismatch
- Contract Query result Data mismatch
- Contract Query result cardinality mismatch

### Full Validation negative controls

positive Proposalが`validation.Error == 0`となることだけでは、将来validator経路がsilent skipされてもtestが偽陽性になる余地がある。そこでcurrent branchless candidateでは、authoritative sourceを変更せずtemporary candidateだけを壊すnegative controlを2件追加した。

1. Pattern Relationを `repair_intake.intake.missing` へ置換し、通常Validationが `UNKNOWN_APPLICATION_PATTERN_RELATION` を返すことを要求する。
2. destination Relation cardinalityを `one` から `many` へ置換し、shared Query `find_by_id -> zero-or-one` との横断検査が `REFERENCE_CONTRACT_QUERY_RESULT_CARDINALITY_MISMATCH` を返すことを要求する。

置換対象文字列が見つからない場合もtest setup failureとして即座にfailする。これによりnegative control自体のsilent no-opも許さない。

```text
positive candidate validation:
  expected error diagnostics = 0

negative unknown Relation:
  expected diagnostic code = UNKNOWN_APPLICATION_PATTERN_RELATION

negative cardinality mismatch:
  expected diagnostic code = REFERENCE_CONTRACT_QUERY_RESULT_CARDINALITY_MISMATCH
```

これらはactual `go test` 未実行のため、現時点ではtest design / static source Evidenceである。

## 5. S5 exact Proposal / freshness pressure

### Shared Acceptance review pressure

`second_application_s5_shared_acceptance_review_pressure_test.go` は旧two-source payloadを廃止し、exact payloadを:

```text
SemanticSource
+ "\n---\n"
+ ApplicationSource
+ "\n---\n"
+ PatternSource
```

とする。

expected fingerprint:

```text
c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d
```

NewDestinationSubjectsは少なくとも:

```text
semantic:repair_intake
semantic-data:repair_intake.intake
semantic-attribute:repair_intake.intake.customer_id
semantic-relation:repair_intake.intake.customer
application:repair_intake
application-pattern:repair_intake.customer_reference
datasource-alias:repair_intake/customer_master
```

を含める。

Runtime authority、Environment mutation、source authoringはfalseのまま維持する。

### Compile-ready S5 delta

fingerprint constantを:

```go
const secondApplicationS5ExpectedProposalFingerprint =
    "c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d"
```

へ更新する。

payloadは:

```go
payload := s4.SemanticSource + "\n---\n" + s4.ApplicationSource + "\n---\n" + s4.PatternSource
```

とする。

`NewDestinationSubjects` には既存4件に加え:

```text
semantic:repair_intake
semantic-data:repair_intake.intake
semantic-attribute:repair_intake.intake.customer_id
```

を追加する。Relation subjectは既存 `semantic-relation:repair_intake.intake.customer` を維持する。

### Freshness pressure

`second_application_s5_freshness_capture_pressure_test.go` には現在:

```text
secondApplicationS5FreshnessBaseRevision =
  6ffaa40ef2872a773baa5e7b127dcb89d94abc5c
```

が残っており、revised S5 documentのreview baseと一致していない。これは今回のsource inspectionで新たに確認したstale executable fieldである。

health-capable code milestoneでは、このconstantをcurrent inspected relevant-source base:

```text
c185937ab27a3a31332fefcd63ba5940b97b021e
```

へ更新する。

同時に `TestSecondApplicationS5FreshnessPreservesExactProposalIdentity` がrevised three-source fingerprint `c18ae0f...` を保持することを確認する。freshness base更新はAcceptance witnessを生成しない。

## 6. Full Validation test placement

temporary workspaceのactual load / Validation pressureは既存parser testへ過剰に詰め込まず、新しいtest file:

```text
tools/reference-business-procedure-check/
  second_application_s4_full_validation_pressure_test.go
```

へ分離する。

このfileでは既存package-level helper `repoPath()` と `copyDirectory()` を再利用し、Repository sourceへの書込みを避ける。test名は `TestSecondApplicationS4ThreeSourceProposalPassesFullWorkspaceValidation` とし、`workspace.Load` errorと `validation.Error` diagnosticsをblocking failureとして扱う。

### Compile-ready full-workspace test

必要importは:

```go
import (
    "os"
    "path/filepath"
    "testing"

    "tei/internal/validation"
    "tei/internal/workspace"
)
```

test bodyは次の責務へ固定する。

```go
func TestSecondApplicationS4ThreeSourceProposalPassesFullWorkspaceValidation(t *testing.T) {
    root := repoPath()
    probeRoot := filepath.Join(t.TempDir(), "workspace")
    copyDirectory(t, filepath.Join(root, "reference", "workspace"), probeRoot)

    candidate := secondApplicationS4Composition(secondApplicationS3Decision())
    sources := map[string]string{
        filepath.Join("semantic", "repair-intake", "intake.tei"): candidate.SemanticSource,
        filepath.Join("applications", "repair-intake", "application.tei"): candidate.ApplicationSource,
        filepath.Join("applications", "repair-intake", "patterns", "customer-reference.tei"): candidate.PatternSource,
    }
    for relative, source := range sources {
        path := filepath.Join(probeRoot, relative)
        if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
            t.Fatal(err)
        }
        if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
            t.Fatal(err)
        }
    }

    loaded, err := workspace.Load(probeRoot)
    if err != nil {
        t.Fatal(err)
    }
    diagnostics := validation.Validate(loaded.World)
    var errors []validation.Diagnostic
    for _, diagnostic := range diagnostics {
        if diagnostic.Severity == validation.Error {
            errors = append(errors, diagnostic)
        }
    }
    if len(errors) != 0 {
        t.Fatalf("revised Second Application Proposal failed full Validation: %#v", errors)
    }
}
```

WarningはこのAcceptance prerequisiteをblockしないが、Errorは1件でもblockする。

## 7. Reference evidence runner追随

current `tools/run-knowledge-portability-reference-evidence.sh` のtest regexは`TestSecondApplication...`を実行するが、gofmt対象filesにはS3/S4/S5 shared-acceptance pressureが含まれていない。

code milestoneでは同scriptの`files=(...)`へ次を追加する。

```text
tools/reference-business-procedure-check/second_application_s3_destination_decision_pressure_test.go
tools/reference-business-procedure-check/second_application_s4_composition_candidate_pressure_test.go
tools/reference-business-procedure-check/second_application_s5_shared_acceptance_review_pressure_test.go
tools/reference-business-procedure-check/second_application_s4_full_validation_pressure_test.go
```

gofmt pressureとactual executed test集合を一致させる。

## 8. Required execution sequence

health-capable cloneでは次の順を固定する。

```text
pull --rebase current main
  -> relevant source drift recheck
  -> S3/S4/S5 review + S5 freshness pressure + evidence runner update
  -> gofmt
  -> focused TestSecondApplication...
  -> tools/run-knowledge-portability-reference-evidence.sh
  -> tools/repository-health-check.sh
  -> only after PASS: direct-to-main commit/push
```

mainが途中で進んだ場合はforce pushせず、rebase / relevant drift reviewをやり直す。

## 9. Evidence and Acceptance boundary

このpacketをmainへ置いても:

- revised executable pressure = absent
- Shared Acceptance = absent
- S6 authoritative authoring = blocked
- Runtime authority = absent

のままである。

health-check PASS後もexplicit Acceptance witnessは独立条件として残す。一般的な作業継続指示をfingerprint `c18ae0f...` のAcceptanceへ読み替えない。
