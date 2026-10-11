#!/usr/bin/env bash
# Isolated synthetic repository regression tests for public documentation selection.
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT
mkdir -p "$work/repo/docs/audits" "$work/repo/plan"
git -C "$work/repo" init -q
printf '# TEI\n\n> 生成AIを用いて開発中。人的レビューは未実施。\n' > "$work/repo/README.md"
printf 'MIT License\n' > "$work/repo/LICENSE"
printf '# Current audit\n\nSummary.\n' > "$work/repo/docs/audits/CURRENT_AUDIT.md"
printf '# 監査記録の移動案内\n\n[移動先](../docs/audits/CURRENT_AUDIT.md)\n' > "$work/repo/plan/OLD_AUDIT.md"
printf '# Index\n\n[Current](audits/CURRENT_AUDIT.md)\n' > "$work/repo/docs/index.md"
git -C "$work/repo" add -A
git -C "$work/repo" -c user.name=Test -c user.email=test@example.invalid -c commit.gpgsign=false commit -qm initial
sha=$(git -C "$work/repo" rev-parse HEAD)
cat > "$work/reviewed" <<'EOF'
README.md
LICENSE
docs/index.md
docs/audits/CURRENT_AUDIT.md
plan/OLD_AUDIT.md
EOF
printf 'plan/OLD_AUDIT.md\tdocs/audits/CURRENT_AUDIT.md\n' > "$work/map"
python3 "$here/select-public-docs.py" "$work/repo" "$sha" "$work/reviewed" "$work/map" "$work/selected" > "$work/report"
grep -q 'omitted=1' "$work/report"
! grep -q 'OLD_AUDIT.md' "$work/selected"
bash "$here/prepare-candidate.sh" "$work/repo" "$sha" "$work/selected" "$work/export" > /dev/null
bash "$here/verify-candidate.sh" "$work/repo" "$sha" "$work/selected" "$work/export" > /dev/null
[[ ! -e "$work/export/plan/OLD_AUDIT.md" ]]
[[ -f "$work/export/docs/audits/CURRENT_AUDIT.md" ]]

must_fail() {
  local source_sha=$1
  local selected_path=$2
  if python3 "$here/select-public-docs.py" "$work/repo" "$source_sha" "$selected_path" "$work/map" "$work/reject" > /dev/null 2>&1; then
    echo 'expected selection refusal' >&2
    exit 1
  fi
  [[ ! -e "$work/reject" ]]
}
# The canonical audit is mandatory when its compatibility path is omitted.
grep -v '^docs/audits/CURRENT_AUDIT.md$' "$work/reviewed" > "$work/without-canonical"
must_fail "$sha" "$work/without-canonical"

# A selected Markdown file that links to the omitted path cannot be published.
printf '# Index\n\n[Old](../plan/OLD_AUDIT.md)\n' > "$work/repo/docs/index.md"
git -C "$work/repo" add docs/index.md
git -C "$work/repo" -c user.name=Test -c user.email=test@example.invalid -c commit.gpgsign=false commit -qm link-to-legacy
linked_sha=$(git -C "$work/repo" rev-parse HEAD)
must_fail "$linked_sha" "$work/reviewed"

# Any locally linked file absent from the selected public tree is a blocker.
printf '# Index\n\n[Missing](missing.md)\n' > "$work/repo/docs/index.md"
git -C "$work/repo" add docs/index.md
git -C "$work/repo" -c user.name=Test -c user.email=test@example.invalid -c commit.gpgsign=false commit -qm unresolved-local-link
missing_sha=$(git -C "$work/repo" rev-parse HEAD)
must_fail "$missing_sha" "$work/reviewed"

# A map to an existing but unrelated file is not enough: the redirect must link there.
printf 'plan/OLD_AUDIT.md\tdocs/index.md\n' > "$work/wrong-map"
if python3 "$here/select-public-docs.py" "$work/repo" "$sha" "$work/reviewed" "$work/wrong-map" "$work/wrong-output" > /dev/null 2>&1; then
  echo 'expected incorrect mapping refusal' >&2
  exit 1
fi
[[ ! -e "$work/wrong-output" ]]

# Textual historical mention (not an active URL) may be retained for audit context.
printf '# Index\n\nHistoric file: plan/OLD_AUDIT.md. [Current](audits/CURRENT_AUDIT.md)\n' > "$work/repo/docs/index.md"
git -C "$work/repo" add docs/index.md
git -C "$work/repo" -c user.name=Test -c user.email=test@example.invalid -c commit.gpgsign=false commit -qm textual-history
historical_sha=$(git -C "$work/repo" rev-parse HEAD)
python3 "$here/select-public-docs.py" "$work/repo" "$historical_sha" "$work/reviewed" "$work/map" "$work/history-selected" > /dev/null
! grep -q '^plan/OLD_AUDIT.md$' "$work/history-selected"
echo 'PASS: public documentation selection and link-refusal tests'
