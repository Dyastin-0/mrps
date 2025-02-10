package config

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestWatch(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config_*.yaml")
	if err != nil {
		t.Fatalf("Failed to create temporary file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	initialConfig := `
domains:
  gitsense.dyastin.tech:
    routes:
      /api/v1:
        dest: "http://localhost:4000"
        rewrite:
          type: "regex"
          value: "^/api/v1/(.*)$"
          replace_val: "/$1"
      /:
        dest: "http://localhost:4001"
        rewrite: {}
    rate_limit:
      burst: 15
      rate: 10
      cooldown: 60000
`
	tmpFile.WriteString(initialConfig)
	tmpFile.Sync()

	logBuffer := new(LogBuffer)
	log.SetOutput(logBuffer)

	go Watch(context.Background(), tmpFile.Name())

	time.Sleep(100 * time.Millisecond)

	updatedConfig := `
domains:
  gitsense.dyastin.tech:
    routes:
      /api/v1:
        dest: "http://localhost:4000"
        rewrite:
          type: "regex"
          value: "^/api/v1/(.*)$"
          replace_val: "/$1"
      /:
        dest: "http://localhost:4001"
        rewrite: {}
    rate_limit:
      burst: 20  # Changed burst value
      rate: 15   # Changed rate value
      cooldown: 60000

misc:
  email: "test@mail.com"
`
	tmpFile.Truncate(0)
	tmpFile.Seek(0, 0)
	tmpFile.WriteString(updatedConfig)
	tmpFile.Sync()

	time.Sleep(500 * time.Millisecond)

	assert.Contains(t, logBuffer.String(), "Watching config file:")
	assert.Contains(t, logBuffer.String(), "Config file changed:")
	assert.Contains(t, logBuffer.String(), "Configuration reloaded successfully.")
}

type LogBuffer struct {
	logs string
}

func (l *LogBuffer) Write(p []byte) (n int, err error) {
	l.logs += string(p)
	return len(p), nil
}

func (l *LogBuffer) String() string {
	return l.logs
}
