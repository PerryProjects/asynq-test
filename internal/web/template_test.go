package web

import (
	"strings"
	"testing"
)

func TestIndexHTMLContainsCoreSections(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	tests := []struct {
		name string
		want string
	}{
		{name: "dashboard title", want: "Asynq Multi-Pod Prototype"},
		{name: "queues section", want: "Queues"},
		{name: "workers section", want: "Connected Workers"},
		{name: "enqueue form", want: "Enqueue Task"},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if !strings.Contains(indexHTML, tc.want) {
				t.Fatalf("indexHTML missing %q", tc.want)
			}
		})
	}
}
