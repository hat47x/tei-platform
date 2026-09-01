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

## RQ-016 AIによる確認提案は控えめで、履歴を持つこと

AI / Editorは、履歴管理、監査、状態遷移、1:N関係、関数従属性、認証・認可等の典型論点について、必要な場合に確認を提案できることを目指します。

ただし、次を満たす必要があります。

- 一度回答した質問を理由なく繰り返さない
- 過去の選択と理由を後から追跡できる
- 選択を後から変更できる
- 不要な提案を無視できる
- 提案カテゴリを抑制・無効化できる
- AIの提案と採用済み決定を区別する

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

## RQ-020 再実行・takeover・unknownを安全に扱えること

外部Runtimeの状態が不明なときに、同じ副作用を自動replayしてはいけません。

ownership、fencing、durable handle、observation等を用いて、誰が現在の実行を調停する権限を持つかと、外部Runtimeがどのgenerationを受理したかを区別できる必要があります。

## RQ-021 ReferenceとConformanceで再現可能性を示すこと

重要な設計境界は、文書だけでなく、小さいReference Workspace、fixture、実行例、Conformance testから再現できることを基本とします。

異なるProviderやTargetを追加するときも、「動いた」という事実だけでなく、共通の意味境界を満たしているかを確認できる必要があります。

## RQ-022 完成製品と設計検証段階を明確に区別すること

研究実装、architecture probe、production-ready機能を同じ表現で扱ってはいけません。

README、status、release note等から、何が検証済みで、何が未保証かを利用者が判断できる必要があります。

## RQ-023 一人でも維持可能な開発構造を保つこと

初期段階では、一人のmaintainerでも全体を追えることを重視します。

Repositoryを細かく分割しすぎず、関連する改修が同一Repository内で完結しやすい構造を基本とします。

一方、CoreとProvider / Adapterの依存境界はコード上で明確にし、将来の分離を妨げない構造を保ちます。

## RQ-024 性能要求をTargetごとに検証可能にすること

業務端末のように短い応答時間が重要なTargetと、B2C Webのように異なる負荷特性を持つTargetを同じ固定性能前提へ押し込めません。

Target / Driver / Providerごとに必要な性能条件を定義し、測定・Conformanceできる構造を持つ必要があります。

現時点で根拠のない一律の数値目標は固定しません。

## 要求を実装へ落とすときの原則

個別IssueやADRを起票するときは、どのProduct Requirementの不確実性を減らす作業なのかを明記します。

特定Providerの深い実装作業が続く場合でも、プロダクト全体として重要な要求が長期間手つかずになっていないかをMilestoneで確認します。
