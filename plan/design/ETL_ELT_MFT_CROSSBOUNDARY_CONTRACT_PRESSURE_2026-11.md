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
