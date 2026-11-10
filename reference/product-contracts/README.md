# プロダクト契約のTEIモデル

TAI配下の5製品をTEIで再表現できるかを検討するための参照workspaceです。実装を置き換えるものではなく、各製品の主要な契約をTEIの型として並べ、共通部分と製品固有部分を分けて見ます。

## 含めるもの

| 製品 | 主な型 |
| --- | --- |
| 共通 | `Evidence` / `Authority` |
| EKI | `TaskRequest` / `Attempt` / `ResultClaim` / `AcceptedResult` |
| SUI | `Document` / `Card` / `Island` / `Edge` |
| SEI | `Decision` / `Unresolved` |
| MMI | `Observation` / `Selection` |

## 能力

| 能力 | 入力 | 出力 | effect |
| --- | --- | --- | --- |
| `product_contracts.eki_task_submit` | `TaskRequest` | `Attempt` | `create Attempt` |
| `product_contracts.eki_attempt_execute` | `Attempt` | `ResultClaim` | `create ResultClaim` |
| `product_contracts.eki_result_accept` | `ResultClaim` | `AcceptedResult` | `create AcceptedResult` |
| `product_contracts.sui_document_put` | `Document` | `Document` | `update Document` |
| `product_contracts.sei_decision_record` | `Decision` | `Decision` | `create Decision` |
| `product_contracts.mmi_observation_select` | `Selection` | `Selection` | なし（read-only） |

## アプリケーション

`applications/product-flow/application.tei`は、`product_contracts`のsemanticモデルを参照するアプリケーションです。

6つの`command` patternが、それぞれ1つのcapabilityを参照します。

- `eki_task_submit`
- `eki_attempt_execute`
- `eki_result_accept`
- `sui_document_put`
- `sei_decision_record`
- `mmi_observation_select`

patternは能力を参照します。`realizations/`では、各capabilityを`custom go`の実装へ結びつける宣言だけを持ちます。capabilityは`effect`で業務意図を宣言します。呼出し順はTEI DSLの専用構文では表現しません。3つのEKI capabilityを`composition`のProviderとして並べ、`Attempt`と`ResultClaim`の依存から`ActivationOrder`が`eki_task_submit → eki_attempt_execute → eki_result_accept`になることをtestで確認します。TAI側でも同じ順序を契約として検査します。

`bindings/`には、入力検証だけを行う決定論的な参照bindingと、EKI Control Planeの`/v1/tasks`へTaskRequestをHTTPで送る`EKITaskSubmitHTTP`があります。後者は呼び出し側からHTTP clientを渡すため、mTLSやbearer tokenは呼び出し側が設定します。

live testは、`EKI_CONTROL_URL`と`EKI_CONTROL_TOKEN`がある場合だけ実行します。

```text
EKI_CONTROL_URL=http://127.0.0.1:18082 EKI_CONTROL_TOKEN=<token> \
  go test ./reference/product-contracts/bindings -run TestLiveEKI -count=1 -v
```

EKIのControl PlaneはWSL上で起動すると、Windows Firewallの許可を都度求めずにloopbackで接続できます。

## bindingの出力と宣言型の差（#3657）

`bindings/output_shape_test.go`が、各bindingの出力キーと`semantic/contracts.tei`の出力型の属性を比較します。次の差は**記録した許容**であり、受理ではありません。差が増減すると試験が落ちます。

| binding | 宣言型 | 出力に無い宣言属性 | 宣言に無い出力キー |
| --- | --- | --- | --- |
| `EKITaskSubmit`（合成） | `Attempt` | `leaseId` `fenceEpoch` `workerIdentity` | `executor` |
| `EKIAttemptExecute`（合成） | `ResultClaim` | `exitCode` `outputDigest` `outputBytes` `claimedAt` | なし |
| `EKIResultAccept`（合成） | `AcceptedResult` | `level` `policyDigest` `acceptedAt` | `outcome` |

- 合成bindingの連鎖（submit → execute → accept）は、`attemptId`と`claimId`をキー名のまま次段へ渡せることを試験で確認します。値の中身は固定の合成値です。
- `EKITaskSubmitHTTP`は**`Attempt`へ接続していません（未接続）**。EKIの外部契約（eki-fabricの`spec/external/openapi.json`、`TaskAccepted`、f8047c1で確認。依拠する部分は[`external-contract-pin.json`](external-contract-pin.json)に機械可読形式で固定している）の202応答は`task_id`・`status`（`queued`）などでTaskの受付を示し、Attemptを含みません。bindingは応答をそのまま返すため、宣言した`Attempt`の属性と一致するキーは無く、次段`eki_attempt_execute`が必須とする`attemptId`も出力されません。
- 同じcapability IDの入力も、合成bindingは`{taskId, executorId}`、HTTP bindingは`{idempotencyKey, request}`（EKIのTaskIR）で異なり、`TaskRequest`の全属性とは一致しません（合成側は宣言属性の一部だけ、HTTP側は共通キーなし）。
- 宣言型をEKIの応答へ合わせるか、bindingに写像層を置くか、写像をEKI側契約へ委ねるか、欠落を許容リストに残すかは未決で、Ownerの判断を待ちます。この文書と試験は、その判断を先取りしません。

## 外部契約の固定（#3836）

この reference workspace が前提にする EKI の外部契約は、
[`external-contract-pin.json`](external-contract-pin.json) に機械可読形式で固定しています。

| 項目 | 内容 |
| --- | --- |
| 生成側 | `hat47x/eki-fabric` の `spec/external/openapi.json` |
| 固定した版 | `f8047c1`（OpenAPI 3.2.1、API 0.1.0） |
| 依拠する部分 | `POST /v1/tasks` の 202 応答 `TaskAccepted`の必須キーと `status` の値 |
| 依拠しない部分 | `Attempt`・`ResultClaim`・`AcceptedResult` |

EKI の契約本文は転載せず、**TEI が依拠する部分だけ**を記録します。

### 何を検査するか

`external_contract_pin_test.go` は、次を常時検査します（外部のチェックアウトもネットワークも不要）。

- 固定ファイルが形式として成立し、バージョンが固定の値（`main` やブランチ名ではない）であること。
- この README が同じ版を引用し、固定ファイルを指していること。
- `bindings/testdata/task_accepted.json`（実行で使う 202 応答の形）が、固定ファイルが求めるキーと値をすべて含むこと。

**EKI 側の実物との比較は常時ではなく手動です。** EKI のチェックアウトは TEI の CI には無く、常時比較を要件にすると CI が外部依存になります。代わりに、比較した版・日付・観測を下の記録に残します。

### 実物と比較する手順

```bash
# EKI のチェックアウトを指定して実行すると、固定した版と実物の差を検出する。
EKI_CONTRACT_REPO=/path/to/eki-fabric \
  go test ./reference/product-contracts -run TestLiveExternalContractPin -count=1 -v
```

環境変数が無い場合は skip されます。**skip を「一致している」の根拠にしないでください**。

### 手動確認の記録

| 日付 | 比較した EKI | 結果 | 観測 |
| --- | --- | --- | --- |
| 2026-11-10 | `f8047c1`（ローカルの `D:/GIT/hat47x/eki-fabric`） | 一致 | `TaskAccepted` の必須キー6つと `status`の`queued`が固定と同じ。EKI 側で `spec/external/openapi.json` は `f8047c1` 以降変更されていない。 |

## 確認

```text
go run ./cmd/tei validate -workspace reference/product-contracts
go test ./reference/product-contracts/... -count=1
```

## 境界

- 型の宣言であり、各製品の実装・権限・検証を置き換えません。
- 共通の`Evidence`と`Authority`は最小限の項目だけを持ちます。現状ではどの属性・関係・能力からも参照しない**意図した省略**です（型の最小化。[#2411](https://github.com/hat47x/tei-platform/issues/2411)の照合記録）。EKI側の受理の正当化は`PolicyDigest`と検証者群・申告Workerの独立性の値であって`principal`/`role`の主体記録ではなく、EKIの`PolicySource`/`ProducerSource`の実体も未実装のため、TEI側の対応語彙はこの時点で固定しません。
- `tei validate`は、宣言した型名の実在までを確認します。能力の出力型と次の入力型の接続は検査しません。たとえば`eki_attempt_execute`の出力を別の型へ替えても`ok`になります(`cmd/tei`の試験で確認、[#2410](https://github.com/hat47x/tei-platform/issues/2410))。接続の宣言形式は未決で、Ownerの判断を待ちます。
- 実際の製品への移行は、契約ごとの意味と責務を確認してから行います。

## 能力の接続を宣言する場所の候補

以下は[#2410](https://github.com/hat47x/tei-platform/issues/2410)の比較であり、新しい宣言形式や配置の採用を決めるものではありません。現状のこの作業領域には能力間の接続宣言がなく、能力の表の並び順から接続を推定できません。

| 候補 | 検討する範囲 | 既存の仕組みとの関係・未決事項 |
| --- | --- | --- |
| application | アプリケーションごとに使う能力と接続を記述する | 現在のpatternは個別の能力を参照するだけ。構成選択を扱う[#1016](https://github.com/hat47x/tei-platform/issues/1016)のApplication Composition Profile案との責務分担が必要 |
| semantic | 能力の意味とともに接続を記述する | 複数のアプリケーションで共有する関係と、用途ごとに異なる呼出し順をどう分けるか未決。現在の能力宣言は各入力・出力の型を持つが、接続を持たない |
| 別の構成宣言 | semanticとapplicationから参照する接続を別に記述する | [項目対応の参照実装](../composition-io-mapping/README.md)は明示された項目の型・多重度等を照合する独立試作であり、製品の宣言形式や`validate`への接続ではない。構成の正本や既存の解決器との重複を避ける必要がある |
| TAI側のハーネス記述 | TEI外で対象シナリオの接続を記述・検査する | TEIの`validate`の保証にはならない。TEI側に接続を置くか、TAI側に留めるかは未決 |

いずれの候補でも、[#1121](https://github.com/hat47x/tei-platform/issues/1121)のSourceから実行までの段階的な契約との受け渡しを整理する必要があります。型の接続検査と、意味の受理・実行時の権限確認は別です。配置と検査の契約はOwner判断待ちです。
