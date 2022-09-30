package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"time"
)

type CMD struct {
	ExecCommand  string `json:"ExecCommand"`
	Shebang      string `json:"Shebang"`
	TimeExec     int    `json:"TimeExec"`
	Token        string `json:"Token"`
	Interpreter  string `json:"Interpreter"`
	ID           string `json:"ID"`
	HTTPUser     string `json:"HTTPUser"`
	HTTPPassword string `json:"HTTPPassword"`
	HTTPSecret   string `json:"HTTPSecret"`
}

type dataResponse struct {
	Error   bool   `json:"Error"`
	Stdout  string `json:"Stdout"`
	Stderr  string `json:"Stderr"`
	ID      string `json:"ID"`
	Token   string `json:"Token"`
	Message string `json:"Message"`
}

var (
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
)

type WebhookSetings struct {
	Version     string `json:"version"`
	URLServer   string `json:"URLServer"`
	Port        int    `json:"Port"`
	LogPath     string `json:"LogPath"`
	FileExecute string `json:"FileExecute"`
}

var WKSetings WebhookSetings

func LogFunc() {
	file, err := os.OpenFile(WKSetings.LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}

	InfoLogger = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningLogger = log.New(file, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorLogger = log.New(file, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func HealtCheak(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed."))
		return
	}
	w.Write([]byte(WKSetings.Version))
	return
}

func saveToFile(dataStruct CMD) bool {
	file, err := os.Create(WKSetings.FileExecute)
	if err != nil {
		tmp := "Unable to create exetuable file. Path: " + WKSetings.FileExecute + ". Shutdown webhook..."
		fmt.Println(tmp)
		sendResult(tmp, dataStruct, true, "", "")
		ErrorLogger.Fatalln(tmp, err)
		return false
	}
	file.WriteString(dataStruct.Shebang + "\n")
	file.WriteString(dataStruct.ExecCommand)
	file.Close()
	err = os.Chmod(WKSetings.FileExecute, 0700)
	if err != nil {
		ErrorLogger.Println(err)
		return false
	}
	return true
}

func IsExistsURL(URL string) bool {
	_, err := http.Head(URL)
	if err != nil {
		ErrorLogger.Println("URL is not exist. URL: ", URL)
		return false
	}
	return true
}

func sendResult(data string, dataStruct CMD, Error bool, Stdout string, Stderr string) bool {
	response := dataResponse{ID: dataStruct.ID, Error: Error, Token: dataStruct.Token, Message: data, Stderr: Stderr, Stdout: Stdout}
	json_data, err := json.Marshal(response)
	if err != nil {
		ErrorLogger.Println(err)
	}
	if IsExistsURL(WKSetings.URLServer) {
		resp, err := http.Post(WKSetings.URLServer, "application/json", bytes.NewBuffer(json_data))
		if err != nil {
			ErrorLogger.Println("An Error Occured ", err)
		}
		if resp.StatusCode == 200 {
			return true
		}
		ErrorLogger.Println("Status code != 200. Status code is ", resp.StatusCode)
		ErrorLogger.Println("Error ID - ", response.ID)
	}
	return false
}

func copyAndCapture(w io.Writer, r io.Reader) ([]byte, error) {
	var out []byte
	buf := make([]byte, 1024, 1024)
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

func Native(dataStruct CMD) string {
	if saveToFile(dataStruct) {
		var timeExecute = time.Duration(dataStruct.TimeExec)
		var stdout, stderr []byte
		var errStdout, errStderr error
		cmd := exec.Command(dataStruct.Interpreter, WKSetings.FileExecute)
		stdoutIn, _ := cmd.StdoutPipe()
		if err := cmd.Start(); err != nil {
			ErrorLogger.Println(err)
		}
		done := make(chan error, 1)
		go func() {
			stdout, errStdout = copyAndCapture(os.Stdout, stdoutIn)
			done <- cmd.Wait()
		}()
		select {
		case <-time.After(timeExecute * time.Second):
			if err := cmd.Process.Kill(); err != nil {
				tmp := "Failed to kill proccess. "
				ErrorLogger.Println(tmp, err)
				sendResult(tmp, dataStruct, true, "", "")
				return (tmp)
			}
			tmp := "Process killed as timeout reached"
			WarningLogger.Println(tmp)
			sendResult(tmp, dataStruct, true, "", "")
			return (tmp)
		case err := <-done:
			if err != nil {
				ErrorLogger.Println("Process finished with error = ", err)
				ErrorLogger.Println("ID - ", dataStruct.ID)
				sendResult("Error, check args and logs.", dataStruct, true, "", "")
				return ("Error, check args and logs.")
			}
			InfoLogger.Println("Process finished successfully")
		}
		err := os.Remove(WKSetings.FileExecute)
		if err != nil {
			ErrorLogger.Println(err)
		}
		if errStdout != nil || errStderr != nil {
			ErrorLogger.Println("failed to capture stdout or stderr")
			sendResult("Failed to capture stdout or stderr.", dataStruct, true, "", "")
			return "Failed to capture stdout or stderr."
		}
		outStr, errStr := string(stdout), string(stderr)
		InfoLogger.Println("Stdout: ", outStr)
		InfoLogger.Println("Stderr: ", errStr)
		sendResult("OK", dataStruct, false, outStr, errStr)
		return "output"
	}
	return "error"
}

func Validate(dataStruct CMD) bool {
	var Token string = "VeryStrongString"
	var SecretURL string = "https://raw.githubusercontent.com/Hera-system/TOTP/main/TOTP"
	var Secret string
	if dataStruct.Token != Token {
		return false
	}
	if IsExistsURL(SecretURL) == false {
		ErrorLogger.Println("Secret url is not exist.")
	}
	res, err := http.Get(SecretURL)
	if err != nil {
		ErrorLogger.Println("Error making http request: ", err)
		return false
	}
	if res.StatusCode == 401 {
		res.Request.SetBasicAuth(dataStruct.HTTPUser, dataStruct.HTTPPassword)
		res, err := http.Get(SecretURL)
		if err != nil {
			ErrorLogger.Println("Error making http request: ", err)
			return false
		}
		out, err := io.ReadAll(res.Body)
		if err != nil {
			return false
		}
		Secret = string(out)
	}
	if res.StatusCode != 200 {
		ErrorLogger.Println("Resonse code is not 200: ", res.StatusCode)
		return false
	}
	out, err := io.ReadAll(res.Body)
	if err != nil {
		ErrorLogger.Println("Error read body: ", err)
		return false
	}
	Secret = string(out)
	if Secret != dataStruct.HTTPSecret {
		ErrorLogger.Println("Error HTTPSecret")
		return false
	}
	return true
}

func ExecuteCommand(w http.ResponseWriter, r *http.Request) {
	var dataStruct CMD
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed."))
		return
	}
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	err := d.Decode(&dataStruct)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Error in processing request."))
		ErrorLogger.Println(err)
		return
	}
	InfoLogger.Println("From IP - ", r.RemoteAddr)
	InfoLogger.Println("Shebang - ", dataStruct.Shebang)
	InfoLogger.Println("TimeExec - ", dataStruct.TimeExec)
	InfoLogger.Println("Interpreter - ", dataStruct.Interpreter)
	InfoLogger.Println("ID - ", dataStruct.ID)
	InfoLogger.Println("ExecCommand - ", dataStruct.ExecCommand)
	if Validate(dataStruct) == false {
		MsgErr := "INVALID VALIDATE!"
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(MsgErr))
		WarningLogger.Println(MsgErr)
		return
	}
	go Native(dataStruct)
	w.Write([]byte("OK"))
}

func IsUrl(str string) bool {
	u, err := url.Parse(str)
	return err == nil && u.Scheme != "" && u.Host != ""
}

func TestAfterStart() bool {
	InfoLogger.Println("Test after start - starting.")
	file, err := os.Create(WKSetings.FileExecute)
	if err != nil {
		tmp := "Unable to create exetuable file. Path: " + WKSetings.FileExecute + ". Shutdown webhook..."
		fmt.Println(tmp)
		ErrorLogger.Fatalln(tmp, err)
		return false
	}
	file.WriteString("TEST STRING")
	file.Close()
	err = os.Chmod(WKSetings.FileExecute, 0700)
	if err != nil {
		ErrorLogger.Println(err)
		return false
	}
	err = os.Remove(WKSetings.FileExecute)
	if err != nil {
		ErrorLogger.Println(err)
		return false
	}
	if WKSetings.URLServer == "None" {
		fmt.Println("Args URL not used. Exit.")
		ErrorLogger.Fatal("Args URL not used. Exit.")
	}
	if IsUrl(WKSetings.URLServer) == false {
		fmt.Println("Error validate URL - ", WKSetings.URLServer)
		ErrorLogger.Fatal("Error validate URL - ", WKSetings.URLServer)
	}
	if IsExistsURL(WKSetings.URLServer) == false {
		fmt.Println("URL is not exist - ", WKSetings.URLServer)
		os.Exit(1)
	}
	return true
}

func main() {
	PortPtr := flag.Int("Port", 7342, "Webhook port.")
	URLPtr := flag.String("URL", "None", "URL send result.")
	LogPtr := flag.String("Log", "/var/log/webhook.execute.log", "Log path.")
	ExecFile := flag.String("ExecFile", "/tmp/webhook.execute", "Path to execute file.")
	flag.Parse()
	if WKSetings.LogPath == *LogPtr {
		fmt.Println("Args -Log not use. Used default path: ", WKSetings.LogPath)
	}
	WKSetings.Version = "v0.0.6"
	WKSetings.LogPath = *LogPtr
	WKSetings.URLServer = *URLPtr
	WKSetings.FileExecute = *ExecFile
	WKSetings.Port = *PortPtr
	LogFunc()

	if TestAfterStart() {
		mux := http.NewServeMux()
		mux.HandleFunc("/execute", ExecuteCommand)
		mux.HandleFunc("/healtcheak", HealtCheak)
		ServerAddress := ":" + fmt.Sprint(WKSetings.Port)
		InfoLogger.Println("Startup on ", ServerAddress)
		fmt.Println("Startup on ", ServerAddress)
		error := http.ListenAndServe(ServerAddress, mux)
		ErrorLogger.Println(error)
	} else {
		fmt.Println("Error test after start.")
		ErrorLogger.Fatalln("Error test after start.")
	}
}
