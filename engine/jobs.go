package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

// ---------------------------------------------------------------- 영상 정보 미리보기

type Preview struct {
	URL     string `json:"url"`
	OK      bool   `json:"ok"`
	Warn    string `json:"warn"`
	Title   string `json:"title"`
	Channel string `json:"channel"`
	Thumb   string `json:"thumb"`
	Kind    string `json:"kind"` // video | shorts | link | playlist | music
}

var videoIDRe = regexp.MustCompile(`(?:youtu\.be/|[?&]v=|/shorts/|/embed/|/live/)([A-Za-z0-9_-]{11})`)

func videoID(u string) string {
	if m := videoIDRe.FindStringSubmatch(u); m != nil {
		return m[1]
	}
	return ""
}

func isYouTube(host string) bool {
	host = strings.ToLower(host)
	return host == "youtu.be" || host == "youtube.com" || strings.HasSuffix(host, ".youtube.com")
}

var oembedBase = "https://www.youtube.com/oembed"

func preview(raw string) Preview {
	p := Preview{URL: raw, OK: true}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		p.OK, p.Warn, p.Title, p.Kind = false, "올바른 주소가 아니에요", raw, "link"
		return p
	}
	id := videoID(raw)
	if id != "" {
		p.Thumb = "https://i.ytimg.com/vi/" + id + "/mqdefault.jpg"
	}
	if !isYouTube(u.Host) {
		p.Kind, p.Title = "link", u.Host
		p.Warn = "유튜브가 아닌 링크예요. 받을 수 있는지 시도해 볼게요."
		return p
	}
	switch {
	case strings.Contains(u.Path, "/shorts/"):
		p.Kind = "shorts"
	case strings.HasPrefix(strings.ToLower(u.Host), "music."):
		p.Kind = "music"
	default:
		p.Kind = "video"
	}
	if id == "" && u.Query().Get("list") != "" {
		p.OK, p.Kind, p.Title = false, "playlist", "재생목록 주소예요"
		p.Warn = "재생목록은 아직 지원하지 않아요. 영상 주소를 하나씩 넣어 주세요."
		return p
	}
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(ctx, "GET", oembedBase+"?format=json&url="+url.QueryEscape(raw), nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 YTROAD/"+build)
	if resp, err := http.DefaultClient.Do(req); err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			var j struct {
				Title        string `json:"title"`
				AuthorName   string `json:"author_name"`
				ThumbnailURL string `json:"thumbnail_url"`
			}
			if json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&j) == nil && j.Title != "" {
				p.Title, p.Channel = j.Title, j.AuthorName
				if p.Thumb == "" {
					p.Thumb = j.ThumbnailURL
				}
				return p
			}
		}
		if resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 404 {
			p.Title = "영상 정보를 불러올 수 없어요"
			p.Warn = "비공개·삭제된 영상이거나 주소가 틀렸을 수 있어요. 그래도 시도해 볼 수 있어요."
			return p
		}
	}
	if id != "" {
		p.Title = "유튜브 영상 (" + id + ")"
	} else {
		p.Title = raw
	}
	p.Warn = "제목을 미리 불러오지 못했어요. 다운로드는 시도할 수 있어요."
	return p
}

func (a *App) handlePreview(w http.ResponseWriter, r *http.Request) {
	var in struct {
		URLs []string `json:"urls"`
	}
	readJSON(r, &in)
	if len(in.URLs) > 200 {
		in.URLs = in.URLs[:200]
	}
	out := make([]Preview, len(in.URLs))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i, u := range in.URLs {
		wg.Add(1)
		go func(i int, u string) {
			defer wg.Done()
			sem <- struct{}{}
			out[i] = preview(strings.TrimSpace(u))
			<-sem
		}(i, u)
	}
	wg.Wait()
	writeJSON(w, map[string]any{"items": out})
}

// ---------------------------------------------------------------- 다운로드 작업

type Job struct {
	ID         string  `json:"id"`
	URL        string  `json:"url"`
	Title      string  `json:"title"`
	Channel    string  `json:"channel"`
	Thumb      string  `json:"thumb"`
	Kind       string  `json:"kind"`
	Fmt        string  `json:"fmt"`
	Quality    string  `json:"quality"`
	OutDir     string  `json:"outDir"`
	State      string  `json:"state"` // queued | starting | downloading | processing | done | error | canceled
	Stage      string  `json:"stage"`
	Part       int     `json:"part"`
	Pct        float64 `json:"pct"`
	Downloaded int64   `json:"downloaded"`
	Total      int64   `json:"total"`
	Speed      float64 `json:"speed"`
	ETA        float64 `json:"eta"`
	File       string  `json:"file"`
	Size       int64   `json:"size"`
	Error      string  `json:"error"`
	AddedAt    int64   `json:"addedAt"`
	StartedAt  int64   `json:"startedAt"`
	FinishedAt int64   `json:"finishedAt"`

	cmd       *exec.Cmd
	cancel    context.CancelFunc
	lastFmt   string
	partBytes int64 // bytes of earlier parts (video + audio)
}

type JobManager struct {
	mu   sync.Mutex
	jobs []*Job
	seq  int
}

func NewJobManager() *JobManager { return &JobManager{} }

func isActive(s string) bool { return s == "starting" || s == "downloading" || s == "processing" }

func (m *JobManager) Snapshot() []Job {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Job, len(m.jobs))
	for i, j := range m.jobs {
		out[i] = *j
		out[i].cmd, out[i].cancel = nil, nil
	}
	return out
}

func (m *JobManager) Active() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, j := range m.jobs {
		if isActive(j.State) {
			n++
		}
	}
	return n
}

func (m *JobManager) Busy() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, j := range m.jobs {
		if isActive(j.State) || j.State == "queued" {
			return true
		}
	}
	return false
}

func (a *App) handleJobsAdd(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Items []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Channel string `json:"channel"`
			Thumb   string `json:"thumb"`
			Kind    string `json:"kind"`
			Fmt     string `json:"fmt"`
			Quality string `json:"quality"`
		} `json:"items"`
		OutDir string `json:"outDir"`
	}
	if err := readJSON(r, &in); err != nil {
		jsonError(w, 400, "bad request")
		return
	}
	outDir := in.OutDir
	if outDir == "" {
		outDir = a.resolvedDefaultFolder()
	}
	s := a.Settings()
	m := a.jobs
	m.mu.Lock()
	n := 0
	for _, it := range in.Items {
		u := strings.TrimSpace(it.URL)
		if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
			continue
		}
		f, q := it.Fmt, it.Quality
		if !validFmt(f) {
			f = s.Fmt
		}
		if !validQuality(q) {
			q = s.Quality
		}
		m.seq++
		m.jobs = append(m.jobs, &Job{
			ID: fmt.Sprintf("j%d%s", m.seq, randID(3)), URL: u, Title: it.Title, Channel: it.Channel,
			Thumb: it.Thumb, Kind: it.Kind, Fmt: f, Quality: q, OutDir: outDir,
			State: "queued", Stage: "⏳ 순서를 기다리는 중", AddedAt: time.Now().UnixMilli(),
		})
		n++
	}
	m.mu.Unlock()
	m.pump()
	writeJSON(w, map[string]any{"ok": true, "added": n})
}

func (a *App) handleJobAction(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID     string `json:"id"`
		Action string `json:"action"`
	}
	readJSON(r, &in)
	m := a.jobs
	m.mu.Lock()
	var job *Job
	idx := -1
	for i, j := range m.jobs {
		if j.ID == in.ID {
			job, idx = j, i
		}
	}
	if job == nil {
		m.mu.Unlock()
		jsonError(w, 404, "목록에 없는 항목이에요")
		return
	}
	switch in.Action {
	case "cancel":
		m.cancelLocked(job)
	case "retry":
		if job.State == "error" || job.State == "canceled" {
			job.State, job.Stage, job.Pct, job.Error = "queued", "⏳ 순서를 기다리는 중", 0, ""
			job.Downloaded, job.Total, job.Speed, job.ETA = 0, 0, 0, 0
		}
	case "remove":
		if !isActive(job.State) {
			m.jobs = append(m.jobs[:idx], m.jobs[idx+1:]...)
		}
	case "reveal":
		if job.File != "" {
			if fileExists(job.File) {
				revealPath(job.File)
			} else {
				m.mu.Unlock()
				jsonError(w, 404, "파일을 찾을 수 없어요. 옮기거나 지웠을 수 있어요.")
				return
			}
		}
	case "open":
		if job.File != "" && fileExists(job.File) {
			openPath(job.File)
		}
	case "top": // 대기 중인 항목을 맨 앞으로
		if job.State == "queued" {
			m.jobs = append(m.jobs[:idx], m.jobs[idx+1:]...)
			m.jobs = append([]*Job{job}, m.jobs...)
		}
	}
	m.mu.Unlock()
	m.pump()
	writeJSON(w, map[string]any{"ok": true})
}

func (m *JobManager) cancelLocked(j *Job) {
	switch {
	case j.State == "queued":
		j.State, j.Stage, j.FinishedAt = "canceled", "⏹️ 취소됨", time.Now().UnixMilli()
	case isActive(j.State):
		j.State, j.Stage, j.Speed, j.ETA, j.FinishedAt = "canceled", "⏹️ 취소됨", 0, 0, time.Now().UnixMilli()
		if j.cancel != nil {
			j.cancel()
		}
		if j.cmd != nil {
			killProcessGroup(j.cmd)
		}
	}
}

func (m *JobManager) CancelAll() {
	m.mu.Lock()
	for _, j := range m.jobs {
		m.cancelLocked(j)
	}
	m.mu.Unlock()
}

func (m *JobManager) ClearFinished() {
	m.mu.Lock()
	keep := m.jobs[:0]
	for _, j := range m.jobs {
		if isActive(j.State) || j.State == "queued" {
			keep = append(keep, j)
		}
	}
	m.jobs = keep
	m.mu.Unlock()
}

// pump starts queued jobs while there are free slots (동시 다운로드 최대 10개).
func (m *JobManager) pump() {
	if app == nil || app.tools == nil || !app.tools.Ready() || app.tools.EngineUpdating() {
		return
	}
	limit := app.Settings().Parallel
	m.mu.Lock()
	active := 0
	for _, j := range m.jobs {
		if isActive(j.State) {
			active++
		}
	}
	var start []*Job
	for _, j := range m.jobs {
		if active >= limit {
			break
		}
		if j.State == "queued" {
			j.State, j.Stage = "starting", "🔎 영상 정보를 확인하는 중"
			j.StartedAt = time.Now().UnixMilli()
			j.Pct, j.Downloaded, j.Total, j.Speed, j.ETA, j.Part, j.File, j.Size = 0, 0, 0, 0, 0, 0, "", 0
			j.lastFmt, j.partBytes = "", 0
			active++
			start = append(start, j)
		}
	}
	m.mu.Unlock()
	for _, j := range start {
		go m.runJob(j)
	}
}

func formatArgs(f, quality string) []string {
	if f == "mp4" {
		h := ""
		if quality != "best" {
			h = "[height<=" + quality + "]"
		}
		// QuickTime·iPhone에서 바로 재생되도록 H.264(avc1) + AAC를 먼저 고르고, 없으면 가장 좋은 것으로
		return []string{"-f", fmt.Sprintf("bv*[vcodec^=avc1]%s+ba[ext=m4a]/bv*%s+ba/b%s/b", h, h, h),
			"--merge-output-format", "mp4", "--remux-video", "mp4"}
	}
	a := []string{"-f", "ba/b", "-x", "--audio-format", f}
	switch f {
	case "mp3":
		a = append(a, "--audio-quality", "320K")
	case "m4a":
		a = append(a, "--audio-quality", "0")
	}
	return a
}

var ppLabel = map[string]string{
	"Merger":             "🧩 영상과 소리를 합치는 중",
	"VideoRemuxer":       "🧩 MP4로 정리하는 중",
	"VideoConvertor":     "🧩 MP4로 바꾸는 중",
	"ExtractAudio":       "🎚️ 오디오로 변환하는 중",
	"FFmpegExtractAudio": "🎚️ 오디오로 변환하는 중",
	"FixupM4a":           "🔧 파일을 다듬는 중",
	"FixupM3u8":          "🔧 파일을 다듬는 중",
	"FixupDuplicateMoov": "🔧 파일을 다듬는 중",
	"FixupStretched":     "🔧 파일을 다듬는 중",
	"EmbedThumbnail":     "🖼️ 표지 넣는 중",
	"Metadata":           "🏷️ 정보 넣는 중",
	"FFmpegMetadata":     "🏷️ 정보 넣는 중",
}

func num(s string) float64 {
	f, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return 0
	}
	return f
}

func (m *JobManager) runJob(j *Job) {
	defer m.pump()
	tools := app.tools
	tmp := filepath.Join(app.workDir, j.ID)
	os.RemoveAll(tmp)
	os.MkdirAll(tmp, 0o755)
	defer os.RemoveAll(tmp)
	result := filepath.Join(tmp, "result.txt")

	args := []string{
		"--no-playlist", "--newline", "--progress", "--no-colors", "--no-mtime", "--no-part",
		"--encoding", "utf-8",
		"-P", filepath.Join(tmp, "out"), "-P", "temp:" + filepath.Join(tmp, "work"),
		"-o", "%(title).150B.%(ext)s", "--windows-filenames",
		"--ffmpeg-location", tools.bin, "--cache-dir", filepath.Join(app.support, "cache", "yt-dlp"),
		"--print-to-file", "after_move:filepath", result,
		"--progress-template",
		"download:@@D|%(progress.downloaded_bytes)s|%(progress.total_bytes)s|%(progress.total_bytes_estimate)s|%(progress.speed)s|%(progress.eta)s|%(info.vcodec)s|%(info.acodec)s|%(info.format_id)s",
		"--progress-template", "postprocess:@@P|%(progress.status)s|%(progress.postprocessor)s",
	}
	args = append(args, formatArgs(j.Fmt, j.Quality)...)
	args = append(args, "--", j.URL)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := toolCmd(ctx, tools.Path("yt-dlp"), args...)
	setProcessGroup(cmd)
	cmd.Cancel = func() error { killProcessGroup(cmd); return nil }
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	m.mu.Lock()
	if j.State == "canceled" {
		m.mu.Unlock()
		return
	}
	j.cmd, j.cancel = cmd, cancel
	m.mu.Unlock()

	if err := cmd.Start(); err != nil {
		m.fail(j, "⚠️ 다운로드 엔진을 실행하지 못했어요. 설정에서 ‘엔진 다시 설치’를 해 보세요.")
		return
	}
	var errLines []string
	var emu sync.Mutex
	done := make(chan struct{})
	go func() {
		sc := bufio.NewScanner(stderr)
		sc.Buffer(make([]byte, 64<<10), 1<<20)
		for sc.Scan() {
			l := sc.Text()
			emu.Lock()
			errLines = append(errLines, l)
			if len(errLines) > 60 {
				errLines = errLines[1:]
			}
			emu.Unlock()
		}
		close(done)
	}()
	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 64<<10), 1<<20)
	sc.Split(scanLinesCR)
	for sc.Scan() {
		m.onLine(j, sc.Text())
	}
	<-done
	err := cmd.Wait()

	m.mu.Lock()
	j.cmd, j.cancel = nil, nil
	j.Speed, j.ETA = 0, 0
	canceled := j.State == "canceled"
	m.mu.Unlock()
	if canceled {
		return
	}

	file := ""
	if b, rerr := os.ReadFile(result); rerr == nil {
		lines := strings.Split(strings.TrimSpace(string(b)), "\n")
		file = strings.TrimSpace(lines[len(lines)-1])
	}
	if err != nil || file == "" || !fileExists(file) {
		emu.Lock()
		msg := friendlyError(errLines)
		raw := strings.Join(errLines, "\n")
		emu.Unlock()
		log.Printf("job %s failed: %v\n%s", j.ID, err, raw)
		m.fail(j, msg)
		if strings.Contains(msg, "🛡️") || strings.Contains(msg, "엔진") {
			go app.tools.UpdateEngineAfterFailure() // 유튜브가 바뀌었을 수 있어요 → 엔진 업데이트
		}
		return
	}
	m.mu.Lock()
	j.Stage = "📁 저장하는 중"
	outDir := j.OutDir
	m.mu.Unlock()
	dest, merr := moveInto(file, outDir)
	if merr != nil {
		m.fail(j, "📁 저장 폴더에 옮기지 못했어요: "+merr.Error())
		return
	}
	var size int64
	if st, err := os.Stat(dest); err == nil {
		size = st.Size()
	}
	m.mu.Lock()
	j.State, j.Stage, j.Pct, j.File, j.Size = "done", "✅ 완료", 100, dest, size
	j.FinishedAt = time.Now().UnixMilli()
	name := filepath.Base(dest)
	if j.Title == "" || strings.HasPrefix(j.Title, "유튜브 영상 (") || strings.HasPrefix(j.Title, "영상 정보를") || strings.HasPrefix(j.Title, "http") || j.Kind == "link" {
		j.Title = strings.TrimSuffix(name, filepath.Ext(name))
	}
	m.mu.Unlock()
	notify("✅ 다운로드 완료", name)
}

func (m *JobManager) fail(j *Job, msg string) {
	m.mu.Lock()
	if j.State != "canceled" {
		j.State, j.Stage, j.Error = "error", "⚠️ 실패", msg
		j.Speed, j.ETA, j.FinishedAt = 0, 0, time.Now().UnixMilli()
	}
	m.mu.Unlock()
}

func (m *JobManager) onLine(j *Job, l string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if j.State == "canceled" {
		return
	}
	switch {
	case strings.HasPrefix(l, "@@D|"):
		f := strings.Split(l, "|")
		for len(f) < 10 {
			f = append(f, "")
		}
		d, t, te, sp, eta, vc, ac, fid := f[1], f[2], f[3], f[4], f[5], f[6], f[7], f[8]
		if fid != j.lastFmt {
			if j.lastFmt != "" {
				j.partBytes += j.Downloaded
			}
			j.lastFmt = fid
			j.Part++
			j.Speed = 0
		}
		j.State = "downloading"
		j.Downloaded = int64(num(d))
		j.Total = int64(num(t))
		if j.Total == 0 {
			j.Total = int64(num(te))
		}
		j.ETA = num(eta)
		raw := num(sp)
		// 시작 직후 순간 속도가 튀지 않도록 부드럽게(지수 평균) 표시
		if j.Downloaded >= 256<<10 {
			if j.Speed > 0 {
				j.Speed = j.Speed*0.7 + raw*0.3
			} else {
				j.Speed = raw
			}
		}
		if j.Total > 0 {
			j.Pct = float64(j.Downloaded) * 100 / float64(j.Total)
			if j.Pct > 100 {
				j.Pct = 100
			}
		}
		hasV := vc != "" && vc != "none" && vc != "NA"
		hasA := ac != "" && ac != "none" && ac != "NA"
		switch {
		case hasV && !hasA:
			j.Stage = "🎞️ 영상 받는 중"
		case !hasV && hasA:
			j.Stage = "🔊 소리 받는 중"
		default:
			j.Stage = "⬇️ 받는 중"
		}
	case strings.HasPrefix(l, "@@P|"):
		f := strings.Split(l, "|")
		if len(f) >= 3 && f[1] == "started" && f[2] != "MoveFiles" {
			j.State, j.Speed, j.ETA, j.Pct = "processing", 0, 0, 100
			if s, ok := ppLabel[f[2]]; ok {
				j.Stage = s
			} else {
				j.Stage = "⚙️ 마무리하는 중"
			}
		}
	case strings.HasPrefix(l, "[Merger]"):
		j.State, j.Speed, j.ETA, j.Pct, j.Stage = "processing", 0, 0, 100, ppLabel["Merger"]
	case strings.HasPrefix(l, "[ExtractAudio]"):
		j.State, j.Speed, j.ETA, j.Pct, j.Stage = "processing", 0, 0, 100, ppLabel["ExtractAudio"]
	case strings.Contains(l, "has already been downloaded"):
		j.Pct = 100
	}
}

func scanLinesCR(data []byte, atEOF bool) (int, []byte, error) {
	for i, b := range data {
		if b == '\n' || b == '\r' {
			return i + 1, data[:i], nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), data, nil
	}
	return 0, nil, nil
}

// moveInto moves the finished file into the save folder ("제목 (2).mp4" when the name is taken).
func moveInto(src, dir string) (string, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("폴더를 만들 수 없어요 (%s)", filepath.Base(dir))
	}
	dest := uniquePath(dir, filepath.Base(src))
	if err := os.Rename(src, dest); err == nil {
		return dest, nil
	}
	// 외장 디스크 등 다른 디스크라면 복사
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	out, err := os.Create(dest)
	if err != nil {
		return "", fmt.Errorf("이 폴더에 저장할 권한이 없어요")
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(dest)
		return "", fmt.Errorf("저장 공간이 부족하거나 디스크를 쓸 수 없어요")
	}
	out.Close()
	os.Remove(src)
	return dest, nil
}

var errRules = []struct {
	re  *regexp.Regexp
	msg string
}{
	{regexp.MustCompile(`(?i)Private video`), "🔒 비공개 영상이라 받을 수 없어요."},
	{regexp.MustCompile(`(?i)Sign in to confirm your age|age[- ]restricted|inappropriate for some users`), "🔞 연령 제한 영상이라 로그인 없이는 받을 수 없어요."},
	{regexp.MustCompile(`(?i)members[- ]only|Join this channel`), "💎 채널 멤버십 전용 영상이에요."},
	{regexp.MustCompile(`(?i)Premieres in|This live event will begin|is upcoming`), "⏰ 아직 공개되지 않은 영상이에요. 공개된 뒤 다시 받아 주세요."},
	{regexp.MustCompile(`(?i)not a bot|HTTP Error 403|Forbidden|Sign in to confirm`), "🛡️ 유튜브가 잠시 다운로드를 막았어요. 잠시 뒤 🔁 다시 시도해 보세요. (엔진은 자동으로 업데이트돼요)"},
	{regexp.MustCompile(`(?i)Video unavailable|has been removed|no longer available|does not exist|This video is not available`), "🚫 삭제되었거나 볼 수 없는 영상이에요."},
	{regexp.MustCompile(`(?i)Unsupported URL|is not a valid URL`), "🔗 지원하지 않는 주소예요. 주소를 다시 확인해 주세요."},
	{regexp.MustCompile(`(?i)Requested format is not available`), "🎞️ 고른 화질이 없는 영상이에요. 다른 화질로 다시 시도해 보세요."},
	{regexp.MustCompile(`(?i)No space left|Disk full`), "💾 저장 공간이 부족해요."},
	{regexp.MustCompile(`(?i)getaddrinfo|Network is unreachable|timed out|Connection (reset|refused|aborted)|Temporary failure in name resolution|Unable to download webpage`), "📡 인터넷 연결을 확인해 주세요."},
	{regexp.MustCompile(`(?i)JavaScript runtime|n challenge|signature (extraction|function)|nsig`), "🧩 유튜브가 바뀌어서 엔진 업데이트가 필요해요. 자동으로 업데이트한 뒤 🔁 다시 시도해 주세요."},
}

func friendlyError(lines []string) string {
	all := strings.Join(lines, "\n")
	for _, r := range errRules {
		if r.re.MatchString(all) {
			return r.msg
		}
	}
	raw := ""
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.Contains(lines[i], "ERROR") {
			raw = lines[i]
			break
		}
	}
	if raw == "" && len(lines) > 0 {
		raw = lines[len(lines)-1]
	}
	if raw == "" {
		return "⚠️ 알 수 없는 이유로 받지 못했어요. 🔁 다시 시도해 보세요."
	}
	raw = regexp.MustCompile(`^ERROR:\s*`).ReplaceAllString(raw, "")
	raw = regexp.MustCompile(`^\[[^\]]+\]\s*[\w-]+:\s*`).ReplaceAllString(raw, "")
	if len([]rune(raw)) > 200 {
		raw = string([]rune(raw)[:200])
	}
	return "⚠️ " + raw
}
