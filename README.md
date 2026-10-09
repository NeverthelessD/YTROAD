# YTROAD

유튜브 영상 · 음악 다운로더 (macOS) — made by. Nevertheless_D

주소를 붙여넣고 형식만 고르면 내 맥에 바로 저장해요. 개인적인 용도로만 사용해 주세요.

- **형식**: MP4(영상 + 소리, 최고 · 4K · 1440p · 1080p · 720p · 480p) · MP3(320kbps) · M4A(원본 음질) · WAV(무손실)
- **여러 개 동시에**: 한 줄에 하나씩 붙여넣으면 최대 **10개**를 동시에 받아요 (설정에서 1~10개 조절)
- **진행 현황**: 전체 속도 그래프 · 영상별 속도/남은 시간 · 취소 · 다시 시도 · 먼저 받기
- **저장 위치**: 이번에만 다른 폴더 고르기 · 기본 폴더 지정
- **화면**: 라이트 / 다크 / 시스템 설정 따르기(기본)
- **자동 업데이트**: 앱은 이 저장소의 `releases/latest.json`을, 다운로드 엔진(yt-dlp)은 하루에 한 번 스스로 업데이트해요. 오른쪽 위 버전 배지에서 수동 확인 · 이전 버전 되돌리기도 돼요.

## 설치

1. [`releases/`](releases) 폴더에서 가장 최신 `YTROAD-x.y.z.zip`을 받아 압축을 풀어요.
2. `YTROAD.app`을 **응용 프로그램** 폴더로 옮겨요. (업데이트가 동작하려면 꼭 옮겨 주세요)
3. 처음 열 때 경고가 뜨면 **시스템 설정 → 개인정보 보호 및 보안 → "그래도 열기"**.
4. 처음 한 번 다운로드 도구(약 200MB)를 자동으로 받아요. 예전 YT Downloader를 쓰던 맥이면 그 도구를 가져와서 금방 끝나요.

모든 구성요소는 `~/Library/Application Support/YTROAD` 한 폴더에만 설치돼요.
지우려면 앱을 휴지통에 버리고 그 폴더도 지우면 끝이에요.

## 사용한 오픈소스

- [yt-dlp](https://github.com/yt-dlp/yt-dlp) (Unlicense) — 다운로드 엔진
- [FFmpeg](https://ffmpeg.org) ([martin-riedl.de](https://ffmpeg.martin-riedl.de) 빌드) — 합치기 · 변환
- [Deno](https://deno.com) (MIT) — yt-dlp가 유튜브 보안 확인에 사용
- 글꼴: [Pretendard](https://github.com/orioncactus/pretendard), [Unbounded](https://github.com/googlefonts/unbounded) (SIL OFL)

도구들은 앱에 들어 있지 않고, 처음 실행할 때 각 배포처에서 직접 받아요.

## 폴더 구조

```
engine/     Go 엔진 (로컬 서버 · 다운로드 대기열 · 도구 설치 · 업데이트)
engine/web/ 앱 화면 (HTML · CSS · JS · 일러스트)
app/        YTROAD.app 껍데기 (실행 스크립트 · Info.plist · 아이콘)
scripts/    빌드 스크립트
releases/   업데이트 파일 (YTROAD-x.y.z.zip, latest.json)
docs/       사용 가이드 이미지
```
