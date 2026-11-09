# std.message：多言語化に依存しないメッセージ定義の参照契約

担当：[#1649](https://github.com/hat47x/tei-platform/issues/1649)。これは**参照実装**であり、実際のPlugin DSL、Editor内の標準Plugin、Runtimeへインストールされた製品機能ではない。

## メッセージの基本契約

- 識別子は所有Pluginの正確なID/VersionとMessage IDで決まる。メッセージ本文をIDに使わない。
- メッセージには原文言語、本文、書式モード、名前付きパラメータの型、重要文言のフラグがある。翻訳無しでも定義の重複、未定義ID、引数の不足・型不一致を検出する。
- `Revision`は本文だけでなく、パラメータと書式モード・重要度等の宣言を含む。本文が同じでも引数の型を変更したら翻訳は旧版として扱う。
- `plain`と`named-v0`は初期の限定形式。変数展開、書式処理、MessageFormat 2の式や複数形選択は**ここでは実行しない**。実際の値を引数として受け取る処理とGo/Webの書式一致は[#1422](https://github.com/hat47x/tei-platform/issues/1422)で扱う。
- 通知配送、メール送信、ログ保管、業務データの文章は、このメッセージ定義とは別の責務。

```sh
go test -race -count=3 ./reference/standard-messages
```

`std.localization`を一切インストールしない通常のアプリケーションで`std.message`を利用できることを回帰試験で確認する。コンポーネントが標準Pluginとして実際に配布済みという意味ではない。

関連： [翻訳保持モデルとアプリ適用](../message-localization/README.md)。
