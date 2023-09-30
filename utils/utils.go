package utils

import (
	"fmt"
	"strings"
	"time"

	"bytes"
	"encoding/json"
	"net/http"

	"github.com/dlclark/regexp2"
	twitchIrc "github.com/gempir/go-twitch-irc/v4"
	"twitch2discordbridge/configuration"
	"twitch2discordbridge/twitchApi"
)

// Get duration as string based on the seconds in int
//
// Ex:
//
//	600 > '10 minutes'
//	610 > '10 minutes and 10 seconds'
//	30612 > '8 hours, 30 minutes and 12 seconds''
func GetDuration(seconds int) string {

	var duration = time.Second * time.Duration(seconds)

	var hours = int(duration.Hours())
	var minutes = int(duration.Minutes()) % 60
	var remainingSeconds = int(duration.Seconds()) % 60

	var parts []string

	if minutes != 0 {
		parts = append(parts, fmt.Sprintf("%d minute%s", minutes, pluralParser(minutes)))
	}

	if remainingSeconds != 0 {
		parts = append(parts, fmt.Sprintf("%d second%s", remainingSeconds, pluralParser(remainingSeconds)))
	}

	var separator = " and "
	if len(parts) == 2 {
		separator = ", "
	}

	if len(parts) > 0 {
		parts = []string{
			strings.Join(parts, " and "),
		}
	}

	if hours != 0 {
		parts = append(parts, fmt.Sprintf("%d hour%s", hours, pluralParser(hours)))
	}

	return strings.Join(reverseStringArray(parts), separator)
}

func pluralParser(value int) string {
	if value == 1 {
		return ""
	}
	return "s"
}

func reverseStringArray(array []string) []string {
	for i := 0; i < len(array)/2; i++ {
		j := len(array) - i - 1
		array[i], array[j] = array[j], array[i]
	}
	return array
}

func SendWebhookMessage(content string, userInfo twitchApi.TwitchUserInfo, config configuration.Config) {
	// Create a payload map
	payload := map[string]string{
		"content":    content,
		"username":   userInfo.DisplayName,
		"avatar_url": userInfo.ProfileImageUrl,
	}

	// Serialize the payload to JSON
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return
	}

	// Set up HTTP headers
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	// Create an HTTP request
	req, err := http.NewRequest("POST", config.WebhookUrl, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return
	}

	// Set HTTP headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Create an HTTP client
	client := &http.Client{}

	// Send the HTTP request
	_, err = client.Do(req)
	if err != nil {
		return
	}

	return
}

func EmoteParser(message *twitchIrc.PrivateMessage, config configuration.Config) {
	messageEmotes := message.Emotes

	availableEmotes := config.EmoteTranslator

	for _, emote := range messageEmotes {
		if newEmote, ok := availableEmotes[emote.Name]; ok {
			var re = regexp2.MustCompile(fmt.Sprintf("(?<=^|\\W)%s(?=\\W|$)", emote.Name), regexp2.None)
			text, err := re.Replace(message.Message, newEmote, -1, -1)
			if err != nil {
				panic(nil)
			}

			message.Message = text
		}
	}
}

func StringArrayContains(array []string, value string) bool {
	for _, item := range array {
		if item == value {
			return true
		}
	}
	return false
}

func StringArrayContainAnyInList(from map[string]int, to []string) bool {
	for _, item := range to {
		if _, ok := from[item]; ok {
			return true
		}
	}
	return false
}

func StringContainsRegex(value string, regex string) bool {
	var re = regexp2.MustCompile(regex, regexp2.None)
	var match, _ = re.MatchString(value)
	return match
}

func StringContainsAnyRegex(value string, regexArray []string) bool {
	for _, regex := range regexArray {
		if StringContainsRegex(value, regex) {
			return true
		}
	}
	return false
}

func HasCheerMessage() {

}
