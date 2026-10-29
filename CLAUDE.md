# YTROAD — notes for Claude

YTROAD is a personal app for macOS (Apple Silicon + Intel) and Windows 10/11 for **Nevertheless_D**, who is not a programmer.
Talk to them in Korean, in plain words. Never ask them to use Terminal; give them a double-clickable result.
It replaced their older "YT Downloader" (deno + server.ts); YTROAD imports that app's tools and settings on first run.

## What it is

On macOS (since 1.2.0) `YTROAD.app` is a **real, visible Dock app**: `Contents/MacOS/YTROAD` is a universal (arm64 + x86_64)
Swift/AppKit shell (`app/shell/main.swift`) with a WKWebView window. It copies `Contents/Resources/ytroad-<arm64|x86_64>` (Go binaries)
to `~/Library/Application Support/YTROAD/bin/ytroad-engine`, runs it as its own child (app → supervisor → engine) on 127.0.0.1 with a token
(same port as last time, from `lastport`), and shows the UI in its window. The engine runs only while the app is open
(⌘Q stops everything; closing the window keeps the app in the Dock). The mac app can only be built on a Mac — see "Releasing an update".

- `engine/` — Go, stdlib only. Embeds `web/`.
  - `main.go` routes/settings (fmt, quality, parallel 1–10, default folder, notify, autoUpdate, appearance), folder picker, reveal
  - `tools.go` first-run installer: yt-dlp (GitHub), ffmpeg/ffprobe (martin-riedl), deno (GitHub; yt-dlp needs it for YouTube JS challenges).
    Copies from the old `Application Support/YTDownloader/bin` when present. yt-dlp self-updates (`-U`) daily and after a 403/bot failure.
  - `jobs.go` oEmbed preview, download queue (max 10 parallel), yt-dlp progress-template parsing, Korean error messages
  - `update.go` app auto-update from `releases/latest.json` in this repo (raw.githubusercontent.com), sha256 check,
    swaps the .app (backup in Support/backup for rollback), restart keeps port+token so the open window reloads.
  - `proc_unix.go` / `proc_windows.go` — OS helpers (process groups, hidden consoles, open/reveal, notifications, message box).
- `engine/web/` — `index.html`, `css/app.css`, `js/art.js` (SVG illustrations), `js/app.js` (all UI logic), fonts.

## Windows

One `YTROAD.exe` (GUI subsystem, icon/manifest from go-winres) is both launcher and engine (`launcher_windows.go`):
double-click → copies itself to `%LOCALAPPDATA%\YTROAD\bin\ytroad-engine.exe`, starts it with `--engine` (hidden), opens an
Edge/Chrome `--app` window. Because the user's YTROAD.exe is not running, the updater can replace it in place
(backup in `%LOCALAPPDATA%\YTROAD\backup`). Feed: `releases/latest-win.json` → `YTROAD-win-<v>.zip` (YTROAD.exe + version.txt).
Folder picking is in-app (`/api/fs/list`, `/api/fs/mkdir`) because the native dialog hides behind the app window.
Tools: yt-dlp.exe, ffmpeg+ffprobe from one gyan.dev essentials zip (fallback yt-dlp FFmpeg-Builds), deno.
UI text switches Finder→탐색기, ⌘→Ctrl, macOS→Windows from `settings.os`.

Design: YouTube red (#ff0033 → #ff5a36 gradient), white / near-black (#0e0e10) themes, Pretendard + Unbounded (brand).
Format colours: MP4 red, MP3 violet #8a6cff, M4A cyan, WAV green. Footer must say `made by. Nevertheless_D`.

## macOS 27: Background App Activity (why the mac app is a Dock app since 1.2.0)

macOS 27 terminates processes that keep running after their app quits with no visible presence ("Background App Activity",
System Settings › General › Login Items & Extensions), unless the app that launched them is in the Dock. Up to 1.1.0 the bash launcher
started the engine with nohup and exited, which macOS 27 kills. **Never reintroduce hidden helpers, nohup engines or SIGTERM-ignoring.**
- Shell env to the supervisor: `YTROAD_SUPERVISOR=1 YTROAD_SHELL=native YTROAD_VIA=native YTROAD_TOKEN YTROAD_SUPPORT YTROAD_APP YTROAD_PARENT_PID YTROAD_PORT(=lastport) YTROAD_PORT_WAIT=1`.
- `engine/supervise.go` (mac only; `supervise_windows.go` is a stub): obeys SIGTERM/INT/HUP, stops when the app (PARENT_PID) disappears,
  restarts the engine only after a real crash (same port; max 5 in 5 min). Engine exit 0 → supervisor exits → app quits.
- Engine: no idle shutdown in native mode; `watchParent`; `watchSignals` cancels downloads (kills yt-dlp/ffmpeg process groups) before exiting.
  `/api/ping` and `/api/poll` include `"shell"`.
- Update restart (`update.go`): native → write `Support/restart-requested` and exit; the shell sees the marker and relaunches itself with
  `NSWorkspace.openApplication` (env without YTROAD_RESTART, plus `YTROAD_RELAUNCHED=1`). Updating from ≤1.1.0: the old updater runs
  `Contents/MacOS/YTROAD` directly with `YTROAD_RESTART=1 YTROAD_PORT=<old>` → the shell relaunches itself properly and reuses that port,
  so the old Chrome window reconnects and shows "YTROAD가 새 모습으로 바뀌었어요" (`IS_SHELL` = UA contains `YTROADShell`).
- `launcher.log` in Support records shell (`[앱 pid]`, incl. "화면 열림") and supervisor steps (ask the user to drag it into the chat when debugging).
`signal.Ignore` can NOT be undone by `signal.Reset`. Never use `pkill -f ytroad-engine` in tests; use `pgrep -x`/`pkill -x`.

## Releasing an update (this is how the user's app updates itself)

macOS (needs Swift + codesign, built on a real Mac by GitHub Actions: `.github/workflows/build-mac.yml` → `scripts/ci-mac.sh` →
`scripts/build-mac.sh` + `scripts/smoke-mac.sh`; the session proxy blocks the Actions API, so results are read via git):
1. Make the change. Bump `VERSION` (semver). Commit and push to `main` (changes under app/, engine/, VERSION, the mac guide png or the CI scripts trigger the build).
2. Wait 5–10 min, `git fetch origin +refs/heads/ci-results:refs/remotes/origin/ci-results`, read `status.json`, `results.txt`, `build.log`, `smoke.log`,
   screenshots (`git show origin/ci-results:results.txt`). It must match the current VERSION/commit with build=ok smoke=ok.
3. `scripts/release.sh --notes "바뀐 점 1" "바뀐 점 2"` (Korean, user-facing) → verifies the CI result + sha256, writes `releases/YTROAD-<v>.zip`,
   `releases/latest.json`, `dist/YTROAD.zip` (first install: folder with the app + usage guide). Keeps the three newest mac zips.
4. Commit and push. Installed apps pick it up within 6 hours or via the version badge.
Windows: `scripts/build.sh --notes …` builds `YTROAD.exe` → `releases/YTROAD-win-<v>.zip` + `latest-win.json` and `dist/YTROAD-Windows.zip`
(needs go-winres: `go install github.com/tc-hib/go-winres@latest`). Keep the three newest Windows zips.
Never publish a feed whose zip is missing or whose `version.txt` differs — the app verifies both. Don't delete the `ci-results` branch.
Commit as `Nevertheless_D <NeverthelessD@users.noreply.github.com>`.

## Testing without a Mac

`go build -o /tmp/ytroad ./engine`, run with `YTROAD_SUPPORT`, `YTROAD_TOKEN`, `YTROAD_PORT`, `YTROAD_TEST_CHOOSE` (fake folder picker),
`YTROAD_FFMPEG_DIR` (Linux: dir with ffmpeg/ffprobe to copy), `YTROAD_OLD_SUPPORT` (fake old YT Downloader folder),
`YTROAD_APP` (fake .app path), `YTROAD_UPDATE_URL` (local latest.json).
YouTube blocks video downloads from the cloud container (HTTP 403), so test the pipeline with yt-dlp's generic extractor against
a local throttled HTTP server serving an mp4 (`http://127.0.0.1:PORT/clip.mp4`). oEmbed previews do work from the container.
Drive the UI with Playwright (Chromium is preinstalled).
Windows: `apt-get install wine64 wine` works; run the real `YTROAD.exe` under Wine (Xvfb :99, `LC_ALL=C.UTF-8` for Korean paths) —
launcher, tool setup, downloads, update/restart/rollback all run there. `YTROAD_FAKE_OS=windows` makes the Linux engine show Windows UI text.

The Mac CI smoke test covers launch, Dock reopen, crash restart, ⌘Q, quit button, app force-kill, update relaunch (incl. from ≤1.1.0), Intel via Rosetta when available.
Things only the user's Mac can confirm: real YouTube downloads, osascript folder picker, notifications, Finder reveal, Gatekeeper prompts, macOS 27 behaviour.
Only real Windows: SmartScreen, Edge app window, Explorer /select, tray notification, antivirus behaviour.

## User-facing text

Never mention GitHub (or the repo name) in anything the user sees — app screens, update messages, usage-guide images.
Say "새 버전이 나오면 앱이 알아서 받아요" instead. The user asked for this for all of their apps.

- **2026-10-29 내려받기용 zip (사용자 요청)**: 저장소 `download/` 폴더에 `YTROAD.zip` · `YTROAD-Windows.zip`(앱 + 사용 가이드)를 올리고, README 맨 위 '앱 받기' 링크(`raw/main/download/YTROAD.zip`)로 바로 받게 했어요. 예전 버전 zip 은 지웠어요 (저장소 `releases/` 에도 업데이트용 최신판만).
- **GitHub 에는 늘 최신판 zip 1개만**: 업데이트를 내보낼 때마다 `scripts/publish-download.sh` 가 앱 + 가이드 zip 을 `download/` 에 정해진 이름으로 바꿔 넣고 다른 파일은 지워요(release.sh 가 자동으로 불러요. GitHub 릴리스 페이지는 이 작업 환경에서 만들 수 없어 저장소 폴더를 써요). `releases/` 의 업데이트용 zip 도 현재 버전만 남겨요 — 설치된 앱은 latest.json 이 가리키는 zip 만 받고, '되돌리기'는 맥 안의 백업을 써서 예전 zip 이 필요 없어요.
