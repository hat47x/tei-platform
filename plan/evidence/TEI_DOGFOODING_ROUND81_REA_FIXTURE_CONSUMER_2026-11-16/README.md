# rea の refactor が fixture を差し替え、production consumer を更新していない（2026-11-16、第81ラウンド）

- 起票: [#5176](https://github.com/hat47x/tei-platform/issues/5176)
- 前段: [第80ラウンド](../TEI_DOGFOODING_ROUND80_COVERAGE_DEBT_2026-11-16/README.md)
- 状態: **studiohost の4テストを修正。manifest の不整合は起票のみ（別欠陥）。**

## 1. 目的

再開後も `repository-health` が赤のままなので、**最大の塊（REA 系10件）の原因を特定して直す。**

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `dc3f310ed`（`origin/main`） |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 3か所の同じエラーを、計測で切り分けた

`LocalREAReader.Read` は3か所で同一の公開エラー `external Evidence is unavailable` を返す（workspace 拒否・ファイル読み取り失敗・選択エラー）。**推測せず計測**した。

```text
STEP 1 host.workspace              ok
STEP 2 ReadSource                  ok bytes=829
STEP 3 ExternalCandidatesFromBytes error: Evidenceにsemantic graphの宣言がありません
```

**原因は fixture の中身だった。**

| 対象 | 値 |
| --- | --- |
| 現在の `rea-6.8.0/analysis.json` | **829 bytes**（`normalized_result` が空） |
| `716fb3612` 時点の同ファイル | **78,000 bytes**（`semantic_graph` あり） |
| テストが固定する digest | `sha256:7ea3780d…` = **旧78KB の digest** |

差し替えたのは `606134e63 refactor(rea): restrict intake to evidence receipts`。**`internal/studiohost` は一切触っていない**（`--stat` で0件）。旧 fixture で同じテストが通ることも確認した（`606134e63~1` の testdata で実行 → `ok`）。

## 4. 修正方針: refactor を巻き戻さない

`decodeSemanticGraph` は main に残り、`LocalREAReader` は production コードで経路も生きている。**receipt 専用化は `internal/externalanalysis` の意図的な変更**であり、欠陥は「読む側を直さなかったこと」。

したがって **`internal/studiohost` に自前の graph-bearing fixture を持たせ**、3テストの参照先をそこへ変えた（78KB を `716fb3612` から逐語コピー、digest 一致を確認）。

```text
internal/studiohost/local_rea_multiroot_test.go      （参照先のみ）
internal/studiohost/local_rea_reader_test.go         （参照先のみ）
internal/studiohost/testdata/rea-6.8.0/analysis.json （新規）
```

**production コードは変更していない。**

## 5. 変異試験（どちらも元の失敗を再現）

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 参照を receipt 専用 stub に戻す | FAIL `external Evidence is unavailable` |
| M2 | コピーした fixture から `semantic_graph` を削る | FAIL `external Evidence is unavailable` |

**M2 が決定的**である。graph の有無だけが原因で、他の違いではない。

## 6. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `internal/studiohost`（全テスト） | **ok**（修正前は4件 FAIL） |
| `internal/externalanalysis`（refactor 側） | ok（**変更していない**） |
| `go vet` / `gofmt` | 指摘なし / 空 |

## 7. 直していないもの: manifest の不整合（別欠陥、#5176）

```text
rea-6.8.0/analysis.json  manifest 520 bytes / 実際 829          MISMATCH
                          manifest sha256:7ea3780d… / 実際 6c84cfb3…  MISMATCH
rea-6.8.0/view.jsonl     manifest 618 bytes / 実際 929          MISMATCH
```

**manifest は旧 fixture の digest を宣言したまま bytes だけ新しい値**になっている。**どちらを正とするかは所有者の判断**なので、本 PR では触っていない。

## 8. 残る赤（別原因）

`internal/operations`（`TestOrgCaseExplicitPersistenceEnablesReadCandidateOnly`）と `cmd/tei`（`TestPeriodsAdvance*`・`TestREADualEntry*`・`TestREAIntake*`）は **REA fixture とは別の失敗**。本ラウンドでは追っていない。

## 9. 読み取れないこと（NOT RUN）

- manifest を新 fixture に合わせるか、旧 fixture を戻すか（#5176）。
- 残る `internal/operations` / `cmd/tei` の失敗（別原因）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 10. 位置付け

**「意図的な変更」と「更新漏れ」を分けた回**である。fixture の差し替え自体は正しく、**読む側を直さなかったこと**が欠陥だった。refactor を巻き戻すのではなく、consumer 側に必要な fixture を持たせた。
