package ndjson

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Writer struct {
	mu   sync.Mutex
	file *os.File
	enc  *json.Encoder
	path string
}

func NewWriter(path string) (*Writer, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("ndjson: mkdir %s: %w", dir, err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0640)
	if err != nil {
		return nil, fmt.Errorf("ndjson: open %s: %w", path, err)
	}
	return &Writer{file: f, enc: json.NewEncoder(f), path: path}, nil
}

func (w *Writer) Write(v any) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.enc.Encode(v)
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

func (w *Writer) Path() string { return w.path }

type Event struct {
	Timestamp string         `json:"ts"`
	Event     string         `json:"event"`
	SprintID  string         `json:"sprint_id,omitempty"`
	AgentID   string         `json:"agent_id,omitempty"`
	Tool      string         `json:"tool,omitempty"`
	DurationMs int64         `json:"duration_ms,omitempty"`
	Extra     map[string]any `json:"extra,omitempty"`
}

func NewEvent(event, tool string) Event {
	return Event{
		Timestamp: time.Now().Format(time.RFC3339),
		Event:     event,
		Tool:      tool,
	}
}
