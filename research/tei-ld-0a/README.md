# TEI-LD-0A corpus v0 — no-training checkpoint

This work unit implements the bounded synthetic corpus mechanics tracked by #876.

## Current evidence status

- The fixture rows are synthetic mechanics fixtures only.
- They are not historical Proposal/Acceptance Evidence.
- They are not training data and must not be used to fit logistic, LightGBM, Jev, LLM, or another learned model.
- No teacher/provider/model output is captured by this corpus v0.
- No row can write Canonical or authoritative source state.

The corpus exists only to exercise schema, identity, validation, candidate-set hashing, split/leakage checks, and derived V1-V3 research views. `DecodeCorpus` / `ValidateCorpus` validate these synthetic mechanics; they do not prove that a Canonical identity exists in an authoritative revision.

## Entry condition for historical evidence

Do not label historical rows as Evidence until #267 provides actual Proposal/Acceptance records with enough provenance to bind:

- source and base revisions;
- candidate identities and candidate-set identity;
- Proposal/decision provenance;
- reload and Validation/Conformance linkage where applicable.

If those records are absent, improve the upstream Evidence path rather than inventing labels.

Before any historical row is called Evidence, run `ValidateCorpusWithCanonicalResolver` with a resolver backed by the authoritative Canonical revision source. An unresolved candidate or final mapping fails closed; a nil resolver is rejected.

## Branch boundary

This is the single TEI-LD-0A corpus work unit. After its repository health gate passes and it is merged to `main`, delete or realign `research/tei-ld-0a-corpus-v0` before any teacher capture or model fitting. Do not create a parallel model-training branch from this checkpoint.

## Derived-view boundaries

The V1-V3 helpers are research projections over the recorded corpus, not new authority.

- V1 candidate-reuse ranking emits a positive only for a current accepted final Canonical mapping that is still a member of the recorded candidate set at the exact revision. Unknown candidates are never negatives. Negatives are limited to explicitly rejected or deterministically incompatible Canonical candidates from the same decision context.
- V2 incompatibility emits targets only from rows with available referenced Evidence. Human rejection by itself is not an incompatibility label, and evidence-free compatible rows are omitted rather than treated as clean negatives.
- V3 escalation is decision-context level. A current accepted Canonical mapping suppresses escalation even when the bounded candidate set also preserves stale/rejected alternatives. Stale accepted mappings, unresolved contexts, and contexts whose final mapping falls outside the recorded candidate set remain escalation cases.

These projections must not feed post-Accept source state, Validation/Conformance outcomes, adjudicator identity, teacher/model output, or future revisions back into the feature set.
