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

## E2.1：認証済み主体のAction引渡し（2026-10-10）

E2の参照用Go Hostに、`PrincipalFromContext(ctx)`を追加しました。Hostは要求の経路、セッション、Issuer、CSRF、Action認可を検証し、すべて通過した要求についてのみ、パッケージ非公開のcontext keyを使って認証済み`Principal`をAction Handlerへ渡します。ブラウザから受け取るヘッダー、クエリ、JSONにこのcontext keyを設定する入口はありません。

`Principal`には`ProviderID`、`Issuer`、`Subject`、`PrincipalID`を残します。**異なる認証元が同じ`PrincipalID`を持っていても、発行元と元の主体識別子を消して一つの認証事実へ畳みません。** また、セッション失効時に別の認証方式へフォールバックする処理は設けません。

**重要な未接続境界：** `PrincipalFromContext`から得られる値はHostが検証した認証結果であって、業務権限ではありません。`PrincipalID`を`legacyauth.AuthenticatedSubject.SubjectID`と等しいと仮定して`RunCommand`へ渡したり、架空のCredentialを渡したりしません。既存`security.authorize` Capabilityによる業務認可を正本として維持します。現在の`go-host`はその既存Capabilityを実際には呼び出していません。`Dependencies.Authorize`は合成テスト用の注入関数です。

### 局所試験の記録

main統合後に残っていた二つの変更を、2026-10-10時点のmainを起点とする後続ブランチ`work/issue-1043-web-principal-handoff`へ移し、異なる認証元の出所保持テストを加えました。GitHub Blob SHAと一致する次の二つのGoファイルを隔離環境へ配置しました。

- `go-host/host.go`: `25cfcee8857193cc052f0e70a5d68321449ccb65`
- `go-host/host_test.go`: `d1abb5db50ec523c61a784c8a5dad30c1779c93c`

Go 1.23.2、`module tei`の隔離環境で`gofmt -l`（出力なし）、`go vet`、`go test -count=1`、`go test -race -count=1`をいずれもPASSと確認しました。既存の経路・監査・認可拒否試験に加え、認証済み主体だけがAction Contextへ渡ること、Clientが主体ヘッダーを偽装しても保護を迂回できないこと、失効したセッションでActionを呼ばないこと、異なる二つの認証元の主体情報が保持されることを検証しています。Go Hostは本番用認証や実Cookie、実CSRF検証、実IdPを提供するものではありません。

**統合資格は未取得です。** この局所試験はリポジトリ全体の`tools/repository-health-check.sh`、既存First Web Targetの実際の`RunCommand`／`security.authorize`結合試験、既存E1 Nodeテストの実行を代替しません。mainへの追加統合前に、後続PRのexact headで必須の全体検査が必要です。

## E1追試と依存経路検査の強化（2026-10-10）

`plan-check.mjs`と`plan-check.test.mjs`のGitHub Blob SHAが一致するファイルをNode.js v22.16.0で直接実行し、従来の14ケースすべてがPASSしました。ここでE1に四つの負例を追加し、最終的に`node --test reference/web-composition/plan-check.test.mjs`で18ケースPASS（fail 0）を確認しました。

- `plan-check.mjs`: `dcd8eeec613e6bc5deff28717c5720310a6e75f3`
- `plan-check.test.mjs`: `145ea9d9a69426265939bfde48331967c3c32f34`

従来は、必要な`guards`をどこかの処理点が宣言していれば、Actionの実行より後ろや無関係の処理点でも構成検査を通る余地がありました。今回、Actionの`invoke`から`requires`の契約依存を逆にたどり、**必須Guardを宣言した処理点が、その依存経路上の先行処理であること**を検査します。`before`や`after`による表示上の順序指定だけでは、Action保護の証拠として採用しません。Audit処理だけが認証Guardを宣言する構成、Action自身がCSRFを宣言する構成、CSRFを順序指定だけで先行させる構成、認証結果を使わずActionを呼び出す構成を拒否します。

**検証範囲の限界は従来どおりです。** この検査は解決済み処理点の自己申告と依存関係の静的整合性を確認するのみです。Hostによる実保護、条件分岐を含むすべての実行経路でのGuard実行、実Providerの真正性、Actionの業務認可成立は保証しません。正式な製品規則へ採用するには、既存のCapability/Provider適合性と実HTTP経路に結び付けた追加検証が必要です。

## E2.2：既存の業務認可Capabilityへの接続（2026-10-10）

`go-host/canonical_authorization.go`に、`NewCanonicalActionAuthorizer`を追加しました。`NewHost`が要求する`Dependencies.Authorize`へ直接渡せる関数を作り、既存の`driver.Resolver.InvokeCapability`から`security.authorize`を実行します。認可を独自のWeb側ルールで判定せず、Capabilityの`allowed`／`denied`というBusiness Resultと実行エラーを区別します。これは既存のAdapter／Bindingを再利用するための参照接続であり、第二のProvider Resolverではありません。

処理前に、対象が既存ApplicationのCommand Patternであることと、`security.authorize`のBindingが解決できることを検査します。要求されたActionが構成済みのCommandと異なれば拒否します。主体対応付けはHostが管理する`SubjectLinkLookup`へ委譲し、`ProviderID + Issuer + Subject`の組から、Applicationの`PrincipalID`と既存認可Storeが参照する`SubjectID`の**両方**を取得します。セッションが提示する`PrincipalID`と対応付け先が一致しない場合は、設定または信頼境界の不整合として失敗させます。メールアドレスや表示名から主体を推測しません。

`go-host/canonical_authorization_test.go`では、実Repositoryの`reference/workspace`、`reference/legacy-auth/users.json`、`reference/legacy-auth/authorization.json`を読み込み、既存Driver・Capability・Adapterを通す結合試験を定義しました。試験項目には企業認証相当とlegacy認証相当の編集者許可、閲覧者の業務拒否、未登録・失効した対応付け、セッション主体との不一致、対応付け先の障害、未知Command、必須接続の欠落を含みます。どの試験もIdPや実Sessionの真正性を証明するものではありません。

### E2.2の確認範囲

GitHub Blobと一致するローカルコピーに対し、Go 1.23.2の`gofmt`による書式確認とGo構文を保つ整形を行いました。

- `canonical_authorization.go`：`a2174e1d7ed4d305733dac294770cde451c223f9`
- `canonical_authorization_test.go`：`33ae2a6f27a0810de6e7f1529a95ef1f44f33298`

**新しい結合試験を実リポジトリで実行した証拠は未取得です。** これら二つのファイルを含むパッケージの`go test`・`go vet`・`go test -race`の成功はまだ主張しません。前節の局所試験PASSは、その時点の二ファイルだけを対象とした結果であり、今回の追加分には継承されません。

`SubjectLinkLookup`の実体は未実装です。実運用向けには、対応付けの管理権限、現在の有効性・失効・無効化、tenant／Application境界を検査できる信頼済みStoreが必要です。特に既存`legacyauthorization.Adapter`は業務roleを評価するものであり、認証済み主体のアカウント状態・再認証・失効を代行するものではありません。

また、現行`commandScenarioRunner.RunCommand`はCredentialsを要求し、自身で`security.authenticate`と`security.authorize`を呼び出します。この新しい認可コールバックは**そのRunnerをまだ呼び出していません**。既存Runnerの業務Validation・Command・Observationを維持するためのセッション専用入口を接続することが次の課題です。架空のCredentialを渡して旧経路を通す実装にはしません。

## E2.3：資格情報を再送しないSession専用Runner（2026-10-10）

既存First Web Targetに`RunVerifiedSessionCommand`を追加しました。従来の`RunCommand`は`loginId`と`credential`を使って`security.authenticate`を呼ぶ互換経路のまま残します。新しい入口は`webcomposition.PrincipalFromContext`が返す**Web Hostで検証済みのPrincipal**を必須とし、既存の`SubjectLinkLookup`を通して`PrincipalID`と業務`SubjectID`の対応を要求します。暗黙のlegacy認証へのfallbackや、架空のCredentialの注入は行いません。対応付けなしは`identity-link-denied`、対応付け異常・Lookup障害はExecution Errorと区別します。セッション認証を、実行していない`security.authenticate` CapabilityのBusiness Resultとして偽装しません。

`command_scenario.go`の旧`RunCommand`は認証に成功した後、共通の`runAuthenticatedSubjectCommand`へ進む構成に変更しました。Session経路も同じ共通処理を利用します。Read → Canonical Validation → **既存`security.authorize` Capability** → Command → Observationを一切複製せず、業務認可をFinal Action境界で再評価します。共通化前後の業務処理164行を比較し、参照する入力変数3箇所以外のテキストが**完全一致**することを確認しました（比較対象の旧Blob `ba1b5426334128448256531508ca97891f3f30da`、新Blob `a2e16ee371d4bcb51b1188f2327a6de70092a8ba`）。

`tools/reference-first-web-target/session_command_scenario_test.go`には、合成の企業認証／従来認証から同じ既存業務処理を通す正常系、既存業務規則による閲覧者の拒否、リンク未登録、Principal不一致、リンクStore障害、未検証Contextからの直接呼出し拒否を追加しました。Host側の早期認可だけに依存せず、Runnerでも既存Capabilityを呼び直すことを試験します。

### 確認済みの範囲と未実施

- `session_command_scenario.go`のGitHub Blob: `e4a497374527ad7ba346d6af82ef4cbbfd1142b5`
- `session_command_scenario_test.go`のGitHub Blob: `ce5806d7fa80f022312991df557ad452d194dfc8`
- この二ファイルは上記Blobと完全一致するローカルコピーで`gofmt -l`が空、Go 1.23.2のParserによる構文確認がPASS。
- E2.2の`canonical_authorization.go`もGitHub Blobと完全一致するローカルコピーを用い、**stub Driver／Binding／Capability Result**による隔離Go契約ハーネスで`go vet`、`go test -count=1`、`go test -race -count=1`がPASS。Session Runner本体についても**stub共通実行処理／Principal Context**による同条件の隔離Go契約ハーネスでPASSしました。これらはテスト補助依存を差し替えた限定的な確認です。
- **実際のReference Workspace／Driver／Command Scenarioを読み込む新結合試験は未実行です。** 既存旧認証経路の回帰を含む実Repositoryの`go test`、`go vet`、`go test -race`、exact-head `tools/repository-health-check.sh`のPASSは取得していません。

**公開経路も未切替えです。** 現在の`webTargetServer.Handler()`の`/scenario/submit`は旧Credential方式のままです。Session方式は本Referenceの`NewHost`を手動で構成した合成テストによる専用入口であり、実IdP、Session Cookie、CSRF Provider、ブラウザ、productionを動作確認したものではありません。本番相当の経路を公開する前には、旧入口との到達可能性の整理、信頼済みProvider／Subject Linkの実装、Tenant境界、account無効化・期限・失効検査、既存Bindingへの適合確認が必要です。
