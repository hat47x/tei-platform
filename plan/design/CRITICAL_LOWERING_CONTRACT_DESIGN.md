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
- `tei.lowering-neutral-plan/v0`: Accepted snapshot RefとContract / effect / required slotのintent。Target、Provider、principal、credential、Environment binding実値を入れない。
- `tei.lowering-witness/v0`: selected Provider、Conformance、既存Capability Resolution Lock、既存Build Artifactの**参照**、Runtime dependency、Target step。これはTarget-bound witnessであり、新しいArtifact Lockでもない。
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
| `conformance-or-context-drift` / `unverified-provider` | Scope・Conformance・Context Slot・publisher Authority不備 | contract ownerに戻りfix |
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
