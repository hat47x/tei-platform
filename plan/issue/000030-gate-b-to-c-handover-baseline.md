# ISSUE-000030: Gate BからGate Cへ渡すHandover Baselineを固定する

- 状態: Done — Baseline `B / R`固定済み、Gate C側でtrace実証済み（`same-B-R-T-and-target-identity`、T = `0ba87a4e`、2026-09-19）
- GitHub Issue: #291
- Gate B監査: `ISSUE-000029` / #273
- Gate B final convergence: `ISSUE-000037` / #387
- 第一Web Target: `ISSUE-000007` / #70
- Gate C Evidence: `ISSUE-000028` / #270
- Product Acceptance Scenario: PAS-02 primary / PAS-03 secondary

## 目的

Gate Bを通過したApplication / Driver meaningを、曖昧な「最新Driver」ではなく**Handover Baseline `B` / baseline repository revision `R`**としてGate Cへ固定します。

HandoverはCanonicalの第二正本ではありません。authoritative source、Canonical identity、Driver Scenario、Gate B Evidence、unresolvedへの参照を束ねます。

## Gate Bから受け取るもの

Required Evidenceは次の三つです。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider + ephemeral fixture Evidence

三つを同じclean candidate revision `R`へ揃えます。

```text
#310 ─┐
#255 ─┼─ repositoryRevision = R
#308 ─┘
       ↓
#387 final convergence
       ↓
same-run child exact Evidence artifacts
       ↓
#273 Readiness Audit for R
       ↓ Ready only
Handover Baseline B / R
```

actual Oracle #214 / #371 / #376は第四Required laneではありません。未完了なら`unresolved`としてHandoverへ渡します。

## same-Rだけでは不十分

Handoverへ参照する#310 / #255 / #308 detailed Evidenceは、単に同じRで成功した任意の三つのJSONではありません。

**#387 final convergenceが実際に同一runで読み、保持したexact child artifacts**をRequired Evidenceとして使います。

固定filename:

```text
deterministic-repository.json
generated-artifact-reproducibility.json
real-postgres-ephemeral-fixture.json
```

Handoverから少なくとも次へ戻れる必要があります。

- #387 final convergence artifact
- `repositoryRevision = R`
- `allLanesSameRevision = true`
- `childArtifactsRetained = true`
- #310 same-run exact artifact
- #255 same-run exact artifact
- #308 same-run exact artifact
- 各childの`evidenceArtifactFingerprint`
- #273 Gate B Ready Audit for R

同じRで後から再実行した別artifactを、#387が実際に読んだartifactへ読み替えません。

PR #461以降、final runner自身もsuccess直前に保存child fileを再openし、regular file / path identity / SHA-256一致を再確認します。ただし#273でも独立に再照合し、runner内検証だけでReadyを宣言しません。

## Handover Baseline B

少なくとも次へtraceできる必要があります。

### Baseline identity

- Handover Baseline identity `B`
- `baselineRepositoryRevision = R`
- Reference Workspace identity

### Canonical meaning

- Application `order_entry`
- Application Pattern
- Data / Relation / Query / Datasource
- Validation
- Authentication / Authorization
- Capability / Binding
- Business Result / named Outcome
- Execution Error
- Observationに使う明示read境界
- Source / Binding traceability

### Driver Scenario

```text
Authentication
 -> browse
 -> record
 -> reference
 -> header-detail
 -> Canonical Validation
 -> Authorization
 -> Command Business Result
 -> separate Observation read
```

Business Result / Execution Error / Observation / DiagnosticをTarget側で一種類へ畳みません。

### Evidence provenance

- #387 final convergence
- #310 same-run detailed Evidence
- #255 same-run detailed Evidence
- #308 same-run detailed Evidence
- child artifact fingerprints
- #273 Gate B Ready Audit
- optional verified actual external Evidence
- unresolved

Handover recordへEvidence本文を複製せず、stable reference / identity / fingerprintから実artifactへ戻れるようにします。

## actual Oracle

#214 / #371 / #376は取得済みなら追加verified external asset Evidenceとして参照できます。

未取得、actual read未完了、またはRへの結び付きが不足する場合は`unresolved`として残します。

第一Web Target側でphysical object / column / SQL / datatype / driver behavior / read-only方式を推測・補完しません。

#310 synthetic Evidenceや#308 PostgreSQL Evidenceをactual Oracle Evidenceへ格上げしません。

## Gate CではB / RとTarget build Tを分ける

第一Web Targetのbuild identityを`T1`とします。

```text
Handover Baseline B
  baselineRepositoryRevision = R
             ↓
First Web Target build T1
             ↓
Gate C Evidence
  compare meaning -> B / R
  observe Target  -> T1
```

`R`と`T1`を一つのrevision fieldへ畳みません。

Gate C Evidenceから少なくとも`B / R / T1`へ戻れる必要があります。

## meaning変更時はsuccessor Handover

Target実装中にbaseline subjectのApplication meaning変更が必要になった場合、旧Bへ黙って継続しません。

1. authoritative sourceへ戻る
2. Application / Driverを変更する
3. 必要なGate B Evidenceを新revision `R2`で取得する
4. #387を`R2`で再実行しsame-run child artifactsを取得する
5. #273で`R2`を再監査する
6. Readyならsuccessor Handover `B2 / R2`を発行する
7. Gate Cの影響Scenarioを再取得する

Web固有field / ruleで不足meaningを隠しません。

## Handover recordの形式

Gate B Ready前に大きなmachine-readable schemaを固定しません。

最初はMarkdown + stable referencesでも、小さいmachine-readable artifactでも構いません。

形式より次を優先します。

- `B / R`を固定できる
- Canonical identityへ戻れる
- Driver Scenarioへ戻れる
- #387 final convergenceへ戻れる
- same-run #310 / #255 / #308 exact artifactsへ戻れる
- child fingerprintを再照合できる
- #273 Ready Auditへ戻れる
- optional external Evidenceと`unresolved`をRequired Evidenceから分離できる
- secretを含まない
- Target固有情報をCanonical baselineへ混ぜない

## 完了条件

- [x] Handover Baselineの目的 / authorityを定義
- [x] 3 Required Evidenceを同じRへ揃える規則を定義
- [x] actual Oracleをoptional / unresolvedとして分離
- [x] #387 same-run child Evidence provenanceをHandoverへ接続
- [x] same-R later rerunをsame-run Evidenceへ読み替えない規則を定義
- [x] baseline `R`とTarget build `T`を分離
- [x] meaning変更時のsuccessor Handover規則を定義
- [x] 実TEI checkoutで#387 actual final runを取得
- [x] #310 / #255 / #308 same-run exact artifactsとfingerprintを#273で監査
- [x] #273がRをGate B Readyと判定
- [x] 実revision Rから第一Handover Baseline Bを作成（[`GATE_B_TO_C_HANDOVER_BASELINE_B.md`](../GATE_B_TO_C_HANDOVER_BASELINE_B.md)、`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）
- [x] actual Oracle未完了範囲を必要に応じて`unresolved`へ記録（Baseline記録内「actual Oracleの扱い」節）
- [x] #70 / #270 Gate C Evidenceから`B / R / T1`へ追跡できる（[Gate C Readiness Audit](../GATE_C_READINESS_AUDIT.md)の`same-B-R-T-and-target-identity`）
- 条件付き（発生せず。`canonical-world-invariant-R-to-T`）: Gate C中のmeaning変更でsuccessor Handoverを実証

Gate Bは **Ready** です（`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）。Handover Baseline `B / R`は固定済みで、第一Web Target（`ISSUE-000007` / #70）へ着手可能です。残る2項目はGate C Evidence取得自体に依存するため、Gate C側の進行に伴い充足します。

## 非目標

- Gate B通過前の汎用Handover schema固定
- Presentation IR設計
- Web framework選定
- Git revisionだけでSemantic Conformanceを証明すること
- same-R別runを#387 same-run Evidenceへ読み替えること
- actual Oracleを第四Required laneへ戻すこと
- external-observed未観測事実をsynthetic / PostgreSQL Evidenceから推測すること
- 第一TargetだけでTarget-neutral taxonomyを完成させること

## 関連

- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000028-first-web-target-conformance-evidence.md`
- `plan/issue/000029-gate-b-readiness-audit.md`
- `plan/issue/000037-gate-b-three-lane-final-evidence.md`
