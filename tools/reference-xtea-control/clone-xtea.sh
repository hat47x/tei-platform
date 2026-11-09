#!/usr/bin/env sh
set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
target=${1:-upstream}
case "$target" in
  /*) target_path="$target" ;;
  *) target_path="$root/$target" ;;
esac
mkdir -p "$target_path"

for name in XeadModeler XeadEditor XeadDriver XeadServer; do
  destination="$target_path/$name"
  if [ ! -d "$destination/.git" ]; then
    case "$name" in
      XeadModeler) url="https://github.com/xead/XeadModeler.git" ;;
      XeadEditor)  url="https://github.com/xead/XeadEditor.git" ;;
      XeadDriver)  url="https://github.com/xead/XeadDriver.git" ;;
      XeadServer)  url="https://github.com/xead/XeadServer.git" ;;
    esac
    printf 'clone %s\n' "$name"
    git clone --depth 1 "$url" "$destination"
  else
    printf 'reuse %s\n' "$name"
  fi
  printf '%s %s\n' "$name" "$(git -C "$destination" rev-parse HEAD)"
done

printf 'source root: %s\n' "$target_path"
