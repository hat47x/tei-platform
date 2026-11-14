# TEIのビジネスルール分析・決定表統合（設計候補）

- 状態：**設計・検証計画（未採用、未実装）**
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
