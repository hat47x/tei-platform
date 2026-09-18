# TEI Design Synthesis / 認知支援 読み方ガイド

この文書は、TEIで検討しているDesign Synthesis、Derived Cognition、local LLMの関係を、**個別アルゴリズムより先にProduct上の目的から理解するための入口**です。

正式なProduct RequirementやArchitectureを追加する文書ではありません。現在の上位計画は[`../plan/DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md`](../plan/DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md)、Derived Cognitionの検討は[`../plan/DERIVED_COGNITIVE_EXPLORATION_OVERVIEW_2026-09.md`](../plan/DERIVED_COGNITIVE_EXPLORATION_OVERVIEW_2026-09.md)以降にあります。

この領域は、LLM、検索、グラフ、constraint、embedding、FlyHash、HDC等の技術名が多くなりやすいため、まず「何をしたいのか」だけを押さえます。

## 1. 目標は「AIに全部書かせること」ではない

TEIが目指しているのは、要件を一文渡してAIに設計書、コード、テストを大量生成させることではありません。

長期的には、人が何度も同じ意味を別形式へ書き直す作業を減らし、**人の仕事を、業務上の意味・例外・責任・選択・受理へ寄せること**を目指します。

現在の開発では、たとえば次のような作業を人が別々に行います。

```text
業務要件を書く
Data Modelを作る
画面を設計する
Databaseを設計する
codeを書く
testを書く
整合しない箇所を修正する
```

TEIが目指す方向は、より次に近いものです。

```text
人がpurpose / business meaningを示す
        ↓
TEIが既存知識と制約を調べる
        ↓
複数の設計案を提示する
        ↓
矛盾する案を理由付きで除外する
        ↓
本当に意味上の選択が必要な箇所だけ人が判断する
        ↓
Accept
        ↓
UI / schema / binding / code / testを可能な限り派生する
        ↓
実行結果を見て必要な意味だけ修正する
```

これは「人を開発から外す」ことではありません。

**機械的に再生成できる作業を人が繰り返さず、人にしか判断できない意味へ集中する**ことが目的です。

## 2. なぜLLMへ最初から丸投げしないのか

たとえば、次の依頼を考えます。

> 顧客を選び、複数の明細を持つ受注登録機能を設計してください。

LLMはそれらしい案を作れます。しかし、TEI Repositoryにはすでに次のような情報が存在する可能性があります。

- Customerは既存Dataとして存在する
- Customer masterはread-onlyである
- `header-detail` Patternがある
- quantityには正数Validationがある
- submit用Capabilityがある
- Web Targetで利用可能なinteractionが決まっている
- 類似するaccepted Referenceがある

これらは、LLMに推測させるより、TEIが正確に取得できる情報です。

したがって、先にローカル処理で既知の事実を集めます。

```text
exact identity lookup
relation / dependency graph
constraint / validation check
Reference / Pattern recall
Target / Provider capability check
cross-view consistency check
```

その後にLLMを使えば、LLMは「Repository全体を覚える」ことではなく、**残った少数の意味上の問題を比較・統合すること**へ能力を使えます。

## 3. 一番重要なのは「案を増やすこと」より「安全に減らすこと」

生成AIでは「何案出せるか」に目が向きやすいですが、TEIではcandidate generationより**candidate contraction（候補空間の縮約）**を重視します。

たとえば、最初に次の三案が出たとします。

```text
A. Order + OrderLine
B. 一枚のflat OrderRow
C. Order本体だけ持ち、明細は外部serviceへ委譲
```

その後、既知の意味を当てます。

```text
既知:
  明細ごとのidentityが必要
  明細のquantity validationが必要
  外部service ownershipの根拠はない
```

すると、

```text
B → 明細identityを保持できない
C → 外部ownershipの根拠がない
A → 残る
```

と理由付きで絞れます。

この時点でも、Aの中で未解決が残るかもしれません。

```text
未解決:
  lineNumberは業務上の識別子か
  単なる表示順か
```

ここはLLMまたは人が扱う価値があります。

**既知の制約で決められることと、意味判断が必要なことを分ける**のがDesign Synthesisの中心です。

## 4. ローカル認知は何をするのか

ここでいう「認知」は、人のような万能知能をTEI内部へ作ることではありません。

主に次のような仕事を、LLMを呼ぶ前に軽量に行う仕組みを指します。

### 正確にわかることを取り出す

- Canonical IDを解決する
- Relationを辿る
- Validationを調べる
- Source / Evidenceを辿る
- read / write境界を確認する

### 既存の類似物を思い出す

- 同じApplication Pattern
- 構造が似たData
- 類似したRelation構造
- 過去のReference
- partial cueから連想できる既存資産

### 変なところへ注意を向ける

- read-onlyなのにwriteしようとしている
- 必要なDataが見当たらない
- 既存Patternから大きく外れている
- 類似候補がほとんどない
- 逆に、既存資産と重複していそう

この結果はFactではありません。

```text
similarity != applicability
novelty    != unknown
attention  != error
candidate  != decision
```

「近そう」「珍しそう」「確認した方がよさそう」という信号として使い、最終判断はdeterministic qualificationや人のAcceptanceへ渡します。

### Knowledgeが存在することとLLMへ渡すことも別

RepositoryやSkill群にKnowledgeが存在しても、現在taskのLLM contextへ全量投入する必要はありません。

```text
available knowledge
      ↓ routing / shortlist
selected candidate
      ↓ deterministic qualification
applicable / incompatible / unknown
      ↓ disclosure decision
bounded context
```

ここでは次を同一視しません。

```text
available
  != selected

selected
  != currently applicable

currently applicable
  != automatically disclosed
```

routing scoreやmodel-native signalはshortlistを作るためのsignalであり、current applicabilityのEvidenceではありません。また、qualification済みでも現在taskに不要なKnowledgeを無条件にpromptへ入れません。

一方、`unknown`やmissing authorityが現在の判断に重要なら、削除せずunresolvedとしてbounded contextへ含めます。context削減のために不確実性を隠すことはしません。

## 5. Derived Cognitionは第二の正本ではない

この領域で特に避けたいのは、検索indexやAI memoryの中にしか存在しない業務意味を作ることです。

TEIでは少なくとも次を分けます。

```text
authoritative semantic memory
  accepted source / Canonical meaning

derived cognitive memory
  index / graph / sparse code / embedding / similarity

experience memory
  proposal / decision / run / correction history
  + condition / action / observed outcome / failure / boundary等のverified local relation候補
```

Derived Cognitionは二番目です。

indexを削除しても、Canonical meaningは失われてはいけません。

algorithmをFlyHashから別方式へ変えても、業務意味が変わってはいけません。

学習によってcandidate rankingが変わっても、それだけで仕様が変わってはいけません。

experience memoryも同じです。過去に成功したcompositionやtrajectoryは、現在の設計で使える候補を呼び戻すhintにはできますが、current applicabilityのEvidenceではありません。

```text
past success
  != current applicability
  != accepted meaning
```

再利用する場合は、元のExecution / Observation / Evidenceへ戻れるprovenanceを保ち、current accepted authority / dependency / Target条件で再qualificationします。experience memoryを削除してもCanonical Worldは変わってはいけません。

## 6. local LLMは何を担当するのか

local LLMへ期待するのは、すべての事実を保持することではありません。

前段で十分に問題を小さくした後、次のような仕事を担当させます。

- A案とB案の違いを説明する
- trade-offを整理する
- 複数案の良い部分を統合する
- unresolvedな前提を質問にする
- cross-view矛盾について仮説を出す
- accepted meaningへ入れる最小差分を提案する
- 宣言的に表現できない残余のcode候補を作る

LLMへ渡す情報も、Repository全部ではなくboundedなcontextにします。

たとえば次です。

```text
目的
accepted snapshot / required authority fingerprints
既知のfact
既存資産
hard constraint
候補案
除外済み案と理由
unresolved
Evidence reference
authority limit
求める出力形式
```

この形なら、小型local LLMでも得意な範囲へ問題を寄せられます。

## 7. remote LLMは「使えると便利」だが必須にしない

TEIでは、少なくとも一つの実用Referenceについて、次の構成で成立することを目標にします。

```text
local deterministic core
+ local index / cognition
+ local encoder if needed
+ local LLM
+ local compiler / generator / test runner
```

remote LLMは、難しいcaseや品質向上のためのoptional escalationです。

これは、remote AIを否定するためではありません。

- 外部serviceが使えない環境
- 機密性の高い業務
- latency / cost
- predictable operation
- offline / edge環境

でもTEIの基本設計支援を成立させられる余地を残すためです。

model名やparameter数は急速に変化するため、Architectureへ固定しません。

## 8. 実装とテストをどう減らすのか

設計案をAIが出せるだけでは、人の実装作業はあまり減りません。

Design Synthesisの後半では、accepted meaningから下流artifactを派生させます。

```text
accepted semantic definition
        ↓
既存component / pattern
        ↓
declarative rule / binding
        ↓
deterministic generator / compiler
        ↓
必要ならconstrained local LLM
        ↓
最後にcustom handwritten code
```

手書き実装を禁止するのではなく、**意味から再生成できない残余へ限定する**方向です。

### testも意味から作る

テストも実装codeだけを見て作ると、実装と同じ誤りをなぞる可能性があります。

そのため、可能な限りaccepted meaningから独立してtest oracleを導出します。

例:

```text
quantity > 0
  ↓
0 / negativeをrejectするtest

relation cardinality = many
  ↓
複数明細を保持できるtest

read-only datasource
  ↓
writeが許可されないtest

Capability outcome
  ↓
各Business Resultを確認するscenario
```

実装とテストが同じaccepted meaningを別経路から参照することで、自己追認を減らします。

### 生成量より「変更増幅」を減らす

Design Synthesisの価値は、generated fileを大量に作ることではありません。

たとえば一つのaccepted semantic definitionから、

```text
Validation
Projection
Target configuration
test / fixture
documentation
bounded AI context
```

を必要に応じて派生できれば、同じ意味変更を各surfaceへ人が転記する必要を減らせます。

重要なのは、

```text
one semantic change
  -> one authority change
  -> affected derived surfaces rebuild / stale
```

という方向です。

そのためReferenceでは、「何個生成できたか」より、同じchange challengeで**何箇所の重複authoring・manual synchronization・handwritten residualが残ったか**を見ます。特殊Interactionやalgorithm等がnative codeとして残ること自体は失敗ではなく、意味の重複が局所化されているかを確認します。

## 9. 人に残す仕事

この構想で、人から取り除きたいのは判断そのものではなく、**再生成可能な転記と反復作業**です。

人には、たとえば次が残ります。

- 何を実現したいか
- 業務用語の意味
- 本当に異なる複数案のどちらを選ぶか
- 法務・組織・責任上の判断
- 例外をどう扱うか
- Evidenceを見て受理するか

逆に、次は機械へ寄せます。

- 既存資産探索
- dependency確認
- consistency check
- standard pattern適用
- 定型UI / schema生成
- binding候補
- test derivation
- impact分析

成功の指標は「AIがたくさん書いたか」ではありません。

**人が同じ意味を何回手入力しなくて済んだか、そして必要な判断だけが人へ残ったか**を見るべきです。

## 10. 最初から全自動を狙わない

この構想は段階的に検証します。

```text
S1 既存資産を見つける
  ↓
S2 複数の設計候補を作り、他の設計要素で縮約する
  ↓
S3 local LLMで残候補を比較・統合する
  ↓
S4 accepted meaningからimplementation / testを派生する
  ↓
S5 実行結果を意味側へfeedbackする
```

最初の`knowledge-first`探索は、Product全体のゴールではありません。

候補生成とauthorityの境界を小さく検証できるため、最初の実証スライスに選んでいます。

## 11. ハエ脳由来技術はどこに入るのか

FlyHash等のハエ嗅覚系由来の着想は、S1〜S2の一部、つまり**既存構造の想起や候補探索**へ使えるかを比較する対象です。

概念的には、

```text
明示的なfeature
  ↓
高次元へ展開
  ↓
少数の強い反応だけを残す
  ↓
近い構造の小さな候補集合を得る
```

という性質が使えるかを見ます。

しかし、TEIが必要としているのはFlyHashそのものではありません。

同じProduct Problemを、

- exact / graph
- weighted set similarity
- SimHash
- FlyHash / DenseFly
- HDC / VSA
- embedding

等で比較し、最も単純な方式で十分ならそれを選びます。

algorithm名をCanonical vocabularyへ入れません。

## 12. この領域で混同してはいけないこと

Design Synthesis / Cognition関連では、次の区別を特に守ります。

```text
提案できる      != 正しい
似ている        != 適用できる
新しそう        != unknownである
LLMが自信あり   != Evidenceがある
Acceptされた    != 実行権限がある
実装できた      != 業務上完了した
AIで自動化した  != Product価値が成立した
```

この区別があるため、AIや認知機能を強くしても、既存のCanonical / Evidence / Authority境界を弱めずに済みます。

## 13. 読む順序

初めてこの領域を読む場合は、次の順を推奨します。

1. 本書 — 何を目指すかを具体例から理解する
2. [`overview.md`](overview.md) — TEI全体の中でAI / local cognitionを見る
3. [`../plan/DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md`](../plan/DESIGN_SYNTHESIS_LOOP_REQUIREMENT_DRAFT_2026-09.md) — 上位Product GoalとDSR候補
4. [`../plan/DERIVED_COGNITIVE_EXPLORATION_OVERVIEW_2026-09.md`](../plan/DERIVED_COGNITIVE_EXPLORATION_OVERVIEW_2026-09.md) — 非LLM認知を検討する理由
5. [`../plan/DERIVED_COGNITIVE_REQUIREMENT_DRAFT_2026-09.md`](../plan/DERIVED_COGNITIVE_REQUIREMENT_DRAFT_2026-09.md) — TEIへ要求する性質
6. `ISSUE-000047 / 000051` — Product Referenceと評価責務
7. Benchmark / Feature Extraction Contract — 実験詳細

詳細なfeature atomやalgorithm比較から読み始める必要はありません。

## 14. 現時点で決まっていないこと

この構想があるからといって、次はまだ確定していません。

- Derived Cognitionを永続的なArchitecture Layerにすること
- FlyHash / HDC / embedding等の採用
- vector DBの採用
- local LLMのmodel family
- local LLMだけですべてのcaseを処理すること
- online learning
- learned associationの永続化
- `internal/cognitive`等のpackage構成
- DSRを正式Product Requirementへ昇格すること

ReferenceでProduct価値と複雑性を比較してから判断します。
