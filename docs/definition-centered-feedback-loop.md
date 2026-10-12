# 定義を中心にした改善の循環

状態：現行の設計方針。意味の正本に新たな概念種別や判定段階は追加しない

## 1. 目的

TEIは、生成AIが大量の設計書やコードを生成すること自体を価値の中心にしません。

中心に置くのは、**人にも機械にも読める簡潔な定義を意味の正本として保ち、そこから実行・可視化・検査・探索のための情報を再生成し、現実から得たフィードバックを意味の責任を持つ層へ戻せること**です。

```text
簡潔な正本の定義
        ↓ 正規化・検証
正本となる意味
        ↓ 導出・変換・表示
画面・対応表・関係図・実行基盤・模擬実行・索引
        ↓ 利用・実行・観測
人による確認・実行結果・外部で実際に起きたこと
        ↓
確認結果
        ↓ 問題を管理する部分へ振り分ける
変更候補
        ↓ 影響を表示・適格性を確認
共通の受理境界
        ↓ 受理された最小限の意味変更
正本となる情報源
        ↓ 読み直し・再生成
        ↺
```

この一連の流れは、AIが正本を自ら書き換える仕組みではありません。AI、認知用の索引、編集環境、実行基盤は、それぞれ異なる役割で定義の理解・検証・改善に関わります。いずれも業務上の意味を確定する権限は持ちません。

## 2. X-TEAとXEADから受け継ぐ考え方

TEIはX-TEAやXEADとの互換製品を目指しません。一方、過去の参照モデルと三要素分析法から、現在も有効な設計原理を取り入れます。

### 継承する原理

- 一つの業務システムを、データ、機能・操作、業務の三つの視点から相互に検証する
- データを起点にするが、データだけで設計を終えない
- 業務上の行為とアプリケーションの機能を同一視しない
- 人・手作業・外部組織・物理世界を含む社会システムとして業務を扱う
- 仕様を可能な範囲で実行・模擬実行し、文書だけで正しさを主張しない
- 参照モデルを単なる試験用データとせず、開発基盤とモデルを相互に改善する実例として使う
- モデル内の相互参照から矛盾を発見し、元の定義へ戻って修正する

### そのまま継承しないもの

- 旧製品に固有の対象種別や項目の構造
- 三つの視点を対称な最上位の対象として定義すること
- 一体化された大規模な実行・検証環境
- 画面、実行基盤、モデラーの密結合
- 旧製品のファイル形式をTEIの正本の形式にすること
- 参照モデルに含まれる構成パターンを無条件に正解として採用すること

TEIでは「外部モデルに項目があるか」よりも、**その項目や視点がどのような設計上の誤りを防いでいたか**を問い直します。

## 3. 定義を中心にするとはどういうことか

定義を中心にするといっても、すべてをTEI独自の定義言語で書き直すわけではありません。

何を定義するかによって、意味の正本となる情報源は異なる場合があります。

```text
TEI固有の業務・意味の定義         → .teiなどのTEI用ファイル
元形式で記述する問い合わせ         → SQL
既存の実装が持つ意味               → Go・既存資産と明示した対応付け
提供実装・環境についての事実       → 提供側の設定・観測結果
```

同じ意味について暗黙に複数の正本を作らず、どの情報源が何を定義しているか説明できることが重要です。

### 3.1 再生成できない意味だけを人が記述する

人が直接記述して維持する対象は、機械的に安全には再生成できない意味に限定します。

```text
明示して記述するもの
  業務上の識別・関係・制約・責任・例外時の意図
  実際に選択の余地がある提供実装・実行先の選択
  業務上の意味を持つ人の作業・外部での仕事

なるべく定義から導くもの
  標準的な登録・参照・更新・削除
  一覧・詳細・入力フォームの通常の表示
  相互参照の対応表
  依存関係の図
  逆引き索引
  変更の影響を示す表示
  検索索引
  認知処理のための表現
  実行先に応じて生成するコード
```

派生情報として生成することは、情報を捨てることではありません。元の定義と生成規則から、いつでも説明・再構成できる必要があります。

### 3.2 必要に応じて詳細を加える

最初から完成したモデルを要求しません。

```text
データと関係で業務上の事実を表す
  ↓ 必要な場合だけ
明示的な業務操作・機能・検証規則
  ↓ 実現方法の違いが意味を持つ場合だけ
アプリケーションの構成・対応付け・実現方法
  ↓ 責任・人の作業・外部での継続が必要な場合だけ
業務手順・引継ぎ
  ↓ 分野によって必要な場合だけ
時間・判断・検証根拠の詳細
```

概念は空欄を埋めるために追加せず、追加しなければ業務上の意味が失われる場合に限って増やします。

### 3.3 意味の密度と見かけだけの簡略化を区別する

定義を短くすることは目指します。ただし、その長さ自体を評価指標にはしません。

```text
意味の密度が高い
  = 少ない記述から、多くの正しい動作や表示を再構成できる

見かけだけの簡略化
  = 必要な業務上の意味を暗黙の推論に委ね、定義を短く見せる
```

人の作業、責任、現実に起きたこと、例外などを削って定義を短くしても、品質が向上したことにはなりません。

## 4. 三種類の記憶を分ける

TEIでは、すべての知識を一つの知識グラフやモデルの重みにまとめません。生成AIや認知技術を使う場合も同様です。

### 4.1 正本となる意味の記憶

受理された意味のうち、長期に保持する必要があるものです。

- authoritative source
- Canonical identity / relation / constraint
- accepted Business meaning
- explicit Binding / Realization choice
- accepted Decision / policy where required

これは削除するとプロダクトの意味が失われます。

### 4.2 派生した連想・構造の記憶

「何を思い出すか」「何が近いか」を助けます。

- exact index / dependency index
- lexical index
- graph neighborhood
- Reference Model pattern index
- embedding
- sparse high-dimensional representation
- HDC / VSA等のrole-aware representation
- candidate ranking / novelty signal

これらは**再生成可能で、対象のリビジョンに結び付いている**必要があります。

```text
連想は依存関係と同じではない
類似は適用可能性を意味しない
活性化は事実と同じではない
新規性は未知であることと同じではない
```

このmemoryを削除してもCanonical meaningは変わりません。

### 4.3 経験や履歴の記憶

「何を試し、どう判断し、何が起きたか」を保持します。

- Proposal / rejection / acceptance record
- Semantic Diff / impact preview
- simulator / runtime run
- Observation / Evidence
- review comment / rationale
- past correction trajectory

過去に成功した変更も、そのまま現在のmeaningにはなりません。current revision / Source / Evidenceへ戻って再評価します。

### 4.4 記憶ごとの権限の境界

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

AIや認知方式を交換・停止しても、accepted systemを再構成できることを不変条件とします。

## 5. 認知機能を一つのAIへ集約しない

TEIの認知は、性質の違う複数方式を協調させます。

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

### 正確な識別と関係の追跡

既知のidentity、dependency、authorityを確定的に辿ります。既知のことを毎回LLMに再推論させません。

### 参照事例から構成パターンを想起する

実践Referenceから観測された構造を、provenanceと適用条件つきの候補として提示します。patternは答えではありません。

### 連想による認知処理

語彙が違うが構造が近い例、partial cue、局所的なnoveltyを候補化します。FlyHash、HDC/VSA、embedding等はこの役割の比較候補です。

### 大規模言語モデル

自然言語intentの解釈、複数候補の比較、欠落仮説、説明、compositionに強みを使います。LLMにSource / Evidenceを与えても、その出力自体をauthorityにはしません。

## 6. 確認結果を修正すべき部分へ戻す

質の高いフィードバックの仕組みでは、見つかった問題をすべてSemantic sourceへ戻すわけではありません。

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

修正先を判断する原則は次のとおりです。

> 目に見える症状ではなく、意味を管理する最小の層を直す。

これにより「UIを直してDataの誤りを隠す」「AI promptを調整してauthority矛盾を隠す」といった局所最適を防ぎます。

## 7. Minimum Semantic Delta

生成AIを使う場合は、モデルやファイルを毎回丸ごと再生成するのではなく、**現在受理されている意味に対する最小限の変更を提案すること**を標準にします。

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

Minimum Semantic Deltaは固定Patch DSLを意味しません。file-native edit、Studio action、structured AI proposal、refactor等が同じ意味差分へ収束できればよいとします。

### full regenerationを標準にしない理由

- 人間reviewで本質的な変更点が見えにくくなる
- model固有のformatやcommentを不要に書き換えやすい
- stale conflictの範囲が広がる
- AIが未要求の意味まで補完しやすい
- accepted meaningとgenerated wordingが混ざる

ただし新規artifact生成や大規模migrationなど、whole-artifact generationが合理的な作業まで禁止しません。重要なのはsemantic diffを局所化してreviewできることです。

## 8. Previewは生成物ではなく因果を見せる

Authoring UXでは、入力formを増やすより、**一つの定義変更が何へ波及するか**を見せることを重視します。

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

各edgeはderived ruleまたはexact dependencyへ追跡できる必要があります。LLMの説明だけでimpactを作りません。

## 9. Local-first Editor

大規模modelを常に一枚のgraphへ描くことを要求しません。

利用者が現在扱う定義を中心に、局所近傍を高密度に見せます。

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

全体graphやmatrixは必要に応じたProjectionです。検索・scope・local neighborhoodにより、数百・数千definitionでも認知負荷を制御します。

AIも同じlocal contextを基本入力とし、Repository全体を毎回自由探索して暗黙知化することを標準にしません。

## 10. Reference Modelを知的社会資本として扱う

Reference Modelには複数の役割があります。

1. **worked example** — 利用者が良い定義の形を学ぶ
2. **executable design content** — Driver / Simulatorで動かして理解する
3. **Conformance pressure** — toy fixtureでは出ない設計欠陥を露出させる
4. **pattern evidence corpus** — AI / cognitionが候補構造を想起する根拠
5. **product feedback source** — Referenceの要求がPlatformの過不足を明らかにする

重要なのは5です。

```text
Platform -> Referenceを実行可能にする
Reference -> Platformの不足・過剰を暴く
```

この双方向性がないReferenceは、単なるdemoやgolden fixtureへ縮退します。

### Reference patternの扱い

Reference群から共通patternを抽出することは有用ですが、patternは次の形で利用します。

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

「8個のReferenceにあるから採用」ではなく、現在のBusiness requirementへ合うかを別途確認します。

## 11. Observationとlearningを分ける

Runtime / simulator / external worldからのObservationはEvidenceです。Observationそのものをonline learningで即時にsemantic writeへ変換しません。

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

Referenceや運用から得た経験で自動更新を検討する場合、最初に対象にするのは再生成可能な候補生成・ranking側です。

- task-local association weight
- frequently useful candidate pattern
- false-positive suppression
- local retrieval hint

それでもrevision / task / outcome / provenanceを持ち、無効化可能にします。

accepted semantic ruleをmodel weightだけへ埋め込むonline learningは標準経路にしません。

## 12. Feedback qualityの観点

現時点で数値KPIへ固定しませんが、設計reviewでは次を観察します。

### semantic density

少量のauthoringでどれだけ正しい意味・Projectionを再構成できるか。ただしfalse compressionを避ける。

### derivation leverage

同じ内容の人手二重管理なしに、UI / Runtime / DB / matrix / docs等へどれだけ正しく派生できるか。

### feedback closure

見つかった矛盾を、その意味を所有するsource / layerへ戻して直せるか。

### reconstructability

index、generated artifact、AI historyを削除してもaccepted systemを再構成できるか。

### locality

局所definitionと近傍だけで安全な変更判断が可能か。

### provenance

candidate / pattern / AI explanation / experienceが、Source / revision / Evidenceへ戻れるか。

これらは比較・review観点であり、達成率を一つのscoreへ畳みません。

## 13. Product invariants

今後のAI・認知機能追加でも次を守ります。

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

さらに、次のdegradationを許容します。

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

このdegradabilityは、AIや特定UIを新しい不可視の正本にしないための設計検査です。

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

具体的Referenceで意味が失われる場合だけ固定化をreopenします。

## 15. 関連

- `docs/architecture.md`
- `docs/business-system-design.md`
- `requirements/TEI_PRODUCT_REQUIREMENTS.md`
- `plan/adr/0022-ai-proposal-canonical-acceptance-boundary.md`
- ISSUE-000046-studio-shared-acceptance-authoring-loop.md
- ISSUE-000047-knowledge-first-semantic-exploration.md
- ISSUE-000027-ai-proposal-acceptance-boundary.md
- ISSUE-000051-derived-cognitive-retrieval-evaluation.md
