package git

import (
	"fmt"
	"os/exec"
	"strings"
)

type Git struct {
	Command string
	Depth   int
}

func runCommand(cmd *exec.Cmd) error {
	out, err := cmd.Output()
	if err != nil {
		if err, ok := err.(*exec.ExitError); ok {
			msg := err.Stderr

			if len(msg) == 0 {
				msg = out
			}

			short := strings.Split(string(msg), "\n")[0]
			return fmt.Errorf("%v: %v", err, short)
		}

		return err
	}

	return nil
}
