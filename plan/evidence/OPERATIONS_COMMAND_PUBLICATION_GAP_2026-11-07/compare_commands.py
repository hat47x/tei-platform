#!/usr/bin/env python3
"""Compare the commands the CLI accepts with the commands it advertises.

The CLI dispatches `acceptedCommands` (cmd/tei/cli_commands.go) but only
describes a subset in `commandCatalog()` (cmd/tei/cli_contract.go). A caller
that discovers commands from `tei commands` cannot learn about a command that is
accepted but not advertised, and a command that is advertised but not accepted
fails at dispatch.

This reference reads the two Go source files as text and compares the NAME
lists only. It does not run the CLI, does not parse Go, and makes no claim about
flags, effects or output shapes.

Usage: compare_commands.py <repo-root> <output.txt>
"""
import re
import sys

ACCEPTED_BLOCK = re.compile(r"acceptedCommands = \[\]string\{(.*?)\n\}", re.S)
CATALOG_NAME = re.compile(r'Name:\s+"([^"]+)"')
STRING = re.compile(r'"([^"]+)"')


def read(path):
    with open(path, encoding="utf-8") as handle:
        return handle.read()


def accepted(root):
    source = read(root + "/cmd/tei/cli_commands.go")
    block = ACCEPTED_BLOCK.search(source)
    if not block:
        raise SystemExit("acceptedCommands block not found")
    names = STRING.findall(block.group(1))
    # helperDispatchedCommands are appended in init(); they are accepted too.
    helper = re.search(r"helperDispatchedCommands = \[\]string\{(.*?)\n\}", source, re.S)
    if helper:
        names += STRING.findall(helper.group(1))
    return names


def advertised(root):
    source = read(root + "/cmd/tei/cli_contract.go")
    body = re.search(r"func commandCatalog\(\).*", source, re.S).group(0)
    # "model catalog" is one advertised name made of two argv words. Compare the
    # argv form so the two lists are about the same thing.
    return [n.split(" ")[0] if n == "model catalog" else n for n in CATALOG_NAME.findall(body)]


def main() -> int:
    root, out = sys.argv[1], sys.argv[2]
    acc, adv = accepted(root), advertised(root)
    missing = [n for n in acc if n not in adv]
    extra = [n for n in adv if n not in acc]
    lines = [
        "accepted commands: %d" % len(acc),
        "advertised commands: %d" % len(adv),
        "",
        "accepted but not advertised (%d):" % len(missing),
    ]
    lines += ["  " + n for n in missing]
    lines += ["", "advertised but not accepted (%d):" % len(extra)]
    lines += ["  " + n for n in extra]
    with open(out, "w", encoding="utf-8", newline="\n") as handle:
        handle.write("\n".join(lines) + "\n")
    print("\n".join(lines))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
