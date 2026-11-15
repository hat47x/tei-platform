# Rule Authoringから実Plugin・製品Consumerまでの第一縦断：実装と検収の接続計画

- **状態**：2026年11月15日の設計・実装引渡し。新しいGo生成器、実Plugin呼出し、Windows Stage2の受入をこの文書では主張しない。
- **優先対象**：内蔵Rule Authoring Markdown Readerで読み取った一つのpureなValidationを、生成Goの判断処理、実Builder、選択済みProvider、通常製品CLIまでつなぐ。
- **前提となる実装事実**：mainの[7921b005](https://github.com/hat47x/tei-platform/commit/7921b005a44d325983d24ca2b8b58b0737dd778d)で、明示選択されたtei.rule-authoring/v0を内蔵ReaderがWorkspace Loadの原文から読み、semantic.ValidationとSource anchorへ投影する経路が追加された。これは外部Language Pluginや正式Graph Readerの製品受入ではない。
- **既存の正本**：[Pluginの役割・合成・Stage2再入](PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md)、[Plugin-firstの実運用検証](PLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.md)、[自己記述](ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)、[製品完成計画](../workstreams/PRODUCT_COMPLETION_PLAN.md)を維持する。本書は一機能の実装差分を決める補助計画であり、第二のContract、Resolver、Meaning、Builder、Runner、Evidence Storeを作らない。

## 1. 最初の成果物は「Pluginらしい定義」ではなく、生成した判断を実際に呼べること

最初の機能は、入力の型・NULL／欠損・ゼロ・境界値を持つ**副作用のないValidation一件**とする。Graphの同名Nodeと多重有向Edgeは、第二例として既存の固定六入力をそのまま利用する。Graphから始めると、現在の内蔵Rule Readerと未接続のGraph Readerを同時に完成させる必要があるため、第一実装はRuleに絞る。

専門家が読むSourceは条件・入力・結果・説明を自然な順序で表現し、Goのstruct、func、型引数や生成器固有の都合をDSLへ持ち込まない。入力の欠損、NULL、型不適合、ゼロ、正値について、結果だけでなく「なぜ判定したか」を参照できるようにする。ただし判定不能を便宜上falseや不適合に落とさない。

成功は、生成Goに実際の比較・分岐があり、Sourceの条件一箇所を変更すると**生成コードと実行結果がともに変わる**ことを、独立した期待値で示した場合に限る。呼出し用Wrapper、固定した期待JSON、手書きGoの規則への委譲は、生成した業務判断として数えない。

## 2. Pluginは配布物ではなく、どの契約へ何を提供したかで扱う

| 観点 | 第一縦断で固定する内容 | 同一視しないこと |
| --- | --- | --- |
| Authoring | 選択済みLanguage Implementation、管理Sourceの原byte、定義位置 | File名から推測したParser、未管理Draft、表示用Markdown |
| Meaning | 既存semantic.Validationの型付き候補、受理したRule／Operatorの版 | Readerの読み込み成功、Source anchor、Ownerの意味受理 |
| Generation | 対象契約・Target・選択Generator・実生成した比較／分岐 | Goファイルが存在したこと、生成器の自己申告 |
| Build | Build Intent、入力閉包、Builderの実入力、ArtifactSetとDigest | 別作業でビルドした同値Binary、別HEADの成功 |
| Runtime | 選択されたProvider、実Entry、現在のScope・Grant、Call結果 | Packageが入っていること、exact Lockだけの存在 |
| Surface | 通常tei CLIのTyped Operationと独立Expected、後続Studio投影 | テスト専用CLI、Reference Go実行、画面上の表示だけ |
| Knowledge | 公開可能と判定した契約・Source-backed Manualの実Read | Inventoryの発見成功、マニュアル候補、公開推測 |

Authoring／Generator／Runtime Provider／Presentationは**利用場面での役割**であり、Coreの固定Plugin種別ではない。同一Packageが複数役割を持ってよい。一方、Build時だけ用いたGeneratorをRuntimeへ無条件に含めず、契約ID、実装ID、Plugin ID、業務上のCanonical IDを混ぜない。

## 3. 三つの実現方式は同じ意味契約に対する代替とする

| 方式 | 適用と検査 | 初回の扱い |
| --- | --- | --- |
| **静的合成・コード生成** | 配布するGo単一Binaryへ必要な生成実装とRuntime Providerを収録。選択された機能だけをBuild閉包に含め、更新時はArtifactと実Consumerを再資格化 | **製品の第一候補**。TEI Studioの通常Desktop配布・生成アプリと整合 |
| **独立Process Provider** | 既存のProcess Runtime境界で契約Version、呼出し権限、観測、失敗・不確定作用を扱う。分離実行が必要な非信頼拡張や独立更新に限る | pureな第一例の必須条件にはしない。第二方式の比較検証に使う |
| **外部HostへのTarget-native生成** | .NET／Java等の既存HostへAdapterと実装を生成し、Hostの認証・認可・Lifecycleに接続。TEI専用常駐Hostを要求しない | Go製品の受入後に一Contractを異Targetへ再利用できるかを確認 |

Goの動的Pluginロードを標準の更新・配布機構として前提にしない。静的合成が満たせない「実行中の独立更新」や「隔離」の要件が具体化した時点で、Process方式と比較する。方式の選択はTargetとBuild／Runtime閉包の一部であり、同じPlugin IDだから全Targetで実現できるとはしない。

## 4. 実装パケットと一Attemptの証拠

**R0：Readerの製品回帰（#2621／#1018）。** 既存内蔵Readerで、選択済みSourceの同一Load byte、Source anchor、旧tei定義の互換性を検査する。2026年11月15日にmain統合済みの[PR #4890](https://github.com/hat47x/tei-platform/pull/4890)は実Load、二Rule、Source Indexの元byte位置、説明文変更による旧Index失効、未選択・契約版違いをGo試験へ追加し、[PR #4891](https://github.com/hat47x/tei-platform/pull/4891)は同じLoadからInspector／TraceへRuleの定義位置を照合する試験を追加した。**両PRの統合と試験コードの存在は確認済みだが、当該新HEADのCI成功は別途確認する。** 実装欠落、複数候補、管理外Draft、Source A→B→A、既存tei定義の互換性など、未確認の条件は個別に残す。内蔵Readerの成功を外部Language Pluginのexact Resolver／Lockの受入へ昇格させない。

**R1：受理Meaningから判断Goを生成（#1121／#4544）。** 同じSource occurrenceの一Ruleについて、入力の型と欠損を明示し、条件を含むGo実装を出力する。実際に生成したSourceと選択Generatorの版、受理Rule、実生成箇所を結ぶ。業務上のMeaning受理が欠ければ、その資格はunknownのままにする。

**R2：実BuilderとPlugin Contributionを接続（#943／#951／#1016）。** Build-only GeneratorとRuntime Providerの閉包を分ける。現在のBuild Intentに対象を表せない場合は、既存Build Execution／ArtifactSetへ最小の適合Adapterを検討し、架空のPlugin Subjectや第二Builderで穴を埋めない。実Builderがどのbyteを読んだかをArtifact Digestと区別して記録する。

**R3：実Providerを通常製品CLIから一度呼ぶ（#1251／#4495／#1158）。** CLIの公開操作は既存のTyped Application Operationの投影とし、別の意味Dispatcherを作らない。選択Provider Bに対してAが動いた場合は、結果が同じでも失格。初回はpure／read-onlyとし、実作用のあるProviderの許可には進まない。

**R4：第二Consumerとセルフビルドへ進める（#1824／#2831／#4583）。** R3を通した同じ意味契約をStudioのread-only Inspectorなどから確認する。別途、正式な非Plugin Desktop Build Subject、Stage1 TEI実Processが生成したStage2、実Windows Wails/WebView2を一つの受入経路で検査する。CLIの二回起動や局所部品の再生成をDesktop Stage2成功と呼ばない。

各段階で既存Source Index、World／Model／Artifact Revision、Plugin Lock、Evidence Packet、Execution Attempt、Actual Consumerを**参照で結合する**。新しい一括State、永続証拠表、万能PASS判定は導入しない。

## 5. 効果を伴うPluginへ拡張する際の境界

pure Ruleの実装が通った後に、DB更新／通知など外部作用を伴うProviderへ拡大する。その際も既存Command／Effect／Failure／Receipt契約を使用し、Lifecycle GraphはGuard・順序・実行主体の表現に限る。

- Previewで候補と必要情報を確認し、Resolveで実Providerを固定し、Admitで現在AuthorityとGuardを検査してからCommitへ進む。Observeで実際の結果を照合する。この並びは検討中の意味上の整理であり、新Runnerの五Phaseを導入するものではない。
- 既存Hookには、宣言したusesの先行呼出しやGuard経路との二重到達の可能性がある。Guard拒否前に外部作用が実行されないこと、許可したCall回数が上限を越えないことを実Providerで検査する。単純に既存HookをGraphへ写しただけでは安全な実装と認めない。
- 実Providerへ到達後、ACK喪失・結果不明・部分効果が生じたらeffectUnknownを維持し、既存Receiptのlookup-firstに従う。未観測を「作用なし」「安全な再試行」と扱わない。
- Runtimeの現在Grant、業務意味のOwner受理、Source／Artifactの一致、外部公開DocumentのDisclosureは別資格である。Local Studioのno-auth既定を共有Web／外部Hostへ引き継がない。

## 6. 追加する独立Expectedの差分（新しい試験正本は作らない）

以下は既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)／[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ引き渡す候補であり、未実行である。

| ID | 一因子の変更 | 実際に確かめること |
| --- | --- | --- |
| PLV-01 | Ruleの比較条件のみ変更 | 生成Goの判断箇所と実Consumer結果の双方が変わる |
| PLV-02 | 入力を欠損／NULL／0／型不適合へ変更 | 欠損と不正値とfalseを混同せず、宣言外の補完をしない |
| PLV-03 | Markdown装飾・表示位置のみ変更 | Source再確認を残し、意味やArtifactの不要な失効を分ける |
| PLV-04 | 同一IDのSourceをA→B→Aで変更 | 旧意味受理・生成資格・現在Grantを自動復活させない |
| PLV-05 | Reader未選択／曖昧／実装なし | 不正な部分Worldや暗黙のGo標準Reader fallbackがない |
| PLV-06 | 選択ProviderをA→B、実CallをAに固定 | 結果同値でも未適格として拒否する |
| PLV-07 | Builderが別版のGenerated Goを消費 | Source／Artifact／Builderの不一致を検出する |
| PLV-08 | Build-only GeneratorをRuntime閉包に混入 | 不要な依存混入を検出し、Runtime向け最小構成と検証済み構成を区別 |
| PLV-09 | Guard拒否前にeffectful Hookのusesが呼ばれる | 外部作用0回を要求し、満たせなければ適用を停止する |
| PLV-10 | 実Call後のACK喪失 | Effect unknownを保持し、無条件に再実行しない |
| PLV-11 | Plugin DocのLockだけ一致、SourceRef／Audienceが欠落 | 文書の閲覧・再公開・Skill手順を現在適格と表示しない |
| PLV-12 | Stage0がStage1／Stage2相当のBinaryを別々にBuild | 実Stage1→Stage2再入の成功へ昇格しない |

## 7. 実装順序、Owner、停止条件

**最短の実装順は R0のReader負例 → R1の生成判断 → R2の実Builder実消費 → R3の通常CLI実Call** とする。第一PRはR0のみ、第二PRはR1〜R3のうち独立Expectedを通せる最小範囲に分けてよい。未対応部分が残る場合は、それを明示して次の差分に送る。R4のWindows Desktop Stage2は並行する製品P0であり、R0〜R3を理由に優先度を下げない。

RuleのMeaningの正本は#2620／#988、Language選択は#1018、Target Loweringと実生成は#1121／#4544、Plugin選択は#951／#1016、通常CLI実Callは#1251／#4495、Desktop Self-buildは#2831／#4583が担当する。受入用の判断基準は#1180／#2514へ返す。Plugin文書は#4679／#4714、Lifecycle Effectは#4198／#1174に残す。

方式を縮小する条件は、（a）一Ruleのために第二のResolver／Runnerが必要になる、（b）生成コードが手書き判断に依存してSource改訂に反応しない、（c）実Builderの入力や選択Providerを追跡できない、（d）意味の欠損・現在Grant・Effect unknownが成功に丸められる、のいずれかである。その場合は抽象化や対象Domainを増やさず、既存の実行境界へ戻して一機能だけを完了させる。

**計測**：固定した機能義務を分母に、TEIから生成した判断、委譲、手書き、未実装を別々に記録する。同じ義務の改訂前後で、保守するRule数、手書き変更箇所、重複する定義、独立検収までの作業とConsumer数を比較する。Plugin数や生成Go行数の増加だけをセルフビルドの進展とはしない。

この文書は実装計画である。mainのReader実装を超えるGo／CLI／Plugin／Windowsの新実行は行っておらず、Ownerによる意味受理も未確認である。

## 8. 他領域と共通化するのは「対象と資格の受渡し」である

Plugin、Studio、SHO、SUI、YIC Harnessの議論を通じて、同じ情報を受け渡すために巨大なContextや独自Agent Runtimeを作る必要はないと整理された。まず一つのSubjectについて、**何を選び、どの版のどの情報を読み、どの操作を依頼し、どこまで観測できたか**を共通の型付きOperationと既存Evidenceで表す。AIへの説明やSkillは、この確実に確認した部分から必要に応じて構成する。

| 受渡しに必要な閉包 | 確認する内容 | 不適格な状態 |
| --- | --- | --- |
| Subject／Information | Workspace・Canonical Subject・Role・Source occurrence・Required／Loaded／Boundの対応と取得範囲 | 別Workspaceの同名Subject、Source A→B→A、partialを完全読取と誤認 |
| Instruction／Documentation | 実CLIの発見結果、選択ContractとPlugin版、Source-backedの公開Manual、必要時取得した短いSkill | 旧コマンド、合成Fixtureの仮想操作、未確認文書を実行手順として提示 |
| Capability／Authority | 選択実装と実Artifact、実Provider Entry、Hostの現在Grant、Effectの実観測、Consumerの受領 | exact Lockだけで実行許可／開示資格があると誤認 |

既存の[共通Application API](https://github.com/hat47x/tei-platform/blob/main/plan/adr/0048-shared-application-api-and-transport-adapters.md)を入口に、CLI、MCP、Studio、Webは同じ型付き意味操作の別投影とする。MCPが必要なら同じOperationをTool／Resourceへ公開するが、独自Agent Loopや常駐TEI Hostを追加する理由にはしない。公開文書についても、Plugin InventoryのID・Lock一致と、文書SourceRefを実読取りし現在Audienceに開示してよいことは別判定である。#4679／#4714へ帰属させ、R0〜R3の新しい先行条件にはしない。

## 9. 操作対象・表示・Source編集は同じ識別子ではない

StudioのWails／WebComponents、SHOの図表、SUIのカードやKJ空間、AI Harnessからの対象引渡しを考えると、**意味上の対象IDと描画ID、Source位置、現在の編集対象**を分ける必要がある。

Graphの同じ表示名のNode二件、同一端点の多重有向Edge、業務上のRelation IDは、それぞれ独立した同一性を保持する。Sourceを表示できても、編集すべき一つのRelationが確定したことにはならない。明示Pinは対象選択の意図であり、現在Grantや外部送信許可ではない。複数候補が残った場合は候補を示し、AIにも一意の編集対象として渡さない。

Frontendは業務データの受理済み意味と表示の糖衣を分離する。Markdown表、Graphviz、Mermaid、Web Componentsの構造やCSSレイアウトは、原文を保存しながら複数の表示を可能にする。ただし、図の配置やCSS変更を業務Ruleの変更として扱わない。逆にEdgeの向き、Ruleの条件、FieldのNULL扱いが変わる場合は、表示上の差だけに押し込めず、生成コードと独立Expectedまで追跡する。

UI実装におけるTypeScript・Web標準技術・Wailsの選択は#1824の担当とし、このPlugin設計は**Target-neutralな表示・操作契約と、必要な実行資格を持つConsumer**までを定義する。生成したStudio Componentが実WebViewへ収録されたことは、Source生成やNode側のテストとは別に、正規Bundle・Go embed・Windows実Consumerで確かめる。

## 10. 次に着手する差分を、現在の製品到達点に合わせて絞る

内蔵Rule Readerとその実Load回帰がmainへ入ったことで、製品の第一作業は「汎用Markdown Readerを新設する」ことではなくなった。**第一実装PRは既存Readerで受理候補となる一Ruleを固定し、その条件が生成したGoの判断処理を決めること**に絞る。生成器の自己記述・外部Language Resolver・全Graph／Window・公開Manual・Wails Updaterは、初回の成功条件へ載せない。

検収では、同じRuleの入力だけを変えたケースと、Rule定義自体を変えたケースを区別する。前者は型・NULL／欠損・0・境界値で結果がどう変わるか、後者は**生成コードの分岐そのものが変わるか**を調べる。どちらも、生成器が返す値をExpectedの根拠にせず、既存#1180／#2514の独立Expectedと照合する。

次のPRではBuilderがこのGoを実消費し、選択されたProviderの実Entryを通常CLIの共通Application APIから呼ぶ。実Callが成立した段階で、別Targetに同じContractを実現させる圧力試験を検討する。**Studio Desktopの正式なStage1実TEI→Stage2実Windows Wails/WebView2は、引き続き独立かつ最上位の製品P0**である。開発用CLIが二度起動した事実、ReferenceのGo成功、旧PRのGreen CIを寄せ集めてStage2成功にしない。

### 追加の反証（既存Independent Expectedに引き渡す）

- **PLV-13**：PinされたSubjectと実CLIの操作対象が別Workspaceの同名IDになる。結果一致でも受領不適格。
- **PLV-14**：製品InventoryのLockは一致するが、実ManualのSourceRef／Audienceが未確認。Skillの使用手順は現行実装の保証にならない。
- **PLV-15**：同じ意味Graphに対して表示用Edge IDと正式Relation IDを混同する。編集対象未確定のままPatchを行わない。
- **PLV-16**：生成TSとNode側試験は新しくてもWailsに埋め込まれたAssetが古い。実Consumer資格は成立しない。

**記録上の注意**：PR #4890／#4891のmain統合は製品Reader回帰の進展であって、外部Language Plugin、生成Plugin／実Provider、Studio Stage2、Owner意味受理の成功ではない。本追補は文書のみであり、追加したPLV-13〜16の試験実行や新しい製品実装を含まない。
