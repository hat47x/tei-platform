#!/usr/bin/env python3
"""Real-CLI positive/negative pair for the Provider Source currency limit (#3262).

For the public fixture only, build disposable consumer+provider workspaces in a
temporary directory, capture the 13 replies of the existing read-only TEI CLI
exactly as the workflow does, and judge them with context_gate.load/assess.

Counterexample shown here: when the bounded Function closure depends on a type
declared by an imported Provider workspace, changing only that Provider's raw
Source or Manifest bytes does NOT change the consumer's worldFingerprint or
sourceIndexFingerprint, so the gate must answer "unknown" and withhold
fingerprints. This is NOT a Provider occurrence check, NOT an atomic multi-source
snapshot, and invokes no Provider. Nothing is written under --root.
"""
from __future__ import annotations

import argparse
import hashlib
import json
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile

sys.path.insert(0, str(Path(__file__).resolve().parent))
import context_gate  # noqa: E402  (same directory; reuse COMMANDS/load/assess)

SCHEMA = "tei.reference.design-context-provider-currency/v0"
ISSUE = "hat47x/tei-platform#3262"
WORKSPACE = Path("reference/workspace")
ORDER_SOURCE = "semantic/sales/order.tei"
OLD_ATTRIBUTE = "attribute externalId type std.text"
NEW_ATTRIBUTE = "attribute externalId type os.process_id"
UNKNOWN_ENTRY = "currency-dependency:os.process_id:provider-source-occurrence-unknown"

PROVIDER_MANIFEST = "workspace: provider\nformatVersion: 1\nsources:\n  - semantic\n"
PROVIDER_SOURCE = ("namespace os\n\ntype ProcessId from std.identifier\n\n"
                   "data Process {\n    attribute id type os.process_id\n}\n")
IMPORT_DECLARATION = "imports:\n  - alias: os\n    path: ../provider\n    namespace: os\n"

# Replies that must be byte-identical when only Provider bytes change.
MUST_BE_IDENTICAL = ("catalog", "inspect", "source-index-all", "validate")

LIMITS = [
    "not a provider occurrence check: the existing CLI exposes no import/provider occurrence",
    "no atomic multi-source snapshot: 13 separate CLI invocations",
    "no Provider invocation",
    "no disclosure/grant evaluation",
    "no transitive/workspace-wide closure: fixed one-hop closure of sales.order and sales.order_line only",
    "test generation not qualified",
]


def digest(data: bytes) -> str:
    return hashlib.sha256(data).hexdigest()


def run_cli(cli: Path, cwd: Path, *args: str) -> bytes:
    result = subprocess.run([str(cli), *args], cwd=cwd, capture_output=True,
                            timeout=120, check=False)
    if result.returncode != 0:
        raise AssertionError(
            f"TEI command failed: {args[:3]} status={result.returncode}, "
            f"stderr={result.stderr[-550:]!r}")
    return result.stdout


def capture(cli: Path, scratch: Path, workspace: Path, folder: Path) -> None:
    """The exact 13 invocations of the workflow step; raw stdout bytes to <name>.json."""
    w = str(workspace)
    invocations = {
        "commands": ["commands"],
        "version": ["version", "-json"],
        "workspace-files": ["workspace-files", "-workspace", w],
        "catalog": ["model", "catalog", "-workspace", w],
        "inspect": ["inspect", "-workspace", w, "-subject-kind", "semantic-data",
                    "-subject-id", "sales.order"],
        "inspect-validation": ["inspect", "-workspace", w, "-subject-kind", "semantic-validation",
                               "-subject-id", "sales.order_line.line_number_positive"],
        "inspect-capability": ["inspect", "-workspace", w, "-subject-kind", "semantic-capability",
                               "-subject-id", "sales.order.submit"],
        "inspect-application": ["inspect", "-workspace", w, "-subject-kind", "application",
                                "-subject-id", "order_entry"],
        "source-index": ["source-index", "-workspace", w, "-subject-kind", "semantic-data",
                         "-subject-id", "sales.order"],
        "source-index-all": ["source-index", "-workspace", w],
        "trace": ["trace", "-workspace", w, "-scope", "reference-observation",
                  "-subject-kind", "semantic-relation", "-subject-id", "sales.order.customer",
                  "-direction", "both", "-mode", "summary"],
        "requirements": ["requirements", "-workspace", w],
        "validate": ["validate", "-workspace", w, "-json"],
    }
    if set(invocations) != set(context_gate.COMMANDS):
        raise AssertionError("capture list differs from context_gate.COMMANDS")
    folder.mkdir(parents=True)
    for name in context_gate.COMMANDS:
        (folder / (name + ".json")).write_bytes(run_cli(cli, scratch, *invocations[name]))


def judge(folder: Path) -> dict:
    docs, failures, _ = context_gate.load(folder)
    if failures:
        raise AssertionError(f"reply load failures: {failures}")
    return context_gate.assess(docs, failures)


def catalog_fingerprints(folder: Path) -> dict[str, str]:
    result = json.loads((folder / "catalog.json").read_bytes())["result"]
    values = {k: result.get(k) for k in ("worldFingerprint", "sourceIndexFingerprint")}
    if not all(isinstance(v, str) and v.startswith("sha256:") for v in values.values()):
        raise AssertionError(f"catalog fingerprints missing: {values}")
    return values


def check(condition: bool, message: str) -> None:
    if not condition:
        raise AssertionError(message)


def write_text(path: Path, text: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(text.encode("utf-8"))  # exact LF bytes


def append_bytes(path: Path, data: bytes) -> None:
    path.write_bytes(path.read_bytes() + data)


def build_p1(root: Path, case: Path) -> Path:
    consumer = case / "consumer"
    shutil.copytree(root / WORKSPACE, consumer)
    provider = case / "provider"
    write_text(provider / "tei.workspace.yaml", PROVIDER_MANIFEST)
    write_text(provider / "semantic/os/process.tei", PROVIDER_SOURCE)
    manifest = consumer / "tei.workspace.yaml"
    text = manifest.read_bytes().decode("utf-8")
    if not text.endswith("\n"):
        text += "\n"
    write_text(manifest, text + IMPORT_DECLARATION)
    return consumer


def apply_n1(consumer: Path) -> None:
    path = consumer / ORDER_SOURCE
    text = path.read_bytes().decode("utf-8")
    check(text.count(OLD_ATTRIBUTE) == 1, "expected exactly one externalId attribute to replace")
    write_text(path, text.replace(OLD_ATTRIBUTE, NEW_ATTRIBUTE))


def admission_would_pass(context: dict) -> bool:
    """Same predicate as t1_line_number_candidate.py main() (lines ~130-141),
    reimplemented here on purpose; that module is not imported."""
    return bool(
        context.get("schema") == "tei.reference.design-context-readiness/v0"
        and context.get("boundedReference") == "partial"
        and context.get("testExecutionAdmission") == "exploration-only"
        and context.get("qualifiedTestGeneration") is False
        and context.get("independentFullRequiredSetProven") is False
        and context.get("externalHarnessDeliveryComplete") is None
        and not context.get("blockers")
        and context.get("observations", {}).get("fingerprintConsistency") == "consistent"
    )


def verdict_summary(v: dict) -> dict:
    return {
        "boundedReference": v["boundedReference"],
        "testExecutionAdmission": v["testExecutionAdmission"],
        "unknowns": v["unknowns"],
        "blockers": v["blockers"],
        "observedFingerprints": v["observations"]["observedFingerprints"],
        "qualifiedTestGeneration": v["qualifiedTestGeneration"],
        "importedProviderSourceOccurrence": v["sourceCurrency"]["importedProviderSourceOccurrence"],
        "importedProviderManifestIdentity": v["sourceCurrency"]["importedProviderManifestIdentity"],
        "workspaceImportDeclarations": v["sourceCurrency"]["workspaceImportDeclarations"],
        "admissionPredicate": admission_would_pass(v),
    }


def expect_partial(name: str, v: dict) -> None:
    check(v["boundedReference"] == "partial", f"{name}: expected partial, got {v['boundedReference']}")
    check(v["unknowns"] == [], f"{name}: unexpected unknowns {v['unknowns']}")
    check(v["sourceCurrency"]["importedProviderSourceOccurrence"] == "not-required-by-bounded-closure",
          f"{name}: provider occurrence not 'not-required-by-bounded-closure'")


def expect_unknown(name: str, v: dict) -> None:
    check(v["boundedReference"] == "unknown", f"{name}: expected unknown, got {v['boundedReference']}")
    check(v["blockers"] == [], f"{name}: blockers {v['blockers']}")
    check(v["testExecutionAdmission"] == "withheld-unknown", f"{name}: admission {v['testExecutionAdmission']}")
    check(v["observations"]["observedFingerprints"] == {}, f"{name}: fingerprints published")
    check(v["qualifiedTestGeneration"] is False, f"{name}: test generation qualified")
    sc = v["sourceCurrency"]
    check(sc["importedProviderSourceOccurrence"] == "unknown", f"{name}: provider occurrence not unknown")
    check(sc["importedProviderManifestIdentity"] == "unknown", f"{name}: provider manifest identity not unknown")
    check(UNKNOWN_ENTRY in v["unknowns"], f"{name}: missing unknown {UNKNOWN_ENTRY}: {v['unknowns']}")
    check(not admission_would_pass(v), f"{name}: downstream admission predicate would pass")


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--root", required=True, type=Path)
    ap.add_argument("--cli", required=True, type=Path)
    ap.add_argument("--out", required=True, type=Path)
    args = ap.parse_args()
    root, cli = args.root.resolve(), args.cli.resolve()
    check(cli.is_file(), f"CLI not found: {cli}")
    check((root / WORKSPACE).is_dir(), "reference workspace missing under --root")

    with tempfile.TemporaryDirectory(prefix="tei-provider-currency-") as directory:
        base = Path(directory)
        cases: dict[str, dict] = {}

        # P0: unmodified copy.
        p0 = base / "p0"
        shutil.copytree(root / WORKSPACE, p0 / "consumer")
        capture(cli, base, p0 / "consumer", p0 / "replies")
        v0 = judge(p0 / "replies")
        expect_partial("P0", v0)
        cases["P0"] = verdict_summary(v0)

        # P1: provider imported, but the Function closure does not reach it.
        p1 = base / "p1"
        consumer1 = build_p1(root, p1)
        capture(cli, base, consumer1, p1 / "replies")
        v1 = judge(p1 / "replies")
        expect_partial("P1", v1)
        check(v1["sourceCurrency"]["workspaceImportDeclarations"] == "not-observable-by-existing-cli",
              "P1: gate claims knowledge of workspace imports")
        catalog = json.loads((p1 / "replies/catalog.json").read_bytes())
        found = [e for e in catalog["result"]["entries"]
                 if e.get("subject") == {"kind": "semantic-type", "id": "os.process_id"}]
        check(len(found) == 1, f"P1: os.process_id catalog entries: {len(found)}")
        check(found[0].get("sourceStatus") == "not-indexed" and "prelude" not in found[0],
              f"P1: imported type not a plain not-indexed entry: {found[0]}")
        cases["P1"] = verdict_summary(v1)

        # N1: closure reaches the imported type; state A, then raw provider drift B and C.
        n1 = base / "n1"
        consumer_n = build_p1(root, n1)
        apply_n1(consumer_n)
        provider_source = n1 / "provider/semantic/os/process.tei"
        provider_manifest = n1 / "provider/tei.workspace.yaml"

        states: dict[str, dict] = {}
        for label in ("A", "B", "C"):
            if label == "B":
                append_bytes(provider_source, b"\n")
            elif label == "C":
                append_bytes(provider_manifest, b"\n")
            folder = n1 / ("replies-" + label)
            capture(cli, base, consumer_n, folder)
            v = judge(folder)
            expect_unknown("N1-" + label, v)
            states[label] = {
                "folder": folder,
                "verdict": v,
                "providerSourceSha256": digest(provider_source.read_bytes()),
                "providerManifestSha256": digest(provider_manifest.read_bytes()),
                "consumerFingerprints": catalog_fingerprints(folder),
            }
        cases["N1-A"] = verdict_summary(states["A"]["verdict"])

        def compare(left: str, right: str, field: str) -> dict:
            a, b = states[left], states[right]
            check(a[field] != b[field], f"{left}->{right}: {field} did not change")
            check(a["consumerFingerprints"] == b["consumerFingerprints"],
                  f"{left}->{right}: consumer fingerprints changed "
                  f"({a['consumerFingerprints']} vs {b['consumerFingerprints']})")
            names = {}
            for name in context_gate.COMMANDS:
                same = ((a["folder"] / (name + ".json")).read_bytes()
                        == (b["folder"] / (name + ".json")).read_bytes())
                names[name] = same
            for name in MUST_BE_IDENTICAL:
                check(names[name], f"{left}->{right}: {name}.json changed")
            return {
                field + "Before": a[field],
                field + "After": b[field],
                "worldFingerprintIdentical": True,
                "sourceIndexFingerprintIdentical": True,
                "byteIdenticalReplies": sum(names.values()),
                "totalReplies": len(names),
                "replyByteIdentical": names,
                "verdictAfter": verdict_summary(states[right]["verdict"]),
            }

        raw_drift = compare("A", "B", "providerSourceSha256")
        manifest_drift = compare("B", "C", "providerManifestSha256")
        cases["N1-B"] = verdict_summary(states["B"]["verdict"])
        cases["N1-C"] = verdict_summary(states["C"]["verdict"])

        report = {
            "schema": SCHEMA,
            "issue": ISSUE,
            "cases": cases,
            "evidence": {
                "consumerFingerprints": states["A"]["consumerFingerprints"],
                "providerRawSourceDrift": raw_drift,
                "providerManifestDrift": manifest_drift,
                "providerFileSha256": {
                    "A": {"source": states["A"]["providerSourceSha256"],
                          "manifest": states["A"]["providerManifestSha256"]},
                    "B": {"source": states["B"]["providerSourceSha256"],
                          "manifest": states["B"]["providerManifestSha256"]},
                    "C": {"source": states["C"]["providerSourceSha256"],
                          "manifest": states["C"]["providerManifestSha256"]},
                },
            },
            "limits": LIMITS,
            "binarySha256": digest(cli.read_bytes()),
        }

    args.out.parent.mkdir(parents=True, exist_ok=True)
    args.out.write_text(json.dumps(report, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    print("PASS: real CLI Provider currency pair: P0/P1 partial (closure does not reach Provider); "
          "N1 unknown with fingerprints withheld in A/B/C while Provider raw Source and Manifest bytes "
          "changed and consumer world/source-index fingerprints stayed identical; downstream admission refused")
    print("LIMIT: not a Provider occurrence check, no atomic multi-source snapshot, no Provider "
          "invocation, no disclosure/grant, no transitive closure, test generation not qualified")


if __name__ == "__main__":
    main()
