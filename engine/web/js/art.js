// YTROAD 일러스트 (SVG) — 색은 CSS 변수(라이트/다크)를 따라가요
window.ART = {
  hero: `
<svg viewBox="0 0 400 290" class="art art-hero" aria-hidden="true">
  <defs>
    <linearGradient id="hSky" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="#ff5a36"/><stop offset=".6" stop-color="#ff0033"/><stop offset="1" stop-color="#a3001c"/></linearGradient>
    <linearGradient id="hHill" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#2a0710" stop-opacity=".55"/><stop offset="1" stop-color="#2a0710" stop-opacity=".85"/></linearGradient>
    <filter id="hSh" x="-20%" y="-20%" width="140%" height="150%"><feDropShadow dx="0" dy="10" stdDeviation="12" flood-color="#000" flood-opacity=".28"/></filter>
  </defs>
  <!-- road -->
  <path class="a-road" d="M70 270 C 120 225, 210 250, 250 205 S 330 150, 360 120" fill="none" stroke-width="26" stroke-linecap="round"/>
  <path class="a-lane" d="M70 270 C 120 225, 210 250, 250 205 S 330 150, 360 120" fill="none" stroke-width="3" stroke-dasharray="10 12" stroke-linecap="round"/>
  <!-- player window -->
  <g filter="url(#hSh)" class="a-float">
    <rect x="22" y="22" width="250" height="168" rx="16" class="a-panel"/>
    <circle cx="40" cy="38" r="4" fill="#ff5f57"/><circle cx="53" cy="38" r="4" fill="#febc2e"/><circle cx="66" cy="38" r="4" fill="#28c840"/>
    <rect x="34" y="52" width="226" height="110" rx="10" fill="url(#hSky)"/>
    <circle cx="214" cy="80" r="13" fill="#ffd36e" opacity=".9"/>
    <path d="M34 140 L84 104 L118 128 L156 92 L214 140 L260 116 L260 152 Q260 162 250 162 L44 162 Q34 162 34 152 Z" fill="url(#hHill)"/>
    <circle cx="147" cy="107" r="24" fill="#fff" opacity=".95"/>
    <path d="M140 95 L160 107 L140 119 Z" fill="#ff0033"/>
    <rect x="34" y="171" width="226" height="5" rx="2.5" class="a-track"/>
    <rect x="34" y="171" width="150" height="5" rx="2.5" fill="#ff0033" class="a-prog"/>
    <circle cx="184" cy="173.5" r="6.5" fill="#ff0033" class="a-knob"/>
  </g>
  <!-- flying download arrow -->
  <g class="a-bob">
    <circle cx="300" cy="78" r="26" fill="#ff0033"/>
    <path d="M300 64 L300 90 M289 80 L300 91 L311 80" fill="none" stroke="#fff" stroke-width="5" stroke-linecap="round" stroke-linejoin="round"/>
  </g>
  <!-- files arriving at the end of the road -->
  <g filter="url(#hSh)">
    <g transform="translate(318 150) rotate(8)">
      <rect width="62" height="76" rx="10" class="a-paper"/>
      <rect x="10" y="12" width="42" height="28" rx="5" fill="#ff0033" opacity=".9"/>
      <path d="M27 19 L37 26 L27 33 Z" fill="#fff"/>
      <rect x="10" y="48" width="30" height="5" rx="2.5" class="a-line"/><rect x="10" y="58" width="22" height="5" rx="2.5" class="a-line"/>
      <rect x="34" y="62" width="34" height="16" rx="8" fill="#111"/><text x="51" y="73.5" text-anchor="middle" font-size="9" font-weight="800" fill="#fff" font-family="Unbounded,Pretendard,sans-serif">MP4</text>
    </g>
    <g transform="translate(262 196) rotate(-7)">
      <rect width="58" height="70" rx="10" class="a-paper"/>
      <circle cx="29" cy="28" r="15" fill="#8a6cff" opacity=".9"/>
      <path d="M25 36 V20 L36 17 V31" fill="none" stroke="#fff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/>
      <circle cx="22.5" cy="36" r="3.6" fill="#fff"/><circle cx="33.5" cy="31.5" r="3.6" fill="#fff"/>
      <rect x="10" y="52" width="26" height="5" rx="2.5" class="a-line"/>
      <rect x="30" y="56" width="34" height="16" rx="8" fill="#111"/><text x="47" y="67.5" text-anchor="middle" font-size="9" font-weight="800" fill="#fff" font-family="Unbounded,Pretendard,sans-serif">MP3</text>
    </g>
  </g>
  <g class="a-spark"><path d="M372 52 l3 8 8 3 -8 3 -3 8 -3 -8 -8 -3 8 -3z" fill="#ffb020"/><path d="M246 32 l2 5 5 2 -5 2 -2 5 -2 -5 -5 -2 5 -2z" fill="#ffb020" opacity=".8"/></g>
</svg>`,

  setup: `
<svg viewBox="0 0 400 290" class="art art-setup" aria-hidden="true">
  <defs><filter id="sSh" x="-20%" y="-20%" width="140%" height="150%"><feDropShadow dx="0" dy="10" stdDeviation="12" flood-color="#000" flood-opacity=".25"/></filter></defs>
  <path class="a-road" d="M40 262 C 140 230, 260 262, 370 214" fill="none" stroke-width="24" stroke-linecap="round"/>
  <path class="a-lane" d="M40 262 C 140 230, 260 262, 370 214" fill="none" stroke-width="3" stroke-dasharray="10 12" stroke-linecap="round"/>
  <g filter="url(#sSh)">
    <!-- box -->
    <path d="M120 120 L210 92 L300 120 L300 220 L210 248 L120 220 Z" class="a-panel"/>
    <path d="M120 120 L210 148 L300 120" fill="none" class="a-edge" stroke-width="2"/>
    <path d="M210 148 L210 248" class="a-edge" stroke-width="2"/>
    <path d="M120 120 L210 148 L210 248 L120 220 Z" class="a-shade"/>
    <path d="M150 111 L240 139 L240 160 L222 154 L222 144 L150 121 Z" fill="#ff0033" opacity=".9"/>
  </g>
  <g class="a-bob">
    <circle cx="210" cy="56" r="26" fill="#ff0033"/>
    <path d="M210 42 L210 68 M199 58 L210 69 L221 58" fill="none" stroke="#fff" stroke-width="5" stroke-linecap="round" stroke-linejoin="round"/>
  </g>
  <g class="a-spin" style="transform-origin:318px 78px">
    <path d="M318 56 l5 0 2 7 6 3 6-4 4 4 -4 6 3 6 7 2 0 5 -7 2 -3 6 4 6 -4 4 -6-4 -6 3 -2 7 -5 0 -2-7 -6-3 -6 4 -4-4 4-6 -3-6 -7-2 0-5 7-2 3-6 -4-6 4-4 6 4 6-3z" fill="#ffb020"/>
    <circle cx="320.5" cy="80.5" r="7" class="a-hole"/>
  </g>
  <g class="a-spin rev" style="transform-origin:96px 92px">
    <path d="M96 76 l4 0 1.5 5 4.5 2 4.5-3 3 3 -3 4.5 2 4.5 5 1.5 0 4 -5 1.5 -2 4.5 3 4.5 -3 3 -4.5-3 -4.5 2 -1.5 5 -4 0 -1.5-5 -4.5-2 -4.5 3 -3-3 3-4.5 -2-4.5 -5-1.5 0-4 5-1.5 2-4.5 -3-4.5 3-3 4.5 3 4.5-2z" fill="#2bc5e8"/>
    <circle cx="98" cy="94" r="5" class="a-hole"/>
  </g>
</svg>`,

  empty: `
<svg viewBox="0 0 220 150" class="art art-empty" aria-hidden="true">
  <g class="a-film">
    <rect x="20" y="38" width="120" height="74" rx="10" transform="rotate(-8 80 75)" class="a-panel2"/>
    <g transform="rotate(-8 80 75)">
      <rect x="20" y="38" width="120" height="12" rx="6" class="a-strip"/><rect x="20" y="100" width="120" height="12" rx="6" class="a-strip"/>
      <g class="a-hole2"><rect x="28" y="41" width="8" height="6" rx="1.5"/><rect x="44" y="41" width="8" height="6" rx="1.5"/><rect x="60" y="41" width="8" height="6" rx="1.5"/><rect x="76" y="41" width="8" height="6" rx="1.5"/><rect x="92" y="41" width="8" height="6" rx="1.5"/><rect x="108" y="41" width="8" height="6" rx="1.5"/><rect x="124" y="41" width="8" height="6" rx="1.5"/>
      <rect x="28" y="103" width="8" height="6" rx="1.5"/><rect x="44" y="103" width="8" height="6" rx="1.5"/><rect x="60" y="103" width="8" height="6" rx="1.5"/><rect x="76" y="103" width="8" height="6" rx="1.5"/><rect x="92" y="103" width="8" height="6" rx="1.5"/><rect x="108" y="103" width="8" height="6" rx="1.5"/><rect x="124" y="103" width="8" height="6" rx="1.5"/></g>
      <circle cx="80" cy="75" r="15" fill="#ff0033"/><path d="M75 67 L88 75 L75 83 Z" fill="#fff"/>
    </g>
  </g>
  <g transform="translate(130 40)">
    <circle cx="18" cy="22" r="12" fill="#fff6dc"/><circle cx="34" cy="16" r="13" fill="#fffaf0"/><circle cx="50" cy="22" r="12" fill="#fff3d0"/><circle cx="28" cy="28" r="11" fill="#ffe9b3"/><circle cx="44" cy="30" r="10" fill="#fff6dc"/>
    <path d="M8 30 L60 30 L54 98 L14 98 Z" fill="#fff"/>
    <path d="M14 30 L20 98 M26 30 L29 98 M42 30 L39 98 M54 30 L48 98" stroke="#ff0033" stroke-width="7"/>
    <path d="M8 30 L60 30 L54 98 L14 98 Z" fill="none" stroke="rgba(0,0,0,.08)" stroke-width="1.5"/>
  </g>
</svg>`,

  folder: `<svg viewBox="0 0 48 40" width="44" height="36" aria-hidden="true"><path d="M4 8 Q4 4 8 4 L18 4 L22 9 L40 9 Q44 9 44 13 L44 32 Q44 36 40 36 L8 36 Q4 36 4 32 Z" fill="#ffb020"/><path d="M4 15 Q4 12 7 12 L41 12 Q44 12 44 15 L44 32 Q44 36 40 36 L8 36 Q4 36 4 32 Z" fill="#ffc94d"/><path d="M24 17 L24 29 M19 24.5 L24 29.5 L29 24.5" stroke="#ff0033" stroke-width="3" fill="none" stroke-linecap="round" stroke-linejoin="round"/></svg>`,

  mp4: `<svg viewBox="0 0 40 40" width="34" height="34" aria-hidden="true"><rect x="3" y="7" width="34" height="26" rx="6" fill="#ff0033"/><g fill="#fff" opacity=".55"><rect x="6" y="9.5" width="4" height="3" rx="1"/><rect x="13" y="9.5" width="4" height="3" rx="1"/><rect x="20" y="9.5" width="4" height="3" rx="1"/><rect x="27" y="9.5" width="4" height="3" rx="1"/><rect x="6" y="27.5" width="4" height="3" rx="1"/><rect x="13" y="27.5" width="4" height="3" rx="1"/><rect x="20" y="27.5" width="4" height="3" rx="1"/><rect x="27" y="27.5" width="4" height="3" rx="1"/></g><path d="M16.5 15 L25 20 L16.5 25 Z" fill="#fff"/></svg>`,
  mp3: `<svg viewBox="0 0 40 40" width="34" height="34" aria-hidden="true"><circle cx="20" cy="20" r="17" fill="#8a6cff"/><path d="M17 27 V12 L28 9.5 V24" fill="none" stroke="#fff" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"/><circle cx="14.5" cy="27" r="3.8" fill="#fff"/><circle cx="25.5" cy="24" r="3.8" fill="#fff"/></svg>`,
  m4a: `<svg viewBox="0 0 40 40" width="34" height="34" aria-hidden="true"><circle cx="20" cy="20" r="17" fill="#2bc5e8"/><path d="M10.5 24 V20 a9.5 9.5 0 0 1 19 0 V24" fill="none" stroke="#fff" stroke-width="3" stroke-linecap="round"/><rect x="8.5" y="21.5" width="6" height="9" rx="2.5" fill="#fff"/><rect x="25.5" y="21.5" width="6" height="9" rx="2.5" fill="#fff"/></svg>`,
  wav: `<svg viewBox="0 0 40 40" width="34" height="34" aria-hidden="true"><rect x="3" y="3" width="34" height="34" rx="9" fill="#2fd39a"/><g stroke="#fff" stroke-width="3" stroke-linecap="round"><path d="M9 20 V20.5"/><path d="M14 15 V25"/><path d="M19 10 V30"/><path d="M24 14 V26"/><path d="M29 17.5 V22.5"/><path d="M33 20 V20.5"/></g></svg>`,
};

window.paintArt = function (root = document) {
  root.querySelectorAll('[data-art]').forEach((el) => { if (!el.dataset.painted && ART[el.dataset.art]) { el.innerHTML = ART[el.dataset.art]; el.dataset.painted = 1; } });
  root.querySelectorAll('[data-icon]').forEach((el) => { if (!el.dataset.painted && ART[el.dataset.icon]) { el.innerHTML = ART[el.dataset.icon]; el.dataset.painted = 1; } });
};
