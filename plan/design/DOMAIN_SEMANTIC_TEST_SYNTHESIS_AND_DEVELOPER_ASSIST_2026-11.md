# TEI ドメイン意味駆動のテストデータ合成・整合検証・開発者支援（設計Draft）

- 起票日：2026-11-07
- 段階：**要求統合／実装候補／実行・意味受理は未実施**。本ファイル自体はGo機能・CLI・Plugin・実行資格の実装を主張しない
- 主担当：[#1131](https://github.com/hat47x/tei-platform/issues/1131) Dataset/Snapshot/Scenario/Expectation、[#1126](https://github.com/hat47x/tei-platform/issues/1126) Data/Key/FD/Value Domain、[#2811](https://github.com/hat47x/tei-platform/pull/2811) AI-first検収、[#2514](https://github.com/hat47x/tei-platform/issues/2514) 独立Expected/反証可能性
- 参照Owner：[#1180](https://github.com/hat47x/tei-platform/issues/1180) Scenario SSOT、[#1129](https://github.com/hat47x/tei-platform/issues/1129) Lifecycle/Rule phase、[#1125](https://github.com/hat47x/tei-platform/issues/1125) Business completion、[#2506](https://github.com/hat47x/tei-platform/issues/2506) typed Context、[#2586](https://github.com/hat47x/tei-platform/issues/2586) Product actual Read、[#1163](https://github.com/hat47x/tei-platform/issues/1163) one-revision evidence、[#2386](https://github.com/hat47x/tei-platform/issues/2386) 組織・代理・有効期間、[#2495](https://github.com/hat47x/tei-platform/issues/2495) Facet/Units
- 方針：モデル正本・Generator Candidate・固定独立Expected・評価器Actual・実Provider Observation・Owner Meaning Decisionを分離。新規の第二Scenario SSOT、第二Model/Owner Registry、AI用業務意味正本を設けない

## 1. 目標と非目標

**目標**：ドメイン専門家が受理したData/Function/Businessの契約を、生成AIと人間が同じCUI/設計書から辿り、(a)最小の妥当な型付きDataset、(b)意図的に一つの制約を破る負例、(c)複数制約の衝突・相互作用、(d)業務イベントと状態差分、(e)組織・データソース・時間を跨ぐ不変条件を提示する。候補の生成、検査、欠けた証拠、失敗原因、修正案を開発者に見せ、必要な時だけ既存Targetへ実行する。

**非目標**：現在のDB観測値から業務の許容値域・業務キー・永続化ルールを自動受理すること、LLMがExpectedを自己採点すること、実用Host/認可/本番DBに自動書込みすること、全業種に対する完全な制約充足器、汎用Event Broker/独立Workflowエンジン、強属性の未確認語義をCoreに固定すること。

## 2. 三要素分析法の扱い：原義とTEI拡張を混同しない

| 参照モデル | 正本とテスト設計への入力 | 自動提案できる検証義務 | やってはいけない推論 |
| --- | --- | --- | --- |
| **Data Model** | Entity/AttributeのSemantic Type・Value Domain、Key/FD、Relation role/Occurrence、Cardinality/optional、単位・精度、業務有効期間・履歴、Ruleへの直接参照 | 値域境界、NULL/missing/withheld、Key衝突、FD違反、孤立FK、親子最小/最大、多重Path、単位/丸め/時点の矛盾 | SQLのPK/FK/実列値だけから業務Key・FD・許容値域・意味Relationを自動確定 |
| **Function Model** | Function/Pattern、UI・帳票・API・バッチ、Query input/output、Data Operation、必須非出力列、検証/編集/検索、Consumer/表示型のBind | 入力条件・表示対象・計算依存の差、選択解除/非表示、正しくないField Binding、TOP1/partial、read/write/grant | 画面表の行数を実体件数へ、表示済みAction Offerを実行権限へ変換 |
| **Business Model** | **イベント/契機を持つ業務ノードとデータソース・ノード間のデータフロー（DFD型）**、目的・責任・入出力、Business Procedure/手作業・判断・例外・引継ぎ、独立Workflow Instance | 未対応イベント、必要なInput欠損、複数業務のReader/Writer不整合、未実施手作業、差戻し、取消、補償、時間条件、完了Claimの不足 | 業務モデルを単に業務ノード間の矢印列とみなす／Data Flowだけから時間順・認可・Txを確定 |

三要素分析法のData/Function/Businessという意味的区分を維持しつつ、TEI独自のTyped Context、Meaning/Authority/Effect/Observedの証拠層を**拡張**として明記する。Business Procedure・Workflow Instance・Event Delivery・DB Commitは相異なるモデル。業務フローを実行された履歴と混同しない。

**「強属性」の保留**：原典の一次定義は現時点で未確認（[#2495記録](https://github.com/hat47x/tei-platform/issues/2495#issuecomment-6683264065)）。仮に「強属性」というラベルを受け取った場合は `sourceKind=legacy-unverified` の候補として、元Sourceの位置・版・説明・対象Entity/Attributeを表示するのみ。Unique/Required/Identifying/FD/Closed/Exclusive/Exhaustiveへ**無条件マッピングしない**。原典の対応表とOwner受理後に限り、それぞれの正規Semantic Constraintへ束縛し、対応不能ならunresolvedに留める。

## 3. ドメイン意味から導く「検証義務」の読み取り専用投影

新たな業務制約正本ではなく、既存Accepted Data/Function/Business Meaningから**Derived Test Obligation**を計算する。最初は既存#1131型付きDatasetと#1180保存Scenarioの説明・未充足診断として表示し、永続Schema/新CLIコマンドを固定しない。

候補の最小情報：`obligation ID`（局所・派生）、`source locator/span + revision`、`accepted meaning status`、`role/path/identity/grain`、`constraint/predicate/operator version`、`input dependency closure`、`required read scope/epoch/completeness/disclosure`、`normal/negative/metamorphic expected class`、`generator support`、`test host`、`qualification ceiling`。

- **D群**：型/値域/Key/FD/Relation/NULL/単位・丸め/時間と、SubTypeでのClosed・Exclusive・Exhaustiveを別制約として扱う。業務Idの同一表記が別Tenantに存在しても別Occurrence。
- **F群**：Query/Commandに必要なInput、投影に含まれないWHERE/Rule依存、Result Sliceの全件/partial/page、同一ContextのForm/Table/Chartへの意味保存、複合更新の逆写像、現在Grantとの独立性。
- **B群**：イベントの発火要件、業務Input/Outputデータフロー、担当/代理/承認、業務有効時間、手作業確認・差戻し/再開、業務完了と単なるProvider effectの差。
- **X群（横断）**：同一資源の予約数量の保存則、受注額/請求額の整合、在庫と出荷/返品の対応、複数Readerの時点差、Tenant/組織/DB間の同一性、履歴/通知/冪等性。X群は単一表CHECKに押し込まない。
- **C群（認知）**：Sourceが未ロード、Meaning未受理、Reader未観測、部分集合、古いRevision、独立Expected未確定、Authority不明、テスト自体がSKIPの場合の**検証不足**。

定義が競合・未受理・未ロードなら`unsupported / unknown / no-solution / insufficient-evidence`を区別。矛盾を「データ生成できなかったから0件」と表示しない。

## 4. テストデータの合成と検査

### 4.1 生成戦略（Generatorの差替えはPlugin候補）

1. **Deterministic Basis**：seed・InputモデルRevision・生成器版を固定し、受理済みValue Domainの最小値・最大値・境界内外、列挙・Closed/Exclusive/Exhaustive、Decimal精度/丸め、Unicode、0/空/明示NULL/missing/not-selected/withheldを区別。Physical値の妥当性とSemantic値の妥当性は別検査。
2. **Relation-aware Synthesis**：業務Identityと物理PK、Relation occurrence/role、1:1/1:N/N:M、選択可能な孤立・重複・多重Parent、2×3 JOIN行増幅、空集合・部分集合を生成。正常Datasetの参照閉包と、意図的な欠落負例を明示区別。
3. **Constraint-directed Negative**：原則として1つの制約だけを反転し、残りを満たせるなら保持する。制約が相互依存して単独違反が不可能なら、その事実と最小競合集合を出し、無理に「単一違反」としない。
4. **Combinatorial**：型・値域・状態・役割・時点・Tenant・Providerの2-wayから開始し、リスクの高い条件で3-way以上へ拡張。被覆は**受理済み因子の組み合わせ**に対するもので、全業務意味網羅ではない。
5. **Stateful/Business Sequence**：Event→Data Source入出力→Function/手作業→状態差分を展開。create/submit/approve/reject/cancel/retry、Concurrent update、ACK lost、外部結果不明。実行可能なAction候補とOwner Authority/Provider Effectを別に扱う。
6. **Metamorphic / Differential**：Tenantや無関係な別Entityの追加、並び替え、同価なRepresentation/SQL、Provider差替え、A→B→Aなどを変形して、保存されるべき性質と**保存されない条件**を明示。JOIN↔EXISTSや部分集合などは無条件同値としない。
7. **Counterexample Shrink**：失敗を保つ最小Dataset/最短Event列を求める。ただしKey/FD/Relation/有効時間/必要Ancestor/Source provenance/再現Seedを破壊して「小さくなったが同じ欠陥でない」例を採用しない。

LLMはドメイン語彙を用いて目的・欠けた条件・候補Dataset/変形/反例を提案できる。**正式制約の解釈・充足・業務Expected/合否の判定は独立した根拠と既存deterministic評価器へ**。同一DSL Rule/SQLでGeneratorとExpectedを両方作る場合は`shared-origin`警告を出す。

### 4.2 データ品質の二段評価

- **Fixture validity**：自動生成した正例は受理された型/Relation/Value Domain/Constraintを満たすか。負例は意図したConstraintを確実に破っているか。他制約の偶発違反と、予期した拒否理由を記録する。
- **Execution evidence**：テストRun/Query/Providerが実際にこのFixtureを読んだか、同一Snapshot/Scope/Owner Lに属するか、Resultが完全か。Go test PASS、SQL Mock一致、実DB観測、Consumer実受領、Owner意味受理を別に記録する。

本番データの値分布からDomain候補を採る場合は[#1126](https://github.com/hat47x/tei-platform/issues/1126#issuecomment-6516063140)のread-only・権限・観測Scopeを使用。PII/秘密のデータをLLMへの既定入力、公開Repoのfixture、アーティファクトやログへ自動転写しない。合成を原則とし、実データは許可・目的・保存期間が確認できる場合だけ用途限定。

## 5. 開発者支援（人間・AI共通のCUI優先）

**A. テスト設計の説明**：対象Function/Data/Businessを指定し、「どのConstraintが実在し受理済みか／テスト候補に昇格できない制約は何か」を理由と出典で提示。正例・負例・境界・関係欠落・手動業務・副作用をひとつの見通し図にする。

**B. テストデータ生成／編集**：公開の合成Datasetから局所Snapshotを作り、親子/参照Key・Value Domain・精度/単位・状態/時刻を同時に検査。編集時にどのBusiness Claim/Function Result/Scenarioが影響されるかを提示。Sourceへ書くIntentとPreviewを分け、正式Accepted Meaningを自動書換えしない。

**C. 失敗の意味診断**：単なるAssertion違反だけでなく、`source-not-loaded`／`unaccepted-meaning`／`generator-unsupported`／`unsatisfiable`／`fixture-invalid`／`predicate-input-not-fetched`／`scope-mismatch`／`read-partial`／`wrong-grain`／`stale-attempt`／`execution-not-run`／`rule-violated`／`owner-review-required`に相当する既存診断を**派生提示**する。新しい製品Error Enumは先行採用せず既存出力語彙とのFit/Gapを記録。

**D. 最小再現・変更影響**：生成seed/失敗時最小Dataset/短縮Event列/独立Expected/Actual/Source・Operator・Provider版を保管。変更したValue Domain/Key/Relation/Workflow/RuleがどのCaseの生成条件とExpectedを失効させるか差分表示。ただし既知依存外のテストを根拠なく全無効化しない。

**E. より高次の支援**：未検証の重要パス、制約矛盾・死んだ業務分岐、届かないFunction、データだけからは分からない承認責務、同じ結果を返す異なるSQLの非同値、広域不変条件のOwner不明を提案。Rule/モデルの自動変更はProposal→Review→正式Acceptance→reload→独立回帰という既存経路へ戻す。

初回は既存`tei inspect / validate / plan / preview`等の**実在するCLIを調査**し、その説明/テストArtifactに派生出力を接続する候補を比較する。未実装の`tei test generate`等を動作済みコマンドとして案内しない。GUIはHeadlessの契約が確立した後、同じContext/Scenarioから投影。

## 6. 固定された最小受入試験候補（独立Expectedは事前承認）

| ID | 入力・操作 | 期待／対象と証拠上限 | Owner |
| --- | --- | --- | --- |
| DS-01 | 数量Domain `1..9`、`0`・`1`・`9`・`10`、NULL/missing | 1,9 は値域内、0,10 は値域外。NULLとmissingは独立。DomainがAcceptedでなければ未資格 | #1126/#1131 |
| DS-02 | 同じ候補Business Keyを別Tenant/同Tenantに生成 | 同Tenantの複合業務Key衝突と、別Tenantで同値の別Entityを区別。物理PKとの同一視は禁止 | #1126 |
| DS-03 | H→D1=2・D2=3・JOIN=6、D2 TOP1 | JOINの6行≠実体6件。D2の一部から全件/NOT EXISTSを断定不可 | #2506/#1131 |
| DS-04 | 閉じた状態集合とExclusive/Exhaustiveを独立設定、OTHER候補 | Closed/Exclusive/Exhaustiveは別検査。既存の列挙観測から自動確定しない | #2495/#1126 |
| DS-05 | 原典ラベル`強属性`だけ入力、意味定義なし | 生成規則に展開せず`unresolved`。対応表受理後だけ相当する正式Constraintへ | #1126/#950 |
| DS-06 | amount = qty×unitPrice、Null/丸め/通貨Unitを変える | 未承認の丸め/変換をExpectedに勝手に補うことは禁止 | #1129/#2514 |
| BF-01 | 受注確定Event→在庫予約→承認→出荷。手作業承認未観測 | 予約実Effectだけで業務Completedを認定せず、manual/外部はunknown | #1125/#1131 |
| BF-02 | 確定後cancel→retry、ACK消失、同じOperationID | Receipt照会とIdempotencyを区別、同一IDを無条件2回確定しない。実DBは別Gate | #1129/#1174 |
| XI-01 | 在庫3、異なる2注文が各3予約。同時実行 | シリアライズ可能な条件下で成功は最大1件。実Tx/後Readがない合成PASSを実保証扱いしない | #1129/#1163 |
| XI-02 | 兼務・代理権失効・組織有効日cut、申請者本人承認 | Membership/OccupancyとGrantを分離、承認時点Rule未受理なら判断保留 | #2386/#1125 |
| XI-03 | 2つのReadが同一CI/DBだが別Snapshot、片方TOP1 | 全件性や時点一致を合成しない。Owner Lと実Provider/Scope/epochを独立比較 | #2506/#1163 |
| XI-04 | 同一表/異なるUnit、履歴訂正、返金負数、明細合計vs帳票合計 | 単位/有効時間/負数/丸めの明示条件なしに広域保存則を肯定しない | #2495/#1126 |
| QX-01 | `$select=orderId`でphysical`NULL AS status`、格納値READY | 未選択スロットnil≠DB NULL、非開示/未取得/評価済みを別に扱う。既存[#2514 Q08](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6703955401)実PG参照 | #2514 |
| QX-02 | `ordered_at >= Oct2`をSQL WHEREで実評価、OutputはOrderIDのみ | 非出力Predicateで9だけ返す正例。別P=`status=READY`まで証明しない。既存[#2514 Q09](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6703988929)実PG参照 | #2514 |
| MT-01 | Rule判定反転とRelationKey末尾欠落の2種Mutation | 固定独立Expectedを変えず固有AI Scenarioで双方の実Failを確認済み（[#3017](https://github.com/hat47x/tei-platform/pull/3017)、[#3128](https://github.com/hat47x/tei-platform/pull/3128)）。他Constraintの検出率へ一般化しない | #2514/#2811 |

DS/BF/XIの**新規生成・Go test・実Provider実行はNOT RUN**。QX/MTの既存履歴を引用するが本Draftそのものの実行結果ではない。QX-02は`ordered_at`のSQL非出力Predicateであり、QX-01の`status`評価ではない。負例は「不正データの保存成功を確認する」のでなく、どの段階で拒否すべきか・どこまで観測できるかを試験する。

## 7. 受入順序・測定・導入判断

| 段階 | 最小作業と成果物 | 成功条件 | 明示的停止線 |
| --- | --- | --- | --- |
| **P0 読取専用Obligation棚卸し** | #1126のSemantic Schema、#1131 Dataset、#1129 Rule、#1180 Scenarioを実HEADで照合。DS-01/02/03/05の発見・根拠/Unknownを出す | 事前固定Expectedで過不足/誤推論を分類。テストデータ生成以前にRequiredとUnavailableを明示 | Source未読込・強属性語義不明は`unresolved` |
| **P1 決定論的Fixture合成** | Seed/Revision固定で正常/単一違反/複合違反を生成し、既存Validator/Rule実装と独立Expectedで検査。失敗縮小案をレビュー | 再生成bytes/レコードIDとViolation identity一致。意図しない副次違反を記録 | 生成済み≠意味受理・実DB整合 |
| **P2 Business Scenario生成** | DFD型イベント/データフローと業務手順を別に投影し、BF-01/02の前提・Event列・manual unknown・Snapshot差分を#1180へ接続 | 既存Scenario/Expectedと衝突せず、未実施を埋めない | 新Workflow Host/不可視Effectを創作しない |
| **P3 相互作用探索** | 関係-aware mutation、2-way/3-way、metamorphic、制約保持Shrink。#2514のKnown Mutantへ新Caseを追加する場合は独立検出力を計測 | Mutation対象の特定Test名と実Failを再現。Positiveを壊さず負例を検出 | 既存2種Mutant killを全体Coverageにしない |
| **P4 実Provider・二Consumer検収** | #2586の実Function Query→DB/Read、#2506 Data Context、Form/Table、#1163一改訂Evidenceを一Claimで連結。実PGの必須非SKIP Gateを使用 | Source/Target/Runtime/Provider/Read/Consumer/Expected同版で、complete/partial/NULL/withheldの独立受入 | Reference SQL Read≠製品E2。Owner/Hostは独立 |
| **P5 AI支援と性能測定** | AI提案 vs 明示制約のみ vs 生成器Plugin方式を比較。無効率・生成時間・最小反例の説明性・Review負荷・見落とし/誤検出・Test mutation killを#2018へ | 再現性/独立性/Target差を指標として実測、性能悪化と保守負担を記録 | LLM自己採点やデータ出所不明の高Coverageを採用しない |

**測定単位**を固定：発見したObligationの種類/Source対応率、実生成可能率、unsatisfiable/unsupported、独立Expectedにより正負区別できたCase、最小反例の正当性、Mutation kill（有効なtarget test実行を分母にする）、モデル変更による再生成/再試験範囲、未完了のOwner/Host/Provider Gate、AIに必要なTokens・人手レビュー量。生成件数だけを品質KPIにしない。

**初期の実装選択**：Go単一バイナリ方針を尊重し、最初は既存Go test＋純粋Generator Adapter/既存Scenarioの一時Fixtureを優先。PBTの生成・縮小の研究参照は [Hypothesis Stateful](https://hypothesis.readthedocs.io/en/latest/stateful.html)、[rapid for Go](https://github.com/flyingmutant/rapid/blob/master/doc.go)、組合せ試験は[NIST SP800-142](https://csrc.nist.gov/pubs/sp/800/142/final)。これらのライブラリ採用・依存追加は**未決定**。必要に応じてPlugin比較を行う。既存の#1131のGenerator範囲を侵さず、Configだけの二重実装を避ける。

## 8. このDraftで確認済み／未確認

- **確認**：#1131は型付きDataset/Scenario/ExpectationのOwner、#1126はKey/FD/Domain、#1180はScenario SSOT。#1131既存コメントは値域境界・正負Relation・決定論的Seed・LLM独立検査を**すでに提案**しているため、本案は三要素/業務Event/横断不変条件/開発者の欠測説明という**追加の統合差分**に限定した。
- **確認**：#2495の過去コメントで強属性の一次定義が未確認、#2514 Q05〜09および#3238の専用実PG試験の一部が成功という限定履歴がある。ただし同一Source/新HEAD／正式Business Completionへは転載しない。
- **未実施**：本設計に基づく新たなDataset生成Engine/テスト実行、User-accepted Domain/強属性の語義調査・新Owner Meaning受理、全ての異Provider/異Tenantでの完全性、AI提案の独立評価、二Consumer製品E2E。
- **必要な最初のOwner判断**：原典の「強属性」が指す属性分類と、業務イベント/DFDにおける実データフローのAccepted部分。本文の例示値域・業務制約は**テスト用合成仮定**であり正式顧客仕様ではない。


## 9. 現行mainのGo Sourceに対するFit/Gap（実物照合）

2026-11-07時点のmain Go Sourceを、本文の理想仕様から区別して照合した。ここでの**未存在**は、確認した型・入口に明示フィールドがないという局所観察であり、他Package/Pluginの同等機能が存在しない証明ではない。

| Source上で確認した具体的な既存要素 | P0で再利用できること | 追加のFit/Gap・不正な推論 |
| --- | --- | --- |
| `internal/semantic/model.go` blob `ff8d555f412586f055aef3658edcc59a6626adfc`：`Type{ID,Kind,Underlying,Values}` | `Values`の**明示列挙**から閉じた候補値を提案、TypeRefを通じてFieldと接続 | `Values`が示されただけでClosed/Exclusive/Exhaustive・NULL扱い・Business Domainの有効期間を自動受理しない。数値範囲1..9を含む汎用Range型はこのType宣言にはない |
| 同`Attribute{TypeRef,Required,Immutable,Role,Value}` | `Required`など**直接宣言済み**の形式制約からBoundary候補・更新禁止候補を提示 | `Value *string`のnil/empty/valueは**書式の値Clauseの三状態**であり、Runtime DB NULL・未取得・非開示ではない。「強属性」の一次語義を`Required`や`Immutable`へ自動結合しない |
| 同`Data{CandidateKeys,FunctionalDependencies,Relations}`と`Relation{Cardinality,Keys,OnDelete}` | 複合Key・明示FD・RelationKeyの対応次数、one/many、deletion ruleの独立負例候補を導ける | CandidateKey/FDは**設計者宣言**。全実DBで既に保証されているとの推論や、`CardinalityMany`→具体的最大件数の導出は禁止。LogicalとPhysical IDを分ける |
| 同`ValidationKind={positive,relation-unique}` | この2種に限って実行器への適合・Known Mutant接続を直接試せる（#3017/#3128参照） | 任意Value Domain Range/FD/Cross Entity Ruleの**汎用実行器はこのEnumでは表現しない**。Generator Candidateを全Validation Kind実行可能と誤申告しない |
| 同`Period/Calendar/EffectRule/RunPlan` | 期間Cut・RunPlan/phase・Effect前後の候補Scenarioを提案 | RunPlanはScheduler/Authorityではない。Calendarの未宣言値を補わず、業務Event/DFD Flowの意味をこれらの型から推測しない |
| `internal/application/model.go` blob `797d21e9f2edccb8a26de3ff61c4b42280380507`：`Pattern{Data,Relation,Query,Capability,Datasource,Filters,Sorts,PageSize,Interaction}` | Pattern→Query・Filter/Sort・PageとData Roleから、Functionの必要入力・結果粒度の不足を提案 | Pattern/Interaction＝実Provider ReadやCurrent Grantではない。非出力Rule依存を表示列に置き換えない |
| `internal/semantic/completion.go` blob `e42610e9159e02ba3b5c7b84c46742a846331e64`：`EvaluateCompletion` | Complete/Partial/Unestablished、空集合、局所TargetResultの独立検証をReferenceで再利用 | `CoverageComplete`・`Holds`は**Caller自己申告**。関係Role/集合L/Read scope・実Predicateを同関数が独立証明するわけではない |

**最初の実装上の判断**：P0は既存`semantic.Type.Values`、`Attribute.Required`、`Data.CandidateKeys/FunctionalDependencies`、`Relation.Keys/Cardinality`、`ValidationKind`、`application.Pattern`の**読み取り専用列挙と出典付け**から開始し、列挙/必須/複合キー/明示FDの4系列のObligation候補とUnsupportedを示す。いきなりRange/MECE/強属性/広域不変条件の汎用Core型を追加せず、表現できない圧力ケースの差分をOwner判断へ戻す。このGo型の直接確認は**source-static**であり、提案したGeneratorの実Go Test/CLI/Provider実行を意味しない。

### #3238の現在性と意味境界

別レーンの[Draft #3238](https://github.com/hat47x/tei-platform/pull/3238)は新HEADで同時進行しており、`ordered_at >= :from`を非出力のまま実PGで検査するSubtestは[Actions #42351796330](https://github.com/hat47x/tei-platform/actions/runs/42351796330) **completed/success**。**同じ値のRowsetでもQuery args/Limitが異なれば母集合の資格は異なる**。これを新Generatorの規則候補へ引き継ぐが、新Draftで合格実行を再実施したという主張ではない。
