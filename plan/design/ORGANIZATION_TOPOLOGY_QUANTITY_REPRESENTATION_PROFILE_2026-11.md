# 組織ドメインの表現力：トポロジー・量／単位・標準表現型の合成Profile（2026-11-07）

**状態：採用前の横断設計と独立Expected。** 既存Organization Source・汎用Data Facet・Representation/Function・Measure契約の組合せで十分な表現力を得られるかを反証する。新Core型、全組織共通Ontology/権限正本、第二Measure/Query/Renderer Engine、独立Test SSOT、TEI独自Hostの採用ではない。新たな実Go/DB/GUI/SHO/Owner試験は **NOT RUN**。

**一次資料の位置づけ**： [W3C ORG](https://www.w3.org/TR/vocab-org/) は組織階層・Role・Post・Membership・履歴の再利用可能な語彙を持つが、すべての説明責任・権能を完結させない。[QUDT 2026](https://qudt.org/doc/2026/02/DOC_SCHEMA-QUDT.html) はQuantity Kind／Unit／Dimensionの違いをモデル化する。[W3C RDF Data Cube](https://www.w3.org/TR/vocab-data-cube/) はDimension／Measure／Attributeを分ける。TEIはこれらの内部スキーマをコピーせず、必要な区別と反例を既存Facetの**適合試験**へ取り込む。BIPMの[SI Brochure](https://www.bipm.org/en/publications/si-brochure/)は物理量に関する参考であり、「人」「FTE」「案件」や組織上の業務意味をSI単位へ無理に還元しない。

## 1. 継承すべきものは「グラフ型」ではなく、構造・数量・意味・用途の合成能力

同一のOrgUnit D1が、指揮階層のノード、費用帰属先、法的事業所、案件の取扱担当、プロジェクト共同体、集計のDimension Memberとして現れる。これらは同じIDを参照し得るが、**異なるRelation Role／時点／責務の別グラフ**を形成する。基本の意味Carrierを一つの巨大な`Organization`オブジェクト階層へ押し込めない。

- **構造Facet**：Node/Edge/多項Relation、直接参照とIDを持つ実体化関係、方向・参加者Role・多重度、Pathと重複、閉路、集合／Bag、順序、接続性・到達可能性、Graph/Tree/DAG/Network。単なる関係Graphと数学上の位相空間（開集合族を持つTopology）を同一視しない（#2495/#2544）。
- **時間・空間Facet**：validAt／knownAt／decisionAt、構造版・公開版・観測時点、予定/Actual、位置Site／座標系／区域。所在地の包含・地理的隣接とOrgの上下関係、歴史組織のas-ofと現在Host Grantを混同しない。
- **Quantity/Measure Facet**：Quantity Kind／Unit／基準量・期間／精度・換算・丸め、Fact identity、集計の粒度と許された演算、集計軸別の加法性、必要な独立対象集合・属性値の完全性。数字／単位文字列／列名の一致で自動換算しない（#1705/#2593/#2544）。
- **Normative/Business Facet**：所属、雇用、役職、指揮、責任、予算帰属、委任・規範とその成立条件。関係・役職の事実から現在Principal・Grantを作らず、業務DFDの授受矢印からWorkflow Instanceを自動生成しない（#2386/#1125/#1174）。
- **Representation/Interaction Facet**：Function所属の表現インスタンスと型付きData Context、意味を保つ／失うProjection、Consumer別Scope/Disclosure、読み取りと逆編集の適格性。SHOは派生文書Consumerであり、TEI Function Instanceと無条件に同じSession／Grantにしない（#3236/#2497/#2506/#2507）。

**「継承」の実用的な意味**：一般標準の複数Facetを型付き参照で**合成・制約強化・Profile適用**する。OO継承・固定`OrganizationTree` Core kind・すべてのProfileへの必須Facet列挙を意味しない。必要なときだけOrgUnit/LegalEntity/Person/Membership/Position/Occupancy/CaseRecord等のドメイン意味Roleを追加する。特定業種/外部製品の既定値はAdapter/Profileに留める。

## 2. 組織が持ち得るトポロジーは一つではない

| 構造層／用途 | 期待される構造と制約候補 | 一つのTreeへ還元したとき失われるもの |
| --- | --- | --- |
| 法人・企業集団 | 法人格の同一性、持分・支配・共同契約などの型付き多項／多重Relation | 組織に含まれることと、持分比率・法的責任・Tenantを混同する |
| 人事/指揮 | 目的別階層、単一親Treeが必要なProfileと、複数Reporting辺が有効なDAG/Profile | 複数の上司、発令時点、同種の二経路と単一役職の違い |
| 配属・在任 | Person×OrgUnit×Role×期間の実体化Membership、Person×PositionのOccupancy、複数Employment | Person=Position、Membershipの1行=1人、空席の不存在 |
| 財務・予算・費用帰属 | 部門・費目・Project・会計期間をまたぐAllocation/Cost Center、経路別の係数と加法性 | 経路を削ることによる配賦損失、同じ費用の無条件重複加算 |
| 責任・承認・代理 | 業務Roleと行為の対象関係、任務／職務分掌・委任の方向、例外と失効時点 | 下位組織へ到達できることを行為権限とする |
| 案件・手順 | CaseのhandlingUnitという直接Relation、Procedure Step/Handoff/Workflow Instanceの別ID | Case移管=部門後継、業務手順=物理更新という誤同一化 |
| 場所・管轄 | Site、拠点のGeo、管轄区域、移動・在宅・複数勤務地と期間 | 地理的包含=指揮系統、拠点=唯一OrgUnit |
| 仮想・計画組織 | 提案Graph/Actual Graph、候補Revision／公開Revision、複数Scenarioの独立世界 | Preview案を確定組織や旧レコードの実更新へ昇格 |

**組織トポロジーの検査単位（正式Schemaではなく既存参照に渡す項目）**：`subject identity / relation role & participants / n-ary or direct / axis & purpose / direction / multiplicity / allowed cycles / path vs entity identity / valid & known time / tenant & source provenance / cardinality and closure`。

同じH→D1→CとH→D2→Cでは、**CというNodeは一つ、CへのPathは二つ**。Treeの経路複製表示は正当なUI手法にもなり得るが、複製された行を別OrgUnitや別の費用Factにしない。権限のScope/Relation到達も、後継組織の業務責任の自動継承も、Path存在だけでは決まらない。

## 3. 単位系は数値型より多層：Quantity Kind、Measure、集計法則

現在mainの`reference/org-headless-workspace/semantic/organization/model.tei`には`Membership.allocation type std.decimal`がある。しかし**0.6/60/60%のどれを採るか、週40時間に対する比率か、契約上の配賦か、Actual工数かはこのSourceだけでは確定していない**。これを本番FTEと自動認定しない。

| 組織のMeasure例 | 種別・単位・基準 | 合法な比較・集計に必要な条件 |
| --- | --- | --- |
| **Headcount** | 相異なるPersonの数［人］。Person occurrenceのdistinctが必要 | 所属Role、対象集合、時点、複数Orgをまたぐ重複排除の範囲。部署別人数の単純合計は全体人数とは限らない |
| **Membership/Assignment Count** | 所属発生件数［件］。人とは異なる | Materialized Relation IDが数えられているか、同じ関係の期間分割・重複Pathはどう扱うか |
| **FTE** | フルタイム相当量［FTE］。比率は表面上無次元でも業務上はPeople Countと別Quantity Kind | Fractionの分母となる基準労働時間、対象期間、Contract/Actualの別、跨Org配賦係数、精度・丸め、比較可能性 |
| **Work Hours / Capacity** | 稼働・要員容量［時間/期間］、定員［枠・FTE等］ | 工数とFTEの変換基準、予定/実績、休暇・残業、期間積分。容量超過を人数だけから計算しない |
| **Cost / Budget** | 金額［JPY等/対象期間/シナリオ］。レートは金額/時間や金額/FTE等 | Cost Center/期間/通貨/為替基準日/配賦、残高対フロー。JPYとUSDをそのまま加算しない |
| **Share / Allocation Rate** | 持分・費用配賦・労働比率・承認クォーラムは**異なるRatio Kind** | 0.4、40%、40を単位宣言なしで同一視しない。経路別係数の合計1.0はOwner規程・対象集合完全性のもとでだけ保証 |
| **Span / Layers / Coverage** | 管理範囲［直接配下のPosition/Person/OrgUnit数］、深さ［辺数］、被覆率［対象/母集団］ | どのGraph axisか、占有職位/空席、DAGの複数Path、集計する母集団・Source閉包・有効時点 |

**単位系は三段階で扱う**：(1) `QuantityKind`が何を量っているか、(2) `Unit/Basis/Period`が尺度と換算を規定し、(3) `Measure × Rollup Axis × Operation`が加法性／重複・重み付け／再集計可否を決める（#1705/#2593）。FTEと費用配賦率はともにDecimal/Ratioでも、互いの値を直接足す・入替えることは原則不可。期間末のHeadcount Snapshotを12月分足して「年間人数」とすることも不可。平均を再集計するには分子・分母や重み等の十分統計量が必要になる。

### 小さな独立Expected（閉じた合成World限定）

試験として一つの同時点・同Source/Scopeの完全な集合を仮定する：P1はD1にA1=0.6 FTE、D2にA2=0.4 FTE、P2はD1にA3=1.0 FTE。ここで**合成Fixtureは「値がFTE fraction、全3 Membershipを漏れなく取得」と明示**する。

- D1：Unique Person=2人、Membership=2件、FTE=1.6。D2：Unique Person=1人、Membership=1件、FTE=0.4。
- D1∪D2：Unique Person=**2人**（各部門Headcountの単純合計3ではない）、Membership=**3件**、FTE=**2.0**。HeadcountとFTEは同じ数値2でも同種の量ではない。
- 同じデータをPeople Table／Matrix／FTE Chartへ出す際、**3行・2人・2.0 FTEを1つの`count`へ潰さない**。別Tenant/Partial/未読込のA2があれば、完全な全社Claimは`unverified`。P1が0.6+0.4=1.0であることは、この合成Owner Rule下の局所説明であり全企業の労働配分規則にはならない。
- 別GraphでH→D1→CとH→D2→Cがあり、C固有の0.4 FTE Factを両Pathが参照しても、HでのUnique Entity集計は**0.4**。0.8は二重計上。配賦係数をOwnerが明示的に`0.5,0.5`と受理したとき、経路別の0.2+0.2=0.4を比較可能とする。Pathの重複を一律DISTINCTして係数を失うことも不合格。
- `allocation`が60、null、未取得、観測済み0、またはFTE以外の費用配賦率である負例では、前述のFTE結果へ自動換算・自動代入しない。Ruleに必要なFieldと対象集合を同じ適格Cohortで実ReadしていないClaimは`unverified`。

この独立Expectedは#2593のMC-01/03/05/07、#2510のORG Corpus、既存§16/39/53に**差分として引き渡す**。新たな本番FTE計算エンジン・集計正本や新Scenario IDを作らない。

## 4. 標準表現型の能力・適合・損失と組織向けProfile

`Data topology → Function Data Context/Result Slice → Function-owned Presentation Instance → Consumer output` の各写像で、**表現可能（representable）・用途に適切（fit）・意味を保存（preserving）・逆編集できる（editable）を独立に判定**する。「描けるが、編集には使えない」ことを正常な結果として許す。対応資格は`supported / conditional-with-evidence / lossy-read-only / unsupported / unresolved`等の**説明用の分類**であり、新Core Enumの提案ではない。

| Representation | 適切な組織上の利用・必要条件 | Loss／編集停止条件 |
| --- | --- | --- |
| **Tree / Org Chart** | 目的/時点を選択した単一親Hierarchy、またはDAGをPath重複として明示するread-only Tree | 多重Parent/異軸の辺を隠して唯一親と宣言しない。Path複製から無根拠なOrg移動を生成しない |
| **Graph / Network / Hypergraph View** | 複数Relation Role/方向/Occurrence、多辺、必要時のn-ary関係の中間表示 | 辺の外観・ラベルやNode配置だけではMembership等のRelation ID・業務意味・Targetが確定しない |
| **Matrix / Pivot / Heatmap** | Person×OrgUnit×Position×期間、MeasureとFact/Cellのgrainを明示 | 0・非開示・空・未取得のセルを同値化しない。多次元を潰したCell値から元の複数行へ自動PATCHしない |
| **Table / Detail / Form** | Membership/Position/Occupancyの独立ID、Field presence、期間、Root/Scope、read-only/編集対象の区別 | QueryのJOIN重複行を別の実体行とみなさない。値の不正な単位変換は表示上も警告 |
| **Chart / KPI / Sankey** | Headcount/FTE/Cost等のQuantity Kind・Unit・Rollup lawを表示。Sankeyは明示したFlow/保存則があるときのみ | 費用帰属や指揮線を根拠なく「流量」と表し、幅の保存則を捏造しない。期間末Snapshot/金額の直接積算を不許可 |
| **Timeline / Gantt / Scenario Diff** | valid/known/decision/published/actualの別、Versionと変更前後のRelationship | 計画上の部門新設や辞令候補を実採用済み・外部DB更新済みとしない |
| **Map / Spatial** | Site/拠点・管轄Geometry、座標系と時点を明示した空間の表現 | 地理的に包含/隣接するだけで指揮階層、企業内所属、行政上の管轄を作らない |
| **SHO／Mermaid風テキスト・図解** | 人が読める概要を生成し、Source/Role/Path/Quantity/Time/Disclosureの必要な来歴を残す | 多重Edgeが1本に簡約された図やMarkdownを正本へ逆編集しない。SHOからTEIモデルのOwner意味受理を自動付与しない |

**Mermaid風の糖衣例（説明用、正式TEI DSLではない）**：

```mermaid
flowchart LR
    H["統括 H"] --> D1["事業部 D1"]
    H --> D2["事業部 D2"]
    D1 --> C["共通センター C"]
    D2 --> C
```

同じCを2経路で表しても、**Graphに現れたNode Cは1個、Pathは2本**。テキストがDiagramを生成できることと、Relation purpose/occurrence・有効期間・係数・非表示Edge・現在Read Scope・編集Targetの全情報を保持したことは別。元Dataへの参照と適格な補助メタデータが欠ける場合はread-only説明に留める。Mermaid/Graphviz/Markdown等の文面は標準Dataの**表現上の糖衣**に留め、Source同一性や必要な編集逆写像は#2497/#2621・SHO Composer Owner側へ委譲する。

## 5. 生成AIと人間が迷わず選択するための最小判断票

AI/CUI/Studioには、新しい「組織専用Widget一覧」の網羅を要求する代わりに、既存Facet/Function/Representationの候補から**目的に対する最小適合理由**を提示させる。

1. **何を答えたいか**：単なる所属の説明／正確な承認候補／Headcount／費用配賦／履歴比較／編集Intent。Claimと業務目的を特定。
2. **どのStructure/Quantityか**：Node/Relation occurrence、軸、方向、時間、Measure kind、単位・分母・Rollup law、必要な必須/非表示依存を列挙。未知は補完せず診断。
3. **どこまで読んだか**：元Source/World、Query/Providerの実Read、対象集合とFieldの完全性、Tenant/Scope、必要なRead cohort・Consumer disclosureを確認。
4. **表現の条件を照合**：複数親ならGraphまたはloss明示付きTree、Unique Person数とFTEなら別Measure、期間あるならTimelineなど。図解の視認性と意味保存の優先を混同しない。
5. **操作の逆写像**：対象Relation/Case/Row/版が一意か、粒度は保持されたか、現在Host Grantと実Effectの境界があるか。満たさなければread-only／明示的な再選択・追加Read。
6. **説明・学習に利用**：同じSourceからTable/Graph/KPI/SHOへ出したとき、何が一致し、何が損失・非開示・未確認かを理由付きで示す。標準表現型候補はLLMが生成できても、その適合証拠をAI自身の言明で代替しない。

これは#2507のFunction Pattern選定と#2497のProjection説明へのread-only引渡しであり、AIによる画面選択を即座に本番実行資格へ変える仕組みではない。

## 6. 検収順序と責務：新機構なしに最大の表現域を反証する

| 段階 | 最小の独立検証 | Owner／不合格条件 |
| --- | --- | --- |
| **P0 構造・数量の合成対照** | 同じPersons/Membershipsから、Tree/Graph/Matrix/TableとHeadcount/FTEが**異なるが正しい**結果を返す。多重Path、n-ary、NULL、既知0、partialを一因子ずつ変える | #2510/#2495/#2593/#2514。構造・単位の無宣言な補完、全件判定の自己申告が不合格 |
| **P0 Native Source照合** | main #2681の実`model.tei`に存在するRelation/Attributeと、Profileで**まだ宣言されていないAllocation basis/Unit/Policy/Geo**を分類する | #1126/#2386。合成Fixtureを実Source capabilityへすり替えるのは不合格 |
| **P1 二つのFunction／二つの表現** | Org ExploreのGraph/TreeとCase HandlingのTable/Pickerが、必要なSubject/Path/Relation粒度、Source/Scope/epochとLossを保持する。外部SHOには別Disclosure | #2506/#2507/#2497/#2516/#3236。図の見かけの一致から編集Targetを導出するのは不合格 |
| **P1 実Readの縦切り** | C1→D1の一件を実Function→Query選択→Provider Read→Consumerへ結び、必要なField presence/rowset/時点を独立観測。追加Count/FTE Claimには依存閉包を要求 | #2586/#2622/#2628/#2611。合成や別FixtureでのPG PASSをOrg Product Readと呼ぶのは不合格 |
| **P2 編集・発火・更新後観測** | 元のMembership/Case Relationへ一意に戻せるIntentだけを送出候補とし、Owner業務Rule、現在Host Grant、Receipt、独立Resource Readを別資格とする | #2500/#2598/#1129/#1174/#1163。Tree移動やChart数値編集から自動Effectは禁止 |
| **P2 異業種への耐性** | 企業・行政・非営利・研究組織・共同事業で同じFacetを使い、特有RuleのみProfile差分にする。意味型/演算の不足を反証する | #2510/#2493/#2018。小PoCの範囲を製品の表現力上限にしない |

**採用停止線**：現mainのNative Org E1は実在するが、組織用FTEのQuantity Kind/基準・多軸Hierarchy・新Representation Instance/SHO出力・実Case Provider E2/Host Actionがすべて実装済みという主張はできない。正式な組織集計式、重複Pathの会計扱い、承認・委任、数値の演算則はOwnerの意味受理が別に必要。P0のC1→D1実Provider Readを妨げる大規模Core改造は行わない。

## 7. 三要素図面と組織Measureのクロスリファレンス：同一Meaningから異なる線・数値・操作を導く

他チャットのX-TEA三要素分析法（Data／Function／Business各モデルの図面表示・編集、CRUD等のクロスリファレンス生成）とSHOの平易なテキスト図解要求を、**既存#1124三視点Modeler／#1125一階層業務DFD／#1803設計書導出の受入に接続**する。§1〜6の「Dataの構造・数量・表現型」の上に、どの**Function/Procedureが、どのData/Relation occurrenceを、何の目的で利用するか**という型付き依存関係を投影する。ModelerのData/Function/Business図、CRUD表、SHOの組織図は別正本ではなく、Script/DSLを含む同一のauthoritative Sourceから導く。

| クロスビュー対象 | 必要な型付き関係／粒度 | 生成図・表 | 誤った自動同一視 |
| --- | --- | --- | --- |
| Data View | Person／OrgUnit／Membership／Position／Occupancy／CaseRecordとRole別Relation、時点・Source ID | ER/Graph／関連詳細／組織構造Diagram | 同じGraph線を雇用・所属・指揮・費用配賦・Case担当と判定する |
| Function View | Function ID、Data Context、Query/Operation Role、依存するRead Set、Case/Relation変更Intent、Quantity種別 | Function→Data/Operationの参照図、入力出力表、CRUD候補 | Tree/Pickerが表示できることを物理CRUD実行・現行Grantと判定する |
| Business View | Business Procedure／一階層DFDのBusiness Step/Actor/Trigger、対象Dataの授受、手作業/外部Step、責任 | 手順DFD、Handoff説明、職務別担当表 | DFD矢印をData Relation、Transaction、Workflow実行順や引継ぎ完了と判定する |
| Cross Reference | 同じCanonical Subject／Role ID、Source locator/revision、Function→Data用途、Procedure→Function/Manualの明示Binding | CRUD/Role/Step表、Source追跡、差分・影響表 | SQL JOIN/SQL UPDATE字面からOwner受理済みの業務CRUDと推定する |
| Quantitative Projection | Measure Ref、Fact/Occurrence、Unit/Basis、Rollup Axis、演算法則と必要集合のRead Witness | 人数・FTE・工数のMatrix/Chart/KPI、SHO図解 | 同じ「2」でUnique Person数、Relation件数、FTEを同一Quantityとする |

### 7.1 CRUD表は「宣言・導出・実測」を混ぜない

**設計時の合成例**（既存mainに実Functionや実書込が存在すると主張しない）。`R`は「明示的に設計上Readを必要とすること」、`U?`は「明示意味の採否・実装・Grant未確認の更新候補」、`—`は「この例から宣言なし」。`U?`をC/U/Dの実実行実績として扱わない。Relationの`Link/Unlink`はRow CRUDの`U`と等価とは限らない。

| Function／Business step | OrgUnit | Membership | Position/Occupancy | CaseRecord | Measure／依存 |
| --- | --- | --- | --- | --- | --- |
| Org Explore（Function候補） | R | R | R（表示目的が明示された場合） | — | PathとAs-of、Entity/Occurrenceを区別 |
| Case Handling（Function候補） | R | — | — | R、担当`U?` | Case→Orgの直接Relation、実TargetとCurrent Grantが必要 |
| Workforce Report（Function候補） | R | R | 必要なReportのみR | — | Unique Person CountとFTEを別に算定、非出力allocationを含む |
| 引継ぎを受領する（Business Procedure） | 業務参照 | — | — | 業務参照 | 人手/外部確認をSystem Uと自動対応させない |

行と列の交差は`Declared / Derived Candidate / Actual Read / Actual Effect / Not observed / Unknown`等の**別資格**を参照できなければならない（正式なCRUDセルSchema新設を意味しない）。設計書Generator #1803は、未取得／未宣言セルを「C/R/U/Dなし、確実に使わない」と断定しない。Queryに非出力のPredicateやJOIN依存がある場合、出力Fieldに現れなくてもRead Setへ残す。Business Actor・Procedure役割と物理Principal/Grantは同じ列へ潰さない。

### 7.2 数量付き図の一貫性は射影と集計の交換可能性を要求する

一般には「GraphをTree展開した後にSUM」と「元の実Fact集合にSUM」の結果が一致するとは限らない。**投影してから集計することと、集計してから投影することが等価なのは、Subject/Fact identity・path係数・集計軸/基準時点・Read集合完全性・NULL規則が保存される場合に限る。** 同じ数値が描画されるだけでは十分でない。

§3のH→D1→CとH→D2→Cにおいて、Cに固有の0.4 FTE Factが一件なら、Pathを2行へ複製した後に単純SUMする0.8は、Ownerが「経路ごとに全量計上する」と受理した特別な場合を除いて元Factの0.4と同値でない。逆に経路別の明示重み付けが業務上の意味なら、Nodeの単純重複排除は経路係数を失う。SHOのDiagramでEdgeを一本に畳む表示は許せても、失われたRole/Path/係数の再生成・逆編集は不許可。

これを組織の標準構造が持つべき**数量操作の適合性**として#2593/#2544へ渡す。新たな数式解釈器やGeneral Graph Engineを組織側で作らない。

### 7.3 図上編集の資格とCRUDの影響解析

同じMeaningに対応する複数の投影を保持する場合でも、図形移動や数値変更を直接Source改変にしない。

1. **D1への構造Edge付替え**：Relationship occurrence/Role、対象Graph axis、有効・認知時点、Source span/revisionを一意に逆引きできるならtyped changeの**候補**。多重ParentのTree展開行／Projection-only配置は編集不資格。#1124の既存Preview→Acceptance→reparse/reproject経路へ渡す。
2. **MatrixのD1 FTE=1.6セル変更**：複数Membershipが寄与した集約であるため、対象Fact/演算の逆写像が一意でない。独立した配置配分のOwner操作が宣言されない限り、元の0.6/1.0への自動PATCHは拒否しSource/Contributorへnavigateする。
3. **Business DFDの「D1→D2引継ぎ」線編集**：Procedure/授受契約の変更候補であり、CaseRecord.handlingUnit D1→D2の実Writeとは別。手順図の変更だけではWorkflow Instanceを実行しない。
4. **CRUD表の`U?`セル編集**：派生された利用状況の表そのものを正本とせず、元のFunction Operation／Rule BindingのSourceへnavigateして変更案を示す。意味受理→生成／実効果を別に表示する。

### 7.4 独立Expected／変異の最小集合（#2514/#1180へ引渡し）

| 対照（公開合成、原実装由来のExpectedではない） | 変えた因子 | 不変でよいもの／更新・停止すべきもの |
| --- | --- | --- |
| **ORG-XV-01 組織Graphの目的変更** | H→Cの同じ端点を`command`→`budget` Relationへ変更 | Node IDは同じでも指揮Tree/Business権能候補は失効。費用配賦図は別Purposeとして再評価。軸を無視したRendererは不合格 |
| **ORG-XV-02 FTEだけ変更** | A1.allocation 0.6→0.5（基準／時点／集合は同一と仮定） | Unique Person／Membership件数は不変、D1 FTE=1.5・全体FTE=1.9へ改訂。Graphの所属線に無関係な再描画を安全違反とはしないが、旧FTE表示/SHO数値維持は不合格 |
| **ORG-XV-03 Path fanout** | Cへの二PathをTree出現行として展開 | CのEntityは一つ／Path二つ、未配賦Fact0.4を0.8にしない。二行を別組織とするCRUD/Chartは禁止 |
| **ORG-XV-04 DFD矢印編集** | 人手Handoffの送り先D1→D2を変更 | Business Procedure/責任の候補差分だけ。CaseのhandlingUnit/Host Grant/Provider Effectは不変（別操作がない前提） |
| **ORG-XV-05 非可逆集計セル編集** | D1 FTE Chartの1.6を1.4へドラッグ | 元A1/A3への一意な更新候補なし→read-only/ambiguous。独立Actionが新設されていない限り自動更新しない |
| **ORG-XV-06 表示一致・内部差異** | 同じPublic Treeを保ち非公開Membership/委任・時点/Sourceのみ変える | 外部SHO/AI/Tool Logに非許可情報・件数が漏れない。Private依存の承認候補は再資格化 |
| **ORG-XV-07 Source版ABA／後着** | A→B→AのSource/Selection復帰と旧Response到着 | 出力一致だけで旧Owner Meaning・Receipt・現在Grant・編集Offerを復権させない |

上記は**既存§16/39/84/85、#2593 MC、#3236 CX、#1124 M0〜M3、#1125 DFD、#2514独立検証**へ交差参照するための診断対照で、第二Fixture SSOTではない。最初のP0は構造・Measure・図/表のread-only反証、P1は#2586の実一件Read→Function Contextからの三視点/Quantity導出、P2は一意なモデル変更候補と受理後再Projection。Modeler/SHO/外部Hostでの実一往復とCRUD実更新、Owner正式受理は**NOT RUN**。

## 8. モデルから導く組織検証義務とC0/C1・業務Coverage（2026-11-07、他チャットのAI-firstテスト設計との接合）

### 8.1 実装を自己採点させない導出順序

他チャットの[モデル由来テスト生成Draft #3258](https://github.com/hat47x/tei-platform/pull/3258)・[#3259](https://github.com/hat47x/tei-platform/pull/3259)・[#3260](https://github.com/hat47x/tei-platform/pull/3260)・[#3261](https://github.com/hat47x/tei-platform/pull/3261)は、三要素の制約からDataset/Scenario/独立期待値候補を作る**未統合・互いに重複し得る設計案**である。組織側は第二のAI Generator・Driver Scenario・独立Expected Storeを採用せず、**検証義務の判別対象と合成World**だけを提供する。

導出手順は、(a) **実際に取得・受理された**Data/Function/Businessの意味制約とSource/Role/時点を固定、(b) Required Obligationと必要入力・対象集合・未取得依存を列挙、(c) 独立した固定Expected（本節の手書き有限値・Ownerに受理されたRule等）と既知Mutantを対応づけ、(d) #1131が型付きDataset/Snapshotの**候補**を生成、(e) #1180の保存Driver Scenarioが受理されてから各TargetのCLI/Web等を導出、(f) #2514の独立検証基準でMutant検出力と適用範囲を測り、(g) #1163のexact-HEAD/Attempt/Provider/Consumer証拠へ接続する順とする。

**Important**：モデル由来の説明を読んだLLMが、Candidate/Implementation/Expectedの三者を同じ推論で作った場合は独立検証としない。未受理の「強属性」・FTE換算・組織の権限継承規程を生成時に補完しない。Syntactic `std.decimal` から業務FTEの正例・負例を生成するためには、別途受理されたFixture契約が必要になる。

### 8.2 独立Expectedと故意の誤実装を最小に揃える

§3/§7で定義した閉じた合成World W0（P1→D1=0.6 FTE、P1→D2=0.4 FTE、P2→D1=1.0 FTE、同時点・同Scope・全3 Membership実体が揃う）に基づき、最初の**固定Expected**は全体Unique Person=2人、Membership=3件、FTE=2.0、D1 FTE=1.6とする。これらは本番TEI意味やProviderのActualではなく、**テスト独立期待値の候補**である。

| 義務（#1131/#2514への引渡し識別用） | 一因子変更・正例と固定Expected | 落とすべき既知Mutant／停止線 |
| --- | --- | --- |
| **ORG-TG-01｜対象同一性** | W0の全体Unique Person=2、Relation件数=3。Person P1は2つのMembershipを持つ | `COUNT(Membership)`をHeadcountとして3と出す／JOINの同じPersonを別ID扱いする |
| **ORG-TG-02｜FTEと数量型** | W0のFTE=2.0。A1だけ0.6→0.5なら独立Expectedは全体FTE=1.9、D1=1.5、Unique Person=2、Membership=3 | 同じ行数で旧FTE=2.0をキャッシュ継続／FTEとHeadcountを同一Quantityとして比較／60を0.6へ無宣言換算 |
| **ORG-TG-03｜二経路と配賦係数** | 同じCへ2つのPath、C固有の一Fact=0.4。Entityに相対した集計=0.4。OwnerがPath重み0.5/0.5を明示した場合のみ0.2+0.2=0.4 | Graph展開行数×0.4=0.8を無条件に合格／`DISTINCT`で正当な経路係数まで消去 |
| **ORG-TG-04｜用途別Relation** | H→Cの端点とIDを保ち、Axisだけ`command`→`budget`。Command到達の依存Claimは再資格化、同一MembershipからのHC/FTEは別条件が変わらない限り維持可能 | 軸を無視したTreeを同じ指揮図として通す／全Facetの不必要な失効／Axisの変更から新Grantを発行 |
| **ORG-TG-05｜不完全な取得** | Required IDs={A1,A2,A3}、Observed={A1,A1,A3}／またはTOP1のみ。A2の完備を独立に証明できないので全体FTE/全件所属は`unverified`。A1の存在という局所Claimは維持し得る | 件数3一致で`CoverageComplete`自己申告／不足Allocationを0へ置換／Partialの局所正例も一律偽にする |
| **ORG-TG-06｜非単射の図と編集** | Cに至る2経路をSHOの1行へ簡約／D1 FTE1.6のChart Cellを1.4へ編集。元Node IDを保持したread-only説明は可能、直接更新先は`ambiguous` | 1行表示から一意の関係Edge／A1・A3の更新値を創作し、画面編集を直接PATCH |
| **ORG-TG-07｜業務DFDと実効果** | D1→D2の引継ぎProcedureの矢印だけ変更、CaseRecord C1の実DB/Hostに操作なし。業務Designの差分のみ | DFDの見た目変更をCase担当移管Commit／人手受領の完了とする |
| **ORG-TG-08｜現在資格と公開範囲** | Public図は同一でPrivate Membership/Delegationだけ異なる2 World。許可されたPublic Consumerに差がない範囲だけ表示可能。Grant/Policy版・Source/epoch ABAではAction Eligibilityを再検査 | 非公開関係の個数がTooltip・SHO・diagnostics/Tool Logへ漏れる／旧Action Offerを現行Grantとして再利用 |

**空虚なVerifierを落とす正例**：A1だけFTE変更したとき、旧FTEを維持する実装と、何も出力しない実装の双方を不合格とする。Axisがbudgetへ切り替わった場合は、新たなbudget表示が必要ならそのPurposeで出力が変わる正例を別途固定する。「一律unverifiedなら安全」という実装も合格にしない。変異実装のExpected自体をCandidate Generatorへ再生成させない。

### 8.3 Coverageは三種類の分母と一つの証拠列を分ける

- **C0／C1（コード側）**：**C0=実行されたコードの文/命令、C1=実行された分岐の真偽（分岐Coverage）**を、具体的な生成Go/Target／Plugin／Runnerのexact Artifact/Versionごとに測る。組織のMeaning SourceからC0/C1達成を推定しない。異TargetのC1はInstrumentation/分母が違うため総計を無条件に合算しない。
- **意味・要件Coverage（モデル側）**：受理済みOrg Relation/Role／Measure/単位・Rollup／Function Operation／Business Procedure・Handoff/現在Grantの各Obligationが分母。関連Data/Function/Businessの同じMeaning IDを参照し、未受理・unsupported・未読込を分母から無言で消さない。
- **判別力（独立検証側）**：独立Expectedに対する正例・一因子負例の両方、既知Mutantの`killed/survived/not run`を記録。UI表示テスト件数・Go PASS件数だけで「この組織ドメインをカバーした」と宣言しない。

**検証状態を同じ一枚にする列（新Core schemaではない）**：`Obligation / Domain Source+accepted meaning ref / required population & fields / generated dataset & seed / independent Expected ref / known mutant / required consumer + branch / located & loaded / actual executed target+HEAD+Attempt / actual Provider/Disclosure/Receipt if relevant / C0-C1 instrumentation / observation / qualified or unresolved`。分母不明ならCoverage率を100%と算出しない。必要なのに存在しないRule・未出力Predicate・手作業Step・実Provider Readerは`not run`や`unsupported`のまま報告する。生成したテストがすべてSKIPでもCoverageを加算しない。

### 8.4 最短実行順：構造→数量→三視点→Product E2

1. **P0-a／モデル探索**：main統合済み#2681 Native `model.tei`を実CLI/Sourceから取得し、存在するPerson/OrgUnit/Membership/Position/Occupancy/CaseRecordと、**存在しない**FTE QuantityKind・Allocation分母・正式Org Parent Graph/Procedure Bindingを観測資格つきで区別する。合成W0のFTE契約は本番Nativeへ暗黙移入しない。
2. **P0-b／独立Expected**：ORG-TG-01/02/03/05の固定正常値と故意の誤結果を、#1131/#2514の既存Fixture/Expectedへ渡し、#2811のAI-first候補でどのMutantが落とせるか比較する。生成候補が受理前なら`candidate`、新Go/CI未実行なら`NOT RUN`。
3. **P1／三視点と変更検証**：#1124/#1125/#1803にORG-TG-04/06/07の型付きGraph/Tree/Function/CRUD/DFDのRead-only OutとSource/Role/版を渡す。SHOは別Consumer資格、図面上で一意なSourceがない変更はPreviewまでに留める。
4. **既存最重要Product E2**：#2586のC1→D1実Function→Query→Provider Read→Consumerを同一Attemptでつなぎ、FTE/人数等の追加Claimは必要な実Field/Scope/Populationを別々に観測した後に限り組み込む。P0-a/bの別合成PASSをProduct E2へ足し込まない。
5. **P2／Effectと受入**：実関係更新、Host Grant、Receipt、独立再Read、Owner業務Ruleは#1174/#1163/Org Ownerの既存境界を通す。#3258〜61の複数Draftから新Org専用Test Generator/SSOTを作らない。

**受入状態**：本節は**既存Org Profileの追加診断・適合計画**であり、ORG-TG-01〜08の新実行・C0/C1測定・AI Generator実装・Product SQL/Org DB/GUI/SHO/Owner Meaning受理は **NOT RUN**。既存完了・実証済みの局所テストと混ぜて新たな一気通貫PASSを宣言しない。
