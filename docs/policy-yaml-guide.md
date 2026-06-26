# CmdGate Policy YAML Validation Guide

This guide explains how to validate an `allowlist.yaml` policy file before
placing it under `/opt/cmdgate/allowlist.yaml`.

## Policy file

CmdGate validates a plain YAML policy file directly:

```text
allowlist.yaml
```

Validation performs these checks:

1. Read the YAML file from the path passed on the command line.
2. Parse it as `allowlist.yaml`.
3. Validate schema rules: `version`, `mode`, command IDs, command strings,
   matcher references, and supported matcher types.

## Validate a policy

```bash
cmdgate policy validate /home/cmdgateadm/allowlist.yaml
```

On success, the command exits with code `0` and prints the validated path:

```text
policy valid: /home/cmdgateadm/allowlist.yaml
```

On failure, the error describes the failed check, for example:

```text
invalid allowlist.yaml: yaml: line 1: did not find expected node content
invalid allowlist schema: command[0]: id is required
```

`cmdgate-exec` writes a `policy_validate` audit record when validation succeeds.

## Install a validated policy

Validation does not change the installed policy. After a policy validates,
copy it into place using your normal privileged deployment flow:

```bash
sudo install -o root -g root -m 0640 /home/cmdgateadm/allowlist.yaml /opt/cmdgate/allowlist.yaml
```

Then inspect the installed policy:

```bash
cmdgate run list
```

To inspect recent audit records:

```bash
cmdgate audit tail 20
```

## Troubleshooting

### `invalid allowlist.yaml`

The file is not valid YAML. Fix the syntax and run validation again.

### `invalid allowlist schema`

Common causes:

- Missing `version` or `mode`.
- A command has an empty `id` or `cmd`.
- A placeholder references an undefined matcher.
- A placeholder type does not match its matcher type.
- The matcher type is unsupported.

## Security notes

- Keep policy files readable only by authorized operators (`0640` or tighter).
- Transfer policies over trusted channels before validation and installation.
- Audit logs are written to `/var/log/cmdgate/audit.log`.
