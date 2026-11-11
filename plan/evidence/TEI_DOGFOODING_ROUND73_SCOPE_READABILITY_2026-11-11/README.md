# 節の無い種でも読めることを `model overview` が言う（2026-11-11、第73ラウンド）

- 対象: [#4128](https://github.com/hat47x/tei-platform/issues/4128)（E 案の拡張）。
- 前段: [第72ラウンド](../TEI_DOGFOODING_ROUND72_UNREAD_SOURCE_2026-11-11/README.md)。
- 実行日: 2026-11-11（Asia/Tokyo）。
- 状態: **`readableWithoutSection` を追加。全成果物が CLI で読め、節の有無と可読性を区別できるようになった。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行する。`customer.customer`（未マッピングだった型）の永続化マッピングと solution release を CLI だけで設計し、**全成果物を CLI で読み返す**。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `2a9a14c07`（`origin/main`） |
| CLI実行物（設計時） | `sha256:d13f164567c81ba30f07cecdaf292942816585e40f04ff17f93b6a11b9daac40` |
| CLI実行物（修正後） | `sha256:f0467cc489e3cc82c343a2d0970fd2faf8c89de32d897c358ac0d0c83931a909` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 設計の経過（4往復）

| Attempt | 応答 | 学んだこと |
| --- | --- | --- |
| 1 | `ok` | 永続化マッピングは受理（読み返しで `cols=2`） |
| 2 | `SOLUTION_RELEASE_NOT_EXECUTABLE ... database release not found: master_data@2.1.0` | 先に database release が要る |
| 3 | `DATABASE_RELEASE_NOT_MANAGED_HERE ... ownership external-observed` | **`master_data` は外部所有なのでこの作業場所から release できない** |
| 4 | `ok`（`kind database-gate` ＋ `contract master_data.customer ^2`） | 他チームが release する**契約を gate する**のが正しい形 |

**Attempt 3 と 4 が対**である。診断は「外部所有だからできない」と名指しし、正しい代替は**構文の細部ではなくモデルの規則**だった。これは価値ある設計上の学びである。

## 4. 見つけた欠落

設計は通り、全成果物は `inspect` で読めた。

```text
database              master_data          ok
environment           development          ok instances=2
persistence-mapping   master_data.customer ok cols=2
solution-release      customer_rollout     ok versions=['1.0.0']
```

しかし `model overview` は依然として `database`・`persistence-mapping`・`environment`・`solution-release` を `notCoveredKinds` に並べていた。

**応答だけを見た設計者は、自分が足したマッピングが届いたのか、その種が単に節にならないだけなのかを区別できない。**

第59ラウンドで covered 側について直した曖昧さが、not-covered 側に残っていた。

## 5. 修正

`ModelSectionScope` に `readableWithoutSection` を足した。**CLI が広告するのと同じリスト**（`operations.InspectionSubjectKinds`）を読むので、両者の答えがずれない。

```go
for _, kind := range notCovered {
    if supportedInspectionKind(kind) {
        readable = append(readable, kind)
    }
}
```

### 5.1 実測（reference/workspace）

```text
readableWithoutSection: database, persistence-mapping, migration,
  application-release, database-release, solution-release, environment   (14中7)
CLI から読めない種: plugin, bundle, migration-provider, custom-go-binding,
  header-detail-contract-query-binding, contract-query-executor-binding,
  contract-query-sql-realization                                       (14中7)
```

`-h` にも「**節が無いことは宣言が無いことを意味しない**」と書いた。

## 6. 試験と変異

| 試験 | 内容 |
| --- | --- |
| `TestReadableWithoutSectionMatchesInspect` | not-covered の全種が readable か「inspect が受けない」のどちらかに会計される。重複も検出 |
| `TestReadableWithoutSectionIsNotEmptyForDeclaredLayers` | 宣言済みの層（database・environment・persistence-mapping）が readable に入る |

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 常に空にする | FAIL `"database" is accepted by inspect but was not listed as readable` |
| M2 | 被覆済みの種（`application`）を混ぜる | FAIL `"application" is readable but is not in notCoveredKinds` |

## 7. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/operations` | ok |
| `go vet` / `gofmt -l .` | 指摘なし / 空 |

## 8. 自分のミスを1つ記録

**証拠ディレクトリに `README.md` を書き忘れ、相対リンク検証が落ちた。** 番号付きの `.txt` だけを作り、README を後から書くつもりで忘れていた。`TestMarkdownRelativeLinksResolve` が両方向（索引と計画）から指摘した。

## 9. 読み取れないこと（NOT RUN）

- `plugin`・`bundle` 等 7 種を inspect に加えるか（#4128 の A 案相当）。
- #4318（workflow の字面固定）は依然 `workflow` スコープ待ち。
- 実DB・実Provider・実Host、HTTP・Desktop、意味の受理・権限・配備。

## 10. 位置付け

**「区別できない」を「区別できる」にした回**である。第59ラウンドは covered 側（`sectionTotal` が全部に見える）を直した。今回は not-covered 側（載っていない＝無い、と読める）を直した。**両方向が揃って初めて、`model overview` は自分の範囲を正しく説明できる。**
