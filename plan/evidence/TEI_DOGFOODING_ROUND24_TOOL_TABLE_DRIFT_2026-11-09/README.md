# tool の引数一覧が schema と独立に手書きされていた（2026-11-09、第24ラウンド）

- 対象: [#3823](https://github.com/hat47x/tei-platform/issues/3823)、[#3420](https://github.com/hat47x/tei-platform/issues/3420)、[#3419](https://github.com/hat47x/tei-platform/issues/3419)、[#3575](https://github.com/hat47x/tei-platform/issues/3575)、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §2.6。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **重複した事実の片方を削除し、導出に置き換えた**。

## 1. 目的

第22・23ラウンドで「手書きの一覧から漏れる」型を2つ閉じた。MCP Adapter も同じ構造を持つはずなので、どの表が独立しているかを測る。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `b26fb6f82acb3632f797cc1135f809cfbd511311`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 観測

### 3.1 tool を扱う表は4つ

| 表 | 場所 | 役割 |
| --- | --- | --- |
| `toolList()` | `main.go` | `tools/list` が広告する tool と `InputSchema` |
| `toolArguments` | `main.go` | `checkArguments` が受け入れる引数名 |
| `dispatch` の `switch` | `main.go` | 実際の呼出し先 |
| `cliCommandNameForTool` | `discovery_parity_test.go` | CLI コマンド名との対応（試験側） |

### 3.2 変異で測った検出力

| 変異 | 試験 | 実行時の挙動 |
| --- | --- | --- |
| `toolList()` に tool を追加（他はそのまま） | **FAIL**（3試験） | — |
| `toolArguments` に `InputSchema` が宣言しない引数を追加 | **PASS（検出できず）** | 却下される（別経路の `rejectUnknownArguments` が偶然止める） |
| `toolArguments` から宣言済みの引数を削除 | FAIL（2試験） | — |

**危険な方向（緩む方向）だけが無検査**だった。`toolArguments` が緩んでも各 tool の `rejectUnknownArguments` が止めるため実害は出ないが、同じ「宣言」を2箇所に書いており、`rejectUnknownArguments` を書き忘れた tool を追加すると素通りする。

`toolArguments` と `InputSchema.properties` の一致を確かめる試験は存在しなかった（`grep toolArguments --include=*_test.go` が0件）。

## 4. 修正

`toolArguments` を削除し、`InputSchema` から導出するようにした。

```go
func declaredArguments(tool tool) []string {
	properties, ok := tool.InputSchema["properties"].(map[string]any)
	if !ok {
		return nil
	}
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)   // エラー文の順序を安定させる
	return names
}

func toolByName(name string) (tool, bool) { ... }
```

`checkArguments` は `toolByName` → `declaredArguments` の順に読む。

**試験で一致を確かめるのではなく、事実を1箇所にした。** 試験は導出が壊れていないことを確かめる役に下がる。

## 5. 修正後の実測

| 呼出し | 結果 |
| --- | --- |
| `tei_model_catalog` + `namespace` | 受理され、結果を返す |
| `tei_model_catalog` + `nope` | `-32602` `unknown argument "nope" for tool "tei_model_catalog"` |
| `tei_validate` + `x` | `-32602` `unknown argument "x"` |
| `tei_validate` + 引数なし | 成功 |

受理と拒否の両方が従来どおりで、`tei_validate` の「宣言が空」も正しく効いている。

## 6. 試験

`tools/reference-mcp-readonly-adapter/tool_arguments_test.go`。

| 試験 | 内容 |
| --- | --- |
| `TestDeclaredArgumentsMatchTheSchema` | 導出結果が `inputSchema.properties` と一致する |
| `TestToolLookupMatchesTheAdvertisedList` | 広告した名前が解決し、未広告の名前は解決しない |
| `TestCheckArgumentsFollowsTheSchema` | 宣言済みは受理、未宣言は拒否 |
| `TestEveryRequiredArgumentIsADeclaredProperty` | `required` の各名前が `properties` にある |

わざと壊した2種:

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | `declaredArguments` が1つの property を落とす | FAIL: `declaredArguments=[], schema properties=[namespace]` |
| M2 | `required` に properties に無い名前を足す | FAIL: `tei_inspect requires "ghost" but does not declare it as a property` |

`go test ./tools/reference-mcp-readonly-adapter -count=1` は ok（39.5s）。gofmt・vet の指摘なし。

## 7. 互換性

挙動は変わらない（受理・拒否の集合は同一）。変わるのは「引数の一覧をどこに書くか」だけ。`toolArguments` を参照する外部コードは無い（同一パッケージ内の `checkArguments` のみ）。

## 8. 読み取れないこと（NOT RUN）

- `dispatch` の `switch` と `toolList()` の一致の機械検査。広告した tool に dispatch が無い場合は既存試験が拾うが、**dispatch にあって広告に無い** tool は検出できない（`checkArguments` が `toolByName` で先に弾くため実害は無いが、死んだ分岐として残りうる）。
- HTTP・Desktop の入口、実DB・実Provider・実Host、意味の受理・権限・配備。

## 9. 位置付け

「同じ事実を2箇所に書かない」を MCP の引数一覧で実行した。前2ラウンドは「手書き一覧が漏れたら試験で落とす」だったが、今回は**一覧そのものを消した**ので、漏れようがない。

残る `dispatch` と `toolList()` の関係は、`checkArguments` が広告を先に見るため実害が出にくい。ただし「広告に無い tool を dispatch に書く」ことは可能なので、必要になれば同じ手法（導出）で閉じられる。
