package tasks

import "testing"

func TestTaskTypeConstants(t *testing.T) {
	t.Parallel()
	t.Cleanup(func() {})

	tests := []struct {
		name  string
		value string
	}{
		{name: "email", value: TypeEmailDeliver},
		{name: "image", value: TypeImageResize},
		{name: "report", value: TypeReportGenerate},
		{name: "webhook", value: TypeWebhookSend},
		{name: "notification send", value: TypeNotificationSend},
		{name: "notification batch", value: TypeNotificationBatch},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.value == "" {
				t.Fatal("task type constant is empty")
			}
		})
	}
}
