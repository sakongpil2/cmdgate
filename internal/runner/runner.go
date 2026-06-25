package runner

import (
	"io"
	"os/exec"
)

// RunWithIO runs cmd with args, attaching the supplied stdin/stdout/stderr,
// and returns the error from exec.Cmd.Run(). If the command exits with a
// non-zero status, the returned error is *exec.ExitError so callers can
// propagate the original exit code.
func RunWithIO(cmd string, args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	c := exec.Command(cmd, args...)
	c.Stdin = stdin
	c.Stdout = stdout
	c.Stderr = stderr
	return c.Run()
}
