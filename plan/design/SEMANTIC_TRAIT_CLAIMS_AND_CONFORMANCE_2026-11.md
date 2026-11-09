# 意味的特性の宣言・観測・適合：閉値域／MECE／属性同一性の横断設計（2026-11）

- **状態**：設計・受入候補。新Canonical Schema／Core Kind／Plugin／Studio機能の採用、実装、テスト実行、Ownerの意味受理は主張しない。
- **目的**：業務設計者の意図と、Java／Go／.NET／SQL／ファイル／API／ETLなどの実現・観測を、出典と資格を失わず照合する。人間とAIが、手入力を最小限にして設計特性を発見・説明・修正できるようにする。
- **関連Owner**：[構造・Facet #2495](https://github.com/hat47x/tei-platform/issues/2495)、[演算と法則 #2544](https://github.com/hat47x/tei-platform/issues/2544)、[業務キーとFD #1126](https://github.com/hat47x/tei-platform/issues/1126)、[Everything is Data #2620](https://github.com/hat47x/tei-platform/issues/2620)、[認識・Review #958](https://github.com/hat47x/tei-platform/issues/958)、[横断変更 #1143](https://github.com/hat47x/tei-platform/issues/1143)、[Scenario #1131](https://github.com/hat47x/tei-platform/issues/1131)、[構造Projection #2497](https://github.com/hat47x/tei-platform/issues/2497)、[用途相対開示 #2516](https://github.com/hat47x/tei-platform/issues/2516)、[正式Acceptance #267](https://github.com/hat47x/tei-platform/issues/267)。
- **既存実装との接点**：[Semantic論理キー・FDの宣言](SEMANTIC_LOGICAL_IDENTITY_FD_AUTHORING.md)は明示Sourceの構造検証まで。ソース内の業務キーを勝手に導出しない。[意味変更とデータ進化](SEMANTIC_CHANGE_DATA_EVOLUTION_DESIGN_2026-10.md)の値影響・DB migrationは別Owner。[ADR-0030](../adr/0030-context-scoped-epistemic-knowledge.md)の認識的注記を第二正本にしない。
- **変更境界**：#2496はFacet上位分類とLawの独立Draft、#2778はSQL/DDL静的事実の独立Draft。双方の担当ファイル・ブランチへ割り込まず、本書は両者の**最小横断評価票と負例**を提示する。新Gate、新Registry、新Semantic Authority、専用Hostは提案しない。

## 1. 結論：Traitの種類と「対象がTraitを満たす」というClaimを別にする

概念的に、次の4要素を既存標準データ構造・Typed Reference・Evidence・Qualificationの**読み取り用合成**として比較する。名称は説明用であり、現行TEI YAML/DSL APIではない。

1. **Trait Definition**：どんな問い・述語を評価するか。例：閉じた値域、排他性、網羅性、不変性、関数従属性、時間的有効性、純粋性。型階層や単一のMECE分類へすべてを押し込まない。#2495のfacetとLawは既存Ownerに従う。
2. **Intent Claim**：設計者が「対象Sは条件C・母集団U・時点Tで性質Pを持つべき」と宣言した意味上の主張。受理済み・提案中・却下・撤回の履歴とActor/Authorityは別。文字列ラベルで権限を得ない。
3. **Observation**：実コード、DDL、取得した実データ、試験、実Provider、生成物などが**実際に示した事実**。ソース内の表記／静的制約／サンプルデータ上の成立／実運用での成立は別。観測は正本Intentにならない。
4. **Assessment**：特定のUse-siteと現在の根拠を比較した説明可能な診断。結果だけでなく、確認した保証・残余Unknown・Scope差・根拠の版・反例の種類を残す。形式適合は正式Acceptance、実行Authority、独立Effect観測ではない。

比較用の表現：

```text
TraitClaim(subject, predicate, quantifier, universe, conditions,
           semanticRevision, effectiveTime, provenance, intentState)
ObservedFact(subjectBinding, predicate, observedScope, sourceRevision,
             observationMethod, targetArtifact, evidenceRef)
Assess(intentClaim, observedFacts, useSite, disclosureContext)
  -> {finding, supportedClaims, uncoveredObligations, evidenceRefs}
```

**Join条件**：対象の同一性は名称一致で代用しない。Typed Subject/Business Identityと物理列・DTO・Operationの対応、Source rev、観測時点・effective time、対象母集団、Reference/Provider/Artifactが対応して初めて比較可能。Scopeが重ならない・参照が曖昧・旧観測しかない場合、空の観測結果から「欠落なし」「適合」を導かない。

**結果の状態**：適合証拠あり、反例あり、主張同士の矛盾、根拠不足、根拠失効、対象外を区別する。ただし#2544の既存語彙と衝突する新しい全域Enumは先行定義しない。特に「**制約による保証がない**」「**制約に違反し得る**」「**違反行を観測した**」は3つの異なる診断。

## 2. 閉値域・MECE・Enumを数学的に分離する

分類対象の**母集団**を `U`、有限または記号的な分類を `C_i` とする。静的構文は値の列挙を示し得るが、それだけでMECEとはならない。

- **Closed Domain**：値の定義・拡張方針が閉じた集合である、という**文脈付き主張**。その場のコードに列挙されたことと、DB・外部API・将来版を含む業務値域が閉じていることは別。
- **Mutually Exclusive**：任意の異なる `i,j` に対して `C_i ∩ C_j = ∅`。複数ラベルを許す分類軸なら非該当。
- **Collectively Exhaustive**：`U = ⋃ C_i`。`U`が未指定なら確認不能。空のデータ集合上の全称命題が真でも、業務の対象全体を列挙した証拠にはならない。
- **Single-valued / Presence / Null**：属性が単値か、必須か、NULL・Missing・Unknownをどう扱うかは独立。SQL三値論理や`CHECK`のUNKNOWN許容を考慮。Java EnumはDBの全書込み経路を閉じない。
- **Version / Extension Policy**：値追加・廃止の互換性、未知値の拒否／保持／退避／forward-compatible処理の方式を明示。いずれも「常に未来の値を含めてMECE」を自動証明しない。
- **Enforcement coverage**：どの境界（Java DTO、Go validator、DB CHECK、API、ETL、ファイル取込、他アプリ）が規則を実際に強制するか。宣言された規則が1つあるだけで全入口の保証とはしない。

業務分類を検証するには、分類ルールの述語、対象となる集合の定義、観測範囲、要求する証拠の種類を先に確定する。サンプル有限列挙で排他・網羅を確認した場合は `bounded example`、一般成立を明示的に証明した場合のみ `proof under assumptions` と区別する。

## 3. 強属性の扱い：原典未確認の用語を正本化しない

設計上検討する候補として、以下の**独立した問い**を分ける。いずれも「渡辺氏の強属性の定義」と同一だと断定しない。

| 観点 | 判別する問い | 設計者／自動 |
| --- | --- | --- |
| Identity-defining | 変更すると**同じ業務個体**か別個体かが変わるか | 意味は設計者。変更処理の観測は自動 |
| Temporal immutability | 生成後、同じ個体の属性値を業務上変更してよいか（例外・訂正・有効期間を含む） | 設計者＋履歴/操作の自動照合 |
| Candidate key / FD | 候補キー・決定属性なのか、単に現データが重複しなかっただけか | 明示宣言＋限定探索。業務意味は推定のみ |
| Ownership / existence dependency | 親・所有者・参照先変更と個体の同一性はどう関係するか | 設計者＋Relation/DDL/操作検査 |
| Derived / authoritative | その値は正本か、条件付き導出値か、materialized cacheか | 宣言＋式/Query/Lineageの静的照合 |

**未解決調査**：「強属性」の厳密な定義が掲載された原記事・版・文脈を確認し、上記のどれに当たるか・または別のFacetが必要かを再判定する。一次資料未確認の現時点では、`strong-attribute`という標準Role／Facet／DSLを追加しない。参考：[渡辺氏による三要素分析の解説](https://atmarkit.itmedia.co.jp/fbiz/cbuild/complete/doa/01/02.html)、[著書の出版社ページ](https://www.njg.co.jp/book/9784534057778/)（**これらは「強属性」の原定義の根拠ではない**）。

## 4. 特性の探索対象と設定・観測責務

区分の軸自体は重ならないよう**問いの責務**を整理する。一つの対象に複数のFacetが同時に適用されることは許容する（#2495）。

| 質問の責務 | 代表Trait候補 | 意図の入力 | 取得可能なObservation |
| --- | --- | --- | --- |
| 値と所属 | ClosedDomain、MECE、範囲、NullPolicy、Optionality | 対象母集団と分類意図は人 | Enum/DDL/JSON Schema/API例。意味MECEは自動確定しない |
| 同一性と関係 | BusinessIdentity、CandidateKey、FD、親子/参照、多重度 | 人が意味を決定 | 宣言済みFD、UNIQUE/PK、実例の重複。PKから業務IDへ昇格しない |
| 時間と変化 | Immutable、WriteOnce、TimeBound、ValidTime、RecordedTime | 人が許容変更を決定 | 更新コード、履歴、period、観測可能な実Effect |
| 状態と業務手順 | StateSet、Transition、Pre/Postcondition、Terminal、Exception | 人が業務意味・例外を決定 | Pattern、Scenario、生成コード、実結果、独立read |
| 導出とデータ品質 | Derived、Materialized、Nullable、Missing、Completeness、Lineage | 正本・導出式は人が受理 | Query、ETL DAG、計算式、データ集合の取得範囲 |
| 演算・数学 | Ordered、Unit/Dimension、Associative、Idempotent、Pure | 法則・演算束縛は明示 | 型/式/解析、有限性検査、Target別actual、#2544のLaw証拠 |
| 副作用と権限 | ReadOnly、Effectful、AuthorizedWrite、PrincipalScope | Policy/業務受理は人と選択Provider | 静的Effect宣言、実Grant、独立Effect観測。能力から権限を推定しない |
| 実現と配布 | TargetCompatibility、ProjectionLoss、ArtifactClosure | 受入条件はOwner | 実生成物、実Source closure、Target/Build receipts。自動セルフビルドの主張は不可 |

**設定方針**：(a)ソースに明示された実装事実は自動抽出、(b)静的構造・データからの仮説は推薦、(c)業務意味・例外・Authorityは人間の受理を要求、(d)モデル上の明示Intentと機械的事実の差だけを定量診断する。ユーザーにすべてのFacetの常時入力を要求しない。

## 5. 実例：Intentと各実装の観測境界

```yaml
# 説明用の判別票。現行TEI正本DSL/JSON schemaではない。
subject: sales.order.status
predicate: closed-domain
intent:
  universe: accepted-sales-orders
  domain: [NEW, ACTIVE, CLOSED]
  excludesNull: true
  effectiveAsOf: revision-R
  acceptance: unresolved  # Owner受理済みだと偽らない
observations:
  - binding: Java-OrderStatus
    evidenceKind: static-enum-declaration
    revision: java-revision-A
  - binding: postgres-orders-status
    evidenceKind: static-ddl
    observed: varchar-not-null-without-value-check
    revision: schema-revision-B
assessment:
  finding: enforcement-gap
  supported: [java-enum-constants-present, db-not-null]
  notSupported: [db-closed-domain, all-writers-validated, business-mece]
  observedViolationCount: not-measured
```

`status IN ('NEW','ACTIVE','CLOSED')`というSQL `CHECK`を追加しても、列がNULL可なら通常のSQL CHECKの三値論理によってNULLが通り得る。NULLの許容・禁止を別途確認する。DBにCHECKがないことは直ちに違反レコードが存在する証拠ではない。逆に実データに違反が0件でも、あらゆる将来書込みが制約される証拠ではない。

## 6. 最小の独立Pressure Case／負例

| ID | 正の観測・必要なClaim | 必ず拒否／保留する誤昇格 | Owner |
| --- | --- | --- | --- |
| TR-01 | Java Enumの値集合とDBの書込み強制を別Evidenceとして出す | Enum存在だけでDB・API・ファイル連携をclosedと断定 | #2495/#1126 |
| TR-02 | `CHECK IN (...)`とNOT NULL、missing、unknownを分離 | `CHECK`だけでNULL禁止・MECE成立と断定 | #2056/#1126 |
| TR-03 | 母集団Uと分類述語を明示し排他・網羅を別々に検証 | 値3件の列挙・サンプル0件からMECE一般証明 | #2495/#2544 |
| TR-04 | 代理PKと業務キー、Identity-defining属性を別に宣言 | DB surrogate PKや観測UNIQUEから業務同一性を創作 | #1126 |
| TR-05 | 強属性候補の不変性・同一性・時点を別説明 | 属性が更新されない実例だけで強属性確定、原典を誤引用 | #1126/#2495 |
| TR-06 | 有効時点別の条件付きFDと利用時点を比較 | 現在のFDが全履歴で常に成立するとみなす | #1126/#2144 |
| TR-07 | 状態遷移の分岐をaccepted state setと照合する | default分岐があるから業務例外がすべて正しく処理されると断定 | #1131/#1125 |
| TR-08 | ETL内の全分岐・sink・対象範囲を列挙する | 参照していないファイル・未ロードSourceの欠落を存在しないと断定 | #2620/#2823 |
| TR-09 | 同じMeaningに対するSource R1/Artifact A1とR2/A2を区別 | 昔のPASSを新しいデプロイの現在適合と判定 | #958/#2497 |
| TR-10 | 実Policyで許されたPublic SliceだけをStudio/CUIへ渡す | 非公開Traitの存在・件数・Pathを診断や候補件数で漏らす | #2516/#1124 |
| TR-11 | 同じ演算名のFacetでも具体的束縛・Law・Use-siteを固定 | `has-facet Monoid`だけから任意のcombineを選択 | #2544 |
| TR-12 | Intentを変更し、独立Observationで差を測り、Ownerが受理する | AI提案やConformance PASSを自動Accept、書込み許可へ昇格 | #958/#267/#1143 |

上記は**未実行の判別票**。既存Corpus（#2501）へ組み込むときは重複を照合し、新しい正本やCI Gateを増やさない。反例を成功Caseと独立に用意し、評価器自身の出力をExpectedに使わない（#2514）。

## 7. Studio / CUI / 生成器・ETLへの表示条件

- **Studio**：Data/Relation/Function/Businessの同じAccepted Meaningを表示し、(a)人間が宣言したTrait、(b)ソース抽出したfact、(c)AI提案・未受理、(d)比較結果の4種を見た目と操作で区別する。未解析・非開示は「存在しない」にしない。既存#1124の選択同一性/Scope・Source/ledger/Consumer freshnessを再利用。
- **CUI/AI**：確認済みEvidenceだけを引用。現在の許容DisclosureによるRead-setを構成し、秘密の件数・Path・候補が推測されないことを別Consumerで試す。提案はReview可能な候補に留める。
- **生成器・移行**：型・DDL・Validation・API・ETLにTraitに対応する制約を生成するのは**明示Meaningが受理された後**。実現にはTarget差や外部Writerがあり、「生成した」≠「全入口で強制された」。新しいEnum値、業務IDの変更、FDの縮小等は既存意味変更・移行EvidenceのOwnerへ渡す。
- **図表とメタ生成**：実際に依存したSource、適用したValidator、Coverage、生成Target、未検証Edgeを図表へ載せる。未知の暗黙処理を完全性と偽らない。セルフビルドの実Build Graphへの昇格は#2831の独立実証を必要とする。
- **TEA三要素**：値域・業務キー・関係はData、状態・責務・例外はBusiness、入出力と制約の強制点はFunctionに対応付くが、Sourceごとに正本を3つ作らずTyped Referenceで接合する。

## 8. 作業計画と受入判定（既存Gateの増設・優先順置換はしない）

| 段階 | 次の具体的成果 | 完了条件／未保証 |
| --- | --- | --- |
| **T0 source inventory** | Java Enum、Go列挙、Postgres列制約、TEI CandidateKey/FDの現行Source・宣言・実装を少数固定SHAで索引。強属性の一次定義は要探索 | 原文・実装事実・解釈未確認を混同しない |
| **T1 negative packet** | TR-01〜04、09、10の正負fixtureを既存Semantic/Source Index/SQL/DisclosureのOwnerへ振分け、独立Expectedを作る | Source coverage/Policy 未知ならUnverified。現在の実装PASSを追加で主張しない |
| **T2 read-only trait view** | 1つのSubjectにIntent/Observed/Assessmentを別列でCUIに提示し、同Source/Scope/RevisionでStudioへ試験的に投影 | 別のCanonical Store・Writer・Permission Engineは作らない |
| **T3 Business & ETL** | 不変性、時限属性、FD、状態遷移、ETL分岐の反例を段階的に追加し、生成Go/Java/.NET/DBへの強制範囲を明示 | Target単体成功を業務全体・Owner受理・実Effect成功へ昇格しない |
| **T4 qualification** | 最新main/PRのexact SHA、独立Oracle、negative、consumer disclosure、必要なactual ProviderとOperator/Owner受理を別々に記録 | #1119・#267などの既存Gateは変更しない |

**最初の狭い縦断受入**：受理前の `sales.order.status` の閉値域案について、Java 3値・Postgres `VARCHAR NOT NULL`・未観測の別Writerを示し、(1)Javaの宣言を観測、(2)DBの値域保証不足を指摘、(3)不正行の存在は未観測、(4)MECE/Business Identity/全書込み経路は未確認、(5)Studio/CUIとも同じ公開Evidenceのみ表示、という5点を同じAttemptで説明できること。実行なしでの机上設計を正式受入とはしない。

## 9. 非目標・Owner引継ぎ

- Core固定Trait enumの増殖、Ontology/Reasoner/Proof Engineの強制導入、別のKnowledge Canonical/Assessment Authority/Review台帳の新設、Host必須化。
- 静的コード・DB PK・データ分布から業務意味や「強属性」を自動採用すること。
- 既存Issueの優先度や担当の一方的変更、Draft #2496/#2778/#2860への別担当変更。
- 成功したテストを、実Provider、実Policy、業務Ownerの意思決定、現在のデプロイに拡大すること。

新設が必要な最小共通契約は、TR Caseと既存契約の表現上の不足が**再現可能に確認された場合だけ**、対象Issueへ差分を返す。本書単体では正式仕様採否を決めない。
