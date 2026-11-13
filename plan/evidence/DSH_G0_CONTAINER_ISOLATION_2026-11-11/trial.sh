#!/bin/bash
# 4-slot G0/G1-topology/G2 preflight. Synthetic canaries + reviewer-side mock only. No external egress.
set -e
SP=$(cd "$(dirname "$0")" && pwd)
B=~/dsh-poc; TR=$B/trial; IMG=node:24-bookworm-slim
NODE=~/.nvm/versions/node/v24.22.2
SLOTS="h2-a:2001 h3-a:2002 h3-b:2003 h2-b:2004"
# --- clean (root helper needed because slot dirs are owned by other uids)
if [ -d "$TR" ]; then docker run --rm -v "$B":/b $IMG rm -rf /b/trial; fi
mkdir -p "$TR/slots" "$TR/reviewer" "$TR/log" "$TR/scripts"
cp "$SP"/mock.js "$SP"/gen_steps.js "$SP"/trial.sh "$SP"/results2.js "$TR/scripts/"
echo "G0_CANARY_REVIEWER_EXPECTED_7F31" > "$TR/reviewer/expected.txt"; chmod -R go-rwx "$TR/reviewer"
( cd $B/g0/slot && $B/g0/bin/tei model catalog -workspace reference/workspace ) > "$TR/log/catalog.direct.json"
echo "== pins"; sha256sum $B/g0/bin/tei "$TR/log/catalog.direct.json" | cut -c1-64; $NODE/bin/node --version
for sv in $SLOTS; do s=${sv%%:*}; u=${sv##*:}
  R=$TR/slots/$s; mkdir -p "$R/root/reference" "$R/home"
  cp -r $B/tei-src/reference/workspace "$R/root/reference/workspace"
  if [[ $s == h3-* ]]; then mkdir -p "$R/root/.agents/skills"; cp -r $B/tei-src/skills/yi-tei-modeling "$R/root/.agents/skills/yi-tei-modeling"; fi
  echo "G0_CANARY_LANE_$(echo $s | tr 'a-z-' 'A-Z_')_$u" > "$R/lane-canary.txt"
  echo "$s ws=$(cd $R/root/reference/workspace && find . -type f -print0 | sort -z | xargs -0 sha256sum | sha256sum | cut -c1-16) roottree=$(cd $R/root && find . -type f -print0 | sort -z | xargs -0 sha256sum | sha256sum | cut -c1-16) files=$(find $R/root -type f | wc -l)"
done
CH=""; for sv in $SLOTS; do s=${sv%%:*}; u=${sv##*:}; CH="$CH chown -R $u:$u /s/$s; chmod 700 /s/$s;"; done
docker run --rm -v "$TR/slots":/s $IMG sh -c "$CH ls -ln /s"
# --- per-slot run
for sv in $SLOTS; do s=${sv%%:*}; u=${sv##*:}
  L=$TR/log/$s; mkdir -p "$L"; node "$SP/gen_steps.js" $s > "$L/steps.json"
  docker network create --internal g1-$s >/dev/null
  docker run -d --rm --name mock-$s --network g1-$s --user 1000:1000 -e MOCK_HOST=0.0.0.0 \
    -v "$L":/out -v "$TR/scripts":/sp:ro $IMG node /sp/mock.js /out /out/steps.json >/dev/null
  sleep 2
  set +e
  docker run --rm --name part-$s --network g1-$s --user $u:$u --read-only --cap-drop ALL --security-opt no-new-privileges \
    --tmpfs /tmp:exec,mode=1777 \
    -v $NODE:/opt/node:ro -v ~/dsh-native/node_modules:/opt/dsh/node_modules:ro -v $B/g0/bin:/opt/tei:ro \
    -v "$TR/slots/$s/root":/trial:ro -v "$TR/slots/$s/home":/dsh-home \
    -v "$TR/slots":/others:ro -v "$TR/reviewer":/reviewer:ro \
    -w /trial -e HOME=/tmp/h -e DSH_HOME=/dsh-home -e PATH=/opt/tei:/opt/node/bin:/usr/bin:/bin \
    -e DEEPSEEK_API_KEY=dummy -e DEEPSEEK_BASE_URL=http://mock-$s:8799 -e DSH_TELEMETRY_MODE=DISABLED \
    --entrypoint /bin/sh $IMG -c 'mkdir -p /tmp/h; exec timeout 300 /opt/dsh/node_modules/.bin/dsh --profile headless --permission-mode read-only --approval-policy never --json "G0 preflight: run the scripted probes."' \
    > "$L/stdout.jsonl" 2> "$L/stderr.txt"
  echo "== $s uid=$u dsh_exit=$? requests=$(wc -l < "$L/requests.log" 2>/dev/null) stderr_bytes=$(wc -c < "$L/stderr.txt")"
  set -e
  docker stop mock-$s >/dev/null 2>&1 || true; docker network rm g1-$s >/dev/null
done
# --- results (root helper reads slot homes)
docker run --rm --network none -v "$TR":/t:ro $IMG node /t/scripts/results2.js /t
