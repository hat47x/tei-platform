# 多言語の数量表現をGoとWebで照合する（E2a）

- **状態**：参照実装。Unicode MessageFormat 2（MF2）の構文解析器・完全な表示処理ではない。
- **担当課題**：[多言語対応E2 #1422](https://github.com/hat47x/tei-platform/issues/1422)
- **前段階**：[E1：意味上の参照と表示資源の選択](../multilingual-resources/README.md)
- **設計上の位置付け**：[TEIの多言語対応](../../plan/design/MULTILINGUAL_SEMANTIC_RESOURCE_DESIGN_2026-10.md)

## 目的

「1件」と「2件」では、言語に応じた複数形や桁区切りが必要になる。TEIはGoとブラウザを利用するが、Goの言語処理ライブラリとJavaScriptの国際化機能が同じ地域別データを使うとは限らない。

最初に、既存ライブラリが持つ**数量の分類と整数の表示**だけを同じ固定データで試す。翻訳文を組み立てる新しい処理機構は作らない。この段階で得られるのは、実装を選ぶための局所的な実測結果である。

## 対象と根拠

- 対象言語：英語（米国）、日本語（日本）、ドイツ語（ドイツ）。
- 入力：0以上、2,147,483,647以下の整数。負の数、小数、文字列、未対応の言語を拒否する。
- 出力：地域別の複数形分類（例：`one`、`other`）と桁区切りを施した整数。
- Go：リポジトリで既に依存している`golang.org/x/text`の複数形規則と表示器。
- JavaScript：標準の`Intl.PluralRules`と`Intl.NumberFormat`。
- 共通試験：[`testdata/integer-forms.json`](testdata/integer-forms.json)。両言語の試験が同じ期待結果を読む。

この方法では各実装が共通の試験データを基準にできますが、GoとJavaScriptが同じCLDRの版を使用しているとは主張できません。Goの`x/text/feature/plural`には古いCLDRの分類表が残っており、JavaScriptは実行環境のICUとCLDRに依存します。試験記録にはGo依存版とNode・ICU・CLDRの版を残します。

## 検証

リポジトリのルートから実行します。

```sh
go test -count=1 ./reference/multilingual-format
go test -race -count=3 ./reference/multilingual-format
go vet ./reference/multilingual-format
node --test reference/multilingual-format/*.test.mjs
```

プルリクエストでは、上記を[専用のGitHub Actions](../../.github/workflows/tei-multilingual-format-ci.yml)から実行する。失敗した言語・値・処理系を区別して調べられるよう、同一の試験事例識別子を使う。

## 次の段階と停止線

MF2の標準仕様では、`{$count :integer}`のような書式指定や、複数形による文章全体の分岐を扱える。この参照実装は、そうした**MF2の構文を読み込んでいない**。独自構文解析器を作っているわけでもなく、MF2準拠を証明するものでもない。

次のE2bでは、JavaScriptの`messageformat` v4などのMF2実装を版固定し、公式仕様の適合性試験に接続する。Go側は対応する実装の要求Go版・ライセンス・対応仕様版を比較し、TEIの実行先と構築契約に適したものを選ぶ。現在のGoの版を引き上げたり、全実行先に国際化ライブラリを組み込んだりする判断は、この参照実装では行わない。

現行E1の翻訳資源選択、プラグイン構成、言語ごとの版や人間の確認状態は別の責務として維持する。

## 外部仕様

- [Unicode MessageFormat 2](https://www.unicode.org/reports/tr35/tr35-messageFormat.html)
- [Goの複数形規則](https://pkg.go.dev/golang.org/x/text/feature/plural)
- [JavaScript Intl.PluralRules](https://developer.mozilla.org/docs/Web/JavaScript/Reference/Global_Objects/Intl/PluralRules)
- [JavaScript Intl.NumberFormat](https://developer.mozilla.org/docs/Web/JavaScript/Reference/Global_Objects/Intl/NumberFormat)
