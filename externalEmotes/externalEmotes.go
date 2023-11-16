package externalemotes

type emotesRepo interface {
	getEmotesFromUserId(string) map[string]ExternalEmote
	getGlobalEmotes() (value map[string]ExternalEmote)
}

type ExternalEmote struct {
	Url string
	X   int
	Y   int
}

var repos = []emotesRepo{
	sevenTV{},
}

func GetAll(channelId string, userId string) (returnValue map[string]ExternalEmote) {
	for _, repo := range repos {
		var channelEmotes = repo.getEmotesFromUserId(channelId)

		for key, value := range channelEmotes {
			returnValue[key] = value
		}

		var userEmotes = repo.getEmotesFromUserId(userId)

		for key, value := range userEmotes {
			returnValue[key] = value
		}

		var globalEmotes = repo.getGlobalEmotes()

		for key, value := range globalEmotes {
			returnValue[key] = value
		}

	}

	return
}
