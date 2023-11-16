package externalemotes

type sevenTVSchema struct {
	ID            string      `json:"id"`
	Platform      string      `json:"platform"`
	Username      string      `json:"username"`
	DisplayName   string      `json:"display_name"`
	LinkedAt      int64       `json:"linked_at"`
	EmoteCapacity int         `json:"emote_capacity"`
	EmoteSetID    interface{} `json:"emote_set_id"`
	EmoteSet      struct {
		ID         string        `json:"id"`
		Name       string        `json:"name"`
		Flags      int           `json:"flags"`
		Tags       []interface{} `json:"tags"`
		Immutable  bool          `json:"immutable"`
		Privileged bool          `json:"privileged"`
		Emotes     []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Flags     int    `json:"flags"`
			Timestamp int64  `json:"timestamp"`
			ActorID   string `json:"actor_id"`
			Data      struct {
				ID        string   `json:"id"`
				Name      string   `json:"name"`
				Flags     int      `json:"flags"`
				Lifecycle int      `json:"lifecycle"`
				State     []string `json:"state"`
				Listed    bool     `json:"listed"`
				Animated  bool     `json:"animated"`
				Owner     struct {
					ID          string `json:"id"`
					Username    string `json:"username"`
					DisplayName string `json:"display_name"`
					AvatarURL   string `json:"avatar_url"`
					Style       struct {
					} `json:"style"`
					Roles []string `json:"roles"`
				} `json:"owner"`
				Host struct {
					URL   string `json:"url"`
					Files []struct {
						Name       string `json:"name"`
						StaticName string `json:"static_name"`
						Width      int    `json:"width"`
						Height     int    `json:"height"`
						FrameCount int    `json:"frame_count"`
						Size       int    `json:"size"`
						Format     string `json:"format"`
					} `json:"files"`
				} `json:"host"`
			} `json:"data"`
		} `json:"emotes"`
		EmoteCount int `json:"emote_count"`
		Capacity   int `json:"capacity"`
		Owner      struct {
			ID          string `json:"id"`
			Username    string `json:"username"`
			DisplayName string `json:"display_name"`
			AvatarURL   string `json:"avatar_url"`
			Style       struct {
				Color int `json:"color"`
			} `json:"style"`
			Roles []string `json:"roles"`
		} `json:"owner"`
	} `json:"emote_set"`
	User struct {
		ID          string `json:"id"`
		Username    string `json:"username"`
		DisplayName string `json:"display_name"`
		CreatedAt   int64  `json:"created_at"`
		AvatarURL   string `json:"avatar_url"`
		Style       struct {
			Color int `json:"color"`
		} `json:"style"`
		EmoteSets []struct {
			ID       string        `json:"id"`
			Name     string        `json:"name"`
			Flags    int           `json:"flags"`
			Tags     []interface{} `json:"tags"`
			Capacity int           `json:"capacity"`
		} `json:"emote_sets"`
		Roles       []string `json:"roles"`
		Connections []struct {
			ID            string      `json:"id"`
			Platform      string      `json:"platform"`
			Username      string      `json:"username"`
			DisplayName   string      `json:"display_name"`
			LinkedAt      int64       `json:"linked_at"`
			EmoteCapacity int         `json:"emote_capacity"`
			EmoteSetID    interface{} `json:"emote_set_id"`
			EmoteSet      struct {
				ID         string        `json:"id"`
				Name       string        `json:"name"`
				Flags      int           `json:"flags"`
				Tags       []interface{} `json:"tags"`
				Immutable  bool          `json:"immutable"`
				Privileged bool          `json:"privileged"`
				Capacity   int           `json:"capacity"`
				Owner      interface{}   `json:"owner"`
			} `json:"emote_set"`
		} `json:"connections"`
	} `json:"user"`
}
