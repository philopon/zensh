package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	git "github.com/libgit2/git2go"
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
	r.task.Update("cloning...")
	return r.parent.GitClient.Clone(
		"https://github.com/"+r.Repo+".git",
		r.Directory(),
	)
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
	asset, err := r.parent.GithubClient.Fetch(owner, repo, "")
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
	AsOid() (*git.Oid, bool)
	AsInt() (int, bool)
}

type intHash int

func (i intHash) String() string {
	return strconv.Itoa(int(i))
}

func (i intHash) AsInt() (int, bool) {
	return int(i), true
}

func (i intHash) AsOid() (*git.Oid, bool) {
	return nil, false
}

type oidHash git.Oid

func (o *oidHash) String() string {
	return (*git.Oid)(o).String()
}

func (o *oidHash) AsInt() (int, bool) {
	return 0, false
}

func (o *oidHash) AsOid() (*git.Oid, bool) {
	return (*git.Oid)(o), true
}

type noHash int

func (n noHash) String() string {
	return ""
}

func (o noHash) AsInt() (int, bool) {
	return 0, false
}

func (o noHash) AsOid() (*git.Oid, bool) {
	return nil, false
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

func (r *Recipe) getHashGit() (*oidHash, error) {
	repo, err := r.parent.GitClient.OpenRepository(r.Directory())
	if err != nil {
		return nil, err
	}
	defer repo.Free()

	head, err := repo.Head()
	if err != nil {
		return nil, err
	}
	defer head.Free()

	return (*oidHash)(head.Target()), nil
}

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
	case Local:
		return noHash(0), nil
	}

	return intHash(0), nil
}
