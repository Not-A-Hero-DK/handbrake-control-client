#!/bin/bash
set -euo pipefail

PROJECT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
VERSION_FILE="$PROJECT_DIR/VERSION"
APP_DIR="$PROJECT_DIR/dist/HandBrake Control.app"
MACOS_DIR="$APP_DIR/Contents/MacOS"
RESOURCES_DIR="$APP_DIR/Contents/Resources"

if [[ ! -f "$VERSION_FILE" ]]; then
  echo "Missing VERSION file" >&2
  exit 1
fi
APP_VERSION="$(tr -d '[:space:]' < "$VERSION_FILE")"
if [[ ! "$APP_VERSION" =~ ^[0-9]+(\.[0-9]+){1,2}$ ]]; then
  echo "VERSION must use numeric semantic versioning, for example 0.1.0" >&2
  exit 1
fi

rm -rf "$APP_DIR"
mkdir -p "$MACOS_DIR" "$RESOURCES_DIR"

GOCACHE="${GOCACHE:-/private/tmp/handbrake-control-go-cache}" \
  go build -o "$MACOS_DIR/hb-agent" "$PROJECT_DIR/cmd/hb-agent"

CLANG_MODULE_CACHE_PATH="${CLANG_MODULE_CACHE_PATH:-/private/tmp/handbrake-control-swift-cache}" \
SWIFT_MODULECACHE_PATH="${SWIFT_MODULECACHE_PATH:-/private/tmp/handbrake-control-swift-module-cache}" \
  swiftc -parse-as-library -framework Cocoa \
  "$PROJECT_DIR/menu/HandBrakeControlMenu.swift" \
  -o "$MACOS_DIR/HandBrakeControl"

cp "$PROJECT_DIR/config/default-agent.json" "$RESOURCES_DIR/agent.json"

cat > "$APP_DIR/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleDevelopmentRegion</key><string>en</string>
  <key>CFBundleExecutable</key><string>HandBrakeControl</string>
  <key>CFBundleIdentifier</key><string>dk.notahero.handbrake-control</string>
  <key>CFBundleName</key><string>HandBrake Control</string>
  <key>CFBundlePackageType</key><string>APPL</string>
  <key>CFBundleShortVersionString</key><string>$APP_VERSION</string>
  <key>CFBundleVersion</key><string>1</string>
  <key>LSUIElement</key><true/>
</dict>
</plist>
PLIST

# Ad-hoc signing makes local builds verifiable. A release build will replace
# this with a Developer ID signature before notarization.
codesign --force --sign - "$MACOS_DIR/hb-agent"
codesign --force --sign - "$MACOS_DIR/HandBrakeControl"
codesign --force --sign - "$APP_DIR"

echo "Built: $APP_DIR"
