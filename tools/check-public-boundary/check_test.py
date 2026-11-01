import unittest

from check import scan_text


class ScanTextTest(unittest.TestCase):
    def test_private_repository_name_is_rejected(self):
        self.assertEqual(
            scan_text(f"非公開の資料庫（{'tei-platform-' + 'private-resources'}）を参照"),
            ["repository-name"],
        )

    def test_archive_path_is_rejected(self):
        self.assertEqual(
            scan_text(f"所在は {'archives/' + 'x-tea-2026-10-14'} の中身を参照"),
            ["archive-path"],
        )

    def test_research_note_path_is_rejected(self):
        self.assertEqual(
            scan_text(
                f"根拠は {'research/low-code-management-tools/' + 'xtea/N-open-questions.md'}"
            ),
            ["research-note-path"],
        )

    def test_clean_text_passes(self):
        self.assertEqual(scan_text("公開してよい合成fixtureです。"), [])


if __name__ == "__main__":
    unittest.main()
