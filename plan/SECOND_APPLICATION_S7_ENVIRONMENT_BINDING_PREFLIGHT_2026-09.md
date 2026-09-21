# Second Application Reference — S7 Environment Binding Preflight

- Date: 2026-09-21
- Primary workstream: ISSUE-000040
- Stage: S7 preflight
- Entry condition for authoring: S6 complete
- Current status: **executable pressure実施済み（2026-09-21、§13）。Proposal identity確定済み。Environment sourceは未変更で、Acceptance decisionも未取得**
- This document is not an Environment Binding Proposal acceptance and does not authorize source writes.

## 1. Purpose

S6でaccepted Second Application sourceが通常Workspaceとして成立した後、Application-owned datasource aliasをEnvironment-owned physical database instanceへ明示的に結ぶ。

S7はApplication source Acceptanceとは別authorityである。

```text
S6 accepted Application
  repair_intake
  datasource alias customer_master
  target Database master_data
  access read
  contract master_data.customer ^2

S7 Environment choice
  development: customer_master -> master_data.dev
  production : customer_master -> master_data.prod
```

origin `order_entry.master` bindingをcopy authorityとして継承せず、同じphysical instanceをdestination Environmentが別bindingとして選ぶ。

## 2. Existing Environment authority

current Referenceには既に:

### development

```text
database-instance master_data.dev {
    definition master_data
    provider oracle
}

bind order_entry.master -> master_data.dev
```

### production

```text
database-instance master_data.prod {
    definition master_data
    provider oracle
}

bind order_entry.master -> master_data.prod
```

がある。

また両Environmentは:

```text
bundle tei.reference.providers ^1
```

を使い、そのbundleには:

```text
plugin tei.database.oracle ^1
```

が含まれる。

Oracle pluginは:

```text
provides database-provider database.oracle {
    provider oracle
}

provides contract-query-executor database.query.oracle {
    provider oracle
}
```

を提供する。

したがってSecond Applicationのために:

- new Database definition
- new database-instance
- new Oracle plugin
- new Provider bundle

を作るEvidenceは無い。

## 3. Minimal binding candidate

S6 complete後に検討する最小source deltaはexactly2 lines。

development:

```text
bind repair_intake.customer_master -> master_data.dev
```

production:

```text
bind repair_intake.customer_master -> master_data.prod
```

owner source:

```text
reference/workspace/environments/development/environment.tei
reference/workspace/environments/production/environment.tei
```

これは現時点ではpreflight candidateであり、まだProposal fingerprintもAcceptance witnessも持たない。

## 4. Why the binding is structurally compatible

S6後のApplication requirementは:

```text
application = repair_intake
alias = customer_master
target database = master_data
access = read
required contract = master_data.customer ^2
```

Environment Validationはbindingごとに:

1. Applicationが存在する
2. aliasがApplicationに存在する
3. target instanceがEnvironmentに存在する
4. instance definitionがApplication datasourceのtarget Databaseと一致する

ことを検査する。

current instancesは:

```text
master_data.dev  -> definition master_data
master_data.prod -> definition master_data
```

なので、上記2 bindingはこの構造条件を満たす設計である。

## 5. Required negative controls

S7 executable pressureでは共有Validationをそのまま使い、少なくとも次をfail closedする。

### unknown Application

S6 sourceが存在しない状態で:

```text
bind repair_intake.customer_master -> master_data.dev
```

を置けば:

```text
UNKNOWN_BINDING_APPLICATION
```

で拒否されること。

このためS7 authoringはS6 complete後に限定する。

### unknown datasource alias

```text
bind repair_intake.unknown_alias -> master_data.dev
```

を:

```text
UNKNOWN_BINDING_DATASOURCE_ALIAS
```

で拒否すること。

### unknown database instance

```text
bind repair_intake.customer_master -> master_data.missing
```

を:

```text
UNKNOWN_DATABASE_INSTANCE
```

で拒否すること。

### target mismatch

developmentで例えば:

```text
bind repair_intake.customer_master -> sales_core.dev
```

を:

```text
DATABASE_BINDING_TARGET_MISMATCH
```

で拒否すること。

### duplicate binding

同じEnvironment内で同じ:

```text
repair_intake/customer_master
```

を二重定義した場合:

```text
DUPLICATE_ENVIRONMENT_BINDING
```

で拒否すること。

S7専用validatorは作らない。

## 6. What this binding does not grant

Application Database Bindingはphysical instance selectionであり、次を自動的には意味しない。

- Permission grant
- database credential
- secret delivery
- network Resource grant
- Lease
- deployment executor
- application runtime executor
- effect-time authority
- user / service principal authorization

S3から保持している:

```text
authorization requirement = required-current-authority-check
```

を維持する。

exact principal / Resource / Permission identityはS7 binding lineから推測しない。

## 7. Existing resources are not inherited by name

development / productionには現在 `sales_core` migration向け:

- database-credential
- database-endpoint
- migration-workdir
- migration.atlas permissions / lease

がある。

これらはscopeがsales_core migration用であり、`repair_intake.customer_master`のruntime read credential / network authorityとして流用しない。

同様に、`order_entry.master`が同じmaster_data instanceへbindされている事実だけで、order_entryのruntime authorityをrepair_intakeへ継承しない。

## 8. Provider verification boundary

physical instanceのproviderはEnvironment sourceで既に:

```text
provider oracle
```

と定義され、`tei.reference.providers` bundleがOracle database-provider / contract-query-executorを提供する。

S7で確認するのはexisting provider resolutionがbinding追加後もValidation上成立すること。

新しいProvider configurationを作る段階ではない。

actual Oracle physical read Evidenceは別Issueのauthorityであり、S7 binding Acceptanceの代替条件にしない。

## 9. Proposed S7 executable pressure after S6

S6 complete後、temporary Workspace copyへ2 bindingだけを追加して:

```text
workspace.Load
  -> validation.Validate
```

を通す。

positive:

```text
development:
  repair_intake.customer_master -> master_data.dev

production:
  repair_intake.customer_master -> master_data.prod

Validation Error = 0
```

negative controlsは§5の5種類。

さらにdiff pressureで:

```text
changed authoritative files = exactly 2 Environment files
Application source unchanged
shared Customer / Contract source unchanged
database-instance definitions unchanged
plugin / bundle source unchanged
Permission / Resource / Lease unchanged
```

を要求する。

## 10. S7 Proposal identity

S7 ProposalはS6完了後のcurrent Workspace revisionをbaseにして初めて固定する。

理由:

- S5 AcceptanceでWorkspace model revisionが変わる
- S7はEnvironment sourceを変更する別authority decision
- pre-S6 revisionへbinding candidateを固定するとstale candidateを作る

したがって現時点では:

```text
S7 exact Proposal fingerprint = sha256:9fb68c2878f468ed57e50b73b142344df9426cbc0eb20c5d60e9389ec367e2e4
S7 baseModelRevision          = sha256:732763a2b645a6783dd8fbe61b861f2a2c7842f464c1759de846f71eaf18f762
S7 Acceptance decision        = absent
```

S6完了（`ACCEPT_COMMIT = 0423b95`）でcurrent Workspace revisionが確定したため、2026-09-21にProposal identityを固定した。

fingerprintの対象payloadは、environmentをID順に並べた次の文字列（193 bytes、UTF-8）である。再計算できるよう定義を明示する。

```text
environments/development/environment.tei
bind repair_intake.customer_master -> master_data.dev
---
environments/production/environment.tei
bind repair_intake.customer_master -> master_data.prod
```

baseModelRevisionはS5受理後のWorkspace revisionであり、S7はこのrevisionに対してのみ有効である。Workspaceが先へ進んだ場合はstale candidateとして扱い、base revisionを黙って書き換えない。

を維持する。

## 11. S7 entry / exit

Entry:

- S6 complete
- accepted `repair_intake` Application / aliasがauthoritative sourceからreloadできる
- current Workspace revisionを再取得
- development / production master_data instance freshnessを確認

Exitには少なくとも:

- exact Environment Binding Proposal identity
- temporary workspace positive Validation PASS
- negative controls PASS
- explicit Acceptance decision for S7 Proposal
- accepted Environment source
- reload / Validation PASS
- no implicit Permission / Resource / Runtime authority
- retained authorization unresolved

が必要。

S7をS5 Acceptanceへ混ぜず、Environment authorityを独立して保持する。

## 12. Handoff to S8

S7後も次は未解決としてS8へ渡す。

- current Driver APIがEnvironment instance selectionをどう受け取るか
- exact execution principal
- database credential / network authority
- First Web Targetでのrepair_intake surface
- Target-specific presentation
- DRIVER_SCENARIO_NOT_RUN
- TARGET_PROJECTION_NOT_EVALUATED

S7 bindingの成立だけでDriver / Target execution成功を主張しない。

## 13. 実行記録 — executable pressure（2026-09-21）

§9が求めたpressureを`tools/reference-business-procedure-check/second_application_s7_environment_binding_pressure_test.go`として実装し、実行した。Repositoryのauthoritative Environment sourceは変更していない（すべて`t.TempDir()`上の複製に対する検査）。

| test | 結果 |
| --- | --- |
| `TestSecondApplicationS7MinimalBindingPassesFullWorkspaceValidation` | PASS。2 bindingを足した複製が`workspace.Load` → `validation.Validate`でValidation Error 0件 |
| `TestSecondApplicationS7BindingFailsClosedOnEachStructuralDefect` | PASS（5 subtest）。§5の5種類がそれぞれ期待するcodeで拒否される |
| `TestSecondApplicationS7BindingChangesOnlyTheTwoEnvironmentFiles` | PASS。変更されたauthoritative fileはEnvironment 2 fileだけで、各fileで増えた行はbinding 1行だけ |

negative controlの対応:

```text
unknown Application        -> UNKNOWN_BINDING_APPLICATION
unknown datasource alias   -> UNKNOWN_BINDING_DATASOURCE_ALIAS
unknown database instance  -> UNKNOWN_DATABASE_INSTANCE
target mismatch            -> DATABASE_BINDING_TARGET_MISMATCH
duplicate binding          -> DUPLICATE_ENVIRONMENT_BINDING
```

いずれも共有`validation.Validate`が出すdiagnosticであり、S7専用validatorは作っていない。

positive testは「Validationが通った」ことに加えて、**loadしたWorld上に`repair_intake.customer_master`のbindingが期待するinstanceで実在すること**を確認する。これが無いと、bindingを一行も挿入していない場合でもtestが通ってしまう。origin側の`order_entry.master` bindingが壊れていないことも併せて確認する。

test自体が空振りしないことを4件の変異で確認した（bindingを挿入しない / developmentがproduction instanceへ向く / validatorがtarget mismatch checkを落とす / validatorがunknown alias checkを落とす）。すべてtestが検出した。

### まだ成立していないもの

```text
S7 Acceptance decision                 = absent
accepted Environment source            = none
Permission / Resource / Runtime authority = 未付与（§6のとおり）
```

このpressureはProposalが構造的に成立しうることを示すだけで、Environment authorityの決定ではない。§11のexit条件のうち、exact Proposal identity・positive Validation・negative controlsは満たし、explicit Acceptance decision以降は未了である。

### Acceptance経路 — 案1を仮採用（2026-09-21）

S5はShared Acceptance Boundaryの型付きoperationを通って受理した。Boundaryには当初Environment binding用のoperationが無く、次の二択があった。

1. Boundaryへ`add-environment-binding`を追加し、S5と同じ経路（Preview → 明示的Acceptance → 決定記録）で受理する
2. Environment sourceを直接authorし、S7独自の検証手順で受理する

**案1を仮採用して実装した。** 理由は、「どの入口から来た変更も同じ境界を通り、Previewと決定記録が付く」ことがこのReferenceの中心的な性質であり、Environmentだけを例外にするとS5の前例とも`RQ-027`の決定記録とも食い違うためである。加えて、型付きoperationは§6「この変更が与えないもの」を機械的に閉じられる（Environment fileのbinding一行だけに限定し、Permission / Resource / leaseへ触れない）。手編集より強い封じ込めになる。

この判断は**仮**である。次の場合に見直す。

- Environmentの決定が、意味の変更とは異なる承認者・承認単位・秘匿情報を必要とすると分かった場合
- Environment固有の知識がBoundaryのresolve / diffへ入り込み、意味中心の設計を濁し始めた場合
- 型付きoperationでは表現できないEnvironment変更（削除・付け替え・条件付きbinding等）が必要になった場合

実装は`tools/reference-shared-acceptance`の`add-environment-binding`である。層をまたぐ正しさは共有Validationに委ね、境界側はEnvironmentの存在・placementの所有関係・binding重複だけを見る。

## 14. 実装と実Reference Preview — 2026-09-21

案1に従い、受入境界へ`add-environment-binding`を追加した。

```text
operation   add-environment-binding
spec        environment / application / alias / instance
placement   そのEnvironmentを宣言しているsource file
挿入        environment blockの閉じ括弧直前へ bind <application>.<alias> -> <instance> を一行
層          environments/
差分        environmentDiff（subject kind = environment-binding、id = <env>/<app>.<alias>）
collection  environments
impact      binding -> environment / application / database-instance、instance -> database定義
unresolved  RUNTIME_AUTHORITY_NOT_GRANTED
```

境界側の検査はEnvironmentの存在、placementがそのEnvironmentの所有fileであること、同じbindingが既に無いことに限る。unknown alias / unknown instance / target mismatchは**共有Validation**が`validate` stageで出す。S7専用validatorは作っていない。

### 実際の`reference/workspace`へのread-only Preview

`main f9d756e`で実行した。`accept`は実行していない。

```text
acceptable             true
stages                 11 / 11 pass
baseModelRevision      sha256:732763a2b645a6783dd8fbe61b861f2a2c7842f464c1759de846f71eaf18f762
candidateFingerprint   sha256:fa359de44243c8fbc38ed8bbef679e5e59487a25773c48e5bdb51f807e64bc24
modelRevisionAfter     sha256:4c0ec01c83b1412454a4ab2ba4adf8cd8904ea7609f1326acb910ba5652f1f5c
semantic diff          0件
application diff       0件
environment diff       added environment-binding:development/repair_intake.customer_master
                       added environment-binding:production/repair_intake.customer_master
changed collections    [environments]
source edits           environments/development/environment.tei（+1行）
                       environments/production/environment.tei（+1行）
impact edges           8
unresolved             RUNTIME_AUTHORITY_NOT_GRANTED ×2
                       DRIVER_SCENARIO_NOT_RUN
                       TARGET_PROJECTION_NOT_EVALUATED
```

Preview前後で`reference/workspace`のfile digestは同一、`git status reference`は空である。

§10で固定した**source-levelのProposal fingerprint `sha256:9fb68c28…`は変わらない**。型付きoperationから生成される2行が、そこで定義したpayloadと同一だからである。`fa359de4…`はbase revisionを含むAcceptance candidateのidentityで、S4の`c18ae0fd…`とS5の`8a936cf9…`の関係と同じである。

### 残る停止条件

```text
explicit human Acceptance decision for the S7 Proposal = absent
accepted Environment source                            = none
Permission / Resource / Runtime authority              = 未付与
```

Previewは受理ではない。`reference/workspace`への`accept`は、exact Proposalに対する明示的なAcceptance decisionがあるまで実行しない。
