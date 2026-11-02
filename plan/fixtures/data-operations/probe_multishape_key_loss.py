#!/usr/bin/env python3
"""Synthetic reference ONLY: key-space/multiplicity loss in cross-shape adapters.

This file is not a TEI implementation or an Oracle for production behavior.
Run: python3 -m unittest -v probe_multishape_key_loss.py
"""
import json
import sqlite3
import unittest
from collections import defaultdict


class MultiShapeKeyLossProbe(unittest.TestCase):
    def test_k01_duplicate_rows_require_explicit_policy(self):
        rows = [("A", 2), ("A", 3)]
        naive = {k: v for k, v in rows}
        self.assertEqual(len(naive), 1)
        self.assertEqual(naive["A"], 3)
        seen = set()
        duplicates = []
        for key, _ in rows:
            if key in seen:
                duplicates.append(key)
            seen.add(key)
        self.assertEqual(duplicates, ["A"])

    def test_k02_grouped_bag_is_not_unique_map(self):
        rows = [("A", 2), ("A", 3)]
        grouped = defaultdict(list)
        for key, value in rows:
            grouped[key].append(value)
        self.assertEqual(grouped["A"], [2, 3])
        self.assertEqual(sum(grouped["A"]), 5)
        self.assertNotEqual(sum(grouped["A"]), dict(rows)["A"])

    def test_k03_typed_key_to_json_property_collision(self):
        typed_map = {1: "integer", "1": "string"}
        self.assertEqual(len(typed_map), 2)
        raw = json.dumps(typed_map)
        pairs = json.loads(raw, object_pairs_hook=list)
        self.assertEqual([key for key, _ in pairs], ["1", "1"])
        self.assertEqual(json.loads(raw), {"1": "string"})

    def test_k04_duplicate_raw_json_property_disappears(self):
        raw = '{"code":1,"code":2}'
        self.assertEqual(json.loads(raw), {"code": 2})
        pairs = json.loads(raw, object_pairs_hook=list)
        self.assertEqual(pairs, [("code", 1), ("code", 2)])
        self.assertNotEqual(len(pairs), len(json.loads(raw)))

    def test_k05_sql_bag_to_map_fanout_loss(self):
        connection = sqlite3.connect(":memory:")
        try:
            connection.execute("CREATE TABLE lines (order_id TEXT, amount INTEGER)")
            connection.executemany("INSERT INTO lines VALUES (?,?)", [("A", 2), ("A", 3)])
            rows = connection.execute("SELECT order_id, amount FROM lines").fetchall()
            self.assertEqual(connection.execute("SELECT COUNT(*) FROM lines").fetchone()[0], 2)
            self.assertEqual(len(dict(rows)), 1)
        finally:
            connection.close()

    def test_k06_nonunique_key_cannot_choose_update_target(self):
        rows = [{"line_id": 10, "sku": "X"}, {"line_id": 20, "sku": "X"}]
        targets = [row for row in rows if row["sku"] == "X"]
        self.assertEqual(len(targets), 2)
        self.assertNotEqual(targets[0]["line_id"], targets[1]["line_id"])

    def test_k07_map_equality_does_not_imply_order_equivalence(self):
        first = {"x": 1, "y": 2}
        second = {"y": 2, "x": 1}
        self.assertEqual(first, second)
        self.assertNotEqual(list(first.items()), list(second.items()))
        self.assertEqual(sorted(first.items()), sorted(second.items()))

    def test_k08_graph_parallel_edges_are_distinct_occurrences(self):
        edges = [("u", "v", "e1"), ("u", "v", "e2")]
        mapped = {(src, dst): edge_id for src, dst, edge_id in edges}
        self.assertEqual(len(edges), 2)
        self.assertEqual(len(mapped), 1)
        self.assertEqual(mapped[("u", "v")], "e2")


if __name__ == "__main__":
    unittest.main()
