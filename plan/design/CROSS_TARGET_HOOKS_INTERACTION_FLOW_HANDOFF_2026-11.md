# API / Web / Desktop / CLI / MCP / VUI横断：生成ExtensionとInteraction Flowの統合引渡し（2026-11-08）

> > **具体的な実装引渡し**：[Lifecycle Graph Compiler Contract・最小縦断検証](LIFECYCLE_GRAPH_COMPILATION_CONTRACT_AND_REFERENCE_SLICES_2026-11.md)に、型付きGraph Role、Obligation、Patch競合規則、L0〜L6、LG-01〜16、B2C Headless/Chat/Voiceの非干渉を整理。既存#2090は完了済みの限定PoCであり、Graph拡張の実装完了を意味しない。

**2026-11-08 後続研究・設計上の修正候補**：LifecycleとHookの点の一覧を先に固定した上で後から拡張を載せるだけでは不十分。実行可能な型付きLifecycle Graph（意味Node/Edge・契約・状態・守るべきObligation）を**コード生成の直接入力**とし、PluginのGraph Patchでライフサイクル自体を拡張できる設計を最優先で検証する。以下§3の「Operation/Interaction/Generationの3系列」は**暫定的な観測/Projection**であり、不可変な共通Core enumや固定順序ではない。詳細・研究根拠・AI Web Chat対応は[Lifecycle Graph研究](../research/LIFECYCLE_GRAPH_VARIABILITY_AND_AGENT_CHAT_REFERENCES_2026-11.md)へ。

> **横断レビューの追加**：[Lifecycle Graph Compiler Contract](LIFECYCLE_GRAPH_COMPILATION_CONTRACT_AND_REFERENCE_SLICES_2026-11.md) §9に、Obligation強度と条件付き保証、Diagram/Script同時編集の競合、稼働中Graphの版・Effect、B2C Headless非干渉、Patchの初期制限を追補。既存LG-01〜16へ引き渡し、新たなGraph契約の並列正本を作らない。

- **状態**：設計候補・既存Ownerへの引渡し。新しいDSL構文、Flow Engine、Plugin ABI、Target生成器、実VUI、実音声Provider、Ownerの意味受理は **NOT ACCEPTED / NOT RUN**。この文書は第二の仕様正本・優先順位表・実装完了宣言ではない。
- **判断の基線**：[ADR-0048](../adr/0048-shared-application-api-and-transport-adapters.md)（共通型付きApplication Operation）、[ADR-0043](../adr/0043-target-native-artifacts-without-tei-host.md)（共通TEI Host不要）、[ADR-0036](../adr/0036-delivery-forms-as-plugin-composition.md)（提供形態はPlugin合成）。
- **検証Owner**：[Interaction Flow #3503](https://github.com/hat47x/tei-platform/issues/3503)。各実装のOwnerは§9を参照。
- **出発点**：[拡張点 #2090](https://github.com/hat47x/tei-platform/issues/2090)、[Plugin Provider #951](https://github.com/hat47x/tei-platform/issues/951)、[合成 #1016](https://github.com/hat47x/tei-platform/issues/1016)、[Lowering #1121](https://github.com/hat47x/tei-platform/issues/1121)、[UI Kit #997](https://github.com/hat47x/tei-platform/issues/997)、[Scenario #1180](https://github.com/hat47x/tei-platform/issues/1180)、[原子的Trait #3278](https://github.com/hat47x/tei-platform/issues/3278)。
- **既知の実装制限**：mainには共通Operationの一部実装と限定CLI/MCP validate投影があるが、全Target同値性、Hook付き実生成、選択Providerの実Invocation、VUI、Business Flowの統一実装は未証明。#1251のCLI/Studio actual selected Provider D1が先行ボトルネック。限定ReferenceのCI成功を本計画全体の成功へ外挿しない。
- **優先順位**：#1119および plan/MASTER_PLAN.md の現行P0/G0〜G4を変更しない。既存検証の再利用を優先する。

## 1. 解決すべき分裂と結論

API、Web、Desktop、CLI、MCP、VUIは同格の六つの「業務エンジン」ではない。APIは原則として通信方式の名前ではなく型付きApplication Operationの契約であり、Web/Desktop/CLI/MCP/VUIは各々異なるTransport・Presentation・Interaction・Hostを持つConsumer/Target/Adapterとなる。Developer-facing CLI/MCP/Studioと、生成された業務ApplicationのCLI/MCP/GUIは公開Scope・Subject・Authorityが異なる。

- **Operation**：何を実行するか。型付き入力、欠落/NULL、制約、Validation、業務Outcome、Effect/Observation。
- **Interaction Flow**：どの情報を集め、何を確認し、どの状態・イベント・条件でOperationに到達するか。
- **Representation / Presentation**：何をどの媒体へ投影・提示し、表示からどのIntentへ戻すか。Data→Function-owned Representation→Consumerへの写像と損失は既存 #2497/#3303 に従う。
- **Transport / Host**：argv、HTTP、IPC、MCP、Voice Stream、local/remote等の経路と真正なSubject/Scope/Grantの検証。TEI独自Hostや全入口HTTP化は要求しない。
- **Extension Binding / Generator**：受理済みSource上の拡張と既存Contract/Provider/Loweringを接続し、対象に適合する実装だけを生成する。

同じ操作は「同じAccepted Meaning、Revision、Host Policy/実Authority、Provider選択、入力/対象、Resource条件」下で比較する。全媒体のwire JSON、画面、音声タイミングの一致は要求しない。意味・不足・失敗・副作用の差は独立Evidenceで説明する。

## 2. 共通Extension Contract：フックは生成物の文字列挿入点ではない

TEI Function / Rule / FlowとTarget生成器を、**宣言された拡張点の型付きBinding**で結ぶ。生成コードへの直書き、曖昧なメソッド名・Go/Javaの継承構造、Provider登録順序を契約の正本にしない。

**契約を構成する情報（標準データ構造のRole/Relationへの候補。新Schema採用ではない）**：

| 内容 | 必須の扱い |
| --- | --- |
| Extension target / point | Function/Operation/Flow/Target、安定ID、Scope、宣言Source位置と契約版 |
| Operation | add（独立規則追加）/ transform（型付き変換）/ replace（許可されたProvider交換）/ compose（複数実装合成） |
| Contract | input/output、nullable/absent、read/write/effect、Failure、Target capability、必要なTrait/Grant |
| Binding | 拡張するTEI Function/Rule/Flow、resolved Provider identity、Generator/Implementation version |
| Ordering | before/after依存、並列可能性、競合のfail-closed、同順の暗黙採用禁止 |
| Evidence | Acceptanceとは別に、解決済みBinding、Build input、actual invoked provider、実Outcome/Observation |

**区別**：Pluginは契約を提供し、Traitは履行義務を付与し、Hook Bindingは実現先を結び、Generatorは選択された構成から成果物を作る。Contract存在、Trait宣言、Provider選択、Build内に存在、実呼出し、Host現在権限は互いの代用にならない。

## 3. フックの概念体系と許される拡張

**3系列のLifecycleは意味を分け、拡張宣言・解決・検証は共通**とする。巨大な単一Event Bus/万能Hook APIを先に作らない。

| 系列 | 最小概念点（暫定） | 可能な拡張 | 不可侵の境界 |
| --- | --- | --- | --- |
| Application Operation | prepare / validate / execute / observe / result | 型付き入力導出、Rule追加、実装Provider選択、結果投影 | 現在の認可、未観測Effect、Owner未受理意味の捏造 |
| Interaction | receive / interpret / transition / present / resume | 入力解釈Provider、Guard、遷移、表現選択 | 未確定Intent→Command、旧確認をAuthorityへ昇格 |
| Generation | resolve / bind / lower / emit / verify | Target固有Implementation/Renderer/コード生成器の交換・追加 | Accepted Meaning、Plugin Lock、Source Provenance、独立Expectedの上書き |

**Operationでの安全順序**：prepareでInput/対象が変わるなら、変更後の値をvalidateし、実際のexecute境界でCurrent Authorityを検証する。Authorizeは任意のbefore/afterフックではなく**必須の実効Guard**。実行結果のresult表現フックは実際のEffectをSUCCESSへ変える権限を持たない。observeは独立read/receiptに基づき、Command受理=Commit確定とはしない。post-commit外部副作用は独立Effectとして扱い、通常のValidation/Transaction内へ黙って割り込ませない。

**合成**：純粋なValidationの集約は規則ID順等の正規化と同値な集合結果を目指し、順序依存副作用は明示順序/故障時意味を要求。二つのreplaceが同一ポイントに競合し一意に解決できない場合は生成拒否。Hookなしは既存標準挙動と一致。拡張点が変わった場合は再生成保持だけでなく、再適合・呼出し結果を検証する（#2090）。

## 4. 人が扱う最小記法（設計例・現行DSL未対応）

想定は標準Data構造へのシンタックスシュガーであり、Goの型宣言、func、import等を日常DSLに露出させない。

~~~tei
extend sales.order.submit {
    on validate:
        add sales.creditCheck
}
~~~

~~~tei
flow sales.order.confirm {
    start: collect

    state collect {
        on orderSelected -> confirm
    }

    state confirm {
        present sales.confirmSummary
        on confirmed:
            execute sales.order.submit
            -> completed
        on rejected -> collect
        on timeout -> expired
    }

    state completed {
        present sales.orderResult
        end
    }
}
~~~

この記法をそのまま製品Syntaxへ採用しない。先に既存#988のLanguage Plugin/標準データ/Roleで表現した機械可読Referenceを用意し、専門家の可読性、Source位置・診断、Hook版と再生成、必要なTarget Consumerを比較する。TEI Function自体の業務意味を変更する場合はHook差分だけでなくOwnerのAccepted Meaningへ戻す。

## 5. VUI：異質な媒体からInteraction Flowを試す

VUIは原則 **ASR/LLM等の候補Intent → 型付き確定Event → Flow State/Guard/Transition → 共通Operation → Presentation/TTS** の接続として検証する。FlowをVUI専用音声処理や汎用Workflow Engineと同一視しない。

| 別資格のState | 例 | 必要な区別 |
| --- | --- | --- |
| Interaction State | 注文内容入力中・確認待ち・中断・再開待ち | 一時/永続、session revision、transition identity |
| Business State | 未確定・承認待ち・確定 | 業務意味とRuleのOwner、履歴/時点条件 |
| Execution/Effect State | 未呼出し・実行中・known-applied・unknown | Receipt/独立Observation、無条件再送の禁止 |

**確認の束縛**：音声「はい」は、どのOperationの、どの対象ID/対象revision、表示した重要入力/金額、どのSession遷移に対する応答かを検証してからIntentへ昇格する。発話途中の部分認識、認識競合、別人の割込み、確認対象の変更、resume後の古いSession、現在Grant失効を正常確定としない。提示内容の省略が意味に影響する場合は確認不成立。Flowは認証・認可の代替手段ではない。

**Flow評価の再現性**：Source/Flow revision、State snapshot、確定Eventと参照した外部観測を固定した決定論的な遷移評価を目標とする。音声/LLMの非決定性は候補Event/Provider Evidenceとして外へ置き、入力event列のreplayと実音声End-to-Endを別試験にする。timeout/interruptはInteraction Eventであり、既に成立した外部副作用をrollbackする証明ではない。

**非同期と保存**：短命な画面/対話Stateはメモリ保持を選択可能。長期承認や会話resumeは保存・版比較・期限・失効が必要となり得るが、単一状態テーブルや汎用Workflow Hostを必須化しない。保存の物理方式・DB Providerは別担当。EKIを使う場合もDelivery成功≠Consumer受領≠Commit/Authorityとする。

## 6. 六つの入口の対応表

| 入口・媒体 | 操作への接続 | 固有の責務 | 対象外の主張 |
| --- | --- | --- | --- |
| API（共通型付き） | 共通Operationそのもの | Typed Input/Result/Failure、Source/Version/Effect | HTTP API一律必須 |
| CLI | argv/stdinとOperation/Flow EventのBinding | Exit/Output、structured argv、ローカル実行 | CLIが業務意味の別Owner |
| MCP | tool/schemaとOperationのBinding | Tool discovery、Elicitation対応可否、Client transport、開示 | MCP Tool annotation=権限 |
| Web | Action/HTTP AdapterとOperation/Flow | Form/Renderer、Session/CSRF/Origin、ブラウザー体験 | Web画面成功=独立Effect証明 |
| Desktop | UI/IPC・プロセス内呼出し | OS Device/Window、WebView隔離、renderer入力検査 | Studio Wails=生成業務Desktop合格 |
| VUI | Speech Intent Event/Flow + Operation | ASR/TTS、割込み・無音・音声認識Confidence、確認提示 | Speech認識=確定Intent/Authority |

MCPはDeveloper向け#3400/#1158と生成Application向け#2093を分ける。Voice Providerのエンジン固有コマンドは音声媒体側に保持し、共通Operationへ流入させない。

## 7. 並行Plugin領域：実装依存と設計のみを分ける

| 優先 | 領域 | 既存Owner / 次の検証 | 新規基盤を避ける点 |
| --- | --- | --- | --- |
| P0-Fit/Gap | Extension Binding / Hook Contract | #2090 + #951 + #1121、標準validate RuleのTEI Function追加 | 第二Registry/Generator ABI |
| P0-Fit/Gap | Interaction Flow / Event Binding | #3503 + #1125/#1129、構造化CLI Eventでの二段階確認 | 第二Workflow Engine |
| P0既存依存 | 共通Operation・Consumer Provider実呼出し | #1158/#1251、CLI/Studio同一selected Provider | transport固有ビジネス処理 |
| P1並行 | Presentation/Renderer/Voice Binding | #997/#2497/#3303、Web/CLI/Voiceで同じSubject/Intent | UI/音声の万能Renderer |
| P1並行 | Scenario/Conformance Generator | #1180/#2514/#259、Flow＋OperationのCoverage差 | 新しいTest SSOT |
| P1研究 | Input Interpretation/音声Provider | #3503 の媒体圧力、Partial Event→confirmed Eventを分離 | LLM判断による意味・許可の確定 |
| P2要求時 | Session persistence、Scheduler/Event、Notification、外部Service | #1125/#1129、#1174、EKI/DB Ownerと能力・責務接続 | 全案件に常駐Workflow/Message Host |
| P2要求時 | Accessibility / Localization / Device | #997/#259、媒体固有Evidence | 見た目を意味の正本とすること |

DB/Schema Access/SQL方言/Transaction実装は本ドキュメントの設計範囲外。Presentation/Voiceを加えたことで、これらに第二のData Provider契約を作らない。

## 8. 段階的実証と独立Expected

**既存P0優先は維持**。先行するread-only API/Inspector Provider実呼出しを犠牲に、Flow/音声を直ちに必須製品依存にしない。

| Gate | 実施内容 | 合格に必要な独立証拠 | 未実証のまま残す資格 |
| --- | --- | --- | --- |
| X0 | #2090/#951/#1158の実Code/Schema/Adapter Fit/Gap。HookなしのBaseline・独立Expectedを凍結 | Source Revision、Owner、選択Binding、欠落と負例の一覧 | VUI/Flow動作 |
| X1 | 一つのTEI Custom Functionをvalidateに追加、CLI/Studio等二Consumerで型付きOutput比較 | selected Provider=actual invoked Provider、Hookの実Call、Referenceの結果/Source、差分 | 全Target同値性 |
| X2 | 構造化Event→Interaction Flow→既存OperationをCLI/もう一入口で実呼出し | State/Event/Version、Guard、Operation identity、業務Outcome、無遷移/拒否 | ASR/TTSの品質 |
| X3 | VUI候補解釈→確定Eventの接続。interrupt/resume/timeout | 音声候補/確定Event、欠落・曖昧時の拒否、現在Authority | 本番音声サービス/生体本人認証 |
| X4 | 同一保存ScenarioからTarget別Generator/Testerへ投影 | Coverage obligation、生成digest/Build/Trait/Lock、実対象でのnegative E2E | 他Target・実DB・正式Owner業務受理 |

### 必須の負例セット（独立Expected先行）

- **H01**：HookなしとHook追加の標準動作差。追加Ruleだけが差を作り、未受理Business Meaningは自動変更されない。
- **H02**：Hook A/B両方が同一Pointをreplace、または型・順序が競合。生成を拒否し、黙って片方を採用しない。
- **H03**：Provider B選択に対し実A呼出し。結果が同値でも不合格。Plugin v1→v2、A→B→Aでも旧適格性を復活させない。
- **H04**：Prepare後に対象/価格変更、古いValidate/AuthorizeのままExecuteを試みる。必須Guard未通過として拒否。
- **F01**：同一State/Event/Revision/外部ObservationでFlowを2回評価。TransitionとIntentの正規化結果一致。未完入力を補完してExecuteしない。
- **F02**：確認後に対象/金額/Session Revision変更。古いConfirmedを失効、Commit呼出し0回。
- **F03**：ASR partial/曖昧発話/無音/キャンセル/複数イベント。候補と確定Eventを混同しない。
- **F04**：実行前Grant失効、resume時の旧Session、MCP/CLIからの直接迂回。認可拒否、Operation実Call 0回。
- **F05**：Commit直後の通信断/音声中断。effect unknownを保持、無条件replay/成功読み上げ禁止。
- **F06**：Web/MCP/VUIで必須Flow Step/Representation Binding/対象Identity欠落。Coverage incomplete/unknownとなり、見かけ上の成功禁止。
- **F07**：Target生成物を手直し、Hook Sourceだけ変更、Plugin版だけ変更、生成器だけ変更。実再生成とSource/Meaning/Build/Evidence失効範囲を別票にする。
- **F08**：Speech/Presenterで秘匿項目や権限外Scopeを返す。表示/読み上げのDisclosureは現在Contextを強制し、秘密や存在数を漏らさない。

生成・静的適合・実Consumer呼出し・Transport到達・Host/Grant適合・実Effect/Observation・正式Owner受理は別のPASS/FAIL/NOT RUN軸。結果値一致のみではProvider Invocation/Voice動作/Source一致を保証しない。検証責務は#1180のDriver Scenarioと#2514のIndependent Expectedへ接続する。

## 9. 責務Ownerへの引渡し（新しい並列実装群の分裂を防ぐ）

| 担当 | この文書から渡す差分 | 先行依存 / 非目標 |
| --- | --- | --- |
| #1158 / #3419〜3422 | Hook有無でOperation Input/Outcome/Failureが変化してもTransport投影が同じ意味を保つ | validate等実read-only収束が先。汎用dispatcherを作らない |
| #2090 | Hook PointとExtension Bindingの入力/出力/契約版・再生成保持・衝突/失効診断 | Generatorの実装Owner。TEI Function自動受理はしない |
| #951/#1016/#3278 | Hook実装の型付きProvider選択とTrait Fulfillment、Source/Artifact Lock | 宣言と実呼出し別、第二Resolver禁止 |
| #1121 / #943/#2600 | Extension BindingをTarget-neutral Planと実Build入力/生成器版へ束縛 | Build再利用≠Evidence再利用 |
| #997 / #2497 / #3303 | Flow Presentationの媒体写像・Identity/Target可視性・情報損失 | VoiceをForm/DOMへ強制しない |
| #1125 / #1129 | Interaction StateとBusiness Procedure/Business State/Effectの区別 | 全業務の単一状態表、独自State Engineを先行採用しない |
| #1180 / #2514 / #259 | Flow Event列・業務OperationのCoverageとVoice/GUI実Target資格の別判定 | CLI PASSをVoice PASSとしない |
| #2093 / #3400 | 開発MCPと業務MCP別Scope、Flow追加質問への相互運用性 | MCP配備方式や確認APIを決め打ちしない |
| #3503 | 異質VUIからのF1〜F08のFit/Gapと最小実証・既存Ownerへの返却 | 新Core/ABI/Workflow Hostを所有しない |

## 10. 今回固定しないもの

- Flow/Hookの正式DSL文法、正式Schema、Hook全種類や中央Runtime構造。
- すべての媒体で同じPresentation・Transport・Stateストレージを使うこと。
- VUIのASR/TTS/LLM製品やリアルタイムプロトコル。
- 全てのHookをaround adviceで上書き可能とすること。
- 認証/認可/実行許可/意味受理をFlowまたはPluginへ委譲すること。
- 処理の順序をPlugin登録順やGenerator出力の文字列位置で決めること。
- VUIをP0 Product Gateへ前倒しし既存CLI/Provider接続を停止すること。

本計画は**既存の検証済み契約へ一段ずつ適用し、反例を満たせた範囲だけ標準候補として昇格**させる。実装がなくても概念を追加しただけで「Everything is Pluginsが成立」「全Target parity」「VUI対応済み」としない。


## 11. 2026-11-08 横断収束：図表優先・不変保証・異質なConsumer（実装前の設計候補）

**本節の目的**は、既存の他レーンが並行して作成した成果を第二の仕様正本へ複製せず、Lifecycleを構成する際の共通判断と実装順を合意可能な形にすること。詳細な受入票は各Ownerが持つ。

- [Draft #3505](https://github.com/hat47x/tei-platform/pull/3505)／RQ-036候補：全体MapからData/Function/Business/Event/State/Pattern/Rule/Provider/Targetへ辿れる図表・ナビゲーション、標準/設定可能/Hook可能/案件置換/観測のみの凡例。これは**人が全体像を読めることを最優先**する要求で、Graph Compiler完成を前提にしない。PR未統合の仕様・実装実績と混同しない。
- [main統合済み PR #3510](https://github.com/hat47x/tei-platform/pull/3510)／#3278：原子的Traitの要求集合と、合成後の実効Graphの**全経路**で成立する保証は別物。Traitを備えるPluginが存在するだけでGuard通過・実Callを保証できない。TLG-01〜08は既存#2514/#1180の独立Expectedへ帰属させる。
- [#3509](https://github.com/hat47x/tei-platform/issues/3509)：B2Cは完全Native/Headless Frontend、独自SSR/Router/UX/実験を妨げない。Journey/Experiment Graphは意味上のActionと開示/実Effectへ参照するが、画面・計測・CV推定をBusiness Completionへ昇格しない。
- [#3274](https://github.com/hat47x/tei-platform/issues/3274)：HTTP Decision/Client Reaction/Recovery Graphは**Application Outcome/Execution Effectの用途別Projection**。HTTP statusやProblem Detailsに業務意味を持たせない。#1174のunknown commitと再送拒否を守る。
- [#2497/#3303](https://github.com/hat47x/tei-platform/pull/3303)：Data D→Function-owned Representation V→媒体Wの写像は一般に非可逆。Flow Diagramの画面上のEdgeやCanvas occurrenceをSemantic Subject/Source Identityへ自動同一視しない。
- #2620/#2619（Everything is Data）、#1124（Modeler）、#1125/#1129（Business/Rule）、#951/#1016（Plugin）、#1121（Lowering）、#2600（Build鮮度）、#3271（Cache）、#3262（Context）を既存Ownerとして維持。異なるGraph型の共通データ表現は許すが、**一個の万能Graph Engine／単一の全種状態表／中心Event Bus**を義務化しない。

### 11.1 標準Lifecycleの「不変」と「可変」は分類ではなく契約の責務

不変とは、全Operationで同じNode列を置くことではない。**当該Operation/Target/Scopeに適用される受理済み契約のObligation**を、Plugin合成後も守ること。Obligation自体も業務/Effect/Targetに相対的であり、静的read-onlyまでログイン画面や事前承認を強制しない。

| Layer | 不変と扱う根拠（適用条件付き） | Plugin/案件による変更可能部分 | 未確定・禁止 |
| --- | --- | --- | --- |
| Accepted meaning | Function/業務Outcome・対象・Rule・Source/Authorityの受理範囲 | Proposal→Owner受理を経た業務変更 | 画面/LLM/Graph案からの自動受理 |
| Operation/Effect | 型、必要Validation、現在Auth、effect identity、unknown/receipt、開示 | Validator追加・Provider選択・分岐・観測実現 | Guard迂回、UnknownをSuccessへ変換 |
| Lifecycle structure | 適用するObligation、Graph/Transitionの追跡可能性 | Node/Edge/Subflow/Guard/タイマー/並行分岐 | 登録順依存、競合Patchの黙殺 |
| Presentation/Interaction | 表示からIntentへ復元するための必要ID/時点/Scope、Disclosure | Web/VUI/Chat/CLIの表示/補助対話/Native UI | 非可逆の表示値から更新対象を捏造 |
| Realization/Delivery | exact Plugin Lock、必要Target Capability、Toolchain/Source束縛 | Go/Java/.NET/Browser/native、Headless、静的/動的合成 | 全TargetへのTEI Host/JS/HTTP常駐強制 |
| Runtime/Measurement | effect/consent/Subject/Request/Resourceの現在資格、独立Observation | Cache/UX/実験割当/計測/通知/非同期Provider | KPI計測値を受理済み業務結果へ格上げ |

**優先する編集体験**：利用者に最初からGraphの内部属性を見せず、ドメイン全体Map → 業務上の操作と例外 → Lifecycle標準部分 → 選択されたPlugin差分 → 合成後の実効Flow → 実Call/失敗の証拠、の認知順を用意する。標準Graph・差分・実効Graphは同じstable IDを共有し、凡例で「固定／設定／Hook可／交換可／観測のみ」と受理/未受理を分ける。生成コードを先に読むことを要求しない。

### 11.2 Graphの形：実行Graphと他のGraphを「型付き参照」で接続する

同一標準Graph表現を**媒体や説明用途まで一つの実行セマンティクスに正規化しない**。少なくとも次のRoleが相互参照できることを目標とする。

| Role別Graph | 主な意味 | Generator／Consumerでの利用 |
| --- | --- | --- |
| Business Procedure/Workflow | Actor、手作業、引渡し、業務状態・時間条件 | Operation参照、Business Diagram、業務試験 |
| Operation Lifecycle | 型付き処理、Guard、Effect、Failure、Observation | 実行Plan、Generator、Authority検証 |
| Interaction Flow | 入力/確認/提示/中断/再開の状態遷移 | CLI/Web/Chat/VUI AdapterのFlow |
| Presentation/Journey | 表示型のインスタンス/選択/画面導線/体験 | Renderer、B2C Native Frontend binding |
| HTTP/Transport Reaction | Protocol状態・Client Reaction・Recovery | HTTP/IPC/MCP等への変換とエラー復旧 |
| Build/Dependency | Plugin、Generator、Resource、入力閉包と失効 | Target Artifact生成/再利用 |

各GraphのNode/Edge identityを、Source/Revision/semantic roleで限定し、他Graphへの関係は明示Referenceで辿る。Operation参照を業務フロー内に置いても、業務フロー自体が再定義されたOperationを自動所有するわけではない。見えない標準GuardやRenderer固有制約は「なし」と扱わず、未表示/未取得/未対応の状態と件数を表示する（#3262、Draft #3505）。

### 11.3 Graph Patchを安全に合成するための最小アルゴリズム候補

~~~text
Source @ R / Accepted Meaning / selected Profile
   ↓ Existing resolver: exact Provider & Plugin Lock L
Base graph G + selected patches P1..Pn
   ↓ Check patch target stable ID, version & preconditions
   ↓ Build dependency DAG and detect collisions
   ↓ Apply only a deterministic, explainable order
   ↓ Normalize effective typed graph G* (layout excluded)
   ↓ Check path obligations / disclosure / effect / version
   ↓ Lower to Target-neutral plan, then target-specific emit
   ↓ Independently verify actual Build / Provider Invocation / Effect
~~~

- **Patch選択**：既存Compositionで選択済みのPluginのみを対象。未選択のPluginは、編集可能なCatalogに見えても実効GraphやBuildへ混入させない。
- **Patch識別**：Base Graph Revision、stable node/edge ID、Hook Contract Version、Source位置、作用Scope、要求Trait、選択Providerの組を残す。描画上のlabel、座標、登録順をIDにしない。
- **構成順序**：順序不要な要求集合は重複排除・正規化。効果/状態に関与するGraph Patchは依存順を明示。同じ対象を変更する非可換Patchは依存/競合の証拠がない限り拒否。
- **変換安全性**：型整合・Relation/対象粒度/NULL/時点・必須参照の閉包、Entry/Exit/terminal、分岐と並行合流を確認。削除/移動は必要なObligationを壊し得るため初期Referenceでは制限する。
- **Guard検査**：あるExecuteへ「Auth Nodeが存在する」だけでは不十分。選択されたContextと入力版が実行時に有効であることを検査し、迂回経路と**Guard後に入力が変更される経路**を独立に拒否する。Graph上の支配関係（dominator）だけで権限保証を立証したことにしない。
- **費用対効果**：大規模な全般Model Checker導入を前提にせず、最初は限定したreachable-path/型/Obligation/競合検査とDriver ScenarioのMutation Testで実証。判定不能はUNKNOWN、静的PASSは実Runtimeの権限保証ではない。
- **変更時の資格**：Base/Patch/Pluginが変更された場合、旧Graphの適合Evidenceは原則そのまま新Graphへ移植しない。Source byte/Meaning/Graph semantic/Plugin/Generator/Artifact/Runtime/Observationをそれぞれ別のrevisionとして説明する（#2600等）。

### 11.4 代表例：受注確定に案件固有の承認を挿入する

~~~text
標準 Operation / Accepted Source:
  order.input → order.validate → order.execute → order.observe

案件Plugin Patch（受注金額が閾値を超えた場合）:
  after order.validate:
    branch(high_value)
    → wait_supervisor_approval
    → resume_and_revalidate

合成後の実効Graph（概念）:
  input → validate → high_value?
                       ├─ no ──────────────────────────┐
                       └─ yes → approval_wait          │
                                         ↓              │
                                      approved?         │
                                         ↓              │
                         resume → recheck input/rev/auth│
                                         └──────────────┤
                                                        ↓
                                               execute → observe
~~~

上図は例示であり、承認待ちのStateをどこへ永続化するか、承認Policy、Concurrent approval、期限、補償を固定するものではない。**どの経路でも実Execute直前の現在Authを検査**し、入力変更後やResume後には適切な再検証が必要。図の「approval_wait」が表示されたことはHost Grant/External Effectを与えない。

**同一Referenceを4つの異質なConsumerで試す**：
- CLI：構造化Eventを入力し、標準/追加Patch両経路の typed Operation Intent を比較。
- Web/Native B2C：完全独自Frontendから同じOperationを呼び出せる。画面表示/実験露出は変えても承認・Effectの意味を変えない。
- VUI：曖昧な「はい」、発話途中、無音・再開後の旧確認を拒否し、表示された承認対象とEventを束縛。
- AI Web Chat：未完成Tool JSON・streaming token・Regenerateを実Operationへ直接接続しない。Tool Callの確定/確認/Host権限が成立した場合だけ同じOperationへ到達。

### 11.5 変更・失効・再生成の判定：Graph意味と媒体の独立性

| 変更 | Meaning再受理 | Graph/Generator | 特に失効するEvidence |
| --- | --- | --- | --- |
| Diagramの座標/スタイルだけ | 不要 | 意味Graph/Operationの再生成不要 | 表示品質/レイアウト関連のみ |
| 表示順/文言/Reader/Voiceテンプレート | 通常不要（ただし確認内容を変えるなら再検証） | 対象Presentation生成・配信の更新 | UI/Voice/a11y/開示、Confirmation binding |
| Edge/Guard/State/対象Rule変更 | 適用するAccepted Meaningに影響するなら必要 | Graphを再解決→検証→依存Targetを再生成 | 経路Coverage/Outcome、旧Flow resume |
| Hookだけ変更（同じContract） | 実装変更だけなら通常不要 | Provider/Artifact/実Call再検証 | Hook/Target conformance・Build |
| Plugin/Generator versionのみ変更 | 意味受理を自動変更しない | lock/build/再生成資格を再検証 | 旧Provider/Build/Target Evidence |
| B2C実験配分/Creativeだけ | 業務意味を変更しない限り不要 | Experimental/Frontend資産・必要Cacheだけ | Exposure/Consent/Variant/SEO/性能 |
| Run中のFlow revision変更 | 過去の意味を上書きしない | 旧版継続か明示Migration、新旧の状態互換検証 | replay/resume/authority/Effect |
| 外部参照Source変更（同名内容変更・ABA） | 参照に関わるMeaning再確認が必要な場合あり | Source closure・受理・生成器資格を別判定 | Source依存/Observation/Build。fingerprint一致だけで復活しない |

**反例**：B2Cの実験設定だけ変えたときに、業務意味の全Owner再受理を要求するのは過剰。一方、Voice確認文言で金額が省略されたなら「見た目だけの変更」ではない。分類は変更ファイル名でなく、適用Contractへの影響から決める。

### 11.6 独立Expectedへ渡す追加の判別力（新Test SSOTは作らない）

| 追加Case | 反例 | 検証する期待 |
| --- | --- | --- |
| LGX-01 | 標準とPlugin Patch/実効Graphが見た目では同じ、内部のEdge方向だけ違う | stable ID/向き/多重辺をSourceから追跡。Effective Planは正しい方向だけを使う |
| LGX-02 | 提示図から必須Guardが隠れたまま「完全」と表示 | 指定Scope・Required集合に対する表示済み/未取得/秘匿のCountを分離 |
| LGX-03 | 複数Pluginの順序を逆転、同一Edgeをreplace | 純粋集合のみ順序非依存。非可換な変更は競合診断 |
| LGX-04 | validate/authorize後に値をGraph Patchで書換える | 実Execute時に古い検証/権限を利用できず、Effect 0回 |
| LGX-05 | Graphの同名別ノード、同端点多重EdgeをDiagramで編集→Sourceへ書戻し | node/edge stable identity/Source spanの保存。曖昧な逆編集は拒否 |
| LGX-06 | Voice/Web Chatのpartial Intent/regenによる重複Tool call | 途中値は実Operationへ流入せず、二重Effectや成功捏造なし |
| LGX-07 | Native B2C routeでTEI Renderer/JS/Hostを全除去 | 契約済みOperationは利用でき、不要なRuntime依存が強制されない |
| LGX-08 | Graph Source/Plugin差替え・旧Flow Resume・A→B→A | 旧Build/Graph/Authority/Evidenceは自動的に現行適合へ復活しない |
| LGX-09 | Graph schema/Consumerが一部未知・非対応のまま図面だけ成功 | 可能なread-only投影とExecutable/Editable資格を分離し、部分結果をPASSにしない |
| LGX-10 | 標準処理とPlugin差込時にレイテンシ/DB read数が増えた | Host/Target/Scenario固定でCPU/IO/Alloc/Cache/Traceを差分観測。性能改善は測定して採否判断 |

LGXは**独立Expectedの設計候補**であり、既存#1180/#2514へ具体的Fixtureとして帰属させる。#3505のMAP-01〜10、#3510のTLG-01〜08、#3274のHDG群と検証概念を重複しても、新しいTest Runnerや硬いGateは作らない。

### 11.7 最短の実装依存と責務の返却

1. **既存最優先**：#1251の同一selected Providerが実CLI/Studio二Consumerから呼ばれることを確認。Source/Target/Host文脈を別証拠にする。
2. **Graph読取り/完全性**：#1124/#3262とDraft #3505。現行Source→Role→Lifecycle Node/Edge→標準/拡張/実効のread-only Mapを最小一Operationで表現し、必要な部分が未Loadedならそれを表示。
3. **Graph Extension**：#2090/#951/#1016/#1121。read-only Hook 1個とBranch Patch 1個を既存Contractへ束縛し、競合とGuard迂回を負例検証。まず簡易Graphを手入力した合成Fixtureで構わないが、その証拠を「TEI Loader済み」としない。
4. **Diagram→Source→Generator**：#1124/#2497/#2620/#1121。図面からの変更候補がSource/Revisionへ正しく戻り、必要なAcceptanceを通りTarget Codeへ反映されることを同一Attemptで確認。
5. **異質媒体**：#3503/#997/#2093/#259/#3509。Web Chatのstreaming/Tool、VUI確認/中断、完全Native B2Cの三つを同じOperationで比較し、媒体固有の実適合を独立取得。
6. **性能・再利用**：#3271/#2600。Graph/Hook/Renderer/Cacheの変更影響範囲を限定し、不要な再生成や高レイテンシの強制を診断する。

これらの番号は本設計の**依存順候補**であり、別の中央Program/Gateを立てるものではない。関連Ownerの既存Issueで必要条件を検収し、追加実装をしないで済む箇所はそのまま再利用する。実Graph Compiler、実Editor書戻し、生成AI Chat/Voice Target、正式Source/Meaning/Authorityの通し実証は引き続き**NOT RUN**。

## 12. 2026-11-08 実装前に確定する最小意味論：Graphは「形」だけでなく遷移の契約（横断引渡し）

**位置づけ**：§11で整理した標準/差分/実効Graphの設計を実装へ渡すため、他レーンの成果と現在mainに統合済みの[#3510](https://github.com/hat47x/tei-platform/pull/3510)（Trait × Lifecycle Graph、TLG-01〜08）を接続して、**まだ未確定なGraph評価・拡張契約**を局所検証可能な形に絞る。Draft #3505のRQ-036（全体Mapと編集凡例）・#3509のB2C・Draft #3515のAdaptive Flowと責務を重複させない。ここで新たなCore DSL・Graph Runtime・検証済み製品能力を採用しない。

### 12.1 「GraphをGeneratorの入力にする」の四つの異なる対象

| 役割 | Source/Authority | 情報として必要なもの | 書戻し・生成の条件 |
| --- | --- | --- | --- |
| **Authoring Graph** | 既存Script/DSL/Markdown/表のauthoritative SourceをLoaderが解釈したRole付き型付き構造 | Node/Edgeのstable qualified identity、Source span、Version、Relation/多重度、未解決 | 単独の第二Sourceではない。図面の編集はSourceへ戻して受理 |
| **Effective Graph** | 既存#951/#1016が選んだPlugin Patch/ProviderとAccepted Sourceを合成した**派生モデル** | 選択済み契約、Patch差分、Node/Edge/Guard/Effect、実TargetのCapability、適用Obligation | #1121で型付きPlanにLower。静的Graphは実Host認可の代用ではない |
| **Diagram/Navigation Graph** | Authoring/Effective Graphの、人・生成AI・SHO向けProjection | 全体Map、固定/設定/Hook可/交換可/観測のみの凡例、対象集合・開示範囲 | 表示位置だけの編集は実コード再生成不要。非可逆表示はread-only |
| **Observed Trace Graph** | 実Client/Provider/Host/Receiptの観測Evidence | attempt、実呼出し先、Flow/Plugin/Artifact/Policy版、Effect known/unknown、未観測 | 実際に通った経路の主張。設計上の「到達可能」と同値でない |

「Graphの絵が同じ」「ログ上で結果が同じ」「Source bytesが同じ」は、それぞれ型付き意味、実Provider選択、現在適合を証明しない。Graphの多重辺・同名異ID・出典・時間をLossy Projectionから推測してSourceへ逆投入しない。読取り・編集・生成・実呼出しの各資格を独立表示する。

### 12.2 生成器に渡すNode/Edgeは型だけでなく動作契約を持つ

**候補の最小Role**（既存標準データ構造へ投影し、Core enum追加を前提にしない）：

- **Node**：stable ID、Operation/State/Rule/Handler/Presentation/Boundary等のRole、input/output type、欠落・NULL・単位/精度、作用区分、Provider/Function参照、Source occurrence、対象/Scope/Version。
- **Edge**：source/targetの独立ID、Flow control／Event trigger／Data dependency／Authority or obligation dependency／Effect observation／Presentation bindingなどの**関係Role**、Guardの型付き参照とイベントの版。複数Edgeが同じNode間を結ぶことを許す。線の見た目・並べ方からEdge Roleを決定しない。
- **Boundary**：subflowのentry/exit、Scope、instance/correlation、failure/cancel/timeout/unknown終端、parallel joinの成立条件、runtime capability。業務TaskとLLM Tool invocationを同じNode kindへ強制しない。
- **Evidence reference**：受理Source・World・Graph・Patch・Plugin Lock・Generator・Build・Run/Attempt・Authority/Receiptの識別を別に保持。Traceの未取得/部分/失効は成功の空集合と区別。

初回ReferenceではControl/Guard/Effect/Observationに必要な役割だけを既存Data/Roleへ投影し、網羅的なGraph型体系を先に固定しない。Graphviz/Mermaid/BPMN/SCXMLをSource互換と無条件に呼ばない。記法の変換はSource位置・Identity・多重辺・編集可能性の保全が確認できた用途に限定する。

### 12.3 動作の単位：Event、Transition、Operation、Effectは同じ「完了」ではない

**実行時の共通約束候補**：

1. 外部Input（音声候補、Web入力、MCP Tool、Chat streaming、timer、CloudEvents等）をまず**Source/Occurrence/Instance付きのEvent Candidate**として受領。Clientが送ったactor/roleを信頼済みContextと扱わない。
2. 対象のFlow/Graph Revision・現在State・必要Resourceを束縛し、型と必要条件を検証してから確定Event/Operation Intentにする。推論・音声/LLMの非決定性は**確定EventまでのProvider側**に隔離し、未完のTool引数で実行しない。
3. 同一Flow instanceで同時に二つのEventが到達した場合、**採用順序か排他/競合の規則を明示**。通信到着順が安定だと仮定しない。並行Regionは因果依存・joinとEffectの衝突を別に扱う。
4. Transitionを評価する際、Flow StateとInput/Policy/Providerの観測Revisionを参照し、正規化したNext StateとOperation Intentを導く。時刻・乱数・外部Readの実値はEvidence/入力として確定してから評価する。timeoutも観測されたEventであり、wall clock値だけから再現性を捏造しない。
5. Operationは現在のHost Grantと必要なValidation、Subject/Target/Expected Revisionを**実Execute境界で検査**。Graph上のAuthorize Nodeを一度通過しただけでは、Resume後の実行許可は成立しない。
6. Tool/Operation受理、Business Outcome、Durable Effect、Receiptと独立Read、Transport成功、Streamの最終表示は別の成否。通信断・中断・結果不明では重複副作用を避け、明示した観測/回復経路へ進む。
7. 実行中Graph更新は、新Revision・Plugin Lock・State Migrationの適合が確認されるまで旧Graphを維持するか停止する。旧Sessionの確認・AI Regenerate・Chat branchで過去Effectを無条件に再実行しない。

この意味論は**単一の汎用Event Queueを持つRuntime**の導入要求ではない。Go静的結合でコンパイルした関数群、Web/IPC呼出し、外部Workflow Provider、Chat Stream Adapterなど、Targetによって物理実現が異なってよい。必要なTarget Capabilityを満たせない場合は未対応を明示する。

**参考**：SCXMLのevent queue/entry-exit/transitionの明示規則 https://www.w3.org/TR/scxml/ 、CloudEventsのsource+idとメタデータ https://github.com/cloudevents/spec/blob/main/cloudevents/spec.md 、AG-UIのRun/Tool/State event family https://github.com/ag-ui-protocol/ag-ui/blob/main/docs/spec/1.0/basic/index.mdx 。これらは意味設計の比較材料であり、TEIのProtocol採用・準拠を決めたものではない。

### 12.4 PluginがLifecycleを変えるときの安全な適用順

**二段階のPatchを分ける**：

- **構成時のGraph Patch**：標準SubflowへのRule/Branch/Guard/Presentation/Provider参照追加、限定Replace。契約IDと対象RevisionへBindingし、依存順・非可換競合を確認してGraphを合成→適用Obligationを検証→#1121でTarget別生成。純粋Rule集合の登録順と、Effectを伴うPipeline順を混同しない。
- **稼働中の選択**：受理済みの状態と、事前に選択・資格確認された選択肢の間で遷移するだけ。LLMが新規の未審査GraphやPluginを要求した場合はProposal→Source/Owner受理→新Lock/Build/Host配備で扱い、当該Runの能力を即時拡張しない。

GraphのPatch合成は次の要件を満たす。①選択されていないPluginのPatchが実効Graphへ入らない、②同じSlotへの複数Replaceや循環・無根拠の順序を診断、③Source/Node/Edge Identityを保持、④Obligationを満たさないEffect到達を拒否、⑤不明なSource closure/動的分岐を未検証として表示、⑥Plugin版・Generator版・Artifact版が変わる場合は旧Proofを安易に再利用しない。

静的に「Authorizeが全経路を支配する」と判定できても、実行時の入力差替え、失効Grant、外部Effectの成功までは保証しない。**静的Path検査・生成された実関数のHook呼出し・Hostの現在権限・独立Observation**を別Evidenceとして要求する。

### 12.5 人が理解する図面を最初に出し、完全性を詐称しない

#3505の全体図ナビゲーション要求とX-TEA由来の「設計書として読めることを先に確保」を採用候補として尊重する。標準全体像→Operation/State→必須保証→変更可能箇所→Plugin差分→実効Graph→生成先→実Call/Evidence、の順序で表示する。

表示の凡例は**標準固定／設定可／Hook追加可／Provider交換可／現在の観測のみ**を一貫させる。固定とは図形を絶対に移動できないという意味でなく、対応するMeaning/Authority/Effect契約をカスタムすることを禁止・制限する意味である。各図はScope/World/Source/Current Disclosure相対のrequired setと、表示・未取得・秘匿・非対応・不明を別に示す。RoleごとのGraphが未解決なら「図が描けた＝全容」とはしない。

Diagram上の意味変更は安定IDのGraph Patchを生成し、Sourceへ戻せることを**適用前に確認**する。Sourceへの曖昧な書戻し、異内容同名Sourceへの誤接続、現在権限のないPatch適用、旧Revision Patchは拒否。位置/表示だけの変更はLayoutを更新し、Business Meaningと実コードは変更しない。SHOは同じGraphの文書投影を扱い、第二の受理済みLifecycle正本を持たない。

### 12.6 Chat / VUI / B2C / Automationを同じOperationへ接続する範囲

| Consumer | 固有の制御/状態 | 共通Contractへ渡すもの | 不当に昇格させないもの |
| --- | --- | --- | --- |
| AI Web Chat | Conversation/Thread/Turn/Run/Message/Stream/Tool/Approval/Branch/Resume | **確定した型付きTool Intent**と適切な現在Authority、Result投影 | partial Tool JSON、Streaming delta、生成カード、Regenerate |
| VUI | ASR候補/確認対象/無音/interrupt/session revision/TTS | 確定Event→同じOperation | 認識候補、読み上げ済み表示、古い同意 |
| Web/Desktop/CLI/MCP | Form/argv/IPС/Tool、transport・Host lifecycle | typed Request/Result/FailureとEffect identity | HTTP 200/JSON-RPC成功、CLI exit、UIの成功表示 |
| B2C Headless | Native SSR/Hydration/Route、Experiment/Exposure/Marketing Consent | 権威あるCart/Checkout等のOperation | 推定CV、Analytics event、Client価格、Frontend上の完了 |
| 外部Scheduler/Automation | timer/event subscription/job retry/endpoint availability | 確定Event、対象/Correlation、許可済みOperation | 期限発生=処理完了、Job受付=Effect確定、無条件Retry |

AG-UI等はChatの増分表示/Tool境界、A2UI等は宣言的Presentationの候補であり、どちらもTEI OperationやBusiness Meaningの所有者としない。Webhook/CloudEvents/外部イベントProviderを増やす場合も既存Hook選択・Source/Authority・Effectの同じ境界を適用する。**KON Dataspaceは任意のSource/Record保管・版/参照基盤、EKIは選択された分散実行のAttempt/Resource委譲、SHOは文書Projection**の候補であり、いずれもTEI独自の中央Hostとして前提化しない。

### 12.7 追加の最小実装受入票（既存TLG/LGXに重複SSOTを作らない）

| 圧力入力 | 具体的な反証 | 既存の引渡し先と必要Evidence |
| --- | --- | --- |
| **Event order** | 同一Session/GraphにConfirmとCancelが並行到着。到着順を逆転させる | #3503/#2514：選択規則/State revision/二重Effectなし。未定義なら競合拒否 |
| **Timeout/replay** | timeout Event発火後に旧Sessionの確認応答、さらにGraph版更新 | #3503/#1180：期限/旧確認の失効、旧Graph再開拒否か明示Migration |
| **Stream candidate** | Chatのpartial tool argsが最終JSONと異なり、UIが先にToolを起動 | #1158/#2093：確定前Operation 0回、current Auth/入力検証は別 |
| **Cross-Graph boundary** | Business Stateの承認済み表示だけ変わり、Effectはunknownのまま | #1125/#1129/#1174：Business State・Flow State・Effect dispositionを独立証拠に |
| **Patch/Source** | Diagramで同名別IDのparallel edge変更、Source途中変更・ABA | #1124/#2497/#2600：Typed Patchの正確なSource帰還、古いEvidence再利用拒否 |
| **Actual target trace** | 一方のTargetで必須Hook呼出しが脱落、別Targetで同じResult | #1251/#1121/#259：selected→built→invoked→observed、Coverage差を不合格 |

この表はLGX-01〜10/TLG-01〜08/MAP-01〜10/HDG群の**既存独立Expectedに足す入力変異**であり、新Runnerや新Product Gateではない。これらを一つの巨大なGoテストとして先に作らず、(a) 実selected Provider二Consumer（#1251）、(b) read-only HookのTarget binding（#2090）、(c) read-only Graph→Source roundtrip（#1124）、(d) FlowのCLI構造化Event一往復（#3503）、(e) Chat/VUIの部分入力と再開、の順で反証する。

**成果を評価する基準**：Graphを経由したことで「人にとってSource/変更位置が追跡しやすい」「Generator側の未実装/Guard迂回を早く検出できる」「Source/Plugin/Target別の再生成を必要最小限に絞れる」ことが、実Source/Target/独立Expectedで測定できるか。単に図面項目やHook点の数を増やすことは成果ではない。これらの試験とOwner受理は現時点で**NOT RUN**。
