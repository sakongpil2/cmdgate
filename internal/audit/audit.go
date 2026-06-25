package audit

import (
	"fmt"
	"os"
	"time"
)

// LogEntry represents one audit log record.
type LogEntry struct {
	Timestamp time.Time
	User      string
	Action    string
	CommandID string
	Command   string
	Result    string
	Reason    string
}

// Writer appends structured audit log lines to a file.
type Writer struct {
	file *os.File
}

// NewWriter opens the audit log file for appending.
func NewWriter(path string) (*Writer, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return nil, err
	}
	return &Writer{file: f}, nil
}

// Write formats and appends a LogEntry to the audit log.
func (w *Writer) Write(entry LogEntry) error {
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	}
	_, err := fmt.Fprintf(w.file, "%s user=%s action=%s command_id=%q command=%q result=%s reason=%q\n",
		entry.Timestamp.Format(time.RFC3339),
		entry.User,
		entry.Action,
		entry.CommandID,
		entry.Command,
		entry.Result,
		entry.Reason,
	)
	return err
}

// Close closes the underlying file.
func (w *Writer) Close() error {
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}
