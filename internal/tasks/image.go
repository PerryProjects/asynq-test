package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

// ImageProcessor implements asynq.Handler (struct-based handler).
type ImageProcessor struct{}

// NewImageResizeTask creates a new image:resize task.
func NewImageResizeTask(url string, width, height int) (*asynq.Task, error) {
	payload, err := json.Marshal(ImagePayload{
		URL:    url,
		Width:  width,
		Height: height,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(
		TypeImageResize,
		payload,
		asynq.Timeout(30*time.Second),
		asynq.MaxRetry(2),
		asynq.Retention(1*time.Hour),
	), nil
}

// ProcessTask handles image:resize tasks — demonstrates struct handler + ResultWriter.
func (p *ImageProcessor) ProcessTask(ctx context.Context, t *asynq.Task) error {
	var payload ImagePayload
	if err := json.Unmarshal(t.Payload(), &payload); err != nil {
		return fmt.Errorf("failed to unmarshal image payload: %v: %w", err, asynq.SkipRetry)
	}

	taskID, _ := asynq.GetTaskID(ctx)
	log.Printf("[image:resize] TaskID=%s → Resizing %s to %dx%d",
		taskID, payload.URL, payload.Width, payload.Height)

	// Simulate image processing with context check
	select {
	case <-time.After(2 * time.Second):
	case <-ctx.Done():
		return fmt.Errorf("image processing cancelled: %w", ctx.Err())
	}

	// Write result via ResultWriter
	result := ImageResult{
		OriginalURL: payload.URL,
		ResizedURL:  fmt.Sprintf("%s?w=%d&h=%d", payload.URL, payload.Width, payload.Height),
		Width:       payload.Width,
		Height:      payload.Height,
	}
	resultJSON, _ := json.Marshal(result)
	if _, err := t.ResultWriter().Write(resultJSON); err != nil {
		log.Printf("[image:resize] TaskID=%s → Failed to write result: %v", taskID, err)
	}

	log.Printf("[image:resize] TaskID=%s → Resize complete: %s", taskID, result.ResizedURL)
	return nil
}
