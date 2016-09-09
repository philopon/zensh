package main

import (
	"fmt"
	"os/user"

	"github.com/BurntSushi/toml"

	"github.com/philopon/zensh/progress"
	"github.com/philopon/zensh/util"
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
		return "gh-r"
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
		return fmt.Errorf("unknown plugin type: %v", text)
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
	Version    string

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

type GitConfig struct {
	Command string
	Depth   int
	Limit   int
}

type GlobalConfig struct {
	Threads     int
	Directories DirectoryConfig
	Github      GithubConfig
	Git         GitConfig
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
			Git: GitConfig{
				Command: "git",
				Limit:   100,
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
