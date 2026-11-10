# 構造保存写像・Projection・編集可能性の横断契約（設計候補、2026-11）

- **状態**：机上設計・反例収集、未採用、実装・数理証明・CUI/GUI/Target実行未実施。
- **Owner**：[#2497 構造保存写像とProjectionの意味忠実度・編集逆写像](https://github.com/hat47x/tei-platform/issues/2497)。
- **上流の概念空間**：[#2495](https://github.com/hat47x/tei-platform/issues/2495)、[メタ概念空間の7役割と多重facet](DATA_STRUCTURE_META_FACETS_AND_TOPOLOGY_2026-11.md)、[要求ポートフォリオ #2493](https://github.com/hat47x/tei-platform/issues/2493)。
- **開発順序**：[AI-first基本方針 #2492](https://github.com/hat47x/tei-platform/pull/2492)：概念と外部要求の机上評価を先行し、既存の構造を活用したCUI＋標準Plugin、ヘッドレス検証を経てUIに反映する。
- **既存担当**：#1120（Subject/Graph）、#1126（Entity/Relation）、#1806（Value DomainとTypeRef）、#1705（Quantityと数値）、#1706（Collectionの完全集合性）、#2387（SQLとLineage）、#1127（Function Data Context）、#951（Plugin Contract）、#997（UI Composition Kit）、#1124（Studio）、#1803（Document/SHO）、#1155/#2354（CUI）、#1180（Scenario）、#1174（権限・副作用）。仕様の正本と実装Ownerは各Issueに残す。

## 1. 構造のMECEから、構造間の意味保存へ

#2495では「構造の分類軸をMECEに整理し、具体例は複数facetを持つ」とした。ここで新しく問うのは、**同じデータが異なる構造・表現・UIへ移るとき、何を保持し何を失うのか**である。

二つの表現が似ている、同じbyte列を持つ、相互に変換できる、UIで同じ絵に見える、という四つの主張は区別する。意味上のSubject/Identity、構成（signature）、適合する法則（laws）、利用条件（use-site）、出所・改訂、具体的な実現Targetを結び、どの主張をどの証拠で支持しているかを明示する。

「一般にAはBより情報量が多い」という普遍的なデータ構造の全順序は仮定しない。**保持したい観測・不変条件をPとしたときに、AからBへの変換がPに関して忠実か**を対象・文脈・版付きで検査する。異なる目的では同じ写像に異なる評価が成立し得る。

## 2. 構造の変換・利用を異なる関係として扱う

以下は設計を比較するための**関係・義務の区分**であり、公開API、Core enum、万能な「Morphism Engine」の採用ではない。

| 役割／Relation | 何を主張するか | 検証したい義務 |
| --- | --- | --- |
| `is-a/refines` | 同じ意味文脈における包含・制約追加 | 親の不変条件と置換対象のScope |
| `has-facet` | 順序・量・時間等の性質への複数適合 | 依存演算と公理、矛盾の有無 |
| `structure-map` | Aの構成をBの構成へ対応させる | 対象と要素の対応、構成規則、同一性 |
| `preserves(P)` | 特定の不変条件・演算・観測Pを保存する | 正負例、適用範囲、必要なら証明 |
| `converts-to` | 値・量・座標・通貨等を別表現へ移す | 前提、外部参照、丸め、誤差、失敗 |
| `forgets(P)` | 順序・重複・根拠等の一部を落とす | 喪失するP、逆復元に必要な追加情報 |
| `encoded-as` | 物理表現を与える | Reader/Writerの契約、バイト・値同等性 |
| `projected-as` | UI/文書等で特定の観点を提示する | 粒度、可視範囲、選択、完全性、出所 |
| `putback-with` | 投影からAuthoritative Sourceへ変更を反映する | 更新対象、逆写像、Authority、競合 |

これらの関係は互いに合成可能であっても自動的に置換可能とは限らない。`encoded-as`が示す型互換と、`preserves(identity)`が示す意味の同一性を混同しない。例えば数学的なvector spaceでは、保存したい加法とスカラー倍を明示してはじめて線形写像として検査できる。単なる座標配列のコピーでは十分でない。

### 宣言の最小項目（概念的な比較表）

一つの変換候補について、次を対象の必要性に応じて説明できるようにする。

- **Source / Target**：Semantic Subject、型・構造、版、Namespace、表示や保存の単位。
- **適用文脈**：Target、利用者/権限、時間、CRS、Unit、外部レート、取得範囲、参照先。
- **構造上の前提**：順序、キー、一意性、cardinality、完全集合性、正規化規則、可変性。
- **保存する義務**：Instance Identity、値、頻度、順序、Relation、演算法則、Quantity、Lineage、作用結果等。
- **損失・誤差**：情報の忘却、丸め・近似、partial、非可逆、誤差境界、未対応。
- **検査手段**：形式的に導出した条件、独立fixture、Consumer差分、観測、受理、現在のAuthority。
- **不明事項**：元データ・入力完全集合性・第三者Contractが不明ならその理由と担当。

変換の「存在」と「実行可能性」、「必要な意味を保存する保証」はそれぞれ別の主張とする。

## 3. 情報保持の典型的な連鎖

### 3.1 順序・重複・集合性

```text
Sequence<T>
   │ forget(element order)
   ▼
Bag<T>
   │ forget(multiplicity)
   ▼
Set<T>
```

ここで`Sequence → Bag`は元の並びを失い、`Bag → Set`は出現回数を失う。元のデータが業務InstanceやEventを指す場合、同じ値の重複と別Instanceの同一性は別である。したがって「重複値を取り除いた」ことと「不要な業務レコードを削除した」ことは同一視しない。

一方、`Map<K,V>`はKeyによる対応という意味を持ち、Hash Tableはその実現候補。順序付きMapはMapに列挙順契約を追加することがある。これは構造特性を組み合わせれば説明できるが、MapとSequenceを一つの全順序で序列化する必要はない。

### 3.2 表示と物理エンコーディングの差

例えば列指向の物理表現は、論理的なList/Struct/Map/Unionと、その記憶レイアウトやDictionary/Run-End Encodingを分けている。Apache Arrow仕様は、型と物理レイアウト、拡張型、MapのList-of-Structsによる実現を説明しており、**一つの意味を複数の格納方式で保持する**具体的な比較元になる。これはTEIがArrowをCoreへ要求することを意味しない（[Arrow Columnar Format](https://arrow.apache.org/docs/format/Columnar.html)）。

### 3.3 演算の違う同形データ

- `Vector3<float>`という形の値が、空間ベクトル、色値、地理座標、確率ベクトル、位置のいずれなのかは別の意味に依存する。
- 温度の「点」と「差」では演算が異なる。点−点は差、点＋差は点だが、点＋点は一般の温度意味では通常の加算として扱わない。
- `Unit`と`Quantity Kind`を分け、単位表記の同一性から量の同一性を決めない。`UCUM`のコード／単位演算、`QUDT`の量種・次元・単位は相互参照の候補とする（[UCUM v2.2](https://ucum.org/ucum)、[QUDT Schema 2026-01](https://www.qudt.org/doc/2026/01/DOC_SCHEMA-QUDT.html)）。
- 通貨換算は外部の時点・レート・丸め方針を必要とし、数学的な次元一致だけから換算結果を導かない。
- 地理位置・空間座標は、軸順・座標参照系・参照時点等が揃って初めて距離・地図投影の適合を判断できる。

### 3.4 「同じデータ構造」を一つの同値性で済ませない

新しい構造が現れたときに似た型を二重追加しないため、**どの意味で同じか**を区別する。これは#2495の分類軸を増やす提案ではなく、比較結果が何を主張できるかの整理である。

| 比較する関係 | 何が一致するか | 具体例・境界 |
| --- | --- | --- |
| 名目的な同一性 | 同じSemantic Subject/Typeのidentity・参照版 | 名前や値の形が同じでも顧客コードと取引先コードは別の意味になり得る |
| 記述上の構造一致 | 構成子・要素型・ラベル・付随制約等の記述 | Product/Recordのフィールド順が意味的に無関係かは、定義に依存。Tupleの順序は意味を持つ |
| 構造同型 | 逆写像を持ち、指定した構造・法則が両方向に保存される | GraphのノードID付替えを同型と呼べる条件と、Source Identityの同一性は別 |
| 観測上の同等性 | 指定したConsumer・操作・文脈から区別できない | 列挙順を観測しないMap Consumerと、順序を要求するConsumerでは結果が変わる |
| 利用箇所への適合 | 特定Operationが要求する前提を満たす | 読取りの代入可能性と書込み可能なContainerの代入可能性は別 |
| 物理・符号化互換 | 対象Hostが読書き可能で同じ値を復元する | 同じ格納byte・Arrow物理レイアウトは業務上の同一性を与えない |
| 意味上の受理済み同等性 | 対象Domain/Scope/SourceのもとでOwnerが対応を受理 | 構造同型や自動テストPASSだけでは得られない |

この区分は、[OWL 2のStructural EquivalenceがSemantic Equivalenceとは異なる](https://www.w3.org/TR/owl-syntax/)点とも対応する。形式的に構造の比較へ用いる等価性と、TEIのSource/Accepted Meaningの等価性を区別する。

**正規化は、意味を保存することが明示された有限の規則に限る。** 例えば可換で順序が意味を持たない構成要素は並び替えて比較できる場合があるが、OrderedMap/Sequence、Source上のField順、UI配置、データ型ID、Quantity Kind、CRSの文脈は黙って正規化しない。同じ見た目を作る二つの構造が、本当に同じ正本の候補かは、変換・法則・適用条件の根拠が必要である。

任意の述語・再帰型・外部関数を含む等価性は、一般に機械的判定が困難な場合がある。比較結果は**支持できる範囲で**一致・不一致・未証明／未対応・権限判断待ちを分け、未証明を不一致とも同値とも扱わない。構造記述のCanonical Formが計算できても、それ自体がAccepted Meaningの新しいAuthorityになるわけではない。


## 4. 構造写像の合成と逆方向編集

写像`A → B → C`の各段で何を失うかが明確なら、最終的なConsumerの必要な性質について段階的に検査できる。例えば`Set`しか残っていないCに、元のSequenceの順序を要求しても一般に復元できない。

ただし、**合成の可否と保存する法則は前提依存**である。途中の写像が実データの未取得、各Targetの演算の非同等、丸め、異なるSnapshot等を含むなら、純粋な数学的合成をそのまま適用できない。Proof義務は、関心のある不変条件・対象版・前提に限定する。

### 4.1 見ることと書き戻すこと

```text
Authoritative Source (S)
       │ get / read
       ▼
Projection / View (V)
       │ user edit / action
       ▼
Typed Intent / Patch
       │ validate + current authority + revision + effect policy
       ▼
Authoritative Source (S')   → independent observation
```

可逆に近い純粋なViewの編集には、双方向変換（Lens）のRound-trip Lawを参考にできる。

- **GetPut**：`put(S, get(S)) = S` — 表示を変更せずに書き戻しても、元の意味は変わらない。
- **PutGet**：`get(put(S,V)) = V` — 許可した変更を戻し、再取得すれば編集後の表示に一致する。

これらは**法則として採用可能かを検討する候補**であり、すべてのUI/SQL Viewに成立する主張ではない。集計・結合・匿名化・ラベル丸め・不可逆なGraph→Tree表示などは原則として元のデータを一意に復元できない。編集が必要なら、元のSource Identityと明示的なAction、競合/現在Authorityを結ぶ。UIの自動保存からDB効果の成功を推論しない（参考：[Bidirectional model transformationsとLens法則](https://pmc.ncbi.nlm.nih.gov/articles/PMC8645541/)）。

### 4.2 UI/SHO投影に必要な最小共通情報

表示器の具体実装ではなく、次の問いに答えられるかを比較する。

| 観測項目 | 問い | 誤った成功例 |
| --- | --- | --- |
| Subject/Grain | 表示点・行・棒・ノードは何のInstanceを代表するか | 集約Groupを元Entityの行と同一視 |
| Element/Relation | ノード・辺・セルの同一性と参照元は何か | RelationのN:Mを単一Treeへ黙って縮約 |
| Cardinality/Completeness | 見えている集合は全体か、ページか、権限限定か | 0件表示で「存在しない」と説明 |
| Order/Time | 意味上の順序・sortキー・時点は何か | 描画順を業務上の先後関係と推論 |
| Quantity/Geometry | Unit、QuantityKind、CRS、尺度・変換は何か | 数値が同じなので違う単位を重ね描き |
| Transformation/Loss | 元モデルから表示まで、何を合成・欠落・丸めたか | Graphの省略した辺を元データから削除 |
| Interaction/Writeback | この表示から何を変更してよいか | chartのbarを動かせば元明細が更新されると仮定 |
| Evidence/Authority | この表示の主張は誰のどの根拠と版によるか | 画面上のvalidationだけで意味受理・実権限を宣言 |

CUIで同じSubject/Source revision/Contract/Plugin解決から構造説明を出すことを基線とし、GUI・SHOは新しい権限や正本を持たないProjectionとして接続する。Read-only説明とUI操作可能性を分離し、実装済みの標準/Native UIは#997に属する。

## 5. 構造的な例外を生かす多様なConsumer

### 組織・代理／委任

時点とScopeを持つ兼務・代理の組織関係では、多重の親や辺が実務上必要になる。権限の参照構造をTreeで描いた場合、複数の上長や委任関係の一部が表示対象外となり得る。そのTree上で親ノードを変更したとき、背後の全Relationが更新されたと推測しない。組織図の見た目、所属の意味、Hostの現在のAuthorityはそれぞれ別の契約とする。

### Canvas/KJ・アウトライン・マインドマップ

自由な2D配置、島・連なり、階層、分岐は異なる構造facetを重ねて表現できる。階層は意味上の親子参照として保持し、キャンバス上の近さや線の交差を業務上のRelationへ自動採用しない。折り畳み・代表テキストは元のカードと注記・違和感・根拠へ戻れる投影とする。自由な並べ替えによって親子のIdentityや未採用仮説を消去しない。

### 科学・数学・地理

ラベル付きTensor、座標付きフィールド、Mesh、行列、確率分布、位相的近傍を同じデータ型階層に潰さず、必要な演算・座標参照・独立した公理の主張を保つ。数学的位相空間はGraphの連結と異なる構造である。KaTeX表示は数式の可読化であり、型付き計算・公理・証明の代わりではない。

### 生成AIが扱う情報・文書

EmbeddingやGraphによる近接、推定した分類と、Sourceの厳密な同一性、意味的受理は別。生成された要約・図表・SHO文書により、元の不確実性や未受理仮説を事実へ昇格させない。非公開のSource/データをAIハーネスへ渡す際は、公開許可がある範囲の要約・参照のみを選択し、元の生データを露出させない。OS/Hostの資源・秘密情報の境界は既存のPolicy/Host Ownerへ委譲する。

## 6. 圧力コーパスを分野横断で拡大する（初期候補）

以下は**将来の比較対象**であり、実装優先の固定リスト・試験PASS・型名の採用ではない。対象数の固定上限は置かず、差異が見つかればPressure Caseを増やす。

| 分野 | 例 | 観測したい構造保存/損失 |
| --- | --- | --- |
| 一般コンテナ | Sequence↔Bag↔Set、OrderedMap、MultiMap | 順序・頻度・Key/Instance identity |
| 関係・DB | Relation、Query Result、JOIN、GroupBy、CTE | 粒度、fanout、NULL、Lineage、逆更新条件 |
| グラフ | Tree/DAG/Graph/Hypergraph、RDF、Knowledge Graph | 辺の方向・次数・共有参照・推論と宣言 |
| 座標・幾何 | CRS付きGeometry、Mesh、Trajectory、CAD形状 | 座標系・隣接・Topology・境界・精度 |
| 数学・数値 | Vector Space、Sparse Tensor、Matrix、Interval、Polynomial | 演算法則、疎性、ラベル、近似 |
| 量・測定 | Absolute/Delta、対数尺度、時刻/期間、Currency | 量種・単位・換算依存・丸め |
| 時系列 | Event Stream、Window、Late Arrival、Snapshot | event/processing time、完全性・順序 |
| 分散状態 | CRDT、Event Sourcing、Bitemporal History | 収束条件、因果、撤回・履歴、Authority |
| 不確実性 | Probability Distribution、Fuzzy Values、Intervals、Epistemic Claims | 確率質量・信頼・主張・未観測 |
| 構造化文書 | AST、Markdown、MathML/KaTeX、引用・改版 | ソース位置、正確な意味、表示、来歴 |
| メディア | Image/Audio/Video、3D、Multimodal Timeline | Sampling、Frame/Pixel、同期、メタデータ |
| 組織 | 複数親・代理・時点・権限の組織関係 | 非Tree性、委任/現権限、取消 |
| 行動・業務 | BPM/State Machine/Petri Net、タスク・イベント | 許可遷移、因果、副作用、手作業 |
| 認知構造 | KJ Canvas、Outline、Mind Map、Annotations | 近接・階層・代表化、仮説と受理の差 |

外部製品の名称や模倣をCore enumへ追加するのではなく、各行の差異を**構造、演算、利用文脈、保存義務、変換の損失**として抽出する。

## 7. 最小の反証・検証計画

| ID | 正例に必要な条件 | 典型的な負例 | 受入に必要な証拠 |
| --- | --- | --- | --- |
| MP-01 | Sequence→Bag→Setで順序・多重度の喪失を逐次説明 | Setから元の頻度を自動復元 | 独立した小fixture、説明の差分 |
| MP-02 | Join/GroupBy後の出力Grainと行寄与を区別 | Aggregateの一行を原Data一行として更新 | #1706/#2387の期待値/Lineage |
| MP-03 | Temporal Organization Graph→Treeで非表示辺を記録 | Treeの保存で別上長・代理辺を消去 | 時点別多親fixture、差分 |
| MP-04 | Quantity Point/Deltaで許容演算を分離 | 単位文字列一致でPoint+Pointを許可 | #1705型付き演算負例 |
| MP-05 | 時系列表示で未完全集合を明示 | 取得済みページを全履歴と誤表示 | #1706 completenessと観測証拠 |
| MP-06 | 対象が同じSource revisionで表・図・SHOへ出る | 一つのTargetだけ古いモデル・CRS/Unit | 固定revision＋比較Oracle |
| MP-07 | 編集可能なLens相当のFieldで無変更時GetPutが成立 | UIフォーマット変換の影響でSourceを変化 | 受理済みSource前後の比較 |
| MP-08 | 許可したView編集の再取得がPutGetを満たす | Query/集計表示から無根拠の自動書戻し | typed Action＋独立Read |
| MP-09 | Schema/Arrow等の物理表現差でも意味上の値を保持 | Dictionary/Null/Union物理差を業務意味差へ昇格 | 実Targetの符号化・復元を別試験 |
| MP-10 | Quantityの換算は単位/時点/誤差を保持 | currencyやCRS変換の外部条件を省略 | Contract/Source＋負例 |
| MP-11 | KJ自由配置と階層アウトラインを複合表示 | キャンバスの位置変更で階層を破壊 | 構造差分/注記保持 |
| MP-12 | AI文書・要約の未確定とSourceを保持 | 高い類似度や図の美観を意味受理と誤認 | Provenance/Disclosure確認 |

各行は未実行。既存#2495のMX判別票を重複実装せず、「構造の分類」後に**変換・表示・書戻し**の差で初めて現れる反例だけを担当する。初回はMP-01/02/05/06/07を机上で照合し、Readerから取得可能なEvidenceで実現できるか比較する。MP-03/11は組織・SUI側の自然なConsumerとして並列に進められる。

## 8. AI-first・Headless-firstで段階的に実証する

- **H0／広域机上探索**：#2495と#2493で多様な構造、複合要件、関係の意味を継続収集。矛盾例・未対応例も探索対象に残す。
- **H1／読み取り専用の比較**：#1120 ID、#1126 Relation、#1706 Completeness、#2387 Query IR、#1705数量、#1806 TypeRef等の既存の構造と証拠を使い、見えること・不足することをCLI/JSONで説明。新しい型Registryや第二Sourceを置かない。
- **H2／標準Plugin候補との接続**：Structure/Projectionの必要条件と提供Capabilityを照合し、表・Graph・Chart・文書・外部Targetの候補を列挙する。未対応・曖昧・競合を明示し、候補提示を配備権限と同一視しない。
- **H3／実行・変更の資格**：独立fixtureで変換・Round-trip・副作用拒否の差を検査し、必要なものだけDriver/Provider/Targetで観測。GUIはH1/H2の説明とH3の契約を利用するAdapterとして実装する。
- **H4／正式な意味と契約の採否**：契約の追加が必要であることを示す反例、既存の適合・影響差分、必要ならProof/Lean等で支持されたLawを、該当Ownerへ提案。数学のProof、合成PASS、実Host受入、Owner意味受理は別の資格。

**境界**：この文書の作成のみでH1〜H4の試験に進捗したとは認定しない。AIトークン単価低下によって多数の独立Pressure Caseを並行探索できても、権利、統合先のSource、実Providerの検証、Owner受理は既存の責任分界に従う。データ構造の分類と構造保存写像のどちらについても、後戻り可能な設計比較の後にCore/Plugin実装を判断する。

## 9. 採用前に確認する設計問題

1. 関係と保存義務を、既存Semantic Relation / Operation Conformance / Evidenceの組合せで記述できるか。読み取り専用Descriptorを追加すべきか。
2. Lens系の法則を**限定された編集可能View**に要求すると、既存のUI/Operationと何が比較可能になるか。書戻し前の競合と現在Authorityは別に保てるか。
3. 変換の合成による情報損失を、任意の新「情報量スコア」でなく、不変条件と用途の集合に対する具体的な保存可否で記述できるか。
4. 単位系・CRS・数式・Ontologyの標準Pluginが必要なとき、外部知識体系の版と適合試験をどこへ置くか。
5. Graph/Tree/KJ・時系列・集計のような表示・編集の異質なConsumerが、同じ最小Contractを再利用する際にどの追加概念を要求するか。

**最初に決めるべきことは、名前やCoreの新しい型階層ではなく、構造の変換・表示・操作のうち何が保存されるかを説明できる最小の関係と検証義務である。** 設計案と実装採用の区別は#2495/#2497のOwner判断を待つ。


## 10. 実Consumer：SUIの所属Relationと多値Fieldが共存するとき（2026-11-02追補）

これは#2497の「構造保存写像」と「同じ意味を複数Projectionで表す」具体的な圧力である。主に以下の公開コード・PRを固定した静的観測を出発点とし、**SUIでの正本・意味的一致はOwner未受理**とする。

- TEI main [`generator.go`](https://github.com/hat47x/tei-platform/blob/main/tools/reference-generator-selection/generator.go)、[`text-record.go.tmpl`](https://github.com/hat47x/tei-platform/blob/main/tools/reference-generator-selection/templates/text-record.go.tmpl)、[`record_projection_test.go`](https://github.com/hat47x/tei-platform/blob/main/tools/reference-generator-selection/record_projection_test.go)：Record Projection Fieldの`many`指定により、文字列配列をJSON配列として出力する候補生成。**`many`は表示/入力形状の条件であり、全件・一意性・順序の業務保証を与えるものではない。**
- TAI main [`sui.island.record` v4](https://github.com/hat47x/tai-pleroma/blob/main/integrations/pilot-sui-document/candidates/island-record/contract.json)：`Island.cardIds`が`many:true`の`std.identifier`。
- TAI main [`sui.affiliation.record` v1](https://github.com/hat47x/tai-pleroma/blob/main/integrations/pilot-sui-document/candidates/affiliation-record/contract.json)：`Affiliation(id,cardId,islandId)`はCardとIslandの多対多所属を表す。
- TAI main [`document_assembler.py`](https://github.com/hat47x/tai-pleroma/blob/main/integrations/pilot-sui-document/assembler/document_assembler.py)：`Island.cardIds`の各Card存在と、各Affiliationの両Endpoint存在を**それぞれ**検査する。**両表現の所属が一致するかの横断チェックは現在の関数には見当たらない。** TAI [PR #253](https://github.com/hat47x/tai-pleroma/pull/253)は空配列・欠落・nullの区別と既知Cardの参照を説明しているが、所属表現間の一致までを実証したとは主張していない。

### 10.1 同じ概念の異なるShapeか、別のMeaningかを最初に識別する

前提として、`Island.cardIds`と`Affiliation`が**同一の所属関係を表す**のか、異なる意味（例：直接表示するCardと実際の所属、任意の表示順、継承された所属）なのかは、元のSUI Contract／Ownerの意図による。

同一の所属関係を表し、同じDocument/Scope/Snapshotで双方とも**全件取得済み**で、各IslandのCard重複が無効と宣言されている場合に限り、次の集合上の関係が要求候補になる。

\[
\operatorname{set}(\mathrm{Island.cardIds}(i)) 
= \{a.cardId \mid a\in\mathrm{Affiliation},\ a.islandId=i\}
\]

この式は**TEI現行DSLに追加された法則ではなく、同一意味だという仮説を検査する独立Oracleの例**である。

- CardIDを集合へ変換すると、`cardIds`の重複数と順序は失われる。順序がMeaningなら集合一致だけでは不足する。Affiliationの個別ID・来歴も集合変換からは復元できない。
- Arrayの`[]`は取得済みの空集合かもしれないが、`missing`（Key欠落）、`null`（unresolved候補）、関係先の未取得、別Scopeの遮蔽とは区別する。TEIの`many`テンプレートによる`present/missing/unresolved`は、これらを完全なDomain状態として列挙したものではない。
- 一方のProjectionが部分取得なら、全件一致を要求する前にフィルタ・権限・時間・行選択と、部分集合側の包含条件を定める。単に「片方に現れない」を不存在としない。
- 二つのShapeを保存していることだけで、`Affiliation`と`cardIds`のどちらが正本か決まるわけではない。正本が一つで他方がDerivedなら、どこから再導出するかを宣言する。両者が独立の業務意味なら、対応する関係の種類を分ける。

### 10.2 判別対：参照整合が通ってもProjection整合が未確認となる

合成データ（**テスト未実行**）：

| ケース | CardとIsland | `Island.cardIds` | `Affiliation` | 判別点 |
| --- | --- | --- | --- | --- |
| SU-01 一致候補 | Cards `c1,c2`, Island `i1` | `["c1"]` | `a1(c1,i1)` | 同一Meaning・同時点・全件なら集合一致 |
| SU-02 矛盾候補 | 同上 | `["c1"]` | `a1(c2,i1)` | 両端点が実在しても、所属一致という追加契約は満たさない |
| SU-03 多重性 | 同上 | `["c1","c1"]` | `a1(c1,i1)` | Set変換後に一致しても配列の重複意味を判定すべき |
| SU-04 空と未取得 | 同上 | `[]`／`missing`／`null` | `[]` | 三状態を同じ「所属0件」とは断定できない |
| SU-05 順序 | 同上 | `["c2","c1"]` | `a1(c1,i1),a2(c2,i1)` | 表示順の意味契約がないならRelationから順序を復元できない |
| SU-06 時点不一致 | 同上 | R2から取得 | R1から取得 | 数値・識別子が一致しても両版の真理を連結できない |
| SU-07 部分集合 | 同上 | 権限・ページ限定の`["c1"]` | 全件`c1,c2` | 完全性が違えば等号より包含の条件を検討 |
| SU-08 編集Intent | 同上 | UI上で`c2`を追加 | 関係レコード未変更 | Arrayの更新が所属を作るのか、表示専用編集なのかをTyped Actionで定義 |

重要な差分は、現在のAssemblerが**個別の参照先実在性**を検査することと、これらの**二つの所属表現の相互一致**を要求することが別の検査である点。SU-02のような入力を実際に実行した試験結果はここでは取得していない。正しく表すべきMeaningが一致する場合の**新しい検証候補**に留める。

### 10.3 拡張可能なUI・標準Pluginへ還元する場合

AI向けCUI、SUIキャンバス、SHO文書、標準Table/Graph投影が同じ関係を表示する場合、次を最低限説明できるかを既存Consumerへ返す。

1. **Subject/Grain**：Card・Island・Affiliation・投影されたCardID要素のどれに対応するか。Affiliation IDとCard IDを混同しない。
2. **Relationship/Derivation**：多値Fieldは一次のデータ定義か、RelationからのDerived Viewか、別の意味か。対応元・Scopeと版を残す。
3. **Order/Multiplicity/Completeness**：一覧の順序・重複・ページ/窓/権限による可視範囲。形状`many`だけから推論しない。
4. **Editing**：操作が`Island.cardIds`の表示用配列を編集するのか、AffiliationのLink/Unlinkを発行するのか。現在のAuthority・CAS/Revision・結果・独立再取得は実行担当に委譲する。
5. **Projection一致**：Sourceから二つの表示を派生するなら、同じSnapshot/意味バージョンと対応する不変条件・不一致理由を示す。

UIを実装する各Pluginが高次概念を手で宣言するのではなく、標準のRelation／Array形状／Source Reference／Operation契約から**必要な情報を組み立てる**構成を評価する。新しい「Island型」「Collection所属型」のCore列挙を先に追加するのではなく、既存のReferenceが保持する情報と、不足している推論/適合項目を個別に計測する。

**担当境界**：#2497はこのような変換と投影の保存義務、#2495/#2544は構造の分類と演算・Law束縛、#2501はPressure Corpus、#2506/#2507はFunction Data ContextとPattern、#2500はView変更→Typed Intent、TAI SUIはDomain上のMeaningを扱う。二つのGitHub Repoのモデル正本や実装をこの文書から変更しない。


## 11. 実Projectionで見えた値忠実度と部分被覆（2026-11-02追補）

**対象・ステータス**：TEI mainで既に追加されたReference Generator Selectionの`record`投影を、基本構造の構成／Presence／数値／多値／Source責務の異なるFacetが交差する圧力として読む。実装上の所有・局所Conformanceは[#2585](https://github.com/hat47x/tei-platform/issues/2585)、一般的な変換・Projectionの意味保存は本#2497、外部形式・Targetの実Consumer実証は#2498、数量の計算上の意味は#1705に残す。

### 11.1 最新mainの具体的な実装と確認できる範囲

- [TEI PR #2576](https://github.com/hat47x/tei-platform/pull/2576)、[#2578](https://github.com/hat47x/tei-platform/pull/2578)、[#2580](https://github.com/hat47x/tei-platform/pull/2580)、[#2581](https://github.com/hat47x/tei-platform/pull/2581)により、多値文字列・数値・optional・booleanのReference Field投影が追加された。
- [`selection.go`](https://github.com/hat47x/tei-platform/blob/main/tools/reference-generator-selection/selection.go)のFieldは`Primitive/Many/Optional`を別属性として持つ。現行`validateContract`はfield名・入力名・Attribute・Primitiveの存在と重複Field名を検査するが、`many×primitive`のサポート行列と各Projection.Field.InputがContract.Inputsで宣言された入力に一致するかの検査は、当該関数内に見当たらない。`validateAgainstWorkspace`はContract.InputsのPrimitiveをTypeRefに照合するが、Field.Inputの参照に対する同一検査は別途必要。
- [`text-record.go.tmpl`](https://github.com/hat47x/tei-platform/blob/main/tools/reference-generator-selection/templates/text-record.go.tmpl)は、`many`を各要素`string`の配列として処理する。多値の数値やbooleanに相当するContractを宣言できても、その形に対応した生成コードを出すとは確認できない。
- 同テンプレートで`std.integer/std.decimal`の単独値は、`map[string]any`を経由して`float64`へcastされる。Go公式[`encoding/json`](https://pkg.go.dev/encoding/json)でも、一般の`interface{}`へ入力されたJSON数値は既定で`float64`。したがって、JSON数値形状の保持は**正確な整数・十進数の値・桁/scaleが保持されたこと**を意味しない。
- `optional`の欠落と明示`null`はともにフィールド出力を省略する。必須の場合は欠落を`missing`、明示nullを`unresolved`として扱う。つまり**同じ入力値状態でも、利用箇所によって区別が表示される場合と失われる場合がある**。optional booleanの`false`は未設定ではなく、値として保持される。
- 新規[`record_projection_test.go`](https://github.com/hat47x/tei-platform/blob/main/tools/reference-generator-selection/record_projection_test.go)には、それぞれの生成sourceに指定のcast/分岐が存在するかの確認がある。**異質フィールドの全組合せ、raw numericの厳密性、入力バイトから生成された実行体までの全値比較を網羅した証拠とは異なる。**

### 11.2 多重Facetの組合せを、名前の分類から利用条件へ移す

データ項目を意味上説明するとき、次の関心事を別々に評価する。これは新しい公開Schemaを採用するものではなく、既存TypeRef/Relation/Operator/Projectionを読んで比較するための表現である。

| 項目 | 独立した質問 | 同じ`Primitive`だけでは答えられないこと |
| --- | --- | --- |
| Value/Quantity | どの値・Quantity Kindか | decimalの正確な尺度・範囲・単位、codeの発行体系 |
| Presence | 入力・出力で未指定・null・値・未取得をどう表すか | optionalで情報を省略してよいか、Patchでclear可能か |
| Cardinality | 単一か、任意個か、最低件数/上限があるか | `many:true`が何件の業務Relationを保証するか |
| Collection semantics | sequence/set/bagのどれか | 順序・多重度・Key一意性・全件性 |
| Element encoding | 要素ごとに何の物理型を使うか | `many:true`が数値/boolean配列へ対応するか |
| Realization | どのTarget/Generator版で変換するか | 字句・精度・形式の損失、実行体とSourceの対応 |
| Provenance/Authority | 誰のどの版・どのScopeの主張か | Derived Viewの未表示属性、独立のEffectやMeaningの受理 |

これらは**相互に必要な制約を持つため、完全に独立したチェックボックスの直積とは限らない**。例えば`optional`は出力の欠落を許す表示上の性質だが、入力契約のNullable、Queryの未取得、編集Intentの`clear`を自動定義するものではない。`many`は配列形状を指定するが、要素の型とCollection Lawの適用は追加の検査を要する。

### 11.3 用途相対のLoss Budgetと書戻し条件

Projection `p : S→V`を観測するConsumer `C`について、必要な観測集合`O(C)`（例：構造/Presence/整数同一性/Quantity/時点/全件性）を定める。各観測について`p`が保持・消失・未判定のどれであるかを説明し、Consumerが**要求する観測がすべて保存される場合にだけ**当該用途のConformanceを主張する。

- **表示・参考グラフ**：任意の属性の欠落や座標の表示誤差をConsumerが許す場合、結果を限定用途で利用できる。画像座標の誤差許容と金額の十進正確性は別。
- **厳密な金額・業務IDの保存**：`float64`を経由するReferenceの`std.integer/std.decimal`を、正確性を要求するContractへ自動適合させない。例：`9007199254740993`は二進64bit浮動小数で整数として完全には表現できない。値・字句・scaleの保存を何で要求するかは業務とTargetの契約に従う。
- **差分・Patch・編集**：欠落とnullを区別する更新要求や、`false`／`0`への明示的変更を扱うには、`optional`による省略だけでは十分でないことがある。元のSourceを参照したTyped IntentやFieldMask相当の明示的な更新範囲が必要かを#2500で判断する（[Protobuf Field Presence](https://protobuf.dev/programming-guides/field_presence/)の比較を参照）。
- **複合文書の読み取り**：対象Fieldの一部だけをTEIで投影し、他の属性を元のSUI Dataから補う場合、結果のDocumentが全てTEI由来だと推論しない。出所・版・FieldごとのOwnerと、各Consumerが最終的に観測した範囲を区別する。

ここでいうLoss Budgetは**用途に応じて許容する損失の契約**を指し、数値的な普遍スコアや二重の権限エンジンではない。

### 11.4 SUI/TAI実Consumerで「全体の一部だけ」を明示する

TAI mainの[実装README](https://github.com/hat47x/tai-pleroma/blob/main/integrations/pilot-sui-document/README.md)および[移行計画](https://github.com/hat47x/tai-pleroma/blob/main/docs/tei-construction-migration-plan.md)には、Card座標`x/y`（[TAI PR #258](https://github.com/hat47x/tai-pleroma/pull/258)）、optionalな`claimType/holdState`（[#259](https://github.com/hat47x/tai-pleroma/pull/259)）、optional `textReviewed` boolean（[#260](https://github.com/hat47x/tai-pleroma/pull/260)）、artifact形状の検証（[#261](https://github.com/hat47x/tai-pleroma/pull/261)）がある。`critiqueHistory`のような構造体配列、`transform`、`createdAt/updatedAt`などは今回の投影対象外とし、**SUI側の元データを正本とする暫定設計**である。DocumentV1の書込みよりread-only artifactを先行する（[#254](https://github.com/hat47x/tai-pleroma/pull/254)）。

ここには少なくとも三つの正当な変換形式がある。

1. **完全な同一意味の転写**：対象の全FieldとRequired Lawが保持される（ConsumerとContractの範囲を限定）。
2. **用途相対の部分Projection**：表示や検討に必要なFieldだけを保持し、省略項目のSource／復帰経路・欠落理由を示す。
3. **複数Sourceからの合成**：異なるSubject/Version/Authority由来の値を、元Sourceを残して同じ表示へ合わせる。

どれを採用するかはConsumerの目的に依存する。**表示可能という事実から、全属性をTEIから復元できること、編集Intentを安全に再適用できること、単一のDelivery正本が成立することは導かれない。** Compositionや部分Projectionの意味はTEI/TAI双方のOwnerへ帰属させる。

### 11.5 受入候補（#2585を実行先とする）

`PFC-01〜12`はIssue #2585に定義。特に、`optional × required`によるPresence差、`many × numeric/bool`の対応範囲、`float64`の数値忠実度、Field→Input→TypeRefの検査、read-only artifactとSUIの非投影Fieldを**同じ固定Versionで別々に**比較する。最初は現行の小さな生成Templateを実行し、Source文字列存在テストを越えて実際のraw JSON出力と終了コードを観測する。

**今回の資格**：source codeとTAIの静的資料を照合した設計仮説。上記PFCを実行した結果やOwnerによる正式な意味受理ではない。標準Core TypeKind・新共通Runtime/Host・独立Source正本の追加は提案していない。


## 12. 時間・増分観測・結論の撤回を構造保存に接続する（2026-11-02追補）

### 12.1 他レーンとの接合：静的Projectionから「変わる観測」へ

[#2586](https://github.com/hat47x/tei-platform/issues/2586)（Query→Data Context→Claim→Pattern→操作候補の単一Function縦断）、[#2506](https://github.com/hat47x/tei-platform/issues/2506)（取得範囲・Context Instance/epoch）、[#2507](https://github.com/hat47x/tei-platform/issues/2507)（異質Consumerの機能Pattern）、[#2505](https://github.com/hat47x/tei-platform/issues/2505)（版間Source意味対応）で、**同じ構造の異なる時点・異なる取得範囲**をどう説明するかが具体化した。SQLの[#2387](https://github.com/hat47x/tei-platform/issues/2387)は演算・Lineage、[#1708](https://github.com/hat47x/tei-platform/issues/1708)はFreshnessを所有する。本#2497はそれらの意味を上書きせず、**ProjectionがClaimの成立条件と撤回条件を保存するか**を扱う。

静的な写像`p:S→V`だけでなく、実際の読み取りを`observe(S, query, scope, snapshot?, epoch, time)`と考え、その結果に適用した`project`がどの観測可能なClaimを支持するかを比較する。これは既存Referenceを読むための記法であり、新しいTEI共通API・Runtime・正本の提案ではない。

**分離する時点と版**：

- Source definition revision／semantic acceptance revision：意味定義の版。
- Query/Provider result revision、read-set／snapshot／scope：どの範囲のデータを観測したか。
- Event time／processing time／observation time：イベント発生、処理、観測の時刻。watermarkは実Providerが保証する範囲に限る。
- Selection epoch／UI rendering revision：利用者の選択や画面に適用する応答の世代。
- Artifact/build revision／consumer load revision：生成・配布・実際の読込みの版。

同一タイムスタンプ文字列だけで同一Snapshot・同一Meaningを宣言せず、異なる版を横断して同一とするには対応する根拠が必要。

### 12.2 「情報が増えれば結論が増える」の成立条件

**単調な問い合わせ**とは、同じ解釈・Universe/Schema・Identity/Scopeを固定し、データを追加するだけで既存結果を失わない問い合わせである。存在確認（肯定的な存在証人）は、追加だけのデータについては保持しやすい。しかし、訂正・削除・権限変更・時点変更を許す実世界では、以前の存在証人も再検証対象となる。

- `SELECT`で見えた一件の存在は、**その観測時点とScope**での肯定的証人。将来の存在を保証するものではない。
- `COUNT`、`SUM`、`AVG`、TOP-N、NOT EXISTS、LEFT JOINの未一致に由来する仮NULLなどの**結果値・結論**は、追加データで変わり得る。特にSUMの下界は、既知入力が非負、同じ単位・Identity・時点・JOIN重複無し、追加のみなどの条件付き。
- 0件の取得は、対象母集団・条件・取得完了が確認できて初めて**その範囲内での不存在**を支持する。未取得・先頭ページ・権限で制限されたSliceから世界全体の不存在を結論しない。
- 集計の1行はEntityの1件と同一の粒度ではない。JOINで2×3=6行となったとき、元の二つの関係の実体件数を6件と報告しない。

ここで区別するのは**入力データの追加に関する単調性**と、**判断・Claimの維持可能性**である。集計の中間状態や近似値を表示することは可能だが、確定値として扱うには対象Consumerの追加条件が必要。

### 12.3 Claimの変更を表す最小状態（概念上の作業票）

| 状態 | 意味 | Viewへ伝えること |
| --- | --- | --- |
| supported-in-scope | 指定Scope/版の観測でClaimを支持 | 対象・根拠・観測時点・条件 |
| provisional | 部分集合・遅延・未確定条件付きの候補 | 未観測部分・更新可能性 |
| contradicted-in-scope | 同じ条件・比較可能な版で反証を観測 | 矛盾した対象と観測根拠 |
| superseded | 後続の同一対象・比較可能な版の結果へ置換 | 旧版と新版、置換理由 |
| unknown-or-stale | 成立判断に必要な観測が欠落・失効 | 不足・再取得・再評価の条件 |

これらは**TEIの新しい正式enumではない**。既存Result/Diagnostic/Claimの表現可能性を比較するための語彙である。例えば、後から到着したLEFT JOINの一致行により旧仮NULL行が消える場合、表示側は単なる追加だけでなく旧行の撤回を扱う必要がある。古いselection epochの非同期応答は、現在選択のClaimへ混ぜない。

### 12.4 同じSourceから二つのConsumerへ投影するときの検査

**一つのFixtureを固定して、二つの異質Consumer**（Headless CUIと表/Chart、またはSHO文書）へ投影する。

1. D1が完全2件、D2が先頭ページ1件（母集団3件）という観測を、二つの独立Relation Pathとして保持する。D1とD2のJOINの表示件数は元のEntity件数へ昇格しない。
2. D2の残りを取得した後、表示中のAggregate、存在／不存在Claim、Pattern適合、操作候補について、**何が再評価され、何が旧版のまま保留か**を比較する。全再取得を基準とし、局所更新の採用は依存条件が観測できる場合に限定する。
3. A選択→B選択の間にAの遅延応答が返っても、BのContextへ帰属させない。Selection epochは業務Operation IDと別。
4. `null`、Query由来の仮NULL、optional Fieldの省略、未取得、Claimの`unknown`を区別する。
5. 一方のConsumerで旧Claimが撤回されたら、他方に旧Claimを確定として残さないための**通知・再解決の責務と観測境界**を示す。実Consumerが非同期更新を受け取った証拠なしに全画面同期を主張しない。
6. Sourceモデル改版（[#2505](https://github.com/hat47x/tei-platform/issues/2505)）がある場合、byte/locatorの変化と意味上の同一性候補、Owner受理を区別する。

**担当の受入経路**：単一FunctionのHeadless縦断実装は#2586、Contextの取得範囲・再評価は#2506/#1708、Pattern/表示は#2507/#997、版間意味対応は#2505、Target/Consumer実証は#2498、本#2497は保存・喪失の説明条件を保持する。新しい共通Changefeed/Claim Engine、第二Authorityや専用Hostは前提にしない。

### 12.5 新しい反証票（TP-01〜12、全件NOT RUN）

| ID | 最小の反例 | 失ってはいけない区別 |
| --- | --- | --- |
| TP-01 | 先頭1ページ0件 | 部分0件と母集団0件 |
| TP-02 | 0件確定後に新規挿入 | 過去の不存在と現在の不存在 |
| TP-03 | 既知非負SUMに負値が追加 | 条件付き下界と一般SUM |
| TP-04 | JOINの重複でSUMが2倍 | Relation Grainと集計対象 |
| TP-05 | LEFT JOIN仮NULL→一致行到着 | NULL値と未一致、撤回 |
| TP-06 | TOP-Nの新しい上位要素到着 | 追加のみの表示と旧行置換 |
| TP-07 | 旧Selection epochの応答到着 | Context InstanceとBusiness ID |
| TP-08 | イベント時刻と処理時刻の逆転 | Event/Processing/Observation Time |
| TP-09 | 同一時刻・異なるSnapshot | タイムスタンプ一致と一貫したRead-set |
| TP-10 | Model revision変化、Source locator同じ | locator一致と意味上の同一性 |
| TP-11 | 一方のConsumerだけ再描画 | 更新通知と実Consumer観測 |
| TP-12 | 数値型は保持、取得Scopeは縮小 | Value fidelityとClaim completeness |

**状態**：他レーンの設計とIssueを照合した反証可能な追加提案。TP-01〜12、実Provider、GUI、Owner意味受理は未実施。外部仕様の引用を正式採用したものではない。


## 13. Measureの再集計とClaimの開示可能性を別の保存条件として扱う（2026-11-02追補）

### 13.1 構造・値が一致してもMeasureの意味は一致しない

他レーンの[#2593](https://github.com/hat47x/tei-platform/issues/2593)は、同じDecimalでもFlow・Stock・Ratio・Distinct Count・Medianで合法な集計軸と再集計状態が異なることを整理した。これは#2544の**演算に束縛された法則**、#2495のFacet、#1705のQuantity、#2506のFunction Data Context、#2497のProjection保存条件をつなぐ具体的な検査対象である。

**再集計は単なる型変換ではない。** 例えばA群の平均10（2件）、B群の平均20（8件）を同じDecimal型で保持していても、単純平均15は全体平均ではない。全体平均18を得るには、少なくとも群ごとのsum/count（または元値）と、互換なScope/Unit/Filter/時点/重複ポリシーが必要。比率50%=1/2、80%=8/10の全体比率も65%ではなく9/12=75%。日末残高は同時点の口座間加算が可能でも、異なる日の残高を足して「現在残高」とすることはできない。

このため、`preserves(value)`だけでなく、**用途相対に`preserves(aggregation prerequisites)`を評価する**。候補の検査軸は次のとおり。

| 軸 | 保持・確認する情報 | 保存されないとき |
| --- | --- | --- |
| Base/Result grain | 元Fact ID、出現・JOIN重複、集計後Group Key | 同じ値の別Factを`DISTINCT value`で消す危険 |
| Aggregation axes | 対象次元、時間軸、演算の合法範囲 | Flow/Stock、期間/時点の誤集計 |
| Recomposition witness | sum/count、numerator/denominator、distinct IDs、sketch/error | 平均/比率/Distinct/分位数の再集計不能 |
| Scope/Completeness | 対象集合、部分取得、Filter、Snapshot、訂正・遅着 | 暫定値を確定値と表示 |
| Value semantics | Unit、Quantity Kind、Scale、丸め、数値精度 | 異単位・異通貨・精度差を暗黙合算 |
| Consumer purpose | Chart表示、業務計算、文書の確定主張、更新Intent | 参考表示の適合を業務Actionへ昇格 |

**再集計不能は必ずしも元データが無効という意味ではない。** 特定のConsumerに必要な中間情報が不足している場合、対象用途で`unsupported/unknown`を返し、別のSourceや専門Providerで計算できる可能性を保持する。一般的な数学的証明・自動SQL最適化を先行条件としない。

### 13.2 「正しく導出した」「根拠を利用できる」「開示できる」は別の主張

[#2516](https://github.com/hat47x/tei-platform/issues/2516)のConstruction Context／AI CUI／SHOと、SQLの[#2387](https://github.com/hat47x/tei-platform/issues/2387)のSource Occurrence/Lineageの検討を接続する。ここで重要なのは、**内部的に正しいClaimがあっても、すべてのConsumerに同じ根拠を開示できるとは限らない**こと。

- **Derivation validity**：どの意味定義・演算・Source版・Scopeから結論が成立するか。静的なSource一致だけで実Queryや業務受理を証明しない。
- **Evidence access**：当該処理が根拠を参照する資格・利用条件を持つか。データへのアクセスと実際のRead観測を分離。
- **Disclosure suitability**：外部LLM、SHO文書、公開レポート等のConsumerへ、どの情報・粒度・根拠を渡せるか。内部閲覧資格から外部開示を推論しない。
- **Consumer sufficiency**：開示可能な情報だけで、そのConsumerがClaimを理解・再検証できるか。要約・匿名化・一部非開示によって根拠が不足する場合は、適用範囲と未確認を残す。
- **Current applicability**：以前の受理・証拠・開示資格が、現行のRule/時点/対象/Grantでも有効か。過去の成功を自動復活させない。

これは新しい中央Authority/証拠Registryを提案するものではない。各既存Ownerの認可・開示・Source・Claimを**read-onlyの用途相対比較**として接合する。秘匿Sourceの内容をIssue/CIログへ複製する必要もない。

### 13.3 Consumer別の同じClaimと情報境界

同じClaimを次の三つの出力へ投影する場合、**同じ結論文が表示できること**と**同じ根拠を公開できること**を別々に評価する。

1. **内部CUI/Headless JSON**：必要なSource参照、条件、失われた情報、未実行の観測を説明する。外部へ出せないRawを、外部公開用と同じ応答へ混ぜない。
2. **SHO設計文書**：対象意味・原典locator・引用可能な範囲・固定納品版と再生成条件を保持。非公開Sourceを公開文書へ展開する場合、要約の適法性・根拠の再解決を独立確認。
3. **外部Plugin/LLM Consumer**：Disclosure Conditionに合致する必要十分なSliceを構成し、実際に送信・受信・利用されたかの証拠を区別。Toolや資料中の命令をOwner判断・実行権限へ昇格させない。

同じSourceから二ConsumerへProjectionした場合、`same value`、`same Claim`、`same evidence visible`、`same authority`は別々の述語である。見せられない根拠を「存在しない」「検証済み」と補完しない。

### 13.4 反証票（AM-01〜12、全件NOT RUN）

| ID | 最小の比較 | 誤成功 |
| --- | --- | --- |
| AM-01 | 2件平均10と8件平均20 | 単純平均15を全体平均とする |
| AM-02 | 比率1/2と8/10 | 50%と80%の単純平均65% |
| AM-03 | 同じ100の二つの異なるFact | `SUM(DISTINCT value)`で一件を消す |
| AM-04 | 二部門に同じ人が所属 | 部門別Distinct人数を加算 |
| AM-05 | 二日分の同一口座残高 | 時間方向にSUMして現在高とする |
| AM-06 | 局所中央値のみ | 全体中央値を確定復元したとする |
| AM-07 | 異通貨・異時点のFlow | 換算時点・Rate/丸めなしに合算 |
| AM-08 | partial Windowへ遅着・訂正 | 暫定値を最終値として残す |
| AM-09 | 内部では根拠を読めるが外部送信禁止 | 内部アクセスを外部開示許可へ昇格 |
| AM-10 | 開示許可の要約が必要根拠を欠く | 同じ結論文だから再検証可能と宣言 |
| AM-11 | Source版同じ、Rule/Grantが変化 | 旧意味受理・旧開示資格を自動継承 |
| AM-12 | CUIへProjection成功、SHO/外部Consumer未観測 | 一Consumer成功を全Consumerの実読込PASSとする |

**Owner/実証境界**：集計意味・独立Fixtureは#2593（MC-01〜10）、型と公理は#2544、Query/Context/Claimの縦断は#2586、情報境界と外部開示は#2516/#1178、SHO原本再解決は#1803/MMI、外部Consumer実証は#2498。ここでは各Issueの正本を参照し、第二のMeasure Engineや第二の承認・開示エンジンを構築しない。

**状態**：横断設計と反例の追加。AM-01〜12の実行、外部送信観測、SHO保存実証、Ownerによる正式受理はNOT RUN。


## 14. Consumer間のClaim比較とDocument Occurrenceの独立性（2026-11-02追補）

### 14.1 同じ表示値は同じClaimを意味しない

[#2516](https://github.com/hat47x/tei-platform/issues/2516)のW8（Cross-Consumer Claim Reconciliation）では、同値・異値だけでConsumer間の整合を判定できないことが明確になった。これを#2497の構造保存条件として取り込む。

- **同値でも比較不能**：CUIとSHOがともに`200`を表示していても、片方が売上合計、他方が在庫数量、または異なる対象集合・時点・通貨なら同一Claimではない。
- **異値でも両方正当**：同じMeasureでも、CUIが部分取得の100、SHOが確定済み全件200を示す場合、それぞれのScopeと暫定／確定資格に相対的には正しい可能性がある。
- **比較可能で異値**：同じSemantic Subject/Operator、同じ対象集合・Grain、Snapshot、必要なSource/Rule/Projection版と完全性が一致し、CUIが200、SHOが201なら、初めて真正の不一致候補として独立Oracleへ戻す。
- **同値でも異なる元状態**：optionalのmissing/nullがともに省略、または`float64`で別の整数が同じ数値に写る場合、値の一致からSource状態を復元できない。read-onlyの表示適合と監査／編集適合を分ける。

比較可能性は、少なくとも`subject + role + operator + grain + scope/completeness + effective time/snapshot + source/meaning revision + consumer purpose`を照合した**Claim相対の関係**として扱う。これは比較用の記法であり、新しい全域ComparatorやClaim Registryではない。必要条件を満たす場合でも、独立Oracleや実Consumerの観測が欠ければ値の正しさは未確定。

### 14.2 観測同値・意味同値・出典同値・編集同値を分ける

投影`p`について、`p(s1)=p(s2)`から`s1=s2`は一般に導けない。逆向きの推論が成立するには、Consumerが要求する区別に関して`p`が単射である、または失われた情報を独立のSource/証拠から補える等の条件が必要。

| 同値性 | 比較するもの | 値一致だけで保証できないこと |
| --- | --- | --- |
| 観測同値 | 同じConsumerの見える結果 | 元状態の同一性、完全性 |
| Claim同値 | 同じ対象・条件・演算・時点の主張 | 証拠の現在性、業務Owner受理 |
| Source同値 | 同じ原本/locator/版/由来 | 同bytes別所在、版間意味対応 |
| Document Occurrence同値 | 文書中の一出現と他の出現 | 同一TEI要素でも同一の局所編集・レビューではない |
| 編集同値 | View変更が同じSource更新を意味するか | 書戻し先、差分、許可、実Effect |
| Disclosure同値 | 同じ情報をConsumerへ出せるか | 内部参照可と外部送信可の同一視 |

### 14.3 TEI→SHOの一対多Projectionと局所編集

SHO側のDocument Occurrence検討（SHO [#4](https://github.com/hat47x/sho-composer/issues/4)、[#6](https://github.com/hat47x/sho-composer/issues/6)）では、同じTEI要素を文書の複数箇所へ出現させ、**出現ごとの説明・配置・局所編集・レビュー状態**を区別する。元のSemantic IDと文書内Occurrence IDを分離することが重要。

比較例：TEIの同じ業務項目`X`を文書Aの第1章と第4章へ投影し、第4章だけ著者が説明を上書きした。その後、TEI側で`X`の意味定義が改訂されたとき、

1. TEI意味定義の版と、二つのDocument Occurrenceの投影元を追跡する。
2. 局所上書きは文書内の独立編集として保持し、元モデルへの変更提案と区別する。
3. 第1章が再生成できても、第4章の説明が現行意味と整合するとは自動判定しない。
4. 文書ファイルの保存成功・Source再解決・読者向け説明の成立・レビュー現在性・納品版固定を別々に扱う。
5. 文書の局所編集をTEIの受理済み意味へ自動逆反映しない。明示的な変更IntentとOwner判断が必要。

SHOが文書の正本、TEIが受理済み意味の正本、MMIがSource/Candidate/Provenanceの再利用可能な処理を担う既存責務を維持する。SHO側の実装・保存再開、MMI C-01/T2の実機受入はこの設計から完了と推定しない。

### 14.4 開示条件を含む比較の非干渉性

[#2516](https://github.com/hat47x/tei-platform/issues/2516)で追加されたDP-M4a/b、DP-M5a/bの問題は、**非公開の情報を見せないだけでなく、比較結果や拒否理由を通じて非公開情報の有無・件数を推測させない**ことにある。

同じ低権限Consumerに見える公開データを持つ二つの世界で、秘密のデータだけが異なる場合、許可された明示的な集計等を除き、Consumerへの出力・比較診断・失敗理由・Traceが秘密差を明らかにしないか検査する。これは設計上の非干渉性の比較候補であり、正式なPolicyやOS隔離が成立したという主張ではない。外部AIへの実送信、反復照会、ログの公開範囲は#2516/#1178等のOwnerが検証する。

### 14.5 反証票（CR-01〜12、全件NOT RUN）

| ID | 最小比較 | 判定したい差 |
| --- | --- | --- |
| CR-01 | CUI売上200、SHO在庫200 | 値一致とClaim不一致 |
| CR-02 | CUI部分100、SHO全件200 | 値不一致と双方のScope相対の正当性 |
| CR-03 | 同一Contract/SnapshotでCUI200、SHO201 | 比較可能な真の不一致候補 |
| CR-04 | 旧版SHO200、新版CUI220 | 文書保存版と現在意味版 |
| CR-05 | optional missing/nullをともに省略 | 表示同値と原状態同値の差 |
| CR-06 | 異なる大整数がfloat64で同値 | 値投影の非単射性と厳密ID不適合 |
| CR-07 | 同じSQL文字列、RLS/Scopeが異なる | 字句一致と実効集合の差 |
| CR-08 | 同一TEI要素→文書内二Occurrence | 意味対象同一と編集対象の相違 |
| CR-09 | 一Occurrenceだけ説明を局所上書き | 再生成・レビュー・意味変更提案の分離 |
| CR-10 | TEI改版、文書表示は同じ | 意味変更と表示byte不変 |
| CR-11 | 公開情報同一、秘密情報だけ異なる二世界 | 比較結果・拒否理由・Traceからの漏出 |
| CR-12 | CUI結果PASS、SHOは保存のみで実表示未確認 | 一Consumer観測と全Consumer受入の差 |

**受入経路**：Consumer間の実比較は#2516 W8、実Record値忠実度は#2585、Functionの同一Fixtureは#2586、SHO文書のOccurrence/編集・レビューはSHO #4/#6、MMIの原本再解決はMMI C-01/T2、開示Policyと隔離は#1178/#2515。#2497は構造保存の一般条件とLossの説明を所有し、第二の文書正本・Comparator・権限機構は設けない。

**状態**：設計文書と反例の追加。CR-01〜12の比較実行、実SHO/OS/AI/Provider/GUI試験、正式な意味受理はNOT RUN。

## 15. 2026-11-02 逆写像の不可能性・集計可換性・情報非干渉を独立した保存義務へ

この節は#2497が所有する**構造写像の成立条件**を、他レーンで実際に遭遇した値・関係・編集・開示の具体例で反証可能にする。新しい共通TypeKind、Lens Engine、Metric Engine、Authorization/Hostを実装する仕様ではない。

### 15.1 情報を潰したProjectionに、自動的な完全逆写像は存在しない

Source集合をS、read-only ProjectionをP:S→Vとする。もし異なるSource状態s1≠s2についてP(s1)=P(s2)なら、表示Vだけを入力する決定的な関数R:V→Sが**全てのsについて**R(P(s))=sを満たすことは不可能。Rは同じ入力P(s1)=P(s2)へ異なる値s1とs2を同時に返せないためである。

これは部分取得・丸め・集計・一部Fieldだけの表示で共通に成立する、限定された数学的事実。具体例は以下。

| Sourceの相違 | 同じProjectionが生じる理由 | 誤った編集／逆写像 | 既存Owner |
| --- | --- | --- | --- |
| `9007199254740992`と`9007199254740993` | いずれもJavaScript `float64`で同じ近似数へ落ち得る | 表示数から正確なInteger IDを復元したと断定 | #2585/#1705 |
| Personの所属Event AとB、同じ「部門人数2」 | 集計はOccurrence/人のIdentityを捨てる | 人数を1増やす編集を特定PersonのLinkへ黙って逆写像 | #2386/#2596/#2500 |
| 空の配列と未取得のRelation、表示上ともに空 | 完全性・Presenceの根拠を落としている | `[]`を受け取り全所属をUnlinkしてよいと断定 | #1706/#2497/#2597 |
| 旧版と新版で同じSHO本文 | 同じ表示ByteがSemantic revisionを表現しない | 文書保存成功から元TEI意味を既に改版済みと宣言 | #2505/SHO #6 |
| 非公開のEvidenceを含む内部状態が異なる | 公開Consumerへ許可された要約が同じ | 表示上の一致を現在の秘密状態一致へ拡張 | #2516/#1178 |

**誤解しない点**：Pが非単射であっても、すべての編集が不可能なわけではない。既存のSource状態sとbase revision、Consumerが出した**型付きAction/Intent**、補足された対象Identity/Parent Role、現在のAuthority、Conflict/Commit/Receiptを用いる`putback(s, intent)`なら、対象操作を限定した書戻しは成立し得る。これを無条件の`inverse(P(view))`やSource/Effectの自動受理へ昇格させない。

### 15.2 二経路が同じ観測を生む条件（commuting comparison）

同じ受理済みSource revision Rと適用Scope Kから、経路A（Query→Context→Measure）と経路B（別Projection→Consumer）へ派生するとき、数値・表示だけを比較して「両経路は可換」とは宣言しない。比較可能な**観測o**（例：PersonのDistinct件数、Membership Occurrence数、FTE、Measure Value）を指定し、Source identity、grain、対象集合、時間、Unit/Rule、消費先での可視範囲を揃える。満たされて初めて`o(A(R,K)) ≈ o(B(R,K))`を有限fixtureで比較できる。

- **別grainの正当な違い**：合成組織例では「Person 3、Membership 4、FTE 3.0、JOIN行4」のように、値が異なっても各観測は別のものを数えている。互いに矛盾と断定しない。
- **同grainでの不一致**：両経路が同じSnapshotのDistinct Person Countを返すと約束しながら3と4なら、Operator/Scope/重複/Source結合かTarget実装を調査し、AIの多数決で正しい方を選ばない。
- **再集計で必要な情報**：グループ平均だけを次元横断の平均へ単純平均できない。合成SQLite例では2件平均10と8件平均20から得る全体平均は18であり、算術平均15を排除。JOIN重複はSUMとSUM(DISTINCT value)の両方を誤らせる場合がある。原Fact identityとsum/countを含む観測契約が必要。
- **変化と撤回**：一方のConsumerのみ再描画された場合、もう一方の古いClaimが保持可能とは限らない。旧Claimの依存条件と実Consumerの読込み・再評価を独立に記録する。
- **正常な非可換性**：PresentationのLayout/省略、公開向けのRedaction、ある用途で認める浮動小数近似などは、保存対象となるObserverが異なる。失われる情報を明示したまま一部Observationについて同値なら、それを「全意味一致」とは呼ばない。

**利用できる既存の合成Reference**：[組織Semantic Viewsの独立JS Oracle／Draft #2596](https://github.com/hat47x/tei-platform/pull/2596)は当該PR説明で**独立JS実行27/27**と報告。Person／Membership／FTE／Joinのgrain差を反例として示す。[Measure集計の独立SQLite Oracle／Draft #2594](https://github.com/hat47x/tei-platform/pull/2594)は同PR説明で**独立SQLite 6/6**と報告。どちらも**TEI Query→Data Context→UI/SHO→Hostの製品経路ではなく、計算と期待値を持つ隔離合成Reference**である。両Referenceの出力を同一Sourceに基づく統合E2EのPASSとして合算しない。

### 15.3 開示可能なProjectionでは、表示内容以外も観測対象になる

内部状態s1・s2が、あるConsumerへ**公開を許された情報に関して区別不能**であるなら、そのConsumerが見られるProjection/Diagnostic/Traceの違いから非公開情報の存在・件数・形を推測できないことが理想的な情報非干渉の条件である。ただしすべての応答・時間・副作用についてこの条件を証明したものではない。ここでは**独立の検査義務の提案**として扱う。

例えば、二つの内部状態がどちらも外部向けに「公開可能な所属数2」と表示される場合、内部の未公開の審査対象が追加されたときに、外部向けの`unresolved: hidden member count=1`や返却件数、例外の詳細、候補の順序が変われば、本文をRedactしていても秘密を漏らし得る。

**同じClaimの正しさ、根拠へアクセスする権限、現在のConsumerへ開示する資格、実際にConsumerが受信できたことは別々**。合法な公開結果でも、反復照会・表示候補数の変化が組み合わせによる漏洩につながり得るため、単一Requestの本文比較だけで十分とは言わない。

- #2516/#1178/#2515の既存Policy・Disclosureの担当を利用。通常の業務データアクセスへTEI独自の新Authorityを被せない。
- Error分類・Missing/Hidden/Unknownを内部では保持し、低開示Consumerへ出す表現はPolicyが認める情報に縮約する。**低権限向けの理由説明自体にsecret factを含めない**。
- 外部向けに十分な根拠を開示できない場合、「内部的には正しいから外部へ全て開示可」とせず、そのConsumer用のClaimの提示を保留・制限する。
- Leak検証は公開可能な値を等しく保ち、非公開だけを変えた**二つの合成世界**を使い、レスポンス本文・コード・件数・候補・エラーメッセージ・試行間の差を比較する。実Policyの実効境界を伴わない資料上の比較は説明候補に過ぎない。

### 15.4 Headless-firstでの最小反証セット（独立Oracleを増設しない）

| ID | 読み取り・編集・開示を比較する対 | 保存／非保存の判別条件 | 検証担当 |
| --- | --- | --- | --- |
| ND-01 | 二つの厳密なInteger→同一float64表示 | 表示が同値でもSource IDの逆復元はできない | #2585 |
| ND-02 | 同じ部門人数を生む異なるMembership | CountだけのViewから個々のLink/Unlinkを自動復元しない | #2386/#2596/#2500 |
| ND-03 | 同じPerson集合で異なるRelation出現数 | Count PersonとCount Relationの観測grainを比較 | #2596/#2586 |
| ND-04 | 同じRaw Fact、JOINを変えたSUM | Fact ID/Query pathを元の数値grainへ適合させ、DISTINCT valueだけで修正しない | #2594/#2593 |
| ND-05 | 部分Relationと空確定を同じ空UIへ表示 | Retrieval ScopeとPatch Targetを分け、全Unlinkの自動書戻しを拒否 | #2597/#2500 |
| ND-06 | 外部向け同値、秘密のReview状態だけ異なる | Error/候補数/Traceの漏れを含めてDisclosureを比較 | #2516/#1178 |
| ND-07 | 二Consumers同じClaim→一方だけ更新 | Claim版・refresh受領とold displayの現在性を区別 | #2582/#2516 |
| ND-08 | 過去のReceiptと現在の表示Adapter | Effectの歴史的記録と現在版でのResult再投影を分ける | #1174/#2390 |

この8件は**既存Issueへ紐付ける判別票であり新しいScenario正本ではない**。ND-01〜08のTEI製品上の横断試験、情報非干渉の実Policy検査、二Target一致、正式な逆写像契約の受理は**NOT RUN**。先に独立ReferenceのOracleとOwnerの既存テストに接続し、対応するSource/Contract/Scopeを固定した縦断実証#2586で不足する前提だけを実装候補へ返す。

## 16. 条件付きClaim Bridgeと公開済み成果物を、別の保存義務として接続する（2026-11-02）

[#2516のW8/W9](https://github.com/hat47x/tei-platform/issues/2516)では、異なるConsumerが記述したClaim同士が「同じか」だけでなく、**あるClaimの限定的な証拠が、別Claimに対してどの方向・条件でどの強さの結論を与えるか**を扱い始めた。これを#2497で扱うProjection保存性と混ぜずに接続する。

### 16.1 三つの関係を区別する

| 関係 | 問うこと | 条件／誤りやすい推論 | Owner |
| --- | --- | --- | --- |
| **Projectionの観測保存** | 同じ対象Sourceから別形へ写した際、指定Observerの結果を保存するか | 同じ数値／同じJSONでもSource・粒度・単位・状態が違えば保存とはいえない | #2497 |
| **Claim間の条件付き導出** | 一方のClaimとその証拠から、別Claimの存在・反証・上限/下限を導けるか | 対象集合の包含、演算の単調性、時間、値域、必要な証拠を明示。一般的な双方向同値ではない | #2516/#2593 |
| **公開物とConsumerの実読取** | 固定Manifestが指す一式を、同じ世代の実ファイルとしてConsumerが読めたか | Generatorでdigestが一致しても、途中更新／旧ファイル混在／読取中の世代切替を消せない | #2604/#2482 |

Bridgeを記述するための**比較票**は、`source claim + target claim / requested inference / explicit premises / evidence scope / permitted explanation / missing premise / actual consumer`という役割を持てるが、新しい共有Claim Database、万能Comparator、証明Runtimeの提案ではない。

### 16.2 論理の推移性と、条件付きの証拠・近似の推移性は異なる

厳密な論理含意`A ⇒ B`と`B ⇒ C`が**同一の意味論・文脈・同時に満たせる仮定**の下で本当に成立するなら、`A ⇒ C`は成立する。一方、別々のSource版／Read Scope／Unit・Rule／Consumer開示条件の下に観測された二つの「Bridge候補」から、それらの前提が接合できるとは限らない。これは論理の推移性を否定する主張ではなく、**条件付きの根拠の有効範囲を合成前に照合する必要**があるということ。

特に誤差`|x-y|≤ε`で定義する近似一致は、通常は推移的な同値関係ではない。例として`a=0, b=0.006, c=0.012, ε=0.01`ならA≈BとB≈Cは成立するがA≈Cは不成立（#2516 CB-05）。**隣接するConsumerのPairwise PASSを結合して、全Consumerが同一値・同一Claimだと表示しない。** 誤差を積算・伝播するときは`ε`と対象演算の保証、丸め時点、独立基準を明示する。

### 16.3 下界・反証は一方向であり、実観測対象の包含の証拠が要る

同じ時点・対象集合の条件で`R ⊆ L`が確定しているなら、Rで観測した実体`a`はLでの存在証人になり得る。ただし、Rでゼロ件だったことはLでの不存在を示さない。R内で`P(a)=false`の反例が確定していれば、Lの全件Pの反証になり得る。一方、Rでの`∀x.P(x)`の成立だけではL全件の成立は未確認である。

集計に際しても、Rの正しいSUMが30であることだけからLのSUM≥30は一般には導けない。**追加要素が非負・同Unit・集合包含・Identity/重複規則が整合・訂正なし**等を独立根拠として得た場合にのみ、条件付きの下界を候補にできる（#2516 CB-03、#2593）。未知の非公開母集団に対する件数・境界の表示は#1178/#2516の開示資格が必要。

### 16.4 完全な生成結果と、公開された一式は別

[#2604](https://github.com/hat47x/tei-platform/issues/2604)が静的に指摘した設計成果物の直接逐次書込みでは、個々のファイルが正しく生成されていても、Manifestを受け取ったConsumerが**同じ生成世代のArtifact set**を読んだとは限らない。書込途中の失敗や旧ファイル残存は`Manifest.Complete`やDigestから解消できるとはいえない。

したがって、CUIとSHOで「同じClaim」「同じSource版」を比較するための**実読取の証拠**には、必要に応じて各Consumerが使用した`manifest/version/digest + actual read bytes/paths + read time/operation`、固定Artifact setの参照、公開方式の実測が必要。採用候補（immutable世代ディレクトリ／stage→publish／Consumer pin）は#2604で比較し、#2497側では**何を比較する前提が満たされていないか**のみを示す。

これは文書を生成した時点のModel/SHO/Artifactの同版性と、納品後の利用者が実際に読んだ版の同一性を分けることにもつながる。SHA/Digest等は内容・来歴の根拠であって、Ownerの意味受理、実Hostの権限、業務Effect、外部Consumerの理解・受入を保証する資格ではない。

### 16.5 読み取り・Claim Bridge・編集はそれぞれ異なる出口

- **Read**：固定Source/Scope/Consumerに対する観測。必要なFacet・Claimを説明し、損失・欠落・開示条件を明示する。
- **Bridge**：別Claimへ伝えられる存在証人・反例・境界・比較可能性を限定して説明する。元Claimの現在性と他Consumerへの開示資格は独立。
- **Edit proposal**：表示からTyped Intentを作り、base revision、Subject/Relation occurrence、未保存Patchの所有者、対象Context Instance、現在Authorityへ渡す。これ自体はCommitではない（#2500/#2597）。
- **Publish/consume**：一式固定版と実読取の証拠を扱う（#2604）。生成した全ファイルが同じ版を示すことと、Readerが一式を読んだことは別。

### 16.6 独立のOwnerへ渡す反証票

| ID | 固定した条件・操作 | 誤った判定 | 既存Owner |
| --- | --- | --- | --- |
| BR-01 | 一部取得RがLの部分集合、Rに存在証人 | Rで未発見→Lで不存在も保証 | #2516 CB-01/02 |
| BR-02 | 非負のSUM、R⊆Lが証明済み。追加値の符号を変える | 条件削除後もSUMの下界を維持 | #2516 CB-03/#2593 |
| BR-03 | 隣接Consumer2対がε以内、両端はε超 | 近似一致の推移閉包で全対象を同値扱い | #2516 CB-05/#2514 |
| BR-04 | R1 ManifestでR1ファイル取得途中にR2更新 | 個々のDigestが正しく見えるので一式同世代と表示 | #2604 AP-01/06 |
| BR-05 | 旧固定SHOと新CUIが同じ本文・別意味改版 | 見た目同一で現在の意味・レビュー受理を継承 | #2505/SHO #6 |
| BR-06 | 非単射Projection上で部分編集 | Viewから全Sourceを自動逆復元／完了と宣言 | #2500/#2597 |
| BR-07 | 低開示Consumerへ境界Claimを出す | 内部母集団の数・非可視存在を境界説明で漏らす | #2516/#1178 |

BR-01〜07は**既存IssueのFixture・Acceptanceを共有する連結検討票**であり独立Scenario正本ではない。外部Consumer上でのBridge、公開の障害注入、TEI Query/SHO実読込、編集/Effectの実証、Owner意味受理はいずれも**NOT RUN**。#2497は構造保存・逆写像の説明、#2516はConsumer別の条件付きClaim Bridge、#2604は物理的な公開一式、#2586は同版の実Headless縦断を所有する。


## 17. 依存閉包と実際の読取りを、同じProjectionの別の証明義務にする（2026-11-02）

### 17.1 表示の意味保存だけではFunction全体を保証しない

[#2624](https://github.com/hat47x/tei-platform/issues/2624)の合成Read Mapping/Rule依存、[#2628](https://github.com/hat47x/tei-platform/issues/2628)のProvider Read Witness、[#2598](https://github.com/hat47x/tei-platform/issues/2598)のView→Request束縛を接合する。表示されたFieldの値が正しいことと、Ruleを評価するのに必要なデータが揃うこと、実Providerが必要以上のデータを読んでいないこと、保存Actionが正しい対象へ適用されたことは**独立した検証義務**である。

同じ公開表示`Order.amount=600`でも、非表示の`Account.credit_limit`が500と700の世界では、受理済みの与信Ruleの判定が変わり得る。表示だけの依存集合`D_display`と、受理済みRuleが実際に必要とする`D_rule`を区別する。保存Actionの実行前提`D_action`、Providerが物理的に読んだ`D_actual`、Consumerへ開示可能な`D_disclose`も別。

これらの集合には**一般的な単純包含関係を仮定しない**。例えばRule入力が表示より広くても、実ProviderのOverfetchはRuleの正当な依存とは限らず、開示可能集合は必要依存集合と一致しない。静的な宣言集合と実行時に条件分岐で参照した集合も異なる。各集合の要素は単なるField名ではなく、必要に応じて`Source/Relation path + role + grain + scope + revision + principal/binding`で識別する。

### 17.2 依存閉包の最小判定

- **Declared**：Functionの明示Data Mapping、Rule Attachment、Action前提、Source/Target Bindingが何を要求するか。Markdown表・DSL・Studio表示が同じMeaningを指しているかは#2621のAuthoring契約で検証する。
- **Required**：特定の`Claim/Operation/Consumer`を成立させるための依存。条件分岐・時間・集計・外部呼出し・非表示Ruleを含め、Sourceの意味受理済み定義へ遡る。未解決依存は`unknown`であって成功ではない。
- **Realized**：Providerへ渡した実Request/Query、実効Alias/RLS/Principal/Scope、実際に読まれたField/Row/Relation。合成Providerの呼出しTraceと、実DB/OSのI/O観測を同一証拠にしない。
- **Available**：返却Result Sliceで必要なFieldが観測済みか、行集合・Relationの完全性が十分か。Rowset completeでも特定Field absentはあり得る。SQL NULL、未取得、optional missingを混同しない。
- **Disclosable**：Rule結果、Action候補の存在、エラー、Traceを含め、Consumerへ何を開示できるか。値を隠しても候補件数や拒否理由で秘密を漏らさない。
- **Effect-bound**：選択したView Occurrenceが、現行のSource/Target/Operation ID/Grant/Requestへ束縛され、実Effectと独立read-backで同じ対象を観測できるか。PreviewやIntentだけで業務完了を宣言しない。

この6項目を**既存Ownerをまたぐ試験観測票**として利用する。新しいDependency Runtime、全域的な静的情報流解析器、第二のData Context正本を導入するものではない。

### 17.3 安全性と最小性は別の性質

必要なRule入力がすべて得られたとしても、Providerが秘密Fieldを余分に読んでいれば物理的な最小性は保証されない。逆に、Providerが宣言通り狭く取得していても、必要なRule依存がMappingから漏れていれば意味的な完全性は保証されない。

**二世界比較**：公開データが同じで秘密データだけが異なる二世界を用意する。公開Projectionが同じでも、Providerの実Read、Rule結果、Action候補、Error/Trace、外部Toolへの送出が異なる可能性を独立に観測する。非干渉性を主張するには、出力の比較対象と観測できるチャネル、Policyで許可された例外、攻撃者の権限・反復可能性を限定する。有限合成FixtureのPASSを実OS/Network隔離の証明にしない。

### 17.4 負例と受入境界（DC-01〜08、全件この節としてはNOT RUN）

| ID | 対照する二条件 | 検証対象とOwner |
| --- | --- | --- |
| DC-01 | 表示値同じ、非表示Rule入力違い | Rule必要依存の閉包。#2624/#1129 |
| DC-02 | 同じ公開JSON、狭いReadと秘密Overfetch | 出力同値と物理Read最小性。#2628/#2622 |
| DC-03 | 全行取得、Rule Fieldはabsent | Rowset完全性とField観測。#2506/#2627 |
| DC-04 | Mappingはall宣言、実Providerはpartial | 宣言完全性と観測完全性。#2506/#2628 |
| DC-05 | 同じSQL ID、実RLS/Principalが異なる | 実効Source/対象集合。#2387/#2611 |
| DC-06 | 表示対象L1、実Request Payload対象L2 | Intentと実Effect対象。#2598/#1174 |
| DC-07 | 最終回答同じ、秘密Action候補の件数が違う | Planner/Tool候補の開示。#2516/#1178/SEI |
| DC-08 | Markdownの説明表を実Ruleとして誤解釈 | Authoring Roleと正式意味受理。#2621/#2626 |

**既存Evidenceの限定**：#2624は隔離合成Mapping20件＋Rule Preview18件、#2628は隔離Synthetic Provider Read Witness12件を報告している。これらの試験は独立に存在し、ここで再実行したものではない。実TEI Rule/Provider、実RLS/OS、外部AI egress、DB Commit/Receipt、SHO/Studioでの受入は別Ownerの未実施条件。#2497は各段階で何を保存・喪失したかを説明し、実Providerは#2622、Contextは#2506、Ruleは#2624/#1129、開示は#2516、書込みは#2598/#1174、Authoringは#2621へ戻す。


## 18. SQL→Data Operatorの意味保存と、編集トランザクションの合成境界（2026-11-02）

### 18.1 Operator名・構文の一致を意味保存の根拠にしない

[#2387](https://github.com/hat47x/tei-platform/issues/2387)のQX-23はSQLを共通Data Operator候補へ写す際、構文変換と意味的同等性を分離する。[#2622](https://github.com/hat47x/tei-platform/issues/2622)のData Operationsと、[#2514](https://github.com/hat47x/tei-platform/issues/2514)の独立Oracleを接続する。

SQLの`JOIN`、`EXISTS`、`LEFT JOIN`、`NOT EXISTS`、`DISTINCT`、`GROUP BY`、`ORDER BY`、`LIMIT`は、同じ結果列名や値だけで意味を比較できない。保存義務は、**入力Bagの出現・同一性、NULL三値論理、結合の行成立、フィルタ適用順、集計粒度、順序とTie、部分取得、実効Scope/RLS、演算副作用・非決定性**に及ぶ。

- `JOIN`と`EXISTS`は、重複する右側行があるとBagの多重度が異なる。真偽の存在判定と行増幅を混同しない。
- `NOT IN`と`NOT EXISTS`はNULLを含む入力で異なる場合がある。SQLのunknownを二値falseへ黙って潰さない。
- `LEFT JOIN`の不一致で生じたNULLと、実在する右側行のNULL、未取得Fieldは別の来歴を持つ。
- `LIMIT`の前後でFilter/Sortを入れ替えると、対象集合が変わる。順序を指定しない`LIMIT`を安定した先頭集合とみなさない。
- `DISTINCT`、`GROUP BY`、平均や比率の再集計は、Fact ID/Grain/中間状態を失うと後続演算が一致しない。
- SQLの字句・IRの同形、合成SQLiteでの一致、実TargetのQuery Plan/実結果、Host/Owner受理は別の証拠である。

### 18.2 変換の可換性は、特定の演算・観測条件に相対的

Source`S`、Projection`p`、演算`op`について、`p(op_S(S)) = op_T(p(S))`が成り立つかは**対象演算、入力条件、Target意味論、Observer**を指定して検証する。値だけの一致ではなく、Bag、NULL、Scope、順序、精度、エラー、時点を含める。一般の`p`が全演算に対して可換であるとは主張しない。

SQLを正本とする既存Queryから候補Operatorを生成する場合、元Queryの意味・版・Target bindingを保持し、実Providerで独立Oracleと比較する。形式的に一般証明できない場合は、対象Fixture・Target・版に限定したEvidenceとして扱う。共通Data Operatorの導入を、SQL正本の置換や第二Query Runtimeの採用条件としない。

### 18.3 複数Functionの編集成功と、全体の不変条件は別

[#2597](https://github.com/hat47x/tei-platform/issues/2597)のEditing Sessionと、[#2601](https://github.com/hat47x/tei-platform/pull/2601)の局所Transaction Referenceを接続する。各Functionがそれぞれの入力検証と保存に成功しても、**並行実行後の業務不変条件**が成立するとは限らない。

例：口座A/Bの残高が各100、両者の合計が100以上という業務条件を維持する。二つのSessionが同じSnapshotを読み、それぞれAまたはBから100引く判断を行い、互いに別行を更新すると、個別の更新条件は通っても最終合計0となり得る（Write Skewの候補）。実際にその現象が起こるかはDB分離レベル、ロック、Predicate検査、HostのTransaction境界、業務Ruleに依存する。**合成SQLiteのBEGIN/ROLLBACK/SAVEPOINT成功を、実TEI/PGでのWrite Skew実証へ昇格しない。**

編集の観測票は、`Function Instance / Editing Session / base revision / Read set and predicate / Save Intent / Target binding / actual Tx boundary / Commit outcome / Receipt / independent post-read / business invariant`を区別する。途中失敗、Commit結果不明、再試行、再評価時のRule/Grant変更を個別に扱い、見えているViewの保存成功を業務全体の確定としない。既存外部HostのTransaction/認可/副作用制御を利用し、TEI独自Hostや共通Transaction Coordinatorを新設しない。

### 18.4 反証票（OP-01〜12、当節としてNOT RUN）

| ID | 二条件の比較 | 不正な昇格 | Owner |
| --- | --- | --- | --- |
| OP-01 | JOINで右側2行、EXISTSで存在のみ | 行増幅と存在判定を同一視 | #2387/#2622 |
| OP-02 | NOT IN対象にNULL、NOT EXISTS | NULL三値論理の差を消す | #2387/#2514 |
| OP-03 | LEFT JOIN不一致と右側実NULL | 同じNULL表示から来歴を同一視 | #2387/#2506 |
| OP-04 | LIMIT前後のFilter/Sort交換 | 演算順序を自由に交換 | #2387/#2622 |
| OP-05 | DISTINCTで異なるFactが同値 | Fact ID/Bag粒度の喪失 | #2593/#2387 |
| OP-06 | AVG再集計の中間状態欠損 | 値型一致で可換と宣言 | #2593/#2544 |
| OP-07 | SQLite合成一致・実Provider未実行 | 全Targetで意味同等と宣言 | #2387/#2498 |
| OP-08 | 同じSQL ID、異RLS/Principal | 同一母集団を宣言 | #2611/#2622 |
| OP-09 | 二Functionが別行を並行更新 | 個別成功で全体不変条件成立 | #2597/#1174 |
| OP-10 | 保存前にRule/Grantが改版 | Preview時の判断を現在権限とする | #2598/#1174 |
| OP-11 | Commit応答喪失・再試行 | 結果不明を未実行と同一視 | #1174/#2597 |
| OP-12 | UIは保存成功、独立post-read無し | Effect/業務完了を確定 | #1174/#1119 |

**証拠の境界**：#2387 QX-23では隔離SQLite 9件の成功が報告され、#2601ではSQLite局所Transaction 3件の成功が報告されているが、この節で再実行していない。これらを足してTEIのQuery→Function→Host→DBの12件E2E PASSとはしない。#2497は演算保存の条件を記述し、SQL/Operator実証は#2387/#2622、編集合成は#2597、実Effect/Recoveryは#1174、独立期待は#2514が所有する。


## 19. 空集合の依存・選択的再資格化・Product Skill実利用の三つの時間境界（2026-11-02）

### 19.1 空結果は「依存がない」ことの証拠ではない

[#2582](https://github.com/hat47x/tei-platform/issues/2582)、[#2387](https://github.com/hat47x/tei-platform/issues/2387)の動的Lineage検討と、SHO #6のClaim再評価を接続する。Queryが0件を返しても、`WHERE`、`NOT EXISTS`、JOIN経路、関連するRelation membership、RLS/Scope、時点・Windowに対する**否定的依存**は残る。0件は指定条件下の観測結果であり、全宇宙での不存在・将来の不変性ではない。

例えば`not exists(select 1 from allocations where order_id=O1)`が真であった後に、同じOrderへAllocationが追加されれば結論は変わる。再び削除されて0件に戻っても、途中で変更がなかったことは導けない（ABA）。同じ結果値・同じ集合・同じDigestは、観測間のイベント履歴やRule/Grantの変化を証明しない。

必要な依存は、単なる表示Fieldだけでなく、`Value lineage / Predicate dependency / Relation path binding / Negative dependency / Read cohort and completeness / Change observation window`を**既存Query・Function Context・Provider・Reactive Contextの参照証拠**として扱う。新しいNegative Cache、Snapshot Manager、共通Reactive Engineを設けない。

### 19.2 変更影響と現在の利用資格を分離する

変更が検出されたとき、影響を受けるClaim/Consumerだけを再評価する候補はある。しかし、依存集合・Change Observation Window・Source版・実Provider Scopeが未確認なら「影響なし」と確定しない。逆に、無関係な変更を理由に全Claimを常に失効させると、増分再利用の価値を失う。

**Change Impact**（どの依存に触れたか）と**Current Eligibility**（現時点・当該Consumerで利用可能か）は異なる。Claimの意味・取得完全性・証拠現在性・開示Policy・実行Grant・納品物の実ロードはそれぞれ別の資格である。意味上再利用できる固定Artifactであっても、現在のPrincipalに開示・実行できるとは限らない。

選択的再資格化は、十分な依存証拠がある範囲でのみ適用する。未知の依存は`unverified`として留め、値の一致を「変更なし」の証拠にしない。#2582がChange/Invalidation、#2516がConsumer Claim、#2018が再利用・配布の資格、TAI #271がHarness側の継続資格を所有する。

### 19.3 Capability発見、Skill呼出し、実行効果を別の段階として観測する

[#2619](https://github.com/hat47x/tei-platform/issues/2619)、[#2605](https://github.com/hat47x/tei-platform/issues/2605)とTAI #271の横断検討では、意味Role→Operation→CLI→Product Skill→Consumer→実Effectを同一視しない。

- Model/RoleにOperation候補が存在しても、当該TargetでCLI・APIが実装済みとは限らない。
- CLIにCommandが存在しても、SkillがそのCommandを実際に呼べる、正しい引数・Workspace・版で呼んだとは限らない。
- Skillの説明が読み込まれても、HarnessがそのSkillを選択し実行したとは限らない。
- Tool候補が内部に存在しても、外部Consumerへ存在・件数・Errorを開示してよいとは限らない。
- CLI/Toolが成功しても、Provider・外部Host・業務Effectが成立したとは限らない。

最小の受入は、同じ固定Source/Revisionから`commands → model catalog → inspect → trace → validate -json`等の**実read-only CLI**を確認し、次にSkill経由の実呼出し・同一AttemptのInput/Output/Consumed Artifact・中断後の再資格化を確認する。正式なCommand列やSkill公開仕様をここで新設しない。SHO文書Reader、外部LLMへの送信、OS隔離、実Host Actionは後段の独立証拠。

### 19.4 反証票（NE-01〜12、当節としてNOT RUN）

| ID | 二条件の比較 | 誤った昇格 | Owner |
| --- | --- | --- | --- |
| NE-01 | NOT EXISTSで0件→追加1件 | 空集合を依存なしと扱う | #2387/#2582 |
| NE-02 | 0→1→0、両端の表示同値 | 履歴変化なしと断定 | #2582/SHO #6 |
| NE-03 | WHERE条件にだけ現れる非表示Field変更 | 表示Lineageだけで影響なし | #2387/#2624 |
| NE-04 | Relation付替え、同じValue集合 | 関係経路の変更を無視 | #2387/#2506 |
| NE-05 | 取得一部で0件、全件未確認 | 全対象不存在と断定 | #2506/#2516 |
| NE-06 | 異Providerの同名Snapshot token | 同じRead Cohortと宣言 | #2611/#2622 |
| NE-07 | 無関係Source更新と依存Source更新 | 両方を一律失効／両方再利用 | #2582/#2018 |
| NE-08 | Claim意味不変、Disclosure Policy変更 | 意味再利用から公開資格を推定 | #2516/#1178 |
| NE-09 | Roleは存在、CLI Command未確認 | 意味定義から製品利用可と宣言 | #2619/#2605 |
| NE-10 | Skill文書は読んだがTool未呼出し | Skill loadを実行成功と宣言 | #2605/TAI #271 |
| NE-11 | 同じ最終回答、異なるTool候補/送信 | 回答同値で情報流同値とする | #2516/TAI #271 |
| NE-12 | CLI read-only成功、Host Effect未実行 | 読取り合格で業務完了と宣言 | #2598/#1174 |

**Evidence区別**：他レーンのQX-31隔離SQLite12件、TAIの合成Harness試験、TEIの限定CLI/CI試験はそれぞれ報告範囲に限定する。NE-01〜12を本節で実行しておらず、実Providerの変更観測・実CLI→Skill→外部Harness・SHO Reader・Host Effect・Owner意味受理はNOT RUN。設計候補の新しいCore型・全域的資格判定Engineは採用しない。


## 20. Artifactの意味Envelope・公開byte・実Consumer読取の三層照合（2026-11-03）

### 20.1 Digestの対象を同一視しない

[#2630](https://github.com/hat47x/tei-platform/issues/2630)と[#2604](https://github.com/hat47x/tei-platform/issues/2604)の実コード観測により、設計成果物の`Artifact.Digest`は`Content`だけのraw SHA-256ではなく、`Kind / InputRevision / Rule / Generator / Content / Unresolved`を含む**Envelope fingerprint**である。一方、現行`tei artifacts -out`が出力する個別ファイルは`Content`を保存する。現行Manifest EntryからはEnvelope検証に必要な全入力を復元できない。従って、`Manifest`のdigestを保存ファイルのraw hashと比較するだけでは同一性を判定できず、出力DIRとManifestだけで`Artifact.Verify`を再現できると仮定しない。

この問題を、既存§16の「一式の物理的な公開」と区別する。**(A) Source/Rule/Generator/Unresolvedを含む生成時の意味Envelope、(B) 実際に公開したContent bytesと一式の世代、(C) Consumerが実際に取得・利用したbytesと時点**は異なる証拠。Aの自己整合がBの完全公開を保証せず、Bの整合がCの同一世代読取を保証しない。Cの一致はOwnerによる意味受理・実Host権限・納品先動作を保証しない。

### 20.2 実装のOwnerと必要な最小検証

- **#2482/#2630**：既存Artifact Envelope/CheckSet/Manifestの整合、ExpectedMemberとDeclared/Observed/Consumedの差、Gap→Unresolvedのexact multiset。新しいArtifact正本を別に作らない。
- **#2604**：公開前の一式構築、途中失敗、旧ファイル残存、読取中の世代切替、Content bytesとManifestの検証可能性。既存`-out`の出力形式・後方互換を確認してから公開方式を選ぶ。
- **#2516/SHO**：Consumerが実際に読んだArtifactの世代・bytes・Source occurrenceを追跡し、文書引用とClaimの現在性を区別する。
- **#2605/TAI**：Product Skill/CLIが実際に消費したScript/CLI/Artifact bytesとCommand実行を観測し、説明文のload成功を実消費へ昇格しない。

### 20.3 対照実験（AE-01〜08、当節としてNOT RUN）

| ID | 変更・故障 | 検証すべき差 |
| --- | --- | --- |
| AE-01 | Content同一・Ruleのみ変更 | raw Content hash同一でもEnvelope fingerprintは変化し得る |
| AE-02 | Content同一・Generator版変更 | 生成規則の差をbyte同一で消さない |
| AE-03 | Content同一・Unresolved説明変更 | 未解決の意味・理由をContent hashだけで同一視しない |
| AE-04 | Manifest取得後に一部ファイル更新 | Consumerが同じ世代を実読取できたか |
| AE-05 | 途中失敗で新旧ファイル混在 | 生成側のCompleteと公開一式を区別 |
| AE-06 | ExpectedMemberと実出力が異なる | 必須成果物の欠落・余分・重複を検出 |
| AE-07 | SHOが旧Artifactを引用、TEI Source更新 | 固定文書の来歴と現在Claimを区別 |
| AE-08 | Skill本文同一、依存Script/CLI変更 | 実消費依存閉包の再資格化 |

**実証順序**：まず既存`Artifact.Verify`と`tei artifacts -out`の実保存形式を固定checkoutで確認し、AE-01〜03のbyte/Envelope差を独立Oracleで再現する。次にAE-04〜06の公開障害注入を#2604へ、AE-07〜08の実Consumer/Skill消費をSHO/#2605へ渡す。出力互換・API変更・公開方式はOwner判断待ち。#2633のmain統合済みCharacterization、#2659のDraft修正、#2607のDraft Skill CIを混同せず、別HEADのCI成功を現在HEADへ転用しない。**新規E2E・Windows/SHO/Harness実読取・Owner意味受理は未実施**。


## 21. Data Model正本・Queryの二経路・生成アクセスの性能境界（2026-11-03）

### 21.1 正本をSQL解析結果へ逆転させない

2026-11-03横断の設計判断として、**Table / Field / Key / RelationはData Modelへ直接宣言した構造を正本**とする。SQLの静的解析から得られるTable/Relation/Keyは、既存DBの取込や診断に用いる**候補**であって、Owner受理済みモデルへ自動昇格させない。SQLは第一級のQuery/変換定義として扱い、通常の取得はData ModelのRelationとFunction Data Contextから対象・経路・範囲を明示し、複雑な集計やDB方言固有処理ではNative SQLを保持する。

**TEI Query**と**Native SQL**の二経路は、単一の実行構文への強制変換を意味しない。SQL→TEI QueryのAI変換は、Source SQLを保持した候補とし、NULL/欠損/空集合、Bag、Collation、Decimal、時点、Scope/RLS、Filter/Sort/Limit、Provider pushdownと局所実行の差を独立Oracleで比較する。生成SQLの実行成功だけでは、元Queryの意味保存を受理しない。構造正本はData Model、Query正本は当該QueryのAuthoring形式、実効ReadはProvider Witnessとして**別の責務**にする。

### 21.2 実行時の検証を無条件に積み増さない

性能を製品品質として扱い、検証を`編集時 → ビルド時 → Provider/Host境界 → 実行時`へ適切に配置する。静的に確定し、生成物・Binding・依存版へ束縛できる不変条件を、各行・各Fieldの実行ごとに重複検証する必要はない。一方、現在のAuthority、外部Input、実Providerの結果完全性、Transaction競合、Effectの結果など、**実行時にしか確定できない条件は省略対象としない**。検証省略は対象条件・前提・失効トリガ・負例・性能差を示して判断する。

生成アクセスについては、Functionごとに明示した`Target / Relation path / Field / Predicate / Scope / Completeness`からQuery計画を生成し、N+1、JOIN fanout、過剰取得、未取得Field、Pushdown順序差を検証する。生成コードの短さやCLIの成功は、実Providerのp50/p95・行数・Read set・Query count・Memory・失敗時のEffectを保証しない。最適化がRead Scope、開示、Ruleの依存閉包を変えないことを先に検証し、性能を後測定する。

### 21.3 受入票（DM-01〜10、当節としてNOT RUN）

| ID | 正例と負例 | 受入の境界 | Owner |
| --- | --- | --- | --- |
| DM-01 | DDLのFK候補とOwner受理Relationが相違 | SQL解析を構造正本に昇格しない | #1126/#1129 |
| DM-02 | 同一Data ModelからTEI QueryとNative SQL | Source/Authoring正本を区別 | #2387/#2622 |
| DM-03 | AI変換でNOT INのNULL差 | 変換成功と意味同等性を分離 | #2387/#2514 |
| DM-04 | LIMIT/FilterのPushdown順序変更 | 取得範囲と演算意味を保持 | #2622/#2628 |
| DM-05 | FunctionでRelation経路が複数候補 | 明示Bindingと曖昧診断 | #2624/#1129 |
| DM-06 | 一覧100件の関連取得が101 Query | N+1を測定し改善 | #2628/#2018 |
| DM-07 | JOINで100件→300件 | Fanoutと業務粒度を分離 | #2593/#2622 |
| DM-08 | 静的型検証を実行時に重複 | 証明済み契約の安全な省略条件 | #2018/#2514 |
| DM-09 | 実行時Grant失効・Provider部分取得 | 静的成功でも境界検証を残す | #1174/#2628 |
| DM-10 | 最適化前後で値同一・Read Scope異なる | 出力値だけで最適化適合としない | #2628/#2516 |

**導入順**：M0＝現行Data Model/SQL/Function MappingのOwnerと正本を固定。M1＝同一Sourceに対するTEI Query/Native SQL/独立Expectedを比較。M2＝実ProviderでQuery count・Row count・p50/p95・Read Witnessを採取。M3＝生成コード/外部Hostで境界検証と省略条件を検査。M4＝Owner意味受理と実業務Effectを別票で判断。新しい汎用SQL Runtime、TEI独自Host、第二Data Model正本は設けない。

<!-- tei-data-form-crosslane-realization-observation-20261106 -->
## 22. データ様式の横断受入：構造・実現方式・観測・Claimを分離（2026-11-06）

**位置づけ**：既存#2497の構造保存/Projection/編集逆写像を、Everything is Data（#2619）、Function Data Context（#2506）、Query（#2387）、Data Operations/Provider（#2622）、Consumer（#2516）、横断実証（#2586）へ結ぶ設計追補。新Core TypeKind、新Cursor Schema、第二のQuery/Cache/Host/Registry/Authority、独立GateやTest SSOTは作らない。業務意味・権限・実装のOwnerを移管しない。

### 22.1 同じデータという言葉が指す、異なる6つの同一性

| 観点 | 固定/比較すべき内容 | 同一であっても証明しないもの | Owner |
| --- | --- | --- | --- |
| Meaning/Definition | Subject、構造Role、Relation occurrence/向き/多重度、Value Domain、単位、時点 | 値・Operation JSONが同じだけで業務意味も同じ | #2495/#1126/#2619 |
| Authoring/Source | 原文Occurrence、byte/Revision、引用位置、非対象trivia | World fingerprintや再描画が同じだけで原文が保全済み | #2621/#2497/#2600 |
| Resolution/Realization | Function/Purpose、DB Instance/Provider/Route、明示Binding、候補の有無/版、選択SQL/Operator | Query ID・共通SQL byteだけで選択も同じ | #2611/#2622/#2387 |
| Actual observation | 実Principal/Scope、Params/Sort、Read Cohort、Page/Rowset/Field coverage、complete/partial | Cursorあり/短いPage/静的解析PASSだけで全件Read済み | #2506/#2627/#2586 |
| Transformation/Claim | 問い・対象粒度、演算法則、保存性、非出力依存、必要な集合閉包 | 結果値・行数一致やcache hitで全Claimの有効性 | #2497/#1063/#2593/#2582 |
| Disclosure/Effect | 現在のConsumer/Grant、実受信、更新Target、Commit/Receipt/独立Read | 内部根拠や旧Grant、Preview/HTTP successで現在開示・更新完了 | #2516/#1174/#2604 |

これは既存Evidence/Source Index/Query Contract/Function Contextを結ぶ**観測の問い**であり、新しい統一データレコードの宣言ではない。結果の存在、全体不存在、Top-N、正確な合計、編集対象、引用・外部開示は必要条件が異なる。局所的な正のWitnessがある場合でも、完全なCOUNT/NOT EXISTSや業務完了へ無断昇格させない。

### 22.2 SQL Realization変更とPage継続の境界

既存[#2387 QX-26](https://github.com/hat47x/tei-platform/issues/2387#issuecomment-6577013855)は同じSQLでPage間のDB更新によりSnapshotが混合する反例。[#2586 HJ-SV-01〜05](https://github.com/hat47x/tei-platform/issues/2586#issuecomment-6658613966)は、**不変DBでもQuery ID一定のまま選択SQLが変わるとPageを混合できる**別反例である。両者を新Fixture SSOTへ重複登録せず、#2586の同一Function横断で接続する。

独立Python/SQLiteの既存11/11 PASS（SHA-256 a68f663d325fe45711bd3841f8eb297d0afd55766d777fa3f595e32b69092198、Repository未保存、CIなし）の例：公開ID 1〜5について共通SQLの全結果は [1,2,3,4,5]、open専用SQLは [2,4,5]。2件ずつのOFFSET Page取得で旧Page1 [1,2] と新SQL Page2 [5] を連結すると [1,2,5] となり、どちらの完全結果とも異なる。最終Pageが短いというだけで complete とする実装は偽陽性になり得る。SQLiteによる実SQL実行と、自作の選択/Continuation判定関数は別の限定証拠。TEI selector/実PG/SQLGlot Worker/Host/外部ConsumerのPASSではない。

再利用判断は以下を別に扱う。
1. **選択判断**：Query/Purpose、実DB/Provider、選択規則・明示Binding・候補集合・「専用候補不在」の負依存。専用ファイル後着なら再評価。共通SQLを明示固定していれば無条件の専用化はしない。固定済み専用SQL消失時は無音共通Fallback禁止。
2. **選択後解析**：実選択SQL/テンプレート分岐のSource byteとSchema、Dialect、Analyzer/Operator/Plugin版、実消費依存閉包。以前の共通SQLの解析Cacheを別SQLへ転用しない。
3. **実Read/Continuation**：実選択Realization、Params/Scope/Principal/Sort、Provider Cohort/issued token保証、各PageのRowset/Field coverageと要求集合の終端。Query ID同一・短い最終Pageは complete の証拠にならない。
4. **Claim/公開**：Predicate/Bag/Row membership/順位/不存在/集計の成立条件と、現在のConsumer Grant/Disclosure。非公開DB Alias/SQL Path/候補の有無・hidden件数をPublicのErrorやAI/Tool JSONへ漏らさない。

**P0停止条件**：Page1とPage2で選択Realizationが違えば旧Continuationを黙って使わず安全に停止・再開始候補を提示する。候補集合のみ変わり選択SQLが不変なら、選択を再検証し同一Cohort/Scope/Grantなどを実証できた範囲に限り継続を検討する。将来のRealization跨ぎ継続はCursor状態の対応を含む明示された同値契約と実証がない限り許可しない。

### 22.3 他データ様式への反証可能な展開

| 様式 | 同じ形や値でも異なる意味になる反例 | 保存・完了を問う対象 | Owner/現資格 |
| --- | --- | --- | --- |
| Table/Relation | SQL切替・JOIN fanout・部分Pageで全体0件と誤認 | 行のID/粒度/Bag/Scope/同一Cohort/完全性 | #2387/#2506/#2622。SQL切替のみ独立SQLite限定PASS |
| Graph/Tree | 同labelの別Node統合、多重有向辺の喪失、Tree投影で循環/向き欠落 | Node/Edge identity、source/target、方向、多重度/辺属性、探索閉包 | #2495/#1126/#2497。実Graph Provider未検証 |
| Time-series/Event | Event timeと受信時間の混同、late arrival/訂正/撤回で旧Windowが失効 | Window境界、順序、最終化保証、retract/replace、時点 | #2582/#2593。実Stream Provider未検証 |
| N次元/科学データ | Shape一致・軸/CRS/単位違いを同値と誤認 | named axis、coordinate、unit/CRS、precision、missing mask | #2495/#2498/#2501。外部Target Conformance未検証 |
| Document/Markdown/DOT | 構造同値でも空白/コメント/layout/引用Occurenceを喪失 | 原文byte/span/Revision、非対象byte、逆写像可能性 | #2497/#2621/SHO。実Writer/Reader未検証 |
| Measure | Stockの時間SUM、平均の単純平均、異単位SUM、JOIN行増幅 | Measure×集計軸×時点×演算、独立Expected、Grain | #2593/#1705/#2514。汎用Measure Engineは不要 |

Graph/時系列/N次元への適用は**設計仮説**であり、SQLite11件を各Provider/構造の実証へ転用しない。構造facet自体に現在の実ReadやDisclosure資格を埋め込む必要はない。

### 22.4 既存Critical Pathと同一Function横断の最小計画

新しい縦断表や製品Gateを追加せず[#1119の4最小Slice](https://github.com/hat47x/tei-platform/issues/1119#issuecomment-6655761834)と#2586 HJ-02/03/04・HJ-SV-01〜05へ次の順で接続する。

| 順 | 必要な成果 | 成立しないなら停止する事項 | Owner |
| --- | --- | --- | --- |
| A（既存局所実証） | 独立SQLite11件と既存QX-26の差、手書きExpectedを保持 | 合成判定器を実TEI Providerとして扱わない | #2586/#2514 |
| B（次のP0） | 実TEI Query ID→DB/Provider Binding→実選択SQL→実解析が消費したSQL/Schemaを一Attemptで固定 | 実Selector/Worker未提供なら NOT RUN: producer-unavailable | #2611/#2622/#2387 |
| C（次のP0） | 同一Function ContextでPage1/2の実Read Cohort、Rowset/Field、terminationを観測 | 実Readなしなら NOT RUN: physical-read-unavailable、Cohort保証なしの全体COUNT/NOT EXISTSはunknown | #2506/#2627/#2586 |
| D（P1） | Public/内部Consumerに実送出した許可情報、Grant失効/後着Responseを検証 | 値一致で現在開示資格や非干渉を認めない | #2516/#1124/SHO |
| E（P1並行） | Graph、時系列、文書のうち異質な二構造の保存性を独立反例で比較 | 新型追加や見栄えのよいGraph表示を意味保存の完了としない | #2495/#2497/#2498 |

実証は1 Providerの2 SQLから始め、第二DB、Studio/SHOの全実装、Writer/Effect、セルフビルドはP0のHard dependencyにしない。Source/Query/Rule/Grant変更時は影響するClaimだけ再資格化する。Source A→B→Aでも中間履歴や現在権限が保たれたとは推定しない。性能・追加Adapter/コード・誤ったcomplete/false-qualification件数は#2018へ、独立Expectedは#1131/#1180/#2514の既存正本へ委譲する。

**2026-11-06の資格**：文書追補。既存独立SQLite11件の再実行成功とGitHub記録の照合はあるが、TEI実Selector/Worker、二SQL実Provider Page、RLS、SHO/AI実送信、Owner意味受理はNOT RUN。PR #2496はDraft維持・main未統合。新Public Schema、Host、Cursor/Query/Cache Engine、採用済みADRは増やさない。

<!-- tei-data-form-bitemporal-relation-claim-composition-20261106 -->
## 23. 時間付きRelationの構造変換：値一致・履歴・更新対象を分離する（2026-11-06）

**今回の追加対象**：§15（一般の非可逆写像）、§16（条件付きClaim Bridge）、§22（六つの同一性）を繰り返さず、**一つの構造から得た複数のProjectionが同じ値になっても、valid/known時点・Relation occurrence・Claim必要条件を保存しているとは限らない**ことを、組織E1→E2の未接続境界に結ぶ。#2497が構造保存/編集の共通条件、#2389/#1126が組織/Relation意味、#2506が実Function Context/Result Slice、#2586が同一Attemptのread側、#2582が失効、#2516が開示を担う。二重のTemporal Modelや新Core TypeKind、新証拠Registryを作らない。

### 23.1 同じ業務有効日でも、知識時点が異なれば集合が異なる

公開人工Fixture（外部の実業務データではない）を次のように固定する。対象有効日を2026-02-05とし、観測される組織Relationを当初の知識時点2026-02-05と、訂正後の知識時点2026-03-01で比較する。

| Relationship occurrence | Person→Unit | Valid time | Known time | 備考 |
| --- | --- | --- | --- | --- |
| r1 (旧版) | P1→U1 | 2026-01-01以降 | 2026-01-02〜02-10 | 当初の認識 |
| r1 (訂正版) | P1→U1 | 2026-01-01〜02-01 | 2026-02-10以降 | 同一occurrenceの有効期間訂正 |
| r2 | P1→U2 | 2026-02-01以降 | 2026-02-10以降 | 遡及追加 |
| r3 | P2→U1 | 2026-01-01以降 | 2026-01-03以降 | 無変更 |
| r4 | P1→U3 | 2026-02-01以降 | 2026-02-10以降 | 遡及追加 |

有効日2026-02-05で **knowledge cut 02-05** のRelation集合は r1,r3、**knowledge cut 03-01** は r2,r3,r4。Distinct Personは双方P1,P2だが、Relation occurrenceは2→3となる。「人物が二人いる」というClaimの値と「P1の所属」「組織別Count」「編集対象occurrence」は異なる。U1/U2は共に画面名Salesという人工条件も含むのでlabel一致からUnit Identityは復元できない。

この例では *valid time（その関係が業務上いつ有効だったか）*、*known time（その解釈をいつ把握していたか）*、*Provider/Read Cohort（実際にどの入力を読んだか）*、*Consumerの現在Grant/送出時点* を混同しない。最後の二つは当Fixtureで実観測しておらず、上流二時点の値だけで作り出さない。別Providerをjoinする際に同じタイムスタンプ文字列があっても統一Snapshot/Authorityは未確立。

### 23.2 変換が保存する情報は、後段で新たに生み出せない

- **Relation Graph→Tree**：循環・多重辺・source/target・relation occurrence/valid-knownを失うTree表示を許す場合、Tree→JSONがそのTreeの値を完全保存しても、元GraphのRelation occurrenceを復元する資格は回復しない。
- **Relation Graph→Distinct Person List**：人物P1/P2のIdentityとPerson Countは保存し得るが、同じPersonからU2/U3へ向かう2本の関係と、どちらの関係が訂正されたかは失う。Person一覧のcache再利用をMembership Count/所属別集計/Business Scope/書戻しに転用しない。
- **失った性質を未申告から推測しない**：各段の「保存していると宣言した性質」は一般の証明でなく、既存#2497の構造写像契約に対する有限な義務候補。後段の判定は、Claimが要求するIdentity/Occurrence/Time/Scope/Grainの各前提と独立Expectedを要する。
- **編集への逆写像**：Person ID P1が一意でも、所属更新というPurposeならr2/r4から更新対象を一意に決定できない。これに対し「Person名の編集」ならPerson IDが対象候補として十分な場合はある。いずれも実Current Grant/Expected Version/Host実行資格は別証拠。曖昧なGraph/Tree値から自動Submitせず、read-onlyまたは明示的な対象選択へ戻す（#2497/#2621/#2598）。
- **Sourceの書式忠実度**：このGraph-to-Treeで生じる構造損失とMarkdown/DOTの空白/comment/layoutやSource occurrence喪失は別。元Sourceへ自動再描画・無条件上書きせず、既存#2621の局所Patch契約を再利用する。

### 23.3 Claim相対の必要入力を、単なる「観測済み」の集合で代替しない

対象集合L、検査できたR、対象cutを固定できるとき、Rに属する存在証人は局所の存在を支持し得る。一方、部分取得で見つからないことは完全な不存在を支持せず、正確な件数にはClaim必要範囲のRowset/Field completenessと同一Read Cohortが必要である。全件completeという**人工フラグ**を立てたテストはProviderが実際に完了保証を発行した証拠ではない。

更新・訂正・追加で同じDistinct値が維持される場合、Person Listの結果を再利用可能な候補としつつ、Relationに依存するCount/Policy/Graph/SHO引用は必要範囲で再資格化。無変更r3の過去観測は履歴として保存できても、現在Grant/現在Provider readが既に成立したとするわけではない。A→B→Aのbyte一致・画面表示一致は時系列の連続性/Owner受理の復活を証明しない。

### 23.4 局所の独立合成Fixtureと未成立の境界

2026-11-06の独立Python標準ライブラリFixture：14/14 PASS、py_compile PASS。ローカルSource SHA-256 = f2e67255a87507b197092d6f5caed90f88e017eeeb39824aa60381a3a1f2b8ae。**Repository未保存・GitHub CI未実行**。固定された期待値を手書きしており、TEI Parser/Query/Provider/実RLSを実行したテストではない。

| 試験 | 局所で反証した誤り | 既存実証の次の受渡し |
| --- | --- | --- |
| 01〜04 | 同じvalid日/Personの数でもknown cut/Relation出現数は違う。表示名はIdentityでない | #2389/#1126/#2506 |
| 05〜08 | Tree/JSONが表示を保存してもGraph occurrenceの復元不可。未宣言のRead Cohortはunknown | #2497/#2621/#2514 |
| 09〜12 | 部分取得の局所存在と全体不存在/正確件数は非対称。異Cohortはcomplete Claimを保留 | #1063/#2506/#2586 |
| 13〜14 | 同一Personの複数Relation更新対象は曖昧。無変更部分の歴史的根拠は保持し得る | #2598/#2582/#2516 |

**次の正式な検証順（2026-11-06 補正）**：
1. **E2a：最短の直接Relation Read**。#2681のNative `CaseRecord.handlingUnit` を、Case ID/Relation Role/元の値で識別し、#2586で実Function Context→実Provider Row+Field/Scopeを一件観測する。独立Edge ID、未実装のRowVersion、二時点検索は先行必須条件にしない。現実にはCaseRecord Persistence/実Provider Binding未接続なら `NOT RUN: physical-read-unavailable` を維持する。
2. **E2b：実体化Relationの時間差**。Membershipの独立 `membershipId` / personId / orgUnitId / validFrom / knownFromを使用し、同一有効日に異なる認識時点でのRowset・Distinct/Count・partial/unknownを比較する。ただし `validFrom`/`knownFrom` は現Nativeでは通常属性であり、時点Queryの意味・適用やProvider pushdownが実装済みではない。現Fixtureのr1〜r4は仮の実体化Relation履歴であり、#2681 CaseRecordの実行証拠ではない。
3. #2497/#1124は、同一実ResultからGraph/Tree/Tableのread-only Projectionを作り、保存された情報/喪失箇所と編集対象一意性を比較。非可逆なら一般的なWrite APIの暗黙呼出しは禁止する。
4. #2516/SHOは現在Consumer Disclosure、Source引用Occurrence、実受領/現行版を確認。#2598/#1174の実Write/Commit/Receiptを**read-only試験の合格へ混ぜない**。

**既存P0（SQL Realization→Page Cohort一Provider）をブロックしない**。この時間付きRelation→Projection試験は#2497の構造保存P1並行圧力であり、データ表現の型追加、二DB、全組織Policy、SHO/Studio完成、ランタイム統一、分散Txを先決しない。比較指標は誤ったClaim昇格、必要時だけの再資格化、Source/意味の二重保持、実Provider照会数・p50/p95、必要Adapter数。#2018へ移し、独立Expectedは#1131/#1180/#2514の既存Ownerへ委譲する。

**状態**：本節は設計・局所合成Fixtureの接続であって、実TEI Runtime、実組織Provider、bitemporal Query/Host、OS/AI送信、業務Owner受理の実証ではない。PR #2496はDraftを維持し、mainへ統合しない。

<!-- tei-data-form-independent-requirement-closure-20261106 -->
<!-- tei-data-form-direct-vs-materialized-relation-20261106 -->
### 23.5 Relationの永続識別：直接参照と実体化Relationを区別する（#2389 §67〜69からの成熟差分）

**補正対象**：本節§23のr1〜r4、Graph→TreeによるRelation occurrence喪失は**独立IDのあるMembership型の人工世界**について有効である。しかし、実Native #2681 のすべてのRelation、特に `CaseRecord.handlingUnit` に独立edge occurrence IDがあるという意味ではない。Graphの描画線ID、JOIN行位置、表示ラベル、SQL Row Numberから永続的なRelation IDを作らない。#2389の組織設計§67〜69の最新補正に従い、実際のSourceと対象操作の区分を明示する。

| 性質 | 実体化Relation：`organization.Membership` 等 | 直接Relation：`organization.CaseRecord.handlingUnit` |
| --- | --- | --- |
| 実Source | `membershipId`, `personId`, `orgUnitId`, `allocation`, `validFrom`, `knownFrom`。Membershipに独立IDがある | `caseId`, `handlingUnitId`, `relation handlingUnit ... one`。別edge IDはない |
| Grain/Identity | Membership occurrence ID＋その関係が指すPerson/OrgUnit。複数Occurrenceはあり得る | **所有Case ID＋Relation Role＋現在の値**。読取・更新時にはSource Revision/対象行の期待版が別に必要 |
| 時間 | Valid/Known属性があるが、自動の二時点評価・History/Context切替実装ではない | 実Sourceにvalid/known属性はない。Membershipの時間条件を自動継承しない |
| Write Target | Membership等の実体化行と期待版を同定し、必要な操作契約で更新 | Case行の該当Role/値を変更。Graphの描画辺IDへDELETEを送る発想は誤り |
| 正式実Read | `persist org_store.membership`の宣言はあるが、Function/Provider Readまで未実証 | **CaseRecordのPersistence Mappingと実Provider Row Readは未確認**。 `org_read` がread指定でも業務Readの実行証拠にならない |
| CAS版 | 実Providerの行版/Ownerの更新意味を別照合 | **Native CaseRecordに`rowVersion`属性は未宣言**。合成Fixtureの7→8は実Providerに仮置きした値で、現TEI Model FieldでもCAS証拠でもない |

**同じ画面上の線であっても、編集対象決定は二系統**：Materialized Relation は元のMembership IDを、Direct Relationは所有Case＋Relation Role＋旧値＋別に資格化したExpected Versionを必要に応じて使う。「全Relationにedge IDを要求」も「同じOwner/Labelなら一意」も採用しない。複数担当Orgが必要なら、単一`handlingUnit`にD2/D3を同時代入せず、担当・協議先など別Roleか実体化Relationかを業務Ownerが受理して選ぶ。これは#2389 §69、#1126/#1125の意味Ownerが決定し、#2497はLossと更新対象の一般契約を所有する。

**Context/Read受入**：まずE2a（Case C1→D1と対照C2→D2、Provider Row/Field/Scopeの独立Expected）を既存#2586/HJの一Attemptへ接合する。次にE2bのMembership（人P1とMembership A1等）のvalid/known・多重出現・Graph/Tree投影を別のSubject/Required集合として試す。2系列の試験の成功を一つの「Org bitemporal E2E」へ加算しない。必要となるVersionは実Provider/Sourceへ明示Bindingができるまで `missing-required-version` として保持し、足りない値をCoreに自動追加しない。新Relation型Enum、二時点Engine、Edge Registry、Host/Owner資格は作らない。

**現在の実証の上限**：Native Model/Org App/`persist membership`のGitHub Source読取、#2389の設計照合のみ。CaseRecord/OrgUnit実Persistence/物理Provider Read、Materialized Relationの真正時点Query、実CAS/Host Grant/Owner意味受理はNOT RUN。既存§23のPython14件は人工Membership型の多重性を示す限定結果であり、この補正の実Go/DB実装テストではない。

## 24. 宣言の自己整合性と独立した必要条件の閉包を分離する（2026-11-06）

### 24.1 今回、実Go Sourceと他レーンのGo Characterizationから確認できた差分

#2619/#2620のEverything is Data／自己記述、#2860の実Parser負例、#2862の実Query Contract構造検査、#2889のRunPlan toy Step試験を照合すると、**宣言された条件への適合検査だけでは、その宣言が必要条件を網羅したことを保証しない**。本節は既存§22（6同一性）・§23（時点とRelation）に「要求元の独立性」を加えるもので、新たな全域SolverやRoot Requirement Engineを提案しない。

GitHub mainの実ソース internal/realization/contract_query_sql.go にある ContractQuerySQLRealization.ValidateContract は、Query Signatureに宣言されたInputsとInput Parameter Mapping、結果列Mapping、Provider、Target、Sourceの空非空と重複を確認する。SQL本文の実行、SQLで実参照されるパラメーター、Predicate/Sort/Scope、物理Read/業務Expectedの充足はこの関数の範囲外。Draft #2862のGo Characterization（3ケース）は、同一Query+SQL本文の複数Providerへの明示Mapping・欠損Mapping拒否、SQL Source変更後のRealization Identity維持を確認するだけで、実SQL Selector/実DB比較ではない。

mainの internal/execution/runplan.go は、unattended RunでPrecondition callbackが無ければ停止するが、callbackが空理由を返した場合、**そのcallbackがHost Grantや業務上必要な全Preconditionを覆ったこと**までは検証しない。同じコードの EffectPreconditions はafter-commit Observerの存在を確認するが、その時点でObserverを呼び実Effectを観測してはいない。Draft #2889の実Go Runner＋toy Steps（fixed-head CI 3/3 PASS）はこのギャップと、後段blockedでも先行Stepの記録が残ることを限定実証。実組織DB/Host/業務Effectではない。

さらにDraft #2860の実Go Parser Characterizationは、同一Attributeの Source に Role を二回書いた場合、現Parser Candidate が後のRole文字列のみを保持して先のRole occurrenceを失い得ると確認した。引用文字列中のRole風表現はRoleとして扱われない。原Source byte/occurrenceと Semantic Candidate が同じ情報を持つとは限らず、同一候補Roleから「Role宣言が一意だった」「Ownerに受理された」「可逆に編集できる」とは結論できない。正式なSource-aware衝突診断の実装とRole意味受理は別。

**まとめ**：データ様式の適合は、宣言済み項目を内部で突き合わせるだけでなく、当該Claimの独立したRequired集合、Source上の消え得るoccurrence、実消費入力、実Targetの観測へ、必要な範囲で照合する。宣言を正本とすることは「その宣言自身が正しいという証明を自ら発行する」ことではない。

### 24.2 既存Evidence票へ加える4種類の照合（永続Schemaではない）

| 種類 | 対象・供給元 | 誤った成立の例 | 必要な応答 |
| --- | --- | --- | --- |
| Required（外部から固定） | 当該Function/Claimの受理済みMeaning、独立Scenario/Expected、必須Source/Role/Read Scope、Consumer用途。#1126/#1131/#1180/#2514 | 定義のrequiredが空なので全条件PASS | 必須条件が独立に確定しないときは unknown/unqualified。無関係Claimには不要なHost Effectを要求しない |
| Declared（モデル/実装の申告） | SourceにあるRole/Relation/Query Input、Provider Binding、Function Mapping、RunPlanが構成したGuards | 同じRoleが重複していてもCandidateのRole文字列が一意だから安全 | Source occurrence/Role衝突、必要Field/Relation/Ruleの申告漏れを独立Expectedと照合 |
| Resolved and Consumed（実際の経路） | Loadで実消費したSource/Plugin/SQL/Schema/Policy、Query分岐、Function Mapping/Provider Route | Query ContractのValidate成功だけでSQL Predicateと実消費が正しい | Source Index・実解析Request/入力閉包・実選択Realizationに帰属。未読込Sourceや無音fallbackを未充足として明示 |
| Observed/Disclosed（結果の資格） | 実ProviderのRowset+Field/Read Cohort/Scope、Host Grant、Effect Receipt/独立Read、実Consumer受領 | toy RunPlan completed/0 rows/短いPageを業務完了・完全不存在へ昇格 | Claimごとに完了根拠を要求。歴史的観測と現在の開示・実行資格を分ける |

全4種類を全ケースで物理的に実行させるのではなく、**Claimに必要な部分だけを束縛**する。たとえば純粋な型付き構造変換はDB物理ReadもHost Commitも不要。一方、全件不存在のClaimはScope+Rowset Closureが必須で、業務更新の完了Claimには実Effectと独立Observerが必要。

### 24.3 一つのSubjectから発生する判別可能な反例

| Case | 入力と固定する独立要求 | 現行実装が実際に保証する範囲 | 未充足なら停止するClaim | Owner |
| --- | --- | --- | --- | --- |
| RC-01: 宣言にない必要Field | Accepted Functionは非表示のcredit_limitを要求、明示Query Contractはidだけ | Query ContractのDeclared Mapping検査が通っても外部Requiredは満たされない | 与信Validation/Action資格のみunknown。id表示は条件付き維持可 | #2506/#2624/#2514 |
| RC-02: Source上のRole二重記述 | Sourceに同AttributeのRoleを二回記述、片方は別の意味 | Parser Candidateの単一Role文字列は衝突/occurrence完全性を証明しない | Meaning受理、同一Source Patch、Role依存Buildの資格を保留 | #988/#2620/#2621/#2497 |
| RC-03: Query RealizationのSource改版 | 同Target/Provider/MappingでSQL Predicateだけ変わる | ValidateContract/Identity同一はSource byte/結果同値の証明ではない | 影響するFilter/Count/Page/Closure Claimを再資格化 | #2862/#2387/#2600/#2586 |
| RC-04: RunPlanのGuard申告不足 | 独立要求にCurrent Host Grantあり、callbackがPeriodだけ確認 | Runner completedはHost Grant/業務Effectが満たされた証拠でない | Write/Business完了のみ未資格。read-onlyな構造情報まで全失効しない | #1129/#1174/#2514 |
| RC-05: Observer存在と実効果 | EffectObserverが存在するがObserveEffect/独立post-image未読取 | Observerが渡された、toy Stepがresolved、Journalがcompletedは実DB Receiptでない | Effect完了Claimをnot-establishedに維持 | #2889/#2564/#1174 |
| RC-06: Resource/Selection負依存 | Source/Provider/SQL候補の追加、未読込必須SourceやA→B→A | 既選択Inputのbyte一致/Cache HitだけではFull Required Closureを示さない | 関係するRealization/現行Consumer Claimのみ再資格化 | #2823/#2600/#2622/#2516 |

これらは**既存の独立Oracle・Scenarioへ渡す受入対照**であり、RC-01〜06を本節独立で実装・実行したという意味ではない。#2860/#2889/#2862各Goテストを同じTEI一往復成功として足さない。重複していたらCase IDではなく対応する既存Issueの負例に付随する列として運用する。

### 24.4 次の最小一Attemptへの組込みと改善指標

最初の実製品接続は既存#2586と#1119のP0 read側を維持し、Subject/Claimと独立Expectedを固定→実TEI Source Load/Role/Query→実選択Realization/解析Input→Function ContextのD/V/Q/A必要項目→Provider Read/Result Slice→現在Consumer資格の順で照合する。**必要条件の宣言が0でも、他のSource/業務Expectedで必須項目が確認できるなら空PASSにしない**。全Requiredが独立に不要と確定している純粋な操作であればemptyの合法性を許す。独立要求とActualの対応が未観測なら、それを明示的な欠落として残す。

次に編集レーンは#2860の実Source→Parserの単一Role/二重Role/引用Literalを#2497/#2621のNo-op/Source span一意性/Preview read-onlyへ引き渡す。現Source Writer/Role Interpreter/Studio機能の有無を確認せず、Parser Testを編集可能の証拠としない。RunPlan/Effectは別のwrite側の一Attemptで現在Grant→Commit/Unknown→Receipt/独立Readに接続する。P0のSQL/Page実ReadへWriteやSource自己構築をHard dependencyとして持ち込まない。

#2018へは必須Source/Field/Relationの未観測・誤ったcomplete数、宣言不足を検知した負例数、静的解析/実Read/Consumerで追加したAdapter数、再資格化の過不足とp50/p95を返す。Candidate由来ExpectedでCandidate自身を検証している場合は独立Oracle PASSに数えない。新Role Resolver/Requirement Engine/Host/Proof Store/DSL用Core TypeKindは採用しない。

**資格（2026-11-06）**：本節は既存main実Goの静的ソース確認＋他Draftの限定Go Characterizationを照合した計画反映。実TEI Function→DB/Consumer、Source-aware正式Role衝突診断、Role Meaning Owner受理、RunPlan実DB Effect、RC-01〜06の統合受入はNOT RUN。PR #2496はDraft維持、main未統合。

<!-- tei-data-form-collision-and-p0r1-20261106 -->
### 24.5 重複・衝突の判断は「データ様式 × 同一性キー × Use-site」の契約で決める（2026-11-06追補）

#2495/#2497で検討する多重Facetと構造保存を、直近の実Go/Scenario/DB移行負例へ接続する。単一の「重複値禁止」規則をCoreに導入せず、**どの構成子の、何がIdentityで、何を一意と要求しているか、重複が操作へどう影響するか**を明示する。

| 入力構造 | 重複の意味 | 失われ得る事実・今回の実例 | 安全側の契約とOwner |
| --- | --- | --- | --- |
| Sequence/SQL Bag | 値が同じ別Occurrenceは合法。PositionまたはFact/Row identityを区別 | JOIN増幅、同額の異FactをDISTINCTで消す | 値の重複排除を自動同一化しない。#1706/#2387/#2593、Corpus PC-100/111/169/173 |
| Graphの多重有向辺/組織Relation | 同じsource/target/labelでも別Relation occurrenceは合法 | 人物2件不変でもMembership occurrence 2→3。Graph→Treeで消える | 辺ID、方向、適用時点、Multiplicityを対象Claimへ渡す。§23/#1126/#2506 |
| JSON Object member | 同一Objectに同じmember nameの複数出現は、少なくともScenario/Oracle入口では曖昧性として拒否する | 一般のdecoderで後値が前値を隠しExpectedが入力順依存。実Issue #2891はScenario系3 Fixture入口に既存RejectDuplicateJSONMembersを追加する修正レーン | Object memberの重複拒否は**Array [1,1]の合法な値重複を禁止しない**。#2891/#1180/#2514 |
| 同一AttributeのDSL Role記述 | 受理前のSyntax上の重複。複数Roleの意味が必要ならUse-siteを含む明示Bindingの設計が別に必要 | 現main Parserは同じAttributeでRoleを繰り返すと最後の文字列へ上書きし、前のSource occurrenceがSemantic Candidateから見えない（Draft #2860） | Candidateが単一Roleでも一意に受理したと推定しない。Source-aware Diagnostic、未解決なら編集/受理保留。#988/#2620/#2621 |
| DB Release StateSnapshot/Provides | 一つの評価対象DB ID／Contract IDに、互いに矛盾する複数版があるのは観測入力の衝突 | 現main internal/database/release/compatibility.go は両者をMapへ無検査代入し、後値が先値を隠す。#2887 MG-29の現状特性テストと専任修正 #2897 | 不確かな入力は勝手に最新/後勝ちを選ばずUnknown/入力拒否へ。修正と現挙動テストのExpected変更を同一統合単位にする。#2897/#2555 |
| 外部Read/History | 同一識別子の重複と、履歴のunknown/未取得は別問題 | #2892 CompareHistoryのunknown誤RepoApplied、Reader未取得を空集合へ潰す | 実Readの完了/History Snapshotの資格を要求し、空/不存在/未観測を区別。#2892/#2627/#2586 |

**反例の一般化限界**：同一の文字列が2回出るだけで違反となるわけではない。一方、Mapへ縮約した後の値が同じでも、重複していたSource occurrence、入力順、Query hidden deps、Snapshot provenanceを復元できない。これは二つの必要条件である：
- *Representation obligation*: Source/Raw Inputの複数出現を、対象構造の意味契約が許す限り区別し、禁止する場合は消失前に診断する。
- *Consumption obligation*: 正規化された構造が合法でも、Consumer/Query/Rule/Actionが使うKey、Use-site、Relation Grain、Scope、現在資格が十分か独立に検査する。

特に**「重複を拒否する」と「一意の対象が存在する」は別**。JSON Scenarioの同一オブジェクト重複拒否やRelease同一DB衝突拒否が成立しても、必要なObject member/DB Client母集団/Relation Scopeが全部読み込まれたとは限らない（#2823の未知Source、#2892の履歴未取得等）。新しいMultiplicity Enumや万能Duplicate Validatorではなく、既存Parser/Scenario/DB Release/ProviderのOwnerへそれぞれ差分を戻す。

現状資格：Releaseコードの二つのMap代入とParserのRole overwriteは現main Source静的照合。#2891は修正PRの限定試験であり、この節では新しいCIや正式Schema適合を測定していない。#2897は安全修正の計画段階。Graph/Arrow/Protobuf/外部Hostを横断した実ConformanceはNOT RUN。

### 24.6 P0-R1の優先順位を明確化：実Read成立後に二SQL切替とContinuationを検証

前節までの「P0＝一Provider二SQL」の書き方は、実際のFunction→Providerが未接続の状態で二Realization切替の製品実証を先行できるように読める。これは他チャットの参照モデル移植[#2783 §11〜12](https://github.com/hat47x/tei-platform/pull/2783)と、#2586の最初の同一Attempt要件を照合して次の**順序へ補正**する。§22 HJ-SV-01〜05を撤回するのではなく、実施条件を分ける。

| 優先 | 一つの実行Attemptで固定する対象 | 必須観測と未充足時の結果 | Owner |
| --- | --- | --- | --- |
| **P0-R0/準備** | 公開合成の1 Tenant・1 Order・2 Line＋他Tenant同番号、独立Expected、固定checkout/Source/Contract、実Target/Provider Alias | `tei dbcheck`が利用可能でも、接続＋TEMP TABLE作成/削除は業務Readではない。実Function Reader/Providerが未実装なら入口GapとNOT RUNを記録。#1131/#2514/#2894 |
| **P0-R1/最初の製品実Read** | **1 Function・1 Query・1 Provider・1 Read Cohort**、明示Relation path/Required Field/Grain/Tenant Scope、同じSource Revision | Required→Declared→実Loaded→SQL/Provider実Consumed→実Rowset/Field coverage→Resultの独立照合。SQL本文/Role/Ruleが不明ならunknown。Queryを選べた/表が見えた/0件だっただけでRead完了としない。#2506/#2622/#2586/#2783 |
| **P0-R1v/選択差分** | R1と同一Contractでcommon/専用という二つの明示かつ契約適合Realization、同じFixture/Scope | 実選択Source・解析Input・Read Witness、SQL変更/候補追加の負依存を確認。commonの自動fallbackは採用しない。Query ID一致では同一結果/Meaningを認定しない。#2611/#2622/#2387/#2862 |
| **P0-R1p/継続差分** | R1vの選択結果と実Provider Page1/2を束縛、必要なら途中で候補/RealizationやSnapshotを変更 | 同一Cohort/Scope/Sort/Params/Relation/Realizationが不変と実証できない場合は旧Continuationを停止。短いPage/visible0をcompleteにしない。#2387 QX-26/#2586 HJ-SV |
| **P1/用途・書込み別資格** | R1成果をGraph/Table/SHOなどへProjection、または別のOwner Write試行へ | current Grant/実Consumer Receiptと歴史的Meaningを分離。Save前拒否やCommit/Effect/独立再読取はwrite側別Attempt。#2516/#2497/#1129/#1174 |

P0-R1の初回負例は[#2783](https://github.com/hat47x/tei-platform/pull/2783)の N-R1a/b/cを再利用し、(a)他Tenant同業務番号を誤包含しない、(b)2 LineのJOIN fanoutや重複を業務件数へ取り違えない、(c)1 Pageだけ・Field missing/NULL/SQL候補改版を正常な完全取得にしない。Source Load後のbyte改訂、Roleの重複上書き、Raw JSONの重複Object memberは**必要なSource/Requiredを誤確定させる入口の負例**として同じ独立Expectedへ引き渡す。ただし全ケースをR1のHard blockerに増やさず、確認した要件不足を明示した局所unknown/unsupportedとして記録する。

P0-R1で要求する単一Evidence Packetは既存#1163/#2586を使用し、Source/Required/Declared/Resolved/Parsed/Actual Consumed、Semantic/Target/Query/Function Context、SQL/Snapshot/Rowset/Field/Result Grain、独立Expected/negative controls、未充足条件/Claim ceiling、Consumer現在資格を**同一Attemptの観測**として束縛。別RunのGo Parser/SQLite/RunPlan/DBCheckの成功を寄せ集めてE2E PASSにしない。独立Expectedそのものの逆転とFixtureの誤りは#2514、実Provider以降の資格は#2586、Owner/HostとEffectは#1174が担当する。

**順序を変更するが、#1119既存G0〜G4、#2586同一Function実証の正本、#1131/#1180 Scenario正本は変えない**。SQL→Providerが実際に存在しない場合は `NOT RUN: producer-or-physical-read-unavailable` とし、実行環境の設定でできる`tei dbcheck`の成功を代わりに使わない。§22のSQLite 11件と§23の時間付きRelation 14件は、**それぞれ過去に報告された独立・隔離合成Fixtureの限定結果**である。両者は別の入力・実装対象であり、本追補で同一固定HEAD・同一実行環境の25件を再実行したという証拠、または原Source SHA-256を実際に再照合した記録はこの変更に付属していない。したがって**「2026-11-06に再実行25/25 PASS」を本追補の実施事実と認定しない**。§22/§23記載のdigestは当時のローカル報告値であり、現在の実行ファイル/CIの検証値ではない。SQLite 11件・時間付きRelation 14件の各歴史的結果をR1/R1v/R1pの実製品資格に加算しない。追加の新Go/実PG/Consumer/Owner試験はNOT RUN。
