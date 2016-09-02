package gitclient

import "github.com/libgit2/git2go"

type GitClient struct {
	GitUserName  string
	GitUserEmail string
}

func registerConfig(cfg *git.Config, getter func() (string, error), level git.ConfigLevel, force bool) {
	path, err := getter()
	if err != nil {
		return
	}

	cfg.AddFile(path, level, force)
}

func LoadGitConfig(force bool) (*git.Config, error) {
	cfg, err := git.NewConfig()
	if err != nil {
		return nil, err
	}

	registerConfig(cfg, git.ConfigFindGlobal, git.ConfigLevelGlobal, force)
	registerConfig(cfg, git.ConfigFindProgramdata, git.ConfigLevelProgramdata, force)
	registerConfig(cfg, git.ConfigFindSystem, git.ConfigLevelSystem, force)
	registerConfig(cfg, git.ConfigFindXDG, git.ConfigLevelXDG, force)

	return cfg, nil
}

func getGitUserInfo(defName string, defEmail string) (string, string) {
	cfg, err := LoadGitConfig(false)
	if err != nil {
		return defName, defEmail
	}
	defer cfg.Free()

	name, err := cfg.LookupString("user.name")
	if err != nil {
		name = defName
	}

	email, err := cfg.LookupString("user.email")
	if err != nil {
		email = defEmail
	}

	return name, email
}

func NewGitClient() GitClient {
	name, email := getGitUserInfo("unknown", "unknown")
	return GitClient{
		GitUserName:  name,
		GitUserEmail: email,
	}
}

type Repo struct {
	*git.Repository
	parent *GitClient
}

func (f *GitClient) OpenRepository(path string) (*Repo, error) {
	repo, err := git.OpenRepository(path)
	if err != nil {
		return nil, err
	}

	return &Repo{Repository: repo, parent: f}, nil
}
