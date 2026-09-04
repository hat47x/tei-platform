# ISSUE-000030: Gate BからGate Cへ渡すDriver baselineを固定する

- 状態: Proposed
- GitHub Issue: #291
- 親Issue: `ISSUE-000007` / GitHub Issue #70
- 関連Issue: `ISSUE-000029` / GitHub Issue #273
- 関連RQ: RQ-011 / RQ-012 / RQ-013 / RQ-018 / RQ-021 / RQ-022

## 背景

Gate Cでは、第一Web TargetがGate Bを通過したDriverと同じCanonical Application上の意味を保持しているかをSemantic Conformanceとして確認します。

既存のGate C Evidence契約では`Driver baseline identity`をEvidenceへ残すことを要求していますが、比較対象を曖昧な「最新Driver」にすると、Web実装の途中でApplication / Driverが変わったとき、どの意味に対するConformanceなのか説明できなくなります。

また、Gate Bの4 Evidence laneがそれぞれsource-fixedになっても、**別々のrevisionで成功したEvidenceを寄せ集めて一つのGate B Ready revisionへ読み替えることはできません**。

## 採用方針

[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md)に従い、Gate B通過時のApplication / Driverを一つのHandover Baselineとして固定します。

Handover Baselineは第二の正本ではありません。次への参照を束ねます。

- 単一repository revision
- Canonical Application / Pattern / Data / Relation / Query / Capability等のidentity
- Driver Scenario identity
- Business Result / Execution Error / Observation境界
- Datasource / authority境界
- Gate Bの4 lane completion Evidence
- Gate B final audit
- unresolved事項

## 4 Evidence laneのrevision収束

Gate B final Ready判定では、次のcompletion EvidenceをHandover Baseline revision `R`へ収束させます。

1. #214 actual external Oracle
2. #308 real PostgreSQL engine + ephemeral fixture
3. #255 generated artifact reproducibility
4. #310 deterministic Repository

少なくとも次を要求します。

- #214 acceptance receipt: `repositoryRevision = R` / `reproducibleRevision = true`
- #214 actual read: Rのruntime / realizationに対するreadであること
- #308 Provider Evidence: `repositoryRevision = R` / `reproducibleRevision = true`
- #255 generated artifact Evidence: `repositoryRevision = R` / `reproducibleRevision = true`
- #310 deterministic Repository Evidence: `repositoryRevision = R` / `reproducibleRevision = true`
- Gate B final audit: RをReady対象revisionとして判定すること

外部観測資料そのものはRより前に取得されていて構いません。ただし、その資料を現在のauthoringへ受け入れたreceiptとactual readはRへ再固定します。

Rと異なるrevisionのcompletion Evidenceしかないlaneは、意味変更がないことを推測して流用せず、Rで再実行・再受入れします。

## Gate B Evidence参照

Handover recordへEvidence本文を複製せず、安定した参照を束ねます。

- #214 external Evidence artifact / `evidenceArtifactFingerprint` / acceptance receipt / actual Oracle read Evidence
- #308 `reference-postgres-header-detail-runtime`成功JSON
- #255 `reference-generated-artifact-evidence` check-only成功JSON
- #310 `reference-gate-b-repository-evidence`成功JSON
- `GATE_B_READINESS_AUDIT.md`の最終Ready判定

#255では`-write`による修復操作をcompletion Evidenceにしません。

#310では少なくとも次の3 targeted check成功へ戻れる必要があります。

- `final-ui-less-scenario`
- `external-evidence-intake`
- `repository-evidence-source-boundary`

#214 acceptance receiptだけをactual Oracle read Evidenceへ格上げしません。

## Gate Cで守ること

- Gate C runごとにどのHandover Baselineを比較対象にしたか追跡する
- Web側の都合でApplication意味を補完しない
- Driver変更を既存baselineへ黙って上書きしない
- Application / Driverの意味が変わった場合はGate Bに関係するEvidenceを再確認し、4 laneを新しいrevisionへ収束してから新baselineを発行する
- Target固有実装やvisual evaluator結果をbaselineへ混ぜない
- Handover recordからCanonicalへ逆書きしない

## 完了条件

- [x] Handover Baselineの目的と権威関係を文書化する
- [x] baselineに必要な最小情報を定義する
- [x] Gate C Evidenceの`Driver baseline identity`との対応を定義する
- [x] Gate C中のbaseline変更規則を定義する
- [x] 4 Evidence laneのcompletion Evidenceを単一baseline revisionへ収束させる規則を定義する
- [x] #214 / #308 / #255 / #310の現在のEvidence artifact単位をhandover参照へ対応付ける
- [ ] Gate B Ready時の実revisionを使って第一Handover Baselineを作成する
- [ ] 4 laneのcompletion Evidenceが第一Handover Baseline revisionへ収束していることを実証する
- [ ] #70 / #270の第一Web Target EvidenceからHandover Baselineへ追跡できる
- [ ] Web実装中のApplication / Driver変更時にbaseline更新規則を実証する

## 非目標

- Gate B通過前にmachine-readable schemaを固定すること
- Presentation IRの設計
- Web frameworkの選定
- Git revisionだけでSemantic Conformanceを証明すること
- Evidenceが異なるrevisionで成功したまま一つのbaselineへ読み替えること
- 外部観測資料の観測時刻をbaseline Git revisionへ無理に合わせること
- 第一TargetだけでTarget-neutral taxonomyを完成させること

## 関連

- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000028-first-web-target-conformance-evidence.md`
- `plan/issue/000029-gate-b-readiness-audit.md`
