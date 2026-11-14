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
