package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
)

const execBinary = "/opt/cmdgate/cmdgate-exec"

var stdout io.Writer = os.Stdout

func main() {
	if len(os.Args) < 2 || os.Args[1] == "help" || os.Args[1] == "--help" {
		printHelp()
		return
	}

	cmd := exec.Command("sudo", buildExecCommand(os.Args[1:])...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Fprintln(stdout, `CmdGate - allowlist-based privileged command executor

Usage:
  cmdgate <command> [args...]

Commands:
  run     Run a pre-approved command
  policy  Validate or apply a policy bundle
  audit   View audit logs
  help    Show this help message

Examples:
  cmdgate run list
  cmdgate run systemctl restart kubelet
  cmdgate policy validate --bundle cmdgate-policy-1.1.0.tar.gz
  cmdgate audit tail 50`)
}

func buildExecCommand(args []string) []string {
	return append([]string{"-n", execBinary}, args...)
}
