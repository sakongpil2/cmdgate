#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
INSTALL_DIR="/opt/cmdgate"
LOG_DIR="/var/log/cmdgate"
WORK_DIR="${INSTALL_DIR}/work"
SUDOERS_FILE="/etc/sudoers.d/cmdgate"

# The operator account allowed to run cmdgate-exec via sudo.
# Override with: CMDGATE_USER=myops ./install-cmdgate.sh
CMDGATE_USER="${CMDGATE_USER:-cmdgateadm}"

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

if ! id -u "${CMDGATE_USER}" >/dev/null 2>&1; then
    cat >&2 <<EOF
Warning: user '${CMDGATE_USER}' does not exist.
The sudoers rule will be installed, but it will only take effect after
the system administrator creates the '${CMDGATE_USER}' user and adds them to
the appropriate groups.

Example:
    useradd -r -s /sbin/nologin ${CMDGATE_USER}
EOF
fi

mkdir -p "${INSTALL_DIR}" "${WORK_DIR}" "${LOG_DIR}"

install -m 0755 "${SCRIPT_DIR}/cmdgate" "${INSTALL_DIR}/cmdgate"
install -m 0750 "${SCRIPT_DIR}/cmdgate-exec" "${INSTALL_DIR}/cmdgate-exec"
install -m 0640 "${SCRIPT_DIR}/allowlist.yaml" "${INSTALL_DIR}/allowlist.yaml"

chmod 0755 "${INSTALL_DIR}"
chmod 0700 "${WORK_DIR}"
chmod 0750 "${LOG_DIR}"

SCRIPTS_DIR="${INSTALL_DIR}/scripts"
mkdir -p "${SCRIPTS_DIR}"
chmod 0750 "${SCRIPTS_DIR}"

# shellcheck disable=SC2016
cat > "${SUDOERS_FILE}" <<EOF
${CMDGATE_USER} ALL=(root) NOPASSWD: /opt/cmdgate/cmdgate-exec *
EOF

chmod 0440 "${SUDOERS_FILE}"
visudo -c -f "${SUDOERS_FILE}"

PROFILE_D_FILE="/etc/profile.d/cmdgate.sh"
printf 'export PATH="/opt/cmdgate:${PATH}"\n' > "${PROFILE_D_FILE}"
chmod 0644 "${PROFILE_D_FILE}"

echo "CmdGate installed to ${INSTALL_DIR} for operator user '${CMDGATE_USER}'"
