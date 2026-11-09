# TEI基本データ概念空間：構造トポロジー・多重分類・数学・単位の横断設計（2026-11）

- **状態**：Ownerの設計方向を受けた広域の設計探索・机上評価案。基本メタモデルの最終採用、全構造の網羅保証、Core/Pluginへの実装、動作の実測ではない。
- **目的**：将来の可視化・可読化・編集とAI-first CUI開発の共通の土台となるデータ構造を、プログラミング言語の型・コンテナ・保存形式の分類より高い抽象度から整理する。後から異なる分類軸を同列に追加して、重複する意味の正本と不整合な機能体系が増えることを防ぐ。
- **基本方針**：[開発全体計画](../MASTER_PLAN.md)、[外部要件の複合圧力](AI_FIRST_REQUIREMENT_PRESSURE_AND_HEADLESS_VERIFICATION_2026-11.md)。
- **関連する既存判断**：[RQ-001/003/008/015/026/031〜034](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)、[製品原則P16/P17/P21](../../product/principles/TEI_PRODUCT_PRINCIPLES.md)、[ADR-0032](../adr/0032-composition-kernel-replacement.md)、[ADR-0033](../adr/0033-core-and-plugin-subprojects.md)。
- **既存担当**：#1120（意味上のIdentity・参照）、#1126（Data・Relation）、#1806（意味型再編）、#951（Plugin Contract）、#1121（Target/Lowering）、#1124（後続GUI）、#1973（PoCの概念評価）、#1119（横断）。
- **研究上の基準**：[ISO/IEC 11404:2007](https://www.iso.org/standard/39479.html)（概念・構造・実装の型区別。2026-11-01時点で次版FDIS審議中）、[OWL 2 Primer](https://www.w3.org/TR/owl-primer/)（クラス共通部分・和・排他性）、[SHACL Recommendation](https://www.w3.org/TR/shacl/)（構造制約・データ検証）、[SHACL 1.2 Working Draft 2026-10-12](https://www.w3.org/TR/2026/WD-shacl12-core-20261012/)（検討中）、[QUDT](https://data.qudt.org/)（QuantityKind/Unit/DimensionVector）、[BIPM SI Brochure](https://www.bipm.org/en/publications/si-brochure/)（単位系）。仕様をそのままTEIの採用仕様とみなさない。

## 1. 核心：MECEにするのは「分類対象」より先に「分類する問い」

具体的な構造は本質的に重なり得る。Treeは適切なGraphの特殊例、Matrixは添字付きデータであり線形写像の表示にもなる、Vectorは数学的なベクトルとプログラミング上の可変順序列という別の意味を持つ。VectorとHashTableも、前者は順序列・後者はキー対応を中心とする抽象／実装であり、単に同居することが誤りなのではない。**異なる抽象レベルや分類軸を、一つの競合する型階層に置くこと**が問題である。

このため、(a)何の意味を扱うか、(b)どんな構造的性質・関係を持つか、(c)どんな演算・法則が成立するか、(d)何の値域・数量なのか、(e)どう実装・表示するか、を別々の「問い」として定義する。各問いの回答が互いに独立とは限らないため、組合せの制約と矛盾を検証する。**メタレベルの語彙・関係種別は重複を防ぐ責務を持つが、具体的概念は複数の分類に同時に属してよい。**

「世界に考え得るすべての構造」は、任意の述語や再帰・新しい演算から無限に拡張できるため有限個の型名として列挙しきれない。代わりに、構成法・分類軸・合成規則・拡張規約を網羅的に探索し、収集した外部要件集合についてどの意味を記述できるか全件照合する。未知や反例は未解決として残す。

## 2. 提案する多層構造：プログラミング言語の型の上に「構造形式」を置く

「トポロジー」は少なくとも**構造形式（structure topology）**、**接続・隣接のトポロジー**、**数学的な位相空間のtopology**を区別する。以下の最上位は数学の位相空間だけを指すものではなく、対象間の構成・接続・順序・添字・関係を表す**構造形式の概念空間**である。

```text
Structure / Topology Concept Space  （構造形式・関係・演算の概念）
   ├─ Structural Properties        （順序、キー、重複、次元、関係、接続…）
   ├─ Mathematical Structures      （群、順序、ベクトル空間、位相空間…）
   ├─ Quantity / Units             （量種、次元、単位、尺度…）
   └─ Constraints & Morphisms      （法則、適合、構造保存変換…）
                    │ 複数の分類・関係で記述
                    ▼
Semantic Shape / Data Concept      （業務・科学・社会の概念、Identity、履歴）
                    │ インスタンス化・Refinement
                    ▼
Logical Value / Type / Schema      （値域、積・和・参照、キー・レコード等）
                    │ Binding / Lowering
                    ▼
Physical Realization                （Go/Java/.NET型、SQL、Columnar、JSON…）
                    │ Projection / Interaction
                    ▼
UI / Document / Visualization      （表、グラフ、地図、数式、図、ガント等）
```

これは責務と抽象度を示す図であって**必ず上から下へ一対一に生成する直列パイプラインではない**。SemanticとStructureは相互参照し、構造制約を満たす型・対象・実装を選べる。Primitive値も必ず高度な数学構造を宣言する必要はない。表現可能であることとTargetで実現可能であることは区別する。

## 3. 単一継承木ではなく、複数親を許す「種類付き概念グラフ」

「親」を一つの`extends`へ押し込めず、関係を区別することを提案する。

| 関係の意味 | 説明 | 例・検証する法則 |
| --- | --- | --- |
| **is-a / subtype** | すべての実例が上位概念の意味を満たす | 有向木は有向グラフの制約付き特殊例。置換可能性を検証 |
| **satisfies / conforms-to** | 複数の構造的性質・法則へ適合 | Indexed、Ordered、Measurableなどを同時に満たす |
| **has-part / component-of** | 構成・包含、所有と共有を含む | SeriesがData Pointを持つ。is-aとは別 |
| **related-by / morphism** | 型付きRelation・接続・変換 | Graphの辺、行列の転置、変換の可逆性 |
| **refines / constrained-by** | 概念に追加制約を課す | 非負、非NULL、履歴・時点制約、有限・連結 |
| **realized-by** | 物理実装・配備・言語表現との対応 | Map→HashTable、Sequence→ArrayList |
| **projected-as** | 可読化・UI・ドキュメントへの投影 | Graph→依存図、Series→時系列グラフ |

複数親は、**同じ対象が複数の契約や数学的法則を満たす**という意味で積極的に認める。例えば`SpatialTimeSeries`は`IndexedStructure`、`TemporalStructure`、`SpatiallyReferenced`、`Quantified`の四条件を同時に満たす候補である。OWL 2のクラスintersectionは比較可能な発想だが、OWL推論の開世界仮定やオントロジー自体の採用を意味しない。TEIの実行時Validationは、明示された閉じた入力・契約を必要に応じて検査する。

**多重継承する「クラス実装」そのものは要求しない。** メソッド・フィールドの多重継承（Diamond問題）より、法則・構造Property・Constraintの多重適合と関係グラフを基本に考える。共通部分での属性型、単位、演算、同値性、Identityの衝突を明示的に検出する。例えば`Ordered`と`Unordered`は常に排他ではない（あるviewだけ順序を持つことがある）。排他性は対象Scopeと両者の定義に即して宣言し、名称だけで推測しない。

概念関係の`is-a`は反射的・推移的な部分順序として整理する候補だが、同値な概念の別名、バージョン違い、同型・相互変換を`is-a`の循環で表さない。必要なときは正規IDと等価性を明示する。多重分類した実例を複数の別データとして複製しない。

## 4. 構造を探索する独立した問い（メタレベルの候補）

| 問い／Facet | 代表例・選択肢 | 失われやすい意味 |
| --- | --- | --- |
| 個体と同一性 | 値、実体、事象、Identity、Version、参照 | 同じ値と同一個体の混同 |
| 構成・多重度 | 原子、Product、Sum、集合、Bag、Sequence、Multimap | 順序・重複・欠損の消失 |
| 関係と接続 | n項Relation、Hypergraph、Graph、Tree、DAG | n項関係を二項化したときの意味損失 |
| 順序・比較 | 全順序、半順序、順序なし、比較不能 | UI表示順と意味上の順序の混同 |
| 索引・次元 | 有限／無限、ラベル付き、添字集合、Tensor、Ragged | indexと物理位置、0と未観測の混同 |
| 変化・時間 | 静的、Event、Stream、Interval、Valid/Transaction Time | 時刻と継続時間、時間軸の混同 |
| 代数・計算 | 演算、Closure、Group/Ring/Field、Vector Space | 配列と数学的ベクトルの混同 |
| 計量・位相 | 距離、近傍、連続、境界、連結、多様体 | GISの接続構造と数学的位相の混同 |
| 数量・尺度 | QuantityKind、次元、単位、比率、絶対／差、対数 | 同じDecimalなら加算できるとの誤解 |
| 不確実性・知識 | 欠測、未知、確率、予測、観測、受理、出所 | 未確認をNULL/ゼロ/真へ圧縮 |
| 演算・更新可能性 | 読取り、挿入、逆変換、競合、副作用、権限 | 集計・射影結果へ無条件に逆更新 |
| 実現・表示 | SQL/JSON/Arrow/Java/Wasm、図/表/数式/地図 | 配備先・画面部品を意味に昇格 |

個々のFacetには定義域・適用Scope・制約・依存を持たせ、矛盾の組合せや未解決を診断する。Facet間に演算の前提があるため、完全に独立した単純なフラグ集合にはしない。

## 5. 数学と単位系の位置付け

### 数学的構造

Set、Relation、Function、Order、Algebraic Structure、Vector Space、Metric Space、Topological Space、Measure/Probability Space等を、単一の`datatype`列挙としてではなく、**構造に与える演算・法則・制約**として比較する。例えばVector Spaceの元はScalar Field・加法・スカラー倍・公理を持つ。ArrayやTensorという物理表現だけでVector Space成立を推測しない。

構造保存写像（Morphism）は、同型、部分写像、単射/全射、情報損失を伴うProjection等に分け、何を保存するかを契約で明記する。圏論的なモデル（SchemaとInstance、関手による写像など）は将来の高度な検証・移行の比較対象として評価し、PoCのCoreへ全面導入することを前提にしない。

### 単位と数量

`QuantityValue`を数値単体とせず、**Quantity Kind、Dimension Vector、Unit、Scale/Reference、精度・不確かさ、Conversion policy**を分ける。物理次元が同じでも、角度／割合等の量種・演算規則は同じとは限らない。

温度の絶対値と温度差（アフィン量と差のベクトル）、対数単位、通貨の換算時点、暦時刻とDurationなどを反例とする。Quantityの型付き演算は数学構造の前提に従い、単位換算の前後で維持すべき不変量を示す。BIPM/QUDT等の外部語彙は、TEI内部の意味型とその出典を対応付ける候補であり、数値型と独立に所有する。

## 6. 典型的な多重適合の試金石

**例A：地理参照された時系列の温度観測**

```text
対象：TemperatureObservationSeries
  satisfies: IndexedStructure
  satisfies: TemporalStructure
  satisfies: SpatiallyReferenced
  satisfies: QuantityBearing
  constraint: 値のQuantityKind=Temperature（絶対）
  constraint: 時間軸はInstant、座標参照系を明示
  representation candidate: Arrow / SQL / JSON / columnar
  projection candidates: 時系列・地図・表
```

これは**提案を説明する概念例であり、現行TEIの実行可能なDSLではない**。四つの親を持つ意味は四種類のデータ実体を作ることではなく、一つのSubject/データ構造に四つの適合契約があること。地図と折れ線へ投影する際も、同じ意味上のIdentityとSourceへ戻れるようにする。

**例B：組織構造**

`OrganizationStructure`が`DirectedRelation`、`TemporalRelation`、`RoleBearing`、`DelegationAware`を満たす場合、組織図では一つの木に見えても、兼務・複数上長・代理・時点を表すデータはDAGまたはより一般の関係として残せる。木Projectionの見た目を正本として採用しない。

**例C：集計テーブル**

`Table`はRecord Collection・Relation・Query Resultなどの性質を持ち得る。ただし`Bag`意味とSQL NULL、Aggregation、group key、更新の逆写像は個別に検査する。DataGridで編集可能なことを型の自明な属性にしない。

## 7. 同一意味の重複・矛盾を防ぐ採用規則（設計候補）

新しい「基本データ構造／型」を追加する前に、次を机上で確認する。

1. **分類する問い**を特定し、構造Property、数学的法則、値域、意味上の対象、実現方式、Projectionのどのレベルに追加するのか説明する。
2. 既存のFacet・Relation・演算・Refinement・Pluginで**合成**できるか調べる。名称が違うだけの既存概念なら別の正本を作らずAlias／Profileで表す。
3. 複数の親に`is-a`を使う場合は、**置換可能性と親の約束の交差**を確認する。単に性質を持つ場合は`satisfies`等の適切な関係を使用する。
4. 同値性・包含・構造保存・可逆変換・情報損失の区別を置き、交換可能性を明示する。矛盾・競合は診断できる形にする。
5. 既存構成で扱えない**具体的な反例**と、追加構造を導入したときの新しい実現可能性・検証・撤回条件を示す（#1973）。
6. Core抽象、標準Semantic Plugin、分野別Plugin、Provider、ProjectionのOwnerへ適切に配置する。最小のCoreへ数学全体を固定しない。

分類を将来修正する場合も、古い概念ID、使用中のデータ、対応するExtension、変換、既存の適合試験を追跡し、同一意味の二重正本を恒久化しない。

## 8. 外部要件に対する机上検査集合（候補／未実施）

| ID | 具体的な題材 | 確認すること |
| --- | --- | --- |
| T01 | Sequence・Set・Bag・Map・OrderedMap・MultiMap・Queue | 順序、重複、キー、アクセス方法、Implementationの区別 |
| T02 | Table・Property Graph・RDF Graph・Hypergraph | 次数、方向、ラベル、多重辺、NULL、変換時の損失 |
| T03 | 再帰木・DAG・循環Graph・共有参照 | 再帰値と参照同一性、循環、Tree制約 |
| T04 | Sparse/Dense Tensor・Ragged Array・ラベル付きMatrix | 添字・次元・疎性・欠測・0と未定義 |
| T05 | Value Domain、Refinement、Unit、Quantity Kind、Dimension | 量種、単位変換、絶対/差、尺度、丸め |
| T06 | Metric・Topological Space・GIS geometry/CRS | 構造的隣接と数学的位相、幾何の境界 |
| T07 | Group/Field/Vector Space、Matrix、測定データ | 演算法則、Scalar Field、数学的VectorとCollection |
| T08 | 実体の時点付き多重所属、権限の代理・委任 | Directed Relation、時点、兼務、表示上の木との違い |
| T09 | 日時・期間・イベント・Stream・因果 | Valid/Transaction/Event/Processing time、順序・欠落 |
| T10 | 確率変数・推定・観測値・未知・Null | ValueとEpistemic state、統計的演算、正本 |
| T11 | 音声・動画・文書・3D/Geometry・混合媒体 | 時間・媒体・局所参照・バージョン、投影と実体 |
| T12 | 同一意味のSQL/JSON/Arrow/Go/Java/Wasm表示・生成 | 構造保存、変換の可逆性、範囲、実行契約 |

各題材で、構造的な説明・複数分類の衝突有無・必要なExtension・保存する不変量・損失の候補を記録する。**適合表全件の机上レビューを基盤の広域設計へ先行**させる。実装/Targetでの成否とOwnerの意味受理は既存の担当に分離し、試験に成功する前から「完全網羅」としない。

## 9. 未決論点と次の差分

- この概念空間を`Structure`/ `Topology`/ `Shape`等のどの名称で公開するか。数学的Topologyとの同名混乱を考慮する。
- 正規形をもつSemantic Shape Descriptorを定義するか、既存のSemantic Extension/Typed Reference/Constraintで十分か。
- `is-a`の部分順序と`satisfies`の多重適合をどう整合判定し、適合の証明・未証明を分けるか。
- Valueの欠損・未定義と、認識上の未解決をどこで保持するか。
- Quantity/Unit・Math DSL・Proof/Lean・Spatial CRS等の外部Ontology/Libraryの交換契約。
- 多重分類からUIが複数のVisualization候補を得た際、更新可能性・権限・Projection選択をどのように診断するか。

次の着手は、新しいCore enumを追加することではなく、**T01〜T12の机上圧力に対し、既存のSemantic/Plugin Contractでどの意味を表せるかと、最小の反例は何かを比較すること**である。現状の証拠・受理状態は`docs/status.md`と各Issueに従う。
