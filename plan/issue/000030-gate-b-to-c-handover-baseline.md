# ISSUE-000030: Gate BからGate Cへ渡すDriver baselineを固定する

- 状態: Proposed
- GitHub Issue: #291
- 親Issue: `ISSUE-000007` / GitHub Issue #70
- 関連Issue: `ISSUE-000029` / GitHub Issue #273
- Gate B scope再監査: `ISSUE-000035` / GitHub Issue #380
- 関連RQ: RQ-011 / RQ-012 / RQ-013 / RQ-018 / RQ-021 / RQ-022
- Product Acceptance Scenario: PAS-02 primary / PAS-03 secondary

## 背景

Gate Cでは、第一Web Targetが、Gate Bを通過したDriverと同じCanonical Application上の意味を保っているかをSemantic Conformanceとして確認します。

既存のGate C Evidence契約では`Driver baseline identity`をEvidenceへ残すことを要求しています。しかし、比較対象を曖昧な「最新Driver」にすると、Web実装の途中でApplication / Driverが変わったとき、どの意味に対するConformanceなのかを説明できなくなります。

また、Gate Bの必須Evidenceがそれぞれsource-fixedになっていても、**別々のrevisionで成功したEvidenceを寄せ集めて、一つのGate B Ready revisionへ読み替えることはできません**。

## 2026-09-04 現行scope（ISSUE-000035 / #380）

`ISSUE-000035` / #380の再監査により、Gate B Ready判定へ必須とするsource-dependent実行Evidenceは次の三つへ整理しました。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL Provider Evidence

Handover Baseline revisionを`R`とする場合、この三つの完了Evidenceを同じ`R`へ揃えます。

`ISSUE-000022` / #214、`ISSUE-000033` / #371、`ISSUE-000034` / #376のactual Oracle EvidenceはGate Bの必須blockerではありません。Handover時に完了済みならverified external asset Evidenceとして参照し、未完了なら`unresolved`として明示します。

未完了の場合、少なくとも次を隠しません。

- `master_data` actual Oracle readが未実証であること
- actual physical object / raw columns / native SQL等が未観測であること
- actual driver integration / read-only実効保証が未完了であること

Web側でこれらを推測・補完せず、#310 synthetic Evidenceや#308 PostgreSQL Evidenceをactual Oracle Evidenceへ格上げしません。

## 採用方針

[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md)に従い、Gate B通過時のApplication / Driverを一つのHandover Baselineとして固定します。

Handover Baselineは第二の正本ではありません。次への参照を束ねます。

- 単一repository revision
- Canonical Application / Pattern / Data / Relation / Query / Capability等のidentity
- Driver Scenario identity
- Business Result / Execution Error / Observation境界
- Datasource / authority境界
- Gate B Ready判定に使う3 Required Evidence
- 必要に応じてactual Oracle等のverified external asset Evidence、またはその`unresolved`
- Gate B最終監査
- unresolved事項

## 3 Required Evidenceを同じrevisionへ揃える

Gate Bの最終Ready判定では、次の三つについて、完了判定に用いるEvidenceをHandover Baseline revision `R`へ揃えます。

1. #310 deterministic Repository Evidence
2. #255 generated artifact reproducibility Evidence
3. #308 real PostgreSQL engine + ephemeral fixture Evidence

少なくとも次を要求します。

- #310 deterministic Repository Evidence: `repositoryRevision = R` / `reproducibleRevision = true`
- #255 generated artifact Evidence: `repositoryRevision = R` / `reproducibleRevision = true`
- #308 Provider Evidence: `repositoryRevision = R` / `reproducibleRevision = true`
- Gate B最終監査: RをReady対象revisionとして判定すること

Rと異なるrevisionの完了EvidenceしかないRequired Evidenceは、「意味は変わっていないはずだ」と推測して流用せず、Rで再実行・再受入れします。

### actual Oracle Evidenceは第四のRequired laneにしない

#214 / #371 / #376のactual Oracle Evidenceは、取得済みならHandover Baselineからverified external asset Evidenceとして参照できます。ただし、Gate B Ready判定に必要な第四laneにはしません。

Oracle EvidenceをHandover Baseline `R`に対するverified claimとして扱う場合は、actual observed artifact、acceptance receipt、actual read、対象revision / realizationの関係を明示します。古いrevisionの成功を黙って`R`の成功へ読み替えません。

external observed artifactそのものの`observedAt`が`R`の時点である必要はありません。以前に観測した資料を使えます。ただし、その資料を`R`に対するverified claimとして扱う受入れとactual readのprovenanceは明示します。

`R`への十分な結び付きが確認できない場合、またはactual read自体が未完了なら、その範囲を`unresolved`として残します。#310 synthetic Evidenceや#308 PostgreSQL Evidenceで代替したことにはしません。

## Gate B Evidenceの参照

Handover recordへEvidence本文を複製せず、安定した参照を束ねます。

### Required Evidence

- #310 `reference-gate-b-repository-evidence`成功JSON
- #255 `reference-generated-artifact-evidence` check-only成功JSON
- #308 `reference-postgres-header-detail-runtime`成功JSON
- `GATE_B_READINESS_AUDIT.md`の最終Ready判定

#255では、`-write`による修復操作を完了Evidenceにしません。

#310では、少なくとも次の3 targeted checkの成功へ戻れる必要があります。

- `final-ui-less-scenario`
- `external-evidence-intake`
- `repository-evidence-source-boundary`

### Optional / unresolved external Evidence

#214等のactual Oracle Evidenceが取得済みなら、少なくとも次を区別して参照します。

- external Evidence artifact / `evidenceArtifactFingerprint`
- source-fixed acceptance receipt
- actual Oracle read Evidence
- actual readが対象としたruntime / realization / repository revision

acceptance receiptだけをactual Oracle read Evidenceへ格上げしません。

未取得またはHandover Baseline `R`に対するverified claimとして不足がある場合は、その不足を`unresolved`へ記録します。

## Gate Cで守ること

- Gate C runごとに、どのHandover Baselineを比較対象にしたか追跡する
- Web側の都合でApplicationの意味を補完しない
- Driver変更を既存baselineへ黙って上書きしない
- Application / Driverの意味が変わった場合はGate Bに関係するEvidenceを再確認し、3 Required Evidenceを新しいrevisionへ揃えてから新baselineを発行する
- actual Oracle等のexternal Evidenceは、取得状況と対象revisionを明示し、Required Evidenceへ暗黙に混ぜない
- Target固有実装やvisual evaluatorの結果をbaselineへ混ぜない
- Handover recordからCanonicalへ逆書きしない

## 完了条件

- [x] Handover Baselineの目的と権威関係を文書化する
- [x] baselineに必要な最小情報を定義する
- [x] Gate C Evidenceの`Driver baseline identity`との対応を定義する
- [x] Gate C中のbaseline変更規則を定義する
- [x] 3 Required Evidenceの完了Evidenceを単一baseline revisionへ揃える規則を定義する
- [x] #310 / #255 / #308のRequired Evidenceと、#214等のoptional / unresolved external Evidenceをhandover参照上で分離する
- [ ] Gate B Ready時の実revisionを使って第一Handover Baselineを作成する
- [ ] #310 / #255 / #308の完了Evidenceが第一Handover Baseline revisionへ揃っていることを実証する
- [ ] #214が未完了または対象revisionへの結び付きが不足する場合、その未検証範囲を`unresolved`として実際に記録する
- [ ] #70 / #270の第一Web Target EvidenceからHandover Baselineへ追跡できる
- [ ] Web実装中のApplication / Driver変更時にbaseline更新規則を実証する

## 非目標

- Gate B通過前にmachine-readable schemaを固定すること
- Presentation IRの設計
- Web frameworkの選定
- Git revisionだけでSemantic Conformanceを証明すること
- Evidenceが異なるrevisionで成功したまま一つのbaselineへ読み替えること
- actual Oracle EvidenceをGate B Readyの第四Required laneへ戻すこと
- external-observed assetの未観測事実をsynthetic / PostgreSQL Evidenceから推測すること
- 外部観測資料の観測時刻をbaseline Git revisionへ無理に合わせること
- 第一TargetだけでTarget-neutral taxonomyを完成させること

## 関連

- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/UPSTREAM_ACCEPTANCE_SCENARIO_TRACEABILITY.md`
- `plan/UPSTREAM_TRACE_AUDIT_2026-09.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000028-first-web-target-conformance-evidence.md`
- `plan/issue/000029-gate-b-readiness-audit.md`
- `plan/issue/000035-gate-b-external-provider-scope-audit.md`
