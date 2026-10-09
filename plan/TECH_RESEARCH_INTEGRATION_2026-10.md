# TEI 技術リサーチ統合方針 — 2026-10

Status: **current research-integration plan**  
Last updated: **2026-10-09**

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

## 9. 2026-10-09 追加調査 — 実用上の価値へ結び付ける

この追補は**採用前の設計・実証候補**である。外部研究の成績をTEIの達成度に読み替えず、上記の統合方針、既存Requirement、受入境界を変更しない。まず、利用者が「何を安心して変更できるか」「何が本当に動くか」を確かめるための小さな比較を追加する。

### 9.1 仕様の不足を、実装前に発見できる価値

[SpecBench（2026-05）](https://arxiv.org/abs/2605.30314)は、コードの修正能力とは別に、仕様案に残る**欠落・曖昧さ・不整合・誤った前提**の検出能力を評価している。既存の設計議論とコードの文脈を持っていても、仕様上の不足を確実に見つけられるわけではない。

[RADIANT（2026-07）](https://arxiv.org/abs/2607.16708)は、要求から複数のモデルを生成し、要素単位の追跡関係と変更影響の計算を接続する研究である。構文上の妥当性と意味の正しさが同じではない点も重要である。形式検証の方式自体をTEIへ輸入するのではなく、異なる設計視点を通した**反証可能な変更影響**に注目する。

TEIへの翻訳:

- RQ-026のData、Function / Interaction、Business Procedureの相互参照から、要求の欠落・矛盾・不明点を実装前に提示する。
- RQ-028では、レビュー対象を生成コードの一覧ではなく、影響を受ける規則・操作・データ・利用者権限・未解決点として提示する。
- 既存の #982、#998、ISSUE-000075、#915、#267へ割り当てる。正本は要求・業務意味・受理されたSourceに保ち、AIが見つけた問題は確認候補として扱う。

**REF-SPEC-01（候補）:** order_entryの既存Customer参照規則を題材に、欠落（必要な参照先がない）、曖昧（更新できるかが不明）、不整合（read-onlyと更新要求の衝突）、古い前提（基準revision変更）の四ケースを用意する。同じ変更案を、(a)通常のSource探索、(b)TEIの依存・差分・未解決表示の二条件でレビューする。重要な不足の見落とし、誤警報、人が開いたSource数、説明可能な影響範囲、確認時間を分けて測る。#915の既存Referenceにない事実や権限は推測で補完しない。

### 9.2 見た目ではなく「操作が働く」価値

[VISTA（2026-09-29改訂）](https://arxiv.org/abs/2605.26144)は、画面を描画できたことと、必要な部品を見つけて実際に操作できることを別々に測る。TEIでも既存Gate Cの視覚評価を、機能や人による承認の代替にはできない。

TEIへの翻訳:

- RQ-013、RQ-010、RQ-021の評価では、部品が存在すること、操作が正しいBusiness Actionへつながること、拒否されるべき操作が拒否されること、実世界の状態を観測できることを別々に検証する。
- #997、#1000、#1001、#259、ISSUE-000046の既存UI・Target検証へ戻す。画面の見た目が一致しても、Business ResultやObservationが一致しなければ成功にしない。

**REF-INTERACTION-01（候補）:** 第一Web Target上で、顧客参照、入力検証、許可された更新操作、許可されない更新要求のうち、既存Referenceで実行可能な一連の操作を選ぶ。各操作についてUI部品→イベント→意味上の操作→Driver結果→Observationへの対応を記録する。キーボード操作・エラー通知はUI固有の評価として加える。視覚評価、人の操作確認、決定論的な操作結果を独立して記録し、人による検証を実施していない場合は「未実施」と明記する。

### 9.3 小さな実行物を、標準機能のプラグイン化と接続する価値

[WASI 0.3（2026-06-11公開、0.3.1は同年08-11）](https://github.com/bytecodealliance/wasi.dev/blob/main/docs/releases/wasi-p3.md)ではComponent Modelに非同期のfuture、stream等が導入された。一方、Runtimeと各言語のtoolchainは同じ速度で対応していない。WASI 0.3が標準化されたことは、TEIのGo製Sourceをそのまま全Hostへ移せることを意味しない。

[Wails v3（2026-08時点でBeta、2026-10時点でもBeta）](https://v3.wails.io/status/)は、GoとWebViewを組み合わせたDesktop候補として、Hostのライフサイクル・マルチウィンドウ・bindingを調べる材料になる。ただし、Studio Host、生成された業務Application Target、Plugin Runtimeは別の責務である。

[TinyGo](https://github.com/tinygo-org/tinygo)はGo SourceをWasm/WASIへコンパイルする候補である。[Hike](https://github.com/kanryu/hike-lang)はGo風の構文を持つ**別言語**であり、一般のGo SourceをそのままHikeでコンパイルできるという前提を置かない。小さい実行物の比較では、この違いを明記する。

TEIへの翻訳:

- #259の第二Target比較に「実行物の形式」「Hostが提供するimport」「ABIと呼出規約」「同期・非同期」「権限と資源」「起動・回復」「実行物の大きさ」「可搬性」を加える。
- #943、#973、#976、#1016では、標準機能を含むプラグインの意味契約・依存解決・Build Artifact Lockを先に保ち、Wasm形式やコンパイラ名をPlugin identityへ混入させない。
- Wasmの採用そのものを第二Application Target達成の証拠とみなさない。Web/スマートフォンのUI実装とも分ける。

**REF-PORTABLE-01（候補）:** 副作用のない小さな意味上の変換を同一入力・期待出力で定義し、利用可能なGo native / TinyGo Wasm等で比較する。Hikeは別Sourceへの移植コストも記録できる場合に限り候補に加える。生成物digest、依存、Host imports、ABI、失敗・取消、CPU・メモリ、初回起動時間、結果の一致を記録する。WASI 0.2と0.3はtoolchain/Hostの対応状況を確認して別条件とし、片方の成功をもう片方へ一般化しない。実行Hostのpermission、resource、leaseは既存のAuthority境界へ束縛し、契約の読み込みだけで実行権限を付与しない。

### 9.4 優先度と着手条件

| 順位 | まず確かめたい利用者価値 | 既存の受け皿 | 続行・見送りの判断 |
| --- | --- | --- | --- |
| 1 | 小さい業務変更で、見落としや矛盾を人が判断できる | ISSUE-000075 / #982 / #998 / #267 | REF-SPEC-01を既存Sourceと受理境界で示せるか。広い仕様生成や新規基盤が必須なら見送る |
| 2 | 画面の各操作が業務上の結果につながる | #997 / #1000 / #259 | REF-INTERACTION-01で機能・視覚・観測を別々に測れるか |
| 3 | 実行環境を替えても、同じ意味と権限条件を保てる | #259 / #943 / #973 / #976 | REF-PORTABLE-01でexact Host/ABIを記録し、既存権限機構を再利用できるか |

この順番は既存Issueの着手承認・統合手順・実装優先順位を上書きしない。ISSUE-000075の候補Bでは、2026-10-08に業務規則の変更が受理されている。ただし、Web画面まで含めた一周の実証や全明細への規則適用を完了したわけではない。REF-SPEC-01はこの受理をやり直す作業ではなく、受理済みの意味と別の設計視点に残る差を読み取り専用で検査する候補である。

**完了の扱い:** この文書の更新は研究の計画への反映にとどまる。REFの実装、比較実験、利用者レビュー、性能測定、適合性の判定は未実施である。どの候補でも、baseline revision、Source、実行環境、期待結果、実測値、反例を残せたときに初めて対応RequirementのEvidence更新を検討する。

### 9.5 REF-SPEC-01の最初の実施手順（静的確認済み／コマンド実行未確認）

第一段階では、新しい仕様生成器を作らない。既存の受注ReferenceとBusiness Procedure checkerを使い、**確認できる問題と、まだ判定できない問題**を分ける。2026-10-09にリポジトリのSourceを読み取り、次の境界を確認した。以下のコマンドの実行結果や時間計測は取得していない。

| 確認対象 | Source上で確認できる事実 | 実行時に確かめること |
| --- | --- | --- |
| 明示された規則 | 受注ProcedureのBP-05には数量の正値と行番号の一意性という二つのValidation参照がある | 現行checkerがどの参照を解決し、どの参照を未解決として返すか |
| 反する更新要求 | Customer参照のBP-02に対する編集要求のprobeはread-only datasourceにread-writeを求める | 期待するアクセス衝突だけを検出し、更新権限を推測で作らないか |
| 三視点間の差 | 2026-10-08に受理した行番号の正値規則はApplication側の三つ目のValidationである。一方、Business ProcedureのBP-05は二つの参照のままである | 意図した独立性か変更影響の見落としかを人が判断できるか。checkerが必ず不整合判定するとは仮定しない |
| 古い基準版の提案 | 候補Bの保存済みcandidateは受理前のbaseModelRevisionを指す | 現行版へ黙って適用しないか。旧Reviewを現在の受理根拠に流用しないか |
| Source上の不在 | Source検索で記述が見つからないだけでは、禁止・不存在を証明できない | 必須参照先の契約違反と未発見を区別できるか |

読み取り専用の再現入口:

    go run ./tools/reference-business-procedure-check

    go run ./tools/reference-business-procedure-check -probe reference/business-system-design/order-entry-customer-edit-conflict.json -expect-diagnostic BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT

既存の説明は [Business-system design Reference](../reference/business-system-design/README.md)、受理履歴は [ISSUE-000075](issue/000075-practical-scenario-mvp-demonstration.md)を参照する。過去のcandidate／preview／reviewファイルは**当時の基準版に対する証拠**であり、現在のWorldへ再受理可能という証明ではない。

最初に作る読取結果は、(1) exact Source・意味identity・衝突根拠から**判定できた事項**、(2) 不在・古い前提・Procedure固有の意図など**人の判断が残る事項**、(3) 実Provider・画面・未実行のDriverなど**根拠が届かない事項**、(4) 変更を提案するか保留・追加調査するかという**人が選ぶ事項**を分ける。

**停止条件:** Customer編集の衝突を既存checkerで再現できない場合、または別視点の差を示すために受理済み業務定義を勝手に書き換える必要がある場合は、実装を増やさず原因を既存Issueへ返す。成功を記録する際は、exact revision、実行環境、コマンド、診断、実行前後のSource不変を残す。今回の静的Source確認だけでは成功扱いにしない。
