# ISSUE-000072: 業務規則を一件変えるとき、人が手で直す箇所を数える

- 状態: Proposed — 評価待ち。計測の手順と、最初の計測ケースの机上予測までを記録した段階で、実際の変更による計測はまだ行っていない
- 提案日: 2026-09-30
- 性格: Product Valueの計測 / Reference Evidence
- 主担当要求: RQ-017（主）/ RQ-003 / RQ-013 / RQ-021 / RQ-022
- 関連Issue: [ISSUE-000073](000073-derive-validation-selection-from-canonical.md)（最初に見つかった手直し箇所を減らす）、[ISSUE-000039](000039-business-change-rehearsal.md)（差分・依存解析）、[ISSUE-000052](000052-sui-reference-adoption-and-plugin-surface-evaluation.md) / [ISSUE-000070](000070-sei-reference-adoption-change2-resumption.md)（他Productとの比較。同じ数え方を使う）
- 価値との対応: [Product Value](../../product/value/TEI_PRODUCT_VALUE.md)「価値の確かめ方 — 組織: 一つの業務規則を変えるとき、人が手で直す必要がある場所の数」、「組織 1. 変更の単位コストを下げる」
- Gate impact: None

## この文書の役割

Product Valueは、組織にとっての価値を「一つの業務規則を変えるとき、人が手で直す必要がある場所の数」で確かめると定めています。しかし、この数を**同じ方法で繰り返し数える手順**がありません。

これまでの数え方は、SEI Change 1の一回限りの比較（[ISSUE-000054](000054-sei-reference-adoption-resume-review.md)。結論は`not-observed`）と、ISSUE-000070 / 000052の比較表の設計だけです。受入境界のtest（`TestEnvironmentBindingAcceptWritesOnlyTheTwoBindingLines`等）は一つのoperationが書く行を固定しますが、規則の変更全体で人が何を直すかは数えていません。

このIssueは、TEI自身の`order_entry`で、規則一件の変更に伴う手直しを分類して数える手順を定め、最初の計測ケースを置きます。

## 背景

「同じ規則を複数の場所へ書き写さない」はTEIの中心的な価値です（原則P3、RQ-003）。ただし、価値があると言うには、書き写しが実際に何箇所あり、どれが機械的に再生成でき、どれが人の手に残るかを示す必要があります。

2026-09-30の読解で、次のことが分かりました。

- 第一Web Target（`tools/reference-first-web-target/command_scenario.go`）は、Validationの**内容**を書き写していません。Canonical Worldから規則の定義を読み込んで評価します。
- 一方、**どのValidationを評価するか**は、同じfileの2か所にCanonical IDの一覧として手で書かれています（`sales.order_line.quantity_positive`と`sales.order.lines.line_number_unique`）。
- 現行のCanonical modelとDriverは、Applicationの`uses validation`宣言から評価対象を導出していません（`APPLICATION_VALIDATION_SELECTION_NOT_CANONICAL`、[ISSUE-000027](000027-ai-proposal-acceptance-boundary.md)）。

このため、受注確定に新しいValidationを一件足すと、正本へ規則を足すだけでは画面からの実行で評価されません。人がTarget側の一覧を直す必要があります。これは、Product Valueの「画面側へ規則を書き写していません」という例が、規則の内容については正しく、評価対象の選択については成り立っていないことを意味します。

## 防ぎたい失敗

- **価値の主張が計測に裏付けられない。** 「一か所の変更が全体へ伝わる」と言いながら、実際に何箇所を人が直しているかを誰も数えていない。
- **数え方が毎回違う。** 数えるたびに対象（test・文書・生成物）の含め方が変わり、比べられない。
- **再生成できる変更と手直しを混ぜる。** golden fileの再生成のような機械的な変更を手直しとして数え、実際の負担を大きく見せる。逆に、手で直したtest期待値を「生成物」として数えず、負担を小さく見せる。
- **正本の外にある二つ目の置き場を見逃す。** Target・Evidence audit・業務手順の参照に、規則の一覧が別に存在していても気づかない。

## 守る原則

- 変更された箇所を、次の区分で数える。区分をまたいで合算した一つの数だけを報告しない。
- 数は、変更前後のrevisionと、実際の`git diff`から数える。予測値と実測値を混ぜない。
- 計測結果は、TEIが良く見える数だけを残さない。手直しが多かった場合もそのまま残す（RQ-022）。

## 数え方（区分）

| 区分 | 意味 | 例 |
| --- | --- | --- |
| C1 正本の変更 | authoritative sourceに対する、意味を変えるための変更 | `reference/workspace/semantic/sales/validation.tei` |
| C2 機械的な再生成 | 正本から決定論的に再生成でき、人が内容を書かない変更。受理toolが書く記録（`.tei/decisions/`）も含める | `reference/world.json`、golden、`reference/workspace/.tei/decisions/*.json` |
| C3 正本の外にある意味の手直し | 正本と同じ意味を、別の場所で人が直す必要がある変更。**減らす対象** | Target側のValidation一覧 |
| C4 期待値の手直し | test・Evidence auditの期待値を人が直す変更 | `*_test.go`、`tools/*-evidence-audit` |
| C5 文書 | 更新が必要な説明文書。`plan/`の監査記録など、過去の事実を残す文書にIDが出てくるだけの場合は数えない | `plan/`、`docs/`、利用者が読むREADME |
| C6 観測用のfixture・入力data | 新規則の効果を観測するために、人が書く入力dataやfixture。正本の意味ではないが、観測できるかどうかを左右する | `tools/reference-first-web-target/read_scenario.go`の固定行 |

C3が0に近いほど、SSOTが保たれています。C4はTEIの外でも必要になる負担なので、C3とは分けて報告します。

数え方の補足を置きます。

- **単位:** fileの数と、人が書いた独立の編集単位（hunk）の数を併記する。例えばC3は「1 file / 2か所」と書く。
- **直す / 足す:** 既存の期待値を直す変更と、新しいtestやfixtureを足す変更を分けて数える。
- **検出されるか:** C3は、直し忘れを検出するtestやauditがあるかを併記する。検出されないC3は、数が少なくても負担が小さいことを意味しない（例: 業務手順ReferenceのValidation一覧は、網羅を確認する仕組みがない）。
- **例示衝突:** 計測する規則が、リポジトリ自身のtestやREADMEの例と同じ場合、その衝突で落ちるtestはC4の本計上へ混ぜず、別行の「例示衝突」として報告する。規則変更一般の負担ではなく、このケースを選んだことによる負担だからである。

## 最初の計測ケース

**変更:** 受注明細の単価が正であることを、受注確定前に守る業務規則として追加する（`sales.order_line.unit_price_positive`、kind `positive`。対象は正本で`unitPrice`と書かれた属性で、Canonical IDは`sales.order_line.unit_price`）。`tools/reference-shared-acceptance/README.md`のcandidate例と同じ変更で、正本側はShared Acceptance Boundaryの`add-validation`で追加できます。

**机上の予測**（revision `efd50e46`時点の静的な読解。Goは実行しておらず、実測ではない。個々のtest関数が落ちるかどうかは、読んだ範囲の根拠を併記し、確認していないものは「未確認」と書く）:

前提として、変更は`semantic/sales/validation.tei`への追記だけとする（`add-validation`は既存fileの末尾へ追記する。`tools/reference-shared-acceptance/boundary_test.go:82-88`）。`order_entry`は`uses validation`を宣言しておらず（`reference/world.json:672-676`は`semanticUses`のみ）、Applicationのsourceは変えない。

**A. C3（Targetの評価対象一覧）を直さない場合**（最初の観測用。ISSUE-000073の前の状態）

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

**B. C3を直し、新規則をTargetが評価するようにした場合**（ISSUE-000073の前に手で直した場合の再計測。新しいIDを一覧の末尾へ足す想定）

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

## 設計詳細

1. **計測はscratch copyで行う。** Gate B / Gate CのBaseline Evidenceを変えないため、実際の`reference/workspace`へは受理しません。
2. **手順:** `revision`でbaseを取る → candidateを`preview` → `accept`（scratch） → golden再生成 → `go test ./...`で落ちたtestを直す → Web Targetから確定操作を行い、新規則が評価されるかを確かめる → `git diff --stat`とfileごとの区分付けを記録する。
3. **「評価されるか」の確認を必須にする。** C3を直さない状態で、先頭の明細の単価が0以下の受注を確定させ、新規則が評価されない（`Validations`に新規則が現れず、`validation-not-satisfied`にならずに`completed`へ進む）ことを先に観測する。これが観測できれば、C3が実在する手直しであることのEvidenceになる。観測の方法は二つあり、どちらを使ったかを記録する。
   - in-processで、読込結果の`OrderLineRows[0]`の単価を差し替えて`RunCommand`を呼ぶ（`command_scenario_test.go:176-190`と同じ形）。fixtureを変えないので、C6は0になる。
   - fixtureの単価を0にして、HTTPの`/scenario/submit`から確定する。`/scenario/submit`は明細を受け取らず、現行のfixtureは単価が常に正なので、この方法ではC6が1以上になる。
   評価されるのは先頭の明細だけなので（`command_scenario.go:391-395`）、2行目以降の単価が0以下の場合は、C3を直した後も止まらない。観測は先頭の明細で行い、その範囲を超える主張はしない。
4. **記録:** `reference/value-measurement/rule-change-<日付>.json`に、変更内容、before / after revision、区分ごとのfile・行、C3を直さなかった場合の観測結果、実行した環境を残す（file名と形式は初回の実測で確定する）。
5. **再計測:** ISSUE-000073でC3を減らした後、同じ変更をもう一度計測し、区分ごとの差を記録する。

## 完了条件

- [ ] 区分C1〜C6をowner reviewで確定する
- [ ] 最初の計測ケースをscratch copyで実行し、区分ごとの実測値と、C3を直さなかった場合の観測結果を記録する（Goのあるtrusted local環境が必要）
- [ ] 実測値と机上予測の差を記録する
- [ ] ISSUE-000073の完了後、同じケースを再計測し、C3の変化を記録する
- [ ] 計測手順をISSUE-000052 / 000070の比較表の数え方と対応付ける（同じ区分で数えられるか）

## 完了後に判断できること

- Product Valueの「一か所の変更が全体へ伝わる」を、どの範囲で主張してよいか
- 次に減らすべきC3がどこにあるか
- 他Product（SUI / SEI）との比較に、同じ数え方を使えるか

## 非主張範囲

- 一つの計測ケースで、TEI全体の変更コストを主張しません。
- 手直しの箇所数は負担の代理指標であり、時間や費用を測るものではありません。
- scratch copyでの計測は、実際の`reference/workspace`やGate Evidenceを変更しません。
