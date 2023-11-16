package utils

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"twitch2discordbridge/configuration"
	"twitch2discordbridge/emotes"
	externalemotes "twitch2discordbridge/externalEmotes"

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

func ParseExternalEmotes(channelId string, message *twitchIrc.PrivateMessage, config configuration.Config) map[string]externalemotes.ExternalEmote {
	return ParseExternalEmotesFromMap(channelId, message, config, emotes.GetEmotes())
}

func ParseExternalEmotesFromMap(channelId string, message *twitchIrc.PrivateMessage, config configuration.Config, availableEmotes map[string]string) (unavailableEmotes map[string]externalemotes.ExternalEmote) {
	emotes := externalemotes.GetAll(channelId, message.User.ID)

	unavailableEmotes = make(map[string]externalemotes.ExternalEmote)

	for name, emote := range emotes {

		if StringContainsRegex(name, "\\W") {
			continue
		}

		var (
			newEmote string
			ok       bool
		)
		if newEmote, ok = availableEmotes[name]; !ok {
			unavailableEmotes[name] = emote
			continue
		}

		emoteReplace := fmt.Sprintf("(?<=^|\\W)(?<!\\<\\:)%s(?!\\:\\d+\\>)(?=\\W|$)", name)

		message.Message = StringReplaceAllRegex(emoteReplace, message.Message, newEmote)
	}

	return
}

func ParseTwitchEmotes(message *twitchIrc.PrivateMessage, config configuration.Config) map[string]string {
	return ParseTwitchEmotesFromMap(message, config, emotes.GetEmotes())
}

func ParseTwitchEmotesFromMap(message *twitchIrc.PrivateMessage, config configuration.Config, availableEmotes map[string]string) (unavailableEmotes map[string]string) {
	emotes := message.Emotes

	unavailableEmotes = make(map[string]string)

	for _, emote := range emotes {

		if StringContainsRegex(emote.Name, "\\W") {
			continue
		}

		var (
			newEmote string
			ok       bool
		)
		if newEmote, ok = availableEmotes[emote.Name]; !ok {
			unavailableEmotes[emote.Name] = emote.ID
			continue
		}

		emoteReplace := fmt.Sprintf("(?<=^|\\W)(?<!\\<\\:)%s(?!\\:\\d+\\>)(?=\\W|$)", emote.Name)

		message.Message = StringReplaceAllRegex(emoteReplace, message.Message, newEmote)
	}

	return
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
