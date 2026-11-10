# 同じ壊れた plugin-lock が2コマンドで別の終了コードになっていた（2026-11-09、第19ラウンド）

- 対象: [#3765](https://github.com/hat47x/tei-platform/issues/3765)、[#3420](https://github.com/hat47x/tei-platform/issues/3420) CAPI-2b、[#3184](https://github.com/hat47x/tei-platform/issues/3184)（`INVALID_PLUGIN_LOCK` の導入元）、[ADR-0048](../../adr/0048-shared-application-api-and-transport-adapters.md) §2.3「失敗」。
- 実行日: 2026-11-09（Asia/Tokyo）。
- 状態: **read-only の比較実測と、失敗分類の統一**。

## 1. 目的

18ラウンドで一度も触っていなかったコマンド（`plugin-lock`・`plugin-resolve`・`plugin-runtime-plan`・`world`・`knowledge-data-design-candidate-review`）を叩く。特に、`-plugin-lock` という**ファイルを入力に取る**経路は、ロックを作って別コマンドへ渡す往復があり、分類が揃っていない危険が高い。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source（修正前） | `139476049be35f26ff57cf45ed9c94f206e08e8f`（`origin/main`） |
| CLI実行物（修正前） | `sha256:6999026a367fe666b720c03b4877d7a89938c5be58930918d52823edba70c500` |
| CLI実行物（修正後） | `sha256:e09bda5fcf1815885166c8bca5d3f28dc08e91a73c7d8b7ac43313b59325d269` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |
| 対象Workspace | `reference/workspace`（environment `development` は8プラグイン） |

## 3. 生データ

| ファイル | 内容 |
| --- | --- |
| `00-binary.txt` | 修正前の実行物 |
| `01-lock.json` | `plugin-lock` の全体ロック（environment 2件） |
| `10-lock-all.json`・`11-lock-dev.json` | 修正前: 全体／`development` 単体のロック |
| `02-resolve-from-envelope.json` | 往復（全体ロックをそのまま `-plugin-lock` へ） |
| `15-lock-tampered.json`・`20-lock-fp-tampered.json`・`22-lock-dropped.json`・`24-lock-extra.json`・`60-lock-malformed.json`・`61-lock-wrongshape.json`・`62-lock-wrongenv.json` | 7種の壊し方 |
| `50-*`・`51-*` | 修正前: 2コマンドの応答 |
| `30-plan.json`・`32-world.json`・`33-candidate-review.json` | 未踏コマンドの応答 |

## 4. 観測

### 4.1 まず正常系

`plugin-lock` は `formatVersion: 2` で environment ごとに `{id, version, fingerprint}` を並べる。その全体ロックをそのまま `plugin-resolve -plugin-lock` へ渡す往復は成功した。`plugin-runtime-plan` も `pluginLockFingerprint` と `runtimePolicyFingerprint` を返して成功した。

`world` は包まず素の本体を返す（`ok`・`schemaVersion`・`result` を持たない）。`commands` の discovery も `jsonContract: raw` と広告しており、**一致している**。

### 4.2 壊れたロックの分類（修正前）

| ロックの欠陥 | `plugin-resolve` | `plugin-runtime-plan` |
| --- | --- | --- |
| 指紋だけ書換 | exit 2 `PLUGIN_RESOLVE_FAILED` | **exit 1** `PLUGIN_RUNTIME_PLAN_FAILED` |
| version 不在 | exit 2 `PLUGIN_RESOLVE_FAILED` | **exit 1** `PLUGIN_RUNTIME_PLAN_FAILED` |
| プラグイン削除 | exit 2 `PLUGIN_RESOLVE_FAILED` | **exit 1** `PLUGIN_RUNTIME_PLAN_FAILED` |
| 宣言外を追加 | exit 2 `PLUGIN_RESOLVE_FAILED` | **exit 1** `PLUGIN_RUNTIME_PLAN_FAILED` |
| environment 名が不正 | exit 2 `PLUGIN_RESOLVE_FAILED` | **exit 1** `PLUGIN_RUNTIME_PLAN_FAILED` |
| JSON 破損 | exit 2 `INVALID_PLUGIN_LOCK` | exit 2 `INVALID_PLUGIN_LOCK` |
| JSON だが形違い | exit 2 `INVALID_PLUGIN_LOCK` | exit 2 `INVALID_PLUGIN_LOCK` |
| 正当（対照） | exit 0 | exit 0 |

**5種で終了コードが食い違った。** 終了コード1は「意味の所見」、2は「入力の誤り」という区別だが、壊れたロックはどちらも入力の誤りである。

### 4.3 原因

`plugin-runtime-plan` の分岐は `plugin.IsInvalidLock(err)` しか見ておらず、`internal/plugin/lock.go` の `ResolveEnvironmentWithLock` は「指紋不一致」「version 不在」「プラグイン欠落」「別 environment」をすべて素の `fmt.Errorf` で返していた。したがって `IsInvalidLock` に該当せず、`PLUGIN_RUNTIME_PLAN_FAILED`（exit 1）へ落ちていた。

## 5. 修正

`internal/plugin/lock.go` に `LockMismatchError` / `IsLockMismatch` / `lockMismatch` を追加した。

**`InvalidLockError` とは分けた。** 既存の `invalid_lock_test.go` が #3184 として「バイトが読めない」と「整形済みだが一致しない」を区別しており、その区別は正しい（前者は何が違うか言えない、後者は言える）。分けたうえで、両方とも**呼出し元が直せるのはロックファイルだけ**という点が同じなので、CLI では同じ符号に写す。

- `ResolveEnvironmentWithLock` の6箇所（environment 不在・version 不在・指紋不一致・artifact digest 不一致・plugin set 不一致・要件不一致）と `locked resolution` の wrap を `lockMismatch` にした。
- `cmd/tei/cli_result.go` の `plugin-resolve` と `plugin-runtime-plan` の両方で `IsLockMismatch` を見て `INVALID_PLUGIN_LOCK`（exit 2）を返すようにした。

`plugin-resolve` の `<version> incompatible with major`（`internal/plugin/resolve.go`）は、ロックではなく**宣言側の解決**の失敗なので変更していない。

## 6. 修正後の実測

7種すべてが両コマンドで `INVALID_PLUGIN_LOCK`・exit 2 になった。正当なロックは両方 exit 0。

| 欠陥 | `plugin-resolve` | `plugin-runtime-plan` |
| --- | --- | --- |
| 指紋書換・version 不在・削除・追加・environment 不正・JSON 破損・形違い | exit 2 `INVALID_PLUGIN_LOCK` | exit 2 `INVALID_PLUGIN_LOCK` |

## 7. 試験

`cmd/tei/cli_plugin_lock_test.go` の `TestBrokenPluginLockIsClassifiedTheSameByBothCommands`（6ケース）。両コマンドの終了コードと符号が一致し、`INVALID_PLUGIN_LOCK`・exit 2 であることを確かめる。

わざと壊した2種はいずれも検出された。

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | `plugin-runtime-plan` が `IsLockMismatch` を見ない（元の欠陥） | FAIL: `exit codes differ: plugin-resolve=2 plugin-runtime-plan=1` |
| M2 | 指紋不一致を素の `fmt.Errorf` に戻す | FAIL: 同上 |

**既存試験を1件更新した。** `cmd/tei/plugin_lock_scope_cli_test.go` は「別 environment を指すロックは resolve で `PLUGIN_RESOLVE_FAILED`・plan で `PLUGIN_RUNTIME_PLAN_FAILED`（exit 1）」を固定していた。これは今回の変更前の挙動そのものであり、統一後の `INVALID_PLUGIN_LOCK`・exit 2 へ更新した。**挙動を意図的に変えた箇所である。**

`go test ./internal/plugin ./cmd/tei` は ok。`./internal/plugin/...`・`tools/reference-mcp-readonly-adapter`・`tools/repository-plan-check` も ok。gofmt・vet の指摘なし。

## 8. 互換性

`plugin-runtime-plan` の「ロック不整合」は exit 1 `PLUGIN_RUNTIME_PLAN_FAILED` から exit 2 `INVALID_PLUGIN_LOCK` へ変わる。**破壊的である。** `plugin-resolve` 側も `PLUGIN_RESOLVE_FAILED` から `INVALID_PLUGIN_LOCK` へ変わる。`plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md` の両行はどちらも `INVALID_PLUGIN_LOCK` を挙げており、今回の変更で**契約文書に実装が追いついた**。

## 9. 読み取れないこと（NOT RUN）

- CLI の `INVALID_ARGUMENT` と MCP の `-32602` の層の統一（#3420 の残り）。
- `plugin-lock` 自体の失敗（`PLUGIN_LOCK_FAILED`）の分類。今回は入力側の壊れたロックだけを扱った。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 10. 位置付け

「同じ入力の欠陥は、どのコマンドが読んでも同じ終了コードと符号になる」という原則を、`-plugin-lock` の経路で確かめて揃えた。#3184 の「バイトが読めない／一致しない」の区別は残したまま、CLI の外向きの答えだけを1つにした。
