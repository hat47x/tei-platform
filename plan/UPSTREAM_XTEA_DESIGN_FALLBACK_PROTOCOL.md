# TEI Lane C — X-TEA Design Fallback Protocol

- Status: Lane C working rule
- Purpose: define how TEI upstream-design questions are checked against the X-TEA / Kozo Watanabe design lineage before inventing new concepts
- Scope: design reasoning and research order; this is not a new semantic authority, Product Requirement, or runtime contract

## 1. Core rule

TEI's upstream design starts from a real lineage: Kozo Watanabe's business-system design practice, the Three-Element Analysis method, X-TEA Modeler / Driver, and the Reference Models.

When Lane C encounters an ambiguous design question, the default is **not** to invent a new TEI abstraction immediately.

Use this loop first:

```text
TEI design question
      ↓
What business-design problem are we actually trying to preserve?
      ↓
Kozo Watanabe's explicit statements / books / articles
      ↓
X-TEA Modeler / Driver manuals and visible specification structure
      ↓
X-TEA Modeler / Driver source code
      ↓
Reference Model usage across realistic content
      ↓
compare with current TEI Principle / Requirement / Reference / Evidence
      ↓
translate vitality, do not mechanically copy shape
      ↓
small TEI Reference experiment
      ↓
accept / reject / defer new concept
```

This is a **design fallback order**, not a legal precedence hierarchy and not a replacement for TEI's current Product Requirements.

## 2. What each source answers

### 2.1 Watanabe's statements answer “why?”

Use books, articles, talks, and the author's own explanations to recover the intended design problem.

Typical questions:

- Why separate Data / Function / Business views?
- Why start from Data?
- Why is UI important in Function design?
- Why treat a business system as a social system?
- Why publish Reference Models?
- Why make specifications executable?

Do not reduce an author's design claim to a literal TEI schema requirement.

### 2.2 Manuals / Modeler structure answer “what did the method ask designers to describe?”

Use manuals, UI structure, exported documents, matrices, and model-file structure to identify the explicit design vocabulary and cross-references.

Examples already observed in X-TEA / XEAD lineage:

- Subsystem
- SubjectArea
- Table / Field / Relationship
- Function
- Task
- TaskAction
- TaskFunctionIO
- Instance
- matrices and generated documents

The existence of a noun is evidence that the viewpoint mattered. It is **not** sufficient evidence that TEI needs the same top-level noun.

### 2.3 Source code answers “what did the tool actually preserve or enforce?”

When documentation is ambiguous, inspect the OSS implementation to establish actual relationships and behavior.

Examples confirmed in the public `xead/XeadModeler` source:

- `TaskAction` is represented separately from Function definitions.
- `TaskFunctionIO` explicitly connects a Task action to a Function.
- Subsystem and SubjectArea are both present in model / matrix handling.
- Instance text and `SortKey` are independently retained; `SortKey` must not automatically be interpreted as business time.

Modeler and Driver repositories currently expose BSD-3-Clause style licenses. Their source can therefore be studied and adapted within those license conditions. Reference Model content must be audited separately rather than assumed to have the same license.

### 2.4 Reference Models answer “does the idea survive realistic business content?”

Reference Models are the strongest pressure against method purity.

They answer questions such as:

- Does a proposed distinction occur repeatedly in real models?
- Does the platform force business content into unnatural shapes?
- Are there business actions with no Application Function realization?
- Are states represented by one status field or derived from multiple facts?
- Do cross-subsystem reads / updates reveal ownership boundaries?
- Does a model need history / ledger patterns rather than current-state rows only?

Past X-TEA corpus analysis already found a material example:

```text
12 XEA[D] models
TaskAction total: 1606
TaskActions with no Application Function realization: 202
```

Therefore Lane C must preserve:

```text
Business Task / Action != Application Function
```

A manual, physical, external-company, human-only, or otherwise non-application action is valid business design information.

## 3. Conflict-resolution rule

Different sources may disagree because X-TEA evolved over time, documents describe intent while source reflects implementation compromises, or Reference Models use a facility in more than one way.

Do not force a synthetic single answer.

Classify the conflict:

```text
A. intent vs implementation difference
B. historical version difference
C. method rule vs content exception
D. UI / presentation convention vs semantic meaning
E. unresolved contradiction
```

Then preserve the distinction in the TEI design note.

Examples:

- If a source field called `SortKey` is used both for state progression and presentation order, TEI must **split the concepts**, not call SortKey a timeline.
- If a Reference Model action has no Function, TEI must not invent a dummy Capability merely to make the graph complete.
- If implementation allows something that the design method warns against, treat it as observed behavior, not automatically as recommended design.

## 4. Translation rule: preserve vitality, not historical shape

For every X-TEA concept considered for TEI, answer these questions in order:

1. What design failure does this concept prevent or expose?
2. Can current TEI express that failure without a new Canonical concept?
3. Can a derived Projection / matrix / diagnostic provide the missing viewpoint?
4. Is the need only Provider / Target / Editor specific?
5. Does the distinction appear across more than one realistic Reference?
6. Does collapsing it lose business meaning, responsibility, lifecycle, or traceability?
7. Can a small executable / reviewable Reference demonstrate the need?

Only then consider a Core / Canonical addition.

## 5. Current mapping guardrails

These mappings are hypotheses, not frozen one-to-one translations.

| X-TEA / Three-Element viewpoint | Current TEI candidate | Guardrail |
| --- | --- | --- |
| Data Model | Data + Relation + Validation + lifecycle | do not reduce Data to DB schema |
| Function Model | Application Pattern / View + Action / Capability + Query + CRUD/call projection | do not reduce Function to backend API |
| Business Model | business Procedure / Task view + actor + manual / external steps + references to Application meaning | do not reduce Task to Function or Workflow Engine node |
| Subsystem | ownership / cohesion / effective CUD / exposed Action boundary projection | do not equate with deployable service |
| SubjectArea | business comprehension / data-domain projection candidate | do not equate automatically with namespace |
| Instance | Dataset / Snapshot / Scenario / Expectation / Presentation Order after semantic split | do not infer time from display sort order |
| Table Script | Action / Rule / Consequence / provider realization depending on meaning | do not expose raw hook as public business Capability |

## 6. Evidence grades for Lane C

When citing X-TEA as a design reason, mark the evidence nature.

```text
W1  explicit author statement
M1  manual / documented model structure
S1  public source-code behavior
R1  one Reference Model example
R2  repeated pattern across multiple Reference Models
T1  TEI Reference experiment reproduces the pressure
C1  TEI Conformance / diagnostic encodes the accepted boundary
```

A new TEI Canonical concept should normally not be justified by `R1` or `S1` alone.

Strong candidates should accumulate at least:

```text
intent evidence (W1/M1)
+ content pressure (R1/R2)
+ TEI-local proof (T1)
```

before becoming a durable semantic contract.

## 7. Source and licensing discipline

### Public OSS source

The public repositories `xead/XeadModeler` and `xead/XeadDriver` contain BSD-3-Clause style license text. Source inspection is therefore an ordinary and repeatable Lane C research route, subject to those license terms.

### Reference Model content

Do not assume that every downloadable model / attached database / manual shares the OSS source license.

Until the exact license of a Reference artifact is confirmed:

- inspect and summarize structure
- record counts / pattern evidence
- reconstruct independent TEI test content
- do not copy large model contents into `tei-platform`
- preserve provenance to the original model / version

TALON's public reproduction of `CONCEPTWARE／受注生産` is useful corroborating evidence that the model is intended as a practical, executable learning/reference artifact, but its own distribution terms remain distinct from X-TEA's source-code license.

## 8. Decision log template

When Lane C reaches a material ambiguity, record:

```text
Question:
Current TEI behavior:
Business-design risk:

Watanabe intent evidence:
X-TEA manual/model-structure evidence:
X-TEA source evidence:
Reference Model evidence:
TEI-local Reference evidence:

Conflicts / version differences:
Candidate translations:
Rejected historical-shape copies:
Unresolved:
Decision:
Revisit trigger:
```

The decision should be traceable without requiring the original chat conversation.

## 9. Immediate application

The first use of this protocol is the `order_entry` Business Procedure experiment.

The key question is:

> Does TEI need a durable lightweight Procedure identity, or can the Business Model viewpoint remain a derived projection over existing Semantic / Application meaning?

X-TEA evidence already prevents one invalid shortcut:

> Do not model every business step as an Application Pattern / Capability merely so the Procedure becomes executable.

The experiment must include at least one manual / external / unknown-realization business action and keep it valid without an Application Function.
