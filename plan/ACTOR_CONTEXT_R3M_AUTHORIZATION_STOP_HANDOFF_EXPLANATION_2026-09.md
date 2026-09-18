# Actor Context R3M — Authorization Stop / Handoff Explanation Projection

- Status: ISSUE-000053 / Phase 3 stop-explanation pressure / Reference-only
- Date: 2026-09-18
- Parent: `plan/ACTOR_CONTEXT_R3L_AUTHORIZATION_ACTION_LOCUS_EVIDENCE_GATE_2026-09.md`
- Related:
  - `plan/ACTOR_CONTEXT_R3J_AUTHORIZATION_ACTION_AUTHORITY_GATE_2026-09.md`
  - `plan/ACTOR_CONTEXT_R3I_AUTHORIZATION_TARGET_EVALUATION_CADENCE_2026-09.md`
- Side effect: None
- Product Gate impact: None
- Production schema impact: None

## 1. 目的

R3Lでaction-authoring locusのproduction選択はEvidence不足として停止した。

read-only AI Business Actor本線では、ここで必要なのはmissing authorityを推測して先へ進むことではなく、**どのauthorityが未解決で、何が既知で、何を次に確認すべきかを正確に説明すること**である。

R3Mではそのstop / handoff explanationをtest-only projectionとしてPressureする。

## 2. current BP-06 stop boundary

current BP-06では:

```text
Authorization requirement = required
protected resource = order_entry.submit_order
Authorization action authority = unresolved
Authorization evaluation = not reached
Authorization invocation = not attempted
```

この状態を:

```text
denied
unauthorized
not allowed
policy failure
```

へ変換しない。

正しい説明は:

```text
Authorization action authority is unresolved.
Evaluation has not been attempted.
```

である。

## 3. known / unresolved / not-attemptedを分離する

stop explanationでは少なくとも:

- Business address
- requirement status
- known protected resource
- unresolved authority dimension
- current blocking reason
- not-attempted downstream stage
- source / drill-down refs
- authority limits

を分離する。

```text
known resource
  != complete target

unresolved action authority
  != authorization denied

not attempted
  != failed
```

を維持する。

## 4. handoff ownerを発明しない

R3Bから維持している通り、stop reasonだけから:

- security team
- application owner
- business owner
- administrator
- human approver

等の具体的handoff ownerを推論しない。

R3Mが示してよいのは:

```text
next authority needed = Authorization action authority
```

までである。

誰がそのauthorityをauthor / acceptするかは別のaccepted ownership Evidenceが必要である。

## 5. sourceへdrill downできる

current BP-06 explanationでは最低限:

- Business Procedure Reference
- protected-target overlay
- R3J action-authority gate

へ戻れるようにする。

ただしR3M projection自身をauthority sourceにしない。

```text
explanation projection
  -> points to authority / Evidence
  != authority itself
```

## 6. runtime denialとの区別

将来action authorityが解決され、Authorization evaluationが `denied` を返した場合は:

```text
authority unresolved
  != evaluated denied
```

として別説明にする。

R3Mではcurrent BP-06についてdenialを生成しない。

## 7. test-only source

- `tools/reference-business-procedure-check/actor_context_r3m_authorization_stop_explanation_pressure_test.go`

Pressureすること:

- current BP-06 stopをaction-authority unresolvedとして説明する
- requirement / resourceの既知情報を失わない
- evaluation / invocationをnot-attemptedのまま保持する
- unresolvedをdenied / failedへ変換しない
- owner / approver / teamを発明しない
- source refsを保持する
- projectionにApproved / Authorized / CanExecute / Confidenceを置かない

## 8. current conclusion

R3Mで固定するのは:

```text
fail closed
  + explain exact unresolved authority
  + preserve known context
  + preserve drill-down refs
  + do not invent handoff owner
```

である。

これはAI Business Actorの停止時UX / operational explanationに必要だが、production schemaやAgent frameworkを追加するものではない。

## 9. 次

次段では、異なるstop reason:

- requirement unresolved
- principal unresolved
- action/resource unresolved
- evaluation denied
- resource/provider unresolved

を同じ説明形式へ潰さず、stage-awareなstop taxonomy / projectionとして比較する。

一つのgeneric error stringへ畳まない。
