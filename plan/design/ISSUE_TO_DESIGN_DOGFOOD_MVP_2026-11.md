# TEI自身を題材とするIssue→設計文書ドッグフーディング（P0縦切り）

- 状態：**探索・受入条件候補**。本書の作成自体はTEI CLI/MCP実行、全設計情報の取得、意味受理、自己構築成立を意味しない。
- 作成日：2026-11-07（JST）
- Issue要件：[#3262](https://github.com/hat47x/tei-platform/issues/3262)（2026-11-07 08:28:31 UTC観測）、[#2605](https://github.com/hat47x/tei-platform/issues/2605)、[#1119](https://github.com/hat47x/tei-platform/issues/1119)、[#2831](https://github.com/hat47x/tei-platform/issues/2831)。
- 比較したmain：`96d276b74d338f47b2027d66a6dc7eedb24faa45`。比較した具体的Sourceは `cmd/tei/cli_contract.go`（blob `f1c167c56005da7f204f310078d3f2c3853d6848`）、`skills/yi-tei-modeling/SKILL.md`（blob `a49c111ff318fc83987de5f95e07d02dd86ff10d`）、`docs/status.md`（blob `ab349b8ea778adb550be4600c17c99de21821fc4`）。
- 関連する既存計画：[#3223](https://github.com/hat47x/tei-platform/pull/3223)はテスト生成前のDesign Contextを担当するDraft。本文の§14を置き換えない。本書は**TEI自身のIssueから設計票を作る一件**と工程上のBlockerを扱う。

## 1. このドッグフーディングで実際に解く課題

**要求入力**：「Issue #3262 の『対象Scopeに必要な設計情報を確実に読めること』を、最短の一Subjectで示し、足りない機能・証拠を設計文書と既存Issueへ戻す」。

**固定実例**：公開合成Reference `reference/workspace` と、`semantic-data:sales.order`。最初はTEI自身のCLIとProduct Skillの動作確認に限定し、対象Domain全件やIssue全件・実顧客データは使わない。Function/Business/Rule/Procedureの存在・不存在や意味受理は、Data Subjectが一件読めたことから推定しない。

**期待する成果物**：要求と根拠が紐付く設計票（本書）、一度ビルドした実TEI CLIからの読み取り結果と閉包診断（CIのStep Summary）、未充足条件を担当Issueへ戻す差分、およびG1/G2/G4へ至る到達基準。JSONをMarkdownへ整形しただけで設計完成としない。

## 2. ユーザー要求→設計義務→既存Owner

| 義務 | 必ず区別するもの | 今回の結果・担当 |
| --- | --- | --- |
| ID-01 要求の同定 | Issue URL/観測時刻/要求Scopeと、モデル内Subject ID | 本書に固定。現在のIssue全件を取り込んだとは主張しない。#3262 |
| ID-02 Sourceの到達 | 宣言されたroot、管理Inventory、実Loader Read、Parser/Role/Accepted Meaning | 現行`workspace-files`の`readByLoader`と`validate`診断は利用可能。Parser完了・Required全件は別。#2822/#2823/#2620 |
| ID-03 同一版参照 | CLI実行物のbyte/checkout、Workspace、World/SourceIndex fingerprint、Tool受領 | 単一実行物で限定比較。fingerprint一致だけでWorkspace全Byteの同時Snapshotは言えない。#2354/#2605 |
| ID-04 三視点と依存閉包 | Data/Function/Business、manual、Rule、隠れたPredicate、Relation、Requirement | 初回Dataのみ実取得し、それ以外は`unknown/not-queried`。依存先推測は禁止。#3262/#1125/#1132/#2582 |
| ID-05 開示・配送 | local read-only、外部Harness送出許可、全Reply、page end | CLI smokeは外部受信資格でない。8,192 byte途中切断・56/55が既知の負例。#2509/#2516 |
| ID-06 設計文書の採否 | Source-backed事実、生成AIの推論、未決、Ownerの意味受理 | 文章化は候補、正式意味受理は別。#1803/#1124/#2514 |
| ID-07 再入力と差分 | 旧結果、新Source、変更影響、再資格化 | fixed refからの再収集・差分が必要。#1119/#2600 |
| ID-08 セルフビルド接続 | 実行Graph、実消費byte、生成Artifact、再入可能な実TEI Process | 文書生成とGo buildは全体Self-Buildではない。#2831 |

## 3. 実行できる範囲の最小手順

既存の `tei.cli.discovery/v0` は `selected-read-only-commands` のみ。外部に使えるTool/認可全体を示さない。`version -json` は通常の `ok/result` EnvelopeではなくIdentity形、`world` はraw、`source-index` はmode依存である。全結果を無条件で同一JSON schemaと仮定しない。

このPRの `.github/workflows/tei-issue-design-dogfood.yml` で、一つのcheckoutからTEIを一度ビルドして次を実行する（新しいTEIコマンドは追加しない）。

1. `tei commands` と `tei version -json` で公開範囲・実Binaryを確認する。
2. `tei workspace-files -workspace reference/workspace` でInventory/Loader読取区別を残す。
3. `tei model catalog` から `semantic-data:sales.order` の曖昧でない存在を検査する。
4. `tei inspect` と `tei source-index` を同じSubjectに実行してSourceと意味要素を**別の観測**として保持する。
5. `tei trace` の公開合成Relationと `tei requirements` を補助観測し、`tei validate -json` のWarning/diagnosticsも出す。
6. 既存 `skills/yi-tei-modeling/verify_skill.py` と14件の合成静的試験も同じcheckoutで実行する。**Product Skillの文書整合検証であり、AI HarnessがSkillを実Loadしたことではない。** 必須なfingerprint位置と型を照合し、欠落/不一致なら失敗。取得していないFunction/Business/Requirement coverageを`complete`にしない。
7. CI Step Summaryへ「要求→実観測→未解決→追加Issue」を機械作成する。外部AI Harnessや実MCPを呼んだと主張しない。

これは**P-1aの限定実装**であり、P-1bの製品共通Design Context Queryそのものではない。Pythonで補助的に行う比較は既存CLIの内部Resolverを再実装せず、試験用の消費者検査のみとする。

## 4. 発見された初期阻害点と直近の対処

| 阻害点 | 診断・反例 | 最短の対処 | Exit |
| --- | --- | --- | --- |
| B1 宣言件数を完全性と誤認 | empty Workspaceでも`validate.ok=true`、builtin Catalogのみ、managedかつ未Load | `workspace-files.readByLoader`と宣言数・必要なSubjectを照合。未解析はunknown | Required集合に対する欠落が診断に残る。#3262/#2822/#2823 |
| B2 Command別JSON形の差 | version identity、mode、raw world | Discoveryの`jsonContract/successShape/modes`に従う。共通Responseを捏造しない | 誤Envelope・schema変更を検出。#2354 |
| B3 Tool受領欠落 | 切断・Reply欠落・stderr混入 | CIでは実Byte/Exit/JSONを別保存。Harnessでは分割・terminal markerを後続実証 | 欠落時`delivery-complete=false`。#2509/#2605/#3262 |
| B4 Issue→正本の飛躍 | AI生成文の「受理済み」偽装 | Source-backed/推論/未取得/Owner判断の独立列を保持 | 採否をAIが偽装しない。#2514/#1124 |
| B5 自己記述から自己構築への飛躍 | Go build/Plugin SB-3のみでG4合格と誤る | #2831のL1/L2/L3、SB-A/B/Cに成果物を渡す | stage1実TEI Processがstage2を駆動する独立証拠。#2831 |
| B6 Skillは設計情報を閉包しない | 現行`yi-tei-modeling`はcatalog/inspect/trace/validate中心 | この実験を既存Skillへ参照追加するのは#2605で行う。新Core Readerを追加しない | 同一TaskのH2/H3比較で未カバー/誤答と呼出負荷を測定 |
| B7 Skill検査がcheckoutを汚す可能性 | #42357274048はSkill静的14/14合格後、clean working tree検査で停止 | `PYTHONDONTWRITEBYTECODE=1`を適用。検査前後の状態と実行物を区別 | #42357337108で再試験成功。OS副作用全体の不存在は未確認 |

## 5. 独立Expectedと圧力ケース（同じ実CLIの結果から自明に作らない）

| Case | 操作 | 判定 |
| --- | --- | --- |
| DF-01 正常 | 固定Public Workspaceで`sales.order`をcatalog→inspect | Subject ID一致、非曖昧、World/SourceIndex照合。これだけで三視点完成とはしない |
| DF-02 必須対象欠落 | 独立Requiredに記したSubjectをCatalog投影から消す（合成Mutation） | 欠落を`incomplete`。空の意味モデルとして成功扱いしない |
| DF-03 版不一致 | inspectのSourceIndex fingerprintだけを別値にする（合成Mutation） | 異版接合拒否。古い説明を継承しない |
| DF-04 非照会領域 | Business/Functionを取得しない | `not-queried/unknown`。未存在と確定しない |
| DF-05 受領切断 | JSON受領を8,192 byteで切断 | JSON parse/terminal失敗をdelivery不適格へ分離。実Harness試験は#2509 |
| DF-06 欠落意味 | NOT EXISTS・手作業・Ruleが出力列に現れない | 欠けた意味/非出力依存を`none`へ正規化しない。#2582/#3262 |
| DF-07 二重の権限 | read-only Commandが実行可能、現開示Grantは未確認 | Tool成功から外部開示の正当性を推定しない |
| DF-08 自己構築 | stage1/stage2のdigestが一致、Stage1実TEI再入場未確認 | L2/L3を混同しない。#2831 |
| DF-09 必須Source未読込 | Required Source `semantic/sales/order.tei`をInventoryに残し、`readByLoader`だけfalseへ変える合成Mutation | Loader消費が失われた事実を`not-loaded`と検出。Parser受理や全閉包には拡大しない。#3262/#2823 |

DF-02/03/09はこのPRのCIで**JSON投影に対する合成Mutation**として実装する。DF-04は明示的にunknownで出力。その他は要求として記録し、実行したか否かを分ける。

## 6. クリティカルパスと一回ごとの受入

| 順 | 実行面 | 通過条件 | 次の担当 |
| --- | --- | --- | --- |
| D0 | Issue要求を仕様義務へ翻訳 | 各義務にSource、既存Owner、独立ExpectedとUnknownがある | #3262 |
| D1 | 実TEI read-only CLI | 1 binary、限定Subject、9結果、独立SourceのLoader確認、世界/入力元指紋の比較、意図的破壊3件検出 | #3262/#2605 |
| D2 | Product Skillが同じ調査を指示 | CLI実出力と同じ報告、欠けた情報・Scopeを隠さずH2/H3で測定 | #2605/#2509、TAI #168 |
| D3 | Issue→採否待ち設計文書 | Data/Function/Business・Requirementsの出典と未取得を表示、意味候補を正本にしない | #1803/#1124 |
| D4 | 一件のSource変更→再検査 | Preview/Acceptance/再解析/影響/Driver/独立Observationが同版で接続 | #1119 G1/G2、#1124、#1180 |
| D5（並行） | TEI全体Build Graph | Graph→実消費→TEI Binary Artifact（L1/L2）→Stage1実TEIの再入場（L3） | #2831 G4 |

**D5はD1〜D4のハード前提ではない**。D1で得た実Binary/Source Fingerprint・使用ツール版は、#2831でBuild Artifactと設計根拠を結合するための入力にできるが、G4の結果へ自動昇格はしない。TEIのGo単一バイナリ配布を基本とし、Go toolchain自体のself-hostingや専用TEI Hostを必要条件にしない。

## 7. 計測と撤回条件

- 実CLI呼出回数、失敗/Warning、未取得義務件数、CLI応答byte、生成する設計票のSource対応率・誤ったcomplete宣言数を記録する。
- H2/H3では実Tool呼出・Skill読込・Token/時間・独立Expected一致を別計測（既存TAI #168/#2018の分母を使う）。AIが答えを書いただけではTool実行済みとしない。
- 初回は**TEI実CLIだけを使用**。整形投影が既存Reader出力だけで成り立てば新Product Queryは後続圧力まで見送る。不足が具体的に観測された範囲だけ、#3262で共通Reader投影を増やす。
- 独立Requiredを満たせない場合、空結果を成功にしない。誤った外部開示・別版混合・Issue意味受理の捏造を検出したら、その成果物の適格性を撤回する。
- 完了区分は**実装・合成試験・実CLI CI・実Harness・Owner受理・製品G1/G2/G4**を別々に記録する。

## 8. 初回の事実資格

2026-11-07の設計前調査では、GitHub接続を使って上記IssueとCLI契約・Skill・既存CIを読み、設計のFit/Gapを作成した。**この時点でGitHub文書に対するread-only調査は実施済み**だが、接続はTEI CLI/MCPの実行口ではないため、本書の起票時点では新しい実TEI CLI・MCP/Harness・Self-Build/Host/Provider試験を実行したとは主張しない。CIを実行した場合は、対象Run・exact commit・結果を本書の歴史的な本文へ上書きせず、PR/Issueへ追加記録する。

## 9. 段階的な実証記録（時点ごとに上書きしない）

- **初回の限定実CLI結果**：Draft [PR #3264](https://github.com/hat47x/tei-platform/pull/3264) head `4677b56d1093eccbaac53873c59ed55ee575b156`、[実CI #42356836861](https://github.com/hat47x/tei-platform/actions/runs/42356836861) Job `128356649098` が `completed/success`。実checkoutはPR仮統合 `2d7b6d642e11a7a0b8575cfebc1fb1b62bafbb1b`、Go `1.26.10`、実TEI Binary SHA256 `c9f745982a616fac0e7c0a996b24192b70b04a4070615477fc9a176a7dcd9071`。9種類のread-only JSON、DF-02/DF-03合成負例がPASS、Validate診断3件、Claim=partial。**当該RunにSkill静的検査とDF-09は含まない**。
- **第二段階（実CI成功）**：既存Product Skill静的チェッカーと14件の合成テスト、独立Required Source `semantic/sales/order.tei` のInventory/実Loader読取、DF-09未読込Mutationを追加。[修正後実CI #42357337108](https://github.com/hat47x/tei-platform/actions/runs/42357337108)／Job `128358194403` は **completed/success**。対応PR Head `470cc8c167fc3beca3bf90af528e8eac323ce9d3`、実checkout（PR仮統合）`64b1f98c9b43959b2e040085a4de2c128cf84006`、実CLI Binary SHA256 `7c881ddd9dbe454d345f3c2bcc4f123e3b97a5a9a45fd5f39b1927eef4485860`。14/14静的負例PASS、9件の実CLI応答、DF-02/03/09すべて検出。固定Sourceの `readByLoader:true` を確認したが、Parser/Owner Meaningや全Requiredは未証明。
- **途中失敗の負例**：[CI #42357274048](https://github.com/hat47x/tei-platform/actions/runs/42357274048)ではSkill静的検査・14テストはPASSしたが、その後の実CLI工程でexit 1。ログ上は `git status` の清潔性確認後、Go buildに達する前に停止。Python testが作る`__pycache__`による作業ツリー汚染を有力原因とみて `PYTHONDONTWRITEBYTECODE=1` を追加し、再試験が成功した。失敗時の`git status`実行出力は未保存のため汚染ファイルの実名までは断定しない。AI Harness/Skillの読取り宣言でも検査補助過程の副作用を独立に点検する必要がある。
- **未実行**：実HarnessのSkill Loader、外部MCP、実OS/Disclosure、全SourceのParsed/Meaning/Rule依存閉包、Stage1実TEI Process再入場はこのCIの受入外。


## 10. テスト能力そのもののドッグフーディング（#2514 / #1180 / #3223）

**動作するテストが存在するだけでは、必要な不具合を発見できるとは限らない。** Issue→Design Context→Scenario/Case候補→実行→独立Expected→Mutant判別→未カバー逆引きまでを独立したProduct能力として検証する。今回は既存のScenario CLIと公開合成Sourceだけを用い、新Test Runner/Scenario SSOT/Oracle Engine/Hostは増設しない。

### 10.1 テスト能力を検証する四つの別の主張

| 能力 | この実験の最小確認 | 合格と取り違えないこと |
| --- | --- | --- |
| 発見・Coverage要求 | Issue #3262 / #1180 / #2514から既知の6 Scenario stepと追加負例を独立義務として固定 | 6 stepの実行をTEI全機能/全要求/非機能の100% Coverageとしない |
| 生成・実行 | Scenario SSOTから生成したCLI argvと各Source/World/Lock/Fixture pinを実ビルド済み`tei`別プロセスから実行 | 単なる`scenario verify`、in-process Go test、synthetic実行を実DB/Web/Host副作用としない |
| 判別力 | 正常／業務拒否と、故意の誤データ・誤Expected・古いpin・改変Planを別の検証基準で識別 | exit非0の件数だけではバグを検出した証拠でない。理由/対象Rule/期待差を照合する |
| 範囲・費用・再資格化 | C0相当のGo statement coverageの局所値、C1未計測、要件/NFR未検証、binary+source hash、失敗理由を記録 | C0をC1と呼ばない。未知の要件分母、未取得Rule、実Provider/開示資格を0件として処理しない |

### 10.2 独立Expectedと既知Mutantの最小試験

`reference/driver-scenarios/order-entry-pattern-smoke.json`はScenario正本の**公開Reference**であり、検証の唯一の正答供給元にはしない。独立Expectedとして、数量>0、行番号>0、同一Relation内の行番号一意を、固定既知ケースに対して別の簡易Property判定と具体的なExpectedにする。これらは製品の業務意味Owner受理ではなく、限定的な合成独立検証基準。

| ID | 意図的な差分 | 必須の判別 |
| --- | --- | --- |
| TA-01 | 有効Fixture（数量2.5、行番号1、Key 1/2） | **実CLI**の版固定`validate-positive`で3 Rule satisfied |
| TA-02 | 数量0、他は同じ | **実CLI**の版固定`validate-denied`で数量だけnot-satisfied。業務拒否を技術エラーにしない |
| TA-03 | 有効FixtureのRelation Keyを1/1へ差替え | **実CLI**が一意Rule `result=not-satisfied` を検出し、元Scenario期待に反するため拒否 |
| TA-04 | 数量0のScenario Expectedだけをsatisfiedと偽装 | **実CLI**が実Evaluator結果と差分を検出し、Expected自己申告を採用しない |
| TA-05 | 版固定コマンド生成後にFixtureだけ変更 | **実CLI**が古いFixture pinを拒否し、前回成功を再利用しない |
| TA-06 | 生成済みCLI Planのargvを勝手に変更 | **実CLI** `scenario check` が生成元との不一致を拒否 |
| TA-07 | 必須6 stepのうち`validate-denied`を候補から削除 | **独立Requirement集合**と比較して未カバー。短い候補が実行可能でもcompleteにしない |
| TA-08 | テスト用判定器を全Rule常時satisfiedへ置換 | **独立Property**がゼロ数量の偽陽性を検出。実TEI EvaluatorのMutation killとは区別 |

実装対象は`tools/reference-issue-test-ability/probe.py`と、既存`.github/workflows/tei-issue-design-dogfood.yml`の限定CI Step。実Binaryは前段と同一、Source改変はrunner一時領域のコピーだけ、実Provider/Secret/DBは使わない。

**独立性の上限**：TA-03/04は実Evaluatorが意図的に壊されたScenario/Fixtureを区別する検査であり、Production実装のあらゆるコードMutantを検出したことにはならない。既知の**実EvaluatorコードMutant kill**は#2514 / 禁止merge Draft #3017の既存実証を参照し、本PRでは再導入しない。TA-08は補助テスト用判定器の反例であってGoの実Mutantではない。

### 10.3 C0/C1、要求・非機能Coverage、テスト生成の未成立点

- `go test ./cmd/tei -run '^TestRegenerateScenarioArgvAndStdinPlan$' -coverprofile=...`で、**この限定テスト選択に対するGo文coverage（C0相当）**を取得する。`go tool cover -func`の`total: (statements) ...%`だけを局所Evidenceに記録。全Package・全TargetのC0とは違う。
- Go標準statement coverageの値をC1分岐Coverageへ流用しない。**C1はunmeasured**。必要なら既存#3223/#1180でBranch Toolや重要分岐の明示Negative Scenarioを検討するが、今回の実験では一般C1達成率を宣言しない。
- 機能要件Coverageの分母はIssue Requirement→Qualified Contextで解決した義務であり、Sourceを1件取得しただけでは未知のRequirement/Rule/Business Procedureを含められない。NFR（性能・可用性・セキュリティ・開示）も今回の合成CLIで測定していない。
- 将来のProduct Test Pattern Recommendation：独立Required/Source閉包→未達義務と適用述語→Pattern候補最大3件の理由付き提案→候補Scenario Review→再生成CLI/Web等→同じ独立Holdout/Mutantで判別。その比較は#3223/#1180/#2514の既存Ownerで行い、専用Product Skillやハーネスには**正答Holdoutを渡さない**。
- 測定する生成AI Test Capability：独立義務からの候補発見率、C0/C1の実測範囲、既知Mutant kill/survive、false-qualified、不要Case重複、実行時間・Token・人手、対象Source/Expected/Runner版の一致。候補数やコード量だけはKPIにしない。

### 10.4 昇格の条件

**T0（今回）**：固定Scenarioの実Binary pin付き6 step＋4種の実CLI誤り検出＋2種の独立判別基準＋C0局所計測（C1未計測）。これは**限定Test Ability**であり、Issueからテストを生成AIが自動設計した証明ではない。

**T1（次）**：Design ContextのRequired集合から、AI/人間がテストパターンを選び、既知Holdoutを見せず独立Scenario候補を作る。候補に同一固定Referenceの正答をあらかじめ埋め込まない。実CLIによる生成・実行・独立Expectedとの比較、false-qualifiedとuncoveredを#2514へ提出。

**T2（後続）**：同一ScenarioからCLI/Web/Studio/外部Targetへ投影・実行し、Provider/Host/Effect/Receipt/Readの別証拠を保持する。G1/G2業務一往復へ接続するが、未知の外部結末とMeaning Owner受理は別。

**T3（Self-Build並行）**：TEI自身のTest CapabilityのSource・Scenario・Build定義・実ArtifactをTEI標準構造で管理し、Stage1実TEIがテスト能力も再現する。#2831のL1/L2/L3とは異なる資格を混同しない。

**現時点**：本節は実装PR候補として作成。新たなCIの実検証結果はこの文書に履歴として追記する。MCP/SkillによるAIテスト自動生成、C1計測、実DB/Host、正式Owner意味受理は**NOT RUN**。
