# 判断材料: 実行の側をどこで宣言へ繋ぐか（2026-11-08）

- 状態: **判断材料・未確定**。決定はこの文書では行いません。
- 対象の決定: [構成Kernel移行Gate](../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md) の「製品としての次の一辺」と、[成立状況](../../docs/status.md)の「製品Consumerからの旧Resolver撤去」
- 下流の正本: [構成Kernel移行Gate](../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)、[プラグインと配布一式](../../docs/plugins.md)、[操作面の設計候補](../design/OPERATION_SURFACE_DESIGN_2026-11.md)
- この文書の役割: 6本の実行記録から読める**選択肢と、決める順序**を示す。決定そのものは書かない。

## 1. 材料の出所

| 記録 | 示すこと |
| --- | --- |
| [構成Kernel移行Gateの依存の形を、いまのmainで確かめる](../../plan/evidence/COMPOSITION_GATE_DEPENDENCY_RECHECK_2026-11-08/README.md) | A（`cmd/tei`）は `composition` を含まず `operations` を含む。B（`go-host`）はその逆。`servicehost` の非試験 importer は0 |
| [servicehost に非試験の呼出し元が無いのはなぜか](../../plan/evidence/SERVICEHOST_CONNECTION_PATH_SURVEY_2026-11-08/README.md) | Invoker を組み立てる非試験の呼出し元は0。Coordinator を作る非試験の呼出し元は1件ある |
| [Runtimeへの経路は`servicehost`だけではない](../../plan/evidence/RUNTIME_CARRIER_ALTERNATE_PATHS_2026-11-08/README.md) | ピン留めあり1件（`servicehost`）、ピン留めなし3件 |
| [ピン留めしない3経路は、何のためにRuntimeを直接呼ぶのか](../../plan/evidence/RUNTIME_UNPINNED_PATH_PURPOSES_2026-11-08/README.md) | 3経路とも `composition.Coordinator` を使わない。用途は別々 |
| [配備実行アダプタは、誰がどう組み立てているか](../../plan/evidence/DEPLOYMENT_EXECUTOR_BINDING_SURVEY_2026-11-08/README.md) | 非試験の組み立ては4件、すべて参照実装。CapabilityID は直書きかローカル変数 |
| [環境の配備実行Bindingは、どこまで使われるか](../../plan/evidence/EXECUTOR_BINDING_CONSUMER_SURVEY_2026-11-08/README.md) | Binding は検証と計画（JSON）まで。`routing.Router` を組み立てる非試験の箇所は0 |

## 2. 現在の形（記録から読める範囲）

```text
宣言（Environment の Binding）
  → 検証（internal/validation）
  → 計画（routing.RouteCommands の結果を JSON へ）
  → 実行は別の入口（参照実装が固定の CapabilityID で直叩き）
```

- **計画と実行の間に、宣言を読む繋ぎ目がありません。**
- `servicehost` は Host世代をピン留めする繋ぎ目として存在しますが、**組み立てる非試験の呼出し元がありません**。
- `routing.Router` は Binding を実行へ繋ぐ繋ぎ目として存在しますが、**非試験で組み立てられていません**。

## 3. 選べる繋ぎ目（材料から読める範囲）

| # | 繋ぎ目 | 何をするか | 既にある部品 |
| --- | --- | --- | --- |
| C-1 | `routing.Router` を CLI で組み立てる | Binding を解決し、`Executor` を登録して実行する | `Router.Dispatch`（試験で動作確認済み） |
| C-2 | `servicehost` を CLI へ入れる | Host世代をピン留めして Runtime を呼ぶ | `NewSelectedServiceCarrierInvoker`（試験で動作確認済み） |
| C-3 | `DeploymentExecutor` の `CapabilityID` を Binding から解決する | 参照実装の直叩きを、宣言由来の能力へ変える | `routing.RouteCommand` の結果 |
| C-4 | 繋がない（現状維持） | 検証・計画だけを製品が担い、実行は外部の主体に委ねる | なし（現状） |

**この文書はどれも推薦しません。**

## 4. 決めると後続が解ける順序

| 順 | 決定 | 決めると解けること | 決めるまで止まること |
| --- | --- | --- | --- |
| 1 | **C-4 を選ぶかどうか**（繋ぐか、繋がないままにするか） | これが「繋がない」なら他3案は不要になる | C-1〜C-3 の比較を続けるかが決まらない |
| 2 | **C-1 か C-2 か**（配備実行から繋ぐか、Service Carrier から繋ぐか） | 製品 CLI の入口をどちらへ通すかが決まる | どちらの部品を先に非試験で組み立てるかが決まらない |
| 3 | **C-3 を足すか**（参照実装の直叩きを宣言由来にするか） | 参照実装4件の `CapabilityID` の扱いが決まる | 参照実装が「固定の能力を試す」意図を保てるかが決まらない |

**理由**: C-4 は他の3案を丸ごと不要にしうるので最初です。C-1 と C-2 はどちらも「実行の入口を製品へ引き込む」案で、**両方を同時に選ぶと2つの経路ができます**（現在の `servicehost` と `Runtime` 直接の関係と同じ形）。C-3 は C-1 か C-2 のどちらを選んでも影響するので最後です。

## 5. この文書で決めないこと

- 決定そのもの（採用・却下・期限）。Gate と成立状況の担当です。
- 選択肢の推薦。
- C-1〜C-3 の実装。部品は既にあり、**組み立てる場所を決めるだけ**です。
- 「繋ぐべき」という前提。C-4 は現状維持として**正当な選択肢**です。

## 6. この文書が示さないこと

- **繋がないことが不安全であること**。実行を外部に委ねる設計は、Gate が「実配備先の現在権限」を別に扱っていることと整合するかもしれません。
- **C-1・C-2 の安全性**。どちらも試験でしか動いておらず、実Host・実Provider では確認されていません。
- **材料の網羅性**。6本の記録はそれぞれ1回の観測で、`main` が進むと数は変わります。
- **どの案が Gate の G1〜G4 に効くか**。判定は Gate 本文の担当です。

## 7. 関連

- [構成Kernel移行Gate](../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)（E0 の棚卸しと「次の一辺」が正本）
- [プラグインと配布一式](../../docs/plugins.md)（選択の規則の正本）
- [操作面の設計候補](../design/OPERATION_SURFACE_DESIGN_2026-11.md)（15章の未決4件）
- [判断材料: 操作面の未決4件と、決める順序](OPERATION_SURFACE_DECISION_BRIEF_2026-11.md)（同じ形式の先例）
