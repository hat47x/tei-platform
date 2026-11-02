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
| [#2624](https://github.com/hat47x/tei-platform/pull/2624)／[#2506](https://github.com/hat47x/tei-platform/issues/2506) | 限定Context Mappingのread-only合成Python 20/20とRule依存の追加18/18（報告上計38件）、未宣言領域の有限非感度 | 実Query/Provider取得Scope、実Observed completeness |
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


## 9. 他レーンで確認された追加境界（2026-11-02横断深化）

### 9.1 「Everything is Data」を厳密に扱う最小原則

**表現がデータで統一できること**と、**そのデータをどの意味で評価できるか**、**どの作用が許可されるか**は分ける。特にFunction・Data Rule・UI Condition・Agent Skillで同じOperator-likeデータが現れた場合、文字列／引用／実参照／Role付与／解釈Pluginの固定版／有効なCapabilityを区別する。

- *Authoring source*（人間の編集正本）→*surface decode*（構文Plugin、Source位置）→*plain-data candidate*（構造・参照・式をデータとして保持）→*semantic resolution*（型・Role・Operator・scope・effect・受理）→*realization/observation*。この並びは処理の証拠鎖であり、五つの実行エンジンを要求しない。
- 最小bootstrapは既存値／構造／安定参照／依存／宣言適用を起点とする**既存Worldとの対応検証**から始める。Role定義の自己再帰、同じRecordへの多重Role競合、Plugin版による意味変化は#2620/#2544で検査する。
- メタレベル・型／Facet・Relation出現・通常データの区別は、必要な情報を標準データ構造に保持し、通常のPlugin作者が特別な構文やメタ管理画面を意識する量を最小化する（#2495/#1126）。ただしRoleの自己記述可能性から信頼/正式承認/Host Grantを推定しない。
- SQL NativeやGraphviz表示など、表層がそれ自体のAuthoritative Sourceとして残る場合は、そのSourceを正本として保持する。明示的なImport・意味移管をOwnerが行うときだけ、DSLを編集正本へ切り替える（#2387/#988/#2497）。

### 9.2 「同じモデル」を比較するときの識別子・時間・対象集合

**意味ID、Relation occurrence、Context Definition、Function Instance、View occurrence、Provider route、Operation/Attempt/Receipt、業務確認資格**は別々の同一性。さらに時刻は`tSource`／`tRead`／`tSelect`／`tEdit`／`tBind`／`tCommit`／`tObserve`を必要箇所だけ明示する（#2597/#2598/#2582/#1125/#1174）。UI上の同じ行番号・ラベルや同じJSON値から、保存対象の同一性・現在性・確認有効性を結論しない。

集約した孫データやDerived Datasetを再集計・更新するための契約には、**集計粒度・単位・NULL/missing・bag/set/sequence・時点・対象の完全性**が必要。全件要求と実Providerのcomplete証拠、論理Resultの同値と物理Read footprint、読み取れることとAIへの開示可否を分離する（#1705/#1706/#2506/#2622/#2516/#2593）。

### 9.3 操作を一種類の「保存成功」に潰さない

**Function Definition／Instance → Editing Session → ViewChange → Typed Intent → Accepted Action → Driver Request → Host Grant → Provider Effect → Receipt → Independent Read → Business Completion**のそれぞれが別Ownerの資格。#2598のIB-01〜08をH4へ引き継ぐ。Intent候補に対象Keyがあっても、Requestを発行した時点の対象、選択Epoch、Scope、Rule revision、既存値、実Request body、権限を再照合する。

同じ受理済みMeaningに対する同じOperation IDでも、Provider/Mapping版・物理対象・Receipt再照会資格は変わり得る。特に`commitできたがACKを失った`、`commit不成立`、`再照会不可`、`過去Receiptは取得できたが現状は変更済み`を区別する。過去の業務確認・受領は値やSource byteが戻っても自動復活させない（#1125/#1174/#2597）。

### 9.4 Domainを超えた同一性は「抽象の使い回し」で測る

業務受注の3層更新と、Sensor/Actorの10Hz認知→60Hz運動、文書の親子章節などを、似たEntity名へ改名しただけのFixtureではなく、**異なる時間意味・関係・順序・作用・改版・Oracle**を持つ要求として比較する。必要になるCore特殊分岐、Role/Operator追加、Plugin適用、Studio専用UIの増分を#2018へ記録する（#2495/#2493/#2507）。

## 10. 接合パケットへの成熟した追加受入（実行は全件NOT RUN）

既存のHX-01〜10を残したまま、複数チャットから集まった**観測可能なギャップ**を以下の差分として追加する。

| ID | 反例・変化 | 期待／必要な根拠 | Owner／受渡し |
| --- | --- | --- | --- |
| **HX-11** | `role: function`を説明用Markdown／引用／AIのTool説明内へ置く | Source occurrenceとMeaning Roleが別であり、実行・開示権限へ昇格しない | #2620/#2621/#2515/#2605 |
| **HX-12** | 同じFunction名／Data StructureだがOperator Pluginの版を変更、または局所変数captureが変化 | 正規化データ候補が同形でも意味資格再判定。実Runtimeに旧版を黙って適用しない | #988/#2620/#2544 |
| **HX-13** | 受注明細2行と独立配送3行のJOINで6表示行 | Entity更新対象は元IDに帰属し、6件更新やSUMの6倍誤計上を防ぐ | #2387/#2506/#2598 |
| **HX-14** | 画面内のFieldがmissing/null/0/false/空文字、外部APIはField Projection不可 | 編集候補のClear/Replace/Patch、取得不明・取得済みNULL、物理Overfetch、外部送出を別に説明 | #1706/#2622/#2516 |
| **HX-15** | 同じFunction Patternを2箇所に配置、選択A→B→A、旧Readが遅着 | Instance/Dirty/anchor/epochを保持。旧結果で現行表示や未保存変更を上書きしない | #2597/#2582/#2506 |
| **HX-16** | 数量/計算値が一度A→B→Aへ戻る、業務確認が撤回される | 数値・byte・意味一致と、確認の現在有効性を分離 | #1125/#2514/#1174 |
| **HX-17** | 10Hz観測値を60Hzの制御入力へ接続、Absolute temperatureとDeltaを同じ数字で表す | 時間粒度/補間/量・単位/時点を明示し、型名・数値のみの同値を拒否 | #2495/#1705/#988/#2507 |
| **HX-18** | 同じ確定データをSHO設計書、Studio、CUI、外部Harnessへ投影 | 同じSubject/Meaningでも、編集/引用/説明/開示/実行資格とSource再解決は用途ごとに照合 | #2621/#1124/#2516/#2605 |

### 10.1 H0〜H5の進め方に追加する具体的引継ぎ物

- **H0→H1**：同じ定義の`authoring occurrence／Syntax Plugin＋版／plain Data digest／Role＋Operator ID＋版／Semantic ID`対応表。正規化データの一致と受理Meaningの一致を別の検査項目とする。AIは既存の`tei commands`、`model catalog`、`inspect`、`trace`、`validate`で取得できる範囲を#2605のProduct Skillから確認する。存在しない`tei role inspect`等のコマンドは実装済みとして呼ばない。
- **H1→H2**：Function Contextの`declared footprint／resolved relation path／actual request／provider access witness／result slice／consumer disclosure`を、同じSource版・Context Instance・Scope/epochへ紐付けた照合票。必要な内部JOINキーをすべて違反とみなさず、根拠・用途と開示先を明示。
- **H2→H4**：`View occurrence/Source entity ID/Relation occurrence/dirty before-after/Action revision/target resolution time/expected row version`の変更票。とくに部分集合から全件Replaceへ変えるときのPolicy、集約→元行への割当規則、RowIndexの非同一性を記載。
- **H4→H5**：実Hostに届いたRequestと`Operation ID／Attempt／Receipt lookup outcome／independent observed state／Business Completion claim`の対応。成功、拒否、競合、部分確定、結果不明、回復後の結果が区別されること。
- **別Domain／AI評価**：生成AIへ供給した`Grammar/Schema/Operator＋Plugin pins/正負例/実CLI discovery`、提出Patch、Diagnostic、手動修正数、Meaning違反、Test Expectedへの変更履歴、token費用を#2018で同一Task・同一予算で比較する。構文の短さより**独立Oracleと意味保持を満たす総費用**を評価。

### 10.2 現在の製品と開発成果の資格

- H0の対応表・HXのExpectedは**計画/設計候補**であり、製品Parserや標準Data normal formの採用を表さない。#2625等の限定Fixture成功を正式Plugin/Target/GUIの成功とみなさない。
- 開発スピード向上・並列AI作業を前提に、異質Domainや適合候補の探索は広げられる。統合時には各OwnerのSource・現在のPlugin版・独立Test・実Target/Provider/Host・Owner受理という必要な資格を別欄へ残す。
- TEIの新しいメタ構造やRule設計は標準Pluginと既存構造の合成から検討し、第二Canonical/第二Host/中央Registry/共通Txエンジンを前提にしない。特定の抽象や命名の採用は、前後の比較・不成立例・撤回条件から判断する。


## 11. Graphvizを文脈に合わせて利用する共通変換部品（2026-11-02追補）

**設計判断候補**：Graphviz/DOTを単なる標準データ構造の直書き記法に限定せず、**特定の文脈に適用する共通部品を介して標準データ構造の形を変換できる**ようにする。入力・出力ともplain Dataとして表現し、部品の適用条件と変換規則もデータとして宣言する。同様の部品方式はMarkdown表、関数表記、メソッドチェーン、他の図・文書形式にも適用できる。

### 11.1 変換の二つの経路

| 経路 | 内容 | 資格 |
| --- | --- | --- |
| **構文のdesugar** | DOT構文 → ノード/Edge/属性などの**構文上のGraphデータ** | Graphviz形式を解析できた。ドメイン意味Roleはまだ未確定 |
| **文脈付き共通変換** | 文脈（Workflow／Entity Relation／Function Dependency等）の受理済みRole・Mappingを適用し、Graphデータ ↔ ドメイン上の標準データ構造を変換 | 用途、写像、Source/Target shape、Identity/Relation、保存条件・欠落が解決された範囲で意味を主張 |

表層としてのGraphvizを使いたい場合には第一経路で足りる。特定文脈の図として利用し、その文脈のStructureをGraph形状へ変換する場合には第二経路を利用する。**どちらも標準データ構造に対する処理であり、新しいGraphviz専用Canonical／特別なStudioモデルを追加しない**。

```text
TEI標準データ（例：Function Workflow / Entity Relations / Dependency）
   │
   ▼
共通変換部品（用途Role + Map + 契約 + 固定版）
   │
   ▼
Graphの標準データ（Node, Edge, Attribute, Occurrence）
   │
   ▼
DOT Syntax Plugin / Graphviz Renderer
   │
   ▼
図示／構造化DOT Source
```
逆方向は「DOT parse → Graphの標準データ → 文脈付き変換部品 → Domainの標準データ候補」。**解釈候補、Source編集、意味受理は別**。DOTに十分な属性・識別子がない場合は図示／部分構造のみとし、勝手にFunction、Relationや更新Intentを補完しない。

### 11.2 共通変換部品の最低契約（新しいCore固定Schemaではない）

- **用途**：`contextRole`（Function依存図、ER関係図、Workflow、状態遷移など）、Source/TargetのShapeと必要Facet、Plugin/Rule ID＋版。
- **参照・束縛**：どのSemantic Subject／Relation occurrence／Operator、どのFunction Data Context/Snapshot/Scope／Source occurrenceに基づくか。Graphvizのnode名・edge表示順だけでDomain IDを認定しない。
- **変換**：node/edge/port/subgraph/attributeの対応、方向・多重度・グループ粒度、順序・循環・NULL/欠落、表示上の演算とDomain上のRelationの違い。Domain Roleで必要な情報が不足すればunresolved。
- **構造保存性**：Identity、Cardinality、Graph edge type、単位/時点/依存順序、完全性、出典の保存・損失を宣言。レイアウト・色などの表示属性と意味属性を区別。
- **編集資格**：read-only display／一意Sourceへ戻す限定編集／明示Actionが要る変更を分類。Graphの辺を削除したことはEntity削除・Relation unlink・Workflow遷移の削除のいずれとも自動同一視しない。GetPut/PutGet等の法則は適用条件が満たされる範囲だけ検証。
- **合成と可用性**：共通部品を既存Plugin/Operator/Transformation契約で合成する。別UI・Graphviz Library・Native Rendererは交換可能。外部ツールの図示失敗と意味データの無効は別。

### 11.3 文脈ごとの違い（同じDOT有向辺に異なる意味）

| 文脈 | 元データ | Graph化の規則 | 編集時の留意点 |
| --- | --- | --- | --- |
| 処理依存 | 演算と引数の関係 | 辺を依存先→利用元へ | 辺を逆転すると評価依存が変化。Effect順序は独立保持 |
| データモデル | EntityとRelation occurrence | 辺を関連、属性をcardinality等へ | Relation削除≠Entity削除。複数親・多重辺・役割を保持 |
| Workflow | 状態・遷移・Guard・Action | nodeを状態、edgeを遷移として投影 | 状態遷移Guardや業務作用がDOTラベルだけでは失われ得る |
| 文書構成 | 章・節と参照関係 | 親子と参照を別edge kindへ写像 | 図上の再配置≠章の移動。順序や原文Sourceを保持 |

### 11.4 既存受入への追加（全NOT RUN）

| ID | ケース | 期待・反証 |
| --- | --- | --- |
| **GX-01** | 同じDOT `a -> b`を依存／データモデル／Workflowの3 Roleへ適用 | 同じEdge構文でも意味Roleが異なれば別の意味。文脈なしでは候補のみ |
| **GX-02** | Typed DOTからWorkflow/Entity-Relationそれぞれの標準データを構成 | Context/Stable ID/Version/Role条件が一致する範囲だけ成功 |
| **GX-03** | 出力DOTのrank/color/座標のみ変更 | Domain Meaning不変、Source presentationの差分は保持 |
| **GX-04** | 本来のGraphが複数辺・複数親・Cycleを持つ | Tree的な一意親に勝手に変換せず、失われる情報を説明 |
| **GX-05** | 図からEdgeを編集するが元データにはNode ID/Relation occurrenceがない | 更新候補はambiguous/read-onlyであり、破壊的削除へ変換しない |
| **GX-06** | 依存図でEffect chainの順序だけ変える | Syntax shape一致を理由に同じFunction意味へ正規化しない |
| **GX-07** | Graph→DOT→Graphの往復で属性・未表示ノードが欠落する | Lossを明示し、Sourceへ限定的にしか書き戻さない |
| **GX-08** | 同一変換部品をCUI/Studio/SHOで再利用 | 各Consumer固有のRenderer/開示は分離し、Core部品の重複追加を検出 |

**Owner割当て**：構造間の保存条件・逆写像＝[#2497](https://github.com/hat47x/tei-platform/issues/2497)、標準データと意味Role＝[#2620](https://github.com/hat47x/tei-platform/issues/2620)、DOT/表層解析＝[#988](https://github.com/hat47x/tei-platform/issues/988)、Graph/Tree/UI表示＝[#997](https://github.com/hat47x/tei-platform/issues/997)/[#1124](https://github.com/hat47x/tei-platform/issues/1124)、Markdown出典＝[#2621](https://github.com/hat47x/tei-platform/issues/2621)。本[#2619](https://github.com/hat47x/tei-platform/issues/2619)のH0→H1（構造/Role/変換契約）・H3（標準Studio/CUI）へ引き渡す。別のGraphviz専用EditorやSyntax正本、独自Hostを前提にしない。


## 12. 高階Data Operatorによる条件・集約のデータ化（2026-11-02追補）

### 12.1 目的：生成AIが書く命令的分岐の量と局所的な意味の重複を減らす

**追加要件**：生成AIが`if { ... }`のコードブロックを大量に生成する代わりに、`filter`、`map`、`flatMap`、`groupBy`、`aggregate`、`any`、`all`、`partition`、`window`、条件選択などの**高階Operatorをデータセット・項目・Queryの各粒度で合成**できるようにする。これらは最終的に標準データ構造の`operator + inputs + bindings + typed arguments`に正規化される。メソッドチェーン、Markdown表、インデントDSL、関数呼出しはその糖衣である（#988/#2620）。

これは「条件が消える」という意味ではなく、**条件を評価可能なデータとして明示し、処理のまとまりを再利用できる**ようにする方針である。例外処理、権限、トランザクション、状態遷移、Effectを安全に扱うために必要な分岐は意味として残し、Targetが必要なら通常の条件分岐としてLoweringする。生成コードの`if`総数だけを品質指標にしない。

### 12.2 三つの適用レベルと統一性

| 粒度 | 入力の例 | 関数的な組成例 | 重要な契約 |
| --- | --- | --- | --- |
| **Dataset** | 明細集合／在庫割当集合／Event Stream | `filter(predicate)` → `map(derive)` → `groupBy(key)` → `aggregate(measure)` | 集合のscope/grain/bag/sequence/完全性/版・結果Identity |
| **Field / Column** | 同じEntityの数量列・単価列、optional項目 | `map(value => ...)`、`select(condition, yes, no)`、`coalesce`、`zipWith` | 列の元Entityキー・位置対応・型/単位/NULL・broadcast条件 |
| **Query / Context** | Relation path＋明示された要求集合・クエリ候補 | `where`、`selectFields`、`groupBy`、`aggregate`、`orderBy` | Queryを単なる文字列ではなく標準Data上の式として保持し、Provider pushdown/Local評価の同値性・Read footprint |

**共通**：どのレベルでもOperator ID＋版、型付き引数、入力・出力Shape、Pure/Effect、評価順・束縛Scope、Lineage/diagnosticsを保持する。フィールド演算を列へ自動拡張する際、同じ行indexだけによる別Entity同士の`zip`や、`many→scalar`の暗黙変換は禁じ、Key/Grainの対応を契約とする。Query演算での外部Predicateは公開結果の列以外を参照し得るため、Read依存・Providerの物理Read・Consumer開示を分離する。

### 12.3 人間向け構文の候補（**採用前の擬似DSL**）

```text
dataset activeLines
  source: order.lines
  select: quantity > 0
  derive lineAmount: quantity * unitPrice

dataset totalsByTax
  source: activeLines
  group: taxClass
  measure total: sum(lineAmount)
```

同じ計算をチェーン表記で書く場合：

```text
order.lines
    .filter(line => line.quantity > 0)
    .map(line => { taxClass: line.taxClass, lineAmount: line.quantity * line.unitPrice })
    .groupBy(row => row.taxClass)
    .aggregate(total: sum(row.lineAmount))
```

これらは見た目を比較する記述例。**Data Contextの`order.lines`取得範囲、Quantity/Moneyの意味型と丸め、演算のResult Grain、集約に必要な完全性**は、別途解決された契約が必要。末尾の集計結果を明細Entityへ自動逆写像して更新可能とはみなさない。スタンドアロンQueryやフィールド単位にも同じOperator IDを使い、`selectFields`による取得列と`filter`の秘密依存列の違いを可視化する。

### 12.4 分岐をデータとして宣言するパターン

| 目的 | 推奨するデータ表現候補 | 意味を保つ条件 |
| --- | --- | --- |
| 行を残す／除く | `filter`、`partition` | predicateのTruth/NULL/Unknown・部分集合扱い |
| 一項目の条件付き導出 | `select(test, valueA, valueB)`、`match` | lazy/eagerの評価契約、型・単位整合、未評価側のEffect禁止 |
| optionalな値の変換 | `option.map`、`coalesce` | missing/unfetched/NULL/zeroの違いを保持 |
| 正常値／エラーの分岐 | `result.map`、`andThen`、`recover` | Validation NotSatisfied・評価不能・権限拒否・Provider失敗を単一booleanに潰さない |
| 集合に存在するか／全て満たすか | `any`、`all`、`none`、`count` | 完全性／単調に確定可能な片側結論を区別 |
| 業務の状態による操作 | 型付きActionの`guard`／適用条件の表 | Data RuleとAuthoritative Action、現在のHost Grant/実Effectは別 |

`select`等の名前は説明用候補。**条件評価は高階Operatorの引数としてデータ化されるが、Effectを伴うActionの実行分岐を純粋`map`の副作用へ隠さない**。複数のGuardや分岐順序が意味を持つ場合は優先順位・網羅性・衝突を別途検証する。

### 12.5 Operatorの意味契約・Provider実現の制約

1. **入力資格**：shape、scope、grain、relation/cardinality、集合のbag/set/order、結果のcomplete/partial/unfetched、時点/selection epoch、null/missing/unknown、QuantityやCurrencyを保持。
2. **演算法則**：純粋性・決定性、述語のTruth policy、短絡/評価順、関数引数のlexical scope/capture、`map`と`filter`の交換が常に合法ではないこと、`filter`と`limit`の順序非同値、`groupBy`後のgrain、順序依存のfold等を明記する。
3. **集約の特別条件**：`sum`/平均/Distinct/Windowの合法な再集計軸と中間状態（sum/count等）、JOIN fanout、空集合、NULLのみ、roundingの段階を保持する。完全性が不足した`sum`を確定値へ昇格させない。部分Readで`any=true`を目撃した場合と`any=false`を主張する場合、`all=false`の反例がある場合と`all=true`を主張する場合を区別する（#1706/#2593）。
4. **Query pushdown**：Providerの`filter/aggregate/project/window`対応だけでなく、対象OperationごとのTruth/NULL、bag/order、predicateへの隠れたRead依存、権限・RLS、時点、結果完全性が同等かを比較する。Local演算に移す場合にも秘密Fieldが外部Consumerへ渡るとは限らず、必要なHost／Provider権限と開示先を明示する。Native SQL原文の正本は明示的Importまで維持する（#2387/#2622）。
5. **更新・副作用の外側**：`filter`や`map`から、取得・候補Intent作成はできても、それだけで保存許可・実Action／Provider呼出しの実効権限を得たことにならない（#2500/#2598/#1129/#1174）。
6. **Target中立の意味**：表層DSL→Data→意味Role/Operatorの解決を一度行い、Java/.NET/Wasm等の各TargetへLoweringする。生成コードが内部で`if`/ループを用いても、契約を満たすならDSL要件には適合する。

### 12.6 Headless受入圧力（**すべて新規NOT RUN**）

| ID | 反例／入力 | 必要な判別 |
| --- | --- | --- |
| **HF-01** | 表／チェーン／軽量インデントDSLで同じ`filter→map→groupBy→sum` | 固定Operator ID/版・scope/型/grain解決後の意味一致、Source位置別管理 |
| **HF-02** | 同じ行index・違うEntityのQuantity/UnitPriceフィールド列を`zipWith` | 行index一致だけで対応とせずKey/Relation/Grainを要求 |
| **HF-03** | `filter(p).take(10)` と `take(10).filter(p)` | 一般に非同値。Provider Pushdownで無断交換しない |
| **HF-04** | `any/all`を部分取得・未知PredicateのDatasetへ適用 | witness/counterexampleによる片側確定と否定/全称の不確定を区別 |
| **HF-05** | SUMを先に丸める/最後に丸める、JOIN fanout、後段再JOIN | `sum(map(round))`≠`round(sum)`と再集計Lawを維持 |
| **HF-06** | `select`の未選択枝に更新Action/失敗式 | lazy/eager・Effect禁止／評価不能の条件を診断 |
| **HF-07** | 表示項目は公開、`filter`が秘密Fieldを参照 | Rule/Query依存・実Read/Result件数による漏えい・Consumer開示を区別 |
| **HF-08** | Query OperatorをSQL/RDB、OData/API、local評価へ配置 | 型・集合・NULL/Truth・順序・集約結果が同じ条件だけPushdown。能力不足の明示 |
| **HF-09** | 同じRuleを行単位、Dataset単位、Function Context単位で適用 | Bind対象・Grain・Scope／同一フィールド二重導出を追跡、暗黙クロス結合を防ぐ |
| **HF-10** | `result.map`で技術失敗を業務拒否やfalseへ変換 | 失敗分類とSource-located Diagnosticを保持し、Action/Authorityが変化しない |

**担当・再利用**：Operatorの意味とDSL sugar＝[#988](https://github.com/hat47x/tei-platform/issues/988)／[#2620](https://github.com/hat47x/tei-platform/issues/2620)、Dataset/Field/Queryの適用・Provider pushdown＝[#2622](https://github.com/hat47x/tei-platform/issues/2622)／[#2387](https://github.com/hat47x/tei-platform/issues/2387)、集合意味＝[#1706](https://github.com/hat47x/tei-platform/issues/1706)、集計法則＝[#2593](https://github.com/hat47x/tei-platform/issues/2593)、入力Context＝[#2506](https://github.com/hat47x/tei-platform/issues/2506)、Action/Effect＝[#1129](https://github.com/hat47x/tei-platform/issues/1129)／[#1174](https://github.com/hat47x/tei-platform/issues/1174)。HF-01〜10を[#2619](https://github.com/hat47x/tei-platform/issues/2619)のH0/H1/H2/H4/H5へ受渡す。**新しいquery runtime、dataframe engine、第二Core AST、Script if-removal transpilerは提案しない**。
