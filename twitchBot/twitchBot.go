package twitchBot

import (
	"fmt"
	"os"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"twitch2discordbridge/configuration"
	"twitch2discordbridge/emotes"
	"twitch2discordbridge/utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/webhook"
	"github.com/dlclark/regexp2"
	"github.com/gempir/go-twitch-irc/v4"
	"github.com/nicklaw5/helix"
)

const (
	MESSAGE_HISTORY_LENGTH int = 800
)

type bot struct {
	fileName        string
	config          configuration.Config
	helixApi        *helix.Client
	client          *twitch.Client
	webhookClient   webhook.Client
	sendMessageLock sync.Mutex
	firstMessages   map[string]int
	overHeatAmount  int
	isOverHeated    bool
	messageHistory  map[string][]twitch.PrivateMessage
	channel         Channel
}

func (b *bot) log(message string) {
	log(b.fileName, message)
}

func log(fileName string, message string) {
	fmt.Printf("%-30s| %s\n\n", fileName, message)
}

func (b *bot) recover() {
	if err := recover(); err != nil {
		fmt.Fprintf(os.Stderr, "%-30s| Error: %s\n\n", b.fileName, err)
		print("test")
		debug.PrintStack()
	}
}

func (b *bot) errorLog(err error) {
	errorLog(b.fileName, err)
}

func errorLog(fileName string, err interface{}) {
	fmt.Fprintf(os.Stderr, "%-30s| Error: %s\n\n", fileName, err)
}

func (b *bot) onConnect() {
	b.log(fmt.Sprintf("connected to [%s]", strings.Join(b.config.Channels, ",")))
}

func (b *bot) onPrivateMessage(message twitch.PrivateMessage) {

	if b.config.OutputLog {
		go b.log(fmt.Sprintf(message.Raw))
	}

	b.messageHistory[message.Channel] = append(b.messageHistory[message.Channel], message)

	if length := len(b.messageHistory[message.Channel]); length > MESSAGE_HISTORY_LENGTH {
		b.messageHistory[message.Channel] = b.messageHistory[message.Channel][length-MESSAGE_HISTORY_LENGTH : length]
	}

	if utils.StringContainsRegex(message.User.DisplayName, "[^\\x20-\\x7F]") {
		message.User.DisplayName = fmt.Sprintf("%s (%s)", message.User.DisplayName, message.User.Name)
	}

	if utils.StringArrayContains(b.config.Blacklist, message.User.Name) {
		return
	}

	if !b.onStreamStatus(message.Channel) {
		return
	}

	var shouldSend = b.config.SendAllMessages

	if utils.StringMapContainsAnyInList(message.User.Badges, b.config.FilterBadges) {
		shouldSend = true
	}

	if utils.StringArrayContains(b.config.FilterUsernames, message.User.Name) {
		shouldSend = true
	}

	if utils.StringContainsAnyRegex(message.Message, b.config.FilterMessages) {
		shouldSend = true
	}

	if utils.ParseCheerMessages(&message, b.helixApi, b.config) {
		shouldSend = true
	}

	if b.shouldSendFirstMessages(&message) {
		shouldSend = true
	}

	if b.config.PreventPing {
		message.Message = utils.StringReplaceAllRegex("@(?=here|everyone)", message.Message, "")
	}

	if shouldSend {
		go b.sendMessage(
			message,
		)
	}

}

func (b *bot) shouldSendFirstMessages(message *twitch.PrivateMessage) bool {
	if b.config.ModTools.LogFirstMessages != nil {
		var amountAllowed = *b.config.ModTools.LogFirstMessages

		if message.FirstMessage {
			message.Message = "`First Message`: " + message.Message
			b.firstMessages[message.User.ID] = 1
			return true
		}

		if amountSent, ok := b.firstMessages[message.User.ID]; ok && amountSent < amountAllowed {
			b.firstMessages[message.User.ID] += 1
			return true
		}

	}

	if _, ok := b.firstMessages[message.User.ID]; ok {
		delete(b.firstMessages, message.User.ID)
	}

	return false
}

func (b *bot) onClearChatMessage(message twitch.ClearChatMessage) {
	if !b.config.ModActions {
		return
	}

	if b.config.OutputLog {
		go b.log(fmt.Sprintf(message.Raw))
	}

	var timeoutMessage = "User was banned permanently"

	if duration := message.BanDuration; duration > 0 {
		timeoutMessage = fmt.Sprintf("User was timed out for %s", utils.ParseIntDuration(duration))
	}

	usersInfo, err := b.helixApi.GetUsers(&helix.UsersParams{
		IDs: []string{message.TargetUserID},
	})

	if err != nil || len(usersInfo.Data.Users) == 0 {
		return
	}

	var userInfo = usersInfo.Data.Users[0]

	var messages []string

	for _, m := range b.messageHistory[message.Channel] {
		if m.User.ID == message.TargetUserID {

			if _, ok := m.Tags["__deleted"]; ok {
				continue
			}

			messages = append(messages, "- "+m.Message)

			m.Tags["__deleted"] = "true"
		}
	}

	var webhookMessage = discord.WebhookMessageCreate{
		AvatarURL: b.getTwitchAvatarUrl(message.Channel),
	}

	usercardURL := fmt.Sprintf("https://www.twitch.tv/popout/%s/viewercard/%s", message.Channel, message.TargetUsername)

	webhookMessage.Username = fmt.Sprintf("%s chat mod action", utils.PluralSufixParser(message.Channel))
	webhookMessage.Embeds = []discord.Embed{
		{
			Author: &discord.EmbedAuthor{
				Name:    fmt.Sprintf("%s (%s)", message.TargetUsername, message.TargetUserID),
				IconURL: userInfo.ProfileImageURL,
				URL:     usercardURL,
			},
			Title:       timeoutMessage,
			Description: strings.Join(messages, "\n"),
		},
	}

	_, err = b.webhookClient.CreateMessage(webhookMessage)
	if err != nil {
		b.errorLog(fmt.Errorf("Couldn't send webhook message. Error: %s", err))
		return
	}

}

func (b *bot) onClearMessage(deletedMessage twitch.ClearMessage) {
	if !b.config.ModActions {
		return
	}

	if b.config.OutputLog {
		go b.log(fmt.Sprintf(deletedMessage.Raw))
	}

	for _, m := range b.messageHistory[deletedMessage.Channel] {
		if m.ID == deletedMessage.TargetMsgID {

			if _, ok := m.Tags["__deleted"]; ok {
				continue
			}

			var webhookMessage = discord.WebhookMessageCreate{
				AvatarURL: b.getTwitchAvatarUrl(m.Channel),
			}

			userProfilePicture := b.getTwitchAvatarUrl(m.User.Name)

			usercardURL := fmt.Sprintf("https://www.twitch.tv/popout/%s/viewercard/%s", m.Channel, m.User.Name)

			webhookMessage.Username = fmt.Sprintf("%s chat mod action", utils.PluralSufixParser(m.Channel))
			webhookMessage.Embeds = []discord.Embed{
				{
					Author: &discord.EmbedAuthor{
						Name:    fmt.Sprintf("%s (%s)", m.User.Name, m.User.ID),
						IconURL: userProfilePicture,
						URL:     usercardURL,
					},
					Title:       "Message Deleted",
					Description: "- " + m.Message,
				},
			}

			_, err := b.webhookClient.CreateMessage(webhookMessage)
			if err != nil {
				b.errorLog(fmt.Errorf("Couldn't send webhook message. Error: %s", err))
				return
			}

			m.Tags["__deleted"] = "true"
		}
	}
}

func (b *bot) onUserNoticeMessage(message twitch.UserNoticeMessage) {

	if b.config.OutputLog {
		go b.log(fmt.Sprintf(message.Raw))
	}

	re := regexp2.MustCompile(`\s#(\w+)`, regexp2.None)
	match, err := re.FindStringMatch(message.Raw)
	if err != nil {
		b.errorLog(err)
		return
	}

	channel := match.GroupByNumber(1).String()

	msgId := message.Tags["msg-id"]

	switch msgId {

	case "raid":
		b.sendRaidMessage(message, channel)

	case "announcement":
		b.sendAnnouncementMessage(message, channel)

	default:
		return

	}

}
func (b *bot) sendAnnouncementMessage(message twitch.UserNoticeMessage, channel string) {

	if !b.config.ShowAnnouncementMessages {
		return
	}

	var webhookMessage discord.WebhookMessageCreate

	webhookMessage.Username = fmt.Sprintf(
		"%s announcements",
		utils.PluralSufixParser(channel),
	)

	webhookMessage.AvatarURL = b.getTwitchAvatarUrl(channel)

	webhookMessage.Embeds = []discord.Embed{
		{
			Author: &discord.EmbedAuthor{
				Name:    message.User.DisplayName,
				IconURL: b.getTwitchAvatarUrl(message.User.Name),
			},
			Description: message.Message,
		},
	}

	_, err := b.webhookClient.CreateMessage(webhookMessage)
	if err != nil {
		b.errorLog(err)
	}

}

func (b *bot) sendRaidMessage(message twitch.UserNoticeMessage, channel string) {

	if value, ok := b.config.ShowRaidMessages.(bool); ok && !value {
		return
	}

	var raidersAmmount int

	if value, ok := message.Tags["msg-param-viewerCount"]; ok {
		raidersAmmount, _ = strconv.Atoi(value)

		if value, ok := b.config.ShowRaidMessages.(int); ok && raidersAmmount < value {
			return
		}
	}

	var webhookMessage discord.WebhookMessageCreate

	webhookMessage.Username = fmt.Sprintf(
		"%s raid message",
		utils.PluralSufixParser(channel),
	)

	webhookMessage.AvatarURL = b.getTwitchAvatarUrl(channel)

	webhookMessage.Embeds = []discord.Embed{
		{
			Author: &discord.EmbedAuthor{
				Name:    message.User.DisplayName,
				IconURL: b.getTwitchAvatarUrl(message.User.Name),
				URL:     fmt.Sprintf("https://www.twitch.tv/%s", message.User.Name),
			},
			Title: fmt.Sprintf("`%d raiders just arrived`", raidersAmmount),
		},
	}

	_, err := b.webhookClient.CreateMessage(webhookMessage)

	if err != nil {
		b.errorLog(err)
	}
}

func (b *bot) loadConfiguration() error {
	client := twitch.NewClient(b.config.TwitchUsername, "oauth:"+b.config.OauthPassword)

	helixApi, err := helix.NewClient(&helix.Options{
		ClientID:        b.config.TwitchClientId,
		UserAccessToken: b.config.OauthPassword,
	})

	if err != nil {
		return err
	}

	b.helixApi = helixApi
	b.client = client

	b.webhookClient, err = webhook.NewWithURL(b.config.WebhookUrl)
	if err != nil {
		return fmt.Errorf("Couldn't start webhook client: %v", err)
	}

	var pong = false

	client.OnPingSent(func() {
		go func() {
			time.Sleep(3 * time.Second)
			if !pong && b.channel.IsOk {
				b.errorLog(fmt.Errorf("Connection timeout reached!"))
				close(b.channel.Channel)
				return
			}
			pong = false
		}()

	})

	client.OnPongMessage(func(message twitch.PongMessage) {
		pong = true
	})

	client.OnConnect(b.onConnect)

	client.OnPrivateMessage(b.onPrivateMessage)

	client.OnClearChatMessage(b.onClearChatMessage)

	client.OnClearMessage(b.onClearMessage)

	client.OnUserNoticeMessage(b.onUserNoticeMessage)

	return nil
}

type Channel struct {
	IsOk    bool
	Channel chan bool
}

func LaunchNewBot(filePath string, channel *Channel) {
	var bot = bot{
		fileName:        filePath,
		sendMessageLock: sync.Mutex{},
		firstMessages:   map[string]int{},
		messageHistory:  map[string][]twitch.PrivateMessage{},
		channel:         *channel,
	}

	defer recover()

	defer func() {
		channel.IsOk = false
		if err := recover(); err != nil {
			errorLog(filePath, err)
			debug.PrintStack()
		}
	}()

	var config, err = configuration.LoadConfigFromFile(filePath)
	if err != nil {
		panic(fmt.Sprintf("Couldn't load configuration file': %v\n\n", err))
	}

	bot.config = config

	bot.log("Starting from config file")

	if err := bot.loadConfiguration(); err != nil {
		bot.errorLog(err)
		return
	}

	if err := bot.startClient(); err != nil {
		bot.errorLog(err)
		return
	}

	defer bot.log("This instance is shuting down")

	for {
		select {
		case _, ok := <-channel.Channel:
			if !ok {
				bot.client.Disconnect()
				return
			}

			bot.log("Reloading config file")

			config, err = configuration.LoadConfigFromFile(filePath)
			if err != nil {
				bot.client.Disconnect()
				return
			}

			bot.config = config

			bot.client.Disconnect()

			if err := bot.loadConfiguration(); err != nil {
				bot.log(fmt.Sprintf("Error: %v", err))
				return
			}

			bot.startClient()
		}
	}

}

func (b *bot) startClient() (err error) {

	go func() {
		err = b.client.Connect()
	}()

	time.Sleep(5 * time.Second)

	b.client.Join(b.config.Channels...)

	return err
}

func parseBadges(message *twitch.PrivateMessage, badges ...string) {
	for _, badge := range badges {
		if _, ok := message.User.Badges[badge]; ok {
			message.User.DisplayName = fmt.Sprintf("%s [%s]", message.User.DisplayName, badge)
		}
	}
}

func (b *bot) getTwitchUserId(name string) string {

	if userInfo, err := b.helixApi.GetUsers(&helix.UsersParams{Logins: []string{name}}); err == nil && len(userInfo.Data.Users) > 0 {
		return userInfo.Data.Users[0].ID
	}

	return ""
}
func (b *bot) getTwitchAvatarUrl(name string) string {

	if userInfo, err := b.helixApi.GetUsers(&helix.UsersParams{Logins: []string{name}}); err == nil && len(userInfo.Data.Users) > 0 {
		return userInfo.Data.Users[0].ProfileImageURL
	}

	return ""
}

func (b *bot) onStreamStatus(channel string) bool {

	if !utils.StringArrayContains([]string{"online", "offline"}, b.config.OnStreamStatus) {
		return true
	}

	userId := b.getTwitchUserId(channel)

	response, err := b.helixApi.GetStreams(&helix.StreamsParams{
		UserIDs: []string{userId},
	})

	if err != nil {
		b.errorLog(err)
		return false
	}

	isStreaming := len(response.Data.Streams) > 0

	if b.config.OnStreamStatus == "online" && isStreaming {
		return true
	}

	if b.config.OnStreamStatus == "offline" && !isStreaming {
		return true
	}

	return false
}

func (b *bot) sendMessage(message twitch.PrivateMessage) {

	if b.overHeatAmount >= 50 {
		b.log("Bot is \"overheating\", likely because it's being rate limited by discord. The bot will now pause to cool off")
		b.isOverHeated = true
	}

	if b.overHeatAmount <= 5 && b.isOverHeated {
		b.isOverHeated = false
	}

	if b.isOverHeated {
		return
	}

	b.overHeatAmount += 1

	defer func() {
		b.overHeatAmount -= 1
	}()

	b.sendMessageLock.Lock()
	defer b.sendMessageLock.Unlock()

	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintf(os.Stderr, "Code panic. Error: %s", err)
			debug.PrintStack()
		}
	}()

	parseBadges(&message, "broadcaster", "moderator", "vip")

	var messageEmbeds []discord.Embed

	if message.Reply != nil {

		var parseMessage = func(msg string) string {
			return strings.Replace(msg, "@"+message.Reply.ParentDisplayName, "", 1)
		}

		message.Message = parseMessage(message.Message)

		var thread []string

		for _, msg := range b.messageHistory[message.Channel] {
			if msg.Reply == nil {
				continue
			}

			if msg.ID == message.ID {
				continue
			}

			if msg.Tags["reply-thread-parent-msg-id"] == message.Tags["reply-thread-parent-msg-id"] {
				thread = append(thread, fmt.Sprintf("`%s`: %s", msg.User.DisplayName, parseMessage(msg.Message)))
			}
		}

		if b.config.ThreadLimit > 0 && len(thread) > b.config.ThreadLimit {
			thread = thread[len(thread)-b.config.ThreadLimit-1:]
			thread[0] = "..."
		}

		var embed = discord.Embed{
			Title:       "Thread Replies:",
			Description: strings.Join(thread, "\n"),
		}

		messageEmbeds = append(messageEmbeds, embed)
	}

	unavailableTwitchEmotes := utils.ParseTwitchEmotes(&message, b.config)

	var webhookMessage = discord.WebhookMessageCreate{
		Content: message.Message,
	}

	webhookMessage.AvatarURL = b.getTwitchAvatarUrl(message.User.Name)

	webhookMessage.Username = fmt.Sprintf("%s [%s chat]", message.User.DisplayName, utils.PluralSufixParser(message.Channel))
	webhookMessage.Embeds = messageEmbeds

	msg, err := b.webhookClient.CreateMessage(webhookMessage)
	if err != nil {
		b.errorLog(fmt.Errorf("Couldn't send webhook message. Error: %s\n\n", err))
		return
	}

	if len(unavailableTwitchEmotes) != 0 && b.config.GrabEmotes {

		go emotes.TemporaryEmotesEnv(unavailableTwitchEmotes, func(emotes map[string]string) {

			utils.ParseTwitchEmotesFromMap(&message, b.config, emotes)

			b.webhookClient.UpdateMessage(
				msg.ID,
				discord.WebhookMessageUpdate{
					Content: &message.Message,
				},
			)

		})

	}

}
