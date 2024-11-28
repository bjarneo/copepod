package logger

import (
	"fmt"
	"os"
	"time"
)

// Logger handles logging to both console and file
type Logger struct {
	file *os.File
}

// New creates a new logger instance
func New(filename string) (*Logger, error) {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &Logger{file: file}, nil
}

// Info logs an informational message
func (l *Logger) Info(message string) error {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	logMessage := fmt.Sprintf("[%s] INFO: %s\n", timestamp, message)
	fmt.Print(message + "\n")
	_, err := l.file.WriteString(logMessage)
	return err
}

// Error logs an error message
func (l *Logger) Error(message string, err error) error {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	errStr := ""
	if err != nil {
		errStr = err.Error()
	}
	logMessage := fmt.Sprintf("[%s] ERROR: %s\n%s\n", timestamp, message, errStr)
	fmt.Printf("ERROR: %s\n", message)
	if err != nil {
		fmt.Printf("Error details: %s\n", err)
	}
	_, writeErr := l.file.WriteString(logMessage)
	return writeErr
}

// Fatal logs a fatal error message and exits the program
func (l *Logger) Fatal(err error) {
	timestamp := time.Now().UTC().Format(time.RFC3339)
	logMessage := fmt.Sprintf("[%s] FATAL: %s\n", timestamp, err.Error())
	fmt.Printf("FATAL: %s\n", err)
	l.file.WriteString(logMessage)
	l.Close()
	os.Exit(1)
}

// Close closes the log file
func (l *Logger) Close() error {
	return l.file.Close()
} 