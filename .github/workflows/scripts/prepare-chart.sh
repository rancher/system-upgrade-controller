#!/bin/bash
# Run from the repository root. Prints the temporary chart directory.
set -euo pipefail

if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <vX.Y.Z> <output-directory>" >&2
  exit 1
fi

TAG=$1
if ! [[ "$TAG" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "Error: only stable vX.Y.Z releases are supported for chart staging" >&2
  exit 1
fi

SOURCE=charts/system-upgrade-controller
if [ ! -f "$SOURCE/Chart.yaml" ] || [ ! -f "$SOURCE/values.yaml" ]; then
  echo "Error: import the Rancher chart into $SOURCE first" >&2
  exit 1
fi
if ! grep -Eq '^name: system-upgrade-controller[[:space:]]*$' "$SOURCE/Chart.yaml" ||
   ! grep -Eq '^version:[[:space:]]*[^[:space:]#]' "$SOURCE/Chart.yaml"; then
  echo "Error: Chart.yaml must name system-upgrade-controller and contain a version" >&2
  exit 1
fi
if grep -Eq '^[[:space:]]+helm\.sh/images:' "$SOURCE/Chart.yaml"; then
  echo "Error: omit generated helm.sh/images from the source; cargueiro must regenerate it from values.yaml" >&2
  exit 1
fi

SOURCE=$(realpath "$SOURCE")
OUT_DIR=$(realpath -m "$2")
case "$OUT_DIR" in
  "$SOURCE"|"$SOURCE"/*)
    echo "Error: the output directory must be outside the source chart" >&2
    exit 1
    ;;
esac
DEST="$OUT_DIR/system-upgrade-controller"
if [ -e "$DEST" ] || [ -L "$DEST" ]; then
  echo "Error: destination already exists: $DEST; use a fresh output directory" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"
mkdir "$DEST"
cp -R "$SOURCE/." "$DEST/"

awk -v version="${TAG#v}" '
  /^appVersion:/ { $0 = "appVersion: \"" version "\""; updated++ }
  { print }
  END {
    if (updated != 1) {
      print "Error: expected exactly one appVersion in Chart.yaml" > "/dev/stderr"
      exit 1
    }
  }
' "$SOURCE/Chart.yaml" > "$DEST/Chart.yaml"

# Limit the edit to the imported chart's controller image, not its kubectl image.
awk -v tag="$TAG" '
  /^[^[:space:]#]/ {
    controller = ($0 ~ /^systemUpgradeController:[[:space:]]*$/)
    image = 0
  }
  controller && /^  [^[:space:]#]/ {
    image = ($0 ~ /^  image:[[:space:]]*$/)
  }
  controller && image && /^    tag:/ {
    $0 = "    tag: \"" tag "\""
    updated++
  }
  { print }
  END {
    if (updated != 1) {
      print "Error: expected exactly one systemUpgradeController.image.tag in values.yaml" > "/dev/stderr"
      exit 1
    }
  }
' "$SOURCE/values.yaml" > "$DEST/values.yaml"

printf '%s\n' "$DEST"
