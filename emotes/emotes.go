package emotes

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
	discordbot "twitch2discordbridge/discordBot"
)

const (
	emote_format = "https://static-cdn.jtvnw.net/emoticons/v2/%s/default/dark/2.0"
)

var emotesCache = map[string]string{}

func GetEmotes() map[string]string {
	return emotesCache
}

type configFile struct {
	fileName string
	lock     sync.Mutex
}

var config *configFile

func LoadConfiguration(fileName string) error {
	config = &configFile{
		fileName: fileName,
	}

	return UpdateCache()
}

func UpdateCache() (err error) {

	if config == nil {
		return fmt.Errorf("Configuration file wasn't initialized.")
	}

	var newCache = map[string]string{}

	data, err := config.readFromFille()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error updating %s: %v", config.fileName, err)
		return
	}

	for _, item := range data {
		if len(item) < 2 {
			continue
		}

		newCache[strings.TrimSpace(item[0])] = strings.TrimSpace(item[1])
	}

	emotesCache = newCache

	println("Emotes loaded\n")

	return
}

func (c *configFile) writeToFile() (err error) {

	file, err := os.OpenFile(c.fileName, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		return
	}

	defer file.Close()

	csvWriter := csv.NewWriter(file)

	var records [][]string

	for key, value := range emotesCache {
		line := []string{
			key,
			value,
		}

		records = append(records, line)
	}

	csvWriter.WriteAll(records)

	return
}

func (c *configFile) readFromFille() (value [][]string, err error) {

	file, err := os.OpenFile(c.fileName, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n\n", err)
		return
	}

	defer file.Close()

	csvReader := csv.NewReader(file)

	value, err = csvReader.ReadAll()

	if err != nil {
		fmt.Printf("Error: %s\n\n", err)
		return
	}

	return
}

type emote struct {
	DiscordName string
	Id          string
}

var temporaryEmotesEnvLock sync.Mutex

func TemporaryEmotesEnv(emotes map[string]string, action func(map[string]string)) {

	temporaryEmotesEnvLock.Lock()

	defer temporaryEmotesEnvLock.Unlock()

	var tmpEmotes = map[string]string{}

	for key, value := range emotes {

		value, err := RequestTwitchEmote(key, value)

		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't request emote %s with key %s. Error: %s", key, value, err)
			continue
		}

		emotes[key] = value.DiscordName
		tmpEmotes[key] = value.Id

	}

	action(emotes)

	time.Sleep(5 * time.Second)

	for _, value := range tmpEmotes {
		discordbot.DeleteEmote(value)
	}
}

func RequestTwitchEmote(emoteName string, emoteId string) (em emote, err error) {
	// Make an HTTP GET request to fetch the image
	response, err := http.Get(fmt.Sprintf(emote_format, emoteId))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer response.Body.Close()

	// Read the image data
	imageData, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading binary file: %s", err)
		return
	}

	format, err := getImageFormat(imageData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting the file format: %s", err)
		return
	}

	result, err := discordbot.CreateEmote(emoteName, imageData, format)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating emotes in discord: %s\n\n", err)
		return
	}

	var animated string
	if result.Animated {
		animated = "a"
	}

	em.DiscordName = fmt.Sprintf("<%s:%s:%s>", animated, result.Name, result.ID)
	em.Id = result.ID.String()

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating emotes in discord: %s\n\n", err)
		return
	}

	return
}

func getImageFormat(data []byte) (string, error) {

	formats := map[string]string{
		"\xFF\xD8\xFF":                     "jpeg",
		"\x89\x50\x4E\x47\x0D\x0A\x1A\x0A": "png",
		"GIF89a":                           "gif",
		"APNG":                             "apng",
		"RIFF\xFF\xFF\xFF\xFFWEBPVP8 ":     "webp",
	}

	for signature, format := range formats {
		if strings.HasPrefix(string(data), signature) {
			return format, nil
		}
	}

	return "", fmt.Errorf("Unknown file format")
}
