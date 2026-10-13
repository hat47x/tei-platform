"""実出力を既存のビルド証拠と照合する。証拠形式や実行権限を追加しない。"""
import hashlib
import json
import pathlib
import sys


def verify(directory, expected):
    allowed = {"engine", "build-evidence.json", "second-evidence.json", "second-error.txt", "response.json"}
    if {path.name for path in directory.iterdir()} != allowed:
        raise ValueError("publication temporary files or required outputs differ")
    evidence = json.loads((directory / "build-evidence.json").read_text())
    digest = "sha256:" + hashlib.sha256((directory / "engine").read_bytes()).hexdigest()
    checks = [
        evidence.get("schema") == "tei.reference.transform-engine-build-evidence/v1",
        evidence.get("repositoryRevision") == expected,
        evidence.get("reproducible") is True,
        evidence.get("artifactDigest") == digest,
        evidence.get("firstArtifactDigest") == digest,
        evidence.get("secondArtifactDigest") == digest,
    ]
    engine = evidence.get("runtimeEngine", {})
    checks += [
        engine.get("implementationDigest") == digest,
        engine.get("id") == evidence.get("engineId") and bool(evidence.get("engineId")),
        engine.get("version") == evidence.get("engineVersion") and bool(evidence.get("engineVersion")),
        engine.get("contractVersion") == evidence.get("engineContract") and bool(evidence.get("engineContract")),
        engine.get("target") == evidence.get("target") and bool(evidence.get("target")),
    ]
    response = json.loads((directory / "response.json").read_text())
    checks += [
        response.get("schema") == "tei.reference.transform-engine-response/v1",
        response.get("output", {}).get("kind") == "string",
        response.get("output", {}).get("string") == "hello from exact engine",
    ]
    if not all(checks):
        raise ValueError("exact artifact evidence or smoke response differs")
    print("artifact-digest=" + digest)
    print("artifact/evidence/response-fields=PASS")


if __name__ == "__main__":
    if len(sys.argv) != 3:
        raise SystemExit("usage: verify-export.py EXTERNAL_OUTPUT_DIRECTORY EXACT_HEAD")
    try:
        verify(pathlib.Path(sys.argv[1]), sys.argv[2])
    except (ValueError, TypeError, AttributeError, OSError) as error:
        raise SystemExit(str(error))
