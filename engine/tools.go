package main

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// 다운로드에 필요한 도구들 (처음 실행할 때 앱 화면에서 한 번만 설치)
//
//	yt-dlp  — 유튜브에서 영상을 받는 엔진 (하루 한 번 스스로 업데이트)
//	ffmpeg  — 영상·소리 합치기, MP3/WAV/M4A 변환
//	ffprobe — 파일 분석
//	deno    — 유튜브 보안 확인을 푸는 데 yt-dlp가 사용하는 도구
type toolDef struct {
	Name   string // file name in bin (without .exe)
	Label  string
	Emoji  string
	Kind   string // raw | zip | copy
	URLs   []string
	Member string // file inside the zip
}

func exe(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func toolDefs() []toolDef {
	arch := "arm64"
	denoArch := "aarch64"
	if runtime.GOARCH == "amd64" {
		arch, denoArch = "amd64", "x86_64"
	}
	gh := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/"
	deno := "https://github.com/denoland/deno/releases/latest/download/"
	switch runtime.GOOS {
	case "darwin":
		mr := func(t string) []string {
			return []string{
				fmt.Sprintf("https://ffmpeg.martin-riedl.de/redirect/latest/macos/%s/release/%s.zip", arch, t),
				fmt.Sprintf("https://ffmpeg.martin-riedl.de/redirect/latest/macos/%s/snapshot/%s.zip", arch, t),
			}
		}
		return []toolDef{
			{"yt-dlp", "다운로드 엔진", "🚀", "raw", []string{gh + "yt-dlp_macos"}, ""},
			{"ffmpeg", "변환 도구", "🎛️", "zip", mr("ffmpeg"), "ffmpeg"},
			{"ffprobe", "분석 도구", "🔬", "zip", mr("ffprobe"), "ffprobe"},
			{"deno", "보안 확인 도구", "🛡️", "zip", []string{deno + "deno-" + denoArch + "-apple-darwin.zip"}, "deno"},
		}
	case "windows":
		ff := []string{"https://github.com/BtbN/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-win64-gpl.zip"}
		return []toolDef{
			{"yt-dlp", "다운로드 엔진", "🚀", "raw", []string{gh + "yt-dlp.exe"}, ""},
			{"ffmpeg", "변환 도구", "🎛️", "zip", ff, "ffmpeg.exe"},
			{"ffprobe", "분석 도구", "🔬", "zip", ff, "ffprobe.exe"},
			{"deno", "보안 확인 도구", "🛡️", "zip", []string{deno + "deno-x86_64-pc-windows-msvc.zip"}, "deno.exe"},
		}
	default: // linux — 개발·테스트용
		ffdir := os.Getenv("YTROAD_FFMPEG_DIR")
		if ffdir == "" {
			ffdir = "/usr/bin"
		}
		return []toolDef{
			{"yt-dlp", "다운로드 엔진", "🚀", "raw", []string{gh + "yt-dlp_linux"}, ""},
			{"ffmpeg", "변환 도구", "🎛️", "copy", []string{filepath.Join(ffdir, "ffmpeg")}, ""},
			{"ffprobe", "분석 도구", "🔬", "copy", []string{filepath.Join(ffdir, "ffprobe")}, ""},
			{"deno", "보안 확인 도구", "🛡️", "zip", []string{deno + "deno-x86_64-unknown-linux-gnu.zip"}, "deno"},
		}
	}
}

// 예전 "YT Downloader" 앱이 쓰던 폴더 (같은 도구를 이미 받아 두었으면 복사해서 씁니다)
func oldSupportDir() string {
	if p := os.Getenv("YTROAD_OLD_SUPPORT"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return defaultSupportDir(home, "YTDownloader")
}

type ToolItem struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Emoji string `json:"emoji"`
	State string `json:"state"` // wait | down | copy | done | error
	Done  int64  `json:"done"`
	Total int64  `json:"total"`
}

type Tools struct {
	mu        sync.Mutex
	support   string
	bin       string
	running   bool
	err       string
	items     []*ToolItem
	migrated  bool
	engineVer string
	engUpd    bool   // yt-dlp 업데이트 중
	engMsg    string // 마지막 엔진 업데이트 결과
	engLast   int64
}

func NewTools(support, bin string) *Tools {
	t := &Tools{support: support, bin: bin}
	t.resetItems()
	if t.Ready() {
		go t.refreshVersion()
	}
	if b, err := os.ReadFile(t.stampPath()); err == nil {
		fmt.Sscan(string(b), &t.engLast)
	}
	return t
}

func (t *Tools) Path(name string) string { return filepath.Join(t.bin, exe(name)) }
func (t *Tools) stampPath() string       { return filepath.Join(t.support, "engine-updated") }

func (t *Tools) resetItems() {
	t.items = nil
	for _, d := range toolDefs() {
		st := "wait"
		if isExecutable(t.Path(d.Name)) {
			st = "done"
		}
		t.items = append(t.items, &ToolItem{Name: d.Name, Label: d.Label, Emoji: d.Emoji, State: st})
	}
}

func isExecutable(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir() && st.Size() > 0
}

func (t *Tools) installed() bool {
	for _, d := range toolDefs() {
		if !isExecutable(t.Path(d.Name)) {
			return false
		}
	}
	return true
}

func (t *Tools) Ready() bool {
	t.mu.Lock()
	running := t.running
	t.mu.Unlock()
	return !running && t.installed()
}

func (t *Tools) Running() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.running || t.engUpd
}

func (t *Tools) EngineUpdating() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.engUpd
}

func (t *Tools) State() map[string]any {
	ready := t.Ready()
	t.mu.Lock()
	defer t.mu.Unlock()
	items := make([]ToolItem, len(t.items))
	for i, it := range t.items {
		items[i] = *it
	}
	return map[string]any{
		"ready":          ready,
		"running":        t.running,
		"error":          t.err,
		"items":          items,
		"migrated":       t.migrated,
		"engineVersion":  t.engineVer,
		"engineUpdating": t.engUpd,
		"engineMessage":  t.engMsg,
		"engineChecked":  t.engLast,
	}
}

func (t *Tools) Start() {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return
	}
	t.running, t.err = true, ""
	t.resetItems()
	t.mu.Unlock()
	go func() {
		err := t.install()
		t.mu.Lock()
		t.running = false
		if err != nil {
			t.err = err.Error()
			log.Printf("setup failed: %v", err)
		}
		t.mu.Unlock()
		if err == nil {
			t.refreshVersion()
			app.jobs.pump()
		}
	}()
}

func (t *Tools) item(name string) *ToolItem {
	for _, it := range t.items {
		if it.Name == name {
			return it
		}
	}
	return &ToolItem{}
}

func (t *Tools) set(name string, fn func(*ToolItem)) {
	t.mu.Lock()
	fn(t.item(name))
	t.mu.Unlock()
}

func (t *Tools) install() error {
	os.MkdirAll(t.bin, 0o755)
	tmp := filepath.Join(t.support, "setup-tmp")
	os.RemoveAll(tmp)
	os.MkdirAll(tmp, 0o755)
	defer os.RemoveAll(tmp)
	oldBin := filepath.Join(oldSupportDir(), "bin")
	fresh := false
	for _, d := range toolDefs() {
		dest := t.Path(d.Name)
		if isExecutable(dest) {
			t.set(d.Name, func(i *ToolItem) { i.State = "done" })
			continue
		}
		// 1) 예전 YT Downloader가 받아 둔 도구가 있으면 복사
		if old := filepath.Join(oldBin, exe(d.Name)); isExecutable(old) && d.Kind != "copy" {
			t.set(d.Name, func(i *ToolItem) { i.State = "copy" })
			if err := copyExecutable(old, dest); err == nil {
				t.mu.Lock()
				t.migrated = true
				t.mu.Unlock()
				t.set(d.Name, func(i *ToolItem) { i.State = "done" })
				continue
			}
		}
		// 2) 인터넷에서 받기
		t.set(d.Name, func(i *ToolItem) { i.State = "down"; i.Done, i.Total = 0, 0 })
		var lastErr error
		for _, u := range d.URLs {
			if lastErr = t.fetchTool(d, u, tmp, dest); lastErr == nil {
				break
			}
			log.Printf("tool %s from %s: %v", d.Name, u, lastErr)
		}
		if lastErr != nil {
			t.set(d.Name, func(i *ToolItem) { i.State = "error" })
			return fmt.Errorf("%s을(를) 받지 못했어요 — %v", d.Label, lastErr)
		}
		if d.Name == "yt-dlp" {
			fresh = true
		}
		t.set(d.Name, func(i *ToolItem) { i.State = "done" })
	}
	if fresh {
		os.WriteFile(t.stampPath(), []byte(fmt.Sprint(time.Now().Unix())), 0o644)
		t.mu.Lock()
		t.engLast = time.Now().Unix()
		t.mu.Unlock()
	}
	return nil
}

func (t *Tools) fetchTool(d toolDef, url, tmp, dest string) error {
	if d.Kind == "copy" {
		return copyExecutable(url, dest)
	}
	progress := func(done, total int64) {
		t.set(d.Name, func(i *ToolItem) { i.Done, i.Total = done, total })
	}
	if d.Kind == "raw" {
		part := filepath.Join(tmp, d.Name+".bin")
		if err := download(context.Background(), url, part, progress); err != nil {
			return err
		}
		return finishExecutable(part, dest)
	}
	zipPath := filepath.Join(tmp, d.Name+".zip")
	if err := download(context.Background(), url, zipPath, progress); err != nil {
		return err
	}
	out := filepath.Join(tmp, d.Name+".out")
	if err := extractZipFile(zipPath, d.Member, out); err != nil {
		return err
	}
	os.Remove(zipPath)
	return finishExecutable(out, dest)
}

// finishExecutable makes the file runnable and moves it into bin only when it is complete.
func finishExecutable(src, dest string) error {
	os.Chmod(src, 0o755)
	if runtime.GOOS == "darwin" {
		exec.Command("/usr/bin/xattr", "-c", src).Run()
	}
	return os.Rename(src, dest)
}

func copyExecutable(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	part := dest + ".part"
	out, err := os.OpenFile(part, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(part)
		return err
	}
	out.Close()
	return finishExecutable(part, dest)
}

// ---------------------------------------------------------------- 다운로드 엔진(yt-dlp) 업데이트

func (t *Tools) refreshVersion() {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	out, err := toolCmd(ctx, t.Path("yt-dlp"), "--version").Output()
	if err == nil {
		t.mu.Lock()
		t.engineVer = strings.TrimSpace(string(out))
		t.mu.Unlock()
	}
}

// autoLoop keeps yt-dlp fresh — YouTube changes often, so we update once a day.
func (t *Tools) autoLoop() {
	time.Sleep(5 * time.Second)
	for {
		t.UpdateEngine(false)
		time.Sleep(3 * time.Hour)
	}
}

func (t *Tools) UpdateEngine(force bool) {
	if !t.Ready() {
		return
	}
	t.mu.Lock()
	if t.engUpd || (!force && time.Now().Unix()-t.engLast < 24*3600) {
		t.mu.Unlock()
		return
	}
	t.engUpd, t.engMsg = true, ""
	t.mu.Unlock()
	// 받는 중인 영상이 있으면 끝날 때까지 기다립니다 (실행 중인 엔진 파일을 바꾸지 않도록)
	for app.jobs.Active() > 0 {
		time.Sleep(3 * time.Second)
	}
	before := t.engineVersion()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	out, err := toolCmd(ctx, t.Path("yt-dlp"), "-U").CombinedOutput()
	cancel()
	t.refreshVersion()
	after := t.engineVersion()
	msg := ""
	switch {
	case err != nil:
		msg = "업데이트를 확인하지 못했어요. 인터넷 연결을 확인해 주세요."
		log.Printf("yt-dlp -U: %v %s", err, lastLine(string(out)))
	case before != after && after != "":
		msg = "새 엔진(" + after + ")으로 업데이트했어요."
	default:
		msg = "이미 최신 엔진이에요."
	}
	t.mu.Lock()
	t.engUpd, t.engMsg = false, msg
	if err == nil {
		t.engLast = time.Now().Unix()
		os.WriteFile(t.stampPath(), []byte(fmt.Sprint(t.engLast)), 0o644)
	}
	t.mu.Unlock()
	app.jobs.pump()
}

// UpdateEngineAfterFailure updates yt-dlp after a failed download, at most once every 30 minutes.
func (t *Tools) UpdateEngineAfterFailure() {
	t.mu.Lock()
	recent := time.Now().Unix()-t.engLast < 30*60
	t.mu.Unlock()
	if !recent {
		t.UpdateEngine(true)
	}
}

func (t *Tools) engineVersion() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.engineVer
}

// toolCmd runs a bundled tool with bin first on PATH (yt-dlp finds ffmpeg and deno there).
func toolCmd(ctx context.Context, name string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = toolEnv()
	return cmd
}

func toolEnv() []string {
	sep := ":"
	sys := "/usr/bin:/bin:/usr/sbin:/sbin"
	if runtime.GOOS == "windows" {
		sep, sys = ";", os.Getenv("PATH")
	}
	env := []string{}
	for _, e := range os.Environ() {
		if !strings.HasPrefix(e, "PATH=") && !strings.HasPrefix(e, "DENO_") {
			env = append(env, e)
		}
	}
	return append(env,
		"PATH="+app.binDir+sep+sys,
		"DENO_DIR="+filepath.Join(app.support, "cache", "deno"),
		"DENO_NO_UPDATE_CHECK=1",
		"PYTHONIOENCODING=utf-8",
		"LC_ALL=en_US.UTF-8",
	)
}

// ---------------------------------------------------------------- helpers

func download(ctx context.Context, url, dest string, onProgress func(done, total int64)) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	req.Header.Set("User-Agent", "YTROAD/"+build)
	resp, err := (&http.Client{Timeout: 30 * time.Minute}).Do(req)
	if err != nil {
		return errors.New("인터넷 연결을 확인해 주세요")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("다운로드 실패 (HTTP %d)", resp.StatusCode)
	}
	part := dest + ".part"
	f, err := os.Create(part)
	if err != nil {
		return err
	}
	total := resp.ContentLength
	var done int64
	buf := make([]byte, 256<<10)
	last := time.Now()
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := f.Write(buf[:n]); werr != nil {
				f.Close()
				os.Remove(part)
				return werr
			}
			done += int64(n)
			if onProgress != nil && time.Since(last) > 150*time.Millisecond {
				onProgress(done, total)
				last = time.Now()
			}
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			os.Remove(part)
			return errors.New("다운로드가 중간에 끊겼어요. 인터넷 연결을 확인해 주세요")
		}
	}
	f.Close()
	if total > 0 && done != total {
		os.Remove(part)
		return errors.New("다운로드가 중간에 끊겼어요")
	}
	if onProgress != nil {
		onProgress(done, total)
	}
	return os.Rename(part, dest)
}

func extractZipFile(zipPath, name, dest string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return errors.New("받은 파일이 손상됐어요. 다시 시도해 주세요")
	}
	defer zr.Close()
	for _, f := range zr.File {
		if filepath.Base(f.Name) == name && !f.FileInfo().IsDir() {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
			if err != nil {
				return err
			}
			_, err = io.Copy(out, rc)
			out.Close()
			return err
		}
	}
	return fmt.Errorf("받은 파일 안에 %s가 없어요", name)
}

func sha256File(p string) string {
	f, err := os.Open(p)
	if err != nil {
		return ""
	}
	defer f.Close()
	h := sha256.New()
	io.Copy(h, f)
	return hex.EncodeToString(h.Sum(nil))
}
