# SUI移管に向けたTEI拡張方針とSUI分割案

- 状態: 評価待ち（未採用）。実装・Reference変更の根拠にしない
- 日付: 2026-10-08
- 親Issue: [ISSUE-000052](../issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md)
- 関連: [SUI Reference Adoption設計基線](../SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md) / [ISSUE-000080](../issue/000080-extensible-plugin-capability-contract.md) / [ADR-0019](../adr/0019-authentication-capability-boundary.md)
- 減らす不確実性: RQ-008（Capability単位の置換）、RQ-019（権限とResource）、RQ-029（知識の再利用・移植）、RQ-005（ApplicationとDatabaseのN:M）、RQ-010（高密度な設計・運用環境）、RQ-031（Coreを特権化しない）、RQ-032（Plugin / DSLでの意味宣言）、RQ-023（一人で維持できる構造）

## 1. この文書の役割

SUI SensemakingをTEIへ最終的に移管する前提で、次の二つを検討します。

1. TEI側で、何を共通部品として用意し、何をCoreへ入れないか
2. SUI側を、移管しやすい単位へどう分けるか

進め方の正本は`ISSUE-000052`と`SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md`です。本書はそれらを置き換えず、「multi-tenant対応を基盤部品にする」という新しい論点と、SUI全体の分割案を足します。`encapsulation before translation`（翻訳の前にまず包む）と、現行`DocumentV1`を当面の正本とする方針は変えません。

## 2. 背景

### 2.1 SUIの現状

SUIはPython（FastAPI、SQLAlchemy）のbackend、TypeScript / Reactのfrontend、MCP serverからなります。backendは86ファイル・約2.65万行、frontendは`src`だけでテストを含め約9.9万行です。

このうち、SaaS対応（multi-tenant、認証edge、session、guest受け入れ、PDP連携）に属するモジュールだけで約6,800行あります。これは、SUIが「親和図を作る道具」であるほかに、「相互に信頼しない複数の顧客を同じ基盤に収容する基盤」の実装も内包していることを意味します。後者の設計判断はADR-0059、0061、0063、0072〜0074、0079、0080に残っています。

### 2.2 同じ問題が他のリポジトリにも現れている

multi-tenantに近い境界は、SUIだけの問題ではありません。

| Repository | 該当する記述 | 現在の扱い |
| --- | --- | --- |
| SUI | ADR-0059 D5〜D10。tenant解決、membership、複合外部キー、DB側のRLS、Data Plane / Admin / Platformの分離 | 実装済み。Python固有 |
| SEI Cognition | ADR-0030。Product Access Scopeをproduct-local概念とし、`denied`と`not evaluated`を分け、Selective DisclosureをProjectionで行う | 「Universal Contractはまだ追加しない。複数deploymentで共通性が実証されたら再検討」と明記 |
| EKI Fabric | tenantをまたぐArtifactGrantの禁止、Adapter conformanceでのtenant越境試験、同一digestによるcross-tenant認可統合の禁止 | 文書で規定済み |
| TEI | ADR-0019。ただし対象は「既存業務システムの利用者認証・認可」であり、tenantの概念はない | tenantに相当する境界は未定義 |

SEIは「複数の実例で共通性が出たら再検討する」と書いています。SUIの実装と、EKIの規定が、その実例に当たります。三つ目の実例が揃った今は、共通化を検討する時機です。ただし、共通化するのは**意味の契約と試験**であり、SUIのPythonコードそのものではありません。

## 3. 防ぎたい失敗

- **tenantを後付けのroleやUIの選択肢として扱う失敗。** データの所在と認可の評価範囲が決まらず、IDOR、一覧や検索からの存在漏えい、キャッシュ越境が起きます。
- **「Pluginを入れ忘れる」ことで分離が静かに無効になる失敗。** 隔離は、有効なときだけ働く機能では意味がありません。基盤部品として切り出すと、この失敗が新しく生じます。
- **TEI Coreへ`Tenant`を足してしまう失敗。** SEIも、tenantやworkspaceをUniversal semantic entityにしないと決めています。SUI一件の都合でCoreが膨らみます（RQ-031）。
- **SUI固有の語彙が基盤へ流入する失敗。** `Card`や`Island`、`human_reviewed`が共通部品の契約に混ざると、第二のApplicationが使えません。
- **移管のために二つの正本を作る失敗。** TEIで読めることと、TEIが正本になることを混同します（`docs/architecture.md`のauthority migration）。

## 4. 守る原則

1. **Coreに`Tenant`を足さない。** 隔離は、Semantic Extension、Database Providerの実現、Runtimeの検査という既存の拡張面の組み合わせで表します。
2. **部品の中心は試験にする。** 各言語の実装は入れ替わります。残るのは、契約と越境試験（conformance）です。
3. **宣言したのに守りが無い構成は動かさない。** 隔離の対象と宣言したデータがあるのに、対応する守りが解決できなければ、既定で動かさず`Unresolved`とします。暗黙のfallbackで隔離を外しません。
4. **四つの「権限」を混ぜない。** 次の四つは別物です。

   ```text
   Plugin / Runtime permission   実装コードが外部resourceへ触れてよいか（RQ-019）
   Application user authorization 誰が業務操作を実行できるか（ADR-0019）
   Product access scope          どの利用者集合のデータ領域に属するか（本書）
   Semantic Responsibility       判断責任の所在（SEI）
   ```

5. **業務要件とインフラを混ぜない。** 要件は「満たすべき性質」で書き、実現方法は`Scope Placement Binding`という設定項目で結ぶ。両者は別々に編集・受理でき、結びつきはエディタUIで見えるようにする（5.5）。
6. **SUIの安全不変条件を緩めない。** SafeMode既定ON、AI出力はproposal-only、`human_reviewed`は人間だけが設定、`SUI_LLM_PROVIDER=none`でも主要価値が成立する、という条件は、移管後も同じ強さで残します。

## 5. 必要能力と、TEI側の拡張方針

### 5.1 「基盤的なプラグイン」は、一つのPluginではなくBundleにする

結論から言うと、multi-tenant対応は共通部品として提供すべきです。ただし、形は「tenantを実装する一つのPlugin」ではありません。`docs/plugins.md`のBundle（互換性のあるPlugin requirementをまとめる配布上の便宜）として、性質の違う四つの層を束ねます。

```text
scope-isolation Bundle（仮称）
  ├─ L0  Semantic Extension     隔離の宣言と、整合のConstraint     実行codeなし
  ├─ L1  Database Provider実現   複合キー、DB側ガード                provider固有
  ├─ L2  Runtime Capability     request境界の解決・照合・認可・監査   Target固有
  └─ L3  Conformance profile    越境の負例マトリクス                 言語非依存

  + Scope Placement Binding    要件区分と実現のあいだの設定項目（5.5）
  + Placement Editor           上記を編集・診断するStudio画面（5.5）
```

`scope`という語を使い、`tenant`はその一つの名前にします。SEIのProduct Access Scopeや、将来のworkspace、組織、案件といった分割軸にも使えるようにするためです。

### 5.2 各層の責務

**L0 — 宣言（Semantic Extension）。** Applicationは「この`Data`は`scope`軸に属する」と宣言します。宣言から次を導けます。

- 識別子は`(scopeId, id)`の組になる
- 従属する`Data`は親と同じ`scopeId`を持つ（SUIのD7にある、`(tenantId, docId)`複合外部キーに相当）
- 外部へ出るキー（cursor、cache key、object storage key、冪等性キー、監査event）にも`scopeId`を含める

この整合は、既存の[ADR-0008](../adr/0008-persistence-foreign-key-identity.md)の外部キー識別の考え方と、`ISSUE-000077`のSemantic Interfaceによる評価で検査できる見込みです。実行codeを持たないdeclaration-onlyの拡張にできるため、RQ-032の要件とも整合します。

**L1 — DB側のガード（Database Provider）。** どのガードを使うかは業務要件ではなく、5.5のPlacement Bindingが選びます。 PostgreSQLではRLSとトランザクション局所のscope設定を生成します。SQLiteなど、DB側のガードを提供できないproviderは、共有スキーマ型では`unsupported`として宣言し、黙って通常動作へ落としません。SUI側の`tenant_db_guard.py`が先行例です。接続プールでscopeが漏れないことが、このproviderの適合条件です。

**L2 — request境界の検査（Runtime Capability）。** SUIのD8にある順序を、契約として固定します。

```text
1. 認証済みprincipalと、信頼できる入力からのScopeContextを確定する
2. membershipが有効であることを検証する
3. resourceをサーバー側で(scopeId, id)として参照する
4. principalのscopeとresourceのscopeが一致するか確認する（不一致ならPDPを呼ばず拒否）
5. SafeModeやread-onlyなど、ローカルの安全ガードを適用する
6. 外部の認可判断（PDP）へcapabilityを問い合わせる
7. 最小限の監査eventを残す（本文・タイトル・tokenは残さない）
```

結果は`allow` / `deny` / `not-evaluated`の三値にし、PDPの停止や無効な応答は`not-evaluated`として扱い、`deny`とは分けます。これは、SEI ADR-0030が定める区別と同じです。SUIのD8が採る「PDP不達なら拒否」は、この三値に対する既定動作の一つです。他のscopeの資源IDは存在を推測させない応答にします。

認証は、ADR-0019の`security.authenticate`と分けたまま、その後ろに`scope.resolve`を置きます。`scope.resolve`は、検証済みclaimか、境界で外部入力を除去した信頼できるhost対応からだけscopeを決めます。browserのheaderやqueryから決めません。

また、「active scopeは認証sessionごとに一つ」「scopeが切り替わったら古いcontextを拒否する」というADR-0061の仕組み（サーバーが発行する`tenantSessionVersion`の比較）は、**文脈の世代による遮断（fencing）**として一般化できます。TEIにはすでに、ownership / fencingの考え方（RQ-020）と、`expectedRevision`をアトミックに比較するcommit境界（`tools/reference-sui-interaction`のE1）があります。同じ原理なので、新しい概念を足さず、既存のfencingへ寄せられるかを先に評価します。

**L3 — 試験（Conformance）。** 成果の中心はここです。SUIのADR-0059の「Implementation gate」5が求める、次の越境の負例マトリクスを、部品の適合条件にします。

```text
同じdocIdを持つscope AとBについて、次の経路で越境が起きないこと
GET / PUT / list / search / count / export / share / import / MCP / webhook / job / audit / cache / agent credential
```

この試験は言語に依存しません。SUIのPython実装でも、将来のGo実装でも、同じ試験を通ることで同じ契約を満たしたと言えます。

### 5.3 Pluginにしてはいけないもの

- **Data Plane、Tenant Admin、Platform Control Planeの分離。** 面ごとにroute・audience・capabilityを分けるのはApplicationの設計であり、Plugin任せにしません。Platform運用者であることを理由に、文書本文の閲覧権限を暗黙に与えません（SUI D9）。
- **IdPの種類ごとの実装。** SUIはADR-0063でOIDC/JWTだけをアプリに実装し、SAMLなどはbrokerへ任せています。共通部品も同じ線引きにします。
- **roleやgroupの編集機能。** roleは外部PDPからの一時的な入力のままにします。
- **break-glass。** 時間制限、目的、承認、通知、監査が揃うまで標準にしません。

### 5.4 現在のTEIとの接点と前提

- `internal/plugin/model.go`の`CapabilityKind`は固定enumです。`scope-guard`を新しい種別として足すのは、`ISSUE-000080`が解消しようとしている構造そのものなので避けます。**このBundleは、`ISSUE-000080`のCapability Contractの仕組みが使えるようになってから、その上に載せます。** それまでは、L0とL3だけを先に評価対象にできます。
- Environmentは業務Applicationを配備先へ結び付ける単位です。scopeは、一つのEnvironmentの内側でデータ領域を分ける軸であり、Environmentとは別です。
- 守りの欠落を成功へ畳まない点は、`docs/plugins.md`の「暗黙のfallbackで意味を変えない」と同じ原則です。

### 5.5 業務要件とインフラを分け、設定項目でつなぐ

ここまでの説明は、隔離を「どう実現するか」に寄りすぎていました。tenantについて、**業務側が求めること**と、**インフラ側が提供すること**は別の事柄です。SUIのADR-0059 D7は、この二つを一つの文に混ぜています（「共有スキーマ型ではRLSを必須とする」）。「アプリとDBの両方で隔離を判定する」は業務・安全上の要求で、「PostgreSQLのRLSを使う」はその一つの実現方法です。混ぜたままだと、専用スキーマや専用DBで実現する顧客のために、要件の書き直しが必要になります。

TEIの既存の層（Canonical → Binding / Realization）に合わせて、次のように分けます。

```text
業務要件（Canonical / Application側）   何を満たす必要があるか。製品名・DB名を含まない
        ↑ satisfies（満たすかを検査する）
Scope Placement Binding（Environment側）  要件を、どの実現へ結ぶか   ← つなぐ設定項目
        ↓ realizes
インフラ（Provider / Target側）          共有スキーマ+RLS、専用スキーマ、専用DB、専用bucket等
```

**業務要件に書くもの（インフラ名を書かない）**

- 隔離の強さの区分。たとえば「アプリとデータ層の二重判定」「データ層が物理的に分かれる」のように、満たすべき性質で書く
- 他のscopeの存在を推測させない応答にするか
- データの所在（地域・法域）、保持期間、削除の完了条件
- exportやimportで、scopeの権限を持ち越さない規則
- 面（Data Plane、Tenant Admin、Platform Control Plane）ごとの閲覧境界

**Placement Bindingに書くもの（業務の意味を書かない）**

- 要件区分ごとの実現方法の選択（例: 標準区分は共有スキーマ+RLS、厳格区分は専用DB）
- 使用するDatabase Providerと、その適合条件（接続プールでscopeが漏れないこと）
- object storageのkey接頭辞やbucket、暗号鍵の参照（**実体や秘密は書かず**、Resource Brokerの参照だけを持つ）
- 地域などの配置先

**検査（satisfies）。** BindingがL3のconformanceを通り、かつ要件区分の下限以上の性質を提供するときだけ「満たす」と判定します。結果は`satisfied / unsatisfied / unsupported / unresolved`に分け、満たさない構成は動かしません（原則3）。`unsupported`（例: SQLiteで共有スキーマ区分を選んだ）を黙って通常動作へ落とさないことが要です。

**配置の二層。** 区分の定義とBindingは、Environmentのsourceとして作る設定項目です。一方、「どの顧客をどの区分へ置くか」は顧客の追加や移動で変わる運用データであり、Platform Control Planeの操作です。後者をsourceに書くと、顧客を増やすたびにAcceptanceが要ります。逆に前者を運用データにすると、変更の経緯と検証が残りません。この境界は未解決として10章に残します。

**エディタUI。** この分離を利用者に見せ、誤って混ぜないための入口として、Studio内に専用のeditor画面を設けます。StudioはDesign Onlyの段階なので、まず画面モックとReferenceの候補として扱います。

| 領域 | 内容 |
| --- | --- |
| 業務要件ペイン | 隔離区分、応答方針、保持・削除、面ごとの境界。インフラ名の入力欄を置かない |
| 配置ペイン | 区分ごとの実現方法、Provider、地域、Resource参照。業務の意味の編集欄を置かない |
| つなぐ表 | 要件区分 × 実現方法の対応表。各セルに`satisfied / unsatisfied / unsupported / unresolved`と理由を表示 |
| 診断 | 満たさない理由の遡り先（どのconformance項目が落ちたか、どのProviderが非対応か） |
| 確定 | 変更はcandidateとして作り、Preview → Accept / Rejectを`add-environment-binding`と同じShared Acceptance Boundaryで通す |

UIが守る規則は次のとおりです。

- UIは**第二の正本にならない**。保存の実体はEnvironment / Semanticのsourceであり、UIはそれを編集して読み戻す入口にすぎない（Studioの既存方針と同じ）
- 秘密（DSN、鍵、token）を入力・保存・表示しない。Resource参照の選択だけを扱う
- 「Bindingを保存できた」ことを「接続できる」「実行できる」と読み替えない。Permission grant、Resource binding、leaseは別に確認する（S7で`RUNTIME_AUTHORITY_NOT_GRANTED`を保持したのと同じ扱い）
- 要件ペインの変更と配置ペインの変更は、別のcandidateとして受理できる。片方の変更でもう片方が暗黙に変わらない
- AIが作った変更も、同じcandidateの経路を通る。AIの提案に承認権限を与えない

SUIの例では、現行のADR-0059 D7を「要件区分: アプリとデータ層の二重判定」と「配置: PostgreSQL共有スキーマ+RLS」に分解し直すことが、最初の具体的な書き換え例になります。

## 6. 具体例: SUIの`PUT /docs/{id}`を例にする

SUIの現行実装（`routes/docs.py`）は、`_authorize_request`でscopeとcapabilityを解決してから、`get_document_row(db, tenant=tenant, doc_id=...)`で資源を引きます。この「scopeを引数に含めない参照を許さない」形が、L0の「識別子は`(scopeId, id)`」に対応します。

一方、同じ関数には共通化する前に直すべき点が一つあります。`If-Match`は任意で、無い場合は後勝ちで上書きされます（`tools/reference-sui-interaction/README.md`の限界4と一致します）。隔離とは別の問題ですが、共通のcommit境界へ載せる前に、SUI側で「`If-Match`を必須にするか、無い場合の挙動を明示するか」を決める必要があります。これは**SUI側の先行修正候補**であり、TEI側の部品化を待つ必要はありません。

## 7. 辺縁にある共通部品の候補

ISSUE-000052の分類（A: 既存のTEI、B: 再利用できる拡張、C: SUI固有、D: TEIの製品上の不足候補）に合わせて、SUIの実装から見つけた候補を並べます。「他の消費者」は、リポジトリ内の記述から確認できたものだけを書きます。

| 候補 | SUIでの根拠 | 他の消費者 | 分類 | 判断 |
| --- | --- | --- | --- | --- |
| **scope isolation**（5章） | ADR-0059、0061、0063 | SEI ADR-0030、EKI | B | Bundleとして評価。最優先 |
| **Disclosure Projection**: 項目ごとに`preserve / redact / omit`を宣言し、export、共有、AI入力を同じ宣言から導く | ADR-0068、`safe_mode.ts`、DocumentV1の項目追加時の判断規則 | SEI（`withheld`と`unknown`の区別、Projectionによる開示） | B | 構造から導出する利点が最も大きい。次に評価 |
| **契約の二重定義の解消** | `test_ts_python_contract_drift.py`、`contract_drift_baseline.json`。Pydanticと TypeScriptと`schemas.md`の三重 | 他にもTEI全般 | A | 既存のTEI（Canonical定義からのProjection）で扱う。新部品は不要 |
| **Safe Artifact Intake**: zip、markdown、request bodyの上限と無害化 | `request_body_safety.py`、`markdown_sanitize.ts`、`artifact_integrity.ts` | EKIのUpload安全処理 | B候補 | 二つ目の実例を確認してから |
| **信頼されたoutbound HTTP**: redirect拒否、サイズ・時間の上限 | `trusted_http.py` | TEIのResource Broker境界 | A | 別Pluginにせず、Resource Brokerの利用規約として整理 |
| **監査event**: 本文を含めない固定metadata、scope別 | ADR-0081、`audit.py` | SEI Product Access audit | B | scope isolationのL2へ含める |
| **外部呼び出しの予算台帳**（reserve → call → settle） | ADR-0082 | EKIのCompute Grant / quota | 要検討 | `ISSUE-000084`が「quota・token capacityはTEIの責務外」としているため、TEIには入れず、EKI側または独立した小部品とする |
| **LLM provider、入力IR、model registry** | `llm/provider.py`、`llm_input_ir.py`、ADR-0065、0069 | SEI | C → SEIへ連携 | ISSUE-000052の方針どおり、TEIのeditor-intelligenceに押し込まずSEIと整理する |
| **提案・決定・適用の分離**（Shared Acceptance） | `proposal_decision_repository.py`、`merge_suggestion_apply.ts` | TEI ADR-0022 | A | 既存のTEI設計への圧力として扱う。Challenge C |
| **文書のrevisionと条件付き保存** | ADR-0070、0076。`If-Match`が任意 | TEI E1 bridge | B候補 | 6章の先行修正の後に評価 |
| **Web client側のscope衛生**: scope別のbrowser保存、切り替え時の破棄、CSRF | `app_browser_storage.ts`、`app_tenant_session.ts`、`csrf.ts` | `reference/ui-customization`のKit | B候補 | UI Composition Kitの一部として評価 |
| **空間canvas、polygon編集** | `canvas/`、`geometry/` | なし（第二の空間Applicationが未確認） | C | native assetのまま。第二の消費者が現れるまでPlugin化しない |
| **設定の登録簿**（`SUI_*`） | `runtime_parameter_registry.md`、`settings.py`（1,014行） | TEIのEnvironment | A | 既存のEnvironment bindingへ寄せる |
| **DB可搬性の登録簿** | ADR-0066、`database_portability.md` | TEIのDatabase Provider | A | SUIがTEIのproviderを使う側に回る。新規拡張しない |

「一つの消費者しか確認できないもの」は、部品化しません。ISSUE-000052の判断基準（SUIで実用価値があり、Coreの意味を増やさず、第二のApplicationで再利用できそう）に満たないためです。

## 8. SUI側の分割案

### 8.1 方針

- **まず同じリポジトリの中で境界を作る。** 別リポジトリへの切り出しは、cutoverの決定が出た境界から後で行います。先に切ると、境界が誤っていたときの戻しが大きくなります。
- **分割は所有の軸で行う。** ファイル数や行数で割りません。
- **依存は一方向にする。** 中核は他を知りません。方向は機械的な検査で固定します（TEI側の[`REPOSITORY_ARCHITECTURE_DEPENDENCY_GUARD_2026-10.md`](REPOSITORY_ARCHITECTURE_DEPENDENCY_GUARD_2026-10.md)が参考になるか、別途確認します）。
- **`SUI_LLM_PROVIDER=none`で成立する範囲を中核とする。** AI層は任意の依存として扱います。

### 8.2 分割した七つの単位

```text
        ┌────────────┐
        │ A 中核ドメイン │ ← 全てが依存する。何にも依存しない
        └─────┬──────┘
   ┌────┬────┴───┬────────┬────────┐
   B    C        D        E        F
 canvas AI層   SaaS基盤   交換    MCP
        (任意)  (切り出し)
                  └── G 永続化 ──┘
```

| 単位 | 内容 | 移管上の扱い |
| --- | --- | --- |
| **A 中核ドメイン** | `DocumentV1`、検証、canonical ops、merge / patch、KJの語彙、カード品質、SafeMode方針 | SUIのProduct Value。最後までSUIが正本。TEIへは契約の投影から始める |
| **B canvas** | `canvas/`、`geometry/`、polygon編集、描画 | C（native）。TEIはInteraction境界だけを提供 |
| **C AI層** | `llm/`、`*_ir.py`、`routes/ai*.py`、`attention_candidates.py`、model registry | 任意の依存。SEIとの境界を整理した後に判断 |
| **D SaaS基盤** | tenant、認証edge、session、guest、control plane、PDP連携。約6,800行 | **最優先の切り出し候補。** scope isolationの最初の参照実装（Python realization）にする |
| **E 交換** | import / export、bundle、zip、markdown、inquiry bundle、worker | Disclosure ProjectionとSafe Artifact Intakeの評価対象 |
| **F MCP** | `mcp/` | 外部agentとの境界。薄いadapterとして、Aの契約だけに依存 |
| **G 永続化** | repository、generation、content store、alembic | TEIのDatabase Providerと対応。A・Dの双方に依存される |

### 8.3 最初に分けるべき場所は、結合の集中点

Dの切り出しで難しいのは、Dそのものではなく、Dを呼ぶ側の混在です。tenantや認証のモジュールを参照している回数を数えると、集中点は次のとおりです。

```text
main.py 56 / settings.py 39 / routes/docs.py 37 / database_content_store.py 29
generation_repository.py 18 / proposal_decision_repository.py 13 / persistence_shapes.py 13
```

つまり、`main.py`（起動と配線）、`settings.py`（設定）、`routes/docs.py`（文書API）、各repositoryがA・G・Dをまたいでいます。この三つを先に整理しない限り、ディレクトリを分けても依存は断ち切れません。

提案する順序は次のとおりです。

1. **配線の分離。** `main.py`の起動時検証（SaaS profileで必須の条件）を、Dが提供する`validate_profile()`のような単一の入口へ寄せる
2. **`settings.py`の分割。** `SUI_*`をA / C / D / Eの設定へ分け、`runtime_parameter_registry.md`の項目に所有単位を付ける
3. **repositoryの引数にscopeを必須化。** `get_document_row(db, tenant=..., doc_id=...)`の形を全repositoryへ揃える（すでに多くが従っている）
4. **`routes/docs.py`の分割。** 認可の前処理（`_authorize_request`）をDの関数として切り出し、routeからはAの検証とGの保存だけを呼ぶ
5. **依存方向の検査を追加。** Aから D / C / E / F を参照するとCIが失敗するようにする

各段階で、既存の越境試験（`test_saas_e2e_tenant_isolation.py`、`test_docs_tenant_isolation.py`など）が通ることを条件にします。分割のたびに、隔離が静かに緩んでいないことを確認するためです。

### 8.4 TEIへ移す順序

```text
Stage 0（現行）  A の契約をread-onlyで投影（IslandHierarchyContractV1）
Stage 1          D の越境試験を、言語非依存のconformanceとしてTEI側へ書き出す
Stage 2          D を、その試験を通る「Python realization」として明示
Stage 3          E の Disclosure Projection を、A の項目宣言から導く参照実装
Stage 4          対象ごとのauthority移管（source owner、equivalence evidence、rollbackを明示）
```

Stage 1が先に来る点が要です。試験さえ移れば、実装がPythonのままでも、「同じ契約を満たしている」とTEI側から言えます。実装の書き換えを急ぐ必要がありません。

## 9. 最初に着手する場合の小さな一歩

実装に入る前に、費用の小さい順で次の三つを提案します。

1. **越境試験の棚卸し。**（2026-10-09に実施済み。結果は付録A） SUIの`test_*tenant*`、`test_saas_*`を、ADR-0059のgate 5のマトリクス（14経路）へ対応付け、欠けている経路を洗い出す。コードは書かない
2. **Disclosure Projectionの紙上評価。** `DocumentV1`の全項目について、現在の`redact / preserve / omit`の判断を表にし、一つの宣言から export、共有、AI入力が導けるかを手で確認する
3. **`If-Match`の扱いの決定。** SUI側の小さな変更として、必須化するか、後勝ちを明示するかを決める

いずれもTEIのCore、Plugin契約、`CapabilityKind`を変更しません。

## 10. 未解決と、この文書が主張しないこと

- **ISSUE-000080の結果に依存します。** Capability Contractの仕組みが無い間、L1・L2をPluginとして表現する方法は決まりません
- **scopeを階層（組織の下に案件）にできるか未検討です。** SUIは「active tenantは一つ」と決めています。階層を許すと、認可の検討が大きく変わります
- **共有スキーマ以外の隔離方式（専用スキーマ、専用DB）の扱いは、L1の適合条件として十分に書けていません。** SUIのD7は触れていますが、実例がありません。5.5の分離で要件側は変えずに済む見込みですが、実現側の適合条件は未検証です
- **「顧客をどの区分へ置くか」を、sourceと運用データのどちらに置くかが未定です。** 5.5では区分とBindingをsource、配置の割り当てを運用データと仮置きしましたが、移動時の履歴と検証の残し方は決めていません
- **要件区分の語彙（隔離の強さの段階）は固定していません。** SUI一件から作ると、SUIの実装に引きずられます。SEI・EKIの要求を見てから決めます
- **エディタUIは、画面モックとReferenceの候補です。** Studio自体がDesign Onlyであり、本書はUIの実装を約束しません
- **SEIとEKIの現行規定は、SUIの実装ほど詳細ではありません。** 共通の契約に載せられるかは、三者の担当が確認してからです
- **ここで述べた行数や参照回数は、2026-10-08時点の大まかな計測値です。** 分割の効果を測る指標（変更の波及箇所数）は、`ISSUE-000052`のchange amplificationで別途取ります
- **この文書は、SUIがTEIへ移管されること、scope isolationが標準化されること、どの分割も実施されることを、まだ保証しません。** 複数のApplicationで再現した後に、ADRとIssueへ分けて判断します
- 実在の機微資料を外部providerへ送らない、というSUIの不変条件は、この検討でも変更していません

## 11. 次に判断を仰ぎたい点

1. 業務要件・Placement Binding・エディタUIの分離（5.5）を、この方向で進めてよいか。進める場合、`plan/design/mockups/`に画面モックを作るか
2. 共通部品の名前と範囲を`scope-isolation`（tenantを一例とする）にしてよいか
3. SEI、EKI側の担当と、conformanceの共有場所（どのリポジトリに置くか）を決める必要があるか
4. SUI側の分割は、本書の八章の順序で`ROADMAP.md`へ載せてよいか（SUI側のAGENTS.mdでは、管理文書を増やさずROADMAPにTODOを書く規則です）

## 付録A. 越境試験の棚卸し（9章の一つ目の実施結果）

- 日付: 2026-10-09
- 方法: SUIのroute定義とテスト名を静的に読み、`grep`で対応を確認した。**テストは実行していない。** 「見当たらない」は、tenant・saas・guest系のテストファイルを`grep`した範囲での所見であり、「試験が存在しない」とは断定しない
- 対象: ADR-0059の実装gate 5が挙げる14経路

| 経路 | SUIの実装 | 越境試験の確認結果 |
| --- | --- | --- |
| GET（文書の取得） | `routes/docs.py` | あり。同じdoc IDで別tenantの内容を返さないこと（`test_saas_e2e_tenant_isolation.py`、`test_docs_tenant_isolation.py`）、PostgreSQLのRLS単独でも遮断すること（`test_tenant_isolation_postgres_rls.py`） |
| PUT（文書の更新） | 同上 | あり。解決済みtenantの行だけを更新すること |
| list（一覧） | `GET /docs`（tenantの文書metadataを返す） | **確認できず。** 上記の隔離試験の中に一覧を扱うものが見当たらない |
| search / count | 該当するrouteなし | 対象機能がない（N/A）。ただし追加時に試験を必須とする |
| export | 書き出しはclient側。サーバー側は`export-audit`でeventを記録 | **確認できず。** audit eventにtenantIdを含める単体試験（`test_audit.py`）はあるが、`export-audit` routeの越境試験は見当たらない |
| share | `document_access_admin`（可視性の更新） | RLSの試験（`test_document_access_rls_postgres.py`）はあるが、HTTP経路での越境は未確認 |
| import | client側で検証し、`PUT`へ流す | `PUT`の試験に含まれる。import時にtenantの権限を持ち越さない規則（D10）の試験は見当たらない |
| MCP | `SUI_RUNTIME_PROFILE=saas-multitenant`では`document_client.ts`が**起動を拒否する**（「tenantに束縛したMCP credentialが未実装」） | 試験以前に設計上は閉じている。解除条件が未定で、解除時に必要な試験は未整備 |
| webhook / job | 該当機能なし | N/A |
| audit | event envelopeに`tenantId`が必須。metadataで`tenantid`キーを予約済み | 単体では確認済み。DBに保存された監査を別tenantが読めないことの試験は確認できず |
| cache | server側に共有cacheはなし。browser保存は`tenantId`で名前空間化 | frontendの試験（`app_browser_storage.test.ts`にtenant言及20件）。切り替え時の破棄は`app_runtime_cleanup.ts`にtenant言及がなく、別の経路かを要確認 |
| rate limit | `rate_limit.py`。使用箇所は管理用のprovisioning routeだけで、キーはclient IP | 現状はtenantに属する経路に制限がなく、D7の「キーにtenantIdを含める」が働く場面がない（食い違いではない）。Data Planeへ制限を足す時にtenantをキーへ入れることを、conformanceの項目にする |
| agent credential | 上記MCPと同じ。未実装 | N/A（実装時に必須化） |
| 補足: 古いcontext | `test_tenant_session_precondition.py`、`test_saas_tenant_resource_gate_postgres.py` | あり。資源参照の前に止まること |

**読み取れること**

1. 中核の経路（GET / PUT、RLSの二重化、古いcontextの遮断）は、試験で固定されている
2. 一覧、export-audit、監査の読み取り、importの権限非継承は、HTTPレベルの試験が見当たらない。越境試験の第一候補
3. rate limitは管理用routeだけにあり、tenantとは無関係に動く。Data Planeへ足す時の規則として、conformanceに載せる
4. MCPは「閉じている」が、解除条件がない。共通部品のconformanceには「credentialをtenantと文書へ束縛する」項目が要る

この結果は、14経路を適合条件にするというL3の案が実在のコードに対して意味を持つことを示す。一方で、14経路のうち4経路はSUIにも機能がないため、共通のconformanceは「該当機能があれば必須」の形で書く必要がある。
