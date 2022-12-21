package execute

import (
	"os"
	"os/exec"
	"time"

	"github.com/hera-system/webhook/internal/log"
	"github.com/hera-system/webhook/internal/vars"
	"github.com/hera-system/webhook/utils"
)

func Native(dataStruct vars.CMD) string {
	log.Info.Println("Native starting")
	if utils.SaveToFile(dataStruct) {
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
			stdout, errStdout = utils.CopyAndCapture(os.Stdout, stdoutIn)
			done <- cmd.Wait()
		}()
		select {
		case <-time.After(timeExecute * time.Second):
			if err := cmd.Process.Kill(); err != nil {
				tmp := "Failed to kill process. "
				log.Error.Println(tmp, err)
				utils.SendResult(tmp, dataStruct, true, "", "")
				return (tmp)
			}
			tmp := "Process killed as timeout reached"
			log.Warn.Println(tmp)
			utils.SendResult(tmp, dataStruct, true, "", "")
			return (tmp)
		case err := <-done:
			if err != nil {
				log.Error.Println("Process finished with error = ", err)
				log.Error.Println("ID - ", dataStruct.ID)
				utils.SendResult("Error, check args and logs.", dataStruct, true, "", err.Error())
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
			utils.SendResult(TmpMsg, dataStruct, true, "", "")
			return TmpMsg
		}
		outStr, errStr := string(stdout), string(stderr)
		log.Info.Println("Stdout: ", outStr)
		log.Info.Println("Stderr: ", errStr)
		utils.SendResult("OK", dataStruct, false, outStr, errStr)
		return "output"
	}
	return "error"
}
