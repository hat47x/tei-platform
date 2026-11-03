#!/usr/bin/env python3
"""Build and run Driver control checks; XeadServer is excluded.

A skipped live Driver check is reported in its own summary.json, not as passed.
"""
import os
from pathlib import Path
import subprocess
import sys

ROOT = Path(__file__).resolve().parent


def main():
    build = ["cmd.exe", "/c", "build.cmd"] if os.name == "nt" else ["sh", "build.sh"]
    commands = [build] + [[sys.executable, "tests/test_" + name + ".py"]
                          for name in ("inventory", "driver_result", "integration", "live_driver")]
    for command in commands:
        result = subprocess.run(command, cwd=ROOT)
        if result.returncode:
            return result.returncode
    return 0


if __name__ == "__main__":
    sys.exit(main())
