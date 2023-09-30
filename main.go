package main

import (
	"path/filepath"
	"twitch2discordbridge/bot"
)

func main() {

	files, err := filepath.Glob("*config.yaml")

	if err != nil {
		panic(err)
	}

	for _, file := range files {
		go bot.LaunchNewBot(file)
	}

	for {

	}

}
