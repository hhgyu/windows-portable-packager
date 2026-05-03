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

func Logf(format string, args ...any) {
	writeLog(fmt.Sprintf(format, args...))
}

func Log(msg string) {
	writeLog(msg)
}

func LogVerbosef(format string, args ...any) {
	if currentLogLevel < LogLevelVerbose {
		return
	}
	writeLog("[verbose] " + fmt.Sprintf(format, args...))
}

func LogVerbose(msg string) {
	if currentLogLevel < LogLevelVerbose {
		return
	}
	writeLog("[verbose] " + msg)
}

func LogErrorf(format string, args ...any) {
	writeLog("Error: " + fmt.Sprintf(format, args...))
}

func LogError(msg string) {
	writeLog("Error: " + msg)
}

func writeLog(msg string) {
	fmt.Fprintln(os.Stdout, msg)
}
