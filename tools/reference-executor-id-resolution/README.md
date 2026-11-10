# reference-executor-id-resolution

環境の **deployment-executor Binding** から executor の `CapabilityID` を解決し、**plugin lock がその ID を認めるか**を確かめる、読み取り専用の参照です。既存の `routing.RouteCommand`・`pruntime.BuildPlan`・`Capability.SupportsAction` だけを使い、新しい解決器は作りません。

~~~text
go run ./tools/reference-executor-id-resolution/cmd/tei-executor-id-resolution \
  -workspace reference/workspace -environment development \
  -action database-release -database sales_core \
  -plugin-lock reference/workspace/.tei/plugin-lock.json \
  -hard-coded-id migration.atlas
~~~

## 3つの段

| 段 | 使う既存の関数 | 意味 |
| --- | --- | --- |
| 解決 | `routing.RouteCommand` | Binding がこの action の executor を選ぶ |
| 受入 | `pruntime.BuildPlan` → `EnvironmentPlan.Capability` | lock から作った plan に、その ID の能力がある |
| 支持 | `Capability.SupportsAction` | その能力が action を支える |

**`-hard-coded-id` は比較のためだけに並べます。解決には使いません。**

## 状態

| 状態 | 意味 |
| --- | --- |
| `RESOLVED` | Binding が ID を選び、plan が認め、能力が action を支える |
| `BINDING_MISSING` | この action に対する Binding が無い（ID は空） |
| `NOT_IN_PLAN` | ID は選ばれたが、lock の plan にその能力が無い（または曖昧） |
| `ACTION_UNSUPPORTED` | 能力はあるが、action を支えない |
| `LOCK_REJECTED` | lock を読めない／形式が違う |

**`-plugin-lock` を渡さない場合も `LOCK_REJECTED`** です。この参照は lock を発明しません。

## 版の形（v0）

C-1・C-2 は `v1` で「入場前の拒否」と「入場後の不確定」を分けます。**C-3 は `v0` ですが、それは遅れではありません。**
この参照は `Executor` も `Carrier` も一度も実行しないため、**入場後の不確定という状態が存在しません**。
5つの状態はすべて**実行前**の判定です。実行を伴うようになった時点で、C-1/C-2 と同じ `ranCount`／
`refusedCount`／`unverifiedCount` の分離が必要になります。

## この参照が示さないこと

- 参照実装の直書きを製品へ変えるべきか（判断材料の C-1〜C-4 は未決）。
- Binding の妥当性（`tei validate` の検証は通しません）。
- 実配備の成立（`executor.Execute` を呼びません）。
- 認証・権限・永続的な receipt・外部状態の確認。

実行記録は [`plan/evidence/EXECUTOR_ID_RESOLUTION_REFERENCE_2026-11-09`](../../plan/evidence/EXECUTOR_ID_RESOLUTION_REFERENCE_2026-11-09/README.md) にあります。
