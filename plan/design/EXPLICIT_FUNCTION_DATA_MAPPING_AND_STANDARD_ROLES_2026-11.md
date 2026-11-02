# Function別の明示Data Mappingと標準構造へのRole適用：Headless検証（2026-11）

**状態**：設計・隔離Reference。正式TEI Data Context/DSL/Studio/Providerへの組込み、実DBの取得範囲証明、Owner意味受理は**未実施**。以下の`Mapping`・`read_explicit`は合成試験専用であり、新しいTEI共通Schema／Query Planner／権限Engineを提案するものではない。

**上位方針**：[#2619](https://github.com/hat47x/tei-platform/issues/2619)（Everything is Data横断）、[#2620](https://github.com/hat47x/tei-platform/issues/2620)（標準構造・意味Role）、[#2621](https://github.com/hat47x/tei-platform/issues/2621)（Markdown/表形式/Studio authoring）、[#2622](https://github.com/hat47x/tei-platform/issues/2622)（Data Operations・Derived Dataset）。**受入Owner**：[#2506](https://github.com/hat47x/tei-platform/issues/2506)（Function Data Context Mapping/Result Slice）。Operation/Providerの正本は#1127/#2485/#1129/#2500/#2598/#1174へ戻す。別の実装を複製しない。

## 1. 成熟した横断設計判断

**データモデルがRelationを定義することと、FunctionがそのRelationを利用することを別の宣言とする。**

モデルには`Order.lines`、`Line.notes`、`Order.payments`があってよい。一方、あるFunctionが実際に使用するのは`Order.id/customer`、`Order.lines.id/amount`、`Order.lines.notes.id/text`に限ることができる。未宣言の`Order.payments`や`private_note/internal_flag/sku`はそのFunctionの**アクセス要求の一部にはならない**。

ここで**構造上の取得範囲（read footprint）**と、**結果の母集団・意味的完全性（claim closure）**を区別する。

- **Read footprint**：Functionが明示した起点、Relation Path、Field、読み取り範囲、対象粒度。定義に存在するRelationを自動的に全展開しない。
- **Claim closure**：当該Claim/Actionが必要とする対象集合・時点・版・値域/NULL・欠損／重複/非単調演算の成立根拠。Functionが「全件を要求した」こととProviderが「全件を実際に返した」ことは独立。
- **Execution/disclosure qualification**：対象へ実アクセスできるか、同じ主体がそのデータやAction候補を他Consumerへ出してよいかは、実Host/Providerと開示契約に依存。正しくMappingできたことは権限を意味しない。

こうした明示Mappingは、単なる性能最適化でなく、生成AIがFunction定義の利用データを把握して不要な経路・秘密Fieldの取得を**要求しない設計**を支援する。ただし実Providerの過剰取得・Cache/ログ/外部送信の防止は**実行経路の独立検査**が必要（#2516/#2515）。

## 2. 全定義体を標準データ構造として表す

`Function`、`Data Model`、`Access Map`、`Projection`、`Rule`、`Test/Expected`を、Record/Sequence/Relation/Expression/参照の組合せで表現し、それぞれの**意味RoleとOperator/Plugin ID・版**によって解釈する。

概念的な標準Record（正式構文・Schemaではない）：

```json
{
  "id": "example.function.order.read",
  "role": {"ref": "example.role.function-data-context", "version": "fixed-example-v1"},
  "root": {"semantic": "Order", "binding": "selected-order"},
  "map": [
    {"id": "order", "path": "Order", "fields": ["id", "customer"], "scope": "selected"},
    {"id": "lines", "path": "Order.lines", "fields": ["id", "amount"], "scope": "all_for_selected"},
    {"id": "notes", "path": "Order.lines.notes", "fields": ["id", "text"], "scope": "all_for_selected"}
  ]
}
```

対応するMarkdown表（Authoringの例）：

| Context | Path | Fields | Scope |
| --- | --- | --- | --- |
| order | Order | id,customer | selected |
| lines | Order.lines | id,amount | all_for_selected |
| notes | Order.lines.notes | id,text | all_for_selected |

**表もRecordも同じ構造への異なる入力表現**。Markdownの見出し／列位置／装飾が自動的にOwner受理済みの業務意味になるわけではない。解釈Pluginの版、Role、Stable ID、Source locator/SpanとMeaning revisionを明示し、曖昧な表・非対応Roleは候補・診断にとどめる（#2620/#2621/#988）。Studioは標準の表／Tree／Graph／式／Instance UI上で候補補完と検証を行い、採用済みMappingへ明示反映する（#1124）。専用Editorや第二Canonicalの増殖を必要条件としない。

## 3. 取得と意味の成立条件

| 項目 | 期待する振る舞い | 負例 |
| --- | --- | --- |
| **Context Identity** | 同じFunctionで独立Context IDを持つ。Function Instanceは配置・選択epochを区別 | 同じ表示行やラベルから業務IDを復元 |
| **Relation Path** | 明示された意味Relation/Roleを検証し、Query上のJOIN/CTE演算と別に扱う | JOIN出現件数を業務Entity件数に昇格 |
| **Field selection** | 明示されたFieldだけを利用・公開する構成 | モデルに存在する秘密Fieldを自動全取得 |
| **Parent Binding** | 必要な親の選択／Scope/Occurrenceを結合し、曖昧な複数Hopを診断 | 子・孫へ曖昧な行Indexを利用 |
| **Scope declaration** | page/selected/all-for-parent等を明示し、暗黙全件を避ける | 範囲省略から全件保証を推定 |
| **Observed completeness** | Provider/固定Fixtureから別に受け取った根拠がある場合に限定し、Unknown/Partial/Completeを区別 | `all_for_selected`という宣言だけで実取得全件と確定 |
| **Derived Dataset** | 3層のNoteを2層の要約欄へ畳み、grain/lineage/不可逆性を表す | 集約文字列を元の一件のNoteへ直接PATCH |
| **Target Binding** | 明示Mappingと実Data Source/Provider/Queryの版・対象Scopeを対応付ける | Semantic Path一致でDB alias/tenant/read replicaを推測 |
| **Consumer disclosure** | 開示可能なField/Action候補・説明を別途選別 | Mapping済みなら外部AIへすべて送信可能と推測 |

### 3.1 意味上の「読み取り契約」とProviderの物理要求は別

同一意味MapをRDBのSQL、OData、REST、Graph、ファイル等にLoweringするとき、**各Providerで実際に要求した列／Relation／Filter／Pagination／Snapshot、実際に返却された集合・エラー・応答版**を別のEvidenceへ結合する。特に、API側が過剰取得しかサポートしない場合、意味上のフィールド限定・外部開示の限定は保てても、**Providerへの物理アクセス量が同等という主張はできない**。Capability不足は#2622で診断し、無音の保証低下を避ける。

Query Source/Workerの解析からのLineageだけでは、RuntimeのRLS/search_path、read replicaの鮮度、Query結果の実Scopeを証明しない（#2387/#2611）。`Data Context Instance / Result Slice / Query revision / Provider binding / actual read`の対応が観測できなければ、不明として保持する。

### 3.2 Derived 3層→2層と書込み

`Order -> Line -> Note`のNote集合をLineの1セルへ結合するようなProjectionは、読み取り専用の値のほか、元のNote ID/出現/取得完全性を必要な用途へ参照できるようにする。**値文字列が同じでも元の集合が同じとは限らない。** 編集から元Noteへの一意な逆対応がない場合、`ViewChange -> typed Intent`はread-only/ambiguous/unsupportedを返し、Ownerが受理した明示Actionがある場合だけ更新候補として扱う（#2500）。保存のAtomicity、部分成功、再試行は#2597/#1174へ委譲する。

## 4. 隔離Reference試験と証拠

`plan/fixtures/function-access-map/test_explicit_mapping.py`は**Python 3.13.5＋標準ライブラリのみ**の自作・架空データReference。実行したファイルの状態：

- `python3 test_explicit_mapping.py`：**20/20 PASS**。
- `python3 -m py_compile test_explicit_mapping.py`：**PASS**。
- **9838 bytes**、SHA-256 `6cf199ac600dfdb01dfe822137ac5f7adfab36d10fe7c7217b63ff2fc72b84ba`、Git blob SHA-1 `e26bdc5406339c177393470d177366834bdd9684`。GitHub保存Blobと実行ファイルが一致。

| 試験 | 意味 |
| --- | --- |
| EM-01 | 厳格に限定したMarkdown表→標準Record Listが一致 |
| EM-02〜05 | 未宣言Relation/Fieldや別Functionが使わない経路を結果から除き、明示Hopを取得 |
| EM-06〜09 | 未定義Relation/Field、Context重複、親Context欠落の負例 |
| EM-10〜11 | page取得はpartial、3層→2層要約は実観測complete根拠付きの読取り専用 |
| EM-12〜13 | Mappingが妥当でもHost許可や実Provider Bindingは証明しない |
| EM-14〜15 | Markdown列曖昧／取得Scope省略を拒否 |
| EM-16〜17 | **取得要求が全件でも実観測根拠なければUnknown**、同じFunction内のContext別complete/unknownを区別 |
| EM-18〜20 | 未宣言秘密Field・別Relationだけが異なる二世界で投影が同一。選択済みFieldの値が異なると出力が変わる |

**正確な到達範囲**：自作データモデルをPython辞書で探索し、宣言されたField/Relationを抽出しただけ。実TEI World/Plugin/Query Worker/Provider呼出し、Studio/SHO/Java/.NET/外部Consumer、実RLS/権限・情報流、実DBへの過剰アクセス、Owner受理は**NOT RUN**。試料中の`observed_complete`は固定された信頼済みFixture入力で、実Providerの全件性や正しい母集団を証明しない。実環境には既存Data ContextのScope/Claim資格を適用する必要がある。


### 4.1 自作二世界の感度・非感度の意味

`EM-18`では未宣言の`private_note/internal_flag`だけを変更、`EM-19`では未宣言`Order.payments`にレコードを追加する。どちらも**限定Referenceの投影結果が同一**であることを確認。`EM-20`は明示された`Order.lines.amount`だけを変更した二世界で、逆に投影結果が異なることを確認する。これは「明示Mapで選んだ対象だけがこの合成Projectionの情報依存となる」という有限の感度試験である。

**この局所二世界試験は、実DB/APIで未選択データを物理的に取得していないことや、実RLS・Error/Status・Timing・反復問い合わせ・Providerログ・AI egressの非干渉を証明しない。** さらに、Content Projectionの同一性はSource/Meaning Revision/実行Grantの同一性を保証しない。実Provider経路とConsumer開示の試験は#2622/#2516/#2515へ引き渡す。

## 5. 次の段階への接続計画

1. **A0：既存Source/契約の同一版調査**。#2506/1126の型・Relation/IDと#988のOperator、#2387のQuery、#1127/#2485の標準Read、#2622のProvider Capability、#2620/#2621のRole/Markdownを固定版で比較し、Source→Mapping→参照先がどこまで一対一か記録。
2. **A1：Headless read-onlyの小さな縦切り**。`Order/Line/Note/Payment`の同じ合成Sourceを既存TEI解析・Data Contextへ読み込み、`Order.payments`・秘密Fieldが暗黙取得・Projectionされないことと、明示したLine/NoteのScopeを結果から検証。Fixtureの`read_explicit`を本番Providerの実装として移植することは前提にしない。
3. **A2：Studio/Markdownの同一構造**。Markdown表、標準Record、Studio標準表/Graphからの編集候補が、安定ID・Role/Operator版・元Source spanへ帰属するか確認。曖昧な編集・Role未提供・循環・Source driftを負例にする。
4. **A3：Derived/Actionと二つのProvider**。完全性が分かったLine→Noteの2層表示と、部分・未取得／非可逆編集を比較。現存のViewChange→Intent/標準OperationとRDB・API系ProviderのCapabilityを各Ownerで確認。新Host、中央DB、業務Action正本を作らない。
5. **A4：異質Domainと生産性**。業務データ以外にもGame World、組織、文書などで、FunctionごとのAccess Mapが利用できるか圧力を追加。Core特殊型・専用Editor・UI固有業務Rulesの追加数、明示Map定義量、生成AI補完の必要修正、実過剰取得・誤実行を測定（#2018/#2493）。

採否は**実TEI経路が新しい意味正本・専用パーサ・独自Hostなしで成立するか**を基準とする。共通A/G/Roleから導出できるものは既存Ownerへ戻し、差分を最小のConformanceへする。Draft/局所PASSだけでMeaning受理・実Provider適合・情報保護の保証へ昇格させない。

## 6. 2026-11-02追補：表示MappingとRule評価依存を分離する（RF-01〜18）

### 6.1 前節のRead Footprintの厳密化

これまでのEM-01〜20は、**Functionに明示された表示／読取りデータの構成**に含まれないField・Relationを、合成Projectionに自動追加しないことを確認した。一方、業務上の保存・操作可否を判断するためには、表示していないデータを**別のRuleが正当に必要とする**場合がある。この境界を整理すると、明示Mappingだけで全データ依存を閉じたものとみなすことはできない。

例：受注金額`Order.amount=600`を表示するFunctionが、受理済み`rule.order.credit`で`Account.credit_limit`（与信上限）を参照して操作候補のValidation結果を求める場合。画面は上限値を表示しなくてもよいが、評価するRuleにはその値または既存Host/Providerの信頼できる評価結果が必要になる。**表示のRead footprint**と、**ValidationのRule dependency footprint**を別の参照として保存し、実Providerがどこでどの入力を処理したかの資格を確認する。

| 独立した観点 | 何を含むか | 依存・Owner |
| --- | --- | --- |
| **Display / Function Context Map** | Functionが選択して表示／変換へ利用するSemantic Path、Field、Scope、Grain | #2506/#2620/#2621 |
| **Rule Dependency Footprint** | Ruleの定義と適用先から導かれる必要入力、Rule/Operator版、評価時点、入力集合の完全性、必要な派生Rule | #1129/#1729/#1720/#1637/#1708 |
| **Target Read / Execution Binding** | 実Providerに要求した物理Input、取得の真のScope/版、Provider側の隠された計算、現在の実Authority | #2622/#2387/#2611/#1174 |
| **Action Write / Effect Footprint** | どのEntity/Relation/FieldへどのOperation/Effectが及ぶか、Transaction/Result | #2500/#2597/#2598/#1129/#1174 |
| **Consumer Disclosure** | 誰にどのField／Action名／Rule結果・理由を公開できるか。値と**判定結果自体**を別に開示判定 | #2516/#1178/#2515 |

**機能が宣言した表示範囲をRule入力のために暗黙拡大することはしない。** 必要なRuleはそのSourceで参照される既存の受理済みRule Definition/Attachmentから解決し、追加の入力依存を明示する。生の秘密入力をClientやAIへ返さなくても、Host側の既存ProviderでValidationを評価して**開示資格のある判定結果のみ**を返す構成を検証する。

したがって、物理的な取得対象、評価可能性、表示可能性、実行許可を一つの`allowed`へまとめない。Rule入力の取得や評価を許可できなければ`unresolved/not_established`であり、「業務上のValidation不成立」とは違う。Ruleが純粋評価であっても、その評価が可能なことは書込みActionのGrantやCommit成功を意味しない。

### 6.2 同じ画面、異なる世界の三つの対照

1. **RF-04：同じ表示と異なるRule結果**。`Order.amount=600`は同じでも、与信上限700なら合成Ruleは成立、上限500なら不成立。**表示Projectionの同値≠操作前提/Validationの同値**。
2. **RF-05〜07/10〜13：Rule入力の品質**。与信上限が未提供、利用資格不足、部分観測、Rule版/入力版不一致、読取りepoch不一致、Scope不一致、NULL方針不足なら、合成Validatorは`unresolved`にとどめる。`not_satisfied`へ無音変換しない。
3. **RF-14/15：判定結果も開示対象**。上限値を出さなくても「与信不足」という判定が秘密上限の情報を推測させることがある。結果の開示が許可されないConsumerには、異なる二世界でも同じ`unavailable`を出す。開示が許可されたConsumerに限定して、合成Previewの成立／不成立を出せる。

この対照から、#2582のReactive Invalidationで監視する必要がある依存も明らかになる。未表示の`Account.credit_limit`が更新されれば、受注金額の表示値は再利用可能でも**Validation/Action候補の資格は再評価対象**。未使用の`Order.payments`が変化しただけなら、そのRule/Contextに根拠を持たない失効を無条件に広げない。実依存Graph・Invalidation Plannerは#2582/#1641が所有し、本PRに第二CacheやObserverは設けない。

### 6.3 自作Fixtureと実測の限定範囲

`plan/fixtures/function-access-map/test_rule_footprint.py` を追加し、Python標準ライブラリだけで**RF-01〜18の18/18 PASS**、`python3 -m py_compile` **PASS**を隔離コンテナで実測した。

- 描画対象の`Order.amount`と、評価にだけ必要な`Account.credit_limit`を独立宣言として照合。
- 合成Validationの`satisfied / not satisfied / unresolved`を区別し、入力欠損・現行資格不足の表現・NULL/版/epoch/Scope不一致・重複・未知Rule意味の負例を固定。
- `after_commit`のEffect Ruleを純粋Validationとして扱わず、局所成立を実行Grantへ昇格しない。
- 同じ公開表示でも秘密Rule入力の変化によりValidationが変わる反例、および**判定結果そのものの開示許可**によるConsumer向け出力差を観測。

**exact tested file**：7969 bytes／SHA-256 `ab19fea1d2097b9451776d05431092c276bf1f3b89a56868dd4d97fe775f651e`／Git blob `a7b028dff2c0fb3d1859fd67366e4179639c8083`（GitHubに保存したBlobと一致）。

`Rule.accepted`、`Evidence.permitted_for_rule`、`allowed_result`、`complete`、`revision`、`epoch`は**すべて合成Fixtureの信頼済み仮定**。正式なTEI Owner Rule受理、実Provider Authority、複数実DB snapshotの同一性、実行時Data/Rule版適合を保証しない。今回の同一`epoch`/同一数値`revision`検査は**限定Fixtureの保守的な比較方法**であって、製品全体がRule版とData版の数値一致を要求する設計ではない。実ReaderのSnapshot整合は#1708、Rule入力→Validationは#1729、Effectの真正性は#1174に戻す。

**既存EM-01〜20との合計は独立Python合成38件PASS**。実TEIのFunction Context→Rule Attachment→Validation→Driver/Host→独立Read/SHO/AI Consumerを同一試行で接続した試験は**NOT RUN**。実行した二つのReferenceのPASS数を製品の受入数として合算しない。

### 6.4 開発計画への差分

- **A0 read-only照合**：既存#2506 Data Context Mappingと#1129 Rule Definition/Attachment／#1729 Validationの間で、実際に取り出せる入力依存・適用先・版・Scopeを一枚の受渡し票へ整理する。画面表示FieldだけでRule全体のRead Footprintとみなさない。
- **A1 Headless縦断**：同一`Order.amount`から、与信上限の入力欠損・不許可・変更・版ずれ・完全性不足を含むRule結果を、正式TEI Rule/Context/Validationの既存実装を通して試す（独立Synthetic Referenceをそのまま製品へ移植することを合格条件にしない）。
- **A2 Provider/Consumer**：Server側で評価し、上限値は非公開のまま結果だけを限定開示する構成を、既存Hostと二Consumerで検証。`not_satisfied`が推測する情報について現行Disclosure Policy・Authorityを検査する。
- **A3 Action/Effect**：未保存Previewで成立しても、保存時点の実Value/Rule/Grantが異なる場合、Owner Action側で再評価・Expected State/Commit Fenceを適用する。#2597のSave Sessionと#1174の実Effect・Receiptを再実装しない。
- **A4 Cross-Domain**：同型の依存を、組織Membership Rule、センサー閾値、ゲームWorldの認知・行動判断、文書Lintへ展開し、Role/Rule/Contextの再利用率、追加Core数、UI専用Business Ruleの削減を#2018で測る。
