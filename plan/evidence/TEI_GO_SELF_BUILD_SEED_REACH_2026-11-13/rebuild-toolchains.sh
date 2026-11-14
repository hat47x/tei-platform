#!/bin/bash
# Part 1 again, without CGO_ENABLED in the environment: the first run set
# CGO_ENABLED=0, which make.bash bakes into cmd/go as the default (go env showed
# CGO_ENABLED='0' in the rebuilt go and '1' in the distributed one).
set -uo pipefail
MOD="$HOME/go/pkg/mod/golang.org"; DL="$HOME/go/pkg/mod/cache/download/golang.org/toolchain/@v"
declare -A VER=( [X]=go1.26.11 [Y]=go1.27.0 ); declare -A ROOT
for s in X Y; do ROOT[$s]="$MOD/toolchain@v0.0.1-${VER[$s]}.linux-amd64"; done
W="$HOME/task7-work"; RES="$W/results-rebuild2.txt"; : > "$RES"
say() { echo "$*" | tee -a "$RES"; }
sha() { sha256sum "$1" | cut -d' ' -f1; }
rebuild() {
  local s="$1" b="$2" t="$W/gosrc2-$1-by-$2"
  say ""; say "== rebuild ${VER[$s]} from source with ${VER[$b]} as bootstrap (CGO_ENABLED unset)"
  chmod -R u+w "$t" 2>/dev/null; rm -rf "$t"; cp -r "${ROOT[$s]}" "$t" && chmod -R u+w "$t"
  rm -rf "$t/bin" "$t/pkg/tool" "$t/pkg/linux_amd64" "$t/pkg/distpack"
  local t0=$(date +%s)
  ( cd "$t/src" && env -i GOMAXPROCS=6 HOME="$W/home" PATH=/usr/bin:/bin TMPDIR="$W/tmp" GOROOT_BOOTSTRAP="${ROOT[$b]}" GOCACHE="$W/cache2-$s-by-$b" bash ./make.bash -distpack ) > "$W/make2-$s-by-$b.log" 2>&1
  local rc=$?; say "make.bash exit=$rc seconds=$(( $(date +%s) - t0 ))"
  [ $rc -ne 0 ] && { say "NOT RUN: $(tail -3 "$W/make2-$s-by-$b.log" | tr '\n' ' ')"; return; }
  local z="$t/pkg/distpack/v0.0.1-${VER[$s]}.linux-amd64.zip"
  say "rebuilt module zip     = $(sha "$z")"
  say "distributed module zip = $(sha "$DL/v0.0.1-${VER[$s]}.linux-amd64.zip")  (go.sum-style hash recorded at download: $(cat "$DL/v0.0.1-${VER[$s]}.linux-amd64.ziphash"))"
  say "moduleZipEqual=$([ "$(sha "$z")" = "$(sha "$DL/v0.0.1-${VER[$s]}.linux-amd64.zip")" ] && echo true || echo false)"
  local same=0 diff=0
  for f in bin/go bin/gofmt $(cd "${ROOT[$s]}" && ls pkg/tool/linux_amd64/ | sed 's|^|pkg/tool/linux_amd64/|'); do
    if [ -f "$t/$f" ] && [ "$(sha "$t/$f")" = "$(sha "${ROOT[$s]}/$f")" ]; then same=$((same+1)); else diff=$((diff+1)); say "  differs or missing: $f"; fi
  done
  say "toolBinaries equal=$same different=$diff"
}
rebuild X Y
rebuild Y X
