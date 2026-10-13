# std.localization：std.messageに依存する翻訳拡張の参照契約

担当：[std.message #1649](https://github.com/hat47x/tei-platform/issues/1649)、[透過的オーバーレイ #1551](https://github.com/hat47x/tei-platform/issues/1551)。

`std.localization`は`std.message`の機能をクラス継承するのではなく、既存`plugin.Definition.Requires`の依存関係を通じて導入する。通常アプリケーションのメッセージ定義は変更しない。

## プラグイン主導のデータ領域

`StoreDeclaration`は翻訳保持領域の論理モデルを**翻訳プラグイン自身が提案**する。`ApplicationBinding`が、どのアプリケーション・環境・Plugin版でその領域を使うかを指定する。資源の内容は`TranslationRecord`として別管理する。領域のモデルと翻訳結果を、TEI Core固定の翻訳テーブルに閉じ込めない。

現在の`StoreSchema=tei.message-translation.v0`は参照用のデータ交換形。物理テーブル、永続化DDL、正式なWorkspace DSL、Editorでの適用処理、ホスト認可を実装したものではない。アプリケーションによるモデルの実際の受理、DB接続やStorage Pluginのライフサイクルへの適用は後続作業。

## 適用ポリシー

- `source-only`：訳語が存在しても使用しない。翻訳プラグインを導入しない通常アプリにも適用できる。
- `translated`（既定）：許可された翻訳候補だけを既存のE1資源選択へ渡す。言語・代替言語・原文フォールバックは明示する。
- `Grant`はアプリ、環境、Plugin版、Message ID、言語で適用可否を絞る。プラグインが自己許可できないことが前提で、実際のHost受理を経たデータだけを信頼する。
- 同一言語に複数の提供元がある場合は、採用するPluginが競合相手を全て明示的に置換する`Override`を要求する。読込み順・最大優先度の推測は使わない。
- 原文やパラメータ契約の変更、未確認の翻訳、型付き変数の不一致を拒否する。重要メッセージでは代替言語を黙って採用しない。

既存`multilingual-resources.Resolve`を利用し、翻訳用の独立した書式処理や別系統のPlugin Resolverは作らない。`ReviewGate`は既存Hostの認証済み受理情報の**接続口**に過ぎない。試験用の`func(ReviewInput) error { return nil }`が本番運用の承認を意味するわけではない。呼出し側の受理記録の取得、権限再評価、同一版・出所の検査は未実装。

```sh
go test -race -count=3 ./reference/standard-messages ./reference/message-localization ./reference/multilingual-resources
```

## 境界と後続

現段階ではメッセージ本文を表示資源として**選択**するだけで、`{orderId}`を実際の引数値に置換しない。HTMLへ安全に挿入できると証明したわけでもない。MF2は別途評価する。

概念ベースの表示名は既存[#1551](../multilingual-overlays/README.md)を利用する。メッセージ以外の概念にも同じStore/Binding/Policyを汎用化することはO2c後続の設計課題。業務データの実体に多言語本文を追加する設計は[#1453](https://github.com/hat47x/tei-platform/issues/1453)に留める。
