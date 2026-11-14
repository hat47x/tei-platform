# 組織C1：二時点保存・判断候補・Consumer接合の横断受入補遺（2026-11-14）

> 状態：**設計・採用前**。この文書の作成は製品実装、独立Expected実行、Owner意味受理、Host認可を意味しない。主計画は [ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md](ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md)、意味Ownerは #2386。既存の #4490 ORG-BEH、#4501 CX、#2510 org-synthetic-r7 を再利用し、第二のScenario/Expected正本・組織専用Runtimeを作らない。

## 1. この補遺で成熟させる判断

- 組織の基本型は既存の Data・Relation・Key・Rule・Query・Persistence・OperationSurface へ束縛する。通常の部門/所属/人数に専用ACLやTenant境界を追加しない。保護すべき特定Field、別Tenant/HostのRead、外部Consumer開示と実Action認可は別に検証する。
- 「同じ所属関係」「その関係の過去の記録版」「DBの保存行」「今回の実Read」「実際の承認・Effect」をそれぞれ独立に追跡し、同じID/表示/byte/World Fingerprintだけで一致を推定しない。
- 意味を採用する業務Owner、現行Grant/Effectを担うHost、永続化のDB Owner、ConsumerごとのDisclosure Ownerを一本化しない。必要な境界にのみ契約を追加する。
- 優先順位は **#2586 通常TEI CLIの最初の実Provider Read** と **#1824/#2831 Studio Desktopセルフビルド** が先行/並行。組織C1の全機能完成をこれらのHard Gateとしない。

## 2. ID・時点・版の最小契約

| 対象 | 識別・時点 | 絶対に短絡しない解釈 |
| --- | --- | --- |
| Membership occurrence | 業務上同一の所属関係ID（例 A2） | 異動で同じ人の別所属（A3）まで同一Occurrenceに畳まない |
| Record version | 旧/新の解釈を表す記録版の識別と発生原因（訂正/取消し/追加） | version IDを新しいMembership IDへ自動置換しない |
| Physical row | Provider/DB/表/PKおよび保存行の実在 | 物理PK=業務IDを永続必須としない |
| validAt / knownAt | 業務有効時点と、その時までに把握・記録された事実の採用時点 | 遡及訂正で過去の確定Receiptを消去しない |
| observedAt / cohort | 今回取得したDB/Provider側観測・snapshot・scope・completeness | knownAt同一でも別Providerが同じRead Cohortとは限らない |
| evaluatedAt / ruleRev | Ruleを評価した時点と版 | Queryのas-ofを現在Grant判定に流用しない |
| effectAt / receipt | 実HostのAction試行と確定/不確定な結果 | 現在の組織値が元に戻っても旧Receipt/Grantを再利用しない |

正しい二時点照会は必要な範囲で **valid_from <= validAt < valid_to** と **known_from <= knownAt < known_to** の半開区間とする。NULLの上端は仕様により無期限を意味する場合だけ扱う。時刻境界の表現、時間帯、上端の有無もSource契約で固定する。二時点の保存が必要ない小規模組織に全履歴表を強制しない。

**現行の具体的なGAP**：Draft #4567の実PG16検証は `as_of_membership` の時点WHERE、物理 `public.org_memberships`、結果Mappingの限定成功。現行 `membership_id` 単独PKでは同一Occurrenceの複数known版を同一表に共存できない。「正しい述語」と「版共存の保存」「訂正後の歴史再読込」「正式Function Context」は別受入である。Draft #4566は同じQuery/Mapping修正が重なるため、二重に取り込まず差分比較する。

### 2.1 DB Ownerが比較する物理方式（未決）

| 候補 | 比較する受入 | 棄却する例 |
| --- | --- | --- |
| 安定Occurrence ID + 独立version/row IDを持つ版別行 | 同一Occurrenceの旧/新版が共存し、両時点から正しい版が読める | 同時点に不適切な二版が選択される・旧版が上書きされる |
| Occurrence + versionの複合キー | 業務キー/版キー/物理PKの違いと一意性を説明できる | known_fromだけを版キーとし、訂正衝突・同時刻の多版を区別できない |
| 現行行+履歴行 | 現在と過去のQueryが同じ契約で再現される | 境界を跨ぐと過去版が脱落・二重計上される |

- 保存方式は #1126/DB Ownerの判断とし、TEI Coreの新kindや共通Temporal Engineを先に追加しない。範囲重複・循環・同時版の排他/両立条件は、Relation軸・業務Ruleごとに検査する。
- **同一Occurrenceの矛盾する有効版が同一(validAt,knownAt)で同時選択**されたとき、最後の一行を採用せず未解決/制約違反。**異なるMembership occurrenceの同時兼務**は通常の正例。
- 必須の実Read検収：同一DBへ版を共存保存→同じvalidAtと異なるknownAtで実SQL+params/rowsetを取得→Source/Scope/Completenessを突合→手書きExpectedで比較。ケースごとにDBを初期化して一行だけ入れ替える試験は「版共存」の証拠にならない。訂正・消滅の表現は期間閉鎖・明示取消し等を候補比較し、黙示の不存在から推論しない。

## 3. r7を使う二時点・数量・非出力依存の受入

既存 `reference/org-semantic-views` の **org-synthetic-r7** をそのまま使う。validAt=2027-06-15、knownAt=2027-07-01 は P1/D1 A1=1.0、P2/D1 A2=0.6、P2/D2 A3=0.4、P3/D2 A4=1.0 で、Unique Persons=3、Memberships=4、Allocated FTE=3.0。knownAt=2027-07-03 では A2 が外れ、Memberships=3、FTE=2.4。これは既存の**合成Expected**であり、現行製品の実Org Rowsetがそれを満たすと主張しない。

- Headcount（distinct Person）、Membership count（Relation occurrence）、FTE（配賦の和）、Capacity/Workload/Budget/Skillは**粒度・単位・時点・Source・重複除去キー**を別々に定める。複数経路のGraphやJOIN fanoutを人数・FTEの増加にしない。FTEと業務責任の移管比率/Grantを同義にしない。
- WHERE/ORDER/EXISTS/NOT EXISTS/集計/Top-N/LEFT JOIN/SELECT */alias.*/NATURAL JOINの依存は表示Field以外にも及ぶ。秘密Fieldを出力しないだけで取得・評価・開示上の影響が無いとは言わない。
- Query/Rule/ConsumerごとにRequired集合と実Read Traceを照合し、**affected / scoped-unaffected / unknown** を返す。partial、未読Plugin、失われた旧依存辺はunknownの原因となり得るが、無関係で完全な別Claimまで一律失効させない（#2582/#4518）。
- 検索0行はcomplete cohortの根拠なしに「不存在」と言えない。検索1行は取得範囲と一意性の根拠なしに「唯一」と言えない。0/1/複数/partialをBusiness Ruleの判断と現在Grantに短絡しない。

## 4. C1責任・動的候補・引継ぎを同じSourceから区別

| 事実/操作 | 確認すべきもの | 禁止する推論 |
| --- | --- | --- |
| P2がD1/D2に兼務、またはD1→D2へ異動 | Membership occurrence、valid/known、Allocation、Read scope | C1のhandlingUnitやSubsystem Owner、実Grantが移った |
| Case C1のhandlingUnitをD1→D2変更 | Case/Responsibility revision、業務Rule、必要Handoff集合 | 人事異動=Case移管、通知成功=ACK |
| 承認候補0/1/複数/partial | 候補集合と完全性、Rule revision、Case revision、Selection context | 候補=唯一の決裁者、候補=現在の実Action許可 |
| 代理read/代理action/緊急昇格 | 委任のscope・期限・取消し・再委任・本人利益相反、現在Host Grant | Person/Role/兼務=別TenantのPrincipal権限 |
| D1→D2→D1 (ABA)、Grant失効/Commit不明 | Source/Case/Policy/Grant epochとAttempt/Receipt/必要ACK | 元の値に戻ったから旧認可・旧Receiptが再有効化 |

候補解決の表示上の4状態：**complete-none / complete-one / complete-many / incomplete-or-unknown**。これはUI/検査用の分類案であり新しい標準Result型や認可Engineの採用ではない。複数候補は共同責任/合議で正常となり得るため、用途別Owner RuleなしにError固定しない。承認候補の提示はCurrent Host Grant・Effect Fenceではない。

## 5. Plugin/Headless/外部Harnessの六つのScopeを誤継承しない

| Scope（設計上の比較軸） | 組織C1との交点 | このScopeから導いてはいけないもの |
| --- | --- | --- |
| Distribution | 組織Profile/Plugin Bundleを利用者へ配布 | 利用先での実Load/実Grant |
| Authoring | DSL・Studio・SHOの編集/Source Pin | DB実保存/Host実Action |
| Composition | Function/Query/Providerの選択と合成 | 選択候補が実Invokerと同一という推定 |
| Organization | Person/Employment/Membership/Case Responsibility | Account/Principal/Tenant/実Grant |
| Runtime | 外部Host/.NET/Java/WASM等の実対象・現在Grant・Effect | 設計モデル/旧セッションからの認可継承 |
| Disclosure | 外部AI/SHO/CLI/Headless B2Cへの用途・受信者別送達 | 送信=受信=保持=意味保存、非表示=非取得 |

- **完全Headless B2C**は標準Renderer/TEI Web Runtimeを必須とせず、同じBusiness Meaningから用途別外部Host/Consumerへ生成できることを比較する。Person、購買者Account、認証Principal、Tenant、REA Resource–Event–AgentのAgent、AI Agent、外部解析ツールREAのNodeは**別Identity**。r7のP2が二TenantのAccountを持つという新事実は既存fixtureに存在しないため、必要なら**別Scenario**として既存Driverへ追加する。
- 外部AIハーネスの活動SessionとAction時の現在Authorityは別。Source/Subject/Occurrence/span/revision/selection epochを固定し、Navigate→Preview→Review→Owner Accept→Reproject。Focus移動では勝手に別対象へ変更しないが、Source/Rule/Policy/Recipient変更・A→B→Aでは旧Patch適用を拒否または再資格化する。
- 通常の組織Dataに専用秘密区分を自動付加しない。ただし実際に保護対象Fieldが混在する場合、Provider Row/Field enforcement・Count/Export/Tooltip/診断/ログ/外部Tool出力を**同じ公開Sliceの二世界**で検査する。非公開側の件数差が公開結果から推定できる経路も対象。
- SUI/CSWのKJ的近接・Island、外部REA Source Observation、Resource–Event–Agent分析レンズはいずれも組織のAccepted RelationやHost Principalを自動生成しない。OwnerのMeaning Acceptanceへは出典・粒度・方向・並行Edge・反例・Coverageを保持して候補として渡す。

## 6. DSL・三要素分析法・Studio Desktopセルフビルドの接合

- **Source正本は既存TEI DSL/標準Data構造**。構文糖衣・Markdown/Graph/SQL表記の違いから第二Meaningを作らない。Goのstruct/func等をドメイン向けDSLに露出させない。DSL自己記述/新構文は初回C1実ReadとDesktopセルフビルドのHard Dependencyではない（#4359、Draft #4599）。
- 三要素分析法はData/Function/Businessの原典との適合を業務Profileとして検証する。#4551がmainへ統合した**明示CRUD-onlyとUnusedDataの修正**は限定Product事実。Studio/SHO/AIの完全参照表示と要レビュー診断・Effectは別に実証する。
- UIから既存Field、新しい永続Field、Derived、Transientを明示選択できる。新Fieldの意思を確定した時だけDesired Patch→Impact Preview→既存Schema Migration/Grant→独立Readへ進む。描画・Treeの隠れEdgeや集計/変換結果の非単射Projectionは**read-only**とし、Source span（値Spanと属性名Spanを区別）を特定しない書戻しを拒否する。
- Desktopの段階は**D0** 生成Workbench込みの実Windows Wails/WebView2起動と同一Process/HEADのConsumer受領、**D1** C1モデル/CRUD read-only受領、**D2** Stage1 TEI実ProcessがStage2 Desktopを再生成し起動、**D3** X-TEA三要素Profile/Plugin、**D4** 同一Source/Selectionを用いたSHO/AI再投影。#4579/#4587の限定WebView2 CIと実Stage2自己再構築は別資格。実受注PG Read(#4539)は組織C1二時点Providerの合格に合算しない。
- Plugin Role（Authoring/Generator/Runtime Provider/Presentation）は**利用役割**であり固定Core enumではない。Stage1のGeneratorをStage2実行Bundleへ無条件同梱せず、実Builder選択・Dependency closure・生成Artifact member・Stage2起動を別Witnessにする（Draft #4601）。
- 変更影響/高速化はWorld Fingerprint、Model/Artifact Revision、Source raw/Generator/Plugin/Provider/Rule/Policy/Recipient/Read Cohortを**資格ごと**に扱う。byte一致や局所CI PASSを意味受理・実行許可・Consumer受信へ昇格させない。A→B→Aの再利用、部分失敗、空集合、未解析Pluginを負例とする（#2600、#2514、#1180）。

## 7. 既存Scenario/Expectedへの受入割当（新しいExpected IDは発行しない）

| 既存受入 | 必須の実観測・反例 | 資格・担当 |
| --- | --- | --- |
| #4490 ORG-BEH-01、#2510 ORG-TM/ORG-KH | 同一Occurrenceの二版共存→同一validAt/異knownAtの実Rowsetとr7 Expectedを照合。旧版上書き・期間重複を失敗扱い | 時点意味 #2386、版Key/保存 #1126、実Read #2622 |
| #4490 ORG-BEH-02、#2510 ORG-QD/ORG-RF | WHERE/ORDER/EXISTS/NOT EXISTS等の非出力依存・complete/partial/unknown・r7 FTE 3.0→2.4 | Query #2387/#2586、Claim #2582 |
| #4490 ORG-BEH-03〜05 | 辞令部分失敗・動的候補0/1/複数/partial・委任失効/別Tenant/自己承認 | Rule/Procedure #1129/#1125、Host #1174 |
| #4490 ORG-BEH-06/07、#4501 CX-01〜06/13〜17 | Membership≠Case責任≠Subsystem Owner、Handoff ACK、Graph二経路と粒度、Studio/SHOの別cohort | Modeler #1124、Handoff #1444、Consumer #2516 |
| #4501 CX-07〜12/18、#4515 PIN-S0 | Source/Selection ABA、AI後着、SHO著者編集競合、保護Field二世界・用途別Export | Pin #4515、SHO側Owner、Disclosure #2516 |
| #2586 HJ-09、#2831 D2-SB、#2514 | 通常CLIの実Provider→Context→Consumer、Stage1→Stage2実生成/起動、各exact HEADと独立Expected | 別々の製品資格。Org C1全体合格とはしない |

**既存の受入票に書く列**：Source occurrence/span/revision、World/Meaning/Rule rev、validAt/knownAt/observedAt、DB/Provider/selected vs invoked/actual SQL+params、Read Cohort/Scope/Completeness、Required vs Consumed/Received、Consumer/Purpose/Recipient、Selection epoch、Case/Responsibility rev、Current Principal/Grant epoch、Expected source independent、Actual mechanism/Result、Effect Receipt/ACK、Git HEAD/CI、Owner acceptance。新たなProduct Evidence Storeを作る指定ではない。

## 8. 最短の統合順と停止条件

1. **T0：競合解消・実Source再照合（read-only）**。#4566と#4567の重複Query/Mapping差分と実CI結果を比較し、現mainのPK・DDL・Sourceを確認。#4555（Desktop §10.12）と#4585（六Scope §10.12）の並行変更を**相互無損失に節番号/リンク調整**して前方統合する。片方の旧main全文上書きは禁止。
2. **T1：History保存方式の一件検証**。DB Owner判断の候補一方式で同一Membership Occurrenceの旧・新版を実隔離PGに**共存保存**。同一validAt/異knownAt、0/1/重複・null/partialを独立Expectedと照合。新Core/自動移行はこの時点では不要。
3. **T2：実Queryから判断候補へ**。#2586の通常CLI最小実Readを確保しつつ、Org C1用Function→Query→Provider→Read Cohort→Context→一Consumerを同一Attemptで接続。候補/Rule/Case revを示し、現在Grantは未検証なら「未検証」のまま。
4. **T3：三要素・Studio・SHO/Headless**。D0/D2自己再構築は並列独立で実証。C1のaccepted occurrenceをloss-aware投影し、Source/Selection、異なるConsumer受領と用途別Disclosureを分離。未接続のProvider結果を組織C1として合算しない。
5. **T4：実Host/引継ぎ**。現在Grant・Effect Fence・Commit/Receipt・ACK・不明時の再照会を別資格で確認。業務RuleはOwner採否、別Host/Tenant効果はそのHostの権限で再検査する。

**Hard Stop**：不完全Readから不存在/唯一を確定、履歴の物理版を黙示上書き、兼務からTenant Grantを導出、表示Treeの隠れEdgeを削除、AIの古いPin/Sourceへ自動Apply、SHO保存を実TEI更新とみなす、DB Query/CIの局所成功を業務完了/Host認可/Stage2セルフビルドへ昇格、Commit不明の無条件再試行は不可。

## 9. 未決とOwnerへの差戻し

| 未決 | 決定先 | 確認すべき反例 |
| --- | --- | --- |
| 同一Occurrenceの版ID、PK、訂正/取消し・履歴保存・重複期間 | #1126 / #2386 / DB Owner | #4567実Read合格でも履歴共存不能 |
| 過去の有効な業務状態と、その時点の把握状態の使い分け | #2386 / 業務Owner | as-of history replayを現在権限に流用 |
| 単独/共同責任、承認時再評価/申請時固定 | #1129/#1125 / 業務Owner | 完全候補複数・部分候補0から誤確定 |
| Handoff ACK前の旧/新担当とCommit不明 | #1444/#1174 / Host Owner | 通知成功=責任完了 |
| 配布/Composition/Headless Consumerと開示範囲 | #2903/#3509/#2516 | Pluginの有効化=Tenant Grant |
| Studio Stage2/三要素Profileと組織Query接続 | #2831/#1124/#2586 | 異なるQuery/AttemptのPASSを加算 |

**現状資格（2026-11-14照合）**：#4539実受注PG Readと#4551三要素修正、#4532 r7 Node合成はmain統合済みの限定成果。#4567の固定HEAD `f2c7c8d6...` は実使い捨てPG 4 subtests成功をPRが記録するがDraft未統合。#4555/#4585/#4599/#4601/#4579/#4587はDraftで、Stage2 Desktopセルフビルド、同一Membership二版保存・Org C1 Function→Provider→Consumer、現在Host認可、正式Owner意味受理は**NOT RUN/NOT ACCEPTED**。各実行時はHEADと実観測ログを再確認する。
