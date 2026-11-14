# ETL/ELT・MFT/ファイルリレー横断契約の圧力試験（設計候補・2026-11-04）

- **状態**: 外部一次資料・既存TEI Issueに基づく **設計・Research / NOT RUN / 未採用**。新Core型、正式Plugin API、Host、Scheduler、業務受理、Target実行保証を採用した文書ではない。
- **調査対象**: ODIP（モデル駆動処理・業務基準日・リカバリ）、Apache Hop（Pipeline/Workflow）、SeaTunnel（Logical/Physical Plan）、HULFT-HUB（蓄積・追越禁止）、AS2（受領証拠）、Apache Camel（ファイル公開/重複取込み）、OpenLineage（Job/Run/Dataset）。
- **照合したmain**: `dd773c1ad8735a8c6aeb67ef51f8437ac3320e17`（GitHub branch APIで2026-11-04に取得）。
- **文書の責務**: 「既存Issue/契約に不足する意味の区別」と「反証可能なHeadless受入圧力」を記録する。要件正本は `requirements/TEI_PRODUCT_REQUIREMENTS.md`、採用判断はADR、実装資格は各Owner。新しい第二のPipeline/Scenario/Authority/Resolver/Lock正本は作らない。

## 1. 問題と全体判断

多くのローコード画面系はデータ加工・大量処理・企業間ファイル転送を別ツールで補完する。TEIでは、標準データ構造と型付き関係を共通基盤として **同じ受理済み意味から、UI/APIと独立した非画面Batch・ETL/ELT・Transfer Adapterなど異なる実装成果物を構成する** ことを検討する。

ただし「ひとつのパイプライン図」はひとつの意味論ではない。少なくとも次の責務を区別する。

| 観点 | 対象・契約 | 成功の意味と限界 |
| --- | --- | --- |
| Data Transform | Dataset, schema, grain, completeness, relation, operator, input/output | 値の導出。NULL/欠損/重複/順序/精度/粒度・時点を保持 |
| Workflow / Trigger | Step dependency, schedule, calendar, business period, retry | 起動・分岐・依存。Dataflowの矢印を実行順序と同一視しない |
| Materialization / Publication | Artifact, snapshot, storage, visibility, commit protocol | 読取可能な成果物への公開。アップロード終了≠公開・可視性保証 |
| Transfer / Relay | Sender, receiver, route/hop, attempt, spool, receipt | Byte/Artifactの配送。転送完了≠後続取込み・業務受理 |
| Ingestion / Reconciliation | FileSpec, import unit, event identity, watermark, reject | データ・事象の照合。重複配送≠重複業務効果 |
| Business Effect / Acceptance | Operation intent, authority, provider commit, independent read, owner acceptance | 実際の更新・受理。Job成功/MDN/ACKからは推論しない |
| Build / Deployment / Observation | target-neutral plan, target-bound binding, artifact set, run/attempt evidence | 実行対象の生成と証拠。生成成功≠実配備≠現在権限 |

**非目標:** ODIP/HULFTを複製すること、すべてを必ずJob化すること、TEIが常駐サーバやJob Scheduler/Resource Authorityになること、Plugin実装やAgentの自己申告のみで業務効果を確定すること。

## 2. 外部製品から抽出する相補的な設計知見

- **ODIP**：入力・加工・出力仕様から中間データとジョブネットを導き、処理基準日・業務カレンダー・期・前回結果を利用するサイクリック処理・リカバリを扱う。**業務基準期間 ≠ 物理実行時刻**。ベンダーが述べる自動化や品質の程度は独立検証済みでない。実行はトランスフォーマ/リポジトリ方式であり、同じ仕様から任意の独立バイナリが生成できる事実ではない。
- **Apache Hop**：Dataflow Pipeline（Transformが並行に動く）とWorkflow（Action実行・完了・分岐）が異なる意味を持つ。GUIの図の見た目と実行トポロジーは同一ではない。
- **SeaTunnel**：Source/Sink/Transform API、エンジン別Translation、LogicalDagとPhysicalPlanの分離。TEIが対応するなら論理契約→Provider能力/Targetに依存する物理計画を可視化し、unsupportedを隠さない。
- **HULFT-HUB**：蓄積（Store-and-Forward）で送信側/受信側の稼働差を吸収。追越禁止は基本的に同一ファイルID内の順序制約で、**多段中継の全ホップ設定が条件**。順序確保は並列性に影響し、蓄積異常・蓄積開始前失敗等の例外を考慮する。これをグローバルな業務イベント順序に拡張しない。
- **AS2 RFC 4130**：MDNと署名付きReceipt/MIC照合は受領証拠に使えるが、受信業務データの意味・Ownerの受理とは異なる。署名なしと署名済み・同期と非同期・異常を区別する。
- **Camel File**：書込み中ファイルの誤取込み対策としてreadLock/done marker/公開先へのmove、冪等ConsumerのKey/Storeが重要。完了マーカーやrenameの強度はファイルシステム/ストレージ/Host依存。
- **OpenLineage**：Job/Run/Datasetとfacetを分ける。TEI Source、Accepted Meaning、Plan、Build Artifact、Transfer Attempt、Provider Effect、Independent Readの来歴を同一の成功フラグに畳まない。

## 3. 標準データ構造への最小対応付け（名前と配置は未採用）

Coreは `Type / Instance / Port / Binding / Graph / Conformance` などの普遍的構造に留め、ドメイン語彙は `tei.std.*` またはドメインPluginにより**型付き役割/facet/関係**として組み立てる。Principal/Account/Business Entity、特定DB・特定Host・固定の実行エンジンを最小バッチに必須化しない。

| 意味上の役割（仮称） | 既存責務に接続する点 | 必須の区別 |
| --- | --- | --- |
| `DatasetRole` / `TransformRole` | #2622 Data Operations / Derived Dataset | Row/Bag/Sequence, grain, completeness, input scope, data time |
| `FlowStepRole` / `DependencyRole` | #1130 Flow、#2744 Buri/TEI/EKI案 | data dependency / temporal order / control / causal edge |
| `TransferContractRole` / `HopRole` | #2542 ファイル・メッセージPlugin面 | endpoint/route/能力・再送・順序の範囲・期限・容量 |
| `ArtifactRole` / `PublicationRole` | #943/#1102 ArtifactSet、#1807 import | logical dataset vs materialized byte object vs published version |
| `ReceiptRole` / `AttemptRole` | #1174 Effect/Receipt、#2542 import unit | transferred/verified/published/ingested/acceptedの独立観測 |
| `PeriodRole` / `RecoveryRole` | #1130、#1129、#2622 | business effective period vs physical run/observedAt、checkpoint vs effect |
| `LineageRole` | #1121 Lowering、#2619 Everything is Data | plan/build/run/attempt/independent observation各版の参照 |

この表は正式な標準型の提案でもない。既存Type/Relation/Plugin/ModelRefで表現できるなら新しいCore種別は不要。固定 `Pipeline` 型・新DSL・第二のArtifact Storeを先行させない。

## 4. 独立して保持すべき契約と反例

### 4.1 同一性の軸

- Semantic Dataset ID と Source snapshot/revision
- DataArtifact version と物理ファイルのbyte digest（algorithm/encoding/length）
- ファイル名/パスと配置先のversion/generation（同名上書き・ABA）
- Transfer contract と配送要求、個々のAttempt（retry/relay hopを区別）
- Import Unit とBusiness Event identity、Provider Effect id/receipt

内容ハッシュで同じbyteでも、別の取引・別宛先・別の期・別送達要求が同一とは限らない。逆に輸送時に圧縮・暗号化・改行変換・文字コード正規化されれば、論理内容が等価でもbyte digestは異なる。何に対するDigestか、変換前後のBoundaryを明示する。ハッシュの桁数・alphabet・algorithm選択は既存ID研究へ接続し、具体的な不可逆変換や衝突安全性を推定しない。

### 4.2 保証を直積として表現し、強い保証へ自動昇格しない

| 独立facet | 例 | 禁止する推論 |
| --- | --- | --- |
| Completeness | complete / partial / unknown | 0件受信⇒0件存在 |
| Order | per-file-id/per-partition/per-destination/unknown | 1ホップの順序⇒多段/全業務順序 |
| Delivery | at-most-once / at-least-once / uncertain | 通信ACK⇒永続受領 |
| Integrity | transferred bytes verified / signed receipt / unknown | byte一致⇒正当な業務内容 |
| Publication | temp/spooled / committed-visible / unknown | Put成功⇒Consumerが完成版を見る |
| Replay | restartable from offset / application replay / unsupported | byte resume⇒Effectを安全に再実行 |
| Business effect | committed / rejected / unknown | Job成功⇒Provider Commit Known |
| Authorization | contract declared / runtime grant observed / expired | PluginのCapability宣言⇒現在実行可能 |

`exactly-once` は多義的であるため、何の境界（配信試行、Import Unit、業務Effect、受領済み事象）に対する一回性かを必要条件として固定する。一般の分散ファイル配送＋外部DB更新で普遍的な原子Commitを主張しない。unknown effectでは照合/Owner回復へ停止。

### 4.3 時点と物理資源

Event time、business/effective period、batch reference date、processing time、transfer/spool time、publish time、observedAt、receipt time は別値。基準日変更は同じデータ変換ジョブの意味を変え得る。Spool budget、storage lifetime、file size、bandwidth/egress、backpressure、CPU/Memory、deadlineとrecovery slackは別単位で示す。実Nodeの資源Admission/Lease/RecoveryはEKI Owner側、TEIは制約・必要な能力と確認結果を受ける（#1130/#2744）。単一ローカルGo executable、OS Scheduler、外部MFT、EKIを選択可能な実行/連携先とし、どれも必須Hostにはしない。

### 4.4 Publish/Receive境界

Writerが最終ファイル名へ直接書くことを既定安全とみなさず、暫定配置・検証・公開条件を明示する。renameの原子性、オブジェクトストレージの世代条件、completion marker、SFTP remote path、後続Consumerのpoll方式の差をProvider capabilityとして検証する。転送/配信側の「完了」、受信側の「検証」、Importerの「受理」、業務Ownerの「確定」は別のReceipt/Observationとして関連付ける。取込み元削除・永久保留レコードの更正には別Authority/Retention/監査契約が必要。

## 5. Headless-firstの最小縦切り（新たなScenario SSOTを作らない）

**入力例**: 売上CSV（処理基準日D）＋店舗マスタから集計Datasetを生成し、CSVとSQLiteのELT/独立Worker-ETLへ実現、SFTP-compatible/in-processの合成Relayを通して取込み・照合する。既存#2145の `FileSpec` / `ImportFile` と #1130のBatch、#1180のScenarioを再利用し、未存在の実MFT Adapterはあるものと扱わない。

以下は**新規の受入Pressure提案・全件 NOT RUN**。独立Expected/実行Script/TargetとOwnerは、既存#1131/#1180/#2514に集約し、本文は単なる比較票とする。

| ID | 入力・反例 | 期待する最小の判定 |
| --- | --- | --- |
| DM-01 | 同じ受理済み意味をSQLite ELTとWorker ETLへ | null/bag/順序/decimal/grainを比較し、異なる場合は不適合。両実装のBuild digestとSourceを別照合 |
| DM-02 | 処理実行日D+1、業務基準日D | 実行日が変わっても選択した業務期・依存するデータ版を誤変更しない |
| DM-03 | 再実行、前日結果からのサイクリック再計算 | 入力版とチェックポイントを識別。前回出力を現在正本として黙って再利用しない |
| DM-04 | 最終名へ書込み中に受信側がpoll | 不完全ファイルをPublished/Accepted扱いしない。Providerが公開原子性を証明できなければunsupported |
| DM-05 | Relay 2ホップ、片方のみ順序制御 | 全路線の順序保証を拒否/unknown。per-file-id順序をglobal orderへ昇格しない |
| DM-06 | spool full / expiry / crash-restart | hold/failed/unknownと実送出証拠を分離。勝手にBusiness Lost/Acceptedとしない |
| DM-07 | timeout後に二重配送、Import再送 | Transfer Attemptは2件、Import Unit/Business Eventはidempotency規則に従って1回のみ確定 |
| DM-08 | 配送成功後、Importerの検証失敗 | Delivered/VerifiedとIngested/BusinessAcceptedの別ステータスを維持 |
| DM-09 | 同一名・異byte、同一byte・別期間 | path/hash/period/transport idを混同せず、再取り込み・上書きの可否を型付き診断 |
| DM-10 | 送信/中継で圧縮・暗号化・文字コード変換 | Digest検証の対象byte・normalization/semantic equivalence・lossを分離 |
| DM-11 | 外部DB commit応答喪失、Retry要求 | unknown effectは自動再実行せず、Receipt/独立Readへ引き渡す |
| DM-12 | 同一FlowをCLI/OS timer/外部MFT/EKIに接続 | 意味・Scenarioは同じ。失われたCapability（順序、保存、Scope、Grant）はunsupported |
| DM-13 | 図のfilter/collapse・不透明な外部MFT・動的Provider | TEI管理範囲外を「図に無いので存在しない」と断定しない（#1124/Topology文書） |
| DM-14 | 転送先/外部Provider権限失効・秘密Field | Capability/Planは実Grantに昇格しない。確認不足時の送出・開示を保留 |

**初回の実装可能な縦切り**: DM-04, DM-07, DM-08, DM-09 を既存限定Referenceへ追加できるか静的確認 → 実File/SQLite独立Read → DM-01/02 の二生成戦略 → 多段転送DM-05/06。その後に実SFTP/HULFT相当への接続候補を選定する。性能、スケール、実取引先、実Windows/HULFT、E2Eを達成済みとしない。

## 6. 既存Ownerとの境界と反映順

| Owner | 今回の追補との関係 |
| --- | --- |
| [#2619](https://github.com/hat47x/tei-platform/issues/2619), [#2622](https://github.com/hat47x/tei-platform/issues/2622) | 標準データ構造とData Operations/Derived Dataset、用途相対の意味同等性 |
| [#1130](https://github.com/hat47x/tei-platform/issues/1130), [Draft #2788](https://github.com/hat47x/tei-platform/pull/2788) | UI非依存Batch、Invocation/Build Unit/ArtifactSet、非常駐実行。ここで新Scheduler/Job Engineを作らない |
| [#2542](https://github.com/hat47x/tei-platform/issues/2542), [closed #2145](https://github.com/hat47x/tei-platform/issues/2145), [#1807](https://github.com/hat47x/tei-platform/issues/1807) | FileSpec、Import Unit、同一事象照合の既存成果。MFT/relayのプラグイン面は**Owner未決** |
| [#1121](https://github.com/hat47x/tei-platform/issues/1121), [#943](https://github.com/hat47x/tei-platform/issues/943), [#1102](https://github.com/hat47x/tei-platform/issues/1102) | Target-neutral plan→実Target/physical Artifact/Build provenance。第二Build Lock不要 |
| [#2744](https://github.com/hat47x/tei-platform/pull/2744), [#1174](https://github.com/hat47x/tei-platform/issues/1174), [#1129](https://github.com/hat47x/tei-platform/issues/1129) | TEI Flow meaning／EKI Task resource／実Provider EffectのOwnerを分ける。新Authority不要 |
| [#2495](https://github.com/hat47x/tei-platform/issues/2495), [#2501](https://github.com/hat47x/tei-platform/issues/2501), [Topology設計](TOPOLOGY_COMPOSITE_VIEWS_AND_PIPELINE_2026-10.md) | Bag/Sequence/Stream/GraphとOrder、opaqueと観測範囲のPressure。新Core型の確定に転用しない |
| [#1180](https://github.com/hat47x/tei-platform/issues/1180), [#1131](https://github.com/hat47x/tei-platform/issues/1131), [#2514](https://github.com/hat47x/tei-platform/issues/2514) | 受入・試験Expectedの正本と独立Oracle。別のテストの正本を作らない |

段階: (P0) 現行Referenceと契約の静的Fit/Gap → (P1) Headless合成 negative fixtures → (P2) 実SQLite/ファイル系別processでSource/実Artifact/Independent Read → (P3) 別Transport/ProviderでCapabilitiesと未知を比較 → (P4) 実MFT/EDI連携・復旧/運用を対象Ownerの権限で実証。各段階で試験済み範囲・未実行・意思決定を独立記録する。

## 7. 一次資料（参照先と検証範囲）

- ODIP 機能: https://www.imkk.jp/product/function
- ODIP コンポーネント/実行まで: https://www.imkk.jp/product/component
- ODIP Tutorial・メタデータ: https://www.imkk.jp/product/tutorial
- Apache Hop pipeline/workflow: https://hop.apache.org/manual/latest/pipeline/pipelines.html / https://hop.apache.org/manual/latest/workflow/workflows.html
- SeaTunnel Architecture: https://seatunnel.apache.org/docs/architecture/overview/
- SeaTunnel Engine Logical/Physical: https://seatunnel.apache.org/docs/architecture/engine/engine-architecture/
- HULFT-HUB蓄積: https://www.hulft.com/help/ja-jp/HUB-V3/COM-HUB/Content/HUB_HUB/Fun_HUB/Acc.htm
- HULFT-HUB追越禁止: https://www.hulft.com/help/ja-jp/HUB-V3/COM-HUB/Content/HUB_HUB/Fun_HUB/Phbt_OVTK.htm
- AS2 RFC 4130: https://www.rfc-editor.org/rfc/rfc4130.html
- Apache Camel File: https://camel.apache.org/components/next/file-component.html
- OpenLineage facets: https://openlineage.io/docs/spec/facets/

**証拠制約:** 公開マニュアルの製品動作はその文書の版・条件に限定。ODIP/HULFT/AS2/Camel/SeaTunnel等の実機実行、コード監査、性能・セキュリティ評価、TEI生成Targetとの接続、実企業取引先、正式Owner受理、全Repository Healthは本書では **NOT RUN**。現在のmain/関連PRが変更されたら再照合する。


## 8. 2026-11-04 再照合：公開・配送・受理の「保証の合成」と停止線（設計追加／NOT RUN）

本節は既存[#2604](https://github.com/hat47x/tei-platform/issues/2604)の複数ファイル公開、[#1130](https://github.com/hat47x/tei-platform/issues/1130)のJob入出力、[#2542](https://github.com/hat47x/tei-platform/issues/2542)のファイル取込み/連携を**別Ownerのまま**つなぐ。新しい成果物Publisher/Runner、第二の受理State Machine、全Provider横断Txは作らない。HULFT-HUBの配信順・S3通知を、業務イベント全順序や取引の完全性に昇格させない。

### 8.1 Five boundaries: 同じ「成功」を使わない

| 証拠境界 | 証明可能な主張（要独立の証拠） | そこからは**導けない**こと／Owner |
| --- | --- | --- |
| **B0 Semantic / Plan** | Accepted Source/意味/Function/Period、型・入力・出力・必要な能力、Target-neutral plan | 未実行の生成計画から実Byteや実権限。#2619/#2622/#1121 |
| **B1 Materialization / Publication** | 物理MemberのByte、Digest algorithm、版、Manifestの一致、Consumerの実Readerが同一世代のMemberを取得可能 | Manifest Complete≠一式atomic公開≠Business Accepted。#2604/#943/#1102 |
| **B2 Transfer / Relay** | Route/Hop、Contractと各Attempt、Spoolの保存/期限、Byte整合性、配送/受領Receiptの範囲 | 配送Ack≠永続公開≠Importer確認。HULFT相当実Adapterは未実装。#2542 |
| **B3 Ingestion / Reconciliation** | Import unit/file spec/検証結果、変換失敗/保留、Event identity、重複適用の結果、Consumer側確認 | 同一File digest≠同一Business Event；Record ingest≠業務意味受理。#2145/#1807/#2542 |
| **B4 Provider Effect / Owner** | 現在Authority、実Request/Commit Receipt、別Read、許可されるBusiness Result、Ownerの意味受理 | Worker exit=0、EKI Task completed、MDN、S3通知だけでCommitを推論しない。#1174/#1129/#1119 |

**契約の合成規則（候補）**:
1. 必要な保証をConsumer/Operation/Period/宛先/データ分類で宣言する。各Providerは対象操作・版・実環境・Scopeに限定した能力を返す。意図した保証より弱い場合は `unsupported`、未照合なら `unverified` とし、勝手に「より弱い成功」へ代替しない。
2. `Transferred`、`IntegrityVerified`、`Published`、`Ingested`、`BusinessAccepted`を一列の単一enumの状態機械に強制しない。各境界に独立したObservation・unknown/partial・時点・Authorityを持たせ、順不同の到着を許す。全段階を必須化せず、不要な段階は `not-applicable` と `unverified` を区別。
3. 互いの証拠は`subject qualified identity + version + attempt + scope + provider + time`で参照関係を張る。路線・内容・業務のIDを単一キーへ自動統合せず、正確に解決できない場合はambiguous/unknown。既存Hash ID方式（#2814/ Draft #2815）はScheme/用途別の候補であり、Artifact Digestや正本IDを置き換えない。
4. `unknown`（ACK失踪・途中失敗・独立Reader不可）時は同じ副作用Jobを自動再送しない。必要に応じHostの現在GrantとEffect/Receipt再照合を要求。純粋な計算の再試行能力を業務Commitへ転用しない。

### 8.2 Publicationに必要なCapabilityは配送能力と別

| Provider/方式（例） | 明示できる可能性のある保証 | 未検証のまま推定しない保証 |
| --- | --- | --- |
| ローカルFile stage→rename | 同一FS/OSでの個別ファイル公開候補。Consumerが期待版のbyteを再読取 | Windows/POSIX/ネットワークFSすべての一式原子公開、永続化までの保証 |
| S3の単一Object PUT | **単一Key**でのatomicな旧版/新版観測、read-after-write、条件付きPutによる上書き競合防止 | 複数Keyの一式公開、通知順序、業務取込み、ETag=内容Digest |
| Immutable generation prefix + manifest pointer | 成果物を別世代のKeyへ保存し、公開Pointerと各Memberの版を照合する実装候補 | Pointerだけで未到達Member/実権限/Readerが読んだByteの保証 |
| HULFT-HUB Store-and-Forward | 蓄積、任意送出、経由Serverの条件付き順序制御 | 受信側Ingestion/業務受理、全イベント欠落ゼロ、全路線に無条件で順序保証 |

**一次資料で確認した限定的な違い**:
- S3は単一Keyの更新に原子性・強いread-after-write整合性を提供し、`If-None-Match` / `If-Match`で単一Key更新を制約できる。**ETagは条件付き更新のTokenであって一般のContent Digestではない**。特にMultipartや一部の暗号化方式ではETagは全Object MD5にならない。独立内容Digestは別のアルゴリズム・検証対象Byteを指定する。
- S3 Event Notificationsは**at-least-once**であり、重複と配達順序逆転があり得る。`sequencer`は同一Object Keyでのイベント比較に使えるが、異なるKeyを跨ぐ全順序や取引の完全性には使えない。Event通知成功をPublication/Reader/Importer受理へ転用しない。
- HULFT-HUB追越禁止は**同一File IDの蓄積開始順**に関する条件付き送出制御。多段の場合は全HUBでの設定が必要。ただし先行データが蓄積異常で削除された、または蓄積開始前にエラーになった場合、後続は待機せず送出され得る。従って`ordered`と`gap-free`は直交する。追越禁止で並列度が下がる条件も、Jobのdeadline/bandwidth/cost評価へ返す。
- AS2 signed MDN/MICは元メッセージと受領証跡の関連付けを支えるが、後続ERP取込み・事業者の業務承認を証明しない。無署名MDN、非同期MDN、保持された元メッセージとの照合有無を区別。
- Camel File Consumerの既定冪等Keyが物理Pathの場合、同名別版ファイルを誤って二度目と判定し得る。TEIは物理ファイル名をImporterの唯一のBusiness Eventキーにしない。

この違いから、**Provider capabilityは単なる `transferSupported=true` でなく、`guarantee + subject + ordering key + scope + version + required evidence` の候補表**で表す。現行TEIにこれらの正式APIがあるとの主張ではない。Capability不一致時の実行前の拒否やTarget fallbackは既存#1121/#951のOwnerへ委譲する。

## 9. 実行経路における「一式性・順序・完全性」の直交チェック

同じ報告用CSV一件だけでは、**物理一式公開**と**ビジネス入力の完全性**を区別できない。

例: 11月3日締めの売上結果 `D` がCSV `A`・補助マスタ `B`・Manifest `M` からなる。新世代 `R2` の `A2` を公開した時点で旧世代 `R1` の `B1` と `M1` をReaderが見た場合、一式公開は成立しない。全Byteがそろっても、受注の連番 `1,2,4`（欠落3）なら業務期の入力完全性は成立しない。さらに配送先が`R2`のCSVを受信しても、受理対象日の意味や改訂済みMasterとの関連付けは別。

**一式性の確認**は[#2604](https://github.com/hat47x/tei-platform/issues/2604) AP-01〜07（と[PR #2817](https://github.com/hat47x/tei-platform/pull/2817)の限定Characterization）を再利用し、新しいPublisher/同名Fixtureを作らない。**Business Datasetの期間/粒度/期待する全件性**は#2622、**Transfer/Relayを跨ぐ列挙・欠落**だけを本ノートの固有Pressureとする。

- **順序（order）**：どのPartition/File ID/宛先/路線で、どの操作の先行制約を保証するか。全ホップが必要保証を満たすかを確認。
- **完全性（coverage / gap-free）**：そのScopeで期待される全件数・連番範囲・締め時刻・Manifest member集合・Import件数やChecksum等の独立の根拠があるか。先行Spool失効・欠落を「順序通り」によって隠さない。
- **一式公開（atomic visibility）**：同じ世代のMember集合をConsumerが再読取し、任意の一つのMemberのみ新世代を参照する状態を拒否するか。S3単一Keyの原子性と複数Key一式性は異なる。
- **鮮度（freshness）**：Source/Meaning、Job Period、Provider snapshot、物理Object version、Receipt/Grant時点が現在の要求条件を満たすか。0→1→0のABAや同名上書きを同値としない。
- **効果（effect）**：取込み後の検証成功、外部DB Commit、OwnerのBusiness Acceptanceの別Receiptを持つか。

## 10. 実証計画を追加する場合の既存Scenarioとの接続（設計／全件NOT RUN）

新たなDMケース番号や第二Scenario正本を設けず、前節DM-01〜14を既存Ownerのテストへ投影する**比較観測列**を追加する。

| 既存Pressureへ重ねる反例 | 必要な独立Observation | 必ず保持する拒否/unknown |
| --- | --- | --- |
| DM-04 + #2604 AP-01/06 | Published manifest revision、Member actual bytes、Readerの実版、切替時刻 | R1/R2混在、旧Manifestを読んだ後のR2上書きをSuccessとしない |
| DM-05 + HULFT-HUB欠落先行 | Hop設定と各蓄積Attempt・失効、同一File ID順序、期待する配信集合 | 先行データ欠落にもかかわらず`ordered and complete`と称しない |
| DM-07/09 + S3 Event通知 | Key/version、通知Event id/sequencer、受信Byte Digest、Import Unit identity | 重複通知/逆順、ETagをSHA-256と解釈、同名別版を一つのBusiness Eventと誤認しない |
| DM-08 + AS2 signed MDN | 送信元Message ID、署名・MIC/元Message照合、Importer実結果、Provider側別Read | MDN verifiedだけでBusiness Acceptedを主張しない |
| DM-02/03 + Period再処理 | 業務基準期、Source/参照Master版、前回Artifact/Checkpointと本日Attempt | 旧期・旧Master混在、同じJob名での意味的ABAを合格としない |
| DM-11 + 外部Effect不明 | OperationID、Host Grant、Provider Commit Receiptまたは独立Readのunknown | EKI Task Result / exit0 でCommit確認済みへ昇格しない |

### 同一試行の最小Witness（**正式Wire Schemaではない**）

- `acceptedMeaningRef + revision + selectedPeriod + functionOrOperationRef`
- `target/build artifact actual digest + runtime provider/capability evidence`
- `logical dataset + resolved input snapshot + output manifest/member byte digests`
- `publication generation / actual reader version / read byte result`
- `route + hop + transfer contract + attempt + ordered key + receipt scope/time`
- `import unit + business event identity + validation/reconciliation result`
- `effect/commit receipt or unknown + separate provider read + owner business acceptance (if applicable)`

すべてを常に生成物に埋め込む要求ではない。各欄の**Owner・出典・時点・開示可否・未観測**を保持し、利用者に公開不能な秘密・他TenantのデータをGraph/LLM/ログへ無条件出力しない。SHO/Studio/CUIはこの比較票のread-only Projection候補であり、見えるから実Authorityがあるとの判断はしない。

**実証順序（対象Ownerへ引渡す計画）**:
1. #2145の実PostgreSQL 16限定取込みReference、#2604/#2817のArtifact混合世代反例、#2666の実CLI→Artifact byte Reader（限定CI）を**再利用可能か静的点検**。歴史的PASSの証拠資格と現在Source版は分ける。
2. Headlessで単一ローカルBuild/ArtifactSetの`R1→R2` Publisher/Consumerを構成し、途中失敗/Member欠落/旧世代読取を固定。**#2604 Ownerの既存AP試験を主張元とする**。
3. 別のDisposable File/SQLite Consumerで、同一実行AttemptのPeriod・Manifest・Route・Import・独立Readを結ぶ。SFTP/HULFT/EKI/クラウドに依存しない最小の正常/負例を得る。
4. S3やMFT等の**第二Provider**で、実能力の差分（conditional write、multipart checksum、Event再送、同一File ID順序、Spool expiry）を表示。保証できないものをunsupported/unknownに残す。
5. 実HULFT製品や企業間EDI/本番Hostへは許諾・実環境・Owner判断後に進む。製品仕様比較のみで商用運用適合、実同期/配信保証を宣言しない。

### 複数レーンの2026-11-04現状との分担

- #1130 の更新にある **Meaning/Invocation、Build membership、Entry/Adapter、Effect/Authority、Result/Consumerの五Claim**は既存受入を尊重。本ノートB0〜B4は**物理データ配送に特化した分解**であり、第二のGate体系ではない。
- Draft #2788のFunction/Build Unit選択はUI+Jobの物理構成に関する未採用案。本ノートは配信契約を新たなFunction型として確定しない。最初は単一Build/非常駐Goでもよい。
- #2604は設計ArtifactSet一式公開がOwnerであり、本ノートは**その公開結果を別Transport/別Consumerへ引き渡すときの保証の合成**のみを扱う。
- Draft #2744とEKIはResource/Task/Attempt/Lease・Job委任のOwnerを区別。現EKI P0の`parameters`/Effect/Artifact bodyの制約を型付き拒否へ接続する際は#1130のBD-01〜07を再利用。実TEI↔EKI Job I/Oは未実証。
- #2814/Draft #2815のHashIdScheme/Wireの候補は、ProviderのETagや移送Checksumと**目的・バイト対象が違う**。既存Artifact Digest/署名/Canonical IDを変更しない。
- #1180の保存Scenarioと独立Expected、#2514のOracleを正本とする。上記観測票はFixture/Scenarioの第二正本にはしない。

### 追加根拠（一次資料、対象版に限定）

- HULFT-HUB追越禁止と例外: https://www.hulft.com/help/ja-jp/HUB-V3/COM-HUB/Content/HUB_HUB/Fun_HUB/Phbt_OVTK.htm
- HULFT-HUB蓄積・送出: https://www.hulft.com/help/ja-jp/HUB-V3/COM-HUB/Content/HUB_HUB/Fun_HUB/Acc.htm
- S3 単一Keyの整合性: https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html
- S3 conditional writes: https://docs.aws.amazon.com/AmazonS3/latest/userguide/conditional-writes.html
- S3 ETag / multipart integrity: https://docs.aws.amazon.com/AmazonS3/latest/userguide/checking-object-integrity-upload.html
- S3 events at-least-once/out-of-order: https://docs.aws.amazon.com/AmazonS3/latest/userguide/notification-how-to-event-types-and-destinations.html
- S3 sequencer per object key: https://docs.aws.amazon.com/AmazonS3/latest/userguide/notification-content-structure.html
- AS2 MDN/MIC: https://www.rfc-editor.org/rfc/rfc4130.html
- Camel File idempotent/readLock: https://camel.apache.org/components/next/file-component.html
- OpenLineage Run/Job/Dataset facet: https://openlineage.io/docs/spec/facets/

**新しい実行・実証はNOT RUN**。本節は既存main/Issue/外部一次資料を使った静的設計追補であり、正式API、受理済みSchema、HULFT/AS2/S3/TEI実E2E、安全な本番送受信、TEI Resource Scheduler、複数Object Atomic Transactionを実装した主張ではない。


## 11. 2026-11-05 追加調査：保証の合成可能性・Snapshot Commit・外部Effect境界（設計／NOT RUN）

本節は新Core型・実行エンジン・許可権限の導入ではなく、**Consumer要求を満たしたと証明するのに何が不足するか**を既存の型付き関係/Conformanceへ対応付ける調査・受入候補。基準は現行main `36bda76271db4db72ddd41a8566d281f02764b8a`。既存[#1130](https://github.com/hat47x/tei-platform/issues/1130)、[#2542](https://github.com/hat47x/tei-platform/issues/2542)、[#2604](https://github.com/hat47x/tei-platform/issues/2604)、[#2831](https://github.com/hat47x/tei-platform/issues/2831)、[#1180](https://github.com/hat47x/tei-platform/issues/1180)の実装・受入を置換しない。

### 11.1 合成できる保証と合成できない保証

計画時の `required claim`、Providerの `declared capability`、条件付き `qualified evidence`、実行後の `observed outcome` を別にする。**保証は強さ一軸ではなく、対象をもつ制約の集合**として扱う。次はテスト/説明のための表であり、正式ABI/新Schemaではない。

| 保証要素 | 適用範囲／必要な根拠 | 合成前に満たすべき条件 | 合成禁止の典型 |
| --- | --- | --- | --- |
| Byte integrity | 元のbyte領域、転送前後の変換、Digest scheme、取得した実byte | 同じbyte列または説明済み変換境界で検証したこと | ETag＝SHA-256、圧縮前と暗号化後のbyte digestを同じものとして比較 |
| Visibility | Publisherの一回の世代/Member集合、Consumerが読んだversion | generation固定、全Member読取/検証、失効前の保持期間 | 単一Object atomic writeから複数Objectの一式可視性を推定 |
| Ordering | file-id/partition/key/route、各hop内の対象イベントと時点 | すべてのhopが同一の順序対象を保ち、変換・再配送で順序keyを失わないこと | 1 hopの追越禁止⇒全世界の業務イベント順序 |
| Gap-free completeness | 期待するManifest/件数/連番/期間、水位、全入力の実取得 | Independent expected setまたは受信完了境界を示すSourceの証拠 | `ordered`、0件観測、spoolのACKだけから欠落ゼロを推論 |
| Idempotent effect | Business OperationID・Scope・Period・保存Owner・実Commit/Receipt | 同一要求IDとstable intent/fenceを用いた再照会・再処理契約がProvider側で成立 | 同一byte/同名Jobを新しいOperationIDで再実行して一回性を主張 |
| Current authority | 送出・読取・Import・Commitごと、主体/対象/有効版/時点 | 各境界の実Host/ProviderでGrant・対象版を確認 | Build時のCapability、MFT ACK、EKI Task resultで更新権限を推論 |

**停止規則:** 同じ必要保証に対して、証拠が `unverified`、対象違い、版不一致なら計画/実行/受理の該当Claimを `unqualified` とする。`unsupported` と `unknown/partial` と `not-applicable` は異なる。あるProviderの欠落を自動的な強度低下や別経路への暗黙Fallbackで隠さない。全体のClaimは対応する全辺のWitnessを必要とする（「各辺が成功」の単純ANDだけでは、異なる主語・世代・粒度を誤結合する）。

### 11.2 Iceberg型Snapshot Commitからの知見：公開ポインタと保持期間

Apache Icebergの仕様は、既存ファイルをその場で更新せず、Snapshotの参照するデータファイル/Manifestを定め、**Catalogのtable metadata pointerのatomic swap**をCommit境界にする。Concurrent writerは基準となるMetadata versionからずれれば再評価し、Readerは選んだSnapshotを参照する。これはTEI全体をIcebergへ依存させる提案ではなく、[#2604](https://github.com/hat47x/tei-platform/issues/2604)の一式公開候補を評価する参照モデル。**Catalog/Storageが実際にatomic swap・条件付き更新を提供することが前提**であり、一般のManifestファイル名変更だけで保証をコピーできない。

一式公開の説明可能な最小経路（物理方式はOwner選択待ち）:

1. `R2`のすべてのMember byteを、`R1`から識別できる不変の配置先へ作り、実byte DigestとMember一覧を準備する。
2. `R2 Manifest`の全Memberを、同じGeneration/Scope・期待するData Snapshot/Periodと照合し、**実Readerが参照する形のMember集合**が欠けていないことを確認する。
3. 公開ポインタを旧版`R1`から`R2`へ切り替えるとき、Writer側の基準版をCAS等で検査し、競合時に一方を無言で上書きしない。公開時の原子性は実ProviderのOperation単位に限定する。
4. Consumerは選択したManifest generationをpinし、列挙したMemberの実byteと期待Digest・versionを読取検証する。ディレクトリ全列挙や通知順を一式Memberの正本にしない。
5. Readerが`R1`をpinした後に`R2`が公開されても、`R1`のMemberを必要な保持期間は読めるか、失効/削除が発生したら `stale/unknown`として失敗にする。**古い世代のGarbage Collection／retentionは、一式公開とは独立の保証条件**。成功したManifest切替だけからReader継続可能性を断定しない。
6. Consumerが全Memberを再読取した証拠と、データ行・業務イベント・Provider Commitの正しさを別に評価する。

特に`R1`のReaderがMember取得途中に、`R2`公開を契機とするGCで`R1`の残りのMemberを消された場合を、#2604 AP-06の受入・失効条件へ引き渡す。`complete manifest`があっても、`generation-retention`が不明なら「消費可能」のClaimは未実証。

### 11.3 Kafka Exactly-onceの境界：Offsetと外部更新を一つの効果と呼ばない

Apache Kafkaは、Kafka Topic間ではtransactional producerとconsumer offsetを同じtransactionに含める方法を説明する。一方、**Kafka外部SinkへのExactly-onceは外部Sinkとの調整が必要**と明記する。TEIではMFT・Jobを含む全体へ`exactlyOnce=true`を拡張しない。

- `TransferAttempt`は何回あってもよく、`ImportUnit`の二重適用排除・`BusinessEffect`の二重確定排除は別の責務とする。
- 不明応答の後、**同一OperationID/同一Intentで実Providerが保存したReceiptを照会し、Commitの有無と要求同一性を確かめる**経路は、無条件再送とは区別する。実PGの限定証拠である[merged PR #2853](https://github.com/hat47x/tei-platform/pull/2853)は同一OperationIDによる取引再照会・takeover後の版不変を示す。ただし永続Receiptの保持、別Provider、本番Grant、外部MFT連鎖の一回性へ自動一般化しない。
- 同じ`RunPlan`名/業務期に対し新OperationIDを発行して再確定する行為は別のEffect要求。`RunKey`、`AttemptID`、`FenceEpoch`、`OperationID`、`ImportUnit`、`TransferAttempt`を別参照として保持し、どれがIdempotencyの正本なのかOwnerが指定する。
- 実DBと外部SFTP/MFT/S3を単一Txにできない場合、双方の結果を一度の「成功」に丸めず、Outbox/Inbox/冪等Consumer/独立Read等から選択した明示的な契約と不明時Recoveryを検証する。特定メッセージブローカーや中央Outbox実装をCoreへ要求しない。

### 11.4 最新mainのRunPlanとETL/MFT研究の境界

[RunPlan設計・進捗](RUN_PLAN_AND_UNATTENDED_EXECUTION_DESIGN_2026-11.md)では `run-plan` 宣言、`internal/runlog` のRunKey/Attempt/Fence、`execution.RunPlan`、ステップ再開、`CombinePreconditions`、`ProjectOutputs`/帳票整形、Function定義書への投影が2026-11-05時点で記録されている。これは**製品Sourceに実装のある局所経路**であるが、全対象での実DB永続化・実MFT輸送・全件的なWorker外部実行/配布資格ではない。

- `ProjectOutputs`の行数は**実Datasetの一式内容・独立Consumerが読んだバイト・配送受理の証明ではない**。0件とReader失敗を区別する現在実装を維持し、追加のDelivery成功を自動生成しない。
- `CombinePreconditions`の成功は、渡されたチェックが通ったことの証拠であり、**Provider/Hostの現在Grantや外部送出と受領の現在性を全件調べた証明ではない**。不要なStepにPrincipalを強制せず、外部Effectを行う入口だけ実Authorityを検査する。
- [Self-build #2831](https://github.com/hat47x/tei-platform/issues/2831)の宣言したBuild Graph対実行工程の完全性を、**ETL/Transfer生成先の実依存Closure**の調査へ再利用できる。Build-onlyの完全性はRuntime/Egress/外部Partnerの実動・不在の証拠ではない。新Build Graph正本を作らない。
- [#2823](https://github.com/hat47x/tei-platform/issues/2823)のWorkspace Manifest/Source closure未検査は、パイプラインのSourceが`missing/unknown`でもvalidate greenと誤認する可能性を示す。Source closureは一式公開の前段の独立Gateであり、ManifestのMember照合だけでは元Sourceの選択完全性を証明できない。
- GUI/StudioのDataflow・Transfer図は標準構造のread-only Projectionとして後段に扱い、解決済みScope/Source/Provider/Build/Attemptの境界が分からない場合は`opaque/unknown`と表示する。図表の完全性を実送出/秘密の不存在証明にしない（Topology設計、#1124/#2516）。

## 12. 最初の具体化：一つの世代、二つのConsumer、実ByteとReceiptを別収集（全件未実行）

**既存Scenario/Pressure再利用**：新しい正式`DM-15`・新Scenario・別のOracleを作らず、DM-04/07/08/09/11 と #2604 AP-01/02/06、#2542の既存ImportUnit、#1130 RunPlan、#1180保存Scenarioへ必要な追加観測項目だけ渡す。

| 既存ケース | 実装Ownerで取得するべき一試行の証拠 | 負例と停止 |
| --- | --- | --- |
| DM-04 + AP-06 | 旧`R1` pointer/read pin、`R2` publish、`R1` Member実Byte再読込、GC時の結果 | 新Manifest公開後の旧Member削除でReaderが失敗した場合、「受領済み」と表示しない |
| DM-07/09 + #2542 | 実配送Attemptを2回、同一/別ImportUnit key、業務Event ID・受理時点の個別Receipt | 同一byte別Business Eventを一つへ畳まない。ファイルPathだけでDedupしない |
| DM-08 + #2604 | `R2`全Memberのbyte・Consumer1とConsumer2の独立Read、Import validationと業務受理Receipt | 一方が`R1`で他方が`R2`、あるいは業務型/Period違いを正常な一式配信としない |
| DM-11 + #2853 | 同じOperationID/IntentのCommit unknown、Receipt再照会と外部配送の別Attempt | 同一IDでの安全照会と、新OperationIDによる再Effectを混同しない |
| DM-02 + #1130 | RunKey/処理基準日/Rule版、RunPlanの行数投影、実成果物と実Sink Readの時点 | 出力行数が期待値と一致しても、別世代/別期の実配送をQualifiedとしない |

**優先順の候補**: P0=既存#2604 AP-06の`Reader pinned generation + retention`を比較票に追加し、#1130の実RunPlan出力と分離。P1=一つの本当の別Process Consumerで全Memberのbyteを読めることを確認。P2=#2542の取込みReferenceを再利用して重複配送・受理を接続。P3=任意の別Transport（SFTP/S3/MFT等）の条件付き能力で、どの保証が失われるか比較。**いずれもこのDraftで実試験したという記録ではない。**

**受入/撤回条件**: `Proof requirement`の概念整理によって、実Consumerが失った版/Member/順序/業務効果を1件でも追加検出でき、従来より少ない独立の定義・検証重複で説明できるか測定する。既存#2604/#2542/#1130の検査と同じClaimにしかならないなら、新しい専用Role・schemaは導入しない。採用する場合も正式な必要条件とOwner判断・Conformanceは既存契約側で決める。


### 12.1 P0引継ぎ票：閉包と実読取を同一試行で照合する

この表を**既存EF-01/EF-02・AP-01/02/06・RPL-05の実装Ownerへ渡す最小の作業票**とする。ここでいう「二Consumer」は、(A)既存Artifactを読むConsumerと、(B)その実byteを独立に読み取る別プロセスまたは取込みReference。Self-build Consumer (#2831)との型付き構造比較 (#2880)とは**別の比較軸**。AとBが同じ実装を内部で共有するだけなら独立観測と主張しない。

**固定する最小例**（テスト用の論理的な入力。実Fixture・CLIは既存Ownerが所有）：

- 同じ業務期Dの期待Member集合は、明細 F と件数・合計 T の2件。R1=(F1,T1)、R2=(F2,T2)。R1/R2は論理世代であり、物理Object名やHashId表示文字列を正本Identityへ昇格しない。
- Reader AはManifestの対象世代・Member一覧をpinし、列挙した全実byteを検査。Reader Bは**別の取得経路**から実byte/世代を再取得。公開前・公開中・公開後・旧版削除時点の観測を混同しない。
- 期待する配送宛先が増える場合、Recipient SetのScope/Revisionと各宛先Receiptを別に固定。HUB蓄積ACKと宛先側のImport/独立Readを同一列に上書きしない。
- Source/Role/Relationの所属ScopeはQualified Source由来で固定し、別Scopeの同名File/Relation、または非公開の端点から対象を推測しない（#2881の合成26件はこの境界の参考証拠であり実Source証明ではない）。

| 既存Ownerへ渡す検査順 | Required / Declared / Selected / Actually consumed / Observedを分けた最低限のWitness | 独立Expectedと先へ進めない条件 |
| --- | --- | --- |
| **0. Source closure** (#2823、#2620、#2880) | 期待するF/Tと期間D、Source revision、declaring Scope/Relation Role、選択Provider、欠落/未読込Source一覧 | Source欠落なら「0件」「対象なし」とせずunknown。UIの限定表示から未宣言を推定しない |
| **1. Render** (#2604 AP-01/02) | 選択済みGeneratorと実Input、出力R1/R2 Member byte/Digest、ManifestとFindingsを別欄 | ProducerのComplete/exit0を実Publisher/Receiver PASSへ昇格しない |
| **2. Publish/Pin** (#2604 AP-06) | 物理世代R1/R2と公開Pointer/取得可能期間、Reader Aが固定したManifest Revision | R1 pinの後R2公開。R1を通読できるか、失効ならstale/unknownで停止。R1 Manifest＋R2 Memberをsuccessにしない |
| **3. Independent Read** (#2542 EF-01) | Reader A/Bの実Byte Digest、全Member、取得時刻、対象Snapshot/Scope、読取経路 | 同一世代かつ全件読めた場合だけ**そのConsumerが読んだ物理集合**をqualified。Reader Bの未観測はnot-runでありPASSではない |
| **4. Deliver/Import** (#2542 EF-02/03、#2145) | Hop/宛先ごとのAttempt・ACK・実受信byte、ImportUnit、Business Event、拒否/保留理由 | ACKだけでは全宛先受理・業務Event確定を証明できない。同Byte別Eventや遅延F2を自動Dedup/自動確定しない |
| **5. Business/Effect** (#1130 RPL-02/05、#1174) | RunPlan業務期、必要Guard/Supplied Guard、実Request/Operation ID/Commit Receiptまたはunknown、別Read、Owner判定 | Reader完了やRun completedはBusiness Acceptedでない。Blockedより前のEffectは残り得る。実Current Grant/Commitは別Ownerの受入 |

**固定反例の期待票**（既存Fixtureへ追加する観測の候補であり、新IDを付けない）：

| 状況 | 物理Reader claim | 配送claim | 期間Dの業務claim |
| --- | --- | --- | --- |
| R1全MemberをAとBが実読取、F1/T1一致 | Reader A/Bの指定世代の物理集合に限りqualified | 送信Attempt未実行はnot-run | 未実行/未受理を保持 |
| AがR1をpin後、BがR2へ切替、T1をGC | AのR1通読失敗ならstale/unknown、BのR2は独立評価 | 別 | R1/R2混合を業務上のD確定にしない |
| 送信元がHUB蓄積ACK、宛先Xが受領、宛先Yがエラー | Xは実byte取得の有無で判定 | X/Yを別資格。全宛先Delivery Complete=false/unknown | ImportとOwner判定が無ければ未受理 |
| F1とF3が先着しF2欠落のままWatermark通過、D+2にF2到着 | 存在するFileの読取とD全件性は別 | 遅延F2のAttempt/Receiptを別版で取得 | 先行計算は暫定/partial。Dが締め済みなら訂正/再受理をOwnerに委ねる |
| 実ProviderがCommit後ACK喪失、同一OperationIDでReceipt照会 | Byte照合は対象ファイル限定 | MFT ACKとは別 | 対象/Intentが同じ実Receiptの範囲のみ再適用を避けられる。別ID/ABAは未保証 |

**P0成果物の最小Evidence**: 既存Scenario ID/Revision + Source/Period D + Expected F/T + selected Target/Provider + Renderer実消費Input + ArtifactSet/Manifest/Member実byte + Reader A/B各結果 + missing/unknown + Test Attempt/Env。実Policyの公開承認済みSliceだけを投影し、機密Source path・Recipient情報・隠し件数をログ/UIへ漏らさない。Effect/Owner列はP0では**not-run**を記録し、空欄の成功扱いをしない。

**引継ぎ順（Owner重複を避ける）**: (i) #2604/AP-06の既存Writer Characterizationと同一Reader A、(ii) #2542/EF-01で別Consumer Bの実byte検証、(iii) EF-02/03で疑似RouteのRecipient別結果、(iv) RPL-05でRunPlanのSource Revision/Period/Outputと結合、(v) 実Effectが必要な時だけ#1174を使う。#1145のInvocation wire、#2933のBatch/Streaming最適化、#2831のSelf-build full closureと性能/可搬性の証拠は各Ownerに保持する。

**最初のコード変更を開始する条件**: 既存#2604のWriterに対し「Reader A/Bが何を読むか」がテストで定義でき、OwnerのFixture/Expectedが選択できること。そこに達する前に統一Completion Engine・専用Batch DSL・新Proof Registry・共通Schedulerを作らない。本PRはこの観測の**設計差分のみ**であり、P0の実2Process/Provider/CIやOwner業務受理を新たに実行したものではない。


### 12.2 既存AP-08〜16を再利用するP0依存と最初の検証器変更（2026-11-06、NOT RUN）

**既知ギャップのOwner確認**: Manifest EntryにPath/Raw-Content-Digest/Generator/Unresolved詳細がなく、`Artifact.Digest`は単なるファイルbyte SHAでないという不足は、[#2604 AP-08/09](https://github.com/hat47x/tei-platform/issues/2604#issuecomment-6576536977)と[#2604 AP-10/11](https://github.com/hat47x/tei-platform/issues/2604#issuecomment-6585308926)で**2026-11-02に既出**。さらにBody multiset一致でもPath↔Body入替えを見逃す反例は[#2604 AP-12〜16](https://github.com/hat47x/tei-platform/issues/2604#issuecomment-6585644900)で**2026-11-03に記録済み**。これは今回新規発見した第二仕様課題ではない。既存Issueの再起票・新Case ID付与・追加Manifest Schemaの先行採用はしない。

**今回の新しい横断照合**（main `f275843b2bd35117f3bbb8d6090a59678d3c33ed` で固定して静的確認）:

| 実装・履歴 | 現行Fit | 次のETL二ConsumerのGap |
| --- | --- | --- |
| `internal/artifact/artifact.go` blob `4a56949d683248185c2d51a22350cefd250362da`／`manifest.go` blob `c45e2b2397974c7b5361ddfae8c65f45c91eb226` | ProducerのArtifact envelope／Manifestは内部指紋・Unresolvedを持ち、出力前の検査に再利用できる | 保存された実BodyとManifest Entryとの紐付け・再検証には物理Path/Byteと必要なEnvelopeの情報が別に必要（既存AP-08〜11） |
| `cmd/tei/cli_operations_artifacts_publication_test.go` blob `6519e47f5af803467f34dfa46decfff97a069e3c`、[merged PR #2817](https://github.com/hat47x/tei-platform/pull/2817) | AP-01〜03の上書き順・旧ファイル残存・命名衝突を実Go試験で記録済み | Helper `publicationSet`は`WorkspaceSet.Manifest`のゼロ値を使用。**正規`NewManifest`＋Findings＋実Member検証の陽性Fixtureではない**。AP-01〜03の成功を有効なManifestのReader受理に転用しない |
| `plan/fixtures/capability-handoff/verify_real_cli_revision.py` blob `918f4cafb136090f8f75a08b041489b0d1da3ffa`、[merged PR #2649](https://github.com/hat47x/tei-platform/pull/2649) | 実CLI stdout `WorkspaceSet` とfresh-directoryの`manifest.json`/Bodyを比較する既存独立Python Consumer経路がある | `verify_reader`は`Counter(expected Content bytes)==Counter(actual filename-selected bytes)`。**Pathを固定した本文交換で成功し得る**ため、実Path→Role→Bodyの対応・世代・Recipient資格は未証明 |
| [merged PR #2630](https://github.com/hat47x/tei-platform/pull/2630) のConsumer相対Expected設計 | Consumerが必要なMember集合と実Producer出力集合を分ける計画が既存mainにある | 用途別Required Memberの独立Expectedを、実Readerが開いたPath/Byte・Scope/時点へ結び付ける実結果は別 |

**#2827から引き渡す最短の修正/検証順（#2604 Ownerの既存ケース・コードを使用）**:

1. **P0-A 実Readerの検査強化**：既存`verify_real_cli_revision.py`と`tei artifacts`の実出力を継続利用。`Counter(Body)`の成功に加えて、独立のテストExpected `(qualified logical member, kind, required relative path, content byte SHA-256)`を固定し、**実Pathごとにbyte照合**。ファイル名を変えずに本文だけ交換する既存AP-12負例で拒否する。PathをWriterの同じ`artifactFileName`関数から無批判に再計算して自己証明しない。
2. **P0-B 有効なManifestの陽性**：`artifact.NewManifest`で同一InputRevisionの2 Member/F/Tを生成し、`Findings`も別に照合。現AP-01〜03のゼロManifest特性Fixtureは不適切な前提として保持したまま**陽性受領のFixtureとしては利用しない**。Full `Artifact` envelopeがあるCLI stdoutとの同Attempt照合と、保存Memberのみでできるbyte検査を別の証拠欄に記録する（AP-10/11）。
3. **P1 世代/Required閉包**：R1/R2一式・旧Member残存・Consumer pin/GC・Path衝突・異Scopeを、既存AP-06/08/09/12〜16と[#2630 AS-01/02](https://github.com/hat47x/tei-platform/pull/2630)のExpectedへ結合。「Body一致」「Path束縛OK」「同世代全Member受領」「用途相対Expected充足」を別表示。**この段階で初めて二Consumerの物理資格**を主張可能か判定する。
4. **P2 Delivery/Business**：#2542 EF-01〜03へ、Recipient別Receipt・Import Unit・Period/Rule版を引き継ぐ。#1130 RunPlan・#1174 Effect/Ownerは別。公開Artifactの物理Readと業務受理は併合しない。

**優先変更の所在**: AP-12の既存Python Reader強化が最小で、Core`ManifestEntry`やファイル公開方式への変更はOwnerがAP-08〜16の比較後に判断する。#2827には新Reader実装、第二Fixture/Scenario/Manifest Registryを設けず、既存#2604/#2630/#2542の受入依存を明示する。現`Artifact.Digest`（full envelope）とphysical Body byte SHA-256、さらに可変長Hash IDは異なる役割である。

**資格**：ここまでの差分は既存Go/Pythonの静的読取とIssue/PR照合だけ。AP-12の新たな実行、正規Manifest＋別Readerの同Attempt E2E、MFT Recipient/実PG/Owner受理はNOT RUN。既存merged testsとRepository Healthの合格は、その限定範囲のみの実証。

### 12.3 2026-11-07 既存AP-12の限定Executable Reference接続

[本設計のmain統合 PR #2827](https://github.com/hat47x/tei-platform/pull/2827)後、受入実証を既存[#2604 AP-12/13](https://github.com/hat47x/tei-platform/issues/2604#issuecomment-6585644900)へ委ねたまま、**小さな候補[Draft #3256](https://github.com/hat47x/tei-platform/pull/3256)**を追加した。これは**一般のManifest→Path方式の採用ではなく、公開`ledger` Fixtureの限定Reader検証**である。

- **入力／Owned Scope**: `reference/standard-crud/workspace`、Application `ledger`。既存`tei artifacts`のfull ArtifactSet/Manifest出力と、`-out`のfresh directoryの同一試行比較を継続。Writerが別途呼ぶ`artifactFileName`をExpected生成に再利用せず、3つのtable（`public.notes`／`public.orders`／`public.order_archive`）、1つのfunction（`ledger`）、operation surfaceの5相対Pathを**Reference固定のExpected Layout**とする。別Applicationへ一般化しない。
- **P0-A負例**: 二つのTableの物理ファイル名は固定のままBodyを交換する。旧Python Readerの`Counter(expected Bodies)==Counter(actual Bodies)`はTrueでも、`path→member→byte`を照合すれば拒否（AP-12）。Rename／余剰／Directory／欠落／重複／Role改変／Body改変も個別に拒否する。
- **局所結果**: Python helper＋単体試験をローカルで**9/9 PASS**（2ファイルをGitHubで再取得し、ローカル実行版とGit blob SHA一致）。これは**合成FixtureのReaderロジック**の結果であって、Go ArtifactSetの正式Manifest出版、複数Consumerの原子的一式読取、HULFT/PG/Business Owner受理ではない。
- **既存実CLI CI**: [`tei-one-revision-capability.yml`](../../.github/workflows/tei-one-revision-capability.yml)へ上記9試験を追加し、同Workflowの実Go CLI build／実`ledger` Artifact/Manifest出力を新Readerへ接続。**exact-headのCI合否が観測されるまでは実CLI検証未成立**。Repository Healthや別PRのCIを代入しない。
- **P1/P2への引継ぎ**: #2604 AP-06/08〜16のR1/R2世代切替・旧Member GC・Manifest自体のPath/Content byte Binding資格、#2630の用途別Required Member、#2542 EF-01〜03のRecipient/ImportUnit、#1130のRunPlan・#1174のEffectを独立した未成立Claimとして保持。Path/byte Witnessの成功だけで`Artifact.Verify`をdiskから再構築したとも、`Manifest.Complete`がConsumer必要集合全件を保証したとも称さない。

**採否**: 一般のPublication Mapping、標準RoleやManifest Entryの変更、任意Transport/MFT Adapterは、現在の5ファイルFixtureだけでは根拠不足。別Reader・別Host・第二Scenario正本を作らず、#2604 Ownerが実証と互換性に基づき方式を選択する。

### 12.4 2026-11-08 結果確認と、Byte／Envelope境界の独立負例

**PR #3256 の最初のexact HEAD `bcf77d7bd8dd1dc7e9242a6d708761c1f613693a`についてGitHub APIで実結果を確認**:

| 実行根拠 | 成立した範囲／別資格 |
| --- | --- |
| [one-revision actual Go CLI＋Python Reader CI #42348103565](https://github.com/hat47x/tei-platform/actions/runs/42348103565) | completed/success。exact HEAD checkout、pinされたGoツールチェーンによる実CLI build、9件のPython単体、同一公開`ledger` WorkspaceのArtifact生成と実Path/Byte Reader、改名によるSource revision差の実検証。これは**一つのfresh-dir**での受領に限る |
| [Repository Health #42348103549](https://github.com/hat47x/tei-platform/actions/runs/42348103549) | completed/success。上記実CLIとは別の全体健全性検査 |
| [Dependency scan #42348103617](https://github.com/hat47x/tei-platform/actions/runs/42348103617) | completed/success。root/studioのgovulncheckと参照Workbench等のnpm auditのWorkflow結果。実Provider接続やBusiness Policyの安全証明ではない |

追加する**同じ既存Unit Testへの負例**（本節執筆時点で新HEADのCIは別に必要）:

- **AP-01/06（読取側の反応）**: R1のManifestを保持しつつ一つのMemberだけR2のBodyに置換。限定ReaderはR1受領資格を拒否することを確認する。ただし**既存Writerが必ず原子公開することの実証ではなく**、失敗後のRecoveryと旧版pin/GCの保証でもない。
- **AP-10/11（過剰資格化の拒止）**: Bodyを同じに保ち、Rule Version／Generator／Unresolved詳細だけを変更。Byte限定Readerは同じBody digestを返し得るため、結果の資格を`ledger-fixture-path-to-bytes-only`に限定し、`Artifact.Verify`・意味の同一性・業務受理を出力しない。**同じbyteだから同じArtifactとはしない**。
- **実装担当を増やさない**: #2604が物理Publication/Consumer Reader、#2482がArtifact Envelopeの内部検証、#2630が用途別RequiredMember、#2542が外部Recipient/Import、#1174がEffectを担当。#2831／PR #3313の外部Module opt-inはBuild Sourceの独立証拠であり、Ledgerのファイル読取PASSをSelf-build全閉包や実MFT保証へ移さない。

**次の受入順**: (i) #3256 exact HEADの11 Python例＋actual Go CLI CI結果、(ii) #2604が所有する**正規Manifestを維持したR1/R2公開競合と独立Consumer pin/GC**、(iii) #2542 EF-01〜03で実送達/ImportUnit/期間、(iv) Owner承認付きEffect。新Renderer、Publication Mappingの正式API、専用Completion Engineの先行採用は不要。製品E2E・Windows/FS差・実MFT・業務Owner受理は引き続きNOT RUN。


### 12.5 2026-11-08 正規ManifestでのWriter実挙動と旧世代pinの残余（P1候補）

**背景と変更範囲**: 既存#2604 AP-01〜03のmain統合済み`cli_operations_artifacts_publication_test.go`は`WorkspaceSet.Manifest`をゼロ値のままWriterへ渡す**逐次書込の特性試験**。一方、[#3256](https://github.com/hat47x/tei-platform/pull/3256)のPython Readerはfresh-directoryの正常Publish＋Body/Pathに限定した資格を得た。**両実装を含む最初の小さな接続**として、同一Go Writerで`artifact.NewManifest`が成功するR1/R2の実Fileを保存し、次の3つの観測を追加した。今回の試験の採否・CIは更新HEADごとに別記する。

| 既存Ownerのケースへ渡す観測 | 実行方法（局所Go実ファイル） | 証拠の上限／禁止する飛躍 |
| --- | --- | --- |
| **正の基線** AP-08/10 | 2 Member `a/b`、同一R1 InputRevisionで`Artifact.New`→`NewManifest`→`Verify`し`writeArtifactFiles`出力。保存された`manifest.json`をファイルから再読取・Verifyし、双方の実Bodyを比較 | **Go内のdisk read**であって別Process Consumerや独立に受理したPublication Bindingではない |
| **途中失敗** AP-01/06 | R1は正規Manifest/2 Memberで公開。R2で`a/b/c`を選び、新Member `c`宛先へ事前にDirectoryを作り書込み失敗を注入。`b`はR2に置換され、R1 Manifestは改訂されない | 論理Manifest Verify成功でもR1の物理Read一致は成立しない。**観測上のReader拒否を検査**するだけでAtomic Publisherの修正ではない |
| **正常R2への移行と旧pin** AP-06/09 | R1のManifest版を固定後、同じ出力Directoryへ正常R2を書き込み、旧Memberの保存Byteが取り出せるか比較 | R2が正常でもこのDirectoryにはR1の実Body保持機構がない。旧版Readerは別世代Pin/Retention契約なしに同一版と主張しない |

**最小の計画差分**: テスト専用`cmd/tei/cli_operations_artifacts_valid_manifest_readback_test.go`（production Writer変更なし）を既存`tei-one-revision-capability.yml`のexact-HEAD Go unitへ追加し、既存Python Readerの**11ケース/実CLI**とは別Stepで結果を取得する。これらを#2604のAP-01/06/08/09へ戻す。**次の実製品P1**は#2604 Ownerがimmutable generation／stage→pointer／pin+verifyのどれを選ぶか、実Consumer2プロセス・FS条件・GCに照合して判断する。Providerの整合・暗号化/署名・Business Owner認証、MFTの実配送/Importは#2542等の別レーンを保持する。

**他チャットとの契約境界**: [merged #3313](https://github.com/hat47x/tei-platform/pull/3313)は選択した外部Go Moduleをopt-inでBuild Source Closureへ供給できることに関する実装であり、Artifact Publication／MFT Transferの一式性を自動保証しない。Build閉包とReceiver実Readは対象・Attempt・実質的な保証が別。#3244外部Registry/Release Readerの署名・現在Trustも別Ownerであり本テストに含めない。

**状態**: 設計は今回の試験候補を記録する。追加Go 3件のCI、最新mainとのmerge時資格はHEADごとに確認する。新Core型、第二Publisher、汎用Manifest Mapping、Scheduler、Hostは未導入。外部Consumer/実MFT/Business Effect/Owner採用はNOT RUN。

### 12.6 2026-11-09 期間状態の原子的保存と一式公開を分離する

**今回の追加根拠**: 2026-11-09確認main `3c8bebefef30a9407fef71c0e6a5a76680f46719` では、[merged PR #3718](https://github.com/hat47x/tei-platform/pull/3718)により `tei periods -advance`（`open→closing→closed`）と `internal/periodstate/writer.go` のtemporary file+renameが実装済み。これを[#1130 RunPlan](https://github.com/hat47x/tei-platform/issues/1130)／[#2542 EF-01/04](https://github.com/hat47x/tei-platform/issues/2542)の業務期へ接続する。ただし**期間状態ファイルの一回の置換原子性と、Artifact Setの一式公開原子性、同時実行の排他・CAS、実業務の確定は別**。

| 境界 | 現mainで確認できる実装 | 現時点で成立しない資格・反証対象 |
| --- | --- | --- |
| **期間状態の単一保存** | `periodstate.Writer.Write`が既存JSONを読み、渡された期間を上書きして同一directoryの一時ファイルを`os.Rename`する。既存テストは単一Writerで他期間保持・新規作成・未知state拒否を検証 | renameにより個別JSONファイルの途中書込み露出を抑えるが、複数Writerの**read/modify/write全体がCAS/直列化された**証拠にはならない。ファイル/ディレクトリfsyncやWindows/Network FSの耐久性・原子性も未証明 |
| **期間遷移** | `operations.NextState`は宣言済み状態からの一段遷移だけを許し、`closed→open`は更正のため明示。CLIは現在の状態を読んで次状態を計算し、別呼出しのWriterで保存する | 二CLIが同じ`open`を読んで両方`closing`へ書く場合、**古い状態を前提にした二つの遷移の成功**を排除する現在Grant/Expected state revision/CASがない。複数期間の並列更新なら一方の状態を他方が消すlost-updateも条件付きで成立し得る。実並走負例はNOT RUN |
| **Artifact Publication** | #3256により現`writeArtifactFiles`のR1正常2Member→R2部分失敗／R2正常上書きを正規Manifestで観測した（旧HEAD実CLI focused CI成功） | **期間stateファイルのrenameをArtifact Setの公開完了・旧版pin・Retention/GC・外部Recipient実読取の保証に転用しない**。公開Memberは別々の`os.WriteFile`であり、Manifestは最後の別書込み |
| **Business Period D** | R1/R2のArtifact Generation、Data Window/Watermark、RunPlan Guard、Period Stateを個別に表現できる | `closed`への状態遷移だけでSource必要File/明細/合計の網羅、全Recipient受領、実Import/Commit/独立Read、Owner受理までが同時に成立したとは言えない。更正時は新しい業務Operation/Generation/ReceiptとOwner判断が必要 |

**優先度P1の有限反証（既存Owner配分）**：

1. **#3718→#1130**: 二Writer A/Bが同じPeriod State revisionを読んで各々別の期間を進める競合を、同期点を持つ参照Fixtureで注入。結果が`Aのみ`または`Bのみ`となる経路なら、単一Writeの原子性と状態更新全体の直列化を分離して**unqualified**へ。RPL-02/04やExisting Period Transitionの受入を再利用し、ETL固有の新Authorityを設けない。
2. **#2604**: 正規Manifest付きR1/R2を別Process Consumerに実際に読ませ、Pin中のR2切替・部分失敗・Retention/GCを対象OS/FSごとに測る。現#3256のGo内Disk ReadとPython Fresh-dir Readを別Attemptと明記し、両者のPASSを原子的一式公開へ合成しない。
3. **#2542 EF-01/04＋#1130 RPL-05**: 業務期Dの`closing`/ `closed`と必要F/Tの欠落・後着、Watermarkと実Recipient ACK/ImportUnitを別列にする。`closed`の観測後にF2が遅延到着しても自動再Accept/自動再Effectをしない。Source/Business ScopeのOwner承認を保留し、Reader/Producer/Providerごとに未実証を表示する。
4. **#2831 / PR #3731（別Owner）**: 同一Planによる二回のBuild byte再現候補は**Build Determinism**の資格。完成したArtifactを実Consumerが読んだか、期間Stateと同一Attemptか、配送されたかとは別。PR #3731はこの観測を`reproduced`とし得るが、L2-b／G4を未資格としている。PRの正式merge/製品実行資格とは混同しない。

**CIの正確な履歴**: [#3256 HEAD `32782a42011ed0d3642c6344caddd845fa3c3a1c` の focused CLI #42434906001](https://github.com/hat47x/tei-platform/actions/runs/42434906001)は成功、[同HEAD Repository Health #42434905882](https://github.com/hat47x/tei-platform/actions/runs/42434905882)は失敗。GitHub Job logの唯一の失敗Stepは`gofmt -l . is empty`、指摘は`reference/dsl-language-plugin/everything_data_cross_surface_test.go`（本PR変更対象外）。失敗ファイルBlob `97483b653a1f04ffe919e70dbc075dfcc24d4346`はPRの分岐元と候補HEADで同一。mainは[後続#3398等の整形](https://github.com/hat47x/tei-platform/commit/7cd70eaee39e608fd0401649bb4c68a7b1d7b1e4)を取り込み、[2026-11-09 main Health #42573811654](https://github.com/hat47x/tei-platform/actions/runs/42573811654)は成功。ただし**PRにmainを統合した新HEADを検証するまではRepository Healthの成功へ昇格しない**。

**設計判断**: 原子性を広範囲に一括要求する共通Gateを作らず、(a)単一ファイル置換、(b)read/modify/write競合、(c)ArtifactSet一式公開、(d)配送/取込み、(e)業務Effect/Owner受理を別の観測・受入として扱う。MFT専門DSL/統一Host/万能Scheduler/第二Manifest・Authorityの増設は現段階で正当化されない。

### 12.7 2026-11-09 公開Generation pinの実CLI→独立Python Reader再取得

**今回の追加試行（#2604 AP-06/09/12へ接続、採否はCI確認後）**：main統合済みCLI `tei artifacts -out`と既存#3256の`verify_real_cli_revision.py`を再利用。公開`ledger`の同一Workspace copyをR1として実Go CLIで`-out <dir>`へ公開し、独立Python側が**保存された`manifest.json`の原バイトをpin**して各MemberをReadする。その後、既存MR-03の`semantic/sales/order.tei`を同byteのままrelocateし、Source treeとInputRevisionが異なるR2を実Go CLIの別呼出しで**同じ`<dir>`へ上書き公開**する。R1/R2それぞれのCLI stdout Manifestと保存Manifest/Memberを照合する。

| 試行点 | 必須の観測結果 | 成立させないClaim |
| --- | --- | --- |
| R1最初の実Go CLI公開 | Independent Python Readerは保存済みR1 Manifest原バイトがmember Read前後に一致し、固定Ledger Path→Member→Byteを照合 | 完全Artifact Envelopeのdiskからの再構成・Producer/Readerの独立Required closure・Sourceのsnapshot transaction |
| R1 Source名relocate→R2 `-out`同一Directory | R1/R2のInputRevision・Manifest Digestは異なる。R2書込後はR1 Pinでの読取を**Manifest変更として拒否**。R1/R2の本文byteが一部同一でも旧版資格は昇格しない | 原子切替・同時刻に並走するReaderの一式受領・旧世代retain・GC・rollback |
| R2公開後の再取得 | R2 `manifest.json`の実バイトとstdout Manifestを比較し、R2 Pinで現在のMember全件が実Read可能かを確認。実測した`unchanged_body_paths`はBody byteのみの観測 | R2のOwner Acceptance、MFT Recipient、実ImportUnit、Period `closed`、Current Grant |
| Reader途中にManifestがR1→R2 | `verify_pinned_ledger_publication_paths`はmember Read前後のManifest byteを照合し、中途変更を**独立負例として拒否** | 2点照合ではR1→R2→R1の**ABA**・Member単位の間欠上書きやFSキャッシュ競合を検出し尽くせない |

**使う部品**: `plan/fixtures/capability-handoff/publication_path_readback.py`へ「固定Manifest byteの前後一致」を追加、同`test_publication_path_readback.py`へ不変Pin陽性・R1旧Pin拒否（Body同一）・Reader途中でのManifest変更・Pin不正値を追加（既存11件→15件の見込み）、`verify_real_cli_revision.py`の既存real CLI呼出しをR2同一Directory再公開へ接続。**新CLI、第二Reader正本、Manifest Schema、Publisher/Rollback機構は導入しない**。成功した場合も`ledger`固定Fixtureの**sequential two-revision actual-publisher/independent-process read**という限定表示のみ。

**他チャットへの整合**: [Draft #3321](https://github.com/hat47x/tei-platform/pull/3321)の同じObserved Snapshotでも物理Root/Read Attemptは同一とは限らない反例を踏まえ、同じManifest byteから独立Source Rootの同一性を主張しない。[merged #3731](https://github.com/hat47x/tei-platform/pull/3731)のPlan二回Build再現は生成側の局所Determinismであり、公開後の現在Reader取得資格ではない。#3252のClaim単位の変更影響と#2604の実Publication Ownerをそのまま維持する。

**P1へ残す最小未証明**: (1)R1をpinしたConsumerがWriterと本当に並走する中で全部のMemberをR1として取得できるか、(2)固定Generationの実バイトを保持する期間とGC後の失効、(3)物理Root/Manifest ABA/同一Directory上書き・Crash/OS別FSの整合、(4)#2542の宛先別配信・ImportUnitと#1130 Period/Owner Effectの別資格。2回の`manifest.json` byte照合をCAS・Directory snapshot・実業務成功に昇格しない。専用CIによる最新HEAD検証前は全追加ケースの実行成立を主張しない。

### 12.7.1 2026-11-09 exact-HEAD実測：5 Body同一・Manifest世代相違

**実行済み**: [Draft #3256](https://github.com/hat47x/tei-platform/pull/3256)のHEAD `06293192394ed908ce13a06474fc5f3d94495cff`で[one-revision actual Go CLI + independent Python reader #42621235600](https://github.com/hat47x/tei-platform/actions/runs/42621235600)は**completed/success**。Log Job `real-cli-one-revision`で**Python 15/15 PASS（0.017s）／実Writerの正規Manifest R1/R2 Go 3件 PASS／実Go Build PASS／実CLI R1/R2上書き公開後の独立Python Reader PASS**を個別に観測。ここでの「独立」はProducerとは別Process・別言語のReadを指し、Publisherと並走するReaderではない。

| 実行ログの比較項目 | R1 | R2 |
| --- | --- | --- |
| Source tree SHA256 | `af320664fbd4a84a903172373f39b78723c43188cc68d7aea1f5177e0abc7279` | `e6eeb6b75aa18cea69ea7a051c910547765985e583779e9b398b2b61855b5716` |
| InputRevision | `sha256:c60ed68c7c85ab218883b18970df85db1ebcae7d9090336452766d7a95d2b04d` | `sha256:b596f00201f39ac542b305dd284a9cc4d3daf59d31f41340c91c390a94f89240` |
| Manifest Digest | `sha256:e8ff4b37acbbaa9cbf343de7ef5614eec120e12f799ff1ad9f8b852f9c947a13` | `sha256:3632c2c8e81ef0423588e408b9acdc25b39e5a108c56e2e3fdd07400bd3d0e3e` |
| WorldFingerprint | **共通** `sha256:bd53dbaca0ac2c20bc911e0b69083467f2ad38273911269d525db666e681817f` | **同左** |
| 公開Body bytes | **5/5同一**：`function-ledger.md`、`operation-surface.json`、`table-public-notes.md`、`table-public-order_archive.md`、`table-public-orders.md` | **5/5同一**（同じ相対Pathのbody SHA256一致） |
| 実Read資格 | R1公開直後のPin Read=PASS | R2再公開後の**旧R1 Pin=REJECTED、新R2 Pin Read=PASS（5 Member）** |

**この反例が示す新しい設計上の分離**: `Body byte identity` と `Artifact Envelope identity` と `Manifest generation membership` が互いに異なる。**同一Bodyの再利用は可能な設計候補**だが、仮に物理Blobをcontent-addressedで共有する場合でも、`(Generation, Consumer Required Member Role, Actual Path/Object, Byte Digest, Source Revision, Disclosure)`の対応と現在性を別に確保しなければR1をR2へ誤昇格する。これは**既存#2604のAP-08〜16と#2630の用途相対Member閉包への受入条件**であり、新共通Manifest型／第二Artifact Registryの採用を意味しない。実Body byteを共有できるかはStorage Provider・GC/Retention・開示範囲が成立する場合に限る。

**残る反証**: ここではSource relocate後に別々の`tei artifacts -out`が**順次**実行された。2点Manifest一致検査はABA/R1→R2→R1・途中のMember上書き・複数FS/OS・R1 pinの継続読取保証を証明しない。実Sourceの物理Root/同一Read Attempt、Recipient ACK、ImportUnit、Business Period`closed`、Effect/Ownerの資格も別。正式なPublication方式の選定は**#2604 Owner**に委ねる。

### 12.8 2026-11-11 Pin前後一致が隠すABAと物理Member参照のTOCTOU

**2026-11-11の受入候補（test-only、実行結果はexact HEAD CI待ち）**：2026-11-10にmain統合済み[PR #3256](https://github.com/hat47x/tei-platform/pull/3256)の`plan/fixtures/capability-handoff/publication_path_readback.py`は、R1の`manifest.json`原バイトを**Member読取前後の二点**で照合し、固定`ledger`の5 Path→Body byteを確認する。これは**同じGenerationが読取中ずっと活動していたこと、同じ物理File objectを読んだことを証明しない**。

| 既存Readerへの追加テスト（合格しても欠陥修正ではない） | 正本・注入／Independent Expected | 解釈の上限 |
| --- | --- | --- |
| **ABA-C1** Manifest R1→R2→R1 | 合成Ledger DirectoryでR1 ManifestをPinした後、最初のMember `Path.read_bytes`のタイミングでR2 Manifestへ一時変更し、R1原バイトへ戻す。Readerの前後二点はR1一致となり`ledger-fixture-path-to-bytes-only`を返せる。**Manifestが期間中常時R1だったというClaimには反例** | テストはPython読取を一箇所interceptする**決定的合成インターリーブ**。実Writerとの並行FS競合、Windows/ネットワークFS、ReaderのABA防止実装はNOT RUN |
| **TOCTOU-C2** Path点検後にMember参照を変更 | 既存ReaderがDirectory Entryを列挙・非symlink判定した後、先に読む`function-ledger.md`を契機に未読の`table-public-orders.md`を削除して**外側の公開合成ファイルへsymlinkで置換**。外側のBodyを期待bytesと同一にすると、Readerは引き続きPath単位byte一致を返し得る。**列挙時の物理Path所属≠実OpenしたObjectの所属** | CIがsymlink作成を許さない環境ではSKIP。実機密データ・実攻撃・実任意Reader/Hostへの到達試験ではなく、テスト用Directoryと合成Contentに限定 |

**共通の教訓**: R1 Manifest二点照合は必要な改訂検出の局所検査だが、読取中の閉包、公開世代の継続安定性、物理Root/Node identity、現在Disclosureには不足する。[merged #3321](https://github.com/hat47x/tei-platform/pull/3321)の「同一Snapshot tokenでも物理Root/Read Attemptの同一性は不明」、[Draft #4213](https://github.com/hat47x/tei-platform/pull/4213)の「検証済みSource byteをそのままConsumerへ渡さなければ宣言PathのABAで別byteが実行され得る」、[merged #3809](https://github.com/hat47x/tei-platform/pull/3809)の「test-onlyの生wire同一性とcurrent Issuer Trustは別」という境界と整合する。ただしTarget Build用のSource拘束策を、そのままArtifact公開/実MFTの規格に昇格しない。

**#2604 Ownerへ戻すP1の最小選択肢**: (A)世代別不変MemberとGeneration Pointerを使い、R1 Pin中はR1を保持・Consumerが必要Setを固定して実Open/byte照合する、(B)既存Storage/FSが提供するversioned Object/conditional readを利用し、個々のMemberの実version・Scope・Digestsを別に拘束する。いずれもProducer側の書込完了／読取側の実Handle・Object version・Path非追従と、GC・rollback条件をOS/Providerごとに実証してから採否を決める。**既存`artifact.ManifestEntry`へ新しい固定Path欄、Core普遍Publisher、第二Manifest/Authorityを無条件追加しない**。test-only Readerを本番で安全と認証した扱いにもしない。

**段階ごとの停止線**: ①今回の合成ABA/Pathスワップ負例の実CI結果、②別Process実Publisherと並走ReaderでのGeneration/Member読み取り（#2604）、③実RecipientのRead/ACK/ImportUnitとBusiness Period D（#2542 EF-01〜04）、④Owner/Effect（#1174/#1130）。各段階の成功は後段の成功と等価でない。Claimを`body-bytes-observed`、`generation-pinned`、`continuous-generation-read`、`business-accepted`など**意味を区別して記述**するが、これらを新しいCore Enum/APIとして先行採用しない。画面でも「証明済みでない」と「不存在」を混同しない。

### 追加一次資料（製品・仕様の版に依存する事実）

- Apache Iceberg spec, snapshots / optimistic concurrency: https://iceberg.apache.org/spec/
- Apache Iceberg reliability: https://iceberg.apache.org/docs/latest/reliability/
- Apache Kafka design, message delivery / external systems: https://kafka.apache.org/design/
- S3 multipart checksum and ETag limitations: https://docs.aws.amazon.com/AmazonS3/latest/userguide/mpuoverview.html

**境界**: 本追補はGitHub Source/Issue/PRと外部公開仕様の静的比較のみ。新しいGo/Node/実PostgreSQL/実SFTP/S3/HULFT/EKI/独立Consumer/本番認証/Owner受理の試験・CIを**実行していない**。既存PRの歴史的実試験はそのexact-head/Provider/Scopeの資格だけを継承し、最新mainと本Draftの同一Attempt E2EはNOT RUN。


## 13. 2026-11-05 セルフビルド・Plugin Bundle設計との接合：配送Adapterの選択とBuild非干渉

[#1251](https://github.com/hat47x/tei-platform/issues/1251)では「配布・発見できる」「Compositionで選択」「実Buildが入力として消費」「成果物へ収録」「実Targetが起動・使用」を**別の資格**として扱う。[Draft #2877](https://github.com/hat47x/tei-platform/pull/2877)／[#2831](https://github.com/hat47x/tei-platform/issues/2831)のセルフビルドP0は、宣言したGraphと実工程、全module（nested Go module含む）のSource/Toolchain closure、実Artifact、Stage収束を照合する計画であり、TEI全体のG4資格が成立済みという意味ではない。ETL/MFTにもこれを重複実装せず横断利用する。

| 比較する二Target | 成立させたい観測と停止線 |
| --- | --- |
| 同一Data Transform、MFT Adapter **未選択** | MFT Bundle追加・改訂だけなら、選択済みBatch/SQLの実Build closure/生成Source/Artifact members/linked binaryに**影響しない**ことを観測。Registry/Catalog候補一覧やRepo HEADの変化は許容。Go init/blank import/global registrationで混入すれば未使用非干渉は不成立 |
| 同一Data Transform、SFTP等の配送Adapter **選択** | 対象Adapterの実Package/Transitive module、Generator、Target capabilityと契約版を実Buildへ束縛。選択Adapterを更新したら必要Output/Build資格を再評価し、業務Meaningや他TargetのArtifactを無条件に無効化しない |
| 選択したAdapterにnested moduleの入力が欠ける | ルート`go test ./...`のPASSから対象ClosureのCompleteを主張しない。nested go.mod/go.sum、実Compiler/Worker実消費入力まで列挙し、未知ならBuild資格を保留 |
| Buildは成功、実配送先では現在Grant/通信経路/保存Policy未確認 | Artifact-included≠Runtime-invoked≠配送・業務受理。#2542と#951/#1174の現時点資格へ戻し、暗黙Fallback・実行試行済み扱いを禁止 |

**既存Pressureへの割付のみ**：DM-12の「同じFlowを複数実行先へ」のうち、(a)未選択MFT Plugin追加、(b)選択SFTP Adapterの依存変更、(c)nested moduleの欠落、(d)実Binaryは同じでRuntime Grantだけ失効、を比較列として使う。新しい独立Core型・Build Graph・SB/PB Fixtureの第二正本や「一Plugin＝一Binary」を導入しない。

**実証状態**：本PRではこの差分のGo/TS/CLI実Build、nested module全件走査、MFT Adapter選択、実SFTP/GitHub CI、配布後Runtime資格は**NOT RUN**。#1251/#2877の採否・同日CIと、実Targetでの適合証拠を別途要する。

## 14. 2026-11-06 Batchの四つの粒度と、配送保証の最小充足判定

**状態**: 既存[#2542 EF-01〜05](https://github.com/hat47x/tei-platform/issues/2542)と[#1130 RPL](https://github.com/hat47x/tei-platform/issues/1130)の受入票に接続する研究上の精緻化。新Core型・新Scenario・Runner・MFT Engine・正本追加ではない。起点mainは2026-11-06取得の bf2e7b0ffe21747a308cd2b37328eb440535f696。mainは更新され得るためmerge時には再確認する。

### 14.1 「Batch」を一つの意味にしない

| 次元（仮称） | 典型的な単位 | 独立に扱う境界 | 暗黙に昇格してはいけない保証 |
| --- | --- | --- | --- |
| **Transport batch** | network chunk、ファイル、AS2 message、転送箱、spool item | transfer attempt、hop ACK、再送、暗号化/圧縮後byte、同報宛先 | packet/file送達＝業務単位atomic commit；chunk再開＝Import冪等 |
| **Data processing batch** | 行集合、partition、window、Watermark、Snapshot／処理基準日 | schema/NULL/Bag/Sequence、Source completeness、処理粒度、導出Output | 変換結果の一致＝受信側の完全性・即時公開 |
| **Business atomic batch** | 同一Ownerが受理するOperation/Unit of Work、業務期間・明細集合 | effect/Commit、partial/unknown、Idempotency Key、独立Receipt | 一つのファイル/Job/SQL Batch＝不可分な業務トランザクション |
| **Execution grouping** | RunPlan、Task/Worker、並列数、Lease、Build Unit | 資源配置・取消し・進捗・Checkpoint、起動/再起動主体 | Task completed／step blocked＝業務Effect有無の証明 |

この四次元は共通の型付きOperation・Port・Relationに載せ得るが、各次元のOwnerと実証する保証は異なる。特にファイル転送の処理単位の結合・分割を、Business Operationの結合・分割へ自動反映しない。各Adapterの最小/最大Chunkサイズのような性能設定を受理済み業務Semanticの変更とみなすかは、実際に意味結果や受入条件を変える場合だけに限定する。

### 14.2 条件付き保証の充足判定（実行前の適合と実行後の事実を別判定）

**要求Claimの比較キー候補**: 対象（qualified subject）、業務期/Source revision、対象Dataset/Artifact generation、宛先・ordering key/Scope、保証の種類（順序/欠落なし/一式可視性/冪等Effect等）、適用時間/鮮度、必要なAuthority。

**Provider側Witness候補**: 対象Operation、実Provider・能力版・Target、保証できる対象Key/Scope、条件と例外、選択済みRouteの全Hop、基準Artifact/Rule版、実行時のAttempt/Receiptまたは未観測理由。**Providerの仕様宣言は実際に送信・読取・Commitした観測とは別証拠**とする。

充足判定の説明規則（製品APIではない）:
1. **意味の対応**: 要求するSubject/Period/Dataset/Operationと選択したProvider/PortのMappingが一意で、転送・変換中にIDと保証対象が失われていないこと。曖昧/欠損ならunknownまたはunsupportedとして止める。
2. **ルートの被覆**: 必要な保証を全Hopが同じ粒度で提供し、Queue/Spool/変換/中継による弱化と例外が明示されること。単一Hopが順序ありでも、全路線順序ありとは主張しない。
3. **全件性・物理公開の分離**: 期待するMember/Range/Watermarkと、Publisher/Consumerが実読取した全Byte、対象世代の保持期間、実取込みの結果を別比較。空集合/timeout/通知欠落を「存在しない」と断定しない。
4. **実行前と事後**: Static conformanceが満たせても、実配送・実Reader・Commit・現在Grantは未観測のまま。到着後に検査できるReceiptは元のRequest/Generation/Scopeと結合し、現在資格へ自動昇格しない。
5. **結果の分類**: Requirementsを満たす証拠がある対象限定Claimだけqualified、満たせないことが分かるときunsupported、資料/受信/時点不足ならunverified/unknown、もともと必要でない要素だけnot-applicable。単なる成功フラグのANDとしない。事後Unknown Effectを自動Retryしない。

### 14.3 Backpressure・取消し・部分失敗のOwnerと反証

| 反例（既存ケースへの追加観測） | 最低限の判定／停止線 | 再利用するOwner |
| --- | --- | --- |
| **Transport Batchだけ四分割**：100行のCSVを4 chunkで配送、3 chunk送信済みの時点で切断 | 転送進捗をImport complete/Business atomicとみなさず、受信側完成版公開前に読まない。再開点はByte/TransportとBusiness両方のIDを参照 | #2542 EF-01/03、#2604 AP-01/06 |
| **Spoolが満杯**：先行ファイル保留、後続が別Hopへ届く、期限超過 | Capacity/Retention/Deadline/BackpressureはHopごとの能力。drop/skip/非通知送出からgap-freeを導かず、必要な全件保証が満たせなければ該当Claimをreject/unknown | #2542 EF-02、#2827 DM-05/06 |
| **取り消し要求の競合**：配送指示取消し時に受信側は既にACK済み、Importerは未実行 | Cancel requested、実際の送信中止、受信側の永続Receipt、業務Effectの取消しを別結果にする。取消し成功から相手先の未取込・既存Commit撤回を推論しない | #2542 EF-03、#1174 |
| **部分成功の一括更新**：100行のうち60行Commit後に残り40行がblocked | Run全体をno effectへ丸めない。既存Effect/Receiptで対象Keyごとに60/40を照合し、全体の業務原子性が必須ならProvider能力不足として事前に拒否する | #1130 RPL-02、#1174 |
| **必要Guard未供給**：Period/Effect Observerの実引数は一部のみ、別HostのGrantは失効 | [merged #2882](https://github.com/hat47x/tei-platform/pull/2882)は実Workspace局所的な前提合成テストでありGuard completenessやEffect実観測は未証明。未供給/古い/別Scopeはqualified不可 | #1130 RPL-01、#1174、#1145 |
| **実行単位だけ変更**：純粋なTransformを1 Process→4 Worker、外部SFTP Adapterは未選択 | Data Meaning/Independent Expectedを固定し、選択Plugin依存Closure/結果の非干渉と実行回数・Costは別測定。業務副作用Jobに同じ最適化を無断適用しない | #1251、#2831、Draft #2877、#2018 |

**優先順位（元のDM/EF/RunPlanテストを追加しない）**: P0はread-onlyな1入力・2 Memberの生成→公開→別Readerの同一版/全Byte比較（#2604/EF-01）。並行して、必要Guard集合と実際にSupplyされた集合の差分だけを評価する静的負例（#2882の受入拡張先）。P1はFileSpec/ImportUnit（#2145既存Reference）と重複通知/取消し/Spool容量を合成。P2で初めて現在Grant・Commit/Receipt/独立Readを要する書込みRouteへ進む。どの段階もProviderの実送出、外部Host/EKI ResourceやBusiness Owner受理を他のCIから継承しない。

**標準化の採否**: 新しいBatchKind/Guarantee証明型/Capability Resolverを固定する前に、既存TypeRef/Shape/Role/Relation/Provenanceで、上記少なくとも二Consumerの必要な差を保持できるか検証。特に[#2880](https://github.com/hat47x/tei-platform/pull/2880)の標準型二Consumer、[#1145](https://github.com/hat47x/tei-platform/issues/1145)のInvocation Request/Context/Transport分離を再利用する。Operator/Plugin合成のほうが例外と変更波及を少なく説明できるなら新しい専用DSLやCore Primitiveは導入しない。実Go/ETL/MFT/Transfer/2Process/DB/EKI試験とOwnerの意味受理は本追補では **NOT RUN**。

## 15. 2026-11-06 到着・期間・公開の有界な完全性（設計／未実証）

**問題**: MFTの送達完了、ストリーム処理のWatermark通過、月次締めRunPlanの完了、Artifact ManifestのCompleteが、いずれも異なる対象範囲のClaimであること。これらを同一の「データセット完全性」に畳み込むと、遅延データ・中継欠落・フィルタ済みQuery・異世代Manifestの混在を見逃す。本節は既存#2542 EF-01〜05・#2604 AP・#1130 RPL・#2933 INVと関連付ける比較票であり、第二のScenario/Completion Engine/Watermark ABIではない。

### 15.1 完全性を一つのboolでなく対象付きClaimとして扱う

| 判定対象 | 必要なWitnessと判定可能な主張 | 「完全」へ自動昇格してはいけないもの |
| --- | --- | --- |
| **Source/input selection** | 受理済みSource Revision、Manifest sourcesの全解決、業務期/入力Snapshot/期待する集合の範囲 | Workspace validate green、Source 0件、Query結果0件だけで「元データなし」（#2823） |
| **Delivery coverage** | 宛先・File ID/Period・期待Artifact/Sequence一覧、各Hop Receipt/期限/欠落の独立検証 | 最後のFile ACK、追越禁止、Watermark通過だけでgap-free／受領全件 |
| **Dataset/window coverage** | 対象のevent-time窓と営業期、入力Sourceごとの水位、allowed lateness、early/on-time/late pane、訂正履歴 | Watermarkは観測/推定に依存する到着指標であり業務上の全件保証・不可逆締切ではない |
| **Published artifact-set** | 一つのGeneration/Member集合、実Byte/Hash、Consumer版pin/保持期限、Readerの全Member再読取 | 単一ファイルのatomic rename／Manifest completeから全Memberの同世代受領 |
| **Business period acceptance** | 業務Ownerの受理規則・全件/欠落許容Policy・Rule/Source版、現Grant、必要なEffect/Receipt | RunPlan completed、0件・一致行数、MFT success、早期計算結果からBusiness Accepted |
| **Graph/plan coverage** | 対象Scope/Target/Selected Plugin/Source Closure、Declared/Resolved/Observed/Consumed/Acceptedとopaqueな外部工程の被覆 | 図に表示されない、Filterで隠れた、未許可情報がある、という理由で不存在を断定 |

この表は**比較用の断面**でありTEI正式State・ABIではない。限定したQueryで「全件」と主張するには、対象集合を列挙できる根拠、または不完全な場合を停止できる業務規則が必要。無限Stream/取引先が全ファイル一覧を公開しない場合には、勝手に「すべて届いた」としない。必要なSLA・期限・最終確認を明示のProvider/Owner Contractへ戻し、証明できない期間はunknownとする。

### 15.2 Late dataを再配送・再計算・業務訂正に分ける

**参考モデル**: Apache BeamのWindow/Watermark/Triggerではevent timeとprocessing timeを分け、早期/定時/遅延のpaneを発行し、遅延許容や蓄積/破棄の方式で集計の出力が変化する（https://beam.apache.org/documentation/programming-guide/）。これはTEIによるストリームEngine採用ではない。HULFT-HUB蓄積とODIPの処理基準日にも、異なる時点の区別を利用できる。

**例**: 営業日Dの受注ファイルF1/F2/F3を期待する。F1とF3は受領済み、F2は中継で蓄積失効した。D+1の時刻到達でWatermarkやJob deadlineが進んでも、期待集合の一部F2が欠落するので「D完全」のEvidenceはない。遅延したF2がD+2に来た場合は、(a)配送の補完、(b)Dの集計再生成、(c)すでに締めた業務期の訂正/再受理、を別Owner・別Attemptとする。Dの既確定Artifactを同名で黙って上書きしたり、旧Receiptを新しいCommitへ転用しない。

| ケース（既存Pressureへ重ねる独立Expected） | 判定すべき停止線 | 引渡し |
| --- | --- | --- |
| **Early result**: まだ全Fileが届かず暫定100行を集計 | provisional/partialの出力と最終結果を区別し、後続到着が修正し得ることをConsumerへ示す | #2622 Dataset/Derived、#1130出力、#2542 EF-02 |
| **Watermark passed, file gap**: 先行F2欠落のまま定刻となる | watermark/時刻はgap-free証拠にならない。必要な全件性が要求されればBusiness Completeを拒否/保留 | #2542 EF-02、#2827 DM-05/06 |
| **Late F2 after period close**: D+2にF2受領、Dの業務期は確定済み | 計算結果の追補・Artifact新版は可能でも、業務再確定は既存Period/Ownerの訂正・権限・Effect規則に従う。自動re-accept禁止 | #1130 RPL-02/03、#1174、#2144 |
| **Optimization changed pane**: 窓集計をSQL PushdownまたはChunk/Fusionに変更 | 独立Expectedについてevent-time window/allowed lateness、重複・NULL・順序、Paneの暫定/確定、Consumer visibility/Scopeが保たれる範囲のみEquivalent。パフォーマンス改善は別Gate | Draft #2933 INV-04/06/10、#2622 |
| **Opaque downstream MFT**: TEIの図では一つのTransfer nodeに見えるが、外部中継がさらに配送・破棄 | その先の内部を「存在しない」と主張しない。Node境界をopaqueとし、取得できる実Recipient Receiptまでを観測範囲に限定 | #1124/Topology、#2542、研究RQ7 |

### 15.3 小さなConformance縦切りと採否

**最短実証**はすでに起票された[#2542 EF-01/02](https://github.com/hat47x/tei-platform/issues/2542)を使い、(1)明細/件数合計の二Member Artifact、(2)F2欠落とF2遅延到着、(3)RunPlanの処理基準日、(4)同一世代/別版Reader、(5)完了/暫定/unknownの独立Expectedだけを追加する。新しいFixture/Oracleの正本は作らず、実OwnerのScenarioへ観測列を渡す。

最初に必要なのは**読み取り専用**の受理前判定。業務再確定や外部MFT送出を先行させない。外部Sinkの二重副作用を扱う場合に限り#1174/#2853の同一OperationID/Receipt照会へ接続し、新OperationIDの自動再適用を避ける。

測定指標は、(i)既存テストで見逃していた欠落/遅延/世代混合のFalse Qualified数、(ii)新しい意味正本・Role・Provider専用コードの増分、(iii)人間/AIが必要な欠落情報と影響範囲を特定する手数、(iv)Streaming/Batch化前後の可視性・Latency/Memory/Cost（#2933）を区別する。**汎用Completion Engineや新Watermark型は、二つの独立Consumerに共通して必要な表現不足が実証されるまで導入しない**。

**状態／証拠上限**: BeamとHULFT等の仕様の比較、および現行TEI Issue/PRの静的照合のみ。本節の新しいGo/Streaming/SQL/実File Reader/実MFT/CI/E2E・業務Owner受理はNOT RUN。外部製品の仕様上のWatermarkや順序制御を、そのままTEI Providerの実保証として採用しない。

### 15.4 HULFT-HUB同報でのACKの主語と再配送範囲

HULFT-HUB Ver.3公式マニュアル第20版（2025-12-01）は、蓄積を使う同報配信で、**HUBへの蓄積正常終了により配信元が転送完了を受ける**一方、HUB→個々の集信先の転送がエラーになり得ると説明する。また、蓄積しない同報の再配信は原則すべての宛先へファイル先頭から実施し、異常宛先のみ/Checkpointからの再配信はできない。蓄積を併用する場合は異常宛先だけを送出可能な条件がある（https://www.hulft.com/help/ja-jp/HUB-V3/COM-HUB/Content/HUB_HUB/Pt_HUBOpe/Multicast_Pt.htm）。蓄積条件・継続条件・配信側へ返す結果の設定によっても、蓄積完了・集信先への送出異常・配信側への転送結果は分岐する（https://www.hulft.com/help/ja-jp/HUB-V3/COM-HUB/Content/HUB_HUB/Fun_HUB/Ope_SET_Acc_Proc.htm）。

- **ACK subject**: 配信元→HUB蓄積完了、HUB→各宛先送出完了、受信先のファイル可視、Importer確認、業務Owner受理を区別する。HUB ACKを「すべてのRecipientに届いた」という集合Claimに昇格しない。
- **Fan-out Recipient Set**: 宛先集合とその版、各宛先Attempt/成功・失敗・保留、再送時に実際に再配送した宛先集合を別に保持。同報の全宛先Retryが正常受信先へ重複送信しても、ImportUnit/Business Effectの冪等性は相手側で別に検証する。
- **Recovery policy**: 全宛先の再送、異常宛先のみの送出、蓄積解放後の送出が利用可能かはProvider/運用設定依存。TEI共通型の固定操作を先行させず、選択Route/各実環境の能力とRequest/Receiptを照合する。

**既存受入への配分**: #2542 EF-02（順序と欠落）・EF-03（ACK喪失と重複）、#2827 DM-06/07/08（Spool/再送/受理）、#1130 RPL-02（部分成功）へ宛先ごとのObservation列を追加する提案。新Fanout Engine/Retry Coordinator/第二Recipient Registryは作らない。実HULFT-HUB同報/実Partner試験とOwner受理はNOT RUN。

## 16. 2026-11-11 期間close宣言・Caller報告・実Effect Receiptの3境界

**現行mainのSource照合**（固定main `b8cbd432aae22ccf3fa75b0119e1d90d87ef0d11`）：`internal/operations/period_requirement.go` blob `578ca2498c17b8b1764437a61d36e1c2a67bd53f` の `CheckCloseRequirement` は、`PeriodRequirement.Capability` が空でなければ **callerから与えられた `ranCapability` と宣言IDの一致**を確認する。 `cmd/tei/cli_operations_artifacts.go` blob `91598f7f05919dd60ca42e49cd71ab8f9c3241b5` の `runPeriodsCommand` は、`-close-capability` をこの比較に渡し、許可された場合 `periodstate.Writer.Write` を呼ぶ。`operations.PeriodRequirementsFromWorkspace` は実宣言を読む。**これらは実Capability実行・Receiptの取得・Output/Effectの独立観測をこの経路では行わない**。既存`period_requirement_test.go` blob `ac95893136393e7e5d299bb24b68d272092dfb83` は一致IDの許可、不一致・省略IDの拒否、宣言無しの場合を検証する。以下はこの実装を否定するのではなく、Claimの射程を限定する。

| 受入対象／対応Owner | Required（意図・必要条件） | Supplied（Caller報告） | Observed（独立の実証） | 現資格 |
| --- | --- | --- | --- | --- |
| **意味宣言** #2144 / `close-requires` | Dの宣言済Close Capability、期とSource revision | 宣言のLoader読取 | Workspaceから`PeriodRequirements`取得 | **宣言読取とID照合は実装済み** |
| **閉鎖リクエスト** #1130 RPL-01/02/04 | 現Period D・期待State/Revision・必要Capability | `-advance D -close-capability X`、`X == declared ID` | `CheckCloseRequirement`の一致/不一致、`periodstate.Writer`の単一ファイル置換 | **一致IDはCallerの主張のみ**。実行済Capability/現在Grant/多Writer CAS/Owner受理は未実証 |
| **Close処理実行** #1130 / #1174 | 同期したScenario/Operation/Attempt・Required Guards/Effect | `X`を実行したというCaller表明 | 同じ対象D/OperationID/Source版への実Runner journal、実Output、独立Effect/Receipt readback | **本CLI経路では未取得**。同名Xが渡っただけで処理完了にしない |
| **Artifact受理** #2604 / #2542 EF-01 | Dに必要なF（明細）＋T（件数・合計）、期待GenerationとRecipient Set | Manifest/公開通知、送信元/HUB ACK | Pinした実Manifest・F/T全Byte、世代一致、各宛先の実Read/ImportUnit/Receipt | **期間Stateとは独立**。F/Tの片方だけ・旧Member・HUB蓄積ACKは全件受理でない |
| **業務期間の受理** #2542 EF-04 / #1174 / Owner | 業務期Dのcutoff・遅延/訂正方針・期待受入結果 | `closed`保存値／RunPlan完了報告 | 独立Business Effect・帳票/実DB照合・Owner受入・再Open/更正の別Operation | **未実証**。`closed`≠すべてのData・Recipient・Effectが完了 |

### 16.1 最小反証と期待結果（既存Issueの受入ケースを拡張、未実行）

| 入力変形 | 既存経路で確認済み／静的に期待できる動作 | 残るRequired→Observed不一致 |
| --- | --- | --- |
| `close-requires X`、Callerが`X`を省略／別ID `Y` | `CheckCloseRequirement`は拒否（既存Go Unit） | 実Provider/Ownerの全実行ではなく局所Guard |
| `close-requires X`、CallerがID `X`だけ渡すが実行Receiptを持たない | **ID比較は通る設計**（現Source-static）。これは安全性の積極的合格ではない | **実Xの実行・Output・Effect・Current Authorityは未証明**。将来の資格付与では`unknown/unqualified`とする |
| 同じID `X`でもR1とR2でSource/Period/Operationが変化 | 名前一致だけでは別Attempt/版の同一性を検査できない | Source/Operation/Period revision/Receiptを一致させ、旧版の受入を拒否する必要 |
| `closed`になった後にF2遅延到着、F/T内の一方がR2・他方がR1 | State Fileが`closed`でも独立File Reader/Business Scoreを行った証拠はない | `closed`から自動再Accept・自動再Effectに進まず、#2542のImport/訂正方針とOwner判断へ |
| 同じDを二Callerが同じ旧Stateから更新 | `Writer.Write`のtemp+renameは個別File更新、read/modify/write CASではない（§12.6） | 複数CallerのExpected State/Revision/現在Grantと実Commit時点の整合が未実証 |

**判定式の表現（新Core Boolean/DSLではなく、既存受入表の列）**：
`DeclaredRequired ⊆ Supplied` は前提の網羅確認であり、**`Supplied` ≠ `Observed`**。`Observed`を付けるには`(D, Source Revision, Operation/Attempt, Effect, Required F/T, Recipient)`の各主語に対応する**実のReader/Receipt**が必要。未申告条件（欠落）、申告したが未観測、観測失敗、対象外を区別する。情報開示不可のScopeで件数や理由を推定して補完しない。必要Collection/Principal/Authorityが選択されない小規模Batchへ共通Hostを強制しない。

### 16.2 優先順・責務分担と実証停止線

1. **P0／#1130**：既存`CheckCloseRequirement`/Period Workspaceの実入口で、`X`名一致のみと実Runner/Effect Witnessを分けた**独立Expected**を記録。要求CapabilityがCallerに渡されなければ拒否される現Unitを再実装せず、先に「一致するが実行証拠なし」を不合格資格として表に足す。Owner受理無しでCLIを変更しない。
2. **P1／#2604・#2542**：同じDで正規F/T二MemberをR1からR2へ切替え、Published Body/Manifest世代、Recipient/ImportUnit、Business Cutoffを別Attempt/証拠で照合。既存[#3256](https://github.com/hat47x/tei-platform/pull/3256)・[#4240](https://github.com/hat47x/tei-platform/pull/4240)はmain統合済みのReader Characterizationに限り、Atomic Publication・二実Reader・実MFTの証明に使わない。実HULFT/Partner送達はNOT RUN。
3. **P1／#1174・#2144（意味のA4判断が必要な部分）**：実Tx/Effect Receipt、`D→Correction Operation`、Owner再受理とCAS/Authorityを段階接続。#2144の宣言モデルP1〜P3はmain統合・Issue close済みだが、実業務Ownerの受入完了と同一視しない。

**重複回避**：#1130がRunner/必要Guardの充足、#2604がArtifactSetの実公開／Member、#2542がFile/MFT／ImportUnit、#1174がEffect/Receipt、#2144が期間意味の受入を所有。原則Hostなし／認証は必要環境のPlugin能力／TEI基本構造の合成を維持し、別のPeriod Runtime、ETL Completion Engine、追加Manifest、Scheduler、万能証拠Registryを導入しない。

**実行状態と証拠の限界**：本節はmainの4つのGo Source/既存UnitとIssue/PRの**静的照合・設計追補**。追加の実CLI/Goテスト、実PG・MFT・二Worker競合、同一AttemptのReceipt連結、実業務Owner受理は**NOT RUN**。[#4240の専用CLI #42933670806](https://github.com/hat47x/tei-platform/actions/runs/42933670806)はsuccessだが、同HEADの[Repository Health #42933670781](https://github.com/hat47x/tei-platform/actions/runs/42933670781)はfailure（`TestParallelClaim`/ `TestToolsReadmeMentionsEveryEntry`、本節のコードと別）であり、全体成功へ読み替えない。新しい本PRのCI合否は別途確認する。

### 16.3 非0終了後の期間状態書込み：ErrorとEffect Unknownを分離

**追加のSource-static条件付き反例（実CLI試験はNOT RUN）**：`runPeriodsCommand`は `-advance` の有効性・`close-requires`のID一致を確認後、**`periodstate.Writer.Write`で期間状態を保存してから** `operations.CheckPeriods(surface, states)`を評価し、拒否されるOperationがあればJSONを出力した後、`fmt.Errorf("%d operation(s) are refused by the period state", ...)`でエラーを返す。したがって「対象Periodに拒否対象Operationが存在し、`-advance`で新Stateを保存後にCheckPeriodsが非空となる」条件では、**非0 exit / CLI Errorでも状態Fileの更新が既に成功している経路**がSource上にある。これは実行の失敗と永続状態が変わらなかったことの**同値でなさ**を示す。実対象Workspace/Go CLIでの到達・File/FS耐久性は未実測。

| 観測主語 | 条件付きの現行動作 | 次の受入判断 |
| --- | --- | --- |
| CLI return / JSON | `CheckPeriods`のRejectionでエラー終了し得る | **Nonzero exitは「変更なし」ではない**。Errorの理由と完了済Stageを別に記録 |
| Period State File | `Writer.Write`が先行し`closing`／`closed`を書込み得る | 次のRetry/Correction前に対象Dの**実Readback＋Revision/Owner**を確認。別Attemptで同じ`-advance`を自動適用しない |
| RunPlan / Artifact / Recipient | CLIはここで実Batch処理、Artifact Publish、ImportUnit/Effectを観測しない | 期間Stateの変更が確認できても、#2604/#2542/#1174の副作用を推定しない。未観測ならUnknown |
| 競合実行 | `Writer.Write`のRenameは単体Fileの置換であり、双方のExpected State CASではない | 一方が`closing`を保存してエラーを返しても、他方の`closed`／Correctionを無条件に上書きしない |

**最小の次試験（既存#1130で採否）**：実WorkspaceのPeriodに拒否対象Operationがある陽性Fixtureを再利用して`tei periods -advance`を一度だけ実行し、(1)終了コードとRejection JSON、(2)実State Fileの再読取、(3)二回目Retryの状態/Effectを独立に保存する。**同一実行のExit StatusとReadbackをセットで比較する**。新Runnerや複数Phase Transactionを作らず、実Stage列のどこで止めるべきかはOwnerが決める。本節はCodeの実行順の静的照合であり、Go/実CLIの新試験・実MFT/DB/Owner受理はNOT RUN。

### 16.4 2026-11-11 現実Workspaceの期間更新後拒否を試験する

**既存Owner #1130 RPL-02/04への局所実証**：`examples/xtea-lowcode`にすでに存在する実Workspace `order_entry`／Period `sales.month`／`closed-rejects update`を用い、Productionコード`runPeriodsCommand`をGo Unitからそのまま呼び出す。元の`examples/xtea-lowcode`にはPeriod宣言がある一方、CRUDの`update`を導出するPersistence mappingがない。このため、**元Workspaceの隔離コピーに`persist sales_core.order`を1件だけ一時追加**して、`operations.SurfaceWorkspace`が`closing`および`closed`で拒否するOperationを実際に導出できることを前提検証する。**元のReferenceソース・Productionコード・テスト用Host・別State Providerは変更・新設しない**。追加mapping自体は試験専用であり、既存Workspaceがそのままで拒否Operationを持つと主張しない。

| 独立Expected | 呼出し前 | 現行実装に対するテスト | 実際の意味 |
| --- | --- | --- | --- |
| `closing`への一段更新 | State File `sales.month=open` | `runPeriodsCommand`が拒否Operationを理由にerrorを返す一方、同一AttemptでState Fileを再Readすると`closing` | Handler Error≠State Effect Zero |
| `closed`への再Advance | 直前のState Fileが`closing` | 同様にerrorでも再Readは`closed` | Errorから次回Retryを自動再実行してはいけない |
| 3回目のAdvance | State Fileが`closed` | `advancePeriod`が遷移を拒否し、Fileの`closed`は保持 | 先のErrorを「未実行」と解釈して再試行することの不適切さ |

**対象ファイル**：`cmd/tei/cli_periods_postwrite_readback_test.go`。テストが確認するのは **既存Workspaceの隔離・最小Persistence投影→現行Go CLI Handler→一時State Fileの書込みとReadback**。外部CLIプロセスの終了コード、Crash/OS別fsync、2WriterのCAS、実Capability/RunPlan/Receipt、MFT/Import/業務Ownerの受理は別で**未実証**。`-close-capability`については既存UnitがCaller ID一致のみを検証済みであり、本テストが実行済CapabilityのReceiptを提供するわけではない。

**採否と停止線**：試験が成立しても修正実装とは見なさず、#1130 Ownerが出力/持続化/拒否のStageを再設計するか、`Effect Unknown/部分成功`として明示するかを決める。#2542への横断引継ぎは`(Period D, Execution Attempt, Return, State Readback, Artifact Generation, Recipient Import, Effect Receipt)`を独立観測列とするのみ。第二Runner/Scheduler・新Process Transaction・共通Completion Engineを追加しない。

**実行状態**：本節の作成時点では新Go Testのexact-HEAD CIを未確認。静的反例（§16.3）と、追加した実Source使用試験の結果を混同しない。結果が出れば、固定HEAD/CIを明記し、既存#1130へ重複のない追補を行う。
