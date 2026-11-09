# 経路間の業務機能完全性とScenario由来E2Eテスト

- Status: design contract / Reference proposed; implemented cross-route generator / persisted Driver Scenario compiler: **not yet verified**
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

既存の`internal/driver/final_gate_b_scenario_test.go`、`gate_b_ui_less_scenario_test.go`等は**Goテスト内で組み立てるtest-local Reference**であり、現時点では全Targetへ生成可能な永続Scenario資産ではない。まず代表例から、既存DriverのOperation／Result／Observation／Providerを参照する**独立保存・版管理・再読込可能なScenario表現**と、その解決・検証を成立させる。Goテストを機械的に逆解析して無検証な業務定義へ昇格しない。新しい第二Driverや、独立した業務Semantic DSLを作らない。記述形式と恒久schemaは最小のactual consumerで検証してから確定する。

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
