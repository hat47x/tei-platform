# 組織C1：観測の十分性から業務判断・Consumerへ進むための受入計画（2026-11-17）

## 位置付け

本書は [組織モデリング段階計画](ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md) §10.20〜10.21と [C1横断受入補遺](ORGANIZATION_C1_TEMPORAL_DECISION_CONFORMANCE_2026-11.md) §43〜44の**実装順・反例・完了条件**を整理する。組織の意味定義を新設せず、#2386/#1126を正本とする。通常CLI実Read #2586とDesktopセルフビルド #2831は先行P0のまま維持する。

## 1. 受入済みの範囲と未受入の範囲

- main統合済みPR #5155の隔離PostgreSQL CI run 43859032163では、同じWorkspaceの型付きRelation/Query宣言を確認したうえで、選択Queryが実DBから手書きExpectedどおりの行を返すことを確認した。**Relationを実JOINで辿った証拠ではない**。
- main統合済みPR #5210の隔離PostgreSQL CI run 43900533159では、同じMembership Queryの3列・4行が完全に一致する二つの候補世界で、仮想Person対応によるdistinct人物数が3人と4人に分かれることを確認した。**仮想対応は正式Person Relationではない**。
- 現行Queryの3列は membership_id / org_unit_id / allocation。Personの正式な識別・リンク、同一A2の正式二版保存、通常CLIの一Consumer受領、二Consumer同版、業務Rule/Host Effectの受入は別資格であり、上記PASSから推定しない。

## 2. Claimごとに必要な情報を決める

| Claim（問い） | 集計単位・必要な情報 | 未確定とする条件 | 受入担当 |
| --- | --- | --- | --- |
| 観測された所属件数 | Membership occurrence、validAt/knownAt、選択QueryのRead Scope | 全件取得を要求する用途でpartial/permission-denied、母集団不明 | #2586/#2506 |
| 重複しない人物数 | Owner受理済みMembership→Person、人物同一性、対象集合の完全性、時点 | Person未リンク、兼務の識別不能、別Read Cohort | #2386/#1126/#2514 |
| 配賦FTE | allocationの単位・期間・合算規則、対象Occurrenceの重複排除、全件性 | 配賦値のみを人数に読み替え、異粒度混合、期間不整合 | #2593/#2514 |
| Caseの責任者・決裁候補 | Case責任、Position/Role/委任、Rule revision、必要Read集合 | 0件/複数候補/partial、Rule未受理 | #4490/#1125 |
| 実行可能な承認 | 現在Principal/Grant/Policy epoch、対象Operation/Resource、Effect Fence | 過去所属や候補だけ、失効、Commit uncertain | #2499/#1174 |
| 表示・外部開示 | Relation occurrence/版、Projection loss、Disclosure slice、Consumer受領 | Graphの隠れEdge、表示からの逆推定、受信未確認 | #2497/#2516/#1124 |

「0件」「unknown」「not-found」「permission-denied」「partial」は同義にしない。情報不足を単なるfalseや0へ潰さず、既存のContext/Scenario/Expectedで扱う。特定の組織専用Decision EngineやACLは追加しない。

## 3. 最小の製品縦断（Gate）

| Gate | 同一Attemptで実証すること | 明示的に未合格のまま残すこと |
| --- | --- | --- |
| G1 通常CLI受領 | 通常CLIでFunctionを選択し、同じSource/World・Query ID・入力・Provider・Read Cohortの結果を**一Consumerが実受領**。独立Expected、非SKIP、固定HEADを照合 | 別のreference testの成功や単なるQuery実行をConsumer受領としない |
| G2 人物Linkの正式採用 | OwnerがMembership occurrence→Person identityと訂正/再所属の区別を受理。正式Source/物理Mapping/Queryを同じ版にそろえ、二時点で再読込 | 仮想Person対応表、単独membership_idの現行物理PKで正式二版共存を宣言しない |
| G3 Claimの情報十分性 | 同じRead Scope/Cohortで必要Person関係が閉じ、手書きExpectedとdistinct人数・配賦FTEを別々に比較。意図的partialではunknown | Membership行数=人数、配賦合計=人数、同じ画面=同じClaimを認めない |
| G4 二Consumer・Projection | Function ContextからCLI/Studio等二Consumerが同じ版とScopeを受領。Graph/Table/Tree/SHOの損失と固定Selectionを検査 | 受信・保持・表示・意味受理・編集Commitを同一視しない |
| G5 判断・Effect | Caseの必要Read、Rule版、候補0/1/複数/unknown、現在Grant、Host Receipt、ACKを別票で検査 | 候補=権限、通知=移管、過去Receipt=現在Grantを認めない |

G1とG2のOwner作業は並行可能。G3は必要な正式Linkが整ってから、G4の表示契約は独立Expectedが固定された範囲から着手できる。G5は既存Host/Businessの受入に従い、組織固有Runtimeを作らない。

## 4. 追加する判別ケース

1. **同一行・異なる人物世界**：既存PR #5210の3人/4人反例を再利用し、Person Relationなしでは人物数Claimをunknownにする。
2. **同一Person・複数Membership**：兼務・出向でMembershipは複数、人物数は1。部門別人数と全社distinctを別Expectedとする。
3. **同一validAt・異なるknownAt**：遡及訂正の旧新版を区別し、物理記録時刻をknownAtに読み替えない。
4. **partial/閲覧不可**：返った行数は正しくても母集団の人物数・唯一決裁者を確定しない。公開Sliceの違いから隠れた人物数を漏らさない。
5. **多重親Graph**：同名Nodeや複数Edgeを表示で縮約しても、元Relation occurrenceと選択対象の同一性を失わない。非可逆ならread-only。
6. **異動・責任移管・Grant失効**：所属変更だけでCase責任を移さず、Handoff ACK前の責任をRuleに従って判断。旧Grant/Receiptの復活を拒否する。
7. **A→B→A**：表示値が元に戻ってもSource revision、Selection epoch、Policy epoch、Read Cohortの旧証拠を再利用しない。
8. **生成Goと汎用評価**：同じMeaning・入力・NULL/欠損・時点・Scopeで手書きExpectedと照合。生成メソッド名は意味IDとしない。

## 5. TEI横断設計への接続

- **標準Data/Relation/Query**：人物同一性・関係方向・多重性・二時点を型付き経路で保持。複合型Port/AdapterもShapeだけでなく意味LawとBindingを検査する。
- **三要素分析・REA**：Agent/Person、資源、イベント、責任を候補関係として結び、出典とOwner受理を経て正式Relationへ進める。外部観測を所属や責任へ自動昇格しない。
- **BRMS/Process**：必要Read集合とRule版を明示し、判断候補・責任移管・Host実行・Receiptを独立に受理する。
- **Studio/SHO/SUI**：組織図、配置表、Tree、文書を同一Sourceから投影。表示損失、固定対象、文書改訂、逆編集資格を別に扱う。
- **Plugin/API/CLI/Wails/B2C**：意味モデルは入口に依存させず、選択→含有→呼出し→受領を段階別に検査。組織所属、Tenant、Account、Principal、配布範囲を同一視しない。
- **Build/Evidence**：Source/World/Model/Artifact revisionとGenerator資格、Expected/Observed、実行Attemptを区別。別runの局所PASSを合算して製品合格としない。

## 6. 次の作業と停止条件

1. #2586にG1の実Consumer受領を最小追加し、既存のQuery/Provider機構を再利用する。別のOrg Readerや新しいTrace Storeは作らない。
2. #2386/#1126でPerson Relation・Assertion revision・単一値性の採否を記録し、正式Persistenceの旧新二版保存と再起動後再読込を別試験にする。
3. #2514/#2593/#4490でG3のClaim別Expectedとpartial負例を追加し、G4/G5へ段階的に渡す。
4. #2831のDesktopセルフビルドは独立P0とし、組織の高度化を新たな起動条件にしない。

**停止条件**：正式Person Linkがないまま人物数を確定する、異なるScope/Attemptを結合する、候補を現在Grantに昇格する、表示/AI/SHOの成功を業務受理にする、局所PG PASSを製品E2Eと呼ぶ、またはS1の簡単な組織利用へ専用Runtimeを強制する場合は該当Claimを未受理へ戻す。

**本書の状態**：設計・実装計画。ここに記した新Gate/判別ケースは実行済みではない。上記既存CIの限定的な実測と区別する。

## 7. 同一Readを「業務判断に使える」と認めるための検収（2026-11-18）

[高度組織モデリング段階計画§10.25](ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md#1025-実readから業務判断へ渡す証拠を問いごとに閉じる2026-11-18)を、既存G1〜G5の検査手順へ接続する。Providerが返した行の範囲と、Consumerがその行から導く結論の十分性は別の資格である。

- **G1／Read**：実際のFunction・Query・SQL/引数・時点・Scopeと、Rowset→Context→一Consumerの受領を同一Attemptで確認する。`covers`の宣言、実SQLの保存関係との照合、実行したQueryの一致はそれぞれ別に記録する。
- **G3／Claim**：Membershipを対象とするEXISTS／NOT EXISTS／高々1件は、同一時点・Scopeのcomplete/partial 0・1・2件で真・偽・未判定がどう変わるかを既存ORG-BEH-02／#2514の独立Expectedと比較する。観測行が対象集合へ属するという証拠がない限り、partialの正例も確定しない。
- **G2→G3／Identity**：同じ4所属からPerson 3人／4人となる二世界の反例を保持する。Person Relationの正式受理、実取得、QueryのCoverageが揃わないうちは、人物数・唯一決裁者はunknown。r7のFTE 3.0→2.4はknownAtの違う世界として扱う。
- **G4→G5／用途・実行**：二Consumerの受信・公開範囲、Case責任と異動、承認候補と現在Grant、通知とACK／実Effectを混同しない。Rule・Scope・Source・Policy・Recipient・Selection epoch変更後の旧結果再利用を拒否する。

### 最初の小PRに必要な検収票

| 列 | 比較する内容 |
| --- | --- |
| Required | 問い、量化、母集団／粒度、必要Relation・Field・Rule・非出力依存、validAt/knownAt |
| Actual Read | exact HEAD／Attempt、Function/Query、実SQL/params、Provider、Snapshot、Scope、取得行Identity、Coverageの根拠と欠落 |
| Claim | 真／偽／未判定、成立側の証人・反例、必要な同一性Linkと受理版、独立Expected |
| Delivery | Contextの受領Consumer・用途・公開Slice、Source/Selection/Policy/Recipient版、Host側はGrant／Receipt／ACKを独立記録 |
| Verdict | 確認できた局所事実、依然NOT RUNの製品接続、Owner未受理事項、停止理由 |

第一候補は#2586の通常CLI一Function・一Query・一Consumerへの読取受渡しで、対象はまず既存の合成受注とする。組織C1のPerson Linkや新しいRuntimeを先行条件にしない。並行中の[PR #5382](https://github.com/hat47x/tei-platform/pull/5382)のCoverage実装、[PR #5431](https://github.com/hat47x/tei-platform/pull/5431)のWeb受信は、各々の実行資格と統合状態を確認してから使用する。**本節は設計のみ。実Provider→Org Rule→二Consumer・Hostの合格は未実証。**

## 8. 複数Resultとページの整合した受領（2026-11-18）

[組織モデリング段階計画§10.26](ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md)は、既存G1〜G5に新Gateを増やさず、**同じ組として使う複数Resultを、ConsumerがどのCutで受信したか**を確認する。バッチ[複数成果物の世代整合§15](DECLARATIVE_RESULT_PROCESS_COMPOSITION_2026-11.md)のMG-02/03/04/06/08と、組織ORG-BEH-03／ORG-XD-02〜05の反例を再利用する。

| 既存Gate | 読取・利用上の最小反例 | 受入可能な境界 |
| --- | --- | --- |
| G1 | 一回のQueryは正しいが、ページ1と2を別Transactionで読む間に公開版が切り替わった | 同一Snapshot/版Pinを実確認するか、複数ページにまたがる全件Claimをunknownとする |
| G2/G3 | r7で新所属一覧3件と旧FTE 3.0を同一現行時点として提示、または同一所属行からPerson数を確定 | 旧4件・3.0／新3件・2.4を別Cutで示し、正式Person Link未受理なら人数はunknown |
| G4 | Treeは新しいSource、Tableは旧Source、SHOはさらに古いPinを保持 | Consumerごとの実受領世代・Scopeを明記し、同一組という表示を拒否。比較用途なら差分として表示 |
| G5 | 組織の新世代公開をC1責任移管・通知ACK・現在Grant・Effect確定と同一視 | Case Responsibility・Handoff・Hostの実証を独立の条件とする |

**最初の比較票**：業務as-of（validAt/knownAt）、各Source/Meaning版、実Read Snapshot/Cohort、必要RelationのCoverage、Result IDと入力依存・Generation、公開された組または「未確認」、実ConsumerのPin/受領/開示Scope、Claim別Expected/Observedを列として照合する。Readerが複数要求の固定Snapshotを提供しなければ、その能力を仮定したテストはPASSにしない。実Targetで原子的な組の公開ができない場合も、個別Resultの成功と共同公開の成功は別に報告する。

**実装順**：#2586の一Function・一Consumerの実Readを先行。次にr7の旧新所属＋FTEを独立Expectedでread-only照合し、#2506/#2516で二Consumer/ページを比較。現行GrantとHandoffは既存#1174/#1444で別検収する。main統合済み#5431のWeb実ReaderとDraft #5382のCoverageを、追加のOrg専用Routerなしで利用可能か検査する。#2831のStudioセルフビルドは並行P0。**本節は設計のみで、実Page Pin・Org共同公開・Host実行の新しい試験はNOT RUN。**

<!-- org-gate-witness-minimal-20261119 -->
## 9. G1からG5へ渡す最小Witnessと、実装を止める条件（2026-11-19）

[段階計画§10.27](ORGANIZATION_STAGED_MODELING_PLAN_2026-11.md)の変更影響・再利用資格を、既存Gateの**実行可能な判別票**へ落とす。新しいOrg Test Runnerや常設Evidence Schemaは作らず、既存Scenario/Expected、Function Data Context、実行Attempt、Consumerの受領観測に必要な列だけを添える。

### 9.1 まず通常CLIの一往復を成立させる

**G1の最小成功例は組織C1ではなく既存の合成受注**とする。固定HEADで通常CLIからFunctionを選び、採用されたQuery/Providerが実行したSQL・引数・Result Row Identity・Scope・Read Cohortを同じAttemptへ束縛する。Function Data Contextへ格納したResultを**同一Attemptの一Consumerが実受領した事実**を、独立Expectedと比較する。

| Case | 入力／変化 | 必要なObserved | 合格としない結果 |
| --- | --- | --- | --- |
| G1-01 正常受領 | 一Function、一Query、一実Provider、一Consumer | 選択ID・Source/Query版・実SQL/params・Scope・Row Identity・Context格納・Consumer受領が同一Attempt | ProviderのSQL実行成功だけ、別RunのConsumer受領、Mockだけの成功 |
| G1-02 空結果 | 同一Queryで対象0件 | 実Providerが空を返し、Consumerが空Resultを受け取ったこと。Coverage未確認ならClaimはunknown | 空結果を即座に「対象なし」と断定 |
| G1-03 部分取得／拒否 | ページ未完了、Scope不足、Provider拒否を個別に与える | partial／permission-denied／unavailable等を、観測に応じて区別して伝える | 失敗を空集合やfalseへ変換 |
| G1-04 版・受信不一致 | Source/QueryまたはSelection epochを変更、旧応答が後着 | 旧Resultの実受領を記録しても、新Attemptの確定結果に採用しない | A→B→Aの表示一致だけで旧受信資格を復活 |

**G1の完了条件**：実Providerが選択されたこと、同一Attemptの一Consumerが実受領したこと、固定HEADの非SKIP試験と独立Expectedが一致したことを別々に示す。Contextの型やログだけを作ってもConsumer実受領とはしない。通常CLI経路が未完成ならG2〜G5の局所試験でG1を代替しない。

### 9.2 組織C1へ進む際の差分検査

| Gate／Case | 同じ合成C1の変更 | 必須Expected | 必要な実証がなければ |
| --- | --- | --- | --- |
| G2／ORG-XD-02 | 同一4 Membershipから3／4 Personとなる候補二世界 | 正式Link未受理ならdistinct Personはunknown。正式受理後は同一Scope/時点の実Readで別判定 | 人物数は未確定 |
| G3／ORG-XD-06 | r7の旧4所属/FTE 3.0と新3所属/FTE 2.4 | 各Cut内の組は許容。新3所属＋旧FTE 3.0を現行同世代として拒否 | 複数Resultの共同公開は未合格 |
| G3／ORG-XD-07 | 表示Rowを変えずPerson Link／Ruleの否定条件を変更 | Membership観測の再利用候補と人物数／承認Claimの再検証を分離 | 依存の非交差を証明できないClaimはunknown |
| G4／ORG-XD-08 | 同じ内部Resultを別Audience／Tenant／Purposeへ配送 | 受領と開示可能Sliceを用途ごとに再照合。非開示Count/Edge/Sourceを出さない | 配送／開示は未合格 |
| G5／ORG-XD-09 | A→B→A、旧通知ACK・Grant/Receipt、遅延Reply | 現在Grant、対象Operation、Effect Fence、ACK/Receiptを独立確認 | 現行許可／Effect／移管完了は未確定 |

**再利用判定の順序**：①対象Claimと必要Relation/Rule・否定条件を定める、②実Readの時点・Scope・Coverage・Snapshotを照合する、③変更が必要依存に交差するか調べる、④再利用候補であってもConsumerのPurpose/Audience/Policyを再検査する、⑤Hostの現在Grant・Effectを独立に判定する。途中のunknownをsuccessに丸めない。

### 9.3 受入票の最小列と報告単位

- **入力資格**：exact HEAD、Attempt、World/Source/Meaning/Query revision、Function、Provider、実SQL/params、validAt/knownAt、Snapshot/Read Cohort、Scope、Row Identity、Coverage根拠。
- **Claim資格**：問いと量化、必要Relation/Rule・非出力依存、粒度、独立Expected、Observed、true/false/unknown、片側証人または反例。
- **配送・作用資格**：ContextのResult revision、Consumerの実受領・Selection epoch、Purpose/Audience/Disclosure、Hostの現在Grant、ACK、Effect Fence/Receiptを**該当Gateのみ**記録。
- **判定**：PASS／FAIL／NOT RUN／NOT ACCEPTEDを混同せず、実行した試験の固定HEAD、非SKIP、対象Provider/Consumer、残った未実証の接続を明記。

実装担当は#2586/#2506でG1-01を最小PRとし、G1-02〜04を負例として拡張する。#2514がG3の独立Expected、#2386/#1126が正式Person意味受理、#2516/#1124がG4、#1174/#2499/#1125/#1444がG5を引き受ける。#2831のStudioセルフビルドは引き続き独立P0。**組織モデルの追加要件を通常CLI・Studioの起動条件へ昇格しない**。

**本節の状態**：受入票と判別ケースの計画。G1-01〜04、ORG-XD-06〜09の新しい製品縦断試験は未実施。既存の隔離PostgreSQL局所PASSやWeb Readerの限定実証を、ここで定めたConsumer・Host受入の合格に読み替えない。

<!-- org-g1-implementation-cut-20261119 -->
## 10. G1-01の実装着手を、既存#2586の合成受注に限定する（2026-11-19）

前節のG1-01〜04を同時実装せず、まず**G1-01の一往復が本当に製品経路を通ったか**だけを確かめる。#2586の既存合成受注（親H、明細D1のL1/L2各100、独立配送D2の3行）を使い、既存のReference CLIと通常の製品`tei` CLIの差を解消する。組織C1固有のPerson Link・時間履歴・Ruleをこの最初のPRへ持ち込まない。

### 実装前に確認する接続点

| 確認順 | 既存実装で調べること | 採用／不採用の基準 |
| --- | --- | --- |
| 1. Entry | 現行`tei` CLIでFunctionを選択する実経路、入力の解決、既存の実Provider起動点 | 新しい`org`コマンドやReference専用実行経路を本番入口として増設しない |
| 2. Read | #4539で限定実証したEnvironment→PostgreSQL→Rowsetのどの部分を通常CLIから再利用できるか | SQL/params、接続先、Query選択理由、取得行のIdentityを実行時に追える |
| 3. Context | #2506のFunction Data Contextへ、実際のRowsetとScope/Completeness/unknownをどう受け渡すか | 空集合と未取得、部分取得と完全取得、SQL NULLと結論unknownを混同しない |
| 4. Consumer | 既存のCLI JSON等で実際にContextのResultを読んで出力する最小Consumerはどれか | 単なるログ、Provider直後のprint、別Attemptの再読込を受領と呼ばない |
| 5. Oracle | #2586の合成受注の独立Expectedをどこから読み、JOIN増幅と業務Entityの違いを照合するか | 実装の返値からExpectedを自動生成しない |

### 一PRの最小受入と意図的な誤実装

初回PRは**一Function、一Query、一Provider、一Consumer、固定Fixture**に限定し、以下の独立した失敗を識別する。

1. **正しい経路**：実DBからD1の異なるL1/L2を取得し、Contextへ格納して一Consumerが受領する。業務Entityは2件、D1単独の金額合計は200。実SQL・引数・結果粒度・Scope・受領を同じAttemptに結ぶ。
2. **JOIN増幅の誤り**：D1×D2のJOIN結果6行を6業務Entity、単純SUM 600をD1単独合計200と扱う実装を独立Expectedで落とす。SQL行数とEntity数は別のClaim。
3. **受領のすり替え**：Providerが成功してもConsumerを未接続にした負例、別AttemptのContextを読ませた負例を落とす。実ProviderのPASSと一ConsumerのPASSを合算しない。
4. **不足の消去**：D2が1/3しか取得できないとき、D2全件やJOIN全件をcompleteとする誤実装を落とす。D1だけに依存するClaimは必要集合が閉じる場合に限って別に評価する。
5. **版の取り違え**：Query/Source/Selection epochを変更して旧応答を遅延させる。旧Attemptの結果が履歴として残ることと、新Attemptの確定結果に採用できることを分ける。

最初のマージ条件は**1と3の実経路検証、および2の独立Expected**。4と5は同じ受渡しの負例として追補してよいが、未実装ならNOT RUNと明記する。G1-01が成立していない状態で、組織のORG-XD-06〜09やG2〜G5を理由に初回PRを肥大化させない。

### 実装・レビューの分担

#2586は接続と一往復の受入だけを所有し、Query/SQLの意味は#2387、ContextとCoverageは#2506、独立Expectedと判別力は#2514、再取得・旧応答の失効は#2582へ戻す。通常CLIの実入口が未接続なら、まず既存の選択・Provider・Context・Consumerの**最初に切れている辺**を一つだけ修復する。製品APIの名称やJSON Schemaを設計文書から先に固定しない。

**レビューで要求する事実**：固定HEAD、実際に呼んだ製品コマンド、実Providerと実SQL、同一AttemptのContext/Consumer観測、独立Expectedの出典、非SKIP試験、負例のFAIL→修正後PASS、未実施範囲。Go単体テストや隔離ReferenceのPASSだけなら「局所PASS」と記載する。

**現時点の状態**：実装の着手順とマージ条件を確定した計画。通常CLIのG1-01製品縦断・上記負例は本追補では実行していない。#2831のStudioセルフビルドは独立P0のままとする。

<!-- org-g1-edge-diagnosis-20261119 -->
## 11. G1-01の切断点を一つずつ特定する診断手順（2026-11-19）

§10の実装PRを、既存Referenceと通常CLIの差分を埋める**最短の接続修復**として扱う。成功した部品を列挙するのではなく、同一Attemptのどの受渡し辺が未成立かを特定する。#2586のHJ-01〜04と#2506のContext、#2514の独立Expectedを再利用し、新しい証拠保存基盤は設けない。

| 辺 | 実際に観測するもの | 切れていた場合の判定と次の作業 |
| --- | --- | --- |
| E0 CLI→Function | 製品`tei`入口で選択したFunction ID、入力、Source/Meaning版 | 入口がない／別Referenceへ迂回ならE0未接続。既存CLIの選択経路を先に修復 |
| E1 Function→Query | 選択されたQuery、Dialect/Binding、SQL候補と実選択理由 | Query宣言だけならE1未接続。#2387の既存Resolverへ戻す |
| E2 Query→Provider | 実接続先、実SQL/params、実DBのRow Identityと取得Scope | Mock/隔離ReaderだけならE2製品未接続。#4539の既存経路を流用 |
| E3 Provider→Context | Result Slice、grain、NULL/empty/partial、Read Cohort、同一Attempt | ログだけでContextへ渡らなければE3未接続。#2506の既存境界を使う |
| E4 Context→Consumer | ConsumerがContextのResultを読み、受領したAttempt/Result版を照合 | Provider直後のprintや別RunのJSONはE4未接続。まず一Consumerに限定 |
| E5 Consumer→Expected | 手書きFixtureから導いた業務Entity 2件、D1合計200と、実受領の比較 | 実装値からWantを生成したら独立性不成立。#2514の既存基準へ戻す |

**判定の原則**：E0〜E5は同じSource/World/Query、Scope、実行Attemptに結ばれたときだけ一往復のPASS候補とする。各辺の局所PASSが別HEAD・別Fixture・別Attemptなら、全体判定はNOT RUN／未接続。E0から順に**最初に切れた辺**を修復し、後段の未観測をFAILと混同しない。逆に実行された辺でObservedがExpectedに反した場合はFAILとする。

### 既存の局所証拠を再利用する条件

PR #4539の実PostgreSQL→一時ContextはE2/E3の限定的な先行証拠だが、E0/E1/E4の通常CLI経路を保証しない。#2586の合成D1/D2のJOIN 6行、D1単独SUM 200、JOIN単純SUM 600、SUM(DISTINCT amount) 100という**互いに異なる問い**を保持する。SQLの値が合っていても、粒度やClaimを取り違えたらE5を合格にしない。空集合・partial・permission-deniedも同じJSON nullへ潰さない。

### レビュー時に残す一枚の結果票

`HEAD / CLI実コマンド / Fixture / Attempt / E0〜E5各Observedと根拠 / Expectedの独立出典 / 実Provider・SQL・params / Context Result ID・Scope・Completeness / Consumer受領 / 非SKIP試験 / 最初の切断辺 / 未実施の後段`。

この票は既存Issue/PR本文とScenario出力で十分であり、常設Schemaを先に採用しない。正常例でE0〜E5を閉じてから、§9のG1-02〜04（空結果・partial/拒否・遅延応答）を順に加える。組織C1の人物数・二時点公開・現在Grantは別Gateのまま維持する。

**状態**：診断手順の確定のみ。E0〜E5を同一製品Attemptで新たに実行したわけではない。#2831のStudioセルフビルドは独立P0。

<!-- org-g1-actual-cli-discovery-20261119 -->
## 12. 実CLI調査でE0の前提を修正する（2026-11-19）

§11のE0「入口がないかもしれない」は、mainの実コードを調べて具体化できた。現行`cmd/tei/cli_commands.go`の`acceptedCommands`には`relation-query`があり、`cmd/tei/cli_relation_query.go`の`runRelationQueryCommand`は`-workspace`または`-world`、`-application`、`-query`を受け付ける。つまり**通常CLIのQuery直指定入口は実在する**。一方、これだけではFunction選択→Query解決→実Provider→Function Data Context→Consumerの同一Attemptを証明できない。

### 次の小PRで実コードを追う順序

1. `cmd/tei/cli_relation_query.go`の`runRelationQueryCommand`から、実行する`driver.RelationQueryExecution`とその結果JSONを追う。どのProvider/Source/Scopeが選択されたか、Contextへ渡す前のResult粒度が何かを確認する。
2. `cmd/tei/cli_commands.go`と`cmd/tei/main.go`の実Dispatchから、既存のFunction選択経路がこのQuery実行を呼ぶか調べる。**Query直指定の成功をFunction選択の成功と呼ばない**。
3. `cmd/tei/cli_result.go`の`cliResultEnvelope`は`schemaVersion/ok/result/error/workspace`を持つ既存のCLI応答枠。これをFunction Data Contextの受領Receiptと同一視せず、既存ConsumerがどのResultを実際に読んだかを別に観測する。
4. Function→Queryの辺が未接続なら、既存`relation-query`の選択・実行処理と#4539の実Provider Readerを**再利用できる最小箇所**でつなぐ。新コマンド、新Provider、新Context Schemaを先に追加しない。

**受入の差**：`relation-query`単独の正常応答は「Query直指定の局所PASS」。Function選択と実Provider、Context、Consumerの同一Attemptが観測できて初めてG1-01の製品縦断PASS候補。実コードの存在は静的確認であり、この追補ではGoテスト・CLI起動・PostgreSQL実行はNOT RUN。§11のE0〜E5はこの差を残して適用する。
