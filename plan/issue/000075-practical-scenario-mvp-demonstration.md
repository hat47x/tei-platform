# ISSUE-000075: 実務シナリオを最初から最後まで通すMVP的な実証を先に行い、機能の積み上げと関係性を示す

- 状態: In Progress — 最初の実証はA（2026-10-05にownerが選択）。Aの実装と実行確認までを記録した。B / Cは未着手
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
