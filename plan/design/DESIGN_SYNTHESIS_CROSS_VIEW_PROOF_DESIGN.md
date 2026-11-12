# Design Synthesis Cross-View Proof Design

状態: **評価待ち（未採用）**

## 1. この文書の役割

この文書は、TEIのDesign SynthesisでData、Function / Interaction、Business Procedure、Security、Runtime / Observation等の異なる視点を使って設計候補を縮約するとき、**何を根拠に「支持される」「矛盾する」「まだ判断できない」と扱うか**を整理する機能設計案です。

この文書自体はProduct Requirement、ADR、Gate、Canonical schemaの正本ではありません。

- Product Requirementの正本: [`../../requirements/TEI_PRODUCT_REQUIREMENTS.md`](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)
- Business設計境界: [`../../docs/business-system-design.md`](../../docs/business-system-design.md)
- RQ-025 / RQ-026解釈: [`../../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md`](../../requirements/BUSINESS_SYSTEM_DESIGN_REQUIREMENTS.md)

主にRQ-026の「異なる視点から相互検証できること」と、未採用DSR-02 / DSR-12の候補縮約を接続するための設計仮説です。

## 2. 背景

TEIでは、Data、Function / Interaction、Business Procedureを、同じ業務システムに対する異なるreview viewとして扱います。

```text
                Business
               /        \
              /          \
           Data -------- Function / Interaction
```

重要なのは、三種類の設計書を揃えることではありません。

たとえば「出荷を確定する」というBusiness Actionがある場合、次の問いは互いに別です。

- そのBusiness ActionはApplicationで支援されるのか
- 支援されるなら、どのPattern / Capabilityが直接realizationなのか
- そのinteractionは何を読み、何を変更するのか
- その変更はaccepted operation authorityを持つのか
- 実行者にAuthorizationがあるのか
- system上の成功と、物理的な出荷完了は同じ意味なのか
- 次工程への責任移転は何を条件に成立するのか

一つのviewだけからこれらすべてを推論すると、もっともらしいが根拠のない設計が完成しやすくなります。

Design Synthesisでは、この問題がさらに大きくなります。AIや探索器が複数候補を出した後に、単なるsimilarity scoreやLLM rankingで候補を絞ると、accepted authorityと推測の境界が失われます。

そこで、cross-view照合を単なる「整合性チェック」ではなく、**小さな証明義務（proof obligation）の集合**として扱う案を検討します。

## 3. 防ぎたい失敗

### 3.1 空欄を否定事実へ変える

```text
Data traceが見つからない
  -> Data dependencyは存在しない
```

とは限りません。trace不足はunresolvedであり得ます。

### 3.2 実装可能性をBusiness意味へ逆流させる

```text
DBへUPDATEできる
  -> Business上Updateしてよい
```

とは限りません。

### 3.3 時系列や隣接をsemantic relationへ変える

```text
Step BがStep Aの直後
  -> AからBへHandoffした
```

とは限りません。

### 3.4 Business actorをSecurity authorityへ変える

```text
actor = 倉庫担当者
  -> Authorization Role = warehouse
```

を自動生成してはいけません。

### 3.5 runtime successを外部世界の事実へ変える

```text
Command accepted
  -> physical shipment completed
```

ではありません。

### 3.6 derived graphを第二の正本にする

探索やcross-view照合のために構築したgraphのedgeが、accepted sourceに戻れないままauthorityになることを避けます。

## 4. 中心原則

### 4.1 証明する対象は「問い」であり、巨大な完全modelではない

TEIが一度に「この業務システムは完全に正しい」と証明する必要はありません。

代わりに、個別の問いを明示します。

```text
Q: Business Action BA-12はApplicationで直接realizeされるか
Q: Pattern AP-04はOrderLineへwrite authorityを持つか
Q: Handoff H-02はObservation O-07だけから成立すると言えるか
```

各問いについて、現在のaccepted authorityからどこまで言えるかを評価します。

### 4.2 proof resultは少なくとも三状態を失わない

```text
supported
  必要なaccepted authorityが解決でき、問いを支持する

contradiction
  accepted authority同士が、対象の問いについて両立しない

unresolved
  identity、authority、actuality、Evidence等が不足し、肯定も否定もできない
```

これは現時点で永続enumを要求しません。まずReference / report上の意味分類です。

### 4.3 「証明できない」と「偽」を分ける

```text
not proven != false
```

特にabsenceをnegative factへ変換しません。

### 4.4 derived resultはauthorityへ逆流しない

```text
accepted sources
    -> resolve / derive
    -> proof result / graph / matrix

proof result / graph / matrix
    -X-> accepted sourcesへの暗黙backfill
```

矛盾はauthoritative sourceを**再検討する理由**にはなりますが、修正はShared Acceptance Boundaryを通してsource側で行います。

## 5. Derived Meaning Graph

cross-view照合の実装では、異なるsourceのidentityとrelationを横断するgraphが有用です。ただし、このgraphを新しいsource of truthにはしません。

```text
authoritative file-native sources
        │
        │ parse / normalize / exact resolve
        ▼
Accepted Semantic Snapshot
        │
        │ derive
        ▼
Derived Meaning Graph
        ├─ cross-view proof
        ├─ impact analysis
        ├─ matrix / diagram
        ├─ candidate contraction
        └─ DesignContextPacket
```

要求する性質は次です。

- graphを削除して再構築してもaccepted meaningを失わない
- algorithmやstorageを変更してもaccepted meaningは変わらない
- edgeは可能な限り、その導出元のaccepted identityへ戻れる
- heuristic / similarity edgeとexact semantic edgeを混同しない
- graphにedgeが無いことをrelation不存在とみなさない

したがって、現段階ではgeneric `SemanticGraph` Canonical kind、graph DB、RDF、knowledge graph製品等をProduct contractにしません。

## 6. Proof Obligationの概念形

実装schemaではなく、reviewで必要になる概念を次のように整理します。

```text
Proof Obligation
  question
  subject / scope
  accepted authorities required
  derivation path
  result
    supported | contradiction | unresolved
  supporting / conflicting refs
  unresolved reason
  evaluated snapshot
```

### question

何を確認したいかです。曖昧な「整合しているか」ではなく、Yes / Noまたは限定された意味分類へ落とせる問いにします。

### subject / scope

どのBusiness Action、Pattern、Data、Handoff、Policy等についての問いかを示します。

### accepted authorities required

結論を出すのに必要なauthorityです。必要なauthorityが未解決なら、都合のよい代替sourceで穴埋めしません。

### derivation path

どのexact refやaccepted relationを辿ったかです。単なる説明文ではなく、再評価可能であることを重視します。

### result

`supported / contradiction / unresolved`を意味上区別します。

### evaluated snapshot

どのaccepted source集合に対して評価したかを再現できるようにします。ここは後述するAcceptance Snapshotの考え方と接続します。

これらをすべて永続fieldとして採用する必要はありません。Referenceでlossが起きるものだけをdurable contract候補にします。

## 7. 最初に持つべきproof family

### P1 — Business Action / Realization

**問い:** このBusiness ActionはApplicationでどのように支援されるか。

許される根拠:

- explicit Application Pattern ref
- explicit Capability ref
- accepted realization relation

禁止する短絡:

- Action名とFunction名の類似
- actorから利用画面を推測
- 前後StepのFunctionを流用

Function realizationが無い場合もBusiness Action自体をinvalidにしません。

### P2 — Business / Data Trace

**問い:** このBusiness Actionについて、accepted refからData / Relationへのtraceを導出できるか。

結果例:

```text
supported:
  exact Pattern -> Query -> Data traceが得られる

unresolved:
  Actionは存在するがdirect realization / Data traceが無い
```

重要なのは、

```text
no current Data trace != no Data dependency
```

を維持することです。

### P3 — Function / Mutation Authority

**問い:** このinteractionは対象Dataを変更してよい意味を持つか。

根拠候補:

- accepted operation contract
- Capability / Actionのmutation semantics
- datasource policy

禁止する短絡:

- physical DBがwrite可能
- Data traceが存在する
- Patternがrecord系である
- lifecycle transitionが見える

### P4 — Business / Security Boundary

**問い:** このBusiness文脈から要求されるinteractionについて、accepted Security authorityと矛盾しないか。

```text
Business actor / responsibility
    != authenticated principal
    != Authorization Role / Policy
```

Business actor labelからSecurity objectを自動authorしません。

### P5 — System Result / Actuality

**問い:** system resultから外部・物理世界の事実まで何を言えるか。

```text
Business instruction
  != command acceptance
  != external / physical actuality
  != Business acceptance / completion
```

Observationやexternal Evidenceが無い場合、failureともsuccessとも断定しません。

### P6 — Responsibility Handoff

**問い:** あるBusiness Stepの仕事が別Stepへ継続する責任関係は、accepted Business meaningから証明できるか。

根拠:

- explicit accepted Handoff source
- from / to / conditionのaddressable meaning

証明しないもの:

- Step adjacency
- Function sequence
- actor change
- Lifecycle transition
- Command success
- Observation alone

ObservationはHandoff condition成立のEvidence候補にはなり得ますが、Handoffそのもののauthorではありません。

## 8. Design Synthesisでの利用

Proof Obligationはcandidateをauthority化するためではなく、**候補空間を安全に縮約するため**に使います。

```text
candidate generation
        ↓
proof obligations生成
        ↓
exact authority resolution
        ↓
result
  supported       -> 候補を残す根拠の一部
  contradiction   -> 除外または修正要求
  unresolved      -> 未解決dimensionとして保持
        ↓
small candidate frontier
        ↓
LLM / human review
```

### contradiction

accepted authorityと明確に両立しない候補は、理由付きで除外できます。

### unresolved

未解決は除外理由ではありません。

たとえば、lineNumberがbusiness identityかpresentation orderか未確定なら、そのdimensionだけを残して候補を比較できます。

### supported

supportedであっても「候補全体が正しい」という意味ではありません。一つのproof obligationが満たされたことだけを示します。

## 9. DesignContextPacketとの接続

Design SynthesisがLLMへ渡すbounded contextには、単なるfact一覧だけでなく、proof結果を要約して渡せます。

概念例:

```text
DesignContextPacket
  purpose
  accepted_snapshot
  exact_facts[]
  candidate_designs[]
  proof_results[]
    question
    result
    authority_refs[]
    unresolved_reason
  eliminated_candidates[]
    reason
  unresolved_dimensions[]
  authority_limits
```

LLMへRepository全体を再推論させず、

- 何がaccepted factか
- 何がderived resultか
- 何がproposalか
- 何がunresolvedか

を分離した状態で渡します。

## 10. Acceptance Snapshotと再評価

cross-view proofでは、全artifactへ一つのglobal revision番号を付けるより、**どのaccepted source集合で評価したか**を再現できることが重要です。

概念的には次です。

```text
Acceptance Snapshot
  Business source @ revision/fingerprint
  Application source @ revision/fingerprint
  Data source @ revision/fingerprint
  Security source @ revision/fingerprint
  other required authorities...
```

ここから導出されたproof resultはsnapshot-boundです。

### Business wordingだけが変わった場合

参照identityと意味が変わらなければ、Data trace等のderived resultまで無条件に変更する必要はありません。

### referenced authorityが変わった場合

Business source自体が不変でも、Pattern / Validation / Capability / Policy等のaccepted authorityが変われば、関連proofはstaleになり再評価が必要です。

### source fileが移動した場合

locator変更とsemantic changeを区別します。

この考え方は、global revision modelの採用を要求しません。まずReferenceで、fingerprint / revision / Source Index等の既存機構だけで十分かを確認します。

## 11. Claim / Interpretationを急いでCanonical化しない

proof設計を進めると、汎用的な`Claim`や`Interpretation` objectをCoreへ入れたくなります。

しかし、現段階で必要なのはまずepistemic distinctionです。

```text
source-backed fact
accepted TEI meaning
derived cross-view fact
proposal / inference
unresolved
```

これをreviewで区別できることと、generic Claim modelを持つことは同じではありません。

次のような具体的lossがrealistic Referenceで観測された場合にのみ、durable Claim identityを再検討します。

- 同一Claimを複数sourceから長期間参照する必要がある
- 相反するInterpretationをidentity付きで保持しないとreviewできない
- Evidence更新後もClaim continuityを維持する必要がある

## 12. Reference / Evaluationで確認した順序

当初の設計案ではR3をcandidate contraction、R4をchange / stale再評価としていたが、その後のread-only評価でContext Packet境界、candidate-local縮約、discovery provenanceを先に独立してPressureした。現在のR番号は、実際にmergeされた評価成果へ合わせて次で固定する。

### R1 — order entry proof reference


- `supported / contradiction / unresolved`の三状態を保持できる
- no traceをnegative factへ変換しない
- DB writeをCapability mutation authorityへ短絡しない
- Security policyをBusiness protected target authorityへ逆流させない

### R2 — manufacturing / inventory actuality / Handoff pressure


```text
Command success
Observation
Business acceptance
Handoff condition
responsibility continuation
```

を一つへ畳まないことを確認した。

この評価はaccepted second Business Handoff sourceの成立を主張しない。second realistic corpus側のpositive Handoff Evidenceは別途未解決である。

### R3 — proof-to-context boundary


- packetはdisposable / snapshot-bound projection
- `supported`はwhole candidate acceptanceではない
- `unresolved`はcandidate frontierへ残す
- local LLMはmissing authorityを発明しない

### R4 — candidate-local proof contraction


- unrelated contradictionでcandidateを除外しない
- decisive contradictionはcompact elimination certificateへ戻れる
- unresolved aggregationでもmember proof / missing authorityを失わない
- required proof欠落はfail closedする

### R5 — candidate discovery provenance


- exact / Pattern / associative discoveryをauthority上で区別する
- 同一candidateが複数経路で見つかってもcandidateを重複させない
- retrieval score / Pattern frequencyをsemantic Evidenceへ昇格しない
- existing candidateはcurrent M1 identityへ再qualificationする

### R6 — accepted snapshot / stale再評価


- proofが依存するaccepted authority fingerprintだけでfreshnessを見る
- unrelated changeで全proofを一律staleにしない
- locator-only changeとsemantic changeを分離する
- missing / ambiguous required authorityはfail closedする
- stale `contradiction`をcandidate eliminationへ使わない

R1〜R6はいずれもProduct Gateやproduction proof lifecycleの採用ではない。actual runner greenが得られていないtest-only pressureについては、source-level Evidenceをactual executionへ読み替えない。

## 13. 最小評価指標

固定KPIにはしませんが、Referenceでは少なくとも次を観測します。

- proof questionからaccepted authorityへ戻れる割合
- name similarity等のheuristicをauthorityとして使わずに済んだか
- contradictionを理由付きで再現できたか
- insufficient Evidenceをunresolvedとして保持できたか
- source change後に影響proofを再評価できたか
- derived graphを捨てて再構築して同じmeaningを得られたか
- Design Synthesisで人へ残る問いが、単純なtrace作業ではなく意味判断へ縮約されたか

## 14. 実装方式をまだ固定しない

この設計は次を要求しません。

- graph database
- RDF / OWL
- generic rule engine
- SAT / SMT solver
- `ProofObligation` Canonical kind
- `Claim` Canonical kind
- global Step ID
- global revision counter
- persisted `supported / contradiction / unresolved` enum
- LLMによるproof判定

最初のReferenceでは、既存resolverと小さいderived representationだけでも十分です。

必要なのは技術方式ではなく、**authority方向と意味分類を壊さずに問いを再評価できること**です。

## 15. 現時点の判断

現時点では、次を設計仮説として採用候補にします。

```text
cross-view review
  = complete-model validation ではなく
    bounded proof obligations の集合として扱う

proof result
  = supported / contradiction / unresolved を失わない

derived meaning graph
  = disposable projection
  != source of truth

Design Synthesis
  = contradictionを候補縮約へ使い
    unresolvedを残し
    residual meaning choiceだけをLLM / humanへ渡す
```

一方、durable proof object、generic Claim、graph storage、global revision等のCore採用は、Reference Evidenceが具体的なlossを示すまで保留します。

この方針により、三要素分析的なcross-viewの強みを保持しつつ、TEIの小さいkernel、file-native authority、Shared Acceptance Boundary、Derived Cognitionの非authority性を崩さずにDesign Synthesisへ接続できます。

## 16. 三視点の検証を説明・操作・実装で一貫させる（2026-11-12 評価案）

> 状態: **追加の設計・受入計画。実装と正式契約の採用ではない。**
> 同じ関心事について複数の図・表・Tool Result・AI説明を作るとき、どの問いを検証したかを失わないことを目的とする。既存Ownerは #1124（モデル投影）、#1127（標準CRUD）、#3250（Subsystem分割）、#3262（必要Design Context）、#2514（独立Expected）、#1163（同一改訂Evidence）とし、現Productの具体的不整合は [#4472](https://github.com/hat47x/tei-platform/issues/4472) で扱う。

### 16.1 技術的な統合：一覧や図表ではなく、問いへ戻れるProjection

最小の説明順は次の通りとする。

1. **何を変えたいか**（Businessの目的、または分野固有の対象・条件）。
2. **何が同一の対象か**（Data / Subjectの業務IdentityとOccurrence、版）。
3. **どんな規則が成立するか**（Constraint / State / Operation / OwnershipのAcceptedな意味）。
4. **どの視点から確認するか**（Data、Function、Business Procedure、CRUD、Graph、AI向け構造化Contextなどの導出）。
5. **何を問うか**（指定Scope内の必要Functionは存在するか／この更新が許されるか／違反はどのSourceにあるか）。
6. **どこまで確認できたか**（構造検査／意味の整合／Provider選択／実呼出／Effect／独立再観測とExpected）。

以上は説明用の論理依存であり、すべてを固定したCore Objectの階層として導入するものではない。

三視点は業務向けの重要なMethod Profileだが、科学データ／通信／編集／XRでも同じSubject・Source・Meaning・Scope・根拠つきProjectionを使う。CoreにBusiness Procedureを必須導入しない。新しいDomain Profileを作る際も、既存の三視点の完全コピーから始めず、分野の問いたいことを先に定義する。

### 16.2 「OK」を一つの真偽値へ潰さない

現Productでは以下を静的確認した（main 2026-11-12、`718a8b74`）。実Go Test・実CLI実行による新たな合格主張はない。

- `internal/threeelement/validate.go`: `report.OK = report.ErrorCount == 0` であり、後から格納する `ContradictoryCRUDs` と独立。したがって`OK=true`でもCRUDの矛盾は存在し得る。
- `cmd/tei/cli_three_element.go`: `three-element crud` は表の導出を行った後、結果の`OK:true`を返す。これは業務上の更新許可や全視点の完全整合を証明しない。
- `internal/threeelement/crud.go`: `GenerateCRUDGaps` は明示CRUDのみからGapを求め、Function OperationのRを無視して「CRUDなし」を返し得る。他方、明示CRUDのRead不足らしき内部`missing`を結果に加えない分岐がある。**Readが必須かは契約相対**であり、全Functionへ固定してはいけない。

望ましいConsumer表示は、**今回実施した操作の成功**、**検出した診断**、**その問いに対する適格性**を混ぜない。新しい一律の`ValidationState` enumや恒久的な`ProofRecord`をCoreへ追加する前に、既存`ValidationReport`／Proof Obligation／DesignContextPacketのread-only表示と適格性を利用する。

例:

| 問い | 実行上の事実 | 意味・適格性として返すべきこと |
| --- | --- | --- |
| CRUD表を導出できたか | CLIはJSONを返した | `projection-produced` は導出成功を表すだけ。対象集合・未収録と矛盾の有無は別 |
| このFunctionが対象Dataを書けるか | 表にUがある | Acceptedな操作の意味／状態条件／現在Grant／Effectへ遡らなければ`authorized`ではない |
| 全視点に矛盾はないか | 構造Error数が0 | Requiredな検査対象が閉じているか、矛盾一覧とunresolvedが残るかに応じて限定 |
| 注文確定は終わったか | Command受理/HTTP 200 | 更新したDB・帳票発行・外部相手の独立観測を、業務完了へ無断昇格しない |

表の文言は採用済みのCLI診断コードや新しい永続属性の確定を意味しない。

### 16.3 コードを誤読しにくくする小さな抽象化

文書化中に見えた実装の読みにくさは、単なる説明追加で回避せず、APIや処理責務の改善対象とする。

- `ThreeElementModel.Index()` は **Data Map / Function Map / Business Map の3つを位置で返す**。 `crud.go` では `_, _, dataByID := model.Index()` としてBusiness MapをData Mapとして扱う箇所がある。Compilerの型が異なる場合に検出されるかどうかに頼らず、`indices.DataByID`のような名前付き結果、または`IndexData()`等の責務別参照を比較する。旧APIとConsumer影響を測って選定する（#4472）。
- `GenerateCRUDTable`／`GenerateFunctionDataCrossReference`／`GenerateCRUDGaps` は、`CRUDDefinitions`の明示側と`FunctionElements.Operations`の導出側で列挙・優先・欠落判定が異なる。**Function×Dataの候補と根拠を保つ中間read-only view**を共通の入力とする案を比較し、衝突を後勝ち縮約しない。新しいCanonical CRUD正本を作るわけではない。
- 表の空欄、未取得、明示的なNo、矛盾、未検査を同じ`false`と読ませない。区別は入力契約と説明の両方で保持し、開示範囲を越える未公開の存在・件数を漏らさない。

これらは内部実装を固定する提案ではなく、より誤用しにくい代替を**独立Fixtureで比較するための選択肢**である。

### 16.4 最小の正負対照と段階判断

**P0／#4472**: Data/Function/Businessに名前を独立に設定した索引誤参照の正例、同一ID異種Kindの負例、明示CRUDのみ、Operation由来のみ、両方一致／競合、必要Read有／正当なWrite-only、構造OKだがCRUD矛盾ありのケースを試す。出力のName/ID、表のPair集合、Gap、Diagnostics、CLI/既存Consumer受領まで比較する。

**P1／#1124/#3250**: 同じ受理済みSubject/Scope/RevisionからData図・Function図・Business Procedure図・CRUD表を導き、モデル上の更新違反またはFunction不足を一件仕込む。Graphに出ていないことを「存在しない」に変えず、証明できないときはunresolvedを維持。Source参照と図面上のOccurrenceは別。

**P1／#3262/#2514/#1163**: AI/CUIとStudioで同じ問いを表示し、Candidate/Accepted/Observedを混同せず、元Source、独立Expected、必要集合と開示Scopeを一致させる。読める説明は`証明義務の根拠と結論`を短く表す。すべての詳細をスライドや一画面に羅列する必要はない。

**P2／実DB・Target Owner**: 注文確定時の保存単価、商品マスター改版、同じ版の画面／新規帳票、訂正の正規Operation、発行済み帳票の別Assetというケースで、Mockの表示一致ではなく、実Provider/Consumer/独立Observationを結ぶ。

段階の判定は、一件の現在のコード欠陥の修正から直接「三視点の製品全体が検証済み」へ広げない。逆に、現実装が未対応であることを理由に将来仕様の表現を貧弱にする必要もない。要求した価値に向けた最小の反例・受入へ戻す。

### 16.5 アウフヘーベンの判断

「三視点の個別ツールを高度化する」「AI向けContextを別実装する」「試験用Proof DBを別に作る」という三つの発想を足すのではなく、**各Consumerが同一のAccepted Subjectと問いに相対的な根拠へ戻れること**を共通原理にする。既存Proof Obligation、Source Index、Canonical/Shared Acceptance、Read-only Projection、共通Operation API、Conformanceを合成すれば、この原理の多くは新しいCore Kindなしに評価できる。

もし共通化によって専門家が扱う自然な記述順やNative SQLの表現力、GUIの安定した対象選択、Target固有の最適化が失われるなら、抽象化の方を修正する。共通化自体は目的ではない。

### 16.6 価値・要求の発見と、実装から独立した仕様の最適化

この設計の評価では、実装を「すでに成立している技術構造の証拠」として使う一方、将来仕様の上限や説明順の根拠とはしない。完成したコードのAPIに合わせてRequirementを後付けするのではなく、**利用者に必要な価値 → 検証可能な要求 → 最も少ない意味・契約境界 → Provider/Target実現 → 実証** の順で考える。

#### 価値の新発見から仕様へ

三視点のProjectorとCRUD列挙、Design Context Query、AI編集支援、独立Expectedは、個別機能の集合としても説明できる。しかし横断して観察すると、より高位の価値候補を検討できる。

> **価値候補 VQ-01（未採用）**：人とAIが、同じ対象・同じ変更について「何を根拠に、どこまで確認できたか」を共有し、複数の表現やTargetを横断してもその理解を失わない。

この価値が必要とする能力を、別の大規模基盤を作る前に既存契約へ分解して比較する。

| 必要な能力 | 第一候補として再利用する既存の責務 | 反例・選択条件 |
| --- | --- | --- |
| 同じ対象を特定 | Qualified Subject / Occurrence / Revision / Source reference | 同名異ID、同Subject別Occurrence、Source move/ABA |
| 同じ問いを固定 | Scopeに相対的なRequired集合、Proof Obligation / Claim | Required不足を成立と誤判定、問いの範囲外の矛盾で過剰拒否 |
| 根拠と結論を追う | Accepted Meaning、導出Graph、Diagnostics、Evidence参照 | Patternや類似度の推測をAcceptedに昇格、構造OKから業務整合OK |
| 消費者ごとに説明する | 図・CRUD表・CLI／Studio／AI Context Projection | 詳細を見せるために常識的な聴衆の理解を妨げる、一方で編集対象の資格が消える |
| 現実の結果と比較する | Driver Scenario、独立Expected、Provider受領・外部観測 | テスト生成器がExpectedも生成して自己証明、実行結果不明を成功へ変える |

**Requirementとして採用するかは未定**。既存RQ-026/RQ-036等がこの価値を既に十分に含むなら新しいRequirement IDは作らず、受入条件へ反映する。既存要求で表現できない実質差分が独立の例で判明した場合だけ、正式要求Ownerへ採用候補を提案する。

#### アウフヘーベンの比較基準

1. **保存すべきもの**：三視点Methodの明瞭さ、Native SQLと既存資産、DSLの専門家向け認知順、Targetごとの適正化、人間の安定したGUI選択。
2. **否定すべきもの**：機能ごとの第二の意味正本、図とAIと試験でのID・Source解決の重複、無根拠な万能OK、暗黙に最初/最後を選ぶ解決、検証できない「共通だから正しい」。
3. **統合して得るもの**：Qualified Subject・問い相対の根拠を共通にし、Methodごとの解釈/ProjectionとTargetごとの実現はPlugin等で可変にする。
4. **棄却条件**：共通化で結果の解釈（NULL・Coverage・粒度・Actor・Authority・Effect）が失われる、分野の表現力が落ちる、Source provenanceが切れる、機構が増えて利用者に説明できなくなる、独立Expectedで差が判別できない。

最小PoCは、注文確定のBusiness Function＋状態/CRUD矛盾と、測定値の単位・欠測・時間の異分野対照を使用する。同じ共通契約で両方を成立させられるか、どこでMethod固有のContractが必要かを記録する。**既存コードが現時点で失敗することだけを理由に理想仕様を縮小しない**。失敗箇所は実装修正・接続Gap・仕様候補のいずれかに分類し、別々のOwnerへ渡す。

#### 改訂計画への出力

- 既存要求内の記述改善と検証不足は当該Issueへ。
- 新たな価値・要求候補は、具体例と反証を添え、製品Requirement Ownerへ採否提案。
- 技術構造の改善は、対案・変換可能性・独立Expected・移行影響まで含める。
- 利用者向け説明では論理の順序を優先し、必要な章節参照だけを追加する。コードAPIの歴史的形状や複雑な検証内部は、読者が理解するために必要な範囲だけ示す。

新たなData Modeler、Proof Engine、Context Registry、Acceptance Authority等の重複した製品機構を、説明の都合だけで導入しない。

### 16.7 意味の同一性とScope相対の結論を、図表・AI・Runtimeで整合させる

実装の現状を将来仕様の上限にはしない。設計したい価値は、**同じSubjectを扱う人・AI・Targetが「その問いに対してどこまで言えるか」を同じ意味で説明できること**である。

ただし、`scope`が同じ文字列だからといって、以下を等号で結ばない。

1. **Identity/Meaning scope**：どのSubject・Occurrence・Accepted Modelの話か（既存#1120）。
2. **Required/Query scope**：判断に必要な集合・粒度・時点・依存は何か（既存#3262/#2506）。
3. **Observed/Coverage scope**：実Providerが読んだ範囲、部分／完全性、単一Attemptは何か（既存#2586/#1163）。
4. **Disclosure scope**：Consumerへ何を開示できるか。見えない存在・件数の推測も禁止条件に含む（既存#2516）。
5. **Authority/Effect scope**：現在どのCallerとTargetがどの変更・副作用を許されているか（既存#1174）。

これは五つのCore型を必須化する提案ではなく、**異なる正本と判定責務の対応関係**である。新しいContext Registry/Permission Engine/Proof DBを作らず、既存の対象参照・Query・Evidenceを使い、主張の用途に必要なScopeだけを照合する。

#### 判別の最小例（未実行の計画）

| 対照 | 想定入力 | 必要な結論 |
| --- | --- | --- |
| SCQ-01 正常0件 | 現在Scopeで対象を完全取得し、RLSやCursor truncationなし、0件を独立観測 | **その問合せScope内の0件**。別Tenant・過去時点・未列挙世界の不存在は主張しない |
| SCQ-02 非開示／未読 | 同じ意味Subject、部分取得、または許可されない属性・Tenantがある | 非存在と断定せず、開示できる状態と診断を返す。秘匿した対象の存在／件数を漏らさない |
| SCQ-03 読めるが書けない | 図・AI ContextへSubjectは表示できるが、現在Edit/Host Grantを持たない | NavigationとReadは可、Edit Intentの適用は不可。図上のSelectionをAuthorityにしない |
| SCQ-04 合成で適格性が失われる | D→Representation→AI概要へ変換し、途中で粒度・欠測・Subject Refが失われる | 各表示が成功していても、元Dataの業務不存在／一意な逆編集／全体Coverageは主張しない |

評価順は「対象と問いを特定→Requiredな意味とSourceを解決→ScopeごとのObserved/Disclosed/Authorizedを照合→Supported/Unresolved/Contradictionを用途相対に返す」。固定された単一巨大Scopeや、普遍的な「全世界で完全」という表示は採らない。

**今後の仕様最適化の問い**：既存のScope参照、Function Data Context、Read-only Projection、Proof Obligationを繋ぐだけでSCQ-01〜04を正しく説明できるか。できなければ欠ける参照・必要な不変条件だけを追加し、既存Projector/Context/Permissionの第二正本は作らない。結果は#3262/#2497/#2516/#1124と共有する。コード移行・実Provider/GUI試験・Owner意味受理は本節では実施していない。

### 16.8 確定Snapshotから実消費までの整合を、問いに相対的に評価する

複数のProof Obligationが局所的にsupportedでも、**同じ業務結果について共同で成立した**とは限らない。設計の正しい構造と、実際の結果の成立を次の順に分ける（製品Schemaの追加ではなく、候補としての設計・検証票）。

1. **意思・対象**：Acceptedな業務Action/Function、Business Outcome、対象集合とIdentity、Scope、版、処理意図。
2. **一貫した入力**：必要なHeader/Detail/規則/時点/版/ProviderのRead cohortと取得閉包。別々のRead PASSや同じSnapshot Token文字列を、共通の一貫した観測と決めつけない。
3. **実作用と生成**：対象/Grant/Provider/Artifactを固定し、実際のRequest、保存した実体、外部Effectを識別。Retry/Unknown時に実未実行とみなさない。
4. **配布・受領**：発行レコード、保存済みArtifact Set、配信受付、受領した実Byteと必要Closureを区別。Manifest内整合と公開途中の混合版/旧版残存を別検査。
5. **Businessとしての結末**：独立Readや外部のObservationとAcceptedな完了条件を比較し、業務Outcomeとして言える範囲を限定する。

#### 重要な共同反例

| ケース | 各局所検査で一見成立すること | 共同Claimとして成立しないこと |
| --- | --- | --- |
| RC-01 別cohortで帳票確定 | Header Query成功、明細Query成功、PDF Renderer成功 | 両Readを一つの業務的に整合したSnapshotとして固定した根拠 |
| RC-02 発行直前の障害 | PDF実体は保存済み | 発行を正式に記録した／相手が配信を受けた |
| RC-03 複数媒体 | CLI exit 0、HTTP 200、Web表示成功 | 両経路の同じ業務Effect／独立観測結果 |
| RC-04 成果物一式の切替 | ManifestのContent digest一致、Generator PASS | Consumerが一式を同じ公開世代として取得し使用した |
| RC-05 ACK喪失と再送 | Requestは送った、応答は失った | 未実行だった、再送しても二重作用が無い、業務完了した |

ここでいうSnapshotは物理DBの単一Isolation方式ではない。業務Ruleに照らした確定済みイベントや再確認を含め、**用途に必要な集合の整合**を構成できることが重要。複数DB/非同期APIにまたがる原子性は、独立に採用された取引・回復方式でのみ認める。Snapshot/Artifact/Receiptが同じ型や表名である必要もない。

#### 統合判断（アウフヘーベン）

帳票、Build成果物公開、通知、外部Commandは各々固有の状態と回復能力を持つ。すべてを同じ新しい「発行Runtime」へ入れるのではなく、**Intent／Required Inputs／Actual Invocation・Produced Artifact／Observed Result／Owner Accepted Outcome**という照合の論理を共有する。設計・業務意味・実行主体・配布先のOwnerを消さない。Generated evidenceだけの自己証明を避け、受領者側の独立Observationを必要なケースで採る。

追加の共通構造が有益かは、既存#1163 Evidence Packet、#1174 Effect Fence/Receipt、#2604 Artifact Set publication、#1130 Batch/Report、#3815 Snapshot帳票、#2514 Independent ExpectedをつなぐRead-only Referenceで反証してから判断する。新Host、Second Receipt Store、中央Publication Registry、万能Transactionを先行採用しない。

**検収の出力**：対象・問い・Required集合→実Read cohort/Provider→不変なArtifact/Effect→相手に届いた実物→業務完了の判断、の各辺に根拠と未検証を記録する。最小例は注文確定単価1,200円に対する注文Header/Detailの版ずれ、帳票保存後の発行記録失敗、別媒体/再送時のUnknown。これは設計提案であり、今回の新規実CLI/DB/印刷/配信試験はNOT RUN。


### 16.9 Derived IndexとFingerprintはAccepted Subjectと資格を変更しない（2026-11-12 追加評価）

モデルの説明・図・AI Context・CRUD表がすべて同じSubjectを指すためには、単に`kind/id`のRefが揃うだけでなく、**参照を導出する過程で入力の同一性や状態が破壊されないこと**が前提となる。

現行公開main `718a8b74`の`internal/threeelement/model.go`では、次のSource-level反例がある（実Go Test・CLI再現は今回NOT RUN）。

- **ID collision**：`Index()`はData/Function/Businessを各mapに`data[e.ID]=e`等で格納する。kind内の二つの要素が同じIDなら後勝ち縮約されるが、`Validate()`の構造診断にkind内ID重複検査がない。`SameNameConflicts()`は名前の重複であり、同ID/異名を検出しない。
- **Derived fingerprint mutates source**：`ModelFingerprint(model ThreeElementModel)`は値コピーされたStructの`DataElements[i].AuthorityLimits=nil`等を実行し、Sliceの共有Backing Arrayを通じて呼出し元の要素Fieldを消す。ハッシュを求める読取り専用処理が入力モデルの状態に副作用を起こす構造。
- **Self-reference**：`json.Marshal(model)`に`model.ModelFingerprint`自身が含まれるため、先に計算して同Fieldへ格納すると、内容が実質同じでも次の計算は別Digestになる。既存`model_test.go`は空Fingerprint状態同士の比較のみ。

この実装上の欠落は公開[Issue #4475](https://github.com/hat47x/tei-platform/issues/4475)へ限定委譲し、#4472のCRUD上のData索引誤用と区別する。

**望ましい仕様の成立条件**：Stable/Qualified Subjectは同Kind同ID衝突を診断し、CrossKind同IDの合法性はKind+Scope契約に従う。Index/Graph/CRUD/AI Context/Fingerprintは原モデルを変えずに計算でき、派生結果を格納しただけで内容Digestが変化しない。モデルの意味的なFingerprintを変える入力と、Derived Diagnostic・現在Authority/Grant・実観測を変える入力は混同しない。現時点の`ThreeElementModel`型・3戻り値・JSON Field構成を将来仕様の制限として採用しない。

**判別例・所有順**：TE-ID-01〜06（#4475）の正負を先に固定し、Input ID衝突が未解決なら「この図のSubjectは一意」とは主張しない。純粋なFingerprint計算は同一Meaning+異AuthorityLimitsでの値と現在資格を別に表示。実装内部の索引改善は#4475、3視点Projectionの利用条件は#1124、Consumerでの問い相対の適格性は本Design Synthesis。新Identity/Permission Registryを先行追加しない。

### 16.10 一つの可読なSourceからの参照閉包（2026-11-12 追加評価）

主線の注文確定単価と、異分野対照の測定値を、単なる例の数ではなく**同じ意味を通す一つの縦断構造**として検査する。

- Input/Source: 人が読めるDomain定義、型・Rule、Query Native資産、Function、Scenarioを明示。説明上のSyntax Sugarが複数あっても`Language Pin→同じMeaning`への変換条件を保持（#4359/#988/#2621）。
- Resolved: すべてのType/Result/Contract/Provider参照が確定、または未確定な理由が説明できる。未宣言のResult TypeやLegacy Namespaceを暗黙に補わない。
- Actual: 実行Query/Raw結果/Mappingの物理DB・Input/Result・粒度/時点からData Context、表示/Action、独立Read/Expectedへ、同一のSource/Subject/Revisionを追える（#2586/#2611/#2506）。
- Proof: 相互参照を表示できるだけで正常終了とせず、Source未取得／別物理DB／未宣言列／同ID衝突／部分Context／Authority差を一因子ずつ変えて結論の適格性が変わるか確認する。

これは3視点以外のDomainにも通じる設計上の**可読な最小例**の条件であり、現在のScript文法・Physical Queryを正本にするものではない。Domain expertの認知順を優先し、関係する契約が増える場合は標準Recipeで隠蔽できる複雑さと専門家が明示すべき判断を分ける。現実装で使えない構文を理由により優れた表現案を棄却しない。一方で、その構文が既存Parserで使えると主張しない。



### 16.11 GUI/CUI共通の安定した対象固定と、問い相対のContext Tray（他チャット11/10〜12の成熟差分）

**狙う利用価値**：利用者が三視点図、CRUD表、Source、規則、SHO文書、SUIのCard等を行き来しながら、**今どの要素／関係／範囲についてAIと話しているかを失わない**。選択を自然に保ちつつ、違う対象・版へ誤適用しない。これは既存§16の「同じ問いに戻れる」を利用者操作で実現するための条件であり、新しい意味正本・ハーネス常駐Host・Session Registryの採用を意味しない。

#### 三層の選択を混ぜない

| 層 | 意味 | 変更される操作と不変条件 |
| --- | --- | --- |
| **Focus** | キーボード・マウス・入力欄の現在位置 | 移動、別窓のクリック、AI入力欄へのFocus移動で変わる。**Pinned Targetを自動解除しない** |
| **Transient Selection** | 現画面の一時的な行・Node・範囲・複数Occurrence | 範囲選択やキャンバス移動で変わる。Pinの候補だが**それだけではSource編集対象／権限を確定しない** |
| **Pinned Target／Task Context** | 利用者が明示固定したQualified Subject・Relation/Cell・Source occurrence・関係・問い・用途・Scopeの組 | 明示的なPin/追加/置換/解除、またはSource改版に伴う**再資格化要求**で管理。クリック外し、Canvas切替、チャット入力では消さない。旧参照が失効しても黙って別対象へ再束縛しない |

Pinned Targetは**機能上の保持参照**であってAccepted Meaning、現在Host Grant、外部AIへの開示許可ではない。UI上の強調表示を完全性証拠としない。長い作業で複数Targetを固定する場合は同一Kind+IDを重複して意味対象へ増殖させず、**意味対象の集合と描画Occurrenceの集合**を別扱いにする。関係Edge、CRUD Cell、SourceのByte Span、Graphの複数Occurrenceは可逆性が異なるため、Node/Cellの区別を消さない。

#### 最小の共有Context Envelope（正式Schema未採用）

新しい公開`ContextPacket`型を増やすのではなく、既存`model locate`、Typed Model Map、`Catalog/Inspect/Trace`、#3262のDesign Context Query、#2516の用途別Disclosureの**既存結果を参照**して、利用者が作業対象を確認できるread-only投影を比較する。

- **Target**：Product・Workspace identity、Qualified Kind/ID、Relation Instance ID／Function×Data Pairと宣言Occurrence、View/Canvas・表示Occurrence、必要ならEditor座標と原Source byte範囲。**同名や同位置だけでは一致判定しない**。
- **Question**：利用者が指示した変更意図・参照対象、Required/Query Scope・Actor/時点、同じ問の許容Operation（inspect／propose／preview等）。適格性不明の変更操作は提案段階で止める。
- **Snapshot/Completeness**：World／Source Index／Model Revision、Source/外部Provider/Plugin versionの観測状態、Required・loaded・returned・omitted、Unknown/Unresolved、bounded page・continuation/end、同時に観測できた範囲。Fingerprint一致だけでSource外の依存やA→B→A履歴を保証しない。
- **Disclosure/Effect**：現在のAudience/Consumerへの開示可能な投影と、禁止・未検証の項目。現在Edit Grant/Meaning Acceptanceは**Apply直前**に別のOwnerで再検証。AIへの送出前に権限・Raw/件数の漏洩を点検し、送信していない部分の存在は推測させない。
- **Lifecycle**：Pin作成元、User明示操作、対象Revision、問い合わせごとのRequest/Response correlation、後着結果の採否、Preview/Acceptance/Apply/Receiptの独立状態。UIには「保持中」「再確認必要」「解決不能」を区別し、裏で別対象へ置換しない。

**CUIの同等性**は、ユーザーが明示指定した`Kind+ID+Workspace+Scope`をテキスト経由でも保持し、後続のread-only操作が同じContextで解決できること。CLIの新flagや永続Sessionファイルを先に必須化しない。既存`model locate`の`limit/total/omitted/coverage.notDerived`と、M0の`partial-projection`を利用し、近傍が省略されても「全件なし」とは答えない。CUIからTargetが複数選ばれた場合も、意図した集合と順序を再現し、曖昧さをFirst-Winsにしない。

#### 複数製品・AIハーネスへのハンドオフ

TEI/SUI/SEI/SHO/CSWが共通のHarness Sessionを利用するとき、**共有するのは作業対象の参照・問い・復帰位置・選択の履歴の必要部分**。各製品の正本Source・Meaning・Host Grant・DisclosureをSessionへ移管しない。AIハーネスは共通Application OperationのConsumer/Orchestratorで、DSH／Codex／Claude Code等の実装差はAdapterの機能宣言で示す。TEI/SHOの画面に特定Harnessを直接組み込むことを必須としない。

CSW/SUIのKJ法的なCard/Islandの集合をTEI Relationへ投影するときは、**多対一・部分集合・表示順・提案Relation**の情報落ちを記録する。図を介した遷移で業務Relation IDが一意に確定しない場合はread-only。SHOの文字選択ではUTF-16表示位置と原文UTF-8 Byte Span・dirty bufferを別に扱う。図面/文書を跨いだ選択でも「同じPinだからSource Write可」とはしない。

### 16.12 Pinの安定性と失効の敵対的受入（提案・未実行）

ここでは「選択が解除されない便利さ」と「無効な選択へ誤作用しない安全性」を**同時**に評価する。既存#1124/#1452/#3262/#2514の検収に以下の正負対照を引き渡し、第二Test Runner/Proof Storeは作らない。

| ID | 変更する一要因 | 合格条件 |
| --- | --- | --- |
| PIN-01 | 図のRelation RをPin→AI入力欄をFocus→別Canvasへ切替 | Pinは保持。FocusやTransient Selectionが変わっても、操作対象はRのまま |
| PIN-02 | 同じSubjectを2 Canvasの3 Occurrenceに表示 | 意味対象は1件、表示Occurrenceは3件。どれを操作したかが復元可能 |
| PIN-03 | 同端点・同種別だが別Relation IDの2辺 | Edgeの見た目や列挙順で自動選択しない。Identity不明ならApply拒否 |
| PIN-04 | Pin作成後にSource移動／Byte改訂／A→B→A | 保持中の対象を勝手に置換しない。`needs-requalification`として再取得・Revision/LocatorとAuthorityを再検証 |
| PIN-05 | Source不変、開示Grantだけ失効 | 表示・Tool送信・既キャッシュの資格を再判定。Pin保持と外部送信可否を分離 |
| PIN-06 | 関連要素100件のうち`limit=1`・Provider必要Source未観測 | 残件・notDerived・Source外依存をUnknown/omittedとし、全100件が見えた・不存在と主張しない |
| PIN-07 | AのInspect開始→BへPin変更→Aの応答後着 | BをAで上書きしない。Requestに束縛したAの結果を、現在Bの結果として表示しない |
| PIN-08 | 複数Pinned Cellを1まとめにAIへ渡す | 各(Function,Data,宣言Occurrence)とUnknown/Conflictを保持。CRUD衝突を後勝ちで消さない |
| PIN-09 | SHOで保存先だけ変更、原Markdown Byteは同じ | 相対Link/Asset解決の再確認が必要。TEI MeaningやSourceは自動改版しない |
| PIN-10 | 同一Harness Sessionが別製品へ遷移 | 参照/問い/復帰位置を維持できるが、新Productの開示・現在Grantを継承したとみなさない |
| PIN-11 | 明示Unpin→Undoで復帰 | Undoは**対象参照の復元**まで。旧Grant・旧Revision・旧AI Replyの自動再許可なし |
| PIN-12 | Read-only Context内でAI変更候補が生成 | Preview/Source Diffは可能でもAccepted/Applied/Observedを自称せず、Shared Acceptanceと再読込に進む |

**採否を決める比較**：同じScenarioと独立Expectedで (A)一時Selectionだけ、(B)Pinned Target＋限定Context、(C)HarnessのSession再利用を比較。選択の再指定回数・対象誤認率・Context漏洩／未解決の明瞭さ・操作の再現性とSource再資格化を測る。作業の容易さのために権限・受理・依存完全性の条件を緩めない。最小PoCは既存`sales.order.lines`と二Canvas、同名異ID Relation、Studio→SHOの参照1件で充分。初回から本番AI／全Pluginを接続する必要はない。

**Owner/実装順**：
- #1124：Modeler内のPin保持、選択Scope・Relation/CRUD Cell Source、read-only Previewと逆写像の可否。
- #1452/#1824：実Target/Studio GUIでのFocus/Selection・旧応答制御、AI入力からの元View復帰。
- #3262/#2516：用途相対ContextのSource閉包／Unknownと現在Disclosure。特に[Draft #4329](https://github.com/hat47x/tei-platform/pull/4329)のConsumer FP外のSource依存を`unknown`で止める結果を共有する。
- #4472/#4475：CRUDの明示宣言・索引欠落と同Kind ID衝突、Fingerprint純粋性を先に修正・検証。**未修正ModelをPinの正しいIdentityとして前提にしない**。
- #2586/#1251：実Function→Provider Readと正式Plugin実Invokeを最優先に維持。GUI PinとM0/LocateはこのP0の新しいHard Gateではない。
- #2514/#1163：独立ExpectedとSource/Provider/Consumerの版・Attemptの照合。実Result/Effectの完成判定は別Owner。
- SHO #19、SUI/CSW/TAI各Owner：共有Session参照とProduct-local Meaning/Authorityの分離。

**状態**：本追加は設計・負例・Owner引渡しのみ。`Pin UI`／`Context Tray`／新CUIコマンド／外部Harnessの実Session連携／Studio・SHO・SUI/CSW実E2E／正式Owner受理は**NOT RUN／NOT PROVEN**。既存モデルの読取りGo・Source Index・M0/Locateの限定成功を、このProduct UXの完成証拠に転用しない。
