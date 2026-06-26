# CmdGate 배포 가이드

이 문서는 릴리즈 압축 파일을 사용해 운영 서버에 CmdGate를 설치하는 절차를 설명합니다.

## 릴리즈 압축 파일 구성

운영 배포 압축 파일에는 대상 서버에 필요한 파일만 들어갑니다.

```text
cmdgate
cmdgate-exec
allowlist.yaml
install-cmdgate.sh
SHA256SUMS
```

`cmdgate-exec`는 sudoers에 등록되는 권한 실행기입니다. 이름을 바꾸려면 `cmdgate`
안의 실행기 경로, 설치 스크립트, sudoers 경로를 모두 함께 바꿔야 합니다.

## 운영 서버 설치

압축 파일을 대상 서버에 복사한 뒤 다음처럼 설치합니다.

```bash
tar -xzf cmdgate-linux-amd64-<version>.tar.gz
cd cmdgate-linux-amd64-<version>
sha256sum -c SHA256SUMS
sudo ./install-cmdgate.sh
```

기본 운영자 계정은 `cmdgateadm`입니다. 다른 계정을 사용하려면 다음처럼 실행합니다.

```bash
sudo CMDGATE_USER=myops ./install-cmdgate.sh
```

설치 스크립트는 `/opt/cmdgate`, `/opt/cmdgate/work`, `/var/log/cmdgate`를 만들고,
두 바이너리와 `allowlist.yaml`을 설치합니다. 또한 `/etc/sudoers.d/cmdgate`를
작성한 뒤 `visudo`로 검증하고 `/usr/local/bin/cmdgate` 링크를 만듭니다.

## 설치 후 확인

운영자 사용자로 다음 명령을 실행합니다.

```bash
cmdgate --help
cmdgate policy validate /opt/cmdgate/allowlist.yaml
cmdgate run list
cmdgate audit tail 20
```

정책이 정상이면 다음 메시지가 출력됩니다.

```text
policy valid: /opt/cmdgate/allowlist.yaml
```

stdout/stderr가 터미널이면 성공 메시지는 초록색, 실패 메시지는 빨간색으로 표시됩니다.
색상을 끄려면 `NO_COLOR=1`을 설정합니다.

## 정책 업데이트

설치된 정책을 바꾸기 전에 새 정책 파일을 먼저 검증합니다.

```bash
cmdgate policy validate /home/cmdgateadm/allowlist.yaml
sudo install -o root -g root -m 0640 /home/cmdgateadm/allowlist.yaml /opt/cmdgate/allowlist.yaml
cmdgate run list
```

## 릴리즈 압축 파일 만들기

저장소 루트에서 다음 명령을 실행합니다.

```bash
VERSION=1.1.0
DIST="dist/cmdgate-linux-amd64-${VERSION}"
mkdir -p "${DIST}"

GOCACHE=/tmp/go-cache GOOS=linux GOARCH=amd64 go build -o "${DIST}/cmdgate" ./cmd/cmdgate
GOCACHE=/tmp/go-cache GOOS=linux GOARCH=amd64 go build -o "${DIST}/cmdgate-exec" ./cmd/cmdgate-exec
cp configs/allowlist.yaml "${DIST}/allowlist.yaml"
cp scripts/install-cmdgate.sh "${DIST}/install-cmdgate.sh"
chmod 0755 "${DIST}/cmdgate" "${DIST}/cmdgate-exec" "${DIST}/install-cmdgate.sh"

(cd "${DIST}" && sha256sum cmdgate cmdgate-exec allowlist.yaml install-cmdgate.sh > SHA256SUMS)
tar -czf "${DIST}.tar.gz" -C dist "$(basename "${DIST}")"
```

## GitHub 릴리즈 절차

릴리즈는 annotated tag로 관리합니다.

```bash
git status --short
GOCACHE=/tmp/go-cache go test ./...
git tag -a v1.1.0 -m "CmdGate v1.1.0"
git push origin main
git push origin v1.1.0
```

GitHub CLI가 있으면 릴리즈 생성과 압축 파일 업로드를 한 번에 할 수 있습니다.

```bash
gh release create v1.1.0 \
  dist/cmdgate-linux-amd64-1.1.0.tar.gz \
  --title "CmdGate v1.1.0" \
  --notes "See README.md and docs/deployment-guide.md for installation steps."
```

`gh`를 사용하지 않는다면 GitHub 웹 UI에서 push된 tag로 릴리즈를 만들고
`cmdgate-linux-amd64-<version>.tar.gz` 파일을 업로드합니다.
