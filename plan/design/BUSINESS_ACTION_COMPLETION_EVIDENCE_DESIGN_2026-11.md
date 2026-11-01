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
- **今回未実行**：BC-01〜06の新Scenario、Go/Java/Windows/実DB、原モデルの実Candidate J5接合、CUI/SHO Consumer、独立Provider完整性、Owner意味の正式受理。
- **未採用**：完了述語の具体的なDSL/API/新型、Contextの永続形式、標準ステータス、全Target共通Host、業務上の正しい部分確定/再試行・引継ぎ条件。Sourceや過去のCI/merge、旧Driverのambient Commitから業務意味を自動導出しない。
