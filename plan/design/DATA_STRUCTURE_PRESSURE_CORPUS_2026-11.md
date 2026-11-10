# データ構造Pressure Corpus（#2501、調査・比較用）

- **作成**：2026-11-01。**状態：外部仕様と既存TEI Issueの静的比較、未採用・未実装・全Corpusの実試験NOT RUN。**
- **Owner**：[#2501](https://github.com/hat47x/tei-platform/issues/2501)。横断分類は[#2495](https://github.com/hat47x/tei-platform/issues/2495)、操作相対の情報保存は[#2498](https://github.com/hat47x/tei-platform/issues/2498)、開発要件の拡張運用は[#2493](https://github.com/hat47x/tei-platform/issues/2493)。
- **位置付け**：異質な構造から概念的な圧力を集めるための**探索台帳**。新しいCanonical/型Registry/Feature enum/Scenario SSOTではない。全個別製品の仕様をTEIが実装したという意味でもない。
- **読み方**：各行の「確認すべき性質」は*試験で見たい条件*、「誤認・損失例」は*反証すべき仮説*。該当する既存Ownerへ戻して具体的Fixtureを設計する。事実が支持された実装と、未実施の提案を区別する。

## このCorpusで比較する七役割

対象の同一性／構成文法／性質・法則／解釈文脈／利用箇所の要求／主張と根拠／実現と表示。各Pressureの意味に関連するM1〜M10（#2495）は重複して構わないが、**一つの性質を二つの独立な意味正本に保存しない**。

以下の192ケースは**初期の異質性を測るための開始点であり、網羅の完了や上限を意味しない**。新規の構造・数学的性質・分野固有の解釈は追加し続ける。関心事が重複する場合も、操作・利用文脈・失われる情報が異なれば別の反例として保持する。

## A. 値と同一性（PC-01〜07）

| ID | Pressure / 資料上の構造 | 構成・表現 | 分けて問う性質 | 誤認・損失の負例 | 既存Owner |
| --- | --- | --- | --- | --- | --- |
| PC-01 | 取引先コード／得意先コード | 同じstring表現・別TypeRef | 発行体系・意味の同一性 | 正規表現一致を業務上同一とみなす | #1806/#1120 |
| PC-02 | 複合ビジネスキー | record/tuple | 構成要素順序・適用Scope・同一性 | DB PK一致を業務ID同一とみなす | #1126 |
| PC-03 | Entity Reference／Value Object | IDを含む値 | 参照先Identityと値同値 | 同じ値の二出現を同じEntityとみなす | #1120/#1126 |
| PC-04 | Event occurrence／状態Snapshot | event列とrecord | 出来事の同一性・版・時点 | 同じ状態の二Eventを統合 | #1131/#2386 |
| PC-05 | Alias型の廃止 | TypeRef/Underlying | Alias利用先と旧版Consumer | 属性の付替え成功を型削除可とする | #1806 |
| PC-06 | 値域の論理矛盾 | 制約の交差 | 充足可能性・型Scope | レコード0件を制約充足と取り違える | #1806/#1935 |
| PC-07 | nullableな利用箇所 | optional field | 型全体の値域とfield必須性 | optionalなAPIから必須なDB列へ黙って写す | #2056/#1129 |

## B. 集合・順序・取得完了（PC-08〜16）

| ID | Pressure / 資料上の構造 | 構成・表現 | 分けて問う性質 | 誤認・損失の負例 | 既存Owner |
| --- | --- | --- | --- | --- | --- |
| PC-08 | Sequence→Bag | 可変長列 | 要素同一性・重複数・順序 | 順序を保持したと誤表示 | #1706 |
| PC-09 | Bag→Set | 多重集合 | 要素の出現数・集合identity | 重複を業務上不要と仮定 | #1706/#1126 |
| PC-10 | Map／Ordered Map | key→value | key一意性・列挙順・index実装 | hash storageで順序保証とみなす | #1127 |
| PC-11 | MultiMap | key→collection | 同一keyの複数値・順序 | 一対一Mapへ上書きして値喪失 | #1127 |
| PC-12 | 稼働Stream／有限ページ | 時間付きイベント列 | 完全集合性・取得範囲・継続点 | 空ページを全件0件と扱う | #1706/#1131 |
| PC-13 | 窓集計 | イベント窓 | event time／arrival time・watermark | 後着Eventを集計済み確定値へ無反映 | #1706/#1640 |
| PC-14 | Array Slice／View | 配列参照範囲 | view所有・寿命・mutationの可視性 | copyとaliasを同一視 | #1127/#2077 |
| PC-15 | Lazy Iterable | 遅延評価 | 有限性・順序・副作用・再実行 | 全列挙可能と仮定してlengthを確定 | #1127/#1129 |
| PC-16 | SQL JOINのfanout | Relation間の結合結果 | 元のOccurrence/多重度・結合由来増幅 | 結果行数を明細instance数へ直結 | #1706/#2387 |

## C. Relation・Graph・組織（PC-17〜24）

| ID | Pressure / 資料上の構造 | 構成・表現 | 分けて問う性質 | 誤認・損失の負例 | 既存Owner |
| --- | --- | --- | --- | --- | --- |
| PC-17 | 候補キーと外部キー | EntityとRelation | 論理ID・結合条件・DB物理キー | SQL列名一致を参照関係の受理とみなす | #1126 |
| PC-18 | Tree／DAG | edge集合＋制約 | 単一親・無閉路・到達可能性 | 階層UI表示をTreeの形式的証明へ昇格 | #1126 |
| PC-19 | 多重親の組織所属 | 時点付き関係集合 | 兼務・兼任・権限・有効時点 | 単一親部門ツリーへ潰す | #2386 |
| PC-20 | HypergraphとN項Relation | 多項tuple | edge arity・参加者role | 2項辺だけで一意に再構成したと仮定 | #1126 |
| PC-21 | BOM構成／数量 | 多段Relation＋Quantity | 部品再利用・数量・改版 | Graphの経路数を発注必要数量と混同 | #1126/#1705 |
| PC-22 | Property Graph | 頂点・辺・属性 | 属性の所有先・型・版 | Graph DB表現からCanonical Relationを推定 | #1126/#2387 |
| PC-23 | 時間変化するGraph | event stream＋edge状態 | 有効時点・トランザクション時点 | 現在の接続から過去の所属を推定 | #2386/#1131 |
| PC-24 | 権限付きGraph traversal | Relation query | Caller/Scope/Read Authority | 到達できるEdgeを閲覧可能とみなす | #1127/#1174 |

## D. 量・時間・数値（PC-25〜32）

| ID | Pressure / 資料上の構造 | 構成・表現 | 分けて問う性質 | 誤認・損失の負例 | 既存Owner |
| --- | --- | --- | --- | --- | --- |
| PC-25 | 絶対温度／温度差 | 数値＋単位＋量種別 | offsetと差分の演算 | point＋pointを正当な温度演算とみなす | #1705 |
| PC-26 | 外貨の合算 | Decimal金額 | 通貨・FX Source・適用時点 | USD＋JPYを同単位の金額として加算 | #1705 |
| PC-27 | 個数×単価 | Quantity×Rate | 単位の約分・税区分・端数時点 | 浮動小数点CASTで金額を丸める | #1705/#2077 |
| PC-28 | Decimalの桁・scale | Decimal型・物理NUMERIC | 数値意味・丸め方式・精度 | 物理値が入ったことを業務計算同等性に拡張 | #1705/#1806 |
| PC-29 | 時点／期間／暦月 | temporal value | instant・duration・calendar period | 1か月を固定30日へ置換 | #1705/#2386 |
| PC-30 | Bitemporal history | valid-time／transaction-time | 同時点・記録時点・訂正履歴 | 更新時刻だけで業務有効時点を推定 | #2386/#1131 |
| PC-31 | 角度と無次元比 | 数値＋dimension | rad/degree・比率・演算 | 同じdouble型なら暗黙にsinへ渡せる | #1705 |
| PC-32 | 割合／確率／スコア | DecimalまたはFloat | 0..1の値域・正規化・母集団 | スコア合計を確率質量1とみなす | #1705/#2495 |

## E. 数学・科学配列・空間（PC-33〜40）

| ID | Pressure / 資料上の構造 | 構成・表現 | 分けて問う性質 | 誤認・損失の負例 | 既存Owner |
| --- | --- | --- | --- | --- | --- |
| PC-33 | 数学的Vector／Float配列 | tuple/sequence | scalar field・加法/スカラー倍公理 | Float[3]なら線形空間を持つと推論 | #2495 |
| PC-34 | Matrix／Tensor | 多次元配列 | 次元数・基底・軸の意味・演算則 | 配列shape一致だけで行列積を許す | #2495/#1705 |
| PC-35 | xarray DataArray | 値＋named dims＋coords | 座標alignment・軸名・ラベル同一性 | ndarray化でalignmentが保存とする | #2495/#2077 |
| PC-36 | Sparse Tensor | 添字＋値の集合 | 暗黙ゼロ・missing・index重複 | 非格納の座標を未取得と同一視 | #2495/#2056 |
| PC-37 | 確率分布／Map | Outcome→weight | 事象・非負・正規化・測度 | 重みのmapを正規化済み確率とみなす | #2495 |
| PC-38 | 数学的位相空間 | carrier＋開集合族 | 空/全体・任意和・有限交わり | Graph adjacencyで位相空間の公理が自動成立とする | #2495 |
| PC-39 | OGC Geometryと座標系 | Point/Curve/Surface | CRS・datum・axis order | tuple(lat,lon)を距離計算可能へ無条件変換 | #2495/#1126 |
| PC-40 | 移動軌跡／地理イベント | 時間付き座標sequence | CRS・時間順・精度・移動主体 | 折れ線を実経路・全履歴と断定 | #2495/#1706 |

## F. 外部Schema・Serialization（PC-41〜48）

| ID | Pressure / 資料上の構造 | 構成・表現 | 分けて問う性質 | 誤認・損失の負例 | 既存Owner |
| --- | --- | --- | --- | --- | --- |
| PC-41 | Arrow List／Struct | offset buffer＋child array | 論理構成と物理buffer | 表示上の入れ子だけで同一形と扱う | #2077 |
| PC-42 | Arrow Map／List<Struct> | 物理上近いnested layout | Key/Valueの意味・Map条件 | 物理同型から業務Map同等性へ推論 | #2077/#2498 |
| PC-43 | Arrow Dense／Sparse Union | tagged unionの別layout | tag・nullness・branch occurrence | 同じunion名でbyte layoutまで一致とする | #2077 |
| PC-44 | Arrow Dictionary encoding | index＋辞書 | value identityと辞書rev | indexを業務列挙IDとして固定 | #2077/#1120 |
| PC-45 | Proto3 optional／implicit scalar | field値＋presence bitの有無 | unset・明示defaultの区別 | 値0と未設定が区別できると断定 | #2056/#2077 |
| PC-46 | Proto3 repeated／map | sequence／key-value | 繰返し欄のpresence非追跡・map順序 | 空と未設定を往復できると宣言 | #2056/#2498 |
| PC-47 | Proto3 oneof | tag付きsum | 選択branch・未設定状態 | 二branch同時値を維持できるとする | #2077 |
| PC-48 | JSON Schema allOf/anyOf/oneOf | schema検証の合成 | assertion vs annotation・branch | 一つの検証PASSを人の意味受理に昇格 | #2495/#1935 |

## G. 文書・UI・Operation・AI（PC-49〜56）

| ID | Pressure / 資料上の構造 | 構成・表現 | 分けて問う性質 | 誤認・損失の負例 | 既存Owner |
| --- | --- | --- | --- | --- | --- |
| PC-49 | UI TableとChart | 同じ列を異なる投影 | Data Context・grain・集計の意味 | 描画できたことを数値集計の妥当性とする | #1127/#997 |
| PC-50 | Gantt／Time-series図 | Temporal Event→投影 | 時点・期間・依存/階層 | UIの位置やピクセルを業務時刻と断定 | #1127/#2391 |
| PC-51 | KaTeX表示式／DSL式 | 文書Textと演算式 | 参照した式の意味・証明・版 | 数式が美しく表示されたので計算成立とする | #1803/#988 |
| PC-52 | SHO引用と出典画像 | Text＋Asset dependency | 固定版・引用本体・表示依存 | 本文digest一致から表示内容一致を推論 | #1803 |
| PC-53 | 外部UI編集Intent | UI instance＋Actor | 選択粒度・対象版・現在権限 | ボタンクリックをDB commitの証明にする | #2391/#1174 |
| PC-54 | AI CUIのRead Result | 構造化説明とevidence | Source Scope・鮮度・unknown・Disclosure | 説明成功を未取得データの存在確認に使う | #2354/#1178 |
| PC-55 | 外部Hostでの意味型利用 | Java/.NET/Wasm成果物 | 入出力意味・NULL・Failure・Authority | 共通JSON shapeで実行等価と断定 | #2077/#2390 |
| PC-56 | モデル改版と納品固定版 | R1 Delivery／R2 Working | Source・Contract・生成物・SHO同版 | 新TypeRefを旧納品物へ自動適用 | #1803/#1124 |

## H. 数学的構造の複数解釈・高階／無限・分散（PC-57〜72、2026-11-02追加候補）

同じCarrierが同種の公理体系を異なる演算で満たす場合や、有限・無限、時点、論理的な情報の違いがあり、一般的な「データ型」の列挙では発見しにくい反例を追加する。詳細なLaw束縛の評価Ownerは[#2544](https://github.com/hat47x/tei-platform/issues/2544)、情報保存は#2497、実Oracleは#2514に置く。**この16行は設計上のPressureであり、新しいSource正本・実装済みCore型・試験PASSではない。**

| ID | Pressure / 資料上の構造 | 構成・表現 | 分けて問う性質 | 誤認・損失の負例 | 既存Owner |
| --- | --- | --- | --- | --- | --- |
| PC-57 | 同じ自然数上の二つのMonoid | Carrier `ℕ`、加法+0、乗法×1 | 演算束縛・単位元・結合律 | `Monoid`分類だけで演算を選ぶ | #2544/#988 |
| PC-58 | 同じ集合上の二つのOrder | 通常の大小関係、整除関係 | 比較演算の同一性・全順序/半順序 | `Ordered`のみで比較結果を断定 | #2544/#2495 |
| PC-59 | 同一Carrierの複数Topology | 離散／密着位相と開集合族 | carrierと位相族、連続性 | Graphの隣接・表示上の近さを位相公理と同一視 | #2544/#2495 |
| PC-60 | 演算名のDiamond競合 | 2つのFacetが`combine`を要求 | 演算ID・公理・Use-siteの束縛 | 親の宣言順で一つを暗黙採用 | #2544/#951 |
| PC-61 | 異なる記述の同じ制約 | 数値の範囲／共通Predicate、Constraint AND | 表現同値・充足可能性・Source | 式形が同じなので業務意味同一と断定 | #2544/#1806 |
| PC-62 | 充足不能と空のDataset | `x>0∧x<0`／0件のDB | 空構造と矛盾する型宣言 | 実データ0件を充足不能の証明とする | #2544/#1806 |
| PC-63 | 反例の有限Scope | Alloy等で一定数の原子を探索 | Bound、存在例、反例、未判定 | 有限探索無反例を無条件の数学証明とする | #2544/#2514 |
| PC-64 | 全称条件と存在条件 | `∀x:P(x)`と`∃x:P(x)` | 空集合での真理・対象母集団 | emptyで全称真から必要レコード存在を断定 | #2544/#1063 |
| PC-65 | 依存次元のVector/Tensor | `Vector(n)`、rank/dims/coords | index上の型条件・次元と形 | 同じbytes/lengthだけで座標系・物理量を同一視 | #2495/#2077 |
| PC-66 | 再帰Treeと循環参照 | 再帰的データ定義とGraphの共有参照 | 木の有限展開、Object Identity、循環 | 再帰型を持つのでinstanceが必ずTreeと断定 | #2495/#1126 |
| PC-67 | 無限・遅延Stream | 潜在的無限系列、有限prefix | 生成規則と現在の取得Scope | 3件取得したので全列挙済みと断定 | #2495/#1706 |
| PC-68 | 因果順序と時計時刻 | 分散Event、happens-before、Lamport順序 | 部分順序、同時性、clockの意味 | UI表示用のtotal sortを因果順序として受理 | #2497/#2386 |
| PC-69 | 可変参照と値のコピー | alias、immutable snapshot、shared state | 所有/可変性・EffectとValue同等性 | 同じ値が読めるので書込み互換と断定 | #2500/#1174 |
| PC-70 | 疎構造のZero/Missing | Sparse Matrix、未格納Entry | 既定0／欠測／非存在の文脈 | 格納されていない座標を測定0と解釈 | #2497/#2056 |
| PC-71 | 確率測度とスコアベクトル | Outcomes→Weights、正規化・事象代数 | 数値配列／確率公理／解釈 | `sum=1`だけから実験・観測の妥当性を主張 | #2544/#2514 |
| PC-72 | 純粋関数と効果を持つAction | 同じ入力・出力型で異なるEffect | 数学的対応と実行Authority・Receipt | `A→B`が一致するだけで副作用や再試行安全性が等価と断定 | #988/#1174 |

対応する数理上の参考：[Lean Type Classes](https://lean-lang.org/doc/reference/latest/Type-Classes/)（演算と公理）、[mathlib hierarchy design](https://leanprover-community.github.io/mathlib4_docs/Mathlib/Algebra/HierarchyDesign)（代数・順序・位相の交差による構造増殖）、[CUE Specification](https://cue.dev/docs/reference/spec/)（Constraintの結合とbottom）、[Alloy](https://alloytools.org/spec.html)（有限Scopeの構造例と反例）。いずれもTEIの標準Runtime・Proof Engineの選定ではない。

## I. 正規形・同じ意味の複数Projection（PC-73〜84、2026-11-02追加候補）

この節は「型や構造を一つの平坦な分類に押し込むと意味が失われる」反例と、実Consumerの投影例を比較する。特にTEI mainの[record projection `many`](https://github.com/hat47x/tei-platform/blob/main/tools/reference-generator-selection/generator.go)と、TAI mainの[`Island.cardIds`](https://github.com/hat47x/tai-pleroma/blob/main/integrations/pilot-sui-document/candidates/island-record/contract.json)／[`Affiliation`](https://github.com/hat47x/tai-pleroma/blob/main/integrations/pilot-sui-document/candidates/affiliation-record/contract.json)は、**同じ所属を指すかどうかをOwnerに確認する必要がある**現実の検討素材となる。新しい共通型やE2E合格の根拠とはしない。

| ID | Pressure / 資料上の構造 | 構成・表現 | 分けて問う性質 | 誤認・損失の負例 | 既存Owner |
| --- | --- | --- | --- | --- | --- |
| PC-73 | SUI Island.cardIdsとAffiliation | 多値Identifier投影／所属Relation Record | 同一意味なのか・双方向整合・Snapshot | 各Cardが存在すれば二つの所属表現が一致すると断定 | #2497/#2501 |
| PC-74 | `many:true`と集合・列の性質 | `[]string` JSON配列 | Cardinality/順序/重複/完全性・参照先 | 配列型があるだけでSet・重複禁止・全件取得と推論 | #2495/#2497 |
| PC-75 | SUI所属の時点/可視範囲 | Island/Card/Affiliationの異なるRead | 対象版・Scope・権限・Partial vs Complete | 部分的に欠ける所属を業務上の不存在と推定 | #2497/#2516 |
| PC-76 | 異種Endpointを持つEdge | `fromKind/toKind + fromId/toId` | Type-tagged Reference・Graph incidence | IDが一致するのでCard/Islandの区別を消す | #2495/#1126 |
| PC-77 | SUIの表示用関係と編集Intent | Card列挙とLink/Unlink Action | 投影の逆写像・現在権限・競合 | 表示配列の変更をAffiliation DB更新完了とみなす | #2497/#2500 |
| PC-78 | SUIキャンバスの座標と所属 | Layout coordinates＋Graph/Relation | 空間的近さと意味上のRelation | Cardを近くへ動かしたので所属関係を自動変更 | #2497/#997 |
| PC-79 | `Map<K,Set<V>>` vs `Set<(K,V)>` | 対応Mapとペア集合 | Valueが空のKeyの存在と保存 | 値が0件のKeyまで完全な同型と断定 | #2544/#2497 |
| PC-80 | `List<A\|B>` vs `List<A>\|List<B>` | CollectionとTagged Unionの構成順 | 異種混在可能性と分岐制約 | 構造子を分配して両者を等価に正規化 | #2544/#2495 |
| PC-81 | `Tuple(A,B)`とNamed Record | 位置・ラベルの異なる積構造 | 要素順・属性名・Source identity | 同じ二つのPrimitiveだから無条件の同一型 | #2544/#1806 |
| PC-82 | エネルギーとトルク | 同じ物理次元・単位 `N·m` | Quantity Kind・意味上の演算 | UnitとDimension一致で同じ量として扱う | #1705/#2544 |
| PC-83 | 同じ値を異なるSourceへ投影 | 同じJSON・別Source/Version | 意味同一性・Provenance・現在受理 | 見た目一致/字句一致でSourceや意味の版を同一視 | #2497/#2516 |
| PC-84 | SUI記録をJSON→CUI/文書へ | `present/missing/unresolved`と説明 | 入力形状・未解決・Consumer側の完全性 | JSON出力成功・全Field検査をDomain全件保証に昇格 | #2498/#2516 |

**反例の具体化**：Cards `c1,c2`、Island `i1`が存在し、`i1.cardIds=["c1"]`と`Affiliation(id="a1",cardId="c2",islandId="i1")`が共存する合成入力では、各参照先の存在を確認するだけでは両所属表現の一致は判定できない。TAIの[`document_assembler.py`](https://github.com/hat47x/tai-pleroma/blob/main/integrations/pilot-sui-document/assembler/document_assembler.py)には各参照先の存在検査がある一方、両者が同一の所属意味かどうかを判定する契約は別の検討を要する。**入力の実行結果を今回取得したわけではない。**

参照：[構造保存写像・Projection設計 §10](STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)、[多重FacetのLawと比較正規形 §8](FACET_LAW_BINDING_AND_COHERENCE_2026-11.md)、[#2500 View変更からTyped Intent](https://github.com/hat47x/tei-platform/issues/2500)。新しい配列生成器の実装は#2576、SUI個別Meaning/AssemblerはTAI Repo側のOwnerとする。

## J. 実Record Projectionの値忠実度・構造の直積（PC-85〜96、2026-11-02追加候補）

TEIの[PR #2576](https://github.com/hat47x/tei-platform/pull/2576)・[#2578](https://github.com/hat47x/tei-platform/pull/2578)・[#2580](https://github.com/hat47x/tei-platform/pull/2580)・[#2581](https://github.com/hat47x/tei-platform/pull/2581)による実装拡張を、**Primitive/Collection/Presence/Realization/Sourceとの交差**として評価。#2495の七役割、[#2497の詳細設計 §11](STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)、[#2585](https://github.com/hat47x/tei-platform/issues/2585)のPFC-01〜12と接続。**この表自体は実際のraw入力→生成main.goのテスト結果ではない。**

| ID | 構成・表現 | Pressure／確認ケース | 保持したい意味・観測 | 誤認・損失の負例 | 対象Owner |
| --- | --- | --- | --- | --- | --- |
| PC-85 | required / optional Field と値の存在状態 | required Fieldのmissing/null/empty | 存在状態と有効な空値 | 必須nullを空文字列や未指定と同じ扱いにする | #2585/#2056 |
| PC-86 | optionalな入力Fieldと送信Payload | optional Fieldのmissing/null | Consumerが区別を要求するか | 出力省略から元のmissing/nullを復元可能とする | #2585/#2497 |
| PC-87 | boolean Field と存在ビット | optional boolean false/absent | falseという値と存在ビット | falseをdefault扱いで消去・review済みと誤推論 | #2585/#2498 |
| PC-88 | JSON数値字句からtyped Integerへのdecode | JSON integerの境界 | `9007199254740993`の正確な整数値 | `float64`の約数に変換しても同値と宣言 | #2585/#1705 |
| PC-89 | 十進値の字句／Decimal表現・Scale | JSON decimalとscale | `0.10000000000000001`・`1.005`の値・字句・精度 | 同じ数値見た目なら十進契約・丸めも同じとする | #2585/#1705 |
| PC-90 | Collection<Numeric/Boolean>の要素型 | `many` × numeric/boolean | 構文上の宣言とruntimeの要素型 | `many`があるため全Primitive配列が動作すると推測 | #2585/#2495 |
| PC-91 | Input Bindingの型付き参照グラフ | Field.Input→Inputs→Semantic TypeRef | 型・参照の結合とNamespace | Field定義のPrimitiveが合っても誤Input参照を許容 | #2585/#1806 |
| PC-92 | JSON入力ストリームと構文終端 | raw JSON duplicate/trailing values | 単一入力として受理したbyte範囲 | 部分Decodeの成功を入力全体の検査成功にする | #2585/#2514 |
| PC-93 | 多値FieldとRelationの別表現 | 文字列配列と業務Relation | 重複・順序・Key・全件Scope | `[]`と`many`だけでBusiness Setの全件と断定 | #2497/#2585 |
| PC-94 | 2D UI配置座標とDecimal金額 | SUI Cardのx/y値と業務Decimal | 表示用座標の精度と金額の精度の異なる用途 | 全てのDecimalへ同じ精度劣化を許容 | #2498/#1705 |
| PC-95 | 複数Sourceを持つSUI Document Projection | 既存SUI原本とTEI部分Projection | `critiqueHistory/transform/time`のOwnerと復元経路 | read-only artifactを完全なDocumentV1 Sourceと誤認 | #2497/#2516 |
| PC-96 | 表示セルとTyped Changeの候補 | Read-only ViewからTyped Change | optional省略/false/0/複数Sourceの変更Intent | 表示値の編集でSourceのclear/Link/Unlinkを暗黙実施 | #2500/#2497 |

**比較の見通し**：`std.decimal`のJSON出力は表示座標では許容できる場合がある一方、金額や厳密な識別番号では基底の数値保存契約へ違反し得る。値の同一性・JSON字句の同一性・意味上の精度・業務的受理は別の条件。optionalの省略は**意図したlossy表示**として成立し得るが、Patch/writebackが同じ損失を許すかは再評価する。

外部の一次資料：[Go encoding/json](https://pkg.go.dev/encoding/json)（interfaceへの数値decode）と[Protobuf Field Presence](https://protobuf.dev/programming-guides/field_presence/)（明示・暗黙の存在区別）。TEI/TAIのコード証拠は各PRの実際の実装・対象SHAを確認する。契約が正式に型保証を追加したわけではない。

## K. 時間・増分観測・Claim撤回（PC-97〜108、2026-11-02追加候補）

[構造保存写像設計 §12](STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)のTP-01〜12と、[#2586](https://github.com/hat47x/tei-platform/issues/2586)（Query→Context→Claim→Patternの縦断）、#2506（Result Slice）、#2505（版間Source）、#1708（Freshness）を接続する。これらは**試験計画**であり、実Provider・GUI・独立Oracleの実測ではない。

| ID | 構成・表現 | Pressure／確認ケース | 必要な区別・保持条件 | 誤認の負例 | Owner |
| --- | --- | --- | --- | --- | --- |
| PC-97 | ページ化されたRelation Result | ページ内0件 | partial zero / complete zero | 先頭ページ0件から全体不存在を宣言 | #2506/#2586 |
| PC-98 | 時点付き0件Claimと追加Event | 0件確定後の挿入 | 版・Scope付き不存在 | 過去の不存在を現在へ無条件継承 | #2586/#1708 |
| PC-99 | 部分集合に対する数値集計 | 非負SUMへの負値追加 | 非負・追加のみの下界条件 | 無条件のSUM下界として保証 | #2586/#1705 |
| PC-100 | 元Fact集合とJOIN結果のBag | JOIN重複と集計 | Entity/Relation Grain、重複 | 2×3 JOIN行を元Entity6件と数える | #2387/#2506 |
| PC-101 | LEFT JOINの未一致行と後続一致 | LEFT JOINの仮NULL撤回 | SQL NULL/未一致/後続一致 | 旧仮NULL行を確定として残す | #2387/#2586 |
| PC-102 | 動的Top-Nの表示集合 | TOP-Nの置換 | 追加と撤回の双方 | 新規上位行だけ足して旧下位行を残す | #2586/#2497 |
| PC-103 | 選択対象と非同期UI応答 | Selection epochと非同期応答 | UI世代/Business Identity | 旧A応答を新B選択へ混入 | #2506/#1708 |
| PC-104 | Event／処理／観測の三時点 | Event/Processing/Observation Time | 時刻の役割と順序 | 処理時刻の順序をイベント順と断定 | #2497/#2507 |
| PC-105 | 複数RelationのRead-set Snapshot | 同一タイムスタンプ異Snapshot | 時刻一致とRead-set整合 | 時刻文字列だけでSnapshot一致を宣言 | #1708/#2506 |
| PC-106 | 旧版・新版のSource locator | Model改版と同じlocator | 版間意味対応の証拠 | locator一致でOwner意味受理を継承 | #2505/#2497 |
| PC-107 | 異なる更新時点の2 Consumer | 片側Consumerのみ再描画 | 更新通知と実受信/描画 | 片側の成功で全Consumer同期を保証 | #2498/#2516 |
| PC-108 | 同値のValueと狭められたRead Scope | Value保持・Scope縮小 | 値の忠実度とClaimの全件性 | 型・byte一致を全件・業務意味一致と誤報 | #2497/#2586 |

**見立て**：単調な入力追加で保存できる結論と、集計・否定・TOP-N・未一致・訂正／削除で撤回を要する結論を分ける。結果の単調性は同じSchema・Universe・Scope・Identityなどの前提に相対的である。現行TEIへ新しいClaim Engine/時間Runtimeを追加することを先行決定しない。

## L. Measure再集計・根拠開示・用途相対Claim（PC-109〜120、2026-11-02追加候補）

[構造保存写像§13](STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)のAM-01〜12を参照し、[#2593](https://github.com/hat47x/tei-platform/issues/2593)のMeasure意味、[#2516](https://github.com/hat47x/tei-platform/issues/2516)のConstruction Context/Disclosureを交差検証する。全件机上の反証候補、実試験NOT RUN。

| ID | 構成・表現 | Pressure／確認ケース | 保存すべき意味 | 偽同値・誤成功 | Owner |
| --- | --- | --- | --- | --- | --- |
| PC-109 | 分割済みAggregate(mean, sum, count) | 平均の再集計 | group sum/count、対象集合 | 平均10/20から単純平均15 | #2593/#2497 |
| PC-110 | 群別比率(num, den)の集約 | 比率の再集計 | numerator/denominator、母集団 | 50%/80%の単純平均65% | #2593/#1705 |
| PC-111 | Fact identity付き同額行とJOIN | 同値の別Fact | Fact ID、JOIN出現 | `SUM(DISTINCT value)`で重複業務値を消去 | #2593/#2387 |
| PC-112 | 部門間に重なるPerson集合 | 二部門に所属する同一人物 | distinct ID、集合の重なり | 部門別人数の和を全社Distinct人数とする | #2593/#1126 |
| PC-113 | 時点の残高Stockと期間Flow | 時点Stockと期間Flow | 集計可能な軸と時点 | 二日分の残高を現在高とする | #2593/#2495 |
| PC-114 | 局所Median／分位数／Sketch | 中央値・分位数の再集計 | 原値/Sketch/誤差条件 | 局所中央値から全体中央値を確定 | #2593/#2544 |
| PC-115 | 通貨・時点を伴うQuantity | 異通貨・異時点の数量 | Quantity Kind、Rate、丸め | Decimal値の一致を加算可能とする | #1705/#2593 |
| PC-116 | 時間窓内の訂正付きStream | 遅着・訂正を伴う時間窓 | 暫定/確定、retract、watermark | 旧集計を最終結果として保持 | #2582/#2593 |
| PC-117 | 内部Evidenceと外部向けProjection | 内部Read可・外部Disclosure不可 | Evidence accessと開示条件 | 内部閲覧資格を外部送信許可とする | #2516/#1178 |
| PC-118 | 根拠を伏せた要約Claim | 開示可能な要約に必要根拠なし | ClaimとConsumer sufficiency | 同じ結論文で根拠充足を宣言 | #2516/#2497 |
| PC-119 | Source版とPolicy/Grantの別版 | Rule/Grant変更、Source byte同じ | Current applicability、Authority | 過去の意味受理/開示資格を自動継承 | #2516/#2505 |
| PC-120 | 同一意味からの複数Consumer出力 | CUI成功・SHO/外部未観測 | 実Consumerごとのread/load | 一Consumerの表示成功を全Target適合とする | #2498/#2516 |

**設計判断**：Value Fidelity、合法な再集計、Claimの成立、根拠アクセス、外部開示、Consumerへの実到達は異なる述語。これらを単一のPASS/LOSSLESSへ集約しない。新Measure Runtime、第二Authority/Registry、数学的な全域証明は前提にしない。

## M. 動的モデルの計算意味・複数Clock・専門Solver（PC-121〜132、2026-11-02追加候補）

**新しく発見する問い**：「何が接続されているか」という静的なGraph/Relationと、「どの時間基準、遷移・スケジューラ・Solver・状態を用いて値や効果が進むか」という*Model of Computation*は別の意味である。同じ循環Graphでも、即時計算の代数ループ、1-stepのDelayを持つfeedback、非同期Eventの循環を区別する。M1〜M10のどこに記録すべきかを圧力試験し、必要なら**動的意味の新しいメタ質問**を#2495へ提案する。新しいTEI Core Runtimeやゲーム/数値Solver内蔵をこのCorpusから自動決定しない。

| ID | Pressure / 資料上の構造 | 構成・表現 | 分けて問う性質 | 誤認・損失の負例 | 既存Owner |
| --- | --- | --- | --- | --- | --- |
| PC-121 | 同じGraph上の純粋なDataflow | Operator DAGと入力値 | 値依存／評価順／副作用なし | Edgeがあるだけで実行順・決定性を保証 | #988/#2495 |
| PC-122 | Delayを持つFeedback Loop | Graph循環＋時刻付きState | 1 tick前の値／初期状態／単位 | 循環を全て不正・全て即時解決と判定 | #988/#2507 |
| PC-123 | 同時刻の代数ループ | X→Y→Xと連立拘束 | Solver必要条件／収束・解の存在／失敗 | Delayを推測したり順番実行で確定値を作る | #2544/#2495 |
| PC-124 | 複数ClockのSample-and-Hold | 60Hz物理更新と10Hz認知出力 | Clock同一性／位相／保持・stale | サンプル値の等しさから同時刻の評価を断定 | #2506/#2497 |
| PC-125 | ODEと数値積分 | 連続時間の微分式と離散Step | 刻みh／Integrator／誤差境界／事象検出 | 数値列一致から解法・連続時間意味の同一性を宣言 | #1705/#2544 |
| PC-126 | Simulation TimeとWall Time | 時間倍率を変えたゲーム・試験 | sim delta／real timeout／input時点／再現条件 | Slow-motionの成功を実時間応答期限への適合とする | #2507/#2497 |
| PC-127 | 状態付きReplay | Initial State・Event Log・乱数種 | 外部入力／Seed／Order／Snapshotの版 | 同じ初期値のみで非決定的実行を完全再現と断定 | #1131/#1180 |
| PC-128 | 同時刻の複数Event | Event QueueとScheduler | 同時刻tie・優先順位・因果順序 | Timestamp一致を実行順序・同一Eventとみなす | #2386/#2497 |
| PC-129 | 同期Functionと非同期Command | 同じI/O shapeのOperation | 副作用／ACK-loss／Receipt／Commit | 値の戻りが同じなので業務完了と推論 | #1174/#1129 |
| PC-130 | AI認知と身体運動の異なる周期 | CPU推論State→運動制御Tick | 遅延／契約するState／入力鮮度／遷移 | 推論が成功したことから次の物理動作が実現とみなす | #2507/#2497 |
| PC-131 | 専門Solverを持つ外部Plugin | ODE/最適化/制約Provider | 要求Operator／Solver依存・精度／回復条件 | 同じFunction Signatureで同じLaw/解を保証 | #951/#2544 |
| PC-132 | 訂正を含むReactive Aggregate | Snapshot差分＋retract/replace | 部分Claim／依存粒度／再評価／開示 | 追加Eventだけ反映して旧集計・旧Claimを確定表示 | #2582/#2593 |

**外部の圧力資料（正本採用ではない）**：[FMI 3.0.2](https://fmi-standard.org/docs/3.0.2/) はModel Exchange・Co-Simulation・Scheduled Executionを区別し、SolverやSchedulerを誰が持つかを分ける。[Modelica Language](https://modelica.org/projects/language/) は連続／離散・イベントを含むモデルを扱う。[Godot Engine physics interpolation](https://docs.godotengine.org/en/stable/tutorials/physics/interpolation/physics_interpolation_introduction.html) は物理Tickと描画Frameの非同一性を説明する。TEIがそれらのRuntimeを内蔵するという意味ではない。

**設計への帰属**：構造・Facetと動的意味の区分は#2495、演算束縛・Lawは#2544/#988、実行先・専門Providerは#951/#2077、Context/時間・Reactivenessは#2506/#2582、効果と権限は#1174、実Consumerでの保存性は#2497/#2498を利用する。各ケースは合成した**試験仮説**でありPC-121〜132の実行・受理はNOT RUN。

## N. Consumer間Claim比較・Document Occurrence・開示非干渉（PC-133〜144、2026-11-02追加候補）

[構造保存写像§14](STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)のCR-01〜12、[#2516](https://github.com/hat47x/tei-platform/issues/2516)のW8/DP-M4a/b、SHO #4/#6のDocument Occurrenceを接続する。**全件机上の反例候補、実Comparator/SHO/OS/AI試験NOT RUN**。

| ID | Pressure | 構成・表現 | 保持・比較する意味 | 誤成功 | Owner |
| --- | --- | --- | --- | --- | --- |
| PC-133 | 売上200と在庫200 | 互いに異なるQuantity KindとClaim Subject | Claim対象/Quantity Kind | 値一致で意味同値とする | #2516/#2497 |
| PC-134 | 部分100と全件200 | 異なるDataset Scopeの2 Observation | Scope/Completeness/時点 | 値不一致で片方を誤りと断定 | #2516/#2506 |
| PC-135 | 同一Claim/版で200と201 | 同じSource/Scope/Grainを主張する2 Claim | 真に比較可能な不一致 | AI多数決で正解を決定 | #2516/#2514 |
| PC-136 | 旧SHO文書と現CUI | 旧固定納品版と現在のCUI Projection | Model/Document/Reviewの版 | 保存成功を現在意味受理とする | #2516/#2505 |
| PC-137 | optional missing/nullが同表示 | Read-onlyなoptional Fieldの状態投影 | 表示同値と元状態同値 | 省略から元のPresenceを復元 | #2585/#2497 |
| PC-138 | 二つの大整数が同じfloat64 | 異なる整数Raw Valueとfloat64での丸め | Projectionの非単射性 | 値一致を厳密なID同一とする | #2585/#1705 |
| PC-139 | 同一SQL文字列、RLSが異なる | 実効RLS/Policy付きQuery Binding | 実効Query Scope/Binding | SQL字句一致で同一集合とする | #2387/#2516 |
| PC-140 | TEI要素一つ→文書二Occurrence | 同じTEI Subjectへの文書参照Occurrence×2 | Semantic ID/Document Occurrence ID | 一箇所編集で全箇所更新とみなす | #2497/SHO #4 |
| PC-141 | 一Occurrenceのみ局所上書き | 一つのDocument Occurrenceの局所Patch | 文書編集/TEI変更提案/Review | 局所説明を元モデルへ暗黙反映 | #2497/SHO #6 |
| PC-142 | 意味改版・表示byte不変 | Semantic Sourceと表示Documentの独立Revision | Semantic revision/説明・レビュー | byte一致でレビュー現在性を保証 | #2505/SHO #6 |
| PC-143 | 公開同一・秘密だけ異なる二世界 | Low-disclosure Projectionと秘密状態2世界 | Disclosure/診断/Traceの非干渉 | 拒否理由で秘密件数を漏らす | #2516/#1178 |
| PC-144 | CUI PASS、SHO実表示未確認 | 同じClaimのCUI/SHO別Consumer Read | Consumer別actual read/load | 一Consumer成功を全成果物の実受入へ拡張 | #2498/#2516 |

**比較原則**：`same value`、`same Claim`、`same Source`、`same Document Occurrence`、`same edit`、`same disclosure rights`を別の関係として扱う。比較可能性の前提が不明な場合、値の同値/異値だけで正否を決定しない。公開できない根拠を不存在へ変換しない。

## O. Everything is Data・意味Role・表記糖衣・Data Operations（PC-145〜156、2026-11-02追加候補）

新設された[#2619](https://github.com/hat47x/tei-platform/issues/2619)の横断方針と[#2620](https://github.com/hat47x/tei-platform/issues/2620)、[#2621](https://github.com/hat47x/tei-platform/issues/2621)、[#2622](https://github.com/hat47x/tei-platform/issues/2622)を圧力試験に接続する。関数・パイプライン・メソッド・Markdown表・Graph等の**構文上の違い**と、解釈Role・Operator束縛・型・Context・Target能力による**意味上の違い**を分離する。詳細な責務境界は[メタ概念設計§24](DATA_STRUCTURE_META_FACETS_AND_TOPOLOGY_2026-11.md)に記載。

| ID | Pressure | 構成・表現 | 保持・比較する意味 | 誤成功 | Owner |
| --- | --- | --- | --- | --- | --- |
| PC-145 | 関数形とPipeline風のwhere | 同じOperandとPredicateを持つ演算適用 | Operator ID/version/Type/Truth semanticsとSource span | 同じ綴りなので別Providerでも同じ作用と確定 | #2619/#2620/#988 |
| PC-146 | Method形式と関数形式の同名where | Namespaceの異なるOperator解決 | Role/Operator binding・Scope・Effect | 受入順・import順により意味の違う演算を暗黙選択 | #2620/#2544 |
| PC-147 | Filter→Limit と Limit→Filter | 同じ2演算のPipeline異順序 | 順序とResult Scope、要素の選択条件 | 同じ演算集合なので同じ結果と正規化 | #988/#2387/#2622 |
| PC-148 | SQL WHEREとGraph辺Filter | RDB bagのRowとGraphのEdge/Path | Cardinality・Traversal/時点・NULL/重複 | 同名Operatorから同一のIterator/Traversalを推定 | #2622/#2387 |
| PC-149 | Tree/Graphの再帰where | 循環Graphと遅延・深さ制限 | Termination、Visited ID、Path occurrence、完全性 | 有限Prefixの取得から全件の一致を宣言 | #2622/#2495 |
| PC-150 | 意味ModelのRelationとFunctionの利用集合 | Data Model Relation／Context Mapping | 起点・path・Field・Grain・Read Scope・完全性 | モデルにRelationがあるので全関連Dataを暗黙取得 | #2506/#2622 |
| PC-151 | Markdownの説明表と関数定義表 | 同型のMarkdown Tableと別Interpretation Role | Source span／Role/Owner受理／実行候補 | 全表の見た目一致を自動実行の許可とする | #2621/#2620 |
| PC-152 | 関数表・式・DSLの同じ計算候補 | 量と数値の型付きExpression／複数Authoring | Operator ID・単位・丸め・NULL・出典と独立Oracle | 字句が違うので意味別正本を重複生成 | #2619/#2621/#1705 |
| PC-153 | 同一Recordに複数Interpretation Role | 構造Record＋Function/Test Roleの束縛 | Required Shape・適用順・Operator競合・版 | Roleの配列順で評価意味を暗黙確定 | #2620/#2544 |
| PC-154 | 自己記述Pluginの依存循環 | Role/Operatorメタ宣言＋bootstrap Reference | 起点となる基本構造・Cycle/Version/Plugin解決 | 自己記述できるから全依存解決済みと表示 | #2620/#1018 |
| PC-155 | Provider pushdown とローカルwhere | 同じData ContextのQuery→Result候補 | 三値論理・Collation・粒度・権限・部分取得 | 少数の値が一致しただけで全Target等価を断定 | #2622/#2498/#2387 |
| PC-156 | Derived Dataset/文書/コードへの同版Projection | 一つのSource＋複数意味Role/Consumer | Semantic Subject・Source/Rule/Artifact/Consumer版・Loss | 生成した表/SHO/Codeの成功を実Consumer/Owner受理へ昇格 | #2619/#2497/#2604 |

**受入と状態**：12件はすべて**提案・NOT RUN**。現在のTEIがTypeScript系、Pipeline構文、Markdown自動解釈やGraph/Treeの全Queryをサポート済みという意味ではない。RDB・Graph・文書など異質なConsumerで、**明示Operator/RoleとContextの条件が揃えば共通構造へ投影できるか／異なるときは何を未解決と表示するか**を検証する。Core TypeKind、新Parser・Data Engine、第二Semantic SSOT、独自Hostの先行追加を求めない。

## P. 観測閉包・物理Read・Action開示の三重境界（PC-157〜168、2026-11-02追加候補）

[メタ概念設計§26](DATA_STRUCTURE_META_FACETS_AND_TOPOLOGY_2026-11.md)の根拠を、[独立Go Reference #2627](https://github.com/hat47x/tei-platform/pull/2627)のRC-18〜20（Complete RowsetとField absentの区別）、[#2616](https://github.com/hat47x/tei-platform/issues/2616)の論理要求／実取得／Consumer公開、[Action適用 #2613](https://github.com/hat47x/tei-platform/pull/2613)の公開可能Action候補に接続する。値の表示一致を、実取得の最小性・全Fieldの観測済み・実権限・開示資格へ昇格させない。

| ID | Pressure | 構成・表現 | 保持・比較する意味 | 誤成功 | Owner |
| --- | --- | --- | --- | --- | --- |
| PC-157 | Rowsetは全件、対象Fieldはabsent | Complete Bag＋未観測Field | 行集合の完全性とField値の既知性 | 全行を読んだのでField比較も一致 | #2627/#2585 |
| PC-158 | 片側はSQL NULL、片側はabsent | Observed-nullとUnobserved-value | 観測値の不一致と比較不能の区別 | NULL≠absentを矛盾したDataと確定 | #2627/#2056 |
| PC-159 | 同じSnapshot token、発行Providerが別 | 異Issuer＋同Token文字列のRead Slice | Cohortの実証・Read-set条件・時点 | token字句が同じなので同一Snapshotと宣言 | #2611/#2622 |
| PC-160 | PartialとCompleteの二Query | 異なるRowset Closure・同Scope申告 | Read Completeの実証と粒度・Bag | 先頭ページに値がないので全件不存在とする | #2506/#1706 |
| PC-161 | 同じ表示、非表示Rule依存が異なる | Display Field＋別Rule Input | D／V／A_dataと適用済みRule版 | 表示JSON同値でValidationも一致と宣言 | #2624/#1729 |
| PC-162 | 出力は同じ、Providerは秘密Fieldを取得 | Logical Map＋Actual Read Trace | データ要求・物理Read・開示の別資格 | 外部JSON一致から過剰Read無しを証明 | #2622/#2515 |
| PC-163 | 公開Subjectは同じで秘密Actionだけ増える | Action Offer集合とDisclosure Filter | 候補数・Status・Orderの開示資格 | 非公開Actionの存在を候補一覧から漏らす | #2613/#2516 |
| PC-164 | Preview後にRule/Grantが変更 | Context epochと更新前提の差分 | Read時判断と保存時Authority/Effect | 古いPreview PASSから保存OKと宣言 | #2598/#1174 |
| PC-165 | 同じ値、別Tenant RLSで観測範囲が異なる | Query Binding＋実RLS Scope | 実Row Visibilityと対象母集団 | 値一致を同じ全件性・同じAudienceとする | #1769/#2622 |
| PC-166 | Actionが存在しても公開は不可 | Semantic Action ID＋Consumer Policy | 適用可能性とAction存在公開の区別 | Capability存在からAction名を外部提示 | #2507/#2516 |
| PC-167 | 同一Sourceで異なるClaim粒度・鮮度 | Display/Rule/Action別依存グラフ | 差分InvalidationとCurrent Claim | Source版の数字が等しいので一括現在扱い | #2582/#2593 |
| PC-168 | 生成物一式の一部をConsumerが読めない | Manifest＋部分Artifact SetとReader | 期待集合の網羅と実Read/公開世代 | 生成成功から外部成果物受入を断定 | #2630/#2604 |

**資格**：PC-157〜168は新しい**設計負例**であり本Corpus内での正式TEI試験はNOT RUN。#2627の独立32件/#2613の独立42件などの局所PASSは、Source→Rule→Provider→Action→実Host/SHOの製品E2Eを示さない。検証の正本・Ownerは既存#2586/#2506/#2611/#2622/#2516/#1174/#2604へ戻す。新しいRead Planner、Policy Engine、Host、固定Core enumを提案するものではない。

## Q. SQL Operator意味保存と複数Function編集の合成（PC-169〜180、2026-11-02追加候補）

[構造保存写像§18](STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)のOP-01〜12に対応する机上Pressure。#2387 QX-23と#2597/#2601の独立Ownerを参照し、**本Corpusでの実試験はNOT RUN**。

| ID | Pressure | 保存・比較する条件 | 誤成功 | Owner |
| --- | --- | --- | --- | --- |
| PC-169 | JOINとEXISTS | Bag多重度・存在判定 | 同じFieldで意味同値 | #2387/#2622 |
| PC-170 | NOT INにNULLとNOT EXISTS | SQL三値論理・NULL | 両者を常に同値とする | #2387/#2514 |
| PC-171 | LEFT JOIN擬似NULLと実NULL | Row存在・Value来歴 | 表示NULLで同じSource状態 | #2387/#2506 |
| PC-172 | LIMIT前後のFilter/Sort | 順序・対象集合・Tie | 演算順序を無条件交換 | #2387/#2622 |
| PC-173 | 同値を持つ別FactへのDISTINCT | Identity・Bag・Grain | Fact IDを値の重複で消す | #2593/#2387 |
| PC-174 | AVG再集計でsum/count欠落 | 再構成Witness | Decimal型だけで再集計可 | #2593/#2544 |
| PC-175 | SQLite一致・別Target未観測 | Target意味・独立Oracle | 全Provider適合とする | #2387/#2498 |
| PC-176 | 同じSQL IDで別RLS | 実効Binding/Scope | SQL字句で同一母集団 | #2611/#2622 |
| PC-177 | 二Functionの並行別行更新 | Predicate/Tx/業務不変条件 | 個別Commitで全体受理 | #2597/#1174 |
| PC-178 | Save前のRule/Grant変更 | 現行Rule・Authority | 旧Previewから実行許可 | #2598/#1174 |
| PC-179 | Commit応答喪失後のRetry | Operation identity/Effect Lookup | 応答不明を未実行と断定 | #1174/#2597 |
| PC-180 | UI保存成功、独立read-backなし | Actual Effect/Business Result | 表示成功で業務完了 | #1174/#1119 |

**注意**：SQL→Data Operatorは既存SQL正本の置換を意味せず、編集Session合成は共通Transaction Engineの導入を意味しない。既存Ownerの隔離SQLite試験を横断製品E2Eへ合算しない。

## R. 空結果の否定依存・再資格化・Product Skill実利用（PC-181〜192、2026-11-02追加候補）

[構造保存§19](STRUCTURE_PRESERVING_TRANSFORMATIONS_AND_PROJECTIONS_2026-11.md)のNE-01〜12に対応。**設計反例であり、本Corpusでの実行はNOT RUN**。

| ID | Pressure | 保持する意味・証拠 | 誤成功 | Owner |
| --- | --- | --- | --- | --- |
| PC-181 | NOT EXISTS 0件→追加 | Predicate/Negative Dependency | 0件を依存なしとする | #2387/#2582 |
| PC-182 | 0→1→0のABA | Change Observation Window | 最終値同一で無変更とする | #2582/SHO #6 |
| PC-183 | 非表示WHERE Field変更 | Predicate Lineage | Display Fieldだけで再利用可 | #2387/#2624 |
| PC-184 | Relation付替え・同値集合 | Relation Path Binding | Value同値で同じ関係とする | #2506/#2387 |
| PC-185 | Partial 0件・全件未確認 | Read Completeness | 全件不存在とする | #2506/#2516 |
| PC-186 | 別Provider同名Snapshot | Cohort Issuer/Scope | Token字句で同一Snapshot | #2611/#2622 |
| PC-187 | 無関係変更と依存変更 | 選択的Invalidation証拠 | 全失効または全再利用 | #2582/#2018 |
| PC-188 | 意味不変・開示Policy変更 | Current Consumer Eligibility | 意味再利用で公開許可 | #2516/#1178 |
| PC-189 | Role定義あり、CLI未確認 | Capability実装証拠 | Roleで利用可と宣言 | #2619/#2605 |
| PC-190 | Skill読込、Tool未呼出し | Harness実Invocation | Skill loadで成功とする | #2605/TAI #271 |
| PC-191 | 回答同値、Tool候補/送信異なる | Planner/egress開示 | 最終回答同値で安全とする | #2516/TAI #271 |
| PC-192 | CLI read-only PASS、Effect未実行 | Consumer/Host実Effect | 読取り合格で業務完了 | #2598/#1174 |

## 外部仕様の照合（2026-11-01読取り）

1. [Apache Arrow Format](https://arrow.apache.org/docs/format/Intro.html) のNested Layouts：List/Struct/Map/Unionの物理配置と型の区別。特にMapはList of key/value structsと共通の物理形に近いが、同じ概念的Map保証をそのまま他Targetへ転送できるわけではない。PC-41〜44の**Source参考**。
2. [xarray Data Structures](https://docs.xarray.dev/en/stable/user-guide/data-structures.html)：DataArrayのvalues/dims/coords/attrs、座標ラベルに基づくalignment。PC-35は**形状と座標意味が別**であることの参照例。TEI適合は未検証。
3. [Protocol Buffers Field Presence](https://protobuf.dev/programming-guides/field_presence/) と [Proto3 Language Guide](https://protobuf.dev/programming-guides/proto3/)：optional/implicit・oneof・repeated・map間でpresenceと順序の契約が異なる。PC-45〜47は**意味情報を失う可能性**の反例。
4. [OGC Simple Features](https://www.ogc.org/standards/sfa/)：Geometry階層とSpatial Reference Systemの文脈。PC-39/40のReference。
5. [W3C SHACL](https://www.w3.org/TR/shacl/) と [JSON Schema 2020-12](https://json-schema.org/draft/2020-12/json-schema-core)：別の制約の検証と結合の考え方。データが符合すること、TEIのOwnerによる業務意味の受理、実Providerでの業務結果は別。
6. [BIPM SI Brochure](https://www.bipm.org/en/publications/si-brochure/)：量・単位の外部資料。通貨・暦・企業固有単位等には別の文脈が必要。

各標準の**仕様が述べる事実**と、表の**TEI向け試験仮説**は異なる。ライセンス確認を経ずに仕様本文・実装物を再配布しない。版・Target条件が変わる場合はその都度参照を固定する。

## 反証と帰属の作業票（既存Issue/Scenarioで実行する）

| 検証レベル | 最初の検証の方法 | 正当化できる範囲／未確認 |
| --- | --- | --- |
| C0 文書分類 | 既存メタ軸と七役割へ正負ペアを手動写像 | 言語化可能・不足軸の発見のみ |
| C1 構造化read-only | 既存Semantic、TypeRef、Relation、Source/Queryを固定版で読み、観測条件と損失をJSON等で提示 | 宣言・依存の可視範囲。型の正式受理・DB効果は未確認 |
| C2 合成fixture | 既存#1131/#1180のOracle/Scenarioで対象Operationに必要なObserverを比べる | fixtureとScopeの内部に限定。外部Consumerは別 |
| C3 外部Target/Host | 実Artifact bytes・Provider/Host・DBや別言語の独立観測を固定する | 実Targetで観測した範囲のみ。Owner意味受理は独立 |
| C4 文書・Studio | 同じSource/Contract/固定版から表・図・SHOへ派生し、消えるfacetを明示 | 表示の同版整合。操作・業務完了は別判定 |

### 調査と実装の並列化

- #2495担当は分類上の**足りない問い**・七役割の重複を評価する。
- #2498担当は**対象Operator/Observerから見た保存性、喪失、可逆性、必要な証拠**を検証する。
- #2501担当はPressure Caseを拡張し、出典と正負例・既存Ownerへの帰属を継続記録する。
- #2493の高スループット体制では三レーンを並列化可能だが、同じSource/Targetの実装・複数IssueによるCore enumの独立変更は競合管理する。
- 大量トークンにより発散・調査量を増やしてよい一方、GitHub Actions、実DB/外部Target、Ownerの意味受理の作業容量を別に管理する。

**撤回・改訂条件**：七役割・10軸で説明できない具体例は、既存の箱へ強制しない。分類不足の証拠を残して軸を拡張する。既存Ownerに同等の検証があれば結果だけをリンクし、再テストや第二SSOTは作らない。

**正式採用待ち**：公開Facet Schema、共通Morphism Engine、標準Pluginの初期公開契約、各Targetの適合範囲。