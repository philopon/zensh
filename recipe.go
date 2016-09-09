package main

import (
	"bufio"
	"container/list"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/philopon/gogit"
	"github.com/philopon/zensh/github_release"
	"github.com/philopon/zensh/util"
)

func (r *Recipe) Directory() string {
	var prefix string

	switch r.Source {
	case Github:
		prefix = "github.com"
	case GithubRelease:
		prefix = "github-release"
	case Local:
		path := util.SafeExpandPath(r.parent.HomeDir, r.Repo)
		if filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(r.parent.Config.Directories.Local, path)
	}

	return filepath.Join(
		r.parent.Config.Directories.Repo,
		prefix, filepath.FromSlash(r.Repo),
	)
}

func (r *Recipe) IsInstalled() bool {
	_, err := os.Stat(r.Directory())
	return err == nil
}

func (r *Recipe) installGithub() error {
	r.task.Update("cloning ...")

	url := "https://github.com/" + r.Repo + ".git"

	if err := r.parent.GitClient.Clone(url, r.Directory()); err != nil {
		return err
	}

	if r.Version != "" {
		r.task.Update(fmt.Sprintf("checkout %v ...", r.Version))
		if err := r.parent.GitClient.Checkout(r.Directory(), r.Version); err != nil {
			return err
		}
	}

	return nil
}

func (r *Recipe) installGithubRelease() error {
	var owner, repo string
	if fs := strings.Split(r.Repo, "/"); len(fs) == 2 {
		owner = fs[0]
		repo = fs[1]
	} else {
		return fmt.Errorf("invalid repository name: %v", r.Repo)
	}

	r.task.Update("fetching release information ...")
	asset, err := r.parent.GithubClient.Fetch(owner, repo, r.Version)
	if err != nil {
		return err
	}

	r.task.Update(fmt.Sprintf("downloading %v ...", asset.Name))
	rc, err := asset.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	if err := util.Unarchive(r.Directory(), asset.Name, bufio.NewReader(rc)); err != nil {
		return err
	}

	js, err := os.Create(filepath.Join(r.Directory(), "..", repo+".json"))
	if err != nil {
		return err
	}
	defer js.Close()

	writer := json.NewEncoder(js)
	if err := writer.Encode(asset); err != nil {
		return err
	}

	return nil
}

func (r *Recipe) Install() error {
	if r.IsInstalled() {
		return nil
	}

	switch r.Source {
	case Github:
		return r.installGithub()

	case GithubRelease:
		return r.installGithubRelease()
	}

	return nil
}

type Hash interface {
	fmt.Stringer
	AsOid() (*gogit.Oid, bool)
	AsInt() (int, bool)
}

type intHash int

func (i intHash) String() string {
	return strconv.Itoa(int(i))
}

func (i intHash) AsInt() (int, bool) {
	return int(i), true
}

func (i intHash) AsOid() (*gogit.Oid, bool) {
	return nil, false
}

type oidHash gogit.Oid

func (o *oidHash) String() string {
	return (*gogit.Oid)(o).String()
}

func (o *oidHash) AsInt() (int, bool) {
	return 0, false
}

func (o *oidHash) AsOid() (*gogit.Oid, bool) {
	return (*gogit.Oid)(o), true
}

func (r *Recipe) getHashGithubRelease() (intHash, error) {
	path := r.Directory() + ".json"

	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var asset github_release.Asset
	dec := json.NewDecoder(file)
	dec.Decode(&asset)

	return intHash(asset.ID), nil
}

func (r *Recipe) openGitRepository() (*gogit.Repository, error) {
	return gogit.OpenRepository(filepath.Join(r.Directory(), ".git"))
}

func (r *Recipe) getHashGit() (*oidHash, error) {
	repo, err := r.openGitRepository()
	if err != nil {
		return nil, err
	}

	head, err := repo.LookupReference("HEAD")
	if err != nil {
		return nil, err
	}

	return (*oidHash)(head.Target()), nil
}

var NoHashError = errors.New("no hash")

func (r *Recipe) GetHash() (Hash, error) {
	if !r.IsInstalled() {
		if r.Source == Local {
			return intHash(0), fmt.Errorf("not found: %v", r.Repo)
		}
		return intHash(0), fmt.Errorf("not installed: %v", r.Repo)
	}

	switch r.Source {
	case GithubRelease:
		return r.getHashGithubRelease()
	case Github:
		return r.getHashGit()
	}

	return nil, NoHashError
}

var DepthLimitError = errors.New("no commit history")
var NoHistoryError = errors.New("no commit history")

func commitsBetween(new, old *gogit.Commit, limit int) ([]*gogit.Commit, error) {
	oldId := *old.Id()
	newId := *new.Id()

	if newId == oldId {
		return []*gogit.Commit{}, nil
	}

	stack := list.New()
	vis := make(map[gogit.Oid]*gogit.Commit)

	vis[newId] = nil
	stack.PushFront(new)
	depth := 0

	for stack.Len() > 0 {
		if limit != 0 && depth > limit {
			return nil, DepthLimitError
		}

		element := stack.Back()
		stack.Remove(element)
		v := element.Value.(*gogit.Commit)
		vId := *v.Id()

		if vId == oldId {
			cur := vis[oldId]
			commits := []*gogit.Commit{cur}

			for {
				cur = vis[*cur.Id()]
				if cur == nil {
					break
				}
				commits = append(commits, cur)
			}

			return commits, nil
		}

		for i := 0; i < v.ParentCount(); i++ {
			u := v.Parent(i)
			uId := *u.Id()

			if _, visited := vis[uId]; !visited {
				vis[uId] = v
				stack.PushFront(u)
			}
		}

		depth += 1
	}

	return []*gogit.Commit{}, NoHistoryError
}

func (r *Recipe) gitHasUpdate(fetch bool) (bool, error) {
	if fetch {
		if err := r.parent.GitClient.Fetch(r.Directory()); err != nil {
			return false, err
		}
	}

	repo, err := r.openGitRepository()
	if err != nil {
		return false, err
	}

	newRef, err := repo.LookupReference("FETCH_HEAD")
	if err != nil {
		return false, err
	}

	curRef, err := repo.LookupReference("HEAD")
	if err != nil {
		return false, err
	}

	return *newRef.Target() != *curRef.Target(), nil
}

var NoInfoError = errors.New("no info")

func (r *Recipe) HasUpdate(fetch bool) (bool, error) {
	switch r.Source {
	case Github:
		return r.gitHasUpdate(fetch)
	case Local:
		return false, nil
	}

	return false, NoInfoError
}
