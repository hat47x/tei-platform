# Second Application Reference — S6 Source / Reload / Validation Packet

- Date: 2026-09-21
- Primary workstream: ISSUE-000040
- Stage: S6
- Entry condition: exact Proposal `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d` has an explicit human Acceptance decision and the S5 Acceptance Execution Packet has completed successfully
- Current status: **blocked on explicit human Acceptance; no authoritative destination source exists yet**
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

- [ ] explicit human Acceptance witnessがISSUE-000040に記録済み
- [ ] Shared Acceptance actual accept成功
- [ ] accepted source 3 filesがauthoritative Workspaceに存在
- [ ] accepted Decision recordが`current` / `intact`として機械到達でき、record無しでもaccepted meaning / model revisionがsourceから自立する
- [ ] `reference/world.json`をaccepted sourceから正規再生成
- [ ] normal `tei validate` PASS
- [ ] current model revision = Preview `modelRevisionAfter`
- [ ] generated World = golden exact match
- [ ] Source Index 6 subjectsがexpected artifactへ戻る
- [ ] shared Customer / Contract / origin Pattern source unchanged
- [ ] Environment source unchanged
- [ ] reference evidence runner PASS
- [ ] repository health check PASS
- [ ] historical Gate B/C scopeをrevision-scopedのまま保持
- [ ] retained unresolvedをS7 / S8へhandoff
- [ ] dedicated portability grammar不要というinitial Reference Evidenceを記録

S6完了後にだけS7 Environment Binding Proposalへ進む。
