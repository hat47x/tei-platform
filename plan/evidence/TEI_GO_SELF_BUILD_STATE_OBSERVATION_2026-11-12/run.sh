#!/bin/bash
# real Go child, state shared behind a matching self-report.
#   run.sh <repo-checkout>   (linux; needs python3, unshare, a Go toolchain)
# Runs, for child modes honest / share / copy:
#   V0  the real `tei self-build` entry (current judgement)
#   B1  observer.py without an OS boundary
#   B2  observer.py inside a user+mount namespace
set -uo pipefail
REPO="$1"
HERE="$(cd "$(dirname "$0")" && pwd)"
export PATH="$HOME/sdk/go/bin:$PATH"
W="$HOME/task6-work"; rm -rf "$W"; mkdir -p "$W/bin" "$W/src/hello" "$W/root/workspace" "$W/side"
cd "$REPO" || exit 1
echo "commit=$(git rev-parse HEAD)"
for t in ./cmd/tei ./tools/reference-build-catalog-authoring ./tools/reference-build-authoring-intent ./tools/reference-go-build-from-plan; do
  go build -o "$W/bin/$(basename "$t")" "$t" || exit 1
done
GO="$(cd "$REPO" && go env GOROOT)/bin/go"; echo "go=$GO $("$GO" version)"

cat > "$W/src/go.mod" <<'EOF'
module task6.example/hello

go 1.24
EOF
cat > "$W/src/hello/main.go" <<'EOF'
package main

import "fmt"

func main() { fmt.Println("task6 fixture") }
EOF
cp "$REPO/plan/evidence/TEI_GO_SELF_BUILD_CHILD_IDENTITY_SCOPE_2026-11-10/fixture/tei.workspace.yaml" "$W/root/workspace/"
cp "$W/bin/reference-go-build-from-plan" "$W/root/builder"
cat > "$W/root/declaration.json" <<'JSON'
{"providers":[{"plugin":"tei.builder.go","version":"1.2.0","implementation":"go.build-from-plan","contract":"tei.plugin-build","contractMajor":1,"artifact":"builder","recipeKinds":["go-command"],"targets":["host/linux/amd64"],"profiles":["release"]}],
 "targets":[{"target":"host/linux/amd64","profiles":["release"]}]}
JSON
cat > "$W/root/intent.json" <<'JSON'
{"formatVersion":1,
 "subject":{"kind":"plugin","id":"tei.task6.hello","version":"1.0.0","definitionSource":"pkg"},
 "sources":[{"kind":"go-package","id":"pkg","ref":"hello"},{"kind":"go-module","id":"mod","ref":"go.mod"}],
 "builder":{"contract":"tei.plugin-build","compatibleMajor":1},
 "recipe":{"kind":"go-command","entrypoint":"./hello"},
 "outputKind":"runtime-artifact","target":"host/linux/amd64","profile":"release"}
JSON
"$W/bin/reference-build-catalog-authoring" -declaration "$W/root/declaration.json" -root "$W/root" -output "$W/root/catalog.json" > "$W/authoring.json" 2>&1 || { cat "$W/authoring.json"; exit 1; }
DIGEST=$(python3 -c "import json;print(json.load(open('$W/authoring.json'))['catalogDigest'])")
"$W/bin/reference-build-authoring-intent" -intent "$W/root/intent.json" -catalog "$W/root/catalog.json" -source-root "$W/src" > "$W/receipt.json" || exit 1
python3 -c "import json;json.dump(json.load(open('$W/receipt.json'))['plan'],open('$W/plan.json','w'),ensure_ascii=False)"
CHILDARG=$(python3 -c "import json;print(json.dumps(['-plan','$W/plan.json','-intent','$W/root/intent.json','-repo','$W/src','-go','$GO']))")

# One wrapper file per mode. It keeps the self-report equal to the passed value.
for MODE in honest share copy; do
cat > "$W/child-$MODE" <<EOF
#!/usr/bin/env python3
import json, os, shutil, subprocess, sys
MODE, SIDE, SHARED = "$MODE", "$W/side", "$W/shared-$MODE"
args = sys.argv[1:]
i = args.index("-state"); passed = args[i + 1]; used = passed
def note(**kw):
    try: open(SIDE + "/wrapper-" + MODE + ".log", "a").write(json.dumps(kw) + "\n")
    except OSError: pass
if MODE == "share":
    used = SHARED
elif MODE == "copy":
    # Second attempt starts from the first attempt's state, copied into the passed directory.
    prev = SIDE + "/prev-" + MODE
    try:
        first = open(prev).read().strip()
        if first and first != passed and os.path.isdir(first):
            shutil.copytree(first, passed, dirs_exist_ok=True)
            note(copied_from=first, into=passed, entries=len(os.listdir(passed)))
    except OSError: pass
    try: open(prev, "w").write(passed)
    except OSError: pass
args[i + 1] = used
p = subprocess.run(["$W/bin/reference-go-build-from-plan"] + args, capture_output=True, text=True)
sys.stderr.write(p.stderr)
out = p.stdout
try:
    d = json.loads(out); real = d.get("stateDir")
    if "stateDir" in d: d["stateDir"] = passed
    out = json.dumps(d)
except Exception: real = None
note(A_passed=passed, B_used=used, C_reported=passed, real_child_said=real, exit=p.returncode)
sys.stdout.write(out); sys.exit(p.returncode)
EOF
chmod +x "$W/child-$MODE"
done

summ='import json,sys
d=json.load(sys.stdin)
print("   verdict=%s bytesEqual=%s selfReportMatchesPassed=%s" % (d["verdict"], d["artifactBytesEqual"], d["selfReportMatchesPassed"]))
for r in d["failReasons"]: print("   FAIL:", r)
for r in d["notObserved"]: print("   not observed:", r)
for a in d["attempts"]: print("   attempt %d exit=%s %.1fs statePost(files,bytes)=%s exe=%s err=%s" % (a["attempt"], a["exit"], a["seconds"], a["statePost"][:2], a["procExeAtStart"], a["buildError"]))'

for MODE in honest share copy; do
  echo; echo "################ child mode: $MODE"
  rm -f "$W/side/prev-$MODE"; rm -rf "$W/shared-$MODE"
  echo "== V0 current entry (tei self-build)"
  rm -rf "$W/v0-$MODE"; mkdir -p "$W/v0-$MODE"
  "$W/bin/tei" self-build -workspace "$W/root/workspace" -intent tei.task6.hello -intent-file "$W/root/intent.json" \
    -build-catalog "$W/root/catalog.json" -build-catalog-digest "$DIGEST" -build-source-root "$W/src" \
    -child "$W/child-$MODE" -child-arg "$CHILDARG" -state "$W/v0-$MODE/state" -output "$W/v0-$MODE/artifact" > "$W/v0-$MODE/report.json" 2>"$W/v0-$MODE/err"
  echo "   exit=$?"
  python3 -c "
import json;d=json.load(open('$W/v0-$MODE/report.json'));l=d.get('l2b') or {};r=d.get('artifactReproducibility') or {}
print('   ok=%s qualified=%s l2b.claimed=%s unmet=%s reproducibility=%s bytesEqual=%s' % (d.get('ok'),d.get('qualified'),l.get('claimed'),l.get('unmet'),r.get('state'),r.get('firstArtifactDigest')==r.get('secondArtifactDigest')))
if d.get('error'): print('   error:', d['error'])"
  for B in none ns; do
    echo "== observer boundary=$B"
    rm -f "$W/side/prev-$MODE"; rm -rf "$W/shared-$MODE" "$W/obs-$MODE-$B"; mkdir -p "$W/obs-$MODE-$B"
    python3 "$HERE/observer.py" --child "$W/child-$MODE" --child-arg-json "$CHILDARG" --work "$W/obs-$MODE-$B" --boundary "$B" \
      --probe "$W/side" --probe "$W" --probe "$HOME" --probe /dev/shm --probe /var/tmp --probe /mnt/c/Users/Public \
      > "$W/obs-$MODE-$B/verdict.json" 2>"$W/obs-$MODE-$B/err"
    python3 -c "$summ" < "$W/obs-$MODE-$B/verdict.json" || tail -5 "$W/obs-$MODE-$B/err"
  done
  echo "-- wrapper side log"; cat "$W/side/wrapper-$MODE.log" 2>/dev/null | cut -c1-260
done
