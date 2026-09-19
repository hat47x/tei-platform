# Second Application Reference — S5 Executable Pressure Implementation Packet

- Status: Implementation contract pinned / actual execution pending
- Date: 2026-09-20
- Primary workstream: ISSUE-000040
- Proposal SHA-256: `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`
- Source-inspection base: `1f3090c1e86de9d80e39879ffe8987c80a946fd4`
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

前回Second Application文書更新 `d8a7fd24...` 以後、inspection時点のmainは:

```text
1f3090c1e86de9d80e39879ffe8987c80a946fd4
docs(r7b): record branchless V1 shared-validator candidate
```

である。

その間の変更はR7 / R7B文書に限られ、Second Application Proposal component、Customer shared authority、Application Pattern validation、reference Contract Query cross validation、Workspace loaderには変更がないことをcompareで確認した。

関連blob identity:

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

## 5. S5 exact Proposal pressure

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

## 6. Reference evidence runner追随

current `tools/run-knowledge-portability-reference-evidence.sh` のtest regexは`TestSecondApplication...`を実行するが、gofmt対象filesにはS3/S4/S5 shared-acceptance pressureが含まれていない。

code milestoneでは同scriptの`files=(...)`へ次を追加する。

```text
tools/reference-business-procedure-check/second_application_s3_destination_decision_pressure_test.go
tools/reference-business-procedure-check/second_application_s4_composition_candidate_pressure_test.go
tools/reference-business-procedure-check/second_application_s5_shared_acceptance_review_pressure_test.go
```

gofmt pressureとactual executed test集合を一致させる。

## 7. Required execution sequence

health-capable cloneでは次の順を固定する。

```text
pull --rebase current main
  -> relevant source drift recheck
  -> S3/S4/S5 pressure + evidence runner update
  -> gofmt
  -> focused TestSecondApplication...
  -> tools/run-knowledge-portability-reference-evidence.sh
  -> tools/repository-health-check.sh
  -> only after PASS: direct-to-main commit/push
```

mainが途中で進んだ場合はforce pushせず、rebase / relevant drift reviewをやり直す。

## 8. Evidence and Acceptance boundary

このpacketをmainへ置いても:

- revised executable pressure = absent
- Shared Acceptance = absent
- S6 authoritative authoring = blocked
- Runtime authority = absent

のままである。

health-check PASS後もexplicit Acceptance witnessは独立条件として残す。一般的な作業継続指示をfingerprint `c18ae0f...` のAcceptanceへ読み替えない。
