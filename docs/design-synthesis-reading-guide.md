# TEIの設計案の統合と認知支援を読むための案内

この文書は、TEIで検討している設計案の統合（Design Synthesis）、派生情報を用いた認知支援（Derived Cognition）、ローカルで動く言語モデル（local LLM）の関係を、**個別のアルゴリズムより先に、製品としての目的から理解するための案内**です。

正式な製品要求や全体構成の決定を追加する文書ではありません。採用済みの要求は[製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md)、現在の実装・検証範囲は[現在の実装状況](status.md)で確認してください。認知支援の仕組みやアルゴリズムの候補は、未採用の研究課題として扱います。

この領域では、言語モデル、検索、グラフ、制約処理、埋め込み表現、FlyHash、HDCなどの技術名が多くなりがちです。まず、何を実現したいのかを押さえます。

## 1. 目標は「AIに全部書かせること」ではない

TEIが目指すのは、要件を一文渡すだけでAIに設計書、コード、テストを大量生成させることではありません。

長期的には、人が同じ意味を何度も別の形式へ書き直す作業を減らし、**業務上の意味・例外・責任・選択・受け入れの判断に、人の仕事を集中させること**を目指します。

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

これは、人を開発から外すことではありません。

**機械的に再生成できる作業を人が繰り返さず、人が判断すべき意味へ集中する**ことが目的です。

## 2. なぜLLMへ最初から丸投げしないのか

たとえば、次の依頼を考えます。

> 顧客を選び、複数の明細を持つ受注登録機能を設計してください。

言語モデルはもっともらしい案を作れます。しかし、TEIのリポジトリには、既に次のような情報があるかもしれません。

- Customerは既存Dataとして存在する
- Customer masterはread-onlyである
- `header-detail` Patternがある
- quantityには正数Validationがある
- submit用Capabilityがある
- Web Targetで利用可能なinteractionが決まっている
- 類似するaccepted Referenceがある

これらは、言語モデルに推測させるのではなく、TEIが正確に取得できる情報です。

そのため、まずローカル処理で既知の事実を集めます。

```text
exact identity lookup
relation / dependency graph
constraint / validation check
Reference / Pattern recall
Target / Provider capability check
cross-view consistency check
```

その後で言語モデルを使えば、リポジトリ全体を記憶させるのではなく、**残った少数の意味上の問題を比較・統合すること**に能力を使えます。

## 3. 一番重要なのは「案を増やすこと」より「安全に減らすこと」

生成AIでは何案作れるかに注目しがちですが、TEIでは候補の生成（candidate generation）以上に、**候補空間を安全に狭めること（candidate contraction）**を重視します。

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

**既知の制約で決められることと、人による意味の判断が必要なことを分ける**のが設計案の統合の中心です。

## 4. ローカル認知は何をするのか

ここでいう認知支援は、人間のような万能の知能をTEIの内部に作ることではありません。

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

こうした類似候補や注意を促す結果は、確定した事実ではありません。

```text
similarity != applicability
novelty    != unknown
attention  != error
candidate  != decision
```

「近そう」「珍しそう」「確認した方がよさそう」という手掛かりとして使います。最終的な判断は決定論的な適用条件の確認や、人による受け入れに委ねます。

### 知識が存在することと、言語モデルへ渡すことは別

リポジトリやスキル群に知識が存在しても、現在の作業で言語モデルへ渡す文脈に、すべてを含める必要はありません。

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

候補の振り分けに使う点数やモデル固有の信号は、候補一覧を絞るためのものです。現在の適用可能性を裏付ける検証根拠にはなりません。また、適用条件を確認済みでも、現在の作業に不要な知識を無条件に言語モデルへの入力へ含めません。

一方、`unknown`や権限の根拠の不足が現在の判断に重要なら、削除せず未解決の情報として、必要な範囲に絞った文脈へ含めます。入力を小さくするために不確実性を隠すことはしません。

### 文脈をまとめた情報を万能の知識基盤にしない

AIがアプリケーションの開発を再開するとき、人が変更を確認するとき、AIが業務上の操作を選ぶときでは、必要な情報が異なります。

```text
same accepted meaning
  -> Builder context
  -> Synthesis context
  -> Human review
  -> Operational context
```

このように、用途に応じて複数の表現を使うことを認めます。

共通に保つのはデータの形式ではなく、同じ正本となる意味、入力元、検証根拠、受け入れ済みの判断へ戻れることです。設計案の統合で使う文脈情報（Context Packet）は、候補比較に必要な範囲へ絞ります。AIを利用した開発全体の知識保管庫にはしません。

### 似た設計の型を見つけることと、知識を別の案件へ移すことは別

参照用の設計パターン（Reference Pattern）は、「この構造が役立つかもしれない」という候補の情報源です。

一方、あるApplicationで受入済みのPolicy、Validation、Capability contract、Infrastructure mapping等を別Applicationへ再利用する場合は、origin側の前提まで含めて扱います。

```text
origin knowledge
  + assumption / dependency / Variation
  + destination context
  -> applicability / mismatch / override
  -> destination Acceptance
```

元のアプリケーションで成立したからといって、移植先でも成立するとは限りません。また、元のアプリケーションが後から変わっても、移植先を自動的には更新しません。

destinationがoriginへlinked dependencyを持つのか、origin provenanceを残したlocal variationとして独立するのかを区別できることが重要です。具体的なpackage / inheritance機構はReference前に固定しません。

## 5. Derived Cognitionは第二の正本ではない

特に避けたいのは、検索インデックスやAIの記憶の中にしか存在しない業務上の意味を作ることです。

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

認知支援のための派生情報（Derived Cognition）は二番目に当たります。

検索インデックスを削除しても、正本となる意味は失われてはいけません。

アルゴリズムをFlyHashから別の方式へ変えても、業務上の意味が変わってはいけません。

学習によって候補の順位が変わっても、それだけで仕様が変わってはいけません。

experience memoryも同じです。過去に成功したcompositionやtrajectoryは、現在の設計で使える候補を呼び戻すhintにはできますが、current applicabilityのEvidenceではありません。

```text
past success
  != current applicability
  != accepted meaning
```

再利用する場合は、元のExecution / Observation / Evidenceへ戻れるprovenanceを保ち、current accepted authority / dependency / Target条件で再qualificationします。experience memoryを削除してもCanonical Worldは変わってはいけません。

## 6. ローカルの言語モデルは何を担当するのか

ローカルの言語モデルに期待するのは、すべての事実を保持することではありません。

前段で十分に問題を小さくした後、次のような仕事を担当させます。

- A案とB案の違いを説明する
- 選択肢間の得失を整理する
- 複数案の良い部分を統合する
- 未解決の前提を質問にする
- 複数の視点にまたがる矛盾について仮説を出す
- 受け入れ済みの意味へ反映する最小限の差分を提案する
- 宣言的に表現できない残りの部分についてコード案を作る

言語モデルに渡す情報も、リポジトリ全体ではなく必要な範囲に絞った文脈にします。

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

この形式なら、小型のローカル言語モデルが得意な範囲へ問題を絞れます。

## 7. 外部の言語モデルは利用できるが、必須にはしない

TEIでは、少なくとも一つの実用的な参照例を、次の構成だけで成立させることを目標にします。

```text
local deterministic core
+ local index / cognition
+ local encoder if needed
+ local LLM
+ local compiler / generator / test runner
```

外部の言語モデルは、難しい事例への対応や品質向上のために、必要に応じて追加利用するものです。

これは、外部のAIの利用を否定するためではありません。

- 外部serviceが使えない環境
- 機密性の高い業務
- latency / cost
- predictable operation
- offline / edge環境

でもTEIの基本設計支援を成立させられる余地を残すためです。

モデル名やパラメータ数は急速に変わるため、全体構成の仕様には固定しません。

## 8. 実装とテストをどう減らすのか

設計案をAIが出せるだけでは、人の実装作業はあまり減りません。

設計案の統合の後半では、受け入れ済みの意味から下流の成果物を生成します。

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

実装コードだけを見てテストを作ると、実装と同じ誤りを繰り返す可能性があります。

そのため、可能な限り、受け入れ済みの意味から実装とは独立してテストの期待結果を導きます。

例は次のとおりです。

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

設計案の統合の価値は、生成ファイルを大量に作ることではありません。

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

そのため参照例では、生成物の数よりも、同じ変更課題に対して**重複した記述・手作業での同期・手書き実装が何か所残ったか**を確認します。特殊Interactionやalgorithm等がnative codeとして残ること自体は失敗ではなく、意味の重複が局所化されているかを確認します。

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

成功の指標は、AIがどれだけ大量に書いたかではありません。

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

最初に行う`knowledge-first`探索は、製品全体の最終目標ではありません。

候補生成と権限の境界を小さな範囲で検証できるため、最初の実証対象に選んでいます。

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

アルゴリズム名を、正本となる意味の基本語彙には取り込みません。

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
3. [製品要求](../requirements/TEI_PRODUCT_REQUIREMENTS.md) — 正式なRequirementsと未決事項を区別する
4. [現在の実装状況](status.md) — AI支援の実装済み範囲と未検証事項を確かめる
5. 必要なReference / Evidence — 個別に再現できる内容だけを検討根拠として参照する

特徴の最小単位やアルゴリズムの比較から読み始める必要はありません。

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

参照例を使って製品価値と複雑さを比較してから判断します。
