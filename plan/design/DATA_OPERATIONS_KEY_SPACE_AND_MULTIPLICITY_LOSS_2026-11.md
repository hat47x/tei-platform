# MS-09：多様なData Structure間のKey空間・多重度・逆更新の喪失を明示する

> 2026-11-03／**検証用Draft・Owner未採用**。TEI製品の実装でも汎用Adapter APIの採用でもない。

## 1. 背景と独立した差分

[Data Operations #2622 の MS-01〜08](https://github.com/hat47x/tei-platform/issues/2622#issuecomment-6586192607)と[階層 HQ-01〜08](https://github.com/hat47x/tei-platform/issues/2622#issuecomment-6586106154)は、Sequence/Bag/Set/Map/Relation/JSONB/Document/Graph等を同じOperator意味へ投影する際、順序・重複・Identity・Presenceの喪失を明示するとした。

**新規の焦点**：多重度喪失だけでなく、**Keyの等価性そのものがCarrier/Target境界で変わり得る**。Sequence<record>から一意Mapへ、typed MapからJSON objectへ、SQL bagからMapへ、Graphのparallel edgesからendpoints keyed Mapへ変換すると、元にあった2要素が1要素に潰れる場合がある。後段の`all`／`count`／`lookup`／更新先選択が一見正常でも、元母集団・Identityを検査したことにはならない。

### 基礎方針

- **モデルはモデルから**：Table/Field/Key/Relationは直接宣言を正とする（#1126）。SQLはLINQ的Data Transformationの主要記述手段。Native SQLのSource byte/方言/Schemaをそのまま保持してよい（#2387）。SQL結果から業務Keyの唯一性を黙って確定しない。
- **「同じOperator」を強制しない**：`toMap`／`groupBy`／`asEntries`等は仮称。選択する際はKey型/Equality/Collation/Normalization、Duplicate policy、Ordering、Occurrence、Identity、失敗、更新対象の特定条件を明示する。異CarrierでLawが満たせない場合は`requires explicit conversion`／`unsupported`／`unknown`を返す候補とする。
- **SourceとProjectionを独立**：元Markdownの表・式・SQLシンタックスシュガーは編集・保存時に自動で標準構造へ書き換えない。内部意味構造・Node/Path/GroupのViewは必要時だけread-onlyに導出し、逆編集時は#2621/#2497がSource span、Occurrence、版、Lossと再束縛を確認する。
- **読取と更新を分離**：読み取り結果を1要素に潰す投影は、曖昧な行・Edge/Document要素へのPATCHを認可しない。型付きViewChange→Intentは#2500、実Provider効果・Receiptは#1174。
- **意味受理とは別**：公開fixture・Python JSON/SQLiteの挙動は一言語/Providerのcharacterizationであり、TEI Operatorや別Targetの同義を確定しない。

## 2. 合成した最小の反例

| ID | 元の構造・操作 | 限定的な観測 | 不正な一般化 |
| --- | --- | --- | --- |
| **MS-K01** | Sequence: `[("A",2),("A",3)]` → 一意Map | last-winsなら2→1 | 入力のAは一意だった |
| **MS-K02** | 同SequenceをgroupBy | Bag [2,3]とsum=5、Map lookup=3は異なる | groupByとtoMapは同義 |
| **MS-K03** | typed Mapの異なるKey `1` と `"1"` をJSON objectへ | Python JSONでは同名文字列Keyが直列化され、読み戻しで衝突 | JSON objectを通しても元typed MapのKeyが保存される |
| **MS-K04** | Raw JSON `{"code":1,"code":2}` をobject parse | 標準Python parseは1件、pair hookなら2 pair観測 | 読取後のMapから元Raw Sourceを完全復元できる |
| **MS-K05** | SQL bagの`order_id="A"`行2件 → Map | SQLite count 2、Map 1 | SELECT結果のrow数とMap値の数は一致 |
| **MS-K06** | 同一SKUを持つline_id 10/20への編集 | SKUだけでは更新対象が非一意 | 表示値や行indexから必ずPATCH先を確定できる |
| **MS-K07** | 値が等しい2つのMap、異なるentry挿入順 | Map等値でもentriesの列順は異なる | 全TargetでMap項目順は安定 |
| **MS-K08** | 同Node対のGraph parallel edges e1/e2 → endpoints Map | edge2件→map1件 | Node pairがEdge occurrenceの唯一ID |

試験入力は架空。JSON string keyとint key、同名JSON propertyの取扱い、Map/Graph等の順序は**Provider/言語固有の結果**として扱い、普遍的規則と誤認しない。

## 3. 受入の提案とゲート

初回の最小Headless受入では、**同じSource上のRow/Path/Edge occurrenceを失わずに表現できる変換と、情報を失う変換を区別できる**ことを確認する。Key collisionがあり、Unique Mapへの写像が指定された場合は、明示的なDuplicate policy/aggregationを要求するか、拒否する。黙ってlast-winsをTEIの普遍意味にしない。別Provider実現は、以下を**個別に**照合する。

1. **Shape/Carrier**：Sequence/Bag/Map/Graph／SQL rows／JSON objectの入力型と出力型、Key空間、Equality/Collation、Canonicalizationの有無。
2. **Semantic/Failure**：Occurrence数、重複要素、集合のComplete、順序、Group値と独立Expected、loss/errorの観測。等しい出力値だけで同義PASSにしない。
3. **Read/Disclosure**：実Provider取得範囲、非出力Key/Predicateを使用したか、読者/AIへ何を開示したか（#2506/#2516）。
4. **Edit**：View行/Map Keyから元Sourceの一意の行・edge・document elementへの戻りが保証されているか。なければread-onlyまたはChoice/Conflictとして扱う（#2500/#2597）。
5. **Authoring/Target**：元のMarkdown/SQL表層を保持し、Sourceと意味Projectionを別にPin。実TEI Operator/Providerごとの実行・独立Oracle、Owner受理は別資格（#988/#2387/#2621）。

### 意図した試験の順序

- P0：本Draftの標準Python＋SQLite **8件の独立した合成Characterization**を再実行。スクリプト：[`probe_multishape_key_loss.py`](../fixtures/data-operations/probe_multishape_key_loss.py)。
- P1：#988/#2622の**実在する型付きReference**のうち利用可能な狭いOperatorへK01/K02/K05を接続。実装の有無と実Input/Output/Failureを記録。見当たらなければNOT RUN/unsupported。
- P2：PostgreSQL JSONB/Document/Graph Providerでの実差異を、各版・Parser・入力Schema・比較規則・実Read証人を固定してK03/K04/K08へ拡大。SQLGlot Worker→Go DerivedIRの実V0/V1はQX-33の停止条件に従う。
- P3：Studio/SHO/別Targetに、情報喪失の箇所・逆編集禁止理由・元Authoring Surfaceを提示し、実Consumerで確認。Providerが使えない場合、合成referenceの結果を実証済み製品品質としない。

## 4. 現在の観測・未決

ローカルPython 3標準`unittest`で **8/8 PASS**（このDraftのscriptと同一論点、SQLiteはPython標準sqlite3のin-memory実行）。結果は公開合成の固定入力に対する狭いcharacterization。**元のSource→TEI DSL Parser→Data Context→実TEI Operator→DB/Doc/Graph→SHO/Host の一試行ではない。** これを法則証明/独立sealed Oracle/所有者受理に昇格させない。実TEI Go/Java/.NET/WASM/実PostgreSQL/MongoDB/Graph/Studio/SHO、実書戻し・実AI開示はNOT RUN。

**Owner未決**：Key Equality/Normalizationの標準表現、Duplicate policyの選択肢、lossy projectionの診断レベル、read-only/UI表示、実Targetの対象順序。正式Syntax/API/Core型/第二Query Engine/独自Host/SSOTは提案しない。担当は#2622（Data Operations）、#2495（構造Facet）、#988（Operator Law）、#2387（SQL由来の解析）、#2500（ViewChange）、#2621（元Source形式保持）。
