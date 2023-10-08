package configuration

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// mandatory
	WebhookUrl     string `yaml:"webhook_url"`
	TwitchClientId string `yaml:"twitch_client_id"`
	TwitchUsername string `yaml:"twitch_username"`
	OauthPassword  string `yaml:"oauth_password"`
	Channel        string `yaml:"channel"`

	// optionals
	SendAllMessages bool        `yaml:"send_all_messages"`
	PreventPing     bool        `yaml:"prevent_ping"`
	ShowBitGifters  interface{} `yaml:"show_bit_gifters"`
	ShowHyperChat   interface{} `yaml:"show_hyber_chat"`
	OutputLog       bool        `yaml:"output_log"`
	ModActions      bool        `yaml:"mod_actions"`
	FilterBadges    []string    `yaml:"filter_badges"`
	FilterUsernames []string    `yaml:"filter_usernames"`
	Blacklist       []string    `yaml:"blacklist"`
	FilterMessages  []string    `yaml:"filter_messages"`

	webhookId    string
	webhookToken string
}

// Loads the configuration based on the bytes given.
//
// Returns err if fails
func LoadConfigConfigFromBytes(value []byte) (Config, error) {
	c := Config{ // default values
		PreventPing:    true,
		ShowBitGifters: false,
		ShowHyperChat:  false,
	}
	err := yaml.Unmarshal(value, &c)

	if err != nil {
		return c, err
	}

	if c.WebhookUrl != "" {
		var split = strings.Split(c.WebhookUrl, "/")

		if len(split) < 2 {
			return c, errors.New(fmt.Sprintf("[%s] is not a valid url", c.WebhookUrl))
		}

		c.webhookToken = split[len(split)-1]
		c.webhookId = split[len(split)-2]
	}

	return c, nil
}

// Loads the configuration directly from a given LoadConfigFromFile
//
// Returns error if fails
func LoadConfigFromFile(fileName string) (Config, error) {

	content, err := os.ReadFile(fileName)
	if err != nil {

		return Config{}, err
	}

	return LoadConfigConfigFromBytes(content)
}
