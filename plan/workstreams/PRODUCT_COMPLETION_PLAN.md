# 理想の利用像を製品の受入へつなぐ計画

- この計画は設計書の読書きから実行結果の確認、分野別の標準構成、導入・変更・引継ぎまでを対象とする。
- 作業と評価の入口は [#3796](https://github.com/hat47x/tei-platform/issues/3796)。G01〜G71はこの計画の引渡しを追うための項目IDである。G01〜G64を引き継ぎ、案件の一巡に必要なG65〜G71を加える。正式な製品要求や独立した試験番号は既存の正本を使う。
- 実装方法は[製品設計](../design/PRODUCT_COMPLETION_DESIGN.md)、着手順の基準は[開発全体計画](../MASTER_PLAN.md)、現行の実証は[実装状況](../../docs/status.md)で確認する。

## 完成と判断する単位

一つの案件で、必要な設計を人が読み、許された範囲を変更し、選んだ実装で実行し、利用側の結果を独立に確認できることを判定する。案件に必要な導入、失敗時対応、更新、引継ぎも受入に含める。

受注では確定した値、公開では利用側から読める版、外部連携では相手側で観測した状態を確認する。受付、コード生成、単体試験、参照実装の成功は、それぞれ確認した範囲として保持する。案件の必須条件を満たした結果と結び付けてから成立に数える。

### 統合後、次の実装で確かめること（2026年11月14日）

**横断の受入境界**：Source/意味/生成Artifact/実Provider/Consumerの各PASSを同一製品成功に合成する条件、変更後の選択的失効、Graph/Ruleから実コードへ至る最小試験、Plugin自己構築率・定義簡素化の測定は[横断統合受入境界](../design/CROSS_CUTTING_ACCEPTANCE_BOUNDARIES_2026-11-12.md)を参照する。この参照は下表の優先順・各Owner・実行済み証拠の資格を変更しない。

[横断統合 #4683](https://github.com/hat47x/tei-platform/pull/4683)と[実対象の回帰試験統合 #4684](https://github.com/hat47x/tei-platform/pull/4684)によって、以前の計画・一部の生成コード・安全性試験はmainに入りました。ただし、**統合できたコード、通常の製品入口から動いた機能、実際に利用先が受け取った結果**は別々の成果です。以前のDraftの状態をそのまま現在の実装状況として読まないため、直近の着手先を以下に固定します。

| 優先順位と着手先 | 最初に通す実装・検証 | 合格と呼ぶ前に残すもの |
| --- | --- | --- |
| **全体P0：Studioの実セルフビルド**（#2831／#4583） | 非PluginのStudio Build Subjectを既存BuildExecution／ArtifactSetへ結び、**Stage1の実ProcessがStage2 Desktopを生成**し、そのStage2をWindows WebView2で起動する | [Windows部品試験 #4587](https://github.com/hat47x/tei-platform/pull/4587)の成功だけではStage2再構築とみなさない。実Child、Builder、Toolchain、生成物のByteと受領先を同一の試行で照合 |
| **並行P0：通常製品入口の実Read**（#2586／#1251） | RA-0／1／2RのReference結果を入口の基線とし、RA-2Pで通常`tei`→選択Provider／実DB／SQL・型付き引数→Function Data Contextを実行。RA-3でTableとAggregate等の異なる二Consumerへ渡す | [RA-0 Draft #4626](https://github.com/hat47x/tei-platform/pull/4626)のE1 `sqlgen`依存拒否（#1182）を残し、参照PGの成功やContextの`partial`を実Consumer・全件取得の成功に読み替えない |
| **CLI先行の小実証：TEI自身による生成**（#2621／#951／#2831） | 既存の[CLI先行計画 #4676](https://github.com/hat47x/tei-platform/pull/4676)を使い、同名Node二つ・別IDの多重有向EdgeとRuleを、Sourceから**実際の判定ロジック**へ生成。向き変更／ID重複／レイアウトのみの変更を対照にする | `*.tei.md`の正式Readerが未対応なら`unsupported`。Goへの委譲だけを自己実装と数えない。選択Pluginの実Call、生成コードの由来、固定した義務の分母、再生成・保守負荷を別記録 |
| **P1：Plugin説明と公開モデルの探索**（[#4679](https://github.com/hat47x/tei-platform/issues/4679)／#951） | 上位A・利用側Bの二Pluginを固定し、Aが公開したContract／Graph／Manual索引へ、Bのread-only Inspectから正しい版で辿る。短いProduct Skillは既存の実コマンドの発見方法だけを案内 | 非公開文書・図・上位依存の存在情報を漏らさない。閲覧可を再公開可・実行可へ昇格させない。新Docs Registry／未実装CLIコマンドを前提にしない |
| **P1：横断Concernと専門計算の適合**（#4198／#3509／#4660） | main統合済みの[Application Graph設計 #4681](https://github.com/hat47x/tei-platform/pull/4681)では、まず一つのRoute／Entryに対するread-only適用表を作る。計算側は[SQLite Window実DB試験 #4670](https://github.com/hat47x/tei-platform/pull/4670)を基線に、必要な入力範囲とProvider能力を調べる | 計測の挿入・同意・二重発火、外部Solver、NoSQLでの同等性、通常CLIの実計算・Consumer受領は未検証。新Aspect Weaverや第二Query Engineを導入しない |

### Web表示の共通契約を、現在のInspector一件で検査する（2026年11月16日、P1）

[Web標準UI横断設計 §§18〜19](../design/WEB_STANDARD_COMPONENT_AND_AGENT_HARNESS_2026-11.md)で、表示技術の比較に先立って **WEB-P1a：検証済みInspectionからread-only ViewModelを導出する** 最小実装を定めた。出発点は現行 `reference/tei-studio-workbench/main.ts` の選択Event→Catalog資格→`responses.run("inspection")`→実Host Inspect→`readInspection`→表示である。TypeScriptの純粋なPresenterと既存テストだけを先に変更し、Wails Bridgeの追加、Source Readerの再実装、Litへの置換、業務Meaningの新正本を同時に行わない。

**WEB-P1aの合格条件**は、同一Subject／Catalog／Host応答から表示文書・版・Source状況・利用制限が変わらず導出され、異Kind・Source Index版不一致・Reader error・部分取得・偽Grantを誤って合格へ変換しないこと。**WEB-P1b**はFocus／Pinの応答laneとEditorの未保存状態を保護する。Source位置の `source-locations.ts` は既存のread-only検証を再利用するが、画面表示と原文再取得・書戻しの資格を分ける。Node試験、Browser、実Windows WebView2、生成Stage2の受入は別票で記録する。

**WEB-P1a実装差分と配布への停止線**：既存`readInspection`を変更せず、別ファイル`inspector-presentation.ts`に表示専用の`presentInspection`と局所試験を追加した（WEB-P1a-1）。現段階ではWorkbenchの`main.ts`から未接続。`presentation.ts`を更新すると生成PoC Bが固定した歴史的な依存ClosureのGit blobに一致せず、Windows CIが`workbench_closure_revision_mismatch`で停止することを確認した。この失効を回避するために旧Pinを上書きせず、今回の差分は独立ファイルへ切り出す。WEB-P1a-2は新Closureを明示的に資格化し、Node24試験／型検査／esbuild、正規JS／HTML／CSS／第三者通知、Wails埋込Bundleのbyte一致、実Consumerを別々に検証する。Focus／Pin、Source実Open、公開Web、Stage1→Stage2は独立した受入とする。

**WEB-P1a-2の実装準備と検証の区切り**：[Web横断設計 §20](../design/WEB_STANDARD_COMPONENT_AND_AGENT_HARNESS_2026-11.md)で、前回PR #5173のCI事実（埋込Bundle・Windows Smoke成功／Host・全体健全性の一部失敗）を資格別に整理し、旧`WORKBENCH_NATIVE_CLOSURE`を保護したまま新Presenterを実画面へ接続する手順を確定した。まず既存Readerと独立Presenterの組合せに、同名異Kind・別Workspace・同版異Subject・明示NULL・partial／unfetchedを検査する局所回帰を加える。画面接続は別PRとし、Node24、実esbuild、Wails埋込JS／HTML／CSS／第三者通知のbyte一致、Static Compositionの新閉包、通常／生成Consumerを同HEADで再資格化する。局所Presenter試験では新Workbenchの配布をPASSとしない。#5168のFocus／Pin独立Expectedを再利用し、内部Inspectorの本文をHTTP公開する際はAudience／Disclosureが別途受理されるまでは転送しない。

**WEB-P1a-2aの具体的な検収**：[Web横断設計 §22](../design/WEB_STANDARD_COMPONENT_AND_AGENT_HARNESS_2026-11.md)に基づき、過去のPinを改変せず、`inspector-candidate.mjs`が元`main.ts`・Reader・独立PresenterのGit blobを一致検査した後、Catalogの適格性検査と表示を一体化した`readInspectorPresentation`を成功時の経路へ接続した別Sourceを導出する。出力の上書きは禁止し、入力改版・固定Closure不整合を拒否する。Workbench埋込CIでは正規Bundleの一致検査を残したまま、候補の型検査・別esbuild・実依存入力・旧埋込JSとの差分を確認する。ここまでを**候補ビルド**とし、実画面の採用判断、新しいSource Closureの正式資格、通常／生成WailsでのConsumer受領はWEB-P1a-2bに分離する。SL-1a（Source位置Presenter #5212）やPin/後着#5168は独立のConsumer課題として再利用し、内部Inspectorの内容はAudience／Disclosureの現在資格なしにWeb公開しない。

**WEB-P1a-1bのコード受入**：[Web横断設計 §21](../design/WEB_STANDARD_COMPONENT_AND_AGENT_HARNESS_2026-11.md)の`readInspectorPresentation`を既存の独立Presenterに追加し、Catalog不在・曖昧、同名異Kindの応答、非read-only、Source Index旧版、制限欠落、表示先Workspaceの入力不正を、既存`presentation.test.mjs`のNode回帰で拒否する。検証済み結果だけを別Subjectへ表示し直す誤配線を防ぐため、現在の選択Contextと検証を同じ関数に渡す。これはread-onlyの局所受入であり、現行`main.ts`・歴史的`presentation.ts`・正規Bundle・旧Closureは変更しない。Source位置Presenter #5212とPin/Focus後着 #5168を別の既存Evidenceとして使い、WEB-P1a-2の実画面・Source閉包・生成Bundle・Windows Consumerと公開Web Disclosureは未検収のまま維持する。

**生成Inspector Readerの未接続を別Gateとして扱う**：PR #5303の同一HEADではWorkbenchの正規Node24 Build／埋込byte一致はsuccessだが、複数の`cmd/tei-studio` Go Jobが`TestEmbeddedWorkbenchCarriesTheCurrentGeneratedInspectionReader`で停止した。既存テスト・生成Reader・埋込JS・`main.ts`・`build.mjs`のGit blobは分岐元と同じであり、今回のPresenter変更が直接原因ではない。WEB-P1a-2ではSource契約のDigest→生成Reader一件→実呼出し→正規Bundle→通常／生成Consumerの順に受入する。配布byte一致のみを生成部品の実組込みに昇格させず、既存Goテストの拒否を回避しない（[Web横断設計 §21.3](../design/WEB_STANDARD_COMPONENT_AND_AGENT_HARNESS_2026-11.md)）。

**WEB-P1a-2bの受入順序：Reader同値化を先行**：[Web横断設計 §23](../design/WEB_STANDARD_COMPONENT_AND_AGENT_HARNESS_2026-11.md)の現行コード照合では、手書きReaderにある必須authorityLimitsの検証が生成Contract・生成Readerに無く、sourceStatusの型検査にも差がある。まずW-Rで生成元Contract／既存Transform／生成TSを同じ保証へ揃え、同一Fixtureの正負例で確認する。次にW-Bで一体型Presenterと生成Readerを単一実行経路へ接続し、過去の固定Closureを保存したまま正規Bundle・Digest・Marker・重複Reader不在・埋込byteを新HEADで検査する。最後にW-Cで通常・生成Studio・実Windows WebView2を独立に受領する。#5303／#5306の局所達成をW-R/B/C、Source Open、HTTP Disclosure、正式Stage1→Stage2、実Providerへ流用しない。#1180／#2514、#997／#1824、#2831／#4583の既存Ownerで検収し、新Host・第二正本・新Hard Gateは作らない。

後続の**WEB-P2**はCLI direct／Studio Host／HTTPの同一Operationを一つのread-only実Consumerで比較し、Actor／Audience／Disclosureと現在Providerの選択を別資格で検査する。**WEB-P3**は公開可能なB2C説明Facet一件をTarget Head Slotと実Asset Memberへ接続する。両者をUI Framework選定やDesktopセルフビルドの前提条件にしない。担当は#997／#1824／#1158／#951／#3509、反証は既存#1180／#2514。ここでの追記は実装引渡しであり、製品の新しいE2E試験合格ではない。

### 図解のP1：定義・生成・実Attemptを証拠つきで比較する（2026年11月15日）

[プロセス実行図の因果・意味保存・証拠照合](../design/PROCESS_EXECUTION_CAUSAL_PROJECTIONS_2026-11.md)の§10〜61に、他の標準Data・Node／Plugin配線・業務Process／REA・Rule・セルフビルドの議論を接続した。異なる図を一つの実行エンジンで統一するのではなく、**Sourceからの投影、Targetへの生成、実際の呼出し、独立した受領の間にどの根拠が必要か**を、対象に応じて切り出す。部分取得から「不存在」や「すべて成功」と推測する図は不合格とする。

**方式の選択と証拠の表示（§40〜44）**：既存の型付きRuleを評価する汎用Functionと、RuleからGoを生成する方式は対立しない。判断・実Read・NULL／未取得・エラーが同じ意味を守れる範囲で比較し、解釈ならEvaluatorの実装版と今回読んだモデル、コード生成ならGenerator／Build Input／実Artifact、外部Providerなら実Called／Grant／Effectを別に確認する。最初は固定`quantity_positive`の一Ruleで既存`validationeval.Evaluate`と`validationgen.GeneratePositive`の局所結果・Read先を比較する。main済み#5051の原文同梱・#5058の実CLI inspectは限定成果として保持し、ScenarioのOperator選択Pinは現行mainのPlan／CLIに実在する。Builder入力取り違え#5056とN2実TEI起動#5059も、各々の限定された受入範囲で扱う。**新Interpreter／Optimizer／Hostを製品P0へ追加しない**。

**実装用の反証と保存・通知の段階（§45〜47）**：main済み#5051で実Source byteを原文コピーとしてBuild Sourceへ収録し、実Binary内Hashまで確認できた事実と、main済み#5056の生成Go／Builder入力の取り違え拒否、#5042/#5052の11依存Sourceを固定した生成Workbench実Windows Consumerを、**別々の受渡しの証拠**として扱う。原文コピーの正しさ、Builder実消費、Binary内同梱byte、Consumerが実際に利用したArtifactの同一性を確認するまでは一つのE2Eに結ばない。画面・保存処理は定義→取得・表示→編集→保存前検査→トランザクション→Commit→非同期→開示をProcess／Operation／Effectで説明し、段階を跨ぐGrantやReceiptを捏造しない。EC-01〜08の反例（別Source、説明改版、生成Go差替え、同Resultでの実装変更、Binary Hashの資格、旧Workbench依存、汎用評価との差、ACK喪失）を既存#1180/#2514と該当Ownerへ渡す。新Engine／Graph Store／Expected正本を作らず、通常CLI実Provider R3・正式Builder R2-P・Stage1→Stage2実WindowsのP0を優先する。

**抽象経路による不変条件検査（§48〜52）**：生成後の関数名やNodeの描画IDではなく、既存#1180/#2514の安定Test ID、Qualified Subject、Guard／Preconditions、経路上のInvariants、独立Expectedからテスト対象を定める。まずmain済み#5087の`quantity_positive`に対する汎用Evaluatorと生成Go Binaryの五入力・実Read一回、および#5056の生成Go／Builder配置不一致拒否をAP-01〜08へ読み取り専用で対応付け、同じ試験を再実装しない。到達可能性、実際の通過、状態・Effectの保存、実Consumer受領を分離し、静的GraphやASTだけで未観測の作用を成功・不在と判定しない。次に#5101の複合Port／Field／AdapterのIdentityと秘匿範囲、#5103の粒度・分類・完全性の変化を、同一抽象経路の不変条件として既存Fixtureで検査する。旧生成Scenarioの選択Pinは現行mainの実装を根拠とし、A→B→Aの履歴失効や正式Provider／Host Grantは別に検収し、Studioの正式Stage1→Stage2／通常CLIのactual Providerを別P0とする。新Test Runner、Graph Engine、メソッド名Registry、Trace Store、第二Expected正本を設けない。

**実Read対象を含む経路資格（§53〜56）**：main済み#5118では、`tei.authoring/v1`のData項目定義表と隣接Ruleを同じLoad／Source Indexから選び、汎用Evaluatorと通常`go build`でコンパイルした生成Goの別Process Binaryを比較した。正値・0・負値・負の0・NULL・未取得についてResult・Error・Read ID・回数を観測し、**同じResultと同じRead回数でも、`allocation`ではなく`person_id`を読んだ実装は不合格**とする負例を設けた。この限定的な実証を#5087の別Ruleに対する`tei self-build`実証と一つのE2Eへ合算しない。次はQ-1〜5として、実Read IDと独立Expected、現行mainの組織型付きRelation回帰、Scenario Operator選択Pin、実CLI／直接Operation／Studio Hostの三入口試験、およびmain #5101とService Flow §12の型付きPort／Field／Adapterの取得ScopeとDisclosureを既存Ownerの小さな反証へ分ける。P1の照合図は確認済み／不一致／不足を区別し、実Provider／正式Builder／Audience／Effectを偽装しない。追加のRead Trace Storeや万能Graph Runtimeを設けず、正式R2-P、通常CLI実Provider R3、非Plugin Studio Stage1→Stage2実WindowsのP0を優先する。

**実装用の反証と保存・通知の段階（§45〜47）**：main済み#5051で実Source byteを原文コピーとしてBuild Sourceへ収録し、実Binary内Hashまで確認できた事実と、main済み#5056の生成Go／Builder入力の取り違え拒否、#5042/#5052の11依存Sourceを固定した生成Workbench実Windows Consumerを、**別々の受渡しの証拠**として扱う。原文コピーの正しさ、Builder実消費、Binary内同梱byte、Consumerが実際に利用したArtifactの同一性を確認するまでは一つのE2Eに結ばない。画面・保存処理は定義→取得・表示→編集→保存前検査→トランザクション→Commit→非同期→開示をProcess／Operation／Effectで説明し、段階を跨ぐGrantやReceiptを捏造しない。EC-01〜08の反例（別Source、説明改版、生成Go差替え、同Resultでの実装変更、Binary Hashの資格、旧Workbench依存、汎用評価との差、ACK喪失）を既存#1180/#2514と該当Ownerへ渡す。新Engine／Graph Store／Expected正本を作らず、通常CLI実Provider R3・正式Builder R2-P・Stage1→Stage2実WindowsのP0を優先する。

**抽象経路による不変条件検査（§48〜52）**：生成後の関数名やNodeの描画IDではなく、既存#1180/#2514の安定Test ID、Qualified Subject、Guard／Preconditions、経路上のInvariants、独立Expectedからテスト対象を定める。まずmain済み#5087の`quantity_positive`に対する汎用Evaluatorと生成Go Binaryの五入力・実Read一回、および#5056の生成Go／Builder配置不一致拒否をAP-01〜08へ読み取り専用で対応付け、同じ試験を再実装しない。到達可能性、実際の通過、状態・Effectの保存、実Consumer受領を分離し、静的GraphやASTだけで未観測の作用を成功・不在と判定しない。次に#5101の複合Port／Field／AdapterのIdentityと秘匿範囲、#5103の粒度・分類・完全性の変化を、同一抽象経路の不変条件として既存Fixtureで検査する。旧生成Scenarioの選択Pinは現行mainの実装を根拠とし、実Provider／Owner意味受理へは拡張せず、Studioの正式Stage1→Stage2／通常CLIのactual Providerを別P0とする。新Test Runner、Graph Engine、メソッド名Registry、Trace Store、第二Expected正本を設けない。

**方式の選択と証拠の表示（§40〜44）**：既存の型付きRuleを評価する汎用Functionと、RuleからGoを生成する方式は対立しない。判断・実Read・NULL／未取得・エラーが同じ意味を守れる範囲で比較し、解釈ならEvaluatorの実装版と今回読んだモデル、コード生成ならGenerator／Build Input／実Artifact、外部Providerなら実Called／Grant／Effectを別に確認する。最初は固定`quantity_positive`の一Ruleで既存`validationeval.Evaluate`と`validationgen.GeneratePositive`の局所結果・Read先を比較する。main済み#5051の原文同梱・#5058の実CLI inspectは限定成果として保持し、選択鮮度#5054は未統合であり、Builder入力取り違え#5056とN2実TEI起動#5059はmainに統合済みとして各々の受入範囲を分ける。**新Interpreter／Optimizer／Hostを製品P0へ追加しない**。

**値の状態と部分取得を守る検収（§57〜61）**：main済み#5142は`validationeval.Decide`で`present`／`null`／`not-fetched`とRelationの`complete`／`partial`を区別し、追加Readで結論が変わり得る場合は理由付きの「判定できない」とする。一方、取得済み二行で確定した一意性違反は、未取得行があっても不成立と判定できる。#5143の記録例表は`tei validate`のExpectedの照合に用いるもので、例の行を実DBの完全取得・実Providerの受領へ転用しない。#5146の暦日`not-after if present`と、審査中#5148のRelation期間重複は別の意味・取得範囲・Group Keyで検証する。次はKD-01〜07を既存`Decide`、`CheckRuleExamples`、Source Index、生成Go実Build回帰へ対応付け、**Read ID／値の状態／Coverage／Scope／Expected／Actualを示す小さな読取専用の照合**に絞る。生成Goの正値Rule局所同値を、状態付きRule・日付Rule・実DB Providerまで一律に広げない。正式R2-P、通常CLIのactual Provider R3、非Plugin Studio Stage1→Stage2実Windowsは独立P0に残す。

最初の実装担当は#1124／#2497／#4198（読取専用図解）と#988／#1121（選択済みRule・Go生成）である。固定した一Ruleの現在のSource／型／Read範囲をCLI/Headlessで表示し、未接続のBuilder・Provider・Effectを未確認のまま示す。同じ入力・Rule IDで参照Attributeを一因子変更し、生成Go／実Read／判断と旧Evidenceの失効を既存#1180／#2514へ返す。Generator自身のGo構造をTEIの意味候補へ戻すP1 dogfoodingは別Subjectとして進める。**Go AST・図の一致、Mock、別HEADのCI成功を、通常CLIの実Provider／Stage2実Windows受領へ加算しない。**

**今回の実装引渡し**：§16〜19では、Rule→Guard、Event→Batch、Effect→通知という異種境界で、型・粒度・時点・配送・現在Grant・Receiptの保証を勝手に繰り上げない。C0/C1は既存Rule読取・生成Goの一因子変更、C2/C3は合成のpartial入力・Effect unknown・Provider取り違えを反証、C4は同じSubjectをCLI/Studioへ読み取り専用で投影する。最初の票は一問・一Subject・現在のSource/観測範囲と不足の理由に限定する。合成のEffect反例は実DB Commitや実通知の成功根拠にしない。既存#4198/#1130/#1174/#4495/#1180/#2514へ受入を返す。

**新たに明確にした受入条件（§20〜24）**：次のOperationを「実行可能」と表示する前に、定義・入力Contextの取得範囲・選択Provider・現在Grant・先行Effect・Audienceを各Ownerの根拠に照合する。非同期のJoin／Batchでは業務時点、Read Snapshot、実Attempt、受領時点を混同せず、問いに必要な原因がそろった範囲だけを説明する。中断後は純粋計算の再評価、Checkpointからの継続、外部Effectの照会と再試行、補償、結果の再配送を別の操作とする。**最初の製品差分はC0〜C1のRule一件**で、同一Source・型・Read・生成Goのread-only照合と一因子変更の反証を既存CLI／Inspectorへ加える。C2〜C3の部分入力・Provider不一致・Effect unknownはまず合成Fixtureで止め、実DB／通知成功へ昇格させない。C4の実Consumer受領は#4495/#1251/#1824/#2831/#4583へ分けて検収する。独立Expectedは#1180/#2514を用い、新Scenario正本・Runner・全般Ready APIは追加しない。

**現行製品に基づく最短のP1差分（§25〜29）**：mainに統合済み#4980の実`tei inspect`正負試験を出発点に、`semantic-validation:sales.order_line.quantity_positive`のSource位置、元byte／Fingerprint、型付き属性、Authority Limitsを、CLI JSON／Inspector／小さなNode図の同一Subjectとして照合する。**同じRule IDの未公開Capabilityと、`text` Fence内だけのRuleを正式な定義へ昇格させない。** `ReadWorkspaceSubject`が同一Loadで取得するModel Revisionと、現行CLIの`worldFingerprint`／`sourceIndexFingerprint`は別物であり、製品CLIに存在しない版を想像して「同じSnapshot」と扱わない。まずread-only／現行契約のFit/Gapと同名異Kind・古いPin・Source改版の独立Expectedを一件で示し、Read Footprintと生成Go、正式Builder消費、実Provider呼出し、別Consumer受領は各Ownerの証拠を順に結ぶ。実Builder／Provider／Effect未接続を成功扱いせず、新Core型・Runner・公開API・Hard Gateを追加しない（#1124/#2621/#988/#1121/#4544/#4495/#1824/#1180/#2514）。

**実装基線の更新（§30〜34）**：#4980の実`tei inspect`とは独立に、#5007で選択済みOperator実装の局所Call、#5009で選択済み`quantity_positive`由来Goの通常製品`tei self-build`・Build Intent・実Binary判断がmainへ追加された。これらを同じRule／Attemptの縦断E2Eと合算しない。まず**Manifestの実装選択A→Bだけが変わり、Canonical Worldは変わらない場合に、旧生成Scenario Commandが古い選択で起動されることを拒否または再資格化する**E1を#4359/#951と並行PluginのCPW-11へ渡す。入力0で`non-negative`と`non-zero`の結果が異なる独立Expectedを用い、World Fingerprint一致を現在の実装選択資格に転用しない。E2のRule→Go→Buildは実Binaryの局所成功と元Markdown／Generator版の正式Build Plan束縛を別票にし、手書きDriverを自己構築の生成部分に数えない。E3の組織MembershipはvalidAt／knownAt、recordedAt／observedAtと二Occurrenceの並存を区別し、隔離PostgreSQL候補と製品正式Queryの受入を分ける。E4のCLI／Studio／AI表示では古いPin・開示・Audienceを維持する。新しく共通Runtime／証拠Storeを作らず、P0を優先する。

**証拠の照合資格と現行製品の受入（§35〜39）**：次の実行図P1は、既存`tei inspect`のQualified Subject・Source byte／位置・型・Authority Limitsから、確実に言えるEdgeと未確認Edgeを区別するread-only表示に限定する。#5037の実CLIと直接Operationの結果一致は共通ReaderのTransport同値であり、独立ExpectedやProvider実Callではない。#5038のREA実Evidence端点照合は技術的参照整合であって業務Relation／Effectの受理ではない。#5027の組織二時点隔離PG候補はOccurrence／validAt／knownAtを区別し、正式Query・Grantを代行しない。#5025の実WebView2 Pin再読込負例は現在観測の再資格化を示すが、Stage1→Stage2受領ではない。**選択A→Bで旧生成Scenarioを拒否するGo実装は#5054が審査中**のため、mainでの既定保護として数えない。新規共通証拠Store・万能Status APIを作らず、同一Load・同一Subject・実Producer・取得範囲・選択鮮度・Audienceを既存Ownerの証拠から照合する。検収結果は既存#1180/#2514と#4198/#1124/#951/#1824へ返し、通常CLI R3／正式R2-P／Stage2実Windowsの既存P0を優先する。

**実現方式と証拠の受渡し（§40〜44）**：標準Rule／型付きSourceは、既存`validationeval.Evaluate`による汎用評価と、`validationgen.GeneratePositive`による生成Goの二経路を別々に実証できる。コード生成が必要な場合と単純なパラメータ評価で足りる場合を選べる余地を残すが、これを口実に新Evaluator／Runner／DSL選択構文を先行しない。main済み#5051では原Markdown byteの同一Load検査、`//go:embed`付きSource Closure、実Binary内SHA-256の確認まで到達した。ただし元LocatorとReader／Generator実装資格の正式Build Plan束縛は未受理。生成Goをコンパイル可能な別byteへ入れ替える#5056と、選択A→B後に旧Scenario Commandを拒否する#5054は**未マージPR**として継続し、mainの合格としない。#5042/#5052の11依存Source／Lockを固定した生成Workbench PoC Bの実Windows受領は限定的に検証されたが、Rule Binaryと同一Artifactの一往復でも正式Stage1→Stage2でもない。次の実装差分は既存テスト内部のEC-01〜08（別Source、同Go／説明改版、生成Go／実Builder不一致、同Resultの実装切替、Binary原文Hashの限界、UI旧依存、汎用評価との比較、Effect unknown）を#1180/#2514へ渡し、定義→Read→編集→保存前→DB操作→Commit→通知→開示を別資格として表す小さなread-only照合票を作る。Source出典の再確認・Artifact再利用・現在Grantを分離し、正式R2-P／R3とWindows Stage2を引き続きP0とする。

P0は引き続き、#4495の公開pure Entry→actual Provider→通常CLIの実二入口と、#2831/#4583のStage1実TEI→Stage2実Windows WebView2を優先する。図解・編集・REA・BRMS・ETLの全対象対応をP0の新しい前提にしない。

### TEI自身の実装をモデル・図・DSLへ折り返すdogfooding（P1、既存P0と並行）

[実装コードの意味保存と自己記述dogfooding](../design/IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md)では、**現在動いているTEI自身のGoコードを観測し、既存の標準Data／Function／Business・Rule／Processへ対応付けて、図表・DSL候補を導出する**ことを検証対象に加える。単にDSLからGoを生成したり、構文木を図にしたりするだけでは合格としない。業務判断、Read／Failure、Generator自身の責務、Plugin／Build／実Consumerの証拠を分け、元Sourceへの対応と未理解の実装範囲を示す。

最初の対象は実在する`validationgen.GeneratePositive`（**生成Function**）と、そこから生まれた`EvaluateSelected`（**評価Function**）の二つである。まずGo AST・型と実Go結果からread-onlyの構造候補を抽出し、既存`quantity_positive`のDSLとGraphvizの対応を照合する。その後、正値／0／負値／NULL／未取得、Rule条件変更、説明文だけの変更を独立Expectedで比較し、実生成byte→正式Builder→実Providerへ段階的に接続する。**コードから推測した意味や図をOwner受理・実行許可に昇格させない。** TEIで実際に置換できた判断・実装責務、残る手書きGo、変更時の労力・性能を測る。新しい逆コンパイルEngine、第二Meaning正本、汎用Runnerは追加しない。

このP1は通常CLIの実Provider／独立Consumerと、Studio Stage1→Stage2実Windows受領の**P0を待たせるHard Gateではない**。既存#2619/#2620（構造）、#988/#2621（DSL）、#1121/#4544（Generator/Builder）、#1180/#2514（独立Expected）、#2831/#4583（Studio）へ分担する。現段階は設計・検収条件であり、実Code→DSLの製品機能を実装済みとはしない。

### 宣言の役割を見分けるP1（2026年11月15日）

[文書手法からアプリケーション抽象への横断設計§8〜10](../design/APPLICATION_ABSTRACTION_DOCUMENT_METHOD_WEAVING_2026-11.md)に、他レーンの実Product Inspector（#5058）、原文を含む実Go Binary（#5051）、独立Shape基線（#5064）を接続した。次のP1は、**記述上のRole／意味上のShape・Law／実装のBindingを分けたまま、同じ正値Ruleの汎用評価と生成Goを五入力・Read回数で比較する**ことから始める。これは方式間の限定的な同値確認であり、正式R2-Pの出典閉包、通常CLI R3の実Provider呼出し、Studio Stage1→Stage2を達成したものではない。DW-11〜16は#2621/#2497/#4544/#1180/#2514へ既存の試験責務として引き渡し、新EngineやP0の追加Gateは設けない。

[宣言的構文の合成契約§18〜20](../design/DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)の実装引渡しは、**表面が同じでも定義・事実・要求・実現選択・実観測を混ぜない**ことから始める。最初の対象は、選択済みMarkdown Rule一件と、その元Source・型・論理Readである。Data項目定義表のPR #5010は採用・統合確認後に対照へ使い、業務記録の表を自動的にData定義へ変えない。

| 小さな受入単位 | 得るもの | 失格となる短絡 |
| --- | --- | --- |
| **P1-A：ReaderとRole**（#2621／#988） | 同一Rule IDの定義Occurrence、元byte／Reader、既存Typed Meaning。追加のData表は受理された実経路からだけ読む | 表の列名だけで意味を確定、同一IDの二重Definitionを黙殺、他Workspaceと合一 |
| **P1-B：接続理由のread-only投影**（#1124／#2497／#4198） | 既存Source／型／Readと、不足しているProvider・Builder・権限の理由を一問一対象で表示 | 「構造適合」を「実行可能」へ昇格、partialから不在を断定、描画Edgeを実Dispatchにする |
| **P1-C：実Builder／実Callの別検収**（#4544／#4495／#951） | R2の生成Go実byte→実Builder消費→Artifact／Consumerと、R3の選択Provider→実CLI Callを別々に観測 | 別AttemptのPASS合算、選択B／実Call A、A→B→Aの旧証拠復活 |

正例・負例はDC-21〜26を既存#1180／#2514へ引き渡し、実装から期待値を逆算しない。P1の診断表示が完成しても、既存P0の通常製品CLI・実ProviderおよびStage1→Stage2実Windows受領の完了を宣言しない。追加のGrammar・Writer・Hostを着手前提にしない。

### P1：構文汎化を実製品の出典照合につなぐ（2026年11月15日追補）

[宣言的構文の合成契約§21〜23](../design/DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)を、前節のP1-A〜Cへ追加する**実装判断**として接続する。現行mainには印付きData項目定義表の製品Load回帰、Operator実選択／局所Call（#5007）、生成Ruleの通常tei self-build（#5009）、原DSL Sourceと生成Go ASTの同時照合（#5020）が存在する。ただし、Data表の原提案PR #5010は未マージであり、実在するコードの所在と提案PRの状態を混同しない。

| 既存の実装担当に渡す問い | 最初の確認方法 | 受理しない短絡 |
| --- | --- | --- |
| **P1-A／Reader・Role**（#2621／#988／#1018） | 現行の明示選択Readerで一つのDefinitionを読み、印のない表を説明として残し、旧DSLと既存Meaning／Source位置を比較 | Data表を読めたという理由で業務記録も定義化、選択Languageを自動推測 |
| **P1-B／Operator Law・選択の鮮度**（#988／#4359／#2514） | 非負と非ゼロの実装を境界値0で区別し、実装の選択・実Call・意味的適合を別Expectedで確認。manifest変更後の旧Scenario Commandを拒否 | 型適合とCall成功だけで業務Rule適合を承認 |
| **P1-C／Build出典**（#1121／#4544／#2595） | 一Ruleの元Markdown・Reader・Generator・生成Go・Build Intent・Builder実入力・最終Artifactを同一Build Attemptで照合 | #5009の実Binary試験だけから元Source/Generatorの完全束縛を推定 |
| **Studio投影**（#1124／#1824／#4583） | 既存Inspectで確認できる欄だけから一問一Subjectのread-only表示を作り、未取得・不一致・古いPinを区別 | GUI表示だけでSource編集や実Provider・Windows Stage2実行を認定 |

DC-27〜32は#1180／#2514の既存独立Expectedへ引き渡す。最初に実装するのは既存Reader／Ruleの検査可能な経路であり、汎用Node Editor、Markdown表の全面Writer、第二のPlugin Resolverは作らない。全体のP0は通常CLIの公開pure Function→実Provider→独立Consumerと、Stage1によるStage2 Desktop実Windows再構築のままとする。


### P1：Data定義表の製品Inspectを最小実装で確認（2026年11月15日）

[宣言的構文設計§24](../design/DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)を受け、既存Data項目定義表のWorkspace Loadから**一つのSource、三つのSubject（Data／Attribute／Rule）、製品Inspector、Source Indexの原byte位置**までを追加のGo回帰で照合する。内部のReader選択記録とInspectorの出典表示を同じLoadから確認し、Studioがまだ公開していないLanguage列を架空のUI機能として扱わない。

実装票はinternal/operations/inspect_markdown_data_table_source_evidence_test.go。印のない例表からDefinitionを生成しないこと、Ruleから公開Capabilityを推測しないこと、説明段落だけを変えた場合に型付き意味を保ちながらSource revisionが変わること、A→B→Aで旧Grantを復元できないことを独立の問いとして扱う。テストファイルの存在とexact HEADのCI合格を区別し、他のReader・SQL／Query・Persistence・生成Artifact・外部Providerへ合格を拡大しない。

このP1の次段は、Product Inspectorが実際に返せるSourceと意味の列だけを使ったread-only診断である。R2-Pの元宣言→実Builderへの出典束縛、R3の公開Entry→実Provider、Studio Stage1→Stage2実Windowsは従来の並行P0とする。新たなGrammar／Reader Registry／Graph Runtime／Writerは前提にしない。


### P1：Data定義表の実CLIと共通Operationを一致させる（2026年11月15日追補）

[宣言的構文設計§25〜26](../design/DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)の具体化として、cmd/tei/cli_inspect_markdown_data_table_process_test.goに**実OS Processでのtei inspect→既存operations.InspectSubject結果との一致**を検査するGo試験を追加する。main済み#5037のRule実CLI／直接Operation照合と対になる、別表層Data項目定義表の最初の製品Consumer確認である。

試験ではData／Attribute／RuleのCanonical、Source Roleと原byte位置、権限制約を照合する。印のない例表やRuleと同じIDのCapabilityはNot Foundで拒否し、説明文だけを変更した後は同じMeaningのまま新Source Fingerprintを返す。**CLIのread-only出典と、公開Function／Plugin Providerの実Callは別資格**とし、Studioが実行・編集・外部開示を許されたと推測しない。

並行する#5051は選択元Markdownのbyteコピーを生成Goの実Build入力に含める限定試験、#5052は生成PoC Bの実Windows WebView2成功を記録したもの。前者は正式R2-Pの元ファイルLocator・Reader・Generator出典閉包全体を、後者は非Plugin Studio Stage1→Stage2セルフビルドを保証しない。次に製品として受け入れる一件は、**元宣言→実Build入力と実Consumerの対応を同じAttemptで証明**することであり、通常CLI実Provider／Stage2を並行P0に保つ。

この追補に伴う新試験の実合否はexact HEADのCIで別途確認する。DC-33〜38は既存#1180／#2514の独立Expected候補へ引き渡し、新しいGrammar／Graph Runtime／Registry／Hostは作らない。


### P1：Data表とRuleの汎用評価・生成を同じ意味で照合（2026年11月15日）

[宣言的構文の合成契約§27〜28](../design/DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)のG（コード生成）／P（パラメータ実行）／D（外部委譲）の区別を、既存Data定義表と隣接Ruleの実装票へつなぐ。internal/operations/inspect_markdown_data_table_rule_evaluation_test.goに2つのFocused試験を追加し、同じ製品Loadから一つの型付きValidationを解決する。

Pは既存validationevalで正値・0・負値・NULL・未取得の五入力と論理Read先を確認する。Gは同じRuleから既存validationgenで生成したGoの参照先と条件を検査する。ただし**今回の生成Goを実ビルド／起動して比較する試験ではない**。Data項目表のallocation型をdecimal→textへ変える一因子負例では、Validatorの拒否とGo生成拒否、Source／World版の変更を別に確認する。

DC-39〜40を既存#1180／#2514の独立Expectedへ接続し、DC-41は実BinaryとのRead Footprint一致を要求する次段候補とする。元宣言→Generator→実Builder入力→Artifactの正式R2-P、通常CLI実ProviderのR3、Studio Stage1→Stage2 Windows受領は並行P0であり、Pの評価成功をGの自己生成達成率へ加算しない。対象HEADでGoテストとRepository Healthを確認するまで新たなPASSを主張しない。

### P1：Data表のRuleを実Binaryまで二方式で照合する（2026年11月15日）

[宣言的構文の合成契約§29](../design/DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)の最小実証として、印付きMarkdown Data項目表のallocationと隣接Ruleを同一Workspace Load／Source Indexから選択し、既存汎用Evaluator（P）と、既存生成器が作ったGoの実Binary（G）の正値・0・負値・負の0・NULL・未取得を独立Expectedで照合するFocused試験を追加する。実装票：internal/operations/inspect_markdown_data_table_rule_binary_parity_test.go。

**結果の一致と読取対象の一致を分ける。** 生成Goの参照先だけをallocationからperson_idへ変更し、同じテスト値を返した場合でも、Binaryが観測したNumberの読取ID不一致で同値を拒否する。これはGraph・Plugin配線の型付きPortを接続する際にも重要な反証である。ただし、今回のSourceはテスト用のpure Sourceであり、実Provider・DB・開示範囲を検証したものではない。

G／P／Dの分類は図表実現の解釈・生成・混合・Hookの配置方法とは別軸に置き、方式切替え後の旧Artifact再利用、現在Grant、Ownerの意味受理を同じPASSへ統合しない。DC-42〜45を既存#1180／#2514へ接続し、今回の実行可能な対象はDC-42／43のみとする。**追加テストのGoビルドとCI合格はexact HEADで別途確認**する。

次の受入は正式Build Intent→Builder実入力→Artifact→独立Consumerの同一Attemptと、通常CLIの公開Entry→実Provider、Studio Stage1→Stage2の実Windows再構築である。新しいInterpreter／Graph Runner／第二意味正本やP0 Hard Gateを増やさない。

### P1：BuildPlan確定後のMarkdown改版を実self-buildで拒否する（2026年11月15日）

[宣言的構文の合成契約§30](../design/DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)の実装として、既存の選択済みRule→生成Go→通常tei self-buildの縦断回帰に、**BuildPlan解決後に同梱元Markdownの説明だけを変更する**一因子の負例を追加する。生成Goのbyteは固定し、旧plan.json／Intentを用いた実CLIがビルドを拒否し、L2-bを主張せず、成果物も公開しないことを検証する。元Markdownを戻したらSource Closure／Plan Fingerprintの記述的な一致を再確認し、既存の正常な実Builder・実Binary確認へ進む。

この追加は、#5109の「Plan確定後にgenerated.goだけを変更する」試験とは独立の検証である。Sourceだけ変わる場合と生成Goだけ変わる場合の**二つの異なる失効原因**を区別する。生成Gと既存Evaluator Pが同じ結果を返しても、旧Planで現在のBuild入力を消費してよいとはしない。型付きPluginのWhole／Field／Adapter配線、実装選択のPin、外部開示・現在Grantも別途確認する。

**今回のGo試験の合否はexact PR HEADのCIで判定**する。正式R2-Pに必要な元Locator／Reader／Generator版の完全なAttempt閉包、R3の通常CLI公開Entry→実Provider、Studio Stage1→Stage2実Windowsは未解決の独立P0を維持する。DC-46〜49のうち今回追加する実回帰はDC-46のみ。DC-48／49は#1180／#2514へ渡し、新Builder／Runtime／Evidence Storeを作らない。

### P1：Rule種別ごとのG／P対応と三値判定を区別する（2026年11月16日）

[宣言的構文の合成契約§31](../design/DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)を、mainへ入った#5142（NULL／未取得／部分取得を区別するDecide）、#5143（記録例の独立Expected）、#5146（暦日二項目のnot-after Rule）と接続する。新しいRule種別をMeaningへ採用しても、既存のGeneratePositiveが同種別を生成できると推測しない。

Focused試験 internal/operations/inspect_markdown_data_table_date_rule_realization_test.go では、既存の印付きMarkdown Data表と同じteiコードフェンスに、テスト専用の暦日比較Ruleを追加する。製品Reader／Inspectorから同じSourceへ結び付け、PのDecideで日付の順序、if present、NULL／未取得の区別、二Attributeへの実Read要求を検証する。**同じ文書内でpositiveの生成Gは可能、not-afterの生成Gは明示的にunsupported**とする。std.dateをstd.datetimeへ変更した場合はValidatorの型検査で拒否する。

今回のDC-50〜52はFocused Go試験へ、DC-53〜54は#1180／#2514の独立Expected候補へ引き渡す。現行Pの評価能力、製品Gの生成能力、外部Dの選択・実Call・現在Grantを独立した欄として扱い、P成功によるG／Dへの無音Fallbackを禁止する。実Builder／正式R2-P、通常CLI実Providerと独立Consumer、Studio Stage1→Stage2は引き続き別P0とし、追加のRule種別を新P0 Hard Gateにしない。

### P1：同じMarkdownの定義・記録例・診断を別の証拠として扱う（2026年11月16日）

[宣言的構文の合成契約§32](../design/DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)に従い、既存の異動申請のData項目定義表＋正値Ruleと同じMarkdownへ、記録例と期待値を明示して追加するFocused Go回帰を置く。実装票は`internal/operations/inspect_markdown_data_table_rule_example_validation_test.go`であり、新しいReader／Generator／Runtimeは作らない。

対象の4行は、正値・0・NULL・負値。記録例を追加しても**既存Data／RuleのCanonical WorldとDefinitionの位置は変わらない**こと、Source Indexは元byteの変更として更新されること、期待を0の1セルだけ誤らせると`RULE_EXAMPLE_MISMATCH`を例の期待セルへ返すことを検証する。さらに、値が空のNULLと、項目列を持たない未取得は別の判定不能理由として扱う。CLI／Studioが同じ診断を表示する場合も、Rule定義の編集位置と例の期待値の位置を混ぜない。

DC-55〜57をFocused試験のExpected、DC-58を#1180／#2514へ渡す。今回の判定範囲は実製品のReader、Source Index、Inspector、Rule例の評価と診断であり、実DB Reader、外部Provider、Gの実生成・Build、Dの実Call、取得範囲・Audience・現在Grant、Studioの実編集／Stage2を成功とはしない。並行する期間Rule／Contract Query Reader／Operator選択／Plugin Field投影のPRは、main統合と実Consumer確認後にだけ計画上の到達済みへ移す。通常CLI実ProviderとStage1→Stage2のP0を維持する。

### P1で接続する業務判断・設計入力・数式（2026年11月15日）

これらは新しい製品P0の前提ではなく、既存の標準Data/Rule/Expression・Function Data Context・表現型を実案件に近づける**受入作業**とする。詳細な正負例とOwner境界は[横断統合受入境界§14〜17](../design/CROSS_CUTTING_ACCEPTANCE_BOUNDARIES_2026-11-12.md)を参照。

| 対象 | 次に確認する最小経路 | 失格となる短絡・Owner |
| --- | --- | --- |
| **BRMS相当の業務判断** | 明示RoleのMarkdown Decision Table→既存Rule/Context→生成Target結果。条件重複、未定義、partial/NULL、有効時点、Rule改訂を独立Expectedと比較 | 暗黙の行順による優先決定、Ruleの説明だけで実Commit成功を主張。#2621/#4370/#1126/#2506/#951/#2514 |
| **多様な設計入力** | 手描き業務図の写真とCSV/Excelを出典位置付き候補として取り込み、三要素分析法のモデルへ人間がReview。会議録・帳票・既存画面は経路別の観測境界を維持 | 画像解釈・列名・画面表示を受理済み意味に昇格、同名別Entityを統合。#1124/#2621/#267/#3394/#2497 |
| **KaTeX数式と演算** | Markdown中の式Source/Role→既存Expression/Operator/Context→表示と独立結果を比較。単位、精度、未観測、時点、マクロ依存を保持 | 数式の表示一致を実計算成功やSource一致に読み替える。#4370/#4350/#4353/#2495/#2514 |

独立したBRMS実行Host、全資料の中央取込サービス、数式専用DBは新設しない。未取得の外部資料や未接続のReader/生成器/Consumerについて、完了を主張しない。

最初の各PRには、**既存のSourceと対象版、具体的な正例・反例、実行された入口・Builder／Provider、独立Expected、確認できたConsumer、未検証の範囲**を短い一枚の引渡しとして残す。Ownerの意味受理、現在のHost Grant、外部Effect、実DB Migrationは別資格とし、推測による補完はしない。図面・Markdown表・Graphvizは標準データへの可読な表現として使い、表示上の変更を意味や実行権限へ昇格させない。

### Script LaunchとBackend配置の製品受入：目標構成と現在の基線を分ける（2026年11月14日）

11月14日の横断検討では、Studio Desktopの**将来の設計目標**を「Wails ClientがローカルBackend子プロセスを管理し、IPCで共通Operationを利用する」構成として整理した。一方、現在の**製品P0の検証基線**は、Wails Nativeの同一プロセス構成である。[Studio配置比較 #4635](https://github.com/hat47x/tei-platform/issues/4635)と、mainへ取り込まれた[Script Launch設計 #4688](https://github.com/hat47x/tei-platform/pull/4688)／[Backend設計 #4690](https://github.com/hat47x/tei-platform/pull/4690)（[後続統合 #4706](https://github.com/hat47x/tei-platform/pull/4706)）は、この二つを混同せず、子Backend採用を新しいStage1→Stage2の必須Gateにしない。設計文書のmain統合は、子Backendの実装、製品Entryの実起動、Stage2セルフビルドの成功を意味しない。

**CLIではADR-0048のin-process既定を維持**する。Wailsには依存させず、軽量・短命の操作は共通Go Application APIを直接呼ぶ。Plugin隔離や長時間処理が必要な場合の同一実行ファイル子Backend、Harness向けの明示Session再利用、接続先を指定したRemote CLIは比較対象であり、既定の変更ではない。Remoteで使うPluginと現在Grantは接続先Serverが所有する。CLI側へ同じPlugin版を要求したり、通信断でLocalへ黙って切り替えたりしない。

| 実証単位 | Ownerと着手対象 | 最小の合格条件／拒否条件 |
| --- | --- | --- |
| **SL-0：現行基線** | #1158／#1824／ADR-0048。通常tei CLI・Native Wails・共通Go Operation | ListWorkspaces／InspectSubject／ValidateWorkspaceのSource、Subject、Revision、Failureを同一Fixtureで照合。CLIのWails非依存と直接実行の維持 |
| **SL-1：Product Function実呼出し** | #4495／#4502。Owner受理済みpure Entry一件 | Typed API→Host-selected actual Go Providerと、通常tei実Process→同じ意味Contractを独立Attemptで検証。Reference実行やMock／宣言だけのProviderでは不合格 |
| **SL-2：配置の比較** | #4635／#3419〜#3421。Native一体と明示local child | 実Child image／argv／親子所有、専有Endpoint／Peer、Protocol／Operation Contract／Server世代、同一Sourceの診断を比較。同一exe Worker、loopback、Pipe／UDSの優劣は実測で判定 |
| **SL-3：Pluginの実利用** | #951／#1251／#2044。Server側Composition／Load | Discovery→Lock→選択→実ロードProvider→actual Callを同一Attemptで照合。Clientが送るPlugin名／Role／Grantは実行資格ではない。RemoteはServer側選択版を正本とする |
| **SL-4：Session・復旧** | #4635／#1174／#1129。短命CLI Child／明示Session／Studio子Backend | Process Instance、Server generation、Session、Invocation、Attempt、Operation、Effect Receipt、Consumer応答を混同しない。Child Crash後の未保存BufferとPin保持、旧世代返信の拒否 |
| **SL-5：TEIの自己生成** | #2831／#4583／#2621。CLI実Codegen→Stage1→Stage2 | SourceのGraph／Ruleが生成コードと動作を決定。Stage0による二度Build、Go委譲のみ、Builderの自己申告を除外。Windows WebView2受領は独立Gate |
| **SL-6：異質Consumer** | #1158／#3421／#2586／#4679。MCP／Remote／Studio／Browser | 意味が同一でもAudience、Host Grant、Read Scope、Plugin版、Service Generationを個別に再確認。Manualの閲覧可をEntryの実行権限にしない |

**子プロセスの生存と、業務Operationの完了は別**である。親はChildの起動・監視・終了と孤児化防止を担当し、子は選択済みPlugin／Providerの実利用とHostによる現在資格の確認を担う。Childの再起動は、前のInvocationを再送する許可ではない。更新・切断・取消・Commit後ACK喪失で外部作用が不明なら、既存FailureのeffectUnknown／lookup_firstに従い、Operation IDとReceiptの独立照合まで自動再送しない。Read-onlyの再取得でもSource RevisionとAudienceを再固定し、旧Cacheを現在の情報として表示しない。

最低限の反例は、(a)別Child／再利用Endpoint、(b)Handshake後のPlugin Lock変更、(c)選択Provider B・実呼出A、(d)親終了による孤児Child、(e)作用後Crash・ACK消失、(f)再起動後の旧Resultが別Subjectへ届く、(g)Remote接続断後の暗黙Local fallbackである。既存#1180/#2514のExpectedへ引き渡し、第二Scenario正本・万能Session Registryを追加しない。

**優先順位と判定上限**：全体P0のStudio Stage1→Stage2、並行P0の実DB Function Read、CLI先行のTEI意味Codegenは変えない。SL-1の製品pure Entryは#4495の並行課題とし、SL-2〜4の子Backend比較・Session復旧を先行Hard Gateにはしない。今回の追補は計画・実装引渡しのみであり、Child IPC／Session再接続／Product Function実Call／Plugin更新／CLI2再構築の新しい試験は**NOT RUN／NOT ACCEPTED**。

**起動契機から実受領までの横断接続（2026年11月15日、[起動契約§30](../design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)）**：Script Launchは、CLI／Desktop／Web／定時処理／AI Harnessなどの**入口が異なっても、受理する業務Entry・現在のOperation資格・実Provider・Effect／Consumerを別々に確かめる**計画とする。Trigger到着・Window表示・AI Proposal・Plugin Discoveryだけで実行許可を与えず、現行Pin／Source Revision・DisclosureとHost側の現在Grantを照合する。Node Viewと標準Processは関係を読みやすく表すProjectionとして再利用し、描画Edge IDやAIの候補を業務Meaningの正本にしない。実証上は2026-11-15にmain統合済みの[PR #5037](https://github.com/hat47x/tei-platform/pull/5037)がSL-0の`inspect`正常／拒否を直接Operationと実CLI別Processで一致させた。該当HEADのAction Surface実CLI workflowはSUCCESS。ただし**子Backend・実Provider・変更操作・Stage2を実証したものではない**。

**配置・配布の採否を分ける**：SL-2のA／B0／B1／B2はProcess／IPC方式、D1 Remote接続・D2単一Image・D3別Server同梱はRelease形態であり、実行方式Aを「D1」と呼ばない。まずSL-0のResult／Failure基線を用い、#4635がCLI B0 read-only一往復→Windows Wails Bridgeを独立に比較する。B0 WorkerとWails Updater Helperは同じexeの別Roleになり得るため、#1824で通常GUI／Worker／Helperの誤分岐、実Image・Peer／Session、旧Reply、Update中の未知Effect、User設定非上書きを負例として確かめる。Wails beta.29→安定版へのexact Pin移行、D2署名付き更新・D3配布・Mobile探索はSL-1の公開Function実呼出しや非Plugin Stage1→Stage2の**先行条件にしない**。必要な独立Expectedは既存#1180/#2514へ差分を渡す。


**起動時に選択する実装と実際に使った実装の照合（[起動契約§31](../design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)）**：main統合済み#5051は選択Ruleの原Markdownを生成Goと同じBuild入力に収録し、実BinaryのSHA-256が元byteと一致した限定実証である。#5054はOperator実装A→B変更後に旧Scenarioコマンドを拒否する修正候補であり、未マージのまま製品資格に加算しない。起動後の実選択Provider・現行Grant・実CallをSource／生成byteの一致から推測しない。定義済みRuleを**既存Evaluatorで解釈する**方式と**Goを生成する**方式は、同じ意味と独立Expectedで比較できるが、前者を自己コード生成の成果と数えない。#5042/#5052の生成PoC B Windows実受領と、非Plugin Stage1→Stage2実再構築も別に検収する。新Interpreter／Hostを作らず、B0とR3を独立Ownerへ引き渡す。

**SL-0のCLI／Studio Host三入口基線（[起動契約§32](../design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)）**：既存#5037の通常製品`tei inspect`別OS Processと`operations.InspectSubject`の同値試験に、Wailsに依存しない`studiohost.ReadOnly.InspectSubject`を追加する。同じ内蔵Markdown Ruleについて、正規化意味・元Source／Workspace Identity／`authorityLimits`を含む`SubjectInspection`全体を照合。同名非公開Capabilityと偽Ruleの`SUBJECT_NOT_FOUND`、未許可Workspaceの`WORKSPACE_NOT_ALLOWED`、`business_rejection`／`not_retryable`／`effectUnknown=false`を実Go試験で確認する。これは**Studio Host Go境界の検証**であってWebView／HTTP公開・現在Grant／正式Provider Callではない。旧#5096のAPI Plugin設計は閉鎖済みのため、AP-1の薄いWails Server公開Adapterを#1158/#1824の次の小差分として扱い、CLIへのWails・HTTP依存を加えない。LoopbackへのbindのみでSourceの開示許可を得たとは扱わない。#5059のN2実CLI子Process／Windows成功はBuild一往復の実証であり、B0常駐Backendや正式Stage1→Stage2の成立と区別する。現行mainのGoソースには三入口テストが収録済み（旧#5119/#5137は未マージ閉鎖）で、main版の検証は別に確認し、P0のStudio再入#2831/#4583・通常CLI実Provider#4495を維持する。


**SL-0から実Provider／実公開へ進む検収列（[起動契約§33](../design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)）**：現行mainの内蔵Rule`inspect`三入口Goテストを局所基線とする。旧#5119/#5137は閉鎖済みで、Goソースのmain収録とHTTP／B0の実受入を混同しない。main統合済み#5118は異動申請Data表の型付きRuleを既存Evaluatorと生成Go実Binaryで比較し、値が等しくてもRead ID／回数が異なる負例を検出する。main統合済み#5109は実`tei self-build`のBuildPlan確定後のGo byte変更を旧Planで拒否する。両者は実Rule／Buildにおける局所的な証拠であり、正式Provider選択・HTTP Disclosure・Studio Stage1→Stage2の完了ではない。実装選択A→Bの旧Scenario失効を扱った後継#5114は未マージ閉鎖であり、現行製品への適用は実コードと試験で別に確認する。

**次の独立した小PR**は、閉鎖済み設計PR #5096の条件を引き継ぐAP-1のWails Server`InspectSubject`一Routeである（#1158/#1824）。許可されたWorkspace ID／Source Revisionの正常Readと、別AudienceのSource位置非開示・未許可Workspace・旧Revision拒否を、現行CLIの結果とは別のHost Admissionで検査する。**HTTPの成功は非Builtin Providerの実Callではない**。B0（#4635）は実Peer／nonce／SessionとCancellationを持つ子Backendを、N2 Build子起動（main #5059）とは区別して比較する。SL-1の実Function／Provider #4495とStudio非Plugin Stage1→Stage2 #2831/#4583はP0として独立・並行させる。LA-01〜08は既存#1180/#2514のIndependent Expectedへ渡し、第二Resolver／Runtime／Hostを増やさない。

**SL-0→AP-1→実Providerを通す際の時点固定（[起動契約§34](../design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)）**：実装の検収は、Source選択（S）・BuildPlan確定から実Builder入力（B）・実Provider呼出し直前（I）・Consumer受領（C）の四時点を区別する。main #5129では、BuildPlan確定後に同梱原Markdownの**説明だけ**を変えた場合も通常`tei self-build`が旧PlanでのBuildを拒否しArtifactを残さないことを試験に追加した。main #5109の生成Go変更後拒否とは別の負例であり、いずれも正式R2-Pの全出典閉包や実Providerを保証しない。LA-09〜15は既存#1180/#2514へ渡し、説明を戻したA→B→Aのハッシュ一致を以前のOwner受理／Grantの復活根拠にしない。

**次の着手**：mainに収録された実CLI／直接Operation／Studio Host三入口Goテストを現行版で確認し、旧#5119/#5137を再マージしない。AP-1では（旧#5096は未マージ閉鎖）HTTPの公開Audience／Scope／Source位置を呼出し時と返答時に確認し、同じ結果を返せることと開示してよいことを分離する。B0 #4635では実Process世代／Peer／旧Replyの比較を行う。実Provider #4495および非Plugin Stage1→Stage2 #2831/#4583のP0を優先維持し、UIのFocus／Pinについてはmainの実WebView2検査#5012/#5025を基線とし、Dirty Buffer／遅延Replyは未実証として扱う（旧#5122は未マージ閉鎖）。新AdmissionTicket Schema／第二Runtime／汎用Ready APIは追加しない。

**SL-0のBuilder・ToolchainとRule未判定を実際の受入票に接続（[起動契約§35](../design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)）**：現行mainの`builderBinding`は計画のBuilder Digestと起動前に読んだ`-child`ファイルを比較し、違えばL2-bを保留する。Go toolchainは任意の`go.toolchain.closureDigest`で名指しでき、指定と異なるBuildは停止する。ただしProcess Imageの原子的同一性やToolchain全閉包の独立確認は未完であり、SB-C0の`qualified=false`／G4 `UNQUALIFIED`を維持する。Ruleの状態付き判定（main #5142）はNULL・未取得・partialと理由付き未判定を分け、Plugin Resolver診断被覆（main #5147）は現在Grantや実Providerの呼出し証拠ではない。LA-16〜21を既存#1180/#2514へ渡し、別Builder／Toolchain違い／部分取得／診断原因連鎖／Consumer世代を独立に検収する。API公開#1158/#1824、B0比較#4635、P0の実Provider#4495・非Plugin Stage1→Stage2#2831/#4583の優先順位は不変。

**SL-0でRuleの実現可否とPlugin版の真因を分離（[起動契約§36](../design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)）**：main #5166の実Go試験では、Markdownの`not-after ... if present` Ruleを既存Readerから選択でき、P（`validationeval.Decide`）では日付順序・NULL・未取得を評価するが、G（`GeneratePositive`）では未対応として生成を拒否する。Sourceと意味の受理だけで全方式を実行可能にせず、G→P／Dの無言Fallbackを認めない。main #5160では不正なCatalog版が`NO_COMPATIBLE_PLUGIN`と報告される現行挙動がテスト化されたが、真の不在／Major非互換／版不正の改善方針はOwner未決とする。LA-22〜28は既存#1180/#2514へ渡し、P/Gの実選択／実Call・Provider Grant・Audienceを分離する。Operator指紋一本化#5150は、旧HEADで整形検査が停止した後、修正HEADでは専用Scenario CI成功を確認してmainへ統合済み。ただし同HEADのRepository Health／Self-buildは失敗しており全体合格には算入しない。Plan Checkを修正したmain #5151もProvider／Stage2のE2E保証ではない。通常製品CLI actual Provider #4495/#1251・正式Stage1→Stage2 #2831/#4583のP0を維持する。

**SL-0でSnapshotの前提と実行資格を分離（[起動契約§37](../design/STUDIO_BACKEND_LAUNCH_COMPATIBILITY_RECOVERY_CONTRACT_2026-11.md)）**：main #5188はCLI／Studioの実入口が同じSubject・意味・Sourceを扱うGo試験を補強し、#5205は未選択`*.tei.md`に利用可能なLanguageと指定方法を示す診断を追加してWorkspaceのimport cycleを解消した。#5204のPlugin `Supports*`は宣言値の厳密な所属検査であり、Provider実Callや現在Grantではない。#5201は古いLock経由でResolverの原因Codeが失われることを固定したが修正方式は未決。未統合#5198は読取りコマンドのSnapshot前提を揃える実装候補であり、Snapshot一致はSource CAS、A→B→Aの履歴、Owner受理、Audience開示資格に代わらない。未統合#5202のLinux書込境界は二回目Buildの書込隔離候補で、読取共有やG4受理を示さない。#5194の入れ子Go Module検査も候補とし、検査対象化・試験実行・製品合格を分ける。LA-29〜36は既存#1180/#2514へ渡し、AP-1／B0は通常CLI実Provider#4495とStage1→Stage2#2831/#4583のP0と独立して検証する。
### SL-1の最初の実コード差分と、受理可能な証拠（2026年11月15日）

Script Launch §25、ADR-0048、#4495に基づき、最初は**公開済みのpure Function一件を直接Go APIから実Go Providerへ呼ぶ**差分を閉じる。次に、同じ意味契約を通常の製品tei実行ファイルの別OSプロセスから呼ぶ。これは二つの独立Attemptであり、プロセス・Request IDが同じであると仮定しない。Stage1→Stage2セルフビルド、実PostgreSQL Function Read、Studio子Backend/IPC、MCP、外部Effectはそれぞれの別Gateに残す。

**実装の着手前条件**：Ownerが受理したFunction ID、公開Entry/Contract Version、入力の意味型（不在／明示NULL／0を区別）、純粋性を保証する依存範囲、対象Host/選択Go Provider、Independent Expectedを一つの票に固定する。数量2×単価300＝600などの例は候補であり、公開・純粋性・現在Grantが未受理なら製品実装のPASSに使用しない。適格候補が無ければ「公開Entry未受理」と明示してOwnerへ返す。

| 最小差分 | 使う既存Goの接合位置 | 新たに確認すること |
| --- | --- | --- |
| **SL-1a：直接API→実Provider** | 既存のComposition/Host選択、typed Application API、internal/semantic/capabilityinvoke.Invoke | Source/Meaning/Entryと、実ロードArtifact・selected Provider・actual Invoke・BusinessResultを同じAttemptで照合。Generic AdapterのPayload anyを業務値の型検証やHost Grant発行に使わない |
| **SL-1b：通常tei CLI別Process** | cmd/teiの既存起動分岐、#1158のTransport-neutral Operation | 製品Entryから同じ意味契約へ到達し、入力／出力Body・Failureを独立Expectedと比較。cmd/tei/cli_view.goのReferenceExecutor/MemoryUnitOfWorkの成功、実行ファイルのexit0、JSONの値一致だけでは不合格 |
| **SL-1c：負例と実Call位置** | 既存#4495 ELケース、capabilityinvokeのGo focused test、#1180/#2514 | Entry非公開・Grant失効・不正な実InputはAdapter call=0。selected B/actual Aを結果同値でも拒否。実Adapter call=1後の未定義Outcome等はResult契約違反とし、外部Effect不在や安全Retryと判定しない |

**現在の実装証拠と未接続部分**

| 検証範囲 | 実際に確認できたこと | 製品に残る不足 |
| --- | --- | --- |
| **Adapterの呼出し境界**（[#4712](https://github.com/hat47x/tei-platform/pull/4712)） | 事前拒否はcall=0。実Adapter呼出し後の未宣言Outcome・異Capability・Mode不一致・取消はcall=1でResultを拒否 | 公開Entry・Host現在Grant・selected／actual Provider・実Effectは未検証 |
| **Source候補とIndex**（[#4722](https://github.com/hat47x/tei-platform/pull/4722)／[#4742](https://github.com/hat47x/tei-platform/pull/4742)） | Graph／Rule表のNode・Edge・Rule ID、byte spanと原文Fingerprint、旧Indexの失効を既存Source Indexで照合 | Source Indexで物理Formatが`md`であることと、業務上のLanguage／Role／Meaningが選択・受理されたことは別 |
| **管理Rootの予備検証**（[#4751](https://github.com/hat47x/tei-platform/pull/4751)） | ManifestのSource Root、Pinned byte、候補とIndexの一致を実WorkspaceのGo試験で確認 | 実`Workspace.Load`は`TEI_MD_PARSER_UNBOUND`を返す。Readerが製品内部で候補を取得したわけではない |
| **Graph／Ruleから実Go**（[#4729](https://github.com/hat47x/tei-platform/pull/4729)／[#4733](https://github.com/hat47x/tei-platform/pull/4733)） | 条件に応じたGo制御分岐を生成。同じ生成BinaryをNode入力を変えて複数回呼び、独立Expectedと突き合わせる | 正式Target Generator、BuildIntent、Host選択済みProvider、製品`tei` CLI／Stage2には未接続 |

**次のS0のGo差分**：既存WorkspaceのManifest／Source Rootが選択したFrontendと、実Loaderが読んだ同一Source byte／Revisionを、既存Source Indexのspan・fingerprintに接続し、**出典付きのread-only候補まで**を独立に検証する。未選択Frontend、契約版不一致、別Rootの同名Source、古いRevision、曖昧な候補は失敗として扱う。候補の取得をCanonical Meaning／業務Roleの受理、Generator実消費、Provider実Callへ昇格させない。Reader未選択の`*.tei.md`は引き続き`TEI_MD_PARSER_UNBOUND`で拒否する。

**Code Fenceの誤読を防ぐ**：[参照Reader修正PR #4725](https://github.com/hat47x/tei-platform/pull/4725)はチルダ・長いバッククォート・インデント付きコード・未閉じFenceにある例示Graph／Ruleを宣言と誤認しないための候補であり、製品Readerの正式採用とは別である。製品化前にその実Go負例とOwner判断を確認する。SourceのA→B→Aで旧受理／Grantを自動復活させない。Rule変更は生成処理と結果を変え得る一方、Node入力だけの変更は同じArtifactで別Invocation、表示だけの変更はSource byteが変化しても意味処理は不変となり得る。これらを既存#1180/#2514の独立Expectedで別々に検収する。

**次のSL-1のGo差分**：Reader作業を待たず、Owner受理済みのpure Function一件について、既存Composition／Hostが公開Entryと現在Grantを確認し、**実際に選択されたGo Provider**を型付きApplication APIから呼び出す。次に通常の`tei`製品バイナリを別OS Processで起動して同じMeaningを検証する。`BusinessResult.Payload any`や参照Executorの値一致を業務型・実Providerの資格とは扱わない。後続のChild Backend、Plugin Hot Reload、実DB Commitは別の受入とする。

**Task受付と結果の受領は別資格**：mainの[EKI Task結果取得 #4749](https://github.com/hat47x/tei-platform/pull/4749)は、送信受理後の未準備応答を待ち、元Task IDを保ったまま型付きFailureへ渡す局所Adapter証拠である。受理メタデータの取得は成果物bytesの確認、業務意味の受理、外部Effectの確定を意味しない。Script Launchでも、ACK喪失・取消後の再試行判断はHostのReceipt照合に戻し、Task IDを失った新規送信で代用しない。

**証拠の記録**：どのPRにも正確なHEAD、Source／Contract／Generator／Artifact版、選択Providerと実Provider、AttemptとCall Witness、入力の不在・NULL・空集合・0の区別、Result Bodyと独立Expected、Consumer、Effect／未確認事項を短い一枚で残す。別のSource／Build／Host正本やRunnerを設けず、実際に使われた契約の証拠から追跡する。Graph／Rule生成の局所PASSをStudio Stage1→Stage2や製品Entryの成功には数えない。

**CIとPR統合**：過去の#4702 HEADで全体Healthが失敗した要因は、GraphのGo整形（main #4742）、SQL `Opaque`旧期待（main #4741）、Query語彙の試験登録（main #4711）、SQLite Window参照試験のGo整形（main [#4749](https://github.com/hat47x/tei-platform/pull/4749)）として、いずれも別のmain統合で対応済みである。ただし、先行HEADの`go test`／`go vet`成功や修正の統合事実だけで**本PRの全体CIが成功したとは判断しない**。最新mainとの仮統合HEADでCacheとRepository Healthの両方が成功してから、製品完成計画を統合する。mainに統合された[SQLite Window入力検証 #4754](https://github.com/hat47x/tei-platform/pull/4754)は、固定7行のID／系列／順序／整数／明示NULLと独立Expectedを照合する別分野の実Go・実SQLite検査である。入力の欠落と明示NULL、0、指数表記、53bit超整数を混同しない反例を、Script Launchの業務Input契約を設計する際の参考とする。ただし、正式Markdown Readerや製品Providerの業務型受理にそのまま転用しない。

### SL-1：事前確認を実行資格に読み替えない（2026年11月15日）

11月15日にmainへ統合された実装から、**事前確認と実行許可が別の資格であること**を、製品の最初の一機能にも適用する。新たなAdmission Engineや実行状態Schemaを導入するのではなく、既存のSource／Plugin／Host／Consumerで確認済みの範囲を越えて成功と表示しないための実装受入条件である。

| 現行の実装と根拠 | 確認できること | Product Entryへ持ち込めない主張 |
| --- | --- | --- |
| [Plugin文書の事前検査 #4800](https://github.com/hat47x/tei-platform/pull/4800) | 選択PluginのVersionとLock一致。exact-version-pinnedでもSourceはunbound、開示は未検査、Read適格性は未判定 | 文書閲覧権限、Plugin実ロード、業務Functionの公開Entry・実行Grant。**文書用Preflightを業務Executionの判定器として再利用しない** |
| [Studio Inspectorの参照制限 #4806](https://github.com/hat47x/tei-platform/pull/4806) | InspectSubjectのauthorityLimitsを保持・表示し、古いReplyやWorkspace切替で以前の表示を無効化 | 選択対象への書込み・実行・公開権限、実Windows配布物の同一性。Workbench／Bundle同期は別検証（#4812） |
| internal/semantic/capabilityinvoke/invocation.go | 宣言の不正・呼出し前取消でAdapter call=0、呼出し後の不正BusinessResultでcall=1。ResultのMode/Outcomeを検査 | 入力実値の業務型、公開Entry、実Provider選択、Host現在Grant、Payload anyの内容の正当性 |
| [Script Launch設計§27〜28](../design/SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md) | Prepare／Admit／Invoke／Observeと、TEI内評価／Mock／実Providerの証拠を分離 | Product実Go Provider／通常製品tei CLIが実行できたこと。Mock成功・CLI exit 0は代替にならない |

**次の実装票は三つに絞る。**

1. **起動候補のRead-only票**（#4495）：固定Source／Accepted Meaning、Functionと公開Entry、型付き入力、対象・期待Revision、選択Contract／Provider／Artifactの所在を明示。未選択・未受理・Source unboundならそのまま停止し、Plugin文書のPreflightやStudio Inspect成功で補わない。
2. **実起動のAdmission／Call票**（#4495／#951）：既存Hostが現在Principal／Scope／Grant、Entry公開、選択Provider／Artifactを確認してから、型付きAPIで実Providerを呼ぶ。call=0の拒否、selected B／actual Aの同値Result、call=1後の不正Resultを既存ELと独立Expectedで区別。Payload anyの実値やNullabilityの受理は選択Adapter／業務契約側で担い、generic形状検査へ意味受理を移さない。
3. **別ProcessのCLI／Consumer票**（#4495／#1158／#1180）：同じMeaningを通常製品teiの別OS Processで実行。Direct APIとは別Attemptを記録し、実Body／Failure／現在Audienceを検収する。参照view -execute、Mock、Plugin Lock、Studio InspectのPASSを製品実呼出しのPASSへ合算しない。

データ依存を持つFunctionでは、どこか一つのEnvironmentでDB Aliasが結合していることと、今回選んだTarget／Environmentで実際に接続できることも区別する。未結合DB要件の警告を追加する[PR #4811](https://github.com/hat47x/tei-platform/pull/4811)は現時点で検証中の候補であり、**マージ前に製品の既存保証として数えない**。実DB Read、部分取得・NULL、HostのRow Scopeは#2586/#2622が別に受理する。

**優先順位は据え置く**。全体P0はStudio Stage1→Stage2セルフビルド（#2831/#4583）、並行P0は一つの公開pure Entryの実Go Provider→通常tei CLI（#4495）。Local Child/Remote、Trigger、Updater、Plugin文書の正式Read、新Mock Runtimeは先行依存としない。新Go Product E2E、実Windows配布物、外部DB Effect、Ownerの業務意味受理は本追補ではNOT RUN。


## 不足を戻す担当

各担当Issueには以下の範囲に属するG項目の作業と正負の受入条件を引き渡す。実装・実行結果はそのIssueと既存のEvidenceに記録する。

| 範囲 | 製品として揃えること | 担当 |
| --- | --- | --- |
| G01〜G07 | 全体案内、通常の編集、診断、改名、復旧、実行の追跡、応答性能 | [#1124](https://github.com/hat47x/tei-platform/issues/1124)、[#3788](https://github.com/hat47x/tei-platform/issues/3788)、[#3789](https://github.com/hat47x/tei-platform/issues/3789)、[#3790](https://github.com/hat47x/tei-platform/issues/3790) |
| G08〜G12 | 同一性、関係、数量、欠測、項目Profile、時点、導出と失効 | [#2495](https://github.com/hat47x/tei-platform/issues/2495)、[#3816](https://github.com/hat47x/tei-platform/issues/3816)、[#3791](https://github.com/hat47x/tei-platform/issues/3791)、[#2600](https://github.com/hat47x/tei-platform/issues/2600) |
| G13〜G20 | 実Read、状態・判断、合成保証、編集保存、現在権限、外部連携、公開確認 | [#2586](https://github.com/hat47x/tei-platform/issues/2586)、[#1125](https://github.com/hat47x/tei-platform/issues/1125)、[#3278](https://github.com/hat47x/tei-platform/issues/3278)、[#1174](https://github.com/hat47x/tei-platform/issues/1174)、[#3796](https://github.com/hat47x/tei-platform/issues/3796) |
| G21〜G29 | 実Provider選択、入出力、構成と配布、独自差分、外部接続、移行、Target、初回利用、セルフビルド | [#1251](https://github.com/hat47x/tei-platform/issues/1251)、[#951](https://github.com/hat47x/tei-platform/issues/951)、[#3267](https://github.com/hat47x/tei-platform/issues/3267)、[#3270](https://github.com/hat47x/tei-platform/issues/3270)、[#1168](https://github.com/hat47x/tei-platform/issues/1168) |
| G30〜G36 | 確定データ、固定・可変・混合版面、PDF/Excel、既存帳票、画像移行、発行運用、符号の実読取り | [#3815](https://github.com/hat47x/tei-platform/issues/3815)、[#3820](https://github.com/hat47x/tei-platform/issues/3820)、[#3798](https://github.com/hat47x/tei-platform/issues/3798) |
| G37〜G44 | 標準表現型としての複合表現、混在座標、多次元、時間、入力・逆編集、Browser/Wasm、科学処理 | [#3827](https://github.com/hat47x/tei-platform/issues/3827)、[#3821](https://github.com/hat47x/tei-platform/issues/3821)、[#4133](https://github.com/hat47x/tei-platform/issues/4133)、[#4122](https://github.com/hat47x/tei-platform/issues/4122) |
| G45〜G48 | 色、精度、資源の上限と代替、VR、実物との状態照合 | [#3827](https://github.com/hat47x/tei-platform/issues/3827)、[#3790](https://github.com/hat47x/tei-platform/issues/3790)、[#1174](https://github.com/hat47x/tei-platform/issues/1174) |
| G49〜G53 | 高度B2C、多言語・アクセシビリティ、AI拡張、用途限定文脈、CLI/MCP/VUIの共通操作 | [#3509](https://github.com/hat47x/tei-platform/issues/3509)、[#3796](https://github.com/hat47x/tei-platform/issues/3796)、[#2515](https://github.com/hat47x/tei-platform/issues/2515)、[#2516](https://github.com/hat47x/tei-platform/issues/2516)、[#3503](https://github.com/hat47x/tei-platform/issues/3503) |
| G54〜G60 | 独立検証、証拠の帰属、多分野の要求化、案件対応率、総費用、資産の再利用、計画の更新 | [#2514](https://github.com/hat47x/tei-platform/issues/2514)、[#1163](https://github.com/hat47x/tei-platform/issues/1163)、[#3791](https://github.com/hat47x/tei-platform/issues/3791)、[#3796](https://github.com/hat47x/tei-platform/issues/3796)、[#2018](https://github.com/hat47x/tei-platform/issues/2018) |
| G61〜G64 | DSL文法と互換、外部形式の取込・交換、オフライン再同期、継続運用と復旧 | [#1063](https://github.com/hat47x/tei-platform/issues/1063)、[#2621](https://github.com/hat47x/tei-platform/issues/2621)、[#3270](https://github.com/hat47x/tei-platform/issues/3270)、[#2597](https://github.com/hat47x/tei-platform/issues/2597)、[#3796](https://github.com/hat47x/tei-platform/issues/3796) |
| G65〜G71 | 業務資産、検索、通知、外部契約、継続イベント、デバッグ・模擬実行、利用者のライフサイクル | [#4426](https://github.com/hat47x/tei-platform/issues/4426)、[#4427](https://github.com/hat47x/tei-platform/issues/4427)、[#4428](https://github.com/hat47x/tei-platform/issues/4428)、[#2542](https://github.com/hat47x/tei-platform/issues/2542)、[#3274](https://github.com/hat47x/tei-platform/issues/3274)、[#3789](https://github.com/hat47x/tei-platform/issues/3789)、[#3412](https://github.com/hat47x/tei-platform/issues/3412) |

## G49 B0の実装Referenceから実Source・Native Consumerへの引渡し（2026-11-12）

- **実装済みの限定足場**：[PR #4506](https://github.com/hat47x/tei-platform/pull/4506)は修正後HEADの健全性・Cache・既知脆弱性CI成功を確認してmain統合（`8c3d1a55`）。`tools/reference-b2c-change-inspector`の明示Graph/Scopeの到達検査と固定Expectedは**合成read-only Reference**であり、実Nativeコード解析・公開許可・Provider/Commerce/Experiment検証ではない。G49受入計画の[PR #4499](https://github.com/hat47x/tei-platform/pull/4499)と[PR #4512](https://github.com/hat47x/tei-platform/pull/4512)もmain統合済み。
- **B0-S0（同一Loadの実Source観測）**：[PR #4517](https://github.com/hat47x/tei-platform/pull/4517)はexact-headの全体健全性・Cache・既知脆弱性検証成功後にmain統合（`2374f338`）。既存`workspace.Load`＋`BuildSourceIndexFromLoad`＋`InspectionFromLoad`を用いる**局所Go試験**で、Source A→B→A、Load後のファイル変更、Workspace manifestの同一性差分、別Root拒否と実Queryの出典を対照する。**製品B0-S1の実Native依存解析の合格ではない**。Load時ByteからのIndexは原子的な多ファイルSnapshot、Imported Provider/Native依存の閉包、Current Grantの証明ではない。
- **B0-S1（次の実装担当）**：[Issue #4518](https://github.com/hat47x/tei-platform/issues/4518)で`Workspace + Qualified Subject/occurrence + Source revision + 既存Operation/Context`から、案件が明示したNative Touchpoint/Action/Consumer Bindingへ限定投影する。Required/Observed/Resolved/Unresolved/Excludedを**用途と質問に相対的に**診断し、未探索のJS/CMS/Edge/外部Tagはunknownとする。Native UIへの項目追加はDB列の作成ではなく、Data/Function Mappingがない場合は未対応として診断する。明示的なData項目削除でMappingが失効してもUI要素は残り得る一方、未取得・非開示・partialを削除として扱わない。`SELECT *`／`alias.*`／`NATURAL JOIN`などの暗黙的な列集合・結合条件への依存は、明示列の参照が0件でも影響候補とし、実Schema/Queryの取得範囲が不足する場合はunknownとする。既存#2582のClaim単位Required Setに従い、影響外と証明できるContextは保持し、影響範囲だけを再検証する。未MappingのNative UI項目はDraftでは保持できても、必要なBindingを欠いた公開・業務Actionを検証済みとしない。削除・改名・DDLの実Applyは既存Data/Schema Ownerが所有し、Source/Scope/Selectionの再確認なしに自動付替えしない。完全なNative Source解析、第二Source Reader/Graph Resolver/Hostは実装しない。
- **問い相対の欠落と再利用**：既存[#2582](https://github.com/hat47x/tei-platform/issues/2582)のRequired Set／Consumer Scopeを利用し、当該変更の到達先・明示的Requiredに接続する欠落は未解決として保持する。一方、無関係な未知Node・未解決Edgeが存在するだけで全Consumerを失効させない。[PR #4530](https://github.com/hat47x/tei-platform/pull/4530)は合成read-only検査の過剰失効と必要依存の見落としを正負テストで抑止し、exact-headの健全性・Cache・既知脆弱性CI成功後にmain統合済み（`8d9a0d97`）。`WHERE`にのみ使う非表示列、`NOT EXISTS`等の否定条件、集計・結合とSource/Schemaの版差を「明示出力列0件＝影響なし」に含めない。実Queryの演算依存は#2387のOwnerに残し、B0に第二SQL解析器を加えない。
- **次の最小受入（B0-S1a）**：[#4518](https://github.com/hat47x/tei-platform/issues/4518)で公開合成Native PDPの**一つの明示Field Binding**を、既存Workspaceの**一つの実Canonical Subject/Source occurrence**と同一Reader結果で照合する。#3262が所有するRequired Universeの列挙根拠、#2586のProvider選択と実Read、実Consumerに到達したResult Scope、#4503/#2516の用途別開示・実送出、現在のGrantを、別の証拠欄として記録する。Toolのページ・End markerが欠落した応答や未取得Native/CMS依存を「閉包済み」としない。対象が一意に解決してもSourceの値Span・書込権限は別条件であり、[#4533](https://github.com/hat47x/tei-platform/issues/4533)の操作単位逆編集の実測をNativeからの自動Source書戻し許可へ拡大しない。**新CLI/API名、第二Reader/Registry、Nativeの必須TEI Runtime、公開Gateは導入しない**。
- **B0-S1aの局所到達とB1aの受渡し（2026-11-13）**：[PR #4540](https://github.com/hat47x/tei-platform/pull/4540)をexact-head CI3種成功後にmain統合（`305e024c`）。一つの宣言Native PDP Fieldを実Workspaceの`semantic-attribute`／Source occurrence／Source bytes／Workspace Rootへ同一Loadで束縛し、誤Target・Source driftを拒否する**Go test-onlyの証拠**を得た。Native/Edgeは入力者が申告した仮説で、ブラウザー・Provider・Consumerの実動作を観測していない。対して[#2586](https://github.com/hat47x/tei-platform/issues/2586)の[PR #4539](https://github.com/hat47x/tei-platform/pull/4539)は、実PostgreSQLへのEnvironment束縛・Query実行・Raw Rows→Function Data Context・独立Reader比較を使い捨てDB上で試行し、exact-head CI成功後にmain統合済み（`ee1af44c`）。**製品の`tei` CLIではなく、別プロセスの限定Reference CLIによる受領**である。Product `tei` CLI接続、独立Native Consumerの実受領、接続先の物理同一性（現状host-asserted）、現行Grant・Owner受理は未実証として残す。B0-S1aの`native-pdp.order-id → semantic-attribute:sales.order.order_id`は、#4539の`order_entry.recent_orders`/`:from`や物理`public.orders.order_id`との正式なMappingではない。近似名・同じSQL Source digestだけでFunction入力・物理列・Provider対象を自動結合しない。#4539では`:from`がQueryの`input`として未宣言（`declaredByQuery=false`）であるため、Native側のTyped Input契約の確認が残る。次のB1aは**同一Attemptの実Product Query/Providerと、Nativeまたは独立CLI Consumerが実際に受け取ったContextを照合**する。RLS/列権限/マスク/0行での`unestablished`、ページ・終了マーカーの欠落、実ReadとBrowser公開・業務Effectの違いを保持し、別Readerによる一致を実Consumer受領へ昇格しない。
- **R0/R1の負の依存とSource存在（B0-S1追加負例）**：入力に`BaseRevision/CandidateRevision`があっても、B0-Rの`Edges`は一式の申告Graphであり**両世界のEdge差分や実Consumed Closureの観測ではない**。R0にあった`Source → Function/Consumer`の依存辺をR1で削除・別Subjectへ再束縛した場合、R1の変更元からConsumerへ到達0でもConsumer不変とは断定しない。旧辺の依存先を変更候補・未知のまま保ち、R0/R1のSource・対象・方向・Scopeを揃えた比較と独立Expectedを[#4518](https://github.com/hat47x/tei-platform/issues/4518) N16／[#2582](https://github.com/hat47x/tei-platform/issues/2582)へ渡す。`SELECT *`／否定述語／Dialect別SQLの新規候補追加も**変わらない元SQL byteだけでは選択・結果・作用の不変を証明しない**ため、Queryの実選択は#2622/#2586 HJ-08で扱う。また、単一Subject参照0件とSubject不存在を分け、[実測Issue #4577](https://github.com/hat47x/tei-platform/issues/4577)と修正PR #4584のCI・採用状況を独立確認する。第二Query Parser/Graph Solver/Registryを作らない。
- **問い相対の旧依存辺の検収（N16/N17）**：明示されたR0-only依存辺から旧Consumerを影響候補に残す限定Referenceは[main #4591](https://github.com/hat47x/tei-platform/pull/4591)へ統合済み（`8709aae7`）。ただし、旧辺すべてを変更元とみなすと**当該Required Consumerとは無関係な削除辺**や、別のRelease ConsumerへのR1下流分岐まで誤って失効させ得る。[PR #4592](https://github.com/hat47x/tei-platform/pull/4592)でR0-only辺とR1候補辺の**問い相対Required上流集合**だけを旧辺変更の対象とする修正・二段旧依存削除・無関係下流Siblingの独立Expectedを提案中。明示`changed`の通常到達は別扱いにする。N16/N17の申告Graph上の成功はR0/R1実Source読取・Imported Source／Native/Queryの完全な依存探索の証明ではない。#3262がRequired Universeの取得根拠、#2582が局所再取得を所有する。#4592のCI・main統合は別確認。
- **Studio同一HEADの実Windows B0とBuild Intentの断線**：[統合Draft #4587](https://github.com/hat47x/tei-platform/pull/4587)の実Windows CIは、一つのexact HEADで生成Inspector readerの使用・配信Workbench JS byte一致・Native Wails WebView2を確認（[#4583 B0](https://github.com/hat47x/tei-platform/issues/4583)）。ただしDraftであり、**N0 Inspector Contract→N1 PoC B Go/TS→N2 esbuild/Embed→N3 nested Studio Wails exeの全工程がTEI BuildIntent/実Builderへ宣言・実消費対応したことは未実証**。B1には変更前のBuild依存辺が削除された場合の実Consumed残留、無関係な旧Sourceの過剰再Build、別HEADの成功の継ぎ合わせをそれぞれ独立負例として追加する（[#4583のBD-07〜09](https://github.com/hat47x/tei-platform/issues/4583#issuecomment-6846881454)）。B2 Stage1実TEI Process→Stage2 Desktop、B3 Stage2 WebView2は別のNOT RUN資格。TEI Studio全体のセルフビルドはB2Cの新Hard Gateを増やさず#2831/#1824で優先する。
- **B0-Rの宣言入力検証とStudio優先度（2026-11-13）**：既存のJSON入力が重複項目・Case Shadow・容量超過により曖昧だった問題は[main #4565](https://github.com/hat47x/tei-platform/pull/4565)で修正済み（`8a119731`）。UnicodeのSimple Fold迂回は[#4580](https://github.com/hat47x/tei-platform/pull/4580)の未マージ修正候補で、確認時のexact-head健全性CIは**gofmt不一致で失敗**。製品全体の最優先は[#2831](https://github.com/hat47x/tei-platform/issues/2831)／[#1824](https://github.com/hat47x/tei-platform/issues/1824)の**実TEI Studio Desktopセルフビルド**であり、[生成Inspector部品の実Windows試行Draft #4572](https://github.com/hat47x/tei-platform/pull/4572)と[生成Desktop WebView2検証Draft #4579](https://github.com/hat47x/tei-platform/pull/4579)は区別して扱う。全体計画main[#4581](https://github.com/hat47x/tei-platform/pull/4581)に不足したBuild graph／実Builderの接続工程を反映済み。B2Cは同じFunction Contextの異質Consumerとして並行し、Studio P0への新規Hard Gateを設けない。
- **順序と証拠の上限**：#2586の実Function→実Query/Provider→Rowset→Function Data Context→ConsumerをP0に維持し、B0-Sの完成をHard Gateとしない。#4350 DF-X06は標準表現/Nativeの異質Consumerの同一意味を検査。#4496/#4515は固定Subject/Scope/Read epoch/操作資格を所有し、#4507/#4514のSource-only driftでは同byteの復帰とSource/Authorityの現行性を区別する。B1a実Native Consumer、B1b実Commerce Effect/別Read、B2実配信/Exposure/Consent、B3のCVR・利益因果評価は**別の未実証資格**。
- **性能・プラグイン・Evidence**：#2018でH0 Native単独/H1 TEIのBuild時契約/H2任意Experiment Adapterを、同じ変更・独立Expectedと導入/改訂/運用費用で比較する。#951/#4198/#3278のProvider/Graph/Traitは実選択と実呼出しを照合。#4307/#2516に従い非公開Evidence・その存在情報をGitHubや外部Harnessへ必須送出しない。公開の権限、Consent/Grant、現在価格、実注文Effect、売上改善をB0で宣言しない。

<!-- b2c-b1a-principal-scope-20261113 -->
## G49 B1a：独立Native Consumerに実データを渡す際の主体と対象範囲（2026-11-13）

**最初の対象を実Functionの意味に合わせる。** main [#4539](https://github.com/hat47x/tei-platform/pull/4539)は`order_entry.recent_orders`の実PG Read→Function Data Contextに限定し、main [#4540](https://github.com/hat47x/tei-platform/pull/4540)は申告されたNative PDPの`order-id`項目→Semantic Source一項目の束縛に限定する。両者を同じ実Frontend Functionとみなさない。B1aの最小候補は非公開の合成データによる**read-only注文履歴Consumer**とし、Function/Query Input/Result列・出典・用途・実Consumer受領の対応を[#2586](https://github.com/hat47x/tei-platform/issues/2586)の同一Attemptで検証する。商品詳細PDPを構築するときは、その目的に適した受理済み商品Data/Queryを別途必要とする。

**B2C利用者と組織所属を分ける。** 公開閲覧者・ログイン済み消費者・ゲスト購入者・業務運用者が、必ずしもOrg Membershipを持つとは限らない。組織モデルをB2C利用の必須条件にせず、案件が必要とする場合に限り[#2386](https://github.com/hat47x/tei-platform/issues/2386)へ結び付ける。認証方法/本人の出所は[#3412](https://github.com/hat47x/tei-platform/issues/3412)、Browser Session/要求保護は[#1043](https://github.com/hat47x/tei-platform/issues/1043)、読み取り用途・行/列Scopeは[#1656](https://github.com/hat47x/tei-platform/issues/1656)、現在の実行Authority/Effectは[#1174](https://github.com/hat47x/tei-platform/issues/1174)、用途別開示は[#2516](https://github.com/hat47x/tei-platform/issues/2516)を既存Ownerとする。TEI Coreに第二のUser/Identity/Tenant台帳を追加しない。**`:from`日時条件やFrontendが指定したcustomerIdは、利用者本人の注文を選別する認可根拠ではない。** Host側の実Principal/業務Subject対応とRead Grant/行範囲が確認できない注文データを一般向けAPIへ公開しない。

**順序・検証資格**：製品`tei`/既存Application経路から実Provider/Raw Resultを読み、独立Native Consumerが同じ値・Presence・Scope/Completeness・版を受領したことを確かめる。Public/Privateの実Hostアクセス、Guest/Checkout、Consent/Experiment、旧Clientの契約版と価格の再確認は後続の独立資格で検査する。既存B0-S1 [#4518](https://github.com/hat47x/tei-platform/issues/4518)を新しい公開APIの前提Gateにしない。独立Expected候補BC-43〜48と担当・停止線は[親Issueの引渡し](https://github.com/hat47x/tei-platform/issues/3509#issuecomment-6847438801)を参照し、別のテスト正本を作らない。

**状態**：B1a Product CLI→実Native Consumer、顧客単位の実Grant/Read、公開Browser、Commerce/売上効果の検証は**NOT RUN / NOT ACCEPTED**。Studio Desktopセルフビルド[#2831](https://github.com/hat47x/tei-platform/issues/2831)の最優先を変えず、Runtime-zeroを選べるHeadless/外部Frontend方式を維持する。

<!-- b2c-b1a-qualified-consumer-20261113 -->
### G49 B1aの受入を二段化する（2026-11-13）

**B1a-I（非公開・隔離した機能結線）**：最初は受理範囲が明示された合成注文履歴のread-only Consumerでよい。既存製品`tei` CLI／型付きApplication Operationから、選択した実Provider、実Queryと宣言済みの型付き入力、実Rowset、Function Data Contextを同一Attemptでつなぎ、**別プロセスのNative Consumer自身が受領した内容**を独立に確認する。DB Reader間の値一致、Reference CLIの成功、宣言Native FieldのSource束縛は、それぞれ異なる証拠であり合算しない。`order_entry.recent_orders`の`:from`は現状Query側の入力宣言を満たしたと認定されておらず、日時条件は本人の注文Scopeも与えない。Query入力・物理列・Function/Result Fieldの対応が未受理なら`unmapped/unsupported`として止める。

**B1a-P（一般消費者への公開資格）**：B1a-Iの成功とは別に、Hostが検証した認証Principal（Issuer/Subject/Session）と業務Customerの対応、対象注文との所有関係、現在Read Grant、行・列・用途別の許可済みProjection、Cache/旧Client/失効の整合を既存Ownerで確認する。公開閲覧者、ログイン済みConsumer、Guest、運用者は同じ主体・権限ではない。消費者に組織Membershipを一律要求せず、Cookie、Frontend指定customerId、メール一致、匿名ID、日時フィルターから権限を導かない。Guest→登録済みConsumerの履歴結合も、選択したCommerce/Hostの明示的なLink・許可なしに行わない。認証だけでは開示資格にならず、診断用ContextのConnection/Provider等をNativeレスポンスやログへそのまま流さない。

**表示・Cacheの受入**：`NULL`／未取得／非開示／RLS・Mask／0行／partial／completeをConsumerまで区別し、Claimに必要な完全性が確立していないとき「全注文を表示」「購入履歴なし」を断定しない。公開資格を確認できない場合は注文内容を返さず、画面側にも無根拠な「0件」成功を表示させない。本人専用レスポンスを匿名CDN/共有Cacheへ混在させず、Client・SSR・Edgeで版、Scope、失効を別々に検証する。実験割当・分析識別子は認証Subject/業務Customerと自動統合せず、Consent/目的・保持条件を別途扱う。

**受領票と停止線**：`Attempt / Source・Function・Query revision / typed parameter declaration / selected→actual provider / Rowset・Context presence / Scope・Completeness / Native receipt / Current Grant / allowed disclosure / cache・release / independent expected / status`を、観測済み・未観測・非開示ごとに分けて記録する。BC-43〜54の独立Expected候補は[親#3509](https://github.com/hat47x/tei-platform/issues/3509)から[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ帰属し、新たなテスト正本は作らない。B1a-Iは[#2586](https://github.com/hat47x/tei-platform/issues/2586)の製品実Read・独立Consumer、B1a-Pは[#3412](https://github.com/hat47x/tei-platform/issues/3412)/[#1043](https://github.com/hat47x/tei-platform/issues/1043)/[#1656](https://github.com/hat47x/tei-platform/issues/1656)/[#2516](https://github.com/hat47x/tei-platform/issues/2516)のHost・認可・開示を利用する。H0 Native単独／H1任意の型付き契約／H2任意の実験Adapterの費用・実測比較は後続B3へ置き、B1a-PやB2C専用Pluginを[#2831](https://github.com/hat47x/tei-platform/issues/2831) Desktop Self-Buildの新Hard Gateにしない。**製品CLI→実Native受領、公開認可、Browser/Cache、Consent、Commerce/売上効果はNOT RUN / NOT ACCEPTED**。

<!-- b2c-crosslane-acceptance-20261114 -->
### G49 横断適合：変更・配布・利用者・実作用の境界（2026-11-14）

B1a-I/Pの二段受入を維持しつつ、独自Frontendを「TEIが制約しない」と認定するための比較対象を明示する。業務上の正本は受理済みSemantic/Function/Action、実行権限はHost/Providerに残し、Frontend・CMS・Edge・実験・検索等の実装先は案件が選ぶ。Journey/Experiment/Releaseは既存の型付きGraph・Operation・Artifact/Scenarioの**投影とAdapter**で扱い、B2C専用のIdentity台帳、Analytics正本、実行Host、Lifecycle/Graph Engineを増やさない。

| 境界・既存担当 | 同じ実Consumerで確認する最小の正負対照 | 判定上限 |
| --- | --- | --- |
| **B1a-Iの値・型・Consumer受領** [#2586](https://github.com/hat47x/tei-platform/issues/2586)/[#2497](https://github.com/hat47x/tei-platform/issues/2497) | 選択した実Providerと実Invoke、Queryの宣言されたTyped Param、Raw Rowset→一個のContext→別プロセスNative/標準表示の受領を対応付ける。NULL・欠損・0行、投影後の同値重複と業務Identity、金額の単位・精度を区別し、同型Fieldからの暗黙Mappingを拒否 | 現在の#4539実PG参照と#4540宣言Native Fieldは**別Attempt**。通常`tei` CLIと実Native受領は未実証 |
| **個人データの開示非干渉** [#1656](https://github.com/hat47x/tei-platform/issues/1656)/[#2516](https://github.com/hat47x/tei-platform/issues/2516)/[#1043](https://github.com/hat47x/tei-platform/issues/1043) | 同一公開資格・公開Sliceで、W0/W1の**非公開の注文行だけ**を変更。HTTP Body/metadata、件数・aggregate・sort/page、diagnostic・digest・ログ・Cache key・Tool応答に秘密由来の差が出ないことをConsumerまで比較。private注文はHost検証済みPrincipal↔Customerと現在Grant/行列/用途Projectionが揃ったときだけ返す | Sourceの参照可能性や`Disclosure.Rows/Columns`欄の存在、Reader成功は実Policy適用の証明ではない。比較していない媒体・Timing等は保証しない |
| **変更影響とAI編集の対象** [#4518](https://github.com/hat47x/tei-platform/issues/4518)/[#2582](https://github.com/hat47x/tei-platform/issues/2582)/[#4515](https://github.com/hat47x/tei-platform/issues/4515) | R0でのみ存在した旧依存辺を消したR1、Required Consumerと無関係な旧辺、C0文言内の実効価格/Consent変更を対照。必要上流は失効対象へ残し、無関係なRelease/Consumerは過剰失効させない。AIの旧Source/Pin、未保存Dirty Buffer、現在選択・実際のBuild入力を区別し、無音Applyしない | #4591は申告Graph局所、#4592は修正提案。実Source全Universe・Source読取り時点・Current Grantの証明に拡大しない |
| **配布・Plugin実消費と撤去可能性** [#1016](https://github.com/hat47x/tei-platform/issues/1016)/[#951](https://github.com/hat47x/tei-platform/issues/951)/[#2604](https://github.com/hat47x/tei-platform/issues/2604) | H0=Native単独、H1=任意の型付き契約/生成Adapter、H2=任意のExperiment Adapterを、同じFrontend・機能・変更で比較。Selected→実Artifact同梱→実Invoke→Consumer受領、未選択Plugin除去、旧Client/SSR/Edge/Service Worker/Cacheの残存と実公開版を別観測 | 「Runtime-zero」はTEI標準UI/常駐JS/Router/Gatewayを必須としない選択であり、外部API/認証/運用コストがゼロという意味ではない。完全静的Compositionは#1016後続P1 |
| **注文Effect・実験計測** [#1174](https://github.com/hat47x/tei-platform/issues/1174)/[#1129](https://github.com/hat47x/tei-platform/issues/1129)/[#1180](https://github.com/hat47x/tei-platform/issues/1180) | Click/Intent→Cart/Checkoutの外部Provider実Call→確定Outcome/Receipt→独立Readを分ける。Timeout/ACK喪失/Hookの`effectUnknown`・`lookup_first`を成功や安全な再送へ降格しない。Assignment≠Exposure≠Outcome、Consent/欠測/SRM/返品等のguardrailを別に観測 | B1b実Commerce、B2実験、B3 CVR/粗利益の因果主張は後段。現在のHook局所Draftを二重購入防止の実証にしない |

**実行順・費用比較**：①#2586の製品`tei`入口からB1a-Iの**同一Attempt実Consumer受領**を一件成立させる、②B1a-PのPublic/Private・W0/W1負例を既存Host/Projectionで実行する、③B2の旧配布版・Effect Unknown・Experimentを段階的に追加し、④#2018でH0/H1/H2の初期実装時間、変更時間、追加JS/Request/TTFB/CWV、誤検出・欠測、Provider/運用費用を同じ独立Expectedで比較する。計測していないCVR改善やRuntime overheadの不存在を主張しない。

**証拠管理・優先順位**：Source/Context/Consumer/Release/Grant/Effect/測定の観測時点と受領媒体を分け、同じID・digestでも現在性や許可を推測しない。非公開注文、Session、内部接続先、非公開Sourceの存在・件数・HashをGitHub/外部AIへ必須送出しない。独立Expectedは[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)の既存Scenarioへ返し、新しいCase台帳は作らない。#2831/#4583のStudio Desktop Self-Build（B1 BuildIntent→B2 Stage1再入→B3 Stage2実Windows受領）を全体P0に据えたまま、B2Cは並行する異質Consumer試験とする。**本節は設計・検証計画であり、各実Target・公開認可・Commerce/Experimentの新しい実行結果を含まない**。

<!-- b2c-b1ai-executable-handoff-20261114 -->
### G49 B1a-I 最初の実行可能な一往復：実装受渡しと停止条件（2026-11-14）

**固定する対象**：`order_entry.recent_orders`の意味に合う、**非公開・使い捨て合成注文履歴のread-only表示**だけを選ぶ。商品詳細PDP用の`native-pdp.order-id`申告Binding（main #4540）はこのQueryへのFunction/Result Mappingではない。注文を一般ユーザーへ公開しない。現在の`:from`は実行用のtyped値を渡せても、Query Source側の`input`宣言・意味束縛が未受理なので、宣言不足を暗黙defaultで埋めない。

| 一Attempt内の検証箇所（既存Owner） | 最小の実Witness | 不合格・不明の場合の扱い |
| --- | --- | --- |
| **A. 対象と入力**（#2586/#4518） | 同一Workspace Load、実Canonical Function/Query Subject、Source occurrenceと版、`:from`のQuery宣言・型・Runtime値・物理SQL選択を照合する | Subject不在を参照0件と扱わず、Queryのinput未宣言ならTyped Contractは`unmapped`。製品`source-index`の不存在拒否はmain [#4584](https://github.com/hat47x/tei-platform/pull/4584)で修正済みだが、Function→Queryの意味受理は別 |
| **B. 製品入口と実Provider**（#2586/#1251/#951） | 通常の実`tei`プロセスまたは既存Product Application Entry、選択Provider/Instanceと**実Call**、実SQL/param shape、物理DB/Session/RLS・Reader時点の観測を区別する | `tools/reference-browse-read`のみ、Provider B選択→A実Call、別物理DBで偶然同値なら製品接続PASSにしない |
| **C. 一つのResult/Context**（#2506/#2497） | 同じ実Rowset→同じContext identity/epoch/Result Sliceから、列Attribute、Presence（missing/NULL/value）、Decimal scale、Grain、Scope、Coverage、完受（EOF/End marker）を保持する | DB cursor完走でも業務母集団/RLSの`complete`は導かない。masked NULL/SQL `NULL AS`/未取得を合算しない |
| **D. 独立Consumerの実受領**（#2586/#2516） | 同じContextを標準表示と独立Native read-only表示の**二Receiver**へ渡し、受信側でAttribute ID・行/Cell・Result revision/ScopeをReadback。Nativeへは事前に許可した投影だけを渡す | 二度のDB Read一致、DB Readerだけの照合、Reference JSON全項目のNative返却はConsumer資格としない。Connection/User/内部Instanceや非公開存在情報を出さない |
| **E. 独立Expectedと結果票**（#1180/#2514/#1163） | 事前固定の正常3行/期間境界/0行、列欠落・NULL/RLS、Provider不一致、Response途中切断を既存Scenarioで比較し、`declared / selected / invoked / consumed / verified`を分けて記録する | 一部局所PASSをB1a-I全体PASSにしない。未実行はNOT RUN、契約不支持はUNSUPPORTED、未確立はUNKNOWNのまま保持 |

**対象外を明示**：ここではPublic APIの認証、本人の顧客ID連携、現在Read Grant、共有CDN/Browser Cache、安全なGuest→会員結合、Cart/Checkout、公開Release/Experiment、CVRを合格とみなさない。B1a-Pでは、許可済みAudience/ScopeのPublic Sliceを先に確定してから、W0/W1で非公開注文だけを変え、Body/DOM/ARIA/metadata/count/sort/aggregate/diagnostic/log/cacheの許可外差分を実Host・Consumerで確認する。情報を伏せたことによる存在・件数の漏えいにも注意するが、Timingや外部経路を未測定のまま非干渉完了とは主張しない。

**変更・配布と次段の接続**：B0-RのR0-only辺／R1 Required上流は[#4592](https://github.com/hat47x/tei-platform/pull/4592)（未統合候補）と#2582へ、Source Root実在性が構造Fingerprint不変でも変わる負例は[#4616](https://github.com/hat47x/tei-platform/pull/4616)の限定提案へ接続する。Desktop側の[#4587](https://github.com/hat47x/tei-platform/pull/4587)ではTEI実CLI `self-build` primitive子起動→Inspector TS→実Windows WebView2まで限定成功したが、正式BuildIntentとStage1→Stage2は未実証。この成功をB1a-IのProduct Read/Native Consumer実証に合算しない。H0/H1/H2の実際の時間・通信/JS・運用費用比較は#2018で別々に採点する。**新CLI名・Host・Identity/Analytics/Commerce正本・Scenario台帳・Hard Gateは作らない**。

<!-- b2c-bridge-entry-temporal-20261114 -->
### G49の実装境界：Query Read・公開Entry・履歴と現在資格を分離する（2026-11-14）

**B1a-I最初の不足辺は公開Functionの新設ではない。** 現行の[Script Launch §12〜13 Draft #4618](https://github.com/hat47x/tei-platform/pull/4618)のSource照合では、製品`cmd/tei/cli_commands.go`に公開Function起動Verbはまだない。既存`semantic/capabilityinvoke.BusinessResult`の単一Output/Outcome区分はあるが、Outputの型/NULL/単位を自動的に全検証するわけではなく、`command.Result`は業務Command/Effect側の契約である。したがって、Read-onlyなB1a-Iではまず#2586の**既存Product Query/Function Data Contextの型付きOperation**を実Providerへ結び、その実Resultを検査する。次に、合意したCLI契約の**薄いAdapter**から同じOperationを呼ぶ。公開Business Entryの発見・発行、pure Functionの正式起動、Command/Effectは[#4495](https://github.com/hat47x/tei-platform/issues/4495)/[#3422](https://github.com/hat47x/tei-platform/issues/3422)の別資格であり、B1a-Iへ公開Entry/Action実行エンジンを偽装・新設しない。

**A〜Eの前段差分と独立Expected**：①現行`order_entry.recent_orders`の`:from`をQuery Sourceで型付き宣言できないなら`unmapped`を維持し、実行時paramだけから公開Operationと認定しない。②同じResult bodyを返しても、選択Provider B→実Invoke A、別物理DB・別RLS Session、欠落したCompletion/Consumer Readbackは不合格。③`source-index`のCanonical存在成功とStudioのPinが成立していても、公開Entry・Provider・現在Grant・実Consumerは未検証。④実ReadのContextは、`Connection.User/Database/Instance`等の内部診断を含むため、Native用の許可済みProjectionだけを配信する。これらは#2586/#1251/#1180/#2514に既存の判別票として引き渡し、新CLI Verb/Schema/Runner/正本をここで確定しない。

**履歴と現在時点の分離**：[組織C1二時点補遺 Draft #4622](https://github.com/hat47x/tei-platform/pull/4622)の`validAt/knownAt/observedAt/effectAt`区分を、B2Cでは必要な場合にだけ採用する。注文履歴の`ordered_at >= :from`は検索期間条件であり、**購入時点の商品価格・注文確定Fact、後日の訂正/返品・現時点の商品価格、現在の本人Read Grant/顧客Link**を保証しない。業務上の履歴値は当該Source/Providerの履歴・訂正契約に従い、Grant/Disclosure/Sessionは**閲覧時**に再評価する。W0（同じ注文内容、現在Grantあり）とW1（同じ注文内容、閲覧前にGrant失効）、または履歴価格同一・現在価格のみ変更を独立対照とし、0行・同byte・以前のReceiptを「購入なし」「現在許可」「現在価格」に昇格させない。全B2Cに二時点テーブルや組織Membershipを強制しない。実Host/DB・販売履歴の二時点試験はNOT RUN。

**Source現在性と配布の独立実証**：[main統合 #4616](https://github.com/hat47x/tei-platform/pull/4616)はWorld/Model revisionを変えずに未モデル化Source Rootの実在が変わり、Catalog/Map/Locate/Validateの判定が変化する局所Go試験を通した。これをNativeの現在Source/Validation再確認へ引き継ぐが、実Source全依存閉包、原子的Snapshot、現在Grantは未証明。Studio側は[#4623](https://github.com/hat47x/tei-platform/pull/4623)でCaller Pinに加え**Opt-inのResolved Builder→起動前Digest照合**を提案し、[#4624](https://github.com/hat47x/tei-platform/pull/4624)で宣言とResolver応答の不一致拒否を局所Goテストで検証した。ただし両PRはDraft・未mainで、B2C Frontendの正式BuildIntent/ArtifactSet、OS Loader TOCTOU、Stage1→Stage2、実B2C Provider受領の保証ではない。全体P0は[#2831](https://github.com/hat47x/tei-platform/issues/2831)/[#4583](https://github.com/hat47x/tei-platform/issues/4583)のB1→B2→B3を維持する。

<!-- b2c-h1-distribution-consumer-witness-20261114 -->
### G49 H0/H1/H2：実成果物・Consumer受領・変更後資格の判定票（2026-11-14）

**B1a-Iの後にH1へ追加するのはビルド時の任意契約であり、TEI標準RendererやBrowser Runtimeではない。** H0 Native単独はこの追加受入の対象外とし、H1/ H2の適用先だけで、案件が選んだContract/Provider/Builder/Memberと実際の生成・配布・受領を同一対象・版・用途で比較する。Plugin管理InventoryやBuilder Resolverの「解決済み」、署名済みReleaseの「検証済み」だけでは、公開画面・注文Effect・現在Grantの資格にならない。

| 検証対象 | 必須の正負対照／不足の場合の停止線 | 現在の実証と担当 |
| --- | --- | --- |
| **H1-A 選択と実Build** | Application/Target Profileが選んだContract・Provider/Builder Aに対して、Resolver応答B・実子image C・必要Artifact Memberを分離。同digest/同値でもID/契約/Target不一致、必須依存欠落は拒否。未選択Pluginの追加・非選択は既存H0成果物と契約選択を勝手に変更しない | [#4624](https://github.com/hat47x/tei-platform/pull/4624)はBuildIntent宣言対Resolverの局所修正（Goテスト確認・Draft）、[#4623](https://github.com/hat47x/tei-platform/pull/4623)は実子Builderの起動前pinと選択Bindingの候補（Draft）。B2C実TargetはNOT RUN。#951/#1016/#1080/#1097を再利用 |
| **H1-B 外部Releaseの独立受領** | H1の生成Member/契約版R1を選んだら、Consumerが実取得したRelease Envelopeと各Member Body/Hash/Targetを独立照合。署名R1なのにBody R0、欠落Member、旧Client/SSR/Edge/Cacheが版混在の場合は**そのReleaseの受領資格なし**。署名確認だけで現在配信・撤去・信頼失効を証明しない | [#4617](https://github.com/hat47x/tei-platform/pull/4617)は別Node ProcessのHTTP署名Release＋実Body取得の限定ReferenceでE1専用CI成功。Draft・固定鍵/localhostなので本番Release/Origin/Trust/CDNはNOT RUN。#2589/#2604/#2516へ |
| **H1-C Nativeの閲覧と編集** | Native FieldのQualified Subject/Sourceと、Data Contextの実Row Identity/Version/Grainは別。表示できても、Projectionがnon-injective・集計・多重Edge・Slot不明ならNative/AIからの自動Source/DB逆編集は禁止。明示Slotだけを変更する場合も現在Source span・OwnerのWriter資格・業務Operationの独立検査を要求 | [#4619](https://github.com/hat47x/tei-platform/pull/4619)は同一製品計画ファイルに別DraftとしてSlot相対のread/edit適格性を提案。#2497/#1124/#2500/#2514がOwner。B2C実Edit/WriterはNOT RUN |
| **B1a-P/B2 現在認可・切替** | W0とW1で注文値同一・Grantだけ失効、または旧Client/R1・新Release/R2・変更後の現在価格/Consentを比較。表示済みの値、Cache、Variant Assignment、Checkout Intentは現在Grantや確定注文/Exposure/Outcomeを代行しない。Plugin/Provider旧世代in-flightと新世代選択、Effect Unknownでは再送前Receipt/現在権限を別照会 | #1043/#1656/#2516（Read/Disclosure）、#1174（Effect/Receipt）、#2044/#1223（Plugin更新）、#2604（Release）、#3509（B2C）。公開Host/切替/Commerce/Experimentの実E2EはNOT RUN |

**評価と作業順**：B1a-I（#2586）を先に実Provider/同一Context/独立Native受領で成立させ、H1-A〜Cは選択して採用したTarget/Pluginに限り局所追加する。B1a-Pの現在Read Grant/W0W1は独立した公開資格、B2の実験はAssignment/Exposure/Consent/Outcomeを別に試す。#2018では同じ変更シナリオをH0 Nativeのみ・H1選択Contract・H2選択Experiment Adapterに適用し、正しい結果・Build/更新時間・JS/通信/Cache・レビュー工数と実費を比較する。H1/H2の自動採用・売上向上を事前に主張しない。新配布Host・Builder/Policy/Graph Engine・Analytics/Identity台帳・Test SSOTを設けない。**この節は既存DraftとReferenceを横断した受入設計であり、新規B2C実行・製品採用はNOT RUN**。

<!-- b2c-b1ai-first-missing-edge-20261114 -->
### G49実装引渡し：B1a-Iの最初の欠落辺を一つずつ埋める（2026-11-14）

**先に未成立の一辺を特定する。** main [#4539](https://github.com/hat47x/tei-platform/pull/4539)のEnvironment→実PG Read→Function Data Contextと、main [#4540](https://github.com/hat47x/tei-platform/pull/4540)の申告Native Field→Source occurrenceは**別の契約・別Attempt**。製品CLI未結線、Query `:from`の`declaredByQuery=false`、Product Function/Query/Resultの正式対応、二Receiverの実受領という欠落を、変更前の固定Workspace/Source/Contract版で区別する。すべてを一PRで修正することを要件にしない。

| 実装Slice（既存Owner） | 最初に固定する最小の実成果と独立負例 | 合格上限・次への引渡し |
| --- | --- | --- |
| **R-A 型付きProduct Read Operation**（#2586/#2506/#2622） | 既存Sourceの`order_entry.recent_orders`と実SQL/`:from`の**宣言・型・Runtime value**を照合し、実Provider選択→実Invoke→一つのFunction Contextの行/列/Scope/PresenceをProduct経路で観測。必要なQuery `input`宣言がなければOwnerへ不足箇所を戻し、暗黙の文字列paramで適合扱いしない。別SQL/別DB・RLS role・Provider A/B取り違えを負例にする | **このSliceだけの局所Product Read**。実ContextであってもCurrent Grant、全母集団、Native受領、正式Business Entry公開は未合格 |
| **R-B 製品CLIへの薄い接続**（#2586/#3422/#1251） | R-Aと**同じProduct Operation**を、通常`tei`実バイナリの子プロセスから呼び、Source/contract/実Provider/Context/CLI exit・stdoutを区別してReadback。正式VerbやWireはCLI Owner判断後に決める。未知Verb/未公開対象/不正入力は副作用・秘密stdoutなしで失敗とする | [Draft #4625](https://github.com/hat47x/tei-platform/pull/4625)は既存`operations.ExternalCandidates`をRead-only CLIへ薄く投影した**別領域の参考実装**。ローカルREA Evidence PathやTool Present、実Process成功をB2C PG Read/現在認可の証拠へ流用しない |
| **R-C 同一Contextから独立Receiverへ**（#2586/#2497/#2516/#3509） | **一度だけ**実Query→Contextを生成し、同じResult/Context identity・epoch・Source/Query/Provider版の許可Projectionを標準表示Aと独立Native read-only表示Bへ配送。受領側プロセスが双方でAttribute ID、セルのvalue/NULL/missing/withheld、行数・grain・Scope/partial/unsupportedを確認する。別々にPGを2回読んで同じ数値を得る方式やPublisher receiptだけでは不可 | Aだけ成功・Bへの配送途中切断・旧Context混入・内部`Connection.User/Database/Instance`のWire漏えい・`COUNT`や`SUM`の未宣言精度/NULLを独立Expectedで拒否。Context ID一致だけでは同一原子的DB SnapshotもGrant受理も保証しない |
| **R-D B1a-Pを別票で資格化**（#1043/#1656/#2516/#1174） | 公開Hostが検証した現在Principal↔Customer/Order、Read Grant、Audience/用途別Projection、Private Cache partitionと失効を確認。注文値同一でGrantだけ失効したW0/W1、別Customerの注文増減、旧Client/ReleaseのCacheを負例にする | R-A〜CがPASSでも非公開合成Readerの資格のみ。自動的なBrowser公開、Guest Link、Checkout Effect、Consent/Exposure/CVRは未合格 |

**選択・配布の横断Witness**：今後H1契約を選択した案件では、[Draft #4624](https://github.com/hat47x/tei-platform/pull/4624)のBuildIntent宣言⇄Resolver整合と、[Draft #4623](https://github.com/hat47x/tei-platform/pull/4623)の**実ResolveしたBuilder digest⇄起動前Child image**を別々に照合する。2026-11-14確認時点で、#4623 exact HEAD `f1e219a`はGitHub Actions 8件すべてsuccess、#4624 exact HEAD `2d403f4`は取得できた5件すべてsuccess。ただし**どちらもDraft・未main、Plugin主体の限定BuildPlan／非Plugin Studio全工程は未受入**。Builder一致は成果物のメンバー実同梱・実配信・Runtime受領でもない。[Draft #4617](https://github.com/hat47x/tei-platform/pull/4617)のtest-only HTTP署名Release/Node Consumer検証とも、同じB2C実Releaseだと合算しない。H0 Native単独にはこれらを必須化しない。

**用途と写像の交差検収**：並行[Draft #4619](https://github.com/hat47x/tei-platform/pull/4619)のD0〜D3、P/J/Q（局所観測・異種Evidence Join・問い相対閉包）を、このR-Cでは**読み取る側の必要条件**として参照する。表・集計・Graphで同じ表示が成立しても、JOIN fanout/重複投影、同Label別Identity、non-injective Map、非表示列や否定条件のUnknownを消さない。表示・閲覧適格性、Slot局所編集候補、現在Writer/Effect Grantは別。#4619のG08等の原文は**同一ファイルを編集する別Draft**に残し、このG49節はB2C実受領への引渡しだけを所有する。採用時は両方の最新main差分を照合し、内容を欠落・二重化しない。

**最小の証拠票と停止条件**：既存#1180/#2514/#1163で`Exact HEAD / Workspace・Source・Function・Query Revision / typed-input declared? / selected→invoked Provider・DB Session / actual Rowset+Context ID・epoch・Scope・Coverage / A,B各Consumer delivered・received Body / allowed Projection・Grant qualification / independent Expected / PASS|FAIL|NOT RUN|UNSUPPORTED|UNKNOWN`を別列管理する。Consumerへ送らない内部接続・個人情報や非公開Source ID/件数/ダイジェストをGitHub/外部Harnessへ必須送信しない。最初に不足した辺を修正し、局所成功を次Sliceの成功へ昇格させない。**この節自体はdocs-only、R-A〜Dの新実行とOwner意味受理はNOT RUN**。

<!-- b2c-ra-source-to-executor-failclosed-20261114 -->
### G49 R-AのCode適合ギャップ：既存Query.Inputsを実Provider呼出し前に束縛する（2026-11-14）

**今回のmain Source静的照合（成果の上限：実装着手点の確定）**：
- `internal/query/model.go`に既存`query.Input {Name core.LocalName, Type core.CanonicalID}`と`query.EmbeddedSQL.Inputs []Input`がある。したがってQueryの型付き引数を表す**新しい公開Model/Core型は不要**。
- `reference/workspace/applications/order-entry/queries/recent_orders.tei`には`query order_entry.recent_orders`、`datasource sales`、SQL source・Result mappingがあるが、**`input from`の宣言がない**。同ディレクトリの`recent_orders.sql`は`WHERE o.ordered_at >= :from`を使用する。Reference WorldのDriver testが通ることと、Query入力意味受理は異なる。
- `internal/driver/browse_query_environment.go`の`ExecuteBrowseQueryInEnvironment`は、Environment/Instance/ProviderとExecutorの照合後に、`map[string]any`を`r.ExecuteBrowseQuery`へ渡す。**Query.Inputsの必須性/型/未知パラメータを、この入口で実Executor呼出し前に検査していない**。Registry側の物理接続が同一Instanceかはhost-assertedで、実Read Scope/現在Grantも別資格。
- `tools/reference-browse-read/main.go`の`parseParams`はCLI引数`from=time:RFC3339`を`time.Time`にする一方、`DeclaredByQuery`を**SQL/Executor実行後**のContext構築前に照合する。したがって参照CLIのtyped tag成功、実PG3行取得、`declaredByQuery=false`をProduct契約適合へ昇格しない。

**Owner #2586/#2622/#3422への実装PR分割案（新Command/APIの採用ではない）**：
1. **RA-0 Source/Parser/型選択**：現行Query DSL Loaderが受け付ける`input`構文とCanonical time型を、既存Reader・Role/型Contractから確認する。採用済み構文・意味へ沿って`recent_orders.tei`へ`:from`を宣言し、再Loadした`Query.Inputs`に**名前・型・Source出典**が実際に反映されることをGo独立Expectedで固定。見た目だけの新DSLや、SQL placeholderからの推測による暗黙宣言はしない。型・構文が未受理なら`UNMAPPED`で止める。
2. **RA-1 Provider前の強制検査**：同じSource/Queryと実Invocationの引数セットを比較し、必須`:from`欠落、型違い、未宣言名、explicit NULL/未取得の不許可、時刻のoffset/precision変換、別Query同名を独立負例にする。**拒否時は既存Counting Executorの実Call 0、Result/成功Receiptなし**。Mapの重複キー表現不可はCLI raw flags段の既存重複拒否と区別する。互換な既存Query/DSLへ一律のBreaking変更を加える前に対象Scope/ContractをOwnerが決める。
3. **RA-2 一Attemptの実Value**：宣言成功かつType適合後だけ、既存#4539の隔離PostgreSQL read-only/REPEATABLE READ Fixtureに実引数を渡し、O-003/O-002/O-001の全3行・10:05境界2行・後方0行、Decimal scale2、Result Scope/0行の`unestablished`を独立Expectedへ照合。`selected provider`・実Executor・SQL byte/params・物理DB/Session/RLSとFunction Contextを同じ実Attemptで記録。選択と実Call一致なし/違うDBで偶然同じ行はFAIL。
4. **RA-3 R-B/R-Cへ引渡し**：受理済み型付きOperationのResultを薄い製品`tei`入口へ渡した後、**同一Context**を標準/Native二Receiverへ配信。Draft [#4625](https://github.com/hat47x/tei-platform/pull/4625)の`external-candidates`は、既存Operation→実CLI childの別領域の**構造参照例のみ**。その実CLIが起動できてもB2C実PGやNative受領ではない。

**Readと更新の境界**：同じ注文が3行返った事実、`:from`の時刻、ProviderのSessionを、全顧客の母集団・本人所有・現在Read Grant・注文確定Effectには読み替えない。非公開合成Readの単一Contextを、必要なAudience/用途の投影を通してからNative Consumerへ渡す。動的依存/Dirty保持/Source A→B→A、二Consumer・二Provider snapshotのCoherenceは移植したG08〜G13データ様式節の#2582/#2597/#1708へ委譲し、B2C用Reactive Engine/中央Snapshot Coordinatorを作らない。

**現行の実コードと受入資格（2026-11-14）**：RA-0 [Draft #4626](https://github.com/hat47x/tei-platform/pull/4626) HEAD `9eecf2b9`は、既存構文`input from std.datetime`を採用し、現在のWorld/Golden/Fingerprintを更新した。歴史RunnerではSHA-256固定の旧Queryだけを隔離Frozen Worktree内で再現する。**同HEADのHealth・歴史Self-buildを含め18 CI中17成功**、API実成果物の`sqlgen`依存検査1件は[#1182](https://github.com/hat47x/tei-platform/issues/1182)の別不合格であり、PR全体は未受入。RA-1 [Draft #4631](https://github.com/hat47x/tei-platform/pull/4631)は既存`Resolver.ExecuteBrowseQuery`で宣言済み入力と実Go値をExecutor前に照合する。`std.datetime`→`time.Time`のみを適格として、欠落・NULL・文字列・未知キー等は実Call 0で拒否し、歴史的に入力宣言のない合成Replayを型付きProduct資格へ昇格させない。**修正RA-0をbaseとした統合[Draft #4644](https://github.com/hat47x/tei-platform/pull/4644) HEAD `b5c23044`では、RA-1のGo/旧Web回帰、RA-2Rの空2 DBを使った実PG16参照Reader親＋7ケース非SKIP、設計検査の3 Workflowがすべて成功**。ただし通常製品`tei` CLIから選択Provider/実Connection/SQL/Params/Rowsetを同一Attemptで受けるRA-2P、標準・独立Native Consumer二者のRA-3、現在Principal/Host GrantのB1a-Pは未実証である。これらのDraftはmain未統合。Stage1→Stage2 Studio Desktopセルフビルド#2831/#4583が全体P0、Product Read#2586が並行P0。

<!-- b2c-ra0-historical-revision-ci-20261114 -->
**Source改訂が他Targetへ波及したときの資格（実CI由来）**：[RA-0 Draft #4626](https://github.com/hat47x/tei-platform/pull/4626)の初回HEAD `1b12c833`では、固定Web Targetの歴史Source受入Fixtureが、新しいQuery InputによりWorld Revision不一致を検出して停止した。これは単なるノイズではなく、**今のSourceが変更された証拠**である。元の受入Fixtureは現在Sourceで自動再受理せず、別のScratch内だけで旧Query byte SHAを照合して復元する変更をHEAD `4f238139`へ追加した。この隔離テストが成功しても、新しいQuery Inputを持つ現行SourceのOwner意味受理・Web/G1資格は得られない。また別Workflowで`tei/internal/database/sqlgen`の予期しないAPIバイナリリンクが見つかった。Query宣言の変更が原因と決めつけず、実API成果物の依存検査Ownerが独立調査し、Allowlist緩和で隠さない。**#4626最新HEAD `4f238139`では歴史Fixture差分の2つのWeb/固定World Workflowは再実行でsuccess**。一方、実APIバイナリの`go tool nm`で`tei/internal/database/sqlgen`が2 symbol出る[E1依存除外failure](https://github.com/hat47x/tei-platform/issues/1182#issuecomment-6869938385)は継続し、セルフビルド系もそのHEADで失敗。Browser Asset側の走査PASSと、E1全体の成功を混同しない。Go/Windows/PGの他WorkflowやRA-1の最新CIを別に確認するまで**PR全体CI PASSは保留**。


<!-- g49-ra1-driver-studio-server-witness-20261114 -->
**Studio Server→B2C公開Hostの連携上限**：[Draft #4627](https://github.com/hat47x/tei-platform/pull/4627)は現行Wails Server Bridgeのexported 15 methodを実Go型・ASTで棚卸しし、非ReadのCommand/外部Child Processも登録されていることを局所検査した。現時点のServerはloopback PoCであり、B1a-Pの公開Host/複数Actor/Workspace隔離/現在Read Grantの実資格ではない。将来のSaaSで必要なread-only façadeと許可済みProjection、WebSocket受信・配信の認可は[#1824](https://github.com/hat47x/tei-platform/issues/1824)/[#3421](https://github.com/hat47x/tei-platform/issues/3421)/[#1043](https://github.com/hat47x/tei-platform/issues/1043)の責務に置く。RA-0/RA-1やStudioセルフビルドのHard Gateへ多利用者Webを追加しない。

<!-- b2c-ra1-gateb-studio-n2n3-capability-preflight-20261114 -->
### G49 横断の実行前拒否とDesktop使用実証を区別する（2026-11-14、実コード・実CI照合）

**共通する設計上の規則は「実実行者へ渡す前に、選択済み契約に照らして拒否できること」であり、すべての操作を単一Runnerへ集約することではない。** 対象・Authority・受入票を次のように区別する。

| 実行境界／Owner | 今回照合した観測 | Product採用へ残る条件 |
| --- | --- | --- |
| **Browse Query入力 #2586** | Draft [#4626](https://github.com/hat47x/tei-platform/pull/4626)の`input from std.datetime`に、積上げDraft [#4631](https://github.com/hat47x/tei-platform/pull/4631)で`Query.Inputs`対`time.Time`をExecutor前照合。最初の実Go CIはGate Bの`:from`欠落を検出してFAILし、Scenarioを修正。旧R0の合成Replayは型付き資格を持たない | 最新HEADのGo/旧Web PASS、Environment選択→実Provider→Function Contextの同Attempt、未宣言/型違い/NULLの実Call 0、通常`tei` CLI・独立Native Receiver |
| **Capability Result契約 #4495/#3422** | [PR #4629](https://github.com/hat47x/tei-platform/pull/4629)は**main統合済み**。不正なResult contractを`capabilityinvoke`の実Adapter前で拒否し、不正6例でCall 0。事前Context cancelと正しいBusiness Outcomeを維持。Exact PR HEADでは9 Workflow success | 公開Entry/選択・実呼出し一致、Owner受理、Current Grant、実Host Effect・Product CLIは**別資格**。Queryの`:from`判定もここへ移管しない |
| **Studio生成WebView2 #4583** | [Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `0fcb7b62`は同一HEADで12 Workflow success。WindowsのN2 esbuild出力（`workbench.bundle`）→N3 Go`EmbedFiles`のSHA・role・Attemptをpre/post照合し、変更byteを拒否。実WebView2配信も対応 | 正式`BuildIntent.Resolve`/Builder選択→実`ArtifactSet`、N0→N2接続、**Stage1 TEI生成→Stage2 Wails再構築・Windows実使用**。N2/N3のbyte一致をセルフビルド全体成功へ昇格しない |
| **Studio Server/Web公開 #1824/#3421/#1043** | [Draft #4627](https://github.com/hat47x/tei-platform/pull/4627)はWails ServerのService/公開メソッドを棚卸しし、CommandやSHO Child起動も存在することを確認 | Local loopback PoCと公共Webを分け、Read-only Facade、Actor/Workspaceごとの現在Grant、WS受信者別開示を別検証。Server従属プロセス＋Named Pipe/UDS案は性能・障害回復・配布の**実測前は方式未確定** |

**切替条件**：現在のSource/Query契約と実引数に合致すること、実Providerへ正しい引数が渡されたこと、Function ContextをConsumerが実受領したこと、公開Hostに現在Grantがあることは**4枚の別票**。Result contractの事前拒否やStudioのN2/N3 byte固定は各票を強めるが相互代替しない。#4629のmain成功・#4587のWindows限定成功を#4631の不合格Runへ加算しない。適格な集計Law/粒度は前節のD1.2、Studioは依然P0 #2831/#4583。新Host・Registry・Measure Engine・全工程を待つ巨大Gateは追加しない。

<!-- g49-ra2-reference-pg-and-shared-actual-consumption-20261114 -->
### G49 RA-2の実DB証拠：参照Reader 7ケース全PASSとProduct受入の欠落辺（2026-11-14）

**新たな実行結果**：[積上げDraft PR #4638](https://github.com/hat47x/tei-platform/pull/4638) HEAD `0ce3a52f070314741f72462982ff940a1f791a28`、base [RA-1 #4631](https://github.com/hat47x/tei-platform/pull/4631) HEAD `02af8339`。専用[実PostgreSQL Workflow #43553849861](https://github.com/hat47x/tei-platform/actions/runs/43553849861) は**completed/success（1/1 Job）**。実PR merge SHA `05ab66fae60aa349900c93f306c605d7a819e697`と実HEAD親を照合し、PostgreSQL 16の空DB 2つとReader Roleで、既存`TestBrowseReadAgainstDisposablePostgreSQL`の親・A/B/C/D/E/G/Fの全7 subtestを**非SKIP・全PASS**で実行。これは`tools/reference-browse-read`の実Connection/SQL/Rowsetを使った証拠で、手書き期待値と独立DB readerを維持。Reference用の第二Fixture正本は追加していない。

| 受入対象と資格 | 今回の実観測 | なお未成立の条件（既存Owner） |
| --- | --- | --- |
| **RA-0** Source宣言 | [#4626](https://github.com/hat47x/tei-platform/pull/4626)新HEAD `9eecf2b9`：`input from std.datetime`、Current World/Golden/Fingerprintと旧Frozen Query rawを分離し、旧HEAD `94876821`の未使用`reflect` importによるGo test/vet失敗も修正。旧HEAD `94876821`で16/18成功・HealthとAPI E1不合格は履歴票 | 同じ`9eecf2b9`の全体CI 18系統を[実Run](https://github.com/hat47x/tei-platform/pull/4626/checks)で再検査。E1は依然`sqlgen`のAPIリンク異常として[#1182](https://github.com/hat47x/tei-platform/issues/1182)に独立不合格。Current GoldenとFrozen Historyの意味受理・main採用・他レーンのCIはこの修正だけでは確定しない |
| **RA-1** Go typed preinvoke | [#4631](https://github.com/hat47x/tei-platform/pull/4631) HEAD `02af8339`で[Go回帰CI #43548961899](https://github.com/hat47x/tei-platform/actions/runs/43548961899) completed/success。Gate Bの欠落`:from`をScenarioの`time.Time`入力に直し、防護を緩めなかった | 実Environment-bound Product入口の毎回適用、全型/NULL契約、旧R0のuntyped ReferenceをProduct Readerへ通さない。最新main/RA-0更新後は**積上げ差分を再検証** |
| **RA-2 実PG参照** | [#4638](https://github.com/hat47x/tei-platform/pull/4638)は2 DB／RLS／Source変更・JOIN重複／0行／NULL／列権限／時刻境界を一つの隔離PG Attemptで観測。全7 Case success | **正式Product `tei` CLIから実Operation→実Provider→Context**のSame Attemptと、選択Provider A／実Call B負例、実Consumed SourceとSession Scope/epochの記録は未成立。参照CLIのpassをProduct E2Eへ昇格しない |
| **RA-3/本件R-C** 実受領 | 参照Contextは実読取の結果だが、Receiver間の同一Context deliveryには未接続 | **同一Context ID・epochを持つ標準Display A＋独立Native Bの各受領Body**、Source/Scope/NULL/partial/projection、Expected独立照合。AggregateはD1.2の別受入 |
| **B1a-P** Current Grant | 参照DB Read・RLS Roleの結果は確認できたが、Public Principal/Customer所有のAuthorityではない | Host current Grant、Audience Projection、失効W0/W1、Cache隔離とRelease/旧Clientの別票 |

<!-- b2c-ra0-current-golden-historical-byte-pin-20261114 -->
**RA-0 Source改訂をCurrent/Historyに分離した修正候補（2026-11-14）**：[Draft #4626](https://github.com/hat47x/tei-platform/pull/4626) HEAD `94876821f784d5fa9f71b15a05ac4156221b2bc8`で、旧HEADの[Repository Health #43544373270](https://github.com/hat47x/tei-platform/actions/runs/43544373270)および[セルフビルド #43544373386](https://github.com/hat47x/tei-platform/actions/runs/43544373386)で実際に失敗した契約を4ファイルで修正した。

| 修正箇所 | 観測済みの旧失敗→修正の意味 | 実装後の受入停止線 |
| --- | --- | --- |
| `examples/data-source-workbench/query_route_separation_test.go` | SQL `:from`は宣言なしという旧Expectedが、実Loader宣言後にも`want [from]`として残っていた。新Sourceには`input from std.datetime`を要求し、未宣言入力集合は空へ更新 | SQL字面の静的検知は実PG Parameter Binderの証明ではない。XTEA別Workspaceの`:status`をこのQueryへ混入しない |
| `reference/world.json` | Canonical current Worldの`order_entry.recent_orders`に`inputs:[{name:from,type:std.datetime}]`が欠け、`workspace.Load`とのWorld同一性でFAIL | 旧受理Worldを新Sourceと同一扱いしない。新Goldenは変更後Current Worldに限定 |
| `internal/workspace/load_listing_fingerprint_test.go` | 旧Fingerpint `61ebb5ca...`に対しGo実CIで`435b687e...`を観測。値の来歴とCurrent World意味改訂を注記し、測定済み新値を固定 | この文字列の更新単体はOwner意味受理や過去のSource履歴保証にならない。最新HEAD Loadを実Testで再検証 |
| `tools/reference-plugin-self-build/historical_runner_parity_e2e_test.go` | 旧RunnerのLoaderは新しい`input`行を解析できず実セルフビルドでFAIL。Frozen Worktreeだけに**実git commit `e888bb8f...`の旧Query raw bytes**を戻す。SHA-256 `a7c1c4f6...`と新Sourceとの差を独立検査し、Frozen Diffは旧Manifest 2件＋旧Query 1件の3パス限定 | **現WorkspaceのQueryから宣言を消さない**。旧RunnerがFrozen版を通過しても、現Worldの新Queryを受理したことにはしない |

**再検査**：[#4626最新HEADのCI](https://github.com/hat47x/tei-platform/pull/4626/checks)はexact HEAD `9eecf2b9`で**18/18 Workflow完了（17 success、1 failure）**。[全体Health #43558268996](https://github.com/hat47x/tei-platform/actions/runs/43558268996)と[歴史Self-build #43558269076](https://github.com/hat47x/tei-platform/actions/runs/43558269076)はsuccess。残るAPI実依存E1 failureは別Owner #1182であり、PR全体PASSではない。別[Browser→JSON API実経路](https://github.com/hat47x/tei-platform/actions/runs/43544373457)の旧HEAD失敗はAPIの`tei/internal/database/sqlgen`2シンボル検出（Browser byte自体の走査成功）で、RA-0による因果は不明。#1182に別の実依存調査を残し、許可リストを黙って拡張しない。新RA-0 HEADの合格を確認したあとRA-1 #4631・RA-2 #4638の**積上げbaseが更新された事実**を明記して新Checkoutで再検証する。旧HEADのRA-1 Go成功、RA-2実PG 7/7成功を新baseのPASSとして再利用しない。

<!-- g49-current-windows-e1c-buildintent-heads-20261114 -->
**他レーンの最新実態を受入順に同期（2026-11-14）**：
- **Desktopの実使用**：[Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) はHEAD `9b58533d`で[Windows WebView2 Workflow #43572235468](https://github.com/hat47x/tei-platform/actions/runs/43572235468)の**通常Desktop・生成Desktopの両Jobが成功**し、同HEADの12系統のCIもすべて成功した。旧HEAD `357345d7`のOS版取得に伴うタイムアウトは、Pythonの`sys.getwindowsversion()`による測定へ変更して解消した。これは両Desktopの限定的な実WebView2検証であり、**Stage1のTEI StudioがStage2のStudio実行ファイルを生成し、その実行ファイルを利用した**というセルフビルド一往復の証明ではない。全体P0 #2831/#4583は維持する。
- **Builder選択／実Child pin**：[Draft #4634](https://github.com/hat47x/tei-platform/pull/4634) 最新HEAD `7d6a1238` は**8/8 Workflow completed/success**。BuildIntent宣言→Resolver応答／選択Builder→起動前Image SHAを同じCheckoutで検証したが、Plugin主体の限定BuildPlanであり、非Plugin Desktop `N0〜N3`・OS Loader read-time byte/外部依存閉包、Stage2生成Desktop完成ではない。
- **署名Release独立HTTP受領**：[PR #4617](https://github.com/hat47x/tei-platform/pull/4617)は**main統合済み**。外部Node Processがテスト用loopback HTTPから署名Releaseと2 Member bytesを取得し、正常1／拒否9を検査。実GitHub Releases/外部Assetの配布・現在Trust・Audience Grant/Availabilityとは別であり、[Draft #4640](https://github.com/hat47x/tei-platform/pull/4640)の外部Release readback（GET 200/302、混入、Issuer差、実Body digest）は設計上の次工程。G49 H0 NativeにPlugin/JS/外部Releaseを必須条件として追加しない。
- **Script Launch**：[PR #4639](https://github.com/hat47x/tei-platform/pull/4639)はmain統合済みの計画改訂で、`#4636`の宣言preflightを現行設計に反映。Query RA-0/RA-1/RA-2のProduct Read、実Go Provider Entry/Effect、Studio Stage2のQualified Passには流用しない。

<!-- g49-currentworld-integrated-ra012-and-provider-identity-20261114 -->
### G49 RA-0〜RA-2Rの一体再検収とRA-2Pの実Provider境界（2026-11-14、最新他レーン整合）

**修正後Current Worldで再検証するための独立Draft**：[PR #4644](https://github.com/hat47x/tei-platform/pull/4644) HEAD `b5c23044ebaf8671d73a140da92fc9c4d74edff4` は、[#4626 RA-0](https://github.com/hat47x/tei-platform/pull/4626) HEAD `9eecf2b9`を**base**とし、[#4631 RA-1](https://github.com/hat47x/tei-platform/pull/4631) のGo本体・検査と[#4638 RA-2R](https://github.com/hat47x/tei-platform/pull/4638) の実PG16 Workflowを同一checkoutへ組み合わせた。旧RA-1の変更前Go blobと新RA-0上のblobが5ファイルすべて一致していることを事前検査し、RA-1のGo実装とテストおよびGo CIは**byte同一**。RA-2R CIは本PRでも起動するため`pull_request.branches`にRA-0 branchを追加したのみで、試験値やExecutor/DB契約は変更していない。元DraftはOwner正本として保持し、本統合PRは競合解消・merge順決定の代行ではない。

| 同一HEADの検収票 | 現時点に存在する証拠 | 失効・未成立の条件 |
| --- | --- | --- |
| **RA-0 Current/History** | `9eecf2b9`の18 Workflowは実起動、exact HEADの18 Workflowは**17 completed/success、1 completed/failure、未完了0**。実[Health #43558268996](https://github.com/hat47x/tei-platform/actions/runs/43558268996)と[歴史セルフビルド #43558269076](https://github.com/hat47x/tei-platform/actions/runs/43558269076)が両方success。旧 `94876821` は未使用`reflect`でHealth失敗、新HEADで除去済み | Current World/Golden/Fingerprintと歴史Frozen Runnerの**限定Go/実セルフビルド回帰は成功**。ただしBrowser→JSON APIの`sqlgen`2-symbol E1 failureは[#1182](https://github.com/hat47x/tei-platform/issues/1182)の別Ownerで、**PR全体は不合格・Draft未main**。Health合格を業務Owner Meaning/Current Grantへ昇格しない |
| **RA-0/1同一checkout** | [統合Draft #4644](https://github.com/hat47x/tei-platform/pull/4644)のRA-1 Go／旧Web回帰 Workflow [#43558633790](https://github.com/hat47x/tei-platform/actions/runs/43558633790)が**completed/success**。統合PRの3 Workflowは**同一HEADで3/3 completed/success**（RA-1 Go回帰、RA-2実PG、Issue-to-Design）。Go Driver/Query/Syntax/Validation・旧Web Target・参照Readerと`cmd/tei`の既存Goテスト・`go vet`をPR merge-to-head/clean照合つきで完走 | 旧#4631の[成功Run #43548961899](https://github.com/hat47x/tei-platform/actions/runs/43548961899)は**旧RA-0 base**の限定Go票。新仮統合HEADで`Query.Inputs`／Actual `time.Time`の正常1 Call・不正0 Callを再検証し、仮統合PASSをmain merge合格にしない |
| **RA-2R実PG16同一checkout** | 新Draftの[PostgreSQL Workflow #43558633773](https://github.com/hat47x/tei-platform/actions/runs/43558633773)は**completed/success、空2 DB/RLS/親＋7 subtestが非SKIP・全PASS**。仮統合SHA `fbf8fe55b92f2537cd4197e3c81df077d6f9ba1c`とPR HEAD `b5c23044...`を実Logで照合した。旧[#4638の実Run #43553849861](https://github.com/hat47x/tei-platform/actions/runs/43553849861)では隔離2 DB/RLS/7 subtest全PASS・非SKIP | **修正RA-0を含む同一Checkoutで実PG全7ケース非SKIP成功を確認済み**。ただし本当の通常製品`tei` CLIからのQuery Provider実選択・実呼出しではなく、**参照Readerのみ**の資格 |
| **RA-2P Product Operation** | 既存`ExecuteBrowseQueryInEnvironment`／`BrowseQueryExecutorMap`を利用できる。Provider・Instance選択などの**宣言値**と、実DBが返したByte/Rowsetは区別可能な受入票を作れる | **通常`tei` CLI→実選択Provider→実DB/Role/SQL/Params/Rowset→Function Data Context**の同一Attemptは未実装・未検証。Context実二ConsumerはRA-3、現在GrantはB1a-Pの別資格 |

**RA-2Pの実装を進める前の所有境界（データ様式Draft #4619の最新検討を採用）**：`DatabaseProviderBoundBrowseQueryExecutor.DatabaseProvider()`はExecutorの**自己申告**、`BrowseQueryEnvironmentExecution.Instance`は宣言側の束縛、`QuerySourceSHA256`は渡したQuery Source byteのDigestであり、**実Connection先PG16のDatabase/Role・実消費SQL/Params・Session/Scopeを保証する情報ではない**。製品PRでは既存DriverとEnvironmentの選択をそのまま使い、正例を`selected A→actual A`、負例を`selected A→actual B`／同名異Root・SQL・別DBへ分ける。事前拒否できる失格ParamはDB/Executor Call 0、**接続後に判別した別DBや権限Scopeの違いは不適格Resultとして成功Contextを返さない**。同一Attemptで確認できない項目は`unknown`と記録する。新しいProvider Registry/Host/Source正本/Reactive Engineは作らず、#2586/#2514/#1180へ実結果を渡す。

**並行する他チャットの最新適格性**：[#4642](https://github.com/hat47x/tei-platform/pull/4642)は**main統合済みのScript Launch計画更新（docs-only）**。[#4641](https://github.com/hat47x/tei-platform/pull/4641)はBuilderの**論理Graph→物理Build→実Artifact Consumer**対応とGCX-01〜08を整理した設計Draftで、非PluginのStage1→Stage2 Windows実成功ではない。[#4640](https://github.com/hat47x/tei-platform/pull/4640)は署名Release test-only loopbackで得た実GET証拠を、**実GitHub Releases Asset GET 200/302・署名Issuer/リダイレクト・実Body・現在Trust**へ継ぐ条件を置いた設計Draft。両者はG49のProvider/Native Readをブロックする追加Hard Gateにしない。Studio Desktopセルフビルド#2831/#4583が全体P0、Product Read #2586が並行P0であり、旧版の合格を同じ受入票へ加算しない。

<!-- g49-cross-lane-acquisition-migration-toolchain-20261114 -->
### G49 実際の受領結果に結び付ける追加条件（2026-11-14）

**一つの受注履歴の読取り**は、型付きQueryが実Providerを呼ぶことだけでは成立しない。Sourceと実行対象の対応、現在の利用者に対する読取り範囲、取得結果の被覆、生成したアプリがDBへ及ぼす作用、利用側で受け取った本文までを区別する。この整理は[main統合済みの横断Consumer設計 #4653](https://github.com/hat47x/tei-platform/pull/4653)のDefinition→Binding→Authority→Execution→Acquisition→Delivery→Acceptanceに従う**既存受入票の補足**であり、新しい共通Runner、権限管理機構、永続Evidence正本を追加しない。

| 境界 | 成熟した成果と、今回計画へ追加する条件 | Ownerが引き続き検収すること |
| --- | --- | --- |
| **取得できた範囲**（D1／RA-2P） | [取得条件観測Draft #4647](https://github.com/hat47x/tei-platform/pull/4647) HEAD `a962a3f6`は7系統のCI成功。実PG16でRLS、LIMIT、宣言外JOIN、ViewやPartitionによる制限を観測できる範囲では`partial`として理由を運び、通常のNative SQLでは`native-sql-predicate-not-closed`を`open`に残す。**制限が見えない状態は`complete`でも不存在証明でもない**。0行と未取得・非開示を区別し、会員自身の履歴に別会員の存在を推測させる表示を作らない | #2586／#2514。W4の実列OID・NULLの保存元、実DBと宣言Instanceの一致、Source現在性、Grantは未実証。#4647の参照Readを通常`tei` CLIのRA-2P成功へ加算しない |
| **業務操作によるDB変更**（G13／G25） | [生成アプリ修正Draft #4662](https://github.com/hat47x/tei-platform/pull/4662) HEAD `095c5a11`は、生成された`query`／`command`／`validate`でMigrationを暗黙実行しないコード候補と、生成バイナリの別Process負例を追加。Repository Health／Cache／既知脆弱性の**3系統CI成功**。通常Queryは既存DBを読取専用で開き、Commandは既存DBに限り更新し、ValidateはDBなしで実行する。**DB未存在・Schema不適合で業務操作が勝手にDDLやDB作成を行わないこと**を製品側の安全条件とする | #4657／#4234／#4125。これはSQLite単一Provider向けのRB0安全停止候補。実Schema/History/Release検証、明示Init/Upgrade、複数DB・PostgreSQL移行、旧新ClientのCutoverは未完成。DB接続成功を移行適格・現在Grantと同一視しない |
| **再現可能なビルド**（Studio P0） | [Toolchain観測Draft #4648](https://github.com/hat47x/tei-platform/pull/4648) HEAD `64fb0f9a`は8系統CI成功。Go 1.26.11／1.27.0の実測で同じBuild計画でも生成バイトが異なる例を確認し、`toolchainObservation.matched`と`namedByPlan=false`を区別する。前者は申告されたファイルの読戻し一致で、実際にそのToolchainを起動した証明ではない | #2831／#4583。ToolchainをBuild Intentの正式入力に含めるか、Build子Processの実起動とStage1→Stage2成果物受領をどう束縛するかはOwner判断。現時点のSeed Trust／Stage2完成を宣言しない |
| **Desktopでの利用** | [Windows再検証Draft #4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `9b58533d`は12/12 CI成功。通常版と生成版のWebView2実Jobが両方成功した | B0の実使用に限定。正式な非Plugin Build Subject→実ArtifactSet→Stage1再生成→Stage2起動→WebView2利用を同じ実行で結ぶB2/B3は未達。計画優先順位は変更しない |
| **専門Toolkit・Plugin** | [数理・統計・科学Toolkit #4663](https://github.com/hat47x/tei-platform/pull/4663)の**設計文書はmain統合済み**。これはWindow集計やPERT等の計算機能が実装済みという意味ではない。[役割別Plugin構成 #4664](https://github.com/hat47x/tei-platform/pull/4664)は未統合で、HEAD `dcb694cd`のRepository Healthに失敗が残る。二つの計画の実装資格は別に検証する。統計Window／WBS・PERT／科学計算等の独立Expected、Studioと他Targetの受領条件を整理した | 新Core型・専用DSL・科学Runtime・第二Plugin Hostは先行採用しない。P0のQuery実Read・Studioセルフビルドを待たせず、外部Runtime不要の小さなWindow集計から既存Operator/Consumerで適合性を測る |

**実装順**：最初に[RA-0〜RA-2R一体検証Draft #4644](https://github.com/hat47x/tei-platform/pull/4644)の実Go／参照PG成功を維持したまま、RA-2Pで**通常製品`tei` CLI→実Provider/実DB・Role・SQL・typed Params→Function Data Context**を一つのAttemptとして検査する。取得範囲を`complete`へ推測昇格せず、通常操作が暗黙DDLを実行しない負例も別Ownerへ渡す。RA-3で標準Displayと独立Native Consumerの受領本文を独立に照合し、B1a-Pで会員・匿名・運用者ごとの現在Grantと開示範囲を別途確認する。Studioは非Plugin DesktopのStage1→Stage2一往復を優先し、Toolkit・Migration本格適用・公開Web共同編集をその完成の前提にしない。

**次に実装する一本の最短経路**：#2586 Ownerが`RA-0 main適格化→RA-1 exact base再検証`を先に行い、RA-2参照PGで既に観測した同じQuery/Input/Resultを**正式Product Operation→通常`tei` CLI（同一Subprocess）→実Context→Receiver A/B**へ接続する。Providerは既存Environment/Registryを選択し、Actual Executor/SQL bytes/Session/Rowsetと独立Expectedが同一Attemptで照合できる場合に限りD1 Productの限定受理を検討する。新Host・第二Resolver・暗黙SQL Placeholder推論・新Measure Engine・関数型DSLの表面露出はしない。RA-2のPG Fixtureから再利用するのは**試験手順と観測上限**であって、Reference入口自体を本番Runnerへ名前変更しない。

**横断的停止線**：main統合された[#4636](https://github.com/hat47x/tei-platform/pull/4636)のCapability Input/Effect宣言preflight、[#4634](https://github.com/hat47x/tei-platform/pull/4634)のResolver→Child Pin、[#4637](https://github.com/hat47x/tei-platform/pull/4637)のREA Host固定Selector byte readは、各別OwnerのDraft・限定CI成功であり、Query/Persist/Native HostのGrantを代替しない。共有するのは`Source/Declaration→Selection→Actual Call/Consumed Bytes→Actual Receiver→Independent Expected→Owner Acceptance`の**受入票の構造**のみ。Web/Wails公開は#1824/#3421へ、Studio Desktopセルフビルド#2831/#4583は全体P0のまま。

**計画統合状況**：同一計画ファイルを変更する並行[PR #4600](https://github.com/hat47x/tei-platform/pull/4600)の「問い相対旧依存辺N16/N17」「Studio Windows B0とBuildIntentギャップ」の二行を、本Draftにも**内容を欠落させず**取り込んだ。双方はまだ別Branch/未mainであり、#4600の採否・閉鎖、mainへの実マージ、競合解消を代行したものではない。新実行試験・Owner Meaning受理はNOT RUN。

<!-- g49-b1ap-trust-scope-matrix-20261114 -->
## G49 B1a-P：B2Cの本人・公開・計測を別資格として検収する（2026-11-14）

**位置づけ**：B1a-IのRA-0〜RA-2Rは、[統合Draft #4644](https://github.com/hat47x/tei-platform/pull/4644)の限定同一checkoutでGo引数拒否・参照PostgreSQL 7ケースが成功したにすぎない。現在の製品入口RA-2P、同じContextの独立二Consumer受領RA-3、実公開HostでのB1a-Pは**未成立**。公開資格を得るために新しいTEI Host・Identity台帳・専用Frontend SDK/Runtimeを設けない。実Hostが既存の認証・認可責務を持ち、TEIの型付きOperationとProviderは業務意味・データ取得・結果構造を共有する。

| 利用場面 | 必要な本人・対象資格 | 開示と操作の境界 | 最小の独立反例 |
| --- | --- | --- | --- |
| **匿名の商品閲覧** | 認証主体を要求しないPublic Scope。公開済み商品・価格の「版」と有効期間は区別する | Publicに分類されたField/Artifactだけ。個人価格、会員在庫予約、運用情報は混入させない | Public CacheへPrivate Projectionを保存、旧公開版だけが残る |
| **ゲストのカート** | 一時Session/所有証拠。認証済み顧客や組織Membershipへ自動昇格しない | 仮CartはOrder確定でもPayment受領でもない。Session回復・紐付け時は別の実Host判定 | 別Guest Sessionから同じCart IDを読める／ログイン時に他人のCartへ誤結合 |
| **本人の注文履歴** | 実Hostで検証した現在Principalと、対象Customer/Orderとの権威ある対応、閲覧時点の現在Grant。期間入力`:from`は絞り込みであって認可ではない | Row Scope＋Field/Column Projection＋用途に沿う説明を独立判定。実DBのRLS/Maskは有効な防御層だが、Customer本人の資格そのものではない | 同じ注文値・同じ`:from`でW0許可→W1失効、他人の注文ID、RLSが0件を返すが全件0と誤認 |
| **運用者・サポート** | 別の運用者Principal、用途・対象・必要な現在権限。一般利用者としてのGrantやOrg Membershipを流用しない | 特権閲覧・訂正・代理操作を混在させず、利用目的と監査を実Hostの契約で判定 | 会員向けPublic/Private Cacheへ運用者の補助列・件数・診断を返す |
| **分析・実験計測** | Consent/目的/有効版を実計測時点で検査。Experiment Allocation IDを本人Customer IDと同一視しない | Assignment・実Exposure・Action Intent・業務Effect・Outcomeを別証拠とする。Consentなし・取得不能を非購入に変換しない | 未同意Event送出、露出のない割当をCVへ計上、母集団/欠測を落としたCVR改善主張 |

**B1a-Pの最小実行順**（R-Dの内部受入手順であり、RA-2P/RA-3の前提Gateにはしない）：

1. **P-A Host identity / audience**：既存Hostで受領したPrincipal・Session/Guest・Audienceの出所を確認。Frontendの自己申告Header、Query、DOM選択、旧Cookieだけでは現在の業務本人としない。匿名Publicはその旨を明示する。
2. **P-B Subject link / Grant**：Customer↔Orderの実関係と操作用途・時点・Scopeを突合。照合不能、permission denied、旧Grant、部分Sourceはunknown/deniedを区別して拒否または限定返却とする。公開向けResponseの許可判定は実返却直前に再確認する。
3. **P-C Actual projection / delivery**：選択Query/Providerだけでなく実SQL/DB Role/実Rowset、Row/Column/Relation/COUNT/SUM/診断Metaの実開示、Serializer/HTTP Response、Cache・旧Clientまで**同一Attempt**で検査。内部DSN、Source Root、秘匿列名、件数などがError/Timing/Cacheから推知できる経路を否定対照に含める。
4. **P-D Revocation / independent readback**：W0閲覧成功→W1 Grant失効、別Principal、旧CDN/Browser Cache、別Tab、Client再送で再試行。実Hostの現在認可と独立受領を確認し、保存済み旧証拠やResult byte一致で権限を復活させない。強い全層原子公開は暗黙の必須要件にせず、残存旧配信を検出・遮断・説明する。

**公開資格票**は `Principal source / customer link / purpose / current Grant / row scope / column projection / actual receiver / response version / cache audience / revocation check / independent Expected / Owner acceptance` を列として持つ。原データが0件、Field NULL、RLSによる部分集合、列Mask、データSource欠落では`empty`と`complete`を同一視せず、取得範囲と不明理由を記録する。既存[#2506](https://github.com/hat47x/tei-platform/issues/2506) Context、[#1656](https://github.com/hat47x/tei-platform/issues/1656)/[#2516](https://github.com/hat47x/tei-platform/issues/2516)開示、[#1043](https://github.com/hat47x/tei-platform/issues/1043)/[#3412](https://github.com/hat47x/tei-platform/issues/3412)本人/Session、[#3271](https://github.com/hat47x/tei-platform/issues/3271)/[#2604](https://github.com/hat47x/tei-platform/issues/2604)失効、[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)独立Expectedへ委譲する。

**先行順位を維持**：全体P0はStudio Desktop Stage1→Stage2セルフビルド[#2831](https://github.com/hat47x/tei-platform/issues/2831)/[#4583](https://github.com/hat47x/tei-platform/issues/4583)。並行P0は[#2586](https://github.com/hat47x/tei-platform/issues/2586) RA-2P通常`tei` Product Read→RA-3二Consumer。B1a-Pはその**外部公開への追加資格**。B1b Cart/Order/Payment副作用、B2 Variant/Experiment、B3 H0/H1/H2の費用・性能比較、B4予約/Subscriptionへの拡張を初回ReadのHard Gateにしない。成功資格はSource宣言、局所Go、参照PG、製品CLI、Native、公開Host、Business Effect、実験効果で別々に記録。本文は設計・受入計画であり、実公開認証・Commerce・Browser E2E・CVR実証を行った意味ではない。

<!-- g49-b1b-effect-measurement-separation-20261114 -->
## G49 B1b〜B3：購入の確定・公開切替・計測の資格を連結せず照合する（2026-11-14）

**採る判断**：B2Cの差別化は新しいCheckout/Analytics Engineを作ることではなく、既存の型付きOperation/Effect/Receiptと、独自Frontend/Commerce/Flag/Analyticsの**異なる正本の間で、何が実際に確定したかを取り違えず変更できること**にある。B1a-Iの正式Product Read（RA-2P→RA-3）とB1a-PのCurrent Grantはそれぞれ未成立のまま。以下はその後段の**実施手順・独立Expected候補**であり、実Commerce・Browser・Field実験の完了を表さない。新StateMachine/Host/決済処理/統計Engine/必須Client SDKは設けず、[#1174](https://github.com/hat47x/tei-platform/issues/1174)、[#1129](https://github.com/hat47x/tei-platform/issues/1129)、[#3503](https://github.com/hat47x/tei-platform/issues/3503)、[#951](https://github.com/hat47x/tei-platform/issues/951)、[#1180](https://github.com/hat47x/tei-platform/issues/1180)を再利用する。

| 一つの購入導線の観測点 | 正本と許される主張 | 禁止する昇格・必須反例 |
| --- | --- | --- |
| **① Product/Offer表示** | Product/Price/Promotionの公開版・通貨・有効時刻・Audience、SSR/Edge/Client/Cacheで実表示したVariantを別票に記録。表示は見積りまたは案内である | 古いCDN/Clientの割引表示を現在の販売価格とみなさない。C0の文言変更に実効Coupon/Disclosure/Consent変更を混入させない |
| **② Cart/Checkout Intent** | 同一操作対象・選択Customer/Guest・数量・Quote版、二重送信を判別できるOperation Identity。Optimistic UIは仮状態に限る | 画面上の「完了」、FlagのAssignment、Click/IntentをOrder・Payment確定とみなさない。Guest→会員結合は現在Hostで別判定 |
| **③ 実Provider確定境界** | 権威あるServer/Commerce Providerが現在の価格・税・Coupon・在庫・対象・実行権限を再評価し、選択Providerと実呼出しProvider・送信要求・応答・Commitを突合 | 旧VariantのOffer、旧Grant、A選択→B実Call、Hookのbefore/after成功だけでCommit成功としない。実Provider能力を超えるexactly-onceを宣言しない |
| **④ Effect/Receiptの確定** | 同じOperation IdentityでReceipt・再照会可能なOrder/Payment Outcome・独立Readを分ける。重複要求はProviderの冪等契約に従い再照合する | Timeout/応答喪失/unknown commitでは**未確定**のまま。安全な自動再送と仮定せずlookup first、照会不能はunknownを維持。Client UIの成功表示やAnalytics Eventで補完しない |
| **⑤ Experiment/Analytics** | Assignment、実際のExposure、許可されたEvent、Action、業務Effect、Outcomeを異なる観測とし、母集団・欠測・Consent版・割当版・観測時刻を記録 | Exposure未観測を非露出確定、計測不能を非購入、Order確定を因果upliftに変換しない。SRM、返品・取消・粗利益等のguardrailは既存分析/Providerを利用 |
| **⑥ Release/撤回** | UI/Creative/Flagのロールバックと、既存Order/Paymentの訂正・返金は**別操作**。旧SSR/Edge/Client/Service Worker・Private Cacheの生存と現在Backend受理条件を照合 | Flag巻戻しで既存の注文Effectを消したと扱わない。既配信・オフライン保存済みbyteを遡及的に消去できる保証はしない。失効後の**新規配信**を現在Grantで抑止し、残存配信は別途検出・運用対処する |

**B1bの最小縦切り**：公開の本番決済ではなく、隔離したCommerce Sandboxを一つ選ぶ。正常Cart→Checkoutだけでなく、同じOperation IDによる重複送信、Commit前拒否、Commit後ACK喪失、旧Quote/旧Grant、Provider選択と実Callの不一致を各一組の正負対照にする。外部Effectについては、Client表示・実Call・Provider Receipt・独立Order再読込みを別列に残す。Writeの効果や権限を証明できないときは`unknown`とし、既存[#1174](https://github.com/hat47x/tei-platform/issues/1174)/[#1129](https://github.com/hat47x/tei-platform/issues/1129)へ未成立を返す。B1a-Pの公開Host合格をSandbox内のEffect実装の着手条件にせず、外部公開の認定には別途現在Grantを要求する。

**B2の最小縦切り**：既存Experiment/Flag Providerを任意選択し、**一つの既存Native画面**でAssignment→SSR実表示→Hydration→Exposure→任意Analytics送出→Checkout Outcomeを時点と版付きで追跡。Variant混在、旧Client復帰、未同意、計測欠落、遅延Event、既存注文取消後のOutcome誤計上を反例にする。Payload/集計で個人識別情報・顧客関係・DSN・内部Evidence位置を無断露出しない。結果の`consent-denied`、`not-observed`、`unavailable`をゼロEvent／ゼロ購入と同一視しない。

**変更影響／失効への引渡し**：Native Frontend・CMS/Creative・Flag/Allocation・Target Build/Release・業務Rule・Provider/Contract・Current Grant/Consent・Source Relationの実効差分から、既存[#4518](https://github.com/hat47x/tei-platform/issues/4518)/[#3271](https://github.com/hat47x/tei-platform/issues/3271)/[#2582](https://github.com/hat47x/tei-platform/issues/2582)/[#2600](https://github.com/hat47x/tei-platform/issues/2600)の必要上流・実消費・失効候補を照合する。ファイル種別が文言/画像でも価格/同意へ影響するなら軽い変更レーンで逃がさない。旧Graphから削除されたRequired辺、A→B→A、Provider/Generator版変更、Audience/時点切替を「byte一致だから安全」と判断しない。逆に無関係なNative Routeや未選択H2 Adapterを巻き込む過剰失効も検出する。Graph/Source Universeがpartial/unavailableなら`no impact`と断定しない。

**B3の比較条件**：[#2018](https://github.com/hat47x/tei-platform/issues/2018)の既存比較に、H0=Nativeのみ、H1=選択した型付きContract/Build-time Adapter、H2=さらに任意Experiment Adapterを、**同じNative Frontend・同じ実Provider/Scenario・同じ機能・同じ変更・同じ測定条件**で投入する。正確性を独立Expectedで先に判定し、初回実装工数・変更時間・Review/復旧工数・追加JS/HTTP Request/TTFB/INP/LCP/CLS・Provider/運用費・発見した不正/欠測を別票に採る。初期値は未測定。H0にTEI JS/DOM/Router/Runtimeを強制しない。H1/H2の費用が便益を超えるときは**Plugin非採用・撤去を正式な合格選択肢**とする。CVR/粗利益の因果効果は標本、割付、Consent、追跡欠測、期間、統計的不確かさを確認した実Experimentがなければ`not measured`とする。

**順序と責務**：全体P0は[#2831](https://github.com/hat47x/tei-platform/issues/2831)/[#4583](https://github.com/hat47x/tei-platform/issues/4583) Stage1→Stage2 Desktop Self-build。並行P0の[#2586](https://github.com/hat47x/tei-platform/issues/2586) RA-2P/RA-3からB1a-Iを実Consumerへ接続。B1a-Pは[#1043](https://github.com/hat47x/tei-platform/issues/1043)/[#1656](https://github.com/hat47x/tei-platform/issues/1656)/[#2516](https://github.com/hat47x/tei-platform/issues/2516)の実Host開示で別検収。B1bは#1174/#1129/#951、B2は#3503/#2604/#3271、B3は#2018へ受入を返す。すべて[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)の既存Independent Expectedに追記し、**B2C専用の第二Scenario/Effect/Analytics正本を作らない**。本文は設計と検収順の更新であり、新規Go/PG/Browser/Commerce/実験試験とOwner意味受理は`NOT RUN / NOT ACCEPTED`。

## プロセス・イベント・応答を共通の意味から説明する（設計候補）

Webの要求、CLIの起動、タイマー、Message、画面操作には、それぞれ異なるHost・起動方式がある。ここで標準化を検討するのは一つのイベントループではなく、**Trigger→型付きInvocation→Function/Rule→State/Effect→Outcome→Response／Consumer受領**という意味の対応である。長寿命Processの起動・停止と、一件の処理の成功・失敗も混同しない。

初期の受入は、受注受付の説明用Fixtureを使い、HTTP/CLIの二入口が同じ意味Operationへ至るかをHeadlessで照合する。期限の境界、現在Grant、DB Commit後の通知結果不明、重複・逆順Event、未実装のTimer Providerを負例にする。外部部品はContractとTODOを先に記述し、既存Driver ScenarioのMockと独立Expectedで基本設計からシミュレーションする。ただし、Mockの成功は実DBやBrokerの実行証拠にならない。

詳細は[Lifecycle・Trigger・Operationの共通意味設計](../design/PROCESS_LIFECYCLE_TRIGGER_OPERATION_SEMANTICS_2026-11.md)を参照する。Ownerは[共通Application API #1158](https://github.com/hat47x/tei-platform/issues/1158)、[Script Launch #4495](https://github.com/hat47x/tei-platform/issues/4495)等の既存Issueに分け、新Runtime・Dispatcher・Schedulerを前提にしない。**現行P0のStudio Stage1→Stage2と通常CLIの実Readには、新しい先行依存を追加しない。**


<!-- g49-lifecycle-process-invocation-acceptance-20261114 -->
### G49の読取りと業務確定を、同じ意味モデルから別々に検証する

[共通実行モデルの設計 #4665](https://github.com/hat47x/tei-platform/pull/4665)と[Lifecycle・Triggerの設計 #4666](https://github.com/hat47x/tei-platform/pull/4666)はmain統合済みである。ただし、両方とも**設計上の対応関係**を定めたもので、共通Event Loop、Scheduler、Graph Compiler、Host、正式DSL構文の新規採用ではない。G49では、Processの生存状態と、利用者による一件のInvocationが成功したかを別に扱う。

**最初は二種類のOperationを混ぜずに試す。** `order_entry.recent_orders`は指定日時以降の注文履歴を読み取るOperationであり、RA-0〜RA-2Rでは型付き入力と使い捨てPGの参照読取りまでを検証している。受注確定`submit`はDB Commit・Receipt・通知などの外部作用が関係し、読取りが成功してもその資格を引き継がない。どちらも、元のSourceにあるSubject・Guard・入力・Output/Outcome・必要なEffectを意味の対応元とし、説明文・処理図・Scenarioを派生表現として生成する。生成した説明文や図を第二の業務意味正本にはしない。

| 段階 | 最小の確認 | 必ず区別する反例 | Owner |
| --- | --- | --- | --- |
| **LifecycleとTrigger** | CLI起動とHTTP要求が同じ意味Operationを指せるか。単発CLIに常駐Loopを要求しない | Triggerの宣言を、実Schedulerの発火や現在の実行許可と混同しない（PL-01/02/06） | [#1158](https://github.com/hat47x/tei-platform/issues/1158)・[#4495](https://github.com/hat47x/tei-platform/issues/4495) |
| **Typed InvocationとRead** | `from`の宣言・実引数・選択Provider・実Connection・取得したRowsetとCoverageが同一Attemptに対応するか | 選択A／実B、型違い、RLSの`partial`、0行を全件不存在と誤認する場合（PL-01/10、PX-10） | [#2586](https://github.com/hat47x/tei-platform/issues/2586)・[#2514](https://github.com/hat47x/tei-platform/issues/2514) |
| **意味評価と実作用** | 受注確定のGuard、期限と現在Grant、提案されたState/EffectをContractとMock／独立Expectedで先に対照する | Mockの受注確定を実DB Commitと呼ぶ、期限の発生時刻と判定時刻を取り違える（PL-03/07） | [#1129](https://github.com/hat47x/tei-platform/issues/1129)・[#1174](https://github.com/hat47x/tei-platform/issues/1174) |
| **EffectとOutcome** | Commit成功・通知送信・通知受領・結果不明を別Receiptで検証する | Commit後のACK喪失を未実行と誤認し、無条件再送で注文を重複確定する（PL-04/05/09、PX-03/04） | [#1174](https://github.com/hat47x/tei-platform/issues/1174)・[#1129](https://github.com/hat47x/tei-platform/issues/1129) |
| **Responseと実Consumer** | HTTP/CLI/Nativeへ同じ対象・版・結果の適切なProjectionが届いたか。必要な現在Grantを受領時にも照合する | HTTP 200、CLI終了コード0、表示成功をConsumerの意味受理／業務確定と同一視する（PL-10、PX-01/11） | [#1251](https://github.com/hat47x/tei-platform/issues/1251)・[#3509](https://github.com/hat47x/tei-platform/issues/3509) |
| **図解・Target移植** | 同じ意味からLifecycle／Control／Effect／Traceの図を読取専用で投影し、Source位置と安定IDを保持する | 図の配置変更で意味Revisionが変わる、同期→非同期移植で順序・期限・Effect保証を落とす（PX-05/06/12） | [#4198](https://github.com/hat47x/tei-platform/issues/4198)・[#1124](https://github.com/hat47x/tei-platform/issues/1124) |

<!-- g49-source-diagnostic-four-layers-20261114 -->
**宣言を編集した時点と実行時の検査も分ける。** [DSLの四層計画 #4652](https://github.com/hat47x/tei-platform/pull/4652)と[引継ぎ・索引修正 #4659](https://github.com/hat47x/tei-platform/pull/4659)はmain統合済み。A＝Source構文・位置、B＝Workspace全体の意味整合、C＝Adapter前の構造拒否、D＝公開Entry・現在Grant・実Consumerの受領を、それぞれ独立に確認する。[実Sourceからの拒否テストDraft #4654](https://github.com/hat47x/tei-platform/pull/4654)はA/C、[同名Capability入力のWorkspace診断Draft #4655](https://github.com/hat47x/tei-platform/pull/4655)はB/CのGo実装候補だが、**両PRとも現HEADのRepository Healthに失敗が残り、main未統合**。ログ上の`gofmt`差分を修正してから同HEADで再試験し、Source位置のないRuntime診断に架空の字句位置を付けない。Parserが受理したことだけで実行許可を付与せず、Bで見つけた違反をCのAdapter call 0と照合する。Dの公開・現在Grant・実Providerは別Ownerで未実証である。

**実現順**は、既存のProduct Read RA-2PとStudio Stage1→Stage2を先に完結させ、共通Lifecycleの最初の読取り専用Projectionをその結果へ結び付ける。TODO・Mockで外部部品を置き換える設計段階では、入力・結果・欠損・副作用・実装未接続を明示し、Mockと本番Providerを同一の成功票にしない。実Targetへの移植は、実際のInvocation、作用、Response、Consumerの受領を比較してから判断する。**この節は設計とScenarioの引渡しであり、新たなGo／DB／Windows／Scheduler実験の完了ではない。**

<!-- g49-effective-placement-evidence-reconciliation-20261115 -->
### G49 横断機能の実効適用：Graphの対象選択から利用側の受領まで（2026-11-15）

**判断の要点**：アプリケーションの構造Graphで「対象を選べた」ことと、その機能が**生成・配布・表示・送信・受領**されたことは別である。全ページ共通の計測、アクセシビリティ補助、監査、キャンペーン挿入などの横断機能に対し、既存のSource/Meaning→Graph索引→Plugin/Hook→Builder→Target→実Consumerという受渡しを、**適用義務単位の照合**として検証する。新しいAspect Weaver、Graph Compiler、常駐Browser SDK、Host、Analytics Engineは追加しない。詳細の受入Ownerは[高度B2C #3509の追補](https://github.com/hat47x/tei-platform/issues/3509#issuecomment-6891903592)、共通Graph/Hookは[#4198](https://github.com/hat47x/tei-platform/issues/4198)・[#3503](https://github.com/hat47x/tei-platform/issues/3503)・[統合設計 #4681](https://github.com/hat47x/tei-platform/pull/4681)とする。

**Graphと意味の分離**：原Sourceを上書きする新しいGraph正本は持たない。NodeはQualified Subject/Role/Scope/Revision、Edgeは方向・関係Role・発生箇所・安定した出現Identityを区別し、同端点間の多重辺を潰さない。Graphの描画座標・折り畳み・視覚的グループ化は意味を変えない。Selectorの対象Universeには、**現在一致したNodeだけでなく、今後一致し得る公開Route・Variant・Plugin公開Facetと未探索依存**を含める。静的なTEI RouteだけでNative Framework、外部CMS、旧CDN/Clientの本番全域を閉じたと主張しない。未観測の外部要素は問い相対のunknownとして持ち、無関係な部分を不必要に失効させない（#2495/#2497/#2582/#3262/#4518）。

| 別々に検収する段階 | 実際に確認するWitness | 誤って認定してはならないこと |
| --- | --- | --- |
| **U：対象と義務** | Subject/Route/公開条件、必須・任意の別、Universeと探索範囲、対象版、除外・未解決 | 宣言Routeを列挙しただけで「本番の全Pageを網羅」 |
| **S：選択** | matched / excluded / undetermined、Selector/Plugin公開Facetと依存版 | matchedだから生成済み、閲覧可能だから実行可能 |
| **G：生成・梱包** | Role付きGraphの対象→Hook Slot/Placement→Builderの実入力→Artifact Member/BOM | 生成ソースに文字列があるだけで配布済み |
| **D：配布・取得** | Release/Response/SSR・Edge・CDN・Cache・旧Clientの実バイト列と時点 | 最新Builderの成功を、利用者の最新Release受領と同一視 |
| **V：利用・露出** | Navigation種別、実表示、Variant、可視化、Consent、実ブラウザ観測 | Feature Flag割付や評価を、実際のExposureと同一視 |
| **R：外部受領・Effect** | Collectorの受領、外部OperationのReceiptと独立Readを別々に照合 | 送信しただけで分析に反映、クリックしただけで購入確定 |

この表は**新しいCore state/API/永続Evidence正本ではない**。U/S/G/D/V/Rは既存の#951/#1016（Composition）、#3278（Trait/Hook）、#1121/#2090（Target/再生成保持）、#2600/#3271（版・Cache）、#2516/#1174（開示・権限・Effect）、#1180/#2514（独立Expected）の観測欄として使う。1 Route→複数Variant/Artifact、1 Asset→複数Route、SSR＋SPA→重複通知があり得るため、**Node件数やScript件数の単純な一致を合格条件にしない**。必要な適用範囲、出現多重度、未観測範囲を明示する。

**表現と工程の再利用**：標準データ構造のGraph/Relation/SequenceとプロセスのTrigger→Invocation→Function/Rule→Outcomeを接合するが、Webページ遷移（Interaction）、販促・実験（Experience）、購入・注文確定（Business Effect）は別Flowのまま対応付ける。Graphは読み取り・変換・生成の共通入力であり、画面の内部状態・業務トランザクション・購買の正本ではない。REAの経済Eventは必要なら業務Outcomeを説明するRoleとして使用し、Analytics EventやFlag Evaluationと同じ実体にしない。Node ViewはPlugin配線、Data Flow、Journey、Lifecycleを別Projectionで表示できるが、Canvasの線をそのまま実Provider Invoke/Grantに昇格させない。Bindingは公開されたContract/Slot/Revisionの範囲だけを継承する（#2619/#4350/#4198/#951/#2386）。

**適用される場所を固定しない**：B2C向けの上位PluginはDistribution（配布・切替）、Authoring（Journey/Rule記述）、Composition（公開Contract/Slot選択）、Organization（担当・承認の責務）、Runtime（任意Providerの実処理）、Disclosure（Audience/用途/同意）の各Scopeへ**分離して実現**する。Pluginを選択しただけで全Scopeの能力・現在のGrantを得ない。H0＝Nativeのみ、H1＝任意ContractとBuild時生成、H2＝任意の実験/計測Adapterは正規の三方式とし、H0/H1はTEI固有Browser JS/DOM/Router/通信Proxyを必須としない。B2Cへの進出を標準Rendererの採用率で評価しない（#2018/#951/#1016）。

**同意と試験母集団を守る**：Assignment（割付）／Flag Evaluation／Rendered・Visible Exposure／User Action／Authoritative Business Effect／Measured Outcomeを分ける。Evaluation回数≠表示回数、計測欠損≠非購入である。未同意の計測送信を「欠測修復」で自動再送せず、実験ID・Targeting Keyから業務Customerや認証Principalを推定しない。公開情報・非公開情報、行列Scope、Cookie/Cache、外部Egress、ログ/診断/集計・件数の漏えいも用途別に区別する。OpenFeatureのEvaluation Context/Hook、Webのbfcache `pageshow.persisted`、WCAG 2.2、Core Web Vitalsは**外部の比較基準**であり、採用済み実装や実適合の証拠ではない（#1043/#1656/#2516/#3412）。

#### 次の実装差分と独立Expectedへの引渡し

| 順番・既存Owner | 最初に固定する一つの試験 | 成立条件と実装しないもの |
| --- | --- | --- |
| **EP-0：read-only Scope/Selector** #4198/#3509/#4518 | 既存Subject・公開Route一つと、未取得Native/CMS Routeを含むUniverseを宣言してGraph/Selectorのmatched・undeterminedを比較 | 実Source/ScopeとDeclarationを区別。静的範囲の一致から全サイトcompleteを発行しない |
| **EP-1：生成物Member照合** #951/#1121/#2090 | EP-0で選択した一つのHook Slotから、既存Builderが実際に含めたMemberを同一Source/World/Generator版でReadback。生成ソースあり・BOM欠落を負例にする | SelectorのPASSだけでは合格にしない。新Graph Engineや全Native Scannerなし |
| **EP-2：配布版とBrowser** #2600/#3271/#3509 | 一Targetの静的ページまたはSSRで、実Response/旧Cache版とBrowserの実Navigation・可視化を別観測 | Release成功からClient受領を推論しない。NativeだけのH0を破壊しない |
| **EP-3：同意/二重送信・効果** #3503/#2516/#1174 | SSR→SPA、bfcache、Consent拒否、Analytics受領と注文Receiptを別々に観測 | 未同意送信ゼロ、必要な多重度の検証、Effect unknown時のlookup-first。実PaymentをMockで置換した合格なし |
| **EP-4：価値の比較と採用** #2018/#3796/#3509 | H0/H1/H2で同じTarget/要求/Provider/独立Expected、変更1回とRollback1回を比較 | 手書き残差、対象漏れ、再生成破壊、JS/通信・Core Web Vitals、検証工数、復旧時間・運用費を測る。CVR改善は適切な実験以外から推論しない |

**負例の固定（新しいScenario正本を作らない）**：既存#3509のAX/GQ/BC、#1180/#2514に `G49-EP-01`〜`G49-EP-10` の参照名で引き渡す。①未探索動的Page、②Member未収録、③SSR/SPA二重通知、④Consent拒否、⑤bfcache復帰・prerender、⑥旧CDN/Client、⑦割付のみで未露出、⑧購入ACK喪失、⑨H0/H1/H2のJS/通信差分、⑩見た目だけの変更に偽装した価格・同意の実効変更を一因子ずつ対照する。前回Issue追補の短縮名`PL-01〜10`は他のProcess/Lifecycle試験名と衝突し得るため、**このG49固有参照名を使い、別のテスト番号体系として採用しない**。

**作業順序と証拠の上限**：全体P0はStudio Stage1→Stage2セルフビルド（#2831/#4583）、並行P0は通常製品`tei`→実Provider→Function Data Context→独立二Consumer（#2586/#1251）。EP-0/1はその足場を再利用する小さなread-only/Builder試験として並走できるが、B1a-P公開Grant、B1b Commerce Effect、B2 Exposure/Consent、B3性能・開発費の実測をP0の前提にしない。B0限定Inspectorと既存Source/Graph設計はmainにあるが、**本節で計画した新たなGo/CLI/Builder/Native Browser/配布/同意/Collector/Commerce実試験、Owner Meaning受理、CVR改善はNOT RUN / NOT ACCEPTED**。


<!-- g49-gpd-placement-mode-20261115 -->
### G49：横断機能の実現方式を選び、実際に使われた成果まで照合する（2026-11-15）

**今回加える判断**：前節のU（対象）→S（選択）→G（生成）→D（配布）→V（実表示）→R（受領）は、どの横断機能でも必ず同じ実行経路を通るという意味ではない。[宣言的実装のG/P/D比較 #5069](https://github.com/hat47x/tei-platform/pull/5069)と[共通Process実現の整理 #5071](https://github.com/hat47x/tei-platform/pull/5071)を受け、**同じ受理済み意味を、対象の場所と時点に応じてどこで実現するか**を先に判断する。Graphの対象選択は共通でも、すべてをGo/JS生成または常駐TEI Interpreterへ通さない。方式はアプリケーション全体に一つではなく、個々のBinding/Placementごとに選ぶ。選択した方式の実証と、別方式への置換可能性は異なる資格である。

| 実現方式 | B2Cで適する例と必要な性質 | 必ず別に確認する事実 |
| --- | --- | --- |
| **G：Targetコード・静的成果物の生成** | 公開可能なSEOメタ情報、静的文言、型付きClient/Server Adapter、変更の少ないHead/Section部品。SEOや初回描画を妨げず、BrowserでのTEI実行を要求しない | Source/Meaning/生成器Pin→実Builder入力→Artifact Member→実配布。コード生成に成功しても公開済み・表示済みではない |
| **P：既存の型付きOperator・汎用Functionによる評価** | 既存RuleやProfileのパラメータに基づく限定判断、Request/Session単位の出し分け。新しい汎用Interpreterを足すのではなく**選択済みの既存評価処理**を適用する | 演算子版、入力の粒度・NULL/未取得、評価時点、Read Footprint、Targetに許される利用範囲。値だけ一致しても不要な秘匿Readや失敗の差は不適合 |
| **D：外部Host/Providerへ委譲** | 最新の価格・在庫・税/クーポン、Cart/Checkout/Payment、現在Grant、外部Flag/Analytics/CMSの権威ある処理 | Selected Providerとactual Call、現在Grant/Consent、外部Effect/Receipt/独立Read。委譲の成功をTEI生成実装の割合へ算入しない |
| **H0：Native実装のみ** | ブランド独自Router/DOM/CSS/状態管理、FrameworkのSSR/SPA/Hydration、手書きFrontend | 型付きContractを使わない場合も正規比較対象。**必須TEI Browser JS/DOM wrapper/Proxyは0**。存在しないTEI経路にG/P/Dの証拠を要求しない |

この節の**方式G/P/D**（コード生成／既存評価／外部委譲）と前節の**段階G/D**（生成物の確認／配布の確認）は、略号が同じでも別の分類である。またH0/H1/H2は、TEI導入と任意プラグイン利用を比較する**構成**であり、方式G/P/Dとは直交する。一つのRouteでHeadはG、Request時の限定表示判断はP、購入確定はD、ブランドの画面はNativeで構成できる。Pで実行した関数がDのProviderを呼ぶ場合には、純粋なPの成功とは別にDの実Call/Effectを確認する。どの方式でもBrowser/Server/Edge/Build-timeの配置はPluginの公開CapabilityとTarget条件から決め、実行先の不適合を暗黙Fallbackで埋めない。

**Bindingの最小検収票（新しいCore schemaではない）**：対象のQualified Source/Meaning・Source occurrence、適用するRoute/Section/Operation/Hook Slot、選択条件とUniverse、配置先とLifecycle phase、必要な多重度、G/P/D選択と実装Pin、利用側のRequired Data Context/Scope/時点、Actual Member/実Invoke、実Consumer、現在の開示・実行資格を**別欄**で記録する。Fragment/Markdown/Graphの見た目やPlugin名だけでBindingの同一性を決めない。Providerの選択と実呼出し、参照したPage集合と未探索の動的Route集合は一致を仮定しない。Requiredな横断適用でScopeが未観測なら全件成功を認定せず、任意の計測・広告が止まっただけで業務Operationを停止させない（#951/#1016/#2497/#3262/#3278/#3503/#4198）。

**実装を急ぐ範囲を限定する**：まず既存[positive Ruleの生成と評価方式](https://github.com/hat47x/tei-platform/pull/5069)を、同じ入力・独立Expectedに対するGとPの差の説明に使う。ただしpositive Ruleの局所結果はB2CのRoute/Hook、実Browserや公開Grantの資格ではない。続けて**公開してよい合成の一Route/一Head Slot**を選び、U/Sの実Source・公開FacetとGの実Builder Memberを同じ版で照合する（EP-0→EP-1）。実Targetの該当Slotがなければ `unbound` と報告し、架空のRouteや生成器を製品の標準機能として追加しない。Generatorのテスト成功から配信D/V/Rを認定しない。[Windowsの実TEI CLI→生成Workbench N2 #5059](https://github.com/hat47x/tei-platform/pull/5059)は子Processと成果物の限定実証であり、**B2C Target生成にもStudio Stage1→Stage2全体にも転用しない**。

**既存の独立Expectedへ渡す最短の反例**：①静的Head MemberがSelectorで選ばれたのに実Bundleへ入らない、②同じ公開RuleのG/Pで値が一致してもPだけ非公開情報を先読み、③Pの入力/演算子Pinが古いのに結果だけ同じ、④Gの旧Release/Edge Cacheが現在版と混在、⑤ConsentなしにD側Collectorへ送出、⑥価格をC0文言扱いで変更しDの再検証を省略、⑦H0にTEI JSを必須注入、⑧選択Aと実Provider Bを同一と扱う。新たなScenario正本を作らず、#3509のBC/AX/GQ/G49-EPと#1180/#2514へ対応付ける。値一致、Browser表示、Analytics成功、実注文Effect、CVR因果改善を一票に合算しない。

**次の引渡し**：EP-0/1はread-only/静的生成の最小縦断として#4198/#951/#1121/#2090、G/P意味比較は#5069/#988/#1121/#2514、Targetへの配信・旧版・Consentは#2600/#3271/#3503/#2516、Commerce Effectは#1174/#1129へ戻す。[型付き配線 #5075](https://github.com/hat47x/tei-platform/pull/5075)と[Plugin充足設計 #5076](https://github.com/hat47x/tei-platform/pull/5076)は**main統合済みの設計**であり、製品Connector実Callや開示・適用の受理ではない。[Plugin実現方式 #5095](https://github.com/hat47x/tei-platform/pull/5095)もmain統合済み。B1a-I（通常`tei`→実Provider→Function Context→独立Consumer）とStudio Stage1→Stage2の製品P0は維持。**本節は設計・実装順の更新であり、新Go/Builder/実Browser/Commerce/実験、Owner意味受理はNOT RUN / NOT ACCEPTED**。

<!-- g49-b2c-typed-placement-minimal-witness-20261115 -->
### G49 EP-0→EP-1：実際のB2C成果物へ至る型付き配置の最小契約（2026-11-15）

[Node配線の意味適合 #5075](https://github.com/hat47x/tei-platform/pull/5075)、[Plugin充足と開示 #5076](https://github.com/hat47x/tei-platform/pull/5076)、[実現方式 #5095](https://github.com/hat47x/tei-platform/pull/5095)を受け、G49の最小実験を**具体的な一つのBinding**へ絞る。候補は、合成・公開可能な商品案内ページの**Headメタ情報一か所**（題名、説明、canonical等）の任意Build時挿入である。これは実装題材の提案であって、現行TEIにNative Route/Head Slot/SEO Generatorが実装済みという主張ではない。既存Targetが当該Slotを公開しない場合、結線は `unbound/unsupported` として報告する。新しいDOM Wrapper、必須JS、Router、SEO Engine、Adapter Solverは作らない。

**一か所をつなぐ際に必要な情報**を、`SourceのSubject/occurrence/Revision` → `提供側の型・意味・公開Facet` → `利用側の必要Shape/多重度・locale・Scope/開示・生成時点` → `明示Adapter（必要な場合）` → `Route/Slotの選択結果と必要性` → `実現方式G/P/Dと実行先` → `実Builder入力・Artifact Member` の順に照合する。型名が一致しても、値の欠損、時点、対象顧客、言語、許容Audience、単位、Error/NULLの意味が合わなければ自動結線しない。逆に値を個々の基本型へ分解することを強制せず、適合する複合型をそのまま結線し、明示Adapterで必要な変換だけ行えるようにする。いずれも新しいCore Port型ではなく、#2497/#951/#1016/#4350の既存評価項目への投影である。

| 対照する一因子 | EP-0（選択・結線）の独立Expected | EP-1（生成・実収録）の独立Expected |
| --- | --- | --- |
| 公開Head一件と、未探索のCMS動的Route | 観測したRoute/Slotの適用義務は判定し、CMSの未知Routeを全サイト適用済みとしない | **実際に選択した一Memberだけ**を生成成果物で照合。知らないRoute用のMemberを捏造しない |
| 必須SEO項目と任意Analyticsを別扱い | 必須項目のBinding欠落は未充足。任意Analytics不採用は業務Operationの失敗ではない | 必須Member欠落は不適合。Analyticsを含めないH0も正規構成として保持 |
| 同じSlotへの二つの適用・順序依存 | 衝突・重複・優先順位が未宣言なら勝手に一つを選ばない。Edgeの出現IDは区別 | 同じMetadata/Scriptの二重収録を単純に「成功2件」と数えない |
| 公開情報の文言変更／価格・同意の変更 | 文字列編集であってもRule/Price/Consent依存があればC0に限定しない | Source/Builderの版と成果物の差を確認。旧Memberや旧配信Responseの現在性を推定しない |
| 公開SEO情報と、非公開Customer/Order入力 | 公開Headへの秘匿依存を、型が合うだけで許容しない。未取得・非開示は空文字として埋めない | 原Source byteや秘密値の無断同梱を拒否。Member Hashが一致しても開示適格性を保証しない |
| 同じMeaningをG/P/Dで実現 | Head静的情報はG候補、現行価格・Cart/Payment・現在Grantは権威あるDに残し、Pは実装済み型付き評価のみ。配置ごとに方式を選ぶ | GならBuilder/Member、Pなら評価時の実Input/Read、Dなら実Provider Call/Receiptを**別の票**とし、不要なArtifact生成を要求しない |

**最小実行順**：(1) 既存Workspaceから一つの対象とSource occurrenceを読み、Route/Slotの実在性を確かめる。(2) 必要Shapeと公開範囲を宣言し、Selectorのmatched/excluded/undeterminedを読み取る。(3) 実Targetがある場合に限り、選択したBuilderの**実Consumed Input**とArtifact Memberを同一World/Generator/Toolchain版で照合する。(4) Memberの削除・重複・未宣言Adapter・非公開入力・旧Sourceという一因子の負例を置く。(5) 配信D、可視化V、外部受領Rは後続の別資格にする。実Target/Slotが存在しない場合には(3)以降を実行済みと偽らず、既存#1121/#2090へTarget拡張要求を返す。

**独立Expectedと採否**：上記は[#3509](https://github.com/hat47x/tei-platform/issues/3509)のG49-EP、[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#2514](https://github.com/hat47x/tei-platform/issues/2514)に統合する**受入項目**であり、第二Scenario正本ではない。観測した「必須の配置義務が満たされた範囲」と、公開に必要な実Release/Grant/Consentを分離する。対照対象のH0 Native-onlyはTEI JS・Renderer・Contract採用ゼロでも適格とし、契約を使うH1だけを追加生成・運用費用で評価する。既存の正値Rule `validationeval` 対 `tei self-build` の実Go回帰（main [#5095](https://github.com/hat47x/tei-platform/pull/5095)が引用する基線）はG/P比較の実装証拠であって、B2CのHead/実配信/Browserの試験ではない。**本節固有のEP-0/1、実B2C Builder/Browser/Commerceの新試験はNOT RUN / NOT ACCEPTED**。Studio全体セルフビルド#2831/#4583と通常製品実Read#2586は並行P0のままにする。


**SEO Headの情報を一括して扱わない**：初回のEP-0/1は公開用の静的な商品説明（例：`meta description`）**一Member**に限定する。商品名や`canonical`はURL・言語・Variantと紐づくため別の受入に分ける。価格・在庫・会員価格・クーポン・税額などは更新時点と権威をもつCommerce Providerのデータであり、静的ビルド時の値を現在の確定値として出力しない。[Google Search CentralのMerchant Listing](https://developers.google.com/search/docs/appearance/structured-data/merchant-listing)は商品情報の初期HTMLへの配置を推奨しているが、それはBuild時に古い価格を固定してよい根拠ではない。公開する必要がある場合は、対象の配信側で現在の公開可能なSnapshot/更新方針を検証し、UI、初期HTML、構造化データ、Checkoutでの価格確定の異なる時点と責務を区別する。これは既存Target/Providerへの制約であり、B2C専用SEOエンジンの追加要件ではない。

| Head情報の区分 | 初回の実現方式と取得・失効の条件 | 否定条件 |
| --- | --- | --- |
| 公開静的説明 `meta description` | G候補。Source・公開Facet・locale・Builderの実Consumed Input・Memberを照合 | 必須Slot不在、NULL/未取得、非公開Source、Builder未収録を成功としない |
| `title` / `canonical` / `hreflang` | 既存Route所有者のURL/言語/Variantと整合させる。Slot・Route Universeを別に確認 | 未探索CMS RouteやVariantに同じcanonicalを推定適用しない |
| 価格・在庫等のProduct/Offer構造化データ | D側の公開可能な現在Snapshotと配信時の有効期間に従う。SEOの初期HTMLと購入時のServer確定を独立観測 | Builder時の古い価格、会員限定・未開示の価格、UIの楽観表示を業務確定として公開しない |
| 実験用・計測用メタ情報 | H2の任意Extension。Flag Evaluation/Assignment/Exposure/Effectは別証拠 | Consent拒否やH0 Native-onlyで必須Collector/Browser JSを挿入しない |

**止める条件と進める条件**：初回は既存Route/Slotの機械的探索を行い、無ければ `unbound/unsupported` を固定票として#1121/#2090に返す。公開静的説明の一Memberが実Targetに存在し、SourceとWorld/Generator/Toolchainが束縛され、実Builderから読み戻せた場合だけEP-1の限定PASSを認める。外部公開、Crawler受理、現在の商品価格、Commerce、CVRは引き続き未検証とする。この設計差分は#3509の正本、#1180/#2514の独立Expectedに引き渡す。Google/OpenFeatureの仕様記載は外部の比較資料であって、TEI実装の動作証拠ではない。


<!-- g49-b2c-crosschat-portability-and-evidence-20261116 -->
### G49：B2Cの表現・接続・実現・公開を分離する横断受入（2026-11-16）

**設計の収束点**：B2Cで必要なものは第二のWeb HostやB2C専用DSLではなく、標準データ構造・Role付きGraph、型付きOperation、Pluginの能力宣言、Target配置と実Consumerに至る証拠の接合である。Node View、Journey、SEO、Experimentは**既存モデルの用途別Projection**であり、画面上の配線だけで実Providerの選択・呼出し・認可が成立するわけではない。#4198/#4350/#951/#5075/#5076のモデルと、#5095/#1121のG（生成）・P（既存評価）・D（外部委譲）を再利用する。新しいB2C Runtime・第二Resolver・独立Graph Compiler・必須Browser SDKを追加しない。

**一つの意味を異なる入口へ渡す**：共通OperationのRequest/Result/Error/Effect契約を、(a)通常製品`tei`のCLI、(b)Wailsに依存しないGo Operation、(c)案件側Web/HTTP Adapterから利用できる構成を目指す。Wails Serverは(c)のHost Adapter候補であってOperationの正本ではなく、Web側がCLIプロセスをshell実行することを標準方式にしない。B2CのH0 Native-onlyは共通Operationも採用しない対照構成として残し、H1では型付きContractと任意Build-time生成、H2では実験/計測Adapterを追加できる。**同じOperationの意味**と**各入口に許される開示・現在Grant**を分けるため、内部CLI出力をそのまま公開HTTPレスポンスへ転送しない。API PluginのCLI直結/Wails公開は別Ownerの検討であり、旧PR [#5096](https://github.com/hat47x/tei-platform/pull/5096)と三入口試験の後継[#5137](https://github.com/hat47x/tei-platform/pull/5137)はClosed/Unmergedである。ただし対象の限定三入口Go試験は現mainの別更新に収録されており、Web公開の検証済み実装とは扱わない。

**Bindingは粒度と用途の両方で適合させる**：複合型のPort接続、個々のField接続、明示Adapterを同じFlowの選択肢とする。Source/Port/EdgeのIdentity、Direction、Multiplicity、Unit/Precision、Time、locale、NULL/未取得/非開示、Actor/Disclosure、可逆性を保持し、画面上の見た目や型名の一致のみで同値としない。B2CのProduct/OfferやCart/Orderは意味のある業務データとし、`assignment`・`exposure`・`analytics event`は別Roleの観測データにとどめる。REAの業務Eventを計測イベントに置き換えず、Node Viewの表示位置をSource編集の対象確定に使わない（#3503/#2497/#1174/#1124）。

| 実証順（既存のOwner） | 最小の正例と失敗例 | 成立したと認める範囲 |
| --- | --- | --- |
| **EP-0 / #4198・#951・#4350** | 合成商品ページ一件の公開`meta description` Source→実Route/Slotを型と開示範囲込みで選択。Slot不在、未探索CMS Route、複合型の明示変換不足、重複Edgeを対照 | 実際に探索できたSource UniverseとSelector結果だけ。未取得を全対象適用済みとしない |
| **EP-1 / #1121・#2090・#2600** | 同一World/Source/Generator/Toolchainで選択した一Memberを実BuilderのConsumed Inputと生成Artifact/BOMからReadback。Member欠落、旧Revision、非公開入力混入を注入 | 一Memberの実生成・収録まで。外部配信・Crawler・Browserの受領は別資格 |
| **B1a-I/P / #2586・#1251・#5096** | 通常`tei`の同一Typed Operationを独立Consumerが受領。CLI/HTTP/Host各入口で正値・未公開Field・現在Grant失効・欠損/NULLを比較 | 実Consumer・開示資格を入口ごとに検証。APIの見た目、内蔵InspectやReferenceだけで非Builtin Plugin/Commerceが成立したとしない |
| **EP-2 / #3271・#3503** | Native SSR/SPA/旧Clientの実Responseを照合。静的説明文、Route/canonical、配信時の価格/在庫Snapshotを別時点で扱う | 表示/配信の観測に限定。価格/在庫の最終確定と混同しない |
| **EP-3〜4 / #2516・#1174・#1180・#2018** | Flag割付だけで露出無し、Consent拒否、注文Commit後応答喪失、同じ変更のH0/H1/H2比較 | 未同意送信禁止、unknown Effectのlookup-first、工数/性能/回復差の実測。因果的CVR改善は独立の実験が必要 |

**公開情報の有効時点を同一視しない**：公開用説明文のBuild時Snapshot、Route/URLとVariant、Product/Offerの配信時Snapshot、Checkoutの現在価格・在庫、本人注文履歴のHost開示を別の判断とする。[Google Search Central](https://developers.google.com/search/docs/appearance/structured-data/merchant-listing)がProduct構造化データの初期HTML掲載を推奨するのは、古い価格の静的固定を認めることではない。会員価格、通貨、Variant、販促期間を変えた場合は表示HTML・構造化データ・現在Commerceの差を独立確認する。表面上の文言修正でも実効の価格・同意・公開範囲が変われば軽い変更区分に格下げしない。FlagのEvaluation ContextやHookは[OpenFeature](https://openfeature.dev/specification/sections/evaluation-context/)のProvider Adapter候補であり、AssignmentとBrowser Exposureを同一視しない。

**初回の停止線**：EP-0で既存TargetのRoute/Head Slotが未接続なら`unbound/unsupported`を記録し、#1121/#2090へ要求を返す。実Slotが見つかった場合だけEP-1の実Builder一Memberを試す。途中で該当Targetを作り足して「当初から利用可能」と扱わない。旧#5096のCLI/HTTP設計や#5119系のHost試験をB2C用の新しいHard Gateにしない。対象限定Go試験の実在と、B2C公開Host/HTTP・非Builtin Providerの証拠は別である。独立Expectedは#3509の既存BC/G49-EPを#1180/#2514へ返し、新しいScenario正本を立てない。全体P0はStudio Stage1→Stage2（#2831/#4583）、並行P0は通常製品`tei`の実Provider Read→独立二Consumer（#2586/#1251）で不変。**この節は設計・実装順の追記のみで、B2Cの新Go/Builder/Browser/Commerce/Consentの実験とOwner受理はNOT RUN / NOT ACCEPTED**。


<!-- g49-b2c-ep0-real-target-witness-20261116 -->
### G49 EP-0：既存Web Targetの能力を先に確認し、実Sourceと一つのHead出力を結ぶ（2026-11-16）

**今回の実装引渡し**：B2Cの初回は新しいSEO機構を作るのではなく、公開できる合成商品案内の`meta description`一項目が、**既存のどのTarget・Route・Slot・Builder契約に置けるか**を調べる。Sourceの意味とSource occurrence、要求される型・locale・公開範囲、配置先、実Builderの消費入力、生成Memberを一つの試行として対応付ける。見た目が同じ、型名が同じ、生成文字列が同じだけでは接続を認定しない。

**実コードで確認した足場とその限界**：`tools/reference-first-web-target/README.md`は既存実装を**診断用Web Reference**と明示している。`combined_scenario_templates.go`と`custom_ui.go`にはGo HTML templateと静的`<head>`があり、同じCommandをHTML/JSON/Custom UIへ投影できる。ただし、これらのファイルに**汎用の公開Head Slotを宣言した証拠は見当たらない**。`internal/artifact/manifest.go`には既存Artifact集合のManifest、`internal/buildintent/resolve.go`にはSource/Builder解決の契約があるが、この二つの存在だけで商品ページのHeadへ実挿入できるわけではない。これは確認したファイルの範囲内の所見であり、Repository全域にHead機構が存在しないとの断定ではない。

| 受入の順番 | 実際に読むもの・固定する条件 | 成立・停止の判断 |
| --- | --- | --- |
| **EP-0a：Capability棚卸し**（#1121/#2090） | 既存Target Catalog、公開されるRoute/Head Slot/Builder ContractとProvider/Generatorの版。調査対象ファイル・探索範囲も記録 | 実在Slotなら候補を固定。見つからなければ調査範囲付き`unbound/unsupported`。未探索を「不存在」や「全体適合」にしない |
| **EP-0b：型付きBinding**（#4350/#951） | 実WorkspaceのSource Subject/occurrence/Revision、公開Facet、単一のlocale、必要Shape/多重度、`present/null/not-fetched`、Audience/Disclosure、Adapterの有無 | 必須入力の欠落・曖昧な公開範囲・未宣言変換は拒否。非公開Customer/Orderを商品Headへ暗黙転送しない |
| **EP-1：Member実収録**（#1121/#2600） | 実際に選んだBuilderのConsumed Input、Source/World/Generator/Toolchain Pin、生成した一Memberと既存Manifest/BOMのReadback | 同一値でも別Read ID、旧Source/Plan、生成元と成果物の版不一致は不適合。Builderを呼んでいなければ`NOT RUN` |
| **EP-2以降：配信・利用**（#3271/#3503/#1174） | Response/Cache/Browserの実バイト、Experimentの実露出とConsent、Commerceの現在Price/Inventory/Receipt | EP-1成功から配布・検索エンジン受領・業務確定・CVR改善を認定しない |

**G/Pの同値性を強化する**：main統合[#5118](https://github.com/hat47x/tei-platform/pull/5118)は、Ruleの直接評価Pと生成Go Gで出力値とRead回数が同じでも、**Read IDの違いを検出する**限定テストを持つ。main統合[#5129](https://github.com/hat47x/tei-platform/pull/5129)はBuildPlan固定後の元Markdown差替えを旧Planで拒否する。main統合[#5142](https://github.com/hat47x/tei-platform/pull/5142)はRule判定でNULL・未取得・部分取得を区別する。これらをEP-0/1の独立Expectedに接続するが、Ruleの局所Go PASSをB2C Headの実Builder PASSとして流用しない。Function Contextの列由来／粒度の拡張[#5144](https://github.com/hat47x/tei-platform/pull/5144)は確認時点でClosed/Unmergedの候補であり、B2Cの現在適格性を先取りしない（#2586）。

**Head Slot未接続でもH0の価値を損なわない**：初回の既存TargetでSlotが発見できなければ、EP-0の結果を`unbound/unsupported`として固定し、#1121/#2090へ最小の拡張要求を返す。その後に案件所有のNative Headへ**任意のBuild-time成果物**を渡すH1方式を試す場合は、別の試行・別の資格として扱う。当初の製品能力の発見結果を後から書き換えず、必須Browser JS/DOM wrapper/通信Proxyは追加しない。H0 Native-onlyは正規の比較対象として残し、TEIを全Routeへ適用するよう強制しない。

**横断計画との整合**：main統合[#5128](https://github.com/hat47x/tei-platform/pull/5128)のWEB-P1〜P3はWeb公開方式とConsumer受入を所有し、main統合[#5130](https://github.com/hat47x/tei-platform/pull/5130)は型付きMappingと実Readを所有する。B2Cはこれらの機構の一Consumerとして検収する。API Pluginの旧[#5096](https://github.com/hat47x/tei-platform/pull/5096)と三入口試験の後継[#5137](https://github.com/hat47x/tei-platform/pull/5137)はClosed/Unmergedであり、正式な非Builtin Provider呼出しやWeb公開の合格には算入しない。独立Expectedは#3509から#1180/#2514へ引き渡し、第二Scenario正本を作らない。全体P0はStudio Stage1→Stage2（#2831/#4583）と通常製品`tei`実Provider Read→独立二Consumer（#2586/#1251）で不変。


**新しい共通機構をそのままHeadの適格性にしない**：main統合[#5166](https://github.com/hat47x/tei-platform/pull/5166)では、暦日の`not-after` Ruleは既存評価Pで扱える一方、生成Gでは`unsupported generated kind`として止まる。このためEP-0bでは、**対象のRule/Operatorごとに選択したG/P/D方式の能力、失敗分類、実Targetでの実行位置**を判定する。Pの存在からGへの暗黙Fallbackを認めない。mainの[Web-P1a設計 #5157](https://github.com/hat47x/tei-platform/pull/5157)にあるread-only ViewModelも、Head向けの生成器や外部Releaseを代替しない。

**公開FieldはWhole Recordの見た目から決めない**：[PluginのField Projection試験候補 #5156](https://github.com/hat47x/tei-platform/pull/5156)は、Record全体を扱うと`internalNote`など追加列を含む一方、明示二Field投影では含めないことを既存純粋Transformで対照する。現時点ではPRの統合資格は別確認であり、TEIの公開Host適格性を実証したものではない。B2CではProduct全体を`meta description`へ暗黙渡しせず、必要な公開Fieldを最小に選ぶ。NULL・未取得・部分取得とRead Footprint、Audience/Disclosure、生成Memberに残った原Source/埋込JSONも検証する。匿名Page/本人注文/運用者画面の各Audienceへ同じ結果を再公開しない。

**実Targetの入口を一段ずつ確認する**：現行`internal/buildintent/resolve.go`の`TargetCatalog.Check(target, profile)`はTarget/Profileの存在確認であって、Route/Head Slot/Builder出力Memberの存在証明ではない。Targetの存在、提供されたSlot、選択したBuilderと実消費、Member/BOMのReadbackを段階別の証拠として記録する。公開用のNative Headを案件側で任意に利用するH1と、TEIが標準Targetとして提供するSlotとを同一能力と数えない。

**現時点の資格**：この節は既存ソースの限定的な棚卸しと実装順の更新である。新規のB2C Head/Builder/Browser/Commerce/Consent/実験のGo試験、外部公開、Ownerによる業務意味の受理は**NOT RUN / NOT ACCEPTED**。

## 最新の作業単位へ接続する

2026年11月11日のmain `9d31f82486b2eac42791f4f3bc5cd588d69cb4fc`と関連Issueを照合した。以下は着手先の更新であり、Issueの起票・限定Reference・候補Parserを製品の完成へ数えない。以後の進捗は各Issueと[実装状況](../../docs/status.md)から再取得する。

| 既存の項目 | 現在の足場と着手先 | 次に満たす条件 |
| --- | --- | --- |
| G08〜G11/G37〜G42 | [#4350](https://github.com/hat47x/tei-platform/issues/4350)のS1〜S3（[#4351](https://github.com/hat47x/tei-platform/issues/4351) / [#4352](https://github.com/hat47x/tei-platform/issues/4352) / [#4353](https://github.com/hat47x/tei-platform/issues/4353)）は合成Headless Referenceとしてmain統合済み。横断Matrix [PR #4464](https://github.com/hat47x/tei-platform/pull/4464)と限定Source編集 [PR #4538](https://github.com/hat47x/tei-platform/pull/4538)はOpen（2026-11-14照合） | 読取り用構造写像の判定・用途別Observer・操作Slot単位の逆写像・現在Grant/実Consumerを分離する。11/32セルの合成判定やSource編集の隔離試験を実Provider→通常`tei`/Studio→独立Consumerの製品合格へ転用しない。単位・軸・完全性の未宣言はunknown/unsupportedで残す。Graph→TreeのNode属性Slotが写像全体の損失で一律拒否されるGapは保持し、既存Checkerで操作相対の条件を表せるまでは自動許可しない |
| G61 | [#4359](https://github.com/hat47x/tei-platform/issues/4359)が構文比較と実装順を所有。比較票は[#4361](https://github.com/hat47x/tei-platform/issues/4361)、候補Parserは[#4362](https://github.com/hat47x/tei-platform/issues/4362) | Parse、既存意味への変換、Source位置、互換性、正式採用を順に区別する。DSL案を製品Loaderの既定へ先に入れない |
| G19/G25/G61 | [#1130](https://github.com/hat47x/tei-platform/issues/1130)のETL/Batch設計は[PR #4409](https://github.com/hat47x/tei-platform/pull/4409)へ、取得・変換の最初の縦断は[#4410](https://github.com/hat47x/tei-platform/issues/4410)、再開・外部実行は[#4411](https://github.com/hat47x/tei-platform/issues/4411)へ分割済み | 同じ固定入力と独立ExpectedをGo/CLIで照合する。新しいETL専用DSLや必須Schedulerを加えず、再実行と外部作用の結果を分ける |
| G21/G23/G28 | プラグイン管理の一覧・診断は[#4376](https://github.com/hat47x/tei-platform/issues/4376)、計画・差分は[#4377](https://github.com/hat47x/tei-platform/issues/4377)、設定源と原子性は[#4378](https://github.com/hat47x/tei-platform/issues/4378) | 実Workspaceからの解決と選択、実呼出し、更新・失効の結果を別に観測する。管理画面の一覧を実Provider適合へ昇格しない |
| G54 | [#4134](https://github.com/hat47x/tei-platform/issues/4134)の検証義務候補P0-Aと、固定有限データP0-Bを[#1131](https://github.com/hat47x/tei-platform/issues/1131)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)へ接続 | 生成した試験の数と判別力を分ける。独立Expectedを固定し、満足例、単一違反、相互作用、観測不足と失敗の最小化を対照にする |
| G13 | [#2586](https://github.com/hat47x/tei-platform/issues/2586)がP0。旧PR #4076はclosed/unmerged。[PR #4539](https://github.com/hat47x/tei-platform/pull/4539)はEnvironment経由の`public.orders/:from`実PG読取、一時的Function Context、別プロセスReference CLIを実装し、exact-head CI成功後にmain統合済み（`ee1af44c`）。使い捨てPG16での局所実測も記録。ただし**製品`tei` CLI・独立Native Consumer・現在Grantは未接続** | Referenceの`public.orders/:from`とXTEAの`sales."order"/:status`は別Query。#4539 mainの実Read/Scope/接続先身元・Typed Param・Rowset・Context/独立Readerを基線に、RLS/0件/欠落列/マスクで`unestablished`を保持する。次に製品`tei` CLI、同一Attemptを受領する独立Native Consumerを順に検証。PG局所PASSやCI成功を現在Grant/全件取得/Checkoutへ転用しない |
| G49（B2C Native変更のCoverage） | [#3509](https://github.com/hat47x/tei-platform/issues/3509)の[最小B0判定票](https://github.com/hat47x/tei-platform/issues/3509#issuecomment-6813494503)をG49の初回受入候補に接続。案件固有Frontend/Edge/CMSは標準Rendererの採用を強制せず、実Source/Artifact R0→R1と既知のOperation/Context/Consumerだけをread-onlyで関連付ける。Native未登録部分の依存関係を自動で完全とみなさない | 既存BC-16/18/19/27/29/32/35/36/37/39/41を使い、通常文言C0と割引・同意条件を含む変更C3、旧Variant/旧Cartの正負を同一Fixtureで比較する。Change identity、Coverage、複数Lane候補、Required verification、Observed evidenceを別票にし、Coverage不足からNo Impact/公開安全/CVR向上を宣言しない。B0は**設計・試験候補**。#2586の実Readを先行し、H0/H1/H2の生産性比較は#2018、Commerce実Effect/実Browser/収益効果は後段で個別に検証 |

### 2026-11-13 データ様式の製品受入：問いに必要な根拠と操作範囲を限定する

既存[#4350](https://github.com/hat47x/tei-platform/issues/4350)の段階計画に沿い、次の差分だけを優先する。これは**新しい共通Schema・Dependency Engine・Runnerの採用ではない**。

- **問いごとの依存閉包**：[#4530](https://github.com/hat47x/tei-platform/pull/4530)のB2C限定Referenceでは無関係な未知依存の過剰失効を抑え、main統合済み[#4550](https://github.com/hat47x/tei-platform/pull/4550)でRequired Consumerの上流にある**多段の**未知入力も検出対象となった。ただしCaller申告Graph内での局所的な検査であり、Source Universeの完全性ではない。TEI全体でも、用途・対象・Scope・Required Source/Relation/SQL/Plugin・取得時点を先に固定し、必要な否定依存を確認する。Required集合が空または根拠不明なら「完全」としない。全世界の完全性をこの局所修正から推論しない（#3262/#2582/#1143）。
- **二Consumerの別々の適格性**：#4539の`Cell=value/null`や`rowset.fullyRead`は取得した行の観測であり、欠損/非選択/非開示の区別、集合の完全性、NULL由来、金額の単位/精度/丸め、外部Disclosureを保証しない。表の行表示と集計の個体・順序・多重度・数量法則について、既存[#2497](https://github.com/hat47x/tei-platform/issues/2497) OQ-01/02/03/05と#2514の独立Expectedを**同一Context・同一Attempt**で比較する。集計値を業務確定値としたり、Chartの一点を元行への一意な更新先としない。
- **読み取りと編集の条件を混同しない**：#4538の`set-attribute-type`限定実験では、Source定義・Span・Revisionを一意に決めた機械的編集と、Graph→Tree全体の非可逆性を分離する必要性が明らかになった。一律に「損失あり=全属性編集不可能」とする過剰拒否は解消候補とするが、Source一意性・対象Slotの情報保持・無関係部分の不変性・現在資格を実証するまでは、操作を自動許可しない。ABAと実Grantは未実証のまま残す（#2497/#2500/#1124）。
- **検証の順番と停止線**：R0a（#4539のmain統合・実PG/CI確認済み）→R0b（通常CLIで実ProviderとContext）→R1（異質二Consumerの受領と独立Expected）。Editorの参照・修正候補（#3788）、`*.tei.md`近傍直接テスト（#2621）、Studio Pin（#4515）、Native B2C（#4518）は別の並行作業として扱い、完了待ちをR0の新しいHard Gateにしない。

実PostgreSQLを用いた限定結果を含む#4539はmain統合済みだが、**この追補による通常製品CLI・二Consumer・実UI・現在Disclosure/Grant・意味受理の新試験はNOT RUN**。PRの状態と試験の達成範囲は、それぞれのOwnerと実行記録で判断する。

### 問いと操作に応じた判定の具体例（既存Expectedへ接続）

写像の分類（preserved/lossy等）は残し、判定対象の**問い・必要Observer・取得Scope・単位と時間・要求操作**を先に固定する。以下は新しい試験正本やCore判定語彙ではなく、#2497 OQ、#2514、#2586 R0P/R1C、#1143 RN、#2582 SC-IMへ渡す具体的な対照である。

| 既存対照 | 入力・問い | 独立に確認する期待結果 |
| --- | --- | --- |
| OQ-02／R1C-01 | #4539のPQ-02で3行を読了し、表が「取得した行は何行か」を問う。次に同じ結果で「全注文は3件か」を問う | 前者は観測した3行と答えられる。後者は`coverage=unestablished`のため確定できない。`rowset.fullyRead=true`を母集団の完全性へ転用しない |
| OQ-02/03／R1C-03 | 同じQueryでもRLSで可視行が3→2に変わる。また0行やマスクされたNULLを観測する | 可視行数と全体件数、DB保存NULLと秘匿由来NULLを区別。可視2行から全体のCOUNTや存在しないことを確定せず、第二ConsumerのChart・順位・診断にも開示条件を適用 |
| OQ-01/05／R1C-02 | JOINで同じ注文行が二度現れる、あるいは同じ行集合を順序なしBagへ写す | Rawの行重複・個体識別と結果粒度を保持。`SUM`・`COUNT DISTINCT`・`first`はそれぞれ独立の要件。観測行の数値和が計算できても、金額・通貨・丸め・全件範囲が未確定なら業務の確定集計としない |
| OQ-01／#2544 | 明示Fixtureで人物P1の所属0.6/0.4 FTE、P2の所属1.0 FTEを集計する | **単位FTE・対象期間・所属の重複処理を明示した場合**、人物2・所属3・FTE合計2.0を区別。現在のTEI SourceでFTE Unitや期間を宣言していなければ、この数値を製品上の受理済み意味として使わない |
| OQ-05/08／#4538 | Chartの集計点とGraph→TreeのNode属性Slotを編集候補として比較 | Chartの点・Graphの表示Edgeを元行/Relationへ直接書き戻さない。一方、個別Node属性は一意なSource Subject・Attribute・Span・版・操作の根拠が揃えば候補として検証できる。写像全体の`lossy`だけで全属性を永久に不可とせず、根拠の無い自動許可もしない |
| RN-07／OQ-06／SC-IM | Source A→B→A、別Workspace同ID・同byte、旧Pin/AI提案の後着 | 表示値やContent hashが戻っても、現在のSource/Workspace/対象/開示・編集資格を再確認するまでApplyしない。旧Replyで自動再束縛しない |
| OQ-04／RN-03 | `NOT EXISTS`/`SELECT *`などの暗黙依存が変化、または無関係なPlugin/列だけ変更 | 問いに必要な否定依存・Schema/Parser/Pluginが未知なら当該Claimを未確認とする。無関係なUnknownまで全Claim失効にしない。#4530の局所結果を全TEIの依存完全性証明へ昇格させない |

**最初に実測する組合せ**はPQ-02の同一Readを表Consumerと集計系の独立Headless Consumerへ渡すOQ-02/05でよい。まず両者が**同一Context/Attemptを実際に受け取った証拠**を残し、未確定のSUM/不存在/書戻しは正しく保留できるか調べる。FTEやGraph→Treeの実Source編集は後続の個別Ownerで検証する。実Consumer・数量/編集Owner受理・現在Grantの新しいPASSはこの表では主張しない。

### Native項目と実Query・Result Consumerをつなぐ際の対応根拠

2026年11月12日夜に[PR #4540](https://github.com/hat47x/tei-platform/pull/4540)がmainへ統合され、公開合成Native PDPの一つのFieldを、実Workspaceの`semantic-attribute:sales.order.order_id`・Source occurrence・Workspace Root・Load時Source bytesへ限定的に束縛する**Go test-only**の検証が成立した。これはNativeコード/DOMの自動解析でも、FunctionやSQLの実呼出しでもない。対象の[main統合済みPR #4539](https://github.com/hat47x/tei-platform/pull/4539)では`order_entry.recent_orders`の入力`:from`が実Query定義では未宣言であり、同じWorkspaceに`order_id`属性があるというだけでNativeの型付き入力、`public.orders.order_id`列、実Providerの接続先を推定できない。

次の小さな受入では、**Native Source Subject→明示Function/Operation Input→Queryの宣言・Parameter Shape→選択Environment/Instanceと実呼出Provider→Rowset列/Attribute Mapping→Function Data Context→Consumer実受領**の各辺を別々のEvidenceとして照合する。後段まで確認できない辺は`unbound`/`unobserved`/`unknown`のままとし、名前一致・同一Digest・別Readerの値一致で補わない。R0実Readの同一AttemptをB1aへ再利用できる場合のみ実Consumer接合を後段で試す（#4518/#2586/#2497/#3262/#2516）。

既存Expectedに足す**一因子の負対照**：①Native `order_id`と未宣言`:from`を同じ列/入力と自動接続しない、②別Workspace同名属性・旧Source occurrence/版を拒否、③同じQuery Source digestでも実Connection/Instance・Grantを確認できなければ受領資格は保留、④別独立Readerの検査成功をNative Consumerの実受信と誤報しない。Source修正前後の型付きMapping/Source位置の照合が不可能なら、公開/書戻しは保留する。新API/Registry/Parserを設けず、Provider Read #2586を優先しB2C Nativeを新たなHard Gateとしない。

**全体優先目標との関係**：TEI Studio Desktopのセルフビルド（#2831等）は引き続き独立最優先課題。このデータ様式計画は実Read・Trace・Consumerの検収で基盤品質を高めるが、Native B1aや二Consumerすべての完成をセルフビルド開始条件にしない。
### 三要素分析法・CRUDの派生結果を表現する際の適格性

main統合済み[#4534](https://github.com/hat47x/tei-platform/pull/4534)では、明示CRUD定義の矛盾を全件報告し、`okScope=structural-diagnostics-only` と `crudConflictStatus=conflicts-require-review` を分けた。構造診断`OK`、CRUD表が生成できたこと、業務上の整合・意味受理は同じではない。X-TEA由来のData/Function/Businessの図・CRUD cross-referenceも、Sourceへの参照・逆参照とレビュー要否を保持してからStudio/SHOへ投影する（#1124/#4472）。

さらに現行`main`の`internal/threeelement/crud.go`には、[#4551](https://github.com/hat47x/tei-platform/pull/4551)の明示CRUD-onlyのFunction×Data行の補完と、矛盾行の`conflictingDeclarations`、`UnusedData()`の明示CRUD参照を考慮する修正が反映されている。ただし、これは**既存三要素モデルのread-only派生とその局所試験**の範囲であり、Studio/SHOの全Consumerが矛盾を不適格停止として扱うことの証明ではない。最低限、(a)明示-only、(b)Operations由来、(c)同一ペアで双方宣言、(d)矛盾定義、(e)同Kind同IDの重複、(f)宣言が指す未知Functionを対照として、表示行数と参照数/unused候補を独立Expectedに照合する。`read-only派生成功≠実CRUD Provider Effect≠現在編集Grant`を守る。新しい三要素モデル正本や第二CRUD生成器は作らない。

### 異なる取得元・表現形式・実装Targetを横断する際の検収

この段は、#2506のCX-C01〜10、#2497のBPR-01〜05、#2498のXT-01/02および既存#2514/#1180の独立Expectedへ引き渡す**検収条件**であり、新しい標準Codec・全形式Adapter・共通Proof Engineの採用ではない。

| 比較対象 | 具体的な反例・検収条件 | 到達範囲と担当 |
| --- | --- | --- |
| **取得の完了** | `rowset.fullyRead=true`（一Cursor読了）、REAのTool/Evidence `complete`（その取得・Pageの読了）、Functionの必要母集団に対するCoverage、Consumerの開示可能性を別々に示す。Tool absent／欠けたPage終端／RLS／0行／部分対象で全件不存在に昇格させない | #2506 CX-C01〜10、#2586、#3394、#2516。main #4539の完了はCursorのみ。外部REA main #4547のObservation/Relationは意味受理でない |
| **生値と直列化値** | Raw `[]byte{0xff}`と`[]byte{0xfe}`はJSON化で同じ置換文字になり得る。現`functioncontext.cell()`の`[]byte→string`とJSON化したRowの重複検出では、Raw byteの異同・Binary Identityまで証明できない。異なるbyteを同一行や同一キーとして確定しない | #2497 BPR-01〜05、#2506、#2498。Product bytea/invalid UTF-8での実テストとLossless Codecの採用は未実施。元Rawとの一致が未確認ならConsumer適格性は保留 |
| **Presence・軸・Quantity** | Protobuf implicit scalarと`optional`のPresence差、xarrayの次元名/座標alignment、Arrow Mapの業務Key一意性、地理CRSの軸順を値/shape/表示順だけで推定しない。組織合成Fixture固有の`raw 100=1 FTE`は金融の100bpとは違い、Stock/Flow・JOIN多重・加重平均・時点・丸め順序を別に検査 | #2498 XT-01/02、#2495/#1705/#2544、#2593。外部ライブラリでの限定比較とTEI Product Adapterの受入は別。新しいCore Unit規約は本追補で採用しない |
| **生成物と実Consumer** | 同じPQ値を返すDriver/PG ReadとSource生成Go CLIでも、選択Provider・Target/Snapshot・Generator/Artifact Revisionが違えば同じAttemptとは言えない。Studio生成Inspector readerの実Windows利用（Draft #4572）と、三要素分析法のモデル用Bridge Pluginのserver画面到達（Draft #4570）を、TEIの一貫したDesktopセルフビルド/実Wails三視点受入と同一視しない | #4559/#1121、#4572/#2831/#1824、#4570/#1124。既存Trace・Receipt・独立Expectedを使い、Stage2 bundlingと実Desktop UIの未達を残す |
| **宣言の解釈の一意性** | 同じJSONキー・大文字小文字違いのキー・容量上限後のtrailing入力で、`declaredScopeClosed`等の意味を後勝ちで変えない。入力が一意に読めても、必要SourceのUniverseが本当に完全とは限らない | #4518の限定Referenceで#4565の負例を再利用。正式DSL/JSON全読込の実適用範囲は別Owner。現在Grant/Effectの証明には使わない |

**最初の着手順**：主線はmain #4539から通常`tei` CLI R0b→同一Contextの二Consumer R1。次にBPRの生値保持とCX-Cの完了判定をConsumerで反証し、Protobuf等の異種Codecは二系統程度から段階実測する。Studioの生成部品・三視点Bridge、組織FTE、外部REAは並走とし、どれもDesktopセルフビルドやR0bの新Hard Gateにしない。外部ライブラリ観測やDraft PRの局所PASSを現mainの製品機能へ格上げしない。

### 「空・不存在・未利用」を誤認しないSource束縛とDesktop検証

**参照の0件は成功ではない場合がある。** `tei source-index`に明示したSubjectが実Sourceに存在しないのに`ok:true`＋空配列を返す現象が[#4577](https://github.com/hat47x/tei-platform/issues/4577)で確認され、修正は[Draft #4584](https://github.com/hat47x/tei-platform/pull/4584)にある。**Subject未発見/重複、Scope内で参照0件、全Indexが実際に空、権限・未取得・partial**をそれぞれ区別する。特にStudioの固定選択（#4515）、項目レベル参照/未使用表示（#1143）、SHO図面/CRUDのApply判定（#1124）では、指定Subjectの取得失敗を「参照0＝未利用」や「対象なしにつき変更安全」に変換しない。正常なSubject特定→当該Source Indexの観測範囲→参照数→各Consumer利用資格の順に、独立Expectedを置く。

**入力解釈の一意性は依存閉包の前提であり、閉包の証明ではない。** mainへ入った[#4565](https://github.com/hat47x/tei-platform/pull/4565)はJSON重複・大小文字影・容量外トレーリングを拒否するが、`strings.ToLower`では`scope`/`ſcope`（Unicode simple-fold同値）によるGo `encoding/json`の上書きを再現できず、追加修正は[Draft #4580](https://github.com/hat47x/tei-platform/pull/4580)で検証中。`DeclaredScopeClosed`やRequiredの上書き、未知Sourceの偽閉包をそれぞれ判別し、Reference入力に限った修正を正式DSL/API全体の同等保証と扱わない（#2514/#4518/#3262）。

**実Desktopで部品が動いたこととStudio全体のセルフビルドを分ける。** [Draft #4579](https://github.com/hat47x/tei-platform/pull/4579)のWindows Wails/WebView2実CIは生成Workbench/Go ServiceをWebView2が受け取ったことまで観測し、[Draft #4572](https://github.com/hat47x/tei-platform/pull/4572)は生成Inspector readerの実Desktop使用を報告した。一方、[Draft #4570](https://github.com/hat47x/tei-platform/pull/4570)の三要素分析法Pluginはserver版の合成Source到達であり、Desktop実Window/実X-TEA原典は別資格。さらに[#4582](https://github.com/hat47x/tei-platform/pull/4582)のSelfBuild S2境界は**書込み共有のみ**制限し、読取り共有・実Builder来歴・Stage1生成Childを証明しない。次のDesktop検収は**実生成Source→Bundleのbyte/版→Wails埋込みArtifact→実Consumer受領→Stage1/Stage2 Buildの帰属**を分け、部分的な実Windows PASSを製品全体の自己再構築完了へ昇格しない（#2831/#1824）。

**旧依存辺を問いに必要な範囲で追跡する**：現mainの[#4591](https://github.com/hat47x/tei-platform/pull/4591)は、Caller申告の`baseOnlyEdges`でR0にあった辺の`to`を変更候補に加え、R1下流の影響を保存する。一方、このままでは**無関係なR0-only辺まで変更扱い**となり得るため、[#4592](https://github.com/hat47x/tei-platform/pull/4592)がRequired ConsumerからR0/R1の宣言辺の和集合を上流へたどり、影響が必要な旧辺に限る修正を提案中。既存#2514/#2582の独立Expectedで「二段の旧依存鎖の削除→当該Consumerは影響/上流不明を保持」と「別Consumerの旧辺削除→当該問いを過剰失効しない」を同時に区別する。**#4592は未統合**のため、現在の実装が両者を満たすとは扱わない。いずれも申告Graphに限る局所検査であって、実R0 Source/Worldの差分観測・完全なDependency Universe/現在Grantの証明ではない。G49のN16とSource不在区別を既存計画に保持し、第二Query/Graph Resolverは設けない（#4518/#2582）。

**同一HEADの生成Desktop受領を到達点として数える**：[#4587](https://github.com/hat47x/tei-platform/pull/4587)のexact HEAD `b26e87781e00b19bf142b22a35cfc62a857e88a3`では、生成Inspectorの契約digest/使用回数と生成Workbench JSの実WebView2配信byteを、同一Windows Wails Desktop Jobで検査して**completed/success**を得た。これは従来の別HEAD成功を足し合わせた証拠ではない。ただしPRは未統合で、Stage1自身がB1 Build Catalog/Intentを通じてStage2 DesktopをBuilderで実BuildするB2、そのStage2生成物の独立WebView2検査B3、実X-TEA Source、Current Grantは未受入。#4583/#2831の既存Gateへ証拠を渡し、この計画に別Build/Scenario正本を設けない。

この3点は既存#2514/#1180・RN/SC-IMとStudio/自動テストのOwnerが所管する反例であり、新たな共通Index/Parser/Build Planner/承認Engineを追加しない。#2586のR0b実Provider CLIと#2831 Desktopセルフビルドは引き続き最優先、#4584/#4580/全Plugin統合を両者の新Hard Gateとはしない。

### G08〜G13/G37〜G42：データ様式の用途・操作相対適格性を実Product経路へ接続（2026-11-14追補）

[#4350](https://github.com/hat47x/tei-platform/issues/4350)は段階・依存の索引、[#2495](https://github.com/hat47x/tei-platform/issues/2495)/[#2544](https://github.com/hat47x/tei-platform/issues/2544)はFacet/演算意味、[#2497](https://github.com/hat47x/tei-platform/issues/2497)は写像/損失、[#4133](https://github.com/hat47x/tei-platform/issues/4133)は表現型、[#2506](https://github.com/hat47x/tei-platform/issues/2506)/[#2586](https://github.com/hat47x/tei-platform/issues/2586)はFunction Data Context/実Read、[#1124](https://github.com/hat47x/tei-platform/issues/1124)は設計Source編集、[#2500](https://github.com/hat47x/tei-platform/issues/2500)は業務データのViewChange→Operation Intentを所有する。本追補はその**製品受入順序**だけを明確にし、第二の意味正本・型Registry・Projection Engine・Action/Grant/Host・Scenario SSOTを作らない。

**判断を混同しない**：A) 構造写像の`verdict`（既存5語彙）、B) **目的・問い・Observer・Scope・許容損失に相対的なread適合**、C) **対象Subject/Relation occurrence・Slot・具体Operationに相対的な逆編集候補**、D) **現在Source/Context epoch・権限・実Target/Effectの資格**は別列とする。一般の`lossy` Graph→Treeでも、明示ID・一意値span・恒等的に表示されたNode所有属性の局所更新は候補になりうる。一方、Edge IDなしのreparent/unlink、集計値の明細への自動配分、同ラベルからのIdentity復元は拒否する。これは既存Checkerの全写像`inverse=ineligible`を直ちに`eligible`へ変更する指示ではない。Slot/Operationスコープと独立Expectedを先に比較する（[#4533](https://github.com/hat47x/tei-platform/issues/4533)、[PR #4538](https://github.com/hat47x/tei-platform/pull/4538)）。

**同一性と現在性**：`Subject ID`／`Edge occurrence`／`Source byte span`／`View occurrence`／`Selection epoch`／`Workspace`／`Provider snapshot`／`Grant`を混ぜない。Raw byte fingerprint一致はA→B→Aの履歴同一性、現在の意味受理、Writer権限を証明しない。編集Apply直前に現在Source/Revision・span値・Writer権限・独立Requiredを再照合し、旧AI Replyや別Contextの選択から書き戻さない。設計SourceへのPatch/Acceptanceと実DBへのAction/Commit/Receipt/独立再Readは別の縦断で検証する。表現の`null / missing / unfetched / withheld`と集合の`empty / partial / complete / unknown`、Queryの非表示・否定依存（`NOT EXISTS`等）は用途相対の必要閉包を保持する。

| 順序 | 既存Owner・既存ケースを使う最小縦断 | 合格として要求する証拠 / 主張しないこと |
| --- | --- | --- |
| **D0：Reference契約の整理** | #4351〜53を基線に、#4464の11/32セル、`best`用途条件、`unknown/unqualified/uncovered`、#4533/#4538のSlot編集差を照合（XMAP-02/03/07、RSP-02、DF-X03、VE-01/03）。 | 同じ構造損失でもreadとSlot Editの判定が区別でき、既知Mutantと独立Expectedで偽許可・偽拒否を発見する。未検査21セルを安全/不可能と決めない。 |
| **D1：実際に読取り、別の利用先で確かめる** | #2586/#1251が担当。型付きSourceのRA-0と実行前検査のRA-1は修復後の仮統合試験[#4644](https://github.com/hat47x/tei-platform/pull/4644)で再検証済み。RA-2Rでは使い捨てPG16の7ケースを非SKIPで実行。次はRA-2Pで通常`tei`入口から実Provider・SQL・行・Contextを結び、RA-3で異なる二つのConsumerがそれぞれ受け取った値を検証する（DF-X01、HJ-03〜05）。 | RA-0のE1依存境界の失敗は別に残る。RA-2Rは参照入口の成功であり、製品CLIや受領者の成功ではない。入力不備は実行前に拒否し、実接続先や取得範囲は観測して判断する。空結果、RLS、NULL、部分取得と非開示を、独立した期待値で区別する。 |
| **D2：設計Sourceの局所編集** | #1124/#2497とPR #4538で、表/Propertyの一意な型付き属性Patch→既存Preview/Review/Accept→実Source再Read/再投影を接続。Graph→TreeはNode属性の正例と多重Edgeの負例を追加（DF-X03）。 | 異Workspace同名、旧Revision、A→B→A、Edge ID欠落、集計編集、非可逆投影を独立に判別し、無関係byte不変・失敗時書込0を確認。限定合成Passを現在のOwner意味受理へ昇格しない。 |
| **D3：業務Function操作・異種Target** | #2500/#1174で同じ明示Entity/Relationに対するIntent候補→現在Grant→実Effect Receipt→独立Read。#2498/#4133/#3827、SHO/SUI、外部Java/.NET Hostは用途に必要な異種Consumerを順に追加（DF-X04〜08）。 | Selection・UI配置からAction/Authorityを創出しない。実Host/DB、Owner Effect、外部Consumer、媒体同期の各Claimにexact版と実観測を要求。 |

**直近のP0実行カード（2026-11-14、RA-0/1/2R同一仮統合HEAD再検証後）**：新規の型・Parser・Storeを増やさず、**取得済みの限定Go＋実PG検証と、未達の製品Operation→実Provider→Context／独立Consumerを切り分ける**ことを優先する。これは既存#2586/#1251の進行表であり、第二のScenario SSOT・Gate・Product Operationを追加しない。

| 段階と現在状態 | 実装担当・最小変更 | 独立した合格／停止条件 |
| --- | --- | --- |
| **RA-0：型付きQuery Source（Draft #4626、現/歴史Source回帰成立、E1未解決）** | `order_entry.recent_orders`の`input from std.datetime`と`Query.Inputs`。現行Golden/World Fingerprintを更新、歴史旧QueryはSHA-256固定しFrozen Worktreeでのみ再現。修正HEAD `9eecf2b9`の[Health #43558268996](https://github.com/hat47x/tei-platform/actions/runs/43558268996)・[Self-build #43558269076](https://github.com/hat47x/tei-platform/actions/runs/43558269076)は成功。 | **18 CI中17成功／1失敗**。残るE1 [#43558269132](https://github.com/hat47x/tei-platform/actions/runs/43558269132)のAPI exe`sqlgen`2 symbol混入は#1182別Owner。SourceのMeaning受理・main統合は未成立。Allowlist緩和を合格扱いにしない。 |
| **RA-1：Executor前型検査（Draft #4631、修復RA-0上でも限定Go成功）** | 実選択`Query.Inputs`と`time.Time`実引数を照合。不正・未宣言等のpre-Executor Call0と正常Call1候補。元[#4631](https://github.com/hat47x/tei-platform/pull/4631)の成功に加え、修復RA-0を基底にした[統合Draft #4644](https://github.com/hat47x/tei-platform/pull/4644) HEAD `b5c23044`の[RA-1 Go/旧Web CI #43558633790](https://github.com/hat47x/tei-platform/actions/runs/43558633790)が成功。 | 同一仮統合HEADのGo成功は、E1の依存境界、正式製品CLI、実Provider/Current Grant、Owner意味受理の成功ではない。元PRの統合順・採否はOwner判断。 |
| **RA-2R：参照PostgreSQL16（Draft #4638、修復RA-0上で7ケース非SKIP再成功）** | [#4644](https://github.com/hat47x/tei-platform/pull/4644) HEAD `b5c23044`の[実PG16 CI #43558633773](https://github.com/hat47x/tei-platform/actions/runs/43558633773)で空の2 DB・RLS Role・独立Readerを使い、**親＋7サブテスト全PASS、SKIP 0**。時刻境界、3/2/1/0行、NULL/列欠落、別DB・JOIN・RLS/権限を再実証。RA-1/Issue-to-Designを含む当該仮統合HEADの**3/3 Workflow成功**。 | 参照入口`tools/reference-browse-read`の取得証拠だけ。通常`tei` CLI→実Provider、Function Context、独立二Consumer、現在Grant、Owner受理は未証明。元RA-0→RA-1→RA-2 Draftを無条件mergeしない。 |
| **RA-2P：製品Operation→実Provider→Function Context（次の実装）** | #2586/#1251。既存`ExecuteBrowseQueryInEnvironment`・`BrowseQueryExecutorMap`で選択Query/Source/Env/Instance/Executorを接続し、**実Connection DB/Role・Consumed SQL/typed Params・Raw Rowset/Context**を同一Attemptで独立観測。並走Draft [#4647](https://github.com/hat47x/tei-platform/pull/4647)ではRLS/Limit/関係差を観測しContextへ`partial`を運ぶW1〜W3実装候補があり、同HEAD **7/7 CI成功**。 | Provider自己申告A→実接続B、選択A→実B、別SQL/Source、旧Contextの負例を分ける。実行前はCall0、接続後の不一致は成功Context0。W1〜W3の成功や参照実PGを通常Product`tei`の実行・二Consumer受領へ昇格しない。 |
| **RA-3：通常CLIと独立Consumer** | #1251/#2586/#1824。既存Typed Application Operationを通常`tei` CLIと独立Native/Studio受領へ結線し、同じSource/Query/Context pinでTableとAggregate等の異質Consumerがそれぞれ受け取った値を確認。 | 各Consumerの`selected/invoked/received`を別々に実観測。JSON/stdoutの複写や固定UI Screenshotだけでは二Consumer合格にしない。Web共有Serverは[#4627](https://github.com/hat47x/tei-platform/pull/4627)の実Bridge exposure（command・外部Process含む）をそのまま公開せず、read-only facadeと現在Actor/Workspace Grantの実証を別に要求。 |

**直近の検証で分かったこと**：RA-0の現行World・Goldenと歴史用Sourceを分ける修正は[#4626](https://github.com/hat47x/tei-platform/pull/4626) HEAD `9eecf2b9`でHealth・歴史セルフビルドとも成功した。旧HEADで失敗していた未使用importやFrozen Readerの互換問題を、現在の未修正課題として数えない。ただし同HEADのE1検証[#43558269132](https://github.com/hat47x/tei-platform/actions/runs/43558269132)は、実APIバイナリに`tei/internal/database/sqlgen`のシンボルが2件含まれるため失敗している。これは#1182で依存の入り口を調べる。許可リストを広げて通過扱いにしない。

修復済みRA-0を土台にした[#4644](https://github.com/hat47x/tei-platform/pull/4644) HEAD `b5c23044`では、RA-1のGo・過去Web回帰とRA-2Rの実PG16・7ケース、Issue-to-Designの**3系統すべてが成功**した。基底PRのマージ、製品入口、現在の権限、業務上の意味の受理まで済んだわけではない。失敗・成功の履歴は試行とコミットごとに残し、異なるHEADの証拠を一回の製品実行にまとめない。

**段階別Contributionと共通意味を同一視しない（Plugin接合）**：[#4646](https://github.com/hat47x/tei-platform/pull/4646) HEAD `77463edb`は、共通のMeaning ID/Version/FingerprintとStudio・Generator・Runtime・Testの各段階Contractを区別し、既存契約Resolverで対応付ける**Reference試験**として3/3 CI成功。段階ごとのPackage版が異なる正例と、Meaning/Target/Scope/Provider/Artifactの不整合・曖昧選択・順序の負例を検査する。これは**実Pluginロード/配布、実Go Builder利用、DB Provider/Current Grant、Semantic Meaning受理ではない**。Pluginを本体とStudioに一体導入する判断は#951/#2639/#4632のOwnerへ戻し、将来のProvider/Target/Artifactの資格を「同じ意味Version」だけで承認しない。第2の意味正本・Resolver・Hostは作らず、Go単一バイナリの配布要件とStudioセルフビルド優先を維持する。

**Function起動とDB読取りの二経路を一つに潰さない（他チャット接合）**：L-Pは[Script Launch #4495](https://github.com/hat47x/tei-platform/issues/4495)が公開Entry→Typed Invocation→選択Capability/Provider→実呼出/Resultを所有し、L-Rは[#2586](https://github.com/hat47x/tei-platform/issues/2586)がBrowse Pattern→Query Inputs/Provider→実Rowset→Function Data Contextを所有する。現在のPG Read #4539 はL-Rの限定実行であり、公開Business FunctionのEntryを実行した証拠ではない。製品計画の**一機能**への受入では、各経路の`declared/selected/actual invoked/actual result/Consumer received`を同一の受理済みFunction/Workspace/Sourceに必要な範囲で結び、別Attemptの成功を新しい合成Runへ昇格しない。L-RのRead成功に業務Effectを足さず、L-Pの契約拒否成功にProvider選択・GrantやRaw Rowsetの証拠を足さない。ProviderとHostの認可・Result/Effectは既存Ownerに残し、共通Invoker/Runner・第二Authorityを追加しない。

### 異なる入力経路に共通する「選択→実受領」の証拠連鎖（2026-11-14更新）

本節は**既存の各Ownerの証拠を比較するための作業票**であり、全Target共通の新Core Schema、統一Invoker、第二の許可/Store/Runnerを設ける提案ではない。Source/Workspaceの同一性、契約・利用者の選択、実際に使われたbyte・実行者、受信者・開示範囲、Ownerによる受理は異なる判断である。**同じdigest、同じ表示値、同じ時刻、同じPRのgreen CIだけでは、その間の接合を実証したことにならない**。Source A→B→A、同名異Root、Actor/Grant失効、旧Responseの後着を個別の負例として残す。

| 経路／現HEADの局所証拠 | 最初に未成立な接合・実際に拒否すべき事例 | Ownerと受入順（既存Issueを使う） |
| --- | --- | --- |
| **L-R：Query→実DB**。[#4644](https://github.com/hat47x/tei-platform/pull/4644) `b5c23044`は修復RA-0＋RA-1 Go＋RA-2R実PG16（非SKIP7ケース）を**同一仮統合HEADで3/3 CI成功**。独立Draft #4626の17/18失敗1件はE1の`sqlgen`依存。 | **RA-2P通常製品Operationから実Provider/Consumed SQL/Contextを同一Attempt受領**、選択A／実Bを拒否。W1〜W3 [#4647](https://github.com/hat47x/tei-platform/pull/4647)は観測制限`partial`であり、native SQL全件性の証明ではない。 | #2586/#1180/#2514。RA-2Rは参照取得済み、RA-2PとRA-3二Consumerを次に行い、D1.2再集計は観測Scopeの上限を継承。 |
| **L-P：Function→Go Adapter**。[#4629](https://github.com/hat47x/tei-platform/pull/4629)はmain統合済みのResult契約不正→Adapter call0。[#4636](https://github.com/hat47x/tei-platform/pull/4636) `da807abc`はInput/Effect宣言不正・nil Contextのcall0を**exact HEAD 9/9 CI成功で確認しmainに統合済み**。 | 実入力値が型/意味に合うか、公開Entryが本当に選択されたか、Host現在Grant・選択Providerが実際に呼ばれたかを別に確認。宣言`Effect`≠Commit Receipt、契約拒否成功≠業務完了。 | #4495/#1174/#3422、既存Script Launch §17とその独立Expected。RA-2実Readの成功票へ加算しない。 |
| **R3：REA Evidence→Host→受信者**。[#4637](https://github.com/hat47x/tei-platform/pull/4637)旧HEAD `f38117dc`は**12/12 CI成功**。同一Raw EvidenceをRoot A/Bから独立に読み、Bの改変はProducer Receiptの不整合で拒否する回帰を修正した新HEAD `9b8f5ca9`は**12/12 Workflow完了・成功**（旧HEADの12/12とは別Attemptで検収）。Host固定Selector/Root、4MiB制限と子Process試験に限定。 | Raw Digestが一致してもProducer Deliveryの資格にはならず、A→B→A・Root差替えを現在Source/Grantの承認へ昇格させない。製品Studio/HTTP/MCPでの配送とAudience別Disclosureは未検収。 | #3394/#4503/#2516/#3419。旧HEADの12/12を新HEADへ移植せず、R3-C独立Consumer、#4627 Web公開は別Owner。 |
| **BuildIntent→実Builder→Desktop**。[#4634](https://github.com/hat47x/tei-platform/pull/4634)限定Builder pin 8/8 CI成功。[#4587](https://github.com/hat47x/tei-platform/pull/4587) HEAD `9b58533d`のWindows生成版・通常版WebView2を含む**12/12 CI成功**。[#4648](https://github.com/hat47x/tei-platform/pull/4648) `64fb0f9a`では同一PlanでGo 1.26.11／1.27.0の実コンパイラが**異なるArtifact byte**を生成したことを実測、Toolchainの申告とEntry側の再読取を追加し**8/8 CI成功**。 | `toolchainObservation=matched`は申告されたpath/digestの一致にすぎず、**実Compilerの選択証明でもPlanに含む入力でもない**（`namedByPlan=false`、`qualified=false`）。Toolchain pinの採用は#2831 Owner未決。Stage1→Stage2 WebView2/BOM/Seed Trustは別P0。 | #2831/#4583/#1119。別Compiler・同じPlan→別Artifact byteを再利用・Owner受理へ昇格しない。 |
| **Wails Server Bridge→公開Web**。[#4627](https://github.com/hat47x/tei-platform/pull/4627) `b9cf5447`はService登録・実Go exported 15 method棚卸し、**5/5 CI成功**のDraft。 | 一覧に`ExecuteCommand*`とSHO子Process入口がある。**静的InventoryのPASSは公開AuthN/AuthZでも安全なWeb配送でもない**。不正Actor/Workspaceからの実HTTP/WS拒否、二受信者間の秘密非開示、read-only façadeを別に確かめる。Browser提供の`allowed []string`をHost Grantと認めない。 | #1824/#3421/#2516。local loopback Server PoCをmulti-user公開へ自動昇格しない。 |

**RA-0の未解決作業票（旧Healthの修復待ちを完了に修正）**：現`input from std.datetime`と現行Golden/Fingerprint、旧QueryのFrozen Worktree内SHA固定復元は[#4626](https://github.com/hat47x/tei-platform/pull/4626) `9eecf2b9`で実装済み、**同HEADのHealthと歴史Self-buildもPASS**。未使用`reflect` importも削除済み。これらを再度「未実装」としない。残る実CIは**E1 Browser→JSON API binaryの`sqlgen`2 symbol混入**のみであり、#1182が実バイナリの依存閉包を調べる。Allowlist緩和・SKIPは不可。元RA-0/1/2の統合順・Meaning受理はOwnerが判断する。

**次の実装順**：① RA-0修復済みSource＋RA-1／RA-2Rの**同一仮統合HEAD #4644 3/3成功**は局所証拠として保持、② #1182でE1の実API依存境界を独立修復、③ #2586 RA-2Pで通常`tei`入口→実選択Provider→実Connection・SQL・Raw→Contextの同一Attemptを接続、④ RA-3の独立二Consumer受領。**E1の別Owner障害解消を理由にRA-2Pの設計・実装準備を停止する必要はなく**、main統合時には各Ownerが未解消CIを別資格で明示する。Desktop Stage1→Stage2の独立P0を遅延させない。Script宣言preflight #4636と計画#4642/#4645に加え、**型付きQuery・取得制約・実Build入力を切り分けるScript §19の[#4649](https://github.com/hat47x/tei-platform/pull/4649)もmain統合済み**。ただし同§はdocs-onlyであり、公開Function実行の資格を与えない。

**Function Data Contextの被覆・欠測・否定判断（TASK 9 W1〜W4、2026-11-14）**：[#4647](https://github.com/hat47x/tei-platform/pull/4647) HEAD `a962a3f6`は**7/7 GitHub CI成功**。別途、WSL2/Linuxの使い捨てPG16で手書きExpected・独立ReaderによるW1〜W3の非SKIP実行がPRに記録されている。ただし当HEADのGitHub「実PostgreSQL未実行スイート」成功は、新規のDSN付き統合試験の非SKIP実行を意味しない。
- **W1（実観測）**：Readと同じDB Transactionで宣言関係を`ACCESS SHARE` Lockし、実引数付き`EXPLAIN (VERBOSE)`、`relkind`、実Roleの`row_security_active`を取得。カタログの`relrowsecurity`やPlanだけで現在のRLS状態を決めない。Observation失敗はSavepointで隔離し、Read本体のエラーを隠さない。Lock以前のSnapshotと並行DDLは過剰な`partial`を生みうる。
- **W2（宣言関係）**：Persistence MappingからDataの宣言済み関係／主キーだけを解く。SQL本文・同名列から意味を推定しない。0件／複数候補は解決不能として保持する。
- **W3（被覆）**：RLS、LIMIT、宣言外JOIN、View/Partition/Foreign/Function scanなど**観測した制限のみ**`coverage=partial`と理由へ写す。制限が見えない場合でも`coverage=unestablished`、`open`には常に`native-sql-predicate-not-closed`。`complete`と`EmptyJustified=true`は出さない。STABLE SQL関数で基表3行を結果2行へ絞るのにPlan上は制限が見えない反例を保持する。
- **W4（列の由来、未実装）**：結果列の表OID／列番号が未観測のため、NULLを`stored`、粒度を`entity`と断定しない。`nullOrigin=unestablished`と`declaredKey`の一意性を分離し、実ProviderのFieldDescription/pgx等への小さな接続が必要なら既存Ownerで検証する。
- **D1.2／問いの閉包**：`partial`は「欠けた全行数の確定」ではなく、制限が少なくとも一つ観測されたという事実。0件から「存在しない」、RLSで縮んだRowsetから全件総計、重複JOINから元Dataの粒度を導出しない。Table/Aggregate等のConsumerへ`Observed Slice`と`unestablished/partial`を改変せず渡す。否定Claim・全母集団・必要集合の完了は#3262/Ownerが別に受理する。**W1〜W3をRA-2P/RA-3の新Hard Gateにしない**。

**RA-2RとRA-2Pの継承禁止（2026-11-14、実PG受入票）**：[#4638の実CI](https://github.com/hat47x/tei-platform/actions/runs/43553849861) は、空DB 2個の作成、`CONFIRM_ISOLATED`、RLS Reader DSN、親＋A/B/C/D/E/G/FのGo JSON eventの**fail/skipゼロ**をJobが検査したもの。これは実Reference Executor/DBの証拠であり、**正式Product `tei` CLIからのselected→invoked→receivedの証拠ではない**。既存`TestBrowseReadAgainstDisposablePostgreSQL`の7ケースを再作成する代わりに、次のRA-2Pへ既存の独立Expectedを参照して接続する。新たに必要な反例は選択A→実B、選択したQuery/SQL byteと実消費byteの相違、Context ID/epoch/Scopeの取り違え、Root・Source A→B→A、他ActorでのRead claimの不当な再利用。Null/Rowset/Maskingの値が参照Runと同じでも現在Grantを創出しない。Source/Provider/Target別の受入票は各Ownerに戻す。

**RA-2Pの最小接続点（既存コード確認に基づく）**：[RA-1の`internal/driver/browse_query_environment.go`](https://github.com/hat47x/tei-platform/pull/4631)には既に`ExecuteBrowseQueryInEnvironment`、`BrowseQueryExecutorMap`（`Environment+Database instance`をKeyとする）、任意実装の`DatabaseProviderBoundBrowseQueryExecutor`、`BrowseQueryEnvironmentExecution`がある。**最初の製品PRではこれらを再利用し、別の汎用Executor Registryを作らない**。ただし現`DatabaseProvider()`はExecutorの**自己申告**であり、`BrowseQueryEnvironmentExecution.Instance`も宣言上の束縛、`QuerySourceSHA256`はExecutorに渡したQuery Source byteのDigestである。いずれも**接続先PG16の実体確認ではない**。実際に開かれたConnectionのDatabase/Role・ReadOnly/権限、実SQLと引数、Raw Rowsetを、Product CLIの当該Attemptから独立観測して初めてRA-2Pを判定する。
- **P-正例**：型付き`from:time.Time`→実選択Env/Instance→該当KeyのExecutor→実PG DB/Role→Source SQL byte→Raw 3/2/0/境界→Mapped/Contextへ受け渡す。参照PGの手書きExpectedを読み直すが、**参照CLIの成功JSONを流用しない**。
- **P-負例**：宣言されたInstance Aに対してRegistryがBのExecutorを返す／Runtime Providerが異なる／Aと名乗るExecutorが実際には別DBへ接続する／Source SHAだけ一致してWorkspace/Revisionが古い、を区別。拒否すべき入力は**実DB Call 0**、接続後に初めて判別できる誤接続は**不適格Result（成功Contextを出さない）**として扱い、無条件に「呼出し0」とは表現しない。
- **後続へ渡す最低限の観測**：`selectedQuery/QuerySourceSHA256/SourceRevision`、`selectedEnv/Instance/Executor`、`actual DB/Role/SQL/typed Params`、`Raw/Mapping/Context epoch+scope/coverage`、`attempt/receiver`のうち、現インターフェースが保持しない値は**未観測をunknown**にする。Carrierの申告値、実OS/DB観測値、Consumer実受領票を一項目へ混ぜない。これは既存#2586/#2514の受入手順であり新たな公開APIや永続Schemaではない。

**RA-2Pの入口契約を閉じる最小実装票（生成Functionとの非同値）**：[Reference生成CLI #4559](https://github.com/hat47x/tei-platform/pull/4559)は、旧SourceのSQL `:from`を見つけても`inputDeclaration=undeclared`として残し、生成CLIは`--from`を**文字列**で受ける。ローカルPGで3/2/0行等の受入結果はあるが、型付き`Query.Inputs`や通常`tei`製品Operationの実証ではない。main統合済み[Script Launch §19 #4649](https://github.com/hat47x/tei-platform/pull/4649)もこの差を明記している。
1. **入口の宣言を一致させる**：まずRA-0現Sourceの`from:std.datetime`とSQLのPlaceholder集合を**既存Parser/Resolverの能力範囲で**照合し、未宣言・余分・重複を成功させない。Placeholder名から型を推定しない。SQL方言/静的解析で完全閉包を証明できない箇所は`unknown`を保持し、公開資格を得たと主張しない。
2. **CLIの文字列→Typed Operation境界**：CLI引数の受信を`time.Time`型適合とせず、型変換を行う入口・認める日時表現・時差と精度の扱いを既存#3422/#2586 Ownerが定める。変換後の`time.Time`だけをRA-1へ渡し、missing／空／明示NULL／不正形式／未知名／重複指定は**実Provider Call0**で棄却する。暗黙の時刻補正や`NULL`の代入をしない。
3. **Runtime・Context・Receiverを別照合**：選んだEnvironment/Instance/Providerと実接続DB/Role・Consumed SQL/typed Parametersを一致させた後、Raw Rowset→Function Contextの`scope/coverage/open/nullOrigin`を不必要に格上げせず渡す。**現在Grant/Audienceを評価した上で**二つの独立Consumerが実際に受領するかをRA-3で確かめる。RLS/関係名/非公開行の存在を示す内部Observationや`partial`の詳細理由は、B2C受信者へ無条件に開示しない。
4. **既存Scenarioでの判別**：旧Source`undeclared`なのに生成CLIが動く、受信文字列をそのままExecutorへ渡す、選択A→実接続B、RLSで0行なのに「注文なし」と表示、会員AのContextを会員Bへ再送、同byte A→B→AのRevision/Grant流用を異なる負例として検査する。ReferenceのJSON転記、UI Screenshot、静的生成Manifestは正常Product Call/独立Delivery Receiptの代わりにならない。

**横断受入設計との役割分担**：[main統合済み#4653](https://github.com/hat47x/tei-platform/pull/4653)は、定義から利用者の受入に至る全工程とCL-01〜10の負例をまとめた正本である。本節はそのうち、型付きQueryを通常`tei`の実Providerへ接続するための作業手順を補う。新たな受入台帳や意味定義を作らない。

### DSL診断から製品実行までの引渡し（#4652と並行する実装）

[設計#4652](https://github.com/hat47x/tei-platform/pull/4652)はmain統合済みで、診断と拒否を四つの段階に分けている。

| 段階 | 先に確かめること | 現状と受入の条件 |
| --- | --- | --- |
| A：Source解析 | 文法、識別子、元のSource位置 | 未知のEffectなど、構文上判定できる誤りは正しい位置で知らせる。Runtimeのエラーへ後から架空のSource位置を付けない。 |
| B：Workspaceの意味検査 | 型・Input名の重複、OutputとOutcomeの矛盾、参照先 | 既存Validatorで診断する。実装Draft[#4655](https://github.com/hat47x/tei-platform/pull/4655)はGoテストを追加したが、HEAD `3261d172`のHealthは整形不一致で失敗。mainの索引欠落は[#4659](https://github.com/hat47x/tei-platform/pull/4659)で修復済み。両者を混同しない。 |
| C：実行直前 | 不正宣言・不適合引数のとき、選択したAdapter／Executorを呼ばない | Capabilityの限定Input/Effect拒否は#4629/#4636でmainへ統合済み。Source→Adapterを同じケースで試す[#4654](https://github.com/hat47x/tei-platform/pull/4654)はGo試験の整形不一致でHealth未合格。RA-1の日時入力検査とは別経路。 |
| D：製品入口と結果の受領 | 公開Entry、実Provider、Context、現在Grant、Consumer | **未受入**。A〜Cの試験が通っても、Functionが公開され、正しいProviderから結果が届いたことにはならない。 |

A〜Dは新たな検査Engineの階層ではなく、既存Parser・Validator・Invoker・Hostの担当分担である。Studioでは診断一覧から該当Sourceへ戻れることを最初に確かめ、Source位置を正確に保持できない場合は誤ったジャンプ先を表示しない。新しい文法・Plugin採用や生成AIの常時接続を、Desktopセルフビルドの前提条件にしない。

### 読取りとSchema変更を切り離す

[main統合済みの生成アプリ修正 #4662](https://github.com/hat47x/tei-platform/pull/4662)では、生成された`query`／`command`／`validate`が、起動時にMigrationを暗黙適用する既存経路が見つかった。**照会を実行しただけでDBを初期化・更新しない**ことを、製品入口の安全条件とする。

`query`は既存DBを読取り専用で開き、`validate`はDBなしで構造を検査できるようにする。変更が必要な`command`もMigrationの暗黙実行とは分離し、DB作成・Schema変更は別の明示Operationと権限・Receiptに委ねる。存在しないSQLite DB、Memory DSN、別Schema、Migration履歴不整合を「暗黙に作り直して成功」としてはいけない。特に**通常`tei`のPostgreSQL実Read（RA-2P）と生成SQLiteアプリの修正（RB0）は別Target**であり、一方の試験成功をもう一方へ適用しない。

#4662は[統合PR #4683](https://github.com/hat47x/tei-platform/pull/4683)を通じてmainへ反映済み。修正元HEAD `095c5a11`は [Repository Health #43576908564](https://github.com/hat47x/tei-platform/actions/runs/43576908564)・Cache・既知脆弱性確認の**3/3 Workflow成功**を確認済みで、正規のGo vet／Go testを含む。ただし本番Schema・History・外部Provider・明示Migrationの適格性は証明されていない。Source Revisionをファイル名や仮のDigestで代用する問題も、[#2600](https://github.com/hat47x/tei-platform/issues/2600)の所有範囲として残す。#4234/#4125のDB移行計画と#2586のProduct Read計画を、同じ手順で安全性を評価できるよう接続する。

**判別の最短ループ（次の実装単位を固定）**：RA-1の**限定Go/歴史Web回帰の同一HEAD成功は取得済み**。修復済みRA-0のHealth/Historyと、#4644仮統合HEADのRA-1 Go・RA-2R実PG7ケース**3/3 CI成功は取得済み**。現在の別Owner障害はE1の`sqlgen`依存#1182のみであり、D1製品として次に不足するのはRA-2Pの正式CLI→実接続→Contextへの実呼出。続く**RA-2P**は、`実選択Query/Inputs→型付きactual Parameters→Environment内の実選択Provider→実Consumed SQL byte/実Rowset→Context Result Slice`を**通常製品Operationの同一Attempt**で観測する。選択A→実Call Aを正例、**選択A→実Call B**・Source/SQL別版を負例とし、SQL Placeholder全閉包・Snapshot/Current Grant・Coverageは実証範囲を超えて昇格させない。次にRA-3通常`tei`と独立Consumerへ進める。**RA-0/RA-1局所PASSをD1製品合格と表示しない**。D1.2 Measure、D2編集、D3 Effect、REA/SHO/SUI、Web共同編集は既存Ownerが並行するが、この最初の実ReadにHard Dependencyを増やさない。RA-1〜3・E1修正・実Web Grantの新試験は本計画変更では**NOT RUN**。

**停止線**：`Source Revision`・`Context epoch`・`Identity/Slot`・`Quantity/Time/CRS`・`Required Scope`・`current Grant`のうち当該操作に必要なものが未確認なら、read説明は可能でもEdit/Executeを許可しない。実行証拠はWorld/Model/Artifact revision、実選択Provider/Generator、実消費byte、独立Expected、Owner受理を分離し、既存[#2514](https://github.com/hat47x/tei-platform/issues/2514)/[#1180](https://github.com/hat47x/tei-platform/issues/1180)/[#1119](https://github.com/hat47x/tei-platform/issues/1119)のGateに載せる。新たな公開DSL・Core TypeKind・固定15表現Enumは異種Consumerの反例を得てからADRで判断する。


### REAを外部診断器として扱い、設計復元をTEIの主線に据える（2026-11-17、方式案）

[REA一Function縦断設計§35](../design/REA_ONE_FUNCTION_EVIDENCE_TO_DESIGN_VERTICAL_SLICE_2026-11.md)に基づき、外部Reverse Engineer Anythingを製品の実行基盤・必須解析入口とはしない。主な移行対象はソース・SQL/DB Schema・業務モデル・画面／帳票・設定・API・実入出力であり、これらを既存Reader／Source Index／三要素／Process／標準GraphへEvidence付きで投影して、Owner Review→生成Go／実Provider→独立Consumerの順に検証する。REAのNode/RelationやPartial/Unknown、元byte・版・出典の保持方法は共通規律として採用する一方、Ghidra/Hopper/IDA等の深いバイナリ解析は、ソースがなく代替できない証拠が必要な例外に限る。既存REAの互換回帰を維持し、解析器の機能数拡大をP0とはしない。通常CLI実PG Function ReadとStudio Stage1→Stage2 Windowsは最優先のままとする。

### 一Function移行の初回対象を決める停止条件（2026-11-17）

[REA縦断設計§37](../design/REA_ONE_FUNCTION_EVIDENCE_TO_DESIGN_VERTICAL_SLICE_2026-11.md)に従い、初回は旧Function・旧Data・操作入口の原本とSource Revisionを特定し、TEIのQualified ID候補との対応を別列でReviewする。X-TEAの合成`sales.order.submit`は旧原典との対応・正式commit・副作用防止が未確認のため、M-05以降の合格例として扱わない。原本が特定できなければ`blocked`で止め、実Read・生成Go・Studio Native受領の成功を推定しない。実PG Function Read #2586とWindows Stage1→Stage2 #2831は並行する独立P0であり、外部REAの採用は任意とする。

### 外部REAなしでも成立する一Function移行の製品検収（2026-11-17）

[REA縦断設計§36](../design/REA_ONE_FUNCTION_EVIDENCE_TO_DESIGN_VERTICAL_SLICE_2026-11.md)のM-01〜M-08を移行ベンチマークの受入票とする。出典二種類の独立Source Byte/Range/Revision照合、Qualified Data/Function候補の三要素図面化、旧Functionの実Read/WriteとOwner判断を分けてから、生成Go・通常CLI実Provider Read・旧新の入力/結果/Effect比較・Windows Studio独立Consumerを別Gateで検証する。REAのGraphだけでは正式な業務意味を受理せず、REAを使わずに同等の証拠が揃えば成功と数える。M-01〜04の設計復元とM-05〜07の製品P0を混同せず、M-08の外部Analyzer採否は必要性と維持費で判断する。

### REAの一Relation質問と現在Sourceへの対応を分けて検証する（2026-11-17）

[REA一Function縦断設計§34](../design/REA_ONE_FUNCTION_EVIDENCE_TO_DESIGN_VERTICAL_SLICE_2026-11.md)をD0の再検収条件へ反映する。#5180はmain統合済みだが当時の専用`relation-review`は整形差分で止まっており、機能配置とGoテスト完了は別に記録する。固定HEADの局所CIで整形・正負試験を確認し、同じProducer Edge IDが別Attempt／別Source Digestで再出現してもEvidence ID／候補集合Fingerprint／Source Revisionを分けるRQ-09を加える。続くR1-Aは既存Source Indexによる独立Byte/Range確認、Owner指定の正式Data/Function候補、入力・出力・Effectの実観測を別票で扱う。main #5212のSource位置表示、#5179の状態付きData/Rule、#5156のField投影、#5190のCRUD矛盾拒否を再利用し、見かけのGraph関係やNodeラベルを実Function Readや業務REA Eventに変換しない。通常CLI実PG ReadとStudio Stage1→Stage2のP0は不変。

### 外部REAの一Relation質問：構造の観測とFunction意味を区別する（2026-11-16）

[REA一Function縦断設計§33](../design/REA_ONE_FUNCTION_EVIDENCE_TO_DESIGN_VERTICAL_SLICE_2026-11.md#33-一つのrelationへの質問を外部occurrenceの読取専用reviewとして実装する2026-11-16)を、既存D0の限定的な実装・検収へ接続する。REA実ツール6.7/6.8の受領は[main #5158](https://github.com/hat47x/tei-platform/pull/5158)の固定HEAD CIで両版成功。その後段は外部`imports`等のEdge occurrenceを明示して一つ選び、既存`CandidateSet/ReviewPacket`から端点Observation・Source Revision・並行辺の個別ID・Unknown・Partialを保つ`relationQuestion`を投影する。Function/Businessを自動生成せず、Read/Writeを`not-established`として返す。既存Source Indexとの独立Byte照合・OwnerによるQualified Data/Function採択・生成Go/Builder実消費・通常CLI実Provider／Native独立Consumerは**別の検収**であり、この質問が成立しただけではD1〜D3もR1意味受理も完了しない。Schema/Graph EngineやExpected正本を増やさず、#3394/#1124/#1180/#2514と通常CLI実Provider #2586、Stage1→Stage2 #2831へ引き渡す。

### 外部候補・三要素・実Readの受領を結ぶ判別条件（2026-11-14追加）

この表は新しいScenarioや型Schemaの正本ではない。D0〜D3で必要な既存Fixture（#4350 DF-X01〜08、#2497 XMAP、#2500 VE、#4472 TE-CR、#3394 REA、#2514/#1180 独立Expected）を**同じデータ様式契約へ接続する順序と、各Ownerの停止条件**だけを示す。REA/SHO/SUIや三要素の実装完了を、実Provider→通常`tei`/StudioのP0 Hard Dependencyにしない。

| 接合点 | 共通データ様式の検査 | 必ず判別する正負・Owner |
| --- | --- | --- |
| **存在と探索結果** | `Subject existence`、`Reference count`、`Read coverage`、`Disclosure`を**別事実**とする。存在しないSubject・存在するが参照0・未列挙のSource・非開示・部分読取を空配列/成功一語に畳まない。 | #4584は「不存在Subjectの成功扱い」をmainで修正済みだが、外部REA→独立Consumerの同一性は未検収（#3394/#4603）。同Label別Subjectと、取得済み0件/未取得を既存Expectedで分離。 |
| **GraphとREA観測** | 外部EvidenceのNode/Edge ID・有向・並行辺・根拠Source・coverageを保った候補Projectionを行う。REA観測→TEI正式Entity/Relation・三要素のData/Function/Businessへの昇格はOwnerの選択と意味検査が必要。 | Node label一致だけで同一化、並行/逆向きEdgeの消滅、未取得Sourceから関係不存在を断定、外部CandidateだけでDSL/CRUDを書き換えるものを拒否（#3394/#1124/#2497、DF-X03/04）。 |
| **三要素CRUDの再投影** | `Data identity`と`Business/Function identity`、明示CRUD定義と推導Operation、矛盾・未掲載を別に扱う。図/表/相互参照の表示成功を業務意味の一致に昇格しない。 | 異なる索引の同名、CRUDDefinitionsのみのpairの脱落、同Pair矛盾の後勝ち、`OK:true`と未解決の矛盾共存を検査。実装Owner #4472、Source編集Owner #1124、DF-X04の再投影へ渡す。 |
| **実DBの行とモデルSubject** | Source Pin（論理Subject/Source位置）と、Provider Result Sliceの行Identity/Version・grain・scopeを分ける。RLS/NULL/empty/partial、JOIN fanout、集計をContract相対のRead claimとして照合。 | 限定PG実ReadのRowsetを同一Function/AttemptでTableとAggregate Consumerへ渡す。**別の既存合成Fixture**の`2行×3行=6 JOIN行`についてはEntity6件への誤昇格を独立に拒否する。独立Readerと二Consumerの実受領を証明するまでは未合格（#2586/#2506、DF-X01/02）。 |
| **編集と実行** | Studioの`Source span Patch→Preview/Review/Accept→再Projection`と、実DBの`Pending Change→Current Grant/CAS→Commit/Receipt→独立Read`を二経路に分ける。Selection PinだけではWriter/DB権限が生じない。 | 旧epoch/Source A→B→A、Row version競合、ACK-loss、非可逆Chart、未許可AI提案を正負比較。失敗または不確定時に自動再書込しない（#1124/#2500/#1174、DF-X05/VE-05）。 |
| **文書・多Consumer・開示** | SUI Card/Island/Canvas→SHO Markdownの非単射性と、Raw byte/改行/BOM/Parser版・Source occurrenceを保持。表示から消えた情報や非公開Evidenceの存在をConsumerへ漏らさない。 | 同じMarkdown byteでも異なる元Graphを識別できない。Source byte一致とRendered一致を別採点。内部Path/Locator、hidden count、診断/DOM/Tool応答への二次開示を既存Disclosure Ownerが判定（#2497/#2516/#3394）。 |

**D0→D1の最初の同一Attempt Witness**（既存Evidenceを結合する比較票であり新公開Schemaではない）：`Workspace+Source/Model Revision`、`実選択Query/Provider/実読byte`、`Context ID+epoch+grain+scope+coverage`、`Consumer A/Bの実受領結果と版`、`各ConsumerのObserver/必要閉包/未知`、`独立ExpectedとCurrent Disclosure/Grantの観測有無`を、同じ試行IDに照合する。各列が揃っていても**同一原子的Snapshot**やOwnerのMeaning Acceptanceを自動主張しない。Consumer Aだけ取得できた場合はAの限定Claimまでとし、Bの成功を作らない。

**順序と非目標**：まずD1の実PG Rowset→Table/集計の二Consumer受領を製品/実Consumer経路で証明する。並行してD0のGraph→Tree Node Slot編集と三要素CRUDの矛盾負例を既存Ownerが検証する。外部REAのR3（CLI＋第二Consumer）は#3394/[Draft #4603](https://github.com/hat47x/tei-platform/pull/4603)が所有し、証拠を再利用するがD1の先行条件にはしない。生成Wailsのread-only WebView2成功 [Draft #4579](https://github.com/hat47x/tei-platform/pull/4579)は固定Fixture/WindowsでのConsumer能力の証拠であり、上記の実DB Context受領・現在Grant・セルフビルドL3を証明しない。第二Resolver/Graph Engine/CRUD正本/Consumer Adapter/公開Gate・Core型・実行Hostは増やさない。


### 問い相対の証拠到達・異種Join・否定判断をD0〜D3へ適用（2026-11-14追加）

横断レーンのREA R3設計 [Draft #4603](https://github.com/hat47x/tei-platform/pull/4603) の **P/J/Q（局所観測／異種Evidence Join／問い相対の必要閉包）** は、標準データ構造やCore型の増設ではなく、既存#4350 DF-X・#2586 HJ・#2497 MP・#4487 TEA-Q41/Q43・#2514/#1180の独立Expectedで**主張の到達範囲を点検する方法**として取り込む。#4603のMASTER_PLAN追補を複製せず、本計画では通常TEI Functionと二Consumerで検証すべき差分だけを所有する。

1. **P：局所観測を壊さない。** 原Sourceの`Required / managed / actually loaded / parsed / selected Function+Query / provider actually invoked / rowset actually observed / consumer actually received`を、既存Reader/Operation/Evidenceから**別々の事実**として照合する。この列は一律の直列APIでも新しい状態Enumでもない。構成上Selected Aでも実Invoke B、SourceをLoadしても実Callなし、Raw digest一致でもParser/Role/Plugin pin更新、Consumerに配送せずCLIだけ成功を、合格へ縮約しない。各段で`UNBOUND_DATASOURCE`、未取得、empty、partial、Provider Error、非開示、Current Source/Grant未確認を該当する観測根拠と結ぶ（#2586/#1251/#2506）。
2. **J：異種Evidenceの結合を無断で行わない。** REA External Node/Edge/Observation、三要素Data/Function/Business、TEI受理済みMeaning、Provider Rowset、旧システムのActual、TEI TargetのActualはProducer・Subject/occurrence・granularity・有効/観測時刻・Source版が異なる。同名、同IDの別Namespace、同じJSON/Source digest、同時刻、同じUI表示では結ばない。Qualified MappingとOwnerの受理範囲を明示し、未受理なら並列候補とunknownを残す（#3394/#1124/#4472/#2497）。
3. **Q：否定・全称の主張は必要閉包を検査する。** 「参照0件」や「三要素図に表示されない」から、未使用・存在しない・安全に削除できる、と結論しない。各Questionに必要なSource/Relation family/動的呼出し/未ロードPlugin/手作業/未取得page/必要権限を既存Required Setで点検。十分でなければ`observed=0`と`absence unknown`を両立して表示する。非公開Source有無で公開件数・diagnostic・DOM/ARIA・Tool JSONが変わる二世界の反例は既存#2516へ渡す。R2のGraphCoverageをData/Function/Business全分野のCompletenessへ一般化しない。

**既存受入に追加する独立の判別（新Fixture正本は作らない）**

| 既存ケース・Owner | 受理側が必ず区別する二つのWorld |
| --- | --- |
| TEA-Q41／#4487・DF-X03/04／#4350 | REAがData候補+Edgeを届けGraph配送completeでも、未観測Function/Businessは`unsupported/unknown`。図の0件と実不存在を別主張にする。 |
| TEA-Q43／#4487・DF-X01/02／#2586 | 旧SourceのFunction宣言と現PG `order_entry.recent_orders`のRead成功だけでは同一Functionの旧実行・移行一致を主張しない。Legacy Actual (L)、Owner Accepted Desired (D)、Target Observed Actual (T)を別観測で比較する。 |
| TE-CR-01〜04／#4472・DF-X04／#4350 | Function×Dataに現れた宣言occurrence数、CRUD表のセル数、実業務更新回数を分ける。Index取り違え・明示のみpairの消失・矛盾後勝ち・`OK:true`への過剰昇格を検出。 |
| HJ/DF-X01／#2586・#2506 | 実Readに成功しても`selected≠invoked`、Consumer Aのみ受領、B未受領、unknown/partialの隠蔽を非合格にする。2×3 JOINの合成反例は現PGの実Read事実と別証拠で扱う。 |
| DF-X05／#1124・#1174・#2516 | Same byte/Source A→B→A/Grant変更・旧AI Replyを組み合わせても編集/再表示資格を自動復帰させない。低権限Consumerにprivate Evidenceの存在・パス/件数を推測させない。 |

**REA二つのレンズを使ったD0〜D3の引渡し（2026-11-15追加）**：[REA一Function設計§25〜28](../design/REA_ONE_FUNCTION_EVIDENCE_TO_DESIGN_VERTICAL_SLICE_2026-11.md)では、逆解析REAのNode/Relationと、業務REAのResource/Event/Agentを**同一のGraph意味へ自動昇格させない**。D0は一Function/一Data/一手作業の既存CandidateSet・ReviewPacketにSource occurrence・Unknownを保持、D1は実Provider Rowsetと宣言CRUDを別資格、D2は必要な場合にのみOwner受理したEconomic Event/Participation/Stock-flow Roleへ接続、D3は生成Artifactの実Builder消費と旧L/期待D/新T、実Effect/Consumerを区別する。実REA 6.8.0保存原本のGraph欠落はReceipt成功と候補投影NOT RUNに分け、合成FixtureのCLI成功をProducer実Graph成功へ昇格しない（main #4962/#4990/#5003）。独立Expected REA-XQ-01〜08は既存#1180/#2514/#2516で所有し、新しいCore/Host/Reader/Meaning Store/試験正本を作らない。**製品P0**の通常tei実Provider #2586 とStage1→Stage2 Windows #2831/#4583は優先・並行して維持する。

**D1への最小引渡し**：まず`order_entry.recent_orders`について同じ固定Workspace・Function・Provider選択・SQL引数・Source Revisionで、`selected → actual invoke → actual rowset → Context instance → Consumer A/B selected/invoked/received`の**最初の欠落辺**を一つだけ確定する。既存#4539の実PG16 Readと、別Headで成立したWebView2 Browser/InspectorやREA Consumerの成功を合算しない。同一Attemptと同一Contextでも全Source原子的Snapshotは推定せず、各Consumerの独立Expectedと現在の必要Scope/Disclosureを照合する。現時点でこの縦断の実行・Owner受理は **NOT RUN**。

**D2/D3への停止線**：P/J/QはRead主張の十分性判定であり、書込みGrantやOperation Intentの発行元ではない。設計Sourceは#1124のAccept→再Projection、業務Actionは#2500/#1174のGrant→Effect→Receipt→独立Read、外部Hostは独立Consumerとして検証する。静的ObservationやAIが示したSource spanから、DB Schema更新・業務Effect・公開権限へ自動昇格しない。Plugin/Build時の`declared/selected/actually invoked/observed/enforced`も既存#2018/#3278/#2831のEvidenceに接続し、CLI/Browserの限定PASSをStage1→Stage2自己再構築成功と呼ばない。


### Desktop独立編集／Web共同編集／AI候補に共通なデータ様式の保存・再資格化（2026-11-14追補）

[Studio #1824](https://github.com/hat47x/tei-platform/issues/1824)の Desktop＝ローカルWorkspace/ローカルGitによる独立編集と、将来のWeb＝共有サーバWorkspaceでのチーム同時編集は、**保存先・同期・Session・認可の違い**である。両者とCLI/AI/SHOは既存Application Operation／Qualified Subject・Source/Model Context／Preview・Acceptance／Function Data Context／Build Evidenceを共通使用する（#1124/#1251/#2621）。これはWeb共同編集機構の実装済みClaimではない。Wails Desktopの実WebView2 read-only成功 #4579をWebでの共同編集・Server HA・同時保存合格へ転用しない。

**6つの現在性・同一性を分ける**（形式/API名の採用ではなく既存Evidenceの比較観点）：(1) editorのunsaved **Draft/Buffer Revision**、(2) 管理Root中の**Source byte/occurrence/Revision**、(3) Owner受理済み**Model/Meaning Revision**、(4) UIの**Workspace/Branch/Session・Selection epoch/Context ID**、(5) 共有用**Git Commit/Tree/merge base**、(6) 生成器と入力閉包に基づく**World Fingerprint/Artifact Revision**。Sourceのbyteが一致してもA→B→A、別Workspace、別Parser/Role/Provider、Grant変更なら旧Meaning/Review/実行資格は戻らない。逆に同じ意味Subjectへの別表示位置・Markdown装飾変更だけでは意味変更を推定しない。

**Web共同編集の三つの競合レベル**を分離する（#1824が体験/Host、#2621がRaw Authoring、#1124が意味編集、#1174/#2598が実Effectを所有）。
- **L1 Buffer/Text**：同時文字編集・同期は草稿の問題。CRDT/Yjs等は候補であり、採用済みの共通Source Writerではない。未解決ConflictでもDraftは失わず別保存・Review可能とする。MarkdownのCRLF/BOM/未知Roleや未対応`*.tei.md`を自動正規化・登録しない。
- **L2 Typed/Semantic Change**：Stable Subject+Source occurrence/Slot+base Source/Model Revision+必要依存の現在性から局所Patch候補を導出。異なるSlotなら自動許可ではなく依存と受理範囲を確認、同一Slotの競合・Graph edge identity欠落・CRUD矛盾・旧AI提案は明示Review/保留。CRDTの収束やGitの自動MergeはMeaning Acceptanceでない。
- **L3 Git/Workspace/Build**：Desktopのcommit/rebaseとWebの共有Workspace確定版は別の操作。Merge後は実Sourceから再Index/再Validate/再Buildを必要範囲で再資格化し、Read-only Preview・Builder選択・実Builder起動・生成Artifact・Desktop実Consumerを別Evidenceへ記録。#4623の限定Plugin Builder選択→Child起動前digest一致はDraft・CI限定の成果で、非Plugin Wails DesktopのBuildIntent→Stage1→Stage2を証明しない（#2831）。

**原本と表示を分ける**：通常Markdownは原文を保持して保存できる一方、TEI意味定義・宣言付きTable/Graph・生成物はAuthoring/Meaning/Read-onlyを区別し、`*.tei.md`へのParser/Role/Loader正式採用前に自動意味昇格しない（#2621）。同じCanonical SubjectでもTable、Graph、SHO Document、SUI Canvas上のView occurrence/座標は別。表示・Selection・AI pinからSourceのWriter・DBのAction・Owner Authorityを発行しない。外部Host/TEI Studioの実Consumerへ開示するContextはAudience/用途相対の許可済みSliceから作り、秘密の存在を公開件数/unknown/診断/DOM/Tool結果から漏らさない（#2516）。Source/Policyの変化をConsumerが**実際に観測したとき**、後着Replyの新規表示拒否と、既表示情報の資格失効を独立に評価する。検知されていない変更の即時撤回保証は主張しない。

**既存ScenarioとExpectedへ渡す5つの最小受入（新Fixture SSOTは作らない）**

| ケース | Desktop/Web/AIで共通に要求する正負対照 | Owner・開始条件 |
| --- | --- | --- |
| **CW-01** | 同じSourceの異Slotを2編集者が更新：Raw Diffで衝突しなくてもMeaning依存が交差する場合はReviewへ。片方の更新によってもう片方の未保存Draftや承認が黙って消えない。 | #1824/#2621/#1124。最初はHeadless 2 Draft/1 Sourceの合成試験。Webサーバ実運用は後段。 |
| **CW-02** | 同じSubject/Slotへの別値・同名別Subject・多重Edgeの差分では、last-writer-winsやGraph表示IDへの転写をしない。既存Source span/Identity/Revisionが一致しないApplyは0件。 | #2497/#4533/#4538/#1124。D2のSlot編集負例に追加。 |
| **CW-03** | Draft A→Source B→A、別Branch/Workspace・Parser/Role版変更・Git Merge後の同byteから、旧Model Acceptance・Review・Build Evidenceを自動復活させない。 | #2621/#1124/#2831/#2600。World/Model/Artifactを別資格として照合。 |
| **CW-04** | 旧Workspace/旧Grantで生成したAI提案や非同期Inspectが新Sessionへ後着：再束縛のない自動Patch/Result表示を拒否。既表示情報の失効と許可済み無関係領域の継続表示を別確認。 | #1124 DS-X11、#2598/#2516。実WebView2/Policy検証は別Claim。 |
| **CW-05** | 同じFunction Contextと許可済みSource SliceをDesktopとWebの異なるConsumerへ投影。各受領/Disclosureを実検証し、Reader/Consumerの受入不足を「描画成功」で代替しない。 | #2586 D1、#1251/#1824。Desktop/Web同時編集をD1のHard Gateへ追加しない。 |

**実装順**：D1の実PG→同一Context→二ConsumerはP0のまま先行。並行してD2でCW-02（既存#4538の局所Patch）とCW-04（既存#1124 DS-X11）のHeadless負例を先に採用候補として検証し、CW-01/03のWeb共同編集とGit Mergeの実装は後段の#1824/#2621へ。何らかのCRDT/集中DB/共有Lock/新Server/新Authorityが必要かは実測してから判断する。最新mainへ本Draft計画が取り込まれたこと、Web共同編集実装・Studio GUI/Policy/DB実験・Owner意味受理の成功は**NOT RUN / NOT ACCEPTED**。


### データ様式の動的意味：依存閉包・差分失効・複数Context保存（2026-11-14追補）

静的な **Structure/Facet→Projection→Operation適格性** だけでは、複数Data Contextが変化したときにどのClaimを維持・撤回・再評価できるか説明できない。[#2582](https://github.com/hat47x/tei-platform/issues/2582)（Query/Context依存とRefresh）と[#2597](https://github.com/hat47x/tei-platform/issues/2597)（複数編集Session/Save）、[#1706](https://github.com/hat47x/tei-platform/issues/1706)（Relation集合の粒度）、[#1708](https://github.com/hat47x/tei-platform/issues/1708)（複数Read Snapshot整合）、[#2564](https://github.com/hat47x/tei-platform/issues/2564)（期待状態/CAS）を**データ様式の時間・変化に対する適格性**として既存D1〜D3へ接続する。新しいReactive Engine、万能Change Log、Snapshot Coordinator、Composite Transaction Runtime、第二Session/Cache/Operation SSOTを作らない。

**変化の意味を4つの独立軸で追跡する**（新Core型名ではない）。
1. **Observed Result Slice**：Subject/Relation/path/aggregateの粒度、値とNULL/未取得/非開示、collectionの必要Scopeとcomplete/partial/unknown、実Consumed Query/Provider Revision。0行はread-covered sliceの結果であり世界の不存在ではない。
2. **Dependency/Required closure**：当該Claimが実際に必要とするField/Relation/Predicate/Query演算/時点。JOINやAggregateの出力列だけでは不足する。`WHERE`の非表示Field、`NOT EXISTS`・LEFT JOIN未一致・Top-N・LIMIT/ORDER・集合差等の**否定/非単調依存**と未知Dependencyを区別し、必要な入力閉包を説明する（#2387/#2582）。
3. **Freshness/Coherence**：Source/Rule/Model版、Function Instanceと選択epoch、Providerの各Read snapshot/評価時点、Consumer実受領を分ける。同じ画面時刻や同じ値でも複数Providerが原子的Snapshotを共有したとは扱わない。Providerが保証する整合性以上は`unqualified`とする（#1708）。
4. **Pending Change/Effect**：View/Sourceの表示差分、Instance+Anchorに属するDirty Change Set、型付きOperation Intent候補、HostがCommitしたResult/Receiptを分離。Read/Refreshは未保存Patchを消さず、Receipt/別Readの値一致だけで当該Attemptの因果Effectを断定しない（#2597/#2598/#1174）。

**局所維持の判断**：必要な依存範囲に変更がなく、現在性・必要Scopeが満たされるClaimだけ維持候補とする。直接変更を観測したClaimは失効/純粋再評価/再Read候補へ。影響依存が未解決なら`no impact`ではなく`unknown/recheck required`へ。Providerが証明できない差分適用の代わりには、既存Queryによるread-only再取得を基準とする。単なるUI再描画やAI推測を整合したデータ変更として扱わない。全面失効も誤成功と同様に計測し、対象外のConsumer/Contextを不必要に停止させない。

| 既存ケース・正本 | 順当な判断／誤判定を防ぐ負例 | 結合先 |
| --- | --- | --- |
| **RC-01/02 (#2582)** | 親H→D1明細2件・D2履歴3件。D2だけ変更してもD1のみの合計は局所保持候補、D1×D2 JOIN/依存Claimは再評価。Opaque依存が残る場合は「無影響」ではない。 | D1の二Consumer実Read後、D1.1 read-only Claim差分判定 |
| **RC-04/05 (#2582)** | LEFT JOINの旧`(H,NULL)`はD2一致行の後着で撤回され得る。NOT EXISTS/Top-N/集合差/集計も追加行で旧結果が消える。部分ページから不存在/完全集計を断定しない。 | #2387/#1706/#1708、D1.1 |
| **RC-03/07/08 (#2582)** | Parent A→Bで遅れて到着したA結果をBへ混入しない。他のInstance Aは保持。Rule改版のみとRenderer改版のみで再計算/再描画範囲を区別し、Provider別Snapshot不一致を明示。 | #1824/CW-04とD1 Consumer、必要ならGo/WASM同じ判断を比較 |
| **ES-01〜04/09 (#2597)** | 同じFunction Definitionの2配置I1/I2でもDirty/Selection/Resultは別。A→B→Aで未保存Patchを誤移転せず、独立Context D1/D2・page1/3に表示された行だけで全件更新を作らない。 | D2 Source編集・Function Edit、#2506/#2500 |
| **ES-05/06/11＋RC-06 (#2597/#2582)** | 2 Patchが同じUI Saveから送られてもall-or-noneではない。実Provider同一Tx確認まで個別結果を保持し、D1成功/D2失敗、ACK-loss/Commit unknown、期待版競合、再Readの因果同一性を別判断。 | D3 #1174/#2564 Effect/Receipt、業務Owner |
| **RC-09/ES-13 (#2582/#2597)** | 遅着・訂正EventのWindowと現在最終値、値は同じでも業務確認/Grantが失効した状態を区別する。可変Stream/Time軸の同値を値一致だけで決めない。 | #4133/#1708、必要なDomainのP2 |

**D1/D2の最初の縦断**：既存#4539の実PG Read→同一Function Context→TableとAggregateの二Consumer受領が成立した範囲でのみ、RC-01とRC-04の**限定Headless** read-only対照を続ける。RC合成Fixture（H/D1/D2）を`public.orders`の実PG Rowsetと混同せず、同一入力を都合よく別世界の証拠へ昇格しない。並行D2ではES-01/02/03/09の2配置・Dirty保持・部分取得を既存#2597/#1180へ委譲する。D3の実Tx・CAS・Receiptは独立Evidenceがある場合だけ検収し、E3実DBの別Operation成功を単一業務Saveの原子性に合算しない。

**実行配置の判断**：Desktop/Webの共通State/Context判定をGo Native／Browser WASM等に置く可能性は#1824/#1145のTarget/Performance比較対象だが、ここでWASM必須ABIや第二のBusiness Rule Engineを決めない。依存失効・Anchor/epoch・Operation候補の**同じ入力→同じ判断**がHeadlessと異種Consumerで再現することを先に確認し、I/O・Grant・CommitはHost/Providerに残す。表示反応速度と再Query回数・過剰失効/誤維持の率を分離して測る。

**今回の資格**：設計・受入順序の統合のみ。#2582/#2597のRC/ESケースは未実装・未実行の計画であり、実PG二Consumer、Headless selective refresh、実Web/WASM、実多Context Tx/Owner意味受理の新試験は**NOT RUN**。新DSL/Core/正式Reactive APIを本追補で採用しない。


### D1.2 測定値の意味・集計軸・再集計の条件付き適格性（2026-11-14追補）

静的なDecimal/NUMERIC型・`SUM/AVG/COUNT`名・Chart値一致だけでは、**Measure×集計軸×粒度×時点×対象集合×演算法則**の合成適格性を決められない。既存[#2593](https://github.com/hat47x/tei-platform/issues/2593)がMeasure/再集計の意味的Pressure、[#2544](https://github.com/hat47x/tei-platform/issues/2544)が演算Law束縛、[#1705](https://github.com/hat47x/tei-platform/issues/1705)がQuantity/Decimal/Unit/丸め、[#1706](https://github.com/hat47x/tei-platform/issues/1706)がRelation集合・fanout、[#2498](https://github.com/hat47x/tei-platform/issues/2498)が異Targetの実Consumer転写、[#2582](https://github.com/hat47x/tei-platform/issues/2582)が変化後の再評価を所有する。新Metric DSL/BI Engine/Query Optimizer/第二Measure Registry/独立Expected正本を導入しない。

**実装基線と資格上限**：[Draft PR #4423](https://github.com/hat47x/tei-platform/pull/4423) は、既存`reference/derivation-relationsets`の集約済み入力を従来`Sum`で再加算する誤成功を拒否し、read-only `Reaggregate(Claim)` で`eligible/ineligible/unknown`を説明する候補実装。PR記載の局所Go/SQLite検査・期待値・独立レビューは**限定Referenceの成果**であり、PR未統合、実SQL生成・実Provider集計・本番Consumer/Owner意味受理は別資格。特に宣言された`Partition`、`Scope`、`AsOf`の真偽や、構成要素の単位整合・丸めの由来を**実元Factから独立に検証しない**ので、`eligible`は「呼び手の宣言が成立する範囲での局所結果」であり、製品の集計保証ではない。

| 既存MCケース／独立Expected | 正当な合成・元Factの条件 | 偽PASSを許さない条件 |
| --- | --- | --- |
| **MC-02 平均** | A:2件×平均10＋B:8件×平均20 → (20+160)/10=**18**。sum/countの対応と被覆が必要 | 平均10と20から単純平均15。局所平均しか残らない場合は中間状態欠落 |
| **MC-03 比率** | A:1/2、B:8/10 → 9/12=**0.75**。分子・分母・対象集合の対応を確認 | 50%と80%の単純平均65%、分母ゼロ、分子分母の対象範囲不一致 |
| **MC-04 重複除去** | A={U1,U2}、B={U2,U3} → 全体**3人**。元Fact識別または適格な distinct 状態を要求 | 部門別2+2=4人、`SUM(DISTINCT amount)`で同額の別業務Factを消す |
| **MC-05 時点残高** | 同日A110+B190=**300**。時点と対象範囲を保持 | 日付を跨ぐ300+300=600を「現在残高」扱い。呼び手のAsOf宣言だけで元行の同時点保証を偽装 |
| **MC-06 取得・NULL・丸め** | 完全な0行／NULL／未取得／partialを区別。0.004×1000は厳密和4と、先に小数2桁丸め済なら0を区別 | 未取得をゼロ、部分AVGを全体AVG、ScaleなしJSON float64再解釈、先丸めと最終丸めを黙って混合 |
| **MC-10 異種Consumer** | 同じMeasure Contract/Observerの結果をGo/SQL/Java・Table/Chart等へ渡し、decimal lexeme/scale、Result Scope/Versionと実受領を対照 | JSON Parse/可視数値一致、集計器単体PASSを外部Provider/Business Meaningの同値とする |

**D1の合格判定を不必要に拡大しない**：まず既存#2586 `order_entry.recent_orders` の限定実PG Read→同一Context→TableとAggregate Consumer受領を成立させる（D1）。次の**D1.2（独立後続の試験単位であり新公開Gateではない）**で、#2593 MC-02/03/04/05の元Fact・明示Scope・演算法則と局所Resultを対照し、Provider側計算とConsumer側計算を実測する。#4423のReferenceは判別用の足場とし、`eligible`を最終製品適合へ自動昇格させない。Quantityの精度/単位・丸めフェーズは#1705、Query演算・JOIN重複は#2387/#1706、異Target転写は#2498/#2077、Provider別Snapshotは#1708と各Ownerで資格判定する。Measureの結果行を元Entityとして編集しない（#2500/#2497）。

**再評価の接続**：#2582の局所失効で`D1-only sum`が維持可能でも、平均の`count`、比率の`denominator`、distinctの共通Person ID、残高の`AsOf`、丸めRule等が変われば合法な再集計条件が変わる。再評価の必要集合には可視値だけでなく**Measure Lawと中間状態の出典**を含め、異なるSource/Rule/Provider版や非単調な入力変化を「値一致なので影響なし」としない。未知または開示できない入力から、許容されない精度・母集団の数値を捏造しない。

**今回の資格**：#4423の既存限定試験とは別に、この追補に伴うGo/SQLite/PG/Java/Wasm/実二Consumer試験・Owner受理は**NOT RUN**。P0実ReadとTEI StudioセルフビルドをMetric BIの全機能実装でブロックしない。正式Measure型・SQL書換え・Core拡張は反例と異種Consumerを検証してから既存Owner/ADRで判断する。

### 標準データの表現と実行を、元の意味へ結び付ける（2026年11月14日）

標準データをGraph、Tree、Table、フォーム、処理図で見せることと、それらを相互に書き戻せることは別である。各画面は同じSubjectと意味定義を参照するが、Sourceの原文、構造間の対応、実行時に選ばれた実装、利用者が受け取った結果を一つの同一性へまとめない。**新しい共通Runtimeや第二の意味正本を作るのではなく、既存の契約に不足する条件だけを実際の反例で確かめる。**

| 対象と設計上の位置付け | 最初に既存試験へ渡す反例・受入条件 | 担当と実装順 |
| --- | --- | --- |
| **構造と宣言的構文**：[main統合済みの設計 #4667](https://github.com/hat47x/tei-platform/pull/4667)はGraph／Tree／Tableの意味対応、JSON Schema、Language Plugin、Studio表示の**実装前の検討**。 | **SM-01／02**：同名Node・多重有向Edge・複数親をTreeへ表示できても、失ったEdge identityを推定して書き戻さない。**SM-06／07／08**：`oneOf`、再帰`$ref`、共有参照の制約と取得範囲を保持できなければread-onlyに退避する。外部Schemaの読み取り、JSONインスタンスの入力、DB Migrationは別操作。 | #2495/#2497/#988/#2620/#1124。まずHeadless表示とSource位置・独立Expectedを比較し、必要になった編集だけPatch→Preview→再解析へ進める。Grammar IR／新Parser基盤の採用は保留。 |
| **処理と図解**：[実行モデル #4665](https://github.com/hat47x/tei-platform/pull/4665)、[Lifecycle #4666](https://github.com/hat47x/tei-platform/pull/4666)、[実行規則の整理 #4668](https://github.com/hat47x/tei-platform/pull/4668)はmainに設計を統合済み。 | 同じFunctionでも**受理したDefinition／選択したEffective Plan／実際のObserved Trace／図面Projection**は別に比較する。**PX-13／14／15**：多重Edgeの誤結合、Provider AとBの取違え、常駐Webと単発CLIの実行規則の違いを判別する。処理切出しでは型、依存、評価順、時刻、Effect、Grant、実受領を欠かさない。 | #4198/#4193/#4502/#1121/#2514。Headlessの意味・分岐確認を先行し、Mock成功を実DB Commitや通知配達に格上げしない。中央Dispatcher／Graph Compiler／独自Hostは増やさない。 |
| **公開FunctionとPlugin**：[Script Launchの設計Draft #4669](https://github.com/hat47x/tei-platform/pull/4669)は未統合。[Plugin構成設計 #4671](https://github.com/hat47x/tei-platform/pull/4671)はmain統合済みで、両者の実行・配布の製品検証はこれから。 | 公開Entry→選択Capability／Provider→実呼出し→Resultを、Source解析成功と区別する。Studio／Generator／Runtime／TestのContributionが同じMeaningを指していても、実Packageロード・Builderの実使用・現在Grantは別証拠。RoleだけでEffect／Actionを実行許可しない。 | #4495/#951/#1016/#1180。既存Resolver／Language Pluginを使い、実Provider・受領者の確認を先に行う。P0にPlugin配布全方式や新ABIを要求しない。 |
| **専門ツールキット**：[数理・統計・計画・科学の設計 #4663](https://github.com/hat47x/tei-platform/pull/4663)はmainに統合済み。[SQLiteの限定実DB回帰 #4670](https://github.com/hat47x/tei-platform/pull/4670)はmain統合済み。ただしTEI製品Query・NoSQL・実Consumerへの横断は未実証。 | WBSの包含関係とPERTの先行関係を区別する。Window集計は粒度・順序・NULL・欠測を、Solverや科学計算は単位・精度・収束条件・独立Expectedを対象ごとに照合する。設計文書中の計算例をR／Mathematica／外部Solverの実行証拠とはしない。 | #4660/#2495/#2544/#2586。Window→WBS／CPM→Solver／DOEの小例を並行評価し、正式なOperator／Provider採用は実測後に判断する。 |

**意味保存を三つに分けて検証する**：[#4667](https://github.com/hat47x/tei-platform/pull/4667)の新しい設計判断を、別のCore enumにはせず、既存のProjection・Query・編集契約の検査条件として使う。

- **構造上の同値**：Node／EdgeのID、向き、多重性、Source occurrence、NULL／missingなどが、必要な範囲で保たれること。GraphをTreeに描けても、二つのEdgeを一つに畳んだ表示から元の関係を復元できるとは限らない。
- **評価上の同値**：同じ入力とProviderの下で、SQLのJOIN粒度・行順・RLS・LIMIT・否定条件・Snapshotなどを含めて、指定された観測が一致すること。列名と型が同じだけでは不十分である。対象全体の取得を確認できないときは、0件を「存在しない」、部分集計を「全件の合計」と説明しない。
- **変更上の同値**：表示からの合法な編集が、意図したSource occurrenceだけを変更し、再解析後にも同じ変更として観測できること。変更していないViewを保存しただけで原文byteを書き換えたり、集計値から更新先の明細を推測したりしない。双方向変換のGet–Put／Put–Get等は既存#2497の**条件付き試験の参考**とし、新しい一律の編集Engineを要求しない。

**次の小さな検証を三件に絞る**。それぞれ既存のIndependent Expectedと担当Issueを使い、別のScenario正本を作らない。

| 順序 | 試すこと | 成功・拒否の判断 |
| --- | --- | --- |
| 1．SourceとViewの安全な往復 | 同名Node・多重有向EdgeのGraph表示に、原文の書式を含む一意属性の局所編集を加える（SM-01／17／18／24、#2497／#1124）。 | 変更しなければSource byteは不変。合法な属性PatchだけをPreview→再Parse→再表示で照合。集計値の明細更新、古いRevision、共有参照先の曖昧な変更は拒否する。 |
| 2．読取専用Schema表示 | 固定したJSON Schema Draft 2020-12の単純・複合制約、共有／再帰参照、未対応VocabularyをHeadlessで読み、Source位置へ戻る（SM-06／07／23、#988／#2620）。 | `oneOf`と`anyOf`、`default`と値の確定を混同しない。未解決参照は不完全と示す。非開示対象の有無を補完件数や診断で漏らさず、編集はまだ有効にしない。 |
| 3．Queryから異なる二つの表示へ | #2586の通常製品Readが成立した**同一Attemptの実Rowset**をTableとAggregateへ渡す（SM-20、#2506／#2586）。 | 実Source・Provider・Row/NULL・粒度とCoverageを保持。RLS・LIMIT・Native SQLの閉包不明を全量へ昇格しない。各Consumerの受領を独立確認し、Reference PGの成功JSONを製品受領の代わりにしない。 |

**処理の切出し・Target移行は上記と異なる検証**である。[main統合済み#4673](https://github.com/hat47x/tei-platform/pull/4673)のPX-23〜29を#4198／#2514へ引き渡す。表示上の辺の置換を制御依存の変更と混同しない。純粋計算の抽出でも、暗黙Read、NULL・丸め・短絡評価、時刻、現在Grant、Generator／Toolchain、実行先の能力とConsumerの受領まで必要な閉包を確かめる。注文確定と通知を分割する場合は、Commit前後、ACK不明、重複配送の差が残るため、同じResult値だけで互換とは判断しない。**World Fingerprint／Model Revision／Artifact Revision、Build省略／Artifact再利用／Evidence再利用も別の判断**として保つ。

**RDBとNoSQLにまたがる統計のFit／Gap**は、[SQLiteの限定実DB回帰をmainへ統合した #4670](https://github.com/hat47x/tei-platform/pull/4670)のCTK-01／04／06を#2387／#2586／#3270へ渡す。四観測のWindow平均（10、15、20、30）、JOIN後の明細金額の過剰集計、全件取得できないDocumentへの黙ったローカル計算を、それぞれ異なる反例にする。Providerが保証する順序・Window・入力範囲・NULL／missing・費用・開示を確認し、能力が足りないときは理由を示して不適格にする。**このFit／Gapや専門ツールキットの実装を、StudioセルフビルドP0の新しい先行条件にはしない**。

**Studioへの引渡し**：どのViewからでも、同じ対象のSource位置・Revision・意味Role・未解決の制約へ戻れることを重視する。図の座標や描画用Edge IDを正式な意味IDにしない。人間とAIの編集候補は共通の型付き対象選択と差分確認を経るが、AIの提案を意味受理や現在の実行権限に昇格させない。未対応のDialect、部分取得、曖昧な逆変換では、読める情報だけを表示し、安全な編集は保留する。

**優先順位は変えない**：全体P0は①非Plugin StudioのStage1→Stage2 Desktop実再構築と当該Stage2のWindows WebView2利用、並行P0は②通常`tei`の実Provider Read→Function Data Context→独立二Consumerの受領である。上記SM/PX/専門ツールキットの試験は既存#1180/#2514の期待値へ分配し、新しい一括受入Gateを作らない。各設計PRのmain統合は実装済み・Ownerによる意味受理済みを意味しない。

## 案件の一巡で追加した七つの作業

申請、現場保守、公開、外部連携、設備監視、予約・制御の模擬実行を[製品設計](../design/PRODUCT_COMPLETION_DESIGN.md#案件の一巡から標準構成の不足を補う)で具体化した。以下の作業は既存G項目の読取り・外部作用・出典追跡へ接続し、案件に必要な入出力と寿命を補う。

| 項目・既存項目との差分 | 最初の作業と独立した正負の受入 | 段階と担当 |
| --- | --- | --- |
| G65 業務添付・大容量資産。G25の接続に、転送・関連付け・派生・利用終了を加える | 小さい合成資産を中断・再開し、確認した版へ関連付けて別Readで取得する。DBだけ成功、物体だけ成功、位置・内容の不一致、期限・権限失効、派生の途中失敗を試す | P1。転送・保管・変換の選択を既存構成へ結ぶ。[#4426](https://github.com/hat47x/tei-platform/issues/4426) |
| G66 アプリケーション内の全文・複合検索。G25の取得に、索引の追随と探索操作を加える | 同じ申請集合へ構造・全文条件とページ継続を適用する。更新・削除・索引遅れ・同順位・非開示の件数/候補/抜粋・失効した継続を対照にする | P1。既存Query/Contextの最小能力、言語のFixture、必要な索引の順で検証。[#4427](https://github.com/hat47x/tei-platform/issues/4427)、[#2622](https://github.com/hat47x/tei-platform/issues/2622) |
| G67 業務通知。G18/G19の送信に、宛先・本文版・期限と結果を加える | 同じ確定申請から合成送信先へ通知し、宛先ごとの結果を読む。確定前、重複、宛先変更、版混在、部分失敗、応答断、期限切れを試す。受付・配達を人の承認にしない | P1。既存Scenarioと隔離Providerを先に使い、実配信は別試験。[#4428](https://github.com/hat47x/tei-platform/issues/4428)、[#1130](https://github.com/hat47x/tei-platform/issues/1130) |
| G68 同期API・イベントの外部契約。G19の方式選択に、Schema・向き・版互換を加える | 一要求・一応答または一イベントをNative原本から既存Operationへ対応付ける。未知拡張、NULL/欠落、送受の逆転、旧Client、新Server、業務不成立と技術エラーを独立Expectedで照合する | P0でread-onlyの契約比較、P1/P2で必要な実接続。型・Functionは#4350、表層は#4359の責務。[#2542](https://github.com/hat47x/tei-platform/issues/2542)、[#3274](https://github.com/hat47x/tei-platform/issues/3274) |
| G69 継続イベントの時間・流量・再接続。G19の連携能力に、続けて受ける間の条件を加える | 固定イベント列で窓、遅着、上限、再開位置を比較する。Snapshot/差分間の欠落、重複、保持期限超過、確定後の訂正、購読失効を試す。過負荷の間引きを取引や実機指令へ流用しない | 必要な案件のP2。有限列の検査は先行可。Broker/Targetの保証範囲を明示する。[#2542](https://github.com/hat47x/tei-platform/issues/2542)、[#1130](https://github.com/hat47x/tei-platform/issues/1130)、[#3790](https://github.com/hat47x/tei-platform/issues/3790) |
| G70 設計に戻るデバッグと模擬実行。G06の出典移動に、観察・停止・再現を加える | 最初は同じ試行の入力・規則・分岐・結果をread-onlyでたどる。段階実行は対応Targetだけに追加する。時計・乱数・外部応答を固定した期限の対照で、再生時の外部作用0件を確認し、旧版・開示不足・対応外を明示する | P0で診断、P2で必要な段階実行・模擬時計。[#3789](https://github.com/hat47x/tei-platform/issues/3789)、[#1180](https://github.com/hat47x/tei-platform/issues/1180)、[#1131](https://github.com/hat47x/tei-platform/issues/1131)、[#988](https://github.com/hat47x/tei-platform/issues/988) |
| G71 利用者とSessionの一巡。G17の現在権限に、参加・回復・終了を加える | 初期案件で選んだ認証方式の登録/招待、再認証、組織切替、終了を通す。別IdPの同名主体、回復の誤結合、失効後の別タブ/端末、IdP接続断を試す。RP・IdP・進行中の操作の終了を個別に確認する | P1。方式別適合は[#3412](https://github.com/hat47x/tei-platform/issues/3412)、Sessionは[#1043](https://github.com/hat47x/tei-platform/issues/1043)、実作用の権限は[#1174](https://github.com/hat47x/tei-platform/issues/1174) |

P1では最初の申請に必要な基本添付・検索・通知・認証を選ぶ。動画の派生処理、専用検索エンジン、複数IdPの連携、無制限の継続配信までを一律の開始条件にしない。各担当は利用に必要な最小能力を先に宣言し、未対応と未実施を残す。

## 最初の申請を、部品の成功から案件の成立へ進める

公開合成データで組織Aの申請者と審査者、組織Bの対照利用者、添付一件、検索語、通知先を固定する。構成は一つの認証方式、DB内検索、小さいファイル、隔離した通知先から始める。独立Expectedには対象ID、確定した値と版、許可される操作、隠す対象、確認する外部結果を先に記す。

| 一巡の区間 | 同じ対象と版で確かめること | 必須の反例 |
| --- | --- | --- |
| 参加・下書き・添付 | 本人と組織の選択、未確定の下書き、内容を検査した資産の関連付け | 組織Bの同名主体、中断、DBだけ成功、転送だけ成功 |
| 検索・提出 | 本文と条件による同じ申請の特定、提出時の現在権限、確定した版 | 非開示の候補・件数、索引遅れ、基準版の競合 |
| 通知・審査 | 確定後の正しい宛先、同じ申請の独立再読取り、審査操作 | 失効したSession、旧宛先、二重通知、一部配信不明、旧版の承認 |
| 帳票・取得・訂正 | 同じ確定値からの帳票、許可付き取得、旧発行物と訂正版の別ID | マスターやテンプレート改版、取得時の失効、旧発行物の上書き |
| 更新・引継ぎ | 構成と版を固定した再実行、案件差分の保持、復旧手順の実行 | Providerの非互換、再生成による独自修正の欠落、復旧後の二重作用 |

この一巡では最初にHeadlessで業務操作と結果を確認し、同じScenarioをWeb等のTargetへ接続する。各担当の単体成功を合算して一巡の成功としない。同じ条件・同じ試行で必要な区間が通った範囲を記録し、画面操作、視覚品質、実配信・印刷、利用者の受入は該当する実環境で確認する。構造契約や合成Providerの検査はP0の実Readと並行できるが、製品の案件成立には実Readと利用側確認を省かない。

## 依存に沿って進める

| 段階 | 優先する利用と成果 | 着手に必要なもの |
| --- | --- | --- |
| P0 | 一つのSourceと改訂から、読める設計、通常の編集・診断、選択した実装、実Read、利用側での確認へつなぐ。独立Expectedと用途限定の文脈を用意する | 既存の管理Sourceと意味の再構築、実行・検証の契約を使う。提供実装の実呼出しと、Functionの実Readはそれぞれの担当で並行して進める |
| P1 | 標準の初回構成で一件を完了し、帳票、基本の複合表現、別Target、導入・更新・復旧まで検収する。性能と費用の基準を得る | 各操作に必要なP0の契約と結果。ヘッドレスで配置・編集差分を検証し、その操作をGUIへつなぐ |
| P2 | 公開サイト、非同期連携、画像からの帳票化、科学・時間媒体、複数Plugin生成を対照に共通化を見直す | 対象分野に必要な標準構造、変換、実行先を固定する。全部のP1が終わることを一律の条件にしない |
| P3 | VR、デジタルツイン、必要な構成の継続運用・段階更新・災害復旧を実環境で確認する | 対象の識別、時間と座標、入力と編集、権限、精度、資源上限に必要な検証。姿勢・観測の鮮度、継続中の仕事と復元後の状態を各実行先で確認する |

段階名は実装順の整理である。個々の作業は、その入力と契約がそろえば先に試せる。例えば帳票の確定データと既存エンジン比較は、複合表現やWasmの完成を待たずに進められる。複合表現の軸・選択・編集の評価もXR機器なしで合成データから始められる。GUIは対応する操作のヘッドレス検証の後に接続し、全分野を一つの依存鎖にしない。

## 最初に通す五つの利用

| 利用 | 同じ試行で確認するもの | 最小の反例と次の判断 |
| --- | --- | --- |
| 一つの受注Functionを読む | Source、Query、Alias、実Provider、Rowset、Context、利用側、独立Expected | 同名の別SQLや別集合、空・partialを検出する。読取りの成功を注文確定や発行の成功へ広げない |
| 一つの設計変更を引き受ける | 未保存の内容、診断、参照、差分、実際に使った定義と再試験 | 外部改訂、旧応答、同名異ID、途中保存を試す。利用者がSourceへ戻り、同じ条件で再確認できることを測る |
| 確定値を帳票へ出す | 同じ確定値からPDFとExcelを出し、保存済み発行物と訂正版を取り出す | 価格・コード表の改訂、多数明細、長文、部分取得を試す。値、版面、実印刷と読取り、配布結果をそれぞれ確認する |
| 表現から同じ対象を編集する | ヘッドレスの配置・選択・差分を、帳票と科学の小入力で照合する | 25%とmmの混在、投影で複数になる候補、旧版・権限不足を試す。その後にWebの保存・再読込み・PDFと別入力手段へつなぐ |
| 案件を標準構成から引き継ぐ | 受注・申請・公開等の必須条件、構成、実接続、更新と復旧を記録する | 未成立や対応範囲外も含めて集計する。既存部品で足りない箇所は短い依頼から拡張し、単体と組合せを検査する |

最初のFixtureは物流の由来とGISの座標変換という既存の分野横断計画も併用する。台帳、医療情報連携、設備、科学、映像、編集、教育・組織、申請、イベント、学習データは #3791・#2501 の事例として継続する。学習データでは同じ対象の学習・評価への混入、同名ファイルの改版、非公開行の混入を反例にする。分野の意味と受入判断を専門の担当へ戻し、共通形式に合わせるために意味を落とさない。

## 最新の横断検証を実Consumerの受入へ接続する（2026年11月14日）

局所的な試験が増えても、同じ要求を通常の製品入口から実行し、必要な情報を権限に応じた利用先へ届けたことは、別に確かめる必要がある。Queryの隔離PostgreSQL参照試験、取得範囲の限定観測、BuildのToolchain観測、外部Evidenceの読取り、Pluginの段階別Contribution、B2Cと組織の二時点照会を一つの成功として扱わない。

新しい検査Engineや独立した期待値の正本を追加するのではなく、各担当の試験へ共通の照合観点を渡す。対象Sourceと意味の版、実Provider・実Child、Contextの取得範囲、現在権限、最終Consumerの受領、独立Expectedを段階ごとに確認する。製品のP0は、**StudioのStage1→Stage2実Windowsセルフビルド**と、**通常CLI→実Read→独立Consumer**の二本を維持する。B2Cの本人確認、組織履歴、外部REA、Pluginの実使用は各Ownerの後続資格として進め、未確認の部分を完成扱いしない。

詳しい境界、10件の負例、Owner別の次の検収は[Sourceから実Consumerへの横断受入設計](../design/CONSUMER_QUALIFIED_CROSS_LANE_ACCEPTANCE_2026-11-14.md)に整理した。以下の「進行の記録」では、その観点を既存IssueのAttemptと結果へ結び付ける。

## CLIを起点とするTEIセルフビルドとPluginの段階的な自己構築（2026年11月14日）

TEIのセルフビルドは、最終的にはTEI Studio Desktopを含むTEI開発基盤自身をTEIで構築することを目指す。ただし**最初の実証にStudioの画面は必要ない**。既存のTEI CLIを起点として、TEIの定義から次の世代のCLIと小さなPluginを生成し、生成されたCLI自身が同じ仕事を続けられるかを先に確かめる。MCPは同じ型付き操作へ接続する別の入口として利用できるが、MCP Serverの完成をセルフビルドの必須条件にはしない。DesktopのStage1→Stage2と実WebView2は、このHeadlessな実証で得た証拠を引き継いで後から接続する。

**Goで書いた既存関数を呼び出すだけの`*.tei.md`は、セルフビルドの成功例とは数えない。** TEI定義に書いた条件や関係が、図表を含む意味モデルとして読まれ、実Goコードや設定へ生成され、成果物の振る舞いを変えることが必要である。手書きGoは当面のSeed、Target依存部品、複雑な計算の実装として許容し、TEIが担う部分と委譲する部分を明示する。Plugin自身についても同じ基準で実装範囲を広げ、定義を簡素化できたかを継続的に確かめる。

### 最初に通す五つの工程

| 工程 | 行うこと | 検収すること |
| --- | --- | --- |
| **SB-0 固定Seed** | 現行Go製TEI CLIを起点として、Source、Target、Builder、依存の版と入力を固定 | 既存Plugin Buildの正常・拒否テストを維持。非PluginのCLI／Studioを架空Plugin IDへ押し込まない |
| **SB-1 意味のある定義** | 一つのread-only Inspectorまたは純粋変換Pluginを選び、`*.tei.md`に処理・関係の**図**と、入力・出力・分岐・制約の**表または構造化宣言**を置く | 図のNode／Edge、表の項目、Source occurrence、意味候補、受理済み意味を対応付ける。未実装の正式DSL文法を捏造しない |
| **SB-2 実コード生成** | 同じ定義から必要なGoコードと契約情報を生成し、実CLIで読み・検査・実行する。TEI側の条件を変更したとき実際の生成内容・結果が変わる一例を作る | 実コンパイル入力と生成SourceのRole／Digest、選択したProviderの実Call、独立Expected。委譲用ラッパーだけなら未達 |
| **SB-3 CLI再入とPlugin自己構築** | SeedがStage1 CLIを生成し、その**Stage1実Process**がStage2 CLIを生成する。同じ仕組みで少なくとも一つのPluginをTEIから生成してCLIへ組み込み、実呼出しする | 親子Build Attempt、実Child・Toolchain・入力Byte・成果物・Entry／実Consumerの関係。Stage0が別々に二回Buildしたものを自己再構築と数えない |
| **SB-4 Studioへ展開** | 検証済みSource／Operation／PluginをStudioと生成Windows Desktopから使用する | 同じ意味と各Targetの実Artifact／実Consumerを照合。WebView2のB0成功をStage1→Stage2の自己再構築へ自動昇格しない |

最初から大きなDSL処理系を作らず、既存Markdown・Markdown表・Mermaid（必要に応じてGraphviz）・標準データ構造・Language／Operator／Target Generatorの機能をできるだけ使う。図の座標や説明文を実行意味の第二正本にはしない。外部で実装する部品はTEI側に型付きInput／Output／Effect／Failure／Provider契約とTODO／Mockを持たせるが、Mock成功と実Provider成功は区別する。Coreに新しい共通Runner、第二Resolver、専用Hostを増やさない。

### 同じ宣言を実現する方式は、生成Goと既存の汎用Operatorから選ぶ（2026年11月15日）

**TEI定義を実際の動作へ変換する手段は、コード生成だけではない。** 標準Graphの出辺列挙、表の型付き変換、既存Ruleの評価など、受理された意味・型・パラメータを既存の汎用Operatorが実行できる場合、Modelを入力としてそのOperatorを呼べば、用途ごとのGoソース生成・コンパイルを省ける。ただし、宣言の構文を読めたことと、正しいパラメータを渡して製品の実Operatorが動いたことは別の資格である。Targetに該当Operatorが存在しなければ、新Engineを自動生成したことにせず、Target向けGo生成／明示委譲／unsupportedから選ぶ。

SB-2の**実コード生成**はセルフビルドにおける独立した必須の実証対象として維持する。一方、通常のアプリ生成では「生成G」「既存汎用Operatorによるパラメータ実行P」「外部コード委譲D」を、対象の機能上の義務ごとに区分する。Pで意味ある処理ができたことはTEIの機能対応範囲へ数えてよいが、**生成した実装の比率やPlugin自身の自己構築率には加算しない**。GとPは同じSource／型付きMeaning／独立Expectedで比較し、PはBinary不変のパラメータ改版、Gは生成Go・実Builder・Artifactの改版をそれぞれ実際に観測する。更新・再Buildの時間、性能、保守負担、Target適合と失効範囲を測り、一律にGを選ばない。

最初の比較は、mainに限定実装がある`sales.order_line.quantity_positive`について、`GeneratePositive`で生成するGと既存`validationeval`系で型付きValidationを実行するPを同じ5入力（正値、0、負値、NULL、欠損）で対照する。これは**今後の比較計画**であり、Pの製品実CallとG/Pの同一Attempt E2Eを実証済みとは扱わない。Node/Edge/PortのGraph汎用関数は別の候補であり、正式Reader／実Operatorが未対応ならunsupportedを維持する。方式別の受入条件とGP-01〜06は[宣言的構文・Node表現の設計§27](../design/DECLARATIVE_COMPOSITION_AND_NODE_SURFACE_CONTRACT_2026-11.md)を正本とする。

**実装順は変更しない**：第一にRuleの生成Goを正式R2-P Builderへ接続し、並行して通常CLI実ProviderのR3と非Plugin Studio Stage1→Stage2を進める。P/Gの比較は既存Operator／Build／Scenario／Expectedの実装Ownerへ渡し、第二Runner・Host・Registryを追加しない。

### 最初の実装PRを、一つのGraph／Ruleから始める

S0〜S4を一度に実装しない。**固定6入力に対するGraph/RuleのGo生成と別Process実行は限定Referenceとしてmainで実証済み**である。次の製品コードPRはS0の正式なSource選択・Reader・Source Index接続に絞り、製品Meaningからの生成（S1）、選択Plugin実呼出し（S2）、Stage再構築と定義変更による再生成（S3/S4）は同じFixtureで別に検証する。

最初に固定する例は、**同じ表示名を持つNode `n-1` と `n-2`、`n-1 → n-2`のEdge `e-1` と `e-2`、Edge IDの重複を拒否する検査規則**である。初期のAuthoringでは**Markdown表を関係の定義元とし、MermaidをGitHubでも読める導出図**とする。Graphviz等の別表層は後続の対応範囲とし、Role／Identity／向き／多重性／規則のうち実装上意味を持つ部分を、実Readerがどこから取得したか追跡する。架空のTEI構文はここでは定めない。

| 固定する対照 | 独立Expectedと合格の上限 |
| --- | --- |
| 元の定義：Nodeは2件、同じ向きの別ID Edgeは2件 | 生成処理でNode IDを区別し、両Edgeを保持した検査／参照結果を返す。**Node名や端点だけで重複排除しない**。書き出しただけの静的JSONではなく、定義から生成した検証／選択ロジックが動いたことを確認する |
| `e-2`の向きだけを逆にする | 実生成コードの該当判断・実Plugin出力が変わり、正方向のEdgeは1件、逆方向は1件になる。外部の固定Go処理へ全部委譲していれば不合格 |
| `e-2`のIDを`e-1`と重複させる | Reader／Validator／生成された規則の担当境界を明示したうえで、実行前または実検査時に不適格を報告する。誤ったGraphを「Edge 1件」として黙って正規化しない |
| Mermaidの配置や文書説明だけを変える | Source byte／表示の変更は残す。意味上のGraphと生成すべきロジックが同じなら、その範囲を不要に再Buildしない。ただしCache／Evidenceの再利用資格は別に判定する |
| Stage1が生成したPluginを次Stageの実Buildへ渡す | Stage1実Process ID／Input closure／選択Builder／生成Artifact／Stage2での実Plugin Callを照合する。**Seedが二回Buildしただけでは達成としない** |

**限定的なGraph／Rule→Go生成実証**：[reference/tei-md-graph-reader](../../reference/tei-md-graph-reader/README.md) では、6入力のMarkdown表からNode/Edgeと明示したRule候補を読み、元byte位置と重複IDを検査する。生成器はRuleの「起点／終点」を実Goの条件式へ展開し、**別Processでコンパイル・実行した結果**を独立Expected候補と比較する。Edge反転とRule条件変更は生成Goが変わり、MermaidのLR/TB変更だけなら生成Goが一致する。ただし、これは**製品外のReferenceコード生成**であって、受理済みTEI Meaningに基づく製品Generator、選択済みPlugin/Providerの実Call、通常CLI Consumerを証明しない。正式`*.tei.md` Reader・Language Plugin/Role/Owner意味受理は未接続で、現行`TEI_MD_PARSER_UNBOUND`を維持する。

**同じ生成成果物への実行時要求（限定検証）**：前記Referenceの生成Goに、従来の全件JSONに加え`-node`で一Nodeを選ぶ入力を追加した。有効な各Fixtureを一度だけコンパイルし、**同一BinaryのSHA-256を固定したまま**全件・二Node・未知Node・明示空ID二方式・余分な引数・候補指紋の単独照会・Nodeと指紋の同時要求拒否の9通りを実行する。選択Nodeの結果は独立Expectedと照合し、未知Node、明示空ID、余分な引数は非ゼロ終了で拒否する。**Node指定なしの全件取得**と、空文字列を指定した要求の区別を保つ。これはS1の**実行時入力に応じた生成処理の局所証拠**であり、製品`tei` CLIが実Providerを選択・呼出したS2の証拠、Stage1によるStage2再BuildのS3の証拠には加算しない。次のP0は既存#4495/#1251/#951の**同じ選択Providerを実CLIと別Consumerで呼ぶ経路**であり、新しいRunnerや独立Hostは設けない。

**生成Artifactと候補構造の対応を確認する補助証拠**：Graph／Rule限定Referenceに、Node ID・Edge ID／向き・Rule条件だけを正規化した`CandidateShapeFingerprint`を追加する。生成Binaryの`-candidate-shape`から同じ値を取得し、基準とMermaid配置変更は一致、Edge向きとRule条件の変更では異なることを回帰で確認する。**Raw Source SHA-256／候補構造指紋／実Binary SHA-256は別の指標**である。いずれの一致も受理済みMeaning、現在のProvider/Grant、実Build入力閉包・Owner受理の代用品にはしない。Referenceの実行時要求を製品`tei` CLIの新たな公開コマンドとして追加するものでもない。実ProviderやStage2のP0は維持する。

**指紋の自己申告を信用しない反例**：生成GoのEdge判定だけを事後差替えしても、Binary内に埋めた`-candidate-shape`の報告値は元のままにできる。両Binaryの実出力と実byte SHA-256は別であることをGo負例で検査する。S1のReference構造指紋は、**実Builderが消費した生成Source memberのbyte／外部で確認した実Artifact／実Consumerが実行した処理**を代替しない。B1〜B3では#4583/#2831の既存Build Intent・ArtifactSet・同一Attempt証拠で選択と実現の差を照合し、自己申告と一致しただけのStage2再入をPASSとしない。

**固定した入力と期待結果**：[testdata/tei-md-graph](../../testdata/tei-md-graph/README.md)に、基準、`e-2`反転、Mermaidのレイアウト変更、Edge ID重複、表と導出図の不一致に、規則の起点→終点切替を加えた**6例**を通常の`*.tei.md`で置き、Graph出辺と生成コードの実行結果の期待値を別の`independent-expected.candidate.json`に記録した。これらは**既存#1180/#2514に採用を引き渡す候補**であり、`*.tei.md`を実Loaderが読めるという意味ではない。管理Source Rootの未対応Guard `TEI_MD_PARSER_UNBOUND`はReaderが正式に接続されるまで維持する。最初の技術課題は、選択した表・Fenceの元byte位置を保存し、Role／意味IDに結び付けること。MermaidのN1/N2やEdgeラベルを実Relation IDへ黙って昇格させない。

**実装担当へ渡す受入票**：最初に「Edge同一性の保持」「重複IDの拒否」「方向別の抽出」など対象の機能上の義務を固定する。義務ごとに、(a) TEIのどのSource/Ruleが実装を規定したか、(b) 生成したGoのどこが挙動を担うか、(c) 外部Goが担当する境界、(d) 実Builderが読み込んだByteとArtifact、(e) 選択Pluginの実Callと独立Expected、(f) 定義変更前後の手修正・重複記述・再検証作業を記録する。**未観測の項目は未達とし、比率の分母を試験後に縮めない**。

最初のプルリクエストで`*.tei.md`が正式にLoadできなければ、結果をunsupportedと明示し、#2621/#4359の**既存Reader/Source索引を通る最小対応**だけを追加する。旧`*.tei`や手動変換を使うReference経路は分けて記録する。文法全体、Studio GUI、汎用Graph Compiler、DB Provider、独立Hostを初回の完成条件に加えない。


### 最新の実コードを踏まえたGraph定義の製品接続条件（2026年11月15日）

**検証済みの範囲を更新する**：[main #4709](https://github.com/hat47x/tei-platform/pull/4709)のGraph/Rule固定6入力、[main #4722](https://github.com/hat47x/tei-platform/pull/4722)の元byte位置付きReaderに加え、[main #4729](https://github.com/hat47x/tei-platform/pull/4729)で**明示的なMarkdown規則表→実Go条件分岐生成→別Go Process実行→独立Expected候補との比較**が成立した。Edgeの方向、Ruleの「起点／終点」切替では生成Goの処理と結果が変わり、Mermaidの配置変更では意味上の生成Goが変わらない。これはReferenceの実成果だが、規則表の日本語見出しは正式な日本語DSLの採用ではない。製品の`internal/workspace/loader.go`は引き続き未選択`*.tei.md`を`TEI_MD_PARSER_UNBOUND`で拒否しており、**製品Meaning/Target Generator/Plugin/通常CLI/Studioへの接続は別作業**である。

<!-- tei-product-generated-rule-ci-admission-20261115 -->
**限定実装の到達点とCIの現在性（2026-11-15）**：[main #4729](https://github.com/hat47x/tei-platform/pull/4729)のGraph/Rule→実Go分岐生成→別Process実行と、[main #4733](https://github.com/hat47x/tei-platform/pull/4733)の同一生成Binaryへ実行時Nodeを渡す試験は、正式`*.tei.md` Readerや製品Pluginの受理とは別の**限定Reference成果**である。#4733当時の全体CI失敗について、SQLの古いOpaqueテストは[main #4741](https://github.com/hat47x/tei-platform/pull/4741)、Graph生成試験のgofmtは[main #4742](https://github.com/hat47x/tei-platform/pull/4742)で修正済みとなった。#4742では既存Source IndexのRaw-byte/Spanを照合するReference試験も追加されているが、`workspace.Load`へ新形式を正式に接続した証拠ではない。

残る`SeverityUnsupported`の古い語彙TODOは、[旧PR #4449](https://github.com/hat47x/tei-platform/pull/4449)の広範なPostgreSQL実行試験・Query語彙テスト計画と重なる。このため最新mainで未変更の**当該一行だけ**を暫定的に解消する[後継PR #4743](https://github.com/hat47x/tei-platform/pull/4743)を作成し、競合した[#4736](https://github.com/hat47x/tei-platform/pull/4736)は二重マージ防止のため未マージで閉じた。#4449の本来の実PG/語彙検収は省略せず、後続のmain追随時に同じ行を再適用しない。**#4743のexact HEAD CIとmain統合は現時点で未確認**。旧#4736ではRepository Healthが成功してもWindows WebView2 smokeが接続拒否・タイムアウトで失敗しており、これは別のDesktop実機資格として[#4583](https://github.com/hat47x/tei-platform/issues/4583)・[#4635](https://github.com/hat47x/tei-platform/issues/4635)へ引き渡す。製品Reader／Meaning／実Provider／Stage1→Stage2の未接続は以上のCI修復とは独立して残る。

#### 定義変更と実行時入力を別々に検証する（#4733反映）

[main統合済み#4733](https://github.com/hat47x/tei-platform/pull/4733)は、#4729のGraph/Rule→実Go生成をもう一段進めた。**一度だけビルドした同じ生成Binary**に実行時Node IDを渡し、Nodeごとの出辺/入辺結果を独立Expected候補で確認する。引数なしは全Nodeの出力、既知NodeはそのNodeの結果、未知Node・明示空ID・余分な位置引数は非ゼロ終了として区別し、試行前後のBinary SHA-256が変わらないことを検査している。これは「生成したコードに実行時入力が効く」限定的な実証であり、製品Function InvocationやPlugin実Call、業務上の認可を示すものではない。

ここで**Sourceを変更すること**と**生成済みFunctionへ入力を渡すこと**を混同しない。前者はSource/Meaning/Generator/Artifactの再検証資格、後者は実Invokeの型・値・結果・失敗の資格として扱う。最初の実装PRでは既存Fixtureに次の直交比較を追加できるよう、検証票を残す。

| 一つだけ変更するもの | 生成と実行の期待 | 不適格な省略・推論 |
| --- | --- | --- |
| **GraphのEdge方向／Rule条件** | 受理済みMeaningの変更なら生成処理に反映し、再生成Goと独立Expectedの差を確認 | 以前のBinaryで結果が出たから新Sourceにも適合すると推測 |
| **実行時Node ID** | 同一Binary・同一Source/Meaningのまま、選択したNodeの結果を返す | Node選択のたびにコード再生成・再ビルドする、入力を固定JSONへ置き換える |
| **Node IDの省略／明示空／未知／既知だがEdge 0本** | 省略＝契約で定めた全件、明示空・未知＝失敗、既知で該当Edge 0本＝正常空配列を別扱い | 入力誤りを正常0件に変換、無指定を空文字指定として扱う |
| **Mermaidの表示配置のみ** | 元Sourceの変更は記録しつつ、意味不変なら生成Goは同byteであることを比較 | Source再確認をせず旧Evidenceを無条件再利用、図のラベルを意味IDへ昇格 |
| **選択Plugin/Target/Host/Consumer** | 同じRuleでも選択・同梱・Load・Invoke・現在Grant・実Consumerを別証拠で検査 | 参照BinaryのSHA-256不変や出力一致を実Providerの呼出し証拠にする |

**Readerの負例もGraphとRuleを一緒に扱う。** [main統合 #4725](https://github.com/hat47x/tei-platform/pull/4725)は、限定ReferenceのGraph表だけでなく、**Rule表の抽出にも共通のFence判定を利用する修正と負例**を提案している。バッククォート/チルダ、長いFence、インデントコードの例示表を実Ruleとして取得しないこと、未閉鎖Fenceを成功にしないことを、両方の表で同じ元byteを使って検証する。この修正は限定ReferenceのGo試験としてmainに統合済みだが、正式製品Markdown Readerの合格とは扱わない。将来の製品Frontend接続時も、この区別を保持して既存#2621/#988/#4704の受入へ引き渡す。

**今後の引渡し**：#4733によって「Caller入力が生成Goの実際の判断を変える」部分はReferenceで確認できた。したがって次の製品コードPRではこのReferenceを再発明せず、**明示的に選択した`*.tei.md`の実Read→元byte/span→既存Source Index/Role候補**の断点を先に解消する。正式Meaning・Operator/Rule適合、Product GeneratorへのLowering、Plugin A/Bの実選択・呼出しは別PR・別Attemptで証拠を取る。試験IDと採否は#1180/#2514のIndependent Expectedに従い、第二のExpected正本を作らない。

#### 既存Source Indexとの実照合後に残るWorkspace選択境界（main #4742）

[main #4742](https://github.com/hat47x/tei-platform/pull/4742)で新設された`reference/tei-md-graph-reader/sourceindex_bridge_test.go`は、固定6入力のうち**有効な5入力**について、`ParseRuleCandidate`のNode/Edge/Rule原byte位置を既存`sourceindex.NewArtifact`／`RangeFromOffsets`／`Index.Validate`に照合する。原文に説明行を追記すれば、意味上のGraphが同じでも元のArtifact fingerprintが変わり、**古いIndexの`Validate`は失敗する**。これは、位置換算やbyte整合性のために第二のSource Indexを増設する必要がないことを示す局所的な証拠である。

ただし、この試験は**テストが指定した`semantic/graph.tei.md`のbyteから`Index{Artifacts: ...}`を直接作る**もので、実`Workspace.Load`がManifest／選択Frontendに従って読んだ証拠ではない。現行のIndexへ**意味Subjectの`References`を登録した試験でもない**。Graphの仮Node IDを正式Subjectへ読み替えたり、`Index.Validate`が通ったことをParser選択・Owner Acceptance・Generator実消費へ昇格させたりしない。

次の製品S0を、既存のReader/Source Indexを拡張する前に、**選択と実Readの一致**に絞る。

| 一つの検証条件 | 合格するには | 防ぐ誤判定 |
| --- | --- | --- |
| **指定したWorkspaceとSource** | 実Manifest/Root/Source pathと選択済みLanguage/Frontend/版を一意に固定し、実Loaderの**同じAttempt**が読んだbyteをSource IndexのArtifactへ渡す | テスト内で直読みした別コピー、または拡張子`.md`だけによる自動採用 |
| **指定していない`*.tei.md`** | 対応Frontend未選択、未対応表、契約版不一致は`TEI_MD_PARSER_UNBOUND`等の適切な明示拒否を維持。既存`*.tei`の通常Loadを壊さない | Guardを全解除してファイルを無視、管理一覧だけで`ReadByLoader=true`とする |
| **定義領域と表示・説明領域** | Node/Edge/Ruleの選択表だけを原byte/spanで読取り、Mermaid/引用/Fenceの例示部分を定義へ混入させない。#4725の負例は未統合として後続Ownerで適格化 | 見た目の図を意味正本にする、例示したRule条件で生成Goが変わる |
| **読取り後のSource変更** | 現在のLoadが消費したSnapshotとSource Index fingerprintを結び、再検証時の不一致を明示。A→B→Aでも旧Owner受理/Grant/Bundle受領は自動復活させない | ディスク再読で別byteを黙って混ぜる、同fingerprintだけで以前の実証資格を継承 |
| **Candidateと意味Subject** | Parserの候補ID/Span、Source Index Artifact、意味Role/Qualified Subject Reference、Owner受理を段階別に提示。未受理の段階では正式Referenceを捏造しない | Indexの物理`format=md`をTEIの選択Profileや意味Roleとして利用 |
| **Build/Consumerへの引渡し** | 正式Meaningが受理された後、Generator選択・実Builder消費byte・実Artifact・Plugin/Consumerを独立に照合 | `Index.Validate`成功や参照`go build`だけで製品Plugin実Callを合格とする |

**実装担当への一番小さい変更依頼**：#2621/#988と[Reader移行Draft #4704](https://github.com/hat47x/tei-platform/pull/4704)の範囲で、**明示選択した一つの`*.tei.md`を`Workspace.Load`の実Source Snapshotへつなぎ、既存Source IndexのArtifact/byte/spanと同じ試行で照合する正例と、未選択・変更後byte・未対応Syntaxの負例**を先に作る。Meaning Subject Referenceの登録、正式Go Target生成、PluginのLoad/InvokeはS0の検査項目とは分け、必要な意味契約をOwnerが受理した段階で実装する。既存[Plugin使用例Draft #4735](https://github.com/hat47x/tei-platform/pull/4735)で`-node`等のReference実行例を案内する場合も、製品`tei` CLIの実コマンドとして誤表示しない。

ここで確認するのは**現行の選択と消費の断点**であり、Graph対象の新Registry、別Parser/Runtime、第二Acceptance、全Markdown構文対応、Studio依存のHard Gateを新設する理由にはならない。独立Expectedの採否は既存#1180/#2514に委ねる。

<!-- tei-declarative-implemented-rule-to-product-selection-20261115 -->
#### 実装済みの低ノイズRule候補を、製品Language選択から切り離して評価する

[main統合済み #4405](https://github.com/hat47x/tei-platform/pull/4405)で、`rule <id>: <attribute> is positive`／`is unique per <relation>` の**低ノイズ構文候補**が、`internal/syntax/rule_authoring.go`の`ParseRuleAuthoringFile`、`rule_authoring_validation.go`の`RuleNamespace`、`rule_authoring_anchor.go`の`ParseRuleAuthoringAnchored`として実装された。候補は既存`semantic.Validation`へ写され、既存Validator/Evaluatorを使う独立Expectedとの局所比較・元byte span・明示Formatterの意味不変検査がある。旧`validation`のParserは変更されず、`rule`の正式Grammar採用、`Workspace.Load`での新Frontend選択、OwnerのMeaning受理、実Generator／Host実行は**行われていない**。旧HEAD #4405のRepository Health／Cacheはfailure、focused Go等の成功を最新mainの全体PASSに読み替えない。

ここで混同しやすい**別の二つの入口**を、同じタスクで実装済みにしない。

| 入り口 | 現在確かめられること | 次の実装責務 |
| --- | --- | --- |
| `*.tei`向け低ノイズ`rule`候補 | 直接呼ばれた専用候補Parser→従来`semantic.Validation`、単一ParseからのAnchor、局所Evaluator／Formatter比較 | 正式構文の採否は#4359。Language／`formatVersion`と従来Loaderへの参加を決めるまで、通常Sourceで自動受理しない |
| `*.tei.md`のGraph／Rule／Window候補 | Manifest／Pinned Root／Source IndexのS0-Pと限定Graph/Rule生成、Window5原本のbyte追跡 | **最初の製品S0-AはGraph／Rule一Source**。実`Workspace.Load`で受理済みLanguage Selectionに基づくReaderを呼び、同じbyte／Span／Indexを照合する |
| Language Plugin選択の限定Reference | `reference/dsl-language-plugin/selection.go`の`ParseSelected`は、**既に受理されたexact Selection**とProvider候補を比較できる | 製品Manifest／既存Plugin Contract／LockからそのSelectionを得るOwner接続は未実装。Referenceの`Resolve`を製品Registryだと仮定しない |

**最初のコードPRをさらに小さく分割する**（[自己記述§47.5](../design/ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)、#988/#1018/#2621、第二Resolverは作らない）。

- **A0：実選択の供給元を確定**。Workspace／Source Root／pathに対し、既存の受理済みLanguage Contract・Provider実装・版／Lockを現在の製品APIから取得できるかを調べる。供給不能・競合・版不一致なら`TEI_MD_PARSER_UNBOUND`または明示unsupportedで停止する。新しいManifest fieldや暫定auto-selectは採用しない。
- **A1：同一Load内のSource実消費**。A0が成立した**選択済みGraph／Rule一原本**のみ、既存`scanManagedFiles`／Pinned Root／`ReadByLoader`の区別を保ち、製品Readerへ元byteを渡す。テスト専用`readPinnedWorkspaceFile`の呼出しはS0-Pに留まる。
- **A2：意味候補と元位置**。同じLoadのSource Index／byte rangeと、Parserが実際に読んだNode／Edge／Ruleの候補を照合する。Source Indexの`Format=md`、`RoleDefinition`等を選択済みLanguageや業務Meaning受理へ昇格させない。
- **A3：負例と旧版回帰**。未選択／同名の管理外Draft／Provider版不一致／Fence中の例示／重複ID／旧Index／A→B→Aを個別拒否し、既存`*.tei`の正常Loadを回帰する。失敗時に部分的な`World`や実行可能Subjectを返さない。

**手戻りを防ぐ優先順**：先にA0の**実製品APIの不足**を明らかにし、未解決ならそこで停止してOwnerへ戻す。A0を突破できないことを理由にReference Parserを`Workspace.Load`へ直接結線しない。A1〜A3の後に、同じ独立Expectedを使って正式Meaning→一Ruleの実Go分岐→実Builder Source member／Artifact／通常CLI・Plugin／別Consumerへ進む（#4544/#2595/#1251）。[main #4822](https://github.com/hat47x/tei-platform/pull/4822)のNULL・0・行不在で同じ集計値になる実DB負例は、後続のQuery／意味同値検収へ回し、A0〜A3の新Hard Gateにはしない。

**評価の軸**：読者の「対象→条件→判断→結果」という認知順、記述量・重複、既存Validationとの意味同値、元位置・診断の品質、製品Loaderでの実消費と、変更時に手書き処理が減った割合を**別々に記録**する。構文が短いだけで自己構築済みとは数えない。全体P0のWindows Stage1→Stage2と通常`tei`実Provider／独立Consumerを独立して継続する。

<!-- tei-a0-reference-negative-to-product-positive-20261115 -->
**A0の実装で更新する正負テスト（既存Reference回帰を継承）**：現行mainの[Language境界Goテスト](../../reference/dsl-language-plugin/workspace_admission_boundary_test.go)は、`baseline.tei.md`を実Manifestの管理Rootに置き、合成Pinで`ParseSelected`がGraph/RuleのNode二件・別ID Edge二件・元byte spanを読めても、同じ原文に対する製品`Workspace.Load`は`TEI_MD_PARSER_UNBOUND`で拒否することを検証している。Pin欠落、実装指紋不一致、同一Pinの重複ProviderではReferenceのParser呼出し自体を拒否する。このテストの存在を正式A0実装やProduct Reader成功とみなさない。原PR #4857はclosed/unmergedであり、今回確認したのは**mainに実ファイルが存在すること**である。

A0を製品に接続する変更では、既存の拒否証拠を消すのではなく、**選択なしのWorkspaceと、受理済み選択ありのWorkspaceを別入力として**次の結果を確かめる。

| 既存証拠から変える一因子 | 製品に必要な結果 | 誤ったPASSの例 |
| --- | --- | --- |
| 同じ管理Source、正式Language選択なし | `Workspace.Load`は従来の未束縛拒否を維持する | Referenceの合成Pinを製品選択として流用 |
| 同じSource、実Workspaceに対応する受理済みLanguage/実装Pinあり | **実Loader**から選ばれたReaderが同じLoadの元byteを消費し、Source位置・Indexへ接続する | TestがReaderを直呼びし、Productの`ReadByLoader`を推定する |
| 同じSource、実装欠落・版/指紋不一致・複数Provider | 解析前に拒否し、部分的に受理済みのWorldや実行資格を返さない | 先に見つかったProviderや旧選択へfallback |
| Source/Manifest・Selection・Load Attemptの一つだけ変更 | 古い候補とIndex/Reader消費の対応を再確認し、旧AcceptanceやGrantを自動再利用しない | 元byte一致やReference指紋一致だけで再受理 |
| 管理外Draftと従来`*.tei`を同時に置く | Draftの同名原文を読まず、従来`*.tei`の正常Loadを維持する | ファイル名・拡張子による暗黙のFrontend選択 |

**移行時のテスト方針**：最初のReader実装で従来の一律拒否が条件付き拒否へ変わるなら、既存の負例は「未選択の世界」に残し、正式に選択された世界の肯定例を**同じFixtureを共有する別テスト**で追加する。拒否テストの一括削除や`TEI_MD_PARSER_UNBOUND`の全面解除はしない。A0のSelection供給は#1018/#988、A1〜A3の実読取・Source Indexは#2621/#1017/#1180/#2514へ引き渡す。新Registry、暫定Manifest拡張、第二Reader、独立Expected正本を作らず、Studio Stage1→Stage2と通常CLI実Providerの独立P0を維持する。

<!-- tei-builtin-rule-md-product-reader-20261115 -->
**製品コードへの初接続を確認する検査**：mainの[内蔵Rule Markdown実装](../../internal/workspace/authoring_language.go)／[Rule Reader](../../internal/workspace/rule_authoring_markdown_loader.go)と実Fixture `internal/workspace/testdata/tei-md-rule`を検収基線とする。選択された`tei.rule-authoring/v0`は、Markdown内の二つの`tei`定義領域を既存`semantic.Validation`と同じSubjectへ写し、`AuthoringSources`に実Readと定義件数を記録する。定義外の説明・`text`例示は読み取らず、元byteをSource Indexに保持する。追加する[製品回帰](../../internal/workspace/tei_md_rule_product_read_test.go)では、Product `Load`の実読取・二Rule IDの正確なbyte Range・同じLoadからのIndex、説明だけの変更時の旧Fingerprint失効、未選択と版違いの拒否を一緒に照合する。**テストのコード追加はCI成功と同義ではない**ため、exact HEAD結果を別途記録する。

次の一歩は新しい普遍Parserではなく、**この実装を一つの契約として確実に検収したうえで**、(a)外部Language Pluginの実Resolver/Lockとの連携、(b)Graph/Windowの異なるSource様式、(c)受理済みMeaningから製品Generatorによる一Rule実Go生成と実Builder／通常CLI Consumerを、それぞれ別のOwnerへ渡すこと。Readerが型付きValidationを作ったことは、業務Ownerの意味受理・Plugin選択・実Provider/Effect・Windows Stage2を意味しない。新Registry／Host／Expected正本は増設しない。

<!-- tei-rule-native-markdown-semantic-equivalence-20261115 -->
**次のS0検収は異なる書式から同じ意味へ至ることを確かめる**：内蔵Rule Markdownの製品入場後は、「二件読めた」だけではなく、**従来の`validation`と`rule`の表記が、同じ`semantic.Validation`へ収束するか**を実`Workspace.Load`同士で比較する。[Go回帰](../../internal/workspace/tei_md_rule_surface_equivalence_test.go)は既存の`reference/workspace/semantic/sales/validation.tei`と内蔵Ruleの実Fixtureを用い、二つのRule IDについてSymbol・Kind・Attribute・Relationを含む型付き値を独立に照合する。同じMeaningでも表層ごとのSource Artifact／元byte範囲／Language選択記録は異なり、同じWorkspaceで両表層から同一IDを二重宣言すれば、読み順で一方を採らず拒否する。

ここから次の二つを切り分ける。(1) **構文上の意味同値**：正の数量検査と受注明細番号の一意性が、旧・新表層のどちらでも同じ型付きValidationになること。(2) **製品としての利用資格**：言語Contractと具体的なSourceが選択され、既存のSource Indexで原文位置を追跡し、Ownerが意味を受理した後に生成器が実際に利用すること。前者のテスト成功だけで後者をPASSにしない。これを#4359のS0比較と#2621/#988のReader受入に戻し、次は#4544/#2595で一Ruleの実Go分岐を生成・実Builderへ渡す。新しいRule Engine、第二Parser／Expected正本を作らず、全体P0の実Provider二ConsumerとStudio Stage1→Stage2を維持する。

#### S0予備検証と正式Reader接続を分け、生成物の実消費までつなぐ（2026年11月15日）

**S0-PのGraph／Window予備検査はmain統合済み。内蔵`tei.rule-authoring/v0`については、選択された`*.tei.md`の製品Loadと`semantic.Validation`／Source anchorへの接続がmainに実装された（2026年11月15日更新）。** ただし、Graph／Windowの製品Reader、外部Language Pluginの実選抜／Lock、Ownerによる意味受理、生成Pluginの実使用は未成立。 [#4751](https://github.com/hat47x/tei-platform/pull/4751)では、実Workspace Manifestの`semantic` Root、未宣言`drafts`の除外、同一Pinned Rootからの元byte、固定されたGraph/Rule候補、既存Source IndexのFingerprint/Span、原文変更時の旧Index拒否を**テスト専用の事前検査**として接続した。旧[#4745](https://github.com/hat47x/tei-platform/pull/4745)は未マージでクローズされ、同内容の実Go試験が#4751からmainへ入った。現行の`Workspace.Load`は未選択`*.tei.md`に`TEI_MD_PARSER_UNBOUND`を返し続ける。これは意図した安全な境界であって、予備検査の成功により製品Readerを解放する理由にはならない。

**CIの資格も分ける**：#4751のexact HEAD（`699ac0bd025a953d2a7a0e7dea2dbe9f1c19a67f`）では複数のWorkflowが成功した一方、[Repository Health](https://github.com/hat47x/tei-platform/actions/runs/43644037417)は失敗している。旧#4745の[Repository Healthログ](https://github.com/hat47x/tei-platform/actions/runs/43641434912)はSQLite Window試験の`gofmt`ずれと`SeverityUnsupported`の旧TODOを示したが、**旧HEADの原因や成功を、新HEADや最新mainに無条件転用しない**。最新main全体の状態は同一HEADのCIで別途判定し、修正済み・全体合格の主張を先取りしない。

**次の実装は三つの資格を別々に満たす**。初回にすべてを一PRへ詰め込まない。

| 段階 | 入力と実際の証拠 | 現在の資格と次の断点 |
| --- | --- | --- |
| **S0-P：Pinned Source予備検査** | Manifest `semantic` Root、非宣言`drafts`除外、Pinned Root読取り、候補Graph/Rule、Source Index fingerprint/Span、変更後の旧Index拒否 | [main #4751](https://github.com/hat47x/tei-platform/pull/4751)へテストが統合済み。ただし読取りはテスト側で明示しており、製品LoaderのFrontend選択は未接続 |
| **S0-A：製品Reader Admission** | 実`Workspace.Load`の同一Attemptに対して、型付きで選択されたFrontend/版、Manifest Root、実際にReaderが消費したbyte/Span、Source Index Artifact/必要なRole候補を対応付ける | **内蔵Rule Markdownの限定経路はコード接続済み**。同一Loader／Indexと正負回帰を検収し、Graph／Window・外部Pluginは別途実装。未選択は従来どおり拒否。管理済み≠Read済み≠Parse済み≠Owner受理 |
| **S1/S2：意味Go・実Plugin** | Owner受理済みMeaning/Rule→Target Generatorの条件ロジック→生成Go→実Builder/Target→選択Pluginの組込み/Call→実Consumer | 限定ReferenceのGo条件分岐と同一Binary実入力は成功済みだが、製品Targetでの実消費は未成立 |
| **Artifactの再入力／Studio** | 生成Source Memberが次Buildの**実入力byte**になり、生成JS等がWails埋込みへ渡る場合も、Builder/ArtifactSetの同じAttemptと子Process/受領先を照合 | [Generator契約Draft #4748](https://github.com/hat47x/tei-platform/pull/4748)の未実証Fit/Gapに接続。生成物の存在だけで次工程が消費したとみなさない |

**次に変更すべき製品境界**：現在の`internal/workspace/loader.go`は`scanManagedFiles`の後、Manifestの`languages:`から内蔵Contract／版を`resolveAuthoringLanguages`で解決し、`admitTEIMarkdownSources`でSourceごとにReaderを選ぶ。選択済みのRule Markdownは従来の`semantic` Loaderの後に`loadAuthoringSources`で同じLoadの元byteを読む。これは**内蔵`tei.rule-authoring/v0`に限った製品接続**であり、GraphのMarkdown表、Window、外部Language Pluginを読める証拠ではない。次は選択あり／なしの両世界でLoader・`AuthoringSources`・Source Index／旧`.tei`を独立照合する。未選択・契約版不一致・実装曖昧性はfail-closedを維持し、`sourceindex.Artifact.Format="md"`だけをFrontend選択の根拠にしない。

**内蔵Ruleの後続実証（2026年11月15日）**：mainで、製品Reader／Source Index／InspectorのGo回帰（#4890/#4891）に加え、旧`.tei`とMarkdownから同じ型付き`semantic.Validation`を作る試験（b89209a2）、選択済みRuleを既存Evaluatorへ渡して入力Read Footprintを観測する試験（#4901）が収録された。数量の正値判定、Relation Keyの重複・完全な空集合と取得範囲不明を区別できるが、Evaluator入力はテスト用Sourceであり、実DB/実Providerの全件取得や時点・権限の受理ではない。次は固定Rule`sales.order_line.quantity_positive`から実Go比較・分岐を生成し、Rule条件変更で生成物と実Consumer結果が変わることを独立Expectedで検査する（#1121/#4544）。実Builder/Artifactへの収録、通常CLIのProvider actual call（#1251/#4495）、Studio Stage1→Stage2 Windows（#2831/#4583）は、それぞれ後続・並行の別Gateを保つ。詳細は[Rule第一縦断§12](../design/PLUGIN_RULE_PRODUCT_VERTICAL_SLICE_2026-11.md)と[横断Plugin§24.5〜24.6](../design/PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md)。

**最小の完了証拠**：同じ一回の`Workspace.Load`で、(a)選択に使ったManifest/Frontend/Source Revision、(b)Root内にある実Sourceの読み取りbyte、(c)Node/Edge/Rule候補の元byte Span、(d)既存Source IndexのArtifact/Fingerprint、(e)`ReadByLoader`の真偽と解析・意味受理の**未達状態**を矛盾なく照合する。Source Indexの`RoleDefinition`等は参照種別であり、業務意味RoleやOwner Acceptanceとは別である。テストコードが`readPinnedWorkspaceFile`を直接呼んだだけの証拠はS0-Pのままとし、**実Loader内のReader消費を確認して初めてS0-A**と呼ぶ。

**S0-Aの最小負例集合**：未選択Frontend、非宣言Rootの同名ファイル、通常`README.md`の説明用表、対応外Profile/版、Fence/引用内の見かけ上の定義、解析対象の重複ID、Load後の元byte改版を別々に拒否または非対象として示す。旧`*.tei`の通常Loadは回帰成功を維持する。型付きFrontendや意味Roleを選ぶ権限がない場合に候補の表示だけはできても、**正式Subject登録・Go生成・選択Plugin/実Provider起動は不可**とする。Source A→B→Aで同byteに戻っても、以前のOwner Acceptance・Current Grant・実Consumer受領は自動回復しない。

S0-Aで最初に提示すべき**独立した正負**は、同一Source Snapshotからの1選択・1Reader・1Artifact整合、未選択Frontend・Root外ファイル・未対応構文・旧`*.tei`への退行・原文変更後の旧証拠再利用・暗黙のReader fallback拒否である。Rootを固定して読んだことは、複数Sourceの**原子的Snapshotを保証するわけではない**ため、取得したbyteと検証に使った版が一致する範囲だけをClaimとする。Owner意味受理前のGraph/Rule候補IDはSource Indexの正式Subjectへ勝手に登録しない。

この順序を守れば、Pluginの説明文書や使用例の資格も整理できる。[main統合済みのPlugin公開文書計画 #4750](https://github.com/hat47x/tei-platform/pull/4750)の`-node`例は、現状では**限定Reference Binary向け**であり、通常の`tei` CLIで呼べる公開Operationではない。将来A PluginのManual/ContractをB Pluginが参照する場合、Source/Doc/Contract/Plugin版の一致、現在のDisclosure、実選択・実Builder消費・実Invocationを分けて検収する。公開文書を読めることを、Pluginが同梱されたことや実行できる資格とみなさない。

**P0/P1の扱い**：本件は#2621/#988/#4704の正式Reader・Source Index接合を進めるP1の小縦断として扱う。全体P0のStudio Stage1→Stage2実Windows再構築（#2831/#4583）と、通常`tei`から実Provider→独立二Consumer（#2586/#1251）を遅らせる新Hard Gateを作らない。S0-Pの参考実装をそのまま製品の第二Loader/第二Registryに発展させず、S0-Aの差だけ既存のWorkspace契約へ移す。GC/GQの既存Expected・受入Ownerを再利用する。

#### 原文・候補構造・生成物の三つの同一性を別々に扱う（main #4758）

[main統合済みのGo実装 #4758](https://github.com/hat47x/tei-platform/pull/4758)では、Node ID集合、Edge ID・起点・終点、明示Rule ID・`from/to`条件を正規化し、`tei-md-graph-reference/shape-v1`という**限定Reference専用のドメイン**に結び付けたSHA-256を計算する`CandidateShapeFingerprint`が加わった。生成Goにはその候補指紋が埋め込まれ、同じ参照Binaryに`-candidate-shape`を渡して照会できる。`-node`との同時指定は拒否し、既存のNode別実行は維持する。#4758のexact HEADはRepository Healthと増分検証に成功した。これを一般的なTEI Meaning Fingerprintや製品CLIの新Commandに昇格させない。

| 同一性と目的 | 何を変えると変化するか | 一致しても成立しない資格 |
| --- | --- | --- |
| **Raw Source／Source Index Artifact** | 原文のMarkdown説明、BOM・改行・図の配置を含む**byte差** | パース成功、業務意味同値、Owner Acceptance |
| **Reference Candidate Shape** | この限定実装ではNode ID、Edge ID／向き、Rule ID／条件 | Sourceの現在性、任意Domainの意味同値、Selected Language/Operator/Provider、正式Model Revision |
| **Generated Go／Binary byte** | 実Generator/Toolchain/Build入力・出力に依存する実ファイルの変化 | どの実Source/意味受理から作成したか、次Builderや実Consumerが選択・使用したか |
| **現在の意味・実行資格** | Subject/受理済みRole、Source/World/Model・Artifact Revision、Plugin/Host Grant、Disclosure、実Target/Consumerの状態 | 一つのハッシュには包含されない。既存Authority・Build Evidenceで個別に確認 |

**限定的な構造同値と元Source失効の衝突を実負例にする**：[main統合済みのGo回帰 #4759](https://github.com/hat47x/tei-platform/pull/4759)は既存`sourceindex_bridge_test.go`へ、(a)基準と`layout-only`、(b)原文説明だけの追記は**Source fingerprintが変わるがCandidate Shapeは同じ**、(c)Edge方向／Rule条件の変更は両方変わる、を追加する。さらに各入力から生成する**Goソースの実byte**も比較し、候補構造が同じ場合は生成Goが一致し、Edge・Ruleを変えた場合には生成Goも変化することを要求する。いずれも**旧Source Indexは失効**し、新byteで作るIndexだけを検証できることを期待する。この試験はmain統合済みの限定Reference回帰であり、正式Readerや通常CLIの実証ではない。

**差分Buildの判断を二段階に分ける**：最初に実消費Raw Sourceと現在のSelection/Role/Owner資格を検証し、続いて生成器が実際に使う**受理済み意味・Rule・Plugin/Target依存閉包**が不変かを調べる。Referenceの候補構造指紋は、後者の局所的な補助観測にはなり得るが、**それだけでGo再生成のskip、Artifact／Evidenceの再利用、A→B→A後のGrant復活を許可しない**。逆にMarkdownの表示や説明だけが変わったときに、実生成ロジックまで常に作り直す必要があるかは既存[#2600](https://github.com/hat47x/tei-platform/issues/2600)の実消費入力と計測で判定する。

この考え方はGraph以外のDeclarative DSLにも適用する。ただし、業務のラベル、数量・単位、NULLと欠損、時間・対象Scopeなどが**意味に関係するDomain**では、#4758の限定指紋の除外項目を流用しない。既存[#4359](https://github.com/hat47x/tei-platform/issues/4359)／[宣言的構文の統合Draft #4756](https://github.com/hat47x/tei-platform/pull/4756)のOwnerが各表層と受理済み意味を明示し、Markdown/Graphviz/Mermaid/数式/Native SQLの意味上の選択と表示変更を分離する。新しい万能構造Fingerprint、意味正本、再利用判定器をこのために設けない。

**Plugin・Studioへの引渡し**：[mainのPlugin Manual計画 #4750](https://github.com/hat47x/tei-platform/pull/4750)はReference `-node`と製品`tei`のコマンドを別資格とし、[main統合済みのPlugin公開文書A/B Fixture #4757](https://github.com/hat47x/tei-platform/pull/4757)は公開範囲・版・Sourceを検査する限定候補である。文書や構造指紋の一致からPluginの選択・Build同梱・Load・Invoke・現在Disclosureを推測しない。Studio編集では元byte・Source位置の再解決を先に行い、Graph図のレイアウトと実Relation IDを混同しない。次の正式Reader S0-A／Product Generator S1／Plugin S2／Artifact GQ-06／Studio Stage2の**既存段階とP0順**を維持する。

#### 次の製品実装PRで確定すること

1. **明示されたFrontendの選択**：Workspaceの正規化済みSource path、Root、選択済みLanguage/Role Profile、選択Plugin/契約版、実Source byte/Revisionをそろえる。拡張子だけで「実行可能な意味定義」を認定しない。`internal/sourceindex/model.go`の`NewArtifact`は現状`path.Ext`で形式を得るため、`*.tei.md`のArtifact Formatは一般`md`と同じになる。これは**現在のSource Indexの書式分類**であり、選択FrontendやTEI宣言資格の証拠ではない。Formatフィールドを無断で変更して従来Consumerを壊すより、まず選択条件を別の既存契約から明示して負例で比較する。
2. **抽出された候補とMeaningの切断**：Referenceの`Node/Edge/GraphRule`はbyte位置付きの**候補**。Source Indexの`RoleDefinition/RoleAuthoring/RoleImplementation/RoleDerived`はSource参照の役割であり、Graph Ruleの業務的意味Roleや受理済みOperator契約と**同じ型ではない**。SourceIndexにReferenceを登録した事実を、Meaningの受理・生成許可とみなさない。Selection→read→parse candidate→Role/Contract qualification→Owner acceptance→実Generator消費は別資格として表示する。
3. **読取りの正直さ**：`managed`と`ReadByLoader`は既存でも別の属性。`ReadByLoader=true`はSource byteの消費を示すだけで、Parserの成功でもRole承認でもない。元のUTF-8 byte/span、BOM/CRLF、Source Revision、元の表見出し・CellのlocatorをReferenceと製品側で照合する。未対応/未選択の`*.tei.md`は引き続き明示拒否し、従来`*.tei`の正常Loadを破壊しない。
4. **安全なMarkdownの部分解釈**：本文・引用・インデントコード・バッククォート/チルダのFence・長いFence・未閉じFenceを含むとき、見かけ上の「ノードの定義」見出しを意味対象として誤抽出しない。[ReferenceのFence修正Draft #4725](https://github.com/hat47x/tei-platform/pull/4725)の負例を既存Reader Ownerへ引き渡す。既存の限定Readerをそのまま汎用Markdown Parserと宣言しない。読めない場合は`unsupported`または該当診断を返し、推測で行を補完しない。
5. **生成の実消費と意味の説明**：正式Meaningと解決したRuleから出辺/入辺の条件付きGo処理を生成する工程を、局所Reference生成から製品Target Generatorへ段階的に移す。選択したGenerator、Source/Meaning/Artifact Revision、実Builderの消費byte、生成コードの該当条件、独立Expected、実Plugin/Consumerを結ぶ。生成Goの実行結果だけで選択Provider/Plugin利用やOwner受理を証明しない。

| 一つの入力に対する受入の問い | 現在の上限 | 次のOwnerと証拠 |
| --- | --- | --- |
| **Graphの同名Node・多重Edge・Source span** | #4722の限定read-only Reference | #2621/#988/#4704：製品Workspace選択済みSourceの実byte/spanとLoader/Index結果 |
| **Ruleの起点／終点による生成Goの分岐** | #4729の限定Go生成・別Process実行 | #2620/#951：受理済みMeaning/Operatorからの正式Target GeneratorとPlugin実Call |
| **Markdown表示・規則・不正構文の境界** | 6 Fixture＋候補Expected。Fence負例は#4725で限定Referenceに統合済み | #2621/#2514：参照Readerの否定例と製品Reader/SourceIndexの差、未選択拒否 |
| **Selectorの全件適用・Bundle実現** | mainのProgram Execution Model §33〜39およびGQ-06は設計/受入候補 | #4198/#3509/#2600：Route Universeの閉包、Hook Binding、実Bundle Member、実Consumer |
| **AI/Studioの設計対象引渡しと編集** | #4515/#267/#1124の限定選択とProposal/Review契約 | PinとFocusの区別、古い非同期応答の棄却、受理前候補の生成混入拒否 |

**同一Fixtureを利用し、余分な正本を作らない**：基準、`e-2`反転、重複Edge ID、layout-only、表/派生図の不一致に加え、**Graph不変でRule条件だけを起点→終点へ変更する第6例**を、各段の独立Expected候補として使う。資料写真/会議/CSV/既存画面から得るStructure Candidateは[#267](https://github.com/hat47x/tei-platform/issues/267)と[#1124](https://github.com/hat47x/tei-platform/issues/1124)のSource/Revision/Disclosure付きread-only Proposalへ渡し、受理前にGenerator/Selectorへ混入させない。BRMS的な「Ruleの適用条件、競合、判断理由」は[Rule統合Draft #4723](https://github.com/hat47x/tei-platform/pull/4723)と既存#2620/#2514に返し、別BRMS/Graph Runtime/意味正本を作らない。

**作業の順番と優先度**：まず正式Frontend選択とSource byteを`Workspace Loader→既存Source Index`で一つのFixtureへ結び付ける小Go PRを#2621/#988/#4704へ返す。次に#4729で実証済みの限定生成ルールを**受理済みMeaning**から呼ぶ製品Generatorとし、選択Pluginの実Call、必要ならGQ-06のGraph Selector→実Artifact→二Consumerへ進む。全体P0のStudio実Stage1→Stage2 #4583/#2831、並行P0の通常`tei`→実Provider→二Consumer #2586/#1251は変えず、Graphの製品接続を新Hard Gateにしない。

**資格**：既存ReferenceのCodegenがmainに入ったことは検証済み。今回追加した正式Frontend/Role/SourceIndex接続、製品Go生成、実Plugin/Host Consumer、Pin/Browser、Owner業務意味受理は **NOT RUN / NOT ACCEPTED**。

### 「TEIで実装できた範囲」と「定義を簡素化できたか」を測る

| 観測 | 記録方法 | 成功とみなさない例 |
| --- | --- | --- |
| **意味実装の内訳** | 最初に機能・分岐・検証規則の対象集合を固定し、①TEI定義からの生成、②TEIから外部Goへ委譲、③手書きGoのみ、④未対応に分類する。PluginについてもRole・Targetを分けて分母と未対応を残す | 一行の委譲コードを生成しただけでFunction全体をTEI実装済みと数える |
| **生成物の実消費** | Source／Model Revision、Generator／Toolchain版、出力Role／Digest、実Child／Builder、実Entry／受信結果を連結する | ファイルの存在や同じJSON結果だけで、選択・コンパイル・実使用が証明できたとする |
| **定義の簡素化** | 同じ独立Expectedで、保守するTEI定義、補助Go、重複設定、例外指定、手修正、意味理解のための行き来を改訂前後で比較する | 行数が減っても隠れたGo条件や暗黙の既定値へ意味を移しただけの場合 |
| **変更への耐性** | 一つの表行／有向関係／Plugin契約を変え、生成差分→再Build→独立Expected→実Consumerを検証する | 旧Artifactの再利用、同名別Subject、A→B→Aの旧Evidenceを現在適格とする |
| **Pluginの自己構築割合** | 先に列挙したPlugin機能集合を対象に、TEIが生成して実呼出しまで確認できた機能数と、外部実装へ委譲した機能数を併記し、改訂ごとに比較する | Packageを一つ生成したので、内部の全OperatorとProviderが自己構築されたと数える |

初期段階に恣意的な百分率目標は置かない。**まず一機能で、TEI定義の変更が実行結果へ反映され、手書きGoの増加なしに正負の独立Expectedが通ること**を証明する。次に性質の異なるPluginへ拡大し、同じ基準で比率と保守量の変化を測る。数値だけを良くするために機能分割や委譲箇所を変更しない。

### セルフビルドの進歩を、実装範囲と保守負担の両面で判定する（2026年11月15日）

既存SB-0〜4とGraph/Ruleの固定Fixtureを、**継続的なドッグフーディングの測定手順**へつなぐ。[自己記述とコード生成の第44節](../design/ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)、[Pluginの役割・生成・実利用設計](../design/PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md)を再利用し、新しい工程表、Runtime、Registry、独立Expected正本は追加しない。

**判断の基準**：Go単一バイナリや既存Go部品の再利用は妨げない。ただし「TEIに記述したどの条件が、生成した実装のどの判断を決めたか」を説明できなければ、TEI主導の実装とは数えない。Goの呼出し契約の自動生成と、判断・検証ロジック自体の生成を区別する。

#### 同じ一機能について、改訂前後に残す記録

| 記録面 | 固定・比較する内容 | 改善と数えない条件 |
| --- | --- | --- |
| **機能上の義務** | Node/Edge同一性、向き、多重性、出辺選択、ID重複拒否を固定したIndependent Expected。第二例では型・NULL・境界値を持つRule | 成功しやすい義務だけを後から分母に残す、機能を小分けにして率を水増しする |
| **TEIによる実装** | 義務IDごとに①TEI由来の意味ロジック生成、②TEIから外部実装へ委譲、③手書き実装のみ、④未実装を分類。実生成箇所はSource occurrence・Role/Operator・Generator版へ戻れるようにする | Wrapper生成のみ、固定JSONの出力、手書きGo側の条件を直して結果を合わせる |
| **Plugin自身の自己構築** | Descriptor、実装ロジック、Validator/Generator、Build依存、実Provider呼出しを**別々の対象**として観測し、TEI由来の生成処理が実行された義務を記録 | Pluginを梱包できただけで、内部のOperatorやGeneratorまでTEI化できたと宣言する |
| **実際に消費された成果物** | Source/Meaning/Artifact Revision、Build Intent、Builder/Toolchain、入力閉包、生成Go Digest、Stage1/Stage2の実Invoker、選択Pluginの実Call、Consumerを各段階で固定 | 同じ値やCI Greenから実Builder入力、実Plugin利用、Stage1再入を推定する |
| **保守負担** | 保守する定義、重複Rule/契約、補助Goの手修正、例外設定、修正が必要な箇所、独立検収までの作業、AIへ渡す文脈と再質問を前後比較 | DSLの行数だけ減らし、隠れたGo分岐や暗黙の既定値、Promptへ複雑さを移す |
| **可読性・再利用性** | 業務専門家が図・表から入力→条件→結果を追えるか、CLIと第二Consumerで同じ意味を再利用でき、Sourceへ局所的に戻して変更できるか | 短くても難解な独自記号が増え、結局生成Goを読まないと判断できない |

**比率**：最初は固定した義務を一件ずつ列挙し、「TEI定義から判断ロジックを生成し、実Consumerの独立Expectedまで通った義務数／固定した義務総数」を示す。委譲、手書き、未実装の件数と分母を併記し、事後に義務の重みを変えない。Pluginでも実行された**実装義務**を単位とし、Plugin数・生成ファイル数・Go行数で代用しない。新規機能で分母が増えたときは、既存範囲の改善と新規範囲の拡大を別に報告する。実測前に恣意的な目標比率は設けない。

**簡素化**は一つの得点にまとめない。少なくとも手書きGoの変更箇所、意味の重複、変更対象のSource/設定数、独立検収に要した実作業を別に記録し、未計測は`unknown`とする。NULL/欠損・順序・Effect/Failureが変化したものを同じ機能の簡素化とみなさない。生成AIを使った比較は、課題・Harness・利用モデル等の条件差と、Token/費用・再試行の実測を分けて示す。

#### 最初の二回の改訂実験

1. **Graphからの実ロジック生成**。既存`testdata/tei-md-graph`で、同名二Nodeと別IDの多重有向Edgeから出辺の判断ロジックを生成する。`e-2`の反転、ID重複、配置のみ変更について、`*.tei.md`の最小修正→Source/意味診断→生成差分→実Builder入力→Plugin実Call→独立Expectedを一続きで照合する。正規Readerが未対応なら`TEI_MD_PARSER_UNBOUND`／`unsupported`を維持し、手変換したJSONを正式成功にしない。
2. **性質の異なるValidation**。既存Validation/Generator Referenceから、数量の正値など、**型・NULL・ゼロ・境界値**を含む処理を同じ方式で生成する。Graph専用の特別処理で終わっていないかを、Role/Generator/Pluginの再利用範囲と、追加で必要となった手書きGoの実測で判断する。丸め・欠損・型不適合の扱いを推測で埋めない。

両例の生成が通った後で、**Plugin自身のGenerator/Validatorの一部分**をTEI定義へ移す。Descriptor生成だけ、実ロジックの生成、Generator自身のTEI由来実装、Stage1の実CLIがStage2へPluginを渡して使ったことは別に記録する。実装順は「Reader不足の最小解消→Graph実ロジック生成→CLI実Plugin Call→第二Rule→GeneratorのTEI化→Stage1実Process再入」。既存#943/#951/#1121/#1251/#2620/#2831へ引き渡し、Studio GUIやMCP Serverを初回の前提にしない。**正式Windows Desktop Stage2（#4583）は並行する最上位の製品出口として維持する**。

生成AIによる開発は、`tei commands`で**実在する操作**を確かめ、対象Subject/Revisionを固定し、Source差分を提案・検証して、既存Build/Provider/Scenarioを動かす。MCPは同じ型付きOperationの別の入口とし、SkillやPluginマニュアルに未実装Commandや古いPlugin版を現在の操作として書かない。生成AIの提案、Build成功、Host Grant、Owner意味受理は別資格である。

**各改訂の報告**は、同一Subjectについて「義務の固定分母と生成／委譲／手書き／未実装の内訳」「Source→生成コード→Builder→実Call→Consumer」「変更前後の手修正・重複・確認作業」「Stage別の成立・未成立」「次にTEI側へ移せる処理」を一枚にまとめる。既存#1180/#2514のIndependent Expectedへ戻し、成功と負例を併記する。各改訂でこの記録を更新できることを、TEIがTEIの開発を段階的に改善しているかの継続判定とする。

**資格**：比較方法と次の実装順の追補であり、Graph/Validationの製品Go生成、実Plugin呼出し、Generatorの自己構築、CLI Stage1→Stage2、定義削減、Studio Stage2の新実測ではない。

### 図表の実現方法を比較し、TEIが担う意味を増やす（2026年11月15日）

セルフビルドで増やしたいのは**生成されたGoの行数**ではなく、TEIの図表・Rule・標準データ構造が責任を持つ**実装上の判断**である。別のコード生成設計では、図表のパラメータを既存の汎用関数が直接解釈する方が簡潔な場合もある。この選択肢は保持し、コード生成をすべての演算に強制しない。ただし「図表を解釈して動作した」と「図表からコードを生成して構築した」は別々に検収する。

| 実現方法 | TEI側に保持すべきもの | 合格として記録する範囲 |
| --- | --- | --- |
| **生成型** | 型付きData／Graph／Rule、意味Role、Operator契約、Generator入力と版 | TEIから出力した実判断コードをBuilderが消費し、選択されたArtifact／Consumerの動作がSource変更に追随した義務。**コード生成・Self-buildの証拠** |
| **解釈型** | 同じ意味のGraph／Ruleと、版固定した汎用Interpreter／Evaluatorの契約 | 固定したGraphの辺の向きやRuleの閾値を変えると、手書きGoの変更なしに実結果が変わる義務。**TEIモデル駆動の証拠**だが、コード生成の証拠ではない |
| **委譲型** | 外部Go／SDK／ProviderへのInput・Output・Failure・Effect・Capability契約 | TEI側で呼出し・構成・観測できたこと。処理の判断条件が外部の手書き実装にあるなら、**意味ロジックのTEI化率には算入しない** |

[型付きPlugin入出力Mapping§19](../design/PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)では、**複合型全体をまとめて渡す／必要なFieldだけつなぐ／明示Adapterで変換する**ことを、実現方式とは別の接続設計として扱う。最初は同じRecordで全体接続と二Field Mappingのread-only適格性を比較し、その後に一つの純粋Adapterを確認する。全体Portの折畳みとField展開は同一Mappingの表示であり、二重Callerや非公開Fieldの配送を許可しない。PIO-01〜10を既存#1180/#2514に引き渡し、#1223/#951/#2497の既存型・対応付けを再利用する。**この検討をStudio Stage1→Stage2／通常CLI実Providerの新しい先行Gateにはしない。**
**接続先で判断できる範囲まで確認する（[Plugin I/O §21](../design/PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)）**：main済み#5142の値状態と関係の取得範囲を、PluginのRecord全体／二Field Mapping／一段pure Adapterの受入条件へ渡す。実行パケットは①TypeRefと必要Field・Source IDのread-only照合、②NULL／未取得／partialを一因子で変える独立Expected、③ProviderまたはEvaluatorによる実Read IDと`undecided`理由の観測、④変換Loss・Disclosureの比較の順とする。main済み#5143のRule例検査と#5146の暦日比較は局所の意味検証、#5147の診断名被覆は到達証拠そのものではない。Draft #5148の期間非重複は未受理とし、正式Provider／Builder／Stage2 Windowsの既存P0を変更しない。


この三方式は優劣の固定順ではなく、**同じ意味契約の異なる実現手段**である。生成型と解釈型を比べる場合は、同じ受理済みSubject、独立Expected、入力・NULL／欠損・失敗・Effectの条件を使う。生成型が対象Sourceの変更による再Buildを必要とするのに対し、解釈型は一般にその変更をデータの再読込みで扱える。ただし、InterpreterやOperatorの変更、古い選択版、現在Grantの失効を「Build不要」と混同しない。実装方式に応じてSource/Meaning/Artifact/実ConsumerのRevisionを別々に照合する。

**最初の比較は既存の二題材に限る。** 同名Nodeと多重有向Edgeの出辺選択は、既存Graphの一つのMeaningから(a)生成Goによる出辺処理、(b)共通Evaluatorによる同じ出辺処理を作り、Edge反転・削除・レイアウト変更・ID重複の対照で実結果と再構築範囲を比較する。数量のpositive Ruleは、現行の生成Goから実CLI Build・Binaryまで到達した[PR #5009](https://github.com/hat47x/tei-platform/pull/5009)を**限定基線**とし、型・NULL・ゼロ・負値・未取得の既存Expectedを使う。両題材の差を通じて、共通Operator／Providerを再利用できたか、Graphだけの特別実装を増やしていないかを判断する。既存の[Source原文同梱と実Binary照合](../design/PLUGIN_RULE_PRODUCT_VERTICAL_SLICE_2026-11.md#1413-原文byte列もbuildplanの入力へ含め実binaryから読取りを確認する)は「原文の実消費」であって、Reader／Generatorの意味受理や本番配布への原文公開許可ではない。

**方式の採否は機能ごとに測る。** 同じテストで実際に修正したTEI定義／補助Go／設定の箇所数、手動で直した生成物の有無、独立検収までの作業、cold/warmの生成・起動・評価時間、Target依存、再利用できるPlugin契約を残す。実速度や保守負担を測らずに、必ず生成型が速い、または必ず解釈型が簡単とは決めない。可読な図表・表の表現を保持し、意味を隠す一行マクロを単純化の成功に数えない。

**Plugin自身にも同じ区分を適用する。** DescriptorやBuildPlanの生成、Pluginの実判断処理の生成、生成器の自己記述、Pluginを利用した次段のBuildはそれぞれ別の実装義務である。特に、解釈型で動くPluginをTEIが構成できたことだけから、Generator自身の自己ビルドが成立したとは判定しない。最初のセルフビルド合格には**意味のあるコード生成の実経路を少なくとも一つ**残し、Stage1のTEI実ProcessがStage2を構築して同じPluginを実利用したことを別に確認する。Source変更で生成Goと実Builder入力を取り違える負例は[PR #5056](https://github.com/hat47x/tei-platform/pull/5056)の独立検証が統合・再検収されるまで既存達成に数えない。

**既存の担当と次の差分**：#2620/#988/#1018がSource/Roleの読取り、#1121/#4544が生成、#951/#1251が実Plugin選択・呼出し、#943/#2831がBuildとStage再入、#1180/#2514が独立Expected、#2600が失効・再利用、#4583がWindows Desktopを受け持つ。次の実装はGraph生成型と解釈型を別Targetで並行構築することではなく、**同じ固定Graph／Ruleの意味IDから二方式を選び、既存Operator／Generator／Providerの実出力と最小負例を比較**する小さな差分にする。MCP／Studioは同じ型付きOperationの投影とし、CLI先行・Go単一バイナリ・外部Hostを独自に置き換えない。新しい万能Interpreter、Graph Compiler、Host、Registry、第二Build Engineを初回の前提にしない。

<!-- tei-product-direct-generated-first-experiment-20261115 -->
**既存実装の再利用**：[Plugin実現方式§10](../design/PLUGIN_REALIZATION_AND_OPTIMIZATION_REQUIREMENTS_2026-11.md)の第一例 `sales.order_line.quantity_positive` は、main `84c335b` の `cmd/tei/cli_self_build_generated_rule_go_test.go` で、**汎用評価と通常`tei self-build`による生成Goの実結果を、同一Rule・正値／ゼロ／負値／nil／取得不能の固定ExpectedとRead回数で比較**する局所実装まで進んだ。このGo回帰を新しいテストへ複写せず、既存の意味契約を直接評価する場合とコード生成する場合の基線として活用する。なお、main収録とexact HEAD CIの成功・正式Builder入力閉包・実Providerの選択資格は別の判定である。

**次の検収**は、同じRule／Functionについて、実現方式Aを選んでいるのに実CallerがBへ到達する不一致を、値が同じでも拒否できるか（RZM-04）。コード生成を省く場合も、Runtimeで使用するモデルbyte・版・Scope・必要Contextの現在性を確認する。#951/#1251が実選択／実呼出し、#2595がBuild入力、#1180/#2514が独立Expectedを担当する。**R2-P出典閉包、R3公開Entry、Studio Stage1→Stage2のP0は引き続き独立**とし、全域用のInterpreter／第二Resolverを設けない。

**現時点の資格**：現行mainにはRuleから生成したGoを通常TEI CLIのBuild経路で実Binaryへ収録する限定的な実証と、原文コピーの実BinaryによるHash照合がある。一方、同一意味のGraphを生成型・解釈型で実比較した結果、Generator自身のTEIによる実再構築、正式Function Entry→選択Providerの製品実Call、Stage1→Stage2のTEI CLI／Windows Studio完全セルフビルド、定義簡素化の定量実測は本節では**NOT RUN／NOT ACCEPTED**。この追補は実装方式を選ぶための計画であり、達成済みという主張ではない。

### Pluginの型付き貢献を、実現方式と配置から独立に検収する（2026年11月16日）

[Pluginの貢献・配置・適格性設計](../design/PLUGIN_PORTABLE_CONTRIBUTION_CONFORMANCE_2026-11.md)と[PC-0〜5の小PR計画](PLUGIN_PORTABLE_CONTRIBUTION_EXECUTION_2026-11.md)に、全体Port／Field Mapping／明示Adapter、解釈／生成／委譲、静的Go／独立Process／Remote、CLI direct／Wails公開を**別軸**として接続した。まず既存Rule／Inspectのread-onlyな接続票から始め、正式Builderと通常CLI actual Providerの成立をそれぞれ確かめる。HTTPの公開資格、Audience、Grant、実Consumerはその次の別票とする。

PCON-01〜09は既存#1180／#2514へ渡す未実行の独立Expected候補であり、Node Viewの線から実行を作る、Plugin Packageを一つのRuntime単位に固定する、第二Resolver／Host／Registryを追加することはしない。**Stage1→Stage2 Windows実受領、R2-Pの実Builder、R3の通常CLI実ProviderというP0を維持**し、プラグイン自身のTEI化は「一つの意味ある義務が生成Go／実Provider／次段Buildで実際に使われたか」で段階判定する。API Pluginの未統合PR #5096のCIや別HEADの成果を達成済みとして算入しない。

**PC-1の最初の実コード差分**：既存`internal/plugin/transform`による全体Record／二Fieldだけの出力を比較するmain統合済み[#5156](https://github.com/hat47x/tei-platform/pull/5156)のGo回帰で、内部項目の非選択・必須項目の欠損拒否・明示NULL保持を確認する（[PC-1a〜c](PLUGIN_PORTABLE_CONTRIBUTION_EXECUTION_2026-11.md)）。これは出力Valueの整形試験であり、Contract Field ID、取得完全性、HostのDisclosure／Grant、通常CLIからの非Builtin Provider呼出しを証明しない。PC-1bの型・取得範囲、PC-1cの明示Adapterを別Ownerと独立Expectedへ渡す。main済み#5147のPlugin Resolver診断コード負例も再利用し、新しい診断正本を作らない。
**PC-1bの着手境界（11月17日）**：[Plugin入出力設計§23](../design/PLUGIN_DEPENDENCY_IO_MAPPING_DESIGN_2026-10.md)と[PC-1b小PR計画§7](PLUGIN_PORTABLE_CONTRIBUTION_EXECUTION_2026-11.md)に、同じAttemptのQuery／Relation／Attribute Ref→実行結果行→`validationread`→`validationeval.Decide`の負例を整理した。mainの#5148の期間重複Ruleと#5149由来のReaderは既存の局所実装だが、Callerの`coverage=complete`を全件の観測証明にしない。#5184の実PG反例は、投影行が等しくても非投影のPerson対応は決まらないことを示す。最初はPC-1b-0の実Read ID・NULL／未取得を一対象で固定し、PC-1b-1でQueryのScope・時点・Pagingを裏付ける。#5160／#5197／#5201／#5204のPlugin解決負例は別Ownerで扱い、曖昧と不在やLockと適合不一致を同じ案内にしない。いずれも正式Provider R3、Builder R2-P、Stage2実Windowsの完了へ繰り入れない。

**並行設計の取り込み（11月16日）**：main済み#5162／#5163の「未取得部分があっても結論を確定できる場合と、保留すべき場合」の区別を[PC-1bの一因子Fixture](PLUGIN_PORTABLE_CONTRIBUTION_EXECUTION_2026-11.md)へ結び付ける。接続可能性・Ruleの判断可能性・Consumerへの配送可能性を三段階として示し、PIO-17〜23／DPC-09〜13の既存Expectedを再利用する。ここでの文書接合とPC-1aのpure Transformテストを、実Runtime Provider／Host公開・正式Builder・Stage2の達成とは数えない。

### 検査義務を一つTEIへ移し、次のTEIで使う（2026年11月15日）

[実装からモデルへのdogfooding §15](../design/IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md)に、`GeneratePositive`の**ID／Attribute必須・Relation禁止のRequired Shape一義務**を、既存標準構造・Ruleによる検査へ移す最初の自己置換手順を定めた。M0（出典・独立基線）→M1（TEI由来の実検査）→M2（実製品経路の旧Go分岐削減）→M3（Stage1実TEIによるStage2構築・当該機能の実利用）を**別々の合格**として扱う。M1は生成・解釈のどちらでもよいが、セルフビルドの実証には意味あるコード生成と実Builder消費を別途要求する。Target固有のGo package名検査やNULL評価のLawを初回の一義務へ混ぜない。

次の実装PRは**M1の一条件だけ**に絞り、既存`TestPositiveRuleShapeIndependentBaseline`の独立Expectedを守る。TEIによる実検査と旧Go分岐の二重保守、旧実装への黙ったfallbackは前進と数えない。追加後は実Goからの判断削減と定義・確認負担を比較してM2へ進む。Graphの6入力、正式Reader未接続、#5054/#5056の別PR、R3実Provider、Windows Studio Stage2は各Ownerの独立検収を維持する。**計画のみであり、M1〜M3と簡素化の実測は未実施。**

### 既存Expectedへ引き渡す負例

SBX-01：図のレイアウト変更では意味が変わらず、関係の向きや多重度を変更すれば生成内容・結果が正しく変わる。SBX-02：TEIの表の条件を変えてもGoの固定処理しか動かない場合は「委譲のみ」として失格。SBX-03：生成GoのDigestが一致しても、Builderが別Sourceや未選択Pluginを実消費すれば失格。SBX-04：Stage0が別々に二つのBinaryをBuildしただけではStage1→Stage2再入不適格。SBX-05：Plugin A→B→Aの後に旧Binding／Grant／Receiptが自動復活すれば失格。SBX-06：定義を短くした結果、NULL／欠損／評価順／Effect・Failureが変われば不適格。これらは[#1180](https://github.com/hat47x/tei-platform/issues/1180)・[#2514](https://github.com/hat47x/tei-platform/issues/2514)の既存Independent Expectedへ渡し、新しいScenario正本は作らない。

### 既存計画との関係

**CLI先行はDesktopセルフビルドの最終目標を撤回するものではない。** 最初の検証をCLIで軽量化しつつ、[#2831](https://github.com/hat47x/tei-platform/issues/2831)・[#4583](https://github.com/hat47x/tei-platform/issues/4583)のBuildExecution／ArtifactSet／親子Attemptを共通化する。現行BuildIntentの`Subject.Kind=plugin`制約は[#1080](https://github.com/hat47x/tei-platform/issues/1080)・[#1097](https://github.com/hat47x/tei-platform/issues/1097)で非Plugin Build Subjectの不足として扱う。Plugin選択・段階別Contributionは[#951](https://github.com/hat47x/tei-platform/issues/951)・[#1251](https://github.com/hat47x/tei-platform/issues/1251)、自己記述Languageと糖衣形式は[#988](https://github.com/hat47x/tei-platform/issues/988)・[#1018](https://github.com/hat47x/tei-platform/issues/1018)の責務とする。

CLI／MCP／Studioは同じ型付きApplication Operationを異なる入口から使うものであり、入口の共通化はHost認可や実効果の証拠を省く理由にならない。Runtimeで使わないGenerator／Editor専用Pluginは、必要のない生成成果物に無条件同梱しない。本節は実装順と測定方法の合意案であり、TEIが生成したGoによるCLI／Plugin再構築、定義の簡素化、Stage1→Stage2製品成功を**実証済みとはしない**。

### 一つの変更を、生成・実行・再検証まで追跡する

SB-1/SB-2で導入するGraph/Ruleの固定例について、**初回から変更前後を同じ独立Expectedで比較**する。最初の実装PRは新しい汎用Graph Compilerや増分Cacheを要求しないが、将来#2600で正しい再利用を行うため、何が変わったかと、どの工程を実行したかを取り違えず記録する。詳細な契約は[実行モデル§37〜39](../design/PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)に集約する。

| 比較する入力 | 最初に見る意味・生成の差 | 実行・再利用上の停止条件 |
| --- | --- | --- |
| 基準：同名Node二件・別IDの同向Edge二本 | 出辺ID・件数の計算とID重複の拒否が、TEIのRuleから生成される | 固定JSONや外部Goへの丸投げは自己生成実装として扱わない |
| `e-2`の方向だけ反転 | `n-1`からの出辺が二本→一本、`n-2`から零本→一本 | Graph表示だけ更新して実Go処理・実Provider結果が旧状態なら不合格 |
| `e-2`を重複IDへ変更 | 一意性違反をReader/Validator/生成規則の適切な位置で拒否 | 同名や同端点を理由にEdgeを黙って統合しない |
| 図の配置・注釈だけ変更 | 意味上の計算は不変、Source/表示の変更は追跡 | Raw Source再確認を省略したと偽らず、無関係な下流再Buildだけを省略可能か検査 |
| Provider/Builderの選択やSourceのA→B→A | 結果が同じでも選択先・実消費byte・現在資格が変化し得る | 旧ArtifactやReceiptを現在の実Provider使用・Owner受理へ自動昇格しない |

記録の最小単位は**一つのSubject、固定したSourceとRole、選択Provider/Generator、Build Attempt、実Artifact、実Consumer**である。未接続ならその境界を明示する。各工程でParser/意味解決/生成/Go Build/Plugin実行に費やした時間、実行を省略した理由、未観測の入力も残す。Source/Meaning/Artifactの三種のRevisionを一つのFingerprintへまとめない。

初回の実Go生成は、意味処理の実装部分と手書きの補助Goを分けて検証し、生成定義から出辺IDの判定が実際に変わることを確かめる。通常`tei` CLIと独立Consumerの実Call、Stage1 CLIによるStage2 CLI再入、正式Studio DesktopのStage1→Stage2は、**異なる到達点として順番に実証**する。部分的な成功を足して全体成功と呼ばない。

この比較はSBX-01〜06と#1180/#2514、#2595の実消費Source、#2600の再利用、#4495/#1251の製品Provider、#2831/#4583の実セルフビルドの担当へ引き渡す。新たなScenario正本やRuntimeは設けない。**状態：実証手順の追補。新Go生成・増分skip・実CLI再入・正式Desktop再入の合格は未確認。**

## プロセス・イベント・応答を共通の意味から説明する（設計候補）

Webの要求、CLIの起動、タイマー、Message、画面操作には、それぞれ異なるHost・起動方式がある。ここで標準化を検討するのは一つのイベントループではなく、**Trigger→型付きInvocation→Function/Rule→State/Effect→Outcome→Response／Consumer受領**という意味の対応である。長寿命Processの起動・停止と、一件の処理の成功・失敗も混同しない。

初期の受入は、受注受付の説明用Fixtureを使い、HTTP/CLIの二入口が同じ意味Operationへ至るかをHeadlessで照合する。期限の境界、現在Grant、DB Commit後の通知結果不明、重複・逆順Event、未実装のTimer Providerを負例にする。外部部品はContractとTODOを先に記述し、既存Driver ScenarioのMockと独立Expectedで基本設計からシミュレーションする。ただし、Mockの成功は実DBやBrokerの実行証拠にならない。

詳細は[Lifecycle・Trigger・Operationの共通意味設計](../design/PROCESS_LIFECYCLE_TRIGGER_OPERATION_SEMANTICS_2026-11.md)を参照する。Ownerは[共通Application API #1158](https://github.com/hat47x/tei-platform/issues/1158)、[Script Launch #4495](https://github.com/hat47x/tei-platform/issues/4495)等の既存Issueに分け、新Runtime・Dispatcher・Schedulerを前提にしない。**現行P0のStudio Stage1→Stage2と通常CLIの実Readには、新しい先行依存を追加しない。**

**2026-11-15実行図の最小Go回帰**：[実行モデル図解§63.1](../design/PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)を起点に、`internal/operations/inspect_capability_source_evidence_test.go`で`reference/workspace`内の実Capabilityを`InspectSubject`→Source Index→`TraceSubjectSources`で読み取り、原byte識別子とArtifact fingerprintを比較する。存在しないCapabilityはTraceの空集合とInspectorの正規化対象拒否を区別する。**Go局所テストの追加・合格だけでS-LOCの実Workbench表示、S-OPEN、正式`*.tei.md` Reader、実Provider呼出し、Windows Stage2合格に昇格しない。** 同設計§63.2では、`ReadWorkspaceSubject`を用いたCapabilityのSource改版（一つの末尾改行）とA→B→AのGo負例も追加し、Canonical同一性、管理対象Source版、Source Index/Artifact byte版、現在Grantを別々に評価する。 **同設計§63.3で扱うRule Markdownの製品入口**：内蔵`tei.rule-authoring/v0`が実選択された`internal/workspace/testdata/tei-md-rule`をGoの`workspace.Load`で読み、二件の`semantic.Validation`と同じLoadから得たSource ID byte spanを`InspectionFromLoad`／`BuildSourceIndexFromLoad`で照合する回帰を追加する。説明用`text`コード領域の規則は定義として取り込まない。 **同設計§63.4の次の実行境界**：実選択したRuleの型付きMeaningを既存Evaluatorへ渡し、合成`validationeval.Source`に対する実際の`Number`／`RelationKeys`呼出しを固定Expectedと突合する。続いて入力データとRule IDを固定し、Markdownの評価対象Attributeのみを変更して、現在のTyped MeaningとRead Footprint・評価結果が変わることを検査する。これは生成Go方式Cや実Provider接続の証拠ではなく、次の一因子対照に引き渡す基線とする。さらにmainで限定実装された`validationgen.GeneratePositive`（#4911）を使い、二つの`std.decimal`属性を持つ固定モデルと同一入力の下で、Ruleの参照Attributeだけを変更し、実Source選択→Go再生成→別Process実行の結果・実Read先が追随するかを新しいGo回帰で照合する。これはR1の局所生成試験であり、Builder実入力・Artifact／Plugin選択・通常CLI実Provider・独立ConsumerはR2〜R4として別検収する。数量0／NULL／未取得、重複キー／完全と明示した空集合／範囲不明を、業務不成立と評価エラーに分離する。**実行したのはEvaluatorであって、製品ProviderやDriverから実データを読んだわけではない**。Input取得範囲・Rule集合・時点・実Provider Callは#988/#1180/#2514/#4495で別受入とする。製品に接続済みなのはこの限定Rule Readerであり、Graph/Windowの正式Reader、外部Language Plugin、通常CLI実Provider/DB Effect、現在Grant/Windows Stage2を完了と扱わない。これは実Goの同一読取境界の検査であり、Studio Rendererが現在のSource/Bundleを受領した証拠や現在Grantの成立ではない。 これらは引き続き既存#1124/#1824/#2621/#4495/#2831/#4583の実Consumer検収とする。

**2026-11-15実行図の製品引渡し**：[実行モデル図解§57〜60](../design/PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)では、最初のread-only回答を「固定した一つのEntry／Functionについて、定義・選択・実現・観測を結合できる範囲まで示す」に絞る。#4800の実Plugin Inventory exact LockはDocument／ProviderのLoad・実Call・Disclosureを保証しない。#4805のREA Candidateは同一ID／申告Digestでも観測Factsが変わり得る。#4792のWindow S0-Pは正式S0-Aと別、#4806のInspector制限表示は#4812の生成／配布Asset・実Consumer検証と別である。**実装の最初の切り方**は、#4812のAsset版一致、#2621の一Graph S0-A、#4495の公開pure Entry→実Go Provider→通常CLI、#2831/#4583の実Stage2を既存Ownerに維持する。Mock/TEI内評価は#4808に従い明示的なシミュレーションとして扱う。新しい共通Tracerや一律Hard Gateは設けず、未観測のCall/Effect/受領をPASSと表示しない。

**SL-1の実Input／Source照合（2026年11月15日、[Script Launch §29.5〜29.6](../design/SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md)）**：mainの[組織as-of Query修正 #4847](https://github.com/hat47x/tei-platform/pull/4847)により、宣言した`valid_at/known_at`をSQL Placeholder／実Binder／実DB用試験と一致させる必要性が具体化した。これを公開pure Function一件の、受理済みInput→typed Request→選択済みProviderの実引数の対照へ移す。未宣言Alias／必須欠落／NULL・0・未取得の混同は実Call前に拒否し、結果値が一致するだけの入力不一致を通さない。

さらに[Capability InspectorのGo回帰 #4853](https://github.com/hat47x/tei-platform/pull/4853)では、実Canonical IDとSource Index／元byte位置の一致が検証された。この**読取り専用の出典証拠**をSL-1のPrepareへ利用する一方、Entryの公開・Meaning受理・Host現在Grant・Provider実Callは別に照合する。次の受入は#4495と#1158、独立Expectedは#1180/#2514が担当し、Reader S0-AとStage1→Stage2のP0を先行依存にしない。**本追補は計画上の接続であり、pure Entryの実Provider呼出しと通常CLIの新しいPASSは未確認。**


**SL-1のRule Markdown実Source・非Entry対照（2026年11月15日、[Script Launch §29.7](../design/SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md)）**：製品`Workspace.Load`には内蔵`tei.rule-authoring/v0`の厳密選択と、同一Load原文から`semantic.Validation`と元byte anchorへの読取経路が既にある。従前の「すべての`*.tei.md` Readerが未接続」は現行mainの実装範囲を正しく表さない。最初のHeadless対照は、実選択済みRuleの`semantic-validation`がSource Indexへ結び付く正例と、Rule IDが同名の公開`semantic-capability`／Function Entryを自動生成しない負例を同一Fixtureで確認する。**Rule読取成功をSL-1の実Go Provider・通常CLI起動成功に加算しない**。[#4900](https://github.com/hat47x/tei-platform/pull/4900)の旧Validation↔Markdownの製品型付き意味同値・二重宣言拒否と、[#4901](https://github.com/hat47x/tei-platform/pull/4901)の既存Evaluatorによる八つの合成入力・Read Footprint照合は、実Reader→既存Evaluatorの前進として区別して記録する。次の実装は合成Sourceの「Read要求に答えた」状態と、実Targetの「必要な属性・関係・時点・範囲を漏れなく取得した」状態を別資格として検収し、未取得を空集合やNULLに置き換えない。続く実装は#4495の受理済みpure Function・Host現在Grant・実Binder引数・selected/actual Provider・独立Consumerの別検収を行う。Graph Markdownや外部Language Plugin／Source Open／Studio Stage2は別OwnerのP0を維持する。

**SL-1の生成Rule Go実消費（2026年11月15日、[Script Launch §29.8](../design/SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md)）**：[生成Go実装 #4911](https://github.com/hat47x/tei-platform/pull/4911)では製品で選択した`semantic.Validation`から`validationgen.GeneratePositive`でGoの正値判断を生成し、別OSプロセスの`go run .`で正値／0／負値／NULL／欠損を検査している。既存EvaluatorでRule原文変更がRead先・結果へ伝わる[#4913](https://github.com/hat47x/tei-platform/pull/4913)とは別のR1証拠である。**両試験を足して「編集した原文から生成Goを再構築し、そのコードが製品Builder・公開Entry・実Providerで使われた」とは判定しない。** 最小の追加は、(1) 原文一因子編集→製品再Load→型解決→生成Goと独立Expectedの差分、(2) 生成Go byte／生成器版の実Builder入力とBundle受領、(3) 公開pure FunctionのHost／実Binder／選択済みProvider実Call、(4) 通常製品`tei` CLI別プロセス・独立Consumerの順で、#1121/#1251/#4495へ分けて実装する。R1のRule生成経路と並行P0のpure Entryは相互の先行条件にしない。Rule IDからFunction Entryを自動発行せず、Synthetic Source・Go Run・製品CLI／Provider Witnessを別欄に残す。

**SL-1のR1.5と参照Builder実消費の到達点（2026年11月15日、[Script Launch §29.9](../design/SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md)／[Plugin Rule §14.5](../design/PLUGIN_RULE_PRODUCT_VERTICAL_SLICE_2026-11.md)）**：[#4932](https://github.com/hat47x/tei-platform/pull/4932)・[#4930](https://github.com/hat47x/tei-platform/pull/4930)で、選択Markdown Ruleの一因子改版→実再Load→型解決→生成Go更新→別プロセス・別バイナリのRead先／判断変更を検査するGoテストがmainへ入った。さらに、旧[#4934](https://github.com/hat47x/tei-platform/pull/4934)は未マージながら、同一Blobの`tools/reference-go-build-from-plan/validationgen_build_binding_test.go`が現行mainに存在する。同PRのHealthログでは参照Builderパッケージの試験が成功しており、生成Go／依存GoをBuildPlanのDigestで固定、古いPlanを拒否、`-isolate`実Build、Artifact Digest・Lock・実Binaryを照合している。これは**参照Builder局所R2の先行証拠**であり、正式製品BuilderがRule原Source・Language／Generator版・Owner意味受理を同じBuild Intentとして選択・消費した証拠ではない。次のR2は#4544／#2595の製品Build入口で、Source→型付きMeaning→生成器版→生成byte→Builder実読取byte→Artifactの束縛と負例を一件に絞る。R3は#4495の**別に受理済みpure Function**→Host現在Grant→実Provider→通常`tei` CLIを並行P0で続ける。Data Ruleと独立Functionの境界（#4937）や、Node／Flowのread-only投影と実行資格（#4924）を混同しない。Stage1→Stage2のWindows実受領も独立P0。


**SL-1の実CLI Inspect／非Entry対照（2026年11月15日、[Script Launch §29.10](../design/SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md)）**：新規`cmd/tei/cli_inspect_rule_markdown_process_test.go`で、実ビルドした通常`tei inspect`を別OSプロセスから実行する。選択済みMarkdown Ruleのread-only Canonical／Source byte Range／Artifact Fingerprint一致、同名`semantic-capability`と非`tei`文書Fence内の見せかけRuleの`SUBJECT_NOT_FOUND`を、固定Expectedと実exit codeで照合する。Go内部Inspectorの既存#4902を複製するのでなく、**CLI境界がKindと出典を保ったまま失敗を公開するか**を追加検収する。既存Action Surface実CLI WorkflowでこのGoテストだけをexact HEADに対して独立実行し、全体Healthの他パッケージ失敗と切り分けて記録する。これは公開Function Entryの実行、Host Grant、actual Go Provider Callや独立Consumerの受領ではない。次の実行側P0は#4495、Builder側R2-Pは#4544／#2595、Studio Stage1→Stage2は#2831／#4583。

**SL-1の通常self-buildとOperator実装選択の接合（2026年11月15日、[Script Launch §29.11](../design/SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md)）**：[PR #5009](https://github.com/hat47x/tei-platform/pull/5009)で、選択Rule→実生成Go→`buildintent.Resolve`→通常製品`tei self-build`→隔離Builder→実Binary判断という経路のGoテストがmainへ統合された。既存セルフビルド専用CIで`TestSelectedRuleGoThroughRealCLISelfBuild`を単独実行し、Linux/amd64、実Entrypoint Source ID、Plan／Artifact Bind、再現性と五つの固定Expectedを検収する。[PR #5007](https://github.com/hat47x/tei-platform/pull/5007)のOperator選択はMeaningではなくWorkspace Compositionであり、変更後の古いScenario Command再資格化・実装Lock・Product CLI全体の呼出しが残る。**Builderが実際に読んだ生成Goの資格と、元Rule／Generator／選択実装の現在版を同じBuild Intentで束縛すること**を次のR2-Pの受入単位にし、R3のOwner受理済みpure Function実Provider Invokeと区別する。Studio Stage1→Stage2のP0は維持する。

## 進行の記録と完了条件

各G項目には、対象と版、既存の足場、次の作業、独立した正例・反例、依存、担当Issue、得た結果または未実施の理由を残す。状態は定義済み、実装候補、局所試験、通常入口での実行、利用側確認、案件受入を用途に応じて記す。Issueをclosedにしただけで、別の組合せや実行先が成立したと扱わない。

中小規模案件の対応率は事前に定めた案件と必須条件の集合に対して測る。標準のみ、既存資産接続、新規拡張、未成立を分け、保留・実施不能の理由を示す。大規模案件の省力化は生成時間に加え、追加質問、手修正、レビュー、統合、更新と引継ぎを比較する。性能は正しい結果を保った入力・構成で測り、量、待ち時間、メモリ、費用と取消後の処理を残す。

計画への反映が完了した状態は全G項目の担当、設計への接続、次の受入、依存を確認できることとする。製品としての達成は各Issueと既存の受入経路で判定する。理想像が更新された場合は、既存の項目との関係を先に確認し、不足する作業と条件を同じ担当へ戻す。 [生産性 #2018](https://github.com/hat47x/tei-platform/issues/2018) · [案件対応範囲 #3796](https://github.com/hat47x/tei-platform/issues/3796)


<!-- tei-product-self-description-operator-selection-handoff-20261115 -->
### 自己記述から実装選択・実使用へ進む最小検収（2026年11月15日）

[自己記述設計§48〜49](../design/ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)を、既存P0を妨げない二つの実装Packetとして引き渡す。

| 並行するPacket | 次の実装と合格範囲 | 既存Owner・停止条件 |
| --- | --- | --- |
| **SD-M1：形と診断** | 製品で選んだ quantity_positive のID／Attribute／Relation／解決型と原Sourceを既存Inspectorからread-onlyで対応付け、Required Shape候補と独立Expectedを照合する。次にGoの重複する検査一箇所を生成処理で本当に置換した場合だけM2とする | #2620／#4359／#1180／#2514。Descriptorを文書へ移しただけではM2ではない |
| **SD-O1：実装選択と実Call** | 同一Operator宣言・入力0のままmanifestの実装名を non-negative→non-zero と変更。Worldが同じでも実Call・結果が変わること、旧生成Scenarioコマンドの選択改版検出が不十分なことを一因子反例で固定する | #988／#1018／#2600とScenario Owner。現行の組込み実装名を外部Plugin Lockと同一視しない |

それぞれの観測は異なるSubjectに属するため、成功を単純加算して全体セルフビルドの合格にしない。後続は既存の**R2-P：生成Go byte→正式Builder入力→Artifact**、並行する**R3：受理済みpure Function→公開Entry→実Provider→独立Consumer**、さらに**Studio Stage1→Stage2のWindows実受領**へ渡す。Node View／Graph／文書／AI向け説明は既存Meaning・Source・版を読み取り専用で投影し、意味受理・実行権限・逆編集権限を自動付与しない。新Core・Registry・Runtime・横断Hard Gateは導入しない。

**状態**：現存のGo実装とPR #5007の限定試験を前提にした受入計画。新Descriptor検査／Go分岐置換、生成Scenarioコマンドの失効修正、外部Plugin、正式Builder、通常CLI actual Provider、Studio Stage2の新検証はNOT RUN。

<!-- tei-product-projection-requalification-20261115 -->
### 設計Projection・実装選択・生成物を同じ成功へ畳まない（2026年11月15日）

[横断受入計画§11〜14](../design/DERIVED_PROJECTIONS_RUNTIME_AND_MIGRATION_ACCEPTANCE_2026-11.md)を製品完成の評価へ接続する。mainにある**実Rule→Go→通常CLIの限定Build（#5009）**、**文書Operator→Workspace選択→Scenario実Call（#5007）**、**二Source→Go AST／元byte位置（#5020）**、**Windows Pin再読込試験のコード（#5025）**は、同じFunctionが全経路で動作した根拠として加算しない。

まずPQ-Aで原RuleのSource／Reader／Generatorと実Builder入力Go byte・Artifactを同一Build Attemptへ束縛し、PQ-Bで同じWorldでも`implementations:`切替後の旧Scenario選択・結果の無条件再利用を拒否する。両者の意味・実装・時点を独立に確認した後、正式Function一件の`入力→Rule→実Provider→独立Consumer`へ段階的に接続する。期待値は既存#1180/#2514を使用し、別のテスト正本を作らない。

次の最上位P0は**通常CLI→実Provider実Read**と**Stage1→Stage2実Windows WebView2**のままとする。Studio Pin試験コードの存在や単独Self-build CI成功をStage2成立としない。業務Snapshot、現在Grant、移行先DB、帳票の実取得はOwner/Host/Providerが持つ別資格を実証するまで未完了。新規Framework／常駐Host／Runtime／Migration Engineは前提にしない。

<!-- tei-product-self-described-conformance-first-witness-20261115 -->
### 自己記述M1から実生成・実Buildへ渡す最初の一義務（2026年11月15日）

[自己記述設計§50](../design/ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)では、Readerや表の種類を増やすよりも、**手書きの意味処理を一箇所置換し、実Consumerが利用したことを証明する**順番を優先する。

| 対象 | 最小実装差分 | 成立させる資格と既存担当 |
| --- | --- | --- |
| **SD-M1／M2** | 選択したpositive RuleのID／Attribute／Relation／解決型をSource付きでread-only投影し、独立Expectedで不足・禁止・型違いを検査。採用できた一条件を生成処理で置換し、旧Goの重複検査を実利用経路から取り除く | M1の説明・検査とM2の実置換を別検収。#2620／#4359／#1180／#2514 |
| **R2-P出典閉包** | main #5009の通常tei self-build実Binaryと、#5020の原Source→生成Go ASTの局所証拠を、同一AttemptのReader／Generator／Build Intent／Builderが読んだ実Go byteへ接続する | Markdownを偽のgo-packageへ混ぜず、既存Build入口を使用。#4544／#2595／#1121 |
| **SD-O1選択鮮度** | OperatorのLawを入力0の独立Expectedで確かめ、manifestの選択変更を旧Scenario Commandの失効条件として実検査する | World同値と選択／実Call／Law適合を区別。#988／#1018／#2600 |
| **外側の製品受入** | 受理済みpure FunctionのR3公開Entry→実Provider→独立Consumerと、非Plugin Studio Stage1→Stage2 Windows実受領をそれぞれ実測する | R2-P／SD-O1の局所PASSでは代用しない。#4495／#1251／#2831／#4583 |

印付きData項目表はtei.authoring/v1で既存Data意味へ入る**別の読みやすい表現**として検収し、Runtime RecordやMeta契約の正式文法とみなさない。Operatorの実装選択を説明できても、対象Lawの適合・外部Pluginの実byte Lock・実行Grantは別途確認する。**変更した手書き意味分岐の数、固定した実装義務の分母、再生成・Review負担**を比較し、Descriptor件数や生成Go行数を実装置換率にしない。

**状態**：既存製品の限定成果を次の作業へ結んだ計画のみ。新M1／M2実コード、正式R2-Pの原文閉包、Scenarioコマンド失効修正、R3実Provider、Stage2 Windows新受領はこの更新で実行・完了していない。


<!-- tei-product-shape-independent-baseline-source-disclosure-20261115 -->
### 自己記述の実装移管に必要な独立Expectedを先に固定する（2026年11月15日）

[自己記述設計§51](../design/ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)に従い、positive一規則の必須ID・Attribute・Relation禁止・解決型について、**固定の正常・負例**をfocused Goで追加した。Go package名は業務DSLの条件に含めず、Target Lowering固有の不正入力として別に検査する。これは現行手書きGeneratorの受入形を反証可能にするための基線であり、M1のDescriptor由来検査やM2の実装置換そのものではない。次はこの基線を保持したまま、TEI定義から検査を一条件生成し、そのGo手書き分岐を実使用経路から取り除く。

mainのRule生成Go CLIビルド回帰は、選択Markdown原文のコピーをGo embedで同梱し、完成Binaryが算出する原文Hashまで観測する段階へ進んだ。これを正式R2-Pの「元Source／Reader／Generator資格を同一Attemptに束縛した」実証に読み替えない。原文をBinaryに収録するかは**開示資格を持つ公開可能なFixture**に限定して検証し、製品・Plugin・Studio・AI配布への原byte同梱は既存DisclosureのOwner判断なしに一般化しない。

並行するScenario実装選択A→Bによる旧生成Command失効はPR #5054と既存#2600に委ね、Go修正を重複させない。生成Studioの通常／生成Windows WebView2両Job成功（#5042／#5052）は製品PoCとして採用するが、Stage1→Stage2の実再入（#2831／#4583）とR3のactual Provider（#4495／#1251）は独立P0のままとする。Go focused回帰のCI合否、手書き検査の実削減、原文配布の開示受理を別の検収票で確認する。


<!-- tei-org-c1-model-to-consumer-bridge-20261115 -->
### 組織C1を標準Data・Rule・Node表示へつなぐ一機能縦断（2026年11月15日）

組織モデリングの製品完成は、組織図や二時点Queryが個別に動くことではなく、**一つのAcceptedな所属関係を、記述から保存・読取・判断・表示まで、意味と出典を保って受け渡せること**で判定する。[組織C1補遺§40](../design/ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md)のO-P1〜5、[段階計画§10.15](../design/ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md)を既存の完了判定とし、新しい製品Gateや第二の期待値正本は作らない。

他レーンのmain成果は三つを再利用する。①[PR #5058](https://github.com/hat47x/tei-platform/pull/5058)の印付きMarkdown項目定義表→既存Data/InspectorとCLI結果の照合、②[PR #5055](https://github.com/hat47x/tei-platform/pull/5055)のREA端点／組織二時点／Node Flowを結ぶ際のMeaning受理境界、③[PR #5061](https://github.com/hat47x/tei-platform/pull/5061)のPlugin接続におけるShape／Law／Binding／Realization／Admission／Consumerの別資格である。これらの成功を組織C1の実Provider・実Rule・Studio E2Eへ足し合わせない。

| 受入単位 | 次の実装・独立Expected | 明示する未接続と担当 |
| --- | --- | --- |
| **ORG-W1 記述と実記録を分ける** | 異動申請TransferRequestの印付き**Data定義表**は既存ReaderでData／Attributeとしてinspectし、型・必須・Source原byte位置を従来Dataと照合する。Membershipの**実所属行**は正式Providerから取得し、定義表の1行を所属データとして取り込まない | #2621/#988のReader・Source、#2386の業務Meaning。Markdown定義が未対応のRelation・複合Key・履歴版・Propertyを推測して埋めない |
| **ORG-W2 二時点の保存と実Read** | Ownerが受理したOccurrence／Assertion revision／Person Linkを正式Persistenceへ反映。validAt・knownAtを宣言したQueryから、同一A2の旧新二版をProviderが実取得する。SQLバインドと非表示allocation_minの依存も固定 | #2386/#1126→#2622/#2586。隔離PGの試験用版と正式Sourceは別。保存時の二軸競合排他は[PR #5063](https://github.com/hat47x/tei-platform/pull/5063)の隔離PG run #43724543096で**一接続の正負例が明示PASS**。二接続の同時Writerは[PR #5099](https://github.com/hat47x/tei-platform/pull/5099)の未確定候補であり、正式Source／Migrationへの採用は別 |
| **ORG-W3 必要な取得の閉包とRule** | 同じvalidAt/knownAt・Scopeで、Membership、Person Relation、Ruleが使うRole/Responsibility、非出力Filterを「必要なRead集合」として明示。全部取得／部分取得／0件／競合／開示拒否を分け、人数・所属件数・FTE・承認候補の粒度を混同しない | #4490/#4501/#2510。BRMSのUnique候補は二軸の「同一所属一版」と別のLaw。Source Revision・Rule revision・Read Cohort・Current Grantを独立に保持 |
| **ORG-W4 二Consumerの投影と逆編集** | 同じAccepted World/Read Cohortの所属A2をCLIのTableとStudio/SHOのGraph／Treeに投影し、Occurrence・版・有効／認識時点・Source位置・Edgeの向き／多重性を照合。第二親と非表示の取得依存を失わない | #1124/#2497/#2516。Source PIN→Preview→再Readを経ないGraph端点ドラッグやAI後着でSource／承認者を更新しない。Studio実WebView2受領は別 |
| **ORG-W5 Plugin・Hostへの公開** | 別Consumer/Pluginで同じShapeが使えても、業務Lawの独立Expected、現在のImplementation/Provider Binding、生成・実Call、公開Slice、Host Guard/Effectをそれぞれ照合する | #951/#1251/#2499/#1174/#5061。所属＝Account/Tenant Grantではなく、Plugin配布＝実Composition／実Invokeではない |

**最小シナリオ**は合成org-synthetic-r7のP2・A2・A3をOwner採否待ちのReferenceとして使う。同じP2が二つのMembershipを持つ時点では「所属件数2／distinct Person1」という粒度の違いを保持し、A2の認識後の訂正で同じvalidAtに対する結果が変わることを観察する。**r7（known境界2027-07-02）と隔離PG候補（2027-07-03）は異なるFixture**であり、同じ正式Worldで実証したことにしない。別Source・別Reader・別Providerの結果は同じ時間ラベルや画面上のNode名が一致しても自動Joinしない。

**Sourceの採否・実装の段階**：まずORG-W1を既存CLI/Inspectorで反証し、ORG-W2は意味Owner判断とProviderの二版保存を待つ。W3は必要Readの全件性を少なくとも一つの負例で示し、W4はRead Cohortの一致が保証できた場合だけ二Consumerへ進め、W5は既存Host・Plugin実CallのOwnerに委ねる。未完了のLinkを説明用Graphで補わず、標準Data/Relation・Query・Rule・Representationの既存層を再利用する。通常CLI実Provider #2586とStudio Stage1→Stage2 #2831は引き続き並列P0であり、相互をHard Gateにしない。

**状態**：これは既存mainの限定成果と、組織C1固有の欠落を結ぶ受入計画である。正式Membership履歴保存、Person Link、製品Read Cohort／二Consumer、BRMSでの実判断、Plugin/Host・Studio E2E、意味Ownerの採用はこの追記だけで成立しない。


<!-- org-matrix-parent-cardinality-realization-20261115 -->
#### ORG-W4a：関係の多重性と、保存・読取で実現できる多重性を照合する

現行[組織Source](../../reference/org-staged-axis/semantic/organization-axes/model.tei)には、`OrgUnit.matrixParent` が **`many` のRelation**として宣言されている。一方、対応するKeyは単一項目`matrixParentId`から`OrgUnit.orgUnitId`への参照であり、現行の`org-staged-axis`にOrgUnit用の正式Persistenceや複数Edgeを実Providerで取得するQueryはない。**多重性を宣言できることと、同じOrgUnitから異なる二つの上位OrgUnitへの関係を保存・再読込できることは別**である。

| 確認する経路 | 独立Expectedと停止線 |
| --- | --- |
| **MP-1 Source／Meaning** | `supervisor`（one）と`matrixParent`（many）の向き・Relation ID・適用時点・業務意味を別に保持する。二つの親が存在する場合も、表示順で本務・指揮系統を決めない |
| **MP-2 保存／Provider** | 同じOrgUnit UからP1・P2の二本の`matrixParent` Relation occurrenceを、別IDとして保存・実Readできるかを正式Relation Storage／中間表／Providerが証明する。単一の`matrixParentId`スカラー、同じUの履歴版二行、`many`という構文宣言だけではこの正例を満たさない |
| **MP-3 標準Graph／Node** | P1・P2への両有向Edgeを別のEdge ID・元Source・Scopeで保持し、同名Node／同ラベルEdgeを潰さない。必要な辺がpartialなら「親は一つだけ」と判定しない |
| **MP-4 Tree／Studio／Rule** | 単一親Treeに投影するとき、省略した第二のEdgeと不可逆性を明示。配置上の主親は表示選択であり、BRMSの唯一承認者・現在Grantの根拠にはならない |

最初の実装は、**一つの正式Relation、二つのEdge occurrence、同じvalidAt/knownAtでの実Provider二件取得**を、Source／Persistence／Query／Graphで独立Expectedに照合すること。これをMembershipの旧新版（二行であって別Edgeではない）や、Personの兼務（別Membership occurrence）と混同しない。#2386/#1126が多重性とRelation保存方式の意味を受理し、#1124/#2497/#2622が実取得と表示上の保持を担当する。Graphの宣言だけから新しいJoin機構、Graph DB、組織専用Authorizationを導入しない。

**現況**：現行Sourceの宣言とKeyの形、OrgUnitの製品Persistence未接続を静的に照合した設計GAP。複数親の実Provider取得、編集往復、Studio／BRMS製品受入は**NOT RUN**。正式所属履歴の二Writer候補#5099とは、対象Relationも試験の合否も独立して扱う。

<!-- tei-product-self-described-validation-data-reader-20261115 -->
### TEI自身の検査規則をDataで記述する製品Reader実験（2026年11月15日）

[自己記述設計§52](../design/ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)の小さな縦断を追加する。組込みtei.authoring/v1の**印付き項目定義表**を使い、検査規則の識別子・種類・対象属性・任意の関係という**定義情報の形**を既存semantic.Dataへ読み込む。実Workspace.Load／型付きData／既存Validator／元byteのSource Indexを同じFixtureで照合し、同名のsemantic-validationや公開Capabilityを自動生成しないことを負例にする。追加Goはinternal/workspace/tei_md_self_description_test.go。これは新Grammarやメタ専用Runtimeを要しない自己記述D1と通常Data Readerの実消費であり、positive Ruleに対するM1契約解釈ではない。

**特に禁止する誤解**：Data表のoptionalはforbiddenを意味しない。relationIdがrequired=noでも「positiveではRelationを指定できない」とは検査できない。最初のM1/M2は、既存のpositiveの固定独立Expectedに対して、未表現の義務を明示したうえで一条件の検査コードをTEIから実生成し、手書きGoの重複分岐を消すところへ進める。元MarkdownのGo embed（#5064以前の§51参照）も原文とGeneratorの意味適合、Disclosure、R3／Stage2を証明しない。CI合格は本テストのexact HEADで確認する。#2620/#988/#1018/#1121/#1180/#2514が担当し、通常CLI実ProviderとWindows Stage1→Stage2のP0を維持する。

<!-- tei-product-self-description-abstract-path-witness-20261115 -->
### 自己記述の実装義務を、実現方式をまたぐ抽象経路で検収する（2026年11月15日）

[自己記述設計§53](../design/ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)では、テストをGoメソッド名へ紐付けず、**正規Subject／実装義務／前提／要求経路／不変条件／独立Expected／実Attempt**の組で対象を特定する。以下は独立Expected #1180／#2514への実装引継ぎであり、新しいTest DSL・Oracle・Trace Storeは設けない。

| 先に検証するもの | 実装Packetと合格の境界 | 既存Owner |
| --- | --- | --- |
| **AW-0：二方式の局所対照** | 既存quantity_positiveを直接Evaluatorと生成Go Binaryで同じ入力／Source／Read一回／NULL・未取得の条件で比較。既存テストの成功と、静的な経路の存在・実際の経由先・状態保存を別々に表示する | #2620／#1121／#1180／#2514 |
| **AW-1：選択と実経路の不一致** | Gを選択したのにPが呼ばれた、別Builder byteを使った、同じ結果を返す別Operatorを呼んだ反例を作る。メソッド名変更だけで業務Subjectは変えないが、実Entry／Export契約の破損は検出する | #951／#2595／#2600／#4544 |
| **AW-2：型付きPlugin Mappingの保存** | PR #5101のWhole／Field／明示Adapterを既存標準型とMappingからread-onlyで比較。必須欠落・二重Binding・秘密Fieldの漏洩・粒度／単位変換のLossを拒否。Node表示の折畳みはSource変更にしない | #1223／#2497／#951／#1180 |
| **AW-3：TEI自身の検査義務へ再入** | 自己記述Data FixtureのGo実証PR #5097は独立検収後に利用し、Relation optionalと禁止を区別。契約から検査一義務を実現し、該当Go手書き分岐の削除と実Callを確認して初めてM2。正式R2-P元Source閉包／R3／Studio Stage2は別資格 | #2620／#988／#4544／#4495／#2831／#4583 |

**Source所有の原則**：DSL正本、Studio／SHOのDesigner投影、Resource、Generator定義、Generated Source、Custom Hook、Build Artifactを役割で区別する。Visual StudioのDesigner・Resource分離は参考とするが、TEIの新しい拡張子をこの段階では採用しない。生成物を手編集して正本化せず、案件固有の処理は既存Hook／Plugin契約で接続する。

**現在の資格**：PR #5073／#5087のP/G比較と#5101のMapping整理を基線にした設計・試験引継ぎのみ。PR #5097は確認時点で未マージであるため、自己記述Fixtureの実行PASSをmainの成立条件へ算入しない。AW-0〜3の新しい抽象経路テスト、自己記述M2処理置換、正式R2-P、実Provider、Studio Stage1→Stage2をこの文書で検収済みとは扱わない。

<!-- mvp-dogfood-decision-transfer-20261116 -->
### MVP dogfoodingの次の実装：検査一条件の移管を、次世代での実使用まで追う（2026年11月16日）

[判断移管の実行パケット](MVP_DOGFOOD_DECISION_TRANSFER_2026-11.md)で、既存I2M §15のM0〜M3と自己記述§53のAW-0〜3を、**実装義務・対象Source・選択経路・同一Attemptの実使用・独立Expected**で接続した。新しいRuntime・Test Registry・Reader・Host・Build Engineは追加しない。

最初に移管するのは現行`validationgen.GeneratePositive`のRequired Shape（Rule ID非空・Attribute非空・Relation禁止）に含まれる**実行可能な一条件**である。Data表に現れた`required=no`はRelation禁止を表せないため、記述だけでM1合格としない。まず現行Goの固定Shape基線を保持してTEI由来の判定を実行するD-1（M1）、その後に当該手書きGo条件を製品実経路から削るD-2（M2）を**別PR**で行う。続いて原Markdown／Reader／Generator／生成byte／実Build入力を同一Attemptで接続するD-3（R2-P）、Stage1実TEIがStage2を作って同検査を実使用するD-4（M3）へ進む。

**変更反証の重点**は、選択G→実P、生成B→配置A、Plan解決後のbyte変異（main統合済みPR #5109）、Source A→B→A、説明変更による出典失効、型付きPortの必須欠落／Loss、実Binary成功と独立Consumer未受領の区別である。既存#1180／#2514をExpectedの正本とし、別HEADの試験成功を移管後の資格として再利用しない。

**MVPの評価値**は、三つの原子的Shape条件のうちTEIが実際に担った数、同義の手書きGo条件の削除数、TEI定義とGenerator／Hookの保守変更量、診断→生成→実Build→Consumerの所要時間とする。生成行数や図面数だけで自己構築率を増やさない。正式Provider Call #2586と非Plugin Windows Studio Stage1→Stage2 #2831／#4583は引き続き**並行P0**とし、D-1〜D-2を両P0の前提Gateへ変更しない。

**現時点の資格**：この追補は計画と受入条件の明確化のみ。D-1〜D-4の新Go製品実装・試験、M2の旧分岐削除、正式R2-P／R3、Studio Stage2の実再入を達成したものではない。

**D-1の実装適格性**：[実行パケット§8](MVP_DOGFOOD_DECISION_TRANSFER_2026-11.md)で、Dataの`required=yes`と元Goの`!= ""`、`required=no`とRelationの`== ""`を区別した。自己記述された項目定義と実ValidationインスタンスのBinding、既存Operatorによる厳密な受理・拒否ができないならM1は`unsupported`とする。元Markdownを正常にLoadできる試験と、不正な型付きValidationを拒否する単体試験も別資格として扱う。

**利用者側の最小dogfooding**：[実行パケット§9](MVP_DOGFOOD_DECISION_TRANSFER_2026-11.md)では、CLI実`tei inspect`と直接`operations.InspectSubject`の同値回帰を基線に、Wails非依存Studio Hostの三入口比較[PR #5119](https://github.com/hat47x/tei-platform/pull/5119)（Draft）を重複実装せず待つ。次はテスト用Ruleの説明変更／参照先変更→再Inspect→生成Go→実Buildという一往復を試す。Sourceの表示・編集Intent・実Provider・Windows Stage2の権限と証拠は混ぜない。

<!-- product-rule-read-eligibility-20261116 -->
### 状態付きRuleを実Readへつなぎ、解釈と生成の対応範囲を誤認しない（2026年11月16日）

[図表実現方式の設計§14](../design/DIAGRAM_TO_REALIZATION_STRATEGY_AND_HOOKS_2026-11.md)と[実証計画DRI-R1〜R5](DIAGRAM_REALIZATION_REFERENCE_EXECUTION_2026-11.md)へ、mainで進んだ値の状態別評価（#5142）、記録例による独立期待との照合（#5143）、暦日の順序Rule（#5146）を接続する。既存のpositive RuleのP／生成G実Binary比較（#5118）を一般的なRule生成へ拡大せず、Ruleの種類・Readの状態／完全性・実Target・実Callerごとに適合を確認する。

**最初に実装する不足辺**は、実Queryが返した値の出典・Presence・Relation Coverageを、既存Function Contextと状態付きRuleへ渡せる範囲の確定である。単に返却値が正しいだけでは足りない。重複のように既知の反例が不成立を確定する場合と、未知の行が残るため成立が未決定の場合を区別する。native SQLの列由来観測を検討した#5144は未マージであり、現行製品のRead Coverageをcompleteへ昇格させる根拠にはしない。期間非重複Ruleの#5148も独立の検証対象として扱う。

**優先P0は維持**する。正式Reader→Builder入力→Artifactの同一Attempt検証、通常CLI→実Provider Read→独立Consumer #2586、Studio Stage1→Stage2 Windows #2831／#4583、および自己記述した検査一条件の手書きGo分岐置換は、それぞれ別の完了条件である。本追補は既存の実装試験に続く作業順を定めたもので、新Go・実DB・Windows受領・Owner意味受理を追加実証したものではない。

### 標準Dataから読取状態と規則の判断を照合する（2026年11月16日）

[標準フロー設計§12.6〜12.8](../design/SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)と[dogfooding§17.5〜17.6](../design/IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md)を受け、データ様式の最初の製品接続を次の順序に絞る。現行mainには値の状態を区別する `Decide` と、印付き `example of` の選択・照合が存在する。これらを別Runnerで作り直さず、**一つの選択済みSource版**から型付きData／Rule／記録例・Source位置を出し、規則の要求と判断できる範囲をread-onlyで示す。

- **DF-P1：型付き入力・理由票**。DPC-09〜11のpresent／NULL／not-fetched、partial／complete、観測済み重複を手書きの独立Expectedと照合する。「未取得だから常に未確定」「行が少ないから必ず確定」のいずれの過剰一般化も拒否する。
- **DF-P2：例・改版票**。DPC-12の `example of` の期待結果と元Source位置を、World意味、実Provider記録、独立Expectedとは別の資格で表示する。例だけを修正してもWorld Fingerprintを意味上改変したとは主張せず、旧検収証拠は現在性を再確認する。
- **DF-P3：異質な規則・Consumer**。DPC-13の暦日／期間と、Read対象ID、失効理由を既存Rule／Source／Projectionへ照合する。実Provider・HTTP公開・更新／Effect、BuilderとStage2は各Ownerで別の実Target検収とする。

既存の #5118（Data表由来G/Pの実Read先ID差分）と #5129（Build Plan後の元Markdown改版拒否）は限定基線として再利用する。並行PR #5097／#5114はclosed/unmergedであり、これらの成功をmain実装に数えない。製品の受入には既存#1180／#2514の独立Expectedとexact HEADのCIを要する。今回の追補は新規Go／DB／Windows E2Eの実行を意味しない。

### 標準Dataの型と取得状態を一試行の判断へ結ぶ（2026年11月16日）

[標準Flow設計§12.9](../design/SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)と[dogfooding§17.7](../design/IMPLEMENTATION_TO_MODEL_DOGFOOD_2026-11.md)に対応する最初の小コード差分を、`internal/operations/inspect_markdown_data_rule_decision_trace_test.go`で扱う。現行のReader／Inspector／Source Indexと状態付きRule評価を再利用し、一つの`Workspace.Load`に対するData・Ruleの意味ID、Read属性ID、Source byte位置、判断／未確定の理由を四入力の独立Expectedへ照合する。

**次の段階**はDB／API等の実Providerから取得したValueState・Relation Coverage・Source occurrence／時点を、選択したOperation・CLI・独立Consumerへ通すことである。テスト用のStatefulSourceや印付き記録例はProvider実行を代用しない。PluginのWhole Record／Field／明示Adapterの接続（PIO-T0/T1）は別の既存Ownerに保持し、表示上の接続から公開GrantやEffectを生成しない。

**合格範囲**：このPRは狭いGo回帰と計画の接続である。新たな実DB／Windows／外部HTTP試験、正式R2-P、R3、Studio Stage2を実施したと扱わず、exact HEADのGo CIが確認できるまで新回帰のPASSを主張しない。

### DataのSource位置と、実際に原文を開けることを別に検収する（2026年11月16日）

[標準Flow§12.10](../design/SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)に沿い、DF-P1の一Load検収の後に、SL-1の相対Path・Source byte位置・省略件数表示を確認する。次にDesktop既存OwnerのSL-2でHostがRoot内の原Sourceを再読取し、改版・対象Occurrence・参照の現在性を確かめる。SL-3の編集・Windows実受領はその先で別途検証する。DPC-14のGo回帰は原文の実Openを試していない。Plugin出力のWhole/Field Projectionやread-only Source表示から、公開権限・更新Grantを推定しない。

### 実SQL Readerが返した範囲を、Ruleへ渡す前に検収する（2026年11月17日）

[標準フロー設計§12.11〜12.13](../design/SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)と[DB適合計画§7](../design/DATA_PROVIDER_CROSS_DB_CONFORMANCE_2026-11.md)に従う。main済み#5149のSQLite SQL Executor→`validationread.FromHeaderDetail`→`Decide`と、main済み#5179の選択済みMarkdown Data／Rule→Inspector→`Decide`を、**同じ意味・実取得・Consumerの間に残る不足証拠**として整理する。

- **DF-P4a：取得範囲の根拠**。DPC-16。実際に4行あるRelationの2行だけが返るSQLから、`CoverageComplete`を無根拠に申告しないことを、既存Query／Function Data Contextを持つCallerの負例で固定する。現行`Detail`が申告を信じて`satisfied`を返し得る事実を、誤った完了根拠としない。
- **DF-P4b：値・列の状態**。DPC-17／18。SQL NULLとSELECT未取得の違い、`std.decimal`へREALを渡す精度不適合を、実SQL列MappingとRuleの同じAttribute IDで確認する。Decimal・暦日などのRuleが扱える型を、全ての実現方式で無条件に利用できるとはしない。
- **DF-P4c：投影と公開の境界**。DPC-19。Whole Record／明示Field／任意Fieldを別契約とし、状態を失うProjectionを`lossy`と示す。公開・Host Grant、Editor Open、Provider実Callは投影成功だけでは認定しない。
- **DF-P4d：Relation判断とConsumer**。DPC-20。見えた重複・期間重複による不成立と、部分取得で違反を見ていない場合の判定保留を分ける。その後、通常CLIの実Provider→独立Consumerへ、実Read対象・取得範囲・Rule結果／理由を一つの試行で渡す。

最初のコードPRはDF-P4aの一因子に限定する。既存の`validationread/sqlite_path_test.go`は誤申告の危険性を示しているが、**製品Callerが虚偽の`complete`を拒否したという証拠ではない**。必要な契約が既存Query／Data Contextで説明できない場合は、欠落条件と代案を受入票に残し、汎用Query Engineや新Stateful Portを先行追加しない。

今回の計画追補だけで新しいGo／DB／Windows E2E、Owner意味受理、R2-P、R3、Stage1→Stage2をPASSとは扱わない。現在mainの限定成果と、審査中PR #5187（条件付き必須）・#5214（任意Field状態損失）・#5213（研究計画）を区別する。独立Expectedは従来どおり#1180／#2514へ引き渡す。

### 全件性の申告を根拠付きで受理する最小パケット（2026年11月17日）

[標準Flow§12.14](../design/SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)のDPC-16a〜dで、既存DF-P4aの最初の実装境界を具体化する。①限定SQLite Fixtureのfirst_page（2/4行）の虚偽completeを製品Callerで拒否、②同じ部分観測で重複違反の確定と一意性成立の保留を区別、③Scope／Binding不一致と0件の誤全件主張を拒否、④Query／Context／実Callの同一Attempt証拠を通常CLI・実Providerへ接続する。

現行validationread.FromHeaderDetailはCallerからのCoverage申告を利用するReaderであり、それ自体で取得完全性を証明しない。既存Coverageの二値は維持し、未立証は検収票でunknown、Rule入力では保守的にpartialとする。単にLIMITがない、結果が0件、ページ終端らしく見えることをcompleteの根拠にしない。新しい汎用Query Resolver、独立Coverage Engine、第二Expected正本やPlugin ABIを先行追加しない。

**今回の追補はdocs-only**。DPC-16a〜dのGo／DB／CLI実行、#2586の実PostgreSQLと独立Consumer、R2-P実Builder、Studio Stage2、Owner意味受理は未実施の独立合格条件とする。

### Rule正規形と標準表現型を、取得の根拠を失わず接続する（2026年11月17日）

[標準Flow§12.15](../design/SERVICE_FLOW_STANDARD_COMPOSITION_2026-11.md)を、[Rule正規形の決定](../design/RULE_NORMAL_FORM_AND_OPERATOR_SIGNATURE_DECISION_2026-11.md)と[三要素モデルの図面契約](../design/THREE_ELEMENT_METHOD_DIAGRAM_CONTRACT_2026-10.md)に接続する。Operator署名が要求するRelation／入力範囲、Contract Query束縛の供給宣言、実Providerの取得、Rule判断、表・Graph等へのProjection、Consumerの受領は、**同じ意味IDに基づくが互いに独立して検収する**。全件宣言をQueryのSQL実行の正しさと混同しない。

既存DF-P4の作業順を維持し、(1) DPC-16aの虚偽completeを製品Callerで拒否する一因子Go回帰、(2) Operatorの署名・選択Query束縛から必要Relation／Scopeを確認（新しいRule正規形の実装待ちの場合は既存Rule IDで同等条件を固定）、(3) NULL／未取得、部分観測の陽性違反／成立保留を実Readから検証、(4) 表・Graphのread-only投影でIdentity／Relation・状態・Source版を損なわないことを確認、(5) 実PostgreSQL→通常CLI→独立Consumerを同一Attemptで受領、の順に検収する。

自己記述のSDM-01〜12、Graphの同名異ID・並列辺、組織C1の同一MembershipからPerson数を一意に決められない反例は、新しいFixture Storeを作らず既存#1124/#1131/#2497/#2514に引き渡す。Source編集往復とStudio Stage1→Stage2は別P0として継続し、Data／Ruleの一局所試験を全Modeler・製品セルフビルドの完成とは呼ばない。**ここで追加したのは設計と受入順であり、Operator正規形移行・DPC-16 Caller実装・実GUI・実Consumerの新規PASSではない。**
