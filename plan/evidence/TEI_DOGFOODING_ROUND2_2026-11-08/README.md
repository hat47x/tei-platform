# TEI自身の設計をCLIで扱う実測 第2ラウンド（2026-11-08）

- 対象課題: [#3149 TEIドッグフーディング](https://github.com/hat47x/tei-platform/issues/3149)。
- 第1ラウンド: [TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11-07/README.md](../TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11-07/README.md)（課題 D-1〜D-10 の起票元）。
- 実行日: 2026-11-08（Asia/Tokyo）。作業開始時点の `origin/main` は `4fd88c2b1`。
- 状態: **read-only の再実測**。意味の受理、権限付与、本番配備、書込みは行っていない。
- 追補: 実測後に main が進み、D-3 は #3366（`9bc015876`）で解消した。当初の観測は `4fd88c2b1` の時点のもので、現在の main の値は§9に別途記録する。

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

同じディレクトリの `*.txt` が実行時の標準出力のコピーである。`.out` はリポジトリの `.gitignore`（`*.out`）に当たるため `.txt` で保存した。**25件すべての標準エラーは空だった**ので、`.err` のファイルは残していない。

| ファイル | 呼出し | 終了コード |
| --- | --- | --- |
| `cli-binary-sha256.txt` | 実行物のSHA-256 | — |
| `cli-version-json.txt` | `version -json` | 0 |
| `01-commands.txt` | `commands` | 0 |
| `02-commands-json-keys.txt` | `commands`（`01` と同じ内容を discovery の検査に使ったもの） | 0 |
| `03-trace-unsupported-kind.txt` | `trace … -subject-kind semantic-candidate-key` | 2 |
| `04-validate-identity.txt` | `validate -workspace reference/workspace -json` | 0 |
| `05-plugin-lock.txt` | `plugin-lock -workspace reference/workspace` | 0 |
| `06-plugin-runtime-plan-envelope.txt` | 上のenvelopeをそのまま `-plugin-lock` に渡す | 0 |
| `07-help-inspect.txt` | `inspect -h` | 0 |
| `08-knowledge-qualify.txt` | 最小の要件JSON（`08-requirements-min.json`）を渡す | 0 |
| `09-inspect-ref.txt`・`10-inspect-xtea.txt` | 2つのWorkspaceで同じ Canonical ID を `inspect` | 0 |
| `11-trace-capability.txt`・`12-trace-capability-full.txt` | `trace`（summary / full） | 0 |
| `13-validate-xtea.txt` | `validate -workspace examples/xtea-lowcode -json` | 0 |
| `14-workspace-files-reference.txt` | `workspace-files -workspace reference/workspace` | 0 |
| `15-source-index-first.txt` | `source-index -workspace examples/first-workspace` | 0 |
| `16-view-application.txt` | `view … -format json` | 0 |
| `17-knowledge-context.txt` | `knowledge-context … -subject-id sales.order` | 0 |
| `18-inspect-not-found.txt` | 不在のsubject | 1 |
| `19-inspect-unsupported-kind.txt` | 未知の `-subject-kind` | 2 |
| `20-namespace-not-found.txt` | 存在しないnamespace | 1 |
| `21-validate-missing-workspace.txt` | 存在しないWorkspace | 2 |
| `22-mcp-surface-survey.txt` | MCP入口の調査（CLI広告・Goソースのgrep・文書の位置づけ） | — |
| `30-view-json.txt`・`31-model-catalog.txt` | D-3の確認（`view` と `model catalog` の版の欄） | 0 |
| `32-validate-markdown-workspace.txt`・`33-files-markdown-workspace.txt` | 設計文書をWorkspaceにした `validate`・`workspace-files` | 0 |
| `addendum/after-view-json.txt` | 実測後の main（`768d8ef3b`）で `view … -format json` | 0 |
| `addendum/after-validate-markdown-workspace.txt` | 実測後の main で設計文書Workspaceを `validate` | 0 |
| `addendum/after-validate-xtea.txt` | 実測後の main で `validate -workspace examples/xtea-lowcode` | 0 |

## 4. 観測

### 4.1 第1ラウンドの課題の現況

| 課題 | 現行mainでの観測 | 状態 |
| --- | --- | --- |
| D-1 `trace` の未対応kind | `UNSUPPORTED_SUBJECT_KIND`・終了コード2。`SUBJECT_NOT_FOUND`（1）と区別できる（`03`） | **閉じた** |
| D-2 Workspace同一性 | `validate` の最上位と `model catalog`・`inspect`・`trace` の `result.workspace` に `name`・`formatVersion`・`manifest` が入る（`04`・`09`・`10`） | **閉じた** |
| D-3 版の欄の不統一 | 実測時点（`4fd88c2b1`）は `view -format json` の `result` が `application`・`patterns`・`schemaVersion`・`semanticFingerprint`・`semanticUses` のみで、`worldFingerprint` を持たなかった（`30`・`31`）。**その後 #3366（`9bc015876`）で解消**し、現在の main は応答の最上位に `worldFingerprint`・`sourceIndexFingerprint` を返す（§9） | 実測時点は残る／現在は解消 |
| D-3 版の欄の不統一 | `view -format json` の `result` は `application`・`patterns`・`schemaVersion`・`semanticFingerprint`・`semanticUses` のみ。`worldFingerprint` を持たず、`semanticFingerprint` は `sha256:f6c4cf29…` で `model catalog` の `worldFingerprint` `sha256:c92de09f…` と異なる（`30`・`31`） | **残る** |
| D-4 `view` の作用と広告 | `view` は `effect: read-only` のまま、`writeFlags` に `-operation-log`・`-lifecycle-state` を挙げる（`01`） | **閉じた**（`effect` 語彙の統一は未着手） |
| D-5 `plugin-lock` のround-trip | envelopeをそのまま `-plugin-lock` へ渡して成功（`06`、終了コード0） | **閉じた** |
| D-6 不正lockの終了コード | 第1ラウンドで観測した2コマンドの差は、mainの修正で終了コード2に揃った（個別の再実行はしていない） | 閉じた（間接確認） |
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

### 4.4 第1ラウンドの再現性に関する注意

第1ラウンドの生データは `cli-*.json`／`cli-*.txt` として保存していたが、そのうち `.out` を使っていたものは無く、`.txt` で残していた。第2ラウンドでは `.out` が `.gitignore`（`*.out`）に当たることに気づいたため `.txt` へ改名した。**第1ラウンドのファイル名との対応を保つため、以降のラウンドも `.txt` を使う。**

## 5. 読み取れること

- 第1ラウンドで起票した10件のうち、7件（D-1・D-2・D-4〜D-8）は現行mainで解消している。**起票した課題が実際に修正へつながった**ことを、同じ手順の再実測で確認できた。
- 残るのは D-3（版の欄の不統一）・D-9（設計文書がCLIの意味として読まれない）・D-10（出力の形の不統一）の3件で、いずれも**単一の欄や語彙の追加では閉じない設計判断**を含む。
- CLIは引数の許容値を discovery で広告するようになり、生成AIが「何を渡せるか」を文書なしで知る経路ができた。

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

## 9. 実測後の追補（main が進んだ分）

実測に使った `4fd88c2b1` の後で main が進み、D-3 が解消した。実測時点の観測を書き換えず、**現在の main の値**を別の実行物で記録する。

| 要素 | 値 |
| --- | --- |
| Source | `origin/main` の `768d8ef3b`（実測後の main。D-3 を直した #3366 `9bc015876` を含む） |
| 実行物 | 同commitのworktreeで `go build -o /tmp/tei-r2c ./cmd/tei` を作り、`vcsRevision` が `2489dbc3d…`（記録ブランチのHEAD）であることを確認した |
| 追補の生出力 | `addendum/after-view-json.txt`・`addendum/after-validate-markdown-workspace.txt`・`addendum/after-validate-xtea.txt` |

### 9.1 D-3（#3345 / #3366）

- 実測時点（`4fd88c2b1`）: `view -format json` の応答は `result.semanticFingerprint` だけを持ち、`worldFingerprint` も `sourceIndexFingerprint` も持たなかった（`30-view-json.txt`）。
- 現在の main: 応答の**最上位**に `worldFingerprint`（`sha256:c92de09f…`）と `sourceIndexFingerprint`（`sha256:1827f4b4…`）を返し、`result.semanticFingerprint` は `sha256:f6c4cf29…` のまま残る（`addendum/after-view-json.txt`）。3つの値が別物として並ぶため、範囲の違いが名前と置き場所から読める。
- #3345 は 2026-11-07T17:31Z に閉じている。閉じた根拠は #3366 の統合と、mainの実バイナリでの確認である。

### 9.2 D-9・D-10（#3346・#3347、いずれも Open）

- D-9: 現在の main でも `docs/`＋`plan/` を `sources` にした Workspace は `ok:true`・`WORKSPACE_DECLARES_NO_MODEL_ELEMENTS`・`sourceRootFindings`（`docs`・`plan`）のままで、設計記述はCLIの意味として読まれない（`addendum/after-validate-markdown-workspace.txt`）。
- D-10: 現在の main でも `validate` の `diagnostics[]` は要素が `diagnostic` の下に入る二重構造で、`proposals[]` は平坦である（`addendum/after-validate-xtea.txt`）。
- どちらも設計判断を含むため、本記録は観測を渡すだけに留め、採否は各Issueへ委ねる。

### 9.3 MCP

実測後に追加されたMCP実装は無い（`22-mcp-surface-survey.txt` の調査は `4fd88c2b1` 時点のもので、その後の main の変更にはMCPサーバーの追加を含まない）。MCP経路は引き続き `NOT RUN`。
