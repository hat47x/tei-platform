# 一つの成果物に二つのプラグインを結合する参照試験

[Issue #1251](https://github.com/hat47x/tei-platform/issues/1251)の受入条件のうち、「プラグインの論理識別子と物理的な構築成果物を同一視しない」を実証するための例です。製品用のプラグイン配布方式を確定するものではありません。

[Alpha](../../plugins/reference-bundle-alpha/alpha.go)と[Beta](../../plugins/reference-bundle-beta/beta.go)は、互いに異なるプラグインID・機能IDを持つ独立した実装です。Go標準ライブラリ以外に依存しない二つのソース単位を一つのプロセス実行ファイルへ静的結合します。

実行ファイルは、ホストが指定したプラグインID・版・機能IDの組を厳密に照合し、対応する実装へ振り分けます。同じ成果物のSHA-256を持つだけでは、別のプラグインを呼べることにはしません。TEI内部の旧実行アダプターとの接続は[実プロセス試験](../reference-composition-process-bridge/multi_plugin_artifact_test.go)で確認します。

## 確認方法

リポジトリのルートで実行します。

```sh
go test -count=1 ./plugins/reference-bundle-alpha ./plugins/reference-bundle-beta ./tools/reference-plugin-bundle
bash tools/check-reference-plugin-bundle-isolation.sh
go test -count=1 ./tools/reference-composition-process-bridge
```

隔離確認は三つのソース単位を一時Goモジュールに複製し、外部のモジュール取得を禁止した状態で単体試験・Vet・実行ファイルの構築を行います。各プラグインに`go.mod`を追加する必要はありません。

## 立証できる範囲

- 二つのソース単位がTEIコアの内部パッケージに依存しない。
- 一つの実行ファイルと一つの成果物指紋の下でも、論理プラグイン・機能・呼出し結果・来歴を別々に追跡できる。
- プラグインと機能の組み合わせが一致しない場合、実行ファイルは要求を拒否する。ホストで誤った配線や現在の権限失効も拒否する。

この例は現行の処理実行形式を用いた参考試験であり、正式な新しいService Contract、複数プラグインの署名・発行権限、OS隔離、実際の外部業務処理、任意の組合せを可能とする製品ビルダーを実装しません。一つのプラグインを複数の実行先へ構築する検証も別条件です。[ADR-0033](../../plan/adr/0033-core-and-plugin-subprojects.md)に沿って、ソース境界と成果物境界を分けて扱います。
