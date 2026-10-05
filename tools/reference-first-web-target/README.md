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

限界:

- developmentのExecutorはsynthetic fixtureで、実Databaseへの接続ではありません。
- productionにはContract Query executor bindingが無く、productionでは実行できません。developmentのExecutorへ代替しません。
- 認証の門はありません（Reference診断用）。
- Customer IDを整数へ変換するのはTarget内のfixture上の選択で、Canonicalな型規則ではありません。

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

これはFunctional Evidenceの再現性を高めるためのartifactであり、Screenshot / Visual Fidelity Evidenceそのものではありません。Visual capture recipeの正本は `plan/GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md` です。

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

手順の正本は `plan/GATE_C_FIRST_WEB_TARGET_EVIDENCE_RUNBOOK.md` です。
