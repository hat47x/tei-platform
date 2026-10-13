"""Linux E1 artifact publication must never clobber a competing destination."""
import concurrent.futures
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest


SCRIPT = Path(__file__).with_name("publish_no_clobber.py")


def stage(parent: Path, name: str, contents: str) -> Path:
    source = parent / name / "derived"
    source.mkdir(parents=True)
    (source / "measurement.json").write_text(contents, encoding="utf-8")
    return source


def call(source: Path, destination: Path) -> subprocess.CompletedProcess:
    return subprocess.run(
        [sys.executable, "-B", str(SCRIPT), str(source), str(destination)],
        capture_output=True, text=True, check=False,
    )


@unittest.skipUnless(sys.platform.startswith("linux"), "Linux renameat2 reference")
class PublishNoClobberTests(unittest.TestCase):
    def test_new_destination_publishes_once(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            src = stage(root, "staged", "correct")
            dest = root / "artifact"
            self.assertEqual(call(src, dest).returncode, 0)
            self.assertEqual((dest / "measurement.json").read_text(), "correct")
            self.assertFalse(src.exists())

    def test_preexisting_directory_is_untouched(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            src = stage(root, "staged", "new")
            dest = root / "artifact"
            dest.mkdir()
            (dest / "measurement.json").write_text("original", encoding="utf-8")
            self.assertNotEqual(call(src, dest).returncode, 0)
            self.assertEqual((dest / "measurement.json").read_text(), "original")
            self.assertEqual((src / "measurement.json").read_text(), "new")
            self.assertFalse((dest / "derived").exists())

    def test_preexisting_file_or_symlink_is_untouched(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            src = stage(root, "staged", "new")
            dest = root / "artifact"
            dest.write_text("existing", encoding="utf-8")
            self.assertNotEqual(call(src, dest).returncode, 0)
            self.assertEqual(dest.read_text(), "existing")
            dest.unlink()
            external = root / "elsewhere"
            external.mkdir()
            dest.symlink_to(external, target_is_directory=True)
            self.assertNotEqual(call(src, dest).returncode, 0)
            self.assertTrue(dest.is_symlink())
            self.assertEqual(list(external.iterdir()), [])
            self.assertTrue(src.is_dir())

    def test_concurrent_publish_allows_one_winner_only(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            first = stage(root, "first", "first")
            second = stage(root, "second", "second")
            dest = root / "artifact"
            with concurrent.futures.ThreadPoolExecutor(max_workers=2) as pool:
                futures = [pool.submit(call, x, dest) for x in (first, second)]
                outcomes = [f.result() for f in futures]
            self.assertEqual(sum(x.returncode == 0 for x in outcomes), 1)
            winner = (dest / "measurement.json").read_text()
            self.assertIn(winner, ("first", "second"))
            self.assertEqual(first.exists(), winner != "first")
            self.assertEqual(second.exists(), winner != "second")

    def test_missing_staging_does_not_create_output(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = Path(tmp)
            dest = root / "artifact"
            self.assertNotEqual(call(root / "missing", dest).returncode, 0)
            self.assertFalse(dest.exists())


if __name__ == "__main__":
    unittest.main()
