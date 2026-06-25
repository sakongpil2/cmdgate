# CmdGate

CmdGate는 운영자가 사전에 허용된 명령만 위임 권한으로 실행할 수 있도록 하는 allowlist 기반 CLI 도구입니다.

- 사용자용 바이너리: `cmdgate`
- 권한 실행기: `cmdgate-exec`
- 정책 파일: `/opt/cmdgate/allowlist.yaml`
- 감사 로그: `/var/log/cmdgate/audit.log`

`cmdgate`는 사용자 입력을 받아 `sudo -n`을 통해 `cmdgate-exec`를 호출합니다. `cmdgate-exec`는 allowlist를 기준으로 요청을 검증하고, matcher를 실행하며, argv 배열 형태로 명령을 실행한 뒤 감사 기록을 남깁니다.

> English documentation is available at [README.md](README.md).

## 바이너리 책임

### `cmdgate`

사용자가 직접 실행하는 CLI입니다. 입력을 검증하지 않고 `cmdgate-exec`에 그대로 전달합니다.

지원 명령:

- `cmdgate run <command> [args...]`
- `cmdgate run list`
- `cmdgate policy validate --bundle <tar.gz>`
- `cmdgate policy apply --bundle <tar.gz>`
- `cmdgate audit tail [n]`
- `cmdgate help`
- `cmdgate --help`

남부적으로는 다음 형태로 `cmdgate-exec`를 호출합니다.

```bash
sudo -n /opt/cmdgate/cmdgate-exec <subcommand> [args...]
```

### `cmdgate-exec`

`sudo` 권한으로만 실행되는 권한 실행기입니다. `/opt/cmdgate/allowlist.yaml`을 읽어 사용자 입력과 비교하고, matcher로 placeholder를 검증한 뒤 `exec.Command(cmd, args...)` 형태로 실행합니다.

지원 명령:

- `cmdgate-exec run <command> [args...]`
- `cmdgate-exec run list`
- `cmdgate-exec policy validate --bundle <tar.gz>`
- `cmdgate-exec policy apply --bundle <tar.gz>`
- `cmdgate-exec audit tail [n]`
- `cmdgate-exec help`
- `cmdgate-exec --help`

주요 동작:

1. `/opt/cmdgate/allowlist.yaml`을 로드합니다.
2. 사용자 argv와 정책 명령을 매칭합니다.
3. `<type:name>` placeholder가 있으면 matcher로 검증합니다.
4. argv 배열 형태로 명령을 실행합니다.
5. 결과를 `/var/log/cmdgate/audit.log`에 기록합니다.

## 빌드

Go 1.22 이상이 필요합니다.

```bash
go build ./cmd/...
```

빌드가 완료되면 `cmdgate`와 `cmdgate-exec` 바이너리가 생성됩니다.

## 설치

`scripts/install-cmdgate.sh`를 사용해 설치합니다. 설치 스크립트는 root 권한으로 실행해야 합니다.

기본 운영자 계정은 `cmdgateadm`입니다. 다른 운영자 계정을 사용하려면 설치 스크립트 실행 전에
`CMDGATE_USER` 환경 변수를 설정합니다. 설치 스크립트는 사용자를 생성하지 않으므로, 설치 전에
먼저 생성해야 합니다.

```bash
# 빌드한 바이너리와 기본 정책 파일을 scripts 디렉터리에 복사
cp cmdgate cmdgate-exec configs/allowlist.yaml scripts/

# 기본 운영자(cmdgateadm)로 설치
sudo ./scripts/install-cmdgate.sh

# 또는 다른 운영자 계정 사용
# sudo CMDGATE_USER=myops ./scripts/install-cmdgate.sh
```

설치 스크립트는 다음 작업을 수행합니다.

1. `/opt/cmdgate`, `/opt/cmdgate/work`, `/var/log/cmdgate` 디렉터리를 생성합니다.
2. `cmdgate`, `cmdgate-exec`, `allowlist.yaml`을 `/opt/cmdgate`에 복사합니다.
3. 파일 및 디렉터리 권한을 설정합니다.
4. 운영자 계정용 `/etc/sudoers.d/cmdgate` 파일을 생성합니다.
5. `visudo -c`로 sudoers 설정을 검증합니다.
6. `/etc/profile.d/cmdgate.sh`를 통해 `/opt/cmdgate`를 시스템 `PATH`에 추가하여
   사용자가 전체 경로를 입력하지 않고 `cmdgate`를 실행할 수 있게 합니다.

설치 후 다음 sudoers 규칙이 적용됩니다. `CMDGATE_USER`를 변경했다면 `cmdgateadm`을
해당 계정명으로 바꿔서 확인하세요.

```sudoers
cmdgateadm ALL=(root) NOPASSWD: /opt/cmdgate/cmdgate-exec *
```

운영자 사용자가 존재하지 않으면 설치 스크립트는 경고를 출력합니다. `cmdgate`를
실행하기 전에 다음과 같이 사용자를 생성하세요.

```bash
useradd -r -s /sbin/nologin cmdgateadm
```

## 사용 예시

### 허용된 명령 목록 보기

```bash
cmdgate run list
```

### 허용된 명령 실행

```bash
cmdgate run systemctl restart kubelet
cmdgate run journalctl -u kubelet -n 50 --no-pager
```

### 정책 번들 검증

```bash
cmdgate policy validate --bundle cmdgate-policy-1.1.0.tar.gz
```

### 감사 로그 조회

```bash
cmdgate audit tail      # 최근 20개 항목
cmdgate audit tail 50   # 최근 50개 항목
```

출력은 `/var/log/cmdgate/audit.log`의 JSON Lines 형식 그대로입니다.

### 도움말 보기

```bash
cmdgate help
cmdgate --help
```

## allowlist.yaml 형식

`allowlist.yaml`은 허용할 명령과 matcher를 정의합니다.

```yaml
version: 1.0.0
mode: allowlist-only

commands:
  - id: systemctl-restart-kubelet
    desc: kubelet 재시작
    cmd: "systemctl restart kubelet"

  - id: journalctl-kubelet-lines
    desc: kubelet 로그 라인 수 지정 확인
    cmd: "journalctl -u kubelet -n <number:lines> --no-pager"

  - id: dnf-install-local-rpms
    desc: 로컬 RPM 설치
    cmd: "dnf install <rpmFiles:k8s-rpms>"

matchers:
  k8s-rpms:
    type: rpmFiles
    multiple: true
    metadataNameIn:
      - kubelet
      - containerd
      - containerd.io
      - kubeadm
      - cri-tools
      - kubectl
      - kubernetes-cni

  lines:
    type: number
```

### 명령 정의

- `version`: 정책 파일 버전입니다.
- `mode`: 정책 모드입니다. 현재 `allowlist-only`만 지원합니다.
- `commands`: 허용할 명령 목록입니다.
- `matchers`: placeholder에서 참조하는 matcher 정의입니다.

### Matcher

`<type:name>` 형태의 placeholder는 해당 위치 인자를 matcher에 위임하여 검증합니다.

#### `string`

placeholder 위치의 인자가 비어 있지 않은 문자열인지 검증합니다. 선택적 `pattern`
필드로 정규식을 지정하면 값을 추가로 제한할 수 있습니다.

```yaml
cmd: "/opt/cmdgate/scripts/<string:script>"
```

```yaml
matchers:
  script:
    type: string
    pattern: '^(?:[a-zA-Z0-9_-]+/)*[a-zA-Z0-9_-]+\.sh$'
```

이 예시는 `backup.sh`, `maintenance/reboot.sh`는 허용하면서 `..`, 절대 경로,
`.sh`가 아닌 파일은 거부합니다.

#### `number`

placeholder 위치의 인자가 10진수 정수인지 검증합니다.

```yaml
cmd: "journalctl -u kubelet -n <number:lines> --no-pager"
```

`50`, `100` 같은 숫자만 허용됩니다.

#### `rpmFiles`

입력된 RPM 파일들의 `NAME` 메타데이터가 모두 `metadataNameIn`에 포함되는지 검증합니다.

```yaml
cmd: "dnf install <rpmFiles:k8s-rpms>"
```

- `type`: `rpmFiles`
- `multiple`: `true`이면 여러 RPM 파일을 한 번에 지정할 수 있습니다.
- `metadataNameIn`: 허용할 RPM 패키지 이름 목록입니다.

## 감사 로그

모든 실행 시도는 `/var/log/cmdgate/audit.log`에 기록됩니다.

로그 항목:

- `timestamp`: 이벤트 발생 시각
- `user`: 명령을 실행한 사용자
- `action`: 수행한 동작 (`run`, `policy_validate`, `policy_apply`, `audit_tail` 등)
- `command_id`: allowlist에서 매칭된 명령 ID
- `command`: 사용자가 입력한 명령
- `result`: 실행 결과 (`success` 또는 `denied`)
- `reason`: 거부된 경우 거부 사유

## 보안 고려사항

- CmdGate는 **argv 배열 방식**으로만 명령을 실행합니다. `bash -c`, `sh -c`, `eval` 등의 shell 문자열 실행은 사용하지 않습니다.
- 권한 상승은 `sudo`를 통해서만 이루어지며, `cmdgate-exec`만 비밀번호 없는 sudo 접근을 허용받습니다.
- `cmdgate`는 입력을 검증하지 않으므로, 검증은 반드시 `cmdgate-exec`에서 수행되어야 합니다.
- `/opt/cmdgate/allowlist.yaml` 접근을 제한하여 일반 사용자가 정책을 임의로 변경하지 못하도록 합니다.
- sudoers 규칙은 운영자 사용자(기본값 `cmdgateadm`)가 `/opt/cmdgate/cmdgate-exec`만
  실행할 수 있도록 제한해야 합니다.

```sudoers
cmdgateadm ALL=(root) NOPASSWD: /opt/cmdgate/cmdgate-exec *
```

운영자 계정을 변경하려면 설치 시 `CMDGATE_USER`를 설정하고, `cmdgate` 실행 전에
해당 사용자가 존재하는지 확인하세요.

- 정책 번들은 manifest와 SHA-256 체크섬 검증을 통과한 뒤에 적용됩니다.
