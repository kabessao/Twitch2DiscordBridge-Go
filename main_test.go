package main

import (
	"testing"
	"twitch2discordbridge/configuration"
	externalemotes "twitch2discordbridge/externalEmotes"
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

func Test7TVgetEmotes(t *testing.T) {

	var config, err = configuration.LoadConfigFromFile("./config.yaml")

	if err != nil {
		t.Error(err)
	}

	tclient, err := helix.NewClient(&helix.Options{
		ClientID:        config.TwitchClientId,
		UserAccessToken: config.OauthPassword,
	})

	if err != nil {
		t.Error(err)
	}

	userInfo, err := tclient.GetUsers(
		&helix.UsersParams{
			Logins: []string{config.Channel},
		},
	)

	if err != nil {
		t.Error(err)
	}

	var (
		users = userInfo.Data.Users
		user  helix.User
	)

	if len(users) != 0 {
		user = users[0]
	}

}
