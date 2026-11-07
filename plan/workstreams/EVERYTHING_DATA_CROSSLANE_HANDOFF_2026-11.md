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

「Every Definition is Data」が製品機能になる最小の経路は**新しいDSL Parserを先に作ること**ではなく、既存Language Plugin／Operator／Worldの実入口へ、自己記述的Role候補と複数表層のSourceをつなぐこと。関連Draft：

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


## 19. 実製品の一改訂・一Attemptに収束する実証順序（2026-11-03）

> 位置づけ：既存の§4 S1〜S8、§6 H0〜H5、§7、§14、§17、§18の実行優先順位とOwnerを**現在のGitHub証拠で再資格化**する差分。新しい共通Host、第二Canonical／Rule／Query／Evidence Engine、中央Registry、新Gateを提案しない。この追補では新規TEI製品試験を実行していない。

### 19.1 mainに入った資産と未統合候補を区別

| 既存資産 | 再取得した状態・実証範囲 | 同一Runに不足するもの |
| --- | --- | --- |
| [PR #2587](https://github.com/hat47x/tei-platform/pull/2587) | **merged**。Operation Surface、CLI operations/artifacts、決定的Artifact生成。 | Function Context/Ruleへの実束縛と実Provider／現在Authority |
| [PR #2624](https://github.com/hat47x/tei-platform/pull/2624) | **merged**。明示Mappingと非表示Rule依存、合成20＋18テストの資産。 | 実TEI Context・物理Read／実Rule／開示Policy |
| [PR #2626](https://github.com/hat47x/tei-platform/pull/2626) | **merged**。現行Go CLIのSource/Model/Inspect/Trace/Validateと複数Consumerの限定的な実読取り・出典移動の実証。 | 別CLIの実同一Snapshotや外部AI/SHO/Hostへの実到達 |
| [PR #2629](https://github.com/hat47x/tei-platform/pull/2629)、[PR #2634](https://github.com/hat47x/tei-platform/pull/2634) | **merged**。本横断計画・実Go DerivedIRの非出力WHERE/ORDER依存の限定特性。 | 実SQLGlot WorkerとそのSQL/Schema/Dialect実消費との同一Attempt対応 |
| [PR #2638](https://github.com/hat47x/tei-platform/pull/2638)、[PR #2639](https://github.com/hat47x/tei-platform/pull/2639) | **open/unmerged**。Role BootstrapとGo Language Plugin Referenceの候補。 | 現行Role/Plugin Lockの受理、exact-head試験、同じContext/Consumer |
| [PR #2649](https://github.com/hat47x/tei-platform/pull/2649) | **open/unmerged**。実Go CLI validate→operations→artifacts→出力ファイル独立Readerの専用スクリプトとCIを保存。 | HEAD 30a486324c6ca1fe0b5cd16d6e358dc7fa5a8fa0 の専用Workflow [#41503235605](https://github.com/hat47x/tei-platform/actions/runs/41503235605) は**completed/success**。実Go1.26.10 CLIをcheckout外にビルドし、ledger のoperations/artifacts/fresh -out・独立Reader byteを確認（manifest 5ファイル、Reader validate true）。同一WorldFingerprintでもSource移動によりrevision/manifest digestが変化し、旧Operation＋新Artifactの結合はqualified=false。Manifest complete=false／gap=4のままなのでProduct全機能・Consumer必要member充足の証拠ではない |
| [Issue #2387／Draft #2388](https://github.com/hat47x/tei-platform/issues/2387) | Go Client/DerivedIR、Mock Helper負例、SQLite二Provider相当の局所観測はある。 | QX-33 V0/V1の**実SQLGlot解析Producerは対象main tree／実行環境で未特定**。mockを実Workerと呼ばない |

**exact-head CIの実際の失敗例**：PR #2639のHEAD 5fdef5de5956ba0b4996d2861fc1ba4b1d94a222 の [Repository Health #41464532199](https://github.com/hat47x/tei-platform/actions/runs/41464532199)を確認した。Go1.26.10/Linuxで go vet、go test全件、tei validate は**PASS**だが、gofmt -l . は reference/dsl-language-plugin/everything_data_cross_surface_test.go の未整形で**FAIL**。全体run結論はfailure。修正Ownerへ[通知済み](https://github.com/hat47x/tei-platform/pull/2639#issuecomment-6585657796)。以前のHEADのCI成功を最新HEADの成功へ流用しない。

### 19.2 実行可能な最初の二経路と停止位置

**経路A：現行Go CLIで実行できるものを先に横断する（#1163／#2649）**。同じ固定checkoutでビルドしたGoバイナリと、非機密Reference workspaceの隔離コピーを用いる。実 argv と exit、Source tree hash before/after、operationsのmodelRevision、artifacts／Manifest／各MemberのinputRevision、freshな -out directory と別Readerが開いた各file byteを一つの試行として記録。具体的な既存スクリプトは [verify_real_cli_revision.py](https://github.com/hat47x/tei-platform/blob/docs/meaning-to-capability-skill-witness-20261102/plan/fixtures/capability-handoff/verify_real_cli_revision.py) にあるので再実装しない。Source本文byteを保ってファイルを移動する負例では、意味上の出力同値とSource Index/locatorの現在性を区別する。前後tree hash/同一revisionの一致は、Command間atomic snapshot、ABA不在、Host権限、業務受理を証明しない。

**経路B：実SQL解析Producerの所在を確定してから進む（#2387／QX-33）**。先に [Worker実体・起動・依存・受入票](https://github.com/hat47x/tei-platform/blob/work/query-structural-fixture-design-20261030/plan/design/QUERY_QX33_V0_V1_WORKER_READINESS_AND_ACCEPTANCE_2026-11.md) の所在・実artifact hashとInterpreter/依存版、実stdin/stdout、SQL Source/Schema/Dialect、Go DeriveFromAnalysis、非出力Predicate/Diagnosticsを**同一Process Attempt**に結び付ける。所在不明なら V0/V1 = NOT RUN: producer-unavailable を維持。QX-14の模擬subprocess、SQLite Oracle、ResultのQuery ID/Protocol一致は、実WorkerのSQL消費を証明しない。まず active=1 と同Query IDの active=0、同SQL/別Schema、unsupported の差を独立Expectedで比較する。実Producerを確認できた分だけ #2506 Function Context、#1729 Rule入力、#2582動的依存、#2622 Provider、#2516開示へ段階的に渡す。

**優先度の変更**：既に合成試験が豊富な領域は、同形のmock/fixture件数を増やすより**実Go CLI／実Worker／実Provider／実Consumerの隣接境界を一つだけ接合**する実験へ作業時間を振り向ける。異質Domainへの要件探索・独立Oracleの拡張自体は継続してよいが、その件数を横断E2Eの進捗へ加算しない。

### 19.3 既存受入票への追加列（第二の証拠Schemaではない）

| 主張 | 同一Runで必要な観測 | 代表的な反証／停止 |
| --- | --- | --- |
| **Source／実装入力の帰属** | exact checkout、実ロードSource/Schema/Dialect/Role/Operator Pin、Process/binaryと元byte | 同じQuery ID・同じSource digestでも別Schema、旧Worker、ABA |
| **必要入力の閉包** | 明示Context、非出力WHERE/ORDER、Rule依存、中間Relation edge、NOT EXISTSの否定依存、要求母集団 | 出力列が同じ、0件だから依存なし、と判定 |
| **Providerに相対的な現在Claim** | Target/tenant/Scope、各Providerの実Read cohort、Rowset/Field完全性・時点、現在Grant | 二Providerが個別にfullでも共同Snapshotの根拠なし、同名snapshotラベル |
| **実Consumerの受領・開示** | CUI/SHO/AIの実ロード元、公開に使ったSource/Artifact byte、用途別のDisclosure | Skill記述／CLI Catalogの存在だけで実Tool利用・秘密送出資格を主張 |
| **書込み・業務完了** | Accepted Action purpose、Request/実対象/ExpectedState、Host現在Grant、Provider Effect、Receipt、独立Read、Owner判断 | Intent候補、ACK、Receipt ReplayだけからBusiness Completionと判定 |

必要なSQL Worker／Provider／Consumerが未実行、Sourceと解析結果が結び付かない、非出力・否定・Rule依存が不足、Read Cohortが不明、開示・Grantの根拠がないときは、**当該用途のClaimだけをunknown／未実証に戻す**。他用途の読み取り成功や過去のReceiptを取り消す主張ではない。

### 19.4 達成・未達と撤回条件

この追補で実施したのは**main/PR状態と特定CIログの照合、既存H0〜H5へ再利用経路と停止点を割り当てる計画変更**。PR #2649の専用**実CLI+独立Reader** exact-head workflow は前述の限定範囲でsuccess確認済み。一方で、正規repository-healthの当該head総合結論、実SQLGlot Worker V0/V1、Function Context→実Provider→二Consumer→Host/Effectの一体E2Eは**未実施／未確認**。本節から新製品実装・独自中央Engineを導入しない。新しい横断正本が発生する場合は既存#1119/#1163/#2586の受渡しへ戻して撤回する。


## 20. Product Skill／動的依存の接合――「結果が同じ」の先にある現在資格（2026-11-03追補）

> 他レーンの[TEI／TAI／SHO横断引継ぎ（2026-11-02 14:25 UTC）]のHC-01〜10と、同日14:37 UTC追補の空集合・ABA・BRC-13〜16を、既存§17〜19の**Owner・実験入口に割り当てるだけ**の差分。新たな正式Schema/共通Host/Capability Registry/Reactive Engine/Negative Cacheを導入しない。TEI/TAI/SHO各製品は独立して利用可能とする。Source・設計文書の編集正本は従来どおり維持する。

### 20.1 一つの成功に合算してはいけない資格

`Source → Syntax/Role/Operator候補 → Accepted Meaning → Function Data Context + Query/Rule依存 → Provider実Read/現在Read資格 → Operation/Action候補 → 実Product CLI/API → Product Skillに固定された実スクリプト → 外部Harnessが発見・ロード・呼出した能力 → Consumer実受領／現在Disclosure → Host実Effect/Receipt`

上記は**証拠照合の順序を説明するもの**であり、全製品が直列に依存する必須Runtimeではない。特に、(i)Roleをデータで書けた、(ii)CLIコマンドが存在した、(iii)Skill文面にコマンドが書かれた、(iv)外部Harnessの候補に表示された、(v)実CLIがその入力を受けた、(vi)現在Grantがある、(vii)実Provider Effectが成立した、は別のClaimである。

- #988/#2620/#2638/#2639：Syntax/Role・推移依存Pin・実Semantic候補。**同じ構造、同じData ID、同じ型付きBinding結果を業務意味・実Executionに昇格しない**。
- #2506/#2622/#2387/#2582：Value列・非出力WHERE/ORDER/Rule・Relation occurrence・Predicate/Scope/NULL/Grain・Provider read cohort／行集合とFieldの完全性を区別。
- #2605/#2649：CLIのactual help/dispatchだけでなく、**実行されたbinary・argv・Source・ファイル書込副作用**を確認。特に`artifacts -out`をread-only discoveryと同一資格にしない。Skill文書・依存Script・Loader・actual invocationは別の観測点。
- TAI #74/#271、TAI Draft #270/#272、SHO #7/#12、TEI #2516/#2604：公開Tool候補、途中Plan、Renderer DOM/a11y、診断、Network/Log、実Reader bytes、現在Audience/Disclosureを、最終回答と別の開示面として検収する。現在のHost Authorityは既存Host側が所有する。

### 20.2 共有する有限Pressure――新しいケースIDを乱発しない

| Owner付き既存Case | 新たに接合する二つの世界 | 追加で必要な実証／不成立の扱い |
| --- | --- | --- |
| **HC-01〜04**（#2620/#988/#2605、TAI #271） | Source/Skill記述は同byteでも、推移Role/Operator Pin、依存Script、実CLI実装版が異なる | 実Load/Invokeと依存bytesの実取得がなければ`installed/discoverable`止まり。旧版の意味受理・現在資格を継承しない |
| **HC-05〜06**（#2516、TAI #272、SHO #7/#12） | Final Answerは同じだが、候補Tool、途中Plan、DOM/anchor/ARIA、診断、Network/Logが異なる | 該当Consumerに開示してよい実観測surfaceだけ比較。実Renderer/Browser・実送出がなければ情報非干渉の合格としない |
| **HC-07〜08／BRC-13〜15／ND-01〜05**（#2387/#2622/#2582/#2506） | 同じ`[]`、同じ`rows/order`、同じ表示金額でも、predicate・Relation先・Rowset/Field完全性・Provider/時点・Filter/TopN順序が違う | `empty_observation`、`scoped_absence`、`current_absence`、`no_intervening_change`を**比較表の語**として分ける。正式enumではない。否定依存の観測なしなら「不存在の現在Claim」未成立 |
| **HC-09〜10／BRC-16**（#2604/#2630/#1124/#1174） | ManifestはCompleteでも実Readerが旧memberを読んだ、Window1には旧選択応答後着、Window2は旧選択を維持、現在Grantが失効 | 画面/操作/Artifact/Window単位の資格に限定して失効。適格な別Windowや歴史Receiptを無条件に消さず、現在Grantを履歴成功から再発行しない |

**否定依存を持つ0件の例**：`Order O1`の有効保留が0件であるRuleは、返却Row IDが空でも、対象O1・Predicate・Relation path・認可母集団・Field observation・Rowset completeness・時点・Provider路線・集合変化検知に依存する。H22新規加入またはO1の参照先がC1→C2に変われば、表示金額600が不変でも再評価対象が変わる。0→1→0（ABA）の現在0件と「期間中変更なし」を同一視しない。partial emptyを虚偽のabsenceにしない一方、Scope内の一致行が実観測された場合の片側反例を無意味にunknownへ落とさない。これらの判定は既存Rule/Query/Provider/Contextの**実Read witnessを要する**。

### 20.3 §19の二経路から先へ進む最小手順（P0〜P5の既存番号を再利用）

**P0／Evidence freeze**：公開の同一Source/Workspaceと独立Expected、PinされたLanguage/Role/Operator／CLI/Script/Skillの実byte、Query/Rule/Contextの必要依存、用途相対Audienceを固定。引用したMarkdown表やSQL/DOTは原文を保存し、情報欠落Projectionを正式Roleへ自動採用しない。

**P1／実TEI単独**：§19経路Aの実`tei validate→operations→artifacts→fresh -out`と実Reader証拠を、#2666/#2649の既存スクリプト・CIで検証する。現行`commands`のread-only subsetだけで全CLI可用性を断定せず、実`-out`はファイルwrite境界として区別。結果だけでなくexit/Warning/SourceIndex/ModelRevision/Read bytesを記録。

**P2／Query・Role・閉包**：§19経路Bで実SQL解析Producerを同一Attemptへ接続できた場合のみ、非出力WHERE/ORDER・NOT EXISTS/TopN・動的Relation先を明示Function Contextへ渡す。実Producer未特定なら`NOT RUN: producer-unavailable`。#2665のSQLite `SQLITE_READ`はSQL**コンパイル時列参照**の局所証人で、物理Row/OS I/O、実Policy・Provider Cohortの証拠には昇格しない。

**P3／Skillの増分寄与**：同じTask・Source・CLI・独立Oracleで`CLIのみ`／`CLI+Product Skill`を先に比較し、その後別Harnessを使う。実install/load/invoke・誤Tool選択・repair回数・Token・Wall time・価格を同条件で測る。架空のToolを補完した試行、未実行Tool候補の列挙、Skill-only正答をProduct E2E PASSと数えない。

**P4／公開面・Consumer**：同一低権限公開二世界で、Response JSONだけでなく候補表示・Plan・Renderer・ARIA・ログ・実Network・CUI/SHOのLoaded Byteを観測。実送出・閲覧・Policy資格が未観測なら`unknown/not-tested`。秘匿対象を指定しない一般の組織データに、一律のOS隔離要件は強制しない。

**P5／用途別選択再資格化**：`Source-only`、Role/Operator、Queryの非出力依存、Provider route/read cohort、Artifact member、Skill script、Host/Disclosure Grantを一軸ずつ変更。`BR-FULL`対`BR-SELECTIVE`対`BR-UNVERIFIED`を、**誤再利用率と誤失効率が判別できた後**に時間/Token/費用で比較。生成物を再ビルドする必要と実行時データを再Readする必要は異なる。過去の合格は歴史として保持し、現在Claim/Action資格の自動復元には使わない。

### 20.4 責務とEvidence採否

ここでの完成条件は**既存#1119 P0-a/P0-b／G0〜G4／#1163の同一試行票**に、`Task/Claim`・`Source+Role+Operator依存閉包`・`実バイナリ/Skill/Script byte`・`actual argv/Tool invocation`・`要求と実Provider read`・`Query/Consumer観測面`・`必要memberとReader消費byte`・`現在Grant/Disclosure`・`独立Expected/Effect`・`unknown/not-run`・`人手修復/経過時間/Token/費用`を**同じ試行**で記録できること。新しい全局Claim DB/共通Contract Enumは作らない。

**資格**：この§20は他チャットで作成済みの引継ぎ資料と現行PR/Issueを**既存の横断計画へ配置した設計変更**。ここで新たな実CLI/SQLGlot/Provider/Host/TAI Harness/SHOの結合試験を実行したわけではない。#2639の前HEADに発生したgofmt driftは2026-11-03に修正Commit `98966e85acd2abf5efce976a3d09a97c798d6519`（ファイルBlob `97483b653a1f04ffe919e70dbc075dfcc24d4346`）へ変更済みだが、**本節追記時点でexact-head CI successは再取得待ち**。Main統合・正式意味採用・Owner受理を示すものではない。

## 21. 汎用基盤と用途相対の意味契約――#2639 EG-09からADR・回帰へ（2026-11-03）

> **位置づけ**：2026-11-03の「汎用基盤性」横断レビューを§17〜20と既存Ownerの実験へ引き継ぐ、採用前のADR判断候補。新Canonical／Meaning Registry／同値判定Engine／全局Dependency Graph／TEI独自Hostは導入しない。既存ADRの改訂・新ADR採用・Owner受理・実装成功の宣言ではない。

### 21.1 四つの「同じ」を使い分ける

| 比較する層 | 汎用機構が照合できる範囲 | それだけで成立しないClaim |
| --- | --- | --- |
| **Authoring Source / occurrence** | 元のbyte・元の構文糖衣・出典位置・読込版・Parser選択 | byte/Meaningが同じことからの引用Locator/SHO Reviewの現在性 |
| **Structural Candidate** | Data/Attribute ID、型、Key/Relation、Role/Operator候補・依存Pin | 税区分・単位・丸め・時点・対象集合・業務上の同一性 |
| **Accepted Meaning / 用途別互換** | 受理済みMeaning/Rule/Operator契約、依存、対象Functionの前提条件 | 別目的への無条件の代入可能性、実Provider取得・現在Grant |
| **Observation / Current Qualification** | 実Query/ProviderのScope・Rowset/Field完全性、Readerが読んだArtifactと版 | 結果が同じことからの現在の不存在、実Action許可、業務完了 |

**汎用性の中心は用途相対の互換性**。たとえば sales.order.amount／std.decimal／1000 が同じでも、税抜金額から税込金額へ意味が変わった場合、画面表示の数値一致と請求Ruleへの代入互換は別。互換性Claimは purpose／accepted meaning & rule／subject & grain／Context & read scope／Provider & time／Consumer の限定条件に結び付ける。これは**既存受入票の列**であり、全Domain共通の意味同値bool/APIを新設するものではない。

**現行実装と反例**：main の reference/dsl-language-plugin/pinned_semantic_binding.go は Namespace 内の指定Data ID一意性と、暫定 rows/<id> 型でのexact-pinned Operator合成を確認する。Attributeの業務意味や全Attribute依存・Source occurrenceの互換性は契約外。未統合[PR #2639](https://github.com/hat47x/tei-platform/pull/2639)のEG-08は実Semantic Parserとtest-only JSON adapterの限定接合、EG-09は同Data ID/限定Binder出力で属性TypeRef差が検出対象外になる反例。**Binderの成功は意味同値やBuild再利用の証明ではない**。main の internal/semantic/model.go のCandidateKey/FunctionalDependencyも業務Assertionの宣言であり、実DB検証/Owner意味受理ではない。

**別Domain圧力**：[Draft #2681](https://github.com/hat47x/tei-platform/pull/2681)のPerson/OrgUnit/Membership/Position/OccupancyではMembershipとOccupancyを別identity/Roleとして扱い、同型の状態列から就任・組織承認・実Grantを自動生成しない。Graph/ゲーム/ロボットの時間粒度や物理単位にも同じ原則を適用する。

### 21.2 既存INV-1〜7の横断停止条件（新Status型ではない）

| INV | 既存担当と比較境界 | 不足時の停止 |
| --- | --- | --- |
| **1 意味資格** | #2620/#988/#2514、ADR-0022、受理済み定義対応#2644 | ID/型/値一致はMeaning受理不可。AI候補やSyntaxの自己宣言から正式受理を推定しない |
| **2 推移Pin** | #2638 Root/Eager Roleと#2639 Language/Operator | Rootの版名だけ一致してもContract/実装Fingerprintと推移依存不足なら再資格化。合法Deferred Ref循環とEager循環は別 |
| **3 取得Scope** | #2506明示Function Mapping→#2387/#2622 Query→#2611実Provider | Declared read、SQLITE_READ列参照、実Acquisition、結果、公開Projectionを混同しない。Rowset完全でも必要Field未観測ならRule unknown |
| **4 不存在** | #2582/#2627、独立Expected#2514 | 0件/未取得/partial/completeと現在不存在Claimを分離。Predicate/Relation/母集団/時点/Providerと変化検知の裏付けが必要 |
| **5 動的依存／再利用** | #2582 Relation付替・集合membership・ABA、#2600 Build reuse | Output digestや終端0件が同じでも期間無変更とは限らない。Build再生成とRuntime再Readは別判断。不要な全局失効も避ける |
| **6 Consumer** | #2604/#2630必要Artifact Member、#2516開示、#1124 Source追跡 | Artifact CheckSet内部一致/Manifest Completeだけで必須Member・同世代・実Reader取得・現行開示資格を代用しない |
| **7 実行資格** | #2598 typed Intent、#1174既存Host/Provider、#2637二地点Grant | Action候補/過去Grant/ACK/Receiptから現在権限・Commit後業務完了を導かない。不明Commitは不明を維持 |

**量化と非単調性**：結果 [] でも WHERE/NOT EXISTS/TopNのPredicate、Relation occurrence、フィルタ対象母集団、RowsetとFieldの完全性、Provider route/read cohort、時点への**否定依存**が残る。0→1→0（ABA）で現在0件は再確認できても「期間中変更なし」は推定できない。partialでもScopeに属する一致行を明確に観測した場合は片側反例を保持できる。Filter→ORDER→LIMITとLIMIT→local Filterの交換は、条件なしに合法としない。異Providerの同名snapshotラベルは共同Snapshotの証拠ではない。

### 21.3 ADRへ記録する判断候補／保留

[採用済ADR-0022](https://github.com/hat47x/tei-platform/blob/main/plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md)のAI/Editor/CLI Proposal→Shared Acceptance、AIのautonomy≠authorityを**既決事項として再利用**する。

| 判断案（提案・未採用） | 固定すべき原則 | Owner |
| --- | --- | --- |
| G-D1 構造≠意味 | Everything is Dataは表現と意味解釈部品の合成の汎用化。業務意味の自動統一ではない | #2619/#2620/#988 |
| G-D2 用途相対互換 | ID/Type/値/限定Binder出力の一致は意味同値の十分条件でない。受理済みMeaningと用途/Rule/時点を要求 | #2620/#2506/#2514 |
| G-D3 Read/Claim分離 | 明示取得、Provider実Read、Rowset/Field完全性、Disclosureを独立に照合。0件と未取得を同一視しない | #2506/#2622/#2582/#2516 |
| G-D4 Reuseの条件 | 推移Role Pin、動的Relation/Predicate/非出力Query、集合membershipまで必要依存閉包を確認。BuildとRuntimeを別失効 | #2638/#2387/#2582/#2600 |
| G-D5 Source Fidelity | SourceのMarkdown/DSL糖衣を普通の保存で全展開・無断整形しない。Meaning不変でもSource locator/文書Reviewは別に再資格化 | [main統合 #2674](https://github.com/hat47x/tei-platform/pull/2674)、#2621/#1124 |
| G-D6 Consumer/Authority | Role/CLI/Skill/Artifact候補≠実Consumer受領/現在Grant/実Effect。外部Hostの既存仕組みに委譲 | #2649/#2604/#1174/#2516 |

**判断保留**：汎用Semantic Equivalence Schema/API、共通Dependency Closure型、Native SQLと標準Operatorの完全一般同値、全Provider共通Snapshot契約、全Consumer共通Grant Token、Go単一バイナリに対するPlugin配備方式の固定、万能Lossless CST/Writer、業務Ruleの暗黙の税/丸め/単位/有効時点、正式DSL Grammar。未実証のため新Engine/Registry/Hostを前提にしない。

### 21.4 最小回帰GR-01〜10（既存OwnerのFixtureへ割付、横断実行NOT RUN）

| GR | 一軸変更の比較 | 検出を担う既存Case／不足 |
| --- | --- | --- |
| **01** | Data/Attribute ID、std.decimal、値1000不変で税抜→税込Meaning変更 | #2620/#2514：業務Owner受理なしに同値扱いしない |
| **02** | EG-09の同Data IDでAttribute TypeRef decimal→string | #2639 EG-09：限定Binder output一致≠Meaning・増分再利用同値 |
| **03** | Root Role版は同じ、Eager依存Operator実装Fingerprintだけ変更 | #2638 RB-20〜24：依存実装のPin漏れで旧受理を継承しない |
| **04** | 同じ [] でも全page/必要Field complete vs 未取得 | #2582 ND-03/#2627/#2665：後者は不存在not-established |
| **05** | 結果が0→1→0、最終値は初期と同じ | #2582 DC-07/BRC-14：現在値と履歴無変更を区別 |
| **06** | Filter→ORDER→LIMIT1 vs ORDER→LIMIT1→local Filter | #2387/#2622/#2640：非可換の反例、黙示Pushdown禁止 |
| **07** | O1.amount=600不変、customerRef C1→C2で信用限度700→500 | #2582 DC-02/#2506：動的関係先再解決、無関係画面の過剰失効なし |
| **08** | Primary→Replica、文字列Snapshot s7は同じ | #2627/#1708：共同Read Cohortを推定しない |
| **09** | Artifact内部整合OK、Consumer必須Member欠落/Reader異世代 | #2630/#2659/#2649：実Readerと必要Memberを照合 |
| **10** | Provider Read中Grant失効/Consumerへの実送出scope変更 | #2637/#2516/#1174：二地点再確認はABA・物理開示の完全保証ではない |

**既存ケースの横断適用**：[main #2674](https://github.com/hat47x/tei-platform/pull/2674) SF-01/07/13/17のNo-op Source byte／二Occurrence／Source移動／AI全Format候補をGR-01/02/09のSource境界へ接続。#2681のMembership/OccupancyをGR-01/07/10へ置換し、組織上の状態から実権限を自動推論しない。SQL/DDLのFK/Statusの構造事実から業務Transitionや実Migration成功を推測しない（#1126/#1129/#2622）。既存Fixtureと独立Expectedを再利用し、10件の別製品実装・第二Oracle Engineは増やさない。

### 21.5 実証順と現在の停止点

- **E0 局所Reference**：#2639 Go Binder、#2638依存Pin、#2674 Source Fidelity（設計法則は実Writer NOT RUN）、#2665 SQLiteコンパイル時READ、#2659 Artifact CheckSetを**別々の版・環境・局所資格**として記録。局所PASS件数を合算しない。
- **E1 実Go CLI**：§19経路A・#1163・#2649の実バイナリ/実Source/argv/Operation/Artifact/独立Readerを利用。同じ固定Sourceで受理Meaningと実Contextまで接合したと推定しない。
- **E2 実Query/Context/Provider**：実SQL解析Producerのexact artifact・起動・SQL/Schema/Dialect/Go DerivedIRを確認してから、非出力/否定依存を明示Context/Providerへ接続。実Producer不明なら NOT RUN: producer-unavailable。SQLITE_READはコンパイル時列参照で、実物理Read/OS開示ではない。
- **E3 ConsumerとBuild/Runtimeの再資格化**：対象Claim別にRequired Member・Reader実byte/世代、Source Locator、現在Audience/Grantを比較。GR-05/07/09でBR-FULL／BR-SELECTIVE／BR-UNVERIFIEDの誤再利用・過剰失効を測る。
- **E4 実EffectとOwner**：実Request Target/Expected State/現在Host Authority/Commit unknown/Receipt/独立Readと業務Meaning受理を分離。#1119 P0-a/P0-b/G0〜G4の優先順位は変更しない。

**既存#1163 Evidence票に残す最小列**：Source元byte・Occurrence・改訂/Parser版、Data/Attribute/Relation identity、Type/必要Facet、受理Meaning/Rule/Role/Operator依存Pin、purpose/subject/grain/Scope、宣言/実Provider取得とRowset/Field completeness/epoch、動的Relation/negative dependency、必要ArtifactとReader実消費byte/世代、現在Consumer/Disclosure/Host Grant、独立Expected・Actual、established/contradicted/not-established/not-run、必要なBuild再生成・Runtime再Read・Owner再受理の範囲。**新共通Evidence Schemaではない**。

**2026-11-03再照合の証拠状態**：Draft #2639 head 98966e85acd2abf5efce976a3d09a97c798d6519 のgofmt修正Blobは保存済みだが、PR-triggered workflow取得結果は空で、このHEADの正規CI成功は未確認。Draft #2667の追補前head 0e9ea97ccbe47067b2143287736e12682b4985f7 の[Repository Health #41510572157](https://github.com/hat47x/tei-platform/actions/runs/41510572157) は **failure**：ログ上は internal/validation/validation.go のgofmt drift。Docs追補の意味試験FAILや他のGo test全体PASSへは読み替えない。この§21による新Product/Provider/SHO/Host統合試験はNOT RUN。別OwnerのGoコードやADR本体は変更せず、Draftのままレビューする。

## 22. DSL可読性から実行契約への引継ぎ――Frontend選定とGo生成を分離する（2026-11-03、計画追補）

> **状態：設計・比較PoC候補／NOT RUN／正式構文・Frontend採用・Owner意味受理は未決。** 本節は§19〜21と既存#988/#2514/#2621/PR #2639を結ぶ引継ぎ票であり、実装済み契約、新Schema、第二Canonical、新Runtime Host、追加Gateを宣言しない。Go単一バイナリは標準実行形態の方針、.NET/Java等は必要時のTargetとするが、全生成物のビルド・意味適合を実証したという意味ではない。

### 22.1 選択肢の直交性と停止原則

- **利用者価値が第一**：ドメイン専門家に `package` / `import` / `struct` / `func` / Go型引数を普段のスクリプトで強要しない。構文の少ない記述でも、型・業務意味・Authorityの検査を弱めない。日本語はサンプルのDomain識別子であり、DSLを日本語専用にする意図はない（UTF-8識別子・論理名表示を別検証）。
- **A Go式AST**：Go互換部分式に限る。Go予約語 `select` とパイプ `|>` などはそのままGo式として解析できない。可読性を犠牲にして語彙を変更させるなら棄却候補。
- **B TEI独自Frontend**：外側の宣言・Pipeline・クエリを自然な構文で解析する。新Parserを構想する前に既存Language Plugin/Parserの境界を確認する。
- **C Hybrid**：TEI構文解析とGo式ASTの部分再利用。前処理でSource位置が曖昧になる、二重の文法/Resolverが必要になる場合はBへ戻す。
- **D Goコード生成**：A/B/Cとは排他的ではないBackend。どのFrontendでも共通の固定済み意味契約からGoコードを生成する候補。Go AST採否と標準Runtime方針を連動させない。
- **停止原則**：Parser成功≠型・意味受理≠生成成功≠実行資格≠業務効果。新しい共通Host/Registry/Meaning SSOT/独自Oracle Runnerは作らない。Native SQLは明示Source/Bindingのまま保持（#2387/#2622）。

### 22.2 Frontendを跨ぐ引継ぎの最小不変条件（新公開Schemaではない）

| 欄 | 保持すべき情報 | 拒否・不明扱いとする反例 | 既存Owner |
| --- | --- | --- | --- |
| Source／文法 | Authoritative Source byte/occurrence/span、Source locator・revision、Language/Grammar Plugin選択・版、Syntax projection | Go生成後の行列位置で原文診断を代用、移動後locatorを未再解決 | #2621/#988 |
| Meaning／受理 | Subject ID・Role・意味解釈依存・Accepted Meaningの版/範囲/Owner受理状態 | 同ID/同型・Parser一致から意味同値/採用を推論、Owner撤回後の自動復活 | #2619/#2620/#1126/#988 |
| Type／Nullability | 論理型・値域・単位・Decimal精度/丸め契約、nullableとactual presenceの区別、Operator ID/実装版 | Goのfloat64/ゼロ値へ黙示変換、未定義丸め規則の補完 | #988/#2622/#2514 |
| 入力観測 | missing/explicit-null/not-fetched/withheld、集合のgrain・multiset・field/row completeness・scope/epoch | 未取得・非開示をNULL/0件、部分取得を全件と解釈 | #2621/#2622/#2514/#1708 |
| 参照／依存 | relation path、候補と曖昧診断、静的・否定/動的依存、実選択入力の版 | 複数候補の先頭採用、同じ結果値から依存不変を推論 | #988/#2600/#2639 |
| Expected／診断 | Candidateから独立したExpected/Oracle出典・版・適用Claim、安定diagnostic kind ID、Subject/occurrenceに結びつく発生追跡 | GotからWantを生成、Source行番号や診断文言を安定IDとして流用 | #2514/#2621/#1649 |
| Go生成物 | Semantic/Operator/Generator/Toolchain/Target版、生成ファイルと原文のSource map、実Build input/ArtifactSet | 生成成功・byte一致だけで意味同値・実Consumer利用を主張 | #1063/#2595/#2600/#2604 |
| 評価／保存 | 計算値・not-satisfied/unknown/rejected、Operation Intent、現在Authority、Provider Attempt、Commit known/unknown、独立Read/Receipt | true/exit 0/ACK をCommit/保存効果・現在認可・Owner受理へ昇格 | #1129/#1174/#2514 |

安定診断は **kind code** と **発生対象（Subject・依存・Source occurrence）** を分ける。Source移動で行番号が変化しても同じ問題を追跡でき、原因/対象が変わった際には別問題と判定できることを試す。診断コードの正式一覧は本節で制定しない。

### 22.3 処理層と責務：実行契約はFrontendから独立

| 層 | 担当 | 担当しないこと |
| --- | --- | --- |
| Frontend（A/B/C） | Token/構文/元Source位置、構文診断、選択済みLanguage PluginのSyntax projection | 業務意味/実行Authority/Provider成功の承認 |
| Semantic Analysis（既存Canonical/Role/Validator/Operatorを再利用） | 型/Nullability、Reference・曖昧性、Operator pin、明示Data Context・grain・Scope、静的/動的依存の必要情報、意味診断 | 実データ完全取得、Commit、Ownerの黙示受理 |
| Code Generation（D） | 意味に従うGo型・Decimal実装・Runtime契約Binding、Source map、Generator/Toolchain/Artifact依存 | 未定義税/丸め/単位を推測、生成物成功から意味受理を推論 |
| Runtime＋既存Provider/外部Host | 入力実測（取得済みと未取得の区別）、評価結果/失敗、実Request/現在Authority、Commit/Receipt/独立Readの観測 | TEI独自Hostの新設、業務意味の再定義、ACKだけで保存完了を保証 |

**引継ぎの停止条件**：未受理Role、unresolved/ambiguous ref、必要Type/Ruleの不明、必須Reference stale/unavailable、完全性不明の全称/不存在、必要Contract/Target非対応、現在Authority不明のeffectは、それぞれの段階で明示的に停止・unknown・unsupportedとする。Offlineの固定Snapshotから過去の説明/計算を再現できても、現在のSource鮮度/権限・実Provider実行を保証しない。A→B→Aでは内容再利用候補と現在Owner受理・実行資格の再評価を分離する。

### 22.4 最小同一Fixture――可読性試験とConformanceを別採点

**Domainと前提**：受注・明細・顧客。可読性試験では「明細金額 = 数量 × 単価」「受注明細を集約」「受注確定可能条件」をA/B/Cの三表層で提示する。Go専用型宣言なし、モデル/Operator版を固定。原文の日本語識別子は例示、グローバル適用時は英語その他UTF-8識別子でも同じMeaning/診断を評価する。Dは生成Backendとして別判定する。

| 試験 | 独立Expected／禁止推論 | 判定Owner・既存Fixtureの接続 |
| --- | --- | --- |
| **DLC-01 表層** | A/B/Cが意味同値と認められた式では同一のOperator/Subject/型へLower。Go `select`/ `|>` を許さない制約を利用者語彙へ押し付けない | #988、PR #2639 EG-08/09。正式Grammarを決めない |
| **DLC-02 Decimal** | 数量3、単価19.995は中間値59.985。scale=2への最終丸めは規則不明なら拒否。明示modeの正負tie・未対応modeを分離 | #2514 N/TAO、#2016 H5-O1/#2291既存反例 |
| **DLC-03 Presence/集合** | 0、explicit NULL、missing、未取得、非開示は別。L={A,B}, R={A}の部分結果だけでは全件SUM/「不存在」成立を宣言しない。取得済みBの反例は適用範囲つきで全称を否定可能 | #2514 C、#2622、#1708 |
| **DLC-04 Relation/動的依存** | 同名候補・1:N fanout・Subject同IDでAttribute Typeだけdecimal→stringを別診断/影響とし、Binder出力一致から再Build安全を推論しない | #988、PR #2639 EG-09、#2600 IB-07/10 |
| **DLC-05 Source fidelity** | Markdownの改行/コメント/CRLF変更、Source移動、元言語の構文エラーで、原文範囲/安定診断kind/発生Subjectを保存・再解決。無変更Sourceはbyte不変。Go生成エラーを利用者原文へ逆写像 | #2621、SHO/MMI C-01/T2、#2674 Source Fidelity |
| **DLC-06 実行とEffect** | 同一意味・実入力・Targetで計算値と拒否/unknownを比較。true、正常終了、Provider ACKのどれも実Commit・保存効果を保証しない。Commit unknown→Receipt/独立ReadとOperation identityを別確認 | #2514 W、#1129/#1174、#2595/#2604 |

**可読性評価の観測**：業務条件/入出力の説明正答、誤読種別、修正箇所の発見、編集所要時間、Go知識への依存、未取得・NULL・部分集合の誤解。対象者・課題・提示順を記録。結果は利用者評価であってConformance PASSではない。閾値は事前にOwnerが定め、事後に都合よく調整しない。

**Conformanceの観測**：上記DLC別のindependent Expected、既知mutantの排除、Source/Operator/Build/Run/Consumerのactual binding、入力scope/epoch/grain、diagnostic code+source map、実計算とEffect Receiptの別結果、現在資格。正負・unknown・not-runを分け、構文成功やGo生成成功だけを合格にしない。#2514のOracle Soundness/Evaluation Binding/Claim Coverageを再利用し、新Runnerや新Oracleを作らない。

### 22.5 方式切替条件・縦切り順序・ADR候補

1. **P0 Source→Meaning（read-only）**：PR #2639の既存Go Language Plugin Reference/限定BinderでDLC-01/04を照合し、Go式対応境界とSource spanを確認。PRの現HEAD exact CIが未確認なら `NOT VERIFIED` のまま。局所EG PASSを全意味同値へ昇格しない。
2. **P1 可読性＋否定Fixture**：A/B/Cの実際の業務記述でDLC-02/03/05を比較し、ドメイン専門家の読解・編集と、独立Expected/診断の別採点を行う。Go文法への譲歩で可読性が悪化する、またはCの前処理・Source mapがBより複雑になる場合、A/Cを棄却してBへ切り替える。方式選択は実測後のOwner判断。
3. **P2 Go生成の最小計算縦切り**：同じ受理済み意味・固定ContractからDLC-02/03の小さな純粋計算をGoへ生成し、実Build binaryと独立Expectedを比較。DはBackendでありA/B/Cと同一投票にしない。実Build input・ArtifactSet/実Reader・CI exact HEADを固定。
4. **P3 Runtime/保存の別縦切り**：既存#1119/#1163/§19経路Aと#1129/#1174を使い、実Provider/Hostがあるケースに限定してIntent→実Request→ACK/Commit unknown→独立観測を検査。純粋計算のPASSを保存まで拡大しない。標準Go binaryの実績を.NET/Javaへ転用しない。

**方式選択の判定**：AはGo互換で自然に読める式に限る。Cは既存Go AST再利用の便益が、二重文法・Source逆写像の複雑さを上回る場合のみ。Bは自然な文法がGo制約で損なわれる場合の正常な選択肢。全方式共通で型/Nullability/完全性/曖昧性/Source/diagnostic/独立Expected/実効果分離に違反すれば採用停止。Go単一バイナリとTEI独自Hostなしの上位方針は維持。

**ADR候補（番号未付与・未採用）**：`DSL Frontend Selection and Source Fidelity`。利用者可読性優先、FrontendとBackendの直交性、Semantic/Runtime契約の不変条件、棄却/切替条件、Go AST利用範囲を記録する案。既存ADR-0022（AI候補と意味受理）、ADR-0023（命令後独立Read）、ADR-0032/0033（構成・Plugin）、ADR-0043（TEI独自Hostなし）を変更・重複せず、#988の採否ゲートで検討する。

**既存Issue/PRへの最小差分**：#988はFrontend→Semanticの不変条件とGo式/独自構文のPressureを所有。#2621はSource occurrence・版・原文往復/安定診断のOwner。#2514は独立Oracle・mutant・Claimの実用資格。PR #2639は現有9件の限定Go Referenceの証拠であり実Runtime試験のOwnerではない。#2600/#2595は依存閉包・実Build入力、#1129/#1174はEffect/Authority、#2667（本節）は横断引継ぎのみ。**この追補による新試験/新コード/新CI実行はNOT RUN。正式DSL文法・Parser方式・Schema・Owner受理は未決定。**

## 23. DSL実装選択の実証順序――言語中立・原文保存・Bootstrap・現在資格を同時に守る（2026-11-03追補）

> **計画／NOT RUN・NOT ACCEPTED。** 本節は§22の「可読性→意味契約→Go実行」を最新の別レーンへ接続する**差分のみ**。正式DSL Grammar・Unicode Profile・Go AST/Hybrid/独自Parser・公開API・Go生成器・Plugin方式を採用するものではない。#1119の既存Gateを増設せず、#988/#2514/#2621/#2600等のOwnerを変更しない。

### 23.1 §22を補正する二つの条件（#2745との重複は参照で解消）

1. **グローバル共通の言語中立構文**：§22の日本語「明細金額」「数量」等はローカルDomain識別子を表す**説明用例**であり、日本語の予約語や日本語専用DSLを正式候補にしない。初期評価の比較例は英語の軽量構文 `amount = quantity * unitPrice`、`lines |> where quantity > 0 |> sum(amount)` 等とし、`|>` 自体も未採用候補。日本語/他Unicode文字のローカルSymbolと表示言語は**構文比較と直交**して検証する。Goの `select` 予約語・`|>` 非対応を利用者へGo型宣言やGo固有語彙を強要する理由にしない。
2. **個人OSSで現実的な利用者評価**：§22にある「業務専門家を集めた読解時間/正答率測定」は、将来、適格な協力者がいる場合の追加方法であって**今回のP0/P1完了条件ではない**。現時点の第一段階は、同じモデル/問題/解答を使う**AI支援の盲検的な表層比較**（構文ノイズ、概念提示順、局所修正箇所、曖昧性の指摘）と、独立Expectedによる機械Conformanceを分離する。AIの評価一致や推定読解時間を「ドメイン専門家が理解できた実測」と報告しない。比較の採否はOwnerが証拠を見て判断する。詳細なGN-01〜16/U0〜U4と五種類の名前の契約は[Draft #2745](https://github.com/hat47x/tei-platform/pull/2745)がOwner。**#2667からその文書を重複編集しない。**

**現行制約**：既存`core/identity/id.go`のCanonical ID/Local NameはASCII制約。Unicode表層識別子を使うには、選択済みLanguage PluginのSource local Symbolと既存Canonical ModelRefの**明示Binding**を検証する。物理名、論理名、ローカルSymbol、Target生成Symbol、Stable IDを混同せず、翻訳・字面類似・NFCの自動正規化でMeaningを決めない。Studioの表示名切替はSource/Meaning/Targetを書き換えない。Unicode識別子のProfile、Bidi/Confusable警告、実ToolへのAI入力のAuthorityは別の受理対象（#988/#2621/#1124/#2515/#2516）。

### 23.2 既存資産との責務と証拠段階

| 境界・再利用先 | 確認した状態 | DSL PoCでの扱い／まだ言えないこと |
| --- | --- | --- |
| **#988** Language Plugin/Operator Contract | Pin、Source span、Operator/NULL等の**局所Reference報告**。正規Semantic Acceptanceと実Providerは別 | A Go式AST・B既存Parser/TEI独自構文・C Hybridで同じ**意味契約へ解決できる範囲**だけ比較。二重Resolverを作らない |
| **merged #2638** Role Bootstrap | 有限Layer0、eager/deferred、推移Pin・実装identityの**合成Reference/設計文書がmainへ統合** | 同名Role/CarrierだけでFunction/Ruleを実行しない。Eager循環と未Pin推移依存は診断。正式Core bootstrap・全Plugin実動の実証ではない |
| **merged #2674** Markdown Source Fidelity | Source sugar/コメント/改行等を守る**設計文書がmainへ統合** | Save≠Format、no-op書込みゼロ・byte不変、Occurrence単位の最小Patch、競合時read-only/Preview。実Source Writer/Studioの完了ではない |
| **Draft #2639** Go Language Plugin cross-surface | EG-08/09で複数表層と型差反例を限定的に検証した旨のPR記録 | Binder出力/同IDの一致≠業務Meaning/NULL完全同値/増分Build再利用可。**現HEAD exact CIは別確認** |
| **Draft #2745** 言語中立/Unicode/AI | 五名称・明示Binding・GN-01〜16・Owner比較の**設計候補** | 日本語DSL正式採用、Unicode Lexer実装、Studio表示切替、Prompt injection耐性は未実証 |
| **#2600/#2595** Build/実Source束縛 | IB-01〜10、Go S4-00とScript ABAの既存残余試験計画 | DSL parse結果と**実Buildが読んだbyte/依存閉包**を区別。Digest/同値/局所PASSで再利用資格を推定しない |
| **#2514/#1129/#1174** Expected/実Effect | 独立Oracleと保存/Authority/Receipt境界の既存Owner | Parsed・Semantic・Built・Evaluated・Persisted・Effect observedを別Claimとして扱う |

### 23.3 追補すべき4本の反証型Fixture（新しいFixture群を量産しない）

| ID（比較票ラベル） | 入力差分・反例 | 独立Expected／適用停止線 | 既存担当へ割付 |
| --- | --- | --- | --- |
| **DLX-01** 表層/表示と意味 | ASCII `amount = quantity * unitPrice`、別構文、UnicodeローカルSymbol版。さらに同じIDでDecimal型→String型（EG-09） | 明示PinとBindingが同じ**かつ**必要Role/Rule/型が同じ場合にのみ対象Claim相対の同値を比較。Unicode同形/翻訳からBindingしない。変更なしSaveはbyte一致、診断は元Source byte spanへ | #988、#2621、#2639 EG-09、#2745 GN-01/12/16 |
| **DLX-02** Bootstrap/Reference鮮度 | Root Role Pinは有効だがeager依存Provider実装が未Pin、外部Referenced Sourceがmove/changed/offline/stale | 循環/未Pinは停止・diagnostic。移動/同byteはlocatorの再解決、内容変更は依存Meaning再検査。Offlineの旧Snapshotは過去の説明/限定評価に使えても**現在の受理・実行・開示資格には代えない**。取得状態とAccepted Meaningの履歴を分ける | #2620/#988（merged #2638参照）、#2621、#2600、SHO/MMI C-01/T2 |
| **DLX-03** 部分集合・否定依存 | `lines |> where ... |> sum(...)`、0件complete vs 0件partial、未取得Field、動的Relation先変更、A→B→A | emptyが「存在しない」を意味するのは対象集合・Field/row completeness/Scope/時点が証明された時だけ。部分Readの局所反例は全称否定に使えても全件肯定は不可。末尾が同値でも動的/否定依存と現在資格は再評価 | #2514 C/M、#2622、#2600 IB-07/09/10、#2582 |
| **DLX-04** Go生成/実効果 | 明細3×19.995の中間59.985、丸め未指定/明示mode、生成Goの型/演算/Source map、業務条件true後のProvider ACK喪失/Commit unknown | 丸め不明/未対応なら拒否。Go `float64`やゼロ値への黙示変換なし。生成後の独立Expectedとactual loaded artifactを照合。計算Got/Intent/Provider attempt/Commit status/独立Readは別Claim。ACKなしは「効果なし」とみなさず、無条件再試行しない | #2514 N/W、#2595/#2600、#1129/#1174、#2604 |

**DLX-02のSource鮮度**：`fresh/stale/unavailable/partial/not-found/permission-denied`は外部Referenceの観測・診断を分類する**比較候補**。Source URL/Revision、fetch time、content fingerprint、Source occurrence、Cache、Owner受理の関係を照合する。内容A→B→Aで旧Observation/Receiptの履歴を残しても、現在利用資格を自動復活させない。新しいSource Registryやプロダクト共通Freshness Engineは導入しない。外部実取得・offline/権限喪失の実試験は**NOT RUN**。

### 23.4 Go実装とPluginの最小境界

- **Frontend**：既存Language Pluginで固定された文法選択→元Source occurrenceのToken/parse→変換候補。Go ASTは互換式の部分的技術に限る。バイト位置・Unicode境界・元構文/コメントは原Source側で管理。Bへ切替可能な中間表現を採用前のPoC範囲で比較。
- **Semantic**：既存Canonical/Role/Operator/Validatorで型・Nullable、missing/not-fetched/withheld、grain/完全性、曖昧Relation、依存Pinを検査。syntax成功を意味受理に昇格しない。有限Bootstrapの解釈根は既存Lock/Resolverへ従属し、自己記述の循環で自身を正当化させない。
- **Generation**：受理済みMeaningからGo単一バイナリ用SourceとSource map、Artifact/Toolchain依存を再現可能に生成する候補。構文Parser方式に固有のGo生成ルートを増やさない。.NET/Javaは必要時の追加Target。**静的合成/動的ロード/外部Processは更新・隔離・再現・Host認可の反例がそろうまで未決**（#951/#988）。
- **Runtime/Provider**：計算の観測とCommit/独立Readを分離、read-only表示とaction executableも分離。TEI独自Host・中央Registry/Policy Engineを作らない。DSLの簡潔さがDBのSave phase、Provider capability、現在のHost認可を省略する根拠にはならない。

### 23.5 既存計画への実行順とDone/Stopの区別

| 順 | 実証・依存 | Doneと呼べる範囲 | 停止/Owner判断 |
| --- | --- | --- | --- |
| **P0** 版・Owner固定 | main統合済み#2638/#2674とDraft #2639/#2745/#2667、参照Source/Plugin/Model/Oracle版の一覧、existing Parser/Role/Source Ownerの再利用点を記録 | 古いCIや別HEADのPASSを転用しない。未受理/未Pin/未取得があればそのClaimをhold |
| **P1** 人間向け構文比較 | 言語中立軽量DSL/Go式/Markdown/chainで同一Subject/Expected、AI支援による表層ノイズ・認知順序・局所編集の比較票（**人間の読解実測とは呼ばない**）、GN/Source negative | Go文法都合で専門語彙を曲げる・CのSource逆写像がBより複雑ならA/Cを棄却候補。正式GrammarはOwner待ち |
| **P2** 既存Go Reference | DLC-01〜05とDLX-01〜03を既存#988/#2639/merged #2638/#2674に結び、実Parserの診断とSource位置、独立Expected・動的依存を照合 | 同じBinder結果のみで意味同値・現行資格・再Build skipを宣言しない。Parser/Writer/Source 実コード未到達ならNOT RUN |
| **P3** 最小Go生成／実Consumption | DLC-06/DLX-04の純粋DecimalとNullabilityをGo単一バイナリで限定試験、実Build入力・ArtifactSet・Readerが読んだbyteを照合しExpectedと比較 | Runtime/Provider/DB/SHO/Host実行は別証拠。保存Actionは適切な既存Ownerへ引き渡す。構文採否には自動昇格しない |

**既存文書との差分だけを更新**：本§23は§22の日本語例を言語中立構文へ再位置付けし、実際の人間被験者なしで行える比較へ補正した。国際化・Unicode詳細は[#2745](https://github.com/hat47x/tei-platform/pull/2745)、Source保存はmain統合[#2674](https://github.com/hat47x/tei-platform/pull/2674)、Bootstrapはmain統合[#2638](https://github.com/hat47x/tei-platform/pull/2638)を参照し再実装しない。ADR候補は§22に記載した無番号のDSL Frontend/Source Fidelity判断へ結果を引き渡すだけとし、既存ADR-0022/0023/0032/0033/0039/0043を勝手に変更しない。ConformanceのOracle/現在資格は#2514、増分/実Source帰属は#2600/#2595、実Effectは#1129/#1174。**本追補は計画文書のみ。新PoC、実Go CLI、実Provider、Studio/SHO/AI Harness、Unicodeの実Lexer、CI exact-HEAD、Owner受理はいずれもNOT RUN。**

## 24. DSLからGo実行までの一改訂縦断――共通契約の実消費を確認するための受入票（2026-11-03追補）

> **状態：具体的な実証計画。テスト／Go生成器／実Providerの実装・実走は本節でNOT RUN。** §22の契約、§23の実装選択・国際化・Bootstrap・Source Fidelityを繰り返して別仕様にせず、**実際にどのbyteがどの段階で消費されたか**と**どこまでのClaimが成立するか**を、既存#1163の一改訂Evidence票へ渡す。#1119のP0-a/P0-b/G0〜G4の順序を変更しない。

### 24.1 意味が異なる四つの完了を分離する

| 判定 | 必要な観測 | まだ保証しないこと |
| --- | --- | --- |
| **F: Frontend適合** | Sourceの実byte・occurrence・選択されたLanguage/Grammar/Plugin版、実Parserの構文結果と原文診断 | 参照意味/型の正しさ・Owner受理 |
| **M: Semantic適合** | 解決されたSubject/Role/Operator/Type・Nullability・Ref・grain/Scope/完全性・version closure、反例と拒否理由 | Go生成/実Provider/現権限/Commit |
| **G: Go生成適合** | 同じ意味契約から生成されたGo Source・Source map・Toolchainと実Build input・生成物/ロード元、独立Expectedの限定一致 | 他Targetの同値、実業務の保存副作用 |
| **E: Effect適合** | 現在のPrincipal/Host Authority・Provider Attempt・Operation identity・Commit状態・Receipt・独立Read（必要な時点/Scope） | 意味受理の自動決定・別Source/別Hostへの汎化 |

F/M/G/Eは**直交する証拠列**。Fが偽でもSource診断は利用でき、Mが未受理でも候補のpreviewや限定Fixture評価は可能。ただし、未受理の候補に**実本番effect実行資格を与えない**。Gを満たした純粋計算の値からEを推論しない。結果 `0` / 空配列 / 読取失敗 / 業務NotSatisfied / Technical Error / Commit unknown は異なる。これらの状態の新しい共通Enum/APIを先行作成することは要求しない。

### 24.2 小さな一改訂縦断：一つのScopeを固定する

**選ぶ題材（構文は仮の表記）**：`amount = quantity * unitPrice` と `total = lines |> sum(amount)`、`confirmable = lines.count() > 0 and lines.all(quantity > 0)`。Source上はGoの`struct/func/import`を要求しない。Canonical IDは既存ASCIIの`sales.order.*`を使い、表示名/ローカルUnicode Symbolは別Bindの試験に回す。まず一つのOrderと二つのLine、Order→LineのRelation、明示したData Contextと全件/一部のscopeを用意する（独立Expected・データモデル・関係・金額精度/丸めContractを**先に固定**する）。説明用Markdown表を勝手に実行DSLとして扱わない。

**最初の観測経路**：
```text
actual Source bytes + source occurrence + selected Language/Operator pins
  → existing ParseSelected / ParsePipelineSyntax / LowerPinnedPipeline (可能な既存経路)
  → existing Canonical/Role/Validator/typed Context (候補/受理状態を区別)
  → pure operator semantics + independent Expected
  → Go generator candidate + actual go build + single binary
  → actual consumed inputs + Got/diagnostics + source map
  → only when separately authorized: typed Intent → Provider/Host → receipt → independent read
```
途中に未提供の正式Go Generator/Binding/実Contextがあれば**`NOT RUN: missing concrete entry`**と止め、テスト用手書きGoを製品Generatorが存在する証明にしない。適切に型・版固定した合成インメモリDatasetで純粋Operatorを先に確かめ、その後#1119/#1163の実CLIと実Provider受入へ引き継ぐ。既存実Go CLIの`validate → operations → artifacts → fresh -out`と独立Readerの限定入口はmain統合[#2649](https://github.com/hat47x/tei-platform/pull/2649)・§19経路Aを再利用し、**`artifacts -out`のディスク書込みとread-only CLI資格を同一視しない**。

**必要な最小Evidence列**：`attempt ID | actual source+occurrence+revision | parser/language/role/operator (including transitive implementation pins) | interpreted meaning status | input row/field closure, scope, epoch | independent Expected & provenance | generated Go source/build inputs/loaded artifact set | Got & diagnostic code+original span | current authority | operation identity/Provider attempt | commit status | post-commit read | claim F/M/G/E, unknowns`。既存#1163/#2514の表・Receiptへの参照で足り、新公開Schema/共通Proof Engineは設けない。

### 24.3 同じ縦断に差す「正例＋反例」の最小マトリクス

| Case（本節の説明ラベル） | 入力差分 | Expected/不許可な推論 | 再利用するOwner |
| --- | --- | --- | --- |
| **VS-1 Source/Frontend** | 同じScopeの二表層、`select`や`|>`をGo式に直接渡す負例、Unicode alias、Source relocation | 表層ごとのparse可否を**実Parserで**測定。Semanticで同じことが成立する場合も元byte/span・構文Profileは別。Go都合の語彙変更を強制しない | #988/#2621、Draft #2639/#2745 |
| **VS-2 型/業務意味** | `quantity=3` `unitPrice=19.995`、丸め未宣言、明示NULL/未取得、同IDでDecimal→String | 正しい中間Decimalは59.985。丸め未定義は拒否、null/missing/not-fetchedの混同不可。同ID・同Binder結果でも意味の一致とはいえない | #2514 H5/TAO、#988、#2639 EG-09 |
| **VS-3 完全性/否定依存** | L={A,B}、R={A}、0行complete/partial、後からBが追加、Relation先/Owner変更、A→B→A | 部分行で`all`真、`not exists`、全件SUMを確定しない。既存反例は全称否定に使える場合がある。履歴の一致≠現在Qualification | #2582/#2622/#2600 IB-07/09/10 |
| **VS-4 実Go artifact** | Sourceが同byteでOperator/Generator版変更、出力の一member欠落、Source byteの検査対象と実Build対象が違う | 正しい再生成・診断/旧Artifact排除を要求。実Builder/Toolchain/Readerが消費した内容に帰属できなければ`unknown`。単一binary生成と全Output closureの区別 | #2595 EB-01/02、#2600/#2604 |
| **VS-5 効果と回復** | 計算上`confirmable=true`、権限変更、Provider ACK消失、Commit unknown、再試行 | 結果trueはIntent候補のみ。Commit不明ならOperation identityで回復/独立Readが必要。結果不明を更新なしとせず、勝手に再実行しない | #1129/#1174/#2514 W、ADR-0023 |

**必要な変種を分割**：正常な`all`は、空集合の場合に言語/Operatorが定義する空集合規則の影響を受ける。業務上「明細が1件以上」を同時に要求する`count()>0`を保持し、`all`単独の数学的真を確定受注可能と取り違えない。部分集合で`count()>0`の局所存在は示せても、完全な総額の確定にはならない。Unit/通貨・Decimal scale/roundingは明示契約とし、`float64`やGoゼロ値へ勝手にlowerしない。

### 24.4 採用判断・効率比較・再利用境界

**Frontend選択の実験（P0/P1）**は、利用者に見える**言語中立な軽量DSL／Go式／Markdown／chain**で同じ意図・Scope・型/Source問題を見せ、AIによるノイズ/認知順序/修正局所性比較を採点補助とする。独立Expectedで確定できる意味的な正誤は機械的Conformanceとして別採点。実際の人間被験者がいなければ人間可読性の定量実証とは呼ばない。構文方式A/B/Cの試験と、Backend D（Go生成）の試験は別。

**切替/停止**：Go ASTが予約語や型宣言を表層へ強要するならAを棄却。Hybrid Cの前処理・逆位置写像・Plugin版間の判定が独自Frontend Bより複雑で、局所診断やSource byte fidelityを損ねるならCを棄却。BになってもGo生成の標準Target方針は維持する。いずれかが不明参照・型/NULL/丸め・Source/Role Pin・独立Expected・実入力閉包・実効果境界を暗黙補完するなら**方式選抜を停止**してContractを修正し、意味受理へ進めない。

**最初の縦断を終えた後の第二領域**：受注で通った汎用化主張は、Membership/Occupancy（別ID・時点・所属/代理の意味）やWorkflowの異質Subject/Transition/Ownerへ**同じ最小Fixture概念を適用し反証**する。既存Draft[#2681](https://github.com/hat47x/tei-platform/pull/2681)のOrganizationモデルや#2386を再利用するが、そこで業務RoleからGrant/承認権限を推測しない。二領域未到達なら「業務受注Scopeに限る」にClaimを限定。新汎用状態表、Host、Registry、全Domain対応DSL Syntaxを先行採用しない。

### 24.5 計画の重複防止と次アクション

- **#988** が文法/Operator（新Parser/第二Resolverの採用可否）、**#2621** がAuthoring/元Source/診断の安定性、**#2514** が独立Oracleの品質とClaim範囲。Draft #2639は最小Go Language Plugin Referenceであり、追加テストの採否/CIはそのPR Owner。Draft #2745の国際化/安全AI Profileは同Branch文書へ転写しない。
- **#2595/#2600/#2604** が実Build input・ArtifactSet・公開Reader・現在資格。**#1129/#1174** はIntent/権限/Tx/Receiptと独立Read、**#2387/#2622** がSQL/Provider固有の意味・実現。**#1119/#1163** が縦断のEvidenceと既存Priorityを維持。元Source上の外部Referenceのfresh/stale/offlineはSource/Dependency担当の既存契約へ引き継ぎ、実ネットワーク取得はここでは行わない。
- **計画上の直近作業候補**：①既存Go Reference/現行CLIの正確な実行入口を記録、②VS-1〜3の意味/Source反例を実現できる最小FixtureとOracleの独立導出元を固定、③不足している正式Generator/Runtime Consumer/Provider Bindingを**missing concrete entry**として列挙、④実Go生成が可能になった範囲だけVS-4を試験、⑤effectは独立のAuthority/Receipt証拠が整ってからVS-5へ引き渡す。**未確認の工程を手書きMockで置き換えて製品合格にしない**。
- **2026-11-03時点のCI区分**：Draft #2667 head `5882ff67e22f1f37f8f6adccd4ccddfc6c480b4d`のPR健全性確認は照合時`queued`、Draft #2639 head `12c864ba83840a7b5750bc8da84af0e8534558e1`の健全性確認は照合時`in_progress`、Draft #2745 head `61f834771091260212d4264855c94a702efca4e2`は照合時`queued`（いずれも2026-11-03 04:28 UTC前後）。旧HEADのPASSを現HEADに移さない。**この§24自体のheadは後続コミットになるのでCIを別途照合する**。

**ADR**：既存§22の無番号 `DSL Frontend Selection and Source Fidelity` は正式採否を保留。既存ADR-0022/0023/0032/0033/0039/0043と#1119のGateを変更しない。今回の成果は**既存計画内の検査票と依存順序**であり、採用済み実装・形式・Runtime・Owner意味受理の宣言ではない。

## 25. DSLとGo Runtimeの検証費用――実行性能を守る配置と選択的な省略（2026-11-03追補）

> **設計・比較計画／実行時最適化の実装・計測はNOT RUN。** ユーザーの追加方針：Go単一バイナリの実行時レイテンシ・スループット・資源使用を重要な製品品質とし、過剰な検証は移動・統合・省略・無効化を検討する。ただし、検証の削除によって意味・現在Authority・保存効果の真偽が変わることを許す方針ではない。#565（closed）のPerformance/Resource Quality Envelope、#988/#2622/#1174/#2514、§22〜24に接続する。新しいQuality Gate/Runtime Host/Validation Engine/Registryは作らない。

### 25.1 「一度で足りる検査」と「実入力ごとの必要条件」を分ける

| 検査対象 | 基本の位置 | 無効化・移動の候補 | 省略してはならない根拠 |
| --- | --- | --- | --- |
| DSL Parse、Unicode/Lint、Source位置・説明書式 | 編集/保存/Build | 本番hot pathで再Parse、Source全照合、Formatterをしない。失敗時だけ原位置を復元 | Source/Language版が変われば再解析・診断。現在の実入力を静的Sourceで代用しない |
| Role/Operator/Plugin依存Pin、静的型、Relation定義、Eager循環 | Build/選択/ロード時 | 固定されたGoバイナリの毎件実行ではResolver/Role/型宣言の再検証を省略候補とする | Plugin/実装/意味版変更、動的Binding、未Pin依存は再資格化が必要 |
| 生成物の来歴、Artifact/Toolchain、独立Oracleの全Fixture | Build/CI/Artifact選択・ロード時 | 全件実行時のSource再hash、Conformanceの再実行を避ける | 実Build入力・実ロードbyteと検証対象の不一致、失効したArtifactを見逃さない |
| 外部データ型、presence、Nullability、Field/Row完全性、grain、時点 | Input/Provider境界、必要な評価時点 | 同じ不変条件を行ごとに繰返し確認する代わりに、検証済みの型付きRecord/Batch/Snapshotの適用範囲に限定して一回検査 | Raw入力、別Provider/別Epoch、新規Row、partial/withheld/missingは黙示的に正値扱いしない |
| Decimal精度・丸め・単位、業務Rule・状態遷移 | 専用計算経路・業務操作の必要点 | 既存ContractからGoの型付き専用演算にLowerし、反射や動的ディスパッチを避ける。証明済み値域なら重複のrange検証を検討 | 不明な丸め、overflow、時点で変化する規則や業務拒否をdebug限定にしない |
| Provider Read、Query Pushdown、動的/否定依存、参照先鮮度 | Provider/Read/再取得の境界 | 無関係な各Rowで再取得・再照合しない。Snapshotの有効範囲内でBatch/Cache | Row/Field completeness、scope、principal/RLS、time、Relationの変化を見失って全件/不存在を誤推論しない |
| 現在認可、Commit対象の版/重複防止、Effect Fence | 実Request/Commitと同じ既存Host/Provider境界 | read-only純粋演算には副作用Guardなし。writeは条件付き更新/Txで一体化し冗長な往復検査を削減 | Build時の認可を現在Grantと見なさない。Commit unknownを成功・失敗へ丸めず、無条件再試行しない |
| Trace/詳細説明/高頻度ログ | 開発・CI、運用で必要な失敗・監査境界 | 成功hot pathの全行Trace/文字列生成を遅延、観測量を限定 | 必須のOperation identity/拒否理由/監査/Receipt・失敗診断を失わない |

**省略可能なのは、同じ性質が同じ版・対象・Scope・時点・実装で既に保証され、その保証が後段まで保持される場合に限る。** 同じ返値、過去のPASS、同ID、Cache hitだけを省略根拠にしない。Go ASTをFrontendで使わない場合でも、同じ意味から同じ高速なGo実行経路を生成できるべきである。

### 25.2 実行形態とGuard削減の候補

1. **静的合成による最短経路**：モデル/Language/Operator/生成器の固定済み版から純粋Operationを型付きGo関数として生成できるか比較。Go単一バイナリの毎件実行でDSL再Parse/全Plugin Resolve/全Schema検証を要求しない。静的合成・動的ロード・外部Process方式の採否は#951/#988の方式Gateで保留する。
2. **Input BoundaryとBatch単位の検査**：型付き/NULL/Scope/完全性の保証済みRecordを内側で再検査しない。保証の作り方・消費元・失効条件は測定対象に記録。検証を省いた結果NULL、Missing、Not Fetched、Undisclosed、Partialが潰れるなら棄却。
3. **Providerでの最適化**：#2622のBatch、Pushdown、Stream、#2593の集約粒度を利用。filter→sort→limit と limit→filter は一般に交換できないので、速度だけで書換えを許さない。物理roundtripを減らす場合も実ProviderのRead scope/時点/権限を保持する。
4. **効果境界での最適化**：#1174のEffect Fenceはread-onlyには要求しない。Writeでは既存Host/Provider側の条件付きTx/原子的version検査が使える範囲で統合する。atomic保証のないSaaS/APIでは、削減した検証で現在Authority/Receiptの虚偽保証を生まない。
5. **診断と観測の低コスト化**：詳細Trace、Source map展開、diagnostic文面、全row logは必要なケースでのみ実体化する。エラー時に元Source span/安定コードを復元できる性質は維持。全製品共通のunsafeモードや任意の「認可OFF」フラグを導入しない。

### 25.3 ベンチマーク／Conformanceを同じ入力で実行

| ID | Baselineと最適化候補 | 独立Expected・停止条件 |
| --- | --- | --- |
| **PF-01 Decimal** | 動的Operator/型検査あり vs 静的Go専用計算、入力境界一回検査 | 3×19.995の厳密中間値59.985、丸め未指定拒否、正負tie、overflow、NULL、不対応modeの出力/拒否が同じ |
| **PF-02 集合** | 1件/大量明細、empty complete/partial、重複・1:N fanoutで行ごと vs Batch/Stream | 集計のgrain/order/bag/presence/完全性が同じ。partialを全件真にしない |
| **PF-03 Read** | 明示Scope/時点/Principalの複数Read vs Batch/Query Pushdown/Snapshot再利用 | 実Row/Field、否定/動的依存、RLSと時点の整合。速くても非可換な順序変更は不合格 |
| **PF-04 Effect** | 現在権限・Target版・Commit/Receiptの通常経路 vs 条件付き更新で往復削減 | 許可・拒否・失効・Commit unknown・リトライで同一業務効果（試験範囲）。read-onlyには副作用Guardを載せない |
| **PF-05 診断** | 詳細Trace/全件診断生成 vs 失敗時の遅延展開/限定Audit | 正常時低コスト、失敗時にSubject/診断ID/元位置/必要なReceiptを失わない |

**性能指標**：p50/p95 latency、throughput ops/s、ns/opまたはms/op、B/op・allocs/op、CPU/GC、Provider roundtrip、cold start、検査実行回数。Source/Accepted Meaning/Operator/Generator/Go Toolchain、実Build/ロードbyte、データ量、Scope/epoch、OS/CPU/並列度、warm/cold、測定分散を固定し、実Goの純計算と外部Provider I/Oを別々に計測する。#2600のBuild/編集待ち時間と本節のRuntime latencyを混同しない。

**採用規則**：まず同じ独立Expected、同じ拒否/unknown、同じ必要Effect証拠に合格し、その後に性能差を判定。性能目標・許容劣化はScenario/Targetごとに事前定義し、一律の数値を先行採用しない。未測定の高速化や実行資格を主張しない。既知の負例（Go Source/Build差、Decimal→string、partial=empty、A→B→A、Grant失効、Commit不明、Source/Artifact混在）のいずれかを見逃せば当該Guard削減は棄却。

### 25.4 実証順序とOwner／未決定事項

- **B0 検査配置を分類**：#988の純粋Operator、#2622のProvider境界、#1174の副作用境界について、compile-time／load-time／input-boundary／effect-boundary／diagnostic-onlyのどこで検証するかを既存受入票へ追記。仕様外・未知の保証は「省略可」にしない。
- **B1 PF-01/02の限定Baseline/Fast path**：#988/PR #2639の既存Go Referenceに接続し、純粋計算の速度と意味を比較。手書きGoの比較結果を正式TEI Generatorの実装済証拠にしない。製品入口不明ならNOT RUN: missing concrete entry。
- **B2 PF-03/04のProvider/Effect実測**：#2622/#1174/#1119の既存Ownerに引継ぎ。部分集合・時点・認可・Commit不明の意味を損なう場合、より高速な案でも棄却する。
- **B3 性能採用・回帰**：#565のQuality Envelope（Subject/Revision/Runtime Target/Scenario/Measure/受入根拠/Evidence/状態）に結び、#2514の独立Oracle、#2595/#2600の実Source/Build/Consumer資格、#1163の一改訂Evidenceを参照。Guardを省略して誤判定したら対象最適化を戻し、古いReceiptと現在資格を分ける。開発/診断時の追加検証を製品Runtimeの常時コストにしない。

**正式判断は未了**：どのGuardを省略・無効化してもよいかは、実Target/Providerが持つ保証とConformance・性能測定を見てOwnerが決める。静的/動的Plugin方式、Backend Generatorの正式API、共通Benchmark threshold、正式DSL構文、Product Runtimeの最適化実装を決定しない。本節はドキュメントのみであり、新しいGo/CLI/SQL/Provider/Hostベンチマーク、Go生成、CI、Owner意味受理はNOT RUN。

## 26. 広域データ品質検査をモデルから派生させる――夜間・Replica・Lean補助Target（2026-11-03追補）

> **設計提案／NOT RUN／正式Job API・Scheduler・検証成果物Schema・Lean Target・運用SLAは未採用。** §25の実行時性能重視を発展させる。業務データの定義域・行内整合・行間/集合整合・時点・数学的法則を、日常アプリとは**独立に実行可能な派生検証成果物**として生成し、夜間やRead Replica上で広く監査できるようにする。検証規則を別チームが第二正本として再入力するのではなく、**同じAccepted Meaning・Data Rule・Operator/Facet・依存契約**から用途別にLowerする。「独立」とは**実行場所・時刻・資源・Receipt**の独立性であり、業務意味の重複管理ではない。Independent Expected/Oracleは候補実装と別根拠を維持する（#2514）。

### 26.1 生成の二系統＋Lean（新共通Runtimeは前提としない）

```text
authoritative Source → existing Canonical / Accepted Meaning
   └─ typed Data Rules / Relation / Quantity / Lifecycle / Operator pins
        ├─ Online Go app / typed Provider operations
        │    └─ low-cost mandatory guards at input / command / effect boundary
        ├─ Derived Data Quality CheckSet (候補)
        │    ├─ standalone read-only Go binary / SQL query suite / existing Provider adapter
        │    └─ scheduled externally or manually on primary snapshot, replica, isolated copy
        └─ optional Lean 4 proof obligations / theorem-check artifacts
               └─ law/algorithm correctness under explicit assumptions
```

「夜間Job」「Replica」「Lean」は**実現/実行Profileの違い**。同じRule/Subject/Scopeの由来、Source/Meaning/Operator/Generator版は相互に追えるが、ジョブの実行時刻・Reader/Provider・Snapshot・結果資格は別。派生CheckSetは正本でも新しいValidation/Proof Registryでもない。cron/CI/既存外部Schedulerで実行できることを先に比較し、TEI専用Host/常駐Schedulerは作らない。Read replicaでは一般に参照Queryだけが可能であり、監査結果の永続化・通知・是正Intentは**別の許可された保存先/運用系**へ出力する。

### 26.2 「検証の後回し」と「制約の不履行」を分ける

| 既存Meaningからの品質主張 | 基本配置 | 夜間/Replicaへ回せる条件・限界 |
| --- | --- | --- |
| 値の型/値域、NULL許容、文字長、数量・通貨単位 | Source/Build/DB constraints＋入力境界 | 過去行の全件スキャンや版移行後の再監査はoffline可能。危険な生入力の拒否や必須DB制約を単に後回しにしない |
| 行内の合計・整合、状態と必須項目の整合 | Cheapな同期Rule/必要なDB制約＋offline再照合 | 発見遅延が許容され、是正方針がある追加監査は移動可能。Commitの正当性が依存する条件は同期で維持 |
| PK/FK/業務Relation、複合/期間キー、重複、孤児行 | DBで提供される制約＋必要な更新時契約 | 広域点検や既存汚染発見を別Job化。物理FK/SortKeyを業務Relation/履歴意味へ自動昇格しない |
| 注文合計＝明細合計、在庫/残高/時間窓、集計grain・重複 | 定義された更新契約＋周期的整合スキャン | Snapshotの全対象・Field completeness、時間・重複・遅着Eventの条件が揃う範囲だけ判定。部分結果のゼロ件は「正常」としない |
| 状態遷移履歴/取消と履歴の一致、外部システム間照合 | 必要なcommand/Tx/receipt境界＋async再照合 | 実副作用を発生させるRepairは検証Jobから自動実行しない。独立ReadとOwner判断の後に既存CommandへIntentを渡す |
| Lean等による演算則・不変条件・矛盾可能性の証明 | Build/CI/夜間の任意の形式検証Target | 「仮定下の命題の証明」は実データ全件・Provider鮮度・翻訳忠実性・Owner業務意味受理・Go実装同等性を証明しない。反例未発見/timeoutを証明成功としない |

**延期可能性はRuleごと・用途ごと・業務Ownerごとに判定する。** 夜間検査結果を処理時の認可・一意性・金額の必須丸め・確定可否に無条件で代用しない。onlineは性能と現時点整合、offlineは全体的品質と時間をかけた再照合。両者を「同じ保証」と呼ばず、許容発見遅延・重大度・是正期限を明示した場合に限り一部の高価な検査をoffline専用へ移す。

### 26.3 Job/Checkの最小引継ぎ票（既存Evidence/Artifactへ参照するだけ）

| 欄 | 必ず保持する理由 |
| --- | --- |
| Rule Subject / Accepted Meaning・Source occurrence / Rule/Operator/Lean theory/Generator version | どの受理された意味から導出した検査か、何を未受理としたかを説明 |
| Check kind (domain/intra-row/inter-row/aggregate/temporal/proof), severity, online/offline placement | 実行時に残す条件と夜間検査へ移す条件を分ける |
| Data Context / Relation path / grain / bag semantics / row+field completeness / scope / predicate / period | 完全性のない全件「異常なし」を防止 |
| Data Provider/DB/dataset/replica identity、snapshot ID・LSN/replay watermark、observed at/checked through | PRIMARY現在値とReplica上の過去時点の値を混同しない。複数Providerは同一watermarkを勝手に仮定しない |
| Raw/typed input interpretation、NULL/missing/withheld/partial、operator/rounding semantics | 不明・非開示を合格0件へ変換しない |
| Independent Expected/Oracle provenance、known mutants、case result | 同じGenerator/Rule interpreterによる自己追認を避ける |
| Run attempt ID、actual executable/SQL bytes + pin、rows scanned、findings、coverage、errors、time/cost | 実施した範囲と、未実施/部分失敗/timeoutを区別 |
| Observation/diagnostic stable ID、source/subject、historical result、current qualification、remediation intent | 結果の説明、再監査、修正提案と実効果を区別。低権限へ生データを漏らさない |

データ品質の判定は少なくとも **`pass-in-scope` / `violations-found` / `incomplete` / `unavailable` / `invalid-contract` / `not-run`** を比較票で区別する候補。これは新Core enumや正式APIではない。たとえ`pass-in-scope`でも「指定Snapshotの対象範囲に限定して違反を観測しなかった」という主張であり、現在のPRIMARY全件保証ではない。歴史的PASSと現在の資格を分離し、定義/Provider/Source/Policy改版やReplica遅延で現在適用を再判定する。

**Replicaの停止条件**：レプリケーション遅延、対象WAL/LSNの未適用、Snapshot未確定、接続先の昇格/経路変更、長時間Queryとのreplay競合、別DB/外部Providerの異時点読取、欠落した対象集合を`0 violations`として合格にしない。多くのReplicaは更新不可なので、Check結果の保存先とDBの読み取り側を分離する。Primaryへの更新は別Authority・既存Command/Receipt境界へ渡す。

### 26.4 最小反証Fixture（新Engine/新Case大量追加なし）

| ラベル | 同じ業務Ruleから導出する観測 | 独立Expected／誤ってはならない結論 |
| --- | --- | --- |
| **DQ-01 行内/定義域** | quantity=3, unitPrice=19.995、明細金額59.985、丸め定義あり/未定義、NULL/missing/0 | 同じ型付き意味とDecimal精度でonline+offline計算。規則未定義や未取得を正常値に補完しない |
| **DQ-02 行間/集計** | Order O1に明細A=100、B=100、配送3件fanout | 期待合計200。JOIN後600やDISTINCT amountで100を採用しない（#2593既存MC-01） |
| **DQ-03 Replicaの鮮度** | Primaryで行Bをcommit、ReplicaはB反映前のLSNでスキャン、その後replay | 旧LSNの違反ゼロは当該過去Snapshotに限定。Primary最新の「全件合格」と報告しない |
| **DQ-04 部分・異時点** | Rowset L={A,B}、取得R={A}、Field欠損、異Provider異epoch | partial/unknown。R内違反は範囲根拠つきで発見可能だが、違反ゼロは全体正常性を示さない |
| **DQ-05 契約・意味改版** | ID/値同じでもRule版、Facet演算、Provider/Source改版、A→B→A | 古いCheck binary・結果の現在資格を再利用しない。影響閉包が証明できる場合だけ選択的再検証 |
| **DQ-06 Lean/実データ境界** | 命題「入力集合Lの全行q>0なら`all(L,q>0)=true`」の形式的根拠、実ProviderはR⊂Lだけ返す | 抽象法則の証明はL=Rを保証しない。Lean proof success ≠ Data pass ≠ generated Go correctness ≠ Owner acceptance |

**追加PoCの判断順**：
1. **D0（read-only設計）**：#2622のData Rule/Context、#988のOperator契約、#2544のLaw/Facetを取り出し、オンラインに必要な検査と周期検査候補を一Ruleずつ比較。どのRuleが単に遅延可能かは業務Owner判断を残す。
2. **D1（実行場所の分離）**：受注/明細の既存Fixtureから**同じRule/Source/意味Pin**に由来するオンラインGo純粋演算と、read-only SQL/Goスキャンの候補を派生。独立Expected（#2514）・Provider readの実対象を照合。最初は人工Snapshot/一時DB、既存外部Schedulerでよい。
3. **D2（Snapshot/Replica）**：一次DBのCommit水準とReplicaのreplay watermark・scan snapshotを記録して DQ-03/04 を反証。Replicaの0件や長時間ReadをPRIMARYの現在受理へ昇格しない。Check実行がReplicaのreplay lag/CPU/I/Oへ与える影響も測る。
4. **D3（Lean任意Target）**：#2544のLaw claimと#2514の独立Expectedを参照し、合成した一つの命題/仮定/Lean proof artifactを作る。実Rule/Go/SQLがその命題を忠実に実装するかは**別Fixture/実Target比較**。Lean EngineをTEI Runtime必須依存にしない。
5. **D4（運用価値）**：findingsを型/Subject/Source/Rule/Owner/再発/未確認範囲で集約し、候補原因・修正案を提示。**修正は意図提案まで**であり自動DB更新しない。検出率/false positive/scan cost/Replica負荷/実行遅延/修正までの時間/人手定義量と既存の手書き監査を比較。未検証の「大手品質チーム相当の保証」を製品達成と主張しない。

### 26.5 既存Owner/ADR境界・変更状態

- **#2622** がData Rule/Function Data Context/Providerと多対象Queryの意味契約、**#1126/#1129** がData定義・キー/Relation、更新時/Tx/状態・副作用、**#2593/#2582/#1708** がaggregate/grain・動的依存・Snapshot境界。
- **#988/#1063** が同じOperator/意味から用途別の検証用成果物へLowerする接続、**#2544** がLean等で検証し得るLaw/Facetと仮定の意味、**#2514** が独立Expected/Claim相対の判定、**#2600/#2595** が生成/実使用Byteと依存失効、**#1174/#1119/#1163** がAuthority/Effect・正式受入Evidence。
- 既存ADR-0022（AI提案/意味受理）、ADR-0023（Command後独立観測）、ADR-0029（Application validation選択）、ADR-0039（Query Missing/NULL、提案中）、ADR-0043（TEI独自Hostなし）と衝突させない。**新ADR候補**は「同一MeaningからOnline GuardとOffline Quality Checkを導出し、Snapshot相対の検査結果と現在資格を分離する」だが、正式の文書・番号・採用判断は実証後とする。
- 本節の設計は**DSL文法、Lean Target、検証Job命令/CLI、Scheduler、品質Dashboard、Result Store/Schema、Runtime Guard省略の正式契約**を採用・実装しない。**実Provider/Replica/Lean/Go生成/Go試験、CI、Owner受理はNOT RUN**。

## 27. 広域品質検査の運用契約――増分監査・EKI委譲・是正までの独立証拠（2026-11-03追補）

> **状態：設計・縦断実証案。NOT RUN／未採用。** §25のオンライン性能、§26の派生CheckSet・Replica・Leanを、**継続運用で「いつ／何を／どこまで検査できたか」を失わず、少人数で異常を調査・是正できる構成**へ接続する。既存#1130（UI非依存Batch）、#2622（Data Rule/Provider）、#2544（Lean/Facet）、#2514（独立Oracle）、#2582/#2600（依存閉包）、#1174（Effect Fence）、EKI Fabric #57（Task/Attempt/Lease/Result verification）と責務を重複させない。

### 27.1 実行基盤ではなく「検査意味と実行実体」を分離する

同じOwner受理済みData Rule／Model／Operator契約から **Online Guard**、**read-only Data Quality CheckSet**、**任意のLean proof obligation** を用途別に派生する。原定義が一つであることは、各Target実装が同じ誤りを共有し得ることも意味するため、独立Expected・既知Mutant・異経路比較を#2514へ残す。

| 所有者・実行レーン | 所有する契約 | 所有しない結論 |
| --- | --- | --- |
| TEI Meaning/Generator | RuleのSubject・Owner受理版・型/NULL/grain/Relation・依存、Quality Checkのread scope/閉包条件・診断、派生Go/SQL/Lean候補 | 実際にJobが起動したこと、実Replicaの鮮度、実Providerの開示・Commit |
| 既存Scheduler・cron・Job Controller（EKIは**選択肢**） | Trigger、予算、Task/Attempt/Lease、Worker配置、retry/reclaim、物理実行のResultClaim/検証・配送 | AcceptedResultだけから「品質違反ゼロ」「業務Rule適合」「Source Meaning受理」を保証すること |
| 実Provider/DB/既存Host | 実Read権限、対象表・Field・Scope/Principal、transaction snapshot/LSN、read-only制約、現在のResource/Query制約 | 異時点の二DB/Replicaを黙って同一Snapshotにすること |
| TEI Quality Report Consumer | 実Query入力/結果の束縛、対象期間とcoverage、異常/不明/未検査、診断ID、版・時点付き結果の説明、再確認要求 | 古いReplica findingでPrimaryを直接修正、実行成功を業務品質合格へ昇格 |

**TEI固有Hostは作らない**。品質検査は明示されたInput/Output契約を持つ独立Goバイナリやread-only SQL等として生成し、まず既存OS Scheduler/CI/手動CLIで実行可能かを比較。TEIが非常駐でも、業務上の状態/Flow/依存/適用条件はTEI定義が正本。EKIを使う場合は**実行管理・CPU/GPU/メモリ・ノード/Lease・取消・再試行**を部分委譲し、TEIの業務状態遷移・Owner判断をEKIのTask状態へ移管しない。

EKI Fabric [#57](https://github.com/hat47x/eki-fabric/issues/57)はTask/Attempt/Lease、Lease失効後のlate Result拒否、ResultClaimと独立VerificationResult、AcceptedResultを区別する**既存の実証Owner**。しかし現時点でTEI品質CheckSetをEKIへ統合・実運用した根拠はない。**EKI AcceptedResultは輸送/実行成果の受理であり、TEI quality-passではない**。Workerが生成した診断値だけで独立な正しさを推論しない。EKI連携を必須依存・巨大Job Manager・新TEI常駐サービスとして前提化しない。

### 27.2 Quality ClaimはSnapshot相対の二段階判定にする

**実行Receipt**（Run/Attemptが実際に何を消費し、完了したか）と**品質Claim**（特定Data Scopeに異常が存在する／観測されなかったか）を別にする。Quality Claimの最小束縛は§26の表に追記するだけで足り、第二Proof Schemaは作らない。

1. **Source/Rule側**：Accepted Meaning/Rule/Operator/Generator/実Go/SQL byte・Target版、適用条件、必要なData Context・Independent Expected。
2. **Read側**：実Provider endpoint/Replica身份、Principal/開示条件、Query byte、snapshot/transaction/LSNまたは整合版、replica replay watermark、対象Period/Row/Field/Relation closure、処理件数と未観測理由。
3. **Execution側**：Task/Attempt/Lease（該当時）、開始/終了時刻、実Artifact、CPU/IO/Query budget、timeout/cancel/retry、例外/row count/violations、結果の完全性。
4. **Consumer側**：得られた `violations-found` / `pass-in-scope` / `incomplete` / `unavailable` / `not-run` のClaimと適用時点、旧Claimの現在資格、必要なOwner再受理・Provider再Read。

**二段階**：①ReplicaのSnapshot S上で「違反観測／対象内違反未観測」を記録、②現在のPrimaryへの適用・修正が必要なら、当該キー・Rule版・最新レコード/関係/状態を**現在の認可下で再検証**。Replicaの旧違反がPrimaryで是正済みの可能性、旧Replicaの0違反後にPrimaryで新違反が発生した可能性の両方を扱う。ReplicaのLSN確認は、当該DBの必要変更が適用されたことを示す条件にはなり得ても、外部API/異DBとの共同Snapshotを自動保証しない。

**実時間の資格はRuleごと**：危険な入力・Commitに必要な業務条件はOnlineで保持し、歴史的/広域/高価な監査だけ非同期化を検討。「Nightly pass」を翌日昼の新しいCommandの認可や業務適合へ流用しない。Quality結果の保存・通知・アクセス制御は既存運用/Providerの管理下に置き、Replicaへ書き込まない。観測した生データのAgent/AIへの開示には、現在のDisclosure Scopeを適用。

### 27.3 全件Scanと選択的再Scanの採用条件

夜間検査でも全件毎回Scanは高コスト。静的依存だけでなく、**候補集合の増減・否定依存・動的Relation・変更履歴・Schema/Rule版**を追跡できる場合に限り選択的再監査を検討する。

| 変更 | 再評価範囲の例 | 正しい「省略」の前提 |
| --- | --- | --- |
| 明細行Amount/Quantity変更 | 当該Lineと関連Orderの合計/制約 | 変更ログ/対象Order/Relationの完全な閉包がある |
| Line挿入・削除 | 当該OrderのCOUNT/SUM/`not exists`／全称Rule | negative dependencyを記録。削除を更新差分から落とさない |
| Customer/Org membershipのRelation先変更 | 変更前・変更後の両参照先に波及するClaim | 動的参照先の旧/new identityと時点を確認 |
| Rule/Operator/Lean前提/Generator変更 | 当該Ruleから導出されたすべての対象Claimを再資格化 | 局所再実行で足りることを依存閉包から証明できる |
| Change feed欠落、Replica lag不明、未観測期間あり | 範囲未確定として`incomplete`、必要ならFull Scan | 最終値がA→B→Aでも中間履歴無変更と推論しない |
| Provider/Snapshotまたぎ・テーブル再構成 | 各Scope/epochへ区切って判定またはUnknown | 異なるwatermark/Read cohortを一つの全件に合算しない |

**増分の開始条件**：前回検証済みのFull Scanの対象集合・snapshot・Rule版が存在し、変更ログにGapがなく、対象Ruleの依存（挿入/削除/否定・時点）をすべて列挙できること。そうでなければ全件Scanへ戻すかUnknown。Previous passにDeltaだけを足して完全性が自動継承されたと主張しない。#2600の「誤再利用ゼロ／過剰失効検出」をData Qualityにも適用。ただし**Builderの増分再生成と実データの増分Scanは別の問題**であり、同じCache命中率で評価しない。

### 27.4 「異常検出→確認→是正→再観測」を独立したEvidenceに

```text
derived read-only CheckSet → observed finding (source/snapshot scoped)
    → triage (kind/severity/owner + suspected cause; AI suggestions optional)
    → current Primary/Provider revalidation (same affected subject + Rule)
    → authorized, typed Repair Intent via existing Command/Host/Provider
    → Commit known/unknown + operation identity + independent Read
    → new Quality Claim (new snapshot), old finding remains historical
```

CheckSetは**原則read-only**。RepairのSQL/CommandはFindingの副作用として即実行しない。修正候補の根拠・作用範囲・更新前後条件と、Ownerの受理・Hostの現在Authorityを検査する。Commit unknown時は二重実行禁止と再照合を優先。是正後の0違反は再Scanした新しいSnapshot範囲のClaimであり、旧不整合が初めから存在しなかったことにはしない。AIは異常分類・原因候補・Patch/修正Intentの提案を支援できるが、Context内の説明を実Authorityへ昇格しない。

### 27.5 最小受入Fixtureと優先順位（#26の再実装を避ける）

| ラベル | 反証する変種・異常 | 独立Expected / 結論範囲 |
| --- | --- | --- |
| **DQX-01** Online vs Offline | 同じ量/明細合計RuleをGo online／SQL read-only offlineへLower。配送JOIN fanoutあり | 違反の有無・集計grainが一致。異なるQuery生成器が同じ誤りを共有し得るため、handwritten Oracle/Mutation対照で検出 |
| **DQX-02** Replicaの古いPASS | S0では0違反、Primaryに新違反B、Replica replayはS0 | 「S0の範囲で異常未観測」だけ。現在Primary PASS・Owner受理を主張しない |
| **DQX-03** 旧違反の是正 | S0で違反A、Primary S1では修正済み。古いFindingの自動Repairを試行 | Primary再検証でRepairを拒否/不要にする。監査履歴は消さない |
| **DQX-04** 増分の見落とし | 明細B削除→`not exists`が成立、CDC gap、A→B→A | Delta-onlyのfalse passを拒否、再Scan/Unknownを選択。FullとIncrementalでClaim一致 |
| **DQX-05** EKI委譲/重複 | Worker A lease失効、Worker B再試行、Aから遅延Result、同一Taskの重複配送 | EKIのResult資格とTEIの品質判定を別々に結ぶ。遅延Resultで品質PASSを上書きしない。**EKI実連携はNOT RUN** |
| **DQX-06** Lean/実装/入力 | LeanでLawが仮定下で証明、SQL/Goに誤丸め/バグ、Providerは部分集合 | Proof PASS・実装不一致・データUnknownを別Claimとして同時報告。形式証明からGo/SQL同値を推論しない |

**実証順（新しいGateではない）**：
- **Q0**：#2622の既存受注/明細Ruleと#2514の独立Expectedを1件固定し、online guard／offline scanの配置理由・発見遅延・Ownerを記録。
- **Q1**：実DBの固定Snapshotを使うread-only全件Check（受注合計と孤児/重複のうち1つ）を、同Rule/Operator版で生成可能か検査。正式Generator未存在なら`NOT RUN: missing concrete entry`、手書きSQLは比較Oracle/候補としてのみ扱う。
- **Q2**：同じ検査にReplica lag・部分取得・異時点・失敗途中を与え、ReceiptのScope/epoch/coverageを検証。
- **Q3**：前回Full Scan＋変更の閉包が成立する場合だけIncrementalとFullを比較し、false reuse/再Scan負担/p50,p95/Replica IOを計測。
- **Q4**：再検証付きRepair Intent、独立Read、必要ならEKIのexecution-only Task委譲をそれぞれ独立に確認。Leanは任意の補助Targetとして後続。EKI/Leanの未完成をQ1 read-only検証の必須ブロッカーにはしない。

**責務引継ぎ**：#1130がUI非依存Batchのtyped input/output、trigger/取消/進捗/結果Artifactを所有。#2622がData Rule/Contextと多Provider Read、#2544がLean Law/Facet、#2514が独立ExpectedとClaim Coverage、#2582/#2600が依存/増分の適用条件、#1174/#1129がRepair/Authority/Commit、#1163が一改訂Evidence。EKI側のJob Controllerは既存EKI Fabric #57等のOwnerであり、このTEI計画からEKIの実装・ADR・Scheduler選定を決定しない。既存ADR-0023（Command後の独立Read）、ADR-0029（Application Validation選択）、ADR-0043（独自TEI Hostなし）を維持。

**停止・非採用**：Replicaの旧Resultを現在の全件PASSと報告、EKI AcceptedResultをTEI Quality Passと混同、Change-feed Gapを見逃した局所Scan、省略後のOnline必須制約漏れ、未確認のRepair自動適用、Lean ProofからProvider実データ合格を推論した場合は採用停止。Job/Proof/Repairの製品API・Scheduler・新Registry/Host/Quality Dashboardの採用、実Go/SQL/Replica/EKI/Lean実行、CI、Ownerの意味/運用受入は**未決・NOT RUN**。



### 27.6 継続品質保証のCoverageと期限――「正常0件」では証明できない部分（2026-11-03、他レーン成熟差分）

> **計画・受入票のみ／未実装・NOT RUN・Owner運用受理なし。** §26 D0〜D4／§27 Q0〜Q4の「品質Checkを派生生成して読取専用で実行」と、§35〜36のEKI資源・Deadline/Recovery最適化を**別の正本を増やさずに接続**する。欠けていたのは**品質Claimの分母・検証義務の期限・例外扱いと、データ時点の帰属**。新Registry、Dashboard Engine、TEI常駐Host、共通Scheduler、正式DSL/Job APIは作らない。

**品質検証義務（Quality Obligation、比較用語であり新Core kindではない）**：同じAccepted Meaning/Data Rule/Operator pinに対し、業務Ownerが、(a)何を・どの期間/粒度/対象Scopeで検査するか、(b)必須Online Guardか延期可能な広域監査か、(c)許容検出遅延と完了期限、(d)判定に必要なProvider/Read一貫性・実証Oracle、(e)異常時の担当と是正手順を選ぶ。単なるDDL型、項目名、JOIN、同名Facetだけで「業務違反」やSeverityを確定しない。オンラインの必須拒否・現在認可・Commit条件は夜間PASSで代替できない（§25、#1129/#1174）。

| 報告する対象 | 必要な独立証拠 | やってはいけない正常化 |
| --- | --- | --- |
| **Applicable Rule/Scope** | Owner受理済みRule版、適用期間、対象母集団、Relation path、Row/Fieldの完全性、Data Context | 未受理Rule・物理Column全件を業務上の検証義務に自動算入 |
| **実行と品質の別Outcome** | Scheduler/EKIのTask/Attempt/Lease/実Artifact、実Read/QueryとSnapshot、独立Expected、Data Quality Claim | Worker/EKI `COMPLETED`を全件正常、Go/SQL一致だけを独立Oracleとする |
| **Verified Coverage** | 検査済みRule×母集団×時間範囲×Reader/Provider×Read Cohortと、未検査箇所の閉包 | 走査件数・出力0件・シャード数をそのまま「100%完了」に変換 |
| **Coverage Debt** | 予定された対象に対する未実施、stale、partial、権限不足、unsupported、timeout、期限超過、CDC gap | 検査失敗・知らない領域を違反0件に混ぜる。分母が不明なら比率もunknown |
| **Finding Disposition** | 実違反/原因調査中/誤検知/Owner承認例外/修正確認済み、対象IDと版/期限、再発・現在性 | 承認例外をRuleから削除する、例外期限切れをPASSとして扱う、AI案をRepairの許可とする |
| **運用費用・遅延** | 検査待ち、開始→実Read→結果検証→Consumer到達、scan CPU/IO/bytes、Replica replay lag、Provider費用 | Job終了のみで期限内品質確認とする。Primary/Replicaの負荷とデータ転送料を無視 |

**複数の時点を保持**：業務有効時点（validAt）、記録/訂正時点（knownAt/recordedAt）、Provider Snapshot/Replay watermark、実Check時点、期限（§36 `latestAcceptableAt`はJobの一般契約。品質義務が求める「確認済みData時点」は別）。組織所属の10月1日発効を11月3日に遡及訂正すると、10月15日as-ofの評価結果は新しい記録知識で変わるが、過去のジョブReceiptは改ざんしない。「昨日検査済みだった」を後から捏造しない。期間がない普通の受注行に汎用bitemporal保存を強制しない（#2386/#1129/#2622、§34）。

**Quality Claimは発見された違反、指定Scope内での違反未観測、検査未了、現在適用資格を別票にする**。特にRLSやProvider側の非開示によりA部門だけ読めたとき、A部門の`pass-in-scope`は全社のPASSを意味しない。複数Shardの結果も、互いの範囲が重複/欠落なく、必要なRead Snapshot一貫性が示せる場合だけ結合する。Snapshot確定のないOFFSETページ分割では、更新/移動による欠落を見逃し得る。Clientへ開示した列の少なさは実Providerの取得範囲の狭さを証明しない。Declared Read・Actual Acquisition・Disclosed Outputを分離する（#2622/#2516）。

**Owner承認例外は違反消去でなくDisposition**：承認対象Rule/Subject・有効期間・承認者/理由・期限を記録。期限切れ/Rule改版/対象の変化では再評価。異常が過去Snapshotでは実在してもPrimaryでは既に修正済みなら、現在Readで是正必要性を判断し、過去のFindingは履歴として残す。修正は既存Host/Providerの型付きIntentと現在Authority/Commit/独立Readへ渡し、CheckSetがReplicaから直接更新しない。

**締切と再検査の判断**：Rule別の最大検出遅延・重要度をTEI Ownerが決め、EKI/既存SchedulerのPolicy/実測待ち・データ転送・実行・結果検証・回復余力を§36の別契約で評価。品質Deadlineに届かない場合はoverdue/unknownとし、既に許可済みの別時間帯・別Replica・部分Scan（partialと明示）やOwner通知へ進む。自動クラウド課金・必須Online Guardの無効化・別Tenantの無断取得をしない。資源効率のため全件Scanを減らす場合も、初期Full Scan＋全変更・挿入/削除・否定依存・Relation旧/new・時点の完全な閉包が必要。CDC Gap/A→B→Aではfull再検査またはunknown（#2582/#2600）。

**最小反証票：既存DQ-01〜06/DQX-01〜06に追加する差分（新Runnerなし）**

| ID | 入力反例 | Expectedと停止条件 |
| --- | --- | --- |
| **DQA-01** | RLSでA部門は全件読めて異常0、未取得B部門に違反 | A部門のScope限定PASS＋全社Coverage Debt。全社緑表示しない |
| **DQA-02** | Membershipが10/1に遡及有効、11/3に訂正。10/15 as-ofを旧/新記録知識で評価 | valid/recorded/snapshot時点を別にする。旧Run Receiptは修正しない |
| **DQA-03** | 違反AをOwnerが期限付き許容、期限後もデータは未修正 | Findingの違反は残る。例外Disposition失効・再確認。Quality PASSへの変換なし |
| **DQA-04** | 読取Shard1=S0/Shard2=S1の間に行移動、各Shardは異常0 | 統合Snapshot/対象閉包がない限り全件PASS不可。再取得/unknown |
| **DQA-05** | 夜間CheckがRun完了したが検査Readerが部分取得・期限超過。EKI結果Accepted | 実行受理／品質Claim／完了期限を別判定し、`overdue/incomplete` |
| **DQA-06** | Rule変更、CDC gap、旧違反をPrimaryで修正済み、Retryで古いResultが遅着 | 今回版のCoverage再資格・修正前の現在Read・Attempt ID照合。旧PASSで上書きしない |

**受入・適用順**：最初は受注の合計整合と組織Membershipの期間整合の**二つの異質なRule**に限定し、(1)OwnerがOnline必須/Deferred・検出期限を選ぶ、(2)固定Source/Meaning/OracleからGoオンライン/SQL等Check候補を派生する、(3)実ReaderのScope/Snapshotを記録する、(4)DQAの異常/unknown/期限超過/承認例外をread-onlyなレポートへ出す、(5)外部Scheduler/任意EKIへの委譲時に仕事の受理と品質Claimの違いを実測する。#2514の独立Expected/Mutation、#565のPerformance Quality Envelope、#1163の一改訂Evidenceを再利用する。**独立Oracle/実Provider/Lean/Replica/EKIのE2Eはこの追補でNOT RUN。** 検査の費用対効果は、検出漏れ・時間内検出・是正までの手間・人が個別に保守するルール数・CPU/IO/転送・Replica負荷を同時に測る。

**外部出力候補**：OpenLineageのJob/Run/Dataset／Input `dataQualityMetrics`/`dataQualityAssertions` をConsumer向けread-only Projectionで比較可能（https://openlineage.io/docs/spec/object-model/）。ただし`Run COMPLETE`≠`Quality PASS`、外部形式の採用/Connector実装は未決。PostgreSQL Hot Standbyのread-only scanはreplay遅延、長時間Query競合/取消があり、適合範囲と負荷は実環境で検証する（https://www.postgresql.org/docs/18/hot-standby.html）。

**責務**：#2622がRule/Provider Read、#2514がOracle/Claim、#1130が非画面Batch、#2386/#1129が業務時点・状態、#2582/#2600が増分閉包、#1174がRepair/Commit、#2544がLean Law、§35〜36/EKI #19/#57が実行資源・期限/Recovery。旧§27 Q0〜Q4と既存Owner/Gateは維持。新Core Kind・Quality Registry・中央例外テーブル・自動Repair/Policy Engine・TEI Host・正式ADR/DSLを決定しない。


### 27.7 広域品質監査のRead権限・実取得・外部開示――全件・増分・標本の保証を分ける（2026-11-03横断追補）

> **設計・受入票／実Provider・Policy・監査Job・性能試験はNOT RUN。** §27.6のCoverage Debtと§37のOrganization由来Record Policy（Draft #2781、合成VP-01〜33）、#2622のDeclared/Actual/Disclosed、#2628の実取得Witness、#2516の用途別開示を接続。中央Policy/品質Registry・TEI Host・第二の意味正本は作らない。

**同一Ruleでも、検査対象の母集団・監査主体が読取可能な母集団・実際に取得したデータ・開示可能な結果は別**。監査目的を理由にHost/ProviderのGrantを拡張しない。§37のRecord Policy例外も、他Policyを迂回する実Grantではなく、品質Ruleの適用除外を意味するわけでもない。

| Footprint | 必要な束縛 | 禁止推論 |
| --- | --- | --- |
| **Required Population** | Owner受理RuleのSubject/適用期間/Relation/粒度/対象Tenant/必要Field/Rule版 | DDL上のKeyや画面一覧から全社Ruleの対象集合を推測する |
| **Authorized Read Scope** | 実Principal・委任・Host Grant・Policy版・Tenant・Provider Enforcement・時点 | Modelに監査Ruleがあるので他Tenantも読める、例外=Allow |
| **Actual Acquired Scope** | 実SQL/API、RLS通過後のRow/Field、Reader route、Read Cohort、Snapshot/LSN、完全性Witness | SELECT allで全件取得したはず、出力0件=母集団空、マスクしたFieldは実際には読んでいない |
| **Permitted Disclosure Scope** | 監査Owner/運用担当/Studio/SHO/AI/外部Consumerごとの現行開示Policyと実送出 | 内部Read成功は全Consumerへの公開許可、匿名化した件数や時刻からは漏れない |

Requiredが全社、Authorized/ActualがTenant A限定なら「Tenant Aの対象内では違反未観測」と「Tenant Bは未検証/非開示」を分け、全社品質PASSへ昇格しない。未検証Tenantの存在・件数をどこまで公開できるかも現行開示Policyに依存する。強権限監査ユーザー・RLS bypass・Tenant横断の一元収集は自動生成しない。

**実現候補**：①Tenant/Policy Ownerが許可したScopeごとにread-only CheckSetを実行、②現行Policyが許す品質メタデータだけを合成、③跨Tenant一意性・相互Relation等の全体制約に限り、必要な横断Grantや安全な比較方式を別途受理する。局所PASSの合算では、各Tenantにまたがる外部ID衝突や関係違反を発見できない。EKI Task/Lease・Workerの配置はデータRead Grantを発行しない。別Nodeへの再配置・retry時もHost/Providerの現在資格を再確認する。

**全件・増分・重点・標本を混同しない**：

| 実施方式 | 費用最適化の候補 | 許される品質Claim |
| --- | --- | --- |
| **Complete Scan** | 固定Snapshot/対象閉包/一貫した実Provider Readで全件検査 | 指定Rule×時点×Scopeでの違反/違反未観測。RequiredとActualの一致がなければ全社PASSでない |
| **Incremental Scan** | 前回Full Scan＋欠落のない変更ログ・挿入/削除・否定依存・旧/new Relation・Rule/Policy版の再資格化 | 証明された閉包に限り前回Claimを継承。CDC Gap/Policy変更により閉包不明ならFull/Unknown |
| **Targeted or Stratified Scan** | 既知異常・重要な変更・高リスク期間の優先検査 | 重点対象内のみの事実。未検査層はCoverage Debtのまま |
| **Statistical Sampling** | 費用を下げて兆候を観測。母集団・抽出方式・選択確率・重複/偏りを記録 | 標本内の観測値と、明示された統計仮定に限定した推定。標本で違反0件を全件の論理的証明としない |

例えば全10万件中1000件の標本が無異常でも、対象全件を検証した意味にはならない。統計的な信頼度や閾値はOwner/統計設計者が事前に設定した前提に従い、実測や抽出設計なしにAIが補完しない。必須Online Guardは標本Scanで置換しない。

**版・鮮度・現在資格**：FindingはRule/Subject、actual Read Scope/Snapshot、対象Occurrence、診断kind、Policy/Provider/Source revisionへ束縛する。Dataset byteが不変でも、Policy root/axis/revision、現在Grant、開示対象、Replica replay、抽出設計の変更で古いCoverageと開示資格を再判定する。A→B→Aでは過去Receiptを保持しても旧Grantを復活させない。現在AI/SHOへ結果を出す際は、診断文面、件数、Source span、error/timingも含め現行Disclosureを照合する。

**§27.5〜27.6／§37／#2628の既存反例に差し込む追加差分（新Runnerなし）**：

| ID | 入力と反例 | Expected/停止条件 |
| --- | --- | --- |
| QDS-01 | 全社Rule、Tenant AだけRLSで取得可・Aに違反0・B未取得 | A限定PASSと全社Coverage Debt。BへのRead Grantを自動発行しない |
| QDS-02 | 秘密FieldをProviderで過大取得し出力で伏字／必要Fieldを未取得 | 出力非表示と物理未取得を区別。missing/withheldをNULL・合格にしない |
| QDS-03 | Record Policyのroot/axis/revisionだけ変化、業務Data/Ruleは同byte | 旧Host Grantと旧全件Coverageを無条件再利用しない |
| QDS-04 | Tenant A/Bがそれぞれ正常、共通外部IDの衝突がTenant間に存在 | 局所PASSから横断一意性/整合性のPASSを推論しない |
| QDS-05 | 10万件のうち1000件を抽出し0違反、未抽出側に違反 | sample passをcomplete passにしない、仮定なしの信頼度を創作しない |
| QDS-06 | CDC Gap＋古いReplica＋EKI再Attempt＋Consumer Policy変更 | 増分のFull同等性や最新Primary/AI Disclosure資格を無条件継承しない |

**小さな二領域の縦断**：受注・明細ではTenant/Field RLSがQuality Claimを狭める例、組織Membershipではvalid/known時点・Record Policy変更が実ReadとDisclosureを変える例を使う。各々同じAccepted Rule/Operator/実Target版に対し、宣言Read・実SQL/API・Providerからの実Row/Field取得・品質Claim・Consumerへ出たbyteを一Attemptで区別する。#2628の人工Read WitnessやDraft #2781のNode試験を実RLS保証に昇格しない。Go/SQL/Lean/Replica/EKI/実Host/Studio/SHO/AIの新しい製品E2Eは未実施。

**責務**：#2622/#2628＝Data Context/Provider Real Read、§37/#2386/#2781＝Organization Policy、#2516/#2515＝用途別開示、#2514＝独立Expected/Claim Coverage、#1130＝読取専用Batch、#2582/#2600＝差分・再資格、#1174＝現在Authority/Effect、§35〜36/EKI＝配置/期限/回復。新しい監査専用万能Grant、Policy Engine、Quality Registry、Schedulerや正式Schema/ADRを採用しない。



### 27.8 派生品質検査そのものの検出力――独立Oracle・故障注入・全件対照と費用の両立（2026-11-03横断追補）

> **設計・実証計画／NOT RUN・NOT ACCEPTED。** §26のオンライン/夜間/Lean派生、§27.6の対象Coverage Debt、§27.7の四Footprint、§35〜36のEKI資源/期限管理は既存のまま。本節は**検査対象が全件でも、検査式・生成Go/SQL・Readerの間違いを見逃す**欠落を埋める。#2514（独立Oracle/Mutant判別）、#1063（生成時/実行時二用途）、#2593（grain/集計法則）、#2622/#2628（Data Operations/実Read）、#2018（成功偏重を避けた比較）へ引継ぎ、第二のTEI Validator/Proof Engineや共通品質Runnerを作らない。

**二つのCoverageを同じ百分率にしない**：
1. **Data coverage**＝適用対象Rule×母集団×Row/Field/Relation×期間×Snapshotのうち、権限を持つ実Readerが完全に検査できた範囲（§27.6/27.7）。分母/閉包不明はunknown。
2. **Fault-detection evidence**＝特定Rule・生成器・実Targetの検査が、独立に定義した既知誤りを拒否できた範囲。固定Mutantを何件検出したかは**試験集合に関する局所的な判別力**であり、未知故障の検出確率、製品のデータ品質、Lean証明、全Provider適合を一般保証しない。

**三つの独立Witness**：A. Rule側のAccepted Meaning・Source/Operator/Nullable/Unit/grain/時点と独立Expectedの導出元。B. 実行されたGo/SQL/Provider Queryの生成元・実消費byte/Target版・Relation join/Predicate・実取得Scope（Declared ReadとActual Acquisitionを分離）。C. 観測された実Data Snapshot・検査件数・violation/unknown/timeout・結果にアクセスできるConsumer/Receipt。**A/B/Cの独立性と版閉包が立証できない共同Claimはunqualified**とする。オンラインGoと派生SQLが同じGeneratorの誤Loweringを共有すると、両者が一致しても真のRuleから逸脱する。Leanの法則証明も、命題生成とProvider/Targetへの対応が誤ればその逸脱を検出しない。

| Pressure（既存DQ/DQX/DQA/QDSへ投入する故障注入） | 故障の具体例 | 合格に必要な観測／誤成功停止線 |
| --- | --- | --- |
| **QFV-01 算術・型** | half-evenをhalf-upとして生成、Decimalをfloatで演算、Decimal→String同ID、未取得をNULL/0へ | 期待値は候補Go/SQL/同じEvaluatorから作らず、例: +1.005を小数2桁half-evenなら1.00。違反/拒否の局所差を検出 |
| **QFV-02 集計/粒度** | 受注明細2件×配送3件のjoin fanout後SUM=600、DISTINCT amountで200を100へ誤縮約 | 独立の受注合計Expected=200、粒度/重複・Relation/Measure法則を固定。Go=SQL一致でも誤結果ならFail |
| **QFV-03 非出力依存** | 表示値同じでもHidden WHERE、EXISTS/NOT EXISTS、sort/limit順、Dynamic Relation先変更 | Query非出力依存の版と実Read cohortを観測し、空/partialと完全不存在を分離。結果byte一致だけでは旧Pass再利用不可 |
| **QFV-04 不完全/認可** | 全社RuleなのにRLSはTenant Aだけ、secret Fieldを内部Overfetchして結果だけマスク | Required/Authorized/Actual/Disclosedの差を検出。RLS下0件を全社無違反にせず、不正取得を出力マスクで隠さない |
| **QFV-05 増分の失効** | CDC gap、行削除、否定依存、A→B→A、Rule/Operator/Policy改版、旧/new Relation片側欠落 | 同じ固定対象のFull ScanとIncremental Scanの差、原因/再Scan/unknownを比較。誤再利用をFail、過剰失効も費用として測る |
| **QFV-06 実Run/効果** | Exit 0だが出力Artifact欠落、Replica S0 PASS後Primary S1に違反、EKI late Attempt、是正Commit unknown | Job受理／品質Claim／現在資格／Repair Receiptを別判定。旧Resultと効果不明を緑色にしない |

**Quality Check生成器の最小受入票**：元Source occurrence + Accepted Rule/Operator/Meaning revision、独立Expectedの具体的算出根拠とCandidateとの共通依存警告、Lowered Go/SQL/Leanの実byteと実Reader、対象Data Context/依存（表に出ないPredicate/Join/Exists/Time/Policyを含む）、Fault注入前後のExpected/Got/diagnostic、完全性と対象Snapshot、該当Targetのpass/fail/unknown、実コスト。既存#2514のTAO/MR、#2628のRead Witness、#2593 MC、§24 VS・§27 DQ/DQXを**同一試行で参照**し、Fixture正本や新Schemaを重複生成しない。

**Baseline / Optimized / Unverifiedを分ける**：
- Baseline：固定されている業務Rule・母集団・Snapshot・独立Expectedに対し、**可能なとき完全なread-only scan**を行う。断定可能なScopeを明示する。
- Optimized：同一条件で、生成したTyped Go演算・Provider pushdown・Snapshot reuse・Incremental・重点検査の費用差を比較。オンライン必要Guardの省略は§25の証明条件に従う。
- Unverified：実Providerを読めない、Full baselineが時間切れ、Mutationを試していない、Oracleが候補と共通出所、権限/Replicaの完全性不明の場合は、**0 cost／0 violation／成功扱いにしない**。既存の成功試行だけから平均を取って速さ・品質を主張しない（#2018）。

**KPIの候補（事前定義が条件）**：オフライン対象の検証完了率とCoverage debt、規定期限内にConsumerへ到達した検査結果、固定Mutant集合に対する検出件数・見逃し件数・拒否理由、false-positive候補の再確認率、Full/Incrementalの誤再利用・過剰再検査、scan CPU/IO/bytes・Provider roundtrip・Replica lag・転送費用、Rule改版から再生成/再監査までの時間、診断/修復の手作業時間。**Mutation detection rateを未知故障発生率の推定に転用しない**。夜間Scanの実測費用と、必須Online Guardのp95/allocsへの影響は別指標とする（§25/#565）。

**実証順**：F0＝受注合計1 Rule＋組織Membership期間整合1 RuleのSource/Meaning・独立Expected・誤実装を固定。F1＝生成器に依存しない人工Fault injectionでQFV-01/02/04の検出力を先に照合（手書きSQLは比較候補で正式TEI Targetの完成証拠にしない）。F2＝実Go/SQL/Provider Readerに到達した場合のみ、同じData Context/Read Witness/実Query byteで正常/異常/unknownとQFV-03/05を対照。F3＝Full/Incremental/Targeted/Statistical Samplingの検出力・費用・結果資格を、分母とDeadlineを保持して比較。F4＝実Job/Replica/EKI/SHO/AI/是正は各Ownerへ渡し、到達できない系統はNOT RUNとして残す。**未実験の改善率や「少人数で大手品質チーム並みの保証を達成した」という主張は禁止**。

**Ownerの分離**：#2514が独立Expected・Fault判別・Claim範囲、#1063/#988が同一MeaningからのTarget lowering、#2622/#2628が実Provider ReadとPushdown、#2593がMeasure/grain、#2582/#2600が動的依存・再使用、#1130/§35〜36がJob/リードタイムとEKI候補、#2544がLean Law、#2516がConsumer開示、#2018が対照実験と費用・人手の生産性指標を所有する。実Go/SQL/Lean/Replica/EKI/Host等の実結合・Owner受理は未実施であり、この節自体はDocs-only。


### 27.9 上流受渡しと外部突合――「DB内全件正常」と「業務事実の取り込み完了」を区別する（2026-11-04追補）

> **設計・受入票／NOT RUN・NOT ACCEPTED。** §27.6〜27.8のData Coverage／Footprint／Fault-detectionは、すでに存在するRowに対しては強いが、**必要なRow・関係・業務イベントがそもそも到着/保存していない場合**、全件Scan・Go/SQL一致・Lean Proofがともに正常でも取りこぼす可能性がある。本節では「モデルから派生するQuality Check」の利点を維持しながら、その**モデル外にある独立事実**をどこから取得し、何を根拠に欠落・過剰・不一致を主張できるかを具体化する。§40〜44のAI-first Scenario／実消費Source・Artifact／一Attempt Evidence、#2514独立Oracle、#2622多Provider Data Operations、#1130非画面Batch、#2386組織のvalid/known timeに接続。新いわゆる万能Truth DB/Registry、Host、監査用認可エンジンは作らない。

**違う三つの整合性Claim**：

| 種別 | 確かめること | あくまで未証明なこと |
| --- | --- | --- |
| **Internal conformance** | DB/Providerで実取得した完全なSnapshotの行/項目/関係が、受理済みData Ruleを満たす | そもそも到着していないOrder、見逃した取引、誤った業務Rule・全Provider横断の事実 |
| **Handoff completeness** | 上流の受理済み入力・取込Envelope・Change Stream・処理Receiptをもとに、必要なIdentity/Occurrence/Versionが下流へ伝搬し、失敗/隔離分を別扱いできる | 上流が事実を漏らしていないこと、上流と下流の業務意味が等しいこと |
| **External reconciliation** | 二つ以上の独立性が吟味されたシステム/外部台帳/承認済みControl Total間で、同じ業務対象・時点・換算規則について食い違いを検出できる | どちらが正しいか、第三の未観測システムや現実世界全体の完全性、未採用の業務同一性 |

**独立性を「別DB」「別Generator」で代用しない**：上流レコードを再計算したControl Totalと下流モデルから生成したSQLが、同一の誤った業務Key/Filter/丸め規則を共有し得る。比較の根拠について、出所（System of Record/伝票・受理された入力・ETL・DB view・外部確認）、生成経路、共通依存、Owner/Authority、対象期間を固定する。独立性が足りなければshared-origin warningとし、二系統一致を独立Oracle成功と呼ばない。業務規則自体の抜け・誤定義は、異常傾向・外部証跡・人の判断から**Rule追加候補**として発見され得るが、AI/夜間JobがAccepted Meaningを自動変更しない。

**受渡しEvidenceの最小対応列（既存Receipt/Artifactへ参照）**：上流Subject+Source System/承認された母集団・期間/識別子の名前空間、外部Event/Record Occurrenceと訂正/取消/重複許容、上流Snapshot/Log offset・watermark、受渡し対象のmanifest/member集合・実byte・件数/キー集合/単位、取込Attempt・受理/拒否/隔離/未処理、下流Business ID/Relation/Map版、実DB/Replica Read Cohort・Scope/LSN、Rule/Operator/Generator版・独立Expected、品質Finding/unknown・残債・期限、現在Owner/Disclosure。**件数合致だけでは行集合が同一、合計額一致だけでは金額誤りがない、hash一致だけでは業務意味が同じとは証明しない**。各リンクが部分的なら、そのリンクだけをunverifiedと表示する。

**時点・運用**：上流で「送信済み」は下流の「受信/受理/Commit/独立Read済み」と異なる。遅延到着・Replay・Backfill・取消・遡及訂正には、業務のvalidAtと上流/下流のobservedAt、recordedAt、検査対象として閉じたwatermarkを別々に使う。水位が揃わない二Providerの差はまず未確定として管理し、Ownerの許容遅延を過ぎた場合に期限超過/要調査とする。データ不一致が実際に見つかった場合は、その限定Claimを記録できるが、未到着・認可外・未照会を「不存在」と断言しない。外部API/Replicaは同時点Snapshotの保証がなければ「横断完全一致」と報告しない。

**オンライン性能への影響を限定**：同期処理に全件外部照合を挿入しない。変更ログ、既存業務Receipt、取込Manifest、Providerに存在するバッチ/スナップショットの証拠を使用し、レプリカ・夜間Job/別実行単位で照合する。必須入力拒否/一意性/認可/Commit判断はオンラインへ残す。照合対象の完全性が証明できる場合のみ増分照合、欠けた履歴や依存がある場合はUnknown/再取得/Full対照を比較。監査ジョブに必要な追加Read GrantはOwner/Hostの明示許可が必要であり、外部システムへ秘密原文を無条件に転送しない。EKI Task Completedは照合が正しいという証明ではない。

**既存AI-firstテスト足場への具体的な差分（QRE-01〜07、独立Expectedを先に固定）**：

| ID | 固定する反例 | 必須の独立Expectedと停止線 |
| --- | --- | --- |
| **QRE-01 Missing accepted record** | 上流受理O1/O2、下流DBはO1のみ。下流の全行は内部Ruleに合格 | Internal PASS/上流→下流のO2欠落をFindingとして別報告。下流全件PASSをhandoff completeにしない |
| **QRE-02 Duplicate or retry** | 同一上流Attemptを再配送、別Occurrenceなのに金額/本文は同じ、取消Eventあり | 再配送と正当な二件をOccurrence/ID/版で区別。Digest一致で重複削除や業務件数縮約をしない |
| **QRE-03 Control total agrees but wrong** | 2件の値が+10/-10相殺し合計一致、またはJOIN fanoutを両系統が共有 | 件数/総額一致をRecord-level一致・独立Oracleとしない。必要なら明示Key・金額・粒度で差分 |
| **QRE-04 Time/watermark** | 上流O2は受理済み、下流WatermarkはO2到達前、Replicaはさらに遅延 | Current full PASSは禁止。追いつく前はpending/unknown、許容期限を過ぎたらoverdue/findingを別表示 |
| **QRE-05 Handoff has gaps** | Transfer manifest 2 member中1 member欠落、取込Job Exit 0、前回Artifact残留 | 実入力member・受理件数・下流Readと照合。Task/Artifactの一部成功を全件成功にしない（§42/#2604） |
| **QRE-06 Model misses business rule** | Ruleが取消後の残高/所属有効期間を規定しない。内部全件・形式LawはPASS | 外部Owner受理Scenario/Control Recordと照合して**仕様不足の候補**を報告。AI案を新業務Ruleとして採用しない |
| **QRE-07 Authorization and disclosure** | 上流2 Tenant、監査ReadはAのみ、Bの存在/件数は秘密。AI要約へ送出 | Required/Authorized/Actual/Disclosedを分離。Bの情報を不許可閲覧者に漏らさず、全体保証はunknown（§27.7） |

**着手順と実体のOwner**：
- **R0（今ある足場で準備）**：#1180のScenario/Case/Expected・#2514独立Oracleに、受注O1/O2＋上流受理ログ＋下流欠落の独立正解を先に固定。#2811のAI-first test候補追加（Draft #2813 DSL JSON／Draft #2828 Driver JSON）に接続できるか確認。JSON追加の成功を実TEI Provider保証へ昇格しない。
- **R1（pure/synthetic）**：QRE-01/02/03/06の正負を既存test-only Referenceで再現し、オンラインGo/夜間SQLが両方内部PASSでも、handoff/外部突合が失敗する反例を実証する。実用Generator未在なら手書き候補/独立Oracleだけの成功と記録。
- **R2（Provider実Read）**：#2622/#2628で実RDBと上流API/file/read-only sourceの独立Readerをそれぞれ用意し、外部から得た実Input member・移送受理・実DB Rowを同一Attemptに束縛。QRE-04/05/07。二Sourceが実際には同じ起源なら独立性を下げて報告。実権限不在ならNOT RUN。
- **R3（運用費用）**：#1130の既存Headless Job、必要ならEKI委譲で、Full vs incremental、Replay/Retry、期限内の確認、レプリカ負荷、誤見逃し、手動で定義/更新した照合Rule量を同一母集団で比較。Event移送のためだけに新常駐Broker/中央Job Hostを前提にしない。
- **R4（受入）**：#2514の独立Expected/共通出所警告・Mutant、#565の性能Envelope、#1163のSource/Build/Consumer Evidence、Ownerの業務意味・データ品質保証期限を別々に確認。どのTarget/実Read入口が欠けても、成功として合算せず停止。

**責務と採否**：#2622＝実Data Provider/Relation・上流/下流Read Mapping、#1129/#1174＝取込Command/Commit/Recovery、#2514/#1180/#2811＝独立ExpectedとAI-first候補テスト、#1130＝Job/Result、#2604/#2595＝実入力Artifact/member・実Builderの消費、#2386/#2593＝時間・grain、#2516＝開示。固定業務Rule・異Source Identityの対応と外部の正本性はBusiness/Data Ownerの受理を要求する。本節は既存§27.6〜27.8と§41〜44のEvidenceを使い、**新DSL/Validator/Proof Engine、汎用外部SoR、別品質正本・中央Registryを採用せず、実装・実E2E・性能実測・Owner受理はNOT RUN**。


### 27.10 外部突合の対応付け・粒度・保存先資格――一対一比較を前提にしない（2026-11-04横断追補）

> **設計・受入条件のみ、NOT RUN／NOT ACCEPTED。** §27.9の上流受渡し完全性・外部突合を実装へ近づける際、単純なSource ID＝Target ID／件数一致／SUM一致だけで受理すると、**分割・統合・取消・遡及訂正・複数DB Alias・通貨/時点の違い**で誤った正常・異常判定になる。#2611（同じSemantic Dataの複数Persistence/DB Alias）、#2593（Measureの粒度・加算可能性）、#1129（業務Lifecycle）、#2622（多Provider Read）、#2514（独立Oracle）と§34・§40〜44を照合し、照合対象の**対応関係自体を出典・版付きの検証可能な仮説**として取り扱う。新Mapping Registry／汎用ETL Engine／Quality Hostは作らない。

#### 27.10.1 対象を「同じ名前のRecord」ではなく、受理された業務事実で対応付ける

| 確認単位 | 明示する意味・実証 | 自動的に言えないこと |
| --- | --- | --- |
| **Fact/Occurrence identity** | 上流System/Namespace・Record/Event ID・発生回/訂正元・業務Keyの受理範囲・Owner | 名前・表示ID・ハッシュ・同額だけで同一の業務事実 |
| **Correspondence cardinality** | 1:1、1:N、N:1、必要な場合はN:M。分割・統合・重複・取消のRule/適用期間/対象集合 | 行数不一致＝欠落、合計一致＝すべてのOccurrenceを追跡済み |
| **Source-to-target binding** | Source System/DB Alias/Provider・実Persistence Mappingと、その役割（現行/履歴/検証用/レプリカ）・Owner/Read Scope | 同じSemantic ID・Operation IDを持つ別DBが同じ現在レコード集合、同一書込先 |
| **Value/Measure transformation** | 粒度、Unit/Currency/Scale/Rounding、許されるSUM/COUNT/Rate/Stock変換・使用した時点/レート・誤差契約 | 月末残高の単純SUM、異通貨の暗黙加算、丸め規則未定義での適合 |
| **Temporal/correction relation** | Event/valid/known/recorded/Source accepted time、訂正/Cancel/Reverse、watermark・再送と再処理の区別 | 遅着/取消前のSnapshotを現在の全件保証、同内容の別OccurrenceをRetry重複 |
| **Read/Disclosure evidence** | Required/Authorized/Actual/Disclosed Footprint、実Query/API・Snapshot・欠落/隔離/未到着と現在の共有資格 | RLS下の0行＝業務不存在、監査用Read許可＝外部AIへ生データ送信可 |

**原則：Correspondenceは静的なModel Relationと実データ上の対応の両方を区別する。** モデルが「受注から請求へ分割可能」と宣言することは、実際の受注O1からI1/I2へ何が保存されたかの証明ではない。モデル/Sourceの構造が同じでも、Rule改版、Data Context、実Provider Binding、Row/Field Completeness、Policy/Grant、Source Revision、Artifact/Consumerで現在資格は変わる（§29/#2611/#2600）。

**Mappingの決め方**：一つの業務Objectを各システムの物理キーへ暗黙に変換しない。原DBが与える安定外部ID、取込Receipt/Manifest、受理済み分割・合成Relation、変更イベントの相関IDなどを順に候補とし、**実測できない対応辺をAIが正解として補完しない**。対応不能/曖昧は`unmatched`／`ambiguous`／`unknown`等として、対象範囲とともに報告する。必要な横断ID/MappingのAuthorityをOwnerが受理していない場合、件数差の原因を「欠落」と断定せず、検出した差異と未確認の対応を別表示する。

#### 27.10.2 保存された金額・件数が一致しても、保存と実業務の整合を証明しない

- **分割**：上流受理1件 U1=100 が下流2件 D1=60、D2=40 に正当に分割され得る。比較してよい量・粒度・期間・丸め/単位がOwnerに受理され、D1/D2の対応辺・完全性が実取得できた場合にのみ100=60+40を局所的に受理する。未紐付けD3=0や旧D2を機械的に除外して「完了」とはしない。
- **統合**：上流U1=60、U2=40が下流D1=100に統合され得る。SUM一致でもU1とU2がどちらも引き渡されたとは証明しない。個別Occurrence・受理/隔離・取込実履歴が必要なClaimでは、Control Totalだけなら未検証。
- **取消・遡及**：上流U1=100、取消イベントC1=-100と、下流の現時点残高0は一致し得る。しかし「初めから取引なし」「取引後に取消済み」は異なる業務履歴。日末Stockは期間Flowの単純総和ではなく、valid/recorded timeとLifecycle Ruleを照合する。
- **相殺/共有の誤り**：U1=70、U2=30に対し、下流D1=80、D2=20でも合計100。集計だけでは二件の不一致を発見できない。Go/SQL/Leanの同一Rule由来Expectedだけでは二系統の共通誤Bindingを検出できない（§27.8）。
- **別DB Alias**：現行DB Aの受注O1@v2と履歴DB BのO1@v1は、同じSemantic Kindでも別実対象・版である。#2611の現行候補でOperation ID一致時に後続Mappingが消える可能性を、外部照合の正しいTarget確定へ流用しない。実Source/Reader・Alias・Role・Read Cohortを明示し、同時点でない二系統の「不一致」は先にEpoch差と実内容差を分ける。

**判定の最小分類（新Core enumでない）**：`matched-in-scope`、`unmatched-source`、`unexpected-target`、`value-mismatch`、`ambiguous-correspondence`、`pending-watermark`、`incomplete-read`、`invalid-contract`。これは下流DBの既存Data Rule PASS/FAIL、Job Attempt/Receipt、修正Intent/現在のAuthorityとは独立。上流側で欠損が発生した疑いがあっても、独立根拠がなければ「実世界に存在したはず」という新たな事実を作らない。

#### 27.10.3 最小の差分Fixture（§27.9 QREを再利用し、新Runnerを増やさない）

| ID | 固定入力と負例 | 独立Expected・停止条件 |
| --- | --- | --- |
| **QRC-01 Split** | U1=100 → D1=60/D2=40、D2は明示関係あり／なしの両方 | 受理済み1:N関係と完全Readがあれば局所一致。関係未確定ならSUM一致だけで完了にしない |
| **QRC-02 Merge** | U1=60/U2=40 → D1=100、U2だけ未受理の実取込証拠 | 合計一致でもU2の受渡し完了は不成立。受理履歴と対応辺を別判定 |
| **QRC-03 Wrong distribution** | 上流70/30、下流80/20、合計両100 | Control Total PASS、Record単位の独立Expected FAILを共存報告 |
| **QRC-04 Cancel+time** | 受注100→取消-100、下流残高0、旧Replicaは100 | Stock/Flow・取消履歴・Watermarkを分離。現状0から取引不存在を結論しない |
| **QRC-05 Duplicate alias** | 同じsales.orderを現行DB/O1@v2と履歴DB/O1@v1から別Aliasで取得 | 読取Source/Role/Revision/実Queryを区別。Semantic/Operation ID一致で対象を無音統合しない（#2611） |
| **QRC-06 Round/Unit** | 明細3×19.995＝59.985の中間値、保存金額の丸め/通貨変換/時点が異なる | Contractに無い丸め・換算をAIが補わずunknown/invalid-contract。正しい独立Expectedは別途固定 |
| **QRC-07 Reconcile retry** | 同じTransferを再配送、別Occurrenceの同内容、Error後の旧Artifact混入 | Replayと新規業務事実、受理と実Commit、全member実取得とResultの世代を区別（§42/#2604） |
| **QRC-08 Scope/Owner** | Tenant A/Bを局所検査、BのSource/Mapping/値は監査主体に非開示 | A限定Claim、横断一致はunknown。Read Grant/開示の追加は明示Owner判断が必要（§27.7） |

**実証順**：C0 #2514/#1180にQRE-01 O1/O2欠落の独立Expectedを置き、受理済み`1:1`のみのBaselineを記録。C1 QRC-01〜04で1:N/N:1・取消時点を増やし、独立Oracle/固定Mutantの検出力と誤警告を確認。C2 #2611/#2622の候補Mappingと実Reader/二Providerを同一Attemptで照合できた場合だけQRC-05/06を実証。C3 #1130のread-only CheckSet/既存OS SchedulerでFull/Incrementalとリードタイム・CPU/IO/Replica負荷を計測。C4 将来のEKI委譲・Lean/SHO/AI説明は、C2実Readerと現在Consumer/Owner資格が成立した後に限定して検討。**AI-first Scenarioのテスト候補追加は可能でも、正式Driver v0未対応Operationや業務Owner未受理のCorrespondenceを実装済みと呼ばない**（§41/#2811/#2828）。

**実装責務・採否**：#2611が同Semantic Kind/別Alias/Mapping候補の衝突と実Target、#2622が多Providerの実Read/取得/欠落、#2593がMeasureの合法演算と粒度、#1129が取消/訂正/保存のLifecycle、#2514が独立Oracle・共通出所警告、#1180/#2811がScenarioとAI提案、#1130が非画面Batch、#2516が開示、#2604/#2595がArtifact/Source実消費証拠、#1174が是正Effect・現在認可。**本節は計画書の追補に限り、外部業務Key規則・照合DSL/CLI/Result Store、実Go/SQL/DB/API/Replica/Lean/EKI、性能測定、Owner受理、main統合はNOT RUN/未採用。**


#### 27.9.1 品質照合ジョブへの入場条件と最小固定Fixture（2026-11-04）

> **Docs-only／新しい試験実走・Owner意味受理はNOT RUN。** §27.9の「受注O2取込欠落」を保証するには、上流O1/O2を実際に読んだSource証拠が先に必要。#2822の空Workspaceでもvalidate OK、#2823の未知Source root・未読込ファイル、#2824のCLI発見情報とJSON契約の不足を、品質監査における**False PASSを避ける入場条件**へ再利用する。Source/Caseの存在と実消費を混同せず、§43のAI-first足場を第二Runnerへ複製しない。

| 段階 | 要求される確認 | 未成立時 |
| --- | --- | --- |
| **Declared** | Owner受理の上流受理母集団、期間・Tenant・Occurrence、下流Rule/Scope・比較に必要なSource/Case ID | 未受理・未宣言はunresolved。自動でCoverageを満たしたことにしない |
| **Discovered** | 実Reader/Manifestが列挙したroot/member/版と必要集合の差 | 未発見、未知root、空のGlobはnot-covered |
| **Loaded** | 実Parser/Providerが消費したbyte/Row/FieldとSnapshot/Source revision | 列挙だけではunloaded。実Inputの欠落を業務上の不存在と推測しない |
| **Validated** | 読込済み対象の型・Role/Rule・Pin・JSON構造・Case/Step IDの検査 | validate OKだけで実Source/独立Expected適合としない |
| **Executed** | どの固定Caseと実Go/SQL/Provider Queryを、どのDatasetで実行したか | 不実行・途中失敗・Read不完全を異常0にしない |
| **Qualified** | 独立Expectedと上流受理／下流実Read／現在Policy／Audienceがつながる | 片側不足ならunknownまたは限定pass-in-scope |

**固定Expectedの最小対照（新Scenario v0 Operationを定義しない）**：人工Sourceの上流Accepted OrderとしてO1=100、O2=200を異なるOccurrenceで事前固定。Data Ruleは下流に実保存された行のみに対して評価するため、欠落があっても内部PASSとなり得る。キー・粒度・期間・水位・独立Expectedを先に固定して比較する。

| Case | Input/実Read | 独立Expected |
| --- | --- | --- |
| **QRA-P1** | 上流O1/O2をLoaded、下流O1/O2を同一ScopeのSnapshotで完全Read | handoff-complete候補。Source/Read/実Artifactが閉じた場合のみQualified |
| **QRA-N1** | 上流O1/O2をLoaded、下流O1だけを完全Read | 内部Rule PASS、受渡しMissing O2を別報告 |
| **QRA-N2** | ManifestはO1/O2を宣言、上流ReaderはO1しかLoadedせず、下流もO1 | source-incomplete/unknown。独立に観測していないO2の欠落を断定しない |
| **QRA-N3** | 上流Manifest R2だが必要memberは旧R1、下流は2件で合計も一致 | 同世代実byteが未証明ならunverified。#2604一式Readerへ |
| **QRA-N4** | AIのJSON Caseに同一Object内のExpectedキー重複、あるいはCase/Step ID衝突 | 不適格候補として拒否。標準JSONデコード後の値だけで重複キーを検出できたとは言わない |

**重複の三種を分離**：同一JSON Objectの重複キー、異Case/StepのID衝突、同一業務Eventの再配送は別。Raw JSONの重複Key検査能力は#2813 DSL候補と#2828 Driver候補それぞれで確認し、実装がなければNOT RUN。AIが実ActualをExpectedへ写したものは#2514独立Oracleの根拠にならない。

**実証順**：#1180/#2811の保存ScenarioでQRA-P1/N1を独立Expectedへ対応付け、#2822/#2823の既存CLI負例を隔離WorkspaceでQRA-N2に接続。#2604のmember実取得へN3、#2813/#2828のRaw入力適格性へN4を振り分け、#2622/#2628に実二ProviderのReadを渡す。既存Case・Fixture・Receiptを利用し、新CLI/Source Loader/Schema/Job Engine/共通Registryを作らない。**オンラインhot pathへ毎回二Source照合を追加しない**。CI・Go/SQL/Provider/Replica/EKI/Leanの新E2Eと性能実測、Ownerの入場条件採否は本節ではNOT RUN。

## 28. 一改訂のValue／Flow／Job／Consumer縦断――参照実行と外部実行を混同しない（2026-11-03横断追補）

### 28.1 並行する三つのOwner計画を、別の正本を作らずにつなぐ

- **Authoring／Source**：[DSL・Studio計画 Draft #2745](https://github.com/hat47x/tei-platform/pull/2745)が言語中立の低ノイズなDSL表層、ローカルUnicode識別子→既存ASCII Canonical IDへの明示Binding、糖衣記法のbyte保存、AI修正Preview／競合停止を所有する。日本語専用DSLやGo AST固有構文の強制はしない。Go単一バイナリ既定というTarget判断とFrontend選択は直交する。
- **意味／業務状態／Flow**：[X-TEA Data Script・Buri/EKI計画 Draft #2744](https://github.com/hat47x/tei-platform/pull/2744)がField標準Validation、Script独自Validation/Derivation、virtual Attributeの宣言と導出、old/new状態条件とEffect／Batchの境界を所有する。**X-TEAの既存Data Model/Scriptを移植・Adapterで再利用できるかが先**で、新たなState Machine、全業務共通の状態テーブル、第二Rule正本を先に作らない。受注の単純状態は本体行、組織所属は時点付きRelation、長期Flowは必要時のInstanceという物理設計の自由を保つ。
- **外部Reference／現在資格**：[外部Reference Draft #2743](https://github.com/hat47x/tei-platform/pull/2743)がSource Observation/Structural Candidate/Accepted Meaning/Artifact/Receiptの再資格化条件、Read cohort、partial/Not Found/permission-denied/offlineを所有する。cache hit／WorldFP同一からSource引用・現時点権限・外部データ完全性を推測しない。

**本節の担当**は三計画を横断する**既存#1163の一改訂・一Attempt受入順序**とOwnerへの停止位置だけ。各PRの詳細Fixture/文法/Provider/API Schemaをこの§28へ複製しない。参照側の意味正本をEKIへ移さず、EKI固有のTask/Attempt/Lease/Resource管理をTEIに再実装しない（#2744 §8、EKI Fabric #57）。

### 28.2 現在mainで区別すべき四つの真実

2026-11-03に統合状態を再確認した限定入口は、[merged #2747](https://github.com/hat47x/tei-platform/pull/2747) Studio `PlanCommand/ExecuteCommand`（`Mode=reference`、参照実行の`Committed=true`を**業務DB commitと読み替えない**）、[merged #2748](https://github.com/hat47x/tei-platform/pull/2748)／[#2749](https://github.com/hat47x/tei-platform/pull/2749) Guardによる閉鎖期間の**driver呼出し前拒否**、[merged #2751](https://github.com/hat47x/tei-platform/pull/2751) Studio read-only表表示、[merged #2768](https://github.com/hat47x/tei-platform/pull/2768) CLI `view -execute -period-state <file>`による参照実行前の期間状態判定。これらは**実DB Commit、実業務Event永続化、EKI Jobの起動、Source変更の逆書戻し、Provider/Hostでの現在の許可**を証明していない。

| 現れる表示／Result | 真のClaim（確認可能な範囲） | **未成立のClaim**と次のObserver |
| --- | --- | --- |
| Studio 表／候補Operation／`PlanCommand` | 参照可能なID・Capability・計画。Source/Role/RefのQualifierを保持 | 編集権限／保存／実Effect。Source元byteとScope・Owner、UI操作経路は#1124/#2621 |
| `ExecuteCommand` `Mode=reference, Committed=true` | 当該Reference Executorの局所結果（実DB保存なし） | DB committed／durable Event／Receipt。#1129/#1174の実Provider commitと独立Readが必要 |
| `COMMAND_PERIOD_CLOSED`／driver未呼出し | 与えられた期間状態を使う限定実行入口で拒否 | 期間状態ファイルの現在性、別入口の回避不能性、実DB拒否。Owner・Provider・Reader版の検査が必要 |
| EKI `AcceptedResult`／再試行・Lease | EKIが特定Attempt/Artifactを受理した（そのContractと版に限る） | TEIの業務遷移確定／品質PASS。TEI Rule・Read/Owner/Hostを再照合し必要時にBusiness Commit |

### 28.3 一回の縦断と既存反証への割当て（新Fixture IDなし）

**受注取消と集計の最小縦断**：固定したSource／Field/Rule／実Providerを参照し、①Studioで構造・State/Rule/期間を表示、②Data Scriptの値導出と更新前後条件を独立Expectedで検査、③同じ期間状態に対しReference Executeを観測、④実Providerが利用可能な場合のみ更新Intent→Commit known/unknown→Receipt/独立Read、⑤必要時だけTEI FlowからEKIへ型付きTaskを委譲、⑥新しいSnapshotでQuality／SHO/AIの出典を再検査する。④以降の**実装入口・権限・観測器が無ければ明示的にNOT RUN**とし、③から成果を補完しない。

| 差し込む反例 | 独立Expected（先に固定） | 所有／停止点 |
| --- | --- | --- |
| **同一Data値、Source参照先だけ移動／改版／offline** | 値やWorldが同じでもSourceIndex・Citationと現在Read資格は別、未取得はNULLや0件にしない | #2743 RF、#2505／#2600。Reader/Cacheの同時点closureが無ければ停止 |
| **virtual値がfalse→true、しかしPreviewのみ** | 計算と発火候補は存在、durable Event・Job配送・Business Commitなし | #2744 Data Script、#1637／#1130。実Effect claimへ進まない |
| **closed periodのCommandを計画／Reference実行** | Plan表示の存在≠実操作許可。現行Guardの拒否事実を記録し、未適用の入口はunknown | #2747/#2768、#1129／#1174。全入口の実Provider制約は未検証 |
| **EKIの古いAttemptが遅延完了／Job結果ACKだけ成功** | 旧Leaseの結果で新Runを上書きせず、Job Resultだけで業務State/Qualityを変更しない | #2744 §8、EKI #57。独立Business Read／Effect Receiptなければ停止 |
| **原文の保存後、AIが古いSpanへPatch** | 未変更byte保持・期待Revision不一致は書込拒否。表示言語変更はID/意味の権限変更にならない | #2745 GN/SF、#2743 RF、#1124/#2621/#2516 |

**最短の順序（既存試験資産を使う）**：

1. **V0 Source＋表示**：merged #2747/#2751/#2768の現行Reference実行と既存Source Fidelityを、同じ固定WorkSpace/Source Revisionで収集。これはReference資格だけ。
2. **V1 Rule/Virtualの意味互換**：#2744のX-TEA移植候補を一つだけ選び、#2514の独立Expectedと未取得/NULL/partial負例でGo実行可否を比較。実TEI機構が無ければ `NOT RUN: missing concrete implementation`。
3. **V2 実Effect**：#1119 P0-a/P0-b・#1129/#1174の実DB/HostとOperation identityを確認。Commit unknown→再送なし→独立Readという現行重要課題を、Syntheticな`Committed`から分離する。
4. **V3 Optional EKI／Consumer**：実Job契約/LeaseとTEI Ruleが明示的に接続できた場合だけEKI側へ委譲。SHO/AIは現在のSource/Disclosure/Read cohortの資格を別判定する。V3はV0/V1の実行前提ではない。

**採否**：TEIの標準実行はGo単一バイナリ、他Targetは用途別の補助。常駐TEI Host／汎用State Store／二重Scheduler／第二Source・Meaning Registry／万能Capability Gateは作らない。Reference UIの`Committed=true`、EKIの`AcceptedResult`、Replicaの`pass-in-scope`を単一の`success`へまとめない。Ownerが業務意味を受理した事実と、実Host/Providerが今回のEffectを許可・観測した事実も別。

**検証・変更の範囲**：この§28は**並行Draftとmerged PRの実在に基づく計画追補のみ**。今回、実TEI Studio/GUI/CLI/DB/Provider/EKI/SHO/AI E2E、V0〜V3、業務Meaning受理を新たに実行・承認したわけではない。§27以前の設計を上書きせず、#2743/#2744/#2745の設計本文・既存Issue/ADR・本番Provider・Runtime APIを変更しない。


## 29. モデルの参照・合成・継承――構造再利用と業務意味・Authorityの漏れを分ける（2026-11-03横断追補）

> **状態：既存計画への選択基準・反証票。正式な`extends`文法、共通Equivalence API、継承Runtime、Owner認可継承、SQL Migration自動生成の採用ではない。** 2026-11-03の「モデル再利用/継承」横断レビューを、#2386/#2510の組織、#2620/#2638の自己記述Role、#1129/#1126/#2622の業務状態と保存、#2505/#1920の改版、#2600の選択的再生成へ接続する。既存§21のGR-01〜10と§24のVS-1〜5を置き換えず、**同じ実対象で互換性を判断する際の不足列**のみ追加する。

### 29.1 再利用する単位と「似ている」の意味

**構造を共有しても、業務意味を共有したとは限らない。** 最初に次の五つを別の比較として記録する。

| 共有単位 | 共有できる候補 | 共有しただけでは成立しないこと |
| --- | --- | --- |
| `Data Shape / Type / Attribute` | 値域、Record/Relationの形、型検査・表示部品 | 単位/精度、FTEと人数、主従関係、業務Key、NULL/欠損/非開示の同義、同一Entity |
| `Semantic Role / Operator / Rule` | 明示Role ID、typed Contract、採用済み意味と依存、検証法則 | quoted `role`文字列の起動、同名版の実装同一、未受理業務規則、Clientへの実行資格 |
| `Organization / Business Relation` | Person/OrgUnit/Membership/Position/Occupancyの一般構造、型付きRelation | Membership⇒承認権限、複数Membership⇒単一所属、職位⇒現行Host Grant、組織改編⇒未完了Caseの自動移譲 |
| `Workflow State / SQL Storage` | 正本Data Rule/Guard、logical stateと明示Storage Binding | `status`列の名前だけで同一Lifecycle、並行tokenを単一列へ無損失縮約、FK⇒業務Relation、SQL CHECK⇒遷移許可 |
| `Source / Artifact / Consumer` | 固定版のSource occurrence、Role Pin、派生Artifact byte、歴史的Evidence | Source移動⇒現引用の自動再同定、Manifest一致⇒実Reader受領、過去PASS⇒現Grant/開示/Commit許可 |

**意味上の「Role」も区別**：構文・意味解釈の`Semantic Role`、業務参加・責任の`Business Role`、外部Hostの`Principal/Grant`は三つの関係である。名前や同じRecord形式による暗黙の写像を作らない。Ownerも「モデル意味を受理するOwner」「Data/Relationの管理Owner」「Provider/Pluginの供給責任」「実行を許すHost主体」を分ける。この区分は新しい標準Owner enumの採用ではなく、既存受入票の識別欄である。通常組織Dataを一律に特別保護せず、必要なField/Provider/運用境界の開示だけを#2516で確認する。

### 29.2 再利用の既定順：参照・合成が基本、継承は契約を限定して採用

| 手段 | 判断基準 | 反例・やめる条件 |
| --- | --- | --- |
| **Reference（優先）** | 既に受理済みの同じ定義・Rule/Operator/型を**明示版とSource/Provider pin付きで**使う。案件のSubject/Owner/Grantは呼出し側で別Binding | 親定義の新版を無条件追随、同じdisplay名を別Tenantの同一IDへ自動解決、型同一で意味同一を主張 |
| **Composition（優先）** | 独立したIdentity/Relation/Role/Rule/Storage実現の関係を、入力・出力・制約・多重度・時点を宣言して組む | 「PositionがMembershipを包含」「Workflow StateがBusiness Objectと同一ID」という単一継承鎖への押込み、二重のCurrent State Writer |
| **Inheritance/refinement（限定）** | 同じ主体・概念の専門化で、**親の観測契約を子で維持できる**ことを独立Expectedと差分検査で確認した場合。定義の継承そのものは実装選択待ち | 追加した必須Input/Guardで親の利用者を拒否、親の許す結果を弱める、作用/副作用を拡大、ParentのOwner受理/Host Grantまで子へ自動コピー |

限定的な置換可能性の判定では、親を使う既存Function/Consumerについて **(a) Inputの受入条件を強めない、(b) Output/結果/不変条件を弱めない、(c) 未宣言Effectを追加しない、(d) Identity/Relation cardinality・bag/grain・NULL/unknown・有効時点の期待を壊さない** ことを確認する。子が親より厳しい業務制約を必要とする場合は、親の代替としての一般互換ではなく、**特定用途での新しい受理・明示Composition**とする。形式的な型変性/置換判定を一般の業務意味受理へ自動昇格させない。

**最初の三つの異質なSubject**（いずれも作業例・正式DSL/受理ではない）：
1. **受注`sales.order`**：`status=pending/submitted/cancelled`はData Script/Ruleが唯一の業務意味、同じ本体レコードの状態Fieldを第一候補にする。取消可能性・更新Permissionは単なる`status`型の継承で成立しない。
2. **組織`organization.membership`**：Person P2にMembership M2/M3が同時に存在し、別の`validAt`/`knownAt`・FTE/Position/Responsibilityを持つ。MembershipはPersonのsubclassや単一`parentOrg`ではなく、**独立Occurrenceを持つRelation**として表現する。組織編成D1→D1a/D1bでRole/Case Q1の責任先・承認Grantを暗黙移植しない。
3. **承認Workflow**：同じBusiness Subjectに対し申請Attempt A1/A2があり、`approvalStatus`が複数active token/履歴を持つなら、単一`Order.status`へ強引に継承して縮約しない。業務本体の複数Field/構造化Field、sidecar、中央/外部Storeを**容量・更新原子性・多重度・運用要件の証拠**で選択。物理保存の第一候補は本体状態属性だが、意味的な非可逆圧縮を許さない。

Buri由来の図表駆動・State Machineは、業務Procedure・Rule/Function・実行中の永続Workflow/Timerと区別する（Draft #2744）。またゲームやUIのState Machineは業務固有のOwner/承認を継承しない別Domainの意味Role候補。全Domainへ同一Workflow Schema/Runtimeを要求しない。

### 29.3 保存版・受理・生成・実行の独立Pin

**固定すべき再利用証拠の最小束縛**（比較票の列であり新永続Schemaではない）：

`source locator+occurrence / original bytes+source index / workspace formatVersion / grammar&role ID+version+contract fingerprint+implementation fingerprint of eager dependency closure / accepted subject+meaning revision+Owner scope / key+Relation+time semantics / binding target+Provider/schema+operator pin / generator/toolchain+artifact member+manifest / actual reader/consumer+purpose / current disclosure&Host grant / observed result+attempt+time`。

- Workspace manifestの`formatVersion`未指定拒否は[merged #2694](https://github.com/hat47x/tei-platform/pull/2694)で導入済み。**2026-11-03現行mainの`loader.go`は版1/2を受理し、未知版3以上を拒否**する。版2のSemantic Sourceでは明示的な`formatVersion 2`ヘッダを要求し、版不一致を拒否する局所Goテストがある（`format_version_test.go`）。この**形式版**はMeaning/Owner/Roleの版やImport元の実Source資格ではない。旧形式の完全変換器、Plugin固有書式、Owner意味受理は別。
- [merged #2638](https://github.com/hat47x/tei-platform/pull/2638)の有限Role Bootstrap・推移Pinは合成Reference/設計の証拠。Role名とversionだけの一致、実装Fingerprint未照合を現在の相互運用資格にしない。新しいInheritance ResolverやRole Authorityを別に作らない。
- Source`A→B→A`の同じ最終byteや、ファイルのrename/content保持から**期間内無変更・Occurrence自動対応・最新Review適用**は推論できない。#2505の多版Source対応はCandidateで、現在の意味Owner受理・SHOの文書Occurrence/Reviewと区別する。
- 本体`status`Field／JSONB／sidecarなどStorage Bindingの改版では、**Key tupleの型と順序、Subject/Workflow Instance/Active occurrenceのID、Provider/Tx/CAS能力、current writer唯一性、必要履歴、NULL/absent/partial** を明示検証する。単純な`ALTER TABLE`や同名FKだけで業務同義のMigration成功を宣言しない。
- 現Grant/Disclosureが変わっただけなら旧Artifact byteや歴史的検収は保存候補だが、現在の実Action/送出資格は再確認する。再利用の適否は`structural candidate`、`accepted meaning`、`actual consumer/host qualification`という別の主張である。

### 29.4 MRU-01〜12：前回横断レビューを既存Owner試験へ割り付ける最小互換性票

| Case | 明示する負例（比較する変更は原則一軸） | 独立Expected／誤受理を拒否する条件 | 既存の受け皿 |
| --- | --- | --- | --- |
| **MRU-01** | `Person`/別Business Person型が同名・同Shape | Shape一致のみなら`structurally comparable`まで。主体同一／Owner同一とはしない | #2495/#2497/#2514 |
| **MRU-02** | 同一Attribute IDでFTE比率→人数、Decimalの丸め/単位変更 | 値・型互換でも意味・粒度・単位の変換が未受理なら代替不成立 | #2386/#1063/#2514 |
| **MRU-03** | Membershipを持つ社員を承認者として提案 | 所属を見せることは可、Business Role/Delegationと現Grantが無ければ実承認不可 | #2386/#2507/#1174 |
| **MRU-04** | P2のM2/M3同時所属とD1→D1a/D1b改編、Case Q1未処理 | Membership occurrence/時点/責任を保持、PositionやCase ownerの一意性を捏造しない | #2681/#2386/#2510 |
| **MRU-05** | Root Roleのversion同一で実装Fingerprint I1→I2 | Version-only pinを適合としない。影響するTyped Meaningを再資格化 | #2620/merged #2638/#988 |
| **MRU-06** | Root不変、Eager依存RoleだけI1→I2（別の無関係Roleは不変） | 依存閉包は再評価、独立Claimの全失効をしない | #2638/#2600 |
| **MRU-07** | Orderの同じRuleをembedded列→複数Field/SidecarへStorage Binding変更 | 状態/Effectは独立Oracleで比較、物理Schema一致を業務同義にしない | #1129/#1126/#2622 |
| **MRU-08** | 並行Token2つを単一`status`文字列へ写像 | Branch/履歴/Instance identityを失えばlossless継承不可 | #1129/Draft #2744 |
| **MRU-09** | 複合Key `(tenant,orderNo)`、異なるSortKey/State Instance/保存先の合成 | 型・順序・Scope・Key同定・CASを固定。物理PK/FKや文字連結で誤結合しない | #1126/#1129/#2622 |
| **MRU-10** | Sourceファイル名移動のみ、内容/World同一 | Meaning/生成再利用候補とSourceIndex/SHO引用再確認を分離 | merged #2649/#2505/#2516 |
| **MRU-11** | Relation先/Credit Limit、`NOT EXISTS`の0→1→0、出力値不変 | 動的/否定依存・Scope/row+field completenessを再評価。変更が未観測ならUnknown | #2387/#2582/#2600 |
| **MRU-12** | Owner承認範囲/現在Host Grant/Disclosureだけ失効 | 旧検証は歴史的に保持、実Command/外部送信は新資格なしに実行しない | #2516/#1174/TAI #271 |

**Referenceの合格と、実TEIの合格を混ぜない**。既存#2638 RB系の合成24件、#2655の組織合成、#2649の実Go CLI/Artifact Readerはそれぞれ独立の証拠。上記**MRU-01〜12を同じSubject・Meaning Role/Plugin Pin/Storage Provider/Hostで走らせた正式E2EはNOT RUN**。

### 29.5 実装済みの足場と、実際にまだ届いていない範囲（2026-11-03現在）

- **実Go Compiler/CLIで観測済み**：[Draft #2681](https://github.com/hat47x/tei-platform/pull/2681) exact HEAD `8efe0d351a244b57fefd00b498a5b47136619ac5`の[Native organization workflow #41580077197](https://github.com/hat47x/tei-platform/actions/runs/41580077197)は**completed/success**。Go 1.26.10で`TestOrgHeadless`、`validate`、`operations -application org_view`、`artifacts`が成功。Membershipの`list/read`2件だけを導出し、更新/承認Bindingを捏造していない。ArtifactのManifest `complete=false`、`findings=0`を保持している。**同じData Modelの継承・変更時互換性、実Provider/DB/Host/Owner受理は未実証**。このDraftを汎用組織管理の製品完成と呼ばない。
- **別Domainの実CLI実証**：merged [#2649](https://github.com/hat47x/tei-platform/pull/2649)の固定実行には`validate→operations→artifacts→-out独立Reader`とSource-only移動・旧新Revision混在拒否がある。しかし異Domainのローカル成功を、MRU-04やMRU-07のRule/Storage互換へ合算しない。
- **Role/形式の基線**：merged #2638は有限Bootstrapの合成Role検証、merged #2694はWorkspace Manifestの形式版拒否であり、前者は正式Role実装、後者はModel内容版/Migrationまでの受理を証明しない。
- **未接続**：二Domain間の実Role/Attribute再利用、実Model Reference/Refinement Grammar、Business Role/Owner受理、Physical SQL State Storage/複合Key/CAS、外部Provider/Host認可・Effect、SHO文書Occurrence/Reviewの実受渡し、Target Generated binaryの同等性、Selective full vs incremental性能測定は**NOT RUN／Owner未受理**。

### 29.6 最小の実証順と採用条件（既存#1119/#1163へ渡す）

**R0 — Source・Shape/Role契約の固定（read-only）**：`reference/standard-crud/workspace`と`reference/org-headless-workspace`を別Domain基準に用い、formatVersion/Source bytes/Subject/Type/Attribute、Role/Operator実装依存Pin、独立Expectedを固定。**再利用元と利用側の同じMeaningを人工的に仮定しない**。MRU-01/02/05/06で、構造可視と意味採否を別票にする。現実の`extends`構文がない範囲は「NOT RUN: missing concrete entry」と明記して手書き継承Engineで製品合格にしない。

**R1 — 合成と限定置換可能性**：まずReference/Composition経路でP2のM2/M3、Order状態Rule、承認Attemptを同じtyped Input/Output/Ruleに束縛可能か検証。追加必須Guard／出力保証弱化／Effect増加／未知関係の縮約を変異として加え、独立Expectedで拒否。受入済み意味の追加・変更は元Owner判断を必要とする。Graph/Tree/UIの見た目一致を互換にしない。

**R2 — 実Go縦断・Storage Conformance**：#2681の実Source→World/OperationSurface/Artifactと#2649の実Readerを、**同一固定HEAD/Workspace/Subject/Role版**で接合。次にEmbedded→Sidecar等の物理Binding変更を#1126/#1129/#2622の実Providerへ委譲し、NULL/Key/状態token/履歴/条件付き更新/独立Readの変化を確認。実Provider入口が未接続なら`NOT RUN`を残す。#1129の現行Writer一つの原則に反して二重状態を書かない。

**R3 — selective regenerationとCurrent Qualification**：Sourceのみ、Rule/Role only、Storage Binding only、Dynamic Relation/0件集合、Consumer/Grant onlyを一軸ずつ注入。`Clean full`と`selected invalidation`を比較し、**(a) Source/Meaning再読取、(b) 生成Build要否、(c) Artifact/文書再読取、(d) 実Data再取得/Claim再評価、(e) 現Grant/開示資格**を別列にする。False reuseだけでなく、無関係成果物の不必要な全再生成も計測する。変更通知Coverageが欠けるときはskipしない。歴史的Receiptを削除しない。

**採否・停止**：2Domainを試しても、型/属性の一致を意味同義、MembershipをApproval Authority、同じ`status`を同じLifecycle、物理保存変換を意味保存、同じSource byteをOwner再受理と自動判定した場合は該当Reuse方式を棄却。親型への置換可能性・有益性が独立Expectedと現実のConsumerで確認されない限り、TEI Coreへ一般継承機構を増やさない。逆にReference/Compositionで目的を満たせるならそれを既定に採用候補とする。

**Owner/ADR帰属**：#2619/#2620/#988は共通DataとRoleの意味、#2495/#2497は構造とProjection、#2386/#2510は組織、#1129/#1126/#2622は状態/SQL/Provider、#2505/#1920はSource/形式版、#2600/#2582はBuild/Runtime失効、#2514は独立Oracle、#2516/#1174はConsumer/Host資格、#1163/#1119は同一Attempt・優先順位。**本節でIssue/ADR本文・優先度・Ownerの正式意味・新Core/Parser/Resolver/Host/認可Engineを変更しない**。継承文法と正式契約は実証が揃うまで保留する。


## 30. 外部Importの編集時診断とAI修正――固定Sourceから現在資格までの最短受渡し（2026-11-03追補）

> **新規実行なし／設計上の接続順序のみ。** 詳細のSource鮮度・失効と既存Go Loader接続は[Draft #2743](https://github.com/hat47x/tei-platform/pull/2743)（RF-01〜24／V0-1〜6）、原文・AI/Unicode Authoringは[Draft #2745](https://github.com/hat47x/tei-platform/pull/2745)（GN-01〜16）およびmain統合済み[#2674](https://github.com/hat47x/tei-platform/pull/2674)（SF-01〜18）が所有する。本節は§28の一改訂・一Attempt受入と両設計を重複なく接続する。独自Source Resolver／Writer／Host／Central Registry、正式DSL文法を新設しない。

### 30.1 Managed Sourceと外部Workspace Importを混同しない

**Managed Source**はWorkspaceの管理範囲内でReaderが実際に読んだbyteとSource occurrence/spanを固定する。**明示的外部Workspace Import**はalias/path/namespace/optional semanticDigest、許可されたScopeとProvider/Language Pinを持つ別の境界であり、「Workspace root外」を理由に一律拒否しない。許可されていないroot越境、redirect/symlinkによるscope逸脱、read-time TOCTOUは既存[#1017](https://github.com/hat47x/tei-platform/issues/1017)／#1002の安全境界で拒否する。実Go入口は`internal/workspace/imports.go`・`workspace.Load`・`internal/sourceacceptance/verify.go`、言語選択からCandidate/Source Indexへの接続は[#1018](https://github.com/hat47x/tei-platform/issues/1018)／#988の未充足条件として扱う。

編集時の解析対象も分ける。宣言済み**必須外部ファイル**が無ければその適用段階でblocking diagnosis、optional不在/一時unavailable/permission-denied/offline/partial/権威未確定not-foundは区別する。`.tei`のDSL Sourceと、別Pluginが所有するnative `.sql`/`.go`/`.json`/`.yaml`/`.md`を無条件に同じParserで再解釈しない。Markdownのコメントやコードフェンスに見える文字列だけで、実行可能なReference/Actionを発見したことにしない。

**検証配置**：編集時＝構文・Source byte span・参照宣言・候補修正、Load/Build時＝外部Importと固定semanticDigest・Role/Operator Pin・実依存closure、実Provider/Host境界＝現在のGrant/partial read/Effect/Receipt。Go単一バイナリの純粋hot pathに毎回の全Source再ParserやOwner意味照合を要求しない（§25）。Source resolved ≠ Semantic accepted ≠ Build/Consumer qualified ≠ Runtime authorized。

### 30.2 既存Fixtureを一試行に差し込む（新IDを増やさない）

| 順序 | 対象と独立Expected | 停止／禁止推論 |
| --- | --- | --- |
| **R0 Source/Import** | #2743 V0-1/V0-2：利用側Managed Source byte不変、外部Namespace A/Bの意味的属性・型と旧Pin拒否を**Loader出力とは独立に**固定。既存`external_acceptance_boundary_test.go`をfocused Go実行 | 既存テストのLoader由来semanticDigestを自己Oracleにしない。Source同一→外部Meaning同一・Owner受理済みとしない |
| **R1 Reference状態** | #2743 RF群：必須/任意、scope外symlink/redirect、offline/permission-denied/partial、A→B→Aを観測。宣言Scope内の正当な外部Importは許す | 404を権威ある不存在、0件を取得完全、Cache hitを現在許可、string path checkをTOCTOU耐性としない |
| **R2 Studio/AI Patch** | #2745 GNと#2674 SF：元のSource digest/revision・byte span・該当occurrence・期待旧byteを伴う差分Preview。Apply直前にSource/参照先を再読取・再Parse、No-op Saveは0 writeで元byte完全維持 | 旧Previewの自動適用、Markdown書式/CRLF/BOM/コメントの勝手な正規化、AI修正候補＝Owner意味受理を禁止 |
| **R3 Build/Consumer** | §28 V2/V3／#2600／#1163：Source Index/Model Revision/実Build Input/ArtifactSet/外部Consumer actual read/現在Disclosureを独立記録 | 同byte移動・同WorldFP・同出力型から引用/Grant/Receiptや歴史的Owner承認の現在資格を復活させない |

AI一括修正で**対象曖昧、複数Patch衝突、base byte不一致、取得/開示権限不明、必須→任意の意味変更、別Importへ黙示付替え、旧応答後着**があれば、read-only Previewに留める。修正候補にSource path・参照先の秘匿内容を無条件開示しない。既存Studio Writerを再利用し、Apply成功後にも再Parse/Reference依存/必要なMeaning・Artifact・Consumer資格を独立に再検査する。Watcher検知だけでCASや外部編集競合を防げたとは扱わない（#1124／closed #1529）。

### 30.3 実証優先順位とPR/Issueの責務

**P0** 実mainのGo入口・正負Fixture・Source byte/digestと独立Expectedを固定し、#2743 R0をまずfocused Goで確認する。**P1** 実Readerのscope境界と外部Importを同じ版で照合し、partial/denied/offlineの診断を検証。**P2** 実Studio/Source Writerが利用できる範囲だけR2（GN/SF）を実行。**P3** 実生成物/実Consumer/実Provider・Hostが存在する段階だけR3へ進む。未接続なら `NOT RUN: missing concrete entry`、読取対象/完全性が不明なら `unknown`、Owner判断が無ければ `not accepted` と明示してその境界で止める。Sourceの構造的候補と外部業務意味の受理を混ぜない。

責務は #988/#1018（言語・解決）、#1002/#1017（安全なSource読取）、#2743/#958/#2582（外部Reference鮮度・失効）、#2621/#1124/#2497/#2745/#2674（原文保存・診断・AI Patch）、#2516（開示）、#2600（増分再資格化）、#1163（同一AttemptのEvidence）に戻す。**既存RF/GN/SFのCaseを再起票しない**。

**今回確認できた限定事実**：[#2639](https://github.com/hat47x/tei-platform/pull/2639) HEAD `12c864ba83840a7b5750bc8da84af0e8534558e1`について[Repository Health run #41578551459](https://github.com/hat47x/tei-platform/actions/runs/41578551459)と[依存確認 run #41578551497](https://github.com/hat47x/tei-platform/actions/runs/41578551497)が**同一HEADでcompleted/success**。これは限定Go Language Plugin ReferenceのCI成功であり、外部Importの独立Expected/実HTTP・offline、Studio動的診断・AI Patch、外部Consumer/Host、Owner業務意味受理の成功ではない。本追補のR0〜R3とそれら製品試験は**NOT RUN**。既存#2743/#2745はDraftのままで、正式ADR/Schema/実装採用は未決。


## 31. 外部参照修正から業務取消・EKI Attemptまでの統合負例――実対象と資格の一改訂束縛（2026-11-03横断追補）

> **設計・横断受入票のみ／実Source Writer・外部Provider・業務DB・EKI・Studio・SHOの新しいE2EはNOT RUN。** 既存§27（広域品質/EKI委譲）、§28（Value/Flow/Job/Consumer）、§29（構造再利用とAuthority）、§30（外部Importの編集時診断/AI修正）の**独立した担当契約は変更せず**、それらで個別に成立したと誤認しやすい境界を「受注 O1 の取消要求＋外部定義を同じAttemptで使用」という一つの反例束に接続する。主眼はRequestの選択TargetとPayload/実Commit Targetの同一性、修正前後の実Source、Job完了と業務確定の分離である。新たなTEI Host、EKI内の第二業務意味正本、中央状態テーブル、Registry、万能Source Editorは作らない。

### 31.1 一つのActionを複数の資格に分割する

| 資格・対象 | 必ず別に保持する事実 | 最終的に判断する既存Owner |
| --- | --- | --- |
| 外部Reference/Source | Declared locator・必須/任意、解決した物理Target、Source byte digest・revision・occurrence/span・取得時点、fresh/stale/unavailable/partial/not-found/permission-denied、Cacheからの復元か否か | #1017 Read boundary、#1018 Language/Workspace dispatch、#988、#2621、#1529 |
| 定義・意味・編集 | Parse/Validateの診断、候補Model/Role、Accepted Meaning/Authority、元Markdown糖衣・書式の原byte、編集対象の一意性、修正案と適用済みPatch、再Parseで得た意味Diff | #2619/#2620/#2621、#1124、main統合#2674 Source Fidelity |
| 業務Action/状態 | 選択したFunction Context、実対象Entity/Row identity・expected version、input/payload内のTarget ID、現行状態と業務上の遷移可否、Owner受理済Rule | #1129 Lifecycle、#1126 Data/キー、#2622 Provider、#2564 CAS |
| 実行経路と現在の権限 | 選択Provider・DB Alias、principal/grant/operation binding、dry-run/prepareと実commit、transaction境界、部分効果・Commit unknown | #2611/#2598、#1174 Effect、既存Host/Provider |
| 分散Task/資源 | EKI task/attempt/lease、配置先・許可Resource/可用性、worker/結果受信時点、dedupe・timeout・再接続、Execution Receipt | EKI側の資源/実行管理。TEIでは業務Flowの依存とTask結果への参照のみ |
| Consumerに伝える結果 | Resultの実取得とDisclosure、SHO引用/Studio診断のSource revision、Eki Job AcceptedとBusiness State CommittedとBusiness Verifiedの相違、historical/current qualification | #2516、#2605、SHO側Consumer、#1163 Evidence |

**論理的な不変条件**：

- Sourceが存在する・Parserが成功する・AIが修正案を生成する・Roleが受理される、のいずれも、編集書込み許可や業務Command実行資格を付与しない。
- EKIのTask/Attempt/Leaseは**資源・実行の状態**。受注確定/取消、組織所属の有効期間、Workflowの業務状態をEKI Task状態へ移管しない。原則は当該業務データと同じ所有・更新境界に状態属性を置き、履歴/イベントまたはJSON/別表は履歴・可変構造・同時更新等の要件が実在する場合に選ぶ。汎用状態テーブルは前提にしない。
- TEIが停止してもEKIに登録済みのTask/Lease管理は自立し得るが、EKIが完了と報告したことは業務Commit、Receipt確定、Providerにおける一意な副作用、Consumerの正しい開示を保証しない。TEIのGo単一バイナリはEKIを利用せず単体実行できることを基本とする。
- Studioの「参照を表示」「Previewが成功」「保存Intentができた」「Source Patchが保存された」「EKIのジョブが受付けられた」「業務DBのCommitが観測された」「後続の業務検証が成立」は、UI・CLI・AIに**別状態**として示す。unknown/partial/staleを成功へ丸めない。

### 31.2 外部参照からAI修正までの書込み前契約（§30の実行前提を参照）

1. **取得/診断**：ParserはSourceの表記・Spanと宣言を出す。#1017の物理read boundaryは、宣言Rootを越えるsymlinkやTOCTOU、実際に読んだbyteとdigestの同一性を検証対象にする。#1018はLanguage/Version/Implementation pin、実Workspace dispatchを担う。欠落必須ファイルは診断エラー、任意参照は用途別のdegraded/unresolvedを表現する。Resolver/Source IndexのOwnerをParser内で重複実装しない。
2. **状態と影響**：参照先のpath移動で内容hashが同じ場合、Meaningの再受理が不要となることはあるが、Source occurrence/引用・Traceの位置は再解決する。A→B→Aも同じbyteだけで「一度も失効なし」としない。offline Cacheで閲覧できても、現在の実Provider読み取り/命令/開示資格は別。影響不明な依存Closureは新鮮扱いしない。
3. **AI提案**：診断コード/元位置と候補修正を構造化し、Patch対象を (workspace identity, canonical Source path, original byte digest, occurrence/span, Language/Plugin/Role pin, proposed replacement) に束縛。候補が複数、Role未確定、意味変更あり、Workspace外、Providerの編集不能または出典曖昧なら**Preview/保留/read-only**。AIが「修復完了」と宣言しても受理しない。
4. **局所適用**：明示的許可を受けたPatchだけ、書込み直前に物理Target/byte digest/現在のAuthorityを再照合し、重複/交差Spanや同一ファイルに対する他編集競合を拒否。no-op Saveは書込みゼロ・原byte保持、通常SaveでMarkdown表・コメント・CRLF/BOM・フェンス等を全面整形しない。変更後は実ファイルを再読取→reparse→Meaning Diff→必要なOwner再受理。**Patch適用成功はSource修正成功の証拠であり、Business Commitではない。**
5. **消費/再資格化**：診断・修正結果のConsumerは対象Source/Rule/Artifact/引用とDisclosure許可を固定して閲覧。実対象以外は選択的に資格更新するが、未列挙の否定依存・読取0件・依存Plugin改版がある場合、無条件skipしない。#2600のBuild skipと#2582の実行時再評価は区別する。

### 31.3 最小の縦断ケースと独立Expected（§27・§28・§30の横断結合）

試行単位は**「受注 O1 の取消要求」一つ**とし、選択済みFunction Context・対象の明細・Source revision・DB Alias・実Provider/外部Host・EKI Task/Attemptを相互に関連づける。SourceはMarkdown定義と宣言された外部参照（初期はローカル1ファイル）、業務状態は受注行の `status`/`version`（**例示・正式DDLではない**）、取消履歴は業務要件により別イベントとして選択。Expectedは候補実装と独立して固定し、Task受付と業務状態の正しさを別列で採点する。

| ケース | 変更・失敗注入 | 最低限必要な結論／Stop条件 |
| --- | --- | --- |
| **XS-01** | 外部必須定義が存在・一致し、参照/業務State/Actionが正当 | Source→Semantic/Meaning候補→Context→Operation→Provider/Host→Receipt→Consumerまで同じAttempt/版で追跡。実Provider/EKIが未接続ならそこでNOT RUN |
| **XS-02** | 必須参照not-found、optional参照のみoffline/cache | 必須は有効なBuild/実行経路へ昇格しない。optionalは用途・freshness・completeを明示。cacheで見えること≠現在利用可 |
| **XS-03** | 参照先が同byte移動、Source spanだけ変わる | 値/MeaningとSource位置/引用/Traceの現在資格を別判定。必要なConsumerのみ再解決 |
| **XS-04** | 解決直後にsymlink/ファイル差替え、A→B→A | チェック時と実read/write時の同一性が示せなければ拒否/unknown。digest一致だけで履歴的鮮度を仮定しない |
| **XS-05** | AIが古いSource spanを修正、別ウィンドウで先行更新 | write前のdigest/対象実体不一致で競合拒否。元Source保持、他Spanへの自動再ターゲットをしない |
| **XS-06** | Markdown表の1値だけ編集、no-op SaveとExplicit Formatを比較 | 通常Saveは元糖衣・他byte完全保持。Format/Refactorのみ明示の書式変更を許す。Semantic Diffとbyte Diffを別提出 |
| **XS-07** | Operation選択対象O1に対しPayloadがO2、またはDB Aliasが複数 | 暗黙の一意BindingやO1権限のO2への流用を禁止。実対象ID/expected version/Provider routeをCommit直前に束縛。未解決なら拒否 |
| **XS-08** | State=confirmed、取消CAS version競合、Rule denied | 同じData RuleをGUI/CLI/Batchで評価。業務拒否と技術失敗を区別し、EKIの再試行で勝手に状態遷移を再決定しない |
| **XS-09** | EKI worker timeout後にcallback重複、Lease失効後に古いAttemptが完了 | Attempt identity・idempotency/Effect Receiptで同一業務効果を二度適用しない。atomic Provider保証なければCommit unknownと再照合を要求 |
| **XS-10** | EKI受理・Job完了、Provider Commitはunknown/部分成功 | Task complete≠Business committed≠Business verified。無条件retry/取消成功表示を禁止し、既存Provider/Host照会・Recoveryへ委譲 |
| **XS-11** | Task完了後にTEI停止/再接続、参照元Plugin版/Source revision変更 | EKI登録済み実行は独立継続可能。再開時に歴史的Receiptを保持し、現在のMeaning/Scope/Consumer資格だけを再照合 |
| **XS-12** | 実行は全件0件/partialまたはPrimary/Replicaの時点が異なる | 「対象なし」「取消条件成立」「全件検証済み」を、Scope/row+field completeness/epoch/Provider Evidenceなしに確定しない（§18・§26） |

**採点列**は、Source bytes/Revision・Reference status・Meaning/Rule pin・Workspace/Principal・Context/Entity ID・Request Target ID・Payload Target ID・DB alias/Provider・CAS version・Task/Attempt/Lease・actual consumed byte・Effects/Receipt/unknown・Consumer actual read/Disclosure・実行環境/費用とする。PII・secret/非開示の値をログやAI提案本文へ載せず、必要な可観測性と権限制御を同時に検査する。

### 31.4 既存Ownerへの実行順・Go/DSLの採用条件

- **F0：静的/CI前提**。#1017/#1018/#988で実read、Language Plugin選択、Source Index/Spanが一致する最小ケースを確保。#2639は実Go Reference 9テストであるが、Workspace E2E/正式Role受理ではない。#2674のSource Fidelity設計はmain統合済みでも、実Writer/Studio Roundtrip実証ではない。
- **F1：Patchを含む一改訂**。#2621/#1124/#1529のSource Writer/競合処理へXS-03〜06を接続。既存の修正・受理入口が見つからなければ `NOT RUN: missing concrete writer` と記録し、代用の新編集Engineを作らない。
- **F2：業務Commitの一改訂**。#1129/#2622/#2564/#1174でO1取消を同じTarget ID・Version・現行Grant・Tx/Receiptへ結合。O1/O2すり替え、CAS不成立、ACK-loss、部分効果を最初に観測。受注状態の物理保存は基本的に業務行の責務とし、イベント履歴等は要件で選択。現mainのread-only Operation/Artifact PASSを実Commit PASSへ拡張しない。
- **F3：EKIを任意のExecution Adapterとして比較**。ローカル単一Goバイナリ実行と、EKI Task/Attempt/Leaseで同じ業務Input/Expectedを処理する経路を比較する。EKIはクラウド・オンプレ・ローカル計算資源の配置/監視/復旧を持ち得るが、TEI専用Hostや無条件Gateway経由のデータ転送は要求しない。CPU/GPU/VRAMの申告・予約と実測値は別。運用性能（待ち/資源占有/追加roundtrip）を測り、TEIとEKIの結合による無駄なホップや不可避の常駐負荷があれば棄却する。
- **F4：Consumer/再利用の適合**。#1163/#2516/#2600とSHOの別Ownerが、実Reader/引用・ArtifactSet・開示・診断の現在資格を同一改訂で判定。§26の夜間監査/Replica検査結果を現在Primaryの業務Commit保証へ昇格しない。

**採否の留保**：DSLの専門家向け表現はグローバル共通・Go固有宣言を日常表層へ強制しない。Go AST/独自Parser/ハイブリッド/コード生成は同じ可読性・意味/診断/Source保存/生成速度のFixtureで比較し、Go実装親和性のために表層を決定しない。Goの単一バイナリは基本実行形態、EKIは必要時に選ぶ分散実行委譲先、.NET/Java等の既存Hostも保持する。実Writer/Provider/EKI/SHO/第二Targetの成功と、Owner意味受理・新ADR・新Runtime APIの採用はすべて**NOT RUN/未決定**。




### 31.5 実Go接続で観測された「対象・確定・結果伝達」の切断を、XS-07〜10の実証前提へ織り込む（2026-11-03追加）

> **本節は既存§31 XS-07/08/09/10、§34 SQL、§35 RG-06/07/11の採点列と実証順を更新する。** 新しいOperation Status enum／Receipt/Tx/Policy Engine／重複するMacro仕様は追加しない。現行コードの危険な配線を特徴づける局所PASSは安全の受理ではない。独立[test-only Draft #2780](https://github.com/hat47x/tei-platform/pull/2780)と[Macro Owner計画 #2602 §16.4](https://github.com/hat47x/tei-platform/pull/2602)に具体のGoテストと実装判断が属する。

**(1) Selected target → explicit Payload → actual Provider targetの非同値。** main統合[#2773](https://github.com/hat47x/tei-platform/pull/2773)の`commandadapter.Executor.Execute`はCallerが明示した`Payload`を、Command `Inputs`と照合せずRunnerへ渡す。#2780 **MC-B01**の実`command.Invoker→commandadapter.Executor→effectcommit.Runner`・Fake Providerでは、Command選択`ORD-1`なのに`Payload.order=ORD-2`が`WriteResource`まで到達する。これは「不一致を安全に拒否した」ではなく、**現行実装経路で不一致が残る**という局所反証である。Command Target/Grant Resource/Provider Alias/期待Version/Request.Payload/実Write対象/Receipt Subjectを単一の業務Owner受理済み変換で相互照合する。正当な親子Entityへの書込みはあり得るため、**文字列IDを単純比較して一律禁止する方式も不採用**。Mapping/Operation identityが不明なら実Commitへ進まない。

**(2) 二つのUnitOfWork/Tx境界を一つのCommitへ畳まない。** #2780 **MC-C01**は、外側`command.Invoker.UnitOfWork`のCommitを失敗させ、Runner内側Fake TransactionのCommit後に`COMMAND_COMMIT_FAILED`が返っても、内側の合成書込記録が残る挙動を観測。外側Rollbackで内側効果が巻き戻ったことにならない。**実PostgreSQLの二重Tx事故やdurabilityが起きたと断定しない**。Ownerの実Providerで、①同一物理Txに束ねられるか、②別Txなら業務作用/回復はどう分割されるか、③ack-loss/uncertainのReceipt Lookupと独立Readが成立するかを別票にする。read-only/Preview経路で実Commitを要求しない。

**(3) RunnerのOutcomeがCommand Consumerへ伝達されるとは限らない。** 現mainの`commandadapter.Executor`は`StatusCommitted`と`StatusReplayed`をどちらも**空の`command.Result{}`とnil error**へ変換する。#2780追加の**MC-D01**はFakeで初回Effect/Receipt保存と、同OperationIDのreplay（重複Writeなし）を再現し、両方のCommand Resultが空であるとする**Characteristic**。**MC-D02**はFakeのCommit応答喪失に対し、外側`COMMAND_EXECUTION_FAILED`の`error`連鎖内に`CommitUncertainError`が`errors.As`で残るという試験。これは「外部UI/CLIが確定/Replay/不明を区別して表示できる」という証拠ではない。**型付きCauseがGo内部に存在することと、Consumerが観測したDispositionは分離**する。外部APIのResult型・互換性/公開方式はOwner判断として保留するが、少なくともConsumer側に`新規確定／既存Receiptを再確認／確定不明`の異同を隠したまま新OperationIDを再発行・自動再実行してはならない。

**(4) Observer COUNT≥1と試行したEffectの因果性も別。** 現main`internal/execution/observe.go`の`SQLEffectObserver`は、指定Queryの行数が1以上なら`Observed=true`とする。Query/Argsの対象束縛が未指定・不完全なら**別Operation/別Rowでも成立し得る**。同`effects.go:RunEffect`はInvoker成功後のObserver errorで`journal.Append`前に返る経路があり、*業務作用が起きた可能性*と*保存された試行記録*が一致しない場合を考慮する。確認するのは特定OperationID/Target/expected-stateに因果的に結び付く結果とJournal/Receiptであり、`COUNT>=1`や画面ToastをCommit証明にしない。これらは**Source-staticの候補リスク**であり、実DB障害注入の事故再現ではない。

**(5) SQL byte同一でもTarget候補の取得集合が変わる。** [Draft #2778](https://github.com/hat47x/tei-platform/pull/2778)のSM-15隔離SQLiteでは、`NATURAL JOIN`左右の列集合に`payments.status`を追加しただけでSQLは同byteなのに結果が**2→1行**、`SELECT *`は**3→4列**に変わった。暗黙Joinは「存在する列」だけでなく**左右の列名全集合および以前その名前が存在しなかったこと**に依存する。SM-16は実Go`query.DeriveFromAnalysis`＋意図的に固定した合成Analyzer ResultでSchema-resolvedなDerivedIRが同じになり得ることを確認するが、実SQLGlot WorkerのSchema closureが本番で欠落した証拠ではない。**Query候補／Function Context／選択したOperation対象**の型・Rowset・Field/暗黙条件の版を束ね、旧SchemaのTarget選択を現在の実Write/Grantへ昇格しない。無関係列追加で過剰失効させず、`JOIN ON`等の明示依存では必要Closureだけを比較する。

#### XS-07〜10と§35への追加受入列（独立Expectedは既存Ownerが先に固定）

| 結合する既存反例 | 観測すべき一件のEvidence | 不合格／保留の条件 |
| --- | --- | --- |
| **XS-07 + MC-B01 + SM-15/16** | 実選択のContext/RowsetとSchema closure、Ownerが許した`Selected→Payload→Provider Target`変換、Grant Resource、expected Version、actual write target | `ORD-1→ORD-2`に未受理の変換がありながらFake Writeへ進む。SQLは同byteでもSchema依存不明のTargetを旧情報で確定 |
| **XS-08/10 + MC-C01** | 外側UoW/内側Runnerの物理Transaction/Commit主体、同OperationIDとTargetに対するReceipt、independent Read | 内側Commitの可能性を外側Failureで「Rollback確定」と表示。実Txの相関不明なら`Commit unknown` |
| **XS-09/10 + MC-D01/D02** | Runnerの新規確定・Replay・Uncertain、Adapterでの失情報、Go error chain、外部Consumerに実届いたStatus/Receipt | 空Result/汎用Errorのみから、全試行新規成功または全試行失敗と判定し無確認再送 |
| **XS-10 + Effect Observer/Journal** | Query/Args/Rowset/Operation IDの因果対応、ObserverのError/false、Journal Appendの実結果 | 別対象COUNT・Observer error後の記録欠測を「効果なし／観測済」と推測 |
| **§35 RG-06/07/11（任意EKI委譲）** | TEI業務OperationID、EKI Task/Attempt/Lease、内部Retry、実Artifact body、Owner Resultの独立読取 | EKI Task done／Ack／再委譲をBusiness Commit/新規Effectの許可として使う |

**最短の実証順（既存#1119 P0-a/P0-bを変更しない）**：まず #2780 の現HEADをGo formatting→focused tests→head固定Evidenceとして再検証し、MC-B/C/DのCharacterizationを**安全な拒否テストに転換できる条件**をOwnerが明記する。次に#2778 SQLGlot実Producerの固定Schema列集合を#2586 Function/Result→#2598 Target選択へ渡す。そこから#1174の現在Grant/実Provider同一Tx/Receipt、#1129の効果観測・Journal、必要なら§35 EKI内部retry/ACKと外部Consumerへ展開する。**実Providerの一Attemptが欠ければ、別PRのSQL PASS・Fake Go PASSを合算したE2E成功は報告しない**。

**証拠の現時点（2026-11-03、特定HEADに限定）**：#2780の`3b0ead5706f914209c4c76e18ea30aac250cb86b`では[Focused #41620889140](https://github.com/hat47x/tei-platform/actions/runs/41620889140)が**2つのMC-B/C Characterization PASS**。その後の`292ad69ca9eba589498ee071ca8e9392d0c43054`の[Focused #41631728388](https://github.com/hat47x/tei-platform/actions/runs/41631728388)は`gofmt -l`の空チェックで**failure（Go Test未実施）**。続くformat-only修正HEAD **`c9927f247839106c28811e14abba16a53bd38e98`**の[Focused #41632111078](https://github.com/hat47x/tei-platform/actions/runs/41632111078)は**completed/success**で、**MC-B01/MC-C01/MC-D01/MC-D02の実Go限定Characterization 4/4 PASS、gofmt一致、test対象blob固定**をログで確認。これは危険な現行配線・情報消失の反証を検出した成功であり、**拒否/回復を行うProduct修正・実PG同一Tx/Owner意味受理のPASSではない**。#2778 HEAD `1568e00ce1d9fcd7d59ed0f80f1e16173bb4715a`の[Repository Health #41620927639](https://github.com/hat47x/tei-platform/actions/runs/41620927639)はcompleted/success、ただしSQLiteとGo Syntheticは独立局所試行であり、Owner・実PG Schema/Effect/EKI/SHO E2Eを示さない。**本節は現Go Source/CIの再取得＋横断計画追記のみであり、新しいGo/DB Testや製品修正はNOT RUN**。

## 32. 外部モデル再利用の資格――Import成功・意味Pin・Source受理の境界（2026-11-03横断追補）

> **状態：現行mainのGo Source・既存Goテストを照合した設計差分と、次の同一Run受入手順。新テスト実行・新Parser/Resolver/Store・業務Owner受理・正式ADRはNOT RUN/未採用。** §29「参照・合成・限定継承」と§30「外部Import・Studio修正」を、実在する`WorkspaceImport`の能力と停止線へ結び付ける。二つの章の繰返しではなく、**Importで取得できる「意味」と、外部原文まで結ばれた「現在の証拠」の欠落**を扱う。

### 32.1 現行Goで確認できた能力と限界（2026-11-03 main Source-static）

| 実入口（Go） | 既にある振る舞い・局所Goテスト | 保証していないこと |
| --- | --- | --- |
| `internal/workspace/imports.go:WorkspaceImport` | 明示`alias/path/namespace`＋**任意**の`semanticDigest`。選択Namespaceの正規化された意味JSONのdigestを比較し、不一致はLoad時拒否。Rootを越える`../provider`は明示Importとして扱い、Managed Source rootの越境は区別して拒否。 | `semanticDigest`はProvider原文bytes、Workspace manifest、Plugin実装、全推移依存、OwnerのMeaning受理を証明するLockではない。未指定なら開発用の可変ImportとしてLoadできる |
| `loadWorkspaceImports` | Providerが**自分で所有する**NamespaceのみをImport。Providerが別Workspaceから輸入したNamespaceを暗黙re-exportしない。Alias/Namespace重複・循環/所有衝突・一部symlink escapeを既存テストで拒否。Imported Role/ModelをConsumerのManaged Sourcesへ混ぜない。 | Importしたことから相手のBusiness Owner承認、Runtime Plugin、Action Grant、SHO Source ownershipを取得しない |
| `internal/workspace/loader.go:Load` | 外部WorkspaceはProvider自身のLoaderで読み、Consumerの`World.Semantics`へ限定Namespaceを組み込む。Consumerが管理するSource bytes/Manifest identityは別。 | Import Providerを含む**単一Atomic Source snapshot**、任意Remote/HTTP Reference全般の鮮度や現在Authority |
| `internal/workspace/source_index.go:BuildSourceIndex` | 通常のIndex経路はConsumerのManaged Sourceを扱える。Imported ProviderのSource位置/byteをConsumer所有のSourceとして捏造しない。 | Imported Sourceまで含む厳密なAccepted Source証拠 |
| `BuildSourceIndexFromLoad` | **Import宣言を含むWorkspaceでは`ErrImportsNotBound`としてfail-closed**。実コードと`TestBuildSourceIndexFromLoadRefusesWorkspaceWithImports`で明示されている。 | 現在の`SourceIndex`にProvider bytesを安全に取り込めたとは報告しない。Import済み`World`を一式受理済みと見なすことはできない |
| `Result.ManifestIdentity` | `Load`時に読んだmanifestのPath/Digestを別保持。 | ManifestがSource Index Artifactに含まれたということ、別時点のImport Redirectを防いだこと |

**実証の出典（既存mainコードとテスト名）**：
- [`internal/workspace/imports.go`](https://github.com/hat47x/tei-platform/blob/main/internal/workspace/imports.go)、[`import_digest_pin_test.go`](https://github.com/hat47x/tei-platform/blob/main/internal/workspace/import_digest_pin_test.go)の`TestExternalSemanticImportOptionalDigestPin`：同じNamespaceのSource移動と無関係Namespace追加では意味Pin維持、意味定義変更はPin拒否、Pin削除後は開発用Import再許容。
- [`imports_test.go`](https://github.com/hat47x/tei-platform/blob/main/internal/workspace/imports_test.go)の`TestExternalNamespaceImportPreservesProviderIdentityAndSourceOwnership`：Imported NamespaceをConsumer Sourcesと混同せず、Plugin実装も自動Importしない。
- [`source_index_load_bound_hardening_test.go`](https://github.com/hat47x/tei-platform/blob/main/internal/workspace/source_index_load_bound_hardening_test.go)の`TestBuildSourceIndexFromLoadRefusesWorkspaceWithImports`と[`source_index.go`](https://github.com/hat47x/tei-platform/blob/main/internal/workspace/source_index.go)：Import時のStrict Index拒否、およびSource IndexとManifest identityの別保証。

**現行の区分**：`Load/Validate`で選択Namespaceを解釈できたという局所成功、`semanticDigest`により**当該Normalize済みNamespaceだけ**が一致したという成功、Import元/先のSource/Owner/Consumerの完全証拠は、**互いに異なるClaim**。特にStrict Index拒否を現行の誤動作と決め付けない（未束縛Sourceを安全に成功としない既存方針）。

### 32.2 三Domainの同一Subject実験（正式Grammarは増やさない）

一つの受注側Consumerが`org-model`から`organization.membership`の型をImportし、受注承認のRuleがそのMembershipを参照、Workflowの現在状態は受注本体`approvalStatus`へBindingする例を比較する。

```text
Provider Workspace "org-model" --owns--> organization (Person, Membership, Position)
  | explicitly import alias=org; namespace=organization; optional semanticDigest
  v
Consumer Workspace "order-entry"
  --owns--> sales.order / Data Script / current approvalStatus / Mapping
  --ref--> organization.membership(occurrence + valid/recorded time)
  --separate--> Business responsibility / Delegation / current Host Grant
```

- **Reference**：Providerから`Membership`の型を使用できても、Consumerがその定義・元Source・Owner承認を所有するようにはならない。Role/OperatorのEager依存はImport pinと別に固定する。
- **Composition**：Consumer側の`Order→Membership`関係・取得Scope/時点、`approvalStatus`保存Binding、Grant照合を明示する。Imported Personに実承認Authorityを追加する「型継承」はしない。
- **限定的Refinement/Inheritance**：`ApprovedMember`が親`Membership`として利用できるかは、親Contractの入力・出力・多重度・時点・NULL/partial・Effectを壊さないことを独立Expectedで確認する。強化した承認Guardが親利用者を拒否するなら一般的置換可能性を棄却し、用途限定Compositionとして明示する。
- **物理**：本体`approvalStatus`は第一候補だが、複数承認Attempt/並行Token/履歴が必要ならSidecar等を選び、現在State Writerを一つに保つ。Import ProviderのScope/Key/Role/Storageを自動上書きしない。

### 32.3 外部再利用の最小反証対照（MRUと既存Fixtureを**結ぶだけ**）

| 対照 | 一軸変更 | 要求する結論・停止点 |
| --- | --- | --- |
| **I0 Baseline** | Provider/Consumer固定、明示Namespace、Pin一致 | `Load+validation`の局所成立は候補。Strict Source IndexはImport時`ErrImportsNotBound`なので、正式な跨Workspace Acceptanceを主張しない |
| **I1 Source Relocation** | Providerが当該Namespaceを別ファイルへ移動、正規化意味同一 | Semantic Pin維持候補。一方、Provider Source occurrence/SHO引用、Manifest/Read現在性は別確認（§29 MRU-10） |
| **I2 Same-name different meaning** | 同一Namespace ID/型名でもField単位/FTE意味・Data Ruleを変更 | Pin一致を期待しない。UnpinnedでLoadできてもOwner意味受理/Function互換まで自動成立しない（MRU-01/02） |
| **I3 Transitive Role/Source** | AがBから輸入したNamespaceをCがA経由で使おうとする／Eager Role実装Pinのみ変更 | 非所有Namespaceの暗黙re-exportは拒否。Normalized Namespace digestだけで推移Role/Plugin Pinが成立したとしない（MRU-05/06） |
| **I4 Manifest redirect** | Consumerのimport pathだけ別Providerへ変更し、選択Namespaceの意味digestは同値 | Semantic候補は同じでもManifestIdentity/供給元・Authority・Source provenance差を検出。以前の受理・実Reader資格を自動転用しない |
| **I5 Partial/offline** | 明示必須Importが一時unavailable／permission denied／partial、Cacheだけ残る | 現在Reader資格は`unknown/unavailable`。旧Cached Meaningを現行Accept/Operationへ昇格しない。実装入口がないHTTP/Network状態は#2743設計・未実行と記録 |
| **I6 Grant-only** | Source/Meaning/Artifact不変、Consumer Principal/Host Grantだけ失効 | 過去のValue/Artifactを保有しても現在のAction/Disclosureを再確認（MRU-03/12） |
| **I7 Full vs Selective** | 無関係Provider Namespace追加、利用中のAlias別Import更新、0→1→0 Relation変更を比較 | Role/Meaning/Source/Query/Artifact/Disclosureの依存が明示された範囲のみ選択的再評価。それ以外は`unverified`、誤再利用も無用な全再Buildも測定 |

**誤った二択を避ける**：`semanticDigest`の一致を理由にすべてを「完全に固定」とするのも、ProviderのSource移動だけで全Consumerを「再生成必須」とするのも誤り。型/Role/Source/Provider/Authority/Consumer別の再資格化を分ける。未観測のProvider version、変換される単位、下流に現れないRule/Predicate依存は既存#2582/#2600の対象。

### 32.4 Strict Acceptanceの不足を解消する設計の比較（採用は保留）

**候補A：Provider側の固定Source・意味受理証拠を委譲参照**。Provider所有のLoad時Source bytes/Manifest identity/選択Namespace/推移Pinを、実Audience・版・Ownerが許したClaimとしてConsumerが参照する。Sourceの所有権をConsumerへ移さず、現行Trusted Host `sourceacceptance.Reader`のAuthorityと矛盾しないよう現在性・失効を別に照合。**導入判断は#1163/#1017/#2743／Owner**。

**候補B：明示的に固定されたVendor/BundleのImport Snapshot**。Provider原文＋manifest＋意味/Plugin依存の必要閉包を一つの実取得Bundleで固定し、Consumerに同梱できるか評価。単なる`semanticDigest`だけではByte/原出典と実ロードを保証しない。Bundleに入ったことが供給元の意味・権限受理の移管を意味しない。Bundleが大きすぎるなら目的相対のPackage closureを比較。

**候補C：今のPreview/Read-only局所Importのまま**。Providerで未束縛なSourceを含む場合、説明・候補表示を行えても、厳密Source Acceptance/生成/公開/実行の資格はそこで停止する。利用価値が十分ならA/Bを急いで製品化しない。

どの案も第二Semantic Canonical、global Model Registry、必須中央Store、TEI独自Host、新認可Engineを要求しない。外部HTTP/Cloud Importを既に実装したという主張もしない。

### 32.5 既存テストの次の実行順・責務

1. **X0 exact-head Go再実行**：main上の既存`TestExternalSemanticImportOptionalDigestPin`、`TestExternalNamespaceImportPreservesProviderIdentityAndSourceOwnership`、`TestBuildSourceIndexFromLoadRefusesWorkspaceWithImports`を`go test ./internal/workspace -run '^(...)$' -count=1`で実行。固定HEAD、Go版、Source blob、テスト実ログを記録。**本追補では実行していない**。
2. **X1 一Consumer二Provider・経路変更**：同じWorkspace Source/Meaningで`alias+path+namespace`、Role/Provider契約・ManifestIdentityを比較。旧/newどちらをOwnerが受理したかは明示的な実証拠が必要。Existing Loaderの局所テストを超える部分は`NOT RUN`。
3. **X2 Strict provenance**：A/Bの設計を#1163のSource資格/Provider SourceReader/Scope/Ownerへ渡し、現在の`ErrImportsNotBound`を仮に解除せず、**先に必要な供給元Byte・閉包・実Readerの実証を定義**。A/B未成立ならCで停止。
4. **X3 Consumerと選択的再生成**：Source移動/Role変更/Import path変更/Grant失効の同じ正負対照を、Go CLI Operation/Artifact、SHO引用、AI Skill候補・Provider実Readの各Ownerで評価。現実のHost/SHO/Runtimeが接続できない工程は未実行とし、独立Claimを保つ。

**現在の変更・合否**：本§31は計画文書のみを変更する。既存GoテストがSourceに存在することは確認したが、この回答での`go test`・Focused CI・実Provider/SHO/Host、厳密な跨Workspace Acceptanceは**NOT RUN／未確認**。§29 MRU-01〜12と§30 Source/AI修正のOwner分担を維持し、Issue/ADR本文とOwner正式判断は変更しない。

## 33. Organization E1からRelation-first実Data Ops/Contextへの橋渡し（2026-11-03、#2386/#2586の採用前横断実証）

### 33.1 現物から確定できる境界

- [Org Native PR #2681](https://github.com/hat47x/tei-platform/pull/2681)の`semantic/organization/model.tei`は**Person、OrgUnit、Membership、Position、Occupancyを直接Dataとして宣言**し、Relationのtarget/keyを明示。`persist`は別物理Mapping。固定HEAD `8efe0d351a244b57fefd00b498a5b47136619ac5`で[Actions #41580077197](https://github.com/hat47x/tei-platform/actions/runs/41580077197)が実Go Loader/CLI/Operations/Artifacts PASS。これはE1であり、実SQL/Provider/Context Readではない。
- main `internal/operations/surface.go`の`SurfaceWorkspace`は`workspace.Load`＋Source Index→`DeriveOperationSurface`、`effect=read-only`で資格を限定する。main `internal/query/model.go`のQueryは`EmbeddedSQL`であり、`syntax/query_authoring.go`がnative SQL body/fileを受け、`derived.go`が解析結果をDerivedIRへ変換する。**現行SQLの実装とTEI標準Relationの取得/Traversal実装を同一視しない**。
- main `internal/applicationview/browse.go`の`BuildBrowse`はSQL結果列MappingからBrowse列/Filter等を構築する**表示用の投影**。[merged #2679](https://github.com/hat47x/tei-platform/pull/2679)によるQuery Input⇔Browse Filterの型付き対応は有効だが、物理Read/実Queryの実行証拠ではない。
- [Org E0 PR #2779](https://github.com/hat47x/tei-platform/pull/2779)の[Node Actions #41607867473](https://github.com/hat47x/tei-platform/actions/runs/41607867473)は固定HEADで`verify.mjs`27件、Node225件がPASS。E0合成とE1 Nativeが別々にPASSしても、同一World/Query/Provider/Consumer runの成立とは言えない。

### 33.2 QueryとData Operationsの責務を分離し、可読性の優先順位を保存する

Ownerの設計方針：Data ModelはSQL解釈から構造生成するのでなく、Data/Field/Key/Relationを**直接宣言**する。業務利用者が通常の関連データを取得する際は、宣言済みRelation pathから標準Data Operationsを導出し、**Functionが使用する範囲を明示Mapping**。Studioは候補/補完を提供するが、非宣言外部Relation/Fieldを無制限に自動Fetchしない。Native SQL/TEI Queryは複雑な変換やDB固有の特殊処理への補助経路であり、普通のReadのためにSQL記述を事実上必須にしてはならない。Go Parser/AST親和性は内部実装上の選択肢で、DSLの普段のスクリプトへ`struct/func/import/package`を露出させない。

- **#1126** は直接宣言Data/Relation/業務Key/物理PK意味契約。#988/#2620は言語/Roleと正本。#2622/#1127は既存Data Ops導出とProvider適用。#2506はFunction明示Data Context。#2387はQuery専用構文/非表示依存/SQL解析のOwner。#2611/#2628は実物理Provider/Read Witness。#2586は**この隣接成果物の一固定Attempt接合のみ**。
- 採用前候補の通常Query Pathは`Semantic Data/Relation → Explicit Function Mapping → Data Operations/Provider-specific realization → actual Result Slice/Read Witness → Claim/Pattern`。SQL経路は`Query SQL Source + Dialect/Schema/Worker revision → DerivedIR → Provider execution → Result Slice`で、SQLのFieldIDはSemantic IDでないため、明示結果Mappingがないまま再利用しない。どちらも現状全経路実測済みではない。
- 資格も用途別に：**candidate**, **declared**, **parsed**, **bound**, **physically observed**, **consumer qualified**, **currently action-authorized**, **Effect observed**を同じstatusで表現しない。現在一部入力が未観測でも普通のDataのL0一覧は利用可能だが、FTE/Rule/External Disclosure/Writeは用途相対の前提を別に確認する。

### 33.3 E2の最小受入試行を同一組織Source/Fixtureに束ねる

[組織設計PR #2389 §56](https://github.com/hat47x/tei-platform/pull/2389)へORG-E2-01〜10を登録した。#2681実`.tei`はSchema/Mapping/OperationSurfaceのみ、A1〜A4実Providerデータは未登録。合成`org-synthetic-r7`とのID/FTE一致は**期待値**であって実DBの結果ではない。

| 受入列 | 最低の正例 | 必須負例・未検証 |
| --- | --- | --- |
| Authoring/Meaning | 直接宣言MembershipがPersonとOrgUnitをRelation target/keyで参照、Member occurrence IDは独立 | Person IDだけからA2/A3を同一化しない、Occupancy⇒Membershipとしない |
| Function Request | `org_view`がMember/Person/OrgUnit必要集合とallocation Fieldを明示 | undeclared Relation/Fieldの実Read、未知Sourceの暗黙Fetch |
| Physical Read | 一固定DB/Route/Revision/cutでA1〜A4とValue/NULL/未取得/部分ReadのWitness | read-only OperationSurfaceの存在・同一HTML出力で実Readしたと主張 |
| Claim/Measure | 4 Membership・3 Person・全FTE3.0、D1=1.6/D2=1.4を独立Expectedと照合 | Join 6行/Field欠損/partial readから全体count/FTE/Capacityを確定 |
| Query/control dependencies | WHERE/ORDER/EXISTS/NOT EXISTSの非表示入力とRole Pin/Rule版を追跡 | 同じ表示値・同じsnapshot名だけで同値/未失効と判定 |
| Current Consumer/Action | Baseline/Proposal/R1公開版を別Projectionにし旧A2の対象を保持 | what-ifをaccepted/authorized/committed、CandidateをGrantと見なす |

**実装計画**：まず既存mainのData Ops/Provider入口が本当にRelation Traversalできるか調査し、なければ`NOT RUN: relation-path-producer-unavailable`として#2622へ責務を戻す。実RDBの固定データ/取得権限・read witnessがなければ`NOT RUN: provider-unavailable`、実SQL Workerなら`NOT RUN: sql-worker-unavailable`。真偽や「0件」で埋めない。特に#1119の実PostgreSQLとOwner-write P0-a/P0-bの優先度は変更しない。

**本節の状態**：GitHubの固定Go/Node CI、実Go Source/SQL/Presentation Producerの静的読取に基づく**採用前の受入計画**。本節自体で新Go/Query/DB/Studio/SHO/Host E2Eを実行したわけではない。第二Canonical/Query/Host/ACL・意味Registry・巨大Context DTOは作らない。


## 34. SQL/DDLの静的事実と組織what-ifの時間・責任を接合する――E1の先で止める条件（2026-11-03追補）

### 34.1 現行の二つのReferenceと一つの設計案を異なる証拠として扱う

- **E1 Native TEI**：[#2681](https://github.com/hat47x/tei-platform/pull/2681)の`reference/org-headless-workspace`はPerson/OrgUnit/Membership/Position/Occupancyを独立IDとして宣言し、MembershipのPerson・Unit Relationと`public.org_memberships`へのPersistence mappingを持つ。宣言上の物理Keyは`primary-key organization.membership.membership_id`。allocation/validFrom/knownFromはoptional。実Go/CLIの限定CI [#41580077197](https://github.com/hat47x/tei-platform/actions/runs/41580077197)は当該PR固定HEADでsuccess。**Providerによる実行、Postgres上でのPK/FK制約、期間非重複、Migration、DB書込みは未検証**。
- **E0 Synthetic Scenario**：[#2779](https://github.com/hat47x/tei-platform/pull/2779)の`org-synthetic-r7`に対する独立copy-on-write組織改編20ケースは、固定HEADの[Node CI #41607867473](https://github.com/hat47x/tei-platform/actions/runs/41607867473)でsuccess。試行は`validAt=2027-06-15`、`knownAt=2027-07-01`、Capacity Rule版が固定された**合成世界**。#2681の`.tei`を実Queryで読んだ値ではない。
- **SQL static / candidate / observed / accepted**：[#2778](https://github.com/hat47x/tei-platform/pull/2778)のS/C/O/A区分を#1126/#1129/#2622/#2387から再利用する。SはSource/Dialectに書かれた構文・制約宣言、CはTEIのKey/Relation/State/Ruleへの対応候補、Oは実DB Providerの時点とScope付き観測、AはOwnerの意味受理。**OとAは上下関係ではない**。静的SQL解釈、合成Scenario、Owner受理、実Tx完了を一つのPASSへまとめない。

### 34.2 提案が総量を保っても業務・物理制約の不変を意味しない

合成BaselineのMembership occurrence A1=P1/D1/100、A2=P2/D1/60、A3=P2/D2/40、A4=P3/D2/100（100 points=1 FTE）を使い、**A2 retire、A3 40→100**を提案する。合計300 points/3 Personsは不変でも、D2は140→200となり、**合成Capacity160を40超過**する。P2のOccupancy、Q1のCase責任、Grantを自動継承・移転しない。D1分割でA1new/A2newを設けた場合は旧D1のCapacity・権限が後継に引き継がれる根拠がなく、**unverified**を保持する。

この反例をSQL・実DBへ移す前に、以下を必ず別に固定する。

| 接続点 | 正当な候補 | 無条件に成立しない事項 |
| --- | --- | --- |
| **Identity／Physical PK** | Native TEIの`membership_id`独立OccurrenceとProviderの明示Key Mapping | 仮の`PRIMARY KEY(tenant_id,membership_id)`やDDL列順からTEIの業務Key、SortKey、Occurrence同一性を推測しない。Tenant Scopeは別途選択 |
| **Temporal State** | `validFrom`／`knownFrom`、選択`validAt`／`knownAt`、ProviderのSnapshot/履歴行とOwnerの時点ルール | Native Sourceに終了時点/重複排除・訂正規則がないのに、完全なbitemporal履歴や同一時点の一意行を証明したとしない |
| **Relation／Capacity** | Membership occurrence、Person・Unit、Occupancy、Caseを独立に取得し、Rule版を固定 | SQL FK/JOINの存在から責任・承認権限、兼務の集約粒度、Capacity RuleのOwner受理を推測しない |
| **Read／Query** | Functionが明示したRelation path、Field/Row/Predicateの完全性、AliasとRead Cohort、独立Expected | `NATURAL JOIN`のSchema列追加で意味が変化、Join多重化、非表示WHERE/NOT EXISTS、部分0件を現在の全件事実に昇格しない |
| **Proposal→Apply** | Source/Rule/Read版を固定したScenario Diff→Owner Decision→Hostの現在Grant/CAS→実Commit/Receipt/独立Read | copy-on-write PreviewやStudio参照実行`Committed=true`をDB Applyと見なさない。EKI Jobの`AcceptedResult`もBusiness Commitではない |

**状態保存**は受注・所属・Workflowの用途によって本体列／期間Relation／Instance／履歴を選び、SQLの`status`や`CHECK IN(...)`だけで遷移・Ownerの意味正本を作らない。#2778 SC反例と#2389の現行/提案/公表を再利用し、第二汎用状態テーブル／第二SQL Engineは起票しない。

### 34.3 Doneの最短経路と独立停止位置

**P0**：#2681の実`.tei` Loader/Operation E1と#2779合成TSを**別Run・別Fixture**として採取し、相互に値やIdentityが一致するとはまだ主張しない。**P1**：#2778 S/CでKey・Relation・Temporal・NullableをSource span付きの候補として照合し、Ownerへ有効/認知時点と責任移転・重複所属の意味を質問。SQLGlot/Workerや実DDL Readerが無ければ`NOT RUN: producer-unavailable`。**P2**：#2586/#2506/#2622で、同一Rule/Subject/validAt/knownAt・必要Field・Relation/Query Scopeを実Providerから取得し、complete/partial/permission-deniedを独立に観測。実Provider未接続なら`NOT RUN: provider-unavailable`。**P3**：その実Readに対してのみ改編Overlayを計算し、合成Capacity超過と後継Capacity未観測を独立Expectedで反証する。**P4**：Owner Decisionと実Action/Commit/Receipt/独立Readを#1119 P0-a/P0-b/#1129/#1174へ委譲。EKIとSHO/Studioへの配布は必要になった場合に限り、Current/Proposal/Accepted/Applied/Verifiedを混ぜず開示・引用資格を確認する。

**優先しないもの**：新SQL/DSL Grammar、全Domain共通State Engine、組織専用Core／ACL、EKIの業務意味正本化、実DBや多Providerの同期Snapshotを暗黙に仮定する一体Runner。既存§33のE2 Relation-first経路を満たさずSQLや合成結果だけからE3/E4へ進めない。

**今回の実証状態**：既存の#2681実Go/CLI限定CI・#2779独立Node合成CIの成功と#2778設計書の静的確認に基づく**Docs-only追補**。新しいSQL Worker／実DB Migration／Provider Query／業務Owner受理／改編PreviewのTEI実装／Effect／Studio・SHO・EKI連携は本節ではNOT RUN。本文§31〜33と各Owner Issueの担当範囲は変更しない。


## 35. TEIジョブ構築とEKI広域資源グループ委譲――低信頼端末・HPC・経済圏の責任境界（2026-11-03横断追補）

> **採用済みの責務分離を具体化する設計・受入計画。実EKI Resource Provider、実TEI Job Builder、HPC/PC管理系、組織横断FederationのE2EはNOT RUN。** 別チャットの「AIジョブの構成・意味はTEI、EKIは委任の核とジョブコントローラー」「EKIはHPCの内部ノードを管理せずクラスタ群を一資源として見る」「社内遊休PCも個体でなく管理済みの資源群として扱う」を、既存[EKI ADR-0013](https://github.com/hat47x/eki-fabric/blob/main/plan/adr/0013-product-core-and-faces.md)の採用済み委任の核、[EKI #67](https://github.com/hat47x/eki-fabric/issues/67)管理クラスタProvider 2系統、[EKI #115](https://github.com/hat47x/eki-fabric/issues/115)委譲先内部再実行、closed [EKI #109](https://github.com/hat47x/eki-fabric/issues/109)資源プール草案、closed [EKI #140](https://github.com/hat47x/eki-fabric/issues/140)広域最適化と照合する。closed issueは採否未確認の設計まで自動的に製品実装済みと呼ばない。§27/28/31のJobとBusiness Effectの分離を維持する。

### 35.1 管理粒度と資格の所有を分離する

| 粒度・対象 | 主な責務と不変条件 | 委譲しない責務 |
| --- | --- | --- |
| **TEI Function/Job Definition** | 入力・出力、演算/DAG、意味のPin、実消費Source/Artifact、資源要件、Data Class/配置制約、費用/期限、効果の有無・再試行可否、独立Expectedと検証条件を構築する。非画面Batch/AI/HPCが同じ明示Contractから導出される | TEIは実物理容量・現在のCompute Grant・EKI Leaseを発行しない。Job DAGをEKIの業務意味正本へ渡さない |
| **EKI Delegation/Job Control** | 許可されたTask/Attempt/Lease、Admission、Budget、Resource Group選択、結果Claim・検証・Recovery、時点つき観測と来歴を執行する。EKI自身の配置計画は利用者の業務判断ではない | DSL・業務Rule・会計処理・HPC内部Scheduler・PC MDM/EDR・TEI独自Hostを持たない |
| **Organization/Resource Provider** | 利用可能な資源Group、管理主体・所有/提供範囲、契約/Policy版、上限・時点・適格性、予約/取消・要求受理結果をEKIへ開示する。EKIが実観測できない値はProvider claimと記録 | 管理主体による申告だけでWorker Assurance/Result真実性を上げない。組織所属のTEI MeaningからCompute Grantを自動発行しない |
| **Execution Backend** | HPC Slurm/Kubernetes等の既存クラスタ、クラウドAPI、MDM/端末管理・ローカルGovernorが内部配置/ノード回復/強制停止を実施する。Adapterは外部Job handleとEKI Attemptを対応付ける | EKIが内部ノードを列挙して再スケジューリングしない。Provider内部retriesを隠して一回実行保証と報告しない |
| **Host/DB Effect/Consumer** | 既存Host・DBが現在の権限、対象Identity/CAS/Tx/Commit/Receiptを実施。Consumerが実Artifact bytes・出典・Disclosureを確認 | EKI `completed` を業務CommitやData Quality PASSへ昇格させない |

**「クラスタ群を一資源」の意味**は、EKIが**管理・予算・配置対象としてGroupを一単位に選ぶ**こと。Group内CPU/GPUの合算が常時予約可能、GPU互換性が均一、任意のノードが要求Data Classを満たす、内側の再試行が1回だけ、という主張ではない。直接Workerを扱う既存EKI実装は残せるが、外部管理されたGroupを扱うときに全ノードの所有権と配置権限を重ねない。Worker/Resource Pool/Compute Grant Provider/Execution Backend/TEI Contextを同じIDに集約しない。

### 35.2 資源群の最小受渡し票（正式Schema/ABIを固定しない）

- **Resource Group identity**：Group/Provider/管理主体/法的Ownerの独立IDと時点つき所属・契約版。親子Organization構造や経済圏所属はTEIの業務Dataとして参照できるが、**所属変更が自動でEKI grant・請求先・信頼等級を変えるとは扱わない**。管理者が実権限を発行・更新した版を別に照合する（TEI #2386/#2499、EKI RQ-047）。
- **Capability envelope**：CPU arch/ISA、メモリ・GPU製品/VRAM・共有制約、NPU、OS/runtime/toolchain、Network/転送時間、Data localityと出力保持、同時実行数/時間・容量、cost/power、Assurance/Data Class。**advertised/eligible/reservable/actually allocated/observed** の段階と時刻を分離し、「GPU総数」だけで型付きAI Jobの適格性を決めない。異質なGroupなら、要求を満たす子集合へ下位Backendが確実に絞れる証拠がない限り、同質Groupへ分割または`unknown/unsupported`。
- **Availability/Reservation**：予測余力/観測余力/仮予約/Provider受理/Lease確定/開始/終了/回収の段階、observedAt/effectiveAt/validUntil・TTL/epoch・Budget枠・quota holderを区別。PC群は端末稼働/持ち主復帰で容量が変動するため、集約予測を保証枠と呼ばない。外部Providerが枠を保証できない場合は *conditional admission* と現在の失効を明示する。
- **Trust boundary**：Provider信頼、実ノード適格性、Data Classを処理してよい根拠、コード隔離/測定、Result verification、Telemetry信頼度を別軸にする。管理Group内に低信頼端末が混在する場合、最も信頼できる端末を根拠にGroup全体を高信頼としない。Providerが実選択先を制約・監査できないなら、そのAssuranceを要求するTaskは拒否。
- **Attempt/Artifact chain**：TEI Source/Job digest・actual loader/Executor digest、入力Artifactの取得と内容、EKI Task/Attempt/Lease/外部Job handle、Provider内部retry/横取り、結果Claimと実Artifact body/digestの独立確認、業務Effect Receipt、Consumer actual readを別に束縛。Task `completed` やstdout digestだけでは成果物本体を取得・検証した証拠にならない（EKI #112/#320/#494）。
- **Disclosure**：利用者・Group管理者・Fabric管理者で集約統計の粒度を制限し、Node/hostname/IP/所在地・他部門の余剰容量・内部価格を推測させない。TEI StudioのGraph/表/シミュレーションも与えられた可視範囲のみ。Source/Modelの公開可能性と実計算資源の照会権限は別。

### 35.3 ジョブ構成と委譲方式：二段階の最適化を混同しない

**TEI側**は専門家が読めるFunction/Data/Ruleから、構造化したJob Planとテスト・Expectedを生成する。Go単一バイナリ（基本）/既存Host内実行/任意EKI委譲の3経路で、同じ業務意味を再解釈せずに使えることが条件。AI Jobで複数Stageを生成しても、各Stageの必要Input/Output/依存/Effect/復旧条件はTEIが決め、EKIに「次の業務ステップを判断」させない。

**EKI側**はTask Admission→許可されたResource Group選択（deadline/budget/data locality/trust/fair share）→期限付き委譲→状態と結果を観測。外部管理HPCでは、EKIのPlacementは*クラスタグループ選択まで*、内部ノードの配置/再配置は既存Backend。低信頼PC群では、EKIは端末管理系が認めるPoolへ仕事を渡し、PC個体の運用/パッチ/出勤時優先と停止はMDM/Agent/Local Governor側。外側EKI Lease失効時の下位Job取消・結果の遅着・残留Effectは返却/回復契約で別に観測する。

**組織・経済圏の制御**は上位のPolicy/Compute Grant・予算・公平性・Data Class/法域・提供可能時間帯の*統治・調達*であり、HPCノード詳細やTEI業務状態の混在ではない。SEI等の意思決定はEKI ADR-0013 §2決定10の**管理者からの範囲・期限付き委任**に従い、TEIのOrganization RoleをEKI管理権限へ自動変換しない。ProviderのquotaとEKIのbudget/fair-shareは独立に課せられ、二重計上・両者不一致の挙動を明示する。クラウド拡張は金銭コストを伴う別の許可・二重上限に従い、自動決定=費用発生の無制限許可ではない。

### 35.4 外部提供先の失敗を見分ける最小Fixture（RG-01〜12、未実行）

| ID | 入力/反例 | 独立Expected / Stop条件 |
| --- | --- | --- |
| **RG-01 HPC集約** | 既存管理クラスタが内部64ノード、EKIにはGroup-Aとして公開 | EKIの外部PlacementはGroup-A。内部ノードは識別・割当不要だが、外部Job handle/Reserve/Resultの一致、実行保証可能範囲を受渡す |
| **RG-02 異種GPU** | Group-Aの一部のみVRAM 80GB、Job要求70GB＋特定ISA | 合計VRAMやGroup最大値を適格性にしない。適格子集合へ制約可能なProviderだけ受理、無理ならGroup分割/unsupported |
| **RG-03 PC群急減** | 勤務開始時にPool-Bの申告余力100→10、既存JobにLease | forecast/observed/reserved/activeを別記。Local opt-out優先、再委任可否・部分実行とTask/業務Effectを別判断 |
| **RG-04 Provider障害** | Group-Aの監視が10分途絶、最後の空き枠申告は30 | staleを現在空き30として採用しない。unavailableとcapacity=0も区別。期限/再取得/停止条件を明示 |
| **RG-05 Assurance偽装** | Pool-Bの一部が非管理端末、AdapterはGroup高信頼と報告 | worker自己申告/平均信頼では不許可。管理者根拠と実制約能力が足りなければ高Data Class Taskを拒否 |
| **RG-06 内側再試行** | HPC Backendが同じ外部Jobを内部で自動再実行、TEI Taskは副作用あり | EKI #115に従い無効化・観測・拒否を判別。内側回数不明ならexactly onceとしない。EffectはHost/Provider側で別照合 |
| **RG-07 取消/遅着** | EKI Lease期限切れ、下位クラスタ取消ACK、遅れてResult | 取消ACK≠処理停止証明、Effectゼロ証明ではない。fence/Job handleとTask/Attempt、結果検証を比較 |
| **RG-08 組織異動/提供停止** | 部門を跨ぎ所有Groupが異動、契約/Compute Grantは旧版 | TEI Organization意味変更≠EKI管理Grant変更。新Policy/Owner許可なく利用可能Group拡張しない |
| **RG-09 二重計上** | 同一GPU資源がGroup-AとProvider-Cの両方に含まれる | EKIの総容量を二重加算しない。disjointnessの証拠が不十分なら保証枠を下げる/unknown |
| **RG-10 所在・転送費用** | Data 800GBは社内、遠方クラウドは速いが転送制限 | Bandwidth/Data Class/料金・転送時間/予算の許可とDeadlineを比較。最短CPU時間だけで選定しない |
| **RG-11 Resultの本体欠落** | EKI ResultにSHA-256はあるがArtifact body取得不能 | ResultClaim/Task phaseを記録してもTEI Output/Consumer同一性は未検証。独立Readerでbytes確認するまではunknown |
| **RG-12 直接実行との比較** | 同一TEI JobをGo単体、EKI Native Worker、EKI→外部Poolで比較 | Meaning/Expectedを同じに固定し、追加ホップ/待ち/実費/再試行回数/漏洩範囲と成功率を別計測。EKIが不要なJobへ一律委譲しない |

### 35.5 実装順・判断を保留する条件

- **G0 固定Fixture/契約差分**：まずEKI ADR-0013（採用済み）・open #67/#115/#112の現行契約と、TEI #1130/#2499/#2514/#1163のJob/Authority/Expectedを比較。closed #109/#140の構想を契約実装とみなさず、RG-01〜12のExpectedを別根拠で固定。既存EKIのTask/Lease APIをTEI DSLに埋め込まない。
- **G1 最小二Provider read-only**：管理HPCと、端末管理系に集約された低信頼PC群またはクラウドGroupの**性質の異なる二系統**について、advertised capability/実制約可能なsubset/現在Reserve/内部retry/観測・開示を比較。EKI #67の2系統実接続に乗せ、TEI独自のHPC管理を作らない。現行API/実Providerが未到達なら `NOT RUN: concrete backend unavailable`。
- **G2 実Job受渡し**：TEIで固定した一件のread-only Job/Executor digest・Input Artifact→EKI Task→Provider handle→Lease/Result/Output Artifact bodyの同一性を観測。実接続不能なら合成PASSと製品E2Eを合算しない。Runtime sourceやArtifactの異版、部分Input、非公開秘密をあいまいに補完しない。
- **G3 回復・権限の負例**：RG-03/05/06/07/08/09/11の停止条件を外部Backend＋host-side effectで検査。副作用を持つ処理は内部自動retryの制御/観測または冪等性の独立証拠なしでは配置しない。TEI/EKI間に新しいAuthority Issuer/Business State Storeを作らない。
- **G4 経済性/性能**：§25/#565のRuntime、#2600のBuild待ち、EKIのqueue/placement latency・resource-efficiency・data-egress/電力・Retry costを区別し、同一Expectedに適合する経路間で比較。クラウド予約/放棄の経済条件はOwnerの許可を要し、最適化候補がHard Policyを突破しない。

**未確定の選択**は「EKIが外部Groupに付与するReserve/Admission契約の強さ」「Resource Group内の異質適格性をProviderがどこまで保証できるか」「委譲先の内部retriesとEKI Attemptとの対応」「EKI Resource GroupとTEI Organization時限性/法域のBinding」「内部情報を漏らさない能力照会」「二Provider構成の実費・性能」の六つ。これらの公式API/標準DSL/RQ/ADR採否はEKI/TEI各Ownerが実検証で決める。TEI生成Go単一バイナリの直接実行・既存Host組込を後退させず、EKI専用の常駐Control PlaneをTEI内に足さない。

**証拠状態**：この§35はリポジトリの既存Issue/ADRと他チャットの設計判断を照合した**文書変更のみ**。新しいGo/SQL/TEI CLI/HPC/PC/クラウド実計測、EKI Worker/Provider Integration、実契約SPI、企業/法域のGrant、Owner正式受理、CI実走は**NOT RUN/未採用**。既存§27〜34と他レーンのDraftを変更しない。


## 36. 許容リードタイムと回復余力から配置を再判断する――TEI Job契約・EKI期限/信頼性/予算最適化の統合（2026-11-03横断追補）

> **設計上の受入候補／新しい製品Scheduler、価格エンジン、Job API、実EKI/TEI縦断・性能測定はNOT RUN。** 別チャットの「Jobの許容リードタイム、過去の成功・所要時間、失敗が重なるときも容認時間内で回復できることを考慮して、与えられた資源を**予算最適化**を基本として賢く配分する」要件を、§35の資源グループの実体適格性と[EKI ADR-0013](https://github.com/hat47x/eki-fabric/blob/main/plan/adr/0013-product-core-and-faces.md)の委任の核に接続。**EKI Coreへ金額・価格形成を導入しない**（既存ADR-0010/#108）。既存[#58](https://github.com/hat47x/eki-fabric/issues/58) Suitability、[#19](https://github.com/hat47x/eki-fabric/issues/19) shadow予測、closed [#108](https://github.com/hat47x/eki-fabric/issues/108) 期限引き上げ、open [#155](https://github.com/hat47x/eki-fabric/issues/155) 回復の公平性、open [#126](https://github.com/hat47x/eki-fabric/issues/126) Task Group、open [#115](https://github.com/hat47x/eki-fabric/issues/115) Backend内部再試行、open [#67](https://github.com/hat47x/eki-fabric/issues/67) 2 Provider接続を**再利用**する。各Issueにない実装や実SLAを先取りしない。

### 36.1 TEIの時間・復旧意味とEKIの計測・予測を分ける

| 層 | 引き渡す意味・観測 | 意図的に委譲しない判断 |
| --- | --- | --- |
| **TEI Job Definition（意味）** | Job/Stage ID、DAG・入力依存・Output/verification、`requestedAt / earliestStart / preferredCompletionAt / latestAcceptableAt`、maxInterruption・retry/idempotency/checkpointability・部分結果の扱い、許容する代替Target/Data Class、回復に許される残時間、外部発効操作の有無。比較票の候補項目であり正式DSLではない | 実行可能性・現在権限・Worker成功確率・予約保証をSource定義の文字列から確定しない |
| **EKI Admission/Suitability** | hard Policy/Resource/Grant/Isolation/Effect適格性と、Local/Existing/EKI委譲の価値比較。期限に間に合わない見込みや不足実測を明示して拒否・代替条件を返す | 実配置の承認をSuitability/予測scoreへ与えない |
| **EKI Placement/Recovery** | 合格したResource Group候補ごとの待ち、起動、入力転送、実行、checkpoint、出力転送、検証、取消/再配置時間を**同じ時計で**比較。予算と再試行上限のうち、残りに応じた初期/予備枠と継続的な再評価を扱う | 業務Flowの次Stepや「結果は正しい」というTEI/Hostの意味受理を生成しない |
| **Provider / Backend** | 実予約可能枠・キャパシティ/内部retry制御・適格子集合・開始/停止/Job handle、ネットワーク/Data localityと可観測な費用・遅延 | Providerの自己申告から保証・Trustを創作しない。Backend内部のスケジューリングをEKIが再実装しない |
| **Cost/Policy Owner（EKI外の金額意味）** | 物理消費量（CPU/GPU・転送byte・保存・関数呼び出し）と、許可された単価表/割引/通貨/税/上限の対応、実支出予測・予備費・エスカレーション許可を決定。価格表の根拠・版・期間を保持 | EKI CoreのBudget・Compute Grantを新しい会計/請求/価格形成エンジンへ置換しない |
| **Consumer / Runtime Effect** | verified outputが`latestAcceptableAt`までに使えるか（必要な実Artifact body/Consumer取得まで）、Task/Attempt/Result Claim、実Provider Commit/Receipt、不明を分ける | `worker finished`、`EKI completed`、業務上の有用な結果、支払成功を同一にしない |

**時間の定義**：`latestAcceptableAt`は「利用者/業務側が最後に役に立つとみなす時刻」の候補であり、EKIのTask timeoutやLease満了と自動一致しない。必要があれば`preferredCompletionAt`をsoft target、`latestAcceptableAt`をhard limitとして区別する。ただし利用側から時刻が提供されなければ無限待機・無期限回復を意味しない。環境Policyと能力開示の既定値を参照し、期限不明は目的別に`needs-measurement`/unresolvedで止める。Deadline判定には原則として「実処理完了」ではなく「結果検証/転送/必要なConsumer到達」を含める（要件が定める別Scopeならその定義を固定）。

### 36.2 確率的配置はhard条件の後、独立Oracleと共に採点する

1. **Hard filter first**：Effect/retry安全性、現在のCompute Grant・Data Class・Resource Group適格子集合、持ち主のopt-out、物理容量/予約、Providerの内部Retry制御、Token/Lease/時点、EKI/クラウド二重予算上限を先に評価する。実行可能性のhard条件に不適合ならAdmission／feasibility段階で候補を除外し、Suitabilityの判断材料が不足する場合は現行3値の`NEEDS_MEASUREMENT`とする。いずれも安い・成功確率が高いという予測で解除しない。
2. **候補完了時間**：待機/起動＋Input転送＋実行（分布）＋Checkpoint/再取得＋Output転送＋Verification/必須後段＋失敗時のbackoff/再配置を含める。各Stageの依存DAGでは直列総和と並列のcritical pathを区別する。実行時刻・Size・キャッシュ、Groupの観測鮮度、Context/Policy改版を固定する。
3. **予測の信頼度**：履歴はResource Group/Backend/Job Class/規模/時帯・Toolchain/Version・Data Classごとに由来と母数を保持。完了時刻の分布、打切り（timeout/中断・未観測）、失敗種別、検証不一致、相関する故障（同じ障害ドメインやクラウドリージョン）を別に扱う。欠損は失敗率ゼロへ補完しない。分位点・信頼区間と校正を比較する。外れた予測には`unknown`を返す。履歴の高成功率はAssurance/権限ではない。
4. **最適化対象**：適格候補の間で、`P(verified by latest acceptable time)`、見込総物理消費、外部価格表に基づく期待支出、データ移動・電力・利用者妨害、最大損失/回復余力を**別軸**で比較。価格/時間/成功率を未宣言の一つの重みScoreへ黙示的に押し込まない。まず予算/期限/成功許容の業務側制約を満たす候補を絞り、その中で利用側が明示した費用優先・時間優先・Pareto等の選択方針を適用する。費用の期待値が低くても、同一Eventの連続障害により期限違反する案は別に示す。
5. **Recovery Reserve（設計候補）**：第一Attemptの予算を使い切らず、必要なら再取得・下位実行・検証・実行先移行に使う**計算/転送/回数/残時間/外部価格表上の余力**を保持する。Reserveは見込み/予約/実際に法的拘束を持つ枠/消費済みを区別し、単に「予算を取り置いた」から物理資源を予約済みとはしない。同じWorkload/部門でのReserve共有と飢餓対策はEKI #155/ #125のPolicyに従う。recovery.Permit/Lease/Attemptは既存EKI #154/ADR-0015のdurable fenceを再利用し、第二回復Ledgerを作らない。
6. **再配置の契機**：Queue/Provider能力/Owner opt-out・現行予算・経過時間・checkpoint残量・推定失敗率が更新された際、`now + feasible_lower_bound_for_remaining_and_verification > latestAcceptableAt`で期限内成功不可能なら、黙って同じ配置を続けない。許可済み代替への早期切替（エスカレーション）が有効でも、切替費用/追加Attempt/元Leaseの取消・回収を評価する。実Budget・Grant・Artifactの現在性を確認した**新たな配置決定**が必要。予約なしの「将来のクラウド追加で間に合う」は保証でない。
7. **結果/通知**：期限違反が高リスク、確率不校正、価格情報なし、予備予算不足、代替なし、Provider/Grant未確認である理由を、開示可能な集約情報だけでClient/管理者に返す。利用側の期限延長/予算追加/品質条件緩和は**新しい明示Decision/Policy**であり、EKIが自己判断で緩めない。結果は`verified-in-scope`/`failed`/`expired-not-run`/`execution-unknown`/`partial`を区別する候補で、新API Enumの採用ではない。

**数理的な停止線**：`1-(1-p)^n`等で反復成功確率を改善するには、失敗が独立同分布という根拠が必要。同一Provider/出所/ネットワーク/Policyに依存する相関故障のとき、複製を独立と仮定しない。中断済みAttemptの「成功例だけ」の時間分布や、回収通知を通常失敗と合算した学習も棄却する。P(success by deadline)は統計的な予測で、契約上のSLAや保証ではない。

### 36.3 独立Expected付き受入票 LR-01〜12（未実行）

| ID | 入力・失敗注入 | 判別すべき結果・停止条件 |
| --- | --- | --- |
| **LR-01 リードタイム** | hard期限60分。候補A：待25＋転10＋計算20＋検証10=65分、B：待5＋転5＋計算25＋検証5=40分（すべて合成） | Aの実行時間20分だけを見て選ばない。Bの期限内候補性だけ確認し、Bの実予約とSLA保証は別 |
| **LR-02 安価だが回復不能** | A初回は10分/安価、同じ障害群で停止頻発。失敗通知から最短Fallbackが25分、残り20分 | 期限内回復は不可能/unknown。安い初回見積りを`recoverable`へ昇格しない |
| **LR-03 予備予算** | 外部価格表の試験用上限100単位、初回70＋Fallback50、EKI物理budgetは別 | 初回だけなら70<=100でも、想定両段階は120>100。reserveの金額判断はCore外のCost/Policy。Budget authorityが不足なら再配置不可 |
| **LR-04 相関障害** | 同じデータセンター/ストレージ障害に依存する二Group、各単独成功率90%と申告 | 二つへの複製で99%成功と断定しない。障害領域の独立証拠がない限り未知 |
| **LR-05 時間切れの中断** | 実行8割進行して再起動が必要、checkpointなし/あり | checkpointなしはゼロから再計算、ありも状態byteの検証と移送時間を含める |
| **LR-06 Capacity急減** | PC Group100→10、Provider観測時刻は古い。Lease/回収に遅れあり | A/Bの余力を現在予約可能としない。再評価がGrant/Reservationを自動発行しない |
| **LR-07 Deadline再評価** | Queue時は間に合う予測、20分後にProvider混雑/移送時間が増加 | 期限までの残り時間から再判定、許可済み代替の切替か早期not-feasible通知。単にretry回数を増やさない |
| **LR-08 隠れたRetry** | 外部HPCが同じJobを自動再実行、TEI側はnon-idempotent effect | 外部Retry禁止/観測/冪等根拠のない経路を拒否。EKI Attempt=1が物理実行1回の証明ではない |
| **LR-09 Reserveと公平性** | Recovery優先Groupが続き、通常Taskが無期限待機 | #155のfair share/agingによる制限を検証。Recovery ReserveはBudget/Quotaを増幅せず、Starvationを隠さない |
| **LR-10 期限後結果** | 期限を過ぎてTask phaseはqueued、別例は実行したか不明で遅着Result | expired-not-runとexecution-unknown/late-resultを区別。遅着を無条件採用/再実行しない |
| **LR-11 Group部分結果** | TEI評価100件のうち80件はVerified、15件timeout、5件未実行/不明 | EKI Task Group/Result Countの記録とTEIの集合/Expectedの業務Meaningを別にする。80件を全100件の合格と呼ばない |
| **LR-12 実行経路比較** | 同じGo単独JobをLocal/EKI Native/EKI→外部Groupで処理、転送/検証/回復/実費を含む | 同じSource/Executor digest/Expectedを固定し、deadline success・合計物理消費・外部価格・追加Hopと結果の実byteを比較。必要ないEKI委譲を強制しない |

### 36.4 既存Ownerでの実証順と採用判断

- **T0 適格性とJob契約**：TEI #1130でFunction→Job Plan/Output/verificationと時間・再試行可能性・部分結果の意味をread-only比較し、EKI #58のdeterministic suitability + hard policy filterへ渡す。実TEI Job Lowererがなければ`NOT RUN: TEI job producer unavailable`。新しいDSL構文を先に採用しない。
- **T1 決定的Baseline**：EKI #19の既存`spikes/artifact-locality-cost`、EKI #108/#140設計のdeadline引き上げ・物理消費と、TEI §35の資源Group候補に`queue + transfer + execution + verification + possible recovery`を加え、LR-01〜07を固定Fixture/独立Expectedで反証する。MLのスコアはBaselineとの差としてshadowへ置き、本番Placementに直結しない。
- **T2 配置と回復のAuthority**：EKI #154（durable recovery予算/Permit、closedだが実配線の現在性は別確認）、#155（fairness）、#115/#67（backend retry/Group）、#111（期限超過の公開状態）、#126（Task Group）の既存OwnerへLR-08〜11を配分。Current policy+Budget/Grant+Lease/fenceを再検証したAttemptだけ新規実行可能とする。実OS/Cluster Provider未達ならその範囲は`NOT RUN`。
- **T3 予測と同一試行評価**：十分な合成/実観測が得られた場合に限り、EKI #19で `P(verified before latestAcceptableAt)`のcalibration、group/time slice、censoring、correlated failuresを測る。deterministic baseline対shadow predictionのfalse escalation/missed deadline/budget overrun/extra retries/false successを独立採点する。Shadowより権限のある学習Schedulerを無断採用しない。
- **T4 最終的な経済性**：既存EKI物理Budget/Contributionと**外部Price/Policy**による支出見込みを別票にし、TEI Go単独/外部Host/EKI Native/EKI Groupの利用価値を比較する。単価表のRevision、コスト範囲、最大損失、時間条件、結果検証費、予備費を明示。必要な業務/組織Policyや外部決済・契約をEKI Coreに実装しない。P0-a/P0-bの実業務Effect/Receipt（TEI #1119/#1174）を前提とせずJob完成を宣言しない。

**正式判断の保留**：soft/hard deadline・停止の優先度、目標とする期限内完了確率、Reserve/Quotaの割当、複製の許否と相関モデル、価格外部AdapterとBudgetの連動、前払い予約の強さ、成功/一部結果の用途別意味、学習による順位決定の採否。これらはOwner/Provider/Deploymentによって異なり、共通SLA・単一価格Score・新中央State/Cost Registryとして先行固定しない。

**証拠状態**：本§36は既存TEI/ EKI契約に対する**Docs-onlyの差分**。LR-01〜12の本番/合成テスト実行、TEI→EKIの時間Contractの実ABI、計量/外部料金、Cluster/PC群の実予約・失敗回復、ML学習・shadow評価、性能/予算最適化の効果、TEI Owner意味受理、EKI Ownerの要件/ADR採用は**NOT RUN/未決**。§35とEKI ADR-0013を変更せず、公式製品の実装済みとは主張しない。


### 36.5 回復許可と期限/予算適合を同一試行で接合する（RA-01〜26、2026-11-03）

§36のLR-01〜12は「期限・回復余力・外部コスト」の資格だが、§31で観測した**Command Targetの不一致／内外Commit分離／初回とReplayの結果情報消失／Commit uncertain**を合わせて判定しない限り、回復のための新Attemptを安全に許可できない。§32のImported Source資格、SQL Schema閉包、§37の組織Policy/現在Grantが古いときも同様である。

この**再実行の安全性A × 期限内にVerified Outputを得る可能性B**の二軸を、[独立した詳細計画](RECOVERY_ADMISSIBILITY_EFFECTS_AND_BUDGET_2026-11.md)にまとめ、[RA-01〜26の合成Reference](../fixtures/recovery-admissibility/check_recovery_admissibility.cjs)を追加。保存したGitHub上のJSを再取得してV8で**26/26 PASS（合成のみ）**を確認した。安全性・費用予測/実Resource Group/Host/実DBを検証する本番Gateではない。

- **先に照合**：Business Commit unknown→同OperationIDのReceipt/独立Effect Read、Task completed/ACKだがBodyなし→Result取得、旧Lease active→Fence/遅着確認。Deadlineや予算が残っていても新しいOperationIDでEffectを再投入しない。
- **再投入候補になった後に評価**：実Grant/Permit・内部Retry可否・現在Reserved Capacity・Core物理Budget・Core外の価格/支出承認、Queue+転送+Compute+Verification+Recovery、障害相関・公平性、Consumer実Body取得を比較。予測・候補・真正の予約/実Executionは区別。
- **Ownerへの受渡し**：TEI #1130/#2598/#1174/#2600、EKI #58/#115/#154/#155/#19/#112の既存契約へRAを分配。特に[Macro #2780](https://github.com/hat47x/tei-platform/pull/2780)の実Go 4件のPASSは危険な配線のCharacterizationであって安全な実業務再実行の合格ではない。独立Expectedで同一Provider/Attemptを実測するまで新Product ABI/Controller/Grantを採用しない。

**証拠範囲**：本節へのリンクとReference合成26件の成功のみ。TEI→EKI実Job一試行、実Go/PG/Result Consumer、Recovery Permit/Resource/価格・SLA、正式Owner受理は**NOT RUN**。#1119 P0-a/P0-b、§31・§35・§37のOwnerを変更しない。

### 36.6 実EKI P0 APIを確認してからTEI Jobを配置候補へ進める（2026-11-03、実Schema Source-static）

TEI [Draft #2787](https://github.com/hat47x/tei-platform/pull/2787)は、[EKI external OpenAPI](https://github.com/hat47x/eki-fabric/blob/main/spec/external/openapi.json)の実Source Git blob `388f2e3ad32ba663b4777fcfb4304b0cf6f48f3f`から選んだSchema/Pathを**検証用Fixtureとして固定**し、WP-01〜19で**TEI Jobの必須意味をEKI v1 TaskRequestへ黙って欠落させないか**を検査する。API Source＝実Task/Worker/Result成功ではない。

| EKI P0外部契約の観測事実 | TEI側の停止条件 |
| --- | --- |
| `TaskRequest.parameters`はあるがSourceに`not passed to workers`と明記 | パラメータ依存の実JobはAdmittedでも実行同値を主張しない |
| `ExecutorRequirement.version_constraint`はAdmissionで未評価、`digest`欄なし | 実Executor版・byteの固定は別Evidence。Exact PinをConstraint文字列で代用しない |
| `ReliabilityRequest.effect_mode`は`none`だけ | DB業務Effectやnon-idempotent Scriptをpure-computeとして受理させない |
| `deadline`は1個のみ | `notBefore`、soft target、検証・Consumer到着込みのlatest usefulを無音で潰さない |
| `/v1/artifacts/{artifact_id}`はGET経路と501応答が宣言される | EKI Resultのdigest/completedは実Artifact body受信・業務Commitを意味しない |
| ResourceRequest物理量はあるがTaskRequestに直接のResource Group/Data-residency欄なし | 資源Group名・勤務地等の自己申告でCompute Grantや実配置を増幅させない |

**比較列**：`not-representable`（v1が必須意味を表せない）／`wire-shape-only`（静的JSON表現は可能）／`unresolved`（Worker伝達・Pin・期限意味・実入力/出力・Grant/Effect不明）／`runtime=NOT_RUN`（現行Source/Schemaテストのみ）。これらはEKIの正式Response enumでも新TEI Core型でもなく、**既存T0〜T4に挿入するT0.5 wire-ABI preflightの受入票**である。

実接続はTEI #1130のFunction→Job Producer、EKI #112/#266のExecutor/Parameter、#320/#494の実Artifact body、#67のProvider Group、TEI #1174の実Commitへ段階的に委譲する。実行先が未適合な場合、Go単体/既存Host実行を残し、EKI経由を強制しない。#2787の限定CIがPASSしても、Worker/Provider/Host/SHO/Owner E2Eは未証明。新Task API/Job Engine/Host/Authorityを先行採用しない。

### 36.7 EKI Suitabilityの実装・ADRの現在値（2026-11-05）

[EKI PR #714](https://github.com/hat47x/eki-fabric/pull/714) はmain統合済み。ADR-0029 §2は実装と同じ **FABRIC_JUSTIFIED／FABRIC_NOT_JUSTIFIED／NEEDS_MEASUREMENTの3値**へ変更された（ADR自体の状態は依然「提案中」）。Schedulerへ進むのはJUSTIFIEDだけ。NOT_JUSTIFIEDは正常な非分散判断であり、**EKI Coreがローカル実行を始める意味ではない**。NEEDS_MEASUREMENTは事実不足や利点が最小差以下の状態。旧NOT_SUPPORTEDはSuitabilityの第四値ではなく、Admission／Security・Data・Resource feasibilityで除外する。

旧FABRIC_ELIGIBLE・LOCAL_OR_EXISTING_EXECUTIONという名称を、今後のTEI Job→EKI契約/独立Expectedに採用しない。実dispatchは固定見積りによる暫定経路を含み、DecisionReceiptの稼働保存・表示、Caller Claim provenance、実測に基づく非分散選択はまだ十分に実証されていない。§36.2のhard filterはSuitability列挙値と別のAuthority/feasibilityとし、FabricEntryも配置・Leaseの認可ではない。EKI #58/#698がOwner。LR-01〜12/TEI実Job→EKI期限内実行はNOT RUN。


## 37. 組織由来Record Policyの広域適用と個別除外：一般Data/RelationによるPolicy候補から実Provider/Host Enforcementへ（2026-11-03、採用前）

### 37.1 要件の適用範囲と意味を独立にする

[組織設計 #2389 §58](https://github.com/hat47x/tei-platform/pull/2389)の方針：組織情報（Person/Membership/OrgUnit/Hierarchy）は**標準Data/Relation**。ただし、業務Recordの`responsibleOrg`等へ**明示Relation Binding**を行うと、利用者が選択したData型・業務範囲に組織由来のレコード可視性・参照専用Policyを**広域一括適用**し、必要なRecord/Functionのみ例外を宣言できる。Organization Profile/Pluginは組織構造・階層axis・時点・Ruleの宣言に使い、第二のACL/Host/中央Policy正本は作らない。実際の認可は既存Host/Providerが担う。

**原則的な区別**：Rule applies / applies with exception / in-scope / out-of-scope / unknown / read-only candidate / current Host-authorized read / actual Provider-enforced read / external Disclosure / write Effectは独立。特に**『広域Organization Ruleから除外した』＝『公開された』ではない**。残るHost/他Rule/実Providerの明示許可へ戻す。Ownerが定義していない`any/all`（複数所属・多経路）や例外のDeny/Allow優先度をデフォルトで勝手に決めない。部分取得・深さ制限・循環・Tenant越境・古いGrant/Sourceは`unverified`または拒否として説明し、無根拠に行の存在・不存在やAuthorityを認めない。

### 37.2 一つのModelを複数のRead/Effect/Consumerに使用する統合試行

既存#2386は組織/時点の意味、#951はProvider/Plugin契約、#1126はData/Relation/Logical ID、#2506はFunction Data Contextの**宣言した**Data/Relation、#2622はData Ops/Provider realization、#2387/#2635はWHERE/ORDER/EXISTS等の非出力Query依存、#2628はactual Read/Enforcement Witness、#2582は現在Claim再資格、#2516はCUI/Studio/SHO/AI開示、#2598/#1174はRequest Target/current Grant/実Effectを所有する。#2586はこれらの**固定一Attemptの引継ぎ**だけを持つ。

| 試行と目的 | 必ず同一版で束縛するもの | 失敗/停止条件 |
| --- | --- | --- |
| PS-1 default on for selected Data | Policy scope+Owner+revision、Record→Org typed Relation、axis/root/cut、principal/tenant/grant epoch | 非適用Dataにも暗黙ACLを掛ける／Record名からOrg推定 |
| PS-2 scoped Read and read-only | Query/Function Mapping、実Providerの行/Field/Route/Read Cohort、現在HostのRead/Update判定 | UI側だけ隠す／Read成功からUpdate可と推論 |
| PS-3 per-record exception | 対象Rule ID/適用除外目的/期間、他Policy・実Host現在判定 | exception⇒allow/public/bypass／優先度不明 |
| PS-4 recursively reachable hierarchy | Organization axis、path occurrenceとnode distinct、valid/known、edge Source/tenant、cycle/limit診断 | 複数Pathで重複件数／別軸やTenant越えの黙示Grant |
| PS-5 actual negative enforcement | List/Detail/Count/Page/Export/Join/Aggregate/Cache/Batch/AI/SHOの実全Route | フィルタ後処理／隠れた件数や検索存在による漏洩 |
| PS-6 staleness/effect | Policy/Role/Source/Grant改版、表示Epoch、Action target/Receipt独立Read、EKI再Attempt | 旧画面・旧Plan/Grantから現在実Read/書込を許可 |

**Syntheticのみの証拠**：[独立Draft #2781](https://github.com/hat47x/tei-platform/pull/2781)へ`VP-01〜30`を追加し、V8互換ローカル30/30成功。fixed-head Node CIを確認してからその結果を別記する。実Host/IdP/Provider RLS・real read witness、業務OwnerのRule/exception受理、Studio/SHO/AI実送出は**NOT RUN**。#2681の実Native TEI E1成功や#2779のWhat-if E0成功を、Policy Enforcementの証拠へ転用しない。

### 37.3 UI・生成AI・EKIとの責務

Studioは「組織軸とRuleの一元設定・効くRecord集合・例外・各Consumerへの影響」を表示し、AIは自然言語から既存Plugin/Rule Bindingの候補を提案できる。ただし適用・Source CAS/Role Pin・Preview・Owner判断は別。通常Dataへの手入力メタ情報を増やさず、必要なDataへのRule Bindingだけを要求する。情報境界を含む外部AI/SHO開示は#2516へ、Graph/Tree表示のPath重複・意味保存は#2497へ戻す。

EKIがTEIの組織関連Jobを受理する場合、EKIの配置可能性・予算・期限・Lease・回復最適化は**組織DataのRead/Disclosure Grantではない**。現在のPrincipal/Delegation/Grant、Source/Policy/Read ScopeをTEI/Host/Providerが引き続き検査し、再試行・別Nodeへの配置で失効済資格を蘇生させない。EKI Coreに新しいOrganization ACL/財務認可正本を設けない（既存§35〜36/#1119を維持）。

**未決のOwner分岐**：複数Org linkのany/all、単一親/DAG、RecordとFunction/Consumer単位の例外粒度、Ruleの合成順位・非適用時の既存契約、閲覧と参照専用の違い、実ProviderでRLS/Filter/Host境界をどう保証するか、同一Scopeでの歴史照会の可否。これらの意味はOrganization/Business Ownerが決定し、実Enforcement資格はHost/Provider Ownerが判断する。未定部分を仮の安全性保証にしない。

### 37.4 E0 fixed-head run と実Enforcement不足の分離

[Org Policy Draft #2781](https://github.com/hat47x/tei-platform/pull/2781)のfixed HEAD `88f3e3e2e1a2352fa53625732f465fb2b8b07761`で、[Node Actions #41632269936](https://github.com/hat47x/tei-platform/actions/runs/41632269936)は**success**。Node v22.16.0、`verify.mjs`27 checks成功、`node --test`235/235成功（新規VP-01〜30を含む）、fail 0。GitHub jobログ上の新Source Blob `bdc4e63649fa3e728c8293fef55b43a0d278b1c1`とexact-head/clean stepを確認した。

**昇格禁止**：実Host Grant/actual Provider Filter/Row-Level Security・実Data Context・テナント間非干渉・Query全経路・SHO/AI出力・実Source/Policy受理は未実施。synthetic `hostSimulation` の結果は実環境の認可ではなく、実Dataの秘密性を裏付けない。#2779 What-if別ブランチのNode成果や#2681 Native Goの成功を本PRの一試行と足し合わせない。#1119の製品Critical PathとSecurity/Privacy/Owner審査を維持し、実Enforcementを先行受理しない。

### 37.5 最終fixed-head 33件とHost Scope根拠の別扱い

[Draft #2781](https://github.com/hat47x/tei-platform/pull/2781)へ**VP-31〜33（Root・Axis・Policy Revision不一致）**を追加した。候補のScopeがD1からHに変わった、`command`から`budget`へ切り替わった、または`p-r1`から`p-r2`へ改版したのに、実Hostの旧Grantを利用してよいと判断する実装を拒否する。もちろんOrg MembershipやDirectory Groupと現在Grantは別事実のままである。

固定HEAD `565a260403445556bb088f0433a53760fdcddda7`の[Actions #41632981999](https://github.com/hat47x/tei-platform/actions/runs/41632981999)は**completed/success**、Node v22.16.0、`verify.mjs`27 checks／`node --test`238/238 PASS・fail 0（VP-01〜33を含む）、Source Blob `9cb39f44a9eb55be51c3e56cc1cb19bbf16c4e8d`を確認。§37.4に書いた235件は旧HEADの局所合成成功の履歴。**現HEADに対する実TEI Provider/Host Enforcement／テナント分離／Query全Read/Export/Batch経路／SHO/AI Disclosure／Owner Meaningの資格はまだない**。

## 38. 組織Record→Org Relationを実TEIで検証したE1と、次の実Provider Enforcement試験（2026-11-03、#2586／#2622／#2516への具体的引継ぎ）

### 38.1 実Goで確認したことと、確認できないこと

[PR #2681](https://github.com/hat47x/tei-platform/pull/2681)の`reference/org-headless-workspace`に`CaseRecord`と`relation handlingUnit → OrgUnit`を直接`.tei`に追加した。Membership/Occupancy/Personと違う一般DataからOrgUnitへ**型付き明示Relation**を持てる構造を、既存Parser/Workspace Loaderで読み、`TestOrgHeadlessCaseRelationDoesNotAuthorizeReadOrWrite`でSemantic target/key一致と無関係なCase Operation/Host Binding非生成を固定。

HEAD `a2110f4a8e8cfa60a2f2c57c1de4a84d75c5bcec`、[Native Actions #41640612514](https://github.com/hat47x/tei-platform/actions/runs/41640612514) **completed/success**：Go1.26.10 Linux、exact-head/clean、Go focused test・実`tei validate`/`operations`/`artifacts`・別JSON Readerがsuccess。このReaderは`findings=0, complete=False`を出力しているため**Artifact生成全体completeや外部Consumer実受領は未成立**。以前の#2681 HEADや#2781のNode合成とは別Run/別Source。

**E1で成立**：`Case→OrgUnit`という明示Relationが実TEI Core Semantic Worldへ残る。**E2以降で未成立**：このRelationが「D1配下のCaseを読む」と業務Owner受理されたか／FunctionがそのPathを要求できるか／実Providerが行集合と必要Fieldを取得したか／実Queryにスコープ制限を施行できたか／現在HostでPrincipalに許可があるか／SHO・AIへ開示してよいか。この順序を逆転しない。

### 38.2 #2586の単一Attemptへ渡す最小入力と欠落時の停止

元`CaseRecord`は**未Persist・未Operation Binding**なので、現状を「RDB接続済みCase一覧」とは扱わない。最小の次段は別Data/Providerサンプルとして、以下を**独立Expected付き**で一つだけ実現する。

| 観測対象 | 実現側の既存Owner | 不合格条件 |
| --- | --- | --- |
| Source/Meaning/Relation | #1126/#2386：Case identity、`handlingUnit` target/key、Owner承認のScope意味 | Data構文を読めたことからRule/Grantを推定 |
| Explicit Function Context | #2506/#2622：Case ID+Org Relation path/Hierarchy axis/root/必要Field/Scope/valid/knownの明示 | 未宣言Field/Relationを実Providerが勝手に読む |
| Physical Data/Policy Enforcement | #2611/#2628/#2387：Provider Route/actual Read/Rowset+Field/Policy版、List/Detail/Count/Page負例 | UIのafter-filterやOwner未受理`any`で許可、部分0件から不存在確定 |
| Display/External Consumer | #2497/#2582/#2516：実Visible Candidate/Read-only/exception+現在Grant/Disclosure、SHO/AI、既表示撤回 | metadata/Count/診断/Export経由でScope外情報漏洩 |
| Current Action/Effect | #2598/#1174：現在Grant/Request Target/Host Effectと実post-imageを別検証 | Case→Org RelationだけでUpdate/Approval/Delegationを実行 |

**第一目標**：同じ実Read/Provider/Root/Axis/Policy版で**scope内Case1件、scope外Case1件、read-only例外1件、適用除外だが他Host Denyで拒否1件、部分取得でunknown1件**を確認する。実Producer/Enforcement APIが未接続なら`NOT RUN: enforcement-unavailable`。Owner未受理は`not accepted`として停止し、#2781の合成33件で代用しない。

Security/Privacy/実Host/DB/Ownerレビューは既存#1119 Gateに委譲。専用TEI Host、組織ACL Engine、第二Query Planner、秘密情報を伴う未承認外部Harness自動試行は追加しない。

## 39. 組織Policy変更後の既表示情報と後着Read：現在資格を用途ごとに再確認する（2026-11-03、実E1からE3/E4へ、採用前）

**現物の境界**：実TEIの`CaseRecord→OrgUnit`を保持する[PR #2681](https://github.com/hat47x/tei-platform/pull/2681)はfixed-head `a2110f4a8e8cfa60a2f2c57c1de4a84d75c5bcec`で[Native Go CI #41640612514](https://github.com/hat47x/tei-platform/actions/runs/41640612514)成功。ただし`tei artifacts`の独立Readerは`findings=0, complete=False`を報告しており、Builder/全Consumer成功ではない。実Query/Context→Provider RLS/Host Grantへの接続もない。

**E0の新条件**：[Draft #2781](https://github.com/hat47x/tei-platform/pull/2781)のPolicy可視性VP-01〜33に加え`verify-visibility-epoch.mjs`のVE-01〜20を追加（作成時V8互換局所20件PASS、新fixed-head CI確認は別）。Source Locator/Meaning/Policy/Role/Root/Axis/valid・known/current Grant/Principal・Tenant/Read Cohort/Query非出力依存/Selection Epochが変わったら、後着Responseを現在のUI・Tool/CUI・Cache・SHO/AI準備物へ復活させない。A→B→Aの同Record IDも別Request epoch。純粋な色/配置変更で適格なData Claimまで一律失効させる必要はない。

**用途別停止**：普通の一覧Readが成功しても、Query Count/Page/Search/Export/AI/SHO/Batch/Updateの実施行資格は別。部分取得/0件/未知Readをfalse/完全空集合に変換しない。Policyからの適用除外≠他Policy/HostのAllow。実Providerに制限を掛けずUIだけで隠す実装は不可。未保存Patchは機密保持の可否を確認した範囲で隔離し、古いIntentの自動Submitは拒否。既に適法に送出されたコピーの外部完全消去は不可能であり、「現時点の閲覧制限」と「過去資料の撤回・通知・監査」を別に扱う。

**既存Ownerの一固定Trial**：#2582/#2506がContext・選択・結果Requalification、#2622/#2628/#2387が同Source/Policy版・実Read/Query非出力/Enforcement、#2516/#2631がUI/DOM/a11y・診断/件数/AI/SHO実消費、#2598/#1174が現在Action/Effect。Read要求→Policy/Grant失効→遅延Response→表示拒否・Cache再資格・下書き隔離→別Provider RouteでのCount/Export拒否を**実Request/Response bytesと独立Expected**で同じAttemptに束ねる。実Endpoint/Hostが未接続なら`NOT RUN: enforcement-unavailable`、Scopeや実Read不明なら`unverified`、Owner意味の受理がなければ`not accepted`。独自TEI Host/第二Reactive Broker/ACL/Policy Engineを作らない。#1119 P0-a/P0-bの実DB・Owner-writeを優先。

### 39.1 VE-01〜20のfixed-head実測

[Draft #2781](https://github.com/hat47x/tei-platform/pull/2781)のfixed HEAD `77ac2f3e49394aa8742c39181168607d8e3aff77`に対する[Node Actions #41641623522](https://github.com/hat47x/tei-platform/actions/runs/41641623522)はcompleted/success。Node v22.16.0、Reference 27 checks PASS、`node --test` **258/258 PASS、fail 0**（新規VE-01〜20を含む）。ソースBlob `93cfbe4146904c2d8c3c613c4a67bae6ecb0e340`。これは合成E0で、実Provider・Host・Studio/SHO/AI・Ownerの許可や実開示はNOT RUN。#2681実Go E1と別試行を合算して製品E2E PASSとしない。

## 40. 横断Evidenceの最小閉包と実装優先順（2026-11-04、横断レビュー・計画段階）

### 40.1 結論と不変条件

直近のDSL/Plugin、外部Reference、高速Build、テスト、組織Policy、EKI、Studio/SHOの各設計が成熟している。しかし個別PASSの合算では、同一Source revision・Function Instanceの意味、対象、実Read、実行資格、実Effectが連続していると証明できない。次の主軸は新しい共通機構ではなく、既存#1119/#1163/#1180と本計画§19–39に沿う「一Attemptの証拠閉包」である。

構造同一≠Meaning同一、AI候補≠Owner受理、Action Offer≠実行Grant、Reference fresh≠必要Read完全、Artifact byte一致≠現在の使用資格、Worker successful≠Host Commit確定、同じ結果0件≠不存在。実装、合成Reference、計画、Owner未受理は別資格とする。

### 40.2 最小閉包（第二Evidence Engineは作らない）

| 段階 | 既存入口・担当 | 同一Attemptで記録する独立証拠 | 未成立時 |
| --- | --- | --- | --- |
| C0 Source | #2743/#2674/#988 | Source byte/span、外部Reference revisionとfreshness | stale/unavailable/partialを明示 |
| C1 Meaning | #2639/#2620/#2644 | Typed Semantic、Role/Operator pin、Owner受理の適用範囲・単位・時点 | Parser成功のみならnot accepted |
| C2 Read | #2506/#2622/#2387 | Function Context、Predicate/Relation/否定依存、row/field completeness、実Provider read | 部分0件から不存在を推論しない |
| C3 Build/Consumer | #1163/#2600/#2604/#2516 | 実Go binary/build digest、Artifact全memberと独立Reader byte、実Disclosure | Manifestのみならunverified |
| C4 Effect | #2598/#1174/#1119 | Intent target、Principal/Host Grant、Provider Request、Commit/unknown、独立post-image | ACK/Tool表示だけでEffect成立としない |

読み取り専用経路はC0–C3で検収し、更新経路だけC4を必須にする。実入口がなければ NOT RUN: concrete-entry-unavailable と記載する。既存Scenario、Receipt、Manifest、Owner受入票を使い、新Host・Registry・Policy Engine・Runnerは追加しない。

### 40.3 優先する3つの反証

- **V-READ：実CLIと独立Reader**。同一Workspaceの一改訂で、完全0件/部分0件、Reference stale、A→B→A、同byte別Meaningを独立Expectedで比較。既存#1163/#1180を利用。まず非更新・単一Provider。
- **V-ACTION：Contextから実Request**。Case→OrgUnitの構造E1をRead資格に昇格せず、scope内/外、read-only、他Host Deny、partialの5例でUI Offer・現Grant・Request対象・実post-imageを分離。実Enforcement未接続ならE2以降NOT RUN。
- **V-DSL：同じExpectedのFrontend比較**。Go式AST親和、軽量独自構文、Hybridの3案をNULL、Decimal、Source位置、診断、部分集合、Go生成で比較。Frontend選択とBackend Go生成を分離。AIの初回成功・修正・tokenを記録し、人間専門家の可読性をAI試験で証明したと主張しない。

### 40.4 優先順位と停止条件

1. P0：#2667/#2639/#2681/#2781のexact HEAD、Draft/main、CI種別、実Provider有無を再確認し、別PRのPASSを合算しない。
2. P1：V-READ。実CLI・独立Expected・Reader・否定依存を確認するまで新規高度機構を増やさない。
3. P2：V-ACTION。Owner受理のない業務RuleやScopeを自動補完しない。実Provider/Hostの現在認可は別判定。
4. P3：V-DSLと生成Go単一バイナリの一改訂縦断。正しさの確認後にp50/p95、build latencyを別々に計測する。
5. P4：価値が確認できたときだけSHO/Studio、別Harness、EKI委譲、異Target・異Domainへ拡張。単独製品に共通基盤の必須依存を課さない。

Owner Meaning未受理、必要Readの不完全、Pin不一致、Grant不明、Commit unknown、参照/Producer不在をsuccessへ昇格しない。表示・Preview・保存Intent・実Effectを区別し、用途別に再資格化する。

**実施状態**：本節は既存Draft計画へのdocs-only追補。新しい一気通貫実験、Go生成、実DB、Windows UI、Owner受理、最新CIは本変更ではNOT RUN。既存Gate・責務を置換しない。正式ADR/文法/Plugin ABIも未採用。


## 41. AI-first Scenario検証とSource・Build・Job境界の最小接合（2026-11-04、他チャットの最新議論との横断整合、Docs-only）

### 41.1 §40の「P1 V-READ」とAI-first P0の順序を明確化

§40のP1 V-READは**最初の製品横断Evidence受入対象**であり、**テスト基盤の実装着手順序**を意味しない。実装着手の先頭は#2811で定義された「生成AIが小さなテストを検査・追加・隔離実行・比較できる足場」。その足場から最初に生み出す縦断対象がV-READである。ScenarioやAI用JSONを正本ごと統合する新Runner／第二DSLは作らない。

現存する二つの入口を区別する。
- **Track A / DSL候補**：Draft #2813 は既存Go ReferenceのParsePipelineSyntax→LowerPinnedPipeline→ComposePinnedと JSON candidate 9+2件を使う。Goテストを書き換えず候補ケースを追加できるReferenceであり、独立Oracle・Coverage分母・実製品の権限資格は未成立。新PR HEADのCIとmain統合は別判定。
- **Track B / Driver Scenario**：実mainに cmd/tei/scenario.go の verify/generate/run/check と internal/scenariotest/scenario.go の tei.driver-scenario/v0 がある。生成CLI planのScenario/World/Plugin Lock/Fixture pinと、既存6-Step例を起点にする。このv0は安全な合成Read/Validation/Observationに限定され、全業務Command/Web/GUI同等性は未証明。旧「Scenario CLIがない」という断定を使わない。

**AI-firstの最短順**：(A0)既存Scenario/Fixture/Operator/Case ID/Targetのinspect → (A1)事前登録のCoverage Obligation・独立Expectedを固定 → (A2)AIがJSONまたは許容v0 Scenarioの追加/変更案を隔離作成 → (A3)既存Validator/ParserでID重複・版・Source・型・未知operationを拒否 → (A4)既存Goテストまたはscenario verify/generate/run/checkを安全なfixtureで実走 → (A5)独立Expected・固定Mutant・反例とCoverageの差を照合 → (A6)Ownerレビュー後にScenario/Fixtureを受理。自動生成や実行成功だけでExpected・Obligation分母・Meaningを更新しない。

### 41.2 AIが増やせるのは「テスト候補」、勝手に変更できないものは独立に固定

| 証拠・入力 | 提案/生成AIの変更許容 | 独立の判定と責務 |
| --- | --- | --- |
| Scenario Candidate / JSON Case | 隔離Branchで追加・修正可。ID namespace衝突、未知Field/Operationは禁止 | #2811/#2813/#1180で静的検証、正式受理はReviewer |
| Coverage Obligationの分母 | AI提案としては可。試行中の再定義不可 | #1180/#2514：既存Canonical/Scopeから先に固定。0件・未探索をcoveredにしない |
| Expected / Oracle | AI候補は可。Actualのコピーを独立Expectedとしない | #2514：独立計算、既知Mutant、必要時Blind holdoutで判別力を測る |
| CLI/Web/GUI派生物 | Scenarioから再生成、Target-specific assertionは別 | #1180/#2391：CLI PASSはWebのsession/CSRF/DOM/a11y/screenshot PASSではない |
| Provider/Hostの権限・Commit | AI生成・探索からGrantを推定しない | #1174/#1119：明示Authority、隔離DB、Effect fence、独立post-image |

**最小反証セット AI-V-01〜08**（新Case IDを正本へ勝手に登録せず既存Ownerへ割付）：
1. JSON候補の重複ID・未知Stage・不正型で拒否。元の基準ケース削除により偽PASSしない。
2. ActualとExpectedを同時に変更した自己追認を、固定Baselineと既知Mutantが検出。
3. 対象0件・partial 0件・未探索を別状態とし、Coverage分母を縮めない。
4. ScenarioのRevision/Plugin Lock変更後、旧CLI planと旧Web生成物がstale。対応不能Stepをskipしない。
5. CLIで同じBusiness ResultでもWebの認可/session/表示/視覚回帰はunverifiedのまま。
6. Source原文のみ変更・Semantic digest不変で、意味の再利用と引用/出典の再資格化を分離。
7. UIでOfferが表示されてもHost Grantが消失したCommandは送信拒否。Commit unknownを自動再送しない。
8. 複数Agentが同一Case ID/Expected/Fixtureを競合編集しても、同一Revisionの根拠に自動採用しない。

**受入KPIを混同しない**：候補ケース追加数、構文/限定Assertion合格数、事前固定ObligationのCoverage、既知Mutantの検出数、実Provider/Targetでの独立Observation、生成・実行費用を別列表示。テストの数・合成PASS・複数AIの同意は保証水準にならない。AIの探索権限はread-only/隔離を既定とし、実外部Operation/秘密の自動送出は行わない。新Agent Runtime・test-only controlの本番残留は許容しない。

### 41.3 Source Witness・Disclosure・後着結果との接合

#2743/#2801/#2812を再利用する。#2801の実Go負例（exact-head Repository Health成功）は**同一Semantic Digestでもraw byteが変わり、LocateImportedSourceの後段再Loadは初回Loadの消費byte証拠ではない**ことを示す。#2812では同一Loadのselected parsed Source/Manifestの局所Witnessを内部保持し、原文Readerをpackage privateに保持する設計へ進み、さらに2026-11-04の現HEAD `8b90b606` では公開Describeアクセサ自体を削除した。内部Source path・raw fingerprint・件数の新規公開も認めない。直前HEAD `7a6071ed` のCIはsuccessだが、現HEADのRepository Healthは確認時点でin_progress。Provider全入力閉包も未成立。

AI/Scenario/SHO/Studioが実Provider Sourceを扱うとき、**Importされたこと≠原文開示許可**、保存Witness≠現在Read Grant、選択Namespaceのみ≠全依存完全性、後着Response≠現在表示資格を維持する。raw byteをログ/外部AIへ自動露出せず、固定Scenarioには許可済みの合成・匿名Fixtureを使う。Source再読込、A→B→A、未解析Source、Manifest変更、negative dependencyは旧結果を再利用できるかを独立判断する。既存#2600/#2516の再資格化を複製しない。

### 41.4 実装開始からV-READへの橋渡し

- **A0（最優先）**：#2813でJSON候補1正例+1拒否を、実Goのexact-head証拠付きで検証。固定Baselineと既知Mutantを1件ずつ紐付ける。Referenceの局所PASSと製品資格を分ける。
- **A1**：#1180のv0 Scenarioを複製し、1 Stepを変更して既存4 CLI操作を実走。旧plan pinを拒否し、独立FixtureのActual/Expectedを照合。新CLIサブコマンドは作らない。
- **A2**：Track A/Bについて共通のCase/Source/Obligation/Oracle根拠/実行Receiptのread-only比較表を作る。JSONとScenarioの第二共通正本を作らず、未対応表現をunmappedとする。
- **A3**：§40 V-READへ接続。実Providerおよび完全性がなければProvider read qualifiedとは記録しない。Web/GUI visual/screenshotは後続のTarget-specific Gate。
- **A4**：異質な第二Domain、Build/Job委譲を段階拡張。上記の独立Oracleとcoverageの測定が成立するまでは広域テスト生成を保証としない。

### 41.5 業務Functionと実行・配備の物理構成を分離

Draft #2788と#1130に従い、**業務System/Subsystem/Function ID → 型付きInvocation Binding → Build Unit → Artifact/Deployment Unit → CLI/cron/Web/EKI Entry Point → 現在のHost Grant/Effect**を区別する。Go単一バイナリを既定とし、**一つのBuild UnitならFunction割当設定は不要**。必要になった場合のみread-only Build Preview→Owner明示採用→default/namespace/path/overrideの最小割当を検討する。Function一覧は既存Model/Bindingからの派生Viewであり、手入力で第二正本を作らない。

同一Functionを複数バイナリに含めても業務意味を複製しない。逆にビルドに含まれてもCLI/cron/Webが公開しているとは限らず、公開されても現在のPrincipalが実行できるとは限らない。MB-01〜09を既存#2788へ渡し、Scenarioの対象Operationと物理Entry Pointの版Pin、Goリンク後の実Artifact member、実行前Grant、unknown Effectを別列にする。正式Build割当構文、複数Buildの実運用、性能影響は未採用・NOT RUN。

### 41.6 EKIは必須Hostではなく、情報を欠落させない任意の委譲先

#2787で固定したEKI P0 OpenAPIの静的対照WP-01〜19は同PR fixed-head Node CIで19/19合格。しかしこれは**実Control Plane/Worker・Artifact-body・Host Effectの実証ではない**。現契約ではparametersがWorkerへ伝達されず、effect_modeはnoneのみ、executor.version_constraintはAdmissionで未評価、deadlineは単一、Artifact GETには501がある。複雑なTEI Jobを契約情報を落として委譲しない。

| TEI Jobに必要な条件 | 現EKI P0への委譲判定 |
| --- | --- |
| 純粋・副作用なし、固定InputとOutputの意味を損なわない | 実入出力配送・実行体版の確認後に限定試験可。静的shape一致だけでは未資格 |
| パラメータ・実行体Digest固定・成果物本文の取得が必須 | 現状不完全。黙って既定値に置換／成功扱いしない |
| DB Effect・冪等性・Commit/復旧の厳密な条件 | effect_mode=noneへ偽装せずnot-representableとして停止 |
| 最終検証まで含む業務期限・資源/データ所在の指定 | 単一deadlineやresource hintへ情報を落とさず、要件未表現をunresolvedとする |
| 軽量認知など低レイテンシ・マルチモーダル応答 | 現P0 Task APIの適合は未確認。同期直実行を比較基準にし、通信・配信方式をOwner選定前に固定しない |

EKI側#112/#266/#67/#115へ契約不足を返し、TEI #1130がJob意味と判定、EKIが配置/Lease/Attempt、Host/Providerが業務Commitを所有する。即応処理でEKI呼出しが遅いなら**分散しない**判断も適格な選択肢。将来の高速プロトコル/streaming/マルチモーダルはControlとData transport、deadline/jitter/backpressure/取消/順序/Frame timestamp・型付きPayload・承認済み開示の比較実験から決める。特定Protocol/gRPC/QUIC/WebSocket等を正式採用したとは記録しない。新しいEKI必須Host/TEI専用Scheduler/第二Task Ledgerを作らない。

### 41.7 提出可能な判定と未実施

- **確認済みの既存入口**：mainのScenario CLI/v0 Source、#2813 test-only JSON Draft、#2801 exact-head Go負例、#2787 exact-head静的OpenAPI試験（各Claimに限定）。
- **計画へ統合**：AI-first A0〜A4、AI-V-01〜08、Source raw Disclosure停止線、Function/Build/Entry分離、EKI lossy delegation停止線。既存各OwnerのIssueやSchemaを複製しない。
- **未確認/未実施**：#2813 `2219fc4d` のHealthはsuccess（限定test-only）だがAI実作成/独立Oracle/Target E2Eは未確認。#2812直前HEAD `7a6071ed` のHealthはsuccess、現HEAD `8b90b606` のHealthはin_progressで新CI未確定。Track A/Bの同一独立Oracle比較、AI生成の実測Coverage、製品Provider/Web/GUI/SHO、実Multi-build、実EKI Task/Worker/Artifact、低レイテンシ実測、Owner Meaning受理。これらを完了・採用・main統合と記載しない。

**変更の性質**：本§41はDraft #2667の横断計画へのdocs-only追加。#1119 Gate・§40 C0〜C4の証拠閉包、既存各WorkstreamのOwnerと正式Grammar/Protocol採用保留を維持する。


## 42. AI検証成果物の一式公開・受領資格と識別子境界（2026-11-04、§40–41への追補・Docs-only）

### 42.1 実装から確認された新しい反例

[Draft #2817](https://github.com/hat47x/tei-platform/pull/2817) は #2604 AP-01〜03 を現行Go実装へ適用し、WSL上のfocused Go testで次を観測した（PR本文に記録、repository-wide/exact-head CI・Windows・外部Consumerは別判定）。
- **AP-01**：R2の一部ファイル書込後に失敗すると、**新R2ファイルと旧R1 manifest**が並存。
- **AP-02**：R2で生成されないR1のファイルがディレクトリに残存。R2 manifestに載らないからといって物理的に消去されたわけではない。
- **AP-03**：Slug名衝突がループ内で発見されるため、エラーの時点ですでに一部の既存ファイルが置換済み。旧manifestのみ残る。

これは「書込み中断の実反例」だが、Producerが生成した各内容の誤り、外部SHOやAIの実消費障害まで証明したものではない。**§40 C3の『実Reader byte』だけでなく、『同一世代の公開済みmember集合』『Consumerが取得した一式』を明示**する必要がある。

### 42.2 互いに独立する5つの資格

| 資格 | 何を検証するか | 何を証明しないか | 所有 |
| --- | --- | --- | --- |
| 意味と定義 | Source/Canonical/Accepted Meaning、Scenario revision、独立Expected | ファイルの公開完了 | #988/#1180/#2514 |
| 生成物同一性 | Artifact raw bytes、Manifest/Member digest、Generator/Toolchain pin | 出力先の世代一式性・公開時点 | #2482/#2600 |
| 一式公開 | 完成Member全件、重複名なし、公開参照の一貫した切替、旧世代隔離 | Consumerが実際に取得したこと | **#2604** |
| 実Consumer受領 | Pin済みManifest→実member bytes、欠落/版混在拒否、用途別Disclosure | 現在のHost Grant・業務Commit | #2516/SHO/#1163 |
| 実行と効果 | 現在のPrincipal/Provider認可、Read/Write/Effect、独立post-image | 未観測の外部副作用・別用途での現在資格 | #1174/#1119 |

**識別子の扱い**：#2814/#2815のHashIdScheme・exactCharacters等は、用途・Scope・Scheme版を束縛したID導出の候補に限定する。Canonical ID、Artifact Digest、Scenario SHA、生成実行Attempt ID、公開世代・納品先での受理を同一視しない。表示用の短いハッシュ文字列からSource/Meaning/Authorityを推論しない。#2815のGo/Node/TSと限定SQLiteの試験成功はPoCの局所資格であり、現行IDや署名・公開方式の正式置換ではない。

### 42.3 #2604への受入拡張：新Publisherは先に作らない

初期の検証は既存 `tei artifacts -out` を**隔離された世代別の出力先**で実行し、前の世代と同じディレクトリに直接上書きしない。CLIの正式仕様をこの文書で変更したとはしない。独立Readerは次を検査する。

1. **発行前**：一式の候補を出力前に列挙し、重複出力名・予約名・許可外Path、必要Member欠落、ManifestとCheckSetの資格差を検査する。予め解決できない条件は未公開で停止。
2. **生成中**：途中失敗やキャンセル時の未完成世代を公開済みと呼ばない。旧世代は参照を維持する限り読取可能で、旧世代を誤って改変しない。
3. **公開時**：一つの世代に紐付くManifest/Member集合・digest・版が揃った状態だけを利用者へ見せる。OS/FSの原子的rename保証を**仮定せず**、方式A: immutable世代と検証済み参照、方式B: stage→atomic切替を#2604で比較。新Artifact Store/Registryは導入しない。
4. **受領時**：Consumerが取得したManifest/各member raw bytesと世代pinを独立照合。公開切替が読取の途中で起きても混合版を黙認しない。Directory内の余剰ファイルをManifest対象として勝手に読まない。
5. **利用時**：Manifest complete、Findingsゼロ、User/Owner Meaning、Disclosure/Host Grantは別列判定。公開成功から業務受理/実行可を推論しない。

**新しい比較ケース PUB-01〜06（既存#2604 APへ割付）**：
- PUB-01 / AP-01：部分書込の注入後にも旧R1 reader byte不変、新R2 publishなし。
- PUB-02 / AP-02：R2 member削除後に旧R1 Bが物理残存してもR2 readerは拾わない。
- PUB-03 / AP-03：`sales/a` と `sales.a` のslug衝突で公開済み一式が壊れない。Windows予約名・大小文字差は別Scope。
- PUB-04 / AP-04：Artifact complete=trueでもFindingありを「納品検収PASS」とせず、CheckSetもConsumerが取得。
- PUB-05 / AP-05/06：同Digest/別生成Attempt、R1→R2切替中の遅延Reader、A→B→Aで現在資格とbyte同値を分離。
- PUB-06 / AP-07：Operation Surface一式を読めても未受理Binding/現在Grantが無ければRequestしない。

**検証優先**：P0は#2817局所反例の復習と独立ReaderでのPUB-01/03、P1は世代分離方式の最小実証とPUB-02/05、P2はSHO・AI/Scenario・別FSの実消費。既存テストを新期待へ変更するのは#2604で方式採用後。途中失敗をsuccessへ変えるために旧manifestを自動書き換えない。

### 42.4 AI-firstテストとDelivery/Jobの結節点

- **#2811/#2813**：AIが候補Caseを追加する行為と、検証済み回帰Caseを公開する行為を区別する。候補はAgent/Run別の隔離Workspaceへ書く。Baseline・独立Expected・Known Mutant・Coverage Obligationは候補と同一変更権限に置かず、G0〜G4（#2811）を通ったCaseのみ明示採用する。Case IDや短いHashIdはOracleの独立性を保証しない。
- **#1180 CLI/Web/GUI**：Scenarioを正本としてCLI/Webの派生物を再生成しても、それぞれのArtifact setとTarget実行Receiptが同じ受理版を消費したか確認。CLI PASSからWeb Auth/DOM/Visual PASSを転記しない。
- **#1130/#2788 Job・#2787 EKI**：Task accepted/completedやoutput digestだけでは成果物本文の実受領・Providerへの適用・業務Commitは不明。実Artifact body未対応、入力/実行体版の不足時に無損失の委譲を装わない。
- **#2827 ETL/ELT・MFT候補**：原ファイル内容Hash、物理版、配送Attempt/ACK、受取側取込Unit、Domain Event/Business Effectを区別。送達成功はImport成功ではなく、Import成功も業務受理ではない。DM-01〜14は**未実行の圧力試験候補**であり、標準Data Operation・専用Relay Pluginの採用は未決。新専用Host/Scheduler/転送基盤を作らない。

### 42.5 計画・ADRの真偽を守る横断チェック（仕様の第二正本は作らない）

2026-11-04の[Issue #2820](https://github.com/hat47x/tei-platform/issues/2820)と[#2825](https://github.com/hat47x/tei-platform/issues/2825)は、Issueのclose/merge状態と説明文、提案ADR、所在不明の計画リンク・既存資料等の不一致を**基準版固定のsource-static監査**で検出している。これは文書鮮度の証拠であり製品E2EやOwner受理の証明ではない。過去のDraft/未着手という時点記録を現在のstateへ自動更新しない。

[Draft #2816](https://github.com/hat47x/tei-platform/pull/2816) のADR-0045〜0047は重複ADR整理の**提案中**で、採用済みADRと同列にしない。特にADR-0043/0044の外部Host／認証Plugin境界を尊重し、#2825 A-09の認証を「TEI Coreが必ず実装する」ような検収へ再導入しない。文書整合修正は既存Ownerの#2820/#2825/#2816/PR #2821/#2826に委ね、本文への二重修正を行わない。

**状態とOwner判断待ち**：本追補は設計計画のdocs-only。#2817の限定Go局所試験、#2815の限定PoC、#2827の文書候補、#2820/#2825の静的監査を区分して接続しただけで、PUB-01〜06の新しい一式公開/Consumer E2Eや新規Go/Windows/SHO/AI/EKI/ETL試験は**NOT RUN**。公開方式A/B、HashIdの製品採用、ETL/MFT Plugin、ADR-0045〜0047は未採用。Draft #2667と既存Gate/Issue責務を維持する。


## 43. 実消費入力からAIテスト・公開Artifact・Consumer実受領までの同一Attempt束縛（2026-11-04、§40〜42の実証入口を最小化／Docs-only）

### 43.1 最新の実証を受入単位に翻訳する

§40 C0〜C4、§41 AI-first G0〜G4、§42 PUB-01〜06は別々の新エンジンの設計ではない。**最も短い一改訂の実験**として、宣言されたSource、実際に消費したSource、テストCase、独立Expected、生成Artifact一式、Consumerが実際に受け取ったbyteと結果を関係付け、**各辺に欠落時の停止条件**を設ける。

- [Draft #2813](https://github.com/hat47x/tei-platform/pull/2813) HEAD `2219fc4d` は [Health #41744813520](https://github.com/hat47x/tei-platform/actions/runs/41744813520) completed/success。JSONで追加するPipeline候補11件を含む**test-only Reference**のCI成功。独立Oracle検出力・AI実作成・Driver/CLI-Web・業務Provider実験の成功ではない。
- [main統合済み #2819](https://github.com/hat47x/tei-platform/pull/2819) は`manifest.Source=audit.go` (D1)と`go build=main.go` (D2)が異なる場合でも現状Selectが候補を通し、出力binaryはD2由来なのにSource/Build Evidenceに宣言D1が残る**局所回帰反例**。ここで判定すべきなのは生成物の同Digestだけではなく**実Builderへの入力同定**。修正は既存#2595 EB-04が所有、反例の成功は修正成功ではない。
- [Draft #2812](https://github.com/hat47x/tei-platform/pull/2812) は選択parsed SourceとManifest原文の初回Load内局所Witnessをprivate保持。前HEAD `7a6071ed` の [Health #41742064273](https://github.com/hat47x/tei-platform/actions/runs/41742064273) はsuccess。ただし内部path等のmetadata開示が残っていたため、**現HEAD `8b90b606137d5d4602b57d8f0a318a783dbf4edd` では公開Describeアクセサを撤去し、詳細Source/Manifest/Path/Raw fingerprintをpackage privateに限定**。現HEADの [Health #41785251763](https://github.com/hat47x/tei-platform/actions/runs/41785251763) は照会時 in_progress（旧HEAD PASS転用禁止）。`FullInputClosure=false`を維持する。
- [Draft #2811](https://github.com/hat47x/tei-platform/pull/2811) §12に**DSL JSON候補／保存Driver Scenarioの2 Consumer × 正負各1例と独立Oracle**の最小Packetがある。直前HEAD `4bfd9f64` のHealth failureは計画Markdownのescaped backticksで、新HEAD `5ff66332` が修正候補（当該HEADのCIは別判定）。#2811はdocs-onlyであり実PacketはNOT RUN。

### 43.2 追加するのは証拠の対応列であり、新しいAuthorityではない

| 結節点 | 記録する最小証拠（既存Scenario/Receipt/Manifestへの対応） | 明示して止める条件 | Owner |
| --- | --- | --- | --- |
| I0 宣言Source → 実消費Input | 宣言Source ID/原byte fingerprint、実Reader/Builderが使用したbyte集合/physical identity、Provider/Toolchain・時点 | 宣言D1でbuild D2、Read closure不明、途中差替え | #2595/#2743/#2600 |
| I1 入力 → 生成Case | 固定Scenario/JSON Case ID・Revision、Contract/Plugin Pin、actual parsing/fixture byte、生成Agent提案と人間Reviewの差 | 不明Step、重複JSONキー、ID衝突、古いScenario/Lock | #2813/#1180/#2811 |
| I2 Case → Oracle/実Run | 事前固定Coverage Obligation・独立Expected・Known Mutant、実Result、tool/runtime/attemptの版 | ExpectedをActualから転記、弱い判別Case、unknownをPASS | #2514/#1163 |
| I3 Run → 公開Artifact | 全member、実File byte、Manifest/CheckSet、世代参照、途中失敗/衝突の拒否 | `-out` 新旧混在、member欠落、stdoutのみのcomplete | #2604/#2482 |
| I4 Artifact → Consumer | 実Reader受領byte・世代Pin・開示目的、Target-specific validation、現在の利用Grant | 意味は同じだがSource引用stale、raw/内部path漏洩、版混在 | #2516/SHO/#1174 |

**AI/Toolへ公開する証拠は最小限の非機密要約に限定する**。Importできることは内部path、raw hash、原文件数、Manifest本文等の公開許可を含まない。必要な実Sourceを使う際は現在のRead/Disclosure資格があるConsumerだけに、既存Ownerが受理した経路で渡す。private Witnessを利用するためだけの公開Accessorを増やさない。

### 43.3 具体的な一つのP0 Work Packet：2 Consumer × 2 Case

**固定Input**：同一main base/正確なPR HEAD、Workspaceと入力Sourceのraw fingerprint、Scenario/JSON schema revision、Operator/Plugin lock、Fixture、Generator/Builder version、適用Scope。これらを同じ試行票で固定し、各Consumer固有のTarget/実行環境は別欄とする。

**Consumer A：Go DSL Pipeline Candidate（#2813）**
- A+：許容Pipeline JSON候補1件を既存Go Referenceで型/Span/Pin等まで実走。AI候補追加はテストコードの変更を要さないか。
- A−：同じ入力から型契約不一致または重複JSONキーを1件導入し、適切に拒否。Baseline固定anchorを同じ編集で変えない。

**Consumer B：Driver Scenario CLI（#1180）**
- B+：既存6-Step Scenarioの範囲内でread-only/validationの許容Step1件を `scenario verify → generate → run → check`。独立FixtureとGenerated Planのpin、actual Observationを照合。
- B−：Scenario/World/Plugin Lock改版または認識不能Stepを注入し、旧plan再使用／黙ったskipが拒否されることを実走。

**独立判別**：A/Bの各Candidateの単なるPASS数を合算せず、対応する1個以上のKnown Mutantや手計算可能な独立Expectedを先に固定する。どちらも共通入力`Contract`と`Expected`を同時に誤改変した場合はG2不成立とし、通ったParser/Go testからMeaning Acceptanceを推定しない。

**Cross-boundary負例（事前に未実行と明示）**：#2819 D1→D2実Builder入力不一致をPacketのI0に差し込み、A/Bが局所PASSでも縦断`qualified=false`にする。Source pathだけ同一でsymlink/読取時差替え、意味同一だが原文が変わる場合は実物理Readと引用の再資格化を別列に置く。公開世代R1→R2に失敗した#2817 AP-01/03もI3を`unqualified`にする。この段階では本番DB/Host/外部AIへ副作用要求を送らない。

**評価レポートの行**：候補数、G0〜G4別合否、事前Obligation分母、Known Mutant検出、入力/Runner/Artifact/Consumerの各実byte照合、`NOT RUN`/unverified/unknown/not accepted、AI作成・実行時間と費用。単純なCase数より、反例検出力と誤資格化ゼロを最重要とする。

### 43.4 P0〜P3の選択、契約不足時は停止

1. **P0（先行）**：#2813 current exact-head Go/CIと#2812 current exact-head CIを再確認し、#2811 §12のWork Packet入力と既存#1180 CLI v0実入口を固定。Branch/Head間のtest PASSを足さない。
2. **P1**：A+/A−とB+/B−を、個別の安全なFixtureに対して実試行。保存Scenarioは唯一のTest SSOTであり、DSL Candidate JSONは別テスト表層の候補Sourceであって第二Scenario正本ではない。
3. **P2**：I0の#2819 EB-01負例とI3の#2817 AP-01〜03を合流させ、**縦断上のどこでunqualifiedが発生するか**を観測。新しい共通Runner/API/Registryで穴埋めしない。
4. **P3**：入力帰属・独立Expected・公開一式・実受領が揃ったConsumerからSHO/Webや実Providerへ段階的に拡張。Go単一バイナリを基本、実Sourceが未束縛ならBuild skip、EKI委譲、実EffectのPASSは保留。

**重要な停止規則**：Local test PASSやHead CI SUCCESSはそのHEAD/Claimの証拠のみ。Source/Providerの全入力closure、Read/Disclosure Grant、実Target Consumer、Host Effectが未観測なら`NOT RUN`または`unverified`を維持する。UnknownとFalse、not acceptedとnot runを混同しない。既存#1119 Gate、#1163 Evidence、#2514 Oracle、#2604 Publication、#2595 Source Binding、#2811 Test AuthoringのOwnerを変更しない。正式DSL Grammar/Artifact Publisher/ID Schema/EKI ABIをここでは採用しない。

**実施状態**：本追補はDocs-onlyで、新しいP0 PacketのGo/CLI/Web/Provider/EKI/SHO、Independent Oracle/Mutant Run、実Consumer受領を本変更では実行していない。上記既存PRの限定Evidenceを横断整理しただけであり、Draft #2667を維持する。



### 43.5 Source入場確認：validate成功と「対象を検査できた」を分ける

[#2822](https://github.com/hat47x/tei-platform/issues/2822) の再現では宣言0件のWorkspaceで `tei validate -json` が `ok: true` を返し、`model catalog` は標準定義13件を表示する。[#2823](https://github.com/hat47x/tei-platform/issues/2823) ではmanifestの未知のSource名、非対象拡張子、定められたファイル名との大文字小文字差で、`workspace-files` に表示されても実Loaderに届かない例がある。[#2824](https://github.com/hat47x/tei-platform/issues/2824) では `tei commands` の19件と契約文書の13件が異なり、各コマンドの `schemaVersion` や `world` の契約外扱いがDiscoveryで分からない。

これらは現mainに対する各Issue記載の限定実行観測であり、本追補で再実行した証拠ではない。**syntax/validate PASS ≠ Source completeness ≠ Scenario applicability ≠ business result PASS** として、AI候補の合格証拠を構成する。既存の `tei.driver-scenario/v0`、6-Step Scenario、`tei scenario verify/generate/run/check` は再利用するが、コマンドの存在と全経路の業務同等性を混同しない。

### 43.6 テスト入場から適用までの区別（新しい中央Registryではない）

既存CLI、Workspace index、Scenario、Target receiptを突き合わせ、各Attemptの証拠に次の異なる列を残す。

1. **Declared**：ManifestのSource root、期待するファイル/宣言、Accepted Canonical/Scenario revision。何を探すべきか。
2. **Discovered**：Source index/Workspace-filesで発見したファイルと拡張子・root。発見できたことはロードされたことを意味しない。
3. **Loaded**：実Parser/Loaderが採用したSource byte、宣言identity、診断、未読込理由。空集合は「読み込み完了した空」と「未探索」を区別。
4. **Applicable**：生成Caseの対象Operation/Operator/Rule/Target、Pin、分岐、Required Obligationが実モデルに解決されること。適用外のCaseはPASS分子へ入れない。
5. **Executed**：実行体が実際に読んだSource/Build入力と対象Case、exit、Result、Effect/Commitの状態。静的Source digest一致だけで実読込を認定しない（[#2595](https://github.com/hat47x/tei-platform/issues/2595)、main統合済みPR #2819）。
6. **Observed/Qualified**：独立Expected/Negative Oracleとの照合、Host/Provider現在Grant、実Consumerの同一世代Artifact受領を必要に応じて別票で確認（[#2604](https://github.com/hat47x/tei-platform/issues/2604)）。既知Mutant検出と製品価値受入は別。

**Coverage分母**は「要求されたObligation」と「現在観測できる適用可能Obligation」を別に保存し、unknown/partial/permission-denied/unsupportedをゼロ件に潰さない。CLI/Webで共通Scenarioを用いてもTarget固有の認可・Operation・Effect・結果/VisualのEvidenceは別。Required Stepを生成器が暗黙skipした場合、同等性PASSは不可。

### 43.7 Source/CLI事前確認の既存反例パケット（独立Expectedを先に固定する）

| ID | 操作 | 合格に必要な判別（現状の修正完了を意味しない） | 既存Owner |
| --- | --- | --- | --- |
| ADM-01 | manifestのみ、利用者宣言0件 | validate成功と「期待した宣言が読み込まれた」を分離。標準std 13件を業務宣言として数えない | #2822 |
| ADM-02 | Source名 `semantic`→`semantics`、未知root追加 | 読み込み対象・未探索・下流診断を区別。未知rootを単なる「業務テスト正常」にしない | #2823 |
| ADM-03 | `workspace-files` 掲載の `.yaml`、`.TEI`、`Application.tei` | DiscoveredとLoadedの差分を可視化。対象外なら事前に判別できる | #2823/#2571 |
| ADM-04 | `tei commands` と各JSON契約の照合 | `world` を契約済み `tei.cli.*` と誤認せず、modeごとの版・失敗形を限定Pin | #2824 |
| ADM-05 | `manifest.Source=audit.go`／実Build `main.go` | 検査byteと実入力の差をSource Witnessとして拒否/未資格扱い。旧実験の結果は自動転用しない | #2595/#2819 |
| ADM-06 | AIのJSON候補1件追加／正負例／重複キー／既知Mutant | 候補G0–G4と独立Oracleの採用を区別。Case増加だけをCoverage増加としない | #2811/#2813/#2514 |
| ADM-07 | 1 Scenario → CLI計画・Web計画再生成 → Target別観測 | CLI実行PASSをWeb Host/Auth/Business Effect成功へ昇格しない。未対応StepはCoverage欠損 | #1180/#2391 |
| ADM-08 | R1公開中にR2途中失敗／旧Artifact残存 | 実Readerが取得したbyteとManifest世代を照合。出力digestだけでConsumer受領を推定しない | #2604/#2817 |

最小P0は **ADM-01〜04 + ADM-06**。ただし現在のCLI/Loaderに不足する列は「未実装・未観測」と記録し、テストツール側で捏造したPASSにしない。P1としてADM-05/07/08を繋ぎ、既存Go/CLI・必要時Web/実Consumerと同一Attemptで負例を実行する。ADM-01〜04の契約追加・警告/エラー選択・schemaVersionは各Issue Ownerが決め、横断計画が先回りしてAPI変更を採用しない。

### 43.8 判定・Owner・停止条件

- **局所実行可**：AI生成FixtureのParseやOperator整合は、Source閉包が未証明でも対象を限定した候補試験として実施可能。ただし全Workspace/全業務のCoverage、Target同等性、適用資格のClaimは停止。
- **製品同等性の判定停止**：Required Source/Step/Branch/Target contractの未読込、unknown、version mismatch、読み込み対象不明、実行体入力不一致、独立Expected不在、Host/Provider Grant不明、Commit unknownのいずれかに依存するとき。単に回避・無視したものをPASSにしない。
- **根拠の書式**：`Scenario revision / Source declared-discovered-loaded / actual build input / CLI schema mode / applicable obligations / independent oracle origin / Target receipt / observed effect / missing-or-unknown / owner-accepted?` を既存VA-02と#1180の証拠票へ追記。閲覧資格のないRawは公開せず、許可されたProjection・fingerprint・隔離Fixtureを使う。
- **責務**：#2822/#2823/#2824がCLI入場証拠、#2595が実Source入力、#2811/#2813がAI候補、#1180がScenario/Target、#2604が公開・Reader、#422が価値比較の横断索引。第二Test SSOT、第二CLI契約、第二Evidence Engine、中央認可/Registry/Hostは作らない。

**今回の状態**：この節はGitHub上の各Issueの限定的な実観測と他チャットの既存方針を結び直した **docs-only計画**。ADM-01〜08を本作業で再実行しておらず、外部Harness/実Web/Provider/Consumer/Owner受理は **NOT RUN**。Draftを維持し、各Issueの採否・実装は所有者の判断に残す。


### 43.9 VA-04／VA-02／CP系列との対応表（既存OwnerとOracleを維持）

横断側の [TAI Draft #423／VA-04](https://github.com/hat47x/tai-pleroma/blob/docs/yi-product-value-scope-review-20261104/evaluations/VA-04-evidence-readiness-preflight.md) は **C1 Declared、C2 Discovered、C3 Consumed、C4 Asserted、C5 Eligible & Received** の5区分を既に持つ。上記§43.5〜43.8の6つの観測段階とは、以下のように**対応づけるだけ**とし、新規状態機械・二重SSOT・共通Gateを作らない。

| TAI VA-04 | 本§43の実証上の詳細 | 検証の限界 |
| --- | --- | --- |
| C1 Declared | §43.6 Declared、I0の要求Source/Scenario | 宣言があるだけでは存在・Read未確認 |
| C2 Discovered | §43.6 Discovered | Source index/Workspace-files列挙≠Loaderによる消費 |
| C3 Consumed | §43.6 Loaded、I0で実Builder/Interpreterが読んだbyteと区別 | Loaderのread確認≠Build入力閉包。後段Buildは独立Witnessが必要（#2595） |
| C4 Asserted | §43.6 Applicable/Executedと独立Expected照合、I1/I2 | 実行・Assertion件数≠独立Oracleや業務意味受理 |
| C5 Eligible & Received | §43.6 Observed/Qualifiedのうち現在Grant/実Consumer受領、I3/I4 | Parser-only候補試験ではC5のEffect/受領を必須にせず、該当Claimを限定 |

**反例の正本とIDを増やさない**：ADM-01〜04はVA-04のPF-01〜04をTEIで詳述したもの。ADM-05は#2595/#2819（CP-03）、ADM-06は#2811/#2813のAI-EXT-01／独立Oracle、ADM-07はPF-05／#1180（CP-08）、ADM-08は#2604 AP-01〜07の接合先。EKI固有PF-06（3値/4値の契約未決）、PF-08（時刻系）はEKI Ownerへ残す。CP-01〜08やPF-01〜08をこの節で再定義・採番しない。

### 43.10 最初の実行票：局所実証の継続と誤った全体合格の拒否

**目的**：最初からWeb/Host/外部Provider全部を組み込まず、同一の固定版に対する「局所テストは実行できるが、未検査を全体PASSにしない」を最短で観測する。これは [#2811 §12](https://github.com/hat47x/tei-platform/pull/2811) の2 Consumer × 正負各1件を置換せず、実行前の小さな確認票を追加するもの。

| 確認 | 最小入力/実行 | 独立Expected | Claim/後続の扱い |
| --- | --- | --- | --- |
| P0-A | 利用者宣言0件のWorkspaceで現行`tei validate -json`と`model catalog` | 標準`std`を利用者宣言に数えない。指定すべき業務Sourceが無いなら「業務モデル検査完了」ではない | 現行CLIの局所観測を記録。schema変更/警告方針は#2822 Owner未決 |
| P0-B | 既知rootの綴り変更、未知root、管理一覧に出る非対象拡張子 | `Declared/Discovered`の差と`Loaded`の不明・未消費を分ける | User Jobの完全性Claimはunqualified。Reader/diagnostic契約の採否は#2823 Owner |
| P0-C | 同一binaryの`tei commands`と対象コマンド実応答（複数modeを区別） | 実応答が認識できた版だけをPin。`world`を安定`tei.cli.*`契約としない | Mode/Failureを含む互換性は#2824 Owner未決。Discoveryから値を補完しない |
| P0-D | 同じfix済みSource/Scenario版で#2813候補と#1180 v0をそれぞれ正負試験 | Expected作成者・固定Known Mutant・対象Step分母をActual/候補生成者から分離 | 局所Parser/CLIのPASSに限定。Web/Auth/Effect/ConsumerはNOT RUNまたは未観測 |

各行に `exact repo/full SHA、PR head、actual binary/toolchain、Workspace/Source selector、fixture digest、Expected origin、対象Scope、実行コマンド/exit/JSON schema、正負結果、Observed vs Unknown、Claim ceiling、Owner` を残す。**外部送信可能な記録は許可されたProjectionのみ**で、source raw/path/fingerprint/hidden dataの公開許可は別途確認する。

**Claim別のGo/No-Go**：候補JSON追加、構文・型・Source位置の限定試験はC1〜C3やTarget受領が不明でも「局所・限定」と明記して前進可能。一方、未読込/未探索がある状態での全業務Coverage、Target等価、再利用可能なBuild/Artifact、実副作用成功の宣言は停止。資料不足を一律の全研究停止にせず、当該Claimが依存する範囲だけfail-closedとする。

**今後の順**：P0-A〜Dの再現（現行main exact版、各Ownerが保有する手順を再利用）→ #2811 G0〜G4の独立Oracle／Known Mutant実測 → #2595の実Builder byte帰属 → #2604の一式公開/Consumer受領 → #1180 Web/Host固有E2E。各段階の証拠を別版・別実行から無言で合成しない。新しい共通Reader/Runner、AIの自動Owner承認、EKI常時介在を前提にしない。

**状態**：§43.5〜43.10は計画上の統合・重複整理のみ。追加の実CLI、Go、CI、別Harness、Web/Host/Provider/Consumer、独立Oracle/Mutantの実行およびOwner受理は本作業では **NOT RUN**。改版対象の実装責務は従来のOwner Issueのまま。


### 43.11 Contract-crossing witness packet: 最小1件の生成AI正負対照を実CLIで止める（2026-11-04、既存§43.10を実行可能な照合票へ圧縮）

**新たな汎用Test Runnerや第二Oracle正本を作らない。** #1180の保存済Driver Scenario（同じ6 Step）と[#2811](https://github.com/hat47x/tei-platform/pull/2811) G0〜G4、[#2813](https://github.com/hat47x/tei-platform/pull/2813) AI追加可能JSON候補を接続する際、まず**同じ1つのRuleを通る正常系1件と、意図的に異なる期待値/実装を持つ負例1件**だけを、実CLI/Sourceを使う2経路で比較する。テストの件数でなく「誤った実装を検出できたか／未読込なのに検証済みと誤判定しないか」を優先する。

| Boundary | Producer → Consumer | Required independent witness / stop |
| --- | --- | --- |
| W0 Source selection | manifest・実Workspace → workspace-files・Parser/Validator | `declared roots` と物理`discovered files` と`actually loaded files`を別列に持つ。#2823の未知root/非対象拡張子を「読込み済み」に補完しない。実Loadedを取得できなければ`unobserved` |
| W1 Semantic candidate | selected .tei byte・Language/Role/Operator Pin → Semantic/Rule Candidate | byte digest・source span/Target Subject・Policy/Meaning版・Diagnostic、Issue #2822のユーザー宣言0件とstd builtinを混同しない。Valid=trueはInput completenessを保証しない |
| W2 AI Test Candidate | 同じScenarioの意味Role／実Data例 → AI JSON候補 | 生成されたExpectedと実装Actualが同じOracle由来なら`shared-origin`として採点保留。独立Expected/Know Mutantを#2514/#2811が管理。Known Mutant不検出時は「テスト拡張成功」としない |
| W3 Actual tested Target | Go built binary・real argv/mode → CLI Scenario/Function Result | #2824: `tei commands`の広告・実出力Schema・exit/warnings・`world`契約外・source selectorを別に確認。元Sourceの宣言未読込や実Build Input不一致があるなら限定PASSに留める |
| W4 Build/member | Builder actual consumed sources + generator/toolchain → ArtifactSet & typed entry | #2595: 宣言Source digest≠実Go生成器に渡した入力になる負例。#2600: World同値でもSource occurrence/negative dependency・Evid資格は別。#2604: 必須member欠落/Consumer実loadを検出 |
| W5 Invocation/Effect | CLI/Web/Host/EKI entry → typed Operation/Provider/Receipt → independently observed post-state | #1130: 非画面起動/Build/Jobの対応。#1174/#2598: 実対象のID・current Grant・CAS/Commit unknown。EKI Task completedやCLI`Committed=true`とDB Effectを混同しない |
| W6 Consumer/Disclosure | Result Artifact actual bytes→ Studio/SHO/AI/harness external Consumer | 届いたbytesとSource/Meaning/Artifact Revisionを追い、外部開示資格と引用Freshnessを確認。受信未観測やReader未対応なら`NOT RUN`で止める |

**代表的な独立の負例**：`source root=semantics`（誤字、実定義rootは`semantic`）、検証対象のBusiness Rule/テストに0件しか到達していないのに空Workspaceで`validate.ok=true`となる。このとき通常の`test pass`や`model catalog.entries=13`（builtin stdのみ）を根拠に「Ruleを検証済み」と表示してはならない。最初の自動生成Agentは、Diagnosticと未取得範囲を提示し、Ownerが受理した期待値を勝手に書換えない。別の正対照は、同じ版・入力・Operatorで実Dataが読み込まれ、既存独立Expectedに照合されること。両者を**同じ実CLI binaryとcommand modeで再現**し、宣言・実消費の差が明示されて初めて次段へ進む。

**P0実行Packet（既存Ownerの作業に割当）**：
1. #2822/#2823/#2824が持つ既存再現を**同一Go binaryの実argv/exit/JSONと現行main HEAD**で再確認し、前回検証版`97aba8146`の歴史的事実を最新環境に無断で転用しない。各Issueが修正/文書化方針を決める。診断未整備でも局所DSL候補テストは`W0-unverified`として走らせられる。
2. #2811/#2813/#2514は保存済6-Step Scenarioから正常・Known Mutant負例各1件を選び、**同一出所のAI出力をExpectedに採用しない**条件をまず確認。AIがJSONを追加してGoテストがPASSしても、独立Oracle/実業務Step/無漏れCoverageは別判定。
3. #2595/#2604/#1130/#1174の本物のBuilder/Artifact/Invocationを使える場合だけ、実消費バイト・必須Artifact member・実ロード・現在Effectへ接続。未実装なら`NOT RUN: concrete producer/consumer unavailable`。EKI #112/旧#2787のwire ABI試験をTask実行成功へ昇格しない。
4. **比較票を一つに保つ**：W0〜W6で`qualified / not-qualified / unknown / not-applicable`と各段のactual evidence URL/head/toolchain/input/independent Expected/Ownerを記録。*not-applicable*は全件検証済みでない意味ではなく「そのUser Jobにその経路を要求しない」ことを明記。見知らぬ依存をN/Aに分類しない。価値判断/Owner正式採用はTAI #422のVA-01〜04へ委譲。

**現時点の証拠**：#2822/#2823/#2824は現行実Go CLIで再現したとIssueに記録されている（main `97aba8146`、WSL Go1.27.0）。#2813固定HEADのGo CI PASSは候補DSL Test-only。#2811/#2667の計画追加はDocs-only。**本節の同一CLI／同一Scenario／独立Oracle／Build／Web／Host／EKI／Consumerの新たな実試行はNOT RUN**。独立の新しい合格主張、正式CLI/DSL Schema採用、Owner意味受理、実Host権限を含まない。


### 43.12 消費しなかった入力の証拠：Negative Dependencyと必要Sourceの未取得（追加受入補強）

#2822/#2823が示す根本問題は、単なる `validate` 終了コードでなく、**宣言されたSourceとParserが実消費したSourceの差分を、対象外/不存在/未探索/権限拒否/不明に理由付きで表現できるか**である。§43.6のLoadedだけでは、必要なファイルがどこかに追加された時に以前の「対象完了」が失効するかを判定しにくい。そこで#2600のnegative dependency／#2743のSource closureを、既存ADM-01〜03の**補助Expected**として接続する。新しいファイル追跡Engine、監視Service、Source Registryは設けない。

**必要なRead closure（設計上の検収列）**：
- **必要とされた集合**：ユーザーの要求・Accepted Model/Scenario/Targetが読込を必須とするRoot、Source種類、選択規則、関連version。Manifestに書かれていないものまで無根拠に必須としない。
- **探索の根拠**：Loaderが実際に適用したRoot/Selector/拡張子・ファイル名規則と探索境界、候補数、非存在/非対象/permission-denied/partial/unavailable。 `workspace-files` の管理対象一覧とは別。
- **実消費の根拠**：Parserが使用した実byteと型付き宣言、Source occurrence/必要なImported Source、未読込理由。外部Providerのraw/内部Pathは現在の開示許可が無いConsumerに出さない（#2812）。
- **否定依存**：あるRoot/Selectorで「現時点で必要なファイルが他に無かった」と主張するには、正確な探索版・対象・完全性・Scopeが必要。未探索のRootや想定外の新ファイルを空集合・不存在へ昇格しない。後からMemberが増える・Root名が変わる・Source移動する場合は再資格化を判定（#2600 IB-11/13）。
- **資格の上限**：必要Sourceの欠落や対象外の説明が未確定なら、限定Parser/DSL Testは局所PASS可能だが、全WorkspaceのCoverage、全Functionの検査、差分影響なし、Build skip、公開済みArtifactの再利用はunqualified。

**既存Fixtureを使う最小変形（新ID・新Oracle正本を作らない）**

| 対照 | §43.7で再利用するCase | 先に固定する独立Expected | 誤判定 |
| --- | --- | --- | --- |
| 利用者宣言0件と13個の標準Catalog | ADM-01 | `std`の定義を利用者Sourceとして数えず、要求集合が未確定なら`coverage denominator=unknown` | `validate ok=true`で業務モデル全件検査済み |
| `semantic`→`semantics`、または存在しないRoot | ADM-02 | 必須Rootと実探索/Loadedの差を区分。下流の参照エラーとRootの入場不足は別診断 | 未読込みのためエラーが出ない＝完全 |
| 管理対象だが規則外の`.TEI`／`Application.tei`、Root内の新ファイル | ADM-03 | 既存読込規則で除外されたか、要求上必須なのに未取得かを区別 | `workspace-files`掲載＝Parser消費済み、除外＝必ず欠陥 |
| Raw移動・コメント改訂・A→B→A、Semantic結果同じ | ADM-02/03と#2600 IB-11/13 | 内容同値/World FPとSource occurrence/引用・negative dependency・現在資格を別票にする | 同じ最終Fingerprint＝旧引用や全Inputの証拠再利用可 |

**採否の境界**：どのSource名/拡張子を許容し、未知Rootをwarning/errorにするか、CLIにどの個数を表示するかは#2822/#2823/#2824のOwner判断を保持。ここで新CLI JSONフィールド、診断code、ファイル種別を正式採用しない。P0はread-onlyの既存出力とIssueが観測したSource-static/実CLI反例の比較票、次段でfocused Go fixtureとexact-head独立Expected。広域Provider/SHO/Studioや実権限の製品資格は別Evidenceとする。

**横断の統合前提**：#2667のDraft作業ブランチとmainが分岐しているため、旧HEADや別PRの局所PASSを合算しない。新たな検収条件を採用するときは対象main、Owner Issue、既存ADR、CIおよび並行PRで重複した節番号・Issue参照を再確認し、Ownerの受理と統合を別判断にする。

**今回実施**：設計上の補助Expectedだけ。新しいSource探索・negative dependencyの実Go試験、実CLI再現、Build/SHO/Web/Hostの同一Attempt E2E、Owner正式受理は **NOT RUN**。


### 43.13 DSL／Driver AI候補の実装済み入口と独立Oracleへの昇格条件（2026-11-04、#2813/#2828の実コードを照合した受入差分、docs-only）

### 43.13.1 異なる二つの既存テストConsumerを正確に扱う

- **DSL Pipeline candidate / [Draft #2813](https://github.com/hat47x/tei-platform/pull/2813)**：`reference/dsl-language-plugin/testdata/ai_pipeline/*.json`から候補を読むGo test-only Reference。実Pipeline Parser/Lowerer/Composerによる型/Pin/Span等の限定検査であり、`2219fc4d` の [exact-head Health #41744813520](https://github.com/hat47x/tei-platform/actions/runs/41744813520) はcompleted/success。11 Caseが合格しても、AIが実際に自律生成したこと、独立業務Oracle、実DB/外部Host/全Source取得の証拠にはならない。
- **Driver Scenario candidate / [Draft #2828](https://github.com/hat47x/tei-platform/pull/2828)**：`reference/driver-scenarios/ai-candidate-order-validation.json` の2 Validation Step、`cmd/tei/ai_scenario_candidate_test.go`、追加手順READMEのtest-only 3ファイル。元の保存済み6-Step SSOTは変更しない。候補JSON glob、重複Scenario/Step ID拒否、固定Anchor保護、`verify → generate → check → pinned run`、Scenario改版に対する旧Plan拒否を既存`runScenarioCommand`で試験する。
- **実行方式の重要な限定**：#2828のGoテストは`runScenarioCommand`を**Goテストプロセス内で直接呼ぶ**。計画内の`argv`が`tei scenario ...`形式でも、実OS subprocessでビルド済み`tei` binaryを別起動したことにはならない。`generated-pinned`／`verified=true`は合成Fixture・当該Source/Scenario/Lockの限定結果だけを支持する。完成した実CLI binary、OS境界、Web E2E、Host Grant、実Providerを追加で確認するまでQualified claimを広げない。
- **資格**：#2828作成HEAD `71d9749c50b0b1cb95b3c2e07284ef56d8fbf772` の [Repository Health #41786950497](https://github.com/hat47x/tei-platform/actions/runs/41786950497) は照会時`in_progress`であり、製品Go試験の全体成功は未確定。他のSecurity/Studio/Web workflowの成功はScenario候補テストの成功を代替しない。#2811はdocs-only、#2828はtest-only、いずれもDraft・未統合。

### 43.13.2 「負例」を三種類に分けて計数する

| 反例の種類 | 代表例 | 実証できるClaimの上限 | 次の独立判定 |
| --- | --- | --- | --- |
| N1 業務拒否 | 不正な数量→`quantity_positive: not-satisfied` | 現行Validationが当該Fixtureを拒否した | Ruleの意味/数値境界は独立ExpectedとOwnerが決める |
| N2 テスト・契約拒否 | Expected改ざん、古いPlan Pin、重複Case/JSONキー、型/Operator Pin不一致 | Schema/Pin/Assertion等の防御が機能した | 必要な対象Population/Source/Stepを除外していないか |
| N3 故障検出力 | 正しいExpectedを固定したまま、実装/演算/Ruleへ**既知Mutant**を加える | 特定Mutantを正しく失敗として検出できた | #2514の独立OracleとMutantのOrigin・判別数/分母 |

**N1＋N2のPASSをN3のPASSへ合算しない**。特に#2828の`ExpectValidations`を意図的に誤変更したときの拒否はN2であり、生成器／業務演算の実装自体を誤らせたKnown Mutantの検出実験ではない。追加された候補JSON内のExpectedは、AIの候補であってOwner受理済みの業務Oracleではない。#2813の型違い等も業務数値のMutant検出と混同しない。

### 43.13.3 2 Consumerの最初の実行票（既存#2811 §12／#1180へ引継ぎ）

1. **同じ独立基準**：Case/Scenario identity、actual Input fixture、required Source/Rule/Step分母、Operator/Plugin/Scenario revision、Expected作成元を先に固定。DSLとDriverは異なるOperation/Meaningを検証するので、**同じ業務Ruleに到達したと主張しない**。現時点ではread-onlyの証拠列を共通比較するだけ。
2. **局所実装テスト**：#2813に`go test -count=1 ./reference/dsl-language-plugin -run '^TestAIPipelineCandidateCases$'`、#2828に`go test -count=1 ./cmd/tei -run '^TestAI'`を適用。PR別のexact HEAD、Go版、Fixture digest、実結果を保存。#2828のCI完了確認が最初の停止線。
3. **実Binary境界を別試験**：必要なGo版のclean checkoutから`tei` binaryをビルドし、`scenario verify/generate/check/run`を実OS process/argvと実出力で行う。Goのin-process test結果をそのまま実CLI subprocess PASSと呼ばない。現在のバイナリの`tei commands`/実JSON schemaとRequired/Loaded Sourceは#2822〜24で別照合。
4. **独立Oracle試験**：N3 Known Mutantを各適用Targetで一つ以上選び、Candidate Expectedを動かさず検出できるかを測定。テスト作成Agentが作ったActual/Expectedを共通に書換えてPassを維持できたならG2はunqualified。N1/N2/N3を別計数し、既知Mutant未検出をskipやn/aで隠さない。
5. **Source revision独立性**：[実Go PR #2681](https://github.com/hat47x/tei-platform/pull/2681)のコメントのみ変更によるWorld FP同一/ModelRevision相違を§44の限定Expectedとして参照する。意味計算の再利用可能性とSource引用/生成物/実受領の再資格化を別票にする。別HEADの組織Native PASSをScenarioの同一Attempt PASSへ合算しない。

**受入条件**：Goテストが通ることに加え、(a) Source/Step分母を独立に固定し、(b) N1/N2/N3のそれぞれの実行区分を記録し、(c) v0 Scenarioの未対応Operation/NULL/missing/decimal/実writeはunsupportedとして隠さず、(d) 必要なConsumerが実Readerで同じArtifact member byteを受領した場合のみそのClaimを許可する。A/Bどちらか未実施なら二者横断合格は`NOT RUN`。Web/GUI、実DB/Host、Owner Meaning Acceptanceは別Gate。

**追加しないもの**：Scenario新正本、共通実行Runner、Oracleの自動承認、第二CLI/Registry/Host、AIへの外部Provider原文・内部Path/Raw fingerprint自動開示。現段階の改訂はP0実証の採否条件を具体化した**docs-only**。#2813/#2828の製品・テストコードは本PRでは変更せず、実Go／独立Oracle／実Binary subprocessは今回NOT RUN。既存#2514/#1180/#2811/#2595/#2604のOwnerを維持する。


## 44. 組織Native E1でSource同一性と意味同一性を切り離した：既存Evidence閉包を実Go反例で補強（2026-11-04、§40–43／#2600へ引継ぎ）

### 44.1 同じWorldとOperationを得ても、同じSource改訂ではない

[PR #2681](https://github.com/hat47x/tei-platform/pull/2681)に実`.tei`コメントだけを変更した`TestOrgHeadlessSourceByteEditPreservesWorldButChangesModelRevision`を追加。[固定HEAD `c3695ffccdcea115d8445505552b3ae0777f782c`](https://github.com/hat47x/tei-platform/pull/2681)の[Native Actions #41786879258](https://github.com/hat47x/tei-platform/actions/runs/41786879258) **completed/success**。Go1.26.10/Linuxで実Workspace/Source Index/Operation Surfaceを通し、**WorldFingerprint同一、ModelRevision相違、Operation同一、Effect read-only**の四つを同時に検査した。別ReaderのArtifact検査は`findings=0, complete=False`なので、Artifact全memberや実Consumer受領を成功としない。

[PR #2801](https://github.com/hat47x/tei-platform/pull/2801)は、外部Importの`LocateImportedSource`による後段再解決が初回Loadの消費byte証拠にはならない実Go負例。#2681のコメント変更ケースとは**別HEAD/別試行**であり、両者を合算して複数Source同時Snapshot・実Build入力閉包を証明しない。

### 44.2 §40のC0〜C4と§43のW0〜W6へ追加する限定条件

| 既存段階 | 実Go負例から決める検収列 | 禁止する昇格 |
| --- | --- | --- |
| C0/W0 Source | `ModelRevision`=管理対象のSource path+byte指紋／Consumer実消費Source/Importの同一Attempt Witness・選択Root完全性 | World同じ→Source未改訂／再Resolve時byte=初回実消費byte |
| C1/W1 Meaning | 正規化`WorldFingerprint`とRole/Operator Pin、Owner受理範囲を別々に記録 | Source comment変更→Meaning変更、Parser成功→Owner意味受理 |
| C2/Read/Policy | Context selection、Query hidden deps、Provider Row+Field/Cohort、Policy root/axis、現在Principal・Grant版 | 同じRead JSON/Operation候補→現行Grant/必要Fieldの実Read証明 |
| C3/W4–W6 Artifact/Consumer | 実Builder入力、Generator/Toolchain、manifest complete・全member digest、現Consumer actual received bytes/現在開示 | `WorldFingerprint`一致→Artifact全再利用・SHO引用Fresh・受領済 |
| C4/W5 Effect | Host Request/Actual Target/Grant/Commit Receipt/Post-image | 同じOrg Role→旧Grant再利用、Job/CLI success→業務Effect |

**差分の最小化**：意味計算の再利用、Source引用の再資格、既存画面の再開示、実Provider再Read、Go Artifactの再生成、Host Grant更新は**必ずしも同じ無効化単位ではない**。#2600/#2516/#2582/#1163の既存依存・受領契約へ返し、第二Global Cache/Source/Policy Engineや全系再処理を導入しない。

### 44.3 AI-first足場で先に固定する独立Expected

§41のAI-first A0〜A6と§43 ADM-01〜03へ、組織Sourceコメント変更の正負対照を一つの**事前固定Expected**として提供する：
`same semantic World = true; same ModelRevision = false; same read-only Operations = true; currently authorized = unknown; real DB/Provider read = NOT RUN`。AIはテスト候補と入力変形を増やせるが、実行ActualをExpectedへコピーしない。Source root未読込、別外部Import、A→B→A、Role/Policy変更、部分Read等は**今回のE1で未検証**と区別し、既存#2811/#2514の独立OracleとCoverage分母を使用する。

**次の実優先**：#2586は合成Policy VP/VEではなく、実`.tei` CaseRecord→OrgUnitの明示Relationと実Providerが発行したRequest・現行適用Scope/Read Witnessを同一Attemptで束縛する。#2622/#2628に実Data Ops/物理施行点が無ければ`NOT RUN: enforcement-unavailable`。未受理Owner Ruleなら`not accepted`。安全性・実Disclosure・Actual Effectの実証を今回のE1 Native PASSに上乗せしない。#1119 P0を維持する。

**更新範囲**：この§44は現行コードと固定CIを調査して記録した設計・実験引継ぎ。独立E2E・新Provider/DB/Build/Studio/SHO/Host実験は未実施。

## 45. AI-first足場の固定HEAD結果と、次の一回の統合反証（2026-11-04、§43/44の実証状況を更新・独立Owner維持）

### 45.1 成熟した局所証拠と、統合でまだ欠ける証拠

同じ「AIがテストを増やせる」という要求のうち、**DSL Pipeline候補の拡張**（#2813）、**保存Driver Scenario候補の拡張**（#2828）、**Scenarioの結果Claim上限**（#2830）は、別のConsumer・別HEADとして作業が進んだ。これらを新しい共通Runnerや第二Scenario正本へまとめず、まず既存CLI/Go testの実結果を確認する。

| 既存作業 | fixed HEAD / 実CI | 確認できること | まだ主張できないこと |
| --- | --- | --- | --- |
| [Draft #2813](https://github.com/hat47x/tei-platform/pull/2813) | `2219fc4d4d28` / [Health #41744813520](https://github.com/hat47x/tei-platform/actions/runs/41744813520) **success** | AIがJSONでPipeline候補を追加し、既存Parse/Lower/Composeの正負例をGo testで検査する限定経路 | 受理済み業務Rule、独立Oracle、保存Scenario、Web/実Provider |
| [Draft #2828](https://github.com/hat47x/tei-platform/pull/2828) | **修正HEAD** `77baf7092ff5` / [Health #41850080590](https://github.com/hat47x/tei-platform/actions/runs/41850080590) **success**（旧HEAD `71d9749c` はgofmtでfailure） | AI候補JSONの既存Scenario CLI経路と、匿名Receipt型の整形のみを修正したHEADの正規CI合格。旧Go test/vet/validate PASSと整形FAILは履歴に分離 | AIが新しいCaseを実際に発見・実行した独立Trial、独立Expected・Known Mutant、実Provider/Web/全Source閉包 |
| [Draft #2830](https://github.com/hat47x/tei-platform/pull/2830) | 新HEAD `019da011c7d0` / [Health #41851323500](https://github.com/hat47x/tei-platform/actions/runs/41851323500) **success**（Go1.26.10/Linux。旧HEAD `38633584`も別CI success） | 保存6 Step全件のstatic Verify時`Verified/evidenceNature`、生成argvごとのPinned Run・限定Receiptを実Go回帰で要求。実ログでgofmt/vet/Go全テスト/Workspace validate PASS | 新規AI候補Caseの全件性、独立Oracle、実外部Binary/Web/Provider/Auth/DB Effect/Source Loaded closureは未証明 |
| [Draft #2812](https://github.com/hat47x/tei-platform/pull/2812) | `634ca10e1e5c` / [Health #41787598611](https://github.com/hat47x/tei-platform/actions/runs/41787598611) **success** | 初回Provider Load時の選択Imported Semantic原文/Manifestをprivate snapshotとして保持・後読可能な限定PoC | `FullInputClosure=false`、全Import/外部Provider、現在Disclosure・実Consumer・原子的Snapshot、許可されないRaw開示 |
| [Draft #2829](https://github.com/hat47x/tei-platform/pull/2829) | `5e33d81cf637` / [Health #41809158439](https://github.com/hat47x/tei-platform/actions/runs/41809158439) **success** | Evidence E1→E2→E1（ABA）で同Fingerprintへ戻る限定characterization。旧Reviewの適格/連続性をFingerprint一致から断定できない反例 | OS Source/Grantの変更、全依存/時系列閉包、正式Evidence資格判定とUser公開 |

上記のCI success/failureは**各PR head・各Claim単位**。#2812のprivate snapshot、#2829のABA、#2830の実行modeは互いに代替せず、合算して一つの実E2Eを構成したと主張しない。#2813/#2828/#2830のDraftがmainへ統合済みという意味でもない。

### 45.2 次の最小実行順と検収

1. **P0-Fmt（完了：修正とCI）**（#2828 Owner）：匿名Receiptの整形差分を修正したHEAD `77baf7092ff5` の [Repository Health #41850080590](https://github.com/hat47x/tei-platform/actions/runs/41850080590) は **completed/success**。旧HEADのgofmt failureは履歴として残す。これはAI候補JSONの受入経路に対する限定CI資格であり、独立Oracle・実Web/Providerの受入ではない。
2. **P0-Required（保存6 Stepの局所範囲で完了／動的候補・Target横断は未達）**（#1180/#2811/#2830／CI個別Owner）：#2830 HEAD `019da011` の実Go CIで、保存6 Stepの全ID・一意性・各`evidenceNature`とstatic未実行／生成Pinned実行の区別を固定した。次段では新AI Case／実Tool／Webなどの**必要`Scenario Step ID`・`AI Candidate ID`・`Target`の分母を事前固定し**、発見・適用・実行・Assertion・必須PASS・SKIP／NOT RUNを別列で示す。**保存6 Stepの局所PASSを動的Candidateや全TargetのRequired Gateへ昇格しない**。[main統合済み #2835](https://github.com/hat47x/tei-platform/pull/2835)では実PostgreSQL試験のSKIPでもGo exit 0となる例とrequired-PASS検査を実証済み（2026-11-04 merge、当該DB専用検査器をAI/CLIへ複製しない）。#2830の6-Step全件`evidenceNature`回帰は別Claimであり、実Web/Hostの必須試験を満たした証拠ではない。TAI #422 VA-TAI-01のH3 Skill loaded witnessも同じ`required ≠ advertised`原則を利用するが、TAI側の比較Checker/Ownerで判断する。
3. **P0-Holdout**（#2811/#2514 Owner）：#2813 Pipeline JSONと#2828 Driver Scenario JSONのそれぞれについて、**Candidate/Actualとは独立したExpectedとKnown Mutant**を1正例/1負例ずつ事前に固定。既存固定Anchorだけでは他のAI候補の業務Oracle独立性を証明しない。
4. **P1-Actual**（#1180/#1163 Owner）：各Draftを別HEADでの局所合格として記録した後、必要なら隔離された統合試験版を選び、**同じ実対象Source/Scenario/Toolchain/Runner版**で実コマンド・実入力・結果を取得。単なるプロセス内`runScenarioCommand`のPASSを外部実バイナリ／Web／Host経路実行へ転記しない。#2822〜24で必要Sourceが未読込ならClaim ceilingを限定する。
5. **P2-Read/ABA**（#2743/#2600/#2516 Owner）：Imported Source P0b-Aのprivate証拠を、required/actual Loaded closureおよび現在Consumer Disclosureとは別票に保持。Source/Meaning一致・digest ABAで過去引用や旧Grantが有効だと推定しない。外部Raw、path、fingerprintをAIへ無条件公開しない。
6. **P3-Target/Effect**（#1180/#1174/各Host Owner）：同一ScenarioからWeb/CLIのTarget固有経路へ進めるのは、必要なSource/Step・独立Expected・Actual Builder Input・現在Grantが揃った範囲だけ。実Commit・Post-state・Consumer受領は別Receipt。欠けるなら`NOT RUN`／`UNKNOWN`／`UNQUALIFIED`とする。

**完了の下限**：P0-FmtとP0-Requiredの**保存6 Stepに関する局所合格**は当該HEADで確認。次は**新候補Caseの必須実行証拠**と、P0-Holdoutの独立Expected／既知Mutant実検出を少なくとも1組、事前固定・同版で記録し、Observed Step/Contractと未観測Targetの境界を示すこと。**これより前でもAI候補追加は継続してよい**。ただしP1〜P3が未観測なら価値実証B1/製品Conformanceを「成功」としない。

**本節の実行水準**：既存固定CIログ・Source・Issueを再照合して計画順序を更新した**docs-only**。本変更で新規Go/CLI/統合/外部Host/SHO/Provider試験、独立Oracle・Known Mutant実測、Owner正式受理、PR mergeは **NOT RUN**。新Engine/中央Registry/実行Host/第二SSOTの設計採用なし。


### 45.3 局所CIから最初の実Harness比較へ進める最小Work Packet（2026-11-05、TAI #168/#422へ接続）

[#2828](https://github.com/hat47x/tei-platform/pull/2828)のJSON候補CIと[#2830](https://github.com/hat47x/tei-platform/pull/2830)の保存6 Step CIを**両方PASSしたからHarnessで実際にSkillが読まれた**とは言わない。[TAI Draft #270／SK-02 §15](https://github.com/hat47x/tai-pleroma/pull/270)ではCLIのCWD・Workspace引数の比較適格性を合成試験で検査し、[CI #41851651173](https://github.com/hat47x/tai-pleroma/actions/runs/41851651173)にSK-02 **60/60**、人工Node子プロセスのSK-03 **19/19**の限定PASSがある。ただし当該Runは当時のHEADからのmerge checkoutであり、後続TAI Draftの現HEADや実Harness四試行を証明しない。次の価値成果は負例を際限なく増やすことではなく、**公開TEI利用者定義1 Subjectの実Loaded確認からTool-only H2／Skill付きH3の同一Harness比較**へ進むこと。

[既存TAI初回Runbook §12](https://github.com/hat47x/tai-pleroma/blob/docs/yi-harness-skills-20261102/docs/yi-constellation/first-real-skill-qualification-runbook-2026-11-02.md)のまま進め、新Schema・第二Harness・中央Testerを設けない。

| 段階 | 実行・必要な証拠 | 不足時の判定 |
| --- | --- | --- |
| T0 Product baseline | **現在のTEI main**と実Go binary SHA、公開Workspaceのmanifest root、実model catalog/inspect/validateのCLI stdout/exit/diagnostics、同じ利用者定義Subject kind/idとSource位置を確認。workspace-files列挙・標準Catalogだけでは不足（#2822/#2823） | 必須Sourceが実Loadedか不明ならH2/H3 Task正答Claim停止。安全なProduct-only調査は継続 |
| T1 比較固定 | 同じSubject、Prompt、Harness、Model、Principal/Grant/Disclosure、実Binary、CWD、-workspace実argv、選択Rootと独立Expected。対象変更・Unknown Source・権限相違は別Group | 比較不適格／証拠不足。Digest・WorldFP一致を実Process同一性に代用しない |
| T2 Tool vs Skill | **H2→H3→H3→H2**の4独立Session。H2はSkill非露出、H3は同じ公開TEI Product Skill rootとrefs/scriptsの**実Loaded**および実Tool呼出しを観測 | 文書掲載・synthetic self-check・Agent自己申告を実Loaded/Invokeにしない |
| T3 独立判定 | 既存SK-02 Checkerは**記録形式／比較適格性**だけ評価し、別OwnerがSource/Warning/Unknownを含む独立Task Oracleで採点。正答・不適格・未知・Token/時間・人手を分離 | eligible-to-score ≠ 正答・機密非漏えい・Skill増分効果。4試行を全Harnessのp95へ外挿しない |
| T4 価値選択 | B0 Product-only／B1 Tool-only／B2 Product Skill付きで比較。品質・現在資格を落とさず追加便益がなければTool-onlyを維持し、必要時にだけSK-03・SHO/EKIへ進む | H3未ロードなら比較未成立。高度な共通Mechanismを先行採用しない |

**Source/意味/実効の分離**：§46と#2681のNative反例では、同byteのSource移動・A→B→A・Relation多重度変更でWorld/Operationが一致してもSource引用やPolicy/下流Read資格を導けない。H2/H3でもSource locator/Revisionを単なるWorldFPで置き換えない。初回は公開read-onlyだけを扱い、内部Raw・秘密Expected・許可されない外向き送信をHarnessへ投入しない。Web/DB/本番Host/低遅延EKI/SHO Writer/全面OS隔離は**そのTaskで必要な場合にだけ条件付き**。

**この節の証拠水準**：#2828/#2830固定HEAD CI、TAIの合成CI、#2823実CLI反例を参照し、既存Runbookへ試行の停止・受入条件を接合した**docs-only**。**T0〜T4同一Attemptの実Harness呼出し、今回の実Source Loaded再取得、独立業務Oracle採点、実Egress/Owner正式受理はNOT RUN**。比較実施OwnerはTAI #168/#422、TEI Source/Skillは#2822/#2823/#2605、Oracleは#2514/#2018に残す。



<!-- 20261105-product-real-source-first-cutover -->
### 45.4 設計候補から実利用価値へ移す一試行の退出基準（2026-11-05／実証順の再評価）

**優先順位は§45.3のH2/H3四Sessionから変えない。** 他チャットで成熟した[#2605](https://github.com/hat47x/tei-platform/issues/2605)の限定実CLIと[#2863](https://github.com/hat47x/tei-platform/pull/2863)のSkill状態訂正、[#2860](https://github.com/hat47x/tei-platform/pull/2860)の実Workspace→Source Index→Go Parser境界試験、[#2862](https://github.com/hat47x/tei-platform/pull/2862)のDB候補探索を、同一Subjectの異なる**試験資格**として整理する。

| 判断 | 現時点の確かな入口 | P0で確定する必要がある不足 |
| --- | --- | --- |
| **製品Source/CLI（必須）** | main [#2607](https://github.com/hat47x/tei-platform/pull/2607)と限定実CLI [#41532988114](https://github.com/hat47x/tei-platform/actions/runs/41532988114)は公開reference/workspaceのsales.order／関係をビルド済みGo CLIで読んだ | 同一AttemptのSourceの**実Parsed/Loaded**、Required Subject・必要なRoot/依存をW0の独立Expectedで確認。validate okとファイル列挙だけでは不可（#2822/23） |
| **Product Skill／Harness（最初の価値判定）** | #2863は実CLI限定成功と外部Harness未実施の表示を整理し、限定Skill CIは成功 | TAI #168 Runbookに従うH2→H3→H3→H2の**実4独立Session**。H3ではSkill本文・参照ファイルの**実Load**、両条件で実CLI Tool呼出し、Task Oracle/Unknown/費用を別確認 |
| **Role/自己記述（並行、Hard dependencyにしない）** | #2860の新Goテストは公開Workspaceのsales.order.total・Role文字列・TypeRef・Manifest版とParser版の分離に着手 | 最新HEADのCI PASSを確認し、parsed Role stringとRole Required Shape/Owner Accepted Meaningの間のGapを記録。実Role自己更新の成功とはしない |
| **Query/DB選択（次の限定read-only）** | #2862はData Context→DB Alias/Instance→Provider→SQL候補の設計とDBX-01〜09を保有 | Source/Roleの入口を保存したまま、初回選択と**固定済み選択の失効**（専用SQL削除・共通SQL無音fallback禁止）を既存Query Resolverのclosed fixtureで検査。二DB実Readは独立P1 |
| **Studio/Effect（条件付き）** | #1124はUI/Source Revision・現在Disclosure、#2853は別の実PG Command/CASを限定実証 | UI側二重Invoke対策だけをProvider冪等性やACK loss・Grantの証拠にせず、実Consumer/Action/Effectの接続が必要になった時だけ当該OwnerのGateへ |

**P0の退出は一本化**：公開で扱える一つのユーザー定義Subjectについて、実Binary/Source/CLI output/警告の一式と、独立Expectedで採点したH2/H3四Sessionの証拠が揃うか、Source-not-loaded／Skill-not-loaded／unsupported-tool／unknownなどの**具体的な不足を再現**できること。実行していないケースは不合格や成功と断定せずNOT RUN。Skillが有益でなければTool-onlyを維持し、第二Harness/追加Bridgeを要求しない。DB Variant、Role自己移行、Studio WebView、実更新の全件実証はP0退出の条件ではない。

**Scope境界**：同一Subjectは正確な試験比較単位であり、異なるPR・Runner・Targetの成功ログを同一Attemptへ合成しない。受入票の必要列はRequired／Observed／Undischarged／Claim ceilingのみ（§47）。現mainへの製品追加や新Runner、Registry、Host、各機構の強制パイプライン化は行わない。今回の§45.4は**設計・計画追加のみ**。新しい外部Harness実試験、最新#2860 Go/CLI E2E、#2862 SQL選択/実DB移行、Studio実UI、正式Owner受理は**NOT RUN**。

## 46. 組織Source差分の三次元反例をAI-firstと増分再利用の検収へ接続（2026-11-04、実Go E1／#2600/#2811/#2586横断）

### 46.1 今回の実証は新しいEngineでなく既存Source/World/Operationを使用

[Org Native PR #2681](https://github.com/hat47x/tei-platform/pull/2681)へ`TestOrgHeadlessSourcePathMoveChangesModelRevision`、`TestOrgHeadlessSourceABAReturnsToOriginalStateButNotHistory`、`TestOrgHeadlessSemanticRelationChangeIsInvisibleToCurrentOperations`を追加した。どれも実`.tei`を一時Workspaceへ複写し、既存Go `SurfaceWorkspace`（Loader/World/Source Index/OperationSurface）を通すもの。独立Expectedは本節/Org設計§62に固定した。

[HEAD `a36ca136957284a9af1dadc3fe3b40940d2f3269`](https://github.com/hat47x/tei-platform/pull/2681)の[Native Actions #41849999230](https://github.com/hat47x/tei-platform/actions/runs/41849999230)は**completed/success**。Go1.26.10/Linux、Go focused `^TestOrgHeadless`7件、実CLI validate/operations/artifactsと独立Reader、exact HEADとCleanを実施。Runner `findings=0, complete=False`なので、Builder/Artifacts/Consumer閉包のComplete資格は依然ない。全PR/mainのHealthとは独立の限定成功。PR open/unmerged。

| 変形 | Source Path/Raw identity | Semantic World | Derived Operation | AI/Build/Policyへの検収 |
| --- | --- | --- | --- | --- |
| コメント追加 | 改訂 | 不変 | 不変 | 出典/原文開示/引用の再確認は必要、意味再計算は省略候補 |
| 同byte SourceのFile名移動 | **Path改訂、byte同一** | 不変 | 不変 | Locator・Import参照/SHO引用は古い可能性。Cacheを全失効する必要はない |
| A→B→A | 初期と終点のState fingerprint同一 | 不変 | 不変 | **履歴不在の証明にはならず**、旧Read/Grant/Consumer再利用は独立 |
| Case→OrgUnit Relation`one→many` | 改訂 | **変更** | **不変** | Operation同値だけで関係のgrain/多重度/Policy/Query/SHOが互換とは言えない |

**重要な失効の非単調性**：Source Revisionは最終的な内容のidentity、実Reader/Host/Source受理のevent epochは別。A→B→Aで同じhashへ戻った場合も実Intermediate Rule/Grant Revoke、あるいは過去の外部Disclosureを元に戻せない。もう一方でSource移動だけなら、影響を受けない業務計算まで一律再実行する必要はない。#2600では`semantic recomputation`、`Source citation refresh`、`artifact build`、`Evidence reuse`、`Consumer authorization`の資格と実測latencyを別列で判断する。

### 46.2 既存Issueへの具体的引継ぎ：新しい名前付きGatewayは作らない

- **#2600 Incremental**：意味同値でもSource path/raw provenanceが異なる負例、A→B→Aの後着失効、Operation同値でも下流Query/PolicyへのRelation dependency変更という三種を、既存Dependency/CAS/Build Evidenceへ割り付ける。受入は一改訂実Builder Input＋ArtifactSet＋必要member実Consumer loadを確認してから。単にSource hash/Operation JSON一致でskipできたと報告しない。
- **#2811/#2813/#2514 AI-first**：今回の実Go Testは**独立Expectedを先に固定する**Mutation Corpusの候補。AIはCase追加・局所検証をしてもよいが、ExpectedをActualで上書きせず、既知Mutantの検出とCoverage分母を別評価する。実AI AgentによるMutation検出力の計測は未実施。
- **#2586/#2506/#2622/#2628 E2 Read**：Case`handlingUnit`の`one→many`で所属先RelationやQuery`EXISTS/COUNT`/Row grainが変わる例に、標準Functionの明示Data Context、実Provider Read/Route/Policy enforcement witnessを接続。現Native WorkspaceにCase Persistence/Query/実DBがないので**実行できない部分は`NOT RUN: provider-entry-unavailable`**。VP/VE合成CIを実Readへ置き換えない。
- **#2582/#2516/#2631 Consumer**：新しいWorldfp/Source locator・Role/Grant epochと旧Responseの資格を区分し、Display/Data reuseとSource citation freshness/External disclosureを分離。Source移動で全数再取得するのも、同じOperation JSONで改編前のGrantを保持するのも禁止。
- **#1119 横断Gate**：Owner意味受理、実Host Grant、Source/Provider Session、実Commit/Receipt/独立post-imageは今回のNative CIではNOT RUN。新全域Registry/Host/ACL/Query/Proof Engineなしで既存Gateへ戻す。

### 46.3 次の最小一試行（P1）と停止条件

**P1-ORG-READ**：既存`.tei`の`CaseRecord.handlingUnit`の1か所をモデルとして維持し、明示Function Readでscope内1・外1・同Caseで多重Org link1・partial1を独立Expectedに固定する。最初は実Provider Readの実行元・実Record ID・Field/read cohort・Policy rule・Consumer受領を**同じ実行Attempt**に束縛する。Providerに複数件のRelation path Readerがなく、Worker/HostでPolicy施行できなければ、`NOT RUN`を明示し、合成Referenceで実Provider成功としない。

**P2-ORG-STALE**：P1が成立した後、Source Move／Relation Cardinality／Grant Revokeのうち1つだけ変更→旧Read後着→可視性/引用/更新Intentを現在資格へ再評価。実Studio/SHO/AI受領・秘密原文の開示は当該Ownerの許可があるときだけ。新Driverや第二Reactive Engineを設けない。

**この節自体の証拠**：実Go E1の固定CIとSource/Workflow静的照合は成功。**実AI生成・独立Mutant検出・P1/P2 Provider実行・Query/SHO/Studio・実認可・Owner受理は未実施**。目的はE1の判別力を上げ、E2で未検証のままになっている部分を同じ受入票へ収めること。

## 47. 必須条件の欠落を局所PASSから区別する：自己記述Role・RunPlan・AIテスト・Harnessへの最小引継ぎ（2026-11-05）

### 47.1 新たに確認できた共通の不足

[main統合済み #2850](https://github.com/hat47x/tei-platform/pull/2850)は、確定後・補償EffectにObserverが無ければ無人RunPlanを止める限定実装。一方、実mainのinternal/execution/runplan.goにある **CombinePreconditions** は、渡されたnon-nil検査が拒否しないことだけを判定し、必要なPeriod/Grant/Target/Effect検査がすべて渡されたことを証明しない。EffectObserverの存在も実Observeを呼んだ証拠ではなく、Step Bのblockedは先行Step Aの無作用を意味しない。[#2616 §17](https://github.com/hat47x/tei-platform/pull/2616)のACU-01〜08がこの差を扱う。

[自己記述 Draft #2860 §6](https://github.com/hat47x/tei-platform/pull/2860)は、Source／Interpretation（Role・Operatorと推移Pin）／Meaning（非出力依存を含む）／Realization・Consumerという **Claim相対の必要閉包**とSDX-09〜15を設計済み。共通問題は「見えた検査はPASSだが、必要な検査が入っていたかが不明」という点である。

**既存Evidenceで突合する4つの列**：

- **Required（要件）**：Ownerの既存契約がその用途・Subject・操作に必要と定めたSource、Role/Operator、検査、Read、Effect、Consumer条件。AI自身の解釈だけで要求集合を確定しない。
- **Observed（実証）**：当該Attemptの実Source/実装版・Source span、実Tool/Runner、検証Check・Receipt・Consumer実受領。過去の異なるPR HEADのCI成功は同一Attemptの証拠にしない。
- **Undischarged（不足）**：Requiredにあり未読込・未実行・stale・partial・unavailable・permission-deniedな条件。**Required集合自体が未確定なら空集合としてPASSせずunknown**。ただしread-only Claimに不要な実DB Commit等は強制しない。
- **Claim ceiling（結論）**：局所Go/Parser/Check/Skill適格PASS、Ownerの意味受理、現在Grant、実Effect、Consumer受領は別の結論。必要条件が欠けた場合、観測済みの局所結果は残しつつ上位資格だけ保留する。

これらは**読取り専用の検収観点**であり、新たなRuntime Gate・共通Schema・Registry・Authority・Test Runnerを作る提案ではない。

### 47.2 既存Ownerの検証票へ渡す6反例

| Case | 局所PASSと未充足条件 | 必要な判定・Owner |
| --- | --- | --- |
| **OQ-01 Source欠落** | Workspaceのroot誤字・未解析suffix・標準Catalogだけでvalidate OK | 必須User SubjectのLoaded/Consumedなしならunknown。#2822/#2823/#1180、SDX-09 |
| **OQ-02 Role閉包欠落** | Role Descriptorは読めるが実Role/Operatorのexact実装・推移eager依存が未Pin | InterpretableとMeaning Accepted/Usableを別判定。#2620/#988/#2860 SDX-12/14 |
| **OQ-03 必須Guard欠落** | Periodが必要なのにEffectPreconditionsしかCombineへ渡さず、渡されたCheckは拒否しない | 安全に実行できると認定しない。**現実装が自動的に拒否すると主張しない**。#2616 ACU-01、#1130/#1174 |
| **OQ-04 Block前の部分Effect** | Step A resolvedの後、Step B blocked。Observer型はあるが実Observeなし | Plan全体effect=0・業務Verifiedに昇格しない。Step別Receipt/Unknownを要求。#2616 ACU-02/04、#1174 |
| **OQ-05 Required Test抜け** | 保存6 StepのGo CIは成功だが、新AI Case/Targetで必須Case欠落・SKIP | #2830の[exact Health success](https://github.com/hat47x/tei-platform/actions/runs/41851323500)を固定6 Stepの局所証拠に限定。#2811/#2828/#2514。実PGのSKIP緑拒否は[merged #2835](https://github.com/hat47x/tei-platform/pull/2835)の既存実装へ委譲 |
| **OQ-06 Skill/Consumer未受領** | Skill記述あり・合成Checker eligible／Artifact Manifestありだが実Load/Readなし | H3実Loaded・Tool実Invoke、必要memberのConsumer実受領・現在Disclosureを別証拠へ。TAI #168、TEI #2604/#2516 |

**最小の実行比較**：同じ公開Semantic SubjectとSource Revisionを固定し、Ownerが必須とする一条件について、正常に実観測された例と欠落した例を事前Expectedで対照する。合成CLI/Goの局所PASSが残っても、必要条件未充足なら上位Claimをunknown/unqualifiedへ落とせるかを確認する。Owner Accepted ExpectedはActualや生成AI候補と独立に固定する（#2514）。現Productにそれを示すWitnessが無い場合はunsupported/NOT RUNとして記録する。

### 47.3 優先順位を増やさない統合順

1. **利用価値を先行**：§45.3／TAI #168の公開TEI利用者定義1 SubjectについてT0実Source Loadedを確認し、既存H2→H3→H3→H2の4独立Sessionへ進む。OQ-01/06を既存Runbookの停止条件として再利用し、新Suiteを実Harness試験の先行義務にしない。
2. **安全な業務実行**：#2616/#1130の既存RunPlanでOQ-03/04の不足検査と途中Effectのcharacterizationを行う。実Providerの現在Grant/Commit直前Fence/Receiptは#1174の実作用境界で別途検証し、RunPlan自体を万能Authorityにしない。
3. **自己記述とBuild**：#2620/#2860が同一実Semantic SubjectのRole/Operator exact Pin・hidden dependencyを扱い、#2831は実Build graph/Seed/ArtifactSetを別の資格で検証。自己解釈の局所成功をセルフビルド成功に昇格しない。
4. **AIテスト採用**：OQ-01/03/05を#2811/#2514の独立ExpectedとKnown Mutant候補へ渡し、#2828/#2830の既存Scenario/Runnerを利用。新たな第二Scenario正本を作らない。

**今回のEvidence**：#2850 main統合とrunplan.goの静的コード確認、Draft #2860のSDX設計、#2830の固定CI、既存TAI SK-02合成CIまで。**OQ-01〜06の同一Attempt実試験、独立Mutant検出、実Harness/Skill Load、実Provider/Commit/Observer受領、正式Role自己解釈・全体セルフビルド、Owner受理はNOT RUN**。Draft #2667への計画追補のみ。#1119の既存P0/G0〜G4を変更しない。

## 48. 実PostgreSQLとEKI Suitabilityの現行証拠を横断の受入票に更新（2026-11-05）

### 48.1 局所実行を完了済みへ更新しつつ、正式資格を留保

| 証拠 | 実証済みの局所内容 | 未受理・未実証の範囲 |
| --- | --- | --- |
| [TEI #2853 main統合](https://github.com/hat47x/tei-platform/pull/2853) | 製品Command Invoker/Lifecycleから実PostgreSQL CAS、正常Commit/競合、模擬Commit応答喪失のEffectUnknown・再送禁止・同OperationID Receipt照会をopt-in実DB試験で確認 | 実ネットワーク切断、プロセス再起動、本番Grant・業務Owner受理 |
| [TEI #2854 main統合](https://github.com/hat47x/tei-platform/pull/2854) | 実PostgreSQLでA→B→A後に同idempotency keyのreplayは再適用せず、新keyでの再適用は **APPLIED** となるRCAS-01反例を確認 | Value CASだけでは論理Operation一度限りの保証なし。version/永続Operation identity/分散fenceの対策選定はOwner待ち |
| [TEI #2857 main統合](https://github.com/hat47x/tei-platform/pull/2857) | 暫定D8〜D11を通した一改訂Web/実PostgreSQL E2E、独立status read、鍵失効・物理Schema異常・業務拒否の負例を実記録 | [監査結果](https://github.com/hat47x/tei-platform/blob/main/plan/evidence/TEI_PROVISIONAL_CHAIN_E2E_20261103.md)は **INCOMPLETE / productGate=UNQUALIFIED**。provisional Acceptance/GrantはG1/G2正式証拠にならず、Trusted full health receiptも未収録 |
| [TEI #2859 main統合](https://github.com/hat47x/tei-platform/pull/2859) | #2857で見つかったRunner型storeの起動時S02物理Schema抜けを、共通schemaCheckで修正。standard/provisionalの実PG単位回帰とrequired CIテスト名を追加 | 実Webプロセス全体でHTTP公開が停止するかのE2Eは未試験。テスト用schemaCheck=nilのfakeはこの検査の対象外 |
| [EKI #714 main統合](https://github.com/hat47x/eki-fabric/pull/714) | ADR-0029 §2の3値を稼働実装に整合（§36.7） | ADR採用状態は提案中、dispatch Receipt実保存/表示と実測比較は別証拠 |

これらは異なるTarget/Runner/Revision/実行環境の証拠。main統合・テストPASS・同一文字列のReceiptを足して一つの正式Business E2E／現在Grant／Consumer受領に昇格しない。#2853の応答喪失はProvider wrapperによる模擬であり、実物理断線の耐障害性ではない。#2854のABAは**検出された弱点**であって防止策の受入ではない。

### 48.2 既存Ownerへ渡す次の二つの実反証

**P0 Web-Startup-Recheck（#1163/#1119/#2859）**：#2857で使用した一改訂・実PG・暫定D8〜D11の同種の使い捨てFixtureを、#2859統合後のmainで再構成。S02に必要なIndexだけ欠けたとき、Web実プロセスの**起動拒否とHTTP未公開**を独立Expectedへ固定する。Schema復旧後だけ起動し、同じAttemptで独立Read・provisional証拠・監査INCOMPLETE/UNQUALIFIEDを記録。新たな正式Owner/Host Grantがない限りG1/G2を合格させない。#2859の単位試験を実Webプロセス試験へ転記しない。

**P1 CAS-ABA-Unknown（#2564/#1174/#2845）**：#2854の同Key replay／新Key＋ABAを、#2853の製品Invoker・同OperationID Receipt lookupへ段階接続。永続Operation identity/版/Owner generation/Commitのどの条件で新Key再適用が拒否されるべきか、業務Ownerの独立Expectedと実PGの結果を区別する。Commit不明で別OperationIDを盲目的に発行しない。実restart/ネットワーク切断/本番Owner fencingはNOT RUN。

**補助の出口**：TEI #2856のCombinePreconditions合成は渡したCheckを実行するが、§47のRequired集合の充足は保証しない。EKI SuitabilityのNOT_JUSTIFIEDをLocal実行済みにしない。TAI [#434 main統合](https://github.com/hat47x/tai-pleroma/pull/434)の2文書に異種artifactを送る8件の限定試験も、今回のTEI一改訂正式Artifact/Consumer受領や外部Harness実Loaderの証拠ではない。

**この§48の変更水準**：実装PR・PG試験記録・mainコード・EKI現ADRの照合によるdocs-onlyの検収更新。ここで新規にP0/P1の実PG/Web/Host/EKI/Harness試験、Owner業務意味受理、G1/G2正式監査、性能/費用比較は **NOT RUN**。既存Source/RunPlan/Effectcommit/Admission/Consumer/Oracleの各Ownerを維持し、第二Runner・Registry・Hostを導入しない。

<!-- 20261105-db-dialect-role-migration-claim-ceiling -->
## 49. DB方言・SQL Asset選択と自己記述Role版移行を、同一Functionの証拠閉包で検証する（2026-11-05、設計・未採用）

### 49.1 追加の横断判断：標準化するのは意味と検収、SQL実装ではない

[Data Provider #2622](https://github.com/hat47x/tei-platform/issues/2622)にRDB-01〜08を追加。DomaのDB別SQLファイル（共通とPostgreSQL用の命名区別）およびtwo-way SQL、Spring JDBCのBinding/Error Translationという先行方式から、**TEIは薄いRealization Selectionと実Provider証拠だけを取り込む候補**とした。新しい共通DB Engineや2waySQL必須構文にはしない。

[#2860 §7](https://github.com/hat47x/tei-platform/pull/2860)はRole/Operatorの旧/新版A/B、Format Reader、Meaning比較、Owner受理とCutoverを分離する。同じLogical FunctionのDSL/Roleが新しいInterpreterでも読めたことと、Providerが異なるDBで実SQLを実行して元の業務意味・NULL/Unit/Scope/Current Grantを維持したことは**別資格**である。

**一つのSubject/Functionについて追跡する最小結合**：Accepted Meaning Revision／Role・Operator exact Pin／Function Data Context／実Data Source alias・DB kind・Provider能力／Native SQL asset選択＋原byte／実生成・実行されたSQLとBound Parameters／実Reader Scope/Snapshot／Query Result Grain／（書込みのみ）current Grant/Target/Tx/Receipt／独立ExpectedとConsumer受領。

### 49.2 DXR-01〜06の同一Revision対照票

| ID | 比較する圧力 | 正しいClaim ceiling／既存Owner |
| --- | --- | --- |
| **DXR-01 方言選択** | 同じTable Bindingでも二つのSource Alias、同一DB kind／別Dialect、明示Native Assetあり | Table名やCallerからDB名を推定しない。実対象Data SourceのResolved Alias/kind/capabilityと選択Asset digestを帰属。#2622 RDB-01/02 |
| **DXR-02 Common fallback** | `find-postgres.sql`が無いので共通`find.sql`を選択。ProviderでNULL/ORDER/Paginationの意味差 | 検証済みConformance範囲外ではunsupported。名前のfallback成功を意味互換性PASSへ昇格させない。#2622 RDB-02/03 |
| **DXR-03 2way SQL** | 開発時の直接実行SQLと、実行時にテンプレートからBound SQLを作った実SQL | SQL Source、Parameter/Identifier制約、実Output/Scopeを分け、SQLツールで開けたことをProvider実行・権限の証拠としない。#2622 RDB-05/#1174 |
| **DXR-04 Role移行** | Source A→B、Role/Operator同形だが丸め・NULL/Grain・失敗動作変更、またはB→A復帰 | Structure互換とMeaning互換／Owner受理／現在利用資格を分ける。#2860 SDX-19/22、#2620/#2514 |
| **DXR-05 Source欠落・実消費違い** | Workspace root未読込／Native Asset D1と実Build入力D2／実SQL経路不明 | `validate ok`、hash再一致、Provider内部成功のどれも入力閉包の成立ではない。#2822/#2823/#2595/#2600 |
| **DXR-06 書込/非同期** | Readは同値でもDB別PATCH/Replace/CASとafter-commit Observerの範囲が異なる | Query Result PASSからCurrent Grant/Commit/Receipt/実Effect観測を推論しない。RunPlanのObserver存在のみでもBusinessVerifiedにしない。#1174/#1130/#2616 ACU-02 |

### 49.3 既存P0の足を引っ張らない、最短の切り出し

1. **P0 read-only：先行する利用価値の証拠を維持**。§47の公開TEI利用者定義1 Subjectについて、実Source Loaded→H2/H3 Skill Tool呼出しと独立Expectedを先に成立させる。DB VariantとRole A/B全互換を初回のHard dependencyにしない。
2. **P0-DB：既存Resolverを静的／合成で反証**。同一の論理Query Intentに二Data Source Alias、共通SQL/方言SQL、明示Bindingの3素材を置き、Asset選択根拠と実Source byteを照合。適合する既存契約がなければunsupportedと示し、先にDialect Registry/APIを作らない。
3. **P1-DB：実2DBの同一Meaning比較**。まずread-only同じ独立ExpectedでPostgreSQLと第二RDB（候補SQLite）を比較。NULL/空集合、業務粒度、Rowset/Field Completeness、Version/Query Plan、取得Scope/Current Providerを異なる列で観測。Native SQLの選択が成功しても業務意味の同値を推測しない。
4. **P1-Role：既存A/B Interpreter受理試験**。#2860のSDX-16/17/19/21を独立Expectedで実Readerに接続する。実Role/Operatorが標準Referenceに未実装なら`NOT RUN: role-reader-unavailable`と表示。A/B Preview成功だけでProvider実用/Artifact Cutover資格を上げない。
5. **P2 書込とConsumerへ**：必要とされた時だけ#2598のAction Target、#1174の実DB Commit/Receipt、#2516の実受領へ接続。実RDB間TX/認証差やEKI/Jobの実行方式から業務Meaningを変更しない。

**配置とOwner**：SQL/Provider Capability/Realization=#2622/#951/#1121、Role Cutover=#2620/#2860、Source/Build実消費=#2595、同一Revision/部分取得・否定依存=#2586/#2582、独立Expected=#2514/#1180、Effect=#1174、Consumer=#2516。新DB Registry/SQL Parser/Authority/Host/Second Scenario Runnerなし。DXR-01〜06の実Go/実SQL/二DB同一Attempt、正式Role移行、Provider Write/Owner受理は**NOT RUN**。本節は設計・受入票であり、本番仕様の採択ではない。


### 49.4 SQL選択Lockの負の依存とRole実装版を同一試行で検収（2026-11-05、#2862／#2860追補）

[DB Query Draft #2862 §6](https://github.com/hat47x/tei-platform/pull/2862)のDBX-01〜09と、[自己記述Draft #2860 §8.5](https://github.com/hat47x/tei-platform/pull/2860)の実Go Fit/Gapを照合。両方の設計に重複する新しいResolverは不要であり、**最初のSQL選択**と**固定済み選択の失効・再選択**を別の行為として観測することが成熟した差分である。

| 変形 | 既存機構で追う必要なWitness | 未観測時の停止・担当 |
| --- | --- | --- |
| **DBX-01／S0 複数Alias** | Function Data Context→Persistence Mapping→DB Alias/Instance→Provider/Version、Read/Write/Scopeの実用途 | 同一Data/Table名から対象を一意化しない。#2611 PM-02/03、#2622 |
| **DBX-02／S1 初回選択** | 明示Binding／Provider専用／共通SQLのCandidate Set、専用候補の不存在、採用Source byteとQuery Contract | 共通SQLの存在だけでProvider適合・実読込・現在Grantとは言えない。#2622 RDB-02 |
| **DBX-02／S2 固定済み専用SQLの消失** | 旧Lock/Asset fingerprint、stale/missing原因、新候補集合・独立Expected | **自動的な共通SQL fallbackを禁止**。明示再選択・検証・必要な再受理までunqualified。#2600/#2862 |
| **DBX-03／S3 専用SQLの後着追加** | 『不存在』を条件とした負の依存、候補順位、実Build Input/Artifact Revision | 同じ共通SQL byteでも選択資格は失効しうる。無関係Artifactの全面再生成は要求しない。#2595/#2600 |
| **DBX-04/05／S4 DB版・Schema/RLS・Role A→B→A** | 実Provider版/Capability、Role/Operator実装Pin、Source occurrence/Review epoch、実SQL bytesとRead Scope | Source/World hash再一致でMeaning・旧Grant・Consumer資格を自動復活させない。#2860 SDX-19/22、#2829 |
| **DBX-06/07／S5 動的条件・2way** | 実Bound SQLとBind型/許容識別子、Required WHERE条件、実Read集合、元Source span | WHERE消滅による全件化、NULL/missing混同、例示literalのRuntime流入を拒否。#2506/#988/#2514 |

**一つの実証Packetの入口**：実TEI Workspaceから公開の利用者定義Logical Query/Subjectを一つ選び、(a) Source declared/discovered/loaded/actual consumed、(b) Role/Operatorが現行Productで本当に型付き解釈された範囲、(c) Data Contextに対する選択DB Alias、(d) SQL Asset選択理由・負の依存・実SQL、(e) 事前固定の独立Expectedと欠落時の診断、(f) 実行時Read/Consumer（実行されなければNOT RUN）を**同一Subject/Revision**で比較する。Role解釈を接続できない場合は`unsupported: product-role-bridge`として局所的なSource/Query選択PoCのみ認定する。新公開状態enumの提案ではない。

**実施順**：(1) #2862が既存DB/Query/Realization/Source IndexでDBX-01〜04をclosed fixtureで反証、(2) #2860が同一Subjectの実Source/Role/Interpreter資格を別受入、(3) #2514のExpected/known-mutantと#1180既存Scenarioに両者を接続、(4) 必要時だけPostgreSQL＋SQLiteのread-only Queryを実行しProvider差を確認する。#2853〜59の実PG試験は本件のSQL Asset選択・二DB互換試験を通したことにはならない。#1119の本来のG0〜G4やTAI実Harness最小比較を、新DBPoCの完了まで一律停止しない。

**状態**：#2862は設計と受入候補、#2860は実Goソース静的Fit/Gap、#2667本追補はdocs-only。DBX-01〜09、SQL選択/2way Go実装、Role実Interpretation、実二DB、独立Oracleの新実行、Owner意味受理は**NOT RUN**。新Dialect Registry／Query Engine／Role Resolver／Test Runner／Hostなし。

<!-- 20261105-lowcode-db-portability-acceptance-lux -->
### 49.5 DB可搬性は日常の操作を変えない――低コード利用者の受入（#2622 LUX-01〜07）

[Data Operations #2622](https://github.com/hat47x/tei-platform/issues/2622)にLUX-01〜07を追加。[DB Query Draft #2862](https://github.com/hat47x/tei-platform/pull/2862)のDBXは技術的なCandidate/Selection/Lock、[#2611](https://github.com/hat47x/tei-platform/issues/2611)のPMは実対象曖昧性、[#1124](https://github.com/hat47x/tei-platform/issues/1124)はStudioの投影。**同じテスト対象を複製せず、LUXは最終利用者の認知負荷と不適合説明の検収条件だけを担当する**。

- **普段の利用**：Data/Relation/Function/取得・保存を主語とし、DB kind・Dialect・SQL Variant・Go型の選択を標準操作の必須入力にしない。特殊なNative SQL・2waySQLは権限付きAdvanced機能へ隔離。業務意味の正本はSQL名でもStudio UIでもない。
- **DBの切替え・移行**：同一Business Function・独立Expectedを固定し、内部適合を`Native／Equivalent／Unsupported／Unknown`という**検収票の比較区分**で評価する（新Core enumの提案ではない）。日常利用者へは「利用できる」「同等結果を確認できた」「この接続先では操作に非対応」「実行可否を確認できない」を業務Function単位で説明し、SQLファイル選択を要求しない。
- **同等性の上限**：Query結果が同じでもNULL/decimal/Collation/Grain/Read Scope/partialが未確認ならEquivalentではない。保存操作ではさらに実Write Target/expected version/CAS/Commit Unknown/Receipt/現Grantを独立に確認。共通SQLが存在することや旧Lock一致から現在実行資格を推測しない。
- **失効・開示**：Source/Role/Operator/Provider/SQL Asset/Grant変更、候補追加・削除、A→B→Aで旧Previewの再資格化を行う。内部のDB Alias/DSN/SQL全文/権限外候補件数や不足Predicateは一般利用者へ無条件に出さない。Studio/Consumer Ownerの既存Scope/Disclosureを尊重する。

**次の試験順**：LUX-01/04/07は既存#2586のHeadlessと#1124のGeneric Inspectで通常操作の必須DB固有入力数（目標0）・対象不明時の業務言葉での診断・開示を測る。LUX-02/03/05は#2862 DBXの既存Fixtureと同じIndependent ExpectedでPostgreSQL＋第二DBを比較。LUX-06は#2600の再資格化で扱う。AIによる可読性評価は比較補助であって、実ユーザー操作・業務Owner受理や同一Attempt実DB効果のPASSに換算しない。**LUX-01〜07、実UI、二DB移行のE2E/ユーザー受入はNOT RUN**。新Registry、Query Engine、専用UI Pluginは追加しない。

## 50. 組織CaseのE2 ReadからE4 Commandまで：SQL/DBの選択と再実行安全性を別資格にする（2026-11-05、#2386/#2586/#2862/#2853/#2854 の横断引継ぎ）

### 50.1 進展した現物・未接続区間を精密に分ける

[Org #2681](https://github.com/hat47x/tei-platform/pull/2681)の実`.tei` Loader/OperationSurface（固定Go CI 7件成功）は**E1**。CaseRecord`handlingUnit`の多重度をone→manyと変えると**Worldは変わりOperation ID集合は同じ**という負例がある。

別Ownerでは[PR #2852](https://github.com/hat47x/tei-platform/pull/2852)の実製品Command Lifecycle（unknown/takeover）と[PR #2853](https://github.com/hat47x/tei-platform/pull/2853)の製品Invoker→実PostgreSQL CASがmainへmerge済み。[PR #2854](https://github.com/hat47x/tei-platform/pull/2854)はValue ABAで新Idempotency Key再試行による二重適用可能性を**実PGで反証**。ただし**Orgの一つのCase/Read/Host Grant/Commit試行を実施したものではない**。

[Draft #2862](https://github.com/hat47x/tei-platform/pull/2862)のDB別Query/2waySQL選択、[Draft #2860](https://github.com/hat47x/tei-platform/pull/2860)の実WorkspaceベースRole自己記述、#2822/#2823のWorkspace必要Source不在診断を、組織E2への**前提と独立証拠**として照合する。Role Descriptorが読めた、Alias名が読めた、`tei validate.ok`、Source/Operation hash一致、SQL解析が通った、Providerが存在する、の各事実はそれぞれ**別段階の資格**。SQL資産の存在から業務Meaning・実Read Grant・Host Commitを導出しない。

### 50.2 一つの業務Functionに必要な最小実行パケット（新Schemaではない）

| 既存担当 | 最初の実取得Caseの最小入力・確認する事実 | 未充足ならどうするか |
| --- | --- | --- |
| #2822/#2823/#2860 | 実に必要な`semantic/organization/model.tei`、Case Data/Relation、実ParseされたRole/Source Selector、Pin/Owner候補 | `std`だけの空WorldやManifest列挙を合格にしない。未取得／未選択を診断 |
| #1126/#2386 | Case/Org/Relation occurrence IDとCardinality、組織軸・valid/known・Policy意味のOwner受理 | 候補構造は保持するが`not accepted` |
| #2506/#2622/#2611 | Functionが必要なRelation path/Row+Field/Scope、宣言済Alias→Environment Instance→Provider/Route、対象の選択 | Binding不明・複数競合→`unverified`、暗黙最初のProviderへfallback不可 |
| #2862/#2387 | Native SQLが必要な場合だけQuery Contract/SQL Source/Dialect/Schema/条件分岐と選択の固定根拠 | 専用SQL不在・未知Dialect/Versionで未承認Fallback不可。標準ReadはRelation-first |
| #2628/#2586 | 実Providerが送信・観測したQuery/HTTP/Relation path、Rowset/Field/Cohort、選択Tenant/Policy施行点と実Result | 実アクセス未知なら`NOT RUN: physical-read-unavailable`。部分0件≠不存在 |
| #2582/#2516/#2631 | 消費時点のPolicy/Grant/Selection/Source、実表示・件数・Export・AI/SHO用途別資格 | Read OKからCount/Export/外部開示の許可を推論しない |
| #2598/#1174/#2564 | Current Host Grant、Operation/Target/expected version、実Invoker Lifecycle、PG CAS/unknown/takeover/Receiptと独立Post-image | Effect unknown≠失敗確定、同Attempt自動再送禁止、Source/値ABAでGrant/Receiptを復活させない |

これは既存の`Function Context`、`Query`、`Provider`、`Command`、`Receipt`の実証票であり、共通万能Context Object、新SQL Selector/Policy Engineや中央Evidence Registryを新設しない。外部Sourceのraw byte/DSN/Private FieldをConsumerの用途資格なく露出しない。

### 50.3 本当に一回で測るべき試験をReadとWriteに二分する

**P1／最初のRead-only E2**（唯一の今回の優先実装候補）：実`.tei`のCase`handlingUnit`に対し、同一FunctionでScope内C1とScope外C2をReadし、Cardinality one→manyを**Relation occurrence数／Case数／Query count**へどう反映するか、独立Expectedで比較。Provider選択の実DB Aliasと現在Scopeを固定し、必要Row/FieldとQuery非出力依存を実Trace/Resultに突き合わせる。利用者の日常DSLにSQL選択を強制しない。実Producerが無ければ`NOT RUN`で止め、E1/VP/VEのGo/Node別Runから合成E2 PASSを作らない。

**P2／同じCaseのWrite E4（P1成立後）**：C1の担当部門D1→D2、予期版7→8を独立Expected（架空テスト条件）として用意し、Current Principal/Host Grant/Typed Intentと対象Relationを再検証したうえで、既存Command Invokerと実PG Providerを再利用。commit応答喪失→unknown/再送禁止/承認済みtakeover→独立Read/Receiptを記録。続けて値ABA（D1→D2→D1）と新Idempotency Keyの負例を**同じOrg Case**で観測した場合にのみ「Org E4」を評価する。CaseRecordに実Persistence/Provider/Hostが未接続なら、既存PG`sales.order`成功をその代用にしない。

| 固定Trial | 反証対象 | 判別に必要な独立Expected |
| --- | --- | --- |
| ORG-XR-01 | 実`.tei`が見つからず、std Catalogだけでvalidate成功 | 必須Case Data/Relationが実Parseされたかを識別。検証対象が0件なら未Covered |
| ORG-XR-02 | データの所属先が1→複数、Operationは同じ | Case ID distinct、Org Relation occurrences、Scope/Count/Join grainの違いを返す |
| ORG-XR-03 | 複数DB Alias、専用SQL消失、共通SQL残存 | 固定DB Instance/Provider/Asset、承認済みFallbackの有無、実Readを確認 |
| ORG-XR-04 | 完全0件 vs partial0件、Query`NOT EXISTS` | Rowset/Field/Scope/Source completeなら偽と判定、未知ならunverified |
| ORG-XR-05 | 旧Source/Policy/Grantで開始したReadが遅れて到着 | 現在のRead/Display/Count/Export資格がないものは採用しない |
| ORG-XW-06 | Read OKから旧ContextでCase移管Update | Current Host Grant/Target/Versionが違えば送信しない |
| ORG-XW-07 | commit応答不明／同Attempt再送／takeover | unknownから同世代再送せず、別owner/genとOperation identity/Receiptを検証 |
| ORG-XW-08 | Case値のA→B→A、別Idempotency Keyによる再適用 | Value一致だけでは履歴・Idempotency成立を保証しない。独立Read/Receipt/row versionが必要 |

**段階の誤算を避ける**：`one→many`が実Parserで受理されたことは、現在のCase schemaがそれを業務意味として受理したという証拠ではない。Role/Owner Meaningも、SQLRealizationと実Providerの同値性も独立。PostgreSQL CASの参考試験は実DBを走った事実を保持するが、Org Case向けのテストは**まだ走っていない**。

### 50.4 計画の進め方

最初に#2586が**P1-ORG-READの実入口と不足情報**を列挙し、#2622/#2628がRelation pathの物理Read ProducerとTraceを確定する。#2862は必要な特殊SQLだけをOwner固定した選択へ適合させ、標準Relation/Function MappingをSQLへ置換しない。#2822/#2823/#2860は必要Source/Role/Languageの実読込・解釈と空Workspaceの誤合格を診断する。#2811/#2514は上の独立ExpectedとKnown Mutantを既存Scenarioで使うが、テストケース数/AI一致を製品保証の指標にしない。

P1が未成立でもP2の別Function実PG実験は独立継続できるが、**組織モデルとしてのE2→E4の接続が完了したとは判定しない**。同じCase・同じSource/Role/Provider/Owner受理版・Host現在資格の一Attempt証拠が揃ったときに初めて縦断を評価。#1119の本来のP0-a/P0-b優先は維持する。

**この追補の検証水準**：既存実Go/実PostgreSQLの別PR証拠を確認して設計・受入計画へ反映したもの。今回新しい組織DB、Query、実Host、AI自動テストやCLI/Web統合を実行したわけではない。Org E2/E4本番資格・Owner正式受理は**未完了**。

## 51. 適正規模のPlugin Bundleと未使用要素の非干渉：Sourceから実Go Binaryまで（2026-11-05、組織Profileの実E1から一般的Build/Providerへ引継ぎ）

### 51.1 新たな抽象層を追加せずに「大プラグイン主義」を受け止める

Organization Profileは、凝集したStandard Data/Relation、Hierarchy、Policy、Capacity、Query/DB Provider、Target Generator、UI/SHOなどのモジュールを**適切なボリュームの配布セット**として扱う。ただし`bundle`の大きさと、内包各部品の凝集度／結合度／実利用を別に評価する。基本Interfaceを関連実装と同じPluginに同梱してもよいが、外部Provider/Targetから利用可能なContractと更新範囲を維持することが条件。

**実コードの重要な事実**：main `internal/plugin/model.go`に`Definition/Bundle/ResolvedSet/Capability`、`internal/plugin/resolve.go`に`Catalog.ResolveEnvironment`が既存。`Bundle`をEnvironment選択すると、その`Plugins`に列挙された全Requirementが解決対象になる。**選択済み/Resolved＝Functionが実使用した＝Binaryへリンクされた＝Host実行可能**ではない。Bundle選択だけに従ってProvider/GeneratorをすべてBuildへ反映するなら、未使用の機能が成果物へ影響しない原則を満たせない恐れがある。ただし現行`ResolveEnvironment`が全員を選ぶことと、実Builderが全員を最終出力へ含めることは**別検証**。この境界を越えて現在の実装が不適合と断定しない。

### 51.2 11月5日の実E1限定検証

[組織 Native PR #2681](https://github.com/hat47x/tei-platform/pull/2681)へ、未宣言Plugin rootとModule Sourceの非干渉を検証する`TestOrgHeadlessUndeclaredPluginRootDoesNotChangeSemanticOrOperations`を追加。元WorkspaceのManifest `sources: semantic, applications, databases`を維持し、`plugins/unused-org-profile/plugin.tei`に**あえて構文不正ファイル**を追加して実Go Loader／Source Index／OperationSurfaceを再実行した。

HEAD `e0cfebf14d40db81bdef77d6c3e4898ed75d999f`に対する[GitHub Actions #42031418190](https://github.com/hat47x/tei-platform/actions/runs/42031418190)は**completed/success**（Go1.26.10/Linux、focused `TestOrgHeadless`8件と実`tei validate`・`operations`・`artifacts`＋独立Reader、exact HEAD/clean）。**ManagedFilesに未宣言Rootなし、WorldFingerprint／ModelRevision／Operation集合は元と一致**。Reader `findings=0, complete=False`に変化なく、実Artifact完全性・全Target・Go Binary非含有は未実証。

**仮定しないこと**：Catalog登録済み／Environment非選択PluginのModelRevision不変、Environment選択済みBundle内の未使用Module非収録、Go LinkerのDead-code eliminationが個々のPluginを確実に除去すること、Target依存がないPluginの変更がBuild Input/Artifact bytesへ完全に影響しないこと、Pluginの提供Capabilitiesを実Principalが使えること。今回証明したのは**Manifest外Sourceが読まれない**という一段階だけ。

### 51.3 既存Ownerでつながる選択・Build・実行の受入票

| 契約 | 原則と測定対象 | Owner |
| --- | --- | --- |
| Source Inventory/Parser | Required/Discovered/Loaded/Indexed、Source raw/locator/manifest/Import、必要Root未探索とoptional root欠落を区別 | #2822/#2823/#2860、#2600 |
| Plugin/Bundle Resolution | Catalog、Environment選択、Bundle展開、requires閉包、Lock/Contract/API版、曖昧提供とCycleの拒否 | #951/#2619 |
| Function/Capability Usage | Organization Case/Policy/Capacity/SHO/SQLRealization/Providerなど**使用されたCapability**と責務/Dependency | #2506/#2622/#2386/#2862 |
| Target Build Intent/Inputs | 使用集合→Generator/Module/Go Source/Toolchain/Build Unit、必要時のみ複数Buildへ対応 | #1130/#1167/#2600 |
| Artifact/Consumer | Artifact set/member/digest/complete、同じ入力で再現、実Consumerが受領したbyte/不足member | #2604/#1163/#2582 |
| Exposed/Authorized/Effect | CLI/Web/JobのEntry露出、Current Principal/Grant、Provider Row/Field/Read、実Command/Receipt | #2598/#1174/#2564/#2516 |

ここで**利用者が選ぶ**のは業務Function・Data・表示/実行目的であり、標準参照でGo Module名やSQL Dialectを毎回選ばせない。Studioは必要なCapability/Providerの不足を業務語で説明できる。内部のDB Alias/秘密Data/DSN/未許可Pluginを同じ説明へ露出しない。

### 51.4 受入用の一連の変更と最小判定

一つの組織`.tei`を正本に、**Org list/read only**と**Org policy + PostgreSQL-specific Query/Report + optional SHO generator**の二つのTarget/Function Build Intentを比較する。前者を先に実行し、未使用モジュールを追加・改版しても**そのTargetのUsed closure／実Build Input／Artifact member**が影響を受けないか、同一Toolchain・Build Flags/GOOS/GOARCH/ロック・Source版で対照する。結果のBinary digest/size、build latency、必要なEntry/Resource、修正時の再Build対象を比較し、出力が同じという期待をテスト前に固定する。初期の単一Binaryと運用上必要な二Buildは#1130に委譲し、業務意味/Function IDは変更しない。

| Trial | 変形 | 期待する差分とNG |
| --- | --- | --- |
| BP-01（実E1） | Manifest外に不正Plugin Sourceを追加 | **PASS：Parsed/Indexed/World/Operation不変**。Binary非含有は測定外 |
| BP-02（未実施） | Catalogに未選択Pluginを追加／版更新 | 未利用Targetの必要閉包は不変。Global Source Indexが変化したことだけでTarget再Build必須にしない |
| BP-03（未実施） | 選択Bundleの未使用Generatorだけを改版 | `ResolveEnvironment`の選択範囲とTarget Used closureを別票にし、Go Build actual inputs/Artifactへ混入したら理由を分類 |
| BP-04（未実施） | 実際にOrg Policy FunctionでGeneratorを使い始める | 該当Targetだけ必要Closure/Artifactが変わる。Case Semantic ID/Owner Meaningは別 |
| BP-05（未実施） | 選択DBをPostgreSQLに固定したまま別DB Query Assetを編集 | #2862のQuery IDと選択SQL/Providerの具体版を固定し、Provider actual Read/Schema/NULL/粒度が等しいかは別試験 |
| BP-06（未実施） | 選択済み必須Providerの版/Capabilityが不適合 | silent fallback/無根拠の未使用判定を拒否、Build/実Readの必要条件を診断 |
| BP-07（未実施） | 組織Write CommandをBinaryに収録、Entryは非公開/Current Grantなし | `compiled-in`≠`exposed`≠`authorized`。実Host/Provider拒否を観測 |
| BP-08（未実施） | Source/Plugin/RoleのA→B→A、候補依存の追加・削除 | 現在のbyte一致から中間受理撤回や実Build/Grant資格の一貫性を推測しない |

**判断基準**：BP-02/03の「未使用」とは**ビルド対象Function/Targetで実消費されない**意味。選択Bundle全体の整合性検査を省略することではない。Compiler Linkerの副作用、Go `init`・自動登録・空import、Generatorの横断Source走査、Asset埋込みなど、使用が見えにくい経路も実Build Inputの差分から点検する。機械的な巨大Plugin分割や、無関係な抽象Plugin追加は適合条件にしない。

### 51.5 次に実際に走らせる最小順序

**P0** は今回のNative E1 BP-01（合格済み）と、現実のBundle Selection/Lock/Source Rootを固定した**read-only解決比較**。**P1** は対象TargetだけのGo Binary実Build＋Artifactと必要Moduleの差分（BP-02/03）、**P2** は必要Capability選択と実DB Read/Policy/Host/Effect（BP-04〜07）、**P3** はBuild latency/size/startup/移行コスト実測と、AI-first独立ExpectedによるMissing/Over-Inclusion検出。P1未実証の段階で「未使用プラグインはビルド結果に影響しない実装が完成」と宣言しない。

**状態**：BP-01の実Go PASSのみ。実Target Build閉包・Binary生成差分・使わないがBundle選択済みのModule除去・二DB/Provider・Host Authority・Studio/SHO/AI/Owner正式受理、BP-02〜08は未実施。#1119既存P0実DB/Owner-writeを優先。新Plugin Manager/Registry、独立Bundle Engine、組織専用Host/ACL、第二Build/Query Solverを設けず、契約差分だけ既存Ownerに渡す。

<!-- 20261105-result-restart-bundle-boundary-refresh -->
## 52. 実行計画・成果物・再起動・Bundle境界の新しいmain実証を横断Gateへ接続（2026-11-05）

### 52.1 局所成功と正式な業務資格の差を、最新の実装で更新する

| mainへ統合された変更 | 実際に得られた限定的な証拠 | 継続する不足・既存Owner |
| --- | --- | --- |
| [#2866](https://github.com/hat47x/tei-platform/pull/2866) | function-definition/v1の機能定義書へRunPlan宣言を決定的に出力、省略を補完しない | 文書の宣言≠実行/Provider effect。#1130/#2604 |
| [#2871](https://github.com/hat47x/tei-platform/pull/2871) | RenderOutputsによりread-only ProjectOutputsの結果を帳票本文へ整形。0件・Reader error・Output宣言なしを別表示 | 件数表示≠対象母集団の完全性/実Reader適合/Consumer実受領。#1130/#2516 |
| [#2870](https://github.com/hat47x/tei-platform/pull/2870) | Go Lifecycleのunknown stageは再起動相当の新インスタンスに引継がれず再Dispatch可能。実PostgreSQLは同一idempotency keyだけ重複適用を止めてReceiptでReplayを返す | 別key+ABA、永続Owner/generation、実通信断と全体exactly onceは未保証。#2564/#1174/#2146 |
| [#2869](https://github.com/hat47x/tei-platform/pull/2869) | Critical Path監査がProvider atomic-with-commitの宣言欠落や仮宣言を診断 | 宣言/Reasonは自己申告であり実Atomicity/正式G2は未証明。実PG暫定ChainはINCOMPLETE/UNQUALIFIED。#1119/#1163 |
| [#2865](https://github.com/hat47x/tei-platform/pull/2865)、[#2868](https://github.com/hat47x/tei-platform/pull/2868) | 参照Storeの候補KeyはUTC日時、時刻、Unicode NFCを正規化 | DBのCollation/Timezone/Precision、Business Keyの意味受理やCurrent Ownerは別 |
| [#2872](https://github.com/hat47x/tei-platform/pull/2872)〜[#2874](https://github.com/hat47x/tei-platform/pull/2874) | internal/commandの新x/text依存でStudioの独立Go moduleのreadonly buildが一時失敗し、module/sum追補で修復 | rootのGo CIだけで全Build Targetへの依存Closureを保証しない。#1251/#2831 |
| [#2875](https://github.com/hat47x/tei-platform/pull/2875) | Application/Database releaseの現在Contract版参照を共有し、一意に導出できるreason等の冗長な候補Viewを縮小 | 欠落した説明Fieldを新しい成功根拠としない。Compatibility Evidence/Owner受理は別。#2619/#951 |

これらは**異なるPR/実行環境/Version/Readerの証拠**であり、同一Attemptの製品一往復完成には足し算できない。一方、実済みの成果までNOT RUNと報告しない。正例と負例・回帰の境界を列ごとに明記する。

### 52.2 次の統合試験を増やさず既存三本へ集約

1. **Source/Skillの最初の価値Gateを優先**：§45.3〜45.4の公開TEIユーザー定義1 Subjectの実Loaded Source/CLIと、Tool-only H2／Skill H3の四独立Sessionをまず成立させる。Role自己解釈やDB移植を新たな前提にはしない。
2. **Build/Bundleの非干渉は別P1**：§51の既存BP-01実Go（Manifest外不正Plugin SourceがWorld/Operationsへ非干渉）を初期値として、[#1251](https://github.com/hat47x/tei-platform/issues/1251)のPB-01〜10でSelected/Build-consumed/Artifact/Runtimeを分離する。未選択PluginがArtifactを変えないことと、選択済みStudio nested moduleの新依存を漏らさないことは両方を必要とする。Go root健全性を全Target Build証拠へ昇格させない。
3. **実行・業務Effectは既存Ownerの一つの書込みPathへ**：[#1130](https://github.com/hat47x/tei-platform/issues/1130)のRPL-01〜05、#2853/2854/2870の製品Commandと実PGの局所証拠、#2869のAudit不足を同一Function/OperationIDに結ぶ。最初はRead-only Output本文と実Consumer、次に同Key/新Key/ABA/再起動/Effect Unknown/Receiptを独立Expectedで比較。Observerの存在、RunPlan completed、文書行・Audit Guarantee自己申告だけでBusinessVerifiedとしない。

### 52.3 新たな停止線

一回の実行で**データの意味、実Read/Write、Plugin/Provider実装、Sourceの版、Current Grant、Recovery段階、実Consumer受領**が連結していない場合は、その欄をunknown/NOT RUNのまま残す。「未使用Pluginは実ビルドに影響しない」「異なるDBでEquivalent」「Lifecycleは再起動しても一度限り」「帳票の0件が対象不存在」「監査のatomic宣言で本番資格」が、現実の局所CIから自動成立したと主張しない。

**Owner**：Plugin bundle/依存=#1251/#1097/#2831、User Skill=#2605/TAI #168、RunPlan/帳票=#1130、DB/SQL=#2622/#2862、Provider Effect=#1174/#2564、Audited Gate=#1119/#1163、Consumer=#2516、Independent Oracle=#2514。新規Core Enum/Host/Registry/Scenario Engine/Universal Policy Engineは導入しない。**この節自体はDocs-only。BP-02〜08/PB-01〜10のGo Binary実Build差分、4実Harness、実業務RunPlan一往復、Role正式受理、二DB同一Function、Owner Grantの統合試験はNOT RUN**。

<!-- 20261106-verification-obligation-crosslane-join -->
### 52.4 検証事項→実証証拠→主張資格を共通の受入票へ接続（2026-11-06）

**今回の変更点**：§52.2の三本の実証経路を作り替えるのではなく、**実行したテストの数と、必要な検証が満たされた量は別**という原則を検収へ固定。[Scenario SSOT #1180](https://github.com/hat47x/tei-platform/issues/1180)にSource/Scope、必須Guard集合、実Provider Effect、Replay、Consumer受領、Role/Buildの6項目のObligation対照票を追加。[独立Oracle #2514](https://github.com/hat47x/tei-platform/issues/2514)には同じ誤った実装からCandidateとExpectedを生成する自己追認・Mutant判別の条件を追加。[一改訂証拠 #1163](https://github.com/hat47x/tei-platform/issues/1163)は**同一Source Revision・Artifact・Target・OperationID・Attempt・Consumerの実際の連結**を所有する。第二Schema/Proof Engine/Scenario Runnerを設けない。

| 現行main・別Draftの証拠 | 認められる限定Claim | 残すべきObligation |
| --- | --- | --- |
| [merged #2882](https://github.com/hat47x/tei-platform/pull/2882)実Workspaceの期間closed/open・Observer無し（3 CI success） | Parser/Operation Surface/RunPlanを合成Stepへ接続し、blocked/completedを検査済み | 必須Guardの網羅、Current Grant/Periodの実Commit Fence、実DB効果0 |
| [merged #2853/#2870](https://github.com/hat47x/tei-platform/pull/2870)の別試行PG CAS・Receipt・再起動時同Key replayと[#2854](https://github.com/hat47x/tei-platform/pull/2854) ABA負例 | 実PG局所Read/Writeと同Key Replayの条件・新Key再適用の危険 | #2882の対象Function/Guardと実PG/Consumerを同一Attemptで照合する条件 |
| [merged #2866/#2871](https://github.com/hat47x/tei-platform/pull/2871)の機能定義書・帳票本文 | Artifact表現とread-only投影の局所正例 | 実データ完全性、Manifest同一Generation、別Consumer実BodyとDisclosure |
| [Draft #2877](https://github.com/hat47x/tei-platform/pull/2877) G4 Fit/Gap、[Draft #2827](https://github.com/hat47x/tei-platform/pull/2827) ETL/MFT | Self-Build/二Consumerは設計候補。#2827のMarkdown/index旧Health失敗は修正コミットを作成 | 実Builder closure・Stage0/1/2と、実転送/Delivery/Importそれぞれの独立実測 |

**導入最小単位**：既存Scenarioの必要なObligationごとに、Required／Expected独立出典／Observed＋Head＋Attempt／negative control／pass・fail・unknown・not-run・not-applicable／Claim ceilingを**既存Reportへの派生参照**で埋める。CallerがPeriod Guardを省略したのに残りのPreconditionが成功した場合は、「実行成功」でも**要求GuardのCoverageが未充足**。同様にFile Bodyが受信されてもBusiness Acceptanceは未証明、Graphに辺が無くても実Builderの未宣言step不存在は未証明と扱う。

**Gate変更なし**：優先するのは公開Product Skillの実4Sessionと、#1119 G1/G2の最短一業務Loop、#2831 G4実Self-Buildを独立P0で並走する従来順。DB2系統・Studio UI・ETL本番配送をG1/G2のHard dependencyへ昇格しない。**この追補はDocs-onlyで、新しいScenario/Provider実Go E2E・ETL二Consumer/Target Build・正式Owner受理はNOT RUN**。

## 53. 組織Caseの無人Action：必須前提の網羅、途中効果、実Receiptを分離（2026-11-06、#2882/#2889を#1130/#1174へ最小接続）

### 53.1 部品の実証と縦断の不足

- [統合済み #2882](https://github.com/hat47x/tei-platform/pull/2882)は実`.tei`のPeriod/EffectRule→Operation Surface→RunPlanでclosed/open/unobservable effectの3条件を実Goで検証した。前提の**内容に対する判定**が可能になったが、**必要な全条件の網羅性**は自動保証しない。
- [独立Draft #2889](https://github.com/hat47x/tei-platform/pull/2889)は`internal/execution`で組織Caseを想定する3つの**toy Step/Journal負例**を追加した。固定HEAD `e671aa41a8b7fd40de8602595e98ebf624043371`の[Actions #42100472921](https://github.com/hat47x/tei-platform/actions/runs/42100472921)は**completed/success**（Go 1.26.10/Linux、OC-RP 3/3 PASS、exact HEAD/clean）。**実DB・CaseRecord`.tei`接続・現行Host Grant・実Receiptは未実証**。
- [統合済み #2869](https://github.com/hat47x/tei-platform/pull/2869)のProvider atomicity`Guarantee`監査は、宣言未提出/仮宣言を`INCOMPLETE`として検出するが、`atomic-with-commit`はProviderの**自己申告**。本物のTransaction/Effectを証明しない。
- [統合済み #2870](https://github.com/hat47x/tei-platform/pull/2870)はin-memory Command Lifecycleのunknown段階が再起動で失われること、別PostgreSQL Testで**同じidempotency keyの再適用だけ**がReceiptにより抑制されることを確認した。新Key/ABA/旧Grant/Owner再試行規則を自動承認する証拠ではない。

組織側の[実E1 #2681](https://github.com/hat47x/tei-platform/pull/2681)は`CaseRecord→OrgUnit`の構造上のRelationを扱うが、**そのCaseを読み取った／実行Hostが移管を許した／RunPlanが実DBへ更新した**わけではない。これらは違うHEAD・Source・Function・実環境の成果であり、合算した「組織E4」を記録しない。

### 53.2 必須条件の宣言・実施・結果を三列で照合する

**前提条件の「全部」を定義する権限はBusiness/Operation Ownerと現在のHost契約が持つ。**`CombinePreconditions`は渡された関数を合成するだけであり、Ownerが必要としたチェックが欠けていても検出しない。新Core Precondition Enumや全域Registryを追加せず、同一Operation/Requestに対する**Owner-accepted required set／supplied check set／observed verification**の三列を既存受入票で比較する。

| 実証Slice | Owner-accepted required set（採用前の例） | 実行・観測の停止条件 |
| --- | --- | --- |
| **E2 Case Read** | Case/Relation Occurrence、Source/Role/Meaning、明示Data Context、Row/Field/Cohort、組織Scope/現在Read Grant | 未受理のOrg Relation/partial Result/Provider施行点未確認ならRead不合格（#2586/#2622/#2628） |
| **E4 Case Transfer Admission** | C1 Target/expected version、Org Policy・valid/known、現在Principal/Host Update Grant、Period Rule、EffectRule、必要Observer、Provider atomicity資格 | 一つのcheck成功でも分母不明・Current Grant/Version欠落なら`unverified`/実行拒否（#2386/#2598/#1130） |
| **E4 Runtime/Recovery** | Operation ID、RunKey/Attempt/owner-gen、Idempotency Key、Commit/Unknown、Durable Provider Receipt、独立Post-image、部分Effect/補償条件 | RunPlan completed≠Commit、Observer present≠observed、blocked≠Effect 0、再起動後in-memory state≠durable fence（#2564/#1174） |
| **Consumer and AI** | 対象Scope・現在Read/Disclosure資格、Source引用・Policy/Grant版、AI Candidate受理・Host Action選択を分離 | AI提案→Provider実行の直結、非許可Recordの件数/Export漏れ、旧Response再採用は拒否（#988/#2619/#2516/#2582） |

### 53.3 次に進める一固定Attempt（新プロジェクトではない）

同じ`CaseRecord` C1と`handlingUnit`に対し、(1) 実Workspace/Function ContextがSource/Meaning/Roleを固定、(2) 実ProviderがCase/Org RelationをReadしてRow/Field/Scope/Read Cohortを独立期待値で確認、(3) Owner受理済み移管IntentとHost Current Grant/Expected Versionを確認、(4) 既存RunPlanに**必須Precondition Coverage**が提供されたことを固定、(5) 既存Command/ProviderでCommit、unknown/partial/ABA/restartの反例を記録し、(6) Effect Receiptと別経路Readで結果を照合する。**最初は(1)〜(2)のE2 Readだけに限定し、実入口が無いときは`NOT RUN: provider-read-unavailable`。**ここへPluginの追加や第二Query/RunPlan/Hostを持ち込まない。

**境界**：#2889のGo3/3はRunnerの負例のみ。`RunPlan`の前提網羅を自動化したことやDB/Hostの安全性を保証しない。既存#1119 P0-a/P0-bと#2586 E2 Readを優先し、実PostgreSQL更新済み別業務Functionの成功をOrg Caseの実移管としては記入しない。今回の§53は**実コード・固定CIに基づく計画への引継ぎ**で、実Case DB・Host・Studio/SHO/AI・業務Owner正式受理はNOT RUN。


<!-- tei-20261106-evidence-discharge-no-second-engine -->
## 54. 未充足の検証事項を最短の実利用へ渡す：Harness実Tool・RunPlan Guard・Studio現在性（2026-11-06）

この節は§52.4／§53を置換せず、**次の担当が実行できる小さな出口**だけを固定する。既存[#1180](https://github.com/hat47x/tei-platform/issues/1180)がObligationの必要集合、[#2514](https://github.com/hat47x/tei-platform/issues/2514)が独立Expected、[#1163](https://github.com/hat47x/tei-platform/issues/1163)が同一Source/Attemptの証拠連結を所有する。TEI／TAIの第二Evidence DB、第二Host、全用途Guard Registryは作らない。

### 54.1 実在した成果と「まだ利用したといえない」境界

| 縦切り | 今回固定できる限定Evidence | 今なお未充足のObligation |
| --- | --- | --- |
| **A：実Product CLI→外部Harness** | [TAI #168](https://github.com/hat47x/tai-pleroma/issues/168)／[Draft #270](https://github.com/hat47x/tai-pleroma/pull/270) の実CLI/公開Workspace `semantic-data:sales.order` で、Source Indexと別途Source byte照合の限定S1が成立。Runbook §18〜21は台帳のargv/cwd改変拒否、参加者Task固定・独立Holdout、R=割付とD=実提供、ITTとPPを別記する設計へ進展した | **外部HarnessがToolを実発行し、そのTaskのargv/cwd/exit/stdoutを受領したH2一Session**、H3のSkill root/reference実読込み、四独立Sessionの実回答・費用。台帳の整合性だけではHarness経由使用を証明しない |
| **B：RunPlan→Effect** | [main #2882](https://github.com/hat47x/tei-platform/pull/2882)は同一合成WorkspaceからPeriod/Effect前提を実RunPlanに接続。[Draft #2889](https://github.com/hat47x/tei-platform/pull/2889)の実Go/toy Step負例OC-RP-01〜03は[専用CI #42100472921](https://github.com/hat47x/tei-platform/actions/runs/42100472921) 3件成功 | Ownerが要求する**Required Guardsの全提供・実評価・Commit時の強制**。Observer存在≠対象Effectの観測、RunPlan completed≠全Runの効果確定。実PG current Grant/Period/Target/Receiptの同一AttemptはNOT RUN |
| **BのRepo統合資格** | #2889固定HEAD `e671aa41a8b7fd40de8602595e98ebf624043371` の専用CIは成功したが、[Repository Health #42100472849](https://github.com/hat47x/tei-platform/actions/runs/42100472849) は `internal/execution/org_case_runplan_boundary_test.go` の**gofmt drift**でFAIL。原因と修正後の確認対象は[Owner PRコメント](https://github.com/hat47x/tei-platform/pull/2889#issuecomment-6669192926)へ報告済み | gofmt後の**新しいexact HEAD**で専用CIと全体Healthを別々に取得する。旧HEADの局所成功を新HEAD/製品G1・G2成功として転記しない |
| **C：Studio現在性** | [#1824](https://github.com/hat47x/tei-platform/issues/1824)で `runCommand` の開始時Workspace取得と後着 `displayCommand` の現在Workspace/入力/世代再照合が未接続と静的確認。Reference `mode=reference, committed=true` は外部DB Commitではない | Workspace Aを実行中にBへ切替えた後のA成功/失敗がBのDOM/ARIA・Command状態へ混入しない**実Consumer試験**。Source/Grant失効通知が無い場合に即時現在性まで仮定しない |
| **D：実Build/構成** | [#1016](https://github.com/hat47x/tei-platform/issues/1016)でComposition Profile/Bundle/選択/実依存と現在Authorityを分離。[#2831](https://github.com/hat47x/tei-platform/issues/2831)では実Graph/Build/非Plugin binary/stageのP0並行試験を定義 | 実Build consumed closureとTarget binaryの独立照合、G4。Aのread-only H2やBの初回PG E2EのHard dependencyにはしない |

### 54.2 最初の二つの「実行」を独立に完了させる

**A0-H2初回Session**：TAI Runbook §20の固定Participant TaskとReviewer-only Expectedを分離し、同じ公開TEI binary/Workspaceを使う。一回目はTool-onlyで、Harness session ID→実Tool request→実CLI argv/cwd・実Executable byte→exit/stdout byte digest→実Tool response→回答の対応を**外部Harness自身が生成した観測**で固定。TAI CIが直接実CLIを呼んだS1、試験用JSONの`tool_invocation_observed=true`、LLM自己申告は代用不可。最初のH2実Toolが未成立ならH3へ進まず`insufficient-evidence/unsupported`を保持。H2成立後、H3では実Skill rootと必要Referenceのロード観測を追加し、別Sessionで`H2→H3→H3→H2`を比較。**R=割付／D=実提供、ITT=失敗込み全割付／PP=実提供成立の条件付き比較、回答正答と費用は独立採点**。4 Sessionを統計的な一般優越へ外挿しない。Owner=TAI #168、TEI #2605/#2018。

**B0 Required Guardの未充足検出→B1実Effect**：既存order-submit／Case read-only Fixtureで、対象Purpose/Scope/Source Pinごとの`Owner-required / Supplied / Evaluated / Enforced-at-Effect`を派生比較する。PeriodだけSuppliedのときにGrantが必須なら、局所Step completedのままでも**業務実行資格は未充足**とする。Observer存在しか分からないEffectは`unobserved`、先行Stepがresolved後に次がblockedなら`partial/unknown`。B1ではmain統合済み#2853の製品Invoker→実PG CAS/Receipt、#2870同Key再起動Replay、#2854別Key ABA反例を**同一Source/OperationID/実Target/Current Grantと独立Postimage**へ結ぶ。要求される強制点が実Provider/Hostへ届かないなら書込みの資格化を停止。Owner=#1130/#1174/#1163。旧X-TEA意味対応は#1969/#940で未受理のまま。

### 54.3 横断失効と停止規則（新しい共通Storeを作らない）

- **Source／Model**：manifest等への宣言だけで実Parser/Role消費を証明しない。required/declaration/discovery/parsed/actual consumedをClaimに応じて比較。#2822/#2823のunparsed Source、Load後Source変更、A→B→Aは現在資格の再評価。
- **Tool／Plugin／Build**：Profile選択・Skill配置・Bundle同梱と、実Tool呼出し・実Skill読込み・実Binary link/loaded bytesは別。実消費SourceやToolchain不明では再利用の合格を作らない（#1251/#2600）。
- **非同期UI／Consumer**：返答のSource Revisionが同じでも、Selection/Inputs/Operation/Grant/Disclosureの現在性がずれれば古い応答をB画面へ採用しない。DOM/ARIA/外部送信を実観測するまでは未実証（#1824/#2516）。
- **副作用／Recovery**：送出前拒否の実Effect=0と、送出後Commit不明を区別。同Key lookupを優先し、別Key盲再送・Observer nonnil・Auditのprovisional GuaranteeをBusinessVerifiedへ昇格させない（#1174/#2884）。

**着手順**：A0-H2一Session（read-only）とB0 Guard分母の実Go再検証を並行→B1製品PG同一Attempt／CのStudio A/B後着負例→必要なTarget/Provider/DB/SHO/EKIを後続へ。Dの全TEI Self-Buildは独立P0を維持。#1180のScenario、#2514の独立Expected、#1163のEvidenceを参照し、テスト件数・別HEAD局所PASSを一つのBusiness Qualificationへ加算しない。

**今回の実証上限**：本節は既存実Run/Issue/Draftの状態を再照合して優先順と停止条件を**文書に追記しただけ**。新規H2/H3実Session、Org/受注の同一PG+RunPlan実E2E、Studio実DOM/ARIA、G4実Build、新Owner意味受理はNOT RUN。既存G0〜G4、既存OwnerとDraft状態は変えない。


<!-- tei-20261106-ambiguous-input-no-silent-qualification -->
## 55. 曖昧な入力の「後勝ち」とUnknownの「成功扱い」を各境界で止める（2026-11-06）

前節§54は必要な観測が**提供されない**問題を扱った。本節は、観測・定義は存在していても**重複やUnknownが黙って別の状態へ置換され、見かけの成功になる**問題を扱う。対応は既存Ownerが各入口で行い、新しい全域Unique/Policy/Proof Engineや単一Runtime Enumを作らない。重複が許されるデータ（多重辺、配列、履歴、同一Roleの別Occurrence）まで機械的に禁止しない。**同一Keyで一意であると主張する対象・Scope・時点・適用条件があるか**を最初に固定する。

### 55.1 実コードで特定された上書き／偽資格の危険

| 対象と事実の強さ | 曖昧さが失われる箇所 | 必要な検証・Owner |
| --- | --- | --- |
| **Migration History**：現mainの internal/database/migration/history.go を静的確認 | CompareHistoryではHistoryにAppliedUnknownが存在しても、失敗・checksum不一致以外の分岐がRepoAppliedへ到達し得る。History取得不可を空Entriesへ変換すればRepoPendingにも誤誘導 | [#2892](https://github.com/hat47x/tei-platform/issues/2892) HIST-01〜05を既存Goテスト＋観測完全性へ接続。新RepoUnknown enumかfail-closed errorかはConsumer影響を調べOwner判断。修正前はG1〜G5に適用成功として供給禁止 |
| **DB Release/StateSnapshot**：[Draft #2887 §15〜16](https://github.com/hat47x/tei-platform/pull/2887)の実コード確認 | 同一Database IDのSnapshotをmapへ取り込むと後の要素が勝ち、旧/新Releaseの配列順がCompatible/Blockedを変え得る。ObservedAt欠落/古さも構造互換では検証不能 | 既存MG-21/29のGo特性化と、実Instance/Provider/Read Cohortの明示選択。二重Snapshot・取得時点unknownをGate PASSにしない（#2555/#2611/#2622） |
| **Semantic Attribute Role**：[Issue #2620](https://github.com/hat47x/tei-platform/issues/2620)／Draft #2860の実Parser Characterization | 一つのAttributeに role accounting_amount role audit_amount があると、現Parserの単一Role値では前のRole occurrenceが見えなくなる。description内の文字列はRoleではない | Raw Source・Occurrence診断・Meaning Candidate・Owner受理を分離。衝突方針前にSource一意性/Role受理/編集逆写像を保証しない（#988/#2621/#2497） |
| **AI/Scenario Fixture JSON**：[Draft #2891](https://github.com/hat47x/tei-platform/pull/2891) | Scenario本体は重複Memberを検査するが、一部Fixtureのencoding/json.Decoderは後値で重複Keyが消え得た。CandidateとExpectedを同じ後勝ちデータで評価すると偽一致の恐れ | 既存RejectDuplicateJSONMembersを既存3 Parserへ再利用する変更をレビュー。固定HEAD 2f677921 の[Health #42107910706](https://github.com/hat47x/tei-platform/actions/runs/42107910706)は**gofmt driftでFAIL**。新HEAD CIまで機能を正式採用済みと記さない。配列の同値重複はJSON Keyの重複と区別 |
| **Studioの旧非同期結果**：[Issue #1824](https://github.com/hat47x/tei-platform/issues/1824)の静的Consumer照合 | Workspace AのRequest後、選択・入力/PeriodをBへ変更したのに、旧A ResponseがBのDOM/Command状態へ後着する可能性 | ResponseのRequest/Workspace/Selection epochと表示対象を実DOM/ARIAで突合。旧成功・旧失敗ともBへ採用不可。ただし実GUI再現はNOT RUN |

**横断上の別の欠落**：Role/Fixture/DB Snapshotの重複拒否が局所合格でも、Query→Function→実Provider ReadのResult Slice、Row/Field/Scope/Read Cohortや、RunPlanのRequired Guards、Receipt/Independent Readの必要集合は保証しない。[#2586](https://github.com/hat47x/tei-platform/issues/2586) HJ-01〜08と[#1180](https://github.com/hat47x/tei-platform/issues/1180)のObligationへ引き渡す。結果0件、Page間でSQL/Provider/Read Scopeが変更、JOIN/ORDERの非出力依存欠落は、現在の不存在・全件取得の証拠にならない。

### 55.2 最小の独立対照と修正の順序

| 対照 | 固定するInput/Expected | 未修正時に禁止する資格 |
| --- | --- | --- |
| **QF-01 Migration Unknown** | 同じMigration ID/定義Fingerprint。History AppliedSucceededとAppliedUnknown、history unavailable/partialと正当な空Historyを別Fixtureにする | UnknownをRepoApplied/RepoPendingの確定根拠にしない。StateSnapshot Compatibleを履歴の代用にしない |
| **QF-02 DB Snapshot order** | 旧Major4/新Major5、同一DB IDの二Snapshotを順序反転して比較。外部History/ObservedAt不在も追加 | 配列順で現行実DB・Read/Write Authority・Cutoverを選べたと主張しない |
| **QF-03 Duplicate Role / JSON Key** | 同一Source上の別Role occurrenceと説明文内literal、JSONの同名Key二値と正当な配列重複。Parser/Reviewerの独立Expectedを固定 | Last-wins Roleを単一Accepted Meaning、Fixtureの後勝ちを独立Oracleと呼ばない |
| **QF-04 Read Cohort change** | Query ID固定でSQL Source/Provider/Scope/ORDER/Page境界のみ変更。取得対象/Read Cohort/Result completenessの独立Expected | Page1+Page2を無条件に同一世代の全件、0件を不存在と認定しない |
| **QF-05 Async selection/Grant** | A request→B selection→A success/error後着。Reference committed=trueと実PG Receipt無しを区別 | Bの表示・操作可・業務Commit完了を旧応答から組み立てない |

**Priorityの補正**：P0として (a) #2892の偽Appliedに対するHIST-01/03負例・最小修正のOwner判断、(b) #2891のgofmt修正後exact-head focused CI、(c) #2586の一件のread-only Query→Context→Claim/Result Slice実受渡しを既存担当で進める。ただし**P0-Aの外部Harness初回H2 Tool呼出しのHard dependencyにはしない**。旧/新DB Cutover/Contractを実施する場合に限り、History/StateSnapshotの確認を**そのGate固有の必須条件**へ昇格し、Ownerの正式許容を求める。書込み実行は#1174/#1163の同一Attempt・効果観測なしに一般許可しない。

**証拠の区分**：#2620のRoleテスト、#2887 MG-21/29、#2891 Duplicate JSON修正、#2895 AI独立有限OracleはいずれもDraft・用途別の局所証拠。新たなQF-01〜05の統合Test、実DB Migration/二DB Query、Studio実GUI、実Harness四Session、Owner Meaning Acceptanceは本§55でNOT RUN。各Ownerの既存Caseを再利用し、QFは**引継ぎ上の対照名であって第二Scenario/Oracle正本ではない**。


<!-- tei-20261106-section55-resolution-compat-reconciliation -->
### 55.3 安全な修正候補と旧特性試験の期待値を、同じ統合単位で整合させる

本節§55のリスク記録後に、**別Draftへ実コードの修正候補**が進んだ。現main修正済みと誤認せず、Ownerごとに次の「旧特性を観測した資料→修正後の独立Expected→exact-head CI」を一組として扱う。

| 問題と新たな候補 | 旧特性・新修正の境界 | 最短の統合受入 |
| --- | --- | --- |
| **MG-29 State/Contract後勝ち**：[Draft #2900](https://github.com/hat47x/tei-platform/pull/2900) | #2900はduplicate Database Stateと同一State内のduplicate ProvidedContractを、既存CompatibilityのUnknown＋理由に変更するGo差分を提示。独立DBの正常判定は維持。**[Draft #2887](https://github.com/hat47x/tei-platform/pull/2887)の TestMG29DuplicateDatabaseObservationsAreOrderDependent／TestMG29DuplicateProvidedContractVersionsAreOrderDependent は旧last-winsに対してCompatible/Blockedを期待**しており、安全修正後は不一致になる | #2887の旧挙動再現は歴史的Observationとして保持。実統合では同テストを「順序反転でもUnknown/拒否」に書き換え、#2900の独立4テストと共に**一つの実Checkoutのfocused＋全体CI**で確認。No silent fallback。#2897がOwner |
| **重複Semantic Role**：[Draft #2898](https://github.com/hat47x/tei-platform/pull/2898) | parseSemanticDataで二つ目のrole optionを位置付きParseErrorにするコード候補。不同値・同値・コメント越し・quoted descriptionとの負例を追加。[Draft #2860](https://github.com/hat47x/tei-platform/pull/2860)の TestSelfDescriptionRoleLiteralAndDuplicateAreNotAcceptedMeaning は「将来ParserがSyntaxで重複を拒否してもよい」という分岐を既に含む | #2898のfocused Syntax/Anchored解析、#2860の現HEADと組合せた回帰、実Source byte/Role Candidate/Accepted Meaningの区分を同じ改訂で再確認。Parserが拒否できても**Role optionの編集用Span・書戻し・Owner意味受理は未実装**。#988/#2620がOwner |
| **Fixture JSON duplicate member**：[Draft #2891](https://github.com/hat47x/tei-platform/pull/2891) | 旧HEAD 2f677921のGoテスト自体は限定PASSでもgofmtで全体Health FAIL。新HEAD 16b8d554は書式修正済みの候補 | 新HEADのRepository Health完了と当該10 caseの実行/非SKIPを確認する。JSON objectの同名Key重複を拒否しても、Relation keys配列の重複という別意味上の負例や独立Oracleは残す（#2890/#2895） |
| **Migration History Unknownの偽Applied**：[Issue #2892](https://github.com/hat47x/tei-platform/issues/2892) | 現main CompareHistoryの AppliedUnknown がchecksum一致時にRepoAppliedへ落ちるコード経路は確認済み。ここは#2900のDB Release互換修正とは別 | HIST-01/03の独立Expectedを固定→既存Consumer適合を比較→RepoUnknownかfail-closed errorかOwner判断→focused Go/実History Adapterへ。**#2900のUnknown修正がこの不具合まで直したとは記さない** |

各PRのSource、想定Expected、実行された試験、主張資格を**そのPRのexact HEAD**で再照合する。Draft別HEADのPASSを横断の製品G1/G2/G4合格へ集計しない。旧Characteristic testは修正後の安全な受入条件の正本として存置しないが、原因分析用の履歴は残す。自動mergeや他担当Draftの共有ファイル改変をこの計画で指示しない。

### 55.4 旧新Consumer・配備世代の共存を、方向付きの互換性として検収する

[並行Draft #2896 §19.9〜19.11](https://github.com/hat47x/tei-platform/pull/2896)の独自成果は、**同じFunctionの複数Entryが同じ配備Generationであるとは限らない**こと、互換性は対象Consumer/方向/時点に相対的であること。この詳細表は#2896と既存Ownerの責務に残し、本§55では**既存の一Attempt受入へ渡す条件だけ**を固定する。#2896と本Draft #2667は同一workstreamファイルを編集しているため、丸ごと二重採用しない（[衝突調整](https://github.com/hat47x/tei-platform/pull/2896#issuecomment-6669657786)）。

**最小の対照例**：order_entryのFunction/意味R0→R1改訂後、WebはR0 Binaryを実ロードしたまま、CLIだけR1、DBはExpand済みでも旧Writerが継続、SHO等の成果物ConsumerはR0 Manifestを保持している状態を考える。これは採用済みProduction Deployment方式ではなく、#1180既存Scenarioと#2514独立Expectedに用いる変更圧力。

- **Definition**：R0の歴史的Meaning受理は記録として残せるが、新CLIのR1結果を旧WebのR1動作の証拠にはしない。NULL/粒度/Role/Rule/業務Key・有効時点はConsumerのRequiredに応じて再確認。
- **Realization/Loaded**：同Function名でも、実Source/Provider Pin・実Build input・物理Binary digest・入口が違えば同一実装と扱わない。単体Go Binaryの通常更新に新常駐Generation Routerを要請しない（#1130/#1167/#2600/#1223）。
- **Operational/Effect**：互換性の方向は旧Reader→新Schema、新Reader→旧Schema、旧Writer→新Schema、新Writer→旧Schemaで独立。旧Writerのin-flight OperationでCommit ACKが消えたまま入口を置換しても新Key再送を許可しない。元OperationID/Receipt/実PostimageとCurrent Grantは#1174/#2555が検証。
- **Publication/Consumer**：R1 ManifestとR0 memberの混在、R0固定引用とR1現在説明は別。実Consumerが必要memberの同一世代byteを取得し、現在開示資格を満たす範囲だけ公表可能（#2604/#2516）。
- **Plugin Lifecycle**：[DSH比較Issue #2899](https://github.com/hat47x/tei-platform/issues/2899)のregistration/disposeの成功はRuntime資源の寿命であり、旧世代が残したDB Commitのrollbackではない。Readiness/drain/無停止は選択済み常駐Process/Wasm Targetのみ#1223で必要時に実測。Third-party Index/Publisherの主張はCurrent Host権限にならない（#2390/#2589）。

受入単位は新Release Objectではなく既存の **Claim × Function/Entry × Consumer × Source/Artifact Revision × Environment/Time × Attempt** に対する証拠参照。実装済みCompat check一つ、Snapshot 0件、Version同名、Source byteの再一致、Runner completedでは、旧新Clientの現在の全体適合・Cutover・業務Effect・公開一式性を保証しない。新Schema/共通Deployment Coordinator/Host/Resolver/Policy/新Test SSOTは不要。

**次の実測順**：#2586のread-only一Functionを最初に通す。旧新互換は#2555/#2887の必要Client集合、#2604の二Generation Reader、#1174のEffect/Unknownを**別Attemptで対照**し、相互関連を#1163へ渡す。#2831全体セルフビルド・外部Harness H2初回は既存P0のまま独立。R0/R1異世代実Client・実DB Cutover・実Reader公開・Owner受理は**本Docs変更でNOT RUN**。

## 56. `*.tei.md`へのSource方針転換と、既存実Loaderからの安全な移行（2026-11-06、Owner方針）

**更新された決定**：2026-11-06、TEI向けSource文書は **`*.tei.md` を標準**とし、従来の `*.tei` は新規作成を原則停止・段階廃止する。旧[TEI #2621](https://github.com/hat47x/tei-platform/issues/2621)に残る「`.tei.md`を標準化しない」「必要時は別`.tei`へ切り出す」は**以前の提案**であり、今回の決定に優先しない。履歴の記述自体は削除せず、後続の変更として追跡する。

**実装の現状**：2026-11-06の現main `internal/workspace/loader.go` はSemanticを `.tei` suffix、Pluginを`plugin.tei`、Bundleを`bundle.tei`、Application等を明示`application.tei`・`*.tei`で選別する。**`*.tei.md`へファイルを改名しただけでは実Read/Parseされない**。`tei.workspace.yaml`は現行Manifest、legacy `tei.workspace.tei`は移行を必要とする例外的Reader経路。`workspace-files`のManaged、LoaderのConsumed、Syntax Parsed、Role/Meaning資格、Builder実入力、Consumer実Readerの差は既存#2823／#2595／#2514へ割り付ける。

移行計画の**唯一の担当正本候補は[Draft #3030 M0〜M6/TM-01〜14](https://github.com/hat47x/tei-platform/blob/docs/tei-md-unified-source-migration-20261106/plan/design/TEI_MD_UNIFIED_AUTHORING_MIGRATION_2026-11.md)**。本ブランチの[以前の横断メモ](../design/TEI_MARKDOWN_SOURCE_MIGRATION_2026-11.md)にあるM0〜M5/TMD-01〜12は#3030への**受渡し対応表・歴史的補助**に降格し、第二の移行仕様・Fixture/Oracle正本にしない。

**OKF／SHOとの明示分離**：`*.tei.md`はMarkdownを基礎とするTEI向けファイルだが、[OKF Draft #3029](https://github.com/hat47x/tei-platform/pull/3029)の`type`付きConcept適合や任意タグ、SHOのWriter採用を一律要求しない。通常`.md`はTEI実行対象ではなく、`tags: [tei]`や拡張子からOwner意味受理・Host Grantを推論しない。**新正式Grammar、実Role Parser、Filename Migration CLI、全Workspaceの拡張子改名はNOT RUN**。#2621/#2823/#988/#1920/#2600/#2514のOwnerが個別に実試験・採用を決め、#1119既存Gateを新規Grammarが未実装の段階で全停止にしない。

## 57. 2026-11-06の限定実証の更新：実AI Mutant検出・Health Receipt・実行先の責務

- **AIと独立検証基準（限定Product Mutant検出）**：[統合Draft #2901](https://github.com/hat47x/tei-platform/pull/2901)のexact HEAD `a6eb80cf` は[Health #42111973682](https://github.com/hat47x/tei-platform/actions/runs/42111973682) SUCCESS。[破壊的Draft #3017](https://github.com/hat47x/tei-platform/pull/3017)はProduct EvaluatorのRelation-Unique重複判定一行だけを誤反転し、[Health #42181290981](https://github.com/hat47x/tei-platform/actions/runs/42181290981) FAIL。実CIログで**AI追加Scenarioの対象Step自体が固定Expectedの不一致を検出**した（限定1 Mutant kill）。Draft #3017は**意図的な欠陥入りで絶対にmainへMergeしない**。NULL/欠損/部分集合・すべての独立検証基準／Owner受理には外挿しない（#2514）。
- **暫定一改訂E2EのHealth証拠**：[main統合済み #3025](https://github.com/hat47x/tei-platform/pull/3025)が、#3020で配線したfullHealth receiptを実PostgreSQL 16.14の一改訂E2Eで読み、`TRUSTED_FULL_HEALTH_NOT_RECORDED`の**一診断だけを解消**した。監査は依然 **status=INCOMPLETE / productGate=UNQUALIFIED**、Provisional Acceptance/Grant/Guaranteeの3診断が残る。Healthあり≠正式G1/G2／実外部Provider保証。#1163/#2619の既存台帳を使用する。
- **実分散とTAIの境界**：[EKI main統合 #773](https://github.com/hat47x/eki-fabric/pull/773)はLease台帳への実登録（失敗時はWorkerへ配らない）を追加したが、期限切れAttemptのResult reconciliationや外部Effect再実行許可は残る。TAI [Draft #489](https://github.com/hat47x/tai-pleroma/pull/489)は固定TEI CLI/Binaryの合成資格と**実Harness H2→H3→H3→H2未実施**を分離。TEI Source改名により既存Binaryが必要Sourceを消費しなくなれば、Skill能力に関係なく比較自体を無資格とする。EKIのLeaseやTAIのSkillがTEI Meaning/Host Grantを代行しない。

**本§56/57は現行GitHub Source・PR/CI報告を照合したdocs-only追加**。新Markdown Reader、実TMD fixture、実Product Scenario Mutation追加、外部Harness実セッション、実PostgreSQL再試験、Owner受理は**本変更ではNOT RUN**。Ownerが既存Issue/PRで取得済みの限定証拠だけを認める。

## 58. 新Readerの誤成功防止と、Command認可・実Effect・実DB判断の証拠を別資格へ（2026-11-06後半）

### 58.1 Source形式移行はGuard PASSと実Reader PASSを分離する

[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)（exact HEAD `63953ccb5c12ead26ec1268e7c0afdba4eff66ae`）では、managed Source-rootを走査した後、既存Native Loader前に `*.tei.md` が存在すると `TEI_MD_PARSER_UNBOUND`でfail-closeする暫定コードを追加。[専用Guard CI #42217379093](https://github.com/hat47x/tei-platform/actions/runs/42217379093)と[Repository Health #42217378867](https://github.com/hat47x/tei-platform/actions/runs/42217378867)はexact-headで **completed/success**。本文のMarkdown Parser/Role/Meaningを実装したわけではない。main統合は未確認、Draftの限定的な成功に留める。

**Legacyの実バイナリ反証は別の証拠**：[TAI Draft #270の実TEI CLI/Binary検証 #42239797449](https://github.com/hat47x/tai-pleroma/actions/runs/42239797449)は、同じ公開`sales.order` Source bytesを`order.tei`から`order.tei.md`へ単純改名すると、旧Loaderの実Parsed Subjectが **1→0件**になったのに `validate.ok=true` が返ることを記録（`EXPECTED_NEGATIVE_BEHAVIOR_REPRODUCED`）。これは従来の空成功を**実Binaryで再現**したもの。#3031のGuardのsuccessは**別HEADでの安全な拒否**であり、両者を一つのE2Eへ加算しない。

**移行の順序を固定**：[#3030への既存Owner引継ぎ](https://github.com/hat47x/tei-platform/pull/3030#issuecomment-6688075247)に従い、M1の`*.tei.md`既定作成は**Raw Draft/未選択の執筆**と**Workspaceで実Load/Buildされる正式Source**に分ける。正式Sourceの既定化はM2 Markdown Reader・M3 Loader/Pinの実接続後にだけ行う。Guardが選択Rootを止める状況を「書式対応済み」へ言い換えない。

次の**TM-02/03/04/12/14**（#3030/#2823/#988）ではGuardで拒否できる基線を保持し、選択された新Readerの Source actual read/parsed span、Required Root全件、同Meaning ID旧新衝突、Role/Operator Pin・Builder/Consumer実消費へ移行する。同一Subjectを選んだ別Checkoutの旧/新結果を独立Expectedと比較し、単にGuardを消しただけの `validate ok` は偽成功として拒否する。旧Source固定Gateや`.tei/decisions/`は自動改変しない。

### 58.2 Host/Command境界の新しいmain実装を過剰資格化しない

| 到達した実装 | 確認済みの範囲 | 依然不足する範囲 |
| --- | --- | --- |
| [#3032 main統合](https://github.com/hat47x/tei-platform/pull/3032) | `issuedBy`を発行記録に含め、要求Capabilityの自己発行を拒否する構文/検査を導入。さらに実PostgreSQL判定6件が[専用Workflow #42235336807](https://github.com/hat47x/tei-platform/actions/runs/42235336807)で**非SKIP PASS**（実ログで6件の `--- PASS` を確認） | 発行者文字列は署名ではない。Holder/Issuerの真正性、必要Grantの全網羅、実Host許可・業務Owner受理を証明しない |
| [#3033 main統合](https://github.com/hat47x/tei-platform/pull/3033) | ReferenceDataExecutorがAppliedOperation（kind/data/key）を報告し、Command Invokerが宣言したEffectと照合、記録/CLI・Studioへ限定表示。Go CI [Health #42235921102](https://github.com/hat47x/tei-platform/actions/runs/42235921102)はsuccess | **Appliedを報告しないExecutorの経路が残る**。Optional Reportから実DB全操作、SQL文、外部副作用、一括効果完全性を推論しない |
| [#3034 main統合](https://github.com/hat47x/tei-platform/pull/3034) | 既存subject/action/resourceの認可規則を、明示的に設定する`command.Invoker.Authorizer`へ接続。拒否と規則評価不能を区別し、無許可時にExecutorへ到達しない限定試験。実PG workflow [#42238564104](https://github.com/hat47x/tei-platform/actions/runs/42238564104)はsuccess | Callerが正しく現在のAuthorizer/主体/Targetを注入した証拠・正式Grant正本・実Operator全経路での強制・本番資格は別 |
| [#3035 main統合](https://github.com/hat47x/tei-platform/pull/3035) | `access.Ground`で発行Scopeを環境宣言→実行時のResolutionへ明示束縛。宣言外Path・別Capability・自己解決・過大Accessを拒否する限定試験。exact HEAD [Health #42239556042](https://github.com/hat47x/tei-platform/actions/runs/42239556042) はsuccess | **`-grant-declaration`なしの発行をwarningで許す経路が残る**。`issuedBy`/`resolvedBy`は署名無しでIssuer真正性を保証せず、発行時の静的照合は現在Host/Providerの実認可とは別 |

**実効果の検査順に残る欠陥候補**：[#1174の既存評価コメント](https://github.com/hat47x/tei-platform/issues/1174#issuecomment-6688069663)は、`checkAppliedOperations`が実Executor呼出しの**後**に走るため、事後拒否でも独立Tx Commit済みの効果を取り消せないこと、Applied=nilでは実Effect=0が言えないこと、同じKindなら未宣言Dataの追加Appliedを許す分岐（F-SameKind）をSource-staticで特定。これは実PG不正作用の再現結果ではない。受理済みEffect/Relation/Targetの必要範囲と実ProviderのCommit前Fenceは#1174/#1129/#2598へ維持し、#3033の局所テスト成功を既存Issueの受入完了へしない。

**安全上の境界**：①Declared capability／Environmentの必要Grantが実在、②それを現在のHostが権威的に発行・解決、③対象Providerが実行直前に検査、④実Applied/Commit/Unknownを独立Read/Receiptで検証、の各段階が揃わなければ、形式検査やInvokerテストだけで `authorized-and-effect-verified` のClaimを出さない。事前の業務意味のOwner Acceptedと、Hostの実行時付与は独立（#1174/#2616/#1130）。

### 58.3 別Issueへ渡す二つの短い一改訂試験

1. **P0-FM-Reader（#3030/#3031/#2823）**：公開 `sales.order` あるいはOrg CaseRecordの**同じSource/Owner Expected**を、legacy `*.tei`に対する現行実Loader／新 `*.tei.md`を拒否するGuard／真正な新Reader（実装後）の3条件で対照。各段階の source-root required、actual parsed、selected Language/Role、source span、Build consumedを記録。まだ新Readerがない間は `not-implemented`、必要Source未読込でGreenならfalse-pass。
2. **P1-Grant-Effect（#3032〜35/#1174/#2564）**：一つの公開Command/Targetについて (A) 現在のIssuerとDeclaration/Resolutionが正当、(B) 未宣言Capability・宣言外Path・自己発行、(C) 実行結果Unknown／部分Applied非報告を独立Expectedで比較。`issuedBy`文字列を偽装できる負例は本番Host境界の未充足として残す。OperatorがAppliedを未報告なら `reported-effects-only` とし、実Provider全作用のVerifiedにしない。#3032の実PG6件をこの縦断の全成果と誤算しない。

**非目標／変更の資格**：同じSubjectを核にするが、Source改名、Ownerの意味受理、Host付与、実Provider Commitを一つの無差別Global Gateへ統合しない。TMDの第二Fixture正本や新認可Engine・Host/Registry/Scenario Runnerも追加しない。ここで実施したのは別PRの実コード/CI・Ownerの最新版計画を照合している**docs-only更新**。新 `*.tei.md` Parser/Builder/Reader、P0-FM-Reader／P1-Grant-Effectの新実行、署名付きIssuer、本番実DB、Owner受理は**NOT RUN**。

### 58.4 2026-11-06深夜の進展：未設定拒否・Unknown保持・Grant照合と外部Effectの回復停止線

**同じ必要条件を満たす責務は「存在」と「実行」を分ける**。[#3042 main統合](https://github.com/hat47x/tei-platform/pull/3042)で、`command.Invoker.Authorizer == nil`なら`COMMAND_AUTHORIZER_UNAVAILABLE`、宣言したPeriodの実状態を取得できなければ`COMMAND_PERIOD_STATE_UNAVAILABLE`としてExecutor前に拒否する。検査順は**Authorize→Period→Execute**。参照用`ReferenceAuthorizer`は**意図的な全許可の明示**にすぎず、実Principal・本番Grantの証拠ではない。同PRのexact HEAD `719afd61`の[Repository Health #42251024310](https://github.com/hat47x/tei-platform/actions/runs/42251024310)・[認可/Effect実PostgreSQL #42251024219](https://github.com/hat47x/tei-platform/actions/runs/42251024219)がsuccess。

[#3042](https://github.com/hat47x/tei-platform/pull/3042)は`Executor.Execute`の**後**で宣言Effect/Outcome/Applied不一致を検出した場合も、独立TxがCommitしている可能性を排除できないため`EffectUnknown=true`としてLifecycleのunknownに残し、同一Owner/Generationの盲再送を拒否するよう修正した。これにより旧§58.2の『事後拒否が通常のrejected扱いとなる』部分は**歴史的な旧挙動**である。ただし**実Commitの取戻し、効果なし、実外部Providerでのno-replay保証を証明したわけではない**。同種だが異なるDataへの追加AppliedをKind一致だけで通す[現main `applied_check.go`](https://github.com/hat47x/tei-platform/blob/main/internal/command/applied_check.go)の分岐（F-SameKind）も残り、Scope/Relation/Targetの適合は#1174/#1129/#2598が所有する。

| 新しい実装／実施 | 達成した局所資格 | 未充足の後続資格・停止線 |
| --- | --- | --- |
| [#3047/#3048 main統合](https://github.com/hat47x/tei-platform/pull/3047) | `GrantAuthorizer`がCapabilityの**宣言要求**と呼出し側の**発行済み許可**を比較、無宣言/部分許可を拒否。参照Goの5件を[Evidence #3048](https://github.com/hat47x/tei-platform/pull/3048)に保存 | 実CLI/Studio/ProviderにこのAuthorizerを構成・現在Grantを注入する経路は未接続。付与ファイル作成やIssuer署名を担わない |
| [#3038 open](https://github.com/hat47x/tei-platform/pull/3038) | 参照Exporterの**読取側**で宣言とGrantを突き合わせる候補、実ファイルで未宣言能力を拒否する限定試験 | openで未統合。`-declaration`未指定時は既存経路を維持し、解像された動的Pathはまだ読取側に渡せない |
| [#3049 main統合](https://github.com/hat47x/tei-platform/pull/3049) | CLIの未知Commandをflag有無に関係なくstderr/exit2へ、実DispatchとUsage一覧をASTテストで照合。exact HEAD [Health #42253805878](https://github.com/hat47x/tei-platform/actions/runs/42253805878) success | `tei commands`のread-only Tool広告対象やAI/外部Harnessでの実Invoker成功は別。CLI Usage列挙だけでCapability利用権を推論しない |
| [EKI #776 main統合](https://github.com/hat47x/eki-fabric/pull/776) | Lease失効回収でAttemptを**閉じてから台帳項目を削除**、猶予中/結果あり/Close失敗は保持。限定Go試験とverifyがPR記録で成功 | **外部Effect可能性をTaskに宣言する印が未接続のため、現在は全件retryableとして閉じる**。別プロセス縦断と実外部作用/重複回避は未証明、[#745](https://github.com/hat47x/eki-fabric/issues/745)のOwner判断待ち |

**P0の新たな停止条件は新Gate名を設けず既存の実行票へ追加**：同一Task/Operationに対し、TEIの`EffectUnknown`／過去Receipt／新AttemptのIssuer+Tenant+Target+OperationIDをEKIの`lease_expired`/retryable判断へ無条件に写像しない。Leaseの終了は『結果が届いていないAttemptが閉じた』証拠であり、**外部作用がなかった証拠ではない**。外部Effect可能性が未知なら自動再送の資格はunknown/調停待ちとして保持する設計を#745/#1174へ提案（EKI #776の現実装がこれを実現済みとは言わない）。実Provider側Operation identity/Receipt照会と別Connection Readにより重複副作用が排除される場合だけ、Ownerの受理済みポリシーで再投入可を判定する。新分散Host/Transaction Managerは不要。

**文書・所有境界の現況（2026-11-07夜）**：`*.tei.md`移行の[main統合済み #3030 M0〜M6/TM-01〜14](https://github.com/hat47x/tei-platform/pull/3030)をAuthoring/Parser/Loaderの詳細な計画・検収ケースの担当とし、[本Draft #2667](https://github.com/hat47x/tei-platform/pull/2667)はConsumer・Grant・Evidence・Build横断の受渡しだけを所有する。#3030 PR本文に残る『docs-only/Draft/main未merge』は**PR提出当時の説明**。main統合で新Readerの実装が完成したわけではない。既存[docs/naming.md](https://github.com/hat47x/tei-platform/blob/main/docs/naming.md)が`.tei`を『標準の技術名称』として記載するのも**現実装の説明**と**新しいAuthoring方針**を区別して次に更新する必要がある。

**検証用語の最新方針**：[main統合 #3039](https://github.com/hat47x/tei-platform/pull/3039)の命名に従い、本文の*test oracle*を『独立検証基準』『独立期待値』『テスト判定基準』として記述し、データベース製品だけ`Oracle Database`と呼ぶ。既存のコード識別子や履歴リンクは機械的に改名しない。旧本文内の残存表記の全面修正は本追補で未実施（#3039担当の整合監査）。

**状態**：実証したのは各PRのGo/限定実DB/CI及びコード静的境界。本追補自体はdocs-onlyで、Issuer真正性、実Grantの発効時点/撤回競合、実Providerでの不一致後Commit/Receipt/独立Read、EKI外部作用ありTaskの回復縦断、TAI外部Harness H2/H3、`*.tei.md`の実Reader、業務Ownerの正式受理は**NOT RUN/NOT ACCEPTED**。既存#1119/#1163/#1174/#745/#3030を再利用し、第二のVerifier/Host/Registryを導入しない。

## 59. Source読込の観測とCLI JSON契約の互換性を同じConsumer視点で再資格化（2026-11-07）

### 59.1 現行mainで確定した観測の粒度

| mainに取り込まれた変更 | 実測・正当に主張できること | なお言えないこと |
| --- | --- | --- |
| [#3073](https://github.com/hat47x/tei-platform/pull/3073)／[#3078](https://github.com/hat47x/tei-platform/pull/3078) | `workspace-files.result.managedFiles[].readByLoader`はLoad中にdomain Loaderの`readSourceForLoad`で取得したSourceにだけ立つ。`sourceRootFindings`は未存在・未知rootや近似名ファイルの診断。PRの参照Workspace実測は**54管理中53件read、README.md 1件未読** | Inventoryに載る≠Required Sourceへ選択された、`readByLoader:true`≠Parser受理、Role occurrence・意味受理・Build/Consumer消費済み |
| [#3069](https://github.com/hat47x/tei-platform/pull/3069)／[#3081](https://github.com/hat47x/tei-platform/pull/3081) | `validate -json`と`model catalog`に`declarationCounts`（`std`前置定義を除外）。0要素なら`WORKSPACE_DECLARES_NO_MODEL_ELEMENTS` warning | 空Workspaceでも**`ok:true`・終了コード0は保持**。0件は『何も宣言していない』という観測であり、想定Subjectが存在しないことや移行成功の保証ではない |
| [Draft #3031](https://github.com/hat47x/tei-platform/pull/3031) | `*.tei.md`の未束縛を`TEI_MD_PARSER_UNBOUND`で止める限定Go/CIがsuccess（別HEADのDraft） | 現mainの正式Markdown Parserや意味的採用ではない。Raw Draftは非管理Rootで編集可能だが、選択済Sourceのロード資格とは別 |
| [#3087](https://github.com/hat47x/tei-platform/pull/3087) | PostgreSQL実行時判断のWorkflowに`main` pushの起動条件・SHA別保存方針を追加 | **特定main SHAで実PostgreSQL試験が完了した証拠**はWorkflow変更だけからは得られない。PR用CI PASSをmain SHAへ流用しない |

**独立した実CLI負例（旧TEI Binary）**：[TAI Draft #270の実行 #42256480002](https://github.com/hat47x/tai-pleroma/actions/runs/42256480002)では同一Workspaceに`order.tei.md`を追加すると**World Fingerprintは同じ、Source Index Fingerprintは変化**、実`semantic-data:sales.order`は1件のままで、新ファイルの内容は未Parseだった。後続の[同一Path原byteのみ変更の実行 #42297500012](https://github.com/hat47x/tai-pleroma/actions/runs/42297500012)も、Source Index変化／World不変／Subject 1件を観測（両runの該当Jobはcompleted/success）。旧`order.tei`を`order.tei.md`へ**単独改名**した別対照ではSubjectが1→0件でも`validate.ok=true`。旧版の負例は改変せず、現在の新Reader/Guardは別Versionで再検収する（#3030 TM-02/03/04/14）。Fingerprint差だけから意味反映済み、World一致だけから無影響・再利用可を推論しない。

### 59.2 `workspace-files`の同じ`v0`名で応答の型が変わった問題を、Consumer互換の受入へ

[#3078の実差分](https://github.com/hat47x/tei-platform/pull/3078)は`tei.cli.workspace-files/v0`の**`result`を配列からオブジェクト**（`managedFiles`／`sourceRootFindings`）に変更。[CLI JSON出力契約](https://github.com/hat47x/tei-platform/blob/main/plan/design/CLI_JSON_OUTPUT_CONTRACT_2026-10.md)も新形を記載した。一方、[main統合 #3092](https://github.com/hat47x/tei-platform/pull/3092)の`TestDiscoverySchemaVersionsMatchActualResponses`（exact HEAD [Health #42296480740](https://github.com/hat47x/tei-platform/actions/runs/42296480740) successful）は**広告したschemaVersion文字列**が実応答に一致するかを調べるが、`result`が旧配列か新オブジェクトかの差は同じ`v0`なら検出できない。既存外部Consumerの互換性は別の要求であり、現状は未合格とする。

**既存#2824の最小固定対照（追加のJSONエンジンは不要）**：旧Consumer（`result`を配列として読む）×旧固定Binaryは正例、旧Consumer×新Binaryは**成功扱いせず明示incompatible／診断**、新Consumer×新Binaryは`managedFiles`を読める正例、新Consumer×旧BinaryはLegacy互換を要求するか事前の版交渉で拒否するかをOwnerが選ぶ。共通Independent ExpectedにはSubject数とreadByLoader数、`declarationCounts`、`sourceRootFindings`、actual required parsedを別列で固定する。`schemaVersion`一致だけ／empty array default／`ok:true`だけの合格を防ぐ。**選択肢は互換な旧v0応答の維持、または新v1と既存Consumer移行**（いずれも未決）；宣言なしに新v0構造を安全と認定しない。CLI v0の版はSemVer一般論から機械決定せず、TEIの契約Ownerが実Consumerの実態と合わせて受理する。

### 59.3 現在の最短実行順と他Laneへの接続

1. **P0-L（既存LegacyのRead）**：#2822/#2823の`required root/subject`→managed→`readByLoader`→実Parser subject/occurrence→独立期待値の充足を、同じSource/Revisionで固定。Rootが未知・未存在なら`sourceRootFindings`、0件なら`declarationCounts.total=0`を表示し、**`ok:true`を十分条件にしない**。既存#2586実Function/Provider read-onlyの進捗を新Markdown Reader未完成で止めない。
2. **P0-M（新形式の実Source）**：#3030 M0〜M6/TM-01〜14を唯一の移行検収IDとして使い、#3031の拒否から、正式Markdown ReaderのSource byte/span・Role/Operator Pin・actual Builder inputまで段階再検収。P0-LとP0-MのSource path/Revisionを混ぜず、同じ業務要求に対する同一独立期待値への合流はP0-Jでだけ判定。横断側TMDは履歴の対応表。
3. **P0-CLI（契約互換）**：#2824の既存CLI出力契約で`workspace-files`旧新両Consumerを実デコード。`tei commands` discoveryの文字列と**応答構造の互換性**を別項目とする。#3092の現状テストは前者に限る。互換判断までAI/TAI等の外部Consumerが新v0を自動で理解するとは記載しない。
4. **G4-SB-0b（オフライン閉包）**：[PR #3091](https://github.com/hat47x/tei-platform/pull/3091)の`cmd/tei`実`go list`隔離Preflightは、外部Module3件が`GOPROXY=off`で未供給という限定実測。**Source自体の失敗やG4 Buildの実証ではない**。検証済み外部Module byte/`go.sum`/Toolchainの事前供給を#2831へ渡す。旧TEI CLIを使ったP0-L／H2-H3の検証へ新しいHard Dependencyを作らない。

**証拠資格**：Sourceコード・既存PR/CI・TAI固定Binary試験の照合に基づくDocs-only追補。`workspace-files`旧新Consumerの実互換試験、`*.tei.md`新Parser、SB-0b供給/実G4、main push実PG結果、実H2/H3四Session、Ownerによる意味受理は本変更では**NOT RUN**。#2824/#3030/#2831/#2514/#1163の所有境界を維持し、第二Source Registry/JSON Contract Runner/Hostは作らない。

## 60. 実外部Harnessと永続化読取の資格を「実行した」と「完全に観測した」に分割（2026-11-07朝）

### 60.1 DSHから実TEI CLIへの初回接続は成立したが、H2比較4枠へ昇格しない

**新たな実行記録**：[TEI #2509のEvidence PR #3113](https://github.com/hat47x/tei-platform/pull/3113)の固定Sourceを確認。DSH `0.2.2-rc.1`/Windows headless、TEI commit `50ebf8014340fd1906f6b35cfa30173e0ee2fc6c`・実Binary SHA256 `186d06ce…69707`、公開 `sales.order` Source、TAI既存Participant Taskで、**9探索試行中1件のParticipant Taskがcompleted**。その実SessionにTool call 56・Tool result 55・TEI実プロセス起動32があり、`commands/version/model catalog/inspect/source-index/validate`の限定的なTool出力を保存。初めて外部AI→Tool→TEI Binaryを一つのSessionで追えるようになった。[TAI #168のRunbook](https://github.com/hat47x/tai-pleroma/issues/168)とのSubject・モデル・Tool範囲同一性はこの既存台帳で維持する。

**それでもH2/H3比較資格は0/4**。9探索試行にはMALFORMED_RESPONSE（モデルが2 Tool JSONを連結、`maxParallelToolCalls:1`でも残存）、INVALID_CONFIG、短い接続確認が含まれ、成功したものだけを事後選抜して四枠に入れない。`read-only`で作業Dir外の固定worktreeを**実際に読めた**ため、ParticipantにReviewer Expected／Skill内容を見せないという実隔離条件が成立していない。`session-log-deepseek.enabled:false`は**実Egress=0の観測ではない**。Windows `workspace-write`のOS SandboxはWin32 5で準備失敗。**先に合成データのみで実OS/Tool Read拒否と送出先を検査し、新GroupでH2→H3→H3→H2を事前割付**する。現H2回答は採点前、H3 Product Skill実Loadと外部独立Expected採点はNOT RUN。

**Tool Resultの全件性も分離**：#3113の`cli-model-catalog*.txt`は両経路とも**8,192 byteでJSON途中切れ**（先頭断片Fingerprintの一致のみ）。完全Catalogの全件一致や`Required Source`全ての実Read完了ではない。PowerShellの`CannotCreateTypeConstrainedLanguage` stderr混入、原DSH JSONL全体が非収録という証拠上限を[#2509の点検](https://github.com/hat47x/tei-platform/issues/2509#issuecomment-6699498276)で保持。`Tool completed`・stdout JSONの部分取得・独立回答採点・ネットワーク非漏出を別の欄にする。CLIの`ok:true`、`declarationCounts`、`readByLoader`だけではResult全件性を満たせない（§59、#2824）。

### 60.2 Persistedを「Receiptの存在」と「資源の現在値」に分け、効果の認定を先に進めすぎない

| 現行実装・証拠 | 直接確認できるもの | まだ示していないもの |
| --- | --- | --- |
| [main #3112](https://github.com/hat47x/tei-platform/pull/3112)／[main #3118](https://github.com/hat47x/tei-platform/pull/3118) | Executorが宣言効果を返さない場合`ReportedEffectMissing`を記録。呼出し側が`commandadapter.Executor.Effect`を渡した場合はCommitted/Replayedの`Result.Effect`に掲載。指定なしはreceipt-only互換 | 宣言効果を返したこと自体は、実ProviderでそのDataが更新され、現在値が読める証拠ではない |
| [main #3124→#3155](https://github.com/hat47x/tei-platform/pull/3155) | 初期#3124の三状態は歴史的な実装。現在mainは`not-applicable`／`not-observed`／`receipt-observed`／`resource-observed`を区別し、**`persisted:true`はresource-observedのときのみ** | `DurableStore`はComposition申告。観測結果の時点・権限/Scope・レプリカ・Relation全件性は別資格。`not-observed`を未保存・no-effectと読まない |
| [main #3127→#3155](https://github.com/hat47x/tei-platform/pull/3155) | `Lookup=true`でReceiptを照会すれば`receipt-observed`、`ResourceRead=true`で別pool接続の`objects.version/value`とCommit payloadの一致を確認できれば`resource-observed`。詳細観測法を区別する | Receipt成功では**persisted:false**。資源値一致も契約で選択した1資源・その読取時点の限定証拠であり、全関係・外部Effect・業務意味の検証ではない |
| [main #3131](https://github.com/hat47x/tei-platform/pull/3131) | Receipt Lookupの実PG CI [#42318032659](https://github.com/hat47x/tei-platform/actions/runs/42318032659)をexact HEADで確認、対象Testは**非SKIP PASS**（3実行）。ローカルDSN未設定でのskipと区別 | これはReceipt-only負例。追加の[main #3155](https://github.com/hat47x/tei-platform/pull/3155)の実Resource Readは別試験であり、合算して業務全件Readの証明にはしない |

**#2998/#1174の独立Expectedに加える二つの不一致**：P-REC1：同OperationIDのReceiptは存在してLookup成功、しかし対象の`objects`／業務Readは変化・欠損・別Tenant/Replica遅延で現値と不一致。このとき`receipt-observed`だけは真、**`resource-effect-observed`は未充足**。P-REC2：`objects`に変更が見えるがReceiptが消失・不読込なら、観測対象が違うので`PersistenceState=not-observed`とDataの現値を自動同一視しない。必要なら既存CLI/Studioの`persisted:true`に**何を観測したか（receipt／resource、対象ID、時点、独立接続）**を結び付ける小さなProjectionを#2998/CLI契約担当が検討する。新しい永続化真理値/Proof Storeは設けず、Value/Commit/Receipt/Observer/Consumer・現在Grantを別Claimのまま保つ。

**EKIとの重要な非推論**：[#745](https://github.com/hat47x/eki-fabric/issues/745)のLease期限切れ・Attempt閉鎖は実Data作用0の確認ではない。Command `EffectUnknown`→EKI自動retryableへ変換禁止は§58.4に維持。Receiptが存在しても、そのReceiptのOperationID/RequestHash・Tenant/Target/Grant epochが別Attemptと同じとは限らず、別冪等Keyの再送可能性は独立に判断する。

### 60.3 優先順：新Blocking Gateを増やさず、観測欠落を実試験で詰める

1. **外部Harness**（TEI #2509/TAI #168）：まず`pwsh`等のTool Resultの**完全受領**・raw stdout/stderr分離・Reviewer資料/Skill rootへの実Read拒否と送出範囲を、合成公開情報だけで計測。新GroupのH2一セッション→H3実Skill Load→残り二セッションとITT/PPの分母固定を継続。H2 Tool call実測は達成済みだが、H2比較のEligible 1/4ではない。
2. **実Provider**（TEI #2998/#1174/#2564）：#3131 Receipt実PGと**#3155の別接続ResourceRead実PGを達成済みの限定基線**とし、次は同一OperationID/Target/版で現在のTenant・Relation多重度・必要集合・Replica時点・Grant失効を独立期待値へ固定。ReceiptとDataを混同せず、Consumerには`receipt-observed`／`resource-observed`の区別を表示する。
3. **実Builder**（TEI #2831）：[main #3091](https://github.com/hat47x/tei-platform/pull/3091)の隔離Module preflightで`missing-module`を観測済み。[Draft #3130](https://github.com/hat47x/tei-platform/pull/3130)の空・stdlib-onlyで`go list`結果に**非標準対象Moduleが0でもsupportedとする空虚な合格**を`unknown`へ直す候補を独立検証。外部Module事前供給（SB-0b）、実G4 L1/L2/L3は依然別資格で、H2/H3実CLI接続の先行条件にはしない。
4. **検証基準/Attestation**（TEI #2514/TAI #514）：OKF Attested Computationは定義・typed params・実Executor/Receipt・決定的Attesterを既存Evidenceに結ぶ**後続候補**。`verified`やモデルの自己申告をOwner意味受理/外部Grant/正しい独立期待値としない。既存H2/H3の初回Blocking条件・試験Task byte・Case IDは増やさない。[意図的Mutant #3128](https://github.com/hat47x/tei-platform/pull/3128)は**絶対にmainへ統合しない**。対象AI Scenario自身の失敗が固定HEADログで確認できるまで新Mutant kill認定なし。

**検証資格と役割（2026-11-07昼時点で更新）**：本§60は実ソース／提出PR／既存CIを照合した**docs-onlyの横断追補**。#3113はOpen（Raw Harness Log完全性・隔離未確定）、#3131と#3155はmain統合・実PGそれぞれ限定PASS、#3130はDraft。`*.tei.md`正式Reader／外部H2-H3四Session／**業務全関係・必要集合・時点を満たす資源Read**／G4 Self-build／EKI外部Effect再実行／業務Owner受理は本追補では**NOT RUN/UNQUALIFIED**。#2667に新Verifier、Host、Scenario Runner、Source Registryを追加しない。

## 61. 現行Resource再読取・DSH副次送出・Build資格・表現型の接続を再資格化（2026-11-07昼）

### 61.1 Resource観測は製品実装へ到達：それでも「全データ受理」とは違う

- **main [#3155](https://github.com/hat47x/tei-platform/pull/3155)**で`DetailedPersistenceObserver`/`ResourceRead`を実装し、PostgreSQL別poolから`objects`の`version/value`を読みCommit payloadと照合。**exact PR HEAD `19f00c7e`の[実PG workflow #42328707393](https://github.com/hat47x/tei-platform/actions/runs/42328707393) はsuccess**、job `128269964949`の`TestPostgresCommandObservesTheStoredResource`は非SKIP PASS（ログで複数回確認）。同実装により`receipt-observed`は`persisted:false`、一致した`resource-observed`のみ`persisted:true`。前節の三状態／Receipt-onlyでpersisted trueという記述は**旧実装の歴史**として扱い、現在のConsumerへは使わない。
- **次の意味上の不足**：`ResourceRead=true`は既定Compositionへ接続済みとは限らず、別Pool Readが比較した1資源の現在値以外、業務Relationの集合・時点・Tenant横断、別Provider/Replica、Grant撤回競合を証明しない。#1174/#2998/#1163は、AのReceiptは有効でもBによる後続更新後にAのPostimageではない負例、同一OperationIDでReceiptありだがResourceRead denied/partial、Read snapshotの時点ずれを既存独立Expectedで検証。Present/Unknown/NotObservedの意味を正本化し、新グローバルProof Engineは作らない。

### 61.2 外部AIの「read-only」は非開示ではない：Readと副次送出の実測が進んだ

- **Open [#3173](https://github.com/hat47x/tei-platform/pull/3173)**のDSH `0.2.2-rc.1` read-only＋ローカルMockによる**合成canary**：許可投影、`raw/secret.txt`、`raw/.env`、cwd外`peer.txt`の4つが2回とも全て`completed`で読め、**モデル入力と`dsh_session_log`（同一eventが2箇所）へ露出**。拒否0件。Sourceは観測資料まで確認。これは機密実データ流出の試験ではなく、**現在のRead防御が期待するH6-E1を満たさない反例**。PRは未統合・全体Health未確定。機密断定や外部本番送出と混同しない。
- **main [#3161](https://github.com/hat47x/tei-platform/pull/3161)**のローカルMockでは、**出荷既定ProfileでSession Log拡張が45,491 bytes**入り、503時は受理水位が進まず同一範囲を後続題名生成要求で再送、200でも`message_stop`なしの空SSEは`STREAM_CLOSED`で水位が進まない。`enabled:false`を設定した別試行の実Provider/Egress 0証明ではない。MockのRequest byte出現≠実Provider側保存。送出統制は`read-only`/Profile名ではなく、OS identity・ACL・mount・network allowlistを実拒否で検証（TEI #2515／TAI #232）。
- **製品の閉じ込め機能とは責務が異なる**：[main #3152](https://github.com/hat47x/tei-platform/pull/3152)の`os.OpenRoot`を用いたFileSystemはジャンクション等でのRoot外**書込み**を実OSで拒否するが、外部DSH Processの許可外**読取り**、通信先、別API呼出しまで制限しない。[main #3166](https://github.com/hat47x/tei-platform/pull/3166)の記録file `0600`はPOSIX owner-only作成の限定保証で、Windows ACL・改竄検出・実Egress防止を証明しない。
- **実CLI作用も広告面だけから決めない**：[main統合済みの実測記録 #3162](https://github.com/hat47x/tei-platform/pull/3162)の固定Binary dogfoodingでは、`view`がread-onlyと広告されても`-execute -operation-log`はJSONL、`-execute -lifecycle-state`はSQLite fileを書いた。`effect=read-only`な見せ方をTool Invocation全体のFile/Network/Provider効果へ伝播させない。実Tool引数/選択Adapter/Host効果を同一Attemptに束縛（#1155/#2507/#2509）。

### 61.3 セルフビルドは実Artifact生成と閉包資格を別々に判定する

- [main #3142](https://github.com/hat47x/tei-platform/pull/3142)は`cmd/tei`に選択された**12件**の外部Go Module（`go.mod`宣言17件とは別）を、隔離供給・`go.sum`の実byte/h1照合へ束縛。未選択5件の欠落は不合格としない。Source Closure/Artifact/G4をそのまま受理する機構ではない。
- [Open #3156](https://github.com/hat47x/tei-platform/pull/3156)のWSL実測は同一Target `./cmd/tei`、供給のみで`buildSucceeded=true`かつ`artifactDigest=sha256:a512ffba…`、一方`ResolveSourceClosure`は`rejected`、したがって**`artifactProducedDespiteUnqualifiedClosure=true`**。実Buildの存在はSource閉包の資格を埋めない。CI/2Build同値/Stage1→Stage2自己再入・外部Module Trustは未確認。G4は引き続きUNQUALIFIED。#2831/#1090/#1251がSource/供給/Artifact資格をそれぞれ持ち、新Builderを作らない。

### 61.4 標準データ構造×標準表現型はFunction内で初めて使用可能：Operation適用とは別

最新のTEI構想に合わせ、**標準データ構造（型・多重度・Identity・時点・Null・取得範囲・数学facetなど）**と、**標準表現型（Panel／TextBox／Select／Toggle／Slider／Table／Tree／Graph／Map／FileSystem等）**を別の候補概念として扱う。ただし標準表現型の**利用可能なInstanceはFunction／Data Context／入力・出力Purpose・Actor Scopeに束縛した場合のみ**とし、表現型カタログ単体を業務Actionや実行可能画面と見なさない。これはUser方針に対する設計引継ぎで、採用済みCore Schemaや実装済みBindingと主張しない。
**最小の4段階**：(1) DataTopology/Facet（Graph 1:N、複合Key、Collection partialなど）、(2) Representation候補と保存/欠落情報の法則、(3) Function Context/Role/View Targetへの明示Binding、(4) 掲示したAction Offerの適用条件・Host Grant・実Effect。Functionを覆うUIは(3)までの成立で、(4)を暗黙付与しない。表示のためのTree化で多重Parentを削除したり、Tableの表示ソート順をPK順、Graph layoutをEdge identity、単一Selectを複数選択に取り違えない。
**実Owner境界**：標準構造・Facetの探索=#2495、Projection loss/逆写像=#2497、Function/Pattern適用=#2507/#2616、Studio/Component=#1824/#997、Authoring=#2621、Provider/Host Effect=#1174。最近[main #3167](https://github.com/hat47x/tei-platform/pull/3167)の実CLI観測は宣言Pattern 5→6でも導出Operation 8が不変、`create/update binding`全件nullで、**宣言Patternと導出Operationの統一条件が未実装**。対応表を文書化した[main統合済み #3172](https://github.com/hat47x/tei-platform/pull/3172)は**文書のみで、W3の結合規則は未採用・未実装**。Read-only commandで宣言Effectが無いからといって書込みActionを補完しない。既存#2497 MPケースと#2507の受入を使い、第二の表示Core/Function Registryを作らない。

### 61.5 PoC自律化の新しい運営境界を認可・意味の捏造へ転化しない

[main #3150](https://github.com/hat47x/tei-platform/pull/3150)でADR-0035のA3機械受理／A4外部不可逆効果の人確認へ開発方針を改訂。PoC内のコード・Issue・計画・試験をOwnerの毎回の手動承認で止めない一方、**リポジトリ内の自律PR統合＝製品側の業務意味受理やHost権限Grantではない**。秘密/顧客情報の送出、外部削除・破壊的移行・対外公開は別A4。本§61はdocs-onlyの受渡しであり、認可定義を変えない。

**状態と実行順**：Resource現在値の単一対象の実PG観測=#3155 completed/success。DSH canary read許可外/Mock送出=#3173のローカル実測だが未merge、実Provider送出は未測定。SB-0b供給検証=#3142 merged、SB-X1実Build≠閉包資格=#3156 Open。Pattern/Representation実Binding/Studio GUI/Function実Provider→Consumer一気通貫/H2-H3四Session/真のG4/外部Effectを伴うEKI回復は**NOT RUN**。最新PRの局所試験を業務Owner受理・新Global Gateへ加算しない。

## 62. 実Design Context→新テスト提案を最短の一Ruleで接続し、Plugin/Function/図面の利用資格を用途別に制限（2026-11-07夜）

### 62.1 現mainで成立したT0と、まだ成立していないT1を一つの計画票へ

| 入口・証拠 | 成立した限定事実 | この入口では未成立の資格 |
| --- | --- | --- |
| [#3264 merged](https://github.com/hat47x/tei-platform/pull/3264) | 同一checkoutの実Go製品CLIで`commands/version/workspace-files/catalog/inspect/source-index/trace/requirements/validate`の**9 JSON応答**を取得、`sales.order`と版を比較。DF-02 Subject欠落、DF-03 Source Index stale、DF-09 Required Source未読込の合成負例を実行。[専用CI #42357451983](https://github.com/hat47x/tei-platform/actions/runs/42357451983) success | `readByLoader`は実Parser/Meaning/Function/Rule/Businessの全必要閉包でも、外部AI/Harnessへの完全受領でもない。Design Contextは**partial** |
| [#3266 merged](https://github.com/hat47x/tei-platform/pull/3266)／[#3268 merged](https://github.com/hat47x/tei-platform/pull/3268) | 既存Scenario SSOTの**6 pinned Step**を固定TEI Binaryで実実行し、4 CLI負例＋2独立検証義務負例を判別。[専用CI #42359069285](https://github.com/hat47x/tei-platform/actions/runs/42359069285) success、mainの[専用CI #42359610226](https://github.com/hat47x/tei-platform/actions/runs/42359610226)も限定T0の後続証拠。局所選択Go **statement C0相当10.1%**、C1未測定 | AIがIssueから**新しい**Test Case/データ規則を作れたこと、独立Rule意味・コードMutant全件、全機能/NFR/C1 coverageは未証明。PR HEADのRepository Healthにはfailureがあるので専用CI PASSと全体Health PASSを同一視しない |
| [Draft #3273](https://github.com/hat47x/tei-platform/pull/3273)／[main #3284→#3296](https://github.com/hat47x/tei-platform/pull/3296) | T1の`line_number=0`新拒否ケースはdocs-only案。既存`context_gate.py`に加え#3296が**Source Root片側欠落／Inventory重複**と、Tool受信の**ファイル欠落・切断・LF欠落・重複JSON member（Unicode同値含む）・NaN・後続JSON**を既存Gateで拒否する負例を追加。[exact-head 専用CI #42371310577](https://github.com/hat47x/tei-platform/actions/runs/42371310577)はcompleted/success。正常時も`boundedReference=partial`／`qualifiedTestGeneration=false`を保持 | Tool返信完全性・Root診断の局所Guardであり、Function/Rule/Business/Pluginの全Required Closure、実AIによる新Test生成、C1/NFR/Owner受理は未証明。[#3296全体Health #42371575373](https://github.com/hat47x/tei-platform/actions/runs/42371575373)はfailureなので専用CIと混同しない |

**成熟した単一ルールの受入列（新Scenario SSOTを設けない）**：公開合成の`semantic-data:sales.order`に結ぶ`line_number_positive` Ruleについて、(A) Required Domain/Function/Business/Validation/Source Root/非表示依存の列、(B) `managed→readByLoader→parsed role/Source span→meaning revision`、(C) Operator/Target/Scope/Grain/NULL・未取得、(D) Rule未カバーの`line_number=0`を**既存Caseと差分で**検出、(E) 生成AIがCandidateにした新しいScenarioを#1180でpinし、#2514がAI候補とは独立なExpected/known-negativeで判別、(F) ConsumerへのTool Result全受領/開示許可、までを一つのRequirement/Versionで追う。RoleやScopeが未取得なら`readiness=partial`のまま**テスト候補の提案は可、qualified Test Generation/Claimは禁止**。独立ExpectedをAIと同一出力から自動作成しない。

**機械的Coverage表示の分母**：Source/RuleのRequired集合と実取得済み集合、Branch C1の対象分岐と観測分岐、Statement C0の対象Go package/selection、Function/Business/HTTP/Graph/Plugin/Provider・NFRの必要Obligation/未測定を**別列**とする。件数0・empty World/未読込、Case本数、`go test exit0`、選択packageの10.1%を全TEIのC0/C1や要件カバレッジへ自動転用しない。Source/Tool水位とScopeが変わればCase候補資格はstale。

### 62.2 設計情報の必要範囲はConsumerによって異なる（「同じQuery」でも全域完備を要求しない）

| Consumer / Feature | Required Design Contextの最小集合 | 実行に入る条件／明示的な非推論 |
| --- | --- | --- |
| **AI-first CLI/Skill/MCP Test Pattern**（#3262/#3223/#2605） | Rule/Function/Branch、実Source/Operator Pin、negative dependency、C0/C1・NFR未カバー、独立Expectedを参照できる識別子 | MCPはCLIと同一内部型付き結果への後続Adapter。発見一覧や`ok:true`だけで正誤判定しない。AIへの非公開Expected値の提示は禁止 |
| **Data/Function/Business図面とCRUD**（[Draft #3263](https://github.com/hat47x/tei-platform/pull/3263)、#1124/#2497） | Relationの同名多重辺/向き/Identity/非表示依存、Function内のRepresentation Instance Binding、Business path、宣言CRUDと実Effectの別欄 | 図の空欄≠関係不存在、Diagram Renderer/SHO文書でLossを検出。実Graph/CRUD Editorと保存逆写像は未実証 |
| **Function→Build Unit→Entry**（[Draft #3283](https://github.com/hat47x/tei-platform/pull/3283)、#2482/#2595/#2604） | Function宣言とOwner Meaning、Required→Loaded→Build consumed、Target/Entry、現在Version/Scope/Grant、実Invoker/Outcome | 関数名が存在・Entryへ広告≠Build収録・実呼出し可能。Go単一バイナリ既定を保持し、需要がある場合だけ複数Buildを試験 |
| **HTTP Response Decision Graph**（[main #3275](https://github.com/hat47x/tei-platform/pull/3275)、#3274） | Business Outcome/Validation/EffectUnknown/Receipt、Server Guard、Transport StatusとClient Reaction、Graph Node/Edge ID | RFC9457 Problem Detailsと409/412/429等の候補は**設計**。202 Accepted≠業務完了、Retry-After≠再送許可。実HTTP Graph→Trace/ClientはNOT RUN |
| **Plugin-first CLI/Studio/Generated Runtime**（[main #3277/#3285](https://github.com/hat47x/tei-platform/pull/3285)、#951/#1251） | Contract/Provider/Role、原子Trait required/default/manual/advisory、selected/linked/actual invoked、Scope/Effect、exact Lock/BOM | D0〜D2、PD-0〜7の新計画はdocs-only。D1は同じread-only Inspector一つをCLI+Studioの**実2 Consumer**で検証。Provider候補を選べた≠Host Grant/Runtime資格。新Resolver/Host不要 |
| **Data Source Workbench**（[main #3276](https://github.com/hat47x/tei-platform/pull/3276)、#3270/#2586） | Data Source/Provider/Actual Field/Row/Scope/欠損、Schema Diff/History unknown、Trait実装、Function Data ContextのTarget Binding | RQ-035でSchema CRUDは標準・DBAは非標準。metadata/introspectやCLI分岐だけで異種DB/ログ/クラウドの実Read可としない。実Provider P0-Rは未実証 |

**横断の重要な削減判断**：各Consumerに全設計モデルを一括取得させない。`required(shape, scope, revision)`と`observed/undischarged/withheld`を**既存#3262のQuery/CLI Projection**上で用途別に提示し、必要な分割取得はcursor・end marker・byte長・part digest・Result受領件数を付ける。8,192-byte切断・Tool Call 56/Result 55の実DSH事例に照らし、**HTTP/CLI/Model/Skillの成功を完全送達としない**。同じデータ形状でも表示/Graph/Schema/Effectで必要な閉包は異なる。全機能に共通の新巨大DTO/中央Proof Storeは導入しない。

### 62.3 Agent/Harness・Trait/BOMの成熟成果はT1に依存させず、資格を交差させる

- **H2/H3**：DSHからH3 Product Skill本文5,482 byte等のTool実受領が限定観測され、TAI #168の既存Checkerは[72/72の限定合成CI #42370311513](https://github.com/hat47x/tai-pleroma/actions/runs/42370311513)でTool Call/Result countと禁制Readを新条件へ追加。ただしread-onlyにおけるRaw/`.env`/cwd外実Read、Session Log副次送出・暗黙`AGENTS/CLAUDE/Skill description`を回避できた証拠はない。**事前割付の比較適格は依然0/4**。DSHをT1の独立Expectedにアクセスさせず、T1のローカルCLI結果をH3受理に転用しない（TEI #2509/#2515、TAI #168/#232）。
- **Trait/Plugin**：[main #3285](https://github.com/hat47x/tei-platform/pull/3285)／[main #3281](https://github.com/hat47x/tei-platform/pull/3281)の`required` vs `advisory`、Default/手動実装義務、合成衝突は計画と受入案。#3278/既存Provider `ConditionalWrite`のManual missing→diagnostic→actual implementation→Conformanceから始め、同じContract/ProviderがCLIとStudioの実Invokeで使われた証拠まで留保。Trait名だけを「その機能が使える」へ昇格しない。
- **BOM**：[Issue #3267](https://github.com/hat47x/tei-platform/issues/3267)のexact Lock/Independent Expectedは[新Draft #3297](https://github.com/hat47x/tei-platform/pull/3297)へ一本化（旧Draft #3269は閉鎖）。Go Referenceには2×2の全4組と、4軸3版で10組等のPairwise候補生成、`AuditWitnessBindings`のSource/Artifact/Target/Scope/Scenario/Expected/Attempt束縛を追加。**Referenceはread-only・synthetic・`qualified:false`**であり、計画Coverage=100%は実組合せConformance=100%ではない。旧HEAD `82fbdaf1`の[Repository Health #42371757204](https://github.com/hat47x/tei-platform/actions/runs/42371757204)は**completed/failure**。失敗した索引登録3件を担当が現HEAD `a6cddd67`で追加したことを`plan/design/README.md`の再読取で確認。**新HEADの[Health #42373235093](https://github.com/hat47x/tei-platform/actions/runs/42373235093)はin_progress**で成功未確認。旧CIの`gofmt`異常も新HEADで解消したとはまだ推論しない。

### 62.4 最小の実装引継ぎと停止条件（既存Ownerの仕事を重複させない）

**P0/T0を基線に固定**：#3264/#3266/#3268の実CLI/限定Scenario/独立負例を現Main基線として保持し、専門家の読める設計票とAI用JSONの同一Source Revisionを比較。#3262のP-1b用の新CLI/Queryを先に大規模実装しない。
**P1/T1最初の一例**：既存`line_number_positive`について`line_number=0`を**新Candidate**として選ばせ、#1180ピン済CLI Runnerと#2514の独立Propertyで誤り検出・不足理由・Coverage差分を判別。CandidateとExpectedを同一AIで自己正当化しない。#3284のConsumer Mutation Gateは既存CIの追加負例であり、製品Query完成ではない。
**P2/二Consumer**：同じ限定Design Contextをread-only CLIとStudio（その後任意MCP）から取得し、欠損・Partial・Source Revision・Representation Slot・Plugin実Provider選択の同等性を検証。静的`tei commands`広告やread-only表記は実Invoke/Effectと別。
**P3/実Provider/HTTP/Graph**：#2586の実Read→Function Data Context→同一Form/Table、#3274のServer/Client Decision Graph、#1174のactual Effect/Receipt/ResourceReadを既存Driver Scenarioで一段だけ合流。未観測外部Provider・NA Role/Grant・Tool Partialはunknown/qualified=false。旧`*.tei`実Provider Readを新`*.tei.md` Reader完成のHard Dependencyにしない（§59）。

**強制しないもの**：初回T1をMCP・三視点編集UI・生成Runtime全Plugin化・全2×2 BOM・EKI外部Task・G4 Self-buildでBlockしない。各レーンは並行検証し、同一Task/Revision/Consumer Scopeの証拠が揃った部分だけ結合。今節自体は**docs-onlyの計画・実証の再分類**で、新テスト実行・Owner Meaning Acceptance・本番Host/External Providerを行っていない。旧頭のRepository Health successがない場合、専用CIの限定PASSだけを掲示する。

### 62.5 最初のPlugin/Design Contextドッグフーディングを、一つのInspectorの選択・呼出・受領で閉じる

**最短のProduct検収切片（P0-D1、既存#1251/#951/#3262/#1180へ追加する期待条件）**：固定公開Workspaceの同じ `semantic-data:sales.order` に対して、PluginのContract K・Provider A/B・Inspector Function・CLI/Studioの2Consumerを選ぶ。表示が同じでも**選択したBではなくAを実Callしたら不合格**。InspectorのResult Schema/Source revision/Scope/unknown、実Call Args/Provider return、Consumerが受けたbyte/digestを確認し、Plugin名がCatalog/Discoveryに載っただけでは合格としない。

| 反証 | 実行時に独立して観測すべきもの | 不合格／保留条件とOwner |
| --- | --- | --- |
| D1-01: CLIはProvider A、StudioはProvider Bを明示選択 | 各ConsumerでResolved Lock→selected→Go link/import→actual invoked Provider/Args→Result full received。Result値が偶然同じでも実Call先IDを比較 | B指定でA実Call/暗黙Fallback/実Invoke証拠なしはNG。#1251/#951、#1824 |
| D1-02: Required Source root片側欠落、CLI JSON途中切断、古いSelection | #3296の既存Context Gateに`sourceRootFindings`・完全受領byte・同Revisionを渡す。`blocked`から正常応答へ自動補完しない | 途中切断で`[]`や`null`を捏造、SourceIndexFPのみを意味同一へ昇格はNG。#3262/#2824 |
| D1-03: Atomic Trait `manual-required`未実装/競合Profile | Required Trait→Fulfillment候補→実Provided Member→Conformanceを同一Contract Pin/Targetで比較 | `manual-required`欠落・同名だけ・advisoryをrequired充足と誤扱いはNG。#3278/#3279/#951 |
| D1-04: Provider A/B×CLI/Studio exact 2×2 | 固定LockとSelected Source/Artifact/Target/Scope、各Tupleの実InvocationとIndependent Expected・Result/Receipt水位、未実行/Skipを分離 | Pairwiseの生成組合せだけやGo Reference `qualified:false`をVerified BOMへ昇格はNG。#3267/#3297、#2514 |
| D1-05: WorkbenchのPostgreSQL行とJSONL logとPlugin未選択 | 既存[main #3287](https://github.com/hat47x/tei-platform/pull/3287)の合成RDB3行・JSONL5行Fixtureを使い、**別々の実Provider Read**→Data Context→Function Representation Slot→CLI/Studio受領を観測 | Fixtureの独立Expected GoテストPASSやWorkbenchモデル表示だけでは実Provider Row/Query/非開示を証明しない。#3270/#2586/#2507 |

**二つの“2”は独立**：同じContractを**CLI/Studioの2 Surface**から使うD1-01は「二Consumer実呼出」確認。A/B Provider×CLI/Studioの**4 exact tuple**はBOM互換試験D1-04。2 Surfaceを確認しただけで全4組のVersion Compatibilityにしない。最初のD1-01はread-onlyの一操作だけに限定し、D1-03/04/05を**D1-01の着手を止めるHard Dependencyにしない**。BOMは追加の候補マトリクスであり、実GrantやRuntimeの正規実装を生成しない。

**MVP受入Packetの列**：`Subject/Source bytes/Selected Source rev/Root and parse completeness/Contract ID+semantic fingerprint/Provider Instance+binary+selection reason/Actual invoked Provider/Consumer surface/Full Tool-result bytes and digest/Trait obligations/Exact BOM tuple/Independent Expected/Evidence observation state` を既存#3262/#951/#3267のread-only投影で用途別に必要最小だけ提示する。APIの巨大共通DTOや新central Plugin Registryは作らない。途中切断・非公開・unavailable/partial・未知を`qualified=false`へ保持。

**今回の証拠上限**：main #3296は局所Context輸送負例のCIがPASS、#3287は異種Source fixtureと独立Expectedの局所Goテスト、#3277/#3285はPlugin-firstの計画・現実装監査、#3297はGo Reference/計画候補で全体Health failure。**D1-01〜05の実2Consumer/Provider再配線・Trait Runtime Fulfillment・正式BOM・実DB/Live Workbenchの同一Attempt E2EはNOT RUN**。実Source/Meaning Owner受理とHost実行Grantは別の証拠であり、PoCの自律PR統合を代替にしない。

