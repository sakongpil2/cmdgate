# CmdGate 정책 번들 업데이트 가이드

이 문서는 CmdGate 정책 번들을 만들고, 검증하고, 적용하고, 롤백하는 방법을
설명합니다. 정책 번들은 초기 설치 이후 `/opt/cmdgate/allowlist.yaml`을
업데이트할 때 사용하는 유일한 방법입니다.

## 정책 번들이란?

정책 번들은 고정된 구조를 가진 gzip 압축 tar 아카이브입니다.

```text
cmdgate-policy-<version>.tar.gz
├── manifest.yaml
├── allowlist.yaml
└── checksums.sha256
```

- `manifest.yaml` — 번들 메타데이터(`version`, `timestamp`).
- `allowlist.yaml` — 새 정책 파일.
- `checksums.sha256` — `allowlist.yaml`의 SHA-256 16진수 값.

`cmdgate-exec`는 번들을 적용하기 전에 다음 검증을 수행합니다.

1. 아카이브 압축 해제.
2. 세 필수 파일 존재 확인.
3. `manifest.yaml` 파싱 및 `version` 필수 여부 확인.
4. `allowlist.yaml`의 SHA-256을 `checksums.sha256`과 비교.
5. `allowlist.yaml` 파싱 및 `ValidateSchema()` 실행(version, mode, command ID,
   matcher 참조, 지원 matcher 타입 확인).

모든 검증을 통과해야 현재 정책을 백업하고 새 정책으로 교체합니다.

## 번들 만들기

소스에 포함된 헬퍼 스크립트를 사용합니다.

```bash
cd /path/to/cmdgate-source
./scripts/build-policy-bundle.sh 1.1.0
```

이 명령은 `configs/allowlist.yaml`을 기반으로
`scripts/cmdgate-policy-1.1.0.tar.gz`를 만듭니다.

다른 allowlist 파일을 사용하려면:

```bash
./scripts/build-policy-bundle.sh 1.1.0 /path/to/your/allowlist.yaml
```

### 수동으로 번들 만들기

헬퍼 스크립트를 사용할 수 없는 경우 직접 실행합니다.

```bash
VERSION="1.1.0"
ALLOWLIST="/path/to/allowlist.yaml"
WORK_DIR="$(mktemp -d)"

# manifest.yaml
cat > "${WORK_DIR}/manifest.yaml" <<EOF
version: "${VERSION}"
timestamp: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
EOF

# allowlist.yaml
cp "${ALLOWLIST}" "${WORK_DIR}/allowlist.yaml"

# checksums.sha256
sha256sum "${WORK_DIR}/allowlist.yaml" | awk '{print $1}' > "${WORK_DIR}/checksums.sha256"

# 번들
tar -czf "cmdgate-policy-${VERSION}.tar.gz" -C "${WORK_DIR}" \
  manifest.yaml allowlist.yaml checksums.sha256

rm -rf "${WORK_DIR}"
```

## 번들 검증

검증은 운영 중인 정책을 변경하지 않고 번들만 확인합니다.

```bash
cmdgate policy validate --bundle /home/cmdgateadm/cmdgate-policy-1.1.0.tar.gz
```

성공하면 출력 없이 종료 코드 `0`을 반환합니다.

실패하면 어떤 검증에서 문제가 발생했는지 알려줍니다.

```text
checksum mismatch
bundle missing required files
invalid allowlist schema: command[0]: id is required
```

## 번들 적용

번들 적용은 `/opt/cmdgate/allowlist.yaml`을 교체합니다.

```bash
cmdgate policy apply --bundle /home/cmdgateadm/cmdgate-policy-1.1.0.tar.gz
```

`cmdgate-exec`는 다음 단계를 수행합니다.

1. 번들 검증.
2. 현재 정책을 `/opt/cmdgate/allowlist.yaml.backup`으로 백업.
3. 새 정책을 `/opt/cmdgate/allowlist.yaml`에 `0640` 권한으로 기록.
4. `action=policy_apply` 감사 로그 기록.

어느 단계에서든 실패하면 운영 중인 정책은 변경되지 않습니다.

## 실패한 업데이트 롤백

새 정책에 문제가 생기면 백업을 복원합니다.

```bash
sudo cp /opt/cmdgate/allowlist.yaml.backup /opt/cmdgate/allowlist.yaml
sudo chmod 0640 /opt/cmdgate/allowlist.yaml
```

복원 후 정책이 정상적으로 로드되는지 확인합니다.

```bash
cmdgate run list
```

## 운영 정책 확인

번들 적용 후 허용 명령 목록을 출력하여 새 정책을 확인합니다.

```bash
cmdgate run list
```

최근 정책 관련 감사 로그를 확인하려면:

```bash
cmdgate audit tail 20
```

## 문제 해결

### `checksum mismatch`

`checksums.sha256` 파일이 `allowlist.yaml`의 SHA-256과 일치하지 않습니다.
헬퍼 스크립트로 번들을 다시 만들거나 체크섬을 직접 다시 계산하세요.

### `bundle missing required files`

`manifest.yaml`, `allowlist.yaml`, `checksums.sha256` 중 하나가 누락되었습니다.
파일 이름이 정확한지 확인하세요.

### `invalid allowlist schema`

`allowlist.yaml`의 의미 검증에 실패했습니다. 흔한 원인은 다음과 같습니다.

- `version` 또는 `mode` 누락.
- `id` 또는 `cmd`가 비어 있는 command.
- 정의되지 않은 matcher를 참조하는 placeholder.
- placeholder 타입과 matcher 타입 불일치, 예: `<number:lines>`를
  `type: string` matcher로 정의한 경우.
- 지원하지 않는 matcher 타입.

### 적용 시 권한 거부

`cmdgate-exec`는 `sudo`를 통해 `root`로 실행되므로, 적용 자체는 운영자 계정이
`cmdgate-exec`에 대한 sudoers 규칙을 가지고 있어야 합니다. 파일 시스템 오류로
적용이 실패하면 `/opt/cmdgate`가 root에게 쓰기 가능한지, 현재
`allowlist.yaml`이 immutable 속성을 갖고 있지 않은지 확인하세요.

## 보안 고려사항

- 번들 파일은 인가된 운영자만 읽을 수 있도록 관리하세요(`0640` 이하).
- 번들은 신뢰할 수 있는 경로로 전달하세요. checksum은 우발적 손상을 방지할
  뿐, 악의적 변조를 막지는 못합니다. 출처 검증이 필요하면 별도 서명을
  사용하세요.
- `/opt/cmdgate/allowlist.yaml.backup`은 민감한 파일입니다. 이전 정책이
  포함되어 있습니다.
- 감사 로그(`/var/log/cmdgate/audit.log`)에 모든 검증 및 적용 시도가
  기록됩니다.
