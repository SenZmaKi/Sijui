package shared

import (
	"log"
	"os"
	"time"
)

var (
	logFilePath = "../sijui.log"
	_           = SetUpLog(logFilePath)
)

func SetUpLog(logFilePath string) bool {
	file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Println("Error opening the log file:", err)
		return false
	} else {
		log.SetOutput(file)
		return true
	}
}

func LogError(errMessage string, err error) {
	clearLogFile()
	log.Printf("[!] %v: %v", errMessage, err)
}

func LogInfo(info string) {
	clearLogFile()
	log.Println("[+]", info)
}

func clearLogFile() {
	fileInfo, _ := os.Stat(logFilePath)
				// 1 GB	
	if fileInfo.Size() > 1000000 * 1000 {
		if err := os.Truncate(logFilePath, 0); err != nil {
			LogError("Error while clearing log file", err)
		}
	}
} 

func RetryOnErrorWrapper(callee func() (interface{}, error), errMessage string) interface{} {
	for {
		if result, err := callee(); err == nil {
			return result
		} else {
			LogError(errMessage, err)
			time.Sleep(5 * time.Second)
			LogInfo("Retrying.. .")
		}
	}
}
