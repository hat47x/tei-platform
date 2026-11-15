# application の database ブロックが bind されていなくても `ok`（2026-11-12、第79ラウンド）

- 対象: 新規。`UNBOUND_DATABASE_REQUIREMENT` を追加。
- 前段: [第78ラウンド](../TEI_DOGFOODING_ROUND78_REA_EVIDENCE_ORPHAN_2026-11-12/README.md)
- 状態: **application の database 要件がどの environment でも bind されていないことを警告する。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行した。2つ目の application `reporting` を CLI だけで設計し、全成果物を読み返した。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `84d1d6dcf`（`origin/main`） |
| CLI実行物（設計時） | `sha256:5cb068f736b2c0c2275841e1925d116d876f452d78fc48c15ecc065bbd3cb40e` |
| CLI実行物（修正後） | `sha256:bd94e207fffc99a1ef748c9ce946e4c9923501fce5525a5a2376ade672c70a64` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 設計（3往復）

| Attempt | 応答 | 学んだこと |
| --- | --- | --- |
| 1 | `ok` | `application X { uses semantic Y }` だけでも通る |
| 2 | `ok` | `database` ブロック + `bind` で完結。bind は `environment` の中 |
| 3 | `ok` | 読み返しで `bindings=4`、`application` は `databases` を返す |

契約の検査も実在した（`contract ... ^99` → `DATABASE_CONTRACT_VERSION_UNSATISFIED`、存在しない契約 → `UNKNOWN_DATABASE_CONTRACT`）。

## 4. 見つけた穴

`reporting.sales` の `bind` を**全部消して** `validate` を実行した。

```text
$ tei validate -workspace <ws>
ok
```

**`reporting` の database 要件がどの environment でも bind されていないのに `ok`。** 既存の `UNBOUND_DATASOURCE` は **query の datasource** だけを見ており、**application の `database` ブロックは見ていなかった**。`deployment.go` の `ACTIVE_RELEASE_DATASOURCE_UNBOUND` は**展開済みリリース**の話で、現在の定義には効かない。

## 5. 修正

`Validate` に `UNBOUND_DATABASE_REQUIREMENT`（warning）を足した。

```text
warning UNBOUND_DATABASE_REQUIREMENT application reporting database sales is not bound
in any environment; add "bind reporting.sales -> sales_core.dev" inside environment
development, which already declares that instance
```

名前は既存の `unboundDatasourceHint` を再利用し、**その作業場所が実際に宣言している instance** を名指しする。

### 5.1 誤検出を1回出して、絞った

最初の実装は **18ワークスペース中5つで警告**した（`examples/xtea-lowcode`・`reference/org-headless-workspace`・`reference/standard-crud/workspace` ほか）。調べると**どれも environment を1つも宣言していない**（`environments=0`, `bind lines=0`）。**bind できない作業場所で「bind しろ」と言うのは雑音**である。

```go
warnsAboutBindings := len(w.Environments) > 0
```

この門を足して **0 / 18** になった。**既存の query 側 `UNBOUND_DATASOURCE` はこの門を持たない**（`examples/xtea-lowcode` では1件出る）。差は意図的で、その旨をコードのコメントに書いた。

## 6. 試験と変異

| 試験 | 内容 |
| --- | --- |
| `TestUnboundDatabaseRequirementIsReported` | 未 bind を warning として報告し、**書くべき bind を名指しする** |
| `TestBoundDatabaseRequirementIsNotReported` | bind 済みなら出さない |
| `TestUnboundDatabaseRequirementStaysQuietWithoutEnvironments` | **environment が無い作業場所では黙る** |

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 常に警告しない | FAIL `an unbound application database was not reported` |
| M2 | 門を外す（誤検出の状態） | FAIL `warned in a workspace with no environments` |

**M1 は最初「壊れた変異」だった**（テキスト削除で構文エラー）。`if false &&` に置き換えて作り直した（**壊れた変異に気づくのは5回目**: 61・67・72・74・79）。

## 7. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/validation` | ok |
| `go vet` / `gofmt -l internal/validation` | 指摘なし / 空 |
| 全18ワークスペースの誤検出 | **0** |

## 8. 読み取れないこと（NOT RUN）

- **bind が必須かどうかの設計判断**（現状は warning。error にすべきかは所有者の判断）。
- 展開済みリリース（`ACTIVE_RELEASE_DATASOURCE_UNBOUND`）との关系。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 9. 位置付け

**「片方だけ見ていた」を揃えた回**である。query の datasource は検査されていたが、application の `database` ブロックは検査されていなかった。**同じ概念（datasource の bind）に対して2つの入口があり、片方だけ守られていた。**
