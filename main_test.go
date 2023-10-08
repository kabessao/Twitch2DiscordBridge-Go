package main

import (
	"testing"
	"twitch2discordbridge/configuration"
	"twitch2discordbridge/emotes"
	"twitch2discordbridge/utils"

	twitchIrc "github.com/gempir/go-twitch-irc/v4"
	"github.com/nicklaw5/helix"
)

func TestCheerMessage(t *testing.T) {

	config, _ := configuration.LoadConfigFromFile("config.yaml")

	helixApi, _ := helix.NewClient(&helix.Options{
		ClientID:        config.TwitchClientId,
		UserAccessToken: config.OauthPassword,
	})

	message := twitchIrc.PrivateMessage{
		User: twitchIrc.User{
			DisplayName: "test",
		},
		Message: "this is a cheer message Cheer100",

		Tags: map[string]string{
			"bits": "100",
		},
	}

	config.ShowBitGifters = true

	if !utils.ParseCheerMessages(&message, helixApi, config) {
		t.Error("should have been true")
	}

	if "test [bits: 100]" != message.User.DisplayName {
		t.Errorf("wrong username, got [%s]", message.User.DisplayName)
	}

	if "this is a cheer message" != message.Message {
		t.Errorf("wrong message, got [%s]", message.Message)
	}

}

func TestHypeMessage(t *testing.T) {

	config, _ := configuration.LoadConfigFromFile("config.yaml")

	message := twitchIrc.PrivateMessage{
		User: twitchIrc.User{
			DisplayName: "test",
		},
		Message: "this is a cheer message Cheer100",

		Tags: map[string]string{
			"pinned-chat-paid-amount": "100",
		},
	}

	config.ShowHyperChat = true

	if !utils.ParseHypeChat(&message, config) {
		t.Error("should have been true")
	}

	if "test [HypeChat: 100]" != message.User.DisplayName {
		t.Errorf("wrong username, got [%s]", message.User.DisplayName)
	}
}

func TestEmotesFile(t *testing.T) {
	emotes.ReadFromFille("./emotes.csv")
}
