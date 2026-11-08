# 実行可能Lifecycle Graph・Plugin可変性・AI Web Chatの横断研究（2026-11-08）

- **状態**：外部の論文・仕様・実装文書を参照した設計研究。TEIへの意味受理、正式Schema、Graph Compiler、VUI/Chat実Target、実行適合は **NOT ACCEPTED / NOT RUN**。
- **横断設計の引渡し先**：[CROSS_TARGET_HOOKS_INTERACTION_FLOW_HANDOFF_2026-11.md](../design/CROSS_TARGET_HOOKS_INTERACTION_FLOW_HANDOFF_2026-11.md)、#3503/#2090/#1158/#1121/#1180。新しい設計研究の正本として既存Ownerを上書きしない。
- **目的**：Lifecycle/Hookを先に全体設計し、不変条件と可変構成を区別しつつ、Lifecycleグラフ自体をPluginが合成・拡張し、その型付き図表を**コード生成の入力**とする。VoiceとAI Web Chatを異質なConsumerとして検証する。

## 1. 学術・規格・実践から得られる設計上の根拠

| 参考資料 | 読み取れる設計上の価値 | TEIへ導入する前の留保 |
| --- | --- | --- |
| David Harel, **Statecharts: A Visual Formalism for Complex Systems** (Science of Computer Programming, 1987; doi:10.1016/0167-6423(87)90035-9) https://weizmann.elsevierpure.com/en/publications/statecharts-a-visual-formalism-for-complex-systems/ | 状態の階層、直交する並行状態、イベントを図で構造化する。Flowを単純な一列の前後Hookから脱却させる | Statechartsの全仕様をCoreに固定する必要はない |
| **W3C SCXML 1.0** (Recommendation, 2015) https://www.w3.org/TR/scxml/ | state/transition/guard、entry/exit、parallel/history、invoke、拡張actionを規格上区別。プラットフォーム拡張は任意の直接状態変更でなくイベントを介する | SCXML XMLまたは全実行意味の直接採用を要件にしない |
| van der Aalstほか, **Workflow Patterns** (MIT Press, 2016) https://mitpress.mit.edu/9780262029827/workflow-patterns/ | 制御・データ・Resourceの反復パターンを、特定のEngine/Notationに依存せず整理 | TEIが最初から全パターン対応するという主張ではない |
| van der Aalst/ter Hofstede, **YAWL: yet another workflow language** (Information Systems, 2005; doi:10.1016/j.is.2004.02.002) https://www.sciencedirect.com/science/article/pii/S0306437904000304 | 複数インスタンス・合流・非局所取消しは単純な制御フローで表現しづらい。形式意味と人のモデル化コストを合わせて評価 | 全FlowをPetri NetへLoweringする案を先に確定しない |
| Hallerbach/Bauer/Reichert, **Capturing variability in business process models: the Provop approach** (2010; doi:10.1002/smr.491) https://onlinelibrary.wiley.com/doi/10.1002/smr.491 | Base Processと差分操作の分離。挿入/削除/移動/変更を使う変種モデルは、案件固有の差分を重複なく保持できる | 削除/移動を無制限に許すと必須GuardやHook IDを破壊する |
| van der Aalstほか, **Preserving correctness during business process model configuration** (2010; doi:10.1007/s00165-009-0112-0) https://link.springer.com/article/10.1007/s00165-009-0112-0 | 変種の設定後も「正しさ」を保存・検証する問題を明示。コンフィグ可能と安全に構成可能は別 | 検証対象・判定可否を固定Fixtureで絞り込み、一般的正しさを約束しない |
| Ehrigほか, **Overview of Formal Concepts for Model Transformations Based on Typed Attributed Graph Transformation** (2006; doi:10.1016/j.entcs.2006.01.011) https://www.sciencedirect.com/science/article/pii/S1571066106001356 | 型付き属性Graph、適用条件・制約・変換の構文/動作/意味の正しさを区別できる | 重い汎用Graph Rewrite Runtimeを要求せず、限定差分を先に検証 |
| **OMG BPMN 2.0.2** (2014) https://www.omg.org/spec/BPMN/ | 意味モデルと図面レイアウトを別仕様（BPMN/DI）として記述。実行可能プロセスのモデル交換が可能 | 規格の全構文・イベント・Engine適合をTEIに固定しない |
| **Temporal Workflow & Workflow Versioning** https://docs.temporal.io/workflows , https://docs.temporal.io/workflow-definition | Event Historyからの決定的Replay、外部効果をActivityへ隔離。長期実行中のDefinition更新にはVersioningが必要 | TEI独自常駐Workflow HostやTemporal依存を導入する話ではない |
| **LangGraph Interrupt / Graph API** https://github.com/langchain-ai/docs/blob/main/src/oss/langgraph/interrupts.mdx , https://github.com/langchain-ai/docs/blob/main/src/oss/langgraph/graph-api.mdx | Human-in-the-loop時のcheckpoint/resume、node再実行と副作用重複の問題を具体化 | Pause=rollback/Exactly-onceではなく、Host Effect fencingは独立 |
| **AG-UI Events / Overview** https://github.com/ag-ui-protocol/ag-ui/blob/main/docs/concepts/events.mdx , https://github.com/ag-ui-protocol/ag-ui/blob/main/docs/introduction.mdx | WebチャットでRun/Message/Tool/State/Activity等を増分Eventとして配信する、双方向の対話Transport/Projection | AG-UIを共通業務Operationや権限の正本にはしない |
| **A2UI Protocol / Google introduction** https://github.com/a2ui-project/a2ui/blob/main/specification/v1_0/docs/a2ui_protocol.md , https://developers.googleblog.com/introducing-a2ui-an-open-project-for-agent-driven-interfaces/ | Agentが信頼済みUI Component Catalogへ宣言的にUI構成を渡す。Modelが自由なスクリプトを直接UI実行しない | A2UIコンポーネント定義をTEI標準表現型と同一視せずMappingの損失を検証 |
| **OpenAI Responses Streaming** https://developers.openai.com/api/docs/guides/streaming-responses | response.created/delta/completed/errorのような型付き増分Eventが重要。部分テキストやTool引数途中データは確定値と違う | Provider固有Eventの直採用でTEI Coreを肥大化しない |
| **Vercel AI SDK UI Stream Protocol** https://github.com/vercel/ai/blob/main/content/docs/04-ai-sdk-ui/50-stream-protocol.mdx | Messageの開始・部分デルタ・終了、Tool等の増分PresentationがWeb chatで重要 | SSE等のwire形式をFlow Business Meaningの正本にはしない |

上記は**参考先の仕様/研究の要点**と**TEIへの設計推論**を列で分離する。実装方式の採否・性能・標準準拠は本書では主張しない。

## 2. Lifecycle自体をPluginで構成する設計

従来の固定列（prepare→validate→execute→observe→result）を**単一の不可変順番**とみなさず、標準のLifecycleを **型付きGraph Template（Base Lifecycle）＋不変条件集合（Obligations）＋可変のPlugin差分（Extensions）** として表現する。

**不変なのはNode列の形ではなく、操作の意味・型・Authority・Effectなど、通過した全経路が満たすべき保証**。例えば：
- 未受理SourceをAccepted Meaningへ自動昇格しない。
- 実Operation Execute時に、対象/入力/期待版に適用される必須ValidationとHostの現行Authorizationが成立している（入口での表示だけでは足りない）。
- Resource/Secret/Disclosureの条件を満たさない経路は、AI/CLI/Voice/別Transportでも拒否する。
- Command受付、Business Outcome、Effect known-applied、独立Observation、Unknownを分離する。
- 生成されていない必須Operation/Edge、未解決Provider、Unsupported Targetは黙ってskipせず診断する。
- Hookの型・版・Source/Build帰属、実呼出し先を追跡できる。

**可変**：Guardの提供者、検証のRule集合、前後の追加処理、業務上の確認/承認、イベントの解釈、表示、並行実行、timeout/compensation、TargetごとのAdapter、追加の状態/分岐。必要なObligation自体はDomain/Contract/Targetによって変わる（例えば公開read-onlyの全経路に同一のログインUIを要求しない）。固定すべきはObligationの検査・失敗扱い。

## 3. 型付きGraphをGeneratorの第一入力にする

既存TEIのEverything is Data / Script-firstと矛盾させないため、**型付きLifecycle GraphをSourceから復元できる標準データ構造**として扱う。Graph自体が入力モデルであり、描画画像/表示座標がコード生成の直接入力ではない。

- Node: stable qualified ID、kind/契約、State/Input/Output/Effect、Provider参照、Source span。
- Edge: stable ID、source/target、event/guard、order/data/authority dependency、必要時多重性・時点。
- Flow/Subgraph: entry/exit、scope、revision、複数instance・並列領域・終了条件・timeout。
- Diagram: Graphの編集可能Projection。node/edge追加・削除・再接続・属性修正は型付きModel Patchへ変換し、Source位置・元Revision・Ownerの受理境界を保持してから再解析/再生成。レイアウトのみの変更はBusiness MeaningやGenerator input fingerprintを変えない。
- Text/Markdown表/Mermaid/Graphviz: 同じGraphへの各Syntax/Projection。構造情報を失うRendererを無条件に双方向編集可能と広告しない。**読める図**と**正本へ安全に書戻せる図**は別Conformance。

~~~text
Accepted Source / typed Lifecycle Graph @ R
        + selected Plugin Extension Patches @ L
        + Hook/Provider Contract & Fulfillment
                      │
         typed compose / conflict check
                      ▼
         Expanded Lifecycle Graph @ R,L
                      │
       Path / Authority / Effect invariants
                      ▼
       Target-neutral Execution/Interaction Plan
                 /   |    |     \
            CLI    Web   MCP     VUI / Chat
                 Target Code Generation
                      │
           Artifact / Actual Invocation Evidence
~~~

**Semantic patchの例（候補・実DSL未採用）**：
- add before/after node（追加）
- wrap fragment with subflow / error boundary（範囲限定の包摂）
- branch if condition（型付きGuardを持つ分岐）
- extend on event / add transition（Interaction eventの追加）
- bind/replace implementation（契約適合Provider選択）
- add parallel child / join（合流条件・effect条件が検証できるときだけ）
- remove/move（必要なObligation/Edgeを維持できる場合に限り許可。初回Referenceでは禁止候補）

Patchの対象は表示名ではなくStable ID/ContractとRevisionへ束縛。Plugin同士の競合、非可換な複数Patch、削除された参照ID、循環、到達不能、型不一致、必須Guard迂回、未定義合流/不正なresumeは、生成時に失敗かUNRESOLVEDとする。

## 4. コンパイル・適合の順序（Generator中心）

1. **Source/Acceptance確認**：Graphの原Source/Revision、受理済みMeaning/対象Scopeを識別。Diagram-onlyの構成候補は自動受理しない。
2. **Contract/Provider解決**：既存#951/#1016で、Hook point、Extension Trait/Target要求、exact selected ProviderとVersionを決定。
3. **Graph Patch適用**：Preconditionとstable IDを確認し、明示した依存順で決定的合成。競合は黙って後勝ちにしない。適用結果の差分とPatch lineageを残す。
4. **静的なObligation検証**：入力/出力型、Entry/Exit・必須Nodeの到達、対象Effectへ至る**全経路**上の必須Guard（経路依存の要件）、取消/timeout/中断、parallel join、欠落Provider/Grant、無限loopの検出可能部分、resume時期とState互換をチェック。一般の停止性を証明済みと偽らず、判定不能はUNKNOWN。
5. **Lowering / Emit**：共通型付きPlanからGo static binary/target-native Web, Desktop, CLI, MCP, Voice/Chat Pluginへ生成。Runtime Engineが必須とはしない。OperationとVoice/Webの外側Presentation/Transportを別で生成。
6. **Conformance**：同じScenario、独立Expected、Graph Obligation/Coverage、Target生成digest、actual selected Provider invocation、実Result/Effect/Receiptを別判定。図面表示成功や静的検証PASSだけでは実Targetの適合を認めない。

**Graph保証の参考検査**：構造検査→型/Contract適合→必須Guardを通過しない副作用経路が無いこと→正常終端/拒否/取消/unknownの経路閉包→並行部分の競合と合流→実Scenario/Mutant。高度なPetri Net解析/Model Checkingは該当Consumer/Patternが必要になった場合のProviderとして導入を比較する。

## 5. Lifecycle-of-Lifecycle：変更が自らを検証する

Lifecycle自体のPlugin拡張には、**構成時（静的）**と**実行中（動的）**の2種を明確に分ける。

- **構成時**：受理済みSourceとExtension Patchを合成し、静的Obligation検査→生成→実Evidenceを実施。新Generator Hook/Flow Branchは原則こちら。
- **実行中**：事前に選択された契約/Capability内でイベント・状態・次Operationが切り替わる。AIが未審査の生コード/新権限/任意Graphを直接実行しない。未定義のGraph変更はProposal→Owner/Policy審査→新Revision/Lock→再生成/配備→旧Execution継続/明示移行。
- **稼働中の版変更**：running Flowは開始時のGraph/Plugin Lock/Policy Referenceを記録。新Graphでresumeしてよいか、旧Graphを使うか、明示Migrationが必要かを区別。現在権限はresume時に再評価。Temporalの歴史Replay互換性を参考にするが、TEIへ常駐Event Logを義務付けない。

この3段階を混ぜない：**Pluginが新しいLifecycleを定義可能**／**現在のモデルに適用可能**／**稼働中の実行へ安全に適用可能**は別資格。

## 6. AI Web Chatを普通の対話Flowに押し込まない

AI ChatはVUI/Webの単なる見た目ではなく、Run・複数Turn・Streaming・Tool Call・Approval・再生成/分岐・Agent間委譲がある**新しいInteraction Profile**。ただし新しい業務Operation SSOTは作らない。

| Chat特有のモデル要素 | 対応する共通責務 |
| --- | --- |
| Conversation / Thread / Turn / Run | Interaction StateとRun lifecycle。恒久保存範囲はTarget/Host選択 |
| Message / Content Part / Attachment | Function-owned Representation/Disclosure/Source。同一本文の部分表示と確定版を区別 |
| Streaming delta / thinking/progress activity | Presentationの増分Patch。文字・音声ChunkごとにCanonical Business Stateを変更しない |
| Model proposal / tool argument partial | untrusted Candidate。最終Tool Intentへ昇格前に型/対象/版/範囲検証 |
| Tool Call / Tool Result | 既存Application Operationへの型付きBinding。Toolだけの裏口Auth禁止 |
| User approval / human handoff / interrupt | Interaction Flowの状態/Guard。承認対象と提示内容のDigest/版を束縛 |
| Agent planning / subagent / variable branches | 許可済みContract内の動的Flow選択。未知Graphを自動配備しない |
| Regenerate / Edit / Branch / Resume | Run/Turn/Graph/Plugin/State Revisionと観測済みEffectの別管理 |
| AI generated form/card/dashboard | Presentation/Renderer Plugin（A2UI等のAdapter候補）。UI宣言の存在≠業務SourceのAcceptance |
| Partial/cancelled stream / transport loss | streaming表示中断とBusiness Operationのabort/unknownを明確に分ける |

### Web Chatの最小受入Scenario

~~~text
user prompt -> Candidate Intent
  -> model streams partial text (presentation-only)
  -> model proposes tool call + incomplete args
  -> finalized typed Tool Intent -> verify subject/scope/args/revision
  -> sensitive write requires current approval on exact payload
  -> shared Application Operation -> effect/receipt/read-back
  -> final response/message/event stream
~~~

- **C01**：partial tool argsを受信しても実Operationを呼ばない。
- **C02**：生成済みUI Cardを押してもHost検証なしでwriteしない。
- **C03**：tool結果未知/通信切断時に「成功」メッセージや自動再送をしない。
- **C04**：Chat regenerate/branchで旧tool effectを再実行しない。Chat本文変更とBusiness data変更を別に扱う。
- **C05**：二つのTool Callの並行実行/同一対象の変更競合で、明示Context/依存/Lockを検証する。
- **C06**：agentが「新Pluginを入れて」と言っても未選択Provider/新権限/未受理Flowを実ロードしない。Proposal/検証後の別Revisionとして扱う。
- **C07**：Web/CLI/MCP/Voiceの実Targetで同じ業務OperationとOperation outcomeを比較し、媒体固有stream/UI/voice品質は別Evidenceにする。

Web Chatのために「Agent専用業務Engine」を作らず、AG-UIはstream/state通信、A2UIは安全な宣言UI、MCPはTool/API境界、LLM Providerは候補生成としてそれぞれ交換可能にする。どのProtocolも単独でTEIのAuthority/Accepted Meaning/Effectを保証しない。

## 7. 次に実装検証する順序と停止条件

**研究だけで新Hard Gateにしない**。次は既存#1251/#1158で進むselected Provider実呼出し検証を再利用する。

1. **R0 Graph inventory**：現行のOperation/Driver/Plugin Hook構造から、Lifecycle Node/Edge/Obligation/Source位置に写せるものを一覧化。最低限一Operation＋一Interaction/Chat Flowで比較し、欠落を隠さない。
2. **R1 Base + Patch proof**：カスタムValidation追加/Branch追加の2例を型付きGraphとして宣言し、差分適用→Guard到達/非到達→Target-neutral Planを検証。Generator実コードは既存#1121を可能な範囲で再利用。
3. **R2 Two-consumer real call**：同じOperationをCLI/StudioまたはCLI/Webから実呼び出し、B選択→A呼出しやGuard bypassを独立Expectedで拒否。
4. **R3 Chat & Voice contrast**：Voice confirmationとWeb Chatのpartial tool args/approval/interruptを同じGraph言語へ載せ、表現できない差は媒体固有Role/Providerに分離。実ASR/LLM/Hostは別資格。
5. **R4 Graph edit to actual build**：EditorでNode/Edge追加/再接続→Sourceへ反映→受理→再生成→実Target invocationまでの一本の証拠。layout-only変更でBusiness/Buildを不必要に再生成しない。元Source/Graph ID/Plugin Lock/Artifact/Operationを同一Attemptで照合。

**停止する条件**：第二Resolver/Host・新Agent Runtimeがないと動かない抽象、意味Graphから元Sourceに安全に戻れないDiagram、Graph PatchでAuth/Effect Guardをバイパスする構成、予測不能な登録順、実未呼出しProviderの成功偽装、旧版中断Flowの無審査resume。一般的な形式的完全性は狭いFixture以外で証明済みとしない。

## 8. 誰が何を引き取るか

- **#3503**：Lifecycle Graph/Interaction FlowとChat/Voice異質ケースのFit/Gap・新しいグラフ/Flow仕様の提案。
- **#2090**：Hook Contract、Graph PatchでのExtension Bindingと再生成保持・break検出。新Graph Engineは所有しない。
- **#951/#1016/#1121**：existing Provider composition / Target-neutral LoweringへGraph入力とPlugin選択を接続。
- **#1124/#2620/#2497/#3303**：Graph/表/テキスト/Studioの編集帰還・identity/情報損失の検証。編集画面の表示状態を意味正本へしない。
- **#1158/#2093/#997**：操作共通契約とMCP/Web/Chat/VoiceのAdapter・Renderer。Generated業務ApplicationとDeveloper Studioを区別。
- **#1125/#1129/#1174**：業務Procedure/State、commit/unknown/副作用を本設計へ接続。
- **#1180/#2514/#259**：独立Expected、Graph mutation反例、Scenario、実Target証拠。R0〜R4を完了条件の追加ではなく必要な担当へ段階引渡し。

**未決**：Coreの最小Graph型（標準Graph既存型へのrole追加のみでよいか）、Graph patch記法/拘束力、実行Graphの静的生成とRuntime Interpreterの切分け、AI ChatのRun StateとBusiness Workflowの境界、正式なSCXML/BPMN互換範囲。実Consumer/Target/Source証拠を得るまで固定しない。
