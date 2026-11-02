# Everything is Data：定義→意味→Data Context→Provider→Studioの横断接合計画（2026-11-02）

> 状態：計画・引継ぎ票の提案。製品API／TEI DSL構文／Data Operationsの正式名称／Owner意味受理は未決。新しい試験実行や本番適合、既存の判定段階の変更を主張しない。

## 1. 位置づけ

親課題 [#2619](https://github.com/hat47x/tei-platform/issues/2619) の「Everything is Data」を、別担当の設計文書と局所参照実装を**同じSource版・構文Plugin・意味Role・Data Context・Provider・Consumer**で接合する作業票にする。

これは各Issueの仕様を複製する「第二の統合仕様」ではない。既存の [#1119 Critical Path](https://github.com/hat47x/tei-platform/issues/1119)／[#1163 one-revision証拠](https://github.com/hat47x/tei-platform/issues/1163)／[#1180 Scenario](https://github.com/hat47x/tei-platform/issues/1180) を利用し、どの不足をどのOwnerへ引き渡すかだけを管理する。現行の [開発全体計画](../MASTER_PLAN.md) と Gate A/B/Cの歴史的な判定、現在の参照実装を維持する。設計候補を現行機能として表示しない。

## 2. 統合の基準線（成熟した設計判断、正式契約は採用前）

- **Everything is Data**：Data Model、Function、Rule、Test、Scenario、UI、Generatorは標準データ構造で記述でき、解釈部品・Operatorが用途上の意味を与える。Lispの構文や実行方式をそのまま移す必要はない。
- **表層構文は糖衣**：Markdown表、Graphviz/DOT、関数・式・チェーン、Goa風宣言、MukuroL風階層DSLは、構文Pluginで共通のplain Dataへdesugarする候補。引用・例示と実行定義、文字列と参照、lexical scope、評価順序、作用順を区別する。意味上の同値はRole／型／ID／Operator版／Grain／NULL・多重度／Effectの束縛後に評価する。
- **正本と表示**：読みやすいSourceを保ち、正規化データ、受理済みMeaning、Target生成物、Studio/SHO表示、実Effectを異なる資格として追跡する。見た目やJSON byteの一致だけで意味受理を証明しない。
- **明示Data Context**：Data ModelのRelationを利用しても、Functionが読むRelation path／Field／Scope／Grainは明示的にMappingする。「allを要求」と「Providerが実際にcompleteを返した」を区別する。
- **Data Operations（仮称）**：従来SmartDBの汎用化。RDB/OData/REST/Graph/File等のProvider Pluginを既存Capability/Mapping/外部Hostへ接続し、Derived Attribute/DatasetやData Ruleを標準Operatorから構成する。
- **双方向の操作**：3階層→2階層表示のような情報圧縮は可能。表示変更から元データへのUpdate IntentにはSource Entity／Relation occurrence／scope／versionと明示Operationが必要。集計値への非一意な書戻しは専用Actionまたは未解決として扱う。
- **標準Studio**：表／Tree／Graph／式／Source／Instance／Diffは同じ構造の別の投影。特殊UIを必須とせず、Target固有UXが必要な場合にその差分価値を検証する。元SourceへのPatchは一意な対応範囲に限定する。
- **AIフレンドリーDSL**：Goaの宣言・契約中心の明確さと、MukuroLの小文法・インデント・属性記法を比較する。学習済み知識に依存しないGrammar/Role/Operator/正負例をAIに配布し、意味修正率・参照保持・局所diff・コストで選ぶ。正式Grammarは未決。
- **対象領域**：業務に加え、状態・時間・空間・Graphを持つゲーム、書籍、ロボティクス、OS等の異質な例を継続して探す。最小合成ケースは探索・製品スコープの上限ではない。

## 3. 現時点で確認できる個別成果と未接続（2026-11-02）

以下の局所PASSは各Draftの報告に基づく。**本計画の同一Runで再実行した結果ではない**。

| 既存Draft／Owner | 確認済みと報告された範囲 | 同一Runに未接続の部分 |
| --- | --- | --- |
| [#2625](https://github.com/hat47x/tei-platform/pull/2625)／[#2620](https://github.com/hat47x/tei-platform/issues/2620)、[#988](https://github.com/hat47x/tei-platform/issues/988) | Markdown表、インデント、chain、JSON、限定Typed DOT→同じplain Data候補。隔離Python 23/23 | 実TEI Language Plugin、汎用Parser、Meaning採用、Source span往復 |
| [#2623](https://github.com/hat47x/tei-platform/pull/2623)／#2620/#2621 | 引用とRole、literal/ref、scope/capture、Effect、Source編集のB01〜14設計 | Bケース実行、実Role/Authority/Studio |
| [#2624](https://github.com/hat47x/tei-platform/pull/2624)／[#2506](https://github.com/hat47x/tei-platform/issues/2506) | 限定Context Mappingのread-only合成Python 20/20と未宣言領域の有限非感度 | 実Query/Provider取得Scope、実Observed completeness |
| [#2627](https://github.com/hat47x/tei-platform/pull/2627)／[#2611](https://github.com/hat47x/tei-platform/issues/2611) | 同一Semantic DataのProvider経路／Alias候補を区別するGo 12/12 | 実Operation Surface、複数実Provider・Target |
| [#2628](https://github.com/hat47x/tei-platform/pull/2628)／[#2622](https://github.com/hat47x/tei-platform/issues/2622) | 結果同値とProvider過剰Readの違いを示す合成Python 12/12 | 実SQL/DB/API Read Trace、実際のOS／AIへの開示 |
| [#2617](https://github.com/hat47x/tei-platform/pull/2617)／[#1124](https://github.com/hat47x/tei-platform/issues/1124) | 標準StudioのScript-first read-only/patch候補の合成Python 14/14 | 実Source変更、CUI＋Wailsの同版投影、意味受理 |
| [#2517](https://github.com/hat47x/tei-platform/pull/2517)／[#2500](https://github.com/hat47x/tei-platform/issues/2500) | ViewChange→型付きOperation候補のGo1.23.2隔離12/12 | 実Request束縛[#2598](https://github.com/hat47x/tei-platform/issues/2598)、実Provider確定、独立Read |
| [#2388](https://github.com/hat47x/tei-platform/pull/2388)／[#2387](https://github.com/hat47x/tei-platform/issues/2387) | SQL NULL/Bag/Join/Truthの独立SQLiteとOperator比較の局所ケース | 実TEI DSL、SQL Worker/Provider pushdown、権限・開示 |

同じ「12/12 PASS」でも別Fixture／別環境／別時点の検証である。上表の数を合算して製品E2E試験数や本番品質とみなさない。各PRは照合時Draft/Open。後続の更新があれば再取得して現在資格を確認する。

## 4. 一件の同一Scenarioで結ぶべきインターフェイス

最初の合成業務例は Order→Line→Allocation/Note の多階層、未MappingのPayment、非表示の項目を使い、次の証拠を**別々の列**に保存する。

| 段階 | 接続の入力→出力 | 重要な判別条件／担当 |
| --- | --- | --- |
| S1 Source | Markdown／軽量DSL／表／DOT→Source occurrence・Parser版・Source span | 引用／例示をFunctionとして自動実行しない。#2621/#988 |
| S2 Plain Data | Syntax Plugin→Record/Sequence/Ref/Expression等にエンコードされたデータ | 同じ見た目でもliteral/ref、scope/capture、Operator版が異なれば別。#2620/#988 |
| S3 Meaning | 意味Role→型付き参照・Operator契約・Acceptance候補 | Roleを記述できることと正式受理／実行許可を分ける。#2620/#1121 |
| S4 Function Read | Data Model Relation→明示Context Mapping→要求Read footprint | 取得宣言、実Request、物理Read Witness、Result Slice、Consumer開示を区別。#2506/#2622/#2516 |
| S5 Derived View | Result Slice＋Operator→2階層の集約・編集Projection | EntityとAggregate粒度、完全性、hidden/null、Snapshot/Selectionを保持。#2497/#2593 |
| S6 Editing | View delta→候補Intent→実Requestの対象・版・Action束縛 | 限定PATCH／Collection差分／多義集約、dirty Instanceを区別。#2500/#2597/#2598 |
| S7 Realization | Intent/Action→提供元の操作・Commit／Receipt／独立Read | Provider pathの候補≠Grant。OData batch≠当然のatomic性。#951/#1174/#2485 |
| S8 Consumers | 同じMeaning→CUI／Studio／SHO／外部Host | 実際に読み込まれたSource/Plugin/Role版と開示可能な範囲を個別に照合。#1124/#2516/#2390 |

**S1〜S8は既存課題を横断する受渡しの観点であり、新しい段階Gateや共通Runtimeではない。**

## 5. 新たに固定する反証ペア（統合ScenarioでNOT RUN）

| ID | 入力を一つ変える | 期待する診断・不変条件 |
| --- | --- | --- |
| HX-01 | Markdown表を引用へ移す | 表の構造が同じでも実行Roleの成立は別 |
| HX-02 | method-chainのEffect順だけ変える | 共通データShapeが似てもEffect/順序同値と判定しない |
| HX-03 | 秘密Fieldだけを変更する | 論理Projectionが変わらなくても物理Overfetch／開示資格は独立 |
| HX-04 | 同じSemantic IDへReplica/Primary/ODataをBinding | Provider実現候補と鮮度・write権限・適用時点を分離 |
| HX-05 | 取得はpage 1だけ、表示上は子/孫全削除 | 限定PATCHは条件付き可能でも全件Replace／削除へ昇格しない |
| HX-06 | 親選択A→B後に旧AのRead/Intentが到達 | Context Instance/epoch/Source/Action版を再照合 |
| HX-07 | 同倉庫の孫割当が複数行 | 集約文字列から一意なUpdate対象を推定せず明示Actionへ |
| HX-08 | 同じDataをCUIとStudioで表示、Source文書を移動 | Meaningを保持してもSource span/locatorの再解決が必要 |
| HX-09 | 同じ意味FunctionをSensor/Actorの異なる時間粒度へ | 10Hz→60Hzを暗黙補間せず時点・単位・適用範囲を説明 |
| HX-10 | 外部APIは部分成功後にTimeout | 再試行／結果不明／Receipt／Business Completionを区別 |

独立Expectedは既存[#2514](https://github.com/hat47x/tei-platform/issues/2514)のOracle契約を再利用する。Providerのログだけで物理Read最小性、独立再読取り、外部開示拒否をすべて保証したと表示しない。

## 6. 接合の進め方（成熟度に応じて並行、固定の開発上限ではない）

| 接合パケット | 具体作業 | 完了の資格／残す証拠 |
| --- | --- | --- |
| H0 同版の手動対応表 | 既存#2625構文例と#2624 Contextを同じSource/Role/IDへ写す。適用RoleのOwner判断待ちを記録 | exact Source/Plugin/Operator/Context版、入力⇔標準Data⇔Meaning候補の対応。**設計票** |
| H1 実TEI解析 | #988の現実のLanguage Plugin/World/Validatorから標準形とSource indexを採取し、合成fixtureとの差を診断 | 実CLI/Go対象・argv、解析結果、失われたSource情報、未解決。**製品読取り局所検証** |
| H2 実Dataアクセス | #2506/#2622/#2611のFunction MappingでQuery/Providerの要求と実Read Traceを照合 | declared/realized/observed/disclosedの四つを混同せず、Provider別の不足を記録。**実RDB/API資格は別** |
| H3 標準UI二Consumer | #2617のgeneric inspectをCUIとStudioの実画面に接続。Definition/Instance/Source occurrenceを追跡 | 同じMeaningの説明・不完全性・開示を比較。編集不能なViewを可視化 |
| H4 更新一往復 | #2500→#2598→#1129/#1174で限定PATCH＋追加/削除→独立再読取り。別に3層→2層の非一意更新を診断 | Host側Authority、実Request、Provider Effect、Receipt、独立Observed stateを区別 |
| H5 異質領域とAI | Goa風/MukuroL風/Markdown/チェーン＋Skillの比較、異質Domainへの適用、Target生成の再現性 | 意味修正成功率、未知語誤用、patch差分、token/修正往復、Core特殊分岐数を#2018へ返す |

H0/H1と異質領域の要件探索は並行可能。H2/H4の実Effect資格は既存#1119優先作業を追い越さない。新しいSchemaやParserが必要かは具体的不足から判断する。

## 7. 統合判定に必要な最小証拠

同一試行で接合したと主張する場合は、少なくとも対象となる範囲について、次を固定・参照可能にする。

1. Source occurrenceとSource revision、実読込byte／位置、元の表記種別。
2. Syntax Plugin・Role・Operator ID/versionと実ロード済みPlugin lock。
3. 標準DataのDigestと型付きMeaning/受理状態。Source byte一致・正規化構造一致・Meaning一致は別欄。
4. Function ContextのDefinition/Instance/selection epoch、Relation path/Fields、宣言Read footprint、Result Slice/scope/completeness。
5. Provider bindingの実経路、実Request、実Read Traceの観測有無、開示先。Requestの狭さと物理I/O/外部送信の保証範囲を区別。
6. View occurrenceと編集差分、元Entity/Relation ID、更新Action・expected version、現在のAuthorityと実Target。
7. Operation ID／Request hash／Commitまたは不明状態／Receipt／別Readで得た観測。業務完了はOwnerの意味判断で区別。
8. Test Expectedの独立出典、実行したコマンド・環境・Commit SHA、未実行箇所、Consumerに開示してよい診断。

必要な証拠がない段階はそのまま「未確認」と記録し、意味受理・実行許可・製品成功へ昇格させない。

## 8. 受入前の判断事項と修正先

| 判断 | 検討Owner | 誤った先行採用を避ける条件 |
| --- | --- | --- |
| 標準Data Normal Formの最小Carrier・Function記述形式とbootstrap | #2620/#988/#2495 | 既存World/Canonicalとの対応とRole競合・参照スコープを実証してから |
| 標準Authoring DSLの形式・Goa/MukuroL比較 | #988/#2621/#2018 | AIのfirst-pass-validだけでなく意味修正率・Source差分・人間の可読性を比較してから |
| Sourceへの逆投影／同一Markdownの説明と定義の境界 | #2621/#1124/#2497 | 編集可能なSource occurrenceと意味Roleの一意性が示されてから |
| SmartDB一般化の製品名とRuleの接続 | #2622/#1129/#951 | 新しい共通Host/Provider Engineを作らず既存Capabilityに適合することを示してから |
| 多階層更新とOData等の別提供元への移植 | #2500/#2597/#2598/#1174 | 同じUI/同じ型の一致だけで同じatomicity/Completionとしない |
| Studio/SHO/AIへの情報開示 | #1124/#2516/#2515 | read-onlyを外部送信許可や秘匿データアクセスと同一視しない |

本計画では要件探索の範囲をあらかじめ狭めず、正式仕様の採用と製品実証を各Ownerの判定へ委ねる。新しい設計提案の追加は可能だが、既存の根拠・Issue・Draftと同じ内容の複製は避ける。
