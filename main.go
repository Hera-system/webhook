package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/hera-system/webhook/internal/execute"
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
	if utils.Validate(dataStruct) == false {
		MsgErr := "INVALID VALIDATE!"
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(MsgErr))
		log.Warn.Println(MsgErr)
		return
	}
	go execute.Native(dataStruct)
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
