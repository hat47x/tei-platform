# Second Application Reference — S4 Full-Validation Preflight / Proposal Revision

- Status: Revised Proposal defined / source-inspection preflight complete / executable pressure pending / Shared Acceptance not attempted
- Date: 2026-09-20
- Primary workstream: ISSUE-000040
- Trigger: S6 preflight against current workspace Validation
- Supersedes Proposal fingerprint: `b347b41b75c9864e66b4b191eefc3d668c0d8bf3dfdf186d0733f794159467c2`
- Current revised Proposal fingerprint: `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`
- Authoritative destination source created: No

## 1. Full-validation gap

旧S4はApplication / Pattern parserでは成立したが、current workspace Validationでは `repair_intake.intake.customer` が実在するSemantic Relationであることを要求する。さらにContract QueryでRelationを辿るにはRelation key、source/target attribute、同一Semantic type、Relation targetとQuery resultの整合が必要である。

したがって旧2-source ProposalはShared Acceptance前にsupersededとする。少なくとも `UNKNOWN_APPLICATION_PATTERN_RELATION` を避けるため、destination Semantic sourceをProposalへ含める。

## 2. Revised destination Semantic

```text
namespace repair_intake

data Intake {
    attribute customerId type customer.customer_id
    relation customer to customer.customer one
        key repair_intake.intake.customer_id -> customer.customer.customer_id
}
```

destination-owned subjects:

- `semantic:repair_intake`
- `semantic-data:repair_intake.intake`
- `semantic-attribute:repair_intake.intake.customer_id`
- `semantic-relation:repair_intake.intake.customer`

shared authority remains `customer.customer` / `customer.customer_id`.

## 3. Revised Application

```text
application repair_intake {
    uses semantic repair_intake
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

## 4. Revised Pattern

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

`one` Relationと `zero-or-one` Query resultはcurrent reference-query compatibility ruleで互換である。

## 5. Exact revised Proposal identity

payloadは `<Semantic source>\n---\n<Application source>\n---\n<Pattern source>` の順で、各component末尾に追加newlineを含めない。

- Semantic SHA-256: `1005c6deb670c277c8b80fb0b353f1eecea690594f2ef4b80df28cc638736e63`
- Application SHA-256: `f96fdb46f9385c205d360f56d0f91a6e841ff31b348ebf1db2cf03381751772f`
- Pattern SHA-256: `41c8f81ebab384486d4df6a7cdaade07df8d07c89d22c25c2291d8b71a583ab1`
- bundle length: `741 bytes`
- bundle SHA-256: `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d`

旧fingerprintへのwitnessをrevised Proposalへ流用しない。

## 6. S6 path preflight

Acceptance後の最小source pathはcurrent loader規約から次でよい。

```text
reference/workspace/semantic/repair-intake/intake.tei
reference/workspace/applications/repair-intake/application.tei
reference/workspace/applications/repair-intake/patterns/customer-reference.tei
```

workspace manifest変更、Persistence Mapping、Environment binding、Provider、Authorization grant、Business Procedure、Target realizationはこの修正には不要である。

## 7. Evidence boundary

既存S3/S4/S5 Go pressureは旧2-source Proposalを前提とする部分があり、このrevisionをExecutable Evidenceとして証明しない。Repository policy上、コード変更はlocal `tools/repository-health-check.sh` PASSが必須であるため、health-capable WSL cloneを実行できないこの接続ではGo sourceを変更しない。

次のhealth-capable executionではS3 decision pressure、S4 Semantic/Application/Pattern pressure、temporary workspace full validation、revised fingerprintをS5 pressureへ反映し、Repository health checkを通す。

S5のrevised review contractは `plan/SECOND_APPLICATION_S5_SHARED_ACCEPTANCE_REVIEW_PACKET_2026-09.md` に再生成した。ただしExecutable pressure未更新のため、Shared Acceptance eligibilityはまだ成立しない。

## 8. Acceptance boundary

revised ProposalはまだProposalである。Shared Acceptanceはfingerprint `c18ae0fdef8e01aa965e2ad84c86ecfc21ca0608c07b1808e73368a5f434458d` を対象にexplicit witnessが得られるまで成立しない。
