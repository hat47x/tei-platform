#!/usr/bin/env python3
"""Count Function types and script UI call candidates without emitting model text."""
from __future__ import annotations

import argparse
from collections import Counter
import hashlib
import json
from pathlib import Path
import re
import xml.etree.ElementTree as ET
import zipfile

FUNCTION_TYPES = tuple("XF" + n for n in ("000", "100", "110", "200", "290", "300", "310", "390"))
UI_CALLS = ("getInputDialog", "getOptionDialog", "getCheckListDialog", "getLongTextEditor",
            "showDialogToChooseFile", "showMessageDialog", "showConfirmDialog", "showInputDialog",
            "showOptionDialog")
MAX_XML_BYTES = 32 * 1024 * 1024


def inspect_model(data: bytes) -> dict:
    if len(data) > MAX_XML_BYTES:
        raise ValueError("XEAF exceeds 32 MiB")
    if re.search(br"<!\s*(DOCTYPE|ENTITY)\b", data, re.IGNORECASE):
        raise ValueError("DTD/entity declarations are not accepted")
    root = ET.fromstring(data)
    counts = Counter(f.get("Type", "") for f in root.findall(".//Function"))
    ui = Counter()
    for element in root.iter():
        # Attribute names are from Driver's Function/Table/System script inputs.
        for key in ("Script", "Text", "LoginScript", "ScriptFunctions"):
            if key == "Text" and element.tag != "Script":
                continue
            text = element.get(key, "")
            for call in UI_CALLS:
                ui[call] += len(re.findall(r"\b" + call + r"\s*\(", text))
    return {"sha256": hashlib.sha256(data).hexdigest(),
            "functionTypes": dict(sorted(counts.items())),
            "scriptUiCallCandidates": {key: value for key, value in sorted(ui.items()) if value}}


def inventory(paths: list[Path]) -> dict:
    models = []
    for path in paths:
        if path.is_dir():
            models.extend(inventory(sorted(p for p in path.rglob("*")
                                           if p.suffix.lower() in (".xeaf", ".zip")))["models"])
        elif path.suffix.lower() == ".zip":
            with zipfile.ZipFile(path) as archive:
                for entry in archive.infolist():
                    if entry.filename.lower().endswith(".xeaf"):
                        if entry.file_size > MAX_XML_BYTES:
                            raise ValueError("archive XEAF exceeds 32 MiB")
                        # Enumerate every member, including backups. Never choose a version implicitly.
                        models.append({"archive": path.name, "member": entry.filename,
                                       **inspect_model(archive.read(entry))})
        elif path.suffix.lower() == ".xeaf":
            if path.stat().st_size > MAX_XML_BYTES:
                raise ValueError("XEAF exceeds 32 MiB")
            models.append({"file": path.name, **inspect_model(path.read_bytes())})
        else:
            raise ValueError("input must be a directory, .zip or .xeaf")
    used = sorted({typ for model in models for typ, count in model["functionTypes"].items() if count})
    return {"models": models, "usedFunctionTypes": used,
            "absentFunctionTypes": sorted(set(FUNCTION_TYPES) - set(used)),
            "note": "Static counts only. UI calls are lexical candidates, including comments/strings; no execution or coverage claim."}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("inputs", nargs="+", type=Path)
    parser.add_argument("--out", type=Path, required=True,
                        help="local output; filenames may be private, do not publish without review")
    args = parser.parse_args()
    try:
        result = inventory(args.inputs)
        if not result["models"]:
            raise ValueError("no XEAF models found; scope is unknown")
        args.out.parent.mkdir(parents=True, exist_ok=True)
        args.out.write_text(json.dumps(result, ensure_ascii=False, indent=2), encoding="utf-8")
    except (OSError, ValueError, ET.ParseError, zipfile.BadZipFile) as error:
        parser.exit(1, f"inventory failed: {type(error).__name__}\n")
    print(f"models={len(result['models'])}, types={','.join(result['usedFunctionTypes'])}")


if __name__ == "__main__":
    main()
