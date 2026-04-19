# Nexos

**MQTT 토픽 하나만 맞추면 대시보드가 저절로 만들어지는 셀프호스팅 IoT 모니터링 플랫폼.**

ESP32를 Nexos에 붙이면 차트가 뜹니다. 센서를 하나 더 추가하면 차트가 하나 더 뜹니다.
YAML도, 대시보드 에디터 조작도, 코드 수정도 없습니다 — 토픽 규약 하나면 끝.

> English README: [README.md](README.md)

<p align="center">
  <img src="docs/public/demo.gif" alt="Nexos 데모 — MQTT publish에서 자동 차트 생성까지" width="720" />
</p>

---

## 왜 Nexos인가

Grafana + InfluxDB + Mosquitto 스택은 훌륭한 부품들을 주지만, 조립은 전부
개발자의 몫입니다. 스키마 설계, Telegraf 설정, 대시보드 레이아웃, 알림
배선 — 센서 하나 추가할 때마다 매번 반복해야 하죠.

Nexos는 그 모든 단계를 **토픽 규약 하나**로 압축합니다.

```
devices/{device_id}/{sensor}   {"value": 23.5}
```

이 토픽으로 publish하면 카드가 생깁니다. 드래그로 배치하고, 임계값을
설정하고, 초과 시 웹훅을 받으세요. 그게 전부입니다.

## 핵심 기능

- **Topic Auto-Discovery** — 신규 `device_id` 혹은 `sensor`가 감지되면 대시보드 카드 자동 생성
- **실시간 차트** — Go ingestion이 WebSocket으로 직접 fan-out
- **드래그 앤 드롭 레이아웃** — gridstack.js, 브라우저 localStorage에 저장
- **과거 데이터 조회** — TimescaleDB hypertable, 1시간 이상 범위는 자동 1분 버킷 다운샘플링
- **임계값 알림 → Generic JSON 웹훅** — Slack, n8n, 자체 서버 어디든 연결 가능
- **단일 바이너리 + Docker Compose** — Kubernetes 불필요, 클라우드 의존 없음
- **TLS + 디바이스별 ACL 기본 적용** — 각 디바이스는 자신의 토픽만 publish 가능
- **MIT 라이선스, 전화집(phone-home) 없음, 계정 등록 없음**

## 5분 퀵스타트

**요구사항:** Docker 24+ (Compose v2 포함), 호스트에 `openssl`, `bash`, RAM 2GB 여유

```bash
git clone https://github.com/sanchez0523/nexos nexos
cd nexos

# 인터랙티브 셋업 — .env, TLS 인증서, 브로커 passwd 파일을 자동 생성합니다
./scripts/setup.sh

# 전체 기동
docker compose up -d

# 브라우저에서 열기 (로컬 CA 경고 수락)
open https://localhost
```

디바이스 등록:

```bash
./scripts/add-device.sh esp32-living
# → username, password, CA 인증서 경로, 토픽 prefix 출력
```

ESP32, 라즈베리파이, 내장된 Go 시뮬레이터 중 하나로 브로커를 찌르면 됩니다.
한 파일짜리 예제 코드는 [examples/](examples/) 참고.

## 아키텍처

```
 IoT 디바이스 ── MQTT/TLS ─▶ Mosquitto ──▶ Go ingestion ──▶ TimescaleDB
                                                │
                                                ├── WebSocket ──▶ SvelteKit 대시보드
                                                └── Webhook dispatch ──▶ 외부 알림 시스템
```

서비스 5개, 전부 `docker compose`로 관리:

| 서비스      | 역할                                                   |
|------------|--------------------------------------------------------|
| `broker`   | Mosquitto 2.x, TLS 1.2+, passwd + ACL                  |
| `db`       | TimescaleDB (PostgreSQL 16 + 확장)                     |
| `ingestion`| Go + Fiber — MQTT 구독, REST API, WebSocket Hub, 알림 엔진 |
| `dashboard`| SvelteKit static 빌드 (SPA, `serve`로 서빙)            |
| `proxy`    | Caddy — HTTPS 종단, `/api`, `/ws`, `/` 라우팅          |

각 선택의 근거는 [ARCHITECTURE.md](ARCHITECTURE.md)의 ADR에서 확인할 수 있습니다.

## 토픽 & 페이로드 규약

- **토픽:** 반드시 `devices/{device_id}/{sensor}` 형식. 다른 토픽은 조용히 드롭됩니다.
- **페이로드:** JSON 오브젝트 `{"value": <number>}` 또는 숫자 하나. 이외는 드롭.

이 제약이 Auto-Discovery를 가능하게 만든 핵심 디자인입니다.

## 알림

대시보드 `/alerts`에서 생성하거나 REST로 직접:

```bash
curl -b cookies.txt https://localhost/api/alerts \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "esp32-living",
    "sensor": "temperature",
    "condition": "above",
    "threshold": 30,
    "webhook_url": "https://hooks.example.com/t/xxxx",
    "enabled": true
  }'
```

임계값 초과 시 Nexos가 POST하는 페이로드:

```json
{
  "device_id": "esp32-living",
  "sensor": "temperature",
  "value": 31.2,
  "threshold": 30,
  "condition": "above",
  "triggered_at": "2026-04-18T10:30:00Z"
}
```

`ALERT_TIMEOUT_SECONDS`로 룰별 쿨다운을 강제해서 펄럭이는 센서가 수신처를
스팸으로 도배하는 걸 막습니다.

## Nexos가 일부러 하지 않는 것

작게 유지하려는 의도입니다. v1 범위 밖:

- **디바이스 제어 / 명령 전송** — 모니터링 전용. `POST /devices/{id}/command` 같은 건 없습니다.
- **멀티 유저 / RBAC** — 설치당 관리자 한 명.
- **메시지 보장 전송** — 일부 손실 허용. Redis Streams나 Kafka 없음.
- **토큰 게이팅 / 라이선스 키 / phone-home 텔레메트리** — MIT 오픈소스, 로컬 실행, 외부 서버에 절대 연락 안 함.
- **바이너리 / Protobuf 페이로드** — JSON 전용.

이런 게 꼭 필요하면 Nexos는 맞지 않습니다. 괜찮아요 — 모든 걸 담는 도구가
되느니 명확히 "아니요"라고 말하는 쪽을 택했습니다.

## 기여하기

모든 기여는 [CONTRIBUTING.md](CONTRIBUTING.md)를 읽고 시작해주세요. 특히
ARCHITECTURE.md의 "Architecture Invariants" 섹션에 정의된 불변 조건을 위반하는
PR은 merge하지 않습니다.

### 한국 개발자 커뮤니티

- **GeekNews / velog / OKKY** — 쇼케이스 환영합니다. 설치 스크린샷
  GitHub Issue로 공유해주세요.
- 질문은 GitHub Discussions로 (영어/한국어 모두 OK).

## 라이선스

MIT — [LICENSE](LICENSE) 참고.
