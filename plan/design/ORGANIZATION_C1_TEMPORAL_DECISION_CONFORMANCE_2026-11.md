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

**Read Projectionの新しいGAP**：現Query出力の3列からは person_id や各記録版の独立identityが観測できない。Membership行の存在確認は可能でも、Query単独でdistinct Personや同一Occurrenceのどの記録版を読んだかを確定したとは言えない。必要なClaimに応じて、(a)同じRead Cohort/Scopeの型付きMembership→Person関係を追加取得する、または(b)必要な識別列を明示投影して実Row Traceを記録する、を既存Query/Relation契約で比較する。全Queryへの新必須列や第二Context Storeは作らない。

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
- Draft #4623のCaller選択Builder実exe SHA事前照合はSelected B/Actual Aの起動前拒否であり、BuildIntent Resolver選択・全依存閉包・OS Loader TOCTOU・Stage2 Desktop実起動を証明しない。#4621/#4601のContract/選択/実Build inclusion/actual Invocation/Effect/Consumer Disclosureの分離を組織C1の生成・読取・再投影へ適用する。
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
