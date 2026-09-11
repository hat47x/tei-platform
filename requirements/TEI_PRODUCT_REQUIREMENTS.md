# TEI Product Requirements

## この文書の位置づけ

この文書は、TEI（鼎／テイ）のProduct Value、Vision、Principlesを、プロダクトとして満たすべき要求へ落とします。

ここでは「現在すでに実装されているか」と「プロダクトとして必要か」を分けます。実装状況は`docs/status.md`、個別の設計判断はADR、作業単位はIssueで管理します。

要求番号は、将来文書が分割されても参照しやすいように固定IDとして扱います。

## RQ-001 業務上の意味を実装技術から独立して保持できること

TEIは、業務概念、型、関係、制約、lifecycle等を、特定の画面、Database、言語、配備先と同一視せずに表現できなければなりません。

Canonical IDはfile pathやphysical object nameから独立し、実装や配置を変更しても意味上のidentityを維持できる必要があります。

## RQ-002 Authoring sourceとCanonical meaningを追跡できること

`.tei`、SQL、将来の他形式を含むauthoring sourceから、どのCanonical subjectへ寄与したかを追跡できなければなりません。

source locator、fingerprint、range、roleはCanonical Worldへ無条件に混ぜず、source-sideの追跡情報として扱います。

file moveやrenameだけでCanonical meaningが変わらないことを検証できる必要があります。

## RQ-003 一つの関心事に暗黙の複数正本を作らないこと

同じ意味を複数のauthoritative sourceからheuristicにmergeしてはいけません。

SQL-authored、Graph-authored等、authoring方式ごとに正本を明示でき、derived artifactからauthoritative sourceへ暗黙に逆書きしない必要があります。

## RQ-004 Native assetをnativeなまま利用できること

SQL、Go、外部設定、既存実装等を、TEIを利用するためだけに全面的に独自DSLへ移し替えることを要求してはいけません。

TEIは、それらを意味、Capability、Realization、Source Indexへ接続し、どの役割で使われているかを追跡できる必要があります。

## RQ-005 ApplicationとDatabaseをN:Mで扱えること

一つのApplicationは複数Databaseを利用でき、Databaseは複数ApplicationへContractを提供できなければなりません。

各Database利用について、少なくとも次を独立して扱える必要があります。

- logical datasource
- Database Contract
- Environment Binding
- read / read-write等の利用形態
- release / migration lifecycle
- transaction boundary
- failure boundary

Application ReleaseとDatabase Releaseは独立したlifecycleを持ちます。

## RQ-006 Desired / Actual / Diff / Unresolvedを分離できること

Database等の外部状態について、望ましい状態、観測された状態、確定した差分、比較未確定を区別できなければなりません。

比較に十分な情報がない場合は、差分なしや破壊的変更へ自動変換せず、Unresolvedとして扱う必要があります。

## RQ-007 Provider固有知識をCoreへ無制限に持ち込まないこと

Database、Runtime、Target等の製品固有知識は、Provider / Adapter / Capability等の境界へ置きます。

provider-neutralな意味とprovider固有の実現・正規化を分離し、offlineで確定できるpolicyとactual connectionが必要な処理も区別できる必要があります。

## RQ-008 Capability単位で標準実装と固有実装を置き換えられること

認証、認可、外部連携、帳票、処理等について、TEI標準だけを強制してはいけません。

Capabilityごとにbuiltin、generated implementation、plugin、既存実装等をBindingでき、必要なCapabilityだけを置き換えられる構造を持つ必要があります。

標準名前空間として`std.*`を予約し、Plugin / packageの名前衝突を避けられる必要があります。

## RQ-009 既存システムの認証・認可仕様を保持した移行が可能であること

ユーザーID、password、無効状態、有効期限、失敗回数、独自ロック、組織情報等を含む既存認証仕様を、移行時にTEI標準へ変更することを必須にしてはいけません。

認証・認可はCapability契約とMappingを通して既存構造へ接続でき、後から必要な単位で置き換えられる必要があります。

## RQ-010 Editorを開発者向けの高密度な設計環境として提供できること

Editorは、簡略化されたフォーム入力だけではなく、複雑な業務システムを一画面で見渡しやすい高密度なProjectionを提供できることを目指します。

Data、Relation、Rule、View、Procedure等を分断された多数の画面へ追い出すのではなく、必要に応じて3階層以上の関係も局所性を保って確認できる設計を可能にします。

Editor自身をCanonical source of truthとはしません。

## RQ-011 同じCanonical meaningを複数のProjectionで扱えること

少なくとも、次のProjectionを同じ上位設計から扱える構造を持つ必要があります。

- DSL / source view
- table / matrix
- graph
- flow
- UI
- natural language
- database workbench
- diagnostic / evidence view

Projection間の切り替えで意味が暗黙に変化してはいけません。

## RQ-012 Driverを設計検証用の実行・シミュレーション環境として利用できること

Driverは、本番Runtimeだけでなく、設計途中のApplicationを動かし、Data、Rule、UI、Function、Database Contract等の整合性を確認する環境として利用できる必要があります。

設計内容をコード生成後まで待たずに検証できることを重視します。

## RQ-013 Target-neutralなPresentation / Interaction境界を持つこと

Web、Desktop、Mobile、API、MCP、VUI等を、Coreへ個別実装を埋め込まず追加できる必要があります。

新しいTargetは独立したAdapter / Packageとして追加でき、Core変更なしで接続可能であることを基本とします。

Targetごとの差はCapability negotiationやRuntime Binding等で明示し、Conformanceによって共通意味が保たれていることを確認します。

Targetが「何を表示・操作できるか」というTarget Capabilityと、「どのtransport・信頼境界・session・Resource条件で実行されるか」というExecution / Trust Profileを、実装都合だけで一つのTarget差へ畳んではいけません。必要な分離は第一Targetと第二TargetのReferenceから検証します。

Target / Hostがinspectionやdevelopment controlのために公開するMCP等のcontrol interfaceと、Application自身が業務上のPresentation / InteractionをMCP Targetとして公開する場合も区別できる必要があります。

## RQ-014 標準的なApplication Patternを再利用できること

browse、record、reference、header-detail、command等、業務システムで頻出するApplication Patternを再利用可能な知識として保持できることを目指します。

標準PatternからCRUD、View、Validation等を導出できる一方、適用できない業務へ無理に当てはめない必要があります。

## RQ-015 生成AIが既存知識を探索してから新規実装へ進めること

生成AI向けの開発支援では、次の順序を基本とします。

1. Existing Domain Component
2. Existing Semantic Pattern / Facet
3. Existing Policy / Variation
4. Composition
5. Declarative expression / rule
6. Capability call
7. Custom implementation

毎回ゼロからコードを生成することを標準経路にはしません。

AIが作った内容を直接Canonical authorityとせず、まずProposal / Structured Intent等の候補として扱い、採用時にはEditor / CLI等と共有できる決定論的なValidation / Acceptance境界を通す必要があります。

構造化された候補から決定論的に生成・変換できる部分は、同じ意味を毎回自由生成する方法より優先します。

過去の成功Proposal、修正trajectory、利用例等を再利用する場合も、Canonical meaningの第二の正本にはせず、元のCanonical ID / Source / Evidenceへ戻れるderived knowledgeとして扱う必要があります。

## RQ-016 AIによる確認提案は控えめで、履歴を持つこと

AI / Editorは、履歴管理、監査、状態遷移、1:N関係、関数従属性、認証・認可等の典型論点について、必要な場合に確認を提案できることを目指します。

ただし、次を満たす必要があります。

- 一度回答した質問を理由なく繰り返さない
- 過去の選択と理由を後から追跡できる
- 選択を後から変更できる
- 不要な提案を無視できる
- 提案カテゴリを抑制・無効化できる
- AIの提案と採用済み決定を区別する
- 提案が参照したCanonical ID / Source / Evidenceを追跡できる
- purpose、target activity / artifact、AI mechanism / type、autonomy levelを比較・監査できる
- autonomy levelとwrite / invoke / Resource authorityを別の軸として扱う
- staleなbase revisionに対する提案を黙って最新版へ適用しない

model名、prompt、tool trajectory等はEvidenceとして保持できても、それ自体を業務上の意味やauthorityの根拠にはしません。

## RQ-017 人の二重管理を最小化すること

Semantic ModelとExecution Mapping、Canonical meaningとderived artifact等について、人が同じ内容を複数箇所へ手入力し続ける設計を避けます。

機械的に生成・推論できるMappingは生成可能にし、必要な差分や明示的な選択だけを人が管理できることを目指します。

変更履歴は、独自の履歴機構を無条件にDSLへ埋め込むのではなく、Git等の既存version管理も活用します。

## RQ-018 CommandとObservationを分離すること

外部処理へのcommand acknowledgementだけで処理完了とみなしてはいけません。

必要な処理ではObservation / Evidenceによってactual stateを確認し、期待状態との整合を判定できる必要があります。

## RQ-019 実行時の権限とResourceを明示的に制約できること

Pluginや外部Runtimeへ与える権限はCapabilityとPermissionに基づき最小化します。

secret、network endpoint、filesystem path等のHost materialをCanonical definitionやDurable artifactへ不要に保存せず、Resource Broker等の境界から解決できる必要があります。

AI、Editor、generated code、automationも同じ権限・Resource境界を迂回してはいけません。AI actorであることや自律度が高いことから、write、Capability invocation、secret / network / filesystem access等のauthorityを暗黙に付与してはいけません。

## RQ-020 再実行・takeover・unknownを安全に扱えること

外部Runtimeの状態が不明なときに、同じ副作用を自動replayしてはいけません。

ownership、fencing、durable handle、observation等を用いて、誰が現在の実行を調停する権限を持つかと、外部Runtimeがどのgenerationを受理したかを区別できる必要があります。

## RQ-021 ReferenceとConformanceで再現可能性を示すこと

重要な設計境界は、文書だけでなく、小さいReference Workspace、fixture、実行例、Conformance testから再現できることを基本とします。

異なるProviderやTargetを追加するときも、「動いた」という事実だけでなく、共通の意味境界を満たしているかを確認できる必要があります。

AIが作成したProposalやgenerated artifactを採用した場合も、AIを介さずauthoritative sourceを再読込し、同じCanonical meaningとConformance結果を再現できる必要があります。

interactive Targetでは、少なくともSemantic Conformance、利用者ScenarioのInteraction / Functional Evidence、Target固有のVisual Fidelity Evidenceを区別します。GUI agent、VLM、image similarity等のvisual evaluatorを利用しても、その結果だけでSemantic / Functional Conformanceを合格にしてはいけません。

## RQ-022 完成製品と設計検証段階を明確に区別すること

研究実装、architecture probe、production-ready機能を同じ表現で扱ってはいけません。

README、status、release note等から、何が検証済みで、何が未保証かを利用者が判断できる必要があります。

prerelease / experimentalなframework、AI evaluator、agent方式等を検証に利用する場合も、その技術が動いたこととTEIのProduct Requirementが成立したことを区別します。外部研究のsuccess rate、token削減率、visual score等を、TEI自身のReferenceで検証せず固定KPIやConformance基準へ転記してはいけません。

## RQ-023 一人でも維持可能な開発構造を保つこと

初期段階では、一人のmaintainerでも全体を追えることを重視します。

Repositoryを細かく分割しすぎず、関連する改修が同一Repository内で完結しやすい構造を基本とします。

一方、CoreとProvider / Adapterの依存境界はコード上で明確にし、将来の分離を妨げない構造を保ちます。

## RQ-024 性能要求をTargetごとに検証可能にすること

業務端末のように短い応答時間が重要なTargetと、B2C Webのように異なる負荷特性を持つTargetを同じ固定性能前提へ押し込めません。

Target / Driver / Providerごとに必要な性能条件を定義し、測定・Conformanceできる構造を持つ必要があります。

現時点で根拠のない一律の数値目標は固定しません。

## RQ-025 業務上の行為・手順をApplication Functionから独立して設計できること

TEIは、業務をApplication内部の実行手順だけへ還元せず、何のための仕事か、何を契機に行われるか、誰／どの主体が担当するか、どこが手作業・外部作業か、どのApplication Functionをどう利用するかを設計・説明できなければなりません。

Business ActionはApplication FunctionやCapabilityと同一視しません。人間、AI、外部組織、SaaS、物理作業、既存システム等による行為は、TEI Applicationによる実現を持たなくても正当な業務設計情報として保持できる必要があります。

Business Procedure上のactorをAuthorization Roleへ暗黙変換してはいけません。

この要求は、`Procedure`を必ずtop-level Canonical objectにすることや、Workflow Engine、branch、timer、retry、scheduler、orchestration semanticsをCoreへ導入することを要求しません。最小表現はReference Evidenceから決めます。

## RQ-026 Data / Function-Interaction / Business Procedureの異なる視点から相互検証できること

TEIは、Data、Function / Interaction、Business Procedureを同じ業務システムの異なる視点として扱い、一つの視点で見つかった欠落や矛盾を他の視点とauthoritative sourceへ戻して修正できなければなりません。

少なくとも次を検証・診断できる構造を目指します。

- Business Procedureが参照するView / Application Pattern / Action / Capabilityの存在とtraceability
- View / Actionが利用・変更するData / Relation / Validationとの整合
- read-only / ownership / subsystem等の境界を越える操作
- Data / Relation変更によるView / Procedureへの影響
- ApplicationのBusiness Resultと、外部世界で観測される業務完了の混同

Matrix、CRUD表、flow、graph等をこの相互検証に利用できますが、それらを独立した第二の正本として人が二重管理することを要求しません。

三つの視点を同型のCanonical top-level objectへすることも要求しません。hard semantic error、architecture / policy violation、review diagnostic、design question、informational impact等のseverityは、現実的なReferenceでdefect classを再現してから確定します。

## RQ-027 Failure / degraded / unknownを意味・実行境界を越えて診断できること

TEIは、失敗や不確実性を一種類の`error`へ畳まず、利用者・開発者・運用者が何が起きたかを必要な範囲で追跡できなければなりません。

少なくとも次を区別します。

- Business Result
- named Outcome
- Execution Error
- Observation
- Developer Diagnostic
- Operational Telemetry

Target / Driver / Capability / Provider / Runtime等の境界を越える場合も、診断から可能な範囲でApplication / Pattern / Capability / Source / runtime component等の安全なidentityへ戻れる必要があります。

dependency unavailable、degraded、timeout、unknown等を黙ってsuccessへ丸めたり、別Provider / Capability / authorityへsilent fallbackしてはいけません。fallbackを行う場合は、明示contractと観測可能な結果を持つ必要があります。

診断可能性のためにcredential、secret、DSN、不要なphysical identityや業務データを通常ログへ複製してはいけません。

Operational Telemetryは運用上のProjectionであり、それ自体をCanonical business meaning、Business Result、Semantic Evidenceの第二正本にはしません。

## RQ-028 変更の影響・互換性・破壊性を適用前に検査できること

TEIは、変更を単なるfile diffとして扱わず、変更対象のmeaning / Binding / realization / runtime / Targetに応じて、適用前に影響と危険性を検査できる必要があります。

必要な領域では少なくとも次を区別できることを目指します。

- semantic no-op
- compatible change
- incompatible change
- destructive change
- unresolved / insufficient observation

外部状態に依存するdestructive operationでは、planning時の古い観測だけを実行許可へ再利用してはいけません。必要に応じてapply直前のfresh observation、precondition、revision / identity照合等を用い、前提が崩れていればfail closedします。

retry、fallback、CASCADE等によって作用範囲を暗黙に広げてはいけません。より強い作用へ進む場合は明示contractと別の受入判断を必要とします。

rollback / recoveryが完全には成立しない変更では、その制約を隠さず、必要なprecondition、backup / restore、forward recovery、manual intervention等の境界を説明できる必要があります。

impact reportやdeployment planはauthoritative sourceから導出できる情報として扱い、それ自体を業務意味の第二正本にはしません。

## RQ-029 Interactionのoperability / accessibilityをTarget条件付きで検証できること

TEIは、高密度な情報提示や多Target対応を、操作可能性・理解可能性・accessibilityを犠牲にして実現してはいけません。

interactive Targetでは、Target / Host / input modality等の条件に応じて、少なくとも次を検証可能にします。

- keyboard / pointer / touch / voice等、対象Targetで必要な入力経路
- focus / navigation / operation order
- validation / error feedbackとrecovery
- 色や位置だけへ依存しない重要情報の伝達
- state / progress / disabled / unavailable等の識別
- assistive technologyまたはplatform相当機能との両立
- high-density表示における局所性と操作対象の判別

Web Targetでは、適用可能なWCAG 2.2 success criteriaを外部のtestable referenceとして利用できます。ただし、Reference / actual Target Evidenceを取得せず「WCAG準拠」を宣言してはいけません。

Desktop、Mobile、API、MCP、VUI等へWeb固有のUI semanticsをそのまま強制しません。それぞれのTargetで同じ利用目的をどのinteraction contractで満たすかを明示します。

Visual Fidelityやscreenshot scoreだけでoperability / accessibilityを合格にしてはいけません。

## RQ-030 品質主張をContext / Measure / Evidenceへ束縛できること

TEIは、「高速」「安全」「信頼できる」「accessibilityが高い」「保守しやすい」等の品質主張を、対象や条件のない形で固定してはいけません。

重要なquality claimは、必要な粒度で少なくとも次へ戻れる必要があります。

- subject identity
- repository / build / Handover等のrevision identity
- Target / Host / Provider / Runtime等のcontext
- Scenario / workload / precondition
- quality property
- measure / probe / observation method
- acceptance conditionまたは判断理由
- Evidence artifact / reference
- observed / unresolved / not-applicable等のstatus

この組合せを設計上`Quality Envelope`と呼びますが、現時点で固定schemaや新しいCanonical kindを要求しません。Quality Envelopeはderivedな監査・Evidence単位であり、Canonical business meaningの第二正本ではありません。

一律の閾値を先に固定せず、まずReferenceでbaselineを測定し、Target / Host / Provider等の条件と利用目的に基づいて必要なacceptance thresholdを定めます。

ISO/IEC 25010等の外部quality model、WCAG、OpenTelemetry semantic conventions、secure-development guideline等は、抜け漏れ確認やProjection / evaluationの外部referenceとして利用できます。ただし、明示的な採用判断とEvidenceなしにTEIの準拠・認証・Product claimへ格上げしてはいけません。

## 要求を実装へ落とすときの原則

個別IssueやADRを起票するときは、どのProduct Requirementの不確実性を減らす作業なのかを明記します。

特定Providerの深い実装作業が続く場合でも、プロダクト全体として重要な要求が長期間手つかずになっていないかをMilestoneで確認します。