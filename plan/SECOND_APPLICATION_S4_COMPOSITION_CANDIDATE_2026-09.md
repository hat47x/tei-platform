# Second Application Reference — S4 Composition Candidate

- Status: Superseded before Acceptance — full-validation revision defined
- Date: 2026-09-19
- Primary workstream: ISSUE-000040
- Input: `plan/SECOND_APPLICATION_S3_DESTINATION_DECISION_PACKET_2026-09.md`
- Current revision: `plan/SECOND_APPLICATION_S4_FULL_VALIDATION_PREFLIGHT_2026-09.md`
- Authoritative Application source created: No
- Environment binding created: No
- Target implementation started: No

## Revision notice

この文書のApplication + Pattern 2-source Proposalは、S6 preflightでdestination Semantic Relation source / owner / keyを欠くことが分かったため、Shared Acceptance前にsupersededした。

- historical fingerprint: `b347b41b75c9864e66b4b191eefc3d668c0d8bf3dfdf186d0733f794159467c2`
- revised fingerprint: `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`
- revised source: `plan/SECOND_APPLICATION_S4_FULL_VALIDATION_PREFLIGHT_2026-09.md`

authoritative sourceへの昇格やShared Acceptanceは一度も成立していない。以下はhistorical proposal recordとして残す。

## 1. 目的

S3のdestination decisionを、既存TEI grammarだけで一つのcomposition candidateへ落とす。

S4が作るのはProposalであり、Repositoryのauthoritative Application / Pattern sourceではない。

```text
S3 decision packet
  -> deterministic S4 composition candidate
  -> Shared Acceptance review input
  != authoritative source
```

## 2. Application candidate

```text
application repair_intake {
    uses semantic customer

    database customer_master {
        target master_data
        access read
        requires {
            contract master_data.customer ^2
        }
    }
}
```

保持する境界:

- Application identity candidate = `repair_intake`
- shared Semantic namespace = `customer`
- destination-owned datasource alias = `customer_master`
- shared Database = `master_data`
- access = read
- shared Contract compatible major = `master_data.customer ^2`

このcandidateにEnvironment instance、Provider、Permission grantを埋め込まない。

## 3. Pattern candidate

```text
pattern repair_intake.customer_reference reference {
    relation repair_intake.intake.customer
    datasource customer_master
    contract-query {
        contract master_data.customer
        query find_by_id
        input customer.customer.customer_id -> id
    }
}
```

保持する境界:

- Pattern identity / Relation / datasource aliasはdestination-owned
- Contract / Query / input Typeはshared authority
- origin `order_entry.customer_reference` はprovenanceのみ
- origin `sales.order.customer` / alias `master` はcopyしない

## 4. S3とのdeterministic correspondence

S4はS3のdecisionを勝手に変更しない。

```text
S3 Application ID
  repair_intake
    -> S4 application repair_intake

S3 Relation
  repair_intake.intake.customer
    -> S4 pattern relation

S3 Pattern ID
  repair_intake.customer_reference
    -> S4 pattern identity

S3 datasource alias
  customer_master
    -> Application database alias
    -> Pattern datasource
```

shared authorityはS1/S2から引き継ぐ。

```text
customer.customer
customer.customer_id
master_data.customer ^2
master_data.customer/find_by_id
```

## 5. Authorization boundary

S3の、

```text
authorization requirement = required-current-authority-check
```

をS4で保持する。

ただしcurrent Application / Pattern compositionへ、未確定のprincipal / Resource / Permission identityを捏造して追加しない。

```text
composition contains read capability requirement
  != read authority granted
```

exact execution authorityは後続authority ownerで解決する。

## 6. Business Procedure boundary

S3どおり、第一ReferenceではBusiness Procedureを追加しない。

理由:

- current scopeは単一のread-only lookup
- step sequence / handoff / state transitionをまだ必要としない
- Procedure追加は今回のcomposition成立に不要

S4 candidateからProcedure DSLを生成しない。

## 7. Environment boundary

S4 candidateにEnvironment sourceを含めない。

```text
Application database requirement accepted later
  -> S7 Environment Binding Proposal
```

であり、

```text
S4 composition
  -> bind repair_intake.customer_master -> master_data.dev
```

とはしない。

development / productionのactual binding targetとProviderはS7までunresolved。

## 8. First Target boundary

S4はTarget-neutralなcompositionまでを扱う。

第一Web Targetで予定する、

- Customer ID input
- lookup action
- zero-or-one result / not-found presentation

はS8の実装対象である。

route / DOM / selector / HTML / CSSをS4 sourceへ持ち込まない。

## 9. Shared Acceptance review packet

S5へ渡すreview inputは次。

### Shared authority

- `semantic-data:customer.customer`
- `semantic-type:customer.customer_id`
- `database-contract:master_data.customer`
- `database-contract-query:master_data.customer/find_by_id`

### New destination-owned meaning

- Application candidate `repair_intake`
- Relation candidate `repair_intake.intake.customer`
- Pattern candidate `repair_intake.customer_reference`
- datasource alias `customer_master`

### Provenance

- `application-pattern:order_entry.customer_reference`

### Explicit unresolved

- exact authorization principal
- exact authorization Resource / Permission
- development binding target
- production binding target
- Provider verification
- Target realization details

### Review questions

1. destination identityはrepair-intake contextを過不足なく表すか
2. Relationがorigin sales contextを誤って継承していないか
3. shared Customer authorityをcopyせず参照できているか
4. datasource aliasがApplication-localであることが明確か
5. Authorization requirementとgrantが分離されているか
6. Environment / Target authorityをS4が先取りしていないか
7. first ReferenceにBusiness Procedure追加が本当に不要か

## 10. Test-only Source pressure

- `tools/reference-business-procedure-check/second_application_s4_composition_candidate_pressure_test.go`

確認すること:

1. S3 decisionからApplication sourceをdeterministicに構成できる
2. existing Application grammarでparseできる
3. S3 decisionからPattern sourceをdeterministicに構成できる
4. existing Pattern grammarでparseできる
5. Application / Pattern間のaliasが一致する
6. shared Contract / Query mappingを保持する
7. origin Relation / aliasをcopyしない
8. Environment / Target / AcceptanceをS4へ混ぜない

## 11. Evidence height

S4の文書・test-only source追加はSource / Design Evidenceである。

actual focused `go test` PASSが得られた場合のみExecutable Evidenceへ昇格する。

またparser PASSが得られても:

```text
parseable Proposal
  != Shared Acceptance
  != authoritative Application source
  != reload / Validation Reference Evidence
```

である。

## 12. Conclusion

S3で選んだdestination meaningを、追加Portability grammarなしに既存Application / Pattern grammarへ一意に写せるS4 candidateを固定した。

次はS5 Shared Acceptanceであり、そこを通るまでauthoritative sourceを作成しない。
