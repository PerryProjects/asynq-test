package tasks

import (
	"sync"
	"testing"
)

var payloadFormatTestMu sync.Mutex

func helperWithPayloadFormat(t *testing.T, format string) {
	t.Helper()
	payloadFormatTestMu.Lock()
	t.Cleanup(payloadFormatTestMu.Unlock)

	if err := SetPayloadFormat(format); err != nil {
		t.Fatalf("SetPayloadFormat(%q) error: %v", format, err)
	}
	t.Cleanup(func() {
		if err := SetPayloadFormat(PayloadFormatJSON); err != nil {
			t.Fatalf("restore payload format error: %v", err)
		}
	})
}
