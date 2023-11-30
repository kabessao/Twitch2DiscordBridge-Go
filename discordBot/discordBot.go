package discordbot

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/rest"
	"github.com/disgoorg/snowflake/v2"
	"gopkg.in/yaml.v3"
)

var botClient bot.Client

var session rest.Emojis

var emoteManipulation sync.Mutex

var config = map[string]string{}

type allowedFormats map[string]discord.IconType

func (t allowedFormats) Contains(key string) bool {
	if _, ok := t[key]; ok {
		return true
	}
	return false
}

var AllowedFormats = allowedFormats{
	"png":  discord.IconTypePNG,
	"gif":  discord.IconTypeGIF,
	"jpeg": discord.IconTypeJPEG,
	"webp": discord.IconTypeWEBP,
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

	botClient, err = disgo.New(config["token"])

	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't load configuration. Error: %v'\n\n", err)
		return
	}

	botClient.OpenGateway(context.TODO())

	defer botClient.Close(context.TODO())

	session = botClient.Rest()

	println("Discord config loaded\n")

	cleanUp()

	return
}

func CreateEmote(name string, image []byte, format string) (emoji *discord.Emoji, err error) {

	emoteManipulation.Lock()

	defer emoteManipulation.Unlock()

	if session == nil {
		err = fmt.Errorf("No discord bot present, skipping")
		return
	}

	emoji, err = session.CreateEmoji(
		snowflake.MustParse(config["server_id"]),
		discord.EmojiCreate{
			Name: name,
			Image: *discord.NewIconRaw(
				AllowedFormats[format],
				image,
			),
		},
	)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't create emote. Error: %s'", err)
		return
	}

	return
}

func DeleteEmote(id string) error {

	emoteManipulation.Lock()

	defer emoteManipulation.Unlock()

	return session.DeleteEmoji(
		snowflake.MustParse(config["server_id"]),
		snowflake.MustParse(id),
	)
}

func cleanUp() {
	emojis, err := session.GetEmojis(snowflake.MustParse(config["server_id"]))
	if err != nil {
		return
	}

	for _, item := range emojis {

		if item.Creator == nil || item.Creator.ID != botClient.ApplicationID() {
			continue
		}

		println("Cleanup: deleting emote " + item.Name)
		session.DeleteEmoji(
			snowflake.MustParse(config["server_id"]),
			item.ID,
		)
	}
}
