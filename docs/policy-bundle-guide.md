# CmdGate Policy Bundle Update Guide

This guide explains how to create, validate, apply, and roll back CmdGate policy
bundles. A policy bundle is the only supported way to update
`/opt/cmdgate/allowlist.yaml` after the initial installation.

## What is a policy bundle?

A policy bundle is a gzip-compressed tar archive with a fixed structure:

```text
cmdgate-policy-<version>.tar.gz
├── manifest.yaml
├── allowlist.yaml
└── checksums.sha256
```

- `manifest.yaml` — bundle metadata (`version`, `timestamp`).
- `allowlist.yaml` — the new policy file.
- `checksums.sha256` — SHA-256 hex digest of `allowlist.yaml`.

`cmdgate-exec` verifies the bundle before applying it:

1. Extract the archive.
2. Confirm all three required files exist.
3. Parse `manifest.yaml` and require a non-empty `version`.
4. Compute the SHA-256 of `allowlist.yaml` and compare it to `checksums.sha256`.
5. Parse `allowlist.yaml` and run `ValidateSchema()` (version, mode, command IDs,
   matcher references, and supported matcher types).

Only when every check passes does `cmdgate-exec` back up the current policy and
replace it.

## Build a bundle

Use the helper script shipped with the source:

```bash
cd /path/to/cmdgate-source
./scripts/build-policy-bundle.sh 1.1.0
```

This creates `scripts/cmdgate-policy-1.1.0.tar.gz` from
`configs/allowlist.yaml`.

To use a different allowlist file:

```bash
./scripts/build-policy-bundle.sh 1.1.0 /path/to/your/allowlist.yaml
```

### Build a bundle manually

If you cannot use the helper script, run these commands manually:

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

# bundle
tar -czf "cmdgate-policy-${VERSION}.tar.gz" -C "${WORK_DIR}" \
  manifest.yaml allowlist.yaml checksums.sha256

rm -rf "${WORK_DIR}"
```

## Validate a bundle

Validation checks the bundle without touching the live policy:

```bash
cmdgate policy validate --bundle /home/cmdgateadm/cmdgate-policy-1.1.0.tar.gz
```

On success, no output is produced and the exit code is `0`.

On failure, the error explains which check failed, for example:

```text
checksum mismatch
bundle missing required files
invalid allowlist schema: command[0]: id is required
```

## Apply a bundle

Applying a bundle replaces `/opt/cmdgate/allowlist.yaml`:

```bash
cmdgate policy apply --bundle /home/cmdgateadm/cmdgate-policy-1.1.0.tar.gz
```

`cmdgate-exec` performs the following steps:

1. Validate the bundle.
2. Back up the current policy to `/opt/cmdgate/allowlist.yaml.backup`.
3. Write the new policy to `/opt/cmdgate/allowlist.yaml` with permissions `0640`.
4. Write an audit record with `action=policy_apply`.

If any step fails, the live policy is left unchanged.

## Roll back a failed update

If a new policy causes problems, restore the backup:

```bash
sudo cp /opt/cmdgate/allowlist.yaml.backup /opt/cmdgate/allowlist.yaml
sudo chmod 0640 /opt/cmdgate/allowlist.yaml
```

Then verify the policy loads correctly:

```bash
cmdgate run list
```

## Verify the live policy

After applying a bundle, list the allowed commands to confirm the new policy:

```bash
cmdgate run list
```

To inspect recent policy-related audit records:

```bash
cmdgate audit tail 20
```

## Troubleshooting

### `checksum mismatch`

The `checksums.sha256` file does not match the SHA-256 of `allowlist.yaml`.
Regenerate the bundle with the helper script or recompute the checksum manually.

### `bundle missing required files`

The archive is missing `manifest.yaml`, `allowlist.yaml`, or
`checksums.sha256`. Confirm the file names are exact.

### `invalid allowlist schema`

`allowlist.yaml` failed semantic validation. Common causes:

- Missing `version` or `mode`.
- A command with an empty `id` or `cmd`.
- A placeholder references an undefined matcher.
- A placeholder type does not match the matcher type, e.g.
  `<number:lines>` defined with a matcher whose `type` is `string`.
- An unsupported matcher type.

### Permission denied when applying

`cmdgate-exec` runs as `root` via `sudo`, so the apply itself requires the
operator account to have the sudoers rule for `cmdgate-exec`. If the apply
fails with a file-system error, check that `/opt/cmdgate` is writable by root
and that the current `allowlist.yaml` is not immutable.

## Security notes

- Keep bundle files readable only by authorized operators (`0640` or tighter).
- Transfer bundles over trusted channels; the checksum only protects against
  accidental corruption, not malicious substitution. Sign bundles separately if
  you need origin verification.
- Treat `/opt/cmdgate/allowlist.yaml.backup` as sensitive; it contains the
  previous policy.
- Audit logs (`/var/log/cmdgate/audit.log`) record every validate and apply
  attempt.
