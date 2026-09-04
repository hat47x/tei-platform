# ISSUE-000030: Gate BからGate Cへ渡すDriver baselineを固定する

- 状態: Proposed
- 親Issue: `ISSUE-000007` / GitHub Issue #70
- 関連Issue: `ISSUE-000029` / GitHub Issue #273
- 関連RQ: RQ-011 / RQ-012 / RQ-013 / RQ-018 / RQ-021 / RQ-022

## 背景

Gate Cでは、第一Web TargetがGate Bを通過したDriverと同じCanonical Application上の意味を保持しているかをSemantic Conformanceとして確認します。

既存のGate C Evidence契約では`Driver baseline identity`をEvidenceへ残すことを要求していますが、何をbaselineとして固定し、Gate C進行中にDriverが変わった場合にどう扱うかを明示するhandover契約が不足していました。

比較対象を曖昧な「最新Driver」にすると、Web実装の途中でApplication / Driverが変わったとき、どの意味に対するConformanceなのか説明できなくなります。

さらに、Gate Bの各Evidence laneが個別にはsource-fixedでも、異なるrevisionのEvidenceを寄せ集めれば「Gate B Ready」とみなせる余地があると、Handover Baselineを一つのrevisionへ固定する契約と矛盾します。

## 採用方針

[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md)に従い、Gate B最終監査で一つの**candidate Ready revision**を選び、そのrevisionへsource-dependent Gate B Evidenceを揃えます。

Gate BをReadyと判定したcandidate revisionを、そのまま第一Handover Baselineのrepository revisionとして固定します。

Handover Baselineは第二の正本ではありません。次への参照を束ねます。

- Gate B candidate Ready revision / repository revision
- Canonical Application / Pattern / Data / Relation / Query / Capability等のidentity
- Driver Scenario identity
- Business Result / Execution Error / Observation境界
- Datasource / authority境界
- Gate B Evidence
- unresolved事項

## candidate Ready revisionの整合条件

Gate B candidateを`R`とする場合、少なくとも次を同じRへ揃えます。

- #310 deterministic Repository Evidence
- #255 generated artifact reproducibility Evidence
- #308 real PostgreSQL Evidence
- #214 source-fixed acceptance receipt
- Rのauthoring / runtimeを用いた#214 actual Oracle runtime / Semantic E2E

Repository revisionを持つEvidenceでは`repositoryRevision = R`と`reproducibleRevision = true`を要求します。

Oracleのphysical facts / provenance等の外部観測事実自体はGit revisionに属しません。同じ外部Evidence artifactが引き続き有効なら再利用できますが、そのartifactをRのTEI authoringへ受理するreceiptと、Rのruntimeでのactual readはRへ結び直します。

異なるR1 / R2 / R3 / R4のEvidenceを寄せ集めてGate B Ready → Handoverとはしません。

## candidate変更時

candidate RのEvidence取得中にsource変更が必要になった場合、Rを新revisionへ黙って読み替えません。

1. 必要なsource / generated artifactを修正する
2. commitする
3. 新candidate R'を選ぶ
4. source-dependent Gate B EvidenceをR'へ揃え直す
5. Gate B ReadyをR'で再判定する
6. ReadyならR'をHandover Baselineへ固定する

#255のrepair commitも同じ扱いです。

mainが別作業で進んでも、既存candidateを自動的に最新mainへ動かしません。後続commitをbaselineへ含める場合だけ新candidateを選び直します。

## Gate Cで守ること

- Gate C runごとにどのHandover Baselineを比較対象にしたか追跡する
- Web側の都合でApplication意味を補完しない
- Driver変更を既存baselineへ黙って上書きしない
- Application / Driverの意味が変わった場合は新candidateを選び、source-dependent Gate B Evidenceをそのrevisionへ揃えてから新baselineを発行する
- Target固有実装やvisual evaluator結果をbaselineへ混ぜない
- Handover recordからCanonicalへ逆書きしない

## 完了条件

- [x] Handover Baselineの目的と権威関係を文書化する
- [x] baselineに必要な最小情報を定義する
- [x] Gate C Evidenceの`Driver baseline identity`との対応を定義する
- [x] Gate C中のbaseline変更規則を定義する
- [x] Gate B candidate Ready revisionとHandover revisionの同一性を定義する
- [x] 異なるrevisionのGate B Evidenceを混在させない規則を定義する
- [x] candidate変更時のsource-dependent Evidence再取得規則を定義する
- [ ] Gate B Ready時に一つのcandidate revisionへ4 laneのsource-dependent Evidenceを揃える
- [ ] Gate B Ready時の実revisionを使って第一Handover Baselineを作成する
- [ ] #70 / #270の第一Web Target EvidenceからHandover Baselineへ追跡できる
- [ ] Web実装中のApplication / Driver変更時にbaseline更新規則を実証する

## 非目標

- Gate B通過前にmachine-readable schemaを固定すること
- Presentation IRの設計
- Web frameworkの選定
- Git revisionだけでSemantic Conformanceを証明すること
- 異なるrevisionのGate B Evidenceを寄せ集めてReady扱いすること
- mainの最新revisionへbaselineを自動追随させること
- 第一TargetだけでTarget-neutral taxonomyを完成させること

## 関連

- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/issue/000007-first-web-target-projection.md`
- `plan/issue/000028-first-web-target-conformance-evidence.md`
- `plan/issue/000029-gate-b-readiness-audit.md`