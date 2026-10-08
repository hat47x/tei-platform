# ISSUE-000072: 業務規則の変更に伴う手直しの記述方法を研究する（価値判断には使わない）

- 状態: Proposed — 研究に留保。価値の判断に使わず、問題領域の知見が蓄積されるまで実測も行わない。2026-09-30の起票時は「手直し箇所の数」を価値の指標にする案だったが、2026-10-02に位置づけを改めた
- 提案日: 2026-09-30（位置づけの改訂: 2026-10-02）
- 性格: 研究 / Reference Evidenceの記述。Product Valueの判断根拠にしない
- 主担当要求: RQ-022（主）/ RQ-017 / RQ-003
- 関連Issue: [ISSUE-000073](000073-derive-validation-selection-from-canonical.md)（この研究の読解で見つかった、正本の外にある意味の置き場を解消する）、[ISSUE-000039](000039-business-change-rehearsal.md)（差分・依存解析）、[ISSUE-000052](000052-sui-reference-adoption-and-plugin-surface-evaluation.md) / [ISSUE-000070](000070-sei-reference-adoption-change2-resumption.md)（同種の比較。扱いは下記「他Issueとの関係」）
- 価値との対応: [Product Value](../../product/value/TEI_PRODUCT_VALUE.md)「価値の確かめ方」。測定に基づく価値判断は留保し、積み上げた実務機能の量・関係性・実用性とMVP的な実証で確かめる
- Gate impact: None

## この文書の役割

このIssueは当初、Product Valueの確かめ方の一つとして「一つの業務規則を変えるとき、人が手で直す箇所の数」を、同じ方法で繰り返し数える手順を定める案でした。

2026-10-02に、ownerから次の指摘がありました。手直しの量は業務要件の変更内容によって決まり、要件固有である。全体へ横串で同じ修正を入れるだけの変更と、ロジックの深い部分に及ぶ変更とでは意味がまるで違う。このため、箇所数を価値の指標にするのは無理がある。

この指摘を受け入れ、次のとおり位置づけを改めます。

- 手直しの箇所数を、TEIの価値の指標にしない。価値の判断にも使わない。
- 測定に基づく価値判断は、問題領域の知見が十分に蓄積されるまで留保し、研究にとどめる。
- 価値は、積み上げた実務的な機能の量と、その有機的な関係性・実用性で確かめる。その実証は、MVP的・プロトタイプ的に先に行う（[Product Value](../../product/value/TEI_PRODUCT_VALUE.md)「価値の確かめ方」）。
- このIssueには、研究の素材として、変更がどこへ及ぶかを記述する語彙と、静的な読解で見つかった事実の記録を残す。

## なぜ手直しの箇所数を価値の指標にしないか

- **要件固有である。** 規則を足す、既存の規則の条件を変える、状態遷移の途中に新しい判断を差し込む、といった変更は、そもそも必要な作業が違う。箇所数は、そのApplicationの要件の形を映すだけで、TEIの良し悪しを示さない。
- **変更の深さで意味が変わる。** 横串で同じ修正を入れる変更は、箇所が多くても機械的である。ロジックの深い変更は、箇所が少なくても判断が重い。同じ「三か所」が正反対の負担を表し得る。
- **代理指標が目標になる。** 箇所数を減らすこと自体が目的になると、数を減らすために、見えにくい形へ意味を寄せる誘因が生まれる。
- **一回の計測は一つの要件の記録にとどまる。** 要件の種類と変更の種類が蓄積されていない段階で数を出しても、比べる相手がない。

## 他Issueとの関係

[ISSUE-000052](000052-sui-reference-adoption-and-plugin-surface-evaluation.md)（SUI）、[ISSUE-000054](000054-sei-reference-adoption-resume-review.md)（SEI）、[ISSUE-000070](000070-sei-reference-adoption-change2-resumption.md)（SEI）は、change amplification（同じ意味変更に必要な変更の広がり）の比較を持っています。これらも、この指摘と同じ種類の測定に基づく価値判断を含みます。本Issueでは書き換えず、扱い（研究にとどめるか、比較の目的を限定するか）をownerの判断に残します。

## 記述の語彙（区分）。数えるためのものではない

下の区分は、規則の変更が正本の内と外のどこへ及ぶかを**記述する**ための語彙です。合算して一つの数にしたり、大小を比べたりしません。

| 区分 | 意味 | 例 |
| --- | --- | --- |
| C1 正本の変更 | authoritative sourceに対する、意味を変えるための変更 | `reference/workspace/semantic/sales/validation.tei` |
| C2 機械的な再生成 | 正本から決定論的に再生成でき、人が内容を書かない変更。受理toolが書く記録（`.tei/decisions/`）も含める | `reference/world.json`、golden、`reference/workspace/.tei/decisions/*.json` |
| C3 正本の外にある意味の手直し | 正本と同じ意味を、別の場所で人が直す必要がある変更。構造上の欠落の手がかり | Target側のValidation一覧 |
| C4 期待値の手直し | test・Evidence auditの期待値を人が直す変更 | `*_test.go`、`tools/*-evidence-audit` |
| C5 文書 | 更新が必要な説明文書。`plan/`の監査記録など、過去の事実を残す文書にIDが出てくるだけの場合は対象にしない | `plan/`、`docs/`、利用者が読むREADME |
| C6 観測用のfixture・入力data | 新規則の効果を観測するために、人が書く入力dataやfixture。正本の意味ではないが、観測できるかどうかを左右する | `tools/reference-first-web-target/read_scenario.go`の固定行 |

C3は、正本の外に同じ意味が置かれている箇所で、構造上の欠落の手がかりになります。C4はTEIの外でも必要になるため、C3とは区別して記述します。「どこへ及ぶか」を記述する際には、次の点を併記します。

- 編集の単位（fileと、独立した編集のかたまり）。
- 既存の期待値を直す変更か、新しいtestやfixtureを足す変更か。
- 直し忘れを検出するtestやauditがあるか。検出されない箇所は、数が少なくても軽いとは限りません（例: 業務手順ReferenceのValidation一覧には、網羅を確認する仕組みがありません）。
- 変更が横串の機械的な修正か、ロジックの深い変更か。同じ箇所数でも意味が違うためです。

## 静的読解の記録 — 単価の正値規則を足す変更（2026-10-02。実測ではない）

以下は、一つの変更を題材に、コードの読解だけで変更の及ぶ先を追った記録です。研究の素材であり、価値の根拠にはしません。Goは実行しておらず、予測は未検証です。

**題材の変更:** 受注明細の単価が正であることを、受注確定前に守る業務規則として追加する（`sales.order_line.unit_price_positive`、kind `positive`。対象は正本で`unitPrice`と書かれた属性で、Canonical IDは`sales.order_line.unit_price`）。`tools/reference-shared-acceptance/README.md`のcandidate例と同じ変更で、正本側はShared Acceptance Boundaryの`add-validation`で追加できます。

**机上の予測**（revision `efd50e46`時点の静的な読解。Goは実行しておらず、実測ではない。個々のtest関数が落ちるかどうかは、読んだ範囲の根拠を併記し、確認していないものは「未確認」と書く）:

前提として、変更は`semantic/sales/validation.tei`への追記だけとする（`add-validation`は既存fileの末尾へ追記する。`tools/reference-shared-acceptance/boundary_test.go:82-88`）。`order_entry`は`uses validation`を宣言しておらず（`reference/world.json:672-676`は`semanticUses`のみ）、Applicationのsourceは変えない。

**A. C3（Targetの評価対象一覧）を直さない場合**（ISSUE-000073の前の状態）

| 区分 | 予測される変更 | 箇所（file / 編集単位） | 根拠 |
| --- | --- | --- | --- |
| C1 | `semantic/sales/validation.tei`へ一件追記（空行を含め数行。正確な行数は未確認） | 1 / 1 | `boundary_test.go:85` |
| C2 | `reference/world.json`の`sales`の`validations`配列の末尾へ一件追加。Canonical IDの昇順で`quantity_positive`の後ろに入る。差分は6行追加と、直前要素の`}`が`},`になる1行の変更（`world.json:450-456`からの見積り） | 1 / 1 | `loader.go:617`、`world.json:442-456` |
| C2 | scratch copyで`accept`を使うと、受理記録`<scratch>/.tei/decisions/<sha256>.json`が一件増える。手で`validation.tei`を書く場合は増えない。`.tei/`はmanaged sourceではなくmodel revisionに入らない。実際の`reference/workspace`へ`accept`する場合は、baseline保護のため既定では記録を書かない | 0〜1 / 1 | `main.go:97-101,166-172`、`decision.go:297-330` |
| C2 | `world.json`は`accept -derived-out`か`go run ./tools/reference-golden -write`で再生成する（`-derived-out`は任意で、既定では書かない） | - | `decision.go:66-67,220-229` |
| C2 | `plugin-lock.json`と`reference/order_rollout.durable-run.m0.21.start.json`は、validation一覧を含まない（`plugin-lock.json`に`quantity_positive`は出現しない）ため変わらない見込み（byte一致までは未確認）。Source Indexは`BuildSourceIndex`が実行時に作り、追跡されるgoldenは見つけていない | 0 | `reference/workspace/.tei/plugin-lock.json`、`workspace/source_index.go` |
| C3 | Web Targetの評価対象一覧は直さないので0。ただし未検出のC3が別にある。業務手順Reference`order-entry-business-procedure.json`のBP-05（58-65行）は、確定前に守る決定的な制約としてValidation IDを手で列挙している。ここへ新規則を足すかどうかは業務上の判断であり、文書の追従ではないため、C5ではなくC3として数える。足さなくても誤りを検出するものはない（参照の存在確認だけで、網羅確認がない） | 0〜1 / 1 | `tools/reference-business-procedure-check/main.go:291-293` |
| C4 | Command scenario testとEvidence auditの期待値は、評価されるIDが2件のまま変わらないので、**直す必要がない**（`command_scenario_test.go:52,154,289`、`evidence-audit/main.go:705`は2件のまま通る） | 0 | 同左 |
| C4（例示衝突） | このケースはリポジトリ自身の例と同じ規則である。実際のworkspaceへ入ると、`tools/reference-shared-acceptance`の`unitPriceCandidate`（`helpers_test.go:110-126`）を使うtestは`VALIDATION_ALREADY_EXISTS`（`resolve.go:126-128`）で受理できなくなる。呼び出し元は21関数（`boundary_test.go` 9、`provenance_test.go` 5、`record_test.go` 7）。個々に落ちるかは未確認（fingerprintだけを見るtestは通る可能性がある）。さらに`tools/internal/changeanalysis`の3 test（`analysis_test.go:186`、`:288`、`:345`）は、同じ規則を重ねて追加するため、`Diff`が同一subjectの重複宣言を拒否して落ちる見込み（`diff.go:48-50`、`internal/workspace/loader.go:271`は重複を除かない）。直す箇所は、helper 1、`boundary_test.go`の期待ID（70、85、104-105、157、165行付近）、`provenance_test.go`、`analysis_test.go`の3 testの規則ID（208-219、291-303、348-351行付近）。**この衝突はこのケース固有で、規則変更一般の負担ではない。C4の本計上とは分けて「例示衝突」として別掲する** | 約4 file / 十数か所（見積り） | 左記の行 |
| C5 | 必須の文書更新は0。`tools/reference-shared-acceptance/README.md:68-90`のcandidate例は、受理済みの規則を再び追加する例になる（任意の追従1）。`plan/`の過去の記録にIDが出てくるだけのものは更新対象に数えない | 0〜1 | `README.md:68-90` |

**B. C3を直し、新規則をTargetが評価するようにした場合**（新しいIDを一覧の末尾へ足す想定）

| 区分 | 予測される変更 | 箇所 | 根拠 |
| --- | --- | --- | --- |
| C3 | `command_scenario.go`の一覧2か所に同じIDを足す。**2か所は必ず同時に直す**。86-89行だけを直すと新規則は黙って評価されない。195-198行だけを直すと、`r.validations[id]`（199行）が零値のままで`Evaluate`がkind未対応のerrorを返し、Execution Errorになる | 1 / 2 | `command_scenario.go:86-89,195-198,199`、`evaluator.go:67-68` |
| C4 | `command_scenario_test.go`の`len(result.Validations) != 2`（52、154、289行）を3へ。1件目で止まる204、244行は、新IDが末尾なら変わらない。`evidence-audit/main.go:705`の`!= 2`を3へ（testではなくGate Cの実Evidenceを監査するcodeなので、期待値を変えるとEvidenceの判定基準が変わる。Gate影響の確認が要る）。`evidence-audit/main_test.go`の合成artifact（227-229、309-311行）へ3件目を足す | 3 file / 6 | 左記の行 |
| C4（追加） | 新規則が不成立になる負のtestは既存にないため新規追加が必要（`command_scenario_test.go:176-190`の`staticReadScenarioRunner`と`setSemanticRowValue`と同じ形で、`sales.order_line.unit_price`を`0`にする）。「直す」ではなく「足す」ので、別に数える | 1 | 同左 |
| C3または追加 | BP-05へ足す場合は、同じ期待値を固定している4つのtest（`d4_ref_kind_pressure_test.go:55-57`、`canonical_meaning_pressure_test.go:175-177`、`explicit_ref_authority_pressure_test.go:34-36`、`actor_context_r6b_...pressure_test.go:119-120`）も直す | 1+4 file | 同左 |
| 任意 | `internal/driver`の5つのtest（`submit_order_gate`、`gate_b_unified`、`final_gate_b`、`gate_b_ui_less`、`submit_order_observation`）は、それぞれ2件のIDを手で持つ見込み。確認したのは`gate_b_unified_scenario_test.go:220-221`だけで、他は未確認。新規則を足さなくても落ちず、新規則はDriver側のtestで評価されないまま残る | 0 | `gate_b_unified_scenario_test.go:220-221` |

**C3を直さない場合に、Web Targetは単価が0以下の受注を確定するか。** 確定する。`RunCommand`のvalidation段階は、手書きの2件（`quantity_positive`、`line_number_unique`）だけを順に評価し（`command_scenario.go:193-212`）、両方がsatisfiedなら`authorization`段階へ進む。Canonical Worldに新規則があっても、`r.validations`の構築（86-97行）にもループ（195-198行）にも現れないため、読まれない。結果は、権限があれば`completed`（341行）、`Validations`は2件になる。

ただし、次の2点が設計詳細の3の手順を変える。

1. **画面（HTTP）からは単価0以下の受注を作れない。** `/scenario/submit`が受けるのは`loginId`・`credential`・`from`・`row`だけで（`command_scenario_test.go:336`）、受注明細の行は`read_scenario.go:355-395`の固定値（単価`4000.00`と`4500.00`。変わるのは受注1002の1行目の数量だけ）である。観測するには、`command_scenario_test.go:176-190`と同じ手順で、読込結果の`OrderLineRows[0]`の単価を差し替えるin-processのtestを使うか、fixtureを直す必要がある。fixtureを直す場合は、C6（観測用のfixture・入力data）として数える。
2. **評価されるのは先頭の明細だけである。** `commandValidationSource.Number`は`s.orderLines[0]`だけを見る（`command_scenario.go:392-395`）。C3を直しても、2行目以降の単価が0以下である受注は止まらない。これは`quantity_positive`にも既にある制約で、このケースが「単価が正でない受注を確定できない」ことまで示すとは主張できない。

**このケースでは変わらない（読んだ範囲の根拠）**: `internal/validation/semantic_business_validation_test.go:81-132`は合成Worldである。`internal/workspace/semantic_validation_source_index_test.go`は一時workspaceである。`reference_business_validation_test.go:21-46`と`evaluator_test.go:108-128`はID検索で件数を見ない。`custom_ui_test.go`は固定の結果を返す差替えrunnerである。業務手順のtest（`actor_context_r6b`の候補は`actor_context_r6b_...pressure_test.go:88`でfixtureから作られる）は明示参照だけを使う（以上は、author報告の行番号を一部のみ再確認した。再確認していないものは実測前に読み直す）。

**注意（実際の`reference/workspace`へ受理する場合）**: Gate Cの`architecture-evidence`は、`reference/workspace`と`reference/world.json`がHandover baseline Rから対象Tの間で変わっていないことを要求し、変わっていれば後継Handoverを求める（`tools/reference-first-web-target-architecture-evidence/main.go:117-140`）。scratch copyでの計測はこの検査に触れないが、実受理では別の手続きが要る。

参考として、同じrevisionで既存規則`sales.order_line.quantity_positive`のIDを含むfileは、正本1、生成物1、test以外のcode 2（上記Target scenarioとEvidence audit）、test 22、文書13、その他1（業務手順Reference）でした。これはIDの出現数であり、変更時に直す箇所の数ではありません。

## この記録から分かった、構造上の事実

数ではなく、事実として次を残します。いずれもISSUE-000073などの機能上の課題として扱い、本Issueでは追いません。

- 評価するValidationの選択が、第一Web Targetのコードと、Driverのscenario testに手で書かれている。正本のApplicationの宣言（`uses validation`）は、実行時に読まれていない（[ISSUE-000073](000073-derive-validation-selection-from-canonical.md)、[ADR-0029](../adr/0029-application-validation-selection-from-canonical-declaration.md)）。
- 業務手順Referenceが、同じValidation IDを別に列挙している。足し忘れを検出する仕組みがない。
- Web Targetは先頭の明細だけを評価する。これは既存の数量規則にも共通する制約である。
- 計測の題材に選んだ規則が、リポジトリ自身のtestやREADMEの例と同じだった。題材の選び方が結果を左右する。

## 後続の実装候補との対応（2026-10-08、元の読解記録は維持）

上記の行番号・C1〜C6の予測と「先頭の明細しか評価しない」という記述は、**当時読んだrevisionに対する静的な観察**であり、最新実装の成立を主張するものではない。履歴上の観察を遡って書き換えず、後続の変更候補を次のとおり区別する。

- [ISSUE-000073](000073-derive-validation-selection-from-canonical.md)は既にDone。現行のApplication Validation選択をCanonicalから導く経路が追加され、当時の「Targetに手書きの選択一覧が残る」という論点には対応済みの実装がある。
- [Draft PR #985](https://github.com/hat47x/tei-platform/pull/985)では、同じ候補でValidationとApplicationの利用宣言を追加し、複数明細の全行を評価する修正・scratch Preview / Accept / reload / Driver / Webの回帰テストを準備した。空・null・空白の業務識別値と、後続行の評価不能を正常な業務拒否で覆わない条件も含む。ただし**exact-headのGo実行・Repository health checkは未実行**であり、修正済みと断定しない。
- `reference/business-system-design/order-entry-business-procedure.json`のBP-05は、当時と同様に非Canonicalなauthoring fixtureとして既存2規則を記載する。scratch上に追加する単価制約はBP-05へ自動反映されない。この差は[ISSUE-000075](000075-practical-scenario-mvp-demonstration.md)の実証B案の三視点照合へ戻し、Referenceに無断で新規則を追加しない。

これらは本Issueの研究再開条件を満たしたことを示さない。**変更箇所数をProduct Valueの代理指標に戻さない**というowner判断も変更しない。

## 研究を再開する条件

次がそろうまで、実測は行いません。そろった後も、変更の種類（横串の機械的な修正か、ロジックの深い変更か等）ごとに分け、単一の数に畳みません。

1. MVP的・プロトタイプ的な実証で、複数の実務シナリオと、複数の種類の変更が積み上がっている。
2. 変更の種類を、要件の側から分類できるだけの知見がある。
3. 再開するかどうかを、ownerが判断する。

## 完了条件

このIssueは研究として保持します。期限や完了を、Productの判断に結び付けません。

- [ ] 区分の語彙（C1〜C6）を、記述の語彙として使うかをowner reviewで判断する
- [ ] ISSUE-000052 / 000054 / 000070のchange amplification比較の扱いを、ownerが判断する

## 非主張範囲

- この記録は、TEIの変更コストが小さいことも大きいことも示しません。
- 手直しの箇所数は、時間・費用・難しさの代理指標になりません。
- 静的な読解は未検証の予測を含み、実測で確かめていません。
- 計測の再開は、価値の主張を伴いません。
