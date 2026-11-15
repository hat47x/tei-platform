# Rule Authoringから実Plugin・製品Consumerまでの第一縦断�E�実裁E��検収の接続計画

- **状慁E*�E�E026年11朁E5日の設計�E実裁E��渡し。R1�E�正値Rule→実判断Go生�E�E��E[PR #4911](https://github.com/hat47x/tei-platform/pull/4911)でmain統合済み。R2の既存参照Builder回帰は製品�E正式Builder選択とは別賁E��。実Plugin呼出し、E��常CLIの業務Entry、Windows Stage2は未受理、E- **優先対象**�E��E蔵Rule Authoring Markdown Readerで読み取った一つのpureなValidationを、生成Goの判断処琁E��実Builder、E��択済みProvider、E��常製品CLIまでつなぐ、E- **前提となる実裁E��宁E*�E�mainの[7921b005](https://github.com/hat47x/tei-platform/commit/7921b005a44d325983d24ca2b8b58b0737dd778d)で、�E示選択されたtei.rule-authoring/v0を�E蔵ReaderがWorkspace Loadの原文から読み、semantic.ValidationとSource anchorへ投影する経路が追加された。これ�E外部Language PluginめE��式Graph Readerの製品受入ではなぁE��E- **既存�E正本**�E�[Pluginの役割・合�E・Stage2再�E](PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md)、[Plugin-firstの実運用検証](PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)、[自己記述](ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)、[製品完�E計画](../workstreams/PRODUCT_COMPLETION_PLAN.md)を維持する。本書は一機�Eの実裁E��刁E��決める補助計画であり、第二�EContract、Resolver、Meaning、Builder、Runner、Evidence Storeを作らなぁE��E
## 1. 最初�E成果物は「PluginらしぁE��義」ではなく、生成した判断を実際に呼べること

最初�E機�Eは、�E力�E型�ENULL�E�欠損�Eゼロ・墁E��値を持つ**副作用のないValidation一件**とする、Eraphの同名Nodeと多重有向Edgeは、第二例として既存�E固定�E入力をそ�Eまま利用する、Eraphから始めると、現在の冁E��Rule Readerと未接続�EGraph Readerを同時に完�Eさせる忁E��があるため、第一実裁E�ERuleに絞る、E
専門家が読むSourceは条件・入力�E結果・説明を自然な頁E��で表現し、Goのstruct、func、型引数めE��成器固有�E都合をDSLへ持ち込まなぁE���E力�E欠損、NULL、型不適合、ゼロ、正値につぁE��、結果だけでなく「なぜ判定したか」を参�Eできるようにする。ただし判定不�Eを便宜上falseめE��適合に落とさなぁE��E
成功は、生成Goに実際の比輁E�E刁E��があり、Sourceの条件一箁E��を変更すると**生�Eコードと実行結果がともに変わめE*ことを、独立した期征E��で示した場合に限る。呼出し用Wrapper、固定した期征ESON、手書きGoの規則への委譲は、生成した業務判断として数えなぁE��E
## 2. Pluginは配币E��ではなく、どの契紁E��何を提供したかで扱ぁE
| 観点 | 第一縦断で固定する�E容 | 同一視しなぁE��と |
| --- | --- | --- |
| Authoring | 選択済みLanguage Implementation、管琁Eourceの原byte、定義位置 | File名から推測したParser、未管琁Eraft、表示用Markdown |
| Meaning | 既孁Eemantic.Validationの型付き候補、受琁E��たRule�E�Operatorの牁E| Readerの読み込み成功、Source anchor、Ownerの意味受理 |
| Generation | 対象契紁E�ETarget・選択Generator・実生成した比輁E���E岁E| Goファイルが存在したこと、生成器の自己申呁E|
| Build | Build Intent、�E力閉匁E��Builderの実�E力、ArtifactSetとDigest | 別作業でビルドした同値Binary、別HEADの成功 |
| Runtime | 選択されたProvider、実Entry、現在のScope・Grant、Call結果 | Packageが�EってぁE��こと、exact Lockだけ�E存在 |
| Surface | 通常tei CLIのTyped Operationと独立Expected、後続Studio投影 | チE��ト専用CLI、Reference Go実行、画面上�E表示だぁE|
| Knowledge | 公開可能と判定した契紁E�ESource-backed Manualの実Read | Inventoryの発見�E功、�Eニュアル候補、�E開推測 |

Authoring�E�Generator�E�Runtime Provider�E�Presentationは**利用場面での役割**であり、Coreの固定Plugin種別ではなぁE��同一Packageが褁E��役割を持ってよい。一方、Build時だけ用ぁE��GeneratorをRuntimeへ無条件に含めず、契約ID、実裁ED、Plugin ID、業務上�ECanonical IDを混ぜなぁE��E
## 3. 三つの実現方式�E同じ意味契紁E��対する代替とする

| 方弁E| 適用と検査 | 初回の扱ぁE|
| --- | --- | --- |
| **静的合�E・コード生戁E* | 配币E��るGo単一Binaryへ忁E��な生�E実裁E��Runtime Providerを収録。選択された機�EだけをBuild閉包に含め、更新時�EArtifactと実Consumerを�E賁E��匁E| **製品�E第一候裁E*。TEI Studioの通常Desktop配币E�E生�Eアプリと整吁E|
| **独立Process Provider** | 既存�EProcess Runtime墁E��で契約Version、呼出し権限、観測、失敗�E不確定作用を扱ぁE���E離実行が忁E��な非信頼拡張めE��立更新に限る | pureな第一例�E忁E��条件にはしなぁE��第二方式�E比輁E��証に使ぁE|
| **外部HostへのTarget-native生�E** | .NET�E�Java等�E既存HostへAdapterと実裁E��生�Eし、Hostの認証・認可・Lifecycleに接続。TEI専用常駐Hostを要求しなぁE| Go製品�E受�E後に一Contractを異Targetへ再利用できるかを確誁E|

Goの動的Pluginロードを標準�E更新・配币E��構として前提にしなぁE��静皁E��成が満たせなぁE��実行中の独立更新」や「隔離」�E要件が�E体化した時点で、Process方式と比輁E��る。方式�E選択�ETargetとBuild�E�Runtime閉包の一部であり、同じPlugin IDだから全Targetで実現できるとはしなぁE��E
## 4. 実裁E��ケチE��と一Attemptの証拠

**R0�E�Readerの製品回帰�E�E2621�E�E1018�E�、E* 既存�E蔵Readerで、E��択済みSourceの同一Load byte、Source anchor、旧tei定義の互換性を検査する、E026年11朁E5日にmain統合済みの[PR #4890](https://github.com/hat47x/tei-platform/pull/4890)は実Load、二Rule、Source Indexの元byte位置、説明文変更による旧Index失効、未選択�E契紁E��違いをGo試験へ追加し、[PR #4891](https://github.com/hat47x/tei-platform/pull/4891)は同じLoadからInspector�E�TraceへRuleの定義位置を�E合する試験を追加した、E*両PRの統合と試験コード�E存在は確認済みだが、当該新HEADのCI成功は別途確認する、E* 実裁E��落、褁E��候補、管琁E��Draft、Source A→B→A、既孁Eei定義の互換性など、未確認�E条件は個別に残す。�E蔵Readerの成功を外部Language Pluginのexact Resolver�E�Lockの受�Eへ昁E��させなぁE��E
**R1�E�型付きMeaningから判断Goを生成！E1121�E�E4544�E�、E* mainの[PR #4911](https://github.com/hat47x/tei-platform/pull/4911)で、実選択したMarkdownの正値Rule�E�Estd.decimal`�E�から`value.Sign() > 0`を含むGoを生成し、別Go Processで数値�E�NULL�E�欠損を検査する局所実裁E��入った。ただし正式Builderによる生�Ebyteの採用、Generator版�E配币E��E��、Ownerの意味受理、実Plugin�E�Providerは**未検収**のままとする、E
<!-- rule-read-target-mutation-vs-generation-qualification-20261115 -->
**R0→R1を結�E実Read変更の対照**�E�mainの[#4890](https://github.com/hat47x/tei-platform/pull/4890)�E�[#4891](https://github.com/hat47x/tei-platform/pull/4891)�E�[#4901](https://github.com/hat47x/tei-platform/pull/4901)で、E��択済みRuleの実Loader・Source Index・既存Evaluatorの局所Readまで到達した。次の[Go回帰](../../internal/operations/selected_rule_markdown_mutation_test.go)は、E*同じRule IDと`positive`を保ったまま、参照Attributeを`quantity`から`line_number`へ一箁E��だけ変更**し、�ELoadされた`semantic.Validation`、Evaluatorが実際に要求する`Number`の引数、判断結果、旧Source Indexの失効と新Indexの成立を一絁E��比輁E��る。判断結果の差にはチE��ト専用の合�E数値�E�数釁E、�E細番号0�E�を用ぁE��ため、実DBのProvider ReadやCoverageの成立ではなぁE��Rule ID一致・旧Source持E���E旧評価結果を理由に生�EGoの再利用を許可しなぁE��R1では、このSource変更に対して**生�E器が実際に作ったGo判断箁E��の変更**を追加で検証し、R2では実Builderの消費byte、R3では通常CLIの実Provider�E�別Consumerまで同じ因果関係を接続する。意味受理は引き続きOwner判断である、E
<!-- rule-read-target-mutation-vs-generation-qualification-20261115 -->
**R0→R1を結�E実Read変更の対照**�E�mainの[#4890](https://github.com/hat47x/tei-platform/pull/4890)�E�[#4891](https://github.com/hat47x/tei-platform/pull/4891)�E�[#4901](https://github.com/hat47x/tei-platform/pull/4901)で、E��択済みRuleの実Loader・Source Index・既存Evaluatorの局所Readまで到達した。次の[Go回帰](../../internal/operations/selected_rule_markdown_mutation_test.go)は、E*同じRule IDと`positive`を保ったまま、参照Attributeを`quantity`から`line_number`へ一箁E��だけ変更**し、�ELoadされた`semantic.Validation`、Evaluatorが実際に要求する`Number`の引数、判断結果、旧Source Indexの失効と新Indexの成立を一絁E��比輁E��る。判断結果の差にはチE��ト専用の合�E数値�E�数釁E、�E細番号0�E�を用ぁE��ため、実DBのProvider ReadやCoverageの成立ではなぁE��Rule ID一致・旧Source持E���E旧評価結果を理由に生�EGoの再利用を許可しなぁE��R1では、このSource変更に対して**生�E器が実際に作ったGo判断箁E��の変更**を追加で検証し、R2では実Builderの消費byte、R3では通常CLIの実Provider�E�別Consumerまで同じ因果関係を接続する。意味受理は引き続きOwner判断である、E
**R2�E�実BuilderとPlugin Contributionを接続！E943�E�E951�E�E1016�E�、E* Build-only GeneratorとRuntime Providerの閉包を�Eける。現在のBuild Intentに対象を表せなぁE��合�E、既存Build Execution�E�ArtifactSetへ最小�E適吁Edapterを検討し、架空のPlugin SubjectめE��二Builderで穴を埋めなぁE��実Builderがどのbyteを読んだかをArtifact Digestと区別して記録する、E
**R3�E�実Providerを通常製品CLIから一度呼ぶ�E�E1251�E�E4495�E�E1158�E�、E* CLIの公開操作�E既存�ETyped Application Operationの投影とし、別の意味Dispatcherを作らなぁE��選択Provider Bに対してAが動ぁE��場合�E、結果が同じでも失格。�E回�Epure�E�read-onlyとし、実作用のあるProviderの許可には進まなぁE��E
**R4�E�第二Consumerとセルフビルドへ進める�E�E1824�E�E2831�E�E4583�E�、E* R3を通した同じ意味契紁E��Studioのread-only Inspectorなどから確認する。別途、正式な非Plugin Desktop Build Subject、Stage1 TEI実Processが生成したStage2、実Windows Wails/WebView2を一つの受�E経路で検査する、ELIの二回起動や局所部品�E再生成をDesktop Stage2成功と呼ばなぁE��E
吁E��階で既存Source Index、World�E�Model�E�Artifact Revision、Plugin Lock、Evidence Packet、Execution Attempt、Actual ConsumerめE*参�Eで結合する**。新しい一括State、永続証拠表、丁E�EPASS判定�E導�EしなぁE��E
## 5. 効果を伴ぁEluginへ拡張する際�E墁E��

pure Ruleの実裁E��通った後に、DB更新�E�通知など外部作用を伴ぁEroviderへ拡大する。その際も既存Command�E�Effect�E�Failure�E�Receipt契紁E��使用し、Lifecycle GraphはGuard・頁E���E実行主体�E表現に限る、E
- Previewで候補と忁E��情報を確認し、Resolveで実Providerを固定し、Admitで現在AuthorityとGuardを検査してからCommitへ進む。Observeで実際の結果を�E合する。この並びは検討中の意味上�E整琁E��あり、新Runnerの五Phaseを導�Eするも�EではなぁE��E- 既存Hookには、宣言したusesの先行呼出しやGuard経路との二重到達�E可能性がある、Euard拒否前に外部作用が実行されなぁE��と、許可したCall回数が上限を越えなぁE��とを実Providerで検査する。単純に既存HookをGraphへ写しただけでは安�Eな実裁E��認めなぁE��E- 実Providerへ到達後、ACK喪失・結果不�E・部刁E��果が生じたらeffectUnknownを維持し、既存Receiptのlookup-firstに従う。未観測を「作用なし」「安�Eな再試行」と扱わなぁE��E- Runtimeの現在Grant、業務意味のOwner受理、Source�E�Artifactの一致、外部公開DocumentのDisclosureは別賁E��である、Eocal Studioのno-auth既定を共有Web�E�外部Hostへ引き継がなぁE��E
## 6. 追加する独立Expectedの差刁E��新しい試験正本は作らなぁE��E
以下�E既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)�E�[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ引き渡す候補であり、未実行である、E
| ID | 一因子�E変更 | 実際に確かめること |
| --- | --- | --- |
| PLV-01 | Ruleの比輁E��件のみ変更 | 生�EGoの判断箁E��と実Consumer結果の双方が変わめE|
| PLV-02 | 入力を欠損／NULL�E�E�E�型不適合へ変更 | 欠損と不正値とfalseを混同せず、宣言外�E補完をしなぁE|
| PLV-03 | Markdown裁E��・表示位置のみ変更 | Source再確認を残し、意味やArtifactの不要な失効を�Eける |
| PLV-04 | 同一IDのSourceをA→B→Aで変更 | 旧意味受理・生�E賁E��・現在Grantを�E動復活させなぁE|
| PLV-05 | Reader未選択／曖昧�E�実裁E��ぁE| 不正な部刁EorldめE��黙�EGo標準Reader fallbackがなぁE|
| PLV-06 | 選択ProviderをA→B、実CallをAに固宁E| 結果同値でも未適格として拒否する |
| PLV-07 | Builderが別版�EGenerated Goを消費 | Source�E�Artifact�E�Builderの不一致を検�Eする |
| PLV-08 | Build-only GeneratorをRuntime閉包に混入 | 不要な依存混入を検�Eし、Runtime向け最小構�Eと検証済み構�Eを区別 |
| PLV-09 | Guard拒否前にeffectful Hookのusesが呼ばれる | 外部作用0回を要求し、満たせなければ適用を停止する |
| PLV-10 | 実Call後�EACK喪失 | Effect unknownを保持し、無条件に再実行しなぁE|
| PLV-11 | Plugin DocのLockだけ一致、SourceRef�E�Audienceが欠落 | 斁E��の閲覧・再�E開�ESkill手頁E��現在適格と表示しなぁE|
| PLV-12 | Stage0がStage1�E�Stage2相当�EBinaryを別、E��Build | 実Stage1→Stage2再�Eの成功へ昁E��しなぁE|

## 7. 実裁E��E��、Owner、停止条件

**最短の実裁E��E�E R0のReader負侁EↁER1の生�E判断 ↁER2の実Builder実消費 ↁER3の通常CLI実Call** とする。第一PRはR0のみ、第二PRはR1〜R3のぁE��独立Expectedを通せる最小篁E��に刁E��てよい。未対応部刁E��残る場合�E、それを明示して次の差刁E��送る。R4のWindows Desktop Stage2は並行する製品P0であり、R0〜R3を理由に優先度を下げなぁE��E
RuleのMeaningの正本は#2620�E�E988、Language選択�E#1018、Target Loweringと実生成�E#1121�E�E4544、Plugin選択�E#951�E�E1016、E��常CLI実Callは#1251�E�E4495、Desktop Self-buildは#2831�E�E4583が担当する。受入用の判断基準�E#1180�E�E2514へ返す。Plugin斁E��は#4679�E�E4714、Lifecycle Effectは#4198�E�E1174に残す、E
方式を縮小する条件は、E��E�E�一Ruleのために第二�EResolver�E�Runnerが忁E��になる、E��E�E�生成コードが手書き判断に依存してSource改訂に反応しなぁE��E��E�E�実Builderの入力や選択Providerを追跡できなぁE��E��E�E�意味の欠損�E現在Grant・Effect unknownが�E功に丸められる、�EぁE��れかである。その場合�E抽象化や対象Domainを増やさず、既存�E実行墁E��へ戻して一機�Eだけを完亁E��せる、E
**計測**�E�固定した機�E義務を刁E��に、TEIから生�Eした判断、委譲、手書き、未実裁E��別、E��記録する。同じ義務�E改訂前後で、保守するRule数、手書き変更箁E��、E��褁E��る定義、独立検収までの作業とConsumer数を比輁E��る。Plugin数めE��成Go行数の増加だけをセルフビルド�E進展とはしなぁE��E
こ�E斁E��は実裁E��画である。mainのReader実裁E��趁E��るGo�E�CLI�E�Plugin�E�Windowsの新実行�E行っておらず、Ownerによる意味受理も未確認である、E
## 8. 他領域と共通化するのは「対象と賁E��の受渡し」である

Plugin、Studio、SHO、SUI、YIC Harnessの議論を通じて、同じ情報を受け渡すために巨大なContextめE��自Agent Runtimeを作る忁E���EなぁE��整琁E��れた。まず一つのSubjectにつぁE��、E*何を選び、どの版�Eどの惁E��を読み、どの操作を依頼し、どこまで観測できたぁE*を�E通�E型付きOperationと既存Evidenceで表す、EIへの説明やSkillは、この確実に確認した部刁E��ら忁E��に応じて構�Eする、E
| 受渡しに忁E��な閉包 | 確認する�E容 | 不適格な状慁E|
| --- | --- | --- |
| Subject�E�Information | Workspace・Canonical Subject・Role・Source occurrence・Required�E�Loaded�E�Boundの対応と取得篁E�� | 別Workspaceの同名Subject、Source A→B→A、partialを完�E読取と誤誁E|
| Instruction�E�Documentation | 実CLIの発見結果、E��択ContractとPlugin版、Source-backedの公開Manual、忁E��時取得した短いSkill | 旧コマンド、合成Fixtureの仮想操作、未確認文書を実行手頁E��して提示 |
| Capability�E�Authority | 選択実裁E��実Artifact、実Provider Entry、Hostの現在Grant、Effectの実観測、Consumerの受頁E| exact Lockだけで実行許可�E�開示賁E��があると誤誁E|

既存�E[共通Application API](https://github.com/hat47x/tei-platform/blob/main/plan/adr/0048-shared-application-api-and-transport-adapters.md)を�E口に、CLI、MCP、Studio、Webは同じ型付き意味操作�E別投影とする、ECPが忁E��なら同じOperationをTool�E�Resourceへ公開するが、独自Agent LoopめE��駐TEI Hostを追加する琁E��にはしなぁE���E開文書につぁE��も、Plugin InventoryのID・Lock一致と、文書SourceRefを実読取りし現在Audienceに開示してよいことは別判定である、E4679�E�E4714へ帰属させ、R0〜R3の新しい先行条件にはしなぁE��E
## 9. 操作対象・表示・Source編雁E�E同じ識別子ではなぁE
StudioのWails�E�WebComponents、SHOの図表、SUIのカードやKJ空間、AI Harnessからの対象引渡しを老E��ると、E*意味上�E対象IDと描画ID、Source位置、現在の編雁E��象**を�Eける忁E��がある、E
Graphの同じ表示名�ENode二件、同一端点の多重有向Edge、業務上�ERelation IDは、それぞれ独立した同一性を保持する。Sourceを表示できても、編雁E��べき一つのRelationが確定したことにはならなぁE���E示Pinは対象選択�E意図であり、現在GrantめE��部送信許可ではなぁE��褁E��候補が残った場合�E候補を示し、AIにも一意�E編雁E��象として渡さなぁE��E
Frontendは業務データの受理済み意味と表示の糖衣を�E離する、Earkdown表、Graphviz、Mermaid、Web Componentsの構造やCSSレイアウト�E、原斁E��保存しながら褁E��の表示を可能にする。ただし、図の配置やCSS変更を業務Ruleの変更として扱わなぁE��送E��Edgeの向き、Ruleの条件、FieldのNULL扱ぁE��変わる場合�E、表示上�E差だけに押し込めず、生成コードと独立Expectedまで追跡する、E
UI実裁E��おけるTypeScript・Web標準技術�EWailsの選択�E#1824の拁E��とし、このPlugin設計�E**Target-neutralな表示・操作契紁E��、忁E��な実行賁E��を持つConsumer**までを定義する。生成したStudio Componentが実WebViewへ収録されたことは、Source生�EやNode側のチE��トとは別に、正規Bundle・Go embed・Windows実Consumerで確かめる、E
## 10. 次に着手する差刁E��、現在の製品到達点に合わせて絞る

冁E��Rule Readerとそ�E実Load回帰がmainへ入ったことで、製品�E第一作業は「汎用Markdown Readerを新設する」ことではなくなった、E*第一実裁ERは既存Readerで受理候補となる一Ruleを固定し、その条件が生成したGoの判断処琁E��決めること**に絞る。生成器の自己記述・外部Language Resolver・全Graph�E�Window・公開Manual・Wails Updaterは、�E回�E成功条件へ載せなぁE��E
検収では、同じRuleの入力だけを変えたケースと、Rule定義自体を変えたケースを区別する。前老E�E型�ENULL�E�欠損�E0・墁E��値で結果がどぁE��わるか、後老E�E**生�Eコード�E刁E��そのも�Eが変わるか**を調べる。どちらも、生成器が返す値をExpectedの根拠にせず、既孁E1180�E�E2514の独立Expectedと照合する、E
次のPRではBuilderがこのGoを実消費し、E��択されたProviderの実Entryを通常CLIの共通Application APIから呼ぶ。実Callが�E立した段階で、別Targetに同じContractを実現させる圧力試験を検討する、E*Studio Desktopの正式なStage1実TEI→Stage2実Windows Wails/WebView2は、引き続き独立かつ最上位�E製品P0**である。開発用CLIが二度起動した事実、ReferenceのGo成功、旧PRのGreen CIを寁E��雁E��てStage2成功にしなぁE��E
### 追加の反証�E�既存Independent Expectedに引き渡す！E
- **PLV-13**�E�PinされたSubjectと実CLIの操作対象が別Workspaceの同名IDになる。結果一致でも受領不適格、E- **PLV-14**�E�製品InventoryのLockは一致するが、実ManualのSourceRef�E�Audienceが未確認。Skillの使用手頁E�E現行実裁E�E保証にならなぁE��E- **PLV-15**�E�同じ意味Graphに対して表示用Edge IDと正式Relation IDを混同する。編雁E��象未確定�EままPatchを行わなぁE��E- **PLV-16**�E�生成TSとNode側試験�E新しくてもWailsに埋め込まれたAssetが古ぁE��実Consumer賁E��は成立しなぁE��E
**記録上�E注愁E*�E�PR #4890�E�E4891のmain統合�E製品Reader回帰の進展であって、外部Language Plugin、生成Plugin�E�実Provider、Studio Stage2、Owner意味受理の成功ではなぁE��本追補�E斁E��のみであり、追加したPLV-13、E6の試験実行や新しい製品実裁E��含まなぁE��E
## 11. 最初�E生�E対象を実Fixtureで固定すめE
設計上�E抽象例ではなく、mainへ統合された実Fixture `internal/workspace/testdata/tei-md-rule/` を最初�E入力にする、E
- **Source**�E�`semantic/sales/order-line-rules.tei.md` の `rule sales.order_line.quantity_positive`。定義は `sales.order_line.quantity is positive`、�E位置は既存Source Indexに帰属する。同斁E��の `line_number_unique` と説明用 `text` ブロチE��は最初�EGo生�E対象に含めなぁE��E- **型�E根拠**�E�隣接する `semantic/sales/order.tei` の `OrderLine.quantity type std.decimal` と対応付ける、Earkdownに書かれてぁE��いGo型や丸め方を生成器が独自に推定しなぁE��型解決できなぁE��合�E、仮の数値型で成功させず診断する、E- **期征E��る対照**�E�数釁E、E、E1につぁE��正の数かどぁE��を判定する、EULL、E��E��欠落、型不適合�E、既存�E入力�EValidation契紁E��基づく拒否�E�未評価を�E示し、勝手にfalseめEへ正規化しなぁE��特にNULLの期征EutcomeはOwnerの独立Expectedが確定するまで未受理とする、E- **変更追隁E*�E�正値条件の一箁E��を変更した時に、生成Goの判断箁E��、実Builderが消費したbyte、実Provider結果を比輁E��る。変更したSourceを読み直さず生�E物だけを更新したケース、生成Goが同じまま手書き実裁E��修正したケースは失格とする、E- **実行�E**�E��E回�E観測は実製品CLIの型付きOperationから選択されたGo Providerへ到達すること。専用のReference runnerを正式な実行�E口と呼ばなぁE���E功後にStudioのInspect�E�Graph表示めE��Targetへ展開する、E
こ�EFixtureはReader・Inspectorを通しぁE*意味候補�E出発点**であり、生成用の正式なRule Operator�E�Provider Contract�E��E力�ENULL意味がOwner受理済みであるとは限らなぁE��足りなぁE��紁E�E#1121�E�E4544�E�E4495の既存担当で確定させ、対象を増やす前に一RuleのSourceから実Consumerまでを閉じる、E
## 12. R1の最初�E実裁E��刁E��生成Goに判断させめE
[PR #4911](https://github.com/hat47x/tei-platform/pull/4911)は、既存Readerが正規化した一Ruleから**条件式を含むGo Sourceを生成すめE*限定実裁E��ある。読取り済みの `semantic.Validation` と解決済みのAttribute垁E`std.decimal` を受け、`value.Sign() > 0` を含む関数を生成する。生成�Eの入力�E結果は既孁E`validationeval.Source`�E�`validationeval.Result` に従う。`validationeval.Evaluate` への単なる委譲でも、固定JSONの出力でもなぁE��E
実GoチE��トでは、製品�E `Workspace.Load` が選択したRule IDとAttribute型を再�E合し、同一LoadのSource Indexが指す原文IDを確認してからLoweringする。生成されたGoを別Processで `go run .` し、`2.5`、`0`、`-1`、NULL、未取得につぁE��独立Expectedで判断結果と読取回数を検証する。未対応Rule kind、未解決型、`std.money`、余�EなRelation、無効なGo package名�E拒否し、Ruleの参�EAttributeを変えた時は生�Eコード�E読取�Eが変化することを確かめる、E
ただぁE*生�E関数に受理済みSourceやOwner権限を自動的に付与するものではなぁE*。現段階ではGenerator選択版のLock、生成�Eから実Builder入力までのDigest、正式Plugin Contribution、E��常CLIが呼ぶ実Provider、Studioの実Windows Consumerを繋いでぁE��ぁE��PR #4911の試験コード�E存在とそ�Eexact HEADでのCI成功も�Eけて記録する。R1局所試験が通ったら、別のチE��ト専用Generatorへ拡張するのでなく、E4544の既存Build入口へこ�E生�EByteを渡すR2を優先する、E
<!-- tei-rule-generated-binary-mutation-witness-20261115 -->
### R1.5�E�同一Rule IDの変更を、生成Goの実行バイナリまで追跡する

[製品Go回帰](../../internal/semantic/validationgen/positive_mutation_binary_test.go)では、`tei.rule-authoring/v0`で選んだ`quantity_positive`のIDと`positive`を固定し、参照先だけ`quantity`から同じ`std.decimal`型�E`secondary_quantity`へ変更する。型のために追加した`secondaryQuantity`は両比輁E��**共通�E固定Data宣言**とし、比輁E��にDataの型定義は変更しなぁE��変更前後�E`Workspace.Load`→`semantic.Validation`→`Source Index`→`GeneratePositive`からそれぞれGo Sourceを作り、同じ手書きConsumerで**別、E�E実バイナリを`go build`・実衁E*する、Expectedは`quantity:satisfied:1`と`secondary_quantity:not-satisfied:1`で固定し、旧Indexの持E��失効、新Indexの成立、生成Source�E�Binary byteの相違を検査する。片方のGo Sourceと固定したJSONだけで二�E力を演�EするチE��トではなぁE��E
ただしこれ�E**直接GoチE�EルチェーンによるR1.5の局所実消費**であって、製品�E選択済みBuilder�E�Build Intent�E�ArtifactSet、生成Plugin登録、E��常`tei` CLI・実Provider�E�別ConsumerによるR2・R3ではなぁE��R2の次PRでは、この**同じ生�Ebyte**を既存�EBuilder選択とSource/Build入力閉匁E��結�E、`Selected B/Actual A`・Build後�ESource差替え�E旧Artifact再利用を拒否する。R3はそ�E正式Artifactを通常CLIで実際に呼び出す。Source IDやBinary SHAの一致だけでOwner意味受理・現在Grant・実Effectを認定しなぁE��テスト追加とHEADのCI合格も別に記録する、E
## 13. ノ�Eド型FlowとPluginを、同じ標準モチE��から展開する

Node-RED�E�Enebular型�Eプログラミング、ESPStudioのようなNode UI、データ連携・マッシュアチE�E・簡易APIに向けても、E*Flow専用の別Meaning�E�Runnerを作らなぁE*。既存�E標準データ構造、Operator Contract、Function�E�Process Flow、Capability Invocationを絁E��合わせ、GUIのNodeはそ�E投影として扱ぁE��E研究統合§16.18、E6.21](../research/TECH_RESEARCH_INTEGRATION_2026-10.md)の議論と方向を合わせるが、現行Plugin第一縦断R1〜R3より先に新しい実行基盤を導�EしなぁE��E
| 一つのFlowを構�Eする要素 | 標準モチE��で保持するも�E | Node画面だけに持たせるも�E |
| --- | --- | --- |
| Operator�E�Service�E�Function定義 | Contract、�E力／�E力�E型と意味、Provider候補、作用・読取篁E��・認可要汁E| 表示名、ノード�E外観、アイコン |
| Node occurrence | Flow冁E�E固有ID、E��択Operatorと版、Port Binding、パラメータ、�E典・変更允E| 画面座標、E��閉、ズーム |
| Edge occurrence | 独立Edge ID、端点Port、Data�E�Control�E�Event�E�Effect等�E関係Role、向き�E多重性 | 曲線�E配置・色・線幁E|
| Process�E�Flow | 入出力、�E岐�E結合、E��E��、Trigger、失敗�E再試行�E補償、State・Receipt契紁E| キャンバスのグループ、折畳み・表示寁E�� |
| 実行と公閁E| 静的生�E先�EGo�E�外部Host、同一Application API、実Provider・AuthZ・入出力Scope | Studio、Web、CLI、MCP等�E操作投影 |

ノ�Eドを「データ取得」「変換」「判断」「外部API」「�E力」「通知」などで刁E��できるが、これ�E**Catalogの検索・表示用Facet**であって、Coreへ新しい排他的なPlugin Role enumを増やすものではなぁE��一つの部品�ECLI�E�DSL�E�Function Chain�E�GraphのぁE��れからも同じContractで利用できる、Elowが一時的な状態を持つ場合も、現在のState�E�Lifecycle�E�Effect墁E��に委�E、GUI実行エンジンに秘寁E�E状態表を持たせなぁE��E
### 最小�E検証例：数量判定�E簡易API

1. 入力データ `OrderLine.quantity`�E�Eecimal�E�を受ける、E2. 型付き入力�E欠損�ENULLを区別し、既存�E `quantity_positive` Ruleを判断Nodeとして使用する、E3. 結果が�E立なら正常応答、不�E立なら業務上�E不�E立を返し、�E力未取得�E型不適合�E評価不�Eとして刁E��する、E4. Flowの表示を変更してもRuleの意味と生�EGoは変化せず、Ruleの参�EAttribute変更では生�E物と実Callが変わる、E5. 静的Go生�Eの第一経路で実Consumerへ到達後、HTTP Trigger�E�API公開／外部Service Adapterを追加する、EPI公開時にはHost認証・現在Grant・外部送信先�ERate Limitを別途�E示皁E��受理する、E
**UI実裁E��の接点**�E�Web標準UI設訁E§12](WEB_STANDARD_COMPONENT_AND_AGENT_HARNESS_2026-11.md)では、こちら�EN0〜N4とStudioのW0〜W4を同じMeaning/Operation/Sourceからの二つのConsumerとして整琁E��る。UI-P1はread-only Inspectorの公開EventとAdapterの墁E��、FLOW-P1は既存Ruleの型付きPort/Graph→生成Goであり、片方の製品検収をもぁE��方のPASSに数えなぁE��新Graph Engine�E�Renderer�E�Registryは導�EしなぁE��E
実裁E��E�E**N0�E�既存Rule�E�OperatorをNodeとしてread-only投影�E��EN1�E�型付きPort�E�Flow検証と一つの刁E���EGo生�E�E��EN2�E�実Builder�E�実Provider�E�CLIを通した一Request�E��EN3�E�Etudio Node GUIで同じFlowを開き、変更と再生成を検収�E�E*とする。外部Trigger�E�Cloud APIはそ�E後�EN4とし、N1のために独立Flow DSL・汎用Stream Runner・SaaS前提のTEI常駐Hostを導�EしなぁE��E
**独立反証候補！E1180�E�E2514�E�E*�E�NG-01 同じNode名に別Occurrence ID、NG-02 同じ端点に多重Edge、NG-03 Portの型不一致、NG-04 部刁E��得を正常な空雁E��と誤認、NG-05 Graph layoutだけ変更して業務Goが変化、NG-06 Rule変更でも古ぁE��成Goを実行、NG-07 API Publish権限と実外部送信許可を混同、NG-08 Effect不�Eなのに再試行して二重送信、をそれぞれ独立に拒否・失効・隔離する、E
**賁E��の区別**�E�現時点でR1の実裁ERは存在するが、N0〜N4、HTTP API実�E開、クラウドConnectorの製品受入を実裁E��みとは扱わなぁE��R2�E�R3が正式な実消費になってぁE��ぁE��階でNodeの可視化だけを「高度な実行機構」として宣言しなぁE��E
## 14. R2で実Builderの生�EGo消費を閉じる�E�E026年11朁E5日�E�E
### 14.1 R1の実裁E��、R2の完亁E��誤認しなぁE
現行mainの `internal/semantic/validationgen/positive.go` は、E��択済みの `semantic.Validation` と解決済み `std.decimal` を受け取り、`validationeval.Source.Number` の呼出しと `value.Sign() > 0` の判断を含むGo Sourceを生成する。`positive_product_test.go` は、そのGoをリポジトリ配下�E一時的なGo packageとして `go run .` でコンパイル・実行し、正値・ゼロ・負値・NULL・欠損と未対応Rule�E�型の拒否を検査する。これ�E**実Go判断生�Eの証拠**であり、既存Builderが同じbyteを�E力として受領し、E�E币E��るArtifactに含めた証拠ではなぁE��E
生�EGoは現状、`tei/internal/semantic/validationeval` に依存する、Eoの `internal` import墁E��を越える別モジュール�E�外部Hostへ、生成ファイルをそのまま移しても一般には利用できなぁE��E*第一経路はTEI Goモジュール冁E��の静的合�E**に固定し、外部Host向けは意味契紁E��適合する�E開API�E�Target-native Adapterを別に生�E・検証する、Eo `internal` の公開化めE��二RuntimeをR2の前提として進めなぁE��E
### 14.2 Builderへ引き渡す�Eはファイル名ではなく、同一Attemptの実�E力である

| 確認箁E�� | 初回一件で要求する証拠 | 停止・失格の条件 |
| --- | --- | --- |
| Ruleの選抁E| `internal/workspace/testdata/tei-md-rule/semantic/sales/order-line-rules.tei.md` の `sales.order_line.quantity_positive` と `order.tei` の型を同一 `Workspace.Load` から取得。Source Indexのbyte位置も�E吁E| Parserを直接呼んだ候補、別版�ERule、同名�E未管琁Eraftを正式�E力にしなぁE|
| 意味とGenerator | `semantic.Validation`、解決済み `std.decimal`、実際に選択しぁE`GeneratePositive` 相当�E実裁E��版、Target用package名�EToolchainを区別 | Owner意味受理の不在をBuilderが補完しなぁE��未対応kind�E�型�E�packageは生�E前拒否 |
| 生�EbyteとBuilder入劁E| R1が生成したGo byteのDigestと、既存Build Intent�E�BuilderぁE*実際に読み込んだbyte**のDigestを比輁E��生成SourceはBuilder所有�E選択済み入力集合へ接綁E| 同じpathを申告してぁE��もbyte相違、別Attempt、古いGenerator、別Toolchainなら失格、Euilderの自己申告だけで一致としなぁE|
| Artifactと収録 | Builderが生成した実Go Binary�E�ArtifactSet Memberを取得し、実コンパイル対象となったSourceの閉包とResultを検査 | 生�E元Goがディスクに存在するだけ、参照GoチE��ト�EPASS、正しい名前の空Artifactを収録成功としなぁE|
| Providerへの引渡ぁE| 選択Contract�E�Lockと実Artifactに含まれるProvider実裁E��確認し、�E通Application APIのtyped Entryへ接続すめE| Build-only Generatorを無条件にRuntimeへ混入しなぁE��選択B・収録A、同値結果での不一致は失格 |

こ�E表の吁E��E��は既孁E943�E�E2595のBuild入力閉匁E��E951�E�E1016のPlugin Selection�E�Lock、E1121�E�E4544のTarget Lowering、E1163のEvidenceと#1180�E�E2514の独立ExpectedめE*参�Eするだけ�E引渡し票**である。新しいBuilder API、Generator Registry、Artifact正本、丁E�EPASS状態�E提案しなぁE��実Builderが生成byteを受け取れなぁE��判明した場合�E、その既存�E口の最小AdapterをOwnerが定め、暫定�EReference Runnerで合格にしなぁE��E
### 14.3 R2→R3の一件の反証と実裁E��佁E
最初�ER2差刁E��は `quantity_positive` だけを、TEI冁E��Reader→解決済み型�E実Go生�E→現行Builderの入力�EGoコンパイルまで接続する。次のR3差刁E��通常 `tei` CLIの型付きOperationから**現在選択されたProvider**を実呼出しする、ELIの専用チE��ト�E口めE��同じ結果を返す手書きGoに委譲して成功とはしなぁE��E
独立Expectedに渡す負例�E、E*R2-01** ReaderのRule�E�型改版後も古いGoをBuild、E*R2-02** Go byteは更新したがBuilderが旧Sourceを実読取、E*R2-03** 選択Generatorまた�EToolchainだけ変更し以前�EArtifactを無検証再利用、E*R2-04** `internal` importを利用できないTargetへ同じGoを�E币E��E*R3-01** 選択Provider Bと実Call Aの食い違い、E*R3-02** Providerのactual call無しでCLIの固定Expectedだけ一致、�E六つとする。既存シナリオに重褁E��るものは新IDの別試験を作らず、差刁E�E観測頁E��だけ足す、E
Source変更A→B→A、生成器版変更、Build時だけ�E依存、�E力取得�E不完�E性、実行時の現在Grant・Effect�E�Receipt、Studio Stage1→Stage2 Windowsは別の賁E��を維持する、E*R2のDoneは実Builderによるbyte受領と実Artifactの成立まで、R3のDoneは通常製品�E口での選択Provider実Call・独立Consumerの受領まで**と定める。R2の完亁E��Studioセルフビルド�E代わりにしなぁE��E
本節は現在mainで確認できたR1の実裁E��Goの墁E��をもとにした計画であり、R2�E�R3を新たに実裁E�E検証したとぁE��報告ではなぁE��E
### 14.4 既存参照Builderに生�EGoを渡す一試行！ER #4934�E�E
実Builder入力�E照合�E、BuildPlanの型を説明するだけでは検証できなぁE��PR #4934はclosed�E�unmergedだが、現在のmainでは統合コミッチE[`c2cf3fe8`](https://github.com/hat47x/tei-platform/commit/c2cf3fe883de7653ef1e0fdb3cf43bd42dcecbcc)を経て `tools/reference-go-build-from-plan/validationgen_build_binding_test.go` が収録されてぁE��。この試験には、宁E`Workspace.Load` ↁE選択Rule・Source Index ↁE`validationgen.GeneratePositive` ↁE一晁Eo package ↁE既孁E`DirSourceResolver`�E�`plugin.BuildPlan` ↁE既孁E`reference-go-build-from-plan` の `-isolate` 実衁EↁEArtifact Digest・BuildArtifactLock照吁EↁE**Builderが生成した実Binaryの独立Expected**とぁE��手頁E��実裁E��れてぁE��、E*mainのファイル収録と、特定HEADのCI成功・Owner受理は別の事実である、E*

こ�E試験では生�EGoを計画確定後に書き換え、古ぁE��画がコンパイル前に拒否されることも確かめる、Eoモジュール斁E��と、生成Goが参照する既孁E`core/identity`、`internal/core`、`internal/semantic`、`internal/semantic/validationeval` を計画Sourceとして明示する。ReaderのSource ID�E�Attribute型、BuildPlanの実�E力Digest、Binaryの実挙動�E別、E��照合し、一致結果だけを根拠にしなぁE��E
**受�E篁E��は既存Reference Builderでの一回�E実Go消費に限る、E* こ�E試験で作ったBuildPlanはチE��ト用であり、正式製品E`tei` が選択したBuilder�E�Plugin Runtimeを証明しなぁE��製品Build Intentから実ProviderのEntryまでの接続と、Stage1→Stage2 Windows実�E入は、それぞれ既孁E4544�E�E4495�E�E2831�E�E4583に残す。PRのマ�Eジとexact HEADのGo�E�CI成功も区別する、E
### 14.5 参�EBuilderの実績と製品R2の残差

現行mainには`tools/reference-go-build-from-plan/validationgen_build_binding_test.go`が実在する�E�旧[PR #4934](https://github.com/hat47x/tei-platform/pull/4934)自体�E未マ�Eジでクローズ�E�。製品Loaderで選択したRuleから生�EGoを作り、既存BuildPlanと`DirSourceResolver`がGo Source�E�依存パチE��ージ�E�Module Contextを実Digestで照合し、`-isolate`ビルド�EArtifact Digest�E�BuildArtifactLock�E�五つの実バイナリ結果を検査する、E4934の実CIでは`tei/tools/reference-go-build-from-plan`パッケージの試験が成功したが、�E体Healthは他パチE��ージの問題により失敗してぁE��、E*こ�E結果を参照Builderでの局所R2先行証拠として記録し、正式製品R2合格には昁E��しなぁE��E*

既存テストが作るBuildPlanには、�EMarkdownのRule Source�E�選択Language・Generator版／Owner意味受理までを製品Builderの入力として束縛する一貫した契紁E�Eまだ示されてぁE��ぁE��実裁E�E次の最小単位�E、E4544�E�E2595の**既存製品Build入口**で次の連鎖を再現することに絞る。`選択済みSource�E�現在版）�E解決済み型付きMeaning→選択Generator実裁E��版�E生�EGo byte→実Builder入力byte→Build Artifact`。各要素のFingerprintめE��を一つの「�E部OK」に潰さず、変更・旧版�E利用・Selected B�E�Actual Aを原因ごとに拒否する。構文上�E同値とOwner受理、同一生�EGoとSourceの現行性、署名�E供給賁E��とGoコンパイル成功を区別する。製品CLIに新しい参老E��マンドを足さず、Builderの重褁E��裁E��行わなぁE��E
Data Rule�E�Table Script�E�Field拡張はチE�Eタ意味の側へ、独立したFunctionとData Contextは公開Entryの側へ収める！E4937�E�、Eode Flowは同じRule�E�Operator�E�Sourceからread-onlyに投影し、�E開�E実行�EHostの実権限とProviderを要する別ゲートとする�E�E4924�E�。最初�ER3対象はこ�ER2を�E行条件にしなぁE��琁E��みpure Function一件とし、E4495の実Providerと通常製品CLIのConsumerを並行して完�Eさせる、E
### 14.6 製品R2の最小一件と、意味変更・説明文変更の扱ぁE
現行`reference-go-build-from-plan`の`go-command`でBuildPlanに載せられる実消費Sourceは、`go-package`と`go-module`である。`checkSourcesAreRead`は実際に読まなぁEindを拒否し、`checkDependencyCoverage`はGoの依存パチE��ージを検査する。参照BuilderにRule Markdownを読ませるため、`*.tei.md`を`go-package`と偽裁E��たり、ビルドが読まないSourceを計画へ足したりする方法�E採らなぁE��E
次の製品R2は、現行�Epureな`GeneratePositive`やGo Builderへ権限�ESource解釈を重褁E��裁E��る�Eではなく、E*既存�E選択�E生�E・Build墁E��を呼ぶ製品�Eの一連の処琁E*として実裁E��る。まず`Workspace.Load`で確定した管琁Eource、Language版、Rule�E�型、Source IndexとWorld�E�Model Revisionを取得する。次に同じ選択版からGoを生成し、E*生�EAttemptの入力指紋�EGenerator実裁E��版�E出力Go digest**を実Builderへ引き渡す、Euilderは従来どおり実Go入力とToolchain�E�依存を独立に検査し、結果のArtifact digest・Build lock・Consumerを生成Attemptへ結�E付ける。忁E��な受渡し形式�E既孁E943�E�E2595�E�E4544の契紁E��で決め、新しい汎用Source kindめE��二BuildPlan、Evidence正本を新設しなぁE��E
| 反証となる変更 | Generator側で確かめること | Build�E��E利用側で確かめること |
| --- | --- | --- |
| **Rule条件・参�E先を変更** | 新しい`Workspace.Load`と型解決、別の意味・Go digest、旧Source Indexの失効 | 古ぁE��成Attempt�E�Build賁E��を拒否し、新Go byteを実Builderが読む。別の実Consumer結果を独立Expectedと比輁E|
| **Markdown説明文・レイアウトだけ変更** | 管琁Eourceの原byte持E���E変わるが、E��択Ruleの意味・生�EGoは同じかを再評価 | **生�EGo�E�Artifactの再利用可否**と、E*旧Sourceに基づくEvidence�E�Owner受理・現在Grantの再利用可否**を�Eけて判定。byte相違だけで全Targetを�E生�Eすると決めず、同一Meaningだけで証拠も�E動復活させなぁE|
| **Generator�E�Toolchain�E�依存版だけ変更** | Ruleが同じでも、実裁E�ETarget・版�E絁E��再検証 | 旧Plan・Artifactの自動受琁E��禁止。�E力Goが同じか、実BuildでどのToolchainを読んだかを別票にする |
| **Source A→B→A、E��択B�E�実行A** | 同byte・同IDでも前回�EAuthorityを継承せず、E��択版とBuild対象を�E示 | 旧ReceiptやGrantを復活させず、実ArtifactとProvider実EntryのIdentityを別照吁E|

**最初に通す二つの負侁E*は、E1) Rule原文を意味が変わるよぁE��変更したのに旧Go�E�旧BuildPlanを使おうとする試行、E2) 説明文のみの変更後に、生成Goの再利用可否とSource-backed Evidenceの再確認を混同する試行とする、Eo BuilderへMarkdownを読ませる暫定対応ではなく、実生成AttemptからBuilderの実�E力へ至る証拠の橋渡しを#4544�E�E2595で実裁E��る。R3の通常`tei` CLIからの選択Provider actual Call、R4の独立Consumer、Studio Stage1→Stage2 Windowsの実�E入は並行P0のまま残す、E
**現状の判宁E*�E�R1.5のチE��ト�Emain収録済み、参照Builderの一連の検査コードもmainに存在する。製品R2で忁E��なRule→Go生�EAttempt→実Builder入力�E連鎖、R3の実Provider、Stage2 Windows、Ownerの業務意味受理は未検収である。この節は実裁E��要求�E墁E��を定めるものであり、Go�E�CLI�E�Windows�E�DB試験を新規に実行した記録ではなぁE��E
### 14.7 Ruleと公開Entryの墁E��を�E確にしてから合流すめE
mainの製品Go回帰`internal/operations/inspect_markdown_rule_entry_boundary_test.go`では、E��択済み`tei.rule-authoring/v0`が`semantic-validation`として正しいSourceを持ってぁE��も、E*同じIDの`semantic-capability`が存在するわけではなぁE*ことを確認する。したがって`sales.order_line.quantity_positive`を生成して実行できても、それだけで呼出し可能な公開Entryは生まれなぁE��Eata側のRule�E�型�E�Relation、Function側の入力�E判断・公開、Business側の状態／�E移とぁE��既存�E責務を維持する、E
**二つの実裁E��ーンを独立して閉じる、E* R2-P�E�E943�E�E2595�E�E4544�E�E951�E��E、E��択済みRule・Generator�E�Toolchain・生�EGoの実byteを既存製品Build Intentへ渡し、Runtimeへ忁E��なContributionがArtifactに収録されたことを実Builderで検証する。R3�E�E4495�E�E1251�E�E1158�E��E、E*既に受理されたpure Function一件**につぁE��公開Entry→現在選択したProvider→実Invocation→通常`tei` CLI�E�独立Consumerを�E行検証できる。R3をR2-Pの完�E征E��にしなぁE��両方の賁E��が揃ってから、Rule由来の生�E実裁E��そ�EFunctionが実際に呼ぶ**結合検査**を追加する、E
| 接続点 | 成立に忁E��な対忁E| 合格にしてはぁE��なぁE��E|
| --- | --- | --- |
| Rule→Function | Functionが参照するRule ID・版、業務上�E用途、評価に忁E��なData Context�E�取得篁E��を�E示 | 同名Rule IDを�E開Function�E�CapabilityのIDとして自動登録 |
| Function→�E開Entry | 正式に公開するEntry ID、�E力�E名前・意味型�E忁E��性、欠損／NULL�E�未取得、Target�E�Scopeとpure�E�effect宣言を対応付けめE| DSL頁E��名とCLI�E�Provider引数名�E不一致、未宣言別名、E��刁E��得から勝手に既定値0を補宁E|
| Entry→Provider実Call | 既孁E951の選択結果�E�Eontract・実裁E�ELock・Artifact�E�に、Hostの現在Grantと実Entry呼出しを照吁E| Selected B�E�Actual A、Plugin Inventoryの発見だけ、旧Grant・旧Binaryからの成功推測 |
| 実結果→Consumer | 同じAttemptで実Providerの入力�E実Outcome・独立Expected・受領�Eを対応付けめE| CLIの期征E��固定�E力、同じ結果の手書き実裁E��Reference Binaryを製品Consumerとして算�E |

Ruleは他�EFunction、標準Operator、Table Script�E�Data Rule、Node FlowからめE*再利用可能な判断部品E*である。ただしRuleの正本をFunctionやNodeへ褁E��しなぁE��EodeのPort名�E型とFunction Entryの入力名を結�E場合も、画面の配線やLabelだけをBindingの根拠にせず、型付きOperationの入出力と忁E��Contextに投影する、EPI公開�E外部Trigger・Effectful HookにつぁE��は、�E示皁E��公開賁E��・現在Grant・Effect�E�Receiptを後続で検収する、E
結合検査の最小反証は、E*RE-01** Rule正本は存在するがFunction Entryが未公開、E*RE-02** Entryの引数名�E型がFunction�E�Data Contextと不一致、E*RE-03** NULLと未取得�E扱ぁE��Providerで潰れる、E*RE-04** R2-Pが生成実裁Eを収録したのにR3が旧実裁Eを呼ぶ、E*RE-05** Rule変更後に旧Artifactと旧Consumer結果を流用する、である。これらは新しいScenario�E�権限正本を作らず、既孁E1180�E�E2514へ負例�E追加として渡す、E
最初�ER3でpure Functionの実呼出しを閉じる間も、E*Studio Stage1実TEI→Stage2実Windows WebView2**�E�E2831�E�E4583�E��E独立した最上位P0とする。本節は製品で実在するRule・Reference Builder・Entry拒否チE��トを踏まえた設計�E引渡しであり、正式製品BuilderとRule由来Functionの統合実行を新たに受理したとぁE��報告ではなぁE��E
**実裁E�Eロファイルの誤用を防ぐ、E* 現衁E`internal/plugin/implementation/model.go` の `static-response-v1` はDeployment Executorの固定応答向けであり、Rule�E�Functionの汎用実裁EontractではなぁE��ここへ期征E��る判定JSONを置ぁE��も、生成Goを選択Providerが実行した証拠にはならなぁE��`composition.Selection` は宣言上�E選択、`composition.Service.Invoke` が返すBindingは実裁E��追跡する観測であり、どちらもHostの現在GrantではなぁE��R3の最小実裁E�E、既存�E共通Application APIと正式なProvider選択�E接続を先に確認する、E
**製品CLIの現在の基線、E* `cmd/tei/main.go` の未宣言 `dbIntrospectSemanticFlag` で実CLIがビルド不�Eだった問題�E[PR #4940](https://github.com/hat47x/tei-platform/pull/4940)で修復してmainへ統合した。同PRのexact HEADではAction Surfaceの実CLI、独立Node Catalog、Source→Web E1-A、製品Skill、Cache、脆弱性確認が成功。一方、Studio�E�Issue-to-Design�E�Self-buildの失敗�E残り、�E体Healthの成功も確認してぁE��ぁE��これらのCI結果を、正式R2のBuilder実選択、R3の公開Entry実Provider Call、Studio Stage2の受�Eと混同しなぁE��E
### 14.8 異なる宣言表現から同じGoを生成し、�E典の賁E��は別に残す

[実Go回帰の提桁EPR #4973](https://github.com/hat47x/tei-platform/pull/4973)では、同ぁE`sales.order_line.quantity_positive` を、�E示選択しぁE`tei.rule-authoring/v0` のMarkdownと従来の `validation` 構文から、それぞれ独立して `Workspace.Load` ↁE型解決 ↁE`GeneratePositive` に通す、Eanonicalを片方からコピ�Eせず、同一の `semantic.Validation` と解決済み垁E`std.decimal` であることを確認したうえで、生成Goのbyte完�E一致を判定する、E*これは作�E中の製品回帰であり、exact HEADのCI合格はまだ未確誁E*である、E
ただし、生成Goが等しぁE��らとぁE��て二つの原Sourceの賁E��を統合しなぁE��選択されたReader、Source Indexの定義位置、ファイル原byte持E��、Model Revisionはそれぞれ保持する、Earkdownの説明文だけを変更した場合、意味と生�EGoは不変であっても古いSource Indexの現行性は失効する。旧Source Evidenceの再確認と、生成Artifactの再利用の可否は刁E��て判断する。送E��、Rule条件・参�E先を変えた時は意味と生�EGoの両方を�E検証する、E
こ�E性質をTEIドッグフ�EチE��ングの検証原則として採用する、E*実裁E��ら抽出した型�E判断・制紁E��、DSL�E�Markdown�E�図表の異なる表現を経ても保存され、�E生�Eによって確認できること**を要求する、Eraphでは画面上�ENode�E�Edge IDと業務Relationの識別子を混同せず、Canvasの配置変更だけで業務Goの意味を変えなぁE��褁E��の意味等価な表現を増やす際にも、新たな意味正本・Reader�E�Generator�E�Builder・Plugin Registryを増やさず、標準データ構造と既存Operator契紁E��再利用する、E
こ�E回帰が�E功しても、正式製品Builderによる実byte消費�E�E2-P�E�、�E開Entryを介した実Provider�E�通常製品CLIの呼出し！E3�E�、Windows Studioの実ConsumerとStage1→Stage2は別の検収である。原文A→B→A、E��択B�E�実Call A、Ownerの業務意味受理と現在Grantは、既存�E第14.6、E4.7節および#4544�E�E4495へ引き続き委�Eる、E
### 14.9 従来構文と選択済みRule Markdownを混ぜずに診断する

R0の入口にも、実Providerと同じ「選択された実裁E��けが入力を解釈できる」とぁE��墁E��を適用する。従来の`.tei`意味ファイルでは`validation`が既存�E規則であり、`rule`は自動的に別名扱ぁE��なぁE��Rule Authoringは、Workspace Manifestが`tei.rule-authoring/v0`を�E示選択した`*.tei.md`だけを受け持つ。未選択�E`*.tei.md`、E��常のMarkdown、管琁E��象外�ESource、従来`.tei`の`rule`を同じものとして読み替えなぁE��E
こ�Eため、従来`.tei`の`rule`を検�Eした場合�E単に「不�Eな宣言」とするのではなく、E*定義したファイルと対象語�E原byte篁E��を示したぁE��で、Rule Authoringの選択条件と置き場所を案�Eする**。エラーの改喁E�E、Readerの自動選択�E意味の受理・旧構文の暗黙変換ではなぁE��未知語�E引き続き未知語として拒否し、構文の未受理を半刁E��け読み込んだモチE��に変えなぁE��E
負例�E既存`internal/syntax/rule_placement_test.go`と`internal/workspace/rule_placement_source_diagnostic_test.go`で扱ぁE��E��択�E未選択�E従来構文の回帰は既存`tei_md_source_guard_test.go`と製品ReaderのチE��トを流用する、EIの監視対象めE*現在存在する`rule_authoring_markdown_loader.go`�E�`parser.go`�E�実テスチE*と一致させ、廁E��した旧guardファイルへの参�Eで失敗させなぁE��これによってR0の診断品質を改喁E��ても、R2-Pの実Builder選抜、R3の正式Provider実Call、Stage1→Stage2 Windowsの実証を合格扱ぁE��なぁE��E
### 14.10 生�ERuleを通常CLIのBuild Intent経路で実ビルドすめE
参�EBuilderを直接呼ぶ§14.4の回帰に加えて、`cmd/tei/cli_self_build_generated_rule_go_test.go`では、E��択済みRuleの正値判断Goを一時的なGo commandとして置き、E*通常の`tei self-build`のBuild Intent解決と実Builder起勁E*を一つの試行に通す、ELIとBuilderは新設せず、既存`writeFullChainDeclaration`、`authorGoCommandCatalog`、`buildintent.Resolve`、`reference-go-build-from-plan`を利用する、E
入力�E実Fixtureの`tei.rule-authoring/v0`が読み込んだ`quantity_positive`と、解決した属性型`std.decimal`に限る。`GeneratePositive`の実byteをGo packageに保存し、そのpackageと実依存closure、`go.mod`、`go.sum`をBuild IntentのSourcesに列挙する、Eo package名をアルファベット頁E��した場合に`definitionSource: pkg0`がEntryを表すとは限らなぁE��め、既存テスト用宣言生�Eヘルパ�Eでは**実EntrypointのSource IDを選択すめE*。これ�E実BuildPlanを正しく導�EするためのチE��ト修正であって、第二Resolverの追加ではなぁE��E
CLI側の独立観測は、実Entry冁E��のIntent解決、BuilderのPlanとの照合、Artifact Binding、同じPlanからの再現性、E��離Go Source閉包と実Binaryの生�Eを別、E��要求する。最後に**CLIが�E力したBinaryそ�Eも�E**を起動し、正値・ゼロ・負値・NULL・未取得を一回ずつ読み取る結果に照合する。`go run`だけ�E検査めE��子Builderの成功自己申告だけでは合格にしなぁE��E
こ�E回帰で証明できるのは「製品CLIの通常`self-build`経路が、E��択されたRule由来のGo bytesをコンパイルし、そのBinaryが判断する」とぁE��限定的なR2-Pの一部である、E*Rule原文、Reader版、Generator実裁E��と生�EGoを同一Build Intentの受理済みSourceとして束縛したわけではなぁE*、Euthoring→Generatorの出典失効、E��常製品が受理したBuilderのAuthority、正式�E開Function、E��択Provider実Invocation、現在GrantとOwner意味受理、Windows Stage1→Stage2は引き続き独立賁E��として残る。試験�ELinux/amd64の既存参照Catalogで行い、ほか�ETargetの適合を主張しなぁE��E
次の実裁E��刁E�E、新しいGeneratorやBuilderを導�Eすることではなく、E*同一Attemptで選択したRule�E�Generatorの賁E��と、E��常CLIが実際に消費したBuildPlan�E�Artifactの賁E��を結合すること**に絞る。その後、E�14.7の正式Function公開Entryと実Provider Callに結合し、E��択B�E�収録A、生成Go改版／旧Entry実行、同値結果だけ�E誤受理をそれぞれ拒否する、E
