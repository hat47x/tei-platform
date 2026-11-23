#!/usr/bin/env python3
"""GitHub実設定の読み取りを、通信しない模擬応答で検証する。"""

import copy
import importlib.util
import json
from pathlib import Path
import subprocess
import unittest
from unittest.mock import patch

ROOT = Path(__file__).resolve().parent
SPEC = importlib.util.spec_from_file_location(
    "tei_main_protection_readback", ROOT / "verify-main-protection.py")
VERIFY = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(VERIFY)
POLICY = json.loads((ROOT / "main-pr-only.json").read_text(encoding="utf-8"))


def summary(rule_id=42, *, source="Repository", name=None):
    return {
        "id": rule_id,
        "name": VERIFY.NAME if name is None else name,
        "source_type": source,
        "enforcement": "active",
    }


def detail(rule_id=42):
    value = copy.deepcopy(POLICY)
    value.update(id=rule_id, source_type="Repository")
    return value


class TestReadOnlyRulesetDiscovery(unittest.TestCase):
    def mock_readback(self, pages, *, response=None):
        seen = []
        actual_detail = detail() if response is None else response

        def fake_get(endpoint):
            seen.append(endpoint)
            prefix = f"repos/{VERIFY.REPOSITORY}/rulesets?per_page=100&page="
            if endpoint.startswith(prefix):
                return pages.get(int(endpoint[len(prefix):]), [])
            if endpoint == f"repos/{VERIFY.REPOSITORY}/rulesets/42":
                return actual_detail
            self.fail(f"Unexpected GitHub API endpoint: {endpoint}")

        with patch.object(VERIFY, "gh_json", side_effect=fake_get):
            result = VERIFY.verify_live()
        return result, seen

    def test_one_page_exact_match_succeeds(self):
        result, calls = self.mock_readback({1: [summary()]})
        self.assertEqual(([], 42), result)
        self.assertEqual(2, len(calls))

    def test_missing_ruleset_is_not_success(self):
        result, calls = self.mock_readback({1: []})
        self.assertTrue(result[0])
        self.assertIsNone(result[1])
        self.assertEqual(1, len(calls))

    def test_organization_rule_is_not_mistaken_for_repo_rule(self):
        result, _ = self.mock_readback({1: [summary(source="Organization")]})
        self.assertTrue(result[0])

    def test_hidden_second_page_duplicate_is_rejected(self):
        first = [summary()] + [
            summary(rule_id=1000 + i, name=f"other-{i}") for i in range(99)]
        second = [summary(rule_id=99)]
        result, calls = self.mock_readback({1: first, 2: second})
        self.assertTrue(result[0])
        self.assertIn("page=2", calls[1])
        self.assertNotIn(f"/rulesets/42", calls)

    def test_candidate_on_second_page_is_found(self):
        first = [summary(rule_id=1000 + i, name=f"other-{i}")
                 for i in range(100)]
        result, calls = self.mock_readback({1: first, 2: [summary()]})
        self.assertEqual(([], 42), result)
        self.assertIn("page=2", calls[1])

    def test_more_than_twenty_pages_fails_closed(self):
        page = [summary(rule_id=1000 + i, name=f"other-{i}")
                for i in range(100)]
        pages = {i: page for i in range(1, 21)}
        result, calls = self.mock_readback(pages)
        self.assertTrue(result[0])
        self.assertIsNone(result[1])
        self.assertEqual(20, len(calls))

    def test_malformed_or_oversized_page_is_rejected(self):
        # 100件未満の要素がないページ（＝終端）に達する前に、
        # 無効な応答や規定を超えるページ数の安全性チェックが正しく働くことを確認する。
        page = [summary(rule_id=1000 + i, name=f"other-{i}") for i in range(100)]
        pages = {i: page for i in range(1, 21)}
        result, calls = self.mock_readback(pages)
        self.assertTrue(result[0])
        self.assertIsNone(result[1])

    def test_non_list_page_response_fails_closed(self):
        result, calls = self.mock_readback({1: {"not": "a list"}})
        self.assertTrue(result[0])
        self.assertIsNone(result[1])