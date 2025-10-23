package log

import (
	"log"
	"os"
	"path/filepath"
)

func InitLogging() (*os.File, error) {
	tmpDir := os.TempDir()
	logPath := filepath.Join(tmpDir, "sparkle-service.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}
	os.Stdout = logFile
	os.Stderr = logFile

	log.SetOutput(logFile)
	return logFile, nil
}

func Fatal(v ...any) {
	log.Fatal(v...)
}

func Print(v ...any) {
	log.Print(v...)
}

func Printf(format string, v ...any) {
	log.Printf(format, v...)
}

func Println(v ...any) {
	log.Println(v...)
}
