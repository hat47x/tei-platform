# Design Synthesis Provider Feasibility Scope Pressure — 2026-09

Status: **Lane D / S2 read-only refinement / no production schema adoption**

## 1. この文書の役割

本書は、`DESIGN_SYNTHESIS_SCENARIO_ACCEPTANCE_DRAFT_2026-09.md`のS2 `step 5 — Target / Provider feasibility`を、current PostgreSQL migration sourceへ当ててPressureする。

新しいProduct Requirement、Gate、PAS、Canonical kind、production Candidate schema、provider abstractionは導入しない。

確認するのは一つである。

> Target / Provider realizationがcurrent sourceで実現不能またはEvidence不足だったとき、どのcandidate scopeまで縮約してよいか。

2026-09-16時点のcurrent `main`にはPR #606 / ISSUE-000533のPostgreSQL migration renderer hardeningが統合されている。ただしactual package greenは未取得であり、本書もsource-level / read-only pressureとして扱う。

## 2. 入力となるcurrent source

主に次を読む。

- `internal/database/providers/postgres/migration_sql.go`
- `internal/database/providers/postgres/destructive_readiness.go`
- ISSUE-000533 / PR #606

current PostgreSQL sourceでは少なくとも次が成立する。

```text
fully resolved Schema Diff
  -> required

DropTable
  -> raw-style RenderMigrationSQLでは拒否
  -> complete provider partition observationが必要

DropTable + partition child / descendants
  -> destructive readiness blocked

DropTable + complete observation + ordinary table
  -> PostgreSQL DROP TABLE realizationへ進める

ChangePrimaryKey
  -> current rendererではexplicit unsupported error

CreateTable with PrimaryKey
  -> current rendererではexplicit unsupported error

AlterColumn Default change
  -> current rendererではexplicit unsupported error

unsupported / unknown ChangeKind
  -> explicit error
  -> partial SQLなし
```

重要なのは、これらが**provider realization上の事実**だということである。

## 3. 中心結論

S2でTarget / Provider feasibilityを使うとき、次を分ける。

```text
semantic candidate
  = 何を意味として成立させたいか

realization candidate / variant
  = そのmeaningを、どのTarget / Provider / binding / execution pathで実現するか
```

したがって:

```text
current PostgreSQL renderer cannot realize X
  != semantic meaning X is false
  != Business intent must be rewritten
```

である。

Provider contradictionが直接除外できるのは、そのprovider realization claimを含むscopeまでである。

## 4. Case A — DropTable observation missing

semantic intent:

```text
accepted desired schemaでは table sales.archive_candidate を削除する
```

realization candidate:

```text
PostgreSQL current providerでDropTableを実行する
```

current source:

```text
DropTable
  + partition observation incomplete / unavailable
  -> readiness executable=false
  -> reason=partition-observation-unavailable
```

このとき:

```text
semantic deletion intent
  -> contradictionではない

PostgreSQL destructive realization now
  -> unresolved / blocked by missing required Evidence
```

と扱う。

observation不足を:

```text
cannot prove safe
  -> semantic candidate invalid
```

へ短絡してはいけない。

これはR1〜R6で維持してきた:

```text
missing authority / Evidence
  != false
```

と同じ方向である。

## 5. Case B — target is partition child / has descendants

semantic intent:

```text
logical desired stateから対象tableを除去したい
```

complete observation:

```text
target is partition child
or
target has partition descendants
```

current PostgreSQL readiness:

```text
Executable=false
reason=target-is-partition-child
or
reason=partition-tree-present
```

この場合、次のbounded claimはcurrent authorityと両立しない。

```text
current PostgreSQL provider may execute this plain DropTable realization safely
```

したがって**そのrealization candidateはcontradiction / incompatibleとして除外できる**。

しかし:

```text
logical desired stateから対象を除去するmeaning
```

自体まで否定したことにはならない。

残り得る問いは:

- partition lifecycleを別手順で扱うか
- detach / recursive operation等を将来別capabilityとしてauthorするか
- desired semantic changeを見直す必要があるか
- Target / Providerの制約をhard requirementとして維持するか

であり、current rendererがBusiness meaningを自動修正してはならない。

## 6. Case C — ChangePrimaryKey unsupported

semantic candidate:

```text
desired physical schemaでPrimary Keyを変更する
```

current provider source:

```text
ChangePrimaryKey
  -> explicit error
  -> constraint identityが安全にrealizeできるまでunsupported
```

ここでexactに言えるのは:

```text
current PostgreSQL renderer realizes ChangePrimaryKey
  -> contradiction
```

である。

一方:

```text
Primary Key changeというdesired meaning
  -> contradiction
```

ではない。

S2 frontierではconceptually:

```text
semantic candidate S-pk-change
  status: still under consideration / accepted upstream meaning if already accepted

realization variant R-postgres-current
  status: incompatible with current renderer

remaining realization dimension
  missing: safe constraint identity / supported DDL realization / alternate execution path
```

のように分離できればよい。

production state enumは要求しない。

## 7. Case D — unsupported / unknown ChangeKind

current rendererは未対応ChangeKindをsilent omissionせず、whole diffをfail closedする。

これはDesign Synthesisにとって重要である。

```text
provider emitted no SQL
```

だけを見ると:

```text
no implementation needed
```

と誤認し得る。

しかしcurrent sourceはexplicit errorを返すため:

```text
meaning absent
  != meaning unsupported
```

を保てる。

S2ではunsupported provider pathをfrontierから除外またはunresolved realizationへ残せるが、unsupported meaningをsemantic no-opへ変換しない。

## 8. hard Target / Provider requirementの場合

Target / Providerが単なるrealization optionではなく、accepted requirementとして固定されている場合がある。

例:

```text
must deploy on PostgreSQL current supported profile
```

このhard constraintと:

```text
requires ChangePrimaryKey realization
```

が両立しないなら、**composite design candidate**は除外できる。

ただし除外理由は:

```text
semantic meaning is wrong
```

ではなく:

```text
semantic meaning + fixed PostgreSQL realization requirement
  -> current provider capabilityと両立しない
```

である。

このscopeを保持すると、後で:

- provider capabilityが増えた
- Target requirementが変わった
- safe alternate realizationが追加された

場合にsemantic meaningまで不要に再authoringせず再評価できる。

## 9. S2 contraction rule refinement

S2のTarget / Provider feasibilityでは、candidate elimination前に次を問う。

```text
Q1
  contradictionはsemantic claimに向いているか

Q2
  realization / binding / provider capability claimに向いているか

Q3
  Target / Provider自体がhard accepted constraintか

Q4
  missing Evidenceなのか、explicit incompatibilityなのか
```

安全な縮約は次である。

```text
missing observation
  -> realization unresolved / blocked
  -> semantic candidateを落とさない

explicit provider unsupported
  -> matching realization variantを除外可能
  -> semantic candidateを自動除外しない

explicit provider incompatibility + provider is hard requirement
  -> composite candidateを除外可能
  -> reason scopeをprovider feasibilityとして保持
```

## 10. S5への接続

Scenario Acceptance Draftでは既に:

```text
providerがCapabilityを実現できない
  -> Binding / provider feasibility concern
  != Business intent rewrite
```

としている。

今回のpressureにより、この境界はS5のfeedback routingだけでなく、**S2のcandidate contraction時点から維持する必要がある**と分かる。

つまり:

```text
S2
  provider constraintでrealization frontierを縮約

S3 / human review
  残るsemantic / realization trade-offを比較

Shared Acceptance
  meaning変更が必要ならここでのみauthor

S4
  accepted meaningからsupported realizationをderive

S5
  runtime/provider feedbackを正しいownerへ戻す
```

となる。

## 11. Acceptance refinement

S2 Acceptanceへconceptually追加すべき条件:

1. Target / Provider feasibility contradictionのscopeをsemantic claimとrealization claimで区別できる。
2. missing provider observationをsemantic contradictionとして扱わない。
3. provider unsupportedをsemantic no-opへ変換しない。
4. provider-specific incompatibilityだけでBusiness meaningを自動書き換えない。
5. hard Target / Provider requirementがある場合だけ、composite candidate eliminationへ拡張できる。
6. elimination reasonから、semantic contradictionかrealization infeasibilityかを人が判別できる。

Negative Acceptance:

- PostgreSQL renderer error -> Business requirement削除
- missing partition observation -> DropTable semantic intent invalid
- unsupported ChangeKind -> no implementation needed
- current provider limitation -> Canonical meaningをprovider都合へ自動縮退
- Target固定条件を無視して「別providerなら可能」とcandidateをsafe扱い

## 12. Context Packetへの含意

production schemaは固定しないが、S3 / human reviewへ渡すbounded contextで次の区別を失わない。

```text
candidate semantic subject
realization / binding variant
provider / target constraint
feasibility result
missing provider Evidence
reason / authority ref
hard vs optional Target constraint
```

一つの:

```text
candidate valid=false
```

へ畳むと、semantic meaningまで否定されたように見えるため避ける。

## 13. current Evidenceの限界

PR #606はcurrent mainへmerge済みである。

しかしIssue #533 / PR #606履歴が示すとおり、GitHub-hosted runner allocation blocker #513によりactual package greenは未取得である。

したがって本書が根拠にするのは:

- current mainに存在するsource shape
- explicit fail-closed behavior encoded in source/tests
- Issue / PR上のauthority boundary

までである。

次を主張しない。

- PostgreSQL provider actual qualification green
- Gate B Ready
- Gate C Ready
- runtime execution成功
- production Design Synthesis engineでこのcontractionが動作済み

## 14. current判断

今回のcurrent Referenceは、pre-Gate-C停止条件に該当する実際のgapを一つ示した。

```text
Target / Provider feasibility
  != semantic validity
```

R1〜R6とS1→S2 frontier pressureはcandidate-local proof scopeを扱っていたが、**semantic candidate scopeとrealization variant scopeの分離**は明示的にPressureしていなかった。

このrefinement後は、provider limitationからBusiness / semantic meaningを逆向きにauthorしない境界までS2へ接続できる。

これを根拠にproduction Candidate階層、Binding schema、ProviderFeasibility enum等を先行導入しない。Gate C後のactual Referenceでこの区別では情報が失われる場合だけreopenする。
