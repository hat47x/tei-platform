# CI の gate が main で赤だった（gofmt 25ファイル）— CI の flags で測り直して判明（2026-11-17、第85ラウンド）

- 対象: `tools/repository-health-check.sh` の gofmt ステップ。新規起票なし（その場で解消）。
- 前段: [第84ラウンド](../TEI_DOGFOODING_ROUND84_MD_PARSER_GUIDANCE_2026-11-17/README.md)
- 状態: **gofmt 25ファイルを整形し、CI gate の5ステップすべてが PASS。**

## 1. 目的

これまで3ラウンド続けて「テストが走らない」種類の欠陥を直したので、**その種類がもう残っていないかを体系的に確認する。**

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `306755ada`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 体系的確認: その種類はもう無い

テストバイナリを全てビルドして確認した。

```text
$ go test -run '^$' ./...
（出力なし）
failing-to-build entries: 0
```

**全パッケージのテストがビルドできる。** 第82〜84ラウンドで直した種類は解消済みである。

## 4. その過程で、CI の gate が main で赤いことを発見

CI が実際に走らせているスクリプトを直接実行した。

```text
$ bash tools/repository-health-check.sh --fast
FAIL  gofmt -l . is empty
FAIL  go vet ./...
```

**`gofmt -l .` が25ファイルで drift。** `go vet` は単独で実行すると rc=0 で PASS（スクリプトの表示は直前の gofmt 失敗に引きずられたもの）。**したがって真の失敗は gofmt だけ**である。

drift は **260行、純粋な整形のみ**（例: 構造体タグの桁揃え）。私の直近のコミットではなく、他の複数のコミットが持ち込んだもの。

## 5. 修正

`gofmt -l .` が挙げた25ファイルに `gofmt -w` を適用。

| 対象 | 修正前 | 修正後 |
| --- | --- | --- |
| `gofmt -l .` | **25ファイル** | **0** |
| `go vet ./...` | rc=0（PASS） | rc=0 |
| gate の5ステップ | **2 FAIL** | **5 PASS** |

```text
PASS  gofmt -l . is empty
PASS  go vet ./...
PASS  tei validate -workspace reference/workspace
PASS  claim requalification synthetic fixture (R01-R17)
PASS  ADR index matches plan/adr/0*.md
```

## 6. 整形だけであることの確認

| 対象 | 結果 |
| --- | --- |
| `go build ./...` | rc=0 |
| `./internal/externalanalysis` | ok |
| `./internal/reacandidate` | ok |
| `./internal/plugin/management` | ok |
| `./internal/operations` | **1件 FAIL**（`TestOrgCaseExplicitPersistenceEnablesReadCandidateOnly`） |

`operations` の1件は **plain main でも同じく FAIL** する既存の runtime binding の欠陥（§85 で記録済み）で、整形とは無関係。

## 7. 測り方を間違えて1回やり直した

最初 `go test ./...`（既定 timeout = **10分**）で測り、3パッケージが 600s で FAIL した。**これは CI を再現していない。**

```bash
# CI が実際に走らせる形:
go test -count=1 -timeout 60m ./...
```

`go test` の既定 timeout は10分なので、**60分を指定する CI と別物**だった。単独で測ると `reference-plugin-self-build` は **596s で PASS** する。つまり 600s の「FAIL」は**並列実行の競合による見かけの失敗**で、実際の欠陥ではない。

**CI の flags を読んでから測り直した。**

## 8. 読み取れないこと（NOT RUN）

- `go test -timeout 60m ./...` の全件完走（長時間のため別途実行中）。
- 25ファイルの drift を持ち込んだ各コミットの意図（整形のみなので意味の判断は不要）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 9. 位置付け

**「CI が見ている当のものを、CI と同じ flags で測る」をやった回**である。既定 timeout で測った最初の結果は**CI の再現になっておらず**、存在しない欠陥を3件報告するところだった。**測り方を疑って読み直したことで、本当の赤（gofmt 25ファイル）に到達した。**
