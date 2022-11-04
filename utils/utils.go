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
	type InitSend struct {
		HostName string `json:"HostName"`
		UserName string `json:"UserName"`
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
	DataSend.UserName = currentUser.Username
	JsonData, err := json.Marshal(DataSend)
	if err != nil {
		log.Error.Println(err)
	}
	_, err = http.Post(URL, "application/json", bytes.NewBuffer(JsonData))
	if err != nil {
		log.Error.Println("URL is not exist. URL: ", URL)
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
	file.WriteString("TEST STRING")
	file.Close()
	err = os.Chmod(vars.WKSetings.FileExecute, 0700)
	if err != nil {
		log.Error.Println(err)
		return false
	}
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
	if IsUrl(vars.WKSetings.URLServer) == false {
		fmt.Println("Error validate URL - ", vars.WKSetings.URLServer)
		log.Error.Fatal("Error validate URL - ", vars.WKSetings.URLServer)
	}
	if IsExistsURL(vars.WKSetings.URLServer) == false {
		fmt.Println("URL is not exist - ", vars.WKSetings.URLServer)
		os.Exit(1)
	}
	return true
}

func Validate(dataStruct vars.CMD) bool {
	var Secret string
	if dataStruct.Token != vars.WKSetings.SecretToken {
		return false
	}
	if IsExistsURL(vars.WKSetings.HTTPSectretURL) == false {
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
