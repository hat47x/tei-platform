# Second Application Reference — S6 Source / Reload / Validation Packet

- Date: 2026-09-21
- Primary workstream: ISSUE-000040
- Stage: S6
- Entry condition: exact Proposal `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d` has an explicit human Acceptance decision and the S5 Acceptance Execution Packet has completed successfully
- Current status: **complete — 2026-09-21、`ACCEPT_COMMIT = 0423b95`について全exit criteriaがPASS（§15）**
- This document is a post-Acceptance verification contract, not an Acceptance witness and not authorization to write source.

## 1. Purpose

S5 Shared Acceptanceの結果としてauthoritative sourceへ入ったSecond Application compositionが、Acceptance tool・AI・exploration indexへ依存せず、通常のTEI Workspaceとして成立することをEvidence化する。

S6で確認するのは「accepted sourceが正本として自立していること」であり、Environment / Driver / Targetへ先回りしない。

```text
S5 explicit human Acceptance
  -> Shared Acceptance accept + golden integration
  -> S6 normal reload / validation / source trace
  -> S7 Environment Binding Proposal
```

## 2. Expected accepted source

S6 entry時点で存在してよい新規authoritative sourceはexactly次の3 fileである。

```text
reference/workspace/semantic/repair-intake/intake.tei
reference/workspace/applications/repair-intake/application.tei
reference/workspace/applications/repair-intake/patterns/customer-reference.tei
```

matching derived golden:

```text
reference/world.json
```

non-authoritative accepted Decision Evidence:

```text
reference/workspace/.tei/decisions/<decision-record-sha256>.json
```

The Decision record is not managed source and does not participate in the Workspace model revision.

この段階で新規作成してはいけないもの:

- Environment binding
- Persistence Mapping
- migration
- Provider configuration
- Permission / Resource grant
- Runtime authority
- Business Procedure
- Driver realization
- Target-specific source

## 3. Identity carried from S5

S6はS5のidentityを作り直さない。

```text
S4 exact source Proposal
  sha256:c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d

pre-accept Workspace model revision
  sha256:f70da9b64a5b48c7682b24c64f455b0b0c2a7278be4faa297b6ff99c2d3707bf

reviewed Acceptance candidate fingerprint
  sha256:8a936cf96ddeb4faa220285dcef5055d418030373f9425ed8d0d1c2b3e7c9b11

expected post-accept Workspace model revision
  sha256:732763a2b645a6783dd8fbe61b861f2a2c7842f464c1759de846f71eaf18f762
```

actual accept直前のre-Previewでこれらがdriftした場合、S6へ入らずS5へ戻る。

## 4. S6 entry evidence

S5 integrationが成功したcommitを `ACCEPT_COMMIT`、その直前を `PRE_ACCEPT_HEAD` として固定する。

ISSUE-000040に少なくとも:

```text
proposalFingerprint
humanAcceptanceDecision
witnessIdentity / locus
decisionTime
PRE_ACCEPT_HEAD
ACCEPT_COMMIT
baseModelRevision
acceptanceCandidateFingerprint
modelRevisionAfter
acceptedSourcePaths
decisionRecordPath
decisionRecordStatus
decisionRecordSourceEdits
referenceGoldenResult
repositoryHealthResult
retainedUnresolved
```

が記録されていること。

一般的な「進めてください」をこのentry evidenceへ代入しない。

## 5. Normal Workspace reload

Acceptance toolを使わず、通常CLI / loaderだけで検査する。

```bash
go run ./cmd/tei validate -workspace reference/workspace
```

要求:

- exit 0
- Validation Error = 0
- final result = `ok`

次にcurrent model revisionを通常Workspaceから再確認する。

```bash
go run ./tools/reference-shared-acceptance revision \
  -workspace reference/workspace
```

要求:

```text
modelRevision = sha256:732763a2b645a6783dd8fbe61b861f2a2c7842f464c1759de846f71eaf18f762
```

これはS5 Previewの `modelRevisionAfter` とexact一致しなければならない。

## 5.1 Accepted Decision reachability

Verify that the accepted Decision is reachable without the originating conversation:

```bash
go run ./tools/reference-shared-acceptance decisions \
  -workspace reference/workspace \
  -out /tmp/tei-second-application-decision-index.json
```

The record matching candidate fingerprint `sha256:8a936cf96ddeb4faa220285dcef5055d418030373f9425ed8d0d1c2b3e7c9b11` must report:

```text
decision = accepted
status = current
sourceEdits = intact
modelRevisionAfter = sha256:732763a2b645a6783dd8fbe61b861f2a2c7842f464c1759de846f71eaf18f762
```

This record is Evidence, not authority. Accepted meaning must still reload from the three TEI source files if the record is ignored or removed.

## 6. Canonical World / golden reproduction

通常Workspace loaderからWorldを再生成する。

```bash
go run ./cmd/tei world \
  -workspace reference/workspace \
  > /tmp/tei-second-application-world.json

cmp /tmp/tei-second-application-world.json reference/world.json
```

さらに正規golden checkerを通す。

```bash
go run ./tools/reference-golden
```

要求:

- current Workspace World = `reference/world.json`
- plugin lock = current Environmentからの正規派生物と一致
- historical Durable Runのplugin-lock dependencyを壊さない

S5独立レビューどおり、`reference/world.json`はpost-Gate-C current Referenceのgoldenであり、過去のGate C `R / T`そのものではない。

## 7. Source ownership / Source Index

accepted meaningが通常loaderのsource mapへ乗ることを確認する。

期待するexact source ownership:

| subject kind | subject id | expected artifact |
| --- | --- | --- |
| `semantic` | `repair_intake` | `semantic/repair-intake/intake.tei` |
| `semantic-data` | `repair_intake.intake` | `semantic/repair-intake/intake.tei` |
| `semantic-attribute` | `repair_intake.intake.customer_id` | `semantic/repair-intake/intake.tei` |
| `semantic-relation` | `repair_intake.intake.customer` | `semantic/repair-intake/intake.tei` |
| `application` | `repair_intake` | `applications/repair-intake/application.tei` |
| `application-pattern` | `repair_intake.customer_reference` | `applications/repair-intake/patterns/customer-reference.tei` |

各subjectを共有Source Index入口から確認する。

```bash
go run ./cmd/tei source-index -workspace reference/workspace \
  -subject-kind semantic -subject-id repair_intake

go run ./cmd/tei source-index -workspace reference/workspace \
  -subject-kind semantic-data -subject-id repair_intake.intake

go run ./cmd/tei source-index -workspace reference/workspace \
  -subject-kind semantic-attribute -subject-id repair_intake.intake.customer_id

go run ./cmd/tei source-index -workspace reference/workspace \
  -subject-kind semantic-relation -subject-id repair_intake.intake.customer

go run ./cmd/tei source-index -workspace reference/workspace \
  -subject-kind application -subject-id repair_intake

go run ./cmd/tei source-index -workspace reference/workspace \
  -subject-kind application-pattern -subject-id repair_intake.customer_reference
```

各結果はdefinition roleで上表のartifactへexactに戻れること。

`customer_master` datasource aliasはApplication内部meaningであり、このReferenceでは独立Source Index subjectへ昇格させない。導出できる内部要素を人手の第二identityとして増やさない。

## 8. Managed-file / derived-index independence

通常Workspaceが3 accepted sourceをmanaged sourceとして列挙できることを確認する。

```bash
go run ./cmd/tei workspace-files -workspace reference/workspace \
  > /tmp/tei-second-application-workspace-files.json

go run ./cmd/tei index -workspace reference/workspace \
  > /tmp/tei-second-application-index.json
```

ここでindexは正本ではない。

S6の成立条件は:

```text
authoritative TEI source
  -> workspace.Load
  -> Canonical World
  -> Source Index / derived index
```

の向きで再生成できることであり、indexからsourceを復元することではない。

探索indexやAI sessionを削除してもaccepted meaningが失われないことを、このauthority directionで示す。

## 9. Shared authority remains shared

Second Application acceptanceがshared authorityをcopy / rewriteしていないことをcommit差分で確認する。

```bash
git diff --exit-code "$PRE_ACCEPT_HEAD" "$ACCEPT_COMMIT" -- \
  reference/workspace/semantic/customer/customer.tei \
  reference/workspace/databases/master-data/contracts/customer.tei \
  reference/workspace/applications/order-entry/patterns/customer-reference.tei
```

要求: 差分なし。

meaning:

```text
customer.customer
customer.customer_id
master_data.customer ^2
master_data.customer/find_by_id
```

は既存shared authorityのexact identityを参照している。

`order_entry.customer_reference`はprovenance-onlyであり、destination authorityへ昇格していない。

## 10. Environment / runtime non-creation

S6でEnvironment-owned decisionを作らない。

```bash
git diff --exit-code "$PRE_ACCEPT_HEAD" "$ACCEPT_COMMIT" -- \
  reference/workspace/environments
```

要求: 差分なし。

同様にAcceptance commitに:

- Permission
- Resource Binding
- Lease
- Provider
- persistence / migration
- Target source

が混ざっていないことをname-status diffで確認する。

S5 Previewの:

```text
ENVIRONMENT_BINDING_NOT_AUTHORED
DRIVER_SCENARIO_NOT_RUN
TARGET_PROJECTION_NOT_EVALUATED
```

はS6終了時点でも後続stageへ保持する。

## 11. Acceptance commit shape

semantic integration unitのsource/golden差分を確認する。

```bash
git diff --name-status "$PRE_ACCEPT_HEAD" "$ACCEPT_COMMIT" -- \
  reference/workspace reference/world.json
```

expected content change:

```text
A reference/workspace/semantic/repair-intake/intake.tei
A reference/workspace/applications/repair-intake/application.tei
A reference/workspace/applications/repair-intake/patterns/customer-reference.tei
A reference/workspace/.tei/decisions/<decision-record-sha256>.json
M reference/world.json
```

`reference/workspace/.tei/plugin-lock.json`をgolden toolが再生成しても、内容差分は期待しない。

Evidence文書更新はこのsemantic diffそのものと混同せず、必要なら後続docs-only commitで記録する。

## 12. Historical Gate B / Gate C scope

過去のEvidence:

```text
B = B
R = e5231839a5dcc62cce1181a2731fcb7c70d16736
T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe
```

はそのexact revisionsについて成立したhistorical Evidenceである。

S6でcurrent Referenceが進化しても:

- historical R / Tを変更しない
- old Gate C Readyを遡及的に再解釈しない
- post-accept Workspaceをold B/R/T baselineとして扱わない

将来Second Applicationを含むTarget Evidenceを固定する必要が生じた時点でsuccessor baseline / revisionを発行する。

## 13. Dedicated portability grammar hypothesis

S6 accepted sourceが:

- existing Semantic grammar
- existing Application grammar
- existing reference Pattern grammar
- existing Database Contract reference

だけで通常reload / Validation / Source Indexへ乗るなら、初回Second Application Referenceについて専用Knowledge Package / inheritance / override / sync grammarは不要だったというEvidenceになる。

これは「将来も絶対に専用grammar不要」という一般化ではない。

後続S7〜S9で、Environment bindingやchange propagationを表現できない具体的Evidenceが出た場合だけschema追加を再検討する。

## 14. S6 exit criteria

すべて成立したときだけS6をcompleteにする。

- [x] explicit human Acceptance witnessがISSUE-000040に記録済み
- [x] Shared Acceptance actual accept成功
- [x] accepted source 3 filesがauthoritative Workspaceに存在
- [x] accepted Decision recordが`current` / `intact`として機械到達でき、record無しでもaccepted meaning / model revisionがsourceから自立する
- [x] `reference/world.json`をaccepted sourceから正規再生成
- [x] normal `tei validate` PASS
- [x] current model revision = Preview `modelRevisionAfter`
- [x] generated World = golden exact match
- [x] Source Index 6 subjectsがexpected artifactへ戻る
- [x] shared Customer / Contract / origin Pattern source unchanged
- [x] Environment source unchanged
- [x] reference evidence runner PASS
- [x] repository health check PASS
- [x] historical Gate B/C scopeをrevision-scopedのまま保持
- [x] retained unresolvedをS7 / S8へhandoff
- [x] dedicated portability grammar不要というinitial Reference Evidenceを記録

S6完了後にだけS7 Environment Binding Proposalへ進む。

## 15. 実行記録 — 2026-09-21

```text
PRE_ACCEPT_HEAD  f8785800c66dbca735caefa75d284f6ec5b33396
ACCEPT_COMMIT    0423b95e87197ea36f1f47e9596db14b9d1e52d8
```

Acceptance toolを使わない通常経路だけで確認した。

| 確認 | 結果 |
| --- | --- |
| §5 `tei validate -workspace reference/workspace` | exit 0 / `ok`（Validation Error 0） |
| §5 current model revision | `sha256:732763a2…`。S5 Previewの`modelRevisionAfter`とexact一致 |
| §5.1 決定記録の到達性 | candidate fingerprint `sha256:8a936cf9…`の記録が`accepted` / `current` / `intact`、`modelRevisionAfter`一致 |
| §5.1 記録はEvidenceでありauthorityではない | 記録を除いた複製でもmodel revisionが同一で、Canonical Worldがbyte一致 |
| §6 World = golden | `tei world`の出力が`reference/world.json`とbyte一致。`reference-golden` PASS |
| §7 Source Index | 6 subjectすべてがexpected artifactへexactに戻る |
| §8 managed source | 3 accepted fileがmanaged source（全54件）に含まれ、決定記録は**含まれない** |
| §9 shared authority | `customer.tei` / `contracts/customer.tei` / `order-entry/patterns/customer-reference.tei`に差分なし |
| §10 Environment | `reference/workspace/environments`に差分なし |
| §11 commit形状 | `A`×4（3 source + 決定記録）、`M reference/world.json`。plugin-lockに差分なし |
| §12 historical scope | `R = e523183` / `T = 0ba87a4`は元のrevisionにそのまま存在 |
| evidence runner | PASS（46 test PASS / 0 FAIL） |
| repository health check | PASS |

S6の実行でRepositoryは一切変更していない（検証後も`git status`は空）。

### §13 dedicated portability grammar — initial Reference Evidence

accepted sourceは次だけで通常reload / Validation / Source Indexへ乗った。

```text
existing Semantic grammar
existing Application grammar
existing reference Pattern grammar
existing Database Contract reference
```

専用のKnowledge Package / inheritance / override / sync grammarは、初回Second Application Referenceでは**不要だった**。

これは「将来も不要」という一般化ではない。共有`customer`意味の再利用は、destination側が既存identityを参照するだけで足りた事例であり、override・変種・同期が必要なcaseはまだ現れていない。S7〜S9でEnvironment bindingやchange propagationを表現できない具体的Evidenceが出た場合だけschema追加を再検討する。

### S7へ引き渡すretained unresolved

```text
ENVIRONMENT_BINDING_NOT_AUTHORED   -> S7 Environment Binding Proposalで扱う
DRIVER_SCENARIO_NOT_RUN            -> S8以降。S6では実行可能性を主張しない
TARGET_PROJECTION_NOT_EVALUATED    -> S8以降。Target Projectionは未評価のまま
```

S6完了はこれらを解消しない。accepted sourceが正本として自立していることだけを示す。
