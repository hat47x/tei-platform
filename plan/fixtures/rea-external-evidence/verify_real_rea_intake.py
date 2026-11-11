#!/usr/bin/env python3
"""Real-REA intake witness for the external analysis Evidence contract.

This is the reproducibility check for `internal/externalanalysis` and the
`tei rea-evidence` entry. It exists because the intake's claims are about bytes
that arrived: the same commands must produce the same bytes, and the same bytes
must be received the same way.

Two modes, chosen by what this environment can do:

* **real**: REA is installable here. The script installs the pinned version
  into an isolated prefix, analyzes an inert synthetic fixture, projects a
  bounded view, runs the negative requests the issue recorded, and then hands
  the saved Evidence to the built `tei` binary to receive, inspect and project.
  It records the digest of every captured byte and the exit status of every
  command.
* **fixture**: REA is not available here. The script verifies the same contract
  against the bytes captured under `internal/externalanalysis/testdata/` and
  records NOT RUN for the parts that need a live tool. It never claims a fresh
  run.

It is NOT a qualification of the analysis result, of TEI meaning acceptance, of
SHO, of a Target, or of any database observation.

Example:
 python3 -B plan/fixtures/rea-external-evidence/verify_real_rea_intake.py \
    --binary /tmp/tei --version 6.8.0
 python3 -B plan/fixtures/rea-external-evidence/verify_real_rea_intake.py \
    --binary /tmp/tei --fixtures-only
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile

FIXTURES = Path("internal/externalanalysis/testdata")
PINNED_VERSIONS = ("6.7.0", "6.8.0")

# An inert synthetic JavaScript package: index.js throws immediately if it is
# ever executed, so no analysis-target code runs.
PACKAGE_JSON = '{"name":"rea-intake","version":"1.0.0","type":"module","main":"index.js"}\n'
INDEX_JS = 'export const message = "hello";\nthrow new Error("Fixture must never be executed.");\n'

VIEW_REQUEST_VIEW = {"kind": "javascript-facts", "property_keys": [], "offset": 0, "limit": 32}


def require(condition, explanation):
    if not condition:
        raise AssertionError(explanation)


def digest(path):
    return hashlib.sha256(Path(path).read_bytes()).hexdigest()


def decode(raw):
    """Decode captured stdout. The same commands write UTF-16LE with a BOM on
    Windows and UTF-8 on Linux, so a verifier that assumes one encoding would
    fail on the other host."""
    if raw.startswith(b"\xff\xfe"):
        text = raw.decode("utf-16-le")
        return text[1:] if text.startswith("\ufeff") else text
    if raw.startswith(b"\xef\xbb\xbf"):
        return raw[3:].decode("utf-8")
    return raw.decode("utf-8")


def run(command, args, expect=0, stdout_path=None):
    """Run one command, optionally capturing stdout to a file. The captured
    bytes are kept as they arrived: the intake's digest covers them, so a
    verifier that re-encoded them would prove nothing."""
    argv = [str(command), *map(str, args)]
    if stdout_path is None:
        result = subprocess.run(argv, capture_output=True, check=False, timeout=900)
    else:
        with open(stdout_path, "wb") as stdout:
            result = subprocess.run(argv, stdout=stdout, stderr=subprocess.PIPE,
                                    check=False, timeout=900)
    require(result.returncode == expect,
            f"{Path(command).name} {' '.join(map(str, args))} exited "
            f"{result.returncode}, want {expect}: {result.stderr[:800]!r}")
    return result


def tei_json(binary, args, expect=0):
    result = run(binary, args, expect=expect)
    try:
        return json.loads(result.stdout.decode("utf-8"))
    except json.JSONDecodeError as exc:
        raise AssertionError(
            f"tei {' '.join(args)} did not answer JSON: {result.stdout[:600]!r}") from exc


def write_synthetic(directory):
    target = Path(directory) / "build" / "rea-intake"
    target.mkdir(parents=True, exist_ok=True)
    (target / "package.json").write_text(PACKAGE_JSON, encoding="utf-8")
    (target / "index.js").write_text(INDEX_JS, encoding="utf-8")
    return target


def derive_request(evidence_bytes, path):
    """Derive the view request from exactly the saved Evidence: source.kind
    inline with the Evidence record embedded, never the response wrapper."""
    envelope = json.loads(decode(evidence_bytes))
    require(envelope.get("ok") is True, "the saved analysis output must carry ok=true")
    evidence = envelope.get("data")
    require(isinstance(evidence, dict) and isinstance(evidence.get("evidence_id"), str),
            "the saved analysis output must carry the Evidence record under data")
    request = {"source": {"kind": "inline", "evidence": evidence}, "view": VIEW_REQUEST_VIEW}
    Path(path).write_text(json.dumps(request, ensure_ascii=False), encoding="utf-8")
    return evidence


def verify_common(binary, evidence_path, version, selection, report):
    """The contract every mode verifies: receive, inspect, derive, validate,
    project."""
    receipt = tei_json(binary, ["rea-evidence", "-analysis-output", str(evidence_path),
                                "-provider-version", version])
    body = receipt.get("result", {}).get("receipt", {})
    require(receipt.get("ok") is True, "the entry must answer ok for a whole receipt")
    # A whole receipt from an environment without the tool is still whole: the
    # caveat is recorded in the tool section, and the bytes are still the bytes.
    require(body.get("state") in ("complete", "tool-absent"),
            f"receipt state={body.get('state')}: {body.get('reason')}")
    if body.get("state") == "tool-absent":
        require(body.get("tool", {}).get("present") is False,
                "a tool-absent receipt must record the absence")
        require(body.get("tool", {}).get("absentReason"),
                "the absence needs a reason a reader can act on")
    require(isinstance(body.get("evidence", {}).get("id"), str),
            "the receipt must carry the Evidence ID")
    require(body.get("raw", {}).get("fingerprint", "").startswith("sha256:"),
            "the receipt must carry the digest of the bytes as received")
    require(body.get("promotion") == {"acceptedMeaning": False, "canonicalWorld": False,
                                      "databaseObservation": False},
            f"the receipt must promote nothing: {body.get('promotion')}")

    with tempfile.TemporaryDirectory() as directory:
        request_path = Path(directory) / "request.json"
        evidence_bytes = Path(evidence_path).read_bytes()
        evidence = derive_request(evidence_bytes, request_path)
        request = json.loads(request_path.read_text(encoding="utf-8"))
        require("ok" not in request["source"]["evidence"],
                "the derived request must not embed the response wrapper")
        require(request["source"]["evidence"] == evidence,
                "the derived request must embed the Evidence record as received")
        validated = tei_json(binary, ["rea-evidence", "-analysis-output", str(evidence_path),
                                      "-request", str(request_path),
                                      "-provider-version", version])
        require(validated.get("ok") is True,
                "a request derived from the Evidence must validate")

        candidates = tei_json(binary, ["rea-evidence", "-analysis-output", str(evidence_path),
                                       "-provider-version", version,
                                       "-observation-id", selection["nodeID"],
                                       "-relation-id", selection["relationID"]])
        view = candidates.get("result", {}).get("candidates")
        require(view is not None, "the candidate projection must answer")
        require(view["observation"]["nodeId"] == selection["nodeID"],
                "the projection must carry the producer's node identity")
        require(view["relation"]["relationId"] == selection["relationID"],
                "the projection must carry the producer's relation identity")
        for limit in ("external-candidate-is-not-accepted-meaning",
                      "external-identity-is-not-a-tei-semantic-subject"):
            require(limit in view["authorityLimits"],
                    f"the projection must state {limit}")
        report["checks"].append({"evidence": digest(evidence_path),
                                 "observation": selection["nodeID"],
                                 "relation": selection["relationID"]})


def selection_from_evidence(evidence_path):
    """Name one observation and one relation the Evidence actually carries.
    A version that states no relation identity has no selection to project, so
    the caller is told instead of being handed a derived handle it cannot use."""
    envelope = json.loads(decode(Path(evidence_path).read_bytes()))
    graph = envelope["data"]["normalized_result"]["semantic_graph"]
    nodes = graph.get("nodes") or []
    relations = graph.get("relations") or []
    require(nodes, "the Evidence carries no observation to select")
    relation_id = None
    for relation in relations:
        if relation.get("relation_id"):
            relation_id = relation["relation_id"]
            break
    return {"nodeID": nodes[0]["node_id"], "relationID": relation_id}


def verify_fixture_mode(binary, report):
    """Verify against the captured bytes when this environment has no tool."""
    manifest = json.loads((FIXTURES / "manifest.json").read_text(encoding="utf-8"))
    recorded = {entry["stdout"]: entry for entry in manifest["runs"]}
    for version in PINNED_VERSIONS:
        analysis = FIXTURES / f"rea-{version}" / "analysis.json"
        view = FIXTURES / f"rea-{version}" / "view.jsonl"
        require(analysis.exists() and view.exists(),
                f"the captured fixture for {version} is missing")
        require(digest(analysis) == recorded[f"rea-{version}/analysis.json"]["sha256"],
                f"the captured {version} Evidence no longer matches its recorded digest")
        selection = selection_from_evidence(analysis)
        if selection["relationID"] is None:
            report["notRun"].append(f"{version}: the producer states no relation identity")
            continue
        verify_common(binary, analysis, version, selection, report)
        inspected = tei_json(binary, ["rea-evidence", "-analysis-output", str(analysis),
                                      "-view", str(view), "-provider-version", version])
        body = inspected.get("result", {}).get("receipt", {})
        require(body.get("state") in ("complete", "tool-absent"),
                f"the captured {version} delivery must inspect as whole: "
                f"{body.get('state')} {body.get('reason')}")
    report["mode"] = "fixture"
    report["notRun"].extend(["live REA analysis", "live view projection",
                             "live negative requests"])
    return report


def verify_real_mode(binary, version, report, tool_prefix=None):
    """Run the real tool and verify the same contract against fresh bytes."""
    with tempfile.TemporaryDirectory() as workspace:
        if tool_prefix is None:
            prefix = Path(workspace) / "tool"
            run("npm", ["install", "--prefix", str(prefix), "--no-audit", "--no-fund",
                        "--ignore-scripts", "--save-exact", f"rea-agents@{version}"])
        else:
            prefix = Path(tool_prefix)
        rea = prefix / "node_modules" / ".bin" / ("rea.cmd" if os.name == "nt" else "rea")
        require(rea.exists(), f"rea {version} was not found in {prefix}")

        target = write_synthetic(workspace)
        before = {path.name: digest(path) for path in target.iterdir()}
        analysis = Path(workspace) / "analysis.json"
        run(rea, ["analyze-javascript-application", target, "--json", "--full-output"],
            stdout_path=analysis)
        after = {path.name: digest(path) for path in target.iterdir()}
        require(before == after, "the analysis must not have changed the target bytes")

        request_path = Path(workspace) / "request.json"
        derive_request(analysis.read_bytes(), request_path)
        view = Path(workspace) / "view.jsonl"
        run(rea, ["inspect-analysis-view", request_path, "--format", "jsonl"],
            stdout_path=view)

        payload = decode(view.read_bytes())
        records = [json.loads(line) for line in payload.split("\n") if line.strip()]
        control = [record for record in records if record.get("$rea") == "jsonl/v1"]
        require(control or len(records) == 1,
                "a delivery is either rows plus a terminal or one document")
        if control:
            terminal = control[-1]
            require(terminal.get("type") == "metadata",
                    f"the terminal record must be metadata: {terminal.get('type')}")

        selection = selection_from_evidence(analysis)
        if selection["relationID"] is None:
            report["notRun"].append(f"{version}: the producer states no relation identity")
        else:
            verify_common(binary, analysis, version, selection, report)
        inspected = tei_json(binary, ["rea-evidence", "-analysis-output", str(analysis),
                                      "-view", str(view), "-provider-version", version])
        body = inspected.get("result", {}).get("receipt", {})
        require(body.get("state") in ("complete", "tool-absent"),
                f"the fresh {version} delivery must inspect as whole: "
                f"{body.get('state')} {body.get('reason')}")

        # The negative requests the issue recorded: a source.kind this version
        # does not accept, and the response wrapper standing in for the
        # Evidence. Both must be refused by the tool, not by the intake.
        for name, source in (("kind", {"kind": "evidence-file", "path": str(analysis)}),
                             ("envelope", {"kind": "inline",
                                           "evidence": json.loads(decode(analysis.read_bytes()))})):
            negative = Path(workspace) / f"negative-{name}.json"
            negative.write_text(json.dumps({"source": source, "view": VIEW_REQUEST_VIEW}),
                                encoding="utf-8")
            result = run(rea, ["inspect-analysis-view", negative, "--format", "jsonl"],
                         expect=1)
            document = json.loads(decode(result.stdout))
            # A refusal is either an explicit ok=false control record or a
            # diagnostic document without one; what it must never be is a
            # successful projection of the negative request.
            require(document.get("ok") is not True,
                    f"the tool must refuse the {name} negative")
            require("error" in document or "code" in document
                    or document.get("$rea") == "jsonl/v1",
                    f"the {name} refusal must carry a diagnostic")
            # The diagnostic is either at the top level or under result,
            # depending on the version's framing.
            diagnostic = document.get("result", document)
            report["checks"].append({"negative": name,
                                     "code": diagnostic.get("code"),
                                     "category": diagnostic.get("category")})
        # The digests are taken while the captured bytes still exist: the
        # record is the evidence that these bytes were received, and a digest
        # taken after the workspace is gone would prove nothing.
        report["captured"] = {"analysis": digest(analysis), "view": digest(view)}
    report["mode"] = "real"
    return report


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--binary", required=True, help="built tei executable")
    parser.add_argument("--version", default="6.8.0", choices=PINNED_VERSIONS)
    parser.add_argument("--fixtures-only", action="store_true",
                        help="verify against the captured bytes without installing REA")
    parser.add_argument("--tool-prefix",
                        help="reuse an already-installed rea-agents prefix instead of installing one")
    parser.add_argument("--report", help="write the run record here as JSON")
    args = parser.parse_args()

    report = {"version": args.version, "binary": str(args.binary), "checks": [],
              "notRun": []}
    if args.fixtures_only:
        verify_fixture_mode(args.binary, report)
    else:
        verify_real_mode(args.binary, args.version, report, tool_prefix=args.tool_prefix)
    if args.report:
        Path(args.report).write_text(
            json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print(json.dumps({"mode": report["mode"], "version": report["version"],
                      "checks": len(report["checks"]),
                      "notRun": report["notRun"]}, ensure_ascii=False))
    return 0


if __name__ == "__main__":
    sys.exit(main())
