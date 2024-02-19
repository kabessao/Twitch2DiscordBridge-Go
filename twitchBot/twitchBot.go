package twitchBot

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"twitch2discordbridge/configuration"
	"twitch2discordbridge/emotes"
	"twitch2discordbridge/utils"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/webhook"
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

	if utils.ParseHypeChat(&message, b.config) {
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

	b.log(fmt.Sprintf(message.Raw))

	var timeoutMessage = "`User got banned permanently`"

	if duration := message.BanDuration; duration > 0 {
		timeoutMessage = fmt.Sprintf("`User got timed out for %s`", utils.ParseIntDuration(duration))
	}

	usersInfo, err := b.helixApi.GetUsers(&helix.UsersParams{
		IDs: []string{message.TargetUserID},
	})

	if err != nil || len(usersInfo.Data.Users) == 0 {
		return
	}

	var userInfo = usersInfo.Data.Users[0]

	var messages = []twitch.PrivateMessage{
		{
			Message: timeoutMessage,
			User: twitch.User{
				DisplayName: userInfo.DisplayName,
				Name:        userInfo.Login,
			},
		},
	}

	for _, m := range b.messageHistory[message.Channel] {
		if m.User.ID == message.TargetUserID {

			if _, ok := m.Tags["__deleted"]; ok {
				continue
			}

			messages = append(messages, m)

			m.Tags["__deleted"] = "true"
		}
	}

	for index, item := range messages {
		b.sendMessage(
			item,
		)

		if index == 0 {
			time.Sleep(time.Second)
		}
	}

}

func (b *bot) onClearMessage(deletedMessage twitch.ClearMessage) {
	if !b.config.ModActions {
		return
	}

	println(deletedMessage.Raw + "\n")

	for _, m := range b.messageHistory[deletedMessage.Channel] {
		if m.ID == deletedMessage.TargetMsgID {

			if _, ok := m.Tags["__deleted"]; ok {
				continue
			}

			m.Message = fmt.Sprintf("`Message Deleted:`%s", m.Message)

			go b.sendMessage(
				m,
			)

			m.Tags["__deleted"] = "true"
		}
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

	client.OnConnect(b.onConnect)

	client.OnPrivateMessage(b.onPrivateMessage)

	client.OnClearChatMessage(b.onClearChatMessage)

	client.OnClearMessage(b.onClearMessage)

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

	if b.config.OnStreamStatus == "" {
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

		var thread = fmt.Sprintf("`%s`: %s\n", message.Reply.ParentDisplayName, parseMessage(message.Reply.ParentMsgBody))

		for _, msg := range b.messageHistory[message.Channel] {
			if msg.Reply == nil {
				continue
			}

			if msg.ID == message.Reply.ParentMsgID {
				continue
			}

			if msg.ID == message.ID {
				continue
			}

			if msg.Reply.ParentMsgID == message.Reply.ParentMsgID {
				thread += fmt.Sprintf("`%s`: %s\n", msg.User.DisplayName, parseMessage(msg.Message))
			}
		}

		var embed = discord.Embed{
			Title:       "Thread Replies:",
			Description: thread,
		}

		messageEmbeds = append(messageEmbeds, embed)
	}

	unavailableTwitchEmotes := utils.ParseTwitchEmotes(&message, b.config)

	var webhookMessage = discord.WebhookMessageCreate{
		Content: message.Message,
	}

	webhookMessage.AvatarURL = b.getTwitchAvatarUrl(message.User.Name)

	webhookMessage.Username = fmt.Sprintf("%s [%s chat]", message.User.DisplayName, utils.PluralSufixParser(b.config.Channel))
	webhookMessage.Embeds = messageEmbeds

	msg, err := b.webhookClient.CreateMessage(webhookMessage)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't send webhook message. Error: %s\n\n", err)
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
