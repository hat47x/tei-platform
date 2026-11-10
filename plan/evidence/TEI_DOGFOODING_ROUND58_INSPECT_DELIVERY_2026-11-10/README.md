# #4128 の C 案: `inspect` で migration・database release・solution release を読めるようにする（2026-11-10、第58ラウンド）

- 対象: [#4128](https://github.com/hat47x/tei-platform/issues/4128)（C 案を実装）。
- 前段: [第57ラウンド](../TEI_DOGFOODING_ROUND57_DESIGN_REEXERCISE_2026-11-10/README.md)で起票した。
- 実行日: 2026-11-10（Asia/Tokyo）。
- 状態: **3種を `inspect` の受理種に追加。#4128 の C 案は解消。**

## 1. 目的

第57ラウンドで、設計した migration・database release・solution release を `model catalog`・`model overview`・`inspect`・`trace` のいずれでも読めないことを検出し、#4128 に5案を書いた。うち**C（`inspect` の受理種を広げる）**を実装する。

`inspect` は単一 subject の詳細を返すだけなので、`catalog`（名前空間フィルタ）や `trace`（グラフに subject が無い）と違い、既存設計を変えずに済む。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `29f694500`（`origin/main`） |
| CLI実行物（修正前） | `sha256:7c0a51bcb183f7690d2070aa7241a6fad2b3f279ddacb54d0e6b20d05dab12c1` |
| CLI実行物（修正後） | `sha256:5b76ebcbb5abab16acb2c70020f598931f95c28197b2c5b90c68351a8a1c954f` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.27.3 linux/amd64`（gofmt は 1.26.10） |

## 3. 実装

`InspectionSubjectKinds` に3種を追加し、`inspectCanonical` に3つの case を足した。

```go
"migration", "database-release", "solution-release",
```

### 3.1 subject ID の決め方（設計判断）

最初は `<owner>@<version>`（例 `sales_core@4.2.0`）にしたが、**`INSPECTION_PROVIDER_INVALID: inconsistent projection` で拒否された**。

理由は `ValidateInspectionProjection` の安全不変条件で、**canonical オブジェクトの最上位 `id` が要求 ID と一致**していることを要求する。release の構造体の `id` は owner（`sales_core`）なので、`sales_core@4.2.0` とは一致しない。

**この不変条件は緩めない。** 代わりに subject を **owner** とし、射影を「その owner の全バージョン」にした。

```go
type releaseSet struct {
	ID       string               `json:"id"`
	Releases []release.Definition `json:"releases"`
}
```

- `database-release` の ID は database。複数バージョンは `releases` に並ぶ。
- `solution-release` の ID は plan。同上。
- **バージョンを ID に混ぜないので、`@` の規約を発明せずに済む。**
- `id` が要求 ID と一致するので安全不変条件を満たす。

## 4. 実測（CLI）

```text
inspect -subject-kind migration -subject-id 20260826_001
  ok | canonical.id = 20260826_001 | keys: database, id, recovery, steps

inspect -subject-kind database-release -subject-id sales_core
  ok | canonical.id = sales_core | versions: 4.0.0, 4.1.0, 4.2.0

inspect -subject-kind solution-release -subject-id order_rollout
  ok | canonical.id = order_rollout | versions: 2026.8.27

inspect -subject-kind database-release -subject-id nope_core
  SUBJECT_NOT_FOUND canonical subject database-release:nope_core not found
```

`commands` と `-h` は単一情報源 `InspectionSubjectKinds` を読むため、15種に追随している（個別の更新は不要）。

```text
$ tei inspect -h
  values: ... application, application-pattern, query, migration,
          database-release, solution-release
```

既存種の回帰なし: `semantic-data:sales.order` は `sourceStatus: indexed`、artifacts 1件のまま。

### 4.1 正直に書く限界

新3種の `sourceStatus` は **`not-indexed`** になる。Source Index が migration・release の参照を記録していないため、出所は返せない。**canonical と fingerprint は返るが、source location は返らない。** これはフィールドが正直にそう答えており、#4128 の範囲（読めるようにする）は満たすが、`trace` 相当の来歴は別途の課題である。

## 5. 試験

`internal/operations/inspect_delivery_test.go`:

| 試験 | 内容 |
| --- | --- |
| `TestDeliveryLayerSubjectsAreInspectable` | 3種が解決し、**`ValidateInspectionProjection` を通り**、canonical の `id` が要求 ID と一致し、release はバージョンを列挙する |
| `TestUnreleasedOwnerIsNotFound` | release の無い owner は `SUBJECT_NOT_FOUND`（空集合を成功として返さない） |
| `TestEveryAdvertisedKindWithADeclarationResolves` | **広告した種のうち、reference に宣言があるものは必ず解決する。** 宣言の無い種は判定しない |
| `TestEveryAdvertisedInspectionKindIsAccepted` | 広告と受理述語の一致 |

### 5.1 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | `application-release` を広告に足すが case は無い | FAIL `advertised kind "application-release" has a declaration "order_entry" but does not resolve` |
| M2 | release 集合の `id` をバージョンにする | FAIL `failed the projection check: ... another subject` / `canonical id is "4.0.0", want "sales_core"` |
| M3 | 空集合でも返す（not-found をやめる） | FAIL `want an InspectionError, got <nil>` |
| M4 | dispatch の case 名を変える | FAIL `advertised kind "solution-release" ... does not resolve` |

**M1・M3 は最初の版では捕捉できなかった。** 第1版の試験は広告リストと受理述語が同じリストを読むだけ、not-found は `database-release` だけを見ていた。**試験を強化して初めて捕まるようになった**（広告と宣言の突き合わせ、2種の not-found）。

## 6. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/operations` | ok |
| `./tools/reference-mcp-readonly-adapter`（enum ピン 12→15） | ok |
| `./cmd/tei` | ok（121.5s） |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 7. #4128 の残り

| 案 | 状態 |
| --- | --- |
| **C** `inspect` の受理種を広げる | **完了（本ラウンド）** |
| E `overview` の対象範囲を明示する | 未着手 |
| A `model catalog` に加える | 未着手（catalog の名前空間フィルタ設計に触れる） |
| B `overview` のセクションに加える | 未着手 |
| D 専用コマンド | 未着手 |

`trace` は別問題（グラフに subject が無い）として残る。新3種の source 来歴が `not-indexed` なのも同じ根。

## 8. 読み取れないこと（NOT RUN）

- E・A・B・D の採否と実装。
- `trace` に release 系を載せる道（subject 化が前提）。
- 新3種の source location（Source Index が記録していない）。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 9. 位置付け

**第57ラウンドで自分で起票した欠陥を、その翌ラウンドで解消した回**である。設計 → 検出 → 起票 → 解消の1サイクルが、計画文書と証拠つきで閉じた。

途中で**安全不変条件にぶつかり、当初の ID 設計を捨てた**。`@` を使う案は実装が簡単だったが、`ValidateInspectionProjection` の「canonical の `id` は要求 ID と一致」を破る。不変条件を緩めるのではなく、subject の持ち方を変えた。

**試験が最初は穴だったことも記録した。** 変異4種のうち2種は第1版で素通りした。広告と実装の突き合わせを足して初めて捕まる。
