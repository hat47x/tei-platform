#!/usr/bin/env bash
# Verify an existing local source candidate against one immutable Git commit.
set -euo pipefail
[[ "$#" -eq 4 ]] || { echo 'Usage: verify-candidate.sh REPOSITORY EXACT_COMMIT_SHA REVIEWED_PATHS_FILE CANDIDATE_DIRECTORY' >&2; exit 2; }
repo_arg=$1
commit=$2
list=$3
candidate_arg=$4
[[ "$commit" =~ ^[0-9a-f]{40}$ ]] || { echo 'invalid exact commit SHA' >&2; exit 2; }
[[ -f "$list" && -r "$list" ]] || { echo 'manifest not readable' >&2; exit 2; }
repo=$(realpath -e "$(git -C "$repo_arg" rev-parse --show-toplevel)")
[[ -d "$candidate_arg" && ! -L "$candidate_arg" ]] || { echo 'candidate directory missing' >&2; exit 2; }
candidate=$(realpath -e "$candidate_arg")
case "$candidate/" in "$repo/"*) echo 'candidate must be outside repository' >&2; exit 2;; esac
[[ "$(git -C "$repo" rev-parse --verify "$commit^{commit}")" == "$commit" ]] || { echo 'source commit not found' >&2; exit 2; }

# Refuse links and special nodes, including inside nested directories.
if [[ -n "$(find "$candidate" ! -type f ! -type d -print -quit)" ]]; then
  echo 'candidate contains symlink or special node' >&2
  exit 3
fi

declare -A expected=()
count=0
while IFS= read -r path || [[ -n "$path" ]]; do
  [[ -z "$path" || "$path" == \#* ]] && continue
  [[ "$path" =~ ^[A-Za-z0-9_./-]+$ ]] || { echo 'invalid manifest path' >&2; exit 3; }
  if [[ "$path" == /* || "$path" == */ || "$path" == *"//"* ||
        "$path" =~ (^|/)\.\.?(/|$) || "$path" =~ (^|/)\.git(/|$) ]]; then
    echo 'unsafe manifest path' >&2
    exit 3
  fi
  [[ ! -v "expected[$path]" ]] || { echo 'duplicate path' >&2; exit 3; }
  expected["$path"]=1
  entry=$(git -C "$repo" ls-tree -r --full-tree "$commit" -- "$path")
  [[ -n "$entry" && "$entry" == *$'\t'* ]] || { echo 'missing source blob' >&2; exit 3; }
  [[ "${entry#*$'\t'}" == "$path" ]] || { echo 'manifest path is not an exact blob' >&2; exit 3; }
  read -r mode kind oid <<< "${entry%%$'\t'*}"
  [[ "$kind" == blob && ( "$mode" == 100644 || "$mode" == 100755 ) ]] || { echo 'unsupported source mode' >&2; exit 3; }
  [[ -f "$candidate/$path" && ! -L "$candidate/$path" ]] || { echo 'candidate blob missing' >&2; exit 3; }
  [[ "$(git -C "$repo" hash-object -- "$candidate/$path")" == "$oid" ]] || { echo 'candidate blob mismatch' >&2; exit 3; }
  permissions=$(stat -c %a -- "$candidate/$path")
  if [[ "$mode" == 100644 ]]; then [[ "$permissions" == 644 ]] || { echo 'file mode mismatch' >&2; exit 3; }
  else [[ "$permissions" == 755 ]] || { echo 'file mode mismatch' >&2; exit 3; }
  fi
  count=$((count+1))
done < "$list"
[[ "$count" -gt 0 && -v expected[README.md] && -v expected[LICENSE] ]] || { echo 'required files missing' >&2; exit 3; }
while IFS= read -r -d '' file; do
  path=${file#"$candidate"/}
  [[ -v "expected[$path]" ]] || { echo 'unexpected file in candidate' >&2; exit 3; }
done < <(find "$candidate" -type f -print0)
head -n 15 "$candidate/README.md" | grep -Fq '生成AI' || { echo 'AI development notice missing' >&2; exit 3; }
head -n 15 "$candidate/README.md" | grep -Fq '人的レビュー' || { echo 'human review notice missing' >&2; exit 3; }
printf 'VERIFIED_CANDIDATE_ONLY source=%s files=%s\n' "$commit" "$count"
printf '%s\n' 'NOT_APPROVED_FOR_PUBLICATION: content and independent legal/security review still required.'
