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
