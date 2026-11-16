# REAの観測から一機能の業務設計へ進む最小縦断実証

- **状態**：設計・試験引渡し案。新しい機能の実装完了、試験成功、Ownerの業務意味受理を意味しない。
- **更新**：2026-11-15
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
| **A1 実Consumer受領** | 通常CLIとNative Wails/WebView2の別Invocation、Host固定Selector・許可済みByte取得・用途別公開Slice | #4661、#4625/#4630/#4637（2026-11-14 main統合済み） | 独立した製品Consumerの実受領。試験用子Processを第二Consumerに数えない |
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

**CLI・MCP**はHeadlessの検証入口とする。REAの通常CLI `tei external-candidates` は#4625によってmain統合済み。ただしローカルPathを許容する入口であり、Native/Remoteへその引数契約を持ち込まない。Local CLIが受け取る任意ファイルPathを、そのままNative/Remoteの引数にしない。Native側はHostが固定したWorkspaceとSelectorからBounded Byteを取得し、既存R2の型付き投影を再利用する。外部Web/Agentへの送出はCurrent GrantとDisclosureの実受理を待つ。

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

[Plugin生成の設計 #4689](https://github.com/hat47x/tei-platform/pull/4689)で整理した、表示名がともにitemのNode **n1,n2**、同じ端点を持つ有向Edge **e1,e2**を、REA V0-Bの**公開合成Graph部分として共有**する。これに別の関係群としてTask T→Action A→Function F→IOTable I→Data D、手作業M、未取得Uを加える。合成Graphが外部REA Captured Evidenceと同じ旧システムを観測したものだとは主張しない。#4733で参照Graph/Rule→Go生成Binaryの局所的な実Node選択はmainへ統合されたが、REA Captured Evidenceからの自動受理・製品Plugin起動は未証明である。

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

V0-Bは、読み取った関係のIdentity・意味・出典を失わず閲覧・説明できるかを検証する。Pluginセルフビルドの後続縦断は、**受理済みで一意に識別できるGraph関係**から「Node n1の出辺Relation ID集合と件数」を実Goへ生成し、通常CLIから呼び出せるかを検証する。同じFixtureとExpected観点を使っても、REA側のRead合格だけで実Codegen合格にせず、Codegen合格だけで外部REA観測を正式Meaningともしない。

Plugin AがContractで公開したGraph/Rule/図解FacetをPlugin Bが再利用する際は、公開Scope・Contract版・必要な依存閉包・受理されたMeaning Ruleを確認する。**A→Bの参照許可は、B→Consumer Cへの自動再公開権限を含まない。** BのOverlayによってAの必須Guardを消したことにもできない。これはmainに統合済み[Plugin合成設計 #4671](https://github.com/hat47x/tei-platform/pull/4671)および#4689/#4733の成果と責務を再利用し、REA側に別の権限Engineを設けない。

実生成では、e2の反転・削除が生成された出辺演算の結果へ実際に反映されること、Graphの配置変更だけでは意味Buildが変わらないことを確認する。TEIのSource/Ruleが実処理を決めず、Goの既存関数へdelegateするだけならSelf-build成果とは数えない。PluginのR0正規Source Read→R1実Go生成→R2 CLI実受領→R3生成物保全→R4 Stage1実再入→R5第二Consumerという別の実装順はPluginの既存設計に保持する。最初のV0-BへR1以降をHard Gateとして加えない。

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

**現況（2026-11-15確認）**：R1 #4547、R2 #4548、R3-A #4625、R3-B #4630/#4637はmain統合済み。R3-CのNative Wails JSによる第二Product Consumer独立受領、V0-B/Cの新しいHeadless実行、Owner意味受理、SHO実Save/Reopen、旧L/D/T実照合は未受入。個別PRのCI成功と統合mainでの製品縦断成功を区別する。この文書も設計と実装分割の引渡しであり、新しいGo/Node/Windows/DB試験を実施していない。

## 9. 2026-11-15横断収束：R3-A/B統合後の不足辺と三要素取り込み

### 9.1 実装済みの線と、まだ接続されていない線

2026-11-14の統合PR [#4683](https://github.com/hat47x/tei-platform/pull/4683)を確認すると、外部Evidence受領R1 [#4547](https://github.com/hat47x/tei-platform/pull/4547)、型付きread-only R2 [#4548](https://github.com/hat47x/tei-platform/pull/4548)、ローカル実CLI R3-A [#4625](https://github.com/hat47x/tei-platform/pull/4625)、byte-only R2入口 [#4630](https://github.com/hat47x/tei-platform/pull/4630)、Host固定Selector/Root/bounded read [#4637](https://github.com/hat47x/tei-platform/pull/4637)はmain統合済みである。

次の製品不足辺は、[R3-C #4661](https://github.com/hat47x/tei-platform/issues/4661)の**Native Wails Service経由で実Windows WebView2 JSが候補を受領すること**。通常CLI子Processの成功や、GoからReaderを二度呼び出した結果はNative第二Consumerの証拠にしない。Hostの固定Workspace/Selectorから取得したbyteだけを既存R2へ渡し、RendererへEvidencePath、CAS/NAS/S3 locator、任意Path、Principal自己申告を渡さない。Native入口追加後に `-tags server` の実Service/Routeへ露出しないことと、公開Reply/DOM/ARIA/ConsoleがPrivate情報の有無で区別できないことも独立に検証する。

すでにmainへ入ったR3-A/Bを再実装するPRは起こさない。#4661を実際のNative登録・独立受領・開示境界へ限定し、Owner Meaning、正式Source編集、SHO Writer、実DB移行は別の検収とする。

### 9.2 原本・観測・対応付け・正式意味を段階別に扱う

| 層 | 何を保持するか | 許されない短絡 |
| --- | --- | --- |
| **原本／Producer Evidence** | Raw byte、文字コード、Tool/Protocol版、取得Attempt、Source locator/Revision、Coverage/Unknown、元のbyte digest | 変換後JSONのdigestを元Rawの同一性に代えること |
| **構造観測** | Observation Node ID、方向付きRelation occurrence ID、端点・種類・StructuralFacts、Producer Source範囲 | 同名Node統合、配列位置・Graph描画Edge IDを正式Relation IDとみなすこと |
| **対応付け候補** | Data/Function/Businessへの根拠付きMapping、出典、複数候補、欠落・矛盾、Question別Required/Consumed | 観測されたDataだけでFunction/Businessの不存在や網羅性を確定すること |
| **Owner受理済みTEI Source** | 確定した業務意味、対象Qualified Refと版、Rule、操作の効果、受理記録 | AIの説明、REA結果、候補のPreviewを業務仕様・実行許可へ昇格すること |
| **生成／Target実測** | Generator/Artifact/実Provider、Source closure、実行・Consumer Receipt、旧L/期待D/新Tの独立Expected | 生成成功を実使用成功、静的SQL一致を旧新Runtime同値とみなすこと |

REA 6.8.0の保存済みUTF-16LE/BOM入力からUTF-8の取込み用派生を作る場合、**元Rawと派生は別byte provenance**として追跡する。[三要素取り込みDraft #4693](https://github.com/hat47x/tei-platform/pull/4693)のJ-03bは、Captured Evidenceに対してData=supported、Function/Business=unsupportedとOwner pendingを確認する設計・回帰である。これは外部REAが三要素全体を解析した証拠ではない。#4693の最新HEAD CI、Native Consumer、Owner受理、実Effectは別途確認する。

### 9.3 Relation同一性と再解析の扱い

三要素Modeler、Graph/Plugin、Source編集で共通に使う公開合成例は、同表示名のNode `n1,n2`、並行する有向Relation `e1,e2`、Task→Action→Function→IOTable→Data、手作業M、未取得Uとする。Captured REAの実観測と、この合成Chainの意味を混ぜない。

- **J-02h〜j**：同じ構造・Source Spanの並行Relationが再採番された場合、ソート順による対応は暫定表示だけ。1対2、2対1、2対2のRelation occurrenceの継続性を確定せず、Source-backed対応が得られるまで編集は止める。
- **J-02k〜m**：Candidate ID/Source Digestが同じでも、`ObservedKind/State/Authority/Coverage`等の外部観測事実が変われば `observed-facts-changed` 等で再評価を促す。新Statusを未知のConsumerが`unchanged`へ黙って丸めない。
- **構造上の損失**：外部Relation ID、TEIのMeaning Relation occurrence、画面の描画Edge、物理DBキーは別の識別子。往復が不可能ならread-only＋Source navigateへ退き、Map/Editorに一意な編集対象があるかのように見せない。

上記の差分検知は[Draft #4693](https://github.com/hat47x/tei-platform/pull/4693)と[Modeler #1124](https://github.com/hat47x/tei-platform/issues/1124)、[Graph #1120](https://github.com/hat47x/tei-platform/issues/1120)の責務を再利用する。正式な第二Graph RegistryやREA専用Canonicalを作らない。旧Statusを読むCLI/Studio/SHO/他Consumerの未知値扱いも受入へ追加する。

### 9.4 三要素の受入は「見つかったもの」ではなく「問いと必要な根拠」で決める

三要素分析法の**データモデル・機能モデル・業務モデル**を忠実に扱い、Task/Action Tree、I/O表、CRUD/逆引き、業務フロー、Subsystemの関連を省略しない。REAの静的参照は候補に留め、`Task→Data`の短縮表示から隠されたFunctionやWrite Effectを生成しない。

1. **P（観測）**：観測したSource断片、SQL/Call/画面、Relation、取得Attempt、未取得範囲を提示する。
2. **J（対応付け）**：各観測がData/Function/Businessの何に対応するか、0/1/複数・不明と根拠を提示する。業務上のRole・完了条件は候補である。
3. **Q（問いに対する充足）**：問いごとにRequired Source/Relationと実Consumedを照合する。「FがDに参照する」の陽性証拠と、「Dはどこからも参照されない」という否定・全称の完全性証明を別にする。Data部分がcompleteでもFunction/Businessがunsupportedなら「存在しない」と答えない。

意味受理を経た後に限り、受理済み`*.tei.md`や標準表現から図・説明・独立Expected・Code Generatorへ接続する。複雑な業務条件はBRMS風のRule/Operatorを標準表現に重ねるが、REAを別BRMS/Rule Engineにしない。

### 9.5 最小検収順と製品優先順位

| 順位 | Owner・対象 | 次の確定条件 |
| --- | --- | --- |
| **独立P0継続** | [#4583](https://github.com/hat47x/tei-platform/issues/4583)/[#2831](https://github.com/hat47x/tei-platform/issues/2831)：Studio Stage1→Stage2実セルフビルド、[#2586](https://github.com/hat47x/tei-platform/issues/2586)：通常tei実PG Query→Function Context | 正式Build/実Consumerと実Provider/Rowsetをそれぞれ独立に実証。REA R3-Cや三要素全量解析をHard Gateにしない |
| **REA-R3-C** | [#4661](https://github.com/hat47x/tei-platform/issues/4661) | 同一Captured Producer Evidenceに対し、実CLIとWindows Native Wails JSが別Invocation/Receive。Host Selector/公開面/後着応答の負例に合格 |
| **V0-B→V0-C** | [#1124](https://github.com/hat47x/tei-platform/issues/1124)、[#4693](https://github.com/hat47x/tei-platform/pull/4693)、[#2514](https://github.com/hat47x/tei-platform/issues/2514) | 合成T/A/F/I/Dと外部Capturedを区別したまま、関係ID・再解析差分・P/J/Q・説明の根拠をread-onlyで検証 |
| **V2** | Owner/Source Acceptance、SHOのSave/Reopen | Qualified Ref/Source Revision/複合Identity・受理済みMeaningからのみSourceへ適用。再読取・再投影で一致 |
| **V3** | 旧実測L／Owner期待D／新Target実測T | 一Functionだけを同じ粒度・時点・Scope・Null/欠損・実Effect/Receiptで照合。静的解析だけのLやMockのTは合格にしない |
| **後続の自己適用** | TEI-native Plugin/GeneratorとCLI-first Dogfooding | REA→受理済みGraph/Rule→実Go生成→実CLI受領を段階別に実証。単なるGo delegateを自己実装に数えず、定義の簡素化・変更箇所・再検証範囲を測る |

各試験で **tested HEAD、Producer/Reader/Consumer Attempt、Source/Tool/Generator版、Expected、実Result、未受理範囲** を分けて記録する。既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)のExpectedと既存Ownerへ分配し、新しいREA専用Runner/Gate/Host/意味正本は設けない。

**この追補の証拠上限**：GitHubの統合状態・既存Source/PR/Issueの静的照合に基づく計画更新。新たなGo/Windows/PG・SHO・Source編集試験を実行した結果ではない。統合mainの更新後実E2E、#4693最新HEAD CI、R3-C Product受領、正式Owner意味受理は未検収。

## 10. 2026-11-15追補：根拠付き業務設計とTEI自己構築を結ぶ

この節は、前節のREA R3-A/B統合状態とJ-01〜05／V0〜V3を前提に、SUI・SHO・三要素分析法・標準Graph/DSL・Rule・Plugin生成・Studio・実Providerの**引継ぎ条件だけ**を明確にする。詳細な機能仕様や独立Expectedの正本は各既存Ownerに残し、REA専用の新しい意味正本・万能Parser・Writer・Host・Gateを作らない。

### 10.1 入力媒体は異なっても、業務の意味を急いで一つにしない

業務設計の材料は、既存ソースやREAの出力に限らない。例えばホワイトボード写真、会議文字起こし、表計算・CSV、既存帳票・画面、DB/SQL、Markdown/図表を含む。各入力にはそれぞれ原本の位置・取得版・取得時点・Toolと、抽出できる構造の範囲がある。

**設計方針**：各媒体の観測を既存のEvidence/Source occurrenceとして保持し、読み取り専用の三要素Candidate/Reviewへ渡す。REA（Reverse Engineer Anything）を全媒体のExtractorとみなさない。[他媒体の設計Draft #4716](https://github.com/hat47x/tei-platform/pull/4716)は独立レーンの設計候補であり、実写真／音声／Excel解析成功や正式TEI Source受理を意味しない。

一つの資料に「受注→在庫」という手描き矢印があり、会議記録に「引当は承認後」という発言があっても、二つのObservationから即座に在庫更新の実Effectや完了条件を決定しない。矢印が画面遷移か依存か更新か不明なら、その不明さを保持し、必要な質問を返す。多数のAI要約が一つの原本を反復していても、多数の独立証拠には数えない。時刻が新しいだけで原典の権威や正しさを確定しない。

SUIは探索・配置・紐付け、SHOは著者の原文・出典・レビュー・保存文書、TEIは受理済み業務意味・Source・生成・実行契約のOwnerとする。同じObservationを参照しても、SUIのItemやSHOのDraftをTEIの業務意味正本として自動登録しない。共有文書管理基盤の物理実装や新しい万能Intake層を最初の縦断条件にしない。

### 10.2 「何を答えられるか」をQuestionと根拠閉包で判断する

最小質問は、既存§1の「旧Function Fは何を読み、何を書き換えるか」で維持する。ただし回答は、**確認できた参照／対応付け候補／確定していない範囲／次に必要な確認**を区別した表示とする。新しい普遍Schemaを先に定めず、既存ReviewPacket・ThreeElementCoverage・SourceIndex・Model Map・独立Expectedの投影で表現できるかFit/Gapする。

| 質問 | 回答可能となる根拠 | 根拠不足のときに返すこと |
| --- | --- | --- |
| FがDを静的に参照するか | 特定Source occurrenceと有向Relation、対象F/DのIdentityが追跡できる | Functionへの対応付け不明、関係曖昧、Source未取得 |
| Fが実際にDを読むか | 実Function／Query／選択ProviderのInvocation、対象DB/params、Read結果の同一Attempt | 静的参照のみ、実Read未観測。Read候補として表示 |
| FがDを書き換えるか | 更新SQLやAPIの構造観測と実Effectの区別。実Writeを述べるならCommit/Observer/Receiptが必要 | 変更の可能性と実更新を分離。Mock/previewを実更新と呼ばない |
| Dは未使用・削除可能か | 利用範囲の完全性、動的参照・未読込Plugin・権限付き不可視データ等の必要Sourceを質問相対に閉包 | 部分的な0件は「未使用」を証明しない。判定不能 |
| Fの業務完了条件は何か | 三要素の業務モデル・Function/Action/RuleとOwnerが受理した意味 | 外部解析や自然文の仮説として提示し、正式仕様にしない |

問いごとに必要なSource/Relation/Ruleの集合（Required）と、実際に参照・使用した集合（Consumed）を対照する。Requiredに欠落・権限拒否・不明がある場合は、その範囲で結論を制限する。観測されたGraphだけがcompleteでもFunction/Businessのcoverageがunsupportedなら、全業務の網羅や不存在は証明されない。

REAの外部EvidenceとTEI Productの実Rowsetを無条件に一つのAttemptへ合算しない。受理の判断は、正しいデータの有無だけでなく、**どの質問に対して何の証拠が必要か**で変わる。

### 10.3 Candidateから正式Sourceへ進むための操作順

観測候補は、作業上の理解を助けるだけでTEIの定義を変更しない。既存のtyped change／Preview／Acceptance／Source Save/ReReadを用い、次の順序で扱う。

1. **候補の固定**：外部Rawの取得版と受領byte、Producer/Reader Attempt、候補IDとRelation occurrence、Source Revision、Coverage/Unknownを選択時のEvidenceとして残す。別Workspace・別Root・別Producerとの取り違えを拒否する。
2. **意味の比較**：Data/Function/Businessへの対応、変更対象のQualified Ref、業務上の解釈、未解決・矛盾する根拠を提示。AIの説明や過去のOwner判断を現在の自動承認とみなさない。
3. **変更案のPreview**：編集可能な正式TEI Source上の一意なSlotと現在Revisionが確認できた場合だけ、既存の編集機構で差分候補を作る。Sourceへ逆写像できないGraph、平行Relation再採番で個別対応が不明な関係はread-onlyに留める。
4. **Owner受理と保存**：明示的に受理された意味だけをSourceの現在の受理経路へ通す。保存後は実Readerによる再読取・再投影・対象選択の一致を別に検査する。SHO Save/Reopen、実生成・実Effect、現在Grantはさらに独立した受入である。
5. **再評価**：Source A→B→A、Tool/Provider版差、観測事実のみの変更、Grant変更、Focus/Pinの更新は、それぞれ影響する候補・説明・表示・実行資格を別に再評価する。Digestが元に戻ったことだけで古い受理、開示、実行許可を復活させない。

編集画面でAを固定表示中にBを閲覧し、後着のAの照合結果を受け取ってもBのFocus/Dirty Buffer/Graphを変更しない。内部EvidencePath、Private Sourceの件数・名称・Digestを許可されないRenderer/Network/AI/SHOへ露出しない。既存#4515/#2497/#2516/#4503の境界を再利用する。

### 10.4 構造Identity、観測事実、資格を三つに分ける

[三要素Candidate Draft #4693](https://github.com/hat47x/tei-platform/pull/4693)では、Source DigestとCandidate IDが同じでも、`StructuralFacts`のラベル／状態／Authority／Coverage等が変わる可能性を負例化している。Draftの`observed-facts-changed`は**比較出力の読み取り専用分類**であって、新しいMeaning/Source Revisionではない。下流のStudio/SHO/CLIで未知の分類を`unchanged`へ黙って丸めず、当該説明の再評価を要求する。

関係については、次の区別を維持する。

- **Identity**：外部Evidence ID/Node ID/Relation occurrence ID、TEI正式Meaning ID、物理DBキー、描画Edge IDを別々に識別する。同名異ID・並行Edge・端点変更・1対N/N対1再対応を個別検査する。
- **Observed Facts**：同じIDであっても、Direction/Relation/State/Coverageが変わり得る。単なる表示変更と観測された構造事実の変更を分離する。
- **資格**：Source現在性、Producer delivery、Operator/Provider版、Owner意味受理、Grant/Disclosure、Consumer実受領、Target実Effectは独立する。同じGraph表示やbyte一致をすべての資格へ転用しない。

二時点・組織関係・会議発言では、業務上の有効時点と「いつその情報を知ったか」を必要に応じて分ける。既存の時系列型／Relation属性で足りるかを先に調べ、全用途へ新しい二時点ストアを強制しない。

なお**Resource–Event–Agent**のREAは、業務・会計のResource、Event、Agentを捉えるモデルであり、本節の外部逆解析ツールREAとは無関係。三要素のBusiness候補を補助する選択的Projectionとして考えられるが、外部解析のNode/EdgeをResource/Event/Agentへ自動変換しない。また、業務上のAgentとAI HarnessのAgent、組織の職務・権限も混同しない。

### 10.5 Graph/DSL/Plugin自己構築へ接続する検収線

実用上の成果は、逆解析で関係を閲覧できたことではなく、**受理済みのTEI定義から必要な実処理を生成し、その結果をCLI/Consumerで確認できたこと**で測る。

mainへ統合済み[#4733](https://github.com/hat47x/tei-platform/pull/4733)は、参照用Graph/Rule→生成Go Binaryに`-node <id>`を与えて実Edge IDを返す限定的な**Reference段階の実証**である。生成Binaryを一度Buildし複数の実Invocationを検証するが、正式`*.tei.md` Reader、製品`tei` CLI、実Plugin selected→invoked→received、Stage1→Stage2 Desktopは未証明。#4733を後段の利用可能な証人とし、前段のREA Candidateから直接Generatorへ渡してOwner受理を飛ばさない。

次の接続順を固定する。

- **Graph Read**：公開合成n1/n2/e1/e2とT→A→F→I→D・手作業・未取得Uの読み取り専用Projected Relationが、元Occurrence／方向／多重性へ戻れるか確かめる。戻れない辺は編集・生成の適格性を認めない。
- **正式TEI Source**：受理済みGraph/Ruleを正式`*.tei.md` Reader/Model Reference/Generatorへ渡す。図表やRuleが実処理を決め、Go委譲だけのWrapperではないことをIndependent Expectedで確認する。
- **生成・起動**：同じSource/Generator契約からArtifact/実Builder／Providerを選択し、実CLIが当該Artifactを受領するまで段階別Receiptを残す。生成結果とRuntimeの結果が偶然同じでも、入力依存関係の実消費を別に確認する。
- **再生成と第二用途**：e2反転・削除、Layout-only、Rule変更、生成物の手修正、A→B→Aの負例を用い、意味Build失効と図の描画変更を区別。Studio/Plugin別用途は先行CLI成功と独立に実受領を確認する。

実装比率は、TEIが生んだ業務・演算Contractの割合、TEIが制御したBuild工程、Plugin自身の再生成範囲、定義の重複・保守負荷を**それぞれ分母付き**で比較する。生成Go行数や既存Goへの委譲回数を単独のセルフビルド率とはしない。#943/#951/#1016/#1121/#1251/#2090/#2831をOwnerとし、REA側へ第二Generator/Build Runnerを設けない。

### 10.6 既存Scenarioへ渡す追加の独立Expected

下表は新しいテスト正本ではない。既存#1180/#2514のScenarioと、#4693のJ-02/J-03、#4661のXP、Plugin生成の既存負例へ不足Assertを渡す作業索引とする。

| 追加Assert | 反例 | 合格の上限／担当 |
| --- | --- | --- |
| **媒体間の矛盾** | 手描き受注→在庫と「承認後に引当」の会議文が不整合 | 異なる原本Occurrenceと解釈候補を保持。Effect未認定（#3394/#4716） |
| **証拠の非独立** | 同じ会議原文のAI要約を3件添付 | 3つの独立出典として数えず、元原文と派生を区別（#3394/#2514） |
| **質問別閉包** | Data complete、Function unsupported、未読込Pluginあり | 参照の陽性候補は示せても未使用／削除可能を断定しない（#1124/#2514） |
| **再解析した同じID** | Digest/ID固定で`StructuralFacts`変更、未知Status Consumer | 再評価が必要と伝え、`unchanged`扱いを拒否（#4693/#1124） |
| **関係の再対応** | 同表示名・平行二辺の両方を再採番／1対2／2対1 | 自動で正式Relation IDを割り当てず、編集を停止（#1120/#2497） |
| **選択と公開の独立** | Reader Pin一致だがGrant取消、NativeのErrorでPrivate Pathが漏れる | read-only選択成功をCurrent Disclosureとみなさない（#4661/#2516） |
| **生成の実利用** | Go Wrapperだけ、Provider Bを選択し実CallはA | 自己実装と呼ばず、selected/invoked/receivedを分離（#1251/#2831） |
| **旧新の同値誤認** | 静的SQLが同じだがRLS/params/粒度が違う。旧Runtimeは未観測 | 旧L/期待D/新Tを別票で比較、旧実行成功を推定しない（#2586/#2514） |
| **A→B→A再資格化** | 原byte・Graphが戻るがSource履歴／承認／開示が変化 | 対象Receiptと必要資格を再評価し、旧Approval/Grantを自動復活させない（#2600/#2516） |

### 10.7 次の小PRに分割する順序

1. **既存P0は維持**：#4583/#2831の実Windows Stage1→Stage2 Desktop、#2586の通常`tei` CLI実PG Query→Function Context→独立Consumerを最優先とする。外部REA Candidateや多媒体取込みの完成を新Hard Gateにしない。
2. **REA Native実受領 #4661**：統合済みR3-A/Bを再利用し、実Wails Service→WebView2 JSで一Observation/一有向Relationを受領。CLIの独立実Invocationと比較し、Server buildに入口を漏らさず、許可外Disclosure・後着・Root差替えを拒否。
3. **三要素Headless #1124/#4693**：一Functionの質問に対するJ-01〜03の結果を、Candidate／Relation occurrence／Question Required/Consumed／未受理で投影。#4693の新分類をConsumerが扱えるかは別検収。先にOwner受理やSource Writeを作らない。
4. **正式Sourceへの限定Apply**：現行Readerが対応する、Owner受理済み・対象一意・Source Revision一致の**一件だけ**を既存Editor/Acceptanceへ通す。再読取とSHO Save/Reopenはそれぞれ独立ConsumerのReceiptで検収する。
5. **TEI自身への自己適用**：受理済みGraph/Ruleの一演算から実Go/Plugin生成・通常CLI受領・再生成までを繋ぎ、手書きGoへのDelegateと区別した四指標を測る。旧L/期待D/新Tの実比較は最初の一Functionに限定する。

**今回の変更の証拠上限**：2026-11-15時点で参照できる既存Issue/PRと設計・現行Product境界の照合に基づく計画追補。ここで新しいGoテスト、Windows Native Wails JS受領、SHO実Save/Reopen、PG Write、正式`*.tei.md` Reader、Owner意味受理を実行・認定したものではない。#4693/#4716はDraftのため最新HEADの全CI結果と統合判断は別に必要。既存のCI成功・個別PRの実証と新たな同一Attemptの製品縦断成功を混同しない。

## 11. 最小縦断の実装・検収票（後続PRへ引き渡す具体例）

この文書自体は設計の引継ぎであり、新しい試験の成功を主張しない。実装時には一件の問い「旧Function FがDをRead/Writeするか」に対し、既存のIssue #2514/#1180に登録した独立Expectedを用いる。次の内容を一枚の検収票で相互参照できるようにする。

| 検収欄 | 記録する内容 | 確認する誤認 |
| --- | --- | --- |
| Questionと必要根拠 | 参照、実Read、実Write、未使用、業務完了条件のうち回答する問い。Required Source/Relation/Rule | 別の問いの根拠を流用しない |
| Producer/Source | Workspace、Source locatorとRevision、Raw digest/Encoding、Evidence ID、Tool/Provider版、取得時点 | 同Byteの別Scope、元RawとUTF-8派生の混同 |
| Relationと意味 | 外部Node/Edge occurrence、方向、多重性、原Sourceへの戻り先、三要素の対応とUnknown | 平行Edgeを集約しない、静的参照を更新Effectにしない |
| 比較対象 | Candidate/正式Meaning、Accepted Source/Generator、旧L・期待D・Target Tの別Revision | Candidateを正式仕様とせず、静的SQLをLegacy Actualにしない |
| Consumer/実行 | 実CLI・Native JS等の独立Invocation/Receiptと開示Scope、Provider selected/invoked/received | 試験用子Process、Mock、別Attemptの成功を合算しない |
| 結論と停止理由 | Supported/Unsupported/Partial/Unknown、未確認事項、Owner判断、次の最小作業 | 0件と不存在、Promptによる承認、Digest一致によるGrant復活を認めない |

**最小正例**は、公開合成のT→A→F→I→Dと、異なるCaptured Producer Evidenceの一Observation・一有向Relationを別々に検査し、双方が出典と方向を失わない範囲を示すこと。**独立負例**は、同名異ID、e1/e2並行関係の再採番、Function/Business unsupported、観測事実のみの変更、Source A→B→A、選択Provider B/実Invoke A、未受理Sourceからの生成を取り上げる。局所Graphの完全性から全体の不存在を認定しない。

R3-CのWindows実Wails Consumer、正式Source Apply、実PG Function Context、Generator→実Plugin、SHO Save/Reopenはこの検収票の**別の行・別のAttempt**として段階的に追加する。最初のHeadless試験が成功しても後続を完了扱いにせず、必要な分だけ既存Ownerへ引き渡す。

**統合上の判断**：旧PR #4603はmain進行により大幅に遅れたため、本書の成熟した部分だけを最新main基点で移した。ほかの開発レーンによるMASTER_PLAN/READMEの追補は維持し、旧ブランチの大量の古い差分を再適用しない。

## 12. 最初の一問をHeadlessで検証する：現行Product型からの実装単位（2026-11-15）

この節は前節の設計を現行mainの実コードに照らして検収可能な単位へ落としたもの。最初の問いは「旧Function FはData Dを実際に読むか」。構造観測で答えられる部分と、実Read・業務意味が未確認の部分を分ける。既存P/J/Qは評価観点であり、別の共通Engine／Response Schema／Runnerを追加する提案ではない。

### 12.1 現行の二つの入口と資格の差

| 現行の型と入口 | 保持できる事実 | これだけでは確認できないこと |
| --- | --- | --- |
| 通常CLI external-candidates → operations.ExternalCandidateView | 保存済みRawから明示IDで選ぶObservation 1件と有向Relation 1件、Attempt.EvidenceID、ReceivedDigest、申告SourceRevision、Tool版、Coverage/Unknown、Unselected件数、AuthorityLimits | 旧業務Functionとの対応、実Read/Write、完全なGraph、現在Grant、正式意味 |
| three-element rea-intake → reacandidate.ReviewPacket | CandidateのStructuralFacts/Inference/SourceRevision/MeaningLoss/Unresolved、Carrier、ThreeElementCoverage、Owner未受理状態 | 正式Source変更、業務の完了条件、実DB Effect、独立Native Consumer |
| reacandidate.EvaluateThreeElementCoverage | 現行の静的REA候補についてDataの構造候補をsupported、Function/Businessをunsupportedと理由・wouldNeedで区別 | 旧システムにFunction/Businessが不存在という証明や、全Sourceの調査完了 |

現行ElementCoverage.Statusの二値はsupported/unsupportedであり、「その三要素の候補を当該Evidenceが支持できるか」の意味に限る。External Evidenceのpartial、取得0件、Reader not-attempted、permission-denied、Source Binding、実ProviderのRead資格をこの二値に押し込まない。将来の未知Statusも、supportedやunchangedへ黙って丸めない。

external-candidatesはProducerの保存済みRawを読む。他方、rea-intakeにUTF-16LE/BOM原本からUTF-8派生を与えるときは、**元Rawと派生byteのDigestが異なることを前提**とする。Producer Evidence ID、元Raw、派生変換、内容の同値、両Reader受領、Consumer Attemptを別々に照合できた場合だけ、同じProducer観測に由来する二つのProjectionとして比較する。似たResultや同じID文字列だけで同一Read Attemptとしない。

### 12.2 初回の回答の書き方と停止条件

Headlessでは、既存Projectionの読取結果から次の構成の**回答候補**を作れるかを試す。恒久的なJSON Schemaを新設しない。

> **確認できた構造**：Producerが報告した特定Source上のNode A→Node BというRelation e1。外部ID、方向、出典、申告版、選択されなかった関係数を保持。
>
> **三要素への対応候補**：Dataの構造候補はあるが、Node Aが業務Function Fかどうかは未確認。Function/Businessのcoverageはunsupportedであり、不存在の意味ではない。
>
> **問いへの回答**：「FがDを実際に読む」は未確認。静的Relationのラベルがimportsやrequiresでも、実Readや更新Effectに直訳しない。
>
> **次に確認すること**：FのEntry・入出力とQuery、選択Providerの実Invocation、物理Relation/params、Read Scope・時点・必要なGrant、独立Consumerが受け取ったRowset。

問いごとのRequired（必要なSource/Relation/Rule/実Reader）とConsumed（実際に参照・取得したもの）を区別する。既知の陽性観測はその範囲で提示できるが、「使われていない」「削除できる」「実Readしない」という否定・全称の結論は必要な閉包がなければ保留する。異なるSource/版/Scopeの証拠、同一原本から派生した複数AI要約、別Attemptの結果を無条件に足さない。

### 12.3 実装を一つの小PRに限定する

最初は既存Reader/CLIとReviewPacketの受領をHeadlessの試験で検証し、必要な範囲のみ読み取り専用Projectionを足す。順序は以下の通り。

1. **独立Expectedを先に固定**：公開合成のTask T→Action A→Function F→IOTable I→Data D、手作業M、未取得Source U、同名異IDのn1/n2と並行するe1/e2を用意する。実Captured REA 6.8.0とは別Fixtureにする。
2. **外部Observationの選択**：通常CLIの明示Observation ID／Relation IDで一件ずつ選び、Evidence ID、元Byte digest、関係の方向・ID、Unselected、AuthorityLimitsを記録。ローカルCLIの任意EvidencePath入力をNative/Webへ転用しない。
3. **三要素Reviewの確認**：現行rea-intakeでData/Function/Businessのsupported/unsupportedとwouldNeedを確認する。派生入力なら元Rawからの変換証拠を別票にする。ReviewPacketのAcceptedMeaning=falseをOwner受理に置換しない。
4. **問いに対する回答**：選んだ構造と必要な実Read根拠の不足を表示し、確認済み／推論候補／未確認／矛盾を説明文の分類として扱う。これを新しい共通真理値Enumにはしない。
5. **別レーンへ引継ぎ**：実Wails JSへの受領は#4661、正式Source編集は#267/#2497、実PG Query→Function Contextは#2586、受理済みRule→生成・実CLIは#951/#1251/#4544へ渡す。

### 12.4 既存Scenarioへ渡す独立Expected（新しいテスト正本は作らない）

| 索引 | 入力と反例 | 最小合格・停止の判断 |
| --- | --- | --- |
| RQ-01 | Captured Rawの一Observation＋有向Relation | 元ID/向き/Digest/Unselectedを保ち、Functionの実Readと混同しない |
| RQ-02 | Data候補あり、Function/Businessはunsupported | wouldNeedを提示し、不存在・未使用・CUDなしと断定しない |
| RQ-03 | 合成T/A/F/I/Dはあるが実Provider Receiptなし | 静的構造は示すが、F reads Dの実行事実は未確認 |
| RQ-04 | 同名別ID、e1/e2並行関係の再採番、1対2・2対1 | 対応が曖昧なら個別Relation編集を止め、並び順で確定しない |
| RQ-05 | 同Candidate IDでもObservedKind/State/Coverage変更 | 外部の観測事実差分を保持し、未知Statusをunchangedへ丸めない。#4693のDraftとmain実装を混同しない |
| RQ-06 | UTF-16LE元RawとUTF-8派生 | Byte digestの違いと同値確認の範囲を区別し、二入口成功を単一Read Attemptとしない |
| RQ-07 | 0件、未取得U、未ロードPlugin、権限制限 | 未使用・削除可能の全称結論を拒否し、Requiredの不足を示す |
| RQ-08 | Source A→B→A、Grant変更、後着Response | 昔のApproval/Disclosureを自動復活させず、他ViewのFocus/Dirty Bufferを汚さない |
| RQ-09 | 未受理REA CandidateをGraph→Goへ直接投入 | 製品生成資格を認めず、main #4733の局所Reference生成と区別する |

RQは#1180/#2514へ不足Assertを引き渡す**索引のみ**。#4693（関係再対応・事実差分）と#4716（他媒体）はDraftのため、現行mainで実装済みの成功とはしない。二Producerや二Readerの別CIを足し合わせて一つのProduct E2E合格とも呼ばない。

### 12.5 次の区切り・製品優先順位

**R0**：現行Product二入口のSource/Raw/Tool版と実Receiptを一件ずつ確認する（RQ-01/02/06）。**R1**：合成三要素の一Function質問をHeadlessに投影し、確認・不答・曖昧Relationの停止を検査する（RQ-03/04/07）。R0/R1の局所合格はR3-CのWindows第二Consumer、正式Source Apply、実Function PG Read、SHO Save/Reopenの合格に流用しない。

main #4751/#4756/#4733/#4777とも、**Source選択→実Reader→Meaning candidate／受理済み意味→生成→実Invoke／Consumer→Evidence**を別々に証明する方針で整合する。製品P0はStudio Stage1→Stage2 Windowsセルフビルド#4583/#2831と、通常tei CLIの実Provider→Function Context→独立Consumer#2586のまま維持する。第二Host/Parser/Graph Compiler/Meaning Store/Scenario Runnerを新設しない。

**証拠上限**：現行Go型・既存CLIと他レーンの実装状態を静的照合した設計・試験引継ぎ。今回の新Go/Windows/PG試験、正式Owner意味受理、Source Apply、SHO保存、Target実EffectやTEI-native Plugin生成は未実施。

## 13. 同一Candidate IDで観測事実だけが変わる場合の比較（2026-11-15）

現行`internal/reacandidate/adopt.go`の`CompareCandidateSets`は、同じCandidate IDを持つ二つの構造候補について、主に`SourceRevision.ArtifactPath/Fingerprint`と`Attempt.EvidenceID`で状態を判定していた。ところが`CandidateID`はRelationラベル等の`StructuralFacts`全体を含むわけではない。同じ外部Occurrenceと申告Source Digestのまま`imports`から`requires`へ観測Kindが変わっても、以前は`unchanged`または`same-current-bytes-different-attempt`となり得た。

**限定修正**：同一Candidate IDの`StructuralFacts`全体を比較し、事実差分を検知した場合は理由と再評価要求をNoteへ残す。申告Source Path/Fingerprintが同じときの分類を`observed-facts-changed`とする。Sourceも変わっていれば従来の`revision-changed`を優先し、観測事実も変わったことをNoteに併記する。Attempt IDをProducerが再利用していても事実差分を隠さない。

| 試験 | 独立Expected | この差分では認めない資格 |
| --- | --- | --- |
| RQ-05a：同じ外部Relation ID／申告Source、別AttemptでKind変更 | `observed-facts-changed`＋再評価Note | 旧Sourceの現在性、業務Meaningの変更確定 |
| RQ-05b：Attempt IDまで再利用したKind変更 | `unchanged`へ丸めない | Producer Attemptの真正性・継続性 |
| RQ-05c：申告Source DigestとObservedKindが同時に変化 | `revision-changed`＋観測差分Note | 差分原因の断定、旧Approval/Grantの自動更新 |

これらを`internal/reacandidate/compare_observed_facts_test.go`のGo回帰に固定する。検出は**read-onlyの比較結果**であり、Source-backedな再観測・正式Meaning受理・Source Patch・Windows Consumerを行わない。Source Digest自体はProducer申告の値であり、同じDigestだから実Sourceが同じと保証するものでもない。外部のRelation occurrence IDと正式TEI Relation IDを同一視せず、再採番の多対多対応や各辺の意味をこの修正で確定しない。

既存Draft [#4693](https://github.com/hat47x/tei-platform/pull/4693)にも同種の差分案があるため、**この小PRを先にmainへ統合した場合、#4693では重複する`adopt.go`・テスト・状態語彙を差し引いて現行mainへ再基準化する**。J-01のUTF-16LE/BOM実二入口、J-02の並行Relation・再採番、J-03のCoverageなどの残作業を完了扱いしない。新しいComparison Statusを読むCLI/Studio/SHOの未知状態対応は、#1124/#2514/#4661の独立Consumer検収として残す。

**証拠上限**：実Go回帰のCI結果は対象HEADで別に確認する。今回の修正は外部Producerの実再解析試験、Owner意味受理、実Provider／Target実Effect、Native Wails JS受領、TEI-native Plugin自己構築の完了を意味しない。

## 14. 保存済みREA原本の二入口と並行Relationの回帰検証（2026-11-15）

前節のRQ-01〜09を現行CLIで検証する。旧Draft [#4693](https://github.com/hat47x/tei-platform/pull/4693)のJ-01（保存原本と派生入力）、J-02（並行Relation）、J-03（三要素Coverage）のうち、既存型のまま独立試験できる二つのGoテストファイルを現行mainへ移す。観測事実変更（旧J-02k〜m）はすでに[#4805](https://github.com/hat47x/tei-platform/pull/4805)がmainに入っているため、重複移植しない。

### 現行の二入口で検証すること

- **Captured 6.8.0原本**：UTF-16LE/BOMの保存済み原本をそのまま外部候補選択へ渡す。Evidence ID、実受領Raw digest、指定したsemantic graphのNode・有向Relation IDを確認する。三要素取込みへは明示的にUTF-8へ変換した別byte列を渡す。原本Digestで派生入力をPinした要求はstale拒否。保存原本を読んだことはProducerの再解析や現在Sourceの保証ではない。
- **二つのGraphの区別**：外部選択のjsrg_（semantic graph）と三要素候補のjag_（application graph）は別の観測範囲であり、同一Evidence IDから同じFunctionやRelationの正式な対応を推定しない。Captured ReviewではDataの構造候補をsupported、FunctionとBusinessをunsupportedかつwouldNeedありとし、Owner Decisionはpendingを維持する。
- **並行する外部Relation**：合成Application Graphで同端点・同ラベルのEdgeを二本用意し、別のExternalOccurrenceIDとCandidate ID、read-only Previewの別CrossReferenceを保持する。既存CarrierAuditのexternal-edge-identity Lossも消さない。Producerが同じEdge IDを重複させた入力は拒否する。
- **構造completeの資格上限**：Graphの観測範囲がcompleteであっても、Functionの実Read/Write、Business意味、実DB Effect、全Sourceの網羅性を示すものではない。0件を「未使用・削除可能」とはしない。

| 対象テスト | 期待する観測 | 主張しないもの |
| --- | --- | --- |
| cli_rea_dual_entry_contract_test.go | 保存原本とUTF-8派生を別入力として受領、Raw digest差、Graph種別、旧Pin拒否、三要素CoverageとOwner pending | 二つのCLI呼出しが同一原子的Attemptであること、別Native Product Consumerの成功 |
| cli_rea_parallel_relation_review_test.go | 並行Edgeの区別、Previewの未受理、Carrierの識別子Loss、重複外部ID拒否、completeでもFunction/Business unsupported | 外部IDを正式な業務Relation IDとして受理したこと、実CRUD／Effect |

テストはCLIのGo handlerを直接呼ぶ。通常teiバイナリの独立起動、Native Wails JS/WebView2による第二Consumer、Source Save/ReRead、SHO Save/Reopen、実Provider/DB、Owner意味受理、生成Plugin実使用、REA外部ツールの新しい実行は、今回の証拠に含めない。R3-Cは引き続き[#4661](https://github.com/hat47x/tei-platform/issues/4661)で検収する。

**統合後の引継ぎ**：Draft #4693の同じテストファイルと#4805の実装は二重マージせず、端点再採番と1対多／多対1の曖昧性、独立Sourceに戻れない個別Relation編集など未完了項目だけを現行mainへ再基準化する。他媒体のEvidence候補Draft #4716は別レーン。追加のParser、Host、Meaning Store、Expected Engine、Source書戻し機構は設けない。

## 15. 比較処理の並行統合に伴う重複の解消と再採番の資格（2026-11-15）

前回の独立PR [#4805](https://github.com/hat47x/tei-platform/pull/4805) は、同じCandidate ID／申告Source DigestでもObservedKindなどの構造事実が変わり得る不備を修正した。その後、別レーンから再採番・並行Relationの説明とGo回帰がmainへ統合された結果、`internal/reacandidate/adopt.go`に`ComparisonObservedFactsChanged`の定数が**重複定義**され、`comparePair`でも同じ差分Noteを**二度追加**する状態になった。定数の重複はGoのコンパイルを阻害する。

この修正では、`ComparisonObservedFactsChanged`を既存の比較Status定数群に一つだけ残し、`comparePair`の`StructuralFacts`差分検出を一度だけ実行する。**申告Sourceが同じ場合は`observed-facts-changed`、Sourceも変わった場合は`revision-changed`を維持**し、それぞれのNoteを重複なく出す。回帰`TestCompareCandidateSetsSameSourceChangedObservedRelation`では、別Attempt／Attempt ID再利用／Source変更併発の3ケースで、差分Noteが各1回だけ現れることを確認する。

最新mainの`TestCompareCandidateSetsReversedEdgeNeedsEndpointMapping`、`TestCompareCandidateSetsParallelRenumberingRequiresOccurrenceMapping`、`TestCompareCandidateSetsAsymmetricParallelKeyRemainsUnverified`などは、別レーンにより**すでに追加されている**。この成果を再実装したり、旧Draft #4693の同名処理を重ねたりしない。

一方、現行の`CompareCandidateSets`は再採番候補を構造キーと順序で仮対応する実装であり、結果が`external-identity-renumbered`でも正式な個別Relationの同一性を証明するわけではない。複数Occurrenceや端点変更はNoteでSource-backedな再確認を求め、Editor/Studio側の個別書戻しはOwnerの意味受理・Source Slotの一意性が確立するまで止める。**構造比較の成功と編集資格は別のClaim**とする。なお、現在の並行Relation検査は注意Noteの存在を確認するもので、順序による仮対応自体を廃止したわけではない。これを自動採用と表示しない。

CIについては、前回PR #4816の対象HEADでRepository Healthが失敗したことを確認した。ログで特定できたのは`internal/plugin/management/document_preflight.go`のgofmt不一致と、Studio静的CompositionのNode UI`workbench_revision_mismatch`であり、REAテスト固有の失敗ではない。この2件が現行mainで修正済みかどうかは最新HEADで別に確かめる。旧CI結果を今回の成功へ転用しない。

この修正はGo比較処理と回帰試験の整合を回復するもので、正式`*.tei.md` Reader、Native Wails第二Consumer、旧新Provider Effect、SHO保存、Owner受理、Plugin自己構築の完成を意味しない。#2514/#1180の既存Expectedを利用し、追加のHost・Meaning Store・Runnerを設けない。

## 16. 再採番された関係の曖昧な対応付けを禁止する（2026-11-15）

前回までの実装は、同一の構造キーを持つ並行Relationが複数ある場合、注意Noteを付けてもCandidate ID順に旧候補と新候補を一対一に組み合わせていた。この組合せを後続Consumerが正式な個別Relationの継続性と解釈すると、誤ったSourceへ編集を誘導するおそれがある。これはUI表示で注意書きを添えるだけでは解消しない。

**今回の限定変更**：既存の`CompareCandidateSets`で、同一Candidate IDは従来どおり比較する。Candidate IDが変化した場合、構造キーが旧新それぞれ**1件ずつ**に限って、従来の`external-identity-renumbered`を「Ownerによる確認待ち」の仮対応として返す。一方、旧新いずれかに同じ構造キーの候補が複数ある場合には、位置やIDの順序による組合せを**作らない**。旧候補、新候補をそれぞれ`ambiguous-structural-match`として返し、`StructuralKey`によって同じ候補群に属することだけを残す。各Entryにはその側のSource/Attemptのみを載せ、未証明の相手側Source/Attemptは設定しない。新しいMeaning・Source正本や照合Engineは追加しない。

| 対照 | 比較結果 | 承認しないこと |
| --- | --- | --- |
| 外部IDと構造が一意に対応（1対1） | `external-identity-renumbered`とfresh Owner Review Note | 正式Relation IDや現在の編集権限 |
| 同構造キーの並行二辺→再採番二辺（2対2） | 旧2件・新2件を別の`ambiguous-structural-match`として残す | Candidate ID順の一対一対応、Source書戻し |
| 同構造キーが1対2／2対1 | 旧新の各候補を`ambiguous-structural-match`として残す | 任意の1件だけをrenumbered、残りをadded/removedと断定 |
| 同じCandidate IDだが観測事実に差がある | 既存`observed-facts-changed`／Sourceも変われば`revision-changed` | 旧Owner受理やGrantの継続 |
| 構造キーが一致しない | 従来どおり`added`／`removed` | 取得Scope外の不存在証明 |

既存`internal/reacandidate/adopt_test.go`の並行Relation・非対称一致を、相手側のSource/Attemptを捏造しない独立Expectedへ改訂する。新Statusを受領するStudio/CLI/SHOの表示・編集では、**未知状態や曖昧状態を`unchanged`に丸めずread-onlyで保留**することを、#1124/#2497/#4661/#2514に引き渡す。Graphの描画Edge ID、外部Occurrence、Candidate ID、正式Relation IDを取り違えない。再採番が1対1でもSource-backedなIdentity適合、Owner受理、Current Grantの検証が必要である。

この実装では既存REA CandidateSetとread-only ReviewPacketのみを使う。Native Wails JSの独立Consumer、正式`*.tei.md` Source Apply/Save/ReRead、実PG ProviderとFunction Read、TEI-native Plugin生成、Studio Stage1→Stage2 Windows実受領は別の製品資格。#2586/#4583/#2831のP0を新しいHard Gateで遅らせない。

## 17. 比較結果を要約で見落とさないための受入（2026-11-15）

前節の`ambiguous-structural-match`はJSONの`comparison.entries`に保存されていたが、既存`ReviewPacket.Summary()`には比較欄がなく、要約だけを読む利用者へ曖昧な関係の再確認が伝わらなかった。詳細JSONが正しくても、利用する側の表示が意味の喪失点になる。

**今回の小さな改善**：既存のread-only`Summary()`が、比較を行わなかった場合は`comparison: not-attempted`、比較済みの場合は各既知Statusの**件数**を固定順で表示する。特に`ambiguous-structural-match`を最初に示し、旧新のRelationを一意に結び付けない。未知Statusは勝手に`unchanged`へ読み替えず`unknown-status=N (review required)`、比較有無と実データが矛盾する場合は`inconsistent-section`、Schema/Effectが非対応の場合は`unsupported-contract`として明示する。未知Statusの原文字列、候補ID、Source Path、Source Byteは要約へ出さない。これは既存ReviewPacketの新しいMeaning資格や承認欄ではなく、**現行JSONを要約するGoの読み取り専用Projection**である。

回帰では、旧二辺→再採番新二辺の4件が`ambiguous-structural-match=4`と表示されること、任意の一対一対応`external-identity-renumbered`が出ないこと、未知状態や壊れたComparison契約を成功表示へ丸めないことを検査する。未実行と空の比較結果も分け、空を業務上の不存在証明にしない。

この表示は**Goの`ReviewPacket.Summary()`の出力に限る**。通常`tei`バイナリ別プロセスの応答、Studioの実Window/JS/WebView2、SHOの保存/再読取、他のConsumerの状態解釈までは検証しない。製品側ではこの要約を参考にするだけでなく、未知Status／現行Grant／Source Slotの一意性を独立検査して個別編集を拒否する必要がある。#1124/#2497/#2514/#4661へ引き渡す。

Graphの構造的全件性、Function Read/Write、Owner受理、Source Patchの成功を混同せず、実Provider/CLIとStudio Stage1→Stage2のP0（#2586/#4583/#2831）は独立して継続する。新しいReader、Meaning Store、権限Engine、第二Expectedを設けない。

## 18. 個別Relationの未確認SourceをJSONに出さない（2026-11-15）

§16では、並行Relationの旧新が一意に対応しない場合、Candidate ID順で相手を仮決定せず、旧新それぞれを`ambiguous-structural-match`として残すようにした。しかし`CandidateComparisonEntry`の`BeforeSource`/`AfterSource`はGoの値構造体だったため、`json:"...,omitempty"`だけではゼロ値の構造体が省略されず、JSONに`beforeSource: {"binding":""}`等の**存在しない相手側Source**が表示されていた。Consumerが「プロパティがある」を「Sourceの相手方が確認された」に読み替えると、§16の安全策がシリアライズ時に失われる。

**実装範囲**：Go公開フィールドと比較Status、`tei.rea.candidate-set-comparison/v0`を維持し、`CandidateComparisonEntry.MarshalJSON`だけでゼロ値のSource側を省略する。Source情報のある側は既存のbyte-backed出典をそのまま出し、未確認側のSourceは**キー自体を出さない**。StatusやOccurrence IDから欠けている相手を推測しない。ReviewPacket全体のJSONについても、旧2件・新2件の曖昧な比較で、旧候補は`beforeSource`のみ、新候補は`afterSource`のみを持つことをテストする。1対1の再採番では両側、追加・削除では片側を保持する。

この修正はread-onlyのWire Projectionに限る。JSONにSourceが存在すること自体も、Sourceの**現在性・編集可能なSlot・Owner意味受理・Current Grant**を示すものではない。SourceのRaw digestと派生digest、Application GraphとSemantic Graph、関係の多重度／時間条件を引き続き分離し、下流Consumerは未知Schema/Statusと相手側Source欠落をfail-closedで扱う。正式`*.tei.md` Reader、Windows Wails JS実受領、実Provider Function Read/Write、Source Apply/Save/ReRead、SHO、Plugin自己構築は別の検証であり、P0の優先順位は維持する。

## 19. 旧CLIの無条件「変更なし」判定と入力情報漏出を停止する（2026-11-15）

現行の`tei three-element rea-intake -before`には、以前のReviewPacketをJSONとして読んだだけで、実際の候補対応を検査せず、3つの固定Candidate IDをすべて`unchanged`と出力する経路があった。§16〜18で整備した`reacandidate.CompareCandidateSets`は、この旧CLIの`-before`経路では**呼ばれていない**。候補IDの一致や同じJSON構造は、Provider観測事実の変化・外部Occurrenceの再採番・Source A→B→A・Owner受理の継続を証明しない。旧CLIの出力を受けるConsumerが`unchanged`を自動採用すると危険である。

**今回の実装判断**：旧CLIにもう一つの独立したComparatorを作らず、`-before`の比較要求は実装済み`reacandidate`とのSource-backedな接続ができるまで**`REA_COMPARISON_UNAVAILABLE`（終了コード2）で拒否**する。以前のReviewPacketが同一、変更済み、空のJSON Objectのいずれでも`unchanged`を捏造しない。壊れたJSON・存在しないファイル・`null`は`BEFORE_SET_READ_FAILED`を返す。拒否時に`-output`へ成功ReviewPacketを書かない。通常の`rea-intake -evidence`の既存read-only Previewは維持し、この修正自体を完全な三要素受理とは呼ばない。

同じCLIの`candidateSetFingerprint`は、`fmt.Sprintf("%x", evidence)`で原本Byte全体を16進文字列化していた。これはハッシュではなく、原本そのものの可逆表現である。今回`sha256:<hex>`の**原本入力byte digest**へ変更する。`fingerprintBasis: raw-evidence-bytes-not-candidate-identity`を明示し、既存フィールド名だけから正式CandidateSetの内容識別やProviderの資格が確定したと誤読しない。SHA256一致だけでSourceの現在性、外部観測内容、問別のEvidence充足、Owner Decisionや編集許可を復活させない。

回帰は旧CLIの実Go handler経由で、同じ原本と変更済み/空の過去Review、無効JSON／欠落／`null`、`-output`の未生成、入力byte digestが正しいことを確かめる。これは`tei`バイナリを独立起動した第二Consumerの証拠ではない。以降は既存`reacandidate` Reader・Comparator・ReviewPacketと、`three-element rea-intake`の入口を無理に統合せず、実Evidenceから実候補へ渡る一経路の同一性／Effect／Provenanceを先に確かめる。その検収まで`-before`を有効化しない。

**製品P0の優先は変更しない**。正式`*.tei.md` Reader／Studio Stage1→Stage2 Windows実受領（#4583/#2831）と通常CLI→実Provider→実Function Context（#2586）は継続し、新しいHost／Parser／Comparator／Meaning Store／独立Expectedを追加しない。Source-backedな正式Writer/SHO Save・Owner受理・Native JS第二Consumer（#4661）は別資格のままとする。

## 20. 型付きCandidateSetを既存比較へ接続するread-only CLI（2026-11-15）

§19で`three-element rea-intake -before`の偽`unchanged`を止めた後、単に拒否を続けるだけでなく、すでに存在する`reacandidate.CompareCandidateSets`と`BuildReviewPacket`を**通常`tei` CLIから呼ぶ最小のConsumer入口**を加える。新しい比較アルゴリズムや独立したREA原本Readerは作らない。

**入口**：`tei three-element rea-compare -before previous-candidate-set.json -after current-candidate-set.json`。入力は既存`reacandidate.GenerateCandidates`が返す`tei.rea.candidate-set/v0`相当の読み取り専用CandidateSet二件であり、旧`rea-intake`の簡易ReviewPacketや外部REAの生JSONを無条件に流用しない。Schema/Effect、Attempt Evidence IDとProviderの識別・版、CandidateSetの内容Fingerprint、固定のAuthorityLimits、Candidate IDの一意性とOwner確認待ちの状態を局所検査する。Fingerprintは正しい形式・自己整合を示すのみで、原本取得や外部Runとの一致、Owner意味受理、現在の閲覧権限の保証ではない。

**出力**：既存`reacandidate.CompareCandidateSets(before,after)`の結果を`BuildReviewPacket(after,nil,nil,&comparison)`に組み込み、既存read-only Packetと`Summary()`をCLI JSON Envelopeに格納する。新しいStatus/Meaning/Decision Engineを作らず、`ambiguous-structural-match`の旧新片側Source／AttemptとReview Summaryを保全する。対象が複数の場合にCandidate ID順の偽対応を作らず、`unchanged`でもOwner受理やSource編集権限には昇格させない。未知Schema/不整合Fingerprint/未確認Statusなどは成功Envelopeを返さず、終了コード2とする。

**受入条件**：最小の型付きCandidateSetを既存Go APIからJSONにし、通常CLIの`runThreeElementCommand`経由で二時点を比較する。並行同構造Relation 2対2は旧2件・新2件を保留、要約で件数4、JSONで架空の相手Sourceがないことを検査する。一意の再採番はfresh Owner Review付きの候補、同じCandidate IDの異なる観測Factsは`observed-facts-changed`であり、元の`rea-intake -before`は依然`REA_COMPARISON_UNAVAILABLE`で拒否する。欠落ファイル・破損JSON・旧ReviewPacket・未知Schema・改変Fingerprint・重複Candidate IDを負例で固定する。

**残る資格**：このPRが証明するのはGo CLI Dispatcher→既存Comparator→既存ReviewPacketへのread-only接合まで。外部REAツールをもう一度実行した証拠、Raw/UTF-8派生Byteの受領、実製品バイナリを別プロセス起動した独立Consumer、Wails JS受領、正式`*.tei.md`からFunction Read/Write、Source Patch/Save/ReRead、Owner受理、Pluginの自己構築、Studio Stage1→Stage2実Windows受領は引き続き別Expected。既存#1180/#2514/#4661/#2586/#4583/#2831を用い、新Host・Runner・意味正本・万能Readiness Gateを設けない。

## 21. 既存REA Readerから型付き候補を生成し、二時点CLI比較につなぐ（2026-11-15）

§20の`rea-compare`は型付き`CandidateSet/v0`の二件比較まで到達したが、実`tei` CLIから原本REAの受領Byteを既存`reacandidate.DecodeAttemptEvidence`→`GenerateCandidates`へ通し、そのまま比較対象として保存する**前段の入口**がなかった。これは旧`three-element rea-intake`の固定候補と混同せず、同じReader/Generatorを二度作らないことを優先する。

**今回の製品入口**：`tei three-element rea-candidates -evidence analysis.json [-output candidates.json]`を通常CLIの同じ`three-element`振分けに追加する。raw Byte列を読んでSHA-256ダイジェストを残し、既存REA decoderとGeneratorのみでSource-backed CandidateSetとread-only ReviewPacketを生成する。標準出力にはCandidateSet、ReviewPacket、Summary、受領したraw Byteのdigestを包み、`-output`には後続の`rea-compare`が直接読める**型付きCandidateSet JSONのみ**を新規保存する。Sourceや既存出力を上書きしない。明示的な`-output`がなければファイルを書かない。

受入試験では、同じREA形式の実入力から`rea-candidates`を経て`rea-compare`へ二度の時点を渡す。合成同端点・同labelの並行Relationを旧二辺／再採番新二辺として与え、Source-backedな候補Identityが保持され、比較では旧2・新2を`ambiguous-structural-match`とし、要約は4件、JSONでは根拠のない相手側Sourceが存在しないことを確認する。独立したRaw digestとCandidateSet Fingerprintを区別し、旧`rea-intake -before`のfail-closedを維持する。入力失敗・不正JSON・Source外部表明の欠落、既存出力ファイルへの上書き拒否も確認する。Go構造体テストに残っていた`ReviewStatus`/`Resolution`二重記述を正し、**テスト自身がコンパイルできない**状態を解消する。

これはGo CLI振分けから既存Reader→既存Generator→JSON CandidateSet→既存Comparator→ReviewPacketを通す、製品コードの**読み取り専用・型付き一往復**である。外部Tool再実行や受領Byteの独立Capture、原Sourceの現在性・生成物の表示/編集資格、Owner意味受理、Wails JS第二Consumer、正式`*.tei.md` Source Apply/Save/ReRead、実Provider Function Read/Write、Plugin自己構築、Studio Stage1→Stage2実Windowsは検収していない。新Host／第二Reader／第二Comparator／意味正本／万能Gateは設けず、既存#1180/#2514/#4661/#2586/#4583/#2831へ分けて引き渡す。

## 22. 実保存REA原本のReceiptと候補生成資格を分離する（2026-11-15）

§21で保存済みREA JSON→既存Decoder/Generator→CandidateSet→Comparatorのread-only CLI経路を実装した。一方、既存の`cmd/tei/rea_evidence_test.go`の末尾には、`reaEvidenceOuterArgs.observationID/relationID`および`reaEvidenceResult.Candidates`を要求する古いテストが残っており、現行`rea-evidence`には存在しないAPIを想定してGoビルドを止めていた。これを理由にProduct APIへ根拠のない選択機能を復活させない。

より重要なのは、テストが参照する**実保存済み6.8.0原本**`internal/externalanalysis/testdata/rea-6.8.0/analysis.json`の内容である。このファイルには受領可能な`evidence_id`・Provider・Subject Digestがあるが、`normalized_result`は`{}`であり、`JavaScriptApplicationGraph`のNodes/EdgesやObservation/Relation IDは含まれない。従来テストが特定の観測と関係を選択できたとするExpectedは、保存ファイル自体が支持しない。

**修正**：既存`rea-evidence`による実保存原本のReceipt取得と、Raw Byte数・Evidence ID・Promotionなしを検証する。同じ原本を§21の`rea-candidates`に渡すと、既存`DecodeAttemptEvidence`が`GRAPH_MISSING`（終了コード2）で拒否し、候補JSONを出力せず原本Byteを変えないことを回帰化する。外部ツールの有無がReceiptの欠けたGraphを埋めることはない。

この変更は**本物の保存ファイルが持つ証拠と、合成Graph付きFixtureが示す操作能力を分ける**ためのもの。§21の合成旧2辺→新2辺比較は引き続き合成受入であり、実ツールのGraph取得成功、再実行による現在性、外部Occurrenceの完全性、正式Owner意味受理、Source Apply/Save/ReReadやNative Studio第二Consumerは未検収。次に実保存Graphまたは正しく束縛されたViewを取得できる段階で、Evidence ID・Raw/派生Digest・Provider版・Coverage・Relation Occurrenceを別の受入として追加する。既存#1180/#2514/#4661/#2586/#4583/#2831のP0を維持し、未実装フィールド・架空Source・第二Reader/Hostを増やさない。

## 23. REA専用CIを「Receipt受領」と「Producer ID付き候補選択」に分離する（2026-11-15）

§21～22で既存REA Reader・型付き候補生成・比較CLIを接続した一方、REA専用CIの`verify_real_rea_intake.py`には旧契約が残っていた。スクリプトは`tei rea-evidence -observation-id ... -relation-id ...`を実行していたが、現行`rea-evidence`は**原本Receiptと派生Requestを受領する入口**であり、候補選択引数を持たない。外部ツール6.7.0／6.8.0の実行後にこの架空のCLI呼び出しで終了コード2となり、実REAが失敗したかのように読めるCI結果を作っていた。

**修正方針**：原本ByteからのReceipt受領・Evidence ID・Raw SHA256・Promotion=false・派生Request検証は、候補Graphの有無にかかわらず毎回実施する。Producerが同じ受領原本内の`normalized_result.semantic_graph`にNode ID／Relation IDの双方を表明したときだけ、既存の`tei external-candidates -evidence-path ... -tool-version ... -observation-id ... -relation-id ...`を呼ぶ。選択結果がRead-only Operation Contractであること、同じ受領Raw SHA256、外部IDとAuthorityLimitsを検査する。

`normalized_result={}`などでGraph／選択IDが欠ける場合、Source-backed候補は**NOT RUN**として記録するが、Receiptと派生Requestの検収は省かない。不正なGraph型を無言で空扱いせず拒否する。Fixtureをテストの都合で加筆せず、実保存Byteの変更やGraph取得成功も主張しない。合成の並行Relation試験と実保存原本の資格を区別する点は§22と同じ。

この是正が確認できるのは、外部REA実走→受領Byte→Receipt／派生Request→存在する場合に限ったR2外部候補投影までである。`three-element rea-candidates`が実保存Graphから三要素CandidateSetを生成できたこと、Old/New Sourceが継続していること、Owner意味受理、Native JS第二Consumer、TEI Studio実Stage2、実PG Function Read/Write、Source Apply/Save/ReReadは別の実Expectedとする。Go契約試験の旧保存FixtureにGraphがあると想定した不整合は独立の修正対象として残し、実証条件を勝手に緩めない。 なお、実ToolがAnalysisとViewを捕捉した直後に原Byte Digestを記録し、候補選択・出力検証で失敗しても`--report`へ失敗状態と短い原因を残す。CIの終了コードは成功へ読み替えず、`always()`のartifact取得を診断に使う。

## 24. 実保存REA原本と合成Semantic Graphの契約試験を分離する（2026-11-15）

§22～23で、保存済み6.8.0の`analysis.json`は**Receiptとして受領できるが`normalized_result={}`でGraphを持たない**と確認した。しかし従来の`internal/operations/external_candidates_test.go`と`external_candidates_ambiguity_test.go`、実バイナリを起動する`cmd/tei/cli_external_candidates_test.go`は、この原本に選択可能なNode・Relation・Unknownが存在するとみなしていた。REA専用CIのGo契約試験は実際に`semantic_graph`をGoの`map`へ強制変換してpanicした。失敗を解消するために保存原本を改変したり、実外部ProviderがGraphを返したかのようなFixtureを正本へ上書きすることはしない。

**二つの独立した試験系列を採用する**。実保存原本は`internal/externalanalysis/testdata/rea-6.8.0/analysis.json`の受領Byteをそのまま読み、Receipt State／Evidence ID／SHA256／Promotion不許可と、`ReadCandidates`および`ExternalCandidates`がGraph欠落を`EXTERNAL_CANDIDATE_UNUSABLE`として拒否する事実を固定する。外部Sourceの現在性や業務意味は推論しない。保存原本の内容変更・生成Graph挿入は一切行わない。

一方、選択可能なObservation／有向Relation／並行Relation／同一IDの重複／Derived Handleの衝突／未選択件数／CoverageとUnknownの保持／Source改訂の**機能的性質**は、別の`internal/operations/testdata/rea_external_candidates_synthetic.json`で検証する。このFixtureは独立の`ev-synthetic-operations-graph-v1`、`meta.command=synthetic-test-only-not-real-rea-capture`、合成Node 7件・Relation 2件・Unknown 1件のGraphを持ち、実保存ファイルとは異なるExpectedとRaw Digestを使う。GoのR2 Operation試験と別プロセスの実CLI R3試験は、同じ合成Fixtureを使用し、成功結果に**実REAのProducer資格を付与しない**。固定の旧Receipt Digestと候補GraphのSHA256を混同せず、合成原本のDigestを入力Byteから計算する。合成変異は一時コピーのみに行い、原本Fixtureも変更しない。

**検収の判定を分ける**：実保存原本からの候補投影の拒否は負例の成功、合成Graphからの候補選択と曖昧性拒否は機能の成功にすぎない。実REA再実行からそのGraphを受領できたか、独立Current Grant、Ownerによる意味受理、Source Patch/Save/ReRead、Wails Studio第二Consumer、`*.tei.md`正式Reader／PG Function実Provider／Plugin自身のTEIセルフビルドは依然として未検証。REA 6.8.0のView配送の別CI失敗（`rea-evidence -view`が終了1）は本修正では成功へ読み替えず、その`--report`の受領Digestと失敗理由から独立調査する。既存P0のStage1→Stage2 Windowsと実Function Providerは維持し、第二Graph正本・新Host・新Comparatorを設けない。
## 25. 逆解析REAと業務REAを三要素モデルへ接続する（2026-11-15、横断設計追補）

REAという略称は、ここでは二つの別概念を指す。外部の **Reverse Engineer Anything** は、プログラムや画面などを調べ、根拠付きの観測を提供する。一方、**Resource–Event–Agent** は、業務資源・経済事象・関与主体から取引や業務の意味を理解する分析方法である。**両者を一つのREA Parserや専用Core型へ統合しない。** 外部解析をCandidateの入力経路、Resource/Event/Agentを受理可能な意味Roleとして扱い、既存の三要素・標準データ構造・Rule・Processへ投影する。

| 異なる軸 | 逆解析から得るもの | 業務REAとして検討するもの | TEIに接続する場所 |
| --- | --- | --- | --- |
| Resource | テーブル・項目・APIレスポンス・状態変数とその出典 | 在庫、債権、サービス提供能力など、業務で増減・移転の対象となる資源 | DataのRole/Relation、単位・粒度・時間条件、Persistence/Query |
| Event | 呼出し・更新・ログ・画面遷移などの観測 | 資源の受渡し・消費・変換など、業務上の経済事象 | BusinessのProcess/Event/Ruleと、必要なFunction呼出し |
| Agent | User ID・担当者欄・認証主体・外部Actor候補 | 取引に参加する内部・外部の当事者と役割 | Business/組織モデルのParticipation・Responsibility・期間 |
| Relation | 有向Edge、Source occurrence、呼出し先、FK・SQL参照 | 資源の流入出、Event間の対応、主体の参加 | 標準Graphの**意味Role付きRelation**と元Sourceへの追跡 |

**Functionの呼出しは業務Eventではない。** 一つのEventに複数のFunctionや手作業が関与し、一つのFunctionが複数Eventの処理を受け持つ場合もある。商品出荷と入金のようなEvent同士の対応（duality）や、資源の増減（stock-flow）を検討しても、コード上の順序、DB更新、同名フィールドだけでは成立しない。これらはOwnerが根拠・条件・対応範囲を確認したときに初めて正式な意味Roleとなる。すべての業務に会計語彙や二重性を強制せず、必要な領域に限って使う。

**Economic Agent ≠ Person ≠ Principal ≠ Host Grant**。顧客、代理人、担当部署、認証済み利用者、サービスアカウント、取引に参加する当事者は、名前が同じでも別の概念である。対応関係は根拠付きのQualified Link、役割、時間条件と対象Scopeで表す。現在の操作権限はHostがその都度確認し、古いREA候補や組織履歴から再生しない。Resourceの識別子も、業務キー・物理PK・Document ID・Source上の表示名を区別する。

**表現は増やすより投影を使い分ける。** 同一のAccepted Sourceから、三要素表、REA三角関係、Node Flow、状態遷移、ER/Graph、Process図、SHO文書を生成する。表示のためにEdgeを短縮・集約したら、元のRelation occurrence、方向、多重性、Source、導出Ruleへの戻り道を残す。戻れない図から編集対象を決めない。SUIの空間配置やStudioの線のIDも、意味上のRelation identityではない。

## 26. 「旧FunctionがDataを読む」から「業務上どの資源が動く」までを一問ずつ検証する

最初の製品Questionは従来どおり、**「Function FがData Dを読む／書くといえるか」**とする。次のQuestionを、同じSourceと意味受理を再利用できる場合だけ派生させる。**「その更新は、Resource Rに関するEvent Eのどの役割を担うか」**である。前問の成功は後問の証拠ではない。

答えの資格を一つのReadyフラグへ圧縮せず、既存Receipt／Review／Inspect／Traceに次の四つの問いを並べて表示する。

| 資格 | 確認する事実 | 不足時の答え |
| --- | --- | --- |
| 観測 P | 実Producer、保存された原Byte、Graphの有無、選択したObservation/Relation ID、Source位置 | Receiptは有効でもGraphが無ければ候補選択は未実行 |
| 根拠閉包 Q | このQuestionで必要なSource/Plugin/手作業/動的実行/期間が取得済みか | 0件でも不在とは断定せず、partial/unknown/unsupportedを区別 |
| 対応付け J | 外部Occurrence→Data/Function/Business→Resource/Event/Agent→正式Qualified Refの対応が一意で、Owner受理されたか | 複数候補を残し、意味RoleやSource Patchを確定しない |
| 実現 A | Source/Rule改版、Generator実消費、実Provider Read/Write、Effect/独立Consumerの受領 | 図、生成Go、模擬結果だけでは旧新同値・業務完了としない |

**現状の重要な反例**：保存済みREA 6.8.0の実原本はReceiptとして受領可能だが、normalized_resultが空で、Semantic Graphに関係IDがない。#4962/#4990/#5003で実原本と合成Graphを切り分けた。合成Graphで候補選択や並行Edge比較が動くことは、実Producerから同じGraphを取得した証拠ではない。この欠落は新しいパーサーや架空のRelation IDで補修せず、実Producer成果物または明示的なViewを、**同一Producer Attempt／Source・Raw/派生Digestを照合しながら**受領する作業へ返す。

さらに、旧システムでの静的SQL参照、TEIの宣言上のCRUD、Providerによる実Rowset/Mutation、DB Commit、外部通知、業務Event完了は異なる。同じFunction名であっても旧実測L、Owner受理した期待D、新実測Tを別に採取する。Read成功によりWriteやEvent成立を証明しない。業務REAのEventを履歴に保持するか、本体行の状態として扱うかは個別要件で選ぶ。汎用Event StoreやREA専用Runtimeは前提にしない。

### 26.1 独立Expectedへの反証引渡し（REA-XQ-01〜08）

| ID | 入力差分 | 期待する判断 |
| --- | --- | --- |
| REA-XQ-01 | 実Receiptあり／Graphなし、合成Graphあり | 実Producerに候補があると偽装しない。Receipt合格と候補未実行を両立 |
| REA-XQ-02 | 同名の二Node・同端点の平行Relation二本 | Labelや図の線番号で同一化せず、Source occurrenceが不明なら逆編集拒否 |
| REA-XQ-03 | Function参照あり、実Provider Read未実施 | 宣言上の利用候補を提示し、実Read・未使用判定は保留 |
| REA-XQ-04 | Agent表示名一致、Role/期間/Principal異なる | 当事者・担当者・認証Principalを自動Linkせず、Grantも推測しない |
| REA-XQ-05 | DBの在庫値減少、出荷Event/取消・二重計上が未観測 | Resourceの更新候補だけ示し、経済Eventを受理・確定しない |
| REA-XQ-06 | 一Eventに二Function＋手作業、Function一件だけ解析 | Function数をEvent数に置換せず、必要なBusiness Scopeをunknownにする |
| REA-XQ-07 | 原SourceがA→B→A、Candidate IDとDigestが見かけ上同じ | 旧Owner受理・生成物・現在Grant・Consumer受領を自動復活させない |
| REA-XQ-08 | 低権限Consumerに非公開Source/当事者が混在 | Disclosure対象外のID・存在・件数・理由を露出しない |

これらは新しいテスト正本ではなく、#1180/#2514の独立Expectedと#2516のDisclosure検収へ追加する**判別条件**である。実行ログが揃うまではすべて試験候補として扱う。

## 27. 実装順序を一つの製品縦断へ絞る

| 順 | 次の実作業・Owner | 合格に必要な証拠 | 依存・停止条件 |
| --- | --- | --- | --- |
| **R0** | #3394：実REA 6.8.0の一成果物で、Receipt・View・Graphの配送とProducer発行IDを区別 | 実保存byte、実Tool/版、Subject/Attempt、Rawと明示変換した派生byte、Graph/IDが存在するときだけ候補投影 | Graph欠落なら「Receiptまで合格」で停止。合成Fixtureで実陽性を代替しない |
| **R1** | #1124/#4487：既存CandidateSet／ReviewPacketで一Function・一Data・一手作業をHeadless回答 | Original RelationとQualified Mapping候補、Required/Consumed、Unknown、曖昧時のfail-closed | 外部Node/Edgeを正式Data/Function/Businessへ自動昇格しない |
| **R2** | #1124/#2386：業務REAのResource/Event/Agentを既存Meaning/Graph/Rule Roleとして限定受理 | 元のF/D・Event参加・時点・粒度・Ownerの適用範囲を同じQuestionに照合 | Event≠Function、Agent≠Principal、Read≠業務Effect。汎用新Core型を作らない |
| **R3** | #2621/#988/#1121/#4544：受理済み一件から図・Rule・生成Go・独立テストへ | 実Builderが同じSource/Artifact byteを消費。図のLayout変更と意味変更を別に失効判定 | Goへdelegateするだけの見せかけのSelf-buildを合格にしない |
| **R4** | #2586/#4495：通常tei CLI→実Provider→Function Context→独立Consumer、必要なら旧L/D/T比較 | selected/actual Provider、実Rowset/Effect、必要なReceiptとConsumer ACKを別々に採取 | 実Readを実Write、MockをCommit、同名旧Functionを実旧System実測にしない |
| **R5** | #4661/#4583/#2831：Native Studioの実Read ConsumerとStage1→Stage2を独立検収 | Windows Wails/WebView2の実受領・正規埋込Asset、Stage1によるStage2再Build | R0〜R4をDesktop P0の新しい前提条件にしない |

**次に最も価値が高いREA固有の実装差分**はR0である。実Graphが届けば、既存の型付き候補・比較CLIで同一外部Attemptの一問を検証できる。届かなければ、Producer/Delivery能力とTEI受領契約のFit/Gapを確定できる。どちらも新Reader/Registry/Host/Expected Engineを増やさずに判断できる。並行してR1の合成反証は続けるが、両者の試験資格を合算しない。

## 28. 自己適用で測る効果と文書化の境界

最初の自己適用対象は、TEI自身のREA候補取得・候補比較・三要素Reviewのうち**意味を持つ一操作**とする。解析器が出した説明をそのまま正式Sourceにせず、レビューで受理したData/Function/BusinessとRelation/Ruleが、実生成コードの条件・参照・出力に反映されることを試す。

**評価指標**は、(1)元Source→意味Role→生成Go→実結果まで追えるRelationの割合、(2)元Sourceへ戻れない表現のLoss件数、(3)同一DSLから図・文書・テスト・Target生成に使えた意味の範囲、(4)手書きGoへ残る処理とその理由、(5)Source/Toolchain/Owner Meaning変更時に正しく失効したArtifact/Consumerの範囲、とする。単なる行数削減やAI説明の滑らかさでは判定しない。

この追補は**設計判断と受入計画の更新**であり、業務REA Roleの製品実装、実REA Graph付き受領、Owner受理、Go生成・Provider/Effect、Windows Stage2の新しい実証ではない。製品の独立P0は、通常CLI実ProviderとStudio Stage1→Stage2のまま維持する。

## 29. Producerの関係端点に基づく一組の選択を、実REA入口で保証する（2026-11-15）

§25〜28でいう「Graph付きEvidenceが届けば最初の一問を試せる」には、選択した観測NodeとRelationが**同じGraphに存在するだけでなく、Producerが表明した関係端点を共有すること**が必要である。従来の`verify_real_rea_intake.py`は最初の`node_id`と最初の`relation_id`を独立に拾っており、Graphの先頭NodeがそのRelationに無関係でも、TEIの`external-candidates`へ二つのIDを渡せた。Node/Relationそれぞれの存在を確認しても、組としての参照整合性は確認したことにならない。

既存の`selection_from_evidence`で、Graph内のProducer発行`node_id`集合と`relation_id`集合を読み、**選択Relationの`source_node_id`または`target_node_id`のいずれかと一致する観測Nodeだけ**を一組として返す。Source端点を先に試すが、未観測ならTarget端点を使える。どちらにも観測Nodeが無ければ候補投影はNOT RUNとし、Receiptと派生Requestの検証を続ける。同種Producer IDの重複は曖昧な選択として拒否する。これらの判定は**実Graphの取得成功や業務上の関係受理の証明ではない**。Source/Target双方の完全性やRelationの真偽は、別の必要根拠・Coverage・Owner資格で判断する。

**固定反例（REA-PAIR-01〜04）**：

| 反例 | 期待する選択・拒否 |
| --- | --- |
| PAIR-01：先頭Node n0は無関係、Relation r1はn1→n2 | n0+r1の見かけ上の組を作らず、n1+r1を選ぶ |
| PAIR-02：Relationの両端は未観測のID | Receiptを捨てず、候補投影のみNOT RUN |
| PAIR-03：Source端点は未観測、Target端点n2は観測 | n2+r1を選べるが、Source側の完備を断定しない |
| PAIR-04：Node IDまたはRelation IDが同一Graph内で重複 | 任意のOccurrenceを選ばず、検証を失敗させる |

これらは既存VerifierのPython transport mockに限定した試験である。外部REA 6.7/6.8の実再実行、Graph付き正例、TEI CLIの実バイナリ第二Consumer、Native JS、正式三要素Function/Business、Source Apply、実Provider/Effectを本変更の合格へ合算しない。保存済み6.8.0原本の`normalized_result={}`は変更せず、引き続き「ReceiptはあるがGraph選択はNOT RUN」の対照例とする。

**他領域へ再利用する判断**：この「関係の双方の端点が同じ文脈に載るか」という技術的保証と、Relation occurrenceの識別、Ownerが受理した意味Link、Processの実Call/Effectは別の資格である。Node Flowの配線、Plugin依存、三要素CRUD、業務REAのEvent/Participation、組織Person/Agent、Studio Graph/Source逆編集にも、最初の**端点参照整合**の後で意味・時点・権限を順に検証する。表示上のEdge IDや同名LabelをProducer Relation IDに見立てず、失われたIdentityを新たな万能Graph Coreで補わない。

**次の製品R0**は実Toolの`analyze-javascript-application`→保存Evidence→`inspect-analysis-view`という一Attemptを観測し、Graphの具体的な配置・端点ID・Scopeを現物で確かめること。保存元とViewの同一Evidence/Raw/派生Digestを別々に確認し、配送GraphがなければProducer・Selectorのどちらが不足したかを報告する。REA-XQ-01〜08の独立Expected（#1180/#2514）と、Native実受領（#4661）を飛ばさない。製品P0の通常CLI実Provider（#2586/#4495）およびStudio Stage1→Stage2 Windows（#2831/#4583）は並行して継続する。

## 30. 実REA 6.7/6.8の同一CI実行から、次の断点を確定する（2026-11-15）

[PR #5038](https://github.com/hat47x/tei-platform/pull/5038) の`REA external evidence intake`、[run 43721894737](https://github.com/hat47x/tei-platform/actions/runs/43721894737)（PR HEAD `35896dec6333e6f8af21744a083476180ca05586`）を、既存の**保存済み6.8.0原本**と明確に区別して読む。今回は外部REAを実際に起動した`real-tool`二系統で、各々別の新規Evidence/Raw DigestとProducer発行Node/Relation IDを観測できた。Graph付きの新規実解析と`normalized_result={}`の過去保存原本は、矛盾する結果ではなく**異なる取得Attempt**である。

| 実行 | 実行記録で確認した事実 | 引き続き未確認の部分 |
| --- | --- | --- |
| REA 6.7.0 real-tool | `completed`、checks=4、notRun=[]。Evidence受領・派生Request、Producer発行端点に接続するObservation/Relationの`external-candidates` read-only投影、工具自身の二種類の不正リクエスト拒否を確認 | Business/Function意味受理、正式TEI Source/実DB Read/Effect、Native独立Consumer |
| REA 6.8.0 real-tool | 失敗Receiptでも`checks`に**原本受領と実Graph関係選択の成功2件**が記録され、`notRun=[]`、解析/Viewの捕捉Digestは別々に残った | その後の`tei rea-evidence -analysis-output … -view … -provider-version 6.8.0`がexit 1。配送全体の合格、失敗原因の分類、後段の不正リクエスト二件は未確認 |
| Python Verifierの局所Contract | 新規4反例を含む10件PASS。無関係Nodeの誤選択、両端未観測、片側のみ、重複Producer IDを判別 | 実ProducerのGraph全件性・Source現在性・独立Windows Consumer |
| Go contract job | Pythonの10件PASSと別に、既存`TestExternalCandidatesRefusesWhatItCannotProject`が`error-source-kind.jsonl`を読めず失敗 | Go contract job全体の成功。Fixtureの出典・正しい負例実体を別Ownerで確認する |

6.7と6.8で表示されたNode IDとRelation IDは同じ文字列でも、Evidence IDとRaw Digestは異なる。**ID文字列一致を同一Attemptや同一Sourceの版同一性として使わない**。6.8は関係選択の成功を証明できたが、後段のView受領不合格を消してはならない。6.7の完全成功も6.8の不足を代替しない。

**次の最小R0-B実装**は6.8の`-view`受領である。検証器が保存したanalysisとviewのByte、Requestの`source.kind`、REA側JSONL framing／terminal metadata、TEI側View Readerの版条件を同一Attemptで照合し、**どの判定でexit 1になったかを型付きの失敗記録として残す**。その際、旧保存FixtureをGraph付きに書き換える、6.8のProvider版を6.7として受理する、不完全なViewをcompleteに変換する、外部Node/Relationを意味Subjectへ昇格させる方法は取らない。必要な変更を既存Reader/Verifierに限定し、実Tool→受領→View→候補の再試験を行う。

並行するR1は、今回成立した構造Candidateを正式三要素・業務REAへ即採用するのでなく、既存`reacandidate`の型付き候補とReview/Source/Questionへどう渡せるかFit/Gapを明確にする。Node/Relation同定の成功、Candidateの三要素分類、OwnerのMeaning受理、正式Source保存、生成Go/Builder実消費、実Provider/Effect、Studio Native独立Consumerを段階別の受入に保つ。製品のP0（通常CLIの実ProviderとDesktop Stage1→Stage2）には新しいHard Gateを加えない。

## 31. REA 6.8のJSONL終端形式を、実際の配送内容に合わせて受領する（2026-11-16）

§30の実REA検証では、6.8.0の新規Analysis/実Graph候補選択まで成功した後、`tei rea-evidence -view`だけが終了1となった。既存`internal/externalanalysis/view.go`を調べると、保存済み6.8.0の`view.jsonl`に含まれる旧式`$rea=jsonl/v1,type=metadata,emitted=7,source_count=1`という終端行と、データ7件を固定的に想定していた。一方、現行REAのJSONL出力では、`{"$rea":"jsonl/v1","type":"metadata","records":{...},"context":{...}}`という**構造化された終端レコード**を出力する。旧Fixtureを更新せず、二つの配送形式を別に検証する必要がある。

**今回の実装境界**：既存`Receipt.WithView`に現行の終端JSONオブジェクトを読む分岐を追加し、`records.path`、`records.emitted`、任意の`records.source`、`context.evidence_id`をViewに保持する。実データ行数と`emitted`が一致して初めてwholeと判定する。終端欠落・不正JSON・未知control/type・負数・`source < emitted`・申告件数不一致は既存StateのPartial/Mismatchへ落とす。現行出力に対して保存済みFixtureの固定7行を一律に要求しない。6.7.0の単一JSONL行と、6.8.0の旧式`$rea=`終端は従来どおり検査する。

**同一性を増やさない**：`context.evidence_id`は受領View自身が申告するIDであり、元AnalysisのEvidence IDと一致すると仮定しない。`ParentLinked`は独立した連結根拠がない限りfalseのままとする。`source`は配送側の対象件数であり、全Graph/Function/Businessに対する観測の完全性ではない。Viewをwholeと判定できても、Raw AnalysisのSource現在性や独立Owner Meaning、編集権限を証明しない。

**最小受入**：既存`TestInspectRealSavedViewIsComplete`による旧Fixture正例に加え、現行JSONメタデータのdata件数2/申告2/総数10、data0件の明示完了、終端なし、件数矛盾、未知type、未知protocol、emitted欠落、source件数矛盾、壊れたdata行を局所Go回帰にする。実際の6.8.0を呼ぶ既存REA専用CIで`tei rea-evidence -view`と後続の工具負例2件まで到達できた場合だけ、R0-Bを実Toolの完了として記録する。過去のrun 43721894737の失敗を新実装の成功へ読み替えない。

**横断適用**：標準データ構造・SQL集計・Plugin Flow・Node Viewでも、配送のレコード数（`emitted`）と元集合の粒度・列挙範囲（`source`/Coverage）は別の情報である。部分集合2件の受領が正常でも「業務FunctionのReadが2件だけ」「他のResource/Eventが存在しない」とは答えない。Source/Meaning/Builder/Provider/Effectの次の資格は既存#1180/#2514、#3394、#2586、#4661へ引き渡す。実行系と業務REAの新Core・意味正本・Hostは増やさず、Stage1→Stage2 Windowsと通常CLI実Providerという製品P0を維持する。

## 32. 派生Requestの原文保持とCLI Receiptの資格を揃える（2026-11-16）

外部REAの6.7/6.8実ツール検証は[PR #5106](https://github.com/hat47x/tei-platform/pull/5106)の[REA専用CI](https://github.com/hat47x/tei-platform/actions/runs/43731902211)で両版とも4 checks / notRun=[]になり、実Graphの端点付き候補投影とViewの受領まで一Attemptで到達した。一方、[PR #5134](https://github.com/hat47x/tei-platform/pull/5134)は未マージのまま終了した。以後の修正は現在mainの実コードを起点に進め、終了したPRを完了実績として扱わない。

現在の`cmd/tei/rea_evidence_test.go`の旧Expectedには、次の四つのずれがあった。

| CLIで問う事実 | 旧Expected／実装のずれ | 今回の修正と合格条件 |
| --- | --- | --- |
| 受領Evidence→派生Request | `json.Marshal(json.RawMessage)`が本文の改行・空白を圧縮し、原文と一致しない | 受領済みEvidence JSONを検証したうえで、派生Requestの**埋込部分に同じ本文Byte**を保持する。Request全体のDigestは別に採取する |
| View framing | 旧テストの`jsonl/v1-rows-and-terminal`と現行`FramingFor`の`jsonl`が一致しない | バージョン別のframing受入と、終端`Control=metadata`を別々に照合する |
| Parent linkage | View側の`parent_evidence_id`が現れるだけで`ParentLinked=true`と判定していた | 未証明のParent Linkはfalseのまま。宣言された参照と独立に検証された関係を同一視しない |
| Tool不在 | 保存済みEvidenceの完全性まで`tool-absent`へ置換する旧Expected | 実際に受領したByteが完全ならReceiptはcomplete、現在Toolが利用できない事実は`Tool.Present=false/AbsentReason`として併記する |

Reader側の既存`Receipt`/`View`を使い、不要な新Stateや専用Hostを導入しない。派生RequestをJSONへ再整形しないことは、Editorの改行・字下げとSource occurrenceの識別を保ち、`Raw Fingerprint`と`Derived Fingerprint`の違いを監査できるための最小要件である。Requestの検証は、見かけのJSON同値だけでなくEvidence ID/Reader/Sourceとの対応も既存`WithRequest`で行う。なお原文を保持しても、取得時点のSourceの現在性や外部Producerが宣言した業務意味を認めたことにはならない。

**次の独立Expected（REA-CLI-01〜04）**：①字下げ・改行を含むEvidence本文が派生JSON内でByte一致し、Request全体のDigestは原本Digestと異なる、②Viewの終端metadata有無がComplete/Partialを分ける、③Parent未検証を`ParentLinked`へ昇格させない、④Tool不在でも保存済みRaw/Source/Receiptを破棄せず、現在の利用不能を別欄へ残す。既存`rea_evidence_test.go`と実CLIの契約回帰に限定する。

**R1への引渡し**：正式なData/Function/Business対応付けには、観測Graphから選択したRelation occurrence、元Sourceの位置・Revision、型付きCandidateSet／ReviewPacket、Ownerの判断とScopeを一問ずつ結合する。組織C1の「生成メソッド名に依存しない抽象経路」の検証案やNode Flow/Pluginの型付きportは利用できるが、関係の技術的整合と業務REAのResource/Event/Agent意味は別。生成GoやDB Providerの実Call、Effect、Studio Native第二Consumerを、このCLI修正の合格に加算しない。製品P0の通常CLI実ProviderとStudio Stage1→Stage2 Windowsは従来どおり優先する。

## 33. 一つのRelationへの質問を、外部Occurrenceの読取専用Reviewとして実装する（2026-11-16）

前回mainへ統合した[PR #5158](https://github.com/hat47x/tei-platform/pull/5158)は、[REA受領CI run 43859175613](https://github.com/hat47x/tei-platform/actions/runs/43859175613)でPython／Goのcontract、REA 6.7.0、REA 6.8.0の**三ジョブがいずれもsuccess**となった。本文Byteを保持した派生Request、型付きReceipt／Viewの受領、実Producerの端点付き候補投影はR0の根拠として扱える。これらは別のSource版や業務Functionの実Readを証明しない。

R1の最初の問いを「旧Function FはData Dを読むか」と置くと、静的な`imports`や`loads`の有向辺だけを根拠に`Read=true`と誤答する危険がある。現行`reacandidate.EvaluateThreeElementCoverage`は、外部JavaScript GraphのData系候補を支持する一方、**FunctionとBusinessはunsupported**と返す。これは機能不足の隠蔽ではなく、今ある証拠から言えることの限界である。質問の実装もその判断を維持する。

**R1-Q（今回の最小製品差分）**：既存`tei three-element rea-candidates -evidence ...`に、任意の`-relation-id`を追加する。これは**Producer発行Edge occurrence ID**であり、表示ラベル／生成Goメソッド名／Canonical Function IDではない。既存の`CandidateSet`と`ReviewPacket`を一度だけ作り、選んだRelation候補・向き・SourceRevision・観測State/Authority・同じGraphで観測されたFrom/ToのData候補Occurrenceを、`relationQuestion`として追加投影する。元のCandidateSetとRaw Evidence Fingerprint、未解決・Unknown・全体Coverageは残す。出力は既存CLI envelopeの任意欄であり、新Graph／Meaning正本・Runner・Readerを作らない。

| 受入ケース | 必須の結果 |
| --- | --- |
| RQ-01：端点付きRelation r1を明示選択 | r1のProducer Edge ID、From/To Node ID、関係ラベル、Source位置・Revisionを同じAttemptのReviewから表示 |
| RQ-02：同じNodeに複数のObservation | 同名・同端点でも個別の外部Occurrence IDを保ち、観測を一つに縮約しない |
| RQ-03：r1と同端点・同ラベルの並行Relation r2 | `-relation-id r2`だけを選び、元CandidateSet内のr1/r2は両方残す。名称による暗黙の選択は不可 |
| RQ-04：両端の片側が未観測／Provider Unknownあり | 存在しない端点を補完せず、`MissingEndpoints`、Partial/Unknown、未解決理由を保つ |
| RQ-05：選択IDなし・外部Edge IDでない・重複 | 読取不能を明示的に拒否。選択不可能を「関係が業務上存在しない」と言い換えない |
| RQ-06：`imports`・`loads`等の静的Relation | `functionDataClaim=not-established-by-external-structure`、`acceptedMeaning=false`、`ownerDecision=pending`。Data/Functionの正式CRUDやRead／Write Effectを捏造しない |
| RQ-07：出力ファイルの指定 | 保存するのは全CandidateSet（no-clobber）だけ。質問応答は読取専用で、正式TEI Sourceを変更しない |
| RQ-08：複数Attempt・Source A→B→A・Producer ID再利用 | 当該AttemptのEvidence ID／CandidateSet Fingerprint／Source Binding資格を保つ。ID文字列だけで別Attemptや現行Sourceへ同一視しない |

**他領域との接続**：組織C1で進んだ「生成メソッド名ではなく意味ID・Relation Key・向き・多重度・Query IDで抽象経路を検査する」方針（#5115）、Node Flow／Plugin間の型付きPort・明示Adapter、プロセス実Call／Effectの証拠照合を、一つの質問の**次段の資格**に使う。技術的なRelationの受領・片端点への一致をもって業務上のResource/Event/Agent ParticipationやAuthZ Grantとみなさない。業務REAは専門Roleの投影であり、全Functionへ会計Eventを強制しない。

**次のR1-A／R2の最小入口**：選択した構造Candidateから、既存Source Indexで実Artifact Byte/Revision/位置を再確認し、Ownerが指定したQualified Data/Function IDとSourceを照合できたときだけ「対応付け候補」を前へ進める。それでも関数の入力・出力・副作用の実観測がなければ`Function → Data Read/Write`は不明のままとする。確定したモデルを使う生成Go、正式Builder、通常CLIの実PG Provider Read、Studio Native独立Consumer、実Effect/Grantは別Attempt・独立Expected（#1180/#2514）で検査する。主P0の通常CLI実Provider（#2586）およびWindows Studio Stage1→Stage2（#2831/#4583）は変えない。
