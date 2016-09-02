package main

import "fmt"

func main() {
	config, err := LoadConfig("plugins.toml")
	if err != nil {
		panic(err)
	}

	zensh, err := NewZensh(config)
	if err != nil {
		panic(err)
	}

	fmt.Println(zensh.Install())
}
