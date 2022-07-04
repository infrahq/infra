package logging

import (
	"gopkg.in/natefinch/lumberjack.v2"
)

type FileLogger struct {
	logger
}

func UseFileLogger(filepath string) {
	writer := &lumberjack.Logger{
		Filename:   filepath,
		MaxSize:    10, // megabytes
		MaxBackups: 7,
		MaxAge:     28, // days
	}

	L = newLogger(writer)
}
