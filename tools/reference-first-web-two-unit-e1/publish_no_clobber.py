#!/usr/bin/env python3
"""Publish an E1 diagnostic artifact directory without replacing any destination.

The staged source and its destination must be on the same Linux filesystem.
Linux renameat2(RENAME_NOREPLACE) makes the destination-existence decision
atomic; a plain 'mv' may instead move the source *inside* a concurrently
created directory or follow a newly created symlink.
"""
import ctypes
import errno
import os
from pathlib import Path
import stat
import sys


AT_FDCWD = -100
RENAME_NOREPLACE = 1


def publish(source: Path, destination: Path) -> None:
    source = Path(os.path.abspath(source))
    destination = Path(os.path.abspath(destination))
    if source == destination:
        raise ValueError("source and destination must differ")
    if not stat.S_ISDIR(source.lstat().st_mode):
        raise ValueError("staged source must be a real directory")
    if source.parent.stat().st_dev != destination.parent.stat().st_dev:
        raise OSError(errno.EXDEV, "staging and destination must be on one filesystem")

    # Refuse publication rather than fall back to racy check-then-rename or
    # a copy when the kernel/libc does not provide this operation.
    libc = ctypes.CDLL(None, use_errno=True)
    renameat2 = getattr(libc, "renameat2", None)
    if renameat2 is None:
        raise RuntimeError("atomic no-replace directory publication unavailable")
    renameat2.argtypes = [
        ctypes.c_int, ctypes.c_char_p, ctypes.c_int, ctypes.c_char_p, ctypes.c_uint
    ]
    renameat2.restype = ctypes.c_int
    code = renameat2(
        AT_FDCWD, os.fsencode(source), AT_FDCWD,
        os.fsencode(destination), RENAME_NOREPLACE,
    )
    if code != 0:
        error = ctypes.get_errno()
        raise OSError(error, os.strerror(error), str(destination))


if __name__ == "__main__":
    if len(sys.argv) != 3:
        raise SystemExit("usage: publish_no_clobber.py STAGED_DIRECTORY NEW_DESTINATION")
    try:
        publish(Path(sys.argv[1]), Path(sys.argv[2]))
    except (OSError, RuntimeError, ValueError) as exc:
        raise SystemExit(f"E1 artifact publication refused: {exc}") from exc
