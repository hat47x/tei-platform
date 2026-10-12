#!/usr/bin/env bash
# Qualify the propagation of a real transform Engine source mutation through
# a reproducible executable build, TEI-native Build Plan, and Evidence v2.
set -euo pipefail

repo="$(git rev-parse --show-toplevel)"
baseline_revision="$(git -C "$repo" rev-parse HEAD)"
if [[ -n "$(git -C "$repo" status --porcelain=v1 --untracked-files=all)" ]]; then
  echo "baseline repository must be clean" >&2
  exit 1
fi
if [[ $# -ne 1 || "$1" != /* ]]; then
  echo "usage: $0 /absolute/output-directory-outside-repository" >&2
  exit 2
fi
output="$1"
case "$output/" in
  "$repo/"*) echo "output directory must be outside repository" >&2; exit 2 ;;
esac

mkdir -p "$output/baseline" "$output/mutation" "$output/host-only"
mutation_repo="$output/worktree"
host_repo="$output/host-worktree"
cleanup() {
  git -C "$repo" worktree remove --force "$mutation_repo" >/dev/null 2>&1 || true
  git -C "$repo" worktree remove --force "$host_repo" >/dev/null 2>&1 || true
}
trap cleanup EXIT
git -C "$repo" worktree add --quiet --detach "$mutation_repo" "$baseline_revision"
git -C "$repo" worktree add --quiet --detach "$host_repo" "$baseline_revision"

target="$mutation_repo/internal/plugin/transform/source_budget.go"
# Change one reachable error string (not a comment or dead code). This is a
# deterministic semantic change to the compiled Engine with no effect on the
# valid Reference Sources used by the self-build.
needle='serialized size limit exceeded'
replacement='serialized size limit exceeded (qualification mutation)'
grep -qF "$needle" "$target"
sed -i "0,/$needle/s//$replacement/" "$target"
grep -qF "$replacement" "$target"
git -C "$mutation_repo" add internal/plugin/transform/source_budget.go
git -C "$mutation_repo" -c user.name='TEI Qualification' \
  -c user.email='tei-qualification@localhost' \
  commit --quiet -m "test: alter reachable transform resource-bound error"
mutation_revision="$(git -C "$mutation_repo" rev-parse HEAD)"
test "$mutation_revision" != "$baseline_revision"

# A real (reachable) Host lease-status message changes without touching the
# transform Engine's source closure, toolchain, target, or artifact bytes.
host_target="$host_repo/internal/plugin/runtime/lease.go"
host_needle='lease acquired'
host_replacement='lease acquired (Host-only qualification mutation)'
grep -qF "$host_needle" "$host_target"
sed -i "0,/$host_needle/s//$host_replacement/" "$host_target"
grep -qF "$host_replacement" "$host_target"
git -C "$host_repo" add internal/plugin/runtime/lease.go
git -C "$host_repo" -c user.name='TEI Qualification' \
  -c user.email='tei-qualification@localhost' \
  commit --quiet -m "test: change Host lease behavior outside transform Engine"
host_revision="$(git -C "$host_repo" rev-parse HEAD)"
test "$host_revision" != "$baseline_revision"
test "$host_revision" != "$mutation_revision"

run_pair() {
  local root="$1"
  local directory="$2"
  local revision="$3"
  (
    cd "$root"
    test "$(git rev-parse HEAD)" = "$revision"
    test -z "$(git status --porcelain=v1 --untracked-files=all)"
    go run ./tools/reference-transform-runtime-engine-build \
      -repo . -artifact-out "$directory/transform-engine" \
      > "$directory/engine-build-evidence.json"
    # The verified binary still needs independent Reference Host acceptance.
    # These records are qualification fixtures, not production authorization.
    jq -n --slurpfile engine "$directory/engine-build-evidence.json" '{
      formatVersion: 1,
      stage: "stage2",
      runtimeEngine: $engine[0].runtimeEngine
    }' > "$directory/stage2-engine-lock.json"
    jq -n --slurpfile engine "$directory/engine-build-evidence.json" '{
      formatVersion: 1,
      stage: "stage2",
      hostId: "tei.reference.host.actions",
      runtimeEngine: $engine[0].runtimeEngine
    }' > "$directory/stage2-host-registration.json"
    go run ./tools/reference-plugin-self-build -repo . \
      -exact-runtime-engine "$directory/transform-engine" \
      -engine-build-evidence "$directory/engine-build-evidence.json" \
      -stage2-engine-lock "$directory/stage2-engine-lock.json" \
      -stage2-host-registration "$directory/stage2-host-registration.json" \
      > "$directory/self-build-evidence.json"
    test "$(git rev-parse HEAD)" = "$revision"
    test -z "$(git status --porcelain=v1 --untracked-files=all)"
  )
  jq -e --arg revision "$revision" '
    .schema == "tei.reference.transform-engine-build-evidence/v1"
    and .repositoryRevision == $revision
    and .reproducible == true
    and .firstArtifactDigest == .secondArtifactDigest
    and .artifactDigest == .firstArtifactDigest
    and .runtimeEngine.implementationDigest == .artifactDigest
  ' "$directory/engine-build-evidence.json" >/dev/null
  jq -e --arg revision "$revision" --slurpfile engine "$directory/engine-build-evidence.json" '
    .schema == "tei.reference.plugin-self-build-evidence/v2"
    and .repositoryRevision == $revision
    and .reproducible == true
    and .transformBootstrap.exactRuntimeEngineBuildFingerprint == $engine[0].evidenceFingerprint
    and .transformBootstrap.runtimeEngine.implementationDigest == $engine[0].artifactDigest
    and (.stage2EngineLockFingerprint | length) == 64
    and (.stage2HostRegistrationFingerprint | length) == 64
    and (.stage2RuntimePolicyFingerprint | length) == 64
    and .transformBootstrap.convergenceResult.converged == true
    and .transformBootstrap.stage1ArtifactDigest == .transformBootstrap.stage2ArtifactDigest
    and .transformBootstrap.packageFirstArtifactDigest == .transformBootstrap.packageOracleDigest
    and .transformBootstrap.packageReproducible == true
  ' "$directory/self-build-evidence.json" >/dev/null
  actual_digest="sha256:$(sha256sum "$directory/transform-engine" | cut -d' ' -f1)"
  test "$actual_digest" = "$(jq -r '.artifactDigest' "$directory/engine-build-evidence.json")"
}
run_pair "$repo" "$output/baseline" "$baseline_revision"
run_pair "$mutation_repo" "$output/mutation" "$mutation_revision"
run_pair "$host_repo" "$output/host-only" "$host_revision"

# The independent Host edit changes an actual Go source file but must not
# rotate the exact interpreter executable, Engine Lock, or Host registration.
cmp -s "$output/baseline/transform-engine" "$output/host-only/transform-engine"
cmp -s "$output/baseline/stage2-engine-lock.json" "$output/host-only/stage2-engine-lock.json"
cmp -s "$output/baseline/stage2-host-registration.json" "$output/host-only/stage2-host-registration.json"
jq -s -e '
  .[0].repositoryRevision != .[1].repositoryRevision
  and .[0].sourceClosureDigest == .[1].sourceClosureDigest
  and .[0].artifactDigest == .[1].artifactDigest
  and .[0].runtimeEngine == .[1].runtimeEngine
' "$output/baseline/engine-build-evidence.json" "$output/host-only/engine-build-evidence.json" >/dev/null
jq -s -e '
  .[0].repositoryRevision != .[1].repositoryRevision
  and .[0].transformBootstrap.runtimeEngineSourceDigest == .[1].transformBootstrap.runtimeEngineSourceDigest
  and .[0].transformBootstrap.runtimeEngine == .[1].transformBootstrap.runtimeEngine
  and .[0].transformBootstrap.runtimeEngineFingerprint == .[1].transformBootstrap.runtimeEngineFingerprint
  and .[0].stage2EngineLockFingerprint == .[1].stage2EngineLockFingerprint
  and .[0].stage2HostRegistrationFingerprint == .[1].stage2HostRegistrationFingerprint
  and .[0].stage2RuntimePolicyFingerprint == .[1].stage2RuntimePolicyFingerprint
' "$output/baseline/self-build-evidence.json" "$output/host-only/self-build-evidence.json" >/dev/null

jq -n \
  --arg before "$baseline_revision" \
  --arg host "$host_revision" \
  --slurpfile original "$output/baseline/self-build-evidence.json" \
  --slurpfile hostEvidence "$output/host-only/self-build-evidence.json" \
  '{
    schema: "tei.reference.runtime-engine-host-isolation/v1",
    baselineRevision: $before,
    changedHostRevision: $host,
    changedHostSource: "internal/plugin/runtime/lease.go",
    engineImplementationDigest: $original[0].transformBootstrap.runtimeEngine.implementationDigest,
    engineFingerprint: $original[0].transformBootstrap.runtimeEngineFingerprint,
    stage2EngineLockFingerprint: $original[0].stage2EngineLockFingerprint,
    stage2HostRegistrationFingerprint: $original[0].stage2HostRegistrationFingerprint,
    stage2RuntimePolicyFingerprint: $original[0].stage2RuntimePolicyFingerprint,
    hostEngineDigestSame: ($original[0].transformBootstrap.runtimeEngine.implementationDigest == $hostEvidence[0].transformBootstrap.runtimeEngine.implementationDigest),
    verified: true
  }' > "$output/host-isolation-report.json"

# An old Host registration must never approve the rotated executable.
(
  cd "$mutation_repo"
  if go run ./tools/reference-plugin-self-build -repo . \
    -exact-runtime-engine "$output/mutation/transform-engine" \
    -engine-build-evidence "$output/mutation/engine-build-evidence.json" \
    -stage2-engine-lock "$output/mutation/stage2-engine-lock.json" \
    -stage2-host-registration "$output/baseline/stage2-host-registration.json" \
    > "$output/mutation/should-not-run.json" 2> "$output/mutation/host-denial.txt"; then
    echo "stale Host registration accepted the mutated Engine" >&2
    exit 1
  fi
)
grep -q "stage2 Host registration differs" "$output/mutation/host-denial.txt"
test ! -s "$output/mutation/should-not-run.json"

jq -s -e '
  .[0] as $before | .[1] as $after |
  ([$before, $after] | all(.[];
    . as $ev |
      any($ev.transformBootstrap.packageBuildPlan.dependencies[];
        .kind == "runtime-engine"
        and .artifactDigest == $ev.transformBootstrap.runtimeEngine.implementationDigest)
      and any($ev.transformBootstrap.packageBuildPlan.parameters[];
        .name == "runtime-engine-fingerprint"
        and .value == $ev.transformBootstrap.runtimeEngineFingerprint)
      and $ev.transformBootstrap.packageBuildPlanFingerprint != ""
  ))
  and $before.transformBootstrap.runtimeEngineSourceDigest != $after.transformBootstrap.runtimeEngineSourceDigest
  and $before.transformBootstrap.runtimeEngine.implementationDigest != $after.transformBootstrap.runtimeEngine.implementationDigest
  and $before.transformBootstrap.runtimeEngineFingerprint != $after.transformBootstrap.runtimeEngineFingerprint
  and $before.stage2EngineLockFingerprint != $after.stage2EngineLockFingerprint
  and $before.stage2HostRegistrationFingerprint != $after.stage2HostRegistrationFingerprint
  and $before.stage2RuntimePolicyFingerprint != $after.stage2RuntimePolicyFingerprint
  and $before.transformBootstrap.packageBuildPlanFingerprint != $after.transformBootstrap.packageBuildPlanFingerprint
  and $before.transformBootstrap.packageFirstArtifactDigest != $after.transformBootstrap.packageFirstArtifactDigest
  and $before.transformBootstrap.packageOracleDigest != $after.transformBootstrap.packageOracleDigest
  and $before.transformBootstrap.packageLockFingerprint != $after.transformBootstrap.packageLockFingerprint
' "$output/baseline/self-build-evidence.json" "$output/mutation/self-build-evidence.json" >/dev/null

jq -n \
  --arg before "$baseline_revision" \
  --arg after "$mutation_revision" \
  --slurpfile original "$output/baseline/self-build-evidence.json" \
  --slurpfile changed "$output/mutation/self-build-evidence.json" \
  '{
    schema: "tei.reference.transform-evidence-mutation/v1",
    beforeRevision: $before, afterRevision: $after,
    before: {
      sourceDigest: $original[0].transformBootstrap.runtimeEngineSourceDigest,
      engineDigest: $original[0].transformBootstrap.runtimeEngine.implementationDigest,
      engineFingerprint: $original[0].transformBootstrap.runtimeEngineFingerprint,
      buildPlanFingerprint: $original[0].transformBootstrap.packageBuildPlanFingerprint,
      packageDigest: $original[0].transformBootstrap.packageFirstArtifactDigest
    },
    after: {
      sourceDigest: $changed[0].transformBootstrap.runtimeEngineSourceDigest,
      engineDigest: $changed[0].transformBootstrap.runtimeEngine.implementationDigest,
      engineFingerprint: $changed[0].transformBootstrap.runtimeEngineFingerprint,
      buildPlanFingerprint: $changed[0].transformBootstrap.packageBuildPlanFingerprint,
      packageDigest: $changed[0].transformBootstrap.packageFirstArtifactDigest
    },
    verified: true
  }' > "$output/mutation-report.json"
cat "$output/mutation-report.json"
cat "$output/host-isolation-report.json"
test -z "$(git -C "$repo" status --porcelain=v1 --untracked-files=all)"
test "$(git -C "$repo" rev-parse HEAD)" = "$baseline_revision"
