# Lifecycle Graph Contract / Composition / Generator 最小Reference（2026-11-08）

**状態**：設計・反例・実証引渡し。正式TEI Graph Schema、DSL、Compiler、Target、Owner受理は NOT ACCEPTED / NOT RUN。既存#3503のGraph研究、#2090 Hook、#1121 Lowering、#1180 Driverの**横断受入候補**であり、第二のRuntime/Workflow/Parser/Registry/Hostを作らない。

## 1. 設計の精緻化：同じ図面から全媒体を作る、ではない

TEI標準Data/Graphに意味Roleを与え、**同じSource/Accepted Meaningを共有するが、複数のLifecycle Profileが存在する**方式とする。

- **Application Operation**：型付きIntent→入力検査→必要なAuthority→Effect→Outcome/Observation。CLI/HTTP/MCP/Web/Desktop/VUI/Chatが同一契約を呼ぶ。
- **Interaction**：入力候補→確定Event→State/Guard/Transition→必要ならOperation→Presentation→Wait/Resume/End。
- **Generation**：Source/Provider/Plugin Lock→Graph Compose→検証→Lowering→Emit→Artifact/Target適合。
- **Release/Experiment**（B2C #3509等）：Route/Journey/Flag/Exposure/Conversionを持つ別Profile。収益UXを共通Rendererへ強制しない。
- **Agent/Chat**：Conversation/Turn/Run/Stream/Tool/Approval/Branchを持つInteraction Profile。AIのRunをBusiness OperationやEffectの代わりにしない。

「全Profileに同じNode/Edge enumを押し込む」のでなく、Graph共通のIdentity/Type/Relation/Source/Revision/Contractを持ち、意味RoleとObligationはProfile/Pluginが提供する。図表/テキストはGraphのProjectionであり、表示座標は実行意味ではない。

## 2. 最小型付きGraph契約（提案。採用Schemaではない）

| 要素 | 最低限の識別・資格 | 意味 |
| --- | --- | --- |
| Graph | qualified ID、profile、revision、accepted meaning reference、entry/exit、scope | 一つのLifecycle Definition |
| Node | stable ID、kind/role、contract ref、source span、read/write/effect classification | Action/Guard/Wait/Present/Invoke/Subflow等。種類はPlugin Roleで追加 |
| Edge | stable ID、from/to、event/guard、priority/dependency、branch/exception/timeout分類 | 遷移。多重辺・向き・edge identityを維持 |
| Port | typed input/output、absent/null、failure、capability、visibility | Operation/Nodeの型付き境界 |
| Patch | ID、target graph/node/edge、expected revision、op、pre/postcondition、owner/source | 差分。選択PluginとLockに帰属 |
| Obligation | ID、適用範囲、path predicate、required guard/effect rule、severity | 不変条件を単一固定処理列でなく検証可能な制約として宣言 |
| Binding | extension point/contract/selected provider/target capability、version | 実装選択。選択と実Invocationは別証拠 |
| Execution Ref | definition revision、instance identity、state version、operation/effect identity | resume/branch/replay時の資格 |

**重要な非同一性**：Graph DefinitionのVersion、図面Layout Version、Plugin/Generator Lock、Business Entity Version、Interaction Session Version、Execution/Effect Receiptは別物。ID/型一致だけで業務意味受理済みとしない。SourceがA→B→Aへ戻っても過去の受理/ビルド/実行資格を無条件再利用しない。

## 3. Plugin Graph Patchの合成意味

Patchを登録順・図面上の左から右で適用しない。**明示依存のDAGと安定ID**から適用順を求め、非可換な競合をfail-closedとする。独立なPatchは順序が違っても正規化GraphとExpectedが一致することを検証する。

初回の許可候補：Add node/edge、Branch on typed guard、Wrap with named subflow、Bind/Replace implementation、Extend event/transition。Remove/Move、任意のruntime graph rewrite、既存Guard/Authority/Effect Obligationの削除は最初のReferenceで禁止する。必要が実証された場合に、対象範囲/事前条件/移行手順を付けて解禁を検討する。

Patchの検証順：source/revision precondition→contract/type/capability→patch conflict→graph reachability→path obligations→lowering feasibility→target conformance。PluginがObligation自体を提供できるが、**必須Obligationを弱めるPluginの選択権限**と**新しい検査を追加するPlugin**は区別する。Policyの適用条件はOwner/Hostの既存権限契約に従い、任意Pluginの自己宣言で安全境界を解除しない。

## 4. 全経路のObligationをどう扱うか

「全経路で同じ順序のAuthorize Nodeを必ず通る」は過剰な固定化となる。代わりにEffectful Node/Operationの直前に、**現在の対象・入力・版・Subjectに有効なGrant/Validationが成立**していることを静的/動的に保証する。静的に判定不能なら実行時Guard/検査証拠を要求し、未提供Targetはunsupported/unknownとして生成・配備を止める。

- **Safety**：未確定Intent/partial tool args/未受理GraphからEffectへ到達しない。必須Guardの迂回がない。
- **Liveness（条件付き）**：cancel/timeout/unknownを含む出口が定義され、並行合流に不正な待ちがない。外部応答待ちの有限時間終了は一般に保証できない。
- **Effect integrity**：operation identity/receipt/unknownを保存し、replay/resume/regenerate/voice interruptで二重Effectを作らない。
- **Disclosure**：音声読み上げ、Chat streaming、MCP Tool Result、Web Cardは同じHostの表示許可/Scopeに従う。
- **Provenance**：Graph→Patch→Provider→Plan→Artifact→actual Invocation→Observationの出典を結び、Accepted Meaningと各Evidenceを混同しない。

## 5. 代表的なカスタマイズ：図面編集→実生成

標準Operation「注文確定」を、一定金額以上の場合に人の承認を待つFlowで包む。業務意味に関わる承認条件はOwnerの受理対象。

~~~text
[Collect Order] -> [Validate] -> <Amount threshold?>
                                | no          | yes
                                v             v
                           [Execute]    [Request Approval]
                                |             |
                                |        [Wait / Resume]
                                |             |
                                |        [Revalidate current data
                                |         and authorization]
                                +-------> [Execute]
                                             |
                                         [Observe]
~~~

図面のノード/辺編集はStable ID付きPatchへ変換し、Source RevisionとOwner Scopeを確認してから採用。差分を元のTEI構造化Script/標準Graphへ戻し、Source→Graph→Lowering→Targetへ生成。UI座標変更はLayoutだけ、承認条件/Edge変更はMeaning/Build/Scenarioへ影響。図面とDSLの同時編集は競合を検出し、暗黙の最終書込勝ちにしない。

**比較する実現方式**：
1. **静的合成・コード生成**：標準。対象Native artifact内にFlowの構造をLoweringし、追加Host/Graph Interpreterを要求しない。
2. **事前選択された実行時Event/Provider分岐**：Chat/Voice等の長期対話に必要な場合。生成された状態機械/Adapter内で動的Eventを処理する。
3. **外部プロセス/Interpreter**：本当に長期継続・熱更新・隔離が必要なTargetだけPluginで選択。単一Go binaryを全案件で犠牲にしない。

## 6. AI Chat / Voice / B2Cにおける圧力

| Profile | 最小独自Event | 共通Graphの何を試すか |
| --- | --- | --- |
| CLI/HTTP | Request, Error, Result | 同一Operation Input/Failure/Outcome |
| Web/Desktop | UI Action, Navigation, Focus/Cancel | Presentationから対象/Identityが失われないか |
| MCP | Tool Discovery, Call, Elicitation | Tool annotationとHost Authorityの非同一性 |
| VUI | Partial ASR, Final Intent, Interrupt, Silence, Confirm | 候補と確定Event、確認版、resume |
| AI Web Chat | Turn, Run, Stream Delta, Tool Partial/Final, Approval, Regenerate/Branch | 部分生成のEffect誤昇格、再実行/branchの二重Effect |
| B2C | Experiment Assignment, Exposure, Route, Conversion | UI自由度、CV計測と実Effectの区別、表示変更の独立再配備 |

B2C #3509の「TEI非関与Routeでは追加JS/DOM/API hopゼロを許す」条件を保つ。AI ChatでAG-UI/A2UI等を使う場合もAdapter候補であり、Protocol/Agent RuntimeをCoreの必須依存にしない。ChatのMessage/Run StateをBusiness Entity Stateへ直接同一化しない。

## 7. 受入マトリクス：独立Expectedを先に置く

| ID | 入力/変化 | Expected | Evidence |
| --- | --- | --- | --- |
| LG-01 | 同じBase/Patch/Lockを順不同に宣言 | 依存が同じなら正規化Graph/Plan一致、衝突は拒否 | Graph digest/差分/診断 |
| LG-02 | 必須Auth/Validationを迂回するEdge追加 | Static rejectまたはTargetに強制Runtime Guardを要求。未対応はfail | Path obligation/Target proof |
| LG-03 | 二Pluginが同一Nodeをreplace | 一意に解決不可ならfail closed | Contract conflict |
| LG-04 | Diagramの座標のみ変更 | Business Meaning/Code Buildの資格は変えない | semantic/layout revision差分 |
| LG-05 | DiagramでEdge変更 | SourceへPatch反映、再受理対象を示し、実Artifact/Scenario更新 | Source span/Graph/Build |
| LG-06 | 同じ型/IDでOwner未受理の意味変更 | Acceptanceを継承しない | Owner scope/meaning evidence |
| LG-07 | Plugin/Generator版だけ変更・A→B→A | 再利用資格と実Provider呼出しを別判定 | Lock/build/invocation |
| LG-08 | Flow中断中にGraph改版 | 旧版継続/明示Migration/拒否を区別、権限は再評価 | instance/revision/receipt |
| LG-09 | AI Chatのpartial tool args/stream delta | Execute 0回、表示だけ進む | event trace/operation call count |
| LG-10 | Chat regenerate/branch/Voice interrupt after effect | 旧Effectを暗黙再実行しない。unknownはunknown | operation identity/receipt |
| LG-11 | Web/CLI/VUI/MCPでTargetが一Guard欠落 | Cross-target parity fail/unsupported | per-target coverage |
| LG-12 | B2C専用Frontendをheadlessで利用 | TEI由来の追加Browser Runtime/hop/DOM強制なし | bundle/network/DOM inspection |

**初回の縦切り**：LG-01/02/03/04/05/09/10を独立Expectedとして固定し、既存order_entry等一Operation＋承認Flow＋Chat partial Tool Eventを構造化Fixtureで実証する。VUIの実音声やB2C本番CVRは独立した後続Gateとする。実コードとTargetの存在がない場合は NOT RUN。結果値が同じでもProvider違い/Guard欠落をPASSにしない。

## 8. 既存Ownerと作業順

- **#3503**：Profile/Graph/Interaction/Chat/VoiceのFit/Gap、LG-01〜12を束ねる窓口。新EngineのOwnerではない。
- **#2090**：Extension Point/Graph PatchのContract・再生成・衝突、LG-01〜03。
- **#1124/#2620**：三視点Modeler/標準GraphのNode/Edge IDと図面編集→Source帰還、LG-04〜06。
- **#951/#1016/#1121/#3278**：選択Provider/Lock/Graph Compose/Target-neutral Lowering、LG-01/07。
- **#1158/#2093/#997**：同一Operation/Transport/Presentation、Chat/MCP/VUI Adapter、LG-09〜11。
- **#1180/#2514/#259**：Driver Scenario、独立Expected、実TargetのCoverage/Negative、LG-01〜12。
- **#1223**：継続稼働TargetのDefinition/Plugin版変更・Migrationの適合、LG-08。
- **#3509**：B2C Headless/Experiment/Journey、LG-12。新Browser Runtimeは非目標。
- **#1174/#940/#1043**：Current Authority/Effect unknown/operation identityの境界。独立Effect検証の代用をしない。

**順序**：read-only Fit/Gap→Graphの最小型とObligation/2 Patchの固定Fixture→Diagram編集のSource帰還→既存selected Providerで二Consumer実Call→Chat/Voice/B2C圧力→Target生成/適合。P0/G0〜G4を変更しない。未実証のFlow/Graph機能を「標準実装済み」と宣言しない。
