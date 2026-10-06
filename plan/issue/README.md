# TEI Planning Issues

このディレクトリを、計画上の作業定義と進行確認の入口とします。背景、制約、完了条件、担当境界は各Issueに置き、採用する設計判断は`../adr/`へ記録します。既存GitHub Issueへの参照は履歴・Evidenceを辿るために残しますが、本整理ではGitHubへ新規起票・更新しません。

TEIのIssueは、単なるTODO一覧ではありません。**どのProduct上の不確実性を減らし、その結果として次に何を判断できるようにするか**を残すための文書です。

## Issueを読むときの四つの問い

技術詳細やチェックリストへ入る前に、まず次を確認します。

1. **何に困っているのか** — どの失敗や不確実性を減らす作業か
2. **このIssueは何を所有するのか** — 隣接Issueとどこで責務を分けるか
3. **何が分かれば完了なのか** — 実装量ではなく、どのEvidenceで判断可能になるか
4. **完了後に何を決められるのか** — ADR、次のReference、production採用等のどこへ接続するか

たとえば「AI探索」という名前だけでは、検索器を作るIssueなのか、意味上の適用可否を判定するIssueなのか、Proposalを受理するIssueなのかが分かりません。TEIでは、それらを別の責務として分けます。

```text
候補を見つける
  !=
意味上使えると確認する
  !=
変更案として受理する
  !=
production authorityへ採用する
```

この区別を保つことで、一つの大きなAI / Business / Runtime Issueへ責務が集まりすぎることを避けます。

## 現在の重要な三つの読み筋

### 1. Design Synthesis / knowledge-first探索

Design Synthesis全体の説明は[`../../docs/design-synthesis-reading-guide.md`](../../docs/design-synthesis-reading-guide.md)から読みます。個別Issueは次のように役割が分かれています。

```text
既存資産を思い出す候補を作る
        ↓
000051 derived cognitive retrieval evaluation
        ↓ candidate identities
000047 knowledge-first semantic exploration
        ↓ applicable / incompatible / unknown
000027 AI Proposal / Acceptance Boundary
        ↓ accepted change only
Shared Acceptance Boundary / authoritative source
```

| Issue | 減らす不確実性 | このIssueが決めないこと | 完了後にできる判断 |
| --- | --- | --- | --- |
| [000051 非LLM認知方式の比較](000051-derived-cognitive-retrieval-evaluation.md) | exact / lexicalだけでは拾いにくい候補を、軽量なlocal方式で有用に増やせるか | Semantic applicability、採用、write authority | どのcandidate mechanismを000047の前段へ残す価値があるか |
| [000047 knowledge-first探索](000047-knowledge-first-semantic-exploration.md) | 見つかった既存資産が現在のintentへ本当に使えるか | AI Proposalの採用、Canonical書込み | `applicable / incompatible / unknown`を理由付きで返し、明示参照candidateへ進めるか |
| [000027 AI Proposal / Acceptance](000027-ai-proposal-acceptance-boundary.md) | 非決定的なAI Proposalを、AI専用authorityを作らず正本へ反映できるか | read-only探索そのもの、Studio UI | shared Acceptance Boundaryを通したminimum semantic deltaを安全に受理できるか |
| [000046 Studio Shared Acceptance](000046-studio-shared-acceptance-authoring-loop.md) | Studio経路もfile / CLI / AIと同じ受理境界へ接続できるか | AI固有の推論方式 | Authoring surfaceが増えてもauthorityを増やさずに済むか |
| [000040 Semantic component composition](000040-semantic-component-composition.md) | 見つけた既存資産を第二Application等へ安全に組み立てられるか | candidate retrieval algorithmの優劣 | knowledge-firstな再利用が実際の設計作業を減らすか |

この系列で最も重要なのは、`000051`のscoreが高くても`000047`のqualificationを飛ばさず、`000047`で`applicable`でも`000027`のAcceptanceを飛ばさないことです。

```text
Similarity != Applicability
Applicability != Acceptance
Acceptance != Execution Authority
```

Gate C前は、これらの本格runtime実装へ進みません。Benchmark、feature contract、Reference Scenario等のpre-design preparationと、Product上の問いの整理を進めます。

### 2. Business Procedure / cross-view readiness

Business System Design全体は[`../../docs/business-system-design-reading-guide.md`](../../docs/business-system-design-reading-guide.md)から読みます。Lane Cでは「Referenceが動いた」ことと「contractを決めてよい」ことと「production authorityへ採用してよい」ことを分けています。

```text
000042 cross-view Reference / pressure
        ↓
000050 independent second realistic corpus
        ↓ supplies missing Evidence
000048 Contract ADR readiness
        ↓ ADR adopted
000049 production adoption readiness
        ↓ production authority cutover
Business Procedure authoringへのStudio / AI接続（000046 / 000027のshared Acceptance Boundaryを再利用）
```

[000046](000046-studio-shared-acceptance-authoring-loop.md) / [000027](000027-ai-proposal-acceptance-boundary.md)自体の着手条件は**Product Gate Cだけ**です（2026-09-19にReady済み）。v1 `order_entry`のData / Relationで先行して実証でき、上の①②の完了は待たず、Contract ADRの前提にもなりません（循環依存を作らない）。上の図の最終段は、同じshared Acceptance Boundaryが採用済みのBusiness Procedure contractへ広がる段を指します。

| Issue | 減らす不確実性 | 主な出口 |
| --- | --- | --- |
| [000042 cross-view Reference](000042-business-procedure-cross-view-reference.md) | Data / Function / Businessを横断した最小contract候補が現実的な業務で壊れないか | Pressure / checker / Evidence。判定state自体は持たない |
| [000050 second Handoff corpus](000050-second-realistic-handoff-corpus.md) | 現行Handoffが一つの受注例だけへの過適合ではないか | 000048 blocker Bを判断する独立Evidence |
| [000048 Contract ADR readiness](000048-business-procedure-contract-adr-readiness.md) | 最小production Canonical contractをADRで固定してよいだけの反証可能Evidenceが揃ったか | ADR起票・採用判断へ接続 |
| [000049 production adoption readiness](000049-business-authoring-production-adoption-readiness.md) | 採用済みcontractを実際のsource authorityへ切り替えて安全か | production authoring / authority cutover |

`000042`のrunnerがgreenでも`000048`が自動的にReadyにはなりません。`000048`でADRが採用されても`000049`のproduction採用条件を満たしたことにはなりません。

## Issueを書くときの文書品質

重要なIssueでは、チェックリストより前に次を自然な文章で説明します。

```text
背景 / 防ぎたい失敗
        ↓
減らすProduct上の不確実性
        ↓
担当境界
        ↓
現在分かっていること / 未解決
        ↓
Reference・Evidenceの取り方
        ↓
完了後に可能になる判断
        ↓
非目標
```

特に次を重視します。

- 「実装するもの」ではなく「何を明らかにする作業か」を先に書く
- Product Requirementとの関係を、RQ番号だけでなく理由でも説明する
- 隣接Issueとの責務境界を示す
- `unknown / unresolved / blocked`を、未完成だからという理由だけで埋めない
- synthetic fixture、actual external Evidence、production adoptionを同じ成功へ畳まない
- algorithm名やpackage名を、Product上の問いより先に固定しない
- 完了条件は作業量ではなく、次の判断を可能にするEvidenceへ結び付ける

短いIssueでは全見出しを機械的に設ける必要はありません。ただし、読者が「なぜこの作業が必要で、終わると何が変わるのか」を推測しなければならない状態は避けます。

## 統合の基準と確認範囲

2026-09-04時点の既存計画Issue、Master Planに残る対象、GitHubの未完了16件と直近の関連Issueを照合しました。`origin/main`の計画基線は`a1f7b4a5787518ee3ae29bf216862ad9456a6ce8`です。

ローカルに未反映だった既存Issue 000033〜000037を取り込み、同趣旨のscope再監査（既存GitHub #380 / #381）は000035へまとめました。計画Issueがなかったpartition設計（既存GitHub #43）は000041へ登録しました。本セッションの根幹機能3提案は、既存番号と衝突しない000038〜000040へ移しています。旧`plan/PRODUCT_VALUE_PROPOSALS_2026-09.md`の一覧・優先順・分担は本一覧と各Issueへ統合しました。

一つのIssueを「一つの成果と完了判定を持つ作業」にします。親Issueは横断的な到達条件を持ち、子Issueは実装・接続・Evidenceの担当範囲を持ちます。同じ解析器や採用境界を複数Issueで実装せず、別の環境・別の証明が必要なEvidenceは一つの成功へ畳みません。

今回取り込んだのは計画文書です。上記revisionで記録された実装・テスト実績と、ローカルcheckoutのコード・実行実績は区別します。特に000037のrunner実装記録は上流revisionの記録であり、この整理でコードをマージしたりactual runを行ったりしたという意味ではありません。実装・検証を始める時は対象revisionと必要なsourceの存在を確認します。

## 直近の主線

Gate Bは2026-09-19にReadyへ到達しました（candidate revision `R = e5231839a5dcc62cce1181a2731fcb7c70d16736`）。[000035](000035-gate-b-external-provider-scope-audit.md)の判断どおり、必須Evidenceは000032 / 000025 / 000031の3系統で、いずれも同じRのsame-runで成立しています。actual Oracleの未取得は継続課題として残り、Handoverへ`unresolved`として渡していますが、Gate Bの追加blockerへは戻しません。000030のHandover Baseline `B / R`も固定済みで、直近の主線は000007（第一Web Target）です。

| 担当Issue | 一つの成果・責務 | 着手・完了の接続 |
| --- | --- | --- |
| [000006 Application Driver](000006-application-driver-vertical-slice.md) | Driver全体の到達条件と残件の親 | 主要意味境界の再実装を起票せず、下記Evidenceへ接続 |
| [000032 Repository Evidence](000032-gate-b-deterministic-repository-evidence.md) | 通常Go runnerでのdeterministic実行Evidence | clean revisionで実行 |
| [000025 生成物再現性](000025-reference-generated-artifacts-refresh.md) | World / Lock / Durable Runの再生成一致 | 修復があればsource確定後にcheck-only再実行 |
| [000031 実PostgreSQL](000031-gate-b-postgres-header-detail-evidence.md) | 実engineでのheader-detailとfixture前後の安全性 | 専用空DBと固定revisionが必要 |
| [000037 三系統の最終収束](000037-gate-b-three-lane-final-evidence.md) | 上記3系統を同じcandidate revisionへ揃える | 個別runnerを再実装せずRunbookどおり照合 |
| [000029 Gate B監査](000029-gate-b-readiness-audit.md) | Evidenceに基づくReady / Not Ready判定 | runner成功だけで判定を代替しない |
| [000030 Handover](000030-gate-b-to-c-handover-baseline.md) | Ready revisionと未確認事項の固定 | 000029のReady判定後 |
| [000007 第一Web Target](000007-first-web-target-projection.md) | 同じApplication meaningのWebへのProjection | Handover固定後 |
| [000028 Gate C Evidence](000028-first-web-target-conformance-evidence.md) | Semantic / Functional / Visualの独立した証明 | 000007と同じbaselineを使用 |

実行順・安全条件は[Runbook](../GATE_B_EVIDENCE_RUNBOOK.md)、総合判定は[Gate B監査](../GATE_B_READINESS_AUDIT.md)を参照します。本一覧を第二の実行契約にはしません。

## Lane C — Business Procedureの三段readiness

上流pressure / audit文書の結論の割り当ては[`UPSTREAM_BREAKDOWN_2026-09.md`](../UPSTREAM_BREAKDOWN_2026-09.md)が正本です。Lane Cは次の三段へ分けて担当します。

| 段 | 担当Issue | 一つの成果・責務 | 判定の正本 |
| --- | --- | --- | --- |
| Reference / Evidence | [000042 cross-view Reference](000042-business-procedure-cross-view-reference.md) | checker / Pressure test / A-set / operational checklistの親 | 本Issueは判定stateを持たない |
| ① Contract ADR | [000048 Contract ADR readiness](000048-business-procedure-contract-adr-readiness.md) | blocker A / Bの解消とADR起票 | [Pre-ADR Audit](../BUSINESS_PROCEDURE_CANONICAL_CONTRACT_PRE_ADR_AUDIT_2026-09.md) |
| ② production採用 | [000049 production adoption readiness](000049-business-authoring-production-adoption-readiness.md) | adapter / shared Acceptance / fresh processの3条件 | 000049の完了条件 |
| Evidence補充 | [000050 second Handoff corpus](000050-second-realistic-handoff-corpus.md) | 二つ目のrealistic corpusでの`from/to/condition` lossless | 000048 blocker Bへ引き渡す |

③のStudio / AI統合は[000046](000046-studio-shared-acceptance-authoring-loop.md) / [000027](000027-ai-proposal-acceptance-boundary.md)がProduct Gate C後に担当します。**Lane C Pre-ADR Gate C**と**Product Gate C**は別物です。

Lane Cのactual runner greenとGate B必須Evidence三系統は別のEvidenceです。片方の成功で他方をgreenへ読み替えません。

### 3. 成熟したReference Modelの転写

[000078](000078-reference-model-transfer-gate.md)は、既存の小規模Referenceとは別に、成熟した外部業務モデル全体をTEIへ転写して表現力をpressureする横断Issueです。最初のbenchmarkにはX-TEA / XEAD系「CONCEPTWARE／受注生産」を使います。

~~~text
外部source / manual / model
        ↓ inventory
source-faithful accounting
        ↓ classify
Canonical / Realization / Native / Derived / Unresolved
        ↓
TEI source / Driver / Target
        ↓
cross-view / execution Evidence
~~~

このIssueはX-TEA互換を所有しません。転写で見つかった具体的gapを000042、000038〜000040、000026、000027、000053、000077等へ戻し、同じ機能を000078内で二重実装しません。判定契約は[Reference Model Transfer Evidence Gate](../REFERENCE_MODEL_TRANSFER_GATE_2026-10.md)を正本とします。

## 外部資産・Providerの継続課題

| 担当Issue | 統合した課題 | 分けて扱う理由 |
| --- | --- | --- |
| [000022 外部master_data Evidence](000022-external-master-data-physical-query-evidence.md) | actual Oracle facts / acceptance / actual readの統合（既存#214） | 外部観測の親。driverや権限の詳細実装を重複所有しない |
| [000033 Oracle driver](000033-gate-b-oracle-driver-integration.md) | driver選定・Host assembly・metadata実証（既存#371） | 接続条件と実driverに依存する成果 |
| [000034 Oracle read-only](000034-gate-b-oracle-read-only-evidence.md) | principal / grant / transaction / runtimeの安全な確認（既存#376） | driverの動作成功と権限保証は別の証明 |
| [000041 Partition Schema Drift](000041-partition-schema-drift-semantics.md) | 未登録のpartition設計とWorkstream Dの同じ論点（既存#43） | 設計判断と最小再現契約に限定し、DDL実行へ広げない |

000033 / 000034の実Oracle条件は未取得のままです。成果は000022へ合流させ、未完了なら000030の`unresolved`へ残します。新しい第4・第5のGate B必須laneにはしません。

## 根幹機能の3提案

新規の根幹機能は次の優先順とします。業務例と所有境界の設計は並走できますが、本格実装はGate C後を基本とし、現行Gateの完了条件へ追加しません。

| 優先 | 担当Issue | 統合した機能範囲 | 初回の完了単位 |
| --- | --- | --- | --- |
| 1 | [000038 業務Lifecycle](000038-business-lifecycle-semantics.md) | 上位計画の最小state transition、既存Validation / Capability / Observationの接続 | 受注確定の一遷移を同じ規則で定義・実行・観測する |
| 2 | [000039 Semantic差分・依存解析](000039-business-change-rehearsal.md) | source追跡、変更影響、Driver比較、Acceptanceで必要な解析 | Validation / Bindingの2種別で差分からScenarioと検証結果へ辿る |
| 3 | [000040 意味資産の探索・組み立て](000040-semantic-component-composition.md) | Domain Component / Pattern / Policy等の再利用、AIのknowledge-first探索 | 第二Applicationに少数の既存資産を明示参照で組み込む |

000038は業務意味の表現力、000039は変更時の信頼性、000040は蓄積した知識の再利用を強化します。導入評価や指標整備を別の先行Issueへせず、単一正本の変更伝播、未知の依存の明示、同じidentityの再利用を各受入条件で確認します。

### 共通機能の担当を一つにする

- 既存のSource Index / derived index / Validationは再利用します。
- 差分と依存経路の共通解析は000039が担当し、000040は候補の探索・適用先の選択を担当します。000040が解析を必要とする場合、000039の最小解析境界を利用し、完全な変更リハーサルの完成までは要求しません。
- 採用・source反映・stale Proposalの受理境界は[000027](000027-ai-proposal-acceptance-boundary.md)が担当します。000039 / 000040はAI専用の書込経路を作りません。
- 最小Editor（最初のauthoring Reference）は[000046](000046-studio-shared-acceptance-authoring-loop.md)が担当し、上記機能を利用します。Editor / AIの着手条件を記録した[000008](000008-editor-and-ai-followup-note.md)は、Gate C Readyの成立で完了しています。
- 第二Targetは[000026](000026-target-capability-execution-profile-boundary.md)が担当します。000040の第二Applicationと混同しません。
- 000038の初回実装は000039 / 000040の完成を待ちません。000040の初回例も新Lifecycleの完成を前提にせず既存資産を使います。

状態は000038 / 000039がProposed、000040がIn Progress（既存資産を使う圧力試験候補の検証中）です。schema、Core primitive、APIの採用は未決定で、必要な判断をADRへ残してから実装します。将来の汎用Workflow、分散transaction、全言語解析、Marketplace、全Provider網羅を、この小さい受入単位の完了で達成済みにしません。

000038については、2026-09-05に要件の詳細化を[業務Lifecycle v1 設計](../design/BUSINESS_LIFECYCLE_V1_DESIGN.md)としてまとめ、所有境界と実行境界の判断を[ADR-0024](../adr/0024-business-lifecycle-ownership.md) / [ADR-0025](../adr/0025-transition-evaluation-and-conflict-result.md)として提案しました。いずれも評価待ちであり、採用前に実装へ進みません。詳細化の結論として、`sales.order.submit`のCapability契約変更はGate B Ready判定と000030のHandover Baseline固定より前に入れない扱いです。

## Product Valueの確かめ方と担当

[Product Value](../../product/value/TEI_PRODUCT_VALUE.md)の「価値の確かめ方」は、個人・組織・社会の各層で観測する事実を定めています。2026-09-30に、各項目を担うIssueと、計測または検査が実在するかを照合しました。担当の無かった項目へ000071〜000074を起票しています。

| 層 | 確かめる事実 | 担当Issue | 2026-09-30の状態 |
| --- | --- | --- | --- |
| 個人 | 会話履歴や聞き取りなしに、変更に必要な文脈へ到達できる | [000053](000053-ai-native-development-and-agent-operational-knowledge.md) Phase 1 / [000040](000040-semantic-component-composition.md) | 決定記録は機械可読。再入場に足りるかを判定する検査は無い |
| 個人 | 承認の対象が、意味差分・影響・例外・未解決点として示される | **[000071](000071-approval-sufficient-projection.md)**（新規） / 000046 / 000027 | Previewは機械可読。人が読む面と、権限・native codeを示す項目が無い |
| 個人 | 一度答えた確認が理由なく繰り返されない | **[000074](000074-answered-confirmation-history.md)**（新規） | 記録も検査も無い |
| 組織 | （留保）規則変更に伴う手直しの数 | [000072](000072-rule-change-hand-edit-count.md)（研究に留保） | 価値の指標にしない。要件固有で、変更の深さで意味が変わるため。Product Valueでは、実務機能の量・関係性・実用性とMVP的な実証で確かめる |
| 組織 | 画面・API・AIエージェントの間で同じ規則が食い違わない | **[000073](000073-derive-validation-selection-from-canonical.md)**（新規） / [000014](000014-business-validation-reference.md) / [000026](000026-target-capability-execution-profile-boundary.md) | 画面とAPIは同じ評価経路。評価対象の選択がCanonicalに無い。AIエージェントの経路は無い |
| 組織 | 再利用した意味・Infrastructure知識の元identity・前提・差分を追跡できる | [000040](000040-semantic-component-composition.md) | identity / provenanceは実行可能なtestあり。Variationは表現が無い |
| 社会 | AIを介さず正本から第三者が根拠を再現できる | [000029](000029-gate-b-readiness-audit.md) / [000028](000028-first-web-target-conformance-evidence.md) / [000058](000058-gate-c-visual-independent-reevaluation.md) | Gate B / Cの独立監査あり。Visualは単一のAI評価者 |
| 社会 | 分からない状態を成功や安全として扱わない | [000009](000009-capability-outcome-and-execution-error.md) / [000041](000041-partition-schema-drift-semantics.md) | Database・Runtime境界で多数のtestあり |
| 社会 | 業務知識を読めることが実行権限へ変わらない | [000053](000053-ai-native-development-and-agent-operational-knowledge.md) | Runtime / Pluginの権限はtestあり。AIエージェントの読み取り経路は製品側に無い |

新規のうち000072は、研究に留保しています。手直しの数は要件固有で、変更の深さで意味が変わるため、測定に基づく価値判断は、問題領域の知見が蓄積されるまで行いません。他の3件の着手順は次を推奨します。

価値の確かめ方の中心は、実務シナリオを通すMVP的な実証です。最初の実証の候補は[000075](000075-practical-scenario-mvp-demonstration.md)に整理しています（owner判断待ち）。これと並行して、成熟した既存業務モデルの規模・cross-reference・暗黙前提にTEIが耐えられるかを[000078](000078-reference-model-transfer-gate.md)で確認します。

1. **000073**は、評価対象の選択をCanonicalな宣言から導く機能上の課題です。方針（ADR-0029）のowner reviewから始めます。
2. **000071**は既存Previewの机上評価まで済んでおり、条件案のowner reviewから始められます。Gap Audit §8の優先順位6（RQ-028の担当決定）に当たります。
3. **000074**は000046 / 000047が確認を出し始める前に、記録の形だけ先に決めます。

この表は各Issueの状態の正本ではありません。状態は各Issueの`状態`行、Requirementごとの実証状況は[Gap Audit](../PRODUCT_REQUIREMENT_GAP_AUDIT.md)が正本です。

## 既存計画への統合と履歴

| 元の論点・作業 | 管理先 | 扱い |
| --- | --- | --- |
| Gate Bのactual Oracle必須性再監査（既存#380 / #381） | [000035](000035-gate-b-external-provider-scope-audit.md) | 同じ設計判断へ統合。判断済みであり再起票しない |
| 3-lane scopeの長文同期（既存#383） | [000036](000036-gate-b-three-lane-document-sync.md) | 上流でDone。今回の文書取り込みを実行Evidence取得と混同しない |
| 通常Go / 生成物 / PostgreSQLの最終収束（既存#387） | [000037](000037-gate-b-three-lane-final-evidence.md) | 実装・pure test記録とactual final runを分離して維持（actual final runはGate B Ready時に取得済みでDone） |
| 最小状態遷移、履歴・Policyの意味知識 | [000038](000038-business-lifecycle-semantics.md) / [000040](000040-semantic-component-composition.md) | 前者は遷移の意味、後者は既存資産の再利用。汎用履歴基盤まで完了範囲を広げない |
| provenance / change trace、AI Acceptanceのimpact analysis | [000039](000039-business-change-rehearsal.md) / [000027](000027-ai-proposal-acceptance-boundary.md) | 解析は前者、受理は後者 |
| Editor、Projection、確認提案・判断履歴 | [000046](000046-studio-shared-acceptance-authoring-loop.md) / [000027](000027-ai-proposal-acceptance-boundary.md)（着手条件の記録は[000008](000008-editor-and-ai-followup-note.md)がDone） | 既存計画へ接続。Editor / AIの先行実装Issueを増やさない |
| Targetごとの性能・可読性 | [000028](000028-first-web-target-conformance-evidence.md) / [000026](000026-target-capability-execution-profile-boundary.md) | 各TargetのEvidenceとして扱う |
| 既存ADR移行（既存#71） | [000004](000004-migrate-existing-adrs.md) | 既存Issueを継続。新提案へ混ぜない |

一人でも維持できる規模を保つため、Master Planに列挙された長期方向を一項目一Issueへ機械的に分解しません。production HA、signed distribution、全Target展開等は、具体的な利用条件・完了Evidenceが定まる時点で既存Workstreamから切り出します。

既存の000010 / 000011には過去から同番号の異なるファイルがあります。本整理で履歴identityを一括変更せず、既存参照はファイル名まで含めて区別します。今回追加・移動した000033〜000041の番号は、確認した上流とローカルの両方で重複しないようにしています。

## Repository健全性・運用の課題

2026-09-19に、Gate B / Gate CのEvidence取得とリポジトリ全体の点検で見つかった、Product Requirement自体ではなく**運用・検証・保管**の課題です。CI（GitHub Actions）が止まっている間に、検証されないまま積み重なった不整合が、実際に実行して初めて見つかりました。

| Issue | 減らす不確実性 | 主な出口 |
| --- | --- | --- |
| [000055 検証gate](000055-verification-gate-while-ci-suspended.md) | CI停止中に、未検証の変更（コンパイルできない、testが走らない）がmainへ入り続けないか | 決定済み（script運用・main直接commit）。main緑化・health check・AGENTS.mdでの必須化を実施 |
| [000056 Issue番号・状態語彙](000056-issue-numbering-uniqueness-and-status-vocabulary.md) | Issue参照が一意に作業を指し、状態を機械的に集計できるか | 解消済み（重複8組、状態語彙21件）。検査を`go test ./...`へ自動化 |
| [000057 Evidence artifact保管](000057-gate-evidence-artifact-retention.md) | Ready判定の根拠artifactを、後から再照合できるか | GitHub Releaseへ格納済み。locatorを判定文書へ追記 |
| [000058 Gate C Visual再評価](000058-gate-c-visual-independent-reevaluation.md) | 単一のAI評価者によるVisual判定を、別の評価者が追認するか | 保留。後日、別の生成AIが再評価する |
| [000059 旧作業branch](000059-unpushed-legacy-consolidation-branches.md) | ローカルだけに残る旧整理作業の扱い | 削除済み（復元用にbundleを退避） |
| [000060 旧workflow](000060-legacy-one-shot-workflows-inventory.md) | Actions再開時に、過去の書き込み権限付きworkflowが起動しないか | 不要な17本を削除済み |

これらは既存のGate判定・Product主張を変えません。

## 番号

Repository内Issueは6桁の連番を使います。

```text
000001
000002
000003
```

ファイル名は次の形式を基本とします。

```text
000001-short-description.md
```

GitHub Issueの番号とは別に管理します。

番号の一意性と状態語彙は、`tools/repository-plan-check`のGo testが検査します（`go test ./...`に含まれます）。

2026-09-20に、重複していた8組の番号のうち、参照の少ない側を新しい番号へ移しました（`ISSUE-000056`）。

| 旧番号のfile | 新番号 | 番号を残した側 |
| --- | --- | --- |
| `000010-command-application-pattern` | 000067 | `000010-application-pattern-validation-integration` |
| `000011-order-entry-browse-reference` | 000068 | `000011-order-line-semantic-reference` |
| `000035-gate-b-actual-oracle-blocker-reaudit` | 000066 | `000035-gate-b-external-provider-scope-audit` |
| `000038-gate-b-repository-source-observation-stability` | 000061 | `000038-business-lifecycle-semantics` |
| `000039-gate-b-final-go-environment-isolation` | 000062 | `000039-business-change-rehearsal` |
| `000040-gate-b-child-go-environment-isolation` | 000063 | `000040-semantic-component-composition` |
| `000041-gate-b-postgres-fixture-ownership-marker` | 000064 | `000041-partition-schema-drift-semantics` |
| `000042-gate-b-final-goenv-launch-guard` | 000065 | `000042-business-procedure-cross-view-reference` |

## 何をRepository内Issueにするか

次のような作業は、Repository内Issueとして残す価値があります。

- 複数のPRや実験にまたがる作業
- Product Requirementとの関係を長く残したい作業
- 設計判断を行う前の調査
- 複数workstreamへ影響するGap Audit
- 別の生成AIへ委ねても背景を失いたくない作業
- 完了条件をRepository上のEvidenceで確認したい作業

一方、誤字修正や小さい局所改修まで機械的にRepository内Issueへ昇格させる必要はありません。

## 推奨する構成

各Issueでは、必要に応じて次を記述します。

- 背景
- 関連するProduct Requirement
- 現在分かっていること
- まだ分からないこと
- 制約
- 作業内容
- 非目標
- 完了条件
- 関連ADR / GitHub Issue / PR

見出しを埋めること自体を目的にはしません。小さいIssueでは不要な項目を省略して構いません。

## 状態

文書冒頭に次のいずれかを記載します。

- Proposed
- Ready
- In Progress
- Blocked
- Done
- Superseded

状態は進捗率ではなく、次に何をしてよいかを示します。