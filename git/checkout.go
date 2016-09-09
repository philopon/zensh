package git

import "os/exec"

func (g *Git) Checkout(dir, version string) error {
	cmd := exec.Command(g.Command, "checkout", "-b", version, "origin/"+version)
	cmd.Dir = dir

	if err := runCommand(cmd); err != nil {
		cmd := exec.Command(g.Command, "checkout", "-b", version, version)
		cmd.Dir = dir
		return runCommand(cmd)
	}

	return nil
}
