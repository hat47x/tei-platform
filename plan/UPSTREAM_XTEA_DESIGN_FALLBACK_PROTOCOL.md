# TEI Lane C — External Design Reference Protocol

- Status: Lane C working rule
- Purpose: define how upstream-design questions are checked against relevant public design references before introducing new TEI concepts
- Scope: design reasoning and research order; this is not a semantic authority, Product Requirement, compatibility promise, or statement of project lineage

## 1. Core rule

When Lane C encounters an ambiguous design question, the default is not to introduce a new TEI abstraction immediately.

For questions related to business-system modeling, X-TEA / XEAD and Kozo Watanabe's published work may be used as **independent external references**, alongside other relevant sources.

```text
TEI design question
      ↓
What business-design problem are we trying to preserve?
      ↓
relevant public design references
      ↓
manual / model structure / public source when useful
      ↓
compare with TEI Principle / Requirement / Reference / Evidence
      ↓
small TEI Reference experiment
      ↓
accept / reject / defer new concept
```

TEI does not claim to be an official successor or derivative of X-TEA / XEAD. External references provide questions and evidence; TEI Product Requirements, accepted Architecture decisions, Reference results, and Conformance evidence determine TEI behavior.

## 2. What different source types can establish

### Published explanations

Use an author's published explanations to understand the design problem being discussed. Do not turn an author's wording or presumed intention into a TEI schema requirement.

Typical questions include why multiple views are separated, why UI belongs to Function design, why manual work remains part of business-system design, and why executable specifications or Reference Models can help validation.

### Manuals and model structure

Manuals, UI structure, exported documents, matrices, and model-file structure can show what a method or tool asks designers to describe. The existence of a historical model noun is not sufficient evidence that TEI needs the same top-level noun.

### Public source code

When documentation is ambiguous, public source can establish actual represented relationships and behavior. Implementation behavior is evidence of implementation, not automatically a recommendation for TEI.

### Reference content

Reference content is useful for checking whether a distinction survives realistic business pressure. It can reveal, for example, business actions without Application Function realization, cross-boundary data dependencies, state derived from multiple facts, or history-oriented patterns.

Where reuse terms are uncertain, inspect and summarize structure without copying the original content into TEI.

## 3. Conflict-resolution rule

Different sources can disagree because versions changed, documentation describes a goal while implementation reflects constraints, or content uses a facility in several ways.

Classify rather than erase the difference:

```text
A. explanation vs implementation
B. historical version difference
C. method rule vs content exception
D. presentation convention vs semantic meaning
E. unresolved contradiction
```

TEI should preserve the relevant distinction and test it locally rather than force a synthetic historical answer.

## 4. Translation rule: preserve the design pressure, not the historical shape

For an external concept considered for TEI, ask:

1. What design failure does this concept prevent or expose?
2. Can current TEI express that failure without a new Canonical concept?
3. Can a derived Projection / matrix / diagnostic provide the missing viewpoint?
4. Is the need Provider / Target / Editor specific?
5. Does the distinction appear across realistic References?
6. Does collapsing it lose business meaning, responsibility, lifecycle, or traceability?
7. Can a small TEI Reference demonstrate the need?

Only then consider a Core / Canonical addition.

## 5. Current X-TEA / XEAD comparison guardrails

These are comparison hypotheses, not one-to-one translations or compatibility commitments.

| External viewpoint | Current TEI candidate | Guardrail |
| --- | --- | --- |
| Data Model | Data + Relation + Validation + lifecycle | do not reduce Data to DB schema |
| Function Model | Application Pattern / View + Action / Capability + Query + CRUD/call projection | do not reduce Function to backend API |
| Business Model | Procedure / Task view + actor + manual / external steps + references to Application meaning | do not reduce Task to Function or Workflow Engine node |
| Subsystem | ownership / cohesion / effective CUD / exposed Action boundary projection | do not equate with deployable service |
| SubjectArea | business comprehension / data-domain projection candidate | do not equate automatically with namespace |
| Instance | Dataset / Snapshot / Scenario / Expectation / Presentation Order after semantic split | do not infer time from display sort order |
| Table Script | Action / Rule / Consequence / provider realization depending on meaning | do not expose raw hook as public business Capability |

## 6. Evidence grades

When an external reference materially affects a TEI design discussion, record the nature of the evidence.

```text
P1  published explanation
M1  manual / documented model structure
S1  public source-code behavior
R1  one Reference example
R2  repeated pattern across multiple References
T1  TEI Reference experiment reproduces the pressure
C1  TEI Conformance / diagnostic encodes the accepted boundary
```

A durable TEI semantic contract should not normally be justified by one external observation alone. External evidence should be followed by TEI-local proof.

## 7. Source and licensing discipline

For public OSS source, follow the license attached to the exact repository and revision used. Do not infer that manuals, downloadable models, databases, or third-party reproductions share the same terms.

Until the reuse terms of a Reference artifact are confirmed:

- inspect and summarize structure where permitted
- record independently derived counts or pattern evidence
- recreate TEI test content independently
- do not copy substantial model contents into `tei-platform`
- preserve enough provenance to identify the source without redistributing it

## 8. Decision-note template

```text
Question:
Current TEI behavior:
Business-design risk:
External evidence:
TEI-local Reference evidence:
Conflicts / version differences:
Candidate translations:
Unresolved:
Decision:
Revisit trigger:
```

A decision should be understandable from Repository-visible evidence without depending on a private conversation or temporary handoff note.

## 9. Immediate application

The `order_entry` Business Procedure experiment asks whether TEI needs durable lightweight Procedure identity or whether the Business viewpoint can remain a traceable projection over existing Semantic / Application meaning.

The experiment must preserve manual / external / unresolved business actions without inventing an Application Pattern or Capability merely to make the Procedure executable.
