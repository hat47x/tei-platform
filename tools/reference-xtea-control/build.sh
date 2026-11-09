#!/usr/bin/env sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
out="$root/build"
classes="$out/agent-classes"
fixture_classes="$out/fixture-classes"
jar_path="$out/xtea-control-agent.jar"

rm -rf "$classes" "$fixture_classes"
mkdir -p "$classes" "$fixture_classes"

find "$root/agent/src/xteaControl" -name '*.java' -print > "$out/sources.txt"
javac --release 8 -encoding UTF-8 -d "$classes" @"$out/sources.txt"

cat > "$out/MANIFEST.MF" <<'EOF'
Manifest-Version: 1.0
Premain-Class: xteaControl.Agent
Agent-Class: xteaControl.Agent
Can-Redefine-Classes: false
Can-Retransform-Classes: false
Implementation-Title: TEI X-TEA Control Agent
Implementation-Version: 1

EOF
jar cfm "$jar_path" "$out/MANIFEST.MF" -C "$classes" .
find "$root/tests/fixtures" -name '*.java' ! -name 'DriverFixture.java' -print > "$out/fixture-sources.txt"
javac --release 8 -encoding UTF-8 -d "$fixture_classes" @"$out/fixture-sources.txt"

printf 'agent:   %s\n' "$jar_path"
printf 'fixture: %s\n' "$fixture_classes"
