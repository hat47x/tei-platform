# TEI自身の設計をCLIで扱う実測（2026-11-07、ドッグフーディング）

- 対象課題: [#3149 TEIドッグフーディング](https://github.com/hat47x/tei-platform/issues/3149)。対応方針とClaim境界は同Issueを正本とする。
- 目的: 生成AI（この実測ではCLIを操作するエージェント）が、TEIの読み取り専用CLIだけでTEI自身の設計記述を発見・読取り・検証し、元の定義へ戻れるかを、同一の実行物で確かめる。
- 実行日: 2026-11-07（Asia/Tokyo）
- 状態: **read-onlyの発見・読取り・検証を実測**。意味の受理、権限付与、本番配備は行っていない（`AGENTS.md`のA0〜A2の範囲）。
- 課題の整理: [TEIドッグフーディング計画](../../workstreams/TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11.md)。

## 1. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `hat47x/tei-platform` commit `0b9c14a836eac0746339c7133f494452cfe56b33`（実測開始時点の`origin/main`の祖先。実測中に`main`は進んだが、判定に使う版はこのcommitに固定した） |
| 実行物 | 同commitのworktreeで`go build -o /home/hat47x/dogfood/bin/tei ./cmd/tei`により作った**単一の実行ファイル**。SHA-256 `bf6930a047cf94f276fb63b082bf71599986d71f1a5c53804105c04aa0021cfd`。以降の全呼出しはこの実行物を使った |
| 実行物の自己申告 | `version -json` → `vcsRevision: 0b9c14a836eac0746339c7133f494452cfe56b33`、`goVersion: go1.27.3`、`goos: linux`、`goarch: amd64`、`source: build-info` |
| 環境 | WSL2 Ubuntu 22.04.5（Windows 11上）、Go `go1.27.3 linux/amd64` |
| checkout | 隔離worktree（`git worktree add --detach <path> origin/main`）。作業ブランチ`docs/dogfooding-cli-self-design-20261107`はこのcommitから切った |
| 対象Workspace | 同commitの`reference/workspace`、`examples/first-workspace`、`examples/xtea-lowcode`、および再実行時に一時作成したWorkspace |

`go run`は使っていない。呼出しごとにソースが読み直される経路を避け、実行物の同一性を固定した。

## 2. 生データの一覧

同じディレクトリの`cli-*`が、実行時の標準出力と標準エラーをUTF-8・LFで保存したコピーである。生データのバイト照合値は記録していない。照合は「§1のcommit＋実行物のSHA-256」で行う。再実行の手順は§6にある。

### 2.1 識別・発見

| ファイル | 呼出し | 終了コード |
| --- | --- | --- |
| `cli-version-json.txt` | `version -json` | 0 |
| `cli-commands.json` | `commands` | 0 |
| `cli-commands-with-argument.txt` | `commands extra`（標準エラー） | 2 |
| `cli-unknown-command.txt` | `nonexistent`（標準エラー） | 2 |

### 2.2 一覧と読取り

| ファイル | 呼出し | 終了コード |
| --- | --- | --- |
| `cli-model-catalog-reference.json` | `model catalog -workspace reference/workspace` | 0 |
| `cli-model-catalog-xtea-lowcode.json` | `model catalog -workspace examples/xtea-lowcode` | 0 |
| `cli-inspect-sales-order.json` | `inspect … -subject-kind semantic-data -subject-id sales.order` | 0 |
| `cli-inspect-std-text.json` | `inspect … -subject-kind semantic-type -subject-id std.text`（`examples/first-workspace`） | 0 |
| `cli-inspect-sales-order-xtea-lowcode.json` | `inspect … -subject-kind semantic-data -subject-id sales.order`（`examples/xtea-lowcode`） | 0 |

### 2.3 参照

| ファイル | 呼出し | 終了コード |
| --- | --- | --- |
| `cli-trace-relation-summary.json` | `trace … -subject-kind semantic-relation -subject-id sales.order.customer -direction both -mode summary` | 0 |
| `cli-trace-relation-full.json` | 同上 `-mode full` | 0 |
| `cli-trace-page1.json` | `trace … -subject-kind semantic-data -subject-id sales.order -mode full -limit 2` | 0 |
| `cli-trace-page2.json` | 前頁の`result.snapshotFingerprint`を`-expected-fingerprint`に渡し`-offset 2` | 0 |
| `cli-trace-stale-snapshot.json` | 同じ呼出しに存在しないトークン | 1 |
| `cli-inspect-candidate-key.json` | `inspect … -subject-kind semantic-candidate-key -subject-id sales.order.order_identity`（`examples/xtea-lowcode`） | 0 |
| `cli-trace-candidate-key-not-found.json` | 同じ対象を`trace` | 1 |
| `cli-trace-unknown-kind.json` | `trace … -subject-kind not-a-kind` | 1 |
| `cli-kind-matrix.json` | `model catalog`の`entries`（`sourceStatus=indexed`）を種類ごとに`inspect`と`trace`へ通した突合（2 Workspace、19行） | — |

### 2.4 検証

| ファイル | 呼出し | 終了コード |
| --- | --- | --- |
| `cli-validate-reference.json` | `validate -workspace reference/workspace -json` | 0 |
| `cli-validate-reference-text.txt` | `validate -workspace reference/workspace`（テキスト） | 0 |
| `cli-validate-xtea-lowcode.json` | `validate -workspace examples/xtea-lowcode -json` | 0 |
| `cli-validate-unknown-type.json` | `std.not_defined`を型にしたWorkspaceのコピーを`validate -json` | 1 |

### 2.5 管理ファイル・入力元索引・業務モデル

| ファイル | 呼出し | 終了コード |
| --- | --- | --- |
| `cli-workspace-files-reference.json` | `workspace-files -workspace reference/workspace` | 0 |
| `cli-workspace-files-first-workspace.json` | `workspace-files -workspace examples/first-workspace` | 0 |
| `cli-source-index-first-workspace.json` | `source-index -workspace examples/first-workspace` | 0 |
| `cli-view-application.json` | `view … -application order_entry -format json` | 0 |
| `cli-view-execute.txt` | `view … -pattern order_entry.submit_order -command sales.order.submit -input order=sales.order -execute`（テキスト） | 0 |
| `cli-view-execute.json` | 同上（`-format json`） | 0 |
| `cli-view-execute-operation-log.txt` | 同上に`-operation-log <一時dir>/op.jsonl`（`op.jsonl` 215バイトを書いた） | 0 |
| `cli-view-execute-lifecycle-state.txt` | 同上に`-lifecycle-state <一時dir>/state.db`（`state.db` 12,288バイトを書いた） | 0 |

### 2.6 宣言の導出要件と拡張の固定情報

| ファイル | 呼出し | 終了コード |
| --- | --- | --- |
| `cli-requirements-reference.json` | `requirements -workspace reference/workspace` | 0 |
| `cli-plugin-lock-envelope.json` | `plugin-lock -workspace reference/workspace`（envelope全体） | 0 |
| `cli-plugin-lock-inner.json` | 上の`result`だけを保存したもの | — |
| `cli-plugin-runtime-plan-envelope-error.json` | `plugin-runtime-plan … -plugin-lock <envelope>` | 1 |
| `cli-plugin-runtime-plan-inner-lock.json` | 同上 `-plugin-lock <inner>` | 0 |
| `cli-plugin-resolve-envelope-error.json` | `plugin-resolve … -plugin-lock <envelope>` | 2 |
| `cli-plugin-resolve-inner-lock.json` | 同上 `-plugin-lock <inner>` | 0 |

### 2.7 設計文書（Markdown）をWorkspaceにした場合

| ファイル | 呼出し | 終了コード |
| --- | --- | --- |
| `cli-validate-markdown-workspace.json` | `validate -workspace <docs+plan>` | 0 |
| `cli-workspace-files-markdown-workspace.json` | `workspace-files -workspace <docs+plan>` | 0 |
| `cli-world-markdown-workspace.json` | `world -workspace <docs+plan>` | 0 |

### 2.8 失敗系

| ファイル | 呼出し | 終了コード |
| --- | --- | --- |
| `cli-failure-inspect-not-found.json` | 不在のsubject | 1 |
| `cli-failure-namespace-not-found.json` | 存在しないnamespace | 1 |
| `cli-failure-inspect-unsupported-kind.json` | 未知の`-subject-kind` | 2 |
| `cli-failure-inspect-unknown-flag.json` | 未知のflag | 2 |
| `cli-failure-validate-missing-workspace.json` | 存在しないWorkspace | 2 |
| `cli-failure-validate-world-and-workspace.json` | `-workspace`と`-world`の同時指定 | 2 |

### 2.9 引数のヘルプと要件JSONの形

| ファイル | 呼出し | 終了コード |
| --- | --- | --- |
| `cli-help-inspect.json` | `inspect -h` | 2 |
| `cli-help-view.json` | `view -h` | 2 |
| `cli-help-knowledge-qualify.json` | `knowledge-qualify -h` | 2 |
| `cli-knowledge-qualify-empty-requirements.json` | 要件JSONに`{}` | 2 |
| `cli-knowledge-qualify-unknown-field.json` | 要件JSONに`{"requirements":[…]}` | 2 |

## 3. 観測の要点

### 3.1 成立したこと

- `commands`は19件を返し、18件が`schemaVersion`と`jsonContract`を持ち、`world`だけが`schemaVersion`を持たず`jsonContract: raw`である。
- `model catalog`は`reference/workspace`で`entries`72件（標準定義の型を含む）と`declarationCounts.total=71`を返し、`worldFingerprint`・`sourceIndexFingerprint`・`modelRevision`・`declarationCounts`を持つ。`entries`の件数と`declarationCounts.total`は一致しない。
- `inspect`は`getting-started.md`が示す12種類を受け付ける。`reference/workspace`では実在IDの10種類が成功し、`semantic-candidate-key`と`semantic-functional-dependency`は宣言が無いため`SUBJECT_NOT_FOUND`（終了コード1）を返した。`examples/xtea-lowcode`では`semantic-candidate-key`が実在し、`inspect`は成功した。
- `trace`は宣言済みの直接参照を返す。`summary`は`total`と`unresolved`だけを持ち、`full`は`rows`を持つ。ページ継続は`result.trace.nextOffset`と外側`result.snapshotFingerprint`で成立し、存在しないトークンは`STALE_SNAPSHOT`（終了コード1）で拒否された。`getting-started.md`の記述と一致する。
- `validate -json`は、意味エラーで元ファイルの範囲を返し、`ok:false`でも`worldFingerprint`・`sourceIndexFingerprint`・`declarationCounts`を持つ。
- `validate`と`model catalog`の同一Workspaceに対する`worldFingerprint`は一致した（`sha256:c92de09f9595b1f46…`）。`inspect`・`trace`も同じ値だった。
- 失敗系の符号と終了コードは、`inspect`の不在（1）、未知namespace（1）、未知kind（2）、未知flag（2）、読込み失敗（2）、排他的引数（2）で、文書の説明と一致した。

### 3.2 成立しなかったこと、または差があったこと

詳細は[計画文書](../../workstreams/TEI_DOGFOODING_CLI_SELF_DESIGN_2026-11.md)の§3に、再現手順・観測・期待との差・影響の形で並べた。ここでは要点だけを示す。

- `trace`は`semantic-candidate-key`を扱えない。`model catalog`が`indexed`として返した同じIDを`inspect`は読めたが、`trace`は`SUBJECT_NOT_FOUND`（終了コード1）を返した。存在しない種類`not-a-kind`の応答と**バイト単位で同一**だった。`inspect`は未知kindに`UNSUPPORTED_SUBJECT_KIND`（終了コード2）を返す。
- Workspaceの名前（`tei.workspace.yaml`の`workspace:`）とmanifestの照合値は、どのCLI応答にも出ない（`workspace-files`の`result`は`managedFiles`と`sourceRootFindings`、`source-index`は`artifacts`と`references`のみ）。同じID`sales.order`が2つのWorkspaceで別の意味を持ち、`worldFingerprint`も違う（`sha256:c92de09f…`と`sha256:8930145c…`）が、応答からどのWorkspaceかは示せない（`cli-inspect-sales-order.json`と`cli-inspect-sales-order-xtea-lowcode.json`）。
- 「版」を表す値の名前と置き場所がコマンドごとに違う。`validate`は応答の最上位、`inspect`・`trace`・`model catalog`は`result`の下、`view`は`semanticFingerprint`（使用する名前空間だけの意味、`sha256:f6c4cf29…`）だけを持つ。`view`の値は同じWorkspaceの`worldFingerprint`と異なる。
- `docs/`と`plan/`を`sources`にしたWorkspaceは`validate`が`ok:true`・終了コード0で、`WORKSPACE_DECLARES_NO_MODEL_ELEMENTS`と`sourceRootFindings`の2件（`docs`・`plan`）をwarningとして返した。`workspace-files`はすべて`readByLoader:false`、`world`は標準定義だけを返す。TEIの設計文書（Markdown）はCLIの意味として読まれない。
- `commands`は`view`を`effect: read-only`と広告する。実際に`-execute -operation-log`は`op.jsonl`（215バイト）を、`-execute -lifecycle-state`は`state.db`（12,288バイト）を作業ディレクトリへ書いた。応答は`committed: true`・`persisted: false`・`persistence: "not-applicable"`・`durability: "none"`で、終了コード0だった。
- `plugin-lock`の応答（envelope）をそのまま`-plugin-lock`へ渡すと`unsupported plugin lock format 0`で失敗する。内側の`result`を渡せば成功する。同じ不正lockに対して、`plugin-runtime-plan`は終了コード1、`plugin-resolve`は終了コード2を返した。
- `-h`は3コマンドで`INVALID_ARGUMENT`（`flag: help requested`、終了コード2）になった。`knowledge-qualify`の要件JSONの形はGoの構造体にしかなく、`{}`は`qualification expectedKind is required`、`{"requirements":[…]} `は`json: unknown field "requirements"`で拒否された。
- 出力の形が揃っていない箇所がある。`validate`の`diagnostics[]`は要素が`{"diagnostic":{…}}`の二重構造、`proposals[]`は平坦。`requirements`の`result`は配列で、他のコマンドはオブジェクト。`model catalog`の`entries`は標準定義の型12件を混ぜるため、`declarationCounts.types=0`のWorkspaceでも`semantic-type`のentryがある。

## 4. 読み取れること

- 同一の実行物で、発見→一覧→要素読取り→参照→検証→管理ファイル→入力元索引までを、機械可読な結果として連続して実行できた。
- 版の同一性は、同一Workspace内では`worldFingerprint`と`sourceIndexFingerprint`で照合できる。`validate`・`model catalog`・`inspect`・`trace`は同じ値で一致した。
- 失敗は終了コードと`error.code`で機械可読に返る（`inspect`・`trace`・`validate -json`・`knowledge-*`）。テキストのみの経路は終了コードと文面に依存する。
- TEI自身の設計文書（Markdown）は、CLIにとっては「管理対象のファイル」であって「意味の定義」ではない。この実測では意味として読む経路が無かった。

## 5. 読み取れないこと（NOT RUN）

- 実DB・実Provider・実Host（PostgreSQL、外部プラグイン実行、Wails/Web）での動作。
- `view -execute`の外部副作用（参照実行は`persistence: not-applicable`、`durability: none`）。ローカルファイルの書込みだけを観測した。
- Shared Acceptance、Proposal、受理、権限付与、配備。
- 複数セッション・複数エージェントでの同時利用、Harness間の可搬性。
- 大規模Workspace（数百MB級）での性能・完全性。`trace`の完全性限界（[#1177](https://github.com/hat47x/tei-platform/issues/1177)）は未評価。
- 改行の異なるcheckoutでの照合値の一致（[#2399](https://github.com/hat47x/tei-platform/issues/2399)の既知の境界）。本実測はWSL上のLF checkoutだけで行った。
- GitHub Actionsでの再現。本記録はローカル実行だけであり、CIの結果ではない。

## 6. 再実行手順

```bash
# 1. 固定した版を取得する
git worktree add --detach /tmp/tei-dogfood 0b9c14a836eac0746339c7133f494452cfe56b33
cd /tmp/tei-dogfood

# 2. 実行物を一つ作る
go build -o /tmp/tei-dogfood-bin ./cmd/tei
sha256sum /tmp/tei-dogfood-bin   # bf6930a047cf94f276fb63b082bf71599986d71f1a5c53804105c04aa0021cfd

# 3. 発見と読取り
/tmp/tei-dogfood-bin version -json
/tmp/tei-dogfood-bin commands
/tmp/tei-dogfood-bin model catalog -workspace reference/workspace
/tmp/tei-dogfood-bin inspect -workspace reference/workspace \
  -subject-kind semantic-data -subject-id sales.order
/tmp/tei-dogfood-bin trace -workspace reference/workspace -scope dogfood \
  -subject-kind semantic-relation -subject-id sales.order.customer -direction both -mode full
/tmp/tei-dogfood-bin validate -workspace reference/workspace -json
```

比較用の2点は次のとおり。

```bash
# traceが扱わない種類（examples/xtea-lowcodeは候補キーを宣言している）
/tmp/tei-dogfood-bin model catalog -workspace examples/xtea-lowcode
/tmp/tei-dogfood-bin inspect -workspace examples/xtea-lowcode \
  -subject-kind semantic-candidate-key -subject-id sales.order.order_identity
/tmp/tei-dogfood-bin trace -workspace examples/xtea-lowcode -scope dogfood \
  -subject-kind semantic-candidate-key -subject-id sales.order.order_identity -direction both -mode summary

# read-onlyと広告されるviewのローカル書込み
mkdir -p /tmp/tei-dogfood-out
/tmp/tei-dogfood-bin view -workspace reference/workspace -application order_entry \
  -pattern order_entry.submit_order -command sales.order.submit \
  -input order=sales.order -execute -operation-log /tmp/tei-dogfood-out/op.jsonl
ls -l /tmp/tei-dogfood-out/op.jsonl
```

書き込みを伴う手順は`/tmp`の一時領域だけを対象にし、リポジトリや元のSourceを変更しない。

## 7. 位置付け

この記録は、TEIが掲げる「生成AIと人による設計支援」を、TEI自身の設計記述に対してread-onlyのCLI経路で確かめた**一つの実測**である。ここで得た結果は、CLIの出力契約やWorkspaceの同一性に関する**課題の候補**であり、修正の採否は各Ownerの判断に属する。意味の受理・権限・本番適格性を示すものではない。

関連: [#3149](https://github.com/hat47x/tei-platform/issues/3149)（方針）、[#2605](https://github.com/hat47x/tei-platform/issues/2605)（Product Skill）、[#2831](https://github.com/hat47x/tei-platform/issues/2831)（セルフビルド）、[#2399](https://github.com/hat47x/tei-platform/issues/2399)（改行と照合値）、[#1177](https://github.com/hat47x/tei-platform/issues/1177)（traceの完全性）。
