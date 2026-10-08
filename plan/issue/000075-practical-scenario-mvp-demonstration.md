# ISSUE-000075: 実務シナリオを最初から最後まで通すMVP的な実証を先に行い、機能の積み上げと関係性を示す

- 状態: In Progress — 最初の実証Aは2026-10-05にownerが選択し、実行確認を記録済み。Bはowner未選択（Draft PR #985で未実行の事前回帰テストを準備中）、Cは未着手
- 提案日: 2026-10-02
- 性格: 実証計画 / Reference Evidence。測定に基づく価値判断は含まない
- 主担当要求: RQ-021（主）/ RQ-022 / RQ-012 / RQ-013 / RQ-014 / RQ-029
- 関連Issue: [ISSUE-000072](000072-rule-change-hand-edit-count.md)（測定は研究に留保）、[ISSUE-000073](000073-derive-validation-selection-from-canonical.md)、[ISSUE-000040](000040-semantic-component-composition.md)（第二Application）、[ISSUE-000046](000046-studio-shared-acceptance-authoring-loop.md) / [ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)（変更を受け入れる側）、[ISSUE-000026](000026-target-capability-execution-profile-boundary.md)（第二Target）
- 価値との対応: [Product Value](../../product/value/TEI_PRODUCT_VALUE.md)「価値の確かめ方」（積み上げた実務的な機能の量と、その有機的な関係性・実用性）
- Gate impact: None

## この文書の役割

Product Valueは2026-10-02に、価値を測定値ではなく、**積み上げた実務的な機能の量と、その有機的な関係性・実用性**で確かめる方針へ改まりました。手直しの箇所数のような指標は、要件固有で変更の深さによって意味が変わるため、問題領域の知見が蓄積されるまで研究にとどめます（[ISSUE-000072](000072-rule-change-hand-edit-count.md)）。

この方針では、価値を示すのは数ではなく、実際に動くものです。そのため、MVP（最小限の実用的な製品）やプロトタイプによる実証を先に行います。このIssueは、その実証で何を見せるべきかと、最初の候補を整理し、ownerの判断を待ちます。

## 背景

これまでの実証（`docs/status.md`が正本）で、`order_entry` Applicationについて、次が積み上がっています。

- Application Pattern（browse / record / reference / header-detail / command）、Validation、Authentication / Authorization、Capability Binding、Business ResultとExecution Errorの分離、Command後の別Observation
- Application Driverでの実行、第一Web Targetでの実行（Gate C）、実PostgreSQLでの検証（Gate B）
- Shared Acceptance Boundaryによる、変更候補の検査・受理・記録（Studio UIとAIによる候補生成の入口は含まない）
- 第二Application `repair_intake` による、顧客の意味の再利用

一方、これらは機能ごとの実証であり、**実務のシナリオを一本通した実証**は、まだありません。機能が積み上がっていても、それらが有機的につながり、実務で使えることは、通して見せるまで示せません。第二Application `repair_intake`の顧客照会をWeb Target経由で通す作業は、`docs/status.md`に次の一歩として記載されています（S8）。

## 防ぎたい失敗

- **機能の一覧が価値の主張になる。** 機能が個別に動くことと、それらが同じ意味でつながり実務に使えることを、同じものとして扱う。
- **測定値で価値を語る。** 実証が無いまま、箇所数や削減率などの数で価値を主張する。
- **synthetic成功の昇格。** test doubleや小さいReferenceでの成功を、実Providerや本番の保証へ読み替える（AGENTS.md）。
- **大きすぎる最初の実証。** 一度に広い範囲を狙い、何が動いて何が動かないかが分からなくなる。

## 守る原則

- 実証は、実際に動くものを、実務に近いシナリオで通す。
- 実証した機能と、その間の関係を、記録として積み上げる。通せなかった部分も残す（RQ-022）。
- 実証の結果を、数に畳んで価値の根拠にしない。
- synthetic fixture、test double、実Provider、本番相当環境を混ぜない。

## 実証で見せること

次の三つを、一つのシナリオで確かめます。いずれも「通ったか、通らなかったか」で確かめ、数の大小では確かめません。

| 観点 | 問い |
| --- | --- |
| 量 | その実務シナリオを通すのに必要な機能が、揃っているか。足りない機能は何か |
| 関係性 | 機能どうしが同じ意味・同じidentityでつながっているか。意味を一つ作ると、複数の機能・複数のApplicationから、写し直しなしに使われ、元へ辿れるか |
| 実用性 | 実務に近い入力から、最後の結果の観測まで、実際に最初から最後まで通せるか。通せない箇所はどこか |

## 候補（Aはownerが選択済み。B / Cは判断待ち）

候補は、すでにある機能の延長で、範囲が小さい順に並べています。いずれも、新しい意味やCore primitiveを先に作りません。

| 候補 | 内容 | 既にあるもの | 足りないもの | 関係性の示し方 |
| --- | --- | --- | --- | --- |
| A. 二つのApplicationを同じ顧客の意味でつなぐ | `repair_intake`の顧客照会を、Customer ID入力からDriverの照会まで、第一Web Targetを経由して通す（`docs/status.md`のS8の次の一歩） | `repair_intake`のsource、顧客Semanticの共有、developmentのexecutor binding（test double） | Web Targetのread scenario runnerが`order_entry`固定であること、productionのexecutor binding、実Database接続 | 同じ`customer.customer`を、二つのApplicationが同じidentityで使うことを、画面から観測できる |
| B. 変更を提案し、人が確かめ、反映され、画面に出る一周 | 規則を一件足す変更候補を、Shared Acceptance Boundaryで検査し、人が読める形で確かめ、受理し、Web Targetの確定操作に反映されるところまで通す | Shared Acceptance Boundary（Preview / accept / 決定記録）、Web Target、Driver | 人が読む面（[ISSUE-000071](000071-approval-sufficient-projection.md)）、評価対象の選択がCanonicalな宣言から導かれること（[ISSUE-000073](000073-derive-validation-selection-from-canonical.md)）、Studio UI | 変更の提案から実行の観測までが、同じ意味の上でつながることを示す |
| C. AIエージェントが業務手順を読み、権限の範囲で実行する | Business ProcedureとCapability、Authorizationを読み、許可された範囲で受注確定まで実行する | Business Procedure Reference、Capability / Authorization | 製品側のAIエージェントの読み取り経路（RQ-030はDesign Only）、fail closedな実行権限の検査 | 人・Application・AIが同じ知識面を使うことを示す |

**整理上の見立て（推奨ではなく、判断材料）:** Aは既存の計画の延長で最も小さく、足りないものも具体的です。BはProduct Valueの個人・組織の価値に最も近く、000071 / 000073の機能とつながりますが、前提となる機能が複数あります。Cは範囲が広く、製品側の経路が無いため、Aの後です。ownerの判断で、AとBを順に、または並行して進めることも考えられます。

## 実証記録A — `repair_intake`の顧客照会を画面から通す（2026-10-05）

**シナリオ。** Customer IDを画面（`/customer-lookup`）またはJSON（`/scenario/customer-lookup`）から入力し、`repair_intake`の参照Pattern `repair_intake.customer_reference`をDriverのEnvironment-aware経路で実行し、顧客を表示する。実装は`tools/reference-first-web-target`で、正本・Core・Canonical meaningは変更していない。

**環境。** WSL2（Ubuntu）、Go 1.23.2。Environmentは`development`、Executorはsynthetic fixture（test double）。実Database、production、本番相当環境ではない。

| 観点 | 通せたこと | 通せなかったこと |
| --- | --- | --- |
| 量 | 入力 → intakeのmapped row → Relation keyからのContract Query parameter → Executor → 顧客の表示。画面とJSONが同じrunnerを使う。不正入力は400、未設定は503 | 認証の門、書き込み（Command）。`repair_intake`にはCommand Patternが無い |
| 関係性 | `order_entry`と`repair_intake`が、同じ`customer.customer`の意味と同じContract Query（`master_data.customer`の`find_by_id`）を、同じExecutor bindingで解決し、同じ顧客の属性を返す（`TestCustomerLookupReusesTheCustomerMeaningAndContractOfOrderEntry`）。Contract Query executorの対応表は一か所（`referenceContractQueryExecutors`）にあり、二つのrunnerが共有する | `order_entry`のPatternの引き継ぎはなく、`repair_intake`は自分のPatternを持つ（意図どおり） |
| 実用性 | サーバを実際に起動し、画面とJSONから照会を通した。manifestと異なるApplicationのrunnerは起動時に拒否される（以前は`order_entry`固定のrunnerが他Applicationのmanifestの後ろで動き得た） | productionでは実行できない（Executor bindingが無く、developmentへ代替しない。testで固定）。実Oracleへの接続は未観測。Customer IDの整数変換はTarget内のfixture上の選択 |

**次に作る機能の候補（記録から）。**

1. productionのContract Query executor bindingをReferenceとして持つか、持たないと決めるか（`order_entry`と共通の未判定事項）。
2. 実Database（actual Oracle）への接続と、そのprincipal / credential / network authorityの観測（未観測のまま）。
3. 照会の入口に認証の門を置くか。`order_entry`のAuthentication / Authorizationを`repair_intake`へ使えるかが、候補Bの前提になる。
4. 候補B（変更を提案し、人が確かめ、受理し、反映される一周）。このとき評価対象の選択（[ISSUE-000073](000073-derive-validation-selection-from-canonical.md)）と人が読む承認の面（[ISSUE-000071](000071-approval-sufficient-projection.md)）が前提になる。

この記録は、一つのシナリオの成功にとどまり、実務全般への適用可能性や、実Databaseでの動作を示しません。

## 実証Bの追加評価 — 業務設計から実行へ戻す一周（2026-10-08、評価待ち）

本節は、[Product triage #982](https://github.com/hat47x/tei-platform/issues/982)で示した着手順を、**既存の候補Bで試せるか**検討するための補足である。候補Bの選択はまだ行われておらず、2026-10-05にownerが選択・実施したAの記録を置き換えない。新しいProduct Gateや別の実証Issueは設けない。

### 先に確かめたいこと

TEIの実用性は、個別にModelやDriverを動かせるだけでは判断できない。小さな業務規則の追加を契機に、データの意味、画面・処理、担当者の仕事が同じ設計へ戻れることを確認する。変更の内容は、既存の[ISSUE-000072](000072-rule-change-hand-edit-count.md)で調査した「受注明細の単価は正である」という**未採用の候補**を第一候補とする。現時点のCanonicalにその規則が受理済みだと仮定しない。

- **Data**: どのentity / field / Relation / validationに関わるか、業務識別条件を誤って変えないか。
- **Function / Interaction**: どのAction、Application Pattern、Driver、Web操作で検査されるか。Target側に独自の同等規則を作らないか。
- **Business**: 不正な単価を見つけたとき、どの担当者が修正を依頼・実施するか。Procedureの現行Referenceで辿れる範囲と、未受理の説明とを区別できるか。

### B実証の最小手順案

1. `reference/workspace`の変更前Sourceとcurrent meaningを読み、対象のData / Functionを特定する。Business Procedureは既存のread-only Referenceと照合し、存在しない関連を推測して補わない。
2. 受注明細の単価Validationを意味変更の候補として作る。生成AI・Studioの利用は必須にせず、file-nativeな変更からでも同じ結果を得られることを優先する。
3. `ISSUE-000071`の承認に足る条件に沿って、変更内容、影響、未解決、対象外のnative code、Sourceを人が読める形で示す。表示していない影響を「影響なし」としない。
4. preview / rejectを非変更操作として確認する。acceptを行う場合は、current base revisionと受理権限を明示し、staleな候補を黙って適用しない。AcceptedなSourceだけを正本としてreloadする。
5. `ISSUE-000073`の論点に沿い、Validationの適用対象がcurrent Canonicalから導かれているか確認する。Driverで正常な単価と不正な単価の双方を実行し、成功・拒否が意味上説明できるか確認する。
6. 第一Web Targetで同じ業務規則が働くかを確認する。Webで不足を埋める独自Validationを正本とせず、Target固有の入力検査があれば別物として記録する。
7. Data / Function / Businessの三視点で、影響を追跡できた箇所とできなかった箇所を示す。業務手順の受理・production昇格は `ISSUE-000042` / `000048` / `000049` の判断に従い、B成功を理由に飛ばさない。
8. 使用した実行環境、syntheticかactual Providerか、CommandとObservationの関係、未確認事項をまとめる。historical Gate B/CのEvidenceを更新したことにはしない。

### Bの具体的な実装候補と確認待ち（2026-10-08）

**Repositoryの静的な読解で確認した現状。** `ISSUE-000073`は2026-10-06に完了済みで、`order_entry`の`uses validation`、Driverの`ResolveApplicationValidations`、Web Targetの評価対象の導出は既に実装されている。候補Bはこの修正を新規にやり直さず、**新規則を追加したときに実際の入力全体で働くか**を問う。

一方、既存のShared Acceptanceの`resolveCandidate`は、`add-application-validation-use`で参照するValidationが変更前Worldに存在することを要求していた。そのため`add-validation`と利用宣言を**一つのcandidate**にまとめると、候補内で追加したValidationでも`VALIDATION_NOT_FOUND`となる。この欠落は、業務規則の追加からApplicationでの評価までを一つの意味変更として扱ううえで障害となる。

さらに第一Web Targetの`commandValidationSource.Number`は、order lineが複数あっても**先頭行しか数値を読んでいない**。既存の`quantity_positive`も、2行目以降の違反を見逃し得る。これは新しい業務規則だけの問題ではなく、現行の実務シナリオそのものに関わる欠落である。

[Draft PR #985](https://github.com/hat47x/tei-platform/pull/985)では、次の修正と回帰テストを**未検証の候補**として追加した。

- 同一candidateに含まれるValidation追加を解決段階で認識し、対応するApplicationの`uses validation`と一度にPreview / Acceptできるようにする。未定義参照や重複追加は引き続き拒否する。
- `positive`な明細Validationは、Command実行前に全明細へ適用する。`relation-unique`のように集合全体を扱うValidationは従来どおり一回で評価する。検査不能な明細を成功扱いしない。
- scratch workspaceで単価の正値制約を追加し、正常値と**2行目の不正値**についてDriver・Web経由の結果を回帰テストする。既存ReferenceのSourceは変更しない。

**追加の実証候補（まだ実行していない）。** #985では、第一Web Targetの回帰テストから`reference-shared-acceptance`のCLIを起動し、scratch workspaceへ`revision → preview → review → accept`を行った後、同じscratch Sourceを通常Loaderで読み直し、DriverとHTTPを評価する一続きのtestを追加した。機械的に作ったreview fingerprintはscratchのテスト専用であり、人間が内容を承認した証拠ではない。2行目の不正な単価・数量、欠落した値、空の明細集合を区別し、副作用へ進ませないnegative caseも用意した。Business Procedure ReferenceのBP-05は独立したauthoring fixtureであり、新しいValidationの追加だけでBP-05が自動更新されたとは主張しない。

現時点ではGitHub連携を通した静的調査とコード変更までで、trustedなcheckout上の`tools/repository-health-check.sh`、実際のPreview/Accept/HTTPテストは未実行である。**PRがDraftであること、コードが書かれたこと、テストが追加されたことは、動作が成立したEvidenceにはならない。** exact-headのfull PASSとpre/post cleanが得られるまで、コードを`main`へ統合しない。

### Bでまだ接続できていない三視点の境界（2026-10-08）

候補Bの事前実装は **Data → Function / Web** の一部を対象にしており、**Business Procedureへの自動同期や利用者による採用判断を実証したわけではない**。比較対象を曖昧にしないため、現在のSource上で参照できる位置を固定する。

| 視点 | 参照するSource / identity | #985で準備した範囲 | 未実証・未採用の範囲 |
| --- | --- | --- | --- |
| Data | `semantic/sales/validation.tei`、`sales.order_line.unit_price`、`sales.order.lines` | scratch上で`sales.order_line.unit_price_positive`を既存Sourceへ追加し、Preview/Accept後に再読込するテスト | この制約を通常のReference Workspaceへ採用する人間の決定 |
| Function / Interaction | `applications/order-entry/application.tei`、`order_entry`のValidation選択、`order_entry.order_lines`、`order_entry.submit_order` | Canonical選択→Driver→Webで全明細を評価。2行目の値が不正・未取得・空の場合を区別。業務不成立と評価不能を別の結果として扱う | 実Database・実Command Provider・実利用者の操作観測 |
| Business Procedure | `reference/business-system-design/order-entry-business-procedure.json`の`BP-03`・`BP-04`・`BP-05`・`BP-07`・`BP-08` | read-onlyで元のidentityと手順の接続箇所を照合できる。BP-04の人による判断、BP-08の外部状態確認は別の行為として残る | 手順のauthoritative受理・自動再導出、担当者や権限の確定 |

**現時点で確認できる具体的な差分:** BP-05の`validations`は既存の`sales.order_line.quantity_positive`と`sales.order.lines.line_number_unique`の**2件のまま**である。scratch上のApplicationへ単価制約を追加するとDriver側の選択は3件になるが、BP-05は`artifactRole: reference-authoring-fixture`かつ`canonical: false`なので自動変更されない。これを三視点の一致と呼ばず、**Procedureのread-only解釈とChange Reviewに残すべき未解決の差**として扱う。新規則を手順へ無断で書き足して見かけの一致を作らない。

この差の判断は既存の`ISSUE-000042 / 000048 / 000049`に戻し、Shared Acceptanceの`Review`が網羅的なBusiness Procedure影響を推論できるとは見なさない。実証Bでは、手順を参照した担当者が「機械的な制約の追加はBP-05の判断をどう変えるか」「BP-04の人の判断とどこが異なるか」を説明できるか確認する。説明ができない場合は **Bを完了にせず**、必要な最小Projectionの欠落を記録する。

### #985の検証実行と未達条件

下記は、当該PRの**exact head**を取得したclean checkoutで実行するためのコマンドであり、成功した実行記録ではない。GitHub Actionsを使わない既存の`tools/repository-health-check.sh`を優先する。

```bash
git status --porcelain
git rev-parse HEAD
go test -count=1 ./tools/reference-shared-acceptance -run '^TestPracticalRuleChange'
go test -count=1 ./tools/reference-first-web-target -run '^(TestPracticalBusinessRuleChange|TestReferenceCommandScenario|TestCommandValidation)'
tools/repository-health-check.sh
git status --porcelain
git rev-parse HEAD
```

受入に必要なのは、少なくとも次の区別が成り立つことである。

- **完全に評価可能・規則不成立:** `validation-not-satisfied`として後続のAuthorization / Command / Observationへ進まない。
- **一部の明細を評価できない:** 前の明細が不正でもExecution Errorを優先し、完全な業務判断が済んだと見せない。
- **正常に評価可能・規則成立:** Commandへ進める。ただし合格は本番の外部状態の成立を意味しない。
- **BP-05との関係:** 選択済みValidationの差が、read-onlyなBusiness Procedure Reference上では未反映であると明示される。

`go test`の成功だけでは、人間によるReview、実Provider、手順のAcceptance、Studio画面での修正、Production適用を証明しない。最初の実証Aの記録とGate B/Cのhistorical Evidenceは引き続き別物として保持する。

#### 受入以前に固定する回帰条件（#985、静的レビューのみ）

| 入力・変更候補 | 期待する結果 | 誤って成立させてはならない状態 |
| --- | --- | --- |
| 同じ候補に単価Validationの追加とApplicationの利用宣言 | Previewで2つのSourceへの変更を可視化し、review fingerprintに結び付いたscratch Accept後にDriverが3規則を解決 | 追加した規則が評価されないままCommandへ進む |
| Validationの未知のAttribute / Namespace、未知の利用先、重複操作 | Shared Acceptanceで拒否し、Sourceは不変 | 規則と適用宣言が別々に受理されたように見える |
| PreviewからAcceptまでにReview fingerprintが不一致になる | scratchの二つのauthoritative Sourceをどちらも書き換えず拒否 | 規則だけ、または利用宣言だけが部分的に受理される |
| 2行目の単価・数量が正値でない | `validation-not-satisfied`、CommandとObservationはなし | 最初の1行だけを見て確定する |
| 前方の行は業務不成立だが後方の行の値が欠落・評価不能 | Execution Error（不完全な評価）、後続副作用なし | 不完全な入力を確定的な業務拒否と呼ぶ |
| 明細が空、Relation keyがnull・空文字列・空白のみ | Execution Error、後続副作用なし | 空集合や欠落した業務識別値を「一意である証拠」とみなす |
| Web経由のsubmit | Driverと同じCanonicalの規則選択と評価結果を返す | Web固有に同じ規則を二重管理する |

この表は**期待結果を定義したテストの範囲**を示し、実行PASSを意味しない。Sourceの内容検証をするテストでも、scratchのAcceptには`reviewFingerprint`を指定し、Reviewを通らない近道を正規の利用経路として例示しない。技術的なfingerprint一致は、実際に人が承認した証拠にはならない。

**実行・統合の停止条件:** 当該PRのexact headで対象テストとRepository health checkが成功し、検証前後のclean状態とrevisionを記録できるまでDraftを維持する。並行作業で`main`が進んだ場合、差分・競合・qualificationの条件を再評価する。既存のReferenceのauthoritative Source、historical Gate Evidence、Product Requirementの採用状態は、この準備によって変更しない。


### 2026-10-08 並行変更の取り込みと統合資格

この候補Bの事前実装はDraft PR #985の同一作業ブランチに保持する。並行開発で進んだ`main`のコミット`5d5731decbc876435d6ee6c35fccbaf1386a9c6a`を、親を二つ持つ通常のmerge commit `d32d7148701a9b830f5b79b1d8ad67bbd7cb48c7`で**作業ブランチへ取り込んだ**。これはmainへ#985の変更を統合したという意味ではない。

三者比較（以前取り込んだmain / 新しいmain / #985）では、候補側の8ファイルのうち両系統で変更があったのは`tools/reference-shared-acceptance/README.md`だけだった。新しいmainに追加されたHTML Review Projectionの説明を失わず、#985の「同じcandidateでValidationと利用宣言を追加する」説明を残した。merge後、PR側は新しいmainからbehind 0、差分は8ファイルであることをGitHub上で確認した。

この統合は**Git上の履歴・内容の整合**を確認したもので、Goのコンパイル、CLI / HTTPシナリオ、Repository health checkの成功を意味しない。資格確認は、**新しいmerge commitを含むexact head**で改めて実行し、その後mainが進んだ場合は再度差分確認を行う。候補Bのowner選択、人間による実際のAcceptance、Business Procedureの採用、旧Gate Evidenceの改訂も発生していない。

### Bの判定と停止条件案

**成立と判断できる範囲:** 変更が一つの正本へ受理され、reload後にDriverとWebが同じ意味を示し、人がData / Function / Businessへの影響と未確認を理解できること。少なくとも正常系と拒否系の両方を実行し、実装側に複製された規則の有無を確認する。

**成立していないと判断する例:** WebとDriverでValidationの選択が異なる、人間向けProjectionが未解決を隠す、Procedureとの接続が文章上の推測だけ、正本へ戻らずWeb固有の修正で通した、syntheticな結果を実Providerの成功と呼ぶ。

**止め方:** 以上を満たせない場合、新しい汎用Semantic概念やAI専用Runtimeを増やす前に、不足した一つの接続に絞って既存Issueへ戻す。成立した場合も、より高度なPlugin self-hosting、マルチTarget、AI Business Actorを自動的な次の着手条件にしない。

この追加評価は、業務実証の候補Bを具体化するものであり、ownerによる候補Bの選択、Studio UI完成、Business Procedure production採用を確定するものではない。

## 設計詳細

1. **記録の形。** 実証ごとに、通したシナリオ、使った機能、機能の間の関係（同じidentityで使われた意味など）、通せなかった箇所、使ったProviderと環境（synthetic / 実Provider）を、1つの記録にまとめる。置き場と形式は、最初の実証で決める。
2. **既存Evidenceとの関係。** Gate B / Gate Cは、それぞれのexact revisionに固定されたhistorical Evidenceとして保つ。実証はそれらを書き換えず、新しいrevisionでの記録として積む。
3. **足りない機能の扱い。** 足りない機能は、実証の失敗ではなく、次に作る機能の根拠として記録する。実証のために、足りない意味を先にCoreへ入れない。

## 完了条件

- [x] 最初に行う実証の候補を、ownerが選ぶ（2026-10-05、Aを選択）
- [x] 選んだ実証を、本Issueの「実証で見せること」の三観点で記録として残す（通せた範囲、通せなかった範囲を含む。下記「実証記録A」）
- [x] 記録に、使ったProviderと環境（synthetic / 実Provider / 本番相当）を明示する（下記「実証記録A」）
- [x] 記録から、次に作る機能の候補を挙げる（下記「実証記録A」。候補Bへ進むかはowner判断待ち）

## 完了後に判断できること

- 積み上がった機能が、実務に近いシナリオを通すのに足りているか、どこが足りないか
- 次に作る機能の優先順位
- 研究に留保している測定（[ISSUE-000072](000072-rule-change-hand-edit-count.md)）を再開できる程度に、問題領域の知見が蓄積されたか

## 非主張範囲

- 一つのシナリオの成功は、実務全般への適用可能性を示しません。
- 実証は、費用削減や生産性向上を示しません。
- test doubleやsynthetic fixtureでの成功を、実Database・実Provider・本番運用の保証にしません。
- 候補の整理は評価待ちで、どれを選ぶかは決まっていません。
