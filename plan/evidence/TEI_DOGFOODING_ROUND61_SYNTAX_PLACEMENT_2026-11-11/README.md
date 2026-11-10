# 構文の配置を教える: パターン種を application 本体に書いたとき（2026-11-11、第61ラウンド）

- 対象: 第60ラウンドで記録した「構文の配置を引くコマンドが無い」という穴のうち、**実際に踏んだ1点**。
- 前段: [第60ラウンド](../TEI_DOGFOODING_ROUND60_DESIGN_VERTICAL_2026-11-10/README.md)。
- 実行日: 2026-11-11（Asia/Tokyo）。
- 状態: **パターン種を書いたときの診断が配置を名指しするようにした。一般的な構文探索の穴は残る。**

## 1. 目的

第60ラウンドの設計演習で、`browse` を application 本体に書いて `expected database, uses, or binds` と言われ、**どこに書くべきか分からなかった**。本ラウンドはこの1点を測り直し、直せるか判断する。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `d2a250bb1`（`origin/main`） |
| CLI実行物（修正前） | `sha256:c02f581f1ac29b90e8de888a852e625b01f7d3e5b61bb4c87379482c89f4b873` |
| CLI実行物（修正後） | `sha256:5127df76fbd1a70809ade06f7b6745fbf7039085acc5ce8bc1131b2ee5ce2b79` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 修正前の実測

### 3.1 構文を引くコマンドは無い

`tei commands` の21コマンドに、構文・文法・配置を返すものは無い。

```text
version, model catalog, model overview, model locate, workspace-files, inspect,
view, trace, source-index, knowledge-context, knowledge-qualify,
knowledge-review-plan, knowledge-review-handoff, knowledge-review-disclose,
knowledge-data-design-candidate-review, validate, world, requirements,
plugin-lock, plugin-resolve, plugin-runtime-plan
```

### 3.2 パターン5種すべてが「でたらめな語」と同じ扱い

```text
browse        PARSE_ERROR  expected database, uses, or binds
record        PARSE_ERROR  expected database, uses, or binds
reference     PARSE_ERROR  expected database, uses, or binds
command       PARSE_ERROR  expected database, uses, or binds
header-detail PARSE_ERROR  expected database, uses, or binds

frobnicate    PARSE_ERROR  expected database, uses, or binds   <- 対照
```

**誤った配置と打ち間違いが区別できない。** 診断は「この語は知らない」としか言わない。

## 4. 修正

`internal/syntax/parser.go` の application 本体の `default:` で、語が `application.PatternKinds` に含まれるかを先に見る。

```go
if isApplicationPatternKind(t.Text) {
	return a, p.errAt(t, fmt.Sprintf(
		"%s starts an application pattern, which is not declared in the application body; "+
			"declare it as \"pattern <application>.<id> %s { ... }\" in its own file",
		strconv.Quote(t.Text), t.Text))
}
```

**`application.PatternKinds` は第52ラウンドで単一情報源として作ったもの**（`#4082`）。パターン解析器が受理するのと同じリストを読むので、種の追加時に取り残されない。

### 4.1 修正後

```text
browse   PARSE_ERROR  "browse" starts an application pattern, which is not declared in
                      the application body; declare it as "pattern <application>.<id>
                      browse { ... }" in its own file
```

5種すべてが自分の配置を名指しする。**未知の語（`frobnicate`）は汎用メッセージのまま**で、既存の2つのピン（`internal/syntax/error_test.go`・`internal/workspace/source_diagnostic_test.go`）も無変更で通る。

正しい配置（`patterns/*.tei` に `pattern probe.something record { ... }`）は `ok` のまま。

## 5. 試験

| 試験 | 内容 |
| --- | --- |
| `TestPatternKindInApplicationBodyNamesItsPlacement` | **`application.PatternKinds` を回す**ので、種が増えれば自動で対象になる。種名を引用し、宣言形を示し、汎用メッセージでないこと |
| `TestUnknownApplicationBodyTokenKeepsTheGenericMessage` | 未知語は汎用メッセージのまま |
| `TestPatternPlacementErrorRangeCoversTheKind` | 範囲が壊れた語（`record`）だけを指す |

### 5.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 配置案内の分岐を削る | FAIL `browse: message does not quote the kind` |
| M2 | リストではなく `browse`/`record` の2種だけ見る | FAIL `reference: message does not quote the kind` |
| M3 | メッセージを汎用に戻す | FAIL `browse: message is still the generic one` |

M2 は「単一情報源を使わずに書き写した」場合を模している。**リストを回す試験なので、書き写しは3種目で落ちる。**

## 6. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/syntax` | ok |
| `./internal/workspace` | ok |
| `./cmd/tei` | ok（101.9s） |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 7. 残った穴（本ラウンドでは閉じない）

**構文の配置を一般的に引く手段は無い。** 今回直したのは「パターン種を application 本体に書いた」という1ケースだけである。

未解決のまま残るもの:

- `pattern` の置き場所（専用ファイル）を知る手段が、診断にもコマンドにも無い。**正しい書き方を知らない人が、正しい書き方を知るために間違える必要がある。**
- 他の配置（`realize`、environment のバインディング、`contract-query-executor-binding` 等）も同様。
- `UNBOUND_DATASOURCE` は何が足りないかは言うが何を書くかは言わない（第60ラウンド記録、未起票のまま）。

[#4169](https://github.com/hat47x/tei-platform/issues/4169) として起票した。

## 8. 読み取れないこと（NOT RUN）

- 一般的な構文探索の実現方法（案は #4169 に書いた）。
- `UNBOUND_DATASOURCE` の正しい解消手順。
- #4128 の A・B・D。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

**「語彙は教えるが配置は教えない」という穴の、最初の1点を閉じた回**である。第53〜56ラウンドで整えた語彙診断は値の集合に効くが、配置には効かない。今回はその区別を実測で示し、最も踏みやすい1ケースを直した。

完全な解決ではない。**正しい書き方を知らない人が、間違えることでしか正しい書き方を知れない**という構造は残っている。
