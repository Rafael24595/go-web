package log

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	MESSAGE string = "MESSAGE"
	WARNING string = "WARNING"
	ERROR   string = "ERROR"
)

type Log interface {
	Custom(string, string)
	Custome(string, error)
	Customf(string, string, ...any)
	Message(string)
	Messagef(string, ...any)
	Warning(string)
	Warningf(string, ...any)
	Error(error)
	Errors(string)
	Errorf(string, ...any)
	Write([]byte) (int, error)
}

type defaultLogger struct {
	mu sync.Mutex
}

func DefaultLogger() Log {
	return &defaultLogger{}
}

func (l *defaultLogger) Custom(category string, message string) {
	l.custom(category, message)
}

func (l *defaultLogger) Custome(category string, err error) {
	l.custom(category, err.Error())
}

func (l *defaultLogger) Customf(category string, format string, args ...any) {
	l.custom(category, fmt.Sprintf(format, args...))
}

func (l *defaultLogger) custom(category string, message string) {
	category = strings.ToUpper(category)
	l.record(category, message)
}

func (l *defaultLogger) Message(message string) {
	l.record(MESSAGE, message)
}

func (l *defaultLogger) Messagef(format string, args ...any) {
	l.record(MESSAGE, fmt.Sprintf(format, args...))
}

func (l *defaultLogger) Warning(message string) {
	l.record(WARNING, message)
}

func (l *defaultLogger) Warningf(format string, args ...any) {
	l.record(WARNING, fmt.Sprintf(format, args...))
}

func (l *defaultLogger) Error(err error) {
	l.record(ERROR, err.Error())
}

func (l *defaultLogger) Errors(message string) {
	l.record(ERROR, message)
}

func (l *defaultLogger) Errorf(format string, args ...any) {
	l.record(ERROR, fmt.Sprintf(format, args...))
}

func (l *defaultLogger) Write(slice []byte) (n int, err error) {
	l.Warningf("%s", bytes.TrimSpace(slice))
	return len(slice), nil
}

func (l *defaultLogger) record(category string, message string) {
	timestamp := time.Now().UnixMilli()
	go l.print(timestamp, category, message)
}

func (l *defaultLogger) print(timestamp int64, category string, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	formatted := fmt.Sprintf("(GO-WEB) - %s - [%s]: %s", FormatMilliseconds(timestamp), category, message)
	println(formatted)
}

func FormatMilliseconds(timestamp int64) string {
	if timestamp == 0 {
		return "N/A"
	}
	seconds := timestamp / 1000
	time := time.Unix(seconds, 0)
	return time.Format("2006-01-02 15:04:05")
}
