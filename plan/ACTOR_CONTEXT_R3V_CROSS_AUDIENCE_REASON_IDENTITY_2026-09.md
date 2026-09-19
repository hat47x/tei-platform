# Actor Context R3V — Cross-audience Re-evaluation Reason Identity

- Status: ISSUE-000053 / Phase 3 cross-audience reason identity pressure / Reference-only
- Date: 2026-09-19
- Parent: `plan/ACTOR_CONTEXT_R3U_MULTI_REASON_AGGREGATION_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3P_PURPOSE_BOUND_STOP_PROJECTION_OMISSION_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3S_PROJECTION_DEPENDENCY_REASON_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3T_PROJECTION_REEVALUATION_REASON_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Uでは一つのbounded projection内で、複数reasonをlosslessにaggregateできるようPressureした。

次に確認するのは、Human orientationとAgent resolution-inspectionのようにselected categoryが異なるprojection間でも、**共有されるreasonが同じdependency provenanceへ戻れるか**である。

```text
different bounded views
  + shared selected category
  + same underlying change
  -> same reason identity
```

をPressureする。

## 2. reason identity

R3Vではtest-only reason identityを次で構成する。

```text
category
+ change class
+ dependency key
+ dependency reason
```

次はidentityに含めない。

- audience
- purpose
- Human display wording
- presentation variant
- list position
- rendering order

したがって:

```text
same machine reason
  rendered differently
  -> same machine reason identity
```

である。

## 3. disclosure setは同じでなくてよい

Human orientationは詳細categoryをomitし、Agent resolution-inspectionはより多くのcategoryをselectedする。

例えば同じBusiness Procedure Reference changeでも:

```text
Human:
  machine-reasons -> affected

Agent:
  machine-reasons -> affected
  known-authority-facts -> affected
```

となり得る。

ここで:

```text
shared machine-reasons identity
  = same provenance

Agent-only known-authority-facts reason
  != Human reason missing / inconsistent
```

である。

## 4. audienceをreason authorityへしない

同じunderlying dependency changeから得たreasonがHuman / Agentで共有されても、その一致から:

- universal impact packet
- preferred audience
- stronger authority
- approval state
- next action

を作らない。

```text
cross-audience agreement
  != stronger authority
```

である。

## 5. dependency key driftは別identity

category / change class / dependency reasonが同じでもdependency keyが異なるなら、別reasonとして扱う。

```text
same wording
  + different provenance
  != same reason identity
```

である。

これによりsource差をpresentation similarityで隠さない。

## 6. audience-specific wordingはderived

Human側のdisplay summaryを変更してもR3V reason identityは変化しない。

Agent側にdisplay textを追加してreason identityを変える設計もしない。

R3O / R3Pの:

```text
presentation
  != machine authority
```

を維持する。

## 7. test-only source

- `tools/reference-business-procedure-check/actor_context_r3v_cross_audience_reason_identity_pressure_test.go`

Pressureするcase:

1. same Business Procedure changeのmachine-reasons identityがHuman / Agentで一致する
2. Agent-only selected categoryはHumanへ強制開示しない
3. audience / purposeをreason identityへ含めない
4. dependency keyが異なるreasonを同一化しない
5. duplicate cross-audience witnessはaudience setだけを正規化する
6. UniversalPacket / PreferredAudience / Winner / StrongerAuthorityを作らない

## 8. current conclusion

R3Vで固定するのは:

```text
shared reason identity
  -> same dependency provenance

different disclosure set
  != different underlying reason

audience-specific presentation
  != reason identity
```

である。

production notification schema、shared inbox、cross-agent protocol、UI message formatは追加しない。

## 9. 次

次段ではshared reason identityをHuman / Agentがそれぞれdrill downした場合に、同じauthority / Evidence rootへ到達することをPressureする。

特に:

- reason identity
- source role
- current locator
- authority / corroboration distinction

を維持し、drill-downの入口差から別のtruthを作らない。
