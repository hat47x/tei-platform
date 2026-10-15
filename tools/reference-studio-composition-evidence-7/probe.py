#!/usr/bin/env python3
"""Black-box PoC B evidence probe; never implements a TEI resolver."""
import argparse
import hashlib
import json
import re
import subprocess
import sys
from pathlib import Path

NEGATIVES = ("unknown", "duplicate", "missing_dependency", "cycle", "incompatible_version")
HEX64 = re.compile(r"[0-9a-f]{64}\Z")


class Blocked(Exception):
    pass


class Failed(Exception):
    pass


def require(condition, why):
    if not condition:
        raise Blocked(why)


def pointer(document, path):
    require(isinstance(path, str) and path.startswith("/"), "JSON pointer must begin with /")
    part = document
    for segment in path.split("/")[1:]:
        segment = segment.replace("~1", "/").replace("~0", "~")
        try:
            if isinstance(part, dict):
                part = part[segment]
            elif isinstance(part, list) and segment.isdecimal():
                part = part[int(segment)]
            else:
                raise KeyError(segment)
        except (KeyError, IndexError) as err:
            raise Failed(f"JSON pointer {path} not found") from err
    return part


def run_case(name, case, cwd, timeout):
    require(isinstance(case, dict), f"{name}: case must be an object")
    argv = case.get("argv")
    require(isinstance(argv, list) and len(argv) > 0 and all(isinstance(v, str) and v for v in argv),
            f"{name}: argv must be a nonempty string array (never a shell command)")
    try:
        result = subprocess.run(argv, cwd=cwd, capture_output=True, timeout=timeout, check=False)
    except (FileNotFoundError, PermissionError, subprocess.TimeoutExpired) as err:
        raise Blocked(f"{name}: command cannot be completed: {type(err).__name__}") from err
    try:
        body = json.loads(result.stdout)
    except (json.JSONDecodeError, UnicodeDecodeError) as err:
        raise Failed(f"{name}: stdout is not one JSON document (exit {result.returncode})") from err
    return result.returncode, body, hashlib.sha256(result.stdout).hexdigest()


def check_success(name, case, cwd, timeout):
    code1, doc1, digest = run_case(name, case, cwd, timeout)
    if code1 != 0:
        raise Failed(f"{name}: expected exit 0, observed {code1}")
    path = case.get("resultPointer")
    value1 = pointer(doc1, path)
    code2, doc2, _ = run_case(name + " repeat", case, cwd, timeout)
    if code2 != 0 or pointer(doc2, path) != value1:
        raise Failed(f"{name}: repeated operation differs")
    return value1, doc1, digest


def file_digest(root, name):
    require(isinstance(name, str) and bool(name), "provenance path is required")
    path = (root / name).resolve()
    require(path.is_relative_to(root) and path.is_file(), f"missing or out-of-root provenance file: {name}")
    return hashlib.sha256(path.read_bytes()).hexdigest()


def check_spec(spec, root):
    require(isinstance(spec, dict), "probe spec must be an object")
    comparison = spec.get("comparison")
    require(isinstance(comparison, dict), "comparison is required")
    timeout = spec.get("timeoutSeconds", 30)
    require(type(timeout) in (int, float) and 0 < timeout <= 120, "timeoutSeconds must be 1..120")
    baseline = comparison.get("regular")
    composed = comparison.get("composed")
    require(baseline is not None and composed is not None, "regular and composed cases are required")
    normal_result, _, normal_digest = check_success("regular", baseline, root, timeout)
    composed_result, composed_doc, composed_digest = check_success("composed", composed, root, timeout)
    if normal_result != composed_result:
        raise Failed("regular vs composed: operation result is not semantically equal")
    observed = {"regularSha256": normal_digest, "composedSha256": composed_digest, "negative": {}, "provenance": {}}

    negative = spec.get("negative")
    require(isinstance(negative, dict) and set(NEGATIVES) == set(negative),
            "all five named negative cases are required")
    for name in NEGATIVES:
        case = negative[name]
        require(isinstance(case, dict), f"negative {name} must be an object")
        expected = case.get("expectedCode")
        require(isinstance(expected, str) and expected, f"negative {name}: expectedCode required")
        code, body, _ = run_case(name, case, root, timeout)
        if code == 0:
            raise Failed(f"negative {name}: unexpectedly accepted")
        actual = pointer(body, case.get("errorPointer"))
        if actual != expected:
            raise Failed(f"negative {name}: wrong error classification")
        observed["negative"][name] = {"status": "EXPECTED_DENY", "code": actual}

    provenance = spec.get("provenance")
    require(isinstance(provenance, dict) and set(provenance) == {"lock", "asset", "build"},
            "lock, asset, build provenance are all required")
    for kind in ("lock", "asset", "build"):
        entry = provenance[kind]
        require(isinstance(entry, dict), f"{kind}: provenance entry must be object")
        expected = entry.get("sha256")
        require(isinstance(expected, str) and HEX64.fullmatch(expected) is not None,
                f"{kind}: fixed 64-character lowercase SHA-256 required")
        actual = file_digest(root, entry.get("path"))
        if actual != expected:
            raise Failed(f"{kind}: file digest differs from independent pinned digest")
        bound = entry.get("composedDigestPointer")
        require(isinstance(bound, str) and bound.startswith("/"),
                f"{kind}: composedDigestPointer required to bind actual output")
        claimed = pointer(composed_doc, bound)
        if claimed not in (actual, "sha256:" + actual):
            raise Failed(f"{kind}: composed output not bound to file digest")
        observed["provenance"][kind] = actual
    return observed


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--spec", type=Path, required=True, help="trusted local probe input, not a TEI contract")
    args = parser.parse_args(argv)
    specpath = args.spec.resolve()
    try:
        spec = json.loads(specpath.read_text(encoding="utf-8"))
        observed = check_spec(spec, specpath.parent)
    except (FileNotFoundError, json.JSONDecodeError, UnicodeDecodeError, Blocked) as err:
        print(json.dumps({"status": "BLOCKED", "reason": str(err)}, ensure_ascii=False))
        return 2
    except (Failed, OSError) as err:
        print(json.dumps({"status": "FAIL", "reason": str(err)}, ensure_ascii=False))
        return 1
    print(json.dumps({"status": "PASS", "evidence": observed}, sort_keys=True))
    return 0


if __name__ == "__main__":
    sys.exit(main())
