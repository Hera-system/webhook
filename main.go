package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
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

var Version string = "v0.0.4"
var URLServer string = "None"
var LogPath string = "/var/log/webhook.executor.log"

func LogFunc() {
	file, err := os.OpenFile(LogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
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
	w.Write([]byte(Version))
	return
}

func saveToFile(dataStruct CMD, fileExecute string) bool {
	file, err := os.Create(fileExecute)
	if err != nil {
		ErrorLogger.Println("Unable to create file:", err)
		os.Exit(1)
		return false
	}
	file.WriteString(dataStruct.Shebang + "\n")
	file.WriteString(dataStruct.ExecCommand)
	file.Close()
	err = os.Chmod(fileExecute, 0700)
	if err != nil {
		ErrorLogger.Println(err)
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
	resp, err := http.Post(URLServer, "application/json", bytes.NewBuffer(json_data))
	if err != nil {
		ErrorLogger.Println("An Error Occured ", err)
	}
	if resp.StatusCode == 200 {
		return true
	}
	ErrorLogger.Println("Status code != 200. Status code is ", resp.StatusCode)
	ErrorLogger.Println("Error ID - ", response.ID)
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
	var fileExecute string = "/tmp/webhook.execute"
	if saveToFile(dataStruct, fileExecute) {
		var timeExecute = time.Duration(dataStruct.TimeExec)
		var stdout, stderr []byte
		var errStdout, errStderr error
		cmd := exec.Command(dataStruct.Interpreter, fileExecute)
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
		err := os.Remove(fileExecute)
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

func main() {
	PortPtr := flag.Int("Port", 7342, "Webhook port.")
	URLPtr := flag.String("URL", "None", "URL send result.")
	LogPtr := flag.String("Log", LogPath, "Log path.")
	flag.Parse()
	if LogPath == *LogPtr {
		fmt.Println("Args -Log not use. Used default path: ", LogPath)
	}
	LogPath = *LogPtr
	LogFunc()
	if *URLPtr == "None" {
		fmt.Println("Args URL not used. Exit.")
		ErrorLogger.Fatal("Args URL not used. Exit.")
	}
	URLServer = *URLPtr
	mux := http.NewServeMux()
	mux.HandleFunc("/execute", ExecuteCommand)
	mux.HandleFunc("/healtcheak", HealtCheak)
	ServerAddress := ":" + fmt.Sprint(*PortPtr)
	InfoLogger.Println("Startup on ", ServerAddress)
	fmt.Println("Startup on ", ServerAddress)
	error := http.ListenAndServe(ServerAddress, mux)
	ErrorLogger.Println(error)
}
