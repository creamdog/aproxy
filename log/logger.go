package log

import (
	"log"
	"fmt"
)

const (  // iota is reset to 0
	DEBUG = iota
	INFO = iota
	WARN = iota
	ERROR = iota 
	FATAL = iota
)

var levelStrings map[int]string = map[int]string{
	0 : "DEBUG",
	1 : "INFO",
	2 : "WARN",
	3 : "ERROR",
	4 : "FATAL",
}

func Log(level int, v ...interface{}) {
	if level >= FATAL {
		log.Fatalf("[%s] %v", levelStrings[level], fmt.Sprintf("%q", v))
	} 
	log.Printf("[%s] %v", levelStrings[level], fmt.Sprintf("%q", v))
}

func Debugf(format string, v ...interface{}) {
	Log(DEBUG, fmt.Sprintf(format, v))
}

func Infof(format string, v ...interface{}) {
	Log(INFO, fmt.Sprintf(format, v))
}

func Warnf(format string, v ...interface{}) {
	Log(WARN, fmt.Sprintf(format, v))
}

func Errorf(format string, v ...interface{}) {
	Log(ERROR, fmt.Sprintf(format, v))
}


func Print(v ...interface{}) {
	Log(INFO, v...)
}

func Printf(format string, v ...interface{}) {
	Log(INFO, fmt.Sprintf(format, v))
}

func Fatalf(format string, v ...interface{}) {
	Log(FATAL, fmt.Sprintf(format, v))
}

func Fatal(v ...interface{}) {
	Log(FATAL, v...)
}