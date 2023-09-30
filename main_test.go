package main

import (
	"reflect"
	"testing"
	"twitch2discordbridge/configuration"
	"twitch2discordbridge/utils"
)

func TestInitConfigFile(t *testing.T) {

	type test struct {
		configValue   string
		yamlKey       string
		expectedValue interface{}
	}

	var tests = []test{
		{configValue: "WebhookUrl", yamlKey: "webhook_url: 'https://discord.com/api/webhooks/123/456'", expectedValue: "https://discord.com/api/webhooks/123/456"},
		{yamlKey: "twitch_client_id: 'YOUR TWITCH ID'", configValue: "TwitchClientId", expectedValue: "YOUR TWITCH ID"},
		{yamlKey: "twitch_username: 'YOUR TWITCH USERNAME'", expectedValue: "YOUR TWITCH USERNAME", configValue: "TwitchUsername"},
		{yamlKey: "oauth_password: 'YOUR TWITCH OAUTH PASSWORD (NOT YOUR REAL PASSWORD)'", expectedValue: "YOUR TWITCH OAUTH PASSWORD (NOT YOUR REAL PASSWORD)", configValue: "OauthPassword"},
		{yamlKey: "channel: 'THE CHANNEL NAME'", expectedValue: "THE CHANNEL NAME", configValue: "Channel"},
		{yamlKey: "send_all_messages: true", expectedValue: true, configValue: "SendAllMessages"},
		{yamlKey: "prevent_ping: true", expectedValue: true, configValue: "PreventPing"},
		{yamlKey: "show_bit_gifters: true", expectedValue: true, configValue: "ShowBitGifters"},
		{yamlKey: "show_bit_gifters: 500", expectedValue: 500, configValue: "ShowBitGifters"},
		{yamlKey: "log_file: false", expectedValue: false, configValue: "LogFile"},
		{yamlKey: "mod_actions: true", expectedValue: true, configValue: "ModActions"},
		{yamlKey: "filter_badges:\n - broadcaster\n - vip\n - moderator\n", expectedValue: []string{"broadcaster", "vip", "moderator"}, configValue: "FilterBadges"},
		{yamlKey: "filter_usernames:\n - soundalerts", expectedValue: []string{"soundalerts"}, configValue: "FilterUsernames"},
		{yamlKey: "blacklist:\n - mizkif", expectedValue: []string{"mizkif"}, configValue: "Blacklist"},
		{yamlKey: "filter_messages:\n - '[^\\x20-\\x7F]'", expectedValue: []string{"[^\\x20-\\x7F]"}, configValue: "FilterMessages"},
		{yamlKey: "emote_translator:\n  'henyatDance': '<a:HenyaDance:>'", expectedValue: map[string]string{"henyatDance": "<a:HenyaDance:>"}, configValue: "EmoteTranslator"},
		{yamlKey: "emote_translator_url: 'https://example.com'", expectedValue: "https://example.com", configValue: "EmoteTranslatorUrl"},
	}

	for _, test := range tests {

		config, err := configuration.LoadConfigConfigFromBytes([]byte(test.yamlKey))

		if err != nil {
			t.Errorf("Loading config returned error: %v", err)
		}

		reflection := reflect.ValueOf(config)

		fieldValue := reflection.FieldByName(test.configValue)

		if !fieldValue.IsValid() {
			t.Errorf("Your tests sucks, you typed %s", test.configValue)
		}

		//if fieldValue.Interface() != test.expectedValue {
		if !reflect.DeepEqual(fieldValue.Interface(), test.expectedValue) {
			t.Errorf("got [%v], wanted [%v]", fieldValue.Interface(), test.expectedValue)
		}

	}

}

func TestGetDuration(t *testing.T) {
	type Tests struct {
		value  int
		expect string
	}

	var tests = []Tests{
		{12, "12 seconds"},
		{612, "10 minutes and 12 seconds"},
		{30612, "8 hours, 30 minutes and 12 seconds"},
		{3600, "1 hour"},
		{3612, "1 hour and 12 seconds"},
		{3660, "1 hour and 1 minute"},
	}

	for _, test := range tests {
		var value = utils.GetDuration(test.value)

		if value != test.expect {
			t.Errorf("got [%s], expected [%s]", value, test.expect)
		}
	}

}

func TestArrayHasAny(t *testing.T) {
	var test = utils.StringArrayContainAnyInList(
		map[string]int{
			"foo": 1,
			"bar": 1,
			"bas": 1,
		},
		[]string{
			"42",
			"bar",
		},
	)

	if test != true {
		t.Errorf("got %v, expected true", test)
	}

	test = utils.StringArrayContainAnyInList(
		map[string]int{
			"foo": 1,
			"bar": 1,
			"bas": 1,
		},
		[]string{
			"42",
			"somethingElse",
		},
	)

	if test != false {
		t.Errorf("got %v, expected false", test)
	}
}

func TestDoesRegexMatch(t *testing.T) {
	utils.StringContainsRegex("cyberdruga", "cy*.")
}
