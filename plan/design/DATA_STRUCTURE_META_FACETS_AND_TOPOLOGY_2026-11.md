# データ構造のメタ概念空間：数学・量・トポロジーと意味型を直交して扱う（設計案）

- **状態**：2026-11-01、構想・調査・反証可能な試験設計。**未採用／実装・検証未実施**
- **Owner**：[Issue #2495](https://github.com/hat47x/tei-platform/issues/2495)
- **関連**：[#1806](https://github.com/hat47x/tei-platform/issues/1806)、[#1120](https://github.com/hat47x/tei-platform/issues/1120)、[#1126](https://github.com/hat47x/tei-platform/issues/1126)、[#1127](https://github.com/hat47x/tei-platform/issues/1127)、[#1705](https://github.com/hat47x/tei-platform/issues/1705)、[#1706](https://github.com/hat47x/tei-platform/issues/1706)、[#1570](https://github.com/hat47x/tei-platform/issues/1570)、[#2387](https://github.com/hat47x/tei-platform/issues/2387)、[#2493](https://github.com/hat47x/tei-platform/issues/2493)
- **目的**：数学・一般情報構造・業務モデル・UI・データ構造を一つの固定された言語の型階層へ押し込まず、**追加可能な概念の軸**で広く説明する。個別の構造が複数の軸に同時所属できる一方、軸同士の概念上の責務を明確にする。
- **PoC価値仮説**：新しく見つかったデータ構造（例：時系列グラフ、座標付きテンソル、多重集合を返す検索結果）を導入するとき、既存のCore型Kindを分裂・重複させるより、構造と制約と実現の組合せとして追加でき、AI CUI・Studio・生成物で同じ意味境界を提示できる。

## 1. 「MECEにしたい」の正確な対象

**MECEに整理する対象は、個々の全データ構造を互いに排他的な箱へ分けることではなく、ある対象を理解するために問うメタレベルの関心事・責務の区分**とする。実際の構造はその軸の複数の性質を併せ持つ。たとえば「時刻で索引された重複可能なイベント系列」は、sequence／temporal index／event occurrence／partial observationを同時に持ち得る。

メタ概念空間の網羅性は、「全宇宙の将来の数学構造を列挙した」と宣言して保証するものではない。未知の構造が来た際に、**どの問い・どの新しい構成要素・どの関係を追加するかが一義的に説明でき、既存の意味を壊さないこと**を拡張可能性の判定とする。

特に次の語を区別する。
- **Topology / 数学的位相**：開集合・連続性・近傍などの数学的構造。必要なら距離や測度と独立に付与する。
- **Graph topology / 接続構造**：頂点・辺・接続関係・向き・連結性・閉路など。TreeやDAGはGraphの制約付き構造として説明可能。
- **Data shape / データの構成形**：record、list、map、relationなどの組み立て。
- **Semantic Type / 意味型**：値の名目的意味・型ID・使用文脈。Domain（対象領域）・Value Domain（値域）と別の候補語。
- **Physical representation / 実現**：SQL・JSON・Java Library・Wasm・UI等での符号化。上の概念の名称と同じ単語でも同一視しない。

## 2. メタレベルの分類軸（互いに異なる問いを担当する）

| 軸 | 問い・責務 | 特性・演算子の例 | 先行Owner |
| --- | --- | --- | --- |
| M1 指示・同一性 | **何を**識別・参照しているか | 値／Entity／Occurrence／Relation／Event／Observation、名目ID、Scope、issuer | #1120 / #1126 / #1806 |
| M2 構成文法 | **何から**構築されるか | atom、product(record/tuple)、sum(variant)、optional、reference、collection、function、relation、recursive、opaque | #1806 / #1127 |
| M3 集合・索引 | **どう列挙・選択・重複表現**するか | set／bag／sequence、index key、key uniqueness、order、partial map、finite/lazy/stream | #1706 / #1127 |
| M4 関係・接続 | **要素同士がどう関わるか** | 二項／多項／hyperedge、方向、cardinality、reachability、tree/DAG/cycle、所有／参照 | #1126 / #2387 |
| M5 数学的構造 | **どの法則を満たすか** | order、group/monoid、vector space、matrix/tensor、metric/topological space、measure/probability | #988 / #1705 |
| M6 量・座標 | **何をどの基準で測ったか** | dimension、unit、absolute vs difference、currency、coordinate reference/frame、precision、rounding | #1705 / #2077 |
| M7 時間と変化 | **いつ・どの版の状態か** | instant／interval／duration、event time／valid time／transaction time、state transition、revision | #2386 / #1174 / #1131 |
| M8 値の適用条件 | **どの値・組が許されるか** | Value Domain predicate、range、format、cross-field invariants、nullable/required/use-site constraint | #1806 / #2056 / #1129 |
| M9 情報・観測 | **何を知り得たか** | absent／null／empty／unknown、partial/complete、row/column provenance、source snapshot、authority scope | #1706 / #958 / #1131 |
| M10 実現・提示 | **どの成果物・表示へ転写するか** | RDB/JSON/columnar/Java/.NET/Wasm、table/form/chart/graph/Gantt/map/SHO、capability adapter | #1124 / #997 / #2077 / #1803 |

この表は実装するCore enumの提案ではない。M1〜M10は**責務を比較するための暫定的な設計軸**であり、相互依存（例：Graphは集合・関係の要素も持つ）は意図的に保持する。役割が同一の軸が見つかれば統合し、必要な問いが不足すれば軸を追加する。

## 3. 記述の正規形（論理的な案。新Wire Schemaではない）

任意の対象について、次の**別々の主張**をつなげて記述できるかを調べる。

1. **Identity claim**：意味型ID、概念／値／instanceの区別、所有するNamespaceと版。
2. **Constructor claim**：その構造をどう組み立てたか（Product、Sum、Collection、Relation、Map、Function等。定義は拡張可能）。
3. **Facet claim**：順序、重複、key/多項関係、数学上の性質、単位、時間軸、幾何、確率などの適用可能な**特性**。
4. **Constraint claim**：制約の対象（Type全体／使用箇所／Operation／dataset）、根拠・適用範囲・既知/不明・版。
5. **Mapping claim**：表示や物理格納への転写、および損失・不確定・変換の明示。
6. **Evidence claim**：誰の意味受理か、形式的導出／合成fixture／実Target観測のどれか、取得Scope、固定版。

同じフィールドにこれらを押し込まず、各担当の既存契約を利用する。最初は設計文書／read-only評価データで表現し、採用前から`internal/semantic.Type`やShared Acceptanceの公開schemaを増設しない。

### 構造上の関係を区別する（多重親を許容）

| 関係 | 意味 | 複数関係を持つときの注意 |
| --- | --- | --- |
| `is-a`／`refines` | 値集合の包含、または明示された意味上の特殊化 | それぞれの制約に矛盾がないか調べる。名目型の同一性とは区別 |
| `has-facet` | 数学的・数量的・順序などの性質を持つ | 複数の上位概念に関連付けても、プログラミング言語の多重継承や自動代入にはならない |
| `composed-of` | 積／和／構成要素 | 所有、Identity、Referenceを同じ合成として扱わない |
| `models`／`represents` | 関係・Event・Observation等の意味対象を表す | 表示された同名の項目から意味同一性を推測しない |
| `encoded-as` | 物理型・成果物へ実現する | 逆変換の損失・版・Target条件を保持 |
| `convertible-to` | 明示した変換契約の下で値を変換できる | 可逆性・丸め・換算時点・換算元証拠を区別 |
| `usable-as` | ある操作・文脈での適合（例：比較・加算・join） | 同じ基底型というだけで全操作を許可しない |

**OWL 2のSubclass / Equivalent / Disjoint / DisjointUnionなどの表現は、概念関係を考える際の参考**になるが、TEIがOWL 2の推論器やopen-world意味論をそのまま採用する案ではない。**JSON SchemaのallOf / anyOf / oneOf**も値の適合性を表す参考であり、Shared Acceptanceの人による業務意味受理とは独立である。

## 4. 値・数値・物理構造を混同しやすい反例

### 4.1 Vector / Hashtable（名前ではなく構造と操作の区別）

- Java等の`Vector`は一実装、数学上のvectorは演算法則を持つ対象。前者を持つことから後者の加法／スカラー積を推論しない。
- `Hashtable`はKey→Valueの格納方式として知られるが、mapの意味上のkey一意性、順序、重複、更新整合は独立に宣言する。順序付きmap、multi-map、hash indexを持つsequenceを表現できるべき。
- 「Vectorの代わりにHashtableを追加した」という実装選択だけを新しい意味型Kind追加の理由としない。

### 4.2 数学的・量の反例

- **絶対温度と差**：20°Cを別の絶対温度へ変換する際はオフセットが必要だが、20°Cの温度差は同じ係数でKの温度差へ換算可能。値が20で表記が°Cでも演算契約は同じでない。
- **JPYとUSD**：どちらも通貨であるが、`JPY+USD`には時点・出所付きのレート等の明示変換が必要。通貨の同じ次元・文字列一致を足し算可能性へ昇格させない。
- **地理座標**：2つの数値があるだけでは緯度・経度でも平面座標でもない。座標参照系・順序・datum等が不明なら距離計算や地図表示の正しさは未確認。
- **数学的vectorと2D配列**：`float[3]`が3次元空間ベクトルの意味を持つとは限らない。vector空間・tensor・matricesと行列状JSON／SQL配列の関係を分類する。
- **確率分布**：`map[Outcome]float`という形だけでは非負性・正規化・標本空間・確率測度が成立したと認定しない。

### 4.3 関係・系列・実行境界の反例

- Treeは有向GraphやRelation集合の**制約付きケース**として表現できることがある。一方で、循環するGraph／多項Hyperedge／複数親の階層を、単一親Treeの固定UIへ押し込まない。
- `set`／`bag`／`sequence`／`stream`は、要素順序、重複、時点、部分性が異なる。1:N RelationをJOINした出力行のfanoutで明細そのものが増えたように見せない。
- 時系列の`event time`、`valid time`、`transaction time`を単一timestampへ潰さず、未受信・部分取得・後着イベントと空集合を分離する。
- `function`（数学的な対応・部分写像）とTEI `Capability`（認可/効果/失敗を持つ実行境界）は別の意味対象。式の導出可能性から実行可能性を推論しない。
- 同じJSON値でも、Key欠落・明示`null`・空配列・取得未完了・実行失敗を区別する。
- KaTeXに書いた数式は表示表現であり、それだけで計算式の意味が受理されたことにはならない。

## 5. 例示の幅（拡張可能なPressure Corpus）

下表は**初期比較**であって全件網羅リストではない。各行に「単一の型名」ではなく複数のfacet、必要な契約、実現差を付ける。

| 系統 | 対象例 | 結び付く複数の軸・問題 |
| --- | --- | --- |
| 値・代数 | 文字列、真偽値、整数、十進数、有理数、複素数、区間、列挙、離散確率 | M2/M5/M8、表示と演算の差 |
| 構成 | tuple、record、tagged union、optional、recursive record、opaque document | M2/M8/M10、NULLとoptionalの差 |
| 集合 | set、bag、sequence、ordered map、multimap、bounded window、lazy collection | M2/M3/M9、重複・完全性 |
| 識別 | Entity、Value Object、Event occurrence、Reference、business key、surrogate ID | M1/M4/M7、identityと物理行 |
| 関係 | 1:N、N:M、n-ary relation、Tree、DAG、cycle、property graph、hypergraph | M3/M4、RelationとJOIN |
| 空間 | GeoPoint、CRS付き座標、polygon、trajectory、grid、mesh、continuous field | M4/M5/M6、座標と幾何 |
| 数学 | vector、matrix、tensor、topological space、metric space、probability distribution | M2/M5/M6、数学法則と物理配列 |
| 量 | 長さ、面積、個数、速度、絶対温度、温度差、金額、為替換算 | M5/M6/M8、単位・時点・精度 |
| 時間 | instant、duration、calendar period、event stream、bitemporal history、time series | M3/M7/M9、実時間と有効時点 |
| 計算 | partial function、aggregation、derivation、effectful Action、asynchronous Task | M2/M5/M7/M9、実行資格 |
| 複合 | 行列付き時系列、座標付きイベントstream、多国通貨帳票、部品BOM、GIS付き工程Gantt | M1〜M10、facet合成と投影 |
| 文書 | natural text、画像、図形、KaTeX、証拠付き引用、structured Markdown | M2/M9/M10、表示と原本・出典 |

新しい数学的構造、工学・科学・文化・業務固有の値が見つかった場合は、まず**どの問いで性質が異なるか**を記し、現在の軸で説明できない差があればメタ軸の追加／再分類を提案する。未知の値を既存項目に無理に詰めない。

## 6. 最小の判別票とOwner

| ID | 入力 | 期待する説明・否定例 | Owner |
| --- | --- | --- | --- |
| MX-01 | 順序付きmap・hash index付きsequence | key性／順序／格納の別軸を同時に記述。固定型Kindの重複を避ける | #2495 / #1127 |
| MX-02 | MoneyAmountにnumeric/currency/scaleの複数facet | 複数facetを許すが暗黙加算・換算・subtypeは未許可 | #2495 / #1705 |
| MX-03 | 20°C absolute vs 20°C difference | 同じ数値・単位表記で演算と換算規則を区別 | #1705 |
| MX-04 | Tree vs cycle Graph、複数親、hyperedge | Treeの制約と一般Graph・n項関係を分ける | #1126 |
| MX-05 | 1:N明細とJOINの重複、partial stream | collection本体と結果fanout、取得完了/未完了を別に表示 | #1706 / #2387 |
| MX-06 | 空間座標／行列／確率分布／KaTeXの表示 | 形状一致から意味同一性・計算資格を推定しない | #2495 / #1803 |
| MX-07 | Shared Type XをY/Zへ分割、Y/Zもfacetが多い | `Attribute→Target TypeRef`を維持し、Facet変更を型再編の自動受理へ混同しない | #1806 / #1935 |
| MX-08 | SQL `NULL`、キー欠落、空list、未取得 | 同じUIの空欄へ潰さず区別。完全性と観測根拠を保持 | #2056 / #1706 |
| MX-09 | model R2のfacet追加、Delivery R1固定 | R1とR2の正本版・派生物を区別して表示 | #1124 / #1803 |
| MX-10 | 別Hostで同じ意味型と量契約を実装 | physical API形状の一致と算術・NULL・Authority同等性を独立判定 | #2077 / #2390 |

**すべて未実行の受入設計**。既存Issue/Referenceのfixtureを再利用し、新しいテスト正本や検証Gateを増設しない。

## 7. 実装へ進む段階（広い探索と、狭い実証を並立）

- **探索T0**：上記の初期系統を起点に、数学・自然科学・ERP/組織/GIS/時系列/AI/文書等からPressure Caseを継続追加。件数・分野の固定上限を設けず、反例や互いに両立しない分類も保持（#2493）。
- **構造比較T1**：各ケースをM1〜M10の問いへ写す。既存の区分で表せない部分・複数親・重複責務・未定義の写像を明示し、メタ軸だけを必要なら改める。OWL/JSON Schema/SIの表現を参考にするが、完全移植・同じ推論意味論を前提にしない。
- **Read-only PoC T2**：既存Canonical TypeRef・Relation・Query IR・Function Data Contextの**別の根拠を束ねて読む**形から開始し、1つのGraph/Time-series/Quantity/Document例をCUI/Studio双方で説明できるか比較。Core TypeKindとSourceを先に拡張しない。
- **受理T3**：必要な型・値域・Facetの新規宣言が実証上不足する場合に限り、所有するIssueへ提案。既存Shared Acceptanceの候補／差分／Owner受理を用いる。新しい「型体系登録Engine」は作らない。
- **適合T4**：Go/Java/.NET/Wasm/DB/UI/SHOの各対象が必要な性質を保つか、正負fixtureで検証。未実現・非対応・部分観測をPASS扱いしない。すべてのTargetで最初から実証する必要はなく、支持された範囲を明記。

## 8. 外部参照（設計の着想とTEIの判断を区別する）

- [W3C OWL 2 Structural Specification](https://www.w3.org/TR/owl-syntax/)：Subclass、同値、互いに素な集合、disjoint union等の関係を独立に扱う例。**TEIの公理・Reasonerの採用ではない**。
- [JSON Schema 2020-12 Core](https://json-schema.org/draft/2020-12/json-schema-core)：assertion／annotation／applicatorの違い、allOf／anyOf／oneOf。**値の検証と人間の意味受理を別にする参考**。
- [BIPM SI Brochure（2026更新）](https://www.bipm.org/en/publications/si-brochure/)・[NIST SI conversion tables](https://www.nist.gov/pml/special-publication-811/nist-guide-si-appendix-b-conversion-factors/nist-guide-si-appendix-b9)：物理量と単位、絶対温度と温度差の換算が異なる例。**通貨・取引・契約固有の演算をSIだけから推定しない**。

## 9. 受理を待つ設計判断

1. M1〜M10をどこまで独立した軸として分けるか。新旧ケースでの重複・説明力・学習負担を測って決める。
2. 多重上位概念を`has-facet`主軸で扱うか、実際の値域包含を表す`is-a/refines`を一部へ追加するか。**複数親許容は重要な要件だが、全関係を同じ型継承として定義しない**。
3. Value Domain・Dimension/Unit・Mathematical Structureの宣言が必要なとき、どの既存Semantic/標準Plugin/Operatorへ載せるか。既存`semantic.Type`を勝手に一般化しない。
4. 数学的性質の証明可能性と、単なる人の宣言・合成テスト・実データによる観測をどう表示するか。型の説明が可能になったことを業務受理／実行権限とみなさない。
5. Studio/AI CUI/外部Hostへ公開する必要最小のProjectionを、具体的Consumerによる差分で決める。

**停止ではなく次の拡張規則**：新しい概念が出たら、まずメタ軸の不足か既存軸の新しいfacetかを分析し、既存Issueと比較し、独立な反例があるときは別Issueを許容する。モデルの可能性を早期に切り捨てず、正式な共有契約の採用のみ証拠に応じて判断する。
