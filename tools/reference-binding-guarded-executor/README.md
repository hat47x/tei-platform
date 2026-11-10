# reference-binding-guarded-executor

[配備実行アダプタは Binding を照合できない](../../plan/evidence/ADAPTER_BINDING_BLINDSPOT_2026-11-09/README.md)という記録への、**呼出し側での最小の対処**を示す参照です。**アダプタは変更しません。**

## 何をするか

アダプタは `Plan` と `CapabilityID` しか持たず、環境の Binding を知りません。そこで**呼出し側が先に `routing.RouteCommand` を引き、要求した ID が Binding の選択と一致するかを確かめます。**

| 段 | 使う既存の関数 |
| --- | --- |
| 解決 | `routing.RouteCommand` |
| 比較 | 要求した ID と、解決結果の `ExecutorID` |
| 受入 | Host の registry にその ID があるか |

**一致しなければ、アダプタへ渡す前に拒否します。**

## 状態

| 状態 | 意味 |
| --- | --- |
| `SERVED` | 要求した ID が Binding の選択と一致し、Host が登録している |
| `NOT_MATCHED` | **Binding は別の executor を選んでいる**（この参照の要点） |
| `NOT_BOUND` | その action に Binding が無い |
| `UNKNOWN_EXECUTOR` | Binding が指す executor を Host が持っていない |

## なぜアダプタを直さないか

**アダプタに `Environment` が無いため、内部では照合できません。** `Environment` を足すのは構造変更になります。この参照は、**構造を変えずに同じ保証を得られるか**を示します。

## この参照が示さないこと

- **アダプタを直すべきか**（設計判断。`Environment` を渡す案、`Plan` に Binding を載せる案などがありえます）。
- **executor が動くこと**。照合は「選ばれた ID か」だけを見ます。
- 実配備、実 Provider、外部 Effect、現在の Grant。
- 認証・権限・永続的な receipt。

記録は [`plan/evidence/BINDING_GUARDED_EXECUTOR_REFERENCE_2026-11-09`](../../plan/evidence/BINDING_GUARDED_EXECUTOR_REFERENCE_2026-11-09/README.md) にあります。
