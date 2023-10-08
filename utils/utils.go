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
	"twitch2discordbridge/emotes"

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

type EmbedAuthor struct {
	Name    string `json:"name"`
	IconUrl string `json:"icon_url"`
}

type WebhookEmbed struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Author      EmbedAuthor `json:"author"`
}

type WebhookMessage struct {
	Content       string         `json:"content"`
	Username      string         `json:"username"`
	AvatarUrl     string         `json:"avatar_url"`
	AllowMentions bool           `json:"allowed_mentions"`
	Embeds        []WebhookEmbed `json:"embeds"`
}

func SendWebhookMessage(url string, message WebhookMessage) {
	// Serialize the payload to JSON
	payloadBytes, err := json.Marshal(message)
	if err != nil {
		return
	}

	// Set up HTTP headers
	headers := map[string]string{
		"Content-Type": "application/json",
	}

	// Create an HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
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

	availableEmotes := emotes.GetEmotes()

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

func StringMapContainsAnyInList(from map[string]int, to []string) bool {
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

	if bits == 0 {
		return false
	}

	if val, ok := config.ShowBitGifters.(bool); ok && !val {
		return false
	}

	if val, ok := config.ShowBitGifters.(int); ok && bits < val {
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

	message.User.DisplayName = fmt.Sprintf("%s [bits: %d]", message.User.DisplayName, bits)

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
	return strings.Trim(text, " ")
}

func ParseHypeChat(message *twitchIrc.PrivateMessage, config configuration.Config) bool {

	if value, ok := config.ShowHyperChat.(bool); ok && !value {
		return false
	}

	var bits int
	if value, ok := message.Tags["pinned-chat-paid-amount"]; ok {
		bits, _ = strconv.Atoi(value)
	}

	if bits == 0 {
		return false
	}

	if value, ok := config.ShowHyperChat.(int); ok && bits < value {
		return false
	}

	message.User.DisplayName = fmt.Sprintf("%s [HypeChat: %d]", message.User.DisplayName, bits)
	return true
}

// uses 's when the string doesn't end with an "s"
//
// uses just a ' if it does
func PluralSufixParser(value string) string {
	if value == "" {
		return value
	}

	if value[len(value)-1] == 's' {
		return value + "'"
	}
	return value + "'s"
}
