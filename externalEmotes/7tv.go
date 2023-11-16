package externalemotes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	discordbot "twitch2discordbridge/discordBot"
)

type sevenTV struct{}

func (sevenTV) getGlobalEmotes() (value map[string]ExternalEmote) {

	value = map[string]ExternalEmote{}

	url := "https://7tv.io/v3/emote-sets/global"

	res, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	content, err := io.ReadAll(res.Body)

	print(string(content))

	var response sevenTVSchema

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	return parseEmotes(response)
}

func (sevenTV) getEmotesFromUserId(id string) (value map[string]ExternalEmote) {

	value = map[string]ExternalEmote{}

	url := fmt.Sprintf("https://7tv.io/v3/users/twitch/%s", id)

	res, err := http.Get(url)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()

	var response = &sevenTVSchema{}

	err = json.NewDecoder(res.Body).Decode(response)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	}

	return parseEmotes(*response)
}

func parseEmotes(response sevenTVSchema) (value map[string]ExternalEmote) {

	emotes := response.EmoteSet.Emotes

	for _, item := range emotes {
		for _, file := range item.Data.Host.Files {
			if !strings.HasPrefix(file.Name, "2x") {
				continue
			}

			if !discordbot.AllowedFormats.Contains(strings.ToLower(file.Format)) {
				continue
			}

			var em = ExternalEmote{
				Url: fmt.Sprintf("https:%s/%s", item.Data.Host.URL, file.StaticName),
				X:   file.Width,
				Y:   file.Height,
			}

			value[item.Name] = em
		}
	}

	return
}
