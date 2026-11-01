#!/usr/bin/env python3
"""QX-16 fixture-only read-only claim/dependency projection; not a SQL evaluator.

All input-scope and model facts are synthetic, explicitly configured below.
Do not use this reference to prove Provider completeness or business acceptance.
"""
import json
from verify_sqlite import qx16_claim_scope_observations

# This fixture-specific dependency graph describes authored SQL operations. It is
# not parsed or inferred from a SQLGlot result and is not a new canonical model.
NODES = {
    "line_sum": {"operator": "aggregate-sum", "inputCollections": ["line"], "grain": "group:H"},
    "line_sum_gte_500": {"operator": "compare-gte", "inputs": ["line_sum"],
                         "inputCollections": ["line"], "grain": "predicate:H"},
    "exists_500": {"operator": "filter-exists", "inputCollections": ["line"], "grain": "predicate:H"},
    "not_exists_500": {"operator": "negate-exists", "inputs": ["exists_500"],
                       "inputCollections": ["line"], "grain": "predicate:H"},
    "join_sum": {"operator": "inner-join-then-sum", "inputCollections": ["line", "delivery"], "grain": "join-pair:H"},
    "left_unmatched": {"operator": "left-join-then-is-null", "inputCollections": ["header", "delivery"], "grain": "header:H"},
}

# This is explicit test metadata, NOT an observation made by SQLite. "complete"
# is an assumption about a closed, synthetic population for this scenario only.
CASES = {
    "R_partial": {"source": "R_A600_only", "lineComplete": False, "deliveryComplete": False, "nonnegative": True},
    "L1_complete": {"source": "L1_A600_B500", "lineComplete": True, "deliveryComplete": True, "nonnegative": True},
    "L2_complete": {"source": "L2_A600_B1000", "lineComplete": True, "deliveryComplete": True, "nonnegative": True},
    "line_complete_delivery_partial": {"source": "L1_one_delivery", "lineComplete": True, "deliveryComplete": False, "nonnegative": True},
    "signed_model_complete": {"source": "L_signed_model", "lineComplete": True, "deliveryComplete": True, "nonnegative": False},
    "empty_partial": {"source": "R_empty", "lineComplete": False, "deliveryComplete": False, "nonnegative": True},
    "empty_complete": {"source": "R_empty", "lineComplete": True, "deliveryComplete": True, "nonnegative": True},
}


def _claim(status, observed, *, conclusion=None, conditions=()):
    return {"status": status, "observedSliceValue": observed,
            "conclusion": conclusion, "conditions": list(conditions)}


def evaluate_case(config, observed):
    """Select eligible explanatory claims; never determine source completeness."""
    line_complete = config["lineComplete"]
    delivery_complete = config["deliveryComplete"]
    total = observed["line_sum"]
    if line_complete:
        line_sum = _claim("exact-synthetic-scope", total, conclusion=total,
                          conditions=("closed-line-population",))
    elif config["nonnegative"] and total is not None:
        line_sum = _claim("conditional-lower-bound", total, conclusion={"gte": total},
                          conditions=("observed-IDs-subset-of-true-population", "all-unobserved-values-nonnegative",
                                      "same-units-and-revision", "no-join-amplification"))
    else:
        line_sum = _claim("unresolved-partial", total)

    if line_complete and total is not None:
        threshold = _claim("exact-synthetic-scope", total, conclusion=(total >= 500),
                           conditions=("closed-line-population",))
    elif not line_complete and line_sum["status"] == "conditional-lower-bound" and total >= 500:
        threshold = _claim("conditional-true", total, conclusion=True,
                           conditions=tuple(line_sum["conditions"]))
    else:
        threshold = _claim("unresolved-partial-or-null", total)

    # A matching witness in an observed subset supports EXISTS; a missing one
    # cannot support non-existence without a closed population.
    if observed["exists_500"] == 1:
        exists = _claim("witness-true", True, conclusion=True,
                        conditions=("same-query-predicate", "subset-of-true-population", "same-revision"))
        not_exists = _claim("witness-false", False, conclusion=False,
                            conditions=tuple(exists["conditions"]))
    elif line_complete:
        exists = _claim("exact-synthetic-scope", False, conclusion=False,
                        conditions=("closed-line-population",))
        not_exists = _claim("exact-synthetic-scope", True, conclusion=True,
                            conditions=("closed-line-population",))
    else:
        exists = _claim("unresolved-partial", False)
        not_exists = _claim("unresolved-partial", True)

    if line_complete and delivery_complete:
        join_sum = _claim("exact-synthetic-scope", observed["join_sum"],
                          conclusion=observed["join_sum"],
                          conditions=("closed-line-population", "closed-delivery-population",
                                      "SQL-join-pair-grain-not-business-line-total"))
    else:
        join_sum = _claim("observed-slice-only", observed["join_sum"],
                          conditions=("join-depends-on-both-collections",))

    return {
        "inputContext": {
            "line": {"grain": "entity:line", "completeness": "complete" if line_complete else "partial",
                     "population": "synthetic:H/line"},
            "delivery": {"grain": "entity:delivery", "completeness": "complete" if delivery_complete else "partial",
                         "population": "synthetic:H/delivery"},
            "nonnegativeModelAssumedForFixture": config["nonnegative"],
            "realProviderGuarantee": "not-verified", "businessAcceptance": "not-accepted",
        },
        "claims": {"line_sum": line_sum, "line_sum_gte_500": threshold,
                   "exists_500": exists, "not_exists_500": not_exists, "join_sum": join_sum},
    }


def explain_left_join(observed, *, right_complete):
    """An observed unmatched row is only an exact absence claim if right is closed."""
    is_unmatched = bool(observed["anti_rows"])
    if right_complete:
        return _claim("exact-synthetic-scope", observed["left_rows"],
                      conclusion={"unmatched": is_unmatched},
                      conditions=("closed-right-collection", "stable-join-predicate"))
    return _claim("observed-slice-only", observed["left_rows"],
                  conditions=("right-collection-partial", "unmatched-output-may-disappear"))


def build_reference():
    observations = qx16_claim_scope_observations()  # SQL was run independently.
    scenarios = {key: evaluate_case(config, observations[config["source"]])
                 for key, config in CASES.items()}
    scenarios["left_partial_no_match"] = {
        "claims": {"left_unmatched": explain_left_join(observations["left_right_empty"], right_complete=False)}}
    scenarios["left_complete_no_match"] = {
        "claims": {"left_unmatched": explain_left_join(observations["left_right_empty"], right_complete=True)}}
    scenarios["left_complete_D1"] = {
        "claims": {"left_unmatched": explain_left_join(observations["left_right_D1"], right_complete=True)}}
    return {"kind": "fixture-only-read-only-query-explanation", "fixture": "QX-16",
            "syntheticSqliteActual": observations, "operationDependencies": NODES,
            "scenarios": scenarios,
            "notProven": ["TEI DerivedIR or real SQLGlot worker", "Provider completeness or read snapshot",
                          "SQL query result to accepted Semantic identity", "Owner-approved operator law",
                          "current business value or Host permission", "generic SQL claim inference"]}


if __name__ == "__main__":
    print(json.dumps(build_reference(), ensure_ascii=False, indent=2, sort_keys=True))
