package discordbot

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"gopkg.in/yaml.v3"
)

var session *discordgo.Session

var config = map[string]string{}

func CloseDiscordBot() {
	session.Close()
}

func LoadConfigFromFile(fileName string) (err error) {

	content, err := os.ReadFile(fileName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't read from file. Error: %v'\n\n", err)
		return
	}

	err = yaml.Unmarshal(content, &config)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't load configuration. Error: %v'\n\n", err)
		return
	}

	session, err = discordgo.New("Bot " + config["token"])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't start the discord bot. Error: %v'\n\n", err)
		return
	}

	session.Identify.Intents = discordgo.IntentsAllWithoutPrivileged

	err = session.Open()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't connect to Discord Bot. Error: %v'\n\n", err)
		panic(err)
	}

	println("Discord config loaded\n")

	return
}

func CreateEmote(name string, base64Image string, format string) (emoji *discordgo.Emoji, err error) {

	if session == nil {
		err = fmt.Errorf("No discord bot present, skipping")
		return
	}

	emoji, err = session.GuildEmojiCreate(
		config["server_id"],
		&discordgo.EmojiParams{
			Name:  name,
			Image: fmt.Sprintf("data:image/%s;base64,%s", format, base64Image),
		},
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't create emote. Error: %s'", err)
		return
	}

	return
}

func DeleteEmote(id string) {
	session.GuildEmojiDelete(config["server_id"], id)
}
