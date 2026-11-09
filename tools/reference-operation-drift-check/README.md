# reference-operation-drift-check

受理済みの意味から導いた**操作面**を、後から**保存した照合値**と比べる、読み取り専用の参照です。[操作面の設計候補](../../plan/design/OPERATION_SURFACE_DESIGN_2026-11.md)の W6（実行時の照合）と15章未決4（既定で有効にするか）について、**案(a)「呼出し側が照合値を渡す」**を動く形で試すためのものです。

既定をどうするかは**未決**のままです。この参照は、案が成立することだけを示します。

~~~text
go run ./tools/reference-operation-drift-check/cmd/tei-operation-drift-check \
  -workspace reference/standard-crud/workspace -application ledger \
  -expected-world-fingerprint <保存した値> \
  -expected-model-revision <保存した値>
~~~

Workspaceは変更しません。操作面を再導出するのは、いま読んだ1回だけです。

## 比べるもの・比べないもの

| | 内容 |
| --- | --- |
| 比べる | `WorldFingerprint` と `ModelRevision`（`tei operations` が既に出力している値） |
| 比べない | `Basis`（persist写像の値であり、属性の追加では変わらないため） |
| 比べない | 操作の一覧・gap・規則の照合値（この参照の対象外） |

## 状態

| 状態 | 意味 | 終了コード |
| --- | --- | --- |
| `current` | 渡された値が、いま読んだ操作面の値と一致する | 0 |
| `stale` | 一致しない。`STALE_OPERATION_SURFACE` を返す | **1** |
| `unchecked` | 照合値を渡していない。**PASS ではない** | 0 |

`unchecked` を `current` と同じ値にしないのは、「照合しなかった」を「照合済み」と誤読させないためです。

## この参照が示さないこと

- 既定で有効にすべきか（15章未決4）。
- 案(b)（保存済み操作面との突き合わせ）と案(c)（実行時の再導出）との比較。
- 実行の入口（`execution.Invoke`・Command）への接続。
- 実行直前に意味が変わったことの検出（読んだ操作面は現在の意味から導いたものなので、この形では一致する）。
- 実DB・実Provider・権限。

実行記録は [`plan/evidence/OPERATION_DRIFT_CHECK_REFERENCE_2026-11-07`](../../plan/evidence/OPERATION_DRIFT_CHECK_REFERENCE_2026-11-07/README.md) にあります。
