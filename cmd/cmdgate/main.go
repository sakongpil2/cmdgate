package main

import (
	"fmt"
	"os"
	"os/exec"
)

const execBinary = "/opt/cmdgate/cmdgate-exec"

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: cmdgate <run|policy> ...")
		os.Exit(1)
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

func buildExecCommand(args []string) []string {
	return append([]string{"-n", execBinary}, args...)
}
