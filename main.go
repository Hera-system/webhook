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
		_, err := w.Write([]byte("Method not allowed."))
		if err != nil {
			log.Error.Println("Error file write")
		}
		return
	}
	_, err := w.Write([]byte(vars.Version))
	if err != nil {
		log.Error.Println("Error file write")
	}
}

func ExecuteCommand(w http.ResponseWriter, r *http.Request) {
	var dataStruct vars.CMD
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_, err := w.Write([]byte("Method not allowed."))
		if err != nil {
			log.Error.Println("Error file write")
		}
		return
	}
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	err := d.Decode(&dataStruct)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_, err = w.Write([]byte("Error in processing request."))
		if err != nil {
			log.Error.Println("Error file write")
		}
		log.Error.Println(err)
		return
	}
	log.Info.Println("From IP - ", r.RemoteAddr)
	log.Info.Println("TimeExec - ", dataStruct.TimeExec)
	log.Info.Println("Interpreter - ", dataStruct.Interpreter)
	log.Info.Println("ID - ", dataStruct.ID)
	log.Info.Println("ExecCommand - ", dataStruct.ExecCommand)
	if utils.Valid(dataStruct) {
		_, err = w.Write([]byte("OK"))
		if err != nil {
			log.Error.Println("Error file write")
		}
		go execute.Native(dataStruct)
		return
	}
	MsgErr := "INVALID VALIDATE!"
	fmt.Println(MsgErr)
	w.WriteHeader(http.StatusForbidden)
	_, err = w.Write([]byte(MsgErr))
	if err != nil {
		log.Error.Println("Error file write")
	}
	log.Warn.Println(MsgErr)
}

func main() {
	ConfFile := flag.String("config", "config.json", "Path to conf file.")
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
