# Business Procedure Uncertainty Preservation Pressure — 2026-09

Status: **Lane C / test-only design pressure**

## 1. 問い

thin Canonical Business meaningへBusiness Procedureを持ち上げる場合、triggerやactorについて「何が書かれているか」だけでなく、**どこまで確定しているか**を失わずに保持できるか。

現在の`order-entry-business-procedure.json`では、trigger / actorはどちらも`status: candidate`である。これは単なる注記ではない。TEI Product Principle P11「分からない状態を、分かったことにしない」に従えば、candidate / unresolvedをaccepted factへ暗黙昇格させるnormalizationはBusiness meaningを変えてしまう。

このPressureは、ProcedureをWorkflow Engine化せず、Authorization identityも混ぜずに、この認識上の不確定性（epistemic status）を保持する最小境界を試す。

## 2. 現時点の仮説

thin Canonical Business meaningがtrigger / actorを保持するなら、最低限次の組をlosslessに保持する必要がある。

```text
trigger
  status
  text

actor
  status
  label
```

ここで`status`はUI decorationではなく、そのBusiness statementの確定度を示す意味の一部として扱う。

ただし、現時点で`candidate | unresolved | accepted | ...`の完全なenumをCore契約として固定しない。Reference pressureがまだ一つの業務文脈に偏っているためである。

## 3. 保存しなければならない意味

### 3.1 candidateをcandidateのまま保持する

`candidate`をloader / normalizer / projectionの都合で`accepted`や既定値へ置換しない。

### 3.2 candidate → unresolvedはsemantic change

triggerまたはactorのstatusを`candidate`から`unresolved`へ変更した場合、Procedure IDやApplication Pattern参照が同じでも、normalized Business meaningは変化したとみなす。

したがってBusiness-only semantic diff / impactで観測可能でなければならない。

### 3.3 unknownを成功へ丸めない

status欠落や未知語彙を「accepted相当」と推定しない。

将来、欠落をsemantic error / review diagnostic / unresolvedとしてどのseverityへ分類するかは別途Reference Evidenceで決める。現段階ではseverityを固定しない。

## 4. actorとAuthorizationを分離する

Business Procedureのactorは、業務上「誰がこの仕事を担うか」を説明するbusiness/job labelである。

ここへ次を混ぜない。

- Authorization Role
- Principal / Subject identity
- Permission
- Credential
- Authority policy

actor labelまたはactor statusを変更しても、Application Pattern / Capability / Authorization contractが暗黙に書き換わってはならない。

同じ`営業担当者`というBusiness actorから、環境や組織ごとに異なるAuthorization realizationが成立し得るためである。

## 5. triggerとRuntime Eventを分離する

Business Procedureのtrigger textは「何を契機に業務が始まるか」というBusiness meaningであり、次を意味しない。

- event subscription
- message queue trigger
- scheduler
- timer
- webhook binding
- retry policy
- workflow state transition

runtime triggerへrealizeする場合は別のexplicit Binding / Capability境界で扱う。

## 6. test-only Pressure

`tools/reference-business-procedure-check/uncertainty_pressure_test.go`では少なくとも次をPressureとして固定する。

1. 現行Referenceのtrigger / actor `status: candidate`をnormalization後も保持する。
2. candidate → unresolved変更をBusiness semantic changeとして観測する。
3. uncertainty変更でApplication Pattern refsを変えない。
4. actor uncertainty変更をAuthorization identity変更へ読み替えない。
5. epistemic statusをProjection時に捨てない。

actual Repository runnerでtestが実行されるまでは、これは**test-only specification**でありgreen Evidenceではない。

## 7. multi-Application Pressureとの関係

Business Procedureは一つのApplication所有物とは限らない。複数Application Patternやmanual / external workを跨ぎ得る。

したがってactor / trigger uncertaintyもApplication-local metadataへ押し込めない。Procedure側のBusiness meaningとして保持し、各Applicationへのrealizationはexplicit ref / Bindingで追跡する。

## 8. Canonical候補への影響

現時点のthin candidateは次のようになる。

```text
Business Procedure
  identity
  purpose
  trigger
    epistemic status
    business meaning
  actor
    epistemic status
    business/job label
  ordered Business Actions
  explicit semantic refs
```

ただしこれはまだCanonical schema採用決定ではない。

特に未確定なのは次である。

- status vocabularyをCore enumにするか、拡張可能なvalueにするか
- missing / unknown statusのvalidation severity
- epistemic statusをtrigger / actor以外のBusiness statementsへ一般化するか
- actor identityを将来必要とするか、labelで十分か

## 9. 現時点の判断

**uncertainty preservationはthin Business meaning候補の必要条件として残すが、汎用Uncertainty frameworkは導入しない。**

Procedure Referenceで実際に現れたcandidate / unresolvedをlosslessに扱うところから始める。

これはP11を守りつつ、Coreへ抽象的なcertainty latticeやapproval state machineを先回りして持ち込まないための境界である。
