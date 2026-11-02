# Business Actionの完了条件と証拠充足：Function参照からの意味移植（2026-11）

- **状態**：実装前設計・Owner未受理。ここに示す表・記法・試験は候補であり、Core Schema、実行Status、横断Gate、製品保証ではない。
- **Owner／入口**：[Business DFD #1125](https://github.com/hat47x/tei-platform/issues/1125)、[統括 #1119](https://github.com/hat47x/tei-platform/issues/1119)。採否の正式判断は[#267](https://github.com/hat47x/tei-platform/issues/267)。
- **関連**：[#1121 Source→Candidate→Lowering](https://github.com/hat47x/tei-platform/issues/1121)、[#2505 Source版間](https://github.com/hat47x/tei-platform/issues/2505)、[#1127/#2506 Function Data Context](https://github.com/hat47x/tei-platform/issues/2506)、[#2500 View→Operation Intent](https://github.com/hat47x/tei-platform/issues/2500)、[#1129 Owner-write Rule](https://github.com/hat47x/tei-platform/issues/1129)、[#1174 Commit/Receipt](https://github.com/hat47x/tei-platform/issues/1174)、[#1131 Scenario](https://github.com/hat47x/tei-platform/issues/1131)。
- **この文書の役割**：元のFunction利用箇所を保全したまま、実行・確定の記録からOwnerが定めたBusiness Actionの**完了を説明できるか**を、一枚の設計・受入票で比較する。構造のReference、業務的な必要条件、実観測を区別する。

## 1. 機能が利用されたことと、業務上達成されたことの間

[業務システム設計](../../docs/business-system-design.md)は、Business ActionとApplication Function、実行結果と現実の観測、業務上の完了・責任引継ぎを別に扱う。原X-TEAのTaskAction→複数TaskFunctionIOやFunction内IOPanel、Button/Callの宣言は、呼出し可能性と設計時の利用関係を表す根拠にはなる。しかし、実際の一試行でどのFunctionが何度動作し、何を確定し、業務上何を終えたかは別の観測とOwner判断を要する。

以下の5層を横並びに残す。新しい共通識別子をこの設計だけで導入しない。

| 層 | 対象 | 確認できること／留保 |
| --- | --- | --- |
| D: Definition | Functionと入出力・potential Table access、Rule/型の定義版 | 1定義が1回動いたわけではない |
| U: Use site | Action→Function-IO利用箇所、Button/Call、Sourceの位置/版 | 同じFunctionでも複数箇所。SortKeyと呼出順は別 |
| I: Invocation | 実際のTargetへの呼出し、入力対象、開始/終了/異常 | 呼出しの成功と確定Effectは別 |
| K: Committed effect | Scope、OperationID/RequestHash、Receipt、Tx owner、独立Observation | Commit記録と対象・結果の業務意味は別 |
| B: Business completion | Ownerの必須条件、対象集合、最終状態、外部/手作業、引継ぎ | 根拠が十分な条件に限って完了を主張 |

既存`internal/xeadsource/crossview.go`は個々のSourceEdgeを保持し、Action/Functionペアを集約してpotential accessを派生させる。既存`TestCrossViewSeparatesSourceEdgesAndPotentialActionEffects`は同Functionの二利用箇所と一つのdistinct pairをassert済み。**計数の再実装は不要**で、残るのはConsumerがUを省略せず提示できるかの確認である（[#1125 10/31の既存票](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6538024565)）。

## 2. Owner完了条件は「対象範囲に相対的な主張」

概念比較として、ある意味版`r`・業務対象`s`・業務時点`t`の完了主張を

`CompletionClaim(r,s,t) = OwnerPredicate_r(RequiredTargetScope, FinalBusinessState, RequiredExternalEvidence, HandoffCondition)`

と記す。**実装APIや必須構文ではない**。

完了条件の候補には、次を含められる。

- **対象集合に対する条件**：全行（∀）、少なくとも一行（∃）、特定件数、特定の集合・関係edge、集計と単位を含む関係式。量化対象がEntityかPath occurrenceかgroupかを明示する。
- **必須・任意・後続許容の違い**：必要な同期状態、Ownerが許す遅延/付随処理、手作業・外部連携の確認を区別する。業務がいつ完了とみなされ、責任がいつ引き継がれるかはOwnerが判断する。
- **時点**：評価に使う業務有効日、読み取ったSnapshot、Commit時点、別接続のObservation時点、外部証憑の成立時点を区別する。必要時にどの時点を完了条件へ使うかOwnerが指定する。
- **意味版**：Rule、Actionの完了述語、許容される部分確定範囲を受理したrevision。旧Receiptの存在から新版Ruleでの適合・完了を主張しない。

機能から見える`R`（実返却行）と業務上必要な`L`（Ownerが定めた対象集合）は別。`R ⊆ L`、`R=[]`、`HTTP 200`、`Validation PASS`は、`∀x∈L: P(x)`を直接証明しない。必要に応じて、同一対象scope/版で`R=L`を保証するProvider契約、真に必要な条件を保証する独立集計、Ownerの確定した外部証憑等を利用し、**全行を必ず列挙する方式を共通必須にしない**。

## 3. 完了主張に必要な証拠と保留の粒度

単一`success`ではなく、少なくとも次の**判定欄**を置く。列は既存Issueの観測・Source/Target/Owner記録を参照し、第二の証拠正本にしない。

| 主張対象 | 必要な根拠 | 不足時の報告 |
| --- | --- | --- |
| C1: Source利用箇所 | Source版、U、Function内IO target、別ビュー対応の根拠 | callsite未解決／版間対応未受理 |
| C2: 操作対象と集合 | UI選択S、Function入力T、Context/Grain、対象L、実Read R、完全性・scope・Snapshot・関係のrole | 対象曖昧／部分取得／実対象未観測 |
| C3: 必須Rule | Accepted Rule/Attachment/phase、検証集合V、Owner-write実経路、最終関連状態、rule evaluation時点 | 検証済みの部分集合のみ／最終不変条件未確認 |
| C4: EffectとReceipt | Kと対象E、Scope/O/Hash、実Tx所有者、別Connection/Invocation read、間接/外部効果 | Effect未確認／Commit結果不明／別業務試行の疑い |
| C5: 外部/手動と引継ぎ | 人または外部システムの確認根拠、Ownerが受理したhandoff condition、時点 | 作業継続中／引継ぎ未確認 |
| C6: 受理済み完了述語 | そのAction意味版で必要なC2〜C5の条件が充足したこととOwner Decision | 未受理／完了未確認／既知の不成立 |

実際に違反を示す行が確認できる場合は**不成立**、必要な対象や時点が確認できない場合は**未確認**、真偽不明のまま異なる版の証拠が衝突する場合は**矛盾要調査**として説明する。いずれもこの設計の**人向け判定語**であり、Runtimeの新しいStatus/ResultMode/Outcomeを追加する提案ではない。

## 4. 反証可能な自作Scenarioと期待値（いずれも提案・NOT RUN）

既存の二Function F1（閲覧・確認）／F2（保守・保存）、親H、子D1={L1,L2}、別関係D2={Q1,Q2,Q3}を用いる。非公開原典固有の値やIDは流用しない。UI・SQLで同じ行が何度投影されても、実体やOwner対象の個数は変化しない。

| ケース | 二つの世界／入力差 | 業務完了の判別 |
| --- | --- | --- |
| **BC-01 全件と部分結果** | OwnerはD1全明細の数量正値を必須とする。L1=2は両世界共通。世界AではL2=1、世界BではL2=-1。しかしF1がL1のみを取得して同じValidation PASSを返す | 観測上同じRでも完了の真偽は異なる。C2の全件証拠なしなら双方`unverified`。#1063の集合欠落・#2506のSlice未取得区分を再利用 |
| **BC-02 JOINと単位** | D1=2件、D2=3件からJOIN表示6行。表示行6はD1の6実体でも「業務上6明細」でもない | LはOwnerのRelation/Grainに束縛。単位/集計/Relation edgeの意味を型付きで維持。#2500 VE-02/03の逆写像を再利用 |
| **BC-03 同じFunction成功・異なる完了** | F2がL1だけを確定してHTTP/Driver正常、Hも更新。ただしOwnerがL2の確定と親合計一致を必須としていた | K+とB未成立は共存する。未確定の必須要素・既確定効果・再送停止を区別。#1129 Owner-write/FS-1・#1174 K/Oの既存試験を借用 |
| **BC-04 最終状態の変化** | 全対象を検証した後、同Txの子EffectがL2を変更して最終Invariantを壊す | `Validate PASS`≠`FinalBusinessState PASS`。#1129 FS-1のOwner判断・fixtureへ返す |
| **BC-05 引継ぎと外部効果** | DB内の出荷依頼記録K+だが外部出荷側の受領証憑が未取得。別世界では同じKで受領まで完了 | Ownerが「依頼記録の確定」で完了とするか「相手受領」を必須とするかで判定が変わる。HandoffとObservationを分離し、新たな共通配送Engineを要しない |
| **BC-06 意味版・利用箇所の変更** | Source版A→BでF2利用箇所Uが移動/分割され、Ownerが完了条件を「一行確認」から「全行確認」へ変更した | exact SourceRef/Function名一致、旧Receipt再生、既存生成Artifactの一致だけで`B_versionB`の完了とはならない。#2505版間候補と#1174歴史的Receiptを参照。意味変更があるかはOwner受理まで候補 |

BC-01は「集合不完全」、BC-04は「完全な集合でも同Tx後続Mutation」、BC-06は「実行の意味版違い」であり**異なる反証**として保持する。Bへの帰属は宣言・推定・実観測・Owner判断を混同しない。

## 5. 実行順と各Ownerの受入境界

1. **S3／#1125（Business）**：既存`crossview_test.go`の二利用箇所・重複pairの試験は再作成せず、U別のSource→Owner説明とBC-01/03/05の**完了判定票**を作る。最初はHeadless/CUI read-only表示で、同じClaimの「充足／既知不成立／未確認」と根拠Scopeを示せるか検証。追加の意味正本や判断者は作らない。Owner受理は#267。
2. **P0-a／#1174・#1127（独立並走）**：既存実PGのpost-commit invalid-result→K/O/Lookupを担当。Business側はそのEffectを必要条件とするかのOwner判断だけを提供する。BC-03の実証を新たなDriver testと呼ばない。
3. **P0-b／#1129・#1127（独立並走）**：Data Rule→二Function PortのBind/Evaluate/Owner-write/FS-1を担当。BC-01のRule対象L/Vに必要な集合完全性とBC-04の最終Invariantを受入項目へ返す。全件問い合わせ方法やSnapshot Fenceは#1708の既存契約と一致させる。
4. **P1／#2506・#2500・#2505（並列探索）**：Result Slice／JOIN粒度（BC-02）、更新Intentの逆写像（BC-02）、Source改版と意味受理の対応（BC-06）をそれぞれ既存PoCで検証。#2はConsumer側の完了主張への影響を読む。SHO設計書とAI Contextへの投影は#1803/#2516を使用。
5. **P1／#1131（Scenario）**：同じDatasetについて、Owner受理済み述語の独立ExpectedとSimulated、実K/OからのObservedを別列で比較。シミュレーション結果を実DB/外部効果の証明に使わない。

**横断受入条件（設計候補）**：一つのBusiness Action revisionについて、必要なL・量化/Grain、U、Rule最終状態、K/O、外部/手動条件が一つのOwner受理票で追跡できること。少なくともBC-01の観測同値・業務非同値とBC-03のK+／B未完了を区別できること。#1119のG0〜G4を追加・改称しない。

## 6. 実証済み・未実証・採否の線引き

- **確認済み（固定公開mainに対するコード静的照合）**：既存CrossViewのSourceEdgeとpair集約、既存unit testに二利用箇所を区別するassertが存在すること。既存IssueにOwner-write/receiptの証拠・負例設計があり、個別の合成/限定実行結果は各Issueへ帰属する。
- **実施済みの限定合成反証（2026-11-02追補）**：下記 §8.1 の自作two-world fixtureをPython 3.13.5で実行し、1 unittest／8 subTest caseをPASS。これは業務・Providerの実装試験ではない。
- **今回未実行**：BC-01〜12の実TEI Consumer/実業務縦断、Go/Java/Windows/実DB、原モデルの実Candidate J5接合、CUI/SHO Consumer、独立Provider完全性、Owner意味の正式受理。
- **未採用**：完了述語の具体的なDSL/API/新型、Contextの永続形式、標準ステータス、全Target共通Host、業務上の正しい部分確定/再試行・引継ぎ条件。Sourceや過去のCI/merge、旧Driverのambient Commitから業務意味を自動導出しない。

## 7. Claimごとの証拠充足を評価する（2026-11-02追補）

11月2日の横断Issueでは、対象集合の完全性に加え、**同じ現在表示からは歴史的な事実や元Operationへの帰属を復元できない**反例が増えた。ここでは[#1125 BC-H1/H2/H3](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6572338466)、[#1125 BC-Q1/Q2/Q3](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6572333373)、[#1174 Effect Replay≠Result Replay](https://github.com/hat47x/tei-platform/issues/1174#issuecomment-6572369365)、[#1174 Intentと実Effect対象](https://github.com/hat47x/tei-platform/issues/1174#issuecomment-6572078615)、[#2497 View写像の識別不能性](https://github.com/hat47x/tei-platform/issues/2497#issuecomment-6572147884)から、**S3のOwner判断に必要な証拠の充足条件**だけを統合する。具体的なProvider・UI・Query・履歴取得の実装は元Issueが所有する。

### 7.1 同じ「完了」表示で混ざりやすい別々のClaim

| Claim（説明用の呼称） | 時点／対象 | 主張できる根拠 | 混同しやすいもの |
| --- | --- | --- | --- |
| **B₀ 初回Action完了** | Business occurrence B1の初回試行／当時の必要条件 | Ownerが受理したB1の完了述語、必要Effect/対象、当時の確定と観測 | 後日、親と明細が整合した現在値 |
| **B₁ 回復後の業務過程完了** | B1に正式帰属する回復Stepを含む業務過程／回復成立時点 | RecoveryのOwner受理規則、O1/O2の因果・主体・対象、必要条件 | O2が無関係な操作でも偶然同じ現在状態となったこと |
| **Sₙ 現在状態が条件を満たす** | 現在の対象集合L(t)、Ruleと有効時点 | Scope/Grain/Rule/Provider readと取得完全性・時間資格 | B1の初回成功、確認後の無変更、実Commit時の状態 |
| **Qₙ 確認を現在利用できる** | 確認対象／確認時点〜現在の関連変更区間 | 有効な業務確認根拠、Ownerの失効規則、値と対象membershipの**履歴被覆**・遅着補正 | 確認recordの存在、#958 reviewの現在状態、現値/hash一致 |
| **K₀ 過去Operationが確定した** | Scope/O/Hashと実Effect E／Commit時点 | K・対応Owner更新・独立O・必要な要求元対象との接合 | RequestHash一致だけから、UI Intent T=実E、または業務完了と断定 |
| **R₀ 初回BusinessResultを再現できる** | 当時の結果値/形成Adapter版・Output meaning | 元Resultの信頼できる保存、または当時の意味版・commit-imageによる検証可能な再構成 | ReceiptがReplayedとなり、現在Adapterが改めて作成した表示 |
| **Pₙ 現在の説明を提示できる** | 現行Consumer/開示権限・Projection版/観測時点 | 現行readと現在の説明/開示契約 | 歴史的Resultや元Action完了の証明 |

B₀/B₁/Sₙ/Qₙ/K₀/R₀/Pₙは**この設計書の見出し**であり、Canonical Type、公開API、実行ResultMode、二番目のStatus体系ではない。各Claimが必要とする根拠と、実際にその根拠が取得・検証された範囲を別に保持する。とくに`Result Replay`はOutput形成の当時版・Result値が保全されていない場合は未保証であり、K₀の成功を代用しない。

### 7.2 反証の共通原理：同一観測・異なる業務結論の二世界

固定した**Claim、Owner仮定、Context、Consumerが閲覧できる証拠E**について、Eと両立する二つの実際の状態（または履歴）`W_a`と`W_b`で、そのClaimの真偽が異なるなら、**Eだけを根拠にそのClaimを確定することはできない**。この命題は、全データを探索して「不存在」を証明する要件ではなく、**誤った十分性を二つの具体例で反証する**ための廉価な設計検査である。あるClaimに必要な根拠を追加した場合は、同じ二世界がなお識別不能か再確認する。

この判定は三つに分ける。
- **充足を支持**：Owner述語・対象scope・時点を固定し、既存Provider/Source/履歴/独立Observation等の適格な証拠から当該必要条件を支持できる。
- **不成立を確認**：そのClaimについて信頼できる反例（必要条件の違反、対象不一致、正式な撤回等）が観測された。
- **根拠不足**：可視データが同じでも異なる結論があり得る。履歴一部欠落・RLSで不可視・取得時点が不一致・Owner述語未受理などは、確認できた事実に応じて内訳を保持する。

これは**Claim相対の人向け判断**であり、特定の業務を全件読み込んで真偽決定する汎用Solverを作る提案ではない。Provider内部で十分な証拠があっても、当該Consumerへ開示できない場合、Consumer向けには「この表示からは判定不能」と示し、非可視対象の件数/存在や機微な訂正を漏らさない。内部監査の判定と閲覧者の判定が異なることは矛盾ではない。

### 7.3 11/2差分の反証ケースと、既存Ownerへの受渡し

| 追加ケース（P1候補） | 観測上等しい二世界 | #2が判定するClaim／必要Evidence | 技術・意味正本 |
| --- | --- | --- | --- |
| **BC-07 回復帰属 EC-1** | t0でB1部分確定→t1でO2により親子整合。O2がB1の正式回復Stepか、無関係な別処理かだけ異なる | `Sₙ`は同値でも`B₁`の判定は変わる。初回`B₀`の結果を後から「成功」に改変しない | [#1125 EC-1](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6557435359)、#1174、#267 |
| **BC-08 値のABAと確認撤回** | 現在は数量2。世界Aは無変更、Bは2→3→2。確認のreviewのみ撤回／正式業務確認も撤回も別 | 現値`Sₙ`、確認をした履歴、`Qₙ`の利用可能性を分ける。Owner仮説P3下で対象・期間の履歴被覆が必要 | [#1125 BC-H1/H2](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6572004348)、#958、#1708 |
| **BC-09 membership ABA/遅着** | 確認時も現在も`{L1,L2}`、世界Bは中間にL3加入→離脱。遅着正式訂正の有無も異なる | 固定集合か動的述語か、値変更だけでなくRelation membershipを含む履歴区間・cutoffの被覆を確認する | [#1125 BC-H3](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6572338466)、#2582、#2506 |
| **BC-10 Intent対象とEffect対象** | 画面は「受注1001を確定」、Receiptは同じK。片方は1001、他方はPayloadが1002を更新 | `K₀`が成立しても`B₀(target=1001)`は未成立候補。S→Intent T→Adapter Payload→Provider E→独立Oの対応が必要 | [#1174 AT-1/P1-Payload-2](https://github.com/hat47x/tei-platform/issues/1174#issuecomment-6572078615)、#2391、#2500 |
| **BC-11 Effect ReplayとOutput再投影** | K1・同O/Hashで書込み増分0だが、初回と再入場のAdapter版またはcurrent readが異なる | `K₀`の再生と`R₀`の歴史的出力再現、`Pₙ`の現在投影を別判定 | [#1174 RR-P1〜P4](https://github.com/hat47x/tei-platform/issues/1174#issuecomment-6572369365)、#2497、#2516 |
| **BC-12 集計・存在の情報損失** | 同じSUM/Chart（例200）でも元内訳が異なる。表示0件でも実在0件か閲覧Scopeの部分結果かが異なる | `Sₙ`や`B₀`の量化が必要とするEntity/Relation/Measureと単位、存在と全件不存在の証拠非対称性を保つ | [#2497 VP-03/VP-05](https://github.com/hat47x/tei-platform/issues/2497#issuecomment-6572147884)、[#2593 Measure](https://github.com/hat47x/tei-platform/issues/2593#issuecomment-6572012409)、#1063 |

BC-08/09のOwner仮説P3、BC-07の正式回復条件、BC-12の量化/再集計Lawは**未受理**。これらは原典の実業務意味を推定したものではなく、架空のOwner規則を明記する**識別不能性の反例**。選択済みSource ref／旧Driverや合成fixtureの実行と同一視しない。

## 8. PR #2518の追加受入計画：Read-only Claim説明と縦断実証の分離

**S3先行／最小Consumer判別**：BC-01、BC-07、BC-08、BC-09、BC-10から二世界の組を選び、(a)同じConsumer表示、(b)Owner仮説下の異なる真値、(c)判別のために必要な追加Evidence、(d)当該Consumerに開示可能な最小説明、を**同一のClaim・時点・対象Scope**で記録する。既存#1125・#2506・#2500・#2516のread-only票を使用し、追加の共通BusinessResult/Claim APIを作らない。最初の合格候補は、両世界に同じ「完了」を出さず、根拠不足と既知の不成立、表示可能と実行可能を区別できること。

**現行Issueの実測進捗を再利用**：[ #2505 RV-01〜12のうち合成10件の隔離実測](https://github.com/hat47x/tei-platform/issues/2505#issuecomment-6564707324)はSource対応候補の実験結果であってOwner受理ではない。[#2593／Draft #2594](https://github.com/hat47x/tei-platform/pull/2594)はSQLite 6/6の集計反例（別プロジェクトの合成試験）、[#2500／Draft #2517](https://github.com/hat47x/tei-platform/pull/2517)はread-only View→Intentの12試験を持つ。それぞれ**その局所構造の根拠**として参照し、BC-07〜12の業務完了、実Provider、原モデル完全移植のPASSに合算しない。

**既存最優先実証**：#1119 P0-a（#1174実PGのCommit後結果異常→K/O/同ID Lookup）、P0-b（#1129/#1127二Function/同Rule/Owner-write）は順序を維持。業務完了の最終判断を追加するために、この二実証へ新しい共通Engineや要求粒度の全面拡張を持ち込まない。#2518の追加成果は**Ownerが完了を主張する前提を説明するS3/P1の設計**に限定し、#1119 G0〜G4は既存のまま。

**S3 Owner判断の追加項目**：元Actionの初回完了／後続Recoveryによる業務過程完了、確認の存在／現行利用資格、固定cohort／動的cohort、承認された部分確定・遅着訂正の効力時点、Intent対象と確定Effectの必要一致、初回Resultの再取得要否、外部/手作業Handoffを**必要なClaim別**に受理する。採否と実証の正本は#267、#1125、#1174、#1129の担当へ保持する。

**実証資格**：2026-11-02 main `9c9ef4d160640908e06ad13afb6c2b45c664c748`と既存Issue／Draft PRの静的照合。BC-07〜12の新しいBusiness Claim Consumer試験、Go/PG/Java/Windows/SHO/実原モデル、Owner意味受理は**NOT RUN**。既存の合成実測とは独立。採用済みDSL/ABI/Host/共通Status/新Gateなし。

### 8.1 自作two-world fixture：表示の同一性だけではClaimを確定できない（限定実行）

[合成fixture `verify_claim_projections.py`](../fixtures/business-completion-claims/verify_claim_projections.py)を、上記§7.3のBC-01/05/07〜12設計票に対する**極小の反証データ**として追加した。

- **実行**：`python -m py_compile plan/fixtures/business-completion-claims/verify_claim_projections.py`、`python plan/fixtures/business-completion-claims/verify_claim_projections.py`。2026-11-02の隔離環境 Python 3.13.5で`Ran 1 test ... OK`、8個の`subTest`対象は全てPASS。GitHub branch上のファイルblob `e9da92a845d3ffa4203b0969c4ee411b07547106`と、試験を実行したローカルファイルの`git hash-object`一致を確認。
- **検証した条件**：各例で`visible(worldA)==visible(worldB)`かつ、**明示した合成Claimの期待真偽**がA=true／B=falseになる。対象は全件の一部取得、外部受領、正式Recoveryの帰属、値・membershipのABA、IntentとEffect対象、Receipt再生と歴史的Output、SUMの内訳である。
- **証明範囲**：与えられた情報を失うProjectionだけからは、二世界に異なるClaimを正しく判定することが不可能、という*有限個の合成反例*。見えない値が実際にあること、OwnerのP3/Recovery規則が受理済みであること、TEI Consumerが現在誤判定していること、過去のReceiptや外部効果を実測したことは証明していない。
- **次に必要な製品接続**：#2506/#2500/#2516の実read-only Consumerと、#1174/#1129が所有する実Provider/Effect証拠に対し、合成の同一表示ケースを投入し、「根拠不足」の説明と実行許可が適切に分離することを観測する。実原モデル・Owner判断は#1125/#267に留める。

このfixtureは**計画の十分性を反証する補助証拠**であり、第二のBusiness Rule Engine／汎用Oracle/Runtime Acceptanceを構成しない。追加の合成Case数を製品Gate PASSへ加算しない。


## 9. 操作意味と共有確定境界の二世界（2026-11-02追補・Claim別反証）

### 9.1 この追補だけの不足：対象・結果の一致では閉じない二つの穴

既存BC-10は「Intent対象1001／Payload・実Effect対象1002」、BC-11は「Effect Replay／歴史的Output再投影」を扱う。今回の**BC-13は対象1001・型・書込み種別がすべて一致したまま、選択されたBusiness Actionだけが変わる**反例であり、BC-10を再実装しない。**BC-14は画面の閉じる・取消・RETURN_CODEが一致しても、子Functionの共有TransactionへのCommit権限で親の保留Effectが確定する**反例であり、単なる書込み成功と異なる。

正本の責務は次のように保持する：[#2598 IB-12](https://github.com/hat47x/tei-platform/issues/2598)はIntent候補→導出Surface→実Requestの**Action Meaning束縛**、[#2485](https://github.com/hat47x/tei-platform/issues/2485)は導出Binding選択、[#1129](https://github.com/hat47x/tei-platform/issues/1129)はTransaction OwnerとNested Function Effect、[#1174](https://github.com/hat47x/tei-platform/issues/1174)はCommit・Request/Receipt・確定時Authority、[#940](https://github.com/hat47x/tei-platform/issues/940)は旧Driver実動、[#1163](https://github.com/hat47x/tei-platform/issues/1163)は同一実行の証拠接合。本書の責務は**OwnerのB₀完了Claimを支えるには何の根拠が不足するか**の説明のみ。

| Case | 可視情報が同じ二世界（自作架空データ） | その情報だけでは成立しないClaim | 判別に必要な証拠 |
| --- | --- | --- | --- |
| **BC-13 同じ対象／異なるAction意味** | UI Intentは「受注1001を提出」、表示は同じ受注ID・更新済み・Receipt K1。Aでは実呼出しCapabilityが受理されたsubmit、Bでは同じ `update sales.order` に対応するcancel。双方の更新対象・書込み種類は一致 | **B₀「提出Actionが成立」**は、対象ID・Generic Operation ID・単なる更新Receiptからは導けない | Intentの受理済みAction/Revision → 生成Surfaceの選択Binding・Rule/World Revision → Actual InvocationのCapability・入力対象/値 → Owner受理済み意味とEffect/Receiptの一試行対応。辞書順で選ばれたBindingを意味同一として受理しない |
| **BC-14 同じ取消UI／異なるAmbient Commit** | 子一覧Functionが選択なしでClose（戻りコード01）、その後親でRollbackを選ぶ画面上の列は同じ。Aでは子は親の未確定更新を確定せず、Bでは子の共有Commitが親の保留更新を先に確定する | **B₀に付随する「親取消後に当該更新が未確定」Claim**は、子の直接Write件数0・RETURN_CODE01・親Rollback操作の記録からは導けない | 親Pendingの存在、同じConnection/Transaction参加・子のCommit呼出しと実効、親Rollback前後の**別Connection**での観測・Resource版、対象Operation/Effectへの帰属。禁止すべきかはOwner判断 |

BC-13の技術的手掛かりは[#2598 IB-12](https://github.com/hat47x/tei-platform/issues/2598#issuecomment-6572593885)と未統合[#2587](https://github.com/hat47x/tei-platform/pull/2587)の固定Source所見。単に`sales.order.submit`と`sales.order.cancel`の文字列を比較すれば業務意味が自動受理されるわけではなく、Application／Capability／対象契約／モデル版／Ownerの受理済み対応が必要になる。

BC-14の**B側の限定的な旧挙動**は、[統合済み#2154](https://github.com/hat47x/tei-platform/pull/2154)の配布Driver＋合成XEAF／H2の実動記録（子XF100を選択せず閉じる→親Rollback後も親更新が残る）を参照できる。A側は**比較のための仮説世界**であり、旧Driverで両世界を実行したという主張ではない。旧配布JARと公開Java Sourceには版差があるため、元業務モデル・全Runtimeへの外挿やD01発生の証明としない。旧Driverの合成実動に対応する正式Business Actionの意味は未受理。

### 9.2 判別順序：PurposeとTransactionの二軸を独立に閉じる

1. **P1／Intent→Requestの意味束縛**：#2598の既存IB-12へBC-13をOwner Claim Consumerとして渡す。対象S=T=Eでも実際に選ばれたCapabilityが受理済みActionと異なるなら、B₀を「業務完了」にしない。Rule FingerprintやOperation IDが不変でもBindingだけ変わる版間差をSource/R1-R2・実Request側で検査する（#2485 B6〜B8を再利用）。
2. **P0-b／Nested TransactionのOwner確定**：#1129の既存二Function/同Rule/Owner-writeと#940の旧Driver証拠を利用し、BC-14では子Function自身の直接DML、**親Pendingへの確定波及**、外部PDF等の別Effectを分ける。新Host、汎用Tx Coordinator、独立Driverエージェントは作らない。Commit Request／実JDBC ACK／確定DB readを別列にする。
3. **P1／複合SaveとTxを混同しない**：[#2597](https://github.com/hat47x/tei-platform/issues/2597)・[Draft #2601](https://github.com/hat47x/tei-platform/pull/2601)の依存順序／atomicity-group／SQLite局所3ケースを参照する。正しい実行順だけで一括原子性は保証されず、対象Targetが提供する実Tx所有・Savepoint・子のCommit権限を確認する。同Fixtureの局所PASSをTEI実Target全体の保証へ変換しない。
4. **S3／Owner受理票**：完成Claimを「対象」「Action Purpose」「必要な検査集合」「確定するEffect集合」「確定境界」「結果・回復」の列で評価。BC-13/14ともB₀とK₀を混同せず、B₀の正しい条件はOwner未受理を保つ。#267の第二Meaning Acceptanceや#1119新Gateを作らない。

### 9.3 合成反証と製品受入の線引き

既存の[自作two-world fixture](../fixtures/business-completion-claims/verify_claim_projections.py)へBC-13/14の二組を追加する。まず同一の公開Projectionに対し架空OracleのB₀/取消Claimが逆になることだけを検証する。その後、**BC-13**は#2598でIntent→Actual Capability/Request、**BC-14**は#1129/1174で子Function→親Pending→独立DB readを同じCase/Revisionで結合する。実Effect帰属や正式Owner意味の判定は、#1163同一Runの証拠が得られるまで保留する。

**今回の資格**：本追加は設計候補と自作合成反例であり、旧Driver/実PG/製品Web/Windows/SHO/正式Owner受理を新しく実行・採用したものではない。#2154の既存実動、#2597のSQLite局所結果、#2598/#2485の設計所見は、それぞれの固定版・範囲に限る。成果物のPublish/Manifest一式整合は独立した[#2604](https://github.com/hat47x/tei-platform/issues/2604)が所有し、設計書のbyte/公開成功をAction Meaning受理へ昇格させない。


## 10. 二世界反証から必要証拠の取得資格へ（2026-11-02：限定的なWitness補充試験）

### 10.1 二世界の識別不能性だけでは、正しい「追加証拠」を決められない

§7・§9のBC-01/05/07〜16は、Consumerが読む可視投影 `P(W)` が等しく、Owner仮説に基づくClaim `C(W)` の真偽が異なる二世界を構成した。これは**その可視情報だけではClaimを確定できない**ことの具体的な反証である。しかし「Field Xを足せばよい」という候補と、**実Provider/Owner/Consumerの下でClaimを確定できる証拠を取得した**ことは別である。

形式的には `P(A)=P(B)`、`C(A)≠C(B)` から `P` の不足を示せる。追加投影 `Z` を与えて `(P,Z)(A)≠(P,Z)(B)` となっても、**その二世界を判別できただけ**であり、すべての実世界で `C` が確定する十分性の証明ではない。特に、`Z` が現在のSource/Target/要求/時点から取得でき、必要な範囲を被覆し、当該Consumerに開示できるかは別の検査になる。

この「不足→補充候補→資格確認」を手軽に点検するため、既存二世界fixtureを**変更・複製せず**に読む追加試験
[ `verify_claim_witness_extensions.py`](../fixtures/business-completion-claims/verify_claim_witness_extensions.py)
を置く。BC-01/05/07〜16の12例それぞれに、架空二世界を判別する**一つのWitness候補のField／確認すべき取得元／必要な資格**を対応付ける。これは仕様上の必要Evidenceの候補一覧であり、新しいClaim Solver、汎用Provider SPI、正式Evidence schema、第二業務正本ではない。

### 10.2 Witnessの資格は「値が異なる」ことより強い

| ケース | 二世界を区別する追加の手掛かり | 実製品で別途確認する資格とOwner |
| --- | --- | --- |
| **BC-01/12** 全件・粒度 | 生明細集合・内訳 | #2506/#1063：Query Scope、Grain、全件の取得被覆、read-setの整合。閲覧可能な集計から元行集合を推定しない |
| **BC-05/07** 外部受領・回復 | 外部受領記録、Recoveryの帰属 | #1125/#1174：受領先主体、Recovery Operationの因果リンク、Ownerが受理した完了条件 |
| **BC-08/09** 値／membership ABA | 確認後の値変更・関係変更の履歴 | #1125/#1708：確認対象範囲・期間全体の被覆、遅着訂正と撤回。イベント0件だけで無変更を確定しない |
| **BC-10/13** 実対象・Action Purpose | Providerの効果対象、実InvocationのCapability | #2598/#1174：Intent/Action→実Request→実Effect、Binding/Meaning Revision、Target認可、Receipt帰属 |
| **BC-11** 過去Result | 初回Outputまたは当時版による再構成素材 | #1174：当時Adapter/Outputの版・実Result値。Effect ReplayをResult Replayへ読み替えない |
| **BC-14** Ambient Commit | 親保留更新の実際の確定有無 | #940/#1129：子FunctionのCommitと親Pendingの因果、Transaction境界、独立DB read。旧Driverの実動範囲を限定 |
| **BC-15** 成果物実ロード | Consumerが読んだArtifact digest | #2604/#2482：Manifestとファイル集合の一式版、実ロードbyteとRuntime世代。Digest一致を業務意味受理へ昇格させない |
| **BC-16** 編集と永続化 | 実Providerの確定状態 | #2597/#1174：Dirty/SubmitとCommitを区別、元Request/Receipt・観測時点・必要な独立Read |

読み取れた内部Evidenceと、個々の閲覧者が利用できるEvidenceも区別する。存在の非開示、部分集合、履歴欠損、実行ログの欠落、Traceの循環上書き、取得不能を**未実行・不存在・拒否の証拠へ変換しない**。情報公開のOwnerは[#1178](https://github.com/hat47x/tei-platform/issues/1178)、Consumer説明のOwnerは[#2516](https://github.com/hat47x/tei-platform/issues/2516)。

### 10.3 補充試験の成立範囲と受入順

新fixtureには二つの局所検査を設ける。(a) 既存12ケースIDを一対一対応し、可視投影では同一だが、追加するCandidate Fieldに限れば二世界が識別できること。(b) Field値があっても、**Source資格／Scope被覆／期間被覆／Consumer開示資格**のどれかが欠ければ、その値をConsumer向けqualified witnessとして公開しないこと。

これは各Fieldを実際のDB/旧Driver/Host/SHOから取得した試験ではない。また、資格フラグは合成であり、正例フラグが揃ってもOwnerの意味受理、業務上のClaim真値、要求と効果の因果を証明しない。特にBC-08/09で空の履歴を使う場合、実Providerの期間被覆がなければ無変更を主張できない。BC-13ではCapability文字列だけで意味一致とは認定しない。

**実行順**：S3／#1125では「必要Witnessの候補と足りない資格」を説明するread-only Consumerを先に作る。P0-a（#1174実PG K/O/Lookup）・P0-b（#1129二Function Owner-write）を優先し、その実測とBC-10/13/14の対象・意味・確定の接合を行う。P1で#2506/#2516のScope相対Consumer、#2604のArtifact公開を独立に接続する。同一runで必要Evidenceを揃えたときだけ#1163の受入証拠に進み、既存#1119 G0〜G4の再定義は行わない。

**取得時点の注意**：本節はSourceと合成テスト設計の追補。各Provider/Java/.NET/WASM Targetでの実行証拠・旧原モデル業務意味・Owner受理は未確認。補充候補が一つ見つかっただけで一般的な十分性・受入済みClaimを宣言しない。

## 11. 複数の真正な証拠を誤って接合しない：証拠間の因果接合（2026-11-02追補）

### 11.1 単独Witnessの資格と、複数Witnessの接合資格は別

§10で、あるClaimの二世界を区別する追加Fieldが見つかっても、取得元・対象scope・期間・開示が適格でなければ利用できないことを確認した。次の独立した問題は、**個々に真正な証拠であっても、別のBusiness Action、別対象、別Operation、別Host/Provider世代で得られたものを一つの「完了証拠」として継ぎ合わせられる**ことである。

例：Ownerが受理した`submit(order-1001)`、`cancel(order-1001)`の実行Receipt、後から別Actorが更新した`order-1001.status=submitted`の現在readは、それぞれ真正であり得る。それらを「同じ提出Actionが確定した」という一件の歴史的証拠へ接合できるかは別。**Receiptの`RequestHash`は提出前の意味型・値・Action purposeの同値性を証明せず**、同じ値が後から観測されても**元OperationのEffectによるもの**とは確定しない（[#2598 IB-14/15の限定Go観測](https://github.com/hat47x/tei-platform/issues/2598#issuecomment-6573334283)、[#1174 Intent↔Actual Effect](https://github.com/hat47x/tei-platform/issues/1174#issuecomment-6572078615)）。

この設計のS3受入票では、同じ「証拠あり」を少なくとも次の**接合可能性の問い**へ分解する。

| 境界 | 接合できるかを確認する情報 | 異なる情報が混入する反例 |
| --- | --- | --- |
| Owner意味→Invocation | Accepted Action/purpose・Meaning版、利用箇所U、実Capability/入力、Targetの実Binding | 同じData/Effect型のsubmitとcancelを同じ業務操作として接合 |
| Invocation→投入Request | 対象ID/Relation role・Provider/Alias、変換版、実送信した不変Payload byte、Scope/O/Hash | 型付きIntentの精度喪失や別byte再直列化を「同一Request」と扱う |
| Request→Receipt | Scope＋OperationID＋RequestHashと対象Tx/Owner、Receiptの実取得先・実行時版 | 別Scopeの同名Operationや、先のattemptの他RequestのReceiptを流用 |
| Receipt→独立Effect Observation | 実更新対象E、Provider/DB namespace、観測時点、Receiptとの因果/版の接合証拠 | 別Actor・別Operationの後続更新を元の確定Effectへ帰属 |
| Effect→Business完了 | Ownerの最終対象集合L、Rule/Invariant、外部Handoff、時点と復旧の正式帰属 | K+でもOwner必要条件が成立しない／現在整合でも初回は部分確定 |
| 生成物→実Consumer | 正しく固定されたSource/Meaning/Target Build、Artifactの公開一式と実Load byte | 文書だけ新世代、実Driver/GUI/Adapterが旧世代。公開Manifestの完全性とConsumer実Loadを混同 |

特に**Attempt ID、論理OperationID、Business occurrence**は一対一ではない。ACK消失後に別Invocation（別Attempt）で**同じ不変Requestに対する既存Receiptを照会**することは正当な構成になり得る。したがって、単純な`attempt_id同値`をすべての接合に強制するのではなく、**再入場先が元OperationのScope/Hash/Ownerと同じか、実際のReceipt・Effectが元要求に因果的に接合するか**を確認する。必要な根拠が足りなければ`未確認`とする。

**再利用する既存Owner**：#1163は同一改訂/試行のEvidenceと実Consumerへの帰属、#1174は実Effect/Receipt/回復、#2598はAction/Intent/投入Requestの意味・byte、#2597はComposite Save/Tx境界、#1129はOwner-write最終Invariant、#2604はArtifact一式公開、#2505はSource版間対応、#2516/#1178は開示資格、#267はBusiness Meaningの正式受理。第二Evidence Registry、汎用Join Engine、共通Hostや新Gateを設計しない。

### 11.2 有効な接合でも、真の業務完了は別の判断

`Accepted Action↔Invocation↔Request↔Receipt↔Observation`の接合が妥当であっても、次の主張は独立である。

- その一操作の実Effectが、Owner完了述語に必要な**全対象L**と必要なRule・関係・外部処理を満たすこと。
- 当該Receiptの後の**異なるOperationによる更新・訂正・撤回**が、歴史的完成Claim／現在の確認資格へ与える影響。
- Composite Saveの全Patchに適切なIntentがあり、各Patchの単独検証がPASSしても、**並行Transactionのwrite skew**で最終Invariantが失われ得ること（[#2597 ES-I1〜5](https://github.com/hat47x/tei-platform/issues/2597#issuecomment-6573266088)）。同一Txと同時実行隔離は別保証であり、実Isolation/Lock/Version/Predicateの効果をDB側で検証する。
- Snapshot/Receipt/Manifest等がそれぞれ同じ文字列・digestを示しても、**何をhash化したか、どのProvider/Build/時点で生成・利用したか**が違う場合があること。内容一致・証拠の同時性・Owner意味の一致は別の主張。

ここで必要なのは、すべての根拠を同一の物理Transactionで取得する方式ではなく、**各根拠間で検証可能な因果・同一性・適用範囲・現在性の連鎖**である。外部Provider間の共通Snapshotを仮定せず、その環境に実在する別々の証拠と制約をOwnerに提示する。

### 11.3 自作の合成負例と観測資格

[合成fixture：`verify_evidence_join_causality.py`](../fixtures/business-completion-claims/verify_evidence_join_causality.py)で、架空の`Intent/Invocation/Receipt/Observation`4票について、**個別にもっともらしい票の組合せでも、証拠の因果接合は不成立**となることを比較した。Fixtureの`effect_ref`はテスト用の**信頼できる因果参照という仮定**であり、現行`effectcommit.Receipt`や一般DBの独立readに同名Fieldが存在するという主張ではない。実製品で同等の接合が可能かは#1174/#1163の測定対象として残す。

| Case | 合成反例 | このテストでの判別 |
| --- | --- | --- |
| **EJ-01〜03** | 同じ対象に見えるが実Actionがcancel／Meaning版が違う／Intent subjectが別 | Business Actionの受理済み意味・実Capability・対象を接合できない |
| **EJ-04〜07** | 同じOperation文字列でもScope/OperationID/投入Request byte/Receipt Effect参照が違う | 当該不変Requestが生んだReceiptであるとは主張できない |
| **EJ-08〜10** | 後続readが別Scope/別業務対象/別Effectによるもの（現在値は同じ） | 今回OperationによるEffectという因果帰属を示せない |
| **EJ-11（正例）** | 完全に接合した合成4票 | あくまで`correlation_candidate`。Business完了PASSではない |
| **EJ-12（再入場正例）** | Attempt AからAttempt Bへ変わってもScope/O/Hash/Effectの接合が維持 | Attempt ID一致だけを要件にしない |
| **EJ-13（欠損）** | ReceiptまたはObserverがない | `unresolved`。Effect不存在や失敗の証明ではない |

**限定的に実行済み**：2026-11-02の隔離Python環境で`py_compile`と`python verify_evidence_join_causality.py`。4 `unittest` methodsをPASS（うち十種類の不一致は`subTest`で検査）。実行ローカルのGit blob SHA `5a193d8af05beb988693e5b0df988997af035240`と、GitHubブランチ上の同一ファイルblobが一致することを確認。

**未実行／未採用**：TEI本体の`effectcommit`/Driver/Owner Business Result/GUIへの試験組込み、実PGと並行Txの受入、実Consumer Artifact Load、原X-TEA Business意味、Owner受理。合成`candidate_join`の判定関数は検証用の簡易比較器であり、製品の業務完了判定器やAPI採用ではない。

### 11.4 既存実装計画への追加（S3/P1、P0を優先継続）

- **S3 #1125**：Owner完了票C1〜C6へ`Witnessの単独資格`に加え、`Witness間の因果接合資格`の列を設ける。まずBC-10/13/14/16に対し、**それぞれ適格な証拠が存在しても、別Operationから組み合わせたら同じClaimは成立しない**ことをRead-onlyで説明する。#267の正式受理を代行しない。
- **P1 #2598/#1163**：投入Request byte・Meaning/Action・実Capability・Effect対象・実消費Artifactを一つの限定Scenarioで接合。再入場時のAttempt変化と同一論理Operationの正例を明示し、原Requestを別byteで再生成しても同じ意味だから自動Retry可とは判定しない。
- **既存P0-a #1174**：実PG後Commit Result異常→Receipt/独立read/Lookupは先行。観測が本当に**当該K/O/ScopeのEffect**かは別証拠欄で判定する。観測値だけの一致は十分条件としない。
- **既存P0-b #1129/#1127**：二Function Smart Data同Rule適用は先行。並行write skewやNested Commitの実証は#2597/#1129で別に進め、Rule検証結果からComposite Save/Business completionの成立を自動昇格させない。
- **独立P1 #2604/#2516**：同じ設計書に記載されたTarget/Ruleと、実ConsumerがロードしたArtifact/Action Bindingの世代を照合する。SHO/MCP/CUIへ展開するとき、説明用の可視性と内部Witnessの利用資格は別。

上記は既存[#1119 Critical Path](https://github.com/hat47x/tei-platform/issues/1119)のS0〜S5/G0〜G4を拡張・改称するものではない。原モデル1→TEI統合モデル1→外部Host向け成果物／SHO設計書という生成の流れの中で、**「生成/保存/読取/観測」それぞれが本当に同じ必要な意味・一式・Actionを扱ったか**をOwner受理可能な範囲で説明するための設計追補である。

## 12. 業務完了証拠を支えるデータ取得の資格：Everything is Dataから実Provider／Consumerへ

### 12.1 本節が補う境界：正しい業務結果と適格なデータ取得は別

[Everything is Data #2619](https://github.com/hat47x/tei-platform/issues/2619)／[標準データRole #2620](https://github.com/hat47x/tei-platform/issues/2620)／[明示Context Mapping #2624](https://github.com/hat47x/tei-platform/pull/2624)／[Providerの実Read Trace #2628](https://github.com/hat47x/tei-platform/pull/2628)／[複数Provider経路 #2627](https://github.com/hat47x/tei-platform/pull/2627)／[Syntax sugar #2625](https://github.com/hat47x/tei-platform/pull/2625)により、Sourceの意味上のData依存から実Readへ進む際に**複数の異なる保証**があることが明確になった。

§10は単独Witnessの出所・対象・期間・開示、§11は真正な複数Witnessを一つのOperationへ因果接合するための証拠である。本節ではさらに、**(a) 業務Claimが成立したか、(b) 何のデータを要求したか、(c) Providerが実際にどれを読んだか、(d) 取得が許容されたか、(e) その結果や証拠を誰に説明できるか**を別の判定欄とする。

業務上正しい結果が生成されても、読取時の情報取得に問題がある可能性がある。逆に許容された取得であっても、そのデータだけで「業務完了」を証明できるとは限らない。閲覧者への開示を控えることは、内部で証拠が存在しないことも、業務失敗も意味しない。

| 境界 | 主張できる範囲 | 不足または異なる場合 |
| --- | --- | --- |
| **Business requirement**（#1125/#267） | Owner受理済みRule・対象集合L・必要な履歴/Final Invariant/外部Handoffを特定 | 真の完了条件・量化・時点が未受理ならBusiness completionは未確定 |
| **Declared logical footprint**（#2506/#2624） | Function Data Contextに明示したRelation/Field/Role/Grain/Read Scope、要求complete | `Relation`がモデルにあるだけで自動的に取得対象へ加えない。宣言された範囲≠実Read≠実取得complete |
| **Actual physical acquisition**（#2622/#2628/#2387） | 実Providerが実際にアクセスしたField/Relation/Query対象/Principal/時点・取得証拠 | 表示JSONが同値でも未宣言の機微Fieldを余分に読んだ可能性。Trace欠損は「無アクセス」ではなく**取得状況未確認** |
| **Provider route / temporal qualification**（#2611/#2627/#1708） | Primary/Replica/Archive/OData等、実Alias/namespace、実Readの時点・取得可能な保証 | 同じ`sales.order`の値でも、異Providerで同等のcurrent snapshot/commit fenceを推論できない |
| **Acquisition authorization**（既存Host/Provider Authority） | 実際の読取が当該Purpose/Principal/Policyに許されたか | 過剰読取の存在が即Policy違反とは限らない。補助join-key等は**許可と必要性**の実証次第。TEI独自中央認可器を追加しない |
| **Consumer-facing disclosure**（#2516/#1178） | 当該SHO/Studio/CUI/AI Consumerへ伝えられる値・Claim・根拠要約 | 内部で十分なWitnessを持っていても、個別行や差分件数の開示が許可されたことにはならない |
| **Executed effect / outcome**（#1174/#1129/#1163） | 当該ActionのRequest→Receipt→独立Effectの因果とOwner完了条件 | Read資格・Rule役割がPASSでも実Commit、対象一致、業務完了を代替しない |

**Data minimizationも別の問い**：宣言通りの出力を得たという有限感度検査（#2624）は、「余分な物理Readがなかった」証拠ではない。一方、実Readに宣言外の補助Fieldが含まれたという事実だけでも、許容の有無や漏洩の有無は確定しない。実Providerのアクセス、Policy適合、Consumerへの情報送出をそれぞれ比較する。Observerに許されない機微なField名や対象件数は説明文へ漏らさない。

### 12.2 Source Roleを業務Ruleに接合する際の確認

Markdown表、構造化Record、インデントDSL、Typed Graphviz、メソッドチェーンは、同じ標準的な構造候補へ正規化できる場合がある（#2625の局所23件）。しかし**構文同値／Role候補／Owner Accepted Meaning／実Run Evidenceは別**。

たとえば`| quantity | positive |`という表が、受理済みValidation Rule・文書内の引用例・独立Expectedのどれに当たるかは、Source occurrence、意味Role/Operator版、Ownerの受理済み対応を参照して決める。意味モデルに付与した`Rule`という役割だけで、原業務のOwner完了述語や実Hostの書込み権限を得たことにはならない。引用文書や未受理のAI提案をActionの実行根拠にしない。

同じBusiness Claimに使用する`Source/Meaning`・`Function Context Mapping`・`Provider Route/Actual Acquisition`・`Evidence Witness`・`Consumer Disclosure`を**既存ID/Refで照合する比較票**を用いる。ここで別のCanonical Claim/Trace/Policy schemaや監査Registryの導入を先決しない。原モデル1→統合モデル1→外部Host LibraryとSHO設計書の導出に、実際に消費したSource/Artifact版を重ねて検証する。

### 12.3 自作合成Reference：AC-01〜06（限定実行済み）

[fixture `verify_claim_evidence_acquisition.py`](../fixtures/business-completion-claims/verify_claim_evidence_acquisition.py)は、**同じ表示値・合成完了Witnessの真値でも取得・利用・開示の資格が異なる**ことを説明する有限比較である。

| ID | 合成観測・期待 | 誤認を防ぐ対象 |
| --- | --- | --- |
| **AC-01** | `order.id/status`だけ読む世界と、許可されない`customer.private_note`も取得する世界は同じ表示と完了Witnessを持つ。架空Policy下では取得資格が異なる | 同じResult/業務完了なら物理Readが許容された、と判断する |
| **AC-02** | ProviderのRead Traceが欠落したとき`unknown` | 取得証拠がないことを「読んでいない」／「違反した」へ変える |
| **AC-03** | 同じ値のPrimary currentと時点保証が確認できないReplica | Route/時点の資格を値一致やSource IDから継承する |
| **AC-04** | 同じ形の「受理済みRule」と「引用されたExample」 | Markdown表・データRoleだけでOwner Ruleの受理とみなす |
| **AC-05** | 内部の証拠は同じだがConsumerへの開示資格が異なる | 内部Witnessの存在から外部共有を自動許可する |
| **AC-06** | 合成Policyで許可された`order.join_key`を物理補助列として読み、表示対象は不変 | 宣言外のReadを常に違反、あるいは常に許容と機械判定する |

**実行の証拠**：隔離Python **3.13.5**で`python -m py_compile verify_claim_evidence_acquisition.py`をPASS、`python verify_claim_evidence_acquisition.py`を実行して**6/6 unittest PASS**。ローカル実行ファイルのGit blob `5fbb01ba5a657badb5ec76ebf2167d1014d5b619`＝GitHub branch上の保存Blobを確認。SHA256 `ac6b351e60d7d6baeab2781b8f8d5c8bd04ba7d262633c7339126ae89aa3bec9`。

**証明の限界**：このfixtureの`allowed_fields`、`source_role`、`completion_witness`、`provider_route`などは自作入力の仮定である。実TEI ProviderがどのFieldを読んだか、実Policyでアクセスが認められるか、業務Ownerが完了を認めるか、外部Consumerへデータを送信したかを測定していない。実SQL/RDB/OData、Studio/SHO、外部Host、製品Goの結合テスト、正式Owner受理は**NOT RUN**。6件は**分類上の独立性の実測**であり、Business Claim Conformanceや情報非干渉のPASSではない。

### 12.4 既存計画への具体的な追加（S3の判定欄とP1の証拠接合）

- **#1125 S3／#267 Owner判断**：C1〜C6のOwner判断票に、`当該Claimの証拠充足`／`取得根拠と現在性`／`取得の適格性`／`Consumerへの開示資格`を**別列**として追補。拒否事実・根拠不足・非開示を混同しない。Owner意味受理の代行はしない。
- **#2506/#2624／#2586**：同一固定Meaning/Contextで宣言されたRelation/Field/Grain/Scopeと、実Provider Readの実対象・完全性の取得可能範囲を接合。#2624の20件合成感度試験を、物理Read不存在の証明に合算しない。
- **#2622/#2627/#2628/#2387**：実ProviderごとのQuery/Request・Raw Read trace・Principal/Route/時点・場合による補助列を既存Adapter/Hostの契約範囲で観測。取得不足と過剰取得、取得時点の違いを残す。#2627独立Go12件と#2628合成Python12件は**局所Reference**であり、TEI製品Providerの証拠は別に必要。
- **#2620/#2621/#2625、#2516/#1178**：Source Role/構文のAuthorityと、開示の資格を各Consumerに保つ。信頼できない引用文書の表が見えても業務Ruleへ昇格させず、未開示の存在や件数を説明文へ漏らさない。
- **#1174/#1129/#1163・#2604**：既存のEffect Commit/Receipt、複合Tx・Rule実行、同一Run／実Loaded Artifactの証拠を継続。今回の合成fixtureがActionの本番実行・Grantを与えることはない。
- **実行順**：P0-a 実PGのpost-commit異常→K/O/Lookup、P0-b 二FunctionのOwner-writeを優先維持。S3/P1としてまずAC-01/02/03/05をread-only Consumerから説明できるか検証し、後続で実Provider Read TraceとSHO/AI向け開示を接続。#1119のG0〜G4・既存担当Issueの受入資格を変更しない。

以上は**Business Actionの完了と、その証拠の取扱いの適格性を分ける、既存構造の上で成立する設計条件**として計画へ組み込む。新Central Host/Registry/Provider Capability/Result Status/Policy Engine・正式DSLの採用を行わない。

## 13. 完了の歴史的証拠と現在の利用資格：Claimごとの選択的再資格化

### 13.1 §12からの新しい問い：正しかった証拠が「今も使える」条件

§10の単独Witness、§11の因果接合、§12の実取得・開示適格性を満たしたという**過去の記録**と、現在のConsumer/Business Actionが同じEvidenceを**利用・再説明・実行判断へ採用できる資格**は異なる。[#1125 BC-TOU1](https://github.com/hat47x/tei-platform/issues/1125#issuecomment-6578854995)が指摘する取得時と開示時のGrantの別時点、[#2600 増分ビルドとEvidence再利用](https://github.com/hat47x/tei-platform/issues/2600)、[#2638 Role Plugin依存Pin](https://github.com/hat47x/tei-platform/pull/2638)、[#2604 複数Artifactの一式公開](https://github.com/hat47x/tei-platform/issues/2604)を踏まえ、**何が変わったらどのClaimだけを再資格化するか**というS3の受入票を追加する。

例：t0に受注1001のEffect K1が実際に確定し、t1でそのConsumerの開示Grantが失効した場合、t0に起きたK1を遡及して「未確定」と書き換えない。ただしt2のSHO/AIへK1の機微な証拠を説明・送出してよい資格は独立に再評価する。また確認後の対象集合membershipが「加入→脱退」と変化して現在値が戻っても、Ownerの仮説P3では確認の現在利用資格は変わり得る。現在のRead snapshot/画面表示一致は区間履歴の被覆を代用しない。

逆に文書の純粋な説明文だけが変わった場合、当該Business Claimがその説明文に依存せず、**依存が閉じていると実証できる場合に限り**、過去Receiptや独立Effectの資格再検査を省略する候補にできる。文書変更一般が安全という意味ではない。Markdownは実行RoleやOwner Ruleにもなり得るため、変更が純粋な説明である根拠が必要になる。

### 13.2 Claim相対の小さな影響判定票（新Core APIではない）

| 歴史的／現在のClaim | 証拠の保存と現在再利用 | 変更時に必要な限定再検査 |
| --- | --- | --- |
| **過去の確定Effect `K₀`** | 過去Receipt・独立Effect観測は過去時点の証拠として保存。現在Host GrantやConsumer開示が変わっても歴史的結果は消さない | Receipt真正性やEffect帰属に関わる**実際の証拠破損・取り違え**があれば、その歴史的主張自体を再点検。リプレイ可能性や現在の閲覧権は別 |
| **確認の現在利用資格 `Qₙ`** | 確認を行った歴史的事実と、いまSubmitに使えるかを別にする | Owner Rule改版、対象値・Relation membershipの履歴、Provider Route/時点の依存が影響するか点検。対象区間の変更履歴・閉包がunknownなら利用可を宣言しない |
| **現在の開示資格 `Dₙ`** | 過去の取得適格性と現在の開示を別にする | Consumer/目的/経路に対するDisclosure Grant、Source Role、実ロードArtifact世代の変化。内部Witnessの存在やt0取得Grantだけで現在開示を許可しない |
| **現在のSubmit候補資格 `Iₙ`** | 保存済みIntent候補と現在Host上の実行許可は別 | Rule/Action Binding、確認の現在利用、Provider Route/対象、現GrantとCommit時の前提。過去の検査成功と実行直前の保証を混同しない |

この票の`K₀/Qₙ/Dₙ/Iₙ`と依存分類は**比較用**であり正式schema、独立Revalidation Engine、全世界で通用する履歴/Policy依存マップではない。Owner固有のRule/意味、具体ProviderのResource版・時点、Runtime実Grant検査を既存Issueのまま保持する。

**判定候補は三つ**：
1. **`recheck-affected-claim`**：変更が当該Claimの既知の依存と同Scopeに当たる。以前の結論を現在へ無条件に再使用せず、該当する既存Ownerに渡す。これは「業務失効が確定」ではない。
2. **`reuse-candidate-for-this-change`**：依存集合の閉包と変更対象の無関係性が十分に確認できる場合、**今回の変更による再検査を省略できる候補**。他の未観測変更、履歴期間、実Grant、Owner承認が有効という証明ではない。
3. **`unresolved-impact`**：依存関係の列挙自体が不足、変更種類が未分類、Scopeが曖昧、対象外の根拠がない場合。**全Claim強制失効／すべて自動再利用**の双方を避け、該当範囲を追加観測する。

同じSemantic IDを持つ別Providerの時点、異Alias/tenant、同じ名前の別Artifact、QueryのWHERE/ORDER等の非表示依存を**機械的な「無関係」**へ変換しない。反対に変更対象が別IDでも、共通Rule、関係述語、全件集計、親子の区間変更がそのClaimへ影響する可能性を考慮する。Scope非重複は単にID文字列が違うことではなく、対象となる依存関係が不交差であることの確認が必要である。

### 13.3 CQ-01〜13：自作の限定合成実測と証明できないこと

[Pythonの自作fixture `verify_claim_requalification.py`](../fixtures/business-completion-claims/verify_claim_requalification.py)は、架空の4種類のClaim（歴史的Effect、現在確認、現在開示、Submit候補）と、明示的な合成依存集合・Changeを固定し、**どのClaimを再点検候補にするか**を比較する。

| Case | 反証／期待 |
| --- | --- |
| **CQ-01** | 既知の説明文だけの変更で、依存閉包が確認できるClaimは選択的再点検省略候補。全ての文書変更を安全とは扱わない |
| **CQ-02** | 開示Grant失効→現在開示は再点検、過去の確定Effectを改竄しない |
| **CQ-03** | 明細対象の加入→離脱→現在値復帰でも、現在確認とSubmit候補を再点検。過去確定Effectは別 |
| **CQ-04** | Source Role / Plugin改版→現行説明の解釈資格を再点検。Receiptの歴史的byte/Effectと同一視しない |
| **CQ-05** | Provider Route変更→現在に相対的な確認/SubmitのWitnessを再点検、過去確定Effectと区別 |
| **CQ-06** | Consumer Artifact版変更→現在開示・実ロードの確認を再点検。歴史的K1は維持 |
| **CQ-07〜08** | 依存集合が閉じていない、または変更Scopeが不明なら`unresolved-impact` |
| **CQ-09〜10** | 依存上の対象外と**証明できる場合**に限って省略候補。別IDというだけでは`unresolved-impact` |
| **CQ-11** | 変更種別がunknownなら無変更と推定しない |
| **CQ-12** | 実Host Grant変更→現在Submit資格を再点検。既存K1を取り消さない |
| **CQ-13** | Receipt真正性の問題なら**歴史的Effectの根拠**を再点検。現在確認Claimへ影響すると無条件に推定しない |

**限定実行済み**：隔離Python **3.13.5**で`py_compile`と`python verify_claim_requalification.py`を実行し、**13 unittest PASS**。実行ファイルとブランチ保存ファイルのGit blob SHA **`c67ac134aaf5fdf6c05375299fad4f7ef0ad961a`**の一致を照合。これは*仮の依存閉包／Scope非重複が真であるとした場合の、表の論理的な使い分け*しか検証していない。実TEIのDependency Graph、実Queryの隠れた述語、実Provider/Grant、実履歴、Plugin pinの再解決、AI/SHO/Studio実ロードの検証は**NOT RUN**。既存の取引真正性を保証する暗号/信頼チェーンは新設しない。

### 13.4 既存計画への反映：変更種類から再実証先を選び、P0は維持

- **#1125 S3／#267**：Owner判定票に`Claim/as-of`、`当時の証拠と現在の利用目的`、`証拠依存の閉包の根拠`、`変更内容/Scope/期間`、`今回の再点検対象`、`業務上の失効条件はOwner未受理か`を追加。§7のClaim別資格と§12の取得・開示を再利用し、第二のCompletion Statusを作らない。BC-TOU1は今回CQ-02のConsumer向け具体例として保持。
- **#2600/#2595/#2638**：Build/Artifact選択的再実行と、Business Claimの**現在再資格化**を別契約で測定。旧Result byteの再利用、履歴Evidenceの保持、現在Grant/Rule/Role Pluginの再検査を別欄にする。閉包が不明ならReuse PASSへ昇格しない。新Cache Scheduler/Role Resolverはこの設計書で作らない。
- **#1708/#2506/#2582/#2622/#2628**：Claim依存に含まれるSnapshot、値・membership区間履歴、Query predicate、実Provider Route/Read Scopeを実証し、ID違いだけで無関係とは宣言しない。
- **#2516/#1178/#2604/#2631**：現在のConsumer開示/標準Studio表示/実ロードArtifactで、再提示に必要な再資格化を観測。AIやSHOへの以前の送信履歴と今回の開示を混同しない。
- **#1174/#1129/#1163**：歴史的Effectの真正性・当時Request接合、実Submit時点のGrant/Rule/Txを既存P0-a/P0-bの実証へ戻す。Consumerの現在開示資格変化を元Effectの再Commit/自動補償の引き金にしない。

**実行順**：P0-a実PG Result異常→Receipt/独立O/Lookup、P0-b二Function同Rule Owner-writeは従来どおり先行。S3/P1としてまずCQ-02/03/07/10を既存read-only Consumerへ投影し、「旧証拠の存在」「今回の再点検」「現在提示／Action利用可否」「根拠不足」を別々に表示できるか確認する。#1119 G0〜G4の追加・改称、独自TEI Host/Policy Engine/Registry、Owner Meaningの自動採用は行わない。

## 14. Claim依存の閉包と、変更観測窓の被覆は別：選択的再資格化のTemporal Witness（2026-11-02追補）

### 14.1 CQ-01〜13の前提を実運用で検証する

§13の`reuse-candidate-for-this-change`は「*入力として与えられた一変更*が、閉じた依存集合に影響しない」という限定的な分類である。依存の列挙が閉じていても、**確認の根拠を取得してから現在の説明・Action候補を決めるまでの全変更を実際に観測できた**ことにはならない。とくに変更通知に欠落・遅延・再接続による穴があると、観測された変更が全て無関係でも、未観測の対象行加入・Relation retarget・Rule改版を見落とし得る。

選択的再点検の省略**候補**を説明するには、少なくとも以下を個別に照合する（正式APIや一般的な履歴完全性証明を要求するものではない）。

1. **Claimと依存**：Owner受理された必要条件のうち、Queryの非出力Predicate/ORDER/NOT EXISTS、Relation path、空集合への新規加入も含めて、実際に利用した依存が閉じているか（#2506/#2387/#2582）。閉包が不明なら`unresolved-impact`。
2. **観測対象と時点**：当初証拠のCheckpointから、今回の判断時点のCutまで、どの**実Provider/tenant/alias/schema/route**のどのChange Streamに依拠したか。Provider AとBの連番・文字列Snapshotが同じでも同じ時点とは仮定しない（#2622/#2627/#1708）。
3. **変更窓の被覆**：各必要Streamで、区間全体が欠落・遅延・未処理・再購読漏れなく被覆されたことを**実取得の証拠**で確認できるか。単なる現在値一致、受信したイベント0件、watermark文字列、Consumerの`last_seen`だけでは無変更の保証にならない。欠落を検知したら最新の適格な再Readへ進むか`unresolved`に留める。
4. **変更と依存の交差**：各変更の型・対象・実Scope/Predicate/Relation bindingが当該Claimと無関係であると*証拠付きで*確認する。行IDが異なるだけでは全件集合・共有Rule・否定条件への無影響を証明できない。1件でも既知の影響があれば再検査、無関係性が不明なら`unresolved`。
5. **判断後の資格**：この比較から得られるのは「あるCutまでの、あるClaimに対する再点検省略候補」のみ。Cut以降の変更と、現在のConsumer開示Grant・実Action/Commit Authority・Receipt真正性は別。開示直前／確定直前に必要な検査は#2516/#1174の既存Ownerに委譲し、検査と送信・外部副作用の原子性を仮定しない。

### 14.2 CWT-01〜12 合成反例（Product E2Eではない）

[独立fixture `verify_claim_change_coverage.py`](../fixtures/business-completion-claims/verify_claim_change_coverage.py)は、§13の一変更判定を**変更窓の被覆条件を与えた合成比較**へ拡張する。架空の`StreamNeed(checkpoint, decision_position, binding)`、`StreamCoverage(first,last,gapless,event_set_complete)`、`Change(category,relation,position)`を入力し、次を反証する。

| Case | 期待する限定結果 |
| --- | --- |
| CWT-01/10 | 閉じた依存＋同じ実Bindingで完全に被覆された変更窓に、無関係と示された変更しかない → 当該窓に限る`reuse-candidate` |
| CWT-02/03/11 | 通知Gap／Cut前のwatermark遅延／受信イベント集合の完全性未確認 → `unresolved`。空イベント配列は無変更の証明ではない |
| CWT-04/12 | membershipの加入→脱退、またはOwner Ruleの既知変更が観測された → `recheck`。最終値復帰でも履歴Claimを自動維持しない |
| CWT-05/08/09 | 依存Predicateが未閉包／Scope交差が不明／イベント分類不能 → `unresolved` |
| CWT-06/07 | 必要な二番目のProvider窓がない／同Stream名でも実tenant/Bindingが異なる → `unresolved` |

**本fixtureの重要な非保証**：`dependencies_closed`、`gapless`、`event_set_complete`、`proven-disjoint`はテスト作者が真偽を注入する**仮定**であって、実TEIのChange Detector/SQL/DB/Providerがその性質を持つことの観測ではない。`decision_position`も外部Provider間の共通時計・線形化時点を提供しない。この試験は**分類が楽観的に`reuse`へ倒れる条件を明示するのみ**で、現在のBusiness completion/Disclosure/Submit許可を計算するEngineではない。

### 14.3 S3の引渡しと最小実証順

- **#1125 S3／#267**：Ownerへ返す判定票に`Claim/as-of`・対象集合L・元Evidence・依存閉包の出所・必要Stream/Checkpoint/Cut・観測Gap・`影響あり／局所無関係候補／不明`・未受理条件を分離する。歴史的K/Effectを現在のGrant失効で消去しない。
- **#2582/#2506/#2387**：Query非出力依存、空集合への負の依存、Relation pathの再束縛、membership ABAを**実データの変更観測**へ接続する。合成CWT-04を本番で被覆できるとは仮定しない。
- **#2622/#2627/#1708**：Provider毎の実Read Scope/Route/時点と、変化検知の完全性が主張可能かを別に実証する。複数Providerに共通Snapshotや全履歴CDCを必須にするのではなく、**保証できるClaim範囲だけ**残す。
- **#1174/#2598/#2516/#2604**：変更窓Cut後のAction Request再束縛、Commit時Grant/Receipt、開示直前の資格、実ロード成果物の版を別資格として検査する。
- **#1119**：P0-a実PG K/O/Lookup、P0-b二Function Owner-write、S0/J5およびG0〜G4を先行。P1の最小製品実証はまず一Provider・一Claimで「既知変更」「観測Gap」「完全性不明」をread-only診断し、次に異Providerと開示Consumerへ接続する。新Host/新中央Claim Engine/共通Event Registry/新Gateは追加しない。

**検証状態**：この追補は隔離した自作Python有限fixtureのみ。実TEI Go/Provider・Change Stream・PG/DB・元X-TEA意味・SHO/AI/外部Target・Owner正式受理は**NOT RUN/NOT ACCEPTED**。既存§13のCQ-01〜13の結果を製品資格へ昇格しない。
