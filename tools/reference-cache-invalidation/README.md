# Dependency-based Cache Invalidation Reference

This directory is a **read-only reference** for a dependency-based cache
invalidation engine. It models World/Model/Artifact/Evidence revisions,
computes a deterministic dependency closure, and decides whether cached
artifacts, builds, and evidence can each be reused.

It is not a product cache, a distributed store, an authorization engine,
or an accepted API. It never writes a Source. It defaults to the safe side
when a dependency is unknown.

## Tracked revisions

| Kind | Meaning |
| --- | --- |
| `world` | World fingerprint (canonical semantic state) |
| `model` | Model revision (managed source index bytes) |
| `artifact` | Artifact revision (output artifact bytes) |
| `evidence` | Evidence revision (observation/audit record) |
| `source` | Source digest (raw input bytes) |
| `operator` | Operator pin |
| `plugin` | Plugin pin |
| `generator` | Generator version |

## Reuse decisions

The engine answers three questions independently:

1. **Artifact reuse** — may the existing artifact bytes be reused?
2. **Build skip** — may the build step be skipped entirely?
3. **Evidence reuse** — may the cached evidence be reused?

Artifact reuse requires source digest, model revision, world fingerprint,
artifact revision, and transitive dependency closure to match. Build skip
adds unchanged operator/plugin/generator pins and no dependency topology
change. Evidence reuse additionally requires scope, subject, and revision
to match, and rejects A→B→A cycles without continuity proof.

## Safety rules

- **Byte-only source match is insufficient.** World fingerprint, model
  revision, and dependency closure must also match.
- **Unknown dependencies deny reuse.** A missing or unresolved dependency
  is treated as stale.
- **Empty dependency set is valid only when explicit.** An empty closure
  is acceptable, but an unknown closure is not.
- **Mid-flight failure denies reuse.** A failed or incomplete build cannot
  produce a reusable artifact or evidence.
- **A→B→A needs continuity proof.** World fingerprint alone cannot prove
  semantic continuity after an intermediate change.

## Conditions verified

- Source change
- Dependency add/remove
- Operator/Plugin update
- Generator version change
- A→B→A cycle
- Partial retrieval (unknown dependencies)
- Empty set (no dependencies)
- Mid-flight failure

## Run

```sh
go run ./tools/reference-cache-invalidation/cmd/tei-cache-invalidation -input request.json
```

Input example:

```json
{
  "previous": {
    "sourceDigest": "sha256:abc",
    "modelRevision": "R1",
    "worldFingerprint": "sha256:wp1",
    "artifactRevision": "A1",
    "evidenceRevision": "E1",
    "operatorPin": "op-v1",
    "pluginPin": "pl-v1",
    "generatorVersion": "gen-v1"
  },
  "current": {
    "sourceDigest": "sha256:abc",
    "modelRevision": "R1",
    "worldFingerprint": "sha256:wp1",
    "artifactRevision": "A1",
    "evidenceRevision": "E1",
    "operatorPin": "op-v1",
    "pluginPin": "pl-v1",
    "generatorVersion": "gen-v1"
  },
  "dependencies": [
    {"from": "artifact", "to": "source", "scope": "default", "required": true},
    {"from": "artifact", "to": "model", "scope": "default", "required": true},
    {"from": "build", "to": "operator", "scope": "default", "required": true},
    {"from": "build", "to": "plugin", "scope": "default", "required": true},
    {"from": "build", "to": "generator", "scope": "default", "required": true}
  ],
  "currentRevisions": [
    {"kind": "source", "scope": "default", "digest": "sha256:abc"},
    {"kind": "model", "scope": "default", "digest": "sha256:mod1"},
    {"kind": "world", "scope": "default", "digest": "sha256:wp1"},
    {"kind": "artifact", "scope": "default", "digest": "sha256:art1"},
    {"kind": "evidence", "scope": "default", "digest": "sha256:ev1"},
    {"kind": "operator", "scope": "default", "digest": "sha256:op1"},
    {"kind": "plugin", "scope": "default", "digest": "sha256:pl1"},
    {"kind": "generator", "scope": "default", "digest": "sha256:gen1"}
  ],
  "previousRevisions": [
    {"kind": "source", "scope": "default", "digest": "sha256:abc"},
    {"kind": "model", "scope": "default", "digest": "sha256:mod1"},
    {"kind": "world", "scope": "default", "digest": "sha256:wp1"},
    {"kind": "artifact", "scope": "default", "digest": "sha256:art1"},
    {"kind": "evidence", "scope": "default", "digest": "sha256:ev1"},
    {"kind": "operator", "scope": "default", "digest": "sha256:op1"},
    {"kind": "plugin", "scope": "default", "digest": "sha256:pl1"},
    {"kind": "generator", "scope": "default", "digest": "sha256:gen1"}
  ],
  "evidence": {
    "scope": "default",
    "revision": "E1",
    "subject": "sales.order",
    "worldFingerprint": "sha256:wp1",
    "artifactRevision": "A1"
  },
  "previousEvidence": {
    "scope": "default",
    "revision": "E1",
    "subject": "sales.order",
    "worldFingerprint": "sha256:wp1",
    "artifactRevision": "A1"
  },
  "dependenciesChanged": false,
  "status": "complete"
}
```

## Status values

| Status | Meaning |
| --- | --- |
| `complete` | Build finished normally |
| `failed` | Build failed |
| `partial` | Build incomplete (mid-flight failure) |

A `partial` or `failed` status denies artifact and evidence reuse.

## Limits

- This is a read-only reference. It does not integrate with the product's
  artifact store, evidence router, or build planner.
- Dependency resolution is synthetic. The caller supplies the revision set
  and dependency graph.
- A→B→A continuity proof is represented as a boolean flag; the engine does
  not verify cryptographic continuity.
