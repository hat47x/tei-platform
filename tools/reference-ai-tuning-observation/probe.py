#!/usr/bin/env python3
"""Bounded, read-only AI tuning observation join. Synthetic test/reference only.

Not a TEI Source reader, product API, optimizer, statistics collector, host,
authority evaluator, or proof of actual Provider invocation.
"""
from __future__ import annotations

import json
import re
import sys
from typing import Any

_HEX256 = re.compile(r"^[0-9a-f]{64}$")


def _unresolved(reason: str) -> dict[str, Any]:
    return {"status": "unresolved", "reason": reason}


def evaluate(packet: Any) -> dict[str, Any]:
    """Validate all joins before emitting details; never echo input strings."""
    if not isinstance(packet, dict):
        return _unresolved("invalid-packet")
    consumer = packet.get("consumer")
    if not isinstance(consumer, dict) or consumer.get("mayDisclose") is not True:
        return {"status": "withheld"}

    design = packet.get("design")
    delivery = packet.get("delivery")
    observations = packet.get("observations")
    statistics = packet.get("statistics")
    if not all(isinstance(v, dict) for v in (design, delivery, statistics)):
        return _unresolved("missing-bound-input")
    if not isinstance(observations, list) or not observations:
        return _unresolved("missing-observation")

    required = ("targetRef", "queryRef", "revision", "scope", "selectedProvider")
    if any(not isinstance(design.get(k), str) or not design[k] for k in required):
        return _unresolved("missing-design-binding")
    if type(design.get("worldGeneration")) is not int or design["worldGeneration"] < 1:
        return _unresolved("missing-world-generation")
    if not isinstance(consumer.get("grantEpoch"), str) or not consumer["grantEpoch"]:
        return _unresolved("missing-current-grant")

    expected = delivery.get("expectedParts")
    parts = delivery.get("parts")
    if type(expected) is not int or expected < 1 or not isinstance(parts, list):
        return _unresolved("invalid-delivery")
    if delivery.get("endMarker") is not True or len(parts) != expected:
        return _unresolved("incomplete-delivery")
    seen: set[int] = set()
    for part in parts:
        if not isinstance(part, dict):
            return _unresolved("invalid-delivery")
        index = part.get("index")
        digest = part.get("sha256")
        if (type(index) is not int or index < 1 or index > expected or
                not isinstance(digest, str) or not _HEX256.fullmatch(digest) or
                index in seen):
            return _unresolved("invalid-delivery")
        seen.add(index)
    if seen != set(range(1, expected + 1)):
        return _unresolved("incomplete-delivery")

    window = statistics.get("windowId")
    if not isinstance(window, str) or not window:
        return _unresolved("missing-statistics-window")
    if (statistics.get("revision") != design["revision"] or
            statistics.get("worldGeneration") != design["worldGeneration"]):
        return _unresolved("stale-statistics")
    total = statistics.get("totalCalls")
    sampled = statistics.get("sampledCalls")
    dropped = statistics.get("droppedCalls")
    if (type(total) is not int or type(sampled) is not int or
            type(dropped) is not int or
            total < 0 or sampled < 0 or dropped < 0 or
            sampled + dropped > total):
        return _unresolved("invalid-sampling")

    repeat = False
    for item in observations:
        if not isinstance(item, dict):
            return _unresolved("invalid-observation")
        if any(not isinstance(item.get(k), str) or not item[k]
               for k in ("attemptId", "queryRef", "targetRef",
                         "selectedProvider", "invokedProvider", "scope",
                         "revision", "grantEpoch", "windowId")):
            return _unresolved("missing-call-binding")
        if (item["targetRef"] != design["targetRef"] or
                item["queryRef"] != design["queryRef"] or
                item["scope"] != design["scope"] or
                item["revision"] != design["revision"] or
                item["windowId"] != window or
                item["grantEpoch"] != consumer["grantEpoch"] or
                item["selectedProvider"] != design["selectedProvider"] or
                item["invokedProvider"] != design["selectedProvider"] or
                item.get("worldGeneration") != design["worldGeneration"]):
            return _unresolved("incompatible-call-binding")
        count = item.get("queryReadCalls")
        if type(count) is not int or count < 0:
            return _unresolved("invalid-call-count")
        if item.get("kind") != "db-read":
            return _unresolved("unsupported-call-kind")
        if count >= 3:
            repeat = True

    coverage = ("window-sampled" if sampled != total or dropped != 0
                else "reported-window-complete")
    return {
        "status": "input-consistent",
        "qualification": "reference-only-not-actual-invocation",
        "coverage": coverage,
        "candidate": "investigate-repeated-reads" if repeat else "none-observed",
        "candidateIsVerified": False,
        "optimizationApplied": False,
        "performanceImproved": False,
        "sampledCalls": sampled,
        "totalCalls": total,
    }


def _unique_members(pairs: list[tuple[str, Any]]) -> dict[str, Any]:
    result: dict[str, Any] = {}
    for name, value in pairs:
        if name in result:
            raise ValueError("duplicate JSON member")
        result[name] = value
    return result


def _no_nonfinite(_: str) -> None:
    raise ValueError("non-finite JSON")


def main() -> int:
    try:
        packet = json.load(sys.stdin, object_pairs_hook=_unique_members,
                           parse_constant=_no_nonfinite)
        result = evaluate(packet)
    except (ValueError, TypeError, UnicodeError, RecursionError):
        result = _unresolved("invalid-input")
    json.dump(result, sys.stdout, ensure_ascii=False, sort_keys=True)
    sys.stdout.write("\n")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
