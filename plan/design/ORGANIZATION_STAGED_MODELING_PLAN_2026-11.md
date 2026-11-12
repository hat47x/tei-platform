# 高度組織モデリングの段階計画：要件定義・基本設計・実装作業単位（2026-11-11・設計・未受理）

## 0. 位置づけと状態

- 本計画は、[#2386](https://github.com/hat47x/tei-platform/issues/2386)（組織・所属・時点の標準意味契約のOwner）が持つ**要件定義・基本設計・実装計画**を、外部モデルが独立に実装・検証できる作業単位（WU）まで具体化する docs-only の計画である。
- **状態: 設計・未受理**。本計画の採用、および後述する各作業単位の実施は、Issue の作業条件として扱う。新しい公開契約（Data/Relation/Validation の新kind、CLI/MCP 契約）、業務意味の正式受理、認可境界の変更は **Owner 判断** とし、本計画は決定しない。
- 本計画が作るもの: 5段階の基本設計、標準データモデル上の具体（Identity・Relation・Cardinality・Time・Validation・Query・表示形式）、独立Expected、14個の作業単位とその対象ファイル・既存API・入力・出力・受入テスト・依存・完了条件・停止条件。
- 本計画が作らないもの: 巨大な固定組織Schema、独立した認可基盤（第二Policy Engine）、専用Workflow Engine（BPMN/DMN/CMMN ランタイム）、組織専用のCore Kind、`Person/Account/Tenant/Principal` の単一超型、第二の Scenario/Expected 正本、組織Dataへの一律のアクセス制御・匿名化・OS隔離。

## 1. 基準

| 項目 | 値 |
| --- | --- |
| 基準 commit | `69f2ebc67262355b1fae5ef0ed6a13d819793fed`（`origin/main`、2026-11-11）。着手時に最新 `main` と対象pathの差分を再確認する |
| 意味のOwner | #2386 |
| Corpus・独立Conformance | #2510 |
| 一般メタ空間（Entity・Relation・Key・型・Facet） | #1126 / #2495 / #1806 |
| Projection（構造保存・逆写像・損失） | #2497 |
| 実Read縦断 | #2586 / #2506 / #2622 |
| 独立Expected・Scenario・Attempt | #1131 / #1180 / #2514 / #1163 |
| 現在のGrant・Effect・Receipt | #1174 / #2499 |
| 三視点・Studio・Modeler | #1124 / #1125 |
| 業務手順・Handoff | #1444 / #1125 |
| Workflow / 承認Rule | #1129 / #1130 |

### 1.1 現行 main で確認済みの部品（本計画が再利用するもの）

| 部品 | 場所 | 本計画での使い方 | 確認済みの上限 |
| --- | --- | --- | --- |
| 標準 Data / Relation / Cardinality / RelationKey / CandidateKey / FunctionalDependency | `internal/semantic/model.go` | 型付きRelationと多重度で組織関係を書く | `Cardinality` は `one`/`many` のみ。期間重複・循環・時点条件は式として書けない |
| Validation | `internal/semantic/model.go` の `ValidationKind`（`positive`・`relation-unique`）と `internal/validation/validation.go` の `Validate(ir.World)` | 属性の正負、Relation target の一意性 | instance データの循環・重複期間・時点は診断しない（`SEMANTIC_TYPE_ALIAS_CYCLE` は型エイリアスの循環のみ） |
| Query | `internal/query/model.go` の `EmbeddedSQL`（`ID`・`Application`・`Datasource`・`Dialect`・`Source`・`Inputs`・`ResultColumns`）、`internal/query/analyzer`（`Scopes`・`Fields`・`Lineage`・`Coverage`・`Relational`） | as-of は SQL と入力引数で表現し、非出力依存を Lineage に残す | SQL 実行・Provider Rowset・Scope 強制は範囲外（#2586/#2622/#1656） |
| Persistence / OperationSurface | `internal/operations/surface.go`（`SurfaceWorkspace`・`DeriveOperationSurface`）、`internal/database/persistence` | read-only の list/read 候補を導出する | 実Grant・Policy Enforcement・Case の read を自動導出しない |
| ModelMap / Projection | `internal/operations/model_map.go`（`MapWorkspace`・`MapWorld`） | 三視点の読み取りとFit/Gap | 組織図・配置表・時点別組織表の専用投影は未実装（#2497/#1124） |
| Native 組織fixture | `reference/org-headless-workspace/`（Person・OrgUnit・Membership・Position・Occupancy・CaseRecord、`membership` のpersistence、read-only な `org_view`）と `internal/operations/org_headless_characterization_test.go` | Stage1 の受入基盤 | `reference/workspace` と同様、Source 変更は `reference-shared-acceptance` 経路と Decision 記録が必要 |
| 合成組織oracle | `reference/org-semantic-views/`（`verify*.mjs`、`source-fixture.json`/`scenario-fixture.json`、ORG-*） | 独立Expectedの再利用元 | TEI 製品契約ではない。実Provider/Host/Owner受理は未実施 |
| 時間の区別 | `reference/derivation-temporal/temporal.go`（valid-time・recorded-at・evaluated-at・Rule revision・confirmed fact、`historical-replay`/`what-if`/`current-computation`/`confirmed-fact-read`） | Stage3 の契約の語彙 | 実DB・Provider ではない |
| CLI の読み取り口 | `tei validate`・`tei operations`・`tei artifacts`・`tei model catalog/locate/overview`・`tei inspect`・`tei trace`・`tei workspace-files`・`tei source-index` | Fit/Gap の観測 | 新コマンド・新MCP schema の追加は Owner 判断（#3262/#1124） |
| CI | `.github/workflows/tei-org-headless-native-ci.yml`・`tei-org-semantic-reference-node.yml`・`tei-org-case-mapping-bridge-ci.yml`・`tei-org-case-runplan-boundary.yml` | 作業単位の検証経路 | 既存workflowの変更は最小限。新規workflowは追加しない |

### 1.2 未統合PRの扱い

- 本計画は **最新 `origin/main` のみ** を基準とする。未統合 Draft（組織系の #4256・#4086・#4181、#3500・#3518・#3492 等）の記述は、main に取り込まれるまで証拠として使わない。
- 並行PR（例: #4335 read-only 変更影響、#4329 Consumer 指紋、#4260 SQL方言）が触る package と衝突した場合は、該当WUの base を取り直し、衝突をPR本文へ書く。

## 2. 要件定義

### 2.1 目的

組織・所属・時点・責任を、**既存の Data・Relation・Validation・Function・Query・Plugin の範囲で**段階的に表現し、次を「無理な項目追加」「実行時の権限混同」「第二正本」なしに説明・検証できるようにする。

1. 部門、組織単位、法人、企業集団、人、雇用関係、所属、役職、職位、役割、責任の区別。
2. 単純な組織階層と、マトリクス組織、兼務、出向、複数法人、プロジェクト組織などの多重関係。
3. 過去・現在・将来の所属、異動、遡及訂正、業務有効時点と記録時点。
4. 組織関係と承認・代理行為・Workflow・権限の接続。ただし**組織上の関係が存在すること**と**実際の操作権限が付与されていること**を区別する。
5. Data・Function・Business の三視点、組織図、配置表、時点別組織表などへの Projection。
6. 一般企業・行政・団体・個人による小規模な利用から、大規模な多法人組織までの段階的対応。

### 2.2 非目標

- 全組織に共通する巨大な固定組織Schema、組織専用のCore Kind、固定テーブル群の標準必須化。
- 独立した認可基盤、第二Policy Engine、組織からの自動Grant生成、全軸への権限自動継承。
- 専用Workflow Engine、BPMN/DMN/CMMN ランタイムの標準必須化（表現・実行Adapterの比較候補に留める）。
- `Person/Account/Tenant/Principal` の単一超型、全権限を扱う第二Core。
- 組織Data（部門・職位・所属・組織図・配置表・人数）への一律のアクセス制御・匿名化・OS隔離。
- SCIM 等の外部Directoryを組織全履歴の正本とすること（Identity 交換adapter候補に留める）。
- 全OrgUnitへの単一 `parent` 列の強制、全グラフへの同一循環制約。

### 2.3 保護対象の区別（#2510 Owner方針の具体化）

- TEI の同じ業務世界にある**通常の組織Data**（部門、職位、所属、組織図、配置表、人数、組織関係）は、組織管理モジュールであるという理由だけでアクセス制御・匿名化・OS隔離の対象にしない。通常のDataと同じ参照・Query・Function連携を基本とする。
- マイナンバー、クレジットカード情報など**保護を要する特定情報**が含まれるときに限り、その**項目・Provider・運用境界**に適した制御を個別に検討する。通常Dataの参照を維持したまま、該当項目だけ必要な管理を適用する。
- 外部Host/別運用Tenantでの実行認可は、通常の組織Data閲覧とは**独立した問題**として扱う。組織図や人数を標準で秘匿しない。
- 実装上の帰結: 本計画の作業単位は、組織Data専用のACL・隔離・匿名化を作らない。保護対象が混在する場合の最小受入票は WU-14 が担当する。

### 2.4 段階と、各段階で「言えること／言えないこと」

| 段階 | 名前 | 言えること | 言えないこと（明示的に未達） |
| --- | --- | --- | --- |
| S0 | 現状確認 | 現行mainの部品と既存ORG-* Expected で何がカバーされているか | 本計画の新規要件の充足 |
| S1 | 基本組織と所属 | Person・OrgUnit・Membership の独立Identity、単一親軸の宣言、read-only list/read | as-of、循環診断、同時点所属1件の強制 |
| S2 | 兼務・複数階層・役職 | 兼務（複数Membership occurrence）、複数軸、Position/Occupancy、法人・出向・雇用の区別 | 軸間の整合（FTE合計等）、Tenant越境の自動許可 |
| S3 | 時点・履歴・遡及訂正 | valid-time と recorded-at の分離、訂正時の occurrence ID 不変、辞令による一括変更 | 半開区間重複の自動診断、大規模改編の原子的公開 |
| S4 | 責任・代理・承認との接続 | 承認候補の導出、委任の構造、自己承認拒否のRule、候補と実行可能性の分離 | 実Grant発行、動的承認Ruleの版固定、進行中案件の再評価 |
| S5 | 三視点モデル・表示表現・編集 | 組織図・配置表・時点別組織表のloss-aware投影、隠れた第二親の保持、逆編集資格 | 専用Graph UI、表示からの書戻し自動化、実Effect |

### 2.5 段階導入のトリガ（反例に基づく判断）

次のどれかを満たすまで、上位段階の構造は導入しない。1つでも満たたら、該当段階の作業単位だけを起票する。

| # | トリガ（観測される反例） | 必要な段階 |
| --- | --- | --- |
| T1 | 同一Personが同時点に2件以上の所属を持つ（P2 が D1 と D2 に所属） | S1（Membership occurrence の複数。Relation Cardinality は `one` のまま） |
| T2 | 指揮命令・原価帰属・プロジェクトで親が異なる | S2（型付きRelationの複数軸） |
| T3 | 親子に循環が現れ、止まらない走査・無限再帰が観測される | S2（単一親軸の循環診断。診断追加そのものは提案） |
| T4 | 役職と所属が同一視され、職位在任から所属が推測される | S2（Position/Occupancy の分離） |
| T5 | 過去時点の組織形態・人数の集計が必要（前年度の旧組織） | S3（valid-time + as-of Query） |
| T6 | 業務確定後に過去分を修正する（遡及訂正） | S3（recorded-at、occurrence ID 不変） |
| T7 | 発効日付きの一括改編（数千〜数万件）を段階取込・差分Previewしたい | S3（辞令Data、原子的公開単位） |
| T8 | 承認者が申請時ではなく**承認実行時**の有効な組織関係から決まる | S4（候補導出 + 実行可能性の分離） |
| T9 | 代理・出向・兼務から操作権限を推測してしまう事故がある | S4（Delegation と Host Grant の分離） |
| T10 | 複数法人・複数Tenantで同じPersonが活動し、越境参照が起こる | S2/S4（LegalEntity・Tenant の非自動対応） |
| T11 | 組織図・配置表・時点別組織表を複数Consumerに同じ意味で配りたい | S5（loss-aware Projection） |
| T12 | 保護対象の特定情報が組織Dataに混在する | WU-14（項目・Provider・運用境界の個別管理） |

## 3. 基本設計

### 3.1 概念の区別と標準構造への対応

| 業務概念 | 標準構造での表現 | Identity | 混同してはならないもの |
| --- | --- | --- | --- |
| 人 | `data Person` | `organization.person_id` | Account・Principal・認証済みsubject。同名・同メールで別人を同一視しない |
| 組織単位（部門等） | `data OrgUnit` | `organization.org_unit_id` | Tenant・Workspace・法人 |
| 法人 | `data LegalEntity` | `organization.legal_entity_id` | OrgUnit。複数OrgUnitを1法人が含み、1法人が複数OrgUnitを持つ |
| 企業集団 | `data EnterpriseGroup`（または LegalEntity 間の型付きRelation） | `organization.enterprise_group_id` | LegalEntity 自身の識別 |
| 雇用関係 | `data Employment`（Person × LegalEntity） | `organization.employment_id` | 所属（Membership）。本務・兼務は Employment の属性ではなく Membership/Employment の組合せで表現する |
| 所属 | `data Membership`（Person × OrgUnit の独立occurrence） | `organization.membership_id` | 雇用関係、役職在任、権限 |
| 出向 | `data Affiliation`（Person × LegalEntity × 方向） | `organization.affiliation_id` | Membership。出向先在籍は Membership として別occurrence |
| 役職・職位 | `data Position`（OrgUnit に属する職位） | `organization.position_id` | 人。役職は「席」、Occupancy は「人が席を占める期間」 |
| 在任 | `data Occupancy`（Person × Position） | `organization.occupancy_id` | Membership。Occupancy は Membership を導出しない |
| 役割（業務上の役割） | `data RoleAssignment`（Person × Role × Scope） | `organization.role_assignment_id` | Position。業務Roleは職位と独立に割り当てられる |
| 責任 | `data Responsibility`（担当事項 × 対象 × 期間） | `organization.responsibility_id` | 役割・権限。責任の存在は実行許可を意味しない |
| 辞令・異動イベント | `data Directive`（複数関係変更を束ねる原因・発効日・出典） | `organization.directive_id` | 現在の所属・権限そのもの |
| 認証済みの操作主体 | Host 側（verified issuer + subject / account link） | 本計画では作らない | Person。メール・氏名一致で別人を同一視しない |
| テナント・Workspace | Tenant/Delivery（#2020）の既存構造 | 本計画では作らない | 法人・部門。法人と一対一とは限らない |

### 3.2 簡潔モデルと高度モデルの比較（反例に基づく判断）

比較対象:
- **簡潔モデル**: `Person`・`OrgUnit`・`Membership`・`Position`・`Occupancy`・`CaseRecord` のみ（現行 `reference/org-headless-workspace`）。
- **高度モデル**: 上記に LegalEntity・Employment・Affiliation・RoleAssignment・Responsibility・Directive と複数軸・二時点を加えたもの。

| # | 業務条件（観測される反例） | 簡潔で足りるか | 高度が必要になる条件 | 追加する標準構造 | 新Core/Engineが要るか |
| --- | --- | --- | --- | --- | --- |
| C1 | 1人=1部門、親子1本、時点不要 | 足りる | — | なし | 不要 |
| C2 | 同時点に2所属（P2: A2 D1 / A3 D2） | 足りる | — | Membership occurrence を2件。Relation Cardinality は `one` のまま | 不要 |
| C3 | 指揮命令/原価/プロジェクトで親が異なる | 不足 | 軸が2本以上になる | OrgUnit に軸別の型付きRelation（`supervisor one`・`matrix_parent many`） | 不要 |
| C4 | 親子の循環（L1→D1→L1）で走査が止まらない | 不足 | 単一親軸で循環が現れる | 単一親軸のみ循環拒否。複数親軸は拒否しない。診断追加は**提案**（現行Validationはinstance循環を見ない） | 診断1種の追加のみ。Engine は不要 |
| C5 | 役職在任から所属を推測してしまう | 不足 | Position と Membership を同一視する実装がある | Occupancy と Membership を別Dataのまま維持（既存） | 不要 |
| C6 | 前年度の旧組織での人数集計 | 不足 | 過去時点の集計が必要 | `validFrom`/`validTo` 属性 + as-of Query | 不要（temporal Engine は作らない） |
| C7 | 5月分を7月に修正する（遡及訂正） | 不足 | 業務確定後の修正が必要 | recorded/known 時間。occurrence ID は不変 | 不要 |
| C8 | 発効日付きの一括改編（数万件） | 不足 | 段階取込・差分Preview・原子的公開が必要 | `Directive` + 差分Preview + 原子的公開単位 | 不要（取込基盤は #1807） |
| C9 | 複数法人・出向で同じPersonが活動 | 不足 | 法人在籍と部門所属を分ける必要がある | LegalEntity・Employment・Affiliation | 不要 |
| C10 | 兼務から他Tenantの閲覧・更新を推測してしまう | 不足 | 越境参照が起こる | LegalEntity/Tenant の非自動対応 + Host Grant の分離 | 不要（#1174/#2499） |
| C11 | 承認者が承認実行時に決まる | 不足 | 申請時と実行時で候補が変わる | 候補導出（read-only Function） + Rule revision | 不要（#1129/#1130） |
| C12 | 代理・成り代わり操作が必要 | 不足 | 代理行为の record が必要 | Delegation Data + Host Grant。成り代わり認証は既定にしない | 不要 |
| C13 | 組織図・配置表・時点別組織表を複数Consumerに配りたい | 不足 | 同一意味を複数表示に渡す | loss-aware Projection（#2497/#1124） | 不要 |
| C14 | HR/IdP から部門・在籍・アカウントを取込む | 不足 | 外部正本との同期が必要 | Source of Record・候補→差分→検証→受理→公開（#1807/#1120） | 不要 |

**判断規則**: 上のどれも満たさない案件では簡潔モデルのまま運用する。C1〜C5 は「新規Dataを増やさずに既存Relation/Cardinalityで足りる」領域であり、ここで新Core Kind を作ることは却下する。C6〜C8 で temporal Engine を、C10〜C12 で認可基盤を、C13 でGraph専用Coreを作ることは却下する。

### 3.3 多重関係（マトリクス・兼務・出向・複数法人・プロジェクト）

- 階層は**単一の `parent` 列ではなく型付きRelationの集合**として宣言する。指揮命令、予算・原価、地理、プロジェクト、法人支配は別のRelationとし、それぞれに `Cardinality` と `Keys` を付ける。
- 単一親を要する軸（例: 人事上の親子）では `relation supervisor to organization.org_unit one` とし、同時点の親が2件・循環は業務Ruleとして拒否する。**ただし現行Validationは instance レベルの循環・重複期間を診断しない**ため、拒否は Rule/Consumer 側の責務となり、その限界をExpectedに固定する。
- 兼務・出向は「別occurrenceを増やす」ことで表現する。Person を統合したり、Relation Cardinality を `many` に変更して Person 側に複数targetを持たせたりしない。
- 到達結果は「到達Node」「使用したRelation occurrence」「到達Path」を区別する。A→B→C と A→C の二経路があっても、同一Cへの到達を一回に集約した表示と、複数経路を保持すべき業務判断は別である。
- 全組織共通 `parent` の設定、全軸への権限継承の自動適用は行わない。

### 3.4 時点（業務有効・記録・評価）

`reference/derivation-temporal` の区別を語彙として再利用し、**一つの `asOf` 引数に畳まない**。

| 時間 | 意味 | 標準構造での置き方 |
| --- | --- | --- |
| valid-time | 業務上有効な期間 | `validFrom`/`validTo` 属性（半開区間 [from,to) を採用する場合は明示） |
| recorded-at | レコードをProviderが記録・更新した時刻 | `knownFrom`/`knownTo` 属性。valid-time と同一視しない |
| evaluated-at | 評価・再評価を行った時刻 | Rule/Query 実行の記録。対象Dataの属性にしない |
| Rule revision | その時点で採用されていた規則版 | Rule の版。現在のRuleを過去データへ遡及適用しない |
| confirmed fact | 過去に実行・送信・確定した事実 | Effect/Receipt（#1174）。履歴解釈の変更で消えない |

- 遡及訂正は**同一 occurrence ID を保つ**。訂正は「訂正後の履歴解釈」を変えるが、過去の実Commit・送信・Receipt は消えず、旧Receiptを現在有効化しない。
- 辞令（`Directive`）は複数の関係変更を束ねる**原因・発効日・出典**として保持する。イベントは現在の所属でも権限でもない。
- 発効前は旧体制、発効後は新体制として、整合した単一revisionを表示する。遅延・部分取込・中断時の半端な状態を正式版として公開しない。

### 3.5 責任・代理・承認・権限の接続

- 契約は常に「業務上どの対象と関係するか」と「実行時その対象へアクセスを許されるか」を分離する。
- 承認候補は「承認実行時の対象業務の有効な組織関係から解決する」ものを Rule として宣言する。候補の割当て（通知/Inbox表示）と、実際の承認可能性（認証・現行権限・職務分掌・本人不可・組織/対象revision）を分ける。
- 異動により承認候補が消えた場合は、未割当て／再解決／管理者引継ぎへ遷移する。失効した担当者による承認は許さない。
- 代理は (a) 読み取り専用の代理閲覧、(b) 明示的な代理行为、(c) 緊急時の限定的権限昇格を分ける。本人への成り代わり認証を設計上の既定としない。実操作者と被代理者、Grantとその権限源、対象Action、対象Tenant/資源、期限、事由、失効・監査を残す。再委任の連鎖は既定で許可しない（許容する場合は深さ・同一性・循環・委任権限の根拠を明示）。
- 委任による利益相反／自己承認は拒否する。ルールの版固定・進行中案件への再適用は業務Owner判断とする。
- 複数法人所属は自動的な複数Tenantアクセス権の取得を意味しない。組織Modelの属性から現行Principal・Tenant・Grantを作ったことにしない（#1043/#1118/#1174）。

### 3.6 三視点とProjection

- 同じ Accepted Subject から Data・Function・Business の三視点と、組織図（軸・時点指定）、配置表（Person×Unit）、時点別組織表（as-of）、異動前後の差分表、「なぜ承認できる/できないか」の説明を導出する。
- 画面の配置・色・見せ方は **Projection metadata** であり、Accepted の関係意味や権限の第二正本にしない。
- Graph→Tree への投影では、表示されない第二の親を保持し、書戻しで消去しない。表示上の Preference（本務の優先）は、Owner が明示した場合だけ使用し、ない場合は**選択不能/要確認**として両関係を保持する。表示用Preferenceが本務・指揮命令・Grant の意味を持つとは限らない。
- 計数のgrain（人数・FTE・件数）を明示し、二経路の同一Factの二重計上を禁止する。
- 編集は「一意なSource対象を持つ型付き変更候補」だけを既存Preview/Acceptanceへ渡す。集計・非可逆表示からの自動書戻しは認めない。

### 3.7 段階別の具体（Identity / Relation / Cardinality / Time / Validation / Query / 表示形式）

#### S1 基本組織と所属

| 観点 | 具体 |
| --- | --- |
| Identity | `organization.person_id`・`organization.org_unit_id`・`organization.membership_id`。Membership は Person×OrgUnit の**独立した関係occurrence**。read のkeyは `membership_id`（Person ID でも OrgUnit ID でも行順でもない） |
| Relation | `Membership.person → Person one`、`Membership.unit → OrgUnit one`、`OrgUnit.supervisor → OrgUnit one`（単一親軸、任意）。`CaseRecord.handlingUnit → OrgUnit one` は構造関係のみ |
| Cardinality | Membership は Person 側 `one`・OrgUnit 側 `one`。複数所属は occurrence を増やす。Relation Cardinality を `many` に変更して Person 側に複数targetを持たせない |
| Time | `validFrom`・`knownFrom` は普通の authored 属性。存在 alone では bitemporal 意味・履歴移行・as-of評価を作らない |
| Validation | `positive`（数量）、`relation-unique`（Relation target の一意性）。同時点所属1件・単一親・循環は未検査 |
| Query | `org_view.list.organization.membership`・`org_view.read.organization.membership`（read-only）。as-of 引数は未サポート |
| 表示形式 | 一覧表（Table）、部門メンバー表。組織図は S5 |

#### S2 兼務・複数階層・役職

| 観点 | 具体 |
| --- | --- |
| Identity | + `organization.legal_entity_id`・`employment_id`・`affiliation_id`・`project_participation_id`・`role_assignment_id`。Position/Occupancy は S1 から維持 |
| Relation | `Employment.person → Person one`・`Employment.entity → LegalEntity one`、`Affiliation.person → Person one`・`Affiliation.entity → LegalEntity one`・`Affiliation.sourceEntity → LegalEntity one`、`Position.unit → OrgUnit one`、`Occupancy.person → Person one`・`Occupancy.position → Position one`、`OrgUnit.matrixParent → OrgUnit many`、`ProjectParticipation.person → Person one`・`.project → OrgUnit one` |
| Cardinality | 軸ごとに `one`/`many` を明示。単一親軸は `one`、マトリクス軸は `many`。兼務は Membership occurrence の複数として表現 |
| Time | 各関係occurrenceに valid 期間。Occupancy は在任期間（半開区間を採る場合は明示） |
| Validation | relation-unique で表現できる範囲のみ。出向元/出向先の同時有効性、FTE合計、職位の重複在任は業務Rule（Owner受理が必要） |
| Query | 軸指定・時点指定の一覧。二経路の到達は Node/Path を区別して返す |
| 表示形式 | 配置表（Person×Unit）、軸指定の組織図、兼務一覧 |

#### S3 時点・履歴・遡及訂正

| 観点 | 具体 |
| --- | --- |
| Identity | occurrence ID は訂正を通して不変（`A2` のまま）。版は `MembershipVersion` 等の別Data、または属性組で表現 |
| Relation | S2 と同じ。履歴は Relation 定義を変えず occurrence と属性で持つ |
| Cardinality | S2 と同じ。期間重複の可否は Cardinality ではなく Rule |
| Time | valid-time・recorded-at・evaluated-at・Rule revision・confirmed fact を別物として保持。辞令（`Directive`）は原因・発効日・出典 |
| Validation | 半開区間の重複、同一occurrenceの同時有効版が2件、未来発効の混在は**現行Validationでは表現できない**。表現案は提案として分離し、Owner判断に委ねる |
| Query | `EmbeddedSQL.Inputs` に as-of（valid）と known を**別引数**として明示。analyzer の `Lineage`・`Coverage` で非出力依存を保持（ORG-QD-01〜12 の再利用） |
| 表示形式 | 時点別組織表、異動前後の差分表、対象件数・保留件数 |

#### S4 責任・代理・承認との接続

| 観点 | 具体 |
| --- | --- |
| Identity | + `organization.responsibility_id`・`delegation_id`・`approval_rule_revision`。Principal・Grant・Tenant は Host 側（#1043/#1118/#1174/#2020）で本計画は作らない |
| Relation | `Responsibility.holder → Person one`・`.subject → Position または RoleAssignment one`・`.scope → ...`、`Delegation.grantor → Person one`・`.agent → Person one`・`.action → ...`・`.scope → ...`。承認候補は Relation からの**導出** |
| Cardinality | 委任の連鎖は深さを明示（既定は 0 = 再委任不可）。候補は 0件・多数・未確認を区別する |
| Time | 委任の有効期間、Rule revision、対象Case revision、副作用発生時点。失効直前・途中・再試行を別扱い |
| Validation | 自己承認拒否・利益相反・職務分掌は Rule（Owner受理）。現行Validationは式を持たないため、Rule/Procedure 側で保持する |
| Query | 候補解決は read-only。実行可能性（認証・現行権限・対象revision）は #1174/#2499 で別判定 |
| 表示形式 | 「なぜこの人がこの申請を承認できる/できないか」の説明。根拠（rule revision / org revision / as-of / relation path）と**現在Authorityを別欄**で表示 |

#### S5 三視点モデル・表示表現・編集

| 観点 | 具体 |
| --- | --- |
| Identity | Projection occurrence は元 Subject/Relation occurrence とは別IDを持ち、元IDを保持する。表示上の Preference は明示がない限り「選択不能」 |
| Relation | Graph の edge occurrence は Relation occurrence に束縛し、edge ID・方向・多重度を保持。Tree化では隠れた第二親を保持 |
| Cardinality | 集計grain（人数・FTE・件数）を明示。二経路の同一Factを二重計上しない。 JOIN 増幅と実対象を区別 |
| Time | 図のSnapshot時点・Source revision・表示 epoch・Consumer鮮度を別物として保持 |
| Validation | 逆写像の一意性・非単射の検出（#2497 MP-01〜12）。部分取得・非開示・未解決セルを「空」としない |
| Query | read-only 投影 + coverage・unknown・未確定の保持。実Readは #2586/#2622 |
| 表示形式 | 軸・時点指定の組織図、配置表、時点別組織表、差分表、説明文。#2497/#1124/#1125/#2516 の既存Contractに接続する |

## 4. 独立Expected

独立Expectedは**実装の出力から生成しない**。既存の ORG-* を再利用できるものは再利用し、新規のみを作る。

| ID | 名称 | 入力（fixture） | 期待 | 禁止する推定 | 新規/再利用 | 担当Owner |
| --- | --- | --- | --- | --- | --- | --- |
| ORG-EXP-01 | 単純な所属 | `reference/org-headless-workspace` の Person/OrgUnit/Membership | list/read が Membership occurrence を返し、read のkeyは `organization.membership.membership_id`。Person ID・OrgUnit ID・表示行順をkeyにしない | 1表示行=1業務対象、Person ID を所属のkeyにすること | 再利用（`TestOrgHeadlessAuthoringToOperations`）+ 新規の手書き期待値JSON | #2386 |
| ORG-EXP-02 | 兼務（同時点2所属） | `reference/org-semantic-views/source-fixture.json` の P2/A2/A3 | 同一Personの2件のMembership occurrence が保持され、distinct person 数は3のまま。人数と所属件数を混同しない | 二経路を二人に数える、occurrence を1件へ統合する | 再利用（ORG-TM-01） | #2510 |
| ORG-EXP-03 | 複数上位組織（軸別） | 新規fixture（WU-02）の `supervisor`（one）と `matrixParent`（many） | 二軸の親が異なることを保持。H→D1→L2 と H→S→L2 なら「到達Node L2 は1、Path は2」と数える | 最初の出現を暗黙の「主たる親」に決める、全組織共通parentを設定する | 新規 | #2386 |
| ORG-EXP-04 | 異動（valid-time切替） | `reference/org-semantic-views` の membershipVersions | 旧所属の終了と新所属の開始が同時点に重ならず、異動前後で同一Personの所属件数が正しく変わる | 重複期間を許可する、終了日を含む区間で二重計上する | 再利用（ORG-TM-02・verify-change） | #2510 |
| ORG-EXP-05 | 遡及訂正 | 同上（known を 1日後へ） | 訂正は同一 occurrence ID を保つ。訂正前後で valid 期間は変わり、known は独立に変わる | occurrence ID を新規発行する、過去の確定Effectを消す、旧Receiptを現在有効化する | 再利用（ORG-TM-02・ORG-KH-*） | #2510 |
| ORG-EXP-06 | 関係の循環 | 新規fixture（WU-02）の L1→D1→L1 と、複数親軸の二経路 | 単一親軸の循環は拒否/診断対象として識別できる。複数親軸の二経路は構造として保持する | 複数親軸まで一律に拒否する、循環を「深さ制限に達した」と末端到達と誤認する | **新規**（現行Validationは instance 循環を診断しないため、負例として固定） | #2386 |
| ORG-EXP-07 | 部分取得 | `reference/org-semantic-views` の acquisition/field-observation fixture | 取得0件・partial は「不存在」と確定しない。必要属性の未取得は unknown として残す | partial 0件を未所属確定にする、全行取得で全属性観測済みとする | 再利用（ORG-FO・verify-acquisition・verify-field-observation） | #2510 |
| ORG-EXP-08 | 代理権限の誤推定 | 新規fixture（WU-10）の P2（A法人本務 + B法人兼務）と Delegation | 兼務・出向・組織パスから Tenant の read/write Grant を推測しない。委任の深さ・循環・取消し・自己承認拒否を保持する | 組織関係から現在Grantを生成する、成り代わり認証を既定とする、失効した担当者の承認を許す | **新規**（ORG-APP-01〜09 は計画のみ） | #2386 |

## 5. 実装計画（作業単位）

共通の変更境界（全WUに適用）:

- `main` へ直接変更しない。作業は Issue 単位のブランチ/worktreeで分離する。
- `reference/workspace` と `reference/org-headless-workspace` の Source は変更しない。新しいfixtureは**別ディレクトリ**に作る（`reference/org-staged-axis/` 等）。既存fixtureの変更が必要になった場合は `reference-shared-acceptance` 経路（Decision記録、`-allow-reference-baseline`）が必要なので停止する。
- 製品コード（`internal/`・`cmd/`）は**変更しない**。必要なのは read-only な Go characterization test と synthetic Node oracle と docs のみ。
- 検証は `gofmt -l`・`go vet`・該当packageの `go test -count=1`・`go run ./cmd/tei validate -workspace <fixture>`・`node --test`。実行できない検証は制約として明示する。
- 期待値は手書きの別ファイルに置き、実装出力から生成しない。

### WU-01 基本組織と所属の受入票を独立Expectedで固定する（S1）

| 項目 | 内容 |
| --- | --- |
| 目的 | 簡潔モデルで足りる領域（C1・C2・C5）を、独立Expected付きで受入票に固定する |
| 対象ファイル | 新規 `internal/operations/org_staged_basic_characterization_test.go`（Go read-only test）、新規 `reference/org-staged-basic/expected-basic.json`（手書き期待値）、新規 `reference/org-staged-basic/verify-staged-basic.mjs`（Node oracle）。`reference/org-headless-workspace/` は読み取りのみ |
| 既存API | `operations.SurfaceWorkspace`・`operations.DeriveOperationSurface`・`workspace.Load`・`validation.Validate`・`internal/semantic` の型 |
| 入力 | `reference/org-headless-workspace` の現行Source（Person・OrgUnit・Membership・Position・Occupancy・CaseRecord） |
| 出力 | 期待値JSON、テスト結果、受入票（本設計 §4 ORG-EXP-01 の確定版） |
| 受入テスト | (a) list/read が Membership occurrence を返す、(b) read のkeyが `organization.membership.membership_id`、(c) Person ID・OrgUnit ID・行順をkeyにしない、(d) Occupancy から Membership を導出しない、(e) CaseRecord の構造Relationから read/write/Policy を導出しない、(f) 既存 `TestOrgHeadless*` を壊さない |
| 依存 | なし |
| 完了条件 | 手書きExpectedを実装が満たし、`tei-org-headless-native-ci.yml`・`tei-org-semantic-reference-node.yml` の該当経路でPASS |
| 停止条件 | `reference/org-headless-workspace` や `reference/workspace` のSource変更が必要になった場合。実DB・Host Grant・Owner受理は範囲外 |
| 要求ロール | `implementer` |

### WU-02 型付き複数軸の宣言差分を最小fixtureで表現する（S1→S2）

| 項目 | 内容 |
| --- | --- |
| 目的 | 単一親軸と複数親軸の宣言差分を、標準Relation/Cardinalityで authorable にする |
| 対象ファイル | 新規 `reference/org-staged-axis/tei.workspace.yaml`・`semantic/organization-axes/model.tei`・`applications/org-axes-view/application.tei`・`databases/org-axes-store/persistence/membership.tei` |
| 既存API | `workspace.Load`・`SurfaceWorkspace`・`validation.Validate`・`internal/query/analyzer` |
| 入力 | 本設計 §3.3・§3.7 S2 の Relation/Cardinality 定義 |
| 出力 | fixture、宣言済みRelation/Cardinality/Keys の一覧、期待診断表 |
| 受入テスト | (a) `relation supervisor to organization.org_unit one` と `relation matrixParent to organization.org_unit many` が authorable、(b) cardinality/Keys が検証される、(c) `CaseRecord.handlingUnit` と同様、構造Relationから操作・権限が導出されない、(d) instance 循環が未診断であることを負例として明示する（`SEMANTIC_TYPE_ALIAS_CYCLE` は型エイリアスの循環のみ） |
| 依存 | WU-01 |
| 完了条件 | 新規fixtureが `tei validate`・`tei operations` を通り、期待診断表との差分が0 |
| 停止条件 | Validation の新kind（temporal-overlap・relation-cycle 等）が必要になった場合。公開契約の変更はOwner判断 |
| 要求ロール | `implementer` |

### WU-03 三視点・組織図・配置表・時点別組織表の読み取りFit/Gap（S1/S5）

| 項目 | 内容 |
| --- | --- |
| 目的 | 既存CLI/ModelMapの読み取り口で組織の三視点がどこまで得られるかを、新コマンドを足さずに明らかにする |
| 対象ファイル | 新規 `internal/operations/org_staged_projection_fitgap_test.go`、新規 `plan/design/` への追記ではなく本計画 §6 の更新 |
| 既存API | `operations.MapWorkspace`・`operations.MapWorld`・`cmd/tei` の `model catalog/locate/overview`・`inspect`・`trace`・`workspace-files`・`source-index`・`validate` |
| 入力 | WU-02 のfixture |
| 出力 | Fit/Gap 一覧（得られる/得られない/unknown）、必要最小の差分 |
| 受入テスト | (a) 同一Source版から Data/Function/Business の読み取りが同じSubjectへ navigate できる、(b) 組織図・配置表・時点別組織表の専用投影が**存在しない**ことを unknown として残す、(c) 部分取得・非開示・未解決セルを「空」としない |
| 依存 | WU-02 |
| 完了条件 | Fit/Gap 一覧が実CLI/Readerの実出力に対応づけられ、推測で「未提供」と断定していない |
| 停止条件 | 新CLIコマンド・新MCP schema・新Rendererが必要になった場合（#3262/#1124/#2497 Owner） |
| 要求ロール | `implementer` |

### WU-04 兼務・出向・複数法人の語彙分離（S2）

| 項目 | 内容 |
| --- | --- |
| 目的 | LegalEntity・Employment・Affiliation を Membership と分離し、越境推測を防ぐ |
| 対象ファイル | `reference/org-staged-axis/semantic/organization-axes/model.tei`（WU-02 の延長）、新規 `internal/operations/org_staged_entity_separation_test.go` |
| 既存API | `workspace.Load`・`validation.Validate`・`internal/semantic` |
| 入力 | 本設計 §3.1 の概念表 |
| 出力 | 語彙分離の期待表、負例（法人在籍からTenant権限を推測しない） |
| 受入テスト | (a) 同一Personが2法人に所属しても Tenant/Grant が自動取得されない、(b) Employment と Membership と Affiliation が別Dataとして保持される、(c) 出向先在籍が Membership として別occurrenceになる、(d) Person の同定は連携元間で自動確定しない（SCIMはIdentity交換adapter候補） |
| 依存 | WU-02 |
| 完了条件 | 語彙分離がfixtureとGo testで確認され、誤推定が負例として失敗する |
| 停止条件 | Tenant/Delivery（#2020）や認証（#1043/#1118）の変更が必要になった場合 |
| 要求ロール | `implementer` |

### WU-05 役職・職位・在任・役割・責任の区別（S2）

| 項目 | 内容 |
| --- | --- |
| 目的 | Position・Occupancy・RoleAssignment・Responsibility の四つの非同一性を固定する |
| 対象ファイル | `reference/org-staged-axis/semantic/organization-axes/model.tei`（延長）、新規 `internal/operations/org_staged_role_separation_test.go` |
| 既存API | `workspace.Load`・`validation.Validate` |
| 入力 | 本設計 §3.1 |
| 出力 | 四つの非同一性の期待表 |
| 受入テスト | (a) Occupancy ≠ Membership ≠ RoleAssignment ≠ Responsibility、(b) 職位在任から所属を導出しない、(c) 役割の割当てから権限を導出しない、(d) 責任の存在が実行許可を意味しない |
| 依存 | WU-04 |
| 完了条件 | 四つの非同一性が test とfixtureで確認される |
| 停止条件 | 業務Rule（職務分掌・利益相反）の正式受理が必要になった場合（Owner判断） |
| 要求ロール | `implementer` |

### WU-06 二時点と履歴occurrence（S3）

| 項目 | 内容 |
| --- | --- |
| 目的 | valid-time と recorded-at を別物として持ち、訂正で occurrence ID を不変にする |
| 対象ファイル | `reference/org-staged-axis/semantic/organization-axes/model.tei`（延長）、新規 `reference/org-staged-axis/README.md`、新規 `internal/operations/org_staged_temporal_test.go` |
| 既存API | `reference/derivation-temporal` の時間区別（valid-time・recorded-at・evaluated-at・Rule revision・confirmed fact）、`workspace.Load` |
| 入力 | 本設計 §3.4 |
| 出力 | 二時点の期待表、訂正時のID不変の負例 |
| 受入テスト | (a) 同一occurrence ID で訂正前後の版が区別できる、(b) known を1日後にずらすと選ばれる版が変わる、(c) 有効期間が重なる2版が同時に選ばれない、(d) 過去の確定Effect/Receiptが履歴解釈の変更で消えない |
| 依存 | WU-02 |
| 完了条件 | 二時点の挙動が synthetic oracle と Go test の両方で確認される |
| 停止条件 | bitemporal の物理DB方式・Provider pushdown・時点Owner受理が必要になった場合 |
| 要求ロール | `implementer` |

### WU-07 as-of Query の独立Expected（非出力依存を含む）（S3）

| 項目 | 内容 |
| --- | --- |
| 目的 | as-of をSQLと入力引数で表現し、非出力依存を Lineage/Coverage に残す |
| 対象ファイル | 新規 `reference/org-staged-axis/applications/org-axes-view/queries/as_of_membership.tei`、新規 `internal/operations/org_staged_asof_query_test.go` |
| 既存API | `internal/query` の `EmbeddedSQL`・`analyzer`（`Scopes`・`Fields`・`Lineage`・`Coverage`）、`internal/validation` のQuery検証 |
| 入力 | WU-06 のfixture |
| 出力 | as-of 引数の契約、非出力依存の期待表 |
| 受入テスト | (a) `Inputs` に valid と known が別引数として現れる、(b) WHERE の非表示条件（配賦しきい値等）で結果が変わることを Lineage が保持する、(c) ORDER 変更で並び順・ページ表示の再利用資格が変わる、(d) EXISTS/NOT EXISTS の変化が別結果になる（ORG-QD-01〜12 の再利用） |
| 依存 | WU-06 |
| 完了条件 | analyzer の出力が期待表と一致する |
| 停止条件 | SQL実行・実Provider Rowset・Row Scope 強制が必要になった場合（#2586/#2622/#1656） |
| 要求ロール | `implementer` |

### WU-08 遡及訂正と辞令イベント（S3）

| 項目 | 内容 |
| --- | --- |
| 目的 | 辞令を「複数変更を束ねる原因・発効日・出典」として持ち、イベント≠現在所属を固定する |
| 対象ファイル | `reference/org-staged-axis/semantic/organization-axes/model.tei`（延長、`Directive`）、新規 `internal/operations/org_staged_directive_test.go` |
| 既存API | `workspace.Load`・`validation.Validate`・`internal/operations` |
| 入力 | 本設計 §3.4 |
| 出力 | 辞令の期待表、異動前後の差分表の構成 |
| 受入テスト | (a) Directive が複数の関係変更を束ねる、(b) Directive 自体が現在の所属でも権限でもない、(c) 発効前は旧体制・発効後は新体制として一貫したrevisionを表示する、(d) 部分取込・中断時の半端な状態を正式版として公開しない |
| 依存 | WU-07 |
| 完了条件 | 差分表の構成と禁否推定が test で確認される |
| 停止条件 | 大規模改編の原子的公開・性能評価（データ規模・投入時間・差分生成・as-of検索・影響分析）が必要になった場合 |
| 要求ロール | `implementer` |

### WU-09 承認候補の動的解決と実行可能性の分離（S4）

| 項目 | 内容 |
| --- | --- |
| 目的 | 「承認実行時の有効な組織関係からの候補解決」と「実際の承認可能性」を分ける |
| 対象ファイル | 新規 `internal/operations/org_staged_approval_candidate_test.go`、新規 `reference/org-staged-axis/README.md` への節 |
| 既存API | `internal/operations`・`internal/semantic` の Capability/EffectRule/RunPlan、#1129/#1130 の宣言 |
| 入力 | 本設計 §3.5 |
| 出力 | 候補解決と実行可能性の期待表 |
| 受入テスト | (a) 候補解決は read-only の導出、(b) 候補0件・多数・未確認・循環・秘匿関係を区別する、(c) 異動で候補が消えた場合は未割当て/再解決/管理者引継ぎへ遷移する、(d) 失効した担当者の承認は拒否する、(e) 候補の割当て（通知）と承認可能性（認証・現行権限・職務分掌・対象revision）を混同しない |
| 依存 | WU-08 |
| 完了条件 | 候補と実行可能性の別資格が test で確認される |
| 停止条件 | 動的承認Ruleの版固定・進行中案件への再適用が必要になった場合（業務Owner判断）、または現在Grant/Effect（#1174/#2499）の実接続が必要になった場合 |
| 要求ロール | `implementer` |

### WU-10 代理権限の誤推定の負例（S4・ORG-EXP-08）

| 項目 | 内容 |
| --- | --- |
| 目的 | 兼務・出向・組織パスから操作権限を推測しないことを負例で固定する |
| 対象ファイル | 新規 `reference/org-staged-axis/scenarios/delegation-misinference.json`（手書き入力と期待）、新規 `internal/operations/org_staged_delegation_test.go` |
| 既存API | `internal/semantic`・`internal/operations`・#1174/#2499 の読み取りBoundaryの説明 |
| 入力 | P2（A法人本務・B法人兼務）、出向者、代理Grant、循環する委任 |
| 出力 | 誤推定の負例一覧と期待結果 |
| 受入テスト | (a) B法人兼務から B Tenant の read/write Grant を推測しない、(b) 成り代わり認証を既定にしない、(c) 委任の連鎖は深さ0（再委任不可）が既定、(d) 循環する委任を拒否する、(e) 自己承認・利益相反を拒否する、(f) 失効・対象すり替え・Scope違いを負例にする |
| 依存 | WU-09 |
| 完了条件 | 負例がすべて失敗として識別される |
| 停止条件 | 緊急時の権限昇格の具体条件・許容される代理操作の種類が必要になった場合（Owner判断） |
| 要求ロール | `implementer` |

### WU-11 Workflow接続の最小（S4）

| 項目 | 内容 |
| --- | --- |
| 目的 | 業務手順・承認を既存の Action/Pattern/Capability/EffectRule/RunPlan で接続し、専用Workflow Engineを作らない |
| 対象ファイル | 新規 `internal/operations/org_staged_workflow_boundary_test.go` |
| 既存API | `internal/semantic` の `Capability`・`EffectRule`・`RunPlan`、`internal/validation` の effect-rule/run-plan 検証、`internal/operations` |
| 入力 | 本設計 §3.5・§2.2 非目標 |
| 出力 | 接続点の一覧と非接続点の明示 |
| 受入テスト | (a) 手順からWorkflow/Actionへの接続が既存宣言で表現できる、(b) BPMN/DMN/CMMN は比較候補に留まる、(c) 新しいStatus Enum・Workflow Host・Proof Storeを作らない、(d) 表示変化やSQL JOINの重複を発火・完了の証拠にしない |
| 依存 | WU-09 |
| 完了条件 | 接続点と非接続点が test とdocsで確認される |
| 停止条件 | Workflow/BPMNランタイムの採用判断が必要になった場合（別Issue） |
| 要求ロール | `implementer` |

### WU-12 三視点Projectionのloss表と逆編集資格（S5）

| 項目 | 内容 |
| --- | --- |
| 目的 | 組織図・配置表・時点別組織表の投影損失と逆編集資格を固定する |
| 対象ファイル | 新規 `reference/org-staged-axis/projection-loss.json`（手書き期待）、新規 `internal/operations/org_staged_projection_loss_test.go` |
| 既存API | `internal/operations/model_map.go`、#2497 の MP-01〜12、#1124/#1125 のReader |
| 入力 | WU-03 のFit/Gap、本設計 §3.6 |
| 出力 | loss表、逆編集資格の判定表 |
| 受入テスト | (a) Graph→Tree で隠れた第二親を保持し書戻しで消さない、(b) 表示Preferenceがなければ選択不能/要確認として両関係を保持する、(c) 集計grainを明示し二経路の同一Factを二重計上しない、(d) 部分取得・非開示・未解決セルを「空」としない、(e) 非可逆変更はread-onlyとする |
| 依存 | WU-03 |
| 完了条件 | loss表と逆編集資格が test で確認される |
| 停止条件 | 専用Renderer・新Graph Core・SHO納品が必要になった場合（#2497/#1124/#1803 Owner） |
| 要求ロール | `implementer` |

### WU-13 編集セッションと一体保存の境界（S5）

| 項目 | 内容 |
| --- | --- |
| 目的 | 図上の選択・編集候補と、Sourceへの一意な書戻しを分ける |
| 対象ファイル | 新規 `internal/operations/org_staged_edit_boundary_test.go` |
| 既存API | `internal/sourceacceptance`・`internal/modelref`・`internal/operations` のPreview/Acceptance経路 |
| 入力 | WU-12 のloss表 |
| 出力 | 編集四分類（read-only・候補Patch・明示Action・却下）の期待表 |
| 受入テスト | (a) 一意なSource対象を持つ型付き変更候補だけがPreview/Acceptanceへ渡る、(b) 集計・非可逆表示からの自動書戻しを認めない、(c) 古いRevision・多重出現・曖昧名はread-only、(d) 図のドラッグ・選択・直接リンク操作が業務Updateと誤認されない |
| 依存 | WU-12 |
| 完了条件 | 編集四分類が test で確認される |
| 停止条件 | 実Host Effect・Owner受理・Studio実編集が必要になった場合 |
| 要求ロール | `implementer` |

### WU-14 保護対象の区別と例外管理（横断）

| 項目 | 内容 |
| --- | --- |
| 目的 | 通常の組織Dataと保護対象の特定情報を区別し、一律の制御を課さないことを受入票にする |
| 対象ファイル | 新規 `plan/design/ORGANIZATION_DATA_PROTECTION_BOUNDARY_2026-11.md`（または本計画への節）、新規 `reference/org-staged-axis/README.md` への節 |
| 既存API | 既存の Provider/運用境界・#2510 Owner方針・#2020 Tenant/Delivery |
| 入力 | 本設計 §2.3 |
| 出力 | 通常Data/保護対象/外部Host実行認可の三区分の受入票 |
| 受入テスト | (a) 部門・職位・所属・組織図・配置表・人数が特別な機密扱いにならない、(b) 保護対象が混在する場合のみ項目・Provider・運用境界で個別管理する、(c) 組織図や人数を標準で秘匿しない、(d) 外部Host/別Tenantの実行認可を通常Data閲覧と混同しない |
| 依存 | なし（並列可） |
| 完了条件 | 三区分が受入票として明文化され、新ACL・OS隔離・匿名化を作らないことが確認される |
| 停止条件 | 実際の保護対象情報がリポジトリに入る必要が生じた場合（秘密情報をIssue/PR/ログへ入れない） |
| 要求ロール | `implementer` |

## 6. 実装順・並列度

| 順序 | 作業単位 | 段階 | 依存 | 並列可否 |
| --- | --- | --- | --- | --- |
| 1 | WU-01 | S1 | なし | 単独開始可 |
| 2 | WU-02 | S1→S2 | WU-01 | 逐次 |
| 3 | WU-03 | S1/S5 | WU-02 | WU-04・WU-06 と並列可 |
| 4 | WU-04 | S2 | WU-02 | WU-03・WU-06 と並列可 |
| 5 | WU-05 | S2 | WU-04 | 逐次 |
| 6 | WU-06 | S3 | WU-02 | WU-03・WU-04 と並列可 |
| 7 | WU-07 | S3 | WU-06 | 逐次 |
| 8 | WU-08 | S3 | WU-07 | 逐次 |
| 9 | WU-09 | S4 | WU-08 | 逐次 |
| 10 | WU-10 | S4 | WU-09 | 逐次 |
| 11 | WU-11 | S4 | WU-09 | WU-10 と並列可 |
| 12 | WU-12 | S5 | WU-03 | WU-05 と並列可 |
| 13 | WU-13 | S5 | WU-12 | 逐次 |
| — | WU-14 | 横断 | なし | いつでも並列可 |

**順序の理由**: S1 の受入票（WU-01）がないと、上位段階の追加が「既存の簡潔モデルを壊しているのか、拡張しているのか」を判別できない。したがって WU-01 を最初の作業単位とし、以降は段階順に進める。実Read縦断（#2586 HJ-09）と組織Caseの実Provider接続は**本計画の範囲外**であり、並行して最優先される。

## 7. 未決事項（Owner判断）

1. **時点の扱い**: 履歴が必要な関係・監査上の遡及訂正・発効前公開を、業務ドメイン別にどう選ぶか。
2. **兼務と権限**: 本務の業務基準、出向者のAuthority根拠、階層継承の禁止/許可の範囲。
3. **異動前後の案件**: 進行中の申請を旧承認経路で維持するか、新承認経路で再評価するか（案件/step別）。
4. **基幹連携**: 個人・所属・組織名称・役職の属性別Source of Record、同期契約、訂正手順。
5. **代理操作**: 許容される操作の種類と緊急権限昇格の具体条件。
6. **循環・重複期間の診断**: 現行Validationに新kindを追加するか、Rule/Consumer側に委ねるか（公開契約の変更を伴う）。
7. **表示Preference**: 本務の優先を誰が宣言し、どこに記録するか。
8. **保護対象の具体**: 保護を要する特定情報の項目一覧と、適用するProvider/運用境界。

## 8. 採否記録と撤回線

- 各WUの結果は、独立Expected・実測・未実行を**別欄**で記録する。合成PASS・局所Go PASS・CI成功は、実Provider・Host・Owner受理に昇格しない。
- 効果が確認できない抽象は撤回する。撤回線: (a) 新規Data/Relationを増やしても既存の簡潔モデルの受入（WU-01）を壊す、(b) 独立Expectedが実装の出力から生成されている、(c) 組織関係からGrant/Effectを推測する実装が必要になる。いずれかで該当WUは停止し、Issueへ論点を返す。
- 未採用の案（Validation新kind、専用Renderer、Workflowランタイム、temporal Engine、認可基盤）は**提案**として本計画に残し、採用判断はADR/Issueで行う。

## 9. 関連

- 親Issue: #2386（組織・所属・時点の標準意味契約）。
- Corpus・独立Conformance: #2510。Projection: #2497。一般メタ: #1126/#2495/#1806。
- 設計の正本: [組織関係・時点・統制の標準意味モデル](../design/ORGANIZATION_RELATION_GOVERNANCE_MODEL_2026-10.md) と [開発全体計画](../MASTER_PLAN.md)。本計画はその**実装作業単位**を定義する資料であり、意味の正本を書換えない。
- 既存fixture: `reference/org-headless-workspace/`・`reference/org-semantic-views/`・`reference/derivation-temporal/`。

## 10. 2026-11-12 横断統合：組織責任・実行・三視点・知識Consumerを一案件で接合する（採用前計画）

### 10.1 今回の基線と、実装済みの意味を過大に扱わない規則

- 本節は #2386 の**既存計画を実行可能な受入条件へ精緻化する追補**であり、新しい意味正本・Core Kind・Operation/Policy/Workflow Runtimeを作らない。業務意味の採否はOwnerが行う。
- 2026-11-12 GitHub読取時点：段階計画 PR #4342、S1〜S4中心のGoテスト/fixture PR #4399、組織責任Cutoverのdocs PR #3741、SUI/CSW Affinity境界のdocs PR #4256はmainへ統合済み。§1.2に列挙された「未統合Draft」は**当時の履歴**であり、着手時には各PRの現在のstateとmainを再確認する。
- PR #4399の新規テストは有用な**構造・非自動権限化のcharacterization**。例：org_staged_temporal_test.goはWorkspace Load、org_staged_asof_query_test.goはDialect/Inputsの宣言、org_staged_approval_candidate_test.goはDataと非Effectを検査する。実二時点結果差、実Query/Lineage呼出し、動的候補解決、Directiveの一体公開、委任の各シナリオ期待照合まで実行した証拠ではない。差分を#4490で管理する。
- TEI→SHO文書Draft/再投影の限定実装 PR #4317/#4467 はmain統合済み。ただし組織C1の実Consumer接続ではない。高度な固定対象指示の計画 PR #4479 はDraft。既存実装／設計候補／実Provider・Hostの現在証拠は別資格とする。
- **共通失敗条件**：宣言・表示・局所PASS・SHO保存・AI応答・GitHub CI成功から、Owner意味受理、実Read適合、現行Grant、実Effect、相手側ACKを推認した時点で不合格。

### 10.2 単一の責任グラフを既存Data/Relation/Ruleから導出する

組織上の関係と業務の責任割当は区別し、案内用RACI等は用途別Projectionとする。固定のRACI列やCaseAssignment Core型は導入しない。

| 事実・候補 | source / occurrence | 依存する解釈 | **導出してはいけないもの** |
| --- | --- | --- | --- |
| Person P1がUnit D1へ所属 | Membership A1（独立occurrence、valid/known） | 所属・兼務・出向の用途別Rule | Case C1の担当、Positionの在任、Tenant Grant |
| P1がPosition Z1に在任 | Occupancy O1 | 職位軸の関係 | Membershipや承認可否の自動確定 |
| Case C1の取扱部門がD1 | CaseRecord.handlingUnit R1 | 案件責任のBusiness Rule候補 | D1の全員の更新権限、本人P1の担当 |
| P1がC1の決裁候補 | Rule Rrev/Responsibility occurrenceとread context | 当該時点・業務範囲・対象Caseに限る候補 | 現行Principal・Grant・確定Effect |
| C1の責任をD1からD2へ移管 | 明示されたCase責任変更 + 旧/新版 | Handoff/受領要否・単独/共同責任の業務Rule | Membership異動や通知だけからのACK |
| 実際の承認・実行 | Hostの検証済み主体/Grant + Effect Fence + Receipt | current Authority・対象・実装Artifact | 単なるCandidate、旧Receipt、別Tenantでの許可 |

- Relation occurrence、Entity、役割、時点、到達Path、Graph上の表示Edgeを混同しない。D1→D2を経由して同じ部門に二経路で到達しても、人数・案件数・責任件数は**集計grain**（Person/Case/Assignment/Path）ごとに計算する。
- 取扱部門変更と責任者変更の連動は**明示的なBusiness Ruleが採用された場合だけ**行う。決裁が旧部門で継続するか、現在部門で再評価するかは業務Ownerの選択。暗黙の一律切替を禁止する。
- 一つのWorkspace内の通常組織Dataは通常のDataとしてQuery/Functionで利用する。保護を要する個別の項目、外部Consumerへの送出、別Tenant/Hostの実Read/Effectはそれぞれ該当Provider/Current Grantの別の条件で評価する。

### 10.3 C1ひとつの横断受入：時間・変更・判断・実行の独立証拠

同じ合成Case C1を、異動R1→R2の前後で繰り返す。ただし同じCase IDやContent Hashは同じRead Snapshot/Execution Attemptの証拠に**ならない**。テストはSourceの公開合成Dataだけを使う。

| 時点 | 組織事実とCase | 必須の別判定 | 未許可の短絡 |
| --- | --- | --- | --- |
| t0 / known0 | P1がD1所属、C1取扱D1、旧Rule版 | Query集合・完全集合性、候補・Rule版・Scopeを記録 | Membership=Case責任 |
| t1 / known1 | P1がD2へ異動、C1はまだD1取扱 | 候補を再評価する条件、旧担当の残余責任、変更なしのC1 | 異動=Case移管 |
| t2 / known2 | C1取扱をD2へ変更、Handoff未ACK | 旧/新責任4象限、通知とACKの区別、D1→D2切替の適用範囲 | 送信成功=受領=移管完了 |
| t3 / known3 | C1処理要求。Grantが失効する、または実Commit不明 | 現在主体/Policy epoch/Case revision/Effect Fence、Receipt、独立post-read、重複副作用防止 | 候補=許可、Commit不明=失敗で安全再実行 |
| t4 / retro-known | C1やA1の遡及訂正が追加される | valid/knownの異なる歴史解釈と過去の確定Receiptを両方残す | 歴史訂正=実際の過去送信の取消 |
| t5 / ABA | D1→D2→D1に戻る | revision/epoch/attemptを比較し旧候補・旧Receipt・選択を現在の証拠にしない | 値がD1に戻った=旧資格の回復 |

**同一試行の観測票（新Core Schemaではない）**：
- S：Source URL/位置/occurrence ID・Source/World/Meaning revision、Owner受理の有無。
- Q：Function/Query・必要Read footprint、実Provider/alias・取得条件・read cohort・Field/Relation trace、partial/unknown、validAt/knownAt。
- D：Business Rule版・Candidate/Decision・理由・関係Path・Case責任/移管版・必要ACK。
- E：実Principal/Issuer/Tenant・現在Grant/Policy epoch・対象Operation/Artifact/Resource・Effect Fence・Receipt/post-readの結果。
- H：Harness Activity Contextと固定Selection（対象のID/Source span/epoch）、用途、受信者、許可済みDisclosure slice、送出/受信/保持/意味保存/ACK。
- V：Expectedの独立性・Observedの実機構・test/CI exact HEAD・Provider資格・Owner受理・制約。

**Evidence joinのルール**：S/Q/D/E/H/Vが同じCaseを指していても、必要なContext、読取の版・Scope、実行試行、受信先は**それぞれ明示して照合**する。Sourceと有効時点が一致しても、多Providerのread cohortが同一だとは証明されない。欠落はunverified/unknownとして停止し、結果を保証したと言わない。

### 10.4 設計モデルと外部・AI Consumerの接続

| 経路 | 既存Owner | 組織C1での最小受入／禁止する誤昇格 |
| --- | --- | --- |
| Data→Function→Business→CRUD | #1124/#1125/#1126/#2497 | C1を処理するFunctionと依存Data、CRUD、責任Ruleを**同じ受理済みRelation occurrence**へ遡及。物理FK・JOIN回数・モデル図の線だけで業務意味を確定しない |
| Studio組織図・配置表・Tree | #1124/#2497 | 軸/valid/known/粒度/可視範囲を表示。多重親/隠れたEdgeを保持し、表示Preferenceと本務・指揮命令を分離。非可逆Selectionはread-only |
| GUI/CUIの固定対象→AI | #1158/#1124、Draft #4479 | 対象ID・Relation occurrence・Source span・Revision・固定時のepochを明示。フォーカス移動/多選択解除で対象が変わらない。再読取で古くなれば再選択を求める。固定対象は実行Grantではない |
| TEI→SHO Composer | TEI #2516/#1124、SHO各Owner、TEI PR #4317/#4467 | C1の説明・図・根拠をDocument Draftへ送るとき、TEI Model Revisionと文書編集Revisionを別保持。再投影しても著者編集を無言で失わない。Draft保存はOrg実Effect/業務受理ではない |
| SUI/CSW→TEI | SUI探索、CSW知識配置、TEI #2386/#3394 | Card/Island/KJ的AffinityはCandidate。許された理解パック→出典・対象・多重性・反例を検査→Ownerが採用した部分のみRelationへ。意味候補の出現を正式所属やCase移管としない |
| 外部Harness/SEI/TAI/EKI | #2516/#2509/#2499/#1174ほか | Activity Sessionの継続とHost再認可を別扱い。指示・計画・Agent提案・EKI委任/Job完了は現在Grant/確定Effect/業務結果を保証しない。開示対象/目的/Recipientの変更ごとに再資格化 |

### 10.5 WU-06〜14の増分受入契約（#4490と#4501との責務分担）

| WU | 次の判別力を持つ独立Expected | 担当／停止条件 |
| --- | --- | --- |
| 06/07 | same-valid・different-knownの結果反転。WHERE非出力依存、ORDER、EXISTS/NOT EXISTS、0件とpartialの反例。SQL Input名だけでPASSにしない | #4490。実Query/Providerは#2586/#2506/#2622 |
| 08 | Directive一件が複数Relationを変えるが、部分受信/中断を正式公開Snapshotとしない。複数Providerの原子性は実証範囲で限定 | #4490、実Effectは#1174 |
| 09/10 | Candidate 0/1/複数/partialと、Host Grant 0/1、失効・自己承認・期限・Scope/別Tenant・再委任循環を対照。各ScenarioのExpectedを**実Evaluator出力と照合**する | #4490。未実装の評価器をテストがあることにしない |
| 11 | Case責任切替・手順・候補/決定・通知/ACK・Effect Receiptを別列にし、同一AttemptでBusiness/RunPlanの接続可否を検証。新Workflow Engineなし | #2386/#1125/#1444。業務RuleのOwner受理前はcandidate |
| 12 | C1の時点別Graph/Table/Tree/CRUD/SHOでID/時点/粒度/隠れEdge・部分取得を照合し、非単射Projectionをread-onlyとする | #2497/#1124/#2516。実Consumer未確認はNOT RUN |
| 13 | C1のRelation edge選択→固定対象→AI変更候補→Preview→Owner Acceptance→Source再投影。Source改版/Grant失効/Selection ABA/再投影競合を**別失効**として検証 | #1124、固定対象Draft #4479、SHO側Owner。AI候補は自動書戻し不可 |
| 14 | 通常の部門・所属等を一律に機密扱いしない一方、特定保護Fieldを含む外部開示/Export/ログだけを適切な境界で制御。同じ公開Sliceの二世界で件数/診断/Tooltip漏えいなし | #2510/#2516/#2020。新組織ACLを作らない |

WU-11〜14をWU-06〜10の「全部完了待ち」にしない。前段の**必要な意味入力が独立Expectedとして固定**され、同一Source/Scopeを共有できた部分から並列実施する。統合受入のみ必要な証拠のjoinを求める。

### 10.6 実測Gate・実装順・撤回条件

1. **G0（既存）宣言Fit**：2026-11-12のmainにあるPR #4399の構造試験を維持。#4343の手書きExpectedとNode oracleが存在するかは実ファイルと実行で別確認。
2. **G1（先行）純粋Behavioral**：#4490のBEH-01/02を最初に実行し、expected resultが入力変更で反転することを確認。次にBEH-03〜05/06。G0のみでG1合格と呼ばない。
3. **G2（製品）実Read**：全TEIの初回 #2586 HJ-09を優先した後、Org C1について実Query→Provider Rowset→Context→一Consumerを同一Attemptで検査。fixtureの局所PASSを使用せず、物理Read/partial/Scopeを記録する。
4. **G3（横断）三視点/SHO/Harness**：同一Sourceの複数Projection、固定Selectionの遷移、Disclosure、受信／保持と意味保存を個別確認。SHO側の既存保存/再投影機能を再実装しない。
5. **G4（限定）Business決定→現在Grant→Effect/Handoff**：#2499/#1174/#1125/#1444 Ownerと、Grant失効/Commit不明/ACKなし/D1→D2→D1を実Host・実対象で別々に受入。実行の取消/再実行を推測で自動化しない。

**反証・撤回条件**：基礎S1に専用Schema/新Runtimeが必須になる、同名/同Hash/同Case IDで版/対象が自動結合される、Expectedが実装の出力から作られる、Display/AI/Documentの成功を現在Grantへ昇格する、取得partialを不存在にする、Commit uncertainを再実行可能と扱う場合は該当縦断を失格としOwner Issueへ返す。

**状態**：本追補は設計・独立Expected候補。これを記述しても新ORG-BEHの実行、実Org Provider/Host、実SHO/Harness送受信、Owner意味受理は成立しない。実測はPR head、利用した機構と出力、CIと独立Expectedの版を固定してからのみ記録する。

### 10.7 三要素分析法・REA・DB/SHO・AIハーネスを組織C1に接続する（2026-11-12追補）

**方法論の帰属**：渡辺幸三氏のX-TEA Modelerに示される**三要素分析法**のData・Function・Businessモデルを基準にする。「三視点モデル」は方法論名として用いない。TEI独自のGraph/CRUD/Source Provenance/Execution Evidenceを接合する部分は**TEI側の拡張**であり、X-TEA原典に存在すると主張しない。一次のモデル要素・操作の同一性を断定するには原本・版・要素ID・出典位置が必要。未照合の名称は候補と明記する。

| 三要素分析法に沿う区分 | C1の最小例 | 保存する境界／横断検証 |
| --- | --- | --- |
| Dataモデル | Person P1、OrgUnit D1/D2、Membership A1、Case C1、Responsibility R1/R2 | Entity・Relation occurrence・業務キー/物理PK・valid/known・grainを区別。Case取扱部門変更はMembership変更ではない |
| Functionモデル | C1照会、候補者解決、Case担当変更、引継ぎ通知、受領記録 | 各FunctionのI/O・対象Data・CRUDを明示。read-only照会・候補生成・更新・実通知のEffectを区別 |
| Businessモデル | 異動、案件担当引継ぎ、承認、合議、手作業による受領 | Task/Action/業務契機とFunction起動を分離。Functionなしの手作業を消さない。Function成功は業務完了やACKではない |
| 三要素間の照合（TEI拡張） | C1の業務手順→Function→Data CRUD→Source/Operation | Source/Meaning revision、実Read/Effect、Owner受理を別のEvidenceとしてjoin。図面だけで現在Grantを認定しない |

**REAとの接続は補助的な分析レンズ**とする。Resource（業務資源）、Event（経済・業務上の出来事）、Agent（関与主体）を区別して、組織変更が責任・資源管理・取引に及ぼす影響を説明する。REAのAgentと認証Principal、REA EventとTEIのDirective/Function Invocation/Effect Receiptは同義ではない。REAの分類・対応はSource-backed Candidate→OwnerのAccepted Meaningを経て、既存Data/Function/Businessへ投影する。REA専用の全業務共通Core Schema、暗黙の組織Grant、二重の業務イベント正本は追加しない。

**DBクライアント／SQL接続**：同一C1について、(a) モデル由来Query、(b) DBクライアントのNative SQL、(c) Provider実Rowset、(d) 業務意味としての採用を別資格で比較する。Schema/PK/FK/SQL JOIN一致だけでResponsibilityの業務意味を決定しない。SQL dialect、Provider、Scope、actual read cohort、NULL/未取得/非開示/0件、ORDER/EXISTS等の非出力依存を保持する。更新はPreview/Host Grant/Effect Fence/Receiptを経由し、DBクライアントの直接編集を業務Functionの正式実行と自動同一視しない。

**SHO／Studio／ハーネス接続**：Modelerの図面・CRUD、Studioの組織図、SHOの説明文・引用、AIの固定対象Selectionは、Source/Relation occurrence・Revision・用途・開示Scopeを損失なく参照する。SHOの文書保存とTEI Sourceの変更は別トランザクション。Pinned SelectionはFocus移動で解除しない一方、Source/Policy/Recipientの失効時には適用を停止する。SUI/CSWのKJ法的なAffinity/Islandは探索Candidateであり、組織正式関係に自動昇格しない。外部ハーネスの提案・委任は実Host Grantや実Effectを代替しない。

#### 追加の判別ケース（#4501への拡張、既存CXのExpectedを再利用）

| ID | 入力差分 | Expectedと検証経路 |
| --- | --- | --- |
| CX-13 | 異動Taskが発生したがCase変更Functionは起動していない | Business Taskは存在、Case C1のDataは不変。TaskEvent≠FunctionLaunchEventを確認 |
| CX-14 | 引継ぎ通知Functionが成功、受領は手作業のまま | Function/Effect成功とBusiness完了/ACKを分離。手作業はFunctionなしでも表現可能 |
| CX-15 | REA Agent P1と認証Principalが異なる／REA EventとDirectiveが異なる | REA構造候補は保持、Principal/Directive/Receiptへの同一視を拒否 |
| CX-16 | DBのFK・JOINが正しく、業務責任Ruleは未受理 | DB構造観測はPASS、Responsibility意味採用は未確定。SQL/Provider/Scopeを記録 |
| CX-17 | 同一C1をStudio/SHO/外部Harnessが別revision・別read cohortで参照 | 個々の描画成功と横断整合を別判定。Source/Meaning/Scope/Recipient一致だけで実Readの同一性を仮定しない |
| CX-18 | 固定SelectionにAI提案、同時にSHO著者編集とTEI Source改版 | Previewは保持できるが自動Applyを拒否。文書再投影・Source編集・現Grantの失効を別々に説明 |

**既存Issueへの割当**：CX-13/14は#1125/#1444、CX-15は#2386/#2510とREA担当、CX-16は#2586/#2387/#2506、CX-17は#1124/#2516、CX-18は#4501/#4479およびSHO側Owner。独立Expectedは#2514/#1180の既存方式を使い、#4490のBEHと重複させない。#2586の初回Product実Readを優先する順序は変えない。

**検証の資格**：本節は設計・試験候補である。X-TEA原本の全要素対応、REA意味受理、Native SQLと業務意味の同値、実Org Provider、SHO/Harness/Studio実C1 E2E、CX-13〜18の実行、Owner受理は **NOT RUN / NOT ACCEPTED**。
