# API / Web / Desktop / CLI / MCP / VUI横断：生成ExtensionとInteraction Flowの統合引渡し（2026-11-08）

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
