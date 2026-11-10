# 修正後の再実測 第4ラウンド（2026-11-09）

- 対象: [TEI自身の設計をCLIで扱う実測（#3149）](../../workstreams/TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11.md)、[#3455](https://github.com/hat47x/tei-platform/issues/3455)、[#3456](https://github.com/hat47x/tei-platform/issues/3456)。
- 前段: [CLI応答のWorkspace同一性と空一覧の実測](../TEI_DOGFOODING_ROUND3_IDENTITY_2026-11-08/README.md)、[第3ラウンド](../TEI_DOGFOODING_ROUND3_2026-11-08/README.md)。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の再実測**。意味の受理・権限付与・配備は行っていない。

## 1. 目的

#3455・#3456 の修正が `main` に統合された後、同じ CLI 操作で解消を確かめる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `hat47x/tei-platform` commit `00ead5a98f64960cc90e3f8072fb9edc0a0c4cd3`（PR #3467 の統合コミット、`origin/main` と同一） |
| 実行物 | この commit の独立worktreeで `go build ./cmd/tei` した単一ファイル |
| SHA-256 | `1a1ce33406297ea71333d4aba7b4c9931160fad54d3f5e1ddddb1acf8fc76dee` |
| `version -json` | `goVersion: go1.27.3`、`goos: linux`、`goarch: amd64`、`vcsRevision: 00ead5a98…`、`source: build-info` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64` |
| 対象Workspace | `reference/workspace`、および設計文書だけを置いた一時Workspace（`docs`・`plan/adr`） |

前段の記録とは実行物が異なる（前段の修正後は未コミットの作業ツリーから作った `8529323b…`）。数値を直接比較せず、「同じ呼出しで何が返るか」だけを見る。

## 3. 生データ

| ファイル | 呼出し |
| --- | --- |
| `00-binary-sha256.txt`・`01-version.txt` | 実行物の識別 |
| `02-validate.txt` | `validate -workspace reference/workspace -json` |
| `03-view.txt` | `view -workspace reference/workspace -application order_entry -format json` |
| `04-si.txt`・`05-si-subject.txt` | `source-index`（index／subject） |
| `06-kc.txt` | `knowledge-context -subject-kind semantic-data -subject-id sales.order` |
| `07-req.txt` | `requirements` |
| `08-trace.txt` | `trace -scope r4 -subject-kind semantic-data -subject-id sales.order` |
| `09-inspect.txt` | `inspect`（同じ subject） |
| `10-files.txt` | `workspace-files` |
| `11-si-empty.txt` | 設計文書Workspaceの `source-index` |
| `12-validate-docs.txt` | 同Workspaceの `validate -json` |

標準出力はすべて標準エラーが空で、終了コードは0だった。

## 4. 観測

### 4.1 Workspace同一性（#3455）

| コマンド | `workspace`欄の位置 |
| --- | --- |
| `validate -json` | 最上位 |
| `view -format json` | 最上位 |
| `knowledge-context` | 最上位 |
| `requirements` | 最上位 |
| `source-index`（index／subject） | `result` |
| `trace`・`inspect`・`model catalog`・`workspace-files` | `result`（変更なし） |

11件の応答すべてが `name`・`formatVersion`・`manifest.path`・`manifest.sha256` を持ち、同一性の無い応答は0件だった。

### 4.2 空の参照一覧（#3456）

設計文書Workspaceの `source-index` は `result.references` を `[]` で返し、`artifacts` は2件（`docs/architecture.md` と `plan/adr/0048-…md`）だった。`workspace` も同じ `result` にある。

### 4.3 変化していないこと

`validate`の診断は前段と同じく `WORKSPACE_DECLARES_NO_MODEL_ELEMENTS`（warning）1件と `sourceRootFindings` 2件で、設計文書は意味の検証対象にならないままである（D-9）。

## 5. 読み取れないこと（NOT RUN）

- 実DB・実Provider・実Host、意味の受理、権限、配備。
- `-world`ファイルだけを入力した経路（実測は `-workspace` のみ）。
- HTTP・MCP・Desktopの各入口（この記録はCLIのみ）。共通アプリケーションAPIの同値性は [#3419](https://github.com/hat47x/tei-platform/issues/3419)・[#3420](https://github.com/hat47x/tei-platform/issues/3420)・[#3421](https://github.com/hat47x/tei-platform/issues/3421) の担当。
- 改行の異なるcheckout（[#2399](https://github.com/hat47x/tei-platform/issues/2399)）、大規模Workspace（[#1177](https://github.com/hat47x/tei-platform/issues/1177)）。

## 6. 位置付け

#3455・#3456 の解消を、統合後の版でCLIから観測した記録である。意味の受理や本番適格性を示さない。
