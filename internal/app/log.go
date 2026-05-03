package app

import (
	"fmt"
	"os"
)

type LogLevel int

const (
	LogLevelNormal  LogLevel = 0
	LogLevelVerbose LogLevel = 1
)

var currentLogLevel LogLevel = LogLevelNormal

func SetLogLevel(level LogLevel) {
	currentLogLevel = level
}

func Log(format string, args ...any) {
	writeLog(fmt.Sprintf(format, args...))
}

func LogVerbose(format string, args ...any) {
	if currentLogLevel < LogLevelVerbose {
		return
	}
	writeLog("[verbose] " + fmt.Sprintf(format, args...))
}

func LogError(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	writeLog("Error: " + msg)
}

func writeLog(msg string) {
	fmt.Fprintln(os.Stdout, msg)
}
