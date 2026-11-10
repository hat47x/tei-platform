# TEIテスト基本設計と証拠台帳：Case再利用とClaim単位の再資格化（2026-11-06、Draft）

**位置づけ**：本書は[AI-firstテスト計画](./CROSS_ROUTE_FUNCTIONAL_PARITY_AND_TEST_DERIVATION.md)の基本設計側の参照文書である。2026-11-05のUnit/Fixture・Conformance・Provider/DB・Host/UI・Owner受理の整理と、#2514/#2622/#1129/#1163/#1119等の既存計画を接続する。**docs-only・提案／新試験NOT RUN・Owner採用未決。** この文書は新Runner、新Registry、新Acceptance Authority、新テストDB、第二Scenario正本、製品Wire Schemaを制定しない。

## 1. 正本・派生物・証拠の分界

- **Meaning/Requirement**：受理されたCanonical/Rule/業務資料、Source locator・Role・版・Owner判断が業務意味の参照先。AI生成したExpected、合成Fixture、既存Goの実行結果は意味の受理ではない。
- **Test Definition**：業務Scenarioは#1180の保存済Driver ScenarioがテストのSSOT。DSL PipelineのJSON Candidate（#2813）とDriverのJSON候補（#2828）は異なる局所Consumerの*候補入力*。正式受理済みScenario・固定Anchor・秘匿検証基準をAgentが上書きしない。Markdown表は#2621によるTestとしての表示・Authoring候補であり、説明用表と独立ExpectedをRoleで区別。
- **Execution Evidence**：Go/CLI/CI、実Provider、Target/Host、実Readerが観測したattemptごとの結果・receipt・参照可能なログ・観測可能範囲。別HEAD・別TargetのPASSを合算して一回の実行にしない。
- **Qualification View**：主張（Claim）ごとに、必要な証拠が同時に成立するかを既存記録からread-onlyで導出。歴史的PASSを削除せず、現在資格だけ再評価する。#1163が一改訂横断証拠、#1119が横断Gate判定のOwner。資格計算そのものを新Authorityへ昇格しない。

## 2. 五層は実行順ではなく証明の範囲

| 層／Owner | 主目的・実観測 | この層のPASSが支持する範囲 | 自動では証明しない範囲 |
| --- | --- | --- | --- |
| **L1 Unit / Fixture**（#1129、#2813/#2828） | 型・数値・Rule・Lifecycle phaseを固定した合成入力/negative controlで検査 | その実装・Fixture/Source/Toolでの局所振舞い | 実Source全閉包、独立検証基準、業務意味、実DB、Target、現在Grant |
| **L2 Conformance**（#2514/#2622） | 固定Contract/Profile/independent Expected/Mutantを対象実装・Provider間で比較。NULL/precision/grain/partial/順序を指定 | 指定Profile×対象Version×適用Scopeに対する限定的な適合 | 未テスト能力・本番対象/権限、実Providerへの接続、全Domainの意味同一性 |
| **L3 Provider / DB実測**（#2622/#1174） | 実Data Source alias、Schema/SQL asset、実Query/Read set、実Commit/rollback/unknown、別接続Read/Receiptを検査 | 特定DB/Provider/Principal/Snapshot/Operation/Transaction条件の実動作 | 別DBでの同等性、Web/Host全体、正式Owner受理、将来のGrant |
| **L4 Host / UI観測**（#1163/#1180/#2516） | 実CLI/Web/Desktop binary、Session/HTTP/Command、操作・表示・Consumer取得body、必要なら実Providerへ一Attemptで接続 | そのTarget・Revision・Principal・経路の観測済み機能 | 他Target、表示外のEffect、現在/将来の許可、業務意味受理 |
| **L5 Owner受理**（業務Owner/#1119） | 出典に照らしたRule/Identity/Scope/許容差・例外・運用条件の意思決定を版付きで記録 | 受理された意味・用途・時点の範囲 | 実行正しさ、現在Grant、実Provider/OS/UI/納品資格 |

**L5は自動試験の上位レベルではない。** Owner判断はL1〜L4の前・後・並行に存在し得る。従ってステージを機械的にL1→L5へPASS継承しない。L2の独立検証基準でも、候補・Reference・Expectedが同一の誤ったRuleから派生していれば資格なし。#2514のSoundness/Binding/Coverage/Observational Sufficiencyを**Claimごと**に照合する。

## 3. Case・Attempt・Claimを分離した証拠台帳

**証拠台帳の一行は原則「Case × Claim × Attempt × Target/Environment」**。同じCaseのL1とL3、同じWeb画面の表示とDB Effectは別行。以下は**既存Scenario/Receipt/CI/Owner Decisionを結合する読み取り専用の報告列の候補**であり、製品JSONの正式キーではない。

| 列群 | 必要な情報・明示するunknown |
| --- | --- |
| Case Identity | Case IDとrevision、Semantic Subject/Role、Function/Rule/Operation、層/Profile、negative-control ID、事前required Case/Obligation集合 |
| Source/Actual Input | 宣言Source locator・原byte digest・Model Revision・World Fingerprint、Discovered/Loaded、実Reader/Builderが消費したbyte/選択されたQuery・SQL asset/Plugin Lock。未読込の必要Sourceや否定依存を0件へ潰さない |
| Independent Expected | 期待値または期待状態遷移、成立前提、Oracular provenance/role/revision、Candidateとの共有由来警告、既知Mutantが検出可能か、Owner意味状態 |
| Environment | Git exact HEAD、実Binary/Generator/Toolchain、OS、Targetと起動方法、Provider/DB/Schema/dialect、実Data Source alias、Principal/Grant、fixture/snapshot、観測時刻 |
| Actual Observation | 実argv/request、Attempt/Operation ID、対象Row/Field/Read scope/as-of、output/result/diagnostic、実Provider call、commit state、receipt、別接続/別invocation read、GUI/DOM/Consumer received bytes。未観測なら明示 |
| Required/Observed Check | 必須precondition/Step/Caseの列挙、実発見/Load/Attempt/終端Outcome、SKIP/NOT RUN/blocked/unknown、Stepごとの副作用可能性。Go exit=0やprecondition PASSだけで必要checkの実施を推測しない |
| Qualification/Expiry | 支持する正確なClaimとScope、判定根拠の独立性、binding/coverage/sufficiency、historic verdict、現在資格、valid-at/valid-until（定義される場合）、invalidating eventと再資格対象、権限付きOwner Decisionへの参照 |

**証拠の結合は共通IDだけでは不十分。** Case ID/Source digestが等しくても、実消費byte、Builder/SQL選択、環境Snapshot、現Grant、必要Obligation、Targetが異なれば別Claim。親子Step、Job、Operation、Provider Receiptの相関を#1163/#1174の既存IDで追跡し、二つのDB接続Readが同時点のデータを表すかはSnapshot/Version Witnessで別確認。機密Raw/DSN/holdoutは公開票へ転載せず、#2515/#2516の許可されたProjectionと内部参照のみ。

### 3.1 三つの直交状態

- **Execution**：PASS / FAIL / SKIP / NOT RUN / BLOCKED / UNKNOWN。歴史的実行結果は不可逆に記録する。
- **Evidence qualification**：qualified / limited / unqualified / stale / unresolved。特定Claimと比較時点を引数として判定する。これらは説明上の例であり製品enum未採用。
- **Meaning acceptance**：Owner accepted / pending / rejected / withdrawn（対象Scope・Source版付き）。Execution PASSからこの列を自動更新しない。

**必要集合を固定したうえで不足を測る。** `requiredCaseIDs - observedPassedCaseIDs`が非空、Requiredチェックが未提出・未発見・SKIP、Loader未読込、unknownな対象母集団がある場合、該当Claimはunqualified/unknown。ケース数だけの分子・分母や全Go Test PASSを全業務Coverageに使わない。#2835の「DSN未設定でもGo試験自体は緑だが必要PG試験はSKIP」を既存負例として再利用する。

## 4. 同一CaseをL1から段階強化する圧力試験

採用候補は[#1129 XL-02/03](https://github.com/hat47x/tei-platform/issues/1129)の小さい独立Expectedを用いる。Caseの安定意味Subjectを維持しても、**各環境で同じExpectedが適用可能か**と**各AttemptのSource/結果**は別照合。業務規則・Expectedの改訂時はCase revision/検証基準版を変更し、旧Receiptを現在資格に流用しない。

| 段階と同一Caseの利用 | 固定Expectedと追加する実観測 | Negative control／受入・停止条件 |
| --- | --- | --- |
| **F0 definition** | `order_line.quantity=2, unitPrice=300 → amount=600`を合成の独立Expectedとして固定（税/丸め/時点・粒度が別条件なら保留）。quantity=0、NULL、missing、partialは各自独立のExpectationとする | Source locator・Role・Meaning未受理のRuleを数値例だけで正本化しない |
| **F1 Unit fixture** | 実TEI純粋Operator/Validationを固定Fixtureで呼び、原値・型・結果/拒否を取得。テストGo/Case revisionを固定 | ResultをWantへコピーする同一起源検証基準、decimal→float、NULL/missing混同、required Case SKIP |
| **F2 Conformance** | Go/別実装または2 Provider候補に同じ意味Profile/Expectedを照合。Read Scope・grain・NULL/ORDER・precision、unsupported capabilitiesを明示 | 同じ表示600でもJOIN fanout/partial/粒度・Scopeが異なれば業務同値を主張しない。未知Dialect/SQL fallbackは未資格（#2622/#2862） |
| **F3 実DB/Provider** | 実DBへ限定Seed→実Read Query/Result set→必要Field/Rowを別Witnessで比較。更新Caseはpre-image/Version/Current Grant→実Command/Tx→Receipt/別接続post-imageを追記 | 読込み不足でFixtureの値を補ってPASSしない。ACK lost→UNKNOWNを再送成功/未作用へ変換しない。新Idempotency keyでABA再適用され得る既知反例を含める |
| **F4 実Target/Consumer** | 同じ受理対象版と実Binary/Host/Principalを使いCLI/Webで実施、操作・HTTP/DOM/表示・実Consumer received bytesと実Provider依存を同一Attemptで結ぶ | Web表示600だけでDB由来・Host Grant・Commitを主張しない。CLI成功からWeb成功を転記しない。Test-only UI instrumentationは本番Artifactへ混入しない |
| **F5 Owner Decision** | Rule算定時点、税・許容誤差、NULL/業務拒否、修正/履歴/現在のAuthorityを確認してMeaning Scopeを受理・保留 | 自動CI/別AIの一致はOwner判断ではない。受理しても現Provider/Hostの許可を与えない |

XL-03（値変更Hook）では同じ業務Subjectに対し別Case revision/IDで、0→1／1→1／NULL→値／二Field変更／rollbackと**read-time・before-write・committed・after-commit effect**を別Expectedにする。変更検出=trueをCommitの証拠にしない。旧X-TEA原文の転載・意味受理を推測しない。実旧Driver互換性は#940/#1129の別証拠が必要。

### 4.1 Required Preconditionsと部分効果の追加反証

[#2616のACU-01〜08](https://github.com/hat47x/tei-platform/pull/2616)の指摘を、F3/F4のCaseに次の二つだけ先行採用する（既存Ownerの実試験IDへ参照、Case正本の二重登録なし）。

- **P-REQ：全必須条件を実チェックしていない**。試行前に必要Checks集合を固定。実行へ渡されたChecksが全てPASSでも、Required中に未指定/未読込/対象違いがあるなら`preflight complete`としない。必要なField・Period・Capability・Grant/Effect等の具体的な集合はOperationごとに固定し、一律に全条件を要求しない。
- **P-PARTIAL：RunPlanはblockedだが先行Stepが作用済み**。StepごとのProvider call/effect/receipt/未知状態を観測し、RunPlan全体のblockedを「一切作用なし」としない。停止直後の無条件再試行を禁止し、Operation ID/Receipt lookup/独立Readで分離。#1130（RunPlan）と#1174（Effect fence）が実行責務。

## 5. 証拠の有効期限：時間・依存・現在資格を分ける

**歴史的Receiptの保存と、現在のClaimを支持できるかは別。** 一律TTLは設けない。結果そのものを後から上書きせず、依存グラフとclaim scopeに応じて選択的にstale/unknownを返す（#2600）。

| 変化イベント | 再資格化すべき主な部分 | 維持可能な部分（条件付き） |
| --- | --- | --- |
| Sourceコメントのみ変更、同World/Operation | Source/Model Revision・SHO引用/Read位置・実消費byte/Artifact生成または再利用資格 | 意味/独立Expectedの数学的内容と非影響な対象の計算結果 |
| Source移動、Import更新・欠落、A→B→A | 負の依存、実Loader/Builder input、Source locator、現在の引用・実Binding | 実行時の古い歴史的Outcome。Source byte一致だけで今のAuthorityを復活させない |
| Expected/Rule/Role/Operator/Plugin版変更 | 該当Case/検証基準 G2、Semantic Scope、Conformance適用、関係するTargetの実評価 | 無関係なCase・既知履歴Receipt（現在合格とはしない） |
| Data/Schema/SQL Asset/Provider/Dialect変更 | 実Read Scope/Query plan/Codec/NULL、DB Snapshot、Conformance/適用対象、実Effect | 意味契約そのものが不変ならOwner受理候補と固定Fixture結果 |
| Host/Artifact/Target/Grant/Policy変更 | 現在Execution/Disclosure、実Binaryと実消費Artifact、principal/session/effect権限 | 以前の受理意味と歴史的局所テスト結果 |
| Suite公開が途中失敗/Readerが旧Memberを取得 | Published Set/Consumed Set、Claimの対象Case集合・Target受領 | 発行前に済んだCandidate局所試験の履歴。#2604/#2516を再利用 |
| Owner受理撤回／用途拡張 | Meaning acceptanceと該当Claimの現在利用資格 | 技術的な歴史的実測が行われた事実 |
| Commit応答喪失／Provider部分結果 | transaction outcome/receiptと別接続Read、再実行安全性 | 実行前の静的Validation/Conformance。未知Effectを否定しない |

**Validityの判定条件**：必要Evidence各々の版・Scope・時点・観測能力が、現在のClaimの要求を同時に満たすこと。期限はclaim依存。固定Fixtureの履歴は保存できても、現在のDB Snapshot/Grantは再Read/再認可が必要になる場合がある。Source A→B→Aのfingerprint復帰やArtifact再利用だけで現在資格を戻さない。

## 6. 横断的な最小受入票と不正昇格の判別

**受入票は一つの表の別行であって、一つの総合真偽値ではない。**

| Claim | 最低必要なWitness | まだ主張しないこと |
| --- | --- | --- |
| Unit-case-passed | 実Source/Tool、Case/Fixture、実Actual/Expected、実行Outcome | 独立検証基準・Business Meaning |
| Conformance-profile-qualified | 独立Expectedの由来/Mutant、Profile/Capability/Scope、必要ケース全実行、観測十分性 | 実DB/Host、本番 |
| Provider-effect-observed | 実Provider/DB/Principal、実Tx/Operation、Receipt、独立Read/Effect、失効チェック | 別Provider・正式Owner Meaning |
| Host-route-observed | 実Binary/Actual Target/Grant、Request→Result、実Consumer body、必要なら実Provider Receipt結合 | 別Target、未観測効果・全Required Coverage |
| Meaning-owner-accepted | 正式Authority、出典Source/Rule版、受理Scope/許容条件・撤回規則 | Runtime健康・現在Execution Grant |
| Product gate | #1119の必要Witness集合が**同一改訂・同一対象Attempt**で成立しOwnerが判定した範囲 | 別版CIの足し合わせ、未受理Domain/将来の保証 |

**必要なnegative control**：①同じCandidateとExpectedを共に誤改変して見かけPASS、②required Case/Checkの未発見やSKIP緑、③Source宣言D1/実Build D2、④Owner受理なしで実値だけ一致、⑤R1/R2混合SuiteをReaderが読む、⑥Commit ACK unknown後に再実行、⑦同表示値でScope/NULL/grain違い、⑧blocked RunPlanの先行Effect残存。この8群を既存#2514/#2595/#1180/#2604/#1174/#1130のFixtureに参照で割当てる。全ケースを新設するのではない。

## 7. 統合先・実装優先・現在の証拠

| 優先度／責務Owner | この基本設計から渡す最小差分 | Doneの判定 |
| --- | --- | --- |
| **P0：#2811/#1180 AI-first** | #2813 DSLと#2828 DriverのCase ID、Required/Discovered/Attempted/PASS/SKIP、Fixed Source・Known Mutant/独立Expectedを実Run票へ記載 | 新JSON追加と独立検証基準判別の実測。既存2 StepのPASSだけではAI新規発見を証明しない |
| **P0：#2514 検証基準** | G2独立性、mutant判別力・観測十分性、Case/Claimの比較可能なExpected provenanceをReport Viewに追加 | 同一起源偽PASSを負例で検出し、G2/unknownを分離 |
| **P1：#1129/#2622 Provider** | XL-01/02→03とDO-04〜07の同一Subject/独立Expected・NULL/partial/Grainを使い、実PostgreSQLと第二DBまたはAPIの差を測る | 実Query/Request・Result/Effect/Scope別にObservedが得られ、未対応はunsupported。既存#2853/#2854/#2857/#2859の実PG成果を複製しない |
| **P1：#1174/#1130 Effect** | required-check completenessとpartial-effect/unknownの二反証。新Key ABAの再適用と同Key Receipt Lookupを分離 | 実操作・実Provider/Grant・Stepごとの観測と安全な停止／照会 |
| **P2：#1163/#1119 Host/Owner** | 同一Accepted Revision→Web/CLI→Provider→独立Consumer、S02/S05/Grant・Owner Scopeを一Attemptで資格化 | 統合PacketとGate Owner判定。#2857の監査がINCOMPLETE/UNQUALIFIEDである間は完了にしない |
| **横断：#2600/#2595/#2604/#2516** | 実消費Input、Source/Meaning/Artifact再利用の失効、一式公開と実受領を個別資格とする | 改版・partial・ABA・未許可Read/開示に対する選択的再資格化 |

**2026-11-06時点で確認可能な限定成果**：
- #2828 test-only DriverのHEAD `77baf709` は [Repository Health #41850080590](https://github.com/hat47x/tei-platform/actions/runs/41850080590) completed/success。#2811既存docs HEAD `3251906b` は [Health #41850309534](https://github.com/hat47x/tei-platform/actions/runs/41850309534) completed/success。両PRはDraftであり本基本設計の追加前HEAD。
- [#2853](https://github.com/hat47x/tei-platform/pull/2853)実PG Command/CAS/Receipt、[#2854](https://github.com/hat47x/tei-platform/pull/2854)ABA別Key適用、[#2857](https://github.com/hat47x/tei-platform/pull/2857)暫定一改訂Web/PG（監査 INCOMPLETE）、[#2859](https://github.com/hat47x/tei-platform/pull/2859)Runner起動前Schema Gate等は**異なる版・個別実走**。同一受理Sourceと現Grantを束縛した一体Web→PG→Consumer合格を主張しない（#1163）。
- 本設計のF0〜F5、P-REQ/P-PARTIAL、台帳Qualification Viewの実装/試行、正式Schema、第二DB/SHO/Windows、Owner受理は**この変更ではNOT RUN／NOT ACCEPTED**。

**採否**：#2514が検証基準の根拠、#1180がScenario、#2622がProvider適合、#1129がData Hook、#1174がEffect、#1163が同一改訂Packet、#1119が横断Gateを引き続き所有する。証拠台帳はその間のread-only比較表であり、一本の万能PASSや新テスト基盤を作る根拠にしない。

## 8. 実証結果を記入した証拠台帳：テストPASSと要求適格を分ける（2026-11-06）

**この章は実際のCI・Go・PostgreSQL試験結果を既存のCase/Claim/Evidence比較票へ転記した例であり、新しいLedger Storeや新実行結果ではない。** GitHub上で観測済みの結果と、未観測の部分を明記する。別HEAD/Environmentの行を足して同一Attemptにしない。

### 8.1 初期Ledger：5件の実例

| 仮ID・既存Case | 固定Environment・証拠 | Test Outcomeと観測事実 | Claim資格／まだ証明しないこと |
| --- | --- | --- | --- |
| **EL-01** [#2828 Driver JSON Candidate](https://github.com/hat47x/tei-platform/pull/2828) | HEAD `77baf7092ff530fb8dee2e53ef4752a8ee01f953`／[Health #41850080590](https://github.com/hat47x/tei-platform/actions/runs/41850080590) completed/success | test-only Go/JSON追加、2固定Validation Anchor、期待値改変・旧Plan拒否を含む限定Fixtureの実Go Test成功 | L1の該当Referenceに限り支持。新しい未知Caseを生成AIが自律発見した証明、独立業務検証基準、Owner受理、実DB/Host資格ではない |
| **EL-02** [#2830 Scenario 6 Step](https://github.com/hat47x/tei-platform/pull/2830) | HEAD `019da011c7d0e1d4ade384c5495c3f91bb12982e`、Go1.26.10/Linux、[Health #41851323500](https://github.com/hat47x/tei-platform/actions/runs/41851323500) completed/success | 6 Step IDの欠落/重複、生成argvの各Step実行、`static-verification`/`generated-pinned`と`evidenceNature`を限定Goテストで照合 | L1のScenario Inventory・Evidence Nature限定適格。`Verified=true`は実Auth/Command/DB/Web E2Eと同じ意味ではない。6 Stepを全業務Coverageの分母にしない |
| **EL-03** [#2870 RCAS-02](https://github.com/hat47x/tei-platform/pull/2870) | HEAD `58d4a0047`、WSL2使い捨てPostgreSQL 14.24、PR記録のGoテスト/実DBログ | **テストPASSは欠陥特性の再現成功**。新Lifecycleでin-memory `unknown`が消失、再Dispatch可能。同じ冪等Keyでは実PGが`IDEMPOTENT_REPLAY`・Applied=0・Receipt1件 | Durable lifecycle/実プロセス再起動安全性は**未成立**。同Keyの限定再送が二重適用を防いだことだけ支持。新Key ABA、Network/実crash、現在Grantは別 |
| **EL-04** [#2882 Workspace/RunPlan](https://github.com/hat47x/tei-platform/pull/2882) | PR HEAD `ee2b37965aaa454be1f594bc049f6f2096241d33`、[Repository Health #42097287333](https://github.com/hat47x/tei-platform/actions/runs/42097287333) `completed/success`、Go1.26/Linux。mainへ統合済み | `runplan_workspace_test.go`で実WorkspaceからPeriod/Effect/CapabilityをLoadし、期間closedでStep未呼出+blocked、openで合成Step resolved+completed、Effect Observer未登録でblockedの有限Go試験が成功 | **L1のWorkspace→Guard→RunPlan限定適格**。Callerが必要Guard全件を供給した証明、実Provider Commit、Observerの実効果、別Stepの先行Effect、本番Host・Current Grantではない |
| **EL-05** [#1163 一改訂Product Gate](https://github.com/hat47x/tei-platform/issues/1163) | #2853実PG Command/Receipt、#2857暫定Web/PG、#2859起動前Schema Gateは**別HEAD/別Attempt** | それぞれの限定的な実PG/実Web/Schemaの成功が存在。#2857の横断監査はINCOMPLETE | **Product Gate=UNQUALIFIED**。同一Accepted Revision・現Grant・実Web→実PG Commit→独立Consumer受領という一経路の統合Receiptを合成できない。Owner受理なし |
| **EL-06** [#2890 AI追加Scenario](https://github.com/hat47x/tei-platform/pull/2890) | #2828基底HEAD `77baf709`、候補HEAD `8e508413fe92cfc95a6176e34f8af8b35fb576ae`。Scenario blob `251b5282`、正常Fixture `bc65e0d7`、重複Fixture `7e9a04e5`（GitHub実Read） | **L0静的入力対照は再計算済み**：両Fixtureはnumbers（quantity=`2.5`、line_number=`1`）、relation/attributeが一致し、Relation keysだけ`[1,2]→[1,1]`。候補Expectedに依存せず `count(keys)==count(distinct keys)` で正常true・重複false、Expected-only反転を拒否することを本照合で確認。追加2 Step・2 JSONのみ、既存Go/Anchor変更なし | **Authored + finite input discrimination observed**。Go/CLIの`Discovered/Attempted/Receipt`・新HEAD Actionsは未観測。数学的なKey重複判定はScope限定の外部照合で、独立した**業務検証基準/Owner Meaning Acceptance**とは別。G2・業務Coverage・実Provider/HostはUNQUALIFIED |

<!-- EL-06-static-witness-20261106 -->
**EL-06の二段階判定**：今回のread-only静的照合はScenarioの`expectValidations`を計算の入力にせず、Fixtureキー配列の集合上の重複という性質だけで正負を比較した。これは`[1,2]`と`[1,1]`の有限対照の**入力判別性**であり、TEI側Canonical Ruleの実装正しさ・業務上の「同一明細番号」のScope・Owner受理・現TargetへのBindingは未確認。次回の実Go受入では`TestAIAddedScenarioCandidateFiles`を追加Scenario込みのexact HEADで走らせ、両Step IDと`generated-pinned` Receiptを検出し、さらにRuleを意図的に誤らせた既知Mutantに独立Expectedを適用する。**候補が期待値を自分で変更してPASSする場合はG2不成立**。#2890はstacked PRであり、この時点のHEADからのActions Workflow Runは照会上0件。基底#2828のgreenを新Candidateの成功に流用しない。

| **EL-07** [#2891 Fixture JSON重複Member拒否](https://github.com/hat47x/tei-platform/pull/2891) | 旧HEAD `2f677921` [Health #42107910706](https://github.com/hat47x/tei-platform/actions/runs/42107910706) はgofmt driftで全体FAIL（同run Go全体はPASS）。整形のみ修正したHEAD `16b8d5548fe916452c3e9cae8f1629d25f91d3bf` の [Health #42110507652](https://github.com/hat47x/tei-platform/actions/runs/42110507652)、セルフビルド、依存チェックは**いずれもcompleted/success** | 既存ScannerをValidation/Browse/Observation Fixture Parse入口へ追加。オブジェクトMember重複7件の明示拒否／正常JSON3件がGo全体テストに含まれ、健全性CI成功 | **G0限定修正がexact HEADで確認済み**。#2890のキー配列値重複は別のValidation対象。#2891単体の成功から#2895 Scenario/検証基準実走、業務Meaning、全Source closure/実PG/Ownerは主張できない |

| **EL-08** [#2895 有限独立検証基準と積層CI](https://github.com/hat47x/tei-platform/pull/2895) | #2828→#2890→#2895。最新HEAD `8d3dc2fc5058ff3945d0981d2979138d4fc461be`、[Repository Health #42109027966](https://github.com/hat47x/tei-platform/actions/runs/42109027966) **completed/success**（Go 1.26.10/Linux amd64、`tei/cmd/tei`含む全Go Test / `gofmt` / `go vet` / `tei validate` PASS）。Web・Studio・依存も同HEADでCI成功。#2895から#2890への差分はGo test-only 1ファイル | Goテストは2 Step ID/Fixture/keyに**候補Expectedとは別の有限集合Property**を固定し、Expected-only反転・Input key-only反転を同じ判定関数で拒否。既存`TestAIAddedScenarioCandidateFiles`は2 Candidate JSONをglobで検出し、`verify/generate/check/generated-pinned run`の各Step Receiptを検査する構成。CI全体Go PASSを確認したが、標準CIログに個別Step/Receipt JSONは独立保存されていない | **L1局所Scenario+固定検証基準テストの実行成功**へ更新。ただし`Test...`の実行結果とSourceコードの内部Assertionを根拠とする限定資格。実Product Ruleを誤変異させたMutant kill、G2業務Expectedの独立受理、全Obligation母集団、実PG/Host/Ownerは未証明。#2895は`main`向けの**検証用stacked Draft／単独merge禁止** |

| **EL-09** [main統合 #2894 `tei dbcheck`](https://github.com/hat47x/tei-platform/pull/2894) | head `556d8e6a723daabed399c35e987fb1e934cc1e3c`、[Repository Health #42108388836](https://github.com/hat47x/tei-platform/actions/runs/42108388836) **completed/success**。現main `cmd/tei/dbcheck.go`/`dbcheck_test.go`を確認 | DSN欠落なら実接続せず`errDSNMissing`/CLI exit 2を区別。指定DSNでは`pgx.Connect`→`SELECT version()`→同Sessionで一時表CREATE/DROPを試行する実装。Go testはDSN欠落・flag優先・サーバー版表示整形を検査 | **L3のEnvironment/Capability事前入場**に限る。接続先DSN未提供の状態は`NOT_ATTEMPTED`でありDBテストPASSではない。実DBに接続できる場合もtemp DDL権限は業務TableのRead/Write、RLS、Snapshot・Schema適合、Commit/Receipt・Owner/Host資格を証明しない。CI成功のみではこのHEADで実DSNを使ったPostgreSQL接続が成功したとは主張しない |

**証拠の読み方**：EL-03は`test=PASS`でも`durable-state safety=UNQUALIFIED`。EL-04は`test code present`だが今回の実Run証拠は`unobserved`。EL-05は個別のPASSが複数でも`one-revision integration=UNQUALIFIED`。この三種を区別できない台帳やCIレポートを受入済みとしない。

### 8.2 三つの結果を混ぜない最小Claim判定

- **Test outcome**：実際にそのExact HEAD/Case/Environmentで何を実行し、PASS/FAIL/SKIP/NOT RUNとなったか。Required Case/Checkの欠落と単純なGo process exit=0を分ける（#2835）。
- **Characterized behavior**：テストが検出した動作は正しい仕様への適合か、それとも**意図どおり再現できた不適合**か。Known Mutant/negative controlの存在をExpectedの出典と共に記録する（#2514）。
- **Requirement verdict**：Owner受理済みMeaning/Obligation、Scope、Snapshot、現Grant、Provider/Target/Reader資格に照らして現在何を主張できるか。Execution PASSをOwner Acceptance・Current Execution Authorityへ昇格させない（#1119/#1163）。

**重要な対照**：#2882の「期間閉じたWorkspaceでは最初のStepを呼ばずblocked」は、*実行前に拒否されたそのCase*のEffectゼロを支持し得る。一方、#2616/#1130の「途中Stepが実行された後のblocked」は全PlanのEffectゼロを支持しない。Evidence台帳では`phase`と`step-level effect witness`を必ず別列にする。

### 8.3 P0で実際に埋める最小受入票

| Case群 | 事前固定のIndependent Expected／Required集合 | 実観測で受理するもの、欠けたら止めるもの |
| --- | --- | --- |
| **E-01 Scenario inventory** | #2830保存済6 Stepと各ID・Expected Evidence Nature | 実Glob/Parse/生成argv/対象StepごとのReceiptを対応。欠落/SKIP/未実行を検出。6 Step外の未知Obligationを100%へ丸めない |
| **E-02 AI追加ケース** | #2828固定Anchorを保持し、[#2890](https://github.com/hat47x/tei-platform/pull/2890) 2 Step（`ai-line-number-unique`/`ai-line-number-duplicate`）を事前固定。[#2895](https://github.com/hat47x/tei-platform/pull/2895)がCandidate期待値とは別に定めた公開有限集合Propertyと反転負例を検証基準側の限定根拠にする。Fixture JSONの重複member拒否は[#2891](https://github.com/hat47x/tei-platform/pull/2891)の別Draftで検証 | #2890の**JSON-only authored**、#2895の**独立テストコード書込み・隔離Go実走**、#2828の**既存Baseline実Go PASS**、新HEADの**Repo全体CI/Scenario実Run/Receipt**を別々に観測。#2895は`main`向け検証用stacked Draftで単独mergeしない。業務Meaningの独立G2・全Coverage・Provider/Ownerは未受理 |
| **E-03 必須Preconditions** | Operationごとに**独立した要求元**（Accepted Meaning/Contract・実Workspace/Capability・Host Targetの適用規約）からPeriod/Effect/Grant/Observer/SchemaのRequired集合を固定。Candidate自身や`CombinePreconditions`へ渡されたSupplied集合から自己生成しない | `Required → Supplied → Evaluated → Enforced-at-Effect`を同じOperation ID/Revisionで比較。Requiredが1件欠けていてSupplied全PASSでもCoverage不適格。実Commit時強制が未観測ならread-onlyの入場確認以上に昇格させない（#2884/#2860）。適用外Checkは不要 |
| **E-04 再起動と副作用** | RCAS-02同Key再送、#2854新Key ABA、#1174 Commit Unknown/Receipt Lookupを独立Expectedとして事前指定 | Attempt/Operation identity、Step実Effect、PG Receipt、別接続Readを結ぶ。Blocked/Unknownの無条件再試行を認めない。実Process crash/Networkは未実行なら明示 |
| **E-05 SQL/Provider選択** | #2622/#2862の同Query ID・異Realization/SQL選択、Read Scope・NULL・Grain・mapping資格 | Query IDや表示Rows一致は実SQL/Providerの同値を証明しない。固定SQL不在時の無音fallbackを拒否。第二DB実測は別 |
| **E-06 一改訂E2E** | #1163のAccepted Source Revision・実Binary・Current Grant・実PG・Receipt・独立Read/Consumer実受領 | #2853/#2857/#2859の別Attemptを合算しない。必要Edgeの欠落を個別unknownとして保持し、#1119 G1/G2へ誤昇格しない |

**表示への参加資格（横断補足）**：実行AttemptのReceiptが正しくても、別Workspace・Selection Epoch・Source Revisionで遅れて届いた非同期応答は、現在のStudio/Web画面への表示・操作許可にならない（#2616 §20、#2884 §7）。`execution-join`（元Request→実Provider）と`presentation-join`（元Context/Epoch→現在Consumer）、現在Disclosure Grantを別列に置く。歴史的な実行証拠は保存し、古い結果を新しい画面Stateへ黙って適用しない。実React/Windowsと後着表示の本変更での再実行はNOT RUN。

**次の最小出口条件**：E-01とE-02の実Case ID/Required集合/Actual Runを一枚で照合し、E-03またはE-04の既知負例で「テスト自体PASS、製品Requirement不適合」を少なくとも一件説明できること。E-05/E-06の実Provider/Hostは本P0のHard Blockerにせず、既存Ownerが個別Evidenceを取得した後に段階強化する。

**更新は設計・証拠のread-only整理のみ。** E-01〜E-06の新たな実行、AI自律生成、独立Mutant試験、実Process再起動、二DB・完全Host/Provider E2E、正式Owner受理は本変更でNOT RUN。新Ledger DB、Runner、Acceptance Engine、Ownerの正本を追加しない。

### 8.4 Fixture重複キーと検証基準自己追認を区別する

[Draft #2891](https://github.com/hat47x/tei-platform/pull/2891)では、`ParseValidationFixture`／`ParseBrowseFixture`／`ParseObservationFixture`がJSON duplicate memberをDecode前に拒否していなかった**実ソース上の不整合**を、既存`RejectDuplicateJSONMembers`の再利用で解消する候補実装を作った。ソースの静的確認と変更ファイル数（4）は確認済み。Go/CI実走の結果は別。

| 対照 | 事前独立に固定する負例・期待 | 実行資格・止めるClaim |
| --- | --- | --- |
| **FIX-01** Validation | 同じ`numbers.sales.order_line.quantity`を2.5と0で重複宣言／根`numbers`重複／Relation`keys`重複を**すべて明示拒否** | Go testの実Errorと`duplicate JSON member`確認まではCandidate。後勝ちで意図しないActual→Expected成立を許さない |
| **FIX-02** Browse | `query`を二重宣言、Row中の`kind`を二重宣言しても明示拒否 | Query/Provider bindingと実Readが正しいことは別。Fixtureの構文受理はProvider Conformance資格ではない |
| **FIX-03** Observation | `readIdentity`二重宣言、`observed.data`二重宣言を明示拒否 | 合成Observationのみ。Provider Commit/独立Readや本番Effectを証明しない |
| **FIX-04** 正常対照 | Validation/Browse/Observation各1件の**重複キー無し**JSONを受理 | 過剰拒否がないことの有限テスト。一般JSONスキーマ完全性・Owner Meaningは別 |

**#2890との関係**：新AI候補のLine Number重複は**Relationの`keys`配列値`[1,1]`**であり、JSONオブジェクトの**同名member重複とは別の対象**。FIX-01〜04がPASSしても#2890のValidation Ruleの正しさ・独立検証基準判別力・実Go Scenario receiptは認定しない。逆に#2890のCandidate実Go PASSでもFIX-01の後勝ち問題がなくなった証拠にはしない。

**受入順**：#2891の固定HEAD `go test ./internal/scenariotest`／`go test ./cmd/tei`／正規CIを確認してG0入力曖昧性を閉じる。#2890のJSON-only追加は別PR/別HEADで既存Goテストの候補発見と実Step Receiptを確認する。両者を一つのAttemptに混合しない。#2514のG2は独立検証基準とKnown Mutantが実際に誤りを検出して初めて判定する。**本追補はdocs-only・新テスト実行なし**。


### 8.4 L3の環境入場を実Provider実測から切り分ける

main統合の[`tei dbcheck`](https://github.com/hat47x/tei-platform/pull/2894)は実PostgreSQL試験を開始できるかを確認する**準備段階の操作**。これを新しいUnit/L3判定階層にせず、前述のL3 Caseに付属するEnvironment Capability Witnessとして記録する。

| 状態 | 取得できるEvidence | 拒否する誤昇格 |
| --- | --- | --- |
| **DSN欠落** | CLI exit 2、未接続／未試行という事実 | `go test`終了0や`dbcheck`未実施を実PG PASSへ転記しない |
| **接続・一時DDL成功** | 指定alias/Principal/Server版・実接続・Session一時表の作成/破棄（実行時のreceiptを確保できた場合） | 業務テーブルのRLS・Schema・更新権限・CAS・Commitや長期のDB可用性を主張しない |
| **実PG Case成功** | 固定Source/SQL/Realization・Snapshot、実Read/Write、Receipt、独立Read、Required CaseのPASS一覧 | 本番Owner/Host/別DB資格を主張しない。環境probeとは別Attempt・別Claimとして記録 |

`dbcheck`が行う一時表CREATE/DROPはread-onlyではなく、対象Session内の**テスト準備の実DB作用**である。DSNの生値・資格情報を公開Ledgerへ転載しない。成功の期限は実Connection時点とPrivilege/Schema/Grant改版に依存するため、以前のprobe PASSを再実行許可として使わない。**今回の文書反映では`dbcheck`の実DB接続はNOT RUN**。#2622/#1174と#1163がProvider/Effect/一改訂Product Gateの実証を引き続き所有する。

### 8.5 実行済み統合Stackと未結合のG0修正を区別する（2026-11-06）

[#2895](https://github.com/hat47x/tei-platform/pull/2895)のHEAD `8d3dc2fc5058ff3945d0981d2979138d4fc461be` は #2828→#2890→#2895 を含む**同一実Go checkout**であり、[Repository Health #42109027966](https://github.com/hat47x/tei-platform/actions/runs/42109027966)のjob logでは `go test -count=1 -timeout 60m ./...`（`tei/cmd/tei` 8.916秒）、`go vet ./...`、`gofmt -l .`、Workspace validation のPASSを確認。コードから、既存`TestAIAddedScenarioCandidateFiles`が新しいJSONをglobで発見し、全Commandごとに生成Pinの`runScenarioCommand`を実行し、`verified=true`かつ限定`evidenceNature`を検査することを確認できる。ただしCIログは単体Stepごとの実Receipt bodyを公開しておらず、保存済みの外部実行Receiptではない。

| 検証段階 | 根拠の帰属／現在資格 | 未成立 |
| --- | --- | --- |
| **G0 Scenario/Plan Parse/Fixture input** | #2895 StackでScenario/Planの既存読込が限定Go PASS。[#2891](https://github.com/hat47x/tei-platform/pull/2891)のFixture object duplicate-member禁止は**別HEAD**（旧HEAD `2f677921`のGo全体PASS/総合CIはgofmt FAIL、書式修正 `16b8d554`のCI再試行中） | 同一HEADで3 Fixture Parser重複拒否＋新AI Candidateを同時検証したG0資格 |
| **G1 Scenario実行** | #2895 Stack HEADの全Go test PASSと実Goテストソースの照合から、新2 Stepの既存CLI機能をtest process内で検査した限定Evidence | 外部CLI binary/完全なSource Loaded閉包、Step Receiptの独立アーカイブ、全業務Obligation |
| **G2 独立性・判別力** | #2895新GoテストはCandidateのExpect値と独立な有限重複集合Propertyで正常/重複の2 Case、Expected-only/Input-only反転の**入力・期待値変異**を実検査。CIもPASS | 実TEI製品Validation Rule本体を変異させて検出したMutant kill、Ownerに受理された業務意味、他Rule/NULL・部分Read・重複Scope全体 |
| **G3 実Provider/Host** | none、not required for local synthetic Reference | 実DB/Effect/Commit/Host Grant/Consumer Body |
| **G4 正式Coverage/受理** | none。明示採用したRequired Obligation集計と独立Owner判断は未実施 | Product Gate / Business Meaning Acceptance |

**次の最小統合判定**：#2891の新HEADでgofmt/全Go/HealthのPASSを取得してから、#2895の固定Stackと#2891のFixture G0修正を同一実HEADへ取り込み（新Runnerを設けず既存Go Sourceを再利用）、*必要Case IDと三種Fixture Parserの負例がすべて実行された*ことをCIに照合する。その後#2514の独立した既知**Product Rule Mutant**を最小1個だけ追加し、有限Property対照と実Rule誤実装の検出を混同しない。

**今回の新実行**：この文書追補自体ではGo/PG/Host/Owner試験を行っていない。Source/CI照合と#2891のGoテスト整形修正は別PRで実施。#2895は検証用stacked Draftのままとし単独main mergeは禁止。

**他レーンに共通する曖昧な入力の受理拒否（比較規則のみ）**：[#2891](https://github.com/hat47x/tei-platform/pull/2891)の同名JSON object member、[#2892](https://github.com/hat47x/tei-platform/issues/2892)のMigration History `AppliedUnknown`／取得不能・部分History、[#2887](https://github.com/hat47x/tei-platform/pull/2887)の同DatabaseID/Contract重複や必要Client母集団欠落は、それぞれ**異なる入力・異なるOwner**である。共通する資格規則は「同名重複を後勝ちで確定しない」「unknown/未取得/partialを成功や真の空集合へ変えない」「必要母集団の外部出典を保持する」。#2892は2026-11-06時点で静的コード照合と独立Expectedの**未実行Issue**であり、Migration修正済みとはしない。#2891 Parser、#2892 History、#2887 Release Planの既存Ownerに各反例を引き継ぎ、万能Validatorや第二Evidence Engineは導入しない。

### 8.6 G0・Scenario・有限検証基準を一改訂で試す隔離PR（2026-11-06）

**[#2901](https://github.com/hat47x/tei-platform/pull/2901) をDraftの統合検証用PRとして新規作成。** #2895の実CI成功HEAD `8d3dc2fc5058ff3945d0981d2979138d4fc461be` を起点に、#2891の整形済み実装4ファイルを**byte同一で複写**。#2895との差分は3 Parserの3行Guard、Goテスト1ファイル（7 negative＋3 positive）のみ。新Runner/Provider/業務Rule・検証基準改訂・Scenario JSON改変なし。

| 証拠項目 | 既に確定した個別Evidence | #2901同一HEADで必要な追加証拠 |
| --- | --- | --- |
| **G0 Input Admission** | #2891 HEAD `16b8d554` [Health #42110507652](https://github.com/hat47x/tei-platform/actions/runs/42110507652) success、7+3件のGo回帰 | #2901で3 Parser GuardとAI Scenario fixture読みを同一ソース/Go版で実行、Required Case実在と非SKIP |
| **G1 AI Scenario execution** | #2895 Stack HEAD `8d3dc2fc` [Health #42109027966](https://github.com/hat47x/tei-platform/actions/runs/42109027966) success、既存test loopがcandidate glob・generated-pinned stepsを検査 | 同一HEADのGo Testが新正負2 Stepを実行し、限定`canonical-validations-synthetic-input-evaluated`と実行結果を識別。CI通常ログは個々のReceipt JSONを保存しない |
| **G2 finite independent check** | #2895のCandidate Expectedとは独立した集合重複検査、Expected-only/Input-only反転拒否がGoで実行 | 同一HEADで有限PropertyとCandidate比較を実行。**実Product Ruleの誤変異を殺すMutantではなく、正式業務検証基準も未受理** |
| **L3〜L5** | 実PGの別HEAD限定証拠とOwner未受理の記録を保持 | 本統合PRでは追加しない。単純な全Go緑・Web smokeを実Provider/Grant/Owner受理へ誤昇格しない |

**現在資格**：#2901 HEAD `a6eb80cffe16a57f4cb2461ff2946065955a4e50`で4ファイル差分をGitHub再取得・比較済み。main向け**検証用Draft／単独マージ不可**。今回の最初の照会では[Repository Health #42111973682](https://github.com/hat47x/tei-platform/actions/runs/42111973682)等がqueued。**この段階では統合Go PASSとは呼ばない**。#2891/2895の既存別HEAD CI成功は、本統合HEADの代替には使えない。

**次の停止線**：#2901 exact-headのGo全体・gofmt/vet/Workspace・必須Test ID確認→#2514での実Product Rule Mutant 1件の独立殺傷評価→必要になった場合にのみ#1180採用Suite/Reader、#1163一改訂Product Gateへ進む。Script/DB MigrationのRequired/Partial/Unknownについては#2884/#2892の既存OwnerがSourceと実Observationを固定して判定し、Fixture G0成功をそこへ一般化しない。**本書はread-only台帳/計画であり、テスト実行は別PR CIによる**。

### 8.7 実Product Evaluator Mutantの独立判別（#3017）

**結合Greenを確定**：[#2901](https://github.com/hat47x/tei-platform/pull/2901) の統合実装HEAD `a6eb80cffe16a57f4cb2461ff2946065955a4e50`で[Repository Health #42111973682](https://github.com/hat47x/tei-platform/actions/runs/42111973682)、セルフビルド・Web E1-A・Studio・依存検査が全て**completed/success**。#2828／#2890／#2895のAI Case・有限独立Key 検証基準と#2891のFixture重複member G0が**同一Go Checkoutで回帰成功**したことを示す。実Readerの個別Raw Receipt、意味受理、実Provider/Host業務E2E、全業務Coverageは別主張。

#### G2候補の次の実証：Green基線と本物のProduct Rule誤実装を対照する

**隔離Draft [#3017](https://github.com/hat47x/tei-platform/pull/3017)** は#2901成功HEADから分岐し、`internal/semantic/validationeval/evaluator.go`の`semantic.ValidationRelationUnique`で重複キーを見つけたときの結果を**`not-satisfied → satisfied`へ意図的に一行誤変更**する。#2901との差分は製品Evaluatorの1ファイルのみ。Scenario/Fixture/独立Expected/テスト本体は変更しない。**このMutantは絶対mainへマージしない。**

| 対照 | 固定するもの | 実際に観測すべき事実／分類 |
| --- | --- | --- |
| **M0 正常Green** | #2901 exact HEAD、`ai-line-number-unique`の`[1,2]`/satisfied、`ai-line-number-duplicate`の`[1,1]`/not-satisfied、#2895独立有限Expected | 正規Health成功、必要なGoケースが実行された事実。これは**基線の履歴**であり現在のProduct G2証明ではない |
| **M1 誤実装Red** | #3017 exact HEAD、Evaluator重複時だけ`ResultSatisfied`へ変える。その他のInput/Expected/Plugin/Case/Go testは不変 | `TestAIAddedScenarioCandidateFiles/ai-candidate-line-number-uniqueness`等の**対象Caseに到達した**fail と `result=satisfied, expected=not-satisfied`を実Job logで確認した場合、同一業務局所CaseによるProduct Evaluator mutant kill を限定認定 |
| **M2 他TestのみRed** | M1と同じ誤実装 | 既存Evaluator別テストだけがFailし、AI追加Caseが走らない/到達しなければ**AI CaseのMutation感度はunknown**。総合CI赤をAI品質に誤転記しない |
| **M3 起動・環境のみRed** | M1と同じ誤実装 | gofmt/Build/依存/Runner/環境エラーだけなら**mutant killではなくinconclusive**。ExpectedをMutantのGotへ修正しない |
| **M4 Survived** | M1と同じ誤実装 | 完全な必要Caseの実走なのに全PASSなら**対象Mutantの見逃し**。Target/Rule/Loaded Source/Read Scopeの欠落を#2514へ返す |

**証拠台帳の最低列**：`baselineHead / mutantHead / exact source diff / selected Canonical Rule+Revision / fixed Scenario+Fixture SHA / independent Expected provenance / test name+actual execution / observed outcome+diagnostic / mutant classification / surviving or missing obligations / expiry`。AgentによるExpected書換え、別HEAD混在、失敗したCIを正常なHealth PASSと言い換えることを禁止。

**現時点の資格**：本追補時点では#3017の[Repository Health #42181290981](https://github.com/hat47x/tei-platform/actions/runs/42181290981)が**in_progress**であり、意図的な不合格が当該AI Caseに到達したかは**未確認**。既存#2901のGreenを#3017へ転用せず、M1/M2/M3/M4を実ログで区分するまで`Product Rule mutant kill = NOT VERIFIED`。Mutationはテスト専用ブランチの実コード変更であり、原Canonical SourceのOwner受理や本番Runtime変更ではない。

#### M1 実結果：AI追加ScenarioがProduct Evaluator Mutantを検出（2026-11-06）

- **baseline**：[Draft #2901](https://github.com/hat47x/tei-platform/pull/2901) HEAD `a6eb80cffe16a57f4cb2461ff2946065955a4e50`、[Health #42111973682](https://github.com/hat47x/tei-platform/actions/runs/42111973682) **completed/success**。
- **mutant**：[絶対merge禁止Draft #3017](https://github.com/hat47x/tei-platform/pull/3017) HEAD `0f795a012dc16200ee85e5f6b3f0b921add7bdc4`、[Health #42181290981](https://github.com/hat47x/tei-platform/actions/runs/42181290981) **completed/failure**。`validationeval/evaluator.go`の重複キーの場合`ResultNotSatisfied`を`ResultSatisfied`へ誤変更した**一行のみ**が差分。Scenario JSON、正常/重複Fixture、AIケースGo、有限独立検証基準テストのBlob SHAがbaselineと等しいことを確認。
- **実失敗の帰属**：job `127771281587`の実ログ577〜579行で`TestAIAddedScenarioCandidateFiles/ai-candidate-line-number-uniqueness`がFAILし、Step `ai-line-number-duplicate`の`Canonical Validation sales.order.lines.line_number_unique result=satisfied, expected=not-satisfied`を出力。ほかのEvaluator回帰もFailしたが、**AI追加Scenario自身が対象Mutantに到達し誤りを検出した**ことを区別して確認した。
- **失敗の種類**：同一jobは`gofmt`・`go vet` PASS、`go test ./...` FAIL。M3（インフラ・構文・書式だけの偽Red）ではなく、**M1 product evaluator mutant killed by added AI Scenario** と限定分類。通常の総合Healthを成功と表現しない。
- **資格の上限**：公開合成`relation-unique`で`[1,1]`を正しく`not-satisfied`と期待した**1 mutant/1 target rule/2 fixed cases**の限定判別力。正式業務Owner Meaning、Source閉包/Rule改版、他Rule・欠損/NULL・複数粒度、実Provider/Host/現在Grant、採用Suite全Obligationは未証明。#3017は意図的な不良コードを含むので本番・mainへ絶対統合しない。

**次の出口**：#2514で独立業務Ruleの出典・版と追加Mutation対象（NULL、未取得、Relation scopeの誤り）を固定し、実CaseのRequired集合・Consumer実Receiptを#1180/#1163に渡す。今回の有限M1を全G2/正式Owner判断に昇格しない。新Runner/判定基準専用DB・Acceptance Authorityは導入しない。

### 8.8 次の検証基準感度とOwner対象集合証拠の二層化（2026-11-06）

[#2901](https://github.com/hat47x/tei-platform/pull/2901)の同一HEAD `a6eb80cffe16a57f4cb2461ff2946065955a4e50`では[Health #42111973682](https://github.com/hat47x/tei-platform/actions/runs/42111973682)がsuccess。[#3017](https://github.com/hat47x/tei-platform/pull/3017) `0f795a012dc16200ee85e5f6b3f0b921add7bdc4`では[Health #42181290981](https://github.com/hat47x/tei-platform/actions/runs/42181290981)がfailure、実ログでAI追加の`ai-line-number-duplicate`が製品Evaluatorの重複判定誤りを検出した。この**履歴として確定したM1**と以下の**未実行の次提案**を区別する。

**次の1 Mutant（設計提案・NOT RUN）**：既存`internal/semantic/validationeval/evaluator.go`の`ValidationRelationUnique`において、重複結果の分岐は正しいまま、**最後のRelation Keyだけを反復対象から落とす**誤実装を別の隔離HEADで試す。変更対象はEvaluatorのループ上限1箇所のみ。#3017の戻り値反転を再利用したり、Fixture/Scenario/Expectedを修正して見かけPASSにしない。現行2要素の正負対照は、正常`[1,2]→satisfied`、重複`[1,1]→not-satisfied`。末尾を黙って除外すれば前者は変わらず、後者を誤って`satisfied`とし得る。**この予測は実測ではない**。

**実験の固定ピン（既存Source/Fixture/検証基準）**：
- Source：[#2901](https://github.com/hat47x/tei-platform/pull/2901) HEAD、`reference/workspace/semantic/sales/validation.tei` blob `fd9e3ad6a81e3201ade8e987377137a3ba702d5f`、`reference/workspace/applications/order-entry/application.tei` blob `00c29ac0e557cb50fdcd13f2477d0b538a528c80`。Rule選択は`order_entry`の`uses validation`が正本。関数の独立ExpectedとProduct Evaluatorを同根の計算で再生成しない。
- Fixture：`fixtures/order_valid.json` blob `bc65e0d76ce09ee96c5f32e20050a30ed1d1a37e`、`fixtures/order_duplicate_line_numbers.json` blob `7e9a04e5afbcfbfb9a857c4f4b50313d6d15429c`、Scenario `ai-candidate-line-number-uniqueness.json` blob `251b5282f6730bc7eb7f383d118bee5662389d77`。
- 検証基準：`cmd/tei/ai_line_number_independent_oracle_test.go` blob `e1c30b0fef60547f8d4f9046a57df5ac1050ff80`。実際に使用されたWorld Fingerprint、Plugin Lock、Artifact/Toolchain、Go Jobの実Attemptは**実行時のReceiptから別途取得**する。ソースのGit blob一致だけで実Loaded一式の同一性を保証しない。

| 観測する結果 | 結論の上限 | 禁止する推論 |
| --- | --- | --- |
| **正常Green→末尾脱落Mutantで追加AI StepがValidation不一致を出す** | 合成`RelationKeys`の**与えられた2要素**の末尾を欠落させる誤りに対する当該Go Caseの局所Mutant kill | Ownerの必要明細集合を取得できていること、実DBで全行を取得したこと |
| **他のGo TestのみRed** | 既存試験が誤りを検出した | AI追加Caseが検出したとの認定 |
| **必要Step実行済みでSurvive** | 対象CaseのMutation感度に不足がある。実Used/Loaded/選択Rule・Observableを調べる | Mutationの安全性、Owner意味の正しさ |
| **Build/gofmt/Run欠落/Pin不一致のみ** | inconclusive（観測不能） | CIの赤だけでProduct Mutant killとすること |

**別の反例：完全性がないと実装が正しくても見かけPASS**。現在の`scenarioValidationSource.RelationKeys`は`[]string, error`を返し、Evaluatorは**渡された配列内**の重複だけを調べる。業務Ownerの必要母集団が2行`[1,1]`でも、Provider/呼出し境界が1行`[1]`を返せばEvaluatorは`satisfied`となり得る。この場合はEvaluatorが全入力要素を走査していても、**Readの完全性が未立証**であり、上の末尾Mutant killでは防げない。現在のSource IFにはScope、Snapshot、ページ終端、RLS/権限の非開示、総件数、Ownerが要求した母集団の証明は存在しない。取得済み0件・未取得・部分取得・非開示を同じ空配列へ変換しない。Fixture内で`keys: []`が正常に解釈されても「Owner対象が0件」を証明しない。

**NULL／unknownは未対応として分離**：現行Fixtureは数値の文字列とRelationの`[]string`しか持たない。NULL Key、未知Key、Provider取得失敗、非開示を単なる`"null"`や空文字などの番兵値へ変換して検証基準を自己追認しない。独立Expectedの定義が必要な場合は#2514のMeaning Ownerで、どの状態が`not-satisfied`／評価不能error／unknownであるべきかを先に区別し、その後#1180/#2586へFixture/実Readの要件を渡す。本節は新しいFixture Schemaを採用しない。

**Source変更時の再資格**：Source内容・`uses validation`選択・Relationの意味/粒度・Operator版・Owner受理が変われば**現行Expectedの適用可能性を再確認**する。Path移動（将来`.tei`から`.tei.md`への移行を含む）は、byte同一なら自動的にMeaning同一とせず、実Source Reference再解決とModel/World、実Compiler/Plugin入力、Acceptance根拠を別々に照合する。失効した過去のGreen/Red Evidenceを削除・上書きせず、`historical valid at HEAD / current unqualified pending recheck`を併記。Source改版は#2595、再資格化は#2600、Read/母集団は#2586/#1163が所有し、#2514の独立検証基準担当Chatと実装・投稿が競合しないよう、本書は**受け渡すべき証拠・Claim上限のみ**を記述する。

**進捗の区分**：#2901 Greenと#3017 M1 Killは実CI確定。末尾脱落Mutation、NULL/unknown、Provider部分取得、`.tei.md`移行後の実Resolved Sourceは**いずれもNOT RUN**。本節の追補はdocs-only。故意に不正なMutantのmain統合は厳禁。

### 8.9 Completionの自己申告Coverageと実Read母集団を別検証基準で照合する（2026-11-07）

**検討時の実Source照合**：旧mainの`internal/semantic/completion.go`（旧blob `512a835ca445d81dd043f8af78ca29df4e7adb3b`、#2926で導入済み。#3051統合後の現main Blobは`e42610e9159e02ba3b5c7b84c46742a846331e64`）では`EvaluateCompletion`が`CompletionEvidence.Targets`、`Coverage`、`EmptyJustified`、`Conflicting`等を受け取る。`CoverageComplete`は**呼び出し側が供給する主張**で、関数自身がOwnerのTargetSet、Providerの実Read、Snapshot、Access/Disclosureを求めて証明するわけではない。対象が非空で全`Holds=true`かつ`CoverageComplete`であれば`CompletionHolds`を返す実装。これは**契約上の明示的な信頼境界**であり、この観察だけで製品バグ／業務意味受理済みとは判定しない。

**二世界の識別不能反例（評価予測、TEI新試験はNOT RUN）**：同一呼出しデータ`Targets=[{L1,true}], Coverage=complete`について、世界AではOwnerの対象集合`L={L1}`、世界Bでは`L={L1,L2}`かつL2が未取得・不成立であっても、Completion APIへ渡す引数が同じなら`holds`となる。この判定の真偽を分ける証拠は`Coverage`文字列ではなく、**Owner集合の版とReaderが観測した対象Identity・範囲・時点・終端**にある。従って「実装が申告データに従い評価した」という局所PASSを「Ownerの完了を確認した」に昇格しない。

| ケースID | 外部から固定する独立条件 | 現行関数の局所挙動／必要な上位判定 | 証拠資格 |
| --- | --- | --- | --- |
| **CC-01 complete true** | Owner L={L1,L2}、同一Scope/SnapshotからL1,L2を実読取、P(L1)=P(L2)=true | `Targets=[L1:true,L2:true], CoverageComplete`なら`holds`。真の全件性は別Reader/Provider証拠と一致して初めて完了Claimに適用 | Source静的／テスト計画。新E2E NOT RUN |
| **CC-02 partial positive** | Owner L={L1,L2}、実読取はL1:trueのみ、L2は未取得 | `CoveragePartial`なら`unverified`。誤って`CoverageComplete`を渡すと`holds`になり得るため、**flagだけのPASSを検収しない** | Sourceの分岐を確認。誤申告防止のProvider試験はNOT RUN |
| **CC-03 partial counterexample** | Owner L={L1,L2}、実読取L1:falseのみ、L2未取得 | 現行`EvaluateCompletion`はCoverageに先立って違反を検出し`fails`。対象Scope内の確実な反例があるときだけ局所否定を主張できる。不存在/全員成功の推論とは非対称 | Sourceの分岐を確認。実Providerのscope証明はNOT RUN |
| **CC-04 empty/unknown/disclosed** | 真の空・未取得・非開示・ページ不足を区別し、Ownerの空集合Policyを明示 | `Targets=[]`だけでOwner集合が空と決めない。`EmptyPolicyAllow`／`RequiresEvidence`／`Deny`の分岐を、実Readの空根拠・Policy・Disclosureと独立照合 | 既存Unit Testで部分分岐は実装済み。外部空証明はNOT RUN |
| **CC-05 Source/Manifest drift** | 移行前後の`*.tei`／`*.tei.md`、`tei.workspace.yaml`、実Loaded Rule選択、Plugin/Generator/Toolchainを個別固定 | Canonical Rule ID・World digestが同一でも、Manifest/参照先/実Loaderの変更を「未変更」と扱わない。未読込・二重選択・Source入替を`unqualified`にする | 移行実Reader/CLI TestはNOT RUN |

**同じSourceでも複数レイヤの負例を混ぜない**：[#3017](https://github.com/hat47x/tei-platform/pull/3017)の1 Mutant killは、合成`[1,1]`が**Evaluatorへ渡された後**に重複を誤判定する欠陥を検出したもの。提案中の末尾脱落MutantはこのEvaluator内部の未走査を次に試すもの。上表CC-02は**取得前／Callerの集合完全性誤申告**であり、既存`scenarioValidationSource.RelationKeys([]string,error)`でも`EvaluateCompletion`でも、単独では真のOwner集合を復元できない。各々の`Case×Claim×Attempt×Environment`と、知っている実Source及び未観測欄を別行に保つ。

**再資格の最小束縛**：`OwnerTargetSetID+revision+scope/asOf, actualProvider+readAttempt+observedRowIDs+pageEnd/snapshot, completenessBasis+currentDisclosure, canonicalRuleID+selection+sourceOccurrence, loadedWorld+ManifestIdentity+actualCompiler/Plugin/Artifact, independentExpectedProvenance, consumerClaim`。本リストは**既存Evidence/Receiptへの参照列**であり新しい公開Schemaではない。現main `internal/workspace/loader.go`（blob `ddae9bf724b685b1109b015c169da72ea740ada5`）では`Result.ManifestIdentity()`がManifest path/SHAを返すが、**ManifestはSource Indexの正本に含まれない**という実装コメントがあり、World同一だけでManifestの同一性を立証できない。Workspace Importsが実Indexに束縛できなければ`ErrImportsNotBound`でfail-closedにする既存入口があるため、Imported Sourceを黙って`complete`へ昇格しない。

**移行との接続**：[#3030](https://github.com/hat47x/tei-platform/pull/3030)の方向決定により新規TEI Authoringは`*.tei.md`標準、旧`*.tei`は原則段階廃止。しかし新Loader／Parser対応・Source span再解決・Owner Meaning再受理は未実行であり、拡張子Renameだけで上記CC-05をPASSにしない。旧新Sourceが共存した期間は、同じMeaning IDが二箇所にあれば**どちらの実使用・受理・選択か**を証拠で決め、後勝ち選択を推論しない。#2621/#3030は形式移行、#2595/#2600はSource/再資格、#2586/#1163はProvider集合、#2514は独立検証基準、#1180はScenarioを所有。

**今回の完了／未完了**：#2811の前HEAD `61ed5e7bf0185ec850584f8d3f4e3f4eb29b8659`に対する[Health #42216506479](https://github.com/hat47x/tei-platform/actions/runs/42216506479)は**completed/success**。本節は現行Sourceと他レーン計画の直接照合、および独立Expected/試験順の文書化のみ。CC-01〜05の新たな実Provider/Source移行試験、末尾脱落Mutant、Owner Meaning再受理は**NOT RUN**。#2514や#3030の担当Issue・コードは編集せず、当初の成功・失敗証拠は履歴保持する。

### 8.10 空集合を断定する前の実証入場：Caller申告と別範囲の証拠（2026-11-07）

**旧main（修正前）の実装から直接確認した分岐**（`internal/semantic/completion.go` blob `512a835ca445d81dd043f8af78ca29df4e7adb3b`）：`EvaluateCompletion`はOwner集合の`Coverage`、`EmptyJustified`を**Caller申告**として受け取り、実ProviderのRowset・Snapshot・実権限を自ら取得・検証しない。前節§8.9の非空partialに加え、**空候補における不確実性の扱い**も具体的な設計検収に必要である。

| 追加の負例（個別に実測・未実測を表示） | 旧main／統合後mainで確認した局所挙動 | 独立検証で確定すべきClaim／禁止推論 |
| --- | --- | --- |
| **CEC-01｜禁止・未取得**：`Targets=[]`, `CoverageUnestablished`, EmptyPolicy=`deny` | **旧mainは`fails`、統合後mainは`unverified`**。#3051 merge `b30ce3708a8385dcf219aed9f53e594bc0e95d12`・[Health #42298116044](https://github.com/hat47x/tei-platform/actions/runs/42298116044) success。局所Go対照も確認済み | `R=∅`は`L=∅`を含意しない。#1125 CDE-02の独立期待値に沿う局所修正。ただしOwner対象集合の実Read・現在Grant・正式意味受理は別 |
| **CEC-02｜独立空証明の自己申告**：`Targets=[]`, `CoverageUnestablished`, EmptyPolicy=`requires-evidence`, `EmptyJustified=true` | **`holds`**。`EmptyJustified`値を条件として使用し、Coverageには別の確認をしない | EmptyJustifiedの独立したProvider Read Scope/時点/完全性が正しいなら成立し得るが、同じboolだけでは根拠を検証できない。出典なし・別Tenant/Snapshot・権限変更時には完了適格にしない |
| **CEC-03｜許可・未取得**：`Targets=[]`, `CoverageUnestablished`, EmptyPolicy=`allow` | **`unverified`**。同様の空候補でもCoverageの状態によって分岐が変わる | `[]`が観測済み空集合なのか未取得かを独立に確認。Policy名だけで真の空を主張しない |
| **CEC-04｜局所反例・partial**：`Targets=[{L1,false}]`, `CoveragePartial`, 非衝突 | **`fails`**。確実な反例があればCoverage不足より先に評価 | 同Scope/版のL1が本来のOwner集合に属することを独立照合。未知のL2全件についての不成立・存在・権限は主張しない |
| **CEC-05｜Source新形式未対応**：旧`*.tei`と新`*.tei.md`が混在し、新形式定義が未読込 | mainの旧Loaderの挙動と、[#3031](https://github.com/hat47x/tei-platform/pull/3031)の暫定`TEI_MD_PARSER_UNBOUND`拒否は**別HEAD・別実装** | #3031 HEAD `63953ccb5c12ead26ec1268e7c0afdba4eff66ae` の[専用CI #42217379093](https://github.com/hat47x/tei-platform/actions/runs/42217379093)・[Health #42217378867](https://github.com/hat47x/tei-platform/actions/runs/42217378867)は成功。証明したのは**黙った未読込の拒否**でありMarkdown実Parser・意味受理・既存Scenario同等性ではない |

**独立期待値の境界**：CEC-01の`unverified`は[#1125 CDE-02](https://github.com/hat47x/tei-platform/issues/1125)の独立期待値に基づく**main統合済みの局所Go契約**で、旧mainの`fails`を修正した。#3051の修正・既存正例は当該Source版でGo/CI PASSだが、業務Meaning Ownerの正式受理や現在の実Provider集合の正しさを自動認定しない。#2514は判別力・独立性、#1180は必要Scenario、#2586/#1163はReader実証・同一Attemptを所有する。CEC-02もboolの値を証明そのものと見なさず、取得母集団を外から確定する。**同一Caller引数で、実世界が全件観測済みか欠落か異なる二世界**の識別には、この関数外の証拠が不可欠である。

**横断実証の追補（2026-11-07）**

- [#2514 CDE-Q01〜04](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6696895001) の独立期待値を参照し、Claim `Q=A/O1/R1`へ別Tenant B/O1の0件・違反、同Tenant旧Snapshot R0の0件を流用しても **Qのholds/failsには資格なし**とする。対象内の確実な違反はpartialでも限定的な否定根拠になり得る。新ケース実Go/実Providerは **NOT RUN**、Expectedの第二正本は作らない。
- [TAI固定旧TEI CLI負例 #42297500012](https://github.com/hat47x/tai-pleroma/actions/runs/42297500012)はcompleted/success。未対応の同一路径 `*.tei.md` bytes変更でSource Index FPは変化し、World FPと旧Catalogは不変。renameのみならSubject 1→0でも`validate.ok=true`。**Index認識とParser/Meaning実消費は別証拠**。`required/managed/indexed/parsed/selected/execution-consumed`を同一Source/Occurrence/Roleで照合し、移行は[#3030](https://github.com/hat47x/tei-platform/pull/3030)の検収へ引渡す。旧固定CLI版の負例であり新Parser合格ではない。
- [#2514のOKF Attested Computation境界](https://github.com/hat47x/tei-platform/issues/2514)：決定的なAttesterでも独立Expected・実Reader Scope・Source実消費・Owner Meaning・Host Grantを自動証明しない。`verified`は実Consumer再読取の代わりにならず、今回のOKF実E2Eは **NOT RUN**。

**局所受入の残余**：#3051は「未取得の空を`fails`にしない」修正であり、Callerが誤って`CoverageComplete`を立てる反例（§8.9 CC-02）と、`EmptyJustified=true`に対応する実Readerの独立空証明（CEC-02）は依然未資格。今後は**対象main HEADごとの必須Go Case非SKIP・Healthと外部Reader Witnessを別判定**する。過去の固定HEAD成功は履歴に残すが、後続mainとの結合PASSへ加算しない。

**修正の可能性と反証**：完成条件の型や`EmptyPolicy`を一律に変えず、①Policyの業務上の受理出典、②Owner TargetSet ID/版／Parent/Tenant／Relation Grain、③Reader Scope/asOf／Snapshot／ページ終端／RLS/Disclosure、④実Caller入力、⑤現在のHost Grant／Reader権限、を固定して関数内部の判定と独立に比較する。特に`empty`、`missing`、`partial`、`permission-denied`、`not-disclosed`のいずれも`Targets=[]`単体へ畳まない。実データの`NULL`値を未知・文字列の代用にしない。実TargetSetが決定不能なら`unqualified`を残し、見かけの`holds/fails`で正当化しない。

**実Sourceと意味再資格**：#3039/#3040の用語規約（テストの判定根拠は「独立検証基準」「独立期待値」、DB製品は「Oracle Database」）を採用。旧来の機械的なテスト名・Package・既存Evidence IDは変更しない。新しいAuthoring Sourceの`*.tei.md`統一は決定済み。現時点の**移行仕様・検収IDの主担当は[#3030](https://github.com/hat47x/tei-platform/pull/3030)（TM-01〜14）**、[#2667](https://github.com/hat47x/tei-platform/pull/2667)は横断依存・引渡し資料であり、第二の移行Gateとしない。#3031の暫定Guardを`actual parsed Source`へ昇格しない。World一致でも実Source File/Occurrence、Manifest path/SHA、Parser/Role/Plugin版、Owner Meaning受理と独立Expectedの版を再照合する。

**資格・作業範囲**：**CEC-01に対応するCompletion API修正と、空集合禁止・partial正負対照4件は[#3051](https://github.com/hat47x/tei-platform/pull/3051)のHEAD `581e699b62b126c86388c88c7c357674394ad252`で追加・Go全体CI PASS**。Health [#42256518693](https://github.com/hat47x/tei-platform/actions/runs/42256518693)はcompleted/success、Self-Build等も同HEAD成功。**main統合済み**であり、現mainのSource・Go Test Blobを直接再取得して確認した。CEC-02の独立空証明検査、CEC-05のMarkdown実Parser、実Provider/Read/Permission、Owner受理、末尾脱落Mutantは**NOT RUN／NOT ACCEPTED**。本節は現行Sourceと[#3031](https://github.com/hat47x/tei-platform/pull/3031)の固定CIを材料とした設計追補。第二検証基準Engine/専用DBや新Runnerは導入せず、#2514/#2586/#1163/#2667の担当Issueへ実装を複製しない。

### 8.11 Claim相対の完全性：同件数・TOP 1・未取得Predicateを独立に検収する（2026-11-07）

**他担当の新しい独立Expectedを接続**：[#2514 CDE-Q05〜07](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6698037813)と[#2506 R05〜07](https://github.com/hat47x/tei-platform/issues/2506#issuecomment-6698048070)を参照する。両Issueが固定したCase ID・Expected・Result Sliceの意味を本台帳へ複製して新しい正本にせず、**Case×Claim×Attempt×Environment**に適用する条件と実証状態を記録する。今回のSource/CI照合では新しいCaseを実走していない。

**固定された入力根拠（現mainで再取得）**：`reference/web-api-contract/fixtures/postgres_scoped_orders.sql` blob `5113f47d131da7c6b00c7f00dd67fcd7298a53da`には、Tenant Aの注文 `3 (OPEN)`・`9 (READY)`とTenant Bの `1,100` がある。`reference/web-api-contract/scoped_postgres_integration_test.go` blob `062fe0ab4262bb066d5a06090c21453e6d1cc73f` は `$select=orderId`、`$orderby=orderId desc`、`$top=1` でAの結果 `[[9]]` を、昇順で `[[3]]` を独立に期待する**実GoテストSource**を含む。実PostgreSQL実行には `TEI_WEB_SCOPED_PG_DSN`と隔離DBの明示Opt-inが必要。一般的なGo/Health成功やテストSourceの存在は**当該PG Caseの実走・非SKIP証拠ではない**。

| #2514の既存Case | 現物／合成の一因子反例 | 局所結果と業務Claimの分離（固定期待） | 認定の上限／現状態 |
| --- | --- | --- | --- |
| **CDE-Q05 / R05 件数一致・集合欠落** | 必要集合 `L={A/3,A/9}`、合成結果 `R=[A/3,A/3]`、行数はともに2 | `len(R)=len(L)`でも重複した**同一Occurrence**を2件と数えない。`distinct occurrence(R)={A/3}`、A/9未取得。Callerの `CoverageComplete`は根拠なし。JOIN増幅による同一行重複を**元データの業務キー違反**とも断定しない | 対象全集合のCompletionは**unqualified**。独立LとRの一意Identity・Multiplicityを照合する新試験は**NOT RUN** |
| **CDE-Q06 / R06 ページ完了と全集合** | `L={A/3,A/9}`、既存Test SourceのA降順 `$top=1` が返す候補は `R={A/9}` | 要求ページ1件の表示成功（当該Scopeでの閲覧Claim候補）≠Owner対象2件の全称成立／不存在。存在肯定は実Read対象Identityが同Scopeなら限定的に支持され得るが、否定依存は全件性を要する | **ページ要求の局所資格候補**。当該実PG試験はSource存在を確認しただけで、本結合Attemptは**NOT RUN** |
| **CDE-Q07 / R07 Predicate必要Fieldの未受領** | P=`status=READY`を独立固定、Consumerの `$select=orderId` には`status`がない。FixtureではA/3=`OPEN`、A/9=`READY` | IDの取得はP評価の証拠ではない。`Holds=true`の自己申告は根拠未確認。未投影を`NULL`・`false`・`true`に変換せず、非開示Fieldを当該Consumerへ無断追加しない | **Predicate evaluation unqualified**。必要ならD/V/Q/Aで明示された許可済み内部Rule Readの実FootprintとConsumer開示境界を別証拠で照合。新結合試験は**NOT RUN** |

**反例の非対称性を維持**：同一Parent/Tenant/Relation occurrence・Snapshotで`R⊆L`と観測違反が確実なら全件未観測でも限定した不成立の根拠候補となる。一方、`len(R)=len(L)`、`rowset success`、ページ応答完了、`status`以外の列を受信済み、`CoverageComplete`のCaller設定、CI PASSはいずれも**全件・全Required Field/Predicateの保証とはならない**。Row identityと表示row indexを混同しない。Unknown/withheld/not-fetched/nullは別状態。

**既存Evidence票への追加の比較列（公開Schemaではない）**：`Claim ID/purpose/required L and occurrence+grain`、`actual R distinct IDs/multiplicity`、`query+args/filter/order/top/page termination`、`required predicate/hidden Rule dependencies and field presence`、`tenant/parent/asOf/snapshot/RLS/Disclosure`、`actual loaded Source/Rule/Reader+Attempt`、`independent Expected origin+Consumer/Claim ceiling`。LをRや同じQuery結果から再生成して自己採点しない。複数Read/Consumerを別Attemptから合成して完了としない。RLS非開示のTarget IDやFieldを非権限Consumerへ露出せず、資格判定側で必要情報を確認できなければunqualifiedとする。

**実行と責務**：#2514がCDE-Q05〜07の独立期待値、#2506がResult Sliceの用途相対Coverage、#1180が既存Scenario SSOT、#2586/#2622がQuery→実Provider Read、#2516が開示、#1163が一改訂Packetを所有する。最初は既存合成Rで「同件数/ページ/投影だけでは全件Claim未資格」を検査し、必要Caseが見つかった後に隔離PG opt-inの実Readを**同一HEAD/Scope/Attempt**に束縛する。新Runner、新Status Enum、専用検証基準Storeを作らない。#3051の局所Completion回帰と#3017のMutant killは歴史的な別Claimとして保持。**CDE-Q05〜07の新Go・実PG結合、Owner意味受理、OKF Attester E2EはNOT RUN**。
