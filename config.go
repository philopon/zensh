package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"./progress"
	"./util"
)

// Source
type Source int

const (
	Github Source = iota
	GithubRelease
	Local
)

func (s Source) String() string {
	switch s {
	case Github:
		return "github"
	case GithubRelease:
		return "github-release"
	default:
		return "local"
	}
}

func (s *Source) UnmarshalText(bytes []byte) error {
	text := string(bytes)

	switch text {
	case "github":
		*s = Github

	case "local":
		*s = Local

	case "github-release":
		*s = GithubRelease

	default:
		return fmt.Errorf("unknown source: %v", text)
	}

	return nil
}

// PluginType
type PluginType int

const (
	Plugin PluginType = iota
	Command
)

func (s PluginType) String() string {
	switch s {
	case Plugin:
		return "plugin"
	default:
		return "command"
	}
}

func (s *PluginType) UnmarshalText(bytes []byte) error {
	text := string(bytes)

	switch text {
	case "plugin":
		*s = Plugin

	case "command":
		*s = Command

	default:
		return fmt.Errorf("unknown source: %v", text)
	}

	return nil
}

type Condition struct {
	Hostname   string
	ZshVersion string `toml:"zsh"`
}

type Hook struct {
	Load  string
	Build string
}

func (hook Hook) String() string {
	return fmt.Sprintf("{%v %v}", hook.Load, hook.Build)
}

type Recipe struct {
	Repo       string
	Source     Source
	Hook       Hook
	Condition  Condition  `toml:"on"`
	PluginType PluginType `toml:"as"`

	// plugin only
	AfterCompinit bool `toml:"after_compinit"`

	// command only
	Rename string

	// internal
	parent *Zensh
	task   *progress.Task
}

type DirectoryConfig struct {
	Repo  string
	Local string
}

type GithubConfig struct {
	Token string
}

type GlobalConfig struct {
	Threads     int
	Directories DirectoryConfig
	Github      GithubConfig
}

type Config struct {
	GlobalConfig GlobalConfig `toml:"config"`
	Plugins      []*Recipe    `toml:"plugin"`
}

func LoadConfig(path string) (*Config, error) {
	config := &Config{
		GlobalConfig: GlobalConfig{
			Threads: 8,
			Directories: DirectoryConfig{
				Repo:  "~/.zensh",
				Local: "~/.zensh/local",
			},
		},
	}

	if _, err := toml.DecodeFile(path, config); err != nil {
		return nil, err
	}

	user, err := user.Current()
	if err != nil {
		return nil, err
	}

	config.GlobalConfig.Directories.Repo = util.SafeExpandPath(user.HomeDir, config.GlobalConfig.Directories.Repo)
	config.GlobalConfig.Directories.Local = util.SafeExpandPath(user.HomeDir, config.GlobalConfig.Directories.Local)

	return config, nil
}

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

	r.task.Update("fetching release information...")
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
