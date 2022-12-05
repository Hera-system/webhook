package vars

type WebhookSetings struct {
	Port           int    `json:"Port"`
	LogPath        string `json:"LogPath"`
	URLServer      string `json:"URLServer"`
	FileExecute    string `json:"FileExecute"`
	SecretToken    string `json:"SecretToken"`
	HTTPSectretURL string `json:"HTTPSectretURL"`
	UniqName       string `json:"UniqName"`
	WebhookURL     string `json:"WebhookURL"`
}

type CMD struct {
	TimeExec     int    `json:"TimeExec"`
	ID           string `json:"ID"`
	Token        string `json:"Token"`
	Shebang      string `json:"Shebang"`
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
