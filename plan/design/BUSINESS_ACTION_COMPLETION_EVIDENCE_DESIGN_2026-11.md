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

[合成fixture `verify_claim_projections.py`](../fixtures/business-completion-claims/verify_claim_projections.py)を、[BC-01/05/07〜12の設計票](#73-112差分の反証ケースと既存ownerへの受渡し)に対する**極小の反証データ**として追加した。

- **実行**：`python -m py_compile plan/fixtures/business-completion-claims/verify_claim_projections.py`、`python plan/fixtures/business-completion-claims/verify_claim_projections.py`。2026-11-02の隔離環境 Python 3.13.5で`Ran 1 test ... OK`、8個の`subTest`対象は全てPASS。GitHub branch上のファイルblob `e9da92a845d3ffa4203b0969c4ee411b07547106`と、試験を実行したローカルファイルの`git hash-object`一致を確認。
- **検証した条件**：各例で`visible(worldA)==visible(worldB)`かつ、**明示した合成Claimの期待真偽**がA=true／B=falseになる。対象は全件の一部取得、外部受領、正式Recoveryの帰属、値・membershipのABA、IntentとEffect対象、Receipt再生と歴史的Output、SUMの内訳である。
- **証明範囲**：与えられた情報を失うProjectionだけからは、二世界に異なるClaimを正しく判定することが不可能、という*有限個の合成反例*。見えない値が実際にあること、OwnerのP3/Recovery規則が受理済みであること、TEI Consumerが現在誤判定していること、過去のReceiptや外部効果を実測したことは証明していない。
- **次に必要な製品接続**：#2506/#2500/#2516の実read-only Consumerと、#1174/#1129が所有する実Provider/Effect証拠に対し、合成の同一表示ケースを投入し、「根拠不足」の説明と実行許可が適切に分離することを観測する。実原モデル・Owner判断は#1125/#267に留める。

このfixtureは**計画の十分性を反証する補助証拠**であり、第二のBusiness Rule Engine／汎用Oracle/Runtime Acceptanceを構成しない。追加の合成Case数を製品Gate PASSへ加算しない。
