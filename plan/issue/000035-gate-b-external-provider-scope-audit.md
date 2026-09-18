# ISSUE-000035: Gate Bのactual external Provider必須範囲を再監査する

- 状態: Done — scope判断完了
- GitHub Issue: #380（Completed）
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- Gate B監査: `ISSUE-000029` / GitHub Issue #273
- 文書同期follow-up: `ISSUE-000036` / GitHub Issue #383
- 関連Issue: `ISSUE-000022` / #214、`ISSUE-000031` / #308、`ISSUE-000025` / #255、`ISSUE-000032` / #310

## 結論

Gate Bの本来の目的、Product Requirements、`MILESTONES.md`を再監査した結果、**actual external Oracle E2EをGate Bの必須blockerから外します**。

これはOracle Database対応を不要とする判断ではありません。

Gate BはMilestone 3「設計をDriverで動かして確認できる」のEvidence Gateです。第一Web Targetへ進む前に必要なのは、Application / Driver上の主要ScenarioがUIなしで再現でき、Target側で意味不足や未観測事実を勝手に補わなくてよいことです。

RQ-004 / RQ-005 / RQ-007 / RQ-009 / RQ-012 / RQ-014を確認しましたが、第一Web Targetへ進む前に特定Oracle環境へのactual E2E成功を要求する条項はありません。

## Gate Bで必須とするEvidence

Gate Bの必須Evidenceは次の三つです。

1. **#310 deterministic Repository Evidence**
   - Canonical Workspace Validation
   - 最終UI-less Scenario
   - external Evidence intake boundary
   - Repository source固定境界
2. **#255 generated artifact reproducibility**
   - World / Plugin Lock / Durable Run
   - clean revision上のcheck-only Evidence
3. **#308 real PostgreSQL Provider Evidence**
   - real provider engine
   - 2件 / 0件 / Execution Error
   - explicit Semantic mapping
   - fixture前後の安全境界

三laneは相互代替しません。Gate Bのcandidate revisionを`R`とする場合、最終判定に使うEvidenceを同じ`R`へ揃えます。

```text
#310 -> R
#255 -> R
#308 -> R
```

2026-09-19に三laneの実runがすべて同じ`R = e5231839a5dcc62cce1181a2731fcb7c70d16736`で成立し、Gate Bは **Ready** です（[`GATE_B_READINESS_AUDIT.md`](../GATE_B_READINESS_AUDIT.md)参照）。

## external-observed assetについてGate Bに残す境界

actual Oracle E2EをGate B必須から外しても、external-observed assetを雑に扱ってよいわけではありません。

Gate Bでは少なくとも次を守ります。

- 未観測のphysical object / column / SQLを推測しない
- external EvidenceをCanonicalへ昇格させない
- explicit realizationとEvidenceを照合する
- actual external Evidence未取得を`unresolved`として保持できる
- synthetic ScenarioやPostgreSQL Evidenceをactual Oracle成功Evidenceへ格上げしない

external Evidence intakeのConformanceは#310で確認します。

## #214 actual Oracleの新しい位置づけ

`ISSUE-000022` / #214、#371、#376は削除しません。

actual Oracle Evidenceは次の実証として継続します。

### Milestone 2 — 現実の既存業務をそのまま受け止められる

- external-observed assetのactual physical facts
- provenance
- read-only実効保証
- native realization
- actual read → explicit Semantic mapping
- legacy / external asset migrationの実証

### Milestone 8 — Provider / Runtimeの交換可能性を実証する

- Oracle driver integration
- PostgreSQL以外のDatabase Providerによるprovider-neutral境界の圧力試験

実Oracle環境情報が得られるまで、これらのProvider固有作業はApplication → Driver → Targetの主線より優先しません。

## Gate Cへの引き継ぎ

#214が未完了のままGate Bを通過する場合、Handover Baselineへ少なくとも次を`unresolved`として明示します。

- `master_data` actual external Oracle Evidence未取得
- actual physical object / SQL / raw columns未観測
- actual Oracle read未実証
- driver integration未確定
- read-only実効Evidence未取得

第一Web Targetでは、Web固有コードからOracle物理情報やread-only方式を補完しません。

Reference / test用の経路を利用してCustomer referenceをProjectionしても、それをactual external Oracle Evidenceへ格上げしません。

## actual Oracleを必須にする主張

少なくとも次を主張する前には#214のactual Evidenceが必要です。

- `master_data`をactual external assetとしてv1 Support対象とする
- legacy / external asset migrationを実証済みとする
- 第二Database ProviderによるProvider-neutral境界の圧力試験を完了扱いする

## 文書同期

scope判断は本Issueで完了しました。

既存のGate B長文には旧4-lane記述が残っているため、内容を一括圧縮せず、`ISSUE-000036` / #383で設計履歴とfail-closed条件を保持したまま段階的に同期します。

同期完了までは、本Issue / #380のscope判断を古い4-lane必須表現より優先します。

## 完了条件

- [x] Gate B本来の目的を再確認した
- [x] Product Requirementsと照合した
- [x] `MILESTONES.md`と照合した
- [x] actual Oracle E2EをGate B必須から外す判断を確定した
- [x] Gate B必須Evidenceを#310 / #255 / #308へ再整理した
- [x] external-observedの未観測事実を推測しない境界をGate Bへ残した
- [x] #214をMilestone 2 / 8の継続Evidenceとして位置づけた
- [x] #214未完了時のHandover `unresolved`を明示した
- [x] #69 / #214 / #273 / #291 / #70へscope判断を同期した
- [x] 長文同期を`ISSUE-000036` / #383へ分離した
- [x] Gate Bを実EvidenceなしにReadyへ変更していない

## 非目標

- Oracle Database対応を削除すること
- #214 / #371 / #376を完了扱いすること
- PostgreSQL EvidenceをOracle Evidenceと同一視すること
- external-observedの未観測事実を推測すること
- 第一Web Target実装をGate B Ready前に開始すること
- GitHub Actionsを再有効化すること

## 関連

- `plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md`
- `plan/MASTER_PLAN.md`
- `plan/MILESTONES.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- GitHub #69 / #70 / #214 / #255 / #273 / #291 / #308 / #310 / #371 / #376 / #380 / #383
