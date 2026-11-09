"""Linux-only diagnostic E1 executable-byte pinning before API invocation.

Existing measurement.json provides the expected SHA-256/size; this does NOT
authenticate that measurement, authorize a Host, verify a Builder, or create a
new Artifact Lock. The smoke Host executes a sealed, checked byte snapshot
rather than reopening a mutable pathname after its preflight.
"""
from contextlib import contextmanager
import fcntl
import hashlib
import os
from pathlib import Path
import re
import stat
import subprocess


HEX256 = re.compile(r"[0-9a-f]{64}\Z")
MAX_API_BYTES = 256 * 1024 * 1024


@contextmanager
def sealed_executable(binary_path, expected_sha256, expected_size):
    """Yield an executable memfd containing *exactly* the expected API bytes.

    Both path substitution and in-place edits after the copy cannot affect
    the final sealed file. This requires Linux /proc/self/fd, memfd and seals,
    and deliberately fails closed when any of those primitives are absent.
    """
    if (not isinstance(expected_sha256, str) or
            not HEX256.fullmatch(expected_sha256) or
            type(expected_size) is not int or
            not 0 < expected_size <= MAX_API_BYTES):
        raise ValueError("invalid expected diagnostic API artifact identity")
    if not hasattr(os, "memfd_create") or not hasattr(os, "O_NOFOLLOW"):
        raise RuntimeError("diagnostic API executable pinning requires Linux memfd and O_NOFOLLOW")

    source_fd = os.open(Path(binary_path), os.O_RDONLY | os.O_CLOEXEC | os.O_NOFOLLOW)
    try:
        info = os.fstat(source_fd)
        if not stat.S_ISREG(info.st_mode) or not (info.st_mode & 0o111):
            raise ValueError("diagnostic API executable must be an executable regular file")
        if info.st_size != expected_size:
            raise ValueError("diagnostic API file size differs from measurement")

        # An independent anonymous byte snapshot: no writable pathname or
        # alternative file can replace the image between hash and exec.
        executable_fd = os.memfd_create(
            "tei-e1-checked-api", os.MFD_CLOEXEC | os.MFD_ALLOW_SEALING,
        )
        try:
            copied = 0
            digest = hashlib.sha256()
            while True:
                block = os.read(source_fd, 1024 * 1024)
                if not block:
                    break
                copied += len(block)
                if copied > expected_size:
                    raise ValueError("diagnostic API file changed during snapshot")
                digest.update(block)
                offset = 0
                while offset < len(block):
                    offset += os.write(executable_fd, block[offset:])
            if copied != expected_size or digest.hexdigest() != expected_sha256:
                raise ValueError("diagnostic API bytes differ from measurement")
            os.fchmod(executable_fd, 0o500)
            seals = (fcntl.F_SEAL_WRITE | fcntl.F_SEAL_GROW |
                     fcntl.F_SEAL_SHRINK | fcntl.F_SEAL_SEAL)
            fcntl.fcntl(executable_fd, fcntl.F_ADD_SEALS, seals)
            if fcntl.fcntl(executable_fd, fcntl.F_GET_SEALS) & seals != seals:
                raise RuntimeError("diagnostic API executable could not be sealed")
            yield executable_fd
        finally:
            os.close(executable_fd)
    finally:
        os.close(source_fd)


def launch_verified_api(binary_path, arguments, expected_sha256, expected_size, log):
    """Spawn exactly the immutable bytes checked above, not the source path."""
    with sealed_executable(binary_path, expected_sha256, expected_size) as fd:
        return subprocess.Popen(
            [f"/proc/self/fd/{fd}", *arguments],
            stdout=log,
            stderr=subprocess.STDOUT,
            pass_fds=(fd,),
        )
