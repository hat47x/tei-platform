# Lifecycle Graph Compiler Contract：不変条件・可変性・生成物・実行証拠の最小設計（2026-11-08）

**状態：Draft/設計検証用。Graph Compiler、正規化Graph、DSL、Runtime、VUI/Chat Targetは未実装・未受理。** 本文は[横断設計](CROSS_TARGET_HOOKS_INTERACTION_FLOW_HANDOFF_2026-11.md)と[研究整理](../research/LIFECYCLE_GRAPH_VARIABILITY_AND_AGENT_CHAT_REFERENCES_2026-11.md)を実装可能な最小契約へ下ろす補足であり、第二のContract/Resolver/Workflow Host/Driver/Plan SSOTではない。#3503がFit/Gapを担当し、既存ADR-0048/#1158/#951/#1016/#1121/#3278/#1180/#2514/#259へ成果を返す。**既に完了した#2090の限定Extension Point PoCは維持し、Lifecycle Graph拡張を完了扱いしない。**

## 1. 最初に固定するのはLifecycleの「図形」ではなく「保証」

- **不変（全Target・全Graph Patch）**：受理済みBusiness Meaning、型/NULL/Absent、選択されたProvider identityと実呼出し、現在のHost Grant、操作対象と版、Effect/Observation/Unknownの区別、Source/Plugin/Generator/Artifact/Scenario Revision、失敗時の正確なResult、Disclosure境界。守れない場合は拒否/未検証。
- **条件付き保証（Operation/Target/Scope依存）**：write前のApproval、監査/再試行/冪等性、Timeout、Human Handoff、Secret表示抑制、長期SessionのMigration。read-onlyや非同期等の別契約へ無条件で同一Hookを強制しない。
- **可変（型付きPatch）**：前処理/検証Rule、条件分岐、サブFlow、Presentation、Agent Tool接続、Voice/Chat Event解釈、Adapter、並行処理、失敗後の補償候補。可変だからといって実Authority/Effectを置換できるわけではない。
- **分離した状態**：Interaction（入力・確認・中断）、Business（業務上の確定・履歴）、Execution/Effect（実呼出し・Commit/Unknown）。必要に応じSession/Run Stateを追加するが、万能状態表/常駐Engineは要求しない。

## 2. Lifecycle Graphの最小中間表現（既存標準GraphへのRoleとして試験）

| Role | 最小情報 | 主な検証 |
| --- | --- | --- |
| Graph | qualified ID、source revision、scope、entry/exit、graph kind、contract refs | 正本/版の混同、空Graph、Target適合 |
| Node | stable ID、kind、typed input/output、effect class、Provider/Function ref、source span | 未解決Provider、型、Effect昇格 |
| Edge | stable ID、source/target ID、event/guard、order/causal dependency、source span | 不正遷移、未到達、欠落、複数辺Identity |
| Region | subflow、並行/選択/待機、join/termination、timeout | 並列競合、合流不能、resume |
| Obligation | predicate ID、scope、affected effect/target、required evidence | Guard迂回、認可の偽装、判定不能 |
| Patch | provider identity、point/target ID、operation、precondition、version、dependency、source span | 非可換競合、古い版への誤適用 |
| Projection | diagram layout/view revision、representation binding、loss/roundtrip capability | Layout変更と意味変更の分離 |
| Compilation Receipt | base/patch/expanded graph digests、provider lock、generator revision、artifact revision、validation verdict | 古いArtifact/Provider混入 |

Graph IDとEdge IDは表示ラベルに依存させない。同名ノード・同名多重辺・向きが異なる辺も識別できること。Graphviz/Mermaidは表示/記述Projectionであり、構造欠落を伴う表現を自動的に双方向編集可能としない。図面編集はSource/Revision付きの型付きPatchへ変換して初めて意味変更となる。座標変更だけならSemantic Digestは変えない。

## 3. Graph Patchと合成：曖昧な優先順位を設けない

1. **既存Resolver**がPlugin/ProviderとTarget Capabilityを選択し、正確なVersion/Source/Lockを得る。Plugin未選択を欠陥と即断しない。
2. 選択済みPatchのprecondition（対象ID/contract revision）を検査。対象消失、旧版、型/Effect不一致なら拒否。
3. Patch依存グラフで順序を決定する。独立・可換と証明できる変更のみ順序自由。複数replace、同一Edgeへの非可換move/branch等は**競合として失敗**。任意の「最後に登録したもの勝ち」は禁止。
4. Graph合成結果を正規化し、Graph差分とPatch適用履歴を保存。合成後のGraph/ObligationをTarget-neutral PlanへLoweringする。全Targetへ共通Runtime Engineを挿入する必要はない。
5. Graph全経路の必須Guard/Effectの到達性、型、状態・Eventの完全性を検査。解析で保証できないものはUNKNOWN/要実証であり、静的PASSを付けない。
6. 実生成後、Provider B選択時に実InvokerがBか、実Target/Hostが指定契約を満たすか、独立Expectedを再確認。生成成功は実呼出し成功ではない。

**適用演算子の候補**：add-node/edge、branch、wrap-subflow、bind-provider、add-event/transition、add-guard。remove/move/parallelは、保証・競合・取消しの検証能力を確認してから段階的に許可する。無制限なGraph書換えAPIは初期段階で提供しない。

## 4. ライフサイクル自体の拡張：構成時と実行中を分ける

- **Build-time composition**：標準Graph＋選択Plugin Patch→展開Graph→検証→生成物。既存Go単一バイナリやTarget-native生成を優先し、TEI Hostは不要。
- **Run-time choice**：受理済みGraphの状態・Event・Guardに従い、事前に許可されたProvider/Operationから次の処理を選択。非決定的LLM出力はCandidate Eventのまま扱い、確定型付きEventとCurrent Authorityを通す。
- **Run-time graph mutation**：新しいGraphやPluginを提案しただけでは即時実行不可。別RevisionのProposal→意味受理/Policy→Build/Lock→配備→旧Session継続または明示Migration。
- **Re-entry/Resume**：Flow開始時のGraph/Plugin Lock/State versionと、resume時の現在権限を別々に検査。旧版Flowを新Graphへ暗黙移行しない。過去のConfirm/Tool Approvalを新入力・対象へ流用しない。
- **Failure boundary**：取消しや表示ストリーム切断は既に確定した外部Effectの取消しを保証しない。Commit unknownの再送は独立Operation identity/receipt/冪等契約に従う。

## 5. 異質なTargetへの共通Graph適用

| Target/Profile | 固有Graph Role | 共有Operationへの接続 | 禁止する省略 |
| --- | --- | --- | --- |
| CLI | structured event / prompt / exit | typed Operation input/result | CLIだけ業務制約を簡略化 |
| Web/Desktop | UI event / session / presentation | 同一Operation・Target Adapter | Rendererを権限の正本化 |
| MCP | tool discovery / elicitation / response | 同一Operation、別Host/Client scope | Tool annotation=実行許可 |
| VUI | ASR candidate / turn-taking / interruption / TTS | confirmed event→Flow→Operation | partial recognition=confirmed event |
| AI Web Chat | thread / turn / run / stream / tool proposal / approval / branch | finalized tool intent→Operation | partial argsの実行、再生成時のEffect再送 |
| B2C Headless | journey / experiment / exposure / checkout intent | route/componentを独立実装しても同一Operation | 標準DOM/CSS/Router/TEI Hostの強制 |

B2Cにおける高度なUX、SEO、性能、実験、Headlessのescape hatchは#3509をOwnerとし、GraphはBusiness Journey/Interaction/Experimentの選択肢を支えるが、標準Rendererの採用を強制しない。生成AI Web ChatではStream deltaをPresentation側へ分離し、Tool確定/Approval/Effectと混同しない。Voiceの音声品質やBrowserの表示品質はGraph/Operation parityと別のTarget Conformanceとする。

## 6. 最小縦断スライス（既存P0の順序を崩さない）

| Gate | 作業 | 独立Expected・成果物 | 失敗条件 |
| --- | --- | --- | --- |
| L0 | mainのOperation/Inspector/Plugin/Scenarioをread-only inventory。Graph RoleへFit/Gap | 実Source位置、受理版、未取得/NULL/部分応答、必要Role | 既存Contractの二重定義 |
| L1 | 同一Inspector Providerを通常CLI/Studioから実呼出し（#1251） | selected Provider B = actual invoked B、CLI/Studio受領、BOM/Source/Host | B選択→A実呼出し、未選択Pluginを全体失敗扱い |
| L2 | 既存Operation一件をBase Graphへ写し、validate addとbranchの2 Patch | Graph前後差分、Guard/型、正規化Graph、Conflict negative | 権限/Effect経路迂回、Source位置喪失 |
| L3 | Graph→Target-neutral Plan→一Targetの実コード生成・実呼出し | selected Binding、Artifact revision、actual Outcome/Effect | 図面だけ/静的PASSだけを実Target PASS扱い |
| L4 | 構造化EventのInteraction FlowをCLIと第二Consumerへ | Driver Scenario SSOT、独立Expected、欠落/重複/取消/unknown | Flow差分をBusiness Meaningへ黙示昇格 |
| L5 | Voice/Chat異質性、B2C HeadlessをProfileとして比較 | partial event/approval/stream/branch/escape負例、Target別Evidence | 未実装ASR/LLM/BrowserをPARITY PASS |
| L6 | Graph editorでNode/Edge変更→Sourceへ反映→再生成→実Target | Roundtrip/Source/Plugin/Build/Invocation連鎖、layout-only負例 | 描画座標をBusiness Sourceへ混入 |

L1が既存Plugin-firstの重要な未完ゲートであり、L2以降の研究のためにL1を迂回しない。L2〜L6は新たな必須製品Gateではなく、#3503のFit/Gapから既存Ownerへ引渡す実証候補。

## 7. テスト行列（Scenario SSOTへ返却する）

| ID | 入力/故障注入 | 期待 |
| --- | --- | --- |
| LG-01 | 同じBase/Lock/Patch、入力順のみ変える | 可換な変更のGraph/Plan digest一致 |
| LG-02 | 同じpointへ二つのreplace | conflict、生成拒否 |
| LG-03 | patchで必須Authorization Guardを迂回 | 静的拒否、実Effectなし |
| LG-04 | sourceの同名Node/多重辺/Edge向き変更 | Identity/意味差分を保持 |
| LG-05 | layoutのみ変更 | Semantic Graph/生成Artifact不変 |
| LG-06 | Source→Graph→Diagram edit→Source | 構造とSource位置のRoundtrip、意味受理を別判定 |
| LG-07 | Provider B選択→A実呼出し | 結果同値でも不合格 |
| LG-08 | Hook/Generator/Plugin版だけ更新、A→B→A | 再資格化とEvidence失効、古い合格の復活なし |
| LG-09 | old Flow実行中にGraph更新・resume | 明示継続/移行/拒否、現在権限を再評価 |
| LG-10 | Chat partial tool args / Stream切断 | 実Commandなし、Commit結果を捏造しない |
| LG-11 | Chat regenerate/branch、旧Effectあり | 旧Tool副作用を無条件再送しない |
| LG-12 | Voice曖昧発話/確認対象変更 | 未確定Eventのまま、再確認 |
| LG-13 | B2C custom frontendが標準Rendererを使わない | typed Operationは維持、UI/SEO/性能は別資格 |
| LG-14 | 実Targetの必須Flow/Representation step欠落 | Target Coverage incomplete、CLI PASSから外挿しない |
| LG-15 | 未選択Pluginに故障、選択Pluginは正常 | 無関係な未選択Pluginで対象Consumerを過剰停止しない |
| LG-16 | TargetでSecret/非開示対象を表示・読み上げ | disclosure/Host境界で拒否、UIの成功を認可としない |

テストSSOTは#1180、Independent Expectedは#2514。各ケースの実装/実行状態は未実証。CI通過を先取りして表記しない。

## 8. Ownerと進行停止条件

- #3503：Lifecycle Graph/Interaction Flow/Voice/ChatのFit/Gapと最小Reference。
- #2090（closed）：既存限定Hook契約の成果を再利用。Graph Patch拡張は別作業であり、完了扱いしない。
- #1158 / ADR-0048：共通OperationとTransport Adapter。
- #951/#1016/#3278/#1121：既存Provider/Lock/Trait/Lowering。第二Resolver/Generator ABIを作らない。
- #997/#2497/#3303：Function-owned Representationと媒体別Projection。Diagram編集のread/write capabilityを明示。
- #1125/#1129/#1174：Business State/Effect/権限とInteractionの境界。
- #1180/#2514/#259：Scenario/Expected/Target Conformance。
- #3509：B2C Headless/Conversion Engineeringの非干渉。

**停止条件**：第二Host/Registry/Driver/Authorityの必要性を無批判に前提化する、GraphからSourceへの安全な帰還がない、必須Guardを任意Pluginが除去できる、Provider選択と実呼出しの違いを隠す、未受理AI提案を実行可能へ昇格、対象Targetを実行していないのにPARITY PASSを付ける。これらが現れた場合は構造を拡張する前に既存Ownerの契約へ差し戻す。

## 9. 横断レビュー追補：Obligationの強度・実装の非干渉（2026-11-08）

- **Obligationの種類**：必須のHost/Effect/Accepted Meaning保証と、Operation/Target/Profileが要求する条件付き保証を区別。Obligationを追加するPluginと、必須保証を弱めるPluginは同じ拡張権限を持たない。静的に判定不能な経路は実行時Guardの実証を要求し、未提供TargetはUNKNOWN/UNSUPPORTED。全read-only経路にwrite承認を強制しない。
- **Graph/Diagram同時編集**：Graph Node/Edge IDとSource revisionに基づく楽観的競合検出を行い、表示名や座標でIdentityを推測しない。図面のみ編集とScriptのみ編集で、同じGraphの正規化結果・再生成結果を比較する。layout-onlyとsemantic patchの失効を分ける。
- **Effectと再開**：resume/branch/regenerateは新Operation identityの発行と旧Effectの再実行を同義にしない。Graph definition revision、Session revision、Business Entity revision、Effect receipt、Plugin/Generator Lockを独立に記録する。長期稼働TargetのMigrationは#1223へ引き渡す。
- **B2C非干渉**：#3509のHeadless/Custom Frontendで、TEIが関与しないRouteに追加JS/DOM/HTTP hopを要求しない。Graphは標準Rendererの強制手段ではなく、Operation/Scenario/契約の投影として利用できる。
- **候補Patchの初期範囲**：add/branch/wrap/bind/eventを優先し、remove/move/任意runtime graph rewriteは独立ExpectedとOwnerの安全保証が揃うまで制限する。Pluginの登録順やUI図面上の配置で意味順序を決めない。
- **反例の追加観点**：B2Cの未使用Plugin故障非干渉、Chat partial tool args→実Call 0回、Diagram layout-only→Semantic/Build資格不変、Graph Edge変更→Source/Owner再受理、Plugin版A→B→A→過去のEvidence自動復活禁止をLGケースの期待に含める。新たな第二テストSSOTは作らない。

**補足**：本節は既存LG-01〜16に検証観点を追加するもので、番号体系や実装完了を変更しない。実装担当は#3503/#2090/#1124/#1180/#1223/#3509の既存Ownerに返す。

## 10. 実装前の最小縦断Fixtureと「何を検証すれば十分か」（2026-11-08）

上記のL2/L3を大規模Graph Compiler構築へ直結させない。**同一Operation・同一Expectedを用いる3構成**で、必要な最小IRとPlugin拡張の費用対効果を比較する。

| Fixture | 構成 | 必須Expected |
| --- | --- | --- |
| F0 Baseline | 既存の受理済みOperationとProvider Bを、Graphに投影するだけ。実行は既存の経路 | Graphを導入しても既存のselected B→actual B・Outcome・Effectが変わらない |
| F1 Validation extension | 同一Operationへ、既存Contractに適合する追加Ruleを一つGraph Patchで挿入 | 正常・拒否・NULL/Absentの独立Expectedが維持され、Ruleが実際に呼ばれる |
| F2 Approval branch | 条件付き承認、Wait/Resume、再検証を追加（承認の業務意味はOwnerの受理を要する） | 対象/版/Grantの変更時は旧承認を再利用せず、unknown commitを再実行しない |

**独立Expectedの原則**：期待結果はGraph Compilerや生成物から逆算しない。既存のOperation/Rule/Effect契約と独立Fixtureを基準とし、生成したGraphの形だけ一致してもPASSにしない。

### 10.1 最小のデータ交換境界

Graph自体は既存の標準Graphデータ構造を再利用する候補とし、Generatorへ渡すものは、少なくとも以下の**意味情報を失わない読み取り専用のPlan View**とする。新しい永続テーブルや第二正本を前提にしない。

~~~text
GraphPlanView {
  source_ref, accepted_meaning_ref, graph_revision,
  selected_plugin_lock, target_capabilities,
  nodes: [stable_id, role, contract_ref, typed_ports, effect_class, source_span],
  edges: [stable_id, from, to, event, guard_ref, dependency],
  obligations: [id, scope, required_predicate, evidence_requirement],
  bindings: [contract_ref, selected_provider, provider_version],
  patch_lineage, normalization_digest
}
~~~

これは**契約確認用の擬似スキーマ**であり、TEI DSL/Go struct/JSON Schemaの採用を意味しない。Source→Graph→Plan→Artifactのどの段階でも、表示名や図面レイアウトを意味IDに代用しない。

### 10.2 合成と検証の最小アルゴリズム

1. SourceとOwner受理範囲を確定し、既存Resolverが選択したPlugin Lockを取得する。
2. Patchの対象stable ID・expected revision・依存を照合し、非可換競合を拒否する。
3. 型付きGraphを正規化し、同じ意味入力なら同じdigestになることを検証する。
4. Effectful Operationへ至る経路のObligationを検査し、静的に判定不能な部分はTargetで強制される実行時検査へ明示的に引き渡す。
5. 既存LoweringへPlan Viewを渡し、対象Targetで選択Providerの**実Invocation**と結果/Receiptを観測する。
6. Diagram/DSLで同一変更を行った場合の意味Graph一致を確認する。Layout変更だけの場合はSemantic digestが不変となる。

**失敗時の契約**：競合・未解決Provider・Graph意味欠落・Guard迂回・実行時検査未提供は、黙ってベース処理にフォールバックせず診断して停止する。既存CLIや未選択Pluginまで巻き添えにして停止する必要はない。

### 10.3 横断的な採否判断

- **継続**：F0〜F2でSource/Graph/Provider/Targetの証拠を一連で結べ、既存Contractを二重化せずに2種類のPatchを合成できる。
- **限定導入**：静的合成だけで成立する場合、Interpreterや常駐Workflow Hostを導入せず、Graphをビルド時の構成入力に限定する。
- **方式見直し**：図面からの編集帰還に意味欠落がある、既存Resolverを複製する、Graph導入によって未選択Pluginの障害が波及する、または実Provider呼出しが証明できない。
- **未判定**：AI Chat/VUIの長期セッションやB2C実験配信は、F0〜F2だけで実装適合を宣言しない。別Target/Profileの追加検証に進む。

このFixtureは既存L0〜L6/LG-01〜16を具体化する**補助入力**であり、新しいGateや第二のScenario SSOTではない。
