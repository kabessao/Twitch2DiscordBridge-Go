package bot

import (
	"fmt"
	"time"

	"twitch2discordbridge/configuration"
	"twitch2discordbridge/utils"

	"github.com/gempir/go-twitch-irc/v4"
	"github.com/nicklaw5/helix"
)

const (
	MESSAGE_HISTORY_LENGTH int = 800
)

type bot struct {
	fileName       string
	config         configuration.Config
	helixApi       *helix.Client
	client         *twitch.Client
	messageHistory []twitch.PrivateMessage
}

func (b *bot) log(message string) {
	fmt.Printf("%-30s| %s\n\n", b.fileName, message)
}

func (b *bot) onConnect() {
	b.log(fmt.Sprintf("connected to %s", b.config.Channel))
}

func (b *bot) onPrivateMessage(message twitch.PrivateMessage) {

	if b.config.OutputLog {
		go b.log(fmt.Sprintf(message.Raw))
	}

	b.messageHistory = append(b.messageHistory, message)

	if length := len(b.messageHistory); length > MESSAGE_HISTORY_LENGTH {
		b.messageHistory = b.messageHistory[length-MESSAGE_HISTORY_LENGTH : length]
	}

	if utils.StringArrayContains(b.config.Blacklist, message.User.Name) {
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

	if b.config.PreventPing {
		message.Message = utils.StringReplaceAllRegex("@(?=here|everyone)", message.Message, "")
	}

	userInfo, err := b.helixApi.GetUsers(&helix.UsersParams{
		Logins: []string{message.User.Name},
	})

	if err != nil {
		b.log(fmt.Sprintf("Error: %v", err))
	}

	if shouldSend {
		b.sendMessage(
			message,
			userInfo,
		)
	}

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

	for _, m := range b.messageHistory {
		if m.User.ID == message.TargetUserID {
			messages = append(messages, m)
		}
	}

	for index, item := range messages {
		b.sendMessage(
			item,
			usersInfo,
		)

		if index == 0 {
			time.Sleep(time.Second)
		}
	}

}

func (b *bot) onClearMessage(message twitch.ClearMessage) {
	if !b.config.ModActions {
		return
	}

	println(message.Raw + "\n")

	for _, m := range b.messageHistory {
		if m.ID == message.TargetMsgID {

			usersInfo, err := b.helixApi.GetUsers(&helix.UsersParams{
				IDs: []string{m.User.ID},
			})

			if err != nil || len(usersInfo.Data.Users) == 0 {
				return
			}

			m.Message = fmt.Sprintf("`Message Deleted:`%s", m.Message)

			b.sendMessage(
				m,
				usersInfo,
			)
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

	defer func() {
		channel.IsOk = false
	}()

	var config, err = configuration.LoadConfigFromFile(filePath)
	if err != nil {
		fmt.Printf("Error: [%s] %v\n\n", filePath, err)
		return
	}

	var bot = bot{
		config:   config,
		fileName: filePath,
	}

	bot.log("Starting from config file")

	if err := bot.loadConfiguration(); err != nil {
		bot.log(fmt.Sprintf("Error: %v", err))
		return
	}

	if err := bot.startClient(); err != nil {
		bot.log(fmt.Sprintf("Error: %v", err))
		return
	}

	defer bot.log(fmt.Sprintf("[%s] This instance is shuting down", filePath))

	for {
		select {
		case _, ok := <-channel.Channel:
			if !ok {
				bot.client.Disconnect()
				return
			}

			bot.log(fmt.Sprintf("Reloading config file [%v]", filePath))

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

	b.client.Join(b.config.Channel)

	return err
}

func (b *bot) sendMessage(message twitch.PrivateMessage, userInfo *helix.UsersResponse) {

	for _, badge := range []string{"broadcaster", "moderator", "vip"} {
		if _, ok := message.User.Badges[badge]; ok {
			message.User.DisplayName = fmt.Sprintf("%s [%s]", message.User.DisplayName, badge)
		}
	}

	message.User.DisplayName = fmt.Sprintf("%s [%s chat]", message.User.DisplayName, utils.PluralSufixParser(b.config.Channel))

	utils.EmoteParser(&message, b.config)

	utils.SendWebhookMessage(
		message,
		*userInfo,
		b.config,
	)

}
