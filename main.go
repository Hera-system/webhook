package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/hera-system/webhook/internal/log"
	"github.com/hera-system/webhook/internal/vars"
	"github.com/hera-system/webhook/utils"
)

func HealtCheak(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method not allowed."))
		return
	}
	w.Write([]byte(vars.WKSetings.Version))
	return
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

func sendResult(data string, dataStruct vars.CMD, Error bool, Stdout string, Stderr string) bool {
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
	if utils.IsExistsURL(vars.WKSetings.URLServer) {
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

func Native(dataStruct vars.CMD) string {
	if saveToFile(dataStruct) {
		var timeExecute = time.Duration(dataStruct.TimeExec)
		var stdout, stderr []byte
		var errStdout, errStderr error
		cmd := exec.Command(dataStruct.Interpreter, vars.WKSetings.FileExecute)
		stdoutIn, _ := cmd.StdoutPipe()
		if err := cmd.Start(); err != nil {
			log.Error.Println(err)
		}
		done := make(chan error, 1)
		go func() {
			stdout, errStdout = copyAndCapture(os.Stdout, stdoutIn)
			done <- cmd.Wait()
		}()
		select {
		case <-time.After(timeExecute * time.Second):
			if err := cmd.Process.Kill(); err != nil {
				tmp := "Failed to kill process. "
				log.Error.Println(tmp, err)
				sendResult(tmp, dataStruct, true, "", "")
				return (tmp)
			}
			tmp := "Process killed as timeout reached"
			log.Warn.Println(tmp)
			sendResult(tmp, dataStruct, true, "", "")
			return (tmp)
		case err := <-done:
			if err != nil {
				log.Error.Println("Process finished with error = ", err)
				log.Error.Println("ID - ", dataStruct.ID)
				sendResult("Error, check args and logs.", dataStruct, true, "", "")
				return ("Error, check args and logs.")
			}
			log.Info.Println("Process finished successfully")
		}
		err := os.Remove(vars.WKSetings.FileExecute)
		if err != nil {
			log.Error.Println(err)
		}
		if errStdout != nil || errStderr != nil {
			log.Error.Println("failed to capture stdout or stderr")
			sendResult("Failed to capture stdout or stderr.", dataStruct, true, "", "")
			return "Failed to capture stdout or stderr."
		}
		outStr, errStr := string(stdout), string(stderr)
		log.Info.Println("Stdout: ", outStr)
		log.Info.Println("Stderr: ", errStr)
		sendResult("OK", dataStruct, false, outStr, errStr)
		return "output"
	}
	return "error"
}

func saveToFile(dataStruct vars.CMD) bool {
	file, err := os.Create(vars.WKSetings.FileExecute)
	if err != nil {
		tmp := "Unable to create exetuable file. Path: " + vars.WKSetings.FileExecute + ". Shutdown webhook..."
		fmt.Println(tmp)
		sendResult(tmp, dataStruct, true, "", "")
		log.Error.Fatalln(tmp, err)
		return false
	}
	file.WriteString(dataStruct.Shebang + "\n")
	file.WriteString(dataStruct.ExecCommand)
	file.Close()
	err = os.Chmod(vars.WKSetings.FileExecute, 0700)
	if err != nil {
		log.Error.Println(err)
		return false
	}
	return true
}

func Validate(dataStruct vars.CMD) bool {
	var Secret string
	if dataStruct.Token != vars.WKSetings.SecretToken {
		return false
	}
	if utils.IsExistsURL(vars.WKSetings.HTTPSectretURL) == false {
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

func ExecuteCommand(w http.ResponseWriter, r *http.Request) {
	var dataStruct vars.CMD
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
		log.Error.Println(err)
		return
	}
	log.Info.Println("From IP - ", r.RemoteAddr)
	log.Info.Println("Shebang - ", dataStruct.Shebang)
	log.Info.Println("TimeExec - ", dataStruct.TimeExec)
	log.Info.Println("Interpreter - ", dataStruct.Interpreter)
	log.Info.Println("ID - ", dataStruct.ID)
	log.Info.Println("ExecCommand - ", dataStruct.ExecCommand)
	if Validate(dataStruct) == false {
		MsgErr := "INVALID VALIDATE!"
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(MsgErr))
		log.Warn.Println(MsgErr)
		return
	}
	go Native(dataStruct)
	w.Write([]byte("OK"))
}

func main() {
	ConfFile := flag.String("conf", "config.json", "Path to conf file.")
	flag.Parse()
	file, err := os.Open(*ConfFile)
	if err != nil {
		fmt.Println(err)
		fmt.Println("Error open conf file -", *ConfFile)
		os.Exit(1)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&vars.WKSetings)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	vars.WKSetings.Version = "v0.0.9"
	log.LogPath = vars.WKSetings.LogPath
	log.LogFunc()

	if utils.TestAfterStart() {
		mux := http.NewServeMux()
		mux.HandleFunc("/execute", ExecuteCommand)
		mux.HandleFunc("/healtcheak", HealtCheak)
		ServerAddress := ":" + fmt.Sprint(vars.WKSetings.Port)
		log.Info.Println("Startup on ", ServerAddress)
		fmt.Println("Startup on ", ServerAddress)
		error := http.ListenAndServe(ServerAddress, mux)
		log.Error.Println(error)
	} else {
		fmt.Println("Error test after start.")
		log.Error.Fatalln("Error test after start.")
	}
}
