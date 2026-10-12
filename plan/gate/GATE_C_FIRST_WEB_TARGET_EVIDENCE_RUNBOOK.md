# Gate C First Web Target Evidence Runbook

## 目的

最初のウェブ実行先について、判定段階Cの検証根拠を再取得するための手順です。引継ぎ基準版`B / R`と実行先の構築版`T`を混同せず、変更のない同じ版から取得できるようにします。

この実行手順書で扱う実行器は、次の検証根拠を取得します。

1. ソースの事前確認
2. 意味の準拠確認に用いる元の成果物
3. 対話操作・機能を検証する元の成果物
4. 標準の派生表示と個別実装の画面のHTML成果物

**表示の忠実度に関する検証根拠は取得しません。**

HTMLが存在するだけでは画面画像や表示の忠実度を検証したことにはなりません。意味と機能の検証に成功しても、表示の検証根拠が未取得なら`gateCReady = false`のままにします。

契約の正本：

- `plan/gate/GATE_C_EVIDENCE_CONTRACT.md`
- `plan/gate/GATE_B_TO_C_HANDOVER_BASELINE_B.md`

実装：

- `tools/run-first-web-target-gate-c-evidence.sh`
- `tools/reference-first-web-target-evidence-audit`

## 前提

リポジトリ全体を取得し、変更のない状態にしたルートディレクトリから実行します。

実行器は少なくとも次の条件を確認し、満たせなければ安全側に停止します。

- 現在の作業ディレクトリがリポジトリのルートであること
- 追跡対象・追跡対象外・無視対象にローカルの変更状態がないこと
- 追跡対象の索引項目が通常の`H`だけであること
- `curl`を実行できること
- 最初のウェブ実行先のソースに`gofmt`を適用済みであること
- 対象のGoパッケージのソース試験が成功すること
- 実行前後で`targetBuildRevision = T`が一致すること
- 成果物のディレクトリがリポジトリの外にあること

Goの実行環境は、実行器の内部で次のように固定します。

```text
GOENV=off
GOWORK=off
GOFLAGS=
```

## Handover Baseline

現在の最初の引継ぎ基準版は次のとおりです。

```text
B
baselineRepositoryRevision R =
e5231839a5dcc62cce1181a2731fcb7c70d16736
```

実行器の既定値もこの基準版を使いますが、検証根拠を採用するときは最終結果に記録された`B / R / T`を必ず確認します。

`T`は実行器の開始時に変更のない作業領域で取得する`git rev-parse HEAD`の結果です。`R`と同じ版には固定しません。

## 実行

リポジトリ外の空の保存先を明示して実行することを推奨します。

```bash
TEI_GATE_C_ARTIFACT_DIR=/tmp/tei-gate-c-first-web-T \
bash tools/run-first-web-target-gate-c-evidence.sh
```

既定の`B / R`を変更する場合に限り、次のように明示します。

```bash
TEI_GATE_C_BASELINE_ID=B \
TEI_GATE_C_BASELINE_REVISION=e5231839a5dcc62cce1181a2731fcb7c70d16736 \
TEI_GATE_C_ARTIFACT_DIR=/tmp/tei-gate-c-first-web-T \
bash tools/run-first-web-target-gate-c-evidence.sh
```

既存の成果物ディレクトリは上書きしません。

## same-run raw artifacts

実行に成功した場合は、少なくとも次の成果物をリポジトリの外に保存します。

```text
run-metadata.json
source-tests.log
server.log
health.txt
projection-index.html
manifest.json
read.json
submit-success.json
submit-authentication-denied.json
submit-authorization-denied.json
submit-validation-not-satisfied.json
submit-execution-error.json
custom-submit-get.html
custom-submit-success.html
custom-submit-authorization-denied.html
custom-submit-validation-not-satisfied.html
custom-submit-execution-error.html
semantic-functional-summary.json
architecture-conformance.json
```

### read

`read.json`は参照事例を診断するための、読取り専用の接続先から取得します。

確認対象は次のとおりです。

- browse
- selected record
- customer reference
- header-detail
- Runtime Binding

この独立した読取り用の接続先を、本番の認証済み読取り機能と同じものとして扱いません。

### successful Command

`submit-success.json`では、同じシナリオの中で次の処理を確認します。

```text
Authentication
 -> read
 -> Canonical Validation
 -> Authorization
 -> Command Business Result
 -> separate Observation
```

命令の受理応答を、その後の状態を観測した結果とは扱いません。

### Business denial

次の結果は別々の成果物に残します。

- Authentication denied
- Authorization denied

いずれもHTTPの通信エラーや実行時エラーと混同しません。

### Validation not-satisfied

`submit-validation-not-satisfied.json`では、正本上の検証規則が`not-satisfied`となり、認可・命令・観測へ進まないことを確認します。

参照事例の実行時試験データでは、注文`1002`をこの検証に使います。これは正本のデータではありません。

### Execution Error

`submit-execution-error.json`では、能力の呼出しを完了できない状態を業務上の結果に変換していないことを確認します。

参照事例の実行時試験データでは、注文`1003`に対する合成の命令変換器の失敗を使います。

HTTP層では、実行先固有の状態コード422の応答に、次の情報を含めます。

- `B / R / T`
- partial Scenario result
- `executionError`

これらを保持します。

これは正本上の新たな結果ではありません。

## 独立audit

実行器は元の成果物を取得した後、次の監査を実行します。

```bash
go run ./tools/reference-first-web-target-evidence-audit ...
```

実行後、`semantic-functional-summary.json`を作成します。

監査では、少なくとも次の項目を再確認します。

- 元の成果物間で`B / R / T`が厳密に一致すること
- 実行前後で、変更のない同じ版`T`であること
- サーバープロセスと稼働確認の成果物
- 五つのアプリケーションパターンの識別情報
- 読取り専用のデータ接続先の境界
- OracleとPostgreSQLの実行時の対応付け
- 命令が成功する経路
- 認証と認可による拒否
- 検証規則が満たされない場合
- 命令の実行時エラー
- 命令の業務上の結果と観測の分離
- 個別実装の画面と標準の操作が共有する意味上の境界
- 元の成果物に参照用認証情報の内容が混入していないこと

元のバイト列に対するSHA-256を最終結果へ記録します。

## Visual Fidelity

この実行器の結果だけでは表示の忠実度を確認できません。

最終結果に期待する値は次のとおりです。

```text
sourceQualification.passed = true
semanticConformance.passed = true
functionalEvidence.passed = true
visualEvidence.status = pending
visualEvidence.required = true
gateCReady = false
```

画面画像または同等に再確認できる表示の成果物を、**同じ実行先の構築版T**について別途取得します。

HTMLの成果物やDOMが存在するだけでは、表示の検証が成立したとは扱いません。

## Credential boundary

実行時の要求には参照事例用の合成認証情報を使いますが、通常の検証根拠には保存しません。

監査では、少なくとも次の情報が混入した場合、安全側に停止して拒否します。

- Reference synthetic credential文字列
- `credential-ref://legacy/`

要求本文そのものは成果物に保存しません。

## 停止条件

次のいずれかに該当する場合、判定段階Cの検証根拠として採用しません。

- ソースの試験が失敗
- ソースが整形されていない
- 作業領域に変更が残る、または索引が非標準の状態
- 実行中にHEADが変化
- サーバーの稼働確認に失敗
- B・R・Tが一致しない
- 期待したパターンと実行時の対応付けが一致しない
- 業務上の拒否が実行時エラーに変換された
- 検証規則を満たさないのに命令へ進んだ
- 実行時エラーが業務上の結果に変換された
- 認証情報の内容が元の成果物に残った
- 表示の検証根拠が未取得なのに判定段階CのReadyを宣言した

## Gate C判定

この実行器の成功は意味・機能の検証根拠を取得できたことを示しますが、判定段階CのReadyそのものではありません。

表示の忠実度を別の検証根拠として取得し、`plan/gate/GATE_C_EVIDENCE_CONTRACT.md`の全条件を独立して監査した後に限り、判定段階Cの合否を決めます。

実際の実行に基づく判定記録：`plan/gate/GATE_C_READINESS_AUDIT.md`


## Visual Evidence intake

意味・機能の検証根拠とは別に、画面画像などの表示に関する成果物を、同じ`targetBuildRevision = T`に結び付けます。

### manifest scaffold

表示の検証記録に必要な`B / R / T`、必須の撮影ID、評価項目と撮影結果への参照先を手作業で複製せず、監査ツールからひな型を生成します。

```bash
mkdir -p /tmp/tei-gate-c-first-web-T/visual

GOENV=off \
GOWORK=off \
GOFLAGS= \
go run ./tools/reference-first-web-target-visual-evidence-audit \
  -semantic-functional-summary /tmp/tei-gate-c-first-web-T/semantic-functional-summary.json \
  -emit-manifest-scaffold \
  -visual-run-id visual-T \
  -evaluator-identity reference-human-review \
  -evaluator-version 1 \
  -evaluation-method screenshot-review \
  -viewport-width 1440 \
  -viewport-height 1000 \
  > /tmp/tei-gate-c-first-web-T/visual/visual-manifest.json
```

ひな型には、意味・機能の最終結果に記録された`B / R / T / Target Adapter / Host / source run identity`を転記し、必須の撮影結果を決定論的なファイル名で列挙します。

表示領域の高さに注意します。上の例の1440×1000では、標準の派生表示も個別実装の画面も下部が切れます。画面全体を撮影する場合は`-viewport-height`を増やし、記録する表示領域と実際の撮影条件を一致させます。最初の参照事例では1440×3000を使い、Playwright同梱の`chrome-headless-shell`を`--window-size=1440,3000 --screenshot=<png>`で起動し、再表示用補助ツールの`/capture/<id>`を撮影しました。

さらに各captureへ、

- `sourceArtifact`
- `sourceSHA256`

を自動設定する。これはsame-runでTarget自身がrenderしたHTML artifactと、そのexact bytes digestである。Screenshotはこのreplay sourceへ明示的に結び付く。

重要:

```text
generated scaffold
  -> every criterion = unresolved
  -> Visual Fidelity passではない
  -> Gate C Readyではない
```

画面画像の取得後、識別情報を記録した評価者が各項目を`pass / fail / unresolved`のいずれかに更新し、必要な根拠を注記に残します。その後、表示の監査へ進みます。

撮影ツール自体は、この参照事例では固定しません。ブラウザーの自動操作、画面操作エージェント、人による撮影のいずれも認めますが、画面画像と評価記録は残します。

監査入口:

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go run ./tools/reference-first-web-target-visual-evidence-audit \
  -semantic-functional-summary /tmp/tei-gate-c-first-web-T/semantic-functional-summary.json \
  -visual-manifest /tmp/tei-gate-c-first-web-T/visual/visual-manifest.json \
  -artifact-dir /tmp/tei-gate-c-first-web-T/visual \
  > /tmp/tei-gate-c-first-web-T/visual-audit.json
```

最初の参照事例では、画面画像の成果物を`image/png`形式で受け入れます。これは正本として標準化した画像形式ではなく、最初の参照事例に限った検証成果物の形式です。

### 必須capture

Visual manifestには少なくとも次のcaptureを含める。

```text
projection-index
custom-submit-initial
custom-submit-success
custom-submit-authorization-denied
custom-submit-validation-not-satisfied
custom-submit-execution-error
```

各captureは:

- ID
- surface
- state
- PNG filename
- media type
- viewport width / height
- source HTML artifact
- source HTML SHA-256

を持つ。

ファイル名は成果物ディレクトリ直下の名前だけを認めます。ディレクトリを遡るパスや、シンボリックリンクの解決によって保存先の外を指すパスは受け付けません。

### Visual capture state recipe

最初に推奨するのは、意味・機能の検証と同じ実行で保存した、**実行先が生成したHTML成果物を再表示して画面画像を取得する方法**です。

replay source mapping:

```text
projection-index
  <- projection-index.html

custom-submit-initial
  <- custom-submit-get.html

custom-submit-success
  <- custom-submit-success.html

custom-submit-authorization-denied
  <- custom-submit-authorization-denied.html

custom-submit-validation-not-satisfied
  <- custom-submit-validation-not-satisfied.html

custom-submit-execution-error
  <- custom-submit-execution-error.html
```

いずれも同じ版`T`について実行先が実際に返した応答から取得し、SHA-256を意味・機能の最終結果に保存します。表示記録のひな型は、その照合値を自動で転記します。

そのため表示を確認する評価者は、認証情報を再入力せずに、保存済みのHTMLをブラウザーで描画して画面画像を取得できます。

```text
same-run Target response HTML
  -> exact SHA-256 in Semantic / Functional summary
  -> sourceArtifact / sourceSHA256 in Visual manifest
  -> browser render
  -> PNG Screenshot
  -> attributed Visual evaluation
```

動作中の実行先から直接撮影する方式も認めます。ただしその場合も、記録の`sourceArtifact / sourceSHA256`を、同じ実行・同じ状態のHTML応答に一致させます。

### digest-verified replay helper

保存済みHTMLを手作業で探さず、安全なloopback routeからrenderするため:

- `tools/reference-first-web-target-visual-replay`

を使える。

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go run ./tools/reference-first-web-target-visual-replay \
  -semantic-functional-summary /tmp/tei-gate-c-first-web-T/semantic-functional-summary.json \
  -artifact-dir /tmp/tei-gate-c-first-web-T \
  -visual-manifest /tmp/tei-gate-c-first-web-T/visual/visual-manifest.json \
  -listen 127.0.0.1:18181
```

replay helperは起動前に:

- Semantic / Functional source qualification pass
- Semantic Conformance pass
- Functional Evidence pass
- Visual pending
- Gate C Readyではない
- required HTML artifactのexact SHA-256一致
- artifact directory外symlink拒否
- credential material非混入

を確認する。

capture route:

```text
http://127.0.0.1:18181/capture/projection-index
http://127.0.0.1:18181/capture/custom-submit-initial
http://127.0.0.1:18181/capture/custom-submit-success
http://127.0.0.1:18181/capture/custom-submit-authorization-denied
http://127.0.0.1:18181/capture/custom-submit-validation-not-satisfied
http://127.0.0.1:18181/capture/custom-submit-execution-error
```

`/capture-manifest.json`には、各経路、元の成果物、そのSHA-256、`B / R / T`を記録します。

`-visual-manifest`を渡した場合、`http://127.0.0.1:18181/`をVisual review indexとして表示する。ここでは:

- Visual run / evaluator provenance
- Screenshot filename
- capture route
- source artifact / SHA-256
- criterionのcurrent result
- criterionが参照するcapture

を一画面で確認できる。

表示確認用の一覧は記録を書き換えず、評価結果も書き込みません。表示の監査の代わりにもなりません。

補助ツールは起動時にHTMLのバイト列をメモリに固定し、そのまま返します。業務処理や認証情報の入力は再実行しません。

```text
Visual replay helper
  != Target runtime re-execution
  != Screenshot Evidence
  != evaluator judgment
  != Gate C Ready
```

状態をlive Targetで再現する必要がある場合のdeterministic recipeは次とする。

```text
projection-index
  GET /

custom-submit-initial
  GET /custom-submit

custom-submit-success
  POST /custom-submit
  loginId = legacy-order-editor
  credential = editor-secret
  from = 2026-09-18T08:30:00Z
  row = 0

custom-submit-authorization-denied
  POST /custom-submit
  loginId = legacy-order-viewer
  credential = viewer-secret
  from = 2026-09-18T08:30:00Z
  row = 0

custom-submit-validation-not-satisfied
  POST /custom-submit
  loginId = legacy-order-editor
  credential = editor-secret
  from = 2026-09-18T08:00:00Z
  row = 1

custom-submit-execution-error
  POST /custom-submit
  loginId = legacy-order-editor
  credential = editor-secret
  from = 2026-09-18T07:00:00Z
  row = 2
  expected HTTP status = 422
```

これらの認証情報は参照事例の実行時入力です。値を画面画像・検証記録・評価者の注記へ転記しません。POST後に表示する画面にも認証情報の入力欄の値を再表示しません。

保存済みHTMLの再表示から撮影する場合、評価者は認証情報を再入力する必要がありません。

意味・機能の検証実行器は、同じ状態のHTML成果物も保存します。そのため表示の評価者は、画面画像で見える状態が元の機能検証成果物と対応しているか確かめられます。ただし、HTMLが一致したことを画面画像による検証の代わりにはしません。

### Visual評価項目

Gate C contractに対応して少なくとも次を評価する。

```text
field-grouping-readability
relation-reference-comprehensibility
header-detail-locality
validation-feedback-distinguishable
authorization-denial-vs-system-error-distinguishable
business-result-vs-observation-distinguishable
standard-vs-custom-ui-difference-explainable
```

各項目は:

```text
pass
fail
unresolved
```

のいずれかとし、評価根拠となるcapture IDを明示する。

特に:

- validation feedback → `custom-submit-validation-not-satisfied`
- authorization denied vs system error → `custom-submit-authorization-denied` と `custom-submit-execution-error`
- Business Result vs Observation → `custom-submit-success`
- standard vs Custom UI → `projection-index` と `custom-submit-initial`

を参照する。

### evaluator provenance

Visual manifestには少なくとも:

- evaluator identity
- evaluator version
- evaluation method

これらを保持します。

人による評価、視覚言語モデル、画像比較などの評価方式を、正本上の意味には含めません。

### Visual auditの意味

Visual auditは:

- Semantic / Functional summaryが同じ`B / R / T`で成立済み
- Screenshot集合が完全
- ScreenshotがPNG artifactとして存在
- capture pathが安全
- Screenshot exact bytes SHA-256
- 各Screenshotの`sourceArtifact / sourceSHA256`がSemantic / Functional summaryのsame-run raw HTML digestへ一致
- required criterionが存在
- criterionが必要なcapture stateを参照
- credential materialがmanifestへ混入していない

ことを確認する。

すべての評価項目が`pass`でも、表示の監査だけでは判定段階CのReadyを宣言しません。

期待:

```text
visualFidelityPassed = true   # evaluatorが全項目passの場合
gateCReady = false
```

最終Gate C判定はSemantic / Functional / Visualを別Evidenceとして揃えた後、Gate C contract全体へ戻って行う。


## Architecture negative Evidence

Semantic / Functional acquisition runnerは同じclean `T`で:

```text
architecture-conformance.json
```

も生成する。

実装：

- `tools/reference-first-web-target-architecture-evidence`

第一Referenceでは次をfail closedで確認する。

### R → T Canonical不変

```text
reference/workspace/**
reference/world.json
```

について、Handover Baseline revision `R`からTarget revision `T`までGit差分がないことを要求する。

これは第一Referenceのcoarseな不変条件であり、将来のproduction semantic diff taxonomyではない。

差分が存在する場合はWeb側で補完せず、baseline subject変更の可能性としてsuccessor Handoverへ戻る。

### Web Target authority boundary

`tools/reference-first-web-target`配下で:

- `.tei` authoring sourceを持たない
- `.sql` sourceを持たない
- `SELECT ... FROM`
- `INSERT INTO`
- `UPDATE ... SET`
- `DELETE FROM`
- DDL / MERGE

等のnative SQL本文を持たないことを確認する。

これは「すべての業務二重実装が数学的に存在しない」ことの証明ではなく、第一Referenceで観測可能なauthority boundary Pressureである。

### shared Command / Result boundary

actual `T`でfocused source testを実行し:

- standard JSON interactionとCustom UIが同じ`commandScenarioRunner`を利用
- 検証規則が満たされない場合後にAuthorization / Commandへ進まない
- 命令の実行時エラーがBusiness Result / Observationへ畳まれない
- Execution Error HTTP envelopeが`B / R / T`を保持

ことを再確認する。

### Repository source

Architecture Evidence自身も:

- clean full checkout
- fixed `GOENV=off / GOWORK=off / GOFLAGS=`
- baseline `R`が`T`のancestor
- run前後同一clean `T`

を要求する。

このEvidenceはVisual Evidenceの代替ではない。


## Final Gate C Readiness Audit

Semantic / Functional、Visual、Architectureのactual Evidenceが**同じ`B / R / T`**へ揃った後だけ最終監査を実行する。

実装：

- `tools/reference-first-web-target-gate-c-readiness-audit`

例:

```bash
GOENV=off \
GOWORK=off \
GOFLAGS= \
go run ./tools/reference-first-web-target-gate-c-readiness-audit \
  -semantic-functional-summary /tmp/tei-gate-c-first-web-T/semantic-functional-summary.json \
  -architecture-evidence /tmp/tei-gate-c-first-web-T/architecture-conformance.json \
  -visual-audit /tmp/tei-gate-c-first-web-T/visual-audit.json \
  > /tmp/tei-gate-c-first-web-T/gate-c-readiness-audit.json
```

### 最終監査が要求するもの

少なくとも:

- Semantic / Functional source qualification成功
- Semantic Conformance成功
- Functional Evidence成功
- Visual required captures完備
- Visual criteriaすべてresolved
- Visual Fidelity pass
- Architecture Evidenceが同一clean `T`でreproducible
- `R → T` Canonical World不変
- Web Target authority boundary成立
- standard / Custom UI shared Command boundary成立
- Business Result / Execution Error / Observation境界成立
- Target差分handoffが復元可能
- すべてのunresolvedが明示分類済み
- blocking unresolvedがない

ことを要求する。

Visual auditやSemantic / Functional auditが単独で`gateCReady = false`を返すのは意図した設計である。最終auditだけが三Evidenceとarchitecture conformanceを合わせてGate C判定を行う。

### unresolved分類

第一Referenceで既知の未解決は、黙ってsuccessへ変換しない。

```text
visual-fidelity-evidence-not-acquired
  -> 独立Visual Evidenceで解消された場合だけ resolved-by-visual-evidence

browse-environment-instance-binding-not-exposed-by-current-driver-api
  -> carry-forward / ISSUE-000026

actual-external-command-observation-not-acquired
  -> carry-forward / external-runtime-evidence
     synthetic Observationをactual external Evidenceへ格上げしない

target-capability-taxonomy-remains-candidate-until-second-target
  -> carry-forward / ISSUE-000026

general-target-neutral-presentation-taxonomy-not-yet-defined
  -> carry-forward / ISSUE-000026
```

これ以外の未知unresolvedが現れた場合、最終auditはfail closedで停止する。

### 第二Targetへの差分handoff

Semantic / Functional auditはactual runから次をTarget difference handoffへ残す。

- Application
- Target Adapter / Host
- observed Target Capability候補
- Runtime Binding
- execution artifact
- host ABI
- Execution / Trust observation
- Web固有差
- Canonicalへ持ち上げてはいけない差
- carry-forward unresolved

`go run`で起動したactual Evidence runではGo version / GOOS / GOARCH / launch mode / loopback bindをrun metadataから取得し、execution artifact / ABIを推測ではなく観測値としてhandoffする。

これは第一Targetで得た観測recordであり、Target CapabilityやExecution Profileの最終taxonomyではない。第二TargetのPressureを受けて`ISSUE-000026`で再検討する。

### Gate C Readyの意味

最終auditが`gateCReady = true`を返しても、次を意味しない。

- production-ready Web product
- actual external Oracle / production Command成功
- Target-neutral Presentation taxonomy完成
- 第二Target不要
- unresolved消滅

Gate C contractに定めた第一Targetの意味・操作・視覚・architecture境界が、同一provenanceのEvidenceから再確認できたことだけを意味する。

actual final auditを取得するまでは、source implementationだけからGate C Readyを宣言しない。

## Evidence artifactの保管

Ready判定に採用したrunのartifactは、Repositoryへcommitせず、次のとおり保管する（`ISSUE-000057`）。

- 容量が小さい場合（目安: 数MB以下）: `MANIFEST.sha256`（各fileのSHA-256）を同梱した`tar.gz`へ圧縮し、対象revision（`R` / `T`）を指すtagを付けたGitHub Release（prerelease）のassetとして格納する。tag名は`gate-b-evidence-R-<short>` / `gate-c-evidence-T-<short>`とする。
- 容量が大きい場合: WSL上に留める（WSLは日次バックアップの対象）。
- 格納前に、DSN・credential・secret pathが含まれないことを確認する（audit toolの拒否に加え、grepで確認する）。
- 格納後、assetを再ダウンロードしてSHA-256が一致することを確認し、判定文書へ保管先（locator）とarchiveのSHA-256を追記する。過去の判定は書き換えず、追記する。

```bash
gh release create <tag> <archive.tar.gz> --target <R/Tの完全なcommit SHA> \
  --title "<title>" --notes-file <notes.md> --prerelease
```

現在採用済みの第一Web Target Gate C Evidenceは次のlocatorを使う。

```text
tag = gate-c-evidence-T-0ba87a4
asset = tei-gate-c-evidence-T-0ba87a4e.tar.gz
archive SHA-256 =
de5f587a038c56c0f4b5776953b2f6729e6521d4eda67ebcc246567638fa12a3
```

再取得時はarchiveだけでなく内部manifestも確認する。

```bash
evidence_root="$(mktemp -d)"

gh release download gate-c-evidence-T-0ba87a4 \
  --pattern 'tei-gate-c-evidence-T-0ba87a4e.tar.gz' \
  --dir "$evidence_root"

printf '%s  %s\n' \
  de5f587a038c56c0f4b5776953b2f6729e6521d4eda67ebcc246567638fa12a3 \
  "$evidence_root/tei-gate-c-evidence-T-0ba87a4e.tar.gz" | sha256sum -c -

tar -xzf "$evidence_root/tei-gate-c-evidence-T-0ba87a4e.tar.gz" -C "$evidence_root"
(
  cd "$evidence_root/gate-c-T-0ba87a4e"
  sha256sum -c MANIFEST.sha256
)
```

filenameが一致するだけの展開directoryをadopted Evidenceとして扱わない。

