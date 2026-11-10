# TEIスクリプトの権限委任と実際の操作対象を結び付ける設計

- **状態**：設計提案・未採用（2026-11-02）。この文書は仕様の正式採用、実装完了、権限付与、本番利用の許可を意味しない。
- **担当課題**：[Delegated Authority Bridge #2499](https://github.com/hat47x/tei-platform/issues/2499)（業務判断から限定権限への受渡し）、[Script / User Macro #2502](https://github.com/hat47x/tei-platform/issues/2502)（製品体験）、[Script Execution Policy #2504](https://github.com/hat47x/tei-platform/issues/2504)（実行隔離）。
- **実行・意味の正本**：[Effect Fence #1174](https://github.com/hat47x/tei-platform/issues/1174)、[Data lifecycle #1129](https://github.com/hat47x/tei-platform/issues/1129)、[組織意味 #2386](https://github.com/hat47x/tei-platform/issues/2386)、[Service Contract #951](https://github.com/hat47x/tei-platform/issues/951)、[Execution Context #1118](https://github.com/hat47x/tei-platform/issues/1118)。
- **製品化の実証計画**：[スクリプト・利用者マクロの製品化と権限委任の実証計画](../workstreams/TEI_SCRIPT_MACRO_PRODUCT_PLAN_2026-11.md)（S0〜S4と担当・受渡し）。本書は意味・権限・効果を扱い、計画は実証順序を扱う。
- **計画との関係**：[横断計画 #1119](https://github.com/hat47x/tei-platform/issues/1119)、[拡張ポートフォリオ #2493](https://github.com/hat47x/tei-platform/issues/2493)。既存P0-a/P0-bとGate A/B/Cは変更しない。
- **参照した現行コード**：main `9c9ef4d160640908e06ad13afb6c2b45c664c748` の `internal/security/effectcommit/commit.go`、`tools/reference-first-web-target/runner_order_submit_store.go`、`runner_order_submit_postgres_test.go`（静的確認）。
- **外部連携**：[TEI Batch #1130](https://github.com/hat47x/tei-platform/issues/1130)、[TAI機密境界 #232](https://github.com/hat47x/tai-pleroma/issues/232)、[EKIの同一Attempt出力断絶 #465](https://github.com/hat47x/eki-fabric/issues/465)。

## 1. なぜ新たな接合が必要か

TEIでは、組織ごとの承認・合議・代理・自動判断の手順を通常の業務モデルで自由に実装する。マクロ作成者、承認者、公開者、起動者、実行時の主体、権限発行者を同一人物と仮定しない。開発者・管理者の信頼済みスクリプトには開発環境の判断に応じて低水準機能を広く開放できる。一方、利用者向けマクロは公開された能力だけを行使できるようにする。

この方針のもとでは、**業務上の承認結果が正しく、実行Grantが有効でも、実際の操作対象や作用の範囲が承認された内容と一致しない**ことがあり得る。たとえば「受注1001に限る」という委任で、上位Functionは受注1001を指しているが、下位提供元へ渡した入力が受注1002を指す場合である。

現行[Effect Fence #1174のAT-1/P1-Payload-2負例](https://github.com/hat47x/tei-platform/issues/1174#issuecomment-6572078615)では、`runnerSubmitRequest`が通常生成する`OperationID=order-submit/1001`と、実際のPayload中の`orderId=1001`を、試験用に**Payloadのみ1002へ差し替える**場合が具体化されている。`RequestHash`は渡されたPayloadを正しく束縛するが、`OperationID`の文字列とPayloadの業務上の対象が一致することまでは保証しない。Provider側の試験DDLはPayloadの`orderId`を用いる。**この不整合による実DB効果は未実測であり、現時点の本番脆弱性を意味しない。**

したがって、以下は独立して確認する必要がある。

1. 組織の業務判断が許可した目的・対象・期間。
2. 権限発行者が実際に発行できる範囲と、発行した限定権限。
3. 承認したScript/依存機能と、実際にロードされた実装。
4. 起動時の操作意図・対象と、Adapterが渡した実入力。
5. 子Function・データ連鎖処理が実際に変更する対象と、確定時点の権限。
6. 確定済みの効果、当時返した結果、現在表示できる情報。

それぞれ既存の意味モデル・実行契約・実行環境へ責任を分担し、第二の共通Hostや認可エンジンを追加しない。

## 2. 接合の中心：操作意図・実入力・実効果

概念上、次の識別を分ける。英字はこの文書の説明用であり、新しいCore型や必須識別子形式ではない。

| 記号 | 対象と責務 | 最低限確かめたいこと |
| --- | --- | --- |
| **I** | 利用者が選択した業務Actionの**操作意図**（#1127/#2500） | 対象の意味ID、粒度、役割、変更する内容、基準版、作用の意図 |
| **D** | Workflowによる**業務判断**（#2386/#1125） | 許した操作・目的・承認規則・対象解決方法・時点・根拠 |
| **G** | 信頼済み発行者から得た**実行Grant**（#2499/#951/#1118） | 許されたService Contract、作用、対象範囲、Tenant、期限、実行主体、委任条件 |
| **L** | 実ロードしたScriptと依存Providerの**実装集合**（#943/#1370/#2390） | 承認対象版との関係、必要な依存・実行能力、実ロード元 |
| **P** | Adapter/子Functionへ実際に渡す**入力と操作対象**（#1121/#1174） | IからPへの意図した対応・対象変換が成立するか |
| **E** | Ownerが実際に確定しようとする、または確定した**効果**（#1129/#1174） | 対象、操作、作用集合、Transaction境界、権限、更新後の不変条件 |
| **O** | 独立した**観測**と各種Receipt（#1174/#1131） | 実際にどの対象へ何が起きたか、観測時点・取得範囲 |
| **V** | 利用者・AI・外部Hostへの**結果の表示・開示**（#2516/#2504） | 何の履歴／現在値で、誰にどこまで開示してよいか |

概念的な接続は次のとおり。

~~~text
TEI Business/Organization Workflow
     -> D: 目的・対象・判断時点を持つ業務判断
     -> Issuer / Host: Dと発行権・Policyを照合してGを得る

Macro source + accepted bindings -> 実際にロードしたL
User/UI/CLI intent I -> Action/Adapter request P
     -> 個別Functionと子Capabilityの対象変換の検査
     -> Owner write E （実際に許された作用だけを確定）
     -> 独立観測O
     -> 開示条件つきの表示V
~~~

重要な関係は、**Iの対象文字列とPやEの対象IDが常に同じこと**ではない。受注1001の確定が、その明細と在庫予約を正当に変更する場合、Eの対象は1001以外にも広がる。その場合は、受理されたBusiness Action/Relation/Owner Operationが「受注1001から、この明細・この在庫予約への作用」を許したと説明できることが必要になる。

反対に、Function引数を1002にすり替えただけで、1001に関係しない顧客・在庫・秘密資料へ作用が広がる場合は、そのための独立した権限根拠が要る。**子Functionのservice identityが高権限であること自体は根拠にならない**。

## 3. 対象拘束の方式はAction/Providerの意味に応じて選ぶ

### 3.1 単一対象

受注1001の確定が受注1001だけを直接更新する場合、操作意図Iの対象と、Payload Pが指す対象と、Ownerが実際に更新する対象Eの対応を具体的なAdapter/Ownerで検査できる。`OperationID`の命名規則から対象を推測したり、汎用Runnerで文字列を解析したりする方式へ固定しない。

### 3.2 正当な関連対象

受注1001が明細A/Bを持ち、各明細から在庫を引き当てる場合、対象の対応は**受理されたRelationの役割と業務規則**によって決まる。対象が異なることだけを違反として扱わない。逆に、単に別のIDが同じデータベースに存在することを、関連対象である証拠にしない。

在庫提供元が別Ownerなら、#1129の`via capability`等の宣言候補を通じてOwnerの操作を呼ぶ。`via capability`の存在は承認・権限委任・同一Transactionを自動的に保証しない。

### 3.3 集合と動的条件

権限の対象が複数行の場合は、少なくとも次を区別する。

- **固定された対象集合**：承認時点でA/Bと特定され、当該Grantの条件として保持される。承認後の集合変化に追随するかは別のPolicy。
- **実行時に解決する対象条件**：指定部門が現在担当する注文など。対象解決時点、必要な組織関係・Predicate、完全性、取得範囲と現在認可を区別する。
- **関連する効果集合**：受注1件から複数の明細・在庫・通知へ広がる。Ownerが必要な作用の上限と確定範囲を説明できるかを見る。

集合の業務上の必須範囲`L_required`と実際に取得した`R_read`が異なるとき、部分取得を「対象全件の完了」と呼ばない（[#1130 P1-BC-1](https://github.com/hat47x/tei-platform/issues/1130#issuecomment-6557423540)）。ただし、業務Actionが独立した一行だけを更新する目的であれば、一般的な「全件不存在証明」を義務化する必要はない。**必要な対象集合の意味はOwnerが定める。**

実行時に集合条件を解決した後、同一Transaction内の後続Mutation、他Transactionによる更新、業務有効日の境界によって集合が変化し得る。どこまで固定し、どの時点のPredicateを確認するかは受理された業務規則とProviderの実保証による。事前Previewが成功しただけで確定時の集合・権限が不変だったとは主張しない。

## 4. 権限委任の意味を狭めすぎず、対象の拡大を防ぐ

権限評価の単純な「利用者権限∩サービス権限」だけでは、組織が正当に認めた代理業務を実現できないことがある。一方、サービス権限の全体を利用者へ渡すと、許可外対象への操作が成立し得る。

そのため、少なくとも次を別の条件として照合する。

1. **起動資格**：当該利用者がそのScript版・業務Actionを起動できるか。
2. **委任資格**：信頼済みIssuerの発行権限と組織Policyが、そのAction・対象・期限での代理実行を許しているか。
3. **子サービス資格**：実際の子Function/Providerが、現在のAuthorityで当該対象と操作を実行できるか。
4. **効果資格**：確定時の更新対象・作用・Owner制約・Effect Fenceの条件が満たされるか。

スクリプト内の「管理者として実行」という記述や、申請データ内の`approved=true`だけで2〜4を満たしたとしない。特権サービスが利用者を代理してよい**操作の範囲**は、サービスの全能力より小さくできる。

### 合議の権限合成

二人の承認者が承認した操作・対象・期間が異なる場合、対象ID集合を単純に結合してはいけない。例えば、承認者Aは「XとYに閲覧」、Bは「YとZに更新」なら、両方が関与したという事実だけで「X〜Zを更新可能」にしない。合議の結果をどのような操作・対象・期限へ写像するかは、業務Workflowの受理済みPolicyが定める。

### 上限と再委任

委任先が別のFunctionやAgentへ仕事を渡す場合、元Grant、Issuerによる追加許可、期限、対象条件、用途、再委任可能性を現在のHost/Providerで確認する。**誰が委任を必要と判断したか**と、**受け手のHostが行使を認めたか**も別の事実である。外部の製品・組織をまたぐ限定委任と成果物取得はEKI側の正本を利用し、TEI内のRun Planを自動的な外部権限保証とみなさない。

## 5. 改版・撤回と「再承認」の自由度

固定の承認フローをTEIが強制しないのと同様、すべての改版に一律の再承認を要求する仕組みにも固定しない。組織は、公開先・利用リスク・変更の種類に応じて任意のWorkflowを定義できる。

**改版時に比較する候補**は次のとおり。

| 変更面 | 変わり得るもの | 確認すべきこと |
| --- | --- | --- |
| ScriptのSource/呼出しグラフ | 子Functionや作用先が増える | 発行済みGrantの範囲で引き続き実行できるか |
| Provider/依存Plugin/実ロード物 | 同じScript文字列でも実処理が変わる | 実Load集合と、必要なConformance・現在利用資格 |
| Data Rule/組織Policy/関係 | 対象集合・評価時点・許可の意味 | 以前の承認条件を今も適用してよいか |
| 実行契機・対象指定 | 手動からイベント・定時・一括へ拡大 | 起動者、対象集合、期間、待機中の失効条件 |
| 結果の投影・開示 | 当時の結果から現在の表示へ変わる | 歴史的結果を復元できるか、誰に何を見せられるか |

組織は「特定の実行物の版まで固定する承認」も、「検証済みの作用・公開範囲を変えない更新については再承認を簡略化する制度」も設計できる。ただし後者には、実際の変更がその範囲内かを示す証拠が必要である。Script byte不変、CI PASS、Contract形状一致だけで、実効作用や現在権限まで不変と認定しない。判別できない変更は`unknown`としてWorkflow側へ返す。

権限取消は、今後の新規実行・まだ確定していない効果への現在資格を変える。取消前のCommitや履歴Receiptを消去したことにはならない。Queue/開始/確定、実行物の世代切替は別に観測する（#1130/#1174/#1223）。

## 6. 実行結果の回収・開示も委任対象の一部

`StatusReplayed`は、確定済みEffectを新たに実行しなかったことを示し得るが、「初回に返したBusinessResultを同じ内容で再生した」とは限らない。現在のEffect FenceのReceiptには、元の出力Projection版・本文そのものは含まれない（[#1174 RR-P1/P2](https://github.com/hat47x/tei-platform/issues/1174#issuecomment-6572369365)）。

組織向けマクロの結果表示では、少なくとも次を分離する。

- **Effectの歴史的記録**：何が確定したかと、その当時の実対象。
- **当時の結果本文**：初回Resultをそのまま保存／再構成できるか。根拠がなければ復元不能と説明。
- **現在の表示**：現在のデータ・説明規則・閲覧資格で再取得した情報。
- **外部への引渡し**：別製品へTask/Claimが渡ったことと、同一Attemptの実出力本文が受領・検証されたこと。

過去に実行した者の権限が、現在の閲覧者へ自動継承されることはない。Receiptの存在、対象ID、件数、診断文、Script Source、外部AIへの送出自体にも開示範囲を適用する。機密の実隔離・送出の検証は#2504／TAI #232／#2515が担当する。

EKI側では[#465](https://github.com/hat47x/eki-fabric/issues/465)に、実Runの出力を別Runnerから取得する静的な接続不備が記録されている。`Task completed`や`Claim ACK`を、Macroの成果物を元依頼者が取得できたという証拠にしない。異なる製品間の同一Attemptの出力接合は#1130の受入票へ委ねる。

## 7. 正負試験と責務を固定する

初期の共通対象は**合成受注1001と1002**。Script`M@d1`に対し、組織Workflowが受注1001への限定Actionを承認したと仮定する。この承認の業務意味も現時点では試験用の仮定である。

| 試験候補 | 正例／負例と期待する区別 | 実証Owner |
| --- | --- | --- |
| **T-1 単一対象の接合** | `I=1001/P=1001/E=1001`は許可候補、`I=1001/P=1002/E=1002`は明示追加委任がなければ拒否。`RequestHash/Receipt`だけでI=Eを推定しない | #1174 AT-1/P1-Payload-2（既存実PG負例を再利用） |
| **T-2 関連データの合法な拡大** | 受注1001→正式に関連する明細A/Bへの作用は契約次第で合法。無関係な明細C/受注1002への作用は拒否 | #1129、#1127、#2499 |
| **T-3 集合解決** | 承認時固定集合と実行時条件で、組織異動後の対象が異なる。必要な全件性が欠けた一括処理を完了扱いしない | #2386、#1708、#1130、#2499 |
| **T-4 特権子Function** | 子Serviceが持つ広い権限から、受注1001に限る委任は利用可能。引数だけ受注1002へ替えた呼出しはEffect拒否 | #2499、#1174 |
| **T-5 改版・依存変化** | 同じMacro名でScript/Provider/Rule/実ロード物だけ変更した場合、承認対象と実際の作用の差を可視化。自動再承認を行うかはWorkflow判断 | #2502、#943、#2499 |
| **T-6 予約中失効** | Queue時は有効、開始前またはCommit前にGrant失効。既確定と新規Effectを分け、現在資格が足りない新規作用を拒否 | #1130、#1174 |
| **T-7 履歴結果・開示** | 同じEffect Receiptで元Result再現可能な場合と、現行投影しか作れない場合を区別。非許可閲覧者へReceipt件数/Rawを開示しない | #1174 RR-P1/P2、#2516、#2504 |
| **T-8 外部成果物受領** | EKI Claim/Task completedでもBodyを取得できなければ利用者向け「納品済み」としない。Task/Attemptと検証済み実Bodyの同一性を確認 | #1130、EKI #465/#409 |

### 実行順

1. **既存P0維持**：#1174の実PostgreSQL確定後Result不正・独立Receipt観測、#1129の二Functionへの共通Data Rule適用を先行。新案をそのGateへ足さない。
2. **P1-Target**：#1174のT-1負例を実DBで実行したEvidenceを再利用し、#2499へマクロ委任条件の違いだけを加えたT-4正負の合成検証を追加する。実DB試験がSKIPならPASSとしない。
3. **P1-Group**：T-2/T-3を有限の独立Scenarioで比較。実Targetが必要な集合/時点/Current Authority保証を持たなければ`unsupported`を返す。
4. **P1-Lifecycle**：T-5/T-6/T-7をCUI・実Load・Current Auth・独立readへ接続。#2502の「作成→公開→利用」製品体験と結び付ける。
5. **P2-External**：T-8と、外部Java/.NET/Wasm/Process、AI Harness、機密HostをTarget別に比較する。外部への委任をすべてのScript実行の必須前提にしない。

### 検証票の列

`Issue/Scenario ID、固定Sourceとモデル版、Script承認対象・実Load物、Workflow Decisionと目的、Issuer/Grantの現在資格、Iの対象・粒度、Adapterに渡したP、Ownerが受理した変換規則、実Eの対象/作用/Transaction、独立Oの観測範囲・時点、R/Resultの履歴性と開示、実行Host/Provider、expected/observed、unknown・未実行、Owner判断`。

新しい共通JSON形式はこの票の実益を二つ以上の実Consumerで確かめてから判断する。既存#1180/#1131のScenarioと、#2514の独立Oracle・観測十分性を利用し、検査対象と期待値の正本を増やさない。

## 8. 不完全な投影からのマクロ実行：表示・操作候補・実行資格を分離する

### 8.1 なぜ「見えているデータが全て」でも「全部見せなければ実行不可」でもないか

利用者マクロは表・単票・グラフ・検索結果など、しばしば元データの**一部を投影した結果**から起動される。どの対象を更新するか、何を変更できるかは、画面に見える値と一致しない場合がある。

- [#2500 VE-P1〜P6](https://github.com/hat47x/tei-platform/issues/2500#issuecomment-6572158168)では、非表示のCostや欠落/nullの違いがあっても、実対象・変更Field・基準版・現在の権限を別に確認できれば「ラベルだけ更新」の限定PATCH候補は成立し得る。一方、部分Viewを元レコード全体だと思ってReplaceすることは別の資格が要る。
- [#2598 IB-01〜08](https://github.com/hat47x/tei-platform/issues/2598)では、Viewから生成した**read-onlyなIntent候補**に、当初の選択epoch、意味版、Query/Read Scope等の根拠が全部入るとは限らない。候補を保存・再利用するConsumerは、既存Evidenceを再解決するか、実Hostで必要条件を再検査しなければならない。
- [#2514 観測上の十分性](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6572161189)では、二つの元状態が同一の投影に写っても、業務上の結論が異なる場合、**その投影だけでは結論を決定できない**ことを一組の反例で示せる。

これらをマクロへ適用する。たとえば、表示Vが共通の二状態を考える。

~~~text
S1 = {id:L1, label:"A", privateCost:300, optionalTitle:missing}
S2 = {id:L1, label:"A", privateCost:900, optionalTitle:null}
P(S1) = P(S2) = {id:L1, label:"A"}
~~~

- **許容され得る操作**：明示的にL1のlabelだけを「B」に変更するAction。Macroが必要な対象とField・基準版をHostへ渡し、Ownerが現在の正本状態で業務制約・権限・競合を検証する。privateCostの値を利用者へ開示せず、必要なら内部制約に基づく拒否もできる。**候補として成立することと、実Commitが成功することを区別する**。
- **追加条件が必要な操作**：Vを完全なレコードとみなすReplaceや、Vに表示されないoptionalTitleを黙ってclearする操作。隠れた情報を元の投影だけから復元することはできない。必要な情報と全体置換の業務契約が無ければ、この用途についてはambiguous/unsupportedとする。
- **別の業務Actionなら可能な操作**：集計の棒「200→220」を元の明細への配分に写す操作。明細[100,100]と[50,150]は同じ集計表示200を作るが、どの明細をどう直すかは表示だけでは決まらない。ただし、Ownerが受理した「差額を特定ルールで配分」のような独立Actionがあれば、その入力・効果・対象集合の契約で評価できる。集計表示が非可逆なことを理由に、すべての合法な業務操作を一律禁止しない。

**設計上の要点**：必要なのは元State全体の再構成ではなく、**そのOperationが要求する対象、変更範囲、必要な判断情報、現在の権限・競合条件を、実際の実行境界で満たせること**。画面・マクロに秘密の原値を渡すことを必須条件にしない。

### 8.2 画面のIntentを、権限付きCommandと取り違えない

同じFunction Definitionが二画面に配置され、片方が親A、他方が親Bを表示する場合、同じrowIndexや表示ラベルは対象同一性の証拠ではない。親A→B→Aの選択変更、遅いQuery応答、未保存Patch、同じQuery ID・異なる実効RLS/検索経路が重なれば、見かけの同じIntentが別対象へ移る可能性がある。

- **候補の時点**：表示/選択したContext Instance、実体/Relation occurrence、親anchor、選択epoch、Queryの必要Read Scope、欠落・部分取得、Action版を確認し、変更候補にする（#2500/#2597）。
- **Bindingの時点**：Action/Service Contractと候補が同一の業務対象・操作を指し続けるかを、既存Evidence参照または現在のHostが再検証する。Intent単体にない情報を推定値で補完しない（#2598）。
- **確定の時点**：実RequestのPayload、Provider/Ownerが実際に変更する対象、現在のGrant/Transactionと必要な業務不変条件を結び、Receipt/独立観測で確認する（#1174/#1129）。Read時点の資格をCommitまで無条件に持ち越さない。

**正しい拒否だけでなく、過剰拒否も測る**。たとえば、D1明細の単一Field変更はD1の完全な読取りと対象識別があれば許容候補なのに、画面上に併置されたD2配送履歴が部分取得であるだけで一律に禁止するのは不必要な制限となり得る。逆に、D1×D2の全件集計を前提に更新するActionは、D2の対象集合の完全性を必要とする。Actionごとに依存するClaim/ScopeをOwnerが定める（#2586/#2514）。

同一画面から二つのContextのSaveを要求した場合も、**画面上の一つのボタンはDB上の一つのTransactionではない**。#2597の編集セッション/複合保存にある合成Python18件と限定SQLite3件（各担当の既報）は局所的な比較であり、TEI/Hostの実業務Tx・同一Command・一括成功を証明したものではない。新しいMacro Executorを作らず、同じBusiness ActionとProviderの原子性・部分成功・取消契約に接続する。

### 8.3 「権限がある」と「情報を見せてよい」は別

仮にWorkflowが同じ承認表示を返しても、内部のGrantが有効な状態S1と失効した状態S2では、新規Action実行の可否が異なる。**UIの承認表示、Macroの保存版、過去のReceiptだけから現在のGrantを復元しない**。Hostが現在の実権限と対象を評価する。

一方、ある利用者の限定業務Actionが正当に成立することは、その利用者が隠れたCostや他の所属情報、Receipt全文、関係する顧客全件を閲覧・AIへ送信してよいことを意味しない。要求の処理、内部の権限検査、実効果、外向きの結果投影にそれぞれ必要な開示条件を適用する（#2504/#2516/TAI #232）。

本書が提案するのは**有限の許可・拒否試験と、試験した出力経路についての非開示確認**である。画面のエラー文、件数、照会時間、ログ、別MCP Endpointまで含むあらゆる観測に対して数学的な完全非干渉を証明したとはしない。許可した外向きProjectionと実OS/Networkの送出結果は#2515/TAI #232の試験を再利用し、秘密の存在自体が非許可Consumerへ漏れる経路も負例で確認する。

### 8.4 改版後の資格は、依存している主張ごとに再確認する

#2582の局所的な再評価と[#2600の増分ビルド資格](https://github.com/hat47x/tei-platform/issues/2600)を、Script/Macroの**再承認手順そのものではなく、再確認が必要な根拠の列挙**に利用する。組織ごとの再承認Workflowは変わらず自由に構成できる。

| 変化 | 維持し得るもの | 独立に再確認するもの |
| --- | --- | --- |
| 同じScript実行物で表示文言のみ改訂 | 作用のSource/Build結果、過去のEffect記録 | 公開説明・開示・レビュー・適用先 |
| 同じScript Sourceだが子Service/Providerを交換 | Script Sourceの履歴、元Decisionの存在 | 実Load実装・子の効果/対象・現在Host資格 |
| 対象集合/組織関係・Actionの意味が変更 | 変更前の歴史的Grant/Receipt | LIVE集合の現在対象・Owner受理・実行資格 |
| 独立Oracle/Conformanceの資格が失効 | Source/Artifact byteの履歴 | 当該Claimの現在の検証根拠。古い「適合済み」を自動復活させない |
| マクロの表示・承認状態は同一、実Grantが失効 | 元の承認履歴、以前のCommit事実 | 新規Invocationと未確定Effectの現在許可 |

全変更に対する一律の全面再承認も、すべてを「互換だから自動許可」とすることも固定方針にしない。**候補となる再利用判断を技術側が提供し、組織の受理済みPolicyが実際の再審査を決める**。依存情報が欠ければunknownとし、根拠なしの自動成功・自動失効のどちらにも丸めない。

### 8.5 追加する独立受入（T-9〜T-13、すべて新規NOT RUN）

| 試験候補 | 観測したい正負と失敗させる誤推論 | 既存の所有者 |
| --- | --- | --- |
| **T-9 隠れたFieldと限定更新** | 表示が同じS1/S2で、label限定PATCH候補は成立し得る。部分ViewのReplace/暗黙clearを拒否。秘密の原値をMacroへ渡さず、Host内部で業務制約と更新結果を確認 | #2500 VE-P1/P2/P3、#2598 IB-03、#2504 |
| **T-10 集計表示と対象同定** | 同じSUM=200から異なる明細集合への自動UPDATEを推測しない。Owner定義の明示配分Actionなら、その具体対象・量・業務規則へ検証を渡す | #2593、#2500 VE-P5、#2514 |
| **T-11 古いIntentの再利用** | 同じ8欄の候補でもContext epoch/親anchor/必要Read Scopeが変化。再解決なしの実行資格昇格を拒否。ただし無関係なD2部分取得だけでD1単独PATCHまで過剰拒否しない | #2598 IB-01/04、#2582、#2597 |
| **T-12 承認表示と現在Grant** | 同じ承認表示で現在Grantが有効／失効の二状態。Hostによる現在認可・実操作結果は異なり得る。低権限Consumerへ秘密情報を付けた説明を返さない | #2499、#1174、#2504/#2515 |
| **T-13 複合保存と部分成功** | 同画面のD1/D2変更を、一体Txなら合成一体保存、別Providerなら部分成功/不明と説明。表示ボタン数=Tx数としない | #2597、#1130/#1174、#2502 |

**独立の受入条件**：各試験で、利用者から見える表示P、実体ID/Context、必要なClaimと入力閉包、許可されたAction、実Providerへの要求、必要なGrant、Ownerが実際に確定した範囲、現在の開示可能性を列として保持する。すべてのケースで実DB効果を強制するのではなく、read-only候補、Synthetic Provider、実DB/外部Hostを実際に確認した範囲で別資格とする。

**優先**：既存#1119のP0-a/P0-bを維持し、#2598のTarget Binding正負と#1174の実対象負例を先行して一つのP1へ接合する。T-9/T-11は既存View InterpretationのFixtureを再利用する。T-12のOS/egressとT-13の実Txは専門Ownerの検証を使用し、ここで第二の認可・Sandbox・Transaction Engineを実装しない。

## 9. Semantic Contextから構成する汎用ActionとScript-firstな標準UI

### 9.1 利用者Macroは既存Actionの組合せとして構成する

[#2507 Semantic Context起点の汎用Action](https://github.com/hat47x/tei-platform/issues/2507)と[#988 Script/DSL first](https://github.com/hat47x/tei-platform/issues/988#issuecomment-6573241174)を、#2499の限定委任へ接続する。汎用ActionはX-TEA由来の固定機能タイプや画面ボタンの種類に束縛せず、**意味主体・関係・Function Data Context・業務操作の適用条件**から発見する。SmartDBはそのような汎用Data処理の標準Plugin候補であり、非業務領域への適用も圧力ケースとする。

利用者Macroは、受理済みの業務Actionと標準PluginをScript/DSLで組み合わせ、現在のContextと手動・イベント・予約などの起動契機へ結び付ける**利用形態**として扱う。固有のMacro実行権限エンジンを独立させる必要性は現時点で示されていない。

- **Actionの適用条件**：#2506/#2507のSubject・Relation Role・粒度・取得範囲/完全性・状態・Type/Unit/時点・必要Service Contractを参照する。Actionが発見可能であることは実行許可を意味しない。
- **Script/DSL**：#988のSource/Parse/Resolve/Inspectを利用する。テキストを正本とするScriptと、構造化Sourceへ埋め込むScriptの双方を扱う。全種類を一つのMacro ASTやノード表現へ強制変換しない。
- **UI**：Studioの標準テキスト・プロパティ・表・ツリー・参照・依存グラフ・差分・PreviewをSourceに対する派生投影として使う（#1124/#1824）。生成アプリ側は#997のRenderer/UI Kitと標準Action Bindingで表示する。専用UIは既存の標準構成で満たせない具体的利用価値を実証してから選択できる。
- **実行権限と結果**：発行Grant（#2499）、現在Host/Provider AuthorityとOwner Effect（#1174/#1129）、実Load版（#943/#2390）、履歴と開示（#2504/#2516）は、UIの配置や表示規則とは別に確認する。

### 9.2 活性と表示は短い式で表せる

次は**説明のための疑似Scriptであり、正式なDSL構文ではない**。

~~~text
visible(discountField) = customer.type == "corporate"
enabled(submitAction) = order.valid && !interaction.busy
offer(recalculateAction) = context.grain == "aggregate" && measure.defined
~~~

表示や活性を決める式は、型・依存・Context Instanceと選択epoch・NULL/未取得・意味版・Rule変更に従って評価する。`enabled=true`でも実行Grantが失効していればEffectは拒否される。`enabled=false`はUI側の状態であり、業務Serviceの現在認可や外部Consumerの実行資格を直接書き換えるものではない。Actionの候補提示・起動・認可・確定・開示を一つの真偽値へ丸めない。

### 9.3 標準データ構造とSource編集の往復

新DSL/Operator Pluginがsymbol/source span/semantic ID、入力依存、値域・Context、Effectの既知/未知、diagnosticを共通inspectへ渡せる場合、Studioの標準UIから参照・検索・説明できるか検証する。未知DSLでもtext/既知metadata/unknown diagnosticを表示し、意味や副作用を捏造しない。

Source span/意味ID/版への逆写像が一意な標準UI編集は、型付き変更提案→Preview→既存Acceptance→再Parse/再投影へ接続する。Macro展開後の派生ノード、複数Source span、古い版の同形ノードではread-onlyと原SourceへのNavigateを提供する。ノード位置やフォームの表示状態をScriptの第二正本にしない。

これにより、新たな意味PluginやDSLを追加するたびにStudio専用画面を改造する依存を抑えられる。一方、任意の未知言語を完全解析できること、実Targetに安全に実行できることまでを無条件には保証しない。

### 9.4 提示と実行の資格を局面ごとに評価する

| 局面 | 必要な証拠 | 成立すること |
| --- | --- | --- |
| Action発見 | Semantic Context/Required Data Shape、Patternの意味条件 | このContextで利用候補となる |
| 提示 | Consumerへの開示範囲、UI Placement、Contextの現在状態 | 利用者へ何を表示できるか |
| 起動要求 | typed Intent、Subject/Relation Role、現在epoch、必要なSource Evidence | 誰が何をどの対象に要求したか |
| 権限付き実行 | 現在のIssuer/Grant/Policy、Host、実Load Provider、実Payload対象 | どこまでの作用を許可するか |
| Effect/説明 | Owner実Commit/Receipt・独立Read・Result版と閲覧権限 | 実際の効果と表示可能な結果 |

表示が同じでも実効権限が違う場合、逆にUIで見えないFieldがあっても限定Actionを正当に実行できる場合を、両方試験する。SmartDBや組織用Action、専門領域Pluginにも同じ契約を適用できるかを比較する。

### 9.5 新しい横断試験（T-14〜T-18、全て未実行）

| ケース | 正負例・観測 | 既存Owner |
| --- | --- | --- |
| **T-14 同一Action複数Consumer** | 同じL1限定PATCHをCUI/標準表UI/利用者Macroに結び付け、表示位置は違っても対象・業務Action・Owner効果が同一に保たれるか | #2506/#2507、#997/#2502、#2598/#1174 |
| **T-15 活性表示と実認可** | UIで`enabled=true`でもGrant失効時に実Effect拒否。UIで`enabled=false`が別Hostからの適法な起動を無条件に消さない | #988/#1124、#2499/#1174 |
| **T-16 新DSL Pluginの標準UI投影** | 新しい構文を追加しても標準inspect/Tree/Property/Diffが機能し、未知Effectはunknownと表示する | #988/#1124/#1824、#2502 |
| **T-17 Source編集の往復** | 一意Sourceへの編集はChange Proposalへ、展開/曖昧ノードはread-onlyとNavigationへ。別のScript正本を作らない | #988/#1124/#267、#2502 |
| **T-18 起動契機と現在のContext** | 同じActionを手動/イベント/予約で利用し、旧epoch・LIVE対象集合変更・実Load版差・失効Grantを再照合する | #2502/#1130、#2582/#2499/#943/#1174 |

**統合順**：先行する#1174実PG P0-aと#1129二Function P0-bは維持。P1では#2507のAction適用条件と#988のScript Inspectを#2502のHeadless CUIに接合し、T-14とT-15を既存#2598/#1174の対象整合試験へ委譲する。Studio標準UIとSource編集の実適合は後段のT-16/T-17で検証する。新しいMacro Executor、第二Authority、画面種別別専用DSLを始める契機にはしない。

## 10. 標準データ構造としてのマクロと明示的なデータ利用契約

### 10.1 データとしての定義と実行能力を分ける

[Everything is Data #2619](https://github.com/hat47x/tei-platform/issues/2619)、[意味役割・自己記述 #2620](https://github.com/hat47x/tei-platform/issues/2620)、[Markdown Authoring #2621](https://github.com/hat47x/tei-platform/issues/2621)、[Data Operations #2622](https://github.com/hat47x/tei-platform/issues/2622)を横断して、利用者マクロを**特別な実行オブジェクトではなく、標準データ構造へ意味役割を適用した処理の組合せ・起動条件・共有条件**として構成できるか検証する。

- **構造**：Record、集合、関係、参照、式、演算子適用、入力・出力・テストなどをデータとして保持。`Function`／`Validation`／`Scenario`／`Macro`といった役割やOperationの意味はPluginが解釈する。複数Roleの適用も候補とし、競合・循環・Plugin版を検査する。
- **表現**：Markdown表、短い式、簡潔なDSL、既存プログラムのテキストは、それぞれAuthoringの表現。標準データ構造へ解決できた範囲と、原文のSource span・コメント・記法固有情報を区別する。すべての既存言語の完全解析や厳密なLisp式の採用は要求しない。
- **適用**：モデルの意味受理、Macroの公開承認、個人保存、実ロード物、現在Grant、Hostの実能力とOwner Effectは、それぞれ別の資格。`Macro`というRoleの付与やテストPASSだけで本番実行を許可しない。
- **UI**：Studio標準のテキスト・表・Tree・Graph・Property・Diffと、AI向けCUIを同じ構造・意味への投影とする。派生した表示を第二の正本へ昇格させず、明示的なSourceと既存Acceptanceへ戻す。

この方針は開発者向けの強力なScriptを廃止するものではない。Native Java/SQL/独自Script等の中身が不透明なら、解釈できた型・入力・出力・効果・実行Bindingだけを標準データとして説明し、未把握の作用は`unknown/opaque`とする。利用者Macroの限定能力はHostが実際に強制する。

### 10.2 構文糖衣と意味同値を区別し、参照の捕捉を検証する

[DSL/Operator #988](https://github.com/hat47x/tei-platform/issues/988)の意味契約に従い、たとえばMarkdown表の「金額 = 数量 × 単価」と別の式・DSL表現が、**同じ型付き参照・Operator ID/版・丸め/NULL/単位/時点の条件**へ解決された場合に限り、当該契約内の同等性を説明する。

構文糖衣の導入時は次の判定を独立に残す。

| 資格 | 判定の対象 |
| --- | --- |
| Source同一性 | 元テキストbyte、Source span、Markdown行列、コメント等 |
| 構造同一性 | 役割適用前の標準データ構造・関係・順序 |
| 意味同等性 | Role/Operator版、参照Binding、必要Data Context、Law、Owner受理 |
| Target適合 | 実Provider/Host・実Load・必要な能力とEffect |
| 現在資格 | 現在Grant、業務判断・公開承認、情報開示 |

Graphviz、関数記法、メソッドチェーン、Markdown表が内部構造の記述を簡潔にしても、文字列や同名変数だけで異なるSubject・Provider・Principalを自動結合しない。**マクロ宣言時の参照、使用時のContext Instance、実行主体のGrant**を分け、マクロ展開による意図しない参照・Privilegeの捕捉を負例にする。これはLisp固有のマクロ評価方式を採用するという意味ではない。

標準UIから元Sourceへの逆写像が一意ならChange Proposal→既存Acceptanceへ。展開された複数Source候補、Source移動、曖昧な同名参照はread-onlyと元SourceへのNavigateを提供する（#1124、[Draft #2617の限定合成試験](https://github.com/hat47x/tei-platform/issues/2617)）。Syntax変換のPASSを意味受理・実行許可へ昇格させない。

### 10.3 Read・Authority・Effect・Disclosureの四範囲は独立する

Data ModelにRelationが存在することは、その先の全データをMacroが暗黙取得できる根拠ではない。[#2506 Function Data Context](https://github.com/hat47x/tei-platform/issues/2506)、#2619/#2621/#2622に従い、**起点Subject、Relation path、Field、grain、対象集合、取得条件、必要な完全性、取得時点**をFunction Data Contextの明示Mappingへ置く。Studio/AIによる補完は候補とし、受理後のMappingを追跡する。

| 範囲 | 定めるもの | 誤った推論の例 |
| --- | --- | --- |
| Read scope | Contextが読むEntity・Relation・Field・集合・時点 | Relationがあるから全部取得する |
| Authority scope | 現在Host/Issuerが許すContract・Action・対象・Tenant/期限 | Macro公開で子Serviceの全権限が使える |
| Effect footprint | Owner Actionが実際に変更する対象・副作用・Tx境界 | 実効果が画面行IDと同じ／別IDなら全部違法 |
| Disclosure scope | 現在利用者・Agentへ開示できる値・ID・件数・履歴 | 正当に更新できればRawやReceiptも全開示 |

四つの範囲は**同じ集合でなくてよい**。狭いRead scopeからOwnerの正当な関連効果へ作用する場合があり、Host内部で広い検証が必要でもConsumerの開示範囲を限定できる。そのためには、Ownerが受理した対象変換と現在Grant・Effect Fenceが必要である。

同じMacroを別部門・別Context Instance・別Hostで起動するとき、**作成者のRowIndex/選択Context・Grant・Provider資格を実行時に暗黙継承しない**。FROZEN対象とLIVE述語、tRead/tBind/tCommitでの必要な再検査、部分取得と全件確認をActionごとに扱う（#2499/#2598/#1174）。

### 10.4 多階層データを圧縮表示した場合のLoad/Save

受注→明細→割当の**三階層を、受注→倉庫別割当合計の二階層に表示**する例を考える。次は読みやすい設計例を示すMarkdownで、採用済みTEI構文ではない。

~~~markdown
## 利用データ
| Context | 起点 | Relation | Fields | 範囲 |
| --- | --- | --- | --- | --- |
| lines | 選択受注 | order.lines | lineId, quantity | 全明細 |
| allocations | lines | line.allocations | allocationId, warehouse, qty | 対象明細の全割当 |

## 表示と操作
| 表示 | ロード時 | 更新時 |
| --- | --- | --- |
| 倉庫別割当 | allocationsの倉庫別SUM | 明示`allocation.change`と対象ID/配分規則を確認 |

## 独立Expected（説明用）
| case | 元の割当 | 操作 | Expected |
| --- | --- | --- | --- |
| no-change | 東京3／大阪2 | 変更なし | 更新0 |
| exact-patch | a=東京3、b=大阪2 | a=4、b=1に明示変更 | 2件のID付きPATCH候補 |
| ambiguous | 東京合計5（元割当複数） | 合計6へ変更 | 配分Actionがなければ対象不確定 |
~~~

取得・集約・変換はRDB、OData、REST、Graph、File等のData Operationsへ接続できるかを評価する。特殊なSQL/native assetは明示Bindingとして保持する（#2622）。Read成功とSaveの可逆性・原子性は別であり、非可逆な合計から元の行を推測して更新しない。業務Ownerが受理した配分Actionがあれば、現在Grant、元集合の閉包/粒度、実更新対象、部分成功とReceiptをProviderが検証する（#2500/#2597/#1174）。

### 10.5 定義・具体例・独立テストをデータとして接続する

Markdownの説明、Function/Rule定義、Case表、Dataset、Expected、実Actualを共通構造に解決し、Model Instance・CUI検査・SHO説明へ投影できるか評価する。ただし**すべてのMarkdown表を実行定義として解釈するわけではない**。明示Role・Project Profile・元Sourceと独立Oracleが必要（#2621/#1131/#1180/#2514）。

テストのClaimは、Source/Meaning、Input/必要集合、独立Expected、実Target/Hostで投入したPayload、実Effect/Observation、現在の開示資格に分ける。単にScriptテキストが同じ、JSON byteが同じ、CIがPASSしたという理由で業務意味・実Effectの受理を確定させない。標準データ化の効果は、異なるConsumerやDomainでの**専用実装削減・再利用・不足を説明できるか**として比較する。

### 10.6 新たな検証候補（T-19〜T-26、全て新規NOT RUN）

| ケース | 正負例・観測 | 既存Owner |
| --- | --- | --- |
| **T-19 意味役割** | 同一構造からFunction/Test等の明示Roleを解決。Role不明/競合/自己再帰で無根拠な一意解釈をしない | #2620/#988 |
| **T-20 構文糖衣** | 表・式・DSLが同じOperator/型/参照版なら局所同等。Source編集の一意逆写像と曖昧な場合を対照 | #2620/#2621、#2617 |
| **T-21 明示Context** | `order.lines` Relationがあっても未宣言ならMacroが暗黙全明細取得しない。明示Mapping後だけ指定Field/範囲を読める | #2506/#2622 |
| **T-22 再利用時の束縛** | 同じMacroを部門A/Bと別Context Instanceで使用。作成時RowIndex/Grantを捕捉せず現在対象/権限を照合 | #2499/#2598/#2502 |
| **T-23 四範囲の差** | 限定Read→合法な関連更新→最小開示。許可外の子更新/Raw開示を拒否 | #1174/#2499/#2504 |
| **T-24 多階層Load/Save** | 3層→2層集約、無変更保存0、明示ID付きPATCHは候補、合計だけの編集は配分規則が必要 | #2622/#2500/#2597 |
| **T-25 多Provider** | RDB系とAPI系に同じ意味Intentを適用し、NULL/集合・更新/atomicityの差を示す。Native SQLも明示接続 | #2622/#951/#1121 |
| **T-26 テストもデータ** | Markdown説明表と採用済みFixture/独立Expected/実Actualを分け、AIの自己生成結果を自己合格させない | #2621/#2514/#1180 |

**実証順序**：既存P0-a/P0-b維持。S0のread-only設計・Script InspectにT-19〜21/26、S1のIntent対象照合にT-22/24、S2の現在Grant/Owner EffectにT-23、S4の実Target差にT-25を合流させる。新しいCore schema、Macro実行Engine、共通Host、DSL固定構文、第二のMeaning/Authority正本をこの文書から採用しない。範囲の広い探索は#2493と#2619で並列継続する。

## 11. 実際の読取りとProvider経路を、マクロの権限委任・業務結果へ接続する

### 11.1 同じ出力でも、実際に読んだ情報は同じとは限らない

[明示Data Mappingの局所Reference Draft #2624](https://github.com/hat47x/tei-platform/pull/2624)では、未宣言Field/Relationを変更しても**合成出力が変わらない**という有限試験がある。一方、[実アクセスの比較 Draft #2628](https://github.com/hat47x/tei-platform/pull/2628)では、**出力が完全に一致しても、内部で秘密Fieldや無関係Relationを余分に読んだProviderと、狭い対象だけを読んだProviderは別である**ことを12件の合成試験で比較した（隔離Python unittest 12/12 PASSという既存Issue報告）。この二つは互いを置き換える証拠ではない。

利用者マクロで`GetOrderLabel(1001)`という同じ結果を得ても、内部Providerがどの項目を読んだかは別問題になる。

~~~text
宣言したData Context:       order.id, order.label
Narrow Provider呼出し:      order.id, order.label を取得
Overfetch Provider呼出し:   order.id, order.label, order.secretNote を取得
外向きMacro Result:         { id:1001, label:"A" } （どちらも一致）
~~~

**結果値・表示の同値性から、内部で行われたデータアクセスの最小性やOS隔離を推論できない。** ただし、JOINキー、Row-level security、現在の権限評価、業務不変条件に必要な内部読取りなど、表示対象外の情報を正当に参照する場合もある。実Providerが実際に読んだ項目と、利用者へ開示した項目を単純に等しくする一律制約ではなく、Owner/Hostが認める内部目的・データ範囲と照合する。

最低限、以下の五つを受入票の別列として保持する。新しいCore型や共通Trace Engineの追加は、この段階の前提ではない。

| 証拠の段階 | 内容 | 混同する危険 |
| --- | --- | --- |
| **Declared access** | Function Data Contextが明示したSubject/Relation/Field/Scope/時点と必要な完全性 | 宣言があるだけで実取得済みとする |
| **Realized request** | 選択されたProvider/Endpoint/Mappingに実際に発行したQuery/API要求・実効Principal | 読取成功なら望んだ範囲だけを読んだとする |
| **Executed read witness** | 実Provider/DB/OS等が観測可能な範囲で示す読取り・利用・拒否の証拠 | Adapterのメソッドログだけで全DB内部I/Oを証明する |
| **Result/Claim** | Consumerへ渡るResult Sliceとgrain/Scope/時点・completeness | 出力同一なら読取経路/全件性も同じとする |
| **Disclosure/egress** | 実際のTool/Host/AI/利用者へ開示されたField・件数・ログ・転送先 | Scriptが返さなければSecretが他経路にも漏れないとする |

実Read witnessが得られないProviderでは`unknown`を保持する。そのようなProviderを一律に全業務で禁止するのではなく、求める保証水準を明示してTarget/業務Ownerが判断する。**「Provider内部で余分な情報を読んでいない」という保証を、その証拠なしにPASSにすることはできない**。実Host/OS/Networkに関する拒否証拠は#2504/#2515/TAI #232が担当し、ScriptやUIの宣言で代用しない。

### 11.2 同じSemantic Dataの複数経路は、選択した実Targetまで別に確認する

[複数Persistence/Alias #2611](https://github.com/hat47x/tei-platform/issues/2611)と[独立Go Reference Draft #2627](https://github.com/hat47x/tei-platform/pull/2627)で、同じ`sales.order`への複数経路（Primary/Replica/Archive/OData/別Mapping）の問題を具体化した。既存報告ではGo1.23.2の合成Reference 12/12、race、vetがPASS。ただし**実TEIの`DeriveOperationSurface`、既存World Validator、実Providerの受入試験ではない**。

マクロのAction・Read側で区別すべきものは次のとおり。

- **業務上のSubject同一性**：同じ`sales.order`が参照されても、同じレコード集合・時点・書込みOwnerの保証とは限らない。
- **経路の同一性**：Provider、Endpoint、Persistence Mapping/Revision、Tenant/Organization Scope、Purpose（Current/History/Replicaなど）、実行資格が同じか。正確に同じ物理経路を指す複数Aliasは統合候補にできるが、元Alias・用途と拒否理由を追跡する。
- **候補の可用性**：Archiveが読めないことと、Primaryからは読めることは両立する。一つのAliasのGapをApplication全体の不可にせず、別Aliasの存在でそのGapを消去しない。
- **実際の選択**：候補集合を並べられることと、どの経路でデータを取得し、どの経路へ書き込むかを受理したことを分離する。Aliasの名前・列挙順だけで実選択が変わる状態を、静かな成功として扱わない。

例えば、`archive`が受注1001の歴史状態を返し、`primary`が現在の受注1001を更新する場合、同じ`operationID`や受注IDの文字列から、Historyの読取りをCurrent writeの前提へ自動転用しない。Business ActionとData Contextが要求するPurpose・時点・対象集合・権限を満たす経路が選べるかを別々に評価する。

[PR #2587の候補コードに関する#2611の静的所見](https://github.com/hat47x/tei-platform/issues/2611)は、`operationID`の重複を内容比較なく消す可能性を指摘する。これを製品で再現したという事実にはせず、当該PRとWorld Validatorの**exact版**を固定した実試験へ渡す。#2602のMacro設計は正式なOperation ID形式や新Provider Resolverを決定しない。

### 11.3 Replicaで読んでPrimaryへ書く場合：対象同一性と鮮度を分離する

「Replicaから取得した受注1001を表示し、利用者マクロがPrimaryで受注を確定する」処理を考える。ReplicaとPrimaryが同じ意味IDを扱っていても、Replicaで観測した版、所属Relation、未確定の明細集合、現在Grantの適用時点がPrimaryのCommit時点と一致するとは限らない。

- **候補作成**：Replicaのreadは、取得したSnapshotと有効な範囲で候補・説明を提供し得る。`complete`の宣言と実取得完全性は別。
- **作用準備**：Owner Actionが必要な基準値・対象集合・資源版・組織関係を明示する。Replicaの旧StateをCurrent Truthとして無条件に流用しない。
- **確定**：Primary/Ownerの実Providerが、その業務操作に必要な現在Authority・資源版・作用対象をCommitの条件へ結び付けられるかを試す。別DBの全読取りと現在の書込みを原子的にできない場合は、その制約を`unsupported/unknown`として説明し、暫定結果を「全件確認済み」としない。
- **回復**：Commit結果が不明なら#1174のOperation/Receipt/独立Readと#1130の再開契約を使う。別Replicaの同じ表示を、追加writeなしの保証や初回BusinessResultの忠実再現とはしない。

データの鮮度を要求しない業務ActionにはReplicaを合法に使う余地がある。したがって「Replicaだから全Actionを禁止」ではなく、**Actionが要求する時点・完全性・Owner不変条件と、実経路が提供できる保証の適合**で判定する。

### 11.4 Failover・Pushdownの「意味を保った実現」を独立に検証する

同じMacroを実行するとき、Primary→Replica、RDB→OData、現行DB→ArchiveなどへFallbackする場合、Providerが同じ型のJSONを返しても、対象集合・最新性・認可・操作種別・部分失敗の保証が異なる可能性がある。

Fallbackは**当該Operationの意味・Owner目的と、現在のHost/Providerが許した経路**に従う。読取りが継続できる経路があっても、書込みの安全性や正しい結果を同じ範囲で保証できるとは限らない。別Providerへの再試行では、前回送信が未確定の場合の重複作用を考慮し、#1174のReceipt/Recoveryを優先する。

同様に、標準Data OperationsのSQL/APIへの`pushdown`とローカル実行では、演算子名が同じでもBag/Set/Sequence、JOIN多重度、Truth/NULL/欠損、FilterとLimitの順序、Group/粒度、SUM/AVG・丸め、時点が異なり得る。[#2387 QX-23のSQL対手書きData Operator限定対照](https://github.com/hat47x/tei-platform/issues/2387)（独立SQLite/Python9件の既報）は意味保存の反例として利用し、Macro→実Provider/他Targetの同値成立へは自動昇格させない。Native SQL資産は明示Bindingとして保持する（#2622/#988）。

### 11.5 追加する既存Reference接続試験（T-27〜T-34、全てMacro側はNOT RUN）

| ケース | 正負対と必要な証拠 | 既存Ownerと参照実験 |
| --- | --- | --- |
| **T-27 同じ結果・異なる実読取り** | 宣言された`id/label`と出力JSONが同じでも、隠れた`secretNote`を余分に読むProviderを判別。Outputの同値から最小性PASSを導かない | #2622/#2506、Draft #2628 RW |
| **T-28 合法な補助Readと過剰Read** | JOINキー/認可条件の内部Readは目的と許可範囲に応じて合法候補。無関係Payment/Secret読取りを、結果へ表示しなかっただけで許可扱いしない | #2622/#2504、#2628 |
| **T-29 Trace不完全と開示** | Adapter Read callのみ観測できるTargetと、実DB/OS拒否まで観測できるTargetを区別。前者の「物理Overfetchなし」資格はunknown、Consumerへの漏出は別に検査 | #2622/#2515/#2516、#2628 |
| **T-30 複数AliasのOperation経路** | 同じSemantic DataをArchive/Primaryで異Mapping参照。辞書順・Alias改名・片経路Gapの違いで実選択が黙って変わらない。正確に同じ物理経路への複数Aliasは統合候補 | #2611 PM-01〜07、Draft #2627 |
| **T-31 Replica→Primaryと古い前提** | Replicaの`order1001=version1`で操作候補を作り、その後Primaryがversion2に変更。現在のOwnerの要求条件を再検査し、旧Snapshotだけで更新完了を主張しない | #2611/#2598/#1174、#2627 |
| **T-32 別経路へのFallbackと委任** | 許可済みCurrent read経路の停止後、別Tenant/HistoryのAPIが利用可能でも自動対象拡大しない。許された同等経路があれば限定継続。未確定writeは盲目的に再送しない | #2611/#2499/#1130/#1174 |
| **T-33 Data Operator実現差** | SQL/nativeとData Operationsの同形結果だけで、JOIN bag/NULL/Filter-Limit/Group/時点の意味保存を認定しない。独立OracleとTarget別適合を照合 | #2387 QX-23/#2622/#2514 |
| **T-34 Outputと実アクセスと外向きEgress** | Macro結果の狭いProjection、内部Read、UI/log/Tool/Sessionの出力経路を三者で対照。閲覧拒否/送出拒否を実Host/OS側で確認 | #2504/#2515/TAI #232、#2628 |

**初期の接続順**：S0にT-27/29のread-only受渡し票、S1にT-30/31のProvider選択とIntent対象資格、S2にT-28/32の現在Grant・実Owner作用・回復、S3にT-34の用途別説明・開示、S4にT-33の別Provider/Target意味保存を割り当てる。既存P0-a/P0-bを維持する。

**既報と新規の区分**：#2627のGo12件、#2628のPython12件、#2387 QX-23の9件は**それぞれ異なる局所Referenceの既報**であり、今回新たに走らせていない。T-27〜34のMacro→Data Context→実Provider→Grant→Effect→Disclosureを通じた製品実験は全てNOT RUN。合成結果を実Providerの全I/O証拠や、OSレベルの秘密保護・権限保証に昇格させない。新しいProvider Resolver、共通Host、実行権限の第二正本、全情報流Checkerは設けない。

## 12. 非出力依存と選択的再資格化：同じ結果・同じSourceでも、現在の利用資格は別

### 12.1 Resultの値に出ない依存を、Macroの判断と変更影響へ接続する

[Query Explain #2387](https://github.com/hat47x/tei-platform/issues/2387)のSQLite読取り比較と[実Go DerivedIRの局所特性Draft #2634](https://github.com/hat47x/tei-platform/pull/2634)（Goテスト保存、実行はIssue記録に従って別確認）、[#2506](https://github.com/hat47x/tei-platform/issues/2506)/[Function依存 #2582](https://github.com/hat47x/tei-platform/issues/2582)を、Script/Macroの公開・実行・結果・更新判断へ結ぶ。

例えば、Macroが「承認対象の注文ID」を返すQueryを使う場合：

~~~sql
SELECT order_id
FROM orders
WHERE secret_score > 50
ORDER BY secret_rank DESC
LIMIT 1
~~~

`order_id`だけが出力Fieldでも、`secret_score`は**行が結果に含まれる条件**、`secret_rank`は**どの行が選ばれるか**を左右する。`secret_score > 50`から`> 70`へ変更しても、固定Fixtureでは同じ`order_id`が返り得る。一方、別の値・時点・対象集合では違う結果になる。このため、出力byte・OutputField.Inputs・Query IDだけでは、Macroが依存する行集合や将来の選択条件の不変性は証明できない。

主張を少なくとも次の観点で区別する（以下は設計用分類であり、新Core enumではない）。

| 依存の種類 | 意味すること | 変更時の扱い |
| --- | --- | --- |
| **Value lineage** | 表示値を計算する直接入力・導出Rule | 当該値・Ruleの再評価 |
| **Membership/predicate** | 行が存在・採用・拒否される条件。WHERE/EXISTS/反結合等 | 結果集合・Action対象・必要な全件性を再評価 |
| **Ordering/selection** | SORT/TOP-N/LIMIT、選択epoch、UI occurrence | 選択した業務対象を再対応付け |
| **Aggregation/grain** | JOIN多重度、Group/Measure、単位・丸め | 合計等の意味・処理順序・独立Oracleを再評価 |
| **Data source realization** | Provider/Endpoint/Mapping/Storage/Role/Read witness | 実経路、入力型・鮮度・Authority・取得範囲を再照合 |
| **Business/Effect qualification** | MacroのAction、Decision/Grant、実ロードScript、Owner Effect | 今後の実行/Commit資格を現在の条件で確認 |
| **Consumer disclosure** | 表示・SHO・ログ・AI送出への公開範囲 | 現在の閲覧者・用途の開示資格を再評価 |

SQLの現行Go DerivedIRで`OutputField.Inputs`に現れないFilter/Sortの生Expressionを、Consumerが閲覧してよいと解釈しない。依存の存在・Role・Sensitivityや検査結果を必要な粒度で示し、具体Predicateや値は#2516/#2504の開示条件に従う。`Lowering=full`等の解析Coverage表示も、入力Analyzerの参照が実際に十分だった証明と分ける。未知の依存を「影響なし」と確定するのではなく`not-established`として次段へ渡す。

### 12.2 歴史的に成功したEvidenceと、現在のClaim資格を区別する

[#2600 増分再利用](https://github.com/hat47x/tei-platform/issues/2600)、[実験手順Draft #2636](https://github.com/hat47x/tei-platform/pull/2636)、[#2582のClaim単位の変化](https://github.com/hat47x/tei-platform/issues/2582)、[ArtifactSet用途別網羅 Draft #2630](https://github.com/hat47x/tei-platform/pull/2630)を接続する。**過去の試験結果・Receiptは履歴事実として保存できるが、現在のMacro実行許可・業務結果・利用者への開示に再利用できるかは別のClaim**である。

例えば、受注確定Macro`M@v1`の出力を昨日受理したとしても、今日は以下のいずれかが変化している可能性がある。

- Script自体、呼出し先Function、Rule/Query predicate、Operatorの意味・版。
- Function Data Contextの関係・対象集合・評価時点、Provider経路・Mapping・Storage、実際に読み出した内容・完全性。
- 生成Library、依存classやPlugin、Builder/Toolchain、実ロードしているArtifact一式。
- 組織Policy、Issuer/Grant、主体と委任対象、Owner Actionが持つ実Effect契約。
- 出力説明・SHOのSource参照、閲覧者/AI Harnessの開示条件、Tool/OS/Networkの到達経路。

`same Source bytes`／`same Operation ID`／`same Output JSON`／`same Artifact digest`の**それぞれから他段階の同等性を推定しない**。ただし、無関係のSHO説明文や、D1のみ依存する値へのD2の局所更新を理由に、すべてのMacro・全Artifact・全Reviewの無条件再実行を要求する必要もない。

変更された入力と**既に判明している実依存閉包**、受理済みMeaning/Owner不変条件、現在Consumer目的から、再確認を要するClaimを選ぶ。依存根拠が不足している部分は局所`unknown`として保持し、必要な再読取・再評価・Owner判断へ渡す。意味受理・組織の再承認Workflowの発火条件は各Ownerが定義する。単にモデルの1文字が変わるたびに中央の固定承認Engineを起動する設計ではない。

### 12.3 MacroのArtifact一式：生成・公開・実読取りの資格を別に持つ

Script AuthoringのSourceと、正式に公開したMacro/Library/依存Plugin、外部Hostで**実際にロードしたbyte集合**を別に追う。[Manifest/内部整合 #2482](https://github.com/hat47x/tei-platform/issues/2482)、[物理公開の版一貫性 #2604](https://github.com/hat47x/tei-platform/issues/2604)、[用途別ArtifactSetの期待member #2630](https://github.com/hat47x/tei-platform/pull/2630)、[実Artifact Go特性Draft #2633](https://github.com/hat47x/tei-platform/pull/2633)を本製品のDelivery/Invocationの受入先とする。

複数ファイルのMacro配布で`manifest.json`が内部整合していても、Consumerの対象Applicationに必要なScript本体・子Library・契約・Resource全件が含まれる保証や、ConsumerがManifestに記載された実byteを一式同世代で読んだ保証とは異なる。公開途中のR1/R2混合、旧Resource残留、同じNameに正規化される別Subject、別ApplicationのMember混入、Readerが解釈できない形式を、正式配布済みと誤認しない。

**公開の操作と、Macroの本番実行資格の間には、少なくとも**以下を別に確認する必要がある。

1. **Expected set**：このConsumer/Action/Hostが必要とする全Memberと版・役割・意味IDを、対象Scopeから決めたか。
2. **Produced set**：生成した各MemberとManifestのbyte/Role/Source・依存の整合。
3. **Published generation**：実際に公開された場所・世代・途中失敗・残存物の扱い。
4. **Read/Loaded set**：Macro利用者/外部Hostが本当に読んでロードしたbyteとMember閉包。検証前に推測しない。
5. **Current qualification**：組織が受理した版/Effect契約・現在Grant・目的/Target・Consumer別公開資格。

ただし、この五段階を一つの中央配布Store/Registryへ実装する必要はない。外部Host・Package Manager・Versioned directory・Manifest Reader等の実現を許し、#2604/#2630が実証する必要範囲だけをContractへ還元する。SHO文書の公開とMacroの実コード配布は必要Member/開示先が異なるため、同じ`Manifest.Complete`真偽値で両方の納品・実行適格性を宣言しない。

### 12.4 選択的再資格化と実行再試行は別の操作

評価対象を**純粋な再評価／再ビルド／過去Evidenceの照合／現行Authorization照合／実Owner Effect**に分解する。履歴の計算値やScript byteを再利用できても、失効Grantで副作用を再実行することはできない。逆に、Owner Commit済みでReceiptがある場合に、再証明のために業務Actionを再Submitすることも適切とは限らない。Effect未確定時は#1174/#1130のReceipt照会・独立Observation・業務Ownerの回復判断を利用する。

選択的再資格化は、毎回過去のEvidenceと**今回の対象・Scope/時点・Consumer目的・現行Grant**との一致を検査する。評価の状態は「以前の試行で成立した」「現在のClaimへ再利用可能」「再確認が必要」「証拠不足」「このTargetでは非対応」などを説明票として区別するが、正式なCore enumやUniversal Claim Engineはこの段階で採用しない。

[実験プロトコル #2636](https://github.com/hat47x/tei-platform/pull/2636)の`F0`（基準全実行）・`F1`（変更後全実行）・`S1`（選択的再実行）・`S2`（負例）と、`J0/J1/J2`（再確認方針の費用対照）をMacroの同一Fixtureへ適用する。**正しい再利用と、不要な再検査を減らすことを両方測る**。実時間短縮は#2600/#2018の測定を伴って初めて主張する。

### 12.5 追加する反証ケース（T-35〜T-44、全てMacro縦断はNOT RUN）

| Case | 同じに見えるものと実際に変わるもの／期待する判別 | 既存Owner |
| --- | --- | --- |
| **T-35 非出力WHERE依存** | 同じ`order_id`が返ってもPredicate閾値50→70で意味上の対象集合が変わる。出力byte一致だけで旧Action選択資格を再利用しない | #2387 QX-25/27、#2582/#2514、Draft #2634 |
| **T-36 非出力Sort/Existence** | 非表示Sort/EXISTS/NOT EXISTSに依存するTOP-N・採否が入力追加で変化。値Lineageだけのキャッシュが旧Claimを確定扱いしない | #2387/#1706/#2582/#2622 |
| **T-37 解析Coverage不完全** | ColumnRefs/依存閉包の一部が欠損しても`full`等の文字列だけで十分性PASSにしない。unknownな範囲は必要な再評価へ | #2387、#2634/#2514 |
| **T-38 Mapping変更とBasis** | 同じOperation ID/Basisでも、選抜Mappingの非PK列やStorage型が変わると必要Inputs/実Read・Target適合が変わり得る。修正前挙動は#2632のGo特性候補で検証 | #2611/#2632/#2600/#2598 |
| **T-39 必要Member欠落** | 全Memberの内部整合がPASSでもMacro/ConsumerのExpected Script/Plugin/Resourceが欠落。Result/Runnable資格を区別 | #2482/#2630/#2633、#2502 |
| **T-40 公開中世代切替** | ManifestはR1、子Artifactの一部がR2等の途中公開に遭遇。Consumerが同世代byteを実読取するか不一致を明示する | #2604/#2630、#943/#2390 |
| **T-41 履歴Receiptと失効Grant** | 初回の業務Effectは確定・履歴Receiptも存在。後日Grant失効しても歴史を改ざんせず、今のMacro再実行・閲覧資格を別評価 | #1174/#2499/#2516 |
| **T-42 無関係変更の局所維持** | D1だけに依存する計算値に対しD2の説明/一部Dataだけ更新。依存閉包が既知なら該当Claimの証拠を保持し、全失効を避ける | #2582/#2600/#2514 |
| **T-43 F0/F1/S1/S2の負例比較** | Query Rule/Provider/Artifactの1要素変更後、fullとselectiveの観測対象・期待値・実Loaded member・再検査費用を比較。誤った再利用と過剰失効を両方数える | #2600/#2636/#2018 |
| **T-44 ABAとCurrent Disclosure** | Script/RuleがR1→R2→R1へ戻りbyte同一でも、組織Policy/開示先/Grantは旧状態へ復活しない。現在Consumerだけの再資格を行う | #2600/#2582/#2499/#2516 |

**担当への受渡し**：S0でT-35〜37/42（依存の説明と選択的影響）、S1でT-38（Source→Mapping→Intent）、S2でT-41/44（現行権限と歴史）、S3でT-39/40（Macro公開・実Loadと開示）、S4でT-43（他Target・多Consumer・再利用費用）を既存Ownerへ接続する。#1119のP0-a/P0-bは維持。新規の共通Graph/Parser/Cache/Receipt/Workflow/Authority Engineをここに創設しない。

**資格**：QX-25の局所SQLite実測、#2634/#2632/#2633の実Goテスト候補、#2600/#2636の文書手順は、実行場所・exact head・対象が異なる。全T-35〜44のMacro〜実Host/Provider〜Publisher〜Disclosure縦断、正式意味受理、配布・性能試験は**NOT RUN**。個別Issueの実測を横断PASSと合算しない。

## 13. 実CLIから外部Consumerへの最小縦断：発見・表示・実行・受領を混同しない

### 13.1 2026-11-02にmainへ統合された実装を、新しい検証起点にする

前節までのSource-staticな基点`9c9ef4d...`と区別し、**[#2587](https://github.com/hat47x/tei-platform/pull/2587)は2026-11-02にmainへ統合済み**（merge commit `8464db0bbbac216ede15d3344e72b7e8bd90cb65`）。[Status更新 #2643](https://github.com/hat47x/tei-platform/pull/2643)を含む参照mainは`d8f5114f17793354bbab0c766190699ad2b5d60b`。これは#2587で実装したread-only Operation Surface・Artifact導出と既存Capabilityへの合成束縛が**製品コードへ存在すること**を意味するが、P0-aの実PostgreSQL並行更新やMacro実ロード、Owner業務意味受理を意味しない。

mainの`cmd/tei/main.go`／`cli_operations_artifacts.go`／`cli_contract.go`を再確認すると：

- `tei operations -workspace <root> [-application <id>]`は`operations.SurfaceWorkspace`を呼び、導出した操作と不足・衝突をJSONで返す。Sourceからのread-only導出であって、そのまま実行許可や利用可能な全操作を示すものではない。
- `tei artifacts -workspace <root> -application <id>`はArtifact Set（Manifest/Findingsを含む）をJSONで標準出力へ出す。`-out <dir>`を指定すると**ローカルファイルを変更する**。その場合、標準出力は`set.Manifest`のみとなり、実際に書かれる`manifest.json`には`{manifest, findings}`が含まれる。`-out`は「read-onlyな情報取得」と同じ扱いにしない。
- `tei commands`の`tei.cli.discovery/v0`は明示的に`selected-read-only-commands`という**限定カタログ**で、現在の`operations`と`artifacts`は列挙されていない。一方、`main.go`のコマンドdispatchには両者が実装されている。**カタログへの非掲載≠CLI未実装**、また**実装存在≠Skill/利用者に対する実行資格**。Skillは実Version/Help/契約を発見し、存在しない命令を創作せず、対応差を報告する。
- [`reference/standard-crud/workspace`](https://github.com/hat47x/tei-platform/tree/main/reference/standard-crud/workspace)のApplication`ledger`は`sales.note`の`read/create missing-semantics`、`sales.order`の`retire missing-input`などを持つ合成負例。`validate`が`ok`でも、Operation SurfaceのGapsは残り得る。
- [#2587の保存済み検証記録](https://github.com/hat47x/tei-platform/blob/main/plan/evidence/SURFACE_AND_ARTIFACT_VERIFICATION_2026-11.md)ではGo build/vetとFocused test、二Workspaceのvalidate成功を報告。実DB競合の試験は`TEI_TEST_POSTGRES_DSN`未設定でSKIP。今回の新Consumer接合のGo/CLI実行とは区別する。

この既存main実装を最初の**製品Sourceに接続できるread-only縦切り**とし、未統合の複数Mapping負例Draft #2632/#2633等の特性記述を、既にmainで修正済みという意味にしない。

### 13.2 最短の一件：同じSubjectを「実コマンド→結果→用途別Consumer」に接合する

**M0（Product CLI確認）**：固定checkoutで`go build ./cmd/tei`または同じソースへの明示実行を選択し、`version -json`・`commands`・`validate -workspace reference/standard-crud/workspace -json`・`operations -workspace reference/standard-crud/workspace -application ledger`を別呼出しで観測する。実バイナリ/Revision、Workspace/Source fingerprints、exit/diagnostics、Operation IDとGaps、`commands`への非掲載を一枚に並べる。異なる呼出しで同一Snapshotを保証できない場合は、そのClaimを限定する。**これは実行手順案であり、今回これらのコマンドを新たに実行していない。**

**M1（安全なArtifact Consumer）**：同じApplicationで`artifacts`の標準出力を取得し、Manifest/Findings/Operation Surfaceの対応を読取り専用で確認する。`-out`を試す場合は明示した隔離ディレクトリを用い、実際に書かれたファイル・manifestの全Member・必要Applicationの範囲を読み直して照合する。Manifest内部整合、期待Memberの充足、物理公開の同一世代、Readerが本当に読んだbytesを別欄にする（#2482/#2604/#2630）。`-out`の既存ディレクトリへの無条件再実行を安全なread-only手順としない。

**M2（Product Skillの正当な案内）**：[TEI Product Skill #2605](https://github.com/hat47x/tei-platform/issues/2605)は現在の`yi-tei-modeling`読取り専用調査の説明・検査を担当する。Skillを使わない同じCLIと、Skill/外部Harnessを使ったread-only調査で、実コマンド・Source・診断の一致、漏れるTool名/Schema/Warning、現在のDisclosureを比較する。[TAI Draft #270](https://github.com/hat47x/tai-pleroma/pull/270)・[TAI #271](https://github.com/hat47x/tai-pleroma/issues/271)・[TAI #272](https://github.com/hat47x/tai-pleroma/pull/272)へ引き継ぐ。**Toolの発見・Skill文書の記述はCapability発行ではない**。User Macroの起動/更新は#2499/#1174が別途認可する。TAIの導入がTEI単独CLIの必須前提になってはならない。

**M3（Operation実作用の別資格）**：read-only Surfaceに現れた`ledger.update.sales.order`はAction候補であり、実Host RequestでもCommit完了でもない。利用者Macro→typed Intent→実Provider経路/現在Grant→Owner Effect→独立Receiptの同一ケースに、#2598/#2499/#1174の既存受入を接合する。データ取得中に権限が変わる負例は[#2637 D2のpreflightとpost-load確認](https://github.com/hat47x/tei-platform/pull/2637)へ委譲する。後段の再確認があっても、その前にRawをOS/Networkへ送った可能性、読取中ABA、実DB Snapshotまで安全と主張しない。

**M4（説明と成果物の実受領）**：業務上確定したEffect・初回BusinessResult・現在の説明・SHOの保存設計書・外部EKI成果物の納品は別。SHO/外部Consumerのread-only引用は#2516/SHO #10、実Reader/Source再解決はMMI C-01/T2と#2604、EKIのAttempt Bodyは[#465](https://github.com/hat47x/eki-fabric/issues/465)/[#409](https://github.com/hat47x/eki-fabric/issues/409)が検証する。**Claim ACK＝同じAttemptのBody受領＝元依頼者の実Read＝納品完了、とはしない**。

最初にM0/M1の同一実Application read-only evidenceを成立させ、その後M2/M3/M4を相互独立のConsumer資格として段階接続する。すべてを一回の巨大な新Runtime/E2Eを作る前提にはしない。

### 13.3 最終回答に現れない「観測可能な面」を分ける

[TAI #272の横断計画](https://github.com/hat47x/tai-pleroma/pull/272)では、**最終Answerが同じでも**実行前のTool候補、UI/SHO Rendererのリンク構造、Accessibility Tree、Session/Network Log、実Reader/出力Bodyが異なり得る合成比較を行っている。Macro/Skill/SHO Consumerでも、次を別の観測面とする。

1. **Source/Role判定**：引用や例示にある`role: trusted-script`、`op=delete`等を、正式Role/実行命令へ昇格させない。[#2638 Role bootstrap](https://github.com/hat47x/tei-platform/pull/2638)の依存Pin・Eager cycleと正当な遅延Refを区別し、Role/Plugin版の自己申告だけでTrustを得ない。
2. **候補の外部表示**：未公開Tool名やMacro/Action名、利用可能性の差、内部検証のBoolean、失敗件数や診断で非公開の存在を漏らす可能性を評価。意図的に許した公開と秘匿Profileを区別する。
3. **Rendererの実出力**：Markdown文字列の一致ではなく、実DOMのリンク、Accessibility、コピーされた文字列、保存されたSHO文書の表示・Source linkをConsumer別に検査。表示文書の適合はマクロの実行Permissionと別。
4. **実呼出しと送出**：実Tool名/args、実PayloadとProvider、OS/Networkへのbyte/送信先、実Grantと拒否箇所を確認。read-only宣言、UI`enabled`、Tool説明をOS隔離の証拠にしない。
5. **結果の回収とDelivery**：Task/Run/Claim/ACKと、同一Attemptの実Body/digest・受信者のRead・現在Disclosureを別に扱う。空の正当Resultと未受領/Reader失敗を同一視しない。

**公開の非干渉を主張する場合**、保護された値・Tool可用性・Source注記だけを変えた二世界について、**対象Consumerに実際に観測される全ての公開面**を比較しなければならない。最終Answer本文だけの一致からTool候補やログの秘密非流出を主張しない。ただしTool候補の差を一律禁止するのではなく、そのConsumerに許した公開範囲と判定する。OS/実Browser/実Harnessの確認はTAI #232・TEI #2515へ委譲する。

### 13.4 M0〜M4に対応する独立受入（T-45〜T-53、全て新規NOT RUN）

| Case | 正負対と観測する事実 | 既存Owner |
| --- | --- | --- |
| **T-45 実CLIと限定Catalog** | main`commands`は`operations/artifacts`を載せず、実dispatchは持つ。非掲載=不存在/実行許可と誤推論せず、実Versionとhelp/コード・具体Contractを照合 | #2605、#1155/#2354 |
| **T-46 Operation Gaps** | `standard-crud`の`validate ok`と`missing-semantics/missing-input`を同時保持。操作候補から現在の実行権限を生成しない | #1127/#2611/#2502 |
| **T-47 Artifact取得方式** | `artifacts` read-only stdoutと`-out`ファイル変更を区別。stdout Manifestだけから実`findings`/必要Member/公開世代をPASSにしない | #2482/#2604/#2630、#2605 |
| **T-48 Source引用とRole依存** | 説明Markdown内の`trusted-script`文字列、RootだけpinしたRole依存、Eager cycleを正式Macroとして公開・実行しない。正当な再帰Refは別評価 | #2620/#2621/#2638、#2502 |
| **T-49 読取中のGrant失効** | 取得開始時に許可、その後失効→返却直前の再確認で拒否。保持中の旧認可からRaw/Derived結果を返さず、既往OS送信やABAまでは保証しない | #2637/#1640/#2515 |
| **T-50 同じAnswer・異なる候補面** | 最終回答同一でも非公開Tool/Actionの候補名・Error/件数が外部へ出るか。許された公開と保護された面を区別 | TAI #272、#2516/#2504 |
| **T-51 Renderer/SHOの実リンク** | 同じMarkdownのテキストでもRenderer構成が違うと実DOM/a11y/出典リンクが変化。本文・Render・保存/公開資格を区別 | TAI #272、SHO #7/#10、#2516 |
| **T-52 ACKと同AttemptのBody** | EKI Claim ACK後も実Bodyが未送達ならMacro成果物納品済みとしない。Body失敗時に業務Actionを新Attemptで無確認再実行しない | EKI #465/#409、#1130 |
| **T-53 二Consumerの根拠接合** | 同一Source/版/SubjectのCLIとAI/SHO Consumerを比較。別Snapshotや開示条件、実Loader版が不明なら部分一致を全体PASSへ昇格させない | #2516/#2605、TAI #271、SHO #10 |

**停止線・引継ぎ**：初回の最小実行はM0→M1のread-only CLI（実`reference/standard-crud`を利用）だけを#2605/#2502が共有。実業務書込みP0-a、二Function Rule P0-bは#1119のOwnerを優先し、新しいGateとして追加しない。M2のSkill実Harness試験、M3の実Host/Effect、M4の文書・Body受領は各Ownerが実装し、同じ版・対象・Consumerの証拠が得られるまで`NOT RUN`／`unknown`で接続する。新YI中央Host/Registry・TEI専用Skill実行Runtime・第二認可/Role Resolver/Document Storeは追加しない。

## 14. Source取得時点と受理境界の限定証拠：モデルを読めたことを、実行資格へ昇格させない

### 14.1 統合済みSource Indexにある「読取り用」と「受理用」の差を明示する

[#1017のLoadとSource Indexの結合](https://github.com/hat47x/tei-platform/issues/1017)、[issuer側修正 #2642](https://github.com/hat47x/tei-platform/pull/2642)、[trace Source読取り #2647](https://github.com/hat47x/tei-platform/pull/2647)を、MacroのM0〜M4に関わる**Source取得資格の相違**として整理する。照合した現mainは`b1c019fec4f05ca65e3d7491608cb849ce523d29`（2026-11-02）。`#2647`は取得時点でOpenであり、mainに取り込まれた扱いにはしない。

現mainの具体的な分岐：

| 入口 | Source Index構築・観測 | ここから言えないこと |
| --- | --- | --- |
| `tei operations`：`internal/operations/surface.go::SurfaceWorkspace` | `workspace.Load(root)`→`workspace.BuildSourceIndex(root, loaded)`→World/ModelRevision/Operation候補。read-only。 | 正式受理済みのSource、現在Grant、実Provider選抜・実Effect |
| `tei artifacts`：`internal/artifact/workspace.go::RenderWorkspace` | 同じく`Load`→**非strict**`BuildSourceIndex`→生成Manifest/Findings。`-out`のみ実ファイル書込み。 | 外部Consumer必要Member・同世代の実読取り・納品済み |
| `sourceacceptance.LoadVerifiedWorkspace` | `Load`→**strict**`BuildSourceIndexFromLoad`→Trusted Readerの現在Source受理を確認。`ManifestIdentity`は別に保持。 | 複数ファイルの原子的Snapshot、import先一式の束縛、実Host/Provider能力 |
| `reference-host-acceptance sign` | #2642統合により`BuildSourceIndexFromLoad`を使用。import宣言Workspaceは、対象が未束縛なので**署名を拒否**する。 | 署名作業だけで業務意味を承認した、Provider実Effectを許可したという主張 |
| `trace`の元Source解決 | #2647の**未統合候補**では`workspace.ReadSource`の固定Root・上限・個別再読取りを検証。 | 毎回のRoot保護から、複数読取りを一回の不可変Snapshotと推定 |

**注意すべき現行コードの事実**：`BuildSourceIndex`は非strictでも、`workspace.Load`から渡された**解析済みSource SnapshotやLoad時に捕捉した管理対象外解析ファイルのbytesが存在すればそれを優先**する。捕捉範囲の外はルートに拘束したファイル読取りへfallbackできる。一方、`BuildSourceIndexFromLoad`はfallback不可・ResultがLoad由来であること・同Root・import未束縛を確認する。したがって「`operations/artifacts`は常にSourceを再読込みしている」と主張してはならない。保証の差は**strict fallback拒否とimport/manifestの束縛範囲**にある。

strict方式でも`workspace.Load`自身のscan/各Sourceの読取り・追加管理ファイル取得は別時点であり、**複数ファイルを原子的に同一時点で取得した証明ではない**。`ModelRevision`やFingerprintが得られたこと、`sourceacceptance`で過去の受理recordが存在することも、実業務効果の現在認可とは別である。

### 14.2 三つの時点と一つのConsumer観測を混ぜない

ここでは新Clock APIを定義せず、既存のSource/Attempt/Context/Receiptを用いて時点別の受渡し票を作る。

1. **Authoring/Source時点**：入力の実bytes、Loader解析済み集合、ManagedFiles/Extra、Manifest/Imports、Role/Operator/Rule Source、Source Index。読み損ね・未知経路を`complete`にしない。
2. **Accepted/Published時点**：Ownerの意味受理/組織公開対象、Issuerが検査したSource、Script/Plugin依存閉包、実Artifact一式と実Loaded bytes。同じSource文字列を含んでもRole版・Plugin版・Manifest受理は別。
3. **Invocation/Commit時点**：Macroを起動する主体/Context Instance/選択epoch、現在Grant・実Provider/物理Read・Operation Intent/実Request/実Effect・Receipt。過去のAccepted recordから現在Provider/Scope/時点を補完しない。
4. **Consumer観測時点**：CUI/Studio/AI/SHO/EKIの実Readerが見たProjection、診断、隠された候補の有無、Rendered DOM・Network/Log・同Attempt Bodyの到達。生成時点の`Manifest.Complete`やTask ACKを納品・開示適格性へ格上げしない。

この四者は**同じ時点を要求するのではなく、Consumerが今主張しようとするClaimに必要なJoin条件を実際に満たすか**で接続する。例えば次を区別する。

- `WorldFingerprint`一致でも、管理ファイルの位置・README等の非解釈管理内容、Manifest/Import、Provider Policyが変わればSource/採用資格は異なり得る。
- `ModelRevision`はSource Indexに由来し、`sourceIndexFingerprint`は別形式の観測値。値が異なるから不一致とする**無根拠な等号比較**をしない。`operations.modelRevision`と`artifacts.inputRevision`は同一Source版を比較する候補だが、別CLI呼出しで一回のSnapshotを共有した証明ではない。
- 同じ`Rule/Operator/Script`のbyteでも、Eagerな依存Role/Pluginが変わる可能性がある。[#2638 Role bootstrap](https://github.com/hat47x/tei-platform/pull/2638)の依存Pinと循環診断を受渡し条件とし、Referenceの自己記述やQuote中のRole名からTrustを生成しない。
- `read-only` CLIの実成功を、`sign`やTrusted Readerへの自動引継ぎとしない。import未束縛のTargetでは、read-only表示は可能でも受理・実行の証拠が不足する場合がある。

### 14.3 M0の証拠票を「値の一致」から「理由を説明できる関係」へ

固定対象は`reference/standard-crud/workspace`の`ledger`。独立の[既存検証記録](https://github.com/hat47x/tei-platform/blob/main/plan/evidence/SURFACE_AND_ARTIFACT_VERIFICATION_2026-11.md)を期待の一部として参照しつつ、**今回のM0は別試行**で実測・記録する。最低限の票は次を持つ。

| 段階 | 記録する証拠 | 受入条件・未知の扱い |
| --- | --- | --- |
| 実行物 | main exact SHA、実binary版`vcsRevision/vcsModified`、Go/OS、実行引数、CLI終了コード | 未識別実行物をmain同一としない |
| CLI発見 | `commands`のscope、実`main.go` dispatch、`operations/artifacts`の可否・非掲載 | 限定Catalog≠全Capability一覧、実装≠許可 |
| Source | `validate`の`ok`/Diagnostics、`worldFingerprint`・`sourceIndexFingerprint`、実Input Scope | Warning/parse errorを隠さず、別呼出しsnapshotは未束縛と記録 |
| Operation | `operations.schemaVersion`、`worldFingerprint`、`modelRevision`、`rule`、Operations/Gaps/Conflicts | Gapありでも`validate ok`。候補/Gap/Conflictを現在Capabilityへ変換しない |
| Artifact | `artifacts.inputRevision`、`manifest.inputRevision`、`manifest.entries`、`findings`、`complete` | 同Revision候補を比較し、内部整合/全Consumer member/納品を別資格化 |
| Source受理 | Trusted Readerでの`BuildSourceIndexFromLoad`とimport/manifestの条件、今の受理record | M0 read-onlyだけでは観測できない→**NOT RUN**を保持 |
| 実行・開示 | 現在Grant、実Provider Request/Effect/Receipt、各Consumer実Read/送出 | M0は観測しない→**NOT RUN**。実Host以外で推測しない |

**手順と結果を分離**：M0では実コマンドを記録し、各結果を生Sourceの再配布にならない場所へ保存する。`sourceIndexFingerprint`と`modelRevision`を無理に文字列比較せず、同じ対象・同じ定義時点に由来するかの説明と、`operations.modelRevision == artifacts.inputRevision`の比較可否を記録する。比較対象が欠落・別版・Opaqueの場合は、同一性を`unknown`とし、値合わせのためにSource/Grantを変更しない。これはコマンドを追加実行して得た結果報告ではない。

### 14.4 追加する境界反例（T-54〜T-61、すべて新規Macro縦断NOT RUN）

| ケース | 正負対・判別 | 主担当 |
| --- | --- | --- |
| **T-54 Read-onlyとVerifiedの区別** | `operations/artifacts`で導出成功。`LoadVerifiedWorkspace`のTrusted Reader/受理条件が不明なら実行受理済みとはしない。正当なVerified Sourceでは両者の差を説明 | #1017/#2605/#2499 |
| **T-55 非解釈の管理ファイル変更** | Worldの構造は不変だが、管理対象README等のSource bytesが変わる。Source Index/ModelRevisionの影響と業務Meaningの変化なしを区別 | #1017/#2600/#2516 |
| **T-56 Load後変更とstrict fallback** | Load直後のディスク変更、捕捉されていない管理ファイル、別RootのResultを組み合わせる。strictは再読込みで修復せず明示拒否。非strictで成功しても同資格としない | #1017/#2642/#2595 |
| **T-57 import/manifestの束縛範囲** | import宣言のSource受理/署名は未束縛として拒否。manifest identityを別に見てもimport先や複数ファイルatomic Snapshotを受理済みとしない | #1017/#2642/#943 |
| **T-58 M0別呼出しのSource drift** | `validate`→`operations`→`artifacts`間でSource変更。`WorldFP`/`ModelRevision`/Artifact入力版の既知比較と、途中のABA検知不能を分ける | #2605/#2600/#2514 |
| **T-59 trace個別読取り** | 固定Root/上限付き`ReadSource`の安全性と、複数ファイルを同時点に読む保証を区別。#2647の未統合候補のClaimをmainの既定挙動にしない | #1017/#2647/#2516 |
| **T-60 Role/Skill依存の実ロード** | Source byte同一でRole Plugin/Skill参照Script変更。依存版Pin・実Tool/Loader・Host Current Grantが揃わなければ公開や自動実行は不可 | #2638/#2605/#2499、TAI #270 |
| **T-61 実成果物/消費者の閉包** | `artifacts`の`complete=true`とstdout/`-out`表示が一致しても、Readerが必要memberを同世代で取得したかとOwner現在権限を別評価 | #2604/#2630/#2633/#1174 |

**接続順**：M0の直接対象はT-54/55/58のread-onlyな出力比較と、実Source範囲を明記する票の作成。T-56/57/59は#1017/#2642/#2647のOwnerが実Source/OS差分を確認。T-60はRole/Skillの既存Owner、T-61はArtifact/Readerと実Hostの既存Ownerへ引き継ぐ。これらの試験は問題を一般的に証明するための新Gateではなく、**M0からM4への不正な資格昇格を発見する具体的な負例**である。P0-a/P0-bは不変。

**証拠水準**：#1017/#2642のmain統合を確認したことと、Sourceに関する上の静的判別ができることを区別する。#2647のOpen PRの現行Windows試験や#2638の合成Role Referenceの結果は各Ownerの記録だけに帰属する。T-54〜61・Macroが実Source受理から業務Effect/外部開示へ到達する一巡・実Harness/OS/実Providerは**NOT RUN**。

## 15. 空結果からの不存在判断と実証の接合

### 15.1 0件という表示と、業務上の「存在しない」は別

[#2586](https://github.com/hat47x/tei-platform/issues/2586)／[#2582](https://github.com/hat47x/tei-platform/issues/2582)の片側Claimと非出力依存を、マクロの業務Actionへ接続する。例えば「未処理の割当が無ければ受注を確定する」とき、画面に0行しか表示されないという観測だけでは、条件を満たす割当が実際に存在しないとは限らない。

比較する内容は`U = 必要な業務対象集合`、`P = 未処理と判定する述語`、`R = 実Providerが返した集合`、`t = 観測時点`である。例示記号であり新しいTEI内部Schema/数理APIを提案しない。

- **存在の証人**：信頼できる`R`の中に`P`を満たす実体が1件あれば、同じScope/時点における「該当なし」は否定できる。全件読取りは必要とは限らない。
- **不存在の成立候補**：`R=∅`だけでは不足。`P`が要求する`U`の完全取得、適切なTenant・Relation・粒度、Query/Providerの実読取り条件、時間・RLS/権限制約が満たされて初めて「`t`における不存在」を候補にできる。
- **部分取得・不明**：検索結果0件、ページ打切り、未ロードRelation、RLSにより除外された行の有無を確認できないなら`not-established`を保持する。表示上の空と、業務Universeの空を同一視しない。
- **現在の実行資格**：`t`時点で完全だったとしても、`tCommit`までに新しい割当が作られる可能性がある。Ownerの現在Action条件、適切なDB制約/Tx/並行更新検査なしに、過去の不存在を確定判断へ転用しない。
- **限定開示**：Hostが非公開行を用いて業務条件を確かめることと、利用者へその行や件数を開示することは別。Secretを含む診断・競合時の現在値をResultに出さない正当な設計を許す。

この区別を`NOT EXISTS`、反結合、空のJSON配列、Graphの「子がいない」、文書索引の「見つからない」、AIの「証拠未発見」まで広げて評価する。各Domainで`U/P/t`と検索の十分性は異なるため、単純な`exists/absent`の真偽値へ統一して十分性を失わせない。

### 15.2 実PostgreSQL CASが確認したことと、集合条件への不足を分ける

[#2651](https://github.com/hat47x/tei-platform/pull/2651)・[#2653](https://github.com/hat47x/tei-platform/pull/2653)の局所実証では、`tools/reference-expected-state-update`のPostgreSQL 14.24に対する単行`UPDATE ... WHERE key=$1 AND value=$2`、競合・不存在・冪等の分離、現在のRaw値を競合結果へ返さない方針が確認された（担当Issueの隔離DB8件の実行記録）。これは**実DBの限定された一行更新の根拠**であり、#1174の実Owner Effect Fenceや、多行の「該当なし」の現在性まで実証したものではない。

次の縦切りは、`order1001`の現在Statusだけを比較して更新できても、未処理割当が存在しない保証にはならないことを正例/負例で示す。Owner受理済みBusiness Actionの必要条件が集合に及ぶ場合、その処理の実Transaction/DB制約と独立Observationで検査する。現在値非開示は守り、内部比較用Rawを無条件にAI/Studioへ戻さない。

### 15.3 M0を実コマンド・実Fixtureに接続した独立Reference（main統合済み）

これまでM0はCLIからのread-only受渡し計画だった。その後[実装実証 #2678](https://github.com/hat47x/tei-platform/pull/2678)がmainへ統合され（2026-11-02 21:52 UTC）、exact-head CIの合成Checker/実Go CLI両jobが成功した。統合したこのReferenceでは、mainで実在する`tei operations`／`tei artifacts`を`reference/standard-crud/workspace`の`ledger`に適用する検査器・合成負例11件・exact-head GitHub Actionsを**既存CLIに依存する別レーン**として実行・確認した。

M0は`validate/operations/artifacts`のWorld版・Rule/Gap・Manifestを照合し、`-out`書込みや実Providerは実行しない。[#2626](https://github.com/hat47x/tei-platform/pull/2626)の実Go CLI→人/AI二表示→Source移動の成功実測とは対象とScopeが異なる。**M0の固定Public Fixture read-only実測は成立**したが、局所PASSをOwner意味受理・Macro実権限・実Provider/SHO/EKIの合格へ自動昇格しない。

後続のM1実Consumer全Memberは[#2630](https://github.com/hat47x/tei-platform/pull/2630)を参照し、JavaのStore閉包は[#2652](https://github.com/hat47x/tei-platform/pull/2652)の実JVM独立6試験（復元class集合から公開10 Fixture）を異Targetの正例として利用する。Main.class単体のDigestは全class依存の完備を示さない。M3実Grant/DBとM4文書・Body実受領は既存Ownerの受入へ渡す。

**追加のCore設計判断は行わない**。既存T-35/36（非表示Predicate・結果選択）、T-41（過去のReceiptと現在資格）、T-54/58（CLI ScopeとSource版）、T-61（実成果物の閉包）の受入条件を、今回の具体Fixtureに結び付ける。Macroの`NOT EXISTS`に必要な正負/時点条件は実Provider・Ownerを接合してから判断する。

## 16. 実Commandから実効果への契約接合：Studio参照実行、Grant、Tx、観測を一つの成功へ潰さない（2026-11-03）

### 16.1 現在mainで利用できる実装と、接続されていない契約

2026-11-03のmain `0e18b916aa15bc7768e9abfb34318b98ee497cb3` から、[merged #2747](https://github.com/hat47x/tei-platform/pull/2747)、[#2746](https://github.com/hat47x/tei-platform/pull/2746)、[#2768](https://github.com/hat47x/tei-platform/pull/2768)、[#2769](https://github.com/hat47x/tei-platform/pull/2769)、[#2771](https://github.com/hat47x/tei-platform/pull/2771)、[#2773](https://github.com/hat47x/tei-platform/pull/2773)の実コードを静的に照合した。**複数の機能がmainに存在することは、これらが実DB/同一Host/同一Attemptで縦断接続された証明ではない。**

| 実装済みの接点 | コード上の確実な境界 | Macroが追加で必要とする証拠 |
| --- | --- | --- |
| Studio Command | `internal/studiohost/command.go`の`ExecuteCommand`は`command.ReferenceExecutor`と`MemoryUnitOfWork`を使用し、`Mode="reference"`でも`Committed=true`を返す | Studioの「参照内Commit」を**実DB確定/Owner Effect/Receipt**として表示・AI説明しない。`PlanCommand`の表示は実行資格ではない |
| Command Invoker | `internal/command/invoke.go`は入力Plan/Periodを確認し、`UnitOfWork`が存在すれば`Begin→Executor→Commit`を呼ぶ | 同じ物理Txに属することが示されない別Runner/UnitOfWorkを二重Commitの一式としない |
| 認可付き書込みAdapter | `internal/security/effectcommit/commandadapter/executor.go`で`command.Executor`→既存`effectcommit.Runner.Execute`へ委譲。`Scope`はCommandのApplication/CapabilityとGrant主体/Action/Resourceから構成。`OperationID`、`ExpectedVersion`、`Payload`、`Grant`は**呼出側入力** | Trusted Issuerの実Grant、承認した業務対象と実Payload/実Provider Effectの一致、現在Owner条件、正しいIdempotency/Resource version |
| Adapter結果 | `committed/replayed`は`command.Result{}`と`nil error`、`denied/conflict`は区別可能なerrorへ変換 | 元の`Receipt`や`committed/replayed`の違いは現Command結果へ伝わらない。成功コードだけで初回確定・再送・実Consumer受領を説明しない |
| Period Guard | `command.Invoker.checkPeriod`は`Periods=nil`で検査を通過する。`command.MapPeriodState.Closed`の**未記載Period**は`false`、`MemoryPeriodState.Closed`の未記載はerror。`execution.PeriodGuard.Source=nil`もGuardを実行しない | 業務MacroでPeriod前提が必須なら、現在状態未取得を`open`扱いする可否をTarget/Ownerごとに明示。Reference既定を本番の失効拒否保証へ昇格しない |
| Effect Rule/Observer | `internal/execution/effects.go`の`RunEffect`は`after-commit/compensation`に対する`Invoker→Observer→Journal`を試みる。`SQLEffectObserver`はQuery/Argsがあれば`count≥1`で`Observed=true` | Queryが**当該Operation/業務対象/時点の効果だけ**を見ていること、実対象ArgsとDB Snapshot、行の有無以外の業務完了条件。SQL count一般を因果Receiptとしない |
| 補償 | `CompensationsFor`は同じCapability/EffectDataを選抜。`Compensate`は宣言順に全件を試み、失敗も記録する設計 | 補償はDB Rollbackや業務全体の回復と同じではない。補償自身の部分失敗・再送/完了判定・別Ownerの効果は未適合 |
| 実PostgreSQL CAS | [merged #2651/#2653](https://github.com/hat47x/tei-platform/pull/2653)の隔離実DBで単一行のCompare-and-Swapと現在値非開示の限定実測 | `effectcommit`の同一DB/同一Tx実Grant・Receipt、複数行`NOT EXISTS`条件、Command Adapterからの実DB効果をこのReferenceのPASSで代用しない |

### 16.2 静的に認められる未接続辺を、具体的な反証にする

**A. 意図した対象≠渡したPayload**：例えばOwnerの許可は受注`ORD-1001`、Command Inputsにも同じ値があるが、Adapterに渡された明示`Payload`が`ORD-1002`を指定する。現在のAdapterは`Payload=nil`の場合には`request.Inputs`をJSON化するが、明示`Payload`とInputs/業務Targetの**意味上の整合は検査しない**。`effectcommit`のScope/Grant照合は主体・Action・Resource等を検査するが、そのまま個々の受注IDとPayloadの意味対応を証明しない。これは**成立し得る構成の未検証条件**であり、実Owner Providerで不正更新を再現したという主張ではない。

同じActionが明細や在庫を合法に更新する場合もあるため、ID文字列を単純に一致させて全更新を拒否するのではない。Ownerが受理した`Intent→許されたTarget集合・変換→実Request Payload→Provider確認済みEffect`を、実Targetの資格・同一Attemptの独立Expectedで照合する（既存T-1、#2598、#2499、#1174）。

**B. 二つのCommit境界**：`command.Invoker.UnitOfWork≠nil`と、別`effectcommit.Runner`の独立した`Provider.Begin/Commit`を同時に指定できる構造がある。後段の外側`UnitOfWork.Commit`だけが失敗しても、内側の副作用が既に確定している可能性を否定できない。正式接続では**副作用の確定を所有する物理Txは一つ**とするか、複数効果なら各Ownerの明示的な独立Receipt/補償契約へ分ける。外側の失敗を一律Rollback済みとして自動再Submitしない（既存T-1/T-41、#1174/#1129）。これも現製品で二重Commit障害が発生したという実測ではない。

**C. Period未取得**：ClosedならReferenceでも拒否できるが、必須Periodの`Periods=nil`や`MapPeriodState`の未登録IDで通過し得る現行経路がある。`no period declared`、意図的にGuardしないTrusted Tool、**宣言があるが現在状態が不明**を別状態にする。一般利用者Macroの本番作用に必要なPeriodについては、Ownerが要求する現在状態と取得時点がない限り、該当する実Effect資格を与えない案をまず反証する。期間を一律専用テーブル/Workflow Engineへ移すことは要求しない（#2144/#2744/#1129）。

**D. Effect観測の相関と記録空白**：`SQLEffectObserver`のテストは固定`SELECT count(*) FROM stock`の結果が2なら`Observed=true`を確かめるが、特定Operationの効果を生んだ2行とは照合しない。さらに`RunEffect`では、Invokerが成功したあと**Observerでerrorが発生した場合、Journal.Appendへ到達する前にreturnする**経路がある。実外部効果の成否が不明のまま記録が欠ける可能性を、**現コード上の境界**として明示する。実Providerへ接続する際は、失敗前の試行/Intent/Receiptと観測不能状態を既存のEffect/Recoveryの正本へ残す方法を検証する。`Observed=false`／`observer error`／`Journal error`／`Effect denied`／`Commit unknown`を一つの`failed`へ畳まない。既存の#1129/#1174/#1130が実装責務を保持し、新しい共通Journalや全局Recovery Engineは設けない。

**E. 文書/Runtimeの型を混ぜない**：Goでビルドした単一実行ファイルをTEIの標準配布・実行形態とする一方、日常DSLのFrontendは**ドメイン専門家に自然な言語中立・低ノイズ構文**を優先する。`func/struct/package/import`をマクロの通常記述へ強制せず、Go式AST／独自Parser／Hybrid／DSL→Go生成は内部方式として別実験する。日本語専用DSLは採用しない。Unicodeローカル名を使う場合も安定したASCII Canonical IDへの明示Binding・元Markdown/Source糖衣保持・診断/修正候補のSource位置を保つ（#2667 `22〜27、[Draft #2745](https://github.com/hat47x/tei-platform/pull/2745)、merged #2674）。自然なAuthoringや`Mode=reference`を理由に、実Host Authorityが発行されたと解釈しない。

### 16.3 既存検証へ戻す最小接続順（新T番号を発行しない）

| 既存検証の再利用 | 最初の実証と独立Expected | 所有先・止める条件 |
| --- | --- | --- |
| **T-17/T-45：Preview/参照表示** | StudioのReference`Committed=true`が表示/AI説明で`実DB成功`へ変換されない。`Mode`/`Receipt`未取得を区別 | #1124/#2747/#2502。実DB未接続なら`Reference only` |
| **T-1/T-38：Payload/Mapping拘束** | 受注1001の許可・Inputs1001、実Payload1002、選抜Provider別Targetなど正負2系統。実WriteはHostの承認対象外へ行かない | #2598/#2499/#1174/#2773。偽Providerで通ること≠実Owner拒否 |
| **T-41/T-58：Period/現在Grant** | 提示した期間がclosed、未取得、更新済み／旧Grant、強制読取時点を変え、実Effectの手前で止める | #2144/#1129/#2744。`nil`と`open`の同値はOwner未受理 |
| **T-1/T-41：TxとCommit unknown** | 内側Commit後の外側Commit失敗、ACK喪失、同OperationID再照会。新ID/別Effectを勝手に再実行しない | #1174/#1129/#1130。実PG同一Tx未証明ならP0-a未合格 |
| **T-27/T-33/T-41：独立観測** | 正しいOperationでは0行、別Operationの行でcount=1、Query/Args誤Binding、読取失敗、Journal失敗。**因果誤成功**と記録空白を検出 | #1129/#2387/#1174。ObserverだけのCount PASSは業務完了ではない |
| **T-41/T-52：補償/外部Job** | 効果失敗、補償部分成功、再実行・EKI Attempt/Body未到達・SHO再読取を分ける | #1129/#1130、EKI #465、#2516。補償起動/Task ACK≠回復/納品 |
| **T-19/T-22/T-54：Authoring→実Load** | 同一Meaningの複数FrontendsとGo Backend、Source修正/Role Pin/Targetの実Loader資格を別列で照合 | #988/#2639/#2667/#2745/#2595。Go実行成功≠DSL可読性採用/Owner意味受理 |

**優先する一件**：既存の`sales.order.submit`を意味受理したと推定せず、公開合成の`order_entry`で`Preview(reference)→呼出側に固定した`Grant/OperationID/Payload`→`Command Adapter→Effect Fence`を**二つのTarget負例（正当な1001／すり替え1002）**で分ける。偽Providerの段階では`Runnerが引数をどう受け取ったか`まで、実DBの段階ではOwnerの受理したTarget/Tx/Receiptの実観測までを独立報告する。新しいHost・Workflow/Job実行器・共通Approval/Role Registry・DSL Grammarを先に作らない。EKIへの委譲は、TEI常駐の有無・資源/Node運用上の必要性がある場合の**任意の手段**であって、業務状態やOwner効果の正本をEKIへ移す決定ではない（#2667 `27）。

**実証状態**：上記のGo Sourceと各PRは静的に照合した。#2773の合成Fake Provider、#2771のQueryer模擬、#2747のStudio Reference、#2651/#2653の隔離PostgreSQL CASはそれぞれ**異なる局所資格**。今回提示した実Command Adapter→実Provider PostgreSQL→AfterCommit SQL Observer→Macro/Studio/外部Jobの**同一Attempt製品E2EはNOT RUN**。追加のT番号・新しいGateは設けない。

### 16.4 M3の最初の実Go負例を別Draftへ分離し、SQL/組織/EKIの意味依存を切らない（2026-11-03）

[独立のDraft #2780](https://github.com/hat47x/tei-platform/pull/2780)で、現在の実`command.Invoker→commandadapter.Executor→effectcommit.Runner`を呼ぶ**Go test-only characterization**を作成した。#2616 OB-07Aの`OperationSurface→Driver` Target漂移負例とは**接続箇所が異なる**。新Product Adapter/権限器を作らない。

| 既存受入番号へ割り当てる実Go Characterization | 固定されたExpectedと実観測の区別 | Ownerの正式合格条件 |
| --- | --- | --- |
| **MC-B01**（T-1/T-38） | Command Input`ORD-1`、呼出側明示Payload`ORD-2`の意図的な不一致。現AdapterがFake Providerの`WriteResource`までPayloadを渡す挙動を**安全性の不足として**記録。テストがPASSしても「不一致を許すことが正しい」の意味にはならない | Ownerが受理した`Intent→対象変換→実Target`と現在Grantを、実Provider/条件付きCommit直前まで照合。不一致を拒否する本来の安全性テストへ移行 |
| **MC-C01**（T-1/T-41） | 内側`effectcommit`のFake Tx Commitが成功した後、外側`command.UnitOfWork.Commit`だけ失敗。Commandが`COMMAND_COMMIT_FAILED`でも、合成内側Commit記録が存在することを観測 | 実Owner Providerで同一物理Txを保証するか、複数OwnerのEffectを個別Receipt/補償として扱う。ACK/Commit unknownから新OperationIDで勝手に再実行しない |

**保存と実行を別報告**：#2780にはGo試験Source・Focused exact-head Workflow・独立ExpectedのREADMEを保存。Source保存やPR作成時のCI開始をテストPASSと呼ばない。**実測更新**：[exact-head Focused Workflow #41620889140](https://github.com/hat47x/tei-platform/actions/runs/41620889140) が `3b0ead5706f914209c4c76e18ea30aac250cb86b`（Go 1.26.10/Linux）で `completed/success`。ジョブログで **MC-B01・MC-C01のGo 2件ともPASS**、`gofmt`検査とworktree不変を確認。検査したTest Blobは `d4f57f06c5e87621e0e389a264e8de35850f4e04`。旧head `27aa489...` は `effectcommit.Transaction.Rollback()` と `command.Transaction.Rollback(ctx)` の混同でコンパイル失敗し、修正後に再取得した。この**限定Characterization verified**は製品安全性/PG/HostのPASSではない。実PostgreSQL・正式Grant/Owner対象・実外部Consumer/OSは独立の未実証領域。

この最短縦切りへ、他レーンで確認された次の意味上の依存を**既存Ownerの受渡し**として追加する。

- [#2778 SM-15](https://github.com/hat47x/tei-platform/pull/2778)の隔離SQLite実測：同じ`NATURAL JOIN` SQL SourceでSchemaに同名列が追加されると結果集合が変化し、`SELECT *`のOutput Shapeも変わる。**SQL Source byte/Command Input/Source Revisionだけ固定しても、Schema/Operator/実Provider/集合完全性の変更がAction対象選択に波及し得る**。SQLをTEIの業務Meaning正本へ逆昇格させず、#1126/#2387/#2622が実Query取得/Schema依存を証明する。
- [#2681組織E1](https://github.com/hat47x/tei-platform/pull/2681)と[#2779合成what-if](https://github.com/hat47x/tei-platform/pull/2779)は、組織/Case移動・期間・権限対象が異なるOwnerの意味に属することを示す参照。組織改編のPreview/シミュレーションを`Grant`や実Case更新の自動承認へ転用しない（#2386/#2598）。
- [Everything is Data #2667 `27/31](https://github.com/hat47x/tei-platform/pull/2667)のEKI連携は**任意**。TEIはRule/Action/意味/検証候補、実Host/Providerは副作用・Receipt、EKI等はTask/Attempt/Lease/端末保護・資源配分。EKIの`AcceptedResult`、Task ACK、シミュレーション成功を業務CommitやCustomer Deliveryへ昇格させず、TEI単独Goバイナリでも通常のActionを実行できる設計を保持する。

**成熟した停止線**：最小製品安全性は`選択Subject/Context→実Target/Scope/Grant→条件付きOwner Commit/Receipt`の同一Attemptで検証する。広域/過去全件のData Quality、Replicaの遅延付き監査、EKIの長時間Taskは独立Runへ移せるが、現在RequestのTarget・Grant・Period・Tx・Commit不明処理を「夜間監査で検査するから」と省いてはならない。新中央Registry/Host/Workflow/Validation Engineは不要。既存T-1〜61とP0/Gateを維持する。

### 16.5 Effect Commitの結果資格をCommandの外へ失わない：成功・再送・不確定を別々に追う（2026-11-03）

既存`effectcommit.Runner`は`StatusCommitted`、`StatusReplayed`、`StatusDenied`、`StatusConflict`、`CommitUncertainError`、`Lookup StatusNotRecorded`を区別する。その直上の**二つの実装済みCaller経路を混同しない**。

- `internal/security/effectcommit/submit/submit.go`：注入`GrantSource`と`Intent`からRunnerを呼び、`submit.Outcome{Disposition,Status,OperationID}`として`Committed/Replayed/NotCommitted/Uncertain/Failed/NotRecorded`を区別。**正式な製品契約には未採用のA2 Reference**であり、Receiptの全てをConsumerへ返す完成品ではない。特に`Uncertain`には既存ReceiptのReplay処理の最後のCommit失敗も含み得る。`Lookup=NotRecorded`を「副作用は起こっていない」の証明にしない。
- `internal/security/effectcommit/commandadapter/executor.go`：Runnerの`committed/replayed`をどちらも`command.Result{}`／`nil error`に写す。`denied/conflict`はSentinel errorへ、CommitUncertainはGeneric`COMMAND_EXECUTION_FAILED`にwrap。ただしGoの`errors.As`では元の`CommitUncertainError`を辿れる。**Commandの通常Resultだけでは初回Effect/再送/Receiptを区別できない**。

この差は[#2780](https://github.com/hat47x/tei-platform/pull/2780)に新設した**MC-D01（初回CommittedとReplayedのCommand Result同一）／MC-D02（CommitUncertainがGeneric Command error下で型付き保持）**でGoの既存接合を通してCharacterizationする。前回のMC-B01／C01と同様、**TEST PASSは製品のResult/Recoveryの安全性が合格した意味ではない**。**実証更新**：[Focused #41632377548](https://github.com/hat47x/tei-platform/actions/runs/41632377548)では exact head `d70a99131b5298f44b6edc739b723621d1679674`、Go1.26.10/Linux、`gofmt`とworktree cleanで、**MC-B01／C01／D01／D02のGo 4/4 PASS**を確認。Test Blobは `d4f57f06c5e87621e0e389a264e8de35850f4e04`／`c212b621c8a06b57284043573cb79e0c904e7514`。#2780のfull Repository Health/全Provider試験の合格は、このFocused Runだけでは主張しない。

**実PostgreSQLの既知正例を誤って未実施へ戻さない**：[Run #41620889113](https://github.com/hat47x/tei-platform/actions/runs/41620889113) は隔離PostgreSQL 16で`tools/reference-first-web-target`の実`RunnerOrderSubmit`試験を実行し、Web/CLI両Callerのhappy/replayed、認可Epoch変更・取消、ACK喪失からのLookup、Receipt/独立ObservationなどがPASSした記録を持つ。同じ候補PRのWorkflowであっても、**この実DB試験は`effectcommit/submit`/Reference Caller経路**の実証であり、`commandadapter.Executor`に同じ永続化Providerを接続した製品Macroの合格ではない。MC-D01/02のFakeと実PGの成功を同一Attemptの閉包として足し算しない。

| 既存受入を再利用する場面 | Runner/実DB側で必要な事実 | Command/Consumerへ渡す資格と停止線 |
| --- | --- | --- |
| **初回確定**（T-1/T-41） | 適格な現在Grant＋Owner Target＋同一TxのWrite/Receiptと確定応答 | `committed`はHistory Resultだが、Userへ開示できること・SHO/AIの実Reader受領は別 |
| **冪等再送**（T-41） | 同OperationID・同RequestHashのReceipt一致、追加Writeなし | `replayed`は初回実行ではない。現在Data/Disclosureが変化しても旧Effectを再生しない |
| **拒否・衝突**（T-1/T-41） | 業務条件/版/権限からNon-commitの理由が分かる | Outcomeが宣言済みか共通非実行Dispositionか、技術ErrorかをConsumer契約で区別。未宣言Outcomeを捏造しない |
| **Commit応答不明**（T-41/T-52） | 同OperationID/Requestで独立`Lookup`、Receipt/実Effect/失効条件を照合 | Generic Command errorやTransport timeoutから`failed/rollback済み`と決めない。`not-recorded`だけで新OperationID再Submitしない |
| **独立観測失敗**（T-41/T-52） | 実Owner CommitのReceiptはあるがSQL Observer/結果配送が失敗 | Effect履歴は保持し、Consumer側の`observed/received`を`unknown`に留める。業務副作用を再実行しない |

**最短受入**：外部向けに一つの新しいResult schemaを先に確定せず、既存`Runner Result/Receipt/Lookup`と`command.Result/ExecutionError`の差分をConsumerごとの適用契約として列挙する。`submit.Outcome`を無条件に`Command.Result`へコピーせず、Ownerが`業務Outcome／非実行Disposition／技術障害／不確定Receipt`の帰属を決める。History Receipt・最新のRead Projection・現在Disclosureを分け、RuntimeのHot Pathへ全Source/全Rule検証を繰返し入れず、現在必要なTarget/Grant/Period/Tx検証だけは残す（#1174/#1129/#1131/#2502/#2667）。

## 17. 判断を残す点

- **業務Owner**：各Actionで「元の対象」「関係する対象」「全件の完了」をどのように定義するか。旧X-TEA `sales.order.submit`等の未受理マッピングは推測で補わない。
- **組織Owner**：固定集合／実行時集合、代理の時点、合議、実行版固定と作用範囲での更新許容、既発行Grantの変更時再判定、結果閲覧をどのPolicyで扱うか。
- **Host/Provider Owner**：現在Grantの取得・対象単位の照合・実Write fence・Raw/Secret/Egressの物理隔離をどこで保証するか。全Target同じ実装を必要条件にしない。
- **成果物/Delivery Owner**：承認対象と実ロード物・依存閉包の証拠、改版後の受理・公開・閲覧資格と取消をどう保持するか。
- **利用者体験Owner**：履歴Receipt／初回回答／現在値、実行/予約/失敗/納品の違いを、不要な専門用語なしに利用者へどう示すか。

### 現状の検証資格

**確認できたこと**：固定mainの既存型・RequestHash・通常Request生成・試験DDLのデータ解釈を静的に確認した。既存IssueにはT-1の具体的な未実行DB負例、Effect replayとResult再投影の未実行負例、EKI出力接続の静的所見がある。

**未実行・未採用**：本書のT-1〜61のMacro縦断実行、信頼済みIssuer/実Grant、利用者Macro実ロード、PostgreSQL負例、Linux/WindowsでのOS隔離、外部EKIの同一Attempt納品、複数Target受入、業務意味の正式受理。新しいCore kind、Grant/Receipt schema、専用Host、共通Workflow Engine、仕様やADRの採用を本書から確定しない。
