# CmdGate

CmdGate is an allowlist-based CLI tool that lets operators run only pre-approved commands through delegated privilege.

- User binary: `cmdgate`
- Privileged executor: `cmdgate-exec`
- Policy file: `/opt/cmdgate/allowlist.yaml`
- Audit log: `/var/log/cmdgate/audit.log`

`cmdgate` receives user input and invokes `cmdgate-exec` via `sudo -n`. `cmdgate-exec` validates the request against the allowlist, runs any configured matchers, executes the command as an argv array, and writes an audit record.

> 한국어 문서는 [README.ko.md](README.ko.md)를 참고하세요.

## Binary responsibilities

### `cmdgate`

The user-facing CLI. It does not validate input; it forwards arguments unchanged to `cmdgate-exec`.

Supported commands:

- `cmdgate run <command> [args...]`
- `cmdgate run list`
- `cmdgate policy validate --bundle <tar.gz>`
- `cmdgate audit tail [n]`
- `cmdgate help`
- `cmdgate --help`

Internally it calls:

```bash
sudo -n /opt/cmdgate/cmdgate-exec <subcommand> [args...]
```

### `cmdgate-exec`

The privileged executor, intended to run only through `sudo`. It reads `/opt/cmdgate/allowlist.yaml`, compares the user's argv against `commands[].cmd`, runs matchers for any placeholders, and executes matching commands with `exec.Command(cmd, args...)`.

Supported commands:

- `cmdgate-exec run <command> [args...]`
- `cmdgate-exec run list`
- `cmdgate-exec policy validate --bundle <tar.gz>`
- `cmdgate-exec audit tail [n]`
- `cmdgate-exec help`
- `cmdgate-exec --help`

Main flow:

1. Load `/opt/cmdgate/allowlist.yaml`.
2. Match the user's argv to a policy command.
3. Validate any `<type:name>` placeholders with matchers.
4. Execute the command as an argv array.
5. Write the result to `/var/log/cmdgate/audit.log`.

## Build

Requires Go 1.22 or later.

```bash
go build ./cmd/...
```

This produces the `cmdgate` and `cmdgate-exec` binaries.

## Install

Use `scripts/install-cmdgate.sh`. The script must be run as root.

The default operator account is `cmdgateadm`. To use a different operator
account, set the `CMDGATE_USER` environment variable before running the
installer. The installer does not create the user; you must create it first.

```bash
# Copy built binaries and the default policy into the script directory
cp cmdgate cmdgate-exec configs/allowlist.yaml scripts/

# Run the installer as root with the default operator (cmdgateadm)
sudo ./scripts/install-cmdgate.sh

# Or use a custom operator account
# sudo CMDGATE_USER=myops ./scripts/install-cmdgate.sh
```

The installer performs the following:

1. Creates `/opt/cmdgate`, `/opt/cmdgate/work`, and `/var/log/cmdgate`.
2. Copies `cmdgate`, `cmdgate-exec`, and `allowlist.yaml` into `/opt/cmdgate`.
3. Sets file and directory permissions.
4. Creates `/etc/sudoers.d/cmdgate` for the operator account.
5. Validates the sudoers file with `visudo -c`.
6. Adds `/opt/cmdgate` to the system `PATH` via `/etc/profile.d/cmdgate.sh`
   so users can run `cmdgate` without typing the full path.

After installation the following sudoers rule is active (replace `cmdgateadm`
with your `CMDGATE_USER` value if you changed it):

```sudoers
cmdgateadm ALL=(root) NOPASSWD: /opt/cmdgate/cmdgate-exec *
```

If the operator user does not exist, the installer prints a warning. Create
the user before invoking `cmdgate`:

```bash
useradd -r -s /sbin/nologin cmdgateadm
```

## Usage examples

### List allowed commands

```bash
cmdgate run list
```

### Run an allowed command

```bash
cmdgate run systemctl restart kubelet
cmdgate run journalctl -u kubelet -n 50 --no-pager
```

### Validate a policy bundle

```bash
cmdgate policy validate --bundle cmdgate-policy-1.1.0.tar.gz
```

### View audit log

```bash
cmdgate audit tail      # latest 20 entries
cmdgate audit tail 50   # latest 50 entries
```

The output is the latest key-value audit log lines from `/var/log/cmdgate/audit.log`.

### Get help

```bash
cmdgate help
cmdgate --help
```

## allowlist.yaml format

`allowlist.yaml` defines allowed commands and matchers.

```yaml
version: 1.0.0
mode: allowlist-only

commands:
  - id: systemctl-restart-kubelet
    desc: Restart kubelet
    cmd: "systemctl restart kubelet"

  - id: journalctl-kubelet-lines
    desc: View kubelet logs with line count
    cmd: "journalctl -u kubelet -n <number:lines> --no-pager"

  - id: dnf-install-local-rpms
    desc: Install local RPM files
    cmd: "dnf install <rpmFiles:k8s-rpms>"

matchers:
  k8s-rpms:
    type: rpmFiles
    multiple: true
    metadataNameIn:
      - kubelet
      - containerd
      - containerd.io
      - kubeadm
      - cri-tools
      - kubectl
      - kubernetes-cni

  lines:
    type: number
```

### Command definitions

- `version`: Policy file version.
- `mode`: Policy mode. Currently only `allowlist-only` is supported.
- `commands`: List of allowed commands.
- `matchers`: Matcher definitions referenced by placeholders.

### Matchers

A placeholder in the form `<type:name>` delegates validation of that argument to a matcher.

#### `string`

Validates that the placeholder argument is a non-empty string. An optional
`pattern` field restricts the value with a regular expression.

```yaml
cmd: "/opt/cmdgate/scripts/<string:script>"
```

```yaml
matchers:
  script:
    type: string
    pattern: '^(?:[a-zA-Z0-9_-]+/)*[a-zA-Z0-9_-]+\.sh$'
```

This example allows `backup.sh` or `maintenance/reboot.sh` while rejecting
`..`, absolute paths, and non-`.sh` files.

#### `number`

Validates that the placeholder argument is a base-10 integer.

```yaml
cmd: "journalctl -u kubelet -n <number:lines> --no-pager"
```

Only values such as `50` or `100` are accepted.

#### `rpmFiles`

Validates that the RPM `NAME` metadata of every provided RPM file is contained in `metadataNameIn`.

```yaml
cmd: "dnf install <rpmFiles:k8s-rpms>"
```

- `type`: `rpmFiles`
- `multiple`: When `true`, multiple RPM files may be supplied at once.
- `metadataNameIn`: List of allowed RPM package names.

## Audit log

Every execution attempt is recorded in `/var/log/cmdgate/audit.log`.

Log fields:

- `timestamp`: Event time
- `user`: The user who invoked the command
- `action`: Action type (`run`, `policy_validate`)
- `command_id`: Matched allowlist command ID, if any
- `command`: The command the user entered
- `result`: `success` or `denied`
- `reason`: Reason for denial, if denied

## Security notes

- CmdGate executes commands as **argv arrays only**. It never uses `bash -c`, `sh -c`, or `eval`.
- Privilege escalation happens only through `sudo`; `cmdgate-exec` is the only binary granted passwordless sudo access.
- `cmdgate` does not validate input, so `cmdgate-exec` must remain the single point of authorization.
- Restrict access to `/opt/cmdgate/allowlist.yaml` so that regular users cannot modify the policy.
- The sudoers rule should allow only the operator user (`cmdgateadm` by default)
  to run `/opt/cmdgate/cmdgate-exec`:

```sudoers
cmdgateadm ALL=(root) NOPASSWD: /opt/cmdgate/cmdgate-exec *
```

To change the operator account, set `CMDGATE_USER` when running the installer
and ensure the user exists before invoking `cmdgate`.

- Policy bundles are validated against a manifest and SHA-256 checksum.
