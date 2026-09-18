# Actor Context R3AB — Cross-audience Freshness Drill-down Provenance

- Status: ISSUE-000053 / Phase 3 cross-audience drill-down provenance pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3AA_AUTHORIZATION_FRESHNESS_STOP_HANDOFF_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3V_CROSS_AUDIENCE_REASON_IDENTITY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3R_PROJECTION_DEPENDENCY_PROVENANCE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3X_AUTHORIZATION_CONTEXT_IDENTITY_BOUNDARY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3Y_AUTHORIZATION_REEVALUATION_REUSE_AUTHORITY_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3Z_AUTHORIZATION_FRESHNESS_PRECEDENCE_EVIDENCE_GATE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3AAでは:

```text
authorization-evaluation-freshness-authority-unresolved
```

をHuman / Agent共通のOperational stopへ接続した。

次に確認するのは、Human orientationとAgent resolution-inspectionが同じmachine reasonをdrill downしたときに、**同じprovenance rootへ戻れるか**である。

```text
same shared reason identity
  -> same derived-analysis chain
  -> same provenance roots

different audience
  != different truth root
```

をPressureする。

## 2. plan / test artifactをauthority rootへしない

R3X / R3Y / R3Z / R3AAはderived analysis / Evidence Gate / projectionを記述する。

それ自体をBusiness / Authorization authorityへ昇格させない。

R3ABではdrill-downを二層に分ける。

```text
derived nodes
  R3AA stop projection
  R3X context identity analysis
  R3Y freshness authority analysis
  R3Z precedence Evidence Gate

provenance roots
  actual/reference source input
  runtime Observation
  explicit test-only context input
  corroborating provider rule
```

したがって:

```text
plan document
  != authority root
```

である。

## 3. current freshness stopはtest-only scenarioである

current Repository-backed BP-06ではAuthorization action authority自体がunresolvedであり、actual evaluation invocationへは進まない。

R3X〜R3AAのsame-tuple freshness caseは、test-only explicit contextを使ったpressureである。

したがってR3ABでも:

```text
action = execute in test-only context
  != current Business/Application action authority accepted
```

を明示する。

test-only contextをReference authorityへ偽装しない。

## 4. provenance root role

test-onlyでは少なくとも次を分ける。

### Reference Evidence

- Business Procedure Reference
  - Authorization requirement pressure
- protected-target overlay
  - protected resource pressure

両方ともcurrent Reference / pressure inputであり、production Canonical authorityへ昇格させない。

### Runtime Observation

- prior Authorization Observation
  - Business address
  - subject
  - action
  - resource
  - observed outcome

outcomeが `allowed` / `denied` でも、R3AA freshness stopではcurrent RuntimeOutcomeへ昇格させない。

### Test-only Context

- R3X explicit Authorization context input

current Repositoryでmissingなaction authorityを補ったproduction sourceとして扱わない。

### Corroboration

- Legacy Authorization policyのmatching allow rule

```text
corroborating policy
  != freshness directive
  != Business/Application action authority
  != current Authorization Observation
```

を維持する。

## 5. root identityとlocatorを分離する

R3Rと同様:

```text
provenance root identity
  != current locator
```

である。

source file move等でlocatorが変わった場合:

```text
same root identity
  + locator changed
  -> drill-down locator refresh

  != different truth root
```

とする。

content fingerprintはcurrent source revisionを説明するために保持できるが、root identityへlocatorを混ぜない。

## 6. Human / Agentのdisclosure差

Human / Agentは同じshared root setへ戻る。

ただしinlineに露出するdetailは同じでなくてよい。

```text
shared roots
  = same

inline disclosure
  = purpose-bound
```

Agentはruntime Observationやcorroborating policyの詳細まで表示できる一方、Human compact viewではそれらをinline omitできる。

しかしomitされたdetailを別rootへ差し替えない。

```text
omitted inline
  != different provenance
```

を維持する。

## 7. derived chain

test-only chainは概念上:

```text
freshness stop reason
  -> R3AA operational stop projection
     -> R3X context identity
     -> R3Y reevaluation / reuse authority axes
     -> R3Z freshness precedence Evidence Gate
        -> provenance roots
```

である。

R3Z Gateが `insufficient-evidence` であることも、missing directiveを暗黙補完する理由にしない。

## 8. test-only source

- `tools/reference-business-procedure-check/actor_context_r3ab_cross_audience_freshness_drilldown_provenance_pressure_test.go`

Pressureするcase:

1. Human / Agentのshared freshness reason identityが一致する
2. Human / Agentが同じderived node setへ戻る
3. Human / Agentが同じprovenance root identity setへ戻る
4. inline disclosure差はshared rootsを変えない
5. prior Observation outcomeをcurrent RuntimeOutcomeへ昇格させない
6. Legacy policyをcorroborationに留める
7. R3X explicit action contextをtest-only inputに留める
8. locator-only moveはroot identityを変えない
9. plan / test artifactをauthority rootへしない
10. TrustedRoot / CanonicalTruth / PreferredSource / Winner / NextAction / HandoffOwner等を追加しない

## 9. current conclusion

R3ABで固定するのは:

```text
shared reason identity
  -> same provenance chain
  -> same root identity

audience-specific disclosure
  != audience-specific truth

derived analysis
  != authority root

corroboration
  != authority

prior Observation
  != current outcome when freshness authority unresolved
```

である。

production provenance graph、UI drill-down protocol、audit store、notification transportは追加しない。

## 10. 次

次段ではdrill-down rootごとに:

- current / changed / locator-refresh
- Reference Evidence / runtime Observation / test-only input / corroboration

を保ったまま、freshness stopのre-evaluation explanationへ接続できるかをPressureする。

特に、runtime Observationの更新とReference source変更を同じ「source changed」へ潰さない。
