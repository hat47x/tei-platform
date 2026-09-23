# TEI-LD-0A corpus v0 — no-training checkpoint

This work unit implements the bounded synthetic corpus mechanics tracked by #876.

## Current evidence status

- The fixture rows are synthetic mechanics fixtures only.
- They are not historical Proposal/Acceptance Evidence.
- They are not training data and must not be used to fit logistic, LightGBM, Jev, LLM, or another learned model.
- No teacher/provider/model output is captured by this corpus v0.
- No row can write Canonical or authoritative source state.

The corpus exists only to exercise schema, identity, validation, candidate-set hashing, split/leakage checks, and derived V1-V3 research views.

## Entry condition for historical evidence

Do not label historical rows as Evidence until #267 provides actual Proposal/Acceptance records with enough provenance to bind:

- source and base revisions;
- candidate identities and candidate-set identity;
- Proposal/decision provenance;
- reload and Validation/Conformance linkage where applicable.

If those records are absent, improve the upstream Evidence path rather than inventing labels.

## Branch boundary

This is the single TEI-LD-0A corpus work unit. After its repository health gate passes and it is merged to `main`, delete or realign `research/tei-ld-0a-corpus-v0` before any teacher capture or model fitting. Do not create a parallel model-training branch from this checkpoint.
