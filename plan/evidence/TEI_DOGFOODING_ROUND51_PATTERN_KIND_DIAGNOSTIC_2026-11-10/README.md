# 設計作業をCLIだけで完遂し、`pattern` の種別診断を改善した（2026-11-10、第51ラウンド）

- 対象: [#4082](https://github.com/hat47x/tei-platform/issues/4082)（起票・本ラウンドで解消）。
- 前段: 第41〜50ラウンドは「モデルにあるが参照グラフに無い」型の追跡に集中した。本ラウンドは**実際の設計作業**に戻った。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **設計演習で摩擦点を1件検出し、その場で修正。**

## 1. 目的

計画 §5 の残った問い（「『効果的』の測り方」）に対し、**CLI の診断だけを頼りに新しい機能を設計できるか**を実際に試した。read-only の観測ではなく、設計作業そのものを行った。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `741fd79bf`（`origin/main`） |
| CLI実行物（修正前） | `sha256:735a8c67a8779cfd2ad42a968290753bf6634cee5b5037ed7b61fc81ef3870c5` |
| CLI実行物（修正後） | `sha256:e9aa023f45fe89b9f02948b6b5b9d8dd5eca064acb523109857ff90427b13e2e` |
| 作業場所 | `reference/workspace` の複製（`/tmp/design52`）。リポジトリは変更しない |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 設計演習

「`security.authenticate` の逆操作（失効）を追加する」という設計課題を、CLI の応答だけを頼りに進めた。

### 3.1 手順と結果

| # | 操作 | CLI の応答 |
| --- | --- | --- |
| 1 | `capability security.revoke` に `std.date-time` を使う | `PARSE_ERROR invalid canonical id "std.date-time"`（`file:5:30`） |
| 2 | `std.nonexistent` に変える | `UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE`（どの属性がどの型を参照しているか明示） |
| 3 | `std.datetime` に直す | `ok` |
| 4 | `pattern order_entry.revoke { ... }`（種別なし） | `PARSE_ERROR unexpected token "{"` |
| 5 | `pattern order_entry.revoke bogus { ... }` | `unsupported application pattern kind "bogus"` |
| 6 | `pattern order_entry.revoke command { ... }` | `ok` |
| 7 | `realize security.revoke with custom go` | `ok` |

### 3.2 設計は完遂できた

新しい capability は参照グラフに正しく現れた。

```text
$ tei trace -scope c -subject-kind semantic-capability -subject-id security.revoke -mode full
total: 6
  realizes  -> semantic-capability security.revoke
  capability-> semantic-capability security.revoke
  outcome   -> semantic-data security.revocation_ack
  input     -> semantic-data security.authenticated_subject
  outcome   -> semantic-data security.authentication_denial
  input     -> semantic-type std.text
```

application 側・pattern 側からも辿れる。

```text
$ tei trace -scope c -subject-kind application -subject-id order_entry -direction outgoing
  owns-pattern -> application-pattern order_entry.revoke
  realizes     -> semantic-capability security.revoke

$ tei trace -scope c -subject-kind application-pattern -subject-id order_entry.revoke
  total: 2
  owns-pattern -> application-pattern order_entry.revoke
  capability   -> semantic-capability security.revoke
```

**設計 → 検証 → グラフ確認のループは機能した。**

### 3.3 摩擦点

`validate` の型診断は高品質だった（`file:line:col` と参照元を明示）。一方、**`pattern` の種別に関する診断だけが手がかりを出さなかった。**

```text
applications/order-entry/patterns/revoke.tei:1:28  error  PARSE_ERROR  unexpected token "{"
```

位置は正確だが、**「種別が必要である」ことも「どんな種別があるか」も言わない。** CLI から種別集合を知る手段は無く、私は `reference/workspace` の既存パターンを `grep` して `reference`/`record`/`header-detail`/`command`/`browse` を割り出した。

不正な種別を書いた場合は `unsupported application pattern kind "bogus"` と出るが、これも**正しい集合を挙げない**。

## 4. 修正

### 4.1 単一情報源にした

`internal/application/model.go` に `PatternKinds` を置き、`Valid()` がそれを読むようにした。診断文と受理集合が同じ定義を参照するので、片方だけ更新して食い違うことがない。

```go
var PatternKinds = []string{
	string(PatternBrowse), string(PatternRecord), string(PatternReference),
	string(PatternCommand), string(PatternHeaderDetail),
}

func (k PatternKind) Valid() bool {
	for _, valid := range PatternKinds {
		if string(k) == valid {
			return true
		}
	}
	return false
}
```

### 4.2 診断で種別を列挙する

`internal/syntax/application_pattern.go` で、種別トークンが識別子でない場合と、識別子だが不正な場合の両方で一覧を出す。

```text
変更前: unexpected token "{"
変更後: pattern order_entry.revoke requires a kind before its body;
        valid kinds are browse, record, reference, command, header-detail

変更前: unsupported application pattern kind "bogus"
変更後: unsupported application pattern kind "bogus";
        valid kinds are browse, record, reference, command, header-detail
```

### 4.3 修正後の実測

修正後の実行物（`sha256:e9aa023f…`）で同じ2ケースを再実行し、上記の文面が出ることを確認した。正常な `command` パターンは引き続き `ok`。

## 5. 試験

| ファイル | 試験 |
| --- | --- |
| `internal/syntax/pattern_kind_diagnostic_test.go` | 種別なしのとき一覧を出す／不正な種別のとき一覧を出す／有効な種別が「種別として」拒否されない |
| `internal/application/pattern_kinds_test.go` | `PatternKinds` と `Valid()` が一致する／宣言された定数が漏れなく載っている／`""`・`"bogus"`・`"Browse"`・`"browse "` を拒否する |

### 5.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 種別チェックの追加を削除（元の挙動） | FAIL `error does not explain that a kind is required: 1:28: unexpected token "{"` |
| M2 | `PatternKinds` から `command` だけ削除 | FAIL（`TestPatternKindsAndValidAgree`） |

## 6. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/syntax ./internal/application ./internal/workspace ./internal/operations` | すべて ok |
| `./cmd/tei` | ok（165秒前後） |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 7. 互換性

**診断文の変更のみ。** 受理する構文・受理する種別集合は変わらない。有効なパターンは従来どおり解析できる。

## 8. 読み取れないこと（NOT RUN）

- 他の構文要素（`capability`・`data`・`interaction` など）でも同様の「列挙されない語彙」があるか。今回は `pattern` の種別だけを扱った。
- `std` 型の一覧を CLI から引く手段。`std.date-time` のときに「canonical id が不正」とは出るが、**正しい `std` 型名の一覧は CLI から得られない**。今回はリポジトリのソースを読んで解決した。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

**設計演習そのものは成功した。** capability・pattern・realize を追加し、グラフで3方向から確認できた。診断も型については十分だった。

一方で、**語彙の列挙という一点だけが CLI から得られなかった。** 生成AIはリポジトリを読めるので回避できたが、「CLI 利用主体」という制約の下では、これは設計者を止める種類の摩擦である。

`pattern` の種別は単一情報源にできたので直した。`std` 型の一覧は索引の設計に関わるため、記録に留める。
