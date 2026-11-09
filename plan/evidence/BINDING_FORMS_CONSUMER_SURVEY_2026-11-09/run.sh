#!/bin/bash
# The Environment declares six binding-bearing fields. C-3 resolved the
# deployment-executor one from the declaration. This survey asks, for each of
# the other five, whether non-test code READS it to reach an implementation, or
# only validates/indexes it.
set -eu

[ "$#" -ge 1 ] || { echo "usage: run.sh <outdir>" >&2; exit 2; }
OUT=$(mkdir -p "$1" && cd "$1" && pwd)
REPO=$(cd "$(dirname "$0")/../../.." && pwd)

if [ -f "$REPO/.git" ]; then
  pointer=$(head -n 1 "$REPO/.git")
  case "$pointer" in
    gitdir:*[A-Za-z]:/*)
      win=${pointer#gitdir: }
      drive=$(printf '%s' "$win" | cut -c1 | tr '[:upper:]' '[:lower:]')
      GIT_DIR="/mnt/$drive${win#*:}"
      export GIT_DIR
      export GIT_WORK_TREE="$REPO"
      ;;
  esac
fi

export PATH=/usr/local/go/bin:/usr/bin:/bin:$PATH
cd "$REPO"

git rev-parse HEAD > "$OUT/head.txt"
go version >> "$OUT/head.txt"
git status --porcelain=v1 --untracked-files=all \
  | grep -v 'plan/evidence/BINDING_FORMS_CONSUMER_SURVEY_2026-11-09' > "$OUT/pre-status.txt" || true

# Only reads through a variable named env/environment/e/`Environment{}` count.
# A bare `.Bindings` on any other struct is not an Environment read.
scan() {
  field="$1"
  label="$2"
  out="$OUT/$(echo "$field" | tr 'A-Z' 'a-z').txt"
  {
    echo "== $label (Environment.$field) =="
    hits=$(grep -rn --include='*.go' -E "(env|environment|e)\.$field\b" internal cmd tools reference examples 2>/dev/null \
      | grep -v '_test.go' || true)
    if [ -z "$hits" ]; then
      echo "  (no non-test reader)"
    else
      printf '%s\n' "$hits" | sed 's/^/  /'
    fi
  } > "$out"
  cat "$out"
  echo ""
}

scan DeploymentExecutors  "deployment executor"
scan ObservationProviders "observation provider"
scan EvidenceProviders    "evidence provider"
scan Instances            "database instance"
scan Bindings             "application database binding"
scan ResourceBindings     "resource binding"
