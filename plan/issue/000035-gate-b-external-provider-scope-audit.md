# ISSUE-000035: Gate Bのactual external Provider必須範囲を再監査する

- 状態: In Progress
- GitHub Issue: #380
- 親Issue: `ISSUE-000006` / GitHub Issue #69
- Gate B監査: `ISSUE-000029` / GitHub Issue #273
- 関連Issue: `ISSUE-000022` / #214、`ISSUE-000031` / #308、`ISSUE-000025` / #255、`ISSUE-000032` / #310

## 目的

Gate Bの本来の目的に照らして、actual external Oracle EvidenceをGate Bの必須blockerとして扱い続ける必要があるかを再監査します。

Gate Bは「UIなしで主要Scenarioを再現できる」ことを確かめ、第一Web Target側でApplication上の意味不足を補わなくてよい状態を作るためのGateです。

一方、現在のGate B監査では、#214 actual Oracle physical facts / read-only / actual readが未取得であることだけでもGate全体がNot Readyになります。

`MASTER_PLAN.md`では、Database / Provider領域が他workstreamより深く検証されており、一つの技術領域だけがProduct全体の進路を決めないよう上位workstreamを並走させる方針を明示しています。

この二つを整合させます。

## Product Requirementとの照合

RQ-004 / RQ-005 / RQ-007 / RQ-009 / RQ-012 / RQ-014を再確認しました。

Product Requirementが求めているのは、たとえば次です。

- native assetを独自DSLへ全面移行せず利用できること
- Application / DatabaseをN:Mで扱えること
- Provider固有知識をCoreへ漏らさないこと
- 既存認証・認可仕様を保持した移行
- Driverで設計途中のApplicationを動かせること
- Application Patternを再利用可能な知識として扱うこと

これらは重要ですが、**第一Web Targetへ進む前にOracle Databaseへのactual E2Eを成功させること**までは要求していません。

## 現在の判断

actual Oracle E2EをGate Bの必須blockerにし続ける必要性は確認できませんでした。

これは「実行環境がないからGateを緩める」という判断ではありません。

Gate Bが証明すべき対象と、Provider-specificなactual external asset実証を分ける判断です。

## Gate Bで必須として残すEvidence

Gate Bでは、次の三つの実行Evidenceを必須とします。

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

これに加えて、external-observed assetについて次の意味境界が成立していることを要求します。

- 未観測のphysical object / column / SQLを推測しない
- external EvidenceをCanonicalへ昇格させない
- explicit realizationとEvidenceを照合する
- actual external Evidence未取得を`unresolved`として保持できる

external Evidence intakeのConformanceは#310で実行します。

## #214の新しい位置づけ

`ISSUE-000022` / #214は削除しません。

actual external Oracle Evidenceは、次を実証する独立したv1作業として継続します。

- external-observed assetのactual physical facts
- provenance
- read-only実効保証
- actual Oracle driver integration
- actual read → explicit Semantic mapping
- 第二Database ProviderによるProvider境界の圧力試験
- legacy / external asset migrationの実証

#371 / #376も#214の子作業として維持します。

ただし、実Oracle環境情報が得られるまで優先度を下げます。

## Gate Cへの引き継ぎ

#214が未完了のままGate Bを通過する場合、Handover Baselineでは未解決事項として明示します。

少なくとも次を隠しません。

- `master_data` actual external Oracle Evidenceが未取得であること
- actual physical object / SQL / raw columnsが未観測であること
- actual Oracle readを成功済みと主張しないこと

第一Web Targetでは、Web固有コードからOracle物理情報を補完しません。

Reference / test用の実行経路を使う場合も、それをactual external Oracle Evidenceへ格上げしません。

Gate C Semantic Evidenceでは、Handover Baselineの`unresolved`を保持したまま比較できることを要求します。

## #214を後段で必須にする条件

#214をGate Bから外すことは、対応を消すことではありません。

少なくとも次のいずれかを満たす前にはactual external Oracle Evidenceを必要とします。

- `master_data`をactual external assetとしてv1 Support対象だと主張する
- legacy / external asset migrationを実証済みと主張する
- 第二Database ProviderによるProvider-neutral境界の圧力試験を完了扱いする

具体的にどのMilestoneへ置くかは、`MILESTONES.md`とv1 Support条件の更新時に確定します。

## Gate Bの最終Evidence収束

Gate Bのcandidate revisionを`R`とする場合、最終判定に用いるsource-dependent Evidenceは次の三laneをRへ揃えます。

```text
#310 -> R
#255 -> R
#308 -> R
```

#214 actual external Oracle EvidenceはGate B Ready判定へ含めません。

ただしHandover Baselineは、#214未完了を`unresolved`として明示的に引き継ぎます。

## 完了条件

- [x] Gate B本来の目的とProduct Requirementを再確認した
- [x] actual Oracle E2EがProduct Requirement上の第一Web Target前必須条件ではないことを確認した
- [x] Gate B必須Evidenceを#310 / #255 / #308へ再整理した
- [x] external-observedの未観測事実を推測しない境界はGate Bに残した
- [x] #214を後段のactual external / legacy / Provider圧力試験として維持する方針を整理した
- [ ] `GATE_B_READINESS_AUDIT.md`を3 laneへ更新する
- [ ] `GATE_B_EVIDENCE_ACQUISITION.md`を3 laneへ更新する
- [ ] `GATE_B_EVIDENCE_RUNBOOK.md`を3 laneへ更新する
- [ ] `GATE_B_TO_C_HANDOVER_CONTRACT.md`へ#214 unresolved引き継ぎを明記する
- [ ] `V1_REFERENCE_APPLICATION_PLAN.md`を更新する
- [ ] `ISSUE-000006` / `ISSUE-000029`を更新する
- [ ] #69 / #214 / #273 / #291へ判断を同期する
- [ ] Gate Bを実EvidenceなしにReadyへ変更しない

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
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- GitHub #69 / #70 / #214 / #255 / #273 / #291 / #308 / #310 / #371 / #376 / #380
