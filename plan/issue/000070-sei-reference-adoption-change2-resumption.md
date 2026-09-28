# ISSUE-000070: SEI Reference AdoptionのChange 2を再開する — Gate C後のApplication PlaneでResumeの表示変更を比べる

- 状態: Proposed — 評価待ち。再開条件が成立したことを記録した段階で、Change 2はまだ実行していない
- 提案日: 2026-09-27
- 性格: Reference adoption / cross-application pressure / ISSUE-000054の後続
- 主担当要求: RQ-003 / RQ-010 / RQ-011 / RQ-013 / RQ-021 / RQ-022 / RQ-023
- 前段Issue: [ISSUE-000054](000054-sei-reference-adoption-resume-review.md)（Done。Stage 0の結論はlimited / Reference-local）
- 設計: [SEI Reference Adoption Stage 0設計](../design/SEI_REFERENCE_ADOPTION_STAGE0_DESIGN.md) §10.2
- 前回の停止判断: [`change2-presentation-gate-boundary-20260918.json`](../../reference/sei-adoption/change2-presentation-gate-boundary-20260918.json)
- SEI counterpart: `hat47x/sei-cognition` ISSUE-000079（Change 2の再開条件を置いたSEI側のIssue）
- Gate impact: None

## この文書の役割

ISSUE-000054で保留したChange 2（SEIのResumeについて、意味を変えずに表示の順序と段階的な開示だけを変える比較）の再開条件が、TEI側で成立しました。本Issueはその事実を記録し、再開したときに何を確かめ、何を条件に公平な比較とみなし、何を主張しないかを定めます。

Change 2そのものはまだ実行していません。本Issueが採用されても、TEIがSEIのApplication Planeとして優れていることや、SEIの移行を広げてよいことは意味しません。

末尾の付記には、同じSEI連携の検討から出た二つの設計メモ（受理記録をSEIの判断の記憶へ写す候補、MCPの扱いをそろえる候補）を置いています。どちらも評価待ちで、本Issueの完了条件には含めません。

## 背景

### Stage 0で分かったこと

ISSUE-000054では、SEIのDecision Continuity fixtureを固定snapshotとして借り、Resume（判断を再開するときに見る要約）の意味要素をTEI側でread-onlyに導けるかを確かめました。意味要素とは、current question、Recommendation（AIや人の提案）、Decision（採った決定）、historical basis（決定時点の根拠）、unresolved（未解決のまま受け入れたこと）、changes since decision（決定後に分かったこと）の六つです。

結果は次のとおりでした。

- Reference-localなmapping（TEIのCoreやPatternを使わない隔離toolでの対応づけ）では、六つの意味の区別を保てた。
- Change 1（決定後にObservationを一件足す）では、意味の互換性は確認できたが、native SEIと比べて変更の手間が減る様子は見えなかった。むしろfixtureのsnapshot同期という手間がTEI側にだけ増えた。
- Change 1は両方式にとって易しすぎ、Application Planeとしての差が出にくいことも分かった。

### Change 2を保留した理由

次の候補だったChange 2は、意味を一切変えずに「何を先に見せ、何を畳んでおくか」だけを変える比較です。2026-09-18の時点ではGate BがNot Ready、Gate CがBlockedで、第一Web Targetも未実装でした。この状態でSEI専用の表示順schemaやPresentation helperを作ると、TEIの既存能力ではなく、この比較のためだけに作った代用品を測ることになります。そのためChange 2は保留し、再開条件を次のどちらかとしました。

1. Gate BがReadyになり、Handover Baseline `B` / revision `R`が固定され、第一Web Targetの作業へ正式に進んでいること
2. SEIとは独立した理由で、Targetに依存しないPresentationの仕組みがTEIに成立していること

SEI側のISSUE-000079も、同じ条件（Gate B Ready + Handover Baseline固定 + First Web Target着手）でChange 2を保留しています。

### 現在の状態

条件1は成立しています。根拠は[`README.md`](../../README.md)と[`plan/README.md`](../README.md)の現在地、および[Gate C Readiness Audit](../GATE_C_READINESS_AUDIT.md)です。

| 条件 | 現在 | 根拠 |
| --- | --- | --- |
| Gate B Ready | 成立（candidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`） | [Gate B Readiness Audit](../GATE_B_READINESS_AUDIT.md) |
| Handover Baseline固定 | 成立（`B / R`） | [Baseline B](../GATE_B_TO_C_HANDOVER_BASELINE_B.md) |
| 第一Web Target着手 | 着手済み。実装とEvidence取得まで完了（build `T = 0ba87a4e0a96f70a198b03ff7652fb00e3826abe`） | [ISSUE-000007](000007-first-web-target-projection.md) |
| Gate C | 2026-09-19にReady。Visual Fidelityは単一のAI評価者による評価で、人による確認ではない | [Gate C Readiness Audit](../GATE_C_READINESS_AUDIT.md) / [ISSUE-000058](000058-gate-c-visual-independent-reevaluation.md) |

条件2は成立していません。Gate Cは汎用のPresentation taxonomyを要求しておらず、`general-target-neutral-presentation-taxonomy-not-yet-defined`は未解決のまま[ISSUE-000026](000026-target-capability-execution-profile-boundary.md)へ引き継がれています。Change 2はこれが無いことを前提に設計します。

ISSUE-000054のResolutionは、再開時には000054を開き直さず、新しいEvidenceに結び付いた後続Issueで扱うと定めています。本Issueがその後続Issueです。

SEI側でも、2026-09-27に行った「SEIを製品群の統合の頂点に置く」ための評価で、この再開がすぐに着手できる項目として挙がりました。評価記録は`hat47x/sei-cognition:product/value/SEI_FAMILY_APEX_VALUE_REVIEW.md`にあり、ISSUE-000079へ再開条件の成立を記す日付つきの節も提案されています。どちらもSEIの未統合branch `docs/family-apex-backflow-20260927`にだけあり、SEI側ではまだ採用されていません。

## 防ぎたい失敗

1. **再開条件の成立を、Change 2の成功と読み替える。** 条件が満たされたのは比較を始められるという意味であり、TEIが有利という意味ではありません。
2. **比較のために代用品を作り、それをTEIの能力として数える。** SEI専用の`display_order`、Resume専用のPresentation model、新しい`resume` / `review` Pattern Kind等を作れば、Change 2は通りやすくなりますが、測っているのは新設した層です。000054で止めた失敗の再発です。
3. **表示の変更に紛れて意味が動く。** たとえば根拠の欄を畳む実装で、決定後に分かったことまで同じ欄へ入れてしまうと、画面は整って見えても、決定時点の根拠に後知恵が混ざります。
4. **違う変更同士を比べる。** native SEIとTEI側で異なる表示変更をしたり、片方だけ別のrevisionを使ったりすれば、差の原因がTEIなのか変更内容なのか分からなくなります。
5. **Gate CのVisual判定を、人にとって読みやすいことの証拠として流用する。** Gate CのVisual Fidelityは単一のAI評価者の判定です。Change 2で画面を比べても、人の使いやすさは主張できません。
6. **依存を作る。** TEIをSEIの必須基盤にすることも、SEIをTEIの必須部品にすることも、この比較の目的ではありません。

## 守る原則

- **SEIの意味の正本はSEIに残す（RQ-003）。** TEI側に置くのは、固定snapshotから導いたProjectionだけです。TEI側で第二のResume sourceを作りません。
- **同じ変更を、同じ入力で、同じ物差しで比べる。** 下の「同値条件」を満たさない比較は、結果を記録しても優劣の根拠にしません。
- **測る土台が無ければ、作ってまで測らない。** 既存の仕組みで測れないと分かった場合は、「測れない」ことを正当な結果として記録します。SEIのために新しい層を作って測れるようにはしません。
- **Evidenceの高さを混ぜない。** Reference-localなcheckerの成功、Targetが描画したHTML、単一AI評価者のVisual判定、人の使いやすさは、それぞれ別の主張です（RQ-022）。
- **依存を作らない。** TEIはSEIが無くても成り立ち、SEIもTEIが無くても成り立ちます。

## Change 2が圧力をかけるもの

Change 1は「recordを一件足す」変更で、どちらの方式にとっても得意な操作でした。Change 2は、同じ意味の**見せ方**だけを変えます。これはTEIが主張している次の価値へ、直接圧力をかけます。

- 同じCanonical meaningを複数のProjectionで扱えること（RQ-011）
- 人が高密度に理解・判断できる表示を作れること（RQ-010）
- Targetごとの差をCoreへ埋め込まず、Target側に閉じ込められること（RQ-013）

Gate C後のApplication Planeには、比較に使える仕組みが少なくとも一つあります。第一Web Target（[`tools/reference-first-web-target`](../../tools/reference-first-web-target)）は、Driver Resolverが解決した5種類のApplication PatternをHTML / JSONへProjectionします。Custom UIもJSON APIと同じ`commandScenarioRunner`を使い、Target固有の差はform・layout・HTMLの描画だけに閉じています。汎用のPresentation IRやWeb frameworkはCanonicalへ追加していません。さらに、`R`から`T`までの間にCanonical Workspaceと生成Worldが変わっていないことを、Gate Cのarchitecture conformanceで確かめています（詳細は[`docs/status.md`](../../docs/status.md)）。

そこでChange 2の問いは次のようになります。

> Resumeの表示順と段階的な開示だけを変えるとき、その差をTarget側の描画に閉じ込め、SEIのsourceにもTEIのCanonical meaningにも触れずに済むか。そのとき、native SEIで同じ変更をするより、変更箇所・手作業の同期・意味の重複が減るか。

これを確かめるには、次の三つが必要です。

1. **載せられること。** Resumeの六つの意味要素を、新しいPattern KindやSEI専用のfieldなしに、既存のApplication PatternとTargetの仕組みへ載せられる。
2. **閉じ込められること。** 表示だけの変更が、Canonical Worldや生成物を変えず、Target側の差分だけで済む。Gate Cのarchitecture conformanceと同じ見方で確かめられる。
3. **確かめられること。** 意味の不変条件を、executableなcheckで確認できる。

このうち1には、まだ分からない点があります。現在のApplication Patternは、Database ContractやContract Queryを通してdataを読みます。一方、SEIのfixtureはSEIが所有するJSON recordです。000054のStage 0では、Resumeを`record`や`browse`へ押し込むと、既存Patternの意味をSEIの都合で広げるおそれがあると判断しました。既存の仕組みだけで1が成り立つのか、SEI専用の層が要るのかは、最初に確かめる必要があります。

## 具体例

000054の固定fixtureは、「通知Provider Aのtimeoutが増えたため、Provider Bへ移行すべきか」という判断を扱います。

```text
Decision（dec-pilot）
  全面切替はせず、送信量10%だけProvider Bへpilotし、Aをfallbackとして残す

Recommendation（rec-full-switch）
  AIはProvider Bへ全面切替する案を提案した

historical basis
  obs-a-timeout    直近7日でAのtimeoutが3回
  obs-b-synthetic  Bのsynthetic testは20/20成功（本番相当負荷は未確認）

unresolved
  unk-b-rate-limit  Bの本番負荷時のrate-limit挙動
  unk-rollback

changes since decision
  obs-b-429          pilot開始2日後、高負荷帯でBから429
  obs-a-maintenance  決定の3日後にAが翌月のmaintenance予定を公表（決定時点では未公表）
```

Change 2の変更内容は実行時に確定しますが、たとえば次のような案が考えられます。

```text
現在の順序
  current question → Recommendation → Decision → historical basis → unresolved → changes since decision

Change 2案 — 再開した人がまず確かめるべきものを先に出す
  changes since decision → unresolved → Decision → Recommendation
  historical basisは件数と見出しだけを出し、本文は開いたときに表示する
```

成功とみなす条件は、表示が変わっても意味が変わらないことです。

- `dec-pilot`と`rec-full-switch`のidentityは変わらない。Recommendationは表示上もDecisionと区別されたままである
- historical basisは`obs-a-timeout`と`obs-b-synthetic`のまま
- unresolvedは二件とも残り、表示を変えただけで解消されない
- `obs-b-429`と`obs-a-maintenance`はchanges since decisionにだけ現れ、根拠の欄を開いても閉じても、historical basisへ混ざらない

反対に、historical basisを畳む実装で、changes since decisionまで同じ「根拠」の折りたたみ欄へ入れてしまえば、画面が整っていても失敗です。このような混入を検出したら、黙って表示せず、fail closedで止めます。

## 設計詳細

### 1. 最初の作業 — 測る土台があるかを確かめる（read-only）

Change 2を実行する前に、current `main`で次を確かめます。コードもReferenceも変えません。

- Resumeの六つの意味要素を、どの既存Application Patternで運べるか。運べない要素はどれか
- SEIのrecordを、意味の正本をSEIに残したまま、どうやってPatternへ届けるか（datasourceの問題）
- 第一Web TargetのTarget側の層で、表示順と段階的な開示の変更を表せるか
- 新しく作らなければならないものが出た場合、それがSEIとは独立した理由を持つTEIの一般的な能力か、SEIのためだけの代用品か

結果は次の二つのどちらかです。どちらも正当な結果として扱います。

| 結果 | 次の扱い |
| --- | --- |
| A. 既存の仕組みで測れる | 下の同値条件を満たしてChange 2を実行する |
| B. SEI専用の層が要る | 作らない。足りない能力を、SEI固有の語を使わない形でTEI側の不足候補として記録する（Presentationの分類に関わるものはISSUE-000026へ渡す）。Change 2は理由を更新したうえで保留を続ける |

### 2. 同値条件 — 公平な比較とみなす条件

native SEIとTEI側の結果を比べるのは、次をすべて満たす場合に限ります。

1. **同じ入力。** 両方が同じ固定SEI source（revision `e943683fbd71ea689bf40fb4c7009531fe98ece5`、blob `6354fce0a01093cb5545ed6bb9730aceb3b20023`）を使う。SEI側でfixtureが変わった場合は、両方を新しいrevisionへ固定し直す。
2. **同じ変更。** 表示順と段階的な開示だけを変える、同じ変更仕様を両方へ与える。意味を変える要素を含めない。
3. **同じ不変条件。** 前回の停止判断で定めた不変条件を両方で確かめる。SEIの意味のauthorityは変わらない。DecisionとRecommendationのidentityは変わらない。historical basisの参照は変わらない。unresolvedの参照は変わらない。決定後の変化は時間的に分離されたままである。
4. **新しい意味を足さない。** どちらの側も、新しいsemantic kind、Pattern Kind、Core field、SEI専用のPresentation IRや`display_order` schemaを追加しない。追加が必要になった時点で、その比較は「測る土台が無い」（結果B）として扱う。
5. **同じ物差し。** 下の「測る項目」を両方について記録する。
6. **同じ高さの実行Evidence。** 両方の結果を、exact revision / blobを記録したexecutableなcheckで確かめる。片方だけ再実行していない場合は、そのことを明記する。
7. **同じ評価方法。** 画面を比べる場合は、両方を同じ方法で評価し、評価者の種類（人か、AIか、何人か）を記録する。単一のAI評価者の判定を、人の使いやすさとして扱わない。

### 3. 測る項目

Change 1の項目に、表示変更に固有の項目を加えます。

| 項目 | 見ること |
| --- | --- |
| source owner数 | 意味の正本を持つ場所が増えていないか |
| 手書きのmapping / Projection変更 | 表示を変えるために、何箇所を手で直したか |
| test変更 | 不変条件のcheckをどれだけ直したか |
| 手作業の同期 | repositoryをまたぐsnapshot同期等が要ったか |
| 意味の重複 | 同じ意味を二箇所で書いていないか |
| 派生物の再生成 | 生成物の再生成が要ったか。要った場合、Canonical Worldは変わっていないか |
| authorityの曖昧さ | どこが正本か迷う箇所が生まれていないか |
| Target側の差分 | 変更がTarget側の描画だけに閉じたか |
| Canonical / Worldの不変 | `R → T`と同じ見方で、Canonical Workspaceと生成Worldが変わっていないか |

### 4. Evidenceの残し方

Change 1と同じく、実行結果は`reference/sei-adoption/`へmachine-readableなreceiptとして残します。exact blob、Go version、実行したcommandと結果を記録し、reportには`authority_scope = pinned-revision-only`と`current_source_conformance = not-claimed`を引き続き持たせます。画面を比べた場合は、評価者の種類も同じreceiptへ記録します。

## 完了条件

- [ ] current `main`で、Resumeの意味要素を既存のApplication Patternと第一Web Targetの仕組みで扱えるかを、read-onlyで判定して記録する（結果AかBか）
- [ ] 結果Aの場合、同値条件を満たしたうえで、同じ表示変更をnative SEIとTEI側へ与え、測る項目を比較する
- [ ] 結果Bの場合、SEI専用の層を作らず、足りない能力をSEI固有でない語でTEI側の不足候補として記録する
- [ ] 結果Aでは実行のreceiptを、結果Bでは判定の記録を、`reference/sei-adoption/`へmachine-readableに残す
- [ ] 結果をISSUE-000054のadoption posture（limited / Reference-local）へ返し、広げる・狭める・据え置くのいずれかを判断する
- [ ] SEI側ISSUE-000079へ、結果とEvidenceの高さを返す（SEI側の記述はSEIで行う）

## 完了後に判断できること

- SEIのPresentation / Review面について、TEI-firstの方針を広げるか、狭めるか、limited / Reference-localのまま据え置くか
- Targetに依存しないPresentationの仕組みについて、SEIとは独立した不足候補をISSUE-000026へ渡すべきか

## 付記 — 同じSEI連携の検討から出た二つの設計メモ（評価待ち・本Issueの完了条件外）

SEI側では2026-09-27に、hat47xの製品群に対するSEIの役割を「統合の頂点」として定義し直す提案が出ました（`hat47x/sei-cognition:plan/adr/ADR-0032-family-integration-apex-and-control-meaning-boundary.md`、提案中）。SEIは製品ごとの記録を統合せず、製品間の対応表（`hat47x/sei-cognition:product/definition/SEI_FAMILY_SEMANTIC_MAP.md`）で翻訳する、という立場です。どちらもSEIの未統合branch `docs/family-apex-backflow-20260927`にだけあります。TEIから見た境界は[TEI Product Value §8](../../product/value/TEI_PRODUCT_VALUE.md)に記しています。

この提案に対して、TEI側で残しておきたい設計メモが二つあります。どちらも小さく、まだ作業として切り出す段階ではないため、AGENTS.mdの「小さい機能では新しい設計文書を作らず、Issue本文へ書く」に従ってここへ置きます。作業に進める場合は、それぞれ独立したIssueへ切り出します。

### 付記A — 受理記録（`.tei/decisions`）をSEIの判断の記憶へ任意に写す

#### 背景

Shared Acceptance Boundary（file / CLI / Studio / AIのどの入口から来た変更候補も、同じ検査を通してから正本へ反映する受理境界）は、受理のたびに`.tei/decisions/<内容のSHA-256>.json`へ記録を残します（[`tools/reference-shared-acceptance`](../../tools/reference-shared-acceptance/README.md)、RQ-027）。この記録は、後から入る人やAIが「このsourceはなぜここにあるのか」を会話履歴なしで読むためのもので、**Evidenceであってauthorityではありません**。`.tei/`はmanaged sourceではないので、記録を全部消しても、model revisionも受理済みの意味も変わりません。

SEI側の対応表案は、この受理記録を「Decisionの記録 + Authority evidence ref（権限の証拠への参照）」と読みます。SEIの`hat47x/sei-cognition:plan/issue/ISSUE-000082-derive-family-afferent-profile-from-sibling-records.md`（未統合branchのみ）は、実在する兄弟製品の記録から、製品がSEIへ送る記録の最小形を導こうとしています。TEIの受理記録はその入力の候補です。

#### 防ぎたい失敗

- SEIに写した受理記録を、TEIの受理権限そのものと取り違える
- 写したSEI側の記録が、TEIの記録より「正しい」版として扱われ、第二の正本になる
- 受理の記録と一緒に、sourceの本文まで不必要にSEIへ広がる
- TEI側の記録に無い情報（たとえば時刻）を、SEI側で推測して埋める

#### 守る原則

- 写すかどうかは任意です。TEIはSEIへ送らなくても成り立ちます。
- 写した結果はSEI側の派生物です。SEI側で消しても、TEIの記録にも意味にも影響しません。
- SEIが持つのはTEIの記録への参照と、その解釈だけです。TEIの受理権限はTEIに残ります。
- 本文ではなく参照を渡すことを基本にします。sourceの本文（`insertedText`）は既定では写さず、fingerprintで指します。

#### 対応の候補

production schemaではありません。意味の損失を確かめるための対応の候補です。

| TEIの受理記録 | SEIでの読み（候補） | 注意 |
| --- | --- | --- |
| `decision: accepted`と`candidateId` | Decision（何を受理したか） | `reject`は記録を残さない。SEI側で「却下の記録が無い」を「却下が無かった」と読まない |
| `origin.kind` / `origin.actor` | 候補がどの入口から来たか（provenance）。`origin.kind = ai`の候補は、SEIのRecommendationにあたる | TEIでもprovenanceにすぎず、判定にもauthorityにも使っていない |
| 受理を決めた境界と主体（Shared Acceptance Boundaryと、利用者の明示的なAcceptance decision） | Authority evidence ref | SEIへ写るのは参照だけで、権限そのものではない |
| `intent` / `evidence` | 判断の理由と根拠 | |
| `baseModelRevision` / `modelRevisionBefore` / `modelRevisionAfter`、各fileの`afterFingerprint`、記録file名（内容のSHA-256） | Exact Binding（判断がどの版に対するものかを固定する束縛） | |
| `unresolved` | unresolved refs | 受理しても未解決は消えない |
| `decisions`コマンドが返す`current` / `superseded`と`intact` / `diverged` | その時点のObservation | 記録の一部ではなく照合結果。照合した時刻とともに写す |

実例として、`reference/workspace/.tei/decisions/`には第二Application `repair_intake`の受理記録が2件あります（Semantic / Application / Patternの3 sourceの受理と、Environment bindingの受理）。SEI側で対応を試す場合は、この2件をそのまま入力にできます。

最初に見えている損失候補が一つあります。受理記録には時刻の項目がありません。SEI側で時刻が必要なら、記録をcommitしたGitの時刻を別の出所として付けるか、不明として残すことになります。TEI側へ時刻を足すかどうかは、対応づけでこの損失が実際に問題になってから判断します。

#### 未解決 / 非主張範囲

- TEI側に書き出しtoolやschemaは追加しません。追加が必要になった場合は、`tools/reference-shared-acceptance`の`decisions`出力の隣にread-onlyのtoolを置く案から検討し、独立したIssueとRepository health checkを通します。
- 対応づけの成否はSEI側（ISSUE-000082）で観測します。意味の損失が観測されても、されなくても、正当な結果です。
- 受理記録がBoundaryを通った変更に限られることは変わりません（Coreには同等の仕組みがありません。[Product Requirement Gap Audit](../PRODUCT_REQUIREMENT_GAP_AUDIT.md)のRQ-027行）。

### 付記B — RQ-013のMCPの区別と、SEIのMCP面（読み取りと提案だけ）をそろえる

#### 背景

[RQ-013](../../requirements/TEI_PRODUCT_REQUIREMENTS.md)は、MCP（Model Context Protocol。AIエージェントが外部のtoolやdataへ接続するための共通protocol）について、二つの使い方を区別するよう求めています。

1. Application自身が、業務上のPresentation / InteractionをMCP Targetとして公開する場合
2. TargetやHostが、inspectionや開発の制御のためにMCP等のcontrol interfaceを公開する場合

[Actor Context Projections §9](../ACTOR_CONTEXT_PROJECTIONS_2026-09.md)は、AI Business Actor向けのmachine interfaceについて、三つ目の候補（Operational Consumer interface）も挙げています。

SEI側の境界設計案（外部のエージェントやprotocolとの境界）は、SEIのMCP面を「Canonical Semantic Access Surface」（SEIの意味へ外から触れる正規の入口）の実現の一つとし、**読み取りと提案だけ**に限ります。決定は、明示された権限の経路で行います。これもSEIの未統合branchにだけある提案です。

#### そろえ方（候補）

- **TEIがSEIのMCPを使う場合。** SEIのMCPは、RQ-013の1にも2にもあたりません。TEIから見れば外部の認知Providerのinterfaceです。返ってくるものは認知の観測や提案であり、Canonical meaningでも実行権限でもありません（[Product Value §7](../../product/value/TEI_PRODUCT_VALUE.md)、[DCR-01 / DCR-15](../DERIVED_COGNITIVE_REQUIREMENT_DRAFT_2026-09.md)）。SEIへ渡す入力には、RQ-019どおり権限やsecretを混ぜません。
- **SEI（またはSEIの文脈を持つ外部エージェント）がTEIへ働きかける場合。** 読み取りはTEIのProjectionを通します。「提案」は、Application定義の変更ならShared Acceptance Boundaryへの変更候補として（ADR-0022）、業務の実行なら、TEIがOperational Planeで評価する依頼（Action Intent）として（ADR-0028）受けます。どちらの場合も、2のcontrol interfaceを業務Actionの入口にしません。
- **設計の収束として記録する。** SEIの「読み取りと提案だけ、決定は明示された権限の経路で」という形は、TEIのADR-0022（ProposalとCanonical受入の分離）とADR-0028（読めることと実行できることの分離）と同じ形です。二つの製品が独立に同じ形へ行き着いたことは、三つ目の候補（Operational Consumer interface）を将来formalizeするときの参考になります。ただし参考にとどめ、形を先に決めません。

#### 未解決 / 非主張範囲

- RQ-013の本文は変えません。本付記はRQ-013の意味を変えず、SEIのMCPを分類するときの読み方を記すだけです。
- TEIがMCP Targetを持つべきかどうかは、引き続きRQ-013が求めるReference Evidenceで判断します。
- SEIのMCP面の仕様はSEIが正本です。SEI側で読み取りと提案に限る方針が採用されるまで、本付記も評価待ちです。

## 非主張範囲

- Change 2は実行していません。本Issueは再開条件の成立と、再開時の進め方を記録しただけです。
- 再開条件が成立したことは、Change 2の成功も、TEIの優位も意味しません。
- Gate C Readyは第一Web TargetのReferenceに限った判定です。Visual Fidelityは単一のAI評価者による評価で、人による確認ではなく、独立した再評価（ISSUE-000058）も保留中です。Change 2で画面を比べても、人の使いやすさは主張しません。
- Targetに依存しないPresentation IR / taxonomyは未成立です。Change 2はそれを要求せず、そのために作りもしません。
- SEIの移行を広げること、TEIのProduct Valueが成立したこと、SEIやTEIが互いの必須依存になることは主張しません。
- SEI側の提案（ADR-0032、ISSUE-000079への追記、ISSUE-000082、対応表）はSEIの未統合branchにだけあり、SEI側で採否が決まっていません。
- 新しいApplication Pattern Kind、TEI Core primitive、Product Requirementは追加しません。

## Natural Japanese review

このIssueが言いたいことは短くまとめられます。前回、比べる土台が無いという理由で止めた比較を、土台ができたので再開できるようになった、ということです。

ただし、土台ができたことと、比べて勝てることは別の話です。再開の最初の仕事は、SEIのResumeをTEIの既存の仕組みに載せられるかを確かめることで、載せられなければ無理に載せず、「まだ測れない」と記録します。載せられた場合も、同じ入力・同じ変更・同じ物差しで比べたときだけ、結果を判断に使います。
