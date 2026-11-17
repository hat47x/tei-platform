# semantic/ の根に別の宣言を書くと、配置を言わない（2026-11-18、第102ラウンド）

- 対象: `internal/syntax/parser.go` の `ParseSemanticFile`。既存 #4169 の残り半分を解消。
- 前段: [第101ラウンド](../TEI_DOGFOODING_ROUND101_ISSUE4178_ALREADY_FIXED_2026-11-18/README.md)
- 状態: **`#4169` の本文が主張する2点のうち、残っていた1点を直した。**

## 1. 目的

前ラウンドで `#4178` が**本文ではなく実測で**判定できることを示した。同じ方法を `#4169` に適用した。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `981e02030`（`origin/main`） |
| CLI実行物（設計時） | `sha256:783b0e50ff381d71f80aad14d0a365a5a840d624e4c146e7a388f6338b7a76ef` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. `#4169` の主張2点を測った

### 主張1: 構文・文法・配置を返すコマンドが無い → **今も真**

```text
$ tei commands  -> 30 commands, grammar/syntax/help-like: (none)
```

catalog の `output` 説明にも配置案内は無い。**未解消。**

### 主張2: 5種すべてが「でたらめな語」と同じ診断 → **既に解消**

```text
browse        "browse" starts an application pattern, ... declare it as "pattern ...
record        "record" starts an application pattern, ...
reference / command / header-detail   同上
frobnicate    expected database, uses, or binds     <- 対照は汎用のまま
```

前段の修正（`#4082` の単一情報源を使う）が**効いている**。

## 4. 見つけた残り: 逆向きの配置

**semantic/ の根**に `pattern` を書くと:

```text
semantic/equip/misplaced-pattern.tei:1:1 error PARSE_ERROR expected "namespace"
```

**`pattern` は宣言表に載っているのに、配置を言わない。** 原因は `ParseSemanticFile` が `expectText("namespace")` を**先に**呼び、`misplacedDeclaration` の分岐に到達しないこと。

## 5. 修正

`expectText` の**前**に配置検査を置いた。`tok.Text != "namespace"` の除外も必要（`namespace` 自体が宣言表にあるため、除外しないと**正しいファイルが「宣言できない」と言われる**）。

```text
BEFORE  semantic/equip/misplaced-pattern.tei:1:1 error PARSE_ERROR expected "namespace"
AFTER   semantic/equip/misplaced-pattern.tei:1:1 error PARSE_ERROR
        "pattern" starts an application pattern, which cannot be declared in a
        semantic namespace source; declare it under applications/<application>/patterns,
        for example <application>/patterns/<id>.tei
```

**unknown 語（`frobnicate`）は汎用メッセージのまま。** 受理・拒否の範囲は変えていない。

## 6. 変異試験（2回やり直した）

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 配置検査を削る | FAIL `"pattern" kept the generic message: expected "namespace"` |
| M2 | `namespace` の除外を外す | FAIL `a valid semantic file was refused: "namespace" starts a semantic namespace, which cannot be declared…` |

**M2 は最初、除外を外しても2つのテストが通ってしまった。** 「未知語は汎用のまま」のテストは**通るのが正しい**（`frobnicate` は表に無いので配置を名指ししない）ため、**そのテストでは除外の必要性を検出できない**。**有効な semantic ファイルを入力する3本目を足してから**、M2 が落ちることを確認した。

## 7. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/syntax` | ok |
| `./internal/workspace` | ok |
| `./cmd/tei` | ok（234秒） |
| CI gate（`--fast`） | **6ステップすべて PASS** |

## 8. 読み取れないこと（NOT RUN）

- `#4169` の主張1（配置を引くコマンド）は**未解消**。設計判断を伴うため本ラウンドでは触っていない。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 9. 位置付け

**「起票の主張を1点ずつ測り、解消済みと未解消を分けた」回**である。主張2は既に直っており、その**逆向き**が残っていた。**変異試験が通ってしまう理由を疑い、テストを足してから再実行した**のが本ラウンドの要点である。
