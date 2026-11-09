#!/usr/bin/env python3
"""Static exclusion/leak inspection of the two REAL E1 artifacts (Issue #1182).

  python -I inspect_artifact_exclusion.py --api <linux api binary> \
      --browser <browser asset directory> [--profile json-api|frozen-world] \
      [--go go] [--out result.json]

(i)  API binary: every symbol reported by `go tool nm` is checked against a
     FORBIDDEN list (presentation, Studio/Desktop host, authoring/compiler front
     ends, test fixtures). Known, documented intrusion (the design-time
     Workspace loader that README.md states the API executable still contains)
     is kept in a separate ALLOWLIST: it is counted and reported, never silently
     accepted, and a `tei/...` package that is in neither list is a NEW
     deviation and fails. The `frozen-world` profile has an empty allowlist.
(ii) Browser directory: all bytes of every file are scanned for synthetic
     credentials, server-only import names, DSN strings and source map
     references.
(iii) A JSON receipt reports inspected-symbol count, forbidden hits and the
     allowlist count. Exit status is 0 only when everything passes.

This is a diagnostic, byte/symbol level observation. It does NOT prove absence
of arbitrary secrets, absence of reachable code (nm lists linked symbols, not
what is executed), production readiness or Builder/Lock authority.
"""
import argparse
import hashlib
import json
import os
import re
import subprocess
import sys

sys.dont_write_bytecode = True

# A package path token must not be preceded by another path character, so that
# `html/template` does not match inside `example.com/html/template`. Go's
# `type:.eq.html/template.state` style prefixes (`.eq.`, `.hash.`) are allowed.
_LB = r"(?:(?<![\w.~/-])|(?<=\.eq\.)|(?<=\.hash\.))"
_END = r"(?=[./)\]*,\s]|$)"


def _pkg(path):
    return re.compile(_LB + re.escape(path) + _END)


# --------------------------------------------------------------------------
# FORBIDDEN: any hit fails, in every profile. (id, matcher, rationale)
# --------------------------------------------------------------------------
FORBIDDEN = (
    ("html/template", _pkg("html/template"),
     "Server-side HTML presentation. tei-api-only-isolation-ci.yml excludes "
     "custom_ui.go/server.go via tei_api_only and checks the 'Custom Submit UI' "
     "marker; #1182 keeps presentation in the Browser unit only."),
    ("text/template", _pkg("text/template"),
     "Template engine behind the presentation path (html/template wraps it); "
     "same rationale as html/template, also covers text/template/parse."),
    ("tei/internal/studiohost", _pkg("tei/internal/studiohost"),
     "Studio host: build/authoring tool, not a runtime unit (#1182 issue text "
     "excludes build-time Editor/Compiler and the Studio Wails Host)."),
    ("tei/internal/desktophost", _pkg("tei/internal/desktophost"),
     "Desktop/Studio host (#1182 non-goal: Desktop implementation)."),
    ("tei/internal/workbench", _pkg("tei/internal/workbench"),
     "Authoring workbench; build-time Editor side (#1182 item 2)."),
    ("tei/internal/xeadsource", _pkg("tei/internal/xeadsource"),
     "Legacy source importer/classifier: build-time Compiler input side."),
    ("tei/internal/scenariotest", _pkg("tei/internal/scenariotest"),
     "Scenario test harness / fixtures; must not ship in a runtime unit."),
    ("tei/internal/sourceacceptance", _pkg("tei/internal/sourceacceptance"),
     "Source Acceptance verification (authority side); not a runtime unit."),
    ("tei/cmd", _pkg("tei/cmd"),
     "TEI CLI / Studio command packages (tei, tei-studio) are tools, not API."),
    ("testing", _pkg("testing"),
     "Go test framework / httptest-style fixtures must not link into a "
     "production unit."),
    ("net/http/httptest", _pkg("net/http/httptest"),
     "Test server fixtures; synthetic-only, never in a runtime unit."),
    ("go/parser", _pkg("go/parser"),
     "Go compiler front end (Compiler is build-time only, #1182 item 2)."),
    ("go/types", _pkg("go/types"),
     "Go compiler type checker (build-time Compiler, #1182 item 2)."),
    ("go/build", _pkg("go/build"),
     "Go build tool library (build-time only)."),
    ("main.presentation-symbols",
     re.compile(_LB + r"main\.(?:\(\*\w+\)\.)?"
                r"(?:handleCustomSubmit|handleIndex|renderCustomSubmit|"
                r"referenceScenarioTemplates)\b"),
     "Presentation entry points present in the tei_order_only (HTML) build and "
     "absent from tei_api_only; observed by diffing the two real binaries and "
     "mirrored by the forbidden-source list in tei-api-only-isolation-ci.yml."),
)

# --------------------------------------------------------------------------
# KNOWN INTRUSION (allowlist, json-api profile only). README.md: "APIバイナリは
# 現段階では設計時Workspace/Driver機構を含みます". These are exactly the tei
# packages in the `go list -deps` closure of the tei_order_only,tei_api_only
# build that are absent from the closure of the tei_frozen_world build (a
# symbol diff of one Go version is not reliable: whether a data-only package
# shows up in nm depends on the Go version, e.g. only as `type:.eq.` symbols).
# --------------------------------------------------------------------------
KNOWN_INTRUSION = (
    "tei/internal/workspace",
    "tei/internal/syntax",
    "tei/internal/sourceindex",
    "tei/internal/plugin/implementation",
    "tei/core/subject",
    "tei/internal/subject",
    "tei/internal/deployment/executor",
    "tei/internal/stdlib",
    "tei/internal/validation",
)

# Runtime mechanism that #1182 item 3 requires the API to keep (Driver,
# Realization, Authorization, Validation, Legacy Authentication). Anything under
# tei/ that is in neither this set nor KNOWN_INTRUSION is a new deviation.
RUNTIME_EXPECTED = (
    "tei/core/effect",
    "tei/core/generation",
    "tei/core/identity",
    "tei/internal/composition",
    # Data-model packages in BOTH the tei_api_only and tei_frozen_world import
    # closures (`go list -deps`). They can show up only as `type:.eq.<pkg>.T`
    # symbols, so whether they are linked depends on the Go version (seen in CI
    # on Go 1.26, absent in a local Go 1.27 build).
    "tei/internal/application",
    "tei/internal/application/release",
    "tei/internal/releaseplan",
    "tei/internal/database",
    "tei/internal/database/migration",
    "tei/internal/database/persistence",
    "tei/internal/database/physical",
    "tei/internal/database/release",
    "tei/internal/environment",
    "tei/internal/query",
    "tei/internal/resource",
    "tei/internal/semantic",
    "tei/internal/driver",
    "tei/internal/driver/queryresult",
    "tei/internal/permission",
    "tei/internal/plugin",
    "tei/internal/realization",
    "tei/internal/security/effectcommit/postgres",
    "tei/internal/security/legacyauth",
    "tei/internal/security/legacyauthorization",
    "tei/internal/semantic/capabilityinvoke",
    "tei/internal/semantic/derivationref",
    "tei/internal/semantic/validationeval",
    "tei/reference/web-composition/go-host",
    "tei/tools/reference-first-web-target/sharedprojection",
)
# Sanity: the binary must really be a linked Go program with the runtime core.
REQUIRED_PRESENT = ("tei/internal/driver",)

PROFILES = ("json-api", "frozen-world")
_TEI_PKG = re.compile(_LB + r"(tei/(?:[\w~-]+/)*[\w~-]+)(?=[.\s/)\]*,]|$)")
_NM_LINE = re.compile(r"^\s*(?:[0-9a-fA-F]+\s+)?[A-Za-z?]\s+(.+?)\s*$")


def parse_nm(text):
    names = []
    for line in text.splitlines():
        match = _NM_LINE.match(line)
        if match:
            names.append(match.group(1))
    return names


def analyze_symbols(names, profile="json-api"):
    if profile not in PROFILES:
        raise ValueError("unknown profile: " + profile)
    forbidden = {}
    intrusion = {}
    runtime = {}
    unexpected = {}
    for name in names:
        for ident, pattern, _why in FORBIDDEN:
            if pattern.search(name):
                hit = forbidden.setdefault(ident, {"count": 0, "examples": []})
                hit["count"] += 1
                if len(hit["examples"]) < 3:
                    hit["examples"].append(name[:200])
        for pkg in set(_TEI_PKG.findall(name)):
            if pkg in KNOWN_INTRUSION:
                intrusion[pkg] = intrusion.get(pkg, 0) + 1
            elif pkg in RUNTIME_EXPECTED:
                runtime[pkg] = runtime.get(pkg, 0) + 1
            elif not any(p.search(pkg) for _i, p, _w in FORBIDDEN):
                unexpected[pkg] = unexpected.get(pkg, 0) + 1
    if profile == "frozen-world":
        # Build-time loader packages are forbidden outright in this profile.
        for pkg, count in intrusion.items():
            forbidden["frozen-world:" + pkg] = {"count": count, "examples": [pkg]}
        intrusion = {}
    missing = [p for p in REQUIRED_PRESENT if p not in runtime]
    if not any(n.startswith("main.main") for n in names):
        missing.append("main.main")
    failures = []
    if not names:
        failures.append("no symbols: stripped or non-Go binary cannot be inspected")
    failures += ["forbidden symbol/package: " + k for k in sorted(forbidden)]
    failures += ["unexpected tei package (not in runtime or allowlist): " + k
                 for k in sorted(unexpected)]
    failures += ["required runtime symbol missing: " + m for m in missing]
    return {
        "profile": profile,
        "symbolsInspected": len(names),
        "forbiddenHitCount": sum(v["count"] for v in forbidden.values()),
        "forbiddenHits": forbidden,
        "knownIntrusionPackageCount": len(intrusion),
        "knownIntrusionSymbolCount": sum(intrusion.values()),
        "knownIntrusionAllowlisted": intrusion,
        "runtimePackagesObserved": sorted(runtime),
        "unexpectedTeiPackages": unexpected,
        "failures": failures,
        "passed": not failures,
    }


def sha256_file(path):
    digest = hashlib.sha256()
    size = 0
    with open(path, "rb") as handle:
        for block in iter(lambda: handle.read(1 << 20), b""):
            size += len(block)
            digest.update(block)
    return digest.hexdigest(), size


def run_nm(binary, go="go"):
    try:
        done = subprocess.run([go, "tool", "nm", binary], capture_output=True,
                              timeout=300, check=False)
    except (OSError, subprocess.SubprocessError) as error:
        raise SystemExit("go tool nm could not run: %s" % error)
    if done.returncode != 0:
        raise SystemExit("go tool nm failed: " + done.stderr.decode("utf-8", "replace")[:500])
    return done.stdout.decode("utf-8", "replace")


def inspect_api(binary, profile, go="go"):
    result = analyze_symbols(parse_nm(run_nm(binary, go)), profile)
    sha, size = sha256_file(binary)
    result.update({"path": os.path.basename(binary), "sha256": sha, "bytes": size})
    return result


# --------------------------------------------------------------------------
# Browser asset scan: every byte of every file. (id, bytes regex, rationale)
# --------------------------------------------------------------------------
BROWSER_PATTERNS = (
    ("synthetic-credential",
     re.compile(rb"(?:editor|viewer|wrong)-secret|reference/legacy-auth/"),
     "Synthetic Reference credentials used by smoke_json_api.py and "
     "real_browser_replay.mjs, and their fixture path; build_json_api.sh checks "
     "the same strings in index.html only."),
    ("private-key-block",
     re.compile(rb"-----BEGIN [A-Z ]*PRIVATE KEY-----"),
     "Private key material; Server/Resource secret must never be bundled."),
    ("cloud-access-key",
     re.compile(rb"\bAKIA[0-9A-Z]{16}\b"),
     "Well-known cloud access key id shape."),
    ("dsn",
     re.compile(rb"(?i)\b(?:postgres(?:ql)?|mysql|mariadb|mongodb(?:\+srv)?|redis|"
                rb"oracle|sqlserver)://"),
     "Database/connection DSN: the DB Provider is Server-only (#1182 item 2)."),
    ("dsn-keyword",
     re.compile(rb"(?i)\bsslmode=|\bpassword=[^\s&\"'<>]"),
     "libpq/keyword-form DSN or password parameter."),
    ("server-only-import",
     re.compile(rb"tei/internal/|github\.com/jackc/pgx|modernc\.org/sqlite|"
                rb"go-ora|database/sql|workspace\.Load|driver\.NewResolver|"
                rb"html/template"),
     "Server-only Provider / Workspace loader / Driver names; build_json_api.sh "
     "and migrate_browser.py check a subset of these in index.html only."),
    ("server-only-js-module",
     re.compile(rb"(?:require\(\s*|from\s+|import\s*\(\s*|import\s+)[\"'](?:pg|mysql2?|"
                rb"sqlite3|better-sqlite3|oracledb|(?:node:)?(?:fs|child_process))[\"']"),
     "Server-side Node modules imported by Browser JS."),
    ("source-map-reference",
     re.compile(rb"(?m)sourceMappingURL|^//[#@]\s*sourceURL="),
     "Source map reference exposes build inputs (#1182 item 2 / README forbids)."),
    ("native-executable",
     re.compile(rb"\A(?:\x7fELF|MZ\x90\x00|\x00asm|\xcf\xfa\xed\xfe)", re.S),
     "A compiled executable/WebAssembly module inside the Browser unit."),
)


def scan_browser(root):
    if not os.path.isdir(root):
        raise SystemExit("browser path is not a directory: " + root)
    files = []
    hits = {}
    for base, dirs, names in os.walk(root):
        dirs.sort()
        # os.walk lists a directory symlink in `dirs` but never descends into
        # it, so its target would be neither scanned nor reported.
        for name in dirs:
            if os.path.islink(os.path.join(base, name)):
                link = os.path.relpath(os.path.join(base, name), root).replace(os.sep, "/")
                hits.setdefault("symlink", []).append(link + "/")
        for name in sorted(names):
            files.append(os.path.join(base, name))
    scanned_bytes = 0
    for path in files:
        rel = os.path.relpath(path, root).replace(os.sep, "/")
        if os.path.islink(path):
            hits.setdefault("symlink", []).append(rel)
            continue
        if rel.lower().endswith(".map"):
            hits.setdefault("source-map-file", []).append(rel)
        with open(path, "rb") as handle:
            data = handle.read()
        scanned_bytes += len(data)
        for ident, pattern, _why in BROWSER_PATTERNS:
            if pattern.search(data):
                hits.setdefault(ident, []).append(rel)
    failures = ["browser leak/inclusion: %s in %s" % (k, ", ".join(v))
                for k, v in sorted(hits.items())]
    if "index.html" not in {os.path.relpath(p, root).replace(os.sep, "/") for p in files}:
        failures.append("browser/index.html missing")
    return {
        "filesScanned": len(files),
        "bytesScanned": scanned_bytes,
        "patternsApplied": len(BROWSER_PATTERNS) + 2,
        "hits": hits,
        "hitCount": sum(len(v) for v in hits.values()),
        "failures": failures,
        "passed": not failures,
    }


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__.split("\n")[0])
    parser.add_argument("--api", required=True, help="Linux API executable")
    parser.add_argument("--browser", required=True, help="Browser asset directory")
    parser.add_argument("--profile", choices=PROFILES, default="json-api")
    parser.add_argument("--go", default=os.environ.get("TEI_GO", "go"))
    parser.add_argument("--out", help="write the JSON receipt here too")
    args = parser.parse_args(argv)
    api = inspect_api(args.api, args.profile, args.go)
    browser = scan_browser(args.browser)
    result = {
        "kind": "diagnostic-artifact-exclusion-static-inspection",
        "api": api,
        "browser": browser,
        "passed": api["passed"] and browser["passed"],
        "notProven": [
            "nm lists linked symbols, not reachable or executed code",
            "browser scan is pattern based; it cannot prove absence of arbitrary secrets",
            "known Workspace/Driver intrusion in the json-api profile is allowlisted, not removed",
            "no Builder/Lock/Acceptance authority and no production readiness",
        ],
    }
    text = json.dumps(result, indent=2, ensure_ascii=False, sort_keys=True) + "\n"
    sys.stdout.write(text)
    if args.out:
        with open(args.out, "w", encoding="utf-8", newline="\n") as handle:
            handle.write(text)
    return 0 if result["passed"] else 1


if __name__ == "__main__":
    sys.exit(main())
