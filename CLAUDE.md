# YTROAD — notes for Claude

YTROAD is a personal macOS app (Apple Silicon first; Intel engine is bundled too) for **Nevertheless_D**, who is not a programmer.
Talk to them in Korean, in plain words. Never ask them to use Terminal; give them a double-clickable result.
It replaced their older "YT Downloader" (deno + server.ts); YTROAD imports that app's tools and settings on first run.

## What it is

A "script-type" app: `YTROAD.app/Contents/MacOS/YTROAD` is a bash launcher (`app/launcher.sh`). It copies
`Contents/Resources/ytroad-<arm64|x86_64>` (Go binaries) to `~/Library/Application Support/YTROAD/bin/`, starts it on
127.0.0.1 with a random token, and opens the UI in a Chrome/Edge/Brave `--app` window (or the default browser).

- `engine/` — Go, stdlib only. Embeds `web/`.
  - `main.go` routes/settings (fmt, quality, parallel 1–10, default folder, notify, autoUpdate, appearance), folder picker, reveal
  - `tools.go` first-run installer: yt-dlp (GitHub), ffmpeg/ffprobe (martin-riedl), deno (GitHub; yt-dlp needs it for YouTube JS challenges).
    Copies from the old `Application Support/YTDownloader/bin` when present. yt-dlp self-updates (`-U`) daily and after a 403/bot failure.
  - `jobs.go` oEmbed preview, download queue (max 10 parallel), yt-dlp progress-template parsing, Korean error messages
  - `update.go` app auto-update from `releases/latest.json` in this repo (raw.githubusercontent.com), sha256 check,
    swaps the .app (backup in Support/backup for rollback), restart keeps port+token so the open window reloads.
  - `proc_windows.go` exists so the engine builds for windows/amd64 (Windows version planned later).
- `engine/web/` — `index.html`, `css/app.css`, `js/art.js` (SVG illustrations), `js/app.js` (all UI logic), fonts.

Design: YouTube red (#ff0033 → #ff5a36 gradient), white / near-black (#0e0e10) themes, Pretendard + Unbounded (brand).
Format colours: MP4 red, MP3 violet #8a6cff, M4A cyan, WAV green. Footer must say `made by. Nevertheless_D`.

## Releasing an update (this is how the user's app updates itself)

1. Make the change. Bump `VERSION` (semver, e.g. 1.0.1).
2. `scripts/build.sh --notes "바뀐 점 1" "바뀐 점 2"` (Korean, user-facing). Writes `releases/YTROAD-<v>.zip` and `releases/latest.json`.
3. Commit everything (including the zip and latest.json) and push to `main`. Installed apps pick it up within 6 hours or via the version badge.
4. Keep only the three newest zips in `releases/`.
5. Also send the user `dist/YTROAD.zip` for a fresh install if they ask.

Never publish a `latest.json` whose zip is missing or whose `version.txt` differs — the app verifies both.
Commit as `Nevertheless_D <NeverthelessD@users.noreply.github.com>`.

## Testing without a Mac

`go build -o /tmp/ytroad ./engine`, run with `YTROAD_SUPPORT`, `YTROAD_TOKEN`, `YTROAD_PORT`, `YTROAD_TEST_CHOOSE` (fake folder picker),
`YTROAD_FFMPEG_DIR` (Linux: dir with ffmpeg/ffprobe to copy), `YTROAD_OLD_SUPPORT` (fake old YT Downloader folder),
`YTROAD_APP` (fake .app path), `YTROAD_UPDATE_URL` (local latest.json).
YouTube blocks video downloads from the cloud container (HTTP 403), so test the pipeline with yt-dlp's generic extractor against
a local throttled HTTP server serving an mp4 (`http://127.0.0.1:PORT/clip.mp4`). oEmbed previews do work from the container.
For an update test, make a fake `YTROAD.app` with a Linux engine named `ytroad-x86_64`, set `HOME` to a temp dir and run the launcher with bash.
Drive the UI with Playwright (Chromium is preinstalled).

Things only a real Mac can confirm: real YouTube downloads, osascript folder picker, notifications, Finder reveal, Chrome app window, Gatekeeper prompts.
