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
