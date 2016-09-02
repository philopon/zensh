package gitclient

import "github.com/libgit2/git2go"

func (repo *Repo) lookupCommitByRef(refStr string) *git.Commit {
	ref, err := repo.References.Lookup(refStr)
	if err != nil {
		return nil
	}
	defer ref.Free()

	cmt, err := repo.LookupCommit(ref.Target())
	if err != nil {
		return nil
	}

	return cmt
}

func (repo *Repo) getCommit(target string) (*git.Commit, string, error) {
	if cmt := repo.lookupCommitByRef("refs/remotes/origin/" + target); cmt != nil {
		return cmt, target, nil
	}

	if cmt := repo.lookupCommitByRef("refs/tags/" + target); cmt != nil {
		return cmt, target, nil
	}

	obj, err := repo.RevparseSingle(target)
	if err != nil {
		return nil, "", err
	}
	defer obj.Free()

	cmt, err := obj.AsCommit()
	if err != nil {
		return nil, "", err
	}

	if obj.Type() == git.ObjectTag {
		return cmt, target, err
	}

	return cmt, obj.Id().String(), err
}

func (repo *Repo) Checkout(target string) error {
	repo.gitStash("zensh checkout")

	cmt, branchName, err := repo.getCommit(target)
	if err != nil {
		return err
	}
	defer cmt.Free()

	branch, err := repo.LookupBranch(branchName, git.BranchLocal)
	if err != nil {
		branch, err = repo.CreateBranch(branchName, cmt, false)
		if err != nil {
			return err
		}

		if err := branch.SetUpstream(branchName); err != nil {
			return err
		}
	}
	defer branch.Free()

	if err := repo.SetHead(branch.Reference.Name()); err != nil {
		return err
	}

	return repo.CheckoutHead(&git.CheckoutOpts{
		Strategy: git.CheckoutForce,
	})
}
