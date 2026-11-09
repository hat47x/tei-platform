"""Fixture-only W0b: two views of a single observed, read-only TEI CLI witness.

The stable key order is a test convenience, not a published TEI Context ABI.
Do not pass private workspaces or unreviewed paths to this fixture exporter.
"""
import json
import re
from pathlib import Path

LABELS = (
    ("意味対象の種類", "subjectKind"),
    ("意味対象ID", "subjectId"),
    ("実行ファイルのGit版", "binaryRevision"),
    ("意味モデルの照合値", "worldFingerprint"),
    ("入力元索引の照合値", "sourceIndexFingerprint"),
    ("入力元モデルの版", "modelRevision"),
    ("参照Sourceの相対パス", "sourceArtifacts"),
    ("直接参照の観測数", "traceRows"),
    ("未解決の参照数", "traceUnresolved"),
    ("検証診断数", "validationDiagnostics"),
    ("読取りの資格", "evidenceKind"),
    ("開示判断の資格", "disclosureStatus"),
    ("未成立の保証", "notEstablished"),
)
KEYS = tuple(key for _, key in LABELS)
SOURCE_PATH = re.compile(r"^[A-Za-z0-9_./-]+$")
HASH_OR_VERSION = re.compile(r"^[a-zA-Z0-9_:.=-]+$")
GIT_SHA = re.compile(r"^[0-9a-f]{40}$")
UNKNOWN = (
    "accepted-business-meaning",
    "physical-provider-snapshot",
    "physical-read-minimality",
    "external-consumer-receipt",
    "external-disclosure-authorization",
    "SHO-save-or-reader-acceptance",
    "host-action-commit-or-effect",
)


def _require(test, message):
    if not test:
        raise ValueError(message)


def validate_witness(witness):
    _require(isinstance(witness, dict) and set(witness) == set(KEYS),
             "witness must contain exactly the fixture fields")
    _require(witness["subjectKind"] == "semantic-relation" and
             witness["subjectId"] == "sales.order.customer",
             "this fixture permits only the public reference subject")
    _require(GIT_SHA.fullmatch(witness["binaryRevision"]) is not None,
             "unverifiable executable revision")
    for key in ("worldFingerprint", "sourceIndexFingerprint", "modelRevision"):
        _require(isinstance(witness[key], str) and
                 HASH_OR_VERSION.fullmatch(witness[key]) is not None,
                 "invalid observed version: " + key)
    paths = witness["sourceArtifacts"]
    _require(isinstance(paths, list) and bool(paths) and
             paths == sorted(set(paths)), "source paths must be unique and ordered")
    for path in paths:
        _require(isinstance(path, str) and SOURCE_PATH.fullmatch(path) is not None
                 and not path.startswith("/") and
                 all(part not in ("", ".", "..") for part in path.split("/")),
                 "non-relative or unsafe source path")
    for key in ("traceRows", "traceUnresolved", "validationDiagnostics"):
        _require(type(witness[key]) is int and witness[key] >= 0,
                 "invalid observed count: " + key)
    _require(witness["traceUnresolved"] <= witness["traceRows"],
             "unresolved count exceeds all direct references")
    _require(witness["evidenceKind"] == "read-only-reference-cli-observation",
             "unexpected evidence qualification")
    _require(witness["disclosureStatus"] == "reference-fixture-only-no-policy-grant",
             "fixture cannot assert disclosure permission")
    _require(witness["notEstablished"] == list(UNKNOWN),
             "required unestablished qualifications were changed")
    return witness


def make_witness(*, subject, binary_revision, catalog, inspect, trace, rows, diagnostics):
    """Material from the *same* validated invocation series in verify_real_cli."""
    refs = inspect["sources"]["references"]
    paths = sorted({ref["artifactPath"] for ref in refs})
    record = {
        "subjectKind": subject["kind"],
        "subjectId": subject["id"],
        "binaryRevision": binary_revision,
        "worldFingerprint": catalog["worldFingerprint"],
        "sourceIndexFingerprint": catalog["sourceIndexFingerprint"],
        "modelRevision": catalog["modelRevision"],
        "sourceArtifacts": paths,
        "traceRows": rows,
        "traceUnresolved": trace["unresolved"],
        "validationDiagnostics": len(diagnostics),
        "evidenceKind": "read-only-reference-cli-observation",
        "disclosureStatus": "reference-fixture-only-no-policy-grant",
        "notEstablished": list(UNKNOWN),
    }
    return validate_witness(record)


def render_markdown(witness):
    validate_witness(witness)
    lines = [
        "# TEI Construction Context：参照用の読取り観測票",
        "",
        "同じ実CLIの読取り結果から生成した人間向け投影です。"
        "意味受理、実DBのSnapshot、外部開示の許可、SHO保存を保証しません。",
        "",
        "| 項目 | 観測値（JSON表記） |",
        "| --- | --- |",
    ]
    for label, key in LABELS:
        # Use JSON for each cell to preserve numbers vs strings/arrays.
        value = json.dumps(witness[key], ensure_ascii=True, separators=(",", ":"))
        _require("|" not in value and "\n" not in value, "unsafe markdown table cell")
        lines.append("| " + label + " | " + value + " |")
    lines.append("")
    return "\n".join(lines)


def parse_markdown(markdown):
    _require(isinstance(markdown, str), "markdown must be a string")
    lines = markdown.splitlines()
    _require(len(lines) == 6 + len(LABELS), "wrong number of markdown lines")
    _require(lines[0].startswith("# TEI Construction Context") and
             lines[4] == "| 項目 | 観測値（JSON表記） |" and
             lines[5] == "| --- | --- |",
             "unexpected markdown projection headers")
    keys = {label: key for label, key in LABELS}
    result = {}
    for row in lines[6:]:
        _require(row.startswith("| ") and row.endswith(" |"), "invalid markdown row")
        columns = row[2:-2].split(" | ")
        _require(len(columns) == 2, "incorrect number of markdown columns")
        label, value = columns
        _require(label in keys and keys[label] not in result,
                 "unrecognized or duplicated markdown row")
        try:
            result[keys[label]] = json.loads(value)
        except ValueError as exc:
            raise ValueError("invalid JSON in markdown cell") from exc
    return validate_witness(result)


def export_views(witness, destination):
    """Create temporary test materials, never deploy them as accepted documents."""
    validate_witness(witness)
    target = Path(destination)
    _require(target.is_dir(), "consumer fixture destination must already exist")
    json_path = target / "construction-context-ai-reference.json"
    md_path = target / "construction-context-human-reference.md"
    json_path.write_text(
        json.dumps(witness, ensure_ascii=False, sort_keys=True, indent=2) + "\n",
        encoding="utf-8",
    )
    md_path.write_text(render_markdown(witness), encoding="utf-8")
    # Read files back as separate consumers would; no HTML render/SHO read here.
    actual_json = json.loads(json_path.read_text(encoding="utf-8"))
    actual_md = parse_markdown(md_path.read_text(encoding="utf-8"))
    _require(validate_witness(actual_json) == actual_md == witness,
             "human/AI views of the same CLI observation disagree")
    return (json_path, md_path)
