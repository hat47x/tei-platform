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


## 13. 高階Operatorの書換え条件と検証母集団（2026-11-02、他レーンSV・QX・Rule依存の接合）

### 13.1 成熟判断：`filter`と`validate`と`authorize`を別契約として扱う

高階Operator中心のTEI Scriptでは、生成AIが`if`文の繰り返しを避け、Dataset／Field／Queryの宣言・組成へ寄せられる。しかし**見かけのコード量が減っても、元の対象集合・評価失敗・副作用・秘匿境界の意味を変えないこと**が優先である。最新の[#988 SV-01〜06](https://github.com/hat47x/tei-platform/issues/988#issuecomment-6577015922)（隔離Python 6/6 PASSと報告）、[#2622 DO-07追補](https://github.com/hat47x/tei-platform/issues/2622#issuecomment-6577028330)、[#2586 HJ-SV1/2](https://github.com/hat47x/tei-platform/issues/2586#issuecomment-6577037688)の検討を、横断受入へ受け渡す。

| 意味 | 操作のイメージ | 不変条件／戻り値 |
| --- | --- | --- |
| **Select** | `lines.filter(q > 0)` | 元Datasetのうち条件に合う行集合を派生。除外された行は「妥当」と証明されていない |
| **Validate** | `validateAll(lines, q > 0)` | **検証対象Scopeの全行**に対するルール判定。NotSatisfied／Unresolved／技術失敗を分け、必要なら違反箇所を記録 |
| **Choose／Derive** | `value.match(...)`／`option.map(...)` | 1値やVariantを条件に応じて導出。条件優先度／未選択枝評価／NULL・missingを契約化 |
| **Authorize／Act** | 受理済みActionへGuard/Contextを束縛 | 操作の対象・現時点のGrant・Effect/Receiptを別検証。表示側のSelect/Validate PASSから許可を発行しない |

**反例**：提出母集団が`L1(quantity=2)`、`L2(quantity=0)`の場合、`filter(q>0)`の結果が`[L1]`になることと、元`[L1,L2]`の「数量は全件正」の業務検証が失敗することは両立する。どの集合を業務上の判定対象とするかは受理済みRule/Actionが決める。

**失敗の反例**：`visible`がfalseかつ`amount="BAD"`の行H、trueかつ`amount="10"`の行Vに対し、`filter(visible).map(parseAmount)`と`map(parseAmount).filter(visible)`は表示値だけ同じでも失敗が異なり得る。どちらの行を**実際に読む／解釈する資格があるか**も、表示演算の結果では決まらない。

### 13.2 Operatorの「意味同値」の観測範囲を広げる

`map/filter/flatMap/groupBy/aggregate/any/all/partition/window`等の組換え、SQL・ODataへのPushdown、ローカル評価、増分結果の再利用について、単純な結果値一致ではなく次の**比較列を既存受入票に付す**（新固定Schema、第二Optimizer、別Gateの追加ではない）。

- **入力と判断の根拠**：Semantic Subject、Operator ID/version、lexical scope・capture、Function Context Mapping、Rule依存／Action事前条件、評価対象Cohort／Result grain／時点、null/missing/absent/partial。
- **観測される計算**：Resultの値・Identity・順序・bag multiplicity・集約法則・丸め時点、Error/Diagnostic/Warning・確定／未解決・短絡するかどうか。
- **アクセスと開示**：実Provider/Query、Field・Predicate依存（公開出力にない隠れたReadも含む）、実Request/Read Trace、Consumer別の開示と結果の有無・件数などによる間接情報流。
- **作用と資格**：Pure/Effect分類、Action/RequestとHost Grant、Tx境界、Operation/Attempt、Receipt、独立Result観測。Operatorの最適化でEffect・失敗・確認資格が変わる場合は意味保存を主張しない。
- **再利用の依存**：表示D、Validation V、Action事前読取りA_data、更新対象A_write、実権限A_exec、Provider routeと実Read Cohortを別の依存として記録。表示が不変でもVだけ変わればRule再評価が必要（[#2600 IB-18〜21](https://github.com/hat47x/tei-platform/issues/2600)、[#2582](https://github.com/hat47x/tei-platform/issues/2582)）。

標準的なQuery DSLを高階関数として利用することと、すべての演算順序を交換してよいことは別。特に`filter`↔`take`、`validate`↔`filter`、`map(parse)`↔`filter`、`sum(map(round))`↔`round(sum)`、`groupBy`後のJOIN/再集約は、必要な不変条件の説明と独立Oracleがある範囲のみ同値候補とする（[#2387](https://github.com/hat47x/tei-platform/issues/2387)、[#2593](https://github.com/hat47x/tei-platform/issues/2593)、[#1706](https://github.com/hat47x/tei-platform/issues/1706)）。

### 13.3 新しい横断受入ペア（全件**本PRではNOT RUN**）

| ID | 同じ高階Operatorを使った二つの処理 | 正しい判別・受渡し |
| --- | --- | --- |
| **HO-01 Select vs Validate** | 元Scope全件検証と`filter`後の検証 | 除外行の違反が消えない。#988→#2586 |
| **HO-02 Failure Observation** | `parse→filter` と `filter→parse` | Result値だけでなくError・Diagnostic・読取り根拠を比較。#988/#2622 |
| **HO-03 Hidden Rule Dependency** | 表示Mapは同じ、検証依存のcredit_limit 700→500 | 表示再利用が可能でもRule/Action判定の再評価は別。#2624/#2600 |
| **HO-04 Query Predicate Leakage** | `select(id)`は同じ、秘密Fieldのwhereが件数を変える | 必要Read、実アクセス、Consumerが得る件数・存在情報を分離。#2387/#2516 |
| **HO-05 Page Cohort** | Page1後のWriter Commit、Page2をKeyset/OFFSETで追加 | 見かけ上全件でも同一実Snapshotを証明できなければ全称・SUM/TopNは未確定。#2622/#2586 |
| **HO-06 Partial Quantifiers** | 一部分だけ取得した`any`/`all` | witnessに基づく片側確定と、全件性が必要な反対側の未解決を区別。#1706 |
| **HO-07 Operator Fusion/Pushdown** | `filter→map→aggregate`をSQL/Provider/localへ配置 | 値・Failure・Row identity・bag・Read footprint・Scope・開示・Effectの維持を必要範囲で比較。#988/#2622 |
| **HO-08 Same View, Different Grant** | Operator出力/画面が同じ、Action Grantが失効 | 更新Intentの候補表示と実Host実行資格を区別。#2598/#1174 |
| **HO-09 DSL Authoring** | Goa風/MukuroL風/Markdown表/チェーンで同じOperator列 | Source忠実度は別、Role/Operator/version/scopeが同じときだけMeaning同義。#2620/#2621 |
| **HO-10 Non-business Domain** | Sensor/Actor時系列の`filter/window/aggregate`と受注集合の同形DSL | Window時点/遅延/単位、Relation、Rule/Action、必要専用Pluginの差を測り、Core分岐数を記録。#2493/#2018 |

[Draft #2629 §12](#12-高階data-operatorによる条件集約のデータ化2026-11-02追補)のHF-01〜10は基本Operator契約、本§13のHO-01〜10は**同一Scenario上での検証対象・失敗・開示・実Provider・再利用の境界**を検査する追加列であり、同じ試験の二重カウントを行わない。

## 14. 既存成果を一つの実TEI経路へ接続する次の実務パケット

ここまでの独立Draft群の成果は、各担当が維持する。横断Ownerは追加の抽象を増やすよりも、**一つのSourceから同じ意味／Contextを実TEIで解決できるか**を調べる。

| パケット | 最初に確認する実経路 | 終了資格／不成立時の記録 |
| --- | --- | --- |
| **C0 語彙とSource固定** | #2625の演算列と#2624のOrder/Line Context、#988の実Language Plugin、#2605の実在CLIコマンドを手動対応させる | Source/Operator ID・version/Role/Context・独立Expectedと、未対応Mappingを固定。**対応表のみ** |
| **C1 実解析・意味解決** | 既存`tei commands`／`model catalog`／`inspect`／`trace`／`validate`の実利用範囲で解析結果を取得 | どのSyntax/Role/Operatorが正式TEI Worldまで届くかを観測。未実装コマンド名をTool説明から呼ばない |
| **C2 高階Operator＋Data Context** | 一つのDatasetについて`filter`と`validate`の違い、Rule依存列、部分取得、Query→Data Context、Provider Readの期待/実の差を照合 | ResultだけでなくError/Diagnostics、入力Scope、実Read Trace、Target/Provider版、開示を記録。別のQuery engineは作らない |
| **C3 標準UI／Source往復** | [#2631](https://github.com/hat47x/tei-platform/pull/2631)のopen-world標準Studio/未知Role表示と[#2617](https://github.com/hat47x/tei-platform/pull/2617)のScript-firstを、同じMeaning候補に適用 | 表示できたこと、編集可能なSourceに一意に戻せること、Role未認識時の退避を別観測。実Studio未接続なら明記 |
| **C4 Actionと実Effect** | #2500→#2598→#1129/#1174の実Request/Receipt/独立Readへ、同じContextとOperator-derived Intentを接続 | 実Grant、選択epoch、更新前提とOperation Identity、Provider部分確定を照合。局所Intent PASSをDB保存PASSにしない |
| **C5 改版・AI・異質Domain** | #2600の増分再利用・#2018の生産性測定に、同じScenarioのSyntax-only変更、非表示Rule改版、Provider切替、Sensor/ActorのWindow変更を登録 | Meaning変更・必要再評価・生成AI修正diff・token/所要時間・Core特殊分岐数を比較。異Versionの過去PASSを流用しない |

**優先順位**：既存#1119のP0-a/P0-bとOwner受理を維持し、C0〜C3はread-onlyの実証を平行可能にする。C4はその既存実行資格の成立後に実施する。C0〜C5は新Gate/Runtimeではなく、[本計画H0〜H5](#6-接合の進め方成熟度に応じて並行固定の開発上限ではない)の具体的受渡し作業である。

**現時点の証拠状態**：#988 SV-01〜06のPython6件、#2631のV8 16 assertions、#2627のGo32件、#2624のPython20+18件、SQL/SQLiteの局所実測は各元Issue/PRの独立報告である。このPRで新たにそれらを実行、同一Source・同一World・同一ProviderでE2E接続したわけではない。PR #2629は引き続きDraft。正式Operator API、Data Normal Form、生成Target、Host Grant、Owner意味受理の採用はそれぞれの担当判断に委ねる。


## 15. Operatorの代数則と増分評価の適合条件（2026-11-02、他レーンとの最終接合候補）

### 15.1 追加の成熟判断：高階Operatorの利点を、正しさが検証できる範囲で最大化する

Everything is Data＋高階Operatorは、`if`を記述し直すためだけの抽象ではない。同じ`filter/map/groupBy/aggregate/window`が、**生成AIによるDSL構築、Data OperationsのProvider Pushdown、Function Data Contextの再評価、増分ビルド、別Target生成**に利用されるなら、各段階で変換を正当化する**Operator固有の条件付きLaw**が必要になる。

本節でいう`Law`は標準データ構造で**意味部品の契約を記述する属性／関係の候補**であり、新しいLaw Registry、Solver、Optimizer、DSL Core型の採用ではない。#988のOperator意味・固定版、#2495のFacet/量、#2497の構造保存、#2593のMeasure、#2582の反応的Context、#2600の再利用資格、#2018の生成AI生産性へ**条件・反例・証拠**を引き渡す。

### 15.2 変換時の最小判断票（新製品APIではない）

| 比較対象 | 具体例 | 安全と主張するために必要な条件／検出すべき誤り |
| --- | --- | --- |
| **構文の置換** | `ds.filter(p).map(f)` ↔ 宣言表・Goa/MukuroL風定義 | 同じ型付きOperator、変数束縛、評価順、純粋性、Scope/参照版、Source出典。構文一致だけで意味を採用しない |
| **map融合** | `map(f).map(g)` ↔ `map(g∘f)` | f/gの引数束縛、決定性、部分関数・失敗や診断、Effect、観測順序、Source追跡が変わらない範囲。値同値だけでは不足 |
| **filter移動** | `filter(p).map(f)` ↔ `map(f).filter(p')` | p'が写像後の値に正確に対応し、元Scope、NULL/Unknown、Partial failure、アクセス/開示、Effectが同じ条件。pを同じ文字列で移せばよいわけではない |
| **filterとlimit** | `filter(p).take(n)` ↔ `take(n).filter(p)` | 一般に不成立。順序と上限までのRead、Queryの隠れた依存・rowset完全性を記録 |
| **aggregate分割** | `sum(whole)` ↔ `sum(sum(partition_i))` | 分割集合が同じCohortを過不足なく一度ずつ覆い、SUMの精度・丸め・NULL/欠損/空集合・型・オーバーフロー・並び/決定性が成立。平均・比率は原則として中間状態（sum/count等）が必要 |
| **量化／単調性** | `any(p)`, `all(p)`, `none(p)` | 未完全集合でも存在証人・反例証人による片側判定は可能。要素削除、Predicate変更、Cohort/Scope改版で過去の証人が無効化し得る |
| **Window/Stream** | イベント追加・遅延訂正と`window().aggregate()` | event-time/processing-time、watermark、暫定/確定、late arrival/retraction、単位・Window境界、順序と再通知資格を保持 |
| **構造投影** | 3層→2層、Graph→Tree、Column→Dataset | Identity、Relation occurrence、grain、order/bag/completeness、Source span、編集可能範囲を明示。集約Viewからの保存は別Actionに委譲 |
| **Provider配置** | SQL/RDB、OData、ローカル評価 | OperatorのTruth/NULL/精度/Failure/行存在/評価Scope/Access/開示とProvider実証Capabilityが合致。物理Read最小性と出力一致は別 |
| **増分reuse** | predicateやRule入力だけ変更 | Actual consumed dependency + Source/Operator/Role/Context/Provider/Target版＋Oracleの現在資格に基づき、**省略可能なStageだけ**reuse。意味結果同じでもSource Fidelity/SHO/実Grantは再確認 |

**Lawの説明に用いるデータ項目（未採用）**：`operator/id/version`、`source/target shape`、`scope/grain/bag/order/time`、`preconditions`、`observable outputs + diagnostic/failure + effect + access/disclosure footprint`、`counterexamples`、`qualifying evidence`、`target/provider constraints`、`source provenance`。既存Meaning/Plugin/Scenarioの情報へ参照でつなぎ、独立のLaw正本や実行許可を作らない。

### 15.3 適用可能性と増分再計算は別の問い

同じOperator定義の`filter`・`aggregate`を再利用する場合、**(A)演算が入力Shapeに適合すること**、**(B)特定の書換えLawが成立すること**、**(C)変更差分だけで結果を更新可能なこと**、**(D)その結果を現在のContext/Consumerに利用してよいこと**を別々に判定する。

- `filter`は追加行・修正行・削除行それぞれのIdentityとPredicate依存を追えれば差分更新候補を作れるが、部分Resultの全件性は差分更新だけで確定しない。
- `sum`は重複を含む入力集合を修正・削除する場合に、元の寄与元ID・入力の正確な数値・丸め時点・Scope/時点を保持する必要がある。先行集約値をあとからJOINして再SUMする変換は[#2593](https://github.com/hat47x/tei-platform/issues/2593)の反例に照らして判断。
- `any`の過去のtrueはWitnessが削除・Scope変更されると失効し得る。`all`の過去のfalseも違反Witnessの削除・Predicate改版で失効する。**単調性は入力変化の許容集合に相対的**。
- `not exists`／全称／TopNは、取得ページ・Effective Snapshotが異なると一つの真偽や順位に確定できないことがある。ContinuationやKeysetが同じでも実Read Cohortの裏付けが必要（#2622／#2586）。
- `where`が出力しない秘密Fieldを参照する場合、表示不変・同じResult DigestであってもValidation、Consumerが知る件数、実取得、外部送信の許可は異なり得る（#2516／#2624／#2387）。
- 最終的なビルド再利用判断は[#2600](https://github.com/hat47x/tei-platform/issues/2600)、反応的データContextの再評価は[#2582](https://github.com/hat47x/tei-platform/issues/2582)がOwner。**生成したコードの再ビルド省略と、実行時データの再取得省略を別レーン**として扱う。

### 15.4 高階Operatorの契約を試す次の受入セット（OL-01〜08：新実行NOT RUN）

| ID | 一軸変化 | 独立期待・判別 |
| --- | --- | --- |
| OL-01 | 同一演算をMarkdown表／チェーン／軽量DSLで記述 | 正規化候補・Role・型・Scopeが一致してもSource occurrenceは異なる |
| OL-02 | 値結果が同じまま`filter`を`map(parse)`の前後へ移動 | 例外・診断・実Read依存が異なるケースは等価と認定しない |
| OL-03 | `map(f).map(g)`を一度の`map`へ融合、gは非純粋/失敗し得る | 結果値だけの比較では不足。許可したEffect/Failure/traceを明示 |
| OL-04 | `sum`対象の行を追加・削除・修正、表示は2階層集計 | Result grain/元Entity ID/complete/空集合/null/丸め/追加差分の区別 |
| OL-05 | `any(p)=true`のWitness行だけ削除、別Pageは未取得 | 旧trueを維持したまま「現在true」と扱わず、再取得・未解決 |
| OL-06 | Filterの出力列は同じ、Predicateに利用する非表示Fieldだけ変更 | 依存再評価、必要実Read、開示/推測可能件数、ビルド意味とは別 |
| OL-07 | `window`のlate event訂正、10Hz→60Hz、Duration・Time Point型の相違 | 結果のretraction、時間単位・Window/Actor条件、暫定/確定の区別 |
| OL-08 | 意味Sourceだけの整形・Role/Operator版のみ変更・実Grantだけ失効 | Source再解析／意味再解決／Target再生成／Context再評価／実行資格を**工程ごと**に判定 |

### 15.5 実装への渡し方（先行導入を避ける）

1. **#988/#2620**：高階Operatorの入出力と、成立条件を標準Data上の属性/関係として説明する。既存Language Pluginの実Parserと比較し、未知語・曖昧なRole・作用的Lambdaを診断。構文や法則の候補だけでは採用しない。
2. **#1706/#2593/#2497**：集合・Measure・構造写像の保存条件と反例Oracleを既存Corpusへ対応付ける。独立Expectedの適用対象を確認する。
3. **#2506/#2582/#2622/#2387**：同じFunction Data Contextから実Read・Rule Input・Predicate依存を切り分け、既存Query/ProviderへPushdown可能な範囲だけを比較。非業務のStream/Graphにも同じ検査列を当てる。
4. **#2600/#2018**：AIによる局所変更とTarget生成で、「手続き的分岐の節減」と「意味/Failure/Source/Proof/修正時間の総費用」を別計測。実行時差分計算とビルドskipを取り違えない。
5. **#2586/#2598/#1174**：同一ScenarioのRead→Validation→Action候補→実Request→独立Observationまで接合。**Result・診断・Read footprint・Effect・Grant**のいずれも未観測ならその資格を未確定として残す。

**成熟度**：Operator Lawを**新しい固定APIとして採用する計画ではなく**、既存Operator/Plugin/Semantic/Scenarioに必要な情報と、変換を許す場合の根拠を明確化したもの。OL-01〜08は本PRで新しいテストを実行していない。独立合成参照の局所PASSを実Provider/Target/Studioへの同一Run合格に換算しない。


## 16. Data→意味→実現→Consumerの変更影響閉包（2026-11-02、最新別レーンの実コード反例を統合）

### 16.1 今回明確になった不足

Everything is Dataが十分に成熟するには、**同じプレーンなデータの記述が得られること**に加え、そのデータを何が解釈し、何を読み、何を生成・実行・開示したかという**実際の依存関係の閉包**を説明できる必要がある。構文・Meaning・Operation・Provider・Artifact・Consumerのそれぞれに、部分的に安定したID／Digestがある。**単一IDや一つのCompleteフラグの一致を、横断的な再利用資格としない**。

既存の[#2600](https://github.com/hat47x/tei-platform/issues/2600)（実消費入力／再利用／ArtifactSet）、[#2630](https://github.com/hat47x/tei-platform/issues/2630)（Consumer相対の必要Member）、[#2611](https://github.com/hat47x/tei-platform/issues/2611)（多Mapping）、[#2387](https://github.com/hat47x/tei-platform/issues/2387)（非出力依存）、[#2516](https://github.com/hat47x/tei-platform/issues/2516)（用途別開示）を、**新しい依存グラフ・Cache Engine・Manifest Schemaを実装せず**、最初は一つの差分表で接続する。

| 依存面 | 具体的な依存 | 変化しても見かけ上不変になり得るもの |
| --- | --- | --- |
| **Authoring／構文** | Source occurrence、Syntax/Role Plugin pin、literal/ref／scope/capture | Markdownの表示文字、plain Data構造Digest、同名Function |
| **意味／評価** | Operator ID＋版、型/量/NULL、非出力Predicate・Sort、Rule/Actionの事前Read | 出力列の一覧、Function定義ID、画面表示値 |
| **取得・Provider経路** | Function Mapping、選択Source/Storage、Provider route、物理Field/Rowset/known-at、Capability | Operation ID/Basis、Alias表示名、Result JSON |
| **生成・実現** | Generator/Target、実消費Source/Mapping、Artifactの**全必要Member**と実Byte、Consumer想定Schema | Manifest Complete、Set digest、ファイル名、Meaning ID |
| **Consumer／作用** | 実際にロードしたArtifact/Snapshot、表示・Rule出力の開示、現在のHost Grant、実Request/Receipt | ProducerのPASS、Studio preview、受理済みMeaningの過去版 |

**区別する二つの閉包**：(a)静的なSource／Plugin／Targetの変更からGenerator・Artifact・テストに必要な**Build再利用**、(b)実行中のData/Rule/Provider/時点変更から再読取・再評価・更新許可に必要な**Runtime再評価**。Graphviz文脈変換部品（§11）は、これらの依存をGraphデータへ可視化する候補とし、見せてよい情報・Node ID・Edge意味を#2497/#2516で解決する。非公開Dependencyを描画で漏らすことは避ける。

### 16.2 既存実コード・実テストからの新しい圧力（望ましい挙動として未採用）

- [Draft #2632](https://github.com/hat47x/tei-platform/pull/2632) は、実TEI `DeriveOperationSurface` を対象にした**7件の特性記述Goテストを保存**。候補挙動として、選択MappingのColumnを増やしても`Operation.ID/Basis`が同じで`create.Inputs`が変わり得る。別のStorage型変更は同Pure Surfaceだけを見ると不変になり得る。World/Target全域の再生成に問題があると確定したものではなく、当該PRにある7件のGo実行は**NOT RUN**。
- [Draft #2634](https://github.com/hat47x/tei-platform/pull/2634) は、実Go DerivedIRへの**非出力Predicate/Sort/相関参照**の伝播と、Analyzer `Lowering=full` をGo側が単独で完全検証していない可能性を示す**5ケースの特性試験**を保存。実SQL Worker・TEI full解析の不具合認定でも、Goテスト成功の報告でもない。
- [Draft #2633](https://github.com/hat47x/tei-platform/pull/2633) は、Manifestに`Complete=true`があってもConsumerの必要Member充足・実ファイルByte／実Schema解釈を確認する必要があることについて、**6件の実Go Characterization Test**を保存。生成Producerの自己整合と実Consumerの正しいロードを別にする。PR作成時点の6件実行は**NOT RUN**。
- [Draft #2631](https://github.com/hat47x/tei-platform/pull/2631) は、**未知Roleの標準データ表示への退避**と構造型・表示Scopeを扱うV8隔離16 assertions PASSを報告。これは実TEI Studio/Language Plugin上の表示・編集実証ではない。

### 16.3 新たな横断反証表（QDC-01〜10、**このPRではNOT RUN**）

| Case | 一軸変更／反例 | 期待する「必要な再資格」と、既存担当 |
| --- | --- | --- |
| **QDC-01** | 同Operation ID/Basisで選択MappingへFieldを追加、`create.Inputs`が変わる | Operator/Field入力をConsumer別に再照合。Basis一致だけでArtifact再利用を許可しない。#2632/#2600 |
| **QDC-02** | 物理Storage型／NULL規則のみ変更、Pure Surfaceが不変 | World/Mapping/Targetの依存範囲を照合し、必要な生成・実Provider試験を保留／選択。#2611/#2632 |
| **QDC-03** | Aliasの名前だけ変更、複数経路選択の順序が変わり得る | 実経路ID/Target/Purposeを見てSourceの選択を再資格化。名前順・第一候補をAuthorityにしない。#2611/#2627 |
| **QDC-04** | Output Field/Query IDは同じ、Predicate秘密Fieldの閾値だけ変更 | Source/IR非出力依存・実Read/Disclosureを追跡し、表示不変でも再評価。#2634/#2387/#2516 |
| **QDC-05** | AnalyzerのColumnRefsが不足するが、`Lowering=full`がある | 状態文字列のみから解析依存網羅を証明しない。実Worker→Go Source lineagesを必要に応じ再資格化。#2634/#2387 |
| **QDC-06** | `Manifest.Complete=true`で必須成果物の一つが欠落 | Consumer別ExpectedMember/必要Scopeを固定し、実消費一式を検査。#2630/#2633/#2604 |
| **QDC-07** | 壊れたJSONに正しい`schemaVersion`らしい文字列が含まれる | 文字列含有ではなくSchemaとして解析して資格化。#2633/#2482 |
| **QDC-08** | Manifest固定後にArtifact bytes／Gap Reasonが変更 | 実Readerが読んだ各Byte・Gap意味／Artifact Setを検証。Producer自己照合のみで成功としない。#2633/#2604 |
| **QDC-09** | 未知Domain/RoleがPlugin導入により出現 | Structureは汎用Studioで読めても、未知のAction/更新可否/解釈済みRoleは`unknown`へ退避。#2631/#2620/#1124 |
| **QDC-10** | 同じ構造をGraphvizで表示、非表示Rule／Secret Predicateの依存が追加 | 変換部品の保存条件とConsumer別Disclosureを再照合。表示構造一致は依存/開示の同値証拠にならない。#2497/#2516/#2621 |

### 16.4 H0〜H5に加える最小作業（既存受入の差分）

- **H0/C0：実依存の棚卸し**。一つのFunction+Read+Rule+Provider route+Artifact Consumerを選び、どのSource FieldがどのOperator/Rule/Mapping/Memberに実際に使われたかを静的に一覧化する。未解析のhidden依存は`unknown`として残す。
- **H1/C1：実TEIから正規化・Roleを取得**。既存の実CLIで観測できる範囲を先に計測。#2625の同義表層は新たなParserを前提にせず、実World上で何が解決できないかをSource位置付きで示す。
- **H2/C2：選択Mapping変更と隠れたQuery/Rule依存を同一Fixtureへ**。#2632/#2634のCharacterizationがどこまで実TEI Go testsで成立したかを**試験実行後に限り**結果として扱う。#2624 Rule footprint／#2627 Provider Cohortと接合する際、同一World・同一Snapshotの証拠を要求する。
- **H3/C3：汎用StudioとGraphviz説明**。未知RoleはPlain Dataへ退避、Source/意味/実現の欠落を診断。編集逆写像は一意Occurrenceのみ。Graphvizに未開示Field/Ruleの内容を無断付加しない。
- **H4/C4：Artifactと実行の検収**。#2633のConsumer別Manifest/Reader負例を#2604実Publish/readと接続したあと、現Targetで読み込まれたByteとOperation/Receipt/独立Readを別の証拠として固定。Source digest同一と現在のHost Grantは独立。
- **H5/C5：増分とAI生産性**。QDC-01〜10を clean全工程・選択実行の両条件で比較し、**False reuse／False invalidation／実行時間／AIの修正差分**を#2600/#2018に還流。Cache命中率ではなく、現在の意味適合とConsumerに必要なArtifactの充足を採否条件とする。

**今回の資格境界**：この§16は既存Draftの観測・特性試験予定・限定実績から導いた横断検証案であり、製品の新Core契約、実Go試験PASS、新Studio、Manifest脆弱性確定、正式Meaning受理を宣言しない。誤再利用が疑われることと、本番事故が確認されたことを分ける。


## 17. 自己記述Roleと現行Go Language Pluginを接続する実証（2026-11-02、他チャット成熟差分）

### 17.1 設計上の到達点と現在の製品境界

**状態の更新（2026-11-03）**：下記の記録後、#2638はmainへ統合され、#2639はDraftではないOpen PRになった。#2638で統合されたのは設計候補と合成Referenceであり、正式なRole Resolverや実ProviderのFingerprint検証が成立したことを意味しない。以下のCI観測は各時点の履歴である。

「Every Definition is Data」が製品機能になる最小の経路は**新しいDSL Parserを先に作ること**ではなく、既存Language Plugin／Operator／Worldの実入口へ、自己記述的Role候補と複数表層のSourceをつなぐこと。関連PR：

- [#2638：Role bootstrap](https://github.com/hat47x/tei-platform/pull/2638)。plain DataでRoleの必要Shape・依存・版／**Contract Fingerprint＋実装Fingerprint**のPinを記述する**限定合成24ケース**（V8実行・GitHub保存Source照合PASSと報告）。先行19件にRB-20〜24（同版別実装、推移Eager依存の実装差、正確な再Pin、無関係Provider追加、版文字列だけのPin不十分）を追加。Eager解釈の循環と、Graph/Tree等に普通にある遅延Ref循環を区別する。Root RoleだけPinしても必須依存RoleのContract/Implementationが固定されなければqualifiedとしない。同名別Providerや引用された`role`文字列は自動採用しない。合成Fingerprintは実TEI Extension Lockや実実装byteの真正性を証明しない。
- [#2639：既存Go Language Plugin Referenceへの接続](https://github.com/hat47x/tei-platform/pull/2639)。`ParseSelected`／`ParsePipelineSyntax`／`LowerPinnedPipeline`／`ResolveOperator`を用いたtest-only JSON language対Pipe syntaxの限定比較。実`syntax.NewParser(...).ParseSemanticFile()`→`BindPinnedSemanticInput`のEG-08、および**同じData ID/限定Operator出力型でもAttribute TypeRefを変えると意味モデルとして同一とは断言できない**EG-09を含み、現HeadでGoテスト9関数を確認。`same data ID`／`same limited binder output`／`same accepted model meaning`を必ず別の比較列とする。**test-only JSON語彙・fixture plain-dataは正式Core schemaやAccepted Meaningではない**。
- [#2644：意味定義と要素の対応](https://github.com/hat47x/tei-platform/pull/2644)／[閉鎖済み#2095](https://github.com/hat47x/tei-platform/issues/2095)。同じ要素に複数の受理済み意味定義が結び付く場合の曖昧性や未記入を、**受理済み定義だけ**に対して列挙する別Reference。要素の**業務的/意味的な一意帰属**は、共通Data CarrierへFunction/Test等の**複数解釈Roleを適用可能**という#2620の設計と矛盾しない。前者はどの意味定義が正本か、後者は同じデータをどの目的で使うかという別軸。
- [#2587：Data-first実装](https://github.com/hat47x/tei-platform/pull/2587)は**mainへ統合済み**。標準Operation/Artifactの既存製品経路は、未統合候補という扱いから、exact mainを確認して検証する接続先へ進める。一方、実Domain Meaning・複数Provider・Consumer必要Member・Host認可の資格は統合そのものから導かない。

**CI観測（版の区別）**：#2638の**旧Head** `47b52240b7dff18706e83a7e89af63e6fb4a3c49` ではRepository Health／依存脆弱性確認のActionsがcompleted/success。ただしRB-20〜24追加後のHead `71d2d1bcaacd0a0f2f3eb16b27b1a34bc3036bd8` は確認時点でWorkflow queuedであり、24件のGitHub Actions PASSではない。#2639の**旧Head** `5be242b664e1cb77d4af2f97930e52a7681fe480` にも同2 Workflowのsuccessがあり、Repository Healthログで`go test -count=1 -timeout 60m ./...` PASSを確認。ただし#2639はその後もHEADが更新されているため、**実行時に最新Headと同一SHAのWorkflow結果を再取得**する。2026-11-02に観測した後続Head `f94fb90f9379aba0e309461be511052c4b183921`、さらに `5fdef5de5956ba0b4996d2861fc1ba4b1d94a222` は、各観測時点でWorkflowがqueuedだった。**旧HeadのPASSを新版へ転用しない**。CI PASSはProduct CLI・Studio・外部Hostでの意味受理を証明しない。

### 17.2 小さな接合試験パケット：PB-01〜10（横断の新実行はNOT RUN）

| Case | 入力の変化／接続 | 完了条件と担当 |
| --- | --- | --- |
| **PB-01** | 同じFunctionをPipeとJSON test-languageで記述 | 実Language Plugin上のSource occurrence／Span／Pin付きOperator解決を分離して一致条件を説明。#988/#2639 |
| **PB-02** | 同じSourceだがOperator/Role Plugin v1→v2 | lock/version固定と現在のMeaning資格を再評価。文字列同一を旧受理の継承理由にしない。#988/#2638 |
| **PB-03** | Root Role Pinのみ残し必須Eager Role依存を欠落 | 部分的な適用候補と解釈完了を区別し、解決不能を明示。#2620/#2638 |
| **PB-04** | Eager A→B→A と、Deferred graph Ref A↔Bを比較 | 前者を循環未解決、後者を用途条件付きで解釈可能として区別。#2620/#2544 |
| **PB-05** | 文書引用内の`role: function`、AI Skillの実行要求テキスト | Source上のData／Role候補／正式Meaning／Tool呼出／現在Grantは別。#2621/#2605/#2515 |
| **PB-06** | 同一CarrierにFunction／Test／Documentationの3 Role | 多用途の構造共有と、受理済み要素意味定義の競合を別判定。#2620/#2095 |
| **PB-07** | 引数名／同名別ID／局所束縛／Operator effectを変える | Plain Dataの同形や値一致だけではMeaning同値を主張しない。#988/#2639 |
| **PB-08** | 対象の異なる構文表層ReferenceとData Mapping Referenceを同時に使用 | 両者を同じSubject／Source revision／Context IDへ実接続した証拠がなければ横断PASSと呼ばない。#2639/#2624/#2586 |
| **PB-09** | 新Role Pluginが増え、旧RoleのEager依存が変化 | 必要なSource/Meaning/Artifact/Studioの段階だけ再資格化し、影響不明は保留。#2638/#2600/#2631 |
| **PB-10** | 承認済みMeaningを生成Target/外部Consumerへ出力 | 実ロードArtifact・Source Index・対象Role/Provider・現在開示/実Grantを確認する。#2587/#2604/#2516/#1174 |

### 17.3 現実的な接続順

1. **B0（read-only）**：#2638の必要依存Closureと#2639の実Go Language Pluginの入力・出力を比較する。Fixtureが異なるので、まず一つのSource/Operator/Role Pin対応表を作る。合法なDeferred Ref、Source span、引用、実CLIに露出する範囲を明示。
2. **B1（現実のParser／World）**：#988の既存`ParseSelected`とSemantic Parserが何を受け取り、どこでplain Data Candidateから受理済みMeaningへの境界があるかを実試験で確かめる。ない機能は「未実装／候補」として記録する。
3. **B2（Context/Operation）**：同じSubjectを#2506の明示Read Mapping→#2622 Data Operations→mainに入った#2587標準Operationへ渡す。Roleで解釈できることと実Providerを一意に選べることを分ける。
4. **B3（Consumerと資格）**：同じ固定Source/Meaning候補をCUI/標準Studio/SHO/実Targetで比較し、出典、Role不明の退避、Manifest必要Member、開示、Host Authority、独立Oracleの不足を列に残す。
5. **B4（選択的再資格化）**：固定Pluginの一版だけ、またはRole依存だけを変え、実消費依存を確認した範囲でBuild skip／Runtime Rule再評価／Grant再照合を判定する。過去のCI成功を現在資格に自動継承しない。

以上は**#2619の既存H0〜H5／C0〜C5へ受渡す作業であり、新Gate/実装API/Compiler/Registryを追加しない**。新実行のある段階のみ、exact Source/Command/CI run/実Consumerの証拠を添える。

## 18. 正常0件・否定依存・ABAをEverything is Dataの再評価へ接続

### 18.1 問題：0件も条件と取得証拠に依存するデータである

[顧客照会の移行設計 Draft #2640](https://github.com/hat47x/tei-platform/pull/2640) と [#2582のContext再評価](https://github.com/hat47x/tei-platform/issues/2582)の別レーンで、**結果0件なのに依存なしと判定する誤り**が具体化した。

「結果行が0件」という値は、`Predicate／Relation path／Provider route／Scope／Read Cohort／Rowset completeness／取得時点`の資格に依存する。要素IDが無いから依存Edgeが0本とは限らない。`NOT EXISTS`、`COUNT=0`、`all`、`TopN`、未取得ページは特に注意する。**Rowsetがcompleteでも個別Fieldがabsentなら、NULL/0と同一には扱えない**（#2627）。Native SQL／Provider pushdown後の`WHERE/ORDER/LIMIT`とLocal filterの順序差も偽の正常0件を生む可能性がある（#2640の限定SQLite実験）。

**ABAの反例**：時点t0で0件、t1で対象Aを追加、t2でAを削除して再び0件。現在の値だけを比較すればt0とt2は同じでも、「変化なし」「古いReadを再利用してよい」「過去の業務確認が有効」のいずれも自動的には成立しない。Source/Context/Read/Rule/Consumerごとに必要な現在性を検証する。

### 18.2 否定依存・現在性の受入セット：ND-01〜07（新実行NOT RUN）

| Case | 状況 | 期待される判別 |
| --- | --- | --- |
| **ND-01** | 元件数0、未Mapping Relationに一致行を追加 | Mapping対象外なら表示は影響なしでも、Ruleが対象Relationに依存するなら再評価候補。#2506/#2582 |
| **ND-02** | 完全Readの0件→追加→削除→0件（ABA） | t0/t2の値同一とRead/Rule/Owner確認の現在資格を区別。#2582/#1125 |
| **ND-03** | page1のみ0件、page2は未取得 | 全件不存在・all/noneの真偽へ確定しない。#1706/#2627 |
| **ND-04** | `Filter(active)→Limit(1)` と `Limit(1)→Local Filter(active)` | 0件/1件が変わる負例で非可換性・Provider pushdown法則を検査。#2387/#2622/#2640 |
| **ND-05** | 同じRowsetでField absent→observed null | 欠落／未観測／明示NULL／0を区別して意味と開示を再評価。#2627/#2516 |
| **ND-06** | Window A→B後にAの0件応答が遅着、Window2はAのまま | 遅着応答をWindow1に採用しない。無関係Consumer/歴史的Receiptは無条件失効させない。#1124/#2582 |
| **ND-07** | SourceのQuery Shape/Roleは不変でProvider Route/Snapshotが変わる | 同値の結果でもCohort・実物理Read・現在Grant・Consumer開示を再資格化。#2627/#2516/#1174 |

**設計の帰結**：定義と結果はEverything is Dataで表せるが、**「存在しない」という判断も、何を探し、どの範囲・時点・Providerで探したかという依存情報が必要**。必要情報は既存Query/Data Context/Rule/Provider/Evidenceに保持し、新しいNegative Dependency Engine、全局Cache、Reactive Hostを追加しない。#2600の増分ビルドと#2582の実行時再評価は、それぞれのOwnerで別に判断する。

### 18.3 外部AI Harness／文書Consumerへの最小受渡し

[TAI #271](https://github.com/hat47x/tai-pleroma/issues/271)／[Draft #270](https://github.com/hat47x/tai-pleroma/pull/270)では、Product間のEvidence変更影響と**現在Consumer／権限の資格**を分ける計画がある。TEIは`Source/Role/Operator Pin／明示Context/Scope/Read witness／Result & Unknown／Claim資格／Provider Route／Consumerへの開示区分`という**実際に確かめた範囲だけ**をProduct Skill・SHO等へ受け渡す。外部Harness/AgentがDSLの`role`や`tool`文字列を見つけても実操作Commandとして採用しない。

TAIのQH-01〜18は公開合成Nodeの有限試験であり、TEI実CLI／外部ネットワーク送出／SHOの保存・配送／Host Grantの実証とは別。TEIのSource/Meaning/Provider・独立OracleはTEIの既存Ownerに残し、TAIに第二のMeaning/Authorityを設けない。現時点で新しいProduct跨ぎの実行試験は**NOT RUN**。


## 19. 汎用System Compositionの実選択・配置・利用資格を同一Subjectで照合する（2026-11-06）

> **状態：設計引渡し／採用前の検証条件。** 11/5〜11/6の他レーン（#1016、#2619/#2620/#2622、#1125/#1126、#1130、#1174、#2831、Draft #2885/#2886/#2887、#2889）を接合する。新Core Schema、第二System/Meaning正本、Resolver、TEI専用Host、中央Runner、独立Gateは作らない。#1119のG1/G2/G4と既存H0〜H5、#2586のHJ、#2500のRT、#1180 Scenarioを維持する。

### 19.1 成熟した原則と今回埋める欠落

Everything is Dataの共通Carrierは、Data・Function・Business・Rule・Test等を同じ表現形式で扱うためのものであり、意味Role、Owner、現在Authority、物理配置、実Effectまで一つにするものではない。Script-firstのMarkdown／表／簡潔なDSLは専門家の認知順・Sourceの可読性を優先し、Go固有構文は普段の記述へ露出させない。標準実行形態はGo単一バイナリ、外部Host向け.NET/Java/WASM等は対象別Targetとして別途成立性を測る。

追加の不足は、**一つのSystemとFunctionが、どのPlugin/Providerを実選択し、どのBuild/入口から、何を実際に読んで作用したか**の関係が、別レーンの証拠からまだ一回の試行で説明できないこと。Application Composition Profile（#1016）を構成選択・適合性照合の既存入口として利用し、別のSystem Manifest/Composition Resolverを導入しない。

| 区別すべき単位 | 既存責務と必要Witness | 誤った昇格をしないこと |
| --- | --- | --- |
| System / Function / Business | #1125/#1126/#2619のData・Function・Business Role、Source occurrence、Semantic ID、Owner Meaning版 | Function所属≠公開入口。Business DFDのmanual/handoff≠実行Workflow。業務Key≠物理PK |
| Role / Provider / Application Selection | #2620のRequired Shapeと実Pin、#951 Capability/Provider contract、#1016 Profileの選択結果と必要Contract | Bundle同梱≠実選択。Descriptorのpure/required自己申告≠検査義務やAuthority。候補追加≠黙ったProvider置換 |
| Data Context / Physical Read | #2506/#2586の明示Relation/Field/Scope/GrainとResult Slice、#2622のProvider Realization、実Readerのrowset/時点/完全性 | Query IDや同じRows≠同じSQL・Provider・Read cohort。0件≠未取得・不存在。partial≠complete |
| Function / Build / Entrypoint | #1130のFunction→1または必要時N Build・CLI/Web/GUI/Job入口、#1167の実リンク閉包、#1097のArtifactとlogical subject分離 | 起動時のApplication選択≠未使用Runnerの静的除外。Function定義の同一性≠物理Artifactの同一性 |
| Actual Build / Runtime | #943/#2600/#2831の実compiler/link input・toolchain・Target・Artifact digest、実loaded/called入口 | 宣言Graph/Manifest/digest一致≠実Build input完全・全Self-Build・実Runtime利用 |
| Invocation / Effect / Consumer | #1130/#1174のRequired/Supplied/Evaluated Guard、現在Grant、OperationID/Attempt/Receipt、#2604/#2516の実Reader member/body・開示 | 起動成功・stdout・ACK・Observer非nil・Manifest Complete≠Commit/Business完了・Consumer受領 |

上表は**単一の新しいUniversal Envelopeを実装する要求ではない**。既存Object/Receipt間の結合可能なKeyと、Claimに必要な観測が現実に存在するかをread-onlyの対応表で示す。Source Revision、World/Model Revision、Meaning Acceptance、Build/Artifact Revision、Invocation/Attempt、Consumer Read Revisionは別identity。必要情報がない場合は当該Claimの資格をunknown/unqualifiedとし、全体を一律不可とする必要も、空集合の自動PASSもない。

### 19.2 Claim相対の必要条件：読み取り、書き込み、生成物を混ぜない

| 主張したい範囲 | 最小のRequired Obligation（既存Owner） | 必須ではないもの／停止理由 |
| --- | --- | --- |
| **C-Read**：特定Functionの値・存在の説明 | 実Loaded Source/Role、Mapping/Scope/Grain、Selected Query/Provider、実Readerの値と取得範囲、独立Expected、用途別開示（#2586/#2514/#2516） | read-onlyにDB Commitや全Self-Buildを一律強制しない。否定Claimなら母集団・条件・完全性が不明な時点でunknown |
| **C-Build**：所定Functionが所定入口に存在 | Accepted/selected Function・Target、Build収録、実コンパイル入力、Artifact digest、実起動/公開入口、試験で実使用したbinary（#1130/#1167/#2600/#2831） | Source定義やruntime flagだけではlink除外／入口実存を証明しない。Plugin CatalogとBundleだけでは実選択不明 |
| **C-Effect**：特定操作が所定の業務Effectをもたらした | C-Readの必要部分＋Accepted Action、対象ID/version、Current Grant、実Guard必要集合/評価、Provider Commit fence、OperationID/Attempt、Receipt、独立Read（#2500/#2598/#1174/#1163） | ACK消失・partial・旧Grant・Unknown Receiptでは完了主張を停止。同key lookupと新key再適用は別 |
| **C-Publish**：同一世代の成果物をConsumerが受領 | Source/Generator/実Build/Manifest、必要member集合、公開世代、Readerが消費した全member byte、開示範囲（#2604/#2516） | producer成功やManifest completeのみでは未受領。静的サイト/SHOにDB Txを一律強制しない |
| **C-Portability**：別Domain・別Targetへの再利用 | 共通構造/Role/Operatorの再利用箇所、Domain固有差、意味と実現の独立Expected・Actual、増えたCore分岐/Adapter/修正費（#2620/#2622/#2018） | 同じJSON形状や画面RowsだけでDomain意味同値／2 Target適合を主張しない |

各Claimに対し、既存#1180/#2514でRequired・Observed・Unknown/Not-run・Independent Expected・Known Mutant・Claim ceilingを記録する。候補AI/Role/Plugin自身がRequired集合を空にしただけで検査義務を免除しない。非適用は理由付きnot-applicableとし、別Revision/Runで成立した局所PASSを同一AttemptのEvidenceへ合算しない。別レーンでの局所実証はそのまま価値を残す。

### 19.3 一番短い接合手順（新Runnerと重複Fixtureを作らない）

1. **P0：mainとDraftのSource/実装棚卸し。** #1016/Profile、#2620 Role、#2622 Query、#1130 Build/入口、#2586 HJの現実のproducer/consumerフィールドを一つのFunctionへ対照。対象は既存order_entryの一Function。merged #2882のWorkspace→RunPlan期間/Effect前提は局所Go synthetic実証、merged #2853/#2854/#2870のPG/CAS・Unknown・ABAは別経路の実PG局所実証と表示し、E2Eに加算しない。Draft #2885/#2886/#2887と#2889は対象HEAD・試験・未実証をそれぞれ独立確認する。
2. **P1：read-only一往復を#2586 HJのOwnerで実行。** 実Source/Role/Function→Context Mapping→Query/Provider Selection→実CLI argv/binary→実Reader body/scope→独立Expectedを同Revision/Claimで追う。Roleが実Interpreterへ接続されていない等、最初の断線位置を欠落として記録。#1016は実選択、#1130はBuild/入口への対応情報を渡すだけとする。
3. **P2：Buildの真偽を別Claimで検証。** 同じFunctionのCLI/Web入口と単一Go Binaryでの実収録・不要Application import除外を#1130/#1167で実Compiler/link inputにより観測。必要時だけ1→N Buildへ拡張。#2831の全Self-Build Stage0/1/2は並行P0で検証し、G1/G2の新Hard dependencyにしない。
4. **P3：書込み一往復は既存#2500 RT→#2598→#1174→#1163へ委譲。** 同一Operation ID、現在Grant、必要/供給/評価済みGuard、実PG Commit/CAS、Receipt/別Connection Read、独立Consumerを固定。ACK lossは同OperationIDでlookupし、新key盲再試行を禁止する。Old/New Client共存と移行Cutoverは#2555/#2887の別資格。
5. **P4：異質Domainを比較。** まず静的サイト/文書をread-only＋Artifact Publicationの対照として用い、次にGraph/Stream/VR的時系列の意味限界をPressureする。Business TxをすべてのDomainへ強制せず、Source/Role/Lineage/Time/Consumerがどこまで再利用できるか測る。実証前に専用Domain Coreを追加しない。

**反証には既存Scenario/Fixtureを流用**：未使用Provider追加→選択不変の確認とビルドbyte不変を区別（#1016/#1251/#2681）；選択済みRole同version実装改変→Pin失効（#2620）；起動時Application切替でもlinkに未選択Runner残留（#1167）；0件/partial/NOT EXISTSとABA・時点変更（§18/#2582）；Required Guardをcallerが渡さずStep成功、Observer存在だけでEffect完了、先行Step効果後のblock（#1130/#2889）；Artifact世代R1/R2 member混在（#2604）。新しい重複番号付き検収表は作らない。

### 19.4 他レーンからの制約をSystem Compositionへ適用する

- **Data / Function / Business**：#1125/#1126とDraft #2886に沿い、Business手作業・疎結合・DFDと実行可能Workflowを区別。業務Key/Occurrence、物理PK、valid/recorded time、Definition/Instanceは見かけ上同じID/値で潰さない。Data中心の表現とBusinessの意味的完了条件は別。
- **DB移行**：#2555とDraft #2887のExpand/Backfill/Validate/Cutover/Contractは旧新ClientのRequired Contract、Schema/Provider技術適合、履歴・Key、read/write正本、Effect/Unknown、Owner判断を分離。external-observed履歴はTEIによる適用を意味しない。MIG成功ログ一つで全段階PASSとしない。
- **Plugin方式**：#1016/#951とDraft #2885では、拡張の必要性（標準Data/静的Libraryで代替できないか）、論理Contract、物理配布（静的合成/動的/外部Process）を直交化。選択済み依存の充足と未使用Bundle/Moduleの非干渉を実Buildと独立Expectedで確認。Data上のPlugin候補はHost Grantを付与しない。
- **AI・Harness・SHO・EKI**：AIへの送出Context/用途別Disclosure、候補/独立Expected/人の受理、Action実呼出を別に扱う。TEIが利用者へ渡すのは実観測したSource/Role/Scope/Result/Uncertaintyのみ。SHO引用の歴史的版と現在Consumer資格、EKI Task/Lease結果とTEI業務Commitを同一結果へ畳まない。現在のAuthorityはそれぞれ既存Host/Providerに残す。
- **Studioと変更影響**：標準Data/Meaningの表・図・式・文書投影は#1124/#2497を再利用。Sourceへ安全に逆写像できないAggregate/Graphはread-only diff。#2582 Runtime再評価と#2600 Build再利用を分離し、ABA・negative dependency・selection epochを既存責務で判断する。

### 19.5 最小受入票と採用の停止線

同一Attemptのread-only HJで報告する列は、(a) Source occurrence/revision・実Loaded、(b) Meaning/Role/Required Shape/Pinと受理資格、(c) Function/Context/Scope/Grain、(d) Profile/Provider/Query選択と実Reader、(e) 実Build/Entrypoint/Artifact/argv、(f) Consumer受領bodyとDisclosure、(g) 独立Expected・Actual・未充足Obligation・Claim ceiling。write時だけCurrent Grant/Guard/OperationID/Effect/Receipt/独立Readを追加する。未観測の内部ID/選択理由は想像で補わずunknownとする。秘密のSQL/DSN/件数/Sourceは用途別開示に従う。

**最初の合格条件**は、同じFunction/Source Revision/実Binary/ReaderでC-ReadのRequiredが充足し、NULL/partial/0件/旧入力の少なくとも既存negativeを正しく保留または拒否できること。C-Build、C-Effect、C-Publish、C-Portabilityは独立判定し、C-Readの合格から自動で許可しない。Core特殊分岐数、Adapter/手修正量、未使用実装の混入、誤ったPASS/過剰拒否、独立Mutant検出力、Tool/CI/Token/人手を#2018の比較へ渡す。

**今回の変更が示すのは設計引渡しのみ**。新実Go/PG/異質Domain/GUIの横断試験、Owner Meaning Acceptance、G1/G2/G4の正式判定はNOT RUN/UNQUALIFIEDのまま。#1016 Profile／#2586 HJ／#1119 GateのOwnerと境界を変更せず、production Schema・APIを追加する判断は実装Gapと独立試験の後に委ねる。

### 19.6 接続種別ごとの成立条件と構成上の閉包（2026-11-06 横断再照合）

**一つのSystem Definitionが存在することから、Runtime接続が存在するとは推論しない。** 接続を一種類の万能Operationへ変換せず、既存のDefinition/Mapping/Selection/Build/Invocation/Receipt/Readerを利用して次の差分をClaim相対に観測する。

- **Required**：受理済みMeaning/Ownerの業務要件、Contract、Scenario、実Consumerの要求から独立に必要と判断したSource・Role・Guard・Result集合。候補Role/AI/RunPlanによる自己申告の空集合は免除根拠にならない。
- **Declared / Discovered**：Manifest/Source上に記述され、既存Loader・Resolverが候補として発見した範囲。Source root不存在や未宣言のModuleは、単なる `validate ok` で読み込まれたことにしない（#2823）。
- **Selected / Resolved**：Application Profile、Operator/Role Pin、Data Source Alias、Query Realizationが実際に選択された範囲。同梱・Catalog掲載・Source同名だけでは選択済みでない（#1016/#951/#2620）。
- **Built / Loaded / Invoked**：実compiler/linkの入力と物理Artifact、そのRuntimeで実ロード・呼出した実対象。特に起動時選択と静的なGo import除外を区別する（#1167/#2831）。
- **Observed / Consumed / Disclosed**：実Provider Read/Effect/Receipt、Artifact memberまたはTask resultのConsumer受領、Consumer/用途ごとの現在開示。生成成功や `completed` 表示から実消費・業務完了を導かない（#1174/#2604/#2516）。

上記は**単一の必須Pipelineではなく、用途別の必要条件の照合面**。read-onlyにWrite Commitを、手作業にProcess Runnerを、静的成果物にDB Txを一律要求しない。未知のRequiredがある場合は`unknown`、必要能力が実Providerで非対応なら`unsupported`、実行直前の認可拒否は`blocked/denied`相当として元の各担当契約を保持する。隠された証拠は`withheld`として適切に投影し、存在否定や値0に置換しない。新しい統一Status enum/APIは採用しない。

| 接続の種別（分類候補） | 最低限の独立した成立条件 | 成立しない場合の典型的な誤認 | 専有Owner |
| --- | --- | --- | --- |
| **Data Read** | Accepted FunctionのContext Mapping／実Source+Role、実Query/Provider/Read cohort、Row/Field completeness、粒度・時点・現在開示、独立Expected | Read 0件を全件不存在、同Rowを同Provider、部分取得を全件とする | #2586/#2506/#2622/#2516 |
| **Command / Write** | 意味上のAction・対象集合/版、Required/Supplied/Evaluated/Effect時Enforced Guard、Current Grant、Operation/Attempt/Receiptと対象別独立Read | Guard未供給でもStep完成、Commit不明をNo Effect、Replayを新規Writeとする | #2500/#2598/#1130/#1174 |
| **Event / Job** | Trigger/契機とRunPlan/Job入場、Step/Attempt/Taskの実結果、各外部Effect/再開資格、必要な業務完了条件 | EKI Task accepted・RunLog completedをDB Commit/Business Acceptedへ昇格 | #1125/#1130/#1174、任意EKI |
| **Publication** | 実生成/公開Generation、Manifest必要member、実Readerが消費した全byte、Disclosureと引用版 | stdout/ファイル存在/Manifest Completeだけで受領済みとする | #2604/#2516、SHO Consumer |
| **Manual Handoff** | Business Actor/Role、発生契機、渡した情報/対象、受領・判断の事実、必要なOwner完了 | DFDのManual stepからAction/Workflow/Current Grantを自動生成 | #1125、業務Owner |

**横断判断**：ある接続の実現を見つけても、そこで保持される`System/Function ID`、物理`Build/Entry`、`Operation/Attempt`、`Business Event`、`Consumer Receipt`は別のidentity。対象・Scope・有効時点・観測時点・Provider/Authorityを欠いた自動的な等号を作らない。

### 19.7 配置・Provider・業務意味を変更したときの選択的再資格化

「影響なし」は対象Claimに対する実消費と否定依存から説明できる場合だけ維持する。全Domain・全Buildを一律stale化するのではなく、#2600（Build）、#2582（Runtime Context）、#2516（Consumer）、#1174（Current Effect）が**各自の依存**を判定し、#1163の同一Attempt Packetで不足を明示する。

| 一軸の変更 | 必須の再確認／維持できる証拠 | 典型的な停止条件・引渡し |
| --- | --- | --- |
| Catalogへ未使用Providerを追加／未宣言Source rootを追加 | #1016でactual selection、#2681でManifest境界を確認。選択・Build/Runtime非干渉を各々独立に判定 | #2681の実CLIでの未宣言root除外は、選択済みBundleの未使用CapabilityやBinary link除外の証拠ではない |
| Selected Role/Operatorの同version内容変更、eager依存の変更 | 実Contract/Implementation digest Pin、必要Source/Meaning・Build依存の再解決 | 文字列version一致・Roleの`pure`宣言では旧受理や未検査義務を保存しない（#2620/#2600） |
| FunctionのEntrypoint／Build配置、Builder/toolchainを変更 | Source/Meaningが不変なら歴史的受理は保持可能。ただし実compiler input、Artifact、入口/Consumer受領を再資格化 | Runtime flagで未使用実装を除外したと主張しない。Stage収束≠全Self-Build（#1130/#1167/#2831） |
| Provider/SQL Route・Schema/旧新Reader/Writerを変更 | 実ReaderのScope・水位・SQL/NULL/粒度、Writer Current Grant/Commit fence、旧新Client Required Contract | Cutover後の同じ結果0件・Migration successログは現行Read/Writer適合を証明しない（#2555/Draft #2887） |
| 組織Membership/OccupancyのRelation/時点を変更 | Relation occurrence・validAt/knownAt/decisionAt、Function Context Read/Ruleを再確認。Host Grantは別途現在照合 | 所属/役職＝認可ではない。#2681のWorkspace E1局所PASS≠実組織Provider/Case移管効果（#1126/#2389） |
| Source/ContextがA→B→A、Preview旧epochの遅延到着 | Source/Context epoch、実Query cohort、Current Grant/Operation対象を再確認。歴史的Receiptは保持 | byte/値/Rowsetが元に戻っても旧Action権限を自動復活させない（#2582/Draft #2884） |
| RunPlan Guardの未供給・先行Step後の停止 | Required集合をCandidate以外から導出し、供給・実評価・Effect時強制・各Stepの既効果を別観測 | #2889 OC-RP-01〜03の実Go負例：`completed`/Observer存在ではGrant確認・実Effect・全Step無作用を証明しない |
| Publish途中失敗・旧member残留・SHO引用版変更 | 公開Generationのmember集合と実Consumer read bytes、歴史的引用と現在用途Disclosureを別に扱う | 新Manifest+旧file混在／受領確認欠落なら公開資格保留（#2604） |

### 19.8 既存Scenarioを増殖させず「接続Edge」を実試行に落とす

1. **最初のread-only試行**：#2586のorder_entry HJ一Functionを主試行にする。#1016から選択済みProfile/Provider、#2620から実Role/Source、#1130からFunction→Build/Entrypoint、#2586からContext→Query→実Reader→独立Expectedを同じ版・実Invocationに照合。必要Field/Ruleが実ロードされない場合は欠落位置を`unknown`として残し、単なる`validate ok`から完全解析を主張しない。
2. **Business/組織の異質Pressure**：#2681のMembership/Occupancy E1は同一仮定の対象別**別試行**として、時点付きRelation/複合Key/Roleと実行Authorityの分離を調べる。#1125のmanual/handoffはWorkflow実行ではなくBusiness Definition/Owner完了の別Claimとする。order_entryの実Reader PASSを組織E2/E3/E4へ加算しない。
3. **RunPlanの否定対照**：merged #2882の期間・Effect前提の合成Workspace実証に、Draft #2889のOC-RP-01（未供給Grant）、02（Observer非呼出）、03（部分Step効果）を既存#1130/#1180のCaseへ参照接続。これは**現状の不十分な動作を特性化**したものであり、安全に拒否できる実装済みとの主張ではない。実PG/Host/Owner資格は#1174/#1163に委ねる。
4. **配備・復旧・非画面Consumer**：#1167でGo単一Binaryの実リンク閉包を検証し、#1130 Jobと#2604 PublicationでEntry/Generationの異なるConsumerを比較。書込みは既存#1174の同OperationID・同PayloadのReceipt Lookup、別Key ABA、実独立Readで個別資格を確認する。旧Client共存や移行Cutoverは#2555で別Attempt。
5. **新Domainの合格条件**：静的サイト/書籍でSource→Role→Data/Graph→Artifact→実Readerを構成し、業務Tx/Store/Authorityの無理な共通化を要求しない。Core固有分岐数、追加Adapter、重複定義、誤ったQualified判定、実変更費用を#2018へ測定引渡し。VR/Robot/Streamは時点・partial・遅着のPressureを先に行い、リアルタイム性能の達成は別途実測が必要。

**次の引渡し物は、新Fixture集合や新Gateではなく既存#2586/#1180/#2514の同一ケースに付けるread-only対応表**とする：各Edgeごとに`Required → Declared/Discovered → Selected/Resolved → Built/Loaded/Invoked → Observed/Consumed/Disclosed`のうち必要な段階、Source/Artifact/Attempt/Consumer identity、独立Expected、実際の欠落、現在利用資格、Owner判断を示す。隣接する既存Receipt間を証拠なしに同一Attemptと宣言しない。AIが生成したRequired/Expected候補を自己採点のOracleにしない。

**証拠の現状（この§19.6〜19.8追補時点）**：#2681のexact-HEAD限定Go/CLIはManifest外Source root非干渉の局所証拠、#2882はWorkspace合成Step、Draft #2889はRunPlanの不十分なGuard/Observer/partialの局所特性試験、#2853/#2854/#2870は実PGの局所Commit/ABA/Replay証拠。いずれも本計画が新たに一改訂・実Provider/Browser/Consumerを結んだ試験ではない。新コード、実CLI/DB/SHO/EKI統合試験、正式Owner受理、G1/G2/G4合格は**NOT RUN／UNQUALIFIED**。既存Gate/Schema/Host/Authority/Readerの責務や採用済み判断を変更しない。
