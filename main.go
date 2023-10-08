package main

import (
	"path/filepath"
	"strings"
	"time"
	"twitch2discordbridge/bot"
	"twitch2discordbridge/emotes"

	"github.com/fsnotify/fsnotify"
)

const (
	FILE_NAME_PATTERN              = "config.yaml"
	DELAY_SECONDS_TO_UPDATE_CONFIG = 2
	EMOTES_FILENAME                = "emotes.csv"
)

var biggestFileNameLength int

func GetSeparatorLengh() {

}

type config struct {
	lastUpdated time.Time
	channel     *bot.Channel
}

func main() {

	files, err := filepath.Glob("*")

	if err != nil {
		panic(err)
	}

	var bots = make(map[string]config)

	for _, file := range files {
		if _, ok := bots[file]; !ok {

			if strings.HasSuffix(file, EMOTES_FILENAME) {
				emotes.UpdateCache(file)
				continue
			}

			if !strings.HasSuffix(file, FILE_NAME_PATTERN) {
				continue
			}

			bots[file] = config{
				lastUpdated: time.Now(),
				channel: &bot.Channel{
					IsOk:    true,
					Channel: make(chan bool),
				},
			}
		}
		go bot.LaunchNewBot("./"+file, bots[file].channel)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		panic(err)
	}

	watcher.Add("./")

	defer watcher.Close()

	for {
		select {
		case event, ok := <-watcher.Events:

			if !ok {
				continue
			}

			if event.Has(fsnotify.Write) && strings.HasSuffix(event.Name, EMOTES_FILENAME) {
				emotes.UpdateCache(event.Name)
				continue
			}

			if !strings.HasSuffix(event.Name, FILE_NAME_PATTERN) {
				continue
			}

			fileName := strings.ReplaceAll(event.Name, "./", "")

			if _, ok := bots[fileName]; !ok {
				bots[fileName] = config{
					lastUpdated: time.Now(),
					channel: &bot.Channel{
						IsOk:    true,
						Channel: make(chan bool),
					},
				}

				go bot.LaunchNewBot("./"+fileName, bots[fileName].channel)

				continue
			}

			if event.Has(fsnotify.Remove) {
				close(bots[fileName].channel.Channel)
				continue
			}

			if !event.Has(fsnotify.Create) && !event.Has(fsnotify.Write) {
				continue
			}

			var instance = bots[fileName]

			if duration := time.Since(instance.lastUpdated); duration < DELAY_SECONDS_TO_UPDATE_CONFIG*time.Second {
				continue
			}

			if !instance.channel.IsOk {
				println("trying to do it again \n")
				instance.channel.IsOk = true
				go bot.LaunchNewBot("./"+fileName, instance.channel)
				continue
			}

			instance.channel.Channel <- true
			instance.lastUpdated = time.Now()

			bots[fileName] = instance
		case err, ok := <-watcher.Errors:
			if !ok {
				panic(err)
			}
		}
	}

}
