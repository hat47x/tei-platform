# TEIテスト基本設計と証拠台帳：Case再利用とClaim単位の再資格化（2026-11-06、Draft）

**位置づけ**：本書は[AI-firstテスト計画](./CROSS_ROUTE_FUNCTIONAL_PARITY_AND_TEST_DERIVATION.md)の基本設計側の参照文書である。2026-11-05のUnit/Fixture・Conformance・Provider/DB・Host/UI・Owner受理の整理と、#2514/#2622/#1129/#1163/#1119等の既存計画を接続する。**docs-only・提案／新試験NOT RUN・Owner採用未決。** この文書は新Runner、新Registry、新Acceptance Authority、新テストDB、第二Scenario正本、製品Wire Schemaを制定しない。

## 1. 正本・派生物・証拠の分界

- **Meaning/Requirement**：受理されたCanonical/Rule/業務資料、Source locator・Role・版・Owner判断が業務意味の参照先。AI生成したExpected、合成Fixture、既存Goの実行結果は意味の受理ではない。
- **Test Definition**：業務Scenarioは#1180の保存済Driver ScenarioがテストのSSOT。DSL PipelineのJSON Candidate（#2813）とDriverのJSON候補（#2828）は異なる局所Consumerの*候補入力*。正式受理済みScenario・固定Anchor・秘匿OracleをAgentが上書きしない。Markdown表は#2621によるTestとしての表示・Authoring候補であり、説明用表と独立ExpectedをRoleで区別。
- **Execution Evidence**：Go/CLI/CI、実Provider、Target/Host、実Readerが観測したattemptごとの結果・receipt・参照可能なログ・観測可能範囲。別HEAD・別TargetのPASSを合算して一回の実行にしない。
- **Qualification View**：主張（Claim）ごとに、必要な証拠が同時に成立するかを既存記録からread-onlyで導出。歴史的PASSを削除せず、現在資格だけ再評価する。#1163が一改訂横断証拠、#1119が横断Gate判定のOwner。資格計算そのものを新Authorityへ昇格しない。

## 2. 五層は実行順ではなく証明の範囲

| 層／Owner | 主目的・実観測 | この層のPASSが支持する範囲 | 自動では証明しない範囲 |
| --- | --- | --- | --- |
| **L1 Unit / Fixture**（#1129、#2813/#2828） | 型・数値・Rule・Lifecycle phaseを固定した合成入力/negative controlで検査 | その実装・Fixture/Source/Toolでの局所振舞い | 実Source全閉包、独立Oracle、業務意味、実DB、Target、現在Grant |
| **L2 Conformance**（#2514/#2622） | 固定Contract/Profile/independent Expected/Mutantを対象実装・Provider間で比較。NULL/precision/grain/partial/順序を指定 | 指定Profile×対象Version×適用Scopeに対する限定的な適合 | 未テスト能力・本番対象/権限、実Providerへの接続、全Domainの意味同一性 |
| **L3 Provider / DB実測**（#2622/#1174） | 実Data Source alias、Schema/SQL asset、実Query/Read set、実Commit/rollback/unknown、別接続Read/Receiptを検査 | 特定DB/Provider/Principal/Snapshot/Operation/Transaction条件の実動作 | 別DBでの同等性、Web/Host全体、正式Owner受理、将来のGrant |
| **L4 Host / UI観測**（#1163/#1180/#2516） | 実CLI/Web/Desktop binary、Session/HTTP/Command、操作・表示・Consumer取得body、必要なら実Providerへ一Attemptで接続 | そのTarget・Revision・Principal・経路の観測済み機能 | 他Target、表示外のEffect、現在/将来の許可、業務意味受理 |
| **L5 Owner受理**（業務Owner/#1119） | 出典に照らしたRule/Identity/Scope/許容差・例外・運用条件の意思決定を版付きで記録 | 受理された意味・用途・時点の範囲 | 実行正しさ、現在Grant、実Provider/OS/UI/納品資格 |

**L5は自動試験の上位レベルではない。** Owner判断はL1〜L4の前・後・並行に存在し得る。従ってステージを機械的にL1→L5へPASS継承しない。L2の独立Oracleでも、候補・Reference・Expectedが同一の誤ったRuleから派生していれば資格なし。#2514のSoundness/Binding/Coverage/Observational Sufficiencyを**Claimごと**に照合する。

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
| Qualification/Expiry | 支持する正確なClaimとScope、Oracle independence、binding/coverage/sufficiency、historic verdict、現在資格、valid-at/valid-until（定義される場合）、invalidating eventと再資格対象、権限付きOwner Decisionへの参照 |

**証拠の結合は共通IDだけでは不十分。** Case ID/Source digestが等しくても、実消費byte、Builder/SQL選択、環境Snapshot、現Grant、必要Obligation、Targetが異なれば別Claim。親子Step、Job、Operation、Provider Receiptの相関を#1163/#1174の既存IDで追跡し、二つのDB接続Readが同時点のデータを表すかはSnapshot/Version Witnessで別確認。機密Raw/DSN/holdoutは公開票へ転載せず、#2515/#2516の許可されたProjectionと内部参照のみ。

### 3.1 三つの直交状態

- **Execution**：PASS / FAIL / SKIP / NOT RUN / BLOCKED / UNKNOWN。歴史的実行結果は不可逆に記録する。
- **Evidence qualification**：qualified / limited / unqualified / stale / unresolved。特定Claimと比較時点を引数として判定する。これらは説明上の例であり製品enum未採用。
- **Meaning acceptance**：Owner accepted / pending / rejected / withdrawn（対象Scope・Source版付き）。Execution PASSからこの列を自動更新しない。

**必要集合を固定したうえで不足を測る。** `requiredCaseIDs - observedPassedCaseIDs`が非空、Requiredチェックが未提出・未発見・SKIP、Loader未読込、unknownな対象母集団がある場合、該当Claimはunqualified/unknown。ケース数だけの分子・分母や全Go Test PASSを全業務Coverageに使わない。#2835の「DSN未設定でもGo試験自体は緑だが必要PG試験はSKIP」を既存負例として再利用する。

## 4. 同一CaseをL1から段階強化する圧力試験

採用候補は[#1129 XL-02/03](https://github.com/hat47x/tei-platform/issues/1129)の小さい独立Expectedを用いる。Caseの安定意味Subjectを維持しても、**各環境で同じExpectedが適用可能か**と**各AttemptのSource/結果**は別照合。業務規則・Expectedの改訂時はCase revision/Oracle版を変更し、旧Receiptを現在資格に流用しない。

| 段階と同一Caseの利用 | 固定Expectedと追加する実観測 | Negative control／受入・停止条件 |
| --- | --- | --- |
| **F0 definition** | `order_line.quantity=2, unitPrice=300 → amount=600`を合成の独立Expectedとして固定（税/丸め/時点・粒度が別条件なら保留）。quantity=0、NULL、missing、partialは各自独立のExpectationとする | Source locator・Role・Meaning未受理のRuleを数値例だけで正本化しない |
| **F1 Unit fixture** | 実TEI純粋Operator/Validationを固定Fixtureで呼び、原値・型・結果/拒否を取得。テストGo/Case revisionを固定 | ResultをWantへコピーする同一起源Oracle、decimal→float、NULL/missing混同、required Case SKIP |
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
| Expected/Rule/Role/Operator/Plugin版変更 | 該当Case/Oracle G2、Semantic Scope、Conformance適用、関係するTargetの実評価 | 無関係なCase・既知履歴Receipt（現在合格とはしない） |
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
| Unit-case-passed | 実Source/Tool、Case/Fixture、実Actual/Expected、実行Outcome | 独立Oracle・Business Meaning |
| Conformance-profile-qualified | 独立Expectedの由来/Mutant、Profile/Capability/Scope、必要ケース全実行、観測十分性 | 実DB/Host、本番 |
| Provider-effect-observed | 実Provider/DB/Principal、実Tx/Operation、Receipt、独立Read/Effect、失効チェック | 別Provider・正式Owner Meaning |
| Host-route-observed | 実Binary/Actual Target/Grant、Request→Result、実Consumer body、必要なら実Provider Receipt結合 | 別Target、未観測効果・全Required Coverage |
| Meaning-owner-accepted | 正式Authority、出典Source/Rule版、受理Scope/許容条件・撤回規則 | Runtime健康・現在Execution Grant |
| Product gate | #1119の必要Witness集合が**同一改訂・同一対象Attempt**で成立しOwnerが判定した範囲 | 別版CIの足し合わせ、未受理Domain/将来の保証 |

**必要なnegative control**：①同じCandidateとExpectedを共に誤改変して見かけPASS、②required Case/Checkの未発見やSKIP緑、③Source宣言D1/実Build D2、④Owner受理なしで実値だけ一致、⑤R1/R2混合SuiteをReaderが読む、⑥Commit ACK unknown後に再実行、⑦同表示値でScope/NULL/grain違い、⑧blocked RunPlanの先行Effect残存。この8群を既存#2514/#2595/#1180/#2604/#1174/#1130のFixtureに参照で割当てる。全ケースを新設するのではない。

## 7. 統合先・実装優先・現在の証拠

| 優先度／責務Owner | この基本設計から渡す最小差分 | Doneの判定 |
| --- | --- | --- |
| **P0：#2811/#1180 AI-first** | #2813 DSLと#2828 DriverのCase ID、Required/Discovered/Attempted/PASS/SKIP、Fixed Source・Known Mutant/独立Expectedを実Run票へ記載 | 新JSON追加と独立Oracle判別の実測。既存2 StepのPASSだけではAI新規発見を証明しない |
| **P0：#2514 Oracle** | G2独立性、mutant判別力・観測十分性、Case/Claimの比較可能なExpected provenanceをReport Viewに追加 | 同一起源偽PASSを負例で検出し、G2/unknownを分離 |
| **P1：#1129/#2622 Provider** | XL-01/02→03とDO-04〜07の同一Subject/独立Expected・NULL/partial/Grainを使い、実PostgreSQLと第二DBまたはAPIの差を測る | 実Query/Request・Result/Effect/Scope別にObservedが得られ、未対応はunsupported。既存#2853/#2854/#2857/#2859の実PG成果を複製しない |
| **P1：#1174/#1130 Effect** | required-check completenessとpartial-effect/unknownの二反証。新Key ABAの再適用と同Key Receipt Lookupを分離 | 実操作・実Provider/Grant・Stepごとの観測と安全な停止／照会 |
| **P2：#1163/#1119 Host/Owner** | 同一Accepted Revision→Web/CLI→Provider→独立Consumer、S02/S05/Grant・Owner Scopeを一Attemptで資格化 | 統合PacketとGate Owner判定。#2857の監査がINCOMPLETE/UNQUALIFIEDである間は完了にしない |
| **横断：#2600/#2595/#2604/#2516** | 実消費Input、Source/Meaning/Artifact再利用の失効、一式公開と実受領を個別資格とする | 改版・partial・ABA・未許可Read/開示に対する選択的再資格化 |

**2026-11-06時点で確認可能な限定成果**：
- #2828 test-only DriverのHEAD `77baf709` は [Repository Health #41850080590](https://github.com/hat47x/tei-platform/actions/runs/41850080590) completed/success。#2811既存docs HEAD `3251906b` は [Health #41850309534](https://github.com/hat47x/tei-platform/actions/runs/41850309534) completed/success。両PRはDraftであり本基本設計の追加前HEAD。
- [#2853](https://github.com/hat47x/tei-platform/pull/2853)実PG Command/CAS/Receipt、[#2854](https://github.com/hat47x/tei-platform/pull/2854)ABA別Key適用、[#2857](https://github.com/hat47x/tei-platform/pull/2857)暫定一改訂Web/PG（監査 INCOMPLETE）、[#2859](https://github.com/hat47x/tei-platform/pull/2859)Runner起動前Schema Gate等は**異なる版・個別実走**。同一受理Sourceと現Grantを束縛した一体Web→PG→Consumer合格を主張しない（#1163）。
- 本設計のF0〜F5、P-REQ/P-PARTIAL、台帳Qualification Viewの実装/試行、正式Schema、第二DB/SHO/Windows、Owner受理は**この変更ではNOT RUN／NOT ACCEPTED**。

**採否**：#2514がOracleの根拠、#1180がScenario、#2622がProvider適合、#1129がData Hook、#1174がEffect、#1163が同一改訂Packet、#1119が横断Gateを引き続き所有する。証拠台帳はその間のread-only比較表であり、一本の万能PASSや新テスト基盤を作る根拠にしない。

## 8. 実証結果を記入した証拠台帳：テストPASSと要求適格を分ける（2026-11-06）

**この章は実際のCI・Go・PostgreSQL試験結果を既存のCase/Claim/Evidence比較票へ転記した例であり、新しいLedger Storeや新実行結果ではない。** GitHub上で観測済みの結果と、未観測の部分を明記する。別HEAD/Environmentの行を足して同一Attemptにしない。

### 8.1 初期Ledger：5件の実例

| 仮ID・既存Case | 固定Environment・証拠 | Test Outcomeと観測事実 | Claim資格／まだ証明しないこと |
| --- | --- | --- | --- |
| **EL-01** [#2828 Driver JSON Candidate](https://github.com/hat47x/tei-platform/pull/2828) | HEAD `77baf7092ff530fb8dee2e53ef4752a8ee01f953`／[Health #41850080590](https://github.com/hat47x/tei-platform/actions/runs/41850080590) completed/success | test-only Go/JSON追加、2固定Validation Anchor、期待値改変・旧Plan拒否を含む限定Fixtureの実Go Test成功 | L1の該当Referenceに限り支持。新しい未知Caseを生成AIが自律発見した証明、独立業務Oracle、Owner受理、実DB/Host資格ではない |
| **EL-02** [#2830 Scenario 6 Step](https://github.com/hat47x/tei-platform/pull/2830) | HEAD `019da011c7d0e1d4ade384c5495c3f91bb12982e`、Go1.26.10/Linux、[Health #41851323500](https://github.com/hat47x/tei-platform/actions/runs/41851323500) completed/success | 6 Step IDの欠落/重複、生成argvの各Step実行、`static-verification`/`generated-pinned`と`evidenceNature`を限定Goテストで照合 | L1のScenario Inventory・Evidence Nature限定適格。`Verified=true`は実Auth/Command/DB/Web E2Eと同じ意味ではない。6 Stepを全業務Coverageの分母にしない |
| **EL-03** [#2870 RCAS-02](https://github.com/hat47x/tei-platform/pull/2870) | HEAD `58d4a0047`、WSL2使い捨てPostgreSQL 14.24、PR記録のGoテスト/実DBログ | **テストPASSは欠陥特性の再現成功**。新Lifecycleでin-memory `unknown`が消失、再Dispatch可能。同じ冪等Keyでは実PGが`IDEMPOTENT_REPLAY`・Applied=0・Receipt1件 | Durable lifecycle/実プロセス再起動安全性は**未成立**。同Keyの限定再送が二重適用を防いだことだけ支持。新Key ABA、Network/実crash、現在Grantは別 |
| **EL-04** [#2882 Workspace/RunPlan](https://github.com/hat47x/tei-platform/pull/2882) | main統合済みのGo試験。期間閉/開、Effect Observer欠落をWorkspace Sourceから入力。**本票では当該merge HEADの実CI Receipt未確認** | 期間閉ではblocked/Step未呼出、期間開ではcompleted、Effect前提欠如ではblockedを検査するテストを追加 | Source実装の存在は確認できるが、**今回の確認だけでexact-head CIの実PASSは主張しない**。`CombinePreconditions`に渡された検査の成功≠必要検査全件の提出、一般RunPlan blocked≠過去Stepの作用0 |
| **EL-05** [#1163 一改訂Product Gate](https://github.com/hat47x/tei-platform/issues/1163) | #2853実PG Command/Receipt、#2857暫定Web/PG、#2859起動前Schema Gateは**別HEAD/別Attempt** | それぞれの限定的な実PG/実Web/Schemaの成功が存在。#2857の横断監査はINCOMPLETE | **Product Gate=UNQUALIFIED**。同一Accepted Revision・現Grant・実Web→実PG Commit→独立Consumer受領という一経路の統合Receiptを合成できない。Owner受理なし |

| **EL-06** [#2890 AIによる別RuleへのScenario追加](https://github.com/hat47x/tei-platform/pull/2890) | #2828 HEAD `77baf709`から分岐。Candidate HEAD `8e508413fe92cfc95a6176e34f8af8b35fb576ae`、新規Scenario＋Fixture JSONの2ファイルのみ | AIがJSON候補 `ai-candidate-line-number-uniqueness` の2 Stepを追加。既存正常FixtureのRelation keys `[1,2]` と新しい`[1,1]`の**一つの入力差分**で `line_number_unique: satisfied→not-satisfied` を予測した。Go/既存6 Step/Anchorは変更なし | **Candidate authored／実Go run結果未確認**。独立Oracleは#2514で未検証。G2/全業務Coverage/Owner意味受理・実Providerは未成立。既存#2828 CI成功を新HEADのPASSには流用しない |

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
| **E-02 AI追加ケース** | #2828 2固定Anchorとは別に、[#2890](https://github.com/hat47x/tei-platform/pull/2890)の2追加Step（`ai-line-number-unique`/`ai-line-number-duplicate`）と一条件差分、独立Expected/Mutantの出典（#2514） | 追加前後のCase集合と実Attemptを比較。JSON-onlyのAuthoredは確認済みだが、executable/observed/independent-oracle-qualifiedは別。期待をActualに合わせて自動修正せず、Candidateと同一起源のWantだけでOracle資格を付与しない |
| **E-03 必須Preconditions** | Operationに必要なPeriod/Effect/Grant/Observer/Schemaの**適用条件付き必要集合**を実行前に固定 | `CombinePreconditions`の入力に必要Checkがすべてあることを照合。渡された分が全PASSでも必要Checkが抜けていたらCoverage不適格。全Operationへ無関係Checkを一律要求しない |
| **E-04 再起動と副作用** | RCAS-02同Key再送、#2854新Key ABA、#1174 Commit Unknown/Receipt Lookupを独立Expectedとして事前指定 | Attempt/Operation identity、Step実Effect、PG Receipt、別接続Readを結ぶ。Blocked/Unknownの無条件再試行を認めない。実Process crash/Networkは未実行なら明示 |
| **E-05 SQL/Provider選択** | #2622/#2862の同Query ID・異Realization/SQL選択、Read Scope・NULL・Grain・mapping資格 | Query IDや表示Rows一致は実SQL/Providerの同値を証明しない。固定SQL不在時の無音fallbackを拒否。第二DB実測は別 |
| **E-06 一改訂E2E** | #1163のAccepted Source Revision・実Binary・Current Grant・実PG・Receipt・独立Read/Consumer実受領 | #2853/#2857/#2859の別Attemptを合算しない。必要Edgeの欠落を個別unknownとして保持し、#1119 G1/G2へ誤昇格しない |

**次の最小出口条件**：E-01とE-02の実Case ID/Required集合/Actual Runを一枚で照合し、E-03またはE-04の既知負例で「テスト自体PASS、製品Requirement不適合」を少なくとも一件説明できること。E-05/E-06の実Provider/Hostは本P0のHard Blockerにせず、既存Ownerが個別Evidenceを取得した後に段階強化する。

**更新は設計・証拠のread-only整理のみ。** E-01〜E-06の新たな実行、AI自律生成、独立Mutant試験、実Process再起動、二DB・完全Host/Provider E2E、正式Owner受理は本変更でNOT RUN。新Ledger DB、Runner、Acceptance Engine、Ownerの正本を追加しない。
