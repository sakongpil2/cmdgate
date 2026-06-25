package audit

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestWriteLog(t *testing.T) {
	f := tempLogFile(t)
	w, err := NewWriter(f)
	if err != nil {
		t.Fatalf("new writer error: %v", err)
	}
	defer w.Close()
	if err := w.Write(LogEntry{User: "opsadm", Action: "run", CommandID: "x", Command: "y", Result: "success"}); err != nil {
		t.Fatalf("write error: %v", err)
	}

	line := readLine(t, f)
	if !strings.Contains(line, `user=opsadm`) {
		t.Errorf("missing user, got %q", line)
	}
	if !strings.Contains(line, `result=success`) {
		t.Errorf("missing result, got %q", line)
	}
}

func tempLogFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp("", "audit-*.log")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	return f.Name()
}

func readLine(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(strings.NewReader(string(b)))
	if !scanner.Scan() {
		t.Fatal("no line")
	}
	return scanner.Text()
}
