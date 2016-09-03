package gitclient

import (
	"github.com/libgit2/git2go"
	"github.com/pkg/errors"
)

func updateSubmoduleRecursive(sub *git.Submodule, name string) int {
	if err := sub.Update(true, nil); err != nil {
		return 1
	}

	repo, err := sub.Open()
	if err != nil {
		return 1
	}
	defer repo.Free()

	if err := repo.Submodules.Foreach(updateSubmoduleRecursive); err != nil {
		return 1
	}

	return 0
}

func (gc *GitClient) Clone(url, path string) error {
	repo, err := git.Clone(url, path, &git.CloneOptions{})
	if err != nil {
		return errors.Wrap(err, "clone failed")
	}
	defer repo.Free()

	if err := repo.Submodules.Foreach(updateSubmoduleRecursive); err != nil {
		return err
	}

	return nil
}
