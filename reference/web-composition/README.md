# Web要求経路の構成検査（E1候補）

**状態：読み取り専用の試作。製品の認証・認可・HTTP保護を実装したものではありません。**

担当課題は[#1043](https://github.com/hat47x/tei-platform/issues/1043)です。設計の背景と既存機構との分担は[Webアプリケーション構成キットの設計候補](../../plan/design/WEB_APPLICATION_COMPOSITION_DESIGN.md)を参照してください。

## 目的

プラグインを組み合わせるとき、HTTP要求の実行前に次の問題を発見できるかを確かめます。

- 必須保護（本人確認・要求元保護・業務操作の認可）の不足。
- 処理の依存関係・実行順序の循環。
- 排他的なミドルウェア接続点の衝突。
- 正常終了以外の拒否・障害について監査処理が宣言されていない状態。
- ブラウザ側の処理をサーバ側の保護として扱う間違い。
- 複数認証方式の共存に伴う主体の誤対応、弱い方式への暗黙の切替え。

## 内容

- `plan-check.mjs`：**既に採用済みの処理点**を入力として、依存グラフと保護条件を検査し、決定的な順序を作ります。同じ名前の機能を提供する実装が二つあれば拒否します。
- `plan-check.test.mjs`：保護された受注操作、認証主体の対応付け、保護の欠落、循環、衝突など12件の正常・異常例を固定しています。

既存の#951のProvider解決は行いません。処理点は**解決済みであるものとして受け取る**ため、この試作が返す順序は実行権限やProviderの適合性を保証しません。

## 実行方法

~~~sh
node --test reference/web-composition/plan-check.test.mjs
~~~

2026-10-09、構成検査と認証設定検査の関数については、V8隔離実行環境で同等の12例を直接評価し、12件が想定どおり成功／拒否されました。**上記のNodeコマンドをリポジトリの実ファイルに対して実行した記録は未取得です。** さらにTEI全体のヘルスチェック、Go HTTPサーバとの接続、実ブラウザ、実IdP、Cookie・CSRF保護の動作試験は行っていません。

## この試作が保証しないこと

`guards`や`onTerminal`は処理点の**宣言**です。実際に要求元検証が行われたこと、拒否時に監査が実行されたこと、認証Providerが真正であることの証拠ではありません。これらを保証するには、信頼できるHostによる処理の強制と、Provider適合性の検査、実HTTP動作試験が別に必要です。

`hostPolicy`と`hostContracts`は信頼済みHostが与える前提です。未信頼の案件拡張やブラウザから自由に指定できる製品APIにしてはなりません。この試作では受注操作を公開経路へ変更する構成を一律に拒否していますが、一般的な公開Actionの採否を製品全体で決めたわけではありません。

認証の`subjectLinks`は、発行元・主体識別子に基づく明示的な対応付けの書式だけを検査します。ログインの実行、OIDCの暗号学的検証、本人同意、リンク設定の管理権限は対象外です。

## 次の検証

1. 実際のNodeテストと、正確なブランチ先端に対するTEI全体のヘルスチェックを実施します。
2. 既存First Web Targetの`http.ServeMux`と`security.authenticate`/`security.authorize`の間に、保護条件を一つずつ接続します。別のProvider Resolverを作らず、#951の契約へ戻します。
3. 拒否・例外・短絡を含むHTTP試験で、保護と監査が**実際に呼ばれたこと**を検証します。
4. 合成OIDC相当と既存認証相当を共存させ、セッションと業務認可を別に確認します。その後に実IdPを使った検証へ進みます。

本試作を正式なTEI言語・Plugin ABI・Web標準に採用するには、上記の検証結果から必要な最小機構を抽出することが前提です。

## E2：Go製HTTP Hostの最小参照実装

`go-host/host.go`は、`net/http`の一つの保護されたPOST経路を構成する試作です。既存First Web Targetは変更せず、実際のHTTP処理に必要な保護の呼出しを、解決済みのHandlerへ接続する前段として取り出しました。

処理順序は、**正確なメソッド・経路の検証 → 信頼済み検証器によるセッション確認 → 認証元と主体の確認 → 要求元保護（CSRF） → 対象Actionの認可 → Action Handler**です。必須機構を渡さない場合は構成を拒否し、拒否・障害・早期終了についても監査関数を呼びます。採用した認証発行元は構成時にコピーし、外部から渡された可変のmapを書き換えても変更できないようにしました。

認証元ごとの主体IDと、業務上のPrincipal IDを区別します。別々の認証方式が同じPrincipalへ結び付くことは許容しますが、その対応付けは本試作が自動推測するものではありません。本人確認、セッション、CSRF、認可の実体はテスト用関数で与えます。**単にHTTP要求ヘッダーを見て本人確認を成立させる実装を標準として提案するものではありません。**

### 検証記録（2026-10-09）

GitHubに登録した次の二つのファイルと、Git Blob SHAが一致するローカルコピーを使用しました。

- `go-host/host.go`: `b66d032481b7cf29bf2dd89aa7bbd23d7b25b07d`
- `go-host/host_test.go`: `008dc9a306f9f7b432915ff0b4edfbfe512e9636`

ローカルの隔離されたGoモジュール（`module tei`、Go 1.23指定）で、以下を実行し、いずれも成功しました。

~~~sh
gofmt -l reference/web-composition/go-host/host.go reference/web-composition/go-host/host_test.go  # 出力なし
go vet ./reference/web-composition/go-host
go test -count=1 ./reference/web-composition/go-host
go test -race -count=1 ./reference/web-composition/go-host
~~~

テストは、許可された二方式での正常なAction呼出し、未認証、未知の認証元、Issuerの不一致、CSRF不成立、権限拒否、認可先の障害、経路・メソッドの不一致、エンコードによる経路の別名、利用者入力のロールを使った権限迂回、構成後のIssuer書換えを扱います。また`httptest.NewServer`を用い、ローカルHTTP接続で拒否と成功を通しました。

**証明できたこと:** この二つのGoファイルが、合成のProviderとAction Handlerを使い、失敗時にActionを実行せず、正常時に認可後だけ実行すること。

**証明していないこと:** 既存TEI実装との結合、実IdPの本人確認、実セッション・Cookie・CSRF対策の安全性、逆プロキシや実運用の異常系、レスポンス書込み後のpanicからの完全回復、ストリーミングと任意の`http.ResponseWriter`拡張への対応、`tools/repository-health-check.sh`を使った**リポジトリ全体の資格確認**。このReferenceを本番利用することはできません。

### 次の接続

最優先は、既存First Web Targetの`/scenario/submit`と`security.authenticate`/`security.authorize`に対して、実Hostが保護機構を強制する接続を検証することです。E1の宣言をGo実行時の保護の証拠に読み替えず、既存#951のProvider解決・Lockを再利用し、認証結果とApplication Actionの意味契約を変えない条件で進めます。


## 既存First Web Targetとの接続に先立つ境界確認

`tools/reference-first-web-target/server.go`の現行`/scenario/submit`は、HTTP本文に`loginId`と`credential`を受け取り、`commandScenarioRunner.RunCommand`へ渡しています。これに対し本Referenceの`NewHost`は、信頼済みセッション検証器が返す`Principal`を使用します。この二つを、単にミドルウェアで囲むだけで同一の認証方式とみなしてはなりません。

**接続に必要な最小契約**は、(1) Webの認証済み主体をTEI Principalへ明示的に対応付ける、(2) 既存の`security.authenticate`と新しいWeb認証を二重に成立させたことにしない、(3) Principalを信頼済みのサーバ内部経路だけでAction境界へ渡す、(4) 業務Actionの`security.authorize`を実行主体ごとに強制する、(5) 既存の`RunCommand`の意味・Business ResultとExecution Errorを維持する、の五点です。

この契約を満たす既存のRunner APIがあるかは未確認です。現在の`commandScenarioInput`には`LoginID`と`Credential`があるため、これらをPrincipalへ安易に置き換えたり、固定値のCredentialを挿入したりする変更は行いません。次段階ではRunnerの内部、Driver、Capabilityの呼出し境界を追跡し、既存の権限モデルと整合する最小のアダプタを特定します。

また、現行`webTargetServer.Handler()`が`/scenario/submit`を登録するため、新しい保護済みHandlerと併用する際には**同一経路を二重に公開しない**ことが必要です。既存Handlerの上から認証だけを付け足す方式では、迂回経路が残り得ます。公開経路の所有者はHost側で一意に決定し、既存経路の互換性は明示的な移行条件として検証します。

2026-10-09時点では、この接続契約は調査結果に基づく候補であり、既存Web Targetへのコード変更や結合テストは未実施です。

## E2接続調査：実行入口の確認

`tools/reference-first-web-target/main.go`を確認すると、`order_entry`では`newReferenceCommandScenarioRunner(loaded.World, legacyUsers, legacyAuthz)`を生成し、`newWebTargetServer`へ渡しています。RunnerはWebサーバの構成時にApplication ID一致を確認されます。一方、`server.go`の`handleCommandScenario`は`commandScenarioInput{LoginID, Credential, From, RowIndex}`を生成して`RunCommand`へ渡し、結果と実行エラーを区別してHTTP応答に写しています。

したがって、セッション認証方式を導入するときの変更点は**単なるHTTP経路の保護だけではなく、Runnerの入力契約と内部の本人確認・権限判定の分離**にあります。従来の`RunCommand`へ架空のCredentialを渡して認証を通過させる方式は採用しません。既存RunnerがPrincipal入力を受け取れるか、Runner内部で既存の`security.authenticate`と`security.authorize`をどこで呼び出しているかは、さらに実装の追跡が必要です。

構成上は、旧来の認証情報を含む経路と、セッションを用いる保護済み経路を**暗黙に同一視しない**ことを受入条件とします。移行期間に両者を併存させる場合も、公開する経路・認証方式・許可されるAction・監査対象をそれぞれ明示し、認証失敗時に弱い方式へ自動的に切り替えません。

現段階ではRunnerを改造しておらず、実際の認証方式の置換・共存は未検証です。

## E2追跡：Runner内部の認証・認可

`tools/reference-first-web-target/command_scenario.go`を確認しました。現行`RunCommand`は、入力検査の後に`driver.Resolver.InvokeCapability`で`security.authenticate`を呼び出します。Binding `order_entry/security.authenticate`には`legacyauth.Adapter`を接続し、`LoginID`と`Credential`を渡します。認証成功時の`legacyauth.AuthenticatedSubject.SubjectID`を取得し、読み取りとCanonical Validationを実行した後、同じ`InvokeCapability`で`security.authorize`を呼び出します。Binding `order_entry/security.authorize`には`legacyauthorization.Adapter`を接続し、`SubjectID`、操作`execute`、対象`order_entry.submit_order`を渡します。許可時のみ`sales.order.submit`を呼び出し、その後に合成状態の観測を行います。

この順序には重要な意味があります。本人確認と業務認可は別Capabilityであり、業務認可の拒否は`authorization-denied`というBusiness Result、Provider実行エラーはExecution Errorです。新しいWeb Hostの`Authorize`だけを実行して既存の`security.authorize`を省略する変更は、権限判定の二重正本を生みます。

**次の実装の受入条件：**

- 既存のCredential入力方式は互換経路として維持し、Webセッション方式とは別の明示的な入口を持つ。移行先で旧経路が無保護の迂回路として残らないこと。
- Webセッションの`PrincipalID`を`legacyauth.AuthenticatedSubject.SubjectID`と同一と決めつけない。Provider・Issuer・Subjectの対応付けと、既存の認可Storeで参照できる業務主体への写像を、信頼済みHostが検証すること。
- 業務認可は既存の`security.authorize` Capability経由を正本とする。Hostの経路保護はその代替ではない。必要な場合はHost側の早期拒否とAction側の最終認可を区別すること。
- 認証済み主体の受渡しには、ブラウザから偽装できるヘッダや本文の`subjectId`を使用しない。呼出し元が認証済みであることを保証する内部APIを検討すること。
- Validation、Command、Observationの順序と、Business Result／Execution Errorの区別を維持すること。失敗した認証方式から別方式へ暗黙にフォールバックしないこと。

ここで確認できたのはコード上の接続順序であり、Webセッションから既存認可Storeまでの主体対応付けは未実装です。したがって、認証方式の共存を動作確認済みとは扱いません。
