package gitclient

import "github.com/libgit2/git2go"

func (r *Repo) gitStash(msg string) {
	r.Stashes.Save(
		&git.Signature{
			Name:  r.parent.GitUserName,
			Email: r.parent.GitUserEmail,
		},
		msg,
		git.StashDefault,
	)
}
