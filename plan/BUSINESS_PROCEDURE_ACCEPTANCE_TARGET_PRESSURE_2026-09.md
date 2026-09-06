# Business Procedure Acceptance Target Pressure — 2026-09

Status: Evidence in progress / pre-Gate-C pressure only

## 目的

Business Procedureのfile-native authoring sourceをShared Acceptanceの最小原則へ通したとき、Reference-local stable keyだけで安全なPreview / Reject / Accept / reloadを成立させられるかを確認する。

`ISSUE-000027`の本格実装はGate C Ready後である。本PressureはAI Proposal APIや共通Acceptance実装を先取りしない。

確認するのは、Business Procedure identityについて将来のShared Acceptanceが必要とする最小条件だけである。

## 1. 既存Shared Acceptance原則

000027は少なくとも次を要求している。

```text
candidate
  ↓
parse / decode
  ↓
normalize
  ↓
validate
  ↓
architecture / policy check
  ↓
semantic diff / impact
  ↓
acceptance policy
  ↓
authoritative source update
  ↓
reload / conformance
```

また、

- Previewではauthoritative sourceを書かない
- Rejectではauthoritative sourceを書かない
- stale proposalを黙って最新版へ適用しない
- file moveをlogical renameと同一視しない
- Accept後はsourceをreloadして再確認する

という原則を持つ。

本PressureはこのうちBusiness Procedure sourceに直接必要な部分だけを小さく再現する。

## 2. Candidate target

experiment-only candidateは次を持つ。

```text
TargetProcedure
BaseSourceFingerprint
Step
BusinessAction candidate
```

`TargetProcedure`はReference-local stable Procedure keyである。

file pathをtarget identityにしない。

`BaseSourceFingerprint`はこのPressureにおけるsource-specific optimistic concurrency tokenであり、000027のrepository / model-level `baseModelRevision`を置き換えるものではない。

## 3. Procedure registry

Pressureでは小さいauthoring registryを使う。

```text
order-entry.order-handling
  -> procedures/order-handling.json
```

このmappingはSource locatorを解決するだけであり、Procedure keyそのものではない。

file move時はmappingだけを更新する。

```text
order-entry.order-handling
  -> procedures/sales/order-handling.json
```

candidate targetは同じkeyのままである。

## 4. Preview

`previewProcedureAcceptance`は次を行う。

1. Procedure keyからcurrent source locatorを解決
2. current bytes / fingerprintを取得
3. candidateのbase fingerprintと比較
4. strict decode
5. source内Procedure keyとcandidate targetをexact比較
6. candidate Business Actionをmemory上だけへ適用
7. current normalized Worldからbefore / after Projectionを生成
8. sourceを書かずに返す

Business-only BP-04変更では少なくとも次を確認する。

```text
Business-side Projection
  changed

Pattern-derived Application meaning
  unchanged

authoritative source
  unchanged during Preview
```

## 5. Reject

Preview後にAcceptを実行しなければsource bytesは変わらない。

Rejectに特別なrollbackを要求しない。

これはcandidateとauthoritative sourceを分離する000027の原則と整合する。

## 6. stale candidate

Preview時点で

```text
candidate.BaseSourceFingerprint
!=
current source fingerprint
```

ならfail closedにする。

またAccept直前にもsource bytesを再読込し、Previewしたfingerprintとcurrent fingerprintが一致しなければ書かない。

この二段階確認により、

```text
candidate作成
   ↓
別edit
   ↓
old candidate Accept
```

を黙って最新版へ適用しない。

## 7. Accept / reload

Accept時だけcandidate後fixtureをfileへ書く。

書込後:

1. new artifact fingerprintを取得
2. Workspace reload
3. Business Procedure sourceを再decode
4.同じProcedure keyを確認
5. derived Matrix / diagnosticを再生成
6. Preview後Projectionとreload後Projectionが一致することを確認
7. normalized `ir.World`がBusiness-only editで変わっていないことを確認

### 観測

Business Procedure contentをfile-native sourceとして変更・reload・re-deriveする機械的なAcceptance安全性は、top-level Canonical Procedure objectなしでも成立する。

## 8. file move

sourceをpure moveしregistry locatorを更新した後、同じProcedure key / base fingerprintのcandidateをPreviewできる。

```text
qualified authoring target
  same

locator
  changed

content fingerprint
  same
```

file pathだけをtarget identityにしない000027の原則を満たせる。

## 9. logical rename

source内部のProcedure keyをrenameし、registry / candidateが旧keyのままならfail closedにする。

```text
registry target
  order-entry.order-handling

source identity
  order-entry.order-handling-renamed
```

を黙って同一targetとして扱わない。

名称類似から自動repairしない。

## 10. ここまでで不要だったもの

Preview / Reject / stale detection / Accept / reload / file move / logical renameの機械的安全性を成立させるために、次は必要なかった。

- Canonical `Procedure` object
- Canonical `Step` object
- Workflow Engine
- Runtime orchestration state
- Business actor -> Authorization Role mapping
- mandatory UUID
- AI-specific Proposal store

Reference-local stable Procedure keyとsource fingerprintで成立した。

## 11. それでも残る一点: RQ-002 canonical trace target

Shared Acceptanceの機械的安全性とは別に、RQ-002はauthoring sourceからCanonical subjectへのtraceabilityを要求する。

現行`subject.Subject`もTEI World内のsemantic / logical targetを指す契約である。

したがって現在の問題は、

> Acceptを安全に実行できるか

ではなく、

> accepted Business Procedure sourceを、RQ-002上どのCanonical targetへ結ぶか

へ絞られた。

## 12. identity optionの再評価

### Option A: minimal Canonical identity shell

World側へBusiness Procedureのpath-independent IDと、ごく小さいidentity contractだけを持つ。

Business Action詳細、manual instruction、handoff、actor、runtime orchestrationまで同時にWorldへ吸収しない。

このOptionが必要なら、根拠はWorkflow実行ではなく**RQ-002 trace target**である。

### Option B: accepted-design SubjectへSubject contractを拡張

World内Canonical objectに限定せず、accepted file-native design meaningもpath-independent Subjectとして扱う。

この場合、RQ-002の`Canonical subject`表現とSource Index責務を再定義する必要がある。

### Option C: Authoring identity registryを正式化

Reference-local key / source registryをformal authoring identity層とし、Canonical subjectへのlinkは別contractにする。

柔軟だがidentity層が一つ増える。

### Option D: full Canonical Procedure model

現Evidenceでは過剰である。

## 13. 現時点の判断

Pressure結果を次のように分ける。

```text
Shared Acceptance mechanical safety
  Reference-local stable keyで成立

Source move / logical rename separation
  Reference-local stable keyで成立

reload / re-derive
  Reference-local stable keyで成立

RQ-002 Canonical traceability
  未解決
```

したがって、Canonical Procedureを導入するなら、その最初の根拠は「Shared Acceptanceに必要だから」ではない。

現時点で唯一残る強い根拠候補は、**RQ-002とcurrent Subject contractをBusiness Procedureのaccepted sourceへ適用するためのCanonical trace target**である。

## 14. 次の判断Gate

新しいCanonical kind / ADRを作る前に、Option A〜Cを次の基準で比較する。

1. RQ-002を文言どおり満たすか
2. RQ-025 / RQ-026のBusiness meaningをFunctionへ還元しないか
3. file-native sourceがauthorityであることを保てるか
4. Source Index / Shared Acceptance / 000039 impact analysisへ再利用できるか
5. mandatory duplicate authoringを増やさないか
6. Procedure詳細をCoreへ過剰吸収しないか
7. one-maintainer運用で理解・保守可能か
8. 将来のSubject Area / Subsystem handoffへ伸ばせるか

比較Evidenceが揃うまでfull Canonical Procedure schemaを作らない。

## Verification

主なPressure test:

- `TestProcedureAcceptancePreviewDoesNotWriteAuthoritativeSource`
- `TestProcedureAcceptanceAcceptWritesThenReloadsWithoutWorldExpansion`
- `TestProcedureAcceptanceRejectLeavesSourceUntouched`
- `TestProcedureAcceptanceRejectsStaleCandidate`
- `TestProcedureAcceptanceTargetSurvivesFileMoveThroughRegistry`
- `TestProcedureAcceptanceLogicalRenameRequiresExplicitTargetUpdate`

これらはGate C後のShared Acceptance実装ではなく、Business Procedure target identityを比較するためのtest-only probeである。

actual Repository runnerがcheckout / Go testへ到達するまではEvidence in progressとする。
