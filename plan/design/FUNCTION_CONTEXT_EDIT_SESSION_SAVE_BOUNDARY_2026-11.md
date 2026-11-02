# 複数Data Contextの編集セッションと保存境界：機能合成の横断設計（2026-11）

- **状態**：2026-11-02時点の設計・受入候補。正式な契約／Core型／DSL／UI仕様の採用、TEI製品実行、外部Host・DBでの検証、業務Ownerの意味受理はいずれも行っていない。
- **実証Owner**：[Issue #2597](https://github.com/hat47x/tei-platform/issues/2597)。
- **横断設計**：[Draft #2503](https://github.com/hat47x/tei-platform/pull/2503)／[#2506](https://github.com/hat47x/tei-platform/issues/2506)／[#2507](https://github.com/hat47x/tei-platform/issues/2507)。
- **既存の実行・観測Owner**：#1127/#2500/#2582/#2586/#1129/#1174/#2485/#1131/#1180。
- **開発方針**：#2492（AI-firstのDraft）、#2493（異質領域への圧力試験）。独立した小さなread-only実証から、CUI／標準Plugin・外部Host・GUIへ展開する。調査・要件・領域の上限は固定しない。

## 1. なぜこの課題が必要か

一つの業務Functionに、ヘッダ、受注明細、配送履歴、集計、承認状態を独立したData Contextとして構成できたとしても、それだけでは**複数の未保存変更をどう扱い、どこまでを一件の業務操作として保存するか**は決まらない。

例えば、親Aから親Bへ移動する直前に、Aの明細を編集していた場合、遅れて到着したAの再取得結果や親Bへの画面切替によって、Aへの未保存変更が消える／Bへ誤適用される危険がある。

また、同じ画面に存在する二つの明細集合が別Providerを使用する場合、利用者の「保存」という一回の操作が、実環境で必ず一つの原子的Transactionになるとは限らない。逆に、同一Providerが明示的な原子性を提供できる場合には、複数Contextの変更を一体のActionとして扱う有用性がある。

したがって、**Functionの構成形状、編集状態の所有、業務Actionの単位、実Transactionの保証、保存の結果、業務完了の判定**を、互いに関係しつつ独立した問いとして扱う。

## 2. 定義・配置・編集・実行・観測を区別する

| 層 | 識別対象・状態 | 正本・担当 |
| --- | --- | --- |
| Function/Pattern Definition | 機能の目的、必要なRelation/Role、受理されたAction、構成条件、バージョン | #1127/#2507/#998 |
| Function Placement/Instance | 定義の利用位置、Instance ID、親Anchor、入力Binding、選択epoch、ローカル設定 | #2507、今回の所有は編集セッションの接続 |
| Context Definition/Result | 集合の粒度・対象・Key/Relation、Query、Scope、取得状態、出所 | #2506/#2387 |
| **Editing Session** | Instance-localなDirty Change Set、対象Subject/Relation occurrence、元の版、暫定ID | **#2597の受入差分** |
| **Save Intent** | どのPatchをどの受理済みAction候補へ束ねるか、期待するEffectの範囲 | **#2597が組立条件を検証、実Actionは#2500/#1129** |
| Invocation/Effect | 操作主体・現在Grant、Tx可否、Provider、Operation ID、Effect、Commit | #951/#1174/#2485 |
| Evidence/Observation | Receipt、結果不明、実DB/外部Hostの独立観測、業務完了 | #1174/#1125/#1131/#1180 |

**重要な分離**：ModelerによるAccepted Sourceの編集は#1124、SHOでの文書編集はSHO/MMIの別契約。この文書は、**生成した業務Functionが利用する業務データの未保存編集**を対象とする。

### 2.1 同じ定義を二回利用した場合

同じFunction Patternを、画面の左右、タブ、二つの独立ウィンドウ、ヘッドレスConsumer二つへ配置できる。このときDefinitionと配布済みPlugin実装は共有し得る。

一方、Instance ID、Parent anchor、Selection epoch、Dirty Change Set、非同期応答、Operation requestの識別を共有してよいとは限らない。**同一Providerからの読み取り結果のキャッシュ再利用と、同一編集セッションの共有は別**である。Resultの共有を許す場合でも、Scope、Snapshot、Predicate、Schema、Binding、Disclosure/Authorityの条件を個別に確かめる。

UIの表示部品が非表示・未描画・破棄状態になっても、業務データの未保存変更を自動的に保存／廃棄する規則は導かれない。保持・明示破棄・確認後遷移等はFunctionの用途に応じて宣言・実証する。

### 2.2 Function間の接続の役割（型付き適用条件）

| 接続の目的 | 入出力が示す意味 | 実証時に守る条件 |
| --- | --- | --- |
| **Data Binding** | 親Subject/Key/Relation roleを子のQuery Inputへ | 多重度・Scope・親の版/選択epochが一致する |
| **Selection Binding** | Entity/Relation/Path occurrenceの選択を別Instanceへ | UI row indexと業務Entity IDを区別 |
| **Derived/Measure** | 完全性・粒度・時点付き集合から派生Claimへ | #2593の再集計条件、失われた根拠、NULL |
| **Change/Invalidation** | Source/Rule/Providerの変化から依存Claim再評価へ | #2582の局所影響・撤回・未知 |
| **Operation Intent** | 変更候補から受理済みBusiness Action入力へ | #2500の候補/曖昧/unsupportedと現在Grant |
| **Observation** | 実Operation/Receipt/独立読取りによる結果説明 | 同値の再読取りを当該Operationの因果証明にしない |
| **Presentation Navigation** | 展開・タブ移動・表示同期 | UIイベント自体から保存・権限付与を起こさない |

これらを全Pluginへ必須の公開Portとして実装する方針ではない。既存Semantic Relation/Context/Operator/Service Contractから必要な入出力を解決できるかを先に調べる。データ上の自己参照Graphに循環があっても、非同期通知の再帰ループや演算の代数的フィードバックを無条件に許可する理由にはならない。Clock/Step付きFeedbackや専門Solverが必要なケースは#988の演算契約へ渡す。

## 3. 未保存変更の同一性と更新結果

**設計比較票の概念的な項目（正式APIではない）**：

`editingSession / functionInstance / contextInstance / parentAnchor / selectionEpoch / subjectOrRelationOccurrence / baseVersion / proposedPatch / temporaryIdentity / dirtyProvenance / ruleRevision / status`

本票は別の保存正本の追加を意図しない。既存のUI/Operation/Contextにどこまで対応するか調べ、必要な最小限だけ共有Contractの候補へする。

### 3.1 親選択と再取得

親Aの行L1を100→120へ編集し、保存する前に親Bへ移動した場合：

- Aに属するPatchは、A／L1のSubject・Context・baseVersionを保持する。
- Bの同じ表示行番号にPatchを移動しない。
- Aについての旧epoch Query応答はBの表示結果に混入しない。
- Aを再表示する場合は、Aの取得結果と未保存Patchの関係（維持／Conflict／再評価）を説明する。
- 表示の切替え、再取得、取消要求、非表示、再マウントを同一の「変更廃棄」操作にしない。

### 3.2 保存前の状態は「値の一致」と別

Clientが表示している値、Providerから最後に読み込んだ値、ユーザーの変更案、送信済み要求、実施済みか不明なEffect、独立読取りで確認した値、Ownerが受理した業務完了の意味は異なる。

最低限、`CurrentRead / StagedPatch / SubmittedIntent / InvocationResult / IndependentObservation / BusinessCompletion`を比較欄として区別する。UI上のbusy/complete表示と業務結果の種類が一対一に対応するとは仮定しない。

## 4. 複数Data Contextの保存単位

### 4.1 一つの画面に「保存」がある場合

Page/Compositeにひとつの保存操作を提示できることと、実Providerが全Patchの原子性を保証できることは独立する。

| 契約候補 | 成立条件 | 誤成功の例 |
| --- | --- | --- |
| **Single Business Action + Single Tx** | 受理済みActionに複数Patchが対応し、実Hostが同一Tx・検査・Commit/rollbackを保証 | 複数Providerを一つの見かけ上のSaveへ渡してall-or-noneと表示 |
| **Multiple Actions + Independent Result** | 一つずつ実行することが業務上許され、各結果と残りの未実施/拒否/不明を区別 | 最初の成功を全体の業務完了とする |
| **Long-running / Coordinated Action** | 承認/外部処理/部分成功・補償/再試行等を既存Procedure/Serviceが明示 | UI保存要求に対するACKを業務完了として返す |
| **Unsupported / Unresolved** | 必要なTx/Authority/対象集合/Actionが実現先で不足 | 勝手に各Patchを独立保存して要件を弱める |

実行方式は**事前に決めた業務上の完了条件**に従って判断する。整合した最終状態に戻す責務と、ユーザーへの受付応答は別。すべてのUIへ単一のTransaction Managerを搭載する要件ではない。

### 4.2 暫定IDと多段編集

親Hを未保存の状態で追加し、その子D1を複数作成する場合、クライアント内の一時識別とProvider確定IDの対応が必要になる。一時IDのScopeと親子Relation/Role、保存後のキー割当て、部分成功・Conflict後に残すPatchを区別する。

暫定IDを正規の業務Keyとして外部に流さず、ID割当ての実Source/Adapterに従う。データベース管理のkeylessな詳細や一意Key未確認のケースは、既存#244/#1126の実装境界に戻す。

### 4.3 複数編集のRule/Validation

D1明細の値がD2の参照情報を必要とする場合、D2が部分取得であることを見過ごして保存を受理しない。一方、D1だけの閉じたClaimについてはD2未取得を理由に一律拒否する必要もない。

Rule実行に必要な関係のRole、集合、時点、値域、必要な完全集合性を照合する（#1706/#1708/#2593）。Ruleは現行Host側の実Execution/Authority境界へ引渡し、UIが別の業務Validator正本を所有する形を避ける。

## 5. 自作合成Fixtureと受入の正負対照（TEI統合実証はNOT RUN）

- **Function Definition**：複数Contextを表示できる同じ定義Fを二配置 I1/I2。
- **親**：A、B。Aの明細D1はL1=100、L2=100。独立集合D2はX1/X2/X3。
- **派生**：D1合計=200。D1×D2を単純JOINすると表示6行だが業務明細2件のまま。
- **開始時状態**：I1は親A、I2は親B。I1でL1の値を120へ変更し、別ContextのX1へ状態変更案を作る。いずれも未保存。
- **比較する保存方式**：同一Host/Providerの単一Txを提供する例と、別Providerの個別Actionしかない例を分ける。実在のTxやProvider実装があるとは仮定しない。

| ID | 正例で確認すること | 負例で検出する誤り |
| --- | --- | --- |
| ES-01 | 同一FのI1/I2に独立Session・epoch・dirty | Definition共有から編集バッファ共有 |
| ES-02 | 親A→B→AでAのPatch帰属が残る | AのPatchをBのrow0へ適用 |
| ES-03 | 遅いA応答でBを上書きせず、Aのdirtyを保持/Conflict診断 | refreshがdirtyを自動消去 |
| ES-04 | D1/D2のPatch・必要Scope/Validationを分離 | JOIN行6件を業務更新6件扱い |
| ES-05 | 単一Txの保証条件を明示し、Hostで確認 | 一つの保存ボタンだけで原子性を宣言 |
| ES-06 | D1成功/D2失敗をPartial/未完了として記録 | Response ACKで全体完了 |
| ES-07 | 一時親子ID・保存後ID対応、失敗時のPatch | 一時IDを外部主キーとして採用 |
| ES-08 | Relation occurrenceのunlinkとEntity deleteを区別 | 子表示削除から実体をdelete |
| ES-09 | 部分ページと業務対象集合を区別 | 表示ページのみで全件操作を確定 |
| ES-10 | hidden/unmountとdirty保存／破棄判断を区別 | 非表示になっただけでsilent commit/discard |
| ES-11 | 保存前に対象Version/Authority/Ruleを再照合 | 旧baseVersionの更新が後勝ちで成功 |
| ES-12 | CUIと別ConsumerでSubject/Intent/Resultを一致させる | Renderer固有の第二Business Action |
| ES-13 | 値不変・Source byte不変でもOwner確認の撤回を独立判定 | dirty無し→現在の業務確認有効と推論 |

### 5.1 正例同士も区別する

**保持できること**と**自動保存できること**は別。選択を切り替えてPatchを維持できても、そのPatchが現在の規則・Grant・対象状態に適用できるとは限らない。

**同じ値が観測されたこと**と**特定要求がコミットされたこと**は別。ACK消失後の独立readが同じ値を返しても、それが当該RequestによるEffectかを確認するにはOperation ID等の因果的なつながりが必要になる。

**一つのFunctionに含まれること**と**同じ資源所有／Transactionを持つこと**は別。複数Data Contextへの編集を一度に提示できるUIでも、実Operationの結果が部分成功であればそのまま伝える。

## 6. 既存Issueの責務境界

| 既存担当 | 再利用するもの | 今回の独自の問い |
| --- | --- | --- |
| #2507 | Function PatternのDefinition/Instance、複数配置、表現先 | 編集Patchと保存候補の状態が配置に正しく帰属するか |
| #2506 | Context Instance/Result Slice、粒度、Scope、選択epoch | RequeryとDirty状態を独立に管理できるか |
| #2582 | 依存に応じた失効/再取得/撤回 | Dirtyを失わず、別Instanceの変化を過剰伝播しないか |
| #2500 | ViewChange→typed Operation Intent候補 | 複数の候補をいつ業務Actionとして組めるか |
| #1129/#1174 | Rule/Owner/Effect/Transaction/Receipt/Recovery | 実Txの成立とBusiness CompletionをUI保存から分離できるか |
| #2485/#2564 | CRUD導出と期待状態・実Driverへの結合 | 一回の更新でも実経路に必要な保証があるか |
| #997/#1000 | 標準UI、案件拡張、Reactのa11y/IME/Save | 同じ業務意味が表示の再構成でも残るか |
| #1131/#1180/#2586 | Scenario/Fixture/独立Oracle、縦断CUI比較 | 複数ContextのPatch→Intent→結果を同版で接続できるか |
| #2493/#2018 | 領域Pressureの拡大と生産性計測 | 追加標準/専門Pluginで重複契約やAdapter修正が増えないか |

## 7. AI-first・Headlessの検証手順

**S0：コードと契約の現状確認（A0）**。最新main・#1127/#2485/#2506/#2500/#2582/#2507と既存Referenceを照合し、実際に存在するCLI/Driver能力と不足を説明する。ここでは未実装の`save-all`コマンドを発明しない。

**S1：read-only構成と合成入力（A1候補）**。二配置・二集合・親切替・epoch・dirtyを、既存Source/Context/typed candidateに関連付ける小さな孤立Referenceを比較する。ES-01〜04/09/10の正負を固定する。

**S2：保存意図の適用性（A1/A2候補）**。同一Txを要求するBusiness Actionと、個別操作に分かれるケースで適用診断を比較。複数Actionを勝手にまとめず、現実のProviderが提供するTransaction/Effectを既存#1174に渡す。ES-05〜08/11を確認。

**S3：独立観測と複数Consumer（A2候補）**。同一Scenario/Source/Ruleのexact版でCUI→実Go Driver→別Readを確認できる経路から始め、Web/Native/外部Hostを第二Consumerとして追加する。画面上の保存表示、独立DB Effect、Host実Grant、業務完了を別欄にする。

**S4：広域化**。組織の多重所属、工程Gantt、文書注記、GIS、長時間承認、IoT等へ編集対象と生存期間を変えた圧力を追加。探索とPoCは並列化するが、必要な業務意味は各DomainのOwnerへ戻す。

## 8. 設計採否を見極める指標

- 定義Fを再利用したInstance数に対して、重複するBusiness意味定義・Adapter数・明示Binding数。
- 親選択や再取得による誤対象Patch適用／dirty喪失／旧応答混入の件数。
- Action適用候補、未対応、現在Authority不足、実Commit unknown、部分成功の診断を正しく分けられた割合。
- 実Provider/Targetの違いによるTx可否・Effect/Receipt・再試行条件の差と、CUI/GUI間の再利用可能な契約。
- hidden/unmount/多重Instanceの不要な通信・メモリ・編集状態保有。性能を測定するまでは最適化方式を固定しない。
- 既存Semantic/Relation/Context/Operationの再利用率、Core Kind追加数、二重Source・新Registryの必要性、生成AIの設定・検証・再修正回数。

**設計の撤回・見直し条件**：Sessionを独立Core正本にしなければ成立しないと早計に断定した場合、画面別のBusiness Rulesが重複し始めた場合、Instance再利用のため編集状態を不当に共有し誤書込みが起きた場合、複数Providerの結果を根拠なく単一Tx保証にまとめた場合。

## 9. 未決の判断と次の受渡し

- **既定の未保存変更保持**：親切替・一時停止・画面終了・Renderer交換・外部Consumer再起動の各場合の既定動作。ユースケースと実Context/ActionのOwnerが決める。
- **Saveの構成**：複数Patchが一つのBusiness Actionである条件、別Actionとして段階実行する際の業務受理と回復、全部拒否と部分成功を表示する要件。
- **共有可能なCache**：複数Instanceが同じread resultを利用する条件、個別Scope/Time/Disclosure/Versionを誤共有しない方式。
- **暫定Identityの境界**：未保存親子の一体保存に対する実Provider API/キー生成・関係の正当性。
- **UI操作と特定業務確認の距離**：編集値が変わらなくても確認撤回・権限失効があり得る。確認・受理・実Commitの責務は既存#1125/#1174へ。

本書が記録するのは、TEIの機能合成によって実現したい**業務データ編集セッションの意味**と、それを既存契約で表現・実行できるかという検証方針である。実装済みの機能や本番利用の保証ではない。

## 10. 局所参照試験の記録（2026-11-02）

孤立したPython標準ライブラリの合成Referenceとして、`plan/fixtures/function-edit-session/verify_edit_sessions.py` を追加した。**TEI製品コード・正式なEditing Session API・実Provider/Driver/Transactionではない。**

- **実測**：隔離コンテナで `python3 verify_edit_sessions.py` を実行し、ES-01/02/03/04/05/06/09/10/11/13に対応する`unittest` 10件すべてPASS。`python3 -m py_compile`もPASS。
- **同一性**：実行ファイルの長さ6917バイト、SHA-256 `009faf31172e7993456bf7e563974f56b92a3ff5d7df9902eae94640b94336dc`、Adler32 `3645067705`。GitHub新規保存入力の長さ・Adler32は同値と確認した。実Target artifactやTEIの実行版保証に拡張しない。
- **未実施**：ES-07/08/12は局所試験未実装。ES-01〜13の正式TEI Context/Query/Action/Driver、実DB/Host、GUI、二Consumer、現在Grant、業務完了の検証とOwner意味受理はすべて**NOT RUN**。
- **境界**：局所`classify_save`は架空Provider識別とFixture中の原子性宣言を分類するだけで、実Transaction・現在権限・因果的Receiptを証明しない。実Effectは#1174/#2485、統合Scenarioは#1180/#2586の既存Ownerへ引き渡す。

## 11. 複合保存の依存関係と原子性保証を分離する（2026-11-02追補）

### 11.1 同じ「まとめて保存」で何が違うか

保存候補は次の**四つの独立した判断**を要求する。

1. **Target binding**：親・子・関係の変更Intentが、それぞれどのSubject/Relation occurrence/期待版に対応するか。対象同一性のOwnerは#2598/#2500。
2. **Dependency order**：未保存親のcreateが完了して実ID対応が得られてから、その子のcreateへ渡せるか。依存関係の循環・欠落と、同Instanceの一時ID衝突を検出する。
3. **Atomicity scope**：依存が正しい順序で実行されることと、全変更のall-or-none保証は別。実Host/Providerが明示した単一の業務Action・Tx/atomicity group・Effect境界を検証する（#1174）。
4. **Result correlation**：部分成功／未試行／不明／失敗の結果が**どの要求**に対応するかを区別する。受付ACK・依存エラー・実Commit/Receipt・独立観測・Business Completionを単一の成功フラグにしない。

外部仕様からの比較：

- [OData 4.01 JSON Batch §19.4](https://docs.oasis-open.org/odata/odata-json-format/v4.01/cos02/odata-json-format-v4.01-cos02.html)：依存要求（`dependsOn`）と原子性グループ（`atomicityGroup`）を区別し、グループ内はall-or-noneを要求する。個別応答のIDで対応を追跡する。これは**その仕様に適合するProviderの能力の例**であり、TEIの全Serviceで保証済みとは扱わない。
- [GraphQL Mutations](https://graphql.org/learn/mutations/)：上位Mutation fieldの逐次実行はDB Transactionではなく、一部成功・一部失敗が起こり得る。
- [RFC 6902 JSON Patch](https://www.rfc-editor.org/rfc/rfc6902)：`test`・`add`・`remove`・`replace`等と順序・対象JSON Pointerを定める。JSON文書内のPatch表現は、TEIの業務Entity/Relation ID・権限・Transactionの意味をそのまま定義しない。
- [PostgreSQL Transaction](https://www.postgresql.org/docs/current/tutorial-transactions.html)：単一DB TxのBEGIN/COMMIT/ROLLBACKやSavepointは有用な実現能力。複数Provider間に無条件で展開する保証にはならない。

### 11.2 自作の合成受入ケース（局所Reference）

追加の試料 `plan/fixtures/function-edit-session/verify_save_composition.py` は**プロバイダを呼び出さない純粋な合成処理**で、次の八件の正例・負例を検査する。

| Case | 検証対象 | 誤成功・未確認のまま保つ内容 |
| --- | --- | --- |
| ES-07 | 同じ`tmp-H`でもI1/I2で別の暫定親ID→受信済み物理ID | 親確認前、他Instanceの仮IDを子の確定Keyに流用 |
| ES-08 | `unlink_relation`は関係出現IDに対する操作 | 人物/取引先等のEntity deleteへ無音変換 |
| ES-12 | CUI/Web由来の同じ型付き変更は同じ候補表現へ | 実GUI・Action/Hostの一致は未確認 |
| ES-14 | 親create→子createの依存順序 | 順序が揃えばTx原子性も成立すると主張 |
| ES-15 | 依存先の欠落・依存循環の拒否 | 循環を自動的に順序付けして成功扱い |
| ES-16 | 単一Provider/異Providerの原子性宣言を別判定 | 逐次実行・一つのUIボタンを同一Txとみなす |
| ES-17 | 一件の結果が欠落すればunknown、余分な応答IDは拒否 | 部分的なReceiptを全件成功または全件失敗へ丸める |
| ES-18 | 局所で全要求の成功を観測できても業務完了とは別 | Business Rule/実Effect/独立Observationを省略 |

この試料の`atomicity_claim`は架空の**Provider一致＋明示グループ＋その能力のFixture宣言**を区別する教育用比較。返値`fixture_declared_only`は実Tx保証を意味しない。実Providerの詳細なTx参加・scope・session・Grant・commit fence・Rollbackは#1174/#2564で再確認する。上流のIntent/Request対象帰属は#2598、複数Action構成は#2597で担当を分ける。

### 11.3 局所試験結果と残余

- `python3 verify_save_composition.py`：Python標準ライブラリ`unittest` **8/8 PASS**。
- `python3 -m py_compile verify_save_composition.py`：PASS。
- 保存したファイルのBlob SHA-1は `49a3c3831e93f8aaa7b24b4ac80990ca53ab74c0`。隔離実行の`git hash-object`と一致し、6686バイト、SHA-256 `72981a2330a45327b1c2007f6c9640445046d77a9457b38518112f2be62f5f43`。
- 既存の`verify_edit_sessions.py` 10件と合わせて**独立合成Reference 18件**を実測。ただし各Referenceの対象と期待は独立しており、正式TEIの同一試行E2Eを示さない。
- **NOT RUN**：実TEI Function/Context/Action/Driver/Host/DB/Tx、真正のBatch API/GraphQL Adapter、Permission/Authority、GitHub exact-head CI、実Web/外部Consumer、Owner業務意味受理。Case自体の局所成功をこれらへ昇格させない。

### 11.4 追加の統合条件

既存#2598（単一Intent→Requestの対象束縛）の確認票に対して、#2597は`SaveGroup / Member Request / Dependency / Result`の**一対多の関連**を、既存のOperation/Receipt/Scenarioを参照して説明できるか検証する。各要求のID、対象Key/Role、期待版、実Provider/Hostの能力、Groupの原子性資格、個別結果と現在適用範囲を別列として渡す。正式Schemaを先に固定せず、既存Source/Contractで説明できた範囲と不足情報を測定する。

**必要な重要区別**：依存関係のDAGがある≠実効資源所有のDAGがある≠同一Txに参加する≠一つの業務として完了した。特にバッチAPIが処理順序を保証するだけの場合、一部成功の結果をそのまま保存・再確認する。

## 12. SQLite限定のTransaction実挙動比較（2026-11-02追補）

合成Plannerだけでは説明できない**実SQL実行における原子性と部分成功**を独立に観測するため、Python標準ライブラリのSQLite in-memoryによる`plan/fixtures/function-edit-session/verify_sqlite_save_boundaries.py`を追加した。

| Case | SQLiteで実際に試したこと | 実観測 |
| --- | --- | --- |
| SQL-T1 | `BEGIN`内でParent INSERT成功、Child FK違反後に`ROLLBACK` | Parent 0件／Child 0件 |
| SQL-T2 | 自動Commitの個別文でParent INSERT成功、次のChild FK違反 | Parent 1件／Child 0件（部分成功） |
| SQL-T3 | `BEGIN`内でParent INSERT後にSavepoint、Child INSERTをSavepointへRollback後に外側Commit | Parent 1件／Child 0件 |

**隔離SQLiteの`unittest` 3/3 PASS**、`py_compile` PASS。隔離実行ファイル2236バイト、SHA-256 `e80f3a852a33ae422a85acbdadc4d5b2baaaba9b8ed9650587d3f21ec0c19b0f`、Git blob SHA-1 `98c4a93cd07ee641276c0d132732635bea54ae7d`。

これにより、**同じParent→Childという機能依存**でも、実行方式によって全体Rollback／部分成功／部分Rollbackが成立し得ることを、局所SQLiteでは観測した。全般的なDB方言、Provider実API、PostgreSQL/Java/.NET、複数ノード、外部SaaS、TEI Driver/HostのTx保証へ外挿しない。外部仕様の[PostgreSQL Tx/SAVEPOINT説明](https://www.postgresql.org/docs/current/tutorial-transactions.html)は比較資料であり、このSQLite試験の本番移植を証明しない。

**合計で合成Python 18テスト＋局所SQLite実挙動3テスト＝21テストPASS。** 三つの独立Referenceであり、正式TEI Query→Data Context→Intent→Host→Effectの一体テスト件数は**0（NOT RUN）**。Sourceの意味受理や同一試行の実Provider/Owner Evidenceも未。今後、#1174/#2564の実DB資源所有・現行Authority・Commit fence、#2598の候補から実Requestへの対象束縛、#2586の単一Function縦断へ接続する。
