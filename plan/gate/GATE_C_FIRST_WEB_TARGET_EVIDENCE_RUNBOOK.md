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

意味・機能の検証根拠を取得する実行器は、変更のない同じ版`T`で次の成果物も生成します。

```text
architecture-conformance.json
```

も生成する。

実装：

- `tools/reference-first-web-target-architecture-evidence`

最初の参照事例では次の条件を確認し、満たさなければ安全側に停止します。

### R → T Canonical不変

```text
reference/workspace/**
reference/world.json
```

これらについて、引継ぎ基準版`R`から実行先の版`T`まで、Git上の差分がないことを求めます。

これは最初の参照事例で使う大まかな不変条件であり、将来の本番向けの意味差分の分類方式ではありません。

差分がある場合はウェブ側で補わず、基準版の対象の意味が変わった可能性を確認して後継の引継ぎ手順へ戻ります。

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

これは業務ロジックの二重実装が一切存在しないことを数学的に証明するものではありません。最初の参照事例で観測できる管理権限の境界を検証するものです。

### shared Command / Result boundary

実際の版`T`で対象を絞ったソース試験を実行し、次の事項を確認します。

- 標準のJSON経由の操作と個別実装の画面が、同じ`commandScenarioRunner`を利用すること
- 検証規則が満たされない場合は、認可や命令へ進まないこと
- 命令の実行時エラーを業務上の結果や観測にまとめないこと
- 実行時エラーのHTTP応答が`B / R / T`を保持すること

ことを再確認する。

### Repository source

Architecture Evidence自身も:

- clean full checkout
- fixed `GOENV=off / GOWORK=off / GOFLAGS=`
- baseline `R`が`T`のancestor
- run前後同一clean `T`

を要求する。

この検証根拠は表示の検証根拠の代わりにはなりません。


## Final Gate C Readiness Audit

意味・機能、表示、構造について実際に取得した検証根拠が、**同じ`B / R / T`**に揃った後でのみ最終監査を実行します。

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

- 意味・機能のソースの事前確認に成功
- 意味の準拠確認に成功
- 機能の検証根拠が成立
- 表示の必須撮影結果がすべて揃っている
- 表示のすべての評価項目について結論が出ている
- 表示の忠実度が合格
- 構造の検証根拠を変更のない同じ版`T`で再現できる
- `R → T`の間に正本のモデルが変わっていない
- ウェブ実行先の管理権限の境界が成立
- 標準と個別実装の画面が命令の境界を共有
- 業務上の結果・実行時エラー・観測の境界が成立
- 実行先の違いを引き継ぐ情報を復元できる
- すべての未解決事項を明示的に分類済み
- 判定を妨げる未解決事項がない

ことを要求する。

表示の監査や意味・機能の監査がそれぞれ単独で`gateCReady = false`を返すのは意図した設計です。三種類の検証根拠と構造の準拠確認を合わせて判定段階Cの合否を決めるのは、最終監査だけです。

### unresolved分類

最初の参照事例で既知の未解決事項を、暗黙のうちに成功扱いにはしません。

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

これ以外の未知の未解決事項が現れた場合、最終監査は安全側に停止します。

### 第二Targetへの差分handoff

意味・機能の監査では、実際の実行で得た次の情報を、実行先の違いを引き継ぐ記録に残します。

- Application
- Target Adapter / Host
- 観測した実行先の能力の候補
- Runtime Binding
- execution artifact
- host ABI
- 実行と信頼についての観測結果
- ウェブ固有の違い
- 正本に取り込んではいけない違い
- 引き継ぐ未解決事項

`go run`で起動して実際に検証根拠を取得した際には、Goの版・GOOS・GOARCH・起動方式・ループバックへの接続待受けを実行記録から取得し、実行成果物とABIを推測値ではなく観測値として引き継ぎます。

これは最初の実行先で得た観測記録であり、実行先の能力や実行方式についての最終的な分類ではありません。二つ目の実行先と比較した結果を踏まえ、`ISSUE-000026`で再検討します。

### Gate C Readyの意味

最終監査が`gateCReady = true`を返しても、次の状態を保証するものではありません。

- 本番運用に耐えられるウェブ製品
- 実際の外部Oracleや本番の命令実行での成功
- 実行先に依存しない表示方式の分類の完成
- 二つ目の実行先が不要であること
- 未解決事項がすべて解消したこと

判定段階Cの契約で定めた最初の実行先における意味・操作・表示・構造の境界を、来歴の揃った検証根拠から再確認できたことだけを意味します。

実際の最終監査結果を取得するまでは、ソースの実装状況だけを根拠に判定段階CのReadyを宣言しません。

## Evidence artifactの保管

Ready判定に採用した実行の成果物は、リポジトリへコミットせず、次のように保管します（`ISSUE-000057`）。

- 容量が小さい場合（目安：数MB以下）：各ファイルのSHA-256を記録した`MANIFEST.sha256`を同梱して`tar.gz`形式に圧縮します。対象版（`R`／`T`）を指すタグ付きのGitHub Release（公開前リリース）に成果物として保存します。タグ名は`gate-b-evidence-R-<short>`／`gate-c-evidence-T-<short>`とします。
- 容量が大きい場合：WSL上で保管します（WSLは日次バックアップの対象）。
- 保存前に、DSN・認証情報・秘密情報のパスが含まれていないことを確認します（監査ツールによる拒否に加え、grepでも確認）。
- 保存後に成果物を再ダウンロードし、SHA-256が一致することを確認します。判定文書には保存先と圧縮ファイルのSHA-256を追記し、過去の判定は書き換えません。

```bash
gh release create <tag> <archive.tar.gz> --target <R/Tの完全なcommit SHA> \
  --title "<title>" --notes-file <notes.md> --prerelease
```

現在採用している最初のウェブ実行先の判定段階Cの検証根拠は、次の保管先を使います。

```text
tag = gate-c-evidence-T-0ba87a4
asset = tei-gate-c-evidence-T-0ba87a4e.tar.gz
archive SHA-256 =
de5f587a038c56c0f4b5776953b2f6729e6521d4eda67ebcc246567638fa12a3
```

再取得するときは、圧縮ファイルだけでなく内部の一覧も確認します。

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

ファイル名が一致するだけの展開先ディレクトリを、採用済みの検証根拠として扱うことはしません。

