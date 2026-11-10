# D-1（catalogの`scope`拡張）を採るかの判断材料

[#2831](https://github.com/hat47x/tei-platform/issues/2831) のSB-C0 §4.5が**D-3（catalogへ載せない）**を既定案として残したまま、§7に未決として残っている**D-1（`scope`を`selected-commands`へ広げる）**の判断材料を実測した記録です。

**判断は下しません。** 何が壊れ、何が壊れないかを測ります。

## 現状の実測

| 項目 | 値 |
| --- | --- |
| `acceptedCommands`（dispatch するコマンド） | **50** |
| `cliCommandCatalog` の record | **19**（`model catalog` のように1 recordが2語の名前を持つものがある） |
| dispatch するが広告しない | **36**（50件のうち、catalogに同名の record があるのは14件） |
| catalog の `effect` の値 | **`read-only` のみ**（19件すべて） |
| `scope` | `selected-read-only-commands` |

**広告しない36件には、書込み系（`deployment-*`・`issue-grant`・`artifacts`）と読取り系（`dbcheck`・`world`隣接）が混在している。** つまり現行の `scope` は「dispatch する全部」ではなく「**読取り専用として選んだ一部**」を意味しており、その意味で一貫している。

## D-1を採ると何が壊れるか

### 1. 二つの不変条件テスト

| 場所 | 内容 |
| --- | --- |
| `cmd/tei/cli_contract_test.go:12` | `catalog.Scope != "selected-read-only-commands"` で失敗 |
| `cmd/tei/cli_contract_test.go:17` | `cmd.Effect != "read-only"` で失敗 |

D-1は `scope` を変え、`effect: writes-artifact` の entry を足すので、**両方が落ちる**。テストの更新は機械的にできるが、それは「契約を変えた」ことの記録であって、変更が無害だったことの証明ではない。

### 2. 実在する consumer が一つ壊れる

`tools/reference-mcp-readonly-adapter/discovery_parity_test.go:98` は、adapter が公開する各 tool について CLI catalog の `effect` を確認し、**`read-only` 以外なら失敗**する。

```go
if record.Effect != "read-only" {
    t.Fatalf("%q is advertised as effect=%q; the adapter must expose read-only operations only", command, record.Effect)
}
```

**これは adopter の前提そのものである。** MCP adapter は「読取り専用の操作だけを公開する」と名乗っており、その名乗りを CLI catalog の `effect` で裏づけている。D-1で catalog に書込み系が混ざると、**adapter は自分が公開しない tool の隣に書込み系の entry が並ぶ catalog を読む**ことになる。adapter のテストは「自分が公開する tool」だけを見るので直ちには落ちないが、**catalog を「読取り専用の一覧」として読む前提は壊れる**。

### 3. `scope` の意味が変わる

`selected-read-only-commands` は「選んだ読取り専用コマンド」という意味を持つ。`selected-commands` へ広げると、**同じ `scope` を読む既存 consumer が「全部読取り専用」と読む可能性**が残る（`effect` を見る実装なら問題ないが、`scope` だけを見る実装は壊れる）。本リポジトリの consumer は `effect` も見ているが、**外在する consumer の実装までは測れない**。

## D-1を採ると何が改善するか

| 改善 | 内容 |
| --- | --- |
| `tei commands` で `self-build` が見える | 呼び手が `-h` や文書を読まずに存在を知れる |
| catalog の広告と実応答の一致を照合に載せられる | 「広告したのに動かない」「動くのに広告しない」を機械的に検出できる |

ただし**後者は D-1 を採らなくても部分的にできる**（広告しない36件について「dispatch するが広告しない」ことの一覧を維持する）。

## D-2（別の discovery 面）との比較

| 観点 | D-1 | D-2 |
| --- | --- | --- |
| 既存 consumer の前提 | **壊れる**（`scope` の意味が変わる） | 壊れない（新しい面を足すだけ） |
| catalog の設計方針 | 一つの catalog に `effect` で混ぜる | 「第二の registry を作らない」方針と衝突しないか要確認 |
| 実装量 | `scope` と `effect` の値、テスト2件 | 新しい flag か新しい応答、consumer の追加 |

## 所見

**D-1は「テスト2件を直す」では済まない。** `scope` は公開契約の一部であり、それを読む consumer の前提を変える。本リポジトリ内で測れる consumer は `effect` も見ているが、**外在する consumer の実装は測れない**。§4.5が「contract decision である」と書いたのはこの点を指している。

**D-3のままでも `self-build` は使える。** `-h` と文書で知れる。欠けるのは発見性だけである。

**したがって本記録はD-1を推奨しない。** 判断には外在 consumer の扱い（無いと見なすか、`effect` を見ることを契約に書くか）を決める必要があり、それは本稿の範囲外である。

## 検証

- `acceptedCommands` と catalog の件数: `cmd/tei/cli_commands.go`・`cmd/tei/cli_contract.go`を機械的に数えた（50 / record 19 / 広告しない36 / `read-only` 19件）
- 不変条件: `cmd/tei/cli_contract_test.go:12`・`:17`を読んで確認
- 実 consumer: `tools/reference-mcp-readonly-adapter/discovery_parity_test.go:98`を読んで確認
- `go test ./tools/repository-plan-check -count=1`：PASS

## 限界

- **外在 consumer は測れていない。** 本リポジトリの外に catalog を読む実装があれば、その影響は不明。
- 本記録は**判断材料**であり、コードも設計も変えない。
- D-2の「第二の registry を作らない」方針との整合は、`plan/design/CLI_FIRST_INTERFACE_DESIGN.md`を読んだ範囲では未決である。
- G4は`UNQUALIFIED`のまま。
