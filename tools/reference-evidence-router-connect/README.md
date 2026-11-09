# reference-evidence-router-connect

`EvidenceProviders` の宣言から provider を解決し、**既存の `evidence.Router` で実際に入る**参照です。観測側（[`reference-observation-router-connect`](../reference-observation-router-connect/README.md)）の証拠側の対で、同じ4段の形が成り立つかを試します。

~~~text
go run ./tools/reference-evidence-router-connect/cmd/tei-evidence-router-connect \
  -workspace reference/workspace -environment development
~~~

## 何をするか

| 段 | 使う既存の関数 | 数える事実 |
| --- | --- | --- |
| 前提 | `rec.Mode == durable.ModeEvidence` | 証拠ゲートか |
| 解決 | `evidence.Binding`（`cmd/tei -plan` が既に使う関数） | `resolved` |
| 選択 | Host の registry を引く | `selected` |
| 入場 | `evidence.Router.Collect` | `invoked` |

**観測側との違い**: `Collect` は `durable.ActionRecord` を要求するため、`Environment` だけでは入り口に立てません。証拠ゲートでない記録は、Router の汎用エラーに流さず `NOT_EVIDENCE_GATE` として手前で分けます。

## 数え方

| 数 | 意味 |
| --- | --- |
| `collectedCount` | `COLLECTED` になった件数 |
| `refusedCount` | それ以外の件数 |
| `invokedCount` | **provider の `Collect` に入った件数**。応答が使えなくても数えます |

## 状態

| 状態 | 意味 |
| --- | --- |
| `COLLECTED` | 解決・選択・入場がすべて成立し、証拠の対象も合った |
| `NOT_EVIDENCE_GATE` | 記録が証拠ゲートではない（**入場前**） |
| `BINDING_MISSING` | その kind の宣言が無い、または重複している（**入場前**） |
| `NOT_REGISTERED` | 宣言が指す provider を Host が登録していない（**入場前**） |
| `NOT_CONFORMING` | provider が kind を支えない、または ID が合わない（**入場前**） |
| `COLLECT_FAILED` | **入場後**の失敗。証拠の対象不一致もここに入る（既存 Router が拒否する） |
| `EVIDENCE_INVALID` | Router が provider に入らずに応答した |

## 部分成功をどう表すか（CON-12d）

`attemptedCount` は試した件数、`complete` は**すべての試行が成立したか**です。
件数だけでは 1/1 と 1/2 を区別できないため、`attemptedCount` を併記します。
**一部だけ成功した場合は `complete=false`** で、拒否はそのまま残ります。
何も試さなかった場合も `complete=false` です。

**`complete` は「世界がそうなっている」証明ではありません。** 試した全部に答えが返ったという意味だけです。

## unknown を refusal と分ける（CON-12d 追補）

| 数 | 意味 |
| --- | --- |
| `refusedCount` | **入る前**に止めた回数。何も呼ばれていない |
| `unknownCount` | **入った後**に失敗した回数。`invokedCount` の内数 |

**入った後の失敗を `refusedCount` に数えると「実行していない」と読めます。** 実際には呼出しは起きており、世界を読んだかどうかは分かりません。そのため `unknownCount` として別に数えます。

**`unknownCount` は `invokedCount` の内数**です。3つの数を足し合わせないでください。

## この参照が示さないこと

- **証拠が外部状態を反映していること**。付属 CLI の Provider は `Outcome: pass`, `SourceRef: "reference-only"` を返し、何も読みません。
- **`pass` が正しいこと**。`pass` は provider の主張です。
- **製品 CLI へ `Collect` を足すべきか**（C-4 も候補）。
- 開示 Scope、Artifact の信頼、現在の Grant、認証・権限・永続 receipt。

記録は [`plan/evidence/EVIDENCE_ROUTER_CONNECT_REFERENCE_2026-11-09`](../../plan/evidence/EVIDENCE_ROUTER_CONNECT_REFERENCE_2026-11-09/README.md) にあります。
