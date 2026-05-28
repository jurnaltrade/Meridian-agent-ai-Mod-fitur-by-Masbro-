package logger

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

var logDir = "./logs"

func init() {
	os.MkdirAll(logDir, 0o755)
}

func Log(category, message string) {
	ts := time.Now().Format(time.RFC3339)
	line := fmt.Sprintf("[%s] [%s] %s", ts, category, message)

	slog.Info(message, "category", category)

	dateStr := time.Now().Format("2006-01-02")
	logFile := filepath.Join(logDir, "agent-"+dateStr+".log")
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err == nil {
		f.WriteString(line + "\n")
		f.Close()
	}
}

func Error(category string, err error) {
	Log(category+"_error", err.Error())
}

func Warn(category, message string) {
	Log(category+"_warn", message)
}

type ActionLog struct {
	Timestamp  string `json:"timestamp"`
	Tool       string `json:"tool"`
	Args       any    `json:"args,omitempty"`
	Result     any    `json:"result,omitempty"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"duration_ms"`
	Success    bool   `json:"success"`
}

func LogAction(action ActionLog) {
	ts := time.Now().Format(time.RFC3339)
	action.Timestamp = ts

	status := "✓"
	if !action.Success {
		status = "✗"
	}
	dur := ""
	if action.DurationMs > 0 {
		dur = fmt.Sprintf(" (%dms)", action.DurationMs)
	}
	fmt.Printf("[%s] %s%s\n", action.Tool, status, dur)

	dateStr := time.Now().Format("2006-01-02")
	actionsFile := filepath.Join(logDir, "actions-"+dateStr+".jsonl")
	f, err := os.OpenFile(actionsFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	data, _ := json.Marshal(action)
	f.Write(append(data, '\n'))
}
