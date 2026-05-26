package ndjson

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestWriterCreateAndWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.ndjson")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	defer w.Close()

	ev := NewEvent("test_event", "test_tool")
	if err := w.Write(ev); err != nil {
		t.Fatalf("Write: %v", err)
	}
	w.Close()

	data, _ := os.ReadFile(path)
	var decoded Event
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if decoded.Event != "test_event" {
		t.Fatalf("expected test_event, got %s", decoded.Event)
	}
	if decoded.Tool != "test_tool" {
		t.Fatalf("expected test_tool, got %s", decoded.Tool)
	}
	if decoded.Timestamp == "" {
		t.Fatal("timestamp empty")
	}
}

func TestWriterConcurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.ndjson")
	w, err := NewWriter(path)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	defer w.Close()

	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			_ = w.Write(NewEvent("concurrent", "tool"))
			done <- struct{}{}
		}()
	}
	for i := 0; i < 10; i++ {
		<-done
	}
}
