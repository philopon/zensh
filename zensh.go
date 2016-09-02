package main

import (
	"net/http"
	"os/user"
	"sync"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"

	"./gitclient"
	"./github_release"
	"./progress"
	"./util"
)

type Zensh struct {
	Config       *GlobalConfig
	Plugins      []*Recipe
	GitClient    gitclient.GitClient
	GithubClient *github_release.Client
	HomeDir      string
}

func NewZensh(config *Config) (Zensh, error) {
	user, err := user.Current()
	if err != nil {
		return Zensh{}, err
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

	zensh := Zensh{
		Config:       &config.GlobalConfig,
		Plugins:      config.Plugins,
		GitClient:    gitclient.NewGitClient(),
		GithubClient: (*github_release.Client)(github.NewClient(ghTc)),
		HomeDir:      user.HomeDir,
	}

	for i := 0; i < len(config.Plugins); i++ {
		config.Plugins[i].parent = &zensh
	}

	return zensh, nil
}

func (z *Zensh) NewSemaphore() util.Semaphore {
	return util.NewSemaphore(z.Config.Threads)
}

type Failed struct {
	Recipe  *Recipe
	Occured error
}

func (z *Zensh) Install() []Failed {
	prog := progress.NewProgress()
	defer prog.Free()
	sem := z.NewSemaphore()
	wait := sync.WaitGroup{}

	errors := make(chan Failed, len(z.Plugins))

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
				errors <- Failed{Recipe: recipe, Occured: err}
			}
			recipe.task.Done("done!")
		}(recipe)
	}

	wait.Wait()

	errSize := len(errors)
	errList := make([]Failed, errSize)

	for i := 0; i < errSize; i++ {
		errList[i] = <-errors
	}

	return errList
}
