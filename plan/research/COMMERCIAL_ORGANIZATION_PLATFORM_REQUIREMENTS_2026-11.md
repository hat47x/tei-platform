# 高度組織を扱う商用製品の要求比較とTEI適合性監査（2026-11-06）

- **区分**：調査記録・採用前Requirements Probe。商用製品の動作をTEIで再現済みと主張しない。出典は各社が公開した公式製品文書／ヘルプ／リリースノート。製品の内部実装や正式契約を推測しない。
- **位置付け**：組織設計 [PR #2389](https://github.com/hat47x/tei-platform/pull/2389) 第1〜73節、[#2386](https://github.com/hat47x/tei-platform/issues/2386)、[#2510](https://github.com/hat47x/tei-platform/issues/2510)の派生調査。正式なモデル/実行/認可/Providerは既存Ownerに所属する。
- **比較視点**：人事組織・給与/職位・複数雇用・企業集団、業務統制、顧客/テナント運用、組織設計シナリオ。組織Dataは原則通常のData（第2.4節）。特別な保護が必要なResourceと別Tenant境界・実行Grantは明示したときだけ制約を強制。
- **事実と判断**：各社仕様＝公式公開資料で直接確認、TEIの採否/充足＝現在の設計・実Source/Referenceを照合した**暫定的判断**。各社の契約版/設定/アドオンやサポート範囲により実動作は変わり得る。

## 1. 製品別に確認した要求（一次資料への入口）

### 1.1 Workday HCM / Workday Adaptive Planning

1. [Superior and Subordinate Organizations](https://doc.workday.com/admin-guide/en-us/manage-workday/organizations/manage-organization-concepts/concept--superior-and-subordinate-organizations.html) は Supervisory Organization の上位親を**一つだけ**とし、子は複数持ち得る。これを全てのOrg Relationへ一般化せず、人事の特定階層が選択する多重度制約として扱う。
2. [Setup Considerations: Organizations](https://doc.workday.com/admin-guide/en-us/manage-workday/organizations/ivq1568755825744.html) は組織改編（新子組織、上位変更、無効化、分割、人員移動）とCustom/Multi-organization、Reporting/Matrix Organization、組織ごとに異なるBusiness Process設定を扱う。階層とマトリクスは別Relationship Role。
3. [Staffing Models](https://doc.workday.com/workday-education/en-us/course-manuals/hcm-core-for-administrators/staffing-models.html) はPosition ManagementとJob Managementを区別し、空席/充足、承認済みPositionでの採用、職位の組織間移動、職位履歴等の要件を示す。職位のIdentityとPersonの配置Occurrenceを混同しない。
4. [Advanced Business Process Configurations](https://doc.workday.com/workday-education/en-us/course-manuals/hcm-core-supplemental-for-administrators/advanced-business-process-configurations.html) は組織別・条件規則による承認経路、Approval Chain、Step Delay、集約承認を扱う。Org階層があるだけでは実承認されない。
5. [Workday Adaptive Planning 2026R1 Org Design and Scenario Modeling](https://doc.workday.com/adaptive-planning/ja-jp/what-s-new/-/2026r1-/2026r1-planning-for-hcm-and-financials/org-design-and-scenario-modeling.html)（2026-03-14本稼働との公式リリース記載）は、稼働中HCMに作用しない複数What-if、Position/階層変更、Headcount/FTE/Cost/Span/Layers、比較・関係者レビュー・承認への送出を提供する。**Model Scenario・Published Org・実HCM Effects・Actualsを四つの証拠として保持**。財務数値の保存則をあらゆる変異へ強制しない。

### 1.2 SAP SuccessFactors Employee Central / Position Management

1. [Synchronization Between Matrix Relationships and Job Relationships](https://help.sap.com/docs/successfactors-employee-central/implementing-position-management/synchronization-between-matrix-relationships-and-job-relationships)（2H 2026版の公式ヘルプ）では、Matrix Manager、HR Manager、Additional Manager、Custom Manager、Second Manager等の型があり、Position側Matrixと従業員Job Relationshipの同期条件が異なる。単一Parent + 汎用Manager だけで全てを説明しない。
2. [Relationships Between Managers and Employees](https://help.sap.com/docs/successfactors-platform/implementing-role-based-permissions/relationships-between-managers-and-employees) は標準Manager/Second Managerと多重Matrix/Custom Managerを区別し、Global AssignmentではHome/Host側のManager/HR Managerが別のRoleになる。
3. [Manage Positions](https://help.sap.com/docs/successfactors-employee-central/employee-central-scope-items/manage-positions) と [Positions that Allow Multiple Incumbents](https://help.sap.com/docs/successfactors-employee-central/implementing-position-management/positions-that-allow-multiple-incumbents) はPositionの存続とIncumbentの多重度を別に設定し、空席・複数在職者を扱う。一職位一人や一在職一職位を普遍規則にしない（既存ERP連携先での制限にも注意）。
4. [Latest People Profile](https://help.sap.com/docs/successfactors-release-information/8e0d540f96474717bbf18df51e54e522/67b091e109b94ad2ae42bff1c38738ce.html) では同一Personの複数Employment（通常、Global、Concurrent、Contingent）を切り替え、Employmentごとに有効/過去や表示できる項目が異なる。Profile UIでのSelectionはPerson/Employmentの同一性やAuthorizationを自動変更しない。
5. [Employee Central Overview](https://help.sap.com/docs/successfactors-employee-central/implementing-employee-central-core/what-you-need-to-do-after-migrations) はDepartment/Business Unit等のCompany-shared Foundation Object、Effective-dated data、Association、Workflow、Custom Generic Objectを区別する。名前が同じでもRole/Scope/Source of Recordを混同しない。
6. [Impact of Multiple Employment on Succession Planning](https://help.sap.com/docs/successfactors-employee-central/implementing-employee-central-core/impact-of-multiple-employment-on-succession-planning) は後継者候補がEmployment/User IDに結び付くことや、異Employment間のNominationが自動統合されない例を示す。Personが同じだから過去の承継候補やRoleを自動統合しない。

### 1.3 Oracle Fusion Cloud HCM

1. [Assignments](https://docs.oracle.com/en/cloud/saas/human-resources/fawhr/assignments.html) はWork Relationshipと一つ以上のAssignmentを区別する。一つの雇用関係に複数Assignmentを許す構成があり、閲覧もAssignment-level securityと責任範囲に依存する。
2. [Transfer Actions](https://docs.oracle.com/en/cloud/saas/human-resources/fawhr/transfer-actions.html) はLocal Transfer（同一Legal Employer内・Assignment identity維持）、Add Assignment、Global Transfer/Temporary Assignmentを区別する。組織異動が常にPerson/Employment/Assignmentの新規IDを意味するわけではない。
3. [Manage Date Effective Objects](https://docs.oracle.com/en/cloud/saas/human-resources/farws/Manage_Date_Effective_Objects.html) はLogical Recordと過去・現在・未来のPhysical/Change Rowを区別し、Update、Correction、End Date/Delete Changes等を使い分ける。SourceのvalidAtとTEI内のknownAt/accepted/publishedは同一概念ではない。
4. [How You Make Multiple Updates in One Day](https://docs.oracle.com/en/cloud/saas/human-resources/faucf/how-you-make-multiple-updates-to-date-effective-objects-in-one.html) と [Effective Sequencing](https://docs.oracle.com/en/cloud/saas/human-resources/faucf/examples-of-effective-sequencing.html) は、同日複数UpdateをSequenceで保持する対象と、Correctionとして過去の同日中間状態を上書きする属性の差を例示する。**同一有効日・二操作を同一状態に圧縮することは一般化できない**。また、Oracleの履歴に残らない中間値をTEIが復元可能と主張しない。

### 1.4 Microsoft Dynamics 365 Finance / Power Platform Dataverse

1. [Organizations and Organizational Hierarchies](https://learn.microsoft.com/en-us/dynamics365/fin-ops-core/fin-ops/organization-administration/organizations-organizational-hierarchies) はLegal Entity、Operating Unit、Team、複数のHierarchy Purpose（法定報告、購買統制等）を区別する。あるHierarchy PurposeのParentを別の権限Roleへ自動継承しない。
2. [Plan Organizational Hierarchy](https://learn.microsoft.com/en-us/dynamics365/fin-ops-core/fin-ops/organization-administration/plan-organizational-hierarchy) はLegal EntityとOperating Unitの機能差、法人間Shared Master、Multi-company ProcessingとHierarchyによる対象法人の選択を扱う。
3. [Modernized Business Units Security](https://learn.microsoft.com/en-us/power-platform/admin/modernized-business-units-security) は利用者の所属Business UnitとレコードのOwning Business Unitを区別し、**別Business UnitのSecurity Role**を持つ人が対象Dataへアクセスし得る。一つのUserの所属をData OwnerやCurrent Grantの根拠へ無条件にできない。
4. [Create or Edit Business Units](https://learn.microsoft.com/en-us/power-platform/admin/create-edit-business-units) はBusiness Unit変更時のSecurity Role・Owner Recordの移行が環境設定により変わることを示す。異動による所有/Grantの引継ぎを黙って固定しない。
5. [Teams in Dataverse](https://learn.microsoft.com/en-us/power-platform/admin/manage-teams) にはRecordを所有するOwner Teamと、所有しないAccess Teamがある。Membership/Team Role/Record Owner/Access権限の分離が必要。
6. [Hierarchy Security](https://learn.microsoft.com/en-us/power-platform/admin/hierarchy-security) はManager/Positionの階層権限を既存Security Role等の上に重ねる。権限継承は**選択された認可機構**の機能であってOrg Relationの普遍公理ではない。

### 1.5 ServiceNow Domain Separation

1. [Exploring Domain Separation](https://www.servicenow.com/docs/r/platform-security/c_DomainSeparation.html) は同一Instance内でData、Business Logic/Process、Adminをドメイン別に分離し、全体で共有するデータ/プロセスとの共存、別Instanceを選ぶ境界も明示。Domainは部門/法人と同一ではない。
2. [Domain Separation Hierarchies](https://www.servicenow.com/docs/r/ja-JP/platform-security/bp-domain-sep-hierarchies.html) はParent-child、Contains（Dataだけ）、Visibility（明示的可視性）を別種の関係とし、デフォルト/グローバルの配置ミスに注意する。Org hierarchyのParentをTenant visibilityへ自動再利用しない。
3. [Visibility and Contains Domains](https://www.servicenow.com/docs/r/platform-security/c_DomainVisibility.html) は可視性を個別に付与でき、ContainsがBusiness Processの共有まで意味しないと明示する。
4. [Domain Separation in GRC: Policy and Compliance](https://www.servicenow.com/docs/r/governance-risk-compliance/policy-and-compliance-management/domain-separation-pol-comp.html) はUI/Cache/Reporting/Rollup/Aggregationまで境界検証が必要な例を示す一方、Business Logic Separationの支援は全ての機能で完全ではない。**Domain separationを選択したという理由だけで全Plugin/機能経路が隔離済みだと判断しない**。

### 1.6 Orgvue / Workforce Scenario Modeling

1. [Orgvue Platform](https://www.orgvue.com/platform/) はAnalysis→Design→Plan→Monitorの連続性、Position-levelのRoles/Reporting/Cost、組織階層のSpans/Layers/FTE・複数Scenario比較を扱う。
2. [Organization Modeling](https://www.orgvue.com/solutions/organization-modeling/) はPosition-levelと高レベルの組織設計を区別し、変更に伴うCost/Role/Hierarchyを分析する。
3. [Finance and HR Transformation case study](https://www.orgvue.com/resources/case-studies/bringing-finance-and-hr-together-to-plan-a-2-year-transformation-and-track-its-implementation/) はHR/Finance Source Dataを合わせて計画を承認後、Quarterly Actualsと比較する事例。**Planの数値、Approved Target、施策実行、Observed Actualを混同しない**。広告的な成果数値はTEIに再現可能な性能・便益として移植しない。

## 2. TEIカバレッジ判定（意味・合成・実製品を三段階で分離）

凡例：**意味○**は既存Data/Relation/Rule/Business等の設計方針で表せる候補があるという意味。**合成一部**は特定のSynthetic Fixture/Referenceの動作があるという意味。**実製品未**はTEI Product Function/Provider/Current Host Grant/Effectの同一Source試験が未達。○は正式Owner受理済みや全商用機能と等価の印ではない。

| 要求 | 商用製品の主な根拠 | TEI意味の対応 | 既存の限定Evidenceと未充足 |
| --- | --- | --- | --- |
| CP-01 階層Roleごとの単一親／複数親、循環、History | Workday/SAP/Dynamics | 意味○ 第2.3・3節/#1126 | 合成で関係/時点の局所ケース。Native E1は構造表現中心。Role別正式検証/Provider E2未 |
| CP-02 複数Manager/指揮命令、Home/Host/Project | SAP/Workday | 意味○ 第2/5節、#2386/#2510 | 合成Membership A2/A3は同一Person多Occurrence。標準Manager種別をCore固定しない。選定Rule/実権限未 |
| CP-03 空席Position・複数Incumbent・PositionとEmployment独立 | Workday/SAP | 意味○ Position/Occupancy Profile | 合成M1/O1/O2あり。Position型ごとの在任多重度/予約/空席、実保存/更新は未 |
| CP-04 Person/Employment/Work Relationship/Assignmentを区別 | SAP/Oracle | 意味○ 拡張Profile候補 | Native E1のPerson/MembershipだけではEmployment/Assignment全体未。複数就業連携は未 |
| CP-05 Local/Global TransferでどのIdentityを保つか | Oracle/SAP | 意味○ Change reason/Succession等 | 前/後IDの不変・変更をOperation別に試すFixture未。外部HR同期未 |
| CP-06 valid/known/publication/decision、Update vs Correction・同日Sequence | Oracle/SAP | 意味○ 第3/4節 | 合成valid/knownの対照あり。同日複数Update/Correctionで中間値欠落、SequenceとknownAtの差の独立Fixture未 |
| CP-07 法人/Operating Unit/Teamと用途別複数階層 | Dynamics/Workday | 意味○ 第2/13節 | 一軸の親制約を全Relationへ流用しない方針。Finance/HCM/購買の異目的RoleとShared Masterの製品検証未 |
| CP-08 組織別Business Process条件・承認候補の再評価 | Workday/SAP | 意味○ 第5・67〜69節/#1125 | 受理済みRule/Current Grant/実承認Effect/Decisionの縦断E4未。候補≠権限 |
| CP-09 User所属BU、Owning BU、Owner Team、Access Teamの分離 | Dataverse | 意味○ 第13・70〜72節 | 現Host/Resource/Record OwnerのモデルBindingと実更新未。Org MembershipだけでAccessしない |
| CP-10 Tenant/Business Org/Deployment/IdPを1:1にしない | Dynamics/ServiceNow | 意味○ 第70〜72節 | File Isolation #2052/PG固定Grantの限定実証。Org NativeとTenant Authorityの結合E2未 |
| CP-11 Parent-child/Contains/VisibilityとBusiness Process適用の区別 | ServiceNow | 意味○ typed Relation + Plugin Contract | 実Domain Provider/Scope/Process統合未。TenantのRelationshipを権限の自動継承にしない |
| CP-12 Tenantなし/誤Default Domain/Globalへの誤配置 | ServiceNow | 意味○ 第70/73節候補 | No Tenant時はResourceごとのdefault denyかLocal Profileを区別。Production quarantine/route未検証 |
| CP-13 Shared Master とPrivate Relationの横断Query/Disclosure | Dynamics/ServiceNow | 意味○ 第70〜73節 | SQL非公開ORDER推論の既知反例あり。Count/EXISTS/Expand/Export/Cache/AI外部先は未 |
| CP-14 What-if Scenario、Plan Approval、PublishとActual Effectsの分離 | Workday Adaptive/Orgvue | 意味○ 第69〜72節 | #2779 Synthetic What-if Draft等はProduct可用性を保証しない。Scenario→正式公開→実Effect/Receipt→Actual照合未 |
| CP-15 FTE/Headcount/Cost/Span/Layersの粒度と依存／計画実績差 | Workday Adaptive/Orgvue | 意味○ Measure/Rule/Query/Read Scope | 合成FTE A1-A4, distinct PersonとJoin Fanout。実財務Cost/Position vacancy/予測系/対象集合完全性は未 |
| CP-16 再編・職位移動・未決Caseと役割の引継ぎ | Workday/Oracle/Dataverse | 意味○ 第67〜69・72節 | 後継OrgからCase/Grantを自動継承しない。Org Caseの実Persistence/Version/Effect/Receipt未 |
| CP-17 実認可の継承・委任・代理、同意・失効・移動後のAccess | Dataverse/SAP/ServiceNow | 意味○ 第5/70〜73節 | ApplicationのAuth CapabilityとDelivery/Host Current Grantを分離。特定Providerの実強制・失効競合未 |
| CP-18 Source Of Record/履歴・複数Provider・Export/Importと整合 | SAP/Oracle/Orgvue | 意味○ 第4/66/72節 | Field/Relation/TimeごとにSourceを別。正式Orgの実Import+Published snapshot、外部HRIS/複数Writerの実Receipt未 |

**判定**：構造意味を表現するための一般モデルは豊富で、全18項目について **既存方針と整合する表現候補**を確認した。ただし「全18項目が正式に解決済み」「商用製品相当の機能が揃う」という意味ではない。少なくともCP-04/05/06/08/09/11/12/14/15/16/17/18は新たな専用negative fixtureと実Provider/Owner受理を要する。商用製品の業務固有機能をTEI標準Coreに直接複製しない。

## 3. 設計を成熟させる独立Expected：ORG-CM-01〜12（NOT RUN）

既存の#2510組織Corpus、#2596合成Source、#2681 Native E1、#2779 What-if Draft、#2052 File Isolation、#1656/1769 Scoped PG、#2586 Product Readerを**異なる証拠資格**で再利用する。新しいMaster SchemaやSynthetic Runnerの重複を防ぎ、商用製品の入出力結果を無断でコピーしない。

| Case | 目的・変異 | 独立Expected / Stop |
| --- | --- | --- |
| ORG-CM-01 | Supervisory treeではD2の同時親2つ、Matrix Roleでは2つの管理線 | 前者はRole固有Cardinality違反、後者は許容。両者を1本の万能Parentに畳まない |
| ORG-CM-02 | Position M1は空席、その後O1就任。同期間にM2は複数Incumbent許可 | Positionは0人でも存続。多重度はPosition型の受理規則に依存 |
| ORG-CM-03 | Person P2にEmployment E1/E2とAssignment J1/J2、Home/Host Manager別 | Person/Employment/Assignment/Manager Role/Current Permissionを別識別。合算人員数は定義粒度で変化 |
| ORG-CM-04 | Local Transfer J1は同一ID、Global Transferは新Employment/Assignmentの対応表が必要 | OperationとOwnerが選んだIdentity規則に従い、氏名/Person一致でMapping自動確定しない |
| ORG-CM-05 | 同一valid dateに2回Update、その後Correctionが中間像を上書きした外部Source | Sourceの実保存/Sequence/recorded factsだけ再構成。消えた履歴を復元済みと主張しない |
| ORG-CM-06 | 同じLegal EntityがFinance hierarchy/Procurement hierarchyで別の親、Business UnitとTeamでも別関係 | Hierarchy PurposeごとにParent/継承の適用範囲を制限。法定集計Roleを購買権限へ流用しない |
| ORG-CM-07 | Manager移動、Owning BUとOwner Teamは不変、別Roleがある | 管理職/所属異動だけでRecord ownership/Private Data/Current Grantを変更しない。実Actionは再評価 |
| ORG-CM-08 | Org-specific Approval Chainの申請中にMatrix ManagerとPolicyが変化 | 候補/開始時固定・Step時再評価/Ownerの受理/現在Grant/確定効果を別々に判定 |
| ORG-CM-09 | Domain ContainsでDataは共有するがBusiness Processは共有しない | Data visibilityをProc/Action execution permissionへ昇格しない。Unsupportedは拒否 |
| ORG-CM-10 | Private RowにTenant Bindingなし。Global/Defaultへ誤配置し得る | 当該ResourceがTenant-privateならquarantine/deny。公開Shared Masterは別の適格条件で扱う。Local StudioをTenant必須にしない |
| ORG-CM-11 | 2つのWorkforce What-ifを比較、人数とFTE/Costは異なる、まだ正式公表なし | 比較/選択/承認候補/正式Publish/実Provider Effect/Actualを別。Subset不完全なら全体人数・費用はunknown |
| ORG-CM-12 | Org split、TenantのDB移設、Old Writer commit応答喪失、月次Actualが遅着 | Org後継だけでCase/Grant/Receiptを自動移管しない。旧新物理TargetとIdempotencyの証拠不足ならunknown、二重Dispatchを避ける |

**試験順位**：第1段はCM-01/02/03/05/06/09/11のread-only意味反証。第2段は現行#2586実Product E2aと複数Source/ScopeのCM-07/10。第3段は#1174実Effectと#1130によるCM-08/12。**最初のProduct E2a実Call Pathを飛ばして商用フル機能を作らない**。

## 4. 統合してよい成熟した判断／Ownerに残す判断

**成熟した横断制約として計画へ反映する候補**
- Relationは人事監督系、Matrix、Finance、Procurement、Ownership、Tenant Visibility等の**用途/Roleごとに多重度、循環、継承、時点を規定**。一つのGraphで表示できても同じRelation型ではない。
- Person、Employment、Work Relationship、Assignment、Position、Occupancy、OrgUnit、LegalEntity、Case、Tenant/Deployment、Principal、Record Ownerは必要に応じ別Identity。最初から全案件に一律必須Schemaとしない。
- ProviderのDate Effective Rowと、TEIのvalidAt/knownAt/Published Revision/Execution Receiptは別。**同日Sequence/Correction**など外部履歴の残存範囲をEvidenceに含め、欠落履歴を創作しない。
- Org membership/manager/position/owner/tenant treeの関係だけでCurrent Grant/Business Approvalを発行しない。Provider/Targetが持つ実権限は別の契約。
- **What-ifと正式Source/実HCMのEffectを分離**。Headcount・FTE・Cost・Span・Layersは対象・粒度・Period・Join依存が証明されたClaimだけ成立し、PlanとActualを別々に観測。
- Domain/Tenantの共有/階層可視性/Process共有は別のContract。DataがPrivate指定なのにScope不明なら拒否/隔離し、通常の組織DataやLocal Studioを一律Tenant-privateにしない。

**人間・Ownerの業務判断として保留**
- 業務Profileにどの雇用形態/Manager Role/Transfer Codeを標準例として含めるか。商用の固有列挙をCoreへ固定しない。
- Position多重在任・Primary Employmentの選び方、FTE/Cost配賦・通貨・時点・算定の契約。
- 「Correctionで消えたSource履歴」をどの外部資料/監査から再構成可能と扱うか。
- Org/Tenant共有時の具体的Recipient・管理者例外・Parent Domain Visibility・User-facing denial。
- Business ProcessのRoute最終決定時点、進行中申請の移管/再承認、SoD/委任の許容。
- どの商用製品の完全相互運用性を最初のTEI Projectの対象とするか。ここでWorkday/SAP/Oracle/Dataverse/ServiceNow標準Connectorsを採用しない。

## 5. 既存Issueへ引渡す差分と証拠ゲート

| 優先 | 渡す差分・正しい停止条件 | Owner |
| --- | --- | --- |
| P0-Meaning | CM-01〜06,09,11の関係/時間/職位/雇用/Hierarchy Purpose/Scenario Previewを既存Corpusで独立Expected化。標準Data/Relation/Ruleに無い意味差だけ抽出 | #2386/#2510/#1126/#2593/#1124 |
| P0-Product | Native Case→Function Context→実Provider Row/Field/Scope→Consumerの一件E2aを先に通す。商用差を理由に第二Reader/Hostを作らない | #2586/#2506/#2622/#2611 |
| P1-Workflow | Manager/Approval Routing/Transfer・Business Record Ownershipと現行Host Permission、Commit/Receiptを別検収 | #1125/#1130/#1174/#2564 |
| P1-Tenant | ServiceNow Domain Parent/Contains/Visibility、Dataverse BU/Owner/Role等の反例を既存Tenant/Scoped Read/Disclosureへ接続。全Pluginが自動分離されると仮定しない | #2020（closed記録）/#1656/#1178/#1043/#951 |
| P1-Planning | Workday/OrgvueのScenario/Measure/Actualを既存#2779 What-ifと#2514独立Expectedで比較。Plan/Approved/Actualの受渡し証拠を固定 | #2510/#2514/#2593/#2516 |
| P2-External | 各社公式API/Extractの具体範囲、History Coverage/Retention/Field Authorityを一つのCaseへ限定し、actual Export/Import/Resultと再読取 | #1807/#2555/#1163 |

**資格**：本文は公式公開資料の読解＋既存TEI Repo/Issueの静的照合。新実験・各社テナントへの接続・課金SKUの検証・製品相互運用・Owner意味受理は行っていない。実Provider/実Host/実外部システム・TEI E2/E4を合格としない。設計の表現可能性はConformance証拠ではない。既存ADR-0036/0043/0044/0019と#1119を変更しない。

## 6. Dataverse階層認可の追加例外：到達可能性とPrivilege/Record Accessの二段階（2026-11-06）

一次根拠：
- [How access to a record is determined](https://learn.microsoft.com/en-us/power-platform/admin/how-record-access-determined) は**対象Table/ActionのPrivilege Check**と、Owner／Role／Shared／Hierarchyによる**Record Access Check**を別に実施。いずれか一つの上位階層Pathだけで全操作のGrantを導かない。
- [Hierarchy security to control access](https://learn.microsoft.com/en-us/power-platform/admin/hierarchy-security) はManager階層とPosition階層を選べ、前者は本人・報告先のBusiness Unit制約、後者はCross-BUを許す構成がある。Position階層の**直接上位者はRead/Write/Append/AppendToの階層候補**になり得る一方、**間接上位者はRead-only**。適用深さ、対象Table、既存Security Roleが条件であり、Hierarchy機能は既定無効。
- [Dataverse Teams](https://learn.microsoft.com/en-us/power-platform/admin/manage-teams) は、RecordとSecurity Roleを所有するOwner Teamと、Recordは所有せず明示共有されたRecordにアクセスするAccess Teamの区別を示す。
- [Modernized business units security](https://learn.microsoft.com/en-us/power-platform/admin/modernized-business-units-security) ではUserの所属BU、RecordのOwning BU、付与されたBU Security Roleを独立させる。

**TEIへの補正**：既存CP-09/17の「所有とGrantは別」をさらに、①採用Roleでの関係到達、②対象Table/Actionの現在Privilege、③Record Owner/Share/Access Teamなど実Record Permission、④実Target/Provider Enforcement、⑤Consumer Disclosureへ分解する。Read-onlyの根拠は単純なGraph距離の普遍則ではなく、選択されたVendor Profile/Policyと対象Actionに相対する。Optionalな組織Linked Policyを**広域適用→用途/Record例外**で構成する場合も、除外はそのPolicyのnot-applicableであって他Policy/HostのGrantを発行しない（組織計画§58）。

[Draft PR #3019](https://github.com/hat47x/tei-platform/pull/3019)のCM-13/14は、Manager vs Position Scope、direct vs indirect Position access、Table Privilege不足、Owner Team vs Access Teamのread-only合成対照を追加。**本試験の合成「candidate」や「not-admitted」はDataverseの実認可結果ではない。** 実TEI Host/Provider・Microsoft環境・本番ACL/Role/Sharedの検証にはならず、Owner選定のDomain PolicyをCoreへ追加する根拠にもならない。

### 6.1 適用対象と反証

| 追加検討 | 誤った実装 | 受入の最小反証 |
| --- | --- | --- |
| CP-17a Table Privilege vs Record Scope | Hierarchy上位なら対象TableのWriteも可能 | Table Write無しならこの経路からWrite候補へ昇格しない |
| CP-17b Manager vs Position | 同じGraph Ancestorなら同じScope | Manager BU境界を満たさなくてもPosition Profileでは同距離のRead候補があり得る |
| CP-17c Direct/Indirect path | 二階層上なら直接上位と同じWrite | 間接Position上位の階層由来Write候補は拒否し、Read候補のみ。別のRecord Share権限は独立 |
| CP-09a Owner Team vs Access Team | ReadできるTeamは必ずOwner | Access TeamのRead候補を認めてもRecord Ownerは不変。明示Share・Table権限不足は拒否 |
| CP-10/17d Tenant/Plugin Binding | 同じOrg Role・PluginをTenant A/Bへ導入→Bへ自動権限 | 配布/構成の同一性はCurrent Host Grantでない。実Scopeが異なれば別試験 |

**状態・実測（2026-11-06）**：公開公式仕様の比較＋固定HEAD c209fa8b4e803a9a9698a42648ec087add56782a の[組織Synthetic Node CI #42196320734](https://github.com/hat47x/tei-platform/actions/runs/42196320734)で、CM-13/14新3件が個別ok、既存214＋追加3＝**217/217 PASS、0 FAIL**とログ確認。これは特定設定を合成したread-only候補判別のみ。実Dataverse/TEI Product/Host Policy/現認可/Provider実施行、Owner受理はNOT RUN。
