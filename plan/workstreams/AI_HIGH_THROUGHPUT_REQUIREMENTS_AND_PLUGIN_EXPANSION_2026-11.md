# AI大量並列開発と要件ポートフォリオ：外部要求からTEIの共通構造を育てる計画

- **位置付け**：2026-11-01、開発規模の拡張を前提とする実行計画案。正式なメタモデル／業務意味／受入条件の採用ではない。
- **調整Issue**：[#2493](https://github.com/hat47x/tei-platform/issues/2493)
- **全体開発順序**：[Draft PR #2492](https://github.com/hat47x/tei-platform/pull/2492)で提案された`plan/MASTER_PLAN.md`と`AGENTS.md`のAI-first方針が上位。Draftの未統合状態は維持する。
- **既存責務**：[#1973](https://github.com/hat47x/tei-platform/issues/1973)（PoC抽象化）／[#1119](https://github.com/hat47x/tei-platform/issues/1119)（横断依存）／[#2017](https://github.com/hat47x/tei-platform/issues/2017)（Agent作業票）／[#1445](https://github.com/hat47x/tei-platform/issues/1445)（開発工程Plugin）／[#2018](https://github.com/hat47x/tei-platform/issues/2018)（生産性比較）。

## 1. 前提変更：拡張の余地を十分に確保する

生成AIトークンを安価に大量調達し、複数のAIエージェントによる調査、設計、実装候補生成、レビュー、差分検査を大幅に増やせることを前提とする。調達価格・利用枠・実スループットは将来の運用条件であり、契約済みの値として扱わない。

これまでの「少数の最小Referenceを順番に完成させる」という考え方は、**早期に反例を得る方法**として維持する。一方、製品領域、要件集合、標準Plugin群、AIエージェントの並列検討とIssue数は、**小さく抑えること自体を目的にしない**。網羅的な外部要件収集と異質な利用条件への圧力試験を、独立に増やしてよい。

開発速度の増加を、単純な生成コード行数ではなく、次の変化で評価する。

- 検討できる要求・Consumer・Target・業務外Domainの種類と、**根拠のある新しい差分**。
- 既存モデル・契約・Pluginの再利用で解ける要求の増加。
- 採用した構造が別の要求群を解く場合の追加実装・Adapter工数の減少。
- 原典・業務受理・実Provider/Hostの差を明示できる反例・検証証拠。
- 独立に統合できる変更の数と、競合・再検証・レビューの実負担。

### 同時に保持する保証

安価な推論・大量生成で得た候補は、**候補の量と多様性**の拡張である。意味の受理、権限付与、顧客データの持出し、実運用配備は`AGENTS.md`のA0〜A4と各Ownerの責任を維持する。これは探索候補数を制限する理由ではなく、候補から採用済みの主張へ至る証拠条件を明確にする理由である。

CPU/GPU、ネット接続、GitHub Actionsの利用上限、実DB/Windows/Browser環境、人間のレビュー時間は、トークン単価と異なる資源である。探索を先行して蓄積し、必要な実環境検証を適切な単位にまとめる。CI消費枠は計測し、不要な全件繰返しを抑えるが、必要な実証条件は維持する。

## 2. 外部要求の収集・分析を並列化する

### 2.1 収集単位

「製品機能名」を直接Core Typeに採用するのではなく、**利用者が何をし、そのとき何が正しくなければならないか**というPressure Caseを収集する。各ケースに次の観測列を対応付ける。保存場所は既存Issue・要求・Scenario・Source参照を優先し、第二の正本DBを作らない。

| 観点 | 最小情報 |
| --- | --- |
| 出典・条件 | 外部製品、原典/公開仕様・観測時点、権利/開示範囲、確認済みと推測の区別 |
| 対象 | Actor/Owner、Domainの意味、操作の対象ID、粒度、利用場面 |
| データ | Entity/Relation、1:N/N:M/自己参照、Snapshot、完全性、欠損/NULL/0件、Data Context |
| 操作 | Function/Action、Intent、Validation、権限、Effect、失敗・取消・再試行 |
| 表現 | CLI/API、Table/Form/Chart/Tree/Timeline/Gantt/Map/Native、Viewerが必要な役割 |
| 実行先 | Plugin/Provider、Host obligations、Runtime/Target、配布/実Load/更新方式 |
| 適合 | TEI共通構造／標準Plugin／専門Plugin／Host Adapter候補、表現損失、未解決 |
| 証拠 | 正例と重要な負例、独立Observation、実験担当Issue、実証の対象版と状態 |

### 2.2 追加候補の広い領域

体系の分類自体を暫定とし、要件を分野横断で集める。

| 系列 | Pressure Case候補 |
| --- | --- |
| 基幹業務とERP | 受発注、契約、在庫、会計、購買、承認、複数帳票、拠点・通貨・税制 |
| 開発・保守 | EAM/CMMS、ITSM、構成/脆弱性管理、PLM、監査・変更申請 |
| 文書・知識 | SHO設計書、引用の原本再解決、KaTeX、版固定、レビュー、複合資産 |
| 分析・データ処理 | SQL/CTE可視化、OLAP、集計、イベント・時系列、ストリーム、ETL |
| 可視化・操作 | 多並列1:N、Graph/Canvas、Timeline/Gantt、地理、ツリー、関係編集 |
| 組織・プロセス | グループ企業、兼務、代理、権限委任、柔軟Workflow、例外経路 |
| 専門計算・研究 | DSL/Lean、統計、シミュレーション、計算結果と説明文の別管理 |
| 端末・IoT・オフライン | 静的Library、WASM、Edge、同期・競合、環境制約、外部Host |
| AI・エージェント | CUI、外部ハーネス/TAI、推論結果の来歴、反例生成、長時間Task、ResultClaim |
| 多形態の提供 | 個人ローカル、SIer案件移管、大規模SaaS、複数Tenant、再配布と更新 |

上表は探索の起点であり、完了したカテゴリ一覧でもv1の固定スコープでもない。独立した価値や新しい反例があれば、派生Issue・標準Plugin候補を積極的に増やす。

## 3. 同時進行の運用方法：発散を広く、実証を具体的に

**要件探索／候補設計／実証実装／横断統合**を、直列工程ではなく互いに結果を供給する並行レーンとして運用する。

| レーン | 主にAIが進める活動 | 出力と利用先 | 終了・停止を要する場合 |
| --- | --- | --- | --- |
| 要件探索 | 外部SaaS・OSS・参照モデル・業務外分野の例、例外、矛盾、相違の収集 | 出典付きPressure Case、異論、追加Issue、未採用候補 | 公開できない資料の転載、出典不明の事実断定 |
| 構造比較 | 複数モデルへの写像、共有Kernelの不足、Plugin境界案、変更影響 | 対応表、反例、設計候補、可逆な差分 | 業務意味の自動受理、既存Ownerの権限変更 |
| 独立PoC | 部分構造のコード、CLI、標準Plugin、Provider/Adapter、fixture | 隔離branch、限定試験、実bytes・版付きEvidence | 不明依存のPASS扱い、未観測Effectの自動再試行 |
| 横断統合 | 重複の整理、Shared Contractの反証、Producer/Consumerの接合、Issue責務整理 | 既存ADR/設計・Issueリンク、採否理由、再検証範囲 | 実行証拠の寄せ集めから全体PASSを作ること |
| 人間の判断 | 意味の受理、重要な製品選択、権限・秘密、ライセンス、本番利用 | Owner Decision、適用Scope/時点、再開方針 | 候補・合成PASSを人の判断へ読み替えること |

**候補の探索・蓄積には予め固定した件数上限を置かない。** ただし同じ課題を重複実装する場合、事前にOwnerと版・変更対象を確認し、別branchで実験し、比較結果を集約する。既存の[#2017作業票](https://github.com/hat47x/tei-platform/issues/2017)とGitHub PR運用を使い、新Agent Manager・受理Engineは実装しない。

比較的安価なAIには、同じSourceからの多視点読解、反例探し、異質なConsumerの制約分析、テスト候補の生成を多く担当させる。**同じAIの肯定だけを独立Oracleと呼ばず**、固定fixture・仕様出典・実Provider・別の観測経路へ接続する。異なるAgentの合意は根拠不足を隠すものでもない。

## 4. Core/Plugin設計を拡大に耐えるものとして試す

### 4.1 共通の考え方

- 意味モデルと業務的受理は**原モデル1→TEI統合モデル1**の連結を維持。コード/SHO文書/シミュレーションは同じ受理版を参照して導出する。
- 物理型/値域型、意味領域としてのDomain、Data Context、Entity/Relation/Function/Procedure、DSL/Operator/Service Contract、Target/Hostは違う責務として比較する。
- Plugin Package、Contract/Definition、Provider、Artifact Set、Host Adapter、Consumer利用単位は1対1に固定しない。
- 生成AI用CUIはTEIの操作入口。TEI外Consumerは必要なContract/Providerを自らのHostに取り込める。生成成果物にGo Runtime、TEI専用Host、共通Registryを必須にしない。
- 標準Pluginを多く増やしても、サービス契約とHostの現在Authorityを分ける。Pluginの自己申告が権限にならない。
- 複数Providerの合成は、個別PASSから全体の完全性・Snapshot・権限・業務完了を自動推論しない。#2390の合成負例を再利用する。

### 4.2 基盤設計の反例にする条件

1. 異なる要求が増えるたびにCore固定enumや固定画面タイプを追加しないと表現できない。
2. D1(2件)とD2(3件)を一画面の二つの独立Data Contextで保持できず、結合した6行が業務対象IDになってしまう。
3. ChartのAggregate Point、GraphのPath occurrence、MapのSpatial Featureを単一Entity Recordに潰し、Actionの対象を誤る。
4. 同じServiceの外部Java/.NET/Process/Wasm Consumerで、TEI Studio/Go/独自Hostを不要とする構成が成立しない。
5. 生成コードとSHO文書を異なる意味正本で維持し、意味変更と説明/書式変更を分けて再生成できない。
6. Provider個別Contract合格から部分取得、異時点、間接Authority、取消後Effectの総合保証を誤って発行する。
7. 標準Pluginを追加すると、CLI以外のGUIやMCP専用経路がない限り業務検証できない。

反例は**基盤改善へつなぐ入力**であって探索・要件追加の停止理由ではない。表現不足が確認できたら、新しい抽象・Plugin・Adapter・Targetを候補として提案する。

## 5. 最初の並列Pressureセット

| Pressure | 最初の実証/比較 | 既存担当を使う範囲 | 新しい課題化の条件 |
| --- | --- | --- | --- |
| P-F1 | 親H→D1(2)/D2(3)、Table/Chart、grain・部分取得・Action対象 | #1127/#1706/#997/#2387 | 実Consumerに必要な最小構造が既存契約へ表現できない |
| P-P1 | 一契約から外部Host Library/Process/Wasmへ選択配布 | #2390/#1861/#2392/#2077 | clean Consumerで新たなAdapter・権限・ロード問題を発見 |
| P-D1 | 受理モデルからJava/.NET等LibraryとSHO説明・引用資産を同時導出 | #1803/#2016/#2077、SHO/MMI | 影響分岐・Review/provenanceの実欠落を確認 |
| P-W1 | 代理・組織Service Identity・動的Workflow・長時間Task | #2386/#1129/#1174、TAI | 共通Contractでは受け止められない権限/時間境界を検出 |
| P-Q1 | CTE/相関/同名・2waySQL/DSLと生成先を往復 | #2387/#988/#1063 | 値域型・SQL意味・Source lossを検証で特定 |

これらは同時に進められる探索候補であり、**すべての完了を単一縦切りの開始条件にしない**。同じFixture/Contractを参照する関係はGitHub上のリンクで管理する。

## 5A. 新しい横断Pressure：多重facetを持つデータをPlugin境界で安全に再利用する（2026-11-01／未採用）

**入力元**：[#2495 データ構造のメタ概念空間](https://github.com/hat47x/tei-platform/issues/2495)／[Draft PR #2496](https://github.com/hat47x/tei-platform/pull/2496)。#2495は「対象・構成・性質/法則・解釈文脈・利用条件・根拠・実現」の七役割を暫定的に整理し、数学的位相とGraph/集合/物理格納を区別する。**分類・Value Domain・数学的法則のOwnerは#2495**であり、本計画はそれを「別Plugin/Consumer/Hostを跨いでも必要な法則が維持されるか」という追加Pressureに変換する。新しいCore型、推論器、標準Plugin規約の採用はこの節だけでは行わない。

### 基本仮説：Shape一致は「操作の意味を保存する写像」の成立を証明しない

ProviderからConsumerへの変換`g`と、Consumerが要求する操作`f`を考える。両者の接続には少なくとも、
- 対象・Identity/Grain（どの出来事/実体/集合についての結果か）、
- 構造・Cardinality/Order/Multiplicity（集合・bag・sequence、集計粒度）、
- 解釈（温度の**点**か**差分**か、単位/換算元、イベント時刻か到着時刻か）、
- 操作の法則と前提（演算の定義域、変換の全域性・情報損失、並べ替え・filter・aggregate・limitの順序）、
- 観測（取得完全性、watermark等の明示的な保証、Snapshot/版、未知の範囲）、
- 利用時Host条件（現在権限、開示先、実Load依存、失敗/unknown）
を比較する必要がある。

演算保存を要求するときは、**適用可能な入力集合を限定したうえで**`g(f_source(x))` と `f_target(g(x))` が一致するかを問う。これらを任意の演算で交換可能とはしない。演算の法則が未宣言、変換が部分的、または対象集合の完全性が不明なら、`candidate/unproven`として扱い、自動適合/自動換算へ昇格させない。既存#951のService Contract・Conformanceと#2390のConsumer/Host適合票の**比較条件**として表し、第二のContract SSOTを作らない。

### 既存の温度イベント合成例をPlugin/Target横断の一件に使う

#2495の合成イベント`e1(10:05,20°C)`、`e2(10:10,21°C)`、`e3(10:07,20.5°C)`を同じ版の入力とする。到着順はe1→e2→e3、測定時刻順はe1→e3→e2。対象時間窓を`[10:00,10:09)`とする。**取得した3件が窓の完全集合である証拠はない**。

| Case | 操作上の正しい扱い | 具体的に誤る経路 | Owner |
| --- | --- | --- | --- |
| **FCT-1 / 時刻** | 測定時刻窓で選ぶと**既知の**対象はe1,e3。最後の2観測を測定時刻順で選ぶならe3,e2 | arrival順の先頭2件e1,e2を取ってからevent-time sortして「最新2件」と宣言／e3遅延を無視して窓を完結 | #2495/#1706/#1127 |
| **FCT-2 / 単位・演算** | 温度点→Kは`T_K=T_°C+273.15`、二温度点の差は`ΔT_K=ΔT_°C`。差分と点の意味を維持 | 点20°C+点21°Cの合計を温度点41°Cとして扱う／差分に273.15を加える | #2495/#1705 |
| **FCT-3 / Identity・集計** | 値が同一でも別Event IDなら両方を保持。Aggregationは対象の完全性が確認できる範囲で主張 | set-of-values化して出来事をdeduplicate／部分窓の平均を完全な期間平均と宣言 | #1706/#1127 |
| **FCT-4 / 外部Provider** | Consumerの契約と時間/単位/Identity/取得条件に対応したHost Adapterを選び、変換の前提・loss・実装artifactを観測 | JSONが等しい、WITのShapeが一致、旧fixtureがPASSというだけで意味互換・完全性を認定 | #2390/#951/#1861 |
| **FCT-5 / 表示・文書** | 同じSource/意味版からCLIの構造説明とグラフ・SHOへの表示を派生、完全性不明を明示 | 表示上の3点・KaTeX説明があるだけで測定範囲の完全性/計算法則/Owner受理を保証済みとする | #1803/#997/SHO |

**反例の数値的判別**：20°Cと21°Cという二つの**温度点**を単純加算する操作自体が未定義なら拒否する。仮に数値だけを加算してから単位変換すると`41+273.15=314.15`、各点を変換して加算すると`293.15+294.15=587.30`となり、`sum`はこのアフィン変換を保存しない。この不一致は「温度点の合計」をContractに自動追加する根拠でなく、**操作・量の種類が不適合**であることを説明するための反例である。温度差は相殺によって正しく変換できる。

### 実証対象と判定境界

1. **机上**：#2495の分類・既存#1705/#1706の入力条件を参照し、FCT-1〜5を`meaning / operation / required evidence / Consumer / Target`へ写像。正式なMeta Schema/単位プラグインを先行固定しない。
2. **headless**：AI向けCUIは同一固定Source/Fixtureの型付きread-only診断として、選択された出来事ID、どの時刻を使ったか、単位と量の種類、取得完全性、不成立操作を説明。#2354のJSON版・失敗形式は既存担当で整備。生成AIの自然文のみを独立Oracleとしない。
3. **Plugin境界**：#2390の外部Consumer比較票へ、単なる`shape/transport`に加えて`operation law / interpretation / scope / time / completeness`の必要条件を連結。外部ConsumerにTEI CUI/Runtimeを要求せず、必要なContract/Adapter/Artifactだけを利用させる。
4. **Target別**：表/時系列図/文書表示と実行Library/Serviceは別のProjection。演算と表示の一致、外部Host権限、実DB効果などは適切なOwnerへ個別証拠を戻す。未知のFacetを“非対応”と即断するより、観測不能/未解決を区別。
5. **拡張**：温度時系列に続いて、通貨/レート時点、組織の代理・時点付き権限、Geometry/CRS、GraphのPath occurrence、確率/Measure、Mutable Collectionを並行Pressureとして追加可能。追加Issueの上限は置かず、意味・試験責務が独立するケースは積極的に分離する。

### この追加Pressureが示す具体的な価値

同じByte/型/Providerが別環境で動くことに加え、**利用者が要求する操作を、どの前提・法則・データ範囲で保存しているかを説明できること**を外部Plugin適合の品質要件候補とする。実装で必要な情報が得られない場合、CLI/Plugin/Hostのどの境界が不足しているかを特定できる。最初の課題は新Core抽象の導入ではなく、既存Contract/Collection/Quantityの証拠を同じCaseへ参照接合すること。

**Evidence区分**：今回のFCT-1〜5は#2495の合成例から構成した追加の設計比較／**NOT RUN**。#2495のDraft #2496も未統合。数学的公理の形式証明、実TEI CUI、外部Provider実行、SHO投影、Owner意味受理はいずれも本計画で新たに成立していない。

## 6. トークン規模が変わっても測定を失わない

### 6.1 分けて測る

- **探索の広がり**：出典付きPressure数、新しい意味差/反例数、利用先の種類、未決の見える化。
- **モデル適合**：Core変更なしに表現できる事例、必要な新Plugin/API、Loss/Unsupported、再利用した契約。
- **試験の質**：固定したcaseとactual実行case、負例の誤PASS、実Host/DB/Browser/独立Readbackの証拠。
- **実装の流れ**：並行Candidate、実PR、統合済み/未統合、Git競合、レビュー待ち、再検証、手戻り。
- **AI使用**：利用モデル・token/時間/実費の入手可能な範囲、同一条件の比較。大量生成による発見量と、製品として受理された価値を別指標にする。

具体的なTEI対通常コーディング/ローコードの比較・測定の正本は[#2018](https://github.com/hat47x/tei-platform/issues/2018)。本計画は**並行開発時の探索と統合の実態**だけを追加する。

### 6.2 レビュー・CI等が先に詰まる場合

同時に実装を増やせても統合作業が滞る場合は、探索/設計比較・isolated prototypeを進め、既存の試験環境を占有しない。実証は関連するfocused tests→必要な横断試験を変更差分に応じて選び、受理範囲の証拠を残す。失敗・未実施・CI予算超過を成功として扱わない。製品の実装済み範囲は[docs/status.md](../../docs/status.md)で確認し、計画の広がりを製品完成の主張へ転換しない。

## 7. 最初の作業と判断点（計画）

1. #2493にP-F1/P-P1/P-D1/P-W1/P-Q1それぞれの**最初の参照先と未解決事項**を紐付ける。各Ownerの現行Issue本文・最新コメントを確認し、重複の少ない差分から始める。
2. 既存の#1127二つのData Context、#2390外部Consumer・#1861実Componentを第一の対照とする。前者の新しいFunction Context end-to-endと後者の独立外部配布は、現時点で実行済みの証拠を持たない。
3. 要件探索を他領域（業務以外を含む）へ広げ、追加のPressureが既存境界へ収まらないと判明した場合は、**新Issueを積極的に起票**し、所有する意味・試験・担当との関係を明確にする。
4. 関連の`AGENTS.md`入口はDraft #2492の全体方針へ任せる。本PRから競合する`AGENTS.md`編集を行わない。Issue #2493と本計画をDraft #2492にリンクして参照可能にする。
5. 追加プラグインやメタモデルの採否は、具体的なConsumer・反例・比較測定が揃った段階でOwnerが判断する。Explorerが候補を多数作ることと、製品仕様の正式受理を分ける。

## 状態・読み方

この計画は**ユーザーが想定するAI開発容量増大を受けた開発プログラムの設計提案**。外部製品の公開情報、既存TEI Issue/Source、他製品の機能要件を個々に観測した結果と、TEI側の採用判断を混同しない。

`P-F1`等は本計画の作業上の参照ラベルであり、新しいCore Contract / Test Gate / Artifact schemaではない。AIエージェントによる実並列開発、実Token予算、関連負例の実行、Owner業務意味受理、Production適合はこの文書では未確認とする。
