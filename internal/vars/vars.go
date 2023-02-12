package vars

type WebhookSetings struct {
	Port          int    `json:"Port"`
	LogPath       string `json:"LogPath"`
	URLServer     string `json:"URLServer"`
	FileExecute   string `json:"FileExecute"`
	SecretToken   string `json:"SecretToken"`
	HTTPSecretURL string `json:"HTTPSecretURL"`
	UniqName      string `json:"UniqName"`
	WebhookURL    string `json:"WebhookURL"`
	ConnectType   string `json:"ConnectType"`
	SleepTime     int    `json:"SleepTime"`
}

type CMD struct {
	TimeExec     int    `json:"TimeExec"`
	ID           string `json:"ID"`
	Token        string `json:"Token"`
	HTTPUser     string `json:"HTTPUser"`
	HTTPSecret   string `json:"HTTPSecret"`
	Interpreter  string `json:"Interpreter"`
	ExecCommand  string `json:"ExecCommand"`
	HTTPPassword string `json:"HTTPPassword"`
}

var (
	WKSetings WebhookSetings
)

const Version string = "v1.0.2"
