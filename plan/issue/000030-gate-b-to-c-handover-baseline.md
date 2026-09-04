# ISSUE-000030: Gate BからGate Cへ渡すDriver baselineを固定する

- 状態: Proposed
- 親Issue: `ISSUE-000007` / GitHub Issue #70
- 関連Issue: `ISSUE-000029` / GitHub Issue #273
- 関連RQ: RQ-011 / RQ-012 / RQ-013 / RQ-018 / RQ-021 / RQ-022

## 背景

Gate Cでは、第一Web TargetがGate Bを通過したDriverと同じCanonical Application上の意味を保持しているかをSemantic Conformanceとして確認します。

既存のGate C Evidence契約では`Driver baseline identity`をEvidenceへ残すことを要求していますが、何をbaselineとして固定し、Gate C進行中にDriverが変わった場合にどう扱うかを明示するhandover契約が不足していました。

比較対象を曖昧な「最新Driver」にすると、Web実装の途中でApplication / Driverが変わったとき、どの意味に対するConformanceなのか説明できなくなります。

## 採用方針

[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md)に従い、Gate B通過時のApplication / Driverを一つのHandover Baselineとして固定します。

Handover Baselineは第二の正本ではありません。次への参照を束ねます。

- repository revision
- Canonical Application / Pattern / Data / Relation / Query / Capability等のidentity
- Driver Scenario identity
- Business Result / Execution Error / Observation境界
- Datasource / authority境界
- Gate B Evidence
- unresolved事項

## Gate Cで守ること

- Gate C runごとにどのHandover Baselineを比較対象にしたか追跡する
- Web側の都合でApplication意味を補完しない
- Driver変更を既存baselineへ黙って上書きしない
- Application / Driverの意味が変わった場合はGate Bに関係するEvidenceを再確認し、新しいbaselineを発行する
- Target固有実装やvisual evaluator結果をbaselineへ混ぜない
- Handover recordからCanonicalへ逆書きしない

## 完了条件

- [x] Handover Baselineの目的と権威関係を文書化する
- [x] baselineに必要な最小情報を定義する
- [x] Gate C Evidenceの`Driver baseline identity`との対応を定義する
- [x] Gate C中のbaseline変更規則を定義する
- [ ] Gate B Ready時の実revisionを使って第一Handover Baselineを作成する
- [ ] #70 / #270の第一Web Target EvidenceからHandover Baselineへ追跡できる
- [ ] Web実装中のApplication / Driver変更時にbaseline更新規則を実証する

## 非目標

- Gate B通過前にmachine-readable schemaを固定すること
- Presentation IRの設計
- Web frameworkの選定
- Git revisionだけでSemantic Conformanceを証明すること
- 第一TargetだけでTarget-neutral taxonomyを完成させること

## 関連

- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000028-first-web-target-conformance-evidence.md`
