# 環境を設計し、database・environment を CLI で読めるようにする（2026-11-11、第68ラウンド）

- 対象: [#4128](https://github.com/hat47x/tei-platform/issues/4128)（A 案の一部）。
- 前段: [第65ラウンド](../TEI_DOGFOODING_ROUND65_INTERACTION_FORM_2026-11-11/README.md)。
- 実行日: 2026-11-11（Asia/Tokyo）。
- 状態: **staging 環境を CLI だけで設計し、`database`・`environment` を `inspect` の受理種に追加。受理種 18。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行する。まだ触っていない層として **environment** を選んだ。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `342d58c79`（`origin/main`） |
| CLI実行物（設計時） | `sha256:0884de185988f6e0c17df6267bc88dcaa911ba5472de09ea91cf871e5ea837e7` |
| CLI実行物（修正後） | `sha256:d3fd8a9df6fa59ad1ff75e487b50076c72390d3f48fce86ca2e34321f763d033` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64` |

## 3. 設計課題（着手前に宣言）

> 既存アプリケーション用に `staging` 環境を作る。sales_core と master_data のインスタンスを宣言し、アプリの datasource alias をそれらに束縛し、validate を通す。その後 CLI で読み返す。

## 4. 経過（2往復）

| Attempt | 応答 | 学んだこと |
| --- | --- | --- |
| 1 | `DATABASE_PROVIDER_PLUGIN_MISSING environment staging instance sales_core.staging provider postgres has no resolved database-provider plugin` ＋ `MIGRATION_PROVIDER_PLUGIN_MISSING` | 環境は**使う provider の plugin を自分で解決する必要がある** |
| 2 | `ok`（`bundle tei.reference.providers ^1` を足す） | バンドル宣言で解決した |

**1往復で通った。** 診断が「何が足りないか」を具体的に言った（instance 名・provider 名・migration tool 名まで）。

### 4.1 学んだことの意味

`development` 環境は `bundle tei.reference.providers ^1` を持っている。**`staging` はそれを持っていなかったので、provider が解決できなかった。** 診断は「plugin が無い」と instance 単位で名指ししたので、バンドルを足せばよいと分かった。

## 5. 読み返しで見つけた欠落

設計は通ったが、**`database` も `environment` も CLI で読めなかった**。

| コマンド | database | environment | database-instance |
| --- | --- | --- | --- |
| `model catalog` | 出ない | 出ない | — |
| `model overview` | **`notCoveredKinds` に載っている** | 同左 | — |
| `inspect` | `UNSUPPORTED_SUBJECT_KIND` | 同左 | `UNSUPPORTED_SUBJECT_KIND` |
| `trace` | `UNSUPPORTED_SUBJECT_KIND` | 同左 | — |
| `world` | 返す | 返す | 返す |

**`model overview` は「environment と database は載っていない」と正直に言っていた**（第59ラウンドの `notCoveredKinds`）。しかし `inspect` でも読めないので、**単体で読む経路が無かった**。

## 6. 修正: `database` と `environment` を inspect に追加

```go
case "database":
	for _, item := range world.Databases { ... }
case "environment":
	for _, item := range world.Environments { ... }
```

受理種は 16 → **18**。MCP アダプタの enum ピンも追随。

### 6.1 database-instance は追加しなかった

`database-instance` は **World 直下の宣言ではない**（`environment.Environment` の中にある）。ID も環境ごとに閉じている。**単体 kind として扱うと、どの環境のインスタンスか曖昧になる。** 今回の範囲では追加せず、`environment` を読めば `instances` が見える形にした。

### 6.2 修正後

```text
database     sales_core  -> ok  id=sales_core  keys: contracts, id, kind, ownership, semanticUses
                           sourceStatus: indexed
environment  staging     -> ok  id=staging     keys: bindings, bundles, id, instances
                           sourceStatus: indexed
database     nope_core   -> SUBJECT_NOT_FOUND
environment  nope        -> SUBJECT_NOT_FOUND
```

**両方とも `sourceStatus: indexed`** である（第58ラウンドで足した配備層は `not-indexed` だった）。Source Index がこれらを既に記録していたため。

## 7. 試験

`internal/operations/inspect_delivery_test.go` を拡張:

- `database:sales_core`・`environment:development` を表に追加（射影の一致と `ValidateInspectionProjection` を通ることを確認）
- not-found の対象に `database`・`environment` を追加

### 7.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | `environment` の dispatch を外す | FAIL `environment:development did not resolve` |
| M2 | `database` の dispatch を外す | FAIL `database:sales_core did not resolve` |

**第58ラウンドで足した「広告と実宣言の突き合わせ」試験が、今回も何も書き換えずに効いた。**

## 8. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/operations` | ok |
| `./tools/reference-mcp-readonly-adapter` | ok |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 9. #4128 の進捗

| 案 | 状態 |
| --- | --- |
| C `inspect` の受理種を広げる | 完了（第58・60・68ラウンド。18種） |
| E 範囲を明示する | 完了（第59ラウンド） |
| **A `model catalog` に種を加える** | **未着手**（名前空間フィルタ設計に触れる） |
| B `overview` にセクションを加える | 未着手 |
| D 専用コマンド | 未着手 |

**C は「World 直下の宣言」をほぼ網羅した。** 残るは `plugin`・`bundle`・`persistence-mapping`・`migration-provider` などの配備層だが、これらは `model overview` の `notCoveredKinds` に載っている。

## 10. 読み取れないこと（NOT RUN）

- `database-instance` を kind として扱うべきか（環境スコープのため保留）。
- `plugin`・`bundle` 等を inspect に加えるか。
- #4128 の A・B・D。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 11. 位置付け

**`model overview` の「載っていない」という正直な申告が、実際に読めないことと一致していた回**である。第59ラウンドで E 案（範囲の明示）を入れたので、この欠落は**自分で宣言されていた**。宣言と実害が一致していることを確かめてから直した。

`database-instance` を安易に kind にしなかったのも意図的で、**環境スコープの ID を World 直下の ID と同じ扱いにすると曖昧になる**。
