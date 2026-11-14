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

## 6. 検証すべき反例（既存Scenarioへの引渡し候補）

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

## 7. 実装計画と優先順

**P0：既存の製品縦断を先に成立させる。** #4495の実Entry→型付きInvocation→実Providerの二入口と、#4583/#2831のStudio Desktop Stage1→Stage2自己再構築を優先する。今回の図解仕様やGraph Compiler完成を、これらの新しい必須Gateにしない。

**P1：既存Graph/Sourceのread-only投影を試す。** #4198 R0の棚卸しとして、一つのpure Functionと一つのTriggerを対象に、Lifecycle／Entry／Operation／Response／Evidenceの関係を既存Graph型へ投影する。元Source、辺の識別子、Representation Loss、未取得依存を明示し、図のLayoutのみ変更した場合の非再生成を検査する。Studio/CLIの固定選択・ソースジャンプは#1124/#4515へ接続する。

**P2：二つの異なるTargetで意味保存を比較する。** 同一Accepted Meaningの一Functionを、まず実Go CLIと既存Studio/Hostで検収し、続いて異なる言語または異なるLifecycle（例：常駐サービス対単発実行）へ適用する。差が出た箇所をRole不足／Target Binding／Provider／Host Authority／生成器不足へ分類し、差が消えるまで「移植可能」と断定しない。

**P3：編集・再生成と応用を検討する。** #4198 R1〜R4、#4193の異質DomainのPressure、#3503のVUI、既存HTTP応答Decision Graph、Batch/ETL、SHO文書生成へ、必要なProjectionだけを広げる。図編集からSource Change Proposal、受理、再生成、実Consumer確認までの可逆性が確認されたものだけ書込編集へ進める。

採否の停止線は、第二Host/Runner/Resolverが必要になる、現在Grantを迂回する、図からAuthoritative Sourceへ戻れない、Plugin登録順で意味が変わる、異質Target間の不足を隠す、未実行なのに製品合格を主張する、のいずれかである。

## 8. 各担当への引渡し

- **#4198**：共通Lifecycle/Control/Effect Graphの最小Roleと、Graph Template + Obligation + Plugin差分が既存標準Graphに収まるかを評価する。正式Schema/Graph Compilerの採用は保留。
- **#4502/#4495/#4497**：公開EntryとTriggerを実Invocationへ接続し、実Call・入場資格・再入・配送結果を検収する。新SchedulerやScript Runnerを作らない。
- **#2620/#988/#1121**：Data＋意味Role＋DSLの糖衣＋Target Loweringを分離し、評価順やRuntime能力を文法に埋め込まない。生成Goを業務意味の正本にしない。
- **#1124/#4193**：図・表・Traceの相互参照、選択の固定、Representation Loss、変更Proposalを担当する。State Machineの採否は異質Domain比較後。
- **#1129/#1174/#2586/#2514**：Effect/Receipt/Provider Read/独立Expectedを各Ownerの検証に接続する。HTTPステータス、ACK、Business Outcome、永続作用の資格を分離。
- **#4583/#2831**：Source→Build Intent→Builder実行→Artifact→Stage2 Desktop→実Windows Consumerの現在の最優先経路を維持する。図解構想を理由にセルフビルドへ別Runtimeを追加しない。

この設計案の統合は実装や採用の完了ではない。次の判断は、#4198 R0のread-only実証と#4495/#4583の独立した製品検収で得られる不足点を見て、既存Ownerが行う。