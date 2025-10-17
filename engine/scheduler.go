package engine

import (
	"github.com/Abraxas-365/relay/pkg/kernel"
	"time"
)

type WorkflowSchedule struct {
	ID         string            `db:"id" json:"id"`
	TenantID   kernel.TenantID   `db:"tenant_id" json:"tenant_id"`
	WorkflowID kernel.WorkflowID `db:"workflow_id" json:"workflow_id"`

	// Schedule config
	ScheduleType    ScheduleType `db:"schedule_type" json:"schedule_type"`
	CronExpression  *string      `db:"cron_expression" json:"cron_expression,omitempty"`
	IntervalSeconds *int         `db:"interval_seconds" json:"interval_seconds,omitempty"`
	ScheduledAt     *time.Time   `db:"scheduled_at" json:"scheduled_at,omitempty"`

	// Status
	IsActive  bool       `db:"is_active" json:"is_active"`
	LastRunAt *time.Time `db:"last_run_at" json:"last_run_at,omitempty"`
	NextRunAt *time.Time `db:"next_run_at" json:"next_run_at,omitempty"`
	RunCount  int        `db:"run_count" json:"run_count"`

	// Metadata
	Timezone string         `db:"timezone" json:"timezone"`
	Metadata map[string]any `db:"metadata" json:"metadata,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type ScheduleType string

const (
	ScheduleTypeCron     ScheduleType = "cron"     // Cron expression
	ScheduleTypeInterval ScheduleType = "interval" // Fixed interval
	ScheduleTypeOnce     ScheduleType = "once"     // One-time execution
)

// Domain methods
func (s *WorkflowSchedule) IsValid() bool {
	switch s.ScheduleType {
	case ScheduleTypeCron:
		return s.CronExpression != nil && *s.CronExpression != ""
	case ScheduleTypeInterval:
		return s.IntervalSeconds != nil && *s.IntervalSeconds > 0
	case ScheduleTypeOnce:
		return s.ScheduledAt != nil
	default:
		return false
	}
}

func (s *WorkflowSchedule) ShouldRun(now time.Time) bool {
	if !s.IsActive {
		return false
	}
	if s.NextRunAt == nil {
		return false
	}
	return now.After(*s.NextRunAt) || now.Equal(*s.NextRunAt)
}

func (s *WorkflowSchedule) MarkExecuted(now time.Time) {
	s.LastRunAt = &now
	s.RunCount++

	// For one-time schedules, deactivate after execution
	if s.ScheduleType == ScheduleTypeOnce {
		s.IsActive = false
		s.NextRunAt = nil
	}
}
