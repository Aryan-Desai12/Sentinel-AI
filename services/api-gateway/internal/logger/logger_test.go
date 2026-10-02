package logger

import (
	"bytes"
	"encoding/json"
	"log"
	"strings"
	"testing"
)

func TestLoggerInfo(t *testing.T) {
	var output bytes.Buffer

	l := &Logger{
		service: "api-gateway",
		logger:  log.New(&output, "", 0),
	}

	l.Info("test_event", map[string]any{
		"node_id": "node-123",
	})

	line := strings.TrimSpace(output.String())

	var entry map[string]any

	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		t.Fatalf("expected valid JSON, got error: %v", err)
	}

	if entry["level"] != "info" {
		t.Fatalf("expected level info, got %v", entry["level"])
	}

	if entry["service"] != "api-gateway" {
		t.Fatalf("expected service api-gateway, got %v", entry["service"])
	}

	if entry["event"] != "test_event" {
		t.Fatalf("expected event test_event, got %v", entry["event"])
	}

	if entry["node_id"] != "node-123" {
		t.Fatalf("expected node_id node-123, got %v", entry["node_id"])
	}

	if entry["timestamp"] == nil {
		t.Fatal("expected timestamp")
	}
}
