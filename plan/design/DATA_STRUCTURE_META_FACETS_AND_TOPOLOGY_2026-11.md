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
