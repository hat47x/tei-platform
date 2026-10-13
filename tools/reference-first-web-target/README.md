# First Web Target Reference

`ISSUE-000007` / Gate Cへ向けた第一Web Targetの最小Reference実装です。

このtoolはProduction Web Runtimeでも、Target-neutral Presentation IRでもありません。Gate Bで固定した`order_entry`のCanonical / Driver meaningを、Web固有の第二正本を作らずにHTTP / HTMLへProjectionできるかを確認するためのReference-specific adapterです。

## 現在のincrement

現在は **standard / Custom UI shared Command boundary** まで進んでいます。

- `reference/workspace`を通常のWorkspace loaderで読む
- `driver.Resolver`でApplication Patternを解決する
- reference / header-detailのContract QueryをDriverでexact resolutionする
- command PatternのCapability Bindingが解決可能であることを確認する
- Handover Baseline `B / R`とTarget build revision `T`を別identityとしてmanifestへ保持する
- 同じProjectionからHTMLとJSON manifestを生成する
- `/scenario/read`でbrowse → record → customer reference → order-lines header-detailを同じmapped rowから実行する
- reference / header-detailはDriverのEnvironment-aware Contract Query resolutionを通す
- `POST /scenario/submit`でAuthentication → read → Canonical Validation → Authorization → Command → separate Observationを実行する
- Authentication / Authorizationの`denied`をBusiness Outcomeとして保持し、Execution Errorへ変換しない
- Command Business ResultとObservationを別結果として保持する
- `/custom-submit`のCustom UIもJSON APIと同じ`commandScenarioRunner`を利用する
- Custom UI固有差をform / layout / HTML renderingへ閉じる
- credentialをresult / Evidenceへ保持しない
- synthetic runtime fixtureはCanonical dataと分離する
- browseはcurrent Driver APIにEnvironment-aware executor selectionがないため、Environment instanceをWeb側で推測しない

まだ行いません。

- actual browser replay / Functional Evidence取得
- screenshot / Visual Fidelity Evidence取得
- Gate C Functional Evidence
- Gate C Visual Evidence

## 実行

### サービス契約を使う受注シナリオ

帳票と一括処理をHTTPなしで動かす手順は[画面を使わない参照実行](NON_SCREEN.md)を参照する。

通常の起動コマンドに`-composition-scenarios`を加えると、`order_entry`の読取・登録、または`repair_intake`の顧客参照を`internal/composition`の選択済みサービスへ接続する。両者は別の契約と登録範囲を使い、既存の業務処理を共有する。PostgreSQLへの接続と画面を使わない帳票・一括処理は引き続き受注専用である。

`/scenario/read`、`/scenario/submit`、`/custom-submit`は従来と同じ操作を使う。登録処理内の読取も同じサービスを呼ぶ。JSON応答の`composition`には契約、提供元、実行ファイルのSHA-256、世代を記録する。認証拒否などの業務結果と実行エラー、登録結果と後続観測は従来どおり区別する。

このフラグだけを指定した場合は、静的に結合した参照用実装と既存の模擬データ・模擬認証を使う。実行ファイルのハッシュは実体の識別であり、配布元の信頼や本番の権限を保証しない。保護されたセッション経路と別プロセスの交換は対象外である。進め方は[接続計画](../../plan/workstreams/REFERENCE_MODEL_PLUGIN_EXECUTION_2026-10.md)を参照する。

実際のPostgreSQLを併用する場合は、さらに`-postgres-dsn-file /secure/connection-file -postgres-fencing-token 1`を指定する。受注と明細を実DBから読み、実DBへ確定し、別読取りで観測する。顧客参照と認証は模擬実装のままで、応答に混在範囲を明示する。接続や許可の確認に失敗した場合は模擬データへ戻らない。サーバーは参照データや許可を作成しない。[使い捨てDBによる再現手順](../../reference/order-workflow/README.md)を参照する。

同じ接続指定を受注専用ビルドと固定WorldのJSON APIでも利用できる。固定WorldのAPIは構築時の業務モデルを使い、実行時にWorkspaceや構文解析器へ依存しない。修理受付専用ビルドにはこの接続設定を追加しない。

`-model-revision sha256:...`を加えると、ホストが選んだ正確なソース版を投影と呼出しに記録する。ソースや構成ファイルが変わった場合は処理を拒否し、明示的な再読込みを必要とする。PostgreSQLの許可には同じ`model_revision`を別に設定する。版の指定は変更案の受理や人の承認の証明ではない。[共通受理から再実行までの検証](../../reference/order-workflow/README.md)を参照する。

```sh
go test -race -count=1 ./internal/composition ./tools/reference-first-web-target
```

Target build revisionは実際に実行するcheckoutから取得します。

```bash
TARGET_REVISION="$(git rev-parse HEAD)"

go run ./tools/reference-first-web-target \
  -workspace reference/workspace \
  -application order_entry \
  -baseline-id B \
  -baseline-revision e5231839a5dcc62cce1181a2731fcb7c70d16736 \
  -target-revision "$TARGET_REVISION" \
  -listen 127.0.0.1:8080
```

入口:

- `/` — minimal HTML projection
- `/manifest.json` — 同じProjectionのJSON
- `/scenario/read?from=2026-09-18T08%3A30%3A00Z&row=0` — Reference診断用read-only slice
- `POST /scenario/submit` — Authentication / Validation / Authorization / Command / Observation
- `/custom-submit` — 同じCommand Scenario境界を使うCustom HTML UI
- `/healthz` — process health

### submit Command例

credentialはReference専用のsynthetic verifierへ渡すruntime inputであり、responseへは残しません。

```bash
curl -sS -X POST http://127.0.0.1:8080/scenario/submit \
  -H 'content-type: application/json' \
  -d '{
    "loginId":"legacy-order-editor",
    "credential":"editor-secret",
    "from":"2026-09-18T08:30:00Z",
    "row":0
  }'
```

`editor-secret`はReference fixture用のsynthetic値であり、production credentialではありません。

### 第二Application `repair_intake` の顧客照会

`-application repair_intake`で起動すると、`order_entry`用のread / command runnerは付かず、顧客照会だけが有効になります。Applicationごとに存在するrunnerだけを付け、runnerが別のApplicationを実行する場合、またはどのApplicationを実行するか宣言しない場合は、起動時に拒否します（manifestと実行内容の食い違いを防ぐため）。

```bash
go run ./tools/reference-first-web-target \
  -workspace reference/workspace -application repair_intake \
  -baseline-id B -baseline-revision e5231839a5dcc62cce1181a2731fcb7c70d16736 \
  -target-revision "$(git rev-parse HEAD)" -listen 127.0.0.1:8080
```

- `/customer-lookup?customerId=11` — Customer IDを入れる画面。照会結果、実行の根拠、結果が主張しないこと、未解決を表示する
- `/scenario/customer-lookup?customerId=11` — 同じrunnerを使うJSON

Customer IDから`repair_intake.intake.customer_id`のmapped rowを作り、`repair_intake.customer_reference`をDriverのEnvironment-aware Contract Query resolutionで実行します。`order_entry`が使う`customer.customer`の意味とContract Query（`master_data.customer`の`find_by_id`）を、コピーせず同じidentityで使います。

JSON結果では、HTTP入力を`requestedCustomerId`として明示します。これは入力値を記録するruntime stateであり、business occurrence identifierの宣言ではありません。同様にRead Scenarioの`selectedRowIndex`はruntime上の選択位置です。mapped row、Contract Queryの結果row、個々のAttribute値も、それだけではbusiness identityのEvidenceになりません。`scenarioId`、`applicationId`、Environment、Instance、executor binding等のidentityも実行文脈を示すもので、business data identityとは別です。

限界:

- developmentのExecutorはsynthetic fixtureで、実Databaseへの接続ではありません。
- productionにはContract Query executor bindingが無く、productionでは実行できません。developmentのExecutorへ代替しません。
- 認証の門はありません（Reference診断用）。
- Customer IDを整数へ変換するのはTarget内のfixture上の選択で、Canonicalな型規則ではありません。
- `requestedCustomerId`、`selectedRowIndex`、mapped row、result rowをbusiness keyへ昇格しません。business identityはCanonical / accepted Evidence側で別に確認します。

## Authority boundary

```text
Authoritative Source / Canonical World
          ↓
Driver Resolver
          ↓
Reference Web Projection
          ↓
HTML / JSON
```

Web側は次をauthorしません。

- Semantic Data / Relation
- Query / Contract
- Datasource access
- Validation
- Authorization
- Capability
- Business Result / Outcome
- Observation meaning

`presentationCandidate`（collection / item / relation-selection / detail-collection / command-invocation）はWeb Targetで観測するためのtarget-local labelです。Canonical Pattern kindではありません。

## Visual review states

Custom UIはVisual Evidenceで比較する主要状態を、raw JSON一枚ではなく構造化HTMLとして分離表示します。

- success: Selected order / Customer reference / Order lines / Validation / Command Business Result / separate Observation
- authorization denied: Business OutcomeとしてAuthorization denialを表示
- validation not-satisfied: Canonical Validation結果を表示し、Command / Observationを出さない
- execution error: partial Scenario stateとExecution Errorを表示し、Business Result / Observationと区別

Gate C acquisition runnerは、これらの状態をHTML artifactとしても保存します。

```text
custom-submit-success.html
custom-submit-authorization-denied.html
custom-submit-validation-not-satisfied.html
custom-submit-execution-error.html
```

これはFunctional Evidenceの再現性を高めるためのartifactであり、Screenshot / Visual Fidelity Evidenceそのものではありません。Visual capture recipeの正本は `plan/gate/GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md` です。

## Evidence高さ

sourceとtestがRepositoryに存在するだけならSource Evidenceです。

actual `go test` / server execution、Functional replay、Screenshot等を取得していない段階では、Gate C Product Evidenceとは扱いません。

特に、

```text
HTTP response exists
  != Functional Evidence complete

HTML looks plausible
  != Visual Fidelity Evidence complete

Projection source exists
  != Gate C Ready
```

を維持します。


## Gate C Evidence acquisition

actual Target runからSemantic / Functional raw Evidenceを取得する入口は次です。

```bash
TEI_GATE_C_ARTIFACT_DIR=/tmp/tei-gate-c-first-web-T \
bash tools/run-first-web-target-gate-c-evidence.sh
```

runnerはclean Target revision `T`からserverを起動し、read / successful Command / Authentication denied / Authorization denied / Validation not-satisfied / Command Execution Error / Custom UIをsame-run artifactとして保存します。

その後 `tools/reference-first-web-target-evidence-audit` がraw artifactを再読し、`B / R / T`、Runtime Binding、Validation not-satisfied、Command Execution Error、Result / Observation境界、credential非保持を確認します。

Visualは `tools/reference-first-web-target-visual-evidence-audit`、architecture negative casesは `tools/reference-first-web-target-architecture-evidence` で別Evidenceとして扱います。Visual audit toolには`-emit-manifest-scaffold`も用意し、Semantic / Functional summaryから同じ`B / R / T`とrequired capture / criterionを持つ未評価manifestを生成できます。生成時のcriteriaはすべて`unresolved`で、scaffoldだけではVisual passになりません。

最終判定は `tools/reference-first-web-target-gate-c-readiness-audit` に三Evidenceとarchitecture conformanceを渡して行います。

Semantic / Functional runner単独ではVisual Fidelity Evidenceを取得しません。Semantic / Functionalが成功してもsummaryは `visualEvidence.status = pending`、`gateCReady = false` のままです。Visual audit単独もGate C Readyを宣言しません。

手順の正本は `plan/gate/GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md` です。

## E1へ向けたApplication別のHTTP構成境界（実装準備段階）

[#1167](https://github.com/hat47x/tei-platform/issues/1167)の実装準備として、`read_scenario.go`に同居していたOracle/PostgreSQLの共有Contract Query Fixtureを`reference_contract_query_fixture.go`へ切り出しました。これは`order_entry`と`repair_intake`が利用する同一実装であり、業務意味や契約を複製しません。

さらに、受注用のHTTP read/commandレスポンス型・ハンドラーと`/custom-submit`のルート登録を`order_scenario_http.go`にまとめ、repair側の`/scenario/customer-lookup`・`/customer-lookup`のルート登録とHTTPハンドラーは`repair_scenario_http.go`へ配置しました。業務RunnerはHTTP依存を持たない`customer_lookup_scenario.go`に残しています。共通`server.go`の`Handler()`は二つの登録関数を呼び出します。

共有行Projectionの型・変換は`read_scenario.go`から、独立Goパッケージ[`sharedprojection`](sharedprojection/README.md)へ移しました。`shared_semantic_projection.go`は従来の名前と関数署名を保つための型エイリアス・薄いラッパーだけを残し、受注Runnerと修理受付Runnerが型や変換処理を複製せず利用できるようにしました。新しいパッケージはWorkspace LoaderやLegacy認証・認可を直接参照せず、Goの依存閉包についても専用テストで負例を固定します。さらに、`repair_intake`のExecutor Registryは顧客参照用Oracleだけを登録する構成へ縮小しました。`order_entry`ではOracle顧客参照とPostgreSQL明細参照の両方を引き続き登録します。`runner_transport_boundary_test.go`で業務RunnerソースへのHTTP/JSON Transport import混入を拒否し、`reference_contract_query_fixture_test.go`で二Applicationの登録範囲を検査します。**これらのテストは最新PR HEADでは未実行です。**

**現行の第一Web Targetは、従来どおり両Applicationのルートを登録する一つのGo `package main`です。** 未設定RunnerへのHTTP 503、非対応methodへの405、存在しないパスへの404を維持するための回帰テストを追加しました。この整理のみで不要なGo実装がコンパイル・リンクから除外されたとは言えません。

さらに、`webTargetServer.ScopedHandler()`を**オプトインの参照構成**として追加しました。ProjectionのApplication IDが`order_entry`なら受注用、`repair_intake`なら修理受付用のルートだけを登録し、他方のHTTPパスは404になります。不明なApplication IDや、別ApplicationのRunnerが混入した構成はHandlerを返さず失敗します。一方、従来の`Handler()`はGate C用の複数Application診断ルートを変更せず維持します。`scoped_routes_test.go`ではルート非公開・Runner混入拒否を、`scoped_parity_test.go`では同じ実Workspace fixtureを使ったread/Command/認証・認可拒否/Execution Error/customer lookupのHTTPレスポンスと想定ステータスの同等性を回帰条件にしています。**これらの追加テストは現HEADでのPASS未取得**であり、Goソースの物理的な分離を証明するものではありません。

現行Go CLIでも、診断時に`-scoped-routes`を指定すると選択Applicationのルートだけを公開できます。省略時は従来の複数Application診断ルートを保ちます。

```sh
go run ./tools/reference-first-web-target \
  -workspace reference/workspace \
  -application order_entry \
  -baseline-id B \
  -baseline-revision "$ACCEPTED_BASELINE_R" \
  -target-revision "$(git rev-parse HEAD)" \
  -scoped-routes \
  -listen 127.0.0.1:8080
```

これは**HTTPルートの公開面積を狭めるだけ**です。同じ実行ファイル内の未選択Application実装のGo依存・権限・Secretが消えたわけではないため、E1の物理除外や本番のAuthorization境界として受け入れないでください。

E1で異なるApplication向けバイナリを成立させるには、共通Transportの契約とApplication固有Route/Runnerの依存を**別Goパッケージとcompile-time entry**へ分離し、その上で既存Gate CのBusiness Result・Execution Error・Observation・Authorizationについて差分回帰を行う必要があります。物理除外は`go list`だけでなく実`go build`・`go tool nm`・成果物digestで確認します。実配備機構や第二のProvider Resolverをここで増設しません。

## Application別コンパイル時Go入口（E1実装・要実行検証）

専用ビルドでも`-composition-scenarios`と`-model-revision`を指定できる。アプリケーションの選択はビルド時に固定し、サービス契約を使っても起動引数から変更できない。受注専用ビルドでは模擬実装を使う。PostgreSQL接続と画面を使わない実行は複合参照ホストの入口だけで提供する。

`tei_frozen_world`を使うAPI専用ビルドでも`-composition-scenarios`を指定できる。構築時に固定したWorldをそのまま既存の実行処理とサービスへ渡し、実行先にWorkspace読込みを持ち込まない。このビルドではライブのSource版を意味する`-model-revision`を拒否する。Worldの指紋とSourceの版は別の識別であり、相互に読み替えない。[固定Worldの構築手順](../reference-first-web-two-unit-e1/README.md)を参照する。

通常のWorkspaceに外部Workspaceのimportがある場合も、ルートのSource版だけを指定した実行版の受理は拒否する。提供元が所有するソースは利用側のSource Indexへ複製されず、ルート版だけでは提供元の内容を固定できない。提供元ごとの版の受理と出所の追跡は後続の接続対象として残す。

修理受付のサービスは`tei.reference.repair-intake.customer-lookup`、提供元は`repair-customer`である。入力の不正、顧客の取得結果、模擬実装であるという制約を既存の処理からそのまま返す。応答の`composition`へ契約・提供元・実行ファイルの指紋・世代を記録し、モデル版を指定した場合は呼出し前に現在のソースも照合する。参照ホストの静的な実行許可であり、利用者の認証や業務上の更新権限は追加しない。

保護セッションによる確定経路も受注側のソースとして選択する。共通HTTPサーバーはタグ別の構成フックを呼び、修理受付には確定用の構成やハンドラーを持たせない。受注・複合ビルドでは既存の保護チェックを維持する。

変更のないLinuxチェックアウトで、実バイナリの接続と物理的な除外を次のコマンドで確認できる。未作成のリポジトリ外ディレクトリを指定する。

```sh
python3 tools/verify-composed-targets.py --output /secure/composed-target-evidence
```

複合・受注専用・修理受付専用の実行物、SHA-256、バイト数、依存集合、シンボルを記録する。接続前・接続後・専用ビルドの結果を同じ入力で比較し、実HTTPのエラーと除外したルートも確認する。修理受付から受注、PostgreSQL接続、帳票整形の実装が除かれていることを検査する。単一プロセス内の静的結合の検証であり、別プロセスの無停止交換、実Oracle接続、本番権限、性能目標の達成を示さない。

従来のタグなし`package main`はGate C用の複合Reference Targetとして残しています。
試験用の`tei_order_only` / `tei_repair_only`で、それぞれのApplication固有Runner・HTTPルート・画面Template・単独`main`をコンパイル時に選択します。

```sh
# 通常の複合Target: 従来のGate C互換を維持
go test -count=1 ./tools/reference-first-web-target

# 専用Target: Applicationを別の起動引数へ変更することはできない
go test -count=1 -tags=tei_order_only ./tools/reference-first-web-target
go test -count=1 -tags=tei_repair_only ./tools/reference-first-web-target
go build -trimpath -ldflags=-buildid= -tags=tei_order_only \
  -o /tmp/tei-order-entry ./tools/reference-first-web-target
go build -trimpath -ldflags=-buildid= -tags=tei_repair_only \
  -o /tmp/tei-repair-intake ./tools/reference-first-web-target

# 同じclean HEAD / Goプロファイルでsource closure・バイナリ・SHA・nmを一括観測
bash tools/reference-first-web-target/measure_tagged_targets.sh \
  /tmp/tei-first-web-e1-evidence-NEW
```

出力先は未作成の絶対パス、リポジトリ外にしてください。計測スクリプトは前後のcheckout clean、HEAD不変、実`go list`によるGoFile選択、3種類の`go build`、`go tool nm`、byte数、SHA-256、およびE0比較JSONを記録します。失敗時は結果を正規Evidenceとみなしません。Goコードによる`tagged_source_selection_test.go`でも、3種類のGoFile集合と相手Applicationの実装除外を検証します。

`repair_intake`専用バイナリについては、同一Goパッケージの受注専用ソース除外に加え、E0の実import閉包に`tei/internal/security/legacyauth`と`tei/internal/security/legacyauthorization`が**存在しない**ことを検査します。複合版・受注専用版には両方が存在することも確認し、検査の空振りを防ぎます。さらに実バイナリの`go tool nm`で修理受付専用版への両パッケージのシンボル混入を拒否します。これはSource / Linkの限定的な負例であり、認可の正しさ、本番Secretや権限の総合監査、同じB/R/Tの業務同等性を証明するものではありません。

ビルド時に選択するApplicationごとに、トップ画面の案内リンクも切り替えます。共通`server.go`はリンク先を決めず、タグ別の`referenceIndexFooterHTML()`がHTML断片を提供します。従来の複合Targetのfooterは変更せず、`order_entry`専用ビルドではread/submit/custom-submit、`repair_intake`専用ビルドではcustomer-lookupのみを案内します。別Applicationのルートが404でも画面上の誘導として残らないよう、タグ別のHTMLテストと実バイナリのオプトインテストを追加しました。これはUI案内上の整合性であり、認可を代替しません。

`repair_intake`では顧客参照用Oracle実装のみをリンク対象にするため、PostgreSQL明細Queryの実装は別ソース`reference_order_contract_query_fixture.go`へ移しました。両Applicationで共用するOracle実装を複製せず、DriverのBinding identityも変えていません。

**検証状態と限界**：2026-10-12の既存E1変更のCI記録は[#1302](https://github.com/hat47x/tei-platform/pull/1302)、[#1340](https://github.com/hat47x/tei-platform/pull/1340)を参照してください。試験と計測は実行した候補版と構築条件に結び付け、過去の合格を現在のHEADへ持ち越しません。原Goソースの区別や参照例のシンボル除外だけでは、秘密情報や権限の完全な除外は示せません。模擬データの結果同等性は、構築成果物の承認、本番の認証・認可、ブラウザーとAPIの別配備、受理済み引継ぎ条件での実環境の完全同等性を証明しません。これらのE1受入条件は[#1167](https://github.com/hat47x/tei-platform/issues/1167)に残します。

### 実バイナリのHTTP結果同等性（明示実行する統合テスト）

`tagged_binary_parity_integration_test.go`では、従来の複合Targetとorder/repair専用Goバイナリを**実際にビルドして起動**し、同じFixture B/R/TでHTTP結果を比較します。正常read、注文送信、認証拒否、認可拒否、Execution Error、入力不正、顧客参照、未公開Route 404を対象にします。成功・拒否のStage、Observationの境界、認証情報の非漏洩も検査します。

通常の`go test`では実行しません。信頼できる完全なclean checkoutで次を実行します。

```sh
TEI_E1_BINARY_PARITY=1 go test -count=1 \
  -run '^TestTaggedBinariesPreserveFirstWebBusinessHTTPParity$' \
  ./tools/reference-first-web-target
```

3バイナリの依存閉包・ファイル・SHA-256・シンボル・サイズ計測にも同じHTTP parityを追加し、実行ログを外部Evidenceフォルダへ残す場合：

```sh
TEI_E1_RUN_HTTP_PARITY=1 \
  bash tools/reference-first-web-target/measure_tagged_targets.sh \
  /tmp/tei-first-web-e1-fixture-evidence-NEW
```

`measurement.json`は、実行を省いた場合`syntheticFixtureHttpParityVerified: false`、成功した場合のみ`true`とし、`tagged-http-parity.test.txt`を保持します。合成Fixtureの一致は実DB接続・本番Authorization・Build Artifact Lockの承認を意味しません。`sameHandoverBusinessParityVerified`・`productionAuthorizationVerified`・`artifactAuthority`などの上位条件は自動的にtrueになりません。

**留意**：GitHubにテストと計測手順をコミットしただけでは、Go build・Go test・Repository HealthのPASS資格は得られません。必ず実行したcheckoutのHEAD/clean前後と証跡を保存してください。

## 保護された受注操作とプラグイン合成の接続（参照実装）

[Issue #1145](https://github.com/hat47x/tei-platform/issues/1145)の次段階として、`session_command_route.go`の**保護済み業務操作の内側**に、合成基盤が選択した実装への接続を追加しました。従来の受注操作は変更せず、案件側が明示的に`SelectedAction`を与えた場合に限って使います。

- Web入口では、本人確認、要求元保護、主体の対応付け、既存の`security.authorize`による操作認可を従来どおり先に行います。
- 認可後、合成基盤の`Coordinator.Acquire`で現在の提供者選択とホストの権限を確認し、呼出し中は選択した世代を保持します。
- 提供者識別子、機能契約の識別子・版、成果物の指紋がホスト側の登録情報と一致した場合だけ、既存の`RunVerifiedSessionCommand`へ進みます。提供者が消えたり実行権限が失効したりした場合は、業務操作に進まず503として監査します。
- 登録した提供者と指紋は構成時に固定し、HTTP要求に含まれる任意の利用者ID、提供者ID、ヘッダーから選択を変更しません。業務操作内のValidation、認可の再確認、結果・Observationの扱いは従来どおりです。

`session_composition_test.go`は実Workspace、既存Driver、既存の業務認可規則と受注操作を使用し、正常系、閲覧者の業務認可拒否、未認証、ホスト権限の失効、提供者撤去、設定不足を検証します。

```sh
go test -count=1 -run '^TestSelected(CompositionRunsCanonicalProtectedOrderAction|OrderActionRejectsIncompleteHostConfiguration)$' ./tools/reference-first-web-target
go test -count=1 ./reference/web-composition/go-host
```

**実証範囲の限界**：本人確認と要求元検査は合成Fixtureです。提供者選択に使用する指紋もテスト用の宣言であり、成果物の実バイト列や署名を検査したものではありません。また、この第一段階では複数の論理的提供者を**同じ既存Action Runner**へ接続するだけで、異なる物理実装への差し替えや実プロセスの無停止更新を証明しません。合成選択の成功だけで実行権限・永続的な副作用・本番利用の安全性が成立するわけではありません。

### 選択済み業務操作の保護経路を固定する

合成基盤で選択した業務操作の呼出し口は、提供者を選択できるだけでは安全になりません。Web Hostの保護経路を経由せず、呼出し口をHTTP経路へ直接登録すると、業務認可を経ない呼出しにつながるおそれがあります。

そのため、[選択済み操作の接続部](../../reference/web-composition/go-host/selected_action.go)は、**現在の要求について、Web Hostが検証済みの利用者と、その操作の認可完了を引き継いだ場合だけ**提供者を取得します。ホストは本人確認、要求元検査、業務認可のすべてに成功した後で、要求内だけで有効な印を付けます。別の業務操作に対する認可では通過できません。印は非公開のGoコンテキストキーで保持し、HTTPヘッダーや本文の値で代用できません。元の保護付き要求が終了すると印も失効し、以前の要求の実行文脈を保存して後から呼び直しても提供者を取得できません。

この印は実行権限そのものではありません。合成基盤の提供者選択、現在の権限確認、および既存の業務操作内部の認可再確認は引き続き必要です。試験では保護経路を経ない直接呼出し、別操作の認可の流用、認証失敗、要求元確認失敗、業務認可拒否、提供者撤去を区別し、下流を呼び出さないことを確認します。

```sh
go test -count=1 ./reference/web-composition/go-host
go test -count=1 ./tools/reference-first-web-target
```

合成された認証、要求元確認、業務Fixtureによる検証であり、実際の認証事業者や本番の権限ストアによる保証を示すものではありません。また、実装を別プロセスへ移すときはGoコンテキストの印をそのまま送信せず、ホストが所有する委譲契約を別途設計する必要があります。
