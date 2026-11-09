# 実装を直したあと文書が古い符号を主張していた（2026-11-09、第18ラウンド）

- 対象: [#3420](https://github.com/hat47x/tei-platform/issues/3420)、[#3733](https://github.com/hat47x/tei-platform/issues/3733)（第17ラウンドで解消）、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §2.3「失敗」。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の確認と、文書の修正および再発防止の試験追加**。

## 1. 目的

第17ラウンドで「必須引数の不足」を `INVALID_ARGUMENT` に統一した。その変更が、同じ規則を説明している他の文書と一致しているかを確かめる。あわせて、並行して出ていた重複 PR を整理する。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `022c85bd97b479a4093a1b8287ea02df2d155b14`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 観測

### 3.1 main の挙動は第17ラウンドの意図どおり

`main` のビルドで実測した。

| 呼出し | rc | `error.code` |
| --- | --- | --- |
| `inspect`（`-workspace` 無し） | 2 | `INVALID_ARGUMENT` |
| `knowledge-context`（subject 無し） | 2 | `INVALID_ARGUMENT` |
| `knowledge-qualify`（subject 無し） | 2 | `INVALID_ARGUMENT` |
| `knowledge-review-plan`（subject 無し） | 2 | `INVALID_ARGUMENT` |
| `inspect` 空白のみの subject | 2 | `INVALID_SUBJECT`（据え置き） |
| `inspect` 実在しない subject | 1 | `SUBJECT_NOT_FOUND`（据え置き） |
| `knowledge-qualify` 不正な要件ファイル | 2 | `KNOWLEDGE_BUILD_FAILED`（据え置き） |

### 3.2 併走していた重複 PR

[PR #3755](https://github.com/hat47x/tei-platform/pull/3755) が同じ #3733 を別実装（`requireKnowledgeSubject`）で解消していた。私の #3749 が先に統合され、#3755 は `CONFLICTING` / `DIRTY` になっていた。すでに別の作業者によって閉じられていたため、追加の操作はしていない。

実装の違い: #3755 は subject 2つの欠落だけをまとめる。#3749 は `argumentError` 型に統一し、`-workspace`・`-qualification-requirements` の欠落も同じ規則で扱う。

### 3.3 文書が古い符号を主張していた

`tools/reference-mcp-readonly-adapter/README.md` の検証一覧に、次の記述が残っていた。

> 必須引数の欠落が、CLI では `INVALID_SUBJECT`、MCP では `-32602` で**両方拒否**されること

コードと試験は `INVALID_ARGUMENT` に変わっている（`inspect_trace_parity_test.go:243`）。第17ラウンドで私は試験だけを更新し、この散文を見落とした。

**文書の主張とコードが食い違う状態**であり、読者は「CLI は `INVALID_SUBJECT` を返す」と誤解する。実際には返さない。

## 4. 修正

文書を実装に合わせ、**散文が再びずれないよう試験で固定**した。

- `tools/reference-mcp-readonly-adapter/README.md`: 「CLI は `INVALID_ARGUMENT`（#3733 で統一）、MCP は `-32602` で、層が違うだけ」と書き直し、`INVALID_SUBJECT` は「値は渡されたが使えない」に残ることを `cli_failure_classification_test.go` が固定すると明記。
- `tools/reference-mcp-readonly-adapter/readme_claim_test.go`（新規）: README の該当箇所が `INVALID_ARGUMENT` を名指しし、`INVALID_SUBJECT` を主張していないこと、および `inspect_trace_parity_test.go` が `INVALID_ARGUMENT` を検査していることを確かめる。

### 4.1 試験が効くことの確認

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | README を旧記述（`INVALID_SUBJECT`）へ戻す | FAIL |
| M2 | README から主張そのものを消す | FAIL |

戻すと PASS。**文書が実装からずれたら落ちる。**

## 5. 起票しない理由

第17ラウンドで起票した #3733 の解消に含まれる不備（同じ変更で直すべき文書の見落とし）であり、新しい欠陥ではない。別Issueを立てず、#3733 の修正の一部として扱う。

## 6. 読み取れないこと（NOT RUN）

- CLI の `INVALID_ARGUMENT` と MCP の `-32602` の層の統一（#3420 の残り）。
- 他の文書（`docs/`・`plan/design/`）に同じ種類の古い記述が残っていないことの全面確認。今回は `INVALID_SUBJECT` を検索して見つかった1件だけを直した。
- HTTP・Desktop、実DB・実Provider、意味の受理・権限・配備。

## 7. 位置付け

「実装を直したら、同じ規則を説明している文書も同じ変更で直す」という原則の抜けを、実際に見つけて塞いだ記録である。散文の主張を実行可能な試験で固定したので、次に符号を変える者は README も直さないと CI が落ちる。
