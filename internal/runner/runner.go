package runner

import (
	"os/exec"
)

func Run(cmd string, args []string) ([]byte, error) {
	return exec.Command(cmd, args...).CombinedOutput()
}
