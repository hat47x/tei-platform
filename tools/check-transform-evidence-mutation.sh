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

mkdir -p "$output/baseline" "$output/mutation"
mutation_repo="$output/worktree"
cleanup() {
  git -C "$repo" worktree remove --force "$mutation_repo" >/dev/null 2>&1 || true
}
trap cleanup EXIT
git -C "$repo" worktree add --quiet --detach "$mutation_repo" "$baseline_revision"

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
    go run ./tools/reference-plugin-self-build -repo . \
      -exact-runtime-engine "$directory/transform-engine" \
      -engine-build-evidence "$directory/engine-build-evidence.json" \
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
test -z "$(git -C "$repo" status --porcelain=v1 --untracked-files=all)"
test "$(git -C "$repo" rev-parse HEAD)" = "$baseline_revision"
