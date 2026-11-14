# プログラム実行モデルの図解とTarget間移植：設計・検証計画

- 状態：**実装前の横断設計案**。ここで整理する共通要素は、正式なDSL構文・Schema・実行Graph Compilerの採用を意味しない。
- 主担当：[実行可能Lifecycle Graph #4198](https://github.com/hat47x/tei-platform/issues/4198)。関係する仕様と検収は、それぞれの既存Issueが所有する。
- 接続先：[Script起動 #4502](https://github.com/hat47x/tei-platform/issues/4502)、[実Entry #4495](https://github.com/hat47x/tei-platform/issues/4495)、[Trigger #4497](https://github.com/hat47x/tei-platform/issues/4497)、[状態機械 #4193](https://github.com/hat47x/tei-platform/issues/4193)、[意味Role #2620](https://github.com/hat47x/tei-platform/issues/2620)、[DSL #988](https://github.com/hat47x/tei-platform/issues/988)、[Lowering #1121](https://github.com/hat47x/tei-platform/issues/1121)、[Studio #1124](https://github.com/hat47x/tei-platform/issues/1124)、[Effect #1129](https://github.com/hat47x/tei-platform/issues/1129)、[独立Expected #2514](https://github.com/hat47x/tei-platform/issues/2514)、[Stage2セルフビルド #4583](https://github.com/hat47x/tei-platform/issues/4583)。
- この文書の役割：既存の意味・実行・図解・移植・検証の境界を一か所で接続すること。業務意味、実行権限、Scenario、検証結果の第二正本にはしない。
- 今回の作業範囲：文書と計画の接続のみ。Go・Java・.NET・Web・実Providerのコード変更やE2E試験、Ownerによる仕様受理は **NOT RUN / NOT ACCEPTED**。

## 1. 問題設定と基本判断

Webアプリケーションでは「基盤の起動→受信待機→要求の到来→処理の選択→実行→応答」が繰り返されることが多い。この構造は、GUI、CLI、Batch、Message Consumer、Timer、OS Service、SaaSのFunction Invocationにも応用できる。ただし、すべてのTargetが同じ待機ループや常駐プロセスを持つわけではない。サーバーレスの起動管理や単発CLIの終了は、実Hostが担うこともある。

**TEIが標準化する対象は、特定のイベントループではなく、起動の契機・処理の意味・実行責任・副作用・結果の観測を区別できるプログラミングモデルである。** 実行ループやSchedulerをTEI固有のRuntimeとして実装することは、この構想の前提としない。

    実行環境の初期化・公開
              │
       起動契機の発生
              │
        Entry・対象の解決
              │
       現在の入場条件を検査
              │
       実ProviderでOperation実行
              │
    ┌─────────┴─────────┐
    │                   │
    値・業務結果       更新・外部作用
    │                   │
    応答／開示       Commit／Receipt
    └─────────┬─────────┘
              │
       独立Observation・終了／再入

上図は**説明のための投影**である。結果の返却と外部作用の確定は、常に左から右・上から下の順で起きるとは限らない。非同期配送、Commit後通知、ACK喪失、キャンセル、部分失敗では別々の因果関係と資格を追跡する。

## 2. 共通モデルの最小構成

| 要素 | 意味として区別する内容 | 実装上の責任 |
| --- | --- | --- |
| Execution Environment / Lifecycle | 起動、準備完了、停止、復旧、再入。初期化単位とInvocation単位の差 | OS・Host・Target Binding |
| Trigger Definition / Occurrence | 手動操作、HTTP、Timer、Message、Data Hookなどの宣言と、実際の発生事実 | #4497。発生は実行成功でも許可でもない |
| Entry / Prepared Invocation | 公開された型付き入口、対象、Input、期待Revision、期限、Operation識別 | #4495/#4502。Entryの発見は呼出権限ではない |
| Admission / Attempt | 現在のGrant、Scope、Provider、実ロードArtifact、Resource、開始の成否 | 既存Host/Driver・#1118/#1145 |
| Operation / Function | 計算、検証、Query、状態遷移、他Functionの呼出し、必要なEffectの要求 | Data/Function/BusinessのAccepted Meaning、Operator/Provider |
| Effect / Outcome | 永続化、外部通知、業務結果、失敗、不確定な効果 | #1129/#1174。結果値と確定済みEffectを混同しない |
| Response / Delivery / Observation | HTTP応答、CLI出力、GUI表示、ACK、Receipt、実状態の再読取 | #3274/#2586/#2514。受信者の実受領を別に検査 |

これらは**意味Roleの候補と既存構成の照合表**であって、新しい一枚岩のSchema、全用途共通の大きなEnvelope、独立のExecution Registryを追加する要求ではない。標準Data/Graph構造に必要なRoleと型付き参照を重ね、必要な箇所だけ対象固有の契約を設ける。

識別子も一種類にまとめない。少なくとも、Source occurrence／意味上のSubject・Revision／Trigger occurrence／Operation ID／Attempt／Provider Execution Reference／Effect Receipt／Traceを区別し、同じ表示名・時刻・応答番号による自動同一視を避ける。単一の万能IDをすべての経路に強制せず、実際に存在する因果関係を型付き辺で結ぶ。

## 3. 実行意味論と図の意味論を分ける

同じ図形の列が、同期処理、非同期処理、並行処理、Eventual Consistencyのすべてを同じ意味で表せるわけではない。次を対象Function/Targetに必要な範囲で明示する。

- **時間と順序**：発生時刻、配送時刻、評価時点、開始・完了時刻、Business Period、タイムゾーン／DST、順序保証を区別する。時計やEvent QueueをCoreへ固定しない。
- **並行・再入**：複数Attempt、親子Invocation、排他・競合、再送、Deadline、Cancellationの扱い。親のGrantを子へ自動継承しない。
- **状態・副作用**：計算による候補状態と、DB Commit・外部Effectの実確定は別。結果が不明な場合を成功・未実行に変換しない。
- **不変条件**：型、必要Input、許容遷移、認可、Effect fence、開示、Targetで表現可能な能力を明示する。PluginのGuard/HookやGraph Patchが回避してはならない義務を持つ。
- **評価方式**：Rule/Decision Table/State Machine/Stream/Batchなどの評価方式は、単一の固定実行順に押し込まない。必要なModel of Computationを適用範囲つきで示し、Targetが満たせなければunsupportedとする。

純粋な意味評価は概念的に「モデル・状態・入力・Contextから値、候補の状態変化、要求される作用、診断を導く」と表せる。ただし、その評価結果が実HostのGrant、外部Commit、配送成功、Consumer受領を証明するわけではない。意味評価、実行入場、実Provider呼出し、独立Observationをつなぐ検査順は、既存の **Prepare → Admit → Invoke → Observe** を用いる。

## 4. 図解の構成と安全な編集

すべての情報を一枚のフローチャートに詰め込まず、同じAccepted Meaningから次の図表を必要に応じて生成する。

| 投影 | 読み手の問い | 図と正本の関係 |
| --- | --- | --- |
| Lifecycle図 | 何が起動し、いつ待機・停止・復旧するか | Host固有部分とTEI定義を明示 |
| Control Flow・Decision図 | どの条件で何が実行されるか | 順序、分岐、例外、unknownを保持 |
| Interaction・Call図 | 誰が何を呼び、どの応答を受け取るか | EntryとInvocation/Consumerを分離 |
| Data・Effect図 | 何を読み、更新し、外部に通知するか | 宣言上の依存と実Read/実Effectを分離 |
| State図・遷移表 | どの状態・Eventで遷移が許可されるか | #4193が採否を判断。IFを無条件に置換しない |
| Trace・Evidence図 | 何を実行・観測できたか | 設計図とは異なる、実Attemptの観測投影 |
| Build・Artifact依存図 | 何を生成し、どの実行先が実際に使ったか | Source/Builder/Artifact/Consumerを別Revisionで確認 |

図の配置、色、線の曲げ方、Markdown・Graphvizなどの表記はPresentation Metadataであり、業務意味や実行順を自動的に変更しない。図からの編集は、SourceのSubject・Revision・出現位置と編集可能範囲を特定した**型付きChange Proposal**に変換し、Preview → Resolve → Admit → Commit → Observeの既存編集・検証境界を通す。

元のMarkdown/DSL/Graphviz等の糖衣表現は保存時に可能な限り保持し、意味のない全ファイル整形を避ける。図への投影で未表現となる属性はloss/unsupportedとして示す。逆写像できないProjectionを完全可逆と表示しない。AIが図やコードから推定したRelationはCandidateのまま保持し、Ownerによる意味受理を代替しない。

## 5. 異なる言語・Targetへの移行

移植元のコードを機械的に別言語の構文へ置換することと、同じ意味を別環境で成立させることは異なる。移行の単位は、必要に応じてFunction、Entry、Procedure、State Transition、Data Relation、Host Bindingへ分割する。

1. **Read-only Inventory**：元のSource/版、Call/Control Flow、イベント起点、実行Context、外部依存と副作用を列挙し、静的解析で不明な動的挙動はunknownとする。
2. **Candidate Semantic Map**：Data/Function/Business、Trigger/Entry/Operation/Effectへ候補対応を作る。旧コードの動作観測・AI推定だけでAccepted Meaningを生成しない。
3. **Gapと実現方式の選択**：同期/非同期、再入、状態、DB/Network、権限、精度、欠損/NULL、配布モデル、時刻と順序について、Target固有のBinding/Generator/Providerで実現できるかを照合する。
4. **生成と比較**：受理された意味からTarget実装を生成する。未実装の外部Componentは契約とMockで全体Scenarioを先に確かめられるが、実Provider適合は別の検収とする。
5. **実Consumer受領**：生成コードがビルドできたことだけでなく、同じSource/Meaning/Artifact/Providerが実CLI・Studio・Web等から使われた事実、必要な外部Effectと受領の範囲を追跡する。

例として同じ受注確定FunctionをHTTPとCLIから呼び出す場合、戻り値が一致するだけでは十分でない。入口が同じ意味上のFunctionに結び付いているか、選択したProviderを実際に呼んだか、同じ入力・権限・版で処理したか、Commit/Receipt/結果の開示が必要範囲で成立したかを別々に検査する。これは既存の#4495/#4502/#2586の検収を再利用する。

Target適合表では、Java/.NETの常駐Web Service、Go/Wails Desktop、単発CLI、Serverless、Batch/Message Consumerを例示できるが、**すべてに常駐Event Loopを生成することはしない**。Hostが所有するライフサイクル、TEIに表せる制約、Providerに委ねる保証、未表現の保証を区別する。

## 6. セクション化・メタプログラミングの適用条件

ここでいうセクション化には、コードの見出し・ファイル・関数を整理する**表層的な分割**と、独立した入力・出力・Effect契約を持つ**意味上の処理単位への分解**がある。見た目の区切りが同じでも、並行実行、再送、例外、Transaction境界をまたぐと意味は変わり得る。TEIが自動変換の対象にできるのは、必要な依存と制約を説明できる範囲だけである。

例えば受注確定処理を「入力の検証→金額計算→受注確定→確認通知」と分割する場合、金額計算は純粋Functionとして抽出できる可能性が高い。一方、受注確定と通知を独立の非同期処理へ分離するには、Commit後通知、Outbox、再送、重複防止、Effect unknownの契約が必要になる。単に四つのNodeへ分けたことは、それらの保証ではない。

分割・抽出・再配置・Target変更を行うメタ変換は、次の情報を失わないことを要求する。

1. **対象と由来**：意味Subject/Revision、Source occurrence、静的・動的参照の探索範囲。未解析の反射呼出し・外部Callはunknownとして残す。
2. **境界の明示**：型・多重度、missing/NULL/未取得、読み取り範囲、Outputと例外、必要なContext/Grant、Provider/Target要件を抽出後のInterfaceへ移す。
3. **観測可能な振る舞い**：Guard、評価順、精度と丸め、状態更新、Eventの因果・順序、Transactionの開始/確定、外部Effectの時点を保つ。効果を持つ処理を純粋と推測しない。
4. **変換責務の分離**：Source→Canonical Meaningの抽出はCandidate、Accepted Meaning→TargetのLoweringはGenerator、実Artifact→実Consumerは検収とする。変換器自身の「一致」判定をOwner受理にしない。
5. **差分の局所性**：型付きGraph PatchとDependency/Build差分を使い、表示Layoutの変更で意味やコードを再生成しない。A→B→A、Plugin/Operator版変更、非対象依存追加での旧Evidence再利用は別に判定する。

この方式なら、既存コードをFunction/Procedure/Effect単位へ切り出して共通部品を生成するだけでなく、Graphから呼出し順やUI/APIの公開面を導出し、言語別Generatorの選択やテスト生成に使える。ただし、Graphからの変換を一般に可逆と仮定しない。**変換前後の型付き入出力、必要な副作用、独立Expected、実呼出し先、Consumerの観測結果**を比較し、不足した保証はGapとして残す。

最初の比較は、同一pure Functionの「元コードの実行／意味モデルのシミュレーション／Go Targetの実起動」とする。次に副作用を伴うFunctionを選び、結果値が同じでも実ReadやEffectが異なる負例を入れる。二つの検証を分けることで、単なる構文移植と意味を保存したメタ変換の違いを判別できる。

## 7. 検証すべき反例（既存Scenarioへの引渡し候補）

| ID | 比較・故障注入 | 不合格を検出する要点 | Owner |
| --- | --- | --- | --- |
| PX-01 | 同一pure FunctionをAPIと実CLIから呼び出す | 値が一致しても実Providerが異なる、または片方がReferenceのみ | #4495/#2514 |
| PX-02 | Manual/HTTP/Timerで同じEntryを指定する | Trigger発生を、Grant付与・実起動・業務完了と誤認 | #4497/#4502 |
| PX-03 | Event重複とACK喪失後の再送 | 同一OperationでもEffectを重複確定、unknownを自動成功・自動再送 | #1174/#1129 |
| PX-04 | Commit前通知・Rollback | 外部送達済みと誤表示し、Effectの位相を飛ばす | #1129/#1174 |
| PX-05 | 同じフローを同期Hostと非同期Hostへ移す | 順序・期限・戻りの表現不一致を隠して同値と判定 | #1121/#4198 |
| PX-06 | 図の座標だけを変更する | 意味Revisionや生成Artifactが不要に変わる | #1124/#2620 |
| PX-07 | Source改版後の旧図・旧Triggerを使用する | 同名の別Subjectへ再束縛、旧Grantで起動 | #1124/#4497 |
| PX-08 | 状態機械をIF/Ruleへ展開する | Guard、直交状態、遅着Event、不変条件の喪失を隠す | #4193/#2514 |
| PX-09 | 可視化で未読込Plugin／秘密Sourceへの辺が欠ける | 不明な依存を「参照0・安全・完全」と表示 | #1251/#1124 |
| PX-10 | 同じ結果を返す実装が余分なデータを読む | 値の一致だけでRead Footprint/Disclosure合格 | #2586/#2514 |
| PX-11 | Buildは成功、実Consumerは旧Artifactを使用 | 生成物と実使用物を同一視 | #4583/#2831 |
| PX-12 | Targetが必要なEvent順序・Effect保証を持たない | unsupportedを自動代替実行して意味を変える | #1121/#4198 |

独立Expectedと合成Fixtureの結果は区別する。未実行の反例をPASSとは表示しない。状態図・表の見た目の一致だけでは、実時刻・実DB・実Deviceの動作が正しいことを保証しない。

## 8. 実装計画と優先順

**P0：既存の製品縦断を先に成立させる。** #4495の実Entry→型付きInvocation→実Providerの二入口と、#4583/#2831のStudio Desktop Stage1→Stage2自己再構築を優先する。今回の図解仕様やGraph Compiler完成を、これらの新しい必須Gateにしない。

**P1：既存Graph/Sourceのread-only投影を試す。** #4198 R0の棚卸しとして、一つのpure Functionと一つのTriggerを対象に、Lifecycle／Entry／Operation／Response／Evidenceの関係を既存Graph型へ投影する。元Source、辺の識別子、Representation Loss、未取得依存を明示し、図のLayoutのみ変更した場合の非再生成を検査する。Studio/CLIの固定選択・ソースジャンプは#1124/#4515へ接続する。

**P2：二つの異なるTargetで意味保存を比較する。** 同一Accepted Meaningの一Functionを、まず実Go CLIと既存Studio/Hostで検収し、続いて異なる言語または異なるLifecycle（例：常駐サービス対単発実行）へ適用する。差が出た箇所をRole不足／Target Binding／Provider／Host Authority／生成器不足へ分類し、差が消えるまで「移植可能」と断定しない。

**P3：編集・再生成と応用を検討する。** #4198 R1〜R4、#4193の異質DomainのPressure、#3503のVUI、既存HTTP応答Decision Graph、Batch/ETL、SHO文書生成へ、必要なProjectionだけを広げる。図編集からSource Change Proposal、受理、再生成、実Consumer確認までの可逆性が確認されたものだけ書込編集へ進める。

採否の停止線は、第二Host/Runner/Resolverが必要になる、現在Grantを迂回する、図からAuthoritative Sourceへ戻れない、Plugin登録順で意味が変わる、異質Target間の不足を隠す、未実行なのに製品合格を主張する、のいずれかである。

## 9. 各担当への引渡し

- **#4198**：共通Lifecycle/Control/Effect Graphの最小Roleと、Graph Template + Obligation + Plugin差分が既存標準Graphに収まるかを評価する。正式Schema/Graph Compilerの採用は保留。
- **#4502/#4495/#4497**：公開EntryとTriggerを実Invocationへ接続し、実Call・入場資格・再入・配送結果を検収する。新SchedulerやScript Runnerを作らない。
- **#2620/#988/#1121**：Data＋意味Role＋DSLの糖衣＋Target Loweringを分離し、評価順やRuntime能力を文法に埋め込まない。生成Goを業務意味の正本にしない。
- **#1124/#4193**：図・表・Traceの相互参照、選択の固定、Representation Loss、変更Proposalを担当する。State Machineの採否は異質Domain比較後。
- **#1129/#1174/#2586/#2514**：Effect/Receipt/Provider Read/独立Expectedを各Ownerの検証に接続する。HTTPステータス、ACK、Business Outcome、永続作用の資格を分離。
- **#4583/#2831**：Source→Build Intent→Builder実行→Artifact→Stage2 Desktop→実Windows Consumerの現在の最優先経路を維持する。図解構想を理由にセルフビルドへ別Runtimeを追加しない。

この設計案の統合は実装や採用の完了ではない。次の判断は、#4198 R0のread-only実証と#4495/#4583の独立した製品検収で得られる不足点を見て、既存Ownerが行う。

## 10. 定義・実効計画・実行記録・図面を区別する

前節までの「共通Graph」は、一つの巨大な実行フローをすべての画面で共有する意味ではない。少なくとも、次の四つを別の投影として扱う。

| 区分 | 何を表すか | 変更と合格の判断 |
| --- | --- | --- |
| 意味の定義（Definition） | Sourceに記述され受理されたFunction、Entry、条件、状態、Event、必要CapabilityやEffectの契約 | Accepted Meaning、Source/Operator/Role revisionが基準。未受理の解析結果はCandidate |
| 構成済み実行計画（Effective Plan） | 選択済みPlugin、Provider、Target、追加Guard/Hook、実行順制約と実装上のObligation | 対象と選択規則を固定した導出物。生成できても実際のロードや呼出しは未証明 |
| 実行の記録（Observed Trace） | 実際に読み込み、入場を許可され、呼び出され、どこまで応答・確定・受領されたか | Operation/Attempt、実Provider、実Read、Receipt、Consumer観測で判定。未観測の辺を実経路にしない |
| 表示・操作用の図面（Projection） | Graph/Tree/Table/Sequence/Timing/State/説明文のレイアウトと操作位置 | 正本への参照を持つView。座標・色・表示順の変更は意味変更ではない |

DefinitionからEffective Planへは既存Composition/Loweringで導出し、Effective PlanからObserved Traceへの対応は実行時の識別子で照合する。Observed TraceがDefinitionの意味を自動更新することはない。AIが実行履歴から推測した依存関係も、受理前はCandidateである。

同じ二つのNode間でも、制御フロー、データ依存、時刻上の先行関係、Effectの因果、権限上の要求、Source出典、実行観測は異なる辺である。型付き辺には、必要な範囲で関係種別、方向、Multiplicity、固有のEdge identity、条件・時点、Source locatorと探索Coverageを持たせる。表示のための線IDを意味上のEdge identityへ流用しない。同じ名称のNodeが二つ存在し、多重有向辺がある例で参照・投影を先に確かめる。

## 11. 実行規則（Model of Computation）の適用範囲

Bootstrap→待機→Event→処理→応答という説明は有用だが、それを万能の逐次フローとして仕様化すると移植時に意味が失われる。**何がいつ評価され、どこまでの順序や状態が保証されるか**は、プログラム構造とは独立した実行規則として、必要なSubjectに適用範囲つきで表す。

| 対象例 | 主な実行規則 | Hostに委ねる部分・保留条件 |
| --- | --- | --- |
| 常駐HTTP Service | Request単位のInvocation、同時実行、Response/Abort、Session等の必要Context | Listener、Thread/Task管理、接続停止。並列度や順序を推測しない |
| CLI単発実行 | Process入口、引数・標準入出力、終了コード、実行単位のResource | Process起動とOS権限。終了コード0とEffect確定を区別 |
| Serverless | Host側の初期化・再利用、Invocation単位のContextと期限 | Instance再利用・同時実行・配送はPlatform依存。常駐Loopを生成しない |
| Desktop/GUI | UI Event、非同期処理、描画スレッドとの境界、画面State | GUI DispatcherとOS/Wails/WebView。古い非同期応答を現Selectionへ混ぜない |
| Schedule/Queue/Batch | 発火、配送、対象集合、遅延、重複、再送、Checkpoint | Scheduler/Broker/Workerは既存Host/Provider。発火時刻と業務評価時点は別 |
| Stream/科学計算/Simulation | Window、Tick、サンプリング、遅着、計算順、単位・精度 | 既存Operator/Toolkitの適合条件。一般時間EngineやSolverをCoreへ追加しない |
| MCP/VUI/AI Web Chat | Typed Operation、部分入力、明示確認、割込み、Tool受領 | Interaction Profile固有の表現を分離。部分的Tool argsを実行指示へ昇格しない |

最小検査項目は、（1）起動・評価の契機、（2）状態の所有単位、（3）必要な順序／並列性、（4）時間・Deadline・業務評価時点、（5）失敗・取消・再送、（6）結果・作用の確定点、（7）Targetが満たす能力、の七つである。すべてのSubjectに七項目の巨大Envelopeを保存させるのではなく、移植や実行に必要な項目のみ既存Role/Trait/Target契約で明示する。不明な保証はunknown、適用先で実現できない保証はunsupportedとして残す。

State Machine、Rule、Decision Table、Reactive Flow、Dataflow、Stream、Batchを単一の実行順へ押し込まない。単純なIF/Ruleで十分なDomainに状態機械を強制せず、#4193の異質Domain比較で導入価値を判断する。

## 12. 処理を切り出すときの境界契約

プログラムを「機能の単位」に切り出す作業を、型付きGraphの部分領域を抽出する操作として説明できるようにする。ただし、Graph上の切り出し線を引いただけでは独立したFunctionにはならない。境界の外から参照する値、制御、状態、権限、時刻、作用を契約として閉じられることが必要である。

| 切り出し後に必要な契約 | 確認する内容 |
| --- | --- |
| 入力・出力 | Typed Ref、Presence/NULL/未取得、単位・精度・丸め、多重度、例外・結果の区分 |
| 依存の閉包 | 呼出すFunction/Rule/Operator、読込Relation・Field・Scope、動的参照、選択ProviderとVersion |
| 制御と時間 | Guard、短絡評価、Evaluation Order、並列実行、再入、Snapshotの評価時点、Deadline |
| 状態・Effect | Read Footprint、候補変更、Commit/通知などの位相、Operation ID・再送・補償 |
| 公開・権限 | Entryの公開面、必要Capabilityと実Host Grant、Scope、開示条件、Target適合 |
| 証拠 | Source/Meaning/Artifact revision、独立Expected、実Call、Receipt、Consumer Observation |

例：受注の「入力確認→金額計算→注文確定→通知」という表示からは、金額計算が独立pure Functionへ切り出せる可能性を検討できる。しかし注文確定と通知を別Taskへ分けるには、DB Commit後の通知であること、Outbox等の実配送方式、ACK不明時の再送規則が契約として必要になる。これらを失う変換は「構造化成功」ではなくGapまたは拒否である。旧X-TEA Sourceから受注Functionの対応が確定していなければ、名称の類似だけで新しいFunction IDへ接続しない。

変換の扱いは次の三段階にとどめる。

- **read-only抽出**：Sourceと静的依存の候補を図・表・説明へ投影する。未知の反射・動的Call・外部Effectは未解決として残す。
- **安全条件つき提案**：純粋計算など閉包が説明できる部分について、抽出先のInput/Output/Dependencyを提示し、対象Revisionに結び付けたChange Proposalを作る。Sourceの糖衣表現と非対象Byteを保存する。
- **実装・実行への反映**：既存Acceptance後にBuilder/Generatorへ渡し、各Targetの出力、実Provider・実Consumer・必要なEffectを別々に検収する。

宣言されたpure Traitと実際に作用しないことは別である。例としてCLIの計算はpureでも、Hostがログやファイルを生成する場合があるため、Function内の作用とHostの作用を混同しない。

## 13. 言語間移行とメタ変換の同等性

「同じ値を返した」というだけでは、異言語移植の合格条件にならない。対象の用途に必要な同等性を、段階的かつ別々に説明する。

| 比較する面 | 同等性を損なう例 | 検証・証拠 |
| --- | --- | --- |
| 構造と型 | 異なるSubjectへ名前で再束縛、NULLとmissingを統合、複合キー・順序・多重辺が欠落 | Source/Meaning ID、型とRefの照合 |
| 計算と評価 | Decimalの丸め、単位、短絡評価、NULL伝播、評価順が変化 | 固定入力、独立Expected、Mutation |
| 時間と制御 | 同期処理を非同期化して順序喪失、再入やDeadlineが異なる | Trigger/Attempt/Timing Trace、Target Capability |
| 状態と作用 | 出力値は同じでもRead Footprintが広い、通知がCommit前、効果不明を失敗扱い | 実Read、Effect Receipt、独立Observation |
| 権限と開示 | Sourceからの要求をGrantに昇格、移植先で秘密を余分に取得・返却 | 現在Host Grant/Disclosureと実Provider Call |
| 配布・受領 | 生成Artifactは最新でもConsumerは旧版を起動 | Build/Artifact/Loaded Bytes/Consumerの照合 |

同等性の主張には、対象Function/Target、入力集合、実行規則、検査した作用と観測範囲を添える。部分的に一致した範囲から、未検証の対象全体や別Targetへ同等性を拡張しない。

変換器の入力と出力も既存の資格を分けて扱う。旧言語のSource解析結果は**候補**、受理済みMeaningは**仕様**、Target-neutral planは**構成済み実現候補**、生成Artifactは**物理成果物**、実行Traceは**観測**である。Targetが必要な保証を表せない場合は、暗黙のFallbackではなく、既存#1121/#951/#3278のConformance上のGapとして返す。

必要に応じ、既存の標準Record/Sequence/Relation/Graph/Expressionに意味Roleを付けることで、言語専用ASTの内部構造を業務DSLへ露出せずに済む。構文定義とParser、意味解釈、Model of Computation、Target実現は別の契約とし、自己記述の有限Bootstrapとexact Pinは#2620/#4359の評価に従う。Goのstruct/func等をDSL必須要素にしない。

## 14. Studioでの表示と編集体験

初期版は新しいGraph Editorを作らず、既存のInspector／型付きReference／Source索引から、選んだFunctionまたはEntryを中心に表示する。

- **全体から局所へ**：アプリケーション→Entry→Function→処理節→Rule/Fieldへ粒度を変えても、同じMeaning IDとSource位置へ戻れる。画面ごとに別の編集正本を持たない。
- **見たい関係を切り替える**：制御、入力と出力、状態、時間、Effect、Authority、実呼出しを重ね合わせず切り替え可能にする。Graph/Table/Tree/Traceの相互参照と表示Lossを示す。
- **確実な対象指定**：現在のFocusとは独立したPinned Selectionに、Subject/Source/Revision/Scopeを保持する。古い結果・曖昧ID・権限失効時は表示を残しても、書込み・実起動・AI送出を保留する。
- **説明文への展開**：受理済みの業務意味と処理の型付き辺から、認知順に沿った説明文、Graphviz、Markdown表、テスト対象一覧を導出する。AIによる補足は候補として区別する。
- **編集の帰着先**：図面でNode/Edgeを変更した場合は、編集可能なSource occurrenceを確定できるものだけProposal化する。対応できないProjectionはread-onlyとし、元のMarkdown/DSLなどの非対象Byteを不必要に整形しない。

三要素分析法については、業務フロー・機能階層・データモデル・CRUD・クロスレファレンスなどの原典に忠実な再現を先に検証する。この汎用プログラムGraphは**TEI独自の追加投影**であり、業務モデルを自動Workflowへ、機能モデルを関数計算へ黙って置換しない。

## 15. 次の小さな実証と既存計画への引渡し

今回の追補は独立のRunner／Parser／Graph Compiler／Statechart Engine／Schedulerを実装する許可ではない。まずP0のStudio Stage1→Stage2セルフビルド（#4583/#2831）と、製品Entry→実Providerの二入口（#4495/#4502）を維持する。以下は#4198 R0〜R4、#4193、#2514の既存検証へ渡す**追加の反例候補**であり、新Gateや第二Scenario正本を作らない。

| ID | 最小の対照 | 確認すること | 既存Owner |
| --- | --- | --- | --- |
| PX-13 | 同じ表示位置に二つの同名Node、同じ端点間に二つの有向辺 | Graph identity／辺種別／方向／多重度をSourceまで追跡。曖昧なら書込拒否 | #2620/#1124/#4198 |
| PX-14 | 定義Graphは同一、Provider A/BとHookだけ差替え | Effective Plan差と実選択／実呼出しを分け、Guard/Effectを迂回不可 | #951/#4198/#2514 |
| PX-15 | 同じControl Flowを常駐HTTPと単発CLIへ投影 | Host待機の有無ではなくEntry/Function/Resultの必要な意味を比較 | #4502/#1121 |
| PX-16 | pure計算を関数へ抽出し、別Targetへ生成 | Input closure、演算・丸め、元Source、独立Expected、実Consumerを照合 | #988/#1121/#2514 |
| PX-17 | Commitと通知の間で処理節を分割 | Effect位相・Outbox・ACK不明・再送条件の喪失を検知し自動分割を拒否 | #1129/#1174 |
| PX-18 | 同じ入力値で依存Function・Provider実装だけ変化 | 構文・値が同じでもPin/Build/Evidence再利用を再判定 | #2620/#2600/#2831 |
| PX-19 | ServerlessのInstance再利用と二重Invocation | 初期化回数やProcess状態を業務一回実行とみなさない | #4497/#4502 |
| PX-20 | GUIで旧選択の非同期応答が新選択へ到着 | Pinned Subject・Revision・Disclosureの不一致を表示し誤反映しない | #1124/#4515 |
| PX-21 | 同じGraphから図だけ並べ替え、説明文だけ変更 | Layout変更は意味・実行を変えない。説明変更は必要な文書再確認を別判定 | #1124/#2600 |
| PX-22 | 値は一致するが実ProviderのRead範囲が部分的 | 値の成功をSource/Read Closure/Consumer完全性へ拡大しない | #2586/#2514 |

**進行順**：最初は一つの受理済みpure Functionと既存Entry/Triggerのread-only投影、次に同FunctionのGo CLI／Studio実Provider呼出しの対照、続いて「pure節抽出」と「Commit後通知の分割拒否」を比較する。異質なTarget間の生成・状態機械・対話UI・書込図編集は、観測された不足を解消する段階まで後ろに置く。実行前のモデル適合と実Targetの呼出し、独立Expected、Host Effect、Consumer受領は異なる資格のままとする。

現時点でPX-13〜22の実試験、異言語移植、Graph Writer、全Target適合、Owner意味受理は **NOT RUN / NOT ACCEPTED**。静的設計文書の変更から、製品機能の成立を主張しない。
