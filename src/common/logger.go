package common

import (
	"log"
	"os"
	"fmt"
)

var (
	Debug       *RoomServerLogger
	Info        *RoomServerLogger
	Error       *RoomServerLogger
	Warn        *RoomServerLogger
	LoggerLevel = INFO
)

const (
	DEBUG = 0
	INFO  = 1
	ANY   = 99
)

type RoomServerLogger struct {
	log.Logger
	Level int
}

func (rsl *RoomServerLogger) Output(calldepth int, s string) error {
	fmt.Println(rsl.Level)
	if rsl.Level == ANY || rsl.Level == LoggerLevel {
		return rsl.Logger.Output(calldepth, s)
	}
	return nil
}

func getLogger(logger *log.Logger, level int) *RoomServerLogger {
	return &RoomServerLogger{*logger, level}
}

func init() {
	Debug = getLogger(log.New(os.Stdout, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile), DEBUG)
	Info = getLogger(log.New(os.Stdout, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile), INFO)
	Error = getLogger(log.New(os.Stderr, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile), ANY)
	Warn = getLogger(log.New(os.Stdout, "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile), ANY)
}
