#!/bin/bash
# YTROAD Windows 빌드 스크립트 — YTROAD.exe 를 만들고 업데이트용 zip 과 releases/latest-win.json 을 씁니다.
# (맥 앱은 1.2.0 부터 맥 빌드 서버가 만들어요: VERSION 커밋 · 푸시 → scripts/release.sh)
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
rm -rf "$DIST"
mkdir -p "$ROOT/releases"

write_feed() { # zip-name feed-name
  local zip="$1" feed="$2"
  python3 "$ROOT/scripts/feed.py" "$ROOT/releases/$feed" "$VERSION" "$ROOT/releases/$zip" "${NOTES[@]+"${NOTES[@]}"}"
}

# ── macOS: 1.2.0 부터 맥 앱(Swift 껍데기 · 앱 전체 서명)은 맥 빌드 서버에서 만들어요 → scripts/release.sh
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
echo "✔ dist/YTROAD-Windows.zip (Windows · 처음 설치용)"
