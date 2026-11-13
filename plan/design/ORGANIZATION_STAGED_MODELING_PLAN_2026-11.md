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

**方法論の帰属・適用範囲**：X-TEA Modelerに関係する**三要素分析法**（データモデル・機能モデル・業務モデル）を、業務システム向けOrganization C1の評価方法として用いる。「三視点モデル」は方法論名としない。業務フロー、機能・業務階層、原典に由来するCRUD・逆引きクロスレファレンス・三者の相互牽制は可能な限り忠実に検証する。一方、TEIの汎用Coreを三要素分析法専用に固定しない。業務系では選択可能な標準/Domain Plugin・モデル投影として利用し、通常コード、非業務ドメイン、任意のGraph/Time/Spatial構造を妨げない。TEI独自のSource Provenance、Read/Execution Evidence、SHO/Harnessへの写像は**TEI側の拡張**と明示。原典と同一の機能・操作と主張するには版・出典位置・実Consumer比較を要し、未照合は候補とする。

| 三要素分析法に沿う区分 | C1の最小例 | 保存する境界／横断検証 |
| --- | --- | --- |
| Dataモデル | Person P1、OrgUnit D1/D2、Membership A1、Case C1、Responsibility R1/R2 | Entity・Relation occurrence・業務キー/物理PK・valid/known・grainを区別。Case取扱部門変更はMembership変更ではない |
| Functionモデル | C1照会、候補者解決、Case担当変更、引継ぎ通知、受領記録 | 各FunctionのI/O・対象Data・CRUDを明示。read-only照会・候補生成・更新・実通知のEffectを区別 |
| Businessモデル | 異動、案件担当引継ぎ、承認、合議、手作業による受領 | Task/Action/業務契機とFunction起動を分離。Functionなしの手作業を消さない。Function成功は業務完了やACKではない |
| 三要素間の照合（TEI拡張） | C1の業務手順→Function→Data CRUD→Source/Operation | Source/Meaning revision、実Read/Effect、Owner受理を別のEvidenceとしてjoin。図面だけで現在Grantを認定しない |

**名称衝突を解消する**：ここでいう**REA（Resource–Event–Agent）**は業務・会計の分析モデルであり、外部ソース解析ツールの**REA（#3394、#4322）とは別物**である。前者は補助的な分析レンズとして、Resource（業務資源）、Event（経済・業務上の出来事）、Agent（関与主体）を区別し、組織変更が責任・資源管理・取引に及ぼす影響を説明する。REA（Resource–Event–Agent）のAgentと認証Principal、EventとTEIのDirective/Function Invocation/Effect Receiptは同義ではない。既存Data/Function/Businessへの対応は業務意味の検討を経て採用する。REA専用の全業務共通Core Schema、暗黙の組織Grant、二重の業務イベント正本は追加しない。

**外部解析ツールREAから得る証拠は別の入口**：#3394/#4322のSource-backed Observation/Node/Relationは、Source locator/byte・解析ツールの版・Attempt・方向・並行Edge・原ID・coverageを保持する**構造候補**であり、Resource–Event–Agentモデルの要素という意味でも、正式なMembership/Case Responsibility/CRUDという意味でもない。重複Node ID、重複Relation ID、並行Relationの派生識別子衝突を最後の候補で上書きして採用しない（#4542）。候補が曖昧なら未選択・不適格として返し、意味Ownerによる対応付けを待つ。**PR #4542は修正を親PR #4322へ取り込んだ限定統合であり、#4322はmain未統合**。固定HEADでのREA Tool/Go試験成功は正式TEIモデル入場、TEI共通APIでの実Receiver受領、Owner意味受理や外部Host効果を示さない。

**DBクライアント／SQL接続**：同一C1について、(a) モデル由来Query、(b) DBクライアントのNative SQL、(c) Provider実Rowset、(d) 業務意味としての採用を別資格で比較する。Schema/PK/FK/SQL JOIN一致だけでResponsibilityの業務意味を決定しない。SQL dialect、Provider、Scope、actual read cohort、NULL/未取得/非開示/0件、ORDER/EXISTS等の非出力依存を保持する。更新はPreview/Host Grant/Effect Fence/Receiptを経由し、DBクライアントの直接編集を業務Functionの正式実行と自動同一視しない。

**SHO／Studio／ハーネス接続**：Modelerの図面・CRUD、Studioの組織図、SHOの説明文・引用、AIの固定対象Selectionは、Source/Relation occurrence・Revision・用途・開示Scopeを損失なく参照する。SHOの文書保存とTEI Sourceの変更は別トランザクション。Pinned SelectionはFocus移動で解除しない一方、Source/Policy/Recipientの失効時には適用を停止する。SUI/CSWのKJ法的なAffinity/Islandは探索Candidateであり、組織正式関係に自動昇格しない。外部ハーネスの提案・委任は実Host Grantや実Effectを代替しない。

#### 追加の判別ケース（#4501への拡張、既存CXのExpectedを再利用）

| ID | 入力差分 | Expectedと検証経路 |
| --- | --- | --- |
| CX-13 | 異動Taskが発生したがCase変更Functionは起動していない | Business Taskは存在、Case C1のDataは不変。TaskEvent≠FunctionLaunchEventを確認 |
| CX-14 | 引継ぎ通知Functionが成功、受領は手作業のまま | Function/Effect成功とBusiness完了/ACKを分離。手作業はFunctionなしでも表現可能 |
| CX-15 | Resource–Event–AgentのAgent P1と認証Principalが異なる／EventとDirectiveが異なる。別途、外部解析ツールREAが同名・多重・重複IDのRelationを返す | 業務分析モデルのAgent/EventとHost Principal/Directive/Receiptを同一視しない。解析ツールのObservation/Edgeは別の構造候補とし、衝突IDを黙って採用しない |
| CX-16 | DBのFK・JOINが正しく、業務責任Ruleは未受理 | DB構造観測はPASS、Responsibility意味採用は未確定。SQL/Provider/Scopeを記録 |
| CX-17 | 同一C1をStudio/SHO/外部Harnessが別revision・別read cohortで参照 | 個々の描画成功と横断整合を別判定。Source/Meaning/Scope/Recipient一致だけで実Readの同一性を仮定しない |
| CX-18 | 固定SelectionにAI提案、同時にSHO著者編集とTEI Source改版 | Previewは保持できるが自動Applyを拒否。文書再投影・Source編集・現Grantの失効を別々に説明 |

**既存Issueへの割当**：CX-13/14は#1125/#1444、CX-15は#2386/#2510（Resource–Event–Agentの意味）と#3394/#4322（外部解析ツール候補の受領・識別）、CX-16は#2586/#2387/#2506、CX-17は#1124/#2516、CX-18は#4501/#4479およびSHO側Owner。独立Expectedは#2514/#1180の既存方式を使い、#4490のBEHと重複させない。#2586の初回Product実Readを優先する順序は変えない。

**検証の資格**：本節は設計・試験候補である。X-TEA原本の全要素対応、Resource–Event–Agentの意味受理、外部解析ツールREA候補の正式Product入場、Native SQLと業務意味の同値、実Org Provider、SHO/Harness/Studio実C1 E2E、CX-13〜18の実行、Owner受理は **NOT RUN / NOT ACCEPTED**。

### 10.8 組織の運営能力・数量・変更シナリオ（2026-11-12追補、#2510/#2495/#2593）

**設計目標**：組織図の静的な正しさから、業務を担う能力・資源・責任・代替性の説明へ広げる。ただし**組織を固定の経営管理SchemaやシミュレーションEngineにしない**。人員、FTE、業務量、予算、スキル、設備、場所は一般Data/Quantity/Unit/Relation/Rule/Projectionの組合せとして表現する。Organization Profileは関係の役割、粒度、単位、集計則、時点と責任範囲を補足する。

#### 10.8.1 数量の意味契約（Headcount≠Membership≠FTE≠Capacity）

| 測定量 | 粒度と集計規則 | 拒否する短絡 |
| --- | --- | --- |
| Unique headcount | Person identityを指定Scope・valid/knownでdistinct。所属occurrenceが2件でも1人 | Membership行数を人数にする |
| Membership count | 有効なMembership occurrence件数。複数所属を保持 | 所属件数を従業員人数と呼ぶ |
| Allocated FTE | Person×Unit×有効期間のAllocation occurrenceを明示したBasisで合算。雇用FTEとの同一性は別Rule | 配賦合計を常に1.0に正規化、二重経路を二重加算 |
| Available capacity | Skill・稼働時間・予定・業務制約・Calendar/Unitの組合せ。FTEからの換算はOwner Rule | 1 FTE=常に一定の業務処理件数と推測 |
| Workload / demand | Case/Task/WorkItem occurrence×必要Skill/工数/期限・時点 | Case件数を工数に置き換える |
| Budget / Cost | Currency・会計期間・法人・CostCenter・配賦Basis・計上ルール | 所属先部門=費用負担部門とする |
| Skill / Qualification | 出典・適用業務・評価時点・有効期間・証明資格 | 技能=役職、資格=現行Host Grant |
| Location / Facility | 地点・時間帯・設備容量・安全上の制約 | 地理的所属=指揮命令・権限継承 |

数量は value/unit/basis/grain/validAt/knownAt/source/coverage を検査する。単位変換可能であっても、Basis（例：月間予定工数と実績工数、予算と実支出）が異なれば無条件に合算しない。未取得・NULL・非開示・0を区別する。配賦合計が1を超えるか否かは雇用形態・用途によるため、Coreが一律に拒否せずOwner Ruleに委ねる。

#### 10.8.2 既存Source fixtureを正本とする：P1/P2/P3とD1/D2

**出典を明示**：mainの `reference/org-semantic-views/source-fixture.json`（`org-synthetic-r7`、blob `d66f29505389f2096e9b825ba4f5e2d799d607d4`）および `scenario-fixture.json`。baselineは `validAt=2027-06-15`、`knownAt=2027-07-01`。P1→D1のA1=100 basisPoints、P2→D1のA2=60、P2→D2のA3=40、P3→D2のA4=100。100 basisPoints=1 FTEという**この合成Fixture内の換算基準**を使う。前稿の「P1がD1 0.6/D2 0.4、P2がD1 1.0、全体2人/3所属/2 FTE」は当該fixtureに存在しない仮例だったため**撤回**し、第二Source/独立Fixtureを作らず修正する。

| 指標 | D1 | D2 | 全体（重複排除） |
| --- | ---: | ---: | ---: |
| Membership occurrence | 2 | 2 | 4 |
| Unique Person（各部門内） | 2 | 2 | 3 |
| Allocated FTE | 1.6 | 1.4 | 3.0 |

**反例**：部門内の実人数2+2=4を組織全体4人とは呼ばない（同一P2の兼務）。D1にProject J1/J2をJOINして4行に増えても配賦FTEは1.6のまま（素朴なJOIN後合算では3.2になる）。Allocation occurrenceとJoin行数を分ける。P2の配賦0.6/0.4は業務責任・実行Grantの自動分割率ではなく、変更してもCase C1の責任は別Source/Ruleなしに変化しない。

**時点変更の非自明性**：validAtを2027-06-15のままknownAtだけ2027-07-03へ進めると、既存fixtureの遡及訂正によりA2は有効対象から外れる。Membership=3、D1=1.0 FTE、D2=1.4 FTE、全体2.4 FTEとなり、全体Unique Personは3のまま。この差は改版した新しい合成データを作らず、既存r7の時点切替だけで得られる。

#### 10.8.3 シナリオは確定Sourceと別資格

シナリオ比較を**read-only Derived View**として最初に提供する。BaselineのWorld/Meaning/Rule/Provider read cohortを固定し、Scenarioは一組の仮定・差分・計算条件を別IDで保持する。計算の出力はObservation/Projectionであり、Ownerによる業務採用・辞令・Case移管・実Commitではない。

- **欠員**：P1が一時的にD1で稼働不可となる仮定では、D1の**利用可能配賦FTE相当**が1.6→0.6となる。ただし実在Membership・在籍・Case責任・基準配賦FTE 1.6は変更しない。稼働可能量をFTEから業務処理能力へ換算するには別途Owner Ruleが必要。
- **仮配賦変更（正式異動ではない）**：P2のA2を0.6→0.0 FTE、A3を0.4→1.0 FTEとする仮定では、D1=1.0、D2=2.0、全体3.0。A2/A3のMembership occurrenceは維持し、実異動Directive・Case責任移管・正式Allocation更新を生成しない。
- **技能・業務量**：必要な技能と有効なQualification、予測処理時間とCapacityを別々に比較し、候補が0件・複数・partialの場合は適任者や代替承認者を断定しない。
- **組織統廃合**：旧部門IDを単純置換しない。Source occurrence、旧/新のRelation・Case責任・Budget Basis・権限Scopeを別々に扱い、D1→D2→D1のABAで旧シナリオを現在の確定版へ再利用しない。
- **複数Provider**：同じScenario IDであっても、取得Source/Scope/時点が異なる場合は単一の観測済みBaselineと扱わず、Coverageとunknownを表示する。

#### 10.8.4 TEIの既存横断機能へ配分

| 連携領域 | 提供する最小機能 | 既存責務・受入境界 |
| --- | --- | --- |
| 標準Data/Quantity/Unit | 人数・FTE・時間・金額・容量・粒度・Basis | #2495/#2593、組織Profile #2510。組織専用Quantity Runtimeを追加しない |
| Query/DB Client/ETL | valid/known付き集計、非出力依存、Provider/Scope/NULL/partialの保持 | #2586/#2506/#2622。SQL結果は業務意味受理でない |
| Rule/Business/Workflow | 必要能力・資格・担当責任・合議・代替者候補 | #2386/#1125/#1444。CandidateとHost Grantを分離 |
| Studio/SHO | 部門配置、FTE集計、欠員シナリオ差分、根拠付き説明 | #1124/#2497/#2516。図面のレイアウトやシナリオ値は第二Sourceにならない |
| SUI/CSW/AI Harness | 事例探索・組織改編案の提案・固定対象レビュー | #4501/#4479。AI提案から正式Directive/Grant/Effectへ自動昇格しない |
| EKI/外部Host | 確定済みFunctionの実行・配布・監査 | #1174/#2499。Scenarioを実行許可とみなさない |

#### 10.8.5 追加Expected（既存r7 fixtureの実測値と明示的に照合）

| ID | 独立Expectedと反例 |
| --- | --- |
| ORG-CAP-01 | r7 baselineのA1=100、A2=60、A3=40、A4=100 basisPoints。全体Unique Person=3、Membership=4、FTE=3.0。部門別Unique 2+2=4を全体人数にしない |
| ORG-CAP-02 | 同じAllocation occurrenceにGraphの別経路が増えてもFTE合計は不変。Relation occurrenceが増えた場合だけ対象Ruleに従う |
| ORG-CAP-03 | D1のP1稼働不可シナリオでD1の利用可能配賦FTE相当=0.6、Baselineの配賦FTE=1.6、確定Membership/Case C1/Receiptは不変（Case/Receipt自体は当該fixtureに未実装） |
| ORG-CAP-04 | P2のA2を0.6→0.0、A3を0.4→1.0へ**仮配賦変更**するとD1=1.0、D2=2.0、全体=3.0、Membership=4。Scenarioと実Directiveは別 |
| ORG-CAP-05 | FTEと時間、予算と実支出などBasisが異なる場合、単位変換できても無条件合算しない。NULL/partial/非開示を0としない |
| ORG-CAP-06 | Skillが一致しても失効資格・自己承認・別Tenant/Grantなしなら実承認不可。候補がpartialなら確定選任不可 |
| ORG-CAP-07 | r7のvalidAt=2027-06-15を固定しknownAt=2027-07-01→07-03でMembership=4→3、FTE=3.0→2.4。Rule/Provider/Policy/Scope/known変更時の再利用資格も再評価し、A→B→Aを同一版としない |
| ORG-CAP-08 | Scenarioの図・SHO説明・AI提案が成功しても、実異動・Case移管・Effect/ACKは発生していない |

**実装順**：#2510にある既存r7の人数/FTE Referenceを再利用し、ORG-CAP-01/02の純粋なQuantity/Graph集計→03/04のread-only Scenario差分→05/06のRule/Scope/partial→07/08の実Consumer/鮮度へ進める。独立Expectedを実装から生成せず、synthetic oracleのPASSを製品経路の合格と呼ばない。#4490のORG-BEH、#4501のCXケースとの重複Expectedは新規作成せず共有する。最初の製品実Read #2586をブロックしない。

**停止条件**：シナリオの仮定をAccepted Sourceへ無断で書戻す、FTEを権限やCase責任へ自動変換する、人数・所属件数・FTEの粒度を暗黙に混同する、SQL/Graph/SHO/AIの表示成功を実Provider/Effect成功へ昇格する場合は不合格。

**2026-11-12 実行可能なReferenceの追加と証拠資格**：main統合済み [PR #4532](https://github.com/hat47x/tei-platform/pull/4532)に、既存r7 Source/Scenario fixtureだけを読む `reference/org-semantic-views/verify-capacity-scenarios.mjs` を追加。ORG-CAP-01〜05/07の**合成範囲の一部**（人数・FTE、Join増幅、欠員の利用可能量、仮配賦、partial/NULL相当、knownAt遡及訂正、元Source不変）をNode v22.16.0で**ローカル7 subtests PASS**。実ソースのblob `d66f29505389f2096e9b825ba4f5e2d799d607d4`、実Scenarioのblob `b8fc779cf60e7816c4077616b4288735b49470e2`、実loaderのblob `d85c2dbe88803ad40f8826d764d45a35eb4a167a` と、試験コードのblob `6f07de5c793810f1e87553318710b9094d79c328` を使った。fixture/sourceとObservedのテスト自体が照合できても、独立Ownerによる業務意味受理には昇格させない。

**未検証**：ORG-CAP-06（Skill/資格/現在Grant/Host）・08（Case/Directive/Effect/ACK）、ORG-CAP-05の実貨幣/工数Basisや実開示・実Provider、ORG-CAP-07の実多Provider/Policy/ABA、TEI製品上のシナリオConsumer、GitHub Actionsの合成Node実行はmain統合済み [PR #4532](https://github.com/hat47x/tei-platform/pull/4532)の正確なhead `fa2d8ce98ff77536f51e32095748f63d935a2d07` について [Actions run #43242282586](https://github.com/hat47x/tei-platform/actions/runs/43242282586) の `synthetic-node` が **SUCCESS**。実ログで新規7件を含む**313 tests / 313 pass / 0 fail**、新スクリプトのGit blob `6f07de5c793810f1e87553318710b9094d79c328`、exact-head checkoutとworkspace cleanを確認。ただしPR #4500自体の新節はdocs-onlyであり、このCI成功は**合成Referenceの資格のみ**。実Provider・Host等のOwner意味受理は**NOT RUN / NOT ACCEPTED**。本節の数値は未来日付2027年の合成データに基づくテスト値であり、実在組織の観測値ではない。

### 10.9 データ構造変更と画面マッピングの整合をC1で固定する（2026-11-12追補）

**他チャットの確定方針を組織C1へ適用**：テーブル定義を直接表現するデータモデルは、その**対象版のテーブル定義の写像**である。正本のテーブルFieldを明示削除し、その版を採用した時点で、**同じ版のデータモデル上からField要素そのものが消える**。実DBへの外部変更を完全な観測で確認した場合も、**Actual Schema投影の現行モデルFieldは消える**。ただし外部DB観測を理由にTEIのDesired Sourceまで無断で書き換えない。Desiredに旧Fieldが残れば、差分・ドリフトを示す。独立定義の業務意味/Computed Fieldは必要に応じ残し、失った物理依存を診断する。Function/UIの**表示要素は削除せず未マッピング警告**、画面追加だけでDB列を作らない。GUI/CLI/AIハーネスで同一の操作意味とする。

#### 10.9.1 三つの構造状態と三つの変更入口

1. **Actual Schema**：特定DB接続・Provider・Scope・観測時点で実際に取得できた構造。完全取得・権限・観測Epochを記録。取得失敗/partialで「列がない」を確定しない。
2. **Desired Schema / Change Intent**：開発者が明示して編集した希望構造。物理DDLの実行結果ではない。Preview/Impactを経てMigration Apply資格のある操作のみ実施する。
3. **Model / UI Projection**：Dataモデルの物理写像、論理Relation/Rule、Functionモデルの画面項目、Businessモデルの処理/CRUD、Studio/SHO文書の各表示。Actual/Desiredのどちらを示すか明示し、UIは独立した要素IDとMap Targetを持つ。UI要素はDB列の別名ではない。

| 操作入口 | 許可する最初の動作 | 自動実行しない動作 |
| --- | --- | --- |
| UI/Functionモデルへ項目を追加 | UI要素だけ生成し、未マッピングを表示。既存Data Field/Derived Field/入力専用項目への明示Bindingを提案できる | 新しいRDB列・業務意味・権限を黙って作る |
| 開発者がDataモデル/Schema Workbenchで新Fieldを明示 | Desired Field・対応Source差分・制約/依存・Migration PlanとPreviewを作る | 実DB Apply、既存UIへの無断Binding、既存Host Grantの推測 |
| 実DBの列を削除/変更 | Schema観測でActualの変更を検出し、現行Data物理写像と下流参照のImpactを再計算 | 依存UI要素や業務責任/Caseを連鎖削除、旧物理列へ黙ってリバインド |

**逆方向の扱い**：Modelerによる列削除指定はまずDesired上の提案であり、実DBを削除したことにはしない。Migration成功ログのみでもActual確定とはせず、別Readで対象列の消失を確認する。別名で同型列が現れても、明示されたRename/Mappingの採否がない限り旧Field Identityを再利用しない。過去Model/文書の引用は版付きで残す一方、**現在の有効Mapping**は外す。A→B→Aでも旧Mapping/AI提案は自動復活しない。

#### 10.9.2 組織C1の最小変更トレース

| ステップ | データ側 | Function/UI側 | Business/Effect側 |
| --- | --- | --- | --- |
| M0 基準 | org_membership.assigned_unit_idを物理Field F1として観測、モデル写像M1 | U1=「配属先」、U1→M1で有効 | MembershipとCase C1のhandlingUnitは別関係 |
| M1 UIに「兼務区分」U2を追加 | Actual/Desiredとも不変 | U2をunmappedとして表示し、Binding候補だけ提示 | 業務Rule/DDL/Grant/Effectなし |
| M2 開発者が兼務Field F2を明示追加 | Desired F2とMigration Planを作り、Apply前はActualにF2なし | U2→Desired F2は設計中と明示、実画面の保存先確定にはしない | 変更内容と受入範囲に応じ通常の開発者Review |
| M3 F2適用・再観測 | 対象実DBでActual F2を確認し物理Mapping確立 | U2のBindingを明示承認すれば有効、未承認ならunmapped | 必要なら既存Function/Validation試験と実Provider試験 |
| M4 F1削除が実適用・完全再観測済み | **現行ActualテーブルモデルからF1 Fieldを除去**。Desiredが旧Fieldを保持するなら差分として診断。旧版Source/履歴は保全 | U1は残すが、Actual側Bindingを無効化し「対応項目なし」を警告。CRUD/SHO/AI参照を失効診断 | Case C1責任やMembership別属性を推測で書き換えない |
| M5 古いSource/AI応答が後着 | F1の旧revisionを参照 | 固定Selectionは意図として残せてもApplyは禁止。Previewで再対応付けを要求 | 古い実行資格・Receiptを流用しない |

**補足**：非RDB/仮想Field/計算Field/APIの画面Bindingには物理列を必須にしない。Source Provider別の能力を説明し、RDBのDDL意味をREST/OData/ログへ強制しない。直接DB操作と業務Function経由の更新は同義ではない。表削除やデータ消失を伴う操作の実適用は既存Migration/Host Grant/Effectと独立Readの範囲内でのみ実証する。

#### 10.9.3 追加の独立受入ケース（既存Driverを再利用）

| ID | 期待と反例 |
| --- | --- |
| ORG-MAP-01 | U2の追加でActual/Desiredの列数が変わらず、U2はunmapped。UI要素数が増えたことはDB列作成の証拠ではない |
| ORG-MAP-02 | 開発者の明示Field追加はDesiredだけ変化。DB Apply前は実Rowset/保存資格が未成立 |
| ORG-MAP-03 | 実F2追加後の独立Schema ReadでF2を確認。UI Bindingは別の明示選択として保持 |
| ORG-MAP-04 | F1の実削除・完全再観測後、**Actualテーブル定義モデルの現行Field一覧からF1が除去**される。U1は保持されunmapped診断、DesiredにF1が残るならドリフト、CRUD/SHO/AI参照は要再確認 |
| ORG-MAP-05 | Migration拒否・部分失敗・別環境のみ成功では、対象ActualにおけるF1削除が**完全に観測**されない限り現行Field要素を勝手に削除しない。Desiredでの削除Intentとは別 |
| ORG-MAP-06 | 同名/同型F1再作成、Rename、A→B→A、古い固定Selectionのいずれでも暗黙Rebindと旧Grant流用を拒否 |
| ORG-MAP-07 | 物理org_membership変更だけでCase C1のhandlingUnit/責任・承認者・ACKが自動更新されない |
| ORG-MAP-08 | Source/Provider/Scope partialで欠落した列を削除と断定しない。Fieldの存在がunknownな場合と、完全観測で存在せずunmapped確定な場合をUI/CLIとも別表示 |
| ORG-MAP-09 | RDB非依存のComputed/Input-only/API BindingはFieldのないことだけで拒否せず、Provider/Ruleと出典を検証 |
| ORG-MAP-10 | GUI、CLI、Target画面上のAI編集のいずれも、同じ対象Identity・意図・差分・現在資格・Preview・結果を示し、明示Applyまで物理変更しない。描画Edge IDを意味Relation IDへ昇格しない |

**Owner配分／既存基盤とのFit**：データ項目/論理物理対応は#1126、三要素分析法の相互牽制・投影は#1124、横断変更資格は#1143、Source/Schema Workbenchは#3270、Migrationは#4125/#2892/#2887、Target画面からの変更は#1452、GUI/CUI固定対象は#4479、失われる情報・逆写像は#2497/#2500、ORG C1業務意味・横断Scenarioは#2386/#4501。**第二のMapping正本、汎用Migration Engine、画面定義専用Core、新Workflow/ACLは作らない**。

**段階順**：最初にM0/M1/M4の**read-only静的Source→モデル/画面参照診断**と手書きExpectedを実施し、次にDesired/Actualの分離（M2/M3/M5）を既存実Schema APIと結合する。その後に実DB独立Read/実Target/SHOを#2586のP0実Readや各OwnerのGateに接続する。全Org-MAPケースの局所試験を全製品のE2Eへ昇格しない。**状態**：新Expected・実GUI/CLI・実DDL/Read・Owner意味受理は NOT RUN。

#### 10.9.4 一操作をどの版のモデルに反映するか（UI/CLI共通Decision Table）

「モデルから削除」の意味を、**何を正本とするモデルか・どの版の話か**まで固定する。**テーブル定義モデルのFieldの削除**と、**Function/UI要素を残したままBindingを外す**ことを別の意味変化として扱い、両方の対象Identityを維持する。データモデルの上位の業務概念を、物理Columnの削除に連動して無条件に消してはならない。

| Event / 観測資格 | 現行テーブル定義モデル | Function/UI・CRUD・派生参照 | 許される操作 |
| --- | --- | --- | --- |
| UIの新しい入力欄追加（org兼務区分U2） | ActualもDesiredも不変 | U2は未対応、候補Mappingを示す | 明示的な新規Field設計への**提案**のみ |
| TEIのDesired Column F2を明示追加 | Desired版にはF2を追加、Actualには未出現 | U2を「設計中」と示し、実保存可能とは表示しない | Schema差分/DDL Preview、明示Apply |
| 実DBでF2の追加を完全に再観測 | Actual版のテーブルモデルにF2が出現 | U2は明示Mapping採用の後に有効 | Actual Reader/Consumerの別検証 |
| TEIのDesired Field F1を明示削除、実DBはまだ旧版 | **Desired版からF1を除去**、Actual版にはF1が残る | U1は残す。Desired投影で未マッピング、実稼働Actualへの影響は別判定 | Impact/Dependency/旧新Client/移行PlanをPreview |
| 直接DB操作等でF1削除を完全再観測、TEI Desiredは旧版 | **Actual版からF1を除去**、Desiredとのドリフトを記録 | U1は残すがActual Bindingを無効、CRUD/派生Query等は要再評価 | 原因と影響調査。Desired Sourceの無断書換え不可 |
| Schema観測がpartial・unavailable・permission denied | ActualではFieldの不存在を**確定しない** | 「存在不明/取得不足」であり削除済みとは表示しない | 再観測。確定不存在を前提とするWrite/移行判定は停止 |
| 同名Column再作成、Rename候補、A→B→A | 新旧のSource/Field occurrence・revisionを区別 | U1の旧Bindingを黙って復活させない | 明示Rename/Mapping受理後に再評価 |

**操作の最小識別子（テスト票であり新Core Schemaではない）**：Subject/Workspace（Pin Intent）、表示Projection版、TEI Source/Desired版、実DB target/Provider/Actual観測Epoch、Field occurrenceとUI element IDを別列にする。

**影響計算**：SQLのSELECT出力だけでなく **WHERE・ORDER・EXISTS・派生Field・Rule・Report・CRUD** を追跡する。F1が画面に表示されていなくても、変更影響は0とは限らない。Read Cohortが不足すれば「全影響なし」を確定できない。

**AI固定対象の段階導入（#4515）**：現行Workbenchで初めに固定するのはSource-backed **Subject**。現在のrelatedGraph()が生成する親子Edgeの描画ID parent:subjectKey(child) は**Relation occurrence IDを保証しない**。同じ端点の複数意味Relation、異なるWorkspace、同offset別Subjectが区別できない状態では、EdgeをAI編集対象にしない。正式Relation ID・Source版・出典へ解決できるまで**Edge操作は閲覧のみ、更新選択はunsupported**。Node Pinも選択意図であり現在Grantではない。既表示DOM/ARIAの開示失効は#2516側で扱う。

**Undo/競合**：ローカルPin/表示編集意図のUndoと実DDL/データ変更のUndoを同一視しない。DROPをUI Undoだけで復元したと表示しない。操作前後にSource/DB/Policy/Grant/ProjectionのRevisionと実観測を再照合し、後着AI応答・同名再作成・A→B→Aでは旧Bindingと旧Grantを復活させない。

**既存Expectedへの追加assert（新ID・第二正本を作らない）**：
- ORG-MAP-04/05/08：完全Readで「Field F1不存在」とpartialで「F1未観測」を**異なる結果**として返す。UI U1は残り、前者だけがActual版Field一覧からF1を除去した状態。Desired版の削除はSource Intentの別操作。
- ORG-MAP-06/10と#4515 PIN-S0-03/04：同名・同offset・描画Edge・A→B→Aの変化で古いPin/Mapping/AI Applyを拒否。未実装ならunsupportedと返し、成功としない。
- ORG-MAP-07と#4490 ORG-BEH-02：F1がWHERE/Rule等に**非出力参照**されれば、削除・型変更に伴いQuery/集計/派生値の再検証が必要。Case C1責任や実Grantは暗黙に更新しない。
- ORG-MAP-03/05と#4125：Desired→DDL→実DB Apply→独立再観測は別証拠。DB変更成功は全旧新Client/Business Meaning受理を意味しない。

**最短の引渡し**：W1は#4515 Workbench Subject Pinと#1124/#1143のread-only schema/UI diff・unmapped診断を並行実証。W2は#3270/#1126の既存Actual/Desired Inspectorへ変更資格を接続。W3は#4125の独立Schema再観測と#2586実Query/Consumer、#1452実Target/SHO/AI後着を照合。W1だけで正式Relation Pin、Schema Apply、Current Grantを完了扱いしない。

**状態**：本節は既存受入条件の明確化。ORG-MAP-01〜10、PIN-S0、実DB/DDL、実GUI/CLI/Host/SHO、Ownerの意味受理について新しい実行証拠は **NOT RUN / NOT ACCEPTED**。

### 10.10 参照ナビゲーション・未使用診断・大規模Studio性能（2026-11-12横断統合）

**設計方針**：組織C1を題材に、定義項目から「定義へ移動」「参照先へ移動」「参照元の一覧」「未使用候補」「変更影響」をGUI・CLIで共通化する。既存のSource Index、ModelRef Derived Graph、三要素分析法のモデルのCross Reference、trace/inspectを再利用。第二の参照正本やStudio専用Parserは作らない。**正確性→対象範囲の完全性→操作応答時間**の順に実証する。

#### 10.10.1 参照一覧を信頼できる条件

1. **基礎の誤りを種類ごとに受入**：#4475のFingerprint入力破壊はmain統合PR #4507、同Kind同IDの重複診断と曖昧なIndex entry拒否はmain統合PR #4524で限定修正済み。ただし、全Derived/CLI/SHO Consumerが不適格入力を排除する保証は別。#4472のData索引取り違えはPR #4513（squash merge `7e36f85ad0f23aed0bf8f8ecec47c7b160574a3d`）で3箇所を限定修正しmain統合済み。明示CRUD-only組の派生表脱落・UnusedDataの誤判定は#4551で修正しmain統合済み（squash 1a2271d8）。ただしStudio/SHO/AI Consumerの参照完全性、レビュー要の表示・適用停止は別試験が必要であり、Product全体の合格ではない。
2. **参照の種類を混ぜない**：宣言上の参照、型付きRelation、Function→Data CRUD、Business Task→Function、SQL WHERE/ORDER/EXISTS等の非出力依存、UI Binding、SHO引用、実Provider Readは別種。逆引きは有向Edgeの逆方向検索であって、業務上の逆向き依存関係を意味しない。
3. **対象の同一性**：Workspace、Kind、Stable Subject ID、Relation occurrence、Source revision、Source位置と必要なvalid/known/Scopeを保持する。Label、図面座標、同じ始点・終点、描画用Edge ID、単独のSource offsetを意味IDとしない。参照の出現位置が複数ならoccurrenceを区別。
4. **0件とunknownを分離**：指定したSource/Provider/Plugin・Dependency Closureを必要十分に取得し、未解決参照がなく、**指定範囲について全件検査できた場合だけ**「参照数0」「その範囲で未使用候補」と表示。partial、permission denied、unavailable、未解析のDynamic ReferenceやPluginには「不明／未検査」を付す。0件、空配列欄の欠落、対象不存在、非表示、非開示は別状態。
5. **未使用候補から自動削除しない**：公開API、外部呼出し可能Function、将来の契約、意図的なManual Task、動的参照は静的0件でも確定Unusedとは限らない。削除・Renameは#1143/#1126/#4125の明示変更Previewへ。参照一覧からの移動はread-only、AI Pinは#4515のSubjectから始める。

#### 10.10.2 C1の既存Expectedを使う操作・負例

#2510の公開合成Source、#4501 CX-04〜09、#4472 TE-CR、#4515 PIN-S0、#4490 ORG-BEH-02の**既存Expectedを再利用**。以下は実行の組合せであり新Expected Storeを作らない。

| 操作・負例 | 判別する期待 | 実装Owner |
| --- | --- | --- |
| OrgUnit D1とMembership A1を順方向・逆方向で移動 | 双方向から同じReference occurrenceとSource位置を開く。表示上のページ分割で全件数の分母を失わない | #3788/#1177、CX-04 |
| P1の兼務で二つの意味Relationが同端点 | Relation occurrence/Role/時点を区別し、表示Edgeだけから変更対象を確定しない | #4515/#2497、CX-05/08 |
| Business Task→Function→CRUD→Data F1 | 明示CRUDも非出力SQL依存も参照に残す。#4472未修正時は未資格と表示 | #4472/#1124、CX-13/16 |
| F1削除後に画面U1の参照元を列挙 | 現行Actual Fieldの確定不存在とpartial未観測は別。U1を残してunmapped、Desiredとのドリフトも区別 | #1126/#1143、ORG-MAP-04/05/08 |
| F_unusedと外部公開Functionの静的参照を比較 | 完全探索で0なら前者は未使用候補。後者は実際の外部利用が未観測で、0を削除資格としない | #3788/#3262/#2514 |
| Workspace/Source改版・ABAと遅延Inspect | 古い検索結果やPinを現在の編集資格に再利用しない | #4515/#2505、ORG-MAP-06/10 |

#### 10.10.3 高速性・Plugin肥大化防止の実測

**既存#1177に性能実測を追加**し、Index・Cache・Plugin Runtimeを複製しない。同じ公開合成C1をSource/Relation/CRUD数で小・中・大に拡張し、trace summary/full/limit=1、逆引き、未使用診断でcold/warm p50/p95、ピークメモリ、Workspace読込み・Source Index・Derived Graph・Query・描画それぞれの時間、変更後の再計算範囲、入力→応答時間を分けて測る。閾値は測定後にOwner判断とし、根拠なく固定しない。

- limit=1で表示が1件でも内部は全件Load/Sortする可能性がある。**全件数の正確さと表示ページの高速さを別に測る**。暗黙の検索打切りで結果を高速化しない。
- Source/Meaning/Dependency/Plugin版、Query Footprint、Scope等が変わったらIndex再利用資格を再評価。結果件数やSource hashが偶然同じでもA→B→Aの旧資格を復活させない。#2600のBuild Artifact/Evidence cacheとは別。
- Studio PluginはCapability発見→必要なPanel/Providerだけ起動する方針を検証し、非使用PluginがExplorer/Inspect/Trace/キーボード応答を遅くしないよう測定する。ただし**未起動Pluginの参照を0と誤報しない**。
- 通常編集でクリック・キーボードから定義へ移動し、参照元一覧から元の位置へ戻る操作を#3788で検証。AIに送る文脈は固定Subjectと必要最小範囲、現在のDisclosure資格に制限。#2516の既表示DOM/ARIA境界を別検証する。

**段階と停止線**：R0=#4472/#4475のProduct参照正確性、R1=#3788/#1177のHeadless前後参照と完全性、R2=#4515のSubject Pinを用いたStudio移動、R3=#1143のRename/Delete Preview、R4=#1824/#2600の大規模/Plugin性能比較。R0/R1を#2586の初回実Provider Readの完了待ちにはしない。どの段階でも、未知の参照を「0件」として削除提案する、旧SourceへAIが書戻す、画面描画成功を実Grant/Effectと呼ぶ場合は停止。

**状態**：本節は既存Issueの受入条件と順序を接続する設計。新しいGo/Node試験、Studio/CLI操作、性能計測、実Provider Read、Owner意味受理は**NOT RUN / NOT ACCEPTED**。

#### 10.10.4 現在の製品資格を消費者別に記録する（実行前チェック、2026-11-13）

| 既存成果・Owner | 実際に確認された範囲 | 組織C1の実行に残るもの |
| --- | --- | --- |
| #4507/#4524 merged（#4475） | Fingerprint入力破壊・自己参照の限定修正、およびData/Function/Businessの**同Kind同ID重複をエラーにしIndexから除外**する修正はmain統合済み | Derived/CLI/SHO等の全ConsumerがValidate結果を必ず評価する保証、現在Disclosure/Grantは別。#4514のSource-only drift修正もmain統合済みだが、Source再束縛と現在Disclosure/Grantは別 |
| #4513/#4534/#4551 merged（#4472） | #4513でData索引を修正、#4534で複数CRUD衝突と構造ok/CRUDレビュー資格を分離、#4551で**明示CRUD-only組を派生CRUD表・Function/Data参照へ追加し、UnusedDataの誤判定を修正**（main squash 1a2271d8）。それぞれ対象HEADのCI成功を確認 | **Studio/SHO/AIの実Consumerで参照完全性・要レビュー表示・Apply停止を検証していない**。衝突を無条件Hard Errorにも業務意味OKにも昇格しない |
| #3906 closed、CLI JSON契約 | trace.unresolvedは**返した行の未解決数**、projectionUnresolvedは**グラフ投影全体で落ちた参照**と明記済み | trace.unresolved=0だけで全Workspace/同Subjectの必要参照が完全と推定しない。両方とScope/Required Closureを照合する |
| #4532 merged（#2510合成Reference） | org-synthetic-r7、3人/4所属/3.0 FTE、known改版の2.4 FTEを参照。固定HEAD fa2d8ce98ff77536f51e32095748f63d935a2d07 の Node Actions #43242282586は313/313 PASS（うち追加7件）、関連4 WorkflowはSUCCESS。main squash `2e52ca47ef8f75c7bf61f86f5791834c67adcdd1` | **合成Nodeのみ**。正式TEI Query/Provider/Context/Studio/Host、ORG-CAP-06/08、Owner受理は別。マージ後のmain新HEADでの全製品再検証を本証拠から推定しない |
| #4539 merged（#2586の実Provider Read） | `order_entry.recent_orders`をEnvironment/Database instanceに明示束縛し、実PG16.15のRaw Rows→Function Data Contextへ限定接続。正常、0件、NULL、列欠落、RLS、権限拒否、SQL差分を隔離環境で対照。該当HEADの7 GitHub Workflow成功、実PostgreSQLの実走はPR記載のローカル証拠 | これは**受注Queryの一経路**であり、組織r7のMembership/Case C1 Queryではない。`tools/reference-browse-read`という別CLI入口であり、通常`tei` CLI/Studio/SHO/AIは未結線。環境→実Provider対応はhost-asserted、Resultの`coverage=unestablished`、0行でも不存在/`EmptyJustified`確定なし。main squash `ee1af44cb9d66507188b80195fe4e59013f89b90`で限定参照実装は統合済み。ただし通常CLI/Org C1実Query/Studio/Host完了扱いは禁止 |
| #4257 merged／#4542は親#4322へ統合（外部解析ツールREA） | REA 6.7は単独JSON、6.8は7 record＋metadata終端という版差を確認。#4542の重複Node/Relation・並行Edge由来の識別子衝突拒否は、親#4322上の限定経路でGo契約・REA 6.7/6.8実Tool CI成功 | **親#4322はmain未統合**。同一Attemptの共通read-only API Receiver、Source/Attempt改版後の再資格化（REA-ID-04）、正式Meaning・CRUD/Unused/Hostは未実証。Accepted Graphへ混入しない |

**read-only参照表示の入場条件**：適格な参照検索を、（A）Accepted Source/Meaningに属するProduct Cross Reference、（B）REA等からの外部Observation/Structural Candidate、（C）Source不足・未解決・partialの三群として扱う。同名や同端点が一致しても群間でReferenceを加算して「利用実績」としない。外部候補はSource locator、Producer version、record/edge ID、方向、多重性、raw digest/受領byte、Coverage、Known/Validの有無を区別し、正式な意味Role/Relation occurrence/業務時点は#2386の意味Ownerが採否を判断する。外部候補しか見つからないData/Function/Businessは「正式参照0・候補あり・Coverage未確認」を区別して提示する。

**最小の次のProduct Exit（重複Expectedなし）**：
1. #4513の索引、#4534の複数衝突と構造ok/レビュー資格、#4551の明示CRUD-only/Unusedをmain実装基線とする。TE-CR-02/03/04では**各CLI/Studio/SHO/AI Consumerの受領**を別検証し、明示行と参照数、unknown、レビュー要の表示/変更Previewを照合する。Go派生試験のみでProduct全体適格としない。#4524の同Kind重複Admissionも別判定。
2. #3906の二種の未解決数とSource Index/Scopeを同一CLI応答で判別し、完全0・投影未解決あり・部分取得の3入力で未使用の結論が切り替わることを実証（#3262/#1177）。UI・AIの未使用削除は禁止。
3. #3394の外部REA T1-Aは、実際の共通read-only APIに1 Observation＋1 Edgeを送出して実Receiverのbyte・識別・方向・partialを照合する。Product Cross Referenceに混入させず候補として保持。外部ProducerのCI成功だけを実TEI Reader成功にしない。
4. #2586/#4539で成立した一受注Queryの実PG read-only経路を**通常`tei` CLIの既存Source/Environment選択入口**へ結び、選択/実Invoke/statement+params/Rowset/Function Context/CLI受領の同一Attempt証拠を照合する。隔離PGで同じQuery Sourceから違う件数が返る反例を保持し、0件と全件取得・存在しないことを混同しない。実Org C1への移植は別Source/Provider/二時点Queryを接続してから。**GUI完成待ちでCLI Product readを遅らせない**。
5. #4515 Subject Pinと#3788 Source移動/CRUD/Studioを**同一受理済み合成Source・同一Revision**へ限定接続。#2497のloss/逆写像を保持し、完全性不明時の変更はPreviewで停止。#2586実Read縦断は引き続き独立P0とし、上記以外のOrigin/Recipient/Effectは別資格。

**Owner/証拠の境界**：組織Meaning #2386、三要素CRUD #4472/#4475、外部REA受領 #3394/#3419、参照/未使用 #3262/#1177/#3788、Studio #4515/#1124、独立Expected #2514/#1180。新しい共通Referencer、第三Source Store、REAを直接意味認定するConverterは作らない。#3906は契約説明として既にclosedであり、同じIssueを再起票しない。§10.10.4は現在状態のレビューと次の受入要求であって、新たなGo/Node/REA Product実行を行った証拠ではない。

### 10.11 UI起点のデータ文脈設計と問い相対の変更影響（2026-11-13横断統合）

**補正方針**：§10.9の「画面項目追加だけではDB列を作らない」は、**UIから新Fieldの設計へ進む経路を禁止する意味ではない**。業務向けModelerではUI配置を開発の入口として許容し、既存Data FieldへのBinding、新しいData Fieldの明示設計、Derived/Computed Field、入力専用Transient Fieldを選べるようにする。決定論的に検証可能な型・Source/Provider能力・依存・一意性は既存Rule/Schema/Contextで評価し、意味が曖昧な候補だけAIの提案対象とする。生成AI提案の提示を正式Mapping採用・Schema Applyと同一視しない。

#### 10.11.1 UI部品からSourceへの到達は段階的な明示操作

| UIで選択する設計意図 | まず作成・更新するもの | 更新前に検査するもの | 実行していないこと |
| --- | --- | --- | --- |
| **既存項目を利用** | U2→既存Field/Function Data ContextのBinding候補とPreview | Subject/Field occurrence、型、NULL、現在版、Owner、利用Purpose、read/書込能力 | RDB列追加、Case責任変更 |
| **新しいデータ項目を設計** | U2→Data Context/Logical Field候補→開発者が明示したDesired Field F2と論理/物理Mapping | 業務キー/物理PK、型、制約、変更影響、RDB/非RDB Provider能力、Migration Plan、他Consumer | UI配置直後の暗黙DDL/実DB書込み |
| **導出値を表示** | U2→Derived/Computed Field＋Source依存・計算Rule | 演算と単位、valid/known、取得完全性、再計算Scope、Read/Disclosure | 物理列の強制作成、業務確定値への自動置換 |
| **一時入力を受ける** | U2→Function input/Command parameterなどのTransient Data Context | 型、入力Validation、Action引数、保存要否、Current Grant | 永続項目の暗黙作成 |

**新Fieldの明示設計操作**では、「この入力欄を永続保存する」「既存Sourceに対応付ける」「計算値にする」等の意思をStudio/CLI/固定対象AIの同じOperation Intentとして確認する。次いでSource/Desired RevisionへCandidate Patch→Impact Preview→既存の変更受理→必要時のみSchema Migration→対象DBの独立Readへ進む。**通常の開発者によるモデル変更は通常のレビュー・テストでよい**。追加の独立したMeaning承認Engineや常時の重い承認フローは設けない。ビジネスルールの解釈や既存権限変更が必要な場合のみ該当Owner/Hostへ戻す。

**逆編集の追加停止線（#4533）**：Source-backed既存属性へUI/表から直接編集を返す場合は、操作ごとに対象一意、属性の恒等表示、局所的な**値Span**、現在Source版とSpan値の一致、変更後のValidate/再投影の5条件を確認する。属性**名**のSpanを属性**値**のSpanと取り違えない。Graph→Treeの隠れたEdge、集計値、丸め・単位換算後の値、非単射のProjectionをそのまま原Sourceへ書き戻さない。#4533の合成一属性パッチ実測は一般UI→DB更新の製品資格ではない。Source Patch適用にも正式のWriter/Grantが別途必要であり、実DDLはさらに別Effectとする。

#### 10.11.2 変更影響は「問い・Claim・Consumer」相対で評価する

同じUI/Schema変更に対して、影響範囲は全Workspaceの単一staleフラグでは判定しない。#2582のClaim別Required集合、#4518/#4526の限定Closure/Native Source Binding、#2387のSQL依存、#3262の必要設計情報を組み合わせ、次の3結果を区別する。

- **affected**：要求された表示・計算・Actionに影響することが既存Source/Query/Rule/Bindingから確定した。再評価/再取得が必要。
- **scoped-unaffected**：指定したClaimのRequired集合と実観測Scopeが閉じており、変更と交差しないことが確認できた。別Claimや全画面への一般化は禁止。
- **unknown**：Requiredが空/不明、Imported Source・Provider・Pluginが未取得、SQLの暗黙列/結合/否定依存が解決できない、受領が途中切断。0件や無影響として再利用しない。ただし**関係のないUnknownで他の完全なClaimまで一律失効させない**。

対象となる非出力依存にはWHERE、ORDER BY、EXISTS/NOT EXISTS、LEFT JOIN、集計、Top-Nだけでなく、**SELECT *、alias.*、NATURAL JOIN**の暗黙的な列集合や結合キーを含む。新Field F2の追加で列集合が変わるSQLや、同名列追加によりNATURAL JOINの結合条件が変わるSQLを「明示Field参照が0だから影響なし」としない。実Rowset/Consumerの必要範囲が未観測ならunknown。Provider差分通知の非対応では既存Queryによる再読取に戻る（#2582）。

#### 10.11.3 組織r7に結び付ける受入の最小対照

| 既存Expectedへの追加観点 | 入力の対照 | 必須の判別結果・Owner |
| --- | --- | --- |
| ORG-MAP-01/02/03 | P2兼務一覧へ「配賦メモ」U2を配置し、既存Binding/新Field/Derived/Transientを明示選択 | 配置だけではActual/Desired不変。**新Field選択時のみ**Desired変更候補とMigration Preview。実保存は別資格（#1124/#1127/#3270） |
| ORG-MAP-04/05/08 | Field F1をDesiredだけ削除、Actualだけ完全観測で削除、partialで未観測 | 対応版のテーブル定義モデルからだけF1除去、UI要素は残して未Mapping/unknownを区別。SHO著者編集を上書きしない（#1126/#4125） |
| ORG-MAP-06/10、PIN-S0 | 同名別Field/Source改版・同Markdown別Subject・AI後着 | Source/Subject/Workspace/Selection Epochを再束縛できない限りApply拒否。SHOが文書を受領しただけではTEI Sourceが現在と一致する証拠にならない（#4515、SHO #50 QH-08） |
| ORG-BEH-02、ORG-CAP-07 | A2のknownAt遡及訂正によるFTE変化、別ClaimはD2のA3/A4だけに依存 | 全体FTEは再計算、D2の必要Read/Ruleが変更と交差しない場合だけ局所Claimを保持。全世界の同時点性は推定しない（#2582/#2510） |
| ORG-MAP-07、CX-16 | F1非表示だがSQL WHERE/NOT EXISTS/SELECT */NATURAL JOINが依存する | 出力列を表示しないことや明示参照0を無影響としない。未知Closureはunknown。Membership変更からC1責任・実Grantを推論しない（#4518/#2387） |

**SHO/AIとの接続の停止線**：同一Markdownであっても元TEIのSource、Subject、Revisionが異なる票を同一の編集対象として扱わない。SHOの既存Draft受領・保存・再開と、TEIの実Source再解決、問い相対のRequired/Consumed/Received、外部Harnessの同一Attempt実受領は別判定（SHO #50 QH-08〜10）。Studio Pin/AI Draft/SHO本文の一致はSource Patch、Host Grant、Provider Effectの証拠ではない。

**着手と帰属**：U0＝#1124/#1127のread-only UI分類・既存Data Context選択、U1＝#1126/#3270のDesired Field明示生成Previewと未Mapping、U2＝#4518/#2582の問い相対affected/scoped-unaffected/unknown、U3＝#4515/#3788のSubject Pin/Source Jump、U4＝#2586/#4539の隔離PG実Read（Reference入口）→通常`tei` CLI/Function Context（未接続）→Org C1固有Query（未実証）、#4125の実Schema/Host独立Readは別資格、U5＝SHO #50/TAIの元Source再束縛とRecipient受領。U0〜U3はP0実Readをブロックせず並行可。**受入は既存ORG-MAP/ORG-CAP/ORG-BEH/CX/PINを再利用し、新しい第二Expected Storeや共通Runtimeを作らない**。

**実証資格**：#4524の同Kind同ID重複、#4513のData索引、#4514のSource-only drift、#4534のCRUD衝突診断、#4551の明示CRUD-only/Unused派生修正はmain統合済み（範囲を限定した修正）。#4532のNode合成7件もCI成功を確認後mainへ統合済み。#4539の実PG read-only経路は正確なPR HEADの7 Workflow成功と隔離PGローカル試行の報告を確認してmainへ統合済み（squash `ee1af44c`）。これらを組織C1・通常`tei` CLI→Studio/SHO/Hostの製品E2Eへ昇格しない。本節の追加経路は設計・受入候補であり、実Go/Node試験・実GUI/DB/SHO/AI連携・Owner意味受理は **NOT RUN / NOT ACCEPTED**。

#### 10.11.4 組織上の担当とサブシステムの更新責任を分離する

**組織の担当変更≠ソフトウェアの境界変更**：OrgUnit（組織単位）、Membership/Occupancy、Case C1のhandlingUnit（案件取扱責任）、Subsystem（論理的なソフトウェア責務）、Function/DataのDirect CUD、実Target/DB配置、Host Principal/Grantは**別の軸**である。業務手順・図・CRUD・データベース配置が連動する場合も、それぞれの対応Ruleと変更Intentを明示する。#3250のSubystem分割案は候補・read-only評価であり、組織改編やGrant変更の実行器ではない。

| 変更事象（同一C1） | 保持するもの／再検証するもの | 行わない推論 |
| --- | --- | --- |
| P2がD1/D2を兼務し配賦A2/A3を変える | Membership occurrenceとFTEの版・有効/把握時点。Case C1担当、SubsystemのCRUD/Owner、Host Grantは別々に再評価 | 配賦率をC1移管率・実権限率とみなさない |
| Case C1のhandlingUnitをD1→D2へ正式に引き継ぐ | Case責任、業務Task/Function、通知/受領ACK、実EffectとReceipt | 技術SubsystemのCUD OwnerやDB配置が自動でD2へ移ったと扱わない |
| Subsystem IDや表示名だけが変わる | #3250の比較結果をID同一性候補・責務署名・Source位置ごとに保存。表示Labelの変更はモデル上の正式改名と区別 | Source/Function/Dataが同じならOwnerのIdentity/現Grantも同じと断定しない |
| 同じCRUDペアのOwnerや更新Effectの帰属が変わる | 直接宣言CUD・派生の可能性・実観測Effectを別記し、越境更新と依存Closure・新旧Ownerを再検証 | 比較図で越境更新0なら物理DB移設/正式業務責任移管が済んだと扱わない |

**既存の反証を再利用**：#3250のDraft #4528にあるSP-X23（Subsystem IDの全単射的付替え）とSP-X24（Snapshot Label変更）は、表示・再符号化と責務移管を分ける限定合成ケースであり、**Draft/未受理**。#4501 CX-01〜03（Caseの責任移管）、#4490 ORG-BEH、#1124の三要素CRUD、#2582/#4518の問い相対Dependency Closureを同一World/Revisionに並べ、ID集合や単純なCRUD件数の一致だけで意味継続・業務移管を認定しない。

**Guard・Effect**：異動・委任・承認・Case引継ぎなど業務操作では、必要な条件（Required）、現在満たされた条件（Supplied）、不足条件（Missing）を既存Rule/HostのPreflightで比較する。条件未確認は許可扱いにしない。開始前の現在Grant、対象・Rule/Source/Provider版、Action Intent、Attempt/Operation ID、外部Effect/Receipt、ACKを分離し、レスポンス喪失後の無条件再実行や過去の承認資格流用を禁止する。業務完了・コミット不明時の扱いは#1129/#1174等の既存Ownerに委ね、組織専用Guard/Workflow/Authorization Runtimeは作らない。

**最短受入と停止条件**：①r7の配賦/時点切替とCase C1移管の非連動を既存Expectedで確かめる、②同Subsystem ID/異責務・異ID/同責務候補の反例を既存#3250のRead-only比較と突き合わせる、③変更前後のCUD/Ownerを#4472の正しい索引と独立Expectedで照合し、未知依存ならunknown、④実Handoff/Grant/ACKは別Provider/Host試験に留保。図やSourceの一致、SHO説明、AI提案、CLIのPreview成功だけで実Effectを主張しない。実Org→Subsystem/Host接続の新しい実装・試験は**NOT RUN**。

### 10.12 Studio Desktopと組織C1の最短実装境界

**最優先製品目標**：TEI Studio Desktopを実際に構築し、TEI自身の標準Build Graphに基づいて同じStudio成果物を再構築する。組織C1はDesktopのモデル/CRUD/Provider/AI接続を検査する代表ケースであり、C1の全業務処理・権限完了をDesktop起動の前提にしない。Desktop/Wails共通部品は#1824、TEI Build GraphのStage0→Stage1→Stage2は#2831、三要素/組織のConsumerは#4501/#1124が既存Owner。Go単一バイナリの開発ツールを基本とし、外部Hostへ新しいTEI専用Hostを要求しない。

| 段階 | 最小の実動作・独立Expected | 禁止する実証の読み替え |
| --- | --- | --- |
| **D0 実Desktop** | **生成Workbenchを埋め込んだ**Wails/Go Desktopを同一HEADから実OSでビルド・起動。生成asset・Service接続・WebView2表示→IPC→Go共通Catalog/Inspect API→画面結果を同一Attemptで照合。Workspace/Source版とSelectionを記録 | 生成Workbenchのserver版Chromium成功＋通常DesktopのWebView2 smoke成功を足し合わせてもD0不成立。Go CLIセルフビルドCI・Bridge単体PASSも代替不可（#1824/#4541） |
| **D1 一つのData/CRUD表示と実Read** | 合成r7 C1のPerson/OrgUnit/Membershipを#4551の派生CRUDと表示。別例の受注Queryで#4539の実Provider Readを通常tei CLI→Studio Consumerへ接続し、Selected/Loaded/Invoked/Receivedを照合 | 異なる受注Queryと組織C1 Queryを同一実Readと合算しない。Org二時点実Query、coverage、現在Grantは未成立（#2586/#4501） |
| **D2 TEIによるDesktop再構築** | D0のStudio Source/Plugin/Toolchain/Target/Artifactを既存TEI Typed Build Graphに束縛。trusted stage0→実stage1 TEIプロセス→stage2 Builder実起動→Studio実Artifact・Image・Receipt照合 | Go launcherが2回ビルド、同digest、Plan PreviewのみをTEIセルフビルドと呼ばない。L1/L2/L3を別判定（#2831） |
| **D3 X-TEA三要素Plugin** | Data/Function/Business図、CRUD、参照/被参照、手作業TaskをStudioの業務向けPluginとして同じ受理Sourceから投影。Plugin無効時も汎用Studio操作は継続 | 汎用TEI Coreを三要素専用に固定しない。X-TEA原典との完全互換や実編集は別試験（#1124/#1125/#1251） |
| **D4 編集とSHO/AI連携** | Source-backed Subject Pin→参照元/先へ移動→Preview→版再照合。SHO著者編集保全とAI後着・Source ABA負例 | 固定Pin/描画EdgeからGrantや正式Relation IDを推測せず、Write/Effect/ACKを別確認（#4515/#2516/#4501） |

**生成UIの実Desktop受入への引渡し**：既存の生成Workbench server/Chromium経路と、通常Wails Desktop/WebView2経路は**二つの独立した合格**である。#4541のGD-0〜4を新Expectedとして再定義せず、生成asset（studio_generated、serverなし）→Desktop配布物→WebView2表示→IPC→Go Catalog/Inspect→応答の**同一Build/実Process/Selection**を一巡する。Wails旧版での隔離再現と新版への移行評価は分離し、成功した片方だけで他方の互換性を宣言しない。D0が成功してもStage1 TEIがStage2を生成したとはならない（D2別判定）。

**最短の証拠票**：exact HEAD、Source Revision、Selected/Loaded/Invoked/Received、Process Image、Build Input/Toolchain、Artifact digest、Consumer、Scope、受入状態を**観測したものだけ**記録。D0とD2を一つのself-build PASSに合算しない。CLI実Provider接続とDesktopセルフビルドは並行P0であり相互のHard Gateではない。既存#1180/#2514の独立Expectedと#1119のCritical Pathを再利用し、第二Builder/Parser/Runner/Host/Scenario正本は作らない。

**現況**：組織計画#4500 main統合（squash e5140059）。明示CRUD/Unused修正#4551 main統合（squash 1a2271d8）。外部解析REAの受領Reader #4547はmain統合済み（R1の限定資格）。型付き候補 Operation #4548は未マージ（R2）。どちらもAccepted Meaning、Current GrantやStudio独立Consumerでの実受領を証明しない。D0〜D4のDesktop実起動、Studio自己再ビルド、組織C1の実DB Query、X-TEA Plugin実受入は**NOT RUN**。


#### 10.12.1 Desktop自己再構築の反証可能な受入（D0/D2の追加条件）

D0/D2の判定では**同じ成果物が存在する**ことと**TEI自身がその成果物を作った**ことを分ける。stage0を信頼することは認めるが、stage2に至る主要な生成・選択・ビルド操作がstage1 TEIの実プロセスと既存Build Graphを経由した事実を、外部から照合する。名前やdigestが一致しただけの別製ビルド、前回生成したArtifactの流用はself-buildの証拠ではない。

| 判別する負例・受入ケース | 必ず区別して照合するもの | Gate・Owner |
| --- | --- | --- |
| **D2-SB-01** stage0のラッパーがstage2を直接ビルドし、stage1は起動しただけ | stage1のProcess Imageと起動引数、子/下流Invoke、Graph内のTask/Target、stage2出力のProducer経路を別々に観測。stage1が実Builderを起動・制御しない場合はD2不合格 | #2831、既存SB-C0/Receipt。新Runnerを作らない |
| **D2-SB-02** stage1のSource/Plugin/Toolchainを変えても、旧stage2のCache結果が残る | Source/Meaning/Plugin契約・Compiler/Generator/Target/OS/archの入力束、再計算判断、出力digestを照合。変化が影響し得るのに旧成果を再利用する場合は拒否 | #2831/#2600、build/Evidence/receipt再利用資格を分離 |
| **D2-SB-03** ビルドが途中失敗したのに旧Receipt・成功画面が出る | Attemptごとに開始/完了/失敗、成果物の実所在、要求入力と出力版、Partialと最終資格を分ける。失敗Attemptを成功に繰り上げない | #2831/#1180、実EffectはHost別資格 |
| **D2-SB-04** stage2のバイナリdigestが同じだがStudio起動/Go API応答がない | D0で試したOS/arch・起動可能なImage・Wails BridgeからGo共通APIへの実応答をstage2でも再観測。stage2 Artifactだけの一致はD0/D2両方のPASSではない | #1824/#2831、実OS受入 |
| **D2-SB-05** 署名・時刻・パッケージmetadata等でバイト一致しない | 決定的なBuild Manifest/Source/Toolchainと、実起動・最低限のUI/API挙動の差を別欄に記録。byte同一性が必要な対象では厳密比較し、非決定的な包装差だけを独立に説明してから同等性を判断する。単にdigest差を無視した自動PASSは禁止 | #2831、比較対象と許容差分を事前宣言 |
| **D2-SB-06** D0ではX-TEA Pluginが有効だがstage2でPlugin閉包欠落・版不一致 | Core、Plugin Manifest、必要契約/依存/配置、三要素Data/Function/Business/CRUDのread-only観測を別確認。Plugin無効時も汎用Studioが動くことを判別 | #1824/#1251/#1124、D3はD2完了の一律Hard Gateではない |
| **D2-SB-07** Sourceや依存がA→B→Aと戻る間に旧Selection/Expected/Build Evidenceを再利用 | 入力byteが戻った場合でも途中の変更・Toolchain/Plugin・World/Meaning資格と現在のConsumer観測を再照合。古い承認を自動復活させない | #2600/#2831/#2514、Result/Meaningの資格は別 |

**比較を先に狭くする**：最初は一つのOS/arch、一つのWorkspace、一つの最小Desktop起動、一つのGo API応答、一つのread-only三要素投影に限定する。OS横断、署名配布、全Plugin/全Target/全業務Functionの完全性をP0の入場要件にしない。P0の完成条件を段階ごとに明確にし、D2の成立後にD3/D4を順次拡張する。ただし#2586の実Provider→通常CLI接続はDesktopとは**並列のP0**であり、双方を相互待ちにしない。

**取り扱う証拠の範囲**：これは#2831/#1824の既存テスト・Receiptへの受入要求の追加であり、新しいCore Model、専用Self-build Runner、第二Receipt Storeを定義しない。実stage1→stage2プロセス、実OSでのstage2 Studio起動、Pluginの実閉包・失敗時復旧・UI/Source同一性の試験は**今回NOT RUN**。それらを確認するまでは、#4551のCRUD派生Go PASSも#4539の受注Provider Read PASSもDesktop自己再構築のPASSと呼ばない。
