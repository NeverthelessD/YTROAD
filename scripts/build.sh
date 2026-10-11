#!/bin/bash
# YTROAD 빌드 스크립트 — 맥(YTROAD.app)과 Windows(YTROAD.exe)를 함께 만들고
# 업데이트용 zip 과 releases/latest.json (맥) · releases/latest-win.json (Windows) 을 씁니다.
#
#   scripts/build.sh                 # VERSION 파일의 버전으로 빌드
#   scripts/build.sh --notes "고친 점 1" "고친 점 2"
#
# 필요: Go 1.24+, python3, go-winres (go install github.com/tc-hib/go-winres@latest)
#       Linux/macOS 어디서든 크로스 빌드됩니다.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
VERSION="$(tr -d ' \n' < "$ROOT/VERSION")"
NOTES=()
if [ "${1:-}" = "--notes" ]; then shift; NOTES=("$@"); fi

DIST="$ROOT/dist"
APP="$DIST/YTROAD.app"
rm -rf "$DIST"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources" "$ROOT/releases"

write_feed() { # zip-name feed-name
  local zip="$1" feed="$2"
  python3 "$ROOT/scripts/feed.py" "$ROOT/releases/$feed" "$VERSION" "$ROOT/releases/$zip" "${NOTES[@]+"${NOTES[@]}"}"
}

# ── macOS ─────────────────────────────────────────────
for pair in "arm64:arm64" "amd64:x86_64"; do
  GA="${pair%%:*}"; MA="${pair##*:}"
  echo "▶ 엔진 빌드 (darwin/$GA, v$VERSION)"
  ( cd "$ROOT/engine" && CGO_ENABLED=0 GOOS=darwin GOARCH="$GA" \
      go build -trimpath -ldflags "-s -w -X main.build=$VERSION" -o "$APP/Contents/Resources/ytroad-$MA" . )
  chmod 755 "$APP/Contents/Resources/ytroad-$MA"
done

cp "$ROOT/app/launcher.sh" "$APP/Contents/MacOS/YTROAD"
chmod 755 "$APP/Contents/MacOS/YTROAD"
sed "s/__VERSION__/$VERSION/g" "$ROOT/app/Info.plist" > "$APP/Contents/Info.plist"
cp "$ROOT/app/AppIcon.icns" "$APP/Contents/Resources/AppIcon.icns"
printf '%s' "$VERSION" > "$APP/Contents/Resources/version.txt"
printf 'APPL????' > "$APP/Contents/PkgInfo"

ZIP="YTROAD-$VERSION.zip"
python3 "$ROOT/scripts/zipapp.py" "$DIST" "YTROAD.app" "$ROOT/releases/$ZIP"
cp "$ROOT/releases/$ZIP" "$DIST/YTROAD.zip"
write_feed "$ZIP" latest.json

# ── Windows ─────────────────────────────────────────────
echo "▶ 엔진 빌드 (windows/amd64, v$VERSION)"
WINRES="${GO_WINRES:-$(go env GOPATH)/bin/go-winres}"
( cd "$ROOT/engine" && "$WINRES" simply --arch amd64 --manifest gui --icon "$ROOT/app/AppIcon.png" \
    --product-name YTROAD --file-description "YTROAD - YouTube downloader" \
    --product-version "$VERSION.0" --file-version "$VERSION.0" --original-filename YTROAD.exe \
    --copyright "made by. Nevertheless_D" --out rsrc )
WIN="$DIST/win"
mkdir -p "$WIN/stage" "$WIN/YTROAD"
( cd "$ROOT/engine" && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 \
    go build -trimpath -ldflags "-s -w -H windowsgui -X main.build=$VERSION" -o "$WIN/stage/YTROAD.exe" . )
rm -f "$ROOT/engine/rsrc_windows_amd64.syso"
printf '%s' "$VERSION" > "$WIN/stage/version.txt"

WZIP="YTROAD-win-$VERSION.zip"
python3 "$ROOT/scripts/zipdir.py" "$WIN/stage" "$ROOT/releases/$WZIP"
write_feed "$WZIP" latest-win.json

# 처음 설치용: YTROAD 폴더 안에 YTROAD.exe (+ 사용 가이드)
cp "$WIN/stage/YTROAD.exe" "$WIN/YTROAD/YTROAD.exe"
if [ -f "$ROOT/docs/YTROAD_Windows_사용가이드.png" ]; then cp "$ROOT/docs/YTROAD_Windows_사용가이드.png" "$WIN/YTROAD/"; fi
python3 "$ROOT/scripts/zipdir.py" "$WIN" "$DIST/YTROAD-Windows.zip" YTROAD
echo "✔ dist/YTROAD.zip (맥 · 처음 설치용), dist/YTROAD-Windows.zip (Windows · 처음 설치용)"
