package twitchApi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"twitch2discordbridge/configuration"
)

type TwitchApi struct {
	TwitchClientId string
	TwitchUsername string
	OauthPassword  string
	Channel        string
}

type QueryType string

const (
	Username QueryType = "login"
	Id                 = "id"
)

func LoadFromConfig(config configuration.Config) (t TwitchApi) {
	t.TwitchClientId = config.TwitchClientId
	t.TwitchUsername = config.TwitchUsername
	t.OauthPassword = config.OauthPassword
	t.Channel = config.Channel

	return t
}

type Wrapper struct {
	Data []TwitchUserInfo
}

type TwitchUserInfo struct {
	Id              string `json:"id"`
	Login           string `json:"login"`
	DisplayName     string `json:"display_name"`
	Type            string `json:"type"`
	BroadcasterType string `json:"broadcaster_type"`
	Description     string `json:"description"`
	ProfileImageUrl string `json:"profile_image_url"`
	OfflineImageUrl string `json:"offline_image_url"`
	ViewCount       int    `json:"view_count"`
	CreatedAt       string `json:"created_at"`
}

func (t *TwitchApi) GetProfileInfo(id string) (c TwitchUserInfo, err error) {

	url := fmt.Sprintf("https://api.twitch.tv/helix/users?id=%v", id)
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		return c, err
	}
	req.Header.Add("Client-ID", t.TwitchClientId)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", t.OauthPassword))

	res, err := client.Do(req)
	if err != nil {
		return c, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return c, err
	}

	var wrapper = Wrapper{}

	err = json.Unmarshal([]byte(string(body)), &wrapper)

	if len(wrapper.Data) == 0 {
		return c, err
	}

	return wrapper.Data[0], err
}
