#!/bin/bash
set -eu
export PATH=/usr/local/go/bin:/usr/bin:/bin
export GIT_DIR=/mnt/d/GIT/hat47x/tei-platform/.git/worktrees/tei-wt-rule-20261108
export GIT_WORK_TREE=/mnt/d/GIT/hat47x/tei-wt-rule-20261108
cd "$GIT_WORK_TREE"
OUT=plan/evidence/PLUGIN_SELECTION_RULE_SURVEY_2026-11-08/raw
rm -rf "$OUT"; mkdir -p "$OUT"
git rev-parse HEAD > "$OUT/head.txt"; go version >> "$OUT/head.txt"
git status --porcelain=v1 --untracked-files=all | grep -v 'plan/evidence/PLUGIN_SELECTION_RULE_SURVEY' > "$OUT/pre-status.txt" || true
{
  echo "== selection diagnostics emitted by internal/validation =="
  grep -o 'errd("[A-Z_]*PLUGIN[A-Z_]*"' internal/validation/validation.go | sort -u
  echo
  echo "== which binding forms have a per-form 'unsupported' check =="
  for needle in DEPLOYMENT_EXECUTOR_CAPABILITY_UNSUPPORTED OBSERVATION_PROVIDER_CAPABILITY_UNSUPPORTED EVIDENCE_PROVIDER_CAPABILITY_UNSUPPORTED ENVIRONMENT_FEATURE_PLUGIN_MISSING DATABASE_PROVIDER_PLUGIN_MISSING MIGRATION_PROVIDER_PLUGIN_MISSING; do
    printf '%-52s %s\n' "$needle" "$(grep -c "\"$needle\"" internal/validation/validation.go || true)"
  done
  echo
  echo "== feature check guard =="
  grep -n 'FeatureSQLAnalysis' internal/validation/validation.go || true
  echo
  echo "== Known features =="
  grep -n 'FeatureSQLAnalysis\|FeatureDatabaseWorkbench' internal/environment/model.go || true
} > "$OUT/rule-scan.txt"
cat "$OUT/rule-scan.txt"