"""Report a small fixed set of real documents without changing them."""
import json
import time
from pathlib import Path

from probe import check


def main():
    root = Path(__file__).resolve().parents[2]
    documents = ("docs/README.md", "docs/architecture.md", "docs/status.md", "docs/getting-started.md")
    output = []
    for document in documents:
        started = time.perf_counter()
        report = check(root, document, (root / document).read_text(encoding="utf-8"))
        output.append({"document": document, "rule": report["rule"], "matching": report["matching"],
                       "denominator": report["denominator"],
                       "findings": [r for r in report["records"] if r["severity"] != "info"],
                       "elapsed_ms": round((time.perf_counter() - started) * 1000, 2)})
    print(json.dumps(output, ensure_ascii=False, indent=2))


if __name__ == "__main__":
    main()
