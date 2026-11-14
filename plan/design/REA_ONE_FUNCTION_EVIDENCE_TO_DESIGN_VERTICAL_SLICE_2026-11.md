# REAの観測から一機能の業務設計へ進む最小縦断実証

- **状態**：設計・試験引渡し案。新しい機能の実装完了、試験成功、Ownerの業務意味受理を意味しない。
- **更新**：2026-11-14
- **対象**：外部逆解析ツールREA（Reverse Engineer Anything）で得た保存済みEvidenceから、一機能のData・Function・Businessを根拠付きで確認し、必要なものだけTEI Sourceに採用する。
- **関連Owner**：REA #3394／R3-C #4661、三要素 #1124・#1125・#1126・#4487、意味参照 #1120、受理 #267、編集 #2497・#4515、Function実Read #2586、起動 #1158・#4495、独立Expected #2514・#1180。
- **優先順位**：非Plugin版Studio DesktopのStage1→Stage2実Windowsセルフビルド #4583/#2831、および通常tei CLIからの実PostgreSQL Provider→Function Context→独立Consumer #2586が製品P0。本実証を新たなHard Gateにしない。
- **採用しないもの**：外部Evidenceからの業務Meaning自動受理、一括自動移行、新Parser／Host／権限・承認Engine／第二Scenario正本、旧Runtimeの未観測動作の断定。

## 1. 一問で使い道を示す

最初の問いは**「旧システムのFunction Fは、どのDataを読み、どのDataを書き換えるか」**とする。利用者は、対象の候補、Sourceの位置、Task・Action・Function・入出力・Dataの関係、既知と未観測の範囲を確かめられる。

答えが候補の段階なら、その理由と追加調査の必要性を示す。正式な参照を作る前に、元の要素とTEIのQualified Refを独立に識別する。表示上の一本の矢印や自然な説明文を、当該Functionの実Read/Write、業務の完了、正式な業務規則に昇格させない。

## 2. 実験入力を混同しない

| 入力 | 根拠 | 実証できること | 実証できないこと |
| --- | --- | --- | --- |
| **A：Captured REA 6.8.0** | mainのR2が読む保存済みProducer Evidence。Observation 1件と有向Relation 1件を外部IDで選択し、実受領Byte digest、申告SourceRevision、Coverage/Unknown/Unselectedを保持 | 保存成果物のReader/Selector/Hostにおける実Byte取得、構造と出典の保持 | 旧X-TEA Functionであること、Task→Dataの全関係、旧実行・旧DB更新、Sourceの現在性、Owner Meaning・現在Grant |
| **B：公開可能な合成三要素Fixture** | Task T→Action A→Function F→IOTable I→Data D、Function参照がない手作業M、同名別IDのF2、並行する有向Edge、未取得Source U | Chain・参照Identity・未解決・問別Coverage・Previewの拒否を再現できる | Aと同じ実システムを解析したこと、旧X-TEAの原典を忠実に転記したこと、実Providerの動作 |

AのNode IDをBのFunction IDへ名前だけで結ばない。Aに業務Functionを認める根拠がなければ、問いへの回答は未確認とし、A0から先へ勝手に進めない。Bでは、Fの宣言上のReadとAction固有のWrite候補を分け、手作業MをFunction欠落や不要な作業とみなさない。SQL不在ならCreate/Read/Update/Deleteのいずれかを推測で埋めない。

## 3. 製品の段階と進行資格

| 段階 | 一巡で確認するもの | 主な担当 | 進むための条件 |
| --- | --- | --- | --- |
| **A0 外部観測** | 解析器Version・Source申告・実Byte・Node/Relation・元Source locator・Unknown・未選択数 | #3394、既存R2 | 対象VersionとProducer Deliveryを検証できる |
| **A1 実Consumer受領** | 通常CLIとNative Wails/WebView2の別Invocation、Host固定Selector・許可済みByte取得・用途別公開Slice | #4661、Draft #4625/#4630/#4637 | 独立した製品Consumerの実受領。試験用子Processを第二Consumerに数えない |
| **A2 一問の根拠付き回答** | 外部IDからSource Chainへ戻る操作、QuestionのRequired/Consumed、正・否定・Unknownの区分、0..N対応候補 | #1124/#4487/#3262 | 不完全な観測を示し、全称・不存在の断定を拒否できる |
| **A3 Owner意味受理** | 対象Source Slot・版・Qualified Ref・業務Rule・影響・Ownerの明示採用 | #267/#2497/#1124 | 編集先の一意性、現在版、正式な判断。閲覧だけでは採用不可 |
| **A4 記述・生成** | 同じAccepted Sourceからの図・業務説明・テスト・コード・SHO文書、保存・再読取・生成入力の一致 | DSL/Method/Generator/SHOの既存Owner | Sourceと意味・ToolchainとArtifact・実Consumerをそれぞれ照合 |
| **A5 実行・比較** | Legacy実測L、Owner期待D、新Target実測T、実Provider/DB・Effect・Recipientの別Receipt | #2586/#1158/#4495/#2514 | 同じCaseの独立Expected、必要な実作用と受領まで確認 |

**A2は、製品R3-C未完成でもBの合成Fixtureで並行して検証できる。** ただし、その成功をA1やA3以降の実達成へ転用しない。各段階はProducer/Reader/Consumer AttemptとSource・Model・Artifact版を別々に記録する。

## 4. 意味を保つために、答えられないことも示す

「FがDを参照する」という陽性主張には、当該Source・Relation・必要な条件を確認した根拠がある。「FはD以外を参照しない」「Dは未使用なので削除できる」といった否定・全称の主張には、対象のSourceとAdapter、動的SQL、未ロードPlugin、外部工程、RLS/LIMIT、必要な版を問別に列挙した完全性が要る。

Graphが局所的にcompleteでもFunction/BusinessのCoverageがunsupportedなら、業務全体の不在証明にはならない。取得不能、partial、permission-denied、stale、未対応は理由を別々に残す。既知の0件と未知は異なる。QueryのRequired(Q)/Consumed(Q)を照合するが、これを新しい全用途共通Schemaに固定しない。

外部Node/Relation ID、Source Token、TEI Qualified Key、物理DB PK、図のOccurrence/表示Edgeは別のIdentity。Task→Dataへ短縮して表示しても、元のAction/Function/IO/Source・関係の向きと平行辺・導出Ruleをたどれるようにする。Derived Edgeを業務のWrite Effectに読み替えない。Source Spanの構造が妥当でも実Subjectとの結び付きが不明なら、編集操作は拒否する。

## 5. CLI、Studio、AI、SUI、SHOの接続

**CLI・MCP**はHeadlessの検証入口とする。REAの通常CLIコマンドは現時点でDraft #4625の提案であり、mainの正式製品機能と呼ばない。Local CLIが受け取る任意ファイルPathを、そのままNative/Remoteの引数にしない。Native側はHostが固定したWorkspaceとSelectorからBounded Byteを取得し、既存R2の型付き投影を再利用する。外部Web/Agentへの送出はCurrent GrantとDisclosureの実受理を待つ。

**Studio**では、候補の列挙→元Sourceへ移動→関係Chainと未知の確認→比較→編集候補Previewの順に扱う。Focus、固定Selection/Pin、Dirty Buffer、Source改訂、Graphの描画Epochを同一視しない。View Bを閲覧中にPin Aの回答が後着しても、Bの編集状態へ適用しない。描画Edgeからの逆写像が曖昧ならread-onlyに留める。

**生成AI**は説明文・追加調査・テストパターンの候補を作れるが、SourceにないWriter、Actor、Business完了条件を確定しない。外部Evidenceを無条件送信せず、Hostが許可した用途別Sliceだけを共有する。日本語の説明では、①原典から確かめた構造、②解釈候補、③Ownerが採用した業務条件、④Mock結果、⑤実Runtime観測を混同しない。「しなければならない」など規範的な説明文は③の受理済み意味に基づく。

**SUI**は観測・候補の探索と整理、**SHO**は出典付きの記録・Review・文書保存、**TEI**は意味正本とSource・Generator/実行契約を担当する。SUIの配置やSHOの文章保存ではTEI Sourceは受理されない。SHOの実Save/Reopenは別Consumer受入として測る。

**Plugin**は外部Evidence取得、三要素のMethod/Projection、Target Generatorなどの既存Contract/Providerに寄与させる。外部REAの実行をTEI配布物の必須Runtimeにしない。標準利用者にDB AliasやSQL方言を常時選ばせず、必要な場合だけ開示可能な詳細へ進める。

## 6. 小さく切り出した作業と独立Expected

| 作業 | 入力と反例 | 最初に得る合格 | Owner・既存Expected |
| --- | --- | --- | --- |
| **V0-A Captured Reader** | 6.8.0の正規Raw、改変Raw、別Workspace Root、Producer Digestの再Pin | 正規Byteは取得、無効Deliveryは拒否。別Rootの選択・失効が混入しない | #3394/#4637、REA-XP-01/03 |
| **V0-B 合成Chain** | T→A→F→I→D、FunctionなしM、同名F2、並行Edge、未取得U | 元のChain/Relationへ復帰でき、短縮線からCUDを創作しない。未知を0にしない | #1124/#4487、REA-XP-07/08/12 |
| **V0-C 説明生成** | 静的な参照はあるが業務完了条件が未確認。AIが流暢な完了説明を提案 | Source観測と意味候補は読めるが、正式仕様やテストの期待値に勝手に採用しない | #1125/#2514、REA-XP-11 |
| **V1 二Consumer** | 実CLI、実Native Wails/JS、Server公開面、秘密Evidenceと別Principal | 実InvokerとRecipientごとに受領・非開示を独立確認。Go Testの子Processを二Consumerに数えない | #4661/#2516、REA-XP-01〜05 |
| **V2 Previewから保存へ** | 対象Slot変更、同名別ID、Source A→B→A、別Tab応答、Dirty Buffer競合 | SourceとOwner受理が確定した場合だけApply。旧Preview/別Viewへ誤更新しない | #2497/#4515、REA-XP-10/13 |
| **V3 Target照合** | 同名別SQL、NULL・粒度、RLS、Mock、Commit後の通知結果不明、旧Runtime未観測 | 実Query/Rowset/Context、実EffectとConsumerのReceiptを別票に残す | #2586/#2514、REA-XP-06/09 |

**最初の実装PRはV0-Bのみ**を基本とする。公開できる合成一Function・一Data・一手作業Actionと、反例の同名F2/並行Edge/未取得Uを、既存の型付き参照とSource投影で確認する。新しいReaderや別Graph Parserを作らない。続くV0-Cで説明文の出典と未受理の扱いを確認する。V0-Aは既存Capturedの検証を利用し、これらを一つの実システムの解析成功として束ねない。

各試験では、何を主張するか、独立した期待値、Required Source、実消費したSource、Attempt、受領者、欠落・矛盾、合格の上限を残す。負例を正しく拒否したことと、製品の実機検証が未実行であることを区別する。別PRの成功・別Fixture・後から読んだSourceを同一のProduct Attemptへ合算しない。


<!-- rea-v0b-identity-fidelity-plugin-share-20261114 -->
### 6.1 V0-Bの初回実装で検出する「関係ID」の不足

2026-11-14の実コード照合では、mainの **internal/modelref/model.go** にKey（Scope/Subject）、Ref、Node、Edge（From/To/Relation/Evidence）、Graph、Validate/Resolveがある。一方、**modelref.Edgeに独立したEdge IDはない**。同端点・同Relationの辺を配列で複数持てることと、各辺を安定して識別し、別の図/表/試験/編集対象へ往復できることは異なる。

既存のStudio Graphには描画用GraphEdge IDがあり、REAの保存Evidenceにも外部Relation IDがある。しかし、**描画Edge ID、外部解析Relation ID、正式な意味Relation ID、Source上のRelation occurrence、派生図のConnection IDは別Identity**である。どれか一つを別のIdentityとして無条件に流用しない。実装を始める前に、型と出典でどの情報を保持でき、どこがLossになるかを実験して明示する。必要なら#1120/#1124/#2497へ最小Contract変更を返すが、第二Graph/Parserや新しい全用途Schemaは採用しない。

[Plugin自己構築の具体例 #4675](https://github.com/hat47x/tei-platform/issues/4675)で検討中の、表示名がともにitemのNode **n1,n2**、同じ端点を持つ有向Edge **e1,e2**を、REA V0-Bの**公開合成Graph部分として共有**する。これに別の関係群としてTask T→Action A→Function F→IOTable I→Data D、手作業M、未取得Uを加える。合成Graphが外部REA Captured Evidenceと同じ旧システムを観測したものだとは主張しない。#4675のFixtureは実ファイル作成前の設計候補なので、現時点では実テストがあると扱わない。

| 変化を加える場所 | 独立Expectedで確かめること |
| --- | --- |
| n1とn2の表示名が同じ | Label一致でNodeを統合せず、Scope/Kind/IDを優先する |
| 同端点・同種類のe1/e2 | 辺の本数だけでなく、各Source occurrence/Relation IDへ個別に戻れるか調べる。戻れなければ**忠実度不足**として表示する |
| e2だけの向き反転・e1だけの削除 | 変更した辺と無関係な辺を区別する。個別Identityが不足するうちはread-onlyに留め、曖昧な編集を許さない |
| Graph上の配置だけ変更 | 意味、Codegen入力、元Sourceは不変。Layout-onlyで意味Buildや全証拠を過剰失効しない |
| Task→Dataの短縮表示 | T/A/F/I/Dの元Chain、方向、出典、導出Ruleへ戻れる。手作業と未知Sourceを省略や0件にしない |
| 旧Projectionが後から到着 | 現在のSource版/Consumer Snapshotと一致しなければ、別ViewのFocus/Pin/Dirty Bufferに混入させない |

**初回の着手順**：現行modelrefのNode/Edge/Evidence/Rule/Validateと既存Source Indexを固定Fixtureで確認する→独立Expectedと実結果の差をFit/Gapへ記録する→Relation occurrenceと出典を明示したread-only Projectionで保持可能性を検証する→必要最小限の型/Method契約変更を既存Ownerへ依頼する。配列添字、図の線番号、ラベル、Source Pathだけでは意味Relationの永続IDにならない。**不足を検出して編集を停止できたことも正しい試験結果**として扱う。

### 6.2 関係の理解と、TEI定義からの実処理生成を別資格にする

V0-Bは、読み取った関係のIdentity・意味・出典を失わず閲覧・説明できるかを検証する。#4675のPluginセルフビルドは、**受理済みで一意に識別できるGraph関係**から「Node n1の出辺Relation ID集合と件数」を実Goへ生成し、通常CLIから呼び出せるかを検証する。同じFixtureとExpected観点を使っても、REA側のRead合格だけで実Codegen合格にせず、Codegen合格だけで外部REA観測を正式Meaningともしない。

Plugin AがContractで公開したGraph/Rule/図解FacetをPlugin Bが再利用する際は、公開Scope・Contract版・必要な依存閉包・受理されたMeaning Ruleを確認する。**A→Bの参照許可は、B→Consumer Cへの自動再公開権限を含まない。** BのOverlayによってAの必須Guardを消したことにもできない。これはmainに統合済み[Plugin合成設計 #4671](https://github.com/hat47x/tei-platform/pull/4671)および#4675の責務を再利用し、REA側に別の権限Engineを設けない。

実生成では、e2の反転・削除が生成された出辺演算の結果へ実際に反映されること、Graphの配置変更だけでは意味Buildが変わらないことを確認する。TEIのSource/Ruleが実処理を決めず、Goの既存関数へdelegateするだけならSelf-build成果とは数えない。PluginのR0正規Source Read→R1実Go生成→R2 CLI実受領→R3生成物保全→R4 Stage1実再入→R5第二Consumerという別の実装順は#4675に保持する。最初のV0-BへR1以降をHard Gateとして加えない。

### 6.3 PRの完了範囲を具体的に固定する

最初のV0-Bの小PRが対象とするのは、**公開合成FixtureのNode/Relation/出典、元Chainへ戻れる範囲、同名異IDと平行辺の区別、未取得を未知のまま保つこと、識別できない辺のSource編集を止めること**。試験は#1180/#2514の既存Expectedを使い、Question・実消費Source/Rule・Observed/Unknown・個別Relation追跡可否・Loss・不合格理由を記録する。仮に現行modelrefだけではRelation identityの往復ができなければ、対象を減らして成功とせず、欠落を既存OwnerへのFit/Gapとして提示する。

初回PRでのNative REA実受領、正式Source変更、Pluginの実Go生成、実DB Effect、Desktop Stage2は**別資格**。初期セルフビルドはCLI/MCP先行でよく、Studio Desktop全体の実Windowsセルフビルドという製品P0は維持する。

**この節は実コード構造と別レーンの現行計画を照合して作成した設計・試験引渡しであり、V0-BやPluginの新しいGo試験、Sourceの往復、実生成・実Consumer受領を実行したものではない。**


## 7. TEI自身の自己適用とDSL簡素化を測る

次段階では、TEIの外部Evidence Operation→CLI/Host/Studioの実装を解析対象とし、設計・Source・コード生成・試験の対応を確認する。そのGo Sourceを即座に正式な業務DSLと扱うのではなく、選んだ契約のうち標準データ構造、図表、Function/Operator、Method/Generator Pluginで表現できる部分をTEIのSourceへ移し替える。

自己適用の前後で、(1)契約のうちTEIで直接記述できた意味の割合、(2)手書きGoに残る境界と理由、(3)生成／手書きコードの変更箇所、(4)Source変更時の再検証範囲、(5)図・DSL・説明・Target結果の意味差、を同じ固定Caseで比較する。行数の減少だけで成功判定しない。Goコードへの単純delegateではなく、図と標準データ構造がコード生成・検証に実際に寄与したことを確認する。PluginのTEI記述比率を高める方針も、この結果に基づいて段階化する。

当初のセルフビルドはCLI/MCP経由でよい。Studio上での操作を先行条件とせず、Go単一バイナリ・外部Host利用を守る。旧X-TEA/Swing/JavaScriptやREA解析器自体を、そのままCoreへ複製しない。

## 8. 完了・停止の判断

**V0**はCaptured Readerと別合成Fixtureの局所試験をそれぞれ実行した状態。**V1**は独立した通常CLIとNative Studioが実製品で別々に受領した状態。**V2**はOwnerの意味受理とSource更新・独立再読取・再投影を確認した状態。**V3**は適用範囲を明示して旧実測L・期待D・新実測T、実Provider/Effect/Recipientを独立照合した状態であり、全旧システムの同値性は主張しない。

EvidenceにないBusiness Ruleを付加する、unsupportedを不存在にする、Digest一致からGrantやMeaningを認定する、曖昧なGraphからSource変更する、旧Static Codeを実Invocationと呼ぶ、Mockを実Commitと呼ぶ、別Fixture/別PRの成功を一つに束ねる場合には、当該資格を停止して各Ownerへ戻す。

**現況**：R2のReadはmain統合済み。R3-A/Bは個別Draftで、R3-CのNative独立受領、V0-B/Cの新しいHeadless試験、Owner受理、SHO実Save/Reopen、旧新の実同値性は未実行。この文書も設計と実装分割の引渡しであり、新しいGo/Node/Windows/DB試験を実施していない。
