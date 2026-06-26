# CmdGate Deployment Guide

This guide describes how to install CmdGate on an operations host from a
release archive.

## Release Archive Contents

A production archive contains only the files needed on the target host:

```text
cmdgate
cmdgate-exec
allowlist.yaml
install-cmdgate.sh
SHA256SUMS
```

`cmdgate-exec` is the privileged executor. Do not rename it unless you also
update the hard-coded executor path in `cmdgate`, the installer, and sudoers.

## Install On A Host

Copy the archive to the target host and run:

```bash
tar -xzf cmdgate-linux-amd64-<version>.tar.gz
cd cmdgate-linux-amd64-<version>
sha256sum -c SHA256SUMS
sudo ./install-cmdgate.sh
```

The default operator account is `cmdgateadm`. To use another account:

```bash
sudo CMDGATE_USER=myops ./install-cmdgate.sh
```

The installer creates `/opt/cmdgate`, `/opt/cmdgate/work`, and
`/var/log/cmdgate`, installs both binaries and `allowlist.yaml`, writes
`/etc/sudoers.d/cmdgate`, validates it with `visudo`, and links
`/usr/local/bin/cmdgate`.

## Post-Install Checks

Run these checks as the operator user:

```bash
cmdgate --help
cmdgate policy validate /opt/cmdgate/allowlist.yaml
cmdgate run list
cmdgate audit tail 20
```

A valid policy prints:

```text
policy valid: /opt/cmdgate/allowlist.yaml
```

Success messages are green and failure messages are red when stdout/stderr are
terminals. Set `NO_COLOR=1` to disable color.

## Update The Policy

Validate a new policy before replacing the installed file:

```bash
cmdgate policy validate /home/cmdgateadm/allowlist.yaml
sudo install -o root -g root -m 0640 /home/cmdgateadm/allowlist.yaml /opt/cmdgate/allowlist.yaml
cmdgate run list
```

## Build A Release Archive

From the repository root:

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

## GitHub Release Flow

Use annotated tags for releases:

```bash
git status --short
GOCACHE=/tmp/go-cache go test ./...
git tag -a v1.1.0 -m "CmdGate v1.1.0"
git push origin main
git push origin v1.1.0
```

If the GitHub CLI is available, create a release and upload the archive:

```bash
gh release create v1.1.0 \
  dist/cmdgate-linux-amd64-1.1.0.tar.gz \
  --title "CmdGate v1.1.0" \
  --notes "See README.md and docs/deployment-guide.md for installation steps."
```

Without `gh`, create a release from the pushed tag in the GitHub web UI and
upload the `cmdgate-linux-amd64-<version>.tar.gz` asset.
