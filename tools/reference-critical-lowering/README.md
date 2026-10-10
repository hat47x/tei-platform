# Critical Lowering Contract Reference (#1121)

評価待ちの合成contract pressure。**TEIのproduction Compiler / Driver / Plugin Resolver / Build Artifact契約は変更しない。**実行先のバイナリやWeb UIは生成しない。

## 検証すること

- Candidate → externally verified Acceptance → Target-neutral intent → explicit Application wiring → exact Capability / Provider / Conformance / Lock参照 → Target-bound witness。
- required Context Slotの宣言がAccepted contractからProviderへ失われないこと。**実値**は現在のtrusted Hostが注入し、planには置かない。
- `build-only`依存がruntime planへ入らないことと、必要なeffectをTargetで提供できないときにfail-closedすること。
- CandidateやProviderの順序・Target・builder変更が、関係のない上流段階を不必要に無効化しないこと。
- current authorization / slot bindingは、loweringが成功してもInvocationごとにHostから再確認すること。

## 実行

Repositoryのクリーンなcheckoutから:

```bash
go test -count=1 ./tools/reference-critical-lowering
go run ./tools/reference-critical-lowering
```

これは局所的なpressureであり、正式な統合判定に必要な全件検証とは別。統合前にはexact headで必ず以下を実施する。

```bash
tools/repository-health-check.sh
```

## 検査用データの位置付け

`main_test.go`は既存`order_entry`のPattern / Data / Capability名に結び付けているが、contract ID、provider、Context Slot、receipt、LockとArtifact参照は**合成値**である。stub Trustは全検査の実Authorityではなく、これらが欠落したときのboundary処理を検証するための置換物。二つのLanguage IDは2種類の入力構造のpressureにとどまり、実Parserを実行しない。

本Referenceが実装していないもの: Accepted sourceのreload、実Pluginの解決、current permission policy、Host sidecar/context injectionの実値、secure BuildArtifactLockの内容照合、Target実行、Actual Observation、second-Target executable conformance。

ステージ契約と責務の所在、選択的再評価の前提、実Consumerで検証すべき残作業は[Critical Lowering Contract Design](../../plan/design/CRITICAL_LOWERING_CONTRACT_DESIGN.md)を参照。

## 受注明細の実資産との照合

`workspace_trace_test.go`は、既存の`reference/workspace`を実際の`workspace.Load`で読み、`driver.NewResolver`から`order_entry`のCommand、Browse、Header/Detail、Referenceを解決する。Source Indexが`submit-order.tei`へ戻せることと、合成Candidateが要求する3件のlogical DB ContractがSourceで宣言済みであることも照合する。

これは**実Source→Canonical→Driver解決の局所Integration Test**であり、actual Web UI/HTTP実行、Permission/Schema互換性の本物の検証、Physical DBへの問い合わせ、第二Targetの観測とは区別する。Test source追加だけではPASSと表明しない。

実行時判定の変更例（#1119 S05）は、同じPlanに対して古いPreview時の判断を使い回さず、各回`ResolutionCurrent`、`RuntimeBindingCurrent`、`DataCompatible`、`ContextBound`、`Authorized`を分けて検査する。これらは現在、テスト用stubであり、DriverやHostの真正な決定ではない。

## 追加した依存・cache境界（2026-10-10）

- 既存Source IndexにあるApplication、Command Pattern、Semantic Data、ValidationのArtifact fingerprintを読み、`Candidate.SourceInputs`と`Origin.SourceInputs`へ保持する。**完全なSource closureの決定は未実装**であり、実際に引用した代表SourceのTraceだけを検証する。
- `DataNeed`は`Datasource` alias、Database、Contract、Compatible Majorを保持する。Host側で解決される物理DB instance/connection/sessionは含まない。
- Build-only dependenciesは既存Build ownerの明示引数として`Keys`へ渡し、`Build`段階のhashだけに加える。`Plan`には渡さず、Runtime Authorityの対象へしない。
- Previewでの成功を認可とみなさず、古いLock / current Provider / Data Compatibility / Host Context / current grantのそれぞれで停止する負例を固定する。
- `AuthorizeInvocation`はGateの**順序を示す合成Reference**であり、独立して返したboolや実行許可を後続のProviderへ再利用する本番APIではない。実装では既存Hostの現在のAuthorityとdispatch境界を再利用し、検査後の失効と実行開始のTOCTOU、失敗後の副作用/再試行を別途検証する。

## Opt-in: 既存Web TargetのHTTP Black-box実行

[web_target_smoke_test.go](web_target_smoke_test.go)は、既存`tools/reference-first-web-target`を**実際に`go build`**してLoopback（Ephemeral Port）で起動し、HTTP APIに対して次を検証するためのopt-in統合テストである。

1. `/manifest.json`に、`order_entry`のCommand/Browse/Reference/Header-Detailの意味参照、対象CheckoutのTarget Revisionが保たれていること。
2. `POST /scenario/submit`の正常系列でAuthentication → 3件のCanonical Validation → Authorization → Command Business Result → **別の**synthetic Observationが通過すること。
3. 認証拒否がBusiness DenialとしてCommand/Observationへ進まず、Credentialがレスポンスへ漏れないこと。
4. Command実行エラーが通常のBusiness Result/Observationと混同されず、HTTP 422と失敗Stageに残ること。

Webバイナリの起動まで含むため、通常の局所Unit Testでは**skip**し、明示指示時に実行する。exact HEADの一致と実行前後のGit cleanを自動検査する。Repository rootで:

```bash
TEI_LOWERING_WEB_SMOKE=1 go test -count=1 -run '^TestWebTargetHTTPBlackBoxTrace$' -v ./tools/reference-critical-lowering
```

このテストは、**実Sourceを読み込む既存WebのHTTP実行**を検証する一方、Loweringの`Trust`がProduction Host/Shared Acceptanceを通過することを検証しない。HTTPの成功を「外部実DBへ保存した」「二つの実DSLからAcceptedされた」「第二Targetを実行した」と読み替えない。Serverはfixtureを使用し、観測も合成。実行していなければ成功証跡は存在しない。実行の際はexact checkout SHA、Go version、pre/post clean、Exit Code、主要Stageの観測を別途Evidenceとして保存する。

## 横断E1証拠で再利用する既設テスト

#1119の検討により、実際の`Source R0→candidate→review→accept→reload`と既存Driver/Webへの意味反映は、すでに以下に存在することを確認した。#1121内に第二のAcceptance実装やfixture injectorを作らず、trusted checkoutでこれらを呼び出して証拠を結ぶ。

```bash
go test -count=1 -run '^TestPracticalRuleChangePreviewsAndAcceptsBothMeaningAndApplicationSelection$' ./tools/reference-shared-acceptance
go test -count=1 -run '^TestPracticalBusinessRuleChangeReachesDriverAndWebWithoutTargetRuleList$' ./tools/reference-first-web-target
```

**実装経路・証拠階層の差**: 既存`TestPracticalBusinessRuleChangeReachesDriverAndWebWithoutTargetRuleList`はscratch Workspaceを受理してDriverと`httptest`のWeb handlerを検査する。今回追加した`TestWebTargetHTTPBlackBoxTrace`は**running-server**だが基線fixtureのままであり、scratchで増設した`unit_price_positive`の負例をrunning-serverで確認しない。両者を組み合わせて「新規ルールのrunning Web拒否は観測済み」と報告することはできない。この差は#1119 E1で別々に報告し、実演証拠の完成は該当Consumer Laneに委ねる。

## Exact-head統合検証の手順

通常のTest・既設Shared Acceptance/Driver/Web Test・opt-in running HTTP Test・全Repository Healthをまとめて検査する[qualify.sh](qualify.sh)を追加した。**PRマージを行うスクリプトではない。** 信頼する完全Checkoutの`origin/main`を事前に更新してから、PR headとmainの40桁SHAを明示して実行する。

```bash
EXPECTED_HEAD="$(git rev-parse HEAD)" \
EXPECTED_MAIN="$(git rev-parse origin/main)" \
bash tools/reference-critical-lowering/qualify.sh
```

スクリプトは実行前後にHEAD、origin/mainのSHA、clean状態を検査する。また**mainが受検HEADのancestorであること**を要求する。古いbranchをそのまま検証して「現在のmainへの統合準備完了」と誤判定しないためである。Formatter検査、`go vet`、局所Unit/Integration、既設Acceptance/Web contract、独立serverのHTTP Smoke、`repository-health-check.sh`をすべて実行する。ただし、ここで得る結果はlocal test evidenceであり、current Host Authority/DB Schema/独立Observation実測・第二Target・親IssueのGate審査の代替ではない。

Source側依存の`SourceInputs`は、プロデューサがClosureを検討したことを表す**非nil**リストである。値`[]`はそのSource自身に外部依存がない正常なケース、`null`/未報告は検討していない状態として拒否する。実WorkspaceのCandidateには代表的な依存を登録しているが、完全集合の証明には依然としてProducerの真正な閉包計算が必要となる。

### Test Evidenceの偽陽性対策

`go test -run`はテスト名が一致せず**0件実行**でも終了コード0となる。また`t.Skip`したテストも通常は成功扱いとなる。このため`qualify.sh`の必須シナリオ3件は、プロセスの終了コードだけでなく、`-v`ログに対象名の`--- PASS: ...`が明示されていることを確認する。命名の変化・無実行・opt-inのskipはQualification失敗となる。

シェルのPASSマーカー判定部分だけについては、ローカルの独立した小型fixtureで「0件実行」「SKIP」「PASS」を模擬して想定どおり判別できることを確認した（`GUARD_SMOKE_OK`）。ただし、これはGitHubの実Checkoutにあるスクリプト全体やTEIのGo Testを走らせた証拠ではない。実際のQualificationをPASSしたと誤記しない。

## Qualification Runnerの回帰防止とE1証拠の扱い

`qualify.sh`は単なる`origin/main`ローカル追跡refの一致では合格としない。開始時と終了時に`git ls-remote --exit-code origin refs/heads/main`で**実リモートmain**のSHAを照合し、実リモートが動いた／到達不能／HEADがmainを含まない場合は停止する。script全体の`bash -n`構文検査を実行前に行い、対象Testの`--- PASS:`を確認する（`go test -run`の0件一致・Skipは合格させない）。

`qualify_gate_test.sh`は、模擬gitと空の隔離ディレクトリを使い、実Network・Source変更・GitHub Actionsなしで検査する回帰テストである。

```bash
bash -n tools/reference-critical-lowering/qualify.sh
bash tools/reference-critical-lowering/qualify_gate_test.sh
```

2026-10-10にこの2件を**隔離ローカル**で実行し、構文検査成功・`QUALIFY_GUARD_PASS`を確認した。隔離実行したスクリプトのGit blob SHAは`qualify.sh=0706c5af28a4eaf08f5086add47d6c4baa4b68ae`、`qualify_gate_test.sh=2a6e6e7583deafc1c8814fb81d3621d6bcecc160`で、GitHub source blobと一致する。これはshellのpreflightに限定した証拠であり、**Go test / gofmt / full Repository Health / production Authorityの成功ではない**。

#1119が保有する`tools/reference-critical-path-evidence-audit/live-r1-smoke.mjs`は、scratchのR0→R1 Test-only Acceptanceからrunning WebのE1-A正例を構成するが、`humanApproval=false`・Providerがsyntheticである。そこで発行する`workspace-r1.json`・`target-launch.json`・`web-positive.json`も**実Authorityへ変換可能なTicketではない**。#1121が今後受け取る場合はSource Revision / Capability / Target identityを照合する監査証拠としてのみ消費し、Trusted Acceptance receipt・Runtime execution grantの代用にはしない。独立DB Read、既存Host Permission、第二Targetのactual実行は未証明。

## Web E2E: Gate B R / Target T と権限拒否の独立性

running Web TargetのBlack-box Testでは、既存Web側のGate B R (`e5231839a5dcc62cce1181a2731fcb7c70d16736`) と当該CheckoutのTarget T (exact HEAD)を別々のCLI引数として渡す。以前のTestは両方へHEADを渡していたため、`targetBuildRevision`のみ正常であれば、**誤ったBaselineを検出できなかった**。修正後はManifestおよび各HTTP Scenario Responseの双方で`baselineRepositoryRevision=R`かつ`targetBuildRevision=T`を検査する。

認証拒否（無効Credential）に加え、正当にAuthenticationされたViewerがAuthorizationで拒否され、Command/Observationに進まない経路をHTTP経由で検査する。実Web実装が現時点で保有するfixtureに対する証拠であり、実Host Grantの独立した運用確認・実DBへの保存の証拠ではない。テスト実行に成功していなければPASSとは呼ばない。
