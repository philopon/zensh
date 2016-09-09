package git

import (
	"os/exec"
	"strconv"
)

func (g *Git) Clone(url, dir string) error {
	args := []string{"clone", "--recursive", url, dir}

	if g.Depth > 0 {
		args = append(args, "--depth", strconv.Itoa(g.Depth))
	}

	return runCommand(exec.Command(g.Command, args...))
}
