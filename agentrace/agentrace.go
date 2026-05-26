package agentrace

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/nfsarch33/helixon-common/ndjson"
)

var (
	globalWriter *ndjson.Writer
	mu           sync.Mutex
	enabled      bool
)

type Config struct {
	LogPath  string
	SprintID string
	AgentID  string
}

func Init(cfg Config) error {
	if os.Getenv("AGENTRACE_ENABLED") != "1" {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	enabled = true

	path := cfg.LogPath
	if path == "" {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, "logs", "runx", "agentrace-mcp.ndjson")
	}
	w, err := ndjson.NewWriter(path)
	if err != nil {
		return err
	}
	globalWriter = w
	return nil
}

func Emit(event ndjson.Event) {
	mu.Lock()
	defer mu.Unlock()
	if !enabled || globalWriter == nil {
		return
	}
	_ = globalWriter.Write(event)
}

func Close() {
	mu.Lock()
	defer mu.Unlock()
	if globalWriter != nil {
		globalWriter.Close()
		globalWriter = nil
	}
	enabled = false
}

func Enabled() bool {
	mu.Lock()
	defer mu.Unlock()
	return enabled
}
