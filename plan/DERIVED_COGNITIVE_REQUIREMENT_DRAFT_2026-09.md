# Derived Cognitive Requirements Draft — 他チャット知見からの要件素案

## 位置づけ

この文書は、TEIの非LLM認知方式について、他チャットで検討してきたSEI Cognition、SUI / kj-atlas、ハエ脳由来の計算原理、SOZA系の認知・意思決定境界から、**TEIへ取り込むべき性質だけを抽出し、Product Requirement候補へ落とすための中間文書**です。

状態: **Requirement draft / 未採用**

この文書はFlyHash、HDC/VSA、SEI Cognition、SUI、local LLM、分散計算基盤等の採用を意味しません。既存RQ-014〜RQ-024を置き換えず、重複する内容は既存RQの具体化候補として扱います。

文書階層上は次の位置です。

```text
Derived Cognitive Exploration Overview
        ↓
本書: 何を要件として求めるか
        ↓
ISSUE-000047: Product Reference
        ↓
ISSUE-000051: candidate retrieval方式の評価
        ↓
Benchmark / Feature Extraction Contract
        ↓
Gate C後の実装方式
```

## まずTEI側の根幹問題へ戻す

他チャットには、認知層そのもの、KJ法的な外在化、局所LLM、分散実行、ウェアラブル / ホームデバイス、組織内遊休計算資源等まで広い構想があります。

TEIへそのまま全部を持ち込むと、TEIが「業務意味を実行可能にする基盤」から「組織全体の認知基盤」へ膨張します。

TEIが必要としているのは、次の狭い能力です。

```text
current intent / editing context
        ↓
既存Canonical / Pattern / Evidenceを軽量に探索
        ↓
確認すべき候補を小さくする
        ↓
exact qualification
        ↓
Proposal / Shared Acceptance
```

したがって他チャットから取り込むのは、**認知主体そのものではなく、TEIのknowledge-first探索を改善する契約・境界・評価原則**です。

## 他チャットから取り込むもの

### 1. SEI Cognitionから取り込むもの

SEI Cognition側で検討されている汎用認知核のうち、TEIへ有効なのは次です。

- deterministic / symbolic → lexical sparse → sparse associative → static semantic → heavier local encoder → LLMという段階的cascade
- 疎連想、novelty、近傍想起、bundle等をLLM非依存で扱う考え方
- 汎用的なfeature入力契約を持ち、tokenizer / embedding / ontologyへ固定依存しないこと
- 高速・低メモリで常時利用できる認知方式を優先すること
- ownershipとdeployment topologyを分離すること
- 最初からremote serviceを前提にせず、in-process / same-hostでも成立すること
- local LLMは候補・制約・Evidenceが整理された後の局所推論に使うこと

TEIへはこれらを**実装所有権ではなくprovider contract上の性質**として取り込みます。

TEIがSEIの内部memory modelや認知主体概念を所有する必要はありません。

### 2. SUI / kj-atlasから取り込むもの

SUI / kj-atlasの検討からTEIへ有効なのは次です。

- 一つのglobal similarityへすべてを畳まない
- 問いに応じて近傍、比較、橋渡し、残余、時系列、来歴等の異なるprojectionを使う
- 明示関係、解析上の近接、provenanceを区別する
- activation / inhibition / novelty / persistence / competition等を「注意・探索」の信号として扱う
- hard routingを避け、複数fieldの候補を保持して後段で照合する
- 近接やクラスタが真理、品質、適用可能性を意味しない

一方、親和図canvas、KJの島作り、人間協働UI、Consensus形成、外部協働空間そのものはSUI側の責務であり、TEIのProduct Requirementにはしません。

### 3. ハエ脳由来の検討から取り込むもの

生物学的再現ではなく、次の計算上の性質を評価対象へ残します。

- normalization / gain control
- high-dimensional expansion
- sparse competition / k-WTA
- pattern separation
- partial cueからのassociative recall
- novelty / familiarity
- persistence / decay / habituation
- local / selective plasticity
- uniform randomだけでなくsemantically biased sparse projectionを比較可能にすること

ただし、これらをProduct Requirementそのものへはしません。

Product Requirementへ上げるのは、「一部のfeature量に支配されない」「近いが異なる資産を分離できる」「partial cueから候補を回収できる」「novel caseをnearest matchへ強制しない」「学習がCanonicalを暗黙変更しない」といった外部から確認可能な性質です。

### 4. SOZA系の認知・判断境界から取り込むもの

TEIの既存境界を補強する形で、次を保持します。

```text
Observation != Evidence != Assurance != Truth
Unknown     != No
Conflict    != latest-wins
Recommendation != Decision
Decision       != Execution Authority
```

Derived cognitionについてはさらに次を要求します。

```text
Activation      != Fact
Similarity      != Applicability
Novelty         != Unknown
Association     != Dependency
Candidate Rank  != Recommendation Authority
Learned Memory  != Decision History
```

### 5. SEIの統合頂点の検討から借りる二つの考え方（2026-09-27追記・評価待ち）

2026-09-27に、SEI側で「SEIを製品群の統合の頂点に置く」ための設計検討が行われました（`hat47x/sei-cognition:product/value/SEI_FAMILY_APEX_VALUE_REVIEW.md`、`hat47x/sei-cognition:plan/adr/ADR-0032-family-integration-apex-and-control-meaning-boundary.md`。どちらもSEIの未統合branch `docs/family-apex-backflow-20260927`にだけあり、SEI側でも未採用です）。そこから、TEIの派生認知にも当てはまりそうな考え方を二つ借ります。

どちらも**要件ではありません**。`DCR-*`の番号を付けず、既存DCRの具体化を検討するときの候補として置きます。TEIのReferenceで確かめるまで、要件の根拠にはしません（DCR-22）。

#### A. 制御の判断を記録する

段階的な探索（DCR-03）では、軽い方式で足りれば重い方式を使わず、必要な場合だけLLMや人へ上げます（DCR-16）。このとき「どの段を使ったか」だけでなく、**何を調べ、何を意図的に調べず、何をより重い認知や人へ上げたか**という制御の判断を、候補と一緒に残せることが望ましい、という考え方です。

たとえば、exact lookupとlexical sparseで候補が十分に絞れたため、semantic encoderとLLMは使わなかったとします。この「使わなかった」ことが記録されていなければ、後から候補の漏れが見つかったとき、方式の限界なのか、意図して省いた範囲なのかを区別できません。

SEI側では、ルーティングや昇格の**仕組み**は将来の基盤モデルやプラットフォームに置き換わっても、**何を見て何を見送ったかという判断の意味**は残る、という整理から出てきた考え方です（ADR-0032の提案）。神経系の比喩から出たものではありません。

TEIでは、[Actor Context Projections §3.4](ACTOR_CONTEXT_PROJECTIONS_2026-09.md)の「Omissionを隠さない」、DCR-10のdiscovery provenance、DCR-21の理由つきqualificationと重なります。新しい記録の型を足す前に、これらの既存の境界で「調べなかった範囲」まで表せるかを先に確かめます。

#### B. 予期された不確実性と、予期しない不確実性を分ける

分かっていないことには二種類ある、という考え方です。

- **予期された不確実性。** すでに「分からない」と認めて受け入れたもの。見直す時期や条件を決めて待つのが適切です（TEIでいえば、明示された`unknown` / `unresolved`）。
- **予期しない不確実性。** 既存の前提と食い違う観測や、これまでのどの候補にも近くない新しさ（novelty）。前提そのものを疑うきっかけとして、注意を向け直すのが適切です。

DCR-09は、noveltyや低activationを`unknown`の代わりにしないことを求めています。この考え方は、その区別をもう一歩進め、二つを**別の注意の信号として扱う**ことを示唆します。前者は予定した再確認へ、後者はcross-viewの違和感の信号（DCR-17）や人への確認へ、という振り分けです。

これは**神経系の比喩から出た考え方**です（予期された不確実性と予期しない不確実性を別の神経修飾系が担うというYu & Dayan 2005の議論を、SEI側が比喩として借りたもの）。CSW（cultural-substrate-weaving）の帰属の語でいえば、対象から支持された知見（`target_supported`）ではなく、枠組みから生成された仮説（`framework_generated`）にあたります。したがって、この比喩から型、field、要件を直接作りません。TEIのReferenceで、二つを分けることで確認の負担や見落としが実際に減るかを観測できた場合に限り、DCR-09 / DCR-17の具体化を検討します。

## TEIへ直接持ち込まないもの

次は関連研究としては価値がありますが、現時点のTEI要件にはしません。

- SEIの「認知主体」「簡易記憶」そのもの
- SUIの親和図canvas、外部協働、Consensus形成UI
- KJ法の島作りengineそのもの
- 組織内遊休計算資源を束ねるdistributed compute fabric
- wearable / home deviceをTEI runtime targetとして常時利用する構想
- 3B〜4B級local LLMの常駐runtime
- whole-fly connectome / SNN simulation
- online learning / reinforcementを最初から有効にすること
- Excel方眼紙format認識、test-data生成等の個別ユースケースをTEI Coreへ一般化すること

これらは将来Provider / external subsystem / import toolとして接続できる余地を残しますが、TEIの意味モデルへ先回りして取り込みません。

## 要件素案

以下は正式RQ番号ではなく、議論用の`DCR-*`です。

### DCR-01 Candidate cognitionは非権威的であること — Must

Derived cognitionが返すのはcandidate、attention signal、association、novelty等であり、Canonical Fact、Semantic applicability、Evidence、Decision、write authorityを直接確定してはいけません。

確認方法:

- candidateを削除してもCanonical Worldが変化しない
- candidate採用には既存Shared Acceptance Boundaryが必要
- scoreだけで`applicable`へ遷移しない

対応RQ: RQ-015 / RQ-016 / RQ-019 / RQ-021

### DCR-02 認知方式なしでもTEI本体が成立すること — Must

Derived cognitionはoptional acceleration / discovery mechanismであり、不在・停止・不採用でもexact lookup、Canonical validation、Driver、Acceptance等の基本Product動作が成立しなければなりません。

確認方法:

- cognitive indexを削除してもCanonical reload / validationが成立する
- cognitive provider unavailable時にauthority pathが停止・破損しない

対応RQ: RQ-003 / RQ-017 / RQ-021 / RQ-023

### DCR-03 knowledge-first探索を段階化できること — Must

検索は一つの巨大modelへ固定せず、少なくとも次を組み合わせられる必要があります。

```text
exact / symbolic
→ graph / dependency
→ lexical sparse
→ associative / structural
→ optional semantic encoder
→ optional LLM
```

すべてを毎回直列実行することは要求しません。軽い方式で十分なときに重い方式を必須にしないことが重要です。

対応RQ: RQ-014 / RQ-015 / RQ-023 / RQ-024

### DCR-04 candidate retrieval方式を交換可能にすること — Must

FlyHash、weighted set similarity、HDC/VSA、embedding等をCanonical modelへ焼き込まず、同じFeature Corpus / Query contractから比較・交換できる必要があります。

確認方法:

- encoder変更でCanonical schema変更を要求しない
- qualification contractをencoder変更から独立させる

対応RQ: RQ-015 / RQ-021 / RQ-022

### DCR-05 partial cueから候補を回収できること — Should

完全なCanonical IDや完全な自然言語説明がなくても、role、relation、pattern、constraint等の一部情報から既存資産を候補へ戻せることを目指します。

ただしpartial cueだけで適用可能性を断定してはいけません。

対応RQ: RQ-014 / RQ-015

### DCR-06 lexical similarityとstructural similarityを分離できること — Must

名前が似ていること、同じidentityであること、構造が似ていること、利用contextが似ていることを同一scoreへ不可逆に潰してはいけません。

少なくとも次を区別可能にします。

- exact identity
- lexical cue
- intrinsic semantic structure
- application-use context
- structural neighborhood

対応RQ: RQ-002 / RQ-014 / RQ-015 / RQ-017

### DCR-07 role / direction / multiplicityを保持した構造探索を可能にすること — Should

Relation source / target、cardinality、Capability input / output、Pattern role等の方向性を保持したまま比較できる方式を少なくとも評価対象に含めます。

bag-of-featuresだけを唯一の構造表現にしません。

対応RQ: RQ-001 / RQ-014 / RQ-026

### DCR-08 一つのglobal cognitive fieldへhard routeしないこと — Should

Data、Relation、Validation、Capability、Application Pattern、Business Procedure等を早い段階で一つのfieldへ排他的に分類し、その分類誤りを後段へ固定伝播させないことを目指します。

複数candidate family / fieldを並行評価し、deterministic qualificationで絞り込める構造を許容します。

対応RQ: RQ-014 / RQ-015 / RQ-026

### DCR-09 novelty / familiarityを`unknown`と分離すること — Must

低activation、高novelty、低margin等はattention signalとして利用できますが、Semantic `unknown`の代替にしてはいけません。

逆にnearest candidateが存在しても、適用可能な候補が存在するとは限りません。

確認方法:

- true novel caseで`no-applicable-candidate`を保持できる
- top-1が常にsuccessにならない

対応RQ: RQ-015 / RQ-016 / RQ-020 / RQ-021

### DCR-10 candidateからCanonical / Source / Evidenceへ戻れること — Must

候補は少なくとも、元のCanonical identity、current qualificationへ戻るSource / Evidence referenceへ追跡可能でなければなりません。derived representationから発見した場合は、そのrepresentationを生成したbuild input / extractor / model等のprovenanceも別に辿れることを要求します。

```text
discovery provenance
  != current semantic qualification authority
```

認知indexだけにしか存在しない業務意味を作りません。

対応RQ: RQ-002 / RQ-015 / RQ-016 / RQ-021

### DCR-11 derived cognitive memoryは再生成可能・stale検出可能であること — Must

index、sparse code、embedding、association等はderived resourceとして扱い、**実際にその表現生成へ使ったbuild input**との対応を保持します。

build inputには対象に応じて、feature sourceのsemantic fingerprint / scoped source snapshot、extractor version / configuration、encoder / model configuration、seed等を含められます。repository全体へ一つのglobal revision counterを導入することは要求しません。

- 削除して再生成できる
- 実依存build inputが変わった場合にstaleを検出または安全に再生成できる
- 無関係なaccepted changeだけで全cognitive representationを一律staleにすることを要求しない
- stale representationを黙ってcurrent qualificationへ渡さない

さらに:

```text
derived representation freshness
  != M1 accepted authority / proof freshness
```

である。retrieval時点でrepresentationがfreshでも、`ISSUE-000047`のcurrent applicability判定やcross-view proofは、必要なcurrent accepted authority / Evidenceへ独立に戻って確認しなければならない。

対応RQ: RQ-003 / RQ-015 / RQ-016 / RQ-020 / RQ-021

### DCR-12 semantic subjectとusage contextを分離すること — Must

あるSemantic assetを利用するApplicationが増えただけで、そのasset自身のintrinsic representationが意味上別物へ変化してはいけません。

Subject BaseとApplication / task context overlayを分離し、必要に応じて組み合わせます。

対応RQ: RQ-001 / RQ-003 / RQ-014 / RQ-017

### DCR-13 lightweight / local-firstに成立すること — Should

初回のnon-LLM cognitionは、外部AI service、GPU、常駐LLMを必須にせず、CPU上で小さいmemory footprintと低latencyを測定可能であることを目指します。

固定数値はReference EvidenceなしにProduct Requirementへ入れません。

測定対象:

- index build / rebuild cost
- incremental update cost
- memory footprint
- p50 / p95 retrieval latency
- candidate set size

対応RQ: RQ-023 / RQ-024

### DCR-14 ownershipとdeployment topologyを分離すること — Should

将来SEI Cognition等の汎用認知kernelを利用する場合でも、library ownership、process placement、remote service化を同一判断へ畳みません。

初期Referenceはin-process / same-hostでも成立可能とし、remote化は共有運用、resource sharing、分離保守等の明確な利益がある場合に評価します。

TEIはremote cognitive serviceを必須依存にしません。

対応RQ: RQ-019 / RQ-023 / RQ-024

### DCR-15 cognitive providerへauthority / secretを暗黙委譲しないこと — Must

外部またはremote cognitive providerを利用しても、write、Capability invocation、secret、network credential、filesystem material等のauthorityを認知機能であることだけを理由に付与しません。

Providerへ渡す入力は必要最小限とし、general feature corpusへruntime secretを混ぜません。

対応RQ: RQ-019 / RQ-020

### DCR-16 LLMをoptional reasoning stageとして接続できること — Should

LLMはintent解釈、仮説、比較、反証、説明、composition等へ利用できますが、candidate retrievalの唯一の入口にはしません。

non-LLM層が候補、制約、provenanceを整理し、その結果をLLM contextとして渡せる構造を目指します。

LLM出力もProposalでありShared Acceptance Boundaryを迂回しません。

対応RQ: RQ-015 / RQ-016 / RQ-019

### DCR-17 cross-viewの違和感をattention signalとして返せること — Later

Data、Function / Interaction、Business Procedure等のcross-viewで、既存patternから外れた箇所、確認対象になりそうな残余、bridge候補等をattention signalとして返す余地を残します。

ただしsignalをhard semantic errorへ自動変換しません。

対応RQ: RQ-016 / RQ-026

### DCR-18 persistence / habituationを明示履歴の代替にしないこと — Later

「前にも見た」「最近繰り返している」といったfamiliarity / persistenceを探索優先度へ使うことはできますが、RQ-016の「一度回答した質問を繰り返さない」は明示的Decision / suppression historyで実装します。

neural habituationやactivation decayをDecision historyの正本にしません。

対応RQ: RQ-016 / RQ-017

### DCR-19 learned associationは局所的・可逆・outcome-awareであること — Deferred

静的retrievalで価値が確認されるまでonline learningを必須にしません。

将来学習する場合、少なくとも次を区別します。

```text
semantic-reuse-success
incompatible
unknown / insufficient evidence
stale-derived-representation
policy-reject
acceptance-failure
ignored / no decision
```

学習対象にするoutcomeと、単に監査へ残すoutcomeを分けます。Decision historyそのものは既存の明示履歴が正であり、association weightへ置き換えません。

局所更新を試す場合は、活動したfeature / candidate fieldに関係するassociationだけを変更し、無関係なfieldやCanonical feature extractorへ副作用を与えないことを最初の安全条件にします。

対応RQ: RQ-015 / RQ-016 / RQ-021 / RQ-022

### DCR-20 normalization / representation biasを評価可能にすること — Later

feature数が多いsubjectや高degree nodeが類似度を支配しないよう、normalization、weighting、feature-group balancing等をencoder側で比較可能にします。

特定normalization方式をProduct Requirementへ固定しません。

対応RQ: RQ-021 / RQ-022 / RQ-024

### DCR-21 再利用できない理由も返せること — Must

knowledge-first探索は「候補を出す」だけでなく、後段qualificationによって、少なくとも`applicable / incompatible / unknown / no-applicable-candidate`を区別できる必要があります。

incompatible / unknownの場合は、scope、dependency、Datasource、Capability、Validation、Evidence不足等の理由へ戻れることを目指します。

対応RQ: RQ-014 / RQ-015 / RQ-016

### DCR-22 Product採否はTEI自身のReferenceで判断すること — Must

外部研究やSEI / SUI上の成功をTEIでの採用Evidenceへ直接転用しません。

少なくとも次をTEI Referenceで比較します。

- exact / graph baseline
- set / hash / sparse associative方式
- role-aware structural方式
- true novel / near-match / partial cue
- stale representation
- 第二Applicationを含むcross-application case
- latency / memory / rebuild cost

対応RQ: RQ-021 / RQ-022 / RQ-024

## 初回Referenceへ持ち込む要件

Gate C後の最初のprobeでは、全DCRを同時に実装しません。

優先するのは次です。

```text
DCR-01  non-authoritative
DCR-02  optional / removable
DCR-03  knowledge-first cascade
DCR-04  replaceable retrieval
DCR-05  partial cue
DCR-06  lexical / structure separation
DCR-09  novelty != unknown
DCR-10  provenance
DCR-11  rebuild / stale
DCR-12  subject / context separation
DCR-13  lightweight measurement
DCR-21  explicit qualification outcome
DCR-22  TEI Reference Evidence
```

`DCR-17`以降のcross-view attention、habituation、learning等は、最初の静的探索がProduct価値を示してから判断します。

## 既存RQへ昇格させる候補

現時点では新しい正式RQ番号を増やさず、まずReferenceで不確実性を減らします。

そのうえで次のどちらかを選びます。

### 既存RQの具体化で足りる場合

主に次へ吸収できます。

- RQ-014 Application Pattern再利用
- RQ-015 knowledge-first生成AI
- RQ-016 AI確認提案と履歴
- RQ-017 二重管理最小化
- RQ-019 authority / Resource
- RQ-021 Reference / Conformance
- RQ-022 research / production区別
- RQ-023 一人で維持可能
- RQ-024 Target別performance
- RQ-026 cross-view検証

### 新しいRQが必要になる場合

Referenceで「LLMの有無に関係なく、lightweight derived cognitionそのものがTEIのProduct価値として必要」と確認できた場合だけ、たとえば次の上位要求を正式化します。

> TEIは、Canonical authorityを変更せず、既存意味資産から確認すべき候補を軽量・再生成可能・provenance付きに導出できること。

この要求を正式化する前に、exact / graphだけで十分ではないEvidenceが必要です。

## 所有境界の暫定整理

```text
SEI Cognition
  generic sparse / associative cognition kernel
  novelty / recall / bundle
  optional local LLM orchestration

SUI
  affinity-specific featureization
  spatial externalization
  human collaboration / consensus context

TEI
  Canonical semantic inputs
  TEI-specific Feature Corpus adapter
  candidate family / qualification context
  Source / Evidence / scoped freshness trace
  Product Reference / Acceptance boundary

shared or external provider
  optional encoder / retrieval implementation
```

TEIがSEI Cognitionを利用する場合でも、TEI固有のsemantic interpretationとqualificationはTEI側に残します。

```text
SEI-like cognition may say:
  "these structures are close"

TEI must still say:
  "this candidate refers to X,
   its discovery representation came from build input B,
   and X is currently applicable / incompatible / unknown
   under the required accepted authority / Evidence"
```

## 次に分解するIssue

本書の次に作るべき作業は、golden atom等の詳細実装ではなく、要件群をProduct Scenarioへ束ねることです。

候補は次の3束です。

### A. Existing asset recall

対象DCR:

- 03 / 04 / 05 / 06 / 07 / 09 / 10 / 21

Product question:

> 名前を完全に知らなくても、既存の再利用候補へ安全に到達できるか。

### B. Cognitive memory safety

対象DCR:

- 01 / 02 / 11 / 12 / 15 / 18 / 19

Product question:

> 派生記憶がstale、学習済み、外部配置になってもCanonical authorityを侵食しないか。

### C. Lightweight cognition / escalation

対象DCR:

- 03 / 13 / 14 / 16 / 20 / 22

Product question:

> 単純な探索をLLMへ委譲せず、必要な場合だけより重い認知へ段階的に上げられるか。

この3束をReference Scenarioへ落としてから、feature atom、API、package、SEI integration等の実装課題へ降ります。
