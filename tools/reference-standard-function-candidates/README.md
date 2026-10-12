# データ定義から標準機能の候補を作る

対象: [#1127](https://github.com/hat47x/tei-platform/issues/1127) の初期段階。[#982](https://github.com/hat47x/tei-platform/issues/982) の製品要求トリアージから切り出した、実行前の小さな参照実装です。

既存のデータ定義とアプリケーションの参照宣言だけを読み取り、標準の一覧、単票、作成、変更の**候補**を生成します。候補は既存の`browse`、`record`、`command`の分類を使い、正本の機能定義は変更しません。派生IDは二次識別子として保持します。

## 実行

リポジトリのルートから次を実行します。

```sh
go run ./tools/reference-standard-function-candidates \
  -workspace examples/first-workspace -application request_log
```

`intake.request`について4件の候補が表示されます。出力の`authority`は`proposal-only`、すべての`executable`は`false`です。属性の識別子と型はモデルから読み取り、型が`std.identifier`でも物理主キーと推定しません。必須入力や権限、保存先、業務上の操作は推定しません。

| 候補 | 既存の機能分類 | 実行に向けて不足するもの |
| --- | --- | --- |
| 一覧 | `browse` | 明示的なQuery、データソース、提供元、権限 |
| 単票 | `record` | 読取りの結合、提供元、権限 |
| 作成・変更 | `command` | Capability、Binding、保存先の対応、提供元、権限、独立した再読込 |

出力の`missing`は次段階で明示的に解決すべき項目の一覧であり、既存の実行契約を検査し終えた診断結果ではありません。これらの項目が実際に解決されても、この道具自身は実行の可否を承認しません。

既存のアプリケーションに同じデータを参照する手書きの一覧・単票があれば、その候補を重複生成しません。別の機能の命名と派生IDが衝突するときや、対象のアプリケーション・意味の参照が曖昧なときは、候補を推測せずエラーにします。

## 確認

```sh
go test ./tools/reference-standard-function-candidates -count=1
```

同じ入力から同じ結果を得られること、属性を追加しても派生IDが安定すること、既存機能を上書きしないこと、曖昧な定義を拒否することを検証します。

これは**画面生成器、CRUDの実行基盤、権限判定器、データベース接続ではありません**。次に必要なのは、既存のApplication Pattern・Driver・Provider・UI Kitへ明示的な契約でつなぎ、保存後の別の読取りを伴う一巡を実測することです。実行の証拠は[#1163](https://github.com/hat47x/tei-platform/issues/1163)の横断検証と接続します。
