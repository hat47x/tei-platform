# ADR-0030: 生成AIが扱う情報を文脈付きの認識状態として保持する

- 状態: 採用
- 日付: 2026-10-06

## 背景

生成AIを設計、研究、知識管理へ継続的に使うと、保存された文書には、人間が確認した事実、ユーザーがその文脈で事実として与えた前提、AIが生成した未確認情報、仮説、誤りが混在します。

保存後にこの区別が失われると、後続の生成AIは未確認情報まで既成事実として参照しやすくなります。AI生成文が別の文書へ転記され、さらに要約や設計判断の根拠として再利用されると、誤りの由来を追えなくなります。

一方、ユーザーがプロンプトで事実として与えた情報まで一律に未確認扱いし、毎回再承認を求める設計も適切ではありません。ユーザーが「この案件では顧客IDは整数です」と述べた場合、その案件の作業では通常の前提として利用する必要があります。

ただし、その情報が特定案件、時点、環境では正しくても、別案件や別versionで常に正しいとは限りません。ユーザーは適用文脈を毎回完全には入力しないため、情報の由来だけでなく、どの文脈で成立する前提なのかを扱う必要があります。

TEIにはすでに、AI ProposalとCanonical Acceptanceを分離するADR-0022、Knowledge qualificationとfreshness、Source / Evidenceへのprovenance、Semantic Extension、Plugin Capabilityの境界があります。本ADRはこれらを一般の保存知識へ広げ、人間入力、AI生成物、確認結果を後続のAIが利用するための認識上の境界を定めます。

## 判断

保存される情報を、単一の「正しい / 未確認」フラグでは管理しません。少なくとも、由来、発話の種類、確認状態、適用文脈、時点、根拠を分離して保持します。

概念上は、少なくとも次を別の軸として扱います。

```text
content provenance
  user / ai / system / imported / external source

ingestion actor
  user / ai / system / connector / import process

statement kind
  fact / observation / requirement / decision / constraint
  hypothesis / question / instruction / preference / other

confirmation
  unreviewed / confirmed / rejected / unknown

lifecycle
  active / superseded / retracted / detached

context binding
  observed / declared / inferred

validity time
  valid from / valid until / checked at / reviewed at

provenance links
  source / evidence / 確認イベント / derivation
```

具体的なschema名やenumはReferenceで決めます。本ADRが固定するのは、これらを一つの状態値へ畳まないことです。たとえば、ユーザーが現在の文脈で直接述べたため通常の前提にできることは「確認済み」と同じ意味ではありません。また、条件付きで確認されたことは確認状態ではなく適用条件の性質であり、後続情報に置き換えられたことはライフサイクルの性質です。

### ユーザーが事実として与えた情報は、現在文脈の前提として扱う

ユーザーがプロンプトで事実として述べた内容は、AI生成文と同じ扱いにはしません。その会話、project、subject等の現在の文脈では、ユーザーが直接与えた前提として通常どおり利用します。ただし、確認イベントがなければ確認状態そのものは未確認のままです。

これは「世界について客観的に検証済み」という意味ではありません。別の文脈へ再利用するときは、適用範囲、時点、対象の一致を別途評価します。

ユーザーが仮説、推測、疑問として述べた内容を事実へ自動変換しません。

### 事実以外の確定情報も、情報の働きを保って扱う

設計作業では、事実だけが前提になるわけではありません。ユーザーが「この方式を採用する」と決めた内容、「この制約を守る」と指定した内容、「この要件を満たす」と定めた内容も、後続の設計や実装に影響します。

これらを事実へ変換すると、世界についての記述と設計上の決定が混同されます。逆に、事実ではないという理由だけで仮説と同じ弱さに落とすと、確定した設計判断を後続AIが無視できます。

そのため、少なくとも事実、観測、要件、決定、制約、仮説、質問、指示を区別します。現在の作業で利用できるかどうかは、情報種別、文脈、確認状態、必要なAuthorityを組み合わせて判断します。ユーザーが現在の作業で直接与えた要件や決定は通常の作業前提として利用できますが、それだけで組織上の正式承認やCanonical Acceptanceを得たことにはしません。

### 入力した人と、内容の出所を分ける

チャットへ入力した主体と、その内容を最初に生成・記述した主体は同じとは限りません。ユーザーがAIの回答や外部資料を貼り付けた場合、その情報をuser由来へ置き換えると、AI生成物や引用文がユーザー自身の主張として扱われます。

そのため、情報の内容がどこから来たかと、誰が今回のシステムへ取り込んだかを分けて保持します。引用、転記、import、connector経由の取得でも、元の出所が分かる場合はそのprovenanceを失いません。出所が分からない既存資料は、ユーザー由来や確認済みへ推定せず、出所不明として扱います。

### AI生成情報は、保存されたことだけでは事実へ昇格しない

AIが生成した情報は原則として未確認から始めます。ファイル、チャット履歴、Issue、設計書等へ保存されたこと自体を人間の確認の代わりにはしません。

人間が内容を確認した場合は、その確認が成立した適用範囲と時点を伴って確認済みとして扱えます。

AIが確認済みの情報を要約、統合、言い換えした場合も、新しい文章は原則として新しいAI生成物です。元情報が確認済みであることだけを理由に、生成結果へ確認状態を継承しません。意味保存を機械的に証明できる転記等が必要になった場合は、別のtransformation contractとして扱います。

### 確認済みとは、ある文脈で採用可能な前提として確認されたことを表す

確認済みであることは、普遍的に真であることを意味しません。

確認結果には、可能な範囲で次を結び付けます。

- 対象subject
- project / repository / branch / version
- document / conversation等のlocal context
- environment
- valid time / checked time
- 確認者 / 確認Evidence
- 条件や例外

確認によって「本番環境では正しいが、外部APIでは異なる」と分かった場合、単純なYes / Noへ畳みません。適用範囲を狭める、条件を追加する、情報を分割する、元情報を置換済みとして扱う等の処理へ進めます。

### 文脈はユーザー入力だけに依存しない

利用者へ毎回、文脈のmetadataを入力させません。

現在のworkspace、project、repository、branch、document、section、conversation、選択対象、environment等から取得できる文脈は、システムが自動で結び付けます。ただし、文脈そのものにも由来があります。

- observed: repository、branch、現在時刻など、システムが直接観測した文脈
- declared: ユーザーや正本資料が明示した文脈
- inferred: AIやheuristicが周辺情報から推定した文脈

推定した文脈だけを根拠に適用範囲を広げません。推定結果は検索順位、候補の絞り込み、確認依頼には使えますが、狭い文脈で成立した情報を広い文脈へ確定的に一般化するAuthorityにはしません。

文脈の階層はReferenceで具体化しますが、下位の適用範囲で成立した情報を上位へ自動昇格しないことを原則とします。

### 検索と認知処理では、関連度と適用可能性を分ける

検索、RAG、SEI Cognition等が情報を再利用するとき、semantic relevanceだけで採用可否を決めません。

少なくとも次を別々に扱います。

- semantic relevance
- epistemic reliability
- context compatibility
- freshness
- contradiction / supersession
- source / authority suitability

一つの総合scoreへまとめる実装は否定しませんが、個別signalを失って理由を説明できなくなる設計にはしません。

similarity、model confidence、retrieval rankをTruthや確認済み状態へ昇格させません。

鮮度も情報側の有効期限だけで決めません。同じ情報でも、月次レポートと副作用を伴う実行判断では必要な新しさが異なります。情報が最後に確認された時点と、利用側が要求する最大経過時間や再確認条件を分け、用途に応じて鮮度を評価できるようにします。

### 確認履歴は上書きせず、現在状態を履歴から導出する

同じ情報を複数人が確認したり、後から確認を撤回したり、条件を追加して再確認したりすることがあります。単一の確認記録を上書きすると、誰が何を確認し、その後に何が変わったかを失います。

確認、条件付き確認、否定、撤回等は履歴として追記し、現在の確認状態は対象となる意味、適用範囲、時点、確認イベントから導出する構造を優先します。人間が内容を確認したことと、その人に組織上の決定Authorityがあることも分けます。

### annotationの対象は編集後に誤って付け替えない

paragraph、card、row、選択範囲等へannotationする場合、文字位置だけを対象identityにしません。利用できる場合は、logical subject、source revision、構造上の位置、引用文字列と前後文脈等を組み合わせて対象を再特定します。

編集後に同じ対象だと安全に判断できなければ、別の文章へannotationを黙って移しません。detached、ambiguous、stale等として再確認できる状態を残します。

### 意味が変わる編集では確認の有効性を再評価する

確認済みの文章を変更した場合、以前のreviewを現在値へ無条件に継承しません。

空白、表示順、format等の意味を変えない変更まで一律に失効させる必要はありませんが、意味が変わる編集を検出した場合は再確認対象に戻します。

どこまでをsemantic changeとみなすかは、Source / Semantic Diff等の既存機構を利用してReferenceで決めます。

### 矛盾を自動的に消さない

確認済みのassertion同士が矛盾した場合でも、AIがもっともらしい一方を選び、矛盾が解消したことにはしません。

双方の適用範囲、時点、出所、確認履歴を保持します。文脈差で両立できる場合は適用範囲を精密化し、両立できない場合は矛盾として確認対象へ送ります。

ユーザーが現在の作業で与えた情報と、既存のCanonical sourceや一次資料が食い違う場合も同様です。ユーザー入力を無視せず、既存正本を黙って上書きもせず、異なるprovenanceを持つ矛盾として扱います。

### Project Knowledge Healthは単純な確認率だけで判定しない

未確認情報の比率はProject Healthのsignalにはできますが、それだけを強いKPIにはしません。

さらに、確認状態が付いている情報だけを分母にすると、まだannotationされていない大量のAI出力や既存資料を見落とせます。そのため、Project Knowledge Healthとは別に、重要な情報領域の状態をどこまで把握できているかというepistemic coverageも扱います。重要領域が未分類、出所不明、対象不明である場合は、それ自体をUnknownとして残します。

Project Healthでは、未確認、stale、矛盾、適用範囲の不一致、coverage不足の情報が、どの判断や実装に依存されているかを重く見ます。

Project Knowledge Healthでは少なくとも次を考慮します。

- epistemic state
- importance / criticality
- downstream dependency impact
- context compatibility
- freshness
- contradiction
- review age

強いalertは、重要な判断や実装が未確認情報へ依存している場合など、利用者が次の行動を取れる条件へ限定します。

### 未確認事項をステークホルダー確認へ接続する

alertを表示するだけで終わらせず、確認依頼へ変換できる契約を持ちます。

確認依頼からは少なくとも次へ戻れるようにします。

- 確認するassertion
- 現在の状態 / 適用範囲
- 確認が必要な理由
- downstream impact
- Source / Evidence
- 確認者候補
- 確認結果
- unresolved / conditional result

AIやruleが確認者候補を提示しても、それを正式なAuthorityや担当者の確定には使いません。

### 既存資料とmetadataを失う入出力を明示的に扱う

既存の文書には、AI生成か人間作成か、どの文脈で確認されたかが分からないものがあります。このようなlegacy資料を、読み込めたという理由だけで確認済みへ昇格しません。出所や確認状態を復元できない場合はunknown provenanceやunclassifiedとして扱い、必要な範囲から段階的に確認します。

Markdown、GitHub Issue、外部ストレージ等へ本文だけを書き出すと、epistemic metadataが失われる場合があります。metadataを保持できる形式では一緒に出力し、保持できない形式では喪失を明示します。metadataなしで再取り込みした本文を、以前の確認状態へ自動復元しません。

### UI上の単純さと内部モデルの精密さを分ける

SUI等のUIでは、確認、未確認、仮説、否定等の少数状態を高速操作として見せられます。ただし、画面上の4状態を内部modelの全状態へ1対1で固定しません。

PCでは範囲選択と1操作、複数選択の一括処理を基準にし、出所、適用範囲、Evidence等の詳細は必要なときだけ表示します。

AI出力へunreviewedを付ける操作は人間へ要求しません。AI生成時に自動付与します。

### TEI Coreへ直書きせず、Extension / Pluginから検証する

本契約はTEI、SUI、SEIで再利用できる一方、最初からTEI Coreの固定enumへ追加しません。

ISSUE-000077のSemantic ExtensionとISSUE-000080 / #951のPlugin Capability contractを利用し、まず標準Extension / Pluginとして成立するかを検証します。

候補namespaceとして`std.epistemic`を検討しますが、最初のReference前には固定しません。

概念上の責務は次のように分けます。

```text
TEI
  persistence / provenance / context binding / 確認イベント / health contract

SUI
  low-friction annotation / review queue / stakeholder handoff / health projection

SEI
  context-compatible retrieval / status-aware context assembly / conflict and stale handling
```

同じepistemic modelを各productへ複製しません。

## 影響

### 良くなる点

- AI生成物を保存しただけで事実化する連鎖を防げる
- ユーザーが与えた前提を自然に利用できる
- 別projectや別versionへの誤った一般化を減らせる
- 確認結果を条件やscope付きで保持できる
- 検索・認知処理が未確認情報をfactと同じ強さで扱うことを避けられる
- 矛盾やstale情報をreview対象として残せる
- Project Healthを「何が未確認か」だけでなく「何がそれに依存しているか」から判断できる
- SUIの操作を単純に保ちながら、SEI / TEIでは精密なmetadataを利用できる

### 増える負担

- 情報の出所と取り込み主体を分けて保持する必要がある
- statement kind、provenance、適用範囲を保持する必要がある
- annotation対象を編集後に再特定し、曖昧な場合は再確認する必要がある
- 確認履歴を上書きせず保持する必要がある
- legacy資料やmetadataを失う入出力のUnknownを扱う必要がある
- 確認済み情報の編集時にvalidityを再評価する必要がある
- retrieval時にcontext compatibility等を評価する必要がある
- 確認依頼とProject Healthの依存解析が必要になる
- inferred contextの誤推定を観測、訂正できる必要がある

これらは、生成AIを長期的な設計・知識管理へ利用するために受け入れます。

## 採用しない案

### 保存された文書をすべて事実として扱う

AI生成物の未確認情報や誤りが、保存を境に事実として再利用されるため採用しません。

### 人間入力もすべて未確認情報と同じ扱いにする

ユーザーが作業前提として与えた情報まで再承認が必要になり、通常の対話や設計作業を不自然にするため採用しません。

### 確認済みをglobal booleanで持つ

どの文脈、時点、対象で確認されたかを失い、別文脈へ誤って一般化しやすいため採用しません。

### 確認済み比率だけをProject Healthにする

重要でない大量の未確認メモと、認証方式等のcriticalな未確認前提を同じ重みで扱うため採用しません。

### Promptで「未確認情報を事実として扱わない」と指示するだけにする

保存、検索、要約、別modelへのhandoffで状態が失われ、強制可能な境界にならないため採用しません。

### 初回から全文をClaim graphへ正規化する

人間の入力負担、分割誤り、保存量、処理量が大きく、最初の価値検証を遅らせるため採用しません。まずparagraph、card、row、selected range等へ軽量にannotationできるReferenceから始めます。

## 検証

ISSUE-000082 / GitHub Issue #958では、少なくとも次を確認します。

- user factual assertionとAI unreviewed assertionを区別できる
- fact / observation / requirement / decision / constraint / hypothesis等の情報の働きを区別できる
- 入力した主体と内容の出所を分離できる
- observed / declared / inferred contextを区別できる
- 複数確認イベントから現在の確認状態を導出できる
- annotation対象を編集後に誤って別箇所へ付け替えない
- legacy資料やmetadata欠落を確認済みへ推定しない
- epistemic coverage不足をProject Knowledge Healthと併せて観測できる
- AI生成情報が保存だけで確認済みへ昇格しない
- content provenance / ingestion actor / statement kind / confirmation / lifecycle / context / time / Evidenceを分離できる
- observed / declared / inferred contextを区別できる
- 狭い適用範囲の情報を広い適用範囲へ黙って一般化しない
- 確認済みtextのsemantic changeで確認の有効性を再評価できる
- AI summaryが元情報の確認状態を無条件継承しない
- contradictionを保持できる
- retrievalでrelevanceとcontext compatibilityを分離できる
- critical dependencyを使ったProject Knowledge Healthを計算できる
- stakeholderへの確認依頼へimpact / Evidence / 適用範囲を渡せる
- SUI #3184とSEI #660が同じ契約を利用できる
- Extension / Pluginを外しても既存Canonical meaningが変化しない

## 関連

- GitHub Issue #958
- SUI Sensemaking #3184
- SEI Cognition #660
- ADR-0022 AI ProposalとCanonical受入境界を分離する
- `plan/issue/000027-ai-proposal-acceptance-boundary.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`
- `plan/issue/000077-semantic-extension-reference.md`
- ISSUE-000080 / GitHub Issue #951
