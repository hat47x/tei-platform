#!/usr/bin/env bash
# Build a local, unapproved source-only publication candidate.
# This command never changes Git refs, creates a repository, or pushes anywhere.
set -euo pipefail
umask 077

usage() {
  printf 'Usage: %s REPOSITORY EXACT_COMMIT_SHA REVIEWED_PATHS_FILE OUTPUT_DIRECTORY\n' "$0" >&2
  exit 2
}
[[ "$#" -eq 4 ]] || usage
repo_arg=$1
commit=$2
list=$3
dest_arg=$4

command -v git >/dev/null 2>&1 || { echo "missing git" >&2; exit 2; }
command -v realpath >/dev/null 2>&1 || { echo "missing realpath" >&2; exit 2; }
[[ "$commit" =~ ^[0-9a-f]{40}$ ]] || { echo "expected full lowercase 40-character commit SHA" >&2; exit 2; }
[[ -f "$list" && -r "$list" ]] || { echo "reviewed path list not readable" >&2; exit 2; }

repo=$(git -C "$repo_arg" rev-parse --show-toplevel)
repo=$(realpath -e "$repo")
dest=$(realpath -m "$dest_arg")
case "$dest/" in
  "$repo/"*) echo "output must be outside source repository" >&2; exit 2 ;;
esac
[[ ! -e "$dest" && ! -L "$dest" ]] || { echo "output already exists" >&2; exit 2; }
[[ "$(git -C "$repo" rev-parse --verify "$commit^{commit}")" == "$commit" ]] || {
  echo "commit not found in local repository" >&2
  exit 2
}

parent=$(dirname "$dest")
[[ -d "$parent" && -w "$parent" ]] || { echo "output parent must exist and be writable" >&2; exit 2; }
tmp=$(mktemp -d "${dest}.tmp.XXXXXX")
cleanup() { if [[ -d "$tmp" ]]; then rm -rf -- "$tmp"; fi; }
trap cleanup EXIT

declare -A seen=()
count=0
while IFS= read -r path || [[ -n "$path" ]]; do
  [[ -z "$path" || "$path" == \#* ]] && continue
  [[ "$path" =~ ^[A-Za-z0-9_./-]+$ ]] || { echo "invalid manifest path" >&2; exit 3; }
  if [[ "$path" == /* || "$path" == */ || "$path" == *"//"* ||
        "$path" =~ (^|/)\.\.?(/|$) || "$path" =~ (^|/)\.git(/|$) ]]; then
    echo "unsafe manifest path" >&2
    exit 3
  fi
  [[ ! -v "seen[$path]" ]] || { echo "duplicate manifest path" >&2; exit 3; }
  seen["$path"]=1

  # ls-tree must resolve to exactly one regular Git blob at the requested path.
  entry=$(git -C "$repo" ls-tree -r --full-tree "$commit" -- "$path")
  [[ -n "$entry" && "$entry" == *$'\t'* ]] || { echo "missing path in source tree: $path" >&2; exit 3; }
  resolved=${entry#*$'\t'}
  [[ "$resolved" == "$path" ]] || { echo "manifest path is not an exact blob: $path" >&2; exit 3; }
  meta=${entry%%$'\t'*}
  read -r mode kind oid <<< "$meta"
  [[ "$kind" == blob && ( "$mode" == 100644 || "$mode" == 100755 ) &&
     "$oid" =~ ^[0-9a-f]{40}$ ]] || { echo "unsupported Git object mode: $path" >&2; exit 3; }

  mkdir -p -- "$tmp/$(dirname "$path")"
  git -C "$repo" cat-file blob "$oid" > "$tmp/$path"
  [[ "$(git -C "$repo" hash-object -- "$tmp/$path")" == "$oid" ]] || {
    echo "blob verification failed: $path" >&2
    exit 3
  }
  if [[ "$mode" == 100755 ]]; then chmod 755 "$tmp/$path"; else chmod 644 "$tmp/$path"; fi
  count=$((count + 1))
done < "$list"

[[ "$count" -gt 0 && -f "$tmp/README.md" && -f "$tmp/LICENSE" ]] || {
  echo "candidate requires at least README.md and LICENSE" >&2
  exit 3
}
head -n 15 "$tmp/README.md" | grep -Fq '生成AI' || {
  echo "README must disclose AI-assisted development at its start" >&2
  exit 3
}
head -n 15 "$tmp/README.md" | grep -Fq '人的レビュー' || {
  echo "README must disclose human-review status at its start" >&2
  exit 3
}

mv -- "$tmp" "$dest"
trap - EXIT
printf 'CANDIDATE_ONLY source=%s files=%s output=%s\n' "$commit" "$count" "$dest"
printf '%s\n' 'NOT_AUDITED: independently review every output file, licenses, metadata and required tests before publishing.'
