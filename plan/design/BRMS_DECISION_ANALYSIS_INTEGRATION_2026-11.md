# TEIのビジネスルール分析・決定表統合（設計候補）

- 状態：**未採用の横断設計・検証計画**。Graph／一行RuleからのGo生成・Pinned Source予備検証は限定Referenceとしてmainに存在するが、一般Decision Analysisおよび正式製品接続は未実装・未受理
- 役割：BRMSの有用な機能を、TEI既存のData Script／Validation／Derivation／Operator／Function／Graph／コード生成へ重ねる横断案。新しい製品要件、Gate、Runtime、権限、Canonical schemaを採用する文書ではない。
- 対応先：#988（DSL・Operator）、#1637（導出）、#1129（効果規則）、#1180（テスト）、#2495/#2497（構造と表現の意味保持）、#2831（self-build）、#4495（製品Function Entry）。変更と受入は各Ownerの既存経路に返す。
- 上位の現在主線：TEI CLIを起点に、*.tei.md の業務定義から実コードを生成して動作を確認し、Pluginにも適用する。Studio Stage1→Stage2、通常teiからの実Provider呼出し・独立Consumerの検収を妨げない。
- 参照先：[導出契約](DATA_DERIVATION_REFERENCE_CONTRACT.md)、[効果規則](EFFECT_RULE_AUTHORING_DESIGN_2026-11.md)、[Operator合成](DSL_OPERATOR_PLUGIN_COMPOSITION_DESIGN.md)、[意味保存とMarkdown](SEMANTIC_STRUCTURE_MAPPING_AND_DECLARATIVE_SYNTAX_VIEW_2026-11.md)、[Script起動](SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md)、[業務設計の品質](BUSINESS_PROCEDURE_DESIGN_QUALITY_MODEL_2026-09.md)。

## 1. 結論：独立BRMSではなく、業務モデルに近接した分析機能

TEIでは標準データ型・標準表現型と型付き意味モデルからコードを生成する。このため、BRMSの条件評価や分岐実行だけを再実装する利点は小さい。追加価値の中心は、決定表の競合・網羅性・到達不能、ルール依存関係、判断理由、変更影響、ルール単位の回帰検証である。

Ruleをすべて横断する新しい実体型・汎用テーブルにせず、既存Data／Function／Business／Validation／Derivation／状態遷移／効果規則に付いた**判断や制約の参照可能な面**として扱う。Rule IDはSource上の同一性とテスト・診断のために利用し、所有者や認可、確定の正本を奪わない。

## 2. 意味上の分類と責任の境界

| 種類 | 既存の主な接続先 | 評価結果 | 意図せず起こしてはいけないこと |
| --- | --- | --- | --- |
| Constraint | Data / Validation | 適合・不適合・評価不能 | 不適合を自動修復・保存する |
| Derivation | Data / Function・Operator | 型付き値・NULL等 | 計算結果を暗黙に保存する |
| Decision | Function / Business / Data Script IF | 意思決定の値・複数値・未決 | 条件一致を実行許可に昇格する |
| Transition guard | Lifecycle / State | 遷移条件の評価 | 条件評価だけで遷移を確定する |
| Effect condition | Capability / Effect Rule（提案） | 効果を起こす条件の評価 | Provider実行・副作用を暗黙に行う |
| Authorization-related policy | 既存Host・AuthZ境界 | Policy上の判断候補 | 現時点のHost Grantを代替する |

Decisionが選択した「注文可」は、受理済みの業務判断であっても、実Provider操作の権限付与・DB commit・外部効果の成功ではない。候補・Accepted Meaning・Generated Artifact・実際の観測を分ける。

## 3. 書式はIFを基礎とし、決定表を糖衣表現として追加する

原則はData ScriptのIFと既存の型付きOperatorである。分岐数が少なければIFで十分。多条件・多数行・業務担当者による点検が必要な場合は、Markdownの表（大きいものはCSV/TSV）を選ぶ。通常Markdownの可読性を維持し、Mermaidは定義元か導出図かを図ごとに区別する。

次の表は**業務規則の例示であり、TEIに実装済みの構文ではない**。

対象：会員割引の判定。入力は会員区分（standard/premium）と購入額（JPY、0以上の十進数）。出力は割引率（%）。一致方針は unique。一致しなかった場合の扱いは別に明示する。

| Rule ID | 会員区分 | 購入額（JPY） | 割引率 |
| --- | --- | --- | --- |
| D01 | standard | 0以上・10000未満 | 0 |
| D02 | standard | 10000以上 | 5 |
| D03 | premium | 0以上 | 10 |

これは表示例であり、「0以上・10000未満」を受け入れるParser、列見出しのInput/OutputへのBinding、型・単位の宣言方法、行IDの安定化は既存Language Plugin／Operatorとの実証で決める。翻訳された見出しから型やRelationを推測確定しない。IFの意味が正本の場合は表を導出Viewにする。表がSourceの場合は表の行を正本にし、IFの生成物を並行して編集させない。

表の意味には適用範囲、入力と出力、行ID、一致方針、明示したdefaultまたはno-matchの扱いを関連づける。ただし独自DSLの構文や新規永続属性を先に固定しない。

## 4. 必要な一致方針は段階的に導入する

最初の縦断は**unique**のみを選ぶ。複数行一致は競合として診断し、一致なしは「許可」「false」「NULL」へ暗黙変換しない。必要に応じて後続で any（重複一致しても出力が同じ）、collect（複数結果を型付き集合にする）を追加する。

first（表の記載順）やpriority（明示した出力順位）は要件上の順序が必要な場合に限り、意味を表に明示して採用する。通常のIFの短絡・順序やEffectの段階を、決定表の一致方針で勝手に変更しない。Drools型のsalience／Agendaを標準にしない。

未取得・明示NULL・型不適合・空集合・失効した取得結果・No Match・複数Matchを区別する。入力が未取得でも、その入力を必要としない条件の結果は決められる場合があるため、無条件の全体停止ではなく、短絡と必要Read-setに従って評価可能性を判定する。

## 5. Decision Analysisが提供すべき検証

| 分析 | 最小の観測と停止条件 |
| --- | --- |
| Overlap | uniqueの複数行一致を具体例付きで報告。明示的な優先順位のない規則を行順で裁定しない |
| Gap | Enum・有限範囲など証明可能な入力領域の未網羅部分を示す。無限領域での未証明を「網羅済み」としない |
| Shadow / redundancy | 到達不能行、同一出力を重複する行、より広い条件に覆われる行を区別 |
| Dependency / cycle | 参照したField・Relation・Operator・他Decisionの有向グラフを作り、循環・未解決・粒度/時点/単位の不整合を示す |
| Effect boundary | pureな判断から保存・通知・他Ownerへの更新が漏れ出していないかを既存Effect検査と照合 |
| Change impact | Source・意味Revision・Operator版・Target生成器版の変更を区別し、影響するDecision／Function／テスト／生成成果物を示す |
| Explain | 判定入力（秘匿・開示に配慮）、利用したRule ID、結果、必要な根拠と未取得入力、Source位置、版を再構成できる |

最初は静的解析可能なEnum／数値区間の交差・包含と、固定Fixtureを使った実行時照合を優先する。一般式・正規表現・任意SQLの完全な充足可能性判定やSMTの全面導入は後回し。解析できなければ Unsupported / Unproven と明示する。

## 6. 説明・来歴・版

説明は診断やテストへ必要な最小情報を生成・保存できればよく、全実行の全入力値を強制記録しない。Source file、元byte span、安定した行ID、受理済み意味参照、Operator契約版、生成器版、World／Model／ArtifactのRevisionを区別する。実行時のRule traceには、公開可能な入力、実際の選択行、結果、利用したRead-setの状態を選択的に残す。

重要なのは「なぜこの結果か」と「なぜ別の行ではないか」を人間／AIの双方が辿れることである。AIが説明文や例外候補を生成しても、それはAccepted Meaningや実行Authorityにはならない。過去版と現行版の差分は、同じ入力集合で再評価した観測と、入力Data／依存Operator／Host Grantの変化を分ける。

有効期間とRule Packの独立配布は後続候補。期間規則を導入する際は業務適用日・記録時刻・取得時刻・評価時刻を混同せず、最新表への差替えだけで過去案件を再判断しない。まずはビルド済みArtifactと受理された意味Revisionの組で再現可能性を確保する。

## 7. 実行形態

既定はSource → Typed Meaning/Operator → 既存Generator → Go（必要な場合SQL/他Target）の直接コード生成。実行時に全Ruleをインタプリタへ送る独立BRMSは作らない。単一バイナリの利点を保ち、データ取得や外部作用は既存Provider／Host／Effect契約を通す。

変更頻度が高い大量のルール、事業部別配布、運用中に差替えたい判断が具体的に確認できた場合に限り、受理済みの型付き表をVersion固定で実行する軽量Evaluator／Providerを比較対象にする。これは意味・Grant・Target互換性の検査を飛ばせる「ホット更新」ではない。RETE/Phreak・Agenda、DMN/FEELフル実行、OPA/Regoの内蔵は標準機構としては採らず、必要性が実証された時だけPlugin方式で評価する。

## 8. Headless縦断の検証案（既存Ownerへ追加する小さなFixture）

| 対照 | 入力・変更 | 独立Expected |
| --- | --- | --- |
| T1 | standard、9999／10000 JPY | 0%／5%（境界一致） |
| T2 | premium、0 JPY | 10%。不要な他条件が未取得でも必要入力が揃えば決定 |
| T3 | 区分未取得／明示NULL／未知区分／負の購入額 | 未取得・NULL・No Match・Constraint不適合を区別し、暗黙許可しない |
| T4 | D04: standardかつ5000 JPY以上を追加 | uniqueのD02等とのOverlapを検出しSourceの両行へ戻れる |
| T5 | D02の閾値だけ変更 | Read→Meaning→生成Goの実差分→CLI結果→影響テストが一致。参照実装にGoへの委譲しかない状態を合格としない |
| T6 | 同一MeaningをIFと表から作る | 短絡、NULL、精度、結果、説明の同値性を差分検証 |
| T7 | Operator／Generator版・Data取得の鮮度を変更 | 意味評価・Artifact再利用・Evidence資格・現在権限を別々に判定 |
| T8 | Pureな判定結果でEffectを呼ぶ | 判断がtrueでも未認可の書込を起こさない。Receipt不明を成功としない |

T1〜T8は**今後の検証候補**でありPASSの記録ではない。生成Go結果と独立Expectedに加え、可能なら最小のReference Evaluatorまたは異Targetとの一致を比較する。通常tei CLIで先に試し、採用済み意味の判断・製品Providerの実Call・実Consumer受領とは別資格で判定する。

## 9. 既存計画への組込み順

1. **D0／整理**：Validation、Data Script IF、Derivation、Effect Rule、Operatorの現行意味と未実装の境界を棚卸し。重複するRule IRやParserを新設しない。
2. **D1／最小縦断**：一枚のMarkdown表または既存IF定義を既存型付き式へlowerし、Goの実分岐を生成する。CLI入力差分・Overlapping診断・Source rowジャンプ・独立Expectedを一件で確認。初期の一致方針はunique。
3. **D2／解析と説明**：有限領域のGap・Shadow、依存Graph、局所再検証、理由・版・Read-setの最小出力。既存Source Index・Revision・Evidenceへ接続。
4. **D3／Studio**：表、IF、Graph、説明Viewを同じ意味モデルから投影し、一意に特定したSource slotだけ局所Patchする。写真／会議テキスト／CSVからの抽出はAI候補としてReviewし、受理前に生成対象へ昇格させない。
5. **D4／選択的な高度化**：明示的な業務要求がある場合だけ、DMN入出力、別配布Rule Pack、時点別Decision、動的Evaluator、RETE／OPA相当のPluginを比較する。

D1はセルフビルドに有益な**第二の意味あるコード生成例**として使える。単一IFのGoへの委譲ではなく、表の変更が生成された分岐・診断・テスト結果を変えることが合格条件。新しい横断Gateは設けず、#1180/#2514/#2831/#4495等の既存Ownerと独立Expectedに接続する。

## 10. 外部BRMSから参照する点

- [OMG DMN 1.6（2026年9月正式版）](https://www.omg.org/spec/DMN/1.6/)：Decision Requirements Graph、決定表のhit policy、DecisionとProcessの分離を参考にする。2026年11月時点で1.7はbeta。TEIのDSLをDMN XML／FEELへ全面置換しない。
- [Apache KIE / DMN decision table](https://kie.apache.org/docs/10.0.x/kogito/)：unique/any/priority/first/collect等の意味の比較に利用する。TEIの初期実装ではuniqueのみとする。
- [Drools rule engine](https://docs.drools.org/latest/drools-docs/drools/rule-engine/index.html)：Working Memory／Agenda／salienceが必要になるのは、規則の結果が新しいFactを生み再評価が連鎖する場合。TEIの標準的なFunction/Decisionに必須とはしない。
- [Open Policy Agentの組込み方式](https://www.openpolicyagent.org/docs/integration)：Policy管理と判断呼出しの分離、Go・Wasm・RESTの実現方式の比較に使う。OPAの判断をHost GrantやTEI Meaning Acceptanceと混同しない。

## 11. 採用前に残す論点

表のInput/Outputと既存OperatorへのBind、列名の多言語表示、No MatchとErrorの契約、導出DecisionのID・出所保持、IF→表の可逆性、Target別生成の同値範囲、顧客固有の有効期間、実Provider Readの鮮度、AI候補の意味受理は未確定。設計候補の記載やReference試験だけで製品対応・Owner受理とはしない。


## 12. 他領域との整合：Ruleを新しい中心概念にしない

他の設計領域との照合から、ここでいう「業務ルール」は一種類の実行オブジェクトではなく、業務上の意味に対する問いの切り口と捉える。「割引率の選択」「売上金額の導出」「締め済み伝票の更新禁止」「組織上の承認条件」「在庫引当ての起動条件」は、見かけ上はすべてルールでも、意味を所有する場所と正本は異なる。

| 連携する領域 | 再利用する意味と責任 | 今回追加する価値 |
| --- | --- | --- |
| Everything is Data／標準構造 | Scalar・Table・Graph・Temporal等、RoleとOperator、Identity・Multiplicity・Presence | IFと表とグラフの意味の保存・照合 |
| DSL自己記述／Markdown | 少数条件はIF、多条件はMarkdown表、大表はCSV／TSV | 表の列の型付きBinding、行IDから原文への逆追跡 |
| Function Data Context／DB | 明示したRead FootprintとClaim Closure、実Providerの取得範囲 | 判定に必要なReadの十分性と部分取得の診断 |
| REA／三要素・Business | Resource／Event／Agent、責任・行為・物理的結果 | 判断とBusiness Action、事実の観測を混同しない |
| 組織モデリング | 組織関係・役割・委譲・時点、保護情報 | ルールの適用範囲と委任期間を検証 |
| Lifecycle／Effect | Guard、Capability、Effect・Receipt・補償 | 条件成立と業務効果の確定を分離 |
| Studio／SHO／Harness | 固定Subject・Source span・受理済み意味・局所Patch | 説明文・図・表・生成コードの根拠対応 |
| Build／Self-build | World／Model／Artifact Revision、Operator・Generator・Plugin版 | 意味変更と表示変更の再検証範囲を区別 |

各領域の意味上の正本は既存文書に維持し、本稿はRule Registryや別の汎用保存表を設けない。

関連文書：[明示Function Data Mapping](EXPLICIT_FUNCTION_DATA_MAPPING_AND_STANDARD_ROLES_2026-11.md)、[REA候補取込](REA_THREE_ELEMENT_CANDIDATE_INTAKE_2026-11.md)、[プログラム実行モデル](PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)、[組織データ保護](ORGANIZATION_DATA_PROTECTION_BOUNDARY_2026-11.md)、[期間状態](PERIOD_STATE_SEMANTICS_DESIGN_2026-11.md)、[横断Consumer受入](CONSUMER_QUALIFIED_CROSS_LANE_ACCEPTANCE_2026-11-14.md)。

## 13. 判断の最小意味契約：問いと入力の資格を固定する

決定表を解析するには行と列だけでは足りない。以下は既存定義から参照で組み立てる検収票の観点であり、新Canonical Schemaの定義ではない。

| 観点 | 検収する内容 | 不合格例 |
| --- | --- | --- |
| Decision Subject | 誰の何の判断か、Function／Business上の正確なIdentity | 同名の別判断を同一化 |
| Input | FieldのSemantic Ref、型・単位、Relation Path、粒度 | 注文金額と明細金額を同一視 |
| Scope | 組織・顧客・Tenant・対象集合・期間と前提 | 部門限定ルールを全社に適用 |
| Time | 業務適用日、参照時点、記録時刻、適用Rule版 | 現行ルールで確定済み過去案件を暗黙再評価 |
| Condition | 型付きOperator、NULL・欠損・短絡の意味 | SQL UNKNOWNを無言でfalseに変換 |
| Selection | Unique等、No Match、競合・例外の意味 | 行順による暗黙の優先 |
| Output | 型付き判断値と結果の意味。Effectは含めない | 承認可を保存済みと記録 |
| Provenance | Rule ID、元Source位置、Operator／Generator版 | 表の並べ替えで業務意味IDを変える |
| Data Completeness | 必要なField・Relation・時点と実取得範囲 | 一部取得を全件として不存在判定 |

意味の正本がIFならMarkdown表は導出View、Markdown表が正式Sourceなら生成IFは導出物とする。IFと表の二重編集を許さない。Mermaidの配置・表示順も優先順位ではない。まず既存の型付きExpressionとFunction/Data参照だけでどこまで意味を保てるか実証し、失情報が示されるまで新IRやGrammarを追加しない。

## 14. 評価結果と判断不能を混同しない

Decisionの評価では、値が確定した場合と決められなかった理由を分離する。以下は説明とテストの区分であり、全システム共通の永続Enumではない。

| 状況 | 判明したこと | 禁止する推測 |
| --- | --- | --- |
| Matched | 必要な入力が充足し、結果が確定 | Grant・Commitの成立 |
| No Match | 必要な範囲が揃い、該当規則なしと判明 | false・NULL・拒否への暗黙変換 |
| Conflict | Uniqueで複数行一致 | 行番号や取得順による裁定 |
| Insufficient Input | 欠損・未取得・非開示・partial等で未確定 | 該当なしと断定 |
| Invalid Input | 型・単位・精度・制約の違反 | 空文字・ゼロへの補正 |
| Unresolved Meaning | Source・Operator・時点・適用範囲が未確定 | 推測で正式Codegen |
| Evaluation Error | 計算／Providerのエラー | 正常な業務拒否と同一視 |

入力の明示NULL、missing、未取得、not found、空集合、partial、permission deniedも別々に保持する。

短絡評価では、ある入力が未取得でも別条件から結果が確定する場合には不要なReadを強要しない。ただしUniqueで一行が一致しただけでは、他行の一致可能性が残っている場合に確定とはしない。重複や否定・不存在を含むルールでは、Read Footprintを宣言した事実とClaim Closureが成立した証拠を区別する。RLS／LIMIT／SQL方言／精度／Collation／時点の差がある場合、Query pushdownとGo評価が同値だと仮定しない。

## 15. 適用範囲と例外の意味

BRMSに固有の実務価値として、例外規則と一般規則の重なりを検査する。

- 全社規則と特定事業所の規則が双方一致する場合、適用範囲が狭いというだけで例外を自動優先しない。優先関係が業務意味として明示・受理されている場合に限る。
- 上位組織への所属と、下位組織のルールを変更する権限は別物。組織図から承認権限や委任を推測しない。
- 適用対象が異なる場合は重複する条件式でも競合ではない。同一対象に重なる場合でも、Hit Policyによる一致処理と業務上の例外を区別する。
- 規則の改訂が例外を失効させる場合は変更影響として表示し、別OwnerのSourceを書き換えない。

D1の実装対象はUniqueだけに限定する。複雑な例外優先は候補解析までとし、先に数値salienceやルール順で裁定する一般Engineを作らない。

## 16. REA・組織・時点を伴う業務例

### 16.1 組織上の承認額

担当者の所属、委任期間、決裁上限、通貨、申請時点から承認可否を判断できる。ただし「承認可」は業務上の判断値であり、当該担当者の現在のHost Grant、承認操作の実行、外部Effect Receiptを代替しない。組織の異動前後、委任期間外、異なるTenantの反例を用意する。

### 16.2 税率・会計期間

旧年度の注文に適用した税率と今日の再見積りに使う税率は、参照したRule版、有効日、記録時刻、対象会計期間が違うことがある。確定済み伝票の値を現行ルールで自動書換えしない。税額の端数、負数、単位・通貨の演算は標準Operatorの契約が持ち、Target固有の丸めを決定表へ隠さない。

### 16.3 REA・三要素の判断

販売契約が成立すること、販売というEventが実際に起きたこと、在庫Resourceが更新されたこと、業務として完了を受理したことは異なる。現場のホワイトボード写真・会議・CSV・既存帳票から見つかった関係とルールは、原資料に紐付いたCandidateとしてレビューし、正式なData／Function／Business／三要素Relationへ暗黙昇格しない。

## 17. Studio・AIとの共同編集

各種資料から判断の条件・対象・結果・例外・時点を抽出し、出典位置と改訂版を保存する。AIが不足を補った部分は提案と明示する。既存IF・Validation・Derivation・Lifecycle・Effectへ対応づけたうえで、現行Shared Acceptance経路に従って正本へ変更を反映する。

Studioは独立のBRMS管理画面を先に作らない。通常の設計要素を選択し、その周辺に「規則」「適用例」「競合・漏れ」「判断理由」「影響する機能・テスト」を表示するContext Viewとして始める。CLI・AI Harnessには固定Subject、Source occurrence、Rule ID、依存範囲、現在Revisionを伝える。

編集する場合は一意のSource Slotを確認し、Markdown表の行移動・見出し翻訳・Graphのレイアウト変更を意味変更へ自動昇格させない。文章による説明は受理済みDSL・表の内容からほぼ直訳できる程度に、入力・条件・結果が業務水準で記述されていることを目標とする。SHOに設計書を出力する場合も候補と受理済みSourceの区別を維持する。

## 18. 判断根拠と変更差分を最小限に保持する

設計時にはSource span、Rule／Function ID、依存するOperatorの契約版、条件の集合関係を保持する。試験時には独立Expected、実際に必要だったRead-set、選択・非選択の理由、生成Artifactを照合する。実行時の診断は必要に応じて選択的に記録し、保護対象Fieldの値を無条件にログへ残さない。

次の変更を区別する。

| 変更の種類 | 原則として再評価する範囲 |
| --- | --- |
| 表の書式・配置だけ | Source位置・Viewの整合。意味は維持され得る |
| 条件・値域・適用対象 | 意味・判断結果・下流Function・Test |
| Data取得元・実取得範囲 | Claim Closure・取得品質・判断可能性 |
| Rule有効期間・Operator意味版 | 時点別評価・受理済み契約・生成物 |
| Generator／Toolchain版 | 意味同じでもArtifact再Buildと検収が必要な場合 |
| Host Grant／Consumer | 現在権限と開示資格。過去の意味受理は権限を与えない |

A→B→AとSourceが戻っても、World Fingerprint／Model Revision／Artifact Revision、現在Grant、過去Evidenceの資格を同一化しない。Graphの同名Node、多重有向Edge、論理Relation ID、描画IDを区別する。Edge反転は意味変更、配置変更は原則として表示変更である。

## 19. 段階的検証に追加する反例

以下のX01〜X10は、既存T1〜T8を補う未実行のExpected候補である。独立Expectedと試験実行結果の正本は#1180／#2514等へ戻す。

| ID | 対象 | 独立Expected／負例 |
| --- | --- | --- |
| X01 | IFとMarkdown表の投影 | Scope・NULL・Hit Policy・出力の意味が一致。二重正本を作らない |
| X02 | 閾値を9999から10000へ変更 | 実Go生成分岐と結果が対応して変わる。既存Go関数への単純委譲では不合格 |
| X03 | 重複行／Gapを追加 | Conflictは両Source行、Gapは具体的な入力反例を示す。証明不能は未証明 |
| X04 | 部分取得・RLS・NULL・No Match | 真の該当なしと情報不足を区別。空集合から全母集団の不存在を証明しない |
| X05 | 所属・委任・時点の変更 | 業務判断と現在Host Grantの区別。Decision=trueだけでは更新なし |
| X06 | 旧版ルールによる過去案件の再現 | Rule版・適用日・入力Data・Artifactを固定して比較。過去記録を上書きしない |
| X07 | 行順・Graphの辺・レイアウト変更 | Uniqueは行順で意味不変。Relationの向き・ID・多重性は保存する |
| X08 | Source／Operator／Generatorの差替え | 意味評価・Build再利用・Evidence・現在Grantを別に判定 |
| X09 | 写真／ExcelからAIが例外を追加 | Candidateは原資料に紐付け、受理前に正式生成物へ混入させない |
| X10 | CLI／Plugin／Provider／Consumer | 実生成コード・実Provider・独立Consumerを別段階で検収。Mock成功の転用禁止 |

## 20. 実装順序と撤回条件

- **D0**：現行Data Script IF／Validation／Derivation／Effect Rule／Operator／Queryの意味と未実装を棚卸し。別のRule ParserやRule Engineを作らない。
- **D1**：小さなMarkdown表またはIFから既存の型付き意味へ接続し、実Goの条件分岐を生成して通常のCLIで実行する。T1〜T4とX01〜X04、競合行のSource位置を検証する。単なるGo Delegateは不合格。
- **D2**：既存Source Index・Graph・Revisionで依存・理由・差分・入力充足を示す。X05〜X08を用い、証明可能と未証明を分ける。
- **D3**：Studio・SHO・Harnessから同じ意味へ到達し、読取専用投影から一意Slotの局所Patchへ進む。X09〜X10を追加する。
- **D4**：具体的な要求と実測が得られた場合のみ、事業部別Rule Pack、有効期間付き動的評価、DMN連携、SQL pushdown、RETE／Phreak・外部Policy Providerを比較する。

決定表を導入しても既存IFより不透明になる、型付きOperatorに意味が降りない、Sourceと生成Artifactが追えない、IFと表の二重正本を避けられない場合は、その抽象を一度撤回して既存の表現に戻す。必要な意味のLossが具体化した範囲だけ再設計する。

全体P0であるTEI CLIからの意味あるGo生成、PluginのTEI-native生成範囲の拡大、Studio Stage1→Stage2セルフビルド、通常製品CLI→実Provider→独立Consumerを維持する。新しいHard Gate／Host／Rule Server／中央結果保存表は先行採用しない。

## 21. 正本と検証状態

本稿の§12〜20は、他領域の既存設計を接続した**未採用の横断設計・未実行の検証計画**である。Rule／Decisionに関する正式なSemantic Role、DSL Grammar、共通API、公開Result Enum、生成Go実動作、SQL・Java同値、Studio編集、Owner意味受理を主張しない。各種の意味・権限・業務効果の判断は、そのSourceとOwnerの既存ADR・Issueを正本とする。


## 22. 実装の再出発点：mainに存在するGraph／Rule→Go参照実証を活用する

2026-11-15時点の現行mainを読み直すと、BRMS周辺の議論で先行実証とみなすべき対象が明確になった。今後は**存在しない新しいParser／Rule Engineを最初から作る計画**ではなく、既存の実コードを限定的な比較基準に固定して、正式なTEI意味・製品経路へ持ち上げる計画とする。

| 確認対象 | mainに存在すること | 現在まだ主張できないこと |
| --- | --- | --- |
| 参照Reader | reference/tei-md-graph-reader/graph.goが限定的なNode／Edge表を読み、重複・曖昧参照・原文位置を検査する（PR #4722統合） | 正式なMarkdown Language Plugin、一般Graph文法 |
| 参照Rule | rule.goが「算定規則の定義」の一行を読み、起点・終点条件を厳格に選択する（PR #4729統合） | 一般の決定表、複数Rule、Unique／Anyの本格的Hit Policy |
| 条件分岐のGo生成 | GenerateGoが実際のe.from／e.to判定を生成し、Go子Processで結果を検証する | 型付きAccepted Meaningからの正式Target Generatorによる実生成 |
| 同じ生成Binaryの動作 | rule_test.goがビルド済みBinaryのSHA-256不変を確かめつつNode IDを変えて複数回呼ぶ | 製品tei CLIのFunction Entry／選択Provider／独立Consumer |
| Source位置の照合 | sourceindex_bridge_test.goでRangeFromOffsetsと元byteのFingerprintを照合する | Source IndexへのSemantic Role／Rule意味Referenceの正式登録 |
| 未接続を明示する境界 | internal/workspace/tei_md_source_guard.goにTEI_MD_PARSER_UNBOUNDの拒否が残る | Workspaceが明示選択した*.tei.mdを製品として読めること |

実証対象の固定例はtestdata/tei-md-graph/baseline.tei.md等の6入力。表示名が同じNode二つ（n-1／n-2）、同じ端点を持つ二つのEdge（e-1／e-2）、辺の反転、Ruleの起点→終点切替、表示図だけの変更を区別する。期待値はindependent-expected.candidate.jsonで**提案用**と明示され、#1180／#2514で正式な独立Expectedを受理した証拠にはならない。

Source Index橋渡し試験の合格は「元byte位置の共有」を示すだけである。NewArtifactは*.tei.mdを現時点では一般のmdとして分類し、接尾辞からRoleを確定しない。Workspace選択・Language Plugin／Role・受理済みMeaning・製品Generator・実Provider・独立Consumerへの接続が次の本当の欠落部分である。

既存の[Graph Reader参照実装（main統合済み）](https://github.com/hat47x/tei-platform/blob/main/reference/tei-md-graph-reader/README.md)、[固定6入力](../../testdata/tei-md-graph/README.md)、[正式Reader接続検討PR #4704](https://github.com/hat47x/tei-platform/pull/4704)、[製品完成計画](../workstreams/PRODUCT_COMPLETION_PLAN.md)を参照する。PR #4704は検討中であり、採用済み製品Readerとはみなさない。

## 23. D1を二つの資格に分解する：限定Referenceと製品への接続

前節までの「D1で決定表→Go」という記述は、現在の参照成果と未達の製品要求を一つの達成表示にまとめないため、次のように分けて運用する。これは新しい全体Gateではなく、既存#2621／#988／#1180／#2514／#4495の受入票を並べる**検査順序**である。

| 縦断点 | 最小の実装差分 | 合格証拠 | 非Claim・棄却条件 |
| --- | --- | --- | --- |
| R0：限定Reference | 一行RuleとGraph候補から実Go制御を生成 | 既存テスト：Rule／Edge変更でGoのif／結果が変化。表示だけならGo不変 | 汎用決定表や正式Meaning／製品Go生成の成立ではない |
| R1：Workspace選択 | Workspaceが明示選択した*.tei.mdとLanguage／Role／Provider選択を束縛 | 対象Source byte・Role・選択理由・固定版を同じ試行で提示。未知Roleは拒否 | 拡張子のみで受理、TEI_MD_PARSER_UNBOUNDの無差別解除 |
| R2：Meaning参照 | 限定Reader候補を既存Role・Typed Expression・Graph Relationへ位置付きで対応させる | n-1／n-2、e-1／e-2の別IdentityとRule条件・対象Scopeを保存 | 表の自然文、Graph表示ID、行順から新しいMeaningを推測 |
| R3：生成物を実消費 | 製品Generatorが選択済みMeaning・Operator契約・Toolchainを実消費しGoを生成する | 元Ruleの変更→実Builder入力→生成条件→ビルドされたArtifactを同一Attemptで照合 | 別のReference Generatorの成功を製品Buildへ転用 |
| R4：実製品入口 | 公開Function Entry・通常tei CLIと独立Consumerが同じ実選択Providerへ到達 | 実ロードArtifact／実Call／型付き結果を独立Expectedと比較 | CLI見かけの成功、Mock、ReferenceExecutorの結果で代替 |
| R5：セルフビルド | Plugin自身のRule／Graph・Operator部分をTEIによって再生成する | Stage1→Stage2の実ビルドと生成意味部分の増加・手書きGo委譲部分の減少を観測 | 偽再入、静的Goデリゲートだけの増加 |

最初の次PRはR1〜R2の**一Subjectだけを確実につなぐ**ことを優先する。R3以降は既存Generator・Product CLI・Plugin受入Ownerと統合し、独立Runtime／Rule Registry／新しい公開CLIコマンドを先に作らない。R5をStudio全体のセルフビルドの代替証拠にしない。

参考：R0の条件切替は技術的なIF生成が可能なことを示すが、まだ一般的なUnique決定表の競合・網羅性を検証してはいない。この区別を設計文書と計画で一貫させる。

## 24. 最初のDecision Analysisは有限で説明可能な領域に限定する

R1〜R3と並行して準備する次のBRMS固有の意味実装は、**有限な区分と区間条件を持つ小さなUnique決定表**とする。最初から推論エンジン・全FEEL式・SMT・RETEを実装しない。

例えば会員区分がstandard／premiumの二値、購入額が0以上のJPY十進数である場合、決定表の三行は、(1)standardで金額が10000未満、(2)standardで10000以上、(3)premiumですべての非負金額、とする。出力は順に0%／5%／10%。ここにstandardかつ5000以上の重複行を加えれば、UniqueのOverlapとなる。

この例が検証する対象は、条件の抽象構文、型・単位、範囲、Scope、一致方針、No Match、Rule ID・元byte位置、生成Goの振る舞いに限る。上記の日本語表現を汎用パーサーが読めることはまだ要求しない。既存型付きExpressionへ手で明示Bindした限定Fixtureから始め、Source側の一般化は必要なLossが確認されてから行う。

| 解析 | 提示する証拠 | 限定すべき主張 |
| --- | --- | --- |
| Overlap | 条件の交差に属する具体的な入力例と双方のRule ID | サンプリング一致は網羅的な競合検出ではない |
| Gap | 明示した入力Domain内で未選択となる具体的な入力例 | 無限・未知Domainでは未証明 |
| Shadow／Redundant | 到達不能の条件・包含関係と、同一結果の冗長条件を分ける | 一致結果が同じだけで行削除を許さない |
| Explain | 当該入力でなぜ選択されたか、他行の排除が証明できるか | 成立した一行だけでUnique確定とはいえない |
| Invariant | 行順・Markdown配置のみの変更では意味／生成結果が不変 | Firstを採る判断では行順は意味に含まれる |

「完全な網羅証明」「未検査」「検査した範囲では反例なし」を区別する。未取得入力が別の行を成立させ得るなら、先に一致した一行があってもUnique判定は未確定とする。否定条件・0件条件については、Read Footprintが要求を示すだけでClaim Closureの証拠にはならない。

## 25. 競合の二種類と、解消の進め方

ルールの「競合」を単なる機械的なエラーとだけ捉えず、次を区別する。

1. **意味の定義として許せない競合**：Unique決定表の重複、型矛盾、循環、同じ業務Subjectへの不整合なBinding。これはSource上の反例として報告し、暗黙の行順・優先順位で実行しない。
2. **業務上の論点としての競合**：組織間の例外、価格政策・安全・納期の優先、異なる関係者の判断基準の相違。Business／SUIで論点や解決候補として整理できるが、その議論をRule実行の自動裁定と同一視しない。

後者では、生成AI・SUIのKJ法・SEIのOntology／文化的文脈による探索も許容する。**一律に人間の判断待ちにしない**。現在受理済みの業務規則と十分なEvidenceから一意に導出できる解決なら、既存の決定的な評価・検証経路で自動処理できる。一方、新しい価値判断・新規の優先順位・権限変更が必要なら、Ownerの既存受理経路を通す。AIの提案だけで実行資格を獲得しない。

この区別により、BRMSのRule ConflictとSUIのDecision Tensionが相互補完できる。ただし新しいDecision Repository／承認サーバーを生やさず、元Source・Issue・Review・Candidateの既存の管理方法を再利用する。

## 26. 変更影響・理由・Target同値の厳密な境界

R0の限定Refで成立した「Mermaid配置だけの変更では生成Go byteが不変」と、製品全体で必要な「Source・Rule・Operator・Builder・Plugin・Consumerが同じ資格で結ばれる」は別の検証である。

変更分類を次のように扱う。

- Source byteのみの変更：元PositionやEvidence再検査の必要性がある。意味・Artifactまで常に失効させるわけではない。
- Rule条件・Relation方向／多重性・時点・Scopeの変更：受理済みMeaningと下流Function／Test／説明を再照合する。
- Generator／Toolchain／Plugin版の変更：Meaningが変わらなくてもBuildの同等性を別に検査する。
- SQL／NoSQL Dataの変更・取得範囲縮小：生成コードが不変でも判断に必要な入力が揃ったとは限らない。
- Host Grant／Audienceの変更：過去の判断結果が同じでも、現在の操作・開示資格は再確認する。

同じモデルをGoとSQLへ展開する場合、両Targetで真偽の扱い・NULL・Decimal／単位・Collation・集合の重複・順序・期間・Read Closureが等しいことを個別の反例で確認する。実装の最適化によるSQL pushdownが追加のRead／Effect権限を作らないことも確認する。Referenceの一致はこの同値性の証明に含めない。

## 27. 次の実装と計画への具体的引渡し

| 実装差分 | 最初の対象／証拠 | 担当する既存Owner |
| --- | --- | --- |
| 1. 正式Source接続の最小一件 | 既存Graph／Rule六入力からbaseline一件を選び、明示選択→Source Index→Role→Meaning参照まで | #2621／#988／#3030／#2495 |
| 2. Target生成での実消費 | Rule条件とEdge向きの変更が実Go分岐・生成物・独立Expectedへ伝わる | #1121／#1180／#2514／#2600 |
| 3. 一枚のUnique決定表 | 2区分×金額区間のOverlap／Gap／No Match／入力不足を反例付き検査 | #988／#1637／#1180／#2514 |
| 4. Product Function／実Provider | 公開pure Function Entry→実Go Provider→通常製品tei CLI→独立Consumer | #4495／#4502／#2586 |
| 5. Ruleと組織・REAの結合 | 実Relation Key・partialと0/1/複数候補を区別し、Rule意味受理と現行Grantは別 | #2506／#3394／#4490 |
| 6. TEI-native PluginとStudio | 手書きDelegateの縮小、Stage1→Stage2、Rule行・生成分岐・説明への同一Subject追跡 | #2831／#4583／#1124 |

実組織の正のRelation候補・partialの10例については[PR #4737](https://github.com/hat47x/tei-platform/pull/4737)にテスト候補がある。ただし検証対象は合成Rowsetであり、承認業務Ruleの受理や実DBの取得完全性を証明しない。BRMS側でこのFixtureを無断に「承認ルールが動いた」証拠へ昇格させない。

まず**1件の実Sourceを正式に解釈し、既存の型付き意味と実Generatorへ橋渡しする**ことが次の価値の高い差分である。決定表の行数やRuleの機能一覧を増やすより、現行製品へ到達した範囲を広げる。対応するSource・実Builder入力・生成Goの分岐・独立Expectedが確認できない場合には、その段階は未達と記録する。

## 28. 現時点で更新する判断と未受理事項

前回の「D1のGo生成は未実装」という一括した言い方は不正確となった。**限定ReferenceのGraph＋一行Rule→実Go生成→別Process実行はmainに存在する**。一方、一般決定表のUnique検査、正式Markdown Language Plugin、受理済みRule Meaning、製品Generator／Go Provider、CLI／Consumer、Studio Self-buildへの接続は未実装または未受理である。

本稿§22〜27の作業内容は、既存の開発計画・Ownerへ加える接続案であって、製品機能の採用・意味受理・新しい全体Gateの導入ではない。Referenceでできたこと、正式Sourceへつないだこと、生成Artifactを使用したこと、製品Consumerが受領したことを、同じ「対応済み」にまとめない。


## 29. 直近の実装提案との役割分担：S0を重複させない（2026-11-15）

[main統合済みの後継PR #4751](https://github.com/hat47x/tei-platform/pull/4751)（旧PR #4745）は、Managed Rootを宣言した一時Workspaceに固定Graph／RuleのMarkdownを置き、既存のManifest解析・Pinned Root・Read・Source Indexと照合する**テスト専用のS0予備検証**である。実Loaderの `TEI_MD_PARSER_UNBOUND` はそのまま拒否し、Sourceの追記後には古いIndexのFingerprintが不一致になることを検査する。実際の製品Frontend選択やLanguage／業務Meaning Roleの受理を行ったものではない。

[PR #4730](https://github.com/hat47x/tei-platform/pull/4730) は製品完成計画でS0→S1→S2を担当する。[PR #4704](https://github.com/hat47x/tei-platform/pull/4704) は正式Markdown Readerの境界を担当する。[PR #4725](https://github.com/hat47x/tei-platform/pull/4725) は限定Readerが長いFence／チルダFence／インデントコード／裸Pipeを誤解釈する負例を扱う。いずれも独立したPRの審査・CI・Owner検収が必要であり、この文書に同内容のReaderやテストを複製しない。

BRMS側の次の作業は、S0の合格後に**固定したRule IDと元条件セルのSource spanを既存の型付きOperator／Function意味へ結び付ける一件**とする。Source Indexの `Role`（原文の位置・参照上の役割）と業務的なMeaning Role（判断の意味）は別に扱う。接尾辞 `*.tei.md`、元byteが一致すること、Referenceに条件が存在することだけを根拠に、後者が採用されたとは判断しない。

[PR #4748](https://github.com/hat47x/tei-platform/pull/4748) はGeneratorの契約上、**生成したGo／TypeScript等が実際に後続のBuild Sourceとして読み取られたか**と、生成物が手書きGoに処理を委譲しただけでないかを区別する。BRMSのRule変更→生成されたif式→実Build Artifact→実Consumerの対応も、既存のGenerator／Build Evidenceを利用して測る。既存BuildGraphEdgeの名前だけで「生成Sourceが実利用された」とは推測せず、第二Build Runnerや別のRule Generatorを追加しない。

| BRMS側で確認する最小例 | 必要な根拠 | 正常に見えても不合格となる例 |
| --- | --- | --- |
| 固定のRule条件「起点→終点」を変更 | 同一元Source・意味参照・実生成Goの比較 | Reference Generatorだけ変わり、製品Builderの入力は旧版 |
| 呼出しNodeを変更 | 同じBinary Digest、異なる引数とResult | 入力変更のたびにGoを再生成して結果を固定 |
| Markdownの図・文章だけを変更 | 意味ID・Dependency Closureの比較と元byteの改訂記録 | byte一致を意味の再受理と混同、説明変更を無視して古いProvenanceを流用 |
| Rule表をFence・引用・未選択Rootに置く | 明示Scope、Parser選択、原文位置、拒否理由 | サンプル中の表を有効な業務ルールへ昇格 |
| 条件に必要なRelationの一部が未取得 | Read Footprintと実ProviderのClaim Closure | 取得できた結果が0件というだけでNo Matchと確定 |

実装の順は「Manifest上の候補を読めた」→「選択Frontendで正式に意味を構成した」→「実Generatorがその意味を使った」→「製品Consumerが実Artifactを使用した」であり、どれも前段の成功だけでは認定しない。次の実装PRにこの五つの例を一括必須とするのではなく、各Ownerが該当段階の独立Expectedへ戻す。

## 30. 最初のUnique決定表に必要な、検証できる判断条件

Hit PolicyをUniqueとしたときは、**値が一つ返ったこと**と、**一致する規則が厳密に一つだと確かめたこと**を区別する。Ruleの定義が正しいか（静的検査）、現在の入力で判定できるか（動的検査）、その判定を操作の許可に使ってよいか（Host資格）も別である。

| 入力・Ruleの状態 | 判断上の結論 | 留保・反例 |
| --- | --- | --- |
| すべての入力が揃い、一行だけ成立、他の行はすべて不成立 | Matched。選択したRule IDと理由を示せる | 作用がCommitされたことは意味しない |
| 二行が成立し、両行の出力値も同じ | Unique違反のConflict | 同じ値なので任意の一行でよいとはしない |
| 一行が成立するが、他行の条件を決めるFieldが未取得 | Insufficient Input | 他行も成立する可能性がある。Uniqueを確定しない |
| 必要な入力が揃い、どの行も不成立 | No Match | Decision契約が拒否・未設定などへどう写すかは別に宣言 |
| すべての入力を取得し、既知の範囲で不足なく検査した | その明示Domain内での網羅・競合を検証 | Domain外や未知のTenant／時点へ証明を拡大しない |
| 取得可能範囲にRLS・LIMIT・部分取得がある | Claim Closureが足りなければ判断未確定 | 0行を全母集団における不存在へ昇格しない |
| Ruleが適用される時点／Operator版が未確定 | Unresolved Meaning | 現行版・行順・既定TZを推測で補完しない |

有限な区分・十進数の閉／開区間から開始し、OverlapとGapには具体的な反例の入力を出す。静的に証明できなければ「未証明」と表示し、検査した入力集合で重複が見つからなかったことを「全域で競合なし」と書かない。最初の実装では、この範囲に限定する方が万能推論よりも実際の検収が容易である。

DMNは参考規格とし、現行正式版は[OMG DMN 1.6](https://www.omg.org/spec/DMN/1.6)、1.7は[Beta](https://www.omg.org/spec/DMN/1.7/Beta1)である。ただし、今回のUnique・Gap・Overlapの限定検証はDMNへの完全準拠、FEELの包括的な型変換、DMN XMLの入出力を採用する根拠にはしない。

## 31. この設計から次の作業へ渡すもの

1. **S0のOwner**：main統合済みPR #4751のManifest／Pinned Source試験とPR #4704の正式Reader接続を比較し、「Readできた」と「解釈できた」の差を明示する。重複Parserを作らない。
2. **意味と生成のOwner**：固定Rule条件のSource span・選択済みMeaning・Operator・実Build Sourceと生成Goのifを結ぶ一件を検収する。Generator契約はPR #4748の主張と突き合わせる。
3. **Rule AnalysisのOwner**：既存#988／#1637／#1180／#2514に、一枚のUnique決定表に対する「同じ値の二重一致」「一行成立＋別行判定不能」「真のNo Match」「適用範囲限定」の反例を渡す。
4. **製品CLI／セルフビルドのOwner**：通常tei CLI→実選択Provider→独立Consumerと、Plugin自身のTEI生成範囲の増大を別に検収する。S0テストやReferenceのGo子Process成功を製品完成の証拠にしない。

**採用境界**：本節の#4751はmain統合済みであり、その他のPRは確認時点の進行中候補であり、正式意味・Rule API・公開CLI・新しいRule Result Enumを採用済みにするものではない。判断を説明する出力は既存Operation／Source／Evidenceから生成する投影として扱い、第二の正本にしない。全体P0、Go単一バイナリ、外部Hostへの成果物配置、Studio Stage1→Stage2という基本方針は変更しない。


## 32. 部分観測のもとで、何が証明できるか（組織・REA・Queryからの統合）

決定表の「Matched」「Conflict」「No Match」と、Relationを検索して得た「候補が存在する」「候補が一人だけ」「候補がいない」は、同じ判定ではない。ただし、どちらにも**部分的な正の証拠と、全域についての結論は非対称**という共通性がある。

- ある対象について必要な条件をすべて満たす二つの異なるRule行が現に証明できれば、Unique違反は確定できる。他の行の取得が不完全でも、すでに証明できた二行の競合は消えない。出力値が同じでもUnique違反である。
- 一行の成立が証明できても、それだけでは「一致は唯一」といえない。他の行について未取得の条件が残れば、二行目が成立する可能性を排除できない。
- 一致行が観測されなくても、Ruleの明示Domainと必要なInput／Relationが充足しなければNo Matchを証明できない。空集合や0行の取得結果を、全対象に関する不存在証明に流用しない。
- 組織のRoleAssignmentとResponsibilityを結ぶ例では、異なる二人への正のRelation経路が揃えば「構造上の候補は少なくとも二人いる」と証明できる。一人に二本のOccurrence経路があるだけなら、人物を二人と数えない。さらに「承認可能な人が二人いる」という業務結論には、必要な不適格条件・適用時点・委譲・現行Grantの別の検証が必要である。
- Rowsetが読み終わっていること（FullyRead）と、対象母集団に対する取得完全性が証明されたこと（Claim Closure）は別。実ProviderのCoverageUnestablishedを、Rule評価側でCompleteへ補正しない。

したがって静的・動的解析では、少なくとも**成立が証明された条件、成立しないと証明された条件、なお成立し得る条件**を区別して説明する。これらは検査時の観点であり、共通の永続Enumや別のRule Result Schemaを新設しない。

| 観測 | 論理上の結論 | 保留すべきこと |
| --- | --- | --- |
| 完全に確かめた異なる二行がともに一致 | UniqueのConflictは確定 | 他行を含めた一致件数の全数 |
| 一行は確実に一致、別行は入力不足 | 少なくとも一行は一致 | 一致は唯一かどうか |
| すべての行について不成立を確認、必要Domainも確定 | No Matchを確認可能 | そのDomain外・他時点への一般化 |
| 部分取得から二人分の正のRelation経路 | 構造候補の少なくとも二人を確認 | 業務適格性と全候補人数 |
| 同一人物への複数経路 | 経路は複数、人物Identityは一人 | 同一性の異なる人物が追加で存在しないこと |
| 非出力Field／NOT EXISTS条件が未取得 | 業務判断に必要な情報が未充足 | 適格候補の確定、権限、業務効果 |

この非対称性は決定表・組織モデル・SQL／NoSQL検索に共通するため、BRMS専用の新しい存在判定Engineを実装せず、既存のFunction Data Contextと独立Expectedに試験を返す。参照：[高度組織C1の検討PR #4622](https://github.com/hat47x/tei-platform/pull/4622)、[組織Relation候補試験PR #4737](https://github.com/hat47x/tei-platform/pull/4737)、#4490（実二時点Query）、#2506（Function Context）、#1180／#2514（独立Expected）。

検証候補は、X13「partialでも二行の完全な正の一致でConflict確定」、X14「FullyReadでもCoverageUnestablishedならNo Matchを確定できない」、X15「同一Personへの二Occurrenceを二人としない」、X16「非出力Fieldの否定条件が未取得なら適格判断を保留」とする。既存Test／Expected Ownerへ渡し、この文書に別の試験正本を置かない。

## 33. BRMSの業務説明と早期シミュレーション：同じSourceから導出する

業務担当者が読む規則は、単なる条件式の羅列よりも「何を対象に、どの時点の、どの事実を使い、何を決めるか」が分かることを優先する。そのため、IF・Markdown表・GraphのSourceには、**判断対象、入力の意味・取得範囲、適用条件、出力、該当なし・情報不足・例外の扱い、必要な後続操作**を、ほぼ直訳できる水準で記述する。Goの型・struct・実装上の分岐命名はDSL表層へ露出しない。

既存のData／Function／BusinessとそのSourceの参照から、次を派生させる。

1. **業務説明**：入力条件と適用範囲を原文位置へ戻せる自然な文章。未定義の運用手順・優先順位をAIが追加した場合はCandidateとして区別する。
2. **決定表の診断**：Overlap／Gap／No Match／未取得の判断不能。静的に検証できるDomainと実行時に必要な取得資格を区別する。
3. **契約に基づくテストパターン**：境界値・NULL・partial・重複・有効日・Rule版を独立Expectedとして既存の検証経路へ提出する。テスト表の内容を業務意味の第二正本にしない。
4. **早期シミュレーション**：外部部品がまだ未実装なら、TEI内で公開する契約とTODOを示し、契約に適合するMockで値と遷移を試す。Mockの成功は実Provider呼出し、実DB commit、外部Effect Receipt、製品Consumer受領とは区別する。
5. **成果物への生成**：Accepted Meaningと選択されたOperator／Generator／ToolchainからGo等へ展開し、実Builderがその生成Sourceを読み取ったことを検証する。生成したデータを手書きGo関数へ渡すだけでは、Ruleの実装をTEIが生成したとはいわない。

これは新しいSimulation Engineや形式証明処理系の採用ではない。Lean風の契約、テストパターン表、実Goの検証は、検証する責務と既存の実装範囲に応じて使い分ける。SHOで説明文や設計書を作る際には、Accepted Meaningから導いた記述と未受理の補足案を分ける。SUIで議論したRule例外や価値判断の候補も、既存Review経路へ戻した後でのみ正式Sourceに反映する。

**次の最小の実装差分**は次の順とする。S0のManifest／Pinned Source予備検証はmain統合済み#4751として再実装しない。先に既存#2621／#988の正式Frontend選択・Reader・Meaning Bindingへ一Ruleを接続する。続いて実Generatorがその型付きRuleを消費した証拠を#4544／#4748で照合する。Unique決定表のX13〜X16はこの接続と独立に有限Fixtureで準備できるが、正式製品経路への統合と同一の達成表示にしない。

[宣言的構文の製品接続PR #4756](https://github.com/hat47x/tei-platform/pull/4756)のSource／Meaning／Generation／Execution／Provenance・Reuseという資格分離とも整合させる。表・Graph・IF・SQLを一つの文法へ統合する必要はない。業務意味を保った表層ごとの解釈とTargetごとのコード生成を優先し、共通Rule Runtimeや第二Source Indexは設けない。

**採用と検証の現在性**：本節は設計・引渡しの案であり、一般Unique決定表の実装、製品Readerの受理、Go／SQL／JavaのTarget間同値、実Provider、Desktopセルフビルド、Ownerの意味受理を新たに証明するものではない。再利用するEvidenceはそのProducer・Artifact・Source Revision・試験Scope・現在の資格とともに確認する。
## 34. 判断の出典から実行まで：「同じ規則で動いた」を何で確かめるか（2026-11-15）

mainへ統合された[Graph／Rule候補構造指紋 #4758](https://github.com/hat47x/tei-platform/pull/4758)は、限定Readerが解釈したNode ID集合、Edge IDと有向端点、Rule IDと条件の種類から、ドメインを区別したSHA-256を作る。生成したReference Binaryでも `-candidate-shape` から照会できる。これは、Ruleの実装をTEIが生成する際の追跡可能性を高める一歩である。ただし、**「Binaryがある指紋を自己申告した」と「そのBinaryの判定ロジックが当該Ruleに一致する」は別の主張**である。

[main統合済みの負例PR #4761](https://github.com/hat47x/tei-platform/pull/4761)は、Goに埋め込まれた指紋を変えずに `e.from == nodeID` を `e.to == nodeID` へ差し替え、自己申告が一致しても別Binary・別実行結果になることを検査する提案である。この負例はmainに統合済み。ただし、限定Referenceに改変を実行時に防止する機能が備わったという意味ではない。既存のBuild／Artifact／実Consumerから**独立して確認した証拠**と結び付ける必要がある。

| 何を特定するか | 既存の証拠と責任 | その証拠だけでは不足すること |
| --- | --- | --- |
| Sourceの原文 | Managed Workspaceの実Source byte、Source Index、Occurrence、Fingerprint | 表示上似ている文書・Candidate・受理済みMeaningと同じとは限らない |
| 参照Readerの候補 | 限定Graph／RuleのCandidateShapeFingerprintと元byte span | 業務意味・会員種別・時間条件まで含めた一般の意味指紋ではない |
| 受理された意味 | Owner／Language・Role・型付きOperator・SourceとModel Revisionの既存受理記録 | 同じ意味をTargetが消費した保証ではない |
| 生成時の実入力 | 選択されたGenerator／Toolchain／Plugin、依存閉包、実Build Source member、Build Attempt | 生成したファイルが後段のコンパイラに読まれた証拠なしでは利用済みといえない |
| 実行成果物 | 第三者が実Artifact byteを測ったDigest、Target、当該AttemptとProducer | Binary自身の `-candidate-shape` 自己申告だけでは制御ロジックを証明できない |
| 実行の観測 | 同一Artifactへの実入力、実Process／実Provider、独立Expectedとの差、Consumer受領 | 対象の業務認可、更新Commit、外部Effect Receiptは別資格 |
| 現在の許可 | Host側のCurrent Grant／Audience／Effect Contract | 過去のRule一致・成功Receipt・ハッシュから現在の権限は導出できない |

上の各行は**説明・検収の観点**であり、新しい永続Receipt形式、Evidenceサーバー、中央Rule Repositoryを定義しない。既存のWorld Fingerprint、Model Revision、Artifact Revision、Source Index、Builder／Provider Receiptを意味に応じて参照する。参照SourceのA→B→Aや同名再作成、Generator版変更を、単なるハッシュ一致で再受理しない。

特に参照実装の候補指紋は、表示名、Markdownの配置、Mermaidの見せ方、Source位置を含まない。**それらを評価しない限定Referenceで等しい**という事実を、他のMeaning Roleや業務説明まで含む製品全体の意味同値へ拡大しない。将来の意味同値は当該Meaning／Operator／Targetの契約と依存集合に沿って個別に判定する。

## 35. 変更を六つの経路に分けて検証する

実装の更新が必要か、実行だけを繰り返すべきか、証拠を再確認すべきかは別々の問いである。BRMSではRuleの変更と、実行時の対象／取得データの変更を明確に分離する。

| 一因子変更 | Source原文 | 限定候補構造／生成Go | 次に必要な確認 |
| --- | --- | --- | --- |
| Mermaidのレイアウト・表示名だけを変更 | 原文Fingerprintは変わる | 限定Referenceの対象構造・生成Goは不変のはず | 旧Source Indexを再利用しない。Meaning／説明の影響範囲は別判断 |
| 原文に説明だけを追記 | 原文Fingerprintは変わる | 限定Referenceの処理Sourceは不変のはず | 証拠位置・参照文書を再検証。自動で業務意味を改訂しない |
| Edgeの向き、Rule条件を変更 | 変わる | 変わる | 依存するMeaning／生成分岐／独立Expected／Artifactを再照合 |
| 同じ生成BinaryへのNode ID指定を変更 | 変わらない | 変わらない | 実入力と結果を比較。入力変更を再生成理由にしない |
| SQL構造は同じでDB行の値が変わる | Rule Sourceは不変 | 生成SQLは不変でもよい | Providerが実際に取得した行・Snapshot／Read Closureと結果を再検査 |
| Generator／Toolchain／選択Plugin版を変更 | 業務Sourceは不変でもよい | 生成Go・Binaryが変わり得る | 生成器の版、実Build Source消費とTargetの同値を再検証 |
| 生成Goの条件だけを事後改変 | Rule Source／候補指紋は不変でもよい | 実GoとBinary、結果は変わり得る | 自己申告指紋は不十分。外部測定Artifactと実実行を比較 |
| SourceをA→B→Aに戻す／Current Grantを変更 | 最終byte／候補構造が再一致し得る | 生成Goが再一致し得る | Source Occurrence／Decision履歴・現在の権限は再確認 |

前四件は既存Graph／Rule Fixtureと同じBinaryへのRuntime Input、次の二件は実DB入力・Toolchainの独立性に対応する。Graph／Rule側の[main統合済みの差分試験PR #4759](https://github.com/hat47x/tei-platform/pull/4759)は「原文SHAが変わり、候補指紋と生成Goが同じでもSource Indexは失効する」を確認する提案であり、mainに統合済みである。

別の領域では[main統合済みのSQLite Window実表入力試験PR #4760](https://github.com/hat47x/tei-platform/pull/4760)が、同一の生成SQLでもDB入力を変更すれば実結果が変わることを限定試験で実証した。ただしこれはWindow Queryの限定試験であって、BRMSの正式SQL Target、SourceからのDecision生成、Provider選択、Go／SQL同値を示さない。**再生成不要と判断できること**と**再実行の入力・結果を検証できること**を混ぜず、既存#2600／#4544／#2514へ渡す。

## 36. 判断の説明は、検査可能な理由と判断の限界を両方示す

SHO・Studio・CLI・AIへ同じ判断を説明するときは、業務担当者にとって分かる文章を優先する。表示の粒度は異なっても、次の内容を元Sourceと実行の証拠にたどれるようにする。

- **何を決めたか**：業務上の対象、結果、適用するRule／時点／Scope。Decisionと後続の更新操作を分ける。
- **何が根拠か**：どの入力・Relation／Rule行の条件が成立し、その証拠がどの取得範囲から来たか。少なくとも一件の成立の根拠を示すのか、Uniqueまで証明できたのかを区別する。
- **何がまだ不明か**：未取得・partial・NULL・否定条件に必要なRead、Ruleの未受理、別時点の状態。競合を証明できた場合と、競合がないと証明できた場合を対称に扱わない。
- **どの実装が評価したか**：正式製品なら受理Meaning→Generator実入力→実Artifact→Actual Consumerの対応。限定Referenceならその資格を明示する。
- **次に何をしてよいか**：ルールが「注文可能」と判断しても、保存や外部送信は別のHost Grant・Effect Contract・Receiptに従う。説明文から権限を新たに作らない。

例えば会員割引で購入額が基準額以上かつstandard区分なら「該当Ruleから割引率5%」と説明できる。しかし他の行の入力が未取得なら、唯一一致したとは書かない。逆に二行が確実に成立したときは、残りの行が未取得でも「Unique違反が確認された」と説明できる。評価不能を通常の「対象外」へ変換しない。

理由の表示は**許可されている内容に限る**。権限がない利用者に対して、非公開Ruleの本文だけでなく、存在を示すID・候補人数・Source digest・内部の版を出すことも避ける。監査やOwner受理に必要な詳細は、適切なAudienceの既存Review／Evidence経路で確認する。AIが書いた説明は原Sourceを上書きせず、生成された説明と追加した推測を区別する。これは[Plugin公開文書・Disclosureの設計PR #4762](https://github.com/hat47x/tei-platform/pull/4762)の境界とも整合する。

## 37. 次の製品PRに渡す最小の検収票

ここから製品機能を増やす際には、**一件のRuleを正式に選び、その生成処理が実行されたこと**と、**一般のUnique決定表を静的に解析すること**を別の作業として進める。

| 作業 | 最小の検査 | Owner／再利用する既存資産 |
| --- | --- | --- |
| Source選択・正式Meaning接続 | 実Workspace内の選択済みSourceを、正しいReader／Role／型付きOperatorに対応付ける。未選択・Fence内・同名別IDは誤採用しない | #2621／#988／#1018／#4359、main #4751、Reader修正PR #4725 |
| 生成ロジックの実消費 | Rule条件変更が実Generatorから出る分岐に効き、実Builderが選択Source memberを読んだことを照合 | #4544／#2595／#2600、Generator設計PR #4748 |
| 生成Artifactと結果 | 候補指紋の自己申告だけを信じず、実Artifact Digestと同一Binaryでの異なる入力・実出力を独立に照合 | main #4758／#4761、#1180／#2514 |
| 有限Unique決定表 | 完全に証明した二行の競合、二行目の入力不足、真のNo Match、重複行が同じ値を返す場合を分ける | #988／#1637／#1180／#2514、前節X13〜X16 |
| 業務説明・Mockでの流れ | 対象／時点／条件／理由／情報不足を自然文にでき、未実装の外部契約はMockで試せる。実Effectと混同しない | Data／Function／Business、SHO、既存Script／Effect設計 |
| 実Providerとセルフビルド | 通常製品CLI→選択Provider→独立Consumer、TEI由来の判断ロジック増加・Delegate削減・Stage1→Stage2を別々に検収 | #4495／#1251／#2586、#2831／#4583、自己記述PR #4763 |

最初に取り込む変更は製品S0-A／Meaning Bindingの担当に寄せ、Graph Fingerprint・Source Index・Reference Generator・独立ExpectedをBRMSで再実装しない。Ruleに適用する生成の責任範囲と、既存Goへ単に委譲する範囲を分ける。実Source／候補形状／受理意味／生成Source／Binary／Runtime入力／実DB入力／結果が同じ試行の中で結び付かなければ、原因を特定して次の差分を作る。

本節は**実装済みの現状説明と未採用の製品受入提案を区別する更新**である。#4758の候補指紋実装を一般意味Fingerprint、製品Artifact保証、権限、BRMSの実装完成へ昇格させない。新Result Enum／新Grammar Engine／Rule Server／Host／第二Source Index／全体Gateは導入しない。


## 38. 判断できることの上下限を示す：Unique決定表の最小検査

部分的な証拠が存在するとき、Ruleの評価を単にtrue／false／unknownの三択として出力するだけでは、確定できる結論を捨てるおそれがある。最初の有限なUnique決定表では、既存の型付き条件式と入力の取得資格から、各行を**確実に一致する／一致する可能性が残る／確実に不一致**に分けて考える。

説明上は、確実に一致する行の集合を `confirmed`、一致し得る行の集合（確実な一致を含む）を `possible` とする。これは検証時の集合であり、新しい公開Result型・Rule Engine・永続テーブルの提案ではない。Domain・Rule行・版・時点・必要入力が明示的に固定されていることが、この分類の前提である。

| 既存の観測に対する検査 | その範囲で確定できる結論 | 依然として未確定のこと |
| --- | --- | --- |
| `confirmed`に相異なる行IDが二つ以上 | Unique違反が確定。出力値が同じでもConflict | 未観測の別行を含めた一致の全数 |
| `confirmed`が一行、`possible`も同じ一行 | UniqueなMatchedが確定 | 未固定のRule版・Scopeへ拡張した一致 |
| `possible`が空集合、全必要入力の不一致が証明済み | No Matchが確定 | 明示Domain外の存在 |
| `confirmed`が一行、`possible`に別行が含まれる | 少なくとも一行は成立 | 一意性は未証明。実行上の選択を保留 |
| `confirmed`が空集合、`possible`に行が残る | まだ成立行は証明されていない | No MatchもMatchedも確定しない |

ここでの「possible」は、欠損した入力を任意に補ったとき条件が成立し得るという論理上の可能性であって、実際にその値が存在すると推定したわけではない。SQL三値論理のNULL・未取得・RLS・partial・permission-denied、時点の不一致、型に適合しない入力は同じ理由としてまとめない。特に**評価エラーや未受理の意味を、単なる入力不足として計算し続けない**。上記の推論は必要な条件の意味が受理され、検証に利用できる場合にのみ適用する。

組織Relationについても同じ非対称性が使える。異なる二人の完全な正の経路が得られれば「構造候補が二人以上」は確定するが、まだ他の人物がいる可能性を排除できない。さらに、否定条件・適用時点・現行Grantが未確認なら「承認できる二人がいる」とは結論しない。既存#4490／#2506／#988／#1180／#2514の評価・期待結果で検査し、BRMS専用のRelation Joinを増設しない。

## 39. 三種類の変更を分け、どこまで再利用できるかを検証する

BRMSでは、差分を受け取るたびに次の三つを別々に判定する。

1. **Source／意味の受理を再確認する必要**：Source byteや位置、Occurrence、Reader／Role、Rule、適用Scope・時点が変わったか。原文の説明追記やA→B→Aのように、候補の構造指紋が同じでもSource Indexや現在の意味受理が有効とは限らない。
2. **生成物を再利用できるか**：受理済みMeaning・Operator・Generator・Toolchain・選択Plugin・実Build Source・依存閉包の対応が確認できるか。Graph／Rule Referenceの候補指紋だけで製品ArtifactのReuseを許可しない。
3. **過去の判断結果を再利用できるか**：実行時入力、Provider Snapshot、Query／Read Closure、Rule有効時点、現在Grant、独立Expectedと実Consumerが変わったか。生成コードが不変でもDB入力や現在の権限が変われば再判断が必要である。

今回のmain実証では、#4759が「原文だけ変わるが限定Graph／Rule構造と生成Goは同じ」を観測し、古いSource Indexの失効を確認した。#4761は自己申告の候補指紋が同じでも実Goの分岐とBinary Digest・結果が異なる反例を固定した。#4760は同じ生成SQLを保持して実DB行の値を変え、結果が変わることを限定SQLiteで実証した。**これら三つの実証は異なる層に属し、統合して製品Reader／Rule Target／Provider／Studioの合格と呼ばない**。

既存World Fingerprint／Model Revision／Artifact RevisionとSource Index・実Builder・実Providerの証拠を使い、次の説明ができることを目標にする。「Rule内容は変わっていないが、取得データが変わったため再判断した」「表示だけ変わったためGo生成候補は不変だが、原文の再確認は必要」「同じ指紋が報告されたが、実Artifactが異なるため再検証を要求した」。この説明に新しい状態保存表や共通Attestationサービスは必要ない。

## 40. 次の二つの小さな実装単位と、計画上の停止線

**第一の実装単位**は、製品S0-Aの正式Markdown Source選択・Reader・元byte spanから、受理された型付きRule意味へ一件だけ接続すること。既存#2621／#988／#1018／#4359に従い、Selected Sourceと実Reader入力、Source Index、Role、Operatorを同じAttemptで照合する。今回の限定Graph／Rule一行を使い、同名Nodeと多重Edgeを保持する。サンプル文やコードFence、未選択Rootを業務意味に昇格させない。

**第二の実装単位**は、その一行のRule条件変更が、製品Generatorの実入力、生成されたGo分岐、実Builderが読んだSource member、実Artifact、同じBinaryへの実行時Node入力と出力へ伝わったことを独立に照合すること。#4544／#2595／#2600／#2514とGenerator設計PR #4748に接続し、手書きGoへ単に委譲した場合は「TEIが判断ロジックを生成した」と数えない。Sourceの同値・Candidate Shape・実Binary SHA・Consumer実Callはそれぞれ証拠を残す。新しいRule Compiler／Runtime／Builder Registryは作らない。

並行して進められる**限定検証**は、有限なUnique決定表に対する静的な重複・漏れ・到達不能と、前節の`confirmed`／`possible`の反例である。主眼は完全な推論エンジンではなく、Ownerが受理したDomainの範囲内で「競合の証拠」と「一致が唯一という証明」を混同しないことである。Markdown表とIFを双方Sourceとして編集せず、どちらかを正本、もう片方を派生表示として扱う。

設計で決めた条件をCLIやStudioで説明する場合には、元Rule行、明示Scope・時点、入力の取得範囲、判断できたことと未確定なことを自然な日本語で示す。業務説明・テスト表・Mockでのシミュレーションは同じSourceから派生させ、実Providerの成功、外部Effect Receipt、現在のHost Grantを偽装しない。説明や図表の更新は必要な再検査を起こし得るが、説明自体を第二Meaning正本へ昇格させない。

この作業は**すでにある#4765設計の継続**であり、新しい全体P0・独立BRMS Engine・承認Server・共通Result Enum・Hard Gateの導入ではない。StudioのStage1→Stage2セルフビルドと通常`tei` CLI→実Provider→独立Consumerを優先する。今後のCI・Owner受理・実際のTargetコード生成・Windows実行は、この文書更新とは別の証拠で検収する。


## 41. 同じ業務判断を一件だけ追跡する：Source入場と判断の受理を分ける

[標準データ様式の設計（§39）](SEMANTIC_STRUCTURE_MAPPING_AND_DECLARATIVE_SYNTAX_VIEW_2026-11.md)は、Graph／RuleとWindowで共通化する対象を、Manifest／Pinned Root／Reader選択／Source Index／Roleの**製品S0-A入場条件**へ限定した。Graph／Ruleの意味をWindowの型付きQueryへ揃える必要はない。この分離をBRMSでも維持し、最初は固定の`baseline.tei.md`の一つのRuleを正式経路へ載せる。

最初の製品接続は、次の四つを**一つずつ、同じSource／Selection／Attemptを参照して**検収する。既存のテスト専用S0-Pで成功した原文取得を、正式S0-Aの合格にはしない。

| 段階 | 具体的に照合するもの | 合格と扱わない例 |
| --- | --- | --- |
| 1. 選択された原文 | Workspace ManifestのManaged Root、明示Frontend／Role、Pinned Source byte、元のRule ID／条件セルspan、Source Index | 拡張子からの自動認定、Root外の草稿・コードFence内の表、先行試験が別途読んだ原文 |
| 2. 型付きの意味 | 同じRule IDの条件を、既存Operatorの型・意味・Scope／時点、Ownerの受理版に結び付ける | Reference Candidateの成功やSource Indexの `md` FormatをMeaning Roleへ昇格 |
| 3. 生成した判断処理 | その受理意味が実Generatorの入力となり、Rule条件の変更で生成Goの実分岐が変化し、実Builderが生成Source memberを読む | 手書きGoへの委譲、Binaryの候補指紋の自己申告、未利用の生成ファイル |
| 4. 製品での判断 | 実Artifact byteとProvider／Invocation、同じBinaryへの実行時入力、独立Expected、通常`tei` CLI／Consumer受領 | 参照Go子Process、Mock、固定JSONの一致だけで製品受領と見なす |

**最初に許す差分は第1段階だけ**とする。ここに同時に新しいDecision Engineや一般Markdown Parserを実装せず、Reader／Workspace Owner #2621／#988／#1018の既存契約へ返す。第2〜4段階は先行実装の証拠を受け取ってから同じRuleで進める。Graph／Ruleの固定6入力やWindowの負例5入力に関するS0-PのGo試験は既存資産として再利用し、同じGo試験の複製をBRMSへ作らない。

## 42. Ruleの一致、業務適格性、実行許可は別の主張である

組織C1／REAや[実行モデルの設計](PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)を踏まえ、判断の評価結果を後続Operationの許可へ直接つなげない。例えば「注文の承認担当候補」を選ぶ場合、**対象Personが組織関係に現れること**、**指定時点で承認条件を満たすこと**、**現在のHost Grantで実際に操作できること**は異なる。

ここでは、既存モデルで検証する順序だけを定める。

1. **Ruleと対象の確定**：業務Subject、同一Decision／Scope、Rule版、有効時点（validAt）と情報確認時点（knownAt）を固定。組織のPerson、REA Agent、外部候補Identityは同名・同IDだけで結合せず、受理済みのQualified Mappingを必要とする。
2. **入力の充足**：肯定条件の成立に必要な行・Relation Pathと、否定条件・全件性・有効期間を確かめるためのRequired Readを区別。異なる二つの完全な一致行が同じDecision／Scopeで成立すればUniqueの競合は確定できるが、行集合・適用Scope・時点が不明なら競合と断定しない。
3. **判断の確定**：前節の`confirmed`／`possible`は、受理済みRuleの有限なDomainの中でのみ解釈する。Source、Provider、Snapshotの取得資格が不足していれば、唯一性や不在の判断を保留する。明示NULL、部分取得、権限拒否、未対応型、評価失敗を「不一致」へまとめない。
4. **操作の許可**：判断で選ばれたRoleや担当候補と、Hostの現在のGrant／Capability・Audienceを別に確認する。Responsibilityや業務上の承認可能性だけからHost Grantを発行しない。
5. **外部効果の確認**：Commit／通知／再送はOperation・Effect・Receiptで実際の試行と結果を確認する。Mockが値を返したこと、事前条件が満たされたことを実DB commitや外部Effectの完了に昇格させない。

これらは新しい承認プロセスエンジンや中央Policy Registryの段階ではない。業務の説明・設計図・CLI診断・Studio／SHOの表示は、既存Source・Typed Meaning・Function Context・Operation／Evidenceを使った**読取り用の投影**とする。権限を持たない利用者に対して非公開Ruleの名称、存在件数、出典指紋まで開示しない。判断の理由は「確認できたこと」と「確認できない理由」を自然な日本語で示し、AIが補った推測を元規則として表示しない。

## 43. 次の独立Expectedへ渡す、有限な検証組合せ

最初のUnique決定表の例（standard／premium、非負JPYの購入額、0％／5％／10％）を、正式なMarkdown表構文の採用前でも**手で型付けした有限Fixture**として検査できるようにする。ここでは入力集合と検査義務だけを固定し、第二のExpected Storeや新しいRule Result Enumを作らない。

| 検証候補 | 固定する入力／変化 | 証明すること・失敗させること |
| --- | --- | --- |
| U-01 | standard、9,999 JPY、Rule三行、完全な入力 | D01のみ一致し0％。D02／D03の不成立も確認 |
| U-02 | standard、10,000 JPY、同じRule版 | 境界でD02のみ一致し5％。9,999と10,000の区間にGap／Overlapがない |
| U-03 | premium、0 JPY、同じRule版 | D03のみ一致し10％。金額0を未取得とみなさない |
| U-04 | standard、5,000 JPYに重複する追加行を設ける | 出力値が同じでも、異なる二行の確定一致ならUniqueのConflict |
| U-05 | D01の条件は成立、追加行の必須Fieldが未取得 | 一行成立が確認できても一意性は未証明。未取得をfalseに置換しない |
| U-06 | 明示したDomain内で全行不成立と確定できる入力 | No Match。値のNULL／不適格／未取得と別の結果として扱う |
| U-07 | 観測できたRule／Relationがpartialで、否定条件のReadが未取得 | 該当行0件や一人の候補を全域の不在・唯一に昇格させない |
| U-08 | 同一Personへの二Occurrenceと、別Personへの二経路を対比 | 経路数と人物数を区別し、業務適格者数・Grantは別検査 |
| U-09 | Sourceの説明追記、Mermaidのみ変更、Ruleの条件変更 | 原文の失効、候補構造／生成Goの差、意味受理を再検証する範囲を分離 |
| U-10 | 候補指紋を維持したまま生成Goの条件を改変 | 自己申告指紋では実Binaryの判断ロジックを保証できない。実Artifact／出力で差を観測 |

U-01〜10は**本設計から既存#1180／#2514へ提出する検収候補**であり、テストが新しく実装・実行・Owner受理されたことを意味しない。特にU-06は、Domainを非負JPY・二区分に固定した場合、三行が全域を覆う基準決定表では構成できない。**試験用に一部の区間を意図的に削除した別のRule版**を使い、基準表のまま「No Match」になる入力を捏造しない。U-05の追加行も既存三行だけでは成立せず、未取得Fieldに依存する試験専用行を明示する。

実装・検証のOwnerは、有限解析 #988／#1637、Source選択 #2621／#1018、Generator／Builder #4544／#2595、Function Context／組織Query #2506／#4490、実CLI／Provider #4495／#2586、独立Expected #1180／#2514とする。全体の最優先はStudio Stage1→Stage2 #2831／#4583および製品CLIからの実Provider・独立Consumerであり、Rule全般の仕様統一や一般的なDMN Runtimeを前提にはしない。


## 44. 決定を確定するには、規則の範囲と入力の範囲を別々に確かめる

第38節の`confirmed`（確実に一致する行）と`possible`（一致する可能性が残る行）は、**評価対象となるRule集合が確定している場合**の判断方法である。実WorkspaceのReaderがまだ一部の表しか選択していない、Ruleの版が不明、対象Tenant・有効時点・Roleによって規則の適用範囲が変わる、といった場合には、その集合自体を未確定として扱う。

ここで検証する対象を二つに分ける。

- **規則側の範囲**：どのSource／Occurrenceを選択し、どのRule ID／版／Scope／有効時点を対象にしたか。未選択の文書、未解析の表、重複ID、競合するRole、未受理のMeaningが残っていないか。単にSource Indexが作成されたことは、Rule集合が確定した証拠ではない。
- **入力側の範囲**：各Ruleの条件に必要なField、Relation、時点、非出力の依存、否定条件のReadが揃い、取得した母集団・Snapshotが主張に十分か。Cursor終端や`FullyRead=true`だけでは母集団の全件性を示さない。NULL、未取得、アクセス拒否、評価失敗は分ける。

| 規則集合と入力の状態 | 言えること | 言えないこと |
| --- | --- | --- |
| 選択済み・受理済みの同一Decision／Scopeで、異なる二行が確実に一致 | **その範囲内のUnique違反を証明できる**。別のRuleが未確認でもこの二行の同時成立は消えない | すべての適用Ruleを発見した、全違反を列挙した |
| 対象Rule集合が確定し、`confirmed`も`possible`も同じ一行 | **そのRule集合・時点・Scope内で一意に一致** | 将来版・別Tenant・未選択Sourceまで一意 |
| 対象Rule集合が確定し、すべての行で成立し得ないことを確認 | **その範囲内でNo Match** | 未取得のRule集合を含む広い範囲での不存在 |
| 入力の取得範囲は完全だが、適用Rule集合が不明 | 観測したRuleについての局所的な一致／不一致 | 決定表全体のUnique／No Match／Gapがないこと |
| Rule集合は確定しているが、否定条件の必要Readが不足 | 確実な正の一致や、必要な反例がある場合の局所競合 | 一致の唯一性、該当なし、適格者の不存在 |
| Ruleの意味・版・Source Roleが未受理／不整合 | 出典付きの候補や構造上の疑義 | 受理済み業務Decisionの確定結果 |

一部の行についてだけ判定できるときは、「その行の成立」と「決定の確定」を分けて説明する。特に`possible`が空であることだけでは不十分で、**そもそも調べるべきRule行が欠けていないこと**を確認する。行の意味が未受理、型検査に失敗、評価中にエラーが起きた場合は、これらを単なる`possible`として数え直さない。

この二つの範囲の検証は、既存Workspace／Source Index／Language・Role選択、型付きMeaning／Operator、Function Context／Queryの取得範囲・Claim Closureを組み合わせる。BRMS専用のRule Registry・Rowset Engine・新しい永続状態を設ける理由にはしない。

## 45. 判断の説明は、実行した処理と開示できる情報に合わせる

[データ様式の横断設計](SEMANTIC_STRUCTURE_MAPPING_AND_DECLARATIVE_SYNTAX_VIEW_2026-11.md)と、[プラグイン構成の設計](PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md)、[実行モデルの図解](PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)から、BRMSの説明に必要な検証を三段階に分ける。

1. **判断の根拠**：選択した業務Subject、Rule行と条件の出典、Rule集合の範囲、必要な入力と取得範囲、確認済みの一致・未確定の理由を示す。説明文は同じ受理済みSourceから派生させ、AIの補足を根拠に書き戻さない。
2. **実際の処理との対応**：どの意味がGeneratorへ渡され、どのSource memberをBuilderが読み、どのArtifactが実行され、どの結果をConsumerが受け取ったか。限定Graph／Ruleの候補構造指紋やBinaryの自己申告だけで、この対応を証明したことにはしない。
3. **現在の開示・操作の資格**：Pluginの実選択版・Lockが一致していても、Ruleの公開文書が存在することや、その内容を現在のAudienceへ開示できることは証明できない。Studioが`authorityLimits`を表示しても、編集・承認・実行権限が新たに与えられるわけではない。Host Grant／Disclosure／Effect Receiptは別に検査する。

例えば「割引率を5％と判定した」という説明には、基準となる会員区分と購入額、Ruleの適用時点、他行が成立しない根拠まで含められる。ただし、権限のないConsumerには非公開Ruleの名称・存在件数・元Source digestを表示しない。理由を伏せるときも、**非公開という制約**と、**評価に必要なデータが不足していること**を同じ内部状態へ変換しない。利用者へ開示する文言や詳細度はAudienceの契約に従う。

main統合済み[Pluginの事前選択・Lock検査 #4800](https://github.com/hat47x/tei-platform/pull/4800)は、文書の正式なSource-backed読取り・Disclosure受理と別資格である。直接上位Pluginの厳密版を追加表示する[PR #4826](https://github.com/hat47x/tei-platform/pull/4826)も、確認時点では未マージであり、BRMSの公開・実行権限を代替する機能ではない。これらは説明やStudioの表示を作る際の既存境界として使い、専用のBRMS権限管理を加えない。

## 46. 次の実装に渡す、二つの範囲の負例と合格条件

第43節U-01〜10のOwnerは変えず、次の負例を**同じ独立Expectedに対する不足条件**として渡す。別の期待結果ファイルやBRMS専用Test Runnerは作らない。

| 追加の照合 | 意図的に欠かすもの | 期待する判定と根拠 |
| --- | --- | --- |
| C-01：見えているD01だけで一致 | 同じScopeのD02／D03を含むRule集合の確定 | D01の局所一致は説明可能。しかしUnique確定とはしない |
| C-02：確認済み二行が一致 | 他のRuleの読取り | 二行が同じ受理済みDecision／Scopeなら局所Conflictを報告。全件の規則数までは主張しない |
| C-03：全入力が揃う | Readerが選択する一つのRule Source | 入力完全でもNo Match／Gapなしを確定できない |
| C-04：全Ruleが見えている | 否定条件に必要な非出力Read | 該当なし／唯一の適格者を確定できない。正の候補存在は別に報告できる |
| C-05：重複するPerson経路 | 異なるPersonの正の証拠 | 経路数を人数に使わない。現在Grantを推測しない |
| C-06：実Pluginの版・Lockだけ一致 | Rule本文のSource Binding／現在Disclosure | バージョン一致をRule開示・実行可能へ昇格しない |
| C-07：参照Binaryが候補指紋を表示 | 実生成Sourceの消費・Artifact外部測定 | 自己申告のみでは実行処理の一致を証明できない |
| C-08：Mockで一連の判断が成功 | 実Provider／Consumer／Effect Receipt | 模擬成功と製品実行・外部作用を分けて報告する |

**最初の実装対象は引き続き一つのGraph／Rule Sourceの正式S0-A入場**である。C-01〜08のために汎用Decision Engine、全DMN／FEEL、組織専用Join、第二のRegistry、Studioの別Runtimeを先行実装しない。Product Reader #2621／#988、型付きOperator #1637、実生成 #4544／#2595、取得範囲 #2506／#4490、検証 #1180／#2514、Provider／Consumer #4495／#2586へ分割して引き渡す。

採用上の停止線も同じである。実装済みなのはGraph／Ruleの限定Reference、WorkspaceによるS0-P取得追跡、実SQLite Window入力の限定検証、Plugin Lockの事前検査など**個別に確認できた範囲だけ**である。正式`*.tei.md` Reader・受理Meaningからの製品Generator／実Builder消費・通常製品CLIの実Provider受領・Studio Stage1→Stage2・独立ExpectedのOwner受理は、この文書更新で成立したとはいわない。
