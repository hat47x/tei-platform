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
