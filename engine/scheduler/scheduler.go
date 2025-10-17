package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/engine/triggerhandler"
	"github.com/robfig/cron/v3"
)

type WorkflowScheduler struct {
	scheduleRepo   engine.WorkflowScheduleRepository
	triggerHandler *triggerhandler.TriggerHandler
	cronParser     cron.Parser
	stopChan       chan struct{}
	running        bool
}

func NewWorkflowScheduler(
	scheduleRepo engine.WorkflowScheduleRepository,
	triggerHandler *triggerhandler.TriggerHandler,
) *WorkflowScheduler {
	return &WorkflowScheduler{
		scheduleRepo:   scheduleRepo,
		triggerHandler: triggerHandler,
		cronParser:     cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
		stopChan:       make(chan struct{}),
	}
}

// Start starts the scheduler
func (s *WorkflowScheduler) Start(ctx context.Context) {
	if s.running {
		log.Println("⚠️  Scheduler already running")
		return
	}

	s.running = true
	log.Println("⏰ Starting workflow scheduler...")

	// Run immediately on start
	go s.processDueSchedules(ctx)

	// Then run every minute
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("⏹️  Scheduler stopped (context done)")
			return
		case <-s.stopChan:
			log.Println("⏹️  Scheduler stopped")
			return
		case <-ticker.C:
			s.processDueSchedules(ctx)
		}
	}
}

// Stop stops the scheduler
func (s *WorkflowScheduler) Stop() {
	if !s.running {
		return
	}
	close(s.stopChan)
	s.running = false
}

// processDueSchedules checks and executes due schedules
func (s *WorkflowScheduler) processDueSchedules(ctx context.Context) {
	now := time.Now()

	// Get all active schedules that are due
	schedules, err := s.scheduleRepo.FindDue(ctx, now)
	if err != nil {
		log.Printf("❌ Failed to fetch due schedules: %v", err)
		return
	}

	if len(schedules) == 0 {
		return
	}

	log.Printf("⏰ Found %d due schedule(s)", len(schedules))

	for _, schedule := range schedules {
		// Execute in goroutine to not block
		go s.executeSchedule(ctx, schedule)
	}
}

// executeSchedule executes a single schedule
func (s *WorkflowScheduler) executeSchedule(ctx context.Context, schedule *engine.WorkflowSchedule) {
	log.Printf("▶️  Executing schedule: %s (workflow: %s)", schedule.ID, schedule.WorkflowID)

	// Prepare trigger data
	triggerData := map[string]any{
		"schedule_id":    schedule.ID,
		"schedule_type":  schedule.ScheduleType,
		"execution_time": time.Now().Unix(),
		"run_count":      schedule.RunCount + 1,
	}

	if schedule.CronExpression != nil {
		triggerData["cron_expression"] = *schedule.CronExpression
	}
	if schedule.IntervalSeconds != nil {
		triggerData["interval_seconds"] = *schedule.IntervalSeconds
	}

	// Trigger workflow
	err := s.triggerHandler.HandleScheduleTrigger(
		ctx,
		schedule.TenantID,
		schedule.ID,
		triggerData,
	)

	if err != nil {
		log.Printf("❌ Failed to trigger workflow: %v", err)
		return
	}

	// Update schedule
	now := time.Now()
	schedule.MarkExecuted(now)

	// Calculate next run time
	nextRun, err := s.calculateNextRun(schedule, now)
	if err != nil {
		log.Printf("⚠️  Failed to calculate next run: %v", err)
	} else {
		schedule.NextRunAt = nextRun
	}

	// Save updated schedule
	if err := s.scheduleRepo.Update(ctx, *schedule); err != nil {
		log.Printf("❌ Failed to update schedule: %v", err)
	}

	log.Printf("✅ Schedule executed successfully: %s", schedule.ID)
}

// calculateNextRun calculates the next execution time
func (s *WorkflowScheduler) calculateNextRun(schedule *engine.WorkflowSchedule, after time.Time) (*time.Time, error) {
	switch schedule.ScheduleType {
	case engine.ScheduleTypeCron:
		return s.calculateCronNextRun(schedule, after)
	case engine.ScheduleTypeInterval:
		return s.calculateIntervalNextRun(schedule, after)
	case engine.ScheduleTypeOnce:
		return nil, nil // One-time schedules don't repeat
	default:
		return nil, fmt.Errorf("unknown schedule type: %s", schedule.ScheduleType)
	}
}

// calculateCronNextRun calculates next run for cron schedules
func (s *WorkflowScheduler) calculateCronNextRun(schedule *engine.WorkflowSchedule, after time.Time) (*time.Time, error) {
	if schedule.CronExpression == nil {
		return nil, fmt.Errorf("cron expression is nil")
	}

	cronSchedule, err := s.cronParser.Parse(*schedule.CronExpression)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	// Get timezone
	loc, err := time.LoadLocation(schedule.Timezone)
	if err != nil {
		loc = time.UTC
	}

	next := cronSchedule.Next(after.In(loc))
	return &next, nil
}

// calculateIntervalNextRun calculates next run for interval schedules
func (s *WorkflowScheduler) calculateIntervalNextRun(schedule *engine.WorkflowSchedule, after time.Time) (*time.Time, error) {
	if schedule.IntervalSeconds == nil {
		return nil, fmt.Errorf("interval_seconds is nil")
	}

	interval := time.Duration(*schedule.IntervalSeconds) * time.Second
	next := after.Add(interval)
	return &next, nil
}
