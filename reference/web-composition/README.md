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
- `plan-check.test.mjs`：保護された受注操作、認証主体の対応付け、保護の欠落、循環、衝突に加え、Guardの因果依存・証拠契約・プラグイン拡張など33件の正常・異常例を固定しています。

既存の#951のProvider解決は行いません。処理点は**解決済みであるものとして受け取る**ため、この試作が返す順序は実行権限やProviderの適合性を保証しません。

## 実行方法

~~~sh
node --test reference/web-composition/plan-check.test.mjs
~~~

2026-10-09、構成検査と認証設定検査の関数については、V8隔離実行環境で同等の12例を直接評価し、12件が想定どおり成功／拒否されました。**上記のNodeコマンドをリポジトリの実ファイルに対して実行した記録は未取得です。** さらにTEI全体のヘルスチェック、Go HTTPサーバとの接続、実ブラウザ、実IdP、Cookie・CSRF保護の動作試験は行っていません。

### E1.2：Guardの証拠契約と認証定義の一意性（2026-10-10）

「必須Guardが`invoke`の因果的先行処理点にある」という検査だけでは、`authenticated`等のラベルを、本人確認を実行しない`request.context`取得処理に貼っても受理される余地がありました。これを防ぐため、標準Guard名と証拠となる`provides`契約の組を照合します。

| Guard | 証拠契約 |
| --- | --- |
| `authenticated` | `identity.principal` |
| `csrf-protected` | `csrf.checked` |
| `action-authorized` | `action.authorized` |

標準Guardの証拠契約はプラグイン側から上書きできません。追加のGuardを必要とするHost Policyは`hostPolicy.guardEvidence`で`Guard名 → 証拠契約`を明示します。例えば`tenant-isolated → tenant.boundary.checked`を宣言し、実際の先行Stepがその契約を`provides`し、`invoke`が`requires`するときだけ、構成上の条件を満たします。単なる`before`の順序指定では不十分です。

また、認証設定では空のProvider/Policy選択、同じPolicy IDの二重定義、同一`ProviderID + Issuer + Subject`のリンク二重定義を拒否します。異なる認証元から同じApplication Principalへ明示的に対応付ける構成は維持します。

**局所テストの確定記録：** Node.js `v22.16.0`を使い、GitHub上の次のBlobと**byte一致する実ファイル2件**を隔離ローカルディレクトリへ配置しました。

- `plan-check.mjs`: `b677347f66acd14e5c47d62afa12cfc263d08e46`
- `plan-check.test.mjs`: `19d971ba6a185f41b783af63143d3205d86fb986`
- `node --test plan-check.test.mjs`: **28 PASS / 0 FAIL**（元の18件と、新規の認証・Guard検証10件）
- `node --check plan-check.mjs`: PASS

これは静的な構成検査であり、Guardに対応したProviderが実際に暗号学的検証・CSRF・業務認可を正しく実行する証拠にはなりません。特にプラグイン定義の`guardEvidence`は**合成された宣言上の制約**であり、実体の適格性は将来のHost / Plugin Conformanceで確認する必要があります。実Repository全体の`tools/repository-health-check.sh`は未実行のため、本PRのDraft維持・統合保留は変わりません。

### E1.3：Actionに必須の保護とProvider主体のスコープ（2026-10-10）

E1.2ではHost Policyが指定したGuard名と証拠契約を検査しますが、Host Policy側が必要な`authenticated`または`csrf-protected`を**そもそも省略した**場合は、保護Actionの構成が成立してしまう抜け道が残っていました。E1.3では`route.actionId`がある場合、信頼済みHost Policyであっても`authenticated`・`csrf-protected`・`action-authorized`の3 Guardと、監査上の`completed`・`rejected`・`error`の3終端を必須とします。Providerによる実行保証とは別の、計画を誤って弱めないための最低条件です。

認証主体リンクの重複判定キーも、Go Hostの既存`ExternalSubject`契約と同じ`(ProviderID, Issuer, Subject)`に統一しました。異なるProviderを明示しているなら、IssuerとSubjectの文字列が同じでも独立した対応付けとして扱います。全く同じ外部主体の二重定義は引き続き拒否します。

**実行済みNodeテスト：** GitHub上の`plan-check.mjs`（Blob `563f429474a1de65136bb6ae7d477f100ab6cab8`）と`plan-check.test.mjs`（Blob `48772184fc60e40df9df5390eaa32974527a3e56`）のbyte一致コピーで、Node.js 22.16.0の`node --test`を実行し、**33件PASS、0件FAIL**でした。以前の28件に5件（認証Guard省略、CSRF Guard省略、終端監査省略、Provider名前空間の独立性、同一主体のPrincipalID衝突）を追加しています。これは静的構成検査の証拠であり、実HTTP Hostや実Providerの適合性の証拠ではありません。

### E1/E2手動横断検証コマンド

`reference/web-composition/verify.sh`は、Web構成の静的Node検査と、Web Host・First Web Target・Legacy User StoreのGo package検査をひとつの**手動実行コマンド**にまとめます。

~~~sh
bash reference/web-composition/verify.sh --race
# mainとの統合資格には、上記とは別にfull repositoryで次も必須
tools/repository-health-check.sh
~~~

スクリプトは`node --test`、対象3 packageの`go vet`、`go test -count=1`を実行し、`--race`指定時のみ`go test -race`を追加します。必要なNode/Goがない、あるいはフルリポジトリの`go.mod`と`plan-check.test.mjs`がないときは、結果を成功とみなさず停止します。**GitHub Actionsの実行・再開は行いません。** Nodeが入っていないHostではこの追加検証はPASSと扱えません。

2026-10-10、GitHub Blob `ccc2de2e6a8cdbe1781ece3b5564a1dccd67186a`と同一の`verify.sh`について、ローカルで`bash -n`、`--help`、不完全なcheckoutでの拒否を確認しました。ただし**実Workspaceを含むリポジトリ全体でこのスクリプトを実行した証拠はまだありません**。これは全体ヘルスチェックの代替ではなく、E1 Node回帰検証の実行漏れを防ぐための補助です。

## この試作が保証しないこと

`guards`や`onTerminal`は処理点の**宣言**です。実際に要求元検証が行われたこと、拒否時に監査が実行されたこと、認証Providerが真正であることの証拠ではありません。これらを保証するには、信頼できるHostによる処理の強制と、Provider適合性の検査、実HTTP動作試験が別に必要です。

`hostPolicy`と`hostContracts`は信頼済みHostが与える前提です。未信頼の案件拡張やブラウザから自由に指定できる製品APIにしてはなりません。この試作では受注操作を公開経路へ変更する構成を一律に拒否していますが、一般的な公開Actionの採否を製品全体で決めたわけではありません。

認証の`subjectLinks`は、発行元・主体識別子に基づく明示的な対応付けの書式だけを検査します。ログインの実行、OIDCの暗号学的検証、本人同意、リンク設定の管理権限は対象外です。

## 次の検証

1. Nodeテストは実ファイル33件PASSを取得しました。次に、正確なブランチ先端に対するTEI全体のヘルスチェックを実施します。
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

## E2.4：既存Web Serverの保護されたSession経路へ接続（2026-10-10）

First Web Targetの`newWebTargetServer`に、信頼済みHostだけが構成する任意の`withProtectedSessionCommand`を追加しました。設定された場合にのみ`Handler()`は**POST `/scenario/session-submit`**を登録します。Host側で`NewCanonicalActionAuthorizer`と`NewHost`を必ず構築するため、呼出し元が未保護の裸の`http.Handler`を挿し込む経路にはしていません。セッション検証、CSRF検証、`ProviderID → Issuer`の許可一覧、信頼済み主体リンク、監査、同一Applicationの既存Command Runnerが欠ける構成は起動時に拒否します。

旧**POST `/scenario/submit`**は従来の`loginId`/`credential`入力と`security.authenticate`実行を維持します。新しいSession経路のJSON入力は`from`と`row`だけです。`loginId`、`credential`、`subjectId`等の未宣言フィールド、複数JSON文書、負の行番号は拒否します。新旧経路のBusiness ResultとExecution ErrorのHTTP応答・Provenance記述は`writeCommandScenarioHTTPResult`で共通化し、Web側に別の意味変換表を作りません。新経路は、Hostの認可とRunner内の既存`security.authorize`の両方を通過して初めてCommandへ進みます。

`tools/reference-first-web-target/session_command_route_test.go`に、既存`reference/workspace`、`reference/legacy-auth`、実`newWebTargetServer`と`Handler()`を利用する試験を追加しました。合成の二つの認証元からの受注完了、旧Credential経路の互換、Session経路未設定時の404、設定欠落時の起動拒否、CSRF欠落、既存業務規則による閲覧者の拒否、JSON経由の資格情報・主体IDの注入拒否、監査記録を検査します。テスト用`X-Synthetic-Session`ヘッダーは**あくまで合成Providerのテストデータ**であり、本番認証手段として提供しません。

### E2.4の検証範囲

- `session_command_route.go`: GitHub Blob `cde1faf11b220379d7670e6cc431b119907611fa`
- `session_command_route_test.go`: GitHub Blob `f56010fb94b053b8c73200c9ed0475aef1102fe0`
- この二ファイルは、GitHub Blob SHAとローカルファイルの`git hash-object`が一致することを確認しました。`gofmt`実行後のファイルは変更なし、Goの構文処理に必要な形式を満たしています。
- 実Workspaceへの依存があるため、**追加したE2.4結合テストの実行結果は未取得**です。旧Credential経路と新Session経路の`go test`、race、vet、actual full checkoutでの全体Repository Healthの成功は主張しません。
- 同じPRへの並行変更を監視しており、2026-10-10時点のmain更新が対象とした`plan/issue/001067-cross-project-plugin-promotion.md`は本PRの変更ファイルと重複しません。ただしheadが最新mainの1コミット後方なので、統合時の資格判断には最新mainを含むexact head再検証が必要です。

これは、本番配備時に利用できるSession管理、実IdP検証、Secure Cookie、CSRF Token発行・検証、subject-link管理・失効、Trusted Proxyの検証を実装したものではありません。新経路は明示的にHost構成した場合に限るReference機構であり、現行`main.go`の通常起動経路では未選択です。安全性宣言と実証を混同せず、導入前には実Provider・ブラウザを使うE3検証が必要です。

## E2.4追試：曖昧な経路構成とJSON入力の拒否

追加の負例で二つの問題を修正しました。

- Session経路を同じWeb Serverへ二重に宣言した場合、最後に宣言された設定で黙って上書きせず、Hostの構成自体を拒否します。処理チェーンのセキュリティ設定を登録順に依存させないためです。
- `encoding/json.Decoder`はstructへの`null`入力をエラーにしないため、Session経路では`*webSessionCommandRequest`として受け、nilの場合は400で拒否します。これにより、暗黙の既定値だけでCommandへ進むケースを防ぎます。

GitHub Blobとbyte一致するローカルコピーで`gofmt -l`（出力なし）およびGo構文検査を行った対象は次のとおりです。

- `session_command_route.go`: `a24077ce7a532012ad321f304498ef9468e88030`
- `session_command_route_test.go`: `ba68f1f49a7fdeac4343945fb875f0e56c363b97`

この二つの**実際の新規ファイル**と、必要な他パッケージを置き換えた一時的な隔離Goモジュール（Go 1.23.2）を使い、`go test -count=1`、`go vet`、`go test -race -count=1`がPASSしました。認証不成立、CSRF失敗、閲覧者の業務拒否、本文からの資格情報／Subject ID注入、二つのJSON文書、`null`、構成欠落、二重宣言等のテストコードを実行しています。

**重要な証拠の限界:** この隔離ハーネスは`workspace.Load`、`webTargetServer`、認証Host、`security.authorize`、既存Runner等をスタブに差し替えています。新規ルートのGo構文と制御分岐は確認できますが、旧Web Target本体やCanonical Resolver、実`reference/workspace`との結合成功の証拠ではありません。統合には、実Repositoryのexact headをクリーンにチェックアウトし、既存および新しいすべてのGoテストと`tools/repository-health-check.sh`を実行する必要があります。未完のためPRはDraftです。

## E2.5：セッション連携時のアカウント状態・失効の再検証

`security.authorize`が評価する既存の業務操作権限と、認証済みアカウントの有効性は別です。既存`legacyauthorization.Adapter`はRoleによる業務許可を扱いますが、Session経路を通ってきたAccountの`disabled`、`valid_from`、`valid_until`、`locked_until`をそれだけでは判断しません。従来のCredential経路が`security.authenticate`で検査していたAccountの状態を、セッション経路で無条件に省略すると危険です。

そこで`go-host/subject_lifecycle.go`に`NewActiveLegacySubjectLinks`を追加しました。Hostが管理する明示的な`ProviderID + Issuer + Subject → PrincipalID + SubjectID`の対応付けに加え、既存の`legacyauthorization.SubjectStore`を`SubjectID`で引き直し、現在のアカウント状態を検査します。未知・無効・期間外・ロック中のアカウントは認可へ進めません。対応付け・Storeの障害、SubjectID不整合は例外として区別し、権限付与を行いません。Roleの判断は引き続き正本の`security.authorize`です。

Web Server側は`AccountNow`という明示的なClock依存を必須にし、**Hostの早期認可時**と**Command実行直前の主体リンク再検証時**の両方でAccountの適格性を確認します。既存のFixtureを使う合成テストではRunnerの固定時刻を注入しますが、これは本番の失効確認を実証するものではありません。本番では信頼済みの最新Account Storeと適切な時刻源が必要です。

追加のHTTP回帰テストは、業務Role上は受注実行権限を持つもののAccountがロック中の`subject-order-locked`と、無効な`subject-order-disabled`の拒否を確認します。さらにHostの早期認可後にAccount有効期限を越える場合と、主体対応付けStoreが実行直前に障害になる場合を試験します。この後段の失効はHTTP 403、Store障害はHTTP 503とし、CommandとObservationへ進みません。後段のStore例外はExecution Errorと区別しますが、バックエンド由来のエラーメッセージはHTTP応答へ露出させず固定文言に変換します。

### 局所検証と統合保留

- `subject_lifecycle.go`: GitHub Blob `5a99d308658718ebb27864ae3444cf0d6a5c34da`。
- `subject_lifecycle_test.go`: GitHub Blob `52d9501a58a47e6989a127851fcd7cd33c36778f`。
- **上記2ファイルとローカルコピーの`git hash-object`一致を確認**しました。Go 1.23.2の隔離`module tei`で、既存`legacyauth.User`／`legacyauthorization.SubjectStore`の公開型に合わせた最小スタブを使い、`gofmt -l`出力なし、`go vet`、`go test -count=1`、`go test -race -count=1`はすべてPASSしました。テスト内12個の正常・異常ケースと依存不足テストが通過しました。
- ただしこの隔離テストのStore・Userは互換スタブです。**実`reference/workspace`、`legacyauth.FixtureStore`、`driver.Resolver`、Web Server全体を利用した新しい結合テストのPASSは未取得**です。
- 主体リンクの二重検査、HTTP 403/503への変換、エラー文言の秘匿、ロック／失効のHTTP負例はコードとテストを追加しましたが、それらの**actual full repository実行は未実施**です。現行mainとPR headの差分を再確認し、`tools/repository-health-check.sh`全項目PASS、実行前後cleanをexact headへ結び付けるまでDraft維持・統合禁止です。

このReferenceは実IdP、Session Cookie、CSRF Provider、企業Directoryの現在性、分散Host間の失効伝播を提供しません。とくにAccount Storeが古いSnapshotである場合、Clockだけが最新でも本番の適格性は保証できません。

## E2.6：副作用直前の主体リンク再確認

E2.5ではHostの早期認可後にRunnerの入口でアカウント状態を再検証しました。しかしその後のRead・Validation・`security.authorize`を実行している間にも主体リンクが失効し得ます。**Runner入口での再検証だけでは、実際のCommand実行時点のauthorityを保証できません。**

このため、`command_scenario.go`に`runAuthenticatedSubjectCommandWithEffectGuard`を追加しました。旧Credential入口は従来どおり`nil` guardで共通業務処理を呼び、業務の意味と既存認証フローは維持します。Session入口だけ、認証元`ProviderID/Issuer/Subject`とセッション`PrincipalID`を閉包に保持し、**Command Capability呼出し直前**に信頼済みSubject Linkを引き直します。見つからない／失効したリンク、または変化した`PrincipalID/SubjectID`ではCommandもObservationも実行しません。既存の`security.authorize`は従来の流れの中で実行され、Web側に二つ目の権限ルールは作りません。

同じ`SubjectLinkLookup`はE2.5でAccount状態の現在性を検査するラッパーに包まれるため、Session経路は**Hostの早期認可 → Runner入口 → 副作用直前**の3つの境界でリンクとAccount適格性を確認します。実Store／Clockが古い場合まで保証するものではありません。

HTTP負例には、(1) Validationと業務認可完了後、Command直前のClockでAccountが期限外になる場合、(2) 同じタイミングでSource Linkが別の業務`SubjectID`に変わる場合を追加しました。前者はHTTP 403かつ`identity-link-denied`、後者は503かつ`identity-link` Execution Errorで停止する設計です。後者の内部対応関係はHTTPへ露出させず、固定文言に置き換えます。いずれもCommand／Observationが空で、監査記録が拒否／エラーとなることを要求します。

**検証状態：** コードと回帰テストをPRへコミットしましたが、これら追加分を含む実Workspace結合テストはまだ実行していません。以前の隔離GoテストはE2.5以前の状態を対象とするため、この変更に合格した証拠にはなりません。正確なPR headのクリーンなフルチェックアウトで`go test`、`go test -race`、`go vet`、`tools/repository-health-check.sh`を実行し、前後cleanを記録するまで統合禁止です。


### E2.6の局所契約試験（exact Blob）

Go 1.23.2の隔離`module tei`に、実際の`session_command_scenario.go`を**GitHub Blobとbyte同一**でコピーしました（`git hash-object=6e8f10ae6ba9929d1f11d5b1da0b161807b68333`）。`PrincipalFromContext`と、認可済み後段を実行するRunnerの内部関数だけをスタブに置換し、**このSession入口の実コード**が保持する再確認閉包を検査しました。実行結果は次の通りです。

- `gofmt -l tools/reference-first-web-target/session_command_scenario.go`：出力なし。
- `go test -count=1 -v ./tools/reference-first-web-target`：**2テスト、5サブケースを含めてPASS**。有効リンク、途中失効、`SubjectID`変更、`PrincipalID`変更、リンクStore障害、未認証Contextの拒否を確認。
- `go vet ./tools/reference-first-web-target`：PASS。
- `go test -race -count=1 ./tools/reference-first-web-target`：PASS。

また、GitHub上の`command_scenario.go`（Blob `ca37e6fd94e314658c8a7c66945842f6f95c60a4`）を静的に確認し、`security.authorize` → Command Pattern解決 → **effectGuard** → `resolver.InvokeCapability(sales.order.submit)`という順序であること、旧Credential入口が`nil` guardを通すことを確認しました。

**この結果を実Repository全体の結合テスト成功とは扱いません。** 共有ビジネス処理の本体、実`Driver`、実`workspace.Load`、Go Host／HTTPステータス／監査は隔離試験ではスタブまたは対象外です。今回追加した実WorkspaceのHTTP回帰試験はリポジトリ全体のクリーンなチェックアウトで実行する必要があります。


## E2.7：Legacy Subject ID重複による業務権限の曖昧さ

セッションからの`security.authorize`接続を追跡した結果、`internal/security/legacyauth/fixture_store.go`の潜在的な主体取違えを発見しました。Loaderは`user_id`重複を拒否していましたが、異なるログインIDが同じ`subject_id`を持つことを許していました。一方`FindBySubject`はGoのmapを線形走査し、一致した最初のユーザーを返していました。このため、同じ`subject_id`に異なる業務Roleが紐付く場合、呼出しごとに違うユーザーを返す可能性があり、既存の認可Capabilityの判定そのものが不定になります。

これを防ぐため、Reference Fixtureを読み込む際に**重複した非空`subject_id`を構成エラーとして拒否**し、`FindBySubject`はLoader構築時に検証した索引`bySubject`から一意に返すよう変更しました。Login IDによる`FindByLogin`は従来どおりです。空`subject_id`のFixture入力は既存互換上ロード自体を変更せず、Subject検索ではヒットさせません。既存`security.authenticate`は実行時に空SubjectをExecution Errorとして拒否します。一般的な企業ディレクトリの複数エイリアス対応を禁止する製品要件を新設したわけではなく、この参照用Storeに**明示されていない1:N対応を推測させない**ためのfail-closed実装です。

### E2.7の検証記録

2026-10-10時点のGitHub Blob SHAとローカルコピーの`git hash-object`が一致することを確認しました。

- `internal/security/legacyauth/fixture_store.go`: `3bced30c25242f402865e804cc0737f13c12e67a`
- `internal/security/legacyauth/fixture_store_subject_identity_test.go`: `2b673d4ecff9e1d341db5919d3ba46ee430adc6c`
- `reference/legacy-auth/users.json`: `21b29893e408cfdb58a4a1149c303d5d792ddd18`（既存Fixture、変更なし）

Go 1.23.2の隔離モジュールに**実際の新コード・テスト・既存Fixtureをbyte一致で配置**し、依存する`legacyauth.User`型だけを最小の互換スタブに置換しました。`gofmt -l`は空、`go vet` PASS、`go test -count=1` 3件PASS、`go test -race -count=1` PASSでした。異なるRoleを持つ同一SubjectIDの重複拒否、一意な検索、既存Editor／Viewer／Locked／DisabledのFixtureをそのまま読み込み検索できることを確認しました。

**制限:** 実Repository全体の`legacyauth`／`legacyauthorization`結合テストおよびexact-headのRepository Healthは未実行です。企業Directoryや外部ユーザーStoreがSubject ID一意性を保証するかはこの参照Fixture検査だけでは証明できません。現PRはDraftを維持します。


## E2.8：業務認可失効のHTTP結果と監査の整合性

Hostの早期認可に成功しても、後段の業務処理で権限が変化する場合があります。従来のSession経路では、Runnerが正しく`Stage=authorization-denied`を返しても、共通HTTP Result Writerが200を返し、HostのAuditが`completed`になり得ました。**業務操作を拒否したのに、HTTP／監査の両方が成功として記録される不整合**を修正しました。

`tools/reference-first-web-target/session_command_route.go`では、セッション専用の結果変換時だけ、`authorization-denied`または`identity-link-denied`かつExecution ErrorなしをHTTP 403にします。既存のCredential経路の応答契約や、`security.authorize`のBusiness Resultは変更しません。Hostの`statusWriter`が最初のHTTP statusをAuditへ採用するため、これらの拒否は`Stage=rejected`として監査されます。`identity-link`でStore等のExecution Errorが発生した場合は503とし、内部エラー内容をブラウザへ露出させず固定文言へ変換します。

新たに、Hostの認可直後にSubject Linkが`subject-order-editor`から`subject-order-viewer`へ変化し、後段の既存認可Capabilityが拒否するケースを、実Workspaceを使用する`session_command_route_test.go`へ追加しました。この場合の受入条件は、HTTP 403、`Authorization.Outcome=denied`、Command／Observationなし、Audit=`rejected`です。

### 実施した局所試験（E2.8）

GitHub Blob SHA=`cf4de5774d534805b156832f3f40276020813b6d`と完全一致する**実際の`session_command_route.go`**をGo 1.23.2の隔離`module tei`へ配置し、既存`server.go`の`writeCommandScenarioHTTPResult`の実ロジックを複写、周辺依存（Host／Resolver／Runner／Web Server構成）だけをスタブへ置換して直接検証しました。

- `gofmt -l`：出力なし
- `go vet`：PASS
- `go test -count=1 -v`：2テスト、うち7つのHTTP結果・入力負例すべてPASS
- `go test -race -count=1`：PASS

HTTP 200正常完了、403の業務認可拒否／主体リンク失効、503の主体Store障害と内部詳細秘匿、422の通常Command Execution Error、400のJSON `null`／Credential注入を個別に確認しました。

**残る検証境界:** これはHostとRunnerのモックを使うHTTP結果処理の局所試験であり、実`reference/workspace`、実`security.authorize`、Go HostのAuditイベント配信を含む結合試験は未実行です。追加した実Workspace回帰ケースのPASSも未取得です。PR #1114はDraftを維持し、candidate exact-headの全体Repository Healthを統合条件とします。

## E2.9：exact-headでのローカル統合資格判定

前段の局所テストがPASSしても、PRのheadとmainが検証中に変わったり、過去のmainを基にしたまま全体検査をPASSした場合、統合資格にはなりません。参照用`qualify.sh`は次の入力を必須とする**読み取り専用の手動Gate**です。

~~~sh
# 引数は、確認する正確な40桁のGitコミットSHAと、その時点のmain SHA
bash reference/web-composition/qualify.sh <candidate-head-sha> <current-main-sha>
~~~

スクリプトは`HEAD`と`refs/remotes/origin/main`がそれぞれ指定SHAであること、Git作業ツリーがテスト前後ともcleanであること、およびcurrent mainがcandidateの祖先であることを検査します。そのうえで`verify.sh --race`（E1 Node + 対象Goパッケージ・race）と、**省略なしの**`tools/repository-health-check.sh`（リポジトリ全体のGo vet/test・Workspace検証等）を実行し、途中で失敗・差分生成・HEAD変更があれば非0終了します。必要に応じてログを保存し、統合直前にはGitHub上のPR headとcurrent mainを再取得して照合します。自動`fetch`や`merge`、GitHub Actionsの再開は行いません。

**スクリプトの局所検証:** `qualify.sh`はGitHub Blob `aeebb93e091a3e8ac835f1daade84b947628fc20`とbyte一致するローカルコピーで`bash -n` PASS。隔離した一時Gitリポジトリで、正しいSHA＋cleanの正常経路、間違ったmain SHA、dirtyなチェックアウト、mainを含まないcandidate、検証中に生成された未追跡ファイルの5ケースを検査しました。正常経路ではテスト実行体をstubにしたため、**これは検証スクリプトのGate条件を確認したもので、TEI全体のテストPASSを意味しません。**

2026-10-10現在、PR #1114はまだmainの更新を含んでおらず、このGateの統合資格は**未取得**です。正確なhead・main上でのクリーンなフルチェックアウトおよびすべてのテストの実行記録が得られるまでDraftを維持します。

## E2.10：Session Action JSONの一意な解釈

Session操作に渡す入力について、従来の`json.Decoder.Decode(&struct)`＋`DisallowUnknownFields`方式ではJSONオブジェクト中の**重複member名**および**大文字小文字を畳み込んだ別表記**を拒否できません。認可済みActionの引数は、HTTP入口と下流Commandで異なる値解釈が生じないよう一意であるべきです。

`tools/reference-first-web-target/session_command_route.go`のSession専用Decoderを、JSON Tokenでトップレベルの単一オブジェクトを解析する実装へ変更しました。`from`と`row`以外のmemberは**正確な大文字小文字を含めて**拒否し、重複キー、null、構造化された想定外値、少数の`row`、JSON配列・スカラー、複数JSON文書、64 KiB超の本文を拒否します。両member自体の省略は従来どおり許し、`from`の既定時刻、`row=0`の既定選択、負数拒否は変えません。HTTPクライアントに返すエラーは固定の400であり、内部デコーダーのメッセージや検証済みPrincipalを露出させません。Legacy Credential経路の入力契約には変更を加えていません。

実Workspaceを構成するHTTP回帰テストに、同名重複、`ROW`等の別表記、各memberのnull、入れ子／非整数、トップレベル配列、64 KiB超の拒否と、対応する拒否監査を追加しました。**このHTTP回帰テスト本体は未実行**です。

限定検証として、GitHub実装からJSON Token Decoderの制御構造を取り出して同じ標準ライブラリAPIを使う独立Go 1.23.2モジュールへ移し、正常・異常20ケースを`go test -count=1`、`go vet`、`go test -race -count=1`でPASSしました。**この実験はGitHub Blob全体をそのままコンパイルしたものではなく、実Workspace／Host認可／HTTP監査の動作証拠にもなりません。** 統合判断は引き続きexact-head・main包含・リポジトリ全体ヘルスチェックによります。

- Session route GitHub Blob: `67476fa6b998a7d90851b2c04afc9bb454434509`
- HTTP integration regression test GitHub Blob: `d6ff45a7a70b307147619c0c4ba21fd888ecb801`

## E2.11：副作用直前の業務権限再評価（2026-10-10）

E2.6ではSessionの主体対応付けとAccount適格性をCommand直前に再評価しましたが、同一`SubjectID`の**業務操作権限が早期認可から副作用までの間に変更される**場合は検知できませんでした。特に初回の`security.authorize`が`allowed`であっても、その後のRole・Permission変更を受け取る新しい評価が必要です。

`command_scenario.go`の業務認可呼出しを`invokeBusinessAuthorization`へ集約し、既存の`security.authorize` Capability、`order_entry/security.authorize` Binding、同じ`SubjectID/action/resource`の組を**初回認可とSession専用の副作用直前認可の両方**で使用します。別の権限判定表・独自Role比較は設けません。Commandに進む条件は、(1) Read/Validation、(2) 初回業務認可、(3) Subject LinkとAccountの最終確認、(4) **Canonical認可の最終確認**がすべて成功することです。最終認可ではOutcomeとGrantの主体／操作／資源まで照合し、想定外のResult ModeやGrant不整合はExecution Errorです。

最終認可結果は新しい任意フィールド`effectAuthorization`に記録します。初回の`authorization`を上書きしないため、両者の判断が異なった場合もBusiness Evidenceを失いません。最終`denied`は`Stage=authorization-denied`、Session HTTP 403、Host監査`rejected`として扱います。認可Capability障害や不整合は`Stage=effect-authorization`、HTTP 503、監査`error`とし、HTTPへ内部基盤エラー文字列を渡さず固定メッセージへ置き換えます。Credential経路はこれまでどおり最初の認可を一度実行し、`effectAuthorization`を生成しません。

`tools/reference-first-web-target/session_effect_authorization_test.go`に実Workspaceと本物のRunner/Host/Capabilityを使う回帰テストを追加しました。Subject Linkの3回目の照合時に合成Permission Fixtureを`deny`へ切り替える場合と、AdapterのRuleを利用不能にする場合について、HTTP、認可結果の来歴、Command／Observationが存在しないこと、監査、実行エラーメッセージ秘匿を検査します。既存のSession正常系は`effectAuthorization=allowed`、旧Credential正常系は`effectAuthorization`なしを要求するよう拡充しました。

### E2.11の検証資格と限界

- `tools/reference-first-web-target/command_scenario.go`: GitHub Blob `c0da4c4a388fa796b9ccc7d2a8145891412bd1cc`
- `tools/reference-first-web-target/session_command_route.go`: GitHub Blob `e943cb9b8d3f2e5ccd4998daf0f0cb9a1a471736`
- `tools/reference-first-web-target/session_command_route_test.go`: GitHub Blob `81a4051d9f49789645ffed0126698add9135df32`
- `tools/reference-first-web-target/session_effect_authorization_test.go`: GitHub Blob `7388750d897f85a4bda8b99c729dcedd2ddfb45f`

コード順序をGitHub Blob上で静的確認した結果、Read → Validation → 初回Canonical認可 → Link/Account再検証 → 最終Canonical認可 → Command Capability呼出しであり、旧Credential入口は`nil`のEffect Guardを渡すため最終認可を追加しません。

**ここに記載した新規テストの成功は未取得です。** この実行環境は`github.com`の名前解決に失敗し、リポジトリの完全なクリーンチェックアウトを作れません。よってGo実行・`gofmt`・`go vet`・Race・Repository Healthの実候補に対するPASSは主張しません。現時点の成果は実装、回帰テスト追加、Blob上の呼出し順序の静的検査です。

また、最終認可と副作用の実行が**同一のトランザクション／原子的な操作として結合されているわけではありません**。最終認可の直後から実際のCommitまでのごく短い期間に起こる失効を完全には排除できません。本番相当の実行では、効果を持つAdapter側での承認フェンス、権限バージョン条件、トランザクション内再検証、冪等性・取消可能性を別途設計・検証する必要があります。合成Fixtureでの時刻操作は実IdPやDirectoryの失効伝播証明ではありません。

統合資格は従来どおり現行mainを含むexact-headのclean full checkoutでの`reference/web-composition/qualify.sh`全PASS、実Workspace Go回帰、未変更のrefs確認を前提とします。CIは使用せず、Draft・未マージを維持します。

## E2.12：Command Adapterの内部まで最終認可を移動（2026-10-10）

E2.11の最終認可はCommandの`resolver.InvokeCapability`を呼ぶ**前**に行われていました。これは業務操作を許可した後、DriverのBinding解決、Adapter解決、Capability呼出しまでの間に権限が変更される競合を残します。E2.12ではSessionの場合にだけ、**Concrete Command Adapterの`Invoke`コールバック内部、合成業務状態を実際に変更する行の直前**へ以下を移しました。

- 信頼済みSubject Link／Accountの再取得と適格性判定
- 同じ`security.authorize` Capabilityを使う業務権限の再評価
- 付与された`AuthorizationGrant`のSubjectID・Action・Resourceの完全一致確認

最初の認可結果は既存`authorization`に記録し、最終認可結果は`effectAuthorization`へ別記します。Adapter内で失効が検出された場合、DriverはAdapterのerrorをExecution Errorとして包みますが、**明示的な私的Sentinelに一致した業務拒否だけ**をRunnerで復元し、`identity-link-denied`／`authorization-denied`としてHTTP 403／Audit拒否へ写像します。その他のBinding・Adapter・権限ストア障害や予期しない結果はExecution Errorを保持し、HTTP 503の固定文言・Audit errorにします。旧Credential経路はGuardなし・認可1回という従来契約を保持します。

初回認可についても、単に`Outcome=allowed`なら許すのではなく、**同じ`requireMatchingBusinessGrant`**を適用して`security.authorize`のCapability ID・Result Mode・主体・Action・Resourceの一致を確認します。最終認可と初回認可で別の権限解釈を作らないためです。

新しい`session_effect_grant_contract_test.go`では、主体／操作／資源／Capability／Mode／Outcome／Payloadの不一致を拒否する表形式試験と、Goの構文木から`effectGuard`とCanonical再認可が**Command Adapter内部の状態変更より前**にあることを検査する静的回帰試験を追加しました。既存の`session_effect_authorization_test.go`は権限の途中失効、最終権限基盤障害のHTTPと監査を引き続き要求します。

### 境界と検証保留

- `command_scenario.go`: GitHub Blob `500ba27ffc2d2e9ce5a86b3b9924c0a57b363335`
- `session_effect_grant_contract_test.go`: 今回追加したCapability Grantと構文木の契約試験
- **実WorkspaceのGoテスト、Race、Repository Healthは未実行です。** GitHubへ変更したファイルはリポジトリ上で確認しましたが、actual full checkoutでの`go test`成功は取得していません。
- これはConcrete Adapter内までチェックを近づけた変更であり、**認可の版条件と副作用Commitが同じトランザクションになったことを意味しません**。外部DB・SaaS API・Queue等の本番相当で原子的に失効を扱うには、プラグインに対応可能なEffect Fence／版条件／再試行保証が必要です。共通要件と競合試験は独立した[Issue #1174](https://github.com/hat47x/tei-platform/issues/1174)へ起票済みです。
- PR #1114はDraftを維持します。最新mainを含むexact headクリーンcheckoutでWeb＋Repository Health全PASSと前後cleanを確認するまで統合しません。


## E2.13：Sessionの業務Validation拒否とHTTP／監査の整合

Sessionでは、HTTP要求が認証・CSRF・Hostの早期業務認可を通過した後でも、RunnerのRead／Validationで**業務的な不成立**となり、Commandが実行されないことがあります。従来のSession結果変換では`Stage=validation-not-satisfied`、`Command=nil`にもかかわらずHTTP 200が返り、Host Auditが`completed`となる余地がありました。これはビジネス操作の成功率、監査、生成AIクライアントの再試行判断に誤解を生みます。

そこでSession専用`handleProtectedSessionCommand`で`validation-not-satisfied`かつExecution Errorなしを**HTTP 422**へ分類し、Business Result本体・Validation Evidenceはそのまま保持することにしました。HostのAuditはHTTPステータスを根拠に`rejected`となります。これにより、HTTP上の受理と実際のCommand完了を混同しません。通常のCredential経路は既存契約（業務拒否の結果を200のJSONとして返す）を保持します。

既存`command_scenario_test.go`の実Workspace入力で不成立が再現される`from=2026-09-18T08:00:00Z`・`row=1`を使い、Session 422・監査`rejected`・Validation`not-satisfied`・認可以降に進まないことと、旧Credential経路が従来どおり200のBusiness Resultを返すことを`session_command_route_test.go`へ追加しました。

- `session_command_route.go`: GitHub Blob `2ce16a6a47b0060a158eaa7ad78c035b8730ebc6`
- `session_command_route_test.go`: GitHub Blob `7dbbf446239be6459cc908c41ab919f3cd424eb9`
- 新規HTTP統合テストの実行結果は**未取得**。Actual full checkoutのGoテスト・Race・Repo Healthを省略して完了とは扱いません。PR #1114はDraftを維持します。


## E2.14：Session実行状態とHTTP結果のfail-closed化

現在の`handleProtectedSessionCommand`は、Session Providerが真正性・CSRF・業務権限を確認した後に、業務ResultをHTTPへ投影します。**Hostの早期認可が通過したことはCommand完了の証拠ではありません。** そのため、Business Resultの終了状態を次のように分類します。

| Runner Result / Error | 新Session HTTP | Host Audit | 意味 |
| --- | --- | --- | --- |
| `completed`、Command・Observationとも確認済、Errorなし | 200 | completed | 合成参照で完了 |
| `validation-not-satisfied`、Errorなし | 422 | rejected | 業務Validationによる未実行 |
| `identity-link-denied` / `authorization-denied`、Errorなし | 403 | rejected | 権限・主体の業務拒否 |
| `identity-link` / `authorization` / `effect-authorization` のExecution Error | 503 | error | 身元・権限基盤が利用できない |
| `command` / `observation` のExecution Error | 500 | error | 実行成否・観測結果が未確定 |
| 予期しない非終端Result、Errorなし | 500 | error | 成功と断定できない |

認可等の基盤ErrorのHTTP応答は固定の`trusted identity or authorization resolution unavailable`、その他の未確定の実行Errorは固定の`action completion unconfirmed; inspect authoritative state before retrying`とし、内部のAdapter/Driverエラーメッセージをブラウザに返しません。特にCommand中のTimeoutや障害では、**HTTP 500を「確実にロールバックされた」と解釈しない**ことが重要です。現在のReferenceが使用する合成状態は外部のDurable Commitではありません。実Providerの操作確定を判定する条件・操作ID・Retry Policyは汎用[Issue #1174](https://github.com/hat47x/tei-platform/issues/1174)で整備します。

既存Credential経路のHTTP 422 Execution Error Envelopeは互換のため変更しません。新旧でStatus Semanticsが異なる部分は専用の参照テストで明示します。さらに、Read Runnerが要求する`from`のRFC3339形式はSession HTTP入口で検証し、壊れた日時や存在しない暦日を**400で拒否**します。これは業務Data取得エラーを入力不正として誤分類しないためです。

### 今回見つかったテスト不一致

E2.11でブラウザへ返す権限基盤障害の文言を固定化しましたが、既存HTTP回帰テストの2か所が旧文言`trusted identity resolution unavailable`を期待したままでした。これを実装に合わせ、固定文言秘匿とAudit動作を同一契約に戻しました。

### 実装と検証状態

- `session_command_route.go`: GitHub Blob `f846ac88a5340287f17b6182be9d1a4ea5715e9f`
- `session_command_route_test.go`: GitHub Blob `01675b77607b1731e6968f01098fddda8297931e`
- 追加した回帰ケースは、`order 1003`の既存合成Command Adapter障害（旧経路422、新Session 500と固定文言、Audit error）、Validation不成立の422、異常日時の400などです。
- GitHub Blobで実装とテストのエラー文字列一致・結果分岐を静的に確認しました。ただし**実Workspace Go結合試験、Go vet/race、exact-head full Repository Healthの成功結果は未取得**です。以前の隔離Go試験を現在headの成功証拠とみなしません。
- PR #1114はDraft・未マージを維持します。合成Fixtureを用いたテストの成功（未取得）と、実ProviderによるDurable Effectの観測（未取得）も区別します。
