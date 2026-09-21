# Second Application Reference — S5 Shared Acceptance Review Packet

- Status: Executable pressure PASS / Shared Acceptance Preview PASS / independent generative-AI review completed（review-clear-with-conditions, 2026-09-21） / explicit human Acceptance decision pending
- Date: 2026-09-20
- Primary workstream: ISSUE-000040
- Proposal: `plan/SECOND_APPLICATION_S4_FULL_VALIDATION_PREFLIGHT_2026-09.md`
- Executable pressure implementation packet: `plan/SECOND_APPLICATION_S5_EXECUTABLE_PRESSURE_IMPLEMENTATION_PACKET_2026-09.md`
- Revised Proposal SHA-256: `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`
- Superseded Proposal SHA-256: `b347b41b75c9864e66b4b191eefc3d668c0d8bf3dfdf186d0733f794159467c2`
- Review base inspected at: `c185937ab27a3a31332fefcd63ba5940b97b021e`
- Shared Acceptance result: Not established
- Current Shared Acceptance implementation compatibility: Established — typed multi-layer operations implemented; read-only Preview against actual `reference/workspace` PASS
- Authoritative destination source created: No
- Environment binding created: No
- Target implementation started: No

## 1. 目的

full workspace Validationのpreflightで修正したthree-source Proposalを、Shared Acceptanceへ渡せるreview contractとして固定する。

2026-09-21時点でrevised three-source executable pressureとmulti-layer Shared Acceptance Previewは取得済みである。ただし、それらを人のAcceptanceへ読み替えず、別の生成AIによる独立レビューと、その後のexplicit human Acceptance decisionを独立条件として残す。

```text
revised review defined
  != executable pressure current
  != accepted

explicit witness
  != runtime authority
```

を維持する。

## 2. Exact revised Proposal identity

exact payloadは次の順で構成する。

```text
<Semantic source>
\n---\n
<Application source>
\n---\n
<Pattern source>
```

各component末尾に追加newlineを含めない。

- Semantic SHA-256: `1005c6deb670c277c8b80fb0b353f1eecea690594f2ef4b80df28cc638736e63`
- Application SHA-256: `f96fdb46f9385c205d360f56d0f91a6e841ff31b348ebf1db2cf03381751772f`
- Pattern SHA-256: `41c8f81ebab384486d4df6a7cdaade07df8d07c89d22c25c2291d8b71a583ab1`
- Bundle length: `741 bytes`
- Bundle SHA-256: `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`

旧fingerprintへのAcceptance witnessをrevised Proposalへ流用しない。

## 3. Semantic Diff

### Added destination-owned meaning

```text
semantic namespace:
  repair_intake

semantic data:
  repair_intake.intake

semantic attribute:
  repair_intake.intake.customer_id

semantic relation:
  repair_intake.intake.customer

application:
  repair_intake

application pattern:
  repair_intake.customer_reference

datasource alias:
  repair_intake/customer_master
```

### Reused shared authority

```text
semantic-data:customer.customer
semantic-type:customer.customer_id
database-contract:master_data.customer
database-contract-query:master_data.customer/find_by_id
```

Customer Data / Customer ID Type / Contract / Queryをdestinationへcopyしない。

### Provenance only

```text
application-pattern:order_entry.customer_reference
```

origin Pattern / Relation / alias / Environment bindingをdestination authorityとして継承しない。

### No change

このProposalは次を変更しない。

- shared Customer Semantic / Type
- `master_data` Database / Customer Contract / Query
- origin `order_entry` Application / Pattern
- Persistence Mapping
- development / production Environment
- Provider configuration
- Runtime Permission / Resource grant
- Business Procedure
- First Web Target implementation

## 4. Validation preflight rationale

current Validation source inspectionでは次を要求する。

- reference PatternのRelationがworkspaceに実在する
- Relation keyのsource / target attributeが実在する
- key両端のSemantic type identityが一致する
- Relation targetがContract Query result Dataと一致する
- Relation cardinality `one` はQuery result `one` または `zero-or-one` と互換

current shared Contractは `find_by_id -> customer.customer zero-or-one` であり、revised Relationは `customer.customer one` なので、この互換規則に合う設計である。

current `validation.Validate()` は `validateQueryResultMappings()` → `validateApplicationDataAccessContracts()` を通じ、Relation key / Contract Query invocation / cross-layer検査を含むことも再確認した。

これはSource inspectionによるDesign Evidenceであり、temporary workspaceをactual load / validateしたExecutable Evidenceではない。

## 5. Persistence / authoring impact after Acceptance

Acceptance後に限り、S6で次をauthoritative source candidateとして作成する。

```text
reference/workspace/semantic/repair-intake/intake.tei
reference/workspace/applications/repair-intake/application.tei
reference/workspace/applications/repair-intake/patterns/customer-reference.tei
```

workspace manifest変更、Persistence Mapping、Environment binding、Provider、Authorization grantは不要である。

```text
Accepted semantic change
  -> S6 three-source authoring
  != Environment accepted
  != runtime authority granted
```

## 6. Authorization / execution boundary

S3の次を保持する。

```text
authorization requirement = required-current-authority-check
```

S5 Acceptanceが成立しても:

- principalをgrantしない
- Resource / Permissionをgrantしない
- Lease / ownership / fencingを作らない
- future executionを恒久許可しない

effectful operationではeffect直前にcurrent authorityを別途確認する。

## 7. Retained unresolved

- exact authorization principal
- exact Resource / Permission identity
- development binding target
- production binding target
- Provider verification
- actual S6 reload / Validation Evidence
- Driver / Target Evidence
- change propagation Evidence

unresolvedをAcceptance時に推測で埋めない。

## 8. Acceptance prerequisites

### A. Exact Proposal identity

fingerprint:

```text
c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d
```

を対象にする。

### B. Current shared-authority freshness

`plan/SECOND_APPLICATION_S5_FRESHNESS_CAPTURE_2026-09.md` のcurrent captureを用い、Customer / Contract / origin provenanceのrelevant sourceが変化していないことを確認する。

### C. Revised executable pressure

health-capable cloneで少なくとも次を更新・実行する。

1. S3 pressureへdestination Semantic namespace / Data / attribute / type / Relation / keyを反映
2. S4 pressureへ`SemanticSource`を追加し、`ParseSemanticFile()` + Application + Patternのthree-source identityを検査
3. `repoPath()` と既存`copyDirectory` helperで`reference/workspace`を`t.TempDir()`配下へcopyし、Proposal 3-sourceだけをtemporary workspaceへ追加
4. `workspace.Load(probeRoot)` でactual loader pathを通す
5. `validation.Validate(loaded.World)` でfull Validationを実行し、`validation.Error`が0件であることを要求
6. revised fingerprint `c18ae0f...` とdestination-owned Semantic subjectsをS5 pressureへ反映
7. S5 freshness pressureのreview-base constantを `c185937ab27a3a31332fefcd63ba5940b97b021e` へ追随させる
8. `tools/run-knowledge-portability-reference-evidence.sh` のgofmt対象へS3/S4/S5 pressureとfull-validation pressureを追加
9. focused Second Application tests、reference evidence script、`tools/repository-health-check.sh` をすべてPASSさせる

この条件が未取得の間、Source inspectionだけでAcceptance-readyへ昇格しない。

### D. Semantic Diff review

reviewerはdestination-owned Semantic / Application / Patternとshared reuseを区別できること。

### E. Explicit Acceptance witness

Acceptance authorityを持つactorのexplicit witnessを要求する。witnessは少なくとも:

- exact revised Proposal fingerprint
- accepted / rejected decision
- decision context
- retained unresolved
- witness identity / locus
- decision timeまたはrevision identity

へ結び付ける。

一般的な「作業を進める」「次へ進む」という実行指示は、このexact Proposalへのaccepted / rejected Decisionとして扱わない。

2026-09-21、利用者の方針: Acceptanceは別の生成AIによる独立レビューの後に行う。これはwitnessの前に置く手順の決定であり、それ自体はwitnessではない。レビュー後にacceptを実行する時、上のwitness項目を[ISSUE-000040](issue/000040-semantic-component-composition.md)へ記録する。

## 8.1 Shared Acceptance implementation compatibility — historical gap（closed 2026-09-21）

この節の以下の記述は、multi-layer operation実装前のcompatibility gapを記録したhistorical contractである。`0f069838...`で`add-data` / `add-application` / `add-reference-pattern`が実装され、actual `reference/workspace` Preview PASSまで取得したため、このgap自体はclosedである。

当時の `tools/reference-shared-acceptance` は:

- candidate operation = `add-validation` のみ
- semantic diff = `World.Semantics` subjectsのみ
- architecture-policy = changed collectionがexactly `semantics`

である。

revised Second Application ProposalはSemantic + Application + Patternを同時に追加し、正当に `applications` collectionも変えるため、current boundaryへそのまま渡すことはできない。

このため、direct source authoringや別Acceptance engineへ迂回せず、ISSUE-000027の同じboundaryへmulti-layer composition pressureを追加する。

正本:

- `plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_COMPATIBILITY_PRESSURE_2026-09.md`

pre-Acceptance停止条件は次の3つとして扱う。

1. revised executable pressureのhealth PASS
2. exact `c18ae...` Proposalをcurrent base revisionへ結び付けたShared Acceptance Preview PASS
3. explicit human Acceptance decision

PreviewはAcceptanceではない。通常の継続指示もAcceptanceではない。

## 9. Current decision state

```text
proposalFingerprint             = c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d
reviewBaseInspected             = c185937ab27a3a31332fefcd63ba5940b97b021e
sharedFreshness                 = captured
revisedExecutablePressure       = obtained (main 41b21d1068bdfabd122b8543afe980efc4b5fdb5)
multiLayerAcceptancePreview     = pass
previewBaseModelRevision        = sha256:f70da9b64a5b48c7682b24c64f455b0b0c2a7278be4faa297b6ff99c2d3707bf
acceptanceCandidateFingerprint  = sha256:8a936cf96ddeb4faa220285dcef5055d418030373f9425ed8d0d1c2b3e7c9b11
previewModelRevisionAfter       = sha256:732763a2b645a6783dd8fbe61b861f2a2c7842f464c1759de846f71eaf18f762
previewStages                   = 11 / 11 PASS
previewChangedSubjects          = 6
previewChangedCollections       = [applications, semantics]
previewImpactEdges              = 15
previewUnresolved               = [ENVIRONMENT_BINDING_NOT_AUTHORED, DRIVER_SCENARIO_NOT_RUN, TARGET_PROJECTION_NOT_EVALUATED]
independentGenerativeAIReview   = review-clear-with-conditions
acceptanceWitness               = absent
decision                        = pending
```

したがって、機械的なpre-Acceptance条件と独立レビュー条件は成立している。現在の停止条件は一つである。

```text
explicit human Acceptance decision for exact c18ae...
```

独立レビュー結果は `plan/SECOND_APPLICATION_S5_INDEPENDENT_REVIEW_RESULT_2026-09.md` に記録した。結果は `review-clear-with-conditions` で、required pre-Acceptance fixは無い。Repository integration条件はactual accept後からcommit前までに適用する。

Preview / health PASS / review依頼そのものをAcceptance witnessへ読み替えない。

## 10. Negative cases

- old fingerprintへのwitnessを新Proposalへ流用しない
- parser-only successをfull Validation successへ昇格しない
- Source inspectionをactual test PASSとして扱わない
- witnessがあってもstale / unverified Proposalをsilent applyしない
- Semantic AcceptanceをRuntime authorizationへ変換しない
- AIが自分のProposalをself-acceptしない

## 11. Evidence height

2026-09-21時点のEvidenceは次の高さまで上がっている。

- revised three-source executable pressure: main `41b21d1068bdfabd122b8543afe980efc4b5fdb5` で取得
  - focused S3–S5: 22 tests PASS
  - full-workspace Validation positive case + negative control 2件 PASS
  - `tools/run-knowledge-portability-reference-evidence.sh` PASS
  - `tools/repository-health-check.sh` PASS
- multi-layer Shared Acceptance Boundary: main `0f069838aecd3cc6627a1bc2c11cac308493021f` で実装
- actual `reference/workspace` read-only Preview: 11 / 11 stages PASS
- `1d720bde...` のRelation impact追加後も再Previewし、fingerprint / revisions / 6 changed subjects / unresolvedは不変、impact edgeのみ14→15へ増加
- authoritative destination source: 未作成
- Shared Acceptance decision: 未成立
- independent generative-AI review: completed — `review-clear-with-conditions`（GPT-5.6 Sol, 2026-09-21）

独立レビューまで完了した。次に必要なのはexact `c18ae...`に対するexplicit human Acceptance decisionである。

## 12. Conclusion

S4 ProposalのRelation source欠落、executable pressure、multi-layer Shared Acceptance compatibilityはいずれも解消済みである。

現在の順序は:

```text
exact c18ae...へのexplicit human Acceptance decision
  -> actual Shared Acceptance accept
  -> reference-golden -write / verify
  -> repository health check
  -> same commit
  -> accepted sourceからReference golden / repository conformanceを再確認
  -> S6 authoritative-source / reload Evidenceを記録
```

である。

独立レビュー用の入力は `plan/SECOND_APPLICATION_S5_INDEPENDENT_REVIEW_HANDOFF_2026-09.md` に固定する。
