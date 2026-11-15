# Rule Authoringから実Plugin・製品Consumerまでの第一縦断：実装と検収の接続計画

- **状態**：2026年11月15日の設計・実装引渡し。R1の限定的な正値Rule→Go生成は[PR #4911](https://github.com/hat47x/tei-platform/pull/4911)でmainに統合済み。Rule改版による再生成・別Process実行を追加検証する段階であり、正式Builder／実Plugin呼出し、Windows Stage2は未受理。
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

**R1：型付きMeaningから判断Goを生成（#1121／#4544）。** mainの[PR #4911](https://github.com/hat47x/tei-platform/pull/4911)で、実選択したMarkdownの正値Rule（`std.decimal`）から`value.Sign() > 0`を含むGoを生成し、別Go Processで数値／NULL／欠損を検査する局所実装が入った。ただし正式Builderによる生成byteの採用、Generator版の配布資格、Ownerの意味受理、実Plugin／Providerは**未検収**のままとする。

<!-- rule-read-target-mutation-vs-generation-qualification-20261115 -->
**R0→R1を結ぶ実Read変更の対照**：mainの[#4890](https://github.com/hat47x/tei-platform/pull/4890)／[#4891](https://github.com/hat47x/tei-platform/pull/4891)／[#4901](https://github.com/hat47x/tei-platform/pull/4901)で、選択済みRuleの実Loader・Source Index・既存Evaluatorの局所Readまで到達した。次の[Go回帰](../../internal/operations/selected_rule_markdown_mutation_test.go)は、**同じRule IDと`positive`を保ったまま、参照Attributeを`quantity`から`line_number`へ一箇所だけ変更**し、再Loadされた`semantic.Validation`、Evaluatorが実際に要求する`Number`の引数、判断結果、旧Source Indexの失効と新Indexの成立を一組で比較する。判断結果の差にはテスト専用の合成数値（数量2、明細番号0）を用いるため、実DBのProvider ReadやCoverageの成立ではない。Rule ID一致・旧Source指紋・旧評価結果を理由に生成Goの再利用を許可しない。R1では、このSource変更に対して**生成器が実際に作ったGo判断箇所の変更**を追加で検証し、R2では実Builderの消費byte、R3では通常CLIの実Provider／別Consumerまで同じ因果関係を接続する。意味受理は引き続きOwner判断である。

<!-- rule-read-target-mutation-vs-generation-qualification-20261115 -->
**R0→R1を結ぶ実Read変更の対照**：mainの[#4890](https://github.com/hat47x/tei-platform/pull/4890)／[#4891](https://github.com/hat47x/tei-platform/pull/4891)／[#4901](https://github.com/hat47x/tei-platform/pull/4901)で、選択済みRuleの実Loader・Source Index・既存Evaluatorの局所Readまで到達した。次の[Go回帰](../../internal/operations/selected_rule_markdown_mutation_test.go)は、**同じRule IDと`positive`を保ったまま、参照Attributeを`quantity`から`line_number`へ一箇所だけ変更**し、再Loadされた`semantic.Validation`、Evaluatorが実際に要求する`Number`の引数、判断結果、旧Source Indexの失効と新Indexの成立を一組で比較する。判断結果の差にはテスト専用の合成数値（数量2、明細番号0）を用いるため、実DBのProvider ReadやCoverageの成立ではない。Rule ID一致・旧Source指紋・旧評価結果を理由に生成Goの再利用を許可しない。R1では、このSource変更に対して**生成器が実際に作ったGo判断箇所の変更**を追加で検証し、R2では実Builderの消費byte、R3では通常CLIの実Provider／別Consumerまで同じ因果関係を接続する。意味受理は引き続きOwner判断である。

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

## 11. 最初の生成対象を実Fixtureで固定する

設計上の抽象例ではなく、mainへ統合された実Fixture `internal/workspace/testdata/tei-md-rule/` を最初の入力にする。

- **Source**：`semantic/sales/order-line-rules.tei.md` の `rule sales.order_line.quantity_positive`。定義は `sales.order_line.quantity is positive`、元位置は既存Source Indexに帰属する。同文書の `line_number_unique` と説明用 `text` ブロックは最初のGo生成対象に含めない。
- **型の根拠**：隣接する `semantic/sales/order.tei` の `OrderLine.quantity type std.decimal` と対応付ける。Markdownに書かれていないGo型や丸め方を生成器が独自に推定しない。型解決できない場合は、仮の数値型で成功させず診断する。
- **期待する対照**：数量1、0、-1について正の数かどうかを判定する。NULL、項目欠落、型不適合は、既存の入力・Validation契約に基づく拒否／未評価を明示し、勝手にfalseや0へ正規化しない。特にNULLの期待OutcomeはOwnerの独立Expectedが確定するまで未受理とする。
- **変更追随**：正値条件の一箇所を変更した時に、生成Goの判断箇所、実Builderが消費したbyte、実Provider結果を比較する。変更したSourceを読み直さず生成物だけを更新したケース、生成Goが同じまま手書き実装を修正したケースは失格とする。
- **実行先**：初回の観測は実製品CLIの型付きOperationから選択されたGo Providerへ到達すること。専用のReference runnerを正式な実行入口と呼ばない。成功後にStudioのInspect／Graph表示や他Targetへ展開する。

このFixtureはReader・Inspectorを通した**意味候補の出発点**であり、生成用の正式なRule Operator／Provider Contract／入力のNULL意味がOwner受理済みであるとは限らない。足りない契約は#1121／#4544／#4495の既存担当で確定させ、対象を増やす前に一RuleのSourceから実Consumerまでを閉じる。

## 12. R1の最初の実装差分：生成Goに判断させる

[PR #4911](https://github.com/hat47x/tei-platform/pull/4911)は、既存Readerが正規化した一Ruleから**条件式を含むGo Sourceを生成する**限定実装である。読取り済みの `semantic.Validation` と解決済みのAttribute型 `std.decimal` を受け、`value.Sign() > 0` を含む関数を生成する。生成先の入力・結果は既存 `validationeval.Source`／`validationeval.Result` に従う。`validationeval.Evaluate` への単なる委譲でも、固定JSONの出力でもない。

実Goテストでは、製品の `Workspace.Load` が選択したRule IDとAttribute型を再照合し、同一LoadのSource Indexが指す原文IDを確認してからLoweringする。生成されたGoを別Processで `go run .` し、`2.5`、`0`、`-1`、NULL、未取得について独立Expectedで判断結果と読取回数を検証する。未対応Rule kind、未解決型、`std.money`、余分なRelation、無効なGo package名は拒否し、Ruleの参照Attributeを変えた時は生成コードの読取先が変化することを確かめる。

ただし**生成関数に受理済みSourceやOwner権限を自動的に付与するものではない**。現段階ではGenerator選択版のLock、生成元から実Builder入力までのDigest、正式Plugin Contribution、通常CLIが呼ぶ実Provider、Studioの実Windows Consumerを繋いでいない。PR #4911の試験コードの存在とそのexact HEADでのCI成功も分けて記録する。R1局所試験が通ったら、別のテスト専用Generatorへ拡張するのでなく、#4544の既存Build入口へこの生成Byteを渡すR2を優先する。

<!-- tei-rule-generated-binary-mutation-witness-20261115 -->
### R1.5：同一Rule IDの変更を、生成Goの実行バイナリまで追跡する

[製品Go回帰](../../internal/semantic/validationgen/positive_mutation_binary_test.go)では、`tei.rule-authoring/v0`で選んだ`quantity_positive`のIDと`positive`を固定し、参照先だけ`quantity`から同じ`std.decimal`型の`secondary_quantity`へ変更する。型のために追加した`secondaryQuantity`は両比較で**共通の固定Data宣言**とし、比較中にDataの型定義は変更しない。変更前後の`Workspace.Load`→`semantic.Validation`→`Source Index`→`GeneratePositive`からそれぞれGo Sourceを作り、同じ手書きConsumerで**別々の実バイナリを`go build`・実行**する。Expectedは`quantity:satisfied:1`と`secondary_quantity:not-satisfied:1`で固定し、旧Indexの指紋失効、新Indexの成立、生成Source／Binary byteの相違を検査する。片方のGo Sourceと固定したJSONだけで二出力を演出するテストではない。

ただしこれは**直接GoツールチェーンによるR1.5の局所実消費**であって、製品の選択済みBuilder／Build Intent／ArtifactSet、生成Plugin登録、通常`tei` CLI・実Provider／別ConsumerによるR2・R3ではない。R2の次PRでは、この**同じ生成byte**を既存のBuilder選択とSource/Build入力閉包へ結び、`Selected B/Actual A`・Build後のSource差替え・旧Artifact再利用を拒否する。R3はその正式Artifactを通常CLIで実際に呼び出す。Source IDやBinary SHAの一致だけでOwner意味受理・現在Grant・実Effectを認定しない。テスト追加とHEADのCI合格も別に記録する。

## 13. ノード型FlowとPluginを、同じ標準モデルから展開する

Node-RED／Enebular型のプログラミング、ESPStudioのようなNode UI、データ連携・マッシュアップ・簡易APIに向けても、**Flow専用の別Meaning／Runnerを作らない**。既存の標準データ構造、Operator Contract、Function／Process Flow、Capability Invocationを組み合わせ、GUIのNodeはその投影として扱う。[研究統合§16.18〜16.21](../research/TECH_RESEARCH_INTEGRATION_2026-10.md)の議論と方向を合わせるが、現行Plugin第一縦断R1〜R3より先に新しい実行基盤を導入しない。

| 一つのFlowを構成する要素 | 標準モデルで保持するもの | Node画面だけに持たせるもの |
| --- | --- | --- |
| Operator／Service／Function定義 | Contract、入力／出力の型と意味、Provider候補、作用・読取範囲・認可要求 | 表示名、ノードの外観、アイコン |
| Node occurrence | Flow内の固有ID、選択Operatorと版、Port Binding、パラメータ、出典・変更元 | 画面座標、開閉、ズーム |
| Edge occurrence | 独立Edge ID、端点Port、Data／Control／Event／Effect等の関係Role、向き・多重性 | 曲線の配置・色・線幅 |
| Process／Flow | 入出力、分岐・結合、順序、Trigger、失敗・再試行・補償、State・Receipt契約 | キャンバスのグループ、折畳み・表示密度 |
| 実行と公開 | 静的生成先のGo／外部Host、同一Application API、実Provider・AuthZ・入出力Scope | Studio、Web、CLI、MCP等の操作投影 |

ノードを「データ取得」「変換」「判断」「外部API」「出力」「通知」などで分類できるが、これは**Catalogの検索・表示用Facet**であって、Coreへ新しい排他的なPlugin Role enumを増やすものではない。一つの部品はCLI／DSL／Function Chain／Graphのいずれからも同じContractで利用できる。Flowが一時的な状態を持つ場合も、現在のState／Lifecycle／Effect境界に委ね、GUI実行エンジンに秘密の状態表を持たせない。

### 最小の検証例：数量判定の簡易API

1. 入力データ `OrderLine.quantity`（Decimal）を受ける。
2. 型付き入力の欠損・NULLを区別し、既存の `quantity_positive` Ruleを判断Nodeとして使用する。
3. 結果が成立なら正常応答、不成立なら業務上の不成立を返し、入力未取得・型不適合は評価不能として分離する。
4. Flowの表示を変更してもRuleの意味と生成Goは変化せず、Ruleの参照Attribute変更では生成物と実Callが変わる。
5. 静的Go生成の第一経路で実Consumerへ到達後、HTTP Trigger／API公開／外部Service Adapterを追加する。API公開時にはHost認証・現在Grant・外部送信先・Rate Limitを別途明示的に受理する。

**UI実装との接点**：Web標準UI設計[§12](WEB_STANDARD_COMPONENT_AND_AGENT_HARNESS_2026-11.md)では、こちらのN0〜N4とStudioのW0〜W4を同じMeaning/Operation/Sourceからの二つのConsumerとして整理する。UI-P1はread-only Inspectorの公開EventとAdapterの境界、FLOW-P1は既存Ruleの型付きPort/Graph→生成Goであり、片方の製品検収をもう片方のPASSに数えない。新Graph Engine／Renderer／Registryは導入しない。

実装順は**N0（既存Rule／OperatorをNodeとしてread-only投影）→N1（型付きPort／Flow検証と一つの分岐のGo生成）→N2（実Builder／実Provider／CLIを通した一Request）→N3（Studio Node GUIで同じFlowを開き、変更と再生成を検収）**とする。外部Trigger／Cloud APIはその後のN4とし、N1のために独立Flow DSL・汎用Stream Runner・SaaS前提のTEI常駐Hostを導入しない。

**独立反証候補（#1180／#2514）**：NG-01 同じNode名に別Occurrence ID、NG-02 同じ端点に多重Edge、NG-03 Portの型不一致、NG-04 部分取得を正常な空集合と誤認、NG-05 Graph layoutだけ変更して業務Goが変化、NG-06 Rule変更でも古い生成Goを実行、NG-07 API Publish権限と実外部送信許可を混同、NG-08 Effect不明なのに再試行して二重送信、をそれぞれ独立に拒否・失効・隔離する。

**資格の区別**：現時点でR1の実装PRは存在するが、N0〜N4、HTTP API実公開、クラウドConnectorの製品受入を実装済みとは扱わない。R2／R3が正式な実消費になっていない段階でNodeの可視化だけを「高度な実行機構」として宣言しない。

## 14. R2で実Builderの生成Go消費を閉じる（2026年11月15日）

### 14.1 R1の実装を、R2の完了と誤認しない

現行mainの `internal/semantic/validationgen/positive.go` は、選択済みの `semantic.Validation` と解決済み `std.decimal` を受け取り、`validationeval.Source.Number` の呼出しと `value.Sign() > 0` の判断を含むGo Sourceを生成する。`positive_product_test.go` は、そのGoをリポジトリ配下の一時的なGo packageとして `go run .` でコンパイル・実行し、正値・ゼロ・負値・NULL・欠損と未対応Rule／型の拒否を検査する。これは**実Go判断生成の証拠**であり、既存Builderが同じbyteを入力として受領し、配布するArtifactに含めた証拠ではない。

生成Goは現状、`tei/internal/semantic/validationeval` に依存する。Goの `internal` import境界を越える別モジュール／外部Hostへ、生成ファイルをそのまま移しても一般には利用できない。**第一経路はTEI Goモジュール内での静的合成**に固定し、外部Host向けは意味契約に適合する公開API／Target-native Adapterを別に生成・検証する。Go `internal` の公開化や第二RuntimeをR2の前提として進めない。

### 14.2 Builderへ引き渡すのはファイル名ではなく、同一Attemptの実入力である

| 確認箇所 | 初回一件で要求する証拠 | 停止・失格の条件 |
| --- | --- | --- |
| Ruleの選択 | `internal/workspace/testdata/tei-md-rule/semantic/sales/order-line-rules.tei.md` の `sales.order_line.quantity_positive` と `order.tei` の型を同一 `Workspace.Load` から取得。Source Indexのbyte位置も照合 | Parserを直接呼んだ候補、別版のRule、同名の未管理Draftを正式入力にしない |
| 意味とGenerator | `semantic.Validation`、解決済み `std.decimal`、実際に選択した `GeneratePositive` 相当の実装と版、Target用package名・Toolchainを区別 | Owner意味受理の不在をBuilderが補完しない。未対応kind／型／packageは生成前拒否 |
| 生成byteとBuilder入力 | R1が生成したGo byteのDigestと、既存Build Intent／Builderが**実際に読み込んだbyte**のDigestを比較。生成SourceはBuilder所有の選択済み入力集合へ接続 | 同じpathを申告していてもbyte相違、別Attempt、古いGenerator、別Toolchainなら失格。Builderの自己申告だけで一致としない |
| Artifactと収録 | Builderが生成した実Go Binary／ArtifactSet Memberを取得し、実コンパイル対象となったSourceの閉包とResultを検査 | 生成元Goがディスクに存在するだけ、参照GoテストのPASS、正しい名前の空Artifactを収録成功としない |
| Providerへの引渡し | 選択Contract／Lockと実Artifactに含まれるProvider実装を確認し、共通Application APIのtyped Entryへ接続する | Build-only Generatorを無条件にRuntimeへ混入しない。選択B・収録A、同値結果での不一致は失格 |

この表の各項目は既存#943／#2595のBuild入力閉包、#951／#1016のPlugin Selection／Lock、#1121／#4544のTarget Lowering、#1163のEvidenceと#1180／#2514の独立Expectedを**参照するだけの引渡し票**である。新しいBuilder API、Generator Registry、Artifact正本、万能PASS状態は提案しない。実Builderが生成byteを受け取れないと判明した場合は、その既存入口の最小AdapterをOwnerが定め、暫定のReference Runnerで合格にしない。

### 14.3 R2→R3の一件の反証と実装単位

最初のR2差分では `quantity_positive` だけを、TEI内蔵Reader→解決済み型→実Go生成→現行Builderの入力→Goコンパイルまで接続する。次のR3差分で通常 `tei` CLIの型付きOperationから**現在選択されたProvider**を実呼出しする。CLIの専用テスト入口や、同じ結果を返す手書きGoに委譲して成功とはしない。

独立Expectedに渡す負例は、**R2-01** ReaderのRule／型改版後も古いGoをBuild、**R2-02** Go byteは更新したがBuilderが旧Sourceを実読取、**R2-03** 選択GeneratorまたはToolchainだけ変更し以前のArtifactを無検証再利用、**R2-04** `internal` importを利用できないTargetへ同じGoを配布、**R3-01** 選択Provider Bと実Call Aの食い違い、**R3-02** Providerのactual call無しでCLIの固定Expectedだけ一致、の六つとする。既存シナリオに重複するものは新IDの別試験を作らず、差分の観測項目だけ足す。

Source変更A→B→A、生成器版変更、Build時だけの依存、入力取得の不完全性、実行時の現在Grant・Effect／Receipt、Studio Stage1→Stage2 Windowsは別の資格を維持する。**R2のDoneは実Builderによるbyte受領と実Artifactの成立まで、R3のDoneは通常製品入口での選択Provider実Call・独立Consumerの受領まで**と定める。R2の完了をStudioセルフビルドの代わりにしない。

本節は現在mainで確認できたR1の実装とGoの境界をもとにした計画であり、R2／R3を新たに実装・検証したという報告ではない。

### 14.4 既存参照Builderに生成Goを渡す一試行（PR #4934）

実Builder入力の照合は、BuildPlanの型を説明するだけでは検証できない。[PR #4934](https://github.com/hat47x/tei-platform/pull/4934)で、`tools/reference-go-build-from-plan/validationgen_build_binding_test.go` に、実 `Workspace.Load` → 選択Rule・Source Index → `validationgen.GeneratePositive` → 一時Go package → 既存 `DirSourceResolver`／`plugin.BuildPlan` → 既存 `reference-go-build-from-plan` の `-isolate` 実行 → Artifact Digest・BuildArtifactLock照合 → **Builderが生成した実Binaryの独立Expected**という試験を加える。

この試験では生成Goを計画確定後に書き換え、古い計画がコンパイル前に拒否されることも確かめる。Goモジュール文脈と、生成Goが参照する既存 `core/identity`、`internal/core`、`internal/semantic`、`internal/semantic/validationeval` を計画Sourceとして明示する。ReaderのSource ID／Attribute型、BuildPlanの実入力Digest、Binaryの実挙動は別々に照合し、一致結果だけを根拠にしない。

**受入範囲は既存Reference Builderでの一回の実Go消費に限る。** この試験で作ったBuildPlanはテスト用であり、正式製品 `tei` が選択したBuilder／Plugin Runtimeを証明しない。製品Build Intentから実ProviderのEntryまでの接続と、Stage1→Stage2 Windows実再入は、それぞれ既存#4544／#4495／#2831／#4583に残す。PRのマージとexact HEADのGo／CI成功も区別する。

### 14.5 参照Builderの実績と製品R2の残差

現行mainには`tools/reference-go-build-from-plan/validationgen_build_binding_test.go`が実在する（旧[PR #4934](https://github.com/hat47x/tei-platform/pull/4934)自体は未マージでクローズ）。製品Loaderで選択したRuleから生成Goを作り、既存BuildPlanと`DirSourceResolver`がGo Source／依存パッケージ／Module Contextを実Digestで照合し、`-isolate`ビルドのArtifact Digest／BuildArtifactLock／五つの実バイナリ結果を検査する。#4934の実CIでは`tei/tools/reference-go-build-from-plan`パッケージの試験が成功したが、全体Healthは他パッケージの問題により失敗している。**この結果を参照Builderでの局所R2先行証拠として記録し、正式製品R2合格には昇格しない。**

既存テストが作るBuildPlanには、元MarkdownのRule Source／選択Language・Generator版／Owner意味受理までを製品Builderの入力として束縛する一貫した契約はまだ示されていない。実装の次の最小単位は、#4544／#2595の**既存製品Build入口**で次の連鎖を再現することに絞る。`選択済みSource（現在版）→解決済み型付きMeaning→選択Generator実装と版→生成Go byte→実Builder入力byte→Build Artifact`。各要素のFingerprintや版を一つの「全部OK」に潰さず、変更・旧版再利用・Selected B／Actual Aを原因ごとに拒否する。構文上の同値とOwner受理、同一生成GoとSourceの現行性、署名・供給資格とGoコンパイル成功を区別する。製品CLIに新しい参考コマンドを足さず、Builderの重複実装も行わない。

Data Rule／Table Script／Field拡張はデータ意味の側へ、独立したFunctionとData Contextは公開Entryの側へ収める（#4937）。Node Flowは同じRule／Operator／Sourceからread-onlyに投影し、公開・実行はHostの実権限とProviderを要する別ゲートとする（#4924）。最初のR3対象はこのR2を先行条件にしない受理済みpure Function一件とし、#4495の実Providerと通常製品CLIのConsumerを並行して完成させる。

### 14.6 Ruleの再利用とFunction公開Entryを分けてから合流する

製品Go回帰`internal/operations/inspect_markdown_rule_entry_boundary_test.go`では、選択済み`tei.rule-authoring/v0`が`semantic-validation`として正しいSourceを持っていても、**同じIDの`semantic-capability`が存在するわけではない**ことを確認する。したがって`sales.order_line.quantity_positive`を生成して実行できても、それだけで呼出し可能な公開Entryは生まれない。Data側のRule／型／Relation、Function側の入力・判断・公開、Business側の状態／遷移という既存の責務を維持する。

**二つの実装レーンを独立して閉じる。** R2-P（#943／#2595／#4544／#951）は、選択済みRule・Generator／Toolchain・生成Goの実byteを既存製品Build Intentへ渡し、Runtimeへ必要なContributionがArtifactに収録されたことを実Builderで検証する。R3（#4495／#1251／#1158）は、**既に受理されたpure Function一件**について公開Entry→現在選択したProvider→実Invocation→通常`tei` CLI／独立Consumerを先行検証できる。R3をR2-Pの完成待ちにしない。両方の資格が揃ってから、Rule由来の生成実装をそのFunctionが実際に呼ぶ**結合検査**を追加する。

| 接続点 | 成立に必要な対応 | 合格にしてはいけない例 |
| --- | --- | --- |
| Rule→Function | Functionが参照するRule ID・版、業務上の用途、評価に必要なData Context／取得範囲を明示 | 同名Rule IDを公開Function／CapabilityのIDとして自動登録 |
| Function→公開Entry | 正式に公開するEntry ID、入力の名前・意味型・必須性、欠損／NULL／未取得、Target／Scopeとpure／effect宣言を対応付ける | DSL項目名とCLI／Provider引数名の不一致、未宣言別名、部分取得から勝手に既定値0を補完 |
| Entry→Provider実Call | 既存#951の選択結果（Contract・実装・Lock・Artifact）に、Hostの現在Grantと実Entry呼出しを照合 | Selected B／Actual A、Plugin Inventoryの発見だけ、旧Grant・旧Binaryからの成功推測 |
| 実結果→Consumer | 同じAttemptで実Providerの入力・実Outcome・独立Expected・受領先を対応付ける | CLIの期待値固定出力、同じ結果の手書き実装、Reference Binaryを製品Consumerとして算入 |

Ruleは他のFunction、標準Operator、Table Script／Data Rule、Node Flowからも**再利用可能な判断部品**である。ただしRuleの正本をFunctionやNodeへ複製しない。NodeのPort名・型とFunction Entryの入力名を結ぶ場合も、画面の配線やLabelだけをBindingの根拠にせず、型付きOperationの入出力と必要Contextに投影する。API公開・外部Trigger・Effectful Hookについては、明示的な公開資格・現在Grant・Effect／Receiptを後続で検収する。

結合検査の最小反証は、**RE-01** Rule正本は存在するがFunction Entryが未公開、**RE-02** Entryの引数名・型がFunction／Data Contextと不一致、**RE-03** NULLと未取得の扱いがProviderで潰れる、**RE-04** R2-Pが生成実装Bを収録したのにR3が旧実装Aを呼ぶ、**RE-05** Rule変更後に旧Artifactと旧Consumer結果を流用する、である。これらは新しいScenario／権限正本を作らず、既存#1180／#2514へ負例の追加として渡す。

最初のR3でpure Functionの実呼出しを閉じる間も、**Studio Stage1実TEI→Stage2実Windows WebView2**（#2831／#4583）は独立した最上位P0とする。本節は製品で実在するRule・Reference Builder・Entry拒否テストを踏まえた設計・引渡しであり、正式製品BuilderとRule由来Functionの統合実行を新たに受理したという報告ではない。
