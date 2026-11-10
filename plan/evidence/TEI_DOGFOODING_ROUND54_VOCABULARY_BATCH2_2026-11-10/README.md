# 語彙診断の第2弾と、未検証だった疑問の解消（2026-11-10、第54ラウンド）

- 対象: [#4094](https://github.com/hat47x/tei-platform/issues/4094)（第1弾から継続）。
- 前段: [第53ラウンド](../TEI_DOGFOODING_ROUND53_VOCABULARY_DIAGNOSTICS_2026-11-10/README.md)。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **4件を追加修正し、前ラウンドの未検証の疑問を「欠陥なし」と確認。**

## 1. 目的

2つ。第53ラウンドで残した「`capability`・`data`・`interaction` の語彙も引けないのではないか」という**未検証の疑問**を確かめること、そして #4094 の残りに着手すること。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `0d09be511`（`origin/main`） |
| CLI実行物（修正前） | `sha256:99a53f8caed6567efebd17644e87534f00a6a7455a0b386d45e9ac56b021d05e` |
| CLI実行物（修正後） | `sha256:f1d52d82097095d58d4adab1680c4561072081f5e663ff2f0bc4b38a73e134f2` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 未検証の疑問は「欠陥なし」だった

第52ラウンドで `std` について「引けない」と誤ったので、同じ型の誤りが残っていないかを実測した。

| 対象 | 不正な語を書いたときの診断 |
| --- | --- |
| `capability` ブロック | `expected input, output, outcome, effect, or emptyPolicy` |
| `data` ブロック | `expected attribute, relation, candidate-key, or functional-dependency` |
| `interaction` ブロック | `expected item, navigate, or command in interaction` |

**いずれも受理する語を列挙している。** 欠陥は無かった。`std` の件は「型名という値の集合」の問題で、ブロック内のキーワードとは性質が違っていた。

**第51〜53ラウンドで「同じ問いが残る」と書いたのは過度な一般化だった。** 記録に残す。

## 4. #4094 の第2弾: 4件を修正

第53ラウンドで17件残したうち、`Valid()` が既に存在し値が少ない4件を選んだ。

| コード | 語彙 | 値 |
| --- | --- | --- |
| `INVALID_APPLICATION_RELEASE_DATABASE_ACCESS` | access mode | read, write, read-write（既存の `database.Accesses` を再利用） |
| `INVALID_DATABASE_OWNERSHIP` | ownership | managed-here, external-observed |
| `INVALID_SEMANTIC_TYPE_KIND` | type kind | primitive, alias |
| `INVALID_SEMANTIC_RELATION_CARDINALITY` | cardinality | one, many |

### 4.1 単一情報源

`database.Ownerships`・`semantic.TypeKinds`・`semantic.Cardinalities` を追加し、各 `Valid()` がそれを読む。`database.Accesses` は第53ラウンドで追加済み。

### 4.2 修正後の実測（CLI で確認）

```text
ownership:
  database sales_core has invalid ownership "managed-elsewhere";
  valid values are managed-here, external-observed

application release access:
  application release order_entry@3.7.0 datasource sales has invalid access mode
  "read-write-nothing"; valid values are read, write, read-write

cardinality:
  relation security.probe.b has invalid cardinality "several";
  valid values are one, many
```

### 4.3 `INVALID_SEMANTIC_TYPE_KIND` は CLI から到達できない

`type` の種別は構文（`from` の有無）から導出されるため、**著者が不正値を書けない**。診断文の改善自体は正しいが、CLI の応答としては観測できない。**この差は明記しておく。**

## 5. 試験

| ファイル | 試験 |
| --- | --- |
| `internal/validation/vocabulary_diagnostic_more_test.go` | ownership・cardinality の診断が有効値を名指し、「valid values are」で導入する |
| `internal/database/ownership_vocabulary_test.go` | `Ownerships` と `Valid()` の一致、未知値・大文字の拒否 |
| `internal/semantic/vocabulary_test.go` | `TypeKinds`・`Cardinalities` と `Valid()` の一致、宣言数との一致、未知値の拒否 |

### 5.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | ownership 診断から値一覧を削る | FAIL `database ownership: message does not name "managed-here"` |
| M2 | `Cardinalities` から `many` を落とす | FAIL `Cardinalities has 1 entries but 2 are declared` |

## 6. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/validation` | ok |
| `./internal/database/...`（10パッケージ） | ok |
| `./internal/semantic` | ok |
| `./internal/workspace ./internal/operations` | ok |
| `./cmd/tei` | ok（124.9s） |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 7. 残り

第53ラウンドの17件のうち4件を直したので、**13件**が残る（うち `INVALID_SEMANTIC_TYPE_KIND` は CLI から到達不能）。残りは `plugin`・`migration`・`deployment`・`releaseplan` 側の語彙で、それぞれ単一情報源の新設が要る。

## 8. 読み取れないこと（NOT RUN）

- 残り13件の修正。
- `plugin`・`migration`・`deployment`・`releaseplan` の語彙が CLI から引けるか（今回の手法で個別に測る必要がある）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

**未検証の疑問を先に潰した回**である。第52ラウンドで `std` について誤ったので、同じ型の誤りが残っていないかを最初に確かめた。結果は「欠陥なし」で、私の一般化が過度だったと分かった。

そのうえで #4094 の第2弾として4件を直した。前ラウンドで「17件は1ラウンドでは無理」と書いた判断は妥当で、今回は4件が適量だった。
