#!/usr/bin/env bash
# Rebuild every app icon artifact from packaging/source/tui-zebra-logo.png.
#
# Only needed when the artwork or the icon geometry changes. The generated
# files are committed, so a normal build and release never runs this.
set -euo pipefail

cd "$(dirname "$0")/.."

step() { printf '\n==> %s\n' "$1"; }

if ! python3 -c 'import PIL' 2>/dev/null; then
  cat >&2 <<'EOF'
Pillow is required to build the icons.

    python3 -m pip install Pillow
EOF
  exit 1
fi

if ! command -v iconutil >/dev/null 2>&1; then
  echo "iconutil not found: the .icns can only be built on macOS" >&2
  exit 1
fi

step "Extracting the mark from the source artwork"
python3 packaging/tools/extract_mark.py

step "Rendering the macOS iconset, .icns and Windows .ico"
python3 packaging/tools/build_icons.py

step "Generating the Windows resource objects"
python3 packaging/tools/mksyso.py

# Linking a malformed .syso fails silently and just yields an iconless exe, so
# build a throwaway binary for each Windows target and read the icon back out.
step "Verifying the icon survives into a real Windows binary"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
for arch in amd64 arm64; do
  echo "  windows/$arch"
  GOOS=windows GOARCH="$arch" go build -o "$tmp/verify-$arch.exe" .
  python3 packaging/tools/verify_syso.py "$tmp/verify-$arch.exe"
done

step "Done. Review packaging/source/preview.png, then commit the results"
