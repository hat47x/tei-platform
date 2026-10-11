# SUI移管に向けたTEI拡張方針とSUI分割案

- 状態: 評価待ち（未採用）。実装・Reference変更の根拠にしない
- 日付: 2026-10-08
- 更新: 2026-10-09に付録Dを追加し、付録Aの「MCP」「agent credential」の行を更新した。章立てと既存の番号は変えていない
- 親Issue: [ISSUE-000052](../issue/000052-sui-reference-adoption-and-plugin-surface-evaluation.md)
- 関連: [SUI Reference Adoption設計基線](SUI_REFERENCE_ADOPTION_DESIGN_2026-09.md) / [ISSUE-000080](../issue/000080-extensible-plugin-capability-contract.md) / [ADR-0019](../adr/0019-authentication-capability-boundary.md)
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

なお、手順2の「membershipが有効であること」と手順6の「外部の認可判断への問い合わせ」は、組織の構成員（member）を前提に書いています。外部のAI協働者（agent）のようにmembershipを持たない主体を同じ手順で扱うと、この二つは読み替えが要ります。SUIの実装から読み取れた読み替えの案は付録D.5に記録しましたが、評価待ちであり、この手順の変更としては採用していません。

認証は、ADR-0019の`security.authenticate`と分けたまま、その後ろに`scope.resolve`を置きます。`scope.resolve`は、検証済みclaimか、境界で外部入力を除去した信頼できるhost対応からだけscopeを決めます。browserのheaderやqueryから決めません。

また、「active scopeは認証sessionごとに一つ」「scopeが切り替わったら古いcontextを拒否する」というADR-0061の仕組み（サーバーが発行する`tenantSessionVersion`の比較）は、**文脈の世代による遮断（fencing）**として一般化できます。TEIにはすでに、ownership / fencingの考え方（RQ-020）と、`expectedRevision`をアトミックに比較するcommit境界（`tools/reference-sui-interaction`のE1）があります。同じ原理なので、新しい概念を足さず、既存のfencingへ寄せられるかを先に評価します。

**L3 — 試験（Conformance）。** 成果の中心はここです。SUIのADR-0059の「Implementation gate」5が求める、次の越境の負例マトリクスを、部品の適合条件にします。

```text
同じdocIdを持つscope AとBについて、次の経路で越境が起きないこと
GET / PUT / list / search / count / export / share / import / MCP / webhook / job / audit / cache / agent credential
```

この試験は言語に依存しません。SUIのPython実装でも、将来のGo実装でも、同じ試験を通ることで同じ契約を満たしたと言えます。

この行列の最後の項目（agent credential）は、SUIに実装が現れるまで、中身を具体的に書けませんでした。実装から読み取れた、行列の各経路で追加して問うべき合格条件の候補は付録D.5にまとめています。ただし、確認できたのはSUI一件の未merge実装で、IdPもテスト用に模したものです。そのため、L3の適合条件としては採用せず、評価待ちの候補として扱います。

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
2. **Disclosure Projectionの紙上評価。**（2026-10-09に実施済み。結果は付録B） `DocumentV1`の全項目について、現在の`redact / preserve / omit`の判断を表にし、一つの宣言から export、共有、AI入力が導けるかを手で確認する
3. **`If-Match`の扱いの決定。**（付録C。案Cで決定済み、SUI側で実装中） SUI側の小さな変更として、必須化するか、後勝ちを明示するかを決める

いずれもTEIのCore、Plugin契約、`CapabilityKind`を変更しません。

## 10. 未解決と、この文書が主張しないこと

- **ISSUE-000080の結果に依存します。** Capability Contractの仕組みが無い間、L1・L2をPluginとして表現する方法は決まりません
- **scopeを階層（組織の下に案件）にできるか未検討です。** SUIは「active tenantは一つ」と決めています。階層を許すと、認可の検討が大きく変わります
- **共有スキーマ以外の隔離方式（専用スキーマ、専用DB）の扱いは、L1の適合条件として十分に書けていません。** SUIのD7は触れていますが、実例がありません。5.5の分離で要件側は変えずに済む見込みですが、実現側の適合条件は未検証です
- **「顧客をどの区分へ置くか」を、sourceと運用データのどちらに置くかが未定です。** 5.5では区分とBindingをsource、配置の割り当てを運用データと仮置きしましたが、移動時の履歴と検証の残し方は決めていません
- **要件区分の語彙（隔離の強さの段階）は固定していません。** SUI一件から作ると、SUIの実装に引きずられます。SEI・EKIの要求を見てから決めます
- **エディタUIは、画面モックとReferenceの候補です。** Studio自体がDesign Onlyであり、本書はUIの実装を約束しません
- **SEIとEKIの現行規定は、SUIの実装ほど詳細ではありません。** 共通の契約に載せられるかは、三者の担当が確認してからです
- **付録Dの項目（AG-1〜AG-12）は、SUI一件の未merge実装に基づく候補です。** 第二のApplicationで再現するまで標準化しません。確認に使ったIdPはテスト用に模したもので、実在のIdPにおける`sub`の意味、PostgreSQLの本番相当の構成、攻撃者の視点での審査は、まだ確認していません。詳細は付録D.6とD.7に書きました
- **ここで述べた行数や参照回数は、2026-10-08時点の大まかな計測値です。** 分割の効果を測る指標（変更の波及箇所数）は、`ISSUE-000052`のchange amplificationで別途取ります
- **この文書は、SUIがTEIへ移管されること、scope isolationが標準化されること、どの分割も実施されることを、まだ保証しません。** 複数のApplicationで再現した後に、ADRとIssueへ分けて判断します
- 実在の機微資料を外部providerへ送らない、というSUIの不変条件は、この検討でも変更していません

## 11. 次に判断を仰ぎたい点

1. 業務要件・Placement Binding・エディタUIの分離（5.5）を、この方向で進めてよいか。進める場合、`plan/design/mockups/`に画面モックを作るか
2. 共通部品の名前と範囲を`scope-isolation`（tenantを一例とする）にしてよいか
3. SEI、EKI側の担当と、conformanceの共有場所（どのリポジトリに置くか）を決める必要があるか
4. SUI側の分割は、本書の八章の順序で`ROADMAP.md`へ載せてよいか（SUI側のAGENTS.mdでは、管理文書を増やさずROADMAPにTODOを書く規則です）
5. 付録D.4のAG-12（tenant確定前の索引が持つ全体での一意制約が、他tenantの対応付けの存在を伝えうること）を、TEI側の部品化を待たずに、SUI側で先に判断し、SUIの`ROADMAP.md`へ載せてよいか

## 付録A. 越境試験の棚卸し（9章の一つ目の実施結果）

- 日付: 2026-10-09
- 方法: SUIのroute定義とテスト名を静的に読み、`grep`で対応を確認した。**テストは実行していない。** 「見当たらない」は、tenant・saas・guest系のテストファイルを`grep`した範囲での所見であり、「試験が存在しない」とは断定しない
- 対象: ADR-0059の実装gate 5が挙げる14経路
- 更新（2026-10-09）: 「MCP」と「agent credential」の2行、および「読み取れること」の4を、SUIの未merge branchの実装に合わせて書き換えた。この更新分だけは、branch先端でtestを実行して確かめている（方法と限界は付録D）。ほかの行は初版のままで、試験は実行していない。SUIのADR番号に衝突があるので、付録D冒頭の注も参照のこと

| 経路 | SUIの実装 | 越境試験の確認結果 |
| --- | --- | --- |
| GET（文書の取得） | `routes/docs.py` | あり。同じdoc IDで別tenantの内容を返さないこと（`test_saas_e2e_tenant_isolation.py`、`test_docs_tenant_isolation.py`）、PostgreSQLのRLS単独でも遮断すること（`test_tenant_isolation_postgres_rls.py`） |
| PUT（文書の更新） | 同上 | あり。解決済みtenantの行だけを更新すること |
| list（一覧） | `GET /docs`（tenantの文書metadataを返す） | **確認できず。** 上記の隔離試験の中に一覧を扱うものが見当たらない |
| search / count | 該当するrouteなし | 対象機能がない（N/A）。ただし追加時に試験を必須とする |
| export | 書き出しはclient側。サーバー側は`export-audit`でeventを記録 | **確認できず。** audit eventにtenantIdを含める単体試験（`test_audit.py`）はあるが、`export-audit` routeの越境試験は見当たらない |
| share | `document_access_admin`（可視性の更新） | RLSの試験（`test_document_access_rls_postgres.py`）はあるが、HTTP経路での越境は未確認 |
| import | client側で検証し、`PUT`へ流す | `PUT`の試験に含まれる。import時にtenantの権限を持ち越さない規則（D10）の試験は見当たらない |
| MCP | **main**では、`SUI_RUNTIME_PROFILE=saas-multitenant`で`document_client.ts`が**起動を拒否する**（「tenantに束縛したMCP credentialが未実装」）。**未mergeのbranch**（`feat/agent-oauth-token-exchange`の先端）では、stdioは`SUI_MCP_AGENT_CREDENTIAL`、HTTPはトークン交換の設定（`SUI_MCP_TOKEN_EXCHANGE_*`）が揃うときだけ起動でき、静的な`SUI_API_KEY`は拒否する（ADR-0093、ADR-0094） | **branch上では試験で固定されている。** backendのHTTP経路は`test_agent_credential_http.py`と`test_agent_oauth_http.py`、MCP側のvitestは`document_client.test.ts`、`token_exchange.test.ts`、`http_server_exchange.test.ts`。起動拒否を解除する条件は、ADR-0093に9項目で書かれている。**限界**: SUI一件で、PR #3214はDraft、残る2 branchは独自のPRを持たず積まれている。トークン交換の確認は、テスト用に模したIdPに限る。MCPとbackendを実際につないだ試験は見当たらない |
| webhook / job | 該当機能なし | N/A |
| audit | event envelopeに`tenantId`が必須。metadataで`tenantid`キーを予約済み | 単体では確認済み。DBに保存された監査を別tenantが読めないことの試験は確認できず |
| cache | server側に共有cacheはなし。browser保存は`tenantId`で名前空間化 | frontendの試験（`app_browser_storage.test.ts`にtenant言及20件）。切り替え時の破棄は`app_runtime_cleanup.ts`にtenant言及がなく、別の経路かを要確認 |
| rate limit | `rate_limit.py`。使用箇所は管理用のprovisioning routeだけで、キーはclient IP | 現状はtenantに属する経路に制限がなく、D7の「キーにtenantIdを含める」が働く場面がない（食い違いではない）。Data Planeへ制限を足す時にtenantをキーへ入れることを、conformanceの項目にする |
| agent credential | **未mergeのbranchで実装済み。** membershipを持たない別種の主体として、不透明な資格情報（ADR-0093）と、トークン交換後のOAuth主体の対応付け（ADR-0094）を扱う。Tenant Adminによる登録・一覧・失効のAPI（`/tenant-admin/agent-credentials`）がある。agentが通れるrouteは、許可リストの4つだけ | **あり（branch上）。** 同じdocIdを持つ別tenantの遮断、付与外・不存在・他tenantを同じ404にすること、失効・期限切れ・付与の取り消しが次の要求で効くこと、書き込み・export・archiveの403、未確認の本文を返さないこと、監査の主体（`test_agent_credential_http.py`）。OAuth経路（`test_agent_oauth_http.py`）、管理API（`test_agent_credential_admin_routes.py`）、agent用の表のFORCE RLS（`test_agent_credential_postgres_rls.py`）。**限界**: 付録D.7。PostgreSQLの試験は、実行した作業者の申告では使い捨てのPostgreSQL 16 clusterで通ったもので、本書は再実行しておらず、記録も確認していない |
| 補足: 古いcontext | `test_tenant_session_precondition.py`、`test_saas_tenant_resource_gate_postgres.py` | あり。資源参照の前に止まること |

**読み取れること**

1. 中核の経路（GET / PUT、RLSの二重化、古いcontextの遮断）は、試験で固定されている
2. 一覧、export-audit、監査の読み取り、importの権限非継承は、HTTPレベルの試験が見当たらない。越境試験の第一候補
3. rate limitは管理用routeだけにあり、tenantとは無関係に動く。Data Planeへ足す時の規則として、conformanceに載せる
4. （初版）MCPは「閉じている」が、解除条件がない。共通部品のconformanceには「credentialをtenantと文書へ束縛する」項目が要る。（2026-10-09更新）SUI側が解除条件を9項目で定め、branch上で試験として固定した。初版で求めた項目は、付録DのAG-1〜AG-12として具体化した。ただし、いずれも評価待ちの候補で、SUI一件にとどまる

この結果は、14経路を適合条件にするというL3の案が実在のコードに対して意味を持つことを示す。一方で、14経路のうち4経路はSUIにも機能がないため、共通のconformanceは「該当機能があれば必須」の形で書く必要がある。

## 付録B. Disclosure Projectionの紙上評価

- 日付: 2026-10-09
- 方法: SUIのfrontend（`inquiry_bundle_safe_mode.ts`、`policy/safe_mode.ts`、`export/`）、backend（`llm_input_ir.py`）、`schemas.md`を読み、項目ごとの開示判断がどこに書かれているかを数えた。コードは変更していない
- 問い: 項目ごとの`preserve / redact / omit`を一か所に宣言し、export・共有・AI入力をそこから導けるか

### B.1 現状

SUIの開示判断は、種類の違う三つの形で書かれています。

| 形 | 場所 | 規模 | 網羅性の担保 |
| --- | --- | --- | --- |
| **項目表** | `inquiry_bundle_safe_mode.ts`の`*_FIELDS` | 44表・282項目。内訳は`preserve` 174、`rebuild` 54、`redact` 45、`omit` 9 | あり。`satisfies Record<keyof T, FieldPolicy>`により、型に項目が増えると`typecheck`が失敗する |
| **実体×文脈の判定** | `SafeModePolicy.canExposeText(entity, context, safeMode)` | 実体5種（`card.text`、`island.summary`、`relation.summary`、`trace.text`、`diagnostics.detail`）。`export/`などに計27箇所の呼び出し | なし。実体の種類を足したとき、呼び出し側に判断が散る |
| **別の軸の判定** | backend `llm_input_ir.py`の`_enforce_safe_mode`、routeの`_reject_unreviewed_*`（`textReviewed`、`titleReviewed`） | 「SafeMode」ではなく「人がレビュー済みか」という別の軸 | Python側。TS側の表と自動では突き合わせていない |

もう一つ、`schemas.md`の文章にも方針が書かれています（例: `affiliations`は`omit`）。

### B.2 読み取れること

1. **「宣言し、網羅性を型で縛る」仕組みは、SUIですでに機能している。** 直近の修正（`affiliations`の追加で`typecheck`が失敗し、方針表に`omit`を明示した）が実例で、項目の追加時に判断を忘れられない。TEIの`Attribute`へ`disclosure`注釈を付け、「宣言のないAttributeはUnresolved」とValidationで扱う形は、この仕組みを一般化したものに当たる
2. **ただし、この表を使うのは投影の一経路（inquiry bundleのexportとshare）だけで、他の出力は別の判定を使っている。** `narrative_export.ts`、`abstract_map_export.ts`、`agent_task_export.ts`、`context_bundle_projection.ts`は実体単位の判定で、項目表を参照しない。つまり「一つの宣言から全出力を導く」はまだ実現していない
3. **開示は項目だけでなく文脈で変わる。** `canExposeText`は`ui / share / review-pack / diff`で結果が違い（たとえば`diff`では`trace.text`を出すが`card.text`は出さない）、現在の項目表は文脈を一つしか持たない。宣言は「項目 × 開示先（audience）」の表になる必要がある
4. **`rebuild`は葉の方針ではなく「中へ入って再帰する」という意味に近い。** 54項目がこれに当たる。TEIの宣言では、leafの`preserve / redact / omit`と、入れ子の構造を辿る`derive`を区別した方が正確になる
5. **SUI固有の軸がある。** 「人がレビュー済みか」（`human_reviewed`）はSafeModeと別軸で、共通部品へ入れるとSUIの語彙が流入する。共通部品は「開示先ごとの方針」までを扱い、レビュー状態は条件として外から与える形が妥当
6. **`omit`は「無い」と区別できない。** SEI ADR-0030は、開示しないこと（`withheld`）と、存在しない・不明なこと（`unknown`）を分けます。SUIの`redact`は`[REDACTED] (len:N)`と秘匿の事実を残しますが、`omit`は項目ごと消えるため、受け手には「無い」ように見えます。共通部品では、`omit`の結果に「項目が伏せられた」ことを示す印を持たせるかを決める必要がある（SUIの現行動作は変えない）
7. **`redact`の副作用に注意が要る。** `[REDACTED] (len:N)`は文字数を、`summarizeForSafeMode`は短いハッシュを出す。`schemas.md`の別の節は、相関に使えるため短縮ハッシュのプレースホルダを避けると書いている。両者は用途が違う可能性があるが、共通部品の`redact`の定義では、何を残すか（長さ、ハッシュ、印のみ）を契約に含める必要がある

### B.3 判断

- 構造から導出する利点は実在する。ただし証拠は**SUI一件**で、SEIは概念の合意（`withheld`とProjection）にとどまる。**標準化は時期尚早**で、ISSUE-000052の基準どおり、第二のApplicationで同じ形が再現するのを待つ
- 次の一歩として、TEIのReference Workspaceにある`order_entry`などで、`disclosure`注釈を付けた場合に、項目の網羅性検査が既存のValidationの枠で書けるかを小さく試す。TEI Coreは変更しない
- SUI側の小さな改善候補（TEI移管と独立）: 実体単位の判定（`canExposeText`）の呼び出し27箇所と項目表の関係を、どちらかが他方から導かれる形に寄せられるか

## 付録C. `PUT /docs/{id}`の`If-Match`の扱い

- 日付: 2026-10-09
- 決定者: SUIの保守担当者。**2026-10-09に案C（`saas-multitenant`だけ必須化）が選ばれた。** 実装とADRはSUI側のbranch `fix/put-doc-precondition-saas`（`ADR-0092`）。以下の本文は、決定前に書いた選択肢と推奨の記録として残す

### C.1 現状

- `If-Match`が無い場合は後勝ち（LWW）。これは`api.md`に仕様として明記されている。指定時は`ETag`が一致したときだけ更新する
- `If-Match: *`は更新で受理され、競合検査を迂回できる
- `If-Match`があって文書が無い場合は409を返す
- 同じバックエンドの他の更新経路は、すでに**必須化済み**。`inquiry_bundles`のPUT/DELETE、`document_access`の更新、model allowlistの更新は、欠落を`428 Precondition Required`で拒否する（model allowlistは「意図的な破壊的変更・移行期間なし」と記録されている）
- frontend（`api/client.ts`）は、既存文書の保存で`If-Match`を送っている。新規作成の経路は別

つまり、`PUT /docs/{id}`だけが、同じバックエンドの中で例外になっています。

### C.2 選択肢

| 案 | 内容 | 利点 | 費用・危険 |
| --- | --- | --- | --- |
| A | 現状維持（LWWを明記したまま） | 変更なし | 共有SaaSで同時編集が静かに上書きされる。TEIの`expectedRevision`比較（E1 bridge）に載せられない |
| B | すべてのプロファイルで必須化（欠落は428、更新で`*`は拒否、作成は`If-None-Match: *`） | 他の経路と一貫する。TEIへ移す境界が単純になる | 外部クライアントやスクリプトが壊れる。新規作成経路の変更が要る。single-tenantの互換（ADR-0059 D10）を崩す |
| **C（推奨）** | `saas-multitenant`プロファイルだけ必須化。`single-tenant`はLWWを維持し、挙動をプロファイルとして明記する。どちらも更新での`*`は拒否する案を別途検討 | 共有SaaSの安全を先に確保し、既存利用者を壊さない。ADR-0059 D10の「互換と移行を分ける」に沿う | 二つの挙動が並ぶ。プロファイル間の差を試験とapi.mdで固定する必要がある |

### C.3 推奨の理由

SaaSの越境境界と同じく、「共有環境でだけ必要な保証」はプロファイルで分けるのがSUIの既存の方針です。単一利用者の端末で`If-Match`を必須にしても守るものは少なく、外部クライアントを壊す費用だけが残ります。一方、共有SaaSでは、同じ文書を複数人が編集する前提のため、LWWは「静かな上書き」を許します。

必須化の実装は、既存の`428 document_access_precondition_required`と同じ形状のコード（例: `document_precondition_required`）で足せる見込みです。新規作成の経路（`If-None-Match: *`）の扱いは、`inquiry_bundles`の既存契約に合わせられます。

なお、これは`api.md`のAPI契約の変更であり、SUIのAGENTS.mdに従って契約・実装・関連テストを同時に更新する必要があります。長期的な設計判断にあたるため、実施する場合はADRに「価値への寄与」とともに記録することになります。

## 付録D. agent資格情報とMCPが示したこと

- 日付: 2026-10-09
- 状態: 評価待ち（未採用）。本付録の項目（AG-1〜AG-12）は、scope-isolation BundleのL3へ加える**候補**であり、採用済みの要件でもCanonicalな意味でもない
- 役割: SUIに現れたagent資格情報の実装を、TEI側から読み直した所見の記録。実装の正本はSUI側のADRと`02_Architecture/api.md`の2.15で、本付録はその第二正本ではない
- 対象: SUIの未merge branch三つ。`feat/agent-credential-mcp`（Draft PR #3214）、`feat/agent-credential-admin-api`、`feat/agent-oauth-token-exchange`。後の二つは前のbranchの上に積まれ、独自のPRを持たない。先端のcommitは`7f02f17`
- 方法: コード、ADR、`api.md`を読み、関連するtestをbranch先端で実行した。
  - 実行したもの: backendの関連8ファイルをSQLite上で実行し、87 passed・6 skippedだった（skippedはPostgreSQLを要する試験）。MCPのvitest 3ファイルは38 passedだった。さらに、台帳の試験が分類漏れを実際に検出するかを、管理APIを追加したcommit（`586d647`）で実行して確かめた（AG-11）
  - 実行していないもの: PostgreSQLを要する試験（`test_agent_credential_postgres_rls.py`など）は、実行環境が無く再実行していない。実行した作業者の申告では、使い捨てのPostgreSQL 16 clusterで通っているが、本書はその記録を確認していない
- 減らす不確実性: RQ-019（権限とResource）、RQ-029（知識の再利用・移植）。SUI一件の実装から、どこまでをconformance項目として一般化してよいかを見分ける
- 注（ADR番号）: 本付録の作成時点で、SUIの`main`には別の`ADR-0094`（`ADR-0094-oversized-document-review-path.md`。送信上限を超えた文書の確認導線）が入っており、branch上の`ADR-0094-agent-oauth-token-exchange-for-mcp-http.md`と番号が衝突している。本書の「ADR-0094」は、断りがない限り後者（agent向けのトークン交換）を指す。SUI側で採番が変わったときは、本書の参照を追随させる

### D.1 背景

付録Aの時点では、MCPは`saas-multitenant`で起動を拒否され、agent credentialは未実装でした。MCP（Model Context Protocol）は、AIエージェントが外部のデータやツールを呼び出すための接続規約です。agentは人間の利用者ではなく、tenantのmembership（その組織の構成員であること）も持ちません。そのため、「利用者を認証し、membershipを確認し、認可する」というこれまでの順序には、そのままでは載りません。

その後、この空白を埋める実装が、SUIに三つのbranchとして現れました。

- `feat/agent-credential-mcp`（ADR-0093）: Tenant Adminが登録する、tenantと文書に束縛した**不透明な資格情報**（opaque token。中身に意味を持たせず、サーバー側の表を引いて初めて意味が決まる文字列）。許可リスト方式のrouteと文書view。MCPのstdio通信
- `feat/agent-credential-admin-api`: Tenant Adminによる登録・一覧・失効のAPI
- `feat/agent-oauth-token-exchange`（ADR-0094）: MCPのHTTP通信が**トークン交換**（RFC 8693。IdP、つまりトークンを発行する認証サービスに、ある宛先向けのトークンを渡して、別の宛先向けの短命なトークンに引き換えてもらう仕組み）を行う。backendは、交換後のトークンの`(identity_provider_id, sub)`（IdPの識別子と、トークンが表す主体の識別子）からagentを引く

付録Aの「読み取れること」の4は、共通部品のconformanceに「credentialをtenantと文書へ束縛する」項目が要ると書いていました。本付録は、この項目を実装に照らして具体化し、どこまで一般化できるかを分けて記録します。

### D.2 防ぎたい失敗

- **トークンに書かれたtenantを信じる失敗。** claim（トークンに書かれた主張）は権限ではなく要求にすぎません。信じると、IdPの設定ミスやclaimの付け替えが、そのままtenant越境になります。
- **静的なAPI keyでtenantを済ませる失敗。** keyは、どのtenantの誰かを証明しません。共有環境でkeyが一つ漏れると、影響が全tenantに及びます。
- **失効しても読み続けられる失敗。** 資格情報の状態をキャッシュしたり、トークンの有効期限だけに頼ったりすると、管理者が失効させた後もしばらく読めてしまいます。
- **拒否の理由が応答の差として観測できる失敗。** 失効、期限切れ、未知、他tenantの文書が別々の応答になると、資格情報や文書の存在を探る手掛かりになります。
- **拒否リスト方式で書いた失敗。** 閉じたいrouteを列挙する方式では、あとで本文由来の派生データを返すrouteが追加されたとき、agentに対して静かに開きます。
- **本文を含む新しい項目が、agentへそのまま出る失敗。** 隠す項目を列挙する方式では、`DocumentV1`に項目が増えるたびに、未確認の本文が出る危険が生じます。
- **受け取ったトークンを、下流へそのまま渡す失敗。** MCP宛てのトークンをbackendへ転送すると、トークンの宛先という境界が消えます。MCPが侵害されたときの被害も、backendまで広がります。
- **監査の主体を、呼び出し側の申告に任せる失敗。** headerの値で記録すると、agentが他人の名前で読んだように見せかけられます。
- **新しい表やrouteの分類を忘れる失敗。** 3章に書いた「Pluginを入れ忘れる」ことで分離が静かに無効になる失敗と同じ形が、表やrouteを足すときにも起きます。

### D.3 具体例: tenant Aのagentが`shared-doc`を読む

付録Aと同じく、同じdocId（`shared-doc`）を持つtenant AとBを例にします。tenant Aのagent（`agent-a`）が`GET /docs/shared-doc`を呼ぶと、SUIのbackendは次の順に処理します。

```text
1  Sui-Sensemaking-Agent-Credential: suiag_... を受け取る   （通常の利用者のAuthorizationとは別のheader）
2  鍵付きハッシュで agent_credential_index を引く            ← tenantが決まる前。RLSなし
     得るもの: (tenant A, agent-a, 資格情報の版) だけ
3  tenant Aのデータ領域を設定する
4  agent_credentials から 状態・期限・版 を読み直す          ← RLSあり。毎要求
5  (メソッド, route) が許可リストに一致するか                ← 一致しなければ 403
6  agent_document_grants に shared-doc の付与があるか        ← 無ければ「存在しない」と同じ 404
7  許可リスト方式の文書viewを返す                            ← 構造と、人が確認した本文だけ
8  監査eventの主体を agent:agent-a にする                    ← x-actor-ref は見ない
```

この図からは、次の三点が読み取れます。

- 2は「どのtenantか」を引く段階なので、tenantで絞れません。そのため、RLS（Row Level Security。DBが行ごとにアクセスを制限する機能）を掛けられず、3以降で状態・期限・付与を、RLS付きの表から読み直します。
- 5と6は、この順に検査します。routeを閉じる`403`は、文書があるかどうかに依存しません。
- tenant Bの`shared-doc`、tenant Aで付与されていない文書、存在しない文書は、どれも同じ`404`になります。

MCPのHTTP通信では、1が次のように変わります。MCPは、呼び出し元のトークン（宛先はMCP）を検証し、IdPで交換して、backend宛ての短命のトークンを得ます。backendはそのトークンを自分で検証し、`(identity_provider_id, sub)`からagentを引く対応付けの行でtenantを決めます。2以降は同じです。

### D.4 読み取れること（AG-1〜AG-12）

AG（agent）は本書内の仮の呼び名で、Requirement IDではありません。各項目は「内容と理由」と「SUIでの根拠」に分けて書きます。一般化できるか、まだ何が示されていないかは、D.6にまとめました。

**AG-1 外部agentは、membershipを持たない別種の主体として扱う**

- 内容と理由: agentにmembershipやbrowser sessionを渡さず、Tenant Adminが登録した資格情報と、文書ごとの明示的な付与だけで読み取りを許します。利用者のsessionをagentへ委譲する案（ADR-0093の案C）は、sessionの権限が読み取り専用の用途より広く、`tenantSessionVersion`をagentにも持たせることになるため、採られませんでした。付与の行が無ければ読めない「構造上の既定拒否」なので、policyの設定を書き忘れても開きません（ゲスト受け入れのADR-0080から引き継いだ方針）。
- SUIでの根拠: `agent_credentials.py`の`AgentRequestPrincipal`、`routes/docs.py`の`TenantContext(membership_id=None, resolved_by="agent_credential")`。この経路は外部のPDPを問い合わせず、付与の行が判断を兼ねます（`reason="agent_document_grant"`）。

**AG-2 tenantは、サーバー側の行からだけ決める**

- 内容と理由: tenantと付与は、資格情報から引いた行（不透明な資格情報なら索引の行、OAuthなら対応付けの行）で決めます。トークンのclaim、`x-tenant-id`のようなheader、query、pathの値は使いません。IdPの設定ミスが、そのままtenant越境になるのを避けるためです（SUI ADR-0063 D8）。
- SUIでの根拠: `test_tenant_hints_in_the_request_are_never_trusted`（`x-tenant-id`とqueryの`tenantId`で他tenantを指定しても404）、`test_the_tenant_comes_from_the_binding_not_from_the_token`（別のtenantを名乗るclaimを足した署名付きトークンでも、tenantは対応付けの行のまま）。

**AG-3 tenantが決まる前に引く索引表はRLSの外に置く。理由を台帳に書き、状態は毎回RLS付きの表から読み直す**

- 内容と理由: tenantを資格情報から導く以上、「どのtenantか」を引く表そのものは、tenantで絞れません。そこで索引表は、鍵付きハッシュ（OAuthでは`(identity_provider_id, sub)`）から`(tenant, agent, 版)`だけを返す最小の表にして、RLSを掛けません。この免除は、理由を添えて台帳に載せます。失効・期限・版・文書の付与は、tenantの領域を設定した後に、RLS付きの表から毎要求で読み直します。索引に状態を持たせると、失効が索引の更新に依存し、更新漏れがそのまま失効漏れや越境になるからです。
- SUIでの根拠: migration `0036`（`agent_credentials`と`agent_document_grants`は同じmigrationでFORCE RLS、つまり表の所有者にもRLSを適用する設定を有効にし、`agent_credential_index`には掛けない）、`0037`（`agent_oauth_bindings`も索引としてRLSなし）、`_active_principal`、`RLS_EXEMPT_TENANT_TABLES`の2項目。`test_agent_credential_postgres_rls.py`は、RLS付きの2表がtenant指定なしでは0件になることと、他tenantへの書き込みが拒否されることを確かめます（PostgreSQLが必要）。

**AG-4 拒否の形を揃える。無効な資格情報は一つの応答、付与外・不存在・他tenantは一つの404**

- 内容と理由: 不正・未知・失効・期限切れ・版の不一致は、区別せず`401 agent_credential_invalid`にします。付与外の既存文書、存在しない文書、他tenantの文書は、状態コードも本文も同じ`404`にします。応答に差があると、資格情報の状態や文書の存在を探る手掛かりになるためです。agent用のheaderがあるのに無効なときは、通常の利用者の経路へ落とさず、そこで閉じます。route許可と書き込みの拒否は付与の確認より先に行うので、`403`も文書の有無に依存しません。
- 残る例外: 二つの形式の資格情報を同時に送ると`400`、署名鍵を取得できないときは`503`です。どちらも資格情報の有効性そのものには依存しません。`503`は「評価できなかった」を表し、通す（`allow`）ことも、拒否する（`deny`）こともしません。これは5.2のL2が定める`not-evaluated`の区別と同じです。登録時の`409`は別の問題を含むため、AG-12で扱います。
- SUIでの根拠: `test_ungranted_documents_look_the_same_as_missing_ones`（状態コードと本文の一致）、`test_malformed_or_unknown_credentials_fail_closed_with_one_response`、`test_invalid_or_unbound_tokens_fail_closed_with_one_response`（7種）、`test_an_unavailable_key_source_is_503_not_a_pass`、`test_sending_both_credential_forms_is_rejected`。

**AG-5 membershipを持たない主体には、拒否リストではなく許可リストを使う**

- 内容と理由: agentが通れるのは、`GET /docs`、`GET /docs/{doc_id}`、`GET /ai/proposals/status`、`POST /docs/{doc_id}/context-audit`の四つだけです。読み取りであっても、本文由来の派生データを返す経路（判断ログ、類似候補）は、個別に検討するまで`403 agent_route_not_enabled`で閉じます。許可の照合には、実際のURLではなく、Webフレームワーク（FastAPI）が解決したroute定義（`/docs/{doc_id}`の形）を使います。
- SUIでの根拠: `_AGENT_ALLOWED_ROUTES`、`test_derived_read_routes_stay_closed_to_agents`（3経路）、`test_agent_credentials_never_write_export_or_change_state`（PUT、export-audit、archive、unarchive）、`test_agent_credentials_do_not_authenticate_other_routes_in_the_saas_profile`（AI系の6経路）。agent用のheaderを解釈するのは`_authorize_request`と`list_documents`の2か所だけで、ほかのrouteでは認証として働きません。
- 注意すべき点: `/ai/proposals/status`だけは、session検査を「agent用のheaderがあれば省く」変種（`require_tenant_scoped_api_precondition_unless_agent`）を使います。この変種は認証をせず、検査を省くだけで、handlerが資格情報を必ず検証することを前提にしています。headerの有無で検査を省く形は、handlerが検証を忘れると素通しになります。SUIには、`/ai/`と`/context/`以下のrouteでこの変種を許す経路を`/ai/proposals/status`の1つに限る試験（`test_all_tenant_content_ai_and_context_routes_install_precondition`）があります。ほかの接頭辞のrouteについては、この限定を確かめる試験を確認できていません。

**AG-6 開示は許可リスト方式の文書viewで行い、伏せたことが分かる印を持たせる**

- 内容と理由: `GET /docs/{doc_id}`は、agentには文書をそのまま返しません。許可した構造の項目（カードの位置・種別、辺、島の構成など）と、人が確認した本文（`textReviewed`が真のカード本文、`titleReviewed`が真の島の題名）だけを写して返します。許可リスト方式なので、`DocumentV1`に本文を含む項目が増えても、明示するまでagentには出ません。拒否リスト方式なら、増えた項目は既定で出てしまいます。
- SUIでの根拠: `_agent_document_view`、`test_only_reviewed_text_and_allowed_structure_reach_agents`（本文に見立てた11個の文字列が応答に出ないこと）。
- 印の揺れ: 伏せたことを示す形が一つではありません。voidやナラティブのように空にできない文字列は`[withheld]`、未確認のカード本文は空文字列、未確認の島の題名と文書の題名は`null`または欠落です。確認済みかを示す旗（`textReviewed`、`titleReviewed`）も同時に返るので、受け手は推測できますが、統一した印ではありません。これは付録B.2の6（`omit`は「無い」と区別できない）と同じ論点で、SEI ADR-0030の`withheld`（開示しない）と`unknown`（不明・無い）の区別を、共通部品の契約として一つの形に決める必要があります。`[withheld]`は、必須項目を空にできないSUIのスキーマ制約から来ているので、そのままは一般化できません。
- 付録Bとの関係: 付録B.1は開示判断の形を三つ数えました。backendの許可リストは、その外に現れた第四の形です。「一つの宣言から全出力を導く」（付録B.2の2）はまだ実現しておらず、この第四の形が加わって、判断の置き場所はむしろ増えています。

**AG-7 受け取ったトークンを下流へ転送しない。トークン交換で宛先を変える**

- 内容と理由: HTTP通信のMCPは、自分宛て（audience、つまりトークンの宛先がMCP）のトークンを検証したあと、それをbackendへ渡さず、IdPのトークン交換でbackend宛ての短命のトークンに引き換えて送ります。ADR-0094によれば、MCPの認可仕様は、自分宛てのトークンを下流のAPIへそのまま渡すこと（token passthrough）を禁じています（本書は仕様の原文を確認していません）。転送すると、宛先という境界が消えるからです。MCP宛てのトークンがbackendでも通るなら、MCPに届いたトークンを持つ者は誰でもbackendを直接呼べ、backendも「このトークンは誰に向けたものか」を検査できません。MCPが専用の資格情報でbackendを呼び、検証済みの`client_id`を申告する案は、MCPが侵害されると任意のagentを名乗れるため採られませんでした。
- SUIでの根拠: `token_exchange.ts`、`http_server.ts`（交換できない要求は`502`で閉じ、backendを呼ばない）、`http_server_exchange.test.ts`（backendへ届くのは交換後のトークンだけで、`Authorization`と`X-API-Key`は付かない）、`test_agent_oauth_http.py`（MCP宛てのaudienceを持つトークンは、backendで401）。

**AG-8 失効は次の要求から効かせ、キャッシュで遅らせない**

- 内容と理由: 資格情報の状態・期限・版と、文書ごとの付与を、毎要求で読み直します。失効の権威は、tenant側の行です。IdPが発行したトークンの有効期限ではありません。OAuth経由でも、資格情報を失効させると、署名が有効な交換後のトークンであっても、次の要求から拒否されます。ADR-0093は、キャッシュするなら`deployment + tenantId + agentId + credentialVersion`をキーにし、トークンの有効期限を越えて保持しないと定めていますが、実装はキャッシュを持ちません。
- SUIでの根拠: `test_credential_revocation_applies_to_the_next_request`、`test_document_grant_revocation_applies_to_the_next_request`、`test_expired_and_version_mismatched_credentials_are_rejected`、`test_revoking_the_credential_stops_the_oauth_path_at_the_next_request`。

**AG-9 監査の主体は、検証済みの主体から求める**

- 内容と理由: 監査eventの主体は、リクエストの`x-actor-ref`ではなく、検証済みの`agent:<agentId>`から作ります。eventには`tenantId`を含め、本文・題名・トークンは含めません。MCPが送る読み取りの監査（`context-audit`）も、資格情報が束縛されたtenantと付与された文書の外は404です。
- SUIでの根拠: `_audit_actor_ref`、`test_audit_event_carries_tenant_and_agent_but_no_content_or_token`（偽の`x-actor-ref`を送っても、記録される主体はagentのまま）、`test_proposal_status_and_context_audit_are_scoped_to_the_grant`。

**AG-10 守りが揃わない構成は起動しない。揃ったことを、解除条件として試験で固定する**

- 内容と理由: MCPは`saas-multitenant`で、stdioなら`SUI_MCP_AGENT_CREDENTIAL`、HTTPならトークン交換の設定が揃わない限り起動を拒否します。静的な`SUI_API_KEY`は、tenantを証明しないので、この構成で設定されていること自体を拒否します。ADR-0093は、起動拒否を解除する条件を9項目で書き、transportごとに解除する手順にしています。これは5章の原則3（宣言したのに守りが無い構成は動かさない）の実例です。L3の適合条件も、この種の「解除条件」として使うのが自然だと考えられます。
- SUIでの根拠: `document_client.ts`の`validateMcpRuntimeProfile`、`document_client.test.ts`、ADR-0093の解除条件の表。

**AG-11 新しい表・routeの分類漏れを、台帳の試験で検出する**

- 内容と理由: 隔離の保証は、新しい表やrouteが分類されないまま増えると、静かに崩れます。SUIには二つの台帳があります。(1) tenantを持つ全ての表を、RLS対象か、理由つきの免除かに分け、対象の表の一覧も明示します。実在しない免除（古くなった台帳）も失敗にします（`RLS_EXEMPT_TENANT_TABLES`）。(2) 登録された全routeについて、tenant境界を設ける共通関数を呼ぶか、理由つきの免除一覧に載せるかを、構文木の走査で確かめます（`_UNGUARDED_ROUTE_EXEMPTIONS`）。
- 実地の確認: 管理APIを追加したcommit（`586d647`）で、route台帳の試験`test_every_registered_route_is_tenant_guarded_or_explicitly_exempt`を実行したところ、`/tenant-admin/agent-credentials`の4経路が未分類として失敗しました（本書の作業で再現）。次のcommit（`bac01f7`）で分類して通っています。表の台帳でも、agentの4表の分類が同じ形で加えられています。表の台帳が失敗することは、コードを読んで確認しただけで、実行していません。
- SUIでの根拠: `test_document_access_rls_postgres.py`、`test_tenant_session_precondition.py`。

**AG-12 tenantが決まる前の索引は全体で一意でなければならず、その衝突はtenantを越えて観測できる**

- 内容と理由: OAuth主体の対応付け`(identity_provider_id, sub)`は、tenantが決まる前に一つのagentを引くので、全tenantを通じて一意でなければなりません。そのため、tenant Aが結んだ主体をtenant Bの管理者が結ぼうとすると、`409 agent_oauth_principal_bound`で拒否されます。`api.md`は「どのtenantのagentかは示さない」と書いていますが、結ばれていること自体は伝わります。AG-4の「他tenantのものは存在しないものと区別しない」の例外です。観測できるのは、自分のtenantに登録済みのagentがあり、`agent.register`のcapabilityを持つ管理者が、他tenantの対応付けのIdP識別子と`sub`を推測できる場合に限られます。不透明な資格情報の索引は、鍵付きハッシュが衝突しないので、この問題がありません。
- 構造上の緊張: 全体で一意にする限り、衝突は観測できます。tenantごとに一意にすると、同じ主体から複数のagentが引けてしまい、どのtenantかを決める入力が別に要ります。それをトークンやheaderから取るとAG-2に反します。
- SUIでの根拠: `AgentCredentialRepository.bind_oauth_principal`、`routes/agent_credential_admin.py`の`409`、`test_a_principal_can_be_bound_to_only_one_agent`（repositoryの層で、別tenantとの衝突を拒否）。HTTPの層で別tenantとの衝突を扱う試験は見当たりません。コードと`api.md`を読んだ範囲の観察で、実害の大きさは評価していません。受け入れるか、別の設計にするかは、SUI側で判断する事項です。

### D.5 L2・L3へ反映する案

**L2の手順の読み替え案**

1. 手順2「membershipが有効であることを検証する」は、主体の種類によって根拠が違います。memberはmembership、agentは資格情報の状態（失効・期限・版）と文書ごとの付与です。「主体の種類に応じた有効性の根拠を、毎要求で検証する」と書き、memberとagentを例に挙げる形が考えられます（AG-1、AG-3、AG-8）。
2. 手順6「外部の認可判断（PDP）へcapabilityを問い合わせる」は、SUIのagentでは行われず、付与の行が判断を兼ねます（ゲストも同じ）。三値で見ると、「問い合わせる必要がない（適用外）」と「問い合わせたが評価できなかった（`not-evaluated`）」は別の状態で、同じ扱いにしてはいけません。SUIは前者として扱っています。未決なのは、外部のPDPが持つ全体の方針（たとえば全面停止）をagentにも適用すべきかどうかで、ADR-0093とADR-0094には記述が見当たりません。

**L3の行列へ加える合格条件の候補**

付録Aの末尾が述べるとおり、いずれも「該当機能があれば必須」の形で書きます。

| 行列の経路 | membershipを持たない主体（agent）で、追加して問う条件の候補 | 項目 |
| --- | --- | --- |
| GET | 同じdocIdを持つ別tenantの文書を読めない。付与外・不存在・他tenantが同じ404になる。tenantはサーバー側の行だけで決まり、header・query・claimでは変わらない | AG-2、AG-4 |
| PUT / export / share / import | agentの資格情報では、書き込み・export・archiveなど状態を変える経路は常に拒否される。許可リストに無いrouteは、読み取りでも拒否される | AG-5 |
| list | 付与された文書のmetadataだけを返し、本文由来の項目（題名など）は返さない。tenant全体の件数や存在を示さない | AG-4、AG-6 |
| MCP | 資格情報の経路が揃わない構成は起動しない。静的なAPI keyでtenantを済ませない。HTTPでは、受け取ったトークンを転送せず、宛先を変える | AG-7、AG-10 |
| audit | 監査の主体は検証済みの主体から求め、headerなどの申告を使わない。本文・題名・トークンを含めない | AG-9 |
| cache | 資格情報の状態をキャッシュする場合は、キーにtenant・agent・資格情報の版を含め、トークンの有効期限を越えて保持しない。SUIはキャッシュを持たない | AG-8 |
| rate limit | 付録Aの項目に加えて、キーをtenantとagentのどちらにするかを決める（ADR-0093の未解決事項） | （付録A） |
| agent credential | 失効・期限切れ・版の不一致・付与の取り消しが、次の要求から効く。索引表は最小で、状態は毎回RLS付きの表から読む | AG-3、AG-8 |
| 経路ではなく構造 | 新しい表やrouteが分類されないまま増えると、試験が失敗する。tenantが決まる前の索引にある一意制約が、他tenantの存在を伝えない | AG-11、AG-12 |

### D.6 項目ごとの一般化の見立てと、まだ示されていないこと

見立ての語は次の意味です。**一般化できそう**は、SUI以外でも同じ問題が起きる理由が文書から読める、という意味で、他のApplicationで確認できたという意味ではありません。**条件つき**は、前提が成り立つ範囲に限ります。**SUI固有**は、SUIの語彙やスキーマに依存します。**未決**は、設計がまだ決まっていません。どの項目も、第二の消費者で再現されるまでは、標準化の根拠にしません。

| 項目 | 一般化の見立て | まだ示されていないこと |
| --- | --- | --- |
| AG-1 別種の主体 | 条件つき。membershipを持たない主体を別の種類にする形は、ゲストとagentで共有されている。ただし二つとも同じ実装の中なので、独立した二例ではない。読み取り専用の用途と、確認済みの本文の扱いはSUI固有 | 付与の検査をゲストとagentで共通化できるか（ADR-0093は範囲外とした）。第三の主体種別。外部のPDPを呼ばないことが意図した設計か |
| AG-2 tenantは行から決める | 一般化できそう。SUIの既存方針の延長で、EKIがtenantをまたぐ付与を禁じているのとも向きが合う（EKI側の個別確認は未） | claimにtenantが入る実在のIdPでの試験。SEI・EKIの実装での再現 |
| AG-3 tenant確定前の索引 | 一般化できそう。tenantを要求から導く主体（ゲスト、agent、OAuth主体）で、同じ形の索引が現れている | 索引の列が最小であること（状態を持たないこと）を機械的に確かめる仕組み。免除理由の妥当性を確かめる仕組み。PostgreSQL以外のDatabase Providerでの扱い（L1の`unsupported`）。実運用のconnection pool |
| AG-4 拒否の形 | 一般化できそう | 応答時間や大きさなど、本文以外に現れる差。例外（400、503、409）の網羅的な洗い出し |
| AG-5 許可リスト | 一般化できそう。membershipを持たない主体の既定は許可リストでよい。許可する4経路の中身はSUI固有 | 全routeを列挙して、許可外が必ずagentを拒否することの機械的な確認（現状は抜き取りの13経路）。許可routeを増やすときの審査の手順 |
| AG-6 許可リスト方式のview | 許可リスト方式は一般化できそう。印の形は未決。`[withheld]`はSUIのスキーマ制約に依存し、SUI固有 | 新しい項目が許可リストに載っていない場合の検出（既定では出さないが、判断した記録は残らない）。受け手が「伏せた」と「空」を区別できるか。SEIの`withheld`との整合 |
| AG-7 トークンを転送しない | 条件つき。宛先を変える原則は一般化できそうだが、交換先のIdPがRFC 8693に対応することが前提 | 実在のIdPでの交換。交換後の`sub`の意味（元の主体か、交換を行った側か）。MCPとbackendを実際につないだ試験（現状は両者を差し替えて代用）。交換結果の再利用の可否と、IdPへの負荷 |
| AG-8 即時の失効 | 一般化できそう。ただし毎要求の読み直しには費用がかかる | 負荷。DBのreplicaを置く構成での「次の要求」の意味。キャッシュを入れる場合のキーと寿命の試験 |
| AG-9 監査の主体 | 一般化できそう | 管理操作（登録・失効）の監査event。現状は`logger.info`で記録し、読み取りと同じ監査の経路には載せていない（コードを読んだ範囲）。保存された監査を別tenantが読めないこと（付録Aのaudit行） |
| AG-10 起動しない構成 | 一般化できそう。5章の原則3の実例 | 解除条件8（ログにトークンが出ないこと）の試験。「固定済み」はADRの表に人が書いた状態で、機械的に生成されたものではない |
| AG-11 台帳の試験 | 一般化できそう。台帳の形は言語に依存しない | 免除理由の中身の妥当性（自由な文章で、機械検査の対象外）。台帳を更新すれば通る点（忘れの検出であって、判断の正しさの検証ではない）。route・表以外の軸（cache key、object storage、job、webhook）の台帳 |
| AG-12 一意制約の観測 | 未決。一般化できる課題だが、解がまだ無い | HTTPの層での試験。実害の評価。tenantごとに一意にした場合の設計 |

### D.7 この付録が主張しないこと

- **SUIの実装が安全であるとは主張しません。** 対象は未mergeのbranchで、PR #3214はDraftです。外部の審査や、攻撃者の視点での検討（脅威モデル）の代わりにもなりません。
- **実在のIdPや実在の資産で確認したとは主張しません。** backendの試験は、テスト用に自前で作った署名鍵でIdPを模し、MCPの試験は、IdPとbackendを`fetch`の差し替えで代用しています。実在の機微資料は使っておらず、SUIの安全不変条件（SafeMode既定ON、AI出力はproposal-only、`human_reviewed`は人間だけが設定）は変えていません。
- **PostgreSQLの本番相当の構成で確認したとは主張しません。** RLSの試験は、実行した作業者の申告では使い捨てのPostgreSQL 16 clusterで通ったものです。本書は再実行しておらず、その記録も確認していません。実運用のconnection poolや権限の構成も確認していません。
- **標準化に十分な証拠が揃ったとは主張しません。** 証拠はSUI一件で、第二の消費者がありません。7章の判断基準（第二のApplicationで再利用できそうか）はまだ満たしていません。
- **TEIのCore、Plugin契約、`CapabilityKind`の変更を求めません。** 本付録の項目は、L3の試験項目の候補にとどまります。
