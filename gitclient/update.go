package gitclient

import "github.com/libgit2/git2go"

func (repo *Repo) GitUpdate() error {
	repo.gitStash("zensh update")

	remote, err := repo.Remotes.Lookup("origin")
	if err != nil {
		return err
	}
	defer remote.Free()

	if err := remote.Fetch([]string{}, &git.FetchOptions{}, ""); err != nil {
		return err
	}

	ref, err := repo.Head()
	if err != nil {
		return err
	}
	defer ref.Free()

	upstream, err := ref.Branch().Upstream()
	if err != nil {
		return err
	}
	defer upstream.Free()

	ac, err := repo.AnnotatedCommitFromRef(upstream)
	if err != nil {
		return err
	}
	defer ac.Free()

	if err := repo.Merge([]*git.AnnotatedCommit{ac}, nil, nil); err != nil {
		return err
	}

	return repo.StateCleanup()
}
