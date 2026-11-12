# OKF K2 実Provider二Consumer受入計画（2026-11-12）

## 位置付けと更新根拠

本書は [TEI/SHO OKF Domain Lint・Document Unit計画](TEI_SHO_OKF_DOMAIN_LINT_DOCUMENT_UNIT_2026-11.md) の**実Provider二Consumer受入**に限定した追加計画。旧[PR #3329](https://github.com/hat47x/tei-platform/pull/3329)はマージ済みで元作業ブランチは削除されているため、履歴の旧HEADを更新せず、現mainから新規Draftとして起票する。旧PRの大型設計差分を現mainに存在すると推定しない。

2026-11-12再照合：TEI [#3479](https://github.com/hat47x/tei-platform/pull/3479)（Provider Source-only drift）とSHO [#27](https://github.com/hat47x/sho-composer/pull/27)（公開TEI原文のprivate SHO実Reader受領）、SHO [#29](https://github.com/hat47x/sho-composer/pull/29)（UTF-16選択→UTF-8 byte範囲）は**マージ済み**。これらは限定的な局所検証であり、K2同一Provider実Invoke、実Writer、Owner Meaning受理の証拠ではない。旧TEI HEAD `1cf0a48a` の健全性CI成功は確認できず、旧HEAD `0d035db6` の成功を新mainへ転用しない。

## 受入の不変条件

- **K2は同じSourceを二つのConsumerが読めたことではない**。一つのProvider契約で選択された実体をTEI CLIとSHO実Reader境界の二つの独立Call Siteが**実際にLoad/Invoke**した証拠が必要。
- `selected_provider_id`、`loaded_provider_id`、`actual_invoked_provider_id`、`consumer_received_provider_id`を別観測として保存。選択Bに対してAが呼ばれれば、結果のbytesが同じでもFAIL。静的Manifestの値を四欄へ転記してPASSにしない。
- Source raw SHA、Manifest digest、Provider Lock、toolchain/generator、Build Artifact、Source Revision、Consumer Receiptを別々に保持。同じdigestでもA→B→A、Owner受理、Current Host Grant、Audienceを推論しない。
- F0形式・F1必要Source/Role/Relation閉包・F2 Owner Meaning/Host Grant/Disclosureは別判定。形式が不明でもDraftとして保存可能なことと、TEI Loader/Builderの受理を混同しない。
- 正常な空`result:[]`、未取得、not-found、permission-denied、partial、unsupported、unknownを分ける。未探索の対象を「存在しない」と推論しない。
- Core最小化／Everything is Pluginを維持。第二のHost、Registry、Parser、Policy Engineを作らず、現mainのPlugin Selector、共通Application API（ADR-0048）、CLI/Inspector/SHO G1b境界を優先する。

## 実装・検収の順序

| Gate | 一件目の実作業 | 合格に必要な証拠 | 未達時の停止 |
| --- | --- | --- | --- |
| P0: 現main探索 | Capability→Selector→実Build/Load→Invoke→CLI Replyをコード/テストから列挙 | 各辺の実Call Site・固定Revision・既存Adapter・未実装箇所 | Offer宣言や選択UIだけならK2-Aへ進めない |
| K2-A | Provider A/Bを用意しB選択→B実Load→B実Invoke→TEI CLI受領 | 同一Attemptの選択/Load/Invoke/受領ID、実Source/Manifest消費、独立Expected | A誤呼出し、未観測、異Attempt Receipt混合はFAIL |
| K2-B | SHO private側の実Reader/Adapterから同じProviderを呼ぶ | private権限内で実Invoke、Raw SHA、Reader版、未解析Reference、独立Receipt | 公開TEI CIにprivate SHOのPAT/Sourceを持ち込まない |
| K2-C | 両ConsumerのRequired→Received差集合を照合 | 事前固定Expected、取得/未取得/unsupportedの別票、Coverage上限 | 2 CI緑やMarkdown tokenだけで完全性を主張しない |
| K2-D | Provider曖昧、A誤呼出し、Source-only/Manifest-only、ABA、権限撤回、partial配送 | 一因子ずつの拒否・失効Receipt、旧Artifact/旧Evidence再利用の個別判断 | 不正な自動Fallback、旧Receiptの現在資格化を拒否 |
| K2-E | 2 Provider×2 Consumerの4 exact tupleとCLI/Studio限定Parity | 実Build/Invoke、独立Expected、同じ対象Scopeの比較 | 表示一致だけでRuntime等価としない |

**実装順序上の重要点**：K2-Aに先立ち、既存の実ProviderがSource/Manifestのどちらを実消費するかを特定する。#3479のSource-only drift検証を、K2-Aの実消費証拠へ接続する。SHO #27のK1R受領はK2-Bの入力Fixture再利用に限り、Provider共通化済みとは呼ばない。

## 横断製品への責務分離

| 製品 | K2と共有するもの | K2に持ち込まないもの |
| --- | --- | --- |
| TEI | 型・Source/Meaning境界、Provider契約、共通API、Context Query、Receipt | Source一致だけによるMeaning受理 |
| SHO | 原文位置・Reader/Editor境界、Candidate Review、Working Copy | K1R読取からの正式Writer/ABA/CAS保証 |
| SUI | Item/Occurrence/Canvasの参照・投影Loss観測 | 表示OccurrenceをSource Identityとみなすこと |
| SEI | 根拠と独立Expected、推論と判断の区別 | 推論結果によるOwner採否の自動代替 |
| KON Dataspace | 将来の保存・版・権限・公開境界とのAdapter | KONをK2必須Host/Registryにすること |
| TAI/外部Harness | 実行Receipt・外部受領の独立検証 | 内部CLI成功からH2/H3完全配送を推定 |

## 証拠状態・更新手順

K1上流Reference限定PASS、SHO G1bのK1R実Reader限定PASS、TEI #3479のSource-only drift PRマージ、SHO #29の選択範囲変換PRマージを**別々の成果**として扱う。**K2-A〜E、正式F0独立Oracle、Owner受理、正式Writer、H2/H3は本計画では未検証**。

本計画の更新時には、対象main SHA、Provider/Consumer実Call Site、各AttemptのSource/Manifest/Lock、CI URL、失敗ログ、実際の効果範囲を記録する。CIのqueued/in_progress/cancelledをsuccessとしない。PRをマージする際には旧#3329の計画を二重正本にせず、必要に応じて親計画へ本書をリンクする。


## 横断設計の追加ゲート（2026-11-12）

### 証拠の所在とGit連動ストレージ

Gitのcommit/PRには**小さいManifestと検証可能な参照**を置き、巨大な原文・実行ログ・画像・録画はNAS/S3等の外部Evidence Storeに置ける構成とする。Evidence URI、Content SHA、bytes、media type、scope、producer、created_at、retention、access policy、redaction statusを参照契約として扱う。ローカルパスや署名付きURLを永続Identityにしない。**Manifestのコミット成功はBlob実在・取得可能・改竄なしの証明ではない**ため、別途HEAD/GET相当の実取得・digest照合Receiptを要する。未取得、partial、permission-denied、expired、deleted、offlineを区別し、Blob取得不能時にGit側の旧Manifestを「最新Evidence」と誤表示しない。保存先・暗号化・鍵・GCは外部Storeの責務であり、K2専用Storage/Registryを新設しない。

### Source/Meaning/Executionを跨ぐ再利用禁止

- Source Observation（原Byteと取得時点）、Structural Candidate（Parser版と範囲）、Accepted Meaning（Owner・Audience・適用Scope）、Build Artifact（Generator/Toolchain版）、Execution Receipt（Actual Invoke/Effect）を**別Identityと別失効規則**で管理。
- Fingerprintが同一でもA→B→Aの中間変更は分からない。Content-addressed Blobの再利用可否と、現在の権限・Owner受理・Consumer完備は独立判定。
- **二Consumerの同一性**はProvider LockとSource/Manifest実消費だけでなく、二つのConsumerのAttempt ID、Raw/Parsed受領範囲、Required差集合、失敗状態の照合で証明する。片側のCI緑を他方へ移さない。

### ハーネスとGUIの安定した選択対象

CLI/Studio/SHOの対象指定は、一時的な画面選択や自然言語の曖昧な参照だけに依存させない。Source ID、revision、qualified path、byte range、semantic target ID、selected scopeを**固定したSelection Context**として渡し、選択解除や画面再描画があっても意図を勝手に変更しない。UTF-16 Editor rangeとUTF-8原文byte rangeの対応はSHO #29の限定成果を利用するが、行番号・文字列一致のみでSource同一性を断定しない。Targetが移動・改訂・曖昧化した場合は再解決・明示確認を要求する。Harnessは共通Session/Tool契約で接続し、特定のDSH/Claude/Codexだけに意味契約を埋め込まない。

### 自己記述DSL・Plugin Lifecycleへの接続

K2の各Gateは将来のTEI標準データ構造とLifecycle図から表現・生成できるようにし、Rule/Relation/Capability/Required/Negative Caseをデータとして保存する。ただし**DSL自己記述が実Provider実装やOwner承認の証明になるわけではない**。Lifecycle Pluginが追加するHookはpre/post条件・Effect class・再試行・失敗時の補償を宣言し、必須検証のスキップや前回Grantの再利用を許さない。実Providerが未実装なら図面上の経路を`not-implemented`として表示する。

### 追加受入シナリオ

| Case | 入力・操作 | 独立Expected |
| --- | --- | --- |
| E1 | Git ManifestはあるがS3/NAS Blobが消失 | `unavailable`、K2 Receipt不成立、旧Blobを自動代替しない |
| E2 | Evidence URIは同じだがBlob byteが異なる | SHA不一致で拒否、Owner Meaning不変でも再受理へ進めない |
| E3 | GUI選択後にSource revision更新・対象移動 | 旧Selection Contextで誤編集/誤Invokeせず再解決 |
| E4 | 同label・別IdentityのGraph node/Relation | Source/target/edge IDを保持し、曖昧Bindingを拒否 |
| E5 | Lifecycle PluginがRequired/Grant Gateを省略 | 実行を拒否し、診断に省略箇所を示す |
| E6 | TEI側はRaw全受領、SHO側は一部Reference未解析 | 二Consumer完備はFAIL/UNKNOWN、各Consumer局所Receiptは保持 |
| E7 | Source digest同値・Provider Manifest版変更 | Build/Invoke再検証、旧Receiptを現在実行の証明にしない |

**状態**：E1〜E7は計画上の独立Expectedであり**NOT RUN**。NAS/S3連携、Selection Context固定、Lifecycle生成、共通Harnessの正式接続をK2実装済みとみなさない。


## 2026-11-12 三要素分析法・標準データ表現・横断検証への接続

### モデルの意味と出典の関係

TEIで業務モデルを記述する際は、渡辺幸三氏の**三要素分析法**を基準用語として用い、X-TEA Modelerで示されるData/Function/Businessの図面・相互参照・CRUD表を可能な限り正確に扱う。OKFのConceptやRelationを三要素分析法のモデルへ自動的に同一視しない。対応付けは**出典付き候補→独立した意味照合→Owner受理**の順とし、X-TEAの原方式から拡張・変更した箇所は明記する。

K2のReaderが受領するものはまず**原文と構造候補**であり、Data/Function/Business上の業務的意味ではない。K2-CのRequired閉包は「必要なSource/Role/Relationを受領したか」を確認する段階であって、業務概念の受理ではない。構造のidentity/方向/多重度を失わず、同label別Identity・多重辺・複合キー・NULL/欠損/非開示を別々に表現する。

### 標準データ構造・DSLとOKFの境界

TEI標準データ表現（基本型、コレクション、表、グラフ、階層、時系列等）へOKFを投影する場合、原文の型・Source Location・Revision・粒度・単位・精度・NULL/欠損・未知拡張を失わないことを受入条件とする。Markdown/Graphviz/関数チェーンは糖衣表現として扱えても、編集・保存時に原表現を破壊しない。**Everything is Data**による関数・Ruleの自己記述は将来の表現基盤として利用できるが、OKFの実Provider実行とOwner受理を代替しない。

日常DSLはドメイン専門家の可読性を最優先とし、Go固有のstruct/func/import等を露出しない。TEI Queryのwhere/aggregateやRDB/階層/グラフの横断取得は独立Capabilityとして扱う。OKF Readerの成功からSQL生成、DB方言吸収、SmartDB保存、業務トランザクションを実装済みとみなさない。

### Provider契約と効果境界

Provider契約はCapability/Offer/Binding/Selection/Build/Invoke/Receiptの段階を分ける。DB Provider、OKF Reader、Target Generator、Language Plugin、Operator Contract Providerを同じ「Plugin」の名で効果的に混同しない。**read-only Reader**の合格をWrite/Deploy/DB Commitの権限に転用しない。

業務Functionの例（`order_entry.recent_orders`）とApplication APIの例（`sales.order.submit`）は、K2 Readerとは別の縦切りとして追跡する。後者のEffect/Operation Identity/重複実行防止/実Receiptは、X-TEA由来のSmartDB・XF実行検証で既知の例外後状態残留やDB制約拒否を考慮し、静的な成功経路だけで受理しない。

### 横断の敵対的テスト

| 反例 | 期待する判定 |
| --- | --- |
| OKFの同名Conceptを別ScopeのData/Function/Businessモデルへ誤Binding | identityとSourceを照合し拒否。名称一致のみで意味受理しない |
| 同label二ノード・多重有向辺をグラフ表示へ投影 | node/edge identity、方向、多重度、属性を保持。Lossがあれば診断 |
| `NULL`、空集合、未取得、permission-deniedを同一値へ畳み込む | 型・状態を区別。Requiredの完備はFAIL/UNKNOWN |
| 構造候補の変更を伴わないSource Revision更新 | Source Observationを再照合。Meaningの自動再受理はしない |
| 同一SourceでGenerator/Toolchainだけ更新 | Artifact/Evidence再利用を別判定。旧Receiptを現実行へ流用しない |
| Read-only OKF ProviderをWrite/Deployとして呼ぶ | Capability/Effect/Current Host Grantで拒否 |
| SQL Nativeと標準Queryの変換で業務条件が欠落 | 独立Expected・負例比較で検出。自動最適化を意味一致とみなさない |

### CI・検証証拠（限定）

2026-11-12確認：本PRの旧exact HEAD `07d68486c9ef814e23a2fde737762f20e275ffc2` に対して、[TEI PR・main健全性確認 #43147121828](https://github.com/hat47x/tei-platform/actions/runs/43147121828) と [TEI Cache Full vs Incremental Verification #43147121814](https://github.com/hat47x/tei-platform/actions/runs/43147121814) は**completed/success**。この証拠は当該HEAD・当該CIに限定し、後続コミットの成功、K2実Provider、三要素分析法の正式受理、DSLの完成を証明しない。

**次のゲート**：P0の実Call Siteマトリクスを現mainのコード・テスト・固定SHAで埋める。K2-Aに必要な「選択B→実Load B→実Invoke B→独立Expected」を一つの実Attemptで確認できない限り、K2-AをPASSにしない。


## 2026-11-12 検証計画の収束：独立Oracle・生成AIハーネス・段階的な正式採用

### 受入を自動化する前に「独立Expected」を固定する

テストDSLと生成AIによるテスト拡張は、同じ実装からExpectedを逆算してしまう循環を避ける。固定Fixtureの原Byte・仕様上の制約・Ownerが受理した範囲・外部Reference Readerの観測を**別々の根拠**として扱う。上流Reference Readerに既知のGapがあれば、上流が受理したことをOKF仕様適合とみなさない。

一件目の独立Oracleは、(a) Provider A/Bの選択、(b) 実Invoke ID、(c) Source/Manifest改訂、(d) Consumer A/BのReceived集合、(e) Required集合、(f) current Grantを入力とし、`PASS/FAIL/UNKNOWN/NOT RUN`を出す**read-onlyの検証器**として設計する。実Providerを実行する経路とOracleの判定ロジックは分離する。Oracleが同じRuntime結果をExpectedとして採用した場合は不合格。

### 受入票の粒度と機械可読な試験DSL

| 契約単位 | 最低限の検査 | 誤判定を防ぐ条件 |
| --- | --- | --- |
| Source/Reference | identity、raw digest、revision、scope、read status | stale/unavailable/partialをfreshへ自動昇格しない |
| Structure | type、source span、node/edge identity、NULL/欠損、parser pin | 見た目の一致で同一構造としない |
| Provider | Capability、Offer、Selection、Load、Invoke、effect class | 宣言上のProvider IDを実Call IDとして流用しない |
| Consumer | TEI/SHO別のreceived、unsupported、missing、partial | 二つの別CI成功を同一Attemptとしない |
| Authority | Owner、Audience、current Grant、Disclosure、Effect | AI提案や古いGrantを現在の許可へ昇格しない |

試験DSLの標準化では、表形式の近傍テストを入力とし、内部表現は独立Expectedと実Observationの二系統を保持する。K2のためだけに新しい専用DSLやPolicy Engineを先行実装しない。

### AIハーネスによる実装支援の安全な責務

DSH、Codex、Claude Code等のハーネスは共通の選択対象・Source Scope・Tool Contract・Receiptを通して等価に扱う。ハーネス固有の実行手順はAdapterに隔離する。AIに実装やテストを委任しても、**Ownerの業務意味受理、Current Host Grant、正式Publish/DB Effectの許可を自動で得たことにはならない**。

生成AIが検証コードとExpectedを同時に生成した場合は、独立性を満たさない可能性を診断する。AIが作った候補はDraftに留め、仕様・固定Fixture・反例から別経路で照合する。対象Selection ContextはSource ID/Revision/Byte span/semantic targetを固定し、GUI操作で偶発的に選択が解除されても勝手に対象を変更しない。

### P0〜K2-Eの具体的な作業分割と停止条件

1. **P0a（実装探索）**：現main SHAを固定し、`Capability→Offer→Selection→Build/Load→Invoke→CLI Reply`の実Call Siteをファイル・行・テストで列挙。見つからない辺を明記。
2. **P0b（独立Oracle）**：Fixture/Required/ExpectedをRuntimeから独立に固定し、選択B→A実Invoke、正常空result、部分配送、Source-only driftの最低4負例を先に定義。
3. **K2-A（TEI実呼出し）**：既存Provider契約を用い、選択B→実Load B→実Invoke B→CLI実受領を一Attemptで観測。静的選択やmockのみなら不合格。
4. **K2-B/C（SHO実呼出しと閉包）**：SHO private側で同一Providerを実Invokeし、TEIとSHOの各Receiptを独立Expectedに照合。権限境界を維持。
5. **K2-D/E（敵対的・拡張検証）**：A/B誤呼出し、Source/Manifest 2×2改訂、ABA、Grant変更、partial、2 Provider×2 Consumerの4 exact tuple、CLI/Studio限定Parityを実行。
6. **Owner採否**：F0/F1/F2、原方式との差分、K2の性能/効果/リスク、正式Plugin契約を別々に審査。K2の限定PASSを正式製品導入へ自動昇格しない。

### 証拠の鮮度

直前HEAD `d141bd8c6320c5823625feee6b546c919f0a38b2` のCache Full vs Incremental CI [#43151569560](https://github.com/hat47x/tei-platform/actions/runs/43151569560) は**completed/success**。Repository Health CI [#43151569576](https://github.com/hat47x/tei-platform/actions/runs/43151569576) は確認時点**in_progress**。この後のコミットについてはCIを改めて確認すること。**P0a/P0b、K2-A〜E、正式Oracle、Owner受理はNOT RUN**。


## 2026-11-12 統合レビュー：実Provider先行・文書Handoff・Typed Lifecycle

### 統合判断：K2の前提を増やさない

K2の最小縦切りは、現行`.tei`定義を起点とする**実Function Provider Read**、既存Typed Application API、既存CLI/Studio Adapter、SHO実Readerの独立受領とする。`*.tei.md`全面移行、OKF独立Host、万能Harness、独立Policy Engine、Plugin Graph Patch ABIの確定は**K2のP0条件にしない**。Markdown/OKFは原文を失わないSource/Reference投影として追加し、業務FunctionやDB Readの存在を形式読取だけで証明しない。

P0aは二つの経路を別票にする。(A) OKF Reader/Source→Provider→TEI/SHO Consumer、(B) `order_entry.recent_orders`等の現行`.tei` Function→Query→DB Binding→実Provider Read→Rowset→CLI Reply。Aの実Reader成功はBの実DB Provider成功を意味せず、Bの成功もOKF F0/F1/F2の合格を意味しない。

### Typed Lifecycle Graphと実行権限の境界

Plugin lifecycleを将来型付きGraphで表す際の最小ノード候補は`Activity`、`Interaction`、`Operation`、`Wait`、`Approval`、`Observation`。各ノード/辺はGuard、Owner、Effect、Version、Source Locatorを持てる。ただしGraphの**編集可能性**と、Hostが実行を許可した**Current Grant**は別資格。GraphにApprovalノードがあることはOwnerの実承認ではなく、Operationノードがあることは実Invokeではない。

Graph投影の同label別Node、複数Edge、Source/Target向き、Condition、Owner、Effectの欠落はLossとして診断し、保存/再読込でidentityを失わせない。Plugin配布方式・BOM Schema・Patch ABIは未決定として留保する。

### SHO・SUI・TEIの文書Handoffと著者編集保護

SHOの`Working Copy`、正式保存（Conditional Update/CAS）、保存後の**独立再読取**を別Gateとする。SUIからSHO/TEIへ渡すHandoffには固定した対象ID・Source Revision・選択範囲・送信元と受信先Scopeを含める。Handoff Receiptは**対象が届いたこと**を示すにとどまり、AIセッションの継続、編集の保存、Ownerの受理、生成文書の公開を意味しない。

TEIからSHOへ文書を生成/再投影する際は、著者編集領域と生成領域を区別し、再生成が人間の編集を黙って上書きしない。Source移動、同名別内容、CAS競合、変更なし保存、再開時の参照再解決を別試験とする。SHO #29のUTF-16→UTF-8範囲変換のマージは必要条件の一部であり、正式保存・再投影競合の合格ではない。

### AIハーネスの最小相互運用試験

共通Typed Application API上で`Capability Discovery→Preflight→Invocation→Post-check→Resume`を実行順序として検証する。ハーネス別Adapter（DSH/Codex/Claude Code等）は共通Scope/Grant/Receiptを保持し、機能差はCapabilityで明示する。Resumeが過去AttemptのReceiptを再表示しても、**新しい実Invokeの証拠にはしない**。Preflightが成功しても実Effect許可・実受領は別票。

比較項目は再現性、対象誤指定、拒否の正しさ、Effect境界、再開時の古いGrant/Source Revisionの失効、実Provider接続、追加レイテンシ。K2のために新規Harness本体を実装しない。

### 既存修正との依存関係

[TEI PR #4507](https://github.com/hat47x/tei-platform/pull/4507) は**マージ済み**（2026-11-12再確認）。Fingerprint自己混入防止、AuthorityLimits保持、図Draft回帰試験の修正成果は、現在のSource/Artifact/Evidence失効設計を検証する際の関連実装として追跡する。ただしPRマージをK2 Provider実Invoke、Current Grant検証、Owner Meaning受理の証拠に流用しない。

### 追加受入票（独立Expected、未実施）

| ID | 実行する反例・正例 | 合格条件 |
| --- | --- | --- |
| I1 | 現行`.tei` Functionから実DB Provider Read | Query/Binding/実Provider/Rowset/CLI Replyの同一Attempt証拠。未接続なら未実装 |
| I2 | OKF原文Readerと実DB Readの証拠を比較 | 各々別Scope・別Receipt。片方のPASSで他方をPASSにしない |
| I3 | Lifecycle GraphのApproval表示だけでInvoke | Current Grantがなければ拒否。Graph表示は許可を発行しない |
| I4 | SUI→SHO Handoff後にSource移動/改訂 | 固定Selection Contextを再解決し、曖昧なら停止 |
| I5 | TEI生成文書の再投影とSHO著者編集が競合 | CAS拒否時はWorking Copy保全。黙った上書きをしない |
| I6 | AIハーネスResumeで古いGrant/Receiptを再利用 | 新しいPreflightとCurrent Grant照合。古いReceiptを新Invokeとみなさない |
| I7 | Fingerprint同一・AuthorityLimits/Generator変更 | Source再利用とMeaning/Artifact/Grant資格を別判定 |

**証拠状態**：I1〜I7は計画でありNOT RUN。旧HEAD `c2ab617097c0eb7292c38265de4b8bd8ec0dd8c1` の[Repository Health #43152455070](https://github.com/hat47x/tei-platform/actions/runs/43152455070) と[Cache Verification #43152455133](https://github.com/hat47x/tei-platform/actions/runs/43152455133) は**completed/success**。この後のHEADは別途CI確認する。
