# 意味的特性の宣言・観測・適合：閉値域／MECE／属性同一性の横断設計（2026-11）

- **状態**：設計・受入候補。新Canonical Schema／Core Kind／Plugin／Studio機能の採用、実装、テスト実行、Ownerの意味受理は主張しない。
- **目的**：業務設計者の意図と、Java／Go／.NET／SQL／ファイル／API／ETLなどの実現・観測を、出典と資格を失わず照合する。人間とAIが、手入力を最小限にして設計特性を発見・説明・修正できるようにする。
- **関連Owner**：[構造・Facet #2495](https://github.com/hat47x/tei-platform/issues/2495)、[演算と法則 #2544](https://github.com/hat47x/tei-platform/issues/2544)、[業務キーとFD #1126](https://github.com/hat47x/tei-platform/issues/1126)、[Everything is Data #2620](https://github.com/hat47x/tei-platform/issues/2620)、[認識・Review #958](https://github.com/hat47x/tei-platform/issues/958)、[横断変更 #1143](https://github.com/hat47x/tei-platform/issues/1143)、[Scenario #1131](https://github.com/hat47x/tei-platform/issues/1131)、[構造Projection #2497](https://github.com/hat47x/tei-platform/issues/2497)、[用途相対開示 #2516](https://github.com/hat47x/tei-platform/issues/2516)、[正式Acceptance #267](https://github.com/hat47x/tei-platform/issues/267)。
- **既存実装との接点**：[Semantic論理キー・FDの宣言](SEMANTIC_LOGICAL_IDENTITY_FD_AUTHORING.md)は明示Sourceの構造検証まで。ソース内の業務キーを勝手に導出しない。[意味変更とデータ進化](SEMANTIC_CHANGE_DATA_EVOLUTION_DESIGN_2026-10.md)の値影響・DB migrationは別Owner。[ADR-0030](../adr/0030-context-scoped-epistemic-knowledge.md)の認識的注記を第二正本にしない。
- **変更境界（2026-11-15更新）**：本書の初版は[PR #3027](https://github.com/hat47x/tei-platform/pull/3027)でmainへ統合済み。旧Draft #2496（Facet分類）と#2778（SQL/DDL）はclosed・未統合であり、両者の内容をmainの実装とみなさない。本書は既存の分類・検証・実装の結果を比較する設計であり、新しいGate、Registry、Semantic Authority、専用Hostを設けない。

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

**未解決調査**：「強属性」がX-TEAの資料・ソースに定義されているかを確認し、上記のどれに当たるか・または別のFacetが必要かを再判定する。X-TEAに定義が見つからない現時点では、`strong-attribute`という標準Role／Facet／DSLを追加しない。

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
- 既存Issueの優先度や担当の一方的変更。過去のDraftの状態を現在の実装状態とみなすこと。
- 成功したテストを、実Provider、実Policy、業務Ownerの意思決定、現在のデプロイに拡大すること。

新設が必要な最小共通契約は、TR Caseと既存契約の表現上の不足が**再現可能に確認された場合だけ**、対象Issueへ差分を返す。本書単体では正式仕様採否を決めない。

## 10. 現行実装から見たFit/Gap（2026-11-15追補）

**確認基準**：main `5a3f8895f0bbd1cf6302d75f8c64e6bb3ddeb1c3`。ここに書く実装の説明は、確認したソースと関連PR・Issueの範囲に限る。今回、新しいCLI試験、DB試験、Studio操作は実行していない。

| 確認したもの | 現在できること | まだ説明できないこと |
| --- | --- | --- |
| [`internal/semantic/model.go`](../../internal/semantic/model.go) | Data属性の`Required`・`Immutable`、明示した`CandidateKey`・`FunctionalDependency`を保持する。`Value`には未記述・明示的空文字・値ありの区別がある | `Immutable`を宣言しただけで、すべての更新経路が実際に変更を拒否すること。候補キーが業務上正しいこと |
| [`internal/semantic/completion.go`](../../internal/semantic/completion.go) | Ownerの完了条件について、成立・不成立・未検証・証拠矛盾を区別する。信頼できる反例は、全件取得がなくても不成立の根拠になり得る | 呼び手が渡す`Coverage`の真偽。実Source・実Consumerの取得範囲をこの評価器自身が確認するわけではない |
| [TEA Trait表示の参照実装](../../reference/tei-three-element-method/trait-attachments.mjs)（[統合PR #2972](https://github.com/hat47x/tei-platform/pull/2972)） | すでに公開許可・対象選定を終えたTrait Claimを、Data／Function／Businessの表示対象へ読み取り専用で対応付ける | Traitを発見すること、実Sourceを解析して評価すること、Policyや現在の権限を自ら確認すること。Snapshot文字列の一致は実Sourceの鮮度証明ではない |
| [三視点の証拠設計](THREE_VIEW_DIAGRAM_CROSS_REFERENCE_PROJECTION_2026-11.md)（[統合PR #4487](https://github.com/hat47x/tei-platform/pull/4487)） | §34.6〜34.10で、存在・全称・不存在・実行事実・業務受理を異なる問いとして整理し、Source改版後の証拠の再利用条件を検討できる | 新しい問い別検証がStudio・実Providerまで一巡したという実証。机上の条件を現在の全Consumerの保証へ広げること |
| [原子的Trait履行のDraft #4330](https://github.com/hat47x/tei-platform/pull/4330)（[#3279](https://github.com/hat47x/tei-platform/issues/3279)） | `tei.operations.inspect`のContract・Provider・実呼出し・不変条件を、一つの実装義務として読み取り専用で評価する候補 | 業務の閉値域や強属性の判定。`Closure`は呼び手の申告であり、全依存を独立取得して確認した証拠ではない。PRは未統合 |
| [モデル由来テスト #4134](https://github.com/hat47x/tei-platform/issues/4134) | 意図された特性から試験候補を作り、独立Expectedで検証する順序と担当を整理する | 生成器が出した試験データや期待値を、そのまま業務意味の正しさとして採用すること |

**最初に埋めるGap**は、新しいTrait型の追加ではない。一つの業務属性について「何を宣言したか」「何を実際に読んだか」「どの更新経路を確認できたか」を、同じSubject・Scope・版で説明することだ。属性宣言を読む機能と実装義務を調べる機能が別々に存在しても、その間の対応が未確認なら、全体の適合を主張できない。

### 10.1 同じ「Trait」という名前でも、二種類の主張を混ぜない

業務属性のTraitと、プラグインが実装する能力のTraitは異なる。

- **業務特性**：`order.status`が一定の値域に属する、`item.category`を変更すると業務上の同一性が変わる、といった意味についての主張。設計者が対象と条件を定める。業務上の正しさはSourceやDBの構造だけでは確定しない。
- **実装義務**：選択したProviderが指定のContractと不変条件を満たし、必要な呼出しを実行できる、という実装についての主張。#4330の原子的Traitはこの種類である。

両者は「対象」「要求」「根拠」「未確認」を整理する方法を共有できる。しかし、同じ`traitId`や表示名を持つだけでは同一の特性ではない。業務特性から実装義務へ進むには、受理済みの意味、対象Operation、実装先、強制すべき条件、必要な観測を**明示的に対応付ける**必要がある。対応がなければ「同じ名前の実装が見つかった」とだけ報告し、業務特性の適合には使わない。

### 10.2 自動判定の三段階を、観測範囲まで明示する

TEIが返す説明は、次の三段階に分ける。新しいCore状態を増やす案ではなく、既存の結果を読むための区別である。

1. **構造を見つけた**：Sourceから`Immutable`や`CandidateKey`などの宣言を読んだ。どのSourceのどの版かを示す。これは意図の表明を観測したのであって、実動作の保証ではない。
2. **実装に対応付いた**：受理済みの要求に対し、該当するValidator、Operation、DB制約、Providerなどを特定した。取得できなかったSource、別のWriter、適用時点が不明なら、調べられた範囲を明記する。
3. **必要な動作を確かめた**：同じ対象と条件で、実呼出しや独立した読取りなどの証拠を得た。単一の実行で成功したことと、今後の全実行・全Targetで保証できることは分ける。

一件の有資格な反例は、その反例が属する全称主張を否定し得る。一方、反例が見つからないことを一般的な成立の証拠にするには、対象の完全な範囲か、それに代わる有資格な保証が必要である。具体的な解釈は[三視点設計§34.6〜34.10](THREE_VIEW_DIAGRAM_CROSS_REFERENCE_PROJECTION_2026-11.md)を参照し、ここで別の判定器を作らない。

## 11. 次の小さな受入例：業務特性と実装義務をつなぐ

最初から多言語・全DBを対象にせず、次の二つを独立に扱う。

**例A：`sales.order.status`の閉値域**。三つの業務状態を許すという案に対し、Java Enum、PostgreSQL制約、API・取込・ETLのWriterを区別する。現行TEIにその値域が正式受理されているとは仮定しない。Javaの宣言が読めた時点では「Javaに3つの定数がある」と説明する。PostgreSQLの実DDLが`VARCHAR NOT NULL`なら、DB列の非NULL制約は確認できるが、値域を3値に限定する根拠は不足する。不正な行が存在するかは別途実データを調べる。MECEは対象母集団と分類述語を受理するまで未確認とする。

**例B：`Immutable`が宣言された属性**。`internal/semantic/model.go`の`Immutable`は設計の静的な入力として使える。ここから、同じ業務個体に対する更新Operationが実際に属性値の変更を拒否するかを調べる。値の変化を有資格な実Readbackで観測した場合、指定した条件の不変性に対する反例候補になる。ただし、訂正・版更新・別個体への置換が許されるかは設計者の受理済みルールに従い、更新を見つけただけで「強属性違反」と断定しない。「強属性」の原典定義は引き続き未確認である。

### 11.1 追加する判別票（設計候補・未実行）

| ID | 入力と問い | 期待する診断 | 担当 |
| --- | --- | --- | --- |
| **TR-13** | Domainの閉値域Claimと、同名のPlugin能力Traitを並べる | 明示的なMeaning→Operation→Contract対応がなければ、業務適合の根拠に使わない | #3027／#3279 |
| **TR-14** | `Immutable=true`、変更可能な書込み経路の存在、実Readbackは未取得 | 宣言と実装の調査結果は表示できるが、実際に変更されたとは断定しない | #1126／#1143 |
| **TR-15** | 同じ対象・適用条件で属性値の変更を独立Readbackで確認 | 対応する不変性主張への反例候補を示す。訂正例外や業務上の同一性は別に確認する | #1129／#1180 |
| **TR-16** | Callerが`Coverage=complete`を申告し、外部Importを実際には読んでいない | `complete`という申告から全Source取得済み・全Writer確認済みへ昇格しない | #3262／#2620 |
| **TR-17** | 同じ公開Trait表示に対し、非公開Sourceだけ異なる | 表示本文、件数、診断、ツール応答から秘密の差を推測できない。実Policy検証は別 | #2516／#1124 |
| **TR-18** | Intentから試験データとExpectedを同じ生成器が作る | 試験実行が成功しても独立Oracleの資格がない。#4134の別Expectedを要求する | #4134／#2514 |
| **TR-19** | SourceやMethod Ruleを更新し、以前の適合結果を再利用する | 旧根拠を履歴として保持し、現在のScope・依存・改版対応が確認できない場合は現在適合を未確認にする | #1143／#2600 |

判別票の追加は計画だけであり、既存テストがTR-13〜19に合格したという記録ではない。#4134のモデル由来テストに候補を渡す際は、同趣旨のMDTケースとの重複を照合する。

### 11.2 最初の成果物は「警告一覧」よりも、根拠を比べられる一画面にする

一つの属性について、設計者が**意図／確認した実装／実行時の証拠／確認できない範囲**を並べて読めればよい。必ずしも新しいStudio画面が必要とは限らない。既存CLIのread-only結果と[三視点のTrait表示](../../reference/tei-three-element-method/trait-attachments.mjs)を使える範囲で比較する。

警告は少なくとも、①受理済み意図に反する有資格な反例、②実装上の保証が見つからない、③確認に必要なSourceや権限が不足する、を別の説明にする。未受理のAI提案は①に入れない。情報を開示できない場合も、非公開の対象や件数を警告から推測できないようにする。

## 12. 計画への適用と停止条件（2026-11-15）

- **P0：実Sourceに即したFit/Gap**。既存`Attribute`・CandidateKey/FD・Source Index・Validation・TEA表示の間で、実際に参照できるSubjectと証拠を一件選ぶ。まずTR-13、TR-14、TR-16の独立Expectedを作る。閉値域の正式Sourceがない場合は、構文を推測して補わずGapとして記録する。
- **P1：観測・反例**。実Provider／DBで対象と時点が一致することを確認できた場合だけTR-15を実証する。#2586の実Reader、#1180のScenarioと独立Read、#3279の実装義務を再利用し、今回独自のRunnerやProviderを作らない。
- **P1：三視点への説明**。#4487の問い別証拠と#2972の公開Trait表示を使い、同じSubjectをCLIとStudioで比較する。#1124の選択対象・鮮度・公開条件を確認するまで、合成表示を実Studioの適合とは扱わない。
- **P2：試験生成・変更影響**。独立Expectedが整った判別票だけ#4134に渡す。SourceやTargetを変更したときの証拠の再利用は#1143/#2600に従う。セルフビルドの工程への接続は#2831の実Build証拠に委ねる。

**停止条件**：対象の同一性が曖昧、業務規則が未受理、必要なSourceが未取得、全Writerの範囲が不明、実ProviderのGrantが未確認、独立Expectedがない、現在の公開許可が成立していない場合は、その条件に依存する判定だけを未確認にする。別の対象で得た有資格な局所証拠まで無効にはしない。形式検証の成功を、業務判断や実行権限の承認にはしない。

この追補による新しいCore Trait型・Reasoner・Source Writer・Gate・業務Ownerの決定はない。採用判断は各IssueのOwnerへ戻す。

## 13. 問いに必要な証拠の閉包と、標準データ様式の再利用（2026-11-15）

この節は、標準データ様式の段階計画 [#4350](https://github.com/hat47x/tei-platform/issues/4350)、構造写像 [#2497](https://github.com/hat47x/tei-platform/issues/2497)、Function Data Context [#2506](https://github.com/hat47x/tei-platform/issues/2506) と、現行のTrait検証をつなぐ**検収上の補足**である。新しい結果型、閉包Engine、汎用Query評価器や第二のScenario正本を定義しない。

### 13.1 「何を見たか」と「どの結論を出せるか」を分ける

同じRecord、Collection、GraphやTableを扱っていても、利用するFunctionの問いによって必要な証拠が違う。結果値の一致、観測済みデータの個数、必要な全データの取得完了を一つの「正しい」にまとめない。観測が部分的でも**局所的な存在証拠や反例は成立し得る**が、単に見つからないことから不存在や唯一性を断定することはできない。

| 問い | 部分取得でも成立し得る局所証拠 | 全体として結論を出す追加条件 |
| --- | --- | --- |
| 指定条件を満たす個体が存在するか | 対象・述語・時点が有資格な一件のWitnessで、存在を確認できる | 存在しないという結論には、対象Scopeの完全取得か、適用可能な制約・証明が必要 |
| 該当数が少なくともN件か | 重複とIdentityを確定できる異なるN件のWitnessで下限を確認できる | 正確な総件数やN件以下の主張には、対象集合と重複規則の閉包が必要 |
| 一意な規則・対象だけが該当するか | 同時に該当する異なる二件の有資格Witnessで、一意性を否定できる | 一意と断定するには、規則集合と評価対象の両方について、Scope・時点・取得完全性を確認する |
| 該当しない、違反しない、他に経路がないか | 一件の有資格な違反・別経路は全称／不存在の主張を反証できる | 違反や経路が見つからないだけでは足りず、探索対象と関係の閉包が必要 |
| 合計・平均・期間集計が正確か | 取得済み範囲についての部分集計・計算過程は説明できる | 母集団の閉包、集計粒度、NULL・未取得・重複、多重所属、端数処理、時点が必要 |

この区別は、論理的には「新しい対象を追加しても成立した肯定的証拠が覆らない問い」と、「追加の対象によって結論が変わる問い」の差を利用する。この性質は、**同じ問・述語・対象Scope・評価時点・個体同定規則の下で、既存観測を更新せずに取得範囲だけ広げる場合**に限って利用する。既存値の更新、Rule版変更、時点移動、認可Scope変更、Sourceの再解析は単なる件数追加ではなく、根拠の再確認が必要である。ただし、**権限や開示の境界を越えて全体の個数・有無を推測してよい**という意味ではない。表示可能なWitness・件数・診断自体を、現在のAudience、Tenant、Purpose、Hostの開示判断に従わせる。

### 13.2 閉包を一枚のフラグにしない

Functionが結論を出す際の確認範囲は、最低限、次を区別する。

| 確認対象 | 確認する内容 | 欠けた場合の扱い |
| --- | --- | --- |
| 対象 | 業務Identity、Graph Relation occurrence、親子集合、Business Key、参照の対応 | 同じ表示ラベル・物理行・描画Edgeから個体を決めない |
| 評価範囲 | Function Data Context、grain、filter、page/window、snapshot、valid/known時点 | 局所Witnessを保てても、全件・不存在の結論は保留 |
| 規則 | 選択済みRule ID・版、適用Scope、優先順位、例外、必要なRule集合の閉包 | 「読めた規則だけで唯一」と判断しない |
| 読取 | 必要なRead Footprint、実際のProvider Read、NULL／missing／unfetched／withheld | 宣言された依存関係と実際に読めた入力を同一視しない |
| 実現・配布 | Source版、生成器、Build入力、Artifact、実Provider、独立Consumer | Reference結果やBuild成功を通常CLI／実Host受領へ昇格しない |
| 操作 | 表現型のSelection、逆写像Loss、現在のGrant、EffectとReceipt | 表示できることをSource編集・業務Actionの資格にしない |

この表は既存のSource Index・Trace・Context・Scenarioから読取専用で構成する**説明票**であり、新しい永続Ledgerではない。出典が非公開・未取得なら、公開面にその存在や件数を漏らさず、可能な範囲の不足理由だけを示す。値が再び一致するA→B→A改版でも、旧Source位置・Attempt・Consumerの資格を自動復活させない。

### 13.3 横断Pressure Caseと引渡し

| ID | 一因子の差・入力例 | 独立Expectedへ渡す判別点 | 既存Owner |
| --- | --- | --- | --- |
| TR-Q20 | 1/3ページだけを取得し、そこに有資格な該当行を一件発見 | 存在の局所Witnessは保持するが、全件件数／不存在を主張しない | #2506／#2514 |
| TR-Q21 | 一意候補を一件だけ取得、規則集合の一部は未取得 | 唯一性は未確認。既知の二件が競合すれば、全件取得前でも競合を示せる | #2620／#1180 |
| TR-Q22 | 多重有向Graphの同端点・同label二EdgeをTreeへ投影 | Edge occurrenceを潰さず、非表示の関係や逆編集で失われる情報を明示する | #2497／#2500 |
| TR-Q23 | 二つの子集合（2件・3件）を同じPanelへ配置 | JOINの6組と原子集合の2件・3件を区別し、集計grainと実行Partitionを混同しない | #2506／#2497 |
| TR-Q24 | 実保存された外部EvidenceのReceiptは存在するが、解析Graphが空 | Receiptの受領は成立する一方、Graph候補生成は未成立として拒否し、架空のRelationを補わない | #2621／#2514 |
| TR-Q25 | SourceのRule参照属性だけ変更、表示値は以前と一致 | Source／Semantic／生成Go／実Builder／Consumerの失効を別に検査し、結果一致だけで旧資格を再利用しない | #1143／#1121／#2514 |
| TR-Q26 | 同じGraph表示部品へBusiness RelationとPlugin配線をそれぞれ投影 | Node/Edge描画IDを業務・Plugin・Operation IDへ変換せず、Declared／Selected／Built／Observedを区別する | #2497／#951／#1124 |

**最初の実装単位**はTR-Q20/Q21の合成Fixtureを、既存のRead FootprintとFunction Context・独立Expectedで説明できるか確かめることとする。TR-Q22/Q26は既存の標準Graph／Node View read-only投影へ、TR-Q23はBatch/ETLの計算粒度・出力粒度の設計へ、TR-Q24は外部Evidenceの元資料と候補生成の検収へ、TR-Q25はRuleから実Go生成・Builder・通常CLIの縦断へそれぞれ戻す。

S1〜S3のread-only Referenceがmainに統合されていても、製品の正式Graph Reader、汎用Trait判定器、実ProviderのRead閉包、Studioでの逆編集、Ownerの意味受理は別の完了条件である。**Studio DesktopのStage1→Stage2実Windows受領と、選択された実Providerを通常CLIから呼ぶ製品P0を変更しない**。この節の判別票は未実行の設計候補として扱い、採否と実装は各Ownerに委ねる。

## 14. 項目定義表から型付き実装へ：意味を増やさない最小縦断（2026-11-15）

### 14.1 データ様式を、定義・記録・表現・処理の四つで再利用する

一つのMarkdown表があれば、そこに書かれているすべてが業務データになるわけではない。**表の役割は、利用する言語契約と明示的なSource位置で決まる**。現行の選択済み`tei.authoring/v1`で`<!-- tei: data TransferRequest in organization -->`が直結する項目表は、一行を一属性とする`semantic.Data`の定義である。印のない「申請例」の行は記録として取り込まれず、DB保存や更新にも使われない。新旧のMarkdown言語が同じ`*.tei.md`を同時に主張すれば、Readerを推測で選ばずLoadを拒否する。

| 使い道 | 同じSourceから取り出すもの | 自動的に成立させないこと |
| --- | --- | --- |
| 定義 | Data ID、Attribute ID、型、required、表示名、出典のbyte位置 | 実レコード、業務Key、参照先の存在、NULLと欠損の扱い |
| 記録 | DB/API/Fileの実Readで得たOccurrence、値、時点、取得範囲 | 定義表のサンプル行からの実データ生成、全件性 |
| 表現 | Form／Table／Tree／Graphなどのread-only ProjectionとSource Mapping | 表示IDからの業務Identity確定、非単射な逆編集、Grant |
| 処理 | 型付きRule／Operator、依存Read、生成コードまたは汎用関数 | Reader合格だけでの実Builder消費、選択Provider起動、業務意味受理 |

`required: no`／未指定を読み取っても、未設定・明示NULL・空文字列・未取得を同じ値へ正規化しない。項目定義で必要性が指定されていないことは、実行時の値の欠落を無条件に許可することでもない。PATCHの省略と明示NULLは、データ取込／更新Intentと現在の書込権限で別に判断する。

### 14.2 実装して判別する最初の一辺

既存の`internal/workspace/testdata/tei-md-data-table`では、`TransferRequest`項目表と`allocation_positive`のRuleが一つの`*.tei.md`に共存する。Loaderは同じSourceから`semantic.Data`と`semantic.Validation`を返し、Source Indexでそれぞれの位置を保持する。新しい検証`TestTEIMDDataTableRuleGoLoweringAndSourceRevision`では、表から解決された`organization.transfer_request.allocation`の型を**同じLoadのRuleに渡し**、既存の`validationgen.GeneratePositive`から`source.Number(...)`と`value.Sign() > 0`を含むGoを生成する。

ここで生成するGoは、選択済みRuleの判断処理を実際に含む。ただし、このテスト単体では生成GoをBuild Plan／製品CLI経由でコンパイル・実行していない。既存Rule由来のGo実Build試験と、このData表由来の型・位置・Source改版試験を**別の証拠として接続**し、元表から通常`tei self-build`・正式Builder・Binary Consumerまでを単一Attemptで通したと認定しない。

次の二つの差分が、結果の一致より重要である。

1. **説明だけ変更する場合**：型付きRuleから生成されるGoが同じでも、管理対象`*.tei.md`のbyteが変わる。古いSource Indexは失効し、新しいLoadとIndexを作り直す。Go byte一致を旧Buildの意味受理・出典資格の継承に使わない。
2. **`std.decimal`を`std.text`に変える場合**：Ruleの文字列や入力例が変わらなくても、生成器は同じLoadから解決した型を使い、Decimal専用の`positive` loweringを拒否する。呼出し側が古い型を直接渡して合格にする経路は設けない。

### 14.3 異なる実例でも保持するIdentityと時間

本縦断に続くGraphと組織二時点データには、同じ原則を適用するが別のOwnerで検証する。

- **Graph／REA**：関係IDが存在することと、観測されたNode IDがその関係の端点に一致することを分ける。main統合済み#5038の実Verifier負例は、物理的な参照整合を試せる。しかし一致だけでは業務Relationの受理・現在のPlugin配線・実Callを証明しない。多重Edgeは端点が同じでも個別Occurrenceである。
- **組織・履歴**：main統合済み#5027の隔離PostgreSQLケースでは、同じMembershipの訂正版と、異なるMembershipの同時成立を分ける。同じ`validAt`・`knownAt`に複数行が存在しても、別Occurrenceなら「同一関係の版衝突」ではない。データ様式側で時刻を一つに潰したり、Graphの端点だけから版を一意化しない。
- **Process／Plugin**：Node Viewは標準Graphの表現型として共通化するが、Declared／Selected／Built／Observedと業務Relationの受理は別。描画線と実Provider呼出しは同一の事象ではない。

したがって、標準データ様式の汎化は「表・図・フローに共通の構造を持たせる」だけでは終わらない。**StructureとOccurrence、適用Scopeと時点、表現Loss、処理に必要なRead、生成・実行の受領資格を、失わずに別々に追跡できること**を横断受入条件とする。専用Graph Runtime／Temporal Engine／新しい意味正本は追加しない。

### 14.4 既存の独立Expectedと計画への引渡し

| ID | 判別する差分 | 現状・次の検証先 |
| --- | --- | --- |
| **DFG-01** | 印つき項目表とネイティブData宣言を、同じAttribute ID・型・required・表示とSource位置で比較 | 既存`tei_md_data_table_test.go`の局所試験。製品Runtimeは別 |
| **DFG-02** | 同じ`tei.authoring/v1`の表の型とRuleから生成Goへ変換し、参照先Attributeと生成分岐を検査 | 新規Go回帰を追加。exact HEADのCI判定まではPASS未記録 |
| **DFG-03** | 説明のみ変更し、Go byteは同じでも旧Source Indexを拒否 | 新規Go回帰を追加。古いArtifact／Consumerの失効は別試験 |
| **DFG-04** | 表の型だけDecimal→Textへ変更し、Rule式は不変 | 新規Go回帰で型不一致の生成拒否を検査 |
| **DFG-05** | 印のない例示表、未指定・明示NULL・部分取得、PATCH省略を混同しない | 表の非取込は既存試験。更新と実Recordは#1127／#1180／#2514 |
| **DFG-06** | 同端点の多重Relation、片端点のみ観測、二時点の訂正・別Occurrenceを同一Graph投影で判別 | #5038／#5027の局所成果を#2497／#2386へ接続。新規製品Graph Readerは未受理 |
| **DFG-07** | 表→Rule→生成Go→実Builder入力→Artifact→通常CLI実Provider／独立Consumerを同じSource Revisionで束縛 | #1121／#4495／#2514の製品縦断で検証。今回の追加試験だけでは未達 |

最初の合格判定はDFG-02〜04の局所Goテストと既存DFG-01の維持に限定する。続く実装では、既存`tei self-build`の生成Go受領とSource Planの双方に**同じ表定義とRule IDが実際に使用されたこと**を紐付ける。自動生成が適する処理と、標準Data／Operationを受け取る汎用関数で十分な処理は、DSLに見える構造を同じまま保って選べるようにする。どちらの経路でも意味の独立Expectedと実Consumerの観測は省略しない。

引渡し先はデータ様式#4350、Reader／DSL#2621、Function Data Context#2506、Mapping#2497、Rule Build#1121、実Provider#4495、独立Expected#1180／#2514、Studio Stage1→Stage2 #2831／#4583。**通常CLIの実Provider受領とStudio DesktopセルフビルドのP0を維持する**。新しいGateを作るのではなく、既存の合否票に今回増えた証拠を接続する。
