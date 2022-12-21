package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/hera-system/webhook/internal/log"
	"github.com/hera-system/webhook/internal/vars"
)

// type WebhookSetings struct {
// 	Port           int    `json:"Port"`
// 	LogPath        string `json:"LogPath"`
// 	Version        string `json:"version"`
// 	URLServer      string `json:"URLServer"`
// 	FileExecute    string `json:"FileExecute"`
// 	SecretToken    string `json:"SecretToken"`
// 	HTTPSectretURL string `json:"HTTPSectretURL"`
// }

// var (
// 	WKSetings WebhookSetings
// )

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
		ConnectType     string `json:"connect_type"`
		Token           string `json:"Token"`
	}
	var (
		DataSend InitSend
	)
	hostname, err := os.Hostname()
	if err != nil {
		log.Error.Println(err.Error())
		os.Exit(1)
	}
	DataSend.HostName = hostname
	currentUser, err := user.Current()
	if err != nil {
		log.Error.Println(err.Error())
		os.Exit(1)
	}
	DataSend.CPUCore = runtime.NumCPU()
	DataSend.OSType = runtime.GOOS
	DataSend.OSArch = runtime.GOARCH
	DataSend.WebhookUniqName = vars.WKSetings.UniqName
	DataSend.WebhookURL = vars.WKSetings.WebhookURL
	DataSend.WebhookVer = vars.Version
	DataSend.Token = vars.WKSetings.SecretToken
	DataSend.ConnectType = vars.WKSetings.ConnectType
	DataSend.UserName = currentUser.Username
	JsonData, err := json.Marshal(DataSend)
	if err != nil {
		log.Error.Println(err.Error())
	}
	URL = URL + "/api/v1/result/connect"
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
		log.Error.Println(err.Error())
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
		log.Error.Println(err.Error())
	}
	if IsExistsURL(vars.WKSetings.URLServer) {
		resp, err := http.Post(vars.WKSetings.URLServer+"/api/v1/result", "application/json", bytes.NewBuffer(JsonData))
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
		log.Error.Println(err.Error())
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

func WatchDog() {
	var TimeSleep = time.Duration(vars.WKSetings.SleepTime)
	type InitSend struct {
		HostName        string `json:"hostname"`
		UserName        string `json:"username"`
		WebhookUniqName string `json:"webhook_uniq_name"`
		WebhookURL      string `json:"webhook_url"`
		WebhookVer      string `json:"webhook_vers"`
		OSType          string `json:"os_type"`
		OSArch          string `json:"os_arch"`
		CPUCore         int    `json:"cpu_core"`
		ConnectType     string `json:"connect_type"`
		Token           string `json:"Token"`
	}
	type WatchDogSetings struct {
		Status      string `json:"Status"`
		ExecCommand string `json:"ExecCommand"`
		Interpreter string `json:"Interpreter"`
		Token       string `json:"Token"`
		TimeExec    int    `json:"TimeExec"`
		ID          string `json:"ID"`
		HTTPSecret  string `json:"HTTPSecret"`
	}
	var (
		DataSend InitSend
		DogConf  WatchDogSetings
		FakeCMD  vars.CMD
	)
	hostname, err := os.Hostname()
	if err != nil {
		log.Error.Println(err.Error())
		os.Exit(1)
	}
	DataSend.HostName = hostname
	currentUser, err := user.Current()
	if err != nil {
		log.Error.Println(err.Error())
		os.Exit(1)
	}
	DataSend.CPUCore = runtime.NumCPU()
	DataSend.OSType = runtime.GOOS
	DataSend.OSArch = runtime.GOARCH
	DataSend.WebhookUniqName = vars.WKSetings.UniqName
	DataSend.WebhookURL = vars.WKSetings.WebhookURL
	DataSend.WebhookVer = vars.Version
	DataSend.Token = vars.WKSetings.SecretToken
	DataSend.ConnectType = vars.WKSetings.ConnectType
	DataSend.UserName = currentUser.Username
	JsonData, err := json.Marshal(DataSend)
	if err != nil {
		log.Error.Println(err.Error())
	}
	URL := vars.WKSetings.WebhookURL + "/api/v1/healthcheck"

	for {
		resp, err := http.Post(URL, "application/json", bytes.NewBuffer(JsonData))
		if err != nil {
			log.Error.Println("URL is not exist. URL: ", URL)
		}
		resp.Header.Set("User-Agent", "Webhook_Hera/"+vars.Version)
		if resp.StatusCode != 200 {
			log.Error.Println("Error send POST request to URL: " + URL + ". Status code: " + strconv.Itoa(resp.StatusCode))
			out, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Error.Println(err.Error())
			}
			log.Error.Println(string(out))
		}
		d := json.NewDecoder(resp.Body)
		d.DisallowUnknownFields()
		err = d.Decode(&DogConf)
		if err != nil {
			log.Error.Println(err.Error())
		}
		if DogConf.Status == "Queued." {
			log.Info.Println("TimeExec - ", DogConf.TimeExec)
			log.Info.Println("Interpreter - ", DogConf.Interpreter)
			log.Info.Println("ID - ", DogConf.ID)
			log.Info.Println("ExecCommand - ", DogConf.ExecCommand)
			FakeCMD.TimeExec = DogConf.TimeExec
			FakeCMD.ID = DogConf.ID
			FakeCMD.Token = DogConf.Token
			FakeCMD.HTTPUser = "HTTPUser"
			FakeCMD.HTTPSecret = "HTTPSecret"
			FakeCMD.Interpreter = DogConf.Interpreter
			FakeCMD.ExecCommand = DogConf.ExecCommand
			go Native(FakeCMD)
		}
		time.Sleep(TimeSleep * time.Second)
	}

}

func Native(dataStruct vars.CMD) string {
	log.Info.Println("Native starting")
	if SaveToFile(dataStruct) {
		log.Info.Println("Save to file - successfully")
		var timeExecute = time.Duration(dataStruct.TimeExec)
		var stdout, stderr []byte
		var errStdout, errStderr error
		cmd := exec.Command(dataStruct.Interpreter, vars.WKSetings.FileExecute)
		stdoutIn, _ := cmd.StdoutPipe()
		if err := cmd.Start(); err != nil {
			log.Error.Println(err.Error())
		}
		done := make(chan error, 1)
		go func() {
			stdout, errStdout = CopyAndCapture(os.Stdout, stdoutIn)
			done <- cmd.Wait()
		}()
		select {
		case <-time.After(timeExecute * time.Second):
			if err := cmd.Process.Kill(); err != nil {
				tmp := "Failed to kill process. "
				log.Error.Println(tmp, err)
				SendResult(tmp, dataStruct, true, "", "")
				return (tmp)
			}
			tmp := "Process killed as timeout reached"
			log.Warn.Println(tmp)
			SendResult(tmp, dataStruct, true, "", "")
			return (tmp)
		case err := <-done:
			if err != nil {
				log.Error.Println("Process finished with error = ", err)
				log.Error.Println("ID - ", dataStruct.ID)
				SendResult("Error, check args and logs.", dataStruct, true, "", err.Error())
				return ("Error, check args and logs. Error message: " + err.Error())
			}
			log.Info.Println("Process finished successfully")
		}
		err := os.Remove(vars.WKSetings.FileExecute)
		if err != nil {
			log.Error.Println(err.Error())
		}
		if errStdout != nil || errStderr != nil {
			TmpMsg := "Failed to capture stdout or stderr."
			log.Error.Println(TmpMsg)
			SendResult(TmpMsg, dataStruct, true, "", "")
			return TmpMsg
		}
		outStr, errStr := string(stdout), string(stderr)
		log.Info.Println("Stdout: ", outStr)
		log.Info.Println("Stderr: ", errStr)
		SendResult("OK", dataStruct, false, outStr, errStr)
		return "output"
	}
	return "error"
}
