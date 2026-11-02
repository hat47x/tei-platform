# Function Patternの前提・保証と合成可能性：異質Domainの反例検証（2026-11）

- **状態**：2026-11-02、設計・有限合成Reference。公開DSL/Manifest・製品契約・一般形式証明・実Target受理・業務意味受理は**未採用／未実施**。
- **Issue Owner**：[#2507](https://github.com/hat47x/tei-platform/issues/2507)（Function Patternの適合・構成）。
- **意味演算・接続のOwner**：[#988](https://github.com/hat47x/tei-platform/issues/988)（Operator、Clock、Model of Computation）、[#2495](https://github.com/hat47x/tei-platform/issues/2495)／[#2544](https://github.com/hat47x/tei-platform/issues/2544)（Facet、Law）。
- **既存Contract/Effectとの接続**：#2506（Data Context）、#2497（意味保存写像）、#2593（Measure再集計）、#2597（編集と複合Save）、#2598（Intent→Request）、#1174（実Authority/Effect/Receipt）、#2390（外部Consumer）、#2586（統合Scenario）、#2514（独立Oracle）。
- **横断計画**：[Draft #2503](https://github.com/hat47x/tei-platform/pull/2503)・[#2493](https://github.com/hat47x/tei-platform/issues/2493)（AI-first・異質領域の要求拡張）。上記の既存担当に新しい第二正本・新共通Host・実行器を追加する計画ではない。

## 1. 問題：「単体の型が合う」は「合成後の業務意味が保たれる」ではない

Function/PatternのDefinitionを同じ型のPortで接続できても、データの粒度、完全性、期待するOperation、時点、可能な失敗、観測可能なEffectが異なれば、全体の適合は成立しない。

- 上流が表示ページの1件を取得し「その1件の数量は正しい」と言っても、下流の「親Hの全明細が適切」を確定できない。
- 上流が集計したAVG=15だけを下流へ送っても、別Grainでの再平均にはsum/countのような中間状態が必要になり得る。
- 元の`submit` Functionと差し替え先の`cancel` Functionが同じ業務Entity・同じFieldに作用しても、目的・業務Actionの意味は異なる（#2598 IB-12）。
- 同じ成功時の値を返すProvider v1/v2でも、v2が新たに`unknown`や部分結果を返すなら、下流の「必ずconfirmed」を保証しない。
- 子FunctionにCommitが発生した後の親のCancelは、過去のEffectを無条件に取り消さない。補償を別Actionで実施したとしても、実施前の歴史・Receiptは残る。
- 10Hz認知器の状態出力と60Hz物理Tick入力は、型が同じでも時間対応が明らかではない。sample-hold等のBridgeを明示した場合も、実Target上の実装適合は別途確認する。

これらはUIの有無やTree/Graph表示とは独立した**Function合成の観測可能な意味**の問題。

## 2. 新しい公開型を確定せず、用途別の前提・保証・Contextを比較する

次の三つを、既存Semantic/Operation/Provider/Contextを参照する**設計比較票**で表現する。

| 観点 | 意味 | 必要に応じて参照する情報 |
| --- | --- | --- |
| **Assumptions / 要求前提** | 特定のConsumer/Claim/Operationが必要とする条件 | Entity/Relation role、Grain/Scope/完全性、Measure中間状態、Unit、Clock、Action purpose、必要なResult/Effect条件 |
| **Guarantees / 提供される保証** | あるSource・Contract・対象範囲と仮定の下で上流が提供すると宣言した内容 | 型付きOutput、結果状態集合、Actor/State分離、Version、観測の粒度・時点、失敗/未知 |
| **Context / 適合環境** | 同じSource/Contractを利用する際の利用時点・実現条件 | Definition/Instance/Anchor、Target/Provider版、Source/Rule revision、読取りScope、Clock、現在Grantの検査位置 |

抽象的な局所適合の問いは、同じContextと明示変換の下で、`G_up ∧ K ∧ T ⇒ A_down`が成り立つかどうか。ただし、下流が現在Grantを要求する場合、型や静的宣言だけでは実行可否を導けない。実際の実行時権限・Commitは既存Host/Providerに委ねる。

**重要**：この含意式は将来の検証対象を示す記法であり、現行TEIに一般的な定理証明器が存在するという主張ではない。

### 2.1 利用目的ごとに必要な保証が異なる

| 用途 | 主要な必要条件 | 合成だけで確定してはいけない事項 |
| --- | --- | --- |
| Read-only説明 | Source/Scope/Grain/Result Sliceの出所、未知の残余 | 全件の正しさ・不存在 |
| 集計/Chart | Measure/軸/時点、Identityと中間状態 | 集計行への自動更新・Provider間同値 |
| View変更→Intent | Action意味、対象ID、Selection epoch、元データ版 | 現在Hostの実行権限 |
| Headless計算/Simulation | Input/State/Clock/Operator Law/Seed・誤差 | 外界やDBのEffect確定 |
| 業務Actionの実行 | Action/Target/Owner、Host現在権限、Tx可能性 | Ackから推定したBusiness Completion |
| 結果再利用/Replay | Request/OperationID/Receiptの因果対応と実Output版 | 確定EffectのReplay＝当時のOutput byte再現 |

「Functionの接続」にすべて同じ検証ルールを適用するのではなく、用途が要求する保証を明示的に指定する。将来の標準Plugin作者には、既存Data/Relation/Operatorで得られる性質を再利用可能にし、上位の代数概念を一律に手書きさせない。

## 3. 局所接続が成立しても複合全体には追加の義務がある

### 3.1 同一性・状態と入れ子

Function Definitionを二つ配置したら、Instance ID、Local State、Selection epoch、Editing Session、Clock/Seed、未完了Requestを区別する。明示的に共有する状態がある場合だけ、その所有者、競合、Snapshot境界の契約を追加する。

Composite内部のFunctionを入れ替える、ネストする、FlatなExecutionへ変換する際には、**外部から見えるOutput/失敗/Unknown/時間/Effect要求の意味保存**が必要。単に内部ノード・ポート数が一致しても同等とは言えない。未観測の内部状態まで保存を要求するかどうかはConsumerに相対的に決める（#2497）。

### 3.2 取消・部分成功・因果関係

子のEffectがすでに確定したとき、親のCancelは少なくとも「今後の実行停止要求」「補償Actionの候補」「保存済みEffectの履歴上の確定」と別の意味を持つ。局所成功の合成による「全体原子性」への昇格をしない。実原子性・Effect Fenceは#1174、複数ContextのSave Intentは#2597が所有する。

`child_unknown`の後で親が`outer_success_only`だけを見せる場合、未確認の結果が隠蔽される。UIが成功だけを表示しても、業務完了を意味しない。同様にReplayで実書込みが重複しなかった事実と、初回BusinessResultが履歴的に再現された事実は異なる。

### 3.3 Clock・Feedback・実現差

Discrete / fixed-step / event-driven / continuousの分類、Actor・Operatorの時間意味、feedback delayや独立Solverの必要性は#988へ委譲する。

局所の型付きPortが接続可能でも、無遅延Cycle、Clock変換、イベント順序、外部物理の単位・精度、Providerの失敗方式は、合成全体で別条件となる。10Hz→60Hzへの明示Bridgeは**候補**であり、その存在だけでGodot実PhysX/物理挙動との同等性を保証しない。

## 4. 有限反例Corpus：AG-01〜20（合成・局所実測）

`plan/fixtures/function-pattern-ag/check_ag.py`は**Python 3.13.5標準ライブラリのみ**の自作試料。構成上の入力`Offer`（提供すると宣言された値）と`Need`（下流要求）を、限定された属性と代表Traceに対して比較する。

結果区分：

- **contradicted**：具体Fixtureで要求と衝突している。提示された入力のもとで要求が一般に成立するという主張を拒む反例。
- **not_established**：未知・必要な中間状態の不足・実行時Grant・一Traceで不足する一般性。**不能と証明したのではない**。
- **fixture_consistent**：定義した有限の宣言欄では衝突が見つからない。**業務意味受理・ターゲットConformance・一般論理的な証明ではない**。

| ID | 圧力と反例 | 判定の焦点 |
| --- | --- | --- |
| AG-01〜04 | partial→full、fullの限定一致、coverage未観測、Group≠Line | 出現・集合・Grainを型名で近似しない |
| AG-05〜06 | AVGしかなくsum/countがない／sum/count宣言済み | 再集計には必要な十分情報を区別（#2593） |
| AG-07〜08 | submit≠cancel、古いAction version | 意味的Action目的とRevisionを別に保持 |
| AG-09〜10 | 10Hz→60Hz、明示sample-hold候補 | Clock不一致とBridge未資格を区別 |
| AG-11〜12 | 新Providerでunknown結果が増える、実Host Grant必須 | 同じSuccessのOutputでも失敗集合/現在実行資格は別 |
| AG-13 | 同じFunction定義の二配置が同じmutable Stateを暗黙共有 | Definition共有とInstance状態分離 |
| AG-14〜17 | Commit後Cancel、子Unknownの親Complete化、隠蔽、正常一Trace | 取消・失敗可視性・一Traceからの一般化を識別 |
| AG-18〜20 | 異単位、異Scope、結果集合未観測 | Unit・対象・観測の資格を別に判断 |

### 4.1 実行と確認済みの限定範囲

- 隔離コンテナで`python3 check_ag.py`：`unittest 20/20 PASS`。
- `python3 -m py_compile check_ag.py`：PASS。
- 実行したファイルは**9292 bytes、SHA-256 `21fd8a3c42f5ba88a0d94c2514e5146fa558c7e08d920a5c6e0faa3e972fef40`、Git blob SHA-1 `e2188955cbf9aed4eb3c36256bd11039b2f8c4e0`**。GitHub保存Blobとの一致を確認した。
- ケースはすべて**自作の有限Fixture**。未実施：正式TEI Function/Data Context/DSL/Operator実入力、Go試験、外部Host、実Provider/Authority、Browser/GUI、Godot、定理証明、Owner受理、実際のTarget間の振る舞い同値。
- 有限反例による局所的な「不適合」は有益。一方、宣言された属性が一致するだけで`G⇒A`の一般証明やProvider実能力の成立を主張しない。独立Oracleの受入と実演算は#2514/#1131。

## 5. 既存Issue/PRの責務へ接続する開発順序

| レーン | 次の実証 | 根拠の昇格を止める境界 |
| --- | --- | --- |
| **FP/A0** | #2507のDefinition/Instance/Realizationと#988のConnector/Operatorを、AG-01〜20の要求情報へ静的Mapping | 属性不足の候補を新Core Type導入理由と直結しない |
| **FP/A1** | 同じ固定H/D1/D2および別Domainを、既存Scenario/CUI read-onlyで二つのFunctionとして接続 | Python合成PASSを正式TEI 接続へ流用しない |
| **FP/A2** | 同じContract形状の二Provider／同一定義の二Instanceで、結果の種類・Clock・Unit・Scopeの差を測る | Provider差替え＝実Host可搬性と扱わない |
| **Save/A2** | #2597/#2598の個別Request/依存・保存単位と#1174のTx/Receiptを既存所有で検証 | A/Gの宣言が実commitを保証しない |
| **Domain Pressure** | 組織・Measure、長時間承認、Sensor→Cognition→Motion、物理Connector、SHOへの説明を増やす | 専門Engine/Hostの責務をTEI Coreへ移さない |
| **Product Evidence** | #2586の同版一気通貫、#2018のAI指示量・Core追加・Adapter再利用測定 | 合成Fixture件数を製品の網羅率にしない |

**設計判断待ち**：比較票のどの欄を既存Context/Typed Reference/Service Contractから参照すべきか、Port/Connectorとして公開すべき最小部分、静的資格と実Target資格の照合点、Nested/Flatteningで保存する外部観測の定義、Plugin差替え時の再資格化条件。

**撤回条件**：A/G用の独立SSOTが増えてMeaning/Action/Contextを複製し始める、標準Plugin作者に無用なFacet/Clock宣言を強制する、有限PASSを一般証明・実Host Effectへ自動昇格する、新しいTEI独自Hostを必要とする設計になった場合。これらが現れれば既存契約への参照設計へ戻し、反例と不足情報をOwnerへ還元する。
