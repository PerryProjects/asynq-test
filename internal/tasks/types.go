package tasks

// Task type constants.
const (
	TypeEmailDeliver      = "email:deliver"
	TypeImageResize       = "image:resize"
	TypeReportGenerate    = "report:generate"
	TypeWebhookSend       = "webhook:send"
	TypeNotificationSend  = "notification:send"
	TypeNotificationBatch = "notification:batch"
)

// EmailPayload is the payload for email:deliver tasks.
type EmailPayload struct {
	To      string `json:"to"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// ImagePayload is the payload for image:resize tasks.
type ImagePayload struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

// ImageResult is the result written by image:resize handler.
type ImageResult struct {
	OriginalURL string `json:"original_url"`
	ResizedURL  string `json:"resized_url"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

// ReportPayload is the payload for report:generate tasks.
type ReportPayload struct {
	ReportType string `json:"report_type"`
	StartDate  string `json:"start_date"`
	EndDate    string `json:"end_date"`
}

// WebhookPayload is the payload for webhook:send tasks.
type WebhookPayload struct {
	URL          string `json:"url"`
	Method       string `json:"method"`
	SimulateCode int    `json:"simulate_code"` // 200, 4xx, 5xx for demo
}

// NotificationPayload is the payload for notification:send tasks.
type NotificationPayload struct {
	UserID  int    `json:"user_id"`
	Message string `json:"message"`
	Channel string `json:"channel"` // email, sms, push
}

// NotificationBatchPayload is the aggregated batch payload.
type NotificationBatchPayload struct {
	UserIDs  []int    `json:"user_ids"`
	Messages []string `json:"messages"`
	Count    int      `json:"count"`
	Group    string   `json:"group"`
}
