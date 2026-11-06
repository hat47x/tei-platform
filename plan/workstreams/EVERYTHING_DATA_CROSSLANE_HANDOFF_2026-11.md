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

独立Expectedは既存[#2514](https://github.com/hat47x/tei-platform/issues/2514)の検証基準の契約を再利用する。Providerのログだけで物理Read最小性、独立再読取り、外部開示拒否をすべて保証したと表示しない。

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

業務受注の3層更新と、Sensor/Actorの10Hz認知→60Hz運動、文書の親子章節などを、似たEntity名へ改名しただけのFixtureではなく、**異なる時間意味・関係・順序・作用・改版・検証基準**を持つ要求として比較する。必要になるCore特殊分岐、Role/Operator追加、Plugin適用、Studio専用UIの増分を#2018へ記録する（#2495/#2493/#2507）。

### 9.5 同じ構造に対するMeaning／Host／Pluginの独立性を対照試験で確かめる（2026-11-06、提案・未実施）

**位置づけ**：[#2619 FW-01〜07](https://github.com/hat47x/tei-platform/issues/2619#issuecomment-6603196417)と[11/6三領域反証](https://github.com/hat47x/tei-platform/issues/2619#issuecomment-6682217255)は、`status=approved`等の同形Carrierから業務意味や実行許可を推定できないことを既に示す。ここで追加するのは**反例そのものではなく、どの変更がどの資格にだけ効くべきかを一変数ずつ確かめる判別試験**。現行#1180 Scenarioと#2514独立Expectedに付帯する比較観点であり、新しいCase SSOT、Status enum、Runner、Proof Engine、Host/ACL、Gateは作らない。Draft #2896 §19のRequired→Consumed／Claim相対証拠票を参照し、同じ横断ラダーを複製しない。

#### 受理と許可の2×2：構造・対象・試行条件を固定した独立対照

固定の公開合成`order_entry`対象O1、Source/Role/Rule/Provider/Operation payload候補、選択Tenant、Request scopeを共通にし、次の二因子だけを変える。**M**＝対象Function/ActionについてOwner受理済みMeaning/Ruleの資格がある、**G**＝当該時点・主体・Tenant・Resourceに対する現在Host Grantが有効。これは新しい実行判定APIではなく、テストで区別する証拠条件。

| M（意味） | G（現在Grant） | 独立Expected／禁止する推論 |
| --- | --- | --- |
| 未受理／根拠不足 | 不許可 | 構造解析に成功しても確定ActionとしてDispatchしない。意味不足と権限不足の**二つの未充足根拠**を、権限ある検査者向けの票で区別。外部Consumerに非公開のGrant/Rule存在を漏らさない |
| 未受理／根拠不足 | 許可 | 既存Grantを業務Ruleの受理とみなさず、意味資格不足を保持。**Grant alone ≠ accepted meaning** |
| 受理済み | 不許可 | 意味の歴史的受理は保持するが、現在の対象へのDispatchは拒否。**Meaning alone ≠ Host capability** |
| 受理済み | 許可 | 当該対象の**実行前提候補**に過ぎない。Required Source/Guardの全件、Selected Provider、実Effect時Fence、expected stateが不足すれば実行不可。実際の効果・Receipt・業務完了は別観測 |

**直交性の反証**：Mだけ変更してもHostの現在Grantを自動発行・更新しない。Gだけ変更しても過去のAccepted Meaningを自動生成・改変しない。ただし実際の適用判断は両者と明示された依存を再評価する。Owner受理の版更新・Grant失効・Plugin Pin変更・Source A→B→Aは別の失効要因であり、単純な「同じ値なら同じ許可」には還元しない（#267/#1174/#2582/#2600）。

#### 2×2が「両方真」でも失敗する証拠の取り違え：対象・意味・実時点の束縛

**既存2×2の適用条件を明示する補正**：M/Gの真偽だけを独立に判定するのは*要件の分離*であって、二つの`true`を論理積で結合すればDispatchできるというルールではない。Mの受理対象は、Function/Action/Ruleの意味契約と適用範囲・Revision（Global Ruleの場合はTenantを個別に含めないこともある）。Gの対象は実Principal・Tenant／Resource scope・許可Operation・現在時刻とEffect確定条件。**実Request・Target Bindingが両資格の適用範囲に含まれていること**を既存Resolver/Host/Providerの観測と突き合わせる。全フィールドの形式的等値は求めず、異なる粒度の証拠を根拠なしに接合しない。

以下は**新Case IDやScenario正本を作らない既存#1180/#2514への追加対照**。元の公開合成Source・対象候補・独立Expectedを固定し、正常対照と差分一箇所を対にする。実Backendで該当Guard/Joinが存在することは未確認なので、安全な拒否を「現状実装済み」としない。

| 変化させるWitness（それぞれ単独では有効） | 誤った合成と独立Expected | Owner／既存試験入口 |
| --- | --- | --- |
| Mは受注O1の確定Rule、Gは**O2の閲覧許可**に差し替え。あるいは同じBusiness Keyが別Tenantの異Resourceを指す | M=trueかつG=trueの単純積がO1の確定を許可するなら**不正な証拠接合**。実Target/Action/Principal/Tenantの一致が未証明ならDispatchを許可しない。内部では未充足Witnessを報告、外部では開示禁止の別Resource存在を漏らさない | #2598/#1174/#1656、別Tenant同Business KeyのDraft #3016（実PG新試験は未確認） |
| Mは「見積承認」の受理、Gは同じO1への「受注確定」実行Grant。数値・表示・Source Carrierは同形 | 異なるBusiness Action間のMeaningの代用を拒否。Provider read/Grantを根拠に受注確定のOwner意味受理を生成しない。#2926の`EvaluateCompletion`が単体成功しても本番のCompletionではない | #1125/#1126/#2514（#267は仮承認） |
| M/Gは**過去Snapshot／Preview時点**では有効、Effect確定前にOwner受理の対象Ruleが改版、または委任Grantが失効 | Source/Meaningの歴史的受理や以前のReceiptを消去するのではなく、*現在の対象Actionへの適用*を再資格化。ProviderがCommit時にGrant epochを原子的に照合できないなら、原子性を保証したと主張しない。送出後ACK喪失はunknown、盲再送不可 | #2389/#1174/#2600、実Provider保証差のConformance |
| Business Procedureの手作業記録を、同形の自動Workflow Stepとして渡す。あるいはRequired Grant Guardの指定をCallerが省略する | 手作業ならAction/Grantの**not-applicable**を理由にDefinition全体を拒否しない。一方、自動Stepなら実行義務とHost Grant確認が必要。`RunPlan completed`やObserver存在で欠落Guard/実Effect/業務完了を合格にしない | #1125/#1130、既存Draft #2889 OC-RP-01〜03 |
| 部分Readの`未承認=0件`を、完全なTargetSet `L=[]`として見積→受注確定の業務完了へ転用 | 実取得Source/Rowset/Fieldの必要母集団とCoverageが不明なら「承認待ちなし」もOwner完了も**unknown**。NULL・未取得・空母集団を区別。`EvaluateCompletion(L, EmptyPolicy, P)`は供給された証拠の評価器であり、その母集団を実DBから完成させる証拠ではない | #2586/#2506/#2622/#1125（main #2926） |

**独立期待結果の判別力**：最初の対照は、同じ独立Expected/固定Targetのまま *Gのresource/action/tenantのみ* を変えるMutantを既存Scenarioへ一件注入し、明示したEffect対象O1が**一度も適用されないこと**と、Current Grant照合が実際に実行されたWitnessを別々に見る。Mutantを「検出した」だけではDB無作用は証明しない。実Provider/Grantの結合がまだ無い場合は`NOT RUN: target-grant-binding-unavailable`と記録し、静的な期待拒否を実Security PASSへ昇格しない。#2514の既存Mutant/独立Expected・#1163のAttempt証拠に委譲する。独立Expectedに反する場合は失敗をそのまま観測し、後からExpectedを実装に合わせない。

**非ブールの留意点**：未受理は明示Rejectと同義ではなく、Host Grant不在は現在の拒否とProvider照会不能を区別する。開示範囲外の情報を診断に表示しない。上のM/G=trueはすべて**この比較に必要な証拠が揃った場合に限る説明上の略記**であり、実Runtimeの共通Boolean/Authority APIを提案しない。

#### 評価器の判定前に観測の帰属を確かめる：部分反例と空集合証明の非対称性

**追加で確認した実コード上の境界**：[main `internal/semantic/completion.go`](https://github.com/hat47x/tei-platform/blob/main/internal/semantic/completion.go)（取得Blob `512a835ca445d81dd043f8af78ca29df4e7adb3b`）の`EvaluateCompletion`は、呼出側から`Targets[]/Coverage/EmptyJustified/HandoffConfirmed/Conflicting`を受け取り、対象のSource/業務Identity/Rule revision/Tenant/Read snapshotを自力で解決しない。既存[Go test](https://github.com/hat47x/tei-platform/blob/main/internal/semantic/completion_test.go)（Blob `8e9c49c55d8b524edbb3750b2a6ed39ff988b7c6`）では**partialでも適格な違反行を一つ観測すれば`fails`、すべて真でもpartialなら`unverified`**という非対称性を固定している。これは全称条件についての有用な局所推論だが、**違反行の帰属を呼出側が偽った／誤った場合まで安全だという証拠ではない**。#3022の限定静的レビューと#2886 §7の派生値用途区分を再利用し、新Completion Engineを作らない。

| 対照：評価器へ投入する前の帰属（Domain違い・独立Expected） | 許される推論と停止条件 |
| --- | --- |
| 受注O1の**同じMeaning Revision／TargetSet L／Scope／Read cohort**に属する明細L2が`P=false`、L1は未取得 | そのL2が真に適格な反例なら、全件取得を待たず該当する**全称Claimがfails**と判断可能。未取得L1の値や、実Commit/取消・Owner業務完了は確定しない |
| 表示・行ID・`P=false`は同じだが、L2は**別Tenant O2／古いRule版／Preview旧epoch**の観測 | 有効な反例を*現在O1のP*へ流用しない。対象/Scope/Rule/Observation identityが結び付くまで、現行Claimは`unverified`または適切な矛盾診断。関数にそのまま渡して返る`fails`は**呼出側入力の評価結果**であり、現行業務不成立の証拠ではない |
| 対象O1の取得済みL1は`P=true`、未取得L2の有無も不明。あるいは`L=[]`だが一頁目のみの0件 | 局所的な満足は認めても全称Claimの`holds`は不可。Provider/Field/PopulationのCoverageが確定しなければ保留。実`EmptyPolicy`／`EmptyJustified`値を推測で埋めない |
| O1の対象全集合に対する**独立の空証明**が存在するが、観測元は旧Snapshot、別Tenant、または限定ACLで母集団が変わっている | 証明の原Scope/Provider/Snapshot/取得完全性が現在のTargetSet Lと同一の対象を扱うか再確認。`EmptyJustified=true`という呼出引数の存在だけでは証明の出典と新鮮性を保証しない。空集合時の`deny/allow/requires-evidence`の意味は#267 Ownerの採否に従う |
| O1の見積金額600をRule r1でPreviewし承認引用。後に価格改訂で現在Preview700、または別O2でも600。GはO1確定に有効 | 観測値・承認Decision・実Requestが同じ対象でも**承認時に引用した値とRule/Source Revision**が異なれば、現在の確定に採用できるかはOwner Rule次第。過去の600承認を抹消せず、700や別O2の600へ自動移転しない。G=trueから引用値の有効性を推論しない（Draft #2886 §7） |
| 手作業Handoffの対象集合が空、または`HandoffConfirmed`の証拠が別Business Instanceから来る | 単体Evaluatorの空集合分岐による`holds/fails`と、該当Instanceの**人間への授受・確認が業務上必要か**を分離。空時の引継ぎ義務はOwner判断であり、`RunPlan completed`や他Instanceの確認から手作業完了を自動生成しない（#1125/#267） |

**最小の実証単位（既存Scenario/独立期待結果へ付帯）**：公開合成`order_entry`の同一Rule/Source/Expectedで、①適格な部分集合中の`P=false`、②**P/値を変えず**対象のTenant・Business InstanceまたはRead epochのみを変更した非適格`P=false`、③適格な部分集合中の`P=true`、④同じ0件でもRead completeness/独立空証明だけ異なる対照を扱う。呼出側に帰属チェックが未実装なら、(a) **現Evaluator単体の戻り値**、(b) **実呼出しが不適格入力を渡さなかった証拠**、(c) **現在のClaimへ出してよい判定**を別票にする。実データのSource/Target/Query/Consumer紐付けは#2586/#2506、意味・業務完了は#1125/#267、Independent ExpectedとMutation判別力は#1180/#2514、現在のEffect/Hostは#1174/#1163へ戻す。評価器の単体PASSや一時的な`fails`を、現時点のBusiness Completion/不完了・安全な拒否・Host/Provider Effectの証明へ昇格しない。

**状態**：上記の関数・既存Go testは静的Source/実テスト記述を直接確認。*今回*の4対照を用いる新Go/実Provider/実Tenant/CLI・実Reader・Owner受理は**NOT RUN**。既存コードのBUG確定、新API/共通Validity Tokenの採用、既存#267の意味変更は行わない。

#### 証拠の入場資格：`*.tei.md`実消費とGrant発行主体の真正性（2026-11-06追加）

**今回の新たな接合点**：これまでのM（業務意味受理）×G（現在Grant）対照には、Mの根拠Sourceが実Loader/Parserで消費されたかと、Gを発行した主体が本当に信頼できるかという**異なる入場条件**がある。同じTarget/Scope/版を指す説明上の`true`同士でも、入場条件が欠ければ実DispatchやOwner完了をQualifiedとしない。新しい第三・第四のBoolean／全域Proof Engineの導入ではなく、既存Source/Loader・Authority・Effect witnessの**何が未観測か**を示す。

**直接確認した現在地**：
- [merged #3032](https://github.com/hat47x/tei-platform/pull/3032)により`internal/access/grants.go`（Blob `24db243bace5fe38671588fe1c04f72bd15ee7a9`）に`issuedBy`と`AcceptIssuedBy`が入り、名前なし／Capability自身による自己発行を拒否する。既存[grants_test.go](https://github.com/hat47x/tei-platform/blob/main/internal/access/grants_test.go)は第三者名の付与が**この発行者検査を通るだけ**で実Grantを作らないと明示する。`issuedBy`は署名でも検証済みIssuer Identityでもない。付与ファイル`0600`は一般利用者からの書換えを制限するが、同じプロセス・書込み可能主体の偽装不能やHost採否は保証しない。`IssueFile`の記録も署名・実Hostの受理ではない。
- [Draft #3030](https://github.com/hat47x/tei-platform/pull/3030)でAuthoring形式は**原則`*.tei.md`へ統一**する方針に進んだ。一方、main `internal/workspace/loader.go`（現main Blob `d2b480edfaec21feb89e34e658805ea1ed0f7339`、#3078の`readByLoader`追加を含む）は現時点で旧`*.tei`等の選択に依存する。[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)は新形式の黙った未読込への診断を提案するが、実Sourceとして新形式をParsing/Meaning/Build消費した資格はまだ示さない。従来`*.tei`の歴史的Meaning受理を自動削除しない一方、新`*.tei.md`を置いただけで現在のMeaning/実入力の同一性も保証しない。

| 一箇所だけ変える対照（既存#1180/#2514 Expectedへ付帯） | 許される局所結果と、上位Claimの停止線 |
| --- | --- |
| **Source移行対照**：同じ業務内容の旧`order_entry.tei`のみ／新`order_entry.tei.md`のみ／両方に異内容で同Subjectを置く | 新形式が実Loaded/Parsed/Consumedされていなければ現在Meaningの資格は**unresolved**。旧版を黙って採用して`validate ok`でも新形式を検証済みにしない。両者が衝突した場合は無根拠なlast-winsを不可とする（#2621/#2823/#3030） |
| **Issuer記述対照**：同一Grant内容で`issuedBy`を空／Capability自身／第三者名へ一因子だけ変える | 現`EncodeDocument/AcceptIssuedBy`の空・自己発行拒否と、第三者**文字列のみの検査通過**を区別。第三者名があっても署名・発行権限・Host採用・実Current Grantは**未確定**。`ParseDocument`と`ParseGrants`の用途を混ぜない（#3032/#1174） |
| **Authority結合対照**：`issuedBy`とPermission Scopeは同じで、承認されたIssuer／実Principal・Tenant・Operation・Resourceの束縛が有る／無い | 署名やHostの受理契約がない環境では、Issuer文字列が別人を名乗っても現関数のローカル検査を通り得る。そこで**Product Grant qualified**とは言えない。Caller要求を「承認済み」と自動投影せず、未確認要素とDisclosureを区別する（#1174/#1163） |
| **効果境界対照**：同じM、同じ名前付きGrantで実ProviderへのDispatch前／Effect時Grant失効／Commit応答喪失 | Preflight/自己発行拒否の単体PASSを**Commit時のGrant Fence**やDB無作用へ代用しない。実Effect時に拒否できる保証が必要ならProvider/Hostの実境界を検査し、できなければunknown/unqualified。ACK喪失時は同OperationIDのReceipt Lookupを優先し、新Key盲再送をしない（#1174） |

**優先する検収の最小差分**：①既存[Issue #2823](https://github.com/hat47x/tei-platform/issues/2823)と[Draft #3031](https://github.com/hat47x/tei-platform/pull/3031)にReader/Parser/Consumedの観測を委譲、②[#3032](https://github.com/hat47x/tei-platform/pull/3032)の既存Issuer局所試験をそのまま用い、Issuer文字列だけでは真正性を証明しないことをOwner検収票に残す、③一つの`order_entry` Functionで**実Consumed Source・Owner Meaning revision・Hostが受け入れた発行権限の根拠・現在Target/Grant・実Provider/Consumer**を[#1163](https://github.com/hat47x/tei-platform/issues/1163)の一Attemptに紐付ける。Source形式移行を扱うためだけに書込みGrantやProduction Commitをread-onlyへ一律要求しない。

**採否・実測の上限**：#3032のコードはmainへ統合済み。追加で[Actions #42235336807](https://github.com/hat47x/tei-platform/actions/runs/42235336807)の実Job `127951340786`のログを直接確認した。**PR #3032のmerge検証Commit `b2a05196dd779e54764de66549d94d3de0e3878d`**（PR HEAD `c2b7e44042fb7148901cd6e2678983012b415a7c`を含む）でPostgreSQLを起動し、期間closed/open・Effect Observer不能・重複拒否・Grantなし/ありの**6実GoケースがPASS、対象6件SKIPなし**。これは実Runtime Decisionの**限定PG証拠**に昇格できる。しかし**Issuerの真正性・現在Host Grantの発効点・別Tenant同Key・本節のSource/Issuer誤接合・業務Meaning受理**を実証したものではなく、この横断対照の追加試験はNOT RUN。#3030/#3031はDraft、`*.tei.md`の実Parser対応・移行・同一Meaning再受理、Issuerの署名/独立Host承認、現Grantと実DB Effect/Owner正式受理は**NOT RUN／NOT ACCEPTED**。既存#2621/#2823/#1174のOwnerと#1119 Gateを維持し、新権限Issuer Registry/共通Host/新Writerを設けない。

#### Commandの認可二系統・追加Scope差分・実保存を独立資格にする（2026-11-07、main実装差分を再照合）

**前回記述の現況訂正**：旧版#3033の`checkAppliedOperations`を前提にした「同種別・別対象の追加操作を無条件に許容する」は**最新版の説明として不十分**。main統合[#3055](https://github.com/hat47x/tei-platform/pull/3055)/[#3057](https://github.com/hat47x/tei-platform/pull/3057)/[#3060](https://github.com/hat47x/tei-platform/pull/3060)/[#3062](https://github.com/hat47x/tei-platform/pull/3062)/[#3065](https://github.com/hat47x/tei-platform/pull/3065)では、**同Kind・別Dataは`Difference{code:additional-scope}`として記録・表示**し、`Applied=[]`は`AppliedUnobserved=true`と明示する。該当Diffがある場合でも**参照実行は成功し得る**。これらの記録を独立したProvider実Effect観測や、関連Relationを確認した「業務上許されるCascade」と混同しない。

**現mainの直接確認（2026-11-07）**：
- `internal/command/applied_check.go`（Blob `9158488a87276d4c6903caa5a26edc90c9815f8f`）は`[]Difference`を返す。同Kind・別Dataは`additional-scope`の未解決差分として保持し、別Kindや宣言効果の実報告欠落は契約違反。差分自体はCommit阻止条件ではない。
- `internal/command/operation.go`（Blob `bfa916a8d97cbd00faac84a4f35e9f511bd578a8`）は`Applied/Differences/AppliedUnobserved`を分ける。参照実行の`MemoryRecorder`はExecutorの報告を記録するが、外部DBの完全な実効果を独立観測しない。`invoke.go`（Blob `9f9099b6a9565d54597f2feb87bb41ce3d0be8d9`）は**Executor実行後に差/Outcomeを検査し、成功経路でRecorderを呼ぶ**。契約違反時は`EffectUnknown`となり、正常`OperationRecord`の不存在を効果ゼロと扱えない。
- [#3047](https://github.com/hat47x/tei-platform/pull/3047)導入`internal/command/grant_authorizer.go`（Blob `ba93c741620a6aea0cb904d1e772b5c927ef0eb1`）の`GrantAuthorizer`は**Capability宣言PermissionとCallerから渡されたGrant Specの被覆**を検査。対象Capabilityの宣言エントリ欠落を拒否するが、Issuer真正性・Host受理・実Resource別のSubject Ruleの検査ではない。[#3064](https://github.com/hat47x/tei-platform/pull/3064)後の`internal/security/legacyauthorization/commandbinding/commandauthorizer.go`（Blob `af12a1899d778eeb26107f6ffa4ebe8ce26d2412`）は**Subject＋Action＋Resource**を既存Ruleへ渡すが、Permission Spec/発行経路の真正性は検査しない。
- **構成上の独立性**：`Invoker.Authorizer`は**単一の`Authorizer`インターフェイス値**。その位置に`GrantAuthorizer`または`CommandAuthorizer`の一方を入れて局所PASSしても、**両方の要求が製品実行時に評価された証拠にはならない**。すべての純粋/Read-only操作に両方を一律義務化する提案ではなく、当該業務Effectに必要とOwner/Hostが明示したobligationの充足を同じRequestで確認する。Compositionの実装方式は既存Host/#1174のOwner判断。
- [#3035](https://github.com/hat47x/tei-platform/pull/3035)/[#3038](https://github.com/hat47x/tei-platform/pull/3038)の`Ground`/Consumer照合は発行Scopeを環境宣言に制約する局所機構。現`access/ground.go`（Blob `f68dde6a14ccbfb1e5fa9e615a0fcc5978f4a1cc`）でも`issuedBy/resolvedBy`文字列の署名・Hostによる真正性確認、実Effectの現行Grant Fenceは証明しない。
- [#3054](https://github.com/hat47x/tei-platform/pull/3054)の`committed=true, persisted=false, durability=none`は**参照メモリUnitOfWork成功と永続化なし**を表示する有限な改善。実Provider経路での`committed/persisted`の意味強化は`plan/design/COMMAND_BOUNDARY_SAFETY_2026-11.md` §8でP1〜P3の未採用候補として残る。

**重複を増やさない最小対照**（公開合成の同一`order_entry`、Meaning/Operation/Fixture/独立期待結果を固定し既存#1180/#2514へ引き継ぐ）：

| 条件のみ変更 | 実装が示す有限な結果 | 未充足／独立Expectedと停止条件 |
| --- | --- | --- |
| **認可片側だけ**：同じO1のWriteについて(a)Permission Specは被覆だがSubject/Resource Ruleはdeny、(b)RuleはallowだがPermission Specは未被覆 | 選択された一方のAuthorizer単体が局所的にPASS/denyし得る | 当該Effectに必要な両obligationの**実評価証拠**がなければProduct Writeの許可成立を主張しない。受信したRequestのSubject/Action/Resource/Tenantと能力＋Grant scopeを別列に残す |
| **同種別の範囲差**：宣言`update sales.order/O1`と、報告`update sales.order/O1`＋`update unrelated.data/U1` | 現行`checkAppliedOperations`は**`additional-scope`差分を返す**。差分は記録され得るが、成功経路を遮断しない | U1が適法なCascadeかの判定は**Relation Policy/対象集合・実Provider Report/独立Read**を使う。差分「表示された」ことを「業務上許容済み」へ昇格しない。実U1不正更新は観測していない |
| **報告能力の差**：実Providerが`Applied`を列挙する／列挙できず`[]` | 列挙不可は`AppliedUnobserved=true`として観測可能 | 未観測≠効果0件。`Differences=[]`や`committed=true`でもProvider Physical Effectの完全性・独立Read・Owner完了はunqualified |
| **事後契約違反**：Executorが書込み後、別Kindの`Applied`や未宣言Outcomeを返す／ACK喪失 | `EffectUnknown`の失敗診断が返り得るが、正常Recorder経路には到達しない場合がある | `OperationRecord`欠落や参照UnitOfWork rollbackをDB無作用の証拠にしない。同OperationID/AttemptのReceipt Lookup・別Readerと再送停止を#1174/#1163の既存契約で要求 |
| **Consumer誤読**：同じ`committed=true`でも`durability=none`の参照実行／実Provider・Receiptあり | 参照実行の`persisted=false`表示は成立 | 外部Harnessが`committed`一項目だけを読んで業務保存済みと判定しない。実Providerの`persisted`判断とCLI schemaVersionの変化は既存Ownerで確認 |

**優先順位と採否**：P0は(a)認可片側だけのすり抜け候補と、(b)`additional-scope`差分の**保存/Consumer表示と業務許容の非同値**を、既存テストに**同じSource/固定Expected**で接続すること。局所Go試験・Reference報告・Schema Version広告を本番Host/実Provider/Owner Meaningの資格へ格上げしない。#2514（独立検証基準/Mutant）、#1180（Scenario SSOT）、#1174（Host/Effect）、#1163（同一Attempt/独立Read）、#1129（Cascade/業務Rule）、#2586（実Provider Data）へ実作業を委譲。新Authorizer Aggregator、全Target共通Grant Engine、第二Effect Registryを**この文書で先行採用しない**。

**証拠状態**：mainの実Codeとmerged #3033/#3034/#3035/#3038/#3047/#3054/#3055/#3057/#3064を直接照合。現行のReferenceテストは実際に`Differences`を保持するが、上の二系統Authorizerの同一Request結合、無関係U1の実DB変更、外部ConsumerでのPersisted資格、正式Owner受理を**今回実試験していない（NOT RUN／NOT ACCEPTED）**。#3051は空集合Deny/Unknown挙動を変更する未統合Draftであり本節で採用しない。

#### 副作用の到達範囲とAttestationの適用範囲を閉じずに合格させない（2026-11-07）

**最新mainで成立した局所改善と、なお残る横断条件**：[#3070](https://github.com/hat47x/tei-platform/pull/3070)でCommandの`checkDatasourceAccess`は`Application.Databases`・`database.uses semantic`を参照し、宣言された**主たる`plan.Effect.Data`**がRead-only datasourceに属するときExecutor前に`COMMAND_DATASOURCE_READ_ONLY`を返す（main `internal/command/datasource_access.go`、取得Blob `a332c4ba7df2a354ab502c68f5e57c28dd1314c9`）。[#3055](https://github.com/hat47x/tei-platform/pull/3055)の`additional-scope`はExecutor後の別Data報告を差分として残す。**主対象の事前チェックがPASSすることと、実作用が到達する全Data／Provider／Tenantの制約充足は別**。両方の局所機構を採用しつつ、一方のPASSで他方を証明しない。

| 固定Source／一因子の負例 | 現行Sourceから言えること | 独立期待結果・必要な次の証拠 |
| --- | --- | --- |
| **関連先Read-only**：公開合成O1の宣言`delete sales.order`はRW、Cascadeの候補に`ledger.entry`（別namespace、RO）を含む。関連先をRW/ROだけ入れ替える | 事前`checkDatasourceAccess`は宣言主対象`sales.order`のみを見る。実行後`Applied`に`ledger.entry`が報告されても`additional-scope`に留まり、RO強制の証明にはならない | Cascadeの越namespace適用がモデル上**非対応ならその時点でunsupported**、対応するなら明示Relation/Owner Effect Policy・選択Datasource・Effect直前のProvider fenceを確認し、RO先に副作用がないことを**独立Read**で検証。単に差を記録して成功表示するだけでは不十分 |
| **同namespace二つのDatasource宣言**：`database A uses sales (read-write)`と`database B uses sales (read)`をAppで両方参照し、宣言の順序だけ反転 | 現`checkDatasourceAccess`は最初にnamespaceが一致したDatasourceで`return nil`またはRead-only拒否するため、**選択Alias未確定のまま記述順で可否が変わり得る** | 実際に使うDatasource alias/Mappingと、重複の有効性／優先則をOwner契約で決める。未確定のときは順序による許可を実権限PASSに昇格しない。新しい一律禁止のDB選択規則をここで採用しない |
| **主対象のDatasource Mappingなし**：Effectは`sales.order`、Appにそれを覆うDB宣言が存在しない | 現検査は推測して拒否せず`nil`を返す。これは**検査対象外／未解決**であって、実Providerへの書込み権限付与ではない | 実Provider選択・Mapping・必要Host GrantとTarget Read/Write fenceの観測を要求する。参照Commandの成功やRO違反なしの診断を実Target資格としない |
| **OKF Attested ComputationからBusiness Completionを導く誤接合**：同じ計算結果と有効Receiptだが、Inputがpartial Read／別Tenant／旧Operator版。あるいはOwnerが意味受理を撤回 | 決定的Attesterが検査できるのはPinされた定義・実入力・実行結果等、**そのAttestationが対象とする限定Claim**。実行・再読取・署名・Owner／Host資格は追加の証拠 | [#2514](https://github.com/hat47x/tei-platform/issues/2514)の既存独立検証基準と[#2614](https://github.com/hat47x/tei-platform/issues/2614)の限定Product Skill候補を利用する。Attestation=`verified`から`CoverageComplete`／Owner Meaning Acceptance／現在Grant／Provider Effectを推論しない。候補のVerifierを新Core/Hostへ移植しない |

**ほかの実装進展との整合**：[#3051](https://github.com/hat47x/tei-platform/pull/3051)は**main統合済み**で、`EmptyPolicyDeny`＋`Targets=[]`＋`CoverageUnestablished`を`fails`でなく`unverified`へ修正済み（`semantic/completion.go` Blob `e42610e9159e02ba3b5c7b84c46742a846331e64`）。適格な`P=false`反例をpartialで`fails`とする既存規則は保持。[#3078](https://github.com/hat47x/tei-platform/pull/3078)/[#3081](https://github.com/hat47x/tei-platform/pull/3081)では`workspace-files.readByLoader`と空宣言warningが実装されたが、`readByLoader=true`や`validate.ok=true`は`*.tei.md`の構文解釈・Meaning受理・実Build消費を保証しない。証拠列の改善を旧失敗のままと書かず、未達のClaimだけを残す。

**作業引渡し（新Scenario正本を作らない）**：最小の新Mutant候補は「同一Sourceで重複Datasourceの宣言順だけを反転」し、選択Aliasと判定が変わるかを固定すること。次に「主対象RW／関連対象RO」のEffect閉包について、対象Relationshipと実Provider対応の有無を確認し、存在する場合のみ隔離DBで安全な拒否・実効果0・Receiptの別証拠を取る。元の`#3070`既存4件は**主対象の宣言による局所拒否**であって、上記二反例のGo/実DB成功ではない。実Owner=#1129（Relation／Cascade）、#1174（Effect/Host）、#2586（実Provider Binding）、#1180/#2514（Scenario・独立期待値）、#2621/#2823（Source）。OKF側のAttester真正性と独立Expectedは#2614/#2514/TAI #514へ委譲。特殊なWorkflow/Attester/DB Router/Policy Engineをこの計画で採用しない。

**状態**：本節はmainの実Sourceと統合済みPRから導いた**反証可能な設計／試験要求**。同namespace重複宣言・関連先ROの新Go試験、実DB強制、OKF Attesterの実Tool/Receipt、正式Owner Meaning Acceptanceは**NOT RUN／NOT ACCEPTED**。受理済み業務MeaningがRO先のCascadeを許容しているとは推測しない。

#### 三領域で破綻しやすい残余

| 比較 | 単一因子だけ変える反例 | 独立Expectedと既存Owner |
| --- | --- | --- |
| **受注：業務語の同形異義** | 同じ`approved`/数値だが「見積承認」対「受注確定」、税・丸めRuleだけ改訂 | 表層Carrier/型が等しくてもAccepted Actionの適合を自動継承しない。従来の確認は再検討候補、現在Grantとは別。#1126/#1129/#2514 |
| **組織：時点とTenantの非継承** | 同じPerson/Role/Plugin、兼務する二法人A/B。同じ`approved`でA側の所属承認だけを追加、または代理権を撤回 | Membership/Occupancy/業務責任は時点付きRelationであり、B側Grantを生成しない。Grant失効は過去の所属・Receiptを削除する理由でない。#2386/#2389/#1016/#1174 |
| **Procedure：非実行操作の存在** | 同形Step`approved`を「人による電話確認」と「自動Provider更新」に差し替える | 前者は正当なBusiness Procedureの手作業・引継ぎであり、Executable Actionが**非適用**でもモデルは不成立としない。後者にはTarget/Grant/Effect観測が必要。Workflow完了≠業務完了。#1125/#1129/#1130 |
| **Plugin：選択と作用の非干渉** | 同じDefinition/Meaningのまま未使用Pluginを同梱する、または同一Pluginを別Tenantで選択 | 実Selection/Build/RuntimeとCurrent Grantを別確認。未選択追加で既存Meaning/Providerを黙って切替えず、別Tenantの作用を推論しない。#1016/#951/#1251 |
| **失敗・回復：外観同一性** | 同じ`approved`表示でCommit ACKあり／ACK喪失、または先行Step effect後に後続Stepがblock | `completed`表示・同値Readから元Operationの成功を推論しない。同OperationIDのReceipt Lookupと対象別独立Readを優先、盲再送禁止。#1174/#1163/#2889 |

#### DSL可読性比較への最小接続

各例の**独立ExpectedとDomain情報量を同一**に固定したうえで、(a) 汎用的な短い`approve(subject)`型の**擬似構文**、(b) 対象と行為（受注確定／Membership承認／手作業記録）を認知順に明示する低ノイズの表・式・DSL候補を比較する。短さやGo AST親和性を自動採用基準にせず、**意味の誤選択、追加質問、診断位置、局所修正差分、Source fidelity、AIの入力・修正コスト**を同一Taskで測る。生成AIの解釈結果と生成AIが作ったExpectedの自己一致は合格証拠とせず、#988の構文比較と#2514の独立期待結果・既知Mutantを再利用する。正式Grammar/日本語キーワード/新Role enumは採用しない。

**判定票の最低列と停止条件**：`actual parsed Source occurrence／meaning role+accepted revision／operation applicability（manual時not-applicable）／required obligations（候補の自己申告以外）／provided+evaluated guards／selected Plugin+Provider+Target／current Grant+Tenant+resource／dispatch結果／Receipt+独立read／独立Expected／Claim ceiling`。未知は`unresolved`、非適用は理由付き`not-applicable`、実権限拒否は既存契約の`denied/blocked`、送出後不明は`unknown`として**概念上区別**する（製品共通enumではない）。実施順は、#1180/#2514へ比較Expectedを紐付ける→#2586の実Read接合で意味/Scopeと実Consumerを確認→#1174の限定PG Effect試行で実Grant/Commit/Receiptを確認。手作業に必須DB実行、read-onlyに書込Guardを一律課さない。

**既存mainの具体的接合先**：[#2926](https://github.com/hat47x/tei-platform/pull/2926)は`semantic.EvaluateCompletion`の`(TargetSet L, EmptyPolicy, Predicate P)`限定評価をmainへ統合済み、[#2671](https://github.com/hat47x/tei-platform/pull/2671)はCommit後ReadとOwner完了Claimの差を記録した。ただし#267は仮承認で、実Runtime/Consumerが対象全集合L・独立Coverage・Pを供給したことや、現Host Grant／Provider Effectを証明したわけではない。上表のM=受理済みは**独立に固定するFixture条件**であり、これらの現行実装から本番の正式受理を推測しない。新Completion Evaluatorは追加せず、#1125/#267を既存Ownerとして再利用する。

**根拠と資格**：#2619の既存反例、#2389の組織計画、#1125 Procedure、#1174の実PostgreSQL限定実証、#2889のRunner特性試験、Draft #2896のSystem Composition計画を基にした**計画上の補正候補**。この2×2／三領域／DSL比較を同一Product Headで新規実行した証拠はない。**実Tenant/Host/Provider/GUI・Owner正式意味受理、#1119 Gate判定はNOT RUN／NOT ACCEPTED**。新しい基盤の実装・既存ADRの採用は提案しない。


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
- **別Domain／AI評価**：生成AIへ供給した`Grammar/Schema/Operator＋Plugin pins/正負例/実CLI discovery`、提出Patch、Diagnostic、手動修正数、Meaning違反、Test Expectedへの変更履歴、token費用を#2018で同一Task・同一予算で比較する。構文の短さより**独立検証基準と意味保持を満たす総費用**を評価。

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

標準的なQuery DSLを高階関数として利用することと、すべての演算順序を交換してよいことは別。特に`filter`↔`take`、`validate`↔`filter`、`map(parse)`↔`filter`、`sum(map(round))`↔`round(sum)`、`groupBy`後のJOIN/再集約は、必要な不変条件の説明と独立検証基準がある範囲のみ同値候補とする（[#2387](https://github.com/hat47x/tei-platform/issues/2387)、[#2593](https://github.com/hat47x/tei-platform/issues/2593)、[#1706](https://github.com/hat47x/tei-platform/issues/1706)）。

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
| **増分reuse** | predicateやRule入力だけ変更 | Actual consumed dependency + Source/Operator/Role/Context/Provider/Target版＋検証基準の現在資格に基づき、**省略可能なStageだけ**reuse。意味結果同じでもSource Fidelity/SHO/実Grantは再確認 |

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
2. **#1706/#2593/#2497**：集合・Measure・構造写像の保存条件と反例検証基準を既存Corpusへ対応付ける。独立Expectedの適用対象を確認する。
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
4. **B3（Consumerと資格）**：同じ固定Source/Meaning候補をCUI/標準Studio/SHO/実Targetで比較し、出典、Role不明の退避、Manifest必要Member、開示、Host Authority、独立検証基準の不足を列に残す。
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

TAIのQH-01〜18は公開合成Nodeの有限試験であり、TEI実CLI／外部ネットワーク送出／SHOの保存・配送／Host Grantの実証とは別。TEIのSource/Meaning/Provider・独立検証基準はTEIの既存Ownerに残し、TAIに第二のMeaning/Authorityを設けない。現時点で新しいProduct跨ぎの実行試験は**NOT RUN**。
