#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALL_DIR="/opt/cmdgate"
LOG_DIR="/var/log/cmdgate"
WORK_DIR="${INSTALL_DIR}/work"
SUDOERS_FILE="/etc/sudoers.d/cmdgate"

if [[ $EUID -ne 0 ]]; then
    echo "This script must be run as root" >&2
    exit 1
fi

for f in cmdgate cmdgate-exec allowlist.yaml; do
    if [[ ! -f "${SCRIPT_DIR}/${f}" ]]; then
        echo "Missing required file: ${SCRIPT_DIR}/${f}" >&2
        exit 1
    fi
done

if ! id -u cmdgateadm >/dev/null 2>&1; then
    cat >&2 <<'EOF'
Warning: user 'cmdgateadm' does not exist.
The sudoers rule will be installed, but it will only take effect after
the system administrator creates the 'cmdgateadm' user and adds them to
the appropriate groups.
EOF
fi

mkdir -p "${INSTALL_DIR}" "${WORK_DIR}" "${LOG_DIR}"

install -m 0755 "${SCRIPT_DIR}/cmdgate" "${INSTALL_DIR}/cmdgate"
install -m 0750 "${SCRIPT_DIR}/cmdgate-exec" "${INSTALL_DIR}/cmdgate-exec"
install -m 0640 "${SCRIPT_DIR}/allowlist.yaml" "${INSTALL_DIR}/allowlist.yaml"

chmod 0755 "${INSTALL_DIR}"
chmod 0700 "${WORK_DIR}"
chmod 0750 "${LOG_DIR}"

cat > "${SUDOERS_FILE}" <<'EOF'
cmdgateadm ALL=(root) NOPASSWD: /opt/cmdgate/cmdgate-exec *
EOF

chmod 0440 "${SUDOERS_FILE}"
visudo -c

echo "CmdGate installed to ${INSTALL_DIR}"
