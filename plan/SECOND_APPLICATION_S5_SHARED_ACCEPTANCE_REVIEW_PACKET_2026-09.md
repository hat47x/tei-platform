# Second Application Reference — S5 Shared Acceptance Review Packet

- Status: Revised review defined / executable health + Shared Acceptance compatibility + explicit human decision pending
- Date: 2026-09-20
- Primary workstream: ISSUE-000040
- Proposal: `plan/SECOND_APPLICATION_S4_FULL_VALIDATION_PREFLIGHT_2026-09.md`
- Executable pressure implementation packet: `plan/SECOND_APPLICATION_S5_EXECUTABLE_PRESSURE_IMPLEMENTATION_PACKET_2026-09.md`
- Revised Proposal SHA-256: `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`
- Superseded Proposal SHA-256: `b347b41b75c9864e66b4b191eefc3d668c0d8bf3dfdf186d0733f794159467c2`
- Review base inspected at: `c185937ab27a3a31332fefcd63ba5940b97b021e`
- Shared Acceptance result: Not established
- Current Shared Acceptance implementation compatibility: Not established — current Reference supports add-validation / semantic-only delta
- Authoritative destination source created: No
- Environment binding created: No
- Target implementation started: No

## 1. 目的

full workspace Validationのpreflightで修正したthree-source Proposalを、Shared Acceptanceへ渡せるreview contractとして固定する。

ただしcurrent Go pressureは旧two-source Proposalを前提としているため、この文書だけでAcceptance eligibilityを成立させない。

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

## 8.1 Shared Acceptance implementation compatibility — 2026-09-21

mainへ決定論的Shared Acceptance Boundaryの最初のincrementが入ったため、S5をその実装へ再照合した。

current `tools/reference-shared-acceptance` は:

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
proposalFingerprint       = c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d
reviewBaseInspected       = c185937ab27a3a31332fefcd63ba5940b97b021e
sharedFreshness           = captured
sourceInspectionPreflight = complete
branchlessCodeCandidate    = ccd2544a8a14501774ae3970813c7535e350ec90
candidateGofmtEvidence     = pass
candidateRunnerSyntax      = pass
candidateStaticAPIAudit    = complete
candidateNegativeControls  = pinned-not-executed
candidateFocusedTest       = absent
candidateHealthEvidence    = absent
revisedExecutablePressure = absent
acceptanceWitness         = absent
decision                  = pending
```

したがって:

```text
S5 review contract = defined
Shared Acceptance eligibility = blocked

blocking dimensions:
  executableHealth             = absent
  multiLayerAcceptancePreview  = unsupported-by-current-reference
  explicitHumanDecision        = absent
  - missing revised executable pressure
  - missing explicit acceptance witness
S6 authoritative authoring = blocked
```

である。

## 10. Negative cases

- old fingerprintへのwitnessを新Proposalへ流用しない
- parser-only successをfull Validation successへ昇格しない
- Source inspectionをactual test PASSとして扱わない
- witnessがあってもstale / unverified Proposalをsilent applyしない
- Semantic AcceptanceをRuntime authorizationへ変換しない
- AIが自分のProposalをself-acceptしない

## 11. Evidence height

このpacketはSource / Design Evidenceである。

current `main` には旧two-source Proposal向けGo pressureと旧freshness base constantが残っている。一方、unreferenced candidate `ccd2544a...` はsource-relevant integration base `40209d25...` 直上でahead 1 / behind 0のcandidateであり、後続docs-only記録commitだけではsource driftと扱わない。revised three-source pressure一式に加え、unknown RelationとQuery cardinality mismatchのfull-Validation negative controlを実装している。残る5 source blobは従来候補と同一で、強化したfull-validation testもactual gofmt PASSとblob identity一致を取得している。runnerのactual `bash -n` PASSとblob identity一致、current Semantic/Validation APIとのstatic source照合も維持している。Persistence未定義もcurrent validator上はそれ自体がerror条件ではない。focused `go test` / reference evidence / full repository health PASSはまだ取得していないため、candidateはExecutable Evidenceではない。

## 12. Conclusion

S6 preflightで見つかったRelation source欠落はProposalへ戻して修正した。

現在の次工程は:

```text
health-checked revised Source pressure
  -> current S5 eligibility review
  -> explicit Acceptance witness
  -> S6 authoritative source authoring
```

である。