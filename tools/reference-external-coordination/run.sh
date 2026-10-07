#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TMP_ROOT="$(mktemp -d)"
STORE="$TMP_ROOT/coordination-store"
WORK_ID="TW-0001"
FOLLOW_UP_ID="TW-0002"

cleanup() {
  rm -rf "$TMP_ROOT"
}
trap cleanup EXIT

if ! git -C "$ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "external-coordination reference requires a Git checkout" >&2
  exit 2
fi

if [[ -n "$(git -C "$ROOT" status --porcelain)" ]]; then
  echo "external-coordination reference requires a clean product checkout" >&2
  git -C "$ROOT" status --short >&2
  exit 2
fi

PRODUCT_HEAD_BEFORE="$(git -C "$ROOT" rev-parse HEAD)"

mkdir -p "$STORE/work" "$STORE/results/$WORK_ID" "$STORE/artifacts/$WORK_ID"
git -C "$STORE" init -q
git -C "$STORE" config user.name "TEI Coordination Reference"
git -C "$STORE" config user.email "coordination-reference@tei.invalid"

cat > "$STORE/work/$WORK_ID.json" <<EOF_WORK
{
  "exchangeVersion": "reference-v0",
  "workRef": {"backend": "git-text", "id": "$WORK_ID"},
  "origin": {"kind": "human", "actorRef": "human-reference"},
  "createdVia": {"kind": "automation", "actorRef": "reference-runner"},
  "objective": "Verify that external coordination remains outside TEI product state.",
  "sourceContext": {
    "repositoryRef": "local-product-checkout",
    "baseRevision": "$PRODUCT_HEAD_BEFORE",
    "sourceRefs": []
  },
  "constraints": ["do-not-mutate-product-source"],
  "contextRefs": [],
  "expectedOutputs": ["reference.evidence"],
  "coordination": {"status": "ready"}
}
EOF_WORK

git -C "$STORE" add "work/open/$WORK_ID.json"
git -C "$STORE" commit -qm "coordination: create human work"
WORK_COMMIT="$(git -C "$STORE" rev-parse HEAD)"

printf 'current evidence\n' > "$STORE/artifacts/$WORK_ID/run-a.txt"
ARTIFACT_DIGEST="$(python3 - "$STORE/artifacts/$WORK_ID/run-a.txt" <<'PY'
import hashlib
import pathlib
import sys

print(hashlib.sha256(pathlib.Path(sys.argv[1]).read_bytes()).hexdigest())
PY
)"

cat > "$STORE/results/$WORK_ID/run-a.json" <<EOF_RESULT_A
{
  "exchangeVersion": "reference-v0",
  "workRef": {"backend": "git-text", "id": "$WORK_ID"},
  "producer": {"kind": "agent", "actorRef": "agent-a", "externalRunRef": "run-a"},
  "observedContext": {"baseRevision": "$PRODUCT_HEAD_BEFORE"},
  "result": {
    "kind": "reference.evidence",
    "summary": "Current-base result",
    "artifactRefs": [
      {
        "uri": "artifacts/$WORK_ID/run-a.txt",
        "digest": {"algorithm": "sha256", "value": "$ARTIFACT_DIGEST"},
        "mediaType": "text/plain"
      }
    ],
    "sourceRefs": [],
    "evidenceRefs": [],
    "followUpRefs": [{"backend": "git-text", "id": "$FOLLOW_UP_ID"}]
  }
}
EOF_RESULT_A

cat > "$STORE/results/$WORK_ID/run-b.json" <<'EOF_RESULT_B'
{
  "exchangeVersion": "reference-v0",
  "workRef": {"backend": "git-text", "id": "TW-0001"},
  "producer": {"kind": "agent", "actorRef": "admin", "externalRunRef": "run-b"},
  "observedContext": {"baseRevision": "0000000000000000000000000000000000000000"},
  "result": {
    "kind": "reference.evidence",
    "summary": "Intentionally stale result with a self-asserted actor label",
    "artifactRefs": [],
    "sourceRefs": [],
    "evidenceRefs": [],
    "followUpRefs": []
  }
}
EOF_RESULT_B

cat > "$STORE/work/$FOLLOW_UP_ID.json" <<EOF_FOLLOW_UP
{
  "exchangeVersion": "reference-v0",
  "workRef": {"backend": "git-text", "id": "$FOLLOW_UP_ID"},
  "origin": {"kind": "agent", "actorRef": "agent-a"},
  "createdVia": {"kind": "agent", "actorRef": "agent-a"},
  "parentRef": {"backend": "git-text", "id": "$WORK_ID"},
  "objective": "Independently review the current-base result.",
  "sourceContext": {
    "repositoryRef": "local-product-checkout",
    "baseRevision": "$PRODUCT_HEAD_BEFORE",
    "sourceRefs": []
  },
  "constraints": ["do-not-mutate-product-source"],
  "contextRefs": [],
  "expectedOutputs": ["reference.review"],
  "coordination": {"status": "ready"}
}
EOF_FOLLOW_UP

python3 - "$STORE" "$PRODUCT_HEAD_BEFORE" <<'PY'
import hashlib
import json
import pathlib
import sys

store = pathlib.Path(sys.argv[1])
product_head = sys.argv[2]
work = json.loads((store / "work/TW-0001.json").read_text())
follow_up = json.loads((store / "work/TW-0002.json").read_text())
results = [
    json.loads((store / "results/TW-0001/run-a.json").read_text()),
    json.loads((store / "results/TW-0001/run-b.json").read_text()),
]

assert work["origin"]["kind"] == "human"
assert work["createdVia"] == {"kind": "automation", "actorRef": "reference-runner"}
assert work["sourceContext"]["baseRevision"] == product_head
assert work["coordination"]["status"] == "ready"
assert follow_up["origin"] == {"kind": "agent", "actorRef": "agent-a"}
assert follow_up["createdVia"] == {"kind": "agent", "actorRef": "agent-a"}
assert follow_up["parentRef"] == work["workRef"]
assert follow_up["sourceContext"]["baseRevision"] == product_head
assert len(results) == 2
assert {result["producer"]["externalRunRef"] for result in results} == {"run-a", "run-b"}
assert all(result["workRef"]["id"] == work["workRef"]["id"] for result in results)
assert results[0]["result"]["followUpRefs"] == [follow_up["workRef"]]

current = [result for result in results if result["observedContext"]["baseRevision"] == product_head]
stale = [result for result in results if result["observedContext"]["baseRevision"] != product_head]
assert len(current) == 1
assert len(stale) == 1
assert stale[0]["producer"]["actorRef"] == "admin"

artifact = results[0]["result"]["artifactRefs"][0]
artifact_path = store / artifact["uri"]
digest = hashlib.sha256(artifact_path.read_bytes()).hexdigest()
assert artifact["digest"] == {"algorithm": "sha256", "value": digest}

for document in [work, follow_up, *results]:
    rendered = json.dumps(document, sort_keys=True).lower()
    for forbidden in (
        "tokenbudget",
        "quota",
        "subscriptioncapacity",
        "credential",
        "privatekey",
        "agentsession",
        "checkpoint",
        "canonicalaccepted",
        "runtimeauthorized",
        "acceptanceauthority",
        "resourceauthority",
    ):
        assert forbidden not in rendered, forbidden
PY

git -C "$STORE" add "results/$WORK_ID" "artifacts/$WORK_ID" "work/$FOLLOW_UP_ID.json"
git -C "$STORE" commit -qm "coordination: append independent agent results"
RESULTS_COMMIT="$(git -C "$STORE" rev-parse HEAD)"

python3 - "$STORE/work/$WORK_ID.json" <<'PY'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
document = json.loads(path.read_text())
document["coordination"]["status"] = "completed"
path.write_text(json.dumps(document, ensure_ascii=False, indent=2) + "\n")
PY

git -C "$STORE" add "work/$WORK_ID.json"
git -C "$STORE" commit -qm "coordination: mark work completed"
COMPLETION_COMMIT="$(git -C "$STORE" rev-parse HEAD)"

if [[ "$(git -C "$ROOT" rev-parse HEAD)" != "$PRODUCT_HEAD_BEFORE" ]]; then
  echo "coordination activity changed the product revision" >&2
  exit 1
fi

if [[ -n "$(git -C "$ROOT" status --porcelain)" ]]; then
  echo "coordination activity changed the product checkout" >&2
  git -C "$ROOT" status --short >&2
  exit 1
fi

STORE_HEAD="$(git -C "$STORE" rev-parse HEAD)"
rm -rf "$STORE"

if [[ "$(git -C "$ROOT" rev-parse HEAD)" != "$PRODUCT_HEAD_BEFORE" ]]; then
  echo "deleting the coordination store changed the product revision" >&2
  exit 1
fi

if [[ -n "$(git -C "$ROOT" status --porcelain)" ]]; then
  echo "deleting the coordination store changed the product checkout" >&2
  git -C "$ROOT" status --short >&2
  exit 1
fi

printf 'product=%s\n' "$PRODUCT_HEAD_BEFORE"
printf 'work-commit=%s\n' "$WORK_COMMIT"
printf 'results-commit=%s\n' "$RESULTS_COMMIT"
printf 'completion-commit=%s\n' "$COMPLETION_COMMIT"
printf 'deleted-store-head=%s\n' "$STORE_HEAD"
printf 'duplicate-results=2\n'
printf 'human-origin-work=1\n'
printf 'agent-origin-follow-up=1\n'
printf 'stale-result=preserved-not-applied\n'
printf 'authority-fields=absent\n'
printf 'self-asserted-actor=non-authoritative\n'
printf 'coordination-store-deletion=product-unchanged\n'
printf 'external coordination git-text reference: PASS\n'
