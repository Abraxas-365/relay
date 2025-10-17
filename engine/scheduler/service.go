package scheduler

import (
	"context"
	"time"

	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
)

type ScheduleService struct {
	scheduleRepo engine.WorkflowScheduleRepository
	workflowRepo engine.WorkflowRepository
	cronParser   cron.Parser
}

func NewScheduleService(
	scheduleRepo engine.WorkflowScheduleRepository,
	workflowRepo engine.WorkflowRepository,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo: scheduleRepo,
		workflowRepo: workflowRepo,
		cronParser:   cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
	}
}

// CreateCronSchedule creates a cron-based schedule
func (s *ScheduleService) CreateCronSchedule(
	ctx context.Context,
	tenantID kernel.TenantID,
	workflowID kernel.WorkflowID,
	cronExpression string,
	timezone string,
) (*engine.WorkflowSchedule, error) {
	// Validate workflow exists
	workflow, err := s.workflowRepo.FindByID(ctx, workflowID)
	if err != nil {
		return nil, engine.ErrWorkflowNotFound().
			WithDetail("workflow_id", workflowID.String())
	}

	if workflow.TenantID != tenantID {
		return nil, engine.ErrWorkflowNotFound().
			WithDetail("workflow_id", workflowID.String()).
			WithDetail("reason", "workflow does not belong to tenant")
	}

	// Validate cron expression
	_, err = s.cronParser.Parse(cronExpression)
	if err != nil {
		return nil, engine.ErrInvalidCronExpression().
			WithDetail("cron_expression", cronExpression).
			WithCause(err)
	}

	// Check if too many schedules exist
	count, err := s.scheduleRepo.CountByWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if count >= 10 { // Max 10 schedules per workflow
		return nil, engine.ErrTooManySchedules().
			WithDetail("workflow_id", workflowID.String()).
			WithDetail("current_count", count)
	}

	// Calculate first run
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	cronSchedule, _ := s.cronParser.Parse(cronExpression)
	nextRun := cronSchedule.Next(time.Now().In(loc))

	schedule := &engine.WorkflowSchedule{
		ID:             uuid.New().String(),
		TenantID:       tenantID,
		WorkflowID:     workflowID,
		ScheduleType:   engine.ScheduleTypeCron,
		CronExpression: &cronExpression,
		IsActive:       true,
		NextRunAt:      &nextRun,
		Timezone:       timezone,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.scheduleRepo.Save(ctx, *schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

// CreateIntervalSchedule creates an interval-based schedule
func (s *ScheduleService) CreateIntervalSchedule(
	ctx context.Context,
	tenantID kernel.TenantID,
	workflowID kernel.WorkflowID,
	intervalSeconds int,
) (*engine.WorkflowSchedule, error) {
	// Validate workflow exists
	workflow, err := s.workflowRepo.FindByID(ctx, workflowID)
	if err != nil {
		return nil, engine.ErrWorkflowNotFound().
			WithDetail("workflow_id", workflowID.String())
	}

	if workflow.TenantID != tenantID {
		return nil, engine.ErrWorkflowNotFound().
			WithDetail("workflow_id", workflowID.String()).
			WithDetail("reason", "workflow does not belong to tenant")
	}

	// Validate interval
	if intervalSeconds < 60 {
		return nil, engine.ErrInvalidInterval().
			WithDetail("interval_seconds", intervalSeconds).
			WithDetail("reason", "minimum interval is 60 seconds")
	}

	if intervalSeconds > 86400*7 { // Max 7 days
		return nil, engine.ErrInvalidInterval().
			WithDetail("interval_seconds", intervalSeconds).
			WithDetail("reason", "maximum interval is 7 days")
	}

	// Check if too many schedules exist
	count, err := s.scheduleRepo.CountByWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if count >= 10 {
		return nil, engine.ErrTooManySchedules().
			WithDetail("workflow_id", workflowID.String()).
			WithDetail("current_count", count)
	}

	// Calculate first run
	nextRun := time.Now().Add(time.Duration(intervalSeconds) * time.Second)

	schedule := &engine.WorkflowSchedule{
		ID:              uuid.New().String(),
		TenantID:        tenantID,
		WorkflowID:      workflowID,
		ScheduleType:    engine.ScheduleTypeInterval,
		IntervalSeconds: &intervalSeconds,
		IsActive:        true,
		NextRunAt:       &nextRun,
		Timezone:        "UTC",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	if err := s.scheduleRepo.Save(ctx, *schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

// CreateOnceSchedule creates a one-time schedule
func (s *ScheduleService) CreateOnceSchedule(
	ctx context.Context,
	tenantID kernel.TenantID,
	workflowID kernel.WorkflowID,
	scheduledAt time.Time,
) (*engine.WorkflowSchedule, error) {
	// Validate workflow exists
	workflow, err := s.workflowRepo.FindByID(ctx, workflowID)
	if err != nil {
		return nil, engine.ErrWorkflowNotFound().
			WithDetail("workflow_id", workflowID.String())
	}

	if workflow.TenantID != tenantID {
		return nil, engine.ErrWorkflowNotFound().
			WithDetail("workflow_id", workflowID.String()).
			WithDetail("reason", "workflow does not belong to tenant")
	}

	// Validate scheduled time is in the future
	if scheduledAt.Before(time.Now()) {
		return nil, engine.ErrScheduleInPast().
			WithDetail("scheduled_at", scheduledAt).
			WithDetail("current_time", time.Now())
	}

	schedule := &engine.WorkflowSchedule{
		ID:           uuid.New().String(),
		TenantID:     tenantID,
		WorkflowID:   workflowID,
		ScheduleType: engine.ScheduleTypeOnce,
		ScheduledAt:  &scheduledAt,
		IsActive:     true,
		NextRunAt:    &scheduledAt,
		Timezone:     "UTC",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	if err := s.scheduleRepo.Save(ctx, *schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

// UpdateSchedule updates an existing schedule
func (s *ScheduleService) UpdateSchedule(
	ctx context.Context,
	scheduleID string,
	tenantID kernel.TenantID,
	updateFn func(*engine.WorkflowSchedule) error,
) (*engine.WorkflowSchedule, error) {
	// Get existing schedule
	schedule, err := s.scheduleRepo.FindByID(ctx, scheduleID)
	if err != nil {
		return nil, engine.ErrScheduleNotFound().
			WithDetail("schedule_id", scheduleID)
	}

	// Verify tenant ownership
	if schedule.TenantID != tenantID {
		return nil, engine.ErrScheduleNotFound().
			WithDetail("schedule_id", scheduleID).
			WithDetail("reason", "schedule does not belong to tenant")
	}

	// Apply update
	if err := updateFn(schedule); err != nil {
		return nil, err
	}

	// Recalculate next run if needed
	if schedule.IsActive {
		nextRun, err := s.calculateNextRun(schedule, time.Now())
		if err != nil {
			return nil, engine.ErrInvalidScheduleConfig().
				WithDetail("schedule_id", scheduleID).
				WithCause(err)
		}
		schedule.NextRunAt = nextRun
	}

	// Save changes
	if err := s.scheduleRepo.Update(ctx, *schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

// ActivateSchedule activates a schedule
func (s *ScheduleService) ActivateSchedule(
	ctx context.Context,
	scheduleID string,
	tenantID kernel.TenantID,
) error {
	_, err := s.UpdateSchedule(ctx, scheduleID, tenantID, func(schedule *engine.WorkflowSchedule) error {
		schedule.IsActive = true
		return nil
	})
	return err
}

// DeactivateSchedule deactivates a schedule
func (s *ScheduleService) DeactivateSchedule(
	ctx context.Context,
	scheduleID string,
	tenantID kernel.TenantID,
) error {
	_, err := s.UpdateSchedule(ctx, scheduleID, tenantID, func(schedule *engine.WorkflowSchedule) error {
		schedule.IsActive = false
		schedule.NextRunAt = nil
		return nil
	})
	return err
}

// DeleteSchedule deletes a schedule
func (s *ScheduleService) DeleteSchedule(
	ctx context.Context,
	scheduleID string,
	tenantID kernel.TenantID,
) error {
	// Verify ownership before deleting
	schedule, err := s.scheduleRepo.FindByID(ctx, scheduleID)
	if err != nil {
		return engine.ErrScheduleNotFound().
			WithDetail("schedule_id", scheduleID)
	}

	if schedule.TenantID != tenantID {
		return engine.ErrScheduleNotFound().
			WithDetail("schedule_id", scheduleID).
			WithDetail("reason", "schedule does not belong to tenant")
	}

	return s.scheduleRepo.Delete(ctx, scheduleID)
}

// calculateNextRun calculates the next execution time
func (s *ScheduleService) calculateNextRun(schedule *engine.WorkflowSchedule, after time.Time) (*time.Time, error) {
	switch schedule.ScheduleType {
	case engine.ScheduleTypeCron:
		if schedule.CronExpression == nil {
			return nil, engine.ErrInvalidScheduleConfig().
				WithDetail("reason", "cron expression is nil")
		}

		cronSchedule, err := s.cronParser.Parse(*schedule.CronExpression)
		if err != nil {
			return nil, engine.ErrInvalidCronExpression().
				WithDetail("cron_expression", *schedule.CronExpression).
				WithCause(err)
		}

		loc, err := time.LoadLocation(schedule.Timezone)
		if err != nil {
			loc = time.UTC
		}

		next := cronSchedule.Next(after.In(loc))
		return &next, nil

	case engine.ScheduleTypeInterval:
		if schedule.IntervalSeconds == nil {
			return nil, engine.ErrInvalidScheduleConfig().
				WithDetail("reason", "interval_seconds is nil")
		}

		interval := time.Duration(*schedule.IntervalSeconds) * time.Second
		next := after.Add(interval)
		return &next, nil

	case engine.ScheduleTypeOnce:
		return nil, nil // One-time schedules don't repeat

	default:
		return nil, engine.ErrInvalidScheduleConfig().
			WithDetail("schedule_type", string(schedule.ScheduleType))
	}
}

