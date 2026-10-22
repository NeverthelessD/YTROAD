#!/bin/bash
# YTROAD 맥 빌드 (맥에서만 돌아요 — 맥 빌드 서버 scripts/ci-mac.sh 가 불러요)
#   Go 엔진(darwin/arm64 · amd64) + Swift 앱 껍데기(app/shell/main.swift, Apple Silicon · Intel 겸용) → dist/YTROAD.app
#   → 앱 전체에 임시 서명(ad-hoc) → dist/YTROAD-<버전>.zip (업데이트용) · dist/YTROAD.zip (처음 설치용, 사용 가이드 포함)
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="$(tr -d ' \n' < "$ROOT/VERSION")"
DIST="$ROOT/dist"
APP="$DIST/YTROAD.app"
rm -rf "$DIST"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

echo "▶ 엔진 테스트"
( cd "$ROOT/engine" && go test ./... )
for pair in "arm64:arm64" "amd64:x86_64"; do
  GA="${pair%%:*}"; MA="${pair##*:}"
  echo "▶ 엔진 빌드 (darwin/$GA, v$VERSION)"
  ( cd "$ROOT/engine" && CGO_ENABLED=0 GOOS=darwin GOARCH="$GA" \
      go build -trimpath -ldflags "-s -w -X main.build=$VERSION" -o "$APP/Contents/Resources/ytroad-$MA" . )
  chmod 755 "$APP/Contents/Resources/ytroad-$MA"
done

echo "▶ 앱 껍데기 빌드 (Swift, Apple Silicon + Intel)"
xcrun swiftc --version
for MA in arm64 x86_64; do
  xcrun swiftc -O -target "$MA-apple-macos11.0" \
    -o "$DIST/shell-$MA" "$ROOT/app/shell/main.swift" \
    -framework AppKit -framework WebKit
done
lipo -create "$DIST/shell-arm64" "$DIST/shell-x86_64" -output "$APP/Contents/MacOS/YTROAD"
rm -f "$DIST/shell-arm64" "$DIST/shell-x86_64"
lipo -info "$APP/Contents/MacOS/YTROAD"
chmod 755 "$APP/Contents/MacOS/YTROAD"

sed "s/__VERSION__/$VERSION/g" "$ROOT/app/Info.plist" > "$APP/Contents/Info.plist"
plutil -lint "$APP/Contents/Info.plist"
cp "$ROOT/app/AppIcon.icns" "$APP/Contents/Resources/AppIcon.icns"
printf '%s' "$VERSION" > "$APP/Contents/Resources/version.txt"
printf 'APPL????' > "$APP/Contents/PkgInfo"

echo "▶ 임시 서명 (ad-hoc) · 확인"
codesign --force --deep --sign - --timestamp=none "$APP"
codesign --verify --deep --strict --verbose=2 "$APP"
codesign -dv "$APP" 2>&1 | sed 's/^/  /'

echo "▶ 압축"
ditto -c -k --norsrc --keepParent "$APP" "$DIST/YTROAD-$VERSION.zip"
mkdir -p "$DIST/first/YTROAD"
ditto "$APP" "$DIST/first/YTROAD/YTROAD.app"
[ -f "$ROOT/docs/YTROAD_사용가이드.png" ] && cp "$ROOT/docs/YTROAD_사용가이드.png" "$DIST/first/YTROAD/"
ditto -c -k --norsrc --keepParent "$DIST/first/YTROAD" "$DIST/YTROAD.zip"

echo "▶ 업데이트 파일을 풀어도 서명이 그대로인지 확인 (앱의 업데이트 기능처럼 unzip)"
CHECK="$DIST/check"; rm -rf "$CHECK"; mkdir -p "$CHECK"
( cd "$CHECK" && unzip -q "$DIST/YTROAD-$VERSION.zip" )
codesign --verify --deep --strict "$CHECK/YTROAD.app"
[ "$(cat "$CHECK/YTROAD.app/Contents/Resources/version.txt")" = "$VERSION" ]
rm -rf "$CHECK"
echo "✔ dist/YTROAD-$VERSION.zip · dist/YTROAD.zip"
