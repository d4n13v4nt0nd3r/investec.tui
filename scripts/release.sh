#!/usr/bin/env bash
#
# Build the downloadable Mac and Windows packages for the Investec TUI.
#
#   scripts/release.sh v1.0.0                 build, sign, notarize, publish
#   scripts/release.sh v1.0.0 --skip-notarize skip the Apple notary service
#   scripts/release.sh v1.0.0 --no-publish    build only, do not touch GitHub
#
# See docs/releasing.md for the one-time Apple certificate setup.

set -euo pipefail

cd "$(dirname "$0")/.."

# ---------------------------------------------------------------- settings --

APP_NAME="InvestecTUI"          # deliberately no spaces: Terminal runs this path
BINARY_NAME="investec-tui"
BUNDLE_ID="com.beammoney.investec-tui"
NOTARY_PROFILE="investec-tui-notary"
DIST="dist"

# ------------------------------------------------------------------- input --

VERSION="${1:-}"
if [[ -z "$VERSION" ]]; then
  echo "usage: scripts/release.sh <version> [--skip-notarize] [--no-publish]" >&2
  echo "example: scripts/release.sh v1.0.0" >&2
  exit 64
fi
shift

SKIP_NOTARIZE=0
PUBLISH=1
for arg in "$@"; do
  case "$arg" in
    --skip-notarize) SKIP_NOTARIZE=1 ;;
    --no-publish)    PUBLISH=0 ;;
    *) echo "unknown option: $arg" >&2; exit 64 ;;
  esac
done

if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+ ]]; then
  echo "version must look like v1.0.0, got '$VERSION'" >&2
  exit 64
fi

step() { printf '\n==> %s\n' "$1"; }

# --------------------------------------------------------------- preflight --

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "this script must run on macOS: it needs codesign, hdiutil and notarytool" >&2
  exit 1
fi

step "Checking tests and vet"
go vet ./...
go test -race ./...

# The signing identity is looked up rather than hard-coded, so the script keeps
# working when the certificate is renewed.
IDENTITY="$(security find-identity -v -p codesigning \
  | sed -n 's/.*"\(Developer ID Application: .*\)"/\1/p' | head -n 1)"

if [[ -z "$IDENTITY" ]]; then
  cat >&2 <<'EOF'
No "Developer ID Application" certificate found in your keychain.

Follow Part A of docs/releasing.md to create one, or re-run with
--skip-notarize to produce an unsigned build for local testing only.
EOF
  if [[ "$SKIP_NOTARIZE" -eq 0 ]]; then
    exit 1
  fi
  echo "Continuing unsigned because --skip-notarize was given." >&2
fi

# A warning rather than a hard stop: notarytool itself gives a precise error if
# the profile really is missing, and the keychain item name is an Apple
# implementation detail that could change.
if [[ "$SKIP_NOTARIZE" -eq 0 ]] \
   && ! security find-generic-password -s "com.apple.gke.notary.tool" -a "$NOTARY_PROFILE" >/dev/null 2>&1; then
  echo "Warning: notarization profile '$NOTARY_PROFILE' was not found in your keychain." >&2
  echo "         If notarization fails, follow Parts B and C of docs/releasing.md." >&2
fi

# ------------------------------------------------------------------ build ---

step "Building $VERSION"
rm -rf "$DIST"
mkdir -p "$DIST/bin"

LDFLAGS="-s -w -X main.version=$VERSION"
build() {
  local goos="$1" goarch="$2" out="$3"
  echo "  $goos/$goarch"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags "$LDFLAGS" -o "$out" .
}

build darwin  arm64 "$DIST/bin/$BINARY_NAME-darwin-arm64"
build darwin  amd64 "$DIST/bin/$BINARY_NAME-darwin-amd64"
build windows amd64 "$DIST/$APP_NAME-$VERSION-Windows-x64.exe"
build windows arm64 "$DIST/$APP_NAME-$VERSION-Windows-ARM64.exe"

step "Merging the two Mac builds into one universal binary"
lipo -create -output "$DIST/bin/$BINARY_NAME-universal" \
  "$DIST/bin/$BINARY_NAME-darwin-arm64" \
  "$DIST/bin/$BINARY_NAME-darwin-amd64"
lipo -info "$DIST/bin/$BINARY_NAME-universal"

# ------------------------------------------------------------ app bundle ----

step "Assembling $APP_NAME.app"
APP="$DIST/stage/$APP_NAME.app"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"
cp "$DIST/bin/$BINARY_NAME-universal" "$APP/Contents/MacOS/$BINARY_NAME"
chmod +x "$APP/Contents/MacOS/$BINARY_NAME"

if [[ -f "packaging/macos/AppIcon.icns" ]]; then
  cp "packaging/macos/AppIcon.icns" "$APP/Contents/Resources/AppIcon.icns"
  ICON_ENTRY='	<key>CFBundleIconFile</key>
	<string>AppIcon</string>'
else
  ICON_ENTRY=''
fi

cat > "$APP/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleName</key>
	<string>$APP_NAME</string>
	<key>CFBundleDisplayName</key>
	<string>Investec TUI</string>
	<key>CFBundleIdentifier</key>
	<string>$BUNDLE_ID</string>
	<key>CFBundleExecutable</key>
	<string>$BINARY_NAME</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>${VERSION#v}</string>
	<key>CFBundleVersion</key>
	<string>${VERSION#v}</string>
	<key>LSMinimumSystemVersion</key>
	<string>11.0</string>
	<key>NSHighResolutionCapable</key>
	<true/>$ICON_ENTRY
</dict>
</plist>
EOF

if [[ -n "$IDENTITY" ]]; then
  step "Signing with: $IDENTITY"
  # The inner binary is signed first, then the bundle that seals it.
  codesign --force --options runtime --timestamp \
    --sign "$IDENTITY" "$APP/Contents/MacOS/$BINARY_NAME"
  codesign --force --options runtime --timestamp \
    --sign "$IDENTITY" "$APP"
  codesign --verify --deep --strict --verbose=2 "$APP"
fi

# ------------------------------------------------------------------- dmg ----

step "Building the disk image"
DMG="$DIST/$APP_NAME-$VERSION.dmg"
ln -s /Applications "$DIST/stage/Applications"
hdiutil create \
  -volname "$APP_NAME $VERSION" \
  -srcfolder "$DIST/stage" \
  -fs HFS+ -format UDZO -ov -quiet \
  "$DMG"

if [[ -n "$IDENTITY" ]]; then
  codesign --force --timestamp --sign "$IDENTITY" "$DMG"
fi

if [[ "$SKIP_NOTARIZE" -eq 1 ]]; then
  echo "Skipping notarization. This build will trigger a Gatekeeper warning."
else
  step "Notarizing (this usually takes a few minutes)"
  xcrun notarytool submit "$DMG" --keychain-profile "$NOTARY_PROFILE" --wait
  xcrun stapler staple "$DMG"
  xcrun stapler validate "$DMG"
  spctl --assess --type open --context context:primary-signature -vv "$DMG"
fi

# -------------------------------------------------------------- checksums ---

step "Writing checksums"
rm -rf "$DIST/stage" "$DIST/bin"
(cd "$DIST" && shasum -a 256 *.dmg *.exe > SHA256SUMS.txt)
cat "$DIST/SHA256SUMS.txt"

# ---------------------------------------------------------------- publish ---

if [[ "$PUBLISH" -eq 0 ]]; then
  step "Done. Artifacts are in $DIST (not published)"
  exit 0
fi

if [[ "$SKIP_NOTARIZE" -eq 1 ]]; then
  echo "Refusing to publish a build that was not notarized." >&2
  exit 1
fi

step "Publishing the GitHub release"
gh release create "$VERSION" "$DIST"/* \
  --title "$APP_NAME $VERSION" \
  --notes "Download the .dmg on a Mac or the Windows .exe on a PC. See the README for setup."

step "Done"
