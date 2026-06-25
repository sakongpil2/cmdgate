# CmdGate Policy Bundle Validation Guide

This guide explains how to create and validate CmdGate policy bundles. The
current CLI supports validation only; it does not apply bundles to
`/opt/cmdgate/allowlist.yaml`.

## Bundle format

A policy bundle is a gzip-compressed tar archive with this structure:

```text
cmdgate-policy-<version>.tar.gz
├── manifest.yaml
├── allowlist.yaml
└── checksums.sha256
```

- `manifest.yaml`: Bundle metadata. `version` is required.
- `allowlist.yaml`: Policy file to validate.
- `checksums.sha256`: SHA-256 hex digest of `allowlist.yaml`.

Validation performs these checks:

1. Read the gzip/tar archive.
2. Confirm all three required files exist.
3. Parse `manifest.yaml` and require a non-empty `version`.
4. Compare the SHA-256 of `allowlist.yaml` with `checksums.sha256`.
5. Parse `allowlist.yaml` and validate schema rules: `version`, `mode`, command
   IDs, matcher references, and supported matcher types.

## Build a bundle

Use the helper script:

```bash
cd /path/to/cmdgate-source
./scripts/build-policy-bundle.sh 1.1.0
```

This creates `scripts/cmdgate-policy-1.1.0.tar.gz` from
`configs/allowlist.yaml`.

To use another allowlist file:

```bash
./scripts/build-policy-bundle.sh 1.1.0 /path/to/allowlist.yaml
```

## Manual bundle creation

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

## Validate a bundle

```bash
cmdgate policy validate --bundle /home/cmdgateadm/cmdgate-policy-1.1.0.tar.gz
```

On success, the command exits with code `0` and produces no output. On failure,
the error describes the failed check, for example:

```text
checksum mismatch
bundle missing required files
invalid allowlist schema: command[0]: id is required
```

`cmdgate-exec` writes a `policy_validate` audit record when validation succeeds.

## Verify the installed policy

Validation does not change the installed policy. To inspect the currently
installed policy, list the allowed commands:

```bash
cmdgate run list
```

To inspect recent audit records:

```bash
cmdgate audit tail 20
```

## Troubleshooting

### `checksum mismatch`

`checksums.sha256` does not match `allowlist.yaml`. Rebuild the bundle or
recompute the checksum.

### `bundle missing required files`

The archive is missing `manifest.yaml`, `allowlist.yaml`, or
`checksums.sha256`. File names must match exactly.

### `invalid allowlist schema`

Common causes:

- Missing `version` or `mode`.
- A command has an empty `id` or `cmd`.
- A placeholder references an undefined matcher.
- A placeholder type does not match its matcher type.
- The matcher type is unsupported.

## Security notes

- Keep bundle files readable only by authorized operators (`0640` or tighter).
- Transfer bundles over trusted channels. The checksum detects accidental
  corruption, not malicious substitution.
- Audit logs are written to `/var/log/cmdgate/audit.log`.
