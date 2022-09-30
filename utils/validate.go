package utils

import (
	"io"
	"net/http"
)

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
