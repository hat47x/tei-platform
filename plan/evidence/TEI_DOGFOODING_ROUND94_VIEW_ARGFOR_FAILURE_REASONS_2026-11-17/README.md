# `tei view` の引数拒否が、私が実際に渡した引数と食い違っていた（2026-11-17、第94ラウンド）

- 対象: `cmd/tei/cli_view.go` の引数ガード。新規起票なし（その場で解消）。
- 前段: [第93ラウンド](../TEI_DOGFOODING_ROUND93_SQL_PLACEHOLDER_MESSAGE_2026-11-17/README.md)
- 状態: **拒否が「どの flag が悪いか」を名指しするようになった。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行した。**宣言した Application が `tei view` で実際に読めるか**を、利用者の経路で確かめる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `28c2b5252`（`origin/main`） |
| CLI実行物（設計時） | `sha256:715b5d45b48aa18d92fb7f110b73ba0f7c27e31d305dc63becf9d6171c8d23dc` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 動いたこと（先に確認）

| 経路 | 結果 |
| --- | --- |
| `view -application loan_desk` | アプリ・semanticUses・パターン・fingerprint を表示 |
| `view -pattern ... browse` | data・query・**列一覧**（`1. borrower : std.text`）を表示 |
| `-format text/json/html/print` | **4形式すべて rc=0**（html は実際に table を描画、print も出力） |
| 交差参照 | `APPLICATION_PATTERN_QUERY_APPLICATION_MISMATCH` で**別アプリのクエリを拒否** |

**宣言は本当に読める。** `model overview` の `needsAttention` も、私が入れた不正な結果列を**その subject で捕まえた**。

## 4. 見つけた穴: 拒否メッセージが argv と食い違う

```text
$ tei view -workspace W -application loan_desk -pattern loan_desk.loan_browse -execute
error INVALID_ARGUMENT
  view requires exact -workspace and -application, at most one of -pattern,
  -package, or -command, without positional arguments
```

**私は `-workspace` と `-application` を渡し、`-pattern` を1つだけ渡していた。** メッセージは「exactly にしろ」「at most one にしろ」と言うが、**私はその通りにしていた。** 本当の理由は `-execute` が `-command` を要求すること。

原因はガードの形:

```go
if fs.NArg() != 0 || *root == "" || ... || (*execute && *commandID == "") || ... {
    return fail("INVALID_ARGUMENT", "<one message for 20 conditions>", 2)
}
```

**20個の条件が1つの文言を共有**していた。

## 5. 修正

`switch` に分け、**それぞれが悪い flag を名指し**するようにした。元の受理・拒否の範囲は保つ（`-environment` の複合条件は意味を変えずに統合）。

```text
BEFORE  20条件 → 1文言
AFTER   view -execute requires -command; a browse pattern is read-only and is not executed
        view -period-state requires -execute
        view -allow-capability requires -execute
        view -input requires -command or -environment
        view does not take positional arguments
        ... （条件ごと）
```

**拒否する組合せは変えていない。** 変えたのは**説明**だけである。

## 6. 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 1つの case を旧汎用文言に戻す | FAIL `message does not say "-execute requires -command"` |

最初の変異は `switch` を壊してビルド不能になった（**変異が壊れているのは証拠にならない**）。組み立て直して、ビルドが通る形で再実行した。

## 7. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./cmd/tei` | ok（581秒） |
| `./internal/applicationview` | ok |
| `./internal/validation` | ok |
| CI gate（`--fast`） | **6ステップすべて PASS** |

## 8. 読み取れないこと（NOT RUN）

- ブラウズの**列が data より少ない**ことを検査すべきか（設計判断。現状は未検査）。
- 実DB への接続、実 Provider 起動。
- 実Host、HTTP・Desktop。

## 9. 位置付け

**「要求はするが、従い方が書いていない」の4例目**である。今度は**私が既に従っていた**ので、文言が**私の argv と矛盾**した。**拒否の範囲を変えず、理由だけを分けた。**
