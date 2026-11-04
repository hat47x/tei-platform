# 経路間の業務機能完全性とScenario由来E2Eテスト

- Status: design contract / **bounded persisted Scenario → CLI implementation confirmed by source** (2026-11-04); Web/other Target E2E emitter, general Scenario/compiler and AI extensibility **not yet verified**
- Requirements: RQ-012, RQ-013, RQ-019〜021, RQ-034
- Implementation owner: [#1180](https://github.com/hat47x/tei-platform/issues/1180)
- Reuse: #1158 (typed operations / UI adapter), #259 (Target capability / Execution / Trust), #1163 (single-revision Evidence), #70/#270 (First Web Target), #1043/#1174/#1178 (security)
- 本文は業務操作をCLI/Webそれぞれの独立した正本へ分裂させないための設計契約。新しいproduction schemaやRunnerを実装済みと宣言する文書ではない。

## 1. 基本原則

**TEIが提供するApplication実行経路は、同じAccepted Applicationに対して業務機能を原則として省略しない。** 
CLI-firstは実装・検証の着手順序であり、CLIを恒久的な業務サブセットとする許可ではない。同様にWeb、Desktop等のUIだけが持つ業務機能を作らない。

CLIで到達できるべきものは、画面のpixelやタッチ操作そのものではなく、それらを通して起きる**業務上の意図・状態遷移・結果・観測**である。例えばWebの「受注を確定」ボタンには、同じCommandの実行入口がCLIにも必要となる。CLIがWebフォームをレンダリングする必要はない。入力方法、対話進行、表示、アクセシビリティ、Host device制御の実現手段はTargetごとに異なってよい。

**禁止する縮退**:
- WebだけでValidation／認可／データ変更を実装し、CLI側から同じ業務契約に到達できない
- CLIなら管理者権限で通るとして、Webの権限制約と別の業務成功を合格扱いする
- 必須操作をメニューから隠す、not-supportedを返す、またはstub成功するだけでTargetのparityを宣言する
- Target固有の便宜フィールド／状態／OutcomeでCanonicalな不足を埋める
- ある経路だけで生成・取得できる業務Data／FunctionをAcceptanceなしに意味へ追加する

対応不能なHost制約は認可済み代替実現（例: device入力を明示Resourceを介してCLIから与える、適切なremote Providerに委譲）により同一の業務結果・作用・失敗条件を満たすか判定する。満たせないTarget/profileは配備・実行前の不適合／未解決であり、低機能版を「同等」とは認証しない。純粋な見た目やdevice固有UX品質の差は別のTarget Evidenceへ送る。

## 2. 同等性の比較単位

比較対象を、同じAccepted `Application/Scenario` とそのRevision、Semantic/Plugin Lock、業務入力・precondition、Subject、Callerの正規化された実行Authority／Policy Revision、Environment／Resource条件に揃える。Web sessionとCLI identityを同一視せず、同じ業務主体・有効なAuthorityを持つ比較用Contextをそれぞれ正規に作る。

同じContextで比較するもの:
1. 操作カタログと到達可能性（read、Command、Validation、branch、manual/external handoff）
2. Input／business precondition／Constraint／Validation／Authorizationの判定
3. Result type、named Business Outcome、Execution Error／Diagnosticの分類
4. durable state delta、外部effect、transaction／rollback／retry／unknownの境界
5. Command acknowledgementとは独立したObservation、および監査に必要なContext
6. Source／semantic identity／Dependency／provenanceの遡及

TargetのHTTP status、CLI exit code、画面文言、DOM selectorは**それぞれの結果表現**であり、Business Outcomeの正本ではない。異なるAuth方式／Policy Revision／Resource到達性による結果の不一致は、正規化せず比較PASSとして扱わず、差の説明と独立Evidenceを残す。

「一経路で業務定義を完全検証できる」とは、**全Accepted業務操作と有効な分岐・失敗・副作用を、対象経路のScenario suiteで意味論的に網羅可能**という意味である。単一のhappy-pathケースで全業務を証明したことにはしない。各Targetのauth、transport、browser/device lifecycle、Visual、accessibility、performanceは別途適合が必要。

## 3. Driver ScenarioをシナリオテストのSSOTとする

**Accepted Canonicalは業務意味のSSOT、保存済みDriver ScenarioはシナリオテストのSSOT**とする。ScenarioはCanonicalから最初に導出・提案してよいが、その後のCLI・Web・Desktop等向けのテストを個別に人手で保守しない。Scenarioの修正と明示Reviewが必要な場合はScenario自身を更新し、すべてのTarget成果物を再生成する。テスト実行結果から期待値を逆算してScenarioを書き換えてはいけない。

既存の`internal/driver/final_gate_b_scenario_test.go`、`gate_b_ui_less_scenario_test.go`等はGoテスト内で組み立てるtest-local Referenceである。一方、2026-11-04のsource再確認により、**`reference/driver-scenarios/order-entry-pattern-smoke.json`（6 Step）、`internal/scenariotest/scenario.go`（`tei.driver-scenario/v0`）、`cmd/tei/scenario.go`（`verify/generate/run/check`）および`cmd/tei/scenario_test.go`の既存限定実装を確認**した。これは保存・再読込・CLI派生の限定Referenceであり、全業務Step/任意Caseの生成、Web E2Eへの生成、実Provider Write/Host資格を含まない。今後はこの現物を起点に、既存DriverのOperation／Result／Observation／Providerへの対応を拡張する。Goテストを機械的に逆解析して無検証な業務定義へ昇格せず、第二Driver・独立業務Semantic DSLを作らない。恒久schemaは複数Consumerの実証後に判断する。

Scenario側が所有するテスト要素:
- scenario identity／revision／origin（生成・編集・review履歴）および参照するAccepted Application／Canonical revision／Scenario source
- 業務上のテスト意図、coverage obligation（operation／validation rule／decision branch／denied outcome／failure／effect）、Step間のデータ依存・並び・分岐
- 入力・前提／fixture参照、期待Business Result／named Outcome／Execution Error／別途Observation、pre/post-state、必要な権限、負例
- 副作用分類／安全なreset／cleanup、未確認や再実行不能の境界

Target側が所有するのは、CLI command mapping／Web locator／framework binding／認証session準備・UI assertionなどの**実現・検証方式**であり、業務期待値の第二正本ではない。Scenarioの修正はCanonicalの変更を意味しない。一方Canonical側に存在しない業務期待値や、accepted meaningと矛盾するAssertionはunresolvedとしてReviewし、勝手に仕様を追加しない。

### 再生成はCLIも対称に扱う

```text
Accepted Canonical Application @ R + Scenario Source @ S + test fixture
                          │ explicit references / validation
                          ▼
               resolved Driver Scenario @ S
                          │ canonical step/branch/expected result
             ┌────────────┼────────────────────────┐
             ▼            ▼                        ▼
         CLI emitter    Web E2E emitter       other Target emitter
             │            │                        │
    generated argv     generated browser      generated Target
    / command plan      E2E steps              E2E steps
             │            │                        │
          execute       execute                   execute
             └────────────┼────────────────────────┘
                          ▼
              evidence + scenario coverage / parity
```

**CLIコマンド列自体も常に派生物**。Scenario `S` を変更したとき、CLIテストだけ旧`S`のコマンドを残さない。CLIの既存コマンド名をScenarioへ手書きし、業務Stepとの対応を固定しない。現行CLIがサポートしない業務Stepは、実装済みと偽らず`unresolved operation binding`にする。#1158の型付きOperationと既存Driverの解決を使い、CLIのargv／stdin／exit／stdoutへ変換する。

最初の検証用CLI interface（**候補であり既存実装のコマンド名ではない**）:

```text
tei scenario verify --scenario <path> --no-execute
tei scenario generate --scenario <path> --target cli --out <dir>
tei scenario run --scenario <path> --target cli --fixture <ref>
tei scenario generate --scenario <path> --target web-e2e --out <dir>
tei scenario check-generated --scenario <path> --target all
```

CLIからは、明示的に紐付いたScenarioを一命令で実行する入口と、Step別の**実際に再実行可能なCLIコマンド（またはstructured argv+stdin plan）**を生成する入口の両方を検討する。生成AIにはまず`verify`で静的欠落を診断し、`generate`で派生CLI経路を得て、承認されたfixtureで`run`する最小手順を提供する。生成に副作用が伴わないことと、生成したCommandの実行権限は分離する。

shell fragmentを唯一の成果物にせず、文字列結合・escapeミス・secretの埋込みを防ぐため**typed argv配列＋stdin／fixture参照**を第一候補とする。POSIX shell／PowerShell scriptは必要時のみHost向けProjectionとして出す。Credential、actual secret、opaque runtime tokenは生成物へ埋め込まず実行時の検証済みContext経由で供給する。CLI runnerを用いても別のAuthorization／Resource bypassを許さない。

### 再生成の同一性と乖離検出

入力を`(Scenario S, Canonical R, semantic/plugin lock L, fixture definition F, Operation/Target binding M, generator version G)`として固定し、同じ入力には同じ正規化済み生成結果を要求する。時刻・random・Host path・secret等の非決定的実値は生成結果のidentityへ混ぜず、実行時Evidenceへ分離する。

- 生成物へorigin Scenario digest、Canonical revision、binding/generator version、target kind、fixture fingerprintとcoverage manifestを関連付ける。
- 各Targetの**全必須Step／branch／assertionが実装されたか**を比較し、未生成・曖昧mapping・意図しないskipは不合格。表示系のTarget-only assertionは別のcoverage dimensionとする。
- 同じ入力の再生成で不一致ならgenerator不決定性または手修正を検出し、不正な成果物を実行に進めない。手修正が必要ならTarget Bindingを修正するか、Scenarioの変更Proposalに戻す。
- ScenarioまたはCanonical変更時は旧CLI／Web E2Eをstaleとし、新生成物を用いて再実行する。Scenarioのみ変更してWebテストを旧版で続行することは禁止。
- 実際のDriver／CLI／Webテストのpass/failと実行済みEvidenceは**生成物から独立**に保存する。生成PASS、実行PASS、業務仕様の網羅、Web Host適合は別判定。

## 4. Scenarioからの生成パイプライン

共通テストの入力は**保存・受理済みDriver Scenarioの特定revision**と、明示参照したCanonical／Policy／Lock／Fixtureである。CLIの実行TraceからWebテストを生成したり、Web DOMからCLIテストを逆生成したりしない。両経路とも同じScenarioを展開し、Step／期待値／Observationを共有する。


既存Driver Scenario、Semantic Operation、Typed Source／Evidence、Shared Acceptance、Application Composition、Capability Bindingを消費する。argvログをWebのクリック列に変換したり、クリック録画から新しい業務意味を推測したりしない。共通テストは独立した業務DSLや第二のDriverを作らず、既存Scenarioの**テストProjection／派生Evidence**とする。

**最小Scenario record（検証候補。固定production schemaではない）**:

このrecordは個別Generator固有ではなく、永続するDriver Scenarioのテスト定義を指す。CLI/Web専用metadataや実行時Evidenceはここへ正本として格納しない。
- scenario identity、Accepted Application／Canonical fingerprint、baseline `B / R`、Scenario source
- input fixture／precondition、operation参照、step順序と分岐、期待Validation／Auth判定
- Business Result／named Outcome／Execution Error、観測対象とexpected state delta
- effect classification（read-only / reversible / irreversible / external / unknown）、fixture reset／cleanup／compensation
- expected Subject／scope／effective authority、Policy revision、Resource条件
- mapping identity/version、Target build `Tcli / Tweb`、runtime/provider、generator version、executed evidence links

期待値はAccepted meaningまたは独立した検証fixtureに由来し、CLIの実際の出力を盲目的に正解として焼き付けない。CLIの検証済みResultは共有expectationとの照合証拠であって新しい正本ではない。

## 5. Generator / Adapter責任境界

- **Shared semantic fixture**: 同一入力・前提・業務Operation・assertion／state-delta観測を解決する。権限を発行しない。
- **CLI binding / generator**: Scenarioの全必須Stepとtyped Operation → 安定したargv配列／options／stdin／fixture参照／実行順／assertionsへ再生成する。CLIによる直接Scenario runも同じステップを利用し、CLI専用Validationや第二テスト定義を実装しない。
- **Web binding / generator**: 同じScenarioの全必須Stepとtyped Operation → page/route、stable semantic locator（UI Kit等が提供する明示binding）、入力・送信・確認、assertionsへ再生成する。Web固有画面を第二業務モデルにしない。
- **Target-specific assertions**: Web login/session、origin/CSRF、navigation、focus／accessibility、transport error、visual evidence、Desktop IPCなど。共有業務assertionと区別して保存。
- **Comparator**: Business Result、Outcome、Error、Observation／副作用を同じ意味に正規化して差分判定。複数経路の実行条件に差があれば「比較不能」とする。異常を期待値に合わせて覆い隠さない。

GUI配置変更時はWeb bindingだけを修正でき、Accepted Scenarioのmeaningとexpectationは不変であるべき。Target-specific custom UIも標準Projectionと同じOperation contractへ結びつける。

## 6. Fail-closedと安全な再現

- 未対応Operation／必須branch／locatorの曖昧さ／欠落mapping → **generation failure or unresolved**（CLIもWebもskip／空testは禁止）
- 古いCLI command plan／Web E2E、生成物の手修正、再生成時のdigest不一致 → **stale/drift**, 実行済みとして認めない
- ScenarioのCoverage obligationをTarget generatorが捨てる／変換不能なassertionを無視する → **incomplete**, Parity PASSにしない
- 旧Revision、異なるLock、Target build不明、Resource・Policy mismatch → **stale/incomparable**
- Denied操作をCLI管理権限で強行 → **authority mismatch**（CLI PASSにしない）
- side-effect Confirmationがない、Observationが取得できない → **not observed**。Command受付をeffect成功にしない
- 成否不明の外部送信、rollback未確認 → **unknown**。再試行による重複effectを作らない
- Shared assertionがないstep／期待値が実結果から自己生成 → **untested**
- E2E非実行なのに生成されたという事実だけ → **generated, not executed**
- screenshotやDOMのみ合格、business error不一致 → **functional failure**
- CLI-only partial slice／read-only fixtureのみ → **coverage incomplete**（全業務適合を宣言しない）

Executionは隔離されたFixture/Disposable DB、認可済みTest Resource、偽装していないSubject/Callerを使う。外部実効果は慎重に扱い、独立Observationのない場合は不確定のまま記録する。

## 7. 最初の実証と受入

#1180では`order_entry`等の**実在しAcceptedな**Scenarioを一つ選ぶ。最初は既存のtest-local Go Driver Scenarioから、一つの**保存可能なScenarioテスト定義**を抽出・解決してCLIコマンド列を生成／実行／再生成する。read-only Lookup/Validationの正負で始め、次に認可を伴うCommandと独立Observationを加える。最初の生成TargetはWeb。Desktop／Mobile等はこの契約を再利用できることを確認してから個別Adapterを実装する。

受入には、(a) **独立保存Scenarioのload→typed Step→CLI command plan生成→CLI実行→同入力で再生成し同digestを確認**、Scenario変更時に新成果物が必要になり旧成果物をstaleとして検出すること、(b) 同じScenario inventoryの全対象OperationとCLI/Web bindingに欠落がない、(c) **同じScenarioから生成された**CLI検証とWeb E2Eを実際に別経路で走らせ同じ業務結果を確認、(d) Validation否定・Auth拒否・Execution Error・欠落mappingの負例がfail-closed、(e) 同じRevision/Lock/Policy Contextで比較可能、(f) Web固有Auth／UI EvidenceがCLI PASSと独立、(g) trusted clean checkoutのactual exact-head test/health receipt、を要求する。

#1155／#1165のC1は限定されたCLI read-only discovery／inspect sliceでよい。初期scope制限はApplication公開Targetの永続的な機能欠落を正当化しない。#1163の横断Evidence Packetは本Generatorの成果を取り込み得るが、Generator自身が新しいEvidence Authorityにはならない。

## 非目標

- Driver Scenarioを業務意味の第二Canonical SSOTへ昇格すること
- CLIコマンド列をScenarioの正本として保守したり、生成AIが実行ログから期待値を採用したりすること
- CLI単一happy-pathのPASSでApplication全体のConformanceを宣言すること
- Web DOM／UI録画を業務Meaningへ逆採用すること
- 全Targetの共通UIを強制すること
- Browser／Desktop native Test frameworkをCoreへ固定すること
- 外部Production effectの自動再実行、Credentialの記録
- 第二Policy Engine／第二Driver／第二Resolver／第二Acceptance Authorityの導入


## 8. AI-first Test Authoring Scaffoldと横断Coverage

- **Status (2026-11-04): 設計追補／提案・実装未実施・新規CLI/Web/GUI/DB試験NOT RUN。** Ownerの追加要求：**生成AIが必要なテストを自ら探索・生成・拡張できる足場を最優先**する。実装済みCLI/Provider/Scenario Compilerと混同しない。
- §1〜7のAccepted Canonical（業務意味の正本）・保存済みDriver Scenario（シナリオテストの正本）・CLI/Web対称生成と安全境界を維持。本追補は新しい第二Driver/Runner/Oracle Authority/Registryを設けない。
- 責務接合：#2621（表/Markdown Authoring）、#1131（Fixture/Dataset/Snapshot）、#2514（独立Expected/Mutation判別）、#1180（Scenario→CLI/Web生成）、#988/#2620（DSL/Operator/Role）、#1163（Evidence）、#2391（UI外部制御/本番物理除去）、#1174（Effect/unknown）、#2506/#2593（Context/データ粒度）、#1119（資格Gate）。

### 8.1 最優先：AIが操作できるテスト作成ループ

モデル・テスト資産をAIが観察し、追加したCase/Scenarioを検証して小さな差分を反復できることを**CLI・JSON・通常のファイル編集**で先に実証する。生成物専用の難解なコード生成APIやGUIを先行要件にしない。

1. **inspect / discover**：Accepted modelのFunction/Rule/Operation/型/Constraint/Relation/State/Operator Contract、Source位置、既存Scenario/Case、Fixtureの利用資格、未検証Coverage Obligation、対象Target/Providerの実在能力と未解決を機械可読に列挙。結果のScope・版・取得完全性・秘匿境界を明記。実在しないOperationや権限をAIが推測しない。
2. **author / extend**：AIは人が読めるMarkdown表や通常の構造化ファイルにCaseと必要ならScenario候補を追加する。決定的ID、元Semantic ID、入力partition/生成seed、preconditions、expectedまたは探索目的、候補のprovenanceを付ける。例示表と実行テストはRoleで識別し、既存Accepted Scenarioを自動改変しない。専門家が表を読めることを最優先し、Go固有記法を日常Authoringへ漏らさない。
3. **verify / explain**：構文・参照/型・Scope/粒度・null/missing/unavailable/redacted・Fixture・Operator lock・Target binding・副作用可否・Expectedの出典独立性・重複Case/到達不能・Coverage対応を**副作用なし**で検査。失敗理由はSource位置と修復可能候補を返す。
4. **run isolated**：既存Driver/Runtime/Target Harnessを再利用し、合成/使い捨てデータ・明示Budget/seed・所要資源上限の下で実行する。AIは実装を自由に変更・試行できるが、実Grant・機密読取・外部送信・Commit再試行の許可は得ない。exploratory runは観測のみでPASS主張をしない。
5. **compare / shrink / report**：Actual vs独立Expected、既知誤実装Mutant、Run間差分、データSnapshot、Rule/Branch/Pattern/Scenario coverage、失敗Traceを同一Case/Scenario/Run IDと版に結ぶ。反例の縮小・再現seed・重複検出はAI支援可能とし、最小反例に変形しても元の失敗が保持されるか実再実行する。
6. **review / adopt**：有用な探索ケースを回帰ケースへ昇格するときはExpectedの独立根拠・対象範囲・Owner受理・必要な権限を区別。AI生成・複数AIの同意・既存評価器が出したActualの転用だけではExpectedを正式Oracleにしない。

CLI名・schema・Agent用Tool形態は**PoCのConsumer実証後に判断**する。#1180の既存候補 `tei scenario verify/generate/run/check-generated` に探索/coverageを増分接続できるかを先に評価し、第二のtest command namespaceや全能力を持つAgent専用Runtimeを作らない。対話的なAIだけでなく他言語の自動生成器や人間が同じファイル・JSON契約を利用できるようにする。

### 8.2 AIが拡張できる最小契約とCoverageの分母

固定の「全ドメイン共通必須テストパターン一覧」は作らない。Domain/Operator/型/Constraintから**適用条件付きCoverage Obligationを実行前に列挙**し、追加プラグイン/Corpus/ユーザー定義テストが新しいObligationを提案・追加できるようにする。新しいCaseの追加にCoreソース改修を要しないことを受入条件とする。

最小の**候補**（正式Core型ではない）:
- Case: `id, sourceRef, semanticSubjectID, scenarioRef?, inputs/fixtureRef, generatorKind?, seed?, precondition, expectedRef? / exploratoryGoal, assertions, tags, provenance, revision`
- Coverage Obligation: `id, subjectID, dimension, applicability, requiredPartitionOrOutcome, sourceRevision, scope/grain/asOf, criticality, provenance, requiredEvidenceLevel`
- Run/Evidence: `scenario/case/obligation IDs, exact source+canonical+plugin+fixture+target binding+artifact revision, attempted/executed/asserted/qualified, observed results, error kind, independent observation, trace links, actual read/effect footprint, limits/unknown`
- Suite: `selectedCaseRefs, scenarioRefs, coverageProfile, fixture/target profiles, isolation/safety policy, resource budget`。Suiteに業務Step/Expectedを複製しない。

Coverageを「既存テストが通った項目の割合」から算出しない。分母はObligationの版付き候補集合として、列挙済み/未解決/対象外/未発見可能性を明示。Code（実Go instrumentationとTEI Source/Meaning対応）、Rule/Branch、Data partition、状態遷移、Scenario/Target step、Provider/Effectを**独立の分母**とし、異なる軸を単一%に足さない。Go standard coverageはbranch/MC/DCの証明ではない。executed、asserted、qualifiedは別。対象集合が不完全・分母0なら「100%」にしない。

AIによるケース拡張の候補は、型/制約由来の境界値・NULL/欠損/部分取得、決定表の重なりと短絡、Property-based/Fuzz/Mutation、Metamorphic、差分テスト、履歴不具合からの回帰、専門Profileの時系列/グラフ/業務/ゲーム等。生成数/予算上限を越えたら`truncated/incomplete`とし、探索しなかったObligationをPASSにしない。AIが未知の意味を発見しても即座にAccepted Ruleへ格上げしない。

### 8.3 CLI/Web/GUIは共通Scenarioの別Consumer（次段）

- 同じ保存済みScenario + Canonical + plugin/target lock + fixture + generator versionから、typed CLI argv/stdinとWeb意味Locator/操作手順を独立生成。全必須Step/Branch/Assertion/Observationの対応Manifestを両経路で突き合わせ、欠落や曖昧さでfail-closed。CLI stdoutやDOM/クリック録画から業務期待値を逆生成しない。
- 「CLIとWebが一致」と「両方が独立Expectedに正しい」は別検収。Fixtureを隔離して両経路を実行し、構造化Business Outcome、NULL/欠損、Key、Multiplicity、順序、金額精度、Data state delta、実Effect、独立Readを比較。権限/Policy Scope/時点/部分取得/Commit unknownが異なるときは`incomparable/unknown`。共通誤実装の自己追認には#2514の独立Oracle/既知Mutantを利用。
- GUIには、操作E2E、DOM/ARIA構造、視覚Baseline/画像差分、実データの独立Observationを**別Assertion列**として接続する。初期Web候補はPlaywright等を比較するがCoreに固定しない。日時/フォント/viewport/renderer/locale/animation/seedの固定、スクリーンショットの機密遮蔽、baselineのReview/Revisionと自動更新禁止を要求。画像一致≠業務成功、データ同値≠操作可能性。
- Wails/WebView/Native/Canvasは実Host固有Evidence。テスト用UI instrumentation/外部制御は#2391に従って本番Artifactから物理的に除去できるか検証し、通常のアクセシビリティを損なわない。Test-build成功をProduction安全性へ昇格しない。

### 8.4 AI-first垂直実証と段階順序

| Priority | 実証する最小成果 | 完了条件（予定、すべて未実行） |
| --- | --- | --- |
| **P0-A: AI authoring scaffold** | 同一モデルのread-only inventory/coverage gapをJSONで取得し、AIが独立したCaseファイルを追加・構文/意味検証し、失敗理由を修正できる | 既存テストを編集せず新規正常・拒否・境界Caseを加え、変更差分/Source位置/対象Obligationを再現。未解決・権限不足は明示 |
| **P0-B: 独立Expectedと判別力** | 生成Candidate、独立Oracle、観測Onlyケースを分離し、既知Mutantを試す | 誤Expectedの自己追認で偽PASSを許さず、無Expected探索をQUALIFIED PASSにしない。seed固定で反例を再現 |
| **P1: Scenario & coverage** | 永続Scenario 1件と追加Case群を既存Driverに接続し、typed CLI plan/実行/coverage reportを取得 | actual Source/Artifact/Fixture/Run版とCoverage分母が結ばれ、欠落Step/旧版を拒否 |
| **P2: Cross-target** | 同一ScenarioからCLI/Web E2Eを独立生成・隔離実行し独立Expected/Observed state照合 | CLI/Web両経路の漏れを検出、片方だけPASSなら全経路PASSにしない |
| **P3: Visual/Desktop** | Web DOM/ARIA・Screenshot回帰の差分、Wails/Native固有Assertion、Production instrumentation除去 | ビジネス差と視覚差の双方のMutantを別々に検出し、実配布Buildの除去を証拠で示す |
| **P4: 変更影響と選択的再実行** | Source/Rule/Plugin/Oracle/Fixture/Target変更でObligationと旧Evidenceを差分 | stale/unknownを明示。選択的再実行と完全ベースラインの見逃し率・コストを比較 |

最初のDomainは公開合成`order_entry`のread-only Query/Validationから、(1)正常、(2)NULLとmissing、(3)重複/部分集合、(4)丸め境界、(5)拒否/評価不能を候補化する。別途非業務Domain（状態機械/ゲーム等）1件で「業務専用のCoreパターン表」が不要なことを確認。Source/Meaningの変更なしに新たなCase・Profileを追加できることを重視する。実DB/書込み/GUIはP0合格の前提にしない。

反証Fixture：`AI-01` 既存期待値をActualからコピーした偽PASS、`AI-02` 新しいNULL partitionがcoverage分母から黙って消える、`AI-03` 旧Scenarioの片Target生成物だけ残留、`AI-04` Target未対応Stepがskipされ空PASS、`AI-05` Web表示一致だが実保存なし、`AI-06` CLI/Webとも同じ誤計算、`AI-07` 画像baseline自動更新で劣化を隠蔽、`AI-08` secret/権限を持つ本番Providerへ探索実行、`AI-09` 部分Readを全件確認と誤認、`AI-10` source/Oracle/Plugin版変更後も旧Evidenceを再利用、`AI-11` exploratory結果をOwner受理済み意味へ昇格、`AI-12` budget上限で探索停止した未検査をPASS扱い。

**完了報告の境界**：本節の追補は計画だけ。実CLIのinspect/generate/run/coverage、永続Scenario Compiler、GUI/Playwright/Wails、AI Agentの実行、安全なOS隔離、本番適格性、Owner意味受理は未証明。実装後はexact HEAD、実Build/Runner、fixture bytes、seed、独立Oracle、実行Trace、Coverage分母と未知範囲、clean repository-health結果を記録する。


## 9. 既存CLI Referenceに接続するAIテスト足場：実装可能なP0入口（2026-11-04）

**区分：現物Sourceの静的確認と設計計画。今回の追加試験は NOT RUN。** §8の「新CLIが未実装」という一律表現を訂正し、実装済みの範囲を正確に起点へ据える。他チャットのDraft #2745（DSL→同一Meaning/Scenario）、#2619（Test as Data）、#2514（TAO独立Oracle）、#2493（多数Agent Lane）、#2595（実Source入力・Tool帰属）、#2600（Artifact再利用とEvidence再資格化）、#2391（GUI/制御面本番除去）と照合した。変更は本Issue #1180の実装責務に限定し、別Ownerの意味/権限/Runnerを重複しない。

### 9.1 現物の再利用と未実装の差分

| 観測項目 | 2026-11-04に静的確認できたSource/契約 | P0で必要な差分 |
| --- | --- | --- |
| 既存Scenario | `reference/driver-scenarios/order-entry-pattern-smoke.json` の6 Step、`tei.driver-scenario/v0` | AI生成Caseを**元ファイルに上書きせず**候補として追加・重複/由来照合 |
| CLI操作 | `cmd/tei/scenario.go` に `verify / generate / run / check` | inventory→Case候補→実在CLIへの投入・診断の最小手順を記述・実証。新command/APIは必要な不足が出てから |
| CLI生成 | `scenariotest.Generate` が全Stepのtyped `argv` と明示 `stdin.mode=none` を生成 | 新Stepの対応を測定。未知Operation・未支持stdinはskipせずunsupported |
| 実行範囲 | `resolve-pattern`, `execute-browse-query`, `evaluate-validations`, `observe-synthetic-state` | **read-only/合成の限定Reference**。Command/Auth/実DB/Web/外部Effectは未実証と明記 |
| ピン留め | Scenario SHA、World SHA、Plugin Lock SHA、fixture SHAとPlanの全内容比較 | 実行Tool/バイナリ、Source実読込、Case独立Expected、Target buildと現在資格は別Evidence |
| 値の表現 | `Value.Decode` は `string/int64/float64/timestamp` の4種 | NULL/missing/redacted/unavailable・decimal精度などは現行形式に**ない**。拒否する負例と将来の型契約追加を分離 |
| 安全性 | duplicate JSON member、trailing content、列名重複、古いpin等の拒否試験がSourceに存在 | 既存の拒否を保ちながらAIの外部Input/探索Budget/FS/OS/Network/Disclosureを別に検収 |
| カバレッジ | Stepごとの結果・`verified`・`evidenceNature` を取得可能 | Rule/Branch/Data/State/Effectの版付き**事前Obligation分母**と「実行済み/検証済み/現在資格」を追加。6/6 Step成功からCoverage100%は導かない |

**現在のCLI実在入口**（表示コマンドは例。実行はこの設計追補内では未実施）：

```bash
go run ./cmd/tei scenario verify -scenario reference/driver-scenarios/order-entry-pattern-smoke.json -workspace reference/workspace
go run ./cmd/tei scenario generate -scenario reference/driver-scenarios/order-entry-pattern-smoke.json -workspace reference/workspace
go run ./cmd/tei scenario run -scenario reference/driver-scenarios/order-entry-pattern-smoke.json -workspace reference/workspace -step validate-denied
```

生成済みplanの`check -plan`は対象のScenario/Workspace同一性と手修正を検証するが、Plan生成自体を実行証拠にしない。`run`のpinなし直接起動は現状`executionMode=exploratory-unpinned`と返す。`verified=true`でも対象はSourceに書かれた限定合成Stepであり、独立Oracle・実Host/Provider/外部Effect全体の資格ではない。

### 9.2 最初の縦断テストを5枚の検収票に分ける

**P0-0 現物Baseline（変更前）**：clean checkout、正確なcommit/Go/toolchain、`cmd/tei/scenario.go`と`internal/scenariotest/scenario.go`のblob、6 Step Scenario/fixture bytesを固定。既存Focused test（`go test ./cmd/tei ./internal/scenariotest`）とCLIの`verify/generate/run/check`を**実際に**試し、成功・失敗・未実行をReceiptへ記録。新しいTest Engineを先に導入しない。

**P0-1 AI Case拡張（元の期待値を改変しない）**：生成AIには現存する検証済みread-only inventory、使用可能なOperation列、source locations、制約、既存Expected、fixture種別、未知/unsupportedを読ませ、**隔離ディレクトリ／別branch**にのみ新しいCase/Scenario/fixture候補を作成させる。最初は既存v0の許容表現に限り、（a）追加した正常Validation、（b）拒否されるValidation、（c）誤Expectedの否定対照を作る。各候補へ`candidateCaseID, sourceRef, intendedSemanticID, parentScenarioRevision, fixtureDigest, oracleOrigin, authoringActor, budget/seed`を**別のmanifest/レビュー表に保持**する。既存`Scenario` schemaへ未対応フィールドを無理に追加せず、Source Parserが拒否する場所にmetadataを注入しない。候補を正式Scenario/Canonicalへ自動受理しない。

**P0-2 静的検証・修復ループ**：現在の`verify`がチェックする範囲と、作成したCase metadata/coverage applicability/oracle独立性の未検査を並記。二段目のread-only checkerはまず小さな検証スクリプト/テストとして比較し、実Consumerの必要が確認された場合のみCLIへ接続。構文破損、unknown Step、duplicate Step ID、unsupported Value kind、missing fixture、Source ambiguity、Scope/Authority mismatch、期待値と実装の共有由来を型別に診断。AIによる修復後、同じ出典に対する再検証で失敗が解消されたことを確認する。

**P0-3 隔離実行・反例縮小**：既存CLIから実行できるCaseはpin付きplanで再実行し、`actual result` / `expected` / `evidenceNature` / `coverage obligation`を別列で保持。生成Agentは期待値が未確定の入力を探索してよいが、QUALIFIED PASSにしない。失敗入力を縮める場合もshrunk fixtureのdigest・seedと元の失敗型を保持し、**縮小後に再実行した同じ失敗**がなければ反例成立と呼ばない。

**P0-4 Coverage差分（分母の正当性）**：Caseを増やす前の**required/known/unknown** obligation inventoryを凍結し、実行後のcovered/uncovered/blocked/unsupportedを報告。Case数、Step数、Rule coverage、Data pattern coverage、実Provider scope coverageは別指標。新しいCaseが見つけた未知partitionは事前分母に対する後発の発見として記録し、新旧分母を同じ割合で比較しない。partial/read denied/zero rows/timeoutを全件未存在の証明にしない。

この5枚の完了を**一件のDemo成功ではなく、正負ペアと実Output digestに基づく受入**として実証。最初からWeb、SQL実Provider、Wails、Agent用MCP、新テスト言語、専用スケジューラを完成条件に含めない。

### 9.3 AI/Agent間の並列作業規則（#2493との接点）

- Task Packetに`base SHA, allowed source/read projection, writable test candidate paths, expected independent oracle, target Scenario/Obligation scope, budget, tool/OS principal, egress policy, output artifact paths, review owner`を持たせる。これは既存Issue/branch/manifestへ記録する運用規約であり、第二Task Authorityではない。
- 各Agentは独立したcandidate path/branch/worktreeへCaseを追加する。共有Scenario・独立Expected・Coverage分母・Baseline画像を**同時に無断更新しない**。同一Obligationに複数の独立反例を提案してよいが、正本採用はDiffとOracle出典を比較して単一の受理経路へ統合。
- 同じAIモデルを複数回呼ぶ、相互一致を得る、同じEvaluatorでExpectedを作り直す、という行為は独立検証ではない。生成側と採点側のSource/Tool/Prompt/Corpusの共有由来を検査し、必要ならsealed holdoutを#2514/#2515の保護下で用いる。
- 生成AIの大量探索は**候補の発散**を許す。実行の同時接続数、Budget、CPU/IO/時間、Fixture隔離、外部Effect、非公開データ・ログの開示は別制御。Tool呼出しの`allow`宣言だけでOS/Networkの実効遮断とみなさない。探索が打ち切られたら`truncated`・残件不明を報告する。
- 提案Caseを現行モデルの不変条件へ組み込む前に、対象ID・Rule/Role/Operator/Plugin版・Source span・Test AuthoringのRoleを固定。Unicode表示名、ローカル識別子、GUIラベルをCanonical IDや実CLI/HTTP公開入口へ推測で変換しない（Draft #2745）。

### 9.4 追加の反証Fixture（AI-13〜20、各々未実行）

| ID | 一入力だけ変える対照 | 不正な成功を拒否する期待 |
| --- | --- | --- |
| AI-13 | 既存Scenarioに第二Case候補を追加するが、Scenario元JSONは変更なし | 外部manifestの候補が既存v0に組み込まれていないことを`pending`と示し、Case追加成功を実行済みとしない |
| AI-14 | `Value.kind=null`、もしくは欠損/explicit-nullを同値扱い | 現行`Value.Decode`の未対応を正しく拒否し、missingを0/空文字へ補完しない |
| AI-15 | AIが既存`expectValidations`をActualに合わせて書換え | Expected出典とBase差分で自己追認を検出。run PASSだけでOracle独立性を主張しない |
| AI-16 | Plugin Lockは同一、候補Caseの実行Sourceや実Toolだけが変更 | 既存pinが固定する範囲と#2595のactual producer証拠を分離。digest一致≠同一生成主体 |
| AI-17 | Agent A/Bが同じStep IDまたはCase IDを追加 | 安定ID衝突を検出。後勝ちで上書きしない。候補の由来を保持 |
| AI-18 | Agentのsource可視範囲外のSecret/holdoutをFixture生成に流用 | 候補を隔離し不許可Read/egressを実権限境界で拒否。prompt制約のみでPASSにしない |
| AI-19 | 分母の探索上限に到達し既存ObligationがすべてPASS | `known covered`と`unknown/unexplored`を分け、完全な100%を宣言しない |
| AI-20 | UI DOM/画像一致・CLI/WebともResult一致だが、同一Generatorの誤期待値 | Target同等性は条件付き成立しても独立Oracleの正しさはFAIL/UNVERIFIED。画像を意味Expectedへ昇格しない |

### 9.5 現在の採否とOwner受渡し

- **採る**：既存Scenario v0 + CLIの実入口をP0のConsumerとして再利用、AIによるCase差分/隔離、独立ExpectedとCoverage分母の分離、ファイル+JSONによるHeadless-first検証、Agent単位のSource/Tool/権限/予算の追跡。
- **保留**：正式のCase Pack schema、Markdown表Parser/Role適用、`null`等の追加値型、汎用Property/Fuzz/Mutation Engine、Web/GUI emitter、Wails操作API、Coverage Store/Benchmark Runner、外部Agent用MCP。最小実Consumerで必要性を観測してから選定。
- **引き渡す**：#2621 Authoring、#1131 Fixture/Instance、#2514 Oracle/Mutation、#2493 Task Lane、#2515 OS/Egress、#2595 実Source束縛、#2600 選択的再資格化、#2391 UI外部制御、#1163 Actual Evidence、#1174 Provider Effect、#1180 CLI/Web Scenario parity。対象不在・機能未接続・実行未観測は各Ownerの未検証を維持。

**この追補はdocs-onlyであり、既存6 Stepのsourceを静的に確認した結果と、これから実施する試験を区別する。** 実Go/CLIの新規Attempt、Agent生成Case、Web、GUI、独立Oracleの新しい結合試験、外部Host/DBの実効果、Owner受理は今回NOT RUN。


## 10. AI Caseの発見・入場・判別力を実証する最小Consumer（2026-11-04）

**位置づけ：P0実装計画の精密化／追加試験 NOT RUN。** 同日に他レーンで開始した[Draft #2813](https://github.com/hat47x/tei-platform/pull/2813)の**test-only実コード**を重複実装せず、[#1180](https://github.com/hat47x/tei-platform/issues/1180)の保存Scenario v0と結ぶ最小Consumerの受入条件を定める。ここで「テストを生成できた」と「不具合を判別できた」と「現在の業務/Target適合を証明した」を分ける。

### 10.1 現在の2つの実在ConsumerとP0差分

| Consumer | 実在する最小足場（Source確認） | 現状のClaim上限 | 次に足すもの |
| --- | --- | --- | --- |
| **DSL試験候補** | #2813の \`reference/dsl-language-plugin/ai_pipeline_candidate_test.go\` と \`testdata/ai_pipeline/{baseline,extra_candidate}.json\`。\`*.json\`追加でケースを増やす。9+2候補。Go内に主要4件の固定Anchor | test-onlyの構文、Pin/型/Source byte span、効果付きOperator拒否。最新HEADのGo/CI合格は別Receipt | Oracle出典・Coverage対象・実装差分の検出力を計測。Caseを既存Anchorsで上書きしない |
| **Driver Scenario** | \`reference/driver-scenarios/order-entry-pattern-smoke.json\`（6 Step）と \`cmd/tei/scenario.go\` の\`verify/generate/run/check\`、Scenario/World/Plugin Lock/Fixture pin | Pattern/Validation/合成Browse/合成Observation/CLI plan。一般のBusiness Command、Web E2E、実DB、実Provider effectではない | AI追加Case候補を現行v0で表現できるか判定し、実行可能なCaseだけを隔離コピーのScenario/Fixtureに採用 |

**同一のTest Subject（SourceのSemantic IDとRevision）を参照しても、DSL候補の構文合格とDriverの業務期待合格は異なるClaim。** 両Consumerを一つのRunnerへ統合しない。追加Caseを「どの対象・Obligation・Expected根拠・実行経路に適用したか」という**薄い機械可読な照合表（Test Case Intake/Qualification view）**で関連付けることから始める。新しい必須production schema/Registryを先に導入せず、既存Issue/fixture/生成Manifestの派生Viewとして試す。

### 10.2 生成AI向けの「情報の入口」と「採点の出口」を非対称にする

- **AIが見てよい入力（許可された範囲）**：Targetの実在能力と未対応点、Source-backedのRule/型/制約/Relation/既存Case、Scenario v0の対応可能なStep/Fixture kind、テスト候補が狙うObligationと現在の未検査状態、Candidateの編集可能path、実行Budget。CLIの\`model catalog\`、\`validate -json\`、Scenario\`verify\`等で実際に得られる項目を利用し、足りない項目は「未取得」とする。現存しない\`tei test generate\`を前提にしない。
- **AIが書いてよいもの**：独立した新規test-only \`*.json\`候補/隔離Scenarioコピー、その対象/理由/seedを記録したレビュー用メタデータ。既存Accepted Rule/Canonical、旧Regression Expected、固定Anchor、実Secret、Baseline画像は変更不可。候補の意図と生成元が異なる場合は別IDにする。
- **採点側だけが保持すべきもの**：固定された独立Expected/独立Oracleの導出根拠、判別用Mutant、秘匿Holdout、適用条件と資格判断。公開合成Expectedを教育用に見せることは妨げないが、**独立採点に用いるHoldoutやMutant正答を生成Agentへ逆流させない**。
- **実証の出口**：\`candidateAdded\`、\`syntaxBound\`、\`executed\`、\`asserted\`、\`independentOracleQualified\`、\`applicableObligationCovered\`、\`targetEffectObserved\`を別列で記録。bool \`verified\`一つ、exit code 0、AIの自己申告、Mutation scoreだけで一括PASSを出さない。

**AIのCase生成の目的関数は総件数ではなく、**「不足Obligationの解消候補」「既知誤実装を独立に見分けた新規性」「再現率・期待値根拠」「CI/実行/レビュー費用」を並べて比較する。まず重複Caseを減らし、既存のAnchor/Negativeを破壊しない。合理的な探索予算上限を超えた場合は未探索数/不明を残す。未知Domainへの追加を禁止する固定上限ではない。

### 10.3 Coverage/Oracle適格性の判定表

| 起きたこと | 合格としてよいClaim | 合格としてはならないClaim |
| --- | --- | --- |
| AIが独立JSONを1件追加しGoテストがexit 0 | ケースが検出・実行され、限定された候補Assertionが一致した可能性 | 独立Oracle合格、業務意味、全分母Coverage、実Target |
| 既知Mutantを新Caseだけが拒否 | そのMutantに対する限定的な判別力（Mutant有効性・差分が検証済みの場合） | 未知の全故障を検出可能、全Domain適合 |
| CLI Scenarioの6/6 Stepが検証済み | 当該revision・Fixture・合成Stepの結果 | NULL/欠損/decimal・全業務Rule・Web/Auth/Writeの網羅 |
| 独立Expectedが未取得 | 観測／探索（qualifiedでない） | Correctness PASS |
| 必要なDataの読取りが\`partial/unavailable/permission-denied\` | 取得済みScope内だけの評価、または判定保留 | 0件/未取得を「対象不存在・全件無違反」とする判定 |
| Source bytes、Role/Operator/Oracle/Grant/Target版が変更 | 変更箇所の診断と再資格化が必要 | 旧Resultの無条件再利用 |
| 画像/DOMが一致、業務Effectの独立Readなし | 表示面の限定回帰結果 | Write/Commit/業務完了の保証 |

Source移動・A→B→A・外部Importの途中欠落・0件の否定依存・Datasetの異時点比較は、**Source fingerprint／Meaning revision／Model/Artifact revision／実Provider Snapshot／Oracle provenance／現Grant**のどれが変化したかを区別する。Selectorが生成したCoverage列だけで不存在や権限の根拠を作らない（#2582、#2600、#2515、#2514）。

### 10.4 最小の2系統実証と反証（新P0のDefinition of Done）

**Track A：DSL Candidate Intake（#2813 Owner、#2811はConsumer契約）**
1. #2813の実HEAD/Go版/実行結果を確認し、11候補の\`*.json\`探索・Case ID重複拒否・固定Anchor維持を実測する。PR説明だけでCI成功としない。
2. 生成AIまたは別生成器が、専用新規JSONへ**正常と拒否を各1件**追加する。共通Goコードを変更せず本当にテスト対象へ入ることを、subtest/出力で確認する。
3. 誤った\`expected\`への書換え、同一EvaluatorからのWantコピー、既知誤実装、\`expected\`欠落・重複JSON object keyを**別々の負例**とする。Goの標準JSON decodeだけではキー重複が黙って後勝ちになる場合があるため、現実装の実際の拒否能力を対照測定し、不足なら#2813 Ownerへ修正を委ねる。

**Track B：Driver Scenario Candidate Intake（#1180 Owner）**
1. 現行v0の6 Step/Fixture/CLI planを基準線とし、生成AIが**別のScenario+Fixture**に1件ずつ正常/Validation拒否候補を追加。元ScenarioのExpectedを編集しない。
2. \`verify→generate→run→check\`を実行し、Scenario/World/Plugin Lock/Fixture pinとevidenceNatureを別保存。現行v0に表現できないNULL・missing・decimal/State/Writeは\`unsupported\`として列挙し、成功に算入しない。
3. TargetとなるSemantic Subject/Operationに同じ名前でも別Instance/Scope/Revisionの候補を与え、誤束縛・欠落Step・旧Planの不正再利用を拒否できることを確認。

**共通採点**：上記の結果を同じCase/Subject IDと対象Source/Fixture revisionで照合するが、DSL試験をDriver業務試験の代替にしない。事前に列挙したObligation分母、追加された新Obligation、未評価/不明を分け、少なくとも**「テスト件数は増えたが独立Oracleの検出力は増えない」負例**と**「少数ケースで既知Mutantを新規検出」正例**を1つずつ採る。実Go/CLIのsource＋run receiptが出るまでDoneとしない。

### 10.5 並列化と後続へのゲート

- **Phase 0（いま）**：A/B各Ownerは独立branchでCase候補を追加可能。外部Agentは既存AGENTS/Task Packetのbase SHA・許可path/Tool・budget/egressを厳守。独立ReviewerはSourceとExpectedの共有由来・試験範囲を監査。#2514の独立Oracle・#2493の並列作業規則を参照。#2811は**仕様と証拠の接合**だけを所有する。
- **Phase 1**：Track A/Bのactual結果から標準データ表現（Test as Data）と、read-only機械可読gap/receiptへの必要最小差分を抽出。Schemaの名称、MCP/Agent Host、Coverage DBは現時点で固定しない。Markdownテスト表は#2621のRole/Source fidelityとの接合検証を経て採用判断。
- **Phase 2**：共通Driver Scenario v0で表せる対象の増加を確認後に、#1180のCLI/Web emitter・#2391のGUI/Instrumentation・#1163の一改訂Evidenceへ進む。テスト用UI支援をProduction成果物へ残さない。
- **Phase 3**：Source/Plugin/Oracle/Policy/Fixture/Targetが変わった時の再資格化と、Full vs Selectiveの誤再利用・実待ち時間を#2600/#2018へ引き渡す。EKIをCI/テスト実行の必須Hostとせず、分散管理や期限管理が必要な場合だけ既存委任契約で評価。

**判定状況**：#2813のtest-only 11候補と#1180の実CLI6 StepはSourceを静的に確認。**今回の新增分テスト・独立Oracle判別・Coverage計測・最新exact-head CI、Web/GUI/実Providerは未実行/未確認。** 正式な新schema/テストEngine/Oracle/Host、Ownerの意味受理を宣言しない。
