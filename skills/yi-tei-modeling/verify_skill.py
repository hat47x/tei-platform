#!/usr/bin/env python3
"""Static, dependency-free check for the draft TEI modeling Skill.

This checks repository documentation against a subset of checked-in CLI source.
It does not invoke TEI, authenticate a caller, or certify a harness/Provider.
"""

from __future__ import annotations

import argparse
from pathlib import Path
import re
import shlex
import sys

SKILL_DIR = Path("skills/yi-tei-modeling")
DOCUMENTS = (
    SKILL_DIR / "SKILL.md",
    SKILL_DIR / "references/read-only-model-investigation.md",
    Path("plan/design/TEI_PRODUCT_SKILL_HARNESS_BINDING_2026-11.md"),
)
REQUIRED_CLI = ("version", "model catalog", "inspect", "trace", "validate")
LINK = re.compile(r"\]\(([^)]+)\)")
SHELL_BLOCK = re.compile(r"(?ms)^```(?:sh|bash)\s*\n(.*?)^```\s*$")
CLI_DECLARATION = re.compile(r'\{Name:\s*"([^"]+)",\s*Effect:\s*"([^"]+)"')
# Narrow allowlist for documentation examples, not a CLI capability registry.
# The actual CLI parser and Host are authoritative; this checks examples only.
EXAMPLE_FLAGS = {
    "commands": {},
    "version": {"-json": False},
    "model catalog": {"-workspace": True, "-namespace": True},
    "inspect": {"-workspace": True, "-subject-kind": True, "-subject-id": True},
    "trace": {"-workspace": True, "-scope": True, "-subject-kind": True,
              "-subject-id": True, "-direction": True, "-mode": True,
              "-offset": True, "-limit": True, "-expected-fingerprint": True},
    "validate": {"-workspace": True, "-json": False},
}
SHELL_SIDE_EFFECT = ("&&", "||", ";", "|", ">", "<", "`", "$(")


def check_repository(root: Path) -> list[str]:
    """Return actionable errors; an empty list proves *static* consistency only."""
    root = root.resolve()
    errors: list[str] = []
    contents: dict[Path, str] = {}
    for relative in DOCUMENTS:
        path = root / relative
        if not path.is_file():
            errors.append(f"missing required document: {relative}")
            continue
        contents[relative] = path.read_text(encoding="utf-8")

    skill = contents.get(DOCUMENTS[0], "")
    frontmatter = re.match(r"\A---\n(.*?)\n---\n", skill, re.S)
    if not frontmatter:
        errors.append("SKILL.md: missing YAML frontmatter")
    else:
        metadata = frontmatter.group(1)
        if not re.search(r"(?m)^name:\s*yi-tei-modeling\s*$", metadata):
            errors.append("SKILL.md: name must match skills/yi-tei-modeling")
        if not re.search(r"(?m)^description:\s*(?:.+|>[-+]?)\s*$", metadata):
            errors.append("SKILL.md: missing description")

    for relative, markdown in contents.items():
        for found in LINK.finditer(markdown):
            target = found.group(1).split("#", 1)[0].strip()
            if not target or target.startswith(("https://", "http://", "mailto:")):
                continue
            if not (root / relative.parent / target).resolve().is_file():
                errors.append(f"{relative}: broken link {target}")

    source = root / "cmd/tei/cli_contract.go"
    main = root / "cmd/tei/main.go"
    if not source.is_file() or not main.is_file():
        errors.append("missing checked-in CLI source (cmd/tei/{cli_contract,main}.go)")
        return errors
    catalog = source.read_text(encoding="utf-8")
    main_source = main.read_text(encoding="utf-8")
    if 'SchemaVersion: "tei.cli.discovery/v0"' not in catalog:
        errors.append("CLI discovery schema version changed; review Skill")
    if 'Scope:         "selected-read-only-commands"' not in catalog and not re.search(
        r'Scope:\s*"selected-read-only-commands"', catalog
    ):
        errors.append("CLI discovery scope changed; review Skill")
    declarations = dict(CLI_DECLARATION.findall(catalog))
    for command in REQUIRED_CLI:
        if declarations.get(command) != "read-only":
            errors.append(f"CLI command missing or not read-only: {command}")
    if 'command == "commands"' not in main_source:
        errors.append("CLI commands discovery entrypoint missing from main.go")

    available = sorted((*REQUIRED_CLI, "commands"), key=len, reverse=True)
    for relative, markdown in contents.items():
        for block in SHELL_BLOCK.findall(markdown):
            for line in block.splitlines():
                line = line.strip()
                if not line or line.startswith("#"):
                    continue
                try:
                    # Split shell operators even when no whitespace surrounds
                    # them. A valid first command must not hide a chained write.
                    lex = shlex.shlex(line, posix=True, punctuation_chars=";&|<>")
                    lex.whitespace_split = True
                    lex.commenters = ""
                    args = list(lex)
                except ValueError as exc:
                    errors.append(f"{relative}: invalid shell example: {exc}")
                    continue
                go_tei = args[:3] == ["go", "run", "./cmd/tei"]
                bin_tei = bool(args) and args[0] in ("tei", "./tei")
                mentions_tei = any(
                    args[i:i + 3] == ["go", "run", "./cmd/tei"] or
                    args[i] in ("tei", "./tei", "./cmd/tei")
                    for i in range(len(args))
                )
                if mentions_tei and not (go_tei or bin_tei):
                    errors.append(f"{relative}: unverified TEI invocation context: {line}")
                    continue
                if not (go_tei or bin_tei):
                    continue
                if any(re.fullmatch(r"[;&|<>]+", token) for token in args):
                    errors.append(f"{relative}: composite TEI shell example: {line}")
                    continue
                tail = args[3:] if go_tei else args[1:]
                match = next((name for name in available if tail[: len(name.split())] == name.split()), None)
                if match is None:
                    errors.append(f"{relative}: unverified/read-write TEI example: {line}")
                    continue
                # A known command name alone does not make an entire shell
                # expression safe or guarantee that its flags exist.
                if any(op in line for op in SHELL_SIDE_EFFECT) or line.endswith("\\"):
                    errors.append(f"{relative}: shell expansion/control requires review: {line}")
                    continue
                flags = EXAMPLE_FLAGS[match]
                rest = tail[len(match.split()):]
                i = 0
                while i < len(rest):
                    name = rest[i]
                    takes_value = flags.get(name)
                    if takes_value is None:
                        errors.append(f"{relative}: unverified CLI flag/argument: {name}")
                        break
                    if takes_value:
                        if i + 1 >= len(rest) or rest[i + 1].startswith("-"):
                            errors.append(f"{relative}: missing CLI flag value: {name}")
                            break
                        i += 2
                    else:
                        i += 1

    reference = contents.get(DOCUMENTS[1], "")
    for label in (
        "result.snapshotFingerprint",
        "result.trace.snapshotFingerprint",
        "result.trace.graphFingerprint",
        "result.trace.continuation.fingerprint",
        "-expected-fingerprint",
    ):
        if label not in reference:
            errors.append(f"reference missing trace pagination binding: {label}")
    if "NOT RUN" not in skill:
        errors.append("SKILL.md must preserve explicit unexecuted status")
    return errors


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--root", type=Path, default=Path(__file__).resolve().parents[2],
        help="TEI repository root (default: derive from this script)",
    )
    root = parser.parse_args().root
    errors = check_repository(root)
    if errors:
        for item in errors:
            print(f"FAIL: {item}", file=sys.stderr)
        return 1
    print("PASS: TEI Skill static document/CLI declaration check (no CLI/Harness run)")
    return 0


if __name__ == "__main__":
    sys.exit(main())
