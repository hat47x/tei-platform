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

## 2. 2026-11-08時点の形（当時の調査記録）

以下は**2026-11-08当時のSource調査**を保存したもので、現在のmainの非試験Caller件数ではない。C-1/C-2のReference実組立て後の状態と、製品接続の未成立は§10〜11で区別する。

```text
宣言（Environment の Binding）
  → 検証（internal/validation）
  → 計画（routing.RouteCommands の結果を JSON へ）
  → 実行は別の入口（参照実装が固定の CapabilityID で直叩き）
```

- **計画と実行の間に、宣言を読む繋ぎ目がありません。**
- 【当時】`servicehost`はHost世代をピン留めする繋ぎ目を持つが、非試験の呼出し元が0件でした。**後続のC-2参照ツールは構築を実行可能**にしましたが、通常CLI/Studioの本番Hostには未接続です。
- 【当時】`routing.Router`はBindingを実行へ繋ぐ繋ぎ目を持つが、非試験の呼出し元が0件でした。**後続のC-1参照ツールは実組立て・実呼出し済み**ですが、製品Dispatcherへの標準組込みは未決です。

## 3. 選べる繋ぎ目（材料から読める範囲）

| # | 繋ぎ目 | 何をするか | 既にある部品 |
| --- | --- | --- | --- |
| C-1 | `routing.Router` をDeployment入口で組み立てる | Deployment CommandのBindingを解決し、Host登録Executorを呼び出す | `Router.Dispatch`（参照で実組立て・有限実行済み。製品入口未接続） |
| C-2 | `servicehost` をService要求入口へ接続する | Host世代をピン留めし、Service RequestをCarrierへ運ぶ | `NewSelectedServiceCarrierInvoker`（参照で実組立て・有限実行済み。製品入口未接続） |
| C-3 | `DeploymentExecutor` の `CapabilityID` を Binding から解決する | 参照実装の直叩きを、宣言由来の能力へ変える | `routing.RouteCommand` の結果 |
| C-4 | 繋がない（現状維持） | 検証・計画だけを製品が担い、実行は外部の主体に委ねる | なし（現状） |

**この文書はどれも推薦しません。**

## 4. 決めると後続が解ける順序

| 順 | 決定 | 決めると解けること | 決めるまで止まること |
| --- | --- | --- | --- |
| 1 | **対象Operation／Targetで、TEI製品内と外部Hostのどちらが実行を所有するか**（C-4も含む） | 外部Host委譲の場合は、その主体から受領すべき証拠契約を定義できる | Ownerと現在Grant、Receipt/独立観測の所在が未確定 |
| 2 | **Deployment CommandはC-1、Service RequestはC-2という役割別の接続要否** | 必要なOwner・Host提供物・入口が各Operationで定まる。異なる役割なら共存可能 | 同一Operationの実行Ownerが重複しないことを確認できない |
| 3 | **C-3を必要なDeployment実行に適用するか**（Capability IDの宣言由来解決） | Deployment参照の固定IDを正式な選択結果と照合できる | 固定IDが意図的なReferenceか、製品Binding不足かの区別ができない |

**理由**：まず実行のOwnerをOperation／Targetごとに確定する。C-4は外部実行を委ねる正当な選択肢であり、全製品の実行方式を一律に選ぶ必要はない。実Source比較ではC-1がDeployment Command、C-2がService Requestであり、**異なる役割の共存は二重実行ではない**。ただし、同一Operation／同一Attemptを二つのOwnerが独立に実行すれば二重実行になるため、その場合は単一の実行OwnerとResult／Effect／Client Receiptの責務を決める。C-3はDeployment側の固定Capability IDの扱いであり、Service Request側へ自動適用しない。([役割の実測](../evidence/CONNECT_POINT_C1_C2_2026-11-09/README.md))

## 5. この文書で決めないこと

- 決定そのもの（採用・却下・期限）。Gate と成立状況の担当です。
- 選択肢の推薦。
- C-1〜C-3の**製品入口**への接続。参照実装は動作済みだが、通常CLI/Studio/本番Hostでの現在実行は未実証です。
- 「繋ぐべき」という前提。C-4 は現状維持として**正当な選択肢**です。

## 6. この文書が示さないこと

- **繋がないことが不安全であること**。実行を外部に委ねる設計は、Gate が「実配備先の現在権限」を別に扱っていることと整合するかもしれません。
- **C-1・C-2 の実運用資格**。両方の参照で組立て可能性は実測したが、実Host・実Provider／Grant／Receiptの安全性は未確認です。
- **材料の網羅性**。6本の記録はそれぞれ1回の観測で、`main` が進むと数は変わります。
- **どの案が Gate の G1〜G4 に効くか**。判定は Gate 本文の担当です。

## 7. 関連

- [構成Kernel移行Gate](../gate/COMPOSITION_KERNEL_MIGRATION_GATE_2026-10.md)（E0 の棚卸しと「次の一辺」が正本）
- [プラグインと配布一式](../../docs/plugins.md)（選択の規則の正本）
- [操作面の設計候補](../design/OPERATION_SURFACE_DESIGN_2026-11.md)（15章の未決4件）
- [判断材料: 操作面の未決4件と、決める順序](OPERATION_SURFACE_DECISION_BRIEF_2026-11.md)（同じ形式の先例）

## 8. HTTP Decision Graphの局所実証を、この判断にどう使うか（2026-11-09追補）

[HTTP実行設計§19〜20](../design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)には、実Reference Workspace→Canonical認可→Driver→**合成**AdapterのGo Host試験がある。さらに合成状態について、503後に適用済みの場合と未適用の場合、202後に適用済みの場合と未適用の場合、およびread失敗を、それぞれ[限定CI #42486408608](https://github.com/hat47x/tei-platform/actions/runs/42486408608)でPASSと確認している。これは**既存部品の局所結合ができる**証拠であり、製品の実CLI/Studio/Providerの接続先が決まった証拠ではない。

この実験からC-1〜C-4に対して追加できる**採否の制約**は、選択した実行Ownerが誰でも、(1)実行するAction/Provider・Current Grantの一意の出所を持つ、(2)Commit/ReceiptとHTTP Ackを混ぜない、(3)unknownの場合に非冪等Commandを安易に再送しない、(4)明示Observationを別の読取り・鮮度として記録できること、の4条件である。**条件を満たせるかはC-1〜C-4のどれについても未検証**であり、本追補でC-1/C-2を推奨したりC-4を棄却したりしない。

| C案 | HTTP実証から持ち越す最小の検証質問 |
| --- | --- |
| C-4 外部Hostへ委譲 | 外部HostがどのAuthority・Operation・Effect readの証拠を返すか。外部の確認責務をTEIが所有したことにはしない |
| C-1 Router | Routerの決定した宣言IDが実Executor/Adapterの呼出しへ到達し、HTTPへの返却とEffect Evidenceを同一Attemptで対応付けるか |
| C-2 Service Host | Host-pinned generationと実Provider/Artifact/Scopeが、失効後の別Attemptに流用されないか |
| C-3 Capability ID導出 | 名称一致ではなく実Canonically selected Binding/実Invoke先を比較できるか |

**本追補の判定**は依然「判断材料・未確定」。実行Owner（C-4を含む）→役割別C-1/C-2→必要なDeployment C-3の順序と正本のOwnerは変えない。追加のHTTP専用Routing/ResolverやProduct schemaは、実行接続の選択前に導入しない。[mainへ統合済みPR #3540](https://github.com/hat47x/tei-platform/pull/3540)により、通常`tei inspect`のdispatchと通常/生成Studio Hostから、**コンパイル済みの静的Reference Provider `tei.reference.inspector.local`**を既存Compositionで選んで呼ぶ**S2-reference**は実装済み（exact-head Health/Wails/CLI等success）。一方、**正式ProfileでProvider Bを選んだときに両Consumerが実Provider Bへ到達するS2-product**、別ProviderのLoad/Link/Grant/BOMは未達。Go Helperの成功を正式Pluginの実行資格へ昇格しない。

## 9. C-1参照実行後の不確定性をHTTP/Command側の実証と結ぶ（2026-11-09）

[PR #3573](https://github.com/hat47x/tei-platform/pull/3573)は、[C-1の実測](../evidence/BINDING_TO_EXECUTION_REFERENCE_2026-11-09/README.md)として既存の`routing.RouteCommand→Router.Dispatch→ValidateResult`を用い、Environment Bindingから**Host登録の記録用Executor**まで到達することを参照実装で確認した。この**技術的成立**をC-1の製品採用、実配備、外部Effect、Current Grantの確定へは昇格しない。実行Owner（C-4を含む）→Operation種別ごとのC-1/C-2→Deploymentに必要なC-3という判断順序を維持する。

この新しい参照をHTTP側[HDG-E01〜E05](../design/HTTP_RESPONSE_DECISION_GRAPH_DESIGN.md)と照合すると、v0 `bindingexecution.Run`に具体的な情報喪失がある。**実Executor呼出し後**に`Dispatch`がエラーを返した場合や`ValidateResult`が不正な返答を検出した場合にも、v0は`refusedCount`を増やしていた。しかし、拒否＝未呼出しとして読むConsumerでは、**既に作用した可能性を隠す**。

対策の最小差分は既存`Router`とHost所有のExecutor registryをそのまま使い、`Execute`入口への到達をtest-only witnessで観測すること。Reference report v1では`ranCount`を**Executor.Execute入場**、`refusedCount`を**入場前停止**、`unverifiedCount`を**入場後エラー・不正応答（ranCountの内数）**と分ける。 `executorInvoked`も入場事実だけであり、Receipt、Host Authority、実外部更新を表さない。[参照実装の仕様](../../tools/reference-binding-to-execution/README.md)と[限定反例テスト](../../tools/reference-binding-to-execution/bindingexecution_effect_unknown_test.go)が対応する。**[限定Go CI #42505232348](https://github.com/hat47x/tei-platform/actions/runs/42505232348)で本v1コードのBXE-01〜06と既存RouterテストはPASS**（Go vet/gofmtもPASS）。この事実は参照実装の局所合格であり、製品実行C-1採用や実外部Effectの受理ではない。

| 反例 | v0で生じた誤読 | v1の独立Expected／残る未資格 |
| --- | --- | --- |
| Executorが`Supports`拒否 | `refusedCount=1` | 入場0、refused1。これだけは実行前停止 |
| 実Executorが状態変更後エラー | `refusedCount=1`として未実行に見える | 入場1、refused0、unverified1。**作用有無はunknown** |
| Executorが更新後に別IDの応答を返す | Validate失敗で同じくrefused扱い | 入場1、refused0、unverified1。妥当なACKとして受理しない |
| ExecutorがAcceptedを返すが状態変更なし | `ranCount=1` | 入場1・妥当ACK1でも実Effectは証明されない |

**上流の実行接続Gateに追加する条件**：選択するC-1/C-2/C-4のどれでも、実行前拒否と呼出し後不確定を別々に伝えること。再試行・取消・補償の判断は既存#1174/#2999のOperation identity・Effect fence／現在Authorityに戻し、`EXECUTOR_NOT_REGISTERED`とPost-Invoke Failureを一つの「再試行可能エラー」にまとめない。CLIがJSONで拒否を報告して**終了コード0**を返すという現状のReference挙動も、無人Harnessの成功判定へ無条件に流用しない。

この追補も**C-1採用の決定ではない**。CLI通常dispatch／実Studio BridgeのPlugin-first Inspector S2、実Provider/Artifact、実DB Receipt/Observation、Client Reactionの全実装と検証は各Ownerへ残す。

## 10. 実行記録による追記（2026-11-09）

- C-1 は、既存の `routing.RouteCommand`・`routing.Router.Dispatch`・`routing.ValidateResult` だけで Binding から実行まで繋げられることを、参照実装で確かめました（[記録](../evidence/BINDING_TO_EXECUTION_REFERENCE_2026-11-09/README.md)）。
- C-2 も、既存の `BindHostApplicationEnvironmentRoute`・`composition.NewSelectedCoordinator`・`NewSelectedServiceCarrierInvoker` を順に呼んで試験の外でも組み立てられることを確かめました（[記録](../evidence/SERVICE_CARRIER_CONNECT_REFERENCE_2026-11-09/README.md)）。
- **両方とも、新しい解決器や選択の仕組みを作らずに動きました。** しかし、これは「繋げられる」ことの証明であって、「繋ぐべき」ことの証明ではありません。
- C-1 と C-2 は対象が違います（配備の動作と Service の要求）。第 4 節の「両方を選ぶと 2 つの経路ができる」は、同じ対象を取り合う場合の話です（[記録](../evidence/CONNECT_POINT_C1_C2_2026-11-09/README.md)）。
- **C-3 も、既存の3関数だけで成立することを確かめました**（[記録](../evidence/EXECUTOR_ID_RESOLUTION_REFERENCE_2026-11-09/README.md)）。`routing.RouteCommand` で Binding から executor の ID を解決し、`pruntime.BuildPlan` の plan がその ID を認めるかと `Capability.SupportsAction` を順に確かめます。実lockで `RESOLVED`、lock無しで `LOCK_REJECTED`、別のactionでは直書きの `migration.atlas` ではなく `runtime.reference` が解決され、**参照実装の直書きが Binding と食い違いうる**ことを示しました。第4節の順序（実行Owner→役割ごとのC-1/C-2→DeploymentのC-3）と、決定のOwnerは変わりません。
- **C-3 には実行側の盲点があります。** `DeploymentExecutor` アダプタは `Plan` と直書きの `CapabilityID` しか持たず、`Environment` を持たないため **Binding を照合できません**。`routing.RouteCommand` も `ValidateResult` も一度も呼びません（[記録](../evidence/ADAPTER_BINDING_BLINDSPOT_2026-11-09/README.md)）。非試験の4つの構築箇所はどれも `Dispatch` を経由せず `adapter.Execute` を直接呼ぶので、`Dispatch` の検証を強化しても届きません。**C-3 の参照は「解決できる」ことを示しましたが、実行経路はその解決結果を使っていません。** アダプタを変えずに呼出し側で一致を確かめる形は参照として試してあり（[記録](../evidence/BINDING_GUARDED_EXECUTOR_REFERENCE_2026-11-09/README.md)）、**本物の `BuildPlan`・`BuildLock`・`DeploymentExecutor` アダプタと組合わせても、guard が拒否すればその executor は一度も呼ばれない**ことを確認しています（[記録](../evidence/GUARD_WITH_REAL_ADAPTER_2026-11-09/README.md)。同じ試験の後半でアダプタを直接呼ぶと呼べてしまうため、**guard が唯一の防御**です）。**どれを採るかは未決です。**
- **版の訂正**：C-1は[PR #3596](https://github.com/hat47x/tei-platform/pull/3596)のReference v1で`refusedCount`と`unverifiedCount`の分離を実Go CIで合格済み。C-2も実コードではCarrier入場後の不確定を分離していたが、`Schema`と`RuleFingerprint`がv0のまま、READMEはv1という不整合が残っていた。C-2も明示的なv1へ更新して版・テスト・CIを独立に確定する（下記§11）。

## 11. 参照の版と実行主体を一致させる最小受入（2026-11-09）

C-1とC-2が**別種のOperationを運ぶ**ことが、[型・Carrierの独立照合](../evidence/CONNECT_POINT_C1_C2_2026-11-09/README.md)と、C-1/C-2それぞれの参照実行で確認された。したがって「実行経路は必ず一種類に統一する」とは決めない。同一Operation・同一Attemptについて、Ownerの一意性とProvider/Result/Effectの責務境界を問う。

| Reference | 実行入口として観測するもの | versionと区分 | 未認定のこと |
| --- | --- | --- | --- |
| C-1 `reference-binding-to-execution` | 実`Executor.Execute`への到達 | `v1`：入場前`refusedCount`、入場後`unverifiedCount`（`ranCount`内数）。[PR #3596](https://github.com/hat47x/tei-platform/pull/3596)のCI completed/success | 外部DB/クラウド実配備、完全なReceipt、現在Authority |
| C-2 `reference-service-carrier-connect` | Host-pinned Service Carrierの`Invoke`への到達 | `v1`：Carrier入場前`refusedCount`、Carrier入場後`unverifiedCount`（`ranCount`内数）。Provider Processの実呼出しは**別証拠** | 実ProviderのProc到達、Artifact/Host Scope現在資格、外部Effect、Client受領 |
| C-3 `reference-executor-id-resolution` | Binding が選ぶ executor の `CapabilityID` の解決と、plan への受入 | `v0`：`RESOLVED`／`BINDING_MISSING`／`NOT_IN_PLAN`／`ACTION_UNSUPPORTED`／`LOCK_REJECTED` を区別 | 実配備、Artifact信頼、実行時Effect |

**C-3 の `v0` は、C-1/C-2 の `v1` に遅れているのではなく、対象が違います。** C-3 の参照は `routing.RouteCommand`・`pruntime.BuildPlan`・`Capability.SupportsAction` だけを呼び、**Executor も Carrier も一度も実行しません**（`executor.Execute` も `Invoker.Invoke` も呼びません）。したがって「入場前の拒否」と「入場後の不確定」という区別は、C-3 には**適用できません**。C-3 の5つの状態（`RESOLVED`／`BINDING_MISSING`／`NOT_IN_PLAN`／`ACTION_UNSUPPORTED`／`LOCK_REJECTED`）は、すべて**実行前**の判定です。C-3 に `unverifiedCount` を足すべきなのは、将来この参照が実行を伴うようになったときです。

C-2の既存実装では、内部のArtifact Verifierが呼出しを拒否した場合、**Carrierへ入場していてもProcessは呼ばれていない**ことがある。この場合もC-2の`carrierInvoked=true`を「実Provider呼出し済み」としない。内部エラーとGeneration不一致を単なる事前拒否に平坦化しないため、v1で不確定応答の理由コードも事前拒否と区別する。C-2 v0の保存済み実測JSON／SHA-256は変更しない。

**限定Exit達成**：[C-1/C-2専用Go CI #42555954660](https://github.com/hat47x/tei-platform/actions/runs/42555954660)は**completed/success**。C-1/C-2両ReferenceのGoテスト、Go vet、gofmtを同じcheckoutで実行し、C-2実コードBlob `f2952030906b80ba0bf052caeed92082f89c302c`と新V1テストBlob `67b4ed10eaaf0842e2719d5cfda724076fbc5008`を記録した。コード・テストは後続の文書改訂でも同一Blobを保持。これはReference限定であり全体Health、実Service Process資格、製品S2とは別。C-4外部Host委譲の判断、通常CLI/Studioの**S2-reference統合済み／S2-product正式Profile差替え未実証**、外部Source/BOM/Effectの資格はそれぞれ別のOwnerが保持する。Referenceのレポート種別・数値を、製品の業務成功KPIや自動再試行判断へ流用しない。

## 12. 宣言の6様式を「読取・解決・実行」と区別する（2026-11-09）

[実測PR #3757](https://github.com/hat47x/tei-platform/pull/3757)は、Environment内の6種類の宣言について、[raw Source・実Caller調査](../evidence/BINDING_FORMS_CONSUMER_SURVEY_2026-11-09/README.md)を`main 0837a6f4`に固定している。[同HEADの全体健全性CI #42606374129](https://github.com/hat47x/tei-platform/actions/runs/42606374129)は**completed/success**。この調査は実行したプログラムのTraceではなく、Caller・Sourceを列挙した**read-onlyの時点付き観測**である。

| 宣言の様式 | 非試験Sourceで確認できた段階 | 実行・受領として足りない一辺 |
| --- | --- | --- |
| `DeploymentExecutors` | `routing.RouteCommand`の宣言解決、C-1の限定Reference `Router.Dispatch`で合成Executor入場 | **製品**Deployment Owner、実Executor/Artifact/Current Grant、外部Commit/Receipt |
| `ObservationProviders` | `observation.Binding`を通常`cmd/tei -plan`から呼ぶ | 非試験の`observation.Router`生成・`Observe`の実Provider実呼出し・取得時点/Scope |
| `EvidenceProviders` | `evidence.Binding`を通常`cmd/tei -plan`から呼ぶ | 非試験の`evidence.Router`生成・`Collect`の実Provider実呼出し・開示Scope |
| `ResourceBindings` | `resource.MatchBindings`による独立解決が存在 | 選択Resourceが実Consumerへ渡された同AttemptのTraceと外部変更/失効 |
| `Instances` | 解析・索引・検証の実読者に加え、**`ResolveContractQueryExecutor` が Instance を解決し、製品の2経路が実行器の選択まで進む**（[記録](../evidence/INSTANCES_BINDINGS_RESOLVER_2026-11-09/README.md)） | ただし解決されるのは **Contract Query の1用途**。Instance 一般の解決器があるとは言えない |
| `Bindings` | 同上。**Binding は Instance と Capability と束ねて解決され、実行器の provider 一致まで確かめられる**（[記録](../evidence/INSTANCES_BINDINGS_RESOLVER_2026-11-09/README.md)） | 同上。**Contract Query の1用途**であり、Application の実体対応一般を実行採用へ昇格させない |

**重要な訂正**：`ObservationProviders`と`EvidenceProviders`にはすでに`Binding`解決器があり、通常`cmd/tei`の`-plan`から**実呼出し**している。したがって「C-3と同種の解決器を新設する」は重複。反面、調査版では`observation.Router{}`／`evidence.Router{}`の**非試験組立ては0件**で、`Observe`／`Collect`が製品入口から実行できるとの受入証拠はない。`ResourceBindings`は同じ6種類に数えるが、Observation/Evidenceと同一な意味のProvider選択とは限らないので、集計件数だけで判断しない。

### P0の接続判断とテスト入力（この節ではNOT RUN）

**最初に「実行が必要なOperation/Targetか、外部Hostに委ねるか」を決める**。C-4外部委譲も候補に残し、`Observe`／`Collect`を無条件に製品CLIへ追加しない。製品内で動かす必要がある場合のみ、既存`Binding`から既存`Router`へ到達する**一つの実Consumer**で、次の独立Expectedを検証する。

| Gate | 反例と独立Expected | 不足した場合 |
| --- | --- | --- |
| CON-12a | `Provider A`がBinding解決済でも、実Observe/Collectに到達していない | `resolved`まで、`invoked=not-observed` |
| CON-12b | `Provider B`を選択したのに、誤ってAが実Invokeされる | 出力が同一でも**不一致として拒否**。静的S2-referenceの成功では代用しない |
| CON-12c | 同じProvider IDでもArtifact digest・Host/Grant epoch・Source Scopeが変わる | 旧Bindingの`loaded`・`received`・`disclosable`を再利用しない |
| CON-12d | `Collect`や`Observe`の一部だけ成功し、呼出し後に失敗／失効 | `partial/unknown`を保持し、HTTP 202やAgent Tool結果から完全取得・業務Effectを推測しない |

C-1/C-2の`ran/refused/unverified`は**別Operationの参照計数**である。ObserverやEvidence Providerへその名称を機械的にコピーするのではなく、実際に選んだOperationの既存Result・Evidence境界を使用する。UI（SUI／Studio）・SHOのGraph/Table/Traceへは`Source/Binding resolved`、`selected`、`actual invoked`、`received`、`current authorization`を別列で渡す。新Registry/第二Owner/追加Trust正本は設けない。

### CON-12のうち、いま何が済んでいるか（2026-11-09追補）

上の表は「この節ではNOT RUN」でした。その後、Observation と Evidence について**参照実装で4段を実際に通しました**。どのGateが済みで、どれが済んでいないかを分けます。

| Gate | 状態 | 根拠 |
| --- | --- | --- |
| CON-12a `resolved` と `invoked` を分ける | **済み** | 両参照で、未登録でも `resolved=true` のまま `invoked=false` になることを実行で確認（[観測](../evidence/OBSERVATION_ROUTER_CONNECT_REFERENCE_2026-11-09/README.md)、[証拠](../evidence/EVIDENCE_ROUTER_CONNECT_REFERENCE_2026-11-09/README.md)） |
| CON-12b 選ばれていないProviderを実行しない | **済み（合成Provider）。ただし製品アダプタ側の identity 検査に不具合があり、2件を修正した** | 両参照のテストで、宣言が選んだAだけが入場し、登録済みのBは0回、IDが食い違うProviderも入場前に拒否されることを確認。**参照ではなく製品コード側**で、`internal/plugin/runtime/adapters.go` の観測アダプタと配備実行アダプタが `out.ProviderID`／`out.ExecutorID` を無条件に上書きしており、`observation.Router` と `routing.ValidateResult` の同一性検査が到達不能だった（[観測](../evidence/ADAPTER_IDENTITY_MASKING_2026-11-09/README.md)、[配備実行](../evidence/EXECUTOR_ADAPTER_IDENTITY_2026-11-09/README.md)）。空のときだけ補う形へ修正し、修正前へ戻すと回帰試験が失敗することを実測した |
| CON-12c Artifact digest・Grant epoch・Source Scopeが変わったら旧Bindingを再利用しない | **両層で済み（Grant epoch のみ未）** | 観測と証拠の宣言は `Kind` と `Provider` しか持たず、比較する項が無い（[記録](../evidence/BINDING_IDENTITY_LAYERS_2026-11-09/README.md)）。識別子を持つ `generation.Binding` では、変更後の新規取得が新しい digest、変更前の lease が古い digest のままになることを実行で確認（[記録](../evidence/DIGEST_CHANGE_REUSE_REFERENCE_2026-11-09/README.md)）。`ServiceProviderSelection` の層も確認（[記録](../evidence/SELECTION_DIGEST_BOUNDARY_2026-11-09/README.md)）。route の Selection が Binding と digest/contract で食い違う場合、`servicehost` が Provider を呼ぶ前に止め、`ranCount=1`／`unverifiedCount=1`（**拒否ではない**。carrier には入っている）となる。**`Source Scope` は `servicehost` の比較項ではなく、`composition` の `validateSelectionLocked` が `Acquire` ごとに照合する**（`p.Scope != s.Scope`）。scope 不一致・registry revision のずれは、どちらも carrier へ入る前に拒否される（[記録](../evidence/CON12C_SCOPE_LAYERS_2026-11-09/README.md)）。**Grant epoch はどの層にも無い**（`Selection.Epoch` は registry epoch で、Grant の epoch ではない） |
| CON-12d 一部だけ成功した場合に partial/unknown を保つ | **partial も unknown も済み** | 両参照に `attemptedCount` と `complete` を足し、1/1 と 1/2 を件数から区別できない問題を直した（[記録](../evidence/PARTIAL_RUN_VERDICT_2026-11-09/README.md)）。さらに、入場後の失敗を `refusedCount` に数えると「実行していない」と読めるため `unknownCount`（`invokedCount` の内数）として分けた（[記録](../evidence/UNKNOWN_VS_REFUSED_2026-11-09/README.md)）。`complete=true` は「全部に答えが返った」意味だけで世界の一致の証明ではない。**失効そのものの検出は未** |

**CON-12a と CON-12b が済んだことは、「製品CLIへ繋ぐべき」の証明ではありません。** Provider はすべて合成で、外部状態を読んでいません。CON-12c は観測・証拠の宣言に比較項が無いため**その2層では問えませんが**、識別子を持つ `generation.Binding` の層では満たされることを確認しました。`ServiceProviderSelection` の層も確認済みです。CON-12d は **partial も unknown も表せるようになりましたが、失効そのものの検出は未**です。
