# Second Application Reference — S5 Shared Acceptance Implementation Packet

- Status: Implementation contract defined / code not yet authored
- Date: 2026-09-21
- Primary workstream: ISSUE-000040
- Shared boundary owner: ISSUE-000027
- Compatibility pressure: `plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_COMPATIBILITY_PRESSURE_2026-09.md`
- Revised S4 source Proposal SHA-256: `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`
- Current executable-pressure candidate: `b38869a48dcd8fb3e826a80de239063ee918c366`
- Authoritative destination source created: No
- Human Acceptance decision: Absent

## 1. Goal

current `tools/reference-shared-acceptance` のpipelineを再利用し、Second Applicationのthree-source compositionをread-only Previewまで通せる最小incrementを定義する。

別Acceptance engineは作らない。

```text
decode
  -> normalize
  -> base revision
  -> resolve
  -> overlay apply
  -> reload
  -> Validation
  -> architecture / containment
  -> semantic + application diff
  -> impact / unresolved
  -> acceptance policy
```

の順序は維持する。

## 2. Candidate operation direction

current `add-validation` と同じtyped operation方式を維持する。

raw file patch / arbitrary text replacementは採用しない。

Second Applicationで実測された意味だけを表せる最小operationとして、実装時には次の3種をpressureする。

```text
add-data
add-application
add-reference-pattern
```

名称はcode reviewで調整可能だが、責務を一つの巨大な `add-composition` operationへ畳み込まない。

### 2.1 add-data

最初のincrementでは、新しいdestination namespaceに一つのDataを追加するcaseだけを扱う。

必要なmeaning:

```text
namespace = repair_intake
data = repair_intake.intake
data symbol = Intake

attribute:
  id = repair_intake.intake.customer_id
  symbol = customerId
  type = customer.customer_id

relation:
  id = repair_intake.intake.customer
  symbol = customer
  target = customer.customer
  cardinality = one
  key source = repair_intake.intake.customer_id
  key target = customer.customer.customer_id
```

既存namespaceへの一般的なData追記は、このReferenceが必要とするまで広げない。

### 2.2 add-application

必要なmeaning:

```text
application = repair_intake

semantic uses:
  repair_intake
  customer

database requirement:
  alias = customer_master
  target = master_data
  access = read
  required contract = master_data.customer ^2
```

Environment bindingは含めない。

### 2.3 add-reference-pattern

必要なmeaning:

```text
pattern = repair_intake.customer_reference
kind = reference
owner application = repair_intake
relation = repair_intake.intake.customer
datasource = customer_master

contract-query:
  contract = master_data.customer
  query = find_by_id
  input customer.customer.customer_id -> id
```

browse / record / command / header-detailへ先行拡張しない。

## 3. Placement and deterministic authoring

各operationはmeaningとplacementを分けて持つ。

このReferenceで許可するplacementはexactly:

```text
semantic/repair-intake/intake.tei
applications/repair-intake/application.tei
applications/repair-intake/patterns/customer-reference.tei
```

resolverは:

- pathがworkspace managed root内である
- expected layerとpath prefixが一致する
- destination fileがまだ存在しない
- application-pattern pathのowner directoryがApplicationと一致する

ことを確認する。

source生成はoperationから決定論的に行う。
callerが任意のTEI textをpatch payloadとして渡す設計にはしない。

これにより同じnormalized meaning + placementから入口に依らず同じsourceを生成する。

## 4. Fingerprint and S4 trace

Shared Acceptance candidate fingerprintはcurrent ruleどおり:

```text
baseModelRevision + normalized operations
```

から作る。

S4 exact source Proposal fingerprint `c18ae...` と同一値にする必要はない。

代わりにadapter pressureで:

1. S4 three-source payloadを再計算して `c18ae...` と一致させる。
2. S4 parsed meaningから3 typed operationsを生成する。
3. typed operationsから決定論的sourceを再生成する。
4. 再生成したthree-source payloadも `c18ae...` と一致させる。
5. current `baseModelRevision` を加えてAcceptance candidate fingerprintを取得する。

これにより:

```text
S4 exact source identity
  <-> normalized typed meaning
  <-> deterministic regenerated source
  <-> current-base Acceptance candidate identity
```

をtraceできる。

## 5. Diff extension

current `changeanalysis.Diff` はsemantic subjectsだけを返す。
既存のadd-validation pathを壊さず、Application-owned diffを追加する。

最小案:

```go
Diff(before, after)                 // existing semantic subject diff
DiffApplications(before, after)     // new application-owned subject diff
```

`DiffApplications` の最初のflatten単位は2種に絞る。

### application subject

```text
kind = application
id = repair_intake
```

raw comparisonにはApplication identityに加え:

- SemanticUses
- Databases

を含め、Patternsは除く。

これによりPatternだけの変更でApplication本体まで二重にchanged扱いしない。

### application-pattern subject

```text
kind = application-pattern
id = repair_intake.customer_reference
```

Pattern全体をraw comparison対象にする。

Datasource alias / Contract requirementはApplication subjectの内部meaningとして保持する。
このincrementで別の `datasource-alias` diff kindを増やさない。

Second Application initial acceptanceで期待するadded subjectは:

```text
semantic:
  namespace:repair_intake
  data:repair_intake.intake
  attribute:repair_intake.intake.customer_id
  relation:repair_intake.intake.customer

application:
  application:repair_intake
  application-pattern:repair_intake.customer_reference
```

合計6 subjectsであり、current `maxSubjectChanges = 8` の範囲内に収まる。

## 6. Changed collections policy

add-validation pathではcurrent ruleを維持する。

```text
expected changed collections = [semantics]
```

multi-layer Second Application candidateでは:

```text
expected changed collections = [applications, semantics]
```

をtyped operationsから導出する。

単純に「semantics以外も許可」に緩めず、actual changed collection setとexpected setのexact equalityを要求する。

次が一つでも変わればrejectする。

```text
databases
persistenceMappings
migrations
environments
queries
customGoBindings
applicationReleases
databaseReleases
solutionReleases
...
```

## 7. Source containment

current `checkContainment` のfail-closed方針を維持し、allowed edit setをtyped operationsから導出する。

Second Application Previewではexactly3 fileだけを許す。

undeclared fourth file、existing shared Customer source、origin `order_entry`、Environment fileへのwriteを拒否する。

## 8. Validation and cross-layer pressure

overlay適用後は既存 `workspace.Load` + `validation.Validate` をそのまま使う。

別validatorを作らない。

Second Application executable pressureで固定済みのnegative cases:

- `UNKNOWN_APPLICATION_PATTERN_RELATION`
- `REFERENCE_CONTRACT_QUERY_RESULT_CARDINALITY_MISMATCH`

はShared Acceptance Preview経路でも同じ通常Validation diagnosticへ到達させる。

## 9. Preview pressure tests

実装時に少なくとも次を追加する。

```text
TestSecondApplicationCompositionPreviewUsesSharedAcceptanceBoundary
TestSecondApplicationCompositionPreviewLeavesSourceByteIdentical
TestSecondApplicationCompositionPreviewBindsExactS4Proposal
TestSecondApplicationCompositionPreviewReportsSemanticAndApplicationDiff
TestSecondApplicationCompositionPreviewRejectsStaleBase
TestSecondApplicationCompositionPreviewRejectsExistingDestinationCollision
TestSecondApplicationCompositionPreviewRejectsUndeclaredFourthEdit
TestSecondApplicationCompositionPreviewRejectsMissingPattern
TestSecondApplicationCompositionPreviewRejectsUnknownRelation
TestSecondApplicationCompositionPreviewRejectsCardinalityMismatch
TestSecondApplicationCompositionPreviewRejectsUndeclaredCollectionChange
```

同じcandidateをoriginだけ:

```text
file
cli
studio
ai
```

へ変えたとき、normalized meaning / generated source / decision結果が変わらないpressureも追加する。

## 10. Accept-path mechanical tests

human Acceptanceそのものとは別に、tool mechanicsはscratch workspaceでtestできる。

少なくとも:

- successful accept writes exactly 3 expected files
- post-write reload / Validation PASS
- accepted diff equals Preview diff
- post-write conformance failure rolls all 3 files back
- before-write concurrent change refuses and does not partially write
- Environment / Runtime authority remains untouched

をtestする。

このmechanical test PASSを、actual Second Applicationのhuman Acceptanceへ読み替えない。

## 11. Implementation files

expected code touch points:

```text
tools/reference-shared-acceptance/candidate.go
tools/reference-shared-acceptance/edit.go
tools/reference-shared-acceptance/pipeline.go
tools/reference-shared-acceptance/decision.go        // only if trace evidence field is needed
tools/reference-shared-acceptance/*_test.go
tools/internal/changeanalysis/diff.go
tools/internal/changeanalysis/analysis_test.go
```

既存add-validation testsを全て維持する。

Second Application専用Acceptance packageは作らない。

## 12. Execution sequence

health-capable full checkoutで:

```text
pull --rebase latest main
  -> relevant source drift recheck
  -> implement typed multi-layer operations
  -> changeanalysis Application diff
  -> focused Shared Acceptance tests
  -> focused TestSecondApplication...
  -> tools/run-knowledge-portability-reference-evidence.sh
  -> tools/repository-health-check.sh
```

を実行する。

全PASS後も、actual Second Applicationで行うのはまずread-only Previewまでである。

explicit human Acceptance decisionが無い限り `accept` をactual destination sourceへ実行しない。
