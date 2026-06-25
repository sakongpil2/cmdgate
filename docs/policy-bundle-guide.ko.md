# CmdGate 정책 번들 검증 가이드

이 문서는 CmdGate 정책 번들을 만들고 검증하는 방법을 설명합니다. 현재 CLI는
번들 검증만 지원하며, `/opt/cmdgate/allowlist.yaml`에 번들을 적용하는 기능은
제공하지 않습니다.

## 번들 형식

정책 번들은 다음 구조를 가진 gzip 압축 tar 아카이브입니다.

```text
cmdgate-policy-<version>.tar.gz
├── manifest.yaml
├── allowlist.yaml
└── checksums.sha256
```

- `manifest.yaml`: 번들 메타데이터입니다. `version`은 필수입니다.
- `allowlist.yaml`: 검증할 정책 파일입니다.
- `checksums.sha256`: `allowlist.yaml`의 SHA-256 16진수 값입니다.

검증 단계는 다음과 같습니다.

1. gzip/tar 아카이브를 읽습니다.
2. 세 필수 파일이 모두 있는지 확인합니다.
3. `manifest.yaml`을 파싱하고 `version`이 비어 있지 않은지 확인합니다.
4. `allowlist.yaml`의 SHA-256 값을 `checksums.sha256`과 비교합니다.
5. `allowlist.yaml`을 파싱하고 스키마를 검증합니다. 검증 대상은 `version`,
   `mode`, 명령 ID, matcher 참조, 지원 matcher 타입입니다.

## 번들 만들기

소스에 포함된 헬퍼 스크립트를 사용합니다.

```bash
cd /path/to/cmdgate-source
./scripts/build-policy-bundle.sh 1.1.0
```

이 명령은 `configs/allowlist.yaml`을 기준으로
`scripts/cmdgate-policy-1.1.0.tar.gz`를 만듭니다.

다른 allowlist 파일을 사용하려면 다음처럼 실행합니다.

```bash
./scripts/build-policy-bundle.sh 1.1.0 /path/to/allowlist.yaml
```

## 수동으로 번들 만들기

```bash
VERSION="1.1.0"
ALLOWLIST="/path/to/allowlist.yaml"
WORK_DIR="$(mktemp -d)"

cat > "${WORK_DIR}/manifest.yaml" <<EOF
version: "${VERSION}"
timestamp: "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
EOF

cp "${ALLOWLIST}" "${WORK_DIR}/allowlist.yaml"
sha256sum "${WORK_DIR}/allowlist.yaml" | awk '{print $1}' > "${WORK_DIR}/checksums.sha256"

tar -czf "cmdgate-policy-${VERSION}.tar.gz" -C "${WORK_DIR}" \
  manifest.yaml allowlist.yaml checksums.sha256

rm -rf "${WORK_DIR}"
```

## 번들 검증

```bash
cmdgate policy validate --bundle /home/cmdgateadm/cmdgate-policy-1.1.0.tar.gz
```

성공하면 출력 없이 종료 코드 `0`을 반환합니다. 실패하면 어떤 검증에서 문제가
발생했는지 오류로 알려줍니다.

```text
checksum mismatch
bundle missing required files
invalid allowlist schema: command[0]: id is required
```

검증이 성공하면 `cmdgate-exec`는 `policy_validate` 감사 로그를 기록합니다.

## 설치된 정책 확인

번들 검증은 설치된 정책을 변경하지 않습니다. 현재 설치된 정책을 확인하려면
허용 명령 목록을 출력합니다.

```bash
cmdgate run list
```

최근 감사 로그를 확인하려면 다음 명령을 사용합니다.

```bash
cmdgate audit tail 20
```

## 문제 해결

### `checksum mismatch`

`checksums.sha256`이 `allowlist.yaml`의 SHA-256과 일치하지 않습니다. 번들을
다시 만들거나 체크섬을 다시 계산하세요.

### `bundle missing required files`

아카이브에 `manifest.yaml`, `allowlist.yaml`, `checksums.sha256` 중 하나가
없습니다. 파일 이름이 정확히 일치해야 합니다.

### `invalid allowlist schema`

주요 원인은 다음과 같습니다.

- `version` 또는 `mode`가 없습니다.
- 명령의 `id` 또는 `cmd`가 비어 있습니다.
- placeholder가 정의되지 않은 matcher를 참조합니다.
- placeholder 타입과 matcher 타입이 일치하지 않습니다.
- 지원하지 않는 matcher 타입을 사용했습니다.

## 보안 고려사항

- 번들 파일은 인가된 운영자만 읽을 수 있도록 관리하세요(`0640` 이하).
- 번들은 신뢰할 수 있는 경로로 전달하세요. 체크섬은 우발적 손상은 찾을 수
  있지만 악의적 변조를 막지는 못합니다.
- 감사 로그는 `/var/log/cmdgate/audit.log`에 기록됩니다.
