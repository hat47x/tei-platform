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
