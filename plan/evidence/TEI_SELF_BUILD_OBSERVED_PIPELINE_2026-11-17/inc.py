"""Change experiment for ./cmd/tei: for each edit, compare three keys with the
artifact that a real, fresh build produces.

  declared key : the BuildPlan fingerprint (digests of every declared source)
  compile key  : names and digests of the files `go list` says are compiled,
                 plus go.mod and go.sum
  artifact     : sha256 of the binary built from scratch in a fresh state

A key is wrong in the dangerous direction when it stays the same while the
artifact changes, and wasteful when it changes while the artifact stays."""
import hashlib, json, os, shutil, subprocess, sys

W = os.path.expanduser("~/fable3")
SRC = W + "/tei"
REPO = W + "/inc-repo"
RUN = W + "/inc-run"
BIN = W + "/bin"
GOROOT = os.path.expanduser("~/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.linux-amd64")
GO = GOROOT + "/bin/go"
ENV = dict(os.environ, PATH=GOROOT + "/bin:" + os.environ["PATH"], GOTOOLCHAIN="local", GOFLAGS="-mod=mod")
SUPPLY = subprocess.check_output([GO, "env", "GOMODCACHE"], env=ENV, text=True).strip() + "/cache/download"


def sha(data):
    return hashlib.sha256(data).hexdigest()


def reset():
    subprocess.check_call(["rsync", "-a", "--delete", "--exclude", ".git", SRC + "/", REPO + "/"])


def compile_set():
    out = subprocess.check_output([GO, "list", "-deps", "-json=Dir,Standard,GoFiles,CgoFiles,SFiles,HFiles,EmbedFiles,IgnoredGoFiles", "./cmd/tei"],
                                  cwd=REPO, env=ENV, text=True)
    dec = json.JSONDecoder()
    i, files, ignored = 0, [], []
    while i < len(out):
        while i < len(out) and out[i].isspace():
            i += 1
        if i >= len(out):
            break
        pkg, i = dec.raw_decode(out, i)
        if pkg.get("Standard") or not pkg["Dir"].startswith(REPO):
            continue
        rel = pkg["Dir"][len(REPO) + 1:]
        for key in ("GoFiles", "CgoFiles", "SFiles", "HFiles", "EmbedFiles"):
            files += [rel + "/" + f for f in pkg.get(key) or []]
        ignored += [rel + "/" + f for f in pkg.get("IgnoredGoFiles") or []]
    return sorted(files), sorted(ignored)


def compile_key(files):
    h = hashlib.sha256()
    for rel in files + ["go.mod", "go.sum"]:
        h.update(rel.encode() + b"\0" + sha(open(REPO + "/" + rel, "rb").read()).encode() + b"\n")
    return h.hexdigest()


def measure(label):
    d = RUN + "/" + label
    subprocess.call(["chmod", "-R", "u+w", d], stderr=subprocess.DEVNULL)
    shutil.rmtree(d, ignore_errors=True)
    os.makedirs(d)
    files, ignored = compile_set()
    receipt = subprocess.run([BIN + "/resolver", "-intent", RUN + "/intent.json", "-catalog", RUN + "/catalog/catalog.json", "-source-root", REPO],
                             capture_output=True, text=True)
    if receipt.returncode != 0:
        return {"label": label, "resolveError": (receipt.stdout + receipt.stderr)[-300:]}
    plan = json.loads(receipt.stdout)
    json.dump(plan["plan"], open(d + "/plan.json", "w"))
    built = subprocess.run([RUN + "/catalog/builder", "-plan", d + "/plan.json", "-repo", REPO, "-go", GO, "-supply", SUPPLY,
                            "-artifact-out", d + "/artifact", "-state", d + "/state"], capture_output=True, text=True, env=ENV)
    rep = json.loads(built.stdout) if built.stdout.strip().startswith("{") else {}
    return {"label": label, "declared": plan["planFingerprint"], "compile": compile_key(files), "nfiles": len(files),
            "artifact": rep.get("artifactDigest"), "buildError": rep.get("buildError")}


def append(rel, text):
    with open(REPO + "/" + rel, "a") as f:
        f.write(text)


def write(rel, text):
    with open(REPO + "/" + rel, "w") as f:
        f.write(text)


EDITS = [
    ("base", lambda: None),
    ("base-again", lambda: None),
    ("edit-test-file", lambda: append("cmd/tei/cli_self_build_test.go", "\n// touched\n")),
    ("edit-readme-in-package", lambda: append("core/effect/README.md", "\ntouched\n")),
    ("edit-testdata", lambda: append("cmd/tei/testdata/l2b-go-command/main.go", "\n// touched\n")),
    ("edit-excluded-os-file", lambda: append("internal/workspace/source_open_unavailable.go", "\n// touched\n")),
    ("trailing-comment-in-compiled", lambda: append("cmd/tei/cli_self_build.go", "\n// touched\n")),
    ("leading-comment-in-compiled", lambda: write("internal/buildintent/intent.go", "// touched\n" + open(REPO + "/internal/buildintent/intent.go").read())),
    ("change-string-in-compiled", lambda: write("cmd/tei/cli_self_build_plan_handover.go",
        open(REPO + "/cmd/tei/cli_self_build_plan_handover.go").read().replace('"handed-over"', '"handed-over2"'))),
    ("add-compiled-file", lambda: write("cmd/tei/zz_added.go", "package main\n\nvar zzAdded = \"added\"\n\nfunc init() { _ = zzAdded }\n")),
    ("add-test-file", lambda: write("cmd/tei/zz_added_test.go", "package main\n")),
    ("add-file-excluded-by-constraint", lambda: write("cmd/tei/zz_added_windows.go", "package main\n\nvar zzWin = 1\n")),
    ("edit-go-mod-comment", lambda: append("go.mod", "\n// touched\n")),
    ("edit-docs-outside-declaration", lambda: append("README.md", "\ntouched\n")),
]

os.makedirs(RUN + "/catalog", exist_ok=True)
shutil.copy(W + "/run-files/intent.json", RUN + "/intent.json")
shutil.copy(W + "/run-files/builder", RUN + "/catalog/builder")
shutil.copy(W + "/run-files/capability.json", RUN + "/catalog/capability.json")
subprocess.check_call([BIN + "/catalog-author", "-declaration", RUN + "/catalog/capability.json", "-root", RUN + "/catalog", "-output", RUN + "/catalog/catalog.json"],
                      stdout=subprocess.DEVNULL)
only = sys.argv[1:]
rows = []
for label, edit in EDITS:
    if only and label not in only and label != "base":
        continue
    reset()
    edit()
    row = measure(label)
    rows.append(row)
    print(json.dumps(row), flush=True)
base = rows[0]
print("\nlabel | declared key | compile key | artifact | verdict")
for row in rows[1:]:
    if "declared" not in row:
        print(row["label"], "| resolve refused:", row.get("resolveError", "")[:160])
        continue
    dk = "same" if row["declared"] == base["declared"] else "CHANGED"
    ck = "same" if row["compile"] == base["compile"] else "CHANGED"
    if row["artifact"] is None:
        ar = "no artifact: " + str(row["buildError"])[:110]
    else:
        ar = "same" if row["artifact"] == base["artifact"] else "CHANGED"
    def verdict(k):
        if ar not in ("same", "CHANGED"):
            return "-"
        if k == "same" and ar == "CHANGED":
            return "FALSE REUSE"
        if k == "CHANGED" and ar == "same":
            return "needless rebuild"
        return "right"
    print(f"{row['label']} | {dk} | {ck} | {ar} | declared: {verdict(dk)}; compile: {verdict(ck)}")
json.dump(rows, open(RUN + "/rows.json", "w"), indent=1)
