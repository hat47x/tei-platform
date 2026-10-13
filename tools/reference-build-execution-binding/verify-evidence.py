"""既存の合成参照例を検査する。真正性・構築回数・実行権限の検証器ではない。"""
import json
import sys


def require(condition, field):
    if not condition:
        raise ValueError("reference evidence differs: " + field)


def verify(evidence):
    require(evidence["schema"] == "tei.reference.build-execution-binding/v0", "schema")
    bindings = evidence["sharedPluginBindings"]
    require(len(bindings) == 2 and len(evidence["sharedPluginBindingFingerprints"]) == 2,
            "shared bindings")
    require(len(set(evidence["sharedPluginBindingFingerprints"])) == 2, "distinct bindings")
    for binding in bindings:
        require(binding["executionPlanFingerprint"] == evidence["sharedResult"]["executionPlanFingerprint"], "shared plan")
        require(binding["artifactSetFingerprint"] == evidence["sharedResult"]["artifactSetFingerprint"], "shared set")
        require(binding["buildExecutionResultFingerprint"] == evidence["sharedResultFingerprint"], "shared result")
    comparison = evidence["multiOutputComparison"]
    require(comparison["reproducible"] is False, "comparison verdict")
    require(comparison["missingFromSecond"] and comparison["extraInSecond"] and comparison["changed"], "comparison details")
    engine = evidence["runtimeEngineBinding"]
    require(engine["engine"]["id"] and evidence["runtimeEngineBindingFingerprint"], "runtime engine")
    require("plugin" not in engine, "engine is not a plugin")
    require(evidence["buildGraphFingerprint"] and evidence["buildGraphOrder"] == ["stage1", "stage2"], "build graph")
    require(len(evidence["buildGraph"]["seeds"]) == 1 and len(evidence["buildGraph"]["edges"]) == 1, "bootstrap members")
    seed = evidence["buildGraph"]["seeds"][0]
    require(seed["trustBoundary"] == "external-stage0-seed", "stage0 boundary")
    require(seed["artifactDigest"].lower() == evidence["bootstrapStage1Plan"]["builder"]["artifactDigest"].lower(), "seed builder")
    edge = evidence["buildGraph"]["edges"][0]
    require((edge["fromNode"], edge["toNode"], edge["outputRole"]) == ("stage1", "stage2", "builder-artifact"), "bootstrap edge")
    require(evidence["bootstrapStage1BuilderBindingFingerprint"], "builder binding fingerprint")
    require(evidence["bootstrapStage1BuilderBinding"]["builder"] == evidence["bootstrapStage2Plan"]["builder"], "builder handoff")
    current, revoked = evidence["currentTrustDecision"], evidence["revokedTrustDecision"]
    require(current["trusted"] is True and revoked["trusted"] is False, "trust decision")
    require(revoked["reason"] == "issuer-revoked", "revocation reason")
    require(current["attestationFingerprint"] == revoked["attestationFingerprint"] == evidence["buildAttestationFingerprint"], "immutable attestation")
    require(current["policyFingerprint"] != revoked["policyFingerprint"], "policy revision")
    require(evidence["buildAttestationVerification"]["attestationFingerprint"] == evidence["buildAttestationFingerprint"], "verification subject")
    required_limits = {
        "build-binding-is-not-acceptance-authority", "build-binding-is-not-runtime-authority",
        "artifact-set-location-is-not-content-identity", "stage0-boundary-is-not-trust-acceptance",
        "reference-fake-signature-is-not-cryptographic-security", "attestation-trust-is-not-runtime-authority",
    }
    require(required_limits <= set(evidence["authorityLimits"]), "authority limits")
    require(not {"accepted", "runtime-authorized"} & set(evidence["authorityLimits"]), "forbidden authority")


if __name__ == "__main__":
    try:
        verify(json.load(sys.stdin))
    except (KeyError, ValueError, TypeError, AttributeError, IndexError) as error:
        raise SystemExit(str(error))
    print("reference binding/graph/revocation/authority-limits=PASS")
