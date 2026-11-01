# データ構造Pressure Corpus（#2501、調査・比較用）

- **作成**：2026-11-01。**状態：外部仕様と既存TEI Issueの静的比較、未採用・未実装・全Corpusの実試験NOT RUN。**
- **Owner**：[#2501](https://github.com/hat47x/tei-platform/issues/2501)。横断分類は[#2495](https://github.com/hat47x/tei-platform/issues/2495)、操作相対の情報保存は[#2498](https://github.com/hat47x/tei-platform/issues/2498)、開発要件の拡張運用は[#2493](https://github.com/hat47x/tei-platform/issues/2493)。
- **位置付け**：異質な構造から概念的な圧力を集めるための**探索台帳**。新しいCanonical/型Registry/Feature enum/Scenario SSOTではない。全個別製品の仕様をTEIが実装したという意味でもない。
- **読み方**：各行の「確認すべき性質」は*試験で見たい条件*、「誤認・損失例」は*反証すべき仮説*。該当する既存Ownerへ戻して具体的Fixtureを設計する。事実が支持された実装と、未実施の提案を区別する。

## このCorpusで比較する七役割

対象の同一性／構成文法／性質・法則／解釈文脈／利用箇所の要求／主張と根拠／実現と表示。各Pressureの意味に関連するM1〜M10（#2495）は重複して構わないが、**一つの性質を二つの独立な意味正本に保存しない**。

以下の56ケースは**初期の異質性を測るための開始点であり、網羅の完了や上限を意味しない**。新規の構造・数学的性質・分野固有の解釈は追加し続ける。関心事が重複する場合も、操作・利用文脈・失われる情報が異なれば別の反例として保持する。

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