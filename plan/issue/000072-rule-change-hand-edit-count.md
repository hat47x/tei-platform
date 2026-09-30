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
| C2 機械的な再生成 | 正本から決定論的に再生成でき、人が内容を書かない変更 | `reference/world.json`、golden |
| C3 正本の外にある意味の手直し | 正本と同じ意味を、別の場所で人が直す必要がある変更。**減らす対象** | Target側のValidation一覧 |
| C4 期待値の手直し | test・Evidence auditの期待値を人が直す変更 | `*_test.go`、`tools/*-evidence-audit` |
| C5 文書 | 説明文書の更新 | `plan/`、`docs/` |

C3が0に近いほど、SSOTが保たれています。C4はTEIの外でも必要になる負担なので、C3とは分けて報告します。

## 最初の計測ケース

**変更:** 受注明細の単価が正であることを、受注確定前に守る業務規則として追加する（`sales.order_line.unit_price_positive`、kind `positive`。対象は正本で`unitPrice`と書かれた属性で、Canonical IDは`sales.order_line.unit_price`）。`tools/reference-shared-acceptance/README.md`のcandidate例と同じ変更で、正本側はShared Acceptance Boundaryの`add-validation`で追加できます。

**机上の予測**（revision `ee5b5b14`時点の読解。実測ではない）:

| 区分 | 予測される変更 | 箇所 |
| --- | --- | --- |
| C1 | `semantic/sales/validation.tei`へ一件追加（`accept`が生成） | 1 |
| C2 | `reference/world.json`の再生成 | 1 |
| C3 | `tools/reference-first-web-target/command_scenario.go`の評価対象一覧（2か所） | 2 |
| C4 | Web TargetのCommand scenario test、Evidence auditの期待値（新規則が評価結果に現れる場合） | 未確定 |
| C5 | 業務手順Reference（`reference/business-system-design/order-entry-business-procedure.json`）が規則を列挙している場合の追従要否 | 未確定 |

参考として、同じrevisionで既存規則`sales.order_line.quantity_positive`のIDを含むfileは、正本1、生成物1、test以外のcode 2（上記Target scenarioとEvidence audit）、test 22、文書13、その他1（業務手順Reference）でした。これはIDの出現数であり、変更時に直す箇所の数ではありません。

## 設計詳細

1. **計測はscratch copyで行う。** Gate B / Gate CのBaseline Evidenceを変えないため、実際の`reference/workspace`へは受理しません。
2. **手順:** `revision`でbaseを取る → candidateを`preview` → `accept`（scratch） → golden再生成 → `go test ./...`で落ちたtestを直す → Web Targetから確定操作を行い、新規則が評価されるかを確かめる → `git diff --stat`とfileごとの区分付けを記録する。
3. **「評価されるか」の確認を必須にする。** C3を直さない状態で、単価が0以下の明細を含む受注を画面から確定し、新規則が評価されない（Validation not-satisfiedにならない）ことを先に観測する。これが観測できれば、C3が実在する手直しであることのEvidenceになる。
4. **記録:** `reference/value-measurement/rule-change-<日付>.json`に、変更内容、before / after revision、区分ごとのfile・行、C3を直さなかった場合の観測結果、実行した環境を残す（file名と形式は初回の実測で確定する）。
5. **再計測:** ISSUE-000073でC3を減らした後、同じ変更をもう一度計測し、区分ごとの差を記録する。

## 完了条件

- [ ] 区分C1〜C5をowner reviewで確定する
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
