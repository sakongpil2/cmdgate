# CmdGate 정책 YAML 검증 가이드

이 문서는 `allowlist.yaml` 정책 파일을 `/opt/cmdgate/allowlist.yaml`에
배치하기 전에 검증하는 방법을 설명합니다.

## 정책 파일

CmdGate는 일반 YAML 정책 파일을 직접 검증합니다.

```text
allowlist.yaml
```

검증 단계는 다음과 같습니다.

1. 명령행으로 전달된 경로에서 YAML 파일을 읽습니다.
2. `allowlist.yaml`로 파싱합니다.
3. `version`, `mode`, 명령 ID, 명령 문자열, matcher 참조, 지원 matcher 타입
   같은 스키마 규칙을 검증합니다.

## 정책 검증

```bash
cmdgate policy validate /home/cmdgateadm/allowlist.yaml
```

성공하면 출력 없이 종료 코드 `0`을 반환합니다. 실패하면 어떤 검증에서 문제가
발생했는지 오류로 알려줍니다.

```text
invalid allowlist.yaml: yaml: line 1: did not find expected node content
invalid allowlist schema: command[0]: id is required
```

검증이 성공하면 `cmdgate-exec`는 `policy_validate` 감사 로그를 기록합니다.

## 검증된 정책 설치

검증은 설치된 정책을 변경하지 않습니다. 정책 검증이 성공하면 운영 환경의 권한
있는 배포 절차로 정책을 복사합니다.

```bash
sudo install -o root -g root -m 0640 /home/cmdgateadm/allowlist.yaml /opt/cmdgate/allowlist.yaml
```

설치된 정책을 확인하려면 허용 명령 목록을 출력합니다.

```bash
cmdgate run list
```

최근 감사 로그를 확인하려면 다음 명령을 사용합니다.

```bash
cmdgate audit tail 20
```

## 문제 해결

### `invalid allowlist.yaml`

파일이 올바른 YAML이 아닙니다. 문법을 수정한 뒤 다시 검증하세요.

### `invalid allowlist schema`

주요 원인은 다음과 같습니다.

- `version` 또는 `mode`가 없습니다.
- 명령의 `id` 또는 `cmd`가 비어 있습니다.
- placeholder가 정의되지 않은 matcher를 참조합니다.
- placeholder 타입과 matcher 타입이 일치하지 않습니다.
- 지원하지 않는 matcher 타입을 사용했습니다.

## 보안 고려사항

- 정책 파일은 인가된 운영자만 읽을 수 있도록 관리하세요(`0640` 이하).
- 정책 파일은 검증과 설치 전에 신뢰할 수 있는 경로로 전달하세요.
- 감사 로그는 `/var/log/cmdgate/audit.log`에 기록됩니다.
