# ISSUE-000066: Gate Bでactual Oracleを必須blockerとする条件を再監査する

- 状態: In Progress — 結論反映中
- GitHub Issue: #381
- 対象Gate: Gate B「UIなしで主要Scenarioを再現できる」
- 親Issue: `ISSUE-000006` / #69
- 監査Issue: `ISSUE-000029` / #273
- 関連Issue: `ISSUE-000022` / #214、`ISSUE-000033` / #371、`ISSUE-000034` / #376
- 起点: PR #379 `PRODUCT_REQUIREMENT_GAP_AUDIT`更新

## 目的

#214 actual external Oracle Evidenceを、Gate B通過前に必ず完了しなければならないProduct Evidenceとして扱う現在契約が、Gate B本来の目的とProduct Requirementsに対して本当に必要かを再監査します。

Oracle固有Evidenceの価値を否定するための監査ではありません。

問いは次です。

> 第一Web Targetへ進む前に、actual Oracle実接続まで待つことが、Gate Bが減らしたい不確実性に必要か。

## 再監査の起点

PR #379で`PRODUCT_REQUIREMENT_GAP_AUDIT.md`を現行v1主線へ更新した結果、現在のProduct優先順位は次になりました。

1. #310 / #255 / #308の、外部Oracleに依存しないGate B Evidenceを先に取得する
2. actual external OracleをGate B必須blockerとする妥当性を、Gate B本来の目的から再監査する
3. Gate B ReadyならHandover Baselineを固定し、第一Web Targetへ進む

Provider / Database領域は他workstreamよりEvidenceが深く、実環境情報がない特定Providerの追加深掘りだけでProduct全体の学習を止めないことが背景です。

## 上位契約から見たGate Bの目的

### `MILESTONES.md`

Milestone 3 / Gate Bの中心は次です。

- ApplicationのData / Rule / Validation / CapabilityをDriverで実行できる
- Database Bindingを含む小さい業務Scenarioを**閉じた環境で再現**できる
- Authentication / AuthorizationをApplication meaningから評価できる
- Business Result / Execution Error / Observationを区別できる
- unknown / unresolvedを安全に保持できる
- Reference Scenarioを**Repositoryだけから再現**できる

Gate Bは「UIなしで主要な業務Scenarioを再現できる」ことを確認するGateです。

特定Oracle環境への実接続完了はMilestone 3の到達条件に書かれていません。

### `V1_REFERENCE_APPLICATION_PLAN.md`

Gate B Ready条件は次の意味境界を要求します。

- 同じApplication定義から主要Scenarioを一続きに再現できる
- 同じ業務identityを境界間で継承できる
- Pattern / Query / Relation / Contract Query / Datasource / Capability / Bindingを追跡できる
- external assetの未観測事実を推測しない
- read-only / secret境界を保持する
- Business Result / Execution Error / Observationを区別する
- synthetic Evidenceと実外部Evidenceを区別する
- Query 0件とExecution Errorを区別する
- Reference artifactを再現できる
- 第一Web TargetへWeb側の意味補完なしにbaselineを渡せる

ここでも「actual Oracle成功」が定義そのものではありません。

### Product Requirements

Gate Bと関係が深いRequirementにも、特定Oracle実接続を第一Target前の必須条件とする要求はありません。

- RQ-005: Application ↔ Database N:M、Datasource / Contract / Binding /利用形態 / failure等を独立して扱えること
- RQ-007: provider-neutral meaningとprovider固有実現、offline policyとactual connection処理を分離できること
- RQ-012: Driverで設計途中Applicationを実行・simulationできること
- RQ-021: 重要境界をReference / fixture / Conformanceから再現できること

actual OracleはRQ-004 / RQ-005 / RQ-007等をさらに強く実証する有用なEvidenceですが、Gate Bの目的と同一ではありません。

## 現在すでに分離できているEvidence

### #310 deterministic Repository

通常Go runnerで次を確認します。

- Canonical Workspace Validation
- 最終UI-less Scenario
- external Evidence intake Conformance
- Repository source固定Conformance

reference Customer pathを含む主要Scenarioの意味・identity・result boundaryを、Repositoryだけから再現するEvidenceです。

syntheticなのでactual Oracle Evidenceには格上げしません。

### #308 real PostgreSQL Provider

実PostgreSQL engineを通して、Database instance / executor Capability / native SQL / parameter / explicit Semantic mapping / 0件 / Execution Errorを確認します。

これにより「provider runtimeが実engineでも成立する」不確実性を、actual Oracleとは独立に減らせます。

PostgreSQL EvidenceをOracle Evidenceへ格上げしません。

### #214 external Evidence intake boundary

actual Oracle factsがまだなくても、TEIが外部事実をどう受け入れるかのfail-closed境界は成立しています。

- external EvidenceをCanonicalへ昇格しない
- SQL realizationがなければEvidenceから補完しない
- provenance / read-only Evidence参照を要求する
- native source fingerprint staleを検知する
- source-fixed acceptance receiptを作れる

#310ではこのintake boundaryのConformanceを通常runnerから再検証します。

したがって「actual Oracleがないから未観測物理情報をWeb側で補う」という経路をGate B通過条件から独立して禁止できます。

## Handoverでunresolvedを保持できる

`GATE_B_TO_C_HANDOVER_CONTRACT.md`は、Gate B通過を妨げない継続事項を`unresolved`として保持する契約を既に持っています。

actual OracleをGate B後へ継続する場合、少なくとも次をHandover unresolvedへ明記できます。

- #214 actual external Oracle Evidence: 未取得
- #371 actual Oracle driver / Host assembly: 未確定
- #376 actual Oracle read-only Evidence: 未取得
- Oracle compatibility / actual external asset integrationを実証済みとは主張しない
- Web側でOracle physical object / datatype / read-only方式 / driver都合を補わない

未検証を消さずに第一TargetのProduct学習へ進めます。

## 判定

### 結論

**#214 actual Oracle EvidenceをGate Bの必須完了laneから外します。**

Gate Bを通過するためのsource-dependent実行Evidenceは、次の3 laneへ収束させます。

1. #310 deterministic Repository / normal Go runner
2. #255 generated artifact reproducibility
3. #308 real PostgreSQL Provider Evidence

加えて、#310の一部としてexternal Evidence intake boundaryを通常runnerで確認し、外部事実を推測しないfail-closed契約を必須とします。

### #214の位置づけ

#214 / #371 / #376は廃止しません。

次を主張するための**継続actual external asset Evidence**としてOpen / Blockedのまま保持します。

- actual Oracle compatibility
- `master_data`実物理資産とのintegration
- actual Oracle driver / Host assembly
- actual read-only principal / grant / transaction境界
- actual read → Semantic Customer

actual Oracle実環境情報が得られた時点で継続します。

### Gate B Readyでも主張しないこと

#214未完了のままGate BがReadyになっても、次は主張しません。

- Oracle Providerがactual環境で動作確認済み
- `master_data` actual assetが接続確認済み
- Oracle datatype / NULL / named bind挙動が確認済み
- Oracle read-only principal / grantが確認済み
- #371 / #376が完了済み

Gate B Readyはあくまで、第一Web Targetへ渡すApplication / Driver meaningと、少なくとも一つのreal provider runtimeをEvidenceで閉じたことを意味します。

## 改訂後のcandidate revision契約

最終Gate B Ready判定では、一つのcandidate repository revision `R`へ次を揃えます。

- #310 `repositoryRevision = R` / `reproducibleRevision = true`
- #255 `repositoryRevision = R` / `reproducibleRevision = true`
- #308 `repositoryRevision = R` / `reproducibleRevision = true`
- Gate B最終監査がRを対象とする

#214 actual external EvidenceはRへの必須収束対象から外します。

ただし、Handover Baseline Rは次を明示します。

- #214 / #371 / #376 = unresolved / not verified
- external Evidence intake Conformanceは#310で成功している
- Web側でunresolved Oracle事実を補わない

Gate C Evidenceもactual Oracle成功Evidenceへ読み替えません。

## #308を残す理由

actual Oracleを外すからといって、Gate Bをsynthetic Evidenceだけへ縮退させません。

Milestone 3はDatabase Bindingを含む小さいScenarioを動かして確認することを求めます。

#308ではreal PostgreSQL engineを利用し、次を確認できます。

- actual provider / driver boundary
- native SQL
- instance-aware runtime
- parameter binding
- explicit raw → Semantic mapping
- 2件 / 0件 / Execution Error
- source-fixed run
- fixture前後の安全状態

したがって#308をGate Bのreal-provider Evidenceとして維持します。

## 第一Web Targetへ渡す制約

#214未完了でGate Bを通過する場合、HandoverとGate Cでは次を守ります。

1. Customer referenceのCanonical meaning / Driver ScenarioはHandoverする
2. actual Oracle physical realizationをHandover済みとは扱わない
3. Web AdapterにOracle object / column / datatype / read-only規則を書かない
4. Target testではsynthetic / deterministicなCustomer referenceとactual Oracleを区別する
5. Web実装中にactual Oracle固有情報が必要になった場合は#214へ戻し、Web独自補完をしない
6. Gate C ReadyをOracle compatibility Readyへ読み替えない

## 安全性

#214をGate B必須blockerから外す判断は、read-only安全条件を弱める判断ではありません。

むしろactual Oracleを利用していない段階で、未観測のprincipal / grant / datatype / driver挙動を「Gateを通すため」に仮定する圧力をなくします。

実Oracle接続を行う段階では、#371 / #376のfail-closed条件を従来どおり満たします。

## 完了条件

- [x] Gate B本来の目的から#214必須性を再評価した
- [x] Product Requirements / Milestonesとの整合を確認した
- [x] actual OracleをGate B必須にしない理由を明示した
- [x] #214 / #371 / #376を継続Evidenceとして残すclaim境界を定義した
- [x] Handover unresolvedへ残す条件を定義した
- [ ] Gate B Evidence lane定義を3必須lane + Oracle unresolvedへ更新する
- [ ] candidate revision契約を#310 / #255 / #308へ更新する
- [ ] Handover契約を更新する
- [ ] #273 / #69 / #214へ結果を同期する
- [ ] `PRODUCT_REQUIREMENT_GAP_AUDIT.md`へ再監査完了を同期する

## 非目標

- #214 / #371 / #376を削除すること
- actual Oracle Evidenceが不要だと主張すること
- PostgreSQL EvidenceをOracle Evidenceとして扱うこと
- synthetic Customer referenceをactual external readとして扱うこと
- Web側にOracle固有事実を仮置きすること
- Gate BをEvidenceなしで通すこと
- Gate Cをactual Oracle compatibility Gateとして扱うこと

## 関連

- GitHub #69 / #70 / #214 / #255 / #273 / #291 / #308 / #310 / #371 / #376 / #381
- PR #354 / #379
- `plan/PRODUCT_REQUIREMENT_GAP_AUDIT.md`
- `plan/MILESTONES.md`
- `plan/V1_REFERENCE_APPLICATION_PLAN.md`
- `plan/GATE_B_EVIDENCE_ACQUISITION.md`
- `plan/GATE_B_EVIDENCE_RUNBOOK.md`
- `plan/GATE_B_READINESS_AUDIT.md`
- `plan/GATE_B_TO_C_HANDOVER_CONTRACT.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
