# 自分の誤りを訂正し、`invalid canonical id` に文字規則を載せた（2026-11-10、第52ラウンド）

- 対象: [#4082](https://github.com/hat47x/tei-platform/issues/4082)（前ラウンドで起票・クローズ。本ラウンドで記述を訂正）、[#2620](https://github.com/hat47x/tei-platform/issues/2620)（`std` の扱い）。
- 前段: [第51ラウンド](../TEI_DOGFOODING_ROUND51_PATTERN_KIND_DIAGNOSTIC_2026-11-10/README.md)。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **公開済みの誤った主張を訂正し、残る実際の欠陥を修正。**

## 1. 目的

第51ラウンドで「`std` 型の一覧は CLI から引けない」と記録した。本ラウンドはその解消に着手する予定だった。**着手前の再実測で、その主張自体が誤りだと分かった。**

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `5d90801e7`（`origin/main`） |
| CLI実行物（測定時） | `sha256:6dcaf09870135e8e654e6b1666712d61abc6065400cd8a4b6788f31e80e6757b` |
| CLI実行物（修正後） | `sha256:930846015bb844e1276c70065a0d405dbeda9b5c393556df5fe16a99d226ecf2` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 訂正: `std` 型は CLI から引ける

| 経路 | 結果 |
| --- | --- |
| `model catalog -workspace <ws>` | `semantic-type` かつ `std.` 始まりが **12件**（`std.binary`〜`std.time`） |
| `inspect -subject-kind semantic -subject-id std` | **12件**、各 `kind=primitive` |
| `model locate -subject-kind semantic-type -subject-id std.datetime` | `ok=true` |

**3経路すべてで引ける。** 私の第51ラウンドの記述は誤りだった。

### 3.1 誤った理由

第51ラウンドでは探査用ワークスペースに `std.date-time` の解析エラーを残したまま `model catalog` を実行し、**0件**を得た。それを「std がカタログに無い」と読んだ。

実際は**ワークスペースのロード自体が失敗していた**ので、カタログは空だった。**壊れた状態から一般化した**のが誤りである。

```text
$ tei inspect ... -subject-id std      # 当時
{"ok":false,"error":{"code":"WORKSPACE_LOAD_FAILED", ...}}
```

この応答がすでに「ロード失敗」と言っていた。私は `model catalog` の空配列だけを見て判断した。

### 3.2 訂正の反映先

| 場所 | 対応 |
| --- | --- |
| `plan/evidence/.../53-not-done.txt` | 原文を残し、`CORRECTION` 節を追記 |
| `plan/workstreams/...md` §53 | 該当行を打ち消し線＋訂正に置換 |
| [#4082](https://github.com/hat47x/tei-platform/issues/4082) | 訂正コメントを追記 |

**原文を消さずに残した。** 何を信じていたかの記録自体に意味があるため。

## 4. 残っていた実際の欠陥

`std.date-time` の診断は `invalid canonical id "std.date-time"` で、**文字規則を言わない**。著者は「型が無い」のか「文字が不正」なのか区別できない。

規則は `core/identity/id.go` の `validCanonicalID` に `^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$` として在る。届いていなかった。

### 4.1 修正

```text
変更前: invalid canonical id "std.date-time"
変更後: invalid canonical id "std.date-time"
        (each dot-separated segment must match [a-z][a-z0-9_]*)
```

`ParseLocalName` も同様に規則を載せた。

### 4.2 修正後の実測

| 入力 | 診断 |
| --- | --- |
| `std.date-time` | `invalid canonical id "std.date-time" (each dot-separated segment must match [a-z][a-z0-9_]*)` |
| `std.Datetime` | 同上（`std.Datetime`） |
| `std.foo`（構文は正当） | `UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE ... unknown semantic type std.foo`（変化なし） |
| `database Bad-Alias` | `invalid local name "Bad-Alias" (must match [a-z][a-z0-9_]*)` |
| 正常なワークスペース | `ok` |

## 5. 試験

`core/identity/id_rule_diagnostic_test.go` の3件。

| 試験 | 内容 |
| --- | --- |
| `TestInvalidCanonicalIDNamesTheRule` | 不正 id の診断が文字規則を含む |
| `TestTheQuotedRuleMatchesTheAcceptedLanguage` | **引用した規則が実際の受理言語と一致する**（規則を満たす id は受理され、破った id は拒否される） |
| `TestInvalidLocalNameNamesTheRule` | ローカル名の診断も規則を含む |

2つ目が重要である。診断文が規則を*嘘をつかずに*語ることを固定している。

### 5.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 規則を削って元の文面に戻す | FAIL `message does not state the accepted character rule: invalid canonical id "std.date-time"` |
| M2 | 規則を**誤った内容**（`[a-z0-9_-]*`）に変える | FAIL（等価性テスト3件が同時に落ち、参照実装との不一致を検出） |

M2 は、診断文が規則を偽っても通らないことを示す。

### 5.2 既存テストへの影響

`core/identity/id_equivalence_test.go` は**エラー文字列を厳密比較**していたため、参照実装側の文面も同じに更新した。このテストが固定している主眼は「受理する文字列の集合が正規表現定義と一致すること」であり、それは変わっていない。

## 6. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./core/identity` | ok（8.5s） |
| `./internal/plugin/...`（5パッケージ） | ok |
| `./internal/syntax ./internal/workspace ./internal/operations` | ok |
| `./internal/realization/...` | ok |
| `./tools/reference-durable-run-golden` | ok |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

`invalid canonical id` を `strings.Contains` で見ている箇所は19件あり、いずれも文面の追加で壊れない。

## 7. 調べたが欠陥ではなかったこと

`attribute Bad-Name type std.text` が `ok` になったので疑ったが、`SymbolToSegment("Bad-Name") → "bad_name"` で、**記号は表示名であり正規化される**仕様だった（`badName` も `bad_name` になる）。欠陥ではないので起票していない。

## 8. 読み取れないこと（NOT RUN）

- `capability`・`data`・`interaction` などの語彙が CLI から引けるか。第51ラウンドで「同じ問いが残る」と書いたが、**今回 `std` については誤りだったので、他の語彙も同様に誤りの可能性がある。**未検証。
- `std` を `.tei` 化するか（[#2620](https://github.com/hat47x/tei-platform/issues/2620) の Owner 判断）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

**公開済みの主張を、測り直して訂正した回**である。第51ラウンドは「設計演習は成功、残る問いは std の語彙」と締めたが、その残る問いは存在しなかった。

原因は**壊れた探査状態から一般化したこと**。第43ラウンド（PR の状態だけを見た）、第49ラウンド（`tei world` を試さず「CLI で答えられない」と書いた）と同じ型で、**3回目**である。

今回は「解消に着手する前に、まず主張を測り直す」を実行したことで、誤りを公開後の1ラウンドで捕まえられた。前2回は他者に指摘されるか、別の作業で偶然判明していた。
