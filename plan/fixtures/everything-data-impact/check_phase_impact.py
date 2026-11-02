"""Synthetic counterexamples to one-digest reuse for TEI Everything is Data.

An independent characterization fixture, NOT a product compiler/cache/effect test.
The fixture deliberately over-invalidates downstream stages; it does not infer
semantic equivalence from formatted text or claim an optimal incremental plan.
"""
from dataclasses import dataclass, replace
import unittest


@dataclass(frozen=True)
class Evidence:
    source_location: str = "model.md#function-order"
    source_bytes: str = "original-source"
    grammar: str = "tei-surface@1"
    role_binding: str = "function@1"
    operator: str = "std.sum@1"
    context_mapping: str = "order.lines:all@1"
    rule_dependencies: str = "credit-limit@1"
    provider_binding: str = "primary-db@1"
    selection_epoch: str = "selection-A@1"
    grant: str = "host-grant@1"
    document_edit: str = "editorial@1"
    publication_generation: str = "published@1"
    disclosure: str = "internal@1"


def signatures(e):
    """Conservative, phase-qualified comparisons; no production cache keys."""
    source = (e.source_location, e.source_bytes)
    parsed = (e.source_bytes, e.grammar)
    meaning = (parsed, e.role_binding, e.operator)
    read = (meaning, e.context_mapping, e.rule_dependencies,
            e.provider_binding, e.selection_epoch)
    runtime = (read, e.grant)
    document = (source, meaning, e.document_edit)
    publication = (document, e.publication_generation, e.disclosure)
    return dict(source=source, parsed=parsed, meaning=meaning, read=read,
                runtime=runtime, document=document, publication=publication)


class ImpactCounterexamples(unittest.TestCase):
    def test_independent_dimensions_and_required_requalification(self):
        # Expected phases to differ on each isolated change (non-exhaustive).
        # These are *qualification scopes*, not predicates that prove effects.
        cases = [
            ("source_bytes", "changed source", {"source", "parsed", "meaning", "document"}, set()),
            ("grammar", "tei-surface@2", {"parsed", "meaning", "runtime"}, {"source"}),
            ("role_binding", "test-case@1", {"meaning", "read", "document"}, {"source"}),
            ("operator", "std.sum@2", {"meaning", "read", "runtime"}, {"source"}),
            ("context_mapping", "order.lines:page1@1", {"read", "runtime"}, {"meaning"}),
            ("rule_dependencies", "credit-limit@2", {"read", "runtime"}, {"meaning"}),
            ("provider_binding", "archive-db@1", {"read", "runtime"}, {"meaning"}),
            ("selection_epoch", "selection-B@1", {"read", "runtime"}, {"meaning"}),
            ("grant", "revoked", {"runtime"}, {"read", "meaning"}),
            ("document_edit", "editorial@2", {"document", "publication"}, {"runtime", "meaning"}),
            ("publication_generation", "published@2", {"publication"}, {"document", "runtime"}),
            ("disclosure", "public@2", {"publication"}, {"runtime", "meaning"}),
            ("source_location", "other.md#function-order", {"source", "document"}, {"parsed", "meaning"}),
        ]
        original = Evidence()
        for field, new_value, must_change, must_stay in cases:
            with self.subTest(dimension=field):
                lhs = signatures(original)
                rhs = signatures(replace(original, **{field: new_value}))
                for phase in must_change:
                    self.assertNotEqual(lhs[phase], rhs[phase], phase)
                for phase in must_stay:
                    self.assertEqual(lhs[phase], rhs[phase], phase)

    def test_same_source_does_not_prove_same_meaning_or_effect_qualification(self):
        baseline = signatures(Evidence())
        for dimension, value, affected_phase in [
            ("role_binding", "quoted-example@2", "meaning"),
            ("operator", "std.sum@2", "meaning"),
            ("context_mapping", "order.lines:partial@2", "read"),
            ("rule_dependencies", "credit-limit@2", "read"),
            ("provider_binding", "archive-db@2", "read"),
            ("grant", "revoked", "runtime"),
            ("disclosure", "external@2", "publication"),
        ]:
            with self.subTest(dimension=dimension):
                candidate = signatures(replace(Evidence(), **{dimension: value}))
                self.assertEqual(baseline["source"], candidate["source"])
                self.assertNotEqual(baseline[affected_phase], candidate[affected_phase])

    def test_same_meaning_does_not_prove_same_consumer_document_or_runtime(self):
        baseline = signatures(Evidence())
        for dimension, value, phase in [
            ("source_location", "renamed.md#function-order", "document"),
            ("document_edit", "new explanation", "document"),
            ("provider_binding", "secondary-db@1", "runtime"),
            ("grant", "revoked", "runtime"),
            ("publication_generation", "published@2", "publication"),
        ]:
            with self.subTest(dimension=dimension):
                candidate = signatures(replace(Evidence(), **{dimension: value}))
                self.assertEqual(baseline["meaning"], candidate["meaning"])
                self.assertNotEqual(baseline[phase], candidate[phase])


if __name__ == "__main__":
    unittest.main()
