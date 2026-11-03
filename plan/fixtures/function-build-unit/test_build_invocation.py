#!/usr/bin/env python3
"""Synthetic topology/reference only. Not the TEI Model, Router, Generator or Host."""
from dataclasses import dataclass
from collections import Counter
import unittest


@dataclass(frozen=True)
class Function:
    id: str
    semantic_revision: str
    input_shape: str
    dependencies: tuple[str, ...] = ()


@dataclass(frozen=True)
class Build:
    id: str
    target: str = "go-executable"
    linked_resources: tuple[str, ...] = ()


@dataclass(frozen=True)
class Placement:
    function: str
    build: str
    exported: bool = False


@dataclass(frozen=True)
class Invocation:
    id: str
    trigger: str
    address: str
    function: str
    build: str
    adapter: str
    input_shape: str
    public: bool = False


@dataclass(frozen=True)
class Topology:
    functions: tuple[Function, ...]
    builds: tuple[Build, ...]
    placements: tuple[Placement, ...] = ()
    invocations: tuple[Invocation, ...] = ()
    excluded: tuple[str, ...] = ()
    cross_build_calls: tuple[tuple[str, str], ...] = ()


def effective_placements(topology):
    """One-build zero-config derives *internal* placement, never public export."""
    if len(topology.builds) == 1 and not topology.placements:
        return tuple(Placement(f.id, topology.builds[0].id)
                     for f in topology.functions if f.id not in topology.excluded)
    return topology.placements


def diagnose(topology):
    """Explain only declarative contradictions; deliberately no inferred grant."""
    errors = set()
    funcs = {f.id: f for f in topology.functions}
    builds = {b.id: b for b in topology.builds}
    if not builds:
        errors.add("no_build")
    if len(funcs) != len(topology.functions):
        errors.add("duplicate_function")
    if len(builds) != len(topology.builds):
        errors.add("duplicate_build")
    if len(topology.excluded) != len(set(topology.excluded)):
        errors.add("duplicate_exclusion")
    for f_id in topology.excluded:
        if f_id not in funcs:
            errors.add("unknown_exclusion")
    places = effective_placements(topology)
    pairs = [(p.function, p.build) for p in places]
    if len(pairs) != len(set(pairs)):
        errors.add("duplicate_placement")
    for p in places:
        if p.function not in funcs:
            errors.add("unknown_placement_function")
        if p.build not in builds:
            errors.add("unknown_placement_build")
        if p.function in topology.excluded:
            errors.add("excluded_but_placed")
    mapped = {p.function for p in places}
    if set(funcs) - mapped - set(topology.excluded):
        errors.add("function_unassigned")
    for f in topology.functions:
        for dependency in f.dependencies:
            if dependency not in funcs:
                errors.add("unknown_function_dependency")
                continue
            for p in places:
                if p.function != f.id:
                    continue
                if (dependency, p.build) not in pairs and (
                    f.id, dependency
                ) not in topology.cross_build_calls:
                    errors.add("dependency_not_placed_or_bound")
    if len(topology.cross_build_calls) != len(set(topology.cross_build_calls)):
        errors.add("duplicate_cross_build_call")
    for caller, callee in topology.cross_build_calls:
        if caller not in funcs or callee not in funcs:
            errors.add("unknown_cross_build_contract")
        elif callee not in mapped or caller not in mapped:
            errors.add("unplaced_cross_build_contract")

    public_bindings = set()
    invocation_ids = set()
    for inv in topology.invocations:
        if inv.id in invocation_ids:
            errors.add("duplicate_invocation_id")
        invocation_ids.add(inv.id)
        key = (inv.build, inv.trigger, inv.address)
        if key in public_bindings:
            errors.add("invocation_address_collision")
        public_bindings.add(key)
        if inv.trigger not in {"ui", "cli", "http", "schedule", "event", "eki"}:
            errors.add("unknown_trigger")
        if not inv.address or not inv.adapter:
            errors.add("missing_invocation_adapter_or_address")
        if inv.function not in funcs:
            errors.add("unknown_invoked_function")
        elif inv.input_shape != funcs[inv.function].input_shape:
            errors.add("invocation_shape_mismatch")
        if inv.build not in builds:
            errors.add("unknown_invocation_build")
        if inv.function in topology.excluded:
            errors.add("invokes_excluded_function")
        found = [p for p in places if p.function == inv.function
                 and p.build == inv.build]
        if not found:
            errors.add("invocation_without_placement")
        elif inv.public and not any(p.exported for p in found):
            errors.add("public_entry_without_export")

    # A browser bundle may not contain server-only implementation or secrets.
    # This static fixture cannot prove physical linked-byte exclusions.
    for b in topology.builds:
        if b.target == "browser-assets" and (
            {"server-secret", "server-db-driver"} & set(b.linked_resources)
        ):
            errors.add("server_dependency_in_browser_assets")

    return tuple(sorted(errors))


def offer(topology, route_id):
    """A sound static route is only a candidate, never an execution grant."""
    found = [i for i in topology.invocations if i.id == route_id]
    if len(found) != 1 or diagnose(topology):
        return "unresolved"
    return "declarative_candidate_not_execution_authority"


def layout(topology):
    """Business function identity and revision do not depend on build names."""
    funcs = {f.id: f.semantic_revision for f in topology.functions}
    return funcs, tuple(sorted((p.function, p.build) for p in
                               effective_placements(topology)))


F1 = Function("order.submit", "meaning-r3", "order-request/v1")
F2 = Function("order.report", "meaning-r5", "report-request/v2")
SINGLE = Topology((F1, F2), (Build("main"),))
MULTI = Topology((F1, F2), (Build("web"), Build("jobs")),
                 (Placement("order.submit", "web", True),
                  Placement("order.report", "jobs", True)))
WEB_ROUTE = Invocation("submit-ui", "ui", "button:submit",
                       "order.submit", "web", "ui-adapter",
                       "order-request/v1", True)
CRON_ROUTE = Invocation("report-cron", "schedule", "daily:reports",
                        "order.report", "jobs", "system-cron",
                        "report-request/v2", True)


class BuildInvocationBoundaryTests(unittest.TestCase):
    def test_bi01_single_build_requires_no_mapping(self):
        self.assertEqual(diagnose(SINGLE), ())
        self.assertEqual(len(effective_placements(SINGLE)), 2)

    def test_bi02_single_implicit_map_has_no_public_export(self):
        self.assertTrue(all(not p.exported for p in effective_placements(SINGLE)))

    def test_bi03_explicit_two_build_map(self):
        self.assertEqual(diagnose(MULTI), ())

    def test_bi04_multi_without_map_is_not_implicit_all(self):
        self.assertIn("function_unassigned", diagnose(
            Topology((F1, F2), (Build("web"), Build("jobs")))))

    def test_bi05_partial_mapping_needs_exclusion_or_other_placement(self):
        t = Topology((F1, F2), MULTI.builds, (MULTI.placements[0],))
        self.assertIn("function_unassigned", diagnose(t))

    def test_bi06_explicit_exclusion_is_not_silently_lost(self):
        t = Topology((F1, F2), MULTI.builds, (MULTI.placements[0],),
                     excluded=("order.report",))
        self.assertEqual(diagnose(t), ())

    def test_bi07_excluded_function_cannot_be_placed(self):
        t = Topology(MULTI.functions, MULTI.builds, MULTI.placements,
                     excluded=("order.report",))
        self.assertIn("excluded_but_placed", diagnose(t))

    def test_bi08_same_function_can_be_in_two_builds(self):
        t = Topology((F1,), MULTI.builds,
                     (Placement(F1.id, "web"), Placement(F1.id, "jobs")))
        self.assertEqual(diagnose(t), ())

    def test_bi09_duplicate_placement_not_silently_deduplicated(self):
        t = Topology((F1,), (Build("main"),),
                     (Placement(F1.id, "main"), Placement(F1.id, "main")))
        self.assertIn("duplicate_placement", diagnose(t))

    def test_bi10_unknown_build_refused(self):
        t = Topology((F1,), (Build("main"),),
                     (Placement(F1.id, "unknown"),))
        self.assertIn("unknown_placement_build", diagnose(t))

    def test_bi11_ui_button_is_binding_not_new_business_function(self):
        t = Topology(MULTI.functions, MULTI.builds, MULTI.placements, (WEB_ROUTE,))
        self.assertEqual(diagnose(t), ())
        self.assertEqual(len(t.functions), len(MULTI.functions))
        self.assertEqual(offer(t, "submit-ui"),
                         "declarative_candidate_not_execution_authority")

    def test_bi12_schedule_uses_same_function_contract(self):
        t = Topology(MULTI.functions, MULTI.builds, MULTI.placements, (CRON_ROUTE,))
        self.assertEqual(diagnose(t), ())

    def test_bi13_api_and_cli_can_share_one_function_definition(self):
        http = Invocation("http", "http", "POST:/orders", F1.id, "web",
                          "http-adapter", F1.input_shape, True)
        cli = Invocation("cli", "cli", "orders submit", F1.id, "web",
                         "cli-adapter", F1.input_shape, True)
        t = Topology(MULTI.functions, MULTI.builds, MULTI.placements,
                     (http, cli))
        self.assertEqual(diagnose(t), ())

    def test_bi14_unmapped_public_route_refused(self):
        inv = Invocation("bad", "http", "/submit", F2.id, "web",
                         "http-adapter", F2.input_shape, True)
        t = Topology(MULTI.functions, MULTI.builds, MULTI.placements, (inv,))
        self.assertIn("invocation_without_placement", diagnose(t))

    def test_bi15_implicit_single_build_does_not_expose_public_route(self):
        inv = Invocation("bad", "http", "/submit", F1.id, "main",
                         "http-adapter", F1.input_shape, True)
        self.assertIn("public_entry_without_export", diagnose(
            Topology(SINGLE.functions, SINGLE.builds, invocations=(inv,))))

    def test_bi16_wrong_input_shape_refused(self):
        inv = Invocation("bad", "http", "/submit", F1.id, "web",
                         "http-adapter", "different-input-shape", True)
        t = Topology(MULTI.functions, MULTI.builds, MULTI.placements, (inv,))
        self.assertIn("invocation_shape_mismatch", diagnose(t))

    def test_bi17_route_collision_detected_even_with_distinct_ids(self):
        second = Invocation("other-ui", "ui", "button:submit", F1.id,
                            "web", "ui-adapter", F1.input_shape, True)
        t = Topology(MULTI.functions, MULTI.builds, MULTI.placements,
                     (WEB_ROUTE, second))
        self.assertIn("invocation_address_collision", diagnose(t))

    def test_bi18_no_executable_adapter_is_not_a_valid_route(self):
        inv = Invocation("bad", "ui", "button:x", F1.id, "web", "",
                         F1.input_shape, True)
        t = Topology(MULTI.functions, MULTI.builds, MULTI.placements, (inv,))
        self.assertIn("missing_invocation_adapter_or_address", diagnose(t))

    def test_bi19_browser_bundle_must_not_claim_server_secret_safe(self):
        t = Topology((F1,), (Build("browser", "browser-assets",
                                 ("server-secret",)),))
        self.assertIn("server_dependency_in_browser_assets", diagnose(t))

    def test_bi20_same_meaning_repackaged_single_to_multi(self):
        self.assertEqual(layout(SINGLE)[0], layout(MULTI)[0])
        self.assertNotEqual(layout(SINGLE)[1], layout(MULTI)[1])

    def test_bi21_semantic_revision_change_requires_qualification(self):
        changed = Function(F1.id, "meaning-r4", F1.input_shape)
        t = Topology((changed, F2), MULTI.builds, MULTI.placements)
        self.assertNotEqual(layout(t)[0], layout(MULTI)[0])

    def test_bi22_build_ids_are_not_accepted_semantic_subject_ids(self):
        self.assertNotIn("web", layout(MULTI)[0])
        self.assertIn("order.submit", layout(MULTI)[0])

    def test_bi23_cross_build_dependency_requires_explicit_contract(self):
        called = Function(F1.id, F1.semantic_revision, F1.input_shape,
                          dependencies=(F2.id,))
        t = Topology((called, F2), MULTI.builds, MULTI.placements)
        self.assertIn("dependency_not_placed_or_bound", diagnose(t))

    def test_bi24_cross_build_call_needs_adapter_at_runtime_not_grant(self):
        called = Function(F1.id, F1.semantic_revision, F1.input_shape,
                          dependencies=(F2.id,))
        t = Topology((called, F2), MULTI.builds, MULTI.placements,
                     (WEB_ROUTE,), cross_build_calls=((F1.id, F2.id),))
        self.assertEqual(diagnose(t), ())
        self.assertEqual(offer(t, WEB_ROUTE.id),
                         "declarative_candidate_not_execution_authority")

    def test_bi25_eki_is_optional_trigger_not_required_host(self):
        eki = Invocation("job", "eki", "report-job", F2.id, "jobs",
                         "eki-adapter", F2.input_shape)
        t = Topology(MULTI.functions, MULTI.builds, MULTI.placements, (eki,))
        self.assertEqual(diagnose(t), ())

    def test_bi26_both_business_and_job_effect_remain_unproven(self):
        t = Topology(MULTI.functions, MULTI.builds, MULTI.placements,
                     (WEB_ROUTE, CRON_ROUTE))
        self.assertEqual(diagnose(t), ())
        self.assertEqual(offer(t, WEB_ROUTE.id),
                         "declarative_candidate_not_execution_authority")
        self.assertNotIn("commit", offer(t, CRON_ROUTE.id))

    def test_bi27_both_cli_and_ui_need_real_transport_binding(self):
        wrong = Invocation("bad", "ui", "button:submit", F1.id, "web",
                           "", F1.input_shape, True)
        t = Topology(MULTI.functions, MULTI.builds, MULTI.placements, (wrong,))
        self.assertEqual(offer(t, wrong.id), "unresolved")

    def test_bi28_duplicate_build_names_do_not_increase_capacity(self):
        t = Topology((F1,), (Build("main"), Build("main")))
        self.assertIn("duplicate_build", diagnose(t))


if __name__ == "__main__":
    unittest.main(verbosity=2)
