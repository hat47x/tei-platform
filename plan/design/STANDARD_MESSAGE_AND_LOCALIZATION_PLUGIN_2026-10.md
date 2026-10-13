# TEI 標準メッセージと多言語拡張のプラグイン分離

- 状態：**候補契約・Go参照実装**（2026-10-13）。正式Plugin DSL、Workspace Acceptance、DB永続化、実Hostインストールは未実装。
- 課題：[std.message #1649](https://github.com/hat47x/tei-platform/issues/1649)、[多言語オーバーレイ #1551](https://github.com/hat47x/tei-platform/issues/1551)。
- 関係する既存機構：[意味上の資源E1](../../reference/multilingual-resources/README.md)、[Plugin Lock](../../internal/plugin/lock.go)、[Workspace Source](../../internal/workspace/loader.go)。

## 決定候補：単一言語の基盤を先に成立させる

1. `std.message`は意味的に安定したMessage ID、原文、本文の書式モード、引数名・型と参照検証を提供する。言語選択や翻訳保存・生成は責務ではない。
2. `std.localization`は`std.message`への通常のPlugin依存を持つ。クラス継承と異なり、Plugin Resolver、Lockのライフサイクルは変更しない。
3. 概念の表示名、メッセージ本文、UIヘルプ等は**共通の翻訳選択基盤**で管理できるが、定義の正本は対象ごとに異なる。通常の概念モデルをコピーしない。
4. 翻訳結果の記録領域はプラグインが宣言し、環境・アプリごとのBindingで適用する。アプリ適用とデータレコード受理はHostが所有する。既存のDB契約・Storage機構に乗せることを優先し、翻訳用の固定Core DBを設けない。
5. ポリシーは「利用しない」「利用する」「競合時に明示置換」「代替言語と原文フォールバック」を宣言可能とする。プラグイン自身による権限昇格は不可。既定では登録順に依存せず競合を拒否する。
6. 重要文言は厳格なフォールバックで扱う。原文・引数宣言の変更による旧訳の失効は、共通のE1資源選択を使う。
7. `MessageFormat 2`の入力・表現機能は書式実装の課題[#1422](https://github.com/hat47x/tei-platform/issues/1422)に分離し、簡易テンプレート解析をMF2互換とは呼ばない。

## 対象別の整理

| 対象 | 定義の正本 | 拡張・選択 | 保持対象 |
| --- | --- | --- | --- |
| 一般メッセージ | `std.message`利用側のMessage定義 | `std.localization` | 翻訳本文・原文版・確認状態 |
| 意味モデルの表示名 | 通常Semantic Data/Function等 | `std.localization`と[#1551](https://github.com/hat47x/tei-platform/issues/1551) | 追加表示資源、適用Scope |
| UI独自ヘルプ | UI標準部品／案件定義 | Resource Adapter | 意味参照または明示Resource ID |
| 業務データ本文 | 業務Dataモデル | [#1453](https://github.com/hat47x/tei-platform/issues/1453) | 業務実体の多言語正本 |

## 既存の受理との接続上の停止線

`StoreDeclaration`・`ApplicationBinding`・`Grant`・`Override`は、どのPluginが提案・所有し、どのアプリへ適用されるかを表す**候補データ**。これらに署名の無いスタブを渡して合成できても、Hostの権限付与・受理済みSourceの証拠とは見なさない。

実装次段階では、Editor/Workspaceが受理したStore定義・Application BindingとPlugin Lockの固定版を検証し、後から拒否・原文更新・Plugin削除を再評価する。UI配布用JSONには出所の認証・改ざん検出が別途必要。

## 受入試験

- `std.localization`なしで単一言語のメッセージが安定して解決できる。
- 後付けの`std.localization`が`std.message`依存を解決し、通常プラグインの定義を変えずに翻訳を選択できる。
- 型変更・原文改訂・未確認翻訳は採用されない。
- 異環境・異アプリの保存領域、許可されない翻訳は採用されない。
- 競合は明示的な置換規則でのみ解決し、登録順に依存しない。
- 翻訳を使用しないポリシーは既定言語と明示的な原文フォールバックで動作する。

これらの局所回帰試験を`reference/standard-messages`および`reference/message-localization`で実行する。
