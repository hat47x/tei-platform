#!/usr/bin/env bash
set -euo pipefail

ssh_keygen="${SSH_KEYGEN:-ssh-keygen}"
verify_time="${VERIFY_TIME:-20261007T120000Z}"
principal="reviewer@example.com"
namespace_accept="tei-authoring-v1.ws-ref-001.accept.scope-aaaaaaaa@tei-platform"
namespace_reject="tei-authoring-v1.ws-ref-001.reject.scope-aaaaaaaa@tei-platform"

root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fixture="${root}/fixture/witness-v0.txt"

if ! command -v "${ssh_keygen}" >/dev/null 2>&1; then
  echo "OpenSSH ssh-keygen is required for this Reference pressure test." >&2
  echo "No result is claimed. Install OpenSSH and rerun this script." >&2
  exit 77
fi

tmp="$(mktemp -d)"
trap 'rm -rf "${tmp}"' EXIT

key="${tmp}/reviewer"
message="${tmp}/witness-v0.txt"
allowed="${tmp}/allowed_signers"
expired="${tmp}/allowed_signers_expired"
future="${tmp}/allowed_signers_future"
revoked="${tmp}/revoked.krl"

cp "${fixture}" "${message}"
"${ssh_keygen}" -q -t ed25519 -N "" -f "${key}"

pub="$(cat "${key}.pub")"
printf '%s namespaces="%s",valid-after="20200101Z",valid-before="20990101Z" %s\n'   "${principal}" "${namespace_accept}" "${pub}" > "${allowed}"
printf '%s namespaces="%s",valid-before="20200101Z" %s\n'   "${principal}" "${namespace_accept}" "${pub}" > "${expired}"
printf '%s namespaces="%s",valid-after="20990101Z" %s\n'   "${principal}" "${namespace_accept}" "${pub}" > "${future}"

"${ssh_keygen}" -q -Y sign -f "${key}" -n "${namespace_accept}" "${message}"
signature="${message}.sig"

verify() {
  local policy="$1"
  local identity="$2"
  local namespace="$3"
  local sig="$4"
  local input="$5"
  shift 5
  "${ssh_keygen}" -Y verify     -O "verify-time=${verify_time}"     -f "${policy}"     -I "${identity}"     -n "${namespace}"     -s "${sig}"     "$@" < "${input}" >/dev/null 2>&1
}

expect_success() {
  local label="$1"
  shift
  if ! "$@"; then
    echo "FAIL: ${label}" >&2
    exit 1
  fi
  echo "PASS: ${label}"
}

expect_failure() {
  local label="$1"
  shift
  if "$@"; then
    echo "FAIL: ${label} unexpectedly succeeded" >&2
    exit 1
  fi
  echo "PASS: ${label} failed closed"
}

expect_success   "exact principal + exact namespace + valid policy"   verify "${allowed}" "${principal}" "${namespace_accept}" "${signature}" "${message}"

expect_failure   "wrong principal"   verify "${allowed}" "other@example.com" "${namespace_accept}" "${signature}" "${message}"

expect_failure   "accept signature reused as reject"   verify "${allowed}" "${principal}" "${namespace_reject}" "${signature}" "${message}"

tampered="${tmp}/witness-tampered.txt"
cp "${message}" "${tampered}"
sed -i 's/candidateFingerprint=sha256:c/candidateFingerprint=sha256:f/' "${tampered}"
expect_failure   "message changed after signing"   verify "${allowed}" "${principal}" "${namespace_accept}" "${signature}" "${tampered}"

expect_failure   "signer policy expired at verification time"   verify "${expired}" "${principal}" "${namespace_accept}" "${signature}" "${message}"

expect_failure   "signer policy not valid yet"   verify "${future}" "${principal}" "${namespace_accept}" "${signature}" "${message}"

"${ssh_keygen}" -q -k -f "${revoked}" "${key}.pub"
expect_failure   "revoked signing key"   verify "${allowed}" "${principal}" "${namespace_accept}" "${signature}" "${message}" -r "${revoked}"

echo "OpenSSH authoring witness Reference pressure passed."
