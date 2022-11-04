package vars

type WebhookSetings struct {
	Port           int    `json:"Port"`
	LogPath        string `json:"LogPath"`
	Version        string `json:"version"`
	URLServer      string `json:"URLServer"`
	FileExecute    string `json:"FileExecute"`
	SecretToken    string `json:"SecretToken"`
	HTTPSectretURL string `json:"HTTPSectretURL"`
}

var (
	WKSetings WebhookSetings
)
