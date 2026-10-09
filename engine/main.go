// YTROAD — YouTube video & music downloader for macOS
// made by. Nevertheless_D
package main

import (
	"crypto/rand"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// build is the app version, set with -ldflags "-X main.build=1.2.3".
var build = "0.0.0-dev"

//go:embed web
var webFiles embed.FS

const maxParallelLimit = 10

type Settings struct {
	Appearance    string `json:"appearance"`    // system | light | dark
	DefaultFolder string `json:"defaultFolder"` // "" = ~/Downloads
	Fmt           string `json:"fmt"`           // mp4 | mp3 | wav | m4a
	Quality       string `json:"quality"`       // best | 2160 | 1440 | 1080 | 720 | 480 | 360
	Parallel      int    `json:"parallel"`      // 1..10
	Notify        bool   `json:"notify"`        // macOS 알림
	AutoUpdate    bool   `json:"autoUpdate"`
}

type App struct {
	support string
	token   string
	port    int
	binDir  string
	workDir string

	mu       sync.Mutex
	settings Settings

	tools *Tools
	jobs  *JobManager
	upd   *Updater

	lastSeen atomic.Int64
}

var app *App

func main() {
	log.SetFlags(log.LstdFlags)
	home, _ := os.UserHomeDir()
	support := os.Getenv("YTROAD_SUPPORT")
	if support == "" {
		support = defaultSupportDir(home, "YTROAD")
	}
	token := os.Getenv("YTROAD_TOKEN")
	if token == "" {
		token = randID(16)
	}

	app = &App{
		support: support,
		token:   token,
		binDir:  filepath.Join(support, "bin"),
		workDir: filepath.Join(support, "work"),
	}
	os.MkdirAll(app.binDir, 0o755)
	os.RemoveAll(app.workDir)
	os.MkdirAll(app.workDir, 0o755)
	if f, err := os.OpenFile(filepath.Join(support, "engine.log"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644); err == nil {
		log.SetOutput(io.MultiWriter(os.Stderr, f))
	}
	app.loadSettings()
	app.lastSeen.Store(time.Now().Unix())

	app.tools = NewTools(support, app.binDir)
	app.jobs = NewJobManager()
	app.upd = NewUpdater(support)

	ln := listen(os.Getenv("YTROAD_PORT"))
	app.port = ln.Addr().(*net.TCPAddr).Port
	writeFileAtomic(filepath.Join(support, "token"), []byte(token), 0o600)
	writeFileAtomic(filepath.Join(support, "port"), []byte(strconv.Itoa(app.port)), 0o644)
	log.Printf("YTROAD %s listening on %d", build, app.port)

	go app.idleWatcher()
	go app.upd.autoLoop()
	go app.tools.autoLoop()
	srv := &http.Server{Handler: hostGuard(app.routes())}
	log.Fatal(srv.Serve(ln))
}

func defaultSupportDir(home, name string) string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", name)
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), name)
	default:
		return filepath.Join(home, "."+strings.ToLower(name))
	}
}

func listen(pref string) net.Listener {
	if p, err := strconv.Atoi(pref); err == nil && p > 0 {
		for i := 0; i < 20; i++ {
			if ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p)); err == nil {
				return ln
			}
			time.Sleep(150 * time.Millisecond)
		}
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	return ln
}

// ---------------------------------------------------------------- lifecycle

// 창을 닫고 10분 동안 아무 요청이 없고, 받는 중인 영상도 없으면 스스로 종료합니다.
func (a *App) idleWatcher() {
	for range time.Tick(15 * time.Second) {
		idle := time.Now().Unix() - a.lastSeen.Load()
		if idle > 600 && !a.jobs.Busy() && !a.tools.Running() && !a.upd.Busy() {
			a.shutdown()
		}
	}
}

func (a *App) shutdown() {
	a.jobs.CancelAll()
	os.RemoveAll(a.workDir)
	if b, err := os.ReadFile(filepath.Join(a.support, "port")); err == nil && string(b) == strconv.Itoa(a.port) {
		os.Remove(filepath.Join(a.support, "port"))
	}
	os.Exit(0)
}

// ---------------------------------------------------------------- settings

func (a *App) settingsPath() string { return filepath.Join(a.support, "settings.json") }

func (a *App) loadSettings() {
	a.settings = Settings{Appearance: "system", Fmt: "mp4", Quality: "best", Parallel: maxParallelLimit, Notify: true, AutoUpdate: true}
	if b, err := os.ReadFile(a.settingsPath()); err == nil {
		_ = json.Unmarshal(b, &a.settings)
	} else {
		a.importOldSettings()
	}
	a.fixSettings()
}

// 예전 "YT Downloader"를 쓰던 분이면 저장 폴더·형식 설정을 이어받습니다.
func (a *App) importOldSettings() {
	b, err := os.ReadFile(filepath.Join(oldSupportDir(), "settings.json"))
	if err != nil {
		return
	}
	var old struct {
		OutDir  string `json:"outDir"`
		Fmt     string `json:"fmt"`
		Quality string `json:"quality"`
	}
	if json.Unmarshal(b, &old) != nil {
		return
	}
	if old.OutDir != "" && old.OutDir != downloadsFolder() {
		if st, err := os.Stat(old.OutDir); err == nil && st.IsDir() {
			a.settings.DefaultFolder = old.OutDir
		}
	}
	if validFmt(old.Fmt) {
		a.settings.Fmt = old.Fmt
	}
	if validQuality(old.Quality) {
		a.settings.Quality = old.Quality
	}
}

func (a *App) fixSettings() {
	s := &a.settings
	if s.Appearance != "light" && s.Appearance != "dark" {
		s.Appearance = "system"
	}
	if !validFmt(s.Fmt) {
		s.Fmt = "mp4"
	}
	if !validQuality(s.Quality) {
		s.Quality = "best"
	}
	if s.Parallel < 1 || s.Parallel > maxParallelLimit {
		s.Parallel = maxParallelLimit
	}
}

func validFmt(f string) bool { return f == "mp4" || f == "mp3" || f == "wav" || f == "m4a" }
func validQuality(q string) bool {
	switch q {
	case "best", "2160", "1440", "1080", "720", "480", "360":
		return true
	}
	return false
}

func (a *App) saveSettings() {
	b, _ := json.MarshalIndent(a.settings, "", "  ")
	writeFileAtomic(a.settingsPath(), b, 0o644)
}

func (a *App) Settings() Settings {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.settings
}

func downloadsFolder() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "Downloads")
}

func (a *App) resolvedDefaultFolder() string {
	if f := a.Settings().DefaultFolder; f != "" {
		if st, err := os.Stat(f); err == nil && st.IsDir() {
			return f
		}
	}
	return downloadsFolder()
}

// ---------------------------------------------------------------- routes

func (a *App) routes() http.Handler {
	mux := http.NewServeMux()
	sub, _ := fs.Sub(webFiles, "web")
	static := http.FileServer(http.FS(sub))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			if r.URL.Query().Get("t") != a.token {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusForbidden)
				io.WriteString(w, "<meta charset=utf-8><body style='font:15px -apple-system;padding:40px'>YTROAD 앱을 다시 실행해 주세요.</body>")
				return
			}
			b, _ := webFiles.ReadFile("web/index.html")
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store")
			w.Write(b)
			return
		}
		if strings.HasSuffix(r.URL.Path, ".js") || strings.HasSuffix(r.URL.Path, ".css") {
			w.Header().Set("Cache-Control", "no-cache")
		}
		static.ServeHTTP(w, r)
	})

	api := func(path string, h http.HandlerFunc) {
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("x-token") != a.token {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			a.lastSeen.Store(time.Now().Unix())
			h(w, r)
		})
	}

	api("/api/ping", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true, "build": build})
	})
	api("/api/quit", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{"ok": true})
		go func() { time.Sleep(300 * time.Millisecond); a.shutdown() }()
	})
	api("/api/poll", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"build":  build,
			"jobs":   a.jobs.Snapshot(),
			"tools":  a.tools.State(),
			"update": a.upd.State(),
		})
	})
	api("/api/settings", a.handleSettings)
	api("/api/choose-folder", a.handleChooseFolder)
	api("/api/open-folder", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			Path string `json:"path"`
		}
		readJSON(r, &in)
		if in.Path == "" {
			in.Path = a.resolvedDefaultFolder()
		}
		os.MkdirAll(in.Path, 0o755)
		openPath(in.Path)
		writeJSON(w, map[string]any{"ok": true})
	})

	api("/api/preview", a.handlePreview)
	api("/api/jobs", a.handleJobsAdd)
	api("/api/job", a.handleJobAction)
	api("/api/jobs/clear", func(w http.ResponseWriter, r *http.Request) {
		a.jobs.ClearFinished()
		writeJSON(w, map[string]any{"ok": true})
	})
	api("/api/jobs/cancel-all", func(w http.ResponseWriter, r *http.Request) {
		a.jobs.CancelAll()
		writeJSON(w, map[string]any{"ok": true})
	})

	api("/api/tools/install", func(w http.ResponseWriter, r *http.Request) {
		a.tools.Start()
		writeJSON(w, map[string]any{"ok": true})
	})
	api("/api/tools/update", func(w http.ResponseWriter, r *http.Request) {
		go a.tools.UpdateEngine(true)
		writeJSON(w, map[string]any{"ok": true})
	})

	api("/api/update/state", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, a.upd.State()) })
	api("/api/update/check", func(w http.ResponseWriter, r *http.Request) {
		a.upd.Check(true)
		writeJSON(w, a.upd.State())
	})
	api("/api/update/install", func(w http.ResponseWriter, r *http.Request) {
		go a.upd.Install()
		writeJSON(w, map[string]any{"ok": true})
	})
	api("/api/update/rollback", func(w http.ResponseWriter, r *http.Request) {
		if err := a.upd.Rollback(); err != nil {
			jsonError(w, 500, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	api("/api/update/restart", func(w http.ResponseWriter, r *http.Request) {
		if err := a.upd.Restart(a.port, a.token); err != nil {
			jsonError(w, 500, err.Error())
			return
		}
		writeJSON(w, map[string]any{"ok": true})
	})
	return mux
}

// Only answer requests addressed to 127.0.0.1 / localhost (DNS rebinding protection).
func hostGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := r.Host
		if h, _, err := net.SplitHostPort(host); err == nil {
			host = h
		}
		if host != "127.0.0.1" && host != "localhost" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) handleSettings(w http.ResponseWriter, r *http.Request) {
	a.mu.Lock()
	if r.Method == http.MethodPost {
		var in map[string]json.RawMessage
		if err := readJSON(r, &in); err != nil {
			a.mu.Unlock()
			jsonError(w, 400, "bad request")
			return
		}
		for k, v := range in {
			switch k {
			case "appearance":
				json.Unmarshal(v, &a.settings.Appearance)
			case "defaultFolder":
				json.Unmarshal(v, &a.settings.DefaultFolder)
			case "fmt":
				json.Unmarshal(v, &a.settings.Fmt)
			case "quality":
				json.Unmarshal(v, &a.settings.Quality)
			case "parallel":
				json.Unmarshal(v, &a.settings.Parallel)
			case "notify":
				json.Unmarshal(v, &a.settings.Notify)
			case "autoUpdate":
				json.Unmarshal(v, &a.settings.AutoUpdate)
			}
		}
		a.fixSettings()
		a.saveSettings()
	}
	s := a.settings
	a.mu.Unlock()
	if r.Method == http.MethodPost {
		a.jobs.pump() // 동시 다운로드 개수를 늘렸다면 바로 다음 영상을 시작
	}
	home, _ := os.UserHomeDir()
	writeJSON(w, map[string]any{
		"appearance":            s.Appearance,
		"defaultFolder":         s.DefaultFolder,
		"resolvedDefaultFolder": a.resolvedDefaultFolder(),
		"downloadsFolder":       downloadsFolder(),
		"fmt":                   s.Fmt,
		"quality":               s.Quality,
		"parallel":              s.Parallel,
		"maxParallel":           maxParallelLimit,
		"notify":                s.Notify,
		"autoUpdate":            s.AutoUpdate,
		"home":                  home,
		"build":                 build,
		"platform":              runtime.GOOS + "/" + runtime.GOARCH,
	})
}

// ---------------------------------------------------------------- OS helpers

func (a *App) handleChooseFolder(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Start  string `json:"start"`
		Prompt string `json:"prompt"`
	}
	readJSON(r, &in)
	if t := os.Getenv("YTROAD_TEST_CHOOSE"); t != "" {
		os.MkdirAll(t, 0o755)
		writeJSON(w, map[string]any{"ok": true, "path": t})
		return
	}
	if runtime.GOOS != "darwin" {
		jsonError(w, 501, "macOS에서만 지원해요")
		return
	}
	start := in.Start
	if st, err := os.Stat(start); err != nil || !st.IsDir() {
		start = downloadsFolder()
	}
	prompt := in.Prompt
	if prompt == "" {
		prompt = "받은 영상·음악을 저장할 폴더를 선택하세요"
	}
	script := []string{
		"on run argv",
		"activate",
		"try",
		"set f to choose folder with prompt (item 2 of argv) default location (POSIX file (item 1 of argv))",
		"return POSIX path of f",
		"on error number -128",
		"return \"\"",
		"end try",
		"end run",
	}
	args := []string{}
	for _, l := range script {
		args = append(args, "-e", l)
	}
	args = append(args, start, prompt)
	out, err := exec.Command("/usr/bin/osascript", args...).Output()
	if err != nil {
		jsonError(w, 500, "폴더 선택 창을 열지 못했어요")
		return
	}
	p := strings.TrimSpace(string(out))
	if p != "/" {
		p = strings.TrimSuffix(p, "/")
	}
	writeJSON(w, map[string]any{"ok": true, "path": p})
}

func openPath(p string) {
	if runtime.GOOS == "darwin" {
		exec.Command("/usr/bin/open", p).Start()
	}
}

func revealPath(p string) {
	if runtime.GOOS == "darwin" {
		exec.Command("/usr/bin/open", "-R", p).Start()
	}
}

func notify(title, msg string) {
	if runtime.GOOS != "darwin" || !app.Settings().Notify {
		return
	}
	exec.Command("/usr/bin/osascript", "-e", "on run argv", "-e",
		`display notification (item 2 of argv) with title (item 1 of argv) sound name "Glass"`, "-e", "end run", title, msg).Start()
}

// ---------------------------------------------------------------- utils

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	json.NewEncoder(w).Encode(v)
}

func jsonError(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]any{"ok": false, "error": msg})
}

func readJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(v)
}

func randID(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func writeFileAtomic(path string, b []byte, perm os.FileMode) {
	tmp := path + ".tmp"
	if os.WriteFile(tmp, b, perm) == nil {
		os.Rename(tmp, path)
	}
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// uniquePath returns dir/name, or "name (2).ext" … when that file already exists.
func uniquePath(dir, name string) string {
	ext := filepath.Ext(name)
	stem := strings.TrimSuffix(name, ext)
	p := filepath.Join(dir, name)
	for n := 2; ; n++ {
		if _, err := os.Lstat(p); errors.Is(err, os.ErrNotExist) {
			return p
		}
		p = filepath.Join(dir, fmt.Sprintf("%s (%d)%s", stem, n, ext))
	}
}

func lastLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "\n"); i >= 0 {
		s = s[i+1:]
	}
	if len(s) > 200 {
		s = s[:200]
	}
	return s
}

func init() {
	mime.AddExtensionType(".js", "text/javascript")
	mime.AddExtensionType(".woff2", "font/woff2")
	mime.AddExtensionType(".svg", "image/svg+xml")
}
