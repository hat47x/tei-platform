# 多重Facetの演算束縛・数学的公理・矛盾の評価（2026-11、設計候補）

- **状態**：机上設計・検証計画。現行DSLやCore Schema、数学的証明の実装、Ownerの意味受理ではない。
- **責務**：[Issue #2544](https://github.com/hat47x/tei-platform/issues/2544)。上位分類は[#2495](https://github.com/hat47x/tei-platform/issues/2495)、構造間の情報保存・Projectionは[#2497](https://github.com/hat47x/tei-platform/issues/2497)、幅広いCorpusは[#2501](https://github.com/hat47x/tei-platform/issues/2501)。
- **既存契約を再利用**：#1120（Typed Subject/Identity）、#1806（Semantic Type/TypeRef）、#1126（Relation/Key）、#988（Operator契約）、#1705（Quantity/Unit/丸め）、#1706（集合/完全性）、#1063（演算と部分データ）、#2514（独立Oracle）、#951（Plugin Conformance）、#1155/#2354（CUI）、#1124/#997（UI）、#2493（広域並列）。
- **前提**：[メタ概念空間](DATA_STRUCTURE_META_FACETS_AND_TOPOLOGY_2026-11.md)のM1〜M10は探索用チェックリスト、7役割は混同を避ける概念上の整理。新しい公開型階層や第二の正本は導入しない。

## 1. 新しい問題：一つのCarrierが同じ構造を複数持てる

同じ型・値集合に対して、**同一名称の数学的構造が複数成立する**。これを区別しないと、「多重親」を許しただけで演算や制約が混ざり、Coreの固定型を増やしたときと同じ重複・不整合を招く。

例：自然数の集合`ℕ`には、少なくとも次の二つの異なるモノイド構造がある。

| Carrier | 二項演算 | 単位元 | 法則 | 結論 |
| --- | --- | --- | --- | --- |
| 自然数 `ℕ` | `addition`（+） | `0` | 結合律と左右単位元 | 加法モノイド |
| 自然数 `ℕ` | `multiplication`（×） | `1` | 結合律と左右単位元 | 乗法モノイド |

単に`NaturalNumber has-facet Monoid`と記録しても、`combine`で加算するか乗算するかは決まらない。`Monoid`は**Carrier＋演算・単位元の束縛＋公理**として初めて意味のある主張になる。

同様に、自然数の通常の大小比較と整除関係は異なる順序を与える。集合に離散位相／密着位相を導入することもできる。**同じ対象集合が同じ種類の構造を複数持つ**ことを、二つのSemantic Type IDへ無理に分裂させる必要はない。ただし構造に伴う意味上の別名／適用文脈・Sourceは維持する。

## 2. 最小の論理モデル：TheoryとInstanceを区別する

比較時に限り、構造を次のように捉える。

```text
Structure Theory (e.g. Monoid)
  carrier variables: T
  operation signature:
    combine : T × T → T
    unit    : T
  law obligations:
    associativity / left identity / right identity
           |
           | explicit binding (typed and scoped)
           v
Structure Instance A on ℕ
  combine = addition; unit = 0; law evidence A

Structure Instance B on ℕ
  combine = multiplication; unit = 1; law evidence B
           |
           | used at an explicit operation/use-site
           v
AI CUI / DSL / UI / Target Conformance
```

ここで「Theory」は演算が満たす法則を指定する抽象概念、「Instance」は具体的な演算・Carrier・根拠の組を指す検討用語である。既存TEIの`Instance`や`Type`等の固有クラスを上書きするものではない。原則として現行のTyped Reference、Operator、Semantic Type、Extension/Plugin契約を束ねた**read-onlyな照合結果**として比較する。

**構造の記述上、少なくとも異なる役割**：
1. **Carrier/対象**：型・対象集合・Identity・値域・Namespaceと版。
2. **Signature/構成**：演算（型付き引数・結果）・定数・関係、全域／部分、必要な外部Context。
3. **Law/義務**：結合律・交換律・逆元・反射性・推移性・開集合族の公理等、対象と量化Scope。
4. **Binding/解釈**：どの具体的な演算・単位元・比較関係・位相族にLawを解釈するか。演算ID・Source版・Contextで識別。
5. **Use-site/観測**：どの操作が何を要求し、観測者は何を確認できるか。
6. **Evidence/資格**：宣言、形式導出、境界付き探索、独立Fixture、実Target観測、Ownerによる意味受理をそれぞれ記録。

この記述は[#2495](https://github.com/hat47x/tei-platform/issues/2495)の7役割を具体的な数学的Interfaceに適用する例であって、新しい固定6種のCore Kindではない。

## 3. 多重親・多重Facetの合成をどう評価するか

### 3.1 SubtypeとHas-Facetを同じ推論規則にしない

- **is-a/refines**：対象の意味と制約の包含を主張する。適用Scope・名目Identity・操作の置換可能性は独立に確認する。
- **has-facet/satisfies**：必要な演算・制約を束縛し、当該構造の法則を満たすという主張。ひとつのCarrierに同じTheoryの異なるInstanceを複数持てる。
- **uses-operation**：あるUse-siteがどの演算束縛を選ぶか。二つのMonoidがあるなら明示選択が必要。
- **converts-to/preserves(P)**：構造間の写像と、指定したLaw・観測が保存されるかを#2497で比較。
- **encoded-as/projected-as**：Go/Java/SQL/Arrow、Graph/表/SHOなどへの実現・表示。演算が同名であることから実Targetの性能・精度・法則の適合を推測しない。

これにより`Ordered`、`Quantified`、`Temporal`、`Geospatial`を併せ持つ一つの業務系列を、4つの独立したCore実体へ複製せずに説明できる。通常のPlugin開発者には高次のメタ概念を手入力させず、標準Type/Relation/Operation契約から導出可能な範囲だけをCUI/Projectionで利用する。

### 3.2 Diamond構成で起きる三種類の問題

| 問題 | 例 | 提案する扱い |
| --- | --- | --- |
| **同名異義** | 親Aの`combine=+`と親Bの`combine=×` | 名前ではなく、演算identityと束縛の解釈に従う。選択先がないときは曖昧 |
| **整合しない制約** | 一つの値に`x>0`と`x<0` | 対象Scopeで矛盾を導出できる場合は充足不能。取得レコード0件と区別 |
| **同値だが異なる経路** | A→B→DとA→C→Dの二経路で参照・演算束縛が異なる | 演算とSourceに基づき同一性を検査。関係経路が2本あることを単純にエラーにしない |

Lean mathlibでは代数・順序・位相の構造を組み合わせる際に型クラスの増殖や継承Diamondが設計上の課題となっている。TEIの多重Facetはプログラムの多重継承として実装する必要がないが、**意味上の演算選択と法則の一貫性**という同種の問題は残る（[mathlib hierarchy design](https://leanprover-community.github.io/mathlib4_docs/Mathlib/Algebra/HierarchyDesign)）。

## 4. 法則が「成立する」と言うための根拠の階層

以下は一つの単調な`PASS`レベルではなく、**異なる証拠の種類**。

| 主張 | 根拠・検査例 | 限界 |
| --- | --- | --- |
| 文法・型が整う | 宣言の参照と演算の引数/結果の対応 | 型が通ってもLawは証明されない |
| 制約の整合を調べた | CUE的unification、対象断片のSolver、Alloyなどで充足例・反例を探索 | 有限探索で反例がないことは全域の証明ではない |
| Lawを宣言した | `associative`を設計契約に記載 | 宣言だけで演算実装が従うわけではない |
| Formal Proofを得た | Lean等で選択した前提と演算に対して定理を証明 | 証明した数学的演算と対象Runtimeの計算が同じとは別途照合が必要 |
| 独立Fixtureで検査した | 法則から導いた正負例と意図的な誤実装比較 | 代表入力だけの検査、独立Oracle品質は#2514 |
| 実Targetで観測した | 固定Artifact、実行条件、Sourceと独立readでConformanceを観測 | 該当Target・時点・入力に限定。業務意味のOwner受理は別 |
| Domain Meaningが受理された | 権限者がSource・解釈・契約の対象/時点を受理 | 実Runtime/権限の保証とは別 |

特に、**制約同士が充足不能である場合と、保存済みの実例が0件である場合**を別に表示する。数学的に空集合を認める抽象構造と、充足可能性を要求する具体的業務型も区別する。全称命題は、対象集合が空の場合に形式上は真となることがあり、業務上の「必要な存在」の証拠にはならない。

- CUEの値・制約のunificationは順序に依存しない結合を実現する参考。TEIの業務意味や実ValueをCUEと同一の値束へ統合する提案ではない（[CUE Spec](https://cue.dev/docs/reference/spec/)）。
- Alloyは関係論理を用いた構造例・反例の探索に適しているが、**有限Scopeで「反例が見つからない」ことを全域証明と呼ばない**（[Alloy specification](https://alloytools.org/spec.html)）。
- Leanの型クラスは演算と公理を含む構造の表現例。TEIの意味型登録時にLeanの証明を必須とするものではない（[Lean Type Classes](https://lean-lang.org/doc/reference/latest/Type-Classes/)）。

### 4.1 検証の結果の区別

`satisfied`、`violated`、`unresolved`、`unsupported`、`not-observed`などは**概念上の区別**であり、いきなりTEIの新しい共有enumにする提案ではない。`unsatisfiable`は対象とLogicの範囲で根拠がある場合、`no-witness-in-bound`は指定した有限探索で成立例が見つからなかった場合などと説明を分ける。実データが空であるという観測も別。

## 5. 反証可能なPressure Cases（新しい実装・正本は不要）

| ID | 正例 | 負例 | 主な既存Consumer |
| --- | --- | --- | --- |
| LA-01 | 同じ`ℕ`を、`Monoid(+;0)`と`Monoid(×;1)`で異なる演算束縛として表示 | `has-facet Monoid`だけで演算を自動決定 | #2495/#988 |
| LA-02 | 同じ対象集合の通常の順序と整除順序を別構造として扱う | 同じ`Ordered`分類名から比較結果を決める | #2495/#1126 |
| LA-03 | 位相空間`(X,τ)`とGraph隣接の双方を表示し、前提が異なる | edgeがあるだけで数学的位相公理成立と推測 | #2495/#997 |
| LA-04 | 多重親の演算名衝突を明示したUse-siteで解決 | 継承親の列挙順による`combine`の暗黙選択 | #2495/#951 |
| LA-05 | `asserted` Lawと`proved` Lawを固定版・前提付きで区別 | 数件のfixture PASSを形式証明へ昇格 | #2514/#1063 |
| LA-06 | `x>0∧x<0`の矛盾を空の実DBと区別 | 0件だから矛盾なし、有限探索無反例だから全域で真とする | #1806/#2495 |
| LA-07 | 温度点／差、通貨／時点、丸めと演算束縛を区別 | `Decimal+Decimal`で自動加算・換算 | #1705/#2077 |
| LA-08 | 関数値、高階演算、無限Stream等の構造記述と観測可能な範囲を分ける | 先頭3件が取れたので全無限系列を取得済みと断定 | #1706/#2495 |

追加で、**空集合に対する全称制約**、**同じUnitで別のQuantity Kind**、**同じ型で異なるSource版**、**構造法則が受理されてもHost Capabilityが未解決**などを組み合わせると、型と法則の証拠混同を発見しやすい。

## 6. AI向けCUI・UI投影に効く具体的な情報

必要なら、CUIのread-onlyな概念説明を、次のような構成で比較できる（現行TEIのJSON Schemaではない）。

```yaml
subject: sample.temperatures
carrier: TemperatureObservation
structure_claims:
  - theory: TemporalOrder
    operation_binding: sample.measured-at-order
    evidence: declared
  - theory: QuantityPoint
    operation_binding: sample.temperature-point-and-difference
    evidence: proposal
use_site:
  operation: chart-series
  requires: [measurement-time-order, quantity-unit]
  completeness: unknown
unresolved:
  - observation-window-completeness
```

CUIで、構造が何に適合するかと、その適合が**名前だけの主張なのか、演算束縛・Lawの根拠があるのか**を区別できる。UI標準プラグインは、このread-only情報から利用可能な図表・可読化の候補を表示できるが、表現可能なFacetがあるだけで値編集・数学的証明・本番Authorityを得るわけではない。

**通常のプラグイン開発者がこのメタ概念をすべて理解し宣言する必要はない。** 汎用の標準構造・契約と専門Pluginの意図から、必要な構造的意味を共通UI/AI Toolが後から取得できる方向を検討する。専門数学・単位系Pluginだけが演算の細かな法則や独立Conformanceを追加できればよい。

## 7. 採否と、他のレーンへの引継ぎ

1. #2495：同じCarrier上に複数の構造Instanceがあるケースを、既存の概念関係で説明できるか。必要な追加の関係・Signatureは何か。
2. #988/#1705：演算束縛の名目的ID/型/単位/丸めをどこまで既存Operatorで読めるか。数学的適合の宣言を実行許可にしない。
3. #2497：保存写像がどのOperator/Lawを保存するかを用途相対に扱う。「同じMonoid名」だけからHomomorphismと推定しない。
4. #2514：有限fixture/独立Oracleと形式証明を分離し、意図的に誤った演算を検出する判別力を比較。
5. #2501/#2493：広いDomain/Target/演算候補を追加し、上記LA番号を既存Corpusのケースへリンク。型Kindの増殖ではなく説明・矛盾検出能力を測定。
6. #1155/#2354/#1124/#997：CUI・Studioでは対象・構造・演算・使用Context・証拠・不明点を同一版で提示する。通常開発者の入力量やUI負担を増やさず、AIによる変更候補と実業務の受理を分離する。

**最小の次段実証**：LA-01・LA-04・LA-06について、既存Semantic Type/Operator/Validationを読み取ったときの説明可能性を机上照合し、read-onlyで表現できない情報だけをOwnerへ返す。これと並行してLA-02/03/07/08を研究Corpusで拡張可能とする。数学全体の証明器、固定Facet enum、一般的な構造分類Runtime、第二Type Registry、TEI独自Hostを先行要件にしない。

**今回の実際の範囲**：CUE/Alloy/Leanの公開仕様と既存GitHub設計を参照した文書・Issue上の提案。Go/CLI/数学Solver/Lean証明/GUI/Host/実DBはNOT RUN。形式的な法則の採用・Owner意味受理も未実施。
