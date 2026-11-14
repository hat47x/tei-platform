# 組織C1：二時点保存・判断候補・Consumer接合の横断受入補遺（2026-11-14）

> 状態：**設計・採用前**。この文書の作成は製品実装、独立Expected実行、Owner意味受理、Host認可を意味しない。主計画は [ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md](ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md)、意味Ownerは #2386。既存の #4490 ORG-BEH、#4501 CX、#2510 org-synthetic-r7 を再利用し、第二のScenario/Expected正本・組織専用Runtimeを作らない。

> **現在の到達点**：組織Sourceの一時コピーから人物との関係候補をDSLで読み込み、整合性を検査し、追加列のDDLを生成するところまではGoテストで確認しました。ただし、人物との正式な対応、同一所属の複数記録版を実DBに共存させる保存、人物数・FTEの実集計、独立Consumerの受領は未検証です。直近の実装順は§21、個別の判断は§2〜20を参照してください。

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

## 10. 実Product境界の具体的な断線と最短受入（2026-11-14横断再照合）

新しいExpected/Engineを定義せず、既存#4490 ORG-BEH-01/02/04、#2510 org-synthetic-r7、#4501 CX、#2586実Read、#1180/#2514の独立Expectedを使い、**実Sourceで確認した未接続の辺**から実証順を固定する。

### 10.1 r7と実PostgreSQL試験は異なる入力である

mainのNative as_of_membership Queryは membership_id / org_unit_id / allocation の3列を出力する。mainに残る known_from >= :known_from、物理テーブル名 organization.membership、valid/known終了時点の欠落はDraft #4567で修正候補となり、同PRは使い捨てPGの4 subtestsを限定実行。ただし #4567 のテストデータはA1〜A9を異なるmembership_idで保存し、A2は一行だけ、A4のknownFromは2027-07-02である。

対してmainの org-synthetic-r7 は、**同じA2の2記録版**（旧版knownTo=2027-07-02、新版knownFrom=2027-07-02・validTo=2027-06-01）と、2027-04-01から既知のA4を持つ。したがって **#4567のPG限定PASSはr7の二版共存、履歴再現、FTE変化、人物数を実証していない**。

| r7照会（validAt=2027-06-15） | Membership集合 | distinct Person | Membership数 | FTE |
| --- | --- | ---: | ---: | ---: |
| knownAt=2027-07-01 | A1,A2,A3,A4 | 3 | 4 | 3.0 |
| knownAt=2027-07-03 | A1,A3,A4 | 3 | 3 | 2.4 |

値は既存Node合成Referenceの**手書きExpected**であり、Product実Rowsetの値ではない。A2に依存するD1のFTEは1.6→1.0、D2は1.4のまま。FTE、Membership count、distinct Personを同一Measureとして扱わない。

**SemanticとRead Projectionの二重GAP（main実Sourceで再確認）**：現Query出力の3列からは person_id と記録版Identityが観測できない。さらに現在の organization.Membership の型付きSemantic宣言そのものにも personId 属性・PersonへのRelationがない。従って (a)「既存型付きMembership→Person Relationを追加取得する」経路は**現mainでは未成立**であり、Relation/Keyを業務Ownerが受け入れてから物理Mapping・同じRead Cohortの実取得を確認する必要がある。(b) Queryに必要な人物識別・記録版の実Witnessを明示する経路も、Source/DB側に対応する根拠を追加する必要がある。どちらも曖昧な名前/表示から人物IDを推測せず、全Queryに新必須列や第二Context Storeを課さない。

**最小受入の順序**：
1. 既存#4566/#4567のSQL/Mapping重複と現行Persistence主キーのSourceを突合。履歴版別ID/物理PKの採否は#1126/DB Ownerへ差し戻す。
2. DB Ownerが選ぶ隔離実装で、r7のA2旧版・新版を**同一DBに共存保存**し、validAt固定、knownAt切替、期間境界・同時に矛盾する二版・取消し・NULLの結果を独立Expectedと比較。別Runで単一A2を入替える試験は版共存の証拠にならない。
3. 同じ実Read Cohort/ScopeからMembership/Person/FTEを**別Claim**として検査する。person_idが実取得されないなら人数はunknownのままとし、r7の3人をQuery実証済みにしない。
4. その実Rowsetを既存Function Data Contextと一Consumerへ同Attemptで受領させ、Case責任/候補Rule/現Grant/Effectは段階ごとの別資格にする。

### 10.2 判断候補の使用資格はClaim・時点・実効権限に相対的

- 候補0/1/複数/partialは**検索結果と取得集合の完全性**の組合せ。完全な1人も、現在Host Grant・委任条件・Case revisionが成立しなければ実行許可ではない。
- A2に対するknownAt変更はA2依存FTE/承認候補へ影響する。一方、独立して完全観測されたD2だけのClaimがA2に依存しなければ scoped-unaffected の可能性を残す。旧辺を含むRequired closureと実Readを確認しない限りunknown。#2582/#4518の既存契約を利用し、無関係なUnknownによる全体失効も、未観測Sourceによる虚偽の無影響も防ぐ。
- #4616のmain統合済みProduct反例では、Source-root実在性が変わるだけでWorld/Model Fingerprintが同一でもValidation結果は変わった。組織C1のSource Pin/Reader/Build/Evidenceは、Model Revision一致から現在性を推定しない。
- 代理・承認・責任移管中のEffectが不明となったとき、Hookの内側のeffectUnknown/lookup-firstを通常warnに落とさない。Draft #4602の局所修正を既存#1174/#1129のReceipt/Effect観測へ接続し、照会前の再試行/後続Effectを止める。PRの局所成功を実Host実証へ昇格させない。
- Case handlingUnitとPerson異動、Subsystem/CUD Owner、通知/ACK、B2C Account/Tenant/Principalは非自動対応。Rule/Case/Policy/Recipient版の変化は、必要Read/現在Grant/開示を用途別に再検査する。

### 10.3 Studio・Plugin・SHOの実使用と選択の区別

- Draft #4579の固定HEADでは生成WorkbenchとGo Serviceの同一Windows Wails/WebView2実行・配信JS digest照合・read-only Catalog/Inspectに限定成功。ただしこれはB0相当で、Stage1 TEI ProcessによるStage2再構築(B2/L3)、r7/Case C1実Consumer、全Provider/全Pluginは未実証。Draft #4587の別HEADの検証を加算しない。
- Draft #4623の現HEAD f1e219a… はCaller指定digestの事前照合に加え、限定Plugin BuildPlanでResolveされたBuilder ArtifactDigestと起動前実子exeの照合・不一致拒否を追加した（PR報告では当該HEADのGo/Health CI成功）。ただしCatalog自体はCaller入力であり、非Plugin Desktop/全入力閉包・OS Loader TOCTOU・Stage2 Desktop実起動を証明しない。#4621/#4601のContract/選択/実Build inclusion/actual Invocation/Effect/Consumer Disclosureの分離を組織C1の生成・読取・再投影へ適用する。
- #4597のPin stateはWorkspace epoch/Generation/Undo/ABAに対する局所状態。Graph/Inspector実UI・Source revision・Recipient disclosureとの実接続は別。C1の編集はSubject・Workspace・Relation occurrence・値のSource span・Source revision・Selection epochを固定し、最新Source再取得→差分Preview→通常のレビュー/必要なOwner受理→Source Patch/再投影へ進む。非単射Graph→Treeの隠れた第二親や集計結果を自動逆編集しない。
- DSL/Typed Script Entry（Draft #4599/#4618）からは部分Lowering/未読Rootを完全なWorldと見なさず、明示Function→実Query/Providerの結線と実Command/Effectを別に示す。Source糖衣表現・三要素/REA/SHOの資料/AI候補を第二Meaning正本へ昇格させない。
- 通常組織Dataを専用秘密扱いしない。保護Fieldが混在する場合のProvider/Count/Export/Tool出力、外部REAのEvidence path/低位Error、SHO/AIへの用途別開示は#2516と外部Observationの担当へ戻し、同一公開Sliceの二世界で漏洩有無を検証する。

### 10.4 優先順位と既存Ownerへの実装引渡し

| 優先 | 最小の次の実行Exit | 再利用するOwner・Expected |
| --- | --- | --- |
| P0並列A | #2586 通常tei CLIのFunction→実Provider→Context→一Consumerを同Attemptで実証（最初は受注Query） | #2586/#2506/#2514 |
| P0並列B | Stage1実Process→実Builder→Stage2 Wails Desktop/WebView2実起動。B0とB2を区別 | #2831/#4583/#4623 |
| P1-Temporal | r7同一A2の二版共存、同一validAt/異knownAtの実Read、Person/FTEのRequiredと実Rowset照合 | #1126/#4490/#2510/#2586 |
| P1-Decision | C1のCase rev/Rule rev/候補0/1/複数/unknownと独立Consumerを同Sourceから判別 | #4490/#4501/#1125/#2516 |
| P2-Effect | 現行Host Grant・代理・Hook Effect・Receipt/ACK・Pin編集・B2C/SHO開示の別資格を検査 | #1174/#1444/#4515/#3509 |

**保留・競合**：#4566/#4567はNative SQL/Mappingを重複修正、#4555/#4585は同じ組織計画§10.12、#4601/#4620などはMASTER_PLANを並行編集。旧branch文書の全文置換や異なるHEADの局所PASS合算は禁止。現行mainとのSource/責務/期待値差分を先に照合し、無損失前方移植で統合する。今回の追補では、新しい実PostgreSQL二版試験・Windows Stage2・Host Effect・Owner業務受理を**実施していない**。


## 11. 履歴保存の導入と既存DB移行を別の受入として接合する（2026-11-14）

### 11.1 組織C1の「最初の実Read」と「既存データの安全移行」を分離

現行mainのPersistenceは public.org_memberships、主キーは organization.membership.membership_id 単独。mainのQueryとDraft #4567の修正版との違い、およびr7旧/新版A2が同時保存できない点は§10参照。本節の新しい論点は、**同じ業務Occurrenceを複数物理版で保存する設計へ、既存利用者と既存DBをどう移すか**である。

- **新規隔離Referenceの受入**：DB Ownerが承認した候補（独立record_version_id/row_id、Occurrence＋版の複合キー、Current＋History等）でA2旧版と新版の共存・二時点照会が可能かを証明。ここでは既存利用者の移行完了と主張しない。
- **既存DBの変更受入**：同じ業務IDを読んでいた旧Reader/Writer、Generated App、CLI、Provider、SQL Dialect、Schema Mapping、データ保持・監査を変更前後で確認する。主キー差替え/列追加のMigration成功や単一新ReaderのPASSだけでは十分でない。
- **通常CLI/Studioの最初の実ReadとDesktopセルフビルドは独立P0**。この段階的移行全体を実Read P0やStage2 P0の完了条件へ追加しない。最初のr7実DB二時点試験は隔離DBでよく、既存本番DBへの移行と混同しない。
- リリース計画は既存 internal/database/migration と releaseplan.Phase を用いる。#4125のG0〜G5は**未採用の検収観点**であり、TEI Coreの新Gate、新Migration Engine、第二Release権威を採用するものではない。実行時の自動DDL適用も本書では採用しない。

### 11.2 履歴の「分かっている範囲」を捏造しない

r7は合成SourceにA2の旧/新版が両方あるため、二時点Expectedを明示できる。しかし既存実DBにMembership A2の現行行が1件しかなければ、旧版が過去に存在したか、何度改訂されたか、そのknownFrom/knownToがいつだったかを**現在行から逆算できない**。

| Input/移行状況 | 言えること | 言えないこと・必要な保留 |
| --- | --- | --- |
| 出典付き旧/新版と確かなvalid/known境界が残る | 当該Sourceが証明する期間範囲の履歴として移す候補 | それ以外の期間の完全履歴、別Providerの事実 |
| 現行行のみ・古い監査記録なし | 観測された現行Snapshotと採取時刻・出典を固定できる | 移行前のknownAt履歴を完全再構成、0行を当時の不存在と断定 |
| データ移行のため新record_version_idを採番 | 物理行Identityを明示的に作成可能 | 採番IDが業務Occurrenceや過去のOwner意味を保証するという推論 |
| 旧Readerが一つの現行行を見るため互換Projectionを提供 | 指定された現行版のRead形状を維持できる可能性 | 全過去版・現在権限・旧Writerの更新動作が同等という推論 |
| 後日Source/CDC/監査の遅延到着で過去記録が見つかる | Provenance付き追加Candidateと再検証の根拠 | 過去の確定Receiptや既承認判断の無断改変 |

歴史的不明期間は「既知の0件」ではなく **not-established/unknown** として保持する。既存契約で状態を表せない場合はそのGAPをOwnerへ返し、架空のknownFromや現在Dateを原歴史の開始日として設定しない。現行Snapshotを新しい観測時点で登録する場合も、**観測開始時点からのKnown範囲**であることを明示し、過去事実への遡及認定は別Owner判断とする。

### 11.3 既存移行Phaseに渡すC1の具体的受入（新しいGateは発行しない）

| 既存#4125の検収観点 | 組織C1で必要な独立観測 | 停止条件・担当 |
| --- | --- | --- |
| G0 Baseline | 旧PK/業務ID、A2含む実Rowsetと期間Coverage、全Reader/Writer・直接SQL/Batch、現Schema/Provider、Source/Rule、手書きr7 Expected | 未観測Writer/不明な旧履歴を既知完全と扱わない。#1126/#2386/#4125 |
| G1 Expand | 旧Readerを壊さず新version Identity/列/物理表・必要Index/互換Projectionを導入。別接続でSchemaと旧Read/Writeを再観測 | 旧Clientが複数版を重複表示、旧Writerが版を黙示上書きするならCutoverへ進まない。#2622/DB Owner |
| G2 Backfill | 出典のある版のみ移行。移行済みCurrentと保持可能なHistoryを別照合、Key対応・nullable end・時間精度/zone、Checkpoint/途中失敗・再開 | 不明の過去版を生成、同名行/UUID変換でBusiness IDを再定義、Commit Unknownの無条件再試行。#1126/#1174 |
| G3 Validate | r7二版を同じ隔離DBへ保存し、validAt固定・異knownAtのRowset/Person/Membership/FTEを検証。旧新Reader/Writer互換、重複期間・NULL・部分Scope、独立Provider Read | 単に行数一致/DDL success、別DB・別Snapshotの集計成功、旧Reader/Writer未試験は保留。#4490/#2586/#2510 |
| G4 Cutover | 書込Owner・Writer Fence、旧新Alias、現在Grant、未完Attempt/CDC・Handoff、Commit→Receipt照会→別接続Read、実Consumer受信 | 二重正本、遅延旧Writer、ACKのみ、Policy失効、作用不明は切替完了にしない。#4125/#1174/#4501 |
| G5 Contract | 旧構造/互換Viewの参照者全件と観測窓、履歴Retention/監査要件、旧Writer退役、最終Schema/Rowsetを独立確認 | 履歴消失、旧Batch未観測、保持Owner未受理、旧Source/書戻し経路残存は削除を保留。#4125/#4234 |

上表は検証マトリクスであり、実行時に常に6つのGateを強制する製品仕様ではない。PostgreSQLのDDL/制約、SQLiteの表再構成/ロック、非トランザクション操作、旧新マッピング・生成Targetごとの制限はDB Ownerが個別に評価し、全Providerで同じ原子性・排他制約が保証されると仮定しない。

### 11.4 訂正の原子性と旧新Reader/Writerの反証

- **単一Occurrenceの訂正**：旧記録版のknownTo閉鎖と新記録版の追加が1つの公開単位になることを検証。複数行更新の途中失敗やリトライで、あるknownAtに旧新が同時適格・両方欠落になる状態を、正式確定済みとして公開しない。複数Providerに跨る原子性は別資格。
- **同時に有効なMembershipが複数ある正例**と、**同じOccurrenceに同一validAt/knownAtで矛盾する二版が選ばれる負例**を分離。異なるOccurrence同士を重複として削除しない。版間重複をLast Write Winsで黙示確定しない。
- **過去版を含む新表への旧SQL直結**は重複行を返し得る。Current-only Reader/互換Viewを置くかはSource/QueryとConsumerのRequiredに応じて選ぶ。旧Reader向けと履歴as-of Reader向けは別の実Query契約とし、互換Viewを置いた事実を旧Writerの安全性の証明にしない。
- **書込みの切替**では旧Writerを退役・Fenceしたか、旧Writerを既存機構で安全に新履歴へ反映できるかを観測する。存在しないCDC・Hook・DB Triggerを既実装のものとして書かない。旧Writerが当初の1行をUPDATEして訂正履歴を消せる構成なら、G4の互換性は未達。
- **結果の一致**は粒度ごとに比較。旧現行行のcount、新履歴全行のcount、as-of active Membership count、distinct Person、Allocated FTEは互換性検収で同一数値を要求しない。#2586でQuery/Read ScopeとEntity Grainの根拠をそれぞれ必要とする。
- **日付とタイムゾーン**：r7は日付文字列、Persistenceはdatetime-tzである。日境界の変換規則とQuery bindのzoneをOwner/Provider側で固定し、日付一致でも異なるOffset・精度のas-ofが同義になると推定しない。

### 11.5 Generated App・StudioとMigrationを混同しない

#4234は生成AppがSQLite等のmigrationを読み込めないProduct Gapを指摘する。しかし「アプリが起動できる」「Migration SQLが置かれている」「自動で適用された」「互換性と意味が受理された」は異なる。最初に **起動時のActual Schema確認→必要なMigrationの診断→明示した既存Upgrade経路→独立接続で再確認** を比較し、無条件の自動破壊的DDLを先行採用しない。生成Appの同梱/適用方式は#4234/#4125/#2831のOwner判断とし、TEI Studioのセルフビルドとは別の製品資格として追跡する。

ER/CRUD/三要素/SHO/AIでは、Occurrenceの業務同一性、物理Row版、投影上の表示行、選択中のSource値Spanを異なるIdentityとして保持。移行でDB PKが変わってもPerson/Membershipの業務IDを勝手に付替えず、読取Scope不完全時の「参照0件」を削除候補に昇格しない。Graph/Treeのlossy描画・集計上の1セルから物理履歴版を推測して書き戻さない。

**最小実行順・Owner引渡し**：
1. #4490/#1126：r7旧/新版A2・同一DB二版、期間の重複/NULL、独立Rowsetと現在のQuery/Physical PK GAPを最小に確認。
2. #4125/#4234：新規隔離DBだけで成功した後、旧Reader/Writerを含むExpand→Backfill→Validateの小さな互換性試験を既存Release/Migrationで計画。履歴の原出典を確認できない期間はunknown。
3. #2586/#2506：同じas-of/Provider Snapshotに属するPerson/Membership/FTE各Claim→Function Context→一Consumer。別Snapshot/別Queryの数値一致を一Attemptに合算しない。
4. #4501/#1174：Case責任、Rule候補、Current Grant、Hook Effect/Receipt/HandoffとSourceの因果を別々に確認。段階間の正式Cutover/Contractは業務/運用Owner採否。
5. 独立P0として#2586通常CLI実Readと#2831 Studio Desktop Stage2は継続し、全DB移行をそのHard Gateへ追加しない。

**本節の実行資格**：設計・未受理。main/既存Issue/既存r7合成fixtureを照合した受入案のみ。SQLite/PGの新Migration、旧新Client共存、履歴不明期間の実データ調査、現在Host Grant/Receipt、Generated App自動適用、Owner正式Meaning、製品Consumer E2Eは **NOT RUN / NOT ACCEPTED**。


## 12. 組織改編の公開境界・決裁再評価・協業編集の受入（2026-11-14）

本節は #4490 ORG-BEH-03/04/06、#4501 CX-01〜09、#1129 Table lifecycle、#1174 Effect Fence、#1444 Handoff、#4125 DB cutover、#4515 Source Pin の**既存受入条件を組み立てる**。新しいOrg Workflow Engine / Global Transaction / Team Lock Service / 第二Expected正本は要求しない。既存r7は所属二時点のReferenceであり、**Case C1・辞令・旧新責任・ACK・Host Grantの実事実を含むと偽らない**。それらは別の既存独立Scenarioを使用し、同一Source/Attemptへの結線をProduct Ownerが別検証する。

### 12.1 改編を一つの状態値へ押し込まない

| 観測上の区切り | 必要な根拠 | 次段へ進んでも自動では得られない資格 |
| --- | --- | --- |
| Proposed（編集候補） | 対象Occurrence・Source Revision・期待する新しいRelation/期間・作成者 | 正式な業務意味、DDL、実効Grant |
| Validated（事前検証） | 部分取得の有無、キー/期間/循環、多重度、旧新Rule、Required集合 | 他Providerの現在性、確定更新 |
| Accepted（業務意味受理） | 業務Ownerが受理した対象・改編意味・有効時点 | DB/Hostへの保存、全Readerへの公開 |
| Committed（保存効果確認） | 対象Providerの実Transaction/RowVersion/Operation ID/Receipt、失敗時の作用不明区別 | 別DB/外部Hostまで原子的に変更されたこと |
| Published（読取に利用する版） | 同じ公開範囲に属する新旧Relationを、実Readが一貫して選べる根拠（snapshotまたは対象Source/Providerの公開版） | 旧Case責任の失効、現在Grant、Consumer受信 |
| Consumed / Handoff acknowledged（受信・引継ぎ） | 用途別Consumerでの受領、業務Task/Case移管に必要な相手方ACKと別のEffect/Receipt | 所属版や承認候補から自動的に業務完了と判断すること |

この区切りは**検証時の観測分類**であり、新しい固定DSL State enumや共通Runtimeを設けない。Publishedというラベルは、必要なReadが一つの一貫した公開状態を示せる場合だけ使用する。単一DB内のTransaction、複数DB間のSaga的手順、外部連携のACKは保証範囲が異なり、異なるProviderの別時刻Readを組み合わせて「全社改編の一斉公開」と呼ばない。

### 12.2 BEH-03の最小交錯・停止条件

対象は架空の改編Directive Xが Membership U/V の期間を更新し、必要ならCase C1のResponsibilityを別操作で変更する**追加の合成シナリオ**とする。r7のA2をDirective Xの事実として書き換えない。既存ORG-BEH-03、CX-01〜03のExpectedへ紐づけ、実装側の結果からExpectedを生成しない。

| 介入/交錯 | 保存・Readの必要な観測 | 正式な成功と誤認してはいけないもの |
| --- | --- | --- |
| 更新U成功のあとV失敗（同一DB、同一公開単位を要求） | Transaction/Writerのrollback または公開前の隔離と、独立Readが旧一貫状態であること | Uだけ新/Vだけ旧を正式Published |
| U/Vを異なるDBへ配置、片方はCommit済み | 各ProviderのReceipt/Scopeと未完状態。原子的公開未保証なら部分完了/保留として可視化 | 共通World ID・Directive IDが同じだけで分散原子性 |
| 旧Readが進行中に新Source/Ruleが公開 | Read Cohort/版/Predicate/valid/knownとRule revの整合を比較し、強い一貫性要件を満たさなければunknown/再読込 | 古い候補と新しいCase Revisionを混合して現在候補一人 |
| Commitは成功したがレスポンス/Receipt取得が失われる | Operation IDでlookup-first、別接続ReadとHost Effect/Receiptの対応を再確認 | timeout=rollback、単純再試行=安全 |
| Cutover中に旧Writer・Queue再送・遅延Consumerが到着 | 旧Writer Fenceと対象版照合、未完Operation/ACK、更新後Provider Snapshotを確認 | 新版公開済だから旧WriterのUPDATEが無害 |
| D1→D2→D1、値は元通り | Source/Case/Rule/Grant/Selection epochとOperation IDを照合 | 表示値一致・同じハッシュで旧Grantや旧Patchを復活 |

一括辞令の**原子性を要求する範囲**は業務Ownerと対象Provider/DB Ownerが決める。複数のMembershipを一つのDBトランザクションで更新可能な場合と、旧人事System・複数DB・外部通知を跨ぐ場合は別能力。後者は業務上の保留・補償・後続確認を設計し、二相Commitや新Global Txを既定としない。

### 12.3 承認候補・旧新責任・現行Authorityの受渡し

| 旧Case責任に現在の資格 | 新Case責任に現在の資格 | Case C1への決定候補の解釈 |
| --- | --- | --- |
| 有効 | 無効 | 旧責任が残る可能性はあるが、引継ぎ成立・旧Grant有効とは推定しない |
| 無効 | 有効 | 新責任への移管候補。旧通知/ACKが不要かどうかはTask/Business Ruleが決める |
| 有効 | 有効 | 共同/移行期間中の重複が妥当か、利益相反・排他Rule違反かをOwnerが評価。任意の一人を採用しない |
| 無効 | 無効 | 未割当て/保留/管理者再解決の候補。partial取得からの「双方無効」は確定できない |

この4象限は**必要なRead Cohortが完全で、旧新関係の意味・Rule・Case版が解決された場合**の業務判断候補である。実ActionではCase Revision、対象Step、Policy/Delegationと現在Principal/Grant epoch、Effect直前のVersion Fenceを必ず別確認する。Ruleが申請時に固定されるか承認実行時に再解決するか、複数人合議・代理・利益相反・空席の扱いは #2386/#1129 の業務Owner判断。組織Personを認証Principalへ黙示変換しない。

**一往復の最小証拠連鎖**：受理済みSource/Meaning → 対象Scopeの実Rowset（validAt/knownAt/Read Cohort）→ 当該Case/Rule Revで導出した候補と不足条件 → Hostの現行Grant照合とAction/Operation ID → Effect Receipt/不明状態 → Handoff ACK → 用途別Consumer実受領。これらは**単一Attempt内で観測できるものと、相関IDで後から取得するものを区別**し、同じWorld hashだけでそれぞれの資格を継承しない。Outbox/Receiptの原子的保証が無い外部Hostでは未確認/部分効果を保持する。

### 12.4 Studio Desktop／Web共同編集でのモデル改編

- **Desktop**：ローカルのGit作業版でSource span/Occurrence/RevisionをPinし、Previewと適切なReviewを経て保存・Git共有。UI上のカーソル選択と保存済Accepted Meaning、DB実Effectは別。Sourceの並行改版に対しては再読込・差分/衝突提示し、古いPinから自動Applyしない。
- **将来のStudio Web**：チーム開発サーバを共有して同時編集する構想でも、単なる画面ロックやAIハーネスSessionの一致を唯一の排他根拠にしない。Dirty Buffer/保存済Source/受理済Meaning/実Provider Snapshotを区別し、**編集操作が対象とするSource Revisionを条件**にしたConflict検出、ReviewとGit Change-setへの受渡しを最小比較対象とする。技術選択はWeb Studio Ownerに留保。
- **AI/SHO**：同じC1をTable/Graph/Tree/三要素/文書へ投影しても、表示Edgeや集計値から元Relation/履歴版への逆写像が一意に定まらない場合はread-only。AIの固定対象はSource/Subject/Occurrence/値Span/Selection epoch、SHOの文書著者編集は文書Revisionも保護する。AI提案の保存と業務Accepted、実Grant/Effectは独立。
- **対象変更・Source移動・ABA**：古いProposalの対象/依存に差分があれば、既存#2582のClaim相対Required Closureと#4515 Pin資格を再確認。変更と無関係なClaimにまで一律再実行を強制しないが、Read/Source根拠なしの「無影響」は許可しない。
- **外部REA/SUI候補**：Legacy Observed Actual、Owner Accepted Desired、Target Observed Actualを別々のEvidenceとして管理する。REA Resource–Event–Agent分析用のAgent、外部REA解析ツールのNode、実Host Principalを同一視しない。AI/外部Observationが提示した組織変更はCandidateであり、Owner Meaningを自動変更しない。

### 12.5 既存Ownerと次の実証の切り方

1. **B0：構造限定の合格を維持**。現在のDirective・Role/Responsibility・Delegation Go testは構造/非権限化の証拠。BEH-03/04/05の実Behaviorとは呼ばない。新しく全組織一括Txを作ることをP0にしない。
2. **B1：同一DB内の改編原子性**。既存#4490 BEH-03の手書きExpectedに、U/V更新途中の失敗介入、別Readによる旧一貫版または新一貫版の検証を追加して既存Provider/Transactionで実行。Org C1 as-of二版保存が未実装なら当該Propertyの実DB試験はNot Runと明示し、合成だけを製品成功にしない。
3. **B2：Case責任と候補のRead-only動的再評価**。#4490 BEH-04/06と#4501 CX-01〜03の既存Case Scenarioへ、旧新責任4象限、0/1/many/partial、Rule/Case Revの前後を結び、Readの独立Expectedを比較。Host現在Grant/Effectは別。
4. **B3：外部Hostと引継ぎ**。#1174/#1129/#1444へEffect Fence/lookup-first/Receipt/ACK/補償の局所実証を委譲。実Hostが無い場合は保留。#4125のCutoverは旧Writer Fence・DB/Consumerの実Readを別に要求。
5. **P0は不変更**：#2586通常CLI実Function→Provider→Consumerと#2831/4583のTEI Studio Desktopセルフビルドは並列最優先。B1〜B3の全達成をP0のHard Gateとしない。

**現状資格**：本節は既存mainの3つの組織Go構造テストとIssue契約からの設計・引渡し。実Directive Tx、異DB原子性、Candidate動的解決、Web共同編集、実Host Grant/Effect/ACK、Org C1 E2Eは **NOT RUN / NOT ACCEPTED**。#4623の最新Draftは限定Plugin BuildPlanで選択Builderと実exe起動前照合をCI実証したとのPR記録があるが、これは組織RuntimeやDesktop Stage2の受入ではない。


## 13. r7合成意味から型付きMembership・実Providerへ移す最小の意味契約（2026-11-14実Source照合）

### 13.1 実Sourceを横断して特定した3つの不足

本節の根拠は、**main実ファイルの読取**と既存r7の独立Expected。独立Referenceの計算成功を製品の実Query/Providerの合格に転用しない。

| 層と現行ファイル | 実際にあるもの | 同一r7の人物数/履歴を検証するための不足 |
| --- | --- | --- |
| 合成意味: reference/org-semantic-views/source-fixture.json | r7 A1〜A4、Person P1/P2/P3、A2の旧新版2行、Allocation basisPoints、valid/known | 原典はテスト用でありTEI正式SemanticでもProvider Rowsetでもない |
| 型付きSemantic: reference/org-staged-axis/semantic/organization-axes/model.tei | MembershipId・OrgUnitId・allocation・validFrom/To・knownFrom/To、OrgUnitへのunit Relation | **Membership.personId と Personへの明示Relationがない**。IDだけ似た別Relationへ黙示結合できない |
| Persistence: reference/org-staged-axis/databases/org-store/persistence/membership.tei | public.org_memberships・membership_id単独PK・org_unit_id・allocation・valid_from/known_from | Personの実Linkは未Mapping（直接のperson_id属性／既存RelationStorageによる物理support columnを別候補として比較）。記録版行Identity/旧新保存も不足。Draft #4567のvalid_to/known_to追加ではPK/人物Linkは未解決 |
| Native Query: reference/org-staged-axis/applications/org-axes-view/queries/as_of_membership.tei | membership_id/org_unit_id/allocationの3列、3入力。mainにはknown述語等のGAP | Person key・記録版の出所・Query Row Grain・Read Cohortの実Witness。Draft #4567の半開述語修正と区別 |
| Reference Context: verify-context.mjs / verify-acquisition.mjs / verify-rule-footprint.mjs / verify-query-dependencies.mjs | r7同じSource/Scenarioから人物関係・Provider Route差・Requiredと取得/公開の違いを合成検証 | Sourceが実Typed Query→DB Provider→Function Context→一Consumerへ渡ったProduct Attempt |

**修正する誤前提**：以前の補遺で想定した「型付きMembership→Personの追加取得」は、現mainの org-staged-axis ではまだ利用できない。SourceにPersonが別Dataとして存在することと、MembershipからPersonへ取得可能なRelation/Keyがあることは別。Source r7のA2.person=P2を、ProductのQuery結果3列だけから後付けで推定しない。

### 13.2 押さえるべき粒度と単位（r7の固定Expected）

| same validAt=2027-06-15 | knownAt=2027-07-01 | knownAt=2027-07-03 | 実装時に必要な識別 |
| --- | --- | --- | --- |
| Membership occurrence集合 | A1/A2/A3/A4 | A1/A3/A4 | stable Membership ID + record versionの選択理由 |
| distinct Person全体 | P1/P2/P3 = 3人 | P1/P2/P3 = 3人 | 実Person key。A2とA3のP2を二人にしない |
| D1 Membership数・配賦 | 2、160 / 100 = 1.6 FTE | 1、100 / 100 = 1.0 FTE | 当該Departmentの完全なRead Scope、配賦単位 |
| D2 Membership数・配賦 | 2、140 / 100 = 1.4 FTE | 2、140 / 100 = 1.4 FTE | D1の訂正をD2の無関係Claimへ無条件伝播しない |
| 全体Allocated FTE | 300 / 100 = 3.0 | 240 / 100 = 2.4 | JOIN fanout回避、relation occurrence単位の合計 |

**単位の注意**：r7のフィールド名 basisPoints はfixture固有であり、ここでは **100 = 1.0 FTE** と定義されている。実org-staged-axisの allocation は std.decimal / PG numeric。100→1.00、60→0.60等の変換は**明示したMappingと独立テスト**がなければ成立しない。財務上一般的なBasis Point定義をこのfixtureへ無断適用しない。Nullable allocationがある場合、0と欠落/NULLを同一視しない。

時間精度にも異なる前提がある：r7は日付文字列、実Persistenceは datetime-tz。validAt/knownAtとSQL bindのzone/精度/半開境界が一致することをProvider Ownerが確認する。同じ表示日でもOffset違いで版が変化する可能性を隠さない。

### 13.3 Productへの最短昇格を「Source→Model→DB→Query→Context」で定義

1. **Meaning/Key Fit**（#2386/#1126）：r7の Membership.person の正式表現は、Membership.personId＋Personへの直接Relation、またはPerson/Employmentをsourceとする型付きRelation＋RelationStorage等の候補を比較し、Source側Relationの方向/Key/Cardinality・Personの同定根拠をOwner受理する。前者を全Profileの必須属性にせず、属性追加無しの後者もIR上の候補とするが、製品DSL/Providerが実際に実現する資格は別試験。Person/Principal/Account/Tenantへの無許可昇格はしない。
2. **Physical Fit**（#1126/#2622）：採用したRelationの人物対応を、通常ColumnMappingか既存Persistence IRのRelationStorageMapping／SourceAttribute→database-owned support column等でどう保存・取得するかを比較。いずれも実DSL Parser・Validator・SQL Generator・Providerの受入を別途必要とし、IR JSON往復だけで製品実装とはしない。record_version_id等の履歴Keyは§2/§11のDB Owner方式選択に委ね、旧Current Reader/WriterとHistory Readerを区別する。
3. **Query Fit**（#2387/#2586）：Native Queryに必要なPerson linkと版根拠を明示し、実SQLのどのRowがどのMembership Record Versionを選択したか、実Statement/Params/SourceとDataSource Aliasを照合。表示用Queryには過剰な個人データや物理version列の開示を強制せず、Provider Trace/Typed Contextに必要情報を保持可能かを比較。
4. **Context Fit**（#2506/#2624）：r7のD1/D2/全社は別Scope。ResultのRelation occurrence・Person key・allocation・valid/known・Provider Snapshot・read completeness・Required/Consumedを同一Read Cohortで評価。D1のcompleteだけで全世界distinct Person=3を断定しない。Queryableでも未取得ならnot-established。
5. **Consumer Fit**（#2497/#2507/#2516）：一実AttemptからTable/Graph/SHO/Headless等の目的別受信を検査。非表示FieldもQueryのfilter/join/Ruleで実取得され得る。取得と開示を別に追跡し、保護対象Fieldがある場合のみ対象Provider/Recipientで制御する。
6. **Decision/Effect Fit**（#4490/#4501/#1174）：C1候補のRule/Case versionとMembership Readを照合。ただしPersonのMappingが増えただけで承認権限・Case責任・Host Grantが成立するとはしない。

上記は**設計上の適格性の順序**であり、すべてがP0のHard Gateという意味ではない。最初の通常CLI実Provider Readは受注Queryの縦断を優先、r7人物/履歴の再現はその後の組織P1。新しいSQL Resolver・Graph Query Engine・Temporal Runtime・第二Sourceを作らない。

### 13.4 同じsnapshot文字列でもRead Cohortは異なる

既存 scenario-fixture.json は replica / primary の二経路を持ち、両方のsynthetic snapshot文字列は s7 だが、knownAtとProvider/Mappingが異なる。verify-acquisition.mjs と verify-rule-footprint.mjs は、**D1がreplicaでA1/A2、primaryでA1だけ**となる合成反例を既に所有する。

従って、単なるSnapshot名・Semantic Subject・Source Revision一致によるContext混合は禁止。必要に応じて selected DataSource Alias、actual DB/Provider + version、Route ID、valid/known cut、Observed Read Cohort、Mapping Revision、Completenessを同じClaimのRequirementへ含める。完全な一致Witnessのない二Providerの集計を同一C1 Decisionの証拠へ合算しない。純粋r7計算の2経路に実RLS/DB isolation/current Grantがあると主張しない。

**最初の反証を固定する**：replica knownAt=2027-07-01のD1 2件/1.6 FTEとprimary knownAt=2027-07-03のD1 1件/1.0 FTEを、いずれもsnapshot=s7という理由で混ぜない。実Productでは両Sourceを使い分けるBusiness Rule/Consistency Guaranteeの有無が未観測ならunknownを保持する。完全に取得したD2だけの独立Claimは、RequiredにA2が含まれない条件を実確認できるなら局所再利用を検討できる。

### 13.5 Web Studioの共同編集はHost Scopeを跨いで移植しない

Draft #4621 §16で整理したStudio Webの **2 Workspace × 2 Principal × 2 Provider** 読取比較と、一Source二SessionのCAS/旧Preview拒否を、C1の編集対象Identity・Source Revision/Selection Epoch・実Providerに適用する。

- DesktopのローカルWails/Go + Git共有、Webの長寿命共有Server、外部Host/Consumerは**異なるHost Scope**。同じ見た目のC1を開いていても、選択中のWorkspace/実Principal/Source Rev/Provider Bindingが違えば一つのActionに合算しない。
- Webで一ユーザーが同じSourceを再読込したり一つのAI Harness Sessionを共有しても、別Principal/Tenantの現在権限は自動継承しない。local no-auth状態をWeb Serverの認可fallbackにしない。
- 同じSource spanの別Sessionによる編集でCASが失敗した場合、古いPreview/AI結果を黙示的に上書きせず差分を再提示。どのField/Relationが意味保存で書き戻せるかは#2497/#1124の判断であり、物理DB PKをUI/Graphの表示行IDへ固定しない。
- 公開先のSHO/AI/CLI/外部Toolへは目的別Disclosureの実Result/Recipientが必要。Serverに届いた事実は特定Consumerが受領・理解・権限付与された証拠ではない。WebのIdP/HA・実HTTP・CAS E2Eは未検証。

### 13.6 受入切り出しと追跡先（既存Expectedを再利用）

| 実行順 | 既存の検証と新しい合格判断 | 失敗時の停止と担当 |
| --- | --- | --- |
| A0 Source Fit/Gap | 現main型付きMembershipにはPerson Relationが無いと明記し、OwnerにMeaning+Key案を返す | 既存の合成r7人物数をProduct Query成功と呼ばない。#2386/#1126 |
| A1 Persistence/Query Fit | r7のA2二版・person_id・version rowと保存・Query Mappingを一貫検査。#4567のvalid/known述語修正は別候補 | Query 3列、PK単独、人物結合無しを放置して人数合格にしない。#1126/#2622/#4490 |
| A2 Context/Scope Fit | 既存r7のPerson3・Membership4→3・FTE3.0→2.4とD1/D2別Claimを、実Read/Required/Completenessと比較 | s7の別Route・旧knownAtと新knownAtの合算は拒否。#2586/#2506/#2510 |
| A3 Org C1/Consumer | 対象Case/Rule/ResponsibilityのSourceを別に受理し、候補0/1/many/unknown、Viewer/SHO/Headless受領を検証 | r7自体にCase/Current Grant/Effectがあると見なさない。#4490/#4501/#2516 |
| A4 Host/Collaborative | 現在Grant/Effect/ACKとWeb/AI Source CAS・二Principal、二Providerの受領を別Attemptで照合 | 実行者同一視、AI提示/ACKのみ、OS/Server権限の黙示継承を停止。#1174/#4515/#3421 |

**新規実装や新Runtimeの採用は本節では行わない**。現main、既存r7 Reference、Draft PRを突合した設計・受入条件のみ。実TEI Person Relation Writer、履歴型PG、実Query→Context→Consumer、Web二Session、IdP、Host/ACK、Owner Meaning Acceptanceは **NOT RUN / NOT ACCEPTED**。


## 14. Person結合のMeaning採否・複数DB Alias競合・並行PRの統合順（2026-11-14）

### 14.1 Person参照を追加することと、人物を確定同定することを区別

mainの `organization.Membership` は `orgUnitId` 等を持つが Person Key/Relation が無い。対照的に `organization.Employment` と `organization.Affiliation` にはそれぞれ personId と person Relation が既にある。しかし、Employment/Affiliationを辿ってMemberのPersonを**自動同定してよいという業務Rule**は宣言されておらず、EmploymentからPersonへ辿れることはMembershipからPersonへ辿れることの証拠にならない。

**意味Ownerへ差し戻す候補**（公開Core Kind・新汎用Member supertypeを提案しない）：

| Profileにおける候補 | 向くScenario | 受理前に解決すること |
| --- | --- | --- |
| **M-P：Member Personの直接Relation**（Membership.personId→Person.personIdは実装候補の一つ） | 従業員・個人の所属を保持するS1/S2/r7 | 原SourceとCardinality/有効期間に加え、PersonをMembership属性で保存するか、Source側RelationとRelationStorageの物理support columnで扱うかを選ぶ。全ProfileでpersonId属性を必須としない |
| **M-E：Employment/Affiliation経由の型付きRelation** | 出向・複数法人など雇用／在籍を別正本で管理するS2/S4 | Membership→Employment/Affiliationの**キーと経路**が別途必要。同じPersonに複数雇用があるとき、自動で一件を選ばない |
| **M-X：Member Subjectを別Domain Dataへ束縛** | チーム・法人・組織間参加など人物以外がMemberの場合 | Person人数のClaimへそのまま使用できない。既存Data/Relation表現で十分か、別Profile/Ruleの採否を先に比較 |

M-Pを最初の小Reference候補にするのはよいが、**所属という語が現れる全モデルをPerson参照必須にする理由にはならない**。どの型を採るかより、既存Source/Person ID/Relation occurrenceの根拠を同じReadと結び付けられるかを優先する。

**曖昧なIdentityの扱い**：
- 外部HR/IdP/ERPの `sourceSystem + externalId + sourceRevision` と、TEIで業務上同一と受理されたPerson IDは別。Name/Email/同名同日/部署名ではPersonを確定しない。ID対応が1対0/1/複数、権限不足、部分取込のとき、未解決のまま保留する。
- `membershipId` は業務Occurrence、recordVersionIdは記録の版、physical row IDはDB固有。遡及訂正でPerson参照が変化した場合、「過去の誤同定の訂正」か「実際の別人への所属移転」かを業務Ownerが判定する。後者を同一Occurrenceへ無断統合しない。
- 入力Snapshotが部分的である場合、見つからないPersonを退職/所属取消扱いしない。Reference r7にはそのSource範囲でPerson Linkがあるが、Product実ProviderのIdentity照合を代替しない。
- 本務/兼務、Employment、Membership、RoleAssignment、Case Responsibilityは別Relation/意味。人物Linkの追加でCurrent Principal/Account/Tenantや承認権限を導出しない。

### 14.2 複数Persistenceの潜在衝突を現行Operation実装で確認

**mainコード読取**：`internal/operations/surface.go` の `deriveApplicationOperations` はAlias名をsortした後、各Alias配下のPersistenceを走査し、`operationID(app, kind, semanticData)` を利用する。`emitOperation` は `seen[operation.ID]` があれば後続候補を捨てる。この箇所はmapping/DB/aliasの同一性を比較せず、`OperationSurface.Conflicts` に異Mapping衝突を記録していない。

これは現行組織Profileに**直ちに実データ誤更新が発生したという証拠ではない**。しかし#2611 PM-01〜06に明記された多DBの再現条件で、旧Current・新Historyの移行時や二Providerの生成CRUDを無条件に合格とできない。

| 再利用する#2611受入 | 組織C1における具体条件 | 必要なOutcome／反例 |
| --- | --- | --- |
| PM-01 重複Aliasの正例 | 同一DB/同一MappingをRead/Write Aliasから参照 | 一つのOperationを安全に重複排除でき、従来の正例を維持 |
| PM-02 異DB・異Mapping | legacy-current と history-provider が同一 organization.membership に束縛 | 一つのOperation IDだけで一方を暗黙採用せず、実Target選択/同値関係を未解決とする |
| PM-03 読取と更新の差 | archive/historical Readerとlive Writerを同じ業務Dataに関連づけ | 読取のas-of/Scopeとwriteの対象PK/Provider/Grantを別契約で選択 |
| PM-04 Alias名称/順序変更 | historyのAliasが名前順で旧Currentより前になる | SourceのAlias改名だけでwrite対象が逆転する実行経路を許さない |
| PM-05 Mapping Basisの不足 | PK/Table/Mapping IDは同じでもperson_idやversion column/NULL制約が変わる | Basis同一から互換性を推測せず、実Input/ColumnMapping/World/Planを再資格化 |
| PM-06 Current vs History | 同じBusiness IDだが期間/Row grain/Record Scopeが異なる | 現行1件Readerを過去全版Readerとして使わず、全件countとas-of件数を別Claim |

**実装への引渡し**：#2611/#1127の既存Operation Surfaceで、(a)完全に同じ実対象か、(b)解決済みの読み/書き別Bindingか、(c)異Mapping競合でunresolvedかを、既存Source/Mapping/Provider契約の根拠を持って診断する。既存 `Conflicts` / Gapの利用可否を先に調べ、新しいResolver/Registryや組織専用CRUDを設けない。読取Query選択やHostでの実Targetは#2586/#2622/#1174の別資格であり、Surface内に更新許可を埋め込まない。

### 14.3 まず1つの反証可能な「人物同定×別DB」を設ける

#4490 ORG-BEH-01/02、#2611 PM-02/04/06、#2386 S4を**別々の独立Expectedのまま**使い、同じ組織C1のScenario/Attemptに関係する識別を対応付ける。新しい統一Expected Storeは作らない。

1. **P-A 成立するPerson Link**：r7でA1→P1、A2/A3→P2、A4→P3を固定。Productに同じ意味が受理され、Person Read Traceがあるときのみdistinct Person=3を要求（Readが全対象Scopeでcompleteであることが必要）。
2. **P-B 意味同定の反例**：P2と別人P2bが表示名だけ同じ、またはHR SourceのPerson外部IDが曖昧。Person数の結論はunresolvedであって、Personを統合した3人や重複した4人へ確定させない。
3. **P-C DB Bindingの反例**：Current DBとHistory DBにA2が見えるが、片側の実Target/Mappingが異なる。Operation IDとDisplay Labelが同一でも、順序に応じた片側採用を正常扱いしない。
4. **P-D read-cohortの反例**：r7 replica と primary のラベル `s7` が一致してもknownAtが異なる。各取得結果のScope/valid/known/Provider Snapshotを揃える独立Witnessが無ければD1 FTEを合算しない。
5. **P-E History/Write境界**：現行行を旧Writerが上書き可能な状態で新Historyの公開を完了扱いしない。Commit不明、旧Writer Fenceなし、Receipt照会未実施はG4 cutoverへ昇格させない。

**初回の小PRのExit**：OwnerがM-P/M-E/M-XとIdentity Sourceを判断した後で、最小の型付きPerson Link（採用するProfileに限る）＋物理Mapping＋実Readの独立小PRへ切り出す。#4566/#4567の二時点Query修正は重複するため差分比較を先行する。#2586通常CLIの受注Query Readと#2831 Studio Desktop Self-buildは、これらの組織固有課題を前提としない独立P0を維持。

### 14.4 実際に競合している計画Draftの無損失統合手順

**source readbackで確認**：現mainの `ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md` は §10.11.4 までで §10.12 がない。一方、開いているDraft **#4555** は §10.12 に「Studio Desktopと組織C1の最短実装境界」（D0〜D4、D2-SB負例）を追加し、**#4585** も同じ §10.12 に「所属とPlugin・B2C・外部Hostの六Scope」を追加している。双方を単純な後勝ちでmerge/全文置換すると、一方の受入条件を失う危険がある。

- **提案する前方統合順**：#4555の節を §10.12（DesktopとD2-SB）に、#4585の節を **§10.13（六ScopeとHeadless）** に保持する。どちらを先にmergeする場合も、後着側は新mainへrebaseし、節番号と節内参照だけを調整する。既存の§10.11本文は保持。
- **§10.13の内容は縮退させない**：Distribution/Authoring/Composition/Organization/Runtime/Disclosureの個別Scope、r7にないTenant/Account/Case C1をScenario overlayとして分離する規則、B2C Headless、CURRENT Grant、Disclosureの二世界負例とO0〜O4を残す。§10.12のD0/D1/D2/D3/D4、D2-SBの各反例・実Windows+Builder Stage2の資格も残す。
- **本PR #4622の扱い**：Stage Planの §0 に**別設計補遺への参照リンクだけ**追加してあり、#4555/#4585の§10.12本文を置換していない。補遺§5/§6/§10〜14は横断の受入索引であり、両PRの独立した元計画全文の代替にならない。各PRの採用・merge順はそれぞれのOwner判断。
- **MASTER_PLAN競合**：#4555/#4585/#4620/#4621/#4622が同一計画を並行編集するため、最新mainの各独立節とリンクを集合として照合して、追補節・受入票・証拠限界を失わずforward-portする。形式的な同名節の重複、旧HEADのDraft局所CIを新main成功に合算しない。

**状態**：組織Meaning/物理PK/Provider Binding/見込Person Identityに関する設計・受入案のみ。本節では新Go/SQL/Node/DB/Generated CRUD、実DB Alias競合試験、Web同時編集・実Host権限/Effect、Owner Meaning Acceptanceを **NOT RUN / NOT ACCEPTED**。PRの重複箇所は上記の衝突事実を確認したが、#4555/#4585本文の自動統合は実施せず、Owner間調整を要する。

## 15. 組織C1の最小検収票：独立Expected・実行Coverage・局所再資格化（2026-11-14）

### 15.1 技術的なPASSと、主張のVerifiedを別に保つ

他レーンの[#2514]独立Conformance/TQC（検証基準の品質と適用範囲）、[#1180]Driver Scenario／実行0件・SKIP偽PASS、[#2600]旧依存辺を含む増分再資格化を、組織固有の**既存**ORG-BEH-01〜07、#2611 PM-01〜06、#4501 CXへ接続する。各Referenceの期待は実装/生成結果から生成しない。新しいScenario SSOT、第二Qualification Engine、組織専用Runnerは作らない。

次の二軸を証拠票上で分離する。

- **Case実行状況**：not-selected / selected-not-started / started-not-completed / completed / skipped / unsupported（#1180の検収分類）。実行されなかったCaseは「失敗0件」の分子にも「合格した必須Case」にも加えない。
- **観測結果と利用資格**：pass / fail / inconclusive / unknown と、#2514の Verified / Failed / Incomplete / Not Executed / Stale / Not Applicable を区別。後者は正式な公開enumを指定するものではなく、Required/Source/Scope/Expected/Version/Consumerを照合するための**検査票の語彙**。局所PASSはRequired Claimの独立Expected・Coverage・実際の対象・当該HEADが揃うまでVerifiedにしない。

**空の成功を拒む**：Go testの0件実行、PostgreSQL opt-inのSKIP、r7のNode合成のみ、構造Go testのみ、未ロードのModel/未取得のProviderを、製品BEH/PM/CX合格へ昇格しない。実E2Eを要求するCaseについてCIの緑・テスト0失敗でもRequired memberの母集団と実行数、SKIP理由を併記する。対象Runが別HEADの場合はそれぞれの資格上限を維持する。

### 15.2 最初の実証に必要な9つの対照（新しいExpected IDを発行しない）

| 既存Expectedを使う組織C1圧力 | 意図的に混ぜる／壊す入力（Mutant） | 独立Expectedによる区別 | 主担当 |
| --- | --- | --- | --- |
| ORG-BEH-01/PM-06 二時点保存 | 同じA2の旧版を上書きし、knownAtだけ変えて実行。別DBへの入れ替えで版共存を偽装 | 同一DBに旧/新版が同時保存されていなければHistory Co-storageは未資格。実Readが両方の既知時点を再現 | #1126/#4490/#4567 |
| ORG-BEH-02 Personと粒度 | 人物P2のA2/A3を別人として数える、Person Link無しで件数だけr7の期待に一致させる | 全ScopeのPerson Linkとcomplete Cohortが無ければDistinct Person Claimはunresolved。r7の3人は合成Expectedであり実Provider証明ではない | #2386/#2506/#2586 |
| PM-01/02/04 Current/History Binding | 同じOperation IDに旧新2 Mapping、Alias名称順を逆転する | 真に同じ実Target/Mappingのみ重複排除可。異なるWrite候補は選択根拠無しに実行可能としない | #2611/#2622 |
| ORG-BEH-02 Read Cohort | Replica/Primaryが同じs7表示、knownAtが2027-07-01/03 | D1は1.6/1.0 FTEと別結果。Scope/時点/Providerが異なるRowsetを一つの完成したClaimへ合算しない | #2506/#2611 |
| ORG-BEH-04/06/CX-01〜03 Case権限 | Membershipだけ変更しCase Rev/Rule Revは旧、旧新Grantの有無を変える | 候補0/1/複数/partial・Case責任・Current Grantは別結論。双方無効という断定はRequired未閉包なら不可 | #4490/#4501/#1174 |
| ORG-BEH-03/TO-05 Commit Unknown | 複数Membership更新途中でタイムアウト・ACK喪失、返信上は失敗 | 当該Operation IDのEffectはunknown。lookup-first/Receiptと独立Reader確認前に非冪等な再Submitを許可しない | #1129/#1174 |
| TO-01/02/03 実行Coverage | PG接続無しでtest skipped、0 tests、旧Suiteのみ全PASS | 局所PASSと全Claim未資格を分ける。必須Test/Target/HEAD/実行母集団の不足を保持 | #1180/#2514 |
| DI-01/02/03 依存削除とProvider変更 | A2→Personの旧辺が削除される、Aliasだけ変わる、Query同IDでWHERE/期間が変わる | 変更前後のRequired閉包と実消費を比較。旧Evidenceの誤再利用を拒み、関係ないD2独立Claimを根拠なく失効させない | #2582/#2600 |
| CX-07〜12/TO-06 Pin/外部受領 | Source/Selection ABA、外部SHO/AIが一部しか受領しない、非公開Fieldのみ変更する二世界 | 保存/Host結果とConsumer受領・Allowed Disclosureを別資格にし、非単射Graphからの逆編集・情報漏洩を拒否 | #4501/#4515/#2516 |

これらは**既存Scenarioに付帯する故障注入と検収条件**であり、実行されていないケースの新しい通し番号や独立Expectedファイルを創設しない。「r7に存在しないCase/Grant/Hostデータ」は独立Scenario overlayとして明記し、r7 Source自身にあると偽らない。

### 15.3 同一Attempt内で観測する5段階と資格上限

| 検査段階 | 最小で観測するもの | ここまでで言ってよいこと／言ってはいけないこと |
| --- | --- | --- |
| **S0 Source/Scenario** | Accepted Source ID/Revision・Person Link/Occurrence/Record Version・固定独立Expected・Required/Target | 入力と期待の所在を特定。Source一致=実Rowset正しい、ではない |
| **S1 Typed Meaning/Projection** | DSL Data/Relation/Key・Query候補とMapping・書式/Scope/粒度・Source Span、未解決の関係 | 意味的な入力候補と静的診断。構造Go PASS=動的候補・実Provider成功、ではない |
| **S2 Actual Provider/Query** | 選択したDB Alias/Mapping/SQLと**実呼出した**DB/Statement/params、返った行・期間・completeness、実Provider Instance | 実Readが限定範囲で確認できた。Person/Historyの欠けた列を補った、ではない |
| **S3 Function/Consumer** | Result Slice/Required/Grain・実Contextと外部Consumerの受領したResult/epoch/Disclosure | 用途別の局所Claimだけが検証可能。送信=受信、表示=実行資格、ではない |
| **S4 Write/Effect/Receipt** | Host Current Grant/Action Resource Rev、Operation ID、実Commit/Receipt/lookupとHandoff ACK | 対象Host/Providerの観測可能範囲に限りEffectを判定。read-only testで書込資格を付与しない |

**必須ではない段階を明示**：初回read-onlyの#2586 P0はS0〜S3でよい。S4を通していないからS0〜S3の局所PASSを失敗としない一方、承認/引継ぎ完了はClaimしない。#2831 Studio Stage2セルフビルドは別Product/Builder/実DesktopのS0〜S3相当であり、組織C1の実DB/Host成功へ合算しない。

**同一Attemptである条件**：Invocationの実Child/Provider/Sourceを観測したものを一致させる。別CI Runのノード合成、別PG Snapshot、別Windows Head、別Target Consumerからの局所緑を寄せ集めない。外部Effectは遅れて独立照会する場合があるため、同一Operation/Attemptの因果識別を残すのであって、同一時刻の一括完了を仮定しない。

### 15.4 変更後の局所再資格化を、誤再利用と過剰失効の両側から検査

対象Case/Claim Cを再利用できるのは、今回の問いに対する Required/Consumed のSource、Provider、Query・Rule、期間/Scope/Recipient/Current Grantが十分に対応し、変更前後の依存（**削除された辺を含む**）で非影響が説明できる場合だけである。

| Changeの対照 | 維持可能な局所Claim | 再検証・保留するClaim |
| --- | --- | --- |
| D1のA2訂正・同じvalidAtでknownAt進行 | D2が独立した完全読取でA2に依存しないと証明できる場合のD2 FTE | D1 Member/FTE、A2依存承認、Global Person CountはRequiredと実Read再照合。人数の値が同じでも資格は別 |
| OrgUnitの表示ラベルだけを変更、業務ID/Relation/Rule不変 | Source/実Rowset等の根拠が揃うMembership数・配賦Claim | ラベルを表示するConsumer再投影やSource使用先、必要な開示再判定 |
| Person Linkを追加/削除、又は旧Membership→Person依存を削除 | Person非依存の完全な組織単位別Membership Countは維持候補 | distinct Person、Person-based Rule、Source edit対象。旧辺を捨てて「依存なし」にしない |
| 同じSemantic IDのSQLを別Alias/Mappingへ選択変更 | Providerを使わない純粋受理済みMeaningは維持候補 | 実SQL/params/Rowset・current completeness/Grant/Consumer、旧Alias Write先 |
| A→B→AでSource値/表示が元通り、Plugin/Rule/Recipient/Version変更 | 変更と無関係が証明された用途限定Claim | 旧Pin/Operation/Effect Receipt/Evidence・検証基準の適用資格を現行と同一視しない |

この表は**再利用候補の判定方法**であり、未観測のDependencyやRead Scopeを自動無影響へ丸める仕様ではない。#2600がClean/Incrementalの同一Source/Targetで false reuse（必要再検証の未実施）とfalse invalidation（不要な再検証）を独立計数する。実測時間短縮の前に、検査結果の正しさを満たす。

### 15.5 着手順と今回「決めたこと／まだ決めないこと」

**直近の三つの小作業（段階間の未達を隠さない）**：

1. **R0（read-only・即着手可）**：現mainとDraft #4566/#4567のTyped Membership/Person、Persistence、SQL、OperationSurface・同一Alias/異Aliasを、上の9対照の「実装済・静的確認・参照のみ・未実行」に分類する。#2514の既存検査票へRequired/Actual Execution/Expected/Exact HEADを付ける。Missing Person Linkを黙ってProduct PASSにしない。
2. **R1（Owner選択後）**：#2386/#1126がPerson RelationのM-P/M-E/M-X選択を受けたProfileで、#4490の最小同一DB二版Readと#2611 PM-01/02/04のAlias衝突診断を別々の小実装・独立Expectedで実証。片方の成功を他方の合格へ流用しない。#4566/#4567重複修正のMerge差分調整を先行する。
3. **R2（Product接続）**：#2586通常CLIの実Provider→Function Data Context→独立Consumerを最初のRead-only縦断とし、組織r7への適用はR1意味/物理整合後の別実行。#2831 Stage1→Stage2実Desktopセルフビルドは完全並列P0。C1のHost Effect/ACKは#1174/#1444が独立Owner。

**採否境界**：採用済み（再利用する）＝既存Typed Semantic/Operations/Driver/Provider/Conformance基盤と既存r7・PM/CX/ORG-BEHの対象。今回の設計で採用しない＝単一Person必須Core、Org専用Operation/Temporal Runtime、第二Scenario/Expected/Qualification Store、第二Migration/Release Gate、独自Host、万能Web共同編集Engine。未決＝Person RelationのProfile別意味・物理Version PK・修復/過去Retention・DB跨ぎCutover/Writer Fence・現在GrantとCase引継ぎRule。P0の範囲を変えない。

**本節の成果資格**：現Issue/PR/mainファイルの静的再照合と受入設計。9対照の新Test、製品Pg/CLI/Windows/Host実行、独立TQC判定器実装、Owner Meaning Acceptanceはいずれも**NOT RUN / NOT ACCEPTED**。


## 16. 最初の実Go GAP特性テストへの引渡し（2026-11-14）

§15のR0「実Sourceに対する検証状況の棚卸し」を、**別の小さなCode Draft [#4633](https://github.com/hat47x/tei-platform/pull/4633)** に切り出した。組織意味/DDL/Query/OperationSurface/Providerの変更を含まないGoテストのみのPRであり、#4622は引き続き横断設計・追跡Planを所有する。**既存PR #2632** は#2611の7つの多Mapping Characterizationを既に実装していた（Closed・Unmerged）ため、#4633ではその再実装をしない。

| #4633に実際に作成した検査 | mainの実Source・独立Expectedに対する検査 | 成功で主張できる上限 |
| --- | --- | --- |
| CurrentMembershipDoesNotHavePersonLink | Workspace Loaderでorganization.Membershipを読込み、Person/Employment/AffiliationへのRelationが無い一方、Employment/Affiliation→Personの正例は存在すると確認 | 現mainのSemantic欠落を実Goテストで特徴付ける。M-P/M-E/M-XのOwner意味採用ではない |
| CurrentPersistenceCannotCoStoreTwoVersionsByMembershipPK | 現Mappingのpublic.org_memberships、membershipId単独PK、Person mapping列不在 | 同一A2旧新両版の保存が現Mappingでは未対応と示す。実PG保存/移行ではない |
| CurrentAsOfQueryDoesNotYieldPersonIdentity | 本物のorg_axes_view.as_of_membership QueryのResultColumnsが3列でPersonを射影していない | Query単独からdistinct Personを証明できない現状を再現。実SQLのRowset成功ではない |
| SyntheticOracleHasTwoA2VersionsAndSharedP2 | 既存r7 Source JSONでA2旧版/新版、P2がA2/A3両方に属することを固定 | 既存Source/OracleとGAPを結ぶ。別Expectedの創設・Node Referenceの代替ではない |

4つの検査は**現状の欠落を表明する特性テスト**であり、開発が進み欠落が解消されたら対応する「不在をassertする検査」を**独立Expectedに基づく正例Product Conformanceへ置換**する。GAP合格を製品実装完了と表示しない。新CI exact HEADでのGo実行結果とGofmt/Repository Healthは、検証済みのRunごとに記録し、未完了の間はpassとは記載しない。

**次のP1実装の引渡し**：#2386/#1126がM-P/M-E/M-XとRelationKeyの採否を判断し、#1126/#2622が実DB保存版、#4490/#2586がProduct Query→Context→Consumerの独立Expectedを検証する。#2611の異Alias競合は既存#2632の特性とProduct Ownerの採否を参照し、現#4633へ取り込まない。Org C1 Host/Effect/ACKは後段。#2586通常CLI P0と#2831 Studio Stage2 self-build P0は独立して維持する。


### 16.1 他レーンから取り込む限定的な製品事実（2026-11-14）

現mainへmerge済みの[Script Launch #4629](https://github.com/hat47x/tei-platform/pull/4629)（squash 76b4d29a）では、Capabilityの**不正なResult宣言をAdapter呼出し前に拒否する構造Preflight**が実装された。これは組織モデルの `organization.approve_membership_change` のようなCapability候補の「宣言整合性」には再利用可能な検査境界だが、**そのCapabilityが現行HostにBinding/選択/認可され、本人の承認行為を実行できる証拠ではない**。C1での実行資格は次の4つを別々に記録する。

| 証明する対象 | 現在確認した到達点 | C1で実証していないもの |
| --- | --- | --- |
| CapabilityのInput/Resultの構造整合 | #4629 mainのGo Preflight・Adapter call0の局所成功 | Owner Meaning、Ruleの実候補、Business Key/Person Link、実Host Grant |
| どのProvider/DB/Mappingが選ばれたか | #2611の複数Alias課題、Closed/Unmerged #2632に特性テスト7件 | #4633で実Provider Route/Write Targetを確定したという証拠 |
| Providerの実Callと帰結 | #4567の別Draft・限定PG as-of Read、#4539の受注実PG Read | 組織C1 as-of版共存・ケース承認・実Effect/Receipt |
| 消費者へ実際に何を返したか | r7 synthetic Context/Disclosure、#2516別Owner | C1のOwner受理済み通知/ACK・Purpose別実受領 |

将来のStudio Webについては[Wails Server公開メソッドInventory Draft #4627](https://github.com/hat47x/tei-platform/pull/4627) が **Goの公開面を固定し、不意のService/Method追加を検出する局所テスト**を実装中。これはWails Serverの認証済みPrincipal×Workspace×DB Providerの実分離、WS BroadcastのAudience隔離、SHO/AIへの安全なData Disclosureを証明しない。#4621 §17〜19/#3421の実Host Scope受入へ委譲し、組織モデルに新しいAuthZやWails Serverの固定依存を追加しない。

**PR #4633の検収上の注意**：このCode PRは単独の新規Go testファイル4件で、既存モデルとr7の欠落を検出するもの。exact HEAD `b529791f452231f06e16d6419b0f80cace12d6fb` のCIは完了したWorkflowだけを局所成功に数え、**Repository Health内の `go test` 合格はそのジョブ/ログが完了するまで未確定**とする。別のWebView smoke、Action Surface CLI、Self-build、既存Postgres Workflow成功を新しい4テストのPASSに流用しない。#4633はmainの既存Go testと競合しない別ファイルを追加し、OwnerのMeaning/PK決定に先行した製品Schema変更をしない。

**コード化の次の停止線**：#4633が静的GAPを検出した後、意味OwnerのM-P/M-E/M-X選択、既存#2611のAlias conflict treatment、#4566/#4567の二時点SQL/Mapping修正差分を受理できる状態になってから、同一DB A2二版保存・実Rowset/Person/FTE・Current Grantを別々の小PRで実証する。read-onlyの通常CLI実ReadとStudio Stage2 self-buildはこの判断をHard Gateにしない。


### 16.2 実行済みGo特性テストの固定HEAD検収（2026-11-14）

**新しい実行証拠**：[Draft #4633](https://github.com/hat47x/tei-platform/pull/4633) 固定HEAD b529791f452231f06e16d6419b0f80cace12d6fb について、[GitHub Actions Repository Health #43548199759](https://github.com/hat47x/tei-platform/actions/runs/43548199759) が completed/success。実ジョブログで Go全件テスト（go test -count=1 -timeout 60m ./...）PASS、tei/internal/operations 2.435s PASS、gofmt -l . と go vet ./... PASS。4つの TestOrgC1... はskip無し、同packageの通常のGo全件testが合格している。

この成功は**本物のWorkspace Loader/IR/Query Mappingと現mainの欠落**を特性化できる証拠であり、r7のPerson Link/履歴Versionの**製品機能が成立したという意味ではない**。同HEADのStudio Wails、Action Surface、セルフビルド、PostgreSQL各Workflow successは別Claim。既存r7の合成Sourceと実DBを混同しない。

| 完了したR0 | 依然未実行の次段R1/R2 |
| --- | --- |
| 4 Go特性テスト、実Workspaceロード、r7 Source検査、full Go/gofmt/vet/Repository Healthのexact-head実成功 | M-P/M-E/M-X Owner採否、Person Link＋Physical Mapping、A2二版の同一DB実保存、正しいas-of実Rowset、実Function Context/Consumer、Host Grant/Case Effect、Owner Acceptance |

**設計／コードの受入境界**：#4622がdocs-onlyの計画Draft、#4633が1ファイル追加のコードDraft。既存Closed/Unmerged #2632の多Alias候補特性は新mainでは実装済みと扱わない。#4566/#4567のNative SQL修正は別Draft・未merge。これらのSource/CI結果を、実際に同じProduct Attemptへ統合しないまま「組織C1のE2E PASS」としない。

**直近の最小実装**は #2386/#1126で直接Person・雇用経由・非Personの意味Profileを選ぶことに限定し、採用Profileの具体Source RelationとRead Witnessを先に作る。DB履歴/別Aliasの安全診断は既存Ownerの別実装とする。固定されたProduct Meaningを人事・勤務・承認の現在権限と自動同一視せず、通常CLIの実Provider ReadとStudio Stage2セルフビルドのP0を独立維持する。


## 17. 属性を増やさないPerson Relation保存候補と、Go検証の資格上限（2026-11-14）

### 17.1 新しく見つかった既存基盤と二方式の比較

main `internal/database/persistence/model.go` の `Mapping.RelationStorage []RelationStorageMapping` は、Source側Relation IDと `RelationStorageComponentMapping{SourceAttribute, Column, Storage, Nullability}` を分離する。コードコメントは**target Semantic Dataへ新属性を増やさず、関係Source Data上の属性をtarget物理表のsupport columnへ保存**する設計であることを明記する。

従って §13/§14の「Person Linkを明示する」という要件は維持しつつ、**Membership.personId属性と同名DB列を必須とする前提は撤回**する。現mainの `organization.Membership` にPersonへの型付き経路がない事実と、IRに物理的RelationStorage候補の表現能力がある事実は両立する。

| 比較する候補 | 明示すべきSemantic関係 | 採用前に検証する物理対応・限界 |
| --- | --- | --- |
| **Direct attribute** | Membership.personId→Person.personIdなど、属性・Relation・Key/CardinalityをOwnerが受理 | ColumnMappingの型・NULL・FK/期間・History版と実Providerの一致。現Sourceでは未実装 |
| **Relation-backed support** | Source側Person等からMembership等への型付きRelationとSourceAttribute/Targetの方向を明示 | 既存DSL Parser→Loader→World Validation→Physical Desired/Migrationには汎用実装があり、組織C1の具体的Relation/方向・SQL/実Rowsetへの適用は未実証。IR JSON往復だけでC1の適合を保証しない |
| **Employment/Affiliation pathway** | Memberと雇用/在籍の受理済みLinkを追加 | Personへの既存Relationが存在しても、その逆方向/Source ID/個人の同定Ruleがない限り取得しない |

**非採用**：Mappingの物理support columnをBusiness IdentityやSemantic Attributeに自動昇格しない。表にperson_id列が見えるだけでPersonを同定しない。RelationStorageという形をすべてのMembership実装へ義務化しない。既存#1126のRelation/Business KeyとPKの分離、#2622のProvider capability、#2611の二DB Alias競合をそのまま使う。

### 17.2 コードGAPテストを実装変更に強くする

[実Go Draft #4633](https://github.com/hat47x/tei-platform/pull/4633) に**5つ目の孤立したGoテスト**を追加し、仮の `RelationStorageMapping` をJSONで往復させ、Membershipの通常ColumnにPerson Attributeを導入せずRelation ID/SourceAttribute/物理support Columnを保持できる**IR構造上の可能性だけ**を特徴付ける。また既存の実Persistence GAPテストには `mapping.RelationStorage` の検査を追加し、将来のsupport mapping実装を単に「person_id通常Columnがない」理由で見落とさないようにした。

- 以前の4件が実Sourceの欠落とr7を追跡するのに対し、5件目は**未受理の仮Candidate**を使ったGo構造のSerialization Test。Product Sourceに追加したと主張しない。
- **Go型・JSONの候補成功 ≠ DSLで宣言できること ≠ Query/SQL Providerの一致 ≠ 業務上のPerson Link受理**。現実のProduct Read/Effect/Current Grant/Consumer ACKに昇格しない。
- 4件版HEAD b529791fのHealth #43548199759、5件版HEAD 509296692343786d96906faabe1defdb17a0ad68の[Health #43549877695](https://github.com/hat47x/tei-platform/actions/runs/43549877695) はそれぞれ完了・success。後者の実ログで全Go test、tei/internal/operations、gofmt、vetがPASS。6件目を加えたHEAD efa4a8fcのCI/Healthは改めて判定し、旧結果を新HEADに転用しない。

### 17.3 最短の独立受入を先に固定

1. **意味Owner (#2386/#1126)**：M-P/M-E/M-XのProfileと、属性直持ち/RelationStorageが表せるSource側Relationの向き、欠損/兼務/履歴期間、同一Person Linkの独立Expectedを選択。これは無条件のDSL/Core追加承認でない。
2. **Parser/Mapping Owner (#1126/#2622)**：選んだ一方式について実TEI Source→Loader→Typed IR→Persistence→SQL/Query Readを一つの最小Referenceで追跡。RelationStorageの場合はGo JSONだけでなく、実Source Parser/Validation/Generatorに同じ意味が残ることを確認する。データ分離が成立しない場合は採用を撤回する。
3. **Product DB/Read Owner (#4490/#2586)**：同一A2旧/新版の実DB共存、同一Read CohortのPerson3・Membership4→3・FTE3.0→2.4を既存r7独立Expectedで検査。SQL semantics/NULL/timezone/物理Key/完全性/alias bindingを分離し、旧Current Writer cutoverは#4125へ。
4. **Consumer/Authority Owner (#4501/#1174/#3421)**：Case C1/Viewer/HeadlessへPerson結果を伝えることと、実Hostでの業務判断・公開・引継ぎを別検証。ローカルDesktop/Gitと共有Wails ServerのAuthorityを自動継承しない。

**実装時に欠かせない点**：Relation Storageの補助列を生成できても、その列へ**どのPerson IDを、どのMembershipの記録版に、どのSourceを根拠として書くか**は決まらない。例えばr7のP2にはA2とA3があり、A2には既知時点の異なる二つの記録版がある。実書込の受入では、人物ID・所属Occurrence・記録版・時点と保存先の対応を同一Attemptで追い、実Readで各版を再現する。既存データでPersonとの対応が不明なら、IDを推測して補完しない。補助列の追加、実データの移行、Queryの読取、Consumerの受領は別の検証とする。

**現時点の検証状況**：[#4633](https://github.com/hat47x/tei-platform/pull/4633)の固定HEAD `85f657002f8b9f05dfd119a72565e583985b8494`では、実Workspaceの一時コピーを使ったTEI DSLの読込・Relation検証・オフラインSchemaDiff/DDL生成について[Repository Health #43572492328](https://github.com/hat47x/tei-platform/actions/runs/43572492328)が成功済み。ただし、正式なPerson Linkの採用、実DBの人物ID書込・二時点履歴、SQL Rowset、独立Consumer受領は**未実証・未受理**である。#2586通常CLI実Provider Readと#2831/#4583 Studio Desktop Stage1→Stage2セルフビルドは、引き続き独立した最優先課題として扱う。


## 18. Relation Storage既存実装への接合とC1限定テストの実証（2026-11-14）

### 18.1 「IRしかない」という前提を訂正し、汎用実装を再利用する

旧PR履歴と現在のsource codeを確認した。既にmainへマージされた #246〜#252 と #257 は、**RelationStorageMappingの型、局所/World Validation、Desired Physical Schema、独立authoring Parser、Workspace Loader/Source provenance、Contract Query SQL realization、SchemaDiff→PostgreSQL Migration SQL**を段階的に実装した。現mainの例：

| 現mainの既存機構 | C1で追加検証すべき差分 |
| --- | --- |
| `internal/workspace/relation_storage_loader.go`：`databases/<db>/relation-storage/*.tei` の独立sourceを読込み、**明示Mapping ID**だけで合流し、別Source identityを保持 | PersonをsourceとするRelationのID/方向・Mapping Targetが一致し、他のDB Aliasや同名Sourceに誤結合しない |
| `internal/validation/persistence_relation_storage.go`：Relation存在/target Semantic Data一致/source Attributeのowner一致を検査 | Person→Membershipを採る場合、SourceAttributeがorganization.person.person_idに属し、target=organization.membershipであること。Membership.membership_idの誤参照を拒否 |
| `internal/database/persistence/realize.go`：通常Columnとは別にsupport columnをDesiredへ追加 | 当該人物support列が明示Type/Nullableで生成され、既存Membershipの単独PKに**自動参加しない**こと |
| 既存#257のSchemaDiff/Migration Conformance | 単なるsupport列ADDと、A2二版の履歴保存用PK変更・既存Current Writer Fenceを別のMigration受入にする |
| 既存#251/#252の受注明細/Query統合Reference | Order→OrderLineのkeyless Relation成功を、Person→Membership/歴史A2二版の意味・実Read成功へ横滑りさせない |

**現在地の注意**：#251/#252のPR履歴では `reference/workspace/databases/sales-core/relation-storage/order-lines.tei` と対応SQLが存在した。しかし最新mainの同Pathを今回直接取得した際にはNot Foundとなった。したがって過去の統合PRの記録と、**現行Checkoutで同じReference Fileが引き続き配置されているという実事実**を分離し、後者は再配置/削除のSource readback後にだけ主張する。Loader/Validator/RealizeDesiredの現実装はmainに存在することを確認済み。旧PR時点の実PostgreSQL本番接続成功まで意味を拡張しない。

### 18.2 第6のGoテスト：実組織Worldを複製し、Production ValidatorとDesired投影へ通す

[実Go Draft #4633](https://github.com/hat47x/tei-platform/pull/4633) に `TestOrgC1CandidatePersonRelationStorageValidatesAndRealizesOffline` を追加。既存の `reference/org-staged-axis` を `workspace.Load` で読込み、テスト内部だけで次を構成する：

1. Personというsource Dataに、Membershipをtargetとする**keyless Relation候補**を明示（Person→Membership）。Personの `person_id` をSourceAttributeとし、Membership Mappingの `person_support_id` support columnへ対応する。
2. `validation.ValidatePersistenceRelationStorageMappings` のProduction codeを実行し、正しい方向・ownerでは診断0であることを確認。
3. `persistence.RealizeDesired` とPostgreSQLの実Provider PolicyでオフラインDesiredを作り、support columnが通常attributeとは別に現れ、Required/Ordinalが保たれ、Membership単独PKも変更されず、FKも無断追加されないことを確認。
4. SourceAttributeを誤ってMembership自身の属性へ変え、既存Validatorが `PERSISTENCE_RELATION_STORAGE_SOURCE_ATTRIBUTE_PARENT_MISMATCH` として拒否することを確認。

このテストは**本物の組織Workspaceを入力に使った独立した候補IR/Validation/Desiredの限定Go実証**であり、採用済みSourceを変更しない。人物が属する先のRelationをPerson→Membershipと表現するのは、現実のMembership→Person取得操作・Context grain・履歴の2版共存を成立させたこととは異なる。反対向きの直接Key Relationも引き続き選択肢。

**採用の停止線**：直接Relation/RelationStorageどちらを採用するにしても、#2386/#1126の業務意味受理、実Source DSL/Queryとの完全一致、#4125の既存データ移行・バックフィル・旧Writer Fence、#4490/#2586の実Rowset/Person/FTE/Consumer Readを別に確認する。特に履歴不足の旧DBへ `person_support_id NOT NULL` を即時ADDしてよいとは限らない。旧行のPerson source coverageが無い場合はNULL/unknownを保持し、Expand→source-proven Backfill→Validateの後にのみRequired/Contract候補を検討する。原典にない過去のPerson LinkやknownFromを生成しない。

### 18.3 検証済み・実行中の区別を変更ごとに保存

- **5件版の固定HEAD 509296692...**：[Health #43549877695](https://github.com/hat47x/tei-platform/actions/runs/43549877695) completed/success。実ログで `go test -count=1 -timeout 60m ./...` PASS、`tei/internal/operations` PASS（1.737s）、gofmt/vet PASSを確認。これはSource欠落4件とIR/JSON候補1件のテストだけ。
- **6件版の新HEAD efa4a8fc0dcc3d2ecdc6971c48910c16e16ce2d9**：Production Validator/Desired追加後の新実行。Health [#43553971231](https://github.com/hat47x/tei-platform/actions/runs/43553971231) の結果で判定し、ジョブログの完全性・同一HEADを再確認するまでsuccessとしない。
- **本節が未実施のもの**：C1用Person Relation source fileの正式Authoring、DBの実Migration/Backfill、A2旧新二版保存、actual PG as-of、Read Context/Consumer、現在Grant/Effect、Web複数Principal/Workspace、Owner Acceptance。#2586通常CLI実Read、#2831 Studio Stage2 Self-buildを独立P0に維持する。


## 19. 実TEI DSLからC1候補のRelation Storageを読む最小検査とSource版資格（2026-11-14）

### 19.1 実TEI DSL／Source Index経路へ限定的に前進

[コードDraft #4633](https://github.com/hat47x/tei-platform/pull/4633) に第7のGoテスト `TestOrgC1CandidatePersonRelationStorageLoadsFromTEIDSL` を追加した。実 `reference/org-staged-axis` Workspace全体を**テストの一時ディレクトリ**へ複製し、複製内に限ってPersonの型付きRelationと `databases/org-store/relation-storage/person-membership.tei` を追加する。元のリポジトリのOrg Semantic/Persistence/Query、既存Expected、正式Source Revisionは変更しない。

候補は実際のTEI DSLとして、次の**二つの別Source**に宣言する。以下はいずれもテスト中の一時コピーであり、正式な採用済みSourceには追加しない。

一時的に編集するSemantic Sourceの該当部分：

```tei
data Person {
    attribute personId type organization.person_id
    relation membershipsCandidate to organization.membership many
}
```

新しく配置するRelation Storageの一時Source：

```tei
relation-storage org_store.membership {
    relation organization.person.memberships_candidate
    component organization.person.person_id -> person_support_id storage varchar optional
}
```

独立していた4→5→6件の実証を、以下へ進める。

| 検証の段階 | 第7テストの具体的な検査 | この段階で言えないこと |
| --- | --- | --- |
| 一時DSL Source | 実SourceのPerson宣言へ候補Relationを追記、別の `relation-storage` ファイルを配置。元Sourceは不変 | OwnerがRelationの業務意味・方向を採用したとは言えない |
| Workspace Loader | `workspace.Load` がその二つの実DSL宣言を型付きWorldに読込み、既存Persistence Mapping `org_store.membership` に明示合流 | Person検索と所属集計の実Consumerを自動生成したことにはならない |
| Source Index | `persistence-relation-storage:org_store.membership/organization.person.memberships_candidate` が作成元ファイルへ解決 | Semantic Relationと物理列のSource同一性が人物の本人確認を代替するわけではない |
| Validation | `ValidatePersistenceRelationStorageMappings` によって関係方向・source Person ID/target Membershipの整合を検証 | 未受理のIdentity merge、Case current Grantは証明できない |
| PostgreSQL offline desired | Nullableの `person_support_id` が通常Semantic列と別に投影され、PKは単独membership_idのまま、FKは自動追加されない | A2二版を同一DBへ保存できる／実SQLへ人物をjoinできるとは言えない |
| 現状GAPの負例 | 既存の実main Characterizationは、**Membershipへの他Dataからのincoming Relation**も検知し、不在assertを漫然と残さない | 今回の候補を正式Sourceへ書き戻したことにはならない |

sourceの `optional` は「過去の人物対応が不明な旧行で強制NOT NULLの失敗を起こさない」という**検査上の仮説**であり、業務OwnerがNullableを採用済みという意味ではない。Source indexの記録・DSL解釈成功・Physical Desired成功は、それぞれの正本からのRead witnessとして独立に観測する。

### 19.2 別レーンの成熟成果との整合

- **DSLの段階的受入（Draft [#4599](https://github.com/hat47x/tei-platform/pull/4599)）**：同DraftのG0〜G4に沿い、構文候補を読み込めたことと、完全World、実Query/Provider、Sourceへの安全な編集/再投影を混同しない。現行Parser/Workspaceに**既存の文法がある**ため、新しい構文案を増やさない。Person Relationの型付きMeaning採用は別のOwner決定。
- **REA/三要素の実受領（Draft [#4603](https://github.com/hat47x/tei-platform/pull/4603)、R3-A [#4625](https://github.com/hat47x/tei-platform/pull/4625)）**：候補Data/Relationを実Sourceにできることと、外部Evidenceからの候補・独立Consumer受領・業務意味の確定を別票とする。REA候補ReaderはPersonの本人照合でも自動承認でもない。
- **Studio編集と公開境界（#3421 / Draft [#4627](https://github.com/hat47x/tei-platform/pull/4627)）**：テストの一時DSLに付くSource ID/PathをStudio本番Workspaceへ転用しない。Source Revision、Workspace、選択epoch、実Host Principal/Provider、Consumer受領は別。Preview上でRelationが見えても逆編集の無条件許可を与えない。
- **歴史移行（#4125）**：Nullable support列を作れることと、旧Current-only WriterがHistory Writerに切替済みであることは別。人物Sourceのない過去行を推定補完しない。単独PKはA2旧新二版共存のまま未解決であり、ColumnADDだけでG4 Cutoverや全面再利用を受理しない。
- **ビルド・Evidence（#2514/#2600）**：Source A→B→A、Relationの追加/削除・Source/Mapping revision、列Nullable変更で検証対象が変わる。旧HEADのGo/CI成功を新HEADへ転載せず、Source Index/生成物/実Providerを別に資格化する。

### 19.3 現状と次の最小実証

- **固定HEAD efa4a8fc**（6件版）：[Repository Health #43553971231](https://github.com/hat47x/tei-platform/actions/runs/43553971231) completed/success。実ログで `go test -count=1 -timeout 60m ./...`、`tei/internal/operations`、gofmt、vetのPASSを確認。**本物のOrg Sourceを元にしたGo候補IR/Validator/Desired検査**としてのみ認定。
- **固定HEAD b124a5fe**（7件版）：実DSL複製→Loader/Source Index→World Validator→PostgreSQL offline Desiredを検査した[Repository Health #43556769631](https://github.com/hat47x/tei-platform/actions/runs/43556769631)が**completed/success**。これは同HEADの限定技術検証に限り、実業務Meaning・Query/PG Person Readの資格に昇格しない。
- **次の独立P1**：#2386/#1126がProfile/Personリンクと向きを選んだ場合に限り、正式SourceでのRelation、Schema差分/実Provider Person Read、r7のA2二版/knownAtの独立Expectedを、別の最小差分で接続する。Case C1/Current Grant/Effect/Host ACKは後段。通常CLIの最初の実Provider Read、Studio Desktop Stage2セルフビルドを**独立P0**のまま維持する。

**実施範囲**：この節とコードPRはAuthoring候補・Parser/Loader・Source Index・Validation・offline Desiredを検査するだけであり、DB変更、実Query結果、過去人物の推論、Owner意味採用、外部Host実行は未実施。

### 19.4 実DSL負例とQuery／独立Consumerへの停止線（2026-11-14追加）

第7テストはPositiveのDSL読込で成功していたが、「Relation Storage DSLから実Query結果を計算できること」や「誤ったRelation sourceが失敗に分類されること」はまだ証明していなかった。後者だけを既存Validator/Loaderに接続し、**新Runner・新Expectedファイルを作らず**、一時Workspaceに対する5つの失敗注入を同一テストのサブケースとして追加した（[Draft #4633](https://github.com/hat47x/tei-platform/pull/4633)）。

| 一時Sourceに注入する反例 | 通常のTEI処理に期待する拒否・観測点 |
| --- | --- |
| Person→Membership Relationで、SourceAttributeをMembershipの属性へ誤変更 | World Validatorの `PERSISTENCE_RELATION_STORAGE_SOURCE_ATTRIBUTE_PARENT_MISMATCH` |
| Relation Storageから、targetがPersonの既存Employment.person Relationを参照 | World Validatorの `PERSISTENCE_RELATION_STORAGE_TARGET_MISMATCH` |
| 存在しないSemantic Relation IDへ差替え | World Validatorの `UNKNOWN_PERSISTENCE_RELATION_STORAGE_RELATION` |
| Relation Storage合流先を存在しないorg_store Mappingに差替え | Source Loaderが**未解決Mappingを拒否**（既存IDから推測しない） |
| 同じMapping/Relation Identityを二つのAuthoringファイルに宣言 | Source Loaderが**二重定義を拒否**（後勝ちで採用しない） |

成功側は従来通り、Person→Membershipのkeyless Source、Nullable support column、Source Index、offline Desiredを確認する。**固定HEAD `2c155383d525b022a4ba0f317192351c4eb0b92a` の[Repository Health #43558325121](https://github.com/hat47x/tei-platform/actions/runs/43558325121)はcompleted/success**。実ジョブログで全Goテスト、`tei/internal/operations`、gofmt、vetのPASSを確認した。旧HEADの合否と混ぜず、本HEADだけの限定技術証拠とする。ここで数えるのはトップレベルGoテスト7件、追加の負例サブケース5件であり、12個の独立Product E2Eではない。

**他チャットの実受入との接続**：[#4619](https://github.com/hat47x/tei-platform/pull/4619)のProduct D1/QJPAは、実Sourceを選ぶ→型付きBinding→選択Providerと実Call→Rowset/Context→独立Recipientの受領を別Claimにする。[#4638](https://github.com/hat47x/tei-platform/pull/4638)の実PostgreSQL16参照Queryは固定HEAD `0ce3a52f` で7サブテスト成功したが、入口は `tools/reference-browse-read` であり、C1の `org_axes_view.as_of_membership`、Product `tei` CLI、人物3人／FTE集計、二Consumerは証明しない。[#4637](https://github.com/hat47x/tei-platform/pull/4637)の限定Host Readerは**別SourceのREA候補を安全に受けるローカルHost境界**であり、C1人物Identityや共有WebのCurrent Principal/Grantを保証しない。

このため、組織C1の次の**Positive Product Test**では別証拠として、(1)正式受理されたPerson→Membership又はMembership→PersonのRelation SourceとCase Profile、(2)同一A2の旧新履歴版が物理DBで共存できるPK/Version ID、(3)actual queryでのknownAt/validAtとPerson IDのread witness、(4)独立Expectedと同一Read Cohortの人数/FTE、(5)Function/CLI/二Consumerの実受領を順に閉じる。Source Spanが一致しても物理列への写像・Queryの実消費・Recipient受領を推測せず、別Principal／外部Host公開は別Ownerの現在権限・Disclosure検査に委譲する。

第7テストの一時WorkspaceはOwner採用されたProduct Sourceの代替ではない。現在の単独membership_id PK、欠けたas-of Person射影、履歴Backfillの不確定性は引き続き未解決。Studio Desktop Stage2セルフビルドと通常CLIの実Provider Readは独立P0を維持する。



## 20. 本物のRelation Storage Sourceから安全なExpand候補を出す試験と取得被覆の停止線（2026-11-14）

### 20.1 G1 Expandは「物理形状の差分」と「既存DBへ適用できる資格」を区別

[Go Draft #4633](https://github.com/hat47x/tei-platform/pull/4633)に、既存第7テストで実TEI DSLをロードした後の**オフラインSchemaDiff→PostgreSQL DDLレンダリング**を追加した。テスト内で、同じ型付きOrg Persistence Mappingから一時的なRelation Storageを除いたSource由来Baselineと、追加後のDesired Physical Schemaを、既存`schemadiff.Compare`で比較する。手書きの`physical.SchemaSnapshot`を現在の実DBと偽って扱わない。

期待する差分は**1件だけ**であり、既存`membership_id`単独PK、FK、通常のSemantic属性を変えない。

```sql
ALTER TABLE "public"."org_memberships" ADD COLUMN "person_support_id" pg_catalog.varchar;
```

このSQLは**Nullableな補助列の追加候補**であり、既存Migration Rendererによる文字列出力を検査するだけで、実DatabaseへのDDL実行、旧行のPerson ID補完、列同定のAuthority、履歴版PK改訂は行わない。必須列`NOT NULL`への昇格やHistory Writer切替は、既存[#4125](https://github.com/hat47x/tei-platform/issues/4125)のExpand/Source-proven Backfill/Validate/Writer Fence/Contractの段階別Owner判断を要する。旧行の欠測を推論で埋めない。

**区別する3つの版・資格**：①現mainのorg-staged-axisは正式Person Linkなし・PK単独で、②テスト内の一時SourceはPerson→Membership keyless Relation/optional supportを持つが正式未受理で、③実DatabaseのIntrospection／移行履歴は未取得。①→②から生成した差分は③の現状を証明せず、最新mainに載っていないDraft #4566/#4567のSQL修正も暗黙に取り込まない。

### 20.2 他チャットのFunction Data Context「partial」と、組織人数・FTEの閉包

[Draft #4647](https://github.com/hat47x/tei-platform/pull/4647) HEAD `a962a3f6`は、実PostgreSQL観測W1〜W3でRLS、LIMIT、宣言外JOIN、View/Partitionなど**見えた制限**を`coverage=partial`で運び、Native SQL本文外の述語（SQL関数等）があるため、制限未観測でも`unestablished`と`native-sql-predicate-not-closed`を維持する。固定HEADのWorkflow 7/7 completed/successは確認できたが、同PRはDraft／main未統合、製品`tei`の通常CLI/Current Grantとは別資格。W4列の由来は未実装であり、`NULL`を`stored`と断定したり粒度`entity`を主張したりしない。

| C1のRead/Measure用途 | 制限を観測した`partial` | 制限不明の`unestablished` | 独立Expectedに対する正式な全体Claim |
| --- | --- | --- | --- |
| Membership件数 | 見えた行だけの局所件数として範囲・理由とともに提示 | 見えた行だけの観測値、完全性不明 | 全体件数の合格不可 |
| distinct Person | Person Link・source identity・兼務A2/A3の同一人物性が未成立なら算定不可 | 同上。0行を「0人」や非所属の証拠にしない | 同一validAt/knownAt・Provider/Reader Scope・完全性を確認後のみ |
| allocation FTE | 見えた行の合計は**表示範囲のSubtotal**、全組織の3.0/2.4と比較不可 | 集計母集団が未確定であれば未資格 | r7の合成Expectedと実Product Rowsetが**同じCohort**で閉じた後のみ |
| Case承認・移管 | `partial`を候補0件や不可判定へ縮約しない | Authority/Rule/Grant・SourceRevisionを別照会 | 現在Grant/Host Effect/Receipt/Consumer ACKを別票で検証 |

重要なのは、`partial`が**何行欠けたかを示す数値ではない**こと、`unestablished`が`complete`の反対証明でもなく**未確立**であること。既存r7の「3 Person、4→3 Membership、3.0→2.4 FTE」はあくまで合成Oracleであり、現mainの実QueryにはPerson射影も履歴版PKも未実装。Native SQL経路で見えている行がOracleと偶然一致しても、Product全体の完全性を立証しない。

### 20.3 P0の入口を変更せず、次の製品検査を絞る

横断の[RA-0/1/2仮統合 #4644](https://github.com/hat47x/tei-platform/pull/4644) HEAD `b5c23044`は3/3の局所CIがsuccess。ただし参照Executorと通常`tei`の製品Provider Readは異なる。元RA-0側のE1/API依存境界の未達は新PRの成功で免責されない。[生成Read-only Function #4559](https://github.com/hat47x/tei-platform/pull/4559)は`recent_orders`専用の実生成・実PG局所成果であり、C1のPerson Relationを含むQueryや二Consumerへ自動流用しない。

最短の次作業は、Ownerが採用したM-P/M-E/M-Xの**一Profile**について、正式なRelation（source/target方向・Key）→実保存されたPerson対応→実QueryのPerson射影→同じRead Cohort・Periodでの人数/FTE→独立Context/Consumer受領、という単一の証拠系列を通すこと。履歴版`record_version_id`等の物理PK、旧Writer移行、Source/Current Grantはそれぞれ別Owner。配布生成物のToolchain観測[Draft #4648](https://github.com/hat47x/tei-platform/pull/4648)はTarget成果物のbyte差・Builder申告の観測であり、C1の業務MeaningやSource→実DBの完全性には昇格しない。

**実行資格**：7テスト＋5拒否サブケースの旧HEADは[Health #43558325121](https://github.com/hat47x/tei-platform/actions/runs/43558325121)で成功。追加したDDL差分テストの最初のHEAD `65b8348780dc92ad19bcc1e5a8ca9a0311f2e216`には**コンパイルエラー**があり、[Studio Source Loop #43572108566](https://github.com/hat47x/tei-platform/actions/runs/43572108566)が失敗、[Health #43572108581](https://github.com/hat47x/tei-platform/actions/runs/43572108581)は後続修正によりcancelled。修正HEAD `85f657002f8b9f05dfd119a72565e583985b8494`の[Health #43572492328](https://github.com/hat47x/tei-platform/actions/runs/43572492328)を**新たな独立Run**として検証するまではPASS未確定。本節は新DDLの実接続／PG実行／Owner採用を報告するものではない。通常CLI実Provider Read #2586 とStudio Stage2セルフビルド #2831 は独立P0のまま維持する。


### 20.4 DDL試験の失敗を固定HEADで再資格化する

- **原因確認**：最初のDDL候補HEAD `65b83487` の[Studio Source Loop Run #43572108566](https://github.com/hat47x/tei-platform/actions/runs/43572108566)で、追加Goテスト内の文字列`before`（一時DSL Sourceの置換前テキスト）と、同名にしたPhysical Baseline変数が衝突し、`no new variables on left side of :=`／文字列を`physical.SchemaSnapshot`へ代入できないGoコンパイル失敗を実ログで確認。この失敗は新Testのコードに由来し、Studio本体の検証不合格とは断定しない。
- **修正**：[PR #4633](https://github.com/hat47x/tei-platform/pull/4633) の新HEAD `85f657002f8b9f05dfd119a72565e583985b8494` ではPhysical Baseline変数だけを`physicalBefore`へ変更し、既存のSemantic DSLの`before`文字列は保持。`schemadiff.Compare(desired, physicalBefore)`に固定。新しい機構やOwner意味変更なし。
- **CIの資格（再確認済み）**：旧HEAD 65b83487のGoコンパイル失敗・Health #43572108581のキャンセルは履歴として保持。修正後HEAD 85f657002f8b9f05dfd119a72565e583985b8494の[Health #43572492328](https://github.com/hat47x/tei-platform/actions/runs/43572492328)はcompleted/success。Job 132379737524で全Go、operations 3.073s、gofmt、vetのPASSを確認したため、DSL→offline SchemaDiff/DDLのコードを当該HEADの限定技術証拠として再資格化する。一方同HEADの[Windows smoke #43572492248](https://github.com/hat47x/tei-platform/actions/runs/43572492248)はPowerShellによるOSVersion取得が15秒timeoutとなりfailed。Go本体の失敗ではないが全Workflow合格でもない。別[Desktop PR #4587](https://github.com/hat47x/tei-platform/pull/4587)の別HEADのWindows成功をこのHEADに転用しない。


## 21. 人物数とFTEを、実SourceからConsumerまで一続きで検証する（2026-11-14）

### 21.1 人数が同じでも、訂正前と訂正後は同じ結果ではない

既存の org-synthetic-r7 は、同じ有効日（2027-06-15）でも、記録時点によって所属の集計が変わることを示している。

| 記録時点（knownAt） | D1の所属／FTE | D2の所属／FTE | 全体の所属／人物／FTE |
| --- | --- | --- | --- |
| 2027-07-01 | A1、A2／1.6 | A3、A4／1.4 | **4件／3人／3.0** |
| 2027-07-03 | A1／1.0 | A3、A4／1.4 | **3件／3人／2.4** |

A2は同じ人物P2の所属であり、旧記録版は7月2日に置き換えられ、新版では有効期間が6月1日で終了する。P2にはD2側のA3もある。したがって、人物数がどちらも3人であることは、時点の選択、所属版の共存保存、取得範囲の完全性を証明しない。FTEはr7の合成配賦値から計算するもので、製品の数値・単位設計を確定するものではない。

**独立Expectedは値だけでなく、所属ID集合、人物への関係、部署別FTE、有効時点／記録時点、取得範囲を組にする。** 新しいExpectedの正本は作らず、既存r7を用いる。RLSやLIMITで取得が制約された場合、見えた行の合計は表示範囲の小計にすぎず、全体の人数やFTEを確定させない。

### 21.2 既存の横断受入を、C1の一つのReadへ適用する

mainへ統合済みの[Consumer横断設計 #4653](https://github.com/hat47x/tei-platform/pull/4653)が示すDefinition→Binding→Authority→Execution→Acquisition→Delivery→Acceptanceを、組織C1のread-only照会へ適用する。新しい組織専用PipelineやRunnerを設けない。

| 引渡し | 必要な具体的証拠 | まだ代用できないもの |
| --- | --- | --- |
| **業務意味→保存**（#2386/#1126/#4125） | Ownerが採用した一つのM-P／M-E／M-X Profile、Person関係の向き・Key・出典、Membershipの記録版PK | 一時DSLのRelation Storageやnullable列ADDだけ |
| **保存→実Query**（#4490/#4567） | 同一DB内のA2旧新二版、選ばれた実Mapping/Provider/SQL、knownAt/validAtとPerson IDのRowset | DDLレンダリング成功、別DBの参照SQL成功 |
| **実Query→Context**（#4647/#2586） | 取得した実Rowsetの粒度・NULL・欠測、RLS/Limit/Scope・Read Cohort、被覆の根拠 | 制限が不明なunestablished、見えた範囲だけのpartial |
| **Context→受領者**（#4653/#4501） | 用途・Audience・Source Revisionに対応した実Consumerの受領と、必要なら独立Invocation | Provider実行PASS、CLI標準出力、Studio Previewのみ |
| **受領→実業務行為**（#1174/#3421） | Caseの現在責任、Host Grant、Effect、Commit Receipt、ACKを別々に確認 | Org RoleやRead結果から実行権限を推測すること |

**最初のProduct Exitはread-onlyでよい。** ContextとConsumerまでの実Readが成立したことを、承認・辞令などHostの書込行為と混同しない。#2586の通常tei CLI実Readと#2831のStudio Desktop Stage2セルフビルドは独立P0のまま維持する。

### 21.3 段階ごとに提供物が異なっても、意味の根拠は失わない

[Plugin Reference Draft #4646](https://github.com/hat47x/tei-platform/pull/4646)は、共通の意味ID／版／Fingerprintと、Studio・Generator・Runtime・Testに固有なContract、Target、Scope、Provider、Artifactを分離する試作である。固定HEAD 77463edbの技術CIは3件すべてsuccessだが、製品の登録・認可・実行を完成させたものではない。

C1でも、人物と所属の意味定義が一致しているだけで、別DB AliasのQuery、異なるRuntime版、別Consumerへ届いたResultまで等価だとは扱わない。逆に、read-only Previewに書込Runtimeの提供物を必須とすることもしない。#2611のAlias競合、#2600のSource差分による再資格化を、既存のOwnerのもとで再利用する。

[Desktop設計 Draft #4650](https://github.com/hat47x/tei-platform/pull/4650)が区別する「Studioを生成して動かす一巡」と「業務Sourceを開いて使う一巡」も分けて扱う。C1の一時DSLがLoaderへ通ることは、Studioの選択対象Pin・安全な逆編集・生成版Stage2の実稼働を証明しない。

### 21.4 次に実装するのは、一つの意味Profileの正の縦断試験

まず#2386/#1126がr7の個人所属に対応する一Profileを採否し、人物の同定根拠、兼務、欠測、関係の向きとKeyを定める。そのうえで#4490/#4567がA2旧新の物理版共存と実Queryを確かめ、#2586が実ProviderからContextと独立Consumerへ結果を渡す。この順序は製品のP0への追加条件ではなく、**C1固有の受入順序**である。

検出すべき反例は、記録時点の取り違え、同じ人物の二所属、A2の旧版消失、RLSによる取得不足、異なるSource Revisionを受け取ったConsumerである。いずれも既存Scenario/独立Expectedへ帰属させ、第二の履歴Engine、Resolver、認可基盤を作らない。

**到達点**：PR #4633のDSL／Validator／offline DDLは固定HEADで技術検証済み。人物同定・二時点履歴の実DB保存・complete Rowset・通常CLIと独立Consumerの受領・Owner意味採用は未実証である。

### 21.5 所属の訂正と、ケースの引継ぎを起動することは別である

[main統合済みの起動・イベント・処理・応答設計 #4666](https://github.com/hat47x/tei-platform/pull/4666)を、組織C1の責任移管へ対応付ける。共通化するのは待受けループではなく、**どの契機でどの契約を起動し、何を読み、何を変え、結果をどう確かめたか**という意味である。組織専用DispatcherやStateMachineは設けない。

人物P2の所属A2が遡及訂正されても、それは所属記録の変更であり、ケースC1の責任移管が完了したという事実ではない。移管には、訂正Sourceの版と既知時点、移管RuleとCase Revision、候補集合の完全性、現在のHost Grant、実際のAction、Effect Receipt、必要な相手のACKを別々に確認する。候補を表示できたことと、移管Operationの実行を許可したことも区別する。

既存のORG-BEH・CX・Effectの反例には、同じ人物数を返す訂正前後、訂正イベントの重複配送、候補表示後のCaseやGrantの改版、Commit結果不明での再試行、通知は送ったがACKがない場合を割り当てる。いずれも自動移管を認めず、結果不明時はOperation IDとReceiptを照会する。有効時点（validAt）、記録時点（knownAt）、Ruleの評価時点、Effectの試行・確定時点を一つの時刻とみなさない。これにより、**過去の所属を説明する照会**と、**現在移管してよいかの判断**を、同じ業務モデルから別々の操作として扱える。


### 21.6 同じ所属を、図・表・時系列へ映しても識別を失わない

[main統合済みの汎用実行モデル #4668](https://github.com/hat47x/tei-platform/pull/4668)が区別する**意味の定義・実効計画・実行記録・表示用の図面**を、組織C1にもそのまま適用する。GraphやTableに描かれた関係は、実際のHost処理の記録や現在の権限ではない。標準データ構造の意味的な対応付けを扱う[Draft #4667](https://github.com/hat47x/tei-platform/pull/4667)は、以下の観測条件を検討する際の参考とし、未採用の新Grammar／Semantic EngineをC1へ先行導入しない。

| 見せ方 | r7で保持すべき同一性 | 注意する操作 |
| --- | --- | --- |
| 部門別ツリー | P2はD1のA2とD2のA3に現れるが、同じ人物であり、所属は別の発生（Occurrence） | ツリー上に二度現れたP2を別人として数えない。片方の見た目から人物全体を無条件に変更しない |
| 関係グラフ | Person、部門、Membershipと、A2旧版／新版の関係・向き・多重性・時点を識別する | 同じ両端の線を一つにまとめたり、図の線IDを所属ID／記録版IDに置き換えたりしない |
| 所属の一覧表 | A1〜A4という所属行と、人物別集計の粒度を区別する | JOINでP2を重複計上しない。3人という集計行から4件の所属を復元して編集しない |
| 二時点の表示 | validAtとknownAtを別の指定として保持し、A2の訂正前後を選べる | 過去記録の表示を現在の承認権限やCase移管と同一視しない |

**最小の編集受入**は、表示対象のIdentity、実Relation/Occurrence、Source位置・Revision、選択時点、変更した項目が、一意に元Sourceへ対応することである。集計表や多重関係を省略したツリーなど、その対応が定まらないViewは、まずread-onlyとする。Studioの選択を保持したままSourceが改訂された場合は、旧表示からの自動適用ではなく、対象再解決と競合検査を求める。

これらは既存r7による**合成モデルの投影・編集安全性の受入条件**であり、現行の製品SourceにMembership→Personの正式Relationが存在することや、Studioで逆編集できることの証拠ではない。実Query／Providerと独立Consumerの検収は§21.2、業務上の移管は§21.5へ戻す。


## 22. C1最小縦断：業務意味から実保存・実集計・複数表示までを一つのSource系列で通す（2026-11-14）

### 22.1 受入の単位は「組織図が描けた」ではなく、元の意味を失わない一巡

最初に閉じる対象は、既存 \`org-synthetic-r7\` の Person P1〜P3、Membership A1〜A4、所属先 D1/D2、A2 の遡及訂正という**一つの小さな業務世界**に限定する。別々のfixtureを組み合わせて成功と数えない。次の六つを同一Source Revisionと、時点・Providerを明示したAttemptで追跡する。

1. **Meaning**：Person・OrgUnit・Membershipの独立Identity、A2/A3が同じP2に対応する根拠、所属の関係方向・Key・配賦単位、訂正の原因と業務上の期間をSourceから読める。
2. **Storage**：同一A2 Occurrenceに対する訂正前・訂正後のRecord Versionが、同一の隔離PostgreSQLに物理的に共存する。Occurrence ID、Version ID、Physical PKを混同しない。
3. **Read**：同じvalidAt=2027-06-15に対しknownAt=2027-07-01/2027-07-03の二照会を、同じProvider契約の実SQL/paramsで実行し、選択したVersion、Person ID、AllocationとRead Cohortを記録する。
4. **Meaning-preserving result**：独立r7 Expectedの Membership 4→3、distinct Person 3→3、FTE 3.0→2.4、部署別の所属集合と小計に一致する。数値一致に加え、同一人物P2のA2/A3が別Occurrenceであることを示す。
5. **Consumer**：少なくともQuery→Function Context→一つの独立Consumerまでを実受領で確認する。別の表示Consumerを追加する場合、同一Read Cohortか明示的に別Cohortかを記録し、CLI標準出力だけを「二Consumer成功」と数えない。
6. **Projection**：組織ツリー・グラフ・所属表・二時点表を同じ意味モデルから導出し、P2の多重表示、A2/A3の別Identity、A2の旧新版、hidden edge、Source span/revisionを保持する。非単射な集計行と省略ツリーはread-onlyにする。

**一連の実測が揃うまで、DSLがGo処理へdelegateしているだけの実装を「組織モデルで業務が実現した」とは評価しない。** Goで業務上のPerson対応・有効期間・配賦ロジックを別に手書きして期待値に合わせる方式は、Product化ではなく不足機構を切り分ける試験用Adapterとして明示する。

### 22.2 保存方式の比較契約と更新の境界

以下の列は**受入に必要な意味情報の例**であり、確定したTEI DSL構文、必須物理テーブル、既存DBの実Schemaではない。

| 種類 | 受入で保持する情報 | 主な負例 |
| --- | --- | --- |
| Membership Occurrence | 安定したA2、Person P2、OrgUnit D1との関係 | Person、Membership、表示Edgeを同一IDと誤認 |
| Record Version | 旧版・新版の別ID、known区間、valid区間、訂正原因 | A2を単独PKにして旧版を上書き、同一時点に矛盾二版を採用 |
| Physical Row | Provider、表・列、実PK、書込結果、read witness | offline DDLと実DBを同一と推定 |
| Source/Mapping | Relationの方向、Source span/revision、version/Personの列対応 | nullable列だけ追加し人物同定済みと扱う |
| Acquisition | SQL/params、snapshot/cohort、coverage、scope、NULL・欠測 | RLS/LIMITのSubtotalを全体値に昇格 |

比較候補は§2.1の三方式を維持する。まずDB Ownerが**一方式を試験用に選ぶ**のであって、組織専用の共通履歴Engineや全Dataへの二時点必須化は先行させない。

今回の試験は物理行に二版があるだけでは合格としない。A2旧版を選ぶknownAtではMembership集計にA2を含み、新版を選ぶknownAtでは同じvalidAtにA2を含めないことを実Queryで確かめる。同一(validAt, knownAt)に矛盾する二版が出る場合、暗黙のORDER BY/LIMIT 1で片方を選ばず、制約違反として停止する。別Occurrenceの兼務A2/A3は矛盾として扱わない。

過去の記録を訂正した事実から既存Caseの責任移管を推論しない。Personの転属、CaseのHandling Unit、承認候補、現在Grant、Host Effect、通知、受信、ACKは§21.5の別契約とする。

### 22.3 独立Expectedで検査する正例・負例

| 試験 | 観測するExpected／停止条件 |
| --- | --- |
| P1 二時点の正例 | validAt=2027-06-15、knownAt=2027-07-01でMembership A1/A2/A3/A4・人物3・FTE3.0。実RowsetにPerson IDとRecord Version証拠がある |
| P2 訂正後の正例 | 同じvalidAt、knownAt=2027-07-03でMembership A1/A3/A4・人物3・FTE2.4。A2旧記録が物理的には残っている |
| N1 Identity | P2のA2/A3を別人物として数える、あるいはA2新旧を別所属として同時計上したら失敗 |
| N2 物理版 | 旧版が上書き、二版が別DBにしかない、version/Person IDがResultから追えない場合は未資格 |
| N3 Coverage | RLS/LIMIT/未解析条件でpartialまたはunestablishedなら、たとえ数値がr7と一致しても全体人数・FTEは未資格 |
| N4 Consumer | QueryやCLIログは成功しても、独立Consumerが未受領、別Revision/別Cohortを受領、または未認可のRecipientへ渡したならProduct Exitを主張しない |
| N5 表示・編集 | ツリーでP2の兼務を落とす、グラフで複数辺を併合、表の集計セルを一意なSourceとして逆編集、Source A→B→A後に旧PinでApplyしたら停止 |
| N6 Effect | 完全な候補集合でもHost現在GrantがなければAction不可。Commit不明で旧Receiptを流用した再試行、通知成功をACKとみなす場合は失敗 |

各試験の独立Expectedの出典は既存#2510/#4490/#4501を用いる。新しいOracle/Scenario Storeは作らない。Product ActualはGit HEAD、Source/Mapping Rev、DB/Provider、実SQL/params、Cohort、Consumer Receipt、必要ならHost Receiptを**同一Attemptの実測として**残す。別HEAD・別DB・別WorkflowのPASSを一件の成功へ合算しない。

### 22.4 図表・DSL・コード生成と、Pluginのセルフビルド可能域

同じPerson/Membership/OrgUnit Relationから、①as-of照会の依存関係、②関係グラフ、③所属表・組織ツリー、④Query/Reader/表示Adapterの必要なコード生成を段階的に導出する。各段階で「TEI Sourceに書かれている業務意味」「Generator/Pluginが共通化した処理」「Target固有の手書き処理」を分類する。

生成の有効性は出力量でなく、**業務意味の二重記述が減ること**と、変更が既存のSource/Rule/Query/Projectionを通じて正しく伝播することで検証する。受入メトリクスは次の通り。

- **業務意味重複数**：Person対応、所属の時点条件、FTE計算、兼務の重複除去について、Source以外に同じ意味の手書き判定が何か所あるか。Goへの単純な委譲箇所を「TEI実現率」に含めない。
- **モデル差分の伝播率**：A2訂正やP2兼務の変化を一度Sourceへ反映したとき、DSL検査、実Query、Expected、表示と生成物へ正しく波及するか。どの段階で手動修正が必要かを列挙する。
- **構造保持率**：Source側のPerson/Occurrence/Version/RelationのID、多重性、時点、方向、根拠が各投影へ受け継がれた割合。重要属性が失われた表示にはloss reportを付す。
- **Plugin再利用率**：同じ型付きGraph/Temporal/Query/Projectionの契約を、組織以外のデータにも使えるか。組織専用の二つ目のResolver/Engineを作らない。
- **自己記述・自己生成の範囲**：Generator/Plugin自身の仕様・テスト・生成可能部分を \`*.tei.md\` で表現し、TEIが再生成できる部分と手書きのBootstrap/Target境界を分けて数える。割合だけでなく、段階ごとに手書き部分がなぜ残るかを記録する。

図・表・Markdown・Queryは標準意味モデルの複数表現として扱う。新しいDSL文法や図法の採用が必要な箇所は別Ownerへ差し戻し、C1一巡のために独自の固定組織Kindを設けない。Plugin方式もGo単一バイナリと外部Host生成の既定を崩さず、静的組込み/生成コード/外部プロセスの方式選定を各境界の要件に委ねる。

### 22.5 着手順・受入票・P0との関係

- **C1-0 Read-only契約固定**：#2386がM-P/M-E/M-Xの一Profile、人物関係の向き・Key・Sourceを採否。DB Ownerが既存のPK/Mapping/Readerの実状を再確認する。
- **C1-1 物理版共存**：隔離PGにA2新旧を同時保存。Source由来のPerson Link・Version ID・PK、独立Expectedで旧新の選択を試す。DDL候補だけでは停止。
- **C1-2 Query→Consumer**：通常\`tei\` CLIの実Provider read入口を利用し、同じSource/DB/Attemptで人数・FTEとContextを導出、独立Consumer受領まで確認する。
- **C1-3 Loss-aware投影**：Read-onlyのTree/Graph/Table/Temporalを同一Sourceから出す。初めはSource位置へのNavigationと差分Previewのみとし、逆編集は一意同定・CASが成立する項目に限定。
- **C1-4 生成・自己記述の削減試験**：上記のための手書きGo/SQL/表示Adapterと生成された箇所を比較し、TEIに移せた業務意味と依存が縮小した箇所を記録。Pluginの自己生成は独立の証拠で扱う。

この順番は**組織C1固有の受入計画**であり、#2586 通常CLI初回実Provider Readと #2831 Studio Stage1→Stage2セルフビルドへの新しいHard Gateではない。Ownerが業務意味を採用していない場合は試験Sourceを「候補」と表示し、Technical PASSをOwner Acceptanceへ昇格しない。§21で整理した既存のAction/Grant/Evidence契約をそのまま利用し、新しいRuntime、専用Host、第二Policy/Expectedは作らない。


## 23. 責任の解決と実行資格をつなぐ最小契約（2026-11-14）

### 23.1 何を一つのモデルとして扱い、何を別資格にするか

組織上の所属・役職・役割・委任を、案件の責任や現在の実行許可へ直結させない。組織の型付きRelationは、**業務上の候補を導く根拠**として利用する。責任の所在はCase/業務Scopeに関する別のData/Relationで表し、実Actionは既存Hostの現在AuthorityとEffect Fenceへ委譲する。

| 層 | 意味上の主体・対象 | 導出するもの | ここから導出しないもの |
| --- | --- | --- | --- |
| 組織上の事実 | Person、OrgUnit、Membership、Position、Occupancy、Employment等 | 誰がどこにどう関わるか、時点付きRelation Path | Case担当、Person=Principal、外部Tenant Grant |
| 業務上の責任 | Case、Duty、責任保持先（Person/Position/OrgUnit/業務Role等）、期間・Scope | 何について誰・どの組織が責任を負うか | 当該人物の現在の操作許可 |
| 判断候補 | 受理済Rule、Case revision、時点・Read Scope、上記Relation | 候補集合、候補ごとの由来、完全性と未解決理由 | Candidate=選任者、Candidate=Grant |
| 実行資格 | 実操作者のverified principal、対象Capability/Action/Resource、現行Grant/Policy/対象版 | Hostが現時点で当該実行を許せるか | Commitが完了したという事実 |
| 業務効果・引継ぎ | Operation ID、Effect/Receipt、受領/ACK、Case新revision | 実行が何を変え、相手が何を受けたか | 過去の所属訂正だけによる移管完了 |

ここでの「責任保持先」は可変的な業務対象への参照であり、Person/Account/Tenant/Principalを統一する新Coreスーパー型ではない。RACI、単独責任・共同責任・合議などは用途別Projection/Rule候補であって、全案件への固定enum/必須カラムではない。

**Sourceの境界**：\`org-synthetic-r7\` は人物・所属・二時点FTEの独立Expectedである。Case C1のhandlingUnit/Responsibility/Handoffに関する合成Sourceとは、暗黙に同じ保存Worldとはみなさない。両者を一つのProduct Attemptで組み合わせる場合、採用されたCase→OrgUnit関係・出典・モデル版・DB Cohortを明示してから照合する。人物P2の兼務を根拠なくC1の責任者と認定しない。

### 23.2 候補解決は型付き関係の読み取りであり、権限の推定ではない

以下は**宣言済み業務意味を既存Data/Relation/Query/Ruleへ束縛するための契約例**であり、新しいTEI DSL構文や製品APIの採用案ではない。

1. 対象Case、要求Duty/Action、責任決定の基準時点、Rule版を選ぶ。申請時に固定するか承認実行時に再解決するかは業務Ownerが決める。
2. Case→handlingUnitまたは明示Responsibility→Holderを、Source revisionと業務上の有効期間付きで読む。既存の組織Relation（Position→Occupancy、OrgUnit→Membership等）を**Ruleが指定した方向と軸だけ**たどる。階層親子・兼務・法人連結を無条件に推移させない。
3. Relation occurrenceのIdentity、参照した版、Query/RuleのRequired Read Set、Provider/Reader/Scope、validAt/knownAt、observedAt/cohortを保持し、結果をdistinct Person候補へ投影する場合も採用したdedupe keyを示す。
4. 0件・1件・複数件に加え、実Readの完全性を確認する。NOT EXISTS、除外条件、代理の取消し判定など**不存在を根拠とする分岐**は、対象範囲の取得完了の根拠なしに真として扱わない。
5. 候補一覧の提示、担当者の選択、Host Grantによる実行許可、Effectの確定を**別の操作**として扱う。未選択の候補や古いUI Pinを実実行対象へ流用しない。

UI/検収の分類案：

| 観測 | 表示・次の判断 | 実Actionへ短絡しない理由 |
| --- | --- | --- |
| complete-none | 有効な候補なし。業務Ownerが定めた例外経路/未割当てへ | 「候補なし」から管理者権限を暗黙発行しない |
| complete-one | 候補1件と根拠Pathを提示 | 候補が唯一でもHost Current Grantが別に必要 |
| complete-many | 候補全件とRule/責任の重なりを提示 | 合議・共同責任・優先順位などの採否はRuleに依存 |
| incomplete-or-unknown | 見えた候補と不足範囲・原因を示し決定保留 | 0件・1件を全体の不存在・唯一に昇格できない |

上表は表示・受入用の分類であり、第二の標準Result型や認可Engineを要求しない。候補を取得できたという事実が機密Fieldの開示を許すわけでもない。ProviderとRecipientの開示範囲を既存のDisclosure契約で区別する。

### 23.3 委任と代理を、責任を変えずに扱う

代理行為は「本人へ成り代わる認証」ではなく、**実操作者・被代理者・委任された業務範囲・現在のGrantを分けて記録する操作**を基本とする。

- 代理閲覧、代理Action、緊急昇格は別の権限条件を要する。緊急昇格の許容条件は業務Owner/Host Ownerへ差し戻す。
- Delegationでは委任者・受任者・権限の出所・対象Action/Resource/Tenant・有効期限・取消版・必要承認を照合する。再委任は明示された許可がない限り不可とし、連鎖・循環・利益相反を検証する。
- 委任関係が有効であることと、当該Principalに現在のHost Grantがあることは別。対象すり替え、Tenant違い、自己承認、Grant失効は実行境界で拒否する。
- 代理者が承認しても、実行Receiptには**実操作者の同一性と誰のためのActionか**を区別して残す。委任の記録を過去の確定Receiptの書換え理由にしない。

### 23.4 Caseの責任移管は、異動・候補変化・通知とは独立したChangeである

P2のMembership A2が遡及訂正されたことや、C1の候補者が変わったことから、C1のhandlingUnitやResponsibilityが変更されたとは推定しない。移管を開始するには、既存Trigger/Procedureから**明示されたCase変更の業務要件**、期待Case revision、対象Duty/Scope、移管Rule、必要な受領条件を確定させる。

Case責任の切替は単独責任の例で、旧責任と新責任の有効性を次の4象限として**観測**する。これは固定StateMachineの採用ではない。

| 旧責任 | 新責任 | 単独責任の用途での解釈 |
| --- | --- | --- |
| 有効 | 無効 | 旧担当が責任を持つ状態。引継ぎ準備中でも新担当の実責任は未確定 |
| 有効 | 有効 | 二重責任の衝突候補。共同責任を許すRuleがあれば別解釈 |
| 無効 | 有効 | 新担当が責任を持つ状態。ただし必要な受領ACKや外部完了は別途検証 |
| 無効 | 無効 | 無担当・保留の候補。自動的に「移管完了」にしない |

通知の送信、受信、業務上のACK、Caseの責任確定、Host Effect Receiptは独立の観測とする。承認・移管を実行するときは、Rule/Case/Responsibility revisionとCurrent Grant/Policy epochを直前まで検証し、対応可能なProviderでは実Commitと同じ原子的境界に入れる。外部APIなどで原子的保証がない場合はその限界をContractへ記録し、Commit不明ではOperation ID/Receipt照会を先に行う。

**ABAの負例**：C1の取扱先がD1→D2→D1と戻っても、元のCase revision、Candidate Selection epoch、Grant、Effect Receipt、ACKを復活させない。表示ラベルの一致と、同じ実行資格は別である。

### 23.5 組織の変更をどの範囲へ伝播させるか

影響分析では、変更されたRelation occurrence/Record Versionから、その関係を**実際に依存として読んだ**Query/Rule/Projection/Consumerへたどる。全ての組織データ変更が全案件の再判断を要するわけではない。

| 変更 | 再評価する対象の例 | 無条件に変更しないもの |
| --- | --- | --- |
| A2のknown版訂正 | A2を読む二時点Query、D1所属/FTE、RuleがA2を要するCandidate | C1 handlingUnit、外部Host Grant、既存Receipt |
| Occupancyの交代 | 当該Positionの候補Ruleと表示 | PersonのMembership、別Positionの責任 |
| C1のResponsibility改版 | C1の候補・引継ぎ・Case表示 | Personの人事記録、全Case |
| Delegationの取消し | その委任に依拠する代理Action候補とHost現在判断 | 過去の確定Effect |
| ProviderのRLS/Scope変更 | 取得被覆に依存する人数、候補集合、否定条件 | 完全性が独立に証明された無関係のClaim |

**影響なし（scoped-unaffected）**とするには、対象Claimに必要な依存集合と観測範囲が十分であることを示す。非出力依存、NOT EXISTS、動的Rule、未知Plugin、取得範囲の変化があるときは、単純な「参照なし」ではなくunknownを返す。Source/Rule/Policy/RecipientがA→B→Aとなる場合も、見た目が元どおりという理由で旧証拠を再利用しない。

### 23.6 既存Expectedへ割り当てる次の小さな実装・検証

新しいORG Scenario正本は作らず、#4490 ORG-BEH-04/05/06、#4501 CX-01〜03、#2510の既存Expected/Driverに次の比較を割り当てる。

| 最小検査 | 正例 | 反例・止める地点 |
| --- | --- | --- |
| R0 責任の出典 | Case→Responsibility→HolderがSource/版付きで一意に追跡できる | MembershipやPositionだけからCase責任を自動確定 |
| R1 候補の結果集合 | complete-none/one/manyをRuleごとに区別し、複数候補の根拠Pathを残す | 不完全な0行を候補なし、partialな1行を唯一と判断 |
| R2 兼務と時点 | 同一Personを候補表示では重複排除しても、根拠A2/A3は両方保持 | Personを二重計上、旧版を誤選択、歴史Grantを現在Grant扱い |
| R3 委任 | 対象・期限・本人/代理者・scopeを別々に検証する | 他Tenant、期限切れ、取消し、自己承認、委任循環を許す |
| R4 責任切替 | 明示的Case revisionと移管Ruleでのみ引継ぎを評価する | 異動だけでCase更新、通知をACKと判断、旧/新双方無効を完了扱い |
| R5 実行境界 | Host Current Grant・対象版・Effect/Receiptを別に検査する | Candidate成功をAction成功に昇格、Commit不明を無条件再試行 |
| R6 影響と再表示 | 変更したSource依存だけを再評価し、Tree/Graph/Tableの同一性を保持 | hidden edge削除、Source/Case/GrantのABA再利用、未完全Readの影響なし判定 |

**検収票の最小列**：Case ID/Case revision、Duty/Scope、Holderの関係Occurrence、Source URL/span/revision、validAt/knownAt/observedAt、Rule revision、Required vs Actual Read footprintとcohort/coverage、候補集合と選択根拠、実Principal/Grant epoch、Effect/Receipt/ACK、Consumer/Purpose/Recipient、独立Expected、Actual mechanism、Git exact HEAD、Owner acceptance。これは既存Evidenceの**用途別表示**であり第二Storeの新設ではない。

実装順は次の通り。

1. **純粋な候補解決を一件実行**：現行WU-09の構造検査を拡張し、同じ合成Sourceで候補0/1/複数/不完全のObservedを独立Expectedへ比較。最初は実Grantを要求せず、既存関係ReaderとRuleのFit/Gapを記録。
2. **代理の負例を実評価**：現行WU-10のscenario存在確認だけでなく、対象・期限・取消し・自己承認・Tenant違いについて実Rule/Policyのどこまで検査したかを別々にAssert。
3. **Case変更とHandoffを独立検証**：#4501のCX-01〜03を流用し、組織異動だけではCase責任が変わらず、移管には明示Changeと必要ACKがあることを観測。
4. **製品への接続**：#2586の通常CLI→Provider→Context→Consumerの受入を再利用した後、組織Queryを接続。現在Grant/Effect Fenceの統合は#1174/#2499のOwnerと実Hostの別資格。
5. **表示・生成の収束**：同じ責任RelationとCandidate Traceから、Studio/SHOの説明図・候補表・手順・必要Targetコードを導出し、業務Ruleの手書き重複が減ったかを§22.4の測定票へ記録。

停止条件は、(a) Caseとr7のSourceを根拠なく合成、(b) 候補から現在Grantを発行、(c) incompleteを否定・唯一へ縮約、(d) 非単射Viewからの逆編集、(e) 実HostなしでEffect完了を主張、(f) 新Core/専用Workflow/第二Policy/Expected Engineを作ること。公開契約の追加・業務Ruleの採否が必要なら#2386、実効果の資格は#1174へ戻す。

**全体P0との関係**：#2586の通常CLI初回実Provider Read、#2831のStudio Stage1→Stage2セルフビルドを本節の完成待ちにしない。現在は**実装を定義した設計段階**であり、候補/代理/C1 Handoffの新Product E2E・Owner Meaning Acceptanceは未実行・未受理である。


## 24. 組織グラフと実行グラフを、標準意味・Plugin・Studioへ接続する（2026-11-14）

> **状態：設計・Owner未受理、新Product試験は未実施。** この節は§22の最小縦断、§23の責任→候補→実行資格を、他レーンの成熟した検討へ結び付ける補遺である。新Core、組織専用のGraph/Workflow/Authorization Engine、第二Scenario/Expected正本を作らない。通常tei CLIの実Provider Read（#2586）とStage1→Stage2セルフビルド（#2831）は独立P0のまま維持する。

### 24.1 組織の意味、処理の計画、実行記録、図面は別の対象である

| 区分 | C1での代表例 | 混同しないもの |
| --- | --- | --- |
| **業務関係グラフ** | 人物P2と所属A2/A3、部門D1/D2。別SourceのCase C1とResponsibility | Person／Membership occurrence／Record Version／表示Edge、r7とCaseのSource |
| **実効計画グラフ** | Caseを読む→Ruleと関係を解決→判断候補を返す。選択したQuery/Provider/Hook | Meaning Definitionと、今回実際に選択された実Invoker |
| **実行記録** | actual SQL/params、Rowset、Read Cohort、Consumer受領、必要ならHost Effect/Receipt/ACK | 実行予定のフロー図、Mock、別Attemptの結果 |
| **表示・編集用グラフ** | Studio/SHOの組織Tree、Graph、Table、二時点表、業務フロー | 図形、座標、集計セル、hidden edgeを元Sourceの実体とみなすこと |

これは[main統合済み #4668](https://github.com/hat47x/tei-platform/pull/4668)のDefinition／Effective Plan／Observed Trace／ProjectionをC1へ対応付けるものであり、新しい汎用Graph Compilerを意味しない。起動・応答には[main統合済み #4666](https://github.com/hat47x/tei-platform/pull/4666)のLifecycle／Trigger／Invocation／Outcome／Deliveryを再利用する。所属が訂正された事実からCase移管Triggerや実Writeを自動生成しない。

### 24.2 標準データ構造の意味保存は三種類に分けて検査する

[意味的対応付けDraft #4667](https://github.com/hat47x/tei-platform/pull/4667)の構造・評価・変更の同値性を、採用前の観測条件として使う。新しい文法や第二Semantic Engineを先に入れず、既存Meaning／Facet／Law／Trait／Projectionと照合する。

| 検査 | 保持するもの | 反例 |
| --- | --- | --- |
| **構造** | Person・Membership occurrence・Record Version・Row、Edge ID、向き・型・多重性・Source位置と版 | P2のA2/A3を1所属へ併合、同名別Personを同一視、同じ端点の並行Edgeを消去 |
| **評価** | validAt/knownAt/observedAt、粒度・単位、bag/set、NULL/欠測、JOIN増幅、非出力依存、取得の完全性 | r7のPerson 3→3だけを見てMembership 4→3・FTE 3.0→2.4を見落とす。partial結果からNOT EXISTSを真にする |
| **変更** | 対象Source occurrence/span/revision、局所差分、選択epoch、CAS、実Provider/Host資格 | Treeのhidden edge、集計行、複数経路から元関係を推測して書戻し、A→B→Aで旧Pinを再利用 |

TreeやRACI表は用途別のProjectionであり、表現を単純化した結果として失う情報を明示する。情報を失った投影はまずread-onlyとする。Get–Put／Put–Get等の性質は一意対応が成立する限定Patchの検査候補に留め、描画やMarkdown整形をSource変更の権限としない。図面、説明文章、生成コードの一致をそのまま業務意味の受理に昇格しない。

### 24.3 公開された上位Pluginのグラフから、横断的な関心事を組み込む

上位Pluginが契約として公開した意味Node/Relation/Process Step/Hook Pointは、下位Pluginから**公開版・ID・Target・Scopeを明示して参照**できる設計候補とする。内部グラフや実装都合のNodeを暗黙継承させない。Meaning Pinと段階別Contributionについては[既存Plugin Reference Draft #4646](https://github.com/hat47x/tei-platform/pull/4646)を参照し、組織専用Resolverを新設しない。

- **適用対象**：公開Meaning ElementのID/Role、Process Node/Operation、包含・除外、Source/Plugin版から選ぶ。名称一致や画面全体の見た目では決めない。候補が曖昧なら停止する。
- **合成**：PluginのDistribution、Authoring、Composition、Organization、Runtime、Disclosureを分け、実Target、Provider/Generatorの選択、Hook順序・重複・依存競合を検査する。Declared≠Selected≠Loaded≠Authorized≠Effectを維持する。
- **処理段階**：候補取得前のread-only説明、Validation、Commit前のGuard、Commit後の通知は異なる作用境界である。Hookが公開されていることから、外部Hostの原子的Commitや相手ACKは推論しない。
- **B2C等の横断要素**：計測、同意管理、監査ログなどを、公開されたアプリ/実行グラフの対象選択→Generator/Hook束縛として比較する。全ページ計測の暗黙許可や、Person/Account/Principal/Tenant/REA Agentの同一視は認めない。標準RendererやTEI独自Web RuntimeをHeadless Consumerに強制しない。
- **最初の実験**：C1のread-only候補照会に、必要Relationの出典・時点・Read Completenessの説明を付ける非Effect Hookを一箇所だけ試す。実Case更新や代理Actionは、Host側の現在Grant/Effect Fenceを別途検証してから接続する。

### 24.4 三要素分析法、REA、SUI/SHO、AI Harnessとの対応

三要素分析法は既存組織段階計画§10.7を優先する。DataにPerson/Membership/Case/Responsibility、FunctionにRead/候補解決/Case変更、Businessに承認・引継ぎ規則を置き、同じSourceを三要素へ投影する。ただし三要素の図を作ったことは、Function/Hostの実行を意味しない。

**REAという名称は二義を明示する。** Resource–Event–Agentは業務資源・事象・関与主体を読む分析レンズであり、そのAgentをPersonやHost Principalへ自動同定しない。一方、Reverse Engineer Anythingは既存プログラム・文書・DBから出典付きのモデル候補を抽出する手段であり、抽出候補をOwnerが受理するまで正式Relation/Ruleへ昇格しない。

SUI/KJ配置上の近接は所属関係の証明ではない。SHO文書やAI生成の図はSource付きの用途別Consumer投影であり、選択固定→Preview→Ownerの判断→再投影を経由する。Source/Rule/Policy/Recipientや選択対象が変化した場合、後着のAI提案、著者編集との競合、A→B→Aを再資格化し、旧Pinの自動Applyを拒否する。

### 24.5 二つのWorldを黙って合成しないC1縦断

第一のSourceはorg-synthetic-r7のP2兼務・A2旧新版・人数/FTEであり、第二のSourceはCase C1のhandlingUnit/Responsibility/Handoffである。両者に業務Ownerが採用したCase→OrgUnit→Role/Position/Personの関係がなければ、「候補未接続」を明示する。P2兼務からC1の責任者=P2というExpectedを捏造しない。

受理された明示Linkがある場合にのみ、①業務Relation Graph、②候補解決のEffective Plan、③実Query/ConsumerのObserved Trace、④Graph/Tree/Table/SHOへのProjectionを、Source/Case/Rule/Plugin版とRead Cohortを明示して照合する。同じCase ID・同じ人物数・同じWorld Fingerprintだけでは一つのAttemptに結合できない。

### 24.6 既存受入票へ割り当てる追加観測（GXは本節内のラベル）

| 観測 | 既存Owner/Expected | 正例・停止条件 |
| --- | --- | --- |
| GX-1 意味と実行グラフ | #4490 ORG-BEH-06/07、#4501 CX-04〜06、#4668 | Relation/Plan/Trace/Viewを別版で区別。処理図だけから実Invocation/Effectを主張しない |
| GX-2 構造と投影 | #2497、#4501 CX-04〜06、#4667 SM候補 | 多重辺・hidden parent・Person/Occurrence/Versionを保持。集計・欠損Treeから逆編集しない |
| GX-3 評価の同値 | #2510 r7、#4490 ORG-BEH-01/02、#2586 | 実SQL/Params/Rowset/Cohort/Completeを確認。partial、NULL、JOIN fanout、NOT EXISTSの意味差を拒否 |
| GX-4 公開Plugin契約 | #951/#4646、#4501 CX-17 | 非公開辺参照・Plugin版違い・重複Hook・競合する実Target/Scopeを合成停止。配布からGrantを生成しない |
| GX-5 責任と起動 | #4490 ORG-BEH-03/04/06、#4501 CX-01〜03、#4666 | A2訂正≠C1移管。明示Trigger/Case改版/Receipt/ACKを分ける |
| GX-6 知識・AI編集 | #4501 CX-07〜12、#4515/#2516 | Source/Selection ABA、異Cohort、保護Fieldの公開二世界、遅延回答/保存競合で誤適用を拒否 |
| GX-7 TEI生成の実効 | #2831/#2600、既存§22.4 | Source側へ移せた業務意味、重複手書き削減、実Generator/Plugin利用、Stage2起動、独立Consumer受領を別々に測る |

GX-1〜7は**補遺内の観測ラベル**であり、新たなExpected ID・Scenario Store・Runnerではない。既存Driver/独立Expectedへ割り当て、Expected SourceとActual Mechanism、Git exact HEAD、受領先、Owner Meaning Acceptanceを別列にする。

### 24.7 実装順と停止線

| 順序 | 最小の成果 | 製品資格の上限 |
| --- | --- | --- |
| **OG-0：意味の対照** | C1のRelation/Process/Trace/ViewのID、Source、時間、被覆、投影損失を一つの対応表に整理 | 読取専用の設計・ギャップ分析 |
| **OG-1：Headlessの候補実評価** | 既存WU-09/BEH-04で候補0/1/複数/partialと根拠Pathを実Rule/ReaderでAssert | 構造の存在検査は合格にしない。実Grantなし |
| **OG-2：実Providerの縦断** | §22のPerson Link・A2二版共存→実SQL→Read Cohort→Context→独立Consumer | Product read-onlyの資格。別HostでのActionは未受理 |
| **OG-3：Plugin/表示** | 公開Graphに限定した非Effect Hook、Tree/Graph/TableのLoss Report、Studio/SHOのSource-backed Preview | Pluginの実Loaded/契約・Consumer受領を別証拠化 |
| **OG-4：実業務効果** | 明示Case移管、現在Grant/Effect Fence、Receipt/ACKを既存Hostへ接続 | Hostの実資格がなければNOT PROVEN |
| **OG-5：自己生成** | DSL/Diagram/Generator/Pluginの二重記述削減を計測し、TEIで生成する範囲を段階拡大 | Builder実選択・Stage2起動・実Target利用を独立に検証 |

OG-0〜5は本節の作業順ラベルで、既存のDB移行G0〜G5とは別である。**初回Product Read #2586とDesktopセルフビルド #2831をOGの完成待ちにしない。** 新DSL文法、Graph Compiler、組織専用Host/Workflow/Policy、全Dataの履歴表、全Plugin共通の強制Hook Runtime、Graph集計の自動逆編集はこの段階では採用しない。

**今回の証拠範囲**：§24は既存Issue/PRを参照したdocs-only計画であり、Go、PostgreSQL、Windows、Host、AI/SHO、実Pluginの新試験を行ったという意味ではない。Draft #4667/#4646/#4633の限定証拠や参照Fixtureの成功を組織C1の製品受入へ昇格しない。
