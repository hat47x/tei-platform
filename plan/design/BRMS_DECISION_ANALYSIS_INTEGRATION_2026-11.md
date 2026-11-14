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
