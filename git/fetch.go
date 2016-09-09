package git

import "os/exec"

func (g *Git) Fetch(dir string) error {
	cmd := exec.Command(g.Command, "fetch")
	cmd.Dir = dir

	return runCommand(cmd)
}
