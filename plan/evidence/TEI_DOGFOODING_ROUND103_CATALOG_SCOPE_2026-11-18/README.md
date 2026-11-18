# model catalog は migration・release を数えるが、列挙しない — そして理由を言わない（2026-11-18、第103ラウンド）

- 対象: `internal/operations/model_catalog.go`。既存 #4128 の残りを解消。
- 前段: [第102ラウンド](../TEI_DOGFOODING_ROUND102_SEMANTIC_ROOT_PLACEMENT_2026-11-18/README.md)
- 状態: **#4128 の主張は「半分正しかった」。数えるが列挙しない。理由を言う欄を足した。**

## 1. 目的

前2ラウンドと同じ方法で、最後に残った自分の起票 [#4128](https://github.com/hat47x/tei-platform/issues/4128) を**本文ではなく実測で**判定した。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `ea19dc858`（`origin/main`） |
| CLI実行物（設計時） | `sha256:91f503252221c8b138b4590ba772c724af22623608af3fe30054c2589ff36395` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. `#4128` の再現手順は**パスが違っていた**

本文は `databases/<db>/migrations/<id>.tei`・`releases/databases/<ver>.tei` と書いていたが、**loader は別の形を読む**:

```go
// internal/workspace/loader.go
migrations  -> databases/<db>/migrations/<id>/migration.tei   (ディレクトリ + migration.tei)
db releases -> releases/databases/<db>/<ver>.tei
solutions   -> releases/solutions/<solution>/<ver>.tei
```

本文どおりに置くと `WORKSPACE_SOURCE_ROOT`（「どの loader も読まない」）が出る。**正しい形に置くと警告は消え、`world` の件数が増えた**（migrations 1→2、databaseReleases 3→4、solutionReleases 2→3）。

## 4. 主張の判定: 半分正しい

| 見た目 | 実測 |
| --- | --- |
| `declarationCounts` | **数える**（migrations 1→2、total 79→82） |
| `entries` | **増えない**（72→72）。`migration`/`database-release`/`solution-release` は entry の kind に存在しない |
| `model overview` | `sectionScope.notCoveredKinds` に**3種とも明記**され、`readableWithoutSection` も持つ |
| `model catalog` | **そんな欄が無い**（`authorityLimits` はあるが「何を列挙しないか」を言わない） |

**「扱いません」は言い過ぎだが、「entry として辿れない」は事実。** そして **catalog はその理由を説明できなかった。**

## 5. 修正

`overview` が既に持つ `sectionScope` を **catalog にも載せた**。同じ関数を共有しているので、2つのコマンドが**別の答えを返せない**。

```text
model catalog -workspace W -> sectionScope:
  rootsAre: catalog-root-entries
  migration:          notCovered=True readable=True
  database-release:   notCovered=True readable=True
  solution-release:   notCovered=True readable=True
```

**挙動は変えていない。** 列挙する範囲は同じで、**「列挙しない理由」を言うようになった**だけである。

## 6. 変異試験（2回目は通ってしまった）

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | `out.SectionScope = sectionScope(...)` を削る | FAIL `scope does not name its selection rule` |
| M2 | 共有関数の covered 集合を空にする | **最初は2テストとも PASS** |

**M2 が通った理由**: 私のテストは「covered と not-covered が重ならない」ことだけを見ていた。**covered が空なら「重ならない」は自明に真**である。**逆向き（entry がある kind が not-covered に居ないこと）を足してから**、M2 が FAIL することを確認した。

## 7. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/operations` | ok |
| `./cmd/tei` | ok（198秒） |
| CI gate（`--fast`） | **6ステップすべて PASS** |

## 8. 読み取れないこと（NOT RUN）

- `entries` に3種を**実際に列挙する**案は、catalog の「addressable な subject」という契約を広げる設計判断なので**触っていない**。
- `inspect`/`trace` が3種を解決できるかは**未検証**（`readableWithoutSection` が「できる」と言っているが、私は実測していない）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 9. 位置付け

**「起票の主張を、再現手順の誤りまで含めて測った」回**である。本文のパスは**loader が読まない形**だったので、そのまま試すと `#4128` を「未再現」と誤判定していた。**正しい形に置くと、主張の半分は事実、半分は言い過ぎ**と分かった。**変異試験が通る理由を疑い、逆方向の検査を足した**のも前ラウンドと同じ轍を踏まないためである。
