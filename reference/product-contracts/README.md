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

## 確認

```text
go run ./cmd/tei validate -workspace reference/product-contracts
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
