package logger

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type Logger struct {
	service string
	logger  *log.Logger
}

func New(service string) *Logger {
	return &Logger{
		service: service,
		logger:  log.New(os.Stdout, "", 0),
	}
}

func (l *Logger) Info(event string, fields map[string]any) {
	l.log("info", event, fields)
}

func (l *Logger) Error(event string, fields map[string]any) {
	l.log("error", event, fields)
}

func (l *Logger) log(level string, event string, fields map[string]any) {
	entry := map[string]any{
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		"level":     level,
		"service":   l.service,
		"event":     event,
	}

	for key, value := range fields {
		entry[key] = value
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	l.logger.Println(string(data))
}

func (l *Logger) Warn(event string, fields map[string]any) {
	l.log("warn", event, fields)
}
