# TEI Plugin：契約から実使用までの横断受入・責務引渡し（2026-11-14）

> 状態：設計統合／実装受入案。新しい正式ABI・Schema・Gate・Owner判断ではない。今回の作業は文書のみであり、新しいGo／Windows／実DB試験は実施していない。

## 1. 目的と適用上限

TEIのEverything is Pluginを「全機能を動的ロードする」ことと解釈しない。TEIにとって必要なのは、契約が意味を保ったまま異なる実装・Target・Hostに渡り、選択したものが実際に生成・呼出し・効果確認されたかを追跡できること。標準配布はGo単一バイナリであり、業務アプリは外部Hostへ配置する。TEI専用の常駐Host、第二Resolver、第二Authority、第二Scenario／Expected正本は作らない。

本書は以下の既存Ownerを横断する「何を同一視しないか」と引渡し条件だけをまとめる：
- Contract・Provider選択／適合性：#951、#1016、ADR-0032。
- 型付きApplication Operationと実入口：ADR-0048、#1158、#1251、#4611、#4612。
- Language／自己記述／原Source：#988、#1018、#2620、#2621、#2497。
- Build・Self-build・生成Desktop：#943、#2831、#4583、#1824、#2600、#3267。
- Hook／Data Lifecycle／作用：#4198、#1129、#1174、#4502。
- 独立Expected：#1180、#2514。Studio管理：#2044。外部Host・分散配布：#2390、#2589。

## 2. 採用方向の不変条件

1. **意味と実現を分離する**：Business/Semantic ID、Service Contract、Provider implementation、Plugin package、Artifact、Target/Hostは別identityである。単一Packageの複数Role提供や、一契約の複数実現を許す。Pluginの種類をCore enumへ都度追加しない。
2. **Scopeを先に解決する**：Semantic Acceptance、Authoring、Build、Application Runtime、Target-local Toolingを混ぜない。AuthoringのSource Adapterがインストール済みでも、RuntimeのDB更新権限や意味受理を得ない。実TargetでAuthentication Providerが欠けてもローカルStudioのno-authへfallbackしない。
3. **選択と実呼出しを別検証する**：既存Compositionがexact Contract/Providerを選ぶ。Hostが実Build在庫からEntryを束縛し、呼出し直前のBinding/Generation/Scope/Host guardを再検査する。Bを選択してAへ入ったら、結果が一致しても拒否する。Providerの自己申告だけで実呼出しを認定しない。
4. **Build資格とRuntime資格を分ける**：Build時GeneratorをStage2 Desktopへ自動同梱しない。逆に選択済みRuntime Providerは、実Build/Input・Link・Consumer使用まで確認する。Lockにない、あるいは候補Catalogだけにある、という情報だけで物理非同梱を証明しない。
5. **入力とSource provenanceを保持する**：Markdown／Graphviz／SQL／関数チェーン等の糖衣構文は元形式を保持し、構造候補、受理済み意味、Source occurrenceを区別する。Source Indexの空参照、同名Subject、描画Edge IDから存在・Identity・Acceptanceを推論しない。未知／partial／permission-deniedは0件確定ではない。
6. **宣言・観測・強制を分ける**：Pluginのrequires/uses/effects宣言、実行されたcall/Effectの観測、Hostが強制したAuthority/Resource制限は別根拠。署名・Digest・symbol・プロトコル成功は単独でConformance、Loaded Bytes、業務受理、実Grant、Commit成功を証明しない。
7. **実作用は再送・復旧契約を経由する**：Hook呼出し回数、Guard前後、トランザクション境界、失敗後のEffect Unknown、receipt lookupを保持する。ack喪失や部分的な適用を「失敗なので安全に再実行」と扱わない。Buri等のワークフロー専用Engineや第二Script Runnerを新設しない。
8. **環境ごとの軽量化**：静的Go、外部Process、Wasm/Java/.NET等のTarget Adapterは同じ意味契約から選択し、共通のABI／Loaderを無条件に必須化しない。process分離はsandbox強制の証拠ではない。MCPはTool公開面であり、汎用Plugin ABI／業務Authorityではない。

## 3. Roleは利用文脈であり、新たな固定PluginKindではない

| 利用文脈 | 代表例 | 主な責務 | 誤って与えない権限 |
| --- | --- | --- | --- |
| Authoring | Language Parser、X-TEA Source Reader、Inspector、Three-view、SHO投影 | Source候補・読取・診断・出典位置 | Canonical Acceptance、Runtime Effect |
| Build | Operator Transform、Target Generator、esbuild/Wails Builder Adapter | 実生成入力・Generator資格・生成Artifact | Runtime Grant、本番配備 |
| Application Runtime | Data Provider、Rule、Operation実装、外部Service Adapter | 型付きRequest/Result/Failure、効果の実行 | 未宣言Effect、別Subsystem更新 |
| Presentation | Studio/CLI/Web/VUI/SHO／SUI Adapter | 同じOperationの表示・操作・Consumer受領 | 独自Operation正本、実行権限の自動付与 |
| Observation | Evidence／Test／Trace Provider | 実観測・Coverage・不明の記録 | 成果を根拠なしにOwner受理へ昇格 |

同じPluginが複数のRoleを担ってもよいが、各Consumerに必要なBuild closureとResource Grantのみを与える。Bundleでまとめても不要実装の強制import、init登録、生成物への自動同梱は行わない。Plugin Manager／公開Registryは初回の実呼出し成立後に扱う。

## 4. 実使用の根拠は単線ではなく複数軸の閉包

**契約軸**：受理されたContract ID/version/shape/fingerprint → Publisher/Provider Conformance → exact Selection/Binding。
**Build軸**：実際に読んだSource bytes/closure → 選択されたGenerator/Builder/Toolchain → Artifact member → 実Link/Embed/配信。
**Invocation軸**：Application Operation → Hostに束縛された選択Provider → observed entry/actual invoked implementation → 型付きResult/Failure・対象Source。
**Effect軸**：実Caller/Subject/Target/Current Grant → Commit Attempt → Receipt/Resource独立再読 → 結果不明時のRecovery。
**開示軸**：Consumer現在のScope/Disclosure → 実際に受領したReply/表示 → 古い応答・部分結果の隔離。

これらを一つのboolean qualifiedへ押し潰さない。N/A、NOT RUN、unsupported、partial、unknown、stale、refused、observed を区別する。Source同一であっても契約やPublisher authorityの失効、Contextの更新、Permission/Resource変更、Generator/Artifactの変更では相応の資格を再判定する。無関係Pluginだけの変更で選択済みBuild Closureを無条件に失効させない。

最小受入票の項目候補（新しい正本Schemaの導入ではなく、既存Evidenceを一覧化するProjection）：
- exact commit/PR HEAD、Workspace/Source revision、Operation/SubjectとSource位置
- Contract ID/version/fingerprint、Scope、Profile、選択Provider、実登録ID/entry
- 実Build inputs、Generator/Toolchain、Artifact/Host/Target identity
- Runtime Policy/Permission/Resource・現在Guardの判定（実作用を含む場合）
- Actual Invocation、Result/Failure/Effect、Consumer受領とCoverage
- 独立Expectedの参照、実行したテスト、未実施と失効条件

## 5. 最初の縦断ケースA：Inspector／X-TEA三視点（P0、read-only）

目的：同じ選択済みauthoring-tool Providerを、CLI Inspector、Studio Inspector、Studio Three-viewで実呼出しできること。#4612で三視点を独立サブプロジェクト化した後にCLI Three-viewも比較する。

- **#4611**：既存InspectorのHost選択・登録実装とのID/digest照合・guard・actual entry観測だけを小さく共通化。三視点のPin、coverage、occurrence、Source Fact/候補/受理Meaningは型付きOperation側に維持。新Registry/Resolverを増やさない。
- **#4612**：入れ子Go moduleの全CI検査、プラグイン分離、CLI/Studio同一実Provider。X-TEA読取部なしで製品packageがbuildできることを保持。
- **#950/#1124/#2497**：Readerのread-only証拠をMeaning受理や書戻し許可に昇格しない。Source変更後のPinはstaleで固定し、同名へ付替えない。

拒否条件：B選択→実A、未知Provider、別Workspace、Permission/Scope違い、古いGeneration、未取得Scope、Source変更後自動再結合、Provider Errorの秘密混入。外側CLI JSONとStudio Envelopeのbyte同一は要求せず、内側型付き結果と実入口を検証する。

## 6. 縦断ケースB：GeneratorとStudio自己再構築（最上位P0、並行）

目的：TEI自身によるDesktop Stage1→Stage2を一つの実Build Intentと実行Evidenceで結ぶ。ここでBuild-only RoleとRuntime Providerを別々に示す。

- **既存実績**：#4583の2026-11-13時点のB0同一HEAD Windows CI成功、B1-AのN0 Inspector／N2 esbuild input／N3 Go selection限定観測を再利用する。B1-B/C正式Builder接続、全Dependency Closure、B2、B3は未成立。
- **B1-B/C**：N0生成TS→N2 esbuild bundle→N3 Go embed/Wails exe の依存・実Invoked Builder・出力digest・次工程の実消費を同一Attemptで突合。Desktop非Plugin成果物をPlugin IDとして偽装せず、#2831の既存Build Execution/ArtifactSetへの薄いTarget Adapterで表現可能かを最初に確認。
- **B2**：CI shellの二回Buildではなく、Stage1実TEI Process imageがStage2のBuild PlanとBuilder呼出しを駆動したことを観測。
- **B3**：B2が出力した正確なStage2 Windows exeの実WebView2使用を確認。B0の別exe成功を足し合わせない。
- **非干渉**：生成時だけ使ったPluginはStage2 Runtimeに必須ではない。未選択Providerに強制import/initがないか、Build Closure/Link/Size/時間への影響を測定する。

Graph Compiler、DSL全面刷新、Trait全適合、実X-TEA原典の完全ModelerはB1-B/C〜B3の新Hard Gateにしない。

## 7. 縦断ケースC：Data Lifecycle／Hook／Script Launch（P1、effectful）

対象：一Function/ActionからData Context→Provider→更新→Hook→結果観測まで。#4502のFunction/Entry/Invocation、#1129のTable Lifecycle、#4198のObligation、#1174の実Grant/Effectを接続するが、新Script Runner/Transaction Engineを作らない。

- Hookの宣言uses、到達条件、guard、実Called、回数、成功・skip・notEvaluated・refused・Effect Unknownを分ける。
- Guard拒否時は実Provider Call 0回。行別処理の途中失敗では全体成功としない。二重呼出し、Rule評価前のeffectful uses、before/afterの誤再入を負例で拒否する。
- DB Commit成功・Receipt観測・Resource現在値の一致を分離。ack喪失、Network取消、post-commit外部作用不明はlookup-firstを守り、盲目的自動再送をしない。
- REA/Query/DB Providerの意味受理・多重度・業務キー・時間条件を、純粋なResult JSON同値で代替しない。

最初は合成read-only検査を先に固定し、実DB/実Hostへのeffectful切替は独立ExpectedとCurrent Grantの条件を満たした範囲だけで進める。Graph全面実装をStudio Self-buildに先行させない。

## 8. 縦断ケースD：外部Hostと関連製品（P2、非Blocking）

TEI自身が生成したProviderをTEI非依存のJava/.NET/Node等Consumerで利用する#2390を代表ケースとする。Contract asset、実Provider、Host binding recipe、Hostの現在のAuthorityを分離する。Wasm/WITの同一物理成果物要求は#1861で実二Host比較後に判断する。MCP/CUIはOperation公開Adapterであり、DSH/Claude Code/Codexなど特定Harnessの仕様をTEIの正本にしない。

SHOはSource/図表・文書投影、SUIは情報統合Consumer、KONは必要時のData/Evidence接続先として利用可能にし、TEIのContract/Meaning/Authorityを複製しない。外部から取得したReferenceがstale/partial/unavailableなら、出典状態と検証条件を保つ。外部製品・配布Registry・Marketplace完成はP0をBlockしない。

## 9. 既存テストへの引渡し（新しい独立Expected正本は増やさない）

| 観測 | 主担当 | 独立の反例・合格線 |
| --- | --- | --- |
| CB-01 Contract/Scope選択 | #951/#1016 | 同名／複数候補、契約版・fingerprint違い、authoring→runtime誤用を拒否 |
| CB-02 Selected→Actual Entry | #1251/#4611/#4612 | B選択→A実行（結果同値）を拒否。Source/Workspace/Generationの変化を保持 |
| CB-03 未選択Pluginの非干渉 | #1251/#2600/#3267 | Closureに不要な実装を含めない。未使用は実Build/Linkで観測 |
| CB-04 Source／逆写像 | #950/#1124/#2497/#2621 | partial、同名別ID、保存後Source改版、非可逆編集を拒否 |
| CB-05 Stage1→Stage2 | #2831/#4583/#1824 | N0〜N3のPlan/実Invoker/実消費とStage2自身の実Windows使用が同一因果経路 |
| CB-06 Hook/Effect | #1129/#4198/#1174/#4502 | Guard前副作用、重複Call、途中失敗、ack喪失後自動再送を拒否 |
| CB-07 失効と独立Evidence | #2514/#1180/#2600 | A→B→A、Provider/Toolchain/Grant変更、unknown/NOT RUNを成功に昇格させない |
| CB-08 外部Consumer | #2390/#1861/#2589 | TEI CLI/Studio非依存で実呼出し、Host authority無検証ならproduction未資格 |

CB番号はこの文書の参照ラベルであり、新しいTest Runner、Schema、Gateを定義しない。各Ownerの既存fixture/Expectedに最小の差分を加え、同じ正常・拒否基準を二重実装しない。

## 10. 段階的実装順と停止線

**P0-a（先行）**：#4583のB1-B/Cに必要な一工程のBuild Consumptionを成立させる。Pluginの一般化やStudio管理UIを前提にしない。
**P0-b（並行）**：#4611→#4612でInspector/Three-viewのHost選択・実Entry・二Surfaceを最小共通化。#4570が未統合の間はPR candidateとmainを区別する。
**P1-a**：#951で新Contract/Provider/Conformanceへの一方向移行を実Consumerで実証し、旧固定CapabilityKindとの二重Resolver/Authorityを停止。
**P1-b**：#1129/#4198/#1174でeffectful Providerの実行順・回数・Failure/Recoveryの一ケースを受理条件つきで検証。
**P2**：Studio Plugin Manager #2044、外部Host #2390、Wasm #1861、公開Registry #2589の順序・Scopeを適用環境ごとに判断。

**停止・差戻し**：
- 第二の契約／Provider／Authority正本が必要になる。
- Script/Hook/Agent/Graphが別のAction正本やEffect Engineを設ける。
- 生成器の自己申告を独立Expectedや実Builder検証の代わりにする。
- Build／Runtimeで実際に必要なProviderを、Plugin Lockにないという理由で未使用扱いする。
- Source、選択済みProvider、Grant、Scopeのdriftがあるのに旧成功Responseを再利用する。
- 部分的なCI PASSをOwner受理、本番安全性、実Desktop Self-buildへ昇格する。

## 11. 計画への反映と整合記録

本書は#4601（Plugin RoleとStudio再入、Draft）、#4593（MASTER_PLAN B1追補、Draft）、#4589（Hook/Lifecycle、Open）に重複する大きな仕様を追加しない。これらが同時に触るMASTER_PLAN.mdやPLUGIN_FIRST_TOOLING_RUNTIME_DOGFOOD_2026-11.mdを今回のPRから直接変更せず、各Issue・PRへの参照を通じて統合する。

2026-11-14の静的参照：mainのプラグイン固定CapabilityKind、Inspector Hostの選択済み関数・symbol観測と実行file digest、#4570のOpen/Draft、#4583のB0/B1-A限定・B1-B/C/B2/B3未成立を確認した。**この文書の新Go／Node／実Windows／実DB実験と独立検収はNOT RUN**。本書をmain統合しない限り製品計画の正式採用ではなく、採用には既存Ownerのレビューと実証が必要である。


## 12. 2026-11-14：Builder Pinから正式Build選択への不足一辺（P0-B1）

### 12.1 前回以降に確認できた実装・検証

- [Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD 24bfaed8／[Windows CI #43450255223](https://github.com/hat47x/tei-platform/actions/runs/43450255223)は、実TEI CLIのprimitive self-buildからN0 Generatorの別プロセスを起動し、親子image digest・子argv・exit・独立再読した生成TS digestを照合。生成TSをN2 esbuild→N3 Go embed→生成Wails/WebView2へ渡した同一HEADの限定成功。これはStage1生成TEIの再入や正式Builder選択ではない。
- [独立Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) exact HEAD 929fcf857cd0f6a792a69fb23cadfb5eccc05e90 は、既存CLI self-build primitiveに任意の expected-child-digest と起動前の実子ファイルhash照合、selectedChildDigest／childPinChecked／childPinMatchedというReceipt項目を追加。Selected B/Actual A、形式不正、Boundary Launcherを迂回させない負例、pin一致でも虚偽Artifact digestは不合格、のGoテストを保持する。[Repository Health #43537292191](https://github.com/hat47x/tei-platform/actions/runs/43537292191)など同HEADの**8 Workflowがcompleted/success**を確認した。
- このpinは**Caller supplied**であってBuildIntent Resolver／Builder Catalog由来ではない。実行前にhashしたパスがOS Loaderで開かれる間のTOCTOU/ABAも保証しない。#4623はOpen/Draftのまま。別HEADの成功を#4587のWindows pin検証へ合算しない。

### 12.2 不足一辺を細分化した実使用の6段階

以下は**既存BuildIntent／Builder／ArtifactSet／Receiptの観測Projection**であり、新しいBuild Schema・第二Runner・第二Authorityの提案ではない。

| 段階 | 確認すべき事実 | 境界／Owner |
| --- | --- | --- |
| S0 Build Subject/Intent | 対象がPluginか非Plugin Desktopか、どの成果物Roleが必要か | #2831/#4583。DesktopをPlugin identityとして偽装しない |
| S1 Resolved Builder | 既存Resolver/Catalogが選択したexact Builder identity、digest、Target、Generation、利用Contract | #2831/#951。Caller pinとは別資格 |
| S2 Prestart Child | 実childファイルhashとS1由来pinを起動前に照合し不一致ではChildもLauncherも0回 | #4623。OS Loaderまでのbyte保証は別 |
| S3 Actual Invoker | Hostが実起動したProcess image/argv、Toolchain、適用した環境制限、終了結果 | #4583/#1370。exit 0だけでConformanceを証明しない |
| S4 Produced Artifact | 実Disk上のTS/JS/exeのbyte hash、宣言したOutput Role、失敗時残存・no-clobber | #943/#2831。Child自己申告だけでは不足 |
| S5 Actual Consumer | N0 TS→N2 esbuild→N3 Go embed→Stage2 Windows exe→WebView2が同一因果Attemptの成果物を実使用 | #4583/#1824。別CI成功の寄せ集めは不可 |

S1とS2の一致、S4とS5の一致は、それぞれ**該当する辺の限定証拠**。OS Loaderの実byte、Read-timeの全依存閉包、OS強制W、Current Grant、Owner意味受理を自動付与しない。Resultの一致のみで実Builder、実Link、Consumerを代替しない。

### 12.3 次の最小実装・判定

1. #4623の共通CLI pin検査を独立したexact HEADで保全し、**既存BuildIntent Resolverが解決したBuilderIdentity.ArtifactDigest**をcaller pinへ渡す薄い接続を一件だけ実装する。子PathやDigestをOperationの入力から自由に付け替えない。Builder選択とRuntime Provider選択は別の目的なので同じAPIへ強制統合しない。
2. S1/S2/S3を同一Attemptで照合し、選択B・実起動Aなら同一Outputでも拒否。pin一致をOS Loaderにロードされたbytesの証明と呼ばない。将来のTOCTOU低減は、不変Artifact Store、Host/OSの実ファイル拘束、実Process image観測で可能な範囲を個別検証する。
3. 一成果物だけを**非PluginのBuild Execution／ArtifactSet**として受け渡す。現BuildIntentのSubject制限だけを緩め、Desktopをplugin.BuildPlanのPlugin欄へ詰める回避策は採らない。
4. N2出力のrole/digestとN3で実EmbedしたJSを実Byteで照合。go listで選択された303件とesbuildで選択された24件は**選択報告**であり、Go/Node/SDK/Cacheの全依存と実Read-time byte閉包は未保証。
5. B1-B/Cが成立してから、**生成Stage1自身**の実ProcessがStage2 Desktop Buildを駆動するB2、**そのStage2実exe**のWindows WebView2受領B3を別々に検証。CI Shellが二回Buildを呼んだだけでB2にしない。

### 12.4 独立Expectedは既存Issueに集約する

- [#4583 BD-13](https://github.com/hat47x/tei-platform/issues/4583)：CallerがBをpinしChild Bが起動してもResolverはAを選択している場合、pin-local PASSをS1→S2のPASSへ昇格しない。
- [#4583 BD-10](https://github.com/hat47x/tei-platform/issues/4583)：Source A→B→Aで前後Hash一致、Read-time閉包はunknown。
- [#4583 BD-11](https://github.com/hat47x/tei-platform/issues/4583)：N2生成JS AとN3実Embed JS Bの相違はProducer/Consumer不一致でFAIL。
- [#4583 BD-12](https://github.com/hat47x/tei-platform/issues/4583)：N3をCI Shellが直接起動したなら、exe実生成・WebView2成功でもB2は未実施。
- [#4623の既存Go負例](https://github.com/hat47x/tei-platform/pull/4623)：Child pin不一致ではBoundary Launcher起動もArtifact生成も0。pin一致でもChildがArtifact digestを偽る場合はCross-check失敗。
- #951/#2600の既存資格：Contract/Target/Source/Toolchain/Grantの変更を必要範囲で再検証し、無関係の未選択Plugin追加・更新では選択済みClosureを過剰失効させない。

## 13. Script Launch／Hook／AI Harnessとの意味境界

[Script Launch Draft #4618](https://github.com/hat47x/tei-platform/pull/4618)のPrepare→Admit→Invoke→ObserveはS0〜S5と**宣言・選択・実呼出し・結果観測の分離原則だけを共有**する。Script RunnerとBuilderを単一Runtime/Claimへ統合しない。Functionの意味受理、Entry公開、現在Grant、実Providerの呼出しとResult値のNULL／欠損／未取得も別資格である。

[Hook Draft #4602](https://github.com/hat47x/tei-platform/pull/4602)は作用不明／lookup-firstがHook.Error等で消え、warnによって後続作用が進む問題に対する局所修正候補。Guard-before-use、二重Called、部分Rowset、実DB receiptなどは#1129/#4198/#1174が引き続き検証する。N0 pure Generator起動の成功を、effectful Hookの安全性へ流用しない。

DSH、Codex、Claude Code、Studio AIは設計・Code・Builder候補を提案できるが、Resolverの正式選択、Hostの起動前検証、Ownerの意味受理を肩代わりしない。Harness変更で実行資格が黙って変わらないよう、Plan提案と現在のAuthority、実使用Artifactを別に観測する。

**今回の追補の資格上限**：既存PRとCIの現在状態の確認、設計文書への引渡しのみ。新たなGo/Node/Windows/実DB試験、B1-B/Cの実BuildIntent接続、B2/B3 Desktop再入、Owner受理はNOT RUN / NOT ACCEPTED。


## 14. 2026-11-14：Plugin管理・静的包含・実行世代を混同しない（#2044 × #1223 × #4620）

### 14.1 正本の重複を避ける責務

[横断主計画Draft #4620](https://github.com/hat47x/tei-platform/pull/4620)は現在の優先順と「選択的再資格化」の上位判断を担当する。本節は、[Plugin Manager #2044](https://github.com/hat47x/tei-platform/issues/2044) S1～S5 の**製品受入へ渡す状態分類と反証票**に限定する。同じ主計画や新Plugin Managerを作らない。

現行製品には plugin.Definition／Environment構成選択／plugin.Lock はあるが、Plugin Manager専用の installed／enabled 永続状態や settingsVersion はない。三つの参考管理パッケージのテストPASSは実CLI/Studio管理操作の合格ではない。[#4376](https://github.com/hat47x/tei-platform/issues/4376)はCLIのread-only inventory、[#4377](https://github.com/hat47x/tei-platform/issues/4377)はread-only plan、[#4378](https://github.com/hat47x/tei-platform/issues/4378)は設定源・原子性・監査、[#4379](https://github.com/hat47x/tei-platform/issues/4379)は実Studio、[#4380](https://github.com/hat47x/tei-platform/issues/4380)は不可逆操作を所有する。

### 14.2 ひとつのenabledを使わずに投影する状態軸

| 独立の状態 | 読取りで観測できる例 | 証拠が無いときの表現 | Owner |
| --- | --- | --- | --- |
| Source宣言・形式 | Workspaceに保持されたDefinition/入力形式、Source provenance | 未ロード・未知形式・partialを宣言なし／不存在へ潰さない | #4376/#1920/#2621 |
| 導入可能性 | 実装Artifact/適合Contract/Packageが解決できるか | 宣言だけではinstalledとしない | #4376/#2589 |
| 有効設定 | Ownerが許した設定Storeの現在版 | 現状not-modelled。Environment選択をenabledへ代入しない | #4378 |
| 構成選択 | 実Environment/Profile/Contractでexact Providerを選んだか | Candidate Catalogにあるだけではselectedでない | #951/#1016 |
| 実Buildへの包含 | selected Target Build Closureと実Link/Embed、Artifact identity | Lockのみならphysical inclusionはunknown | #2600/#3267/#2831 |
| 実行時利用 | loaded/observed entry/actual invoked、Generation、active in-flight | embeddedだが非呼出なら invoked とはしない | #1251/#1223/#1174 |
| 現在Authority・Effect | Target現在Grant、Resource scope、Effect・Receipt | 旧Grant/旧Receipt/旧世代PASSを現在資格に昇格しない | #1174/#1223 |
| 設計データの保持 | 原Source bytes、版、未知断片、保持対象 | plugin disable/remove はSourceやデータを削除しない | #4378/#4380/#2497 |

静的Go Reference Hostの無条件importは「固定バイナリへ含まれている」ことの候補でしかなく、案件Profileの選択や現行Runtime使用の証拠にはならない。Go単一バイナリへ動的Loader/Watcherを必須としない。長寿命Process/Wasm等Targetを選んだ場合だけ#1223のreadiness→cutover→drain→cleanupを検証する。

### 14.3 Inventory → Plan → Apply の再検証契約

**Read**：#4376 は現Workspace/Source、Declaration/Selection/Lock、判明したArtifact/Scopeを収集するが、存在しないinstalled/enabled/settingsVersionは補完・推定しない。未取得なら原因つきnot-modelled/unknown。秘密や顧客データ本体をEvidenceへ入れない。

**Plan**：#4377 は変更案・影響Contract・必須依存、Source/保持データの影響、必要な再生成/再起動/Generation切替、許可・禁止・unknown、正確なPrecondition revisionを提示する。Plan作成は状態変更・権限付与・現在Grantの保証ではない。required依存A→B→CでBの無効化を要求されたらAの扱いを明示候補として返し、勝手にAとCを停止しない。任意依存は現行DSLに無いため、任意性を仮定した自動判定をしない。

**Apply**：#4378/#2550 は、Ownerが採用した設定源のsettingsVersion、Workspace Source revision/bytes、Contract/Provider Lock/依存集合、対象Environment/Target、Actor/Current Grantと実操作のEffect範囲を**操作直前に再照合**する。Preconditionが不明・古い・競合するなら適用せずread/planの更新を要求。AgentによるPlan提案やUIの古い成功・確認済み表示を実Apply権限に変えない。変更可能な設定範囲と現Source保存・不可逆データ削除を一つのTransactionで実行したように広告しない。

**After**：設定への書込みが成功した場合、実際に変化した設定版／Generation／Runtime Authorizationを再Readで確認する。新規SelectionやBuildが必要でも成功を先取りしない。Source bytesを変更しないdisableと、Sourceを実際に変える編集／Format migrationは別操作。Runtime切替済みでも旧in-flightのEffect/Receiptを消失したものと扱わない。

設定源の候補（Host Store、Workspace管理ファイル等）は#4378 D1で未決定。ここでは新Store/Namespaceを指定せず、Sourceを変えずに設定を変えるならSource宣言と管理設定が二つの**別用途の情報源**になる点だけを明示する。「第二Resolver/Meaning正本を作らない」ことと「設定の保持先が別である」ことは同じ命題ではない。

### 14.4 変更種別ごとの最小再資格化（新Gateではなく既存Issueへ接続）

| 変更・障害 | Source/Meaning | Build/Artifact | 実行・権限・Generation | Owner/試験 |
| --- | --- | --- | --- | --- |
| 未選択Pluginだけを追加（選択閉包に不変） | 原Source/Accepted Meaningは維持可能 | 実Build非包含を証明できれば再生成省略候補。静的import実包含なら別 | 現Consumer/Grantに影響なしなら再資格不要候補。未観測はunknown | #4377/#2600：PM-C01 |
| 必須Guard/Providerを無効化 | Source原文は保持 | 新構成の生成/選択可否を再判定。勝手にfallbackしない | 現Operationを拒否または既存Authorityで安全にdrain。新Grantなしに実行継続しない | #951/#1223/#1174：PM-C02 |
| ProviderをA→B→Aで戻す／Source移動 | 同byteでもRevision/Source location/前回受理資格を再確認 | 実Build入力とGenerator/Lockの時点を比較し再利用可否を個別判断 | 古いRuntime Grant/世代/Consumer受領は自動復活しない | #2600/#2514：PM-C03 |
| 実行中に新Generationへcutover、旧Grantを失効 | Source変更なしもありうる | Static Goの場合は次のBinary/再起動、動的Targetなら現物実切替を検査 | 新規呼出しは新世代へ。旧in-flightのCommit直前Grant、Effect Unknown/receipt lookupはHostで再評価 | #1223/#1174：PM-C04 |
| Diagram layout-only変更 | Business IDと意味は不変（条件つき） | 意味Codeの再生成不要候補。表示Assetだけ変わる場合はそのConsumerに限定 | Runtimeの業務Effect資格を無条件失効しない | #2497/#1124/#2600：PM-C05 |
| Plugin disabledだが旧Sourceに未知断片あり | Raw Draftなら元byte保存・再開可。構造受理不可の場合は非受理 | Parsed/Acceptedを装って生成しない | ManagerのdisableとSourceの構造編集を分ける | #4378/#4380/#2621：PM-C06 |
| plan作成後settingsVersion/Lock/Source/Permissionが更新 | 元Sourceは変更しない | 旧Plan由来のBuild操作は適用不可 | 古いUI/AI confirmationを無効とし、適用前に再Read/再Plan | #4377/#4378/#1529：PM-C07 |
| remove-pluginとdelete-dataを一括指定 | Source/保持データの境界を明示 | Artifact/Source削除を自動実行しない | 後者は別のOwner承認、監査・復旧・外部Resourceの確認が必要 | #4380/#2553：PM-C08 |

PM-C01〜08は本設計のCross-caseラベルであり、新Scenario/Expected正本を作らない。#1180/#2514および#4376〜#4380既存PM1〜PM5の該当負例へ最小限接続する。該当変更の閉包がpartial/unknownなら「影響なし」ではなく不確定を明示する。

### 14.5 形式版とRaw保存の停止条件

[#1920](https://github.com/hat47x/tei-platform/issues/1920)はGitHub上Closedだが、本文の形式版・中央変換・未知断片保持の完了チェックは空欄であり、**現製品の未知属性・未知子要素を持つ .tei のlossless parse/editを証明しない**。Plugin Managerは未知Sourceを消去・自動format変換せず、許されたRaw Draft保存と、構造的に受理済みのSemantic Edit/Promoteを別操作にする。Known parserで拒否したSourceをempty inventory・未インストール確定・safe deleteへ昇格しない。

**次の最小引渡し**：P0 Studio Self-build #4583とInspector selected-entry #4611/#4612はそのまま最優先。並行で#4376のread-only inventoryを実Workspace/Lock/CLIで一件成立させ、#4377の非破壊plan、#4378の設定源採用判断を順に行う。#4378の設定源が未決定のままApply/disable UIを製品化しない。

**今回の資格上限**：既存Issue/PR/sourceの静的照合と設計受入案。新Go/Windows UI/外部Resource操作、実Generation cutover、Plugin Manager製品E2E、Owner採用・意味受理は NOT RUN / NOT ACCEPTED。


## 15. 2026-11-14：Resolved Builder Pinの現HEAD検収と次に切る差分

§12は起点時点の不足として保持する。現時点では[Draft #4623](https://github.com/hat47x/tei-platform/pull/4623) exact HEAD f1e219a25976946bcc8a19ef57f5490333ec8a84 で **opt-inのin-process BuildIntent resolver→実Childの起動前Digest束縛**が実装候補として追加された。実コードでは bind-resolved-builder を要求するとBuild Catalog/digest/Source Rootが必須となり、ResolvedBuildReceipt の PlanFingerprintとBuilder ArtifactDigestを照合して既存driveSelfBuildのpinへ渡す。不一致は子・成果物を作らず SELF_BUILD_BUILDER_SELECTION_MISMATCH のErrorとして報告する。回帰テストにはcatalog選択→実childの正例、選択B/実Aの拒否、Catalog無しの拒否が含まれる。

[このHEADの8 Workflow](https://github.com/hat47x/tei-platform/pull/4623)は2026-11-14に全件completed/successと確認済み（うち[Repository Health #43539856799](https://github.com/hat47x/tei-platform/actions/runs/43539856799)、[TEI Self-build #43539856841](https://github.com/hat47x/tei-platform/actions/runs/43539856841)）。したがって**§12 S1→S2の「Resolver選択digestをpinへ接続する」という限定Gapは、#4623 Draft上のopt-in Go経路で前進した**。#4623未マージであり、同HEADの正式Desktop/Wails CI、OS Loader TOCTOU、Product非Plugin Build Intent、N2/N3のBuildExecution/ArtifactSet接続、B2/B3、Owner受理は別の未検証条件である。

次に分割する小差分：
1. **Product Buildの一成果物**：Plugin Subject以外のDesktop executableとN2 JavaScript Bundleを、既存BuildExecution/ArtifactSetで正しいSubject/Roleとし、Resolver pin→actual Builder→Produced/Consumed digestを同一Attemptで記録する。#4583/#2831担当。
2. **実Invokerの証拠範囲**：Child file hash照合の後でpathから起動するTOCTOU、外部Toolchainの未観測、現在Authority、Source A→B→AのRead-time漂流をunknownとして返す。prestart successが完全Closureと表示されないことを確認。#2600/#1174担当。
3. **失敗の型付き投影**：現#4623差分は実Child pin不一致をエラーテキストの部分一致で識別して公開Failure Codeへ変換する経路を持つ。実装レビューでは可能なら既存共通Failure/typed sentinelへの適合を検査し、異なるOS Errorや同文言の別FailureをBuilder identity mismatchに誤分類しないことを独立負例にする。新Failure基盤は追加しない。#4623/#4315担当。
4. **実Windows再検収**：#4587は現時点でDraft・mainとの競合があり、旧HEADのWindows PASSは最新#4623の実Desktop N0 pin適用のPASSを保証しない。統合済み候補HEADでordinary/generated両Jobを再実行後、B2/B3を独立に判定する。#4583/#1824担当。

今後の計画更新は、#4620がMASTER_PLAN上位主線、#4621が本Cross-case詳細、#4583がBuild実装受入、#4376～#4380がManager実装受入を所有する分離を守る。


## 16. 2026-11-14：Studio Web共有サーバ・複数利用者・同時編集でのPlugin適格性

### 16.1 前提・Scope（Web版の新設計候補、未採用）

ユーザー構想：Desktop版は開発者ごとのローカルWails/Go環境で作業しGitリポジトリ経由で共有する。Web版は単一または必要時HA化した**開発サーバ上でのチーム開発・同時編集**を想定する。画面・Editor・Source/Model/Build/Plugin/AIとの接続は共通Operationを可能な限り利用し、Desktop→Web差分を小さくする。ただし**Webの共有セッション・認可・同時変更はDesktopのlocal Host条件をコピーしない**。

本節は#1824のStudio Host/Web分離、#1158/#3421の共通Operation/HTTP Adapter、#1016/#951の選択と適合、#1124/#2621のAuthoring/Acceptance、#1529の閉鎖済み外部編集競合Reference、#4378の管理設定源、#1223のGeneration、#1174の現在Grant、#2515のHarness機密境界へ**追加の横断受入条件**を渡す。Web Collaboration Engine、第二Source Registry/Resolver、TEI専用認証Host、CRDT実装、共有Git Serverの新正本をここでは採用しない。

### 16.2 多利用者で区別する7つのコンテキスト

| Context | 正しい責務 | 同一視してはいけないもの |
| --- | --- | --- |
| Server Deployment／Plugin Inventory | 実Process/Artifactにロード・同梱され得るProviderとHost限界 | 案件で選択済み、利用者に許可済み |
| Workspace／Project／Branch | Source/Manifest/Accepted Meaning/ToolchainとGit作業版 | 他Workspaceの同名Subject、全サーバのPlugin設定 |
| Composition Snapshot | Contract/Provider/Runtime/Profile/Lockのexact選択、適用Generation | installed/enabled、Clientが任意指定するProvider |
| Authenticated Principal／Effective Scope | Hostで検証済みの利用者・Workspace・Actor/許可Resource・現在失効 | Browserから送ったactor/role文字列、local Studioのno-auth |
| Authoring Session／Draft | Client/Tab・編集中Buffer、Source base byte/revision、対象Pin、ProposalとPreview | 受理済みSource、他利用者のDraft/AIセッション |
| Invocation／Build Attempt | 実Entry/Builder/Source Closure、どのGenerationを使用したか、Effect/Receipt | 過去成功、同じ結果、同じPlugin package |
| Consumer／Disclosure | 実際に返す値・Error/Trace/AI Tool/LogのAudienceとPolicy時点 | Serverが読取れた全Data、他利用者への公開権限 |

「同じPlugin ID」の共有は**同じImplementationを使える可能性**に過ぎない。Providerの内部Cache・Session・Temporary File・Connection PoolにはWorkspace/Principal/Resourceごとの強制分離が必要であり、共用可能かどうかはProvider Contractに応じたHostの実隔離Evidenceで判断する。静的Goバイナリに一つのProviderが同梱されても、各Workspaceの選択／Runtime Grant／現在Audienceは独立に適用する。Workspace単位で「未選択」を理由に物理的な非リンクを要求しない（物理除外はServer/Target Artifact単位のBuild資格）。

### 16.3 一つのOperationがDesktopとWebで変わらない条件

共通Operationは既存ADR-0048の型付きRequest/Result/Failure/Unknown/Scope/Effect契約を維持する。Wailsはlocal IPC + trustedRoot、WebはAuthenticated HTTP/session + Hostが解決したWorkspace Rootを使い、**Transportへ業務意味や第二Provider Resolverを置かない**。Target固有なのは、認証・Connection origin・CSRF/CORS・Host Session・共有Resourceの実強制である。Webで認証Provider不在/不正ならlocal no-authへfallbackしない。

Authoring/Preview/Applyは原則として二時点：
1. Read/Preview時：Workspace/branch/base Source byte fingerprint、Meaning/Contract/Provider/Scope/Session epoch、Subject/Occurrence/Source locationを凍結し、Read/partial/Unknownを維持。
2. Apply直前：HostでActor・Root・Resource・Current Grant、実Source byte/Revision、受理の対象と依存、Provider binding/Generation、Write capabilityを**再読取・再検証**。異なるならstale/conflict/refusedを返し、自動merge/Acceptしない。成功後は実Source再読込・独立Expectedで確認。#1529閉鎖済みReferenceのSource drift・Watcher欠落負例をWeb共有環境で再利用するが、Go合成PASS＝実HTTP/HAの証拠ではない。

Git commit hashやブランチ名はSource baseの一部だが、**Gitは同時編集の排他保証ではない**。未コミットSource byte、Server内のAuthoring Draft、Git commit、Accepted Meaning、実Artifact Revisionを分ける。最初の実証は**一Sourceにつき同時Apply時のCAS/競合拒否**に限定し、Operational TransformationやCRDT、複数File横断トランザクションは採用せず必要時に独立評価する。Webにおける可視化の同時編集（Cursor/Selection）とMeaningの同時変更も別の問題である。

AI Harness/Agentは利用者Sessionの許可済みSource/Projectionを**明示的に委譲された範囲だけ**使う。AgentのTask/Proposal・Share/Review・Model outputをHostのBusiness Acceptance/Grantへ昇格させない。DSHの共用checkoutやSession broadcast/Logから、他利用者・他WorkspaceへRaw/Traceが送出されないことは#2515/#1178が実送出観測を所有する。

### 16.4 Plugin管理と共有Serverの切替単位

- **Workspace-local selection/configの変更**：当該Workspaceの将来呼出しと必要なBuild/Generationに影響する。他WorkspaceのProvider選択やSourceを無条件に変更しない。#4378の設定源は未採用なので、Web用Global Storeを本節から決めない。
- **Server-level Plugin unload/update**：同じProcessに複数Workspace/SessionのActive Callがある。単一Workspaceでdisableしても、他Workspaceで必要なImplementationを即物理停止してはならない。共通Artifact切替時は全使用ClosureとIn-flight/Grant/Effect/Resource leaseを#1223で検査し、必要に応じて変更を拒否または後段へ延期する。
- **静的Go Desktop/Server**：製品のProvider更新は原則新Artifactと安全なProcess再起動/Deployment。静的Providerの任意hot-swapや無停止は標準保証ではない。
- **動的/HA Target**：readiness→新規RequestのGeneration切替→旧in-flight drain→cleanupに加え、**旧世代に結び付いたSession／Workspaceの現在権限**、永続Effect Unknown/Receiptと重複作用を#1174で再検査。Replica間で古い設定・旧Actor Grant・異なるPlugin lockを保持する場合に無停止資格を出さない。
- **Server failure/Failover**：Web Session/Source Base/Operation Identity/Receiptの再取得可能な範囲を区別し、未永続なHost内Memoryを正本と見なさない。HAの合意・永続化・Fencing方式はこの段階で採用せず、単一Server正例→Failoverで偽PASSしない負例という順で検証する。Webの最初のP0でHA実装は要求しない。

### 16.5 既存Expectedへ渡すWeb Plugin Cross-cases（番号は参照ラベル）

| Case | 正常・拒否を区別する要点 | 既存Owner |
| --- | --- | --- |
| SWP-01 同一Serverで二Workspace | Workspace A選択Provider P1／B選択P2を同じServerで処理。Source/Lock/Entry/Consumerは混ざらない | #951/#1016/#1251/#3421 |
| SWP-02 他利用者のRequest改変 | Web Browserがactor/Workspace/Provider IDを偽装。他ActorのSource/Trace/AI結果を開示せずHostで拒否 | #3421/#1043/#1174/#1178 |
| SWP-03 同一Sourceの同時編集 | Editor X/Yが同byteをRead、XがApply後Yが旧PreviewをApply。Yはconflict・再Review、XのSource byteは維持 | #1824/#1124/#1529/#2621 |
| SWP-04 プラグイン切替と旧Session | A WorkspaceがProviderをdisableしてもB Workspaceの使用を物理停止しない。A旧in-flightのCommit/Receiptは別資格 | #4378/#1223/#1174 |
| SWP-05 同じActorの別Tab/別Buffer | Tab1のPin/DirtyとTab2のSource Readを混ぜない。A→B→Aの後着Replyで旧Successを再提示しない | #1824/#4515/#2597 |
| SWP-06 Agent出力の横断漏洩 | Agent XがSource/Logを扱っても別Actor Y向けTool/Team/Log/AI outputへ未許可開示しない。送出実Byteを観測 | #2515/#1178/#3421 |
| SWP-07 権限変更とCache | 同じResultをCacheから返せてもAudience/Grant/Workspaceが変わればDisclosure再評価。旧Grantを自動復活させない | #1174/#1178/#2600 |
| SWP-08 HAまたは再起動 | 旧Session/旧Generation/未確定Effectを成功扱いしない。未永続Draftはlost/unavailableとし、重複Writeを自動再送しない | #1223/#1824/#1174 |

SWP-01〜08は**既存#1180/#2514のExpectedへ引き渡す参照名**であり、新しい認可正本・Editor/Scenario Runner・Session Storeを定義しない。Synthetic 2 Workspace × 2 Principal × 2 Provider、同名別Subject・違うSource bytes、Read/Apply各2時点を最初の最小fixtureにする。

### 16.6 優先順位と採用判定

**P0（変更なし）**：#4583のStudio Desktop BuildIntent→Stage1→Stage2→Windows Consumer、#4611/#4612のInspector/Three-view実Providerと#1824の実IPC。Webの本番認証/同時編集/HAをこれらの新Hard Gateにしない。

**P1（並行してHeadless準備）**：#3421の実HTTP read-only共通Operationへ二Workspace/二PrincipalのHost BindingとSWP-01/02/07を接続。#4376のread-only Plugin InventoryでServer available／Workspace selected／Session authorizedを別列で返す候補を検証する。#1158の既存Operationを共用し、Web用の新CLIやSecond Resolverを作らない。

**P2（Web共有編集）**：#1824/#1124/#2621で一Sourceの二Session CAS/Preview/Apply/Source再Read（SWP-03/05）。#4378のSource不変設定源の採否と、Actor/Workspace単位のPlugin apply/disableの範囲を実証。CRDT等の同時入力共有機構は実際の競合頻度・Latency・操作粒度を測ってから評価する。

**P3（選択した長寿命Targetのみ）**：#1223/#1174のGeneration切替/drain/failover（SWP-04/08）、#2515のAgent送出境界（SWP-06）の実Server/OS強制を確認。HA構成・IdP方式・Storage実装を現時点で採用済みとしない。

**検証上限**：Studio Webの共有サーバ・同時編集・実認証・HA・Web版Plugin Managerは今回実装していない。本節は既存実装・Issue/議論の設計統合のみ。単一Serverの二Principal実HTTPテストもNOT RUN。採用判断は各Ownerの実測・独立Expected・正式レビュー後。


## 17. 2026-11-14：Wails ServerとClient-side Go/WASMの役割分離（#1824 / #1158 / #3421）

### 17.1 確認した現在の実装と公式仕様

- 現行mainの cmd/tei-studio/go.mod はGo 1.26.11・Wails v3.0.0-beta.29。window_server.goはserver build tagの差分であり、cmd/tei-studio/README.mdは**127.0.0.1限定の参照PoC**を明記。Serverでビルドできることは共有開発サーバへの本番公開・認証済みWeb版の完成を示さない。
- cmd/tei-studio/main.go のWails StudioBridgeはread-onlyのWorkspace操作に加え、ExecuteCommand系、OpenShoDocument/ResumeShoDocument、参照Storeへの操作などを公開メソッドとして持つ。**Desktopのserviceをそのままpublic Serverへ公開しない**。public web用には契約・認可が検証されたread-only facadeを最小限列挙する方式を比較し、第二Business Operation正本は作らない。
- internal/operations/dependency_test.goはWails、network、syscall/jsへの逆依存を禁止。Browser Go/WASM候補の追加でも、このHost中立の既存保証を維持する。
- Wails公式（https://wails.io/guides/server-build/ 、2026-11-13更新）ではserver tagで同じGo Service binding/Web frontendがheadless HTTPサーバとして動く。**TEI Studio Web版にWails Serverを現実の比較候補として採用**し、別HTTP Frameworkを自動採用しない。ただし本番Web Authority/同時編集/HAをWailsが自動提供するわけではない。
- 公式Server文書の標準イベントは**全接続ブラウザへWebSocket broadcast**し、event.Senderはbrowser-Nという接続識別子。**browser-N≠Principal≠Workspace≠Audience**。Source、Trace、Provider結果、AI Task/Promptなどを無条件にEventへ載せない。一般通知は非機密、対象を持つ結果はHostで認証済みAudience/Scopeを確認した応答・個別配信経路を必要とする。
- Wails ServerのOrigin/Host/loopback制限は確認できるが、それだけでユーザー認証やWorkspace認可を証明しない。Webでは既存#1043/#3421のTLS/Origin/CSRF/Cookie/Session/Resource/Rate/Request sizeの実強制を別に試験する。
- Go公式（https://go.dev/wiki/WebAssembly）ではブラウザGo/WASMはGOOS=js GOARCH=wasmと同一Go系統のwasm_exec.jsを使う。WASI実行（wasip1）や外部Host/WIT Plugin ABI（#1861）は別用途であり、一つのWASM実行契約へ自動統合しない。

### 17.2 共通化する五層と保持する境界

| 層 | Desktop Wails | Web Wails Server | Browserの任意Go/WASM |
| --- | --- | --- | --- |
| Source/Meaning/Contract | TEI正本・既存受理 | Workspace別Source/意味・同じContract | Hostが明示的に公開したread-only投影／未確定診断 |
| Pure Computation | Go native | 共通Go native | 純粋・決定的で移植できる部分集合だけ候補 |
| Application Operation | Hostが確認した型付きGo Operation | 認証・Workspace確認後に同じOperation | Result投影だけ。独自Resolver・Commit・Authorityなし |
| Transport/Host | local Wails IPC/OS root制限 | Wails HTTP/WSとHost認証・現Grant | Browser memory/Worker・明示した安全なメッセージ |
| Presentation/Session | TS CodeMirror、Focus、Dirty、Pin | 同じTS UI、tab/session競合 | 重い純粋Projectionは検証候補。DOM/IME/FocusはTSが既定 |

「状態・コンテキスト管理」のうち、**Semantic ContextのID/意味/Source occurrence/Result shape**は共通Go契約で扱える。一方、Client View Context（DOM/Focus/IME/Scroll/Dirty Buffer/Tab/Selection epoch）はTS Presentationに置き、Authority/Grant、実Provider Selection、Commit対象をBrowser/WASMに委譲しない。WASMのPreviewはHostの受理結果・実Read・実Grantへ昇格しない。

### 17.3 Plugin RoleとExecution Placementを直交させる

Language/Operator/Generator/Data/Presentationという役割と、native Go / server Go / browser Go-WASM / process / external Targetという配置を分ける。WASM追加を理由にCore固定CapabilityKind、第二Plugin Resolver、常駐Hostを増設しない。**Browserに配信されたWASMは計算用Artifactであり、ロード成功・Digest一致だけでProviderの業務実行権限や認可を得ない**。

最初の移植対象は、一つのSource-backed read-only projectionの純粋な部分（Inspector/三視点のOccurrence表示変換または既存Context Preview）だけとする。WASMへ全Application Operations・任意Native Provider・DB Provider・現在Grantを移植しない。TSで簡単に解ける小さなローカル状態は残し、Go native⇄WASMの意味忠実度と性能・再利用価値を測って採否を決める。

比較尺度：初回配信byteとload latency、再利用時の応答／heap、JS/WASM境界のmarshalling、WorkerでEditor main threadを阻害しないか、必要なwasm_exec.jsの一致、Browser/OSごとの欠損・partial・stale・Null/Emptyの保全、Build/Assetの増分再利用、不要Profileへの物理非包含。**同じGoで書けるから高速になるという主張はしない**。WASM欠如時はpure Previewをunsupported/Host read-only経路へ明示的に切替える候補を評価するが、権限拒否・Source不整合・Commit失敗にlocal fallbackしない。

### 17.4 横断受入候補（既存#1180/#2514のExpectedへ）

| Case | 検査と拒否条件 | 主なOwner |
| --- | --- | --- |
| WSP-01 Server Reference | 既存127.0.0.1 Wails Server実起動、CLI/Desktop/Serverの同一read-only OperationのSubject/Source/Failure対照。外部bind不可のPoCを維持 | #1824/#3421 |
| WSP-02 Service Surface | Wails実公開methodを棚卸しし、public Shared ServerでSHO Path・Command系に意図せぬ匿名／別Actor到達があれば拒否 | #1824/#1043 |
| WSP-03 Event Broadcast | 2 Workspace×2 Principal×2 Tabで全WS受信byteを比較し、他WorkspaceのSource/Prompt/Trace/Resultを許可無しに受け取らない。browser-NをPrincipal扱いしない | #3421/#1178/#2515 |
| WSP-04 Native/WASM | 一つのpure projectionをGo nativeとブラウザjs/wasmで比較。Subject/Occurrence/Span、missing/null/empty、partial/stale、集合順・精度・失敗分類を保全 | #1824/#1158/#2497 |
| WSP-05 Wasm Build Evidence | Go Toolchain＋wasm_exec.js＋main.wasm＋N2 JS Bundle＋実Browser読み込みByteを各digestで固定。未選択Profileへ不要Asset/Initを含めない | #2831/#2600/#3267 |
| WSP-06 Dirty/Authority Drift | WASM旧Preview後にSource/Grant/Provider Generationを変更。Browser側計算一致でもHost Applyで拒否・再Reviewを要求 | #1124/#1529/#1174 |
| WSP-07 Cost Comparison | 同Fixture/DeviceでTS/native/WASMの起動・描画応答・memory/配布sizeを測定。優位性がなければWASM導入保留 | #1824/#2018 |
| WSP-08 Product Scope | Studio Wails Server/Go-WASMの成功を生成業務Web App、第二Application Target #259、Stage2 Desktop #4583の成功へ流用しない | #259/#4583 |

これらWSPラベルは新Runner/Expected正本ではなく既存fixtureへ紐付ける比較票。

### 17.5 段階的実装順と停止線

**P0維持**：Studio Desktop #4583の正式BuildIntent→N2/N3→Stage1/Stage2/実Windows Consumerと、#4611/#4612のHost selected Provider観測。Wasm、Web共有IdP、Stateの全面Go化をHard Gateに加えない。

**P1（Wails Web共通PoC）**：#1824/#3421で現127.0.0.1 Serverのread-only実RPCとService Surface/Event隔離を検査し、Wails Server継続か独立の薄いHTTP Transportが必要かを実差分から判断。外部公開・複数Principal本番資格は与えない。

**P2（選択的WASM）**：#1824/#1158/#2497でGo nativeとBrowser JS/WASMを一pure projectionだけ比較、#2831/#2600へ実Artifact closureを渡す。性能/Bundle/意味上の利点がない場合はTS/nativeを継続する。

**P3（共有Webと異Target）**：#3421/#1223の多Principal/Generation、#1861のWIT比較、#259の第二生成Application Targetを独立に評価。全PluginのBrowser WASM化や無停止更新を当然視しない。

**検証上限**：公式ドキュメントと現mainの静的照合のみ。新Wails serverの公開認証、WASM build/runtime、実Browser/二Principal、Wails版更新、Owner受理はNOT RUN / NOT ACCEPTED。


### 17.6 Wails Server公開面の実コード監査・限定Goテスト（2026-11-14）

**mainの実物**：現 cmd/tei-studio/main.go は application.NewService(&StudioBridge{host: host}) を一つ登録しServer/Desktopで共用する。StudioBridgeのexported Go methodは**15件**であり、Workspace read/projection 8件、Command plan 1件、reference ExecuteCommand系4件、SHO external processのOpen/Resume 2件に分類できる。現在のService全体を「read-onlyだから全Methodをpublicに公開可能」と評価しない。Wails Serverの既存Origin/Host middlewareは主に /wails/runtime に限定され、/wails/events 等全経路への認証・Principal単位の認可を意味しない（現 cmd/tei-studio/README.md）。本PoCはloopback Listenerなので外部侵入・漏洩を実証したわけではない。

**公式仕様の追加確認**：[Wails Server Build](https://wails.io/guides/server-build/) の2026-11-13版は、/wails/runtimeのOrigin制約がある一方、curl等のOriginを送らない非Browser要求は他条件を満たせば処理されることを明記。Loopback Host checkもportまでは比較しない。したがって**Origin検査のみをPrincipal認証や現在Grantとして使わない**。Eventの全接続Browser向けbroadcastは仕様上の配送範囲であり、現Studioに機微イベント漏洩が発生した証拠ではない。

**実装候補を新規作成**：[独立Draft PR #4627](https://github.com/hat47x/tei-platform/pull/4627)は、最新mainから分けたcmd/tei-studio/server_bridge_surface_test.go一件（server build tag）。実main.go内のWails NewService登録をASTで照合し、reflect.TypeOf(&StudioBridge{})で実Go exported method set（別ファイル・埋込み昇格を含む）を列挙する。未分類のMethod、追加されたService、既知Method消失はレビューを要求してテストFail。**テストPassはpublic Web公開の安全性や実Binder/HTTP route・WS隔離を証明しない**。既知の非read/外部IO入口を NOT PUBLIC-QUALIFIED とログ出力する棚卸しであり、現Server/Desktopの実動作・Provider/意味・Authorityは変更しない。root go test ./... はnested Go moduleを対象外にするため、cd cmd/tei-studio && go test -tags server -run TestServerBridgeExportedMethodInventoryRequiresReview をexact HEADで実行し、PR/CIで独立検収する。

**WSP-02の分割**：WSP-02aは#4627のGo method/Service登録一覧の静的評価。WSP-02bは#1824/#3421の実Wails Server経由で正規・匿名・別Principal・異Workspace・Originなしの要求と実HTTP/WSを送る独立Expected。共有Web版候補ではread-onlyの実到達だけを許し、Command/SHO/外部Pathの無許可到達を拒否する。Wails専用read-only facadeと認証済み薄いHTTP Adapterの差は、実Service名・JS binding・Source/Host Context/ブラウザ受信で比較してから決める。新第二Business Operation正本・第二Resolverは不要。

**Desktop Build主線の新しいEvidence**：[Studio統合Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) exact HEAD 0fcb7b6265d7dfc5b5e93017e5c99e296eb13134 で、[Windows WebView2 Run #43542664470](https://github.com/hat47x/tei-platform/actions/runs/43542664470)を含む**12 Workflowがcompleted/success**。N2 esbuild実生成JS→N3 Go EmbedFiles選択memberを同一Attemptの実Disk byte hashでBuild前後照合し、改変コピーの負例を拒否した限定H1成功。Compiler read-timeの実byte捕捉・完全Toolchain closure・Productの非Plugin ArtifactSet・Stage1生成TEI→Stage2 Desktop・Owner受理は別資格で、旧HEADのCIと合算しない。

**停止線**：#4627はDraft、実Server公開・認証・Go/WASM/Browser/二Principalの新E2Eは今回NOT RUN。P0 #4583 B1-B/C→B2/B3を維持し、WSP-02a・02bを並行P1、WASM pure ProjectionはP2の比較とする。


### 17.7 #4627 exact HEAD の限定Go/Server/Browser CI実績

[Draft #4627](https://github.com/hat47x/tei-platform/pull/4627) HEAD 9491bf9e081bf98f39056d0a4142c2764c31289d の[Wails Host Workflow #43544229895](https://github.com/hat47x/tei-platform/actions/runs/43544229895)はcompleted/success。server-build Job ID 132291601976の実ログから、**CGO_ENABLED=0 go test -mod=readonly -count=1 -tags server . → ok tei/cmd/tei-studio** を確認した。新しいBridge公開面Inventory Testはこのserver tag付きGo Testに含まれる。Wails Server build/loopback RPCと、実Chromiumの二BrowserContexts・後着Catalogを分離する試験も同Job成功。同HEADの[Desktop WebView smoke #43544229876](https://github.com/hat47x/tei-platform/actions/runs/43544229876)、[Cache #43544229829](https://github.com/hat47x/tei-platform/actions/runs/43544229829)も成功。

**判定範囲を縮めず、拡げすぎない**：このPassは既存Source/Go Method Inventoryが意図した分類から漂流していないこと、現Loopback Browserの表示が旧応答で混じらないことの限定確認である。Web shared serverの別認証Principal・別Workspace実Grant、JS Method Binder全経路、WebSocketの機微payload分離、外部インターネット到達・Tool送出の拒否を確認していない。独立WSP-02b/03はNOT RUN。上記HEADの全Workflow完了状況は継続的に変わるため、この時点の確認では全Repository Health/依存検査まで合格と宣言しない。新実装の本番採用とOwner受理もなし。

## 18. 2026-11-14：ローカルPathをWeb権限に変換しない／独立Build資格を接続する

### 18.1 他チャットと実コードの交差：SHO・REAのローカルPath

[REAローカルCLI Draft #4625](https://github.com/hat47x/tei-platform/pull/4625)は、保存済みEvidenceの受領を既存operations.ExternalCandidatesから読むread-only入口を作った。evidence-pathは**local CLIの入力Path**であり、ReadFailedの内部Pathが外部Disclosure可能であると保証しない。exact HEAD d2175d97e11b6a6f9bc6b390ddae98c519a576fe は9 Workflowのうち8 success、[Repository Health #43543262432](https://github.com/hat47x/tei-platform/actions/runs/43543262432) failure。全体の受理資格を成功とはしない。 Health job 132288471320のログでgo vet/go test/Workspace validate/ADR indexはPASS、gofmt -l .のみFAILし、drift対象はcmd/tei/cli_flag_contract.goと特定した。R3-AのOwnerへ[PRコメント](https://github.com/hat47x/tei-platform/pull/4625#issuecomment-6870141370)で新HEADの整形・再検証を引渡した。

現mainのWails StudioBridge.OpenShoDocument(draftPath,documentPath)／ResumeShoDocument(documentPath)は、Host設定のSHO_DOC_COMMANDで外部子プロセスへ中継する。現在loopback-onlyなので「外部から任意Pathへ侵入可能」という実証ではないが、このMethodをそのままpublic multiuser Webへ公開してよい理由にもならない。

**共通化するのは意味Operationであり生File Pathの信頼ではない。** Local CLI/OS Hostの明示Pathと、Browser/MCP/AIが送る未信頼Pathを区別する。WebではHostの認証済みPrincipal＋Workspace/Resource IDから正規Resourceと現在Grantを解決し、許可したread-only Operationへ渡す。Browserが指定したraw Pathを直接OS read／SHO processへ渡さず、Remote Clientの失敗時にLocal権限へ黙ってfallbackしない。Source/Trace/Error・Pathの存在・DigestもDisclosure対象になり得る。受領後の返却/AI送出時点のAudienceを再検査する。第二Meaning Resolver/Authority/汎用File Serverを増やさない。

| 参照Case | 仕込み・拒否または受入条件 | Owner |
| --- | --- | --- |
| LRB-01 Raw Path | BrowserがWorkspace外／別WorkspaceのPathを渡す。OS読取・SHO子プロセス実Callは0、秘密PathはResponseに現れない | #3421/#1824/#1002 |
| LRB-02 REA Resource | local Evidence Pathの成功と、RemoteのPath拒否・Host受理済みResource IDのread-only成功を分離する | #3394/#3421 |
| LRB-03 Disclosure失効 | 取得中にPrincipalの開示資格を撤回。読取済Byteと実送出を分け、未許可Audienceへ新送出しない | #1174/#1178/#2515 |
| LRB-04 Error/Existence | not-found、permission-denied、partialを区別し、Raw Evidence/内部絶対PathをBrowser/AI Errorへ漏らさない | #1158/#1178 |
| LRB-05 Cross-Workspace | 同名Resourceや同Digestでも別Workspace/ActorのCache/Resultを流用して開示しない | #1016/#951/#1178 |
| LRB-06 SHO Child | Host設定があってもClient Pathで無承認子起動しない。Receipt successを業務完了に昇格しない | #1824/#3394/#1174 |
| LRB-07 Plugin世代 | Preview後無効化/A→B→Aでも旧Session/Provider結果から現在Grantを復活させない | #4378/#1223/#1174 |

LRBラベルは独立Expected正本ではなく、既存#1180/#2514へ渡す横断比較用。Wails実HTTPに対するWSP-02b（read-only正例＋Command/SHO/Raw Path拒否）を先に実施し、REA/SHOの実Resource bindingはその後に別Consumerで検査する。WebのIdP/HAをStudio DesktopのP0へ持ち込まない。

### 18.2 BuildIntent三層の独立した証拠

| Draft | 最新exact HEADと確認したCI | 証明した限定事項／未証明事項 |
| --- | --- | --- |
| [#4624](https://github.com/hat47x/tei-platform/pull/4624) | 2d403f4996c205514dc57d5fc1237127c6bd994b：5/5 Workflow success、[Health #43540612927](https://github.com/hat47x/tei-platform/actions/runs/43540612927) | Resolver応答が宣言Source/Builder/Dependencyと異なれば拒否。実Child起動は別 |
| [#4623](https://github.com/hat47x/tei-platform/pull/4623) | f1e219a25976946bcc8a19ef57f5490333ec8a84：8/8 success、[Health #43539856799](https://github.com/hat47x/tei-platform/actions/runs/43539856799) | 同一Attemptのselected Builder digest→起動前のChild digest照合。Loader read-timeは別 |
| [#4587](https://github.com/hat47x/tei-platform/pull/4587) | 0fcb7b6265d7dfc5b5e93017e5c99e296eb13134：12/12 success、[Windows #43542664470](https://github.com/hat47x/tei-platform/actions/runs/43542664470) | N2生成JS→N3 EmbedFiles selected memberの前後byte/改変コピー負例。非Plugin Desktop BuildIntent/Stage1→Stage2は別 |

これら3 PRは**別HEAD・別Attempt・別Subject**であり、合算してend-to-end成功としない。#4583主線は一つの実Attemptで宣言→Resolver→Builder起動前pin→N2 Produced→N3 selected/consumed→Stage2 Windows Consumerを既存BuildExecution/ArtifactSetへ束縛する。Desktop exeをPluginへ偽装しない。新Registry/Runner/Hostは作らない。

### 18.3 Go公開面監査テストのMutation強化

[Draft #4627](https://github.com/hat47x/tei-platform/pull/4627)の新HEAD f2f49a7c9732b3f88e8dc21c75cbdcc1084c0b70 は、新Service登録を単にソース全体のNewService呼出し数で数えるのではなく、**実application.Options.Servicesの設定表現**をASTで検査する変更を追加。未分類Service追加・同Bridge重複・未知Service・空一覧・動的変数一覧・Services欠落の6合成Mutationを検出する負例を追加。ExecuteCommandWithPolicyのClient allowed配列はHost Grantではないことを分類に明記した。静的登録試験であり、Wails実Binder/HTTP/WSや将来の動的登録の全閉包を証明しない。

**新HEAD f2f49a7c9732b3f88e8dc21c75cbdcc1084c0b70の実CIを確認**：[Wails Host #43545456013](https://github.com/hat47x/tei-platform/actions/runs/43545456013) completed/success。server-build Job 132295613443の実ログで CGO_ENABLED=0 go test -mod=readonly -count=1 -tags server . → ok tei/cmd/tei-studio 0.680s を確認。6 Mutation negativeを含む新Server Go試験はこの成功したスイートへ収録されている。同HEADのDesktop WebView smoke #43545456086、Cache、依存検査もcompleted/success。Repository Health #43545455990はこの照合時点ではin_progressであり、全Workflow合格や本番Web公開資格とはしない。旧HEAD 9491bf9eの成功と取り違えず、実Browser二Context表示分離≠別PrincipalのCurrent Grant/WS配信分離という資格上限を維持する。

**未実行**：LRBの実Web/SHO/REA/認可/AI送出、PR横断BuildとStage1→Stage2再入、Owner意味受理。本章は既存PlanのOwner分離と実受入条件へ引渡す設計成果。


## 19. 2026-11-14：外部Byte・Query Admission・UI Appearanceの独立資格

### 19.1 実装の進展と横断的な意味

異なる他チャットで進んだ3系統の変更は、一つの「Pluginの有効化」ではなく、次の独立した段階に位置付けられる。

| 新しい実装候補 | 実際の作用 | 別に必要な資格 |
| --- | --- | --- |
| [REA Host-owned Bytes #4630](https://github.com/hat47x/tei-platform/pull/4630) | ops.ExternalCandidatesFromBytes がHost調達済みraw Byteから既存R2 Candidateを投影。EvidencePathが非空なら拒否 | Host認可済みResource選択・bounded取得、Current Disclosure、実Consumer Reply。意味受理は別 |
| [Product Queryの型付きInput #4626](https://github.com/hat47x/tei-platform/pull/4626)＋[Driver pre-Executor検査 #4631](https://github.com/hat47x/tei-platform/pull/4631) | Sourceに実Query.Input from/std.datetimeを宣言し、実Provider前に参数の必須/型/未宣言などを拒否する限定実装候補 | 全型一般化・SQL placeholder全閉包、実DB/通常CLI、現在Grant/Effect。RA-1はRA-0へ積上げており単独受理しない |
| [Studio Appearance #4628](https://github.com/hat47x/tei-platform/pull/4628) | TS/CSSのlight/dark/system・densityを局所保存し、選択/Editor/Dirtyの維持を目標とする実装Draft | Plugin選択済・Workspace設定・Current Grantではない。実WebView・アクセシビリティ・長期採否は別 |

**同じ「State」という名称でまとめない**。Theme/densityのローカル保存はPresentation Preference、Plugin Managerのenabled/settingsVersionは#4378で未決の管理状態、Data ContextのActive Input/Pinは#2597/#2500、Current Grantは#1174、Provider Selectionは#951/#1016であり、どれもSource Accepted Meaningの正本と同義ではない。

### 19.2 Product/API/Hostが持つ五つの時点

1. **Discover**：型付きOperation/Toolが実装されているか・CLI/API/MCP/Studio/Webどの入口で呼べるかを観測。存在は権限や成功でない（#1158/#1251）。
2. **Select/Acquire**：正しいWorkspace/Resource/Source Revision/Provider/Contractを選び、必要なByte/Query Parameterを取得。local PathをRemote Pathへ拡張しない（#1016/#3394）。
3. **Admit**：解析用ByteならPathを持たない既存R2へ渡す。DB Queryなら宣言済みParameterとの型適合とCurrent Grant/実対象をExecutor/Effect前に検証し、不適合時はactual Executor Calls 0。legacy Sourceで宣言が空というだけでtyped-qualifiedとはしない（#4631/#1174）。
4. **Invoke/Observe**：選択済みProviderの実Entry、実受領Byte、効果/Receipt/partial/Unknownと独立Readを別々に観測。Reference実行やSource Revision一致から業務完了を導かない（#1251/#1174）。
5. **Deliver**：Result/Failure/Tool/Trace/AI TaskのうちAudienceが今受け取れるFieldだけを送る。実送出Byte・現在Disclosureを別検査。Modelに渡せたか/外部Consumerが受領したか/Skillが公示できるかも別である（#1178/#2515）。

この五時点は**既存OperationやEvidence票の観測Projection**であり、新しい中央Server/Operation Schema/Global Authorization Gateway/Runnerの採用ではない。特にClientにあるUI Theme変更がPlan/Grantを変更したり、Browser WASMの同値計算が実ProviderのAdmitを代行したりしない。

### 19.3 Byte-onlyの実コード上の開示負例

#4630 HEAD 1ad749fc0eaacc195dd28ea88a97bd44d9581050 の実コードでは、ExternalCandidatesFromBytesがOSファイルReadをしない一方、ExternalCandidateView.Attempt.Toolはexternalanalysis.Toolをそのまま含む。同型にはjson公開のResolvedPath、AbsentReasonがある。ResultのEnvelope/Producer Error/Reasonも送信先に応じて開示検査が必要。**Pathless InputとPathless Outputは別資格**である。実漏えいを観測したとは主張しない。

既存#1178のDisclosure Ownerに[具体的受入コメント](https://github.com/hat47x/tei-platform/issues/1178#issuecomment-6870458945)、#4630の実装Ownerに[同じレビュー](https://github.com/hat47x/tei-platform/pull/4630#issuecomment-6870443976)を引渡し。Host Raw Pathに合成canary、Producer ErrorにRaw marker、同名別Workspace Evidence、Read後のDisclosure失効を用意し、**Receiverが実際に受け取ったHTTP/WS/MCP/AI Tool Byte**に未許可markerがないことを測る。許可済みread-only正例が通ることも同時に測る。

### 19.4 Source変更による旧Evidenceの再利用を明示的に拒否

#4626では recent_orders に input from std.datetime を追加した結果、旧R0 Source Revisionを期待する受入Fixtureが現在の新Sourceと一致しなくなった。新しいModel Revision／World Snapshotの適格性を、過去のR0の実績から移さないため、Scratch Copyで**旧byteを元に戻した過去世界**の再現性と、現在のRA-0 Source/型付き入力の資格を独立に持つ。既存E1のバイナリ依存検査でsqlgen symbolが検出された失敗は、Source Input宣言と同一原因と証明されておらず、allowlist緩和で合格にしない。

#4631では、Source Inputが空の旧Reference直呼出しは歴史挙動を残すが、現Environmentで未宣言の実Parameterが渡された場合はExecutor呼出し前に拒否する積上げ設計。**旧Reference PASSを現在の型付きProvider資格としない**。これを新しい第二Driver/Rule Engineにはせず、既存driver query executionのCallerが実使用したことを独立Expectedへ残す。

### 19.5 今回のexact HEAD・CI・Owner分担

- [REA local CLI #4625](https://github.com/hat47x/tei-platform/pull/4625) HEAD 24863bd69a9139e1a75459a7c6607514dbc73ad8：確認した9 Workflowは全てcompleted/success（Health #43545393255）。旧HEAD d2175d97のgofmt失敗と混同しない。R3-A local入口限定でRemote Path/Host認可・第二Consumerは未実装。
- [REA byte-only #4630](https://github.com/hat47x/tei-platform/pull/4630) HEAD 1ad749fc0eaacc195dd28ea88a97bd44d9581050：確認時点、REA contract/Healthを含む11 Workflow success、Windows smokeはqueued。新Go test成功は、外部Browser/AI DeliveryとOwner意味受理を示さない。
- [Studio Server Inventory #4627](https://github.com/hat47x/tei-platform/pull/4627) 旧HEAD f2f49a7c：nested Server Go TestとWindows smoke等4 Workflow successだがHealth #43545455990は**本PR追加server_bridge_surface_test.goのgofmt不一致でFAIL**。新HEAD a085004721629f2e8d2141aac8a1fb7f07507d7b でそのGo map整形を修正。新HEADのHealth終了を確認する前にPASSと記さない。
- [Studio appearance #4628](https://github.com/hat47x/tei-platform/pull/4628) HEAD 4be527f2cc2d12a2d135eb0e9f32743bfa379352 は実CSS/JSとSmokeを含むDraft。CIはこの照合時点で複数実行中なので、外観のProduct/実Windows全資格は未判定。
- [RA-1 pre-executor #4631](https://github.com/hat47x/tei-platform/pull/4631) HEAD f0db7bb07d59edd30e60fb8cc0e20a7eb066e369 は#4626上の積上げDraft。新HEADの実Go driver/実PG全面PASSは未確認。ValidationやEndpoint到達から実DB Contract適格性を推測しない。

**優先度は維持**：P0 #4583 Stage1→Stage2 Desktop BuildIntent/ArtifactSet実結合。P1は#4627のmain統合可能なGo test健全性と#4630のHost-owned bytes/Disclosureのread-only Consumerを並行確認。P2は型付きQuery Inputの実Provider前拒否、Studio Web Auth/REA二Consumerの外部Consumer比較。P3でUI Themeの広範囲適合、Browser WASMの選択的実測、共有Webの共同編集/HAを個別に評価。これらは他PRの採用や新Hard Gateの創設ではない。


## 20. 2026-11-14：段階別ContributionとDesktop実行配置の合流

### 20.1 将来の製品能力を現在のPoCから切り離して設計する

[理想設計Draft #4632](https://github.com/hat47x/tei-platform/pull/4632)は、Pluginを一つの実行関数の登録とせず、**意味定義→Studio編集UI→検査・変換→Build→生成アプリのUI/サービス→試験・運用**の段階別Contributionとして記述する案である。Package/Contract/Contribution/Variant/Artifact/Configuration/Resolutionの各概念を分け、利用者の標準Profileから始め、必要な機能の設計・生成・実行・テストを同じ選択に沿って展開する。

**現在の実行規律と理想アーキテクチャの自由度は両立する。** 現行#951/#1016/#1158を再利用してP0/P1を進めるが、現PoCのRegistry型、固定Go API、ディレクトリ配置、単一のPlugin主語BuildPlanを将来の標準Schemaに昇格しない。新規Conceptの採用前には既存Contractの意味・Owner受理・Test Expected・Source provenanceを維持できるかを比較し、拡張／Adapter／置換を選ぶ。全面置換は可能な比較候補であり、無条件の全面作り直しではない。

Contributionを複数箇所に提供しても一つの実行資格にまとめない。例えばDBのStudio Query Editor、Source Model、Query検証、Generator、生成AppのProvider、実PG read、Consumer Replyは**同じ契約と版を参照する別Artifact/Role/Effect/Authority**。Studioだけ新版でProductが旧版、選択Bで実Call A、未選択PluginのAsset収録、誤ったSource Revision再利用を、独立Expectedで検出する。

### 20.2 二つの軸と四つのScopeを固定する比較票

| 独立軸 | 選択の対象 | 選択から自動的には生じないもの |
| --- | --- | --- |
| **Contribution（何を提供するか）** | Language/Schema・Studio UI・Operator/Validation・Generator/Builder・Runtime Provider・Test/Operations等 | すべての段階への自動適用や実行許可 |
| **Execution Placement（どこで動くか）** | Go静的結合・Desktop in-process・同一exe従属Worker・別Process/Server・Web/Browser Wasm・外部Target Host | 選択されたProvider・契約適合・同じOS権限・ABI互換 |
| **Scope（誰/何に適用するか）** | Server/Device Inventory、Workspace/Profile、Window/Document/Editor Session、Product Application/Invocation | Server導入済＝全Workspace選択済、Window表示済＝全Principal開示許可 |
| **Version/Authority（いつ有効か）** | Package/Contract/Source/Model/Toolchain/Artifact/Settings/Host Generation、Current Grant | byte一致・同じ名前・旧Release成功による現在受理 |

実装ではこれらを一つの万能Registry/単一Version文字列へまとめない。Profile Resolve、Artifact Build、Actual Invocation、Current Grant、Deliveryを既存Ownerの独立責務として結ぶ。将来Contribution Registryの型を改める場合にも**既存契約の成功・失敗・不可知結果の独立Expected**を保持する。

### 20.3 Desktop Host配置は三案を測ってから選ぶ

他チャットで提案された「DesktopのWebViewは薄く、ローカルServer/WorkerでGoを共通化する」案を、次の比較としてP1に置く。

| 案 | 実行構成 | 利点の候補 | 不利/未確定と実検査 |
| --- | --- | --- | --- |
| **A 現行** | Wails Desktop Window→同一Process Go Bridge→型付きApplication Operation | 起動とローカルCallが単純。配布1実体。IPC marshalling不要 | UI/Goの故障境界の共有、Web Hostとの差分 |
| **B 従属Worker** | Desktop GUI→Named Pipe(Windows)/UDS(Unix)→self-spawnした同一exe Worker | Go業務Coreの実Process隔離、Worker監視・再起動を分けられる | 同一exe二Roleの起動／停止・双方向protocol・ACL・OS user・child identity。**Wails server buildとは別構成** |
| **C 別Server** | Desktop GUI→従属Wails server Binary→loopback HTTP/WS | Web向けBridge/Frontend reuseの余地 | 二Binary/二Build Artifact・Port/Origin/WS/Audience・配布/更新/故障/起動費用。Web共有Hostの認証とは別検収 |

Go/WailsのUIをDesktopとWebで共有できても、通信Protocol以外の差は消えない。Service lifecycle、secret/OS Permission、Plugin実LoaderとAvailability、Process exit中のEffectUnknown、実Source/Provider Generation、Tool stdout/AI送出、更新・Rollbackを案別に比較する。Native in-processはNetwork IPCと同じコストではないため、Named PipeがHTTPより軽い可能性を理由にAより速いと断定しない。

**従属Workerは生成業務ApplicationへTEI独自Hostを義務付ける判断ではない。** Developer-facing Studioの配置候補に留め、#259のTarget/Host差と切り離す。既存Wails PoCはAを基準とし、Cのloopback serverは現Public Web資格なしのPoCのまま比較する。

### 20.4 LPC受入と実測表（#1824/#1158/#4583へ）

| Case | 独立Expectedの最小観測 | 不合格または未確定 |
| --- | --- | --- |
| LPC-01 Actual Worker | selected Build/Role→実起動Process exe SHA/args/Version→実Operation Response | self-spawn説明やBinary存在のみを実Call証拠にしない |
| LPC-02 Local IPC Authority | Windows Pipe ACL/remote access制限、Unix Socket directory/mode、別OS User・他Workspace拒否 | loopback・same-machineという理由だけでTrusted Principal扱いしない |
| LPC-03 Crash/Orphan | GUI/Worker各々を異常終了→実行中Task/Source Draftが失われない範囲、再接続・Process cleanup | 旧Replyを新Sessionへ無条件受理しない |
| LPC-04 Effect Unknown | Commit途中の切断、再起動時のoperation identity/receipt lookup-first | 通信再接続時の自動reinvoke/重複外部副作用を拒否 |
| LPC-05 Generation/ABA | A→B→A PluginやSource移動・内容復帰中、旧Worker Session/Grantを拒否 | 同Digestのみで新世代受理しない |
| LPC-06 Disclosure | Worker logs、Progress、WS、外部Tool/AI Sinkの全送信byte | Browser/Window/Socket識別子を権限と推定しない |
| LPC-07 同一仕事の性能 | 実Windows及びUnixでA/B/C同一Fixtureの起動、p50/p95/p99、CPU/RSS、編集中の入力停止、Preview latency | 仮の性能値や異なるOperationを比較しない |
| LPC-08 Build/配布 | A/B/CのN0〜N3における選択Artifact/Toolchain/Dependency/実Consumed byte | CのServer BinaryをDesktop一バイナリ実績へ加算しない |

これらLPCは新Runnerや新Authority正本ではなく#1180/#2514の独立Expectedへ引渡す。P0 #4583は**A現行構成の一Attempt Desktopセルフビルド**を先に完成させる。B/Cの評価はP1独立比較であり、Worker未実装を理由にP0停止しない。

### 20.5 生成成果物とStudioを結ぶ最小縦断実証

#4632の理想構想は、Core Schemaを先に全面固定せず、既存のorder_entry/recent_ordersを用いた**小さなDB Provider縦断**で圧力試験する。SourceのQuery Input from/std.datetime（#4626）、Driver呼出し前の型検査（stacked #4631）、実Environment/PG Provider（#2586）、Studio Query view、Target Generator/Artifact、実CLI/業務Consumerを同じVersion/Subjectへ関連付ける。

- **Design**：StudioのProvider選択・Query編集とSource ID/Revision/Occurrenceを保存。Style/Theme（#4628）は意味Definitionと独立する。
- **Build**：生成対象に必要なContributionとArtifactだけを含む。未選択Provider/Studio-only Fixture/EditorはRuntimeへ含めない。Builderの宣言と解決（#4624）、actual child pin（#4623）、N2→N3 handoff（#4587）を一Attemptへ結ぶ。非Plugin Desktopを偽のPlugin扱いしない。
- **Run**：選択B/実Call Bを実Provider Entryで確認。invalid/missing/null/extra datetimeはExecutor Call 0。現PGからのReadはHost Current Grant/DB Instance/Source Revisionを別観測。
- **Consume**：UI/CLI/APIで同じOperation意味と異なるAdapter失敗分類を比較。旧Source Fixtureの過去合格、New Source/DB/Current Grantを別票にする。
- **Change/Disable**：Plugin A→B→A、DB/Themeの変更、Scope違い、追加/削除、古いPreview、モデル改版による旧Expected失効を検査。Client EditorのDirty/IME/Focusは基本的に保持し、Host Effectと混同しない。

この縦断が成功する前に、全Pluginの高度Dependency Scheduler/自動置換を先に導入しない。DB・認証・組織・テストの四系統を完全合成する#4632の長期資格は、P0の硬い依存にはしない。

### 20.6 実装済みResult Contract preflightを正しく位置づける

[PR #4629](https://github.com/hat47x/tei-platform/pull/4629)は**Closed/Merged**（HEAD 00ea85234720db01a06279fd87e2c72cf1001605、同HEAD 9 Workflow success）。internal/semantic/capabilityinvokeでMalformed Result ContractをAdapter Call前に拒否し、6つの不正契約に対するCall 0を検証した。今後のScript Launch/Plugin A1でもこの実装を使い、別Result Validatorを増やさない。

ただしResult Contract形状の正常性は、Entry公開・Owner意味受理・実Selected Provider・現在Granted Scope・外部Effect/Receipt・Target Windows Consumerを証明しない。前段のpreflightを、公開Host側の実Actor/Resource Authorityや後段のDisclosure検査にすり替えない。

### 20.7 現時点のCI資格と非干渉

- [Wails Server公開面テストDraft #4627](https://github.com/hat47x/tei-platform/pull/4627)：前HEAD f2f49a7cおよびa0850047のRepository Healthが、追加Goテストのgofmt driftで**failure**と実Jobで確認された。新HEAD b9cf5447bcd3881aec06d97e4f3060f7de80b331 で原因の余分な空行を除去。最新HEADの新CIは確認時点でqueued/in_progressであり、旧HEADのHost/Windows Go成功を合算せず**Healthは再検収待ち**。
- [REA bytes Draft #4630](https://github.com/hat47x/tei-platform/pull/4630)：HEAD 1ad749fc0eaacc195dd28ea88a97bd44d9581050、12/12 Workflow completed/success。Windows Smokeは**Attempt 1でPowerShell OS情報取得Timeout失敗→該当Job再実行Attempt 2でsuccess**。成功したのは一般Studio Windows Smokeであり、REA→Studio実Consumerの受領ではない。Attempt間の履歴を保持。
- [Studio Appearance #4628](https://github.com/hat47x/tei-platform/pull/4628)、[RA-1 stacked #4631](https://github.com/hat47x/tei-platform/pull/4631)、[Plugin理想設計 #4632](https://github.com/hat47x/tei-platform/pull/4632)はいずれもDraftであり、設計案・局所Go/Browser試験を全Target受理や正式PoC置換へ昇格させない。RA-1はRA-0 #4626の先行受理後に合流可否を判定する。

**変更範囲**：本書の追加のみ。#4632と#4620が編集する plan/MASTER_PLAN.md、#4632の理想設計原本、#4587の実Windows Buildなど他Ownerの並行ファイルは編集しない。理想設計は選択肢として保持し、#951/#1016/#1158で今使える操作をP0に接続する。Go/IPC/OS Process/P99比較・共有Web認証・Studio→Generated App新DB E2E・Owner受理は本節でNOT RUN。


## 21. 2026-11-14：分散した合格を製品の一往復へ接続する統合順序

### 21.1 「個別技術の検証成功」から「実際の使用」までの段階

同日並行開発の各PRは、実装の一部を同一HEADで試験する水準へ達した。ここからは、**実行段階の境界を越えた実際の消費・現在権限・依存PRの統合順序**を独立した受入票にして、成功数の合算や「登録済み＝実利用済み」という判断を避ける。

| 現在の成果 | 最新の確認結果 | 次に必要な製品上の接続 |
| --- | --- | --- |
| [Wails Server Inventory #4627](https://github.com/hat47x/tei-platform/pull/4627) | HEAD b9cf5447bcd3881aec06d97e4f3060f7de80b331 の**5/5 Workflow completed/success**。前2 HEADのgofmt失敗を解消。現Service登録と15公開Methodの静的監査、6 Mutation Case | WSP-02b/03：実Wails HTTP/WS、正規/匿名/別Principal/別Workspace/異AudienceのReceiver別拒否。現loopbackをpublic Web適格としない |
| [Self-build合流 #4634](https://github.com/hat47x/tei-platform/pull/4634) | 初回統合HEAD 653d0d9a1d0125631aec4379550d2bce20364207 と現HEAD 7d6a1238e6efbd83fb2bd4cfda9667c181567920 は、**各HEAD独立に8/8 Workflow completed/success**。宣言↔Resolver一致＋選択Builder↔prestart Child digestを同一main-based HEADで検証 | Product非Plugin DesktopのBuildExecution/ArtifactSet、N0〜N3・Stage1生成→Stage2 Windows Consumerまで**同一Attempt**で接続。別PR #4587のN2→N3 Byte結果を自動合算しない |
| [REA local CLI #4625](https://github.com/hat47x/tei-platform/pull/4625) → [byte-only #4630](https://github.com/hat47x/tei-platform/pull/4630) → [Host selector #4637](https://github.com/hat47x/tei-platform/pull/4637) | #4625は9/9、#4630は12/12（Windowsは再実行Attempt 2で成功）、#4637 HEAD 170f9be5907cea4af981582ab4c11c5dafb91e87 は**12/12 success**。Host固定Selector→4MiB bounded Source read→同R2 ProjectionのGo実装 | #4630を先に統合、#4637の重複byte-only差分を除いた新HEADで再検収。Studio実Consumer/Delivery Receipt、現在Principal/Grant/Disclosure、別Consumerを独立検証 |
| [Capability Result Preflight #4629](https://github.com/hat47x/tei-platform/pull/4629) → [Input/Effect Preflight #4636](https://github.com/hat47x/tei-platform/pull/4636) | #4629はmainへ**merged**、Result不正6例のAdapter call0。#4636 HEAD da807abc65d560134e9a6aa4c4f7b8d92d3b2fef は**9/9 Workflow successを経てmainへmerged**、不正Input宣言/Effect宣言・nil contextをGo Adapter call前に拒否 | L-P0受理済み公開Entry→L-P1 Host選択の実Provider Go call→L-P2 CLI subprocess。実入力値、Nullable、Current Grant、実Effect receiptは未受理 |
| [RA-0 Query Source #4626](https://github.com/hat47x/tei-platform/pull/4626) → [RA-1 Driver #4631](https://github.com/hat47x/tei-platform/pull/4631) | #4631 HEAD 02af8339e3a08665f3db69173ebc1bedea4c7967 の**stacked 専用Go Workflow #43548961899 success**。datetime不適合時Executor call0を実検証 | #4626 Sourceの先行受理、実隔離PG、Target/通常tei CLI二Consumer、Current Grantを別資格にする。E1 symbol除外失敗は新Source型と原因を混同しない |

これらの「success」は、各PRの提示した正負Fixtureで対象コードが動いたという限定評価。**Draft/未mergeであればmain採用ではない。** Test Runnerで確認したGo/Browser/Windowsの対象と、実際の業務Consumerが同じかを別途示す。

### 21.2 REAで初めて成立したHostへの具体的な接続と残る境界

[PR #4637](https://github.com/hat47x/tei-platform/pull/4637)の実コードを読むと、Trusted Hostが起動時に固定する LocalREABinding は Workspace ID、Selector、正規化済み相対File、Tool Version、Evidence ID、Raw SHA-256、任意Source Revisionからなる。実Callerは `Read(workspaceID, selector, observationID, relationID)` を呼び、**任意Path/Root/ReadAtを指定できない**。

Readerは既存ReadOnly.workspace()のRoot実体照合を行い、workspace.ReadSourceで4MiB上限付きのByteを一度取得して、既存ExternalCandidatesFromBytesへ渡す。異Workspace、未知Selector、過大・変更Raw、Root差替え、symlink脱出、古いPinを拒否するGoテストを持つ。失敗時は低位Pathや秘密Relation名を出さず共通OPERATION_FAILEDとし、成功時もTool.ResolvedPath/AbsentReasonを空にしてからProjectionする。

**残る差を消さない**：このReaderはstudiohost内のローカルGo APIで、Wails Bridge/HTTP/MCPへ登録していない。成功ViewそのものにはEvidence ID/Raw Digest/Source range・Producer Coverage等が含まれ得る。よってLocal内部メタ情報の簡易除去に成功しても、認証済み実Actor・現Workspace Grant・ConsumerごとのDisclosureは未成立。A→B→AのSource/Permission変更の中間履歴は同SHA pinだけで捕捉できず、#1124/#4503のSource/Reader epochと現AuthZを用いる。

**次のR3-B最短Product Slice**：
1. 既存Hostの固定Workspace/Resource Selectorを、ClientはPathでなくSelector IDとして要求。HostがRoot/Source/Byte/Grant現在性を検査する。
2. 既存read-only Typed Operationで同ByteのCandidateを投影。Raw ReceiptとAnalysisのUnselected/Unknownは失わず、成功をMeaning Acceptanceへ格上げしない。
3. 外部送出直前にAudience別の開示可能Fieldを再判定し、Result/Failure/WS/AI Sinkの実送信ByteとDelivery Receiptを検証。
4. 異質の第二Consumerで**独立のInvocationと受領**を確認。CLI・Hostが同一内部関数を呼ぶだけでは二Consumer適格性の完全な証明ではない。

これらをREAの独立Expectedへ渡し、Web一般公開・R3-C・Owner意味受理を別ゲートとする。PluginコードをLoaderへ登録しただけで業務Sourceを変更してよい権限は生じない。

### 21.3 BuildとStudio Hostの成果を正しい順で統合する

- #4634は#4623と#4624のGo変更を一main-based HEADへ集約した候補である。元2本をそれぞれmergeしてから#4634をさらに適用するのではなく、**#4634を統合単位としてOwnerが差分・受理基準を確認**し、元PRは参照履歴として扱う。別PR #4587が生成WorkbenchのN2→N3 Byte handoffと実Windows WebViewを持つため、これらを同じBuild Attempt/NonPlugin Subject/Targetへ接続する次実装を#4583で担当する。
- #4627は5/5 CI成功により、**静的Service Inventoryガードとしてレビュー可能**になった。Web公開を許可する入口ではない。Wails Serverで明示したread-only Facadeの実Endpoint/Actor/Workspaceと、Event broadcastのAudienceによる拒否は#3421で別資格を取得する。
- #4636は、#4629のmain統合済みResult検査を再利用する入力・Effect宣言の構造preflightを実装し、同HEADの9/9 CIを経て**mainへ統合済み**。Capability Input/Effect宣言の構造不備をAdapter Call=0で止める局所防御は成立。実入力値の型/NULL/時間条件やAuthorityが未確定でも一般Result検査を厳しくしすぎない。L-P0→L-P1→L-P2の一往復を既存Script Launch #4495/Plan #4618へ委譲する。
- Studio外観設定 #4628のPreferenceはClient UI状態として維持し、#4378のPlugin Configuration/Workspace Lockと混同しない。理想Contribution #4632を現PoCの実Loaderへ自動昇格せず、既存解決・実Call・Artifact/Targetをもとに移行案を比較する。

### 21.4 追加する統合受入Case（新独立Runnerではない）

| Case | 実際に拒否/保存する観測 | Owner |
| --- | --- | --- |
| HND-01 Source→Build Identity | 宣言と異なるResolver返答/選択B実Aを子呼出し前に拒否し、正常時は実ChildとReceiptを記録。同一HEADだけでは非Plugin Desktop完了としない | #4583/#2831 |
| HND-02 Stage2 Consumer | 実N2 Bundle→Go Embed選択member→実Stage2生成exe→そのexeのWindows WebView実受領を同一AttemptのArtifactSetで追跡 | #4583/#1824 |
| HND-03 Host Bounded Evidence | CallerのPath/ReadAt指定不可、Host固定Selectorの実Raw digest/4MiB限界/Root差替え/Unknownと失敗分類を実観測 | #3394/#4503 |
| HND-04 Disclosure at Delivery | Reader成功後、別Actor/Workspace/AI SinkにTool Path/Source range/存在情報を渡さないことをReceiver実Byteで検証 | #1178/#2515/#3421 |
| HND-05 Capability Preflight vs Use | 不正Input/Effect/Result宣言はAdapter Call0。正規Owner公開Entry/選択Providerを実Callし、EffectUnknownを成功にしない | #4495/#1174 |
| HND-06 Query/Evidence Revision | 歴史R0/旧REA Evidenceと現在Source/Query Input・Provider Revisionを混同せず、A→B→Aおよび同名異対象の再資格を要求 | #1124/#2586/#3394 |
| HND-07 Scope/Contribution | Studio Editor、選択Target Generator、実Provider、Runtime Asset、Test Fixtureを別Artifactで観測し、未選択Pluginの本番混入を拒否 | #951/#1016/#4632 |

**P0は非Plugin Studio Desktopの完全セルフビルドが主線**。REAのHost接続、Script Launch Preflight、Web Service Static Inventoryは並行する限定検収であり、P0へ新しい必須依存を追加しない。生成アプリ/Studio双方を統一する理想Plugin設計は引き続き比較案である。

### 21.5 PR統合の明確な前提と新CIの扱い

依存の順序：**#4630 → #4637**（#4637はmain比較の差分に#4630由来2ファイルを含むため、前者統合後に再base/差分削減、新HEADで再検収）、**#4626 → #4631**（Source型宣言先行、積上げGo Test合格と実PGは別）、**#4629 merged → #4636 merged（現在main実装）**、**#4623/#4624 → #4634を一統合単位としてレビュー**、**#4627は独立して限定static guardをレビュー**。これはmerge実行指示ではなく、重複導入・古い基準との照合漏れを避ける受入順序。

観測の履歴を保持：#4630のWindows SmokeはAttempt 1で環境Timeout・Attempt 2成功。#4627の初期HEADはgofmt不一致でHealth失敗・修正HEAD b9cf5447で5/5成功。失敗履歴を消して初回から成功したとは記述しない。PR更新でHEADが変わった場合、前HEADの成功を新HEADへ流用しない。

**資格上限**：この章は各GitHub PRの現HEAD/CI実績と実コード・既存Ownerの受入条件を照合した実装引渡しであり、当章追加による新Go Test・実Web AuthZ/WS/OS IPC/実DB・Stage1→Stage2 Build・Owner正式受理はNOT RUN。並行PRが編集するMASTER_PLAN.md/理想設計原本を重複変更しない。

### 21.6 2026-11-14 追補：Script Launchの宣言preflightがmainに統合

[PR #4636](https://github.com/hat47x/tei-platform/pull/4636)はHEAD da807abc65d560134e9a6aa4c4f7b8d92d3b2fef（9/9 GitHub Workflow completed/success）からmainへ**merged**。merge commit 18470e69d2755b29ce55285ff847305e2e3e420d を確認。これはInput名・型参照/重複、DataEffect宣言、nil contextという**静的宣言のpreflight→実Adapter Call=0**を現実にmainの共通Capability Invokeへ追加する変更である。前段#4629のResult Contract preflightと同じ汎用境界であり、新第二Validatorは作らない。

**製品資格の停止線を維持**：元のInput値・意味Type/NULL/単位、公開EntryをHostが実選択したこと、Current Principal/Grant、実Provider Go call、Commit Receipt/EffectUnknown、通常tei subprocessはmainへのマージだけでは成立しない。#4495/#4618のL-P0→L-P1→L-P2をOwnerへ委譲し、Plugin一段階の契約構造適格をStudio/Generated App全面受理へ拡張しない。§20.7は当時のDraft状態の履歴記述、現在の実装状態は本追補を正とする。


### 21.7 N0→N2の実選択と統合Build PRの新HEAD（2026-11-14継続）

[Desktop Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)は、従来のH1「N2 produced JS→N3 Go Embed実Member」に加えて、**H0「N0で生成したInspector TypeScript→N2 esbuildで実選択されたSource」**を実装した。N0 self-build生成物のrole/byte長/SHA-256/相対Path/HEAD/Attempt/Invoker receiptを固定し、N2起動直前の実Disk byteと、N2出力metafile.inputsが選んだSource memberを照合する。変更コピー、誤Role、誤Pathを拒否する3負例も含む。

最初のH0 HEAD 641b19eb89b1bfbea45553448436616075135577の[Windows CI #43553869203](https://github.com/hat47x/tei-platform/actions/runs/43553869203)では、H0 Producerと実N2 esbuild選択のローカル証拠が同DigestでPASS、実選択Source 24件を確認した。一方、生成Windows Job全体は**後続N3 EvidenceのPowerShell重複keyでFAIL**。修正HEAD [357345d73b3ab74d67ec8bda675fcd63768e9c48](https://github.com/hat47x/tei-platform/commit/357345d73b3ab74d67ec8bda675fcd63768e9c48)で重複したesbuildSelectedCountを整理。**新HEADのWindows WebView2 Workflow #43554075677はcompleted/failure**。実ジョブを分けて確認すると、generated-desktop-webview2-smoke（Job 132322840114）はcompleted/successで、実ログにN0→N2同Digest照合、N2→N3 Embedの実照合、SMOKE_RESULT=PASS_NATIVE_WEBVIEW2_EDITOR_KEYBOARD_AND_OBSERVEXEADが出ている。一方、通常desktop-webview2-smoke（Job 132322839843）はpowershell.exe による OSVersion.VersionString の15秒Timeoutで失敗し、通常版UI試験の成功は得ていない。失敗は生成H0のDigest誤照合と同定されていないが、Workflow全体はFAILである。同HEADのほか11 Workflowはcompleted/success。初回H0部分PASSと修正後のgenerated Job成功を、同HEAD Workflow全体PASSへ昇格しない。

[Builder合流 Draft #4634](https://github.com/hat47x/tei-platform/pull/4634)も、Go6ファイルと当時のMASTER_PLANを含むHEAD 653d0d9a1d0125631aec4379550d2bce20364207の8/8 CI successを得た後、主計画のH0/H1最新状態を更新した**新HEAD 7d6a1238e6efbd83fb2bd4cfda9667c181567920**へ進んだ。**新HEAD 7d6a1238自身の8 Workflowもcompleted/successを確認**。前HEADの成功を流用せず新Attemptを根拠にする。

**製品P0の最短差分をさらに限定**：H0のN0生成Byte/選択Role/実N2 member → H1のN2 JS Byte/実N3 Embed member → 非Plugin Studioの正式BuildIntent/BuildExecution/ArtifactSet → 生成Stage1→Stage2の実Windows WebView2を、同一Subject/HEAD/Run/Attemptに結ぶ。成功は各境界で発生した実Consumed/Delivered byteにより判定し、Compiler/esbuildの完全read-time byte捕捉は未実証とする。CLI/N0と生成UIの単なる文字列一致、WorkflowがGreenであること、旧Sourceに対するWindows受入を同一視しない。

**統合順の注意**：#4634の新HEAD／#4587修正HEADのCIを独立に確認してから、両者を同じAttemptで接続する後続実装へ進む。二PRを手動でByteコピーして「全工程受理済み」としない。P1のWails従属Worker/HTTP/WASM選定をP0の必須条件へ追加しない。


## 22. 2026-11-14：非Pluginセルフビルドの実製品ゲートと配布資格の分離

### 22.1 最新コードと他チャットを照合した実装上の欠落

[Generator契約Draft #4641](https://github.com/hat47x/tei-platform/pull/4641)の実コード調査によれば、既存BuildIntentはGoコマンドRecipeを持つものの、BuildIntent.ValidateのSubject.Kindはpluginを前提とし、Resolverもplugin.BuildPlan.Pluginへ格納する。このため**Goコマンドを実行できることと、Studio本体というnon-Plugin Subjectを正式BuildExecution/ArtifactSetへ対応付けられることは別資格**である。

[Builder統合Draft #4634](https://github.com/hat47x/tei-platform/pull/4634)の実Builder選択・起動前pinを、Desktop #4587のH0（N0生成Inspector TS→N2実esbuild選択）およびH1（N2生成JS→N3実Go埋込み選択）と接続するには、まず**正式non-Plugin Studio Subjectと同一Attempt**を既存BuildExecution/ArtifactSetへ結ぶ。Studioを架空のPluginに見せる方法、新Builder Registry、別Runner、新Hostは採らない。対象と成果物の対応だけを既存#1080/#1097/#2831/#4583 Ownerが最小変更で設計する。

[Generator #4641](https://github.com/hat47x/tei-platform/pull/4641)が指摘するもう一つの条件は、論理Graphを最適化した際に物理Stepと1対1でなくても、**要求された意味・NULL/欠損・順序・丸め・Effect・Source→Artifactの対応が保たれること**である。グラフに描画されないだけのSourceを「無い」と判断しない。一般Optimizerや第二GeneratorをP0へ導入せず、意味保存の負例を既存Expectedへ委譲する。

### 22.2 同日新HEADの技術実績と停止線

| 実装 | exact HEAD / CI | 次の未受入事項 |
| --- | --- | --- |
| #4634 Builder合流 | 7d6a1238e6efbd83fb2bd4cfda9667c181567920：**8/8 Workflow success**（旧HEAD 653d0d9とは別Attempt） | plugin主体のResolver/Child pinからnon-Plugin Studioの実Build Subjectと実Artifactへ接続 |
| #4587 H0/H1 | 357345d73b3ab74d67ec8bda675fcd63768e9c48：**12件中11 Workflow success、Windows #43554075677はfailure** | Windows内のgenerated Job 132322840114はH0/H1 Byte照合＋実WebView2 success。ordinary Job 132322839843はOS VersionStringのPowerShell 15秒Timeoutでfailure。Workflow全体成功・通常版成功ではない |
| #4627 Wails監査 | b9cf5447bcd3881aec06d97e4f3060f7de80b331：**5/5 Workflow success** | 静的なService/15 Method監査のみ。Webの認証・異Workspace・WS Event開示は別 |
| #4637 REA Host Reader | 旧HEAD 170f9be5907cea4af981582ab4c11c5dafb91e87：**12/12 Workflow success**。**新HEAD f9e355d5ccdc96bf4a25ff939fca444a18de1942 は別OS子ProcessのGo試験を追加し、CIは進行中** | 4MiB以内のHost固定Selector実Byte読取。Wails/HTTPへ公開済みではなくCurrent Grant/Deliveryは未検証 |
| #4636 Script Input/Effect Preflight | da807abc65d560134e9a6aa4c4f7b8d92d3b2fef：**9/9 success後main merged** | 宣言不整合のAdapter Call=0まで。実Provider/Owner受理/Effect/CLIは別 |
| #4639 Script Launch計画 | 5507150353f4da0ca0b008687b610191c275e551：**main merged** | 既存契約の計画反映であって新Provider実行ではない |

失敗履歴、WorkflowとJob、HEADとAttemptを保持し、異なるHEADの成功数を加算して製品完成としない。

### 22.3 Desktop P0は四つの独立した受入条件に絞る

| Case | 実Positiveと負例 | 受入Owner |
| --- | --- | --- |
| P0-A 正式Build対象 | non-Plugin Studio Subject→宣言→実Resolver→selected Builder→prestart実Child image→Receiptを同一Attemptで観測。仮Pluginへの偽装・選択B/実A・宣言外依存は子起動前拒否 | #1080/#1097/#4583 |
| P0-B 実Source/Member | N0生成TSのN2 selected Input、N2生成JSのN3 selected Embedをrole/path/digest/HEAD/Attemptで対照。誤Role/Path/別Attempt/旧Sourceの一致を拒否し、read-time完全閉包は未証明として保持 | #4583/#2831 |
| P0-C Stage2 Consumer | Stage1実ProcessがStage2を生成したReceipt、実Stage2 exe・BundleのDigest、生成したStage2自身によるWindows WebView2実受領を照合。既存exe直接起動をSelfBuildと認定しない | #4583/#1824 |
| P0-D Windows失敗分類 | 同一HEADでgenerated/ordinary JobとOS前提診断を別報告。PowerShell版情報Timeoutを意味/Provider失敗と混同せず、診断の回避だけで未実行のGUI検査まで成功扱いしない | #4583/#1824 |

これらは新しい中央受理エンジンの契約ではなく、既存BuildExecution/ArtifactSetと#1180/#2514の独立Expectedへ渡す実行ケースである。P1のWails別Worker/HTTP/WASM方式比較はP0をブロックしない。

### 22.4 外部Registry/ReleaseはBuildの次に別資格として扱う

[Registry Draft #4640](https://github.com/hat47x/tei-platform/pull/4640)は、main統合済みE1-Cのtest-only HTTP Signed Release/Member試験から、実GitHub Releases等への公開・独立Readbackへ移るときの区別を提案する。実GitHub Release発行、外部Asset取得、Immutable Release設定・公開承認は**未実証**であり、現在のDesktop BuildのHard Gateにしない。

Build Artifact Digestの一致だけでは、Publisher Identity、外部Store/ReleaseのMember、HTTP 200/302 Redirect時の認可転送、実Body Digest、Consumer Readback、現在Trust、Private Evidence Disclosureを証明しない。配布が必要になった時点で#2589/#2604/#1105の既存Ownerが別Attemptで公開→外部Provider→Consumer取得を検収する。第二Publisher/Store/Authorityは作らない。

### 22.5 プラグインと製品計画への接続

**P0**：non-Plugin Studio Build SubjectとH0/H1、Stage2実Consumerの接続を最優先。  
**並行P1**：REAは#4630→#4637の依存順で先行Byte APIを統合してから重複差分を整理し、新HEADで検収。Wailsは#4627の静的ガードと#3421の実HTTP/WS認可を区別する。  
**P2以降**：#4641の論理/物理Graph意味保存、#4640の外部Release Readback、#4632の段階別Contribution管理を独立Expectedと実Consumerの必要性に従って評価する。

主計画 plan/MASTER_PLAN.md は#4620/#4634の並行Ownerが更新する。本書では対応する検証条件とexact HEAD事実だけを保持し、重複した主計画の編集・新Core Schema採用を行わない。**本節による新Go/Windows/実Provider/配布試験とOwner意味受理はNOT RUN/NOT ACCEPTED。**

### 22.6 REAの新実行境界は製品Consumerと区別する（同日追跡）

[REA Host Reader Draft #4637](https://github.com/hat47x/tei-platform/pull/4637)は、旧HEAD 170f9be5 の12/12 success後、新HEAD f9e355d5ccdc96bf4a25ff939fca444a18de1942 へ進んだ。追加された local_rea_reader_process_test.go は**Goテスト自身を別OS子Processで起動**し、その子がHost固定Binding→実Evidence Read→型付きProjectionを行ったPID・Evidence ID/Digest/Observation/Directed Relation/ReadAtを親へ戻して検証する。未許可Selector/他Workspace/変更Rawで秘密Metadataが子Receiptへ出ない負例も含む。

これはプロセスをまたいだread-only投影の検証を一段深めるが、**製品のStudio UI/Wails Bridge/MCP/別Transport Consumerが実際に受信したこととは異なる**。別OS Processという実行形態だけで異質Consumerや現在Principal/Grantを証明したと扱わない。新HEADのGo/REA/Health/Windows全CIは今回の照合時点で未完了（既知成功数を継承せず）。#4630先行merge→#4637重複差分のrebase/再検収、R3-Cの実Consumer Delivery/Audienceと独立Expectedという停止線は変えない。


## 23. 2026-11-14：main再基線とRA-2実PostgreSQL検収、直近のCI停止線

### 23.1 計画の正本とGitHub統合順

前Draft [#4621](https://github.com/hat47x/tei-platform/pull/4621) は現mainの進行後にmergeable=falseとなり、設計本文自体の採用ではなく**Git統合が止まっている状態**。本PRは最新mainから新規branchを作り、承継する§1〜22の文書を**同一Pathに新設**し、最新のRA-2/Build/REA実行証拠だけを追補する。README索引はmainの現byteから必要な1行だけ追加する。MASTER_PLANは並行[#4620](https://github.com/hat47x/tei-platform/pull/4620)/[#4634](https://github.com/hat47x/tei-platform/pull/4634)が所有するため触れず、旧#4621は履歴・レビュー原本として保持し、新PRの採用時に重複PRとして片付ける。**本書の正本はGitHubでOwner受理後にmainへ入る一ファイル**であり、Draftを実装済みと見なさない。

### 23.2 RA-2の実DB成功はReference限定で保持する

[RA-2 stacked Draft #4638](https://github.com/hat47x/tei-platform/pull/4638) HEAD `0ce3a52f070314741f72462982ff940a1f791a28` の専用[PostgreSQL 16 Workflow #43553849861](https://github.com/hat47x/tei-platform/actions/runs/43553849861)はcompleted/success。**親＋7 subtest（A/B/C/D/E/G/F）の省略・SKIP・FAILが0**。使い捨て2DBに明示DSN・隔離同意を設定し、型付きfrom・境界・0件・別DB・不正値・NULL/欠損Column・JOIN増殖・RLS/Masking・独立Readを確認した。すべて`tools/reference-browse-read`の実Readであり、**通常製品`tei` CLI・正式Function Context・Native二Consumer、Current Principal GrantやOwner意味受理ではない**。依存順は#4626 RA-0 → #4631 RA-1 → #4638 RA-2を維持し、後続RA-3でProductの実呼出しを別検収する。

### 23.3 RA-0新Source改版のCI停止線を具体化

[RA-0 Draft #4626](https://github.com/hat47x/tei-platform/pull/4626) HEAD `94876821f784d5fa9f71b15a05ac4156221b2bc8` は18 Workflow中16 success、**Repository Health #43556757857**と**Browser→JSON API専用Binary #43556757879**がfailure。
- Health Job 132331320052の実ログに`examples/data-source-workbench/query_route_separation_test.go:6:2: "reflect" imported and not used`とあり、Go vetとGo testのBuildが止まっている。単にSource宣言が変わったための「期待どおりの意味不一致」ではなく、**Goコンパイル衛生を直して同HEADの新試行で再検収すべき具体的失敗**。
- Browser→APIでは、`go tool nm`で`tei/internal/database/sqlgen` 2 symbolを検出した既存E1のBinary依存排除ゲートがfail。Queryの型宣言・Go import変更が因果であると断定せず、#1182で実import graph/Link Closureを調査。**依存allowlist拡大/ゲート無効化でPASSを作らない**。
- Current Sourceの`input from std.datetime`、`reference/world.json`の新Golden、既存Runnerの歴史Frozen Worktree内だけの旧Query byte復元を分ける。旧Worldの再現に成功しても、現SourceのOwner意味受理や新Provider Grantへ昇格しない。#4631/#4638の旧base CIをRA-0新HEADへ自動継承しない。

### 23.4 REA実子ProcessはTransport Consumerではない

[REA Studio local Host Draft #4637](https://github.com/hat47x/tei-platform/pull/4637) 新HEAD `f9e355d5ccdc96bf4a25ff939fca444a18de1942` は、別OS子ProcessでHost固定Selector→bounded Byte→R2 ProjectionのReceiptを親へ返すtestを追加した。専用REA契約Jobは成功しているが、**Repository Health #43556801954がfailure**。Job 132331388978の実ログに追加`internal/studiohost/local_rea_reader_process_test.go`の`gofmt` driftを確認。他の技術Workflowが成功しても新HEAD全体のPASSにはならない。整形修正と新HEADのGo/Healthを確認するまで、旧HEAD `170f9be5`の12/12を新HEADの受理へ転用しない。

この子Processはテスト実行環境であり、**実Studio/Wails Browser/AI/MCPが別Consumerとして許可済みEvidenceを受領した実証ではない**。次のProduct Exitは#4630 Byte API先行merge→#4637重複差分の解消→実Selected Workspace/Actor/Disclosure/Receipt→異質Consumer受信の順。公開Webにlocal raw Pathを渡す方式は許可しない。

### 23.5 次の実装を小さくし、独立Expectedを保つ

P0-A：既存BuildIntentの`Subject.Kind=plugin`制約に対して、非Plugin Studioの正式SubjectとBuildExecution/ArtifactSetを対応付ける。実Go Recipe存在を形式資格とみなさず、Plugin偽装を拒否。P0-B/C：H0/H1 Source MemberとStage1実Child→Stage2 Windows実Consumerを同一Attemptで検証。P0-D：Windows OS情報取得のTimeoutを、WebView2実機能失敗とは独立に扱う。

並行P1：REA Host Reader/Consumer、Wails公開面の実HTTP/WS Current Grant、Query RA-0の衛生・Binary依存失敗の修正。P2：RA-2から正式Product Provider/通常CLI二Consumerへの接続と実Release Asset公開→別Consumer取得。理想段階別Contribution、生成器論理/物理Graph最適化の汎用化、Web共有配備は未受理の設計比較に留める。

**本章の追加はDocs-only**。ここに記載した既存CIは各PRの実行結果であり、本書の追記による新Go/PG/Windows/HTTP/OS子Process実験、コード修正、Owner正式意味受理は行っていない。


### 23.6 並行担当による修正HEADへの追随（履歴を上書きしない）

§23.3の失敗HEAD後、[RA-0 #4626](https://github.com/hat47x/tei-platform/pull/4626)は**新HEAD 9eecf2b932345861b1b5d28c22c2191fb164e5eb**へ更新された。前HEADのunused reflectは修正対象として扱い、新HEADの[Health #43558268996](https://github.com/hat47x/tei-platform/actions/runs/43558268996)は照合時点in_progress。いっぽうBrowser→APIの[実依存Workflow #43558269132](https://github.com/hat47x/tei-platform/actions/runs/43558269132)は**新HEADでもcompleted/failure**。Byte Source型宣言と別のE1実API Link Closureとして#1182 Ownerの判定が必要。未修正依存を許容することでGreenにしない。

§23.4の失敗HEAD後、[REA #4637](https://github.com/hat47x/tei-platform/pull/4637)は整形修正後の**新HEAD f38117dc69b722e14de36a69c3fae6478766d5d4**へ進んだ。旧HEAD f9e355d5のHealth失敗は追加Go testのstruct fieldsにおけるgofmtずれ。新HEADの[Health #43558330965](https://github.com/hat47x/tei-platform/actions/runs/43558330965)・REA専用Jobは照合時点queuedであり、旧12/12成功を新HEADに移さない。

この節は同日の**観測履歴**であり、競合するGoファイルを本PRから書き換えない。修正完了と当該HEADの独立CI成功、Ownerレビューの3つが揃って初めて対応するProduct Claimを更新する。


## 24. 2026-11-14：実データ被覆・Toolchain・段階別Contributionの受入結合

### 24.1 最新の3つの独立実装を、同じ意味モデルの異なる辺へ置く

今回の横断成果は、**選択された機能が正しいこと、実際に取得したデータの範囲を正しく述べること、生成した成果物が何に依存したかを正しく述べること**を、それぞれ別の資格とすることで結び付く。

| 実装 | exact HEAD と実CI | 示せること | 示せないこと |
| --- | --- | --- | --- |
| [Contribution Reference #4646](https://github.com/hat47x/tei-platform/pull/4646) | `77463edb17642d6400202d0229f33ec69db23b90`、Health/Cache/依存検査 **3/3 completed/success**。本人のGo/race/vet試験と区別 | 意味ID・版・指紋を各段階のContract・Target・Scope・Provider・Artifactへ対応付け、12種の不整合等をReferenceで拒否 | 実Pluginロード、GUI/API/生成Productでの利用、全契約の意味適合、現Grant、配布Trust |
| [Function Data Context W1〜W3 #4647](https://github.com/hat47x/tei-platform/pull/4647) | `a962a3f6b13c58d93ad740998e5d94728c53e88d`、**7/7 Workflow completed/success**。実PostgreSQL16の詳細はPRで提示されたWSL上の別実測 | native SQL Readの実取得条件をHost/実DB側で観測。RLS/LIMIT/未宣言関係等**見えた制限**を`partial`として運ぶ | 未観測の絞込みがないこと、Native SQLの全母集団`complete`、結果列の由来W4、現在Grant、通常製品tei/Studioでの実呼出し |
| [Self-build Toolchain観測 #4648](https://github.com/hat47x/tei-platform/pull/4648) | `64fb0f9abdf1a8cf947a844a973eb6d48685623f`、**8/8 Workflow completed/success**。Go 1.26.11/1.27.0の相互試験はPR記載のLinux別実測 | 同じPlanでも使う**コンパイルToolchain**が変われば成果物Byteが異なる。Builder申告＋Entry側の申告File再読取をReportへ保持 | Builderが実際にそのToolchainを起動したこと、Seed Trust/G4、Windows/macOS/arm64での再現、Toolchainの契約上Pin |

**三つの階層を併合しない**。Contributionの選抜が成功しても、実Query被覆や成果物のToolchainを証明しない。PGの実Readに成功しても、`complete`・現在AuthZ・GUIからの受領を推定しない。同一Planで二つのToolchainによる`l2b.claimed=true`が成立しても、Seed TrustやStage1→Stage2 Studioを認定しない。

### 24.2 Queryの取得被覆：`partial`と`unestablished`を別の事実として保存

[W1〜W3 #4647](https://github.com/hat47x/tei-platform/pull/4647)の実コード方針は、PostgreSQLの宣言関係を`ACCESS SHARE`で先にロックし、system identifier、実引数つき`EXPLAIN (VERBOSE)`の関係/Node、関係種別、`row_security_active`を観測する。失敗した追加観測はsavepointで隔離して、本来のRead失敗を上書きしない。Persistence Mappingから実Dataの宣言関係・PKを選ぶが、0件・複数候補なら選ばない。SQL本文から推測しない。

確認される制限は`partial`（例：`limit-in-plan`、`row-security-active`、`relation-not-declared`、`append-or-partitioned`）。何も制限が見えなかった場合も`complete`へ上げず、`unestablished`と`open: native-sql-predicate-not-closed`を維持する。STABLE SQL関数に隠れた述語が、同じPlanの関係・RLS・LIMITには現れない反例を保持。**欠落が観測されないことは、欠落しないことの証拠ではない**。native SQLの`EmptyJustified=true`はここでは許可しない。

`NULL`の保存元の由来・列OID/列番号、Entity粒度を確定するW4は未実装。`nullOrigin=unestablished`を正とし、宣言PKが一意らしいことだけで`stored`/Entity/Full Scopeへ昇格しない。RLSは現在Session Roleで変わり、system identifier・同じQuery text・同じrowsだけではTenantとAuthorityを示さない。取得条件・観測時点・実Recipientをそれぞれ独立Expectedへ残す。

[同一HEADのRA-0/1/2再資格 #4644](https://github.com/hat47x/tei-platform/pull/4644) `b5c23044ebaf8671d73a140da92fc9c4d74edff4` はRA-1 Go／歴史WebとRA-2実PG16の専用CIなど**3/3 Workflow success**。これは旧RA-0と旧RA-1/2を一つの仮統合HEADで試験できた前進であり、**製品Operation→通常tei CLI→実Provider→Function Context二Consumerは未受理**。#4626 base `9eecf2b9`はHealth/歴史Self-build等成功でもBrowser→API実Binary依存除外の`sqlgen`2 symbolが別ゲートでfailure。型付きSourceの受理やPGの7ケースにE1依存ゲートの失敗を混ぜず、#1182の調査を継続する。

**横断Product Exit**：RA-2Rの`reference-browse-read`からRA-2Pへ移行し、既存FunctionContextへW1〜W3の被覆注記を伝える。Readの成功/失敗、観測の成立度、Consumerへの開示とOwnerの正式なMeaning/Source受理を四つの独立判定にする。W4の列由来、構造化Queryでの`complete`採用、MappingのPK型とSeed値の不一致は#2586 Ownerが別に決める。Function Contextを第二Data Authorityにしない。

### 24.3 Go Toolchainを現時点では「Build時に観測した事実」として保持

[#4648](https://github.com/hat47x/tei-platform/pull/4648)のX=Go1.26.11／Y=Go1.27.0相互試験は、駆動役の種をX/Yへ変えても**コンパイルToolchainが同じなら同じArtifact Byte**、Toolchainを変えると**同一PlanでもArtifactが異なる**ことを示した。現`l2b.claimed`の五条件を両方が満たすので、既存のPlan FingerprintとArtifact Revisionは同じ概念ではない。

新しい`toolchainObservation`はbuilder申告の実行ファイルとByte DigestをEntryがもう一度読み直し、`matched`/`mismatched`/`unreported`を出す。**matchedは申告パスのByte一致であり、実OS Execの観測ではない**。`unreported`も現コードではBuild拒否条件でない。BuilderがToolchainを読めず`notObserved`とした場合は、完了したBuildと「コンパイラの特定不可」を別票で残す。周辺Toolchain Source全体のHashが性能へ与える影響も未計測。

採否判断を分割する。(T1) ReportのProvenanceとして現実に消費したToolchainの資格をどう測るか、(T2) BuildPlanへToolchain Pinを**意図として要求するか**、(T3) Generator/Builder/Compiler/OS実Childの依存閉包とFingerprint再使用条件、(T4) Seed Trust/G4を外部の独立系統でどう検証するか。T1のReport実装成功でT2〜T4を受理しない。

P0 Studioの非Plugin Subject連結時は、`selected Builder`と`actual Child image`だけでなく、`actual compiler toolchain`の観測値またはunreported理由をAttempt/ArtifactSetへ紐付けるよう**受入票を拡張**する。ただし現行Planを無断変更しない。異なるToolchainでもArtifact Digest一致を必須化したり、同Digestだから同じMeaning受理として再使用したりしない。#2831/#1119がPlan入力化のOwnerである。

### 24.4 Pluginの段階別Contributionを理想設計ではなくReferenceで選抜検証する

[#4646](https://github.com/hat47x/tei-platform/pull/4646)は既存#951 Referenceに、**共通MeaningのIdentity/Version/Fingerprint**と、Editor/Generator/Runtime/Test等のContribution固有Contract・Package Version・Target・Scope・Provider/Artifactを切り分ける小さな構成検証を追加した。同じMeaningへ異なるPackage版を対応付けられること、曖昧なProviderや明示Wiring、実行Runtime Contributionが無い場合、Editorだけの要求を独立検査する。

ここでの「有効な構成」は**論理候補の計画**であって、実Plugin Providerがロードされたこと、正しいRuntime Binary/Targetへ包含されたこと、実Productで呼ばれたこと、Current Grant/Disclosureが許可したことではない。

次の受入順は、(C1) Meaning/Target/Scope/Contribution契約の**宣言選抜**、(C2) 実BuildIntent/Target Artifactの**実包含**、(C3) 実OS Child/Runtime Providerの**実Binding**、(C4) UI/CLI/APIの異質Consumerへの**実Deliver**、(C5) Owner受理・Current Grant・版変更A→B→Aの再資格、の順。既存#951/#1016/#1251/#2831/#1180に分割し、#4632の理想管理Schemaを新しい必須Coreに先行導入しない。Referenceが12種のMismatchを検出したことだけで実行時Permissionが保証されたとは言わない。

### 24.5 採用する統合ケースと優先順位

| Case | 独立Expectedの実観測と拒否条件 | Owner/優先 |
| --- | --- | --- |
| XC-01 Toolchain差分 | 同一Plan/Sourceで異Compiler→Artifact Byte差、Entryの`matched`と実OS Exec同定を別判定。実使われたToolchainが不明なら`unreported`として残す | #2831/#1119、**P0補助** |
| XC-02 non-Plugin Self-build | 正式Studio Subject→Builder/Toolchain/Child→N0/N2/N3→Stage1が生成したStage2→実Windows WebView2を同一Attemptに束縛。Stage0の既存exe直接起動・Plugin偽装を拒否 | #4583/#2831、**P0本線** |
| XC-03 SQL Coverage | 実RLS/Limit/別Relationを`partial`、隠れ関数述語を`unestablished/open`。Queryが0件でも`EmptyJustified`へ上げない | #2586/#3509、**P1** |
| XC-04 RA-2P Product Read | RA-0→1→2Rの仮統合成功から、正式Product `tei` CLI→実Provider→同Context二Consumerを独立受領し、Current Grantを別検証 | #2586/#4495、**P1** |
| XC-05 Contribution actual use | 同一MeaningのEditor・Generator・Runtimeを独立に選択し、未選択Target混入、選択B実A、旧Artifact/Scopeを検出。実Consumerを確認 | #951/#1016/#1251、**P1/P2** |
| XC-06 Data provenance and Disclosure | RLSにより見える行・Source存在・Tool metadata・原因ErrorをAudience別に返却直前判定し、実受領Byteで拒否を確認 | #1174/#1178/#3421、**P1/P2** |

**主線変更なし**：P0はTEI Studio Desktopのnon-Plugin正式Build SubjectからStage2 Consumerまで。RA-2P・FunctionContext被覆・REA Host Reader・実HTTP認可とContribution実用検収は並行。外部Release Readback／理想全Plugin Manager／WASM／Web共同編集／Seed Trust G4は、個別Ownerが別の必要性・実Evidenceを示すまでP0を止めない。

**今回の作業範囲**：本書の受入・設計追補のみ（Go/PG/Windowsの新たな試験はNOT RUN）。#4644/#4646/#4647/#4648は**それぞれDraft・未main統合**で、各HEADのCI成功を本書のRuntime/Product E2E合格に移さない。main統合済み[#4645](https://github.com/hat47x/tei-platform/pull/4645)のScript Launch A0〜A4設計は既存Ownerの作業分解として活用し、ここから新Dispatcher/Run Engineを増やさない。
