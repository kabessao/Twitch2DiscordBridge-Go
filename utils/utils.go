package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"bytes"
	"encoding/json"
	"net/http"

	"twitch2discordbridge/configuration"

	"github.com/dlclark/regexp2"
	twitchIrc "github.com/gempir/go-twitch-irc/v4"
	"github.com/nicklaw5/helix"
)

// Get duration as string based on the seconds in int
//
// Ex:
//
//	600 > '10 minutes'
//	610 > '10 minutes and 10 seconds'
//	30612 > '8 hours, 30 minutes and 12 seconds''
func ParseIntDuration(seconds int) string {

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

func SendWebhookMessage(message twitchIrc.PrivateMessage, userInfo helix.UsersResponse, config configuration.Config) {

	var imageUrl string

	if users := userInfo.Data.Users; len(users) > 0 {
		imageUrl = users[0].ProfileImageURL
	}

	// Create a payload map
	payload := map[string]string{
		"content":    message.Message,
		"username":   message.User.DisplayName,
		"avatar_url": imageUrl,
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
			message.Message = StringReplaceAllRegex(fmt.Sprintf("(?<=^|\\W)%s(?=\\W|$)", emote.Name), message.Message, newEmote)
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

func ParseCheerMessages(message *twitchIrc.PrivateMessage, helixApi *helix.Client, config configuration.Config) bool {
	var bits int
	if strBits, ok := message.Tags["bits"]; ok {
		bits, _ = strconv.Atoi(strBits)
	}

	if bits > 0 && bits >= config.ShowBitGifters {
		return false
	}

	var cheerMap = map[string]any{}
	if cheermotes, err := helixApi.GetCheermotes(&helix.CheermotesParams{}); err == nil {
		for _, emote := range cheermotes.Data.Cheermotes {
			cheerMap[emote.Prefix] = emote.Prefix
		}
	}

	matches := FindAllMatches("(?<=^|\\W)[a-zA-Z]+(?=\\d+(\\W|$))", message.Message)

	for _, item := range matches {
		if _, ok := cheerMap[item]; ok {
			message.Message = StringReplaceAllRegex(item+"\\d+", message.Message, "")
		}
	}

	if strings.TrimSpace(message.Message) == "" {
		message.Message = "`Empty message`"
	}

	return true
}

func FindAllMatches(regex string, from string) (matches []string) {
	re, _ := regexp2.Compile(regex, regexp2.None)

	m, _ := re.FindStringMatch(from)

	for m != nil {
		matches = append(matches, m.String())
		m, _ = re.FindNextMatch(m)
	}

	return matches
}

func StringReplaceAllRegex(regex string, in string, to string) string {

	var re = regexp2.MustCompile(regex, regexp2.None)
	text, _ := re.Replace(in, to, -1, -1)
	return text
}

func ParseHypeChat(message *twitchIrc.PrivateMessage) bool {
	if value, ok := message.Tags["pinned-chat-paid-amount"]; ok {
		message.User.DisplayName = fmt.Sprintf("%s [HypeChat %s]", message.User.DisplayName, value)
		return true
	}
	return false
}

func PluralParser(value string) string {
	if value == "" {
		return value
	}

	if value[len(value)-1] == 's' {
		return value + "'"
	}
	return value + "'s"
}
