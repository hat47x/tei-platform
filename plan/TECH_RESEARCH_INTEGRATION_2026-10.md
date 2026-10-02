# TEI 技術リサーチ統合方針 — 2026-10

Status: **current research-integration plan**  
Last updated: **2026-10-02**

## 1. 位置づけ

この文書は、2026-09末〜10月初頭の技術リサーチを、TEIのProduct Value / Requirements / existing Issueへ翻訳する現在の計画正本です。

外部研究、軽量認知方式、LLM、Wasm、Desktop framework等をTEIの仕様根拠そのものにはしません。取り込むのは、TEIの中心価値に対する不確実性を減らす境界・比較軸・Reference pressureです。

順序は維持します。

~~~text
research / implementation evidence
        ↓
Product Valueとの照合
        ↓
existing Requirement / Issueへ割当
        ↓
small Reference / pressure
        ↓
TEI自身のEvidence
        ↓
ADR / Requirement / Milestoneを更新
~~~

研究を読んだこと自体でEvidence tierを上げません。

## 2. Product Valueとの対応

直近の知見は、新しいProduct Valueを追加するより、既存価値を次の方向へ具体化します。

| Product Value上の価値 | 今回の統合方針 |
| --- | --- |
| 業務の意味を一つのSSOTへ保つ | AI / cognition / model outputはProposal・candidate・derived knowledgeに限定し、Canonical authorityへ直接昇格させない |
| 人の確認を実質的にする | whole regenerationよりminimum semantic delta、impact、unresolved、provenanceを優先し、決定論的検査でreview対象を狭める |
| 技術選択の自由を保つ | Wails / WebView / Wasm / native / model runtimeをHost・Adapter・execution profileへ隔離し、meaningと分離する |
| AI活用の上限を上げる | exact / deterministicで解ける仕事をAIへ送らず、曖昧さが残る場合だけ軽量認知→local model→external modelへ段階昇格する |
| 再利用と再入場を容易にする | candidate生成根拠、accepted Decision、contract / implementation / config / policyのrevisionを再現可能なEvidenceへ結び付ける |
| 説明責任を保つ | Proposal、Decision、execution authority、Command acknowledgement、Observationを別状態として追跡する |
| 一人でも維持可能にする | provider / modelごとの専用subsystemを増やさず、同一のcandidate / qualification / acceptance contractへ収束させる |

## 3. 直近リサーチから固定する設計仮説

### 3.1 認知方式は段階昇格する

Knowledge / AI laneの標準順序を次の仮説でPressureします。

~~~text
M0  explicit / exact structure
      ↓ unresolved only
M1  deterministic inference / graph / rule
      ↓ unresolved only
M2  lightweight associative / sparse / statistical candidate generation
      ↓ unresolved only
M3  bounded local cognitive model
      ↓ unresolved only
M4  local SLM / LLM
      ↓ unresolved only
M5  external LLM
~~~

M2〜M5は候補生成・ranking・novelty・abstention等へ利用できますが、Semantic applicability、Acceptance、write authority、Runtime authorityを所有しません。

段階を上げる理由と、下位段階で解けなかったunresolvedをEvidenceへ残します。

### 3.2 epistemic stateとmechanismを分ける

「何を知っているか」と「どの方式で得たか」を一つのconfidence値へ畳みません。

候補として次を直交させます。

~~~text
epistemic state
  observed / inferred / confirmed / abstained

mechanism
  deterministic / statistical / provider-dependent
~~~

名称やproduction schemaはReference後に判断します。重要なのは、model confidenceやsimilarity scoreをauthorityへ短絡しないことです。

### 3.3 軽量認知はbounded decision layerとして扱う

Jev系、FlyHash / sparse associative方式、embedding、LightGBM等は、LLM代替の万能reasonerではなく、次の限定用途で比較します。

- candidate shortlist
- ranking
- novelty / mismatch detection
- known false-positive suppression
- escalation / abstention signal
- local routing

その結果は必ずcurrent Source / Canonical / Evidenceへ戻してdeterministic qualificationします。

### 3.4 Local model runtimeは任意Providerとして扱う

Ollama、LM Studio等の既存local runtimeを利用できる場合は再利用候補にしますが、TEI Coreの必須依存にはしません。

model asset、runtime、revision、configはmeaningと分離し、再現性が必要なEvidenceでは利用したprovider / model / revision / configを記録します。

model bytesやruntimeが無くても、Canonical Worldと決定論的なValidation / Acceptanceが成立することを維持します。

### 3.5 portable executionはTargetと同一視しない

WasmはWeb等のPresentation Targetそのものではありません。

~~~text
Canonical / Semantic Operation
        ↓
deterministic typed realization
        ↓
native / Wasm / external process 等
        ↓
Target / Host Adapter
        ↓
Execution / Observation
~~~

Hikeのような極小Wasm系実装は「小さいportable execution artifact」のpressure sourceとして扱いますが、TEIのGo Coreを置換する前提にはしません。Go / TinyGo / Hike-like subset等を固定標準にせず、artifact form / ABI / host imports / resource conditionを比較可能にします。

### 3.6 Hostは意味を所有しない

Wails / WebView / browser / desktop lifecycleのreadiness、renderer recovery、request cancellation等はHost / Adapterの責務です。

これらの失敗をBusiness Outcomeへ短絡せず、Host observation / diagnosticとして保持します。Studio Wails HostとGenerated Desktop Targetも引き続き分離します。

### 3.7 実行可能性はcontractとcurrent authorityへbindする

AI / Rule / Humanのどのpolicyから来ても、effectful invocationは同じvalidated semantic actionへ収束させます。

再現・監査が必要なReferenceでは、少なくとも次を別々に識別できるかPressureします。

- semantic operation / contract revision
- implementation revision / digest
- config / binding revision
- current policy / authority snapshot
- Resource resolution
- execution result
- Observation / Evidence

過去のacceptanceやmodel memoryだけをfuture authorityにしません。

## 4. Existing Issueへの割当

### ISSUE-000047 — knowledge-first Semantic exploration

最優先で、read-only explorationに段階昇格を適用します。

1. exact identity / Source / graph neighborhoodをbaselineにする
2. deterministic inferenceで候補を増やす
3. lightweight associative / statistical方式をcandidate generatorとして比較する
4. local / external modelは、下位方式でunresolvedが残るcaseだけへ利用する
5. 最終的な applicable / incompatible / unknown はdeterministic qualificationで決める

評価では単純なtop-k精度だけでなく、次を分けます。

- useful candidate discovery
- false positive / incompatible
- unknown / abstention
- additional confirmation cost
- stale / provenance failure
- model / provider依存性
- latency / artifact size / runtime dependency

### ISSUE-000027 — AI Proposal / Shared Acceptance

Proposal生成方式をauthorityから分離したまま、上記の段階昇格からcandidateを受け取れるようにします。

ReferenceではProposalに、少なくとも次を追跡できるか確認します。

- candidate origin / mechanism class
- baseModelRevision
- exact target identity
- minimum semantic delta
- supporting Source / Evidence
- unresolved / abstained
- model / provider情報（利用した場合）
- acceptance前後のdeterministic Validation / Conformance

confidenceやmodel scoreからAcceptを自動導出しません。

### ISSUE-000026 — Second Target / portable execution pressure

第二Target選定では、Target Capabilityだけでなく、同じSemantic Operationを異なるexecution artifact / Hostへ落としたときの差を観測します。

第二TargetをWasmに固定はしません。選定時に次を比較軸へ含めます。

- native / portable artifactの交換可能性
- ABI / Host import差
- startup / recovery / resource condition
- semantic result / Observationの一致
- Target固有差をCanonicalへ逆流させないこと

### ISSUE-000053 — AI-native re-entry / Operational Knowledge

Builder / Agentが再入場するとき、chat historyやprovider memoryではなく、current accepted meaningとEvidenceから必要contextを再構成できるかをPressureします。

特に、実行候補について次を区別します。

~~~text
known operation
  != currently applicable
  != currently authorized
  != resource-ready
  != invoked
  != observed complete
~~~

context packetへmodel固有stateを必須化せず、contract / implementation / config / policyのrevisionを追えることを優先します。

## 5. Studio / Human Reviewへの含意

RQ-010 / RQ-028では、AIの自然言語要約だけをreview surfaceにしません。

高密度Projectionで少なくとも次を分けて表示できることを今後のReference条件へ入れます。

- authoritative / observed fact
- inferred candidate
- confirmed / accepted Decision
- unresolved / abstained
- source / evidence / impact drill-down
- provider / model由来の補助情報

人間はscoreそのものを承認するのではなく、意味差分と根拠を承認します。

## 6. 実装順

新しい横断subsystemやGateは追加しません。既存Issueを次の順で深めます。

1. ISSUE-000047でexact / deterministic baselineと段階昇格のReference条件を固定する
2. lightweight cognitionをcandidate generationへ限定して比較する
3. ISSUE-000027でcandidateからminimum semantic delta Proposalへ接続し、Shared Acceptanceを通す
4. ISSUE-000026でportable artifact / Host差を第二Target比較へ含める
5. ISSUE-000053でre-entry / operational contextへexecution provenanceを接続する
6. Studio / RQ-028で人が意味差分・unknown・provenanceを理解できるかactual Evidenceを取る

この順序は完全な直列ではありません。1〜3と第二Target pressureは並行できますが、上位方式の採用を下位方式のbaselineより先に固定しません。

## 7. 非目標

- Jev、FlyHash、embedding、LightGBM、特定LLMをTEI標準reasonerにすること
- Ollama / LM Studioを必須Runtimeにすること
- Hike / TinyGo / WasmをTEI Coreの必須実装にすること
- model confidenceをSemantic Evidence / Acceptance / authorityにすること
- Studio / Agent専用の第二Canonical Storeを作ること
- providerごとに別Acceptance / Permission semanticsを作ること
- research benchmarkのscoreをTEIの固定KPIへ輸入すること
- 研究だけを根拠にRQのEvidence tierを上げること

## 8. 完了条件

この研究統合は、文書作成だけでは完了扱いにしません。

少なくとも既存IssueのReferenceから次を説明できる状態を目指します。

- deterministicで解けるcaseが上位modelへ不要に送られない
- lightweight cognition / modelが候補生成へ寄与してもauthorityを持たない
- unknown / abstentionが成功へ丸められない
- Proposalがminimum semantic deltaとしてShared Acceptanceへ入る
- model / providerを交換・削除してもCanonical meaningが変わらない
- execution artifact / Hostを交換してもSemantic Operationを追跡できる
- current authority / Resource / Observationが過去のProposalやmodel memoryと分離される
- Human reviewでmeaning / inference / unknown / provenanceを区別できる

各Evidence tierの更新は、それぞれのRequirement auditでactual Reference成立後に行います。
