package main

import (
	"os"
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/fatih/color"
	ansi "github.com/k0kubun/go-ansi"

	"github.com/philopon/zensh/util"
)

func (z *Zensh) InfoCommand() int {
	maxLen := 1
	for _, recipe := range z.Plugins {
		if l := len(recipe.Repo); l > maxLen {
			maxLen = l
		}
	}

	green := color.New(color.FgGreen)
	red := color.New(color.FgRed)

	for _, recipe := range z.Plugins {
		ansi.Printf("%-"+strconv.Itoa(maxLen)+"v  ", recipe.Repo)
		green.Printf("%6v ", recipe.Source.String())

		hash, err := recipe.GetHash()
		if err != nil {
			red.Println("[Error]", err)
		} else if recipe.Source == Local {
			ansi.Println("ok")
		} else if oid, ok := hash.AsOid(); ok {
			ansi.Println(oid.String()[:7])
		} else {
			ansi.Println(hash)
		}
	}

	return 0
}

func (z *Zensh) InstallCommand(ask bool) int {
	err := z.Install()
	if err == nil {
		return 0
	}

	if ae, ok := err.(ActionError); ok {
		logrus.Error(ae)

		for _, re := range ([]RecipeError)(ae) {
			_, err = os.Stat(re.Recipe.Directory())
			if err == nil {
				os.RemoveAll(re.Recipe.Directory())
			}
		}

		if !ask {
			return 1
		}

		ans, err := util.Ask("retry?[y/N]: ", "retry?[y/N]: ", true,
			func(ans string) bool { a := strings.ToLower(ans); return a == "y" || a == "n" },
		)

		if err != nil {
			logrus.Error(err)
			return 1
		}

		la := strings.ToLower(ans)

		if la == "y" {
			return z.InstallCommand(ask)
		} else {
			return 1
		}

	} else {
		logrus.Error(err)
		return 1
	}
}

func main() {
	color.Output = ansi.NewAnsiStdout()

	config, err := LoadConfig("plugins.toml")
	if err != nil {
		panic(err)
	}

	zensh, err := NewZensh(config)
	if err != nil {
		panic(err)
	}

	os.Exit(zensh.InstallCommand(false))
}
