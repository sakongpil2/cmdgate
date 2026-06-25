# CmdGate — Agent Design Document

> 이 문서는 CmdGate 프로젝트의 설계 방향과 에이전트 작업 규칙을 정의합니다.  
> **마지막 규칙:** 사용자에게 본 프로젝트와 관련해 답변할 때는 항상 한국어로 응답합니다.

## 1. 프로젝트 목표

CmdGate는 운영자가 사전에 허용된 명령만 위임 권한으로 실행할 수 있도록 하는 allowlist 기반 CLI 도구입니다.

- 사용자용 바이너리: `cmdgate`
- 권한 실행기: `cmdgate-exec`
- 정책 파일: `allowlist.yaml`
- 설치 경로: `/opt/cmdgate`
- 로그 경로: `/var/log/cmdgate`

`cmdgate`는 사용자 입력을 받아 `sudo -n`을 통해 `cmdgate-exec`를 호출하고, 실제 명령 검증과 실행은 `cmdgate-exec`가 수행합니다.

## 2. 구현 원칙

1. Go 바이너리 2개로 물리 분리합니다.
2. Go 코드는 최대한 간결하게 작성합니다.
3. daemon, socket, API server, plugin 구조는 사용하지 않습니다.
4. shell 문자열 실행, `bash -c`, `sh -c`, `eval`을 사용하지 않습니다.
5. 검증 통과 후 argv 배열 방식으로 실행합니다.
6. `cmdgate-exec`에만 위임 권한 실행 로직을 둡니다.
7. **TDD(Test-Driven Development)**로 개발합니다. 모든 동작은 먼저 실패하는 테스트를 작성한 뒤, 최소한의 코드로 통과시킵니다.
8. 문서(README.md, 설계 문서, 주석)를 꼼꼼히 작성하여 다른 사람/에이전트가 이어받을 수 있도록 합니다.

## 3. 디렉터리 구조

```text
/srv/cmdgate/
├── AGENTS.md                 # 본 문서 (설계 및 작업 규칙)
├── README.md                 # 프로젝트 사용법 및 개요
├── go.mod                    # Go 모듈 정의
├── cmd/
│   ├── cmdgate/
│   │   └── main.go           # 사용자용 CLI
│   └── cmdgate-exec/
│       └── main.go           # sudo-only 권한 실행기
├── internal/
│   ├── allowlist/
│   │   └── allowlist.go      # allowlist.yaml 파싱 및 명령 검색
│   ├── matchers/
│   │   ├── rpmfiles.go       # rpmFiles matcher
│   │   └── number.go         # number matcher
│   ├── runner/
│   │   └── runner.go         # argv 기반 명령 실행
│   ├── audit/
│   │   └── audit.go          # 감사 로그 기록
│   └── policy/
│       └── policy.go         # 정책 bundle 검증/반영
├── scripts/
│   └── install-cmdgate.sh    # 설치 스크립트
├── configs/
│   └── allowlist.yaml        # 기본 allowlist 예시
└── docs/
    └── ...                   # 추가 설계/가이드 문서
```

## 4. 바이너리 책임

### 4.1 cmdgate

- `cmdgate run <command> [args...]`
- `cmdgate run list`
- `cmdgate policy validate --bundle <tar.gz>`
- `cmdgate policy apply --bundle <tar.gz>`
- 사용자 입력을 검증하지 않고 `cmdgate-exec`에 그대로 전달합니다.
- `sudo -n /opt/cmdgate/cmdgate-exec <subcommand> [args...]` 형태로 호출합니다.

### 4.2 cmdgate-exec

- `cmdgate-exec run <command> [args...]`
- `cmdgate-exec run list`
- `cmdgate-exec policy validate --bundle <tar.gz>`
- `cmdgate-exec policy apply --bundle <tar.gz>`
- `/opt/cmdgate/allowlist.yaml`을 읽어 사용자 입력과 비교합니다.
- matcher placeholder가 있으면 matcher 검증을 수행합니다.
- 검증 성공 시 argv 배열 방식으로 실행합니다.
- 실행 결과를 `/var/log/cmdgate/audit.log`에 기록합니다.

## 5. 핵심 설계 결정

### 5.1 명령 실행 (argv-only)

- `allowlist.yaml`의 `commands[].cmd`를 파싱하여 argv 슬라이스로 변환합니다.
- 사용자 입력 argv와 정책 argv를 순차 비교합니다.
- `<type:name>` placeholder는 matcher에 위임하여 검증합니다.
- 실행 시 `exec.Command(cmd, args...)` 형태로 실행합니다.

### 5.2 Matcher

- `rpmFiles`: 입력된 RPM 파일 목록의 `NAME`이 `metadataNameIn`에 모두 포함되는지 검증합니다.
- `number`: placeholder 위치의 인자가 10진수 숫자인지 검증합니다.
- placeholder 형식: `<type:name>` (예: `<rpmFiles:k8s-rpms>`, `<number:lines>`).

### 5.3 정책 Bundle

```text
cmdgate-policy-<version>.tar.gz
├── manifest.yaml
├── checksums.sha256
└── allowlist.yaml
```

- `manifest.yaml`: 버전, 타임스탬프 등 메타데이터.
- `checksums.sha256`: allowlist.yaml의 SHA-256 체크섬.
- 반영 순서:
  1. bundle 압축 해제
  2. manifest/checksum 검증
  3. allowlist.yaml schema 검증
  4. 기존 allowlist.yaml 백업
  5. 신규 allowlist.yaml 반영
  6. audit log 기록

### 5.4 감사 로그

- 파일: `/var/log/cmdgate/audit.log`
- 형식: 구조화된 텍스트(JSON Lines 사용 고려)
- 항목: `timestamp`, `user`, `action`, `command_id`, `command`, `result`, `reason`

## 6. 기술 스택

- Go 1.22+
- YAML: `gopkg.in/yaml.v3`
- 테스트: 표준 `testing` 패키지, table-driven tests
- 빌드: `go build ./cmd/...`
- 설치: `scripts/install-cmdgate.sh`

## 7. TDD 규칙

1. 먼저 실패하는 테스트를 작성합니다.
2. 테스트가 예상한 이유로 실패하는지 확인합니다.
3. 테스트를 통과시키는 최소한의 코드를 작성합니다.
4. 테스트가 통과하는지 확인합니다.
5. 리팩토링은 테스트가 통과한 후에만 수행합니다.
6. 모든 신규 함수/메서드는 테스트가 있어야 합니다.

## 8. 설치 스크립트 (`scripts/install-cmdgate.sh`)

다음 작업을 수행합니다:

1. `/opt/cmdgate` 및 `/opt/cmdgate/work` 생성
2. `/var/log/cmdgate` 생성
3. `cmdgate`, `cmdgate-exec`, `allowlist.yaml` 복사
4. 파일 권한 설정 (`0755`, `0750`, `0640`, `0700`, `0750`)
5. sudoers 파일 생성 (`/etc/sudoers.d/cmdgate`)
6. `visudo -c` 검증

sudoers 예시:

```sudoers
cmdgateadm ALL=(ALL) NOPASSWD: /opt/cmdgate/cmdgate-exec *
```

## 9. 제외 범위

다음은 의도적으로 구현하지 않습니다:

- daemon / systemd service
- Unix socket / HTTP API
- mTLS
- plugin 구조
- shell 문자열 실행 (`bash -c`, `sh -c`, `eval`)
- 오타 자동 보정
- alias 자동 추론
- 명령어 fuzzy match
- `executables` 매핑
- Go 코드 내 명령 절대경로 하드코딩

## 10. 에이전트 작업 규칙

1. `AGENTS.md`를 변경할 때는 사용자 승인을 받습니다.
2. 구현 전 `writing-plans` 스킬을 사용하여 구현 계획을 작성합니다.
3. 복잡한 하위 작업은 `subagent-driven-development` 스킬을 사용합니다.
4. 모든 코드는 TDD로 작성합니다.
5. 문서를 함께 업데이트합니다.
6. **사용자에게 답변할 때는 항상 한국어로 응답합니다.**
