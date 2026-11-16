package persist

import (
	"os"
	"path/filepath"
	"spored/internal/base"
	"spored/internal/cparser"
	"spored/internal/event"
	message_event "spored/internal/event/message"
	"strings"
	"testing"
	"time"
)

type responseCapture struct {
	*base.Base
	messages chan string
}

func (c *responseCapture) OnMessage(ctx message_event.IMessageContext) {
	c.messages <- ctx.Message()
}

func TestRetrieveResponsePreservesBody(t *testing.T) {
	event.Init()
	storage := New()
	storage.dataDirectory = t.TempDir()
	storage.Open()
	defer storage.Close()
	capture := &responseCapture{Base: base.New(), messages: make(chan string, 1)}
	event.Subscribe(capture)
	defer event.Unsubscribe(capture)

	const command = "dev.sporeos.SPORE.state.retrieve"
	tests := []struct {
		name string
		body string
	}{
		{"empty", ""},
		{"quoted characters", "a \"quote\" \\path\tvalue\r"},
		{"multiline", "first line\nsecond line"},
		{"boundary newlines", "\nfirst line\n"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(storage.dataDirectory, "test.node.config.persist")
			if err := os.WriteFile(path, []byte(tt.body), 0600); err != nil {
				t.Fatal(err)
			}
			event.Dispatch().Request.Notify(message_event.Request(
				"state.retrieve key=config ~h1", "test.node", command, "dev.sporeos.SPORE", "h1"))

			var raw string
			select {
			case raw = <-capture.messages:
			case <-time.After(time.Second):
				t.Fatal("timed out waiting for retrieve response")
			}
			if strings.Contains(tt.body, "\n") {
				if !strings.HasPrefix(raw, "~h1:"+command+" ok body=<<\n") || !strings.HasSuffix(raw, "\n>>") {
					t.Fatalf("multiline body is not framed as a block: %q", raw)
				}
			}

			parsed, ok := cparser.ParseAndValidate(raw)
			if !ok {
				t.Fatalf("response could not be parsed: %q", raw)
			}
			if got, exists := parsed.Args["body"]; !exists || got != tt.body {
				t.Errorf("body = %q (present %t), want %q", got, exists, tt.body)
			}
		})
	}
}
