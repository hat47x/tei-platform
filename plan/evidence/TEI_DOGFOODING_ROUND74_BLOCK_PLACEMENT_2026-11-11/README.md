# ブロック内専用の項目を閉じ括弧の後ろに書いたとき、所属を名指しする（2026-11-11、第74ラウンド）

- 対象: [#4169](https://github.com/hat47x/tei-platform/issues/4169) の A 案、5件目。
- 前段: 第64ラウンドで `bind` の置き場所を構文エラーとして踏んだ。
- 実行日: 2026-11-11（Asia/Tokyo）。
- 状態: **閉じ括弧の後ろに書いた「ブロック内専用の項目」が、どのブロックに入るかを名指しするようになった。**

## 1. 目的

第64ラウンドで `bind` を environment ブロックの外に書いて `unexpected trailing token "bind"` になった。**「どこに入るか」を診断が言わない**点を直す。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `7cef632ee`（`origin/main`） |
| CLI実行物（修正前） | `sha256:71554b9f3e0436130d6f24dce5fbf24c1e039c5f0d5cd5152ec490131273a24d` |
| CLI実行物（修正後） | `sha256:82598abe59d7cb56559a044ac6bd89775fe460d93f300a21e175624ed7678cc6` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 途中で自分の探査が壊れていた

最初の3つの探査は**すべて同じ警告**を返した。

```text
warning WORKSPACE_SOURCE_ROOT managed file "... is a TEI source that no loader reads..."
```

**ファイル名を `staging-stray.tei` にしていたため、ローダが読んでいなかった。** ローダは `environments/<id>/environment.tei` という正確な名前を要求する。**第72ラウンドで自分が入れた警告が、その探査の誤りを教えてくれた。**

正しい名前で置き直すと、狙った2つの診断が出た。

```text
'binding' と綴りを間違えた（ブロック内）:
  expected plugin, bundle, feature, database-instance, bind, resource, grant,
  lease-policy, deployment-executor, observation-provider, or evidence-provider
  ↑ これは語彙を列挙しており、良い診断

閉じ括弧の後ろに bind:
  unexpected trailing token "bind"
  ↑ 置き場所を言わない。これが直す対象
```

## 4. 修正

`expectEOF` で、まず既存の `misplacedDeclaration`（トップレベル宣言の誤配置）を見、次に**ブロック内専用の項目**を見る。

```go
if blockOnly := blockOnlyEntryHint(t.Text); blockOnly != "" {
	return p.errAt(t, fmt.Sprintf(
		"%q cannot appear after the closing brace of its block; it belongs inside one, for example %s",
		t.Text, blockOnly))
}
```

対象: `bind`・`resource`・`grant`・`lease-policy`・`deployment-executor`・`observation-provider`・`evidence-provider`・`feature`（environment 内）、`column`・`primary-key`・`index`（persist 内）。

### 4.1 修正後

```text
"bind" cannot appear after the closing brace of its block; it belongs inside one,
for example environment staging { ... ; bind <application>.<alias> -> <database-instance> }
```

**未知の語（`frobnicate`）は汎用メッセージのまま**、トップレベル宣言（`migration`）は既存の「自前のファイルに書く」メッセージのまま。

## 5. 試験と変異

| 試験 | 内容 |
| --- | --- |
| `TestBlockOnlyEntryAfterClosingBraceNamesTheBlock` | 所属と宣言形を示し、汎用メッセージでない |
| `TestBlockOnlyEntryHintCoversBlockEntriesOnly` | 8種がヒントを持ち、`frobnicate`・`application`・`environment`・空・`migration` は持たない |
| `TestTopLevelKeywordStillGetsTheDeclarationMessage` | トップレベル宣言は既存の文言を保つ（**優先順位の固定**） |

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | ブロック判定を削る | FAIL `message does not say the entry belongs inside a block: unexpected trailing token "bind"` |
| M2 | 未知の語にもヒントを返す | FAIL `"frobnicate" must keep its own message` |

**M1 は最初アンカーが合わず `ValueError: substring not found` で落ちた。** 変異を作り直した（壊れた変異に気づくのは4回目）。

## 6. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/syntax` | ok |
| `./internal/workspace` | ok |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 7. #4169 の進捗

A 案は5件（パターン配置・契約の版・bind・interaction の形・閉じ括弧後の項目）。

## 8. 読み取れないこと（NOT RUN）

- ブロック内で綴りを間違えた場合の語彙列挙は既に良いので、変更していない。
- 他のブロック（pattern・application・persist 以外）の専用項目をどこまで表に載せるか。
- #4128 の A・B・D。実DB・実Provider・実Host、HTTP・Desktop。

## 9. 位置付け

**自分の探査の誤りを、前ラウンドで入れた警告が捕まえた回**である。`staging-stray.tei` という名前で3回続けて同じ結果が出たとき、警告が「読まれていない」と言ったので気づいた。**自分で入れた道具が自分の誤りを検出した**のは、道具が効いている証拠である。
