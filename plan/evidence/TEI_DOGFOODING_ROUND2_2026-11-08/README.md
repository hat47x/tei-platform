# TEI自身の設計をCLIで扱う実測 第2ラウンド（2026-11-08）

- 対象課題: [#3149 TEIドッグフーディング](https://github.com/hat47x/tei-platform/issues/3149)。
- 第1ラウンド: [TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11-07/README.md](../TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11-07/README.md)（課題 D-1〜D-10 の起票元）。
- 実行日: 2026-11-08（Asia/Tokyo）。作業開始時点の `origin/main` は `4fd88c2b1`。
- 状態: **read-only の再実測**。意味の受理、権限付与、本番配備、書込みは行っていない。

## 1. このラウンドの目的

第1ラウンドで起票した D-1〜D-10 のうち、現行 `main` で何が閉じ、何が残っているかを、**同じ実行物を固定して**確かめる。あわせて、依頼の範囲が「CLI/MCP利用主体」に広がったため、**MCP 入口が現行のTEIに実在するか**をCLIとリポジトリの側から調べる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `hat47x/tei-platform` commit `4fd88c2b13fb259182441b4079bd6179e7bb63fc`（`origin/main`） |
| 実行物 | 同commitの隔離worktreeで `go build -o /home/hat47x/dogfood/bin/tei-r2 ./cmd/tei` により作った単一の実行ファイル。SHA-256 `16c9090d0c4fa1b08bb25b412aede508489179cbd1260b673a4c9e2f1f9b6be0` |
| 実行物の自己申告 | `vcsRevision: 4fd88c2b1…`、`goVersion: go1.27.3`、`goos: linux`、`goarch: amd64`、`source: build-info` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64` |
| 対象Workspace | 同commitの `reference/workspace`・`examples/xtea-lowcode`・`examples/first-workspace`、および一時Workspace |

第1ラウンドの実行物（`bf6930a0…`、base `0b9c14a8…`）とは**別の実行物**であり、数値を直接比較しない。各節は「現行mainでその振る舞いが観測できるか」だけを見る。

## 3. 生データ

同じディレクトリの `cli-*` と、`0N-*.out`／`0N-*.err`（`N` は実行順）が生出力である。主な対応:

| ファイル | 呼出し | 終了コード |
| --- | --- | --- |
| `cli-binary-sha256.txt` | 実行物のSHA-256 | — |
| `cli-version-json.txt` | `version -json` | 0 |
| `01-commands.out` | `commands` | 0 |
| `03-trace-unsupported-kind.out` | `trace … -subject-kind semantic-candidate-key` | 2 |
| `04-validate-identity.out` | `validate -workspace reference/workspace -json` | 0 |
| `05-plugin-lock.out` | `plugin-lock -workspace reference/workspace` | 0 |
| `06-plugin-runtime-plan-envelope.out` | 上のenvelopeをそのまま `-plugin-lock` に渡す | 0 |
| `07-help-inspect.out` | `inspect -h` | 0 |
| `08-knowledge-qualify.out` | 最小の要件JSON `{"expectedKind":"semantic-data"}` | 0 |
| `09-inspect-ref.out`・`10-inspect-xtea.out` | 2つのWorkspaceで同じ Canonical ID を `inspect` | 0 |
| `11`・`12` | `trace`（summary / full） | 0 |
| `13`〜`17` | `validate`・`workspace-files`・`source-index`・`view`・`knowledge-context` | 0 |
| `18`〜`21` | 失敗系（不在1・未知kind2・未知namespace1・読込み失敗2） | 1/2/1/2 |
| `22-mcp-surface-survey.txt` | MCP入口の調査（CLI広告・Goソースのgrep・文書の位置づけ） | — |
| `30`・`31` | `view -format json`・`model catalog` | 0 |
| `32`・`33` | 設計文書（`docs/`＋`plan/`）をWorkspaceにした `validate`・`workspace-files` | 0 |

`02-commands-json-keys.out` は `01` と同じ内容を discovery の検査に使ったもの。

## 4. 観測

### 4.1 第1ラウンドの課題の現況

| 課題 | 現行mainでの観測 | 状態 |
| --- | --- | --- |
| D-1 `trace` の未対応kind | `UNSUPPORTED_SUBJECT_KIND`・終了コード2。`SUBJECT_NOT_FOUND`（1）と区別できる（`03`） | **閉じた** |
| D-2 Workspace同一性 | `validate` の最上位と `model catalog`・`inspect`・`trace` の `result.workspace` に `name`・`formatVersion`・`manifest` が入る（`04`・`09`・`10`） | **閉じた** |
| D-3 版の欄の不統一 | `view -format json` の `result` は `application`・`patterns`・`schemaVersion`・`semanticFingerprint`・`semanticUses` のみ。`worldFingerprint` を持たず、`semanticFingerprint` は `sha256:f6c4cf29…` で `model catalog` の `worldFingerprint` `sha256:c92de09f…` と異なる（`30`・`31`） | **残る** |
| D-4 `view` の作用と広告 | `view` は `effect: read-only` のまま、`writeFlags` に `-operation-log`・`-lifecycle-state` を挙げる（`01`） | **閉じた**（`effect` 語彙の統一は未着手） |
| D-5 `plugin-lock` のround-trip | envelopeをそのまま `-plugin-lock` へ渡して成功（`06`、終了コード0） | **閉じた** |
| D-6 不正lockの終了コード | 第1ラウンドで観測した2コマンドの差は、mainの修正で終了コード2に揃った（`22`の調査対象外。個別の再実行はしていない） | 閉じた（間接確認） |
| D-7 `-h` | `inspect -h` は終了コード0・標準出力にusage（`07`） | **閉じた** |
| D-8 要件JSONの例 | 読み案内に欄の表と実例ファイルへの参照があり、最小形 `{"expectedKind":"semantic-data"}` が `applicable` で通る（`08`） | **閉じた** |
| D-9 設計文書が意味として読まれない | `docs/`＋`plan/` のWorkspaceは `ok:true`、`WORKSPACE_DECLARES_NO_MODEL_ELEMENTS` と `sourceRootFindings`（`docs`・`plan`）、管理2ファイルは `readByLoader` が0（`32`・`33`） | **残る** |
| D-10 出力の形の不統一 | `validate` の `diagnostics[]` は要素が `diagnostic` の下に入る二重構造のまま。`proposals[]` は平坦（`13`） | **残る** |

`commands` は19件で、`effects` は `read-only` のみ。`writeFlags` を持つのは `view` だけだった。

### 4.2 MCP入口

- `commands` が広告する19件に、`mcp` を含む名前は無い（`22`）。
- リポジトリのGoソースで `mcp` を含むファイルは、`tools/reference-cli-trace-benchmark/main.go`・`tools/reference-generator-selection/*`・`tools/reference-target-profile/process_model_test.go` のみで、**MCPサーバーやアダプタの実装は無い**（`22`のgrepは0件）。
- `docs/architecture.md` は「TEI独自の開発用MCPサーバーは必須にしない」「将来追加する場合も別の正本解釈・保存先・検証・権限モデルを作らない」と位置づけている。MCPは**現時点で実在しない入口**である。

したがって今回の実測で確かめられたのは **CLI経路だけ**である。MCP経路の検証は、MCP入口が実装されるまで `NOT RUN` として扱う。

### 4.3 変わった点（第1ラウンドとの差）

- `commands` の出力が 6,957 バイトから 9,513 バイトへ増えた。`argumentValues`・`argumentType`・`argumentDefaults`・`writeFlags` が加わったためで、discovery から引数の許容値と出力flagが読める。
- `view -execute` のテキスト出力に `authorization: reference-allow-all` の行が入った（第1ラウンドの固定実行物には無かった）。
- 第1ラウンドで `plugin-lock` の envelope を渡すと `unsupported plugin lock format 0` で失敗した経路が、現行mainでは成功する。

## 5. 読み取れること

- 第1ラウンドで起票した10件のうち、7件（D-1・D-2・D-4〜D-8）は現行mainで解消している。**起票した課題が実際に修正へつながった**ことを、同じ手順の再実測で確認できた。
- 残るのは D-3（版の欄の不統一）・D-9（設計文書がCLIの意味として読まれない）・D-10（出力の形の不統一）の3件で、いずれも**単一の欄や語彙の追加では閉じない設計判断**を含む。
- CLIは引数の許容値を discovery で広告するようになり、生成AIが「何を渡せるか」を文書なしで知る経路ができた。ただし `subject-kind` の許容値のように、`argumentValues` に載っているかを今回は確認していない（`22`の範囲外）。

## 6. 読み取れないこと（NOT RUN）

- MCP入口での動作（入口自体が無い）。
- `go test ./...` 全体、CIの再現。
- 実DB・実Provider・実Host、意味の受理・権限・配備。
- 改行の異なるcheckoutでの照合値（[#2399](https://github.com/hat47x/tei-platform/issues/2399)）、大規模Workspaceでの完全性（[#1177](https://github.com/hat47x/tei-platform/issues/1177)）。
- D-6 の終了コードは個別に再実行していない。第1ラウンドの観測と main の修正記録からの推定である。

## 7. 再実行手順

```bash
git worktree add --detach /tmp/tei-r2 4fd88c2b13fb259182441b4079bd6179e7bb63fc
cd /tmp/tei-r2
go build -o /tmp/tei-r2-bin ./cmd/tei
sha256sum /tmp/tei-r2-bin   # 16c9090d0c4fa1b08bb25b412aede508489179cbd1260b673a4c9e2f1f9b6be0

/tmp/tei-r2-bin version -json
/tmp/tei-r2-bin commands
/tmp/tei-r2-bin validate -workspace reference/workspace -json
/tmp/tei-r2-bin inspect -workspace reference/workspace -subject-kind semantic-data -subject-id sales.order
/tmp/tei-r2-bin view -workspace reference/workspace -application order_entry -format json
```

## 8. 位置付け

第1ラウンドと同じく、この記録は read-only の実測であり、意味の受理・権限・本番適格性を示さない。課題の現況は「どの振る舞いが観測できたか」の記録であり、修正の採否は各Ownerの判断に属する。
