package main

import (
	"fmt"
	"net/http"
	"os/user"
	"sync"

	"github.com/google/go-github/github"
	"golang.org/x/oauth2"

	"github.com/philopon/zensh/gitclient"
	"github.com/philopon/zensh/github_release"
	"github.com/philopon/zensh/progress"
	"github.com/philopon/zensh/util"
)

type Zensh struct {
	Config       *GlobalConfig
	Plugins      []*Recipe
	GitClient    gitclient.GitClient
	GithubClient *github_release.Client
	HomeDir      string
}

func NewZensh(config *Config) (*Zensh, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}

	var ghTc *http.Client
	if tok := config.GlobalConfig.Github.Token; tok == "" {
		ghTc = nil
	} else {
		ghTc = oauth2.NewClient(
			oauth2.NoContext,
			oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tok}),
		)
	}

	zensh := &Zensh{
		Config:       &config.GlobalConfig,
		Plugins:      config.Plugins,
		GitClient:    gitclient.NewGitClient(),
		GithubClient: (*github_release.Client)(github.NewClient(ghTc)),
		HomeDir:      user.HomeDir,
	}

	for i := 0; i < len(config.Plugins); i++ {
		config.Plugins[i].parent = zensh
	}

	return zensh, nil
}

func (z *Zensh) NewSemaphore() util.Semaphore {
	return util.NewSemaphore(z.Config.Threads)
}

type RecipeError struct {
	Recipe *Recipe
	Error  error
}

type ActionError []RecipeError

func (ae ActionError) Error() string {
	return fmt.Sprintf("%v errors occured", len(ae))
}

func (z *Zensh) Install() error {
	prog := progress.NewProgress()
	defer prog.Free()
	sem := z.NewSemaphore()
	wait := sync.WaitGroup{}

	errors := make(chan RecipeError, len(z.Plugins))

	for _, recipe := range z.Plugins {
		if recipe.Source == Local {
			continue
		}

		if recipe.IsInstalled() {
			continue
		}

		sem.Acquire()
		wait.Add(1)

		recipe.task = prog.NewTask(recipe.Repo, "")

		go func(recipe *Recipe) {
			defer wait.Done()
			defer sem.Release()

			if err := recipe.Install(); err != nil {
				recipe.task.Done("Error: " + err.Error())
				errors <- RecipeError{Recipe: recipe, Error: err}
			}
			recipe.task.Done("done!")
		}(recipe)
	}

	wait.Wait()

	errSize := len(errors)
	if errSize == 0 {
		return nil
	}

	errList := make([]RecipeError, errSize)

	for i := 0; i < errSize; i++ {
		errList[i] = <-errors
	}

	return ActionError(errList)
}

func (z *Zensh) CheckUpdate() ([]fmt.Stringer, error) {
	prog := progress.NewProgress()
	defer prog.Free()
	sem := z.NewSemaphore()
	wait := sync.WaitGroup{}

	fmt.Println(sem, wait)

	return []fmt.Stringer{}, nil
}
