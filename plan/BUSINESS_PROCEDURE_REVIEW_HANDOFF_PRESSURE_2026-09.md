# Business Procedure Review / Handoff Pressure — 2026-09

Status: Evidence in progress

## 目的

Business Procedureをtop-level Canonical kindへ追加する前に、前段のReferenceで残った二つのidentity pressureを分離して確認する。

1. 内容変更を伴っても同じBusiness Procedureとしてreview / approval historyを継続できるか
2. Procedure間handoffを表現するためにCanonical Procedure identityが必要か

この文書はWorkflow Engine、Canonical `Procedure`、新しいProduct Requirementを提案するものではない。

## Requirement / Lane境界

`requirements/TEI_PRODUCT_REQUIREMENTS.md`ではRQ-025 / RQ-026がすでに正式要求である。

- RQ-025: Business Action / ProcedureをApplication Functionから独立して保持する
- RQ-026: Data / Function-Interaction / Business Procedureを相互検証する

一方、`plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`のmain基線には現時点でPAS-07は統合されていない。

PAS trace正本の更新はレーンB側へ残し、本レーンCはReference / Evidence圧力だけを所有する。

## 1. identityを三段階に分ける

Reference実験では、次を同一視しない。

```text
A. Canonical business semantic identity
B. Reference-local authoring identity
C. Source / Evidence locator
```

今回、二つのProcedure fixtureの`procedure.id`をBとして使う。

```text
identityStatus = reference-local-stable
canonical = false
```

これは「Canonical Procedureを採用した」という意味ではない。

Reference corpus内でreview target、handoff endpoint、cross-referenceを安定して指すためのauthoring keyである。

## 2. Review / approval continuity

### 2.1 review target

review historyの対象はReference-local Procedure keyで束ねる。

```text
review target
  = procedure.id
```

file pathをreview target identityにはしない。

### 2.2 Evidence stamp

一方、approvalが有効だったsource snapshotは次で固定する。

```text
procedure key
repository revision
artifact fingerprint
source locator
review decision
```

ここでProcedure keyは履歴を束ねるidentityであり、revision / fingerprintは「どのsource snapshotをreviewしたか」を固定するEvidenceである。

### 2.3 content change

同じProcedure keyを保ったままBusiness Action文言等を変更した場合:

```text
review target
  same

artifact fingerprint
  changed

repository revision
  changed

old approval
  not automatically applicable
```

つまり、履歴の連続性とapproval validityを分離できる。

同じBusiness Procedureであることを保つために、旧approvalを新sourceへcarry-forwardしてはいけない。

### 2.4 pure file move

bytesを変えずfile pathだけ移動した場合:

```text
procedure key
  same

fingerprint
  same

source locator
  changed

repository revision
  changed if committed as a new revision
```

semantic target continuityは保てる。

ただしsource-dependent Evidenceはrevision-boundであるため、別revisionへapprovalを黙って継承しない。

carry-forwardを将来許す場合も、明示Policy / Evidenceとして扱う。

### 2.5 logical rename

`procedure.id`自体をrenameした場合は別review targetとして扱う。

これはfile renameと異なる。

```text
file move
  locator change

Procedure key rename
  logical authoring identity change
```

旧review historyを新IDへ関連付けたい場合は、明示migration / rename Evidenceが必要になる。

名称類似から同一性を推測しない。

## 3. Procedure handoff

### 3.1 Reference set

`reference/business-system-design/order-entry-procedure-reference-set.json`を追加した。

このsetはProcedure keyとartifact locatorを分離する。

```text
order-entry.order-handling
  -> order-entry-business-procedure.json

order-entry.follow-up
  -> order-entry-follow-up-procedure.json
```

handoffはartifact pathではなく、Procedure key + Step keyを参照する。

```text
order-entry.order-handling / BP-09
  -> order-entry.follow-up / FU-00
```

condition:

```text
follow-up is required after the earlier order interaction
```

### 3.2 file move

artifact fileを移動してもProcedure keyを保つ場合、Reference setのlocator mappingだけを更新する。

handoff semantic endpointは変えない。

したがってProcedure間handoffを表すためだけなら、Canonical World上のProcedure identityは必須ではない。

### 3.3 logical Procedure rename

handoff targetのProcedure keyをrenameし、handoff linkを更新しなかった場合はfail closedとする。

古いkeyから類似名称を探して自動接続しない。

この性質により、source locator変更とBusiness-side logical renameを区別できる。

## 4. 今回得られた新しいEvidence

前段では「基本cross-viewだけならdurable Procedure identityは不要」と観測していた。

今回、review continuityとhandoffを加えることで、初めて**安定したauthoring-level logical keyには具体的価値がある**ことが確認できた。

ただし必要なのは現時点では次である。

```text
Reference-local stable Procedure key
```

であって、

```text
top-level Canonical Procedure object
```

ではない。

## 5. identity pressure matrix

| Pressure | Reference-local key | Canonical Procedure |
|---|---|---|
| Business Action -> Pattern cross-view | optional |不要|
| multiple Procedure -> same Pattern |有用|不要|
| manual residual authoring |有用|未要求|
| review history continuity |必要性あり|未要求|
| approval snapshot binding |key + revision + fingerprint|不要|
| pure file move |keyでcontinuity、locator更新|不要|
| Procedure間handoff |必要性あり|未要求|
| Runtime orchestration |本Reference対象外|導入理由にならない|
| 外部system間のdurable Business Procedure contract |未検証|未解決|

## 6. Source Indexとの関係

既存`subject.Subject`はpath-independentなCanonical / logical targetをSource locatorから分離する。

今回のReference-local Procedure keyも同じ分離原則を利用するが、現時点で`subject.Subject{Kind:"procedure"}`をCanonical Source Indexへ追加しない。

理由:

- Business Procedure fixtureはまだReference experimentである
- accepted file-native source formが未確定
- stable authoring keyが必要というEvidenceと、Canonical World subjectにすべきというEvidenceは別である

Source Indexへの正式接続は、authoritative Business Procedure sourceを採用するときに再評価する。

## 7. 000039 / Shared Acceptanceとの境界

`ISSUE-000039`はrevision-aware semantic diff / dependency propagationを将来所有する。

本Referenceはそのgraphを先取りしない。

今回固定するのは、change analysisが将来使えるidentity境界である。

```text
same Procedure key + changed fingerprint/revision
  = same review target, changed source snapshot

changed Procedure key
  = logical identity change
```

Shared Acceptanceについても、approval recordやreview key自体をCanonical meaningへ混ぜない。

## 8. 次に確認すること

今回の二つのpressureから、Canonical Procedureの是非より先に次を確認すべきことが明確になった。

1. Business ProcedureをReference fixtureからfile-native accepted authoring sourceへ進めた場合、Reference-local stable keyでreload / Source Index / Shared Acceptanceが成立するか
2. Business-only changeをShared Acceptanceへ通した後、derived Matrix / diagnosticを再生成して意味差を確認できるか
3. handoffを複数Subject Area / Subsystemへ広げてもreference-local identityで十分か
4. external contractや長期reuseでCanonical Procedure subjectが必要になる具体例が出るか

これらで不足が反復して実証されるまでは、Canonical `Procedure` / `Step`を追加しない。

## Verification

主なtest:

- `TestReferenceLocalStableProcedureKeySupportsHandoff`
- `TestHandoffSurvivesArtifactMoveWhenProcedureKeysStayStable`
- `TestHandoffFailsClosedAfterLogicalProcedureRenameWithoutLinkUpdate`
- `TestReviewTargetContinuesAcrossContentChangeButApprovalDoesNotCarryForward`
- `TestReviewTargetSurvivesPureFileMoveButEvidenceRemainsRevisionBound`
- `TestLogicalProcedureRenameCreatesDifferentReviewTarget`

actual Repository runnerがcheckout / Go testへ到達するまではEvidence in progressとする。
