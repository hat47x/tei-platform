# ISSUE-000030: Gate BからGate Cへ渡すDriver baselineを固定する

- 状態: Proposed
- GitHub Issue: #291
- 親Issue: `ISSUE-000007` / GitHub Issue #70
- 関連Issue: `ISSUE-000029` / #273、`ISSUE-000035` / #381
- 関連RQ: RQ-011 / RQ-012 / RQ-013 / RQ-018 / RQ-021 / RQ-022

## 背景

Gate Cでは、第一Web Targetが、Gate Bを通過したDriverと同じCanonical Application上の意味を保っているかをSemantic Conformanceとして確認します。

既存のGate C Evidence契約では`Driver baseline identity`をEvidenceへ残すことを要求しています。しかし、比較対象を曖昧な「最新Driver」にすると、Web実装の途中でApplication / Driverが変わったとき、どの意味に対するConformanceなのかを説明できなくなります。

また、Gate Bの必須Evidenceがそれぞれsource-fixedになっていても、**別々のrevisionで成功したEvidenceを寄せ集めて、一つのGate B Ready revisionへ読み替えることはできません**。

PR #379を起点とした`ISSUE-000035` / #381の再監査により、actual OracleはGate Bの必須完了条件から外し、Handoverへverified / unresolved状態を明示する継続Evidenceへ変更しました。

## 採用方針

[`GATE_B_TO_C_HANDOVER_CONTRACT.md`](../GATE_B_TO_C_HANDOVER_CONTRACT.md)に従い、Gate B通過時のApplication / Driverを一つのHandover Baselineとして固定します。

Handover Baselineは第二の正本ではありません。次への参照を束ねます。

- 単一repository revision
- Canonical Application / Pattern / Data / Relation / Query / Capability等のidentity
- Driver Scenario identity
- Business Result / Execution Error / Observation境界
- Datasource / authority境界
- Gate Bの3必須Evidence
- #214 actual external Oracleのverified / unresolved状態
- Gate B最終監査
- unresolved事項

## 3必須Evidenceを同じrevisionへ揃える

Gate Bの最終Ready判定では、次の3 Evidenceについて、完了判定に用いるEvidenceをHandover Baseline revision `R`へ揃えます。

1. #310 deterministic Repository
2. #255 generated artifact reproducibility
3. #308 real PostgreSQL engine + ephemeral fixture

少なくとも次を要求します。

- #310 deterministic Repository Evidence: `repositoryRevision = R` / `reproducibleRevision = true`
- #255 generated artifact Evidence: `repositoryRevision = R` / `reproducibleRevision = true`
- #308 Provider Evidence: `repositoryRevision = R` / `reproducibleRevision = true`
- Gate B最終監査: RをReady対象revisionとして判定すること

Rと異なるrevisionの完了Evidenceしかない場合は、「意味は変わっていないはずだ」と推測して流用せず、Rで再実行します。

## #214 actual external Oracleの扱い

#214 / #371 / #376は廃止しません。actual Oracle compatibility / actual `master_data` integrationを主張するための継続Evidenceです。

Handoverでは次のどちらかを明示します。

### verified

Handover時点でactual Oracle Evidenceが完了している場合:

- external Evidence artifact / `evidenceArtifactFingerprint`
- source-fixed acceptance receipt
- #371 actual Oracle driver / Host assembly
- #376 actual Oracle read-only Evidence
- actual Oracle read → Semantic Customer
- Handover Baseline Rへのsource traceability

へ戻れるようにします。

### unresolved / not verified

未完了の場合でもGate Bを自動的にNot Readyへ戻しません。その代わり次をHandover unresolvedへ明示します。

- #214 actual external Oracle Evidence: not verified
- #371 actual Oracle driver / Host assembly: 未確定または未検証
- #376 actual Oracle read-only Evidence: 未取得
- Oracle compatibility / actual `master_data` integrationを実証済みとは主張しない
- Web側でOracle object / column / datatype / read-only方式 / driver都合を補わない
- #310 synthetic Evidenceや#308 PostgreSQL EvidenceをOracle Evidenceへ読み替えない

## Gate B Evidenceの参照

Handover recordへEvidence本文を複製せず、安定した参照を束ねます。

- #310 `reference-gate-b-repository-evidence`成功JSON
- #255 `reference-generated-artifact-evidence` check-only成功JSON
- #308 `reference-postgres-header-detail-runtime`成功JSON
- #214 actual Oracleのverified / unresolved状態
- `GATE_B_READINESS_AUDIT.md`の最終Ready判定

#255では、`-write`による修復操作を完了Evidenceにしません。

#310では、少なくとも次の3 targeted checkの成功へ戻れる必要があります。

- `final-ui-less-scenario`
- `external-evidence-intake`
- `repository-evidence-source-boundary`

`external-evidence-intake`は、actual Oracleが未完了でも外部事実を推測しないfail-closed境界を通常runnerから確認するために必須です。

#308ではreal provider engineで2件 / 0件 / Execution Errorとfixture前後安全性へ戻れる必要があります。

## Gate Cで守ること

- Gate C runごとに、どのHandover Baselineを比較対象にしたか追跡する
- Web側の都合でApplicationの意味を補完しない
- actual Oracle unresolved時にOracle固有物理情報をWeb側で補わない
- Gate C成功をactual Oracle compatibility成功へ読み替えない
- Driver変更を既存baselineへ黙って上書きしない
- Application / Driverの意味が変わった場合はGate Bに関係する必須Evidenceを再確認し、3 Evidenceを新しいrevisionへ揃えてから新baselineを発行する
- Target固有実装やvisual evaluatorの結果をbaselineへ混ぜない
- Handover recordからCanonicalへ逆書きしない

## 完了条件

- [x] Handover Baselineの目的と権威関係を文書化する
- [x] baselineに必要な最小情報を定義する
- [x] Gate C Evidenceの`Driver baseline identity`との対応を定義する
- [x] Gate C中のbaseline変更規則を定義する
- [x] 4 Evidence laneを単一revisionへ揃える旧契約を定義した（PR #354、後に#381で再監査）
- [x] Gate B必須収束を#310 / #255 / #308の3 Evidenceへ更新した（ISSUE-000035 / #381）
- [x] #214 actual Oracleをverified / unresolvedとしてHandoverする契約を定義した
- [ ] Gate B Ready時の実revisionを使って第一Handover Baselineを作成する
- [ ] #310 / #255 / #308の完了Evidenceが第一Handover Baseline revisionへ揃っていることを実証する
- [ ] #214 actual Oracleのverified / unresolved状態を第一Handover Baselineへ記録する
- [ ] #70 / #270の第一Web Target EvidenceからHandover Baselineへ追跡できる
- [ ] Web実装中のApplication / Driver変更時にbaseline更新規則を実証する

## 非目標

- Gate B通過前にmachine-readable schemaを固定すること
- Presentation IRの設計
- Web frameworkの選定
- Git revisionだけでSemantic Conformanceを証明すること
- #310 / #255 / #308が異なるrevisionで成功したまま一つのbaselineへ読み替えること
- actual Oracle unresolvedをactual Oracle成功として扱うこと
- PostgreSQL EvidenceをOracle Evidenceとして扱うこと
- Web側でOracle物理情報を仮置きすること
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
- `plan/issue/000035-gate-b-actual-oracle-blocker-reaudit.md`
- GitHub #214 / #273 / #291 / #381
