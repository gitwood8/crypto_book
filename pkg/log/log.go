package log

import (
	"log"
	"os"
)

var (
	// infoLogger  = log.New(os.Stdout, "INFO: ", log.LstdFlags|log.Lshortfile)
	infoLogger  = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger = log.New(os.Stderr, "ERROR: ", log.LstdFlags|log.Lshortfile)
	debugLogger = log.New(os.Stdout, "DEBUG: ", log.LstdFlags|log.Lshortfile)
	warnLogger  = log.New(os.Stdout, "WARN: ", log.LstdFlags|log.Lshortfile)
	// InfoLogger  = log.New(file, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
)

func Info(v ...any) {
	infoLogger.Println(v...)
}
func Infof(format string, v ...any) {
	infoLogger.Printf(format, v...)
}

func Error(v ...any) {
	errorLogger.Println(v...)
}

func Errorf(format string, v ...any) {
	errorLogger.Printf(format, v...)
}

func Warn(v ...any) {
	warnLogger.Println(v...)
}

func Warnf(format string, v ...any) {
	warnLogger.Printf(format, v...)
}

func Debug(v ...any) {
	debugLogger.Println(v...)
}
func Debugf(format string, v ...any) {
	debugLogger.Printf(format, v...)
}
