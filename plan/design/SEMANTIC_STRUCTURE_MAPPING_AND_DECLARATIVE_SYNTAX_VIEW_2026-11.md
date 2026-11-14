# 標準データ構造の意味的な対応付けと、構文・図解・編集の共通設計

- **状態**：実装前の横断設計・検証計画。正式なSemantic Schema、Grammar IR、公開Plugin ABI、Studio機能、実Provider適合、Ownerの意味受理を主張しない。
- **目的**：JSON Schemaなどの外部構造定義や多様なDSLを、TEIの標準データ構造・意味Role・共通UIへ安全に接続する。同時に、Graph／Tree／Table／時系列などの相互表現に必要な意味保存条件を明示する。
- **最優先事項**：Studio DesktopのStage1→Stage2セルフビルドと、製品Function Entry→実Provider→独立Consumerの縦断を妨げない。文法生成器や新しいRuntimeの構築を先行条件にしない。
- **Owner境界**：[構造・Facet #2495](https://github.com/hat47x/tei-platform/issues/2495)、[演算法則 #2544](https://github.com/hat47x/tei-platform/issues/2544)、[構造Projection #2497](https://github.com/hat47x/tei-platform/issues/2497)、[Language／Operator #988](https://github.com/hat47x/tei-platform/issues/988)、[意味Role・自己記述 #2620](https://github.com/hat47x/tei-platform/issues/2620)、[Studio #1124](https://github.com/hat47x/tei-platform/issues/1124)、[共通UI #1824](https://github.com/hat47x/tei-platform/issues/1824)、[独立Expected #2514](https://github.com/hat47x/tei-platform/issues/2514)。外部資料の取込は#958、Source忠実性は#2621／#2497、開示は#2516、変更影響は#1143、実行権限は#1174に従う。
- **重複させない文書**：[意味TraitのClaim](SEMANTIC_TRAIT_CLAIMS_AND_CONFORMANCE_2026-11.md)、[原子的Trait](TRAIT_ATOMIC_COMPOSITION_AND_FULFILLMENT_2026-11.md)、[Syntax正規化](EVERYTHING_DATA_SYNTAX_NORMAL_FORM_EQUIVALENCE_2026-11.md)、[SourceとRole](EVERYTHING_DATA_INTERPRETATION_AND_SOURCE_BINDING_2026-11.md)、[Studio開放性](GENERIC_STUDIO_OPEN_WORLD_PROJECTION_CONFORMANCE_2026-11.md)、[Source編集](SCRIPT_FIRST_GENERIC_STUDIO_PROJECTION_REFERENCE_2026-11.md)、[DSLとOperator](DSL_OPERATOR_PLUGIN_COMPOSITION_DESIGN.md)、[プログラム実行図](PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)。
- **非目標**：第二のCanonical正本、独立Semantic Engine／Resolver／Host／Registry／Gate、あらゆる外部構文を完全に可逆化する万能Grammar、任意のViewからの無条件の原文書戻し、UI操作による権限の自動付与。

## 1. 結論

TEIの標準データ構造は、単にデータの形を共有するためのものではない。構造ごとに「何を同一とみなすか」「どんな関係・順序・時間・数量があるか」「どの操作が成立するか」を、必要な範囲で明示できるようにする。その性質を利用して、**構文、意味、表示、編集、実装を別々に進化させる**。

新設するのは大きな型階層ではなく、既存のMeaning／Facet／Law／Trait／Projection契約をつなぐ、小さな**対応付けの検証票**である。本文の契約項目と識別子は比較用の語彙であり、本番用Schemaや公開APIへの採用案ではない。

利用者には数学的な性質名を前面に出す必要はない。「複数の親を持てる」「循環がある」「同じ関係が複数回現れる」「この編集は原文に戻せない」といった自然な説明へ投影する。Studioの表・図・入力フォームは、独立した意味の正本にはならない。

## 2. 共有する意味特性と、混同してはいけないもの

| 観点 | 明示すること | 取り違えた場合の問題 |
| --- | --- | --- |
| Identity | Stable Subject ID、Occurrence、業務キー、物理PK、Graph Node/Edgeの別 | 同名ノード、多重辺、同名再作成を併合する |
| Relation | 方向、両端の役割、属性、多重性、適用時間 | 所属・先行・参照を同一視する |
| Collection | 集合／多重集合／列、順序、重複、一意性、粒度 | 行数・件数・集計結果が変わる |
| Presence | 存在、明示NULL、欠損、未取得、非開示、空集合 | 「見つからない」を「存在しない」と断定する |
| Quantity | 数量の意味、単位、精度、丸め、測定尺度 | 不適切な加算・平均・比較を許す |
| Time | 有効時点、記録時点、観測時点、ルールの適用版 | 過去の意味や現在の観測を混同する |
| Scope | 母集団、参照範囲、取得完全性、Context、Revision | 部分読取で否定条件を真にする |
| Authority | 受理済みの業務意味、現在Grant、実Effectの別 | 見えているデータを実行許可と誤認する |

これらを一律に全データへ付与することはしない。必要なMeaning Claim／Operator／Consumerに応じて特性を参照し、未宣言は未宣言のまま扱う。型ラベルの一致は、意味的な適格性の証明ではない。

### 三種類の同一性

1. **原文の同一性**：ファイル、byte列、出現箇所、Source Revision、文法／変換器の版。改行、注釈、図の配置も保持対象となる。
2. **構造候補の同一性**：参照が解決された標準データ構造と、対象Claimに必要なRole・順序・識別子・制約。原文が違っても一致し得る。
3. **受理された意味／実際の振舞いの同一性**：OwnerによるMeaning受理、Operator Law、Provider／Target、実観測・受領を含む。構造候補のFingerprintが同じだけでは成立しない。

World Fingerprint、Model Revision、Artifact Revisionも混同しない。Source移動、構文Pluginの更新、生成器の変更、Consumerの取得範囲は、影響するClaimごとに失効条件を定める。

## 3. 構造間の対応付けは「変換できる」だけでは不十分

以下を区別して扱う。

| 関係 | 意味 | 例と注意点 |
| --- | --- | --- |
| 表現上の同値 | 明示した観測について同じ結果になる | 同じDataを表・ツリーで読む。原文一致までは要求しない |
| 埋め込み | 元の要素・関係を失わず別構造へ載せる | Tree→Graph。ただし根・辺属性・順序の保存方法を決める |
| 射影 | 指定した部分だけを取り出す | Graph→隣接表、Table→集計Chart。通常は逆変換できない |
| 集約 | 粒度を変え、情報を要約する | Event→日別集計。個別Eventは復元できない |
| 解釈 | 構造に業務上の役割を与える | Graph→作業先行関係。構造だけでは意味受理されない |
| 近似 | 精度・点数・空間解像度等を意図して落とす | 時系列の間引き。誤差と適用範囲を明記する |

同値の判定は、何を観測するかによって変わる。ある変換 f が、指定した観測 O について O(source)=O(f(source)) を満たしても、原文忠実性・全制約・実行効果まで等しいとは言えない。「無損失」を主張する場合は、対象Scope、構造・属性・参照・順序の保存条件を列挙する。

**Graph→Treeの負例**：二つの同名Node、同じ両端を持つ複数の有向Edge、別の親から入るEdge、循環、Edgeの属性がある世界では、単純なTree表示だけでは元のGraphを一般に復元できない。読み取り専用のTree Viewは許容し得るが、Tree編集から元Graphへの自動書戻しは別判定とする。

**WBSの例**：作業の包含関係と、作業間の先行関係は別の意味Roleとする。階層図とPERT図を同一モデルから派生できても、包含から先行関係を勝手に生成しない。

## 4. 小さな対応付け検証票（正式Schemaではない）

構造変換や表示を評価する際、既存の#2497 Projectionと#2495/#2544のFacet／Lawに、次の観点を照合する。

| 項目 | 最低限の問い |
| --- | --- |
| source / target | 入出力の型、対象Subject、Scope、対象版は何か |
| preconditions | どの構造的・意味的条件で適用できるか |
| preserved / omitted | 何が維持され、何が省略・近似されるか |
| identity / provenance | Node、Edge、行、Source Occurrence、参照先の対応は何か |
| read / edit | 表示できる範囲と、原文へ戻せる編集範囲は何か |
| authority / disclosure | Consumerに何を見せてよいか。実行権限はどこで判定するか |
| evidence / invalidation | どのIndependent Expectedを満たし、何が変われば再検証するか |

例：Graph→Treeで「非循環・根が一つ・各Nodeの親が高々一つ」を必要条件にするときも、Edge IDと属性、子の順序、孤立Nodeなどを別に照合する。適用条件が揃わない場合は「変換不可」または「情報損失ありの読み取り専用表示」とし、成功扱いで捨てない。

同じ契約をOperatorの適用可能性にも使える。たとえばtopologicalSortはGraphという名前ではなく、対象Relationが要求する非循環性・方向性・Scopeを満たすときに適用できる。一方、sum／averageは数量の単位・尺度・NULL・丸め・粒度に依存する。可換・結合・分配などの法則は名前だけで推論せず、#2544の検証可能なClaimに従う。

## 5. JSON Schemaは「Recordの別表記」ではない

最初の外部仕様の実証例としてJSON Schemaを用いる。ただし、JSON Schemaは**JSONインスタンスの妥当性を定める制約言語**であり、Schemaそのものと検証対象のRecordを混同しない。

読み込み時に保持すべきもの：
- 仕様Dialect／Vocabulary、`$schema`、`$id`、参照Scope、原文byte・Occurrence。外部参照は取得元・版・時点・完全性を別に記録する。
- `properties`／`required`／`additionalProperties`などの制約と、注釈・表示用説明の別。
- `$ref`／`$dynamicRef`、`$defs`、再帰参照、`allOf`／`anyOf`／`oneOf`／`not`、`if`／`then`／`else`。
- `unevaluatedProperties`など、評価経路・注釈収集に依存する制約。単純なフィールド一覧へ平坦化して「同等」とは扱わない。
- 真偽値Schema（`true`／`false`）、存在しない属性と明示NULLの違い、`default`などが自動補完・書込みの義務とは限らないこと。

**Studioの基本表示**：項目名、型の候補、必須かどうか、参照、制約の有無をまず示す。複合制約・原文・Dialect・参照解決・検証上の詳細は展開やツールチップに載せる。JSON Schemaが制約から導かないこと（例えば確定した業務キーやDB制約）は追加推論しない。

**編集時**：Source byte範囲、解決済み参照、対象Revision、変更意図が一意の場合だけ候補Patch→差分Preview→再検証へ進む。共有された`$ref`先、複数Schemaの合成、派生した画面では、意図せず広域変更を起こさないよう原則read-onlyまたは明示的に編集対象を選び直す。Schemaの表示・Source編集・JSONインスタンスへの入力・DB Migrationは四つの異なる操作である。

**外部仕様の契約**：Dialect未対応、参照取得不能、部分取得、Vocabulary未解釈は成功した完全解釈にしない。従来どおりplain-dataの構造表示へ退避できる範囲と、検証／編集できない範囲を区別する。

## 6. DSLの自己定義を、専用の巨大なParser基盤にしない

言語ごとに「構文の読み方」と「標準データ候補への対応」を分ける。

```text
Authoritative Source (TEI DSL / Markdown / JSON Schema / DOT / ...)
  → Language Pluginがparse・Source位置・診断を供給
  → plain standard data candidate
  → Role／Operator／参照の解決、制約検証、OwnerのMeaning受理
  → Studio共通View／構文固有View／Generator／Scenario
```

文法は、EBNF／ANTLR／Tree-sitterなどの既存技術を比較対象とする。ただし、**Grammar IRの新設や全言語の共通Parser生成は未採用**。まず#988の既存Language Plugin Referenceから、構文規則、エラー回復、補完候補、Source occurrence、Semantic Bindingのどこまでを共通化できるか測る。

入力支援を三段階に分ける：
1. **構文候補**：次に置ける記号、キーワード、繰返しなど。
2. **意味候補**：Role、型、束縛済み参照、単位、Cardinality、Operatorの適格性。
3. **Context候補**：許可された現在のSubject、Providerが実際に読み取った候補、開示できる値。未取得を空の候補集合と同一視しない。

Studio内のGUIからGrammar／Viewを記述できる方向は維持する。ただし初期の独自Viewは宣言的なレイアウト、型付きBinding、参照ジャンプ、Tooltip、可読な警告を優先する。任意コードの注入や自動実行許可を標準拡張にしない。専門Viewは#951のPlugin契約で追加できる余地を保つ。

自己記述の到達目標は、文法・Role・Viewの一部を標準データとして説明・検証できること。自分自身のParserを完全再生成できたことや、全ターゲットで意味同値であることは別の検収とする。固定された最小Reader／bootstrapと、自己定義を更新する際の互換性・退避経路を保つ。

## 7. Studioでの一貫した操作

同じSubjectを、テキスト、構造ツリー、関係図、表、フォーム、Source Diffで切り替えられることを目標とする。中心は**安定した対象選択**であり、図上の座標や表示Edgeを意味IDにしない。

- 表示は意味未解決でも、開示可能な`path / structural type / disclosed value`へ退避する。
- Tooltipは意味Role、制約、適用範囲、診断、Source位置を見せる。未受理の候補は未受理と表示する。
- 図表と原文のジャンプはSource ID／Occurrence／Revisionに基づく。Sourceの移動やA→B→Aは、構造・編集・受理履歴・Consumerが何を同一視するかに応じて照合する。内容byteが同じことだけで、却下履歴や外部Consumerの資格まで復活させない。
- 編集は`read-only`、`candidate patch`、`explicit source edit`、`supported reversible edit`を分け、後二者も権限・意味受理・実Effectを代替しない。
- 開示の違いから非公開のSubjectの有無・件数・診断が分からないよう、#2516のConsumer別試験に従う。
- Studio内の人間とAIは同じ型付きSelectionとPreviewを使うが、AI候補の生成とOwnerの意味受理を分離する。

新しいDomain Pluginを追加した際の重要指標は、専用のStudio Core分岐を追加せず、既存のText／Property／Tree／Graphへ必要な情報を表示できるかである。Graphや高度な可視化をすべて汎用UIへ押し込むのではなく、専用UIが必要な理由と効果を比較する。

## 8. 反例を先に固定する検証計画

次のケースは**Independent Expectedの候補**であり、本書作成時点で新たな実試験を実施したという意味ではない。既存#2497／#1131／#2514／#1124のScenarioへ割り振り、類似Fixtureを重複新設しない。

| ID | 入力／操作 | 期待する判定 |
| --- | --- | --- |
| SM-01 | Graphに同名Node二つ、同一両端のEdge二つ、属性あり | Node／Edge同一性と多重性を保存。Treeへの無損失書戻しを拒否 |
| SM-02 | Graphに複数親・循環・孤立Node | Tree適格性を個別診断。適切なread-only表示へ退避 |
| SM-03 | WBSの包含辺と先行辺 | 両者の意味を分離。包含だけで依存辺を生成しない |
| SM-04 | Tableのbag、NULL、missing、partial取得 | 件数・否定判定・集計の意味を誤って確定しない |
| SM-05 | 時系列の単位・精度・観測時点を変える | 明示した観測範囲でのみ同値性を判定 |
| SM-06 | JSON Schemaの`oneOf`／`allOf`／`if` | 単なる継承・項目表へ誤変換せず、制約を保持または未対応を表示 |
| SM-07 | 再帰`$ref`／`$dynamicRef`／未取得参照 | 解決状態と適用Scopeを表示。不完全な検証を成功としない |
| SM-08 | 同じ参照先が複数画面で利用されるSchema | 図編集で参照先を勝手に広域変更しない |
| SM-09 | 同じ標準DataがMarkdown、DSL、JSONから導かれる | 構造候補の同一性とSource identityを区別 |
| SM-10 | Grammar／Role／Operatorの版を変更 | 古い意味受理や補完候補を無検証で再利用しない |
| SM-11 | 未知Roleと`trusted-script`という入力値 | plain-data表示に退避し、Action／Grantを昇格しない |
| SM-12 | Viewの部分開示、stale Selection、曖昧な逆写像 | 情報漏えいと誤書戻しを防ぎ、理由を示す |
| SM-13 | `default`注釈や任意プロパティのフォーム表示 | 注釈を強制入力・値の自動確定と誤解しない |
| SM-14 | Sourceのコメント／DOT配置だけ変更 | Source差分は残し、意味差分・Artifact再利用を別判定 |

テストは二段階にする。まずHeadlessで同じSource・正解・Diagnostic・Source Jumpを固定する。次にStudio実画面／CLI／独立Consumerで、同じSubjectと判定を受領できるか検証する。図が描けることを意味保存やTarget適合の証拠にはしない。

## 9. 既存計画へ割り当てる順序

| 順位 | 作業と完了条件 | 既存Owner |
| --- | --- | --- |
| **P0（既存主線を継続）** | Stage1→Stage2 Studio自己再構築。並行して製品Entry→実Provider→実Consumerの資格を独立確認 | #4583／#4495／#2586 |
| **S0（並行・設計のみ）** | 既存のMeaning／Facet／Law／Trait／Projectionと本書のSMケースを対応付け、既存契約で記述できない最小差分だけ列挙 | #2495／#2544／#2497 |
| **S1（Headless）** | Graph／Tree／Tableの二つ以上の相互Viewを、同一ID・Scope・独立Expectedで比較。無損失／損失／readonlyを出し分ける | #2497／#1131／#2514 |
| **S2（外部Schema）** | JSON Schemaの単純例と複合例を読み、Source位置・制約・未対応状態付きで構造表示。編集はまだread-only | #988／#2620／#1124 |
| **S3（Studio限定編集）** | 一意のOccurrenceだけを候補Patch→Preview→再解析。共有参照・古いRevision・未対応Dialectで停止 | #1124／#2497／#2621 |
| **S4（宣言的View）** | 既存の汎用UIで新しいView定義を使い、Sourceジャンプ・Tooltip・入力補完を提供。追加PluginによるStudio固有分岐数を計測 | #1124／#1824／#951 |
| **S5（文法定義・自己記述の比較）** | 既存Language Pluginの不足を実測した後、Grammar規則の共通記述と外部Parser生成技術の採否を比較 | #988／#2620／#1121 |

S1〜S5は**P0の先行Gateにしない**。S0で既存契約だけで十分と判明した項目には新しいSchemaやEngineを導入しない。新しいIssueをむやみに増やさず、まず各既存Ownerへ検証条件と反例を渡す。

## 10. 採用判断

次のすべてを示せた部分だけ、既存契約の拡張候補とする。

1. 誰がMeaningのOwnerか、どのSource／版に帰属するかを説明できる。
2. 同一性・順序・多重性・NULL／missing／partial・Time／Unitなど、対象Claimに必要な意味を欠かさない。
3. 損失のあるProjectionを安全な編集可能Viewと取り違えない。
4. Studio固有処理を増やさず、新しい構文・Schema・意味Roleの少なくとも一例を表示できる。
5. 既存Expectedで正常例と負例が再現でき、Source→View→候補Patch→再解析の範囲が明確である。
6. 実Provider／Target／Consumer／Owner受理の資格を、合成試験や図解の成功と混ぜない。

**判断**：意味特性・構造間写像・構文とViewの拡張性を、TEIの長期的な共通基盤として計画に加える。ただし採用単位は既存契約に対する小さな差分とし、正式なGrammar IR・新Semantic Engine・全面的なSchema双方向編集は、実証結果に基づいて改めて判断する。


## 11. 意味保存を検査する単位を、実際の利用経路に合わせる（2026-11-14追補）

今回追加で確認したのは、**構造間の写像が数学的に正しくても、それだけでは利用者が期待する機能の同値性を保証できない**という点である。構文を読み取る段階、Queryを評価する段階、UIへ描画する段階、編集を原文へ戻す段階で、それぞれ保存すべき観測が異なる。

| 境界 | 対象となる「意味」 | 同値性の主要な観測 |
| --- | --- | --- |
| Syntax → Data candidate | 名前束縛、Literal/Ref、順序、Operator ID、Source位置 | 同一Source・Grammar/Role Pinで同じ候補と診断が得られるか |
| Data/Query → 実Rowset | Relation Path、JOIN粒度、bag、NULL、絞込条件、評価時点 | 同じSource/Query/実Provider、実入力から、対象Scopeの実値・取得範囲が一致するか |
| Data → View | Identity、関係、参照、単位、時間、開示可能な値 | Consumerに必要な情報を維持するか。省略情報を明示できるか |
| View edit → Source patch | 編集対象OccurrenceとRevision、原文忠実性、変更意図 | 対象以外のbyteを不変にし、再読込後の指定された意味だけが変化するか |
| Meaning → Target | Operatorの法則、失敗・副作用、順序、時間、Provider適合 | 独立Expectedと実Target／Consumer観測が一致するか。実Effectは別途確認するか |

**構造同値／評価同値／変更同値の三つを分離する。** 二つのQueryが同じ列の名前と型を返しても、JOINの重複やNULLの解釈、LIMIT、RLS、Top-N、NOT EXISTSの再評価が異なれば評価同値ではない。読み取り結果が一致しても、一方の更新が原文全体を書き換えたり、共有参照先へ意図しない影響を与えたりすれば変更同値ではない。

この分類を別のCore enumに固定せず、既存#2497のProjection、#2544のOperator Law、#2387のQuery、#1143の変更影響、#2514の独立Expectedに対する**Claim別の照合観点**として扱う。

### 11.1 同値性には条件と観測範囲を必ず添える

「元の意味を保つ」という表現は、無条件の保証に使わない。例えば、有限集合上で二つの純粋Operatorの計算結果が一致しても、遅延評価、短絡、失敗、時刻、Effectが異なれば任意のContextで置換可能とはいえない。

同値を主張する前に、少なくとも以下を決める。

- 対象Subject／Relationと時点、母集団、必要な列・辺・属性・単位は何か。
- 元のSource／Plugin／Operator／Meaning／Provider／Toolchainのどの版に依存するか。
- どの観測（値、順序、件数、エラー、Effect、診断、Source byte）が一致すれば十分か。
- 何を確認できなかったか。未取得、部分取得、非開示、未解釈Pluginは、完全な一致の証拠にはしない。
- 変換後のConsumerに必要な情報が、投影結果で本当に充足されたか。

## 12. 編集可能なViewには、双方向変換の法則を参考にする

一般の双方向変換（Lens）の考え方は、#2497の編集契約を検査する際に役立つ。ただし「Lensを新しい本番抽象として採用する」という意味ではない。

読み取りViewを \`get(S)\`、Viewの更新候補を原文へ反映する操作を \`put(S,V)\` と書く。

- **Get–Put**：何も変えなかったViewを戻しても、意味上の更新は起こらない。Markdownなどの原文保存では無関係なbyteも変更しない。
- **Put–Get**：許容されたView変更をSourceへ反映して再解析すると、その変更が指定のScopeで再び観測できる。
- **Put–Put**：更新の順序や競合を扱う場合、後から行った変更が以前の更新をどう上書きするかを明示する。一般に全形式で成立すると仮定しない。

これらの法則は、**特定のSource、Revision、操作、Scopeに対する条件付きの検査**である。整形、注釈、共有参照、Macro、外部Schema参照などがある場合、意味上の等価性とSource byteの忠実性は別条件になる。

### 12.1 Studioでの更新経路

\`\`\`text
固定されたSubject + Source Occurrence + Revision
        → UIが返す型付き編集Intent
        → Preview：元byteと提案Patch、関係する別Viewへの影響
        → Resolve：現在のSource／Role／Operator／参照を再解決
        → Admit：既存Owner／権限／競合条件で採否を判断
        → Commit：許されたSource Patchまたは別途許されたDB更新
        → Observe：独立した再読取りでSource／View／必要な実結果を確認
\`\`\`

この手順は既存の編集・実行Ownerへ渡す**説明上の接続**であり、共通Commit Engineを新設しない。Sourceへの書戻しとDBへのCommitは別の取引であり、一方の成功を他方の成功と見なさない。

**更新不可として残す条件**：複数のSource occurrenceから同一View要素を導出した場合、原文参照が曖昧な場合、取得範囲が部分的な場合、権限が不確かな場合、対象Revisionが古い場合、意味RoleまたはGrammarが変更された場合、共有されたSchemaの変更範囲が確定できない場合。表示とSourceへのNavigateは、開示可能な範囲で継続できる。

## 13. 情報の有無を表す状態を、構造的な空集合へ畳まない

Graphに表示する辺が0本、SQLのRowsetが0行、JSON Schemaの\`required\`が空、という事実は、いずれも「業務上その関係・制約・事象が存在しない」ことの証明とは限らない。

| 受け取った状態 | 成立する主張の例 | 成立しない主張の例 |
| --- | --- | --- |
| 完全なScopeで0件を観測 | 明示したScope・時点・Queryでは該当件数0 | 全期間・全Provider・全主体について不存在 |
| 取得件数0、Coverage不明 | この呼出しから0件が返った | 対象が存在しないため0件になった |
| partialなRowset | 返却済みの行と値は観測した | NOT EXISTSが真、集計が全量 |
| Schemaの未宣言属性 | その定義箇所に宣言が見当たらない | 実データに存在しない／禁止されている |
| 非開示のSubject | このConsumerにはその情報を提示できない | 「非公開のものが存在する」と表示してよい |

たとえば\`enabled(order.submit) = not exists approval.pending\`は、許された範囲に実際の未承認行が1件見つかれば否定できる場合がある。一方、空の返却だけで肯定するには、対象母集団、条件、Snapshot、取得完全性、Providerの制限が必要になる。これは#2582／#2516／#2497の既存境界を再利用する。

新しい未知値の万能enumを作るのではなく、既存のPresence・Coverage・Disclosure・Freshnessの根拠を組み合わせ、**各Consumerが言ってよいことの上限**を明示する。非開示の有無を診断メッセージや補完候補数で漏らさない。

## 14. DB／専門領域との接続を、二つの最小縦断で検証する

### 14.1 外部Schemaの構造表示：JSON Schema → Studio

最初は読取専用に限定する。固定したJSON Schema Sourceについて、Dialect、Source occurrence、参照範囲、型・\`required\`・条件付き制約・注釈を識別し、項目ツリー／制約詳細／原文ジャンプへ投影する。

**合格条件候補**：
- 同じSourceとPlugin Pinで、Headlessの読み取り結果・診断・Sourceジャンプ先が再現される。
- \`oneOf\`と\`anyOf\`、\`allOf\`と継承、\`default\`と値の確定、未対応Vocabularyと成功を混同しない。
- \`$ref\`の共有先・循環・未解決、外部取得のpartial/unavailableを画面から見分けられる。ただし開示不可情報は漏らさない。
- 複合制約を表に展開しきれなくても、制約自体は保持し、Viewが表現した範囲だけを明示する。
- 元のSchemaを編集・保存する前の段階では、Sourceを一切書き換えない。

**仕様に基づく詳細条件**：最初の固定DialectはJSON Schema Draft 2020-12を比較候補とする。同版では`prefixItems`／`items`の配列制約、`$dynamicRef`／`$dynamicAnchor`の実行時の動的Scope、`format-annotation`と`format-assertion`のVocabulary差を、単純な静的フィールド表へ丸めない。特に`$dynamicRef`の解決先は、原文で示された参照URIだけでは十分に確定しない場合がある。出典：[JSON Schema Draft 2020-12](https://json-schema.org/draft/2020-12)、[Core specification](https://json-schema.org/draft/2020-12/json-schema-core)。

**導入を急がない機能**：任意のDialectの完全な検証、すべての制約の自動フォーム変換、双方向Schema Migration、文法生成器の一般化。まず固定Dialect・有限Fixtureで不適合を説明できることを優先する。

### 14.2 実Queryの構造投影：PostgreSQL Read → Table／Aggregate

#2586の製品Read資格を得た後のConsumer検証として、既存の\`order_entry\`受注読取り等を題材に、**同一の実RowsetからTableと集計Viewを別々に派生**する。実PG読取りそのものはこの設計文書が所有せず、既存Ownerの資格を参照する。

**合格条件候補**：
- Query、引数、Subjectと実Provider、Snapshot、必要なField、粒度、NULL、Row ID、Row Count、制約・取得限界を同一Attemptから照合できる。
- Tableでは行とKeyが保持される。集計では集計単位、条件、NULL・重複の扱いと、失った個別行情報が説明できる。
- LIMIT／RLS／Native SQLの未知条件で全量性が立証されていなければ、集計結果を「対象全件の合計」と表示しない。
- CLIとStudioなど異なるConsumerでは、同じ意味の受領を**それぞれ**検査する。描画の成功とProvider実Call、正しい開示と対象全量取得を混同しない。
- Derived Viewは原則read-only。Row Editを許す場合、Subject Pin、条件付き更新、Commit Receipt、独立Post-commit Readを別経路で受け入れる。

これはData Source Workbench #3270、Query #2387、Function Data Context #2506、Projection #2497、Consumer資格 #2586に対する**同一の受入連鎖**であって、独立したDB抽象や第二のQuery IRを提案しない。RDBとNoSQLの双方へ展開する際も、原文Query、Provider特有の評価、実取得のCoverageを消して共通化しない。

### 14.3 WBS・科学計算は「意味特性の異種検証例」として後続に置く

[専門ツールキット計画](CROSS_DOMAIN_MATH_STATISTICS_PLANNING_TOOLKITS_2026-11.md)／#4660の題材を、共通基盤の適格性を測る対照群として使う。

- **WBS／PERT**：包含と先行の辺を分け、同名Task・多重辺・循環を扱う。単なるGraph表示からクリティカルパスの計算可能性を推定しない。
- **統計／Window**：PARTITION、ORDER、Frame、NULL、行粒度、データ全量性を伴う。SQL方言またぎの見かけの同型を意味同値へ昇格しない。
- **数量・科学計算**：単位、尺度、精度、許容誤差、境界条件、Solver/アルゴリズム版、計算の決定性を明示する。浮動小数点の厳密一致を無条件に要求せず、独立Expectedが指定した誤差・条件で判定する。

専門Toolkitの実装はStage2セルフビルドのP0を止めない。汎用Coreへ追加するのは、複数分野で必要性が実測され、既存Facet/Law/Projectionで表現できない最小要素だけとする。

## 15. 自己記述・差分・Build再利用の境界

GrammarやRoleの自己定義は、[Role Bootstrap設計](ROLE_BOOTSTRAP_AND_SELF_DESCRIPTION_2026-11.md)の有限Seedから始める。\`formatVersion\`、Language／Parserの版、Role／Operatorの版、Generator／Toolchain、Source Revisionを区別する。文法の自己記述が読めること、生成されたParserが動くこと、TEI自身を再ビルドできることは三つの別成果である。

区別して扱う変更例：

| 変更 | 必ず再評価する対象 | 自動的には変化しないと主張できない対象 |
| --- | --- | --- |
| Markdown注釈、DOTレイアウト | Source byte／Occurrenceと表示差分 | 語彙内の参照、生成対象、受理済みMeaningへの影響 |
| SourceのA→B→A、移動、同名再作成 | Source内容、Occurrence、依存Closure、Decision履歴、Consumer Selectionを主張ごとに照合 | 内容が一致しただけで却下履歴、実Consumerの受領、現在のAuthorityを復活させること |
| Grammar／Language Pluginの更新 | Parse／診断／正規化／補完と、必要なMeaning Claim | 既存の受理済み意味・Target適合 |
| Role／Operator Law更新 | 対象Operationの意味・結果・適格性 | 旧生成物の意味同値・Cacheの正当性 |
| Generator／Toolchain更新 | Artifact生成と実利用資格 | Sourceが同じときのBundle再利用資格 |
| Provider／Scope／Grant更新 | 実Read、開示、Action Authority、Consumer受領 | 過去の合成Fixture PASSの製品資格 |

変更影響は、**affected／scoped-unaffected／unknown**を既存の依存ClosureとClaim別評価へ返す。Unknownをすべて無影響と扱わない一方、無関係な未知を理由に全Sourceを必ず再受理させる必要もない。古いResultや同名のSourceを使って、失効したConsumer・Authority・受理の資格を復活させない。ただし同一byteに対する純粋な候補Patchの適用可能性は、対象Sourceと依存が一致する範囲で改めて判定できる。履歴を理由に無関係な純粋編集まで一律失効させない。

## 16. 追加反例と既存Issueへの引渡し

§8のSM-01〜14を維持し、他チャットとの整合で不足していた反例だけを以下に追加する。すべて**未実行のIndependent Expected候補**であり、新しいScenario台帳を増やす意図はない。

| ID | 検証対象 | 独立した期待結果 | 主な既存Owner |
| --- | --- | --- | --- |
| SM-15 | 同じ列構造でJOIN粒度・bagが異なる二Query | Shape一致でも評価同値と判定しない | #2387／#2497 |
| SM-16 | NOT EXISTSで0件、ただしReadはpartial | 否定条件の真を証明しない | #2582／#2497 |
| SM-17 | ViewからSourceへ同じ値を保存、Markdown書式あり | No-opでは原文byte不変。最小Patch以外の再formatを拒否 | #2621／#2497 |
| SM-18 | 集計Viewで値を変更するIntent | 戻す行が一意でないため自動逆書戻しを止める | #1124／#2497 |
| SM-19 | Grammar版だけ更新、Source byteとRole名は同じ | ParseとMeaningの必要Claimを再評価し、旧受理を自動継承しない | #988／#2620 |
| SM-20 | 同一実RowsetからTableとAggregate、RLS/LIMITあり | 受領した範囲の集計と全件集計を別表示。実Read資格は別 | #2586／#2387 |
| SM-21 | 同じGraphをWBS階層図とPERT図で描く | 包含と先行のRoleを分離し、計算入力の循環を診断する | #2495／#4660 |
| SM-22 | Solver版・精度・単位だけ異なる科学計算結果 | 独立Expectedが定めた適格性と誤差でのみ同値を認める | #2544／#4660 |
| SM-23 | Schema Viewerが非開示の参照先を補完／診断する | Consumer側の公開結果から存在・件数・Source名を推測させない | #2516／#1124 |
| SM-24 | SourceをPreviewした後、Revision／Grantが変化 | Resolve／Admitをやり直し、古い選択でCommitしない | #1124／#1174 |
| SM-25 | Schema／Viewの変更後、実DBの列が消えたように見えるがCoverage不明 | Desired Sourceや受理済み業務意味を自動削除しない | #3270／#1143 |

### 16.1 どの段階で何を成果と呼ぶか

- **S0（設計で確認できる範囲）**：SM-01〜25を既存の契約・Issue・Fixtureへ振り分け、重複・未表現・判断保留を列挙する。ここに書いたことで製品適合とはしない。
- **S1（Headlessの最小実証）**：Graph↔Tree／Tableの同一性と損失判定に加え、SM-17／18で編集可否の差を確認する。Graphの同名Node・多重Edgeは最初から含める。
- **S2（Schema読取専用）**：JSON Schemaの単純・複合・未解決参照・Dialect不適合を同じStudio generic表示へ出す。完全対応率を捏造しない。
- **S3（限定編集）**：Source occurrence／Revision／byte差分を備えた一箇所の合法Patchのみを通し、SM-17／18／24を負例にする。
- **S4（共通View）**：新しいRole／Schemaを追加してもStudio Coreにドメイン分岐を増やさず、表示・Navigate・開示資格の差を確認する。
- **S5（構文定義比較）**：既存Language Pluginの実測結果と必要な支援機能を比較し、Grammar IR／生成技術を採用する十分な根拠が出た場合に限ってOwnerが判断する。
- **製品縦断（既存P0の成果を参照）**：#2586の実Read資格が成立した後、同じAttemptからのTable／Aggregateを独立Consumerで照合する。S0〜S5や専門ToolkitをP0達成の先行Gateにはしない。

**次の着手単位**は、まず既存#2497のProjection／編集契約とSM-01／04／17／18のExpectedを照合し、「新しい型を増やさずに書ける項目」と「本当に不足する項目」を一枚で示すことである。そこにGapが認められない限り、Semantic Registry、Grammar IR、Edit Engine等の新規共通基盤を要求しない。


## 17. 損失のあるProjectionにおける項目・操作別の編集資格（2026-11-14追加）

**補正する判断**：Graph→Treeのように写像全体が非可逆でも、「表示されているNodeの所有属性」のような局所操作まで無条件に拒否する必要はない。一方、ノード移動やEdgeの付替えは、見えない親・並行Edge・Edge IDに依存するため、同じ画面から操作できるように見えても許可できるとは限らない。

これは新しいProjection Engineの要望ではない。[#2497](https://github.com/hat47x/tei-platform/issues/2497)の写像契約を、**全体の情報保存判定と、選択したSlot／操作に限った逆写像判定**に分けて検証する提案である。対応する根拠は[#4533](https://github.com/hat47x/tei-platform/issues/4533)の既存実測と、[Draft #4538](https://github.com/hat47x/tei-platform/pull/4538)のGo Reference。

### 17.1 先に判定する五条件

以下の五条件が成立する範囲だけを「原文へ変更候補を返せる」とする。

1. **対象の一意性**：表示要素から、型付きSubjectと編集対象Occurrence／Slotが一つに解決される。同名ノードでもIDが別なら混同しない。
2. **値の出所**：編集値が当該Subjectの所有する属性に対応し、集計・結合・丸め・換算・多対一変換を通じて生成された値ではない。
3. **局所Patch**：原文中の変更範囲が確定し、依頼されていないコメント、空白、他属性、別のSourceを変更しない。
4. **現時点の対応**：Sourceの内容・対象span・依存・必要なSelectionの現在性を検査する。Sourceの純粋なPatch適用可能性と、Owner受理・Authorityの現在資格は別。
5. **再検証**：Preview→既存review／accept→再parse／再投影で変更が一致する。No-opなら原文byteを変更しない。

これは「編集候補を作れる」ための条件であり、Ownerによる意味受理、Source書込権限、実DB更新、Effect Receiptを与える条件ではない。

| Projection／操作 | 全体の損失 | 局所判断 | 理由 |
| --- | --- | --- | --- |
| Graph→Tree：Node所有の名称変更 | 第二の親・多重Edgeなどが失われる | 一意のSource属性なら候補可能 | 変更対象Slotは写像で消えていない |
| Graph→Tree：Node移動 | 同上 | 非表示辺と関係IDを含めた別検査が必要 | Tree上の親はGraph全辺を表さない |
| Graph→Tree：Edge属性変更 | 同上 | Edge ID・属性所有者が未確定なら拒否 | 同じ端点の別Edgeを誤更新し得る |
| JOIN→集計Chart：数値の直接変更 | 元行・重複・粒度を失う | 自動逆書戻しは拒否 | 集計1点の変更を行へ一意に配分できない |
| 部分取得→Table：読取済みセル編集 | 未取得行がある | 明示Row ID・Revision・条件付き書込を別に確認 | 既知の一行の更新と全件置換は異なる |
| Markdown表→Property：同じ値の保存 | 表記・空白・注釈の損失があり得る | No-opならSource byte不変 | 通常Saveは糖衣構文の再生成ではない |

[#4538](https://github.com/hat47x/tei-platform/pull/4538)の限定Referenceでは、Property/Tableの一属性編集・Source往復と拒否は確認されたが、Graph→Treeの**Node属性Slot**を全体損失から独立に判定する実装はまだ不足する、と報告されている。この具体的なGapを[#4351](https://github.com/hat47x/tei-platform/issues/4351)／#2497へ戻す。Graph一般の安全な書戻しを証明したことにはしない。

### 17.2 SourceのA→B→Aをどう読むか

[#4558](https://github.com/hat47x/tei-platform/issues/4558)の検討から、「A→B→Aなら常に全編集を拒否する」という一律規則は採らない。要求する主張ごとに以下を分ける。

| 主張 | 必要な確認 | A→B→Aで注意する点 |
| --- | --- | --- |
| 純粋なSource内容Patchが適用できる | 現在byte、対象span、依存、再解析 | byteと対象条件が同じなら適用可能性を再評価できる |
| Ownerが以前に却下していない | Decision履歴、同一候補、明示的なsupersedes | 内容が戻っても却下記録は消えない |
| Source書込を現在許可されている | 実施時点のAuthorityと対象Scope | 過去のReview／表示結果はGrantではない |
| 外部Consumerに現在公開してよい | Consumer、Audience、Source／Meaning Pin、開示条件 | Bを受け取ったConsumerの資格をAのbyte一致から回復させない |
| Build成果物を再利用できる | World／Model／ArtifactとToolchain依存 | SourceだけがAに戻っても別の実装・利用資格が必要 |

Decisionの却下履歴を受理時にどう扱うか、過剰なWorkspace全体Revision失効をどこまで局所化するか、Source書込Grantを誰が検査するかは#4558／既存Acceptance Ownerの判断として残す。新しい全体単調カウンタやAuthorityをここから採用しない。

## 18. 「表示できる」と「表示し続けてよい」を分ける

Studioの非同期読取では、Source／Context／Grantの古いResultが後から到着し得る。入力時点の条件が有効でも、**描画を確定する時点（Render Commit）**に同じ情報を公開してよいとは限らない。

既存[Studio open-worldの検証計画](GENERIC_STUDIO_OPEN_WORLD_PROJECTION_CONFORMANCE_2026-11.md)では、Lane単位の失効、全Lane reset、以前描画した値の残留、後着Responseを限定Referenceで検討している。共通のepoch tokenや第二Policy Engineを作るのではなく、この確認をView利用条件へ接続する。

初期表示、非同期更新、選択後のTooltip、補完候補、検索索引、AIに渡すDesign Contextは、それぞれ**当該ConsumerのDisclosureと現在Selectionに相対**である。必要なCurrent Claimがない場合、公開可能なplain-data表示へ退避するか、関連内容を非表示にする。

**重要な反例**：権限変更時にResponse Scopeをresetしても、既に表示したDOM／ARIA／検索候補／Graph Node／AI Contextが消去されたとは限らない。変更通知を受け取れない実Consumerは即時撤回を保証したことにせず、次の利用・描画時点の資格確認と既存内容の再投影・消去の責務を別に検査する。

| 状況 | 表現の判定 | 更新・実行に関する判定 |
| --- | --- | --- |
| 未対応Role、Sourceは公開可能 | plain-dataとしてinspect／Source Navigate可 | 意味を推測する編集・実行は不可 |
| Sourceは読めるが、共有参照の更新範囲が曖昧 | 参照・制約の表示可 | 当該参照先の一括更新は保留 |
| 非開示の参照先が必要 | 公開可能な断片だけ表示 | 参照の存在・件数を漏らす補完／診断は禁止 |
| 古いRead ResultがUIへ到着 | 現在の選択・開示条件を再検査 | 過去のallowを現在のallowへ流用しない |
| Previewの後でAuthority／Sourceが変更 | 既存Previewは履歴として区別 | Resolve／Admitし直すまで適用不可 |

## 19. 既存ReferenceへのFit/Gapと最初の検証票

S0では新しい実装を始める前に、既存の成果を正確に再利用する。既にあるReferenceと今回の提案を混同しないため、次を一枚の対照として扱う。

| 既存Owner／実証 | すでに比較できること | 残る不足 | 本設計から渡すSMケース |
| --- | --- | --- | --- |
| [#4351](https://github.com/hat47x/tei-platform/issues/4351) のFacet Mapping | 12列契約とXMAP負例、情報損失・逆写像のread-only判定 | 操作別Slotの意味損失と実Source編集は別 | SM-01／02／04／18 |
| [Draft #4464](https://github.com/hat47x/tei-platform/pull/4464) の投影Matrix | 4構造×8表現型のうち11セルを限定Go Referenceで評価、未カバー21セルを区別 | 未判定セル、実Renderer、実Provider、Owner受理 | SM-01／05／21 |
| [#4533](https://github.com/hat47x/tei-platform/issues/4533)／[Draft #4538](https://github.com/hat47x/tei-platform/pull/4538) の局所編集 | Property/Table値のSource span／Revision／Preview／再投影の限定往復 | Graph→Tree Node属性Slotの過剰拒否、Source書込Authority、正式Studio | SM-17／18／24 |
| [#4558](https://github.com/hat47x/tei-platform/issues/4558) の履歴・権限 | byte／Review／Decision／Grant／Consumerの現在資格を分離 | 却下後受理、Source書込Grant、全体Revisionによる過剰失効のOwner判断 | SM-19／24 |
| [Studio Open World](GENERIC_STUDIO_OPEN_WORLD_PROJECTION_CONFORMANCE_2026-11.md) | plain-data退避、未知Role、非同期Resultの限定失効対照 | 実DOM／ARIA／補完／AI受領時の再許可と撤回 | SM-11／12／23 |
| [#2586](https://github.com/hat47x/tei-platform/issues/2586)／[RA-0〜RA-3計画](SCRIPT_LAUNCH_ENTRY_TEST_QUALIFICATION_2026-11.md) | 型付きQuery、限定実PG Read、Coverageの区別について既存受入段階を定義 | 通常製品CLI／実Provider／同一Contextの独立二Consumer | SM-15／16／20 |

**S0の成果物はこのFit/Gap表と既存Expectedへの照合**であり、表の行を新しいIssue／Fixture／Validatorとして実装することではない。最初の小さな実装候補は、#4351と#4538が実際に示した「全体はlossyでも一つのNode属性Slotは編集候補になり得る」というGapに限定する。#2497のOwnerが受入条件を確定してから既存Referenceを拡張する。

**合格を示す最短の対**：
- 正例：同じGraph→Tree投影で、安定Node IDと一意Source spanを持つ名称変更が、既存Preview／Acceptance／再投影で成立する。
- 負例：同じ表示からEdge IDなしのNode移動を要求した場合、非表示の辺と多重性を破壊するため拒否する。
- どちらも同じSource／Model／Plugin Pinと独立Expectedを用い、実Graph WriterやGraph専用UIの新設を必要条件にしない。Node属性を編集できてもGraph一般の変換が無損失になったとは報告しない。

### 19.1 過剰な開発範囲を避ける停止条件

「投影できる」という判定にProperty編集・Graph再構築・実DB更新・Stage2セルフビルドの達成をまとめない。既存の12列契約のClaimで区別できるなら新しいSemantic Schemaは不要。別途新しい共通Primitiveを提案するのは、**異なる分野の二例以上で再現可能な不足**があり、既存Facet／Law／Trait／Projectionでは表現不能であると示せた場合に限る。

また、JSON Schemaの読取専用表示、Studio実UIの安全編集、実PostgreSQL→二Consumerの受領はそれぞれ別の段階である。Product P0のStage1→Stage2と実Readを最優先に維持し、本書のS0〜S5を新たなHard Gateへ格上げしない。


## 20. 合成Projectionで失われる意味と、Pluginから公開される表現（2026-11-14横断整理）

既存[合成表現型の適格性 #4122](https://github.com/hat47x/tei-platform/issues/4122)／[設計本文](COMPOSED_REPRESENTATION_MAPPING_QUALIFICATION_2026-11.md)は、D→V→Wの各段が局所成功でも、最後のWについてSubject同一性・完全性・逆編集可能性・現在開示を保証できないことを扱う。本書の役割はその原則を**構文・標準データ・プラグイン公開View・Studio編集へ適用する差分**に限る。第二の合成写像EngineやProjection Contractを追加しない。

### 20.1 例：多重Graph→Tree→Markdown

同じ人物が二部署に属する時点付きGraphをTreeへ変換し、さらにSHO文書内のMarkdown見出しへ変換する。

- 初段のTreeが主所属だけを表示した時点で、兼務を表すEdgeとEdge IdentityはViewから失われ得る。Markdownがきれいに生成されても元Graphの全関係が復元されたとは主張しない。
- Tree上でNode名を編集する操作は一意のSource属性へ対応し得るが、二段目のMarkdown見出しでSource occurrenceやStable Subject IDが失われた場合は、最後の表示からの直接書戻しを許可しない。
- 途中で失われた属性を下流が元Sourceへの資格付きReferenceから取得する場合、その情報は**純粋な合成写像による回復ではなく、新しいRead／Binding／Disclosureを伴う再取得**である。独立した根拠が必要。
- 一つのSourceからTreeとTableを並行に生成しても、両者のSnapshot、取得対象、Consumer資格が同じとは限らない。Viewごとの選択や現在性を明示する。

合成結果に対して問うのは、全構造の無条件の「可逆／不可逆」ではなく、用途が求めるObserver、Slot、操作、SourceとConsumerの必要閉包が最後まで通っているかである。**途中で失われた必要Claimを、後段で保存されたことに書き換えない。**

### 20.2 上位Pluginの図解・意味情報を下位Pluginで再利用する

別チャットで明確になった要件：あるPluginが契約に基づき公開した図解、型付きRelation、Role、注釈、Source位置などを、依存先の別Pluginが継承的に利用できるようにする。

ただし「継承」は、プログラミング言語のクラス継承やSourceの所有権移転を意味しない。**契約で公開された情報を、下流の用途で再解釈・合成・表示する**ことである。[Plugin貢献別合成](PLUGIN_ROLE_COMPOSITION_AND_STUDIO_REENTRY_2026-11.md)と既存#951／#1016／#2497／#2516／#1124へ割り当てる。

| 上位から公開される内容 | 下位で許される利用の例 | 自動的には与えられないもの |
| --- | --- | --- |
| 画像・SVG等の完成した図面 | 説明資料への埋込み、出典へのNavigate | Node/Edgeの全Identity、個別編集権限 |
| 型付きGraph／表現Descriptor | 表／Tree／別図への再投影、既存のGraph IDでの選択同期 | 非開示属性、失われた関係、受理済み業務意味 |
| Role／Constraint／Operator注釈 | 詳細説明、補完候補、領域に応じた専用View | Roleの正式採用、Providerの実適格、Host Grant |
| Source参照／Span／Revision | 原文へ移動、候補Patchの対象確定 | Source所有権、編集後の自動Accept |
| 検証結果／Evidence参照 | 下流の必要Claimに対する既存検証の再利用候補 | 新しいConsumer・Target・Plugin版の合格 |

初期に下流へ渡す**照合情報**は、選択済み上位Contract／Provider IDと版、公開SliceのSubject・Relation・Occurrence／Source位置、Scope・時点・Disclosure、どの意味特性が保存／省略されたか、下位が要求するOperator／Projectionの版、実Consumerと必要なCurrent Claimである。これは既存の型付きReferenceとPlugin選択の束縛情報を組み合わせるものであり、新しい公開ABIの採用案ではない。

下位Pluginが独自の意味Roleや辺を付加する場合、その追加分は**下位側が所有する意味候補**として識別する。上位から得た事実を上書きしない。同じNode IDと表示名でも、所属と先行、記録上の参照と実呼出しを黙って同じRelationへ変えない。複数上位Providerの同名Contributionが競合する場合は、Pluginロード順による勝手な選択をしない。

### 20.3 公開面の段階と退避動作

公開された図面だけが利用可能なら、下流は**図面の閲覧と出典参照**から始める。型付きGraph／Source Anchorがなければ、図形の配置から業務関係を推測して編集可能にしない。

型付きDescriptorと現在のMeaningが解決できる場合は、汎用StudioのProperty／Graph／Tree／Tooltipへの再投影を候補にする。編集は§17のSlot／操作別条件と既存Preview／Acceptanceに戻す。任意の実行Offerは型付きOperation／現在Host Grantと独立に検査する。

上位から非開示の関係を受け取れない下位Pluginが、隠されたEdgeの本数・存在・名前を補完候補や図解注釈で露出させてはならない。また、以前取得した上位の公開Sliceを、Grant失効後のStudio／SHO／AI Consumerへ黙って再掲しない。必要な場面では§18のRender Commit時の資格を取り直す。

## 21. 意味モデルから自然な説明を導出する条件

業務に使う図解は、設計意図と処理の関係を読者に伝えるためのものとする。図形化が必要な関係（Subjectの構造、依存先、時点、状態遷移）は図で示し、集計や式の説明は無条件に処理ノードへ展開しない。式やOperatorの内容に応じて、**表・数式・短い自然文**を選べるようにする。

説明文の生成は、受理済みの意味Role、型付き関係、Operator、条件、失敗・Effect、時点、Source位置など、実際に存在する情報を順に並べ替える処理として扱う。たとえば「対象の受注を条件で絞り、金額を合計する」と説明するなら、対象母集団、述語、NULL、集計粒度、結果の単位が確定している範囲だけを記述する。「全注文を集計する」などの全件性は、Coverageが不明なら補わない。

図解・説明・コード生成・テストが同じ意味IDを参照できるかを検査する。文章が自然であること、AIがもっともらしい説明を生成したことはMeaning Acceptanceの根拠ではない。説明に必要な情報が欠けている箇所は、Sourceへジャンプ可能な未解決事項として表示する。

この方針は[プログラム実行モデルの図解](PROGRAM_EXECUTION_MODEL_PROJECTIONS_2026-11.md)と、業務DSLを元に処理説明・外部部品Mock・独立Expectedへ展開する既存#4198／#988／#1124／#2514の受入へ戻す。すべての図を生成する新しい図面Compilerは導入しない。

## 22. 追加の受入対照と次の一手

この追補は#2497/#4122の合成写像、#951のPlugin Contract、#1124のStudio、#2516のDisclosureが交差する**具体的な検証条件**だけを追加する。以下は既存Expectedへ割り当てる**未実行の候補**であり、新しいテスト正本・番号体系を製品へ固定しない。

| ID | 反例 | 独立に確認すること | Owner |
| --- | --- | --- | --- |
| SM-26 | Graph→Tree→Markdown。兼務Edgeが初段で非表示 | 最終Markdownの成功でGraph全辺保存を主張しない | #2497／#4122 |
| SM-27 | Tree内のNode属性は一意だが、MarkdownではSource occurrenceが二つ | 最終Viewからの自動編集を止め、元Sourceを案内する | #2497／#1124 |
| SM-28 | 上位Pluginが図画像だけを公開、下位PluginがEdge ID編集を要求 | 画像の閲覧は可能だが型付きEdge更新は拒否 | #951／#2497 |
| SM-29 | 上位が型付きGraphと出典を公開、下位は別Roleで補助図を生成 | 公開Subjectと元Graph IDを保ち、追加Roleを下位候補として区別 | #951／#2620 |
| SM-30 | 上位の非開示Edgeを下位Tooltip／AI補完に利用 | 公開結果から非開示の存在・件数を推測できない | #2516／#1124 |
| SM-31 | 上位Pluginの版とSource Pinが変わるが、図の見た目は同じ | 旧Meaning受理・編集／Consumer資格を画面一致だけで継承しない | #951／#1143 |
| SM-32 | 集計図から「対象全件を合計」と説明、実Rowsetはpartial | 実際に確認したScopeだけを説明し、不足を表示する | #2387／#2514 |

**最初の一巡**はGraph→Tree→MarkdownのSM-26／27でよい。既存#4351の写像判定・#4538の属性編集・#4122の合成契約を使い、(a)全Graph情報の喪失、(b)一意のNode属性だけの編集資格、(c)二段目でSource位置が曖昧になった場合の拒否を、同じ独立Expectedから判定する。次にSM-28／29で上位Providerからの**公開図面のみ**と**型付き構造の公開**の差を比較する。プラグインの正式ABI、Stage2 Desktop、実Provider、実SHO ConsumerはこのHeadless検査の合格とは別である。

### 段階計画への割当て

- **S0 Fit/Gap**：#2497／#4122の合成写像と#951の貢献別公開で必要な情報が揃うか確認。重複契約を作らない。
- **S1 Headless**：SM-26／27を既存Referenceの延長として検証。失われた辺、Source occurrence、局所編集の三つを独立判定する。
- **S2 Schema表示**：JSON Schemaの共有\`$ref\`や複合制約を、上位の型付きSliceが公開できるか評価する。まずread-only。
- **S3 限定編集**：一意SlotのSource-backed Patchのみ実行し、合成後の不明なSource位置で止める。
- **S4 Plugin公開View**：上位が図だけを公開する例と、意味Descriptorまで開示する例を下位の同一Generic Studioで比較。正式な意味採用・開示・Authorityは別。
- **S5 DSL自己記述／生成**：そのViewと型付き契約をTEI Sourceから実際に説明・再生成できるかを確認。Go処理への単純delegateや、図が飾りであるだけの構成をセルフビルド成果とは数えない。

**全体P0は不変**：非Plugin Studio Desktop Stage1→Stage2の実再構築と、通常\`tei\` CLI→実選択Provider→独立Consumerの実受領を優先する。初期のHeadless合成／公開Viewは並行実証であり、新たなHard Gateではない。

**採否と記録先**：本書はPRレビュー用のFit/Gap整理にとどめる。合成写像の正式判断は#2497／#4122、Plugin開示は#951、Source編集は#1124、受理は各Owner／Issue／ADRを正本とする。日付入り設計文書を意味受理や要件の第二正本としない。

## 23. アプリケーション構造を型付き関係として辿る（2026-11-14横断追補）

他チャットのB2C向け横断的関心事、Lifecycle Graph、Plugin/Studio設計の共通点は、**アプリケーションのどの要素に何を適用するか**を型付きの関係として辿れることにある。ただし、アプリケーション全体の第二正本になる単一巨大Graphは作らない。[#4198](https://github.com/hat47x/tei-platform/issues/4198)のLifecycle Graph、[#1158](https://github.com/hat47x/tei-platform/issues/1158)の共通Application API、[#951](https://github.com/hat47x/tei-platform/issues/951)のPlugin選択、[#1143](https://github.com/hat47x/tei-platform/issues/1143)の変更影響、[#2497](https://github.com/hat47x/tei-platform/issues/2497)の構造Projectionを、**用途別の型付きGraph View**として照合する。

### 23.1 区別すべき三つの視点

| 視点 | 何を表すか | 主な問い | 主張しないこと |
| --- | --- | --- | --- |
| Definition Graph | Sourceに明示されたData／Function／UI／Relation／Operation／Plugin依存 | どの要素が存在し、どの契約を参照するか | すべてのProviderが実際に選択・実行された |
| Effective Composition Graph | 選択されたTarget／Application Profile／Provider／Hookと生成経路 | この構成ではどこに何が接続されるべきか | 生成物への収録、実Call、現在Grantが成立した |
| Observed Trace Graph | Attempt、実Provider Call、Artifact、実Effect、Consumerの受領 | 本当にどの経路を通ったか | 観測されない辺が存在しない、業務意味が正式受理済み |

これらのGraphは、既存Source／型付き参照／Function／Context／Build／Receiptに対するProjectionであり、新しいGraph Storeや中央Authorityを要求しない。実行観測からDefinitionへ「辺を発見」した場合も、Source-backed Observation／Candidateとして扱い、Owner受理なしに正式Relationへ昇格させない。

Graphの辺は最低限、所属・参照・呼出し・必要条件・実装・生成・表示・観測を区別する。例えば「画面がFunctionを参照する」と「そのFunctionが実Providerを呼んだ」は違う。また、Source occurrence、安定したNode／Edge ID、方向・多重辺、対象時点、Relation RoleとSource/Provider/Artifactの版を用途に必要な範囲で保持する。画面上の位置や順路番号を安定IDにしない。

### 23.2 全体Graphの完全性は仮定しない

特に横断拡張では「すべてのページ」「全Request」「あらゆる公開Entry」が対象になり得る。しかし、読み込んだSource集合だけでアプリケーション全体の閉包が確定したとは限らない。

Graphから対象を探索する際は、対象Universe、必要な依存閉包、未読み込み／非開示／動的登録／外部生成Targetの有無を区別する。探索で0件だった場合も、対象が実際に0件なのか、必要なSource／Plugin／Targetをまだ見ていないだけなのかを区別する（#2582／#2516）。重要な否定条件や全件選択では、必要なClosureの完成を根拠とする。

## 24. 横断的拡張はGraphを探索する明示的な規則として扱う

B2Cサイト全体のアクセス計測、監査、リクエストの相関ID、認証前後の検査、共通ヘッダー等は、対象Page／Entry／Operationに横断して適用したい関心事である。AOPの「Join Point／Pointcut／Advice」は比較のための語彙として有用だが、TEIの標準DSLへ採用済みと見なさない。

**設計提案**：既存のPlugin Contribution／Hook／Generatorに対し、次を既存Graph/Contractの参照で指定可能にする。
- **対象**：型付きのPage、Entry、Operation、またはLifecycle Hook。単なるSource文字列やURLパターンだけで実行主体と作用範囲を確定しない。
- **選択条件**：公開済みのMeaning Role、Target、Scope、必要Trait、依存Plugin/Operationの存在。全件条件には探索Universeの完全性を要求する。
- **介入地点**：before／after／on-error／render-timeなど、既存Lifecycle Graphが定義する拡張点だけを使う。任意の内部処理位置へ無制限にコード注入しない。
- **順序と競合**：同一Join Pointへの複数拡張、依存関係、短絡、例外、Effect、再試行の優先関係。未解決の競合はロード順で解決しない。
- **実現と観測**：Source上の宣言、選択したPlugin/Target、生成されたArtifact、実Hostの適用、実Consumerでの効果を別に検査する。

### 24.1 計測タグを全ページへ適用する例

「全ページへ分析タグを入れる」という要件は、**静的Pageの集合を取得できること**、**Web Targetのどの位置に生成するか**、**実ブラウザが条件を満たして読み込むか**、**同意・利用目的・開示要件に適合するか**という別々の問題である。Google Analytics等の計測SDKと広告配信タグは同じものとは限らず、実装方法・目的・同意要件を区別する。

初期実証は外部配信を伴わない**合成の計測Contribution**で行う。Source側に「公開Pageのrender Hookでテスト用マーカーを生成する」という候補を置き、選択済みTargetの二つのPageで生成物を検査する。第三Page追加で必要な対象閉包が変わること、非公開Pageが混入しないことを確認する。

同意・PrivacyやContent Security Policy、実Hostの認可・CSP/Browser動作はTarget／専門Plugin／HostのOwnerとし、Graphの選択に成功しただけで満たしたことにはしない。非同意・未取得・未対応Targetを「タグ挿入済み／稼働済み」と報告しない。Graphは適用候補を示すのであって、政策判断や実Effectの第二正本ではない。

### 24.2 変更影響と再生成を局所化する

新しいPage／Entryが増えた場合は、対象選択に影響するClosureを再評価する。計測Pluginの説明文だけを変更した場合、必要な技術契約・生成入力と分離できるなら実Buildや意味受理を過剰失効させない。Selector条件、Hookの順序、Provider版、Target／Generatorの版が変わった場合は、関連するEffective GraphとArtifact再資格化が必要になり得る。

この判別は既存#1143／#2582／#2600／#3267へ戻す。Static Source差分、Meaning影響、Build Artifact、実Runtime/Consumer効果は別票とし、新しい増分Build Engineを立てない。

## 25. プラグインの説明と依存文書を、同じ契約から辿れるようにする

別チャットで具体化した要求は、各Plugin自身に**短い概要、用途、必要条件、公開するRole／Operator／Hook／View、選択と利用の手順、制約・負例、詳しい文書への参照**を備え、人間と生成AIの両方が、上位・依存Pluginの説明へ自然に辿れることである。

これを別のPlugin Documentation Registryとせず、既存#951のContract／Provider／Dependency、#1016のApplication Profile、#2605のProduct Skill、#1124のStudioとCLI/MCPの既存の型付きInspectに接続する。

| 利用者の問い | 最初に見せる情報 | 詳細へ辿る先 |
| --- | --- | --- |
| このPluginは何か | 一文での目的・主な貢献・採用状態 | 契約、版、対応Target、Source・文書 |
| 何に依存するか | exactな必須Dependencyと任意Contribution | 上位Pluginの公開契約・説明・Pin |
| このデータをどう見せられるか | Public View／対応Facet、必要Role | Projection損失、Source Anchor、図面・操作条件 |
| この操作を実行できるか | 利用可能な型付きOperationと必要入力 | 実選択Provider、Host Grant、効果・失敗契約 |
| なぜ使えないのか | missing／ambiguous／unsupported／withheld等の理由 | 対象Source、依存Closure、必要資格と既存診断 |

文書の継承的な閲覧は**文書の参照関係の辿り方**である。上位の説明を下位へコピーして第二正本にしない。上位の版やContractが変わった場合、表示する説明と生成AIの参照先も選択済みDependency Pinに従う。同名Plugin・同名Roleを自動選択せず、複数の候補は明示的に区別する。

**AI Harness向けSkill入口**は、TEIの基本用語、CLI優先での安全な読取、対象選択、詳細文書を探す経路、Preview／検証／変更提案と実適用の境界を、短い入口文書として提供する。その入口に全Pluginのマニュアルを展開しない。詳しい説明・型・Scenario・適格性は、その場で製品から問い合わせる。実CLIのコマンド名は既存の実装を検査してから記載し、未実装の探索コマンドを実在するものとして案内しない。

図面に表示した要素の選択をAIへ渡す際は、表示文字列や画面座標だけでなく、型付きSubject／Relation／Source occurrence／Revision／Context・開示状態を可能な範囲で固定する。選択結果をAIへ送ったことは編集対象のAcceptや現在Grantの付与ではない。AIが開示できない上位Pluginの文書や非公開Relationを逆探索できないよう、公開可能なSliceをOwnerに従って制限する。

## 26. 最小受入条件と実装順（新たな製品Gateではない）

次の候補を既存#1180／#2514のIndependent Expectedへ引き渡し、#4198／#951／#2497／#1124／#3509に担当を割り当てる。**本書で新しいテストIDを製品固定したり、全TargetのGraph Compiler完成を求めたりしない。**

| ID | 入力・変更 | 独立に期待する判定 | 主なOwner |
| --- | --- | --- | --- |
| SM-33 | 同じ名前の二つのPage／異なるSubject ID | タグ適用先を名前だけで併合しない | #4198／#2497 |
| SM-34 | 静的Page二件、外部に未読込Entry一件 | 「全Page対象」と断定せず、探索Scope／未知を示す | #4198／#2582 |
| SM-35 | Page追加、計測用Contributionは不変 | 有効なSelector Closureに追加Pageを含み、必要Artifactだけ再評価 | #4198／#2600 |
| SM-36 | 同じJoin Pointに競合する二つのHook | 順序／短絡／Effectの衝突を診断し、ロード順で黙って解決しない | #4198／#951 |
| SM-37 | PluginがGraph画像だけを公開、下位がRelation編集要求 | 画像の閲覧だけを可能にし、Edge編集は拒否 | #951／#2497 |
| SM-38 | Pluginが型付きGraphとSource Anchorを公開 | 下位が公開Scopeで再Projection可能。元の意味・IDと追加候補を区別 | #951／#1124 |
| SM-39 | 依存Plugin A@v1→A@v2、同名Role・説明変更 | 選択Pinと文書の参照版を照合し、旧版意味・実適格性を自動継承しない | #951／#2605 |
| SM-40 | AIが図の同名Nodeを指定、Source occurrenceが曖昧 | 候補の一覧とSource Jumpを提示し、編集・実行への自動昇格を止める | #1124／#2516 |
| SM-41 | 非公開の上位文書が依存閉包に存在 | 下位の文書・補完・AI Contextから非公開内容や存在を漏らさない | #951／#2516 |
| SM-42 | Studio表示・Plugin選択は成功、生成ArtifactにHookなし | Definition／Effective／Observedの差を示し、機能実装完了と扱わない | #4198／#2831 |

**次の実証順（AG系列：既存S0〜S5の独立した補助検証）**は次のとおり。

1. **AG-0：既存Owner Fit/Gap**：#4198の型付きGraph、#951のContribution/Dependency、#2497のProjectionと#4122の合成、#2605のSkill、#1124のSelectionの既存契約でSM-33〜42をどこまで書けるか調べる。
2. **AG-1：Headless、読取専用**：同じSourceからDefinition Graphの固定Sliceを抽出し、Page/Operation/Providerの別種Relationを識別。Graphの不完全性や同名別IDを診断する。
3. **AG-2：横断Selectorの限定検証**：合成の二Page＋一動的／未読込Entryに対し、公開PageのHook適用候補を生成。実Targetのタグ配信・Consentはまだ合格としない。
4. **AG-3：Plugin文書ナビゲーション**：二階層の既存Plugin依存を用いて、概要→Contract→依存先の説明／Sourceへ追跡。人間とAIで同じID／版・Disclosureを確認する。
5. **AG-4：Studio／AI選択連携**：共通UIの一要素を型付きSelectionとして取り出し、Source Jumpとreadonly診断を再現。候補Patchは§17の一意Slotに限定する。
6. **AG-5：Build／実Consumer**：既存#2831／#4583のStage2と#1251／#2586のProduct経路が成立した範囲で、実選択Plugin・生成Artifact・実Call／Consumer受領を独立照合する。

AG-0〜AG-4は**Studioセルフビルドと通常CLIの実Readという全体P0の新たな前提条件にしない**。既存の構造・型付き参照・Projection・CLI Inspectで実現できる範囲に留める。新しいUniversal App Graph Store、Graph Compiler、Aspect Engine、文書Registry、別のAI Skill Runtimeを先に導入しない。

**停止条件**：対象Universeが不明なのに「全Pageに適用済み」、Pluginの文書が存在するだけで「実Capability利用可」、合成図から「元関係を完全復元」、上位の説明を見たことで「現在のGrantあり」と主張する実装は、用途に応じて不適格／未検証へ戻す。
