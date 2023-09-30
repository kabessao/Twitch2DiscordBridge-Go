package bot

import (
	"fmt"

	"twitch2discordbridge/configuration"
	"twitch2discordbridge/twitchApi"
	"twitch2discordbridge/utils"

	twitchIrc "github.com/gempir/go-twitch-irc/v4"
)

const (
	MESSAGE_HISTORY_LENGTH int = 800
)

func LaunchNewBot(filePath string) {
	var messageHistory []twitchIrc.PrivateMessage

	var config, err = configuration.LoadConfigFromFile(filePath)
	if err != nil {
		panic(err)
	}

	client := twitchIrc.NewClient(config.TwitchUsername, "oauth:"+config.OauthPassword)

	var api = twitchApi.LoadFromConfig(config)

	client.OnConnect(func() {
		fmt.Println("connected to " + config.Channel)
	})

	client.OnPrivateMessage(func(message twitchIrc.PrivateMessage) {

		go fmt.Print(message.Raw + "\n\n")

		messageHistory = append(messageHistory, message)

		if length := len(messageHistory); length > MESSAGE_HISTORY_LENGTH {
			messageHistory = messageHistory[length-MESSAGE_HISTORY_LENGTH : length]
		}

		var userInfo, err = api.GetProfileInfo(message.User.ID)
		if err != nil {
			return
		}

		userInfo.DisplayName = fmt.Sprintf("%s [%s chat]", userInfo.DisplayName, pluralParser(config.Channel))

		utils.EmoteParser(&message, config)

		utils.SendWebhookMessage(
			message.Message,
			userInfo,
			config,
		)
	})

	client.Join(config.Channel)

	err = client.Connect()
	if err != nil {
		panic(err)
	}

}

func pluralParser(value string) string {
	if value[len(value)-1] == 's' {
		return value + "'"
	}
	return value + "'s"
}
