package tasks

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	taskspb "github.com/asynq-test/internal/tasks/pb"
	"google.golang.org/protobuf/proto"
)

const (
	PayloadFormatJSON  = "json"
	PayloadFormatProto = "proto"
)

var (
	payloadFormat   = PayloadFormatJSON
	payloadFormatMu sync.RWMutex
)

// SetPayloadFormat configures payload serialization format used for task payloads.
// Supported values: json, proto.
func SetPayloadFormat(format string) error {
	f := strings.ToLower(strings.TrimSpace(format))
	if f == "" {
		f = PayloadFormatJSON
	}
	if f != PayloadFormatJSON && f != PayloadFormatProto {
		return fmt.Errorf("invalid payload format %q (expected json or proto)", format)
	}
	payloadFormatMu.Lock()
	payloadFormat = f
	payloadFormatMu.Unlock()
	return nil
}

func getPayloadFormat() string {
	payloadFormatMu.RLock()
	defer payloadFormatMu.RUnlock()
	return payloadFormat
}

func marshalPayload(v any) ([]byte, error) {
	switch getPayloadFormat() {
	case PayloadFormatProto:
		msg, err := toProtoMessage(v)
		if err != nil {
			return nil, err
		}
		return proto.Marshal(msg)
	default:
		return json.Marshal(v)
	}
}

func unmarshalPayload(data []byte, out any) error {
	switch getPayloadFormat() {
	case PayloadFormatProto:
		return fromProtoMessage(data, out)
	default:
		return json.Unmarshal(data, out)
	}
}

func toProtoMessage(v any) (proto.Message, error) {
	switch payload := v.(type) {
	case EmailPayload:
		return &taskspb.EmailPayload{
			To:      payload.To,
			Subject: payload.Subject,
			Body:    payload.Body,
		}, nil
	case ImagePayload:
		return &taskspb.ImagePayload{
			Url:    payload.URL,
			Width:  int32(payload.Width),
			Height: int32(payload.Height),
		}, nil
	case ReportPayload:
		return &taskspb.ReportPayload{
			ReportType: payload.ReportType,
			StartDate:  payload.StartDate,
			EndDate:    payload.EndDate,
		}, nil
	case WebhookPayload:
		return &taskspb.WebhookPayload{
			Url:          payload.URL,
			Method:       payload.Method,
			SimulateCode: int32(payload.SimulateCode),
		}, nil
	case NotificationPayload:
		return &taskspb.NotificationPayload{
			UserId:  int32(payload.UserID),
			Message: payload.Message,
			Channel: payload.Channel,
		}, nil
	case NotificationBatchPayload:
		userIDs := make([]int32, 0, len(payload.UserIDs))
		for _, id := range payload.UserIDs {
			userIDs = append(userIDs, int32(id))
		}
		return &taskspb.NotificationBatchPayload{
			UserIds:  userIDs,
			Messages: payload.Messages,
			Count:    int32(payload.Count),
			Group:    payload.Group,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported proto payload type %T", v)
	}
}

func fromProtoMessage(data []byte, out any) error {
	switch payload := out.(type) {
	case *EmailPayload:
		var msg taskspb.EmailPayload
		if err := proto.Unmarshal(data, &msg); err != nil {
			return err
		}
		payload.To = msg.GetTo()
		payload.Subject = msg.GetSubject()
		payload.Body = msg.GetBody()
		return nil
	case *ImagePayload:
		var msg taskspb.ImagePayload
		if err := proto.Unmarshal(data, &msg); err != nil {
			return err
		}
		payload.URL = msg.GetUrl()
		payload.Width = int(msg.GetWidth())
		payload.Height = int(msg.GetHeight())
		return nil
	case *ReportPayload:
		var msg taskspb.ReportPayload
		if err := proto.Unmarshal(data, &msg); err != nil {
			return err
		}
		payload.ReportType = msg.GetReportType()
		payload.StartDate = msg.GetStartDate()
		payload.EndDate = msg.GetEndDate()
		return nil
	case *WebhookPayload:
		var msg taskspb.WebhookPayload
		if err := proto.Unmarshal(data, &msg); err != nil {
			return err
		}
		payload.URL = msg.GetUrl()
		payload.Method = msg.GetMethod()
		payload.SimulateCode = int(msg.GetSimulateCode())
		return nil
	case *NotificationPayload:
		var msg taskspb.NotificationPayload
		if err := proto.Unmarshal(data, &msg); err != nil {
			return err
		}
		payload.UserID = int(msg.GetUserId())
		payload.Message = msg.GetMessage()
		payload.Channel = msg.GetChannel()
		return nil
	case *NotificationBatchPayload:
		var msg taskspb.NotificationBatchPayload
		if err := proto.Unmarshal(data, &msg); err != nil {
			return err
		}
		userIDs := make([]int, 0, len(msg.GetUserIds()))
		for _, id := range msg.GetUserIds() {
			userIDs = append(userIDs, int(id))
		}
		payload.UserIDs = userIDs
		payload.Messages = msg.GetMessages()
		payload.Count = int(msg.GetCount())
		payload.Group = msg.GetGroup()
		return nil
	default:
		return fmt.Errorf("unsupported proto output type %T", out)
	}
}
