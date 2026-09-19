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
