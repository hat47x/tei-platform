# business-procedure で自分の capability が解決でき、realization の拒否が有効値を言わない（2026-11-18、第97ラウンド）

- 対象: `internal/businessprocedure/project.go` の `BUSINESS_STEP_REALIZATION_UNKNOWN`。新規起票なし（その場で解消）。
- 前段: [第96ラウンド](../TEI_DOGFOODING_ROUND96_EDGE_IDENTITY_STATEMENT_2026-11-18/README.md)
- 状態: **capability の解決は成立。拒否が3つの有効値を名指しするようになった。**

## 1. 目的

目的文の「再びドッグフーディングに戻る」を実行した。**自分の equip ドメインを business procedure に組み込み、宣言した capability が本当に解決されるか**を確かめる。

## 2. 固定した入力

| 要素 | 値 |
| --- | --- |
| Source | `b4e1fd731`（`origin/main`） |
| CLI実行物（設計時） | `sha256:c3e005005261f4a106f43457291c76bc4eb711384a4b8e8b57f5abe068e05fc1` |
| 環境 | WSL2 Ubuntu 22.04.5、Go `1.26.11 linux/amd64` |

## 3. 設計の経過（CLI のみ）

| Attempt | 応答 | 学んだこと |
| --- | --- | --- |
| 1 | `json: unknown field "identityStatus"` | `DisallowUnknownFields`。推測したフィールドは通らない |
| 2 | `procedure schemaVersion "business-procedure-reference-v0" is not "business-procedure-model-reference-v0"` | **正しい版を名指ししてくれた** |
| 3 | `ok`、ただし `BUSINESS_STEP_REALIZATION_UNKNOWN` と `BUSINESS_MANUAL_STEP_HAS_FUNCTION` | realization の語が違う |
| 4 | `ok`、診断0件、**function 解決** | `realization: "application"` |

**Attempt 2 は模範的**だった。**期待値をそのまま述べている**ので、私は次の試行で当てられた。

## 4. 動いたこと（解決の実測）

```text
step L-01 | manual: False | noFunction: False
function: {kind: capability, id: equip.loan_register,
           subject: semantic-capability:equip.loan_register,
           artifact: semantic/equip/desk.tei, line: 3, column: 12,
           role: definition, fingerprint: sha256:dd57c0f5...}
unresolvedFunction: None
diagnostics: []
```

**宣言した capability が、Source の位置と fingerprint まで含めて解決された。**

境界も確かめた:

| 入力 | 結果 |
| --- | --- |
| 存在しない capability | `unresolvedFunction` に残り、`BUSINESS_REFERENCE_UNRESOLVED`（**推測しない**） |
| 別アプリ配下の capability | **解決される**（resolver は application で絞らない） |

## 5. 見つけた穴: 有効値を言わない拒否

```text
step "L-01" declares unknown realization "application-use"
```

`Realization` は**3語の閉じた集合**（`manual`・`application`・`system`）で、`Valid()` がそのリストを回しているのに、**拒否文は一語も挙げない**。私は `application-use` と書いて落ち、**次の正解を推測する必要があった**（Attempt 4 まで1往復）。

## 6. 修正

同じリストから文言を組み立て、**3語を名指しする**ようにした。

```text
BEFORE  step "L-01" declares unknown realization "application-use"
AFTER   step "L-01" declares unknown realization "application-use";
        valid values are manual, application, system
```

**受理する集合は変えていない。** 変えたのは説明だけである。

## 7. 変異試験

| # | 変えた箇所 | 結果 |
| --- | --- | --- |
| M1 | 旧文言に戻す | FAIL `the refusal omits "manual"` |
| M2 | 文言が3語のうち2語しか挙げない | FAIL `realizationNames lists 2 of 3` |

**M2 は最初、`strings.Join` の外に文を足す形で書いてビルドを壊した。** 変異が壊れているのは証拠にならないので、`realizations[:2]` に変える形へ組み立て直した。

## 8. 回帰確認

| 対象 | 結果 |
| --- | --- |
| `./internal/businessprocedure` | ok |
| `./cmd/tei` | ok（332秒） |
| CI gate（`--fast`） | **6ステップすべて PASS** |

## 9. 読み取れないこと（NOT RUN）

- resolver が application で絞らないことが**意図かどうか**（実測はしたが、設計意図は未確認）。
- 実DB・実Provider・実Host、HTTP・Desktop。

## 10. 位置付け

**「要求はするが、従い方が書いていない」の5例目**である。ただし今回は**同じコマンドが直前の試行で正しい版を名指ししてくれた**（Attempt 2）。**できる時とできない時の差**が、この欠陥類型の実態である。
