# E33：実案件Go＋Lit拡張を、既存CompositionのProvider構成境界へ照合する

- **管理**：[Issue #1016](https://github.com/hat47x/tei-platform/issues/1016)。受注明細Webへの案件固有Go＋Lit拡張を、実際の既存Compositionの`Registry.InspectWiringSet`と`handoff.ReconcileCurrent`を使って評価する。
- **再利用した正本**：2026-10-13にmainへ統合された[#1233の既存Composition構成案・宣言鮮度確認](../../internal/composition/proposal.go)、[既存の証拠参照照合](../composition-evidence-handoff/README.md)、[E32のGitHub固定差分](E32_REAL_PROJECT_PLUGIN_FOOTPRINT.md)。
- **Go試験**：[E33回帰試験](e33_project_plugin_wiring_boundary_test.go)。既存の製品Providerや契約を本番登録するのではなく、**一時的なテストRegistryにだけ仮Providerを追加**して境界を確認する。実行先・製品Runtime・第三者Plugin Registryは変更しない。

## 実案件のどこがつながり、何がまだ足りないか

並行して開発されている[Go＋LitのPR #1567](https://github.com/hat47x/tei-platform/pull/1567)は、架空の顧客コードで案件用メモの補助提案を返すGoパッケージと、診断用HTTP Host・Lit画面、受注明細の読取り専用参照を実装している。

[E32固定スナップショット](e32_project_plugin_pr1567_snapshot.json)で照合した実Sourceは以下の通り。これは**当時の固定改訂** `be193266f1556e090b59cf2c237a23c0ca60236e` に対応し、PR #1567の後続HEADを自動追跡しない。

| 資産 | 観測できるもの | 正式な構成として未確認のもの |
| --- | --- | --- |
| `plugins/reference-project-customer-advice/advice.go` | 案件固有の`Evaluate`、入力`customerCode/note`、提案結果`kind/message/advisory` | 発行されたService Contract、契約指紋、Provider登録、Conformance |
| `tools/reference-project-advice-lit/main.go` | Go直接importと診断用`POST /project/advice` | Product RuntimeによるContract選択・現在のHost実行権限 |
| `tools/reference-project-advice-lit/order_entry_context.go` | 明示ループバックの`GET /scenario/read`を使う読取り専用参照 | 既存ApplicationのAction Hookまたは正式なProject Extension |
| `tools/reference-project-advice-lit/web/advice-element.mjs` | Lit Custom Elementと案件用の入力・表示 | 正規UI Kit／Hostとの構成、Web Targetの採用と受入 |

**注**：PRが追加した20ファイルには、`internal/composition/`や`reference/workspace/`の変更は含まれない。PRから正式契約やProvider登録の存在を推測してはいけない。一方、正式契約が将来不要と確定したことも意味しない。

## 既存Composition機構を実際に通した結果

テスト専用の架空Contract（`reference.project.customer-advice`、版`test-only-v0`）は、**実際の#951公開ContractやFingerprintではない**。架空の`Shape`もGoソースの契約を認証するものではない。

| 構成検査 | 既存機構の結果 | 解釈 |
| --- | --- | --- |
| E32の実ファイル出典があってもProvider宣言を登録していない | `unknown / no-known-provider` | ソースがあることだけからProviderを捏造しない |
| テストRegistryに対象Scopeの仮Providerを**明示登録** | `candidate` | 宣言構造を照合できたにすぎない |
| `ReconcileCurrent`へ実#951証拠参照を渡していない | `reference-missing` | Composition構造候補に受理可能な実装証拠がない |
| 全項目を埋めた**架空の**EvidenceReferenceを手渡し | `reference-linked-unverified` | **文字列の対応**を確認しただけで、偽装できる参照に対する信頼や認可は一切ない |
| 同じScopeに同ContractのProviderが2件ある | `conflict / ambiguous-providers` | 名前順や最新バージョンで自動選択しない |
| Providerの適用Scopeが異なる | `unknown` | 案件境界を越えて選ばない |
| 無関係な案件の宣言を更新する | 既存候補は継続してcurrent | 一律の失効処理を避ける |
| 対象ProviderのContract宣言を変更する | `ProposalCurrent=false`、`ReconcileCurrent`はBlocked | 古いReviewを現在の証拠として再利用しない |

この通し検証で足りないのは新しい「構成候補用Resolver」ではなく、**案件Goコードの発行された契約・正確な成果物・正規Providerと#951証拠の対応**、および実TargetでのHost認可・UI Kit適合である。既存Compositionと#951の証拠照合の接続部分は、既存機構で概念上表現できる。

## 次の実導入で比較すべきこと

1. 案件Goの補助提案を、**助言**として保持するか、正式なApplicationのService Contractへ登録するか、案件ごとの選択を記録する。今の`Evaluate`直接importだけで合法的なProvider公開を宣言しない。
2. 正式公開する場合は[#951](https://github.com/hat47x/tei-platform/issues/951)の実Contract/Fingerprint、Provider、Artifact Digest、適合証拠を**その正本の持ち主から**取得し、`composition.InspectWiringSet`→既存`#951`Resolver→`handoff.ReconcileCurrent`で対象の対応を確認する。今回のテスト専用契約を正式登録しない。
3. UIについては[#997](https://github.com/hat47x/tei-platform/issues/997)のUI Kit/Targetを選び、Lit画面のProject所有差分と依存バンドルの版を判断する。標準UIと案件専用UIの使い分けを記録する。
4. 最後にHostは**実際の**Permission・Lease・実配備成果物・承認済みApplication意味を確認する。Referenceが`reference-linked-unverified`を出しても、実行許可や正式な受注条件を発行してはいけない。
5. 実PRの[差分20ファイル・1,267行](E32_REAL_PROJECT_PLUGIN_FOOTPRINT.md)を工数と混同せず、実際の新規接続宣言、手動判断、AI往復、再検証を[E28計測](E28_WORKFLOW_MEASUREMENT.md)に記録する。二方式の同条件比較が揃うまで独立Profile採用を決めない。

## 再現

```sh
go test -count=1 ./reference/application-composition -run '^TestE33' -v
```

この試験は**一時的なRegistry内での構造選択と証拠欠落・鮮度の負例**を実行する。#1567の実Go/Litソースはmainには未統合のDraft PR上にあり、Goコードを直接実行・importしていない。Sourceの固定Blobを[E32](e32_project_plugin_pr1567_snapshot.json)で照合するだけである。実Provider登録、#951の適合検査の実結果、署名・本番Host Authority・現在の実配備は未実行／未検証。

## E33の判断

既存`composition.Registry`／`handoff.ReconcileCurrent`を活用する方向を維持する。新しいComposition Profile形式、第二のLock、別Graph、独立Resolverを追加しない。次に必要なのは**既存Service Contractを案件Pluginへ正式に結び付ける実アプリケーション実験**であり、業務AcceptanceとPermissionを分けたまま取り組む。
