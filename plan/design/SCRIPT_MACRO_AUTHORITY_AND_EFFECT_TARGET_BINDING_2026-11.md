# TEIスクリプトの権限委任と実際の操作対象を結び付ける設計

- **状態**：設計提案・未採用（2026-11-02）。この文書は仕様の正式採用、実装完了、権限付与、本番利用の許可を意味しない。
- **担当課題**：[Delegated Authority Bridge #2499](https://github.com/hat47x/tei-platform/issues/2499)（業務判断から限定権限への受渡し）、[Script / User Macro #2502](https://github.com/hat47x/tei-platform/issues/2502)（製品体験）、[Script Execution Policy #2504](https://github.com/hat47x/tei-platform/issues/2504)（実行隔離）。
- **実行・意味の正本**：[Effect Fence #1174](https://github.com/hat47x/tei-platform/issues/1174)、[Data lifecycle #1129](https://github.com/hat47x/tei-platform/issues/1129)、[組織意味 #2386](https://github.com/hat47x/tei-platform/issues/2386)、[Service Contract #951](https://github.com/hat47x/tei-platform/issues/951)、[Execution Context #1118](https://github.com/hat47x/tei-platform/issues/1118)。
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

表示や活性を決める式は、型・依存・Context Instanceと選択epoch・NULL/未取得・意味版・Rule変更に従って評価する。§enabled=true§でも実行Grantが失効していればEffectは拒否される。§enabled=false§はUI側の状態であり、業務Serviceの現在認可や外部Consumerの実行資格を直接書き換えるものではない。Actionの候補提示・起動・認可・確定・開示を一つの真偽値へ丸めない。

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
| **T-15 活性表示と実認可** | UIで§enabled=true§でもGrant失効時に実Effect拒否。UIで§enabled=false§が別Hostからの適法な起動を無条件に消さない | #988/#1124、#2499/#1174 |
| **T-16 新DSL Pluginの標準UI投影** | 新しい構文を追加しても標準inspect/Tree/Property/Diffが機能し、未知Effectはunknownと表示する | #988/#1124/#1824、#2502 |
| **T-17 Source編集の往復** | 一意Sourceへの編集はChange Proposalへ、展開/曖昧ノードはread-onlyとNavigationへ。別のScript正本を作らない | #988/#1124/#267、#2502 |
| **T-18 起動契機と現在のContext** | 同じActionを手動/イベント/予約で利用し、旧epoch・LIVE対象集合変更・実Load版差・失効Grantを再照合する | #2502/#1130、#2582/#2499/#943/#1174 |

**統合順**：先行する#1174実PG P0-aと#1129二Function P0-bは維持。P1では#2507のAction適用条件と#988のScript Inspectを#2502のHeadless CUIに接合し、T-14とT-15を既存#2598/#1174の対象整合試験へ委譲する。Studio標準UIとSource編集の実適合は後段のT-16/T-17で検証する。新しいMacro Executor、第二Authority、画面種別別専用DSLを始める契機にはしない。

## 10. 判断を残す点

- **業務Owner**：各Actionで「元の対象」「関係する対象」「全件の完了」をどのように定義するか。旧X-TEA `sales.order.submit`等の未受理マッピングは推測で補わない。
- **組織Owner**：固定集合／実行時集合、代理の時点、合議、実行版固定と作用範囲での更新許容、既発行Grantの変更時再判定、結果閲覧をどのPolicyで扱うか。
- **Host/Provider Owner**：現在Grantの取得・対象単位の照合・実Write fence・Raw/Secret/Egressの物理隔離をどこで保証するか。全Target同じ実装を必要条件にしない。
- **成果物/Delivery Owner**：承認対象と実ロード物・依存閉包の証拠、改版後の受理・公開・閲覧資格と取消をどう保持するか。
- **利用者体験Owner**：履歴Receipt／初回回答／現在値、実行/予約/失敗/納品の違いを、不要な専門用語なしに利用者へどう示すか。

### 現状の検証資格

**確認できたこと**：固定mainの既存型・RequestHash・通常Request生成・試験DDLのデータ解釈を静的に確認した。既存IssueにはT-1の具体的な未実行DB負例、Effect replayとResult再投影の未実行負例、EKI出力接続の静的所見がある。

**未実行・未採用**：本書のT-1〜18の実行、信頼済みIssuer/実Grant、利用者Macro実ロード、PostgreSQL負例、Linux/WindowsでのOS隔離、外部EKIの同一Attempt納品、複数Target受入、業務意味の正式受理。新しいCore kind、Grant/Receipt schema、専用Host、共通Workflow Engine、仕様やADRの採用を本書から確定しない。
