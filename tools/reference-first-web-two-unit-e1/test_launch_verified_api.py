"""Real Linux executable and hostile-path negatives for E1 diagnostic Host."""
import fcntl
import hashlib
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

from launch_verified_api import launch_verified_api, sealed_executable


@unittest.skipUnless(
    hasattr(os, "memfd_create") and Path("/proc/self/fd").is_dir(),
    "Linux executable memfd and /proc/self/fd are required",
)
class SealedDiagnosticLaunchTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix="tei-e1-pinned-exe-")
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.source = self.root / "order-entry"
        shutil.copy2("/usr/bin/true", self.source)
        data = self.source.read_bytes()
        self.size = len(data)
        self.digest = hashlib.sha256(data).hexdigest()

    def test_executes_exact_verified_bytes(self):
        process = launch_verified_api(
            self.source, [], self.digest, self.size, subprocess.DEVNULL,
        )
        self.assertEqual(process.wait(timeout=5), 0)

    def test_path_replacement_after_verification_cannot_swap_executable(self):
        # Opening another executable after a traditional path hash would run
        # its new bytes. In contrast, the already sealed snapshot stays
        # attached to the original executable even when its name is replaced.
        with sealed_executable(self.source, self.digest, self.size) as fd:
            replacement = self.root / "replacement"
            shutil.copy2("/usr/bin/false", replacement)
            os.replace(replacement, self.source)
            self.assertNotEqual(
                hashlib.sha256(self.source.read_bytes()).hexdigest(),
                self.digest,
            )
            result = subprocess.run(
                [f"/proc/self/fd/{fd}"], pass_fds=(fd,), check=False,
                stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL,
            )
            self.assertEqual(result.returncode, 0)
            all_seals = (
                fcntl.F_SEAL_WRITE | fcntl.F_SEAL_SHRINK |
                fcntl.F_SEAL_GROW | fcntl.F_SEAL_SEAL
            )
            self.assertEqual(fcntl.fcntl(fd, fcntl.F_GET_SEALS) & all_seals, all_seals)
            with self.assertRaises(OSError):
                os.write(fd, b"override")

    def test_changed_or_replaced_path_is_denied_before_child_launch(self):
        shutil.copy2("/usr/bin/false", self.source)
        with self.assertRaisesRegex(ValueError, "size differs|bytes differ"):
            launch_verified_api(self.source, [], self.digest, self.size, subprocess.DEVNULL)

    def test_symlink_and_non_executable_are_denied(self):
        self.source.unlink()
        self.source.symlink_to("/usr/bin/true")
        with self.assertRaises(OSError):
            launch_verified_api(self.source, [], self.digest, self.size, subprocess.DEVNULL)
        self.source.unlink()
        shutil.copy2("/usr/bin/true", self.source)
        self.source.chmod(0o400)
        with self.assertRaisesRegex(ValueError, "executable regular file"):
            launch_verified_api(self.source, [], self.digest, self.size, subprocess.DEVNULL)

    def test_invalid_receipt_identity_is_denied(self):
        for sha, n in (
            ("X" * 64, self.size),
            (self.digest, 0),
            (self.digest, True),
            (self.digest, 256 * 1024 * 1024 + 1),
            (self.digest, self.size + 1),
        ):
            with self.subTest(sha=sha, n=n), self.assertRaises(ValueError):
                launch_verified_api(self.source, [], sha, n, subprocess.DEVNULL)


if __name__ == "__main__":
    unittest.main()
