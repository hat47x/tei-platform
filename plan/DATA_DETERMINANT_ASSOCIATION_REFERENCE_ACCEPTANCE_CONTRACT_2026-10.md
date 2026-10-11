# Data Determinant Association Reference Acceptance Contract — 2026-10

Status: **Reference acceptance contract / no new Core Requirement or Canonical syntax**

Related:

- `reference/business-system-design/data-determinant-association-pressure.md`
- `reference/business-system-design/data-determinant-association-seed-ledger.json`
- `plan/DATA_DETERMINANT_PROMOTION_GATE_2026-10.md`
- `plan/DATA_DETERMINANT_CORE_CONTRACT_CANDIDATE_2026-10.md`
- `plan/issue/000047-knowledge-first-semantic-exploration.md`

## 1. 目的

CoverageAssignment Referenceを、単に「もっともらしいData設計例がある」状態で終わらせず、どのEvidenceが揃えばどのclaimまで上げてよいかを段階化する。

この契約がacceptする対象は**Reference pressureの品質**であり、business occurrence identifier / candidate key / functional dependencyのCore syntaxではない。

~~~text
Reference pressure accepted
  != Core determinant contract accepted
  != parser / Canonical IR change authorized
~~~

## 2. 三段階status

### DAR-P1 — epistemically safe TEI-original association pressure

次をすべて満たす。

#### provenance / nature

1. CoverageAssignment scenarioがTEI-original Reference authoringであることを明示する。
2. 外部Source-backed factが無い場合、`sources` / `sourceBackedFacts`を空のまま保つ。
3. 外部domainで一般に成立するfactとして表現しない。
4. TEI-original decision、proposal、unresolved、cross-view questionを区別する。
5. Canonical authorityを主張しない。

#### business pressure

6. ServiceContractとAssetがCoverageAssignmentから独立したoccurrenceとして説明される。
7. 同じServiceContract / Asset pairを終了後に再activationでき、別CoverageAssignment occurrenceになることをReference decisionとして明示する。
8. `effectiveFrom`をこのReferenceのoccurrence identity候補に含め、`effectiveTo`を同一occurrenceの終了条件として区別する。
9. history revision identityをbusiness occurrence identityから分離する。
10. Relation先の差し替えを同一occurrenceのidentity mutationへ丸めず、新しいoccurrenceとして扱う。

#### realization independence

11. surrogate primary key + business unique constraintとcomposite physical primary keyの双方で同じbusiness meaningを説明できる。
12. physical primary keyの形をbusiness occurrence identityのauthorityにしない。

#### bounded uncertainty

13. temporal overlap policy、relation semantics contract、Core syntax、Shared Acceptance authoring path等の未解決事項を明示する。
14. full contract management、billing、authorization、workflow等をReference scopeへ持ち込まない。

P1で主張できること:

~~~text
reviewable TEI-original association determinant pressure exists
~~~

P1でまだ主張してはいけないこと:

~~~text
external domain validity proven
determinant Core contract validated
business identity syntax accepted
functional dependency syntax accepted
~~~

### DAR-P2 — cross-view determinant pressure integrated

P1に加えて、Repositoryの既存Knowledge / Business System Design pressureへ統合し、次を確認する。

1. contained repeatの`Order / OrderLine`では、accepted scoped uniquenessからderived candidateを出せる。
2. changeable-reference shapeではRelationの存在だけからidentity / FD candidateを出さない。
3. actual Repositoryの`security.AuthorizationGrant`で、`subjectId / action / resource`という複数軸からcomposite business keyを推測しない。
4. CoverageAssignment positive pressureでは、pairだけ、`effectiveFrom`だけ、surrogate IDだけ、revisionだけではidentityが不足する理由を説明できる。
5. occurrence identity / candidate key / functional dependencyを一つの万能`Key`へ統合しない。
6. determinant term語彙の共通化候補が、contained repeatとassociationの双方を表現できる。
7. changeable referenceへ同じtermを自動適用しない。
8. functional dependencyのdirectionとdependentをidentityから分離する。
9. occurrence identityとhistory / version identityを別assertionとして説明できる。
10. Data Design ContextのReview ProtocolがRelation archetype確認をProposal形成前に要求する。
11. derived candidate / fingerprint / Review Protocolをsemantic authorityへ昇格しない。
12. Source / Evidence / Shared Acceptance traceを既存provenance境界へ接続する方針を維持する。

P2で主張できること:

~~~text
association determinant pressure is integrated as source/test-level design evidence
~~~

P2でもまだ主張してはいけないこと:

~~~text
Core syntax ready
all relation archetypes covered
external business-domain acceptance obtained
~~~

### DAR-P3 — independent-review and execution-qualified Reference evidence

P2に加え、current Reference revisionに対して独立reviewとactual Repository execution Evidenceを得る。

最低限:

~~~text
independent reviewer decision
exact Reference revision / fingerprint
gofmt / format check
go test ./internal/knowledgecontext
go test ./cmd/tei
go test ./tools/reference-business-procedure-check
full tools/repository-health-check.sh
clean pre/post repository state
~~~

独立reviewは少なくとも次のDDA checklistを確認する。

#### DDA-01 occurrence definition

CoverageAssignmentが何を一つのbusiness occurrenceと数えるかを、physical row identityに依存せず説明できる。

#### DDA-02 recurrence

同じServiceContract / Asset pairを時間を変えて再activationできるため、pairだけをidentityとしない。

#### DDA-03 effectiveFrom role

`effectiveFrom`がこのReferenceではoccurrence identity componentである理由を説明できる。ただしdatetime一般へ一般化しない。

#### DDA-04 effectiveTo role

`effectiveTo`変更を同一occurrenceのvalidity更新として扱い、identity componentへ自動昇格しない。

#### DDA-05 relation replacement

ServiceContract / Asset差し替えを同一CoverageAssignmentの更新として扱わない。

#### DDA-06 history/version

revision、history row、latest flag等をbusiness occurrence identityと分離する。

#### DDA-07 persistence independence

surrogate PK / composite PKの差でbusiness meaningが変わらない。

#### DDA-08 FD direction

candidate determinantからdependentへの方向を明示し、reverse dependencyを仮定しない。

#### DDA-09 relation archetype boundary

contained repeat / changeable reference / independent associationを同じRelation shapeとして扱わない。

#### DDA-10 epistemic separation

TEI-original Reference decisionをexternal Source-backed factへ誤分類しない。

#### DDA-11 unresolved preservation

Relation semantics contract、Core syntax、temporal overlap、Shared Acceptance path等の未解決を残す。

#### DDA-12 no premature Core promotion

Reference reviewのPASSをParser / Canonical IR primitive採用へ自動変換しない。

P3で初めて次を主張できる。

~~~text
CoverageAssignment determinant Reference is independently reviewed
and execution-qualified at the current Reference scope
~~~

それでもCore contract採用には`DATA_DETERMINANT_PROMOTION_GATE_2026-10.md`の再評価が必要である。

## 3. review outcome

review outcomeは少なくとも次を区別する。

~~~text
review-required
review-blocked
review-passed
~~~

`review-passed`にはreviewed revision / fingerprintが必要である。Reference本文を変更した場合、旧reviewを自動carry-forwardしない。

reviewer自身がblocker remediationをauthorした場合は、独立性を失ったものとして再reviewを要求する。

## 4. current status — 2026-10-10

~~~text
DAR-P1
  candidate evidence exists
  independent review not yet obtained

DAR-P2
  implementation / Reference pressure exists
  execution qualification not yet established for current head

DAR-P3
  Not Ready

Core determinant contract
  Not Ready
~~~

current seed ledgerの`independentReview = required`を維持する。

## 5. closure rule

Reference acceptanceとCore promotionを別々に閉じる。

~~~text
DAR-P3 PASS
  -> Reference evidence quality is sufficient

Promotion Gate re-evaluation
  -> decide whether a durable Core contract is justified

Shared Acceptance
  -> accept an actual Core change if proposed
~~~

この順序を逆転させない。
