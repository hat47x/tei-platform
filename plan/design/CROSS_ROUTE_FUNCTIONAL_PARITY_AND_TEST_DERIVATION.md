# 経路間の業務機能完全性とScenario由来E2Eテスト

- Status: design contract / Reference proposed; implemented cross-route generator: **not yet verified**
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

## 3. Source of truthと生成パイプライン

```text
Accepted Application / Driver Scenario / Policy / Lock
             │  same baseline B / semantic revision R
             ├──> Scenario inventory + typed operation binding
             │        └──> shared semantic assertions & fixture contract
             ├──> CLI adapter / test execution ──> qualified CLI evidence
             │
             └──> Web adapter mapping ──> generated Web E2E ──> actual browser evidence
                       │
                       └──> target-specific auth / transport / UI / a11y assertions
                                  ↓
                  compare business results / effects / Observation
                  + Target-specific evidence + explicit gaps
```

既存Driver Scenario、Semantic Operation、Typed Source／Evidence、Shared Acceptance、Application Composition、Capability Bindingを消費する。argvログをWebのクリック列に変換したり、クリック録画から新しい業務意味を推測したりしない。共通テストは独立した業務DSLや第二のDriverを作らず、既存Scenarioの**テストProjection／派生Evidence**とする。

**最小Scenario record（検証候補。固定production schemaではない）**:
- scenario identity、Accepted Application／Canonical fingerprint、baseline `B / R`、Scenario source
- input fixture／precondition、operation参照、step順序と分岐、期待Validation／Auth判定
- Business Result／named Outcome／Execution Error、観測対象とexpected state delta
- effect classification（read-only / reversible / irreversible / external / unknown）、fixture reset／cleanup／compensation
- expected Subject／scope／effective authority、Policy revision、Resource条件
- mapping identity/version、Target build `Tcli / Tweb`、runtime/provider、generator version、executed evidence links

期待値はAccepted meaningまたは独立した検証fixtureに由来し、CLIの実際の出力を盲目的に正解として焼き付けない。CLIの検証済みResultは共有expectationとの照合証拠であって新しい正本ではない。

## 4. Generator / Adapter責任境界

- **Shared semantic fixture**: 同一入力・前提・業務Operation・assertion／state-delta観測を解決する。権限を発行しない。
- **CLI binding**: Typed Operation → argv/options/stdin またはin-process typed invocation、Result／Diagnosticの抽出。CLI専用Validationを実装しない。
- **Web binding**: Typed Operation → page/route、stable semantic locator（UI Kit等が提供する明示binding）、入力・送信・確認。Web固有画面を第二業務モデルにしない。
- **Target-specific assertions**: Web login/session、origin/CSRF、navigation、focus／accessibility、transport error、visual evidence、Desktop IPCなど。共有業務assertionと区別して保存。
- **Comparator**: Business Result、Outcome、Error、Observation／副作用を同じ意味に正規化して差分判定。複数経路の実行条件に差があれば「比較不能」とする。異常を期待値に合わせて覆い隠さない。

GUI配置変更時はWeb bindingだけを修正でき、Accepted Scenarioのmeaningとexpectationは不変であるべき。Target-specific custom UIも標準Projectionと同じOperation contractへ結びつける。

## 5. Fail-closedと安全な再現

- 未対応Operation／必須branch／locatorの曖昧さ／欠落mapping → **generation failure or unresolved**（skip／空testは禁止）
- 旧Revision、異なるLock、Target build不明、Resource・Policy mismatch → **stale/incomparable**
- Denied操作をCLI管理権限で強行 → **authority mismatch**（CLI PASSにしない）
- side-effect Confirmationがない、Observationが取得できない → **not observed**。Command受付をeffect成功にしない
- 成否不明の外部送信、rollback未確認 → **unknown**。再試行による重複effectを作らない
- Shared assertionがないstep／期待値が実結果から自己生成 → **untested**
- E2E非実行なのに生成されたという事実だけ → **generated, not executed**
- screenshotやDOMのみ合格、business error不一致 → **functional failure**
- CLI-only partial slice／read-only fixtureのみ → **coverage incomplete**（全業務適合を宣言しない）

Executionは隔離されたFixture/Disposable DB、認可済みTest Resource、偽装していないSubject/Callerを使う。外部実効果は慎重に扱い、独立Observationのない場合は不確定のまま記録する。

## 6. 最初の実証と受入

#1180では`order_entry`等の**実在しAcceptedな**Scenarioを一つ選ぶ。最初はread-only Lookup/Validationの正負、次に認可を伴うCommandと独立Observationを加える。最初の生成TargetはWeb。Desktop／Mobile等はこの契約を再利用できることを確認してから個別Adapterを実装する。

受入には、(a) Scenario inventoryの全対象OperationとCLI/Web bindingに欠落がない、(b) **同じScenarioから生成された**CLI検証とWeb E2Eを実際に別経路で走らせ同じ業務結果を確認、(c) Validation否定・Auth拒否・Execution Error・欠落mappingの負例がfail-closed、(d) 同じRevision/Lock/Policy Contextで比較可能、(e) Web固有Auth／UI EvidenceがCLI PASSと独立、(f) trusted clean checkoutのactual exact-head test/health receipt、を要求する。

#1155／#1165のC1は限定されたCLI read-only discovery／inspect sliceでよい。初期scope制限はApplication公開Targetの永続的な機能欠落を正当化しない。#1163の横断Evidence Packetは本Generatorの成果を取り込み得るが、Generator自身が新しいEvidence Authorityにはならない。

## 非目標

- CLI単一happy-pathのPASSでApplication全体のConformanceを宣言すること
- Web DOM／UI録画を業務Meaningへ逆採用すること
- 全Targetの共通UIを強制すること
- Browser／Desktop native Test frameworkをCoreへ固定すること
- 外部Production effectの自動再実行、Credentialの記録
- 第二Policy Engine／第二Driver／第二Resolver／第二Acceptance Authorityの導入
