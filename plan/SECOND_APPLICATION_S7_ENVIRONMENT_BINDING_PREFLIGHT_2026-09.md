# Second Application Reference — S7 Environment Binding Preflight

- Date: 2026-09-21
- Primary workstream: ISSUE-000040
- Stage: S7 preflight
- Entry condition for authoring: S6 complete
- Current status: **design/source-inspection preflight only; Environment source unchanged**
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
S7 exact Proposal fingerprint = not yet assigned
S7 baseModelRevision          = not yet captured
S7 Acceptance decision        = absent
```

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
