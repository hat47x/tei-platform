#!/usr/bin/env python3
"""Finite synthetic contrasts: business-claim evidence vs acquisition qualification.

A comparison fixture only. It does not inspect a real Provider/Host or enforce
access policy, and its dictionaries are not TEI canonical or wire schemas.
"""
from __future__ import annotations

from dataclasses import dataclass
import unittest


@dataclass(frozen=True)
class Observation:
    displayed: tuple[tuple[str, object], ...]
    # An explicit witness of the hypothetical completed business condition.
    completion_witness: bool
    # Read trace refers to physical fields actually consulted by the Provider.
    physical_fields: tuple[str, ...] | None
    # The fixture assumes the stated access policy has been independently accepted.
    allowed_fields: frozenset[str]
    # The fixture's current-claim contract requires a confirmed current route.
    provider_route: str
    route_current_confirmed: bool
    # A cited Markdown/DSL passage is merely a text span until role and meaning bind.
    source_role: str
    meaning_accepted: bool
    # The view Consumer may have a narrower disclosure permission than the auditor.
    consumer_may_disclose_witness: bool


def physical_acquisition(obs: Observation) -> str:
    if obs.physical_fields is None:
        return "unknown"
    if not set(obs.physical_fields).issubset(obs.allowed_fields):
        return "prohibited-by-fixture-policy"
    return "allowed-by-fixture-policy"


def current_witness_qualified(obs: Observation) -> bool:
    return obs.route_current_confirmed and obs.provider_route == "primary-current"


def role_qualified(obs: Observation) -> bool:
    return obs.source_role == "accepted-rule" and obs.meaning_accepted


def disclosure(obs: Observation) -> str:
    return "available-to-consumer" if obs.consumer_may_disclose_witness else "withheld"


BASE = dict(
    displayed=(("order", "H1"), ("status", "submitted")),
    completion_witness=True,
    physical_fields=("order.id", "order.status"),
    allowed_fields=frozenset({"order.id", "order.status"}),
    provider_route="primary-current",
    route_current_confirmed=True,
    source_role="accepted-rule",
    meaning_accepted=True,
    consumer_may_disclose_witness=True,
)


def sample(**overrides: object) -> Observation:
    return Observation(**(BASE | overrides))


class SeparateClaims(unittest.TestCase):
    def test_ac01_equal_output_and_completion_different_physical_access(self) -> None:
        narrow = sample()
        broad = sample(physical_fields=("order.id", "order.status", "customer.private_note"))
        self.assertEqual(narrow.displayed, broad.displayed)
        self.assertEqual(narrow.completion_witness, broad.completion_witness)
        self.assertEqual(physical_acquisition(narrow), "allowed-by-fixture-policy")
        self.assertEqual(physical_acquisition(broad), "prohibited-by-fixture-policy")

    def test_ac02_no_trace_is_unknown_not_approved_or_prohibited(self) -> None:
        no_trace = sample(physical_fields=None)
        self.assertTrue(no_trace.completion_witness)
        self.assertEqual(physical_acquisition(no_trace), "unknown")

    def test_ac03_same_values_primary_and_replica_are_not_same_freshness_proof(self) -> None:
        primary = sample()
        replica = sample(provider_route="replica", route_current_confirmed=False)
        self.assertEqual(primary.displayed, replica.displayed)
        self.assertTrue(current_witness_qualified(primary))
        self.assertFalse(current_witness_qualified(replica))

    def test_ac04_same_text_not_same_authoritative_rule(self) -> None:
        accepted = sample()
        quotation = sample(source_role="quoted-example", meaning_accepted=False)
        self.assertEqual(accepted.displayed, quotation.displayed)
        self.assertTrue(role_qualified(accepted))
        self.assertFalse(role_qualified(quotation))

    def test_ac05_internal_witness_not_automatically_external_disclosure(self) -> None:
        internal = sample()
        public = sample(consumer_may_disclose_witness=False)
        self.assertEqual(internal.displayed, public.displayed)
        self.assertEqual(internal.completion_witness, public.completion_witness)
        self.assertEqual(disclosure(internal), "available-to-consumer")
        self.assertEqual(disclosure(public), "withheld")

    def test_ac06_access_policy_is_not_equivalent_to_minimal_read_footprint(self) -> None:
        broad_but_allowed = sample(
            physical_fields=("order.id", "order.status", "order.join_key"),
            allowed_fields=frozenset({"order.id", "order.status", "order.join_key"}),
        )
        self.assertEqual(physical_acquisition(broad_but_allowed), "allowed-by-fixture-policy")
        self.assertNotEqual(broad_but_allowed.physical_fields, BASE["physical_fields"])


if __name__ == "__main__":
    unittest.main(verbosity=2)
