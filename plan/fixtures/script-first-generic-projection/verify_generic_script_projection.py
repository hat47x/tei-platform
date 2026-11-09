"""Synthetic, read-only/patch-candidate characterization for a script-first generic UI.

No TEI parser, Studio UI, provider, permission authority, or canonical write is used.
The 'language' descriptor is stipulated by the fixture, not inferred from text.
Run: python3 -m unittest -v verify_generic_script_projection.py
"""
from __future__ import annotations

import hashlib
import unittest
from dataclasses import dataclass
from typing import Callable, Mapping


@dataclass(frozen=True)
class RuleDescriptor:
    node_id: str
    target: str
    dependency: str
    expression: str
    source_sha256: str
    span_start: int  # UTF-8 byte offset, not Unicode character index
    span_end: int
    authored: bool = True
    source_occurrences: int = 1
    effect: str = "pure"


@dataclass(frozen=True)
class ReadContext:
    revision: str
    epoch: int
    scope: str
    values: Mapping[str, object]
    disclosure: frozenset[str]
    complete: bool = True


class CharacterizationError(ValueError):
    pass


def sha256(source: bytes) -> str:
    return hashlib.sha256(source).hexdigest()


def proposed_edit(
    source: bytes,
    descriptor: RuleDescriptor,
    *,
    revision: str,
    expected_revision: str,
    target: str,
    replacement: bytes,
) -> bytes:
    """Build a patch candidate only; never commits to an authoritative source."""
    if revision != expected_revision:
        raise CharacterizationError("stale-source")
    if descriptor.source_sha256 != sha256(source):
        raise CharacterizationError("source-digest-drift")
    if descriptor.target != target:
        raise CharacterizationError("wrong-target")
    if not descriptor.authored or descriptor.source_occurrences != 1:
        raise CharacterizationError("readonly-derived-or-ambiguous")
    if not (0 <= descriptor.span_start < descriptor.span_end <= len(source)):
        raise CharacterizationError("invalid-byte-span")
    if source[descriptor.span_start : descriptor.span_end] != descriptor.expression.encode("utf-8"):
        raise CharacterizationError("source-span-mismatch")
    # The result is proposed bytes, not an accepted/committed TEI model revision.
    return source[: descriptor.span_start] + replacement + source[descriptor.span_end :]


def generic_inspect(descriptor: RuleDescriptor) -> dict:
    """A new plugin can populate this read-only view without a UI-specific widget."""
    return {
        "node": descriptor.node_id,
        "target": descriptor.target,
        "depends_on": [descriptor.dependency],
        "effect": descriptor.effect,
        "span": [descriptor.span_start, descriptor.span_end],
        "editable_candidate": descriptor.authored and descriptor.source_occurrences == 1,
    }


def inspect_control(
    descriptor: RuleDescriptor,
    context: ReadContext,
    *,
    expected_revision: str,
    expected_epoch: int,
    expected_scope: str,
    plugin_eval: Callable[[object], bool | None],
) -> dict:
    """Evaluate only a supplied plugin callback; UI never guesses operator meaning.

    Context is assumed already purpose/disclosure projected by its authority.
    'unavailable' is intentionally identical for absent and unauthorized data.
    """
    if (context.revision, context.epoch, context.scope) != (
        expected_revision, expected_epoch, expected_scope
    ):
        return {"status": "undetermined", "reason": "stale-context"}
    if not context.complete:
        return {"status": "undetermined", "reason": "incomplete-context"}
    dep = descriptor.dependency
    if dep not in context.disclosure or dep not in context.values:
        return {"status": "undetermined", "reason": "unavailable"}
    if context.values[dep] is None:
        return {"status": "undetermined", "reason": "null-input"}
    result = plugin_eval(context.values[dep])
    if result is None:
        return {"status": "undetermined", "reason": "plugin-unknown"}
    if type(result) is not bool:
        raise CharacterizationError("invalid-evaluator-result")
    return {"status": "true" if result else "false", "reason": "evaluated"}


def presentation_offer(evaluation: dict, *, is_visibility: bool = False) -> dict:
    """A rendering choice, not an authorization grant or a business outcome."""
    status = evaluation["status"]
    return {
        "renderable": status == "true" if is_visibility else True,
        "enabled_hint": status == "true" and not is_visibility,
        "evaluation": status,
    }


class ScriptFirstGenericProjectionTests(unittest.TestCase):
    def setUp(self):
        self.expression = 'customer.kind == "corporate"'
        self.source = ('// 表示条件\nvisible(order.discount) = ' + self.expression + '\n').encode("utf-8")
        start = self.source.index(self.expression.encode("utf-8"))
        self.descriptor = RuleDescriptor(
            node_id="order.discount.visible", target="order.discount",
            dependency="customer.kind", expression=self.expression,
            source_sha256=sha256(self.source), span_start=start,
            span_end=start + len(self.expression.encode("utf-8")),
        )
        self.ctx = ReadContext(
            revision="R1", epoch=7, scope="order:1001",
            values={"customer.kind": "corporate"},
            disclosure=frozenset({"customer.kind"}),
        )
        self.eval = lambda value: value == "corporate"

    def check(self, context=None):
        return inspect_control(
            self.descriptor, context or self.ctx,
            expected_revision="R1", expected_epoch=7,
            expected_scope="order:1001", plugin_eval=self.eval,
        )

    def test_sf01_utf8_span_proposal_is_byte_exact(self):
        updated = proposed_edit(
            self.source, self.descriptor, revision="R1", expected_revision="R1",
            target="order.discount", replacement=b"false",
        )
        self.assertEqual(updated.count(b"false"), 1)
        self.assertIn("// 表示条件".encode("utf-8"), updated)
        self.assertEqual(self.source[: self.descriptor.span_start], updated[: self.descriptor.span_start])
        self.assertNotIn(self.expression.encode("utf-8"), updated)
        self.assertGreater(self.descriptor.span_start, self.source.decode("utf-8").index(self.expression))

    def test_sf02_stale_revision_rejected(self):
        with self.assertRaisesRegex(CharacterizationError, "stale-source"):
            proposed_edit(self.source, self.descriptor, revision="R2", expected_revision="R1",
                          target="order.discount", replacement=b"false")

    def test_sf03_digest_drift_rejected(self):
        with self.assertRaisesRegex(CharacterizationError, "source-digest-drift"):
            proposed_edit(self.source + b"\n", self.descriptor, revision="R1", expected_revision="R1",
                          target="order.discount", replacement=b"false")

    def test_sf04_derived_and_ambiguous_rejected(self):
        for desc in (RuleDescriptor(**{**vars(self.descriptor), "authored": False}),
                     RuleDescriptor(**{**vars(self.descriptor), "source_occurrences": 2})):
            with self.subTest(desc=desc):
                with self.assertRaisesRegex(CharacterizationError, "readonly-derived-or-ambiguous"):
                    proposed_edit(self.source, desc, revision="R1", expected_revision="R1",
                                  target="order.discount", replacement=b"false")

    def test_sf05_wrong_target_rejected(self):
        with self.assertRaisesRegex(CharacterizationError, "wrong-target"):
            proposed_edit(self.source, self.descriptor, revision="R1", expected_revision="R1",
                          target="order.shipping", replacement=b"false")

    def test_sf06_true_and_false_are_evaluation_not_permission(self):
        self.assertEqual(self.check()["status"], "true")
        other = ReadContext(**{**vars(self.ctx), "values": {"customer.kind": "personal"}})
        self.assertEqual(self.check(other)["status"], "false")
        self.assertTrue(presentation_offer(self.check(), is_visibility=True)["renderable"])
        self.assertFalse(presentation_offer(self.check(other), is_visibility=True)["renderable"])
        self.assertNotIn("authorized", presentation_offer(self.check()))

    def test_sf07_missing_and_withheld_are_observationally_identical(self):
        missing = ReadContext(**{**vars(self.ctx), "values": {}})
        withheld = ReadContext(**{**vars(self.ctx), "disclosure": frozenset()})
        self.assertEqual(self.check(missing), self.check(withheld))
        self.assertEqual(self.check(missing), {"status": "undetermined", "reason": "unavailable"})

    def test_sf08_null_is_not_false(self):
        ctx = ReadContext(**{**vars(self.ctx), "values": {"customer.kind": None}})
        self.assertEqual(self.check(ctx)["status"], "undetermined")
        self.assertNotEqual(self.check(ctx)["status"], "false")

    def test_sf09_stale_context_does_not_eval(self):
        stale = ReadContext(**{**vars(self.ctx), "epoch": 6})
        self.assertEqual(self.check(stale), {"status": "undetermined", "reason": "stale-context"})

    def test_sf10_incomplete_context_does_not_claim_truth(self):
        partial = ReadContext(**{**vars(self.ctx), "complete": False})
        self.assertEqual(self.check(partial)["status"], "undetermined")

    def test_sf11_new_domain_inspects_without_new_widget(self):
        second = RuleDescriptor(**{**vars(self.descriptor), "node_id": "bear.alert.visible",
                                   "target": "bear.alert", "dependency": "sensor.alert_level"})
        original_keys = set(generic_inspect(self.descriptor))
        self.assertEqual(original_keys, set(generic_inspect(second)))
        self.assertEqual(generic_inspect(second)["depends_on"], ["sensor.alert_level"])

    def test_sf12_unknown_plugin_result_is_not_false(self):
        result = inspect_control(self.descriptor, self.ctx,
                                 expected_revision="R1", expected_epoch=7,
                                 expected_scope="order:1001", plugin_eval=lambda _: None)
        self.assertEqual(result, {"status": "undetermined", "reason": "plugin-unknown"})

    def test_sf13_bad_evaluator_contract_is_rejected(self):
        with self.assertRaisesRegex(CharacterizationError, "invalid-evaluator-result"):
            inspect_control(self.descriptor, self.ctx,
                            expected_revision="R1", expected_epoch=7,
                            expected_scope="order:1001", plugin_eval=lambda _: "true")

    def test_sf14_binding_requires_exact_span_not_equal_text_elsewhere(self):
        double = self.source + self.expression.encode("utf-8")
        desc = RuleDescriptor(**{**vars(self.descriptor), "source_sha256": sha256(double),
                                 "source_occurrences": 1})
        result = proposed_edit(double, desc, revision="R1", expected_revision="R1",
                               target="order.discount", replacement=b"false")
        self.assertEqual(result.count(self.expression.encode("utf-8")), 1)
        self.assertEqual(result[-len(self.expression.encode("utf-8")):],
                         self.expression.encode("utf-8"))


if __name__ == "__main__":
    unittest.main()
