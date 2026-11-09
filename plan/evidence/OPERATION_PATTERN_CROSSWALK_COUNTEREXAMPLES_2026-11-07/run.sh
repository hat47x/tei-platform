#!/bin/bash
# Counterexamples for the operation/pattern crosswalk rule.
#
# Usage: run.sh <outdir>
#
# Each case copies the accepted reference Workspace into a temporary directory
# and adds ONE declaration, so the effect of that declaration on the crosswalk
# is visible in isolation. The accepted Workspace itself is never edited.
set -eu

[ "$#" -ge 1 ] || { echo "usage: run.sh <outdir>" >&2; exit 2; }
OUT=$(mkdir -p "$1" && cd "$1" && pwd)
REPO=$(cd "$(dirname "$0")/../../.." && pwd)

# A worktree created on Windows records an absolute Windows path in .git, which
# a Linux git cannot resolve. Derive GIT_DIR from the worktree location; on a
# native checkout this is a no-op.
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

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

git -C "$REPO" rev-parse HEAD > "$OUT/head.txt"
go version >> "$OUT/head.txt"
git -C "$REPO" status --porcelain=v1 --untracked-files=all \
  | grep -v 'plan/evidence/OPERATION_PATTERN_CROSSWALK_COUNTEREXAMPLES_2026-11-07' > "$OUT/pre-status.txt" || true

for name in base effect-cmd readpath conflict no-persist; do
  cp -r "$REPO/reference/workspace" "$WORK/$name"
done

# A: a capability that declares an effect, and a command Pattern bound to it.
#    The rule must link only because the effect exists.
cat >> "$WORK/effect-cmd/semantic/sales/order.tei" <<'EOF'

capability sales.order.register {
    input order sales.order
    output sales.order
    effect create sales.order
}
EOF
cat > "$WORK/effect-cmd/applications/order-entry/patterns/register-order.tei" <<'EOF'
pattern order_entry.register_order command {
    capability sales.order.register
}
EOF

# B: a persistence mapping for the master alias. The reference Pattern already
#    names that datasource, so the read operation it needs can exist.
mkdir -p "$WORK/readpath/databases/master-data/persistence"
cat > "$WORK/readpath/databases/master-data/persistence/customer.tei" <<'EOF'
persist master_data.customer {
    database master_data
    semantic customer.customer
    table public.customer

    column customer.customer.customer_id -> customer_id storage int64 required
    column customer.customer.name -> name storage varchar required

    primary-key customer.customer.customer_id
}
EOF

# C: a second browse Pattern on the same Data. Two declarations may not both
#    own one derived operation; the rule must report a conflict, not choose.
cat > "$WORK/conflict/applications/order-entry/patterns/recent-orders-again.tei" <<'EOF'
pattern order_entry.recent_orders_again browse {
    data sales.order
    query order_entry.recent_orders
}
EOF

# D: a command whose capability declares an effect on a Data that has no
#    persistence mapping. The effect exists, but no write operation is derived.
cat >> "$WORK/no-persist/semantic/customer/customer.tei" <<'EOF'

capability customer.customer.rename {
    input customer customer.customer
    output customer.customer
    effect update customer.customer
}
EOF
cat > "$WORK/no-persist/applications/order-entry/patterns/rename-customer.tei" <<'EOF'
pattern order_entry.rename_customer command {
    capability customer.customer.rename
}
EOF

cd "$REPO"
for name in base effect-cmd readpath conflict no-persist; do
  go run ./tools/reference-operation-pattern-crosswalk/cmd/tei-operation-pattern-crosswalk \
    -workspace "$WORK/$name" > "$OUT/$name-crosswalk.json" 2> "$OUT/$name-crosswalk.err.txt"
done

python3 - "$OUT" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
lines = []
for name in ("base", "effect-cmd", "readpath", "conflict", "no-persist"):
    report = json.load(open(out / (name + "-crosswalk.json"), encoding="utf-8"))
    lines.append("== %s ==" % name)
    for app in report["applications"]:
        if app["id"] != "order_entry":
            continue
        lines.append("links (%d):" % len(app.get("links") or []))
        for link in app.get("links") or []:
            lines.append("  %s <- %s [%s] via=%s"
                         % (link["operation"], link["pattern"], link["patternKind"], link["via"]))
        lines.append("unlinked (%d):" % len(app.get("unlinked") or []))
        for item in app.get("unlinked") or []:
            lines.append("  %s (%s) %s: %s"
                         % (item["pattern"], item["kind"], item["code"], item["detail"]))
        lines.append("conflicts (%d):" % len(app.get("conflicts") or []))
        for conflict in app.get("conflicts") or []:
            lines.append("  %s <- %s" % (conflict["operation"], ",".join(conflict["patterns"])))
    lines.append("")
(out / "summary.txt").write_text("\n".join(lines) + "\n", encoding="utf-8", newline="\n")
print("\n".join(lines))
PY
