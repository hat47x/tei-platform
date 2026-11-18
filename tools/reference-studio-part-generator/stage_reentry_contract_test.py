#!/usr/bin/env python3
"""Independent expectations for Stage2's generated Reader observations.

Do not derive these expectations from stage_reentry.self_test(): a merge that
changes both the judgement and its own examples must still fail this test.
This tests the observation contract, not a Windows/WebView2 process.
"""
import unittest

import stage_reentry


CONTRACT = "sha256:" + "c" * 64


def successful_step(invoker: str, artifact: str, invoked: str = "child") -> dict:
    return {
        "invokerDigest": invoker, "invokedExecutableDigest": invoked,
        "invokedExitCode": 0, "artifactDigest": artifact,
        "artifactCrossCheckMatched": True, "failureCause": None,
        "artifactOnDisk": artifact,
    }


def stage(invoker: str) -> dict:
    return {
        "invokerOnDisk": invoker,
        "teiBuild": successful_step(invoker, "tei1"),
        "generatorBuild": successful_step(invoker, "gen"),
        "partGeneration": successful_step(invoker, "part", "gen"),
        "studioDesktopBuild": successful_step(invoker, "studio"),
    }


def desktop_observation(**changes) -> dict:
    smoke = {
        "exitCode": 0, "exeDigest": "studio",
        "observedContract": CONTRACT, "initialObservedContract": CONTRACT,
        "readerCallsAfterInitialInspect": 1, "readerCalls": 3,
    }
    smoke.update(changes)
    return {
        "embedded": {
            "desktopDigest": "studio", "embedsCommittedBundle": True,
            "regeneratedPartContract": CONTRACT, "bundleContract": CONTRACT,
        },
        "smoke": smoke,
    }


def observed_verdict(**changes) -> bool:
    result = stage_reentry.judge(
        stage("tei0"), stage("tei1"), "part", desktop_observation(**changes)
    )
    return result["stage2DesktopUsedGeneratedReaderInNewProcess"]


class Stage2ReaderTemporalContractTest(unittest.TestCase):
    def test_initial_inspect_then_graph_reuses_same_generated_reader(self):
        self.assertTrue(observed_verdict())

    def test_total_calls_are_not_enough_without_increase(self):
        self.assertFalse(observed_verdict(readerCallsAfterInitialInspect=2, readerCalls=2))

    def test_first_and_last_reader_identity_must_match_generated_contract(self):
        self.assertFalse(observed_verdict(initialObservedContract="sha256:" + "d" * 64))
        self.assertFalse(observed_verdict(observedContract="sha256:" + "d" * 64))

    def test_missing_initial_observation_is_not_success(self):
        self.assertFalse(observed_verdict(initialObservedContract=None))
        self.assertFalse(observed_verdict(readerCallsAfterInitialInspect=None))

    def test_boolean_or_invalid_counts_are_not_integer_calls(self):
        self.assertFalse(observed_verdict(readerCalls=True))
        self.assertFalse(observed_verdict(readerCallsAfterInitialInspect=True))
        self.assertFalse(observed_verdict(readerCallsAfterInitialInspect=0))
        self.assertFalse(observed_verdict(readerCalls=1))

    def test_wrong_or_missing_executable_cannot_qualify_the_reader(self):
        self.assertFalse(observed_verdict(exeDigest="different-stage2-binary"))
        self.assertFalse(observed_verdict(exeDigest=None))

    def test_observation_reader_uses_both_distinct_page_snapshots(self):
        facts = {
            "exe_sha256": "a" * 64,
            "page_state": {
                "inspectionReader": CONTRACT,
                "inspectionReaderCalls": 1,
                "graphSelection": {
                    "inspectionReader": CONTRACT,
                    "inspectionReaderCalls": 3,
                },
            },
        }
        observed = stage_reentry.smoke_observations(facts)
        self.assertEqual(observed["initialObservedContract"], CONTRACT)
        self.assertEqual(observed["observedContract"], CONTRACT)
        self.assertEqual(observed["readerCallsAfterInitialInspect"], 1)
        self.assertEqual(observed["readerCalls"], 3)
        self.assertTrue(observed_verdict(**{
            key: observed[key] for key in (
                "initialObservedContract", "observedContract",
                "readerCallsAfterInitialInspect", "readerCalls",
            )
        }))


if __name__ == "__main__":
    unittest.main()
