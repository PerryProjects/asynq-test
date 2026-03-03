package tasks

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hibiken/asynq"
)

// NewReportGenerateTask creates a new report:generate task.
// Demonstrates: Deadline, ProcessIn (delayed), and periodic scheduling via cron.
func NewReportGenerateTask(reportType, startDate, endDate string) (*asynq.Task, error) {
	payload, err := marshalPayload(ReportPayload{
		ReportType: reportType,
		StartDate:  startDate,
		EndDate:    endDate,
	})
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(
		TypeReportGenerate,
		payload,
		asynq.MaxRetry(2),
		asynq.Deadline(time.Now().Add(10*time.Minute)),
		asynq.Retention(4*time.Hour),
	), nil
}

// HandleReportGenerate processes report:generate tasks.
// Simulates a long-running job with periodic context checks.
func HandleReportGenerate(ctx context.Context, t *asynq.Task) error {
	var p ReportPayload
	if err := unmarshalPayload(t.Payload(), &p); err != nil {
		return fmt.Errorf("failed to unmarshal report payload: %v: %w", err, asynq.SkipRetry)
	}

	taskID, _ := asynq.GetTaskID(ctx)
	log.Printf("[report:generate] TaskID=%s → Generating %s report (%s to %s)",
		taskID, p.ReportType, p.StartDate, p.EndDate)

	// Simulate long-running report generation (5 steps)
	for step := 1; step <= 5; step++ {
		select {
		case <-ctx.Done():
			log.Printf("[report:generate] TaskID=%s → Cancelled at step %d: %v",
				taskID, step, ctx.Err())
			return ctx.Err()
		case <-time.After(1 * time.Second):
			log.Printf("[report:generate] TaskID=%s → Step %d/5 complete", taskID, step)
		}
	}

	// Write result
	result := fmt.Sprintf(`{"report_type":%q,"status":"completed","generated_at":%q}`,
		p.ReportType, time.Now().Format(time.RFC3339))
	if _, err := t.ResultWriter().Write([]byte(result)); err != nil {
		log.Printf("[report:generate] TaskID=%s → Failed to write result: %v", taskID, err)
	}

	log.Printf("[report:generate] TaskID=%s → Report generation complete", taskID)
	return nil
}
