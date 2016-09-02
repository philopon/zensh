package github_release

import (
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"

	"github.com/google/go-github/github"
)

type Client github.Client

type AssetNotFound string

func (anf AssetNotFound) Error() string {
	return fmt.Sprintf("binary release of %v for %v-%v not found", string(anf), runtime.GOOS, runtime.GOARCH)
}

type StatusCodeError int

func (sce StatusCodeError) Error() string {
	return http.StatusText(int(sce))
}

func checkOS(name string) bool {
	switch os := runtime.GOOS; os {
	case "darwin":
		return strings.Contains(name, "darwin") || strings.Contains(name, "osx")
	default:
		return strings.Contains(name, os)
	}
}

func checkArch(name string) bool {
	switch arch := runtime.GOARCH; arch {
	case "amd64":
		return strings.Contains(name, "amd64") || strings.Contains(name, "x86_64") || strings.Contains(name, "x86-64")
	case "386":
		if strings.Contains(name, "x86_64") {
			return false
		}
		return strings.Contains(name, "386") || strings.Contains(name, "x86")
	default:
		return strings.Contains(name, arch)
	}
}

type Asset struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Owner  string `json:"owner"`
	Repo   string `json:"repo"`
	parent *Client
}

func (a *Asset) Open() (io.ReadCloser, error) {
	rdr, url, err := a.parent.Repositories.DownloadReleaseAsset(a.Owner, a.Repo, a.ID)
	if err != nil {
		return nil, err
	}

	if rdr == nil {
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		if !(200 <= resp.StatusCode && resp.StatusCode < 300) {
			return nil, StatusCodeError(resp.StatusCode)
		}

		rdr = resp.Body
	}

	return rdr, nil
}

func (gh *Client) Fetch(owner, repo, tag string) (*Asset, error) {
	var release *github.RepositoryRelease
	var err error
	if tag == "" {
		release, _, err = gh.Repositories.GetLatestRelease(owner, repo)
	} else {
		release, _, err = gh.Repositories.GetReleaseByTag(owner, repo, tag)
	}
	if err != nil {
		return nil, err
	}

	var asset *github.ReleaseAsset
	for _, a := range release.Assets {
		if a.ID == nil || a.Name == nil || a.ContentType == nil {
			continue
		}
		name := strings.ToLower(*a.Name)
		if checkArch(name) && checkOS(name) && strings.HasPrefix(*a.ContentType, "application") {
			asset = &a
			break
		}
	}

	if asset == nil {
		return nil, AssetNotFound(owner + "/" + repo)
	}

	return &Asset{ID: *asset.ID, Name: *asset.Name, Owner: owner, Repo: repo, parent: gh}, nil
}
