# 閉じた語彙の診断が有効値を挙げるようにした（2026-11-10、第53ラウンド）

- 対象: [#4094](https://github.com/hat47x/tei-platform/issues/4094)（起票）、[#4082](https://github.com/hat47x/tei-platform/issues/4082)（前ラウンドの同型修正）。
- 前段: [第52ラウンド](../TEI_DOGFOODING_ROUND52_ID_RULE_AND_CORRECTION_2026-11-10/README.md)。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **同型の欠陥3件を修正し、残り17件を家族として起票。**

## 1. 目的

第52ラウンドで「語彙を診断で列挙する」を `pattern` の種別について行った。同じ型が他の語彙にもあるかを実測した。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `d41d584e0`（`origin/main`） |
| CLI実行物（修正前） | `sha256:e0428fe2d1e095f3af3910cc551c95f87713dd6885223b4fa9207b61484e98be` |
| CLI実行物（修正後） | `sha256:3c3fef4ea0720912f32de47d93fab033b294eac7035952b816922fec2d75a78b` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 観測（修正前）

3つの閉じた語彙に不正値を入れて、診断が有効値を挙げるかを見た。

| 語彙 | 診断（修正前） | 有効値を挙げるか |
| --- | --- | --- |
| storage type | `has invalid storage type "NUMBERISH"` | **いいえ** |
| nullability | `has invalid nullability "maybe"` | **いいえ** |
| access mode | `has invalid access mode "read-write-nothing"` | **いいえ** |

いずれも `file:line:col` は出る。しかし**有効値が分からない**ため、著者はソースか既存の `.tei` を読む必要があった。

## 4. 修正

### 4.1 単一情報源にする

各語彙に値のスライスを置き、`Valid()` がそれを読むようにした。

```go
// internal/database/persistence/model.go
var StorageTypes = []StorageType{
	StorageInt64, StorageVarchar, StorageText, StorageDateTimeTZ, StorageDecimal, StorageJSON,
}
var Nullabilities = []Nullability{Required, Optional}

// internal/database/model.go
var Accesses = []Access{AccessRead, AccessWrite, AccessReadWrite}
```

### 4.2 診断で列挙する

`internal/validation/validation.go` に小さなヘルパを置き、3つの診断が同じ形で列挙する。

```go
func validValues[T ~string](values []T) string { ... }
```

### 4.3 修正後の実測

```text
INVALID_PERSISTENCE_STORAGE_TYPE
  ... has invalid storage type "NUMBERISH";
  valid values are int64, varchar, text, datetime-tz, decimal, json

INVALID_PERSISTENCE_NULLABILITY
  ... has invalid nullability "maybe"; valid values are required, optional

INVALID_DATABASE_ACCESS
  ... has invalid access mode "read-write-nothing";
  valid values are read, write, read-write
```

正常なワークスペースは `ok` のまま。

## 5. 試験

| ファイル | 試験 |
| --- | --- |
| `internal/validation/vocabulary_diagnostic_test.go` | 3つの診断がそれぞれの有効値をすべて名指し、「valid values are」で導入する |
| `internal/database/persistence/vocabulary_test.go` | `StorageTypes`・`Nullabilities` と `Valid()` が一致し、宣言された定数が漏れない。大文字・空白付き・未知値を拒否 |
| `internal/database/access_vocabulary_test.go` | `Accesses` と `Valid()` の一致 |

### 5.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | storage 診断から値一覧を削る | FAIL `storage type: message does not name "int64": ... has invalid storage type "NUMBERISH"` |
| M2 | `StorageTypes` から `text` を落とす | FAIL `TestStorageTypesAndValidAgree`（宣言された定数が漏れている） |

## 6. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/validation` | ok |
| `./internal/database/...`（10パッケージ） | ok |
| `./internal/workspace ./internal/operations` | ok |
| `./cmd/tei` | ok（233.9s） |
| `./tools/reference-access-survey` | ok |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 7. 起票: 残り17件

[#4094](https://github.com/hat47x/tei-platform/issues/4094)。`internal/validation/validation.go` を実測し、同じ形（`has invalid X %q` で終わり有効値を挙げない）の診断が**17件**残っていることを記録した（`INVALID_PLUGIN_RUNTIME`・`INVALID_SEMANTIC_RELATION_CARDINALITY`・`INVALID_DATABASE_OWNERSHIP`・`INVALID_ROLLOUT_STRATEGY` など）。

**1ラウンドでまとめて直さなかった理由**: 17件は語彙の定義場所が `plugin`・`semantic`・`database/release`・`deployment`・`releaseplan` に分散し、それぞれに単一情報源の追加が要る。実測した3件を先に通し、残りを Issue として残す方が安全と判断した。

## 8. 読み取れないこと（NOT RUN）

- 残り17件の修正。
- `INVALID_APPLICATION_RELEASE_DATABASE_ACCESS` は `database.Accesses` を再利用できるため他より容易（Issue に記載）。
- `capability`・`data`・`interaction` の語彙が CLI から引けるか（第52ラウンドで `std` について誤っていたため、同様の誤りの可能性がある）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

**同じ型の欠陥を、実測してから直した回**である。第52ラウンドで `pattern` の種別を直したとき「他にもあるか」を確かめずに終えていた。今回は3つ測って3つ直し、残りを数えて起票した。

**部分修正であることは明示した。** 3件を直して17件を残すのは「半分だけ塞いだ」に近い。ただし17件は語彙ごとに定義が分散しており、1ラウンドで安全に通す範囲を超えると判断した。数を明示して Issue にしたので、次に着手する人が全体像を持てる。
