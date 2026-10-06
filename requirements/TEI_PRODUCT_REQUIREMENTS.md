# TEI Product Requirements

## この文書の位置づけ

この文書は、TEI（鼎／テイ）のProduct Value、Vision、Principlesを、プロダクトとして満たすべき要求へ落とします。

ここでは「現在すでに実装されているか」と「プロダクトとして必要か」を分けます。実装状況は`docs/status.md`、個別の設計判断はADR、作業単位はIssueで管理します。

要求番号は、将来文書が分割されても参照しやすいように固定IDとして扱います。

## RQ-001 業務上の意味を実装技術から独立して保持できること

TEIは、業務概念、型、関係、制約、lifecycle等を、特定の画面、Database、言語、配備先と同一視せずに表現できなければなりません。

Canonical IDはfile pathやphysical object nameから独立し、実装や配置を変更しても意味上のidentityを維持できる必要があります。

業務上の識別条件をaccepted meaningとして扱う場合は、単なる`identifier`属性の存在だけで成立済みとしてはいけません。識別を構成する項目、一意性が成立するscope、時間・履歴条件、Relation / Constraintとの関係、根拠となるSourceへ追跡できる必要があります。外部Systemや制度が一意性を保証する場合は、その責任をContract / Sourceとして明示できる必要があります。

文字列や外部コードを正規化・変換した結果の一致を、それだけでbusiness identityの一致へ読み替えてはいけません。原Sourceの表記を保持する必要がある場合は、exact representationとnormalized / operational representationを区別し、同値性を判断するPolicy / MappingのscopeとSourceへ追跡できる必要があります。

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

複数DatabaseやServiceへDataを分散しても、同じ業務対象のidentity、Relation、Constraint、ownershipをlogicalな意味として追跡できなければなりません。APIやmessageで接続できることだけを、Data meaningが統合された状態とは扱いません。

反対に、logicalに一つのData Modelとして扱うことを、単一Databaseへの物理統合要件にはしません。法令、組織責任、可用性、性能、既存資産等による配置差をEnvironment Binding / Provider / Realizationとして表現できる必要があります。

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

## RQ-010 人間向けの高密度な設計・運用認知環境を提供できること

Editor / Review Surfaceは、簡略化されたフォーム入力だけではなく、複雑な業務システムを高い情報密度で理解できるProjectionを提供できなければなりません。

Data、Relation、Rule、View、Procedure等を分断された多数の画面へ追い出すのではなく、必要に応じて3階層以上の関係も局所性を保って確認できる設計を可能にします。

生成AIがApplicationを構築・変更した場合、人間がgenerated code全体を読まなくても、少なくとも次を確認できる必要があります。

- semantic diff
- impact
- unresolved / exception
- decision / approval
- provenance / Evidence
- native custom implementationへ残る箇所

Editor / Review Surface自身をCanonical source of truthとはしません。

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

Application Pattern、View、Target、ORM等の実装都合から、Dataのbusiness identifierやkey構造を逆算してはいけません。特定のRealizationがsurrogate keyや単独keyを必要とする場合も、acceptedなcandidate key、uniqueness、Relation meaningを失わせず、Realization差として追跡できる必要があります。

## RQ-015 生成AIが既存知識から継続的にApplicationを構築・変更できること

生成AI向けの開発支援では、次の順序を基本とします。

1. Existing Domain Component
2. Existing Plugin / DSL Semantic Extension
3. Existing Semantic Pattern / Facet
4. Existing Policy / Variation
5. Composition
6. Declarative expression / rule
7. Capability call
8. Custom implementation

毎回ゼロからRepositoryとコードを読み直してApplicationの意味を再構成したり、毎回ゼロからコードを生成したりすることを標準経路にはしません。

生成AIは、Canonical meaning、Source、Decision、Pattern、Policy、Capability、Binding、Evidenceから、今回の変更に必要なcontextを機械的に取得できる必要があります。

Data / Relationを設計・変更する場合、そのcontextから、業務上の識別条件、既知または候補の関数従属性、candidate key、Relationの意味、cardinality、時間・履歴条件、Source / Evidence、unresolvedへ到達できる必要があります。これらが現在のCanonical schemaでfirst-classに表現されていない場合も、AIが名称や一般的なschema patternから補完した値をaccepted factとして扱ってはいけません。

生成AIは、`identifier`、`external_id`、`code`等の属性名や外部から移行した識別子値だけを根拠に、business identifierやcandidate keyが成立したと判断してはいけません。acceptedな識別条件がある場合は、一意性のscope、時間・履歴条件、Relation / Constraint、Source / Contractを含むcontextへ到達できる必要があります。

生成AIがUUIDやsurrogate key、単独`id`等を実装候補として使うことは妨げません。ただし、それらを追加することで業務上の識別や従属性の分析を省略したり、既にacceptedな複合識別やRelation topologyを失わせたりしてはいけません。

Core内蔵の概念とPlugin / DSLによる意味拡張は、生成AIから見て同じ探索可能な意味資産として扱える必要があります。

既存Applicationへの変更では、少なくとも変更対象、影響範囲、維持すべき不変条件、再生成可能なderived artifact、慎重に扱うnative assetを区別できる構造を目指します。

AIが作った内容を直接Canonical authorityとせず、まずProposal / Structured Intent等の候補として扱い、採用時にはEditor / CLI等と共有できる決定論的なValidation / Acceptance境界を通す必要があります。

構造化された候補から決定論的に生成・変換できる部分は、同じ意味を毎回自由生成する方法より優先します。

過去の成功Proposal、修正trajectory、利用例等を再利用する場合も、Canonical meaningの第二の正本にはせず、元のCanonical ID / Source / Evidenceへ戻れるderived knowledgeとして扱う必要があります。

## RQ-016 AIによる確認提案は控えめで、履歴を持つこと

AI / Editorは、履歴管理、監査、状態遷移、1:N関係、関数従属性、認証・認可等の典型論点について、必要な場合に確認を提案できることを目指します。

識別子や関数従属性を確認するときは、単独`id`、field名、画面上のまとまり、一般的なschema patternだけを根拠にしません。currentなData / Relation / Constraint / Source / Evidenceから確定できる内容、設計候補、未解決点を区別し、根拠とともに提示できる必要があります。

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

Plugin / Semantic Extensionを増やしたこと自体を理由に、Targetの起動時間、初期転送量、常駐memory、storage footprint、network request数、resolver時間を無制限に増やしてはいけません。特に標準機能をExtensionとして提供する場合は、logicalな拡張単位と物理的なartifact / process / Wasm moduleの単位を分け、Targetごとに未使用実装の除外、共通runtimeの共有、artifact重複排除、static linking、lazy loading等を選べる必要があります。

性能最適化のために複数Extensionを一つのartifactへ統合しても、元のidentity、version、Lock、provenance、Authorityへ逆追跡できなければなりません。反対に、一つのExtensionを一つのprocessや一つのWasm moduleとして配布することを標準前提にしてはいけません。

Web Targetでは、process起動、任意filesystem access等のブラウザで成立しない実行方式をSemantic Extensionの成立条件へ含めてはいけません。Target固有のRuntime要件を満たせない場合は、意味を暗黙に変更してfallbackせず、build / Conformance段階で非対応またはUnresolvedとして扱える必要があります。

現時点で根拠のない一律の数値目標は固定しません。Referenceでは、少なくともcold / warm start、初期転送量、install / cache footprint、steady memory、Extension追加時の増分cost、resolver / evaluator overheadをTargetごとに観測できるようにします。

## RQ-025 業務上の行為・手順をApplication Functionから独立して設計できること

TEIは、業務をApplication内部の実行手順だけへ還元せず、何のための仕事か、何を契機に行われるか、誰／どの主体が担当するか、どこが手作業・外部作業か、どのApplication Functionをどう利用するかを設計・説明できなければなりません。

Business ActionはApplication FunctionやCapabilityと同一視しません。人間、AIエージェント、外部組織、SaaS、物理作業、既存システム等による行為は、TEI Applicationによる実現を持たなくても正当な業務設計情報として保持できる必要があります。

AIエージェントがBusiness Actionのactorである場合、必要なData / Constraint / Capability / Authorization / Resource / expected Evidenceを機械可読な形で取得し、許可された範囲でApplication UIを介さず業務を遂行できる余地を持つ必要があります。

ただし、actorとしてAIが指定されていることをAuthorization Roleや実行権限へ暗黙変換してはいけません。

Business Procedure上のactorをAuthorization Roleへ暗黙変換してはいけません。

この要求は、`Procedure`を必ずtop-level Canonical objectにすることや、Workflow Engine、branch、timer、retry、scheduler、orchestration semanticsをCoreへ導入することを要求しません。最小表現はReference Evidenceから決めます。

## RQ-026 Data / Function-Interaction / Business Procedureの異なる視点から相互検証できること

TEIは、Data、Function / Interaction、Business Procedureを同じ業務システムの異なる視点として扱い、一つの視点で見つかった欠落や矛盾を他の視点とauthoritative sourceへ戻して修正できなければなりません。

少なくとも次を検証・診断できる構造を目指します。

- Business Procedureが参照するView / Application Pattern / Action / Capabilityの存在とtraceability
- View / Actionが利用・変更するData / Relation / Validationとの整合
- read-only / ownership / subsystem等の境界を越える操作
- Data ownershipとFunction / InteractionのCRUDを重ねたときのSubsystemの凝集・結合上の違和感
- Data / Relation変更によるView / Procedureへの影響
- ApplicationのBusiness Resultと、外部世界で観測される業務完了の混同

Matrix、CRUD表、flow、graph等をこの相互検証に利用できますが、それらを独立した第二の正本として人が二重管理することを要求しません。

三つの視点を同型のCanonical top-level objectへすることも要求しません。hard semantic error、architecture / policy violation、review diagnostic、design question、informational impact等のseverityは、現実的なReferenceでdefect classを再現してから確定します。

## RQ-027 AIが長期間にわたりApplicationへ再入場できる開発contextを提供できること

生成AIが数日後、数か月後、別model、別agentとして開発へ再入場しても、会話履歴だけに依存せず現在のApplicationを理解できなければなりません。

少なくとも次へ機械的に到達できる必要があります。

- Canonical meaning
- authoritative source
- accepted Decision / Variation
- Capability / Binding
- native asset
- generated / derived artifact
- unresolved concern
- validation / conformance
- Evidence / provenance
- dependency / impact

過去のchat transcriptや特定agentのmemoryをApplication知識の唯一の容器にしてはいけません。

## RQ-028 人間が実装詳細を全面的に読まずに変更を理解・承認できること

生成AIによる継続開発では、人間のreviewをsource code全量の精読だけへ依存させてはいけません。

少なくとも、意味差分、影響、例外、未解決点、権限変更、native custom code、Evidenceを高密度なProjectionで確認し、必要な場合だけsource / runtime detailへ掘り下げられる必要があります。

人間向けProjectionは要約だけでなく、Canonical ID / Source / Evidenceへ逆追跡できなければなりません。

## RQ-029 Domain・Infrastructure知識を適用条件付きで再利用・移植できること

TEIは、Domain model、Pattern、Procedure、Policy、Validation、Capability contract、Database / Infrastructure mapping等を、元Applicationのsource codeへ閉じ込めず再利用可能な知識として扱える必要があります。

再利用時には、少なくとも次を追跡できる必要があります。

- origin
- dependency
- assumption / precondition
- Variation
- Provider / Target constraint
- Evidence / Decision
- destination-specific override

同じ知識を別Applicationへ適用したとき、copy元とcopy先が黙って分岐するだけの再利用を標準にはしません。

## RQ-030 AIエージェントがTEI成果物を業務遂行のOperational Knowledgeとして利用できること

AIエージェントは、TEI成果物からBusiness Procedure、Business Action、Data、Constraint、Policy、Capability、Authorization requirement、Runtime Binding、Evidence requirement等を取得し、許可された範囲で業務を直接遂行できる構造を持つ必要があります。

この要求は、すべてのBusiness Procedureをworkflow engineへ変換することを意味しません。

AIエージェントは、少なくとも次を区別できなければなりません。

- 自分が実行可能なAction
- 人間判断が必要なAction
- 外部組織・物理作業等へhandoffすべきAction
- 実行前に必要なConstraint / Authorization
- 利用可能なCapability / Resource
- Command acknowledgement
- actual Observation / Evidence
- completion / failure / unresolved

TEI成果物を読めること、Business Procedureを理解できること、AIのautonomy levelが高いことは、それだけでwrite / invoke / secret / network / external side effect authorityを与えません。

Agentが行った業務もRQ-018〜RQ-020のCommand / Observation / Resource / retry / unknown境界を迂回してはいけません。

## RQ-031 特定のモデル体系をCoreで特権化しないこと

TEIは、Data Model、Function / Interaction、Business Procedure、UI、workflow等のいずれか一つを、すべてのApplicationが従う唯一の上位モデルとして固定してはいけません。

データモデルを含む各体系は、同じシステムの意味を特定の観点から構造化する表現として扱います。新しい観点が必要になった場合も、既存のモデルへ無理に押し込まず、意味上の関係を保ったまま追加できる必要があります。

新しい業務・設計概念を追加するたびにCore変更を要求してはいけません。Coreへ共通機構を追加するのは、複数の意味体系に共通する欠落がReferenceで確認された場合を基本とします。

標準機能として提供する意味体系にも、原則として第三者Extensionと同じidentity、dependency、conflict、discovery、Conformanceの契約を適用します。実装をbuiltinにすることは許容しますが、標準機能だけが利用できる別の意味追加経路を前提にしません。

標準namespaceのauthorityも同じ原則に従って明示します。`std.*`等の予約namespaceは、最初に宣言したExtensionやload順によって所有権を得る仕組みにしてはいけません。標準機能が複数packageへ分かれる場合も、namespace ownerからの明示的な委譲や同等の契約で追跡できる必要があります。

## RQ-032 Plugin / DSLで意味論を宣言・合成できること

Plugin / DSLによる拡張は、実装関数を追加するだけでなく、概念が何を意味し、他の概念とどう関係し、何を満たす必要があるかを機械可読に宣言できる必要があります。

少なくとも、拡張のidentity、namespace、semantic version、適用範囲、依存条件、既存概念との関係、Constraint / Validation、ProjectionやCapabilityへの接続を追跡できなければなりません。意味論と、その意味を読み書き・検証・実現する規則は、同じ拡張のidentityへ辿れる必要があります。

Semantic Extensionの依存は、特定packageのidentityを必要とするものと、Semantic Interface等の契約を満たせばよいものを区別して表現できる必要があります。Semantic Interface自体も、identity、contract version、Source、contract fingerprint等へ辿れる意味契約として扱います。providerが同じInterface名を自己申告しただけで適合済みとしてはいけません。

意味解釈に影響するSemantic依存を、Environment固有のRuntime PluginやToolingの依存と同じ解決scopeへ暗黙に混ぜてはいけません。Validation結果がShared AcceptanceやConformanceへ影響するEvaluatorも、EnvironmentごとのRuntime provider選択だけでAccepted semantic validityを変えない構造にする必要があります。

依存先と互換範囲を明示し、root requirementから推移依存を決定論的に解決できなければなりません。同じ入力から得られるresolved graphは、load順やcatalogの列挙順に依存してはいけません。さらに、宣言したconstraint languageの範囲で有効解が存在する場合、greedyなversion選択や探索順だけを理由に解決不能としてはいけません。

複数の有効解が存在し、Acceptedなresolution policyでも一意に決められない場合は、solver内部の探索順で一つを暗黙選択せず、候補またはUnresolvedとして扱える必要があります。Semantic Interfaceを満たすproviderが複数ある場合も同様です。

Accepted Semantic Setは、root requirements、exact version、declaration fingerprint、Semantic Interface contract fingerprint、解決済みdependency / interface wiring、resolution policy、fingerprint canonicalization規則を再現できるlock、または同等の表現へ固定できる必要があります。Catalogへ新versionが追加されたことだけを理由にAccepted Semantic Setを再解決してはいけません。root requirementsや解決規則が変わった場合はstaleとして検出し、意味変更候補として扱います。

同じnamespaceやsemantic identityの意味を、互換性のない定義へ黙って差し替えてはいけません。複数の拡張が同じ意味へ異なる定義を与える場合も、読み込み順や偶然の実装順序で上書きせず、競合、優先関係、互換性を明示的に判定します。一つのCanonical Worldで互換性のない世代の意味体系を同時に有効化する場合は、namespaceやtranslationの境界を明示します。一意に決められない場合はUnresolvedとして扱います。

semantic versionとは別のcompatibility family / epoch / levelを必須軸として先に固定しません。version range等では表現できない互換性の差が複数Referenceで確認された場合に追加を検討します。

Extension packageの依存関係に生じるcycleと、業務上のRelation graphに存在するcycleは区別します。package dependency cycleは原則としてAccepted graphへ入れず、共通Interfaceの抽出やBridge Extension等によって解消できる構造を目指します。

あるExtensionが存在するだけで、別Extensionの既存の意味が暗黙に変わる方式を標準にはしません。組合せによって新しい意味が必要な場合は、Bridge Extensionや明示的なroot requirement等としてSemantic Setに現れる必要があります。

optionalなdiagnosticやProjectionであっても、その存在がShared AcceptanceのAccept / RejectやCanonical validityへ影響する場合は、単なるTooling依存として扱わずSemantic Setへ明示する必要があります。

Semantic Extensionの追加やversion変更、Profileによるroot requirement集合の変更によってAccepted Semantic Setが変わる場合は、sourceが変わっていなくても意味変更候補として扱います。新しいcompatible versionが見つかったことだけを理由に、Accepted Semantic Setを自動更新してはいけません。

Semantic Extensionのlogical identityと、実行・配布に使うartifact境界は同一である必要はありません。同じAccepted Semantic SetからTargetごとに異なるpackaging planを導出し、複数Extensionを一つのbuiltin / linked artifactへまとめたり、必要な実装だけをWasm等で遅延読込したりできる必要があります。この変換でSemantic Set fingerprintを物理配置へ依存させてはいけません。

実行codeを持たないdeclaration-only ExtensionまでRuntime artifactを要求してはいけません。実行codeが必要な場合も、Targetに応じてbuiltin / linked、portable sandboxed module、process等の実現方式を選べる余地を保ちます。Targetごとのpackaging差は、Semantic contractとConformanceを満たす限り、意味の差として扱いません。

意味を宣言できることは実行Authorityを意味しません。Plugin / DSL拡張もRQ-019の権限・Resource境界を迂回してはいけません。

## RQ-033 AIとToolが意味拡張を実装のレールとして利用できること

生成AI、Editor、Driver等は、Core内蔵の概念とPlugin / DSL拡張を同じ探索対象として扱い、利用可能な概念、適用条件、依存、Constraint、Validation、Projection、Capabilityへの接続を機械的に取得できる必要があります。

依存が推移的に大きくなっても、AI / Toolはresolved setだけでなく、各Extensionがどのroot requirementから導入されたか、どのRequirementをどのprovider / Semantic Interfaceが満たしたか、どの要素がそのExtensionへ逆依存しているか、解決できない場合は何が競合しているかを機械的に辿れる必要があります。

生成AIによる実装では、再利用可能な意味拡張の探索とComposition、決定論的に導出できる変換・Validationを、自由生成やApplication固有のcustom implementationより先に利用できる構造を目指します。Catalog全体を毎回読み込ませるのではなく、対象subjectとSemantic dependency graphから必要なclosureを切り出し、必要に応じて理由や影響を追加で探索できる構造を優先します。

複数の実装候補が同じ制約を満たして一意に決められない場合や、判断に必要な情報が不足している場合は、AIが暗黙に一つを選ばず、ProposalまたはUnresolvedとして扱える必要があります。依存競合も単なる「解決失敗」にまとめず、原因となるrequirementとその由来を説明できることを目指します。

resolution budget超過、Catalog不足、Interface authority不足等も、実際のdependency conflictと区別して説明できる必要があります。AIやToolが「解けなかった」ことを「解が存在しない」ことへ読み替えてはいけません。

この要求はAIが誤らないことを前提にしません。実装上の自由度を必要な箇所へ限定し、機械的に検証できる境界を増やすことで、同じ設計論点の再判断とApplication固有コードを減らすことを求めます。

## 要求を実装へ落とすときの原則

個別IssueやADRを起票するときは、どのProduct Requirementの不確実性を減らす作業なのかを明記します。

特定Providerの深い実装作業が続く場合でも、プロダクト全体として重要な要求が長期間手つかずになっていないかをMilestoneで確認します。
