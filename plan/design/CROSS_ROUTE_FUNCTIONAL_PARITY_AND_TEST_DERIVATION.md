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
- §1〜7のAccepted Canonical（業務意味の正本）・保存済みDriver Scenario（シナリオテストの正本）・CLI/Web対称生成と安全境界を維持。本追補は新しい第二Driver/Runner/判定権限/Registryを設けない。
- 責務接合：#2621（表/Markdown Authoring）、#1131（Fixture/Dataset/Snapshot）、#2514（独立Expected/Mutation判別）、#1180（Scenario→CLI/Web生成）、#988/#2620（DSL/Operator/Role）、#1163（Evidence）、#2391（UI外部制御/本番物理除去）、#1174（Effect/unknown）、#2506/#2593（Context/データ粒度）、#1119（資格Gate）。

### 8.1 最優先：AIが操作できるテスト作成ループ

モデル・テスト資産をAIが観察し、追加したCase/Scenarioを検証して小さな差分を反復できることを**CLI・JSON・通常のファイル編集**で先に実証する。生成物専用の難解なコード生成APIやGUIを先行要件にしない。

1. **inspect / discover**：Accepted modelのFunction/Rule/Operation/型/Constraint/Relation/State/Operator Contract、Source位置、既存Scenario/Case、Fixtureの利用資格、未検証Coverage Obligation、対象Target/Providerの実在能力と未解決を機械可読に列挙。結果のScope・版・取得完全性・秘匿境界を明記。実在しないOperationや権限をAIが推測しない。
2. **author / extend**：AIは人が読めるMarkdown表や通常の構造化ファイルにCaseと必要ならScenario候補を追加する。決定的ID、元Semantic ID、入力partition/生成seed、preconditions、expectedまたは探索目的、候補のprovenanceを付ける。例示表と実行テストはRoleで識別し、既存Accepted Scenarioを自動改変しない。専門家が表を読めることを最優先し、Go固有記法を日常Authoringへ漏らさない。
3. **verify / explain**：構文・参照/型・Scope/粒度・null/missing/unavailable/redacted・Fixture・Operator lock・Target binding・副作用可否・Expectedの出典独立性・重複Case/到達不能・Coverage対応を**副作用なし**で検査。失敗理由はSource位置と修復可能候補を返す。
4. **run isolated**：既存Driver/Runtime/Target Harnessを再利用し、合成/使い捨てデータ・明示Budget/seed・所要資源上限の下で実行する。AIは実装を自由に変更・試行できるが、実Grant・機密読取・外部送信・Commit再試行の許可は得ない。exploratory runは観測のみでPASS主張をしない。
5. **compare / shrink / report**：Actual vs独立Expected、既知誤実装Mutant、Run間差分、データSnapshot、Rule/Branch/Pattern/Scenario coverage、失敗Traceを同一Case/Scenario/Run IDと版に結ぶ。反例の縮小・再現seed・重複検出はAI支援可能とし、最小反例に変形しても元の失敗が保持されるか実再実行する。
6. **review / adopt**：有用な探索ケースを回帰ケースへ昇格するときはExpectedの独立根拠・対象範囲・Owner受理・必要な権限を区別。AI生成・複数AIの同意・既存評価器が出したActualの転用だけではExpectedを正式検証基準にしない。

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
- 「CLIとWebが一致」と「両方が独立Expectedに正しい」は別検収。Fixtureを隔離して両経路を実行し、構造化Business Outcome、NULL/欠損、Key、Multiplicity、順序、金額精度、Data state delta、実Effect、独立Readを比較。権限/Policy Scope/時点/部分取得/Commit unknownが異なるときは`incomparable/unknown`。共通誤実装の自己追認には#2514の独立検証基準/既知Mutantを利用。
- GUIには、操作E2E、DOM/ARIA構造、視覚Baseline/画像差分、実データの独立Observationを**別Assertion列**として接続する。初期Web候補はPlaywright等を比較するがCoreに固定しない。日時/フォント/viewport/renderer/locale/animation/seedの固定、スクリーンショットの機密遮蔽、baselineのReview/Revisionと自動更新禁止を要求。画像一致≠業務成功、データ同値≠操作可能性。
- Wails/WebView/Native/Canvasは実Host固有Evidence。テスト用UI instrumentation/外部制御は#2391に従って本番Artifactから物理的に除去できるか検証し、通常のアクセシビリティを損なわない。Test-build成功をProduction安全性へ昇格しない。

### 8.4 AI-first垂直実証と段階順序

| Priority | 実証する最小成果 | 完了条件（予定、すべて未実行） |
| --- | --- | --- |
| **P0-A: AI authoring scaffold** | 同一モデルのread-only inventory/coverage gapをJSONで取得し、AIが独立したCaseファイルを追加・構文/意味検証し、失敗理由を修正できる | 既存テストを編集せず新規正常・拒否・境界Caseを加え、変更差分/Source位置/対象Obligationを再現。未解決・権限不足は明示 |
| **P0-B: 独立Expectedと判別力** | 生成Candidate、独立検証基準、観測Onlyケースを分離し、既知Mutantを試す | 誤Expectedの自己追認で偽PASSを許さず、無Expected探索をQUALIFIED PASSにしない。seed固定で反例を再現 |
| **P1: Scenario & coverage** | 永続Scenario 1件と追加Case群を既存Driverに接続し、typed CLI plan/実行/coverage reportを取得 | actual Source/Artifact/Fixture/Run版とCoverage分母が結ばれ、欠落Step/旧版を拒否 |
| **P2: Cross-target** | 同一ScenarioからCLI/Web E2Eを独立生成・隔離実行し独立Expected/Observed state照合 | CLI/Web両経路の漏れを検出、片方だけPASSなら全経路PASSにしない |
| **P3: Visual/Desktop** | Web DOM/ARIA・Screenshot回帰の差分、Wails/Native固有Assertion、Production instrumentation除去 | ビジネス差と視覚差の双方のMutantを別々に検出し、実配布Buildの除去を証拠で示す |
| **P4: 変更影響と選択的再実行** | Source/Rule/Plugin/検証基準/Fixture/Target変更でObligationと旧Evidenceを差分 | stale/unknownを明示。選択的再実行と完全ベースラインの見逃し率・コストを比較 |

最初のDomainは公開合成`order_entry`のread-only Query/Validationから、(1)正常、(2)NULLとmissing、(3)重複/部分集合、(4)丸め境界、(5)拒否/評価不能を候補化する。別途非業務Domain（状態機械/ゲーム等）1件で「業務専用のCoreパターン表」が不要なことを確認。Source/Meaningの変更なしに新たなCase・Profileを追加できることを重視する。実DB/書込み/GUIはP0合格の前提にしない。

反証Fixture：`AI-01` 既存期待値をActualからコピーした偽PASS、`AI-02` 新しいNULL partitionがcoverage分母から黙って消える、`AI-03` 旧Scenarioの片Target生成物だけ残留、`AI-04` Target未対応Stepがskipされ空PASS、`AI-05` Web表示一致だが実保存なし、`AI-06` CLI/Webとも同じ誤計算、`AI-07` 画像baseline自動更新で劣化を隠蔽、`AI-08` secret/権限を持つ本番Providerへ探索実行、`AI-09` 部分Readを全件確認と誤認、`AI-10` source/検証基準/Plugin版変更後も旧Evidenceを再利用、`AI-11` exploratory結果をOwner受理済み意味へ昇格、`AI-12` budget上限で探索停止した未検査をPASS扱い。

**完了報告の境界**：本節の追補は計画だけ。実CLIのinspect/generate/run/coverage、永続Scenario Compiler、GUI/Playwright/Wails、AI Agentの実行、安全なOS隔離、本番適格性、Owner意味受理は未証明。実装後はexact HEAD、実Build/Runner、fixture bytes、seed、独立検証基準、実行Trace、Coverage分母と未知範囲、clean repository-health結果を記録する。


## 9. 既存CLI Referenceに接続するAIテスト足場：実装可能なP0入口（2026-11-04）

**区分：現物Sourceの静的確認と設計計画。今回の追加試験は NOT RUN。** §8の「新CLIが未実装」という一律表現を訂正し、実装済みの範囲を正確に起点へ据える。他チャットのDraft #2745（DSL→同一Meaning/Scenario）、#2619（Test as Data）、#2514（TAO独立検証基準）、#2493（多数Agent Lane）、#2595（実Source入力・Tool帰属）、#2600（Artifact再利用とEvidence再資格化）、#2391（GUI/制御面本番除去）と照合した。変更は本Issue #1180の実装責務に限定し、別Ownerの意味/権限/Runnerを重複しない。

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

生成済みplanの`check -plan`は対象のScenario/Workspace同一性と手修正を検証するが、Plan生成自体を実行証拠にしない。`run`のpinなし直接起動は現状`executionMode=exploratory-unpinned`と返す。`verified=true`でも対象はSourceに書かれた限定合成Stepであり、独立検証基準・実Host/Provider/外部Effect全体の資格ではない。

### 9.2 最初の縦断テストを5枚の検収票に分ける

**P0-0 現物Baseline（変更前）**：clean checkout、正確なcommit/Go/toolchain、`cmd/tei/scenario.go`と`internal/scenariotest/scenario.go`のblob、6 Step Scenario/fixture bytesを固定。既存Focused test（`go test ./cmd/tei ./internal/scenariotest`）とCLIの`verify/generate/run/check`を**実際に**試し、成功・失敗・未実行をReceiptへ記録。新しいTest Engineを先に導入しない。

**P0-1 AI Case拡張（元の期待値を改変しない）**：生成AIには現存する検証済みread-only inventory、使用可能なOperation列、source locations、制約、既存Expected、fixture種別、未知/unsupportedを読ませ、**隔離ディレクトリ／別branch**にのみ新しいCase/Scenario/fixture候補を作成させる。最初は既存v0の許容表現に限り、（a）追加した正常Validation、（b）拒否されるValidation、（c）誤Expectedの否定対照を作る。各候補へ`candidateCaseID, sourceRef, intendedSemanticID, parentScenarioRevision, fixtureDigest, oracleOrigin, authoringActor, budget/seed`を**別のmanifest/レビュー表に保持**する。既存`Scenario` schemaへ未対応フィールドを無理に追加せず、Source Parserが拒否する場所にmetadataを注入しない。候補を正式Scenario/Canonicalへ自動受理しない。

**P0-2 静的検証・修復ループ**：現在の`verify`がチェックする範囲と、作成したCase metadata/coverage applicability/oracle独立性の未検査を並記。二段目のread-only checkerはまず小さな検証スクリプト/テストとして比較し、実Consumerの必要が確認された場合のみCLIへ接続。構文破損、unknown Step、duplicate Step ID、unsupported Value kind、missing fixture、Source ambiguity、Scope/Authority mismatch、期待値と実装の共有由来を型別に診断。AIによる修復後、同じ出典に対する再検証で失敗が解消されたことを確認する。

**P0-3 隔離実行・反例縮小**：既存CLIから実行できるCaseはpin付きplanで再実行し、`actual result` / `expected` / `evidenceNature` / `coverage obligation`を別列で保持。生成Agentは期待値が未確定の入力を探索してよいが、QUALIFIED PASSにしない。失敗入力を縮める場合もshrunk fixtureのdigest・seedと元の失敗型を保持し、**縮小後に再実行した同じ失敗**がなければ反例成立と呼ばない。

**P0-4 Coverage差分（分母の正当性）**：Caseを増やす前の**required/known/unknown** obligation inventoryを凍結し、実行後のcovered/uncovered/blocked/unsupportedを報告。Case数、Step数、Rule coverage、Data pattern coverage、実Provider scope coverageは別指標。新しいCaseが見つけた未知partitionは事前分母に対する後発の発見として記録し、新旧分母を同じ割合で比較しない。partial/read denied/zero rows/timeoutを全件未存在の証明にしない。

この5枚の完了を**一件のDemo成功ではなく、正負ペアと実Output digestに基づく受入**として実証。最初からWeb、SQL実Provider、Wails、Agent用MCP、新テスト言語、専用スケジューラを完成条件に含めない。

### 9.3 AI/Agent間の並列作業規則（#2493との接点）

- Task Packetに`base SHA, allowed source/read projection, writable test candidate paths, expected independent oracle, target Scenario/Obligation scope, budget, tool/OS principal, egress policy, output artifact paths, review owner`を持たせる。これは既存Issue/branch/manifestへ記録する運用規約であり、第二Task Authorityではない。
- 各Agentは独立したcandidate path/branch/worktreeへCaseを追加する。共有Scenario・独立Expected・Coverage分母・Baseline画像を**同時に無断更新しない**。同一Obligationに複数の独立反例を提案してよいが、正本採用はDiffと検証基準出典を比較して単一の受理経路へ統合。
- 同じAIモデルを複数回呼ぶ、相互一致を得る、同じEvaluatorでExpectedを作り直す、という行為は独立検証ではない。生成側と採点側のSource/Tool/Prompt/Corpusの共有由来を検査し、必要ならsealed holdoutを#2514/#2515の保護下で用いる。
- 生成AIの大量探索は**候補の発散**を許す。実行の同時接続数、Budget、CPU/IO/時間、Fixture隔離、外部Effect、非公開データ・ログの開示は別制御。Tool呼出しの`allow`宣言だけでOS/Networkの実効遮断とみなさない。探索が打ち切られたら`truncated`・残件不明を報告する。
- 提案Caseを現行モデルの不変条件へ組み込む前に、対象ID・Rule/Role/Operator/Plugin版・Source span・Test AuthoringのRoleを固定。Unicode表示名、ローカル識別子、GUIラベルをCanonical IDや実CLI/HTTP公開入口へ推測で変換しない（Draft #2745）。

### 9.4 追加の反証Fixture（AI-13〜20、各々未実行）

| ID | 一入力だけ変える対照 | 不正な成功を拒否する期待 |
| --- | --- | --- |
| AI-13 | 既存Scenarioに第二Case候補を追加するが、Scenario元JSONは変更なし | 外部manifestの候補が既存v0に組み込まれていないことを`pending`と示し、Case追加成功を実行済みとしない |
| AI-14 | `Value.kind=null`、もしくは欠損/explicit-nullを同値扱い | 現行`Value.Decode`の未対応を正しく拒否し、missingを0/空文字へ補完しない |
| AI-15 | AIが既存`expectValidations`をActualに合わせて書換え | Expected出典とBase差分で自己追認を検出。run PASSだけで検証基準独立性を主張しない |
| AI-16 | Plugin Lockは同一、候補Caseの実行Sourceや実Toolだけが変更 | 既存pinが固定する範囲と#2595のactual producer証拠を分離。digest一致≠同一生成主体 |
| AI-17 | Agent A/Bが同じStep IDまたはCase IDを追加 | 安定ID衝突を検出。後勝ちで上書きしない。候補の由来を保持 |
| AI-18 | Agentのsource可視範囲外のSecret/holdoutをFixture生成に流用 | 候補を隔離し不許可Read/egressを実権限境界で拒否。prompt制約のみでPASSにしない |
| AI-19 | 分母の探索上限に到達し既存ObligationがすべてPASS | `known covered`と`unknown/unexplored`を分け、完全な100%を宣言しない |
| AI-20 | UI DOM/画像一致・CLI/WebともResult一致だが、同一Generatorの誤期待値 | Target同等性は条件付き成立しても独立検証基準の正しさはFAIL/UNVERIFIED。画像を意味Expectedへ昇格しない |

### 9.5 現在の採否とOwner受渡し

- **採る**：既存Scenario v0 + CLIの実入口をP0のConsumerとして再利用、AIによるCase差分/隔離、独立ExpectedとCoverage分母の分離、ファイル+JSONによるHeadless-first検証、Agent単位のSource/Tool/権限/予算の追跡。
- **保留**：正式のCase Pack schema、Markdown表Parser/Role適用、`null`等の追加値型、汎用Property/Fuzz/Mutation Engine、Web/GUI emitter、Wails操作API、Coverage Store/Benchmark Runner、外部Agent用MCP。最小実Consumerで必要性を観測してから選定。
- **引き渡す**：#2621 Authoring、#1131 Fixture/Instance、#2514 検証基準/Mutation、#2493 Task Lane、#2515 OS/Egress、#2595 実Source束縛、#2600 選択的再資格化、#2391 UI外部制御、#1163 Actual Evidence、#1174 Provider Effect、#1180 CLI/Web Scenario parity。対象不在・機能未接続・実行未観測は各Ownerの未検証を維持。

**この追補はdocs-onlyであり、既存6 Stepのsourceを静的に確認した結果と、これから実施する試験を区別する。** 実Go/CLIの新規Attempt、Agent生成Case、Web、GUI、独立検証基準の新しい結合試験、外部Host/DBの実効果、Owner受理は今回NOT RUN。


## 10. AI Caseの発見・入場・判別力を実証する最小Consumer（2026-11-04）

**位置づけ：P0実装計画の精密化／追加試験 NOT RUN。** 同日に他レーンで開始した[Draft #2813](https://github.com/hat47x/tei-platform/pull/2813)の**test-only実コード**を重複実装せず、[#1180](https://github.com/hat47x/tei-platform/issues/1180)の保存Scenario v0と結ぶ最小Consumerの受入条件を定める。ここで「テストを生成できた」と「不具合を判別できた」と「現在の業務/Target適合を証明した」を分ける。

### 10.1 現在の2つの実在ConsumerとP0差分

| Consumer | 実在する最小足場（Source確認） | 現状のClaim上限 | 次に足すもの |
| --- | --- | --- | --- |
| **DSL試験候補** | #2813の `reference/dsl-language-plugin/ai_pipeline_candidate_test.go` と `testdata/ai_pipeline/{baseline,extra_candidate}.json`。`*.json`追加でケースを増やす。9+2候補。Go内に主要4件の固定Anchor | test-onlyの構文、Pin/型/Source byte span、効果付きOperator拒否。最新HEADのGo/CI合格は別Receipt | 検証基準出典・Coverage対象・実装差分の検出力を計測。Caseを既存Anchorsで上書きしない |
| **Driver Scenario** | `reference/driver-scenarios/order-entry-pattern-smoke.json`（6 Step）と `cmd/tei/scenario.go` の`verify/generate/run/check`、Scenario/World/Plugin Lock/Fixture pin | Pattern/Validation/合成Browse/合成Observation/CLI plan。一般のBusiness Command、Web E2E、実DB、実Provider effectではない | AI追加Case候補を現行v0で表現できるか判定し、実行可能なCaseだけを隔離コピーのScenario/Fixtureに採用 |

**同一のTest Subject（SourceのSemantic IDとRevision）を参照しても、DSL候補の構文合格とDriverの業務期待合格は異なるClaim。** 両Consumerを一つのRunnerへ統合しない。追加Caseを「どの対象・Obligation・Expected根拠・実行経路に適用したか」という**薄い機械可読な照合表（Test Case Intake/Qualification view）**で関連付けることから始める。新しい必須production schema/Registryを先に導入せず、既存Issue/fixture/生成Manifestの派生Viewとして試す。

### 10.2 生成AI向けの「情報の入口」と「採点の出口」を非対称にする

- **AIが見てよい入力（許可された範囲）**：Targetの実在能力と未対応点、Source-backedのRule/型/制約/Relation/既存Case、Scenario v0の対応可能なStep/Fixture kind、テスト候補が狙うObligationと現在の未検査状態、Candidateの編集可能path、実行Budget。CLIの`model catalog`、`validate -json`、Scenario`verify`等で実際に得られる項目を利用し、足りない項目は「未取得」とする。現存しない`tei test generate`を前提にしない。
- **AIが書いてよいもの**：独立した新規test-only `*.json`候補/隔離Scenarioコピー、その対象/理由/seedを記録したレビュー用メタデータ。既存Accepted Rule/Canonical、旧Regression Expected、固定Anchor、実Secret、Baseline画像は変更不可。候補の意図と生成元が異なる場合は別IDにする。
- **採点側だけが保持すべきもの**：固定された独立Expected/独立検証基準の導出根拠、判別用Mutant、秘匿Holdout、適用条件と資格判断。公開合成Expectedを教育用に見せることは妨げないが、**独立採点に用いるHoldoutやMutant正答を生成Agentへ逆流させない**。
- **実証の出口**：`candidateAdded`、`syntaxBound`、`executed`、`asserted`、`independentOracleQualified`、`applicableObligationCovered`、`targetEffectObserved`を別列で記録。bool `verified`一つ、exit code 0、AIの自己申告、Mutation scoreだけで一括PASSを出さない。

**AIのCase生成の目的関数は総件数ではなく、**「不足Obligationの解消候補」「既知誤実装を独立に見分けた新規性」「再現率・期待値根拠」「CI/実行/レビュー費用」を並べて比較する。まず重複Caseを減らし、既存のAnchor/Negativeを破壊しない。合理的な探索予算上限を超えた場合は未探索数/不明を残す。未知Domainへの追加を禁止する固定上限ではない。

### 10.3 Coverage/検証基準適格性の判定表

| 起きたこと | 合格としてよいClaim | 合格としてはならないClaim |
| --- | --- | --- |
| AIが独立JSONを1件追加しGoテストがexit 0 | ケースが検出・実行され、限定された候補Assertionが一致した可能性 | 独立検証基準合格、業務意味、全分母Coverage、実Target |
| 既知Mutantを新Caseだけが拒否 | そのMutantに対する限定的な判別力（Mutant有効性・差分が検証済みの場合） | 未知の全故障を検出可能、全Domain適合 |
| CLI Scenarioの6/6 Stepが検証済み | 当該revision・Fixture・合成Stepの結果 | NULL/欠損/decimal・全業務Rule・Web/Auth/Writeの網羅 |
| 独立Expectedが未取得 | 観測／探索（qualifiedでない） | Correctness PASS |
| 必要なDataの読取りが`partial/unavailable/permission-denied` | 取得済みScope内だけの評価、または判定保留 | 0件/未取得を「対象不存在・全件無違反」とする判定 |
| Source bytes、Role/Operator/検証基準/Grant/Target版が変更 | 変更箇所の診断と再資格化が必要 | 旧Resultの無条件再利用 |
| 画像/DOMが一致、業務Effectの独立Readなし | 表示面の限定回帰結果 | Write/Commit/業務完了の保証 |

Source移動・A→B→A・外部Importの途中欠落・0件の否定依存・Datasetの異時点比較は、**Source fingerprint／Meaning revision／Model/Artifact revision／実Provider Snapshot／独立期待値の出典／現Grant**のどれが変化したかを区別する。Selectorが生成したCoverage列だけで不存在や権限の根拠を作らない（#2582、#2600、#2515、#2514）。

### 10.4 最小の2系統実証と反証（新P0のDefinition of Done）

**Track A：DSL Candidate Intake（#2813 Owner、#2811はConsumer契約）**
1. #2813の実HEAD/Go版/実行結果を確認し、11候補の`*.json`探索・Case ID重複拒否・固定Anchor維持を実測する。PR説明だけでCI成功としない。
2. 生成AIまたは別生成器が、専用新規JSONへ**正常と拒否を各1件**追加する。共通Goコードを変更せず本当にテスト対象へ入ることを、subtest/出力で確認する。
3. 誤った`expected`への書換え、同一EvaluatorからのWantコピー、既知誤実装、`expected`欠落・重複JSON object keyを**別々の負例**とする。Goの標準JSON decodeだけではキー重複が黙って後勝ちになる場合があるため、現実装の実際の拒否能力を対照測定し、不足なら#2813 Ownerへ修正を委ねる。

**Track B：Driver Scenario Candidate Intake（#1180 Owner）**
1. 現行v0の6 Step/Fixture/CLI planを基準線とし、生成AIが**別のScenario+Fixture**に1件ずつ正常/Validation拒否候補を追加。元ScenarioのExpectedを編集しない。
2. `verify→generate→run→check`を実行し、Scenario/World/Plugin Lock/Fixture pinとevidenceNatureを別保存。現行v0に表現できないNULL・missing・decimal/State/Writeは`unsupported`として列挙し、成功に算入しない。
3. TargetとなるSemantic Subject/Operationに同じ名前でも別Instance/Scope/Revisionの候補を与え、誤束縛・欠落Step・旧Planの不正再利用を拒否できることを確認。

**共通採点**：上記の結果を同じCase/Subject IDと対象Source/Fixture revisionで照合するが、DSL試験をDriver業務試験の代替にしない。事前に列挙したObligation分母、追加された新Obligation、未評価/不明を分け、少なくとも**「テスト件数は増えたが独立検証基準の検出力は増えない」負例**と**「少数ケースで既知Mutantを新規検出」正例**を1つずつ採る。実Go/CLIのsource＋run receiptが出るまでDoneとしない。

### 10.5 並列化と後続へのゲート

- **Phase 0（いま）**：A/B各Ownerは独立branchでCase候補を追加可能。外部Agentは既存AGENTS/Task Packetのbase SHA・許可path/Tool・budget/egressを厳守。独立ReviewerはSourceとExpectedの共有由来・試験範囲を監査。#2514の独立検証基準・#2493の並列作業規則を参照。#2811は**仕様と証拠の接合**だけを所有する。
- **Phase 1**：Track A/Bのactual結果から標準データ表現（Test as Data）と、read-only機械可読gap/receiptへの必要最小差分を抽出。Schemaの名称、MCP/Agent Host、Coverage DBは現時点で固定しない。Markdownテスト表は#2621のRole/Source fidelityとの接合検証を経て採用判断。
- **Phase 2**：共通Driver Scenario v0で表せる対象の増加を確認後に、#1180のCLI/Web emitter・#2391のGUI/Instrumentation・#1163の一改訂Evidenceへ進む。テスト用UI支援をProduction成果物へ残さない。
- **Phase 3**：Source/Plugin/検証基準/Policy/Fixture/Targetが変わった時の再資格化と、Full vs Selectiveの誤再利用・実待ち時間を#2600/#2018へ引き渡す。EKIをCI/テスト実行の必須Hostとせず、分散管理や期限管理が必要な場合だけ既存委任契約で評価。

**判定状況**：#2813のtest-only 11候補と#1180の実CLI6 StepはSourceを静的に確認。**今回の新增分テスト・独立検証基準判別・Coverage計測・最新exact-head CI、Web/GUI/実Providerは未実行/未確認。** 正式な新schema/テストEngine/検証基準/Host、Ownerの意味受理を宣言しない。


## 11. P0採用ゲート：生成Caseの期待値・契約・Coverageを独立に審査する（2026-11-04）

**状態：既存PR/Codeの追加Source確認に基づく計画・未実装／本節の新たな実試験NOT RUN。** 同日[DSL候補Draft #2813](https://github.com/hat47x/tei-platform/pull/2813) HEAD `2219fc4d4d28430839bcd1ffab615811e6ecf798` のSourceから、`ai_pipeline_candidate_test.go` が `scenariotest.RejectDuplicateJSONMembers` を再利用してJSON objectキー重複を拒否するようになったことを確認。先行§10の「重複キー拒否が未実装の可能性」は**このHEADでは再利用Sourceが存在する**と更新する。実際のGoテスト／CI完了を示すClaimではない。PR #2811自体は引き続きdocs-onlyで、別AgentのPR #2813の編集・mergeをしない。

### 11.1 次の実装で解くべき欠落は「Case生成」ではなく「採用判断」

#2813のJSON候補には、AIが同一ファイル内の`contracts`（Operatorのid/version/fingerprint/effect/inputType/outputType）と`cases[].expected`を**同時に提案可能**な構造がある。`aiPipelineAnchors`は固定4ケースのsource/stage/output/errorを保持するが、**新規追加Caseについて提案ContractとExpectedが同じ誤りに沿っている**ことを単独では検出できない。これは事前の危険分析であり、現在の実Goに偽PASSが起きたと観測したわけではない。

したがって、`candidate passed`から`qualified regression`へ自動昇格しない。優先する受入条件は次のとおり。

| Gate | 必要な入力と判定 | 不足した場合の表示 |
| --- | --- | --- |
| **G0 入場** | ファイル/Case/Step一意ID、Source byte/span、Role、型/Contract参照、fixture境界を正しく解釈できる。曖昧JSONキーを拒否 | `invalid-candidate` |
| **G1 適用** | 対象Semantic ID/Revision、Contract実Pin、Source/Scope/grain、Caseが対象に達する根拠、実Targetの対応可能Operationを照合 | `unresolved` / `unsupported` |
| **G2 独立Expected** | 受理済みの意味、候補実装から独立したReference/既知Mutant/外部定理・契約等に基づくExpected、独立性の限界と再資格化条件を記録。異なるAIという理由だけでは独立とみなさない | `exploratory-only` / `oracle-unqualified` |
| **G3 実行証拠** | Exact Source/Scenario/Fixture/Toolchain/Plugin/Artifact、実argv/exit/Receipt、Actual、Assertion・独立Readの実観測範囲を固定 | `not-run` / `stale` / `unknown` |
| **G4 Coverage資格** | 実行前に固定した適用Obligation集合のどれを独立Assertionが判別したか、実測到達範囲・未観測・探索打切りを列記 | `covered-scope-only` / `incomplete` |

**G0～G4は連続した一つの合格率ではなく異なるClaimの資格**。G2は新たな業務意味を創作しない限り毎回Owner手作業を必須化しない。Accepted Meaningと独立したReference、検証済みの機械判定が使えるケースは自動資格化を目指せる。一方、未受理の業務Rule、丸め方式、期間・金額精度、認可意味を推測した場合はOwnerの意味判断が必要。正式導入の可否は既存のAcceptance担当が判断し、この計画が新Authorityにならない。

### 11.2 最小の反証対：AIがContractとExpectedを同時に変えてしまう

**適用対象を限定した合成PoC（設計値、未実行）**：公開DSL候補の仮想`std.select`について、Accepted/独立Referenceが`rows/line → rows/selected`と定める条件があると仮定する（これを既存TEIのAccepted宣言とは呼ばない）。

- **正例 E1**：Candidate `std.select: rows/line → rows/selected`、Expected `rows/selected`、別根拠の検証基準も`rows/selected`、実Goが一致。G0～G4のうち実際に取得した層だけ合格。
- **偽PASS反例 E2**：生成Agentが`contracts.select.outputType`と`cases[].expected.output_type`を両方`rows/wrong`へ変更。候補の内部整合がPASSしても独立ReferenceではFAIL。**ContractとExpectedが同一起源の場合、内部PASSからG2へ昇格しない**。
- **Mutation反例 E3**：独立Expectedは`rows/selected`に固定したまま、実装/候補Contractの変換結果だけを`rows/wrong`に変更。G2のReferenceに対してFAILを検出できることを実Goで実証する。実行可能な変異方法・テスト対象の由来は#2514 Ownerと調整し、契約変更のみが「実装Mutant」とは限らない点を明記。
- **対象不一致 E4**：同じ名前`std.select`でもOperatorのversion/fingerprint/入力型が異なる。誤Pinに基づく「検出した/しなかった」を比較対象外へ分類。Candidateのラベルだけで実Contract identityを決めない。

追加の正負：既存固定Anchor削除、Case ID重複、JSONキー重複、見えないHoldoutのExpectedコピー、NULL/欠損/decimalの現行v0未対応、`evidenceNature`がsynthetic-onlyなのに実Provider Write PASSとするケース。各反例の実行結果を取得する前にPASSと書かない。

### 11.3 AIに渡す最小Work Packet（新しい業務Schemaではない）

| 項目 | 例／用途 | 注意点 |
| --- | --- | --- |
| Baseline | `main HEAD`、対象PR HEAD、対象Source/Scenario/Fixture/Plugin pin | 分岐中のPR結果を現main実績に読み替えない |
| Target | 1つのSemantic Subject/Operator/Driver StepとScope | Unicode表示名・相対pathだけで業務IDを推測しない |
| Allowed changes | Agent専用 `testdata/.../<new-id>.json` または隔離Scenario/Fixtureコピー | 既存Accepted/Anchors/検証基準/本番Provider/他Agent候補はread-only |
| Inputs | Scope付きモデル/操作inventory、事前Obligation、読取可能Fixture、生成Budgetとseed | `unknown`や未取得を「存在なし」にしない |
| Output | Candidate差分、diagnostic、実argv/Receipt、独立検証基準照合、Coverage差分、未検査理由 | log/secret/holdout/raw保護データを公開PRへ複製しない |
| Stop | Authority不足、効果不明、fingerprintずれ、適用対象曖昧、Budget終了 | 反復自動実行や期待値自動書換えで解消しない |

これにより、生成AIは**ケースの候補作成・改良を広く自律実行**できるが、業務Meaningの正本更新や外部Effect/秘密開示の資格は取得しない。実OS/Network隔離は#2515、分散/非同期作業や期限は既存EKI/Buri委任Ownerに委ね、P0の実行必須インフラとしない。

### 11.4 実装を増やす前に必要な採点と運用測定

#2018（開発生産性）と#2600（再利用/再資格化）へ、少なくとも以下の**分母/対象/費用を別々に**渡す。
- 候補投入数／構文入場数／実行された数／G2独立検証基準資格数／既存Obligationの新規充足数／知らなかったObligationの新規発見数。
- 同一BaselineのKnown Mutantについて、追加Case前後で新たに検出できるか。Mutantが判別可能でない、Target/Scope対象外、検証基準が共有起源のときは検出率の分母へ機械的に算入しない。
- ファイル1個追加までの作業回数、Go/CLI試行回数、CI時間、CPU/IO、実レビュー負担、見落とした故障、ケースの重複、トークン費用の**可観測範囲**。実Cost未測定ならunknown。
- 生成したCaseがSource/Role/検証基準/Operator/Target改版でいつstaleになったかを追跡。Source path移動・A→B→A・empty集合の依存/欠落、Artifact再利用とEvidence再資格化を混同しない。

**次の小さい着手単位**：#2813 Ownerが最新HEADのFocused Go+CIで既存11件＋JSON重複拒否を実測（実施済みかどうかは実Receipt照会で確定）。#2514と独立検証基準/Mutant正負対の採点根拠を合意。#1180側で元6 Stepを保持した隔離Candidate追加と既存CLI`verify/generate/run/check`実行を行い、実際の差分を取得。その後にだけTest Case Intake Viewや構造化Coverage出力の最小追加を判断する。

**未検証のまま維持するもの**：新Caseの独立Expected判別、各Gateの実装・測定、Web/GUI/Desktopでの同一Scenario実走、実DB/Host、副作用、Owner採用、正式Test Schema、CI完全通過。本節は文書契約・反証計画の更新である。

## 12. CIで確認した成熟成果と最小実証の停止線（2026-11-04）

**今回の差分は設計資料とMarkdown整形の修正のみ。** CIで検証済みのPR/HEADと、その結果から導けない主張を分離する。前版の#2811 GitHub Actions [#41745134335](https://github.com/hat47x/tei-platform/actions/runs/41745134335) は **failure**。Jobログは `TestMarkdownHasNoEscapedBackticks` が本設計文書304行目からのエスケープ済みバッククォートを拒否したことを明示。Go vet、gofmt、Workspace validate、外部連携限定検査は同Runで成功し、Go全体テストはMarkdown検査が理由で失敗。**本コミットはその全出現を通常のMarkdownインラインコード表記へ修復**する。旧Run失敗の解消は修正後の**exact-head CI結果**で判定し、この文書では成功を先取りしない。

### 12.1 他レーンから実際に再利用できる成果

| Source と実検証 | 成立が確認できた限定Claim | #2811の設計へ接続すること |
| --- | --- | --- |
| [DSL AI候補 Draft #2813](https://github.com/hat47x/tei-platform/pull/2813)、head `2219fc4d`、[Repository Health #41744813520](https://github.com/hat47x/tei-platform/actions/runs/41744813520) **completed/success** | このexact-headではJSON追加式test-only11候補と既存健全性検査が成功。既存Scenarioの重複JSON member拒否も再利用 | **P0-Aの最小実装Referenceとして受入試験へ使える**。独立検証基準やBusiness/CLI/Web CoverageまでPASSとはしない。PRはDraft/未統合 |
| [Go Source束縛 PR #2819](https://github.com/hat47x/tei-platform/pull/2819)、main統合 | 宣言された`manifest.Source`と実際に`go build`が消費した`main.go`が違っても、既存選択器が拒否しない**負例がテスト化**された。SourceDigestと実Built Artifactの寄与がずれる | **AIテストの独立Expected/対象ピンに実消費Input Witnessも必要**と判明。SHAが付いているだけでは正しい対象をテストした証拠にはならない。実装側の拒否修正（EB-04）未実施 |
| [外部Semantic Load Draft #2812](https://github.com/hat47x/tei-platform/pull/2812)、head `7a6071ed`、[Repository Health #41742064273](https://github.com/hat47x/tei-platform/actions/runs/41742064273) **completed/success** | selected Namespaceについて同一Provider Loadのparsed Source/Manifestをprivateに保持し、公開側はmetadataのみ。限定CI合格 | AIの`inspect`に**許可範囲内のmetadataだけ**を渡す。Raw/Full Input Closure/未選択Namespace/Source開示を自動許可しない。Draft/未統合 |

#2813の実Go/CI成功は、AIが自動で新しい未知テストを生成した実験、独立検証基準の品質、実Targetへの到達を証明するものではない。実際に何件をAIが独立生成し、どの不具合を新規に検出したかを次の計測対象にする。

### 12.2 P0の最小テスト実行Packetを一枚に収束させる

再計画を繰り返す代わりに、**2 Consumer × 各1正例/1負例 + 独立検証基準対照**だけを一回の再現性あるAttemptとして実施する。

1. **固定入力**：mainと#2813のexact HEAD、Go toolchain、公開合成モデルとScenarioのSource byte、accepted Rule/Operator Contract/Plugin lock、固定検証基準・Known Mutant・許容Scopeを先に記録。異PRの結果を同一HEADの統合資格として扱わない。
2. **Agentへの入力**：公開可能なモデル/Operator/Case一覧、Target/Gateの能力、編集可能な**新しいJSONパスだけ**、未検証Coverage候補、上限と停止条件。検証基準 holdout、Owner受理のない意味、非公開Source/Provider生データは渡さない。入力が部分取得なら`unknown`もそのまま渡す。
3. **Track A DSL**：AIは#2813形式で正常・拒否各1件を新規ファイルへ追加。Go Source本体や固定Anchorを変更せず、対象Go testのsubtestが増えた事実、JSON重複・ID衝突・誤Operator Pinの拒否を確認。
4. **Track B Driver**：AIは現行`tei.driver-scenario/v0`で表現可能なValidation正例/拒否候補を、元6 Stepから隔離したScenario+Fixtureへ追加。`scenario verify/generate/run/check`の実argv/exit/結果/Source+World+lock+Fixture pinを取得。表現不能なNULL/missing/decimal/実Write/Webは`unsupported`のまま記録。
5. **採点**：一つのCaseについて(A)期待値を実Evaluatorから丸写しする対照と(B)別Sourceの独立Expectedを使う対照を区別。Known Mutantが見逃されたときは`inconclusive/undetected`とし、同じCandidate ContractをWant生成にも使っただけの内部PASSを正しさへ格上げしない。
6. **Coverageと費用**：生成前にObligation候補集合を固定し、追加Case/accepted/実行/独立Assert/新規検出/未探索を個別に集計。実Go試行回数、run時間、レビュー件数と失敗を実測。少ないケースで検出力が増したかを#2018へ返す。

**Exit/Stop判定**：独立Expectedを固定できない／現在のSourceやBuilder実Inputを照合できない／対象やScopeが曖昧／secret・実副作用の許諾がない／fixtureが部分取得で必要前提を満たさない／打切りに達した場合は、探索結果を残して`unqualified/unknown/blocked`とする。AIの生成の自由は維持し、テストの正式適格性・実書込み権限を推測で補わない。

### 12.3 次段へ進む条件を小さくする

- **今着手するもの**：上記Packetを既存#2813/既存#1180の実テストで測定し、既存Case/Scenarioを改変せずに候補を増やせたか、独立Expectedの判別力があるか、欠落Coverageに作用したかを実観測する。
- **採点可能性が明確になってから**：同一ScenarioからCLI/Web E2Eの再生成へ接続。GUI screenshot/ARIA/実業務状態は別Target Assertionであり、CLIの成功は他Targetの資格にならない。
- **今は作らない**：第二のTest Runtime/Registry、全ドメイン専用Schema、Agent MCP必須Host、GUI第一、秘匿検証基準の公開、全Source閉包が未証明な段階での完全なIncremental Test Skip。

Ownerは#1180（Scenario）、#2813/#988（DSL Candidate）、#2514（検証基準）、#2515（開示/OS）、#2595（実Source/Builder）、#2600（依存再資格）、#1163（Evidence）、#2391（GUI）、#2018（費用）へ分散したまま。**本節は新しい一元管理Owner・Host・意味正本を設立しない**。

## 13. AIテスト候補から採用済み回帰資産への公開・受領境界（2026-11-04）

**区分：横断設計追補／本節の実験はNOT RUN。** [横断計画§41–42（Draft #2667）](https://github.com/hat47x/tei-platform/blob/docs/everything-data-real-evidence-cut-20261103/plan/workstreams/EVERYTHING_DATA_CROSSLANE_HANDOFF_2026-11.md)の成果を再利用する。最優先は依然としてAIがJSONまたはScenario v0にCaseを隔離追加できるP0足場。その上で、**テスト候補が限定的にPASSしたこと**と**検証済み回帰Suite一式が公開され、Consumerが同じ版を受領できること**を別資格とする。新Runner/Publisher/Registry/意味正本は設けない。

### 13.1 採用後の資格はG0〜G4の外に分離する

| 状態 | 独立に確認すること | 保留すること |
| --- | --- | --- |
| Candidate | Agent別隔離PathへCase/Fixtureを生成。G0構文・G1対象/Scopeの妥当性 | 独立Expected、業務意味、実Target |
| Qualified Regression | G2独立検証基準、G3実Run、G4事前Obligation・新規判別力を評価 | 世代一式公開、外部Consumer受領 |
| Published Suite | Accepted Case/Fixture/Expected/Obligation/Target bindingの**必要Member集合・実byte/manifest・公開世代**を照合（#2604） | Consumerが取得したこと、実Grant |
| Consumed Suite | Consumerが指定Manifestにあるmemberを読み、各byte/版を照合。余剰memberを勝手に取り込まない（#2516/#1163） | Target実行、現在の権限と効果 |
| Executed Target | 実入力・Target artifact・runner receiptと独立Observationを束縛 | 別Target/別時点/業務意味の自動受理 |

**この追加資格はP0の候補Case追加・局所テスト実行の必須前提にはしない。** CIや他Agent/Web/SHOに採用済み回帰一式を渡す段階でだけ発行・受領を厳格化する。G0〜G4をもう一つの合格率へ足さない。

### 13.2 最新の実Go反例と対応する最小負例

[Draft #2817](https://github.com/hat47x/tei-platform/pull/2817) は #2604 AP-01〜03を局所Goで実証：①R2書込み途中で新memberと旧R1 manifestが混在、②R2で使わない旧memberの物理残留、③名前衝突が検知される前に旧memberが置換される。これは既存Artifact Writerの現状であり、AI Case Suiteの公開が既に同実装を使っているという実装事実ではない。

| Test | AI回帰資産の反証入力 | 受入期待 |
| --- | --- | --- |
| AT-PUB-01 | R2のCaseだけ新しく、Expectedは旧R1 | 混合版は拒否。旧Expectedを最新Caseの独立検証基準に流用しない |
| AT-PUB-02 | R2 manifestにない旧R1 Caseがディレクトリに残留 | Consumerは旧memberをR2のCaseとして実行しない |
| AT-PUB-03 | 2つのCase/Subjectが同じ物理出力名へ衝突 | 旧公開一式を途中破壊したのに成功としない。Case IDと物理ファイル名を区別 |
| AT-PUB-04 | Manifest completeだが検証基準由来不明、CheckSet Findingあり | 物理的な一式性と回帰意味適格性を分ける |
| AT-PUB-05 | 同一の短縮Hash ID／別Scopeまたは別Scheme | 文字列一致だけで意味ID・検証基準・実行権限を結合しない（#2814/#2815） |
| AT-PUB-06 | 同一Scenarioから生成したCLI/Webの一方だけ古いMemberを消費 | 各Targetの実Consumer Receipt/Pinを別検証し、CLI PASSをWebへ流用しない |
| AT-PUB-07 | 外部AI公開IssueにSecret/holdout/Import Rawが証拠として混入 | #2515/#2516の公開可能Projectionのみ出し、原文の保存を開示許可と混同しない |
| AT-PUB-08 | EKI Job完了だが成果物本文未取得／Provider Commit不明 | Job完了・成果物受領・業務Effectを別資格で保留。自動再実行で不明Commitを解決しない |

### 13.3 P0の次に行う1件だけのConsumer実証

§12の最小Packet（DSL JSON正常/拒否とDriver Scenario正常/拒否、固定検証基準/Mutant）が実測できてから、**採用可能なCaseを1件だけ**隔離世代として記述する。既存#2604のManifest/Member/Reader手法を利用できる範囲で、Producerが書いた全必須Case/Fixture/Expected bytesを示す。別のread-only Consumerが実際に取得したbytesをManifestのpinと照合する。R1→R2途中失敗、旧Case残存を最低1つずつ注入し、**Publisherが成功でもConsumerが取得していない**、または**実行は成功でも検証基準が共有由来**という二つの不正昇格を拒否する。

固定ManifestからCLI/Web各派生へ進むのは、その版とConsumerの実入力を照合できてから。特定のAtomic Rename・OS/FS保証は未採用であり、物理公開方法は#2604が判断する。AgentごとにCandidate独立編集を許し、採用済みExpected・固定Anchor・Holdout/世代公開先を勝手に編集させない。

### 13.4 横断計画との整合・責務

- §41の順序は**AI-first P0足場を先に整備し、その最初の製品横断Evidence対象がV-READ**という意味。P0足場を実ProviderやSHO/GUI完成まで待たせない。
- #2604＝Artifact一式公開、#2516＝実Consumer/Disclosure、#1163＝一改訂Evidence、#1180＝ScenarioのCLI/Web派生、#2514＝独立検証基準、#2391＝GUI外部制御とProduction物理除去。#2811は**受渡しと採用条件**だけを扱う。
- #2817の試験はWSL Go/既存Artifact Writerの局所負例で、AI Suite/Windows/外部SHO/実ConsumerのE2E証明ではない。#2815のHashId PoCは現行Case ID・Digest・署名の正式採用ではない。
- #2820/#2825の文書整合監査に従い、歴史的なDraft/未実行の記載を現在のmerge/CIの真偽に読み替えない。正しさと本番資格は一つのStatusへ集約しない。

**本節はdocs-only。** AT-PUB-01〜08、AI Caseの採用済み一式公開、Reader受領、Web/GUI/Provider/Host、Owner意味受理は本変更ではNOT RUN。新Publisher/Schema/Authorityは追加しない。

## 14. AIテスト入場前のSource完全性とCLI契約：ゼロ件・未読込を成功にしない（2026-11-04）

**区分：既存Issueの実CLI再現結果を受入条件へ組み込んだdocs-only変更。今回のCLI/Go試験はNOT RUN。** [横断計画§43（Draft #2667）](https://github.com/hat47x/tei-platform/blob/docs/everything-data-real-evidence-cut-20261103/plan/workstreams/EVERYTHING_DATA_CROSSLANE_HANDOFF_2026-11.md)を所有するレーンと責務を重ねず、#2811の「生成AIがどのCaseを作ってよいかを決める入場前提」として受け取る。

### 14.1 新たに裏付けられた三つの入場リスク

| 入場リスク | 実CLIの限定観測（他Issueの報告） | #2811での帰結と現在のClaim制限 |
| --- | --- | --- |
| **利用者宣言0件と成功** | [#2822](https://github.com/hat47x/tei-platform/issues/2822)：manifestだけのWorkspaceで `tei validate -json` は `ok=true`、`model catalog` は標準`std`の13 entries、利用者宣言は0件。空のmodelRevisionとcatalogが並存 | `validate.ok`・`catalog.entries.length`から対象業務Rule/Caseの存在を推論しない。空を意図したWorkspaceは**そのScopeに限り合法**でも、必要宣言が存在したことの証拠ではない |
| **発見と読み込みの差** | [#2823](https://github.com/hat47x/tei-platform/issues/2823)：manifestの未知/存在しないSource rootの診断が直接返らない、`workspace-files`で発見した`.yaml`/`.TEI`/`Application.tei`等が実Loaderで読み飛ばされ得る | `discovered`≠`loaded`。Required Sourceが実Loadに入ったというWitnessがない場合、Case候補の局所Parseは行えても、Workspace完全性・全Rule Coverageを`qualified`としない |
| **CLI契約版の未知** | [#2824](https://github.com/hat47x/tei-platform/issues/2824)：`tei commands`は19項目、文書内13と不一致。`schemaVersion`/契約外`world`がdiscoveryでは区別できない | コマンドの存在≠安定JSON ABI。実CLIの出力modeごとの契約を確認できなければ`contract-unknown`とし、発見された名称をそのままAIの実行プランへ採用しない |

**修正方法をこのPRでは決定しない。** `validate`へ件数欄追加、warning化、`workspace-files`の読込状態欄、discoveryへの版/契約外表示の選択・互換性判断は#2822/#2823/#2824のOwnerが行う。#2811は結果を正しく解釈し、機械可読な不足を`unknown/unloaded/unsupported`で残すConsumer。

### 14.2 G0の手前に置くReadiness前提（第二Gate/新APIではない）

`Declared → Discovered → Loaded → Applicable → Executed → Qualified`は**一つの実行結果に同居する独立した証拠列**であって、第二のSource Registryや新Test Engineではない。

- **Declared**：試行前に固定したRequired Source/業務Model/Rule/Scenario/Target/Obligation集合、scope、版、expectationの出典を記録。Declaredが不明なら「0件」ではなく`requirements-unknown`。現在のWorkspace manifestと試験要求の**必要集合**は区別する。
- **Discovered**：`workspace-files`や既存Indexで実際に列挙したpath/root・状態を記録。単にリストに載ったことはLoader入場を証明しない。
- **Loaded**：Parser/Loaderが実際に取り込んだ対象、raw source byte/source location、必要なimport/否定依存、診断・スキップ理由。既存CLIがこの列を返さないなら`not-observed`と書き、完了したことを仮定しない。原文は許可された内部Evidenceに限定（#2812/#2516）。
- **Applicable**：対象Model/Operator/Rule/Step/Targetのpin、型、Scope、必要分岐、Source位置、partial/permission-denied、実装対応の可否を照合。未知Stepは暗黙skipせず`unsupported`。
- **Executed**：実argvと実行したTool/Artifact/Scenario/Fixture/Workspace、実際に消費したBuild Input（#2595/#2819）、exit/result/evidenceNatureを記録。`validate.ok`は実業務Step実行を代替しない。
- **Qualified**：§11 G2独立Expected、G3実Receipt、G4 Coverage分母に対して必要証拠を判定。公開されたSuite/実Reader受領は§13で別資格。

**強いFail-closed条件**：あるObligationを満たすためのRequired Source/Loader/Contract/Targetが未読込または不明な場合、そのObligationのCoveredや全Target同等性を宣言しない。ただし探索・候補ケースの局所実行を妨げず、`candidate-only`と明示する。未取得/非開示の対象存在自体をAPI回答から推定しない。

### 14.3 最短の「AIが間違ったテストを生成しない」実証

既存の#2822〜24再現と#2813/#1180の受入試験を**別Runnerを作らず一枚の対照表**にまとめる。次の6対照を**新P0の入場反証 ADM-A〜F**と呼ぶ（横断計画§43のADM-01〜08と二重のCase正本を作らず、そのSubset/期待を引用する）。

| 対照 | 変更する入力は一箇所 | AI向けDiscovery/Fixtureへの期待 |
| --- | --- | --- |
| ADM-A / §43 ADM-01 | 業務宣言を全て外し、標準`std`のみ | Case inventoryの業務分母を「標準13件」としない。要求業務宣言が必要なら`unresolved` |
| ADM-B / ADM-02 | manifest `semantic`を未知`semantics`へ | 下流Validation失敗だけでSource未ロード原因を隠さない。名前補正候補はAI提案でも自動採用しない |
| ADM-C / ADM-03 | `.tei`を`.TEI`へ、または固定ファイル名の大小文字差 | 発見されたpathを実Load済み扱いにしない。現行観測できない理由は`unknown`と記録 |
| ADM-D / ADM-04 | `tei commands`で契約外`world`を選ぶ | 契約済み`tei.cli.*`を勝手に付与せず、mode・versionが未知なら拒否/限定候補 |
| ADM-E / ADM-05 | 宣言`audit.go`・実build`main.go` | 実消費Input不一致を`unqualified`。main統合#2819の負例を「修正済み」へ誤昇格しない |
| ADM-F / ADM-06 | #2813形式で1件を追加し、Expectedを候補Contractと共に誤変更 | JSON・局所Goの内部PASSと、固定検証基準/Mutantによる判別力を別に評価 |

**最小実証順**：①#2813 exact-head成功済みの11件を再利用して追加Case正常/拒否を測る → ②#2822〜24のSource/契約入場負例を必要集合と突き合わせる → ③#1180 Scenarioの隔離Fixture/CLI planへ到達するCaseのみ実行 → ④独立Expected/Mutantの正負対を採点する。③の前にSource closureが観測できなければ`blocked/unknown`を残す。#2604のSuite一式公開は採用した後段で検証。

対象ごとに**追加されたCase数、候補として実行可能なCase数、G2独立Expectedが適格なCase数、Requiredのうち実LoadedとApplicableなObligation数、未知/未探索/未対応数**を別集計。ゼロ/不明の分母を100%へ変換しない。実Go/CLIの正負Receiptとread-only外部Readerが揃うまで実証Doneではない。

### 14.4 今回の成果と残る判断

- **事実として参照できるもの**：#2822〜24が記録した特定main版・WSL Go実行の現象、#2813 HEAD `2219fc4d` のexact-head [Health #41744813520](https://github.com/hat47x/tei-platform/actions/runs/41744813520) completed/success、#2819統合の実Source/Build不一致負例、mainにあるScenario CLI v0/6 Step。版・Scopeを跨いで成功を合算しない。
- **横断レーンのOwnerを維持**：#2822（空宣言）、#2823（Source入場）、#2824（CLI ABI）、#2571（Workspace Reader）、#2595（実Build Input）、#2813（DSL JSON Candidate）、#1180（Scenario）、#2514（独立検証基準）、#2604（Suite公開）、#2516（Disclosure）。#2811は**AIテスト作成時に必要な前提を照合する受渡し**を担当。
- **未実施**：ADM-A〜Fの本作業による再実行、Source-loaded機械可読APIの新実装、実Provider/Web/GUI、選択的再資格化、AI外部Harness実試験、Owner意味受理。これらを実装済みと記録しない。

**採用判断**：P0の成功条件を「AIが新しいJSONを生成できる」に限定せず、**実際に読まれたSourceと対象契約が不明なら正しさ・網羅性を認定しない**ところまでとする。ただし既存IssueのCLI修正を全てP0実装の直列必須条件にしてAIの試行を止めず、局所Candidateと製品横断Qualifiedを分離する。


## 15. Driver側AI Case Intakeの最初の実装Reference（Draft #2828、2026-11-04）

**状態：実Goテスト/Scenario JSON/READMEを別Draftで作成。最新HEADの実CI資格は確認待ち。** 第12/14章の最小実装Track Bとして[Draft #2828](https://github.com/hat47x/tei-platform/pull/2828)（branch: test/driver-ai-json-candidate-intake-20261104、作成時HEAD: 71d9749c50b0b1cb95b3c2e07284ef56d8fbf772）を新たに開始した。#2811自身はdocs-onlyであり、実コードは既存#1180のScenario/CLI担当範囲の独立PRへ分離する。DSL側の[Draft #2813](https://github.com/hat47x/tei-platform/pull/2813)とは試験Subjectが異なる。新Runner/業務DSL/第二検証基準は作らない。

### 15.1 コードで具体化したP0-B

| 追加ファイル | 再利用する既存Consumer | この段階で確認対象となるClaim |
| --- | --- | --- |
| reference/driver-scenarios/ai-candidate-order-validation.json | 保存済tei.driver-scenario/v0、参照Workspaceの既存Validation Fixture | 新JSON ScenarioでValidation正常・拒否の2 Stepを独立保存できる |
| cmd/tei/ai_scenario_candidate_test.go | scenariotest.Parseと既存Scenario verify/generate/check/pinned run | ai-candidate-*.jsonの追加だけでGo側変更不要。Scenario/Step ID衝突、固定Anchorの保護、各Stepの生成CLI実行・限定Receipt、Expected誤改変・旧Plan失効を検査 |
| reference/driver-scenarios/AI_CANDIDATE_TESTING.md | 既存CLI、test-only制約、Source/Owner権限 | Agentが候補だけを書ける範囲と、独立Expected・Coverage・実Provider等の対象外を明記 |

**作成した2 Stepは既知の合成Fixtureを使った候補であり、生成AIが未知のテストを自律発見した実績ではない。** Goの限定試験成功が得られても、テストの独立検証基準、全Source Loaded、Web/Auth/実DB Effect、業務意味受理、全Required Coverageは別Claim。元の6 Step Scenarioは変更しない。

### 15.2 PR #2828の実測による受入手順

1. **Exact HEAD固定**：HEAD/Go/toolchain/Source・Scenario/Fixture bytes/Plugin Lockと実CI run IDを記録。後続の修正・rebaseは別HEADとして判定。
2. **Focused test**：既存Goの TestAIAddedScenarioCandidateFiles と TestAIScenarioCandidateRejectsWrongExpectedAndOldPlan を実走。候補の追加検出、verify→generate→check→generated-pinned run、2件の固定Anchor、期待値改ざん拒否と旧Plan拒否を区別する。
3. **Full Repository Health**：Focused testだけでなく正規Format/Go vet/Go全体/Workspace Validateの結果を確認。別PRの成功は合算しない。
4. **AIによる新規Case生成**：専用branchでJSONのみを1件追加し、既存のAnchor/独立Expected/検証基準/holdout/Canonicalは編集禁止。最初はread-only Validationの正負、次に異質Domainへ拡張。実Agent argv・Source・Result・費用/実時間を記録する。
5. **独立検証基準とMutation**：#2514側で事前固定した別根拠のExpected・既知Mutantが追加Caseの新規判別力を示すか測定。生成CandidateとExpectedが同じ誤りを持つときはqualifiedへ昇格させない。
6. **Source入場/Obligation**：#2822〜24で確認したDeclared/Discovered/Loadedの欠落、CLI契約外/未知Targetを別列で保持。局所合成Caseの成功は全Workspace/全業務の完全性を保証しない。

### 15.3 次段への橋渡し・二重機構防止

- #2813のPipeline JSON候補と#2828のScenario JSON候補を**一つの新共通Schema/Engineへ先に移さない**。実Provider、Target、型と期待値の契約が異なるため、対象とClaimを分離する。
- 共通化対象はまずread-onlyの **CaseID/Subject/Source revision/Contract Pin/Obligation/検証基準出典/実Tool・Fixture・Receipt/Scope/unknown** の比較列だけ。実Runnerの出力に無い欄は未観測とし、勝手に生成しない。
- CLI/Web E2Eの対称生成、UI DOM/Visual、採用Suiteの世代一式公開とConsumer受領は#1180/#2391/#2604の既存Ownerに後続委譲。P0で新PublisherやEKI必須Hostを導入しない。

**本設計文書追補はdocs-only。** PR #2828の最新実CI、AIによる未知Caseの実生成、G2独立検証基準、G4業務Coverage、実Web/GUI/Provider/Owner受理は本作業でNOT RUN／未確認。

## 16. AI生成テストの実行漏れ・SKIP緑・Source再資格化を分離する（2026-11-04）

**区分：最新CI/Go実装の確認に基づく既存P0受入契約の更新。§15のSource・Runner・検証基準責務を置換しない。** 今回確認した既存成果は、(a) #2828の実Go全体テスト成功とgofmt原因のHealth失敗、(b) #2835の必須試験SKIP/欠落検出、(c) #2681のWorld/Source改訂非同一性。いずれも**局所の限定Evidence**であり、AIが未知ケースを生成して不具合を検出したことは未証明。

### 16.1 履歴とexact HEADを区別して受入状態を更新

| 対象 | 確認した実証・結果 | 成立するClaim／未成立 |
| --- | --- | --- |
| [#2811 設計PR](https://github.com/hat47x/tei-platform/pull/2811) | 旧設計HEAD `a4483e85` の[Health #41787123130](https://github.com/hat47x/tei-platform/actions/runs/41787123130)は **completed/success** | 旧HEADのdocs正規CI成功。この§16追記後HEADの新CIへ結果転用不可 |
| [#2828 Driver Candidate](https://github.com/hat47x/tei-platform/pull/2828) | 旧HEAD `71d9749c` の[Health #41786950497](https://github.com/hat47x/tei-platform/actions/runs/41786950497)は**completed/failure**。実ログ：`gofmt -l .`は`cmd/tei/ai_scenario_candidate_test.go`で失敗、`go vet ./...`・`go test -count=1 -timeout 60m ./...`・Workspace validateは**PASS** | 対象旧HEADの限定Goテスト実走は成功、総合Healthは不合格。別コミット `77baf709` でReceipt匿名構造体の整列のみ修正（製品ロジック不変）。**修正後HEADのCI成功は確認してから主張** |
| [#2813 DSL JSON Candidate](https://github.com/hat47x/tei-platform/pull/2813) | HEAD `2219fc4d` の[Health #41744813520](https://github.com/hat47x/tei-platform/actions/runs/41744813520) **completed/success** | test-only 11候補・重複JSONキー拒否の実Go基準線。独立検証基準や業務Coverage資格にはしない |
| [#2835 必須PG試験入場](https://github.com/hat47x/tei-platform/pull/2835) | 実PostgreSQLへのGo testを実施したローカル記録と、DSN無しで`go test`が成功しつつ21件SKIPされる対照。現mainの`.github/workflows/tei-effect-fence-postgres-ci.yml`に`required test did not PASS`とSKIP拒否検査が存在 | **実行結果の緑≠要求された試験の実行完了**を実証。Effect-fence専用CIのPolicyを全AI Testへそのまま複製するのではなく、「必須Obligation/Caseを実際に実行したか」の契約へ移植 |
| [#2681 組織Native Source反例](https://github.com/hat47x/tei-platform/pull/2681) | 実`.tei`のコメントのみ変更して**WorldFingerprint同じ・ModelRevision異なる・read-only Operation同じ**を検証した限定Native [#41786879258](https://github.com/hat47x/tei-platform/actions/runs/41786879258)成功 | 同じ業務意味が使える場合でもSource改訂/出典引用・実Consumer再受領は別資格。全Source closureやArtifact completeを保証しない |

### 16.2 G3とG4に「実行したはず」の確認を追加

**required Caseの事前固定**：AIへ配るWork Packetに`requiredCaseIDs`、そのCaseが対象とする`obligation IDs`、対象の実行environment/fixture capability、禁止されたSKIPを記録する。これはScenario/Issue/既存CIの**派生read-only受入表**であり、二つ目のSuite正本・新Runnerを作らない。

**試験実行後の判定**を4つに分ける。
- **discovered**：Glob/Scenario Loader/CIが必要Case IDを見つけたか。globの上限/unknown path/途中失敗を「全件発見」にしない。
- **scheduled / attempted**：Caseが実行計画に含まれ、実際にAttemptされたか。生成されたPlanが存在するだけでは実行済みとしない。
- **terminal outcome**：`PASS / FAIL / SKIP / NOT RUN / BLOCKED / UNKNOWN`を明示。必須試験がログ欠落、SKIP、timeoutのときは**当該Obligationをqualifiedにしない**。任意環境の補助テストのSKIPはそのプロファイルの明示ポリシーに従い、すべてのskipをグローバルなbuild failureにしない。
- **qualified**：G2独立ExpectedとG1 Source/Scope/Targetの適合、G3実Receipt/G4必要Obligation分母と照合し、**Required Caseを全件PASSした範囲だけ**で宣言する。数だけ合ってもCase ID違い・同一Caseの重複実行・旧Headのログ流用は認めない。

合格条件の例：`requiredIDs - observedPassedIDs = ∅` かつ `observedSkippedRequiredIDs = ∅`、各Receiptの実Source/Tool/fixture/attempt版が一致、必要な独立Expectedがqualified。これを満たしても**要求Obligationが不完全な場合はCoverage 100%としない**。

### 16.3 AIが実際にテストを増やす最小の実証

- **P0-A**：#2813の既存JSON test-only 11件は固定し、Agentが新しいファイルで正常1件・拒否1件を追加。追加前にRequired Case IDsを定め、`go test`の出力・実Subtestの存在・重複ID拒否・独立検証基準未確定を区別。
- **P0-B**：#2828の既存Candidate JSONに対し、既存Anchorとtest-only Goを変更せず**新しい`ai-candidate-*.json`を1ファイル追加**。実CLI `verify→generate→check→generated-pinned run`でStep全数を実行したことを確認し、対象の`evidenceNature`を限定保存。新規Caseと固定Anchorを同じ意義のテストにしない。
- **P0-C**：上記の必須Case一件を「glob対象外拡張子にする」「Testの`-run`フィルタで除く」「任意条件でSKIPさせる」「CIがログを一部切り詰める」のうちまず2件の変化注入で試し、見かけのGo成功と**資格失敗**を別に出す。GoテストRunnerへの新しい必須機能を先に増やさず、既存出力と独立受入表で比較。
- **P0-D**：#2681の事前固定Expected（`World同一=true、ModelRevision同一=false、Operation同一=true`）をInput/Mutationとして供給。AIが実ResultからWantを再計算してもG2独立Expectedの改変は不可。Source引用の旧版扱い、モデル意味の再利用、Artifact/Consumerの再資格化は分離する。

このP0の後にのみ、同一Scenario→Web/CLI派生、GUI/Visual、Published Suite/Consumer receipt、実DB/Hostへ進む。実Sourceについては#2822〜24のDeclared/Discovered/Loaded、実Builder入力については#2595、実消費byteについては#2604/#2516に既存Ownerを維持。

### 16.4 独立した負例を重複登録しない

新Case IDをいきなり正式Scenario/Obligation正本へ増殖させず、以下を**既存#2835・#2813・#2828・#2681の反証を利用する受入表**として固定する。

| 反例 | 受入判定 |
| --- | --- |
| 必須Case IDが1件欠けるが`go test`終了0 | `qualified=false, missingRequired` |
| 必須Caseが`SKIP`、他はPASS | `qualified=false, skippedRequired`。任意テストのskipは別 |
| 旧HEADで合格した同名Caseの結果だけが存在 | `stale`。新実行Receiptがない限り現HEADの資格にしない |
| globで拾わない新しい拡張子にAI Candidateを置く | `not-discovered`。未知母集団なら全件Coverage主張を停止 |
| Sourceコメントだけ変更したがWorld/Operationは同じ | 意味計算の再利用候補とSource改訂・引用資格を分ける |
| 独立Expectedがないまま全Case PASS | `exploratory/locally-asserted`、G2不成立 |

**採用範囲**：既存の#2514は独立検証基準/Mutation、#1180はScenario/TargetとObligation、#2835は既存Effect-fence CIの実行漏れ検査、#2681/#2600はSource/Meaning/Artifact再資格化、#2811は各Producer/Consumerの**試験採用条件の受渡し**のみ。生成AIにCase追加と修復の自由は残し、Expected/Required分母/実Grant・非公開holdoutの無条件書換えを許可しない。

**本§16はdocs-only。** 今回新たなAgent未知Case生成、P0-Cの注入、検証基準/Mutant、Web/GUI/Provider、全Source closureの実行はNOT RUN。#2828の書式修正コミットは別PRであり、この設計PRにGoコードを含めない。

## 17. テスト基本設計・証拠台帳への接続（2026-11-06）

**独立した基本設計参照**：[テスト基本設計と証拠台帳（Issue #2514）](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6773033710)（本Draftで追加・docs-only）。本章は§8〜16のAI-first Case作成・Required/Discovered/Loaded/Executed/Qualifiedの仕組みを、Unit/Fixture・Conformance・実Provider/DB・実Host/UI・Owner受理へ**誤資格昇格させずに**接続する引継ぎであり、第二Scenario正本・新Evidence Store/Runner・新Gates/Authorityを導入しない。

**層別の最低受入契約**：
- L1は指定Fixture/実装の局所結果。L2は独立Expected/Mutant/Profile/Scopeの限定適合。L3は実Provider/DB/Read・Commit・Receipt/独立Read。L4は同一Binary/実Target/Host/Consumer受領。L5は業務Ownerの意味・Scope・版受理（自動試験の上位PASSではない）。
- 同一Caseを引き継いでも、AttemptとClaimは別票：固定Source/Model Revision/実消費Input、独立検証基準の来歴、Environment、観測対象、negative control、失効条件、現在資格を混ぜない。特にSource同値≠現Grant、表示値同値≠Read Scope/粒度・意味同値、CI PASS≠必須Case実走、RunPlan blocked≠先行Effectゼロ。
- 基線は#1129 XL-02（合成quantity=2, price=300→amount=600）とXL-03（0→1、1→1、rollbackなど）を分け、#2514 independent Expected/観測十分性、#2622 DO/DB Conformance、#1174 Commit Unknown/Receipt、#1163同一一改訂Packet、#1119正式Gateに接続。#2853/#2854/#2857/#2859の個別実PG/Host成功は同一Attemptの製品Gate PASSへ合算しない。

**今後の小さな実証**：既存#2813/#2828のJSON追加試験で、必須Caseが実際に発見・実行されたか、独立Expectedが既知Mutantを検出できるかを先に測る。必要なRead/Effect/Host証拠は既存Ownerで実測し、台帳は**その読取View**で比較。Required checks不足（#2616 ACU）や途中Block後先行Effectあり（#1130）の負例を共有し、無言の全PASSや無副作用Claimを拒否する。

**状態**：基本設計文書・本接続はdocs-only。基本設計の台帳プログラム実装、XL-02/03の全層実走、新AI生成/Mutation実験、実第二DB・一体Web→PG→Consumer、Owner意味受理はNOT RUN。既存#2811／#2828の旧exact-head CI成功を新HEADへ転用せず、変更後のCIは別Runで確認する。

## 18. 実証済みEvidenceの記入例とAI-first最小検収（2026-11-06）

**参照・状態**：[テスト基本設計と証拠台帳§8](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6773033710)に、異なるSource/Run/Environmentで得られた実験を**ケース×Claim×Attempt**に分けて記入した。#2811はAI-firstのCase作成・実走への入口のみを担当し、検証基準/Provider/Host/Owner Gateの別正本を作らない。本接続はdocs-only、新しいGo/Provider/E2E/AI自律生成実験はNOT RUN。

### 18.1 実証した範囲の更新

- [#2828](https://github.com/hat47x/tei-platform/pull/2828) のHEAD `77baf709` [Repository Health #41850080590](https://github.com/hat47x/tei-platform/actions/runs/41850080590) は**completed/success**。旧§15の「Go/CI未確認」は作成時履歴であり、現時点では限定Case intakeのGo回帰成功と区別する。AIの未知Case自律発見・独立検証基準・Business Coverageは未確認。
- [#2830](https://github.com/hat47x/tei-platform/pull/2830) のHEAD `019da011` [Repository Health #41851323500](https://github.com/hat47x/tei-platform/actions/runs/41851323500) は**completed/success**。保存済6 Stepの全IDと生成argv、各Stepの実行モード・`evidenceNature`の限定同値を検査する。既存`Verified=true`の意味を実Provider/Command/Web資格に拡大しない。
- [#2870](https://github.com/hat47x/tei-platform/pull/2870) は、in-memory `unknown`が再起動相当の新Lifecycleに残らないという**欠陥特性をテストで確認**。実PostgreSQLの同Key `IDEMPOTENT_REPLAY`は二重適用を抑えた限定証拠であり、durable Stage実装や新Key ABA防止が達成されたわけではない。
- [main統合済 #2882](https://github.com/hat47x/tei-platform/pull/2882) はWorkspace Source→Period/Effect/RunPlanのGo試験を追加。PR HEAD `ee2b3796`の[Repository Health #42097287333](https://github.com/hat47x/tei-platform/actions/runs/42097287333)でGo全体テスト・vet・gofmt等**completed/success**を実ログ確認。これは期間closedで最初のStep未呼出、openで合成Step resolved、Effect Observer未登録でblockedという**局所合成**の証拠。必要Guard集合の自動網羅、実Provider Effect、一般RunPlan途中停止の先行Effectゼロを証明しない。
- [#1163](https://github.com/hat47x/tei-platform/issues/1163) の実Web・実PG・Schema Gateは別々のHEAD/Attempt。監査が`INCOMPLETE/productGate UNQUALIFIED`である限り、個別CI成功を一改訂Product GateのPASSへ合算しない。

### 18.2 AI-first P0で受理すべき最小の比較票

| 比較単位 | 事前固定するもの | 実行後に要求するもの／禁止する誤昇格 |
| --- | --- | --- |
| **A. Scenario Inventory** | 6 Step ID、必須Case/Obligation、Source/Scenario/Fixture/Plugin Pin、Expected Nature | 各Stepの実Discovered/Attempted/Outcome/Receipt。6件が実行済みでも業務全体の分母にしない |
| **B. AI New Candidate** | #2828固定Anchor＋[#2890](https://github.com/hat47x/tei-platform/pull/2890)のJSON-only新2 Step/Fixture＋[#2895](https://github.com/hat47x/tei-platform/pull/2895)の別Goテストで固定した集合一意性・Expected/Input反転Negative Control（異なる証拠Role） | **exact-head統合Go PASS**：#2895 HEAD `8d3dc2fc` [Health #42109027966](https://github.com/hat47x/tei-platform/actions/runs/42109027966) はGo 1.26.10/Linux、全Go Test・gofmt・vet・Workspaceがcompleted/success。既存`TestAIAddedScenarioCandidateFiles`はCandidate Glob→Scenario生成→Step毎`generated-pinned` Receiptをassertする構成で、#2895は固定2件のKey一意性と2種の負例を独立にassert。**個別のRaw ReceiptはCI成果物として保存されていない**ため外部Reader Evidenceは未成立。#2890単体のActions0件、Product Rule Mutant kill未実証、業務G2/全Coverage/実DB/Owner未受理。[#2891](https://github.com/hat47x/tei-platform/pull/2891)は別Draft・未積層 |
| **C. Required Check** | Accepted Contract・Workspace/Capability・対象Host Grantなど、**候補やSupplied Checksとは独立のRequired出典**をOperationごとに固定。適用範囲外のCheckは加えない | `Required / Supplied / Evaluated / Enforced-at-Effect`の対応を同一Source/Operation/Attemptで比較。未指定が1件でも残れば`precondition complete`を拒否。#2882の合成Guard PASSを実Commitフェンスへ昇格しない（#2616/#2884/#2860） |
| **D. Effect and Restart** | Step順・OperationID・Attempt/Generation、未知Commit・Provider Receipt/別Read、同Key/新Key Expected、Selection/Context Epoch | 全Plan`blocked`は先行StepのEffectゼロを意味しない。Restartでstage消失・新Key ABA再適用を安全とせずReceipt照会優先。非同期の旧Context結果を現在画面へ誤適用しない（#2870/#1174/#2616 §20） |
| **E. Cross-layer Qualification** | Source・検証基準・Consumerの同一Attempt資格、Owner Meaning状態、対象TargetのScope | L1/L2/L3/L4/L5の単純合算を拒否。#1163/#1119のGate Ownerが同一改訂の必要証拠を判定する |

**Exit**：A/Bで新規Caseが実発見・実走し、独立Expectedで既知Mutantを判別した有限の正負対を確認できること。C/Dで、テストPASSでも製品要件未充足となる既知負例を正しく報告できること。Eは既存Ownerの実Provider/Host証拠が揃う段階で昇格し、AI P0の開始条件にはしない。test-only Scenario候補・Ledger Viewから製品実行許可を得たとみなさない。

**未完了**：本章の新P0実行・自律生成試行、独立Mutation評価、Ledger実装、現Grantを含む一体Web→PG→Consumer、正式Owner受理。過去の成功した別HEAD CIを本HEADのCI結果へ転記しない。

### 18.3 Fixture memberの一意性をG0で検査する（Draft #2891）

**本Planへの導入はG0の入力明確性だけ。** 現mainの`scenariotest.Parse`は`RejectDuplicateJSONMembers`を実装済みだが、`ParseValidationFixture`／`ParseBrowseFixture`／`ParseObservationFixture`は単一`json.Decoder`によりobject memberの重複を拒否していなかった。重複メンバーが後勝ちの場合、候補JSONとFixture、Actual/Expectedの比較の前提となる入力が曖昧になる。

- [#2891](https://github.com/hat47x/tei-platform/pull/2891)：mainから独立したG0 Draft。旧HEAD `2f677921`の[Health #42107910706](https://github.com/hat47x/tei-platform/actions/runs/42107910706)はGo全体/vet/Workspace PASSだがgofmt driftで総合failure。**新HEAD `16b8d554`で書式のみ修正**し、[Health #42110507652](https://github.com/hat47x/tei-platform/actions/runs/42110507652)は照会時in_progress。旧結果と新HEADの資格を分け、完了後のCIでG0修正候補を評価する。Scanner再利用＋重複Member7／正常3件、製品Runner/業務Rule変更なし。
- [#2890](https://github.com/hat47x/tei-platform/pull/2890)：既存#2828に積んだJSON-only AI候補。正常`[1,2]`／重複`[1,1]`の行番号配列について、リポジトリから実Readした2 Fixtureに対し独立な集合重複の簡易計算を行い、期待の`satisfied/not-satisfied`と整合することを確認した。これは**合成入力の算術的sanity check**であり、TEI CLIの実Step Receipt／業務Owner独立検証基準／L2 Conformance資格ではない。現PR headのActions Runが取得できない限り`NOT RUN`。
- **区別する負例**：Relation`keys`の`[1,1]`（正当な重複データとしてValidationが拒否する対象）と、JSON objectの同名`keys`member 2回（Parserが拒否する入力不正）は別。どちらかの拒否成功でも他方や実Target Coverageを自動PASSとしない。

**次の最小検収**：①#2891 exact-head Go Test/Repository HealthでFixture解釈の一意性を確認、②#2890の積み重ね型ブランチを合成したcheckoutで既存`TestAIAddedScenarioCandidateFiles`を実行し、新Stepが実際に`discovered/attempted`かをログ/Receiptで確認、③事前Expectedを変えず`line_number_unique`の誤判定Mutantを判別して初めて#2514 G2へ渡す。#2891と#2890は別HEADなので受入証拠を合算しない。


### 18.4 実DB事前入場とEffect証拠を分離する（main #2894）

main統合済み[PR #2894](https://github.com/hat47x/tei-platform/pull/2894)の`tei dbcheck`を、**実PostgreSQL検証に入れるかどうかのEnvironment Witness**として#2622/#1163へ渡す。HEAD `556d8e6a`の[Health #42108388836](https://github.com/hat47x/tei-platform/actions/runs/42108388836)は**completed/success**。実mainの`cmd/tei/dbcheck.go`はDSN欠落時には接続を試みずCLI exit2、DSN指定時は`pgx.Connect`・`SELECT version()`・同Session一時表CREATE/DROPを試行する。現mainのGoテストはDSN欠落・flag/env優先・表示整形を検査。**このCI成功が実PostgreSQL DSNを使った接続確認の実走を意味するわけではない**。

入場判定は`DSN absent→NOT ATTEMPTED`、`dbcheck reached＋temp DDL PASS→specified DB connection/capability observed`、`Required PG Case actually PASS＋実Receipt/独立Read→限定L3 Provider claim`を分ける。実DB Probe自身も一時DDLという作用を持つが、本業務TableのRead/Write・RLS・Snapshot・Commit・Current Grantの証拠ではない。DB接続先の生DSNは公開しない。#2835の**SKIP緑禁止**は実PG Case入場に引き続き適用し、dbcheck成功をRequired PG Caseの代理PASSにしない。

**この追補はdocs-only**。#2895のGo CI成功と#2891の書式CI失敗を正しく区別し、新しいProvider/実PG接続・Source/Owner受理・外部Reader/Prod Hostの実測は本作業ではNOT RUN。#2895は積層Draftのためmainへ単独mergeせず、#2828→#2890の依存順と#2514の独立検証基準受理境界を守る。

### 18.5 有限AI候補の実CIとG0修正の未結合を確認する（2026-11-06）

[証拠台帳§8.5](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6773033710)の現在判定を本計画へ引き継ぐ。**#2895の積層HEAD `8d3dc2fc5058ff3945d0981d2979138d4fc461be`**は、[#2828](https://github.com/hat47x/tei-platform/pull/2828)→[#2890](https://github.com/hat47x/tei-platform/pull/2890)→[#2895](https://github.com/hat47x/tei-platform/pull/2895)を同一Go checkoutへ含む。実[Health #42109027966](https://github.com/hat47x/tei-platform/actions/runs/42109027966)は**completed/success**。job logで`go test -count=1 -timeout 60m ./...`（`tei/cmd/tei`含む）、gofmt、vet、Workspace validateのPASSを確認した。比較Diffから、#2890→#2895の追加は**独立有限検証基準のGo test-only 1ファイル**と確認できる。#2890単独HEADのActionsは0件であり、その単独CI成功と誤表示しない。

**証明範囲**：#2895の単一Go Testでは、固定2 Step・入力Fixture・有限集合の行番号重複Property、Expectedだけ反転/入力Keysだけ反転の対照を検査。既存#2828 Go Testは候補Glob・既存Scenario `verify/generate/check/generated-pinned run`・Stepごとの限定`evidenceNature`を検証する。**同HEADのGo全体PASSはこの局所Scenario検収を支持する**が、個別Stepの外部Receipt JSONをCIから独立保存した証拠・業務Ownerが受理したMeaning・実Rule本体Mutant kill・実Provider/Target/全Coverageの証拠にはならない。

**G0独立ギャップ**：[#2891](https://github.com/hat47x/tei-platform/pull/2891)のFixture JSON Object重複Member拒否は#2895 Stackに含まれない。#2891の旧HEAD `2f677921`は全Go test/vet/Workspace PASSながらgofmt driftで[Health #42107910706](https://github.com/hat47x/tei-platform/actions/runs/42107910706)失敗。今次`16b8d554`にて**整形のみ**修正、[新Health #42110507652](https://github.com/hat47x/tei-platform/actions/runs/42110507652)を別判定中。G0の修正成功とG1/G2候補の別HEAD成功を**同一受入Attemptへ足さない**。

**既存計画の優先検収差分**：P0a #2891 exact-head総合CI PASS＋必須10 Case非SKIP確認 → P0b #2895で追加2 Stepが既存Go CLI test経由で実走した証拠の再取得 → P0c 一体HEADに必要な最小4ファイルを適用し、G0と限定G1/G2候補を同一Checkoutで試す（新Runner/新Schemaなし）→ P1 #2514独立既知Product Rule Mutantの実kill。最終Owner Meaning、Provider/Host・一改訂Web→PG→独立Consumer資格は引き続き#1163/#1119が所有する。

**今回の計画追補はdocs-only**。実Goの修正は#2891、実新Case/検証基準は別PRであり、本Planに製品コードを複写しない。#2895 Stackは検証用Draft・単独main merge禁止。


**横断の負例出典（Test SSOTの追加ではない）**：[#2892](https://github.com/hat47x/tei-platform/issues/2892) のMigration History `AppliedUnknown`が`RepoApplied`に誤分類され得る問題／取得欠落を真の空Historyへ置換してしまう問題と、[#2887](https://github.com/hat47x/tei-platform/pull/2887)の同DatabaseID・Contract後勝ち、#2891のJSON同名Member後勝ちは、入力種別は異なるが「曖昧・未取得・重複を黙って成功にする」という**資格違反**を共有する。それぞれ#2892（HIST-01/03の独立Expected、NOT RUN）、#2887/#2897（DB移行Gate、実業務未検証）、#2891（test-only Fixture Parser）へOwnerを維持する。#2811は必要ObligationとUnknown表示の受渡しだけを担当し、新共通Validator/History Engineを実装しない。

### 18.6 同一改訂検収用Draft #2901と次のMutant Gate（2026-11-06）

**新規統合Draft**：[#2901](https://github.com/hat47x/tei-platform/pull/2901)は、既存#2895の有限検証基準/AI Scenario Stack成功HEADに、#2891の実CI成功HEADの**Fixture Parser 3ファイル＋7負/3正のGoテスト1ファイルだけ**を重ねた隔離検証。#2895とのDiffは4ファイル（Goの新Runner、Canonical Rule、Scenario JSON、既存Expectedは不変）。これは#2891/#2895のOwner PRの代用・新しいプロダクト正本ではなく、**同一実HEADの結合CIが成立するかを調べる一時的なDraft**であり、mainへ単独mergeしない。

- **既存確定**：#2891 HEAD `16b8d554` の[Health #42110507652](https://github.com/hat47x/tei-platform/actions/runs/42110507652)、セルフビルド、依存検査は完了success。独立#2895 HEAD `8d3dc2fc`の[Health #42109027966](https://github.com/hat47x/tei-platform/actions/runs/42109027966)も完了success。別HEADの2成功は同一Claimへ足さない。
- **現統合Attempt**：#2901 HEAD `a6eb80cffe16a57f4cb2461ff2946065955a4e50`の[Health #42111973682](https://github.com/hat47x/tei-platform/actions/runs/42111973682)が作成時queued。**CIの終了/Go結果はこの追補時点で未確認**。未実行・未発見・SKIPのRequired Caseを合格にしない。
- **出口の検証対象**：G0は3種類のFixture入力object member重複拒否（7負・3正）、G1は新2 Stepの`verify→generate→check→generated-pinned run`と限定Receipt Nature、G2候補はCandidateとは別のKey集合Property・Expected/Input反転拒否。同じcheckoutのGo実結果を要求。**実Rule本体Mutant killは未実施**、正式G2業務意味とG4 Coverageへ昇格しない。

**P0後の選定**：次は#2514の小さな公開・合成の実Product Rule Mutant（特定Validation Ruleの比較・一意性判定の意図的誤実装）を**事前Expected固定**の状態で試す。Mutant killなら対象Contract/Case/Environment/Roleの限定判別力、見逃したら必要Observation/Rule Selectの不足を記録する。Candidate Expected・Required Obligationの分母をAgentが自己修正して「PASS」にしない。#2892 Migration Historyのunknown偽成功、#2884 Required Guardの欠落と現在Grantは別Owner/別Evidenceで扱い、同じ仕組みへ無理に統一しない。

**証拠台帳参照**：[基本設計§8.6](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6773033710)。今回の計画更新はdocs-only。#2901のコード・CI実行とは別の改訂として保存し、当該Docs HEADのCIも独立確認。

### 18.7 実Product Rule Mutantを使う判別力検収（2026-11-06）

**P0 統合Greenは確定**：[#2901](https://github.com/hat47x/tei-platform/pull/2901) head `a6eb80cffe16a57f4cb2461ff2946065955a4e50`の[Health #42111973682](https://github.com/hat47x/tei-platform/actions/runs/42111973682)・セルフビルド・Studio・Web E1-A・依存チェックが同HEADで**completed/success**。これによりG0（FixtureのJSON Object重複拒否7負/3正）、G1（AI追加2 Stepを既存Scenario CLI Goテストから検査）、G2前段（正負Key集合・Expected-only/Input-only反転の有限独立対照）を**一つのGo checkoutの限定回帰**として確認。Raw Step Receipt・Business Owner受理・Provider/Host一改訂E2Eはここには含まない。#2901は検証用Stack Draft、単独merge禁止。

**新しく試すP1の本質**：[#3017](https://github.com/hat47x/tei-platform/pull/3017) は、#2901成功HEADを固定し、実Product Evaluator `internal/semantic/validationeval/evaluator.go`の`relation-unique`重複時 `not-satisfied`を誤って`satisfied`に返す**一行Mutant**だけを作成。Scenario/Fixture/Goテスト/独立Expectedは変えず、**期待するのはCI Red**。故意に不正な製品コードを含むため絶対mainへ統合しない。#2514が判別力の責務Owner。

- **M1 kill 成立条件**：#2901 exact HEAD Green ＋ #3017 exact HEADの差分一行 ＋ #3017の実CIジョブログで`TestAIAddedScenarioCandidateFiles/ai-candidate-line-number-uniqueness`の実行・`result=satisfied, expected=not-satisfied`のような対象Rule誤実装に起因するFailを確認。単に別のEvaluator回帰のみFailなら「既存テストに殺された」であって**AI追加Caseの検出力は未確認**。
- **M2/M3 fail分類**：Go Build、gofmt、Test漏れ/全SKIP、環境失敗、他TestだけのFailは`inconclusive`として分ける。CIの赤をそのままG2合格にしない。
- **M4 survivor**：対象Caseの実走と固定Expectedが揃っても緑ならRule Mutantがsurvive。Scenario選択・actual Loaded Source/Rule・実Observationが不足している可能性を記録し、AIがExpectedをGotに合わせて更新しない。
- **出口**：M1なら**当該合成Rule/対象Case/固定Revision限定のMutant kill**だけ認定。業務OwnerによるMeaning受理や全Required Coverage、他Rule/NULL/部分Read、実Provider/Hostを証明しない。#1163/#1119 Product Gateは別途維持。

**証拠参照**：[基本設計台帳§8.7](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6773033710)。#3017の[Health #42181290981](https://github.com/hat47x/tei-platform/actions/runs/42181290981)は本計画追加時点で**in_progress**。該当Go Testの実FAILログ確認前は「Mutant kill NOT VERIFIED」。本章はdocs-onlyであり、実コードは隔離Draftに保持。新Runner、Policy Engine、判定基準専用DB、第二Case正本は作らない。

#### 18.7.1 実CIによるProduct Mutant kill確定（M1）

[#2901](https://github.com/hat47x/tei-platform/pull/2901)の**正常HEAD** `a6eb80cffe16a57f4cb2461ff2946065955a4e50`（[Health #42111973682](https://github.com/hat47x/tei-platform/actions/runs/42111973682) completed/success）と、[#3017](https://github.com/hat47x/tei-platform/pull/3017)の**製品Evaluator一行だけ誤変更したMutant HEAD** `0f795a012dc16200ee85e5f6b3f0b921add7bdc4`（[Health #42181290981](https://github.com/hat47x/tei-platform/actions/runs/42181290981) completed/failure）を実際に比較。

**JobログによりM1を認定**：`TestAIAddedScenarioCandidateFiles/ai-candidate-line-number-uniqueness`が実Fail、`ai-line-number-duplicate`は `Canonical Validation sales.order.lines.line_number_unique result=satisfied, expected=not-satisfied` と出力。Go全体はFail、gofmt/vetはPASS。製品Evaluator本体の誤変更を**追加AI Scenarioが直接検出できた**ことが、単に「別のEvaluatorユニットテストがFailした」という場合とは違う成果である。Candidate/Fixture/Expected/検証基準 TestのBlobは正常HEADと不変。これは**1公開合成Rule/1 injected mutant**の限定感度であり、正式な独立業務Meaning、他Rule/NULL・欠損、Provider/Host/Owner、全Obligationは未資格。

**計画更新（完了と次を分離）**：
- **P0 completed (limited)**：#2828→#2890→#2895＋#2891のG0/G1/有限検証基準同一HEAD Go健全性実走（#2901）と、Product Evaluator誤実装一行MutantのM1 kill（#3017）。
- **P1 next**：#2514でMeaning/Rule版・独立Expected provenance・選択された実Ruleを対応付け、欠損/NULL、別粒度、Relation scopeの誤実装を**必要な一つずつ**負例にする。既知Mutantをsurvivedにした場合は不足した観測を別記。採用Suite/Readerの実受領は#1180/#2604/#2516、実Provider/Hostや一改訂Packetは#2622/#1163。
- **禁止**：不正コードの#3017は**永久にDraft、絶対mainへmergeしない**。CIのFailureを正常なRepository Health successと呼ばない。AI生成による検出率・Business Coverage一般化も主張しない。

詳細Evidenceは[基本設計台帳§8.7](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6773033710)と[#2514](https://github.com/hat47x/tei-platform/issues/2514)へ記録。本追補はdocs-onlyで、新Runtime/Provider/Generatorの実装は追加しない。

### 18.8 AI Scenarioの局所Mutation感度とOwner母集団の資格を分離する（2026-11-06）

[証拠台帳§8.8](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6773033710)の境界を、本計画の次の受入順に接続する。**既に実証済み**：[#2901](https://github.com/hat47x/tei-platform/pull/2901) `a6eb80cffe16a57f4cb2461ff2946065955a4e50` [Health #42111973682](https://github.com/hat47x/tei-platform/actions/runs/42111973682) successと、[#3017](https://github.com/hat47x/tei-platform/pull/3017) `0f795a012dc16200ee85e5f6b3f0b921add7bdc4` [Health #42181290981](https://github.com/hat47x/tei-platform/actions/runs/42181290981) expected failure（AI追加Caseが誤って重複を受理するEvaluatorを直接検出）。**次提案は未実行**であり、当該成功を全件Read/Owner資格に一般化しない。

**次の1 Mutant**は`ValidationRelationUnique`の**最後のキーを比較ループから外す**実装誤り。#2901の固定Source・Application`uses validation`・2 Fixture（`[1,2]`正常／`[1,1]`重複）・Scenario・独立Expected・Go Testの各Blobを不変にして、誤実装1箇所だけを別の**絶対merge禁止**隔離HEADへ置く。予測：AI追加`ai-line-number-duplicate`が本来`not-satisfied`なのに`satisfied`となる。実Go Test/Receiptでその対象Stepが失敗するまでは`Mutant killed`と記載しない。基線CI Green／Mutant CI Redの**失敗の帰属**（AI Case自身、別Goテストのみ、Build/infraのみ、survived）を分離する。

**同じテストでは証明不能なOwner対象集合**：既存`scenarioValidationSource.RelationKeys`は`[]string,error`であり、対象行数・全ページ取得・Snapshot・実RLS/現Grant・Owner定義の必要集合の同一性を伝えない。例えばOwner母集団が`[1,1]`でも実Readが`[1]`だけならEvaluatorの実装が正しくても`satisfied`になり得る。「取得された集合をすべて評価した」ことと「必要な集合をすべて取得した」ことを**別Claim**にする。これを確かめる実Provider試験は#2586/#1163へ委譲し、**0件、部分Read、取得失敗、非開示**を同一`[]`へ潰す設計を禁じる。NULL/unknownに対する業務Expectedは現行Fixtureでは未定義で、#2514 Ownerの意味判断以前に勝手な`satisfied`/error/unknownを確定しない。

**入場／再資格ルール**：①同一Source File/Revision・Rule選択・Relation scope・Fixture/Expected Blob・World/Plugin Lock/Toolchainの*実使用*を照合（#2595）、②ActualのCase ID/Step IDと実Attemptの実行・欠落・SKIPを確認（#1180）、③Owner対象集合の定義と実ProviderのRead completeness、Snapshot、Access／Disclosure範囲を独立に照合（#2586/#1163）、④Source/Operator/Plugin/Owner Meaning受理が変更されたときは過去の有限M1履歴を保存しつつ現在のClaimを失効・再資格化（#2600）。特に`.tei`→`.tei.md`へ文書表層を移す際も、同じ名前・バイトを意味受理・実Source解決の代用にしない。

**担当分離**：#2514＝独立Expected/Mutationの意味・判別力、#1180＝Scenario SSOT/必要Case、#2595/#2600＝実Source/失効、#2586/#1163＝実Read/Owner母集団/一改訂Packet、#2811＝AI-firstの検収順と証拠リンク。#2514の担当Chatへ重複して投稿・Mutant実装を行わない。第二Runner／検証基準 Store／全領域共通Meaning正本も作らない。

**今回の達成状態**：比較条件と再資格条件を計画文書に反映したのみ。末尾脱落Mutant、NULL/unknown Fixture、実Provider全件性確認、移行後の`.tei.md` Source実解決は**NOT RUN／NOT ACCEPTED**。

### 18.9 新実装のCompletion契約とAI 検証基準を接続する最小一往復（2026-11-07）

**今回の成熟差分は「Owner母集団の完全性」についての**現在の実装境界**を確定し、既存のG0/G1/G2を実Target Claimに無断拡張しない検収順を具体化したことにある。[#3022](https://github.com/hat47x/tei-platform/pull/3022)は`EvaluateCompletion`の静的分岐、Coverage自己申告の問題を§10.11〜10.15で調査済み。本計画は別のCompletion設計・検証器を発明せず、**AI-first ScenarioのEvidenceからCompletion Claimへ昇格するための独立入場条件**だけを所有する。

**検討時mainの直接ソース確認**：`internal/semantic/completion.go` 旧blob `512a835ca445d81dd043f8af78ca29df4e7adb3b`（#3051統合後の現main Blobは`e42610e9159e02ba3b5c7b84c46742a846331e64`） は`CoverageComplete/Partial/Unestablished`を**Callerの申告として受ける**。非空の観測集合が全`Holds=true`で`CoverageComplete`の場合、実Provider/Owner集合をAPI内部で照合しなくても`CompletionHolds`を返す。同じ`Targets=[L1:true],CoverageComplete`の呼出しでOwner集合が世界A`{L1}`と世界B`{L1,L2}`でもAPI結果は同じになり得る。これは**Caller入力が真なら局所的に正しい関数**と、**Callerの集合申告が実世界と一致したという証拠**の差であり、現コードから直ちに製品欠陥確定とはしない。

| 検収順 | 既存担当／固定する材料 | 独立Negativeと受理線 |
| --- | --- | --- |
| **P0-A Evaluator局所感度** | #2514 検証基準と#1180 Scenario。#2901 Green／#3017 Product Rule一行MutantのAI Case直接killは**実証済み**。次の末尾Key走査脱落は**未実行** | `[1,2]`正常・`[1,1]`重複、Source/Fixture/Expected Blob固定。MutationによりAI追加Stepが対象Ruleで失敗した時だけ局所Kill |
| **P0-B Coverage-claim偽陽性** | #2586/#1163がOwner対象Lと実Provider RowIDを同Scope/asOf/Revisionで用意。既存`EvaluateCompletion`を読み、CC-01〜03をテスト候補として渡す | Owner L={L1,L2}なのにReaderはL1:trueのみ。`CoveragePartial`なら`unverified`、Callerが同入力を`CoverageComplete`と偽申告すればAPIが`holds`でも**実業務Claimを拒否**。実Read/終端/Current Grantなしに`complete`を受理しない |
| **P0-C 反例の非対称性** | #2514が独立Expected。ReaderがL1:falseを確実に同Scopeで観測（L2未取得）したCase | `fails`は確実な違反観測を根拠に限定的に言える場合がある。一方同一partialで`holds`／「存在しない」／「影響なし」は言えない。Null/unknown/非開示は別State |
| **P1 Source/Authoring切替** | #3030/#2621で`*.tei.md`移行、#2595でSource actual input、#2600でEvidence再資格。Workspace `ManifestIdentity()`とSource Indexを別照合 | 旧`*.tei`のまま実Loaderが読む／新`*.tei.md`は未解析／Manifest変更でSelection変化／同ID二箇所を後勝ち、などを区別。ファイル改名、同じWorld digest、CI Greenだけで現行意味受理にしない |

**最小の一往復Receipt**：Scenario ID/Step ID、受理意味の対象とScope版、Callerが渡したTargetIDsとCoverage自己申告、外部Readerが独立に見たTargetIDs・Snapshot/ページ終端・RLS/Disclosure、実Consumed Source occurrence＋Manifest path/SHA、実World/Plugin/Toolchain/Artifact、Actual/Expected/ConsumerのClaimと限界。**既存Scenario・Provider Receipt・Source Index・Build Evidenceを引用する比較票**に限り、新たなCase SSOT/Runner/判定基準専用DB/Host/証拠Ledgerの実装を開始しない。

**再資格の最短規則**：Owner対象L・Predicateの意味/粒度、DataSource/Reader/snapshot/Grant、Workspace Manifest/Source参照/Plugin/Toolchain、独立Expectedの受理範囲が変わるたび、*影響するClaimだけ*を`historical-pass / current-unqualified`へ移し、必要なReaderとCaseを再実行する。単なるRaw Source Path移動は別途実Resolveを確かめるが、意味・Runtime権限を自動否定／自動再受理しない。Source A→B→Aで旧Receiptを新時点の実Read資格として復活させない。

**責務と未実証**：#2811は観測束縛と検収順、#2514は独立Expected/Mutant、#1180はScenario、#2586/#1163は実Reader/同一Attempt、#2595/#2600は実Sourceと失効、#2621/#3030はAuthoring移行を担当する。#2514・#3030の主担当PRや製品Codeへ重複投稿・変更しない。現行Sourceの静的比較と過去CIの再取得以外、新Completion Caseの実Goテスト、実Provider全件性、末尾Mutant、`*.tei.md` Reader移行、Owner受理は**NOT RUN**。[台帳§8.9](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6773033710)に独立条件・境界の詳細を記録。

### 18.10 空集合Policyの未取得断定を抑える入場条件と実読込Sourceの版（2026-11-07）

**§18.9の補正・旧mainと現行実証の区別**：修正前のmain `internal/semantic/completion.go`（blob `512a835ca445d81dd043f8af78ca29df4e7adb3b`）では、`CompletionEvidence.Targets=[]`でも`CoverageUnestablished`のとき、`EmptyPolicyDeny`なら`fails`、`EmptyPolicyRequiresEvidence`かつ`EmptyJustified=true`なら`holds`を返す。前者は[#3051](https://github.com/hat47x/tei-platform/pull/3051)で`unverified`へ修正、merge commit `b30ce3708a8385dcf219aed9f53e594bc0e95d12`で**main統合済み**。[統合側Health #42298116044](https://github.com/hat47x/tei-platform/actions/runs/42298116044)成功。いずれもCallerの申告であり、実Providerの全件Read/Ownerが要求した集合を関数内では独立立証しない。前者の`fails`を**実際に0件と確認した業務不成立**と読み替えるのは危険であり、後者の`holds`もboolだけで正当化しない。[`completion_test.go`](https://github.com/hat47x/tei-platform/blob/main/internal/semantic/completion_test.go)の既存単体テストはCoverage既知の空Policyを中心に検査しており、これらの「空だと申告されたが完全集合かは未取得」対照の実行証拠ではない。

**追加受入票**（[台帳§8.10](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6773033710) CEC-01〜05）。計画の順序は検証対象を一つずつ増やし、担当Ownerを跨ぐ証拠を統合せずに引き渡す。

1. **C0（Source/Case固定）**：`Owner TargetSet ID/revision/scope/asOf`、独立期待値の出典、`EmptyPolicy`、対象Row IDs、同版のReader Snapshot/ページ終端・RLS/Disclosure、実Caller引数を固定。正しいSource/期待の意味受理がない段階で正式Conformance PASSとは呼ばない。
2. **C1（局所入力の対照）**：空集合禁止で`Targets=[]+CoverageUnestablished`なら**旧mainは`fails`、現mainは`unverified`**。#1125 CDE-02に沿った修正と4つの追加正負対照は同一HEADのGo全体テスト成功で確認済み。ただしmainの局所Go修正と実Readerによる完全性・現Grant・Owner受理は別資格。`requires-evidence+EmptyJustified=true`はCoverage未確立でもコード上`holds`。別Scope/旧Snapshotのboolは独立空証明として利用不可。
3. **C2（実Reader witness）**：#2586/#1163が同一Attempt/ScopeのOwner集合・行Identity・全ページ終端・Snapshot/時点・current Disclosureを実確認。true empty、missing、partial、not-disclosed、permission-deniedを別状態にし、`[]`を万能な完了Witnessにしない。確実な反例L1:falseは限定`fails`を支持し得るが、部分集合L1:trueから全員成立は導けない。
4. **C3（Authoring source admission）**：`*.tei.md`への形式統一はOwner決定。未対応形式を黙って読み飛ばさず`TEI_MD_PARSER_UNBOUND`で拒否する[#3031](https://github.com/hat47x/tei-platform/pull/3031)の固定HEAD `63953ccb5c12ead26ec1268e7c0afdba4eff66ae`は[専用CI #42217379093](https://github.com/hat47x/tei-platform/actions/runs/42217379093)と[Health #42217378867](https://github.com/hat47x/tei-platform/actions/runs/42217378867)が成功（**Draft限定**）。Markdown Parser実行・受理Meaning・Source移行同等性は別。移行仕様・検収IDの主Ownerは[#3030](https://github.com/hat47x/tei-platform/pull/3030)（TM-01〜14）。[#2667](https://github.com/hat47x/tei-platform/pull/2667)は横断経緯・依存Owner引渡しの位置付けで、第二の移行Gateを新設しない。
5. **C4（Claim再資格）**：World/Rule IDが同じでも実Consumed Source、Manifest path/SHA、Scope、Operator/Plugin/Toolchain、実Provider/Grant、独立期待値の版が変われば、影響するClaimの適用資格を再評価。Source A→B→A、旧新二重ファイル、未ロード定義、異Tenant取得を反例とし、過去Go/CI PASSの履歴を保持して現在適格だけを`unknown/unqualified`へ戻す。

**#2514独立検証基準・Source実消費への追補（2026-11-07）**

- **実際に主張してよい最小Scope**：[#2514 CDE-Q01〜04](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6696895001)のClaim `A/O1/R1`に対し、Callerが`CoverageComplete`を与えても、Readerが別Tenant B/O1または旧Snapshot R0を読んでいれば**業務Claimはunqualified**。局所Completionの`holds/fails`を変更するのではなく、同一Tenant/Parent/Relation粒度/as-of/Read Witnessで外側の資格を判定。独立Expectedの正本・試験実装Ownerは#2514/#2586/#1163に残し、本Planから新Enum/Runnerを要求しない。
- **Indexed ≠ Parsed/Consumed**：[TAI固定旧TEI CLI #42297500012](https://github.com/hat47x/tai-pleroma/actions/runs/42297500012)は`*.tei.md`の原byte差でSource Index Fingerprintのみ変化し、Worldと旧Catalogが変わらない期待負例を再現。改名のみではSubjectが消えても`validate.ok=true`になり得る。#3031のfail-closed CIを新Parserの合格とせず、[#3030](https://github.com/hat47x/tei-platform/pull/3030)の既存TM検収で`required→managed/indexed→parsed/selected→execution-consumed`を測る。実新Parser/同一Source→実Provider/Ownerは未実施。
- **Attestationの判定上限**：[#2514](https://github.com/hat47x/tei-platform/issues/2514)のOKF Attested Computationを活用する場合も、Attesterの決定性・`verified`を独立期待値の出典や実Read/Owner/Hostの受理に昇格しない。検証器自体の版・入力bytes・job IDの独立再読取・結果Consumerを別Witnessとして扱う。OKF Attester製品試験は **NOT RUN**。

**次の単一反証の優先度**：#3051によって`CoverageUnestablished`時の不当な`fails`は局所的に改善された。次は**実Owner集合 `{L1,L2}`／Reader観測 `{L1}`／Caller申告`CoverageComplete`**の二世界対照を、#2586/#1163の同一Scope・Snapshot・Row identityを固定して検証する。`EvaluateCompletion`単体が`holds`を返しても、それだけで業務完了と認定しない。#2514の独立期待値はCallerのActualから導出しない。**新検収の実Provider試験は未実施**。

**検収終了条件**：C1のGo分岐の実観測、C2のRead完全性、C3の実Source解決、C4のOwner受理を**同一Attemptで観測できたものだけ**限定受理。#2514は独立検証基準、#1180はScenario SSOT、#2586/#1163はRowset/Provider、#2595/#2600はSource失効、#2667/#3030はAuthoring移行の担当。新判定Engineや第二Case正本を作らない。#3039/#3040の用語規約に従い、新しい記述は「独立期待値・独立検証基準」を使用し、現存コード名・機械的IDは保持する。

**状態**：現mainに#3051のCompletion Sourceが取り込まれたことを再読取し、#3051 exact HEAD `581e699b62b126c86388c88c7c357674394ad252`の[Health #42256518693](https://github.com/hat47x/tei-platform/actions/runs/42256518693)は**completed/success**（Go全体、gofmt、vet成功）。[#3031](https://github.com/hat47x/tei-platform/pull/3031)の未対応Source拒否も別HEADで成功。**Provider/Owner対象全集合の実Reader証拠、誤った`CoverageComplete`申告の拒否、CEC-02独立空証明の実確認、実Markdown Parser、Owner意味受理、末尾脱落MutantはNOT RUN**。この§18.10はdocs-only、#2514/#3030の担当Issue・実装コードに変更なし。

### 18.11 テスト導出の次の一手：Claim別Coverageの3反証を実受領に結び付ける（2026-11-07）

**先行成果との接続**：[台帳§8.11](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6773033710)の固定Sourceと、独立期待値Owner [#2514 CDE-Q05〜07](https://github.com/hat47x/tei-platform/issues/2514#issuecomment-6698037813)・Result Slice Owner [#2506 R05〜07](https://github.com/hat47x/tei-platform/issues/2506#issuecomment-6698048070)を使用する。#3051はmain統合済みの**局所Completionの未観測空否定修正**、#2901/#3017は**既知の合成Relation判別力**。いずれも今回の「全Owner対象・Required Predicate Field・Reader Completeness」の合格を代替しない。

| 実装を増やさず先に行う検収 | 最小の固定材料 | 採点するClaim／停止線 |
| --- | --- | --- |
| **P0 R05：同件数・違うOccurrence** | `L={tenant-a/3,tenant-a/9}`、合成`R=[tenant-a/3,tenant-a/3]`、P/Owner ScopeとSource固定 | Row count = 2でもdistinct対象は1件。全件Coverageの誤申告を**unqualified**と判定する。JOIN複製と実注文2件は別。既存の独立Expectedから実Rを生成せず、二つのSourceが共有する誤Query Filterの可能性も記録 |
| **P0 R06：正常なTOP 1** | 現main `postgres_scoped_orders.sql`、`scoped_postgres_integration_test.go`。Aで`$select=orderId&$orderby=orderId desc&$top=1`の期待`9` | **ページ表示**の成功と、対象注文`{3,9}`の**全件Completion**の未資格を両立。実PG Caseの実行には`TEI_WEB_SCOPED_PG_DSN`・隔離DB Opt-inとテスト非SKIPの証拠が必要 |
| **P0 R07：Rule列の未投影** | `L={A/3,A/9}`、P=`status=READY`、Consumerは`$select=orderId`。Fixtureでは3がOPEN・9がREADY | ID列の受領だけでPの結果を捏造しない。`status`の**内部Rule Read**が実際に必要なら、D/V/Q/Aの明示Mapping・現在Grant・実Read Footprintを#2506/#2586側で検査し、ConsumerへUnauthorized列を付加しない |
| **P1 実Readに結合** | 上の独立Q05〜07と同じOwner Population/Query/Source/Rule版、Tenant/Parent/Scope/as-of、Reader Result Slice、Snapshot、page-end、Field Presence/Disclosure、Actual Consumer | #1163の同一Source/Runtime/Provider/AttemptでSource/Ruleの**実消費**とExpected/Actualを結合。PGページの成功、Go evaluatorの成功、別CIの健康状態を一回の統合PASSに足し合わせない |

**実証に使う既存コードの確認**：現mainの `reference/web-api-contract/fixtures/postgres_scoped_orders.sql` blob `5113f47d131da7c6b00c7f00dd67fcd7298a53da`、`reference/web-api-contract/scoped_postgres_integration_test.go` blob `062fe0ab4262bb066d5a06090c21453e6d1cc73f`を直接取得した。後者の`$top=1` Caseは**既存Go Source上の正負期待**であり、今回の試行で実PostgreSQLを起動した証拠ではない。#2514 Q05・Q07は合成受入計画、Q06も**本統合AttemptとしてNOT RUN**。固定したSource/Expectedが観測されていなければ`not-run`、対象/粒度/Read cohort不一致なら`unqualified`、取得エラー・Schema/版不一致なら`unknown/needs-requalification`と記録する。

**機械化する場合の最小差分**：既存Scenarioに`case/step/required obligation`の参照を付け、既存Result SliceとReader Receiptの**比較表**で、`requested page completed`、`Owner target membership covered`、`required predicate fields observed`、`Source actually parsed/selected/consumed`を別列にする。`Satisfaction`値そのものと**Claim qualification**を同じEnumにしない。既存Driver v0が`satisfied/not-satisfied`を持つだけなら新しい`unknown`を無理に追加せず、資格の不足はEvidence側で表現する。非出力依存（WHERE、JOIN、NOT EXISTS、Ruleの隠れたField）をConsumer投影だけから推定せず、明示D/V/Q/A参照を使う。

**再資格・権限境界**：Tenant/Parent Relation occurrence、Owner集合・Ruleの粒度、`status`等の必要Field、Query filter/order/top、Source/Plugin/Manifest/Operator版、Provider Grant/RLS、Snapshot/selection epoch、Consumer用途が変われば**影響Claimだけ**を再評価する。`.tei.md`形式移行ではIndex反映と実Parser/Meaning消費を分ける。OKF deterministic Attester/`verified`もExpectedの独立性・Source消費・実Read集合・Host Grant/Owner Acceptanceを証明しない。

**責務と次の停止線**：#2811は検収順・証拠台帳リンクのみを所有。独立期待値の判別力=#2514、Case集合=#1180、Result Slice=#2506、実Query/Provider=#2586/#2622、同一改訂Packet=#1163、開示=#2516、Source移行=#3030。新しいRuntime/Runner/独立期待値Store・Owner認可は追加しない。**当該R05〜07の新テスト、PG実行、CallerからCompletionへの実連結、Owner受理、OKF実Attestationは今回NOT RUN**。

**統合ブランチの再適用条件（2026-11-07照合時）**：本Draft #2811 は現mainと大きく分岐している（照合時点のGitHub Compareでahead 57／behind 925）。mainの同名計画書は別Blob `722e1711e74c08ca967efb9554325a2ed9891f79`、本台帳ファイルはmainに未配置。したがって、Draft内で章を増補・旧CI成功を確認しても**現在mainに計画が採用されたことにはならない**。マージ前にmainの現行Source/文書・既存Owner責務との重複を再照合し、影響した本文だけを新しいmain基線へ移植／コンフリクト解決する。古い成果の一括上書きやDraft 57コミットの無条件合算を禁止し、新しい統合HEADでDiff・リンク・必要CI/必要Obligationを再検収する。これは**統合作業の未実施条件**であり、今回rebase/mergeや統合後のテストを行ったものではない。