"""Synthetic reference: route choice is not a disclosure-filtered choice.

This is not the TEI OperationSurface implementation or a Host authorization test.
"""
from dataclasses import dataclass, replace
import unittest


@dataclass(frozen=True)
class Route:
    alias: str
    endpoint: str
    mapping_version: str
    scope: str
    purpose: str
    authority_profile: str
    granted: bool = True
    fresh: bool = True
    disclosed: bool = True

    @property
    def meaning_route(self):
        return (self.endpoint, self.mapping_version, self.scope,
                self.purpose, self.authority_profile)


@dataclass(frozen=True)
class Decision:
    status: str
    matched: str = ""
    diagnostic: str = ""


def resolve(routes, purpose, explicit_endpoint=None):
    """Candidate qualification model; explicitly not an execution request."""
    groups = {}
    for r in routes:
        if r.purpose == purpose:
            groups.setdefault(r.meaning_route, []).append(r)
    if explicit_endpoint is not None:
        groups = {key: members for key, members in groups.items()
                  if key[0] == explicit_endpoint}
        if not groups:
            return Decision("unresolved", diagnostic="no matching route")
    if not groups:
        return Decision("unresolved", diagnostic="no applicable route")
    if len(groups) != 1:
        return Decision("ambiguous", diagnostic="distinct route classes")
    key, members = next(iter(groups.items()))
    # Same class only when scope, mapping, purpose and authority profile agree.
    if not any(r.granted for r in members):
        return Decision("denied", diagnostic="no current grant")
    if not any(r.granted and r.fresh for r in members):
        return Decision("stale", diagnostic="no current eligible witness")
    return Decision("candidate", matched=key[0])


def publish(decision, may_disclose_route):
    """Consumer projection; information-flow security is NOT proven."""
    if may_disclose_route:
        return (decision.status, decision.matched, decision.diagnostic)
    if decision.status == "candidate":
        return ("candidate", "", "")
    return ("unavailable", "", "")


class RouteDisclosureCharacterization(unittest.TestCase):
    def setUp(self):
        self.primary = Route("primary", "primary-db", "map-v1",
                             "tenant-A", "update", "policy-v1")
        self.archive = Route("archive", "archive-db", "map-v1",
                             "tenant-A", "update", "policy-v1", disclosed=False)

    def test_same_physical_route_alias_equivalent(self):
        other = replace(self.primary, alias="secondary")
        self.assertEqual(resolve((self.primary, other), "update").status,
                         "candidate")

    def test_visible_single_route_does_not_prove_unique(self):
        self.assertEqual(sum(r.disclosed for r in (self.primary, self.archive)), 1)
        self.assertEqual(resolve((self.primary, self.archive), "update").status,
                         "ambiguous")

    def test_one_granted_route_does_not_select_a_meaning(self):
        hidden_denied = replace(self.archive, granted=False)
        self.assertEqual(resolve((self.primary, hidden_denied), "update").status,
                         "ambiguous")

    def test_alias_sort_order_does_not_choose_route(self):
        a = resolve((self.archive, self.primary), "update")
        b = resolve((replace(self.primary, alias="a"),
                     replace(self.archive, alias="z")), "update")
        self.assertEqual((a.status, a.matched), (b.status, b.matched))

    def test_explicit_target_qualifies_without_disclosing_identity(self):
        d = resolve((self.archive, self.primary), "update", "primary-db")
        self.assertEqual(d.status, "candidate")
        self.assertEqual(publish(d, may_disclose_route=False),
                         ("candidate", "", ""))

    def test_denied_target_does_not_fallback(self):
        d = resolve((replace(self.primary, granted=False), self.archive),
                    "update", "primary-db")
        self.assertEqual(d.status, "denied")
        self.assertEqual(d.matched, "")

    def test_purpose_specific_route_can_be_unique(self):
        read_route = replace(self.archive, purpose="read")
        d = resolve((self.primary, read_route), "update")
        self.assertEqual((d.status, d.matched), ("candidate", "primary-db"))

    def test_replica_read_is_not_primary_write_freshness(self):
        read = replace(self.archive, purpose="read", fresh=True)
        write = replace(self.primary, fresh=False)
        self.assertEqual(resolve((read, write), "read").status, "candidate")
        self.assertEqual(resolve((read, write), "update").status, "stale")

    def test_hidden_ambiguity_and_denial_share_generic_projection(self):
        ambiguous = resolve((self.primary, self.archive), "update")
        denied = resolve((replace(self.primary, granted=False),), "update")
        self.assertEqual(publish(ambiguous, False), publish(denied, False))

    def test_mapping_version_is_not_alias_equivalence(self):
        updated = replace(self.primary, alias="other", mapping_version="map-v2")
        self.assertEqual(resolve((self.primary, updated), "update").status,
                         "ambiguous")

    def test_scope_change_prevents_alias_equivalence(self):
        other = replace(self.primary, alias="other", scope="tenant-B")
        self.assertEqual(resolve((self.primary, other), "update").status,
                         "ambiguous")

    def test_disclosure_permission_does_not_change_internal_decision(self):
        a = resolve((self.primary, self.archive), "update")
        b = resolve((replace(self.primary, disclosed=False),
                     replace(self.archive, disclosed=True)), "update")
        self.assertEqual(a, b)


if __name__ == "__main__":
    unittest.main()
