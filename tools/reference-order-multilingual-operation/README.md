# reference-order-multilingual-operation

技術検証用サンプル[`order-multilingual-operation`](../../reference/order-multilingual-operation/README.md)(サンプル8)の工程1〜6を、一つの通しの記録として実行する実装です(作業中)。設計の正本は[技術検証用サンプルの統合設計 4.8節](../../plan/design/TECH_VALIDATION_SAMPLES_2026-10.md#48-受注業務の英語化運用と原文改訂時の訳の失効order-multilingual-operation)です。

## 調査の結論(実装前)

### 既存資産を呼べるか

| 工程 | 既存資産 | 呼べるか | 理由と扱い |
| --- | --- | --- | --- |
| 1 | `reference/standard-messages` | ライブラリとして呼べる | `Catalog.Validate`、`Lookup`、`Revision`、`ValidateArguments`が公開。版は定義全体のJSONのSHA-256 |
| 2 | `reference/message-localization` | ライブラリとして呼べる | `ResolveWorkspace`が実際の作業領域を読む。ただし`std.message`が環境で選ばれている必要があり、`reference/workspace`にはないため、複製に追加する。項目名は資産内で`message-body`に固定 |
| 3・4 | `reference/multilingual-overlays` | ライブラリとして呼べる | `BuildLockedLocaleBundle`、`CheckLocaleBundleIntegrity`、`CheckLocaleBundleAgainstLock`、`BuildLockedDisplay`。表示名の版は表示文字列のSHA-256。`BuildWorkspaceDisplay`系は原文を作業領域の`Symbol`(ASCIIの識別子`Order`)から取るため、日本語の原文表示名を渡せない |
| 3・4 | `reference/multilingual-resources` | 上の二つの内部で呼ばれる | 固定データ`testdata/order-entry.json`は所有者`order-entry`、項目名`action-label`、版`source-12`で、他と合わない |
| 4 | `reference/multilingual-integration` | 呼べない | 試験ファイルだけのパッケージで、公開の入口がない |
| 5 | `reference/multilingual-business-data` | ライブラリとして呼べる | `Select`、`CheckProposal`、`PrepareTranslation`。`testdata/schema.sql`はファイルとして読む(変更しない) |
| 6 | `reference/multilingual-format` | 呼べる | Goの`Probe`と、JavaScriptの`probe.mjs`の`probe`。メッセージの引数とは未接続 |

### 識別子が資産ごとに別の合成値であること

| 識別子 | 資産ごとの値 |
| --- | --- |
| 所有プラグイン | `sales.order_ui@1.0.0`(message-localizationの試験)、`std.order_entry@1.2.0`(overlaysの`ReferenceOrderCase`)、`tei.order.ui@1.0.0`(overlaysの作業領域の試験)、`order-entry`(resourcesの固定データ) |
| 環境 | `development`(message-localization)、`env.order_reference`(overlays) |
| 原文言語 | `en-US`(message-localizationの試験、overlaysの作業領域の試験)、`ja-JP`(overlays、resources) |
| 項目名 | `message-body`(message-localization)、`display-name`(overlays)、`action-label`(resources) |
| 対象 | 範囲`sales`の種類`function`(overlays、resources)、`workspace.sales`の`semantic-data`(overlaysの作業領域の試験) |

`reference/workspace`には、受注の業務定義を所有するプラグイン、`std.message`、言語プラグイン、日本語の表示名がありません。構文の識別子はASCIIだけで、`Symbol`は識別子です。
