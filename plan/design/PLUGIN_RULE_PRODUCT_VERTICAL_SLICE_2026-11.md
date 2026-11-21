# Rule Authoringから実Plugin・製品Consumerまでの第一縦断：実装と検収の接続計画

- **状態**：2026年11月15日の設計・実装引渡し。R1（正値Rule→実判断Go生成）は[PR #4911](https://github.com/hat47x/tei-platform/pull/4911)でmain統合済み。R2の既存参照Builder回帰は製品の正式Builder選択とは別資格。実Plugin呼出し、通常CLIの業務Entry、Windows Stage2は未受理。
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

**生成Goの配布先は別に検収する（EX-01〜04、未実行）。** 現行の`GeneratePositive`が出力するGoは`tei/internal/semantic/validationeval`をimportしており、R1の試験もTEIリポジトリ内で実行する。同じGoファイルを別モジュールの一般的なGoアプリへ移しても、Goの`internal`参照制限によりそのままでは利用できない。従って、R1／R2の製品内ビルド成功を「TEIなしでも通常保守できる」証拠にはしない。既存[#4191](https://github.com/hat47x/tei-platform/issues/4191)のEX-01〜04では、一つのRuleについてTEIを含まない独立したGoプロジェクトでの通常ビルド、生成器を呼ばない一変更、Markdownと独立Expectedの追跡を確認する。配布方法は、公開可能な契約を最小限のAdapterとして渡す方式と、生成先の型・関数だけで完結させる方式を比較し、必要な依存関係を明示して決める。TEI専用Runtimeや新しいHostを前提にしない。着手は[現在の優先順位](../NOW.md)の初回利用者検収と製品Studio保存→Stage2の後とし、この節だけで新しい配布方式を正式採用しない。

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

実Builder入力の照合は、BuildPlanの型を説明するだけでは検証できない。PR #4934はclosed／unmergedだが、現在のmainでは統合コミット [`c2cf3fe8`](https://github.com/hat47x/tei-platform/commit/c2cf3fe883de7653ef1e0fdb3cf43bd42dcecbcc)を経て `tools/reference-go-build-from-plan/validationgen_build_binding_test.go` が収録されている。この試験には、実 `Workspace.Load` → 選択Rule・Source Index → `validationgen.GeneratePositive` → 一時Go package → 既存 `DirSourceResolver`／`plugin.BuildPlan` → 既存 `reference-go-build-from-plan` の `-isolate` 実行 → Artifact Digest・BuildArtifactLock照合 → **Builderが生成した実Binaryの独立Expected**という手順が実装されている。**mainのファイル収録と、特定HEADのCI成功・Owner受理は別の事実である。**

この試験では生成Goを計画確定後に書き換え、古い計画がコンパイル前に拒否されることも確かめる。Goモジュール文脈と、生成Goが参照する既存 `core/identity`、`internal/core`、`internal/semantic`、`internal/semantic/validationeval` を計画Sourceとして明示する。ReaderのSource ID／Attribute型、BuildPlanの実入力Digest、Binaryの実挙動は別々に照合し、一致結果だけを根拠にしない。

**受入範囲は既存Reference Builderでの一回の実Go消費に限る。** この試験で作ったBuildPlanはテスト用であり、正式製品 `tei` が選択したBuilder／Plugin Runtimeを証明しない。製品Build Intentから実ProviderのEntryまでの接続と、Stage1→Stage2 Windows実再入は、それぞれ既存#4544／#4495／#2831／#4583に残す。PRのマージとexact HEADのGo／CI成功も区別する。

### 14.5 参照Builderの実績と製品R2の残差

現行mainには`tools/reference-go-build-from-plan/validationgen_build_binding_test.go`が実在する（旧[PR #4934](https://github.com/hat47x/tei-platform/pull/4934)自体は未マージでクローズ）。製品Loaderで選択したRuleから生成Goを作り、既存BuildPlanと`DirSourceResolver`がGo Source／依存パッケージ／Module Contextを実Digestで照合し、`-isolate`ビルドのArtifact Digest／BuildArtifactLock／五つの実バイナリ結果を検査する。#4934の実CIでは`tei/tools/reference-go-build-from-plan`パッケージの試験が成功したが、全体Healthは他パッケージの問題により失敗している。**この結果を参照Builderでの局所R2先行証拠として記録し、正式製品R2合格には昇格しない。**

既存テストが作るBuildPlanには、元MarkdownのRule Source／選択Language・Generator版／Owner意味受理までを製品Builderの入力として束縛する一貫した契約はまだ示されていない。実装の次の最小単位は、#4544／#2595の**既存製品Build入口**で次の連鎖を再現することに絞る。`選択済みSource（現在版）→解決済み型付きMeaning→選択Generator実装と版→生成Go byte→実Builder入力byte→Build Artifact`。各要素のFingerprintや版を一つの「全部OK」に潰さず、変更・旧版再利用・Selected B／Actual Aを原因ごとに拒否する。構文上の同値とOwner受理、同一生成GoとSourceの現行性、署名・供給資格とGoコンパイル成功を区別する。製品CLIに新しい参考コマンドを足さず、Builderの重複実装も行わない。

Data Rule／Table Script／Field拡張はデータ意味の側へ、独立したFunctionとData Contextは公開Entryの側へ収める（#4937）。Node Flowは同じRule／Operator／Sourceからread-onlyに投影し、公開・実行はHostの実権限とProviderを要する別ゲートとする（#4924）。最初のR3対象はこのR2を先行条件にしない受理済みpure Function一件とし、#4495の実Providerと通常製品CLIのConsumerを並行して完成させる。

### 14.6 製品R2の最小一件と、意味変更・説明文変更の扱い

現行`reference-go-build-from-plan`の`go-command`でBuildPlanに載せられる実消費Sourceは、`go-package`と`go-module`である。`checkSourcesAreRead`は実際に読まないkindを拒否し、`checkDependencyCoverage`はGoの依存パッケージを検査する。参照BuilderにRule Markdownを読ませるため、`*.tei.md`を`go-package`と偽装したり、ビルドが読まないSourceを計画へ足したりする方法は採らない。

次の製品R2は、現行のpureな`GeneratePositive`やGo Builderへ権限・Source解釈を重複実装するのではなく、**既存の選択・生成・Build境界を呼ぶ製品側の一連の処理**として実装する。まず`Workspace.Load`で確定した管理Source、Language版、Rule／型、Source IndexとWorld／Model Revisionを取得する。次に同じ選択版からGoを生成し、**生成Attemptの入力指紋・Generator実装と版・出力Go digest**を実Builderへ引き渡す。Builderは従来どおり実Go入力とToolchain／依存を独立に検査し、結果のArtifact digest・Build lock・Consumerを生成Attemptへ結び付ける。必要な受渡し形式は既存#943／#2595／#4544の契約上で決め、新しい汎用Source kindや第二BuildPlan、Evidence正本を新設しない。

| 反証となる変更 | Generator側で確かめること | Build／再利用側で確かめること |
| --- | --- | --- |
| **Rule条件・参照先を変更** | 新しい`Workspace.Load`と型解決、別の意味・Go digest、旧Source Indexの失効 | 古い生成Attempt／Build資格を拒否し、新Go byteを実Builderが読む。別の実Consumer結果を独立Expectedと比較 |
| **Markdown説明文・レイアウトだけ変更** | 管理Sourceの原byte指紋は変わるが、選択Ruleの意味・生成Goは同じかを再評価 | **生成Go／Artifactの再利用可否**と、**旧Sourceに基づくEvidence／Owner受理・現在Grantの再利用可否**を分けて判定。byte相違だけで全Targetを再生成すると決めず、同一Meaningだけで証拠も自動復活させない |
| **Generator／Toolchain／依存版だけ変更** | Ruleが同じでも、実装・Target・版の組を再検証 | 旧Plan・Artifactの自動受理を禁止。入力Goが同じか、実BuildでどのToolchainを読んだかを別票にする |
| **Source A→B→A、選択B／実行A** | 同byte・同IDでも前回のAuthorityを継承せず、選択版とBuild対象を明示 | 旧ReceiptやGrantを復活させず、実ArtifactとProvider実EntryのIdentityを別照合 |

**最初に通す二つの負例**は、(1) Rule原文を意味が変わるように変更したのに旧Go／旧BuildPlanを使おうとする試行、(2) 説明文のみの変更後に、生成Goの再利用可否とSource-backed Evidenceの再確認を混同する試行とする。Go BuilderへMarkdownを読ませる暫定対応ではなく、実生成AttemptからBuilderの実入力へ至る証拠の橋渡しを#4544／#2595で実装する。R3の通常`tei` CLIからの選択Provider actual Call、R4の独立Consumer、Studio Stage1→Stage2 Windowsの実再入は並行P0のまま残す。

**現状の判定**：R1.5のテストはmain収録済み、参照Builderの一連の検査コードもmainに存在する。製品R2で必要なRule→Go生成Attempt→実Builder入力の連鎖、R3の実Provider、Stage2 Windows、Ownerの業務意味受理は未検収である。この節は実装と要求の境界を定めるものであり、Go／CLI／Windows／DB試験を新規に実行した記録ではない。

### 14.7 Ruleと公開Entryの境界を明確にしてから合流する

mainの製品Go回帰`internal/operations/inspect_markdown_rule_entry_boundary_test.go`では、選択済み`tei.rule-authoring/v0`が`semantic-validation`として正しいSourceを持っていても、**同じIDの`semantic-capability`が存在するわけではない**ことを確認する。したがって`sales.order_line.quantity_positive`を生成して実行できても、それだけで呼出し可能な公開Entryは生まれない。Data側のRule／型／Relation、Function側の入力・判断・公開、Business側の状態／遷移という既存の責務を維持する。

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

**実装プロファイルの誤用を防ぐ。** 現行 `internal/plugin/implementation/model.go` の `static-response-v1` はDeployment Executorの固定応答向けであり、Rule／Functionの汎用実装Contractではない。ここへ期待する判定JSONを置いても、生成Goを選択Providerが実行した証拠にはならない。`composition.Selection` は宣言上の選択、`composition.Service.Invoke` が返すBindingは実装を追跡する観測であり、どちらもHostの現在Grantではない。R3の最小実装は、既存の共通Application APIと正式なProvider選択の接続を先に確認する。

**製品CLIの現在の基線。** `cmd/tei/main.go` の未宣言 `dbIntrospectSemanticFlag` で実CLIがビルド不能だった問題は[PR #4940](https://github.com/hat47x/tei-platform/pull/4940)で修復してmainへ統合した。同PRのexact HEADではAction Surfaceの実CLI、独立Node Catalog、Source→Web E1-A、製品Skill、Cache、脆弱性確認が成功。一方、Studio／Issue-to-Design／Self-buildの失敗は残り、全体Healthの成功も確認していない。これらのCI結果を、正式R2のBuilder実選択、R3の公開Entry実Provider Call、Studio Stage2の受入と混同しない。

### 14.8 異なる宣言表現から同じGoを生成し、出典の資格は別に残す

[実Go回帰の提案 PR #4973](https://github.com/hat47x/tei-platform/pull/4973)では、同じ `sales.order_line.quantity_positive` を、明示選択した `tei.rule-authoring/v0` のMarkdownと従来の `validation` 構文から、それぞれ独立して `Workspace.Load` → 型解決 → `GeneratePositive` に通す。Canonicalを片方からコピーせず、同一の `semantic.Validation` と解決済み型 `std.decimal` であることを確認したうえで、生成Goのbyte完全一致を判定する。**これは作成中の製品回帰であり、exact HEADのCI合格はまだ未確認**である。

ただし、生成Goが等しいからといって二つの原Sourceの資格を統合しない。選択されたReader、Source Indexの定義位置、ファイル原byte指紋、Model Revisionはそれぞれ保持する。Markdownの説明文だけを変更した場合、意味と生成Goは不変であっても古いSource Indexの現行性は失効する。旧Source Evidenceの再確認と、生成Artifactの再利用の可否は分けて判断する。逆に、Rule条件・参照先を変えた時は意味と生成Goの両方を再検証する。

この性質をTEIドッグフーディングの検証原則として採用する。**実装から抽出した型・判断・制約が、DSL／Markdown／図表の異なる表現を経ても保存され、再生成によって確認できること**を要求する。Graphでは画面上のNode／Edge IDと業務Relationの識別子を混同せず、Canvasの配置変更だけで業務Goの意味を変えない。複数の意味等価な表現を増やす際にも、新たな意味正本・Reader／Generator／Builder・Plugin Registryを増やさず、標準データ構造と既存Operator契約を再利用する。

この回帰が成功しても、正式製品Builderによる実byte消費（R2-P）、公開Entryを介した実Provider／通常製品CLIの呼出し（R3）、Windows Studioの実ConsumerとStage1→Stage2は別の検収である。原文A→B→A、選択B／実Call A、Ownerの業務意味受理と現在Grantは、既存の第14.6〜14.7節および#4544／#4495へ引き続き委ねる。

### 14.9 従来構文と選択済みRule Markdownを混ぜずに診断する

R0の入口にも、実Providerと同じ「選択された実装だけが入力を解釈できる」という境界を適用する。従来の`.tei`意味ファイルでは`validation`が既存の規則であり、`rule`は自動的に別名扱いしない。Rule Authoringは、Workspace Manifestが`tei.rule-authoring/v0`を明示選択した`*.tei.md`だけを受け持つ。未選択の`*.tei.md`、通常のMarkdown、管理対象外のSource、従来`.tei`の`rule`を同じものとして読み替えない。

このため、従来`.tei`の`rule`を検出した場合は単に「不明な宣言」とするのではなく、**定義したファイルと対象語の原byte範囲を示したうえで、Rule Authoringの選択条件と置き場所を案内する**。エラーの改善は、Readerの自動選択・意味の受理・旧構文の暗黙変換ではない。未知語は引き続き未知語として拒否し、構文の未受理を半分だけ読み込んだモデルに変えない。

負例は既存`internal/syntax/rule_placement_test.go`と`internal/workspace/rule_placement_source_diagnostic_test.go`で扱い、選択・未選択・従来構文の回帰は既存`tei_md_source_guard_test.go`と製品Readerのテストを流用する。CIの監視対象も**現在存在する`rule_authoring_markdown_loader.go`／`parser.go`／実テスト**と一致させ、廃止した旧guardファイルへの参照で失敗させない。これによってR0の診断品質を改善しても、R2-Pの実Builder選抜、R3の正式Provider実Call、Stage1→Stage2 Windowsの実証を合格扱いしない。

### 14.10 生成Ruleを通常CLIのBuild Intent経路で実ビルドする

参照Builderを直接呼ぶ§14.4の回帰に加えて、`cmd/tei/cli_self_build_generated_rule_go_test.go`では、選択済みRuleの正値判断Goを一時的なGo commandとして置き、**通常の`tei self-build`のBuild Intent解決と実Builder起動**を一つの試行に通す。CLIとBuilderは新設せず、既存`writeFullChainDeclaration`、`authorGoCommandCatalog`、`buildintent.Resolve`、`reference-go-build-from-plan`を利用する。

入力は実Fixtureの`tei.rule-authoring/v0`が読み込んだ`quantity_positive`と、解決した属性型`std.decimal`に限る。`GeneratePositive`の実byteをGo packageに保存し、そのpackageと実依存closure、`go.mod`、`go.sum`をBuild IntentのSourcesに列挙する。Go package名をアルファベット順にした場合に`definitionSource: pkg0`がEntryを表すとは限らないため、既存テスト用宣言生成ヘルパーでは**実EntrypointのSource IDを選択する**。これは実BuildPlanを正しく導出するためのテスト修正であって、第二Resolverの追加ではない。

CLI側の独立観測は、実Entry内でのIntent解決、BuilderのPlanとの照合、Artifact Binding、同じPlanからの再現性、隔離Go Source閉包と実Binaryの生成を別々に要求する。最後に**CLIが出力したBinaryそのもの**を起動し、正値・ゼロ・負値・NULL・未取得を一回ずつ読み取る結果に照合する。`go run`だけの検査や、子Builderの成功自己申告だけでは合格にしない。

この回帰で証明できるのは「製品CLIの通常`self-build`経路が、選択されたRule由来のGo bytesをコンパイルし、そのBinaryが判断する」という限定的なR2-Pの一部である。**Rule原文、Reader版、Generator実装版と生成Goを同一Build Intentの受理済みSourceとして束縛したわけではない**。Authoring→Generatorの出典失効、通常製品が受理したBuilderのAuthority、正式公開Function、選択Provider実Invocation、現在GrantとOwner意味受理、Windows Stage1→Stage2は引き続き独立資格として残る。試験はLinux/amd64の既存参照Catalogで行い、ほかのTargetの適合を主張しない。

次の実装差分は、新しいGeneratorやBuilderを導入することではなく、**同一Attemptで選択したRule／Generatorの資格と、通常CLIが実際に消費したBuildPlan／Artifactの資格を結合すること**に絞る。その後、§14.7の正式Function公開Entryと実Provider Callに結合し、選択B／収録A、生成Go改版／旧Entry実行、同値結果だけの誤受理をそれぞれ拒否する。

### 14.11 Rule生成Goの実CLI検査を、個別の検収対象として維持する

PR #5009で追加した`TestSelectedRuleGoThroughRealCLISelfBuild`は、既存の`tei self-build`を経由して生成Goの実Binaryを構築・実行する回帰である。PR #5009はmainへ統合済みだが、従来のSelf-build全鎖CIが成功しただけでは、この新規テストを実行したことにはならない。そこで既存のSelf-build実証ワークフローへ**このテスト名を指定した実行ステップ**を設け、結果を独立して確認する。

Build Intentの`definitionSource`はGo依存パッケージの先頭ではなく、実際のEntrypointを含むPackageへ対応付ける。結果確認では、CLIから導出したPlan、Builderが読んだPlan、再現されたArtifact、完成Binaryの正値・ゼロ・負値・NULL・未取得の判断を分ける。これは**既存の参照Builderと製品CLIの組合せ**による限定的な実証であり、生成器・Rule原文のSource資格、正式なBusiness Function Entry、実行時の選択Provider／Grant、Windows Stage2を自動受理しない。

また、並行マージ時に本計画の日本語が文字化けした事例を再発防止対象とする。正常な§14.10までの内容をGit履歴の元byte列から復旧し、Repository Plan Checkで不正なUTF-8、置換文字、Markdownに不適切な制御文字を拒否する。これは文書の自然さすべてを機械判定するものではなく、**原文を破壊するエンコーディング事故を確実に検出する下限**である。読める日本語の保持と、Source／Meaning／Build／Invocationの受入資格の維持を別々に検査する。

### 14.12 生成元と実ビルド入力を別の閉包として照合する

[dogfooding設計§13](IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md)では、現行`BuildIntent.Sources`、`DirSourceResolver`、`go-command`の`definitionSource`を実コードと照合し、R2-Pの次の実装を**生成元の閉包と実コンパイル入力の閉包の接合**に絞った。

現行の`definitionSource`はGo Entrypoint packageを示す。元Rule MarkdownやGeneratorをこの欄へ入れ替えると、Buildの定義と生成元の役割が壊れる。また、Resolverがファイル／ディレクトリの実byteをFingerprintへ取り込めることだけでは、その原文が選択済み`tei.rule-authoring/v0`のReaderで実際に解釈されたこと、GeneratorがそのRuleを使ったこと、Builderが生成されたGo byteを消費したことは証明できない。

既存`TestSelectedRuleGoThroughRealCLISelfBuild`を基線に、同じ試行の`Workspace.Load`／Source Index・型付きRule／Generator・生成Go／実Entrypoint Source・Plan／Builder実入力・Artifactを**別々の由来を保ったまま照合**する。第一差分はテスト内のread-only対応表と一因子の負例で十分であり、新Source Kind・Resolver・Receipt schemaの正式採用はOwnerの検証後に判断する。

最初の負例は**生成Goとして選択したBと試験packageに置いたAが異なる場合**とし、既存のBuilder／Artifact照合に届く前に差異を検出する。次に原文説明のみの変更ではSource Evidenceだけが失効すること、Generator版変更で出力byteが不変でも生成資格は別に受け直すことを検査する。実際のBuilderが読んだbyteを観測できなければ、形式上のDigest一致から正式R2-P合格を主張しない。R3実Provider・Windows Stage2・Owner意味受理は引き続き別列とする。

### 14.13 原文byte列もBuildPlanの入力へ含め、実Binaryから読取りを確認する

既存のRule生成Go→通常`tei self-build`回帰では、生成GoをBuildPlanで収録して判断できる一方、選択元Markdownの原文byte列がそのBuildPlanの入力に入っていなかった。そこで既存`TestSelectedRuleGoThroughRealCLISelfBuild`を拡張し、同一`Workspace.Load`で得たSource Indexの位置・選択Readerと、現行原文byte列を照合してから、生成Goの隣に原文の**byte同一コピー**を置く。既存のGo `//go:embed`を利用して原文を最終Binaryに収録するため、第二のReaderやBuilderは不要である。

Build Intentは、その原文を含むGo Packageを実Source閉包として列挙し、参照Builderは選択したSourceの現行digestとの一致を確認して隔離ビルドする。テストはBuilderの成功報告だけで合格とせず、`tei self-build`から得た**実Binary自身**が埋め込んだ原文のSHA-256を計算した結果を、テスト側で選択元ファイルから独立に求めた値と照合する。従来の正値・ゼロ・負値・NULL・未取得の期待結果もそのまま確認する。Source Indexが失効した状態で古い意味モデルに新しい原文を結び付けることは拒否する。

これは「選択されたRule原文のコピーと生成Goが、一つのGo Package Sourceとして実Builderに読まれ、完成Binaryに達した」という限定的な**R2-P原文消費の証拠**である。元ファイルのWorkspace相対LocatorやReader・Generatorの実装byte列まで同一Build Intentに正規のDependencyとして束縛したわけではなく、コピーの正当性は検査コード側の観測に依存する。Generator改版、原文A→B→AでのRevision失効、生成Goと原文の意味適合、正式Function公開Entry→選択Provider実Call、Owner受理、現在Grant、Windows Stage1→Stage2の証明は引き続き別の検収とする。

次の段階では、この実績を根拠に新たなBuildPlanやRegistryを作るのではなく、既存のBuild Intent Source／Build Receiptに、Reader・Generator・原文の資格と実消費結果を同一Attemptで紐付ける。その際、元Sourceが編集されても生成Goが偶然同値なら旧Source Evidenceは失効し、Build Artifact自体の再利用可否は独立判断とする。

### 14.14 選択した生成Goと実ビルド配置先の取り違えを先に拒否する

§14.12の最初の負例を、既存の通常CLIセルフビルド試験へ追加する。選択済みRuleから現行Generatorが返したGoのbyte列を、実際にBuilderへ渡すGo Package内の`generated.go`と読み戻して照合する。**選択した生成結果Bに対し、配置先だけをコンパイル可能な別byte列Aへ変えた場合は、Build Intentの解決やBuilderの起動に進む前に拒否する**。元のbyte列に復元した場合だけ後続の実CLI／実Binary検査へ進める。

この検査は、新たな意味受理機構ではなく、同じ試行におけるGenerator出力とステージングされた実入力の取り違えを防ぐ最小の境界である。変更後のGoが同じ判断結果を返しても、byte不一致を見逃さない。反対に、byte一致だけを根拠としてReader／GeneratorのAuthority、元RuleのRevision、正式Providerの選択、現在Grant、Owner受理を昇格させない。

次の負例は、**元Rule原文をA→B→Aと戻したときのSource Revision失効**と、**Generator実装版を変えても生成Goが同一となる場合の生成資格失効**である。既存のSource Index、Build Intent、Receiptの各責務を維持し、再ビルドの必要性と意味受理の必要性を別々に記録する。

### 14.15 変更後の再利用判断を、出典・成果物・実行資格に分ける

Ruleの原文と生成Goを同じBuild入力へ収録できても、**入力の同一性、生成物の再利用、業務上の受理、実行許可は別の判断**である。§14.14の配置先照合を足場に、変更の種類ごとに失効する資格と、再利用を検討できる対象を分ける。

| 変更・事象 | Source／生成資格 | Build Artifact | 実行・意味受理 |
| --- | --- | --- | --- |
| 原文の説明文だけを変更 | 旧Source Evidenceは失効。新しい原文位置・Revisionを再確認 | 生成Goと実Build閉包が同一なら再利用を検討。ただし原文を同梱する現行実証ではBuild閉包が変わる | 説明が業務意味を変えないかOwnerが判断。旧Grantを自動復活させない |
| 原文をA→B→Aと編集 | 最終byteがAでも旧Revisionの資格は復活しない | 最終Build入力が同じならArtifact再利用を別途判断 | 過去の受理を変更履歴の代わりに用いない |
| Generator版を変更し、出力Goは同一 | 旧Generator資格は失効。新しい版の選択と実消費を確認 | Go閉包が同じなら再ビルド不要となる余地がある | 新Generatorによる意味適合は独立に受理する |
| Readerまたは選択Pluginを変更 | 選択したReader／Plugin Lockを再解決。旧解釈資格は失効 | byte一致だけで新しい意味解釈の正当性を推定しない | 選択Provider・Authority・Grantを再確認 |
| 生成Goと配置先Goが不一致 | 同一Attemptの対応付けに失敗 | Builder起動前に拒否し、旧Artifactへ無言で切り替えない | 実行不可。Owner受理へ進めない |
| Operator選択をA→B→Aと変更 | 現行選択と履歴を照合。旧選択資格は復活しない | Artifactが同じでも実行対象の選択は別 | 旧CLI／旧Grantの無条件再利用を拒否 |

**検収の依存関係**は、(1) 現行SourceとReader／Generator／Plugin Lockを確定し、(2) 選択した実現方式の入力と、必要ならBuild Intent／Artifactを照合し、(3) Ownerによる当該業務意味の受理と現在の認可・Grantを**実行前**に確認し、(4) 正式Entryから選択Providerが実際に呼ばれたことを、Result／Effect／Receiptで検収する。受理とGrantは相互に代替できず、順序を固定した一つの承認ステップにもまとめない。未受理・未認可の対象に副作用を発生させてからOwner受理を求めてはならない。途中で失敗した場合は、後段の成功履歴で埋め合わせない。

この表は**実装済み機能の一覧ではなく、次の負例と受入票の判定基準**である。現行のSource IndexとBuild Intentだけで編集履歴A→B→AやGenerator版の全資格を証明できるとは主張しない。まず既存のRevision／Receipt／Plugin Lockで保持できる観測値を調べ、不足が確認された項目だけを既存契約へ追加する。新たな万能Registryや別系統のBuild機構は設けない。

実装順は、原文A→B→Aの失効を示す独立負例、Generator版変更でGo byteが同じ場合の資格分離、正式Entry→Provider実Callの順とする。直接解釈するOperatorについても、生成Goの代わりに**選択したモデルと実際に解釈されたモデル**を照合し、生成方式と直接解釈方式を同じ受入票で比較する。Windows Studio Stage2、R3、Owner受理はそれぞれ独立した検収のまま維持する。

### 14.16 実現方式の選択と、実際に呼ばれた方式を同じ受入票で検証する

図表の実現方式として、生成コード、既存の型付きEvaluatorによる直接解釈、明示的なDelegate、これらの組合せを認める。ただし、**採用候補が複数あること**と、**特定Attemptで選択・呼出しを受理したこと**は異なる。方式を示さず結果が同じだからよい、という判定は行わない。これは [実装からモデルへのdogfooding §15](IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md) および [Plugin接続検収 §11](PLUGIN_CONNECTION_EVIDENCE_AND_COMPOSITION_SLICE_2026-11.md) と共通の境界である。

| 受入票の欄 | コード生成 | 直接解釈・Delegate |
| --- | --- | --- |
| 選択 | Rule原文・Revision、Reader、Generator版、Target、選択Operator | Rule原文・Revision、Reader、Evaluator／Delegateの識別・版、選択Operator |
| 実際の入力 | 生成Goのbyte列と配置先を照合し、Build Intent／Builder実入力へ接続 | 選択モデルのbyte列・型・Revisionと、実際にEvaluator／Delegateへ渡した入力を照合 |
| 実現結果 | Build Receipt／Artifact／正式Entryと実Consumerの識別 | 実Evaluator／Delegateの呼出し記録と、実Consumerの識別 |
| 独立Expected | 正値・ゼロ・負値・NULL・未取得、Read先と回数、失敗種別 | 同じExpectedを別経路で検証。結果一致だけでは方式選択の証明としない |
| 実行前の資格 | Ownerが受理した意味・現在のProvider選択・Grant／Effect条件 | 同左。直接解釈だから認可を省略しない |
| 実行後の証拠 | 選択した生成版が呼ばれたResult／Effect／Receipt | 選択したEvaluator／Delegateが呼ばれたResult／Effect／Receipt |

最初の負例は **「解釈方式を選択したのに、生成Goへ無言でフォールバックする」**、その逆、および **「結果は同じでも別Providerが呼ばれる」** の三つとする。選択対象が未取得、失効、未対応なら、黙って別方式に切り替えず、診断に選択と実Callの差を残す。UIやNode Viewの接続線、Pluginの型適合、Targetへの配置だけでは実Callの証明にならない。

複合型の全体連結、項目単位の連結、Adapter経由の変換は、標準の型付きMappingへ委ねる。各段で型、NULL／欠損、多重度、変換後の意味とEffectを保持し、**型が適合することと権限があることを別々に判定する**。CLI、MCP、Wails Desktopの入口は共通Operationへ接続するが、WailsをSemantic CoreやAPI Pluginの必須Hostにしない。

**実装状況の境界**：選択済みRuleの生成Go→実Builder→Binaryと、同一Ruleの汎用Evaluator比較の限定的な証拠は既にある。一方、正式R3 Entry→選択Providerの実Call、両方式の実行時選択を拒否する負例、独立した二Consumerでの実利用、Windows Studio Stage2は未検収である。新しい統一Runner、万能Interpreter、第二の意味正本は導入しない。まず既存のEvaluator／Generator／Scenarioの試験へ負例を追加する。

### 14.17 BuildPlan解決後の入力変更を正式CLIで拒否する

§14.14の`requireSelectedGeneratedGo`は、生成結果をGo packageへ配置した**その時点**の一致を検査する。一方、`buildintent.Resolve`が実byteを取得してPlanを決めた後、BuilderがSourceを読むまでに内容が変わった場合、配置時点の照合だけでは検出できない。そこで既存の`TestSelectedRuleGoThroughRealCLISelfBuild`を一つの反例で拡張し、**選択したPlanに含まれる生成Goと実Builder入力が時間差で食い違う場合**を確認する。

試験は、同じ`Workspace.Load`と`GeneratePositive`で選んだRuleから`generated.go`を作成し、実`Build Intent`を解決した後にだけ、Goの判断結果を変えないコメントbyteを一行追記する。**古い`plan.json`を渡した通常`tei self-build`の実CLI試行が失敗し、合格のL2-bと公開可能なArtifactを残さないこと**を要求する。その後に選択された元byteへ戻し、同じCLI→Builder→実Binaryの正・ゼロ・負・NULL・未取得の正例を維持する。

| 時点 | 正例・負例の条件 | 受入の上限 |
| --- | --- | --- |
| 選択・生成 | Rule ID／型／Readerと生成Goが一致 | 出典の現在性と業務意味受理は別 |
| Plan解決 | BuildIntentから生成Goを含むGo packageを選ぶ | `definitionSource`はGo Entrypointのまま |
| **Plan後・Builder前** | Go byteだけ変え、元Planをそのまま使う | **CLI/Builderによる実入力不一致の拒否**。配置時のGuardだけでは代替不可 |
| 復元・実Build | 元byteを再配置し、実CLIからBinaryを受領 | 当該Go packageに限ったR2-Pの追加検査 |

これは新たなBuild/Source Schemaや常駐監視器ではなく、**既存Plan・Builder・Artifact照合の時間的な停止線**を一つ実測する回帰である。Generator実装版や元MarkdownのA→B→A改訂による資格失効、実Provider／現在Grant、Windows Studio Stage1→Stage2はこの試験だけでは成立しない。Go test単体の結果は変更HEADで確認し、リポジトリ全体のHealthとは分けて記録する。
