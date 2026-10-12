# Definition-Centered Feedback Loop

状態：現在の設計方針。意味の正本の要素種別や新たな判定段階を追加するものではない。

## 1. 目的

TEIは、生成AIが大量の設計書やコードを生成すること自体を価値の中心にしません。

中心となる価値は、**人にも機械にも読める簡潔な定義を意味の正本として保ち、実行・可視化・検査・探索に必要な情報をそこから再生成し、現実から得た知見を意味の責任を持つ層へ戻せること**です。

```text
small authoritative definitions
        ↓ normalize / validate
Canonical meaning
        ↓ derive / compile / project
UI / matrix / graph / runtime / simulator / index
        ↓ use / execute / observe
human review / runtime result / external actuality
        ↓
feedback
        ↓ route to owning concern
candidate change
        ↓ preview / qualify
Shared Acceptance Boundary
        ↓ accepted minimum semantic delta
authoritative source
        ↓ reload / regenerate
        ↺
```

これはAIが自ら意味の正本を書き換える仕組みではありません。AI、認知用の索引、設計・編集環境、実行環境は、業務上の意味を確定する権限を持ちません。それぞれの役割から、定義の理解・検証・改善を支援します。

## 2. X-TEA / XEADから受け取るもの

TEIは、X-TEAやXEADとの互換製品を目指しません。ただし、従来の参照モデルと三要素分析法から、現在にも生かせる設計原理を取り入れます。

### 継承する原理

- 一つの業務システムを、データ・機能と操作・業務の異なる視点から照合する
- データを起点とするが、データだけで業務全体を説明しようとはしない
- 業務上の行為とアプリケーションの機能を区別する
- 人・手作業・外部組織・物理世界を含む社会システムとして業務を扱う
- 仕様は可能な範囲で実行・模擬し、文書だけで正しさを主張しない
- 参照モデルを単なる試験用データにせず、基盤とモデルを相互に改善する実践的な材料にする
- モデル上の相互参照から矛盾を見つけ、元の定義に戻って修正する

### そのまま継承しないもの

- 旧製品で使われた要素の種類や項目の構造
- すべての視点を対等な最上位の要素として扱うこと
- 一体化した大きなドライバー
- 画面・実行環境・モデラーを密結合させること
- 旧製品のファイル形式をTEIの意味の正本の形式にすること
- 参照モデルに含まれる設計の型を、無条件に正解として採用すること

TEIでは、外部モデルに項目が存在すること自体よりも、**その項目や視点が、どのような設計上の失敗を防いでいたか**を重視します。

## 3. Definition-centeredという意味

定義を中心とする設計とは、「すべてをTEIのDSLへ書き直す」という意味ではありません。

意味の正本となる情報源は、扱う関心事によって異なります。

```text
TEI-native business / semantic meaning -> .tei等のTEI-native source
native query meaning                  -> SQL
existing implementation meaning       -> Go / existing asset + explicit binding
provider/environment fact              -> provider-owned configuration / observation
```

同じ意味について複数の正本を暗黙に作らず、どの情報源が何を定義するかを説明できるようにします。

### 3.1 irreducible meaningだけをauthorする

人が手作業で維持するのは、安全に自動再生成できない意味に限ります。

```text
author explicitly
  business identity / relation / constraint / responsibility / exceptional intent
  provider or target choice when it is genuinely a choice
  manual / external work when it is business meaning

prefer derived
  standard CRUD
  ordinary list/detail/form projection
  cross-reference matrix
  dependency graph
  reverse index
  impact view
  search index
  cognitive representation
  generated target code
```

情報を派生して生成することは、情報を失うことではありません。元の定義と生成規則から、その内容をいつでも説明し、再構成できなければなりません。

### 3.2 progressive disclosure

最初から完成したモデルを求めません。

```text
Data / Relationで業務事実を表せる
  ↓ 必要な場合だけ
explicit Action / Capability / Validation
  ↓ 実現差が意味を持つ場合だけ
Application Pattern / Binding / Realization
  ↓ 責任・手作業・外部継続が必要な場合だけ
Business Procedure / Handoff
  ↓ domainで必要な場合だけ
time / decision / evidence detail
```

空欄を埋めるためだけに概念を追加しません。追加しなければ業務上の意味が失われるときに限り、必要な概念を増やします。

### 3.3 semantic densityとfalse compressionを区別する

定義は簡潔にしますが、短さそのものを評価指標にはしません。

```text
high semantic density
  = 少量のauthoringから、多くの正しいderived behaviorを再構成できる

false compression
  = 必要なBusiness meaningを暗黙推論へ追い出し、短く見せる
```

手作業、責任の所在、実際に起きたこと、例外などを省いて定義を短くしても、品質が高くなるわけではありません。

## 4. 三種類のmemoryを分ける

TEIでは、生成AIや認知技術を活用する場合も、すべての知識を一つの知識グラフやモデルの重みに集約しません。

### 4.1 authoritative semantic memory

長期に保持する、正式に受理された意味です。

- authoritative source
- Canonical identity / relation / constraint
- accepted Business meaning
- explicit Binding / Realization choice
- accepted Decision / policy where required

これらを失うと、製品が扱う意味を再構成できなくなります。

### 4.2 derived associative / structural memory

関連する情報を探し、近い事例を想起するために使います。

- exact index / dependency index
- lexical index
- graph neighborhood
- Reference Model pattern index
- embedding
- sparse high-dimensional representation
- HDC / VSA等のrole-aware representation
- candidate ranking / novelty signal

これらは**再生成でき、どの改訂版に対応するかが分かる**必要があります。

```text
association != dependency
similarity  != applicability
activation  != fact
novelty     != unknown
```

この種類の記憶を削除しても、正本の意味は変わりません。

### 4.3 episodic / experience memory

何を試し、どのように判断し、その後何が起きたかを記録します。

- Proposal / rejection / acceptance record
- Semantic Diff / impact preview
- simulator / runtime run
- Observation / Evidence
- review comment / rationale
- past correction trajectory

過去に成功した変更であっても、そのまま現在の意味として扱いません。現在の改訂版、入力元、検証根拠に照らして評価し直します。

### 4.4 memory間のauthority rule

```text
authoritative semantic memory
        ↑ accepted change only
Shared Acceptance Boundary
        ↑
proposal / experience / cognition

associative memory  -X-> implicit semantic write
episodic memory     -X-> implicit semantic write
model weights       -X-> only durable record of a Decision
```

AIや認知方式を交換・停止しても、受理済みのシステムを再構成できることを、変えてはならない条件とします。

## 5. 認知機能を一つのAIへ集約しない

TEIの認知支援は、性質の異なる複数の方式を組み合わせます。

```text
Intent / local editing context
        │
        ├─ exact identity / graph traversal
        ├─ Reference pattern recall
        ├─ associative / sparse cognition
        └─ LLM interpretation / hypothesis / explanation
                ↓
           candidate space
                ↓
 deterministic qualification
 applicable / incompatible / unknown
                ↓
       explicit-reference proposal
```

### Exact / graph

既知の識別情報、依存関係、権限を確定的にたどります。分かっていることを、毎回大規模言語モデルに推論させません。

### Reference pattern recall

実践的な参照例から得られた構造を、出所と適用条件を添えて候補として示します。設計の型は、そのまま正解になるわけではありません。

### Associative cognition

用語は異なっても構造が近い例や、断片的な手掛かり、局所的な新規性を候補として見つけます。FlyHash、HDC/VSA、埋め込み表現などは、この役割を担えるか比較する方式です。

### LLM

自然言語で表された意図の解釈、候補の比較、欠落の仮説、説明、案の統合に活用します。大規模言語モデルに入力元と検証根拠を与えても、その出力自体には確定権限を持たせません。

## 6. Feedbackを正しい所有層へroutingする

問題が見つかっても、すべてを意味の正本となる入力元に戻して修正するわけではありません。

| Feedback | 主な戻り先 | 戻してはいけない先の例 |
| --- | --- | --- |
| Business fact / relation / constraintの欠落 | Semantic / Business authoring source | UIだけの補修 |
| Business responsibility / manual workの欠落 | Business Procedure source | fake Application Function |
| Target固有のinteraction問題 | Presentation / Target Adapter | domain Dataの歪曲 |
| Provider固有の実現問題 | Binding / Realization / Provider | Canonical business meaning |
| DB actual drift | Observation / Mapping / Migration candidate | Semantic Relationへの自動逆書き |
| Runtime failure | Capability / Execution / Resource境界 | LLM promptだけの調整 |
| 検索取りこぼし / ranking問題 | derived index / cognition | Canonical relationの捏造 |
| Reference patternが合わない | candidate ranking / applicability rule | Reference patternの自動採用 |

どこを修正するかは、次の原則で判断します。

> 目に見える症状ではなく、意味を管理する最小の層を直す。

これにより、画面の修正でデータの誤りを隠したり、AIへの指示を調整して権限上の矛盾を隠したりするような、その場しのぎの対応を防ぎます。

## 7. Minimum Semantic Delta

生成AIを使う場合も、モデルやファイル全体を毎回作り直すのではなく、**現在受理されている意味について、必要最小限の変更を提案すること**を基本とします。

```text
base revision
   + purpose / observed problem
   + exact target identities
   + minimum proposed meaning change
   + provenance / evidence
        ↓
preview
   Semantic Diff
   affected subjects
   derived UI / runtime / persistence impact
   unresolved / contradiction
        ↓
Accept / Reject
```

最小限の意味の変更は、特定の修正用DSLを必須としません。ファイルの直接編集、設計・編集環境からの操作、構造化したAIの提案、構造の整理などが、同じ意味の差分として扱えればよいと考えます。

### full regenerationを標準にしない理由

- 人が確認する際に、重要な変更点が見えにくくなる
- モデル固有の書式や注記まで不要に書き換えやすい
- 古い改訂版との競合が広がりやすくなる
- AIが求められていない意味まで補ってしまいやすい
- 受理された意味と、生成された文章が混ざる

ただし、新しい成果物の生成や大規模な移行など、全体の生成が適した作業まで禁止しません。重要なのは、意味上の変更点を絞り込み、人が確認できることです。

## 8. Previewは生成物ではなく因果を見せる

定義を記述・編集する画面では、入力欄を増やすよりも、**一つの定義を変えると何に影響するか**を示すことを重視します。

例は次のとおりです。

```text
Add required Relation
  ├─ Canonical: relation-added
  ├─ Function: Action input requirement changes
  ├─ UI: reference selector appears
  ├─ Persistence: migration required
  ├─ Runtime: referential integrity enforced
  ├─ Business: impacted Procedure candidates
  └─ Target: affected projections
```

影響を表すそれぞれのつながりは、導出規則または確定した依存関係までたどれる必要があります。大規模言語モデルの説明だけを根拠に影響関係を作りません。

## 9. Local-first Editor

大規模なモデル全体を、常に一枚の関係図へ描く必要はありません。

利用者が扱っている定義を中心に、関連する近くの要素をまとめて示します。

```text
selected definition
  ├─ authored local meaning
  ├─ exact incoming / outgoing refs
  ├─ Data / Function / Business lenses
  ├─ relevant source / evidence
  ├─ affected projections
  ├─ candidate patterns / analogous cases
  └─ unresolved / diagnostics
```

モデル全体の関係図や表形式の一覧は、必要に応じて生成する表示です。検索、対象範囲、近くの関連要素を活用し、定義が数百・数千あっても把握しやすいようにします。

AIも同じ局所的な文脈を基本の入力とします。毎回リポジトリ全体を自由に探索し、その結果を出所の分からない暗黙知として扱う方法は標準としません。

## 10. Reference Modelを知的社会資本として扱う

参照モデルには、複数の役割があります。

1. **worked example** — 利用者が良い定義の形を学ぶ
2. **executable design content** — Driver / Simulatorで動かして理解する
3. **Conformance pressure** — toy fixtureでは出ない設計欠陥を露出させる
4. **pattern evidence corpus** — AI / cognitionが候補構造を想起する根拠
5. **product feedback source** — Referenceの要求がPlatformの過不足を明らかにする

特に重要なのは、五つ目の「基盤を改善するための情報源」という役割です。

```text
Platform -> Referenceを実行可能にする
Reference -> Platformの不足・過剰を暴く
```

両方向の改善につながらない参照モデルは、単なる実演や基準となる試験用データにとどまります。

### Reference patternの扱い

複数の参照モデルから共通する設計の型を見つけることは有用です。ただし、その型は次のように候補として扱います。

```text
intent
  ↓
multiple candidate patterns
  + provenance
  + observed frequency / examples
  + applicability conditions
  + semantic differences
  + optional Simulation Dataset
  ↓
qualification / review
```

「八つの参照例にあるから採用する」とは判断せず、現在の業務要求に合うかを別途確かめます。

## 11. Observationとlearningを分ける

実行環境、模擬実行、外部の現実から得た観測結果は、検証根拠として扱います。観測結果をオンライン学習によって直ちに意味の正本へ書き戻すことはしません。

```text
expected meaning
        ↕ compare
observed actuality
        ↓
contradiction / drift / unknown
        ↓
proposal or routing decision
        ↓
Shared Acceptance Boundary
```

### 学習可能なのはまずderived memory

参照モデルや実際の運用から得た経験を使って自動更新を検討する場合、まずは再生成可能な候補の生成や順位付けを対象とします。

- task-local association weight
- frequently useful candidate pattern
- false-positive suppression
- local retrieval hint

その場合も、改訂版、対象の作業、結果、出所を記録し、更新を無効にできるようにします。

受理済みの意味上の規則をモデルの重みだけに埋め込むオンライン学習は、標準的な方法にしません。

## 12. Feedback qualityの観点

現時点では数値目標として固定せず、設計の確認では次の点を評価します。

### semantic density

少ない記述から、どれだけ正しい意味と派生情報を再構成できるかを見ます。ただし、必要な意味を省略して短く見せることは避けます。

### derivation leverage

同じ内容を人手で二重管理せず、画面・実行環境・データベース・表・文書などへ、どれだけ正確に展開できるかを見ます。

### feedback closure

見つかった矛盾を、その意味を管理する入力元や層まで戻って修正できるかを見ます。

### reconstructability

索引、生成物、AIとのやり取りの履歴を削除しても、受理済みのシステムを再構成できるかを見ます。

### locality

個々の定義と、その周辺の関連要素だけで、安全な変更を判断できるかを見ます。

### provenance

候補、設計の型、AIによる説明、過去の経験について、入力元、改訂版、検証根拠までたどれるかを見ます。

これらは比較と確認の観点であり、すべてを一つの点数へまとめません。

## 13. Product invariants

今後AIや認知支援の機能を追加するときも、次の区別を守ります。

```text
Definition != Projection
Projection != Authority
Observation != Definition
Similarity != Applicability
Proposal != Decision
Autonomy != Authority
Reference Pattern != Requirement
Runtime Success != Business Completion
Generated Artifact != Source of Truth
```

また、一部の機能が使えない場合にも、次のように動作を継続できるようにします。

```text
LLM unavailable
  -> exact / graph / manual authoringでsystemを扱える

associative cognition unavailable
  -> exact / graph / lexical探索へ退化できる

Studio unavailable
  -> authoritative sourceを直接review / reloadできる

Derived index deleted
  -> Canonicalから再生成できる

Target removed
  -> Canonical business meaningは残る
```

この機能低下時の扱いは、AIや特定の画面が、見えない新たな正本になっていないか確かめるための設計上の検査です。

## 14. 現時点で固定しないもの

- permanentな`Cognitive Layer`というArchitecture層
- 一つのuniversal Knowledge Graph
- vector database
- FlyHash / HDC / VSA / embeddingの特定方式
- online learning
- Proposal / Patchのplatform-wide schema
- feedback categoryの永続enum
- AI provider / agent framework
- whole-model graph UI
- Reference patternの自動採用
- Reference Model file format

具体的な参照例で意味の欠落が確認された場合に限り、これらを固定する必要があるか再検討します。

## 15. 関連

- `docs/architecture.md`
- `docs/business-system-design.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- ISSUE-000046-studio-shared-acceptance-authoring-loop.md
- ISSUE-000047-knowledge-first-semantic-exploration.md
- ISSUE-000027-ai-proposal-acceptance-boundary.md
- ISSUE-000051-derived-cognitive-retrieval-evaluation.md
