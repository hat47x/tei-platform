# 永続化マッピングを設計し、persistence-mapping を CLI で読めるようにする（2026-11-11、第71ラウンド）

- 対象: [#4128](https://github.com/hat47x/tei-platform/issues/4128)（C 案の拡張、3回目）。
- 前段: [第68ラウンド](../TEI_DOGFOODING_ROUND68_ENVIRONMENT_LAYER_2026-11-11/README.md)。
- 実行日: 2026-11-11（Asia/Tokyo）。
- 状態: **永続化マッピングを CLI だけで設計し、`persistence-mapping` を `inspect` に追加。受理種 19。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行する。まだ触っていない層として **persistence** を選んだ。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `b2e5aaab1`（`origin/main`） |
| CLI実行物（設計時） | `sha256:d6d23d24abeaa95d91cb8b49dd3f60fab3c44c5e6a38ee516727c62fce64a797` |
| CLI実行物（修正後） | `sha256:aa6ce20999f0a839bfcfcc74e341c4a0f2dca9d133f9ca199761e3785fd771c8` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64` |

## 3. 設計課題（着手前に宣言）

> `repair_intake.intake` を `master_data` の表 `public.intakes` に永続化マッピングとして宣言し、3属性を列に結び、CLI で読み返す。

## 4. 経過（6往復）

| Attempt | 応答 | 学んだこと |
| --- | --- | --- |
| 1 | `ok`（**しかし何も起きていない**） | `persistence` という語は受理されないが、**ファイルが読まれていないだけ**だった |
| 2 | `expected "persist"` | ファイルを `persistence/` 配下に置くと読まれる。キーワードは `persist` |
| 3 | `expected "storage"` | `database` の後は `semantic`、列には `storage` と nullability が要る |
| 4 | `PERSISTENCE_UNDECLARED_SEMANTIC_DEPENDENCY` ＋ `UNKNOWN_PERSISTENCE_ATTRIBUTE` ×2 | database が `uses semantic` を宣言する必要。属性は実在するものだけ |
| 5 | `DATABASE_CONTRACT_UNDECLARED_SEMANTIC_DEPENDENCY` ＋ `PERSISTENCE_MAPPING_NON_RELATIONAL_DATABASE` | `kind relational` が要る |
| 6 | `ok` | 完成 |

### 4.1 Attempt 1 の「静かな失敗」が最重要

**Attempt 1 は `ok` を返したが、マッピングは World に入っていなかった。**

```text
mappings: 2   （既存の sales_core.order と order_line だけ）
```

ファイルを `databases/master-data/intake-mapping.tei` に置いたが、**ローダは `databases/<db>/persistence/*.tei` しか読まない**。`validate` は「読まなかったファイル」について何も言わない。

**これは配置の誤りが「成功」として返る例**である。#4169 の「構文の配置を引く手段が無い」問題の、より悪い形（**黙って無視される**）。

## 5. 読み返しで見つけた欠落

設計は通ったが、**`persistence-mapping` は CLI で読めなかった**。

| コマンド | persistence-mapping |
| --- | --- |
| `model catalog` | 出ない |
| `model overview` | **`notCoveredKinds` に載っている** |
| `inspect` / `trace` | `UNSUPPORTED_SUBJECT_KIND` |
| `world` | 返す |

## 6. 修正: `persistence-mapping` を inspect に追加

受理種は 18 → **19**。MCP アダプタの enum ピンも追随。

### 6.1 修正後

```text
persistence-mapping master_data.intake -> ok  id=master_data.intake
  table: public.intakes | columns: 1 | sourceStatus: not-indexed
persistence-mapping sales_core.order   -> ok  table: public.orders | columns: 6
persistence-mapping nope.thing         -> SUBJECT_NOT_FOUND
```

**`sourceStatus: not-indexed`** である（Source Index が永続化マッピングを記録していない）。

## 7. 試験と変異

`inspect_delivery_test.go` に `persistence-mapping:sales_core.order` を追加、not-found の対象にも追加。

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | `persistence-mapping` の dispatch を外す | FAIL `persistence-mapping:sales_core.order did not resolve` |

## 8. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/operations` | ok |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 9. #4128 の進捗

C 案で読めるようになった種は **19**。`model overview` の `notCoveredKinds` に残るのは `plugin`・`bundle`・`migration-provider`・`application-release` などの配備層。

## 10. 読み取れないこと（NOT RUN）

- **Attempt 1 の「静かな失敗」を起票していない。** 再現条件（`databases/<db>/persistence/` 以外に置いた `persist` ファイルが無視される）は確定したが、**正しい修正が「警告を出す」なのか「もっと広い場所を読む」なのかを決めていない**。第52ラウンドの誤りを避けるため、起票せず記録のみ。
- `plugin` 等を inspect に加えるか。
- #4128 の A・B・D。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 11. 位置付け

**「成功として返る失敗」を記録した回**である。Attempt 1 は `ok` だったが、何も起きていなかった。**`validate` が通ることは「意図が反映された」ことを意味しない。** 読み返し（`world`）を必ず行う理由が、ここで実証された。
