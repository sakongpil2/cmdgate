#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
VERSION="${1:-}"
ALLOWLIST_PATH="${2:-${SCRIPT_DIR}/../configs/allowlist.yaml}"

if [[ -z "${VERSION}" ]]; then
    cat <<EOF
Usage: $0 <version> [path/to/allowlist.yaml]

Example:
    $0 1.1.0
    $0 1.1.0 /path/to/allowlist.yaml
EOF
    exit 1
fi

if [[ ! -f "${ALLOWLIST_PATH}" ]]; then
    echo "allowlist.yaml not found: ${ALLOWLIST_PATH}" >&2
    exit 1
fi

WORK_DIR="$(mktemp -d)"
trap 'rm -rf "${WORK_DIR}"' EXIT

BUNDLE_NAME="cmdgate-policy-${VERSION}.tar.gz"
BUNDLE_PATH="${SCRIPT_DIR}/${BUNDLE_NAME}"

ALLOWLIST_BODY="$(cat "${ALLOWLIST_PATH}")"
CHECKSUM="$(printf '%s' "${ALLOWLIST_BODY}" | sha256sum | awk '{print $1}')"
TIMESTAMP="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

cat > "${WORK_DIR}/manifest.yaml" <<EOF
version: "${VERSION}"
timestamp: "${TIMESTAMP}"
EOF

printf '%s' "${ALLOWLIST_BODY}" > "${WORK_DIR}/allowlist.yaml"
printf '%s\n' "${CHECKSUM}" > "${WORK_DIR}/checksums.sha256"

tar -czf "${BUNDLE_PATH}" -C "${WORK_DIR}" manifest.yaml allowlist.yaml checksums.sha256

echo "Created ${BUNDLE_PATH}"
