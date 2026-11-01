"""Reproducible, narrowly scoped injection experiment on a real TEI document."""
import json
import time
from pathlib import Path

from probe import check


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
    start = time.perf_counter()
    renamed = check(root, document, text.replace(original, "`ref:docs/architecture-renamed-2309.md`"))
    fabricated = check(root, document, text + "\n`ref:docs/fabricated-2309.md`\n")
    general = check(root, document, text + "\n`ordinary`\n")
    planned = check(root, "plan/synthetic.md", "`planned:docs/fabricated-2309.md` `ref:docs/fabricated-2309.md`")
    elapsed = (time.perf_counter() - start) * 1000
    missing = lambda report, name: any(r["name"] == name and r["result"] == "unresolved" for r in report["records"])
    print(json.dumps({
        "document": document, "baseline_candidates": baseline["denominator"],
        "baseline_unresolved": [r["name"] for r in baseline["records"] if r["result"] == "unresolved"],
        "injected_reference_defects_detected": sum((missing(renamed, "docs/architecture-renamed-2309.md"),
                                                   missing(fabricated, "docs/fabricated-2309.md"))),
        "injected_reference_defects_total": 2,
        "general_word_extra_candidates": general["denominator"] - baseline["denominator"],
        "planned_results": [r["result"] for r in planned["records"]],
        "four_checks_elapsed_ms": round(elapsed, 2),
    }, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
