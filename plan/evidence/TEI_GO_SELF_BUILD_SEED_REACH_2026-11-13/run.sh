#!/bin/bash
# How far does the seed reach? (2026-11-13 record; not a product tool)
#   run.sh <repo-clone> <commit>   (linux/amd64; needs two Go toolchains in the module cache)
# Part 1  rebuild each Go toolchain from its own source with the OTHER one as
#         bootstrap, and compare with the distributed toolchain (diverse double compiling).
# Part 2  2x2: driver set built by seed D  x  compiling toolchain G, through the
#         real `tei self-build` full chain on ./cmd/tei.
set -uo pipefail
REPO="$1"; COMMIT="$2"
MOD="$HOME/go/pkg/mod/golang.org"
DL="$HOME/go/pkg/mod/cache/download/golang.org/toolchain/@v"
declare -A VER=( [X]=go1.26.11 [Y]=go1.27.0 )
declare -A ROOT
for s in X Y; do ROOT[$s]="$MOD/toolchain@v0.0.1-${VER[$s]}.linux-amd64"; done
W="$HOME/task7-work"; rm -rf "$W" 2>/dev/null; chmod -R u+w "$W" 2>/dev/null; rm -rf "$W"; mkdir -p "$W"
RES="$W/results.txt"; : > "$RES"
say() { echo "$*" | tee -a "$RES"; }
sha() { sha256sum "$1" | cut -d' ' -f1; }
CLEAN_ENV=(env -i GOMAXPROCS=6 HOME="$W/home" PATH=/usr/bin:/bin TMPDIR="$W/tmp" GOTOOLCHAIN=local GOFLAGS= GOWORK=off GOENV=off CGO_ENABLED=0 GOPROXY=off GOSUMDB=off)
mkdir -p "$W/home" "$W/tmp"

cd "$REPO" && git checkout -q --detach "$COMMIT" && [ -z "$(git status --porcelain)" ] || { echo "repo not clean at $COMMIT"; exit 1; }
say "commit=$(git rev-parse HEAD) host=$(uname -sm) date=$(date -u +%FT%TZ)"
for s in X Y; do say "seed $s = ${VER[$s]} bin/go=$(sha "${ROOT[$s]}/bin/go") moduleZip=$(sha "$DL/v0.0.1-${VER[$s]}.linux-amd64.zip") ziphash=$(cat "$DL/v0.0.1-${VER[$s]}.linux-amd64.ziphash")"; done

############ Part 1: rebuild toolchain S from source, bootstrapped by the other seed
rebuild() { # $1 = seed whose source is rebuilt, $2 = bootstrap seed
  local s="$1" b="$2" t="$W/gosrc-$1-by-$2"
  say ""; say "== rebuild ${VER[$s]} from source with ${VER[$b]} as bootstrap"
  cp -r "${ROOT[$s]}" "$t" && chmod -R u+w "$t"
  rm -rf "$t/bin" "$t/pkg/tool" "$t/pkg/linux_amd64" "$t/pkg/distpack"
  local t0=$(date +%s)
  ( cd "$t/src" && "${CLEAN_ENV[@]}" GOROOT_BOOTSTRAP="${ROOT[$b]}" GOCACHE="$W/cache-$s-by-$b" bash ./make.bash -distpack ) > "$W/make-$s-by-$b.log" 2>&1
  local rc=$?
  say "make.bash exit=$rc seconds=$(( $(date +%s) - t0 ))"
  [ $rc -ne 0 ] && { say "NOT RUN: rebuild failed: $(tail -3 "$W/make-$s-by-$b.log" | tr '\n' ' ')"; return; }
  local z="$t/pkg/distpack/v0.0.1-${VER[$s]}.linux-amd64.zip"
  if [ -f "$z" ]; then
    say "rebuilt module zip   = $(sha "$z")"
    say "distributed module zip = $(sha "$DL/v0.0.1-${VER[$s]}.linux-amd64.zip")"
    say "moduleZipEqual=$([ "$(sha "$z")" = "$(sha "$DL/v0.0.1-${VER[$s]}.linux-amd64.zip")" ] && echo true || echo false)"
  else say "no module zip produced"; fi
  local same=0 diff=0
  for f in bin/go bin/gofmt $(cd "${ROOT[$s]}" && ls pkg/tool/linux_amd64/ | sed 's|^|pkg/tool/linux_amd64/|'); do
    if [ -f "$t/$f" ] && [ "$(sha "$t/$f")" = "$(sha "${ROOT[$s]}/$f")" ]; then same=$((same+1)); else diff=$((diff+1)); say "  differs or missing: $f"; fi
  done
  say "toolBinaries equal=$same different=$diff (bin/go, bin/gofmt, pkg/tool/linux_amd64/*)"
}
rebuild X Y
rebuild Y X

############ Part 2: driver sets and the 2x2
say ""; say "== driver sets (tei + builder + authoring tools), one per seed"
for s in X Y; do
  mkdir -p "$W/drv-$s"
  for t in ./cmd/tei ./tools/reference-go-build-from-plan ./tools/reference-build-catalog-authoring ./tools/reference-build-authoring-intent; do
    ( cd "$REPO" && "${CLEAN_ENV[@]}" GOCACHE="$W/cache-drv-$s" GOMODCACHE="$HOME/go/pkg/mod" GOFLAGS=-mod=mod GOPROXY=off "${ROOT[$s]}/bin/go" build -trimpath -buildvcs=false -o "$W/drv-$s/$(basename "$t")" "$t" ) 2>>"$W/drv-$s.err" || { say "driver build failed for $s $t: $(tail -2 "$W/drv-$s.err")"; exit 1; }
  done
  say "driver set $s: tei=$(sha "$W/drv-$s/tei") builder=$(sha "$W/drv-$s/reference-go-build-from-plan")"
done
[ -z "$(cd "$REPO" && git status --porcelain)" ] || say "WARNING: the driver build changed the working tree: $(cd "$REPO" && git status --porcelain | head -3)"

# The declaration: every module-local package ./cmd/tei compiles, plus go.mod and go.sum.
python3 - "$REPO" "${ROOT[X]}/bin/go" "$W/intent.json" <<'PY'
import json, os, subprocess, sys
repo, go, out = sys.argv[1:4]
env = dict(os.environ, GOTOOLCHAIN="local", GOFLAGS="-mod=mod", GOWORK="off", CGO_ENABLED="0", GOPROXY="off")
dirs = sorted({d for d in subprocess.run([go, "list", "-deps", "-f", "{{if not .Standard}}{{.Dir}}{{end}}", "./cmd/tei"], cwd=repo, env=env, capture_output=True, text=True, check=True).stdout.split("\n") if d.startswith(repo)})
sources = [{"kind": "go-package", "id": "pkg%d" % i, "ref": os.path.relpath(d, repo)} for i, d in enumerate(dirs)]
sources += [{"kind": "go-module", "id": "go.mod", "ref": "go.mod"}, {"kind": "go-module", "id": "go.sum", "ref": "go.sum"}]
json.dump({"formatVersion": 1, "subject": {"kind": "plugin", "id": "tei.command.tei", "version": "0.0.0", "definitionSource": "pkg0"},
           "sources": sources, "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
           "recipe": {"kind": "go-command", "entrypoint": "./cmd/tei"}, "outputKind": "runtime-artifact",
           "target": "host/linux/amd64", "profile": "release"}, open(out, "w"), indent=2)
print("declared sources:", len(sources))
PY
say "intent=$(sha "$W/intent.json")"

cell() { # $1 = driver label, $2 = dir holding tei + builder, $3 = compiling seed
  local d="$1" drv="$2" g="$3" c="$W/cell-$1-$3"
  mkdir -p "$c/root/workspace"
  printf 'workspace: task7\nformatVersion: 1\nsources:\n  - semantic\n' > "$c/root/workspace/tei.workspace.yaml"
  cp "$drv/reference-go-build-from-plan" "$c/root/builder"
  cat > "$c/root/declaration.json" <<'JSON'
{"providers":[{"plugin":"tei.builder.go","version":"1.2.0","implementation":"go.build-from-plan","contract":"tei.plugin-build","contractMajor":1,"artifact":"builder","recipeKinds":["go-command"],"targets":["host/linux/amd64"],"profiles":["release"]}],
 "targets":[{"target":"host/linux/amd64","profiles":["release"]}]}
JSON
  "$drv/reference-build-catalog-authoring" -declaration "$c/root/declaration.json" -root "$c/root" -output "$c/root/catalog.json" > "$c/authoring.json" 2>&1 || { say "cell $d/$g: catalog failed"; return; }
  local digest=$(python3 -c "import json;print(json.load(open('$c/authoring.json'))['catalogDigest'])")
  "$drv/reference-build-authoring-intent" -intent "$W/intent.json" -catalog "$c/root/catalog.json" -source-root "$REPO" > "$c/receipt.json" 2>"$c/resolve.err" || { say "cell $d/$g: resolve failed: $(head -2 "$c/resolve.err")"; return; }
  python3 -c "import json;json.dump(json.load(open('$c/receipt.json'))['plan'],open('$c/plan.json','w'),ensure_ascii=False)"
  local arg=$(python3 -c "import json;print(json.dumps(['-plan','$c/plan.json','-intent','$W/intent.json','-repo','$REPO','-go','${ROOT[$g]}/bin/go']))")
  local t0=$(date +%s)
  "$drv/tei" self-build -workspace "$c/root/workspace" -intent tei.command.tei -intent-file "$W/intent.json" \
    -build-catalog "$c/root/catalog.json" -build-catalog-digest "$digest" -build-source-root "$REPO" \
    -child "$drv/reference-go-build-from-plan" -child-arg "$arg" -supply "$HOME/go/pkg/mod/cache/download" \
    -state "$c/state" -output "$c/artifact" > "$c/report.json" 2>"$c/err"
  local rc=$?
  python3 - "$c/report.json" "$d" "$g" "$rc" "$(( $(date +%s) - t0 ))" <<'PY' | tee -a "$RES"
import json, sys
try: r = json.load(open(sys.argv[1]))
except Exception as e: print("cell driver=%s compile=%s exit=%s NOT DECODED: %s" % (sys.argv[2], sys.argv[3], sys.argv[4], e)); raise SystemExit
l = r.get("l2b") or {}; p = r.get("artifactReproducibility") or {}; c = r.get("childRun") or {}; pr = r.get("planReceipt") or {}
print("cell driver=%s compile=%s exit=%s seconds=%s ok=%s qualified=%s l2b.claimed=%s unmet=%s reproducibility=%s invoker=%s plan=%s artifact=%s" % (
    sys.argv[2], sys.argv[3], sys.argv[4], sys.argv[5], r.get("ok"), r.get("qualified"), l.get("claimed"), l.get("unmet"), p.get("state"),
    (c.get("invokerDigest") or "")[7:23], (pr.get("planFingerprint") or "")[7:23], c.get("artifactCrossCheckedDigest")))
if r.get("error"): print("   error:", json.dumps(r["error"])[:300])
PY
}
say ""; say "== 2x2: driver seed x compiling toolchain (real tei self-build, ./cmd/tei)"
for d in X Y; do for g in X Y; do cell "$d" "$W/drv-$d" "$g"; done; done

say ""; say "== comparisons"
for g in X Y; do
  a="$W/cell-X-$g/artifact"; b="$W/cell-Y-$g/artifact"
  [ -f "$a" ] && [ -f "$b" ] && say "compile=$g: artifact(driver X) == artifact(driver Y): $([ "$(sha "$a")" = "$(sha "$b")" ] && echo true || echo false)"
  [ -f "$W/cell-$g-$g/artifact" ] && say "compile=$g: stage2 artifact == stage1 tei (seed build of driver $g): $([ "$(sha "$W/cell-$g-$g/artifact")" = "$(sha "$W/drv-$g/tei")" ] && echo true || echo false)"
done
[ -f "$W/cell-X-X/artifact" ] && [ -f "$W/cell-X-Y/artifact" ] && say "driver=X: artifact(compile X) == artifact(compile Y): $([ "$(sha "$W/cell-X-X/artifact")" = "$(sha "$W/cell-X-Y/artifact")" ] && echo true || echo false)"

# Stage 3 where stage 2 is not byte-identical to stage 1: let the stage-2 artifact drive.
for g in X Y; do
  a="$W/cell-$g-$g/artifact"
  if [ -f "$a" ] && [ "$(sha "$a")" != "$(sha "$W/drv-$g/tei")" ]; then
    say ""; say "== stage 3 for compile=$g: the stage-2 artifact drives the same build"
    mkdir -p "$W/drv3-$g"; cp "$a" "$W/drv3-$g/tei"; chmod +x "$W/drv3-$g/tei"
    cp "$W/drv-$g/reference-go-build-from-plan" "$W/drv-$g/reference-build-catalog-authoring" "$W/drv-$g/reference-build-authoring-intent" "$W/drv3-$g/"
    cell "S2$g" "$W/drv3-$g" "$g"
    [ -f "$W/cell-S2$g-$g/artifact" ] && say "compile=$g: stage3 artifact == stage2 artifact: $([ "$(sha "$W/cell-S2$g-$g/artifact")" = "$(sha "$a")" ] && echo true || echo false)"
  fi
done
say ""; say "done $(date -u +%FT%TZ)"
