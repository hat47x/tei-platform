# Second Application Reference — S5 Executable Pressure Implementation Packet

- Status: Implementation contract refined / current freshness rechecked / actual execution pending
- Date: 2026-09-20
- Primary workstream: ISSUE-000040
- Proposal SHA-256: `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`
- Source-inspection base: `e1f13ae97b420a2703eec8598eb98c0ab057258b`
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

## 2. Current main drift review

前回の実装契約commit `64f0a5b8167fcb4aa90e56a6c47d7158bba336b8` 以後、inspection時点のmainは:

```text
e1f13ae97b420a2703eec8598eb98c0ab057258b
docs(r7b): pin branchless integration packet
```

である。

`64f0a5b8...` から `e1f13ae9...` までの12 commitをcompareし、変更pathは `plan/issue/000069-first-web-target-r7b-exact-t-replay.md` のみだった。Second Application Proposal component、Customer shared authority、Application Pattern validation、reference Contract Query cross validation、Workspace loaderには変更がない。

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

Acceptance prerequisiteとしては、少なくとも`validation.Error`が0件であることを要求する。特に次をsilent skipしない。

- `UNKNOWN_APPLICATION_PATTERN_RELATION`
- Relation key source / target不整合
- Relation key type mismatch
- Contract Query input binding mismatch
- Contract Query result Data mismatch
- Contract Query result cardinality mismatch

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

### Freshness pressure

`second_application_s5_freshness_capture_pressure_test.go` には現在:

```text
secondApplicationS5FreshnessBaseRevision =
  6ffaa40ef2872a773baa5e7b127dcb89d94abc5c
```

が残っており、revised S5 documentのreview baseと一致していない。これは今回のsource inspectionで新たに確認したstale executable fieldである。

health-capable code milestoneでは、このconstantをcurrent inspected relevant-source base:

```text
e1f13ae97b420a2703eec8598eb98c0ab057258b
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
