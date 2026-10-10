# ISSUE-000075: 実務シナリオを最初から最後まで通すMVP的な実証を先に行い、機能の積み上げと関係性を示す

- 状態: In Progress — 実証Aは2026-10-05にownerが選択し、synthetic環境での実行を記録済み。実証Bでは2026-10-08に業務規則の変更がownerによって受理された。ただし受理前のscratch proofと、受理後のWeb画面を含む一周の実証を混同しない。Cは未着手
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

## 実証候補（Aは実行記録あり。Bは規則変更を受理済み、一周の実証は未完了。Cは未着手）

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

## 実証記録B — 変更を提案し、受理し、画面の確定へ反映される一周（受理済み、2026-10-08）

**減らす不確実性。** 変更を提案し、人が読んで確かめ、受理し、評価と確定操作へ反映されるまでの一周が、一つのCanonicalな意味の上で通るか（主担当要求 RQ-021）。

**状態。** 受理済み（2026-10-08）。受理の根拠は、この会話でリポジトリのownerが行った回答である（下の「受理の記録」を参照）。受理により、`reference/workspace`の`order_entry`の宣言と`semantic/sales/validation.tei`、`reference/world.json`、決定記録一件が変わった。受理の記録は下の「受理の記録」を参照する。以下の証明のうち受理前の部分は、workspaceのcopy（scratch copy）の中で行った scratch proof である。この受理はGate BまたはGate Cのevidenceではなく、既存のhistorical evidenceも書き換えていない。

**reference/との差分と比較の基準。** 比較の基準は統合時点の`origin/main`（`d3aab052`）である。ローカルの`main`は`origin/main`と一致しないことがあるため使わない。このブランチには、そのmainを通常のmerge commitで取り込んだ。取り込んだmainの変更には、`reference/`配下のうち`business-system-design`と`plugin-self-build`の文書とWorkspaceが含まれるが、受理前の`9310d97f`では、`reference/workspace`と`reference/world.json`は1d8fcc95、`d3aab052`、このブランチのHEADで同じだった。受理後は、この二つを変えたのは受理commitだけである。revisionは`sha256:4234d86eba95e86a1bf0fd34df7c896f095cca1aa10dc5650d9ddd4dc2b08db4`のままである。取り込み後のtreeでpreview / reviewを読み取りだけで再生成すると、既存の`preview.json`と`review.txt`とbyte単位で一致し、fingerprintも変わらない。

**完了条件のチェック。** 実証記録Bの作業では、完了条件のチェックを一つも付けていない。完了条件にある四つのチェック（2026-10-05）は実証記録Aに属し、この記録Bの受理と画面の確定までの一周を完了させるものではない。

**変更の題材。** 受注明細の行番号（`sales.order_line.line_number`）は1から始まる。0以下を業務上の不成立とする規則`sales.order_line.line_number_positive`（kind `positive`）を一件足す。対象を`sales.order_line.unit_price`ではなく行番号にしたのは、`unit_price`のIDが`_test.go`の9ファイルに計31回（出現回数）使われており、今回の変更の影響を切り分けにくいためである。`sales.order_line.line_number_positive`は、この変更以前のリポジトリで使われていないことを`grep`で確かめた。

**候補。** `plan/issue/000075-practical-b/candidate.json`（schema `tei.reference.change-candidate/v0`、id `cand-order-line-number-positive`、origin `ai` / `claude-haiku-5-5`、基準revision `sha256:4234d86eba95e86a1bf0fd34df7c896f095cca1aa10dc5650d9ddd4dc2b08db4`）。deltaは次の二つである。

1. `add-validation`：namespace `sales`、kind `positive`、attribute `sales.order_line.line_number`、placement `semantic/sales/validation.tei`
2. `add-application-validation-use`：application `order_entry`、validation `sales.order_line.line_number_positive`、placement `applications/order-entry/application.tei`

**候補のintent（審査面に出る文）。** 受注明細の行番号は1から始まるため、0以下を業務上の不成立とするValidationを一件追加する。この規則は、Canonicalな意味では受注明細ごとに適用される。ただし現在のCommand経路は、読み取った一覧の最初の受注明細だけを評価する（既存のquantity_positiveと同じ制限）。この候補は、全明細への評価を広げることを主張しない。

**人が読む面（read-only Preview）。** `plan/issue/000075-practical-b/preview.json`と`review.txt`を、上のintentに書き直した後に、実際の`reference/workspace`に対して作り直した。読み取りだけで、`reference/`のファイルは変えていない。Previewの全stageはpassで、Reviewは「受入可能」、coverageは表示可能3 / 一部5 / 未評価0である。

- candidate fingerprint: `sha256:da12cd6346a7e02b81359ec95f7a52709be1a9898329995bd7d7f0fadc869f51`（intentの変更の前後で同じ。candidate fingerprintはOrigin / Intent / Evidenceを含まない（`candidate.go`）ため）
- evidence fingerprint: `sha256:54eb30be346e118b2aa6d6115cf82dd7235205726260439ca5ad56bef7104c5b`（変わらない）
- review fingerprint: `sha256:9eb0f1ed5513820a03857f148981ce7678e91af4cd486212f3249b3046c4babb`（intentを含むReview Projectionへのbinding。受理するときはこの値を指定する。fingerprintは人が読んだことや承認そのものを証明しない）

intentの変更はcandidate fingerprintには現れず、review fingerprintにだけ現れる。承認の面で読まれる文面が変わったかは、review fingerprintで確かめる。

**Shared Acceptanceの修正。** 候補を最初にPreviewしたとき、`resolve`が`VALIDATION_NOT_FOUND`で拒否した。`uses validation`の参照先を、変更前のWorldにだけ探していたためで、同じcandidateで追加するValidationを参照できなかった。ADR-0029の確かめ方1は、規則を一件足し、宣言を一行足すだけで評価されることを求めている。そのため`tools/reference-shared-acceptance/resolve.go`を変更し、同じdeltaの`add-validation`を参照先として認めるようにした。同じcandidateの`add-validation`で追加される規則を参照しているかは、`resolve.go`が、追加するValidationのIDとの完全一致（exact-ID membership、`deltaValidations`）で確かめる。overlay適用後の共有validate（`internal/validation`の`UNKNOWN_APPLICATION_VALIDATION`、`internal/validation/business_validation_reference.go`）は、これとは別の確認で、多層防御として残している。同じcandidateの経路を通すtest（`same_candidate_validation_use_test.go`）は、受理されることを確かめるだけで、この経路でvalidateが`UNKNOWN_APPLICATION_VALIDATION`を返す場合は通していない（その分岐は`internal/validation`のtestが単独で確かめる）。どのcandidateにも無い規則の宣言は、従来どおり`resolve`が`VALIDATION_NOT_FOUND`で拒否する（同じtest）。この修正は受理の境界に触れるため、owner の確認が要る。

**scratch copyでの受理。** 新しいreview fingerprintを`-expected-review-fingerprint`に指定し、`mktemp -d`の下に作ったcopyに対してacceptした（`-allow-reference-baseline`は使っていない。copyの場所は`reference/workspace`で終わらない）。結果はacceptedで、変わったのは次の三つだけである。

- `semantic/sales/validation.tei`：validationを一件追加した
- `applications/order-entry/application.tei`：`uses validation sales.order_line.line_number_positive`を一行追加した（既存の最後の`uses validation`の後ろ）
- `.tei/decisions/`：決定記録が一件増えた（Canonical sourceではない）

受理後のderived world JSONを、コミット済みの`reference/world.json`と比べると、二つの配列に一件ずつ追加がある（`semantics`のsales側の`validations`に`sales.order_line.line_number_positive`の定義、`order_entry`の`validations`の末尾に同じIDの宣言）。ad hocな`diff`では、この二箇所以外の差分は見えなかった。この比較は受理を試したときの確認であり、テストでは固定していない。

**Go testが示したこと。** `tools/reference-first-web-target/practical_b_test.go`は、上の操作をShared Acceptance CLIで行い（`main` packageはimportできないため、CLIをbuildして実行する）、受理後のcopyを`workspace.Load`で読み直して、次を確かめた。

- a. `ResolveApplicationValidations("order_entry")`は、`quantity_positive`、`line_number_unique`、`line_number_positive`の順に三件を返す。
- b. 既定の一覧行（RowIndex 0、2026-09-18T08:30:00Z）のCommand runnerは`completed`になり、Validationは三件、すべて`satisfied`になる。
- c. 最初の受注明細の行番号だけを0にした読み取り結果では、stageは`validation-not-satisfied`になる。`quantity_positive`と`line_number_unique`はsatisfied、`line_number_positive`はnot-satisfiedである。数量は既定の正の値のまま残しているため、失敗するのは新しい規則だけである。
- d. 受理後の実際の`reference/workspace`を読み直すと、`order_entry`の宣言は三件（`quantity_positive`、`line_number_unique`、`line_number_positive`）になる。このtestは、読み直しの前後でrevisionが変わらないことも確かめる。

`internal/driver/practical_b_line_number_test.go`は、Driverの確認用sourceが行番号を数値として渡すことを確かめる。行番号1はsatisfied、0はnot-satisfiedで、行番号が無いときは業務結果ではなくエラーになる。

**この実証の位置づけ。** ここまでの証明は、`mktemp -d`の下に作ったcopyの中だけで行った scratch proof である。その後、同じ候補を実際の`reference/workspace`へ受理した（下の「受理の記録」を参照）。ここまでの証明は、その受理の前の状態について、scratch copyの中で行ったものである。画面の既定データ（`read_scenario.go`の`find_by_order`のfixture）の行番号は、どの注文でも1と2だけで、すべて有効な値である。そのため、実画面では、この規則の行は satisfied としてだけ現れ、不成立は見えない。不成立は、読み取り結果を差し替えたtestでだけ観測された。Command評価は、読み取った一覧の最初の受注明細の行番号だけを見るため、この規則はまだ全明細へ適用されていない（下の「owner が決めること」3を参照）。

**受理の記録。**

- 受理: 2026-10-08。この会話で、受理するかどうかの質問に対し、リポジトリのowner（hat47x）が「受理する（推奨）」と回答した（Claude Code sessionの質問への回答）。
- 同じ質問への回答: `resolve.go`の緩和は「残す（推奨）」、評価範囲は「先頭明細のみ（推奨）」、ADR-0029の残骸の行は「削除する（推奨）」。
- 直前に提示した内容: 規則`sales.order_line.line_number_positive`（kind `positive`、attribute `sales.order_line.line_number`）、先頭の受注明細だけを評価する制限、candidate fingerprintとreview fingerprint、受理で起きること（`reference/workspace`と`reference/world.json`が変わり、現在の`main`ではGate Cのarchitecture conformance toolが停止し得ること）。
- 提示していないこと: `review.txt`の全文と、部分的にしか確かめられていないsection（A2 影響、A3 例外・診断、A5 権限の変更、A6 native code、A7 逆追跡）の詳細。ownerが全文を読んだという記録はない。
- 手続き: `accept`を`-expected-review-fingerprint sha256:9eb0f1ed…`と`-allow-reference-baseline`（baseline保護の上書きを明示）付きで実行した。決定記録のoriginは候補の`ai` / `claude-haiku-5-5`であり、ツールは承認者を記録しない。`decisions`の照会では`decisionActorStatus`が`unresolved`になる。承認の根拠は、上の回答であり、決定記録ではない。
- candidate fingerprint: `sha256:da12cd6346a7e02b81359ec95f7a52709be1a9898329995bd7d7f0fadc869f51`。review fingerprint: `sha256:9eb0f1ed5513820a03857f148981ce7678e91af4cd486212f3249b3046c4babb`。
- revision: 受理前は`sha256:4234d86eba95e86a1bf0fd34df7c896f095cca1aa10dc5650d9ddd4dc2b08db4`、受理後は`sha256:f414e2b09d2f6ea5d0f5f282aea55612985c0e8155877ad8d310250633cf3fd2`。
- 決定記録: `reference/workspace/.tei/decisions/a043abc84702feb4a4558e2fbfd6fa17bd60867075f65d126d648998a11f2523.json`。
- 変わったもの: `applications/order-entry/application.tei`（`uses validation`を一行）、`semantic/sales/validation.tei`（validationを一件）、`reference/world.json`（派生物）、決定記録一件。受理後のConformanceは七件すべてpassedである。
- このbranchに含まれる、受理より前からのコード変更: `tools/reference-shared-acceptance/resolve.go`（同じcandidateで追加する規則を`uses validation`から参照できる緩和。`deltaValidations`）、`internal/driver/practical_b_line_number_test.go`、`plan/issue/000075-practical-b/`（候補、preview、review）。
- 期待値の更新: 宣言数が二件から三件になったため、宣言数に依存するtestの期待値を同じbranchで直した。assertionは弱めていない。
- ADR-0029の影響節は「Evidence auditの期待値（Validation 2件）は変えない」と述べている。この受理後は成功経路の期待値が3件になっている。採用済みADRの本文は書き換えず、この記録で補う。
- 残る制限: Commandの評価は、読み取った一覧の最初の受注明細だけを見る（`quantity_positive`と同じ制限）。全明細への適用は、この受理では主張しない。
- 2026-10-09追記: 上の「先頭明細のみ」は受理時点のrevisionの記述であり、owner判断（2026-10-08）の経緯として残す。その後 main の `6809b947`（fix: evaluate line-level validation across all order lines）で、positive kindのCommand評価は読み取った全明細に適用されるようになった（`evaluateCommandValidation`、`TestReferenceCommandScenarioEvaluatesPositiveRuleOnEveryOrderLine`）。全明細化の判断をownerが改めて確認したかどうかは、この追記からは分からない。

**REF-SPEC-01との接続（2026-10-09、静的なSource照合のみ）。** 受理されたApplication側のValidationは3件となった。一方、現時点のBusiness Procedure authoring fixture（BP-05）は、数量の正値と行番号の一意性の2件を明示したままである。BP-05はCanonical Worldの第二正本ではなく、ここから直ちに業務上の不整合を断定しない。ただし、三視点をまたぐ変更影響として**人が確認すべき差**であり、現行checkerがこの差を自動検出したとも主張しない。既存Sourceと診断を使った読み取り専用の確認案は[技術リサーチ統合方針 §9.5](../TECH_RESEARCH_INTEGRATION_2026-10.md)で扱う。

**証明していないこと。**

- owner による受理の事実の検証。受理の根拠は、この会話での回答であり、ownerが`review.txt`の全文を読んだという記録はない。
- Gate BまたはGate Cの Evidence。既存のhistorical evidenceは書き換えていない。
- HTTPとブラウザの画面。ここで通したのは、第一Web Targetの確定経路が使うCommand runnerを、同じworldに対して直接呼ぶところまでである。
- 実画面の既定データでの不成立。画面の既定データ（`tools/reference-first-web-target/read_scenario.go`の`find_by_order`のfixture）の行番号は、どの注文でも1と2だけである。そのため、行番号の観点では、実画面で新しい規則はsatisfiedとしてだけ現れる。0による不成立は、読み取り結果を差し替えたtestでだけ観測された。
- 全明細への適用。Commandの評価は既存の仕組みどおり、読み取った一覧の最初の受注明細だけを見る（`commandValidationSource.Number`）。`quantity_positive`と同じ扱いである。

**owner が決めること。**

1. 2026-10-08に受理した（上の「受理の記録」）。
2. 2026-10-08に「残す（推奨）」と回答した。`resolve.go`の緩和は、Shared Acceptance Boundaryの変更として残る。
3. 2026-10-08に「先頭明細のみ（推奨）」と回答した。全明細への適用は、評価の形を変える別の提案として扱う。

受理は、Gate Cのarchitecture conformance toolを現在の`main`で停止させ得る。後継Handoverを今は発行しないという判断は、2026-10-06にownerが行っている（[ISSUE-000073](../issue/000073-derive-validation-selection-from-canonical.md)）。

**再現。**

```text
go run ./tools/reference-shared-acceptance preview -workspace reference/workspace -candidate plan/issue/000075-practical-b/candidate.json -out plan/issue/000075-practical-b/preview.json
go run ./tools/reference-shared-acceptance review -workspace reference/workspace -candidate plan/issue/000075-practical-b/candidate.json -format text -out plan/issue/000075-practical-b/review.txt
go test ./tools/reference-first-web-target/ -run PracticalB -count=1 -v
```

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
- 実証Aの選択・実行記録と、実証Bの規則変更の受理は記録済みです。Bの一周の実証、Cの着手、将来の要件・Targetの採用判断は未完了です。
