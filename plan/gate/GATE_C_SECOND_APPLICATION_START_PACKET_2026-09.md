# Gate C → Second Application Reference Start Packet

- 状態：進行中。S0〜S3完了／S4は全体検証の事前確認後に改訂／S5はレビューの再生成待ち
- 日付：2026年9月19日
- 開始後の主担当：ISSUE-000040
- 入力：ISSUE-000047／ISSUE-000053の第5段階／ISSUE-000027／ISSUE-000039
- 判定段階より前の契約：
  - `plan/workstreams/KNOWLEDGE_PORTABILITY_R0_BOUNDARY_2026-09.md`
- 判定段階CのReady前に正本のアプリケーションへ与える影響：なし

## 1. 目的

判定段階CのReady後に第二の参照アプリケーションへ着手する際、探索・知識の移植可能性・構成案・受理を別々の作業正本に分けません。ISSUE-000040の一つの作業として継続できるようにします。

この文書は、第二アプリケーションのソースではありません。

```text
Start Packet
  != Gate C Ready
  != Application Proposal
  != Shared Acceptance
  != Canonical source
```

## 2. Entry Gate

開始前の機械的な確認には、次を正本として用います。

- `plan/gate/GATE_C_SECOND_APPLICATION_ENTRY_PREFLIGHT_2026-09.md`
- `tools/reference-business-procedure-check/gate_c_second_application_entry_preflight_pressure_test.go`

第二アプリケーションの作業を開始するには、判定段階CのReadyを**明示的な検証根拠**で確認します。

計画文書の見出しや将来の予定からReadyと推測しません。

開始時には、次の情報を一つの開始記録に固定します。

```text
gateC.status = ready

baseline:
  handover baseline B
  baseline repository revision R

first target:
  Web build revision T1

evidence:
  semantic conformance = <ref>
  interaction / functional = <ref>
  visual fidelity = <ref>

target differences:
  <explicit summary>

unresolved:
  <remaining items>
```

### Stop

次のいずれかに該当する場合、第二アプリケーションのソースの記述へ進みません。

- 判定段階Cの状態がReadyではない
- `R`と`T1`を区別できない
- 意味の検証根拠がない
- 機能の検証根拠がない
- 表示の検証根拠がない
- 実行先の違いが整理されておらず、第二アプリケーションの設計へ混入する可能性がある

表示の検証根拠だけで開始条件を満たしたとは扱いません。

## 3. Development-flow / workstream continuity

この文書が定める単独開発の進め方では、ブランチやPRを製品の状態やレビューの境界として使わず、mainへ直接コミットします。

第二の参照アプリケーションの作業の継続性は、ブランチではなく、ISSUE-000040、構成案の照合値、ソースと検証根拠、受理の状態によって管理します。

原則は次のとおりです。

```text
current main
  ↓
ISSUE-000040 qualification / portability
  ↓
composition proposal
  ↓
Shared Acceptance boundary
  ↓ Accept後のみ
authoritative source
  ↓
reload / validation
  ↓
Environment / Driver / Target evidence
  ↓
change propagation Evidence
```

### 禁止

- 探索・移植可能性・パターン・環境・ドライバーごとにリモートブランチを作ること
- 同じ第二アプリケーションの正本候補を複数の作業系列で並行して記述すること
- ブランチ名やPRの状態を受理や正本の権限とみなすこと
- 並行するセッションが保持する古いmainの状態から、そのまま変更を適用すること
- 健全性確認をせずにコードの変更をmainへ入れること

各作業ではpushの直前に現在のmainへのrebaseまたは最新状態の確認を行い、コードを変更する場合はリポジトリで定めた健全性確認を通します。文書のみの変更でも、リンク、状態、判定段階と検証根拠に関する主張、正本との関係を確認します。

## 4. First Reference scope

最初の実際の参照事例は、読取り専用の顧客検索に限定します。

### Shared authority candidates

- `customer.customer`
- `customer.customer_id`
- `master_data.customer ^2`
- `master_data.customer/find_by_id`

### Destination-owned decisions

判定段階Cの後で初めて決定します。

- 正本上のアプリケーションID
- 業務上の文脈
- 移行先の関係
- 移行先のパターンID
- データ接続先の別名
- 認可の要件
- 最初の実行先における表示・操作面

### Environment-owned decisions

- 開発環境での対応付け
- 本番環境での対応付け
- 実際に必要となる場合の資源・権限

### Out of scope

- 顧客データの書込み
- 顧客データの保守管理権限
- 新たな書込み契約
- 書込みの能力
- 二つ目の実行先
- 汎用パッケージ管理機構
- 移植可能性に関する継承・上書き
- リポジトリをまたぐ同期

## 5. Work sequence

### S0 — Gate C entry capture

判定段階Cの検証根拠を開始記録に固定し、開始前の確認で`entry-evidence-complete`であることを確かめます。

この事前確認は判定段階Cの判断権限そのものではありません。受理済みの検証根拠が、第二アプリケーションの開始に必要な形で揃っているかを確かめます。

Output:

- Gate C Evidence refs
- `R`
- `T1`
- target difference summary
- unresolved

ソースは変更しません。

### S1 — ISSUE-000047 candidate qualification

目的：

> destination Business contextからCustomerをIDで参照したい。

現在のリポジトリから、候補を読取り専用で探索します。

最低限の対象：

- 顧客の意味上の定義
- 顧客IDの型
- 顧客に関する契約
- 契約に基づく問い合わせ
- 元となる参照パターンの来歴

判定：

- applicable
- incompatible
- unknown

結果には、権威ある入力元・発見の来歴・適格性の判断理由・不足条件を記録します。

### S2 — Phase 5 portability relation

S1で得た候補を次のように分類します。

- linked reuse
- conditional linked reuse
- adopted local variation
- rebind-required

保持する情報：

- 管理権限の所有者
- 依存関係の対象
- 来歴
- 情報の鮮度を判断する入力
- 移行先の要件

`applicable`を`accepted`として扱いません。

### S3 — Destination decision packet

次の未決定事項について、明示的に判断します。

1. アプリケーションID
2. 業務上の文脈と関係
3. パターンID
4. データ接続先の別名
5. 認可の要件
6. 開発環境と本番環境の対応付け方針
7. 最初の実行先における表示範囲

未決定の項目は`unknown`として残し、元の参照事例から推測して補いません。

### S4 — ISSUE-000040 composition candidate

既存grammarだけでProposalを作る。

概念形:

```text
application <destination> {
    uses semantic customer

    database <destination-alias> {
        target master_data
        access read
        requires {
            contract master_data.customer ^2
        }
    }
}
```

reference Patternはdestination-owned。

origin `order_entry.customer_reference`をコピー正本にしない。

### S5 — Shared Acceptance

composition candidateをISSUE-000027 / ADR-0022のAcceptance boundaryへ渡す。

Acceptance前:

- authoritative sourceを変更しない
- Environment bindingを作らない
- AI Proposalをaccepted Decisionとみなさない

Acceptance後のみsource authoringへ進む。

### S6 — Source / reload / validation

accepted compositionを通常のTEI sourceとしてauthorする。

確認:

- AI / exploration indexなしでreload可能
-同じCanonical meaningを再構成
- Validation / Source Indexが成立
- dedicated portability grammar不要という仮説を再評価

### S7 — Environment Binding Proposal

Application source acceptance後にEnvironment側を別authorityとして扱う。

preflight:


current source inspectionでは、development / productionに既存の `master_data.dev` / `master_data.prod` Oracle instanceがあり、最小候補はdestination-owned aliasをそれぞれへ結ぶ2 bindingである。

```text
development:
  bind repair_intake.customer_master -> master_data.dev

production:
  bind repair_intake.customer_master -> master_data.prod
```

新しいDatabase instance / Oracle plugin / Provider bundleを作るEvidenceは無い。

確認:

- destination Application / alias
- target Database instance
- Database definition
- Provider resolution
- Permission / Resource requirement
- BindingがPermission / Resource / Runtime authorityを暗黙grantしないこと

origin `order_entry.master` bindingをcopy authorityとして扱わない。

### S8 — Driver / First Web Target

preflight:


read-only customer lookupを、

1. Driver
2. 第一Web Target

で再現する。

developmentには既に`master_data.customer/find_by_id`のContract Query executor bindingがあるため、synthetic/deterministic Driver scenarioは既存境界でpressureできる。productionには同等executor bindingが無いため、development設定を暗黙流用しない。

current First Web TargetはProjectionだけ`-application`で切替可能だが、read scenario runnerが`order_entry`固定である。そのため`repair_intake` manifestと`order_entry` executionが混在し得る。S8ではこれをfail closedし、Customer ID input -> `repair_intake.intake.customer_id` mapped row -> Driver reference lookupというTarget-local read surfaceを追加pressureする。

同じBusiness / Semantic meaningをTarget側で再定義しない。

S8のTarget runではhistorical `B / R`を再利用しない。`repair_intake`を含むDriver EvidenceをS8aで取得後、Handover contractのsuccessor規則に従って `B2 / R2` を固定し、First Web Target buildを別identity `T2` として追跡する。

### S9 — Change propagation Evidence

最低限次をPressureする。

- shared Semantic change → dependency requalification
- Contract compatible minor → compatibility再評価
- Contract major → dependency再評価
- origin Pattern change → provenance drift
- destination Pattern change → originへback-propagateしない
- origin alias / Environment binding change → destinationへ伝播しない
- unrelated origin change → stale-allしない

ISSUE-000039が利用可能なら共通impact解析を使う。

独自impact engineをISSUE-000040へ作らない。

## 6. Stop conditions

workstream中に次が発生した場合、次stepへ自動進行しない。

### Candidate unknown

必要Evidenceを追加取得する。

originの選択を推測して埋めない。

### Candidate incompatible

別候補またはdestination設計変更へ戻る。

Semantic identity自体まで不用意にincompatibleへ広げない。

### New schema required

既存grammarで表現不能な具体的Evidenceを記録する。

先にKnowledge Package / inheritance / overrideを作らない。

### Authorization unresolved

read routeが分かっても実行authorityを推測しない。

### Environment unresolved

Application acceptanceからBindingを自動生成しない。

### Target semantic divergence

Target実装都合でCanonical meaningを黙って変えない。

必要ならbaseline / accepted source変更へ戻る。

## 7. AI Builder handoff

AI Builderへ渡すcontextは少なくとも次を含む。

```text
Gate C entry
  R / T1 / Evidence

intent
  read-only customer lookup

shared candidates
  identity / authority / qualification

local decisions
  accepted / unresolved

provenance
  origin Pattern / source

portability
  reuse relation / freshness

acceptance
  current stage

implementation
  allowed source kinds
  forbidden shortcuts
```

AIへRepository全体を無差別に読ませることを標準入口にしない。

## 8. Human review handoff

Human Reviewではgenerated codeより先に、

- intent
- shared authority
- destination-owned meaning
- Environment-owned choices
- unresolved
- impact candidates
- exact source refs
- Acceptance status

を提示する。

```text
less code reading
  != less Evidence
```

を維持する。

## 9. Commit granularity

ISSUE-000040 workstream内では、意味の境界に沿ってmainへcommitする。

推奨単位:

1. Gate C entry / qualification Evidence
2. portability / destination decision
3. composition Proposal
4. accepted source
5. Environment Binding
6. Driver / Target Evidence
7. change propagation Evidence

「ファイルごと」ではなく「検証可能な意味単位」でcommitする。

一つの意味変更を複数の作業系列へ分けず、並行sessionがある場合はcurrent mainと関連Source / Evidenceを直前に再確認する。

## 10. main commit completion

各意味単位のcommitは、最低限次のうち該当する情報を正本文書またはEvidenceへ残す。

- Gate C entry Evidence
- shared / local / Environment boundary
- reused Canonical identities
- destination-owned identities
- unknown / incompatible
- Acceptance Evidence
- reload / Validation Evidence
- Driver / Target Evidence
- propagation Evidence
- 追加schemaを導入したか / 見送ったか、その理由

branch / PRを完了条件にはしない。コードを変更した場合はpush前のRepository health checkを必須とし、文書だけの場合もリンク・状態・Gate / Evidence claimを確認する。

## 11. 現在状態

2026-09-19、第一Web TargetはGate C Readyに到達した。

actual entry:

- `B / R / T = B / e5231839a5dcc62cce1181a2731fcb7c70d16736 / 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`

S0は完了した。

S1 / S2は、

- `tools/reference-business-procedure-check/second_application_s1_s2_actual_pressure_test.go`

でSource-definedまで進んだ。



2026-09-21時点で、revised executable pressure、multi-layer Shared Acceptance実装、actual `reference/workspace` read-only Preview、別生成AIによる独立レビュー（`review-clear-with-conditions`）まで完了した。required pre-Acceptance fixは無い。残るS5停止条件はexact `c18ae...`に対するexplicit human Acceptance decisionだけである。

Acceptance実行契約:

Acceptance成功後のS6 Source / reload / validation契約:

一般的な作業継続指示をAcceptance witnessへ読み替えない。

## 12. Start Packet完了条件

- [x] Gate C entry recordの必須項目を固定した
- [x] 一つのISSUE-000040 workstreamへ収束し、remote branchを増やさない規則へ更新した
- [x] first Reference scopeをread-only customer lookupへ限定した
- [x] ISSUE-000047 → Phase 5 → ISSUE-000040 → Acceptanceの順序を固定した
- [x] Application / Environment authorityを分離した
- [x] unknown / incompatibleのstop条件を固定した
- [x] existing grammar firstを維持した
- [x] AI Builder / Human Review handoffを定義した
- [x] direct-to-main / health-check / concurrent freshness規則へ更新した
- [x] Gate C前にsecond Application sourceを追加していない
