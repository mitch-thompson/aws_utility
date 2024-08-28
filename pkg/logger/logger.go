package logger

import (
	"log"
	"os"
	"strings"
)

const (
	DebugLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	LevelEnvVariable = "LOG_LEVEL"
)

var CurrentLogLevel int

func Init() {
	levelStr := os.Getenv(LevelEnvVariable)
	switch strings.ToLower(levelStr) {
	case "debug":
		CurrentLogLevel = DebugLevel
	case "info":
		CurrentLogLevel = InfoLevel
	case "warn":
		CurrentLogLevel = WarnLevel
	case "error":
		CurrentLogLevel = ErrorLevel
	default:
		CurrentLogLevel = InfoLevel
	}
}

func Debug(v ...interface{}) {
	if CurrentLogLevel <= DebugLevel {
		log.Println(v...)
	}
}

func Info(v ...interface{}) {
	if CurrentLogLevel <= InfoLevel {
		log.Println(v...)
	}
}

func Warn(v ...interface{}) {
	if CurrentLogLevel <= WarnLevel {
		log.Println(v...)

	}
}

func Error(v ...interface{}) {
	if CurrentLogLevel <= ErrorLevel {
		log.Println(v...)
	}
}
