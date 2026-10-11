#!/usr/bin/env python3
"""Select public-document paths without legacy redirect-only Markdown files.

This command filters an already reviewed manifest; it does not certify content
for publication, inspect private GitHub issues, or create a public repository.
"""
import os
import pathlib
import posixpath
import re
import subprocess
import sys
from urllib.parse import unquote, urlsplit

PATH = re.compile(r"^[A-Za-z0-9_./-]+$")
SHA = re.compile(r"^[0-9a-f]{40}$")


def stop(message: str) -> None:
    raise SystemExit(message)


def git(repo: pathlib.Path, *args: str) -> bytes:
    return subprocess.check_output(
        ["git", "-C", str(repo), *args], stderr=subprocess.DEVNULL
    )


def read_paths(file: pathlib.Path) -> list[str]:
    items: list[str] = []
    seen: set[str] = set()
    for line in file.read_text(encoding="utf-8").splitlines():
        path = line.strip()
        if not path or path.startswith("#"):
            continue
        if (
            not PATH.fullmatch(path)
            or path.startswith("/")
            or "//" in path
            or any(part in ("", ".", "..", ".git") for part in path.split("/"))
        ):
            stop("Invalid file path in review manifest")
        if path in seen:
            stop("Duplicate file path in review manifest")
        seen.add(path)
        items.append(path)
    return items


def read_map(file: pathlib.Path) -> dict[str, str]:
    result: dict[str, str] = {}
    for line in file.read_text(encoding="utf-8").splitlines():
        if not line or line.startswith("#"):
            continue
        parts = line.split("\t")
        if len(parts) != 2:
            stop("Invalid compatibility document mapping")
        source, destination = parts
        if source == destination or source in result:
            stop("Ambiguous compatibility document mapping")
        for p in (source, destination):
            if (
                not PATH.fullmatch(p)
                or p.startswith("/")
                or any(v in (".", "..", ".git") for v in p.split("/"))
            ):
                stop("Invalid compatibility document path")
        result[source] = destination
    return result


def links(markdown: str) -> list[str]:
    # Inline Markdown links and image links, plus Markdown reference definitions.
    found = re.findall(r"!?\[[^\]\n]*\]\(\s*<?([^)> \t\n]+)", markdown)
    found += re.findall(r"(?m)^\[[^\]\n]+\]:\s*<?([^\s>]+)", markdown)
    found += re.findall(r"(?i)\b(?:href|src)=[\"']([^\"']+)", markdown)
    return found


def main() -> None:
    if len(sys.argv) != 6:
        stop("Usage: select-public-docs.py REPO SHA REVIEWED_LIST MAP_TSV OUTPUT_LIST")
    repo = pathlib.Path(sys.argv[1]).resolve(strict=True)
    sha = sys.argv[2]
    input_file = pathlib.Path(sys.argv[3]).resolve(strict=True)
    mapping_file = pathlib.Path(sys.argv[4]).resolve(strict=True)
    output = pathlib.Path(sys.argv[5]).resolve()
    if not SHA.fullmatch(sha):
        stop("Expected exact 40-character Git commit SHA")
    if output.exists() or output.is_symlink():
        stop("Output already exists")
    if output == input_file or output == mapping_file or output.is_relative_to(repo):
        stop("Output must be a new file outside the source checkout")
    if not output.parent.is_dir():
        stop("Output parent directory does not exist")
    if git(repo, "rev-parse", "--verify", sha + "^{commit}").decode().strip() != sha:
        stop("Commit SHA not found")

    blobs: set[str] = set()
    raw = git(repo, "ls-tree", "-r", "-z", "--full-tree", sha)
    for entry in raw.split(b"\0"):
        if not entry:
            continue
        meta, name = entry.split(b"\t", 1)
        mode, kind, _ = meta.decode("ascii").split()
        if kind == "blob" and mode in ("100644", "100755"):
            blobs.add(name.decode("utf-8"))
    reviewed = read_paths(input_file)
    if not reviewed or not set(reviewed) <= blobs:
        stop("Reviewed manifest contains a missing/non-regular source path")
    mapping = read_map(mapping_file)
    if not set(mapping) <= blobs or not set(mapping.values()) <= blobs:
        stop("Compatibility mapping contains a missing source/destination")
    selected = set(reviewed)
    excluded = selected & set(mapping)
    for path in sorted(excluded):
        if mapping[path] not in selected:
            stop("Cannot omit a redirect when its canonical document is not included")
        source_text = git(repo, "show", sha + ":" + path).decode("utf-8")
        if len(source_text) > 1600:
            stop("Mapped document is too large for automatic redirect-only omission")
        if not any(marker in source_text for marker in ("移動案内", "移動しました", "旧URL", "既存の参照先との互換")):
            stop("Mapped source is no longer an obvious compatibility-only document")
        target = mapping[path]
        candidate_links = links(source_text)
        normalized = {
            posixpath.normpath(posixpath.join(posixpath.dirname(path), unquote(urlsplit(link).path)))
            for link in candidate_links
            if not urlsplit(link).scheme and not urlsplit(link).netloc
        }
        if target not in normalized:
            stop("Mapped document does not link to its canonical destination")

    remaining = selected - excluded
    for path in sorted(remaining):
        if not path.endswith(".md"):
            continue
        text = git(repo, "show", sha + ":" + path).decode("utf-8")
        for link in links(text):
            parsed = urlsplit(link)
            if parsed.scheme or parsed.netloc or link.startswith(("/", "#")):
                continue
            target = posixpath.normpath(
                posixpath.join(posixpath.dirname(path), unquote(parsed.path))
            )
            if parsed.path == "":
                continue  # An anchor in the same source file.
            if target.startswith("../") or target == "..":
                stop("Link escapes the repository in retained document: " + path)
            if target in excluded:
                stop("Kept document still links to a removed compatibility path: " + path)
            if target not in remaining:
                directory_index = posixpath.join(target, "README.md")
                if directory_index not in remaining:
                    stop("Retained document has an unresolved local link: " + path)

    # File selection is deterministic. A final content and link audit remains required.
    temp = output.parent / (output.name + ".incomplete")
    if temp.exists() or temp.is_symlink():
        stop("Output staging path already exists")
    try:
        with temp.open("x", encoding="utf-8") as stream:
            for path in sorted(remaining):
                stream.write(path + "\n")
        # Hardlink creation refuses to overwrite a concurrent writer's output.
        # The staging file is in the same directory, hence on the same device.
        os.link(temp, output)
    finally:
        if temp.exists():
            temp.unlink()
    print(f"PUBLIC_CANDIDATE_PATHS_ONLY included={len(remaining)} omitted={len(excluded)}")
    print("NOT_APPROVED: content, historical references and external links require independent review.")


if __name__ == "__main__":
    main()
