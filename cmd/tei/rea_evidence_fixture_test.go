package main

// syntheticEvidence is a minimal record that carries every field the real tool
// requires of an Evidence. It is written by the test, not captured from a run.
//
// It was lost from main: commit 7b0ab6b63 added the two uses in
// cli_discovery_modes_test.go and cli_discovery_write_flags_test.go, but the
// file that defined it (rea_evidence_test.go) was never merged, so cmd/tei's
// test package stopped building. The definition is restored here from
// 895f3f468, the commit that introduced it.
const syntheticEvidence = `{
  "evidence_id": "ev_synthetic_intake_fixture",
  "subject": {"name": "synthetic", "digest": {"sha256": "000d9d6f7f73531725d781c5742c685611c36a74d6f2c29b73c36d5c61c53b54"}, "local_path": "/synthetic/target", "format": "directory"},
  "provider": {"id": "rea-javascript-application", "name": "REA JavaScript application analyzer", "version": "1"},
  "analysis_profile": null,
  "predicate_type": "rea.javascript-application-analysis",
  "operation": "analyze_javascript_application",
  "parameters": {"input_path": "/synthetic/target"},
  "raw_result": null,
  "normalized_result": {"summary": {"modules": 1}},
  "confidence": "derived",
  "authority": "shipped-artifact",
  "environment": null,
  "limitations": ["synthetic fixture: no real target was analyzed"],
  "relationship_coverage": null,
  "locations": [{"kind": "artifact-path", "path": "artifact-root"}],
  "evidence_links": []
}`

// syntheticView is one delivery in the framing the recorded tool emits: two rows
// plus the terminal metadata record that names the parent. Restored with
// syntheticEvidence from 895f3f468 for the same reason.
const syntheticView = `{"observation_id":"obs_one","kind":"node","node_id":"n1"}
{"observation_id":"obs_two","kind":"node","node_id":"n2"}
{"$rea":"jsonl/v1","type":"metadata","records":{"path":["normalized_result","rows"],"emitted":2,"source":2},"context":{"evidence_id":"ev_synthetic_view","normalized_result":{"parent_evidence_id":"ev_synthetic_intake_fixture","coverage":{"status":"page","examined":2,"total":2,"exhausted":true},"limitations":["synthetic fixture"],"unknowns":[]}}}
`
