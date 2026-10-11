// YTROAD — 화면 스크립트
// made by. Nevertheless_D
'use strict';

const T = new URLSearchParams(location.search).get('t') || '';
const $ = (sel, root = document) => root.querySelector(sel);
const $$ = (sel, root = document) => [...root.querySelectorAll(sel)];
const esc = (s) => String(s ?? '').replace(/[&<>"']/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }[c]));

async function api(path, body) {
  const init = { method: body === undefined ? 'GET' : 'POST', headers: { 'x-token': T } };
  if (body !== undefined) { init.body = JSON.stringify(body); init.headers['content-type'] = 'application/json'; }
  let r;
  try { r = await fetch(path, init); } catch { throw new Error('앱 엔진과 연결이 끊겼어요. YTROAD를 다시 실행해 주세요.'); }
  let j = {};
  try { j = await r.json(); } catch { /* ignore */ }
  if (!r.ok || j.ok === false) throw new Error(j.error || `요청에 실패했어요 (${r.status})`);
  return j;
}

const FMT = {
  mp4: { n: 'MP4', c: '#ff0033', d: '영상 + 소리' },
  mp3: { n: 'MP3', c: '#8a6cff', d: '음악 · 320kbps' },
  m4a: { n: 'M4A', c: '#1aa6c9', d: '원본 음질' },
  wav: { n: 'WAV', c: '#1fb985', d: '무손실' },
};
const QN = { best: '최고 화질', 2160: '4K', 1440: '1440p', 1080: '1080p', 720: '720p', 480: '480p', 360: '360p' };
const ACTIVE = ['starting', 'downloading', 'processing'];

const S = {
  settings: null, folder: '', fmt: 'mp4', quality: 'best',
  previews: [], jobs: [], tools: null, update: null, filter: 'all', step: 1,
};

// ════════════════════════════════════════════════ 작은 도우미
let toastTimer;
function toast(msg, ms = 3200) {
  const t = $('#toast');
  t.textContent = msg; t.hidden = false;
  t.style.animation = 'none'; void t.offsetWidth; t.style.animation = '';
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => (t.hidden = true), ms);
}
function mb(b) {
  if (!b) return '0 MB';
  return b >= 1073741824 ? (b / 1073741824).toFixed(2) + ' GB' : (b / 1048576).toFixed(1) + ' MB';
}
function speedTxt(b) { if (!b) return '0'; const m = b / 1048576; return m >= 10 ? m.toFixed(0) : m.toFixed(1); }
function timeTxt(s) {
  s = Math.round(s); if (!s || s < 0) return '';
  const h = Math.floor(s / 3600), m = Math.floor((s % 3600) / 60), x = s % 60;
  return h ? `${h}:${String(m).padStart(2, '0')}:${String(x).padStart(2, '0')}` : `${m}:${String(x).padStart(2, '0')}`;
}
// 맥 / Windows에 따라 달라지는 말
const L = { fm: 'Finder', mod: '⌘', os: 'macOS', win: false };
function setPlatform(os) {
  if (os === 'windows') Object.assign(L, { fm: '탐색기', mod: 'Ctrl', os: 'Windows', win: true });
  document.documentElement.dataset.os = L.win ? 'windows' : 'mac';
  $('#nextBtn .kbd').textContent = L.win ? 'Ctrl ↵' : '⌘↩';
}
function shortPath(p) {
  const home = S.settings?.home;
  if (!L.win && home && (p === home || p.startsWith(home + '/'))) p = '~' + p.slice(home.length);
  return p;
}
const folderLabel = (p) => p.split(/[\\/]/).filter(Boolean).pop() || p;

// ════════════════════════════════════════════════ 화면 모드
function applyTheme(v) {
  if (v === 'light' || v === 'dark') document.documentElement.dataset.theme = v;
  else delete document.documentElement.dataset.theme;
  try { localStorage.setItem('ytroad-theme', v || 'system'); } catch { /* ignore */ }
  $$('#themeToggle button').forEach((b) => b.classList.toggle('on', b.dataset.theme === (v || 'system')));
  requestAnimationFrame(() => drawLine($('#chart'), totalHist, { grid: true }));
}
$$('#themeToggle button').forEach((b) => (b.onclick = async () => {
  applyTheme(b.dataset.theme);
  S.settings = await api('/api/settings', { appearance: b.dataset.theme }).catch(() => S.settings);
}));

// ════════════════════════════════════════════════ 1단계 · 주소 입력
const URL_RE = /(?:https?:\/\/)?(?:[\w-]+\.)*(?:youtube\.com|youtu\.be)\/[^\s<>"'`]+|https?:\/\/[^\s<>"'`]+/gi;
function extractUrls(text) {
  const out = [];
  for (let u of text.match(URL_RE) || []) {
    u = u.replace(/[),.;]+$/, '');
    if (!/^https?:\/\//i.test(u)) u = 'https://' + u;
    if (!out.includes(u)) out.push(u);
  }
  return out;
}
function onUrlsInput() {
  const n = extractUrls($('#urls').value).length;
  $('#nextBtn').disabled = n === 0;
  $('#detect').innerHTML = n ? `🔍 주소 <b>${n}개</b>를 찾았어요` : '주소를 붙여넣으면 자동으로 찾아 드려요';
}
$('#urls').addEventListener('input', onUrlsInput);
function addText(t) {
  const ta = $('#urls');
  ta.value = (ta.value.trim() ? ta.value.trim() + '\n' : '') + t.trim();
  onUrlsInput();
  ta.focus();
}
$('#pasteBtn').onclick = async () => {
  try {
    const t = await navigator.clipboard.readText();
    if (!t.trim()) return toast('📋 클립보드가 비어 있어요');
    if (S.step !== 1) setStep(1);
    addText(t);
    if (!extractUrls(t).length) toast('🤔 복사한 내용에서 주소를 찾지 못했어요');
  } catch { toast(`⌨️ 입력칸을 누르고 ${L.mod}+V로 붙여넣어 주세요`); $('#urls').focus(); }
};

function setStep(n) {
  S.step = n;
  $$('#steps li').forEach((li) => {
    const s = +li.dataset.s;
    li.classList.toggle('on', s === n);
    li.classList.toggle('past', s < n);
  });
  $('#step1').hidden = n !== 1;
  $('#step2').hidden = n !== 2;
}

// ════════════════════════════════════════════════ 2단계 · 확인
function thumbHtml(thumb, kind, fmt) {
  const ph = kind === 'link' ? '🌐' : kind === 'playlist' ? '📚' : kind === 'music' ? '🎵' : '🎬';
  const badge = kind === 'shorts' ? '<span class="badge shorts">Shorts</span>' : fmt ? `<span class="badge">${FMT[fmt].n}</span>` : '';
  if (!thumb) return `<span class="ph">${ph}</span>${badge}`;
  return `<span class="ph">${ph}</span><img src="${esc(thumb)}" alt="" loading="lazy" onerror="this.parentNode.classList.add('noimg');this.remove()"><div class="play"><span></span></div>${badge}`;
}

function summaryText() {
  const f = FMT[S.fmt];
  return `${f.n}${S.fmt === 'mp4' ? ' · ' + QN[S.quality] : ''} · 📁 ${folderLabel(S.folder)}`;
}

function renderPreviews() {
  const list = S.previews;
  $('#pvCount').textContent = list.length ? list.length + '개' : '';
  $('#s2Sum').textContent = summaryText();
  const good = list.filter((p) => p.ok !== false).length;
  const loading = list.some((p) => p.loading);
  $('#startBtn').disabled = good === 0 || loading;
  $('#startBtn').innerHTML = good > 1 ? `⬇︎ ${good}개 다운로드 시작` : '⬇︎ 다운로드 시작';
  $('#pvList').innerHTML = list.map((p, i) => p.loading ? `
    <div class="pv"><div class="thumb skeleton"></div><div class="tt">
      <div class="skeleton" style="height:13px;width:70%;margin-bottom:7px"></div>
      <div class="skeleton" style="height:10px;width:35%"></div></div></div>` : `
    <div class="pv${p.ok === false ? ' bad' : ''}">
      <div class="thumb">${thumbHtml(p.thumb, p.kind)}</div>
      <div class="tt">
        <div class="t" title="${esc(p.title)}">${esc(p.title || p.url)}</div>
        <div class="c">${p.channel ? '📺 ' + esc(p.channel) : esc(p.url)}</div>
        ${p.warn ? `<div class="w${p.ok === false ? ' bad' : ''}">${p.ok === false ? '🚫' : '⚠️'} ${esc(p.warn)}</div>` : ''}
      </div>
      ${p.ok === false ? '' : `<select data-i="${i}" title="이 영상만 다른 형식으로 받기">
        <option value="">기본 (${FMT[S.fmt].n})</option>
        ${Object.entries(FMT).map(([k, v]) => `<option value="${k}" ${p.fmt === k ? 'selected' : ''}>${v.n} · ${v.d}</option>`).join('')}
      </select>`}
      <button class="x" data-i="${i}" title="목록에서 빼기">✕</button>
    </div>`).join('');
  $$('#pvList select').forEach((s) => (s.onchange = () => { S.previews[+s.dataset.i].fmt = s.value; }));
  $$('#pvList .x').forEach((b) => (b.onclick = () => {
    S.previews.splice(+b.dataset.i, 1);
    if (!S.previews.length) { setStep(1); return; }
    renderPreviews();
  }));
}

async function goStep2() {
  const urls = extractUrls($('#urls').value);
  if (!urls.length) return;
  S.previews = urls.map((url) => ({ url, loading: true }));
  setStep(2);
  renderPreviews();
  try {
    const r = await api('/api/preview', { urls });
    S.previews = r.items.map((p) => ({ ...p, fmt: '' }));
  } catch {
    S.previews = urls.map((url) => ({ url, title: url, kind: 'link', warn: '정보를 불러오지 못했어요. 그래도 받을 수 있어요.', fmt: '' }));
  }
  if (S.step === 2) renderPreviews();
}
$('#nextBtn').onclick = goStep2;
$('#backBtn').onclick = () => { setStep(1); $('#urls').focus(); };

$('#startBtn').onclick = async () => {
  const items = S.previews.filter((p) => p.ok !== false && !p.loading).map((p) => ({
    url: p.url, title: p.title, channel: p.channel, thumb: p.thumb, kind: p.kind, fmt: p.fmt || S.fmt, quality: S.quality,
  }));
  if (!items.length) return;
  $('#startBtn').disabled = true;
  try {
    await api('/api/jobs', { items, outDir: S.folder });
    setStep(3);
    toast(`🚀 ${items.length}개 다운로드를 시작했어요`);
    S.filter = 'all'; paintTabs();
    setTimeout(() => { $('#urls').value = ''; onUrlsInput(); S.previews = []; setStep(1); }, 900);
    setTimeout(() => $('#dash').scrollIntoView({ behavior: 'smooth', block: 'start' }), 250);
    poll();
  } catch (e) { toast('😢 ' + e.message); $('#startBtn').disabled = false; }
};

// ════════════════════════════════════════════════ 오른쪽 · 형식 · 화질 · 폴더 · 동시 다운로드
function paintOptions() {
  $$('#fmtGrid .fmt').forEach((b) => b.classList.toggle('on', b.dataset.fmt === S.fmt));
  $$('#qGrid button').forEach((b) => b.classList.toggle('on', b.dataset.q === S.quality));
  $('#qualityCard').classList.toggle('off', S.fmt !== 'mp4');
  if (S.step === 2) renderPreviews();
}
$$('#fmtGrid .fmt').forEach((b) => (b.onclick = () => {
  S.fmt = b.dataset.fmt; paintOptions();
  api('/api/settings', { fmt: S.fmt }).catch(() => {});
}));
$$('#qGrid button').forEach((b) => (b.onclick = () => {
  S.quality = b.dataset.q; paintOptions();
  api('/api/settings', { quality: S.quality }).catch(() => {});
}));

function renderFolder() {
  const def = S.settings.resolvedDefaultFolder;
  $('#folderName').textContent = folderLabel(S.folder);
  $('#folderPath').textContent = '\u200E' + shortPath(S.folder) + '\u200E';
  $('#folderMain').title = S.folder + `\n눌러서 ${L.fm}에서 열기`;
  const isDef = S.folder === def;
  $('#folderPin').hidden = isDef;
  $('#folderSub').innerHTML = isDef
    ? `<span class="pin-ok">📌 기본 저장 폴더예요</span><span class="sp"></span>${S.settings.defaultFolder ? '<button class="link" id="folderReset">다운로드 폴더로 되돌리기</button>' : ''}`
    : `<span>이번에만 이 폴더에 저장해요.</span><span class="sp"></span><button class="link" id="folderBack">기본 폴더로</button>`;
  const r = $('#folderReset');
  if (r) r.onclick = async () => {
    S.settings = await api('/api/settings', { defaultFolder: '' });
    S.folder = S.settings.resolvedDefaultFolder; renderFolder(); toast('📁 기본 폴더를 다운로드 폴더로 되돌렸어요');
  };
  const bk = $('#folderBack');
  if (bk) bk.onclick = () => { S.folder = def; renderFolder(); };
  if (S.step === 2) $('#s2Sum').textContent = summaryText();
}
async function chooseFolder(prompt) {
  if (S.settings.folderPicker === 'web') return pickFolderWeb(S.folder, prompt);
  const j = await api('/api/choose-folder', { start: S.folder, prompt }).catch((e) => { toast('⚠️ ' + e.message); return null; });
  return j?.path || '';
}

// 앱 화면 안에서 폴더 고르기 (Windows)
function pickFolderWeb(start, title) {
  return new Promise((resolve) => {
    const el = document.createElement('div');
    el.className = 'fp-back';
    el.innerHTML = `
      <div class="fp">
        <div class="fp-head"><div><b>📁 폴더 고르기</b><small>${esc(title || '')}</small></div></div>
        <div class="fp-main">
          <div class="fp-side" id="fpSide"></div>
          <div class="fp-list-wrap">
            <div class="fp-path"><button class="btn ghost sm" id="fpUp" title="상위 폴더">⬆︎</button><span id="fpPath"></span></div>
            <div class="fp-list" id="fpList"></div>
          </div>
        </div>
        <div class="fp-foot">
          <button class="btn ghost sm" id="fpNew">➕ 새 폴더</button>
          <span class="sp"></span>
          <button class="btn ghost" id="fpCancel">취소</button>
          <button class="btn primary" id="fpOk">이 폴더 선택</button>
        </div>
      </div>`;
    document.body.append(el);
    let cur = null;
    const done = (v) => { el.remove(); document.removeEventListener('keydown', onKey, true); resolve(v); };
    const onKey = (e) => { if (e.key === 'Escape') { e.stopPropagation(); e.preventDefault(); done(''); } };
    document.addEventListener('keydown', onKey, true);
    async function go(path) {
      let j;
      try { j = await api('/api/fs/list', { path }); } catch (e) { toast('⚠️ ' + e.message); return; }
      cur = j;
      $('#fpPath', el).textContent = '\u200E' + j.path + '\u200E';
      $('#fpUp', el).disabled = !j.parent;
      $('#fpSide', el).innerHTML = [
        '<div class="fp-cap">바로 가기</div>',
        ...j.places.map((p) => `<button data-p="${esc(p.path)}" class="${p.path === j.path ? 'on' : ''}">${p.icon} ${esc(p.name)}</button>`),
        '<div class="fp-cap">드라이브</div>',
        ...j.drives.map((p) => `<button data-p="${esc(p.path)}" class="${p.path === j.path ? 'on' : ''}">${p.icon} ${esc(p.name)}</button>`),
      ].join('');
      $('#fpList', el).innerHTML = j.dirs.length
        ? j.dirs.map((d) => `<button data-p="${esc(d.path)}"><span class="fp-ico">📁</span>${esc(d.name)}</button>`).join('')
        : `<div class="fp-empty">${j.readable ? '안에 다른 폴더가 없어요.<br>이 폴더에 저장하려면 ‘이 폴더 선택’을 누르세요.' : '이 폴더는 열 수 없어요.'}</div>`;
      $$('[data-p]', el).forEach((b) => (b.onclick = () => go(b.dataset.p)));
      $('#fpList', el).scrollTop = 0;
    }
    $('#fpUp', el).onclick = () => cur?.parent && go(cur.parent);
    $('#fpCancel', el).onclick = () => done('');
    $('#fpOk', el).onclick = () => done(cur?.path || '');
    el.addEventListener('mousedown', (e) => { if (e.target === el) done(''); });
    $('#fpNew', el).onclick = async () => {
      const name = prompt('새 폴더 이름을 입력하세요', 'YTROAD');
      if (!name || !cur) return;
      try { const j = await api('/api/fs/mkdir', { path: cur.path, name }); go(j.path); }
      catch (e) { toast('⚠️ ' + e.message); }
    };
    go(start);
  });
}
$('#folderChange').onclick = async () => {
  const p = await chooseFolder('받은 영상·음악을 저장할 폴더를 선택하세요');
  if (p) { S.folder = p; renderFolder(); toast(`📁 이제 ‘${folderLabel(p)}’ 폴더에 저장해요`); }
};
$('#folderPin').onclick = async () => {
  S.settings = await api('/api/settings', { defaultFolder: S.folder });
  renderFolder(); toast('📌 다음에 앱을 켤 때도 이 폴더에 저장해요');
};
$('#folderMain').onclick = () => api('/api/open-folder', { path: S.folder }).catch(() => {});

function paintParallel() {
  const n = S.settings.parallel;
  $('#parRange').value = n;
  $('#parVal').textContent = n + '개';
  $('#leadPar').textContent = n;
  $('#sPar').textContent = '/ ' + n;
}
$('#parRange').oninput = (e) => { $('#parVal').textContent = e.target.value + '개'; };
$('#parRange').onchange = async (e) => {
  S.settings = await api('/api/settings', { parallel: +e.target.value }).catch(() => S.settings);
  paintParallel();
};

// ════════════════════════════════════════════════ 속도 그래프
const H = 120; // 0.5초 × 120 = 60초
const totalHist = Array(H).fill(0);
const jobHist = new Map();

function drawLine(canvas, data, { fill = true, grid = false } = {}) {
  if (!canvas) return 0;
  const dpr = window.devicePixelRatio || 1;
  const w = canvas.clientWidth, h = canvas.clientHeight;
  if (!w || !h) return 0;
  if (canvas.width !== Math.round(w * dpr) || canvas.height !== Math.round(h * dpr)) { canvas.width = Math.round(w * dpr); canvas.height = Math.round(h * dpr); }
  const ctx = canvas.getContext('2d');
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  ctx.clearRect(0, 0, w, h);
  const css = getComputedStyle(document.documentElement);
  const red = css.getPropertyValue('--red').trim() || '#ff0033';
  const line = css.getPropertyValue('--stroke2').trim();
  const top = Math.max(Math.max(...data), 1);
  const pad = 3;
  if (grid) {
    ctx.strokeStyle = line; ctx.lineWidth = 1; ctx.setLineDash([3, 4]);
    for (let i = 1; i <= 2; i++) { const y = Math.round(pad + (h - 2 * pad) * (i / 3)) + 0.5; ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(w, y); ctx.stroke(); }
    ctx.setLineDash([]);
  }
  const pts = data.map((v, i) => [(i / (data.length - 1)) * w, h - pad - (v / top) * (h - 2 * pad)]);
  const path = () => {
    ctx.beginPath();
    pts.forEach(([x, y], i) => {
      if (!i) return ctx.moveTo(x, y);
      const [px, py] = pts[i - 1]; const cx = (px + x) / 2; ctx.bezierCurveTo(cx, py, cx, y, x, y);
    });
  };
  if (fill) {
    path(); ctx.lineTo(w, h); ctx.lineTo(0, h); ctx.closePath();
    const g = ctx.createLinearGradient(0, 0, 0, h); g.addColorStop(0, red + '55'); g.addColorStop(1, red + '00');
    ctx.fillStyle = g; ctx.fill();
  }
  path();
  ctx.strokeStyle = red; ctx.lineWidth = 2; ctx.lineJoin = 'round'; ctx.stroke();
  const [lx, ly] = pts[pts.length - 1];
  if (data[data.length - 1] > 0) { ctx.fillStyle = red; ctx.beginPath(); ctx.arc(lx - 3, ly, 3.2, 0, 7); ctx.fill(); }
  return top;
}
window.addEventListener('resize', () => drawLine($('#chart'), totalHist, { grid: true }));

// ════════════════════════════════════════════════ 다운로드 목록
const cards = new Map();

function actionsFor(j) {
  if (j.state === 'queued') return `<button class="act" data-a="top" title="이 영상을 먼저 받기">⏫ 먼저</button><button class="act danger" data-a="cancel">✕ 취소</button>`;
  if (ACTIVE.includes(j.state)) return `<button class="act danger" data-a="cancel">✕ 취소</button>`;
  if (j.state === 'done') return `<button class="act go" data-a="reveal">🔍 ${L.fm}에서 보기</button><button class="act ic" data-a="open" title="열어서 재생하기">▶︎</button><button class="act ic" data-a="remove" title="목록에서 빼기 (파일은 그대로)">✕</button>`;
  return `<button class="act" data-a="retry">🔁 다시 시도</button><button class="act ic" data-a="remove" title="목록에서 빼기">✕</button>`;
}

function makeCard(j) {
  const el = document.createElement('div');
  el.className = 'job';
  el.innerHTML = `
    <div class="thumb">${thumbHtml(j.thumb, j.kind || (j.thumb ? 'video' : 'link'), j.fmt)}</div>
    <div class="body">
      <div class="t1"><span class="title"></span><span class="tag"></span></div>
      <div class="sub"></div>
      <div class="stage"></div>
      <div class="bar"><div class="fill"></div></div>
      <div class="meta"></div>
      <div class="err" hidden></div>
    </div>
    <canvas class="spark"></canvas>
    <div class="acts"></div>`;
  el.querySelector('.acts').addEventListener('click', async (e) => {
    const a = e.target.closest('[data-a]')?.dataset.a;
    if (!a) return;
    if (a === 'cancel' && ACTIVE.includes(j.state) && !confirm('이 다운로드를 취소할까요?')) return;
    try { await api('/api/job', { id: j.id, action: a }); poll(); } catch (err) { toast('😢 ' + err.message); }
  });
  el.querySelector('.thumb').addEventListener('dblclick', () => { if (el.dataset.state === 'done') api('/api/job', { id: j.id, action: 'open' }).catch(() => {}); });
  return el;
}

function updateCard(el, j) {
  el.dataset.state = j.state;
  el.className = `job ${j.state}${ACTIVE.includes(j.state) ? ' active' : ''}`;
  const title = el.querySelector('.title');
  const tt = j.title || j.url;
  if (title.textContent !== tt) { title.textContent = tt; title.title = tt; }
  const tag = el.querySelector('.tag');
  const tagTxt = `${FMT[j.fmt].n}${j.fmt === 'mp4' && j.quality !== 'best' ? ' · ' + QN[j.quality] : ''}`;
  if (tag.textContent !== tagTxt) { tag.textContent = tagTxt; tag.style.setProperty('--c', FMT[j.fmt].c); }
  const sub = `${j.channel ? '📺 ' + j.channel + ' · ' : ''}📁 ${folderLabel(j.outDir)}`;
  const subEl = el.querySelector('.sub');
  if (subEl.textContent !== sub) subEl.textContent = sub;
  el.querySelector('.stage').textContent = j.stage + (j.state === 'downloading' && j.fmt === 'mp4' && j.part > 1 ? ` (${Math.min(j.part, 2)}/2)` : '');
  el.querySelector('.fill').style.width = (j.state === 'done' ? 100 : j.pct) + '%';
  let meta = '';
  if (j.state === 'downloading') {
    meta = `<span><b class="num">${j.pct.toFixed(1)}%</b></span>
      <span class="num">💾 ${mb(j.downloaded)} / ${j.total ? mb(j.total) : '?'}</span>
      <span class="num">⚡ <b>${speedTxt(j.speed)}</b> MB/s</span>
      ${j.eta ? `<span class="num">⏱️ <b>${timeTxt(j.eta)}</b> 남음</span>` : ''}`;
  } else if (j.state === 'done') {
    const secs = Math.max(1, Math.round((j.finishedAt - j.startedAt) / 1000));
    meta = `<span class="num">💾 ${mb(j.size)}</span><span class="num">⏱️ ${timeTxt(secs)} 걸림</span>`;
  } else if (j.state === 'queued') {
    meta = '<span>앞의 다운로드가 끝나면 자동으로 시작해요</span>';
  }
  const m = el.querySelector('.meta');
  if (m.innerHTML !== meta) m.innerHTML = meta;
  const err = el.querySelector('.err');
  err.hidden = !j.error;
  if (err.textContent !== j.error) err.textContent = j.error;
  const acts = el.querySelector('.acts');
  if (acts.dataset.state !== j.state) { acts.dataset.state = j.state; acts.innerHTML = actionsFor(j); }
  const spark = el.querySelector('.spark');
  const hist = jobHist.get(j.id);
  const live = j.state === 'downloading' || (ACTIVE.includes(j.state) && hist.some((v) => v > 0));
  spark.style.display = live ? '' : 'none';
  if (live) drawLine(spark, hist, { fill: true });
}

function matchFilter(j) {
  switch (S.filter) {
    case 'active': return ACTIVE.includes(j.state) || j.state === 'queued';
    case 'done': return j.state === 'done';
    case 'error': return j.state === 'error' || j.state === 'canceled';
    default: return true;
  }
}
function paintTabs() { $$('#jobTabs button').forEach((b) => b.classList.toggle('on', b.dataset.f === S.filter)); }
$$('#jobTabs button').forEach((b) => (b.onclick = () => { S.filter = b.dataset.f; paintTabs(); renderJobs(S.jobs, false); }));

let lastJobs = [];
function renderJobs(jobs, tick = true) {
  const list = $('#jobs');
  const ids = new Set(jobs.map((j) => j.id));
  for (const [id, el] of cards) if (!ids.has(id)) { el.remove(); cards.delete(id); jobHist.delete(id); }
  // 받는 중 → 대기 → 끝난 항목(최근 것 위) 순서
  const rank = (j) => (ACTIVE.includes(j.state) ? 0 : j.state === 'queued' ? 1 : 2);
  const ordered = [...jobs].sort((a, b) => rank(a) - rank(b) || (rank(a) === 2 ? b.finishedAt - a.finishedAt : a.addedAt - b.addedAt));
  let idx = 0;
  for (const j of ordered) {
    if (!jobHist.has(j.id)) jobHist.set(j.id, Array(40).fill(0));
    if (tick) { const h = jobHist.get(j.id); h.push(j.state === 'downloading' ? j.speed / 1048576 : 0); h.shift(); }
    let el = cards.get(j.id);
    if (!el) { el = makeCard(j); cards.set(j.id, el); }
    if (!matchFilter(j)) { if (el.parentNode) el.remove(); continue; }
    if (list.children[idx] !== el) list.insertBefore(el, list.children[idx] || null);
    updateCard(el, j);
    idx++;
  }
  const shown = idx;
  const cnt = {
    all: jobs.length,
    active: jobs.filter((j) => ACTIVE.includes(j.state) || j.state === 'queued').length,
    done: jobs.filter((j) => j.state === 'done').length,
    error: jobs.filter((j) => j.state === 'error' || j.state === 'canceled').length,
  };
  $('#cAll').textContent = cnt.all; $('#cActive').textContent = cnt.active; $('#cDone').textContent = cnt.done; $('#cErr').textContent = cnt.error;
  $('#empty').hidden = shown > 0;
  if (!shown && jobs.length) {
    $('#empty p').textContent = '여기에 해당하는 항목이 없어요';
    $('#empty small').textContent = '다른 탭을 눌러 보세요';
  } else if (!jobs.length) {
    $('#empty p').textContent = '아직 받은 영상이 없어요';
    $('#empty small').textContent = '위에 유튜브 주소를 붙여넣고 시작해 보세요 🍿';
  }
  $('#clearBtn').hidden = !(cnt.done + cnt.error);
  $('#retryAllBtn').hidden = !jobs.some((j) => j.state === 'error');

  const active = jobs.filter((j) => ACTIVE.includes(j.state));
  const speed = active.reduce((s, j) => s + (j.state === 'downloading' ? j.speed : 0), 0);
  $('#sSpeed').innerHTML = `${speedTxt(speed)}<small>MB/s</small>`;
  $('#sActive').textContent = active.length;
  const queued = jobs.filter((j) => j.state === 'queued').length;
  $('#sQueued').textContent = queued;
  $('#sDone').textContent = cnt.done;
  if (tick) { totalHist.push(speed / 1048576); totalHist.shift(); }
  const top = drawLine($('#chart'), totalHist, { grid: true });
  $('#chartMax').textContent = top > 1 ? `최고 ${top.toFixed(1)} MB/s` : '';

  // 전체 진행률
  const tracked = jobs.filter((j) => j.state === 'downloading' && j.total);
  $('#overall').hidden = !active.length && !queued;
  if (tracked.length) {
    const d = tracked.reduce((s, j) => s + j.downloaded, 0), t = tracked.reduce((s, j) => s + j.total, 0);
    const pct = t ? (d / t) * 100 : 0;
    $('#oFill').style.width = pct + '%';
    $('#oPct').textContent = pct.toFixed(0) + '%';
    const eta = speed ? (t - d) / speed : 0;
    $('#oEta').textContent = eta ? `· ⏱️ 약 ${timeTxt(eta)} 남음` : '';
  } else { $('#oFill').style.width = '0%'; $('#oPct').textContent = '—'; $('#oEta').textContent = ''; }

  // 위쪽 '받는 중' 표시
  $('#topLive').hidden = !active.length && !queued;
  $('#topLiveText').textContent = `${active.length}개 받는 중${queued ? ` · ${queued}개 대기` : ''} · ${speedTxt(speed)} MB/s`;
  document.title = active.length ? `⬇︎ ${active.length} · YTROAD` : 'YTROAD';

  // 완료·실패 알림 (창 안)
  if (tick) {
    for (const j of jobs) {
      const prev = lastJobs.find((p) => p.id === j.id);
      if (prev && prev.state !== 'done' && j.state === 'done') toast(`✅ 완료: ${j.title}`);
      if (prev && prev.state !== 'error' && j.state === 'error') toast(`⚠️ 실패: ${j.title || j.url}`);
    }
    lastJobs = jobs;
  }
}
$('#clearBtn').onclick = () => api('/api/jobs/clear', {}).then(poll).catch(() => {});
$('#retryAllBtn').onclick = async () => {
  for (const j of S.jobs.filter((x) => x.state === 'error')) await api('/api/job', { id: j.id, action: 'retry' }).catch(() => {});
  poll();
};

// ════════════════════════════════════════════════ 처음 한 번 준비
let setupKicked = false;
function renderSetup(t) {
  const show = !t.ready;
  $('#setupCard').hidden = !show;
  $('#composer').hidden = show;
  if (!show) return;
  if (!t.running && !t.error && !setupKicked) { setupKicked = true; api('/api/tools/install', {}).catch(() => {}); }
  $('#setupList').innerHTML = t.items.map((i) => {
    const pct = i.state === 'done' ? 100 : i.total ? (i.done / i.total) * 100 : 0;
    const st = i.state === 'done' ? '✅ 완료' : i.state === 'error' ? '⚠️ 실패' : i.state === 'copy' ? '📦 가져오는 중'
      : i.state === 'down' ? `${mb(i.done)}${i.total ? ' / ' + mb(i.total) : ''}` : '대기 중';
    return `<div class="setup-item"><span class="e">${i.emoji}</span><span class="n">${esc(i.label)}</span>
      <div class="bar"><div class="fill" style="width:${pct}%;${i.state === 'done' ? 'background:var(--ok)' : ''}"></div></div>
      <span class="s num">${st}</span></div>`;
  }).join('');
  $('#setupNote').hidden = !t.migrated;
  $('#setupNote').textContent = '📦 예전 YT Downloader에 있던 도구를 그대로 가져와서 빨리 끝나요.';
  $('#setupErr').hidden = !t.error;
  $('#setupErrText').textContent = t.error ? '😢 ' + t.error : '';
}
$('#setupRetry').onclick = () => api('/api/tools/install', {}).catch(() => {});

// ════════════════════════════════════════════════ 창 (모달)
const M = { open: false, onClose: null };
function openModal({ title, sub = '', body, actions = [], onClose }) {
  M.open = true; M.onClose = onClose || null;
  $('#modalTitle').textContent = title;
  $('#modalSub').textContent = sub;
  setModalActions(actions);
  const b = $('#modalBody'); b.innerHTML = ''; b.append(body);
  $('#modal').hidden = false;
}
function setModalActions(actions) {
  const a = $('#modalActions'); a.innerHTML = '';
  for (const act of actions) {
    const btn = document.createElement('button');
    btn.className = 'btn' + (act.primary ? ' primary' : ' ghost');
    btn.textContent = act.label; btn.disabled = !!act.disabled; btn.onclick = act.onClick;
    a.append(btn);
  }
}
function closeModal() {
  if (!M.open) return;
  M.open = false; $('#modal').hidden = true;
  const cb = M.onClose; M.onClose = null; cb && cb();
}
$('#modalClose').onclick = closeModal;
$('#modal').addEventListener('mousedown', (e) => { if (e.target.id === 'modal') closeModal(); });

// ════════════════════════════════════════════════ 설정
function fmtWhen(ms) {
  if (!ms) return '아직 확인 안 함';
  const d = new Date(ms), now = new Date();
  const hm = d.toLocaleTimeString('ko-KR', { hour: 'numeric', minute: '2-digit' });
  return d.toDateString() === now.toDateString() ? `오늘 ${hm}` : `${d.getMonth() + 1}월 ${d.getDate()}일 ${hm}`;
}

let settingsRender = null;
function openSettings() {
  const body = document.createElement('div');
  body.style.cssText = 'display:flex;flex-direction:column;gap:12px';
  const render = () => {
    const st = S.settings, t = S.tools || {}, u = S.update || {};
    body.innerHTML = `
      <div class="card"><div class="card-title">🌓 화면 모드</div>
        <div class="seg" id="stTheme"><button data-v="system">시스템 설정</button><button data-v="light">라이트</button><button data-v="dark">다크</button></div>
        <div class="note">‘시스템 설정’을 고르면 ${L.os}의 라이트/다크 모드를 그대로 따라가요.</div></div>
      <div class="card"><div class="card-title">📁 기본 저장 폴더</div>
        <div class="folder-main" style="cursor:default"><span class="folder-ico" data-art="folder"></span><div class="folder-text"><div class="folder-name">${esc(folderLabel(st.resolvedDefaultFolder))}</div><div class="folder-path">&lrm;${esc(shortPath(st.resolvedDefaultFolder))}&lrm;</div></div>
        <span class="sp"></span><button class="btn ghost sm" id="stFolder">변경…</button></div>
        <div class="folder-sub"><span class="note">앱을 켤 때마다 이 폴더가 저장 위치로 선택돼요.</span>${st.defaultFolder ? '<span class="sp"></span><button class="link" id="stReset">다운로드 폴더로</button>' : ''}</div></div>
      <div class="card"><div class="card-title">🔔 알림</div>
        <label class="switch-row"><input type="checkbox" id="stNotify" ${st.notify ? 'checked' : ''}><span class="switch"></span>다운로드가 끝나면 ${L.os} 알림 보내기</label></div>
      <div class="card"><div class="card-title">🚀 다운로드 엔진 <span class="note" style="margin-left:auto">yt-dlp</span></div>
        <div class="kv"><span>엔진 버전</span><b>${esc(t.engineVersion || '—')}</b></div>
        <div class="kv"><span>마지막 업데이트 확인</span><b>${t.engineChecked ? fmtWhen(t.engineChecked * 1000) : '—'}</b></div>
        <div class="note">유튜브는 자주 바뀌어요. 엔진은 하루에 한 번 스스로 업데이트하고, 받기에 실패하면 바로 업데이트를 시도해요.${t.engineMessage ? `<br><b>${esc(t.engineMessage)}</b>` : ''}</div>
        <button class="btn ghost sm" id="stEngine" ${t.engineUpdating || !t.ready ? 'disabled' : ''}>${t.engineUpdating ? '⏳ 엔진 업데이트 중…' : '🔄 지금 엔진 업데이트'}</button></div>
      <div class="card"><div class="card-title">ℹ️ 정보</div>
        <div class="kv"><span>버전</span><b>YTROAD ${esc(st.build)}</b></div>
        <div class="kv"><span>앱 업데이트</span><b>${u.autoUpdate === false ? '자동 업데이트 꺼짐' : '자동 (6시간마다 확인)'}</b></div>
        <div style="display:flex;gap:6px"><button class="btn ghost sm" id="stUpdate" style="flex:1">🔄 업데이트 확인…</button><button class="btn ghost sm" id="stQuit" style="flex:1">⏻ YTROAD 종료</button></div></div>
      <div class="settings-foot">made by. Nevertheless_D</div>`;
    paintArt(body);
    $$('#stTheme button', body).forEach((b) => {
      b.classList.toggle('on', b.dataset.v === (st.appearance || 'system'));
      b.onclick = async () => { applyTheme(b.dataset.v); S.settings = await api('/api/settings', { appearance: b.dataset.v }); render(); };
    });
    $('#stFolder', body).onclick = async () => {
      const p = await chooseFolder('기본으로 사용할 저장 폴더를 선택하세요');
      if (p) { S.settings = await api('/api/settings', { defaultFolder: p }); S.folder = S.settings.resolvedDefaultFolder; renderFolder(); render(); }
    };
    const rs = $('#stReset', body);
    if (rs) rs.onclick = async () => { S.settings = await api('/api/settings', { defaultFolder: '' }); S.folder = S.settings.resolvedDefaultFolder; renderFolder(); render(); };
    $('#stNotify', body).onchange = async (e) => { S.settings = await api('/api/settings', { notify: e.target.checked }); };
    $('#stEngine', body).onclick = async () => { await api('/api/tools/update', {}).catch(() => {}); toast('🔄 다운로드 엔진을 확인하고 있어요'); setTimeout(poll, 300); };
    $('#stUpdate', body).onclick = () => { closeModal(); openUpdate(true); };
    $('#stQuit', body).onclick = quitApp;
  };
  render();
  settingsRender = render;
  openModal({ title: '⚙️ YTROAD 설정', body, onClose: () => { settingsRender = null; } });
}
$('#settingsBtn').onclick = openSettings;

// ════════════════════════════════════════════════ 버전 · 업데이트
// 앱이 GitHub(NeverthelessD/YTROAD)의 releases/latest.json 을 보고 새 버전을 받아요.
const U = { open: false, sig: '', restarting: false, toldReady: '' };
const updBody = document.createElement('div');
updBody.style.cssText = 'display:flex;flex-direction:column;gap:12px';

function onUpdateState(st) {
  S.update = st;
  renderVersion();
  const sig = JSON.stringify(st);
  if (U.open && sig !== U.sig) renderUpdate();
  U.sig = sig;
  if (st.status === 'ready' && st.installed && U.toldReady !== st.installed && !U.open) {
    U.toldReady = st.installed;
    toast(`✨ YTROAD ${st.installed} 준비 완료! 오른쪽 위 배지를 눌러 다시 시작하세요.`, 6000);
  }
}

function renderVersion() {
  const st = S.update;
  if (!st) return;
  let txt = 'v' + st.current, isNew = false;
  if (st.status === 'available') { txt = `새 버전 ${st.latest.version}`; isNew = true; }
  else if (st.status === 'downloading') { txt = `받는 중 ${Math.round(st.progress * 100)}%`; isNew = true; }
  else if (st.status === 'installing') { txt = '설치 중…'; isNew = true; }
  else if (st.status === 'ready') { txt = '다시 시작해서 업데이트'; isNew = true; }
  $('#verText').textContent = txt;
  $('#verBadge').classList.toggle('new', isNew);
}

function renderUpdate() {
  const st = S.update;
  if (!st) return;
  const L = st.latest;
  let t1 = '', t2 = '', showVers = false, acts = [];
  switch (st.status) {
    case 'checking': t1 = '새 버전을 확인하고 있어요…'; t2 = 'GitHub에 물어보는 중이에요.'; break;
    case 'available':
      t1 = '새 버전이 나왔어요! 🎉'; t2 = '받는 데 몇 초면 끝나요. 받는 중인 영상은 그대로 둬도 돼요.'; showVers = true;
      acts = [{ label: '⬇︎ 지금 업데이트', primary: true, onClick: installUpdate }]; break;
    case 'downloading': t1 = '새 버전을 받고 있어요…'; t2 = `${Math.round(st.progress * 100)}%`; showVers = true; break;
    case 'installing': t1 = '설치하고 있어요…'; t2 = '잠시만 기다려 주세요.'; showVers = true; break;
    case 'ready':
      t1 = `YTROAD ${st.installed} 준비 완료 ✨`; t2 = '다시 시작하면 새 버전으로 바뀌어요. (몇 초 걸려요)';
      acts = [{ label: '↻ 지금 다시 시작', primary: true, onClick: restartApp }]; break;
    case 'uptodate': t1 = '최신 버전을 쓰고 있어요 👍'; t2 = `YTROAD ${st.current}`; break;
    case 'error': t1 = '업데이트를 확인하지 못했어요'; t2 = '인터넷 연결을 확인한 뒤 다시 시도해 주세요.'; break;
    default: t1 = `YTROAD ${st.current}`; t2 = '버튼을 눌러 새 버전이 있는지 확인해 보세요.';
  }
  const busy = ['checking', 'downloading', 'installing'].includes(st.status);
  if (!['available', 'ready'].includes(st.status)) acts = [{ label: '🔄 업데이트 확인', primary: true, disabled: busy, onClick: checkUpdate }];
  const notes = ['available', 'downloading', 'installing', 'ready'].includes(st.status) && L?.notes?.length
    ? `<div class="card"><div class="card-title">📝 ${esc(L.version)}에서 바뀐 점</div><ul class="upd-notes">${L.notes.map((n) => `<li>${esc(n)}</li>`).join('')}</ul></div>` : '';
  updBody.innerHTML = `
    <div class="upd-hero"><img src="/icon.png" alt=""><div><div class="t1">${t1}</div><div class="t2">${esc(t2)}</div></div></div>
    ${showVers && L ? `<div class="upd-vers"><div class="upd-ver"><small>지금 버전</small><b>${esc(st.current)}</b></div><span class="upd-arrow">➜</span><div class="upd-ver new"><small>새 버전</small><b>${esc(L.version)}</b></div></div>` : ''}
    ${st.status === 'downloading' || st.status === 'installing' ? `<div class="upd-bar"><i style="width:${st.status === 'installing' ? 100 : Math.round(st.progress * 100)}%"></i></div>` : ''}
    ${st.status === 'error' && st.error ? `<div class="upd-err">${esc(st.error)}</div>` : ''}
    ${notes}
    <div class="card">
      <label class="switch-row"><input type="checkbox" id="updAuto" ${st.autoUpdate ? 'checked' : ''}><span class="switch"></span>자동 업데이트 (새 버전을 알아서 받아 둬요)</label>
      <div class="kv"><span>지금 버전</span><b>YTROAD ${esc(st.current)}</b></div>
      <div class="kv"><span>마지막 확인</span><b>${fmtWhen(st.lastCheck)}</b></div>
      ${st.canRollback ? `<div class="folder-sub"><span class="note">문제가 생겼다면 이전 버전(${esc(st.backupVersion)})으로 되돌릴 수 있어요.</span><span class="sp"></span><button class="link" id="updRollback">되돌리기</button></div>` : ''}
    </div>`;
  $('#updAuto', updBody).onchange = async (e) => {
    S.settings = await api('/api/settings', { autoUpdate: e.target.checked });
    S.update.autoUpdate = e.target.checked;
    toast(e.target.checked ? '✅ 자동 업데이트를 켰어요' : '자동 업데이트를 껐어요. 필요할 때 직접 확인하세요.');
  };
  const rb = $('#updRollback', updBody);
  if (rb) rb.onclick = rollbackUpdate;
  setModalActions(acts);
}

function openUpdate(check = false) {
  U.open = true;
  openModal({ title: '🔄 업데이트', body: updBody, onClose: () => { U.open = false; } });
  renderUpdate();
  if (check && !['available', 'ready', 'downloading', 'installing'].includes(S.update?.status)) checkUpdate();
}
$('#verBadge').onclick = () => openUpdate(true);

async function checkUpdate() {
  if (S.update) { S.update.status = 'checking'; renderUpdate(); renderVersion(); }
  try { onUpdateState(await api('/api/update/check')); } catch (e) { toast('⚠️ ' + e.message); }
}
async function installUpdate() {
  try { await api('/api/update/install', {}); } catch (e) { toast('⚠️ ' + e.message); }
  if (S.update) { S.update.status = 'downloading'; S.update.progress = 0; renderUpdate(); }
}
async function rollbackUpdate() {
  if (!confirm(`이전 버전(${S.update.backupVersion})으로 되돌릴까요?\n되돌리면 자동 업데이트는 꺼져요. (설정에서 다시 켤 수 있어요)`)) return;
  try { await api('/api/update/rollback', {}); } catch (e) { toast('⚠️ ' + e.message); return; }
  toast('↩︎ 이전 버전을 준비했어요. 다시 시작하면 바뀌어요.');
  poll();
}

async function restartApp() {
  const busy = S.jobs.some((j) => ACTIVE.includes(j.state) || j.state === 'queued');
  if (busy && !confirm('받는 중인 영상이 있어요. 다시 시작하면 처음부터 다시 받아야 해요.\n그래도 지금 다시 시작할까요?')) return;
  const from = S.update?.current;
  U.restarting = true;
  closeModal();
  const veil = document.createElement('div');
  veil.className = 'restart-veil';
  veil.innerHTML = '<img src="/icon.png" alt="">새 버전으로 다시 시작하고 있어요…<span>창을 닫지 말고 잠시만 기다려 주세요.</span>';
  document.body.append(veil);
  try { await api('/api/update/restart', {}); }
  catch (e) { veil.remove(); U.restarting = false; toast('⚠️ ' + e.message); return; }
  const t0 = Date.now();
  const tick = async () => {
    try {
      const j = await api('/api/ping');
      if (j.build && j.build !== from) { location.reload(); return; }
    } catch { /* 아직 다시 켜지는 중 */ }
    if (Date.now() - t0 > 60000) { showEnded('다시 시작에 시간이 걸리고 있어요. YTROAD 앱을 다시 실행해 주세요.'); return; }
    setTimeout(tick, 600);
  };
  setTimeout(tick, 1200);
}

// ════════════════════════════════════════════════ 종료
let ended = false;
function showEnded(msg) {
  ended = true;
  document.body.innerHTML = `<div class="ended"><img src="/icon.png" alt=""><b>${msg}</b><span>이 창은 닫아도 돼요. 다시 쓰려면 YTROAD 앱을 실행하세요.</span></div>`;
}
async function quitApp() {
  const busy = S.jobs.some((j) => ACTIVE.includes(j.state) || j.state === 'queued');
  if (!confirm(busy ? '받는 중인 영상이 있어요. 모두 취소하고 종료할까요?' : 'YTROAD를 종료할까요?')) return;
  await api('/api/quit', {}).catch(() => {});
  showEnded('YTROAD를 종료했어요 👋');
  setTimeout(() => window.close(), 400);
}
$('#quitBtn').onclick = quitApp;

// ════════════════════════════════════════════════ 끌어다 놓기 · 단축키
let dragDepth = 0;
const hasText = (e) => [...(e.dataTransfer?.types || [])].some((t) => t === 'text/uri-list' || t === 'text/plain');
window.addEventListener('dragenter', (e) => { if (!hasText(e)) return; e.preventDefault(); dragDepth++; $('#dropVeil').hidden = false; });
window.addEventListener('dragover', (e) => { if (hasText(e)) e.preventDefault(); });
window.addEventListener('dragleave', () => { if (--dragDepth <= 0) { dragDepth = 0; $('#dropVeil').hidden = true; } });
window.addEventListener('drop', (e) => {
  dragDepth = 0; $('#dropVeil').hidden = true;
  const t = e.dataTransfer.getData('text/uri-list') || e.dataTransfer.getData('text/plain');
  if (!t) return;
  e.preventDefault();
  if (S.step !== 1) setStep(1);
  addText(t.split('\n').filter((l) => !l.startsWith('#')).join('\n'));
});

document.addEventListener('paste', (e) => {
  if (e.target.id === 'urls' || M.open) return;
  const t = e.clipboardData.getData('text');
  if (!extractUrls(t).length) return;
  e.preventDefault();
  if (S.step !== 1) setStep(1);
  addText(t);
});

document.addEventListener('keydown', (e) => {
  if (M.open) { if (e.key === 'Escape') { e.preventDefault(); closeModal(); } return; }
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
    e.preventDefault();
    if (S.step === 1 && !$('#nextBtn').disabled) goStep2();
    else if (S.step === 2 && !$('#startBtn').disabled) $('#startBtn').click();
    return;
  }
  if (e.key === 'Enter' && S.step === 2 && e.target.tagName !== 'SELECT' && !$('#startBtn').disabled) { e.preventDefault(); $('#startBtn').click(); }
  if (e.key === 'Escape' && S.step === 2) $('#backBtn').click();
});

// ════════════════════════════════════════════════ 주기적 갱신
let fails = 0;
async function poll() {
  if (ended || U.restarting) return;
  try {
    const p = await api('/api/poll');
    fails = 0;
    S.jobs = p.jobs; S.tools = p.tools;
    renderSetup(p.tools);
    renderJobs(p.jobs);
    onUpdateState(p.update);
    if (settingsRender && M.open) {
      const sig = JSON.stringify([p.tools.engineVersion, p.tools.engineUpdating, p.tools.engineMessage, p.update.autoUpdate]);
      if (sig !== poll.sig) { poll.sig = sig; settingsRender(); }
    }
  } catch {
    if (++fails >= 6) showEnded('YTROAD 엔진이 종료되었어요');
  }
}

// ════════════════════════════════════════════════ 시작
(async function init() {
  paintArt();
  if (!T) { showEnded('YTROAD 앱에서 열어 주세요'); return; }
  try { S.settings = await api('/api/settings'); }
  catch { showEnded('앱 엔진에 연결하지 못했어요'); return; }
  setPlatform(S.settings.os);
  applyTheme(S.settings.appearance);
  S.folder = S.settings.resolvedDefaultFolder;
  S.fmt = S.settings.fmt; S.quality = S.settings.quality;
  $('#verText').textContent = 'v' + S.settings.build;
  renderFolder(); paintOptions(); paintParallel(); paintTabs();
  setStep(1);
  await poll();
  setInterval(poll, 500);
  $('#urls').focus();
})();
