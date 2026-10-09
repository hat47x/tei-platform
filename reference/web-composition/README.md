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

- \`plan-check.mjs\`：**既に採用済みの処理点**を入力として、依存グラフと保護条件を検査し、決定的な順序を作ります。同じ名前の機能を提供する実装が二つあれば拒否します。
- \`plan-check.test.mjs\`：保護された受注操作、認証主体の対応付け、保護の欠落、循環、衝突など12件の正常・異常例を固定しています。

既存の#951のProvider解決は行いません。処理点は**解決済みであるものとして受け取る**ため、この試作が返す順序は実行権限やProviderの適合性を保証しません。

## 実行方法

~~~sh
node --test reference/web-composition/plan-check.test.mjs
~~~

2026-10-09、構成検査と認証設定検査の関数については、V8隔離実行環境で同等の12例を直接評価し、12件が想定どおり成功／拒否されました。**上記のNodeコマンドをリポジトリの実ファイルに対して実行した記録は未取得です。** さらにTEI全体のヘルスチェック、Go HTTPサーバとの接続、実ブラウザ、実IdP、Cookie・CSRF保護の動作試験は行っていません。

## この試作が保証しないこと

\`guards\`や\`onTerminal\`は処理点の**宣言**です。実際に要求元検証が行われたこと、拒否時に監査が実行されたこと、認証Providerが真正であることの証拠ではありません。これらを保証するには、信頼できるHostによる処理の強制と、Provider適合性の検査、実HTTP動作試験が別に必要です。

\`hostPolicy\`と\`hostContracts\`は信頼済みHostが与える前提です。未信頼の案件拡張やブラウザから自由に指定できる製品APIにしてはなりません。この試作では受注操作を公開経路へ変更する構成を一律に拒否していますが、一般的な公開Actionの採否を製品全体で決めたわけではありません。

認証の\`subjectLinks\`は、発行元・主体識別子に基づく明示的な対応付けの書式だけを検査します。ログインの実行、OIDCの暗号学的検証、本人同意、リンク設定の管理権限は対象外です。

## 次の検証

1. 実際のNodeテストと、正確なブランチ先端に対するTEI全体のヘルスチェックを実施します。
2. 既存First Web Targetの\`http.ServeMux\`と\`security.authenticate\`/\`security.authorize\`の間に、保護条件を一つずつ接続します。別のProvider Resolverを作らず、#951の契約へ戻します。
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

