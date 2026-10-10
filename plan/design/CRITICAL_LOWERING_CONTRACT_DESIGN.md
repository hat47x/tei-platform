# Critical Lowering Contract — 宣言からTarget実現への段階的接続（候補）

- **状態**: 評価待ち。2026-10-10時点のReference pressureであり、採用済みproduction API、実Targetの動作証拠ではない。
- **管理**: [#1121](https://github.com/hat47x/tei-platform/issues/1121)、横断依存とGateは[#1119](https://github.com/hat47x/tei-platform/issues/1119)。
- **目的**: 既存Parser / Semantic Acceptance / Driver / Capability Resolver / Build Artifact契約を置き換えず、境界を越えるときに失ってはいけない最小情報を固定する。
- **参照実装**: [reference-critical-lowering](../../tools/reference-critical-lowering/README.md)。同ReferenceのID・fingerprint・Trust callbackは合成入力であり、実権限を主張しない。

## 1. 採用すべき境界とowner

| 境界 | 必要な入力 → 出力 | 正本・権限のowner | 失うと危険な情報 |
| --- | --- | --- | --- |
| Source → Parse | Source artifact digest、Language ID/version、Reader範囲 → syntax / source span | #988 / #1018 / #1017 | parser選択、元source、parse診断 |
| Parse → Candidate | syntax / Operator・Semantic Set → typed candidate + source index | #988 / #1120、Semantic Extension | 型、cardinality、NULL規約、対象の識別、effect、Context Slot宣言 |
| Candidate → Acceptance | exact candidate digest + Human/authorized decision → authoritative sourceの再読込とAccepted snapshot | Shared Acceptance / #1120 | 未受理Proposalと受理済み意味の区別、原source・判断根拠 |
| Accepted → Compose | snapshot + Application Profile / explicit wiring → required Contract群 | #1016 / #951 | Scope、Contractのexact版・fingerprint、選択者と適用対象 |
| Compose → Capability resolve | published Contract、Provider、Conformance、required Context Slot declaration → existing Capability Resolution Lock参照 | #951 / #976 / #1118 | Providerの真正性、required slotの指紋、実装の作用域 |
| Resolve → Target-neutral intent | Accepted meaning + Contract / effect宣言 → Contractを呼ぶ最小plan | Driver / Semantic Capability | business resultとexecution errorの区別、read / write / invoke effect |
| Neutral intent → Target realization | target features、Host、exact provider/artifact → target-specific step | #259 / #997 / #1043 / #943 | unsupported effect、Target差、runtime/build依存の区別 |
| Realization → Build artifact | exact build inputs / builder / target → existing BuildArtifactLockとdigest | #943 / #979 | provenance、bootstrap seed、artifact identity |
| Artifact → Invocation | Hostが解決する現在のContext Slot実値、権限、Resource / Lease → Providerへの呼出し | #1118 / #976 / #1043、既存Driver | current authorizationの失効、認証情報の非混入、拒否と不明の違い |

これは固定的な「全件必ずコンパイルする」pipelineではない。Interpret実行・既製UI・既存Go実装を用いる場合も、通過した境界について必要なAuthorityとTraceだけを守る。**Sourceから受理を自動で推論しない**こと、および**Contractの存在からRuntime Authorityを推論しない**ことが不可欠。

## 2. 最小のversioned交換形式（Reference v0）

新しいUniversal IR / 重複Lockは設けない。具体的なDTOはReferenceに限り、productionへ直接コピーしない。

- `tei.lowering-candidate/v0`: Source Ref、Language Contract Ref、三視点などのtyped subject Ref、required Contract / scope / effect / Context Slot宣言。source spanやSource Index本文は持たず、正本へ参照で戻る。
- `tei.lowering-acceptance-link/v0`: candidate digest、Accepted snapshot Ref、trusted Acceptance receipt Ref。参照可能であることは受理権限の証明ではなく、現行のauthoritative Shared Acceptanceで照合する。
- `tei.lowering-neutral-plan/v0`: Accepted snapshot Ref、Origin（source / language / candidate digest / Acceptance receiptへの参照）、Contract / effect / required slotのintent。Target、Provider、principal、credential、Environment binding実値を入れない。
- `tei.lowering-witness/v0`: OriginとAccepted snapshotを引き継ぎ、selected Provider、Conformance、既存Capability Resolution Lock、既存Build Artifactの**参照**、Runtime dependency、Target step。これはTarget-bound witnessであり、新しいArtifact Lockでもない。
- `Ref`: `{id,version,fingerprint}`のexact照合。major番号一致のみから互換性を推定しない。Referenceで使う`sha256:...`文字列は合成ラベルで、実digest検証・発行者の真正性は実装していない。

**Context Slotの宣言**はCandidate digestとAccepted Contract / Provider conformanceへ束縛する。slotが欠けたとき「context不要」として実行することを禁止する。一方、session / verified principal / access refなどの**実値**はSnapshot / Lock / Plan / Artifact / Cache Keyに記録しない。毎回trusted HostでbindingとAuthorizationを検査し、Providerへ必要なslotだけ渡す。Auth engine / Resource Brokerの第二実装を作らない。

`Trust`は外部システムへの必須境界であり、`Accepted`、`Published`、`Conformant`、`ContextBound`、`Authorized`を呼ぶ。Reference testのstubがtrueを返すことは**外部AuthorityのEvidenceではない**。

## 3. error / evidence / cache規約

正規のBusiness Result、`not-satisfied`、Execution Error、Unknown、Development Diagnosticは混ぜない。Boundary errorは例えば次を区別する。

| fail-closed診断 | 破れた契約 | 再試行・対応の方向 |
| --- | --- | --- |
| `unaccepted-meaning` | candidate digestとAcceptance receipt / snapshotの不一致 | Authoring / Shared Acceptanceに戻る |
| `missing-capability` / `missing-wiring` | 明示解決先がない | #951 / #1016 の選択へ戻る |
| `incompatible-version` / `contract-drift` | exact identity、schemaまたはfingerprint不一致 | Contract / Providerを再解決 |
| `ambiguous-provider` / `ambiguous-wiring` | 複数候補を暗黙選択しようとした | 一意なwiringを要求 |
| `invalid-context-declaration` / `conformance-or-context-drift` / `unverified-provider` | Scope・Conformance・Context Slot・publisher Authority不備 | contract ownerに戻りfix |
| `unsupported-target` | Targetがeffect / realizationを提供できない | 明示的に別Target案へ（意味を変えない） |
| `missing-execution-context` / `authorization-denied` | 実行時のHost slot binding / current grant不備 | **Providerを実行しない** |

入力構造が同じならdigestとdiagnosticsを決定論的に再現する。暗黙のProvider列挙順で優先度を変えない。Evidenceには`source/accepted/ref/lock/artifact`の参照と、実際に行った検査のissuer・scope・時点・入力digestが必要。単なるcontent hashはAcceptanceやConformanceを証明しない。

段階別の最小再評価鍵（Referenceの`Keys`）:

1. **Interpret** = exact Source + Language。Source moveだけの意味同値性はこのReferenceでは判断せず、#1120 / Source Indexへ委譲。
2. **Accept** = Candidate digest + Acceptance link。外部のAuthority撤回はcache有効でも別途current検証が必要。
3. **Resolve** = Accepted identity + 選択されたContract / Provider / Conformance / Resolution Lock等。無関係なPlugin追加では無効化しない。Provider切替はここから下流を無効化する。
4. **Target** = Resolve + Target features + Target realization / artifact ref。Targetだけの条件変更は上流のAccepted meaningへ影響しない。ただし別Providerが必要ならResolveから再実行する。
5. **Build** = Target + builder / build-only input refs。build-only依存の変更でRuntime dependencyを追加しない。現Referenceの`Keys`はbuilderの代表refのみで、#943のBuild Plan全閉包の代わりにはならない。

current AuthorizationとHost Bindingをキャッシュしたまま呼出しを省略してはならない。またContext依存resultは単純なsemantic request digestだけで横断再利用しない。

## 4. 三視点の`order_entry`横断Trace

既存のReference sourceから確認できる事実と、今回の合成pressureを分ける。

| 三視点 | existing source | loweringで保持する参照 |
| --- | --- | --- |
| Business / Function操作 | `reference/workspace/applications/order-entry/patterns/submit-order.tei` | `order_entry.submit_order` → `sales.order.submit` |
| Data | `reference/workspace/semantic/sales/order.tei` | `sales.order` / `sales.order_line`、関係`sales.order.lines` |
| Function / Validation | `reference/workspace/semantic/sales/validation.tei` | `sales.order_line.quantity_positive`、`sales.order.lines.line_number_unique` |

既存`tools/reference-first-web-target`は実際のDriver / `order_entry`を用いてread → validation → authorization → command → separate observationへ進む実装入口である。今回のGo testは**そのWeb TargetやDriverを起動していない**。同じ名称で三視点Subject Ref、read/write Capability、required Context Slot、TargetごとのProvider切替を合成し、**段階間で情報が消えないか**だけをテストする。二つのLanguage ID（`tei.language.dsl` / `tei.language.sql-projection`）もparserを実行した証拠ではなく、schema互換のpressureである。

第二Target（Desktop）は別Providerへ切り替えてもNeutralPlan snapshotを保てることを合成データで比較するのみ。UI、バイナリ、Host権限、Semantic / Functional / Visual Conformanceのactual Evidenceは未取得。

## 5. Gateと統合条件

現段階は**契約候補 + isolated reference source / test**。実装をproductionへ持ち込む前の検証順序は次とする。

1. #1120のStable ID / typed refを接続し、Source Indexとaccepted snapshotを本物のRevisionで突合する。Native SQLとTEI DSLのsource ownershipをSource実値で確認する。
2. #988 / #1018のLanguage pluginで2 producerのCandidateを実際に出す。Parserのunknown / 版不一致 / 未宣言fallbackを負例として追加。
3. #951 / #1016 / #1118のexisting resolver / explicit wiring / Host Contextを使い、合成Trust stubを排する。missing Capability、version mismatch、authorization deniedのfail-closed証跡を実Hostから収集する。
4. existing`order_entry`DriverとWeb Targetへtracing adapterを足し、Source→Acceptance→Contract→Runtime result→separate Observationの実走査を再現する。意味をWeb側でauthorしない。
5. 第二Targetで同じsemantic baselineに対するSemantic / Functional / Visualのactual差分を取る。#943のBuild artifact契約を参照し、別Artifact Lockを増やさない。
6. exact PR headとcurrent mainを記録し、trusted clean checkoutの`tools/repository-health-check.sh`全PASSを得たときだけレビュー・統合を検討する。**Draft PRは統合しない**。

上記は#1119のG2/G3に必要な作業を具体化したものであり、Reference testが通っただけでG2/G3を達成したとは扱わない。

## 6. #1119 C3 / S05: Data Contractと実行時Authorityを別の根拠にする

#1119 のV04→V05の接続では「受理済みの意味」だけでは保存を許可できない。対象`order_entry`の既存Sourceには、案件からDatabaseへ次の**logical requirements**が宣言されている。

```text
order_entry.sales  -> sales_core
  sales_core.order       ^4
  sales_core.order_line  ^1
order_entry.master -> master_data
  master_data.customer   ^2
```

これはEnvironmentの`sales_core.dev`等の現在の物理Schema Versionそのものではない。Application Release RevisionとDatabase schema versionの数値一致を要求するのではなく、Database側が所有するCompatibility判定が必要である。

Referenceでは`DataNeed {database, contract, versionConstraint}`をAccepted Candidate / Neutral intent / Target-bound stepまでコピーする。具体的なDatasource alias → Environment / Instance / Provider選択、およびCurrent Physical Schemaの取得・比較は既存`environment`・`driver`・`database`・`realization`へ委譲し、これらの実値をCandidateにAuthorしない。Target stepへ移すのは**必要Contract**であり、Schema互換を検証したという虚偽の決定ではない。

Runtime preflightで独立に照合すべき根拠を明示する。

| 検査 | 固定する宣言・Evidence参照 | 実行時のCurrent検査 |
| --- | --- | --- |
| Meaning | exact Candidate digest → trusted Acceptance → Accepted Snapshot | 呼出し対象が当該snapshotに適合すること |
| Capability / Target | exact contract/provider/conformance/Resolution Lock/Build Artifact参照 | 現在の実行用bindingとartifact選択が有効であること |
| Data | logical `DataNeed`のdatabase/contract/version constraint | 対象Datasource → Instanceの現在bindingとSchema互換判定 |
| Execution Authority | required Context Slotの契約だけ | 現在のslot binding、Principal、Permission / Resource / Lease等 |

`Trust`への`ResolutionCurrent(plan)`、`RuntimeBindingCurrent(step)`、`DataCompatible(step)`、`ContextBound(step)`、`Authorized(step)`は上表の分離を試す**Reference callback**であり、productionのDB introspector、permission checkerを再実装するものではない。少なくとも次の負例は副作用を呼ぶ前に停止する。

- **S05-A** Preview後にCapability / artifact / runtime Provider bindingが取り消された → `runtime-binding-unavailable`。
- **S05-B** Database instanceが再接続され、実SchemaがAccepted Contract requirementに非互換となった → `data-contract-incompatible`。
- **S05-C** 必須Host Context Slotが解決できなくなった → `missing-execution-context`。
- **S05-D** 権限が取り消された → `authorization-denied`。
- **S05-E** 過去のResolution Lockに基づくplanのまま実行しようとした、またはcallerがeffectを書き換えた → `stale-or-untrusted-resolution`。

同じ認可の失敗でもApplication-level business outcome（業務規則による拒否）とHost permission gate failure（実行権限不足）は混同しない。Retry、business completion、Observationも別の契約。上記テストは合成callbackで否定経路だけをpressureしており、実EnvironmentでのS05 actual evidenceではない。

**停止条件:** 受注明細を実際に保存する際の正規Driver経路、Datasource Resolver、current Schema互換性証跡、Runtime Host Grantを同時に示せるまでは、このReferenceをproduction enforcementとして採用しない。実施した検査の根拠・時点・scoped identityが提出できない場合はUnknown/Unverifiedとして拒否し、static fingerprintのみから「現在も許可」としない。

## 7. Source dependency witness・Datasource alias・Build-only cache

### Authoring→AcceptanceのSource fingerprint

先の合成Referenceが持っていた`Candidate.Source`（Application fileのみ）では、PatternやSemantic定義の変更がSource側の変更として表現されず、古いAcceptanceを誤って再利用するおそれがあった。追加した`SourceInputs []Ref`は、解釈に入力した依存SourceのID/Version/Fingerprintを明示する。Authoring→Accepted Candidateのdigest、Target-neutral Origin、Target-bound PlanのOriginまで欠落させない。重複・未fingerprint・非決定的な順序は拒否し、Source Indexを読んだintegration testで参照したSource fileの**実SHA-256**をBindingする。

ただし、現時点の入力集合は受注のApplication/Command Pattern/Sales Data/Validationの代表Sourceであり、Productionに必要な**全依存Source closure**が網羅されたという意味ではない。完全な依存グラフとAcceptanceの正当性はWorkspace Loader/ParserとTrusted Acceptance ownerに帰属し、Reference側で署名や承認を発行しない。CandidateのfingerprintはSource Indexから取得可能な実値に置き換えられるが、それをTrustの自己申告だけで正当化してはならない。

### Datasourceと物理Instanceを混同しない

`DataNeed`にはApplication-local `Datasource` aliasを追加した。`sales`・`master`は実Source上の明示名であり、`sales_core.dev`・`master_data.dev`という実行環境固有のDB instanceとは異なる。Alias + Database + Contract ID + Compatible Majorの組がSourceと一致することをintegration testで照合する。これにより、複数aliasで同一Databaseを参照する案件でも、Contractだけから物理接続を推定しない。

### Build-only dependency変更の再生成

Reference `Keys`は`Builder`に加え、既存Build Artifact ownerが解決した**選択済みBuild-only dependency集合**をartifact stageのkeyへ明示的に受け取る。Build-only depsの追加・version変更は`Build`だけ無効化し、`Interpret`/`Accept`/`Resolve`/`Target`は維持する。並び順を正規化し、Plugin registration orderだけの差で再ビルドしない。逆に、この集合はTarget-neutral intent・runtime Stepsには保存せず、Runtime depsやcurrent grantと混同しない。

これらは`/v0`候補の境界実証であり、既存の`#943` Build Artifact型、`#951` Resolver型、`#1118` Context型へ新しい本番用重複データ構造を導入する提案ではない。

### 正確な残作業

1. Source Indexから完全な依存closureを決定しTrusted Acceptanceへの受理証跡を結ぶ。
2. Capability Resolverと既存Lock/Artifactの**実IDとfingerprint**を使ったBindingを追加する。
3. `order_entry` Web HTTP実行でactual Target trace（Auth/Validation/Business Result/Separate Observation）を獲得する。
4. 第二Targetは意味の不変条件、Capability非対応の失敗または具体的実行差分を実証する。
5. exact-head clean checkoutでGo test・gofmt・repository-health-checkを実行し、証跡を残す。現時点でこれらは未実行。

## 8. 既存Web Target実行レーンの準備と証拠区分

新たなWeb実装やHTTP Driverを#1121へ複製せず、既存`tools/reference-first-web-target`の実行バイナリを対象とする、opt-inのBlack-box smoke testを新規Reference内に追加した。試験環境は`TEI_LOWERING_WEB_SMOKE=1`でのみ起動し、信頼する**実CheckoutのHEAD**を取得し、Gitの前後clean状態を照合する。RuntimeはLoopback ephemeral portで起動し、持続的なproduction serverとは別とする。

試験する三段階は次のとおりである。

| 段階 | 実装済みテストの観測契約 | 証拠の限界 |
| --- | --- | --- |
| Source→Canonical→Composition | `workspace.Load`、既存DriverのCommand/Browse/Reference/Header-Detail、Source Indexの実Fingerprint、ApplicationのDatasource/DB要求、Environment Instance binding、既存`plugin.Catalog.ResolveEnvironment` | current DB Schemaやproduction Capability Authorizeの実証ではない |
| Canonical→Web Target | 既存Webバイナリを実際に`go build`しHTTP `/manifest.json`を読み、Application/Pattern/Data/Command IDをReference意味主体と照合 | Synthetic `Acceptance`を実Webが消費したことにはならない |
| Web runtime scenario | `POST /scenario/submit`で認証→3件のValidation→Authorization→Command Result→別Observation、認証拒否、Command実行エラー、Credential非露出を観測 | 既存fixtureとsynthetic Observationであり、外部DBへの永続書込や実Principalの権限ではない |

以上は**真正のHTTP実走査を再現できるテストソースを作成した**段階であり、Tool実行環境にtrusted完全Checkoutがないため、実走査結果をまだ得ていない。既存Web Target自身の既設単体テストと取り違えない。

### Contextと実行権限の結合で残る危険

現状のReferenceで`AuthorizeInvocation(plan, trust)`を呼んでも、その結果を実HostがExecution Dispatcherと同一のrevocation/lease boundaryで確認したわけではない。実権限、datasource instance、Plugin、artifactが検査と実行の間に変わるTOCTOUは、独立したProduction側の課題である。ここへ新しいstandalone Authorize APIを追加するのではなく、#951/#1118の既存Host/Capability Invocationを拡張・検証する形に寄せる。

またPlugin環境解決`plugin.Catalog.ResolveEnvironment`と、#951のGeneric Capability Contract Resolverは別の責務である。前者の成功をもって後者のConformance・Contract fingerprint・実Artifact/Lockの証明とはしない。

### Gate判断

新Web smoke testは通常の`go test`からはskipされ、明示実行でのみバイナリをbuildし起動する。これを含め、focused/full testが**実行済みであるという記録は存在しない**。次に必要なのはtrusted checkoutでのSmoke実行・Go formatter・full Health Check、actual Shared Acceptance/Generic Resolver接続、および二つ目のTargetからの独立した実測証拠である。
