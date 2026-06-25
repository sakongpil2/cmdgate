# CmdGate 요구사항정의서

## 1. 개요

CmdGate는 운영자가 사전에 허용된 명령만 위임 권한으로 실행할 수 있도록 하는 allowlist 기반 CLI 도구다.

```text
사용자 바이너리: cmdgate
권한 실행기: cmdgate-exec
정책 파일: allowlist.yaml
설치 경로: /opt/cmdgate
로그 경로: /var/log/cmdgate
```

CmdGate는 sudo를 대체하지 않는다.  
`cmdgate`가 사용자 입력을 받고, 내부적으로 sudo를 통해 `cmdgate-exec`를 호출한다.  
실제 명령 검증과 실행은 `cmdgate-exec`가 수행한다.

---

## 2. 바이너리 요구사항

### 2.1 바이너리 구성

```text
cmdgate       : 사용자 실행용 CLI
cmdgate-exec  : sudoers로만 실행되는 권한 실행기
```

### 2.2 구현 원칙

```text
1. Go 바이너리 2개로 물리 분리한다.
2. Go 코드는 최대한 간결하게 작성한다.
3. daemon, socket, API server, plugin 구조는 사용하지 않는다.
4. shell 문자열 실행, bash -c, sh -c, eval을 사용하지 않는다.
5. 검증 통과 후 argv 배열 방식으로 실행한다.
6. cmdgate-exec에만 위임 권한 실행 로직을 둔다.
```

---

## 3. 설치 구조

```text
/opt/cmdgate/
├── cmdgate
├── cmdgate-exec
├── allowlist.yaml
└── work/

/var/log/cmdgate/
```

### 3.1 파일 권한

| 경로 | 권한 | 설명 |
|---|---:|---|
| `/opt/cmdgate` | `0755` | 설치 기준 경로 |
| `/opt/cmdgate/cmdgate` | `0755` | 사용자 실행 바이너리 |
| `/opt/cmdgate/cmdgate-exec` | `0750` | 권한 실행기 |
| `/opt/cmdgate/allowlist.yaml` | `0640` | 허용 명령어 정책 |
| `/opt/cmdgate/work` | `0700` | 내부 작업 디렉토리 |
| `/var/log/cmdgate` | `0750` | 로그 디렉토리 |

소유자는 기본적으로 시스템 운영부서 기준으로 설정한다. 예시는 `root:root` 기준이다.

---

## 4. sudoers 요구사항

운영자 계정은 `cmdgate-exec`만 sudo로 실행할 수 있다.

```sudoers
opsadm ALL=(ALL) NOPASSWD: /opt/cmdgate/cmdgate-exec *
```

개별 시스템 명령은 sudoers에 직접 허용하지 않는다.

```text
허용하지 않음:
  /usr/bin/systemctl
  /usr/bin/dnf
  /usr/bin/kubeadm
  /usr/bin/crictl
  /usr/bin/ctr
  /usr/bin/journalctl
  /bin/bash
  /bin/sh
```

---

## 5. 실행 흐름

사용자 실행:

```bash
cmdgate run systemctl restart kubelet
```

내부 호출:

```bash
sudo -n /opt/cmdgate/cmdgate-exec run systemctl restart kubelet
```

처리 순서:

```text
1. cmdgate가 사용자 입력을 받는다.
2. cmdgate가 sudo -n으로 cmdgate-exec를 호출한다.
3. cmdgate-exec가 allowlist.yaml을 읽는다.
4. 사용자 입력과 commands[].cmd를 비교한다.
5. matcher가 있으면 matcher 검증을 수행한다.
6. 검증 성공 시 argv 배열 방식으로 실행한다.
7. 실행 결과를 audit log에 기록한다.
```

---

## 6. allowlist.yaml schema

정책은 한 줄 명령어 기반으로 정의한다.

```yaml
version: 1.0.0
mode: allowlist-only

commands:
  - id: <명령 ID>
    desc: <설명>
    cmd: "<cmdgate run 뒤에 입력할 명령어>"

matchers:
  <matcher 이름>:
    type: <matcher 타입>
```

### 6.1 원칙

```text
1. commands[].cmd는 사용자가 입력할 명령어와 동일하게 작성한다.
2. 고정 명령은 exact match로만 허용한다.
3. 오타 보정, fuzzy match, alias 추론은 하지 않는다.
4. 동적 인자가 필요한 경우에만 <type:name> placeholder를 사용한다.
5. allowlist.yaml에는 executables 매핑을 두지 않는다.
6. 실행 파일 검색 경로는 시스템 운영부서가 관리하는 PATH 또는 sudoers secure_path 기준을 따른다.
```

---

## 7. matcher 요구사항

### 7.1 rpmFiles

로컬 RPM 파일 설치 검증에 사용한다.

```yaml
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
```

검증 방식:

```text
1. 입력된 RPM 파일 목록을 식별한다.
2. 각 RPM에 대해 rpm -qp를 실행한다.
3. NAME / VERSION / RELEASE / ARCH를 추출한다.
4. 모든 RPM의 NAME이 metadataNameIn에 포함되는지 확인한다.
5. 하나라도 허용되지 않은 NAME이면 전체 설치를 거부한다.
6. 모두 허용되면 dnf install <rpm 파일 목록>을 실행한다.
```

내부 조회 예시:

```bash
rpm -qp --queryformat '%{NAME} %{VERSION} %{RELEASE} %{ARCH}\n' <rpm-file>
```

복수 RPM은 허용한다.

```bash
cmdgate run dnf install /home/opsadm/packages/rpms/*.rpm
```

단, 확장된 모든 RPM이 검증을 통과해야 한다.

### 7.2 number

숫자 인자 검증에 사용한다.

```yaml
matchers:
  lines:
    type: number
```

예시:

```yaml
cmd: "journalctl -u kubelet -n <number:lines> --no-pager"
```

---

## 8. 기본 allowlist 예시

```yaml
version: 1.0.0
mode: allowlist-only

commands:
  - id: systemctl-restart-kubelet
    desc: kubelet 재시작
    cmd: "systemctl restart kubelet"

  - id: systemctl-restart-containerd
    desc: containerd 재시작
    cmd: "systemctl restart containerd"

  - id: systemctl-stop-kubelet
    desc: kubelet 중지
    cmd: "systemctl stop kubelet"

  - id: systemctl-stop-containerd
    desc: containerd 중지
    cmd: "systemctl stop containerd"

  - id: systemctl-start-kubelet
    desc: kubelet 시작
    cmd: "systemctl start kubelet"

  - id: systemctl-start-containerd
    desc: containerd 시작
    cmd: "systemctl start containerd"

  - id: systemctl-status-kubelet
    desc: kubelet 상태 확인
    cmd: "systemctl status kubelet"

  - id: systemctl-status-containerd
    desc: containerd 상태 확인
    cmd: "systemctl status containerd"

  - id: systemctl-enable-kubelet
    desc: kubelet enable
    cmd: "systemctl enable kubelet"

  - id: systemctl-enable-containerd
    desc: containerd enable
    cmd: "systemctl enable containerd"

  - id: systemctl-daemon-reload
    desc: systemd daemon reload
    cmd: "systemctl daemon-reload"

  - id: dnf-install-kubelet
    desc: kubelet 설치
    cmd: "dnf install kubelet"

  - id: dnf-install-kubeadm
    desc: kubeadm 설치
    cmd: "dnf install kubeadm"

  - id: dnf-install-local-rpms
    desc: 로컬 RPM 설치
    cmd: "dnf install <rpmFiles:k8s-rpms>"

  - id: dnf-info-containerd
    desc: containerd 패키지 정보 확인
    cmd: "dnf info containerd"

  - id: kubeadm-version
    desc: kubeadm 버전 확인
    cmd: "kubeadm version"

  - id: kubeadm-config-images-list
    desc: kubeadm 이미지 목록 확인
    cmd: "kubeadm config images list"

  - id: kubeadm-config-images-pull
    desc: kubeadm 이미지 pull
    cmd: "kubeadm config images pull"

  - id: kubeadm-upgrade-plan
    desc: kubeadm upgrade plan
    cmd: "kubeadm upgrade plan"

  - id: kubeadm-upgrade-node
    desc: kubeadm upgrade node
    cmd: "kubeadm upgrade node"

  - id: kubeadm-certs-check-expiration
    desc: kubeadm 인증서 만료 확인
    cmd: "kubeadm certs check-expiration"

  - id: crictl-info
    desc: CRI 정보 확인
    cmd: "crictl info"

  - id: crictl-ps
    desc: 컨테이너 목록 확인
    cmd: "crictl ps"

  - id: crictl-images
    desc: CRI 이미지 목록 확인
    cmd: "crictl images"

  - id: ctr-images-ls
    desc: containerd 이미지 목록 확인
    cmd: "ctr -n k8s.io images ls"

  - id: ctr-containers-ls
    desc: containerd 컨테이너 목록 확인
    cmd: "ctr -n k8s.io containers ls"

  - id: journalctl-kubelet
    desc: kubelet 로그 확인
    cmd: "journalctl -u kubelet --no-pager"

  - id: journalctl-containerd
    desc: containerd 로그 확인
    cmd: "journalctl -u containerd --no-pager"

  - id: journalctl-kubelet-lines
    desc: kubelet 로그 라인 수 지정 확인
    cmd: "journalctl -u kubelet -n <number:lines> --no-pager"

  - id: journalctl-containerd-lines
    desc: containerd 로그 라인 수 지정 확인
    cmd: "journalctl -u containerd -n <number:lines> --no-pager"

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

---

## 9. 명령 동작 요구사항

### 9.1 run list

```bash
cmdgate run list
```

`allowlist.yaml`의 `commands[]`를 기준으로 허용 명령 목록을 출력한다.

출력 항목:

```text
id
desc
cmd
```

### 9.2 run

```bash
cmdgate run <command> [args...]
```

요구사항:

```text
1. 입력 명령은 allowlist.yaml의 commands[].cmd와 일치해야 한다.
2. matcher placeholder가 있는 경우 matcher 검증을 수행한다.
3. 일치하는 정책이 없으면 실행하지 않는다.
4. 실행 실패 사유를 사용자에게 출력한다.
```

---

## 10. 정책 변경 요구사항

정책 검증:

```bash
cmdgate policy validate --bundle /home/opsadm/packages/cmdgate-policy-1.1.0.tar.gz
```

정책 반영:

```bash
cmdgate policy apply --bundle /home/opsadm/packages/cmdgate-policy-1.1.0.tar.gz
```

bundle 구조:

```text
cmdgate-policy-1.1.0.tar.gz
├── manifest.yaml
├── checksums.sha256
└── allowlist.yaml
```

반영 순서:

```text
1. bundle 압축 해제
2. manifest/checksum 검증
3. allowlist.yaml schema 검증
4. 기존 allowlist.yaml 백업
5. 신규 allowlist.yaml 반영
6. audit log 기록
```

---

## 11. 감사 로그 요구사항

로그 파일:

```text
/var/log/cmdgate/audit.log
```

기록 항목:

```text
timestamp
user
action
command_id
command
result
reason
```

예시:

```text
2026-06-24T20:10:00+09:00 user=opsadm action=run command_id=systemctl-restart-kubelet command="systemctl restart kubelet" result=success
2026-06-24T20:12:00+09:00 user=opsadm action=run command="systemctl restart sshd" result=denied reason="no matching command"
```

---

## 12. 설치 스크립트 요구사항

설치 스크립트명:

```text
install-cmdgate.sh
```

필수 입력 파일:

```text
cmdgate
cmdgate-exec
allowlist.yaml
```

설치 작업:

```text
1. /opt/cmdgate/work 생성
2. /var/log/cmdgate 생성
3. cmdgate 복사
4. cmdgate-exec 복사
5. allowlist.yaml 복사
6. 파일 권한 설정
7. sudoers 파일 생성
8. visudo 검증
```

sudoers 예시:

```sudoers
opsadm ALL=(ALL) NOPASSWD: /opt/cmdgate/cmdgate-exec *
```

---

## 13. 제외 범위

```text
daemon
systemd service
Unix socket
HTTP API
mTLS
plugin 구조
shell 문자열 실행
오타 자동 보정
alias 자동 추론
명령어 fuzzy match
executables 매핑
Go 코드 내 명령 절대경로 하드코딩
```
