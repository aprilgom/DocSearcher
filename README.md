# DocSearcher

## 개요
HWP/PDF 파일을 인덱싱하고 웹 UI에서 검색하는 Go 기반 문서 검색기입니다.

이 프로젝트는 Python/PyQt와 Elasticsearch로 만들었던 `HwpPdfSearcher`를 사용하며 느낀 실행 환경과 구조상의 불편을 개선하기 위해 다시 만든 버전입니다. 주변에서 문서 내용을 찾는 데 걸리는 시간을 줄이는 것이 목적이었고, 서버와 클라이언트를 나누고 로컬 검색 인덱스를 사용하도록 구조를 단순화했습니다.

## 주요 내용
- 감시 폴더의 `.hwp`, `.hwpx`, `.pdf` 문서 텍스트 추출
- Bleve 인덱스 생성 및 검색
- 웹 서버와 WebView 클라이언트 분리
- 설정 파일(`config.json`)로 감시 경로 관리 (`config.example.json` 참고)

## 기술 스택
- Go 1.24.3
- Bleve
- fsnotify
- go-webview2
- ledongthuc/pdf
- 로컬 모듈 `goHwpTxt`

## 실행 방법
서버:

```bash
go run ./cmd/app
```

기본 서버 주소는 `http://localhost:8080`입니다. WebView 클라이언트는 `server.txt`의 URL을 읽습니다.

```bash
go run ./cmd/client
```

감시 경로는 실행 중 웹 UI에서 추가하거나, `config.example.json`을 `config.json`으로 복사해 로컬 환경에 맞게 수정합니다. `config.json`, 검색 인덱스(`hwp-index.bleve/`), 테스트용 실제 문서(`goHwpTxt/testdata/`)는 공개 저장소에 포함하지 않습니다.

## 구조/참고
- `cmd/app`: 검색 서버 실행 진입점
- `cmd/client`: Windows WebView 클라이언트
- `internal/indexer`: 파일 순회 및 인덱싱
- `internal/parser`: HWP/PDF 텍스트 추출
- `internal/search`: Bleve 검색 엔진
- `web/templates`: 검색 화면 템플릿

## 개선한 점
- `HwpPdfSearcher`의 Python GUI 중심 구조를 Go 서버와 WebView 클라이언트 구조로 분리했습니다.
- 외부 Elasticsearch 실행 의존을 줄이고 Bleve 기반 로컬 인덱스를 사용했습니다.
- 문서 파싱, 인덱싱, 검색, 화면 표시 역할을 패키지별로 나눠 유지보수하기 쉽게 정리했습니다.
