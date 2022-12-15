package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"

	"github.com/hera-system/webhook/internal/log"
	"github.com/hera-system/webhook/internal/vars"
)

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

func IsExistsURL(URL string) bool {
	if strings.Contains(URL, vars.WKSetings.URLServer) {
		return SendInfoToHera(URL)
	}
	resp, err := http.Head(URL)
	if err != nil {
		log.Error.Println("URL is not exist. URL: ", URL)
		return false
	}
	resp.Header.Set("User-Agent", "Webhook_Hera/"+vars.Version)
	if resp.StatusCode != 200 {
		log.Error.Println("Error send POST request to URL: " + URL + ". Status code: " + strconv.Itoa(resp.StatusCode))
		out, err := io.ReadAll(resp.Body)
		if err != nil {
			return false
		}
		log.Error.Println(string(out))
		return false
	}

	return true
}

func SendInfoToHera(URL string) bool {
	type InitSend struct {
		HostName        string `json:"hostname"`
		UserName        string `json:"username"`
		WebhookUniqName string `json:"webhook_uniq_name"`
		WebhookURL      string `json:"webhook_url"`
		WebhookVer      string `json:"webhook_vers"`
		OSType          string `json:"os_type"`
		OSArch          string `json:"os_arch"`
		CPUCore         int    `json:"cpu_core"`
		Token           string `json:"Token"`
	}
	var (
		DataSend InitSend
	)
	hostname, err := os.Hostname()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	DataSend.HostName = hostname
	currentUser, err := user.Current()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	DataSend.CPUCore = runtime.NumCPU()
	DataSend.OSType = runtime.GOOS
	DataSend.OSArch = runtime.GOARCH
	DataSend.WebhookUniqName = vars.WKSetings.UniqName
	DataSend.WebhookURL = vars.WKSetings.WebhookURL
	DataSend.WebhookVer = vars.Version
	DataSend.Token = vars.WKSetings.SecretToken
	DataSend.UserName = currentUser.Username
	JsonData, err := json.Marshal(DataSend)
	if err != nil {
		log.Error.Println(err)
	}
	URL = URL + "/connect"
	resp, err := http.Post(URL, "application/json", bytes.NewBuffer(JsonData))
	if err != nil {
		log.Error.Println("URL is not exist. URL: ", URL)
		return false
	}
	resp.Header.Set("User-Agent", "Webhook_Hera/"+vars.Version)
	if resp.StatusCode != 200 {
		log.Error.Println("Error send POST request to URL: " + URL + ". Status code: " + strconv.Itoa(resp.StatusCode))
		out, err := io.ReadAll(resp.Body)
		if err != nil {
			return false
		}
		log.Error.Println(string(out))
		return false
	}

	return true
}

func IsUrl(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func TestAfterStart() bool {
	log.Info.Println("Test after start - starting.")
	file, err := os.Create(vars.WKSetings.FileExecute)
	if err != nil {
		tmp := "Unable to create exetuable file. Path: " + vars.WKSetings.FileExecute + ". Shutdown webhook..."
		fmt.Println(tmp)
		log.Error.Fatalln(tmp, err)
		return false
	}
	_, err = file.WriteString("TEST STRING")
	if err != nil {
		log.Error.Println("Error file write")
		return false
	}
	file.Close()
	err = os.Remove(vars.WKSetings.FileExecute)
	if err != nil {
		log.Error.Println(err)
		return false
	}
	if vars.WKSetings.SecretToken == "" {
		fmt.Println("Args SecretToken not used. Exit.")
		log.Error.Fatal("Args SecretToken not used. Exit.")
	}
	if vars.WKSetings.HTTPSectretURL == "" {
		fmt.Println("Args HTTPSecret not used. Exit.")
		log.Error.Fatal("Args HTTPSecret not used. Exit.")
	}
	if vars.WKSetings.URLServer == "" {
		fmt.Println("Args URL not used. Exit.")
		log.Error.Fatal("Args URL not used. Exit.")
	}
	if !IsUrl(vars.WKSetings.URLServer) {
		fmt.Println("Error validate URL - ", vars.WKSetings.URLServer)
		log.Error.Fatal("Error validate URL - ", vars.WKSetings.URLServer)
	}
	if !IsExistsURL(vars.WKSetings.URLServer) {
		fmt.Println("URL is not exist - ", vars.WKSetings.URLServer)
		os.Exit(1)
	}
	return true
}

func Valid(dataStruct vars.CMD) bool {
	var Secret string
	if dataStruct.Token != vars.WKSetings.SecretToken {
		return false
	}
	if !IsExistsURL(vars.WKSetings.HTTPSectretURL) {
		log.Error.Println("Secret url is not exist.")
	}
	res, err := http.Get(vars.WKSetings.HTTPSectretURL)
	if err != nil {
		log.Error.Println("Error making http request: ", err)
		return false
	}
	if res.StatusCode == 401 {
		res.Request.SetBasicAuth(dataStruct.HTTPUser, dataStruct.HTTPPassword)
		res, err := http.Get(vars.WKSetings.HTTPSectretURL)
		if err != nil {
			log.Error.Println("Error making http request: ", err)
			return false
		}
		out, err := io.ReadAll(res.Body)
		if err != nil {
			return false
		}
		Secret = string(out)
	}
	if res.StatusCode != 200 {
		log.Error.Println("Resonse code is not 200: ", res.StatusCode)
		return false
	}
	out, err := io.ReadAll(res.Body)
	if err != nil {
		log.Error.Println("Error read body: ", err)
		return false
	}
	Secret = string(out)
	if Secret != dataStruct.HTTPSecret {
		log.Error.Println("Error HTTPSecret")
		return false
	}
	return true
}

func SendResult(data string, dataStruct vars.CMD, Error bool, Stdout string, Stderr string) bool {
	type dataResponse struct {
		Error   bool   `json:"Error"`
		ID      string `json:"ID"`
		Token   string `json:"Token"`
		Stdout  string `json:"Stdout"`
		Stderr  string `json:"Stderr"`
		Message string `json:"Message"`
	}
	response := dataResponse{ID: dataStruct.ID, Error: Error, Token: dataStruct.Token, Message: data, Stderr: Stderr, Stdout: Stdout}
	JsonData, err := json.Marshal(response)
	if err != nil {
		log.Error.Println(err)
	}
	if IsExistsURL(vars.WKSetings.URLServer) {
		resp, err := http.Post(vars.WKSetings.URLServer, "application/json", bytes.NewBuffer(JsonData))
		if err != nil {
			log.Error.Println("An Error Occurred ", err)
		}
		if resp.StatusCode == 200 {
			return true
		}
		log.Error.Println("Status code != 200. Status code is ", resp.StatusCode)
		log.Error.Println("Error ID - ", response.ID)
	}
	return false
}

func SaveToFile(dataStruct vars.CMD) bool {
	file, err := os.Create(vars.WKSetings.FileExecute)
	if err != nil {
		tmp := "Unable to create exetuable file. Path: " + vars.WKSetings.FileExecute + ". Shutdown webhook..."
		fmt.Println(tmp)
		SendResult(tmp, dataStruct, true, "", "")
		log.Error.Fatalln(tmp, err)
		return false
	}
	_, err = file.WriteString(dataStruct.ExecCommand)
	if err != nil {
		log.Error.Println("Error file write")
		return false
	}
	file.Close()
	err = os.Chmod(vars.WKSetings.FileExecute, 0700)
	if err != nil {
		log.Error.Println(err)
		return false
	}
	return true
}

func CopyAndCapture(w io.Writer, r io.Reader) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024)
	for {
		n, err := r.Read(buf[:])
		if n > 0 {
			d := buf[:n]
			out = append(out, d...)
			_, err := w.Write(d)
			if err != nil {
				return out, err
			}
		}
		if err != nil {
			// Read returns io.EOF at the end of file, which is not an error for us
			if err == io.EOF {
				err = nil
			}
			return out, err
		}
	}
}
