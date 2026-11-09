# CLI応答のWorkspace同一性と空一覧の実測（2026-11-08、WSL2）

- 対象課題: [#3149](https://github.com/hat47x/tei-platform/issues/3149)、[#3455](https://github.com/hat47x/tei-platform/issues/3455)、[#3456](https://github.com/hat47x/tei-platform/issues/3456)。
- 関連記録: [第1ラウンド](../TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11-07/README.md)、[第2ラウンド](../TEI_DOGFOODING_ROUND2_2026-11-08/README.md)、[MCP経路](../TEI_DOGFOODING_MCP_ROUND_2026-11-08/README.md)。同名の「第3ラウンド」記録が Windows 環境であり、こちらは応答の同一性に絞った別の実測である。
- 実行日: 2026-11-08（Asia/Tokyo）。
- 状態: **read-only の実測**。書込み・受理・権限付与・配備は行っていない。

## 1. 目的

現行 `main`（`3b4f5519d`）の CLI で、次の2点を確かめた。

1. 第2ラウンドで残るとした D-9（設計文書が CLI の意味として読まれない）は、`docs/` と `plan/` を `sources` に置いた場合に何が観測できるか。
2. 読み取りコマンド横断で、Workspace 同一性（#3455）と空の一覧の直列化（#3456）がどうなっているか。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| 実測前（before） | `main` `3b4f5519dd99a98cf6d7980cbd698083001d5458`。実行物 `go build ./cmd/tei`、SHA-256 `28532d6e97bd617b6871290f7814af3c7f58c9565c8a835a49a7bc1cdbb8e9e3` |
| 実測後（after） | 同じ `3b4f5519d` からの作業ツリーで、#3455・#3456 の修正を加えた実行物。SHA-256 `8529323b7e17d7d9c10556a68dd64b1c65f70ec7fc675e01ec1330ceaac83c92`、`vcsModified: true` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（CIと同じ `go1.26.10` の gofmt も併用） |
| 対象 | `reference/workspace`、`examples/first-workspace`、および設計文書だけを置いた一時Workspace |

`after` の実行物は未コミットの作業ツリーから作っているため、`before` と数値を直接比べるのではなく、「同じ呼出しで何が変わったか」だけを見る。

## 3. 生データ

`before-*.txt` が実測前、`after-*.txt` が実測後である。

| ファイル | 呼出し |
| --- | --- |
| `before-01-commands.txt` | `commands` |
| `before-02-validate-ref.txt` | `validate -workspace reference/workspace -json` |
| `before-03-view-ref.txt` | `view -workspace reference/workspace -application order_entry -format json` |
| `before-04-trace-ref.txt`・`before-05-inspect-ref.txt` | `trace`・`inspect`（同じ subject） |
| `before-06-validate-design-docs.txt`・`before-07-files-design-docs.txt`・`before-08-source-index-design-docs.txt` | 設計文書Workspace（`sources: [plan, docs]`）の `validate`・`workspace-files`・`source-index` |
| `before-10-si-subject.txt`・`before-11-kc.txt`・`before-12-req.txt` | `source-index`（subject）・`knowledge-context`・`requirements` |
| `before-14-model.txt`・`before-15-inspect.txt`・`before-16-files-empty.txt`・`before-17-validate-empty.txt` | 空に近い Workspace の `model catalog`・`inspect`・`workspace-files`・`validate` |
| `after-cli-binary-sha256.txt`・`after-cli-version-json.txt` | 実行物の識別 |
| `after-01-validate.txt`〜`after-08-inspect.txt` | 同じ8コマンドの実測後 |
| `after-09-si-empty.txt`・`after-10-validate-design.txt` | 設計文書Workspaceの `source-index`・`validate`（実測後） |

## 4. 観測（実測前）

### 4.1 設計文書をWorkspaceにした場合（D-9の現況）

- `validate`は`ok:true`で、`diagnostics`に`WORKSPACE_DECLARES_NO_MODEL_ELEMENTS`（warning）1件、`sourceRootFindings`に`plan`と`docs`の2件（「Loaderが解釈しないroot」）を返した（`before-06`）。
- `workspace-files`は管理2ファイルを`readByLoader:false`で返した（`before-07`）。
- `source-index`は2ファイルを`format:md`で索引に載せ、`result.references`は**`null`**だった（`before-08`）。

つまり D-9 は「CLI が設計文書を管理ファイルとして索引には載せるが、意味としては読まない」という形で残っている。`source-index` が `.md` を `artifacts` に載せるため、位置の追跡はできる。一方で意味の定義には入らない。

### 4.2 Workspace同一性の有無（#3455）

| コマンド | `workspace`欄（実測前） |
| --- | --- |
| `validate -json` | 最上位 |
| `view -format json` | **なし** |
| `source-index`（index／subject） | **なし** |
| `knowledge-context` | **なし** |
| `requirements` | **なし** |
| `trace`・`inspect`・`model catalog`・`workspace-files` | あり |

### 4.3 空の参照一覧（#3456）

`source-index`の`mode=index`は、参照0件のWorkspaceで`result.references`を`null`で返した（`before-08`）。同じSource Indexの参照を`inspect`は配列で返す。

## 5. 観測（実測後）

- `validate`・`view`・`knowledge-context`・`requirements`は**最上位**に、`source-index`（index／subject）は`result`に、`workspace`（`name`・`formatVersion`・`manifest.path`・`manifest.sha256`）を持つようになった（`after-01`〜`after-08`）。
- `trace`・`inspect`の既存の`result.workspace`は変わらない。
- `source-index`の空の参照一覧は`[]`になり、`workspace`も`result`に付いた（`after-09`）。
- `validate`・`workspace-files`の既存の欄と終了コードは変わらない（`after-01`・`after-10`）。

## 6. 読み取れること

- 読み取りを行うコマンドの応答が、同じ名前と形で読み元 Workspace を示すようになった。複数の応答を結合する呼出し元は、呼出しパスを別に記録しなくても版を照合できる。
- 空の一覧が`[]`になったことで、「0件」と「欄が無い」を読み手が区別する必要がなくなった。
- D-9 の「設計文書が意味として読まれない」は変わっていない。`source-index` には載るが、`validate` の `diagnostics` は0件で、意味の検証対象にはならない。

## 7. 読み取れないこと（NOT RUN）

- 第4ラウンド（修正後の main での再実測）。
- `go test ./...` 全体、CI、実DB・実Provider・実Host、意味の受理・権限・配備。
- `-world`入力だけを使うコマンドの`workspace`欄（実測では`-workspace`のみを使った）。
- 改行の異なるcheckout（[#2399](https://github.com/hat47x/tei-platform/issues/2399)）、大規模Workspace（[#1177](https://github.com/hat47x/tei-platform/issues/1177)）。

## 8. 再実行手順

```bash
git worktree add --detach /tmp/tei-r3 3b4f5519dd99a98cf6d7980cbd698083001d5458
cd /tmp/tei-r3
go build -o /tmp/tei-r3 ./cmd/tei
sha256sum /tmp/tei-r3   # 28532d6e97bd617b6871290f7814af3c7f58c9565c8a835a49a7bc1cdbb8e9e3

/tmp/tei-r3 validate -workspace reference/workspace -json
/tmp/tei-r3 source-index -workspace reference/workspace
```

## 9. 位置付け

この記録は read-only の実測であり、意味の受理・権限・本番適格性を示さない。§5 の「実測後」は未コミットの作業ツリーから作った実行物による観測であり、統合後の main での確認は次のラウンドで行う。
