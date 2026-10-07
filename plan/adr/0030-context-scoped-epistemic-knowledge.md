# ADR-0030: 生成AIが扱う情報を文脈付きの認識状態として保持する

- 状態: 採用
- 日付: 2026-10-06

## 背景

生成AIを設計、研究、知識管理へ継続的に使うと、保存された文書には、人間が確認した事実、ユーザーがその文脈で事実として与えた前提、AIが生成した未確認情報、仮説、誤りが混在します。

保存後にこの区別が失われると、後続の生成AIは未確認情報まで既成事実として参照しやすくなります。AI生成文が別の文書へ転記され、さらに要約や設計判断の根拠として再利用されると、誤りの由来を追えなくなります。

一方、ユーザーがプロンプトで事実として与えた情報まで一律に未確認扱いし、毎回再承認を求める設計も適切ではありません。ユーザーが「この案件では顧客IDは整数です」と述べた場合、その案件の作業では通常の前提として利用する必要があります。

ただし、その情報が特定案件、時点、環境では正しくても、別案件や別versionで常に正しいとは限りません。ユーザーは適用文脈を毎回完全には入力しないため、情報の由来だけでなく、どの文脈で成立する前提なのかを扱う必要があります。

TEIにはすでに、AI ProposalとCanonical Acceptanceを分離するADR-0022、Knowledge qualificationとfreshness、Source / Evidenceへのprovenance、Semantic Extension、Plugin Capabilityの境界があります。本ADRはこれらを一般の保存知識へ広げ、人間入力、AI生成物、review結果を後続のAIが利用するための認識上の境界を定めます。

## 判断

保存される情報を、単一の「正しい / 未確認」フラグでは管理しません。少なくとも、由来、発話の種類、確認状態、適用文脈、時点、根拠を分離して保持します。

概念上は次を直交する軸として扱います。

```text
origin
  user / ai / system / imported / external source

assertion mode
  fact assertion / hypothesis / question / instruction / other

review state
  native assertion / unreviewed / reviewed / rejected / superseded

context
  explicit scope / inferred scope / subject / environment

validity time
  valid at / checked at / review time

provenance
  source / evidence / review record / derivation
```

具体的なschema名やenumはReferenceで決めます。本ADRは、これらを一つの状態値へ畳まないことを決めます。

### ユーザーが事実として与えた情報は、現在文脈の前提として扱う

ユーザーがプロンプトで事実として述べた内容は、AI生成文と同じunreviewed既定値にはしません。その会話、project、subject等の現在文脈では、ユーザーが与えた前提として通常どおり利用します。

これは「世界について客観的に検証済み」という意味ではありません。別の文脈へ再利用するときは、適用範囲、時点、対象の一致を別途評価します。

ユーザーが仮説、推測、疑問として述べた内容をfact assertionへ自動変換しません。

### AI生成情報は、保存されたことだけでは事実へ昇格しない

AIが生成した主張は原則としてunreviewedから始めます。ファイル、チャット履歴、Issue、設計書等へ保存されたこと自体をreviewの代わりにはしません。

人間が内容を確認した場合は、その確認が成立したscopeと時点を伴ってreviewedとして扱えます。

AIがreviewed factを要約、統合、言い換えした場合も、新しい文章は原則として新しいAI生成物です。元情報がreview済みであることだけを理由に、生成結果へreview stateを継承しません。意味保存を機械的に証明できる転記等が必要になった場合は、別のtransformation contractとして扱います。

### review済みとは、ある文脈で採用可能な前提として確認されたことを表す

reviewedは普遍的なTruthを意味しません。

確認結果には、可能な範囲で次を結び付けます。

- 対象subject
- project / repository / branch / version
- document / conversation等のlocal context
- environment
- valid time / checked time
- reviewer / review Evidence
- 条件や例外

確認によって「本番環境では正しいが、外部APIでは異なる」と分かった場合、単純なYes / Noへ畳まず、scopeを狭める、条件を追加する、assertionを分割する、元assertionをsupersededにする等の処理へ進めます。

### 文脈はユーザー入力だけに依存しない

利用者へ毎回context metadataを入力させません。

現在のworkspace、project、repository、branch、document、section、conversation、選択対象、environment等から取得できる文脈は、システムが自動で結び付けます。

ただし、明示された文脈と推定した文脈を区別します。推定したscopeが不足している場合、狭い文脈で得た情報を広い文脈へ黙って一般化しません。

文脈の階層はReferenceで固定しますが、下位scopeで成立したassertionを上位scopeへ自動昇格しないことを原則とします。

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

similarity、model confidence、retrieval rankをTruthやreview済み状態へ昇格させません。

### semantic editはreview validityを再評価する

review済みの文章を変更した場合、以前のreviewを現在値へ無条件に継承しません。

空白、表示順、format等の意味を変えない変更まで一律に失効させる必要はありませんが、意味が変わる編集を検出した場合は再確認対象に戻します。

どこまでをsemantic changeとみなすかは、Source / Semantic Diff等の既存機構を利用してReferenceで決めます。

### 矛盾を自動的に消さない

review済みのassertion同士が矛盾した場合でも、AIがもっともらしい一方を選び、矛盾が解消したことにはしません。

双方のscope、time、source、review recordを保持します。文脈差で両立できる場合はscopeを精密化し、両立できない場合はConflictとして確認対象へ送ります。

### Project Knowledge Healthは単純な確認率だけで判定しない

未確認情報の比率はProject Healthのsignalにはできますが、それだけを強いKPIにはしません。

Project Healthでは、未確認、stale、矛盾、scope不一致の情報が、どの判断や実装に依存されているかを重く見ます。

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

alertを表示するだけで終わらせず、review requestへ変換できる契約を持ちます。

review requestからは少なくとも次へ戻れるようにします。

- 確認するassertion
- 現在のstatus / scope
- 確認が必要な理由
- downstream impact
- Source / Evidence
- reviewer候補
- review結果
- unresolved / conditional result

AIやruleがreviewer候補を提示しても、それを正式なAuthorityや担当者の確定には使いません。

### UI上の単純さと内部modelの精密さを分ける

SUI等のUIでは、確認、未確認、仮説、否定等の少数状態を高速操作として見せられます。ただし、画面上の4状態を内部modelの全状態へ1対1で固定しません。

PCでは範囲選択と1操作、複数選択の一括処理を基準にし、origin、scope、Evidence等の詳細は必要なときだけ表示します。

AI出力へunreviewedを付ける操作は人間へ要求しません。AI生成時に自動付与します。

### TEI Coreへ直書きせず、Extension / Pluginから検証する

本契約はTEI、SUI、SEIで再利用できる一方、最初からTEI Coreの固定enumへ追加しません。

ISSUE-000077のSemantic ExtensionとISSUE-000080 / #951のPlugin Capability contractを利用し、まず標準Extension / Pluginとして成立するかを検証します。

候補namespaceとして`std.epistemic`を検討しますが、最初のReference前には固定しません。

概念上の責務は次のように分けます。

```text
TEI
  persistence / provenance / context binding / review event / health contract

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
- review結果を条件やscope付きで保持できる
- 検索・認知処理が未確認情報をfactと同じ強さで扱うことを避けられる
- 矛盾やstale情報をreview対象として残せる
- Project Healthを「何が未確認か」だけでなく「何がそれに依存しているか」から判断できる
- SUIの操作を単純に保ちながら、SEI / TEIでは精密なmetadataを利用できる

### 増える負担

- assertionのprovenanceとscopeを保持する必要がある
- review済み情報の編集時にvalidityを再評価する必要がある
- retrieval時にcontext compatibility等を評価する必要がある
- review requestとProject Healthの依存解析が必要になる
- inferred contextの誤推定を観測、訂正できる必要がある

これらは、生成AIを長期的な設計・知識管理へ利用するために受け入れます。

## 採用しない案

### 保存された文書をすべて事実として扱う

AI生成物の未確認情報や誤りが、保存を境に事実として再利用されるため採用しません。

### 人間入力もすべてunreviewedにする

ユーザーが作業前提として与えた情報まで再承認が必要になり、通常の対話や設計作業を不自然にするため採用しません。

### review済みをglobal booleanで持つ

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
- AI生成主張が保存だけでreviewedへ昇格しない
- origin / assertion mode / review state / scope / time / Evidenceを分離できる
- explicit contextとinferred contextを区別できる
- narrow scopeのfactをwide scopeへ黙って一般化しない
- reviewed textのsemantic changeでreview validityを再評価できる
- AI summaryが元factのreview stateを無条件継承しない
- contradictionを保持できる
- retrievalでrelevanceとcontext compatibilityを分離できる
- critical dependencyを使ったProject Knowledge Healthを計算できる
- stakeholder review requestへimpact / Evidence / scopeを渡せる
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
