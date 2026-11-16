# manifest の自己矛盾した digest 宣言を実ファイルに合わせた（2026-11-17、第89ラウンド）

- 解消: [#5176](https://github.com/hat47x/tei-platform/issues/5176)
- 前段: [第88ラウンド](../TEI_DOGFOODING_ROUND88_REA_CANDIDATES_RESTORED_2026-11-17/README.md)
- 状態: **`internal/externalanalysis` の manifest 不一致 4件 → 0件。全パッケージ FAIL=0。**

## 1. 目的

前ラウンドで残った `TestREADualEntryCapturedInputAndIdentityBoundaries` と #5176 を解消する。

## 2. 最初に試した案（失敗）

捕捉原本（78000バイト）を `analysis.json` に**上書きで戻す**:

| パッケージ | 変更前 | 原本に戻した後 |
| --- | --- | --- |
| `./internal/operations` | FAIL=0 | **FAIL=1** |
| `./cmd/tei` | FAIL=2 | **FAIL=4** |

```text
--- FAIL: TestCapturedREAReceiptCannotSupplyAbsentSemanticGraph
    captured normalized_result={} must not produce a semantic graph
```

**stub は意図的なものだった。** グラフを持たない受領書が fail-closed になることを検査する3テストが、まさに現状を必要としている。**上書きは誤り。**

## 3. 次に試した案（採用されたが、後に取り下げた）

`captured-delivery.json` を `externalanalysis/testdata` に追加し、回帰テストの参照先をそこへ変えた。テストは通った（変異2件で捕捉）。

**しかし main が先に同じ #5176 を別の形で解消していた。**

```text
948efe9b6 test(cli): REA二入口の回帰が自前のcaptured deliveryを読むようにする (#5176)
```

main は回帰テストに**専用の捕捉**（`cmd/tei/testdata/rea-6.8.0/analysis.json`、78000バイト、`7ea3780d…`、README 付き）を持たせていた。**私の追加分は同一バイト・同一 digest の重複**で、読み手がいない。**重複を作らず main の形に合わせ、自分の追加分を削除した。**

## 4. 本ラウンドの実質的な成果: manifest の自己矛盾を直した

`manifest.json` は設計文書上「捕捉bytesの digest を列挙する」ものだが、実ファイルと食い違っていた。

| エントリ | 宣言（修正前） | 実ファイル |
| --- | --- | --- |
| `rea-6.8.0/analysis.json` | 520バイト / `7ea3780d…` | **829バイト / `6c84cfb3…`** |
| `rea-6.8.0/view.jsonl` | 618バイト / `dd936b0c…` | **929バイト / `561e927d…`** |
| `rea-6.7.0/analysis.json` | 516バイト | **829バイト** |
| `rea-6.7.0/view.jsonl` | 152バイト | **125バイト** |

**520バイトと `7ea3780d…`（78000バイトの digest）を同時に宣言している**という、それ自体で矛盾した状態だった。全エントリを実ファイルに合わせ、**不一致 0**。検証はスクリプトで全エントリを突き合わせた。

## 5. 変異試験（採用しなかった案に対して）

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 参照先を stub に戻す | FAIL `captured fixture is not the recorded UTF-16LE BOM delivery` |
| M2 | 捕捉原本の BOM を UTF-8 に差し替える | FAIL 同上 |

**この2件は「私の案が機能していた」ことの証明**であり、最終的には main の形を採ったため不要になった。

## 6. 結果

| 対象 | 変更前 | 変更後 |
| --- | --- | --- |
| `./cmd/tei` の FAIL | **2** | **0** |
| `./internal/operations` | 0 | **0** |
| `./internal/externalanalysis` | 0 | **0** |
| `./internal/workspace` | 0 | **0** |
| manifest の不一致 | **4件** | **0件** |
| CI gate | 6 PASS | **6 PASS** |

## 7. 読み取れないこと（NOT RUN）

- 元の REA 実行環境での再捕捉（本ラウンドは既存 bytes の整理のみ）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 8. 位置付け

**「同じ問題を main が別解で先に解いていた」回**である。私は重複する仕組みを作りかけたが、**main の設計の方が良い**（テストが自分の入力を同梱し、README で説明する）と判断して取り下げた。**残ったのは manifest の訂正**で、これは main の変更でも直っていなかった自己矛盾である。
