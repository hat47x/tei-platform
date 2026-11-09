# trace -mode full が参照0件のとき rows を出さない（2026-11-09、第15ラウンド）

- 対象: [#3693](https://github.com/hat47x/tei-platform/issues/3693)、[#3456](https://github.com/hat47x/tei-platform/issues/3456)（同じ種類の先行修正）、[#3419](https://github.com/hat47x/tei-platform/issues/3419)、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §2.3・§5 T01/T07。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の比較実測**。この記録は #3693 を起票した根拠であり、修正は `main` 側で行われた。

## 1. 目的

第14ラウンドまでで `knowledge-*` の連鎖を通した。このラウンドは、その連鎖の手前にある `trace` の応答が、関係0件の Workspace でも機械的に読めるかを確かめる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source（比較の基準） | `cd45e05edf4a8d2fec998580b8eed3a9ae57c80f`（当時の `origin/main`） |
| CLI実行物 | `sha256:49c0ef003b8799df66adfeb90a56f0505a57edf0af9c9f81e8d37c2f5bf9243d` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

対象Workspaceは2つ。関係を持たない最小Workspace（`/home/hat47x/dogfood/r15-ws`）と、関係を持つ `reference/workspace`。

```tei
namespace sales

data order {
    attribute orderId type std.text
    attribute customerId type std.text
    attribute note type std.text
}

data customer {
    attribute customerId type std.text
}

data product {
    attribute productId type std.text
}
```

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-head.txt` | 対象 commit |
| `30-validate.json` | 最小Workspaceの validate（`ok:true`、宣言9件） |
| `31-before.json` | 修正前: 関係0件の `-mode full` |
| `33-summary.json` | 修正前: 同じWorkspaceの `-mode summary` |
| `40-fixed-binary.txt` | 記録中に試した作業ツリーの実行物（採用していない） |
| `41-full-after.json` | その実装での関係0件 `-mode full` |
| `42-summary-after.json` | その実装での `-mode summary`（rows は出ない） |
| `43-ref-after.json` | 関係ありの `-mode full`（rows 2件、`nextOffset`、`continuation`） |
| `44-beyond-after.json` | `-offset 99`（rows は `[]`） |
| `45-mutations.txt` | 試した実装をわざと壊した3種の検出 |
| `46-packages.txt` | 影響パッケージの試験 |
| `47-gofmt.txt` | 0 byte |
| `01-inspect-missing-id.json`・`02-inspect-empty-id.json`・`03-inspect-absent.json` | 必須引数欠落の分類（副次的な観測） |

## 4. 観測（修正前）

関係0件のWorkspaceで `-mode full` を実行すると、`result.trace` に `rows` が**存在しない**。

```json
{"ok":true,"result":{"trace":{"mode":"full","total":0,"unresolved":0,
  "graphFingerprint":"sha256:0c717ba1…","snapshotFingerprint":"sha256:0c717ba1…"}}}
```

`mode=summary` も `rows` を持たない。両者は `mode` と `total` で区別できるが、**`rows` という欄の有無は同じ**である。

原因は `internal/modelref/trace.go` の `Rows` の `json:"rows,omitempty"` タグ。`omitempty` は nil だけでなく**長さ0のスライスも落とす**ため、`mode=full` の「0件という答え」と「欄が無い」が同じ表現になる。

`mode=summary` が `rows` を出さないのは契約どおりである。summary は全体を数えるだけで行を確保しない（`internal/modelref/trace.go` の `if options.Mode == "summary" { continue }`）。

## 5. main での解消

起票のあと、`main` の `3799997a0`（「fix(cli): trace full の参照0件でも rows を [] で返す (#3693)」）が同じ問題を解消した。`TraceResult` に `MarshalJSON` を足し、`mode=full` では常に `rows` を書く。フィールドを宣言順に明示することで、`rows` を強制しても文書の順序が変わらないようにしている。

修正後の実測（`main` `3ac019807`）:

| 入力 | `rows` |
| --- | --- |
| 関係0件・`mode=full` | `[]` |
| 関係0件・`mode=summary` | 出ない（契約どおり） |
| 関係あり・`mode=full -limit 2` | 2件（変更なし） |
| 関係あり・`-offset 99` | `[]` |

## 6. 重複の扱い

この記録のために私が書いた実装（`mode=full` かつ0件のときだけ上書きする案）は採用しなかった。最初に `mode=full` のすべてを上書きする形で書き、埋め込み構造体が `rows` を最後に回すため既存の full 応答のバイト列が変わり、`reference-cli-trace-benchmark` の golden が2件 FAIL した。0件のときだけに絞れば通ることは確認したが、`main` の形（全欄を宣言順に明示）が上位互換であるため、コードは持ち込まず実測記録だけを残す。

## 7. 読み取れないこと（NOT RUN）

- HTTP・Desktop の入口。
- 実DB・実Provider・実Host、意味の受理・権限・配備。
- `trace` 以外のコマンドで同じ種類の欄落ちをしないことの全面確認（`source-index` は #3456 で修正済み）。

## 8. 位置付け

「`mode=full` は全行を返す」という契約に対し、0件のときだけ欄が消えていた。0件と欄の不在を区別できるようにする必要がある、という根拠を残した記録である。解消は `main` の `3799997a0`。
