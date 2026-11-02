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
