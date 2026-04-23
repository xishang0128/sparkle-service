package log

import (
	"log"
	"os"
	"path/filepath"
)

func currentLogPath() string {
	return filepath.Join(os.TempDir(), "sparkle-service.log")
}

func previousLogPath() string {
	return filepath.Join(os.TempDir(), "sparkle-service.previous.log")
}

func rotateLogFiles() error {
	currentPath := currentLogPath()
	previousPath := previousLogPath()

	if err := os.Remove(previousPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	if err := os.Rename(currentPath, previousPath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func InitLogging() (*os.File, error) {
	if err := rotateLogFiles(); err != nil {
		return nil, err
	}

	logFile, err := os.OpenFile(currentLogPath(), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
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
