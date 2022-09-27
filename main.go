package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"time"
)

type CMD struct {
	ExecCommand *string `json:"ExecCommand"`
	Shebang     *string `json:"Shebang"`
	TimeExec    *int    `json:"TimeExec"`
	Token       *string `json:"Token"`
	Interpreter *string `json:"Interpreter"`
	ID          *string `json:"ID"`
}

type dataResponse struct {
	Error   *bool   `json:"Error"`
	Stdout  *string `json:"Stdout"`
	Stderr  *string `json:"Stderr"`
	ID      *string `json:"ID"`
	Token   *string `json:"Token"`
	Message *string `json:"Message"`
}

var (
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
)

var TOKEN string = "VeryStrongString"

func init() {
	file, err := os.OpenFile("/tmp/webhook.executor.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
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
	w.Write([]byte("OK"))
}

func saveToFile(dataStruct *CMD, fileExecute string) bool {
	file, err := os.Create(fileExecute)
	if err != nil {
		ErrorLogger.Println("Unable to create file:", err)
		os.Exit(1)
		return false
	}
	file.WriteString(*dataStruct.Shebang + "\n")
	file.WriteString(*dataStruct.ExecCommand)
	file.Close()
	err = os.Chmod(fileExecute, 0700)
	if err != nil {
		ErrorLogger.Println(err)
		return false
	}
	return true
}

func sendResult(data string, dataStruct *CMD, Error bool, Stdout string, Stderr string) bool {
	response := dataResponse{ID: *&dataStruct.ID, Error: &Error, Token: *&dataStruct.Token, Message: &data, Stderr: &Stderr, Stdout: &Stdout}
	json_data, err := json.Marshal(response)
	if err != nil {
		ErrorLogger.Println(err)
	}
	resp, err := http.Post(os.Getenv("URL_SERVER"), "application/json", bytes.NewBuffer(json_data))
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

func Native(dataStruct *CMD) string {
	var fileExecute string = "/tmp/webhook.execute"
	if saveToFile(dataStruct, fileExecute) {
		var timeExecute = time.Duration(*dataStruct.TimeExec)
		var stdout, stderr []byte
		var errStdout, errStderr error
		cmd := exec.Command(*dataStruct.Interpreter, fileExecute)
		stdoutIn, _ := cmd.StdoutPipe()
		//stderrIn, _ := cmd.StderrPipe()
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
				ErrorLogger.Println("failed to kill process: ", err)
				sendResult("Failed to kill proccess.", dataStruct, true, "", "")
				return ("Failed to kill proccess.")
			}
			WarningLogger.Println("Process killed as timeout reached")
			sendResult("Process killed as timeout reached.", dataStruct, true, "", "")
			return ("Process killed as timeout reached")
		case err := <-done:
			if err != nil {
				ErrorLogger.Println("Process finished with error = ", err)
				ErrorLogger.Println("ID - ", *dataStruct.ID)
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
		InfoLogger.Println("\nout:\n%s\nerr:\n%s\n", outStr, errStr)
		sendResult("OK", dataStruct, false, outStr, errStr)
		return "output"
	}
	return "error"
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
	InfoLogger.Println("Shebang - ", *dataStruct.Shebang)
	InfoLogger.Println("TimeExec - ", *dataStruct.TimeExec)
	InfoLogger.Println("Interpreter - ", *dataStruct.Interpreter)
	InfoLogger.Println("ID - ", *dataStruct.ID)
	InfoLogger.Println("ExecCommand - ", *dataStruct.ExecCommand)
	if *dataStruct.Token != TOKEN {
		MsgErr := "INVALID TOKEN!"
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(MsgErr))
		WarningLogger.Println(MsgErr)
		return
	}
	go Native(&dataStruct)
	w.Write([]byte("OK"))
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/execute", ExecuteCommand)
	mux.HandleFunc("/healtcheak", HealtCheak)
	ServerAddress := ":" + os.Getenv("PORT")
	error := http.ListenAndServe(ServerAddress, mux)
	InfoLogger.Println("Startup on ", ServerAddress)
	ErrorLogger.Println(error)
}
