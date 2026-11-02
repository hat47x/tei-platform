"""Reproducible, narrowly scoped injection experiment on a real TEI document."""
import json
import time
from collections import Counter
from pathlib import Path

from probe import check

# Manually reviewed occurrences in docs/README.md at baseline main 497b63bf.
# This is an occurrence-level oracle, not an assertion about the entire corpus.
KNOWN_EXISTING = Counter({"tools/README.md": 1, "docs/architecture.md": 1,
                          "docs/business-system-design.md": 1, "docs/status.md": 1,
                          "docs/conformance/README.md": 1, "requirements/README.md": 2})


def main():
    root = Path(__file__).resolve().parents[2]
    document = "docs/README.md"
    text = (root / document).read_text(encoding="utf-8")
    original = "`docs/architecture.md`"
    if text.count(original) != 1 or not (root / "docs/architecture.md").is_file():
        raise RuntimeError("evaluation baseline changed; inspect it before measuring")
    if any((root / name).exists() for name in ("docs/architecture-renamed-2309.md", "docs/fabricated-2309.md")):
        raise RuntimeError("injected names now exist; select new absent names")
    # Exclude the known disputed location rather than calling it a false positive.
    baseline = check(root, document, text)
    selected = [r for r in baseline["records"] if r["name"] in KNOWN_EXISTING]
    if Counter(r["name"] for r in selected) != KNOWN_EXISTING:
        raise RuntimeError("known baseline occurrences changed; review the oracle before measuring")
    start = time.perf_counter()
    renamed = check(root, document, text.replace(original, "`ref:docs/architecture-renamed-2309.md`"))
    implicit_renamed = check(root, document, text.replace(original, "`docs/architecture-renamed-2309.md`"))
    fabricated = check(root, document, text + "\n`ref:docs/fabricated-2309.md`\n")
    general = check(root, document, text + "\n`ordinary`\n")
    planned = check(root, "plan/synthetic.md", "`planned:docs/fabricated-2309.md` `ref:docs/fabricated-2309.md`")
    elapsed = (time.perf_counter() - start) * 1000
    missing = lambda report, name: any(r["name"] == name and r["result"] == "unresolved" for r in report["records"])
    print(json.dumps({
        "document": document, "baseline_candidates": baseline["denominator"],
        "baseline_unresolved": [r["name"] for r in baseline["records"] if r["result"] == "unresolved"],
        "known_existing_occurrences": len(selected),
        "known_existing_false_positives": sum(r["result"] != "resolved" for r in selected),
        "excluded_uncertain_occurrences": sum(r["name"] == "docs/audits/" for r in baseline["records"]),
        "injected_reference_defects_detected": sum((missing(renamed, "docs/architecture-renamed-2309.md"),
                                                   missing(fabricated, "docs/fabricated-2309.md"))),
        "injected_reference_defects_total": 2,
        "implicit_rename": next((r["severity"] for r in implicit_renamed["records"]
                                 if r["name"] == "docs/architecture-renamed-2309.md"), "not-extracted"),
        "general_word_extra_candidates": general["denominator"] - baseline["denominator"],
        "planned_results": [{"result": r["result"], "severity": r["severity"]} for r in planned["records"]],
        "five_checks_elapsed_ms": round(elapsed, 2),
    }, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
