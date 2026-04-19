# velog — 한국어 런칭 글

## 제목 (검색 최적화용 후보 3개)

1. "MQTT 토픽 규약 하나로 대시보드가 저절로 만들어지는 IoT 플랫폼을 만들었습니다 — Nexos"
2. "셀프호스팅 IoT 모니터링 스택을 5주 만에 만든 이야기 (Go + SvelteKit + TimescaleDB)"
3. "왜 또 IoT 대시보드를 만들었냐면요 — Grafana가 싫어서는 아닙니다"

추천: 1번. 차별점이 제목 안에 들어가서 GeekNews 크로스포스트에도 강함.

## 태그

```
iot, go, golang, sveltekit, self-hosted, mqtt, timescaledb, opensource
```

## 본문

```markdown
## TL;DR

- IoT 센서 데이터를 받으면 대시보드가 **자동으로** 구성되는 셀프호스팅
  모니터링 플랫폼을 만들었습니다.
- Docker Compose 한 번으로 기동, MIT 라이선스, 텔레메트리 없음.
- 저장소: https://github.com/OWNER/REPO
- 한국어 README: https://github.com/OWNER/REPO/blob/main/README.ko.md

![Nexos 데모](https://raw.githubusercontent.com/OWNER/REPO/main/docs/public/demo.gif)

## 왜 만들었는가

IoT 사이드 프로젝트를 할 때마다 Grafana + InfluxDB + Mosquitto 스택을
조립했습니다. 부품은 훌륭한데, 조립은 매번 반복입니다:

1. InfluxDB 스키마 고민
2. Telegraf/Mosquitto 연동 설정
3. Grafana 들어가서 대시보드 빌드
4. 센서 하나 늘리면 → 3번 다시
5. 알림 룰 넣으려면 → Alertmanager 별도 배선

"센서 하나 늘리는데 왜 매번 UI를 열어야 하지?"가 출발점이었습니다.

## 해결 방식 — 규약 하나

Nexos는 딱 하나의 제약만 요구합니다.

```
devices/{device_id}/{sensor}    {"value": 23.5}
```

이 토픽 규약만 지키면:
- 새 `device_id`가 감지되면 디바이스 목록에 자동 추가
- 새 `sensor`가 감지되면 대시보드에 카드 자동 생성
- 드래그 앤 드롭으로 배치, 브라우저 localStorage에 레이아웃 저장
- 임계값 룰 → Generic JSON 웹훅 발송

코드 변경도, 설정 파일도, Grafana 에디터 조작도 없습니다.

## 아키텍처

5개 서비스로 정리했습니다.

| 서비스       | 기술                          |
|-------------|-------------------------------|
| MQTT Broker | Mosquitto 2.x (TLS + ACL)    |
| Ingestion   | Go 1.22 + Fiber v2           |
| DB          | TimescaleDB (PostgreSQL 16)  |
| Dashboard   | SvelteKit + Tailwind         |
| Proxy       | Caddy (자동 HTTPS)           |

Redis, Kafka, Kubernetes 없습니다. "50대 디바이스 규모에서 필요 없는 건
안 넣는다"가 설계 철칙이었습니다.

## 개인적으로 배운 것

### Go 채널 기반 fan-out

MQTT 메시지를 받으면 3군데로 보내야 했습니다 — DB 저장, WebSocket
브로드캐스트, 알림 엔진. 전통적 답은 Redis Pub/Sub인데, 규모가 안
나오는데 Redis를 넣으면 운영 포인트만 늘어납니다.

결국 buffered Go channel 하나로 fan-out 하고, DB 쓰기만 블로킹 전송,
WebSocket과 알림은 논블로킹 (버퍼 차면 드롭) 으로 구성했습니다. 트레이드
오프를 명시적으로 결정한 게 포인트입니다.

### TimescaleDB 단일 hypertable

센서별로 테이블을 나누고 싶은 유혹이 있었지만, 토픽이 동적으로 늘어나는
Auto-Discovery 설계상 DDL을 런타임에 실행해야 합니다. 그냥 단일 `metrics`
테이블에 `(device_id, sensor, time DESC)` 복합 인덱스로 해결했고, 1시간
이상 조회는 `time_bucket('1 minute', ...)`로 자동 다운샘플링합니다.

### WebSocket + httpOnly 쿠키

처음엔 JWT를 localStorage에 넣고 WebSocket URL에 `?token=...` 박는
일반적인 방식을 쓰려 했습니다. 그런데:
- XSS로 토큰 탈취 위험
- URL에 JWT가 액세스 로그에 찍힘

대신 httpOnly + SameSite=Strict 쿠키로 통일했습니다. WebSocket 핸드셰이크도
HTTP 요청이라 쿠키가 자동 전송됩니다. REST와 동일한 인증 경로.

## 퀵스타트

```bash
git clone https://github.com/OWNER/REPO nexos
cd nexos
./scripts/setup.sh           # .env + TLS 인증서 + 브로커 passwd 자동 생성
docker compose up -d
./scripts/add-device.sh esp32-living
```

ESP32 예제, 라즈베리파이 Python 예제, Go 시뮬레이터 모두 저장소 `examples/`
하위에 있습니다. 하드웨어 없어도 시뮬레이터로 바로 체험 가능합니다.

## 일부러 하지 않은 것

- **디바이스 제어 (단방향 only)** — 서버에서 디바이스로 publish 안 함
- **멀티 유저** — 관리자 한 명이 쓰는 도구
- **메시지 100% 보장** — 허용 가능한 유실 있음
- **phone-home / 라이선스 게이팅** — MIT 오픈소스, 외부 서버 호출 0건

작게 유지하는 게 이 프로젝트의 핵심입니다. 필요한 기능이 빠져 있다면
Issue/Discussions에서 설득해주세요.

## 앞으로

v1 릴리즈 직후 반응 보면서 결정할 목록:
- Prometheus 메트릭 export (요청이 많으면)
- i18n (영어/한국어/일본어)
- ESP8266, LoRa 게이트웨이 예제 추가

피드백 환영합니다. GitHub Discussions든, 이 글 댓글이든, 트위터든
편한 곳에서.

저장소: https://github.com/OWNER/REPO
```

## 게시 체크리스트

- [ ] velog 썸네일에 `demo.gif`의 첫 프레임 스크린샷 사용
- [ ] GeekNews에도 링크 공유 (`https://news.hada.io`)
- [ ] 첫 6시간 내 댓글 전부 응답
- [ ] OKKY 질문 게시판에 "IoT 모니터링 툴을 만들었는데..." 형태로
      의견 요청 글 별도 작성 (velog 링크 자연 유입 유도)
