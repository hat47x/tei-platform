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


## 10. 2026-11-01 再精査：10軸は「独立した型の上位階層」ではなく、責務のチェックリスト

**先行§2を補足・訂正する。** M1〜M10は網羅的な問いを探すための**暫定的なチェックリスト**であり、数学的な意味で互いに独立した座標でも、直積の10要素でもない。実際には以下の重複がある。

| 重複して見えるもの | 一次的に問う内容 | 他軸へ残す独立の問い |
| --- | --- | --- |
| M2 collection / M3 set・bag・sequence | 構成子「要素を多く含むもの」 | 重複、順序、索引、一意性、評価の完全性 |
| M2 relation / M4 graph | 多項組・参照を持つ構成 | 辺の接続、方向、到達可能性、循環、業務Relationの意味 |
| M2 tuple / M5 vector・tensor | 位置付き複合値という表現 | 体、加法、スカラー倍、基底、次元、テンソル変換規則 |
| M5 ordered set / M3 ordered sequence | 順序の公理／演算 | 要素がどの順に観測・列挙されるか、順序が保証か偶然か |
| M6 precision・rounding / M8 value constraints | 値の表現・測定の契約 | どの時点・操作で丸めるか、型全体か利用箇所だけか |
| M7 event time / M9 observation freshness | 事象が起きた時刻 | いつ取得し、どの対象集合が観測済みか |
| M1 identity / M10 storage key | 実体・値・出来事の同一性 | 主キー・インデックス・生成物ID等の物理実現 |

**対応**：これらは無理にどちらか一軸へ吸収するのではなく、構成形／性質／文脈／利用条件／観測／実現の**主張を分離**し、同じ事実を二つの正本に重複保存しない。M1〜M10のメタ分類自体は反例に応じて増減できる。

### 10.1 概念を記述する七つの役割（検討用。実装schemaではない）

1. **対象（carrier / bearer）**：その値、要素、関係、Entity・出来事は何か。名目的TypeRef、Identity、Namespace、Source版。
2. **構成（constructor / signature）**：atom、直積、直和、参照、集合、対応、再帰などをどの要素・引数で組むか。演算の定義域・値域も宣言。
3. **性質・法則（property / law）**：一意性・順序、代数・数学上の位相・距離・測度、グラフの無閉路性などの性質。適用の前提となる構成・演算も併記。
4. **解釈文脈（interpretation / frame）**：業務での意味、発行体系、単位・次元・座標参照系、時間、主体・テナント。
5. **利用箇所の要求（use-site / operation obligation）**：演算、代入、集計、更新、検索、描画などの用途ごとの前提と、要求する確実性。型全体の不変条件との区別。
6. **主張と根拠（claim / evidence）**：設計者の宣言、形式的に導出された結論、合成fixture、実データ観測、人の意味受理、実Hostの権限を独立に記録。
7. **実現・表示（representation / projection）**：Target別のbyte、JSON/SQL型、UI役割、SHO説明。意味的な損失・可逆性・出典と版。

M1〜M10は**何を見落としていないか**を問う探索用、上の七役割は**同じ事実をどう混同せず記述するか**の整理用である。どちらも固定Core enumの提案ではない。

## 11. 多重親を扱うための、関係別の意味契約

### 11.1 「上位概念に属する」を自動的な型代入へ昇格させない

- **名目的同一性**：同じSemantic Type ID/固定版であるという主張。名称や値域が一致しても同じIDとは限らない。
- **包含・特殊化**：同一の対象集合・同じ解釈文脈について、`A refines B`が `values(A) ⊆ values(B)` 等の主張をする場合、その**意味する集合・制約と根拠**を明示する。異なる発行体系やEntity identityを持つコード型の間で単に制約が一致しても同一性は導かれない。
- **複数facet**：Aが「順序付き」「重複可能」「有効時点付き」を同時に持つのは`has-facet`の複数利用であって、3親から実装やすべての操作権限を継承する意味ではない。
- **操作適合**：`usable-as`は操作＋文脈＋事前条件＋必要ならTargetを引数に判定する。例えば読み取り専用の`Sequence<Subtype>`を`Sequence<Supertype>`として読める場合があっても、書込み可能な`Sequence`で同じ代入を許すと不正な要素が挿入され得る。**可変性・副作用**は型階層だけに任せない。
- **変換**：`convertible-to`は方向、誤差、情報損失、時点・レート・座標系、失敗/未確定を伴う。可逆性と合成の可否を個別に判断。変換できることから代入可能性や同じ業務意味は導かれない。
- **物理一致**：`encoded-as`は利用可能なTarget上での表現。等しいJSONや同じ`DECIMAL(18,2)`から上のいずれの意味関係も自動推論しない。

**主な否定例**：`A is-a B` と `A has-facet C` を同じ親リンクとして閉包計算すると、Bで定義した演算の前提・Cに必要な演算の実装が存在しないまま、「型として通る」という偽の成功が起こる。

### 11.2 数学的性質は「名前」だけでは成立しない

| 主張したい性質 | 最小の前提・検証対象 | 同じ表現でも成立しない例 |
| --- | --- | --- |
| vector space | carrier V、scalar field F、加法・スカラー積および公理 | 長さ3の数値配列、地理座標のtuple |
| topological space | carrier X、開集合族 τ ⊆ P(X)、公理（空集合・全体集合、任意和、有限交わり） | edge list、Graphの接続だけ |
| graph / tree | vertex/edgeのidentity、incidence、向き、閉路・親数などの制約 | ただの2列テーブル、UIのインデント |
| ordered set | 比較関係と反射性・反対称性・推移性等、全順序なら比較可能性 | ORDER BY無しのSQL結果、hash格納の偶然の順番 |
| probability distribution | 標本空間／事象と非負・正規化等の条件、必要なら測度 | `map<string,float>`、正規化前のスコア |
| quantity / unit | 対象となる量の次元、unit、基準（差分と絶対点）、許可演算 | 同じ「°C」でも絶対温度と温度差、異通貨の足し算 |

数学上の概念を**interfaceの宣言として記述できる**ことと、それを満たす具体的演算実装があること、さらに実値・実Targetで公理が成立することを区別する。証明可能性はLean等への出力候補だが、表示上のKaTeXや生成AIの説明そのものを証明にしない。

### 11.3 多重親が矛盾・未解決となるとき

- 値制約の共通部分が空である場合（`x>0`かつ`x<0`など）は、**実際の値がまだ0件**なのとは区別して「宣言が充足不能」という判定候補を残す。
- 基礎となるcarrierが異なる、演算の同名異義・衝突、単位・座標参照系の不一致、mutable/nonmutableの要求不一致は、単に二つの親があることでは解決しない。
- 未知の前提を含む場合には、満たすとも反するとも宣言せず、`unknown`を根拠付きで保持する。制約検査上の失敗、検査手段の未対応、観測データの未取得を同一のエラーへ丸めない。
- 複数の親から同じ`Name`や`Operator`が提供された場合、暗黙の順序・勝ち負けによる解決は避ける。どの意味上の演算に束縛したかを明示する（SQLの同名異Scope解決 #2387と同じ原則）。

## 12. 情報の損失・操作の適合を明示する例

| 変更・利用 | 保持されるもの | 失われるもの・必要な補足 | 既存の責務 |
| --- | --- | --- | --- |
| `sequence → bag` | 要素と重複数 | 要素順序の情報。順序復元には追加根拠が必要 | #1706 |
| `bag → set` | 異なる要素の集合 | 同じ値の出現回数。集合上の一致を業務instance identityへ昇格させない | #1706/#1126 |
| `ordered map → unordered map` | キー・値対応 | 列挙順。物理hash indexの存在は順序保証ではない | #1127 |
| `event stream → finite page` | 取得した一部のイベント | 継続時点、後着・欠落、完全集合性。空ページから「対象不存在」を導かない | #1706/#1131 |
| `temperature point − point` | 温度差という量 | 原点の絶対基準を相殺、`point + point`と操作契約が異なる | #1705 |
| `currency A → currency B` | 条件付きの金額換算 | 換算レート、時点、丸め、経済的意味に関する不確実性 | #1705 |
| `Tuple(lat,lon) → GeoPoint` | 二つの数値 | CRS、datum、軸順の宣言がなければ意味変換不成立 | #1126/#2495 |
| `immutable read view → mutable writer` | 読み出せる値 | 書込みを許す不変条件・現在のAuthorityの検査 | #1129/#1174 |
| `Semantic Type X → Y,Z` | Attribute identityと明示された配分 | 単一Intoへ畳めば割当意図を喪失。#2404の停止条件維持 | #1806 |
| `model R2 → build / SHO` | 指定版からの派生 | R1納品固定版への暗黙更新不可。修正された意味と表示修正を別扱い | #1803/#2077 |

変換の種類は単なるYes/Noでなく、**全域・部分、単射/情報損失、可逆性、外部参照要否、観測可能な完全性**を個別に判別する（これらの列はConceptualであり新しい公開schemaではない）。構成が同じでも、有効な変換は利用文脈に依存する。

## 13. AI-first・Headless-firstの最初の検査契約

#2492/#2493の「外部要件机上評価 → 共通構成 → AI向けCUIと標準Pluginの並行 → headless E2E → GUI」の開発順序へ合わせる。今回の**メタ分類の調査・反証票は広く**、最初の実行対象は**狭い証拠単位**で構成する。

### 13.1 Read-only分類インターフェイスが答えるべきこと（暫定の出力項目）

- 対象の固定版、任意のSemantic TypeRef、識別とScope、構成の要素、適用されたfacetと依存する演算・公理を列挙できるか。
- `asserted`（宣言）／`derived`（規則から導出）／`tested`（合成fixture）／`observed`（実Target・実値）／`accepted`（権限者による意味受理）を**独立の根拠属性**として示せるか。それらの1項目が自動的に他の根拠の資格を表すものではない。
- 不足するcarrier、未宣言の単位・Reference frame、Targetで扱えない性質、出力時の情報損失、未知の外部Consumerを隠さない。
- 同じSubjectと固定Source版を対象に、AI向けCLI/JSONとStudioの図・表・SHOへの投影が**同じ主張**を提示できるか。最初の受入はGUIの新実装でなく、構造化出力と独立Oracleで検証する。

### 13.2 最小の判別対（新しいテスト正本を先行しない）

| ID | 正例 | 負例／区別されるべき出力 |
| --- | --- | --- |
| MX-11 | 順序付きmapはmapと順序のfacetを併せ持つ | 物理hash indexだけから順序を推定 |
| MX-12 | 配列の構成としてのtupleと、演算公理を持つvectorを別表現 | `float[3]`から線形代数演算が有効と推定 |
| MX-13 | 多重親がある型で、使う演算の意味と前提を明示して利用 | 同名Operatorの暗黙優先順位・不正な可変collection代入 |
| MX-14 | 限定Scopeで充足可能な値域と、値0件の観測を別表示 | 矛盾する制約を「空集合だから正常」と表示 |
| MX-15 | seq→bag→setの変換で順序と多重度の損失を説明 | `set`から元の業務instance数を逆算 |
| MX-16 | 同じSource版のquery/Model/QuantityをCLI/Studioで説明 | 違うSource版のSQL lineageや外部単位を勝手に結合 |
| MX-17 | 数式の意味と表示・人の説明を別根拠として出力 | KaTeX表示の成功を証明・計算結果受理に昇格 |

上記は**現時点ではすべて提案・NOT RUN**。M1〜M10と七役割のどちらでも説明しにくい負例が出れば新しい関心事を追加してよい。実装では先に既存#1120のidentity/graph、#1705の量演算、#1706のcollection、#2354のAI向けCLI結果、#1124のProjectionを再利用する。

### 13.3 正式採用を判断するための比較

- **説明上の過不足**：同じ事実が不整合な二つの場所に保存されないか、何がunknownか、構造を増やした際に破壊的な型変更が必要か。
- **操作適合**：同じ型のデータでもread/write、演算、集計、投影で許すことが変わるかを説明できるか。
- **独立Consumer**：一つのGraph・Unit・Temporal例をAI向けCUIと別Host/Studioで説明したとき、Target固有の形式が概念の同一性へ混入しないか。
- **Owner受理**：数学的に成立する主張、宣言済み制約、業務上の意味の受理、Hostの実行許可、実データ観測を別々に説明できるか。

**採用判断待ち**：七役割を正式なメタモデルにするか、M1〜M10の再整理を必要とするか、外部Ontology・Constraint機構の標準Plugin連携が必要か。現時点ではいずれも文書比較であり、Core TypeKind／Plugin共通ABI／Validator／Hostを追加しない。

## 14. 2026-11-01 参照資料の追加と限界

- [W3C SHACL Recommendation](https://www.w3.org/TR/shacl/) はRDF上の**データグラフとShapeグラフを分け、検査の入力を変更しない**検証方式を示す。TEIの意味型と値の適合性を別の主張にする設計の参考。TEIがRDFやSHACLをCore必須要件にするものではない。
- [W3C OWL 2 Structural Specification](https://www.w3.org/TR/owl-syntax/) はclassの交差、包含、排他、同値を別の公理として記述する例。複数親と制約の組合せを考える参考だが、推論意味論をそのままTEIへ移す提案ではない。
- [JSON Schema 2020-12 Core](https://json-schema.org/draft/2020-12/json-schema-core) はassertion、annotation、applicatorを区別し、`allOf`等による検査条件の合成を扱う。**構造検証成功が業務の意味受理を与えるわけではない**という境界に利用する。
- [BIPM SI Brochure（2026更新）](https://www.bipm.org/en/publications/si-brochure) は単位の定義・使用上の基本資料。絶対量と差分、座標や通貨などは個々の意味と変換契約を明示する。

これらは**外部仕様の参照**であり、そこからTEI固有のCore schemaやプラグイン必須依存が直接導かれるものではない。

## 15. 圧力ケースの通し例：温度観測時系列から表・グラフ・AI向け説明へ

**これは考え方を照合するための手作業による合成例。TEIの現行Worldに以下のすべてが表現済み・実行済みという意味ではない。**

**対象**：一つのセンサー S の観測イベント e1/e2/e3 を、業務上の固定観測範囲 [10:00,11:00) で扱う。e1は10:05・20°C、e2は10:10・21°C、e3は10:07・20.5°Cで後から到着。到着順と測定時刻順は異なる。入力はページ境界を持ち、この期間の全イベントを取得し終えた証拠はない。

| 七役割 | この合成例で表すもの | 誤って導いてはいけないこと |
| --- | --- | --- |
| 対象 | 観測イベントe1/e2/e3のidentityとセンサーS | 数値が等しい二行を同じ出来事とする |
| 構成 | センサー参照・測定時刻・温度点からなるEventのsequence | 配列になっていることから測定時刻順を保証する |
| 性質・法則 | EventIDによる同一性、重複・遅延・順序の条件 | 数値arrayからベクトル空間の公理を得る |
| 解釈文脈 | °Cの絶対温度、測定時刻、受信時刻、センサー固有の校正情報 | 温度差の換算と絶対温度の換算を同一式とする |
| 利用条件 | 時間順折れ線には測定時刻による並べ替えが必要 | 受信時刻順をそのまま測定時刻順として表示する |
| 主張・根拠 | 3件は取得済み、当該窓の完全集合性はunknown | 「3件取得」を「この期間は3件だけ」と表示する |
| 実現・表示 | 行表、時系列グラフ、AI向け構造化説明、SHO設計書 | 図に3点描けたことを履歴の完全性とみなす |

**正例**：AI向けread-only説明は「表示している3点」「観測窓」「測定時刻による並び」「全件取得は未確認」「単位と測定条件を確認した範囲」を別々に返す。Studioの表・グラフ・設計文書は同一Source版を使い、この説明と矛盾する場合は差分を示す。

**負例**：e3を無視して時間順グラフが完成したと断定する、e1/e3の値が近いから同一Eventとする、イベント数を重複排除後の異なる温度値の件数に置き換える、温度点を二つ加算する、取得範囲外を「観測ゼロ」と表示する。

この例には将来の専用時系列Core型を必要条件としていない。既存Data、Relation、SourceRef、Dataset/Scenario、Query・Collection completenessなどを読む段階で**説明できない問い**を記録し、その不足だけを#2495の新しい抽象候補として判断する。

## 16. 値域と多重facetの包含関係：型の付替えとの連結

値域（Value Domain）は**あるCarrierと解釈・適用Scopeの上で許される値の条件**として捉える。条件は有限列挙・範囲・正規表現・述語・要素間不変条件・演算前提などから記述され得るが、それらの機構がTEI Coreに実装済みであるという意味ではない。

- **型全体の不変条件**：例えばCode体系が数字10桁を要求。Semantic Typeの名目的同一性やIssuerの一致は、文字列正規表現からは導かない。
- **利用箇所の条件**：同じCustomerCodeでもある画面は必須、あるAPIはNULL可能、ある取引日には失効コードを受け付けない。型自体の許容値とField/Operationの条件を分ける。
- **物理的な格納条件**：`VARCHAR(10)`、`DECIMAL(18,2)`は物理的な表現・制限であり、業務上の識別体系や単位を確定しない。
- **実データ適合**：`OldDomain ⊆ NewDomain`という証明があっても、変換の意味的同一性や外部Hostの実装適合は別。逆に`NewDomain ⊂ OldDomain`の場合、変更先に適合する既存値が全件かは、対象Snapshot・Provider・取得完全性を伴う観測または明示変換契約で判別する。
- **数学的な包含の判定不能性**：任意の述語間の包含や多重facetの整合は常に機械的に決定可能とは限らない。`true/false`へ無理に二分せず、`unknown`、`unsupported`、`not-observed`等を必要な範囲で独立に示す。
- **空のデータと矛盾した定義**：保存レコードが0件なのと、二つの親制約が同時に満たせないのは違う。後者の充足不能を示すには対象Carriers/Scopeと制約の論理を確認する。

値域条件の種類・検査時点・依存するContextは、既存のValidation、Operator、Provider、データ移行計画[#1935](https://github.com/hat47x/tei-platform/issues/1935)と[#1806](https://github.com/hat47x/tei-platform/issues/1806)で必要に応じて具体化する。新しい一般的なTypeKindや第二のAcceptance Engineを追加する前に、既存契約で表現できる範囲を実証する。

**Ownerに求める最初の設計判断**：10軸や七役割を正式な固定分類として採用するかではなく、上記の反例をどの関係・検査・根拠で表示すれば、誤った自動推論と型の重複を避けられるか。その説明が欠けるところから抽象を増分採用する。

## 17. 2026-11-01 外部エコシステムでの圧力試験：同じShapeでも異なる意味

同じ構造を二つの製品で表現できたことと、同じ意味が保たれることは違う。この違いを一貫して説明できるように、[#2501の56ケースPressure Corpus](DATA_STRUCTURE_PRESSURE_CORPUS_2026-11.md)を追加した（探索台帳、実装済み機能一覧ではない）。

### 17.1 一次資料で確認した圧力と、TEIへ持ち帰る論点

| 外部の代表例 | 参照仕様で確認できること | TEI側に必要な概念上の区別 | 採用状態 |
| --- | --- | --- | --- |
| [Apache Arrow Nested Layouts](https://arrow.apache.org/docs/format/Intro.html) | List、Struct、Map、Union、辞書符号化の形状と物理レイアウトを区別。MapはList of key/value structsに近い物理構造 | **配布時の表現**・**key/valueの意味**・**元の出現数／順序**・**式での利用可否** | 情報収集。Arrow Provider採用ではない |
| [Protocol Buffers Field Presence](https://protobuf.dev/programming-guides/field_presence/) | explicit presenceを持つ要素と、repeated/mapのようにunsetと空を区別できないフィールドがある | **欠落／空／デフォルト値／null** と、元の観測可能情報の保持 | 特定wire契約の採用ではない |
| [xarray Data Structures](https://docs.xarray.dev/en/stable/user-guide/data-structures.html) | ndarrayに加えてnamed dimensions、coordinates、attributesを持ち、ラベルを使ってalignできる | **array shape**、**Dimension identity**、**座標値・ラベル**、**alignmentが採る集合範囲** | 科学配列Plugin案の参考 |
| [OGC Simple Features](https://www.ogc.org/standards/sfa/) | GeometryのPoint／Curve／Surface等と、Spatial Reference Systemを伴う | **形状**、**CRS・軸順・datum**、**距離・投影操作の文脈** | GIS例の参考。全OGC準拠を主張しない |
| [W3C SHACL](https://www.w3.org/TR/shacl/)／[JSON Schema](https://json-schema.org/draft/2020-12/json-schema-core) | 外部値に対する制約・検証と、複数条件の合成・検証結果の考え方を提供 | **受理済みの意味**と**値／Structureの適合**、検査手段によるcoverage | Validatorの必要条件を考える資料 |

上表の「TEI側に必要な区別」は**推論／要件仮説**であって、外部規格の規定でも、既存TEIの実測PASSでもない。たとえばArrowのMap形を受け入れるだけで、業務Mapの唯一キー／列挙順／重複／欠損の意味が満たされることは保証されない。

### 17.2 重要な教訓：情報を落とす変換の多くは局所的には正常

- 集合へ変換して件数が変わっても、重複が業務意味なら損失であり、重複を問わない集合上の要件なら適切な変換になり得る。
- 座標を持つArrayを単なるArrayとして保存できても、座標によるalignmentが必要なConsumerでは意味が足りなくなる。
- Protobuf repeated fieldを空のListにできても、「項目を送らなかった」と「意図的に空を送った」の差を必要とするAPIではその差が失われ得る。
- 絶対温度を数値と単位だけにしても、温度差演算との区別・換算の原点が失われる可能性がある。

**全てに共通する汎用lossless変換を要求するより、どの用途・観測について情報を保持するかを精密に指定する方が、実装の自由度と意味上の安全性を両立する**という設計仮説を採る。

## 18. 操作・観測を基準にした「意味保存」の最小定式化（採用前）

変換を f: S → T、対象の固定版と解釈文脈を K、利用操作が要求する観測集合を O(K) と呼ぶ。ここでいう変換には、物理符号化、意味型の付替え、表示への投影、Plugin境界越え、生成先Targetへのloweringを含み得る。ただしこれらが同じ実装で行われるとは限らない。

### 18.1 観測の保存条件

ある観測 o について、変換の前後で対象となる値／集合／出来事から得られる結果が**明示された同値関係の下で比較可能**かを調べる。

- 例示：o = 出現数、順序、Event ID、Value Domain membership、Quantityの単位・演算結果、Time Window completeness、Source revision、権限を伴う実行効果。
- 比較で必要な前提：変換の定義域、対象となるSource/Target版、入力Snapshotや全件性、可変性、単位・参照文脈、許される誤差、必要な操作。
- 観測に関する一致を**試験済みの有限Fixtureの性質**として報告するときと、全入力に対する**形式的な法則・証明**として報告するときを分ける。
- 完全性を必要とする集計で取得対象が未知の部分集合の場合、観測できた範囲の値が正確でも「全件の集計が保存された」は未確認。

表記例は o_S(s) ≈ o_T(f(s))（観測oについて指定された同値関係）の形で考えられるが、**この形をそのまま公開Formal DSL／型検査器へ採用する判断ではない**。

### 18.2 変換の性質と適合を区別する

| 概念 | 問うこと | 誤った一般化 |
| --- | --- | --- |
| 構造上の転写可能性 | Targetで値を表現できるか | 変換ができたから同じ意味 |
| 値情報の保持 | 可逆か、重複・順序・Presenceが損失しないか | 物理型が同じならlossless |
| 観測相対の保存 | 利用Operationが必要なObserverに限り意味を維持するか | 全ての用途で同じ可否 |
| 名目的同一性 | Semantic Type/Issuer/Scope/Versionが一致するか | 値域の包含なら同一Type |
| 実行資格 | Hostに必要なAuthority、Provider、契約、Tx/Commitが揃うか | 構造・演算の適合なら実行できる |
| 証拠上の資格 | 宣言・形式証明・合成テスト・実Target観測・Owner受理は何か | ある一つのPASSを全てへ昇格 |

**重要な反例**：Sequence→Bagは順序を失うが、同じ値・出現数を保つ条件では加算集計は成立し得る。Sequence→Setは重複を失い、SUMやCOUNTを一般に保存できない。したがって「SequenceはBagのsubtypeか」「BagはSetの親か」という一つの型階層だけで、全用途の代入・変換・演算を決定しない。

### 18.3 変換の合成では前提の引継ぎが必要

S→TがObserver O1を保存し、T→UがObserver O2を保存するだけでは、S→Uが任意のOを保存するとは言えない。合成の成否には、O1/O2の対応、各変換の定義域・出力値域、参照版、必要な単位と文脈、対象データの完全性、損失した情報の再取得可能性を確認する必要がある。

最小の試験票は、(a) 個別変換の正例、(b) 情報損失の負例、(c) 合成したときのみ失敗する反例、(d) 同じ物理Targetだが解釈文脈が違う反例を含める。

**所有**：構造保存写像・Projection・編集逆写像の共通設計は[Issue #2497](https://github.com/hat47x/tei-platform/issues/2497)および[同一Draft上の設計文書](STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)が担当する。[Issue #2498](https://github.com/hat47x/tei-platform/issues/2498)はArrow/Protobuf/xarray/CRSなど**実際の外部Serialization／Consumerに限定したConformance実証**を担当する。#1806の型分割安全化Draft #2404は継続し、AttributeID→TargetTypeRefの割当意図を失う候補を禁止する。

## 19. AI大量並列開発時の担当・根拠の分割

安価なトークンを大量調達してAIエージェントによる独立探索・PoCを増やす前提を、**候補領域の探索を広げ、検証範囲を明示しながら統合する設計**に反映する。

| 担当 | 増補できる対象 | 成果物・再利用する基盤 | 他レーンから独立した成立条件 |
| --- | --- | --- | --- |
| #2495 メタ概念 | 新しい構成子・facet・公理・多重親の関係 | 本設計文書、#1120参照、#1705の量等 | 既存分類で説明できない具体例と境界を提示 |
| #2497 構造写像・投影 | 構造の保存写像、Observer、UI/SHO Projection、編集の逆写像 | [共通の構造保存設計](STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)、#1124/#997 | 用途・文脈・保存不変条件／Putbackの反例を提示 |
| #2498 外部形式Conformance | Arrow／Protobuf／xarray／CRS・Java/.NET Target等の変換・Consumer検証 | 既存Scenario/Oracle、#2056/#2077/#2390 | 形状一致とPresence/Coords/単位等の意味保存を実Consumerで比較 |
| #2501 Corpus | 科学・業務・外部Protocol・GIS・UI等の多様なPressure | [56ケース台帳](DATA_STRUCTURE_PRESSURE_CORPUS_2026-11.md)、出典、Issue参照 | 既存型と違う要件・反例を収集。対象数の上限なし |
| #2493 要件ポートフォリオ | 製品領域・異質Consumer・独立PoC・Agent作業票の拡充 | 既存#2017/#1445運用・Issueトリアージ | 独立探索、検証、統合の待ち時間・負担を区別 |
| #1124/#2354 Consumer | Studio、AI CUIへのread-only説明と同版差分の表示 | Source/Typed Ref/既存CLI・UI Kit | 観測した範囲・鮮度・Disclosure・未対応を明示 |

**並列化の優先境界**：探索用の異なる数学・科学・業務の候補は独立に拡張する。実装へ移す場合、同じCanonical/Shared Acceptance/Schema/Hostへ複数のAgentが独立した変更を競合投入しないよう、既存Owner/Issueと変更予定ファイルを明示する。全探索を一本の縦切りに縮約する義務は置かず、同一意味の正本だけを統一する。

## 20. 次に試す、説明力の高い6つの観測対

| 票 | 正例に必要な操作・Observer | 負例：実証で見分けたい失敗 | 実行先 |
| --- | --- | --- | --- |
| OB-01 | Sequence→Bag後に重複込みSUMが等しい（固定値・演算規則を明示） | 出現数を落とすSetでSUM一致を誤認 | #1706 / #2497 |
| OB-02 | Presenceを要求しないConsumerへProto repeatedを渡す | 欠落と空を区別するConsumerで情報が復元可能と断定 | #2056 / #2497 |
| OB-03 | xarrayのdims/coordsを保った投影でlabel alignment結果を再現 | ndarrayのみで同じshapeの別座標を同じ観測対象へ結合 | #2497 / #2077 |
| OB-04 | OGC GeometryのCRSを明示し、対応する処理対象で空間演算 | CRS未宣言の座標pairで距離値を確定 | #2497 / #1126 |
| OB-05 | 温度点の変換・点の差分という合法な演算を区別 | point＋point、差分にoffsetを加える誤換算 | #1705 / #2497 |
| OB-06 | Source R2の新TypeRefとその結果を新Target/SHOへ固定版で転写 | 固定納品R1の成果物が自動的にR2へ更新済みと表示 | #1803 / #2077 / #2497 |

全票は**追加した検証提案であり、ここでの新しいGo/CLI/DB/外部Provider実測はNOT RUN**。元のSource/Scenario/Expected Oracleは既存担当を利用し、値域・意味型・物理型・操作適合・証拠を二重に管理する新しい統合Engineは設けない。

**今回の結論**：M1〜M10のチェックリストと七つの記述役割を上から型継承にするより、広いPressure Corpusを基に「必要なObserverをどの条件で保存できるか」を説明できるモデルが、異質Targetを扱うTEIのメタモデルに向いているかを反証可能な形で検証する。正式な契約採用・Owner意味受理は、その比較結果と既存Product/Target責務を踏まえて別判断とする。


### 20.1 並列IssueのOwner境界（同日追補）

[#2497](https://github.com/hat47x/tei-platform/issues/2497)は共通の構造保存写像とUI/文書の編集逆写像を担当する。[#2498](https://github.com/hat47x/tei-platform/issues/2498)はその概念を**外部形式に投影したときの実Target Conformance**に限定し、独立した一般Morphismモデルや新しいLens規則は作らない。両者は同じSource/Contractの版を参照するが、読み取り専用Corpus、変換の観測、編集の更新権限の証拠を混同しない。上表のOB票は共通観測の設計例であり、外部ConsumerのXT-01〜08は#2498の票へ参照させる。多くのAIエージェントを同時に動かす場合も、同じ意味の正本を複数Issueで独立に定義しない。

## 21. 2026-11-02 動的意味の検討：構造・法則に加えて「どう進むか」を記述する

[Pressure Corpus PC-121〜132](DATA_STRUCTURE_PRESSURE_CORPUS_2026-11.md)で、静的なStructureだけでは説明が足りない例を拡張した。**同じGraph/データ型・同じOperator signatureを持っていても、どの時間の進み方、状態保持、評価順、Solver、外部入力に従うかによって動作の意味は異なる。**

M1〜M10は既に時間と数学法則を含むが、M7の「いつの値か」とM5の「どんな公理か」だけでは、遅延付きfeedback、連続系Solver、非同期イベントの実行モデルを比較しにくい。そこで**M11候補（Model of Computation／動的意味）**を探索上の追加質問として提案する。**正式な11番目のCore TypeKindやRuntime種別の採用を意味しない。** 

| 観点 | 区別する問い | 既存責務／仮説 |
| --- | --- | --- |
| Structure／M2・M4 | 何がどの辺・Port・Relationでつながるか | #1126、#2495 |
| Law／M5・#2544 | 何の演算束縛・不変条件を満たすか | #988、#2544 |
| Time／M7 | Event time、valid time、処理時刻のどれを指すか | #2506、#2582 |
| **Dynamics／M11候補** | 値と状態が**いつ・何を契機に・どの評価規則で**進むか | 専門Plugin／#988/#2507。実証不足を先に評価 |
| Realization／M10 | どの外部Solver/Engine/Hostが実現・観測するか | #951/#2077。TEI独自Hostは前提にしない |

### 21.1 同じ循環Graphの異なる動的意味

1. **即時の代数ループ**：同時刻における二つの変数の拘束。逐次1回評価だけでは確定値を求められない場合があり、Solver、初期値、解の存在・一意性、収束・精度条件が必要。
2. **Delayを持つfeedback**：一方の辺が前tickのStateを参照する。循環を即時計算の矛盾とせず、Stateの初期化、tick基準、更新の原子性と版を宣言する。
3. **非同期Event循環**：入力Eventが後続Eventを発行する。時刻・順序・重複・取消・副作用/Receiptにより、同じGraphでも違う観測が生じる。

同じGraphであることから三者を自動分類・実行しない。関係の役割（構造的参照、Dataflow、Event因果、制約、Operation/副作用）と使用するModel of Computationを区別する。実装では専門Pluginの演算・状態・時間の能力宣言を参照する案を比較し、Coreが全てのSolverやスケジューラを内蔵する構成を前提としない。

### 21.2 異周期とSimulation Timeの統合負例

合成例：Physics 60Hz、CPU推論10Hzで姿勢・認知状態を更新。物理側が6回tickする間に推論の新値がない場合、**最後の値を保持するのか、古い値を無効とするのか、補間/予測するのか**を利用条件として記述する。表示Frameのレート、物理tick、CPU推論周期、Simulation Time、Wall Time、外部Hostのタイムアウトは別の時計である。

速度を落としてAI Botを動かす試験により入力を間に合わせられても、通常速度での性能・精度・実Hostの応答期限を保証したことにはならない。異周期のリサンプリングでLossが生じる対象は、#2497の用途相対の保存性と#2514の独立Oracleへ渡す。

**外部比較元**：[FMI 3.0.2](https://fmi-standard.org/docs/3.0.2/)はModel Exchange・Co-Simulation・Scheduled ExecutionでSolver/Schedulerの所有を分ける。[Godotの物理補間](https://docs.godotengine.org/en/stable/tutorials/physics/interpolation/physics_interpolation_introduction.html)は物理tickと描画Frameの差を説明する。これはTEIのFMI適合・Godot本体機能の実装完了を主張するものではない。

## 22. Claimの成立条件を「意味の前提」と「観測の前提」に分け、変更後の再評価へ接続する

意味型やFacetについて**演算が可能だという宣言**と、ある結果値が**現在の範囲で正しいというClaim**は、変更による失効条件が違う。この差が、#2582の増分観測、#2593のMeasure再集計、#2516の用途相対Construction Context、#2586のQuery→Context→Claim→Pattern縦断を結ぶ。

### 22.1 何の版を保持するか（単一のglobal revisionを強制しない）

| 成立対象 | 必要となる固定条件 | 内容が変わったときの問題 |
| --- | --- | --- |
| 構造・Lawの主張 | Semantic Type/FacetとOperator束縛・制約・解釈版 | 旧Lawが新しい演算実装に適用可能か |
| 派生Measureの値 | Data Slice/Fact identity、Relation grain、Snapshot/Read-set、評価時点・丸め条件 | 集計の母集団・重複・再集計可能性が変わる |
| Query/Contextに関するClaim | Query Source/パラメータ、Context Instance/選択epoch、取得Scope・完全性 | 局所結果と全件結果の範囲が変わる |
| ConsumerのProjection | Claim参照、用途、表示する粒度、Consumer版、開示ルール | 同じ正しい結論でも表示・開示できる範囲が変わる |
| 実Hostでの効果 | Host Authorityの現在値、実Provider/Commit/Receipt・独立観測 | 型検証・解析成功から実行結果へ昇格できない |

**変更の発生箇所だけを見て全てのClaimを一律に無効化する方式も、byteが同じだから全て有効なままにする方式も採らない。** Claimが依存するSubject・法則・Dataset・時点・Scope・Consumer・Authorityの**既知の関係の範囲**で差分を伝播し、不明な依存は未確認として残す（#2497/#2505/#2582）。

### 22.2 再評価の条件（設計用対照表）

| 変化 | 維持し得ること | 再評価・撤回が必要なこと |
| --- | --- | --- |
| 型のUnderlyingだけ変更 | 名目ID・業務上の宣言が同じことはあり得る | Target表現、Generator適合、実Data/Consumer。型の意味一致は別判断 |
| Operator束縛・丸め規則変更 | 元のFactとSource provenance | Law適合、計算結果、Target fixture、SHO内の計算説明 |
| データに追加入力（挿入のみ） | 固定された過去Scopeの正の存在Claim等は維持可能 | COUNT/AVG/TOP-N、全件不存在、閉じていない窓の最終値 |
| 訂正・取消・retract | 変更が及ばない独立Subjectの主張 | 旧SUM・平均・存在/不存在・JOIN仮NULL・行のSelection |
| Query/Contextの取得範囲縮小 | 既に得た個々の値の字句は同じ場合がある | 全件性、対象母集団、集計grain、外部への開示対象 |
| UI選択epoch変更 | Business ID・Sourceの型宣言 | 古い非同期応答の表示先と操作候補 |
| Disclosure Policy/Grant変更 | 内部的な計算根拠・歴史的な受理済み意味 | **現在Consumerへの送出資格**。非公開根拠の値/件数を漏らさない |
| Target/Build版変更 | 固定されたAccepted Source意味 | そのTarget実物でのLaw・Serialization・外部Consumer適合 |
| 文書説明・テンプレートだけ変更 | Sourceの型/演算意味 | 文書の出力版・引用/表示依存。実Targetを一律に再実行する理由にはしない |

### 22.3 具体例：平均の値とその主張の寿命

独立の合成Oracle（[#2593](https://github.com/hat47x/tei-platform/issues/2593)を再利用）：

- グループA：2件、平均10 → sum=20。
- グループB：8件、平均20 → sum=160。
- 同じUnit/対象集合・互いに重ならないFact ID・同一Snapshotという条件下で、全体平均は **(20+160)/(2+8) = 18**。局所平均を単純平均して15とするのは不正。
- Bの8件中1件が20→30に訂正されれば、**sum=170、全体平均19**。型・Operator・集計Formulaが不変でも、以前の「平均18」は旧Snapshotに限定された履歴Claimであり、現在の結果として表示すべき値ではない。
- さらにBの取得が途中ページのみだったなら、18/19という数値が一部データに対して算出できても、**全対象の平均**というClaimの必要条件は未確認。表示上の精度の問題と、集計母集団の完全性の問題を区別する。

#2593はMeasure/reaggregation、#2582は差分・retract、#2586はHeadless縦断、#2514はOracleの正本、#2516はConsumer毎の開示資格を所有する。本Issueは**これらの結果を意味型のFacet・Law・必要なContextへ接続する抽象の不足**を扱う。同じ計算器や異なるApproval/Claim Engineをもう一つ実装する提案ではない。

### 22.4 AI向けCUIとSHO向けの出力資格を分ける

同一のClaimが内部CUIで根拠付きで提示可能でも、SHOの外部配布版へ必要な出典・値・件数を転送できるとは限らない。独立して評価するのは (a) 導出条件の適合、(b) 参照する証拠のScope・Freshness、(c) Consumerの表示目的に対する十分性、(d) Disclosure Policyの現在値、(e) 実Consumerの読込み・表示観測である。

公開向けの説明は未開示のData Slice、内部権限規則、非公開の件数を推測可能にすることもある。**送出を許された情報だけで同じClaimを支持できるか**を検証し、支持できない場合は未確定・保留／要約変更とする。どの詳細を開示できないかについても、内部のPolicy情報を外部へ漏らさない。

### 22.5 最小の統合反証（新しいTest SSOTにしない）

| ID | 既存の独立Fixtureに追加する操作 | 誤った成功として排除すること | 実行Owner |
| --- | --- | --- | --- |
| AQ-01 | 同じType・Law版でFactを1件訂正する | 型と式が同じなので現在のAverage Claimも不変 | #2593/#2582 |
| AQ-02 | D1完全／D2途中ページからSummaryを要求 | 取得済み値だけで全件平均・全件不存在を確定 | #2506/#2586 |
| AQ-03 | 非同期A応答の後にBを選択 | 古いContext epochのClaimをBへ表示・編集 | #2506/#2500 |
| AQ-04 | Operatorの丸め方式だけ変更 | 旧Source byte一致や古いfixture PASSで新Contract適合を宣言 | #1705/#2514 |
| AQ-05 | 内部CUI→外部SHOでPolicyを変更 | 技術的に導出済みなら実値・根拠を無条件送出 | #2516/#1178 |
| AQ-06 | 型・Data固定でRuntime成果物を差し替える | CIやSchema検査が成功したので実Hostの業務効果も成立 | #2077/#2390 |
| AQ-07 | 物理60Hz、認知10Hz、試験の時間倍率を変える | Tickを同一視し、通常速度のリアルタイム適合へ昇格 | #2497/#2507 |
| AQ-08 | 同じ循環GraphにDelayと即時制約の2種類を記述 | Graphの形だけでSolver不要・実行可能と確定 | #2495/#2544 |

**状態**：AQ-01〜08は既存Issueのfixtureへ接続するための**反証設計／NOT RUN**。M11の正式採用、時間/Simulationの共通Core型、実Provider/Hostの検証、開示Policyの実装、Owner受理は未実施。広い探索・独立試験を進め、十分な観測結果がある不足に対してのみ具体的なContract追加を判断する。

## 23. 2026-11-02 標準UIへ返すのは「型名」ではなく用途相対の操作可能性（Affordance）

他のFunction Data Context／標準UI／AI向けツールの議論（#1127/#2506/#2507/#997/#2500/#2597）との接点として、**Semantic TypeのKind/多重FacetだけからUI部品と編集権限を一意に決めるのでなく、対象Context・Operation・必要なEvidenceが揃う範囲で、標準UIが候補を提示できる**という考え方を採用前の比較案として明確化する。

### 23.1 「表示できる」「編集候補を出せる」「業務実行できる」は別資格

| 役割 | 判断材料 | 取得できる結果 | まだ保証しないこと |
| --- | --- | --- | --- |
| **構造表示可能** | TargetのData Shape、型付きRole、Grain、Source版、Scope、Facet | Table/Form/Tree/Graph/Gantt/Map/文書の表示候補 | 専門Layout/Graphの数学的・業務的意味、全件性 |
| **Pattern適用候補** | #2507のRequired Shape/Operationと既存#2506のFunction Context、適用すべきRule/Time | 適合・条件付き・曖昧・未対応と不足前提を説明 | 関係/単位/エンティティを型名で自動受理すること |
| **編集Intent候補** | View Occurrence→Source/Relation Occurrence、Action Ref、base revision、Context Instance、Loss/逆写像条件 | 特定の対象に対するTyped Intent、または複数候補／read-only | 現在HostでのAuthority、Tx成功、Business完了 |
| **実行・観測** | 既存Host/ProviderのPolicy、Operation Binding、Commit/Receipt、独立再読取 | 実際に観測されたEffectと失敗分類 | 別Host/Target/他版での自動的な成功 |
| **AI/CUI/SHO用説明** | 同じ受理Sourceに対する用途、開示Scope、Consumer版、根拠/unknown | 許可された範囲の構造説明／操作候補／出典付き文書 | 裏側の機密情報の開示、外部Consumer実受入 |

「数学的にTreeと説明できる」ことと「業務Relationを一つのTreeへ損失なく表示できる」ことは別。DAGを表示する標準Graphは可能でも、単一親を前提とするTree編集が正しいとは限らない。Measureに数値型が付いていても、時点Stockの期間Sumや平均の再平均をChartの標準Aggregationとして提供しない（#2593）。`many`のFieldはListとして表示できても、重複/全件性/個別Relation Identityが不明なら、自動で全件Unlinkする操作を作らない。

### 23.2 TEI DSL／専門Pluginと標準UIの分担

標準のデータ構造・型付きRelation・Operation参照・単純なTEI DSL/Scriptを入力として、Studio/標準UI側は**一般の構造編集・表・図・式の可視化**を提供し、専門Pluginは必要な演算・公理・特別なLayout・外部計算・Providerを追加する。概念上のM1〜M11を普段のPlugin作者がすべて意識・宣言することは要求しない。AIが提案したScriptも同じ標準のSource/受理経路へ戻し、説明可能なAction候補を受理済み意味や本番実行許可へ自動昇格させない。

**Ownerは増設しない**：型とFacetは#2495/#2544、Context・Grainは#2506、Pattern適合は#2507、標準UIは#997/#1124、View→Intentは#2500、複数編集ContextのDirty State・Saveは#2597、実Effect/Hostは#1174、AI/SHO向けDisclosureは#2516。今後の最初の確認は「同じ受注Header/Detailまたは組織所属Contextを、TableとGraph/CUIで説明しても同じSource/Meaningを参照でき、どの編集候補を許さないか説明できるか」とし、既存#2586の統合Fixtureに帰属する。

**状態**：標準UIへの投影を検証するメタモデル上の要件整理。自動Component選択、編集Intent生成、本番Host、デスクトップUIの製品実装やOwner受理は**NOT RUN／未採用**。新しくUI種類ごとのCore型階層や全用途の万能UI Generatorを導入した主張ではない。

## 24. 2026-11-02 Everything is Data：標準構造・意味Role・構文糖衣・実行を接続する

新しい横断親[#2619](https://github.com/hat47x/tei-platform/issues/2619)、担当[#2620](https://github.com/hat47x/tei-platform/issues/2620)（標準構造と意味解釈）、[#2621](https://github.com/hat47x/tei-platform/issues/2621)（Markdown Authoring）、[#2622](https://github.com/hat47x/tei-platform/issues/2622)（Data Operations）との関係を確認した。M1〜M10と動的意味M11候補は、新Core型Kindの一覧でなく、標準データ構造へRoleを適用する際に**意味・保存性・利用条件を問う分析項目**として使う。

### 24.1 「全てがデータ」と「全てが同じ意味」を分離

Everything is Dataは、Function、Model、Rule、Test、Scenario、Projection、Generator等の定義を標準構造（Record、Sequence、Relation、Expression、Referenceなど）の組合せとして**記述・編集・追跡できる**ことを指す。全てを同じPrimitive型に変えるのでも、全ての記述を実行可能コードにするのでもない。

| 記述側の役割（候補） | 保存する対象 | 意味が成立するための追加条件 |
| --- | --- | --- |
| Value / Literal | 入力値・Record・例示 | 元値・Presence・型・出現とScope |
| Definition / Declaration | Function、Data Model、Rule等 | 安定ID・Namespace・Source span・版・Owner |
| Reference / Binding | TypeRef、OperatorRef、Relation path、Plugin | 解決先・Scope・曖昧性・版 |
| Expression / Application | Filter、演算、Query、変換 | Operator束縛・引数・型/Effect・評価文脈 |
| Interpretation Role | Function/Test/Model等の意味の付与 | Required Shape・Role/Operator ID・Version・依存関係 |
| Evidence / Acceptance | 形式主張・Expected・Observed・意味受理 | asserted/derived/tested/observed/acceptedを別記録 |

表の**Interpretation Role**は、§10の七つの「対象・構成・性質・文脈・利用条件・根拠・実現」という**説明上の役割**とは別概念。前者は意味解釈部品が構造へ与えるRole、後者は同じ情報を二重正本へ保存しないための整理である。同じRecord shape・同名Functionから同じ意味IDや許可を推論しない。

複数Roleは適用可能とし、同一Carrierに対する複数の演算構造・異なるLaw・同名Operator競合・適用条件の矛盾は#2544のOperator束縛へ返す。意味Roleの適用だけからOwnerの意味受理、Hostの実行能力、実Providerの副作用資格へ昇格させない。

### 24.2 構文糖衣は同じ意味束縛へ到達したときだけ統合する

関数形式・パイプライン形式・メソッドチェーン・Markdown表／式・Graphの構文は、**明示された同じOperatorとOperandへ解決した場合に限り**同じ型付きExpressionへ投影できるという仮説を検証する。

比較のための擬似Authoring（現行TEI Script構文として採用済みという意味ではない）：

    where(orders, isOpen)             // function
    orders |> where(isOpen)           // pipeline
    orders.where(isOpen)              // method
    | result | operator | input  | predicate |
    | active | where    | orders | isOpen   |

解決の正本候補は字句としての「where」でなく、**Operator ID/versionと引数Role、Source Reference、型／Effectを持つApplication**。構文違いを消してもSource span、文書上のOccurrence、提示順、記述者の意図、未受理状態は失わない。

同じ意味と呼ぶために確認すること：
1. 入出力のSemantic Subject、Operator namespace/ID/version、引数の型と束縛の一意性。
2. Boolean／三値論理／missing/null、Providerのcollation、精度と丸め、時点とEffect/Purity。
3. 対象集合のset/bag/sequence、順序・重複、Grain、Relation path、取得範囲・完全性、Limit/Sort/Joinとの評価順。
4. Provider pushdownとローカル実行の同等性・未対応・必要な独立Oracle。
5. Source spanと意味版、Syntax修正／意味変更／Target再生成の差分。

RDB行のwhereとGraphの辺・Pathのwhere、Treeの再帰探索、SQLのWHEREという同じ字句を、必ず同じOperationとして推論しない。何をFilterするか、時点・可視範囲・重複・無限性が一致しない場合はRole解決を保留する。**表の一列にwhereと書いてあるだけでその表を実行コードにしない**（#2621）。

別チャットでは既存言語構文を活かす方向（TypeScript系が第一候補、Starlark/CEL等を比較）を議論している。これは**検討中のAuthoring候補**であって、TEI言語の正式採用ではない。TEI固有の価値はSyntaxの独自性ではなく、Semantic Context、Facet、Relation、Operator、Plugin、Loweringとその出典/意味保存へ置く（#988/#1018/#1063）。AIフレンドリー性は可読性に加え、曖昧なBindingと未対応ReasonをCUI/JSONで説明できるかで評価する。

### 24.3 意味解釈部品の自己記述とbootstrap

Role/Operatorの解釈部品自身をRequired Shape・適用条件・Operator/Law・Effect・Version・Target条件という標準データで表現する案は#2620が所有する。検証の出発点は次の段階を分けること：

- **最小bootstrap**：Literal、Record/Sequenceの最低限の構成、安定Reference、Source span、Plugin解決の規則。
- **Role候補解決**：必要なShapeとScope、依存するRole/Operatorと版、未提供Plugin・曖昧な候補。
- **意味解釈**：型付きExpression、Lawの適用条件、EvidenceとOwnerによる意味受理を明示。
- **Target実現**：Provider capability、Lowering、実Host/Authority、Effect/Receiptを独立に確認。

依存循環、二Roleの同じOperator名の衝突、Plugin版の非一致で、暗黙の優先順位・デフォルト実装への置換により成功したと見せない。新しい共通Runtimeや第二Semantic Registryをこの比較の前提にしない。

### 24.4 Data Operationsを非RDBも含む圧力として使う

#2622のData Operations候補は、Data Modelの構造・Function Data Contextの使用範囲・Providerの物理実現を分ける比較場となる。Data ModelにRelationが存在することと、あるFunctionがそれを取得することは異なる。**利用する起点・Relation path・Field・grain・Read Scope／完全性・選択Snapshotを明示Mapping**する（#2506/#2622）。

where／select／join／traverse／aggregate等のOperatorが合成できる場合でも、RDBのbag/JOIN・SQL NULL、Graphの巡回とCycle、Treeの子探索、Streamの未完全集合では、保持できるObserverが異なることを示す。特殊なSQL/native Queryは明示Bindingで保ち、Queryの正本とLineageを失わない（#2387）。ProviderにPushdownした演算がローカルの結果と同等とされるのは対象Scope/Oracleで観測した場合に限る。書込みは#2500/#2597のTyped Intentから既存Owner Action/Hostへ接続し、read-only Derived Datasetや集約表から暗黙の全件置換を起こさない。

### 24.5 Markdown・表テスト・Studio／SHOへの投影

#2621に合わせ、自由文章、モデル定義表、Function表、独立Expected表、単なる例示、コードフェンス、数式を、Profile・明示Role・Source位置によって分類する。表や式の形が同じという理由だけで意味定義・テスト・実行を同一視しない。Source／出典・Meaning／受理・Expected／独立Oracle・Target実測は別。

同じ安定Subject ID/版から、標準Studioの表・図・式、CUI構造説明、コード/Target、SHO文書、Scenario Example/Testを出しても、**生成成功と各Consumerのactual read/load成功は別**（#2497/#2516/#2604）。文書上の同一TEI参照の複数Occurrenceと局所説明上書きを区別し、意味への変更は受理済みSourceの更新経路へ渡す。文書の見た目とモデルの業務意味の一致も別検査とする。

### 24.6 既存Ownerへ渡す具体的実証

| 担当 | 最小の反証可能な問い | 資格 |
| --- | --- | --- |
| #2620 | Function/Model/Rule/Test/Projectionの5定義を標準構造とRoleで説明できるか。複数Role・循環・版衝突の負例 | 設計／未実行 |
| #988/#1018/#1063 | 関数形・Pipeline・Markdown演算表が同じ型付き束縛へ到達する条件、異なるときの診断 | 設計／未実行 |
| #2621 | 表の行順・Markdown装飾・ファイル移動でSource位置が変わっても意味IDを保持し再解決できるか | 設計／未実行 |
| #2622/#2387/#2506 | RDB・Graph/Tree・RESTで演算を合成する場合のbag/grain/NULL/Scope/Provider差の報告 | 設計／未実行 |
| #2497/#2500/#2597 | 集約・Display Projectionの情報損失を説明し、Typed Edit候補の必要前提を示すか | 設計／未実行 |
| #2514/#2586/#1174 | 独立Oracle→同版Headless→外部Host Effectの証拠が段階別に示されるか | 設計／未実行 |

**現状**：他チャットおよびGitHub #2619〜22の横断設計をメタ概念空間へ接続した**採用前の設計比較**。Role解決、構文同等性、Provider pushdown、Studio/SHO/CUI実読込、外部Host、Ownerの意味受理は**NOT RUN**。本#2495で新Parser・Runtime・全用途のOperation Engineを重複実装しない。

## 25. 2026-11-02 同じ表示からの判断を成立させる依存閉包・Provider Cohort・意味束縛

Everything is Dataの横断実証が進んだ結果、**表記構造が同じこと、表示結果が同じこと、Rule/Actionの結論が同じこと、実Providerが同じ対象を読んだこと**が相互に独立だと分かった。以下は[Syntax正規化の隔離Reference／Draft #2625](https://github.com/hat47x/tei-platform/pull/2625)、[明示Function MappingとRule依存の隔離Reference／Draft #2624](https://github.com/hat47x/tei-platform/pull/2624)、[複数Provider経路の隔離Reference／Draft #2627](https://github.com/hat47x/tei-platform/pull/2627)の**独立して検証済みと報告された範囲**をつなぐ、追加の契約候補である。三PRの試験件数を一つのTEI統合試験PASSへ合算しない。

### 25.1 同一性は段階ごとに照合する

| 段階 | 同じと言うための対象 | 同一性を保証しない条件 |
| --- | --- | --- |
| **Source表現** | Document/Scriptのファイル・Occurrence・Span・Revision | 表記が違っても構造候補は同じ場合がある |
| **標準データ候補** | Record/Ref/Application、明示Role/Operator ID/版、引数と順序 | CandidateのByte一致はOwner意味受理や実行許可でない |
| **受理済み意味** | Semantic Subject・型、Rule/演算の契約、Meaning Version | 同じ意味でも各Consumerの入力取得Scope/版が異なり得る |
| **Function利用文脈** | 起点、Relation Path、Field/Grain、Read/Rule/Actionの用途 | 同じ画面を表示してもRuleに必要な非表示Dataが異なり得る |
| **実Provider Read** | route、Binding、Query実効Scope、取得Slice、Cohort、完全性 | 同じBusiness ID/同じ数値Revisionで異Providerの同一Snapshotは証明できない |
| **Consumer結果** | Claim、Visibility、使用したArtifact set、実Read・Disclosure | 同じ文字列や画面表示で同じClaim・実Host Effectとはいえない |

この表は新しい公開Canonical／統合Registryを提案するものではない。既存Source Index、Role、Function Context、Operation Surface、Artifact/Providerの**現在の責務を分けて参照するための受渡し票**である。

### 25.2 Display Read、Rule Dependency、Action Requirementを別のFootprintとして扱う

あるFunction Fで、表示に必要な**データ依存**を **D(F)**、受理済みRule Rの評価に必要なデータ依存を **V(F,R)**、Operationの事前条件を検査するために必要なデータ依存を **A_data(F,Action)** と呼ぶ。これらとは別に、**更新対象・Effect範囲 A_write**、**Host/Authority/Receipt等の実行条件 A_exec**を保持する。集合記法は概念を比較する仮記号であり、正式な依存グラフ/Query Plannerの実装宣言ではない。

- Dは表示に使う値と構造の**宣言された**範囲。#2506のFunction Data Contextが明示し、存在するRelation全てを暗黙取得しない。
- VはRuleが評価する条件のデータ依存。**Dと同一である必要はなく、Dに含まれるとも限らない。** #1129/#1729から必要なField、Relation、時点、NULL/completeを追跡する。
- A_dataは対象ID、関連する更新不変条件、保存時に再読取が必要な入力等の**データ要求**。A_writeは更新するSubject/Relation/FieldとEffectの対象、A_execはbase revision、現在Grant、Host/Provider、Commit/Receiptなどの**データ以外の実行資格**。Vを満たすPreviewが、保存時点でA_execを満たす証拠にはならない。
- **同じ種類のデータ依存だけ**を集合として扱えば、D∪V∪A_dataは必要な入力の*論理的な候補集合*になる。ただし、実Providerでの取得やClient/LLMへの送出を許可するものではない。A_writeやA_execをデータ集合に混ぜない。実効Read Grant、Ruleの評価主体、結果/診断の開示Policyは別途評価する。

**合成反例（別PRの隔離Fixtureに対応）**：表示対象は Order.amount=600 のみ。非表示の Account.credit_limit が700の世界と500の世界では、表示Projectionが同じでも、受理済みの「amount ≤ credit_limit」Ruleを実際に評価できる条件なら判定は異なる。Dしか見ないParser/Studio/AIが「同じFunction状態なのでAction Offer・Validationも一致」と断定すれば誤り。反対にRule結果が非公開のConsumerへ「閾値超過」「隠し上限との差」を返すことも不適切となり得る。既存Host/Policyに許可された結果だけを送る必要がある。

### 25.3 必要データの宣言、実Readの完全性、現在Ruleの適用可能性

- **Declared**：Function/RuleがどのScope・Field・Grain・必要な完全性を要求したか。全件要求は全件観測の証拠と異なる。
- **Bound**：実Provider/Mapping/Scope/Version候補がどれか。Candidateが1つでも当該Routeへの実Grant、Source/Time整合、最新データを保証しない。
- **Observed**：各Readが実際に返したBag/Identity/NULL・Absent/Redacted、Complete/Partial/Unknown、Snapshot/Read-set等のEvidence。主張されただけの共通Cohortは実共有Snapshotでない。
- **Evaluated**：当該Data/Rule/Operator版と評価時点に対する結果、未評価・Rule失敗・入力資格不足・業務拒否の区別。
- **Delivered**：表示・診断を公開できるConsumer/Scopeと実読込の証拠。CUIに渡せたことからSHOや外部Harnessへ自動開示しない。

データ取得の**経路が複数**なら、同じSemantic SubjectでもProvider/Endpoint/Mapping/Mapping Revision/Scope/Purposeが異なる候補は区別する。Alias名が違っても**同一物理Routeだと確認された場合の**グループ化は可能だが、辞書順先頭のAliasへ寄せて別候補を失わせない（#2611）。ReaderのReplicaとWriterのPrimaryではSnapshot・Effect対象が違い得る。

二つのProviderで観測した値が一致した場合も、別Cohort、異なるRLS/Query Binding、partial、bag重複、Unknownのいずれかがあれば「同じ全体集合が読まれた」と断定しない。逆に不一致値だけからデータ競合と判断せず、比較可能なScope/Time/Query/Grainの資格を先に確認する。

### 25.4 局所Referenceをつなぐ最初の実証（同一試行で行う必要がある）

| 票 | 固定する入力・観測 | 同じと誤るもの | 既存Owner |
| --- | --- | --- | --- |
| IF-01 | Markdown表・Chain・Function等を同一Operator/Role候補へ解決 | 表記候補同値から受理済みMeaningや実Rule実装まで同一と断定 | #988/#2620/#2621・PR #2625 |
| IF-02 | Order.amount=600を同じ表示にし、Credit Limit 700/500だけ変える | Display Footprintが同じなのでRule結果も同じとする | #2506/#1129/#1729・PR #2624 |
| IF-03 | Ruleの非表示Fieldだけ欠落、または部分取得 | 入力不足をRule false／業務拒否と断定 | #1729/#2586・PR #2624 |
| IF-04 | OrderとCreditを別Provider/Replicaから読む | 数値のversionが等しいから共有Cohortと断定 | #2611/#2506/#2622・PR #2627 |
| IF-05 | 同一Semantic Subjectへ2つのMapping/Route候補 | Operation ID一致で実Write対象を無音統合 | #2611/#2598・PR #2627 |
| IF-06 | 未Mappingの秘密Fieldだけ変え、Ruleへの依存は別に扱う | 表示不変性を実OS／DB非読取や全機密非干渉の証明とする | #2516/#2515・PR #2624 |
| IF-07 | ViewからTyped Intentへ引継ぎ後、保存直前にRule/Grantが更新 | 読取り時のPreview成功を現在のCommit許可へ昇格 | #2500/#2597/#1174 |
| IF-08 | 同じ受理SourceをSHO/AI向け成果物へ出す | Artifactの生成成功から同一版の実読取・公開可まで断定 | #2604/#2630/#2516 |
| IF-09 | 異種Providerの同一Queryをpushdown／local評価 | 一部Fixture一致からNULL/Bag/Collation/RLSの全入力同値を主張 | #2622/#2387/#2498 |
| IF-10 | 関数のContextを限定してActionを取り消す/保存する | 前提のread-only説明からOwnerの業務完了を保証 | #2598/#1174/#2514 |

**最初の最小縦断**はIF-01→IF-02/03→IF-04/05→IF-07→IF-08。#2586がOwnerとして同じ固定Source/Rule/Model/Build版、実Provider Input/Output、実Context Instance、実Consumerを一枚のEvidence Packetへ接合する。#1119のG1/G2を新しい横断Gateへ置換しない。実Provider／Hostを接続していない段階では、上流の独立合成PASSを加算して「製品上の縦断が成立」と宣言しない。

### 25.5 実際に存在する証拠の使い方

| 独立Reference | PRに記録された限定実測 | 当該成果からは未実証 |
| --- | --- | --- |
| [#2625](https://github.com/hat47x/tei-platform/pull/2625) | 限定Syntax→plain-data Normal FormのPython 23/23 PASS、複数表層の有限正負例 | 正式TEI Language Plugin、全DSL、Owner意味受理、実Target |
| [#2624](https://github.com/hat47x/tei-platform/pull/2624) | 明示Data Mapping 20/20、Rule依存の別Fixture 18/18 PASS（いずれも隔離Python） | 本物のTEI Rule/Context、実Providerの取得制約、実OS/LLM開示、実Action |
| [#2627](https://github.com/hat47x/tei-platform/pull/2627) | Provider Route 12件＋Read Slice比較20件、隔離Go計32/32 PASS／race/vet報告（RC-18〜20で行集合完全性とField観測を分離） | 実TEI DeriveOperationSurface、Provider共有Cohort、実DB/RLS、Host Effect |

これらは**異なる試験対象の局所Reference**である。単に23+20+18+29という件数を加算して統合機能が90件通過したとは表示しない。各PRに記載された試験機・版と、現在のTEI mainでの実行結果も同一資格ではない。合流時はテストを実際のTEI Source→Context→Rule→Provider→Action/Consumerへ接続し、各段階で支持できた範囲と未取得の証拠を別々に記録する。

**採否**：#2620は意味Role、#2621はAuthoring、#2506はRead Mapping、#1129/#1729はRule依存、#2611/#2622は多Provider、#2497は保存性、#2516はDisclosure、#2586は縦断、#1174はEffectを所有する。本#2495は**型・Facet・動的意味の説明から、なぜ同じStructureでも利用資格が違うかを示す**。新しい共通Footprint Runtimeや独立Meta Schemaは先に設けない。IF-01〜10の正式製品試験、Provider/Host/SHO実観測、Owner意味受理は**NOT RUN**。

## 26. 2026-11-02 観測完全性は多次元：要求・物理取得・開示・Action適合を別に証明する

[複数Provider Readの隔離Reference／Draft #2627](https://github.com/hat47x/tei-platform/pull/2627)は、初回の12＋17件から**12＋20件、32件の合成Go Reference PASS**へ増補された。同ReferenceのRC-18〜20では、完全なRowsetであってもFieldが未観測（absent）なら値の同値は未確立とするよう修正した。観測済みSQL NULLと、値の取得・定義・可視性が未確立なabsentを別扱いにする。以下は[Semantic Context Action設計 #2616](https://github.com/hat47x/tei-platform/issues/2616)のI0〜I5、[Function Pattern A/G・Action Offer #2613](https://github.com/hat47x/tei-platform/pull/2613)の隔離合成42件と、既存#2506/#2516/#2622に対する**共通の反証可能な設計問い**である。新しいプライバシーEngineや汎用Read Plannerを導入した主張ではない。

### 26.1 三つのFootprintと、Rowset/Field/Timeの成立を分離

| 比較対象 | 内容・必要な証拠 | 早合点してはいけないこと | Owner |
| --- | --- | --- | --- |
| **Logical Required Footprint** | Functionの明示Mapping（表示D）、Rule依存V、Action前提データA_data；起点/Path/Field/Grain/Scope/時点 | この宣言があればProviderが実際にその範囲だけ取得したという主張 | #2506/#1129/#1729 |
| **Physical Acquisition Footprint** | Query Worker/Providerが実際に読んだRow/Column/Relation、実Alias/Mapping、RLS/Scope、Fetch/Cache/Log/転送経路と実行版 | 出力JSONから物理Readの最小性を証明したり、Provider自己申告だけで実効Authorityを証明 | #2622/#2611/#2387/#1769 |
| **Consumer Disclosure Footprint** | Consumerごとに実際に公開したData/Action名・候補数/診断/Trace、実Recipient/転送・開示Policy | 値を隠しただけでActionの存在・否定理由・件数まで秘匿済みと推論 | #2516/#1178/#2515 |
| **Rowset Closure** | 対象Query/Scope/Grain/時点に対し全行/Bagを観測した根拠 | 行集合Completeから全Field値・全Relation・他Providerの完全性を推論 | #2506/#1706/#2627 |
| **Field Observation** | 各Fieldの観測済みValue / 観測済みNULL / absent・unknown / withheldの根拠、Field単位のSchema/Provider条件 | Rowset Completeかつ両者absentから値が等しいと確定、NULLと未観測を同一視 | #2585/#2056/#2627 |
| **Cohort/Temporal Consistency** | Snapshot issuer/実Read-set/as-of、RuleとDataの版適用性、別Provider間の実共有Cohort | 同じSnapshot token文字列や数値Revisionを、別発行者の共有Snapshot保証とみなす | #2611/#2582/#2622 |
| **Action Offer/Effect Qualification** | Accepted Action identity/目的、Pattern A/G、Source/Context/Rule、実Provider/Grant/Commit/Receipt | Functionのread-only候補提示またはUI表示の成功をBusiness Actionの実行可と扱う | #2507/#2598/#1174 |

D∪V∪A_dataは**論理的な必要入力候補**であり、実際に何を取得するかは物理Provider/Rule実行主体が決める。同じ入力が必要でも権限上許されない場合は、未確立・非対応と適切な拒否を区別する。データが必要であるという説明から、そのデータのClient/外部Harness送出資格を派生させない。

### 26.2 結果値の「同じ」と、どこまで知ったかの「同じ」

Rowset Closure、Field Observation、Cohortは直列に全部一致させる単純なboolではなく、**比較対象とClaimが要求する情報に相対的な証拠**として扱う。例えば全Order行のIDが列挙済みでも、credit_limitが未取得なら、全件の与信判定を完了したとは言えない。片側が観測済みNULL、他方がabsentなら、それだけでは実値の矛盾を確定できない。

**比較の区分**：条件を満たす独立Fixtureで行・値が一致すれば「fixture範囲で一致」、同じ前提の下で観測済み値・Bagが矛盾すれば「反証」、必要な観測・共通Cohort・開示資格が欠ければ「未確立」。#2627のfixture_consistent / contradicted / not_establishedはこの限定的な参照であり、そのまま新しいCore status enumに採用する提案ではない。

さらに、Queryの全行取得と更新用の現在Snapshotは異なる資格である。RuleのSnapshot・Grant・対象Readが変更されれば、表示とPreviewが不変でもOwner Actionの保存条件を再評価する。独立Readで最終Effectが確認できない場合は成功へ昇格させず、Unknown Commitと再送の副作用防止は#1174/#1130へ委譲する。

### 26.3 開示の非干渉はValueだけでなくAction存在・診断にも及ぶ

[Pattern/Action Offer独立Reference #2613](https://github.com/hat47x/tei-platform/pull/2613)は、20件のA/G反例に、22件のAction applicability正負例（計42件隔離Python PASS）を追加し、Consumerに公開できるActionのIDを信頼済みfixture入力として扱う。**Action候補として内部に存在すること、現在の条件に適合すること、Consumerへ候補名の存在を知らせてよいこと**は別。

秘密Actionが存在する内部世界と存在しない世界で、公開Consumer向け候補一覧に差が出れば、Action名そのものを明示しなくても候補数・順序・ステータスが機密の手掛かりになり得る。さらに実Providerが余分な秘密Dataを取得しているかどうかは、公開結果が同じという二世界実験だけでは分からない。このため物理Readトレース・公開結果・外部送出の各観測は別受入とする。

**二つの非干渉候補は別**：Output Noninterference（秘密だけを変えた二世界で、低開示Consumerの可視出力が変わらないか）と、Acquisition Minimization（宣言した必要範囲を超えてProvider/Query/Cache/Logが実際に読まないか）。前者の有限試験がPASSしても、後者の物理的抑制やタイミング／反復照会の情報非干渉の一般証明にはならない。

### 26.4 追加の判別票（既存Ownerへ配る。第二Scenario正本ではない）

| ID | 独立して比較する二つの世界・観測 | 誤った成功判定 | 主Owner |
| --- | --- | --- | --- |
| QF-01 | 両者Rowset Complete、同じFieldがabsent | 全行が読めたのでField値も等しい | #2627/#2585 |
| QF-02 | 一方FieldのNULL観測、他方absent | NULL≠absentからデータ矛盾が確定 | #2627/#2056 |
| QF-03 | 同一Semantic Subject・同じSnapshot token、別発行Provider | token文字列一致で実共有Cohortを認定 | #2611/#2622 |
| QF-04 | 類似する二Read、片側だけRelation全件を取得 | Partialを全件不存在・全件一致へ昇格 | #2506/#1706 |
| QF-05 | 表示値同じ・非表示Rule入力だけ変更 | Displayの一致からAction/Validation一致を主張 | #2624/#1729 |
| QF-06 | 同一公開結果、秘密FieldをProviderが過剰Read | 公開JSON一致から物理アクセス最小化を証明 | #2622/#2515 |
| QF-07 | 公開可能なSubject同じ・秘密Actionだけ追加 | Action候補数/Status差から内部Actionを漏らす | #2613/#2516 |
| QF-08 | 旧Rule入力でPreview成立・保存直前Rule/Grant更新 | Preview成功を現在の保存許可とする | #2598/#1174 |
| QF-09 | Field単位では比較可能だが別ProviderのRLS実効範囲が相違 | 値一致で全件・同Scopeと断定 | #1769/#2622 |
| QF-10 | 全必要Dataは揃うが外部ConsumerでAction名の公開不可 | Capability/Action存在をConsumerの適用候補に無条件掲載 | #2507/#2516 |
| QF-11 | 同じOutput Data、表示/Rule/Actionの依存粒度が違う | 一つのSource revisionで全Claimの鮮度一致を断定 | #2582/#2593 |
| QF-12 | 同じSourceでSHO用生成物の一部未公開 | Producerの成功をConsumer全ArtifactSetの読込へ昇格 | #2630/#2604 |

**Owner接合順**：#2619/#2620 Source/Role → #2506 Required Context/Result Slice → #1129/#1729 Rule依存 → #2611/#2622 実Provider/Read/Scope → #2507 Action Offer → #2516開示 → #2598 Request/現在条件 → #1174 Effect/Receipt → #2630/#2604 Consumer実Read。#2586が同一試行で接続し、#1119の既存Gateを使用する。QF-01〜12は**検査案／正式TEI製品上でNOT RUN**。各独立PRの合成PASSは組み合わせた実Product PASSではなく、Owner意味受理も未取得。

## 27. 2026-11-02 横断Claimの合成・選択的再資格化・未知の意味の保持

§22の単一Contextに対するClaim寿命、§25のDisplay/Rule/Provider依存、§26の観測閉包・公開資格を、**TEIだけでなくSEIの判断、TAI/外部HarnessのTool利用、SHOの文書成果、EKIの配送**などへまたがる*一つの目的の達成Claim*に接続する。[TEI #2582](https://github.com/hat47x/tei-platform/issues/2582)の変更影響、[#2516](https://github.com/hat47x/tei-platform/issues/2516)の用途相対ClaimとDisclosure、[#2630](https://github.com/hat47x/tei-platform/issues/2630)のConsumer相対ArtifactSet、[#2600](https://github.com/hat47x/tei-platform/issues/2600)のBuild Evidenceと現在資格を再利用する。横断のClaim説明を追加することで新しい全製品共通Claim Store、権限Engine、Host、統合Schedulerを必須にするものではない。

### 27.1 「各部分が成功」と「組合せが現在有効」は異なる

概念例：「受理済み注文Ruleに照らして現在の注文を判断し、その理由を許可された資料へ記し、実際の配布先へ届けた」という複合Claimを考える。

- TEIのFunction/Query・Rule評価が、**同じSource/Meaning/Rule/Context/Provider Cohort**で成立したこと。
- SEI/AIの判断が、どの限定された根拠を利用し、何を仮説・未確認として残したかを区別していること。
- TAI/外部HarnessのSkillやToolが、単に配布・宣言済みというだけでなく、呼出し対象、現在Version、Tool/Hostの実権限、実行結果が同じ活動に結び付いていること。
- SHOへ渡す説明がそのConsumerへ開示可能であり、期待されたArtifactSetの同一公開世代を実際に読めたこと。
- EKI等の配送が、あるAttemptについて発生・受領されたことと、対象の最終業務結果が成立したことを別の観測として保持していること。

個々の合成Reference、Skill配布、Manifest検証、Action Preview、配送Receiptは、**それぞれが証明する事実の範囲で再利用**する。複合Claimの合成には「同じ活動/対象に属する」「版とSnapshot/時点が整合」「必要前提を満たす」「対象Consumerへの開示が現在許可される」「必要な外部観測を実際に取得した」などの*追加の接合根拠*が要る。

### 27.2 受渡しの比較項目：資格と版を一つの数字へ畳まない

以下は既存Issue間の受渡しを比較するための列であり、**提案APIやグローバルSchemaではない**。

| 証拠の役割 | 例として固定する識別・版 | 実測・受理の境界 |
| --- | --- | --- |
| **Meaning/Source** | Semantic Subject、Source locator、Accepted Meaning revision、Owner Acceptance | Source byte一致と業務意味受理は別 |
| **Rule/Context** | Rule/Operator ID・版、Function Context Instance、Read/Rule/Action依存、Selection epoch | 表示同値とRule成立条件は別 |
| **Provider/Observation** | Provider/Mapping/Query binding、Snapshot issuer/Cohort、実Rowset/Field completeness、Access Witness | Read要求・実取得・Consumer公開の範囲を別列に |
| **Action/Host** | Typed Intent、Execution Request ID、Current Grant/Authority、Commit/Receipt、独立Read | Candidate/Preview/実Commit/最終Effectを別資格に |
| **Skill/Tool** | Skill Package/Manifest、実ロードPlugin、Tool/CLI target・args・Binary、実呼出Attempt | インストール済み・Tool発見済み・実行可能・実行成功を区別 |
| **Document/Artifact** | Source revision、Generator/Template版、Expected ArtifactSet、Manifest/digest、Consumer actual read | Manifest整合・期待集合充足・同一公開世代・Readerの受信を区別 |
| **Disclosure/Egress** | 実Principal、Audience、現在Policy/Grant、公開したValue/Action名/診断、送出先の観測範囲 | 内部で成立したClaimと外部説明の資格は別 |
| **Cross-product handoff** | 同じ活動・対象・前提・入力/出力を指すリンク、Consumerが受理したEvidence | 異なるPRの成功件数と因果的に結合された一試行を区別 |

全列がすべての操作で必要なわけではない。**Claimが要求する前提だけを型付きに辿り、欠けた必須前提を示す**。別の論理版番号がたまたま同じ数値でも同じ実行時点とはいえない。HistoryとしてのAccepted/Receiptを保管することと、現在Audienceへその内容を開示できることも別。

### 27.3 再資格化は変更した依存と要求する観測から決める

概念上、各Claim Cに「読んだ／意味として参照した／評価に必要／開示した／実効果を要求した」などの**用途付きの依存**を結び付ける。変更Δを受けたら、関係が分かる範囲では到達する下流Claimの再評価候補を抽出し、到達しないことの根拠がある独立Claimは保持できる。依存が不透明／外部Providerが不明な部分は**未確立／再読取対象**として扱う。これは#2582/#2600への影響比較の考え方であり、強制的な新しいGlobal Dependency Graphを実装すると決めるものではない。

| 変更Δ | 維持を検討できる内容 | 再評価・公開制限を求める内容 |
| --- | --- | --- |
| Markdown装飾・行移動のみ | 安定Subject/Operator/Contextへ正しく再解決できる場合の受理済み意味 | Source span・文書Occurrenceの対応、局所書戻し対象 |
| RuleまたはOperatorの意味版 | 元のFact・歴史的な観測値 | 依存RuleのClaim、Action適用条件、必要なTarget再計算 |
| あるDataset/Fieldへの訂正 | 既知の独立DatasetのみのClaim | 訂正範囲に依存するResult、JOIN/AVG/TOP-N/NOT EXISTS等の非単調演算結果 |
| Provider Mapping/Snapshot/Read資格 | 意味Model/受理済み定義そのもの | 新Read Slice・Cohort・実Alias・Rowset/Field完全性を使うClaim |
| Action Context epoch/Grantの変化 | 元のReadという歴史事実 | 現在のOffer、保存時のRule/Grant、未完了Request/実Effect |
| Skill実ロード・Tool対象/版の変更 | 過去のSkill文書・過去の実行Receipt | 現在そのToolを呼べるか、出力と公開条件が満たせるか |
| Consumer/Audience・Policy変更 | 計算済み内部根拠、歴史的なReceipt | **現在の送出・提示資格**、Tool候補/エラー/件数を含むDisclosure |
| Generator/ArtifactSet・公開世代変更 | 意味が不変と独立確認されたSource/Rule | 必要Memberの充足、同版実Read、署名/内容差分、SHO/TargetのConsumer実受入 |
| 未知の新Interpretation Role | 元データ・Role/Attribute/Sourceの所在 | 新Roleの操作適合・編集逆写像・実行効果。標準Studioではread-only表示・診断を優先 |

**完全性と単調性の注意**：肯定の存在証人などは追加に対して維持し得る場合がある一方、NOT EXISTS・TOP-N・LEFT JOIN未一致行・AVG/中央値・全件Claimは追加だけでも失効し得る。Snapshot/Field観測の不確かさやRuleの負例を単なる数値比較の問題へ押し込めない。合成Fixtureでの局所一致と一般のデータ更新下での保存可能性を区別する。

### 27.4 資格判定の複数軸と、公開側での説明

内部の比較票には、少なくとも次の**別の問い**を残す：①Meaning/演算が適用可能か、②必要な観測が十分か、③いまの時点で新鮮か、④実Host/Toolへ到達できるか、⑤そのConsumerへ出力してよいか、⑥実結果を観測したか。

「十分に確認された」「反証が見つかった」「必要な証拠が欠ける」「時間や版の変更で再確認が必要」「そのConsumerへの開示が許可されない」を、内部では根拠付きで分ける。ただし**低開示Consumerへ内部の失敗理由・隠れた入力件数・秘密Actionの存在をそのまま伝えない**。内部理由を伏せる場合も、正しい値がないことと情報の不存在を混同しない。

標準Studioへ未知Roleが来た場合は、構造・元Source/属性・未対応Reasonをread-onlyで提示する候補を維持する。[Draft #2631](https://github.com/hat47x/tei-platform/pull/2631)のopen-world標準UI有限JS Referenceは16 assertions PASSと報告するが、正式TEI Studioの実装済み機能とみなさない。未知Roleを解釈できるようになっても、Sourceからの逆写像・Actionの許可は改めて確認する。

### 27.5 横断の反証・Evidence接合票

| ID | 変更／合成パターン | 誤った総合判定 | Owner |
| --- | --- | --- | --- |
| CQ-01 | TEI Readは成立、Ruleの非表示入力が別版 | 画面が同じなのでSEI/SHOの判断も現在正しい | #2506/#1129/#2586 |
| CQ-02 | CUI表示は不変、Providerだけ秘密Fieldを過剰Read | 公開JSON同値を実物理アクセス最小性の証拠とする | #2622/#2515 |
| CQ-03 | Rowset Complete、Credit Field absent | 全件取得なので全Rule判定済み・全件Claim成立 | #2627/#2516 |
| CQ-04 | Skill配布・Tool一覧は存在、実ロード先またはGrant失効 | 既存Skillと選択したToolだけで現在の実行権限成立 | #2605/#2515/#2504 |
| CQ-05 | TEI/SEI/TAIの個別合成PASS、Source/Activityが異なる | 別試行のPASS件数合算で一業務成果を完了扱い | #1119/#1163/#2586 |
| CQ-06 | Rule変更・独立データ集合D1のみの表示Claim | 全Claimを失効、またはRule依存Claimを保持 | #2582/#2600 |
| CQ-07 | 追加Eventで旧NOT EXISTS/LEFT JOIN未一致が変わる | 追加のみなので旧全件Claimを保持できると確定 | #2582/#2593 |
| CQ-08 | Consumer Audience/Grant失効後に過去Receiptを配送 | 過去の成功Receiptを今の送出許可として利用 | #2516/#2515 |
| CQ-09 | Skill/Toolに新しいRole、StudioはRoleを解釈できない | 未知Roleを破棄または編集/実行可能な既知Roleへ偽装 | #2620/#1124/#2631 |
| CQ-10 | 同じSourceへSHO/Targetを生成、Artifact Memberが1つ欠落 | Manifestの内容自己整合から期待一式の実Readが完了 | #2630/#2604 |
| CQ-11 | EKI配送AttemptのACKが不明、外部結果は同値 | 結果値一致を同一AttemptのEffect/配達完了Receiptと断定 | #1174/#1163 |
| CQ-12 | UI・Policyだけ変わり、純粋計算結果・受理済みMeaningは同じ | 一律全Target再実行または現在公開資格を無検証継承 | #2600/#2516 |

これらは既存#2582のRCケース、#2516のW8〜W10、#2630のAS、#2600の増分Build/資格へ**担当を渡す設計検査票**であって、第二のScenario SSOTではない。最初の共同実証は合成の受注額600／与信上限700対500／部分Provider Read／秘密Action／SHO公開版を固定し、**実際に同じ活動・Source/Rule/Mapping/Grant/Consumerを指す証拠**を一回の試行で接合する。全再評価と依存選択的再評価の結果差、誤維持、過剰失効、Provider/Tool再呼出数、必要時間を比較する。

**現状**：横断比較・実証計画のみ。TEI製品のClaim合成、外部Harness Tool実呼出、SHO実Consumer/EKI実配送、OS/DBの機密強制、正式Owner意味受理は**NOT RUN**。#2631、#2628、#2627、#2613等の隔離PASSは各PRの固定Fixtureに限る。採否は各Ownerが証拠に基づき判断する。

## 28. 2026-11-02 非出力依存の保存：値Lineage・結果成立・選択・順序・Current Authority

[Query Explain #2387](https://github.com/hat47x/tei-platform/issues/2387)から[Go DerivedIR特性記述 Draft #2634](https://github.com/hat47x/tei-platform/pull/2634)と[Draft #2635](https://github.com/hat47x/tei-platform/pull/2635)が並行提案され、[Function Mapping/Rule入力 #2624](https://github.com/hat47x/tei-platform/pull/2624)、[増分再利用手順 #2636](https://github.com/hat47x/tei-platform/pull/2636)、[D2読取Grant再確認 #2637](https://github.com/hat47x/tei-platform/pull/2637)の具体的接続点が明らかになった。既存§22/25〜27の「依存と現在資格」を、Queryの**値に表れない選択根拠**へ拡張する。新しい公開Query Schemaや第二の解析器・Cache/Permission Engineの導入ではない。

### 28.1 同じOutputField.Value lineageでも、同じResult Claimとは限らない

合成Query（構文の説明例でありTEIが全件実行できることの証明ではない）：

```sql
SELECT id
FROM item
WHERE secret_score > 50
ORDER BY secret_rank, id
LIMIT 1
```

表示値`id`の直接的な値Lineageは`item.id`だけに見える。しかし`secret_score`は**採用される行の集合**、`secret_rank`は**採用順序／先頭の選択**、`LIMIT`は**採用境界**を左右する。秘密列の値を変更しても`id`列の式とOutputField IDは不変であり得る一方、結果に現れるIDは変わり得る。したがって`OutputField.Inputs`が同じという証拠だけでは、Result Slice、Claim、キャッシュ再利用を認められない。

比較のため、依存の**役割**を分ける。これらは分析観点であって製品の固定Enumではない。

| 依存の役割 | 典型的な演算・根拠 | 変化すると何が変わるか | 担当 |
| --- | --- | --- | --- |
| **値の算出（Value）** | SELECT/計算式の入力、演算版、単位・NULL/Decimal | 既に選ばれた出力行の値 | #2387/#2593 |
| **行の成立・選択（Membership）** | WHERE、EXISTS/NOT EXISTS、JOIN条件、HAVING、RLS、Relation path | 出力行の有無／採用対象。表示値の式が同じでも変化 | #2387/#2506/#2622 |
| **行の多重度（Multiplicity）** | SQL bag、1:N Join fanout、DISTINCT、GROUP BY | 同じ値の出現数、SUM/COUNT/AVG等 | #1706/#2593 |
| **順序・境界（Selection）** | ORDER BY、LIMIT/OFFSET、Window、Tie-break | 先頭n件、順位、ページ。旧結果行の元値が不変でも採否が変化 | #1640/#2387/#2582 |
| **閉包・不存在の根拠（Closure）** | 反結合・NOT EXISTS、全部分集合の検査、完全なRead Slice | 不存在/全称のClaim。追加Eventで旧Claimが撤回され得る | #2506/#2516 |
| **実観測・適用時点（Observation）** | Provider実取得、Field observed/absent、Snapshot issuer/Read-set、Rule版 | その依存を実際に満たしたか・現在も使えるか | #2611/#2627/#2586 |
| **現在のRead/Disclosure資格（Authority）** | 実Principal、RLS、取得時/返却時Grant、Consumer向けPolicy | 実取得を許すか、値・Action名・存在/件数を外へ出せるか | #1178/#1769/#2516 |

**重要な限界**：値/行成立/順位等の区別は、現在のTEI `DerivedIR`に全種類の**型付きDependency Edgeが既に存在する**という意味ではない。#2634/#2635の合成`analyzer.Result`では、WHERE/ORDER/相関EXISTSの参照列やExpressionが別経路に保持される例がある一方、Value lineage `OutputField.Inputs`はそれらを全部含まない。「Expressionが保持された」「IR coverageがfullと記録された」から、実SQLGlot Worker・全演算・Reader/CUIへ必要な依存が伝わったとは言えない。

### 28.2 値の出所と、結果を採った理由は異なる「なぜ」の問い

例えば、`SELECT id FROM item WHERE score > 50`で結果が一件も出ない場合、出力値Lineageは空でも、**結果が空であることの成立条件**は`item`の対象範囲・score・Predicate・実Rowset Closureに依存する。今見えている0件は他Tenantや未取得Pageに対する不存在の証明ではない。

同様に、`NOT EXISTS`の「現在該当行なし」というClaimは、一行の追加によって偽になる可能性がある。正の存在証人は同じScopeへの追加だけなら保存可能な場合があるが、対象削除・時点・Authority/Visibilityの変更では再評価が必要になる。**演算の単調性・負の依存・完全性の前提**を明示し、単純に「前回の出力値が変わっていないからClaim有効」としない。

Query変更Δに対し次のように比較する：

- Value Expression不変でも、Predicate/Join/HAVING/Sort/Limit/Scope/RLS/Rule/Provider Cohortに変化があれば、それを必要とするClaimの再資格化候補にする。
- 実Readで観測できなかった項目は、観測済みNULLや0件に潰さない。Rowset CompleteとField Observedは別。
- 正当なSource/Operator依存が`Opaque`・未解決であれば、必要範囲での`unresolved`または全面再評価候補へ戻し、**依存がないという推定**でskipしない。
- Value/Selection/Closureの静的依存が揃っても、実Providerに実際に読ませたQuery、Execution/Authority、Consumer向け開示・実Resultの現在性は別の実証とする。

### 28.3 非出力列がIRに含まれるときの情報境界

AnalyzerやGo DerivedIRが`secret_score > 50`の条件文を保持することは、内部の構造Explain・影響解析に有用だが、**内部IRが持つ文字列・値・列名を低開示Consumerへ無条件に返してよいという資格ではない**。条件定数、秘密Column名、Predicateが存在すること自体が非公開の場合がある。

- 内部Read Footprintの必要性、**実Providerでの取得資格**、CUI/SHO/AI/外部Harnessへどの説明を開示するか、実送出先を別に記録する（§26、#2516/#2515）。
- 非公開Predicateの詳細を省略する際も、公開Consumerには**不十分な観測のまま真/偽を確定させない**。内部診断の具体的な欠落理由は現在Policyが許可する粒度で投影する。
- Query/Action実行前だけのRead Grant確認は、実行中の変更と現在の公開資格の問題を残す。[Draft #2637](https://github.com/hat47x/tei-platform/pull/2637)は同一AttemptのpreflightとProvider取得・評価後の返却直前にGrantを再確認する**限定的な二地点検査**。失効→復活するABA、返却後のConsumer egress、別Host/OS/RLS、分散Snapshotの一致はそれだけでは証明できない。実装PRが既存D2の安全拒否やProvider責務を変更せずに扱う範囲を尊重する。

### 28.4 「再計算しなくてよい」の証拠を、独立Oracleで反証する

[増分Build比較計画 Draft #2636](https://github.com/hat47x/tei-platform/pull/2636)のF0 baseline／F1 changed full／S1 selective／S2 negative controlsを再利用する。**Byte一致、値Lineage一致、Artifact cache hit、既存のRun成功状態**のいずれも、選択的な再計算・Claim再利用の資格とは別。

初回はQuery・Rowset/Field/Rule/Authorityを一つずつ変え、**結果値は不変でもMember/Rank/Complete/Disclosureが変わる**正負ペアを作る。F1 fullの出力だけをS1 selectiveのWantへ複製すると、同じ解析器/Generatorの欠陥を共有する恐れがある。[独立Oracle #2514](https://github.com/hat47x/tei-platform/issues/2514)へ、手計算・独立評価・原典Scenario/Accepted Ruleから期待を立てた根拠と、Candidateとの共通由来を明示して渡す。

[Draft #2634](https://github.com/hat47x/tei-platform/pull/2634)と[Draft #2635](https://github.com/hat47x/tei-platform/pull/2635)は**同じQX-27について並行したGo Characterization**であり、互いを独立Oracleと数えない。両者の重複する観測を比較したうえで#2387のOwnerが一つの受入経路へ整理し、Coverage/相関/非出力値・誤表示の**不足する負例**だけを残す。両PRとも現時点ではDraftで、片方の成功はもう片方や実SQLGlot→Goの成立を保証しない。

### 28.5 次の実証票（既存Ownerで実施）

| ID | 固定条件・単一変更 | 誤った再利用／誤成功 | Owner |
| --- | --- | --- | --- |
| LQ-01 | SELECTの値Inputs不変、WHEREの閾値だけ変更 | 値Lineage ID一致→同じRow membership | #2387/#2634/#2635 |
| LQ-02 | 非表示ORDER BY列の順位だけ変更、LIMIT=1 | 出力Value Expressionが不変→最上位行も不変 | #2387/#2582 |
| LQ-03 | NOT EXISTS用Relationへ1行追加 | 出力値を変更していない→旧不存在Claim維持 | #1706/#2582/#2516 |
| LQ-04 | JOINの関連Occurrenceだけ追加、値は同じ | 重複/AVG/COUNTに影響しないと判定 | #2593/#2387 |
| LQ-05 | SQL/IR値Lineage一致、AnalyzerのColumnRefが一部欠落 | Lowering=fullフラグだけで全Dependencyが網羅済み | #2387/#2514 |
| LQ-06 | Field取得先がabsent、RowsetはComplete | 0/NULL扱いで不存在ClaimやRule判定を受理 | #2627/#2506 |
| LQ-07 | 内部IRが秘密Predicateを保持、低権限CUIにExplain | 内部IRを生出力し、秘密列/閾値を漏らす | #2516/#1178 |
| LQ-08 | Provider取得中にRead Grantが失効 | preflight成功を返却時Authorityと断定 | #2637/#1174 |
| LQ-09 | Grant失効→復活ABA、または返却後の失効 | 二地点検査が全時間の継続的権限・送出資格を保証 | #2637/#2515 |
| LQ-10 | Query変更なし、Provider CohortまたはRLS Scopeを変更 | 同じQuery digestから同じ現在Result Claimと判定 | #2622/#2611 |
| LQ-11 | F1 fullとS1 selectiveが一致、Expectedが候補由来 | 実装とOracleに共通誤りがあっても独立受理 | #2514/#2600 |
| LQ-12 | FullとSelectiveが結果同値、依存不明・Cache Hit差 | 観測値の一致だけで全入力上の安全なskip/性能改善を確定 | #2600/#2582 |

実施の順は、**Q0** Source-static/型付きIRで値・非出力依存を分ける → **Q1** 実SQLGlot Worker→Go DerivedIRの実連結を検証 → **Q2** #2624のFunction Mapping/Ruleと実Provider/Rowset+Field/Cohortを照合 → **Q3** 低開示CUI/SHOへの説明・#2637の実Host/権限境界 → **Q4** #2636に沿うfull/selected・独立Expected・現在再利用資格と時間の比較。#2387はQuery構造Explain、#2582は変更影響、#2600は増分Build/計測、#2514はOracleの独立性、#2586は同版実一体試行を担当する。

**状態**：この節は独立Draftに存在する設計・局所Go試験の比較と、既存Ownerへの受渡し。LQ-01〜12の**同一TEI製品経路／実DB/Host/AI/SHO**の一気通貫検証とOwnerの意味受理は**NOT RUN**。業務Rule/SQLが解決済みだからと既存#1640 D2の安全拒否を解除しない。
