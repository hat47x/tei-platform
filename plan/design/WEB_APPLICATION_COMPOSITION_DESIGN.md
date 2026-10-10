# Webアプリケーション構成キット：安全な要求経路と認証方式の組合せ

- **状態**: 評価待ち（2026-10-09）。既存実装の読み取りと設計候補をまとめたものであり、採用済み仕様や動作確認済みの機構ではない。
- **担当課題**: [#1043](https://github.com/hat47x/tei-platform/issues/1043)。
- **関連する正本**: [案件全体のプラグイン構成](PROJECT_PLUGIN_COMPOSITION_DESIGN.md)、[実装アーキテクチャ](../../docs/architecture.md)、[セキュリティ境界](../../docs/security.md)。
- **責務**: Webの要求・応答経路と認証方式を安全に接続する構成上の規則を検証する。業務仕様、UI部品、プラグインの共通解決機構をここで再定義しない。

## 1. 解消したい問題

Webアプリケーションは画面の集合ではない。ブラウザ、HTTP入口、業務操作への接続、認証・セッション・認可、入力制限、監査、障害時の応答が一続きになって初めて使える。しかし、標準機能をプラグインとして分割するだけでは、それらをどの順序と条件で接続し、どこまで交換できるかが定まらない。

例えば、既存のID・パスワード認証から企業のシングルサインオンに変更した場合、業務操作の許可条件は変わってはならない。また、一つのアプリで複数の認証方式を認めても、弱い方式への無条件の切替えや、別人の利用者情報の混同を起こしてはならない。

目的は、**接続可能な部品を増やすだけでなく、危険な組合せを実行前に拒否できること**である。

## 2. 構成資産と実行権限を分ける

| 資産・機構 | 所有する責務 | 所有しないもの |
| --- | --- | --- |
| Web構成キット | HTTP要求の分類、必要な処理点、処理順序の制約、標準構成例 | 案件固有の業務意味、特定IdPの認証情報 |
| アプリケーション構成プロファイル（#1016） | 案件が採用するWeb構成・UI構成・プラグインと参照 | 第二のProvider解決器、実行許可 |
| 機能契約・Provider（#951） | 契約・実装・適合性・作用域の照合 | HTTP要求経路全体の意味 |
| UI構成キット（#997） | ブラウザ内の描画、入力、操作接続、案件UI差分 | サーバ認可、秘密情報、HTTP保護の解除 |
| Web Host / Go net/http | HTTP要求・応答の処理、経路の接続 | Canonicalな業務意味の変更 |
| Application Driver | 確定したAction / Capabilityへの接続、業務結果 | ログイン画面、Cookie、OAuthプロトコル |
| Environment / Resource Broker | 接続先、秘密情報、実行権限とその現在性 | 案件の業務定義、認証済み利用者の業務権限 |

「Web構成キット」は複数の資産を束ねる設計・配布上の単位であり、単独のRuntime Plugin種別ではない。実行時に各HTTP要求でProviderを再検索することもしない。既存のPlugin Lock・Environment Binding・Capability Scopeに基づき、構成時に採用対象を固定する。要求の入口で確認すべき現時点の権限・セッション・失効は別途検査する。

既存の業務仕様テンプレート（#998）は、このキットの外側にある。

## 3. Web全体の境界

~~~text
Browser
  ├─ UI構成キット / 状態・操作
  ├─ 通信接続 / 応答表示
  └─ ブラウザ固有の機能
            │ HTTPS
            ▼
Web Host
  ├─ 信頼済みHTTP入口 / 経路の分類
  ├─ 経路ごとの保護処理の組立て
  ├─ 認証方式の選択 / 認証結果の検証
  ├─ セッション / 要求元保護 / 認可
  └─ Application Actionへの接続
            │
            ▼
TEI Driver / Capability / Plugin Runtime
            │
            ▼
Environment Resource Broker / DB / 外部サービス
~~~

標準的な企業向けWebアプリでは、ブラウザ用バックエンド（BFF）を第一候補とする。OIDC等のトークンをサーバ側に保持し、ブラウザには安全なセッションを渡す構成である。BFFは**OAuthを使用するWeb構成の推奨候補**であって、すべてのWebアプリの必須構造ではない。従来型のサーバ描画、API分離型、サーバ不要の公開サイトにも適用可能な構成分類を残す。

通信方式や描画方式を変えても、Application Actionの識別子、Validation、Authorization、Business Result、Execution Errorの意味を変えない。

## 4. HTTP処理は条件付きの有向グラフとして表す

単純な「優先順位1、2、3…」の登録順に依存すると、追加プラグインや短絡応答によって保護・監査が抜ける。そこで各処理点に、入力と出力、実行する段階、作用できる範囲、前後条件、短絡・失敗時の動作を宣言させる。

### 4.1. 少数の処理段階

- **受信段階**: 信頼する通信入口の確定、経路の選択、要求ID、サイズ・頻度制限。
- **入口保護段階**: 経路に応じた認証、要求元保護、セッション確認、認可の前提形成。
- **操作実行段階**: 入力の構文検証、Application Actionへの接続、業務Validation・認可・実行。
- **応答段階**: Business Result / denied / errorの変換、必要な保護ヘッダー、監査、終了処理。

これは全処理を単一の直列チェーンに固定する規則ではない。例えば認証コールバックは通常の画面と異なる要求処理を持つ。CORS事前要求、静的資産、Webhookにも専用経路が必要である。応答処理や監査は正常終了だけでなく、入口で拒否された要求にも適用し得る。

処理点には次の性質を区別する。

| 性質 | 意味 |
| --- | --- |
| 提供契約・必要契約 | 何を作り、事前に何が必要か |
| 処理段階 | 受信、入口保護、操作、応答のどこに参加するか |
| 適用条件 | どの経路・要求種別・Targetで有効か |
| 前後制約 | 必要な他の処理より先／後に実行すること |
| 効果 | 主体の確定、ヘッダー追加、監査記録など、変更可能な対象 |
| 短絡時の振舞い | 拒否・障害・早期応答でも実行される後処理 |
| 信頼境界 | Hostによる保護、隔離の必要性、Resourceへの要求 |

処理点を自由な文字列で並べるだけでは契約にならない。既存の#951の機能契約を参照し、必要なWeb固有の処理段階・効果・順序制約だけを小さな記述として追加する候補を検証する。

### 4.2. 最小の構成例（仕様候補であり実行不可）

~~~yaml
web:
  routes:
    - id: sales.submit
      match:
        method: POST
        path: /scenario/submit
      policy: interactive
      action: order_entry.submit_order

    - id: login.callback.corporate
      match:
        method: GET
        path: /login/callback/corporate
      policy: oidc-callback

  policies:
    interactive:
      session: web.session
      authentication: employee
      csrf: required
      authorize: action-boundary

    oidc-callback:
      authentication: corporate.callback
      callback_protection: state-and-code-validation

  processing:
    - id: web.request-context
      provides: [request.context]
    - id: web.session
      requires: [request.context]
      provides: [session.context]
    - id: web.action-authorization
      requires: [authenticated.principal, action.identity]
      before: [application.action.invoke]
    - id: web.audit
      requires: [request.context]
      observes: [request.started, response.completed, response.rejected]
~~~

この例の値やフィールドは現行TEIの実装済みスキーマではない。実際の適用範囲はメソッド、Host、経路の正規化と信頼済みのルーティング結果で判定する。利用者から届いた任意のHeaderや入力値を理由に、保護経路から公開経路へ切り替えない。

### 4.3. 実行計画を確定する際の検査

1. 既存の機能契約とProvider解決で候補を確定する。Plugin ID、版、成果物の同一性、作用域、適合性を照合する。
2. HTTP経路を正規化し、経路ごとの保護方針を対応付ける。重複した経路・曖昧な公開範囲を拒否する。
3. 処理点の必要契約、前後条件、排他的な実装を照合し、循環依存・欠落・競合を拒否する。
4. 認証・認可・要求元保護等の必須条件について、早期応答、拒否、例外、応答の各経路でも実行条件が成立するかを確認する。
5. 条件を満たす場合だけ、Targetに適した処理構成を生成する。同じグラフを毎回動的探索せず、Go側は通常のHandler連結等に落とし込む。
6. 構成の根拠を既存のLock・Provider選択へ逆追跡できるようにする。構成記録そのものを新たな実行権限にしない。

処理順序に複数の安全な解がある場合は安定した規則で再現可能にする。ただし、複数の処理が同じ単独公開点を所有する場合や、異なる認証方式間の選択が曖昧な場合は、名前順で勝者を決めず明示的な選択を求める。

**保護条件は単なる拡張点ではない。** 業務操作の認可、公開経路の指定、Cookieに依存する状態変更の要求元保護などを、任意のUI拡張や未信頼の処理点が削除できない構成とする。実行Host自身の強制と、Pluginが自己申告する制約を混同しない。

## 5. 複数の認証方式を共存させる

### 5.1. 認証方式と業務主体を分ける

~~~text
経路と認証方針
     │
     ├─ 企業OIDC ──────┐
     ├─ 既存認証 ──────┤
     ├─ Passkey ───────┤
     └─ サービス間認証 ─┘
                       ▼
       出所付きの認証結果を検証
                       ▼
       明示的な主体対応付け
                       ▼
       セッションまたは要求単位の本人確認
                       ▼
       Application Action / security.authorize
~~~

複数方式の共存は、**方式の選択、本人確認、利用者の対応付け、セッション、業務認可**の五つを分離して成立させる。

- **方式の選択**: 案件・経路・必要な認証強度から、許可された方法を提示／要求する。Provider障害時の自動的な弱い方式への切替えは禁止する。
- **本人確認**: 各方式が独自のプロトコルで検証する。OIDCのコールバックやサービス間の証明書認証を、ID・パスワード入力と同じ手順に押し込まない。
- **主体対応付け**: 原則として認証発行元と主体識別子の組を保持する。同じメールアドレス・表示名による自動統合をしない。テナント境界も明示する。
- **セッション**: ブラウザの利用者が認証を継続するための独立した契約。サービス間の要求単位認証と混同しない。
- **業務認可**: 最終的にはサーバ側のApplication Action境界から既存のsecurity.authorizeへ接続する。認証方法やUI表示だけで操作を許可しない。

単一Providerの交換は既存#68/#150で意味契約の範囲まで実証済みである。しかし「OIDCと既存認証が一つのWebアプリで共存すること」や「実IdPと実Cookieセッションを使った安全性」まで実証済みではない。

### 5.2. 最低限の認証結果とその限界

| 情報 | 説明 |
| --- | --- |
| 発行元と主体ID | どの認証元が誰であると主張したか。認証元の検証を前提とする |
| 認証した時刻 | 再認証や追加認証の判定に使う |
| 認証方式・強度 | 管理操作などが求める水準との照合に使う。値の意味はProviderごとの契約で明確にする |
| 対象Application・Tenant | どの範囲で使用してよいか。入力に含まれるだけでは許可とはしない |
| 検証の有効期間と現在性 | 期限・失効・セッションとの結び付き。保持済みの結果だけで永続保証しない |

OIDC等から受け取った属性は、そのままTEIの業務権限ではない。既存利用者の組織・役割・無効化・有効期間などは、必要な既存Adapterで判断する。

### 5.3. セッションと要求元保護

ブラウザ側の標準候補はサーバ管理セッションとし、CookieはSecure、HttpOnly、適切なSameSite・適用範囲などの条件を備える。認証後のセッションID再生成、期限、ログアウト・失効、追加認証の有効期間を契約に含める。

Cookieに依存して状態を変更する要求では、CSRF対策が必要である。SameSite属性だけで十分と扱わず、トークンや要求元検証を利用形態に応じて選ぶ。CORS許可を認証・認可の代用にしない。OIDCコールバックではstate、認可コード、PKCE、Issuerの取り違えなどを防ぐ検証が必要である。

公開認証開始・認証復帰・静的資産・ヘルスチェック・CORS事前要求などは、それぞれ必要な保護の異なる**明示的な例外経路**とする。例外の存在によって業務操作が未認証で実行される構成は拒否する。

## 6. リクエストの安全性と業務の意味を混ぜない

ブラウザの要求を処理する際、三つの判定を混同しない。

1. **通信・要求の保護**: 経路、要求元、署名、Cookie、利用制限、入力サイズなど、Web Hostが強制する。
2. **本人確認・業務操作の認可**: 認証主体を特定し、Application Actionと対象Resourceに対して許可を判断する。拒否は業務結果か実行エラーかを既存契約で区別する。
3. **Plugin自身の実行権限**: 実行環境のGrant、Resource Broker、Lease等で確認する。利用者認可を通過しただけではPluginに無制限の権限を与えない。

応答は、認証の不成立、認可の拒否、業務Validationの不成立、業務結果、実行障害を無差別に「HTTP失敗」へ畳まない。HTTP状態への変換はWeb側の実現であり、TEIの業務結果・Execution Errorの意味を変えない。

監査と追跡情報は業務結果に混入させず、失敗や早期応答も対象とする。資格情報、セッションID、認証トークン、秘密情報を通常のログ・監査・生成物に保存しない。

## 7. 現行Referenceとの対応

2026-10-09にGitHubのmain上で確認した範囲は次のとおり。設計候補を現行の実装機能として読み替えない。

| 現在の参照点 | 確認できた内容 | 本提案でなお必要なもの |
| --- | --- | --- |
| tools/reference-first-web-target/main.go | Goのhttp.Server、ApplicationごとのRunner接続、起動時のApplication照合 | 環境別Web構成の選択、保護処理の生成 |
| tools/reference-first-web-target/server.go | ServeMuxのHTTP経路、/scenario/submit、表示・JSON応答 | 経路方針、セッション・CSRF、認証開始／復帰、監査 |
| reference/ui-customization/web-react/README.md | 既存Go ActionのJSON契約に接続するUI参照構成 | 実React・ブラウザでの動作、サーバの実認証・保護 |
| #68 / #150 | 同じsecurity.authenticate Bindingの実現方式交換、既存認可との接続 | 複数方式共存、OIDC、主体リンク、ブラウザセッション |
| #951 | 汎用機能契約とProvider適合／権限のReference候補 | 本Web処理点での使い勝手と実Consumerによる検証 |

現在のFirst Web Targetでは、認証情報を要求JSONに含む参照用シナリオがある。これは将来の一般利用Webアプリ向け認証フローとして推奨するものではない。BFF方式を検証する際は、UIから既存の資格情報を送る動作と、CookieセッションでActionへ接続する動作を別々の実証として扱う。

## 8. 代表経路と失敗例

| 対象 | 成立すべき条件 | 拒否する例 |
| --- | --- | --- |
| 通常の業務操作（POST） | セッション、要求元保護、対象Actionの認可、業務Validation | UIからの直送だけで認可を迂回 |
| 認証復帰（GET） | 事前の認証開始と結び付き、認証プロトコルの検証が完了 | 任意のcallback URLや不正stateでログイン成立 |
| サービス間API | 対象サービスの独立した認証と認可 | ブラウザのセッションをサービス権限へ流用 |
| 静的資産／健全性確認 | 明示的に公開範囲を限定 | 共通pathのあいまいな一致で業務操作まで公開 |
| エラー／短絡応答 | 安全なHTTP応答、秘密情報を含まない監査 | 処理の途中終了で保護や監査が抜ける |
| 認証方式の共存 | 明示的に選択された本人確認と主体対応付け | メール一致で別Issuerの主体を統合、障害時に弱い方式へ降格 |
| プラグイン更新 | 契約・正確な成果物・権限を再照合 | 読込順、同名Provider、版番号だけで勝者を決定 |

## 9. 段階的な実証

**E0：設計の対応付け（本書）**。現行のWeb経路、既存Capability、UI参照構成、未保護の処理を分類する。まだ実装と実行試験は行わない。

**E1：読み取り専用の構成検査**。初版の[参照実装候補](../../reference/web-composition/README.md)では、保護された業務操作の処理グラフと、二つの認証方式の明示的な主体対応付けを検査する。保護の欠落、前後制約、排他的接続点、公開範囲、作用域、暗黙の弱い方式への切替えを負例とした。12件の条件判定をV8隔離環境で直接確認したが、リポジトリの実ファイルに対するNodeテスト、実HTTP、実IdPの検証は未実施である。処理点が保護を宣言したことを、実際の保護の成立とは扱わない。

**E2：Goの実HTTP処理との結合**。[E2のGo参照実装](../../reference/web-composition/README.md)を追加した。合成Providerを使う一つのPOST経路について、Host側でセッション確認・要求元保護・Action認可を強制し、拒否・障害・短絡応答と監査を試験する。Goの`httptest.NewServer`を使ったローカルHTTP試験、race test、vet、gofmtに成功した。ただし**既存First Web Targetの実Command Actionにはまだ接続していない**ため、現行ServeMuxと`security.authenticate`/`security.authorize`を通した検証は残る。従来のシナリオを置き換えず、受入境界を分ける。

**E3：認証の交換と共存**。まず合成ProviderでOIDC相当と既存認証相当の選択・主体対応付け・セッションを試し、その後に本物のIdPとWebブラウザで検証する。合成試験の成功を実IdPや本番セキュリティの成立としない。

**採用判断**。既存#951と#1016の契約だけで足りなかったWeb固有の最小要素を整理し、必要ならADRに移す。新しい固定Capability enum、第二のProvider Resolver、独立したLock、巨大な汎用処理エンジンは初期の非目標とする。

## 10. 採用前に残る論点

- 処理点の「効果」と「短絡時の後処理」をどの粒度で型付けすれば、Go Handlerの実装負担を増やさずに安全性を検証できるか。
- テナント別・経路別の認証方針をEnvironment設定へ置く範囲と、Projectが所有する利用者要件の境界。
- Proxy/CDNが終端するTLS、信頼済みForwarded Header、Rate Limitと、Web Host内部で実際に強制する保護の境界。
- OIDC・SAML・Passkey・mTLSを標準実装する優先順位。初版で必要なProvider種類を実案件の要件から決める。
- 動的な設定更新が必要な場合の、構成版の原子的な切替え、既存セッションへの影響、ロールバック。
- ブラウザ実行、デスクトップWebView、モバイルで共有する契約と共有しない信頼境界。

## 参考

- [RFC 10017：ブラウザ向けOAuthアプリケーション](https://www.rfc-editor.org/rfc/rfc10017.html)。BFF、トークン仲介型、ブラウザ単独型の比較。
- [RFC 9700：OAuth 2.0のセキュリティ](https://www.rfc-editor.org/rfc/rfc9700.html)。
- [OWASP：セッション管理](https://cheatsheetseries.owasp.org/cheatsheets/Session_Management_Cheat_Sheet.html)、[CSRF対策](https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html)、[認可](https://cheatsheetseries.owasp.org/cheatsheets/Authorization_Cheat_Sheet.html)。
