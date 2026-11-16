# 受領書stubと捕捉原本を分離し、manifest を実ファイルに合わせた（2026-11-17、第89ラウンド）

- 解消: [#5176](https://github.com/hat47x/tei-platform/issues/5176)
- 前段: [第88ラウンド](../TEI_DOGFOODING_ROUND88_REA_CANDIDATES_RESTORED_2026-11-17/README.md)
- 状態: **`cmd/tei` の FAIL が 2 → 1。manifest の不一致は 0。**

## 1. 目的

前ラウンドで残った `TestREADualEntryCapturedInputAndIdentityBoundaries` を解消する。#5176 は「原本を戻すか manifest を直すか」の判断だったが、**根拠を測れば決まる**ので決めた。

## 2. 最初に試した案（失敗）

捕捉原本（78000バイト、`7ea3780d…`）を `analysis.json` に**上書きで戻す**:

| パッケージ | 変更前 | 原本に戻した後 |
| --- | --- | --- |
| `./internal/operations` | FAIL=0 | **FAIL=1** |
| `./cmd/tei` | FAIL=2 | **FAIL=4** |

```text
--- FAIL: TestCapturedREAReceiptCannotSupplyAbsentSemanticGraph
    captured normalized_result={} must not produce a semantic graph
```

**stub は意図的なものだった。** グラフを持たない受領書が fail-closed になることを検査する3テストが、まさに `analysis.json` の現状を必要としている。**上書きは誤り。**

## 3. 決めた形: 2つの役割を2つのファイルに分ける

| ファイル | 役割 | 根拠 |
| --- | --- | --- |
| `analysis.json` | **受領書のみ**（823→829バイト） | 「グラフの無い受領書は fail-closed」を検査する3テストが使う |
| `captured-delivery.json` | **捕捉原本**（78000バイト、UTF-16LE BOM） | 実配信を読む回帰テストが使う |

`cli_rea_dual_entry_contract_test.go` の参照先を後者へ変え、**なぜそうしたかをコメントに残した**。

## 4. manifest の不一致も直した（#5176 の本体）

`manifest.json` は「捕捉bytesの digest を列挙する」と設計文書に書かれているが、実際と食い違っていた。

| エントリ | 宣言（修正前） | 実ファイル |
| --- | --- | --- |
| `rea-6.8.0/analysis.json` | 520バイト / `7ea3780d…` | **829バイト / `6c84cfb3…`** |
| `rea-6.8.0/view.jsonl` | 618バイト / `dd936b0c…` | **929バイト / `561e927d…`** |
| `rea-6.7.0/analysis.json` | 516バイト | **829バイト** |
| `rea-6.7.0/view.jsonl` | 152バイト | **125バイト** |

**520バイトと `7ea3780d…`（78000バイトの digest）を同時に宣言している**という、それ自体で矛盾した状態だった。全エントリを実ファイルに合わせ、`captured-delivery.json` のエントリを追加した。**不一致は 0。**

## 5. 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 参照先を stub に戻す | FAIL `captured fixture is not the recorded UTF-16LE BOM delivery`（**元の症状を再現**） |
| M2 | 捕捉原本の BOM を UTF-8 に差し替える | FAIL 同上（**UTF-16LE 検査が効いている**） |

## 6. 結果

| 対象 | 変更前 | 変更後 |
| --- | --- | --- |
| `./cmd/tei` の FAIL | **2** | **1** |
| `./internal/operations` | 0 | **0** |
| `./internal/externalanalysis` | 0 | **0** |
| `./internal/workspace` | 0 | **0** |
| manifest の不一致 | **4件** | **0件** |

残る1件 `TestValidateReportsSyntaxErrorsInIndependentFiles` は **plain main でも FAIL** する既存件。

## 7. 読み取れないこと（NOT RUN）

- 残る1件の原因（plain main でも FAIL）。
- 元の REA 実行環境での再捕捉（本ラウンドは既存 bytes の整理のみ）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 8. 位置付け

**「上書きで戻す」を先に試して壊し、役割の分離に直した回**である。stub が意図的なものだと分かったのは**壊した後**だった。**同じ置き場所に2つの役割を負わせていたのが原因**で、ファイルを分けることで両方の検査が同時に成立した。
