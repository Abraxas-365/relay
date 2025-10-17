package engineinfra

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/jmoiron/sqlx"
)

type PostgresScheduleRepository struct {
	db *sqlx.DB
}

var _ engine.WorkflowScheduleRepository = (*PostgresScheduleRepository)(nil)

func NewPostgresScheduleRepository(db *sqlx.DB) *PostgresScheduleRepository {
	return &PostgresScheduleRepository{db: db}
}

// ============================================================================
// CRUD Operations
// ============================================================================

// Save creates a new schedule
func (r *PostgresScheduleRepository) Save(ctx context.Context, schedule engine.WorkflowSchedule) error {
	query := `
        INSERT INTO workflow_schedules (
            id, tenant_id, workflow_id,
            schedule_type, cron_expression, interval_seconds, scheduled_at,
            is_active, last_run_at, next_run_at, run_count,
            timezone, metadata,
            created_at, updated_at
        ) VALUES (
            $1, $2, $3,
            $4, $5, $6, $7,
            $8, $9, $10, $11,
            $12, $13,
            $14, $15
        )
    `

	metadataJSON, err := json.Marshal(schedule.Metadata)
	if err != nil {
		return engine.ErrInvalidScheduleConfig().
			WithDetail("reason", "failed to marshal metadata").
			WithCause(err)
	}

	_, err = r.db.ExecContext(ctx, query,
		schedule.ID,
		schedule.TenantID,
		schedule.WorkflowID,
		schedule.ScheduleType,
		schedule.CronExpression,
		schedule.IntervalSeconds,
		schedule.ScheduledAt,
		schedule.IsActive,
		schedule.LastRunAt,
		schedule.NextRunAt,
		schedule.RunCount,
		schedule.Timezone,
		metadataJSON,
		schedule.CreatedAt,
		schedule.UpdatedAt,
	)

	if err != nil {
		return engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "save").
			WithCause(err)
	}

	return nil
}

// Update updates an existing schedule
func (r *PostgresScheduleRepository) Update(ctx context.Context, schedule engine.WorkflowSchedule) error {
	query := `
        UPDATE workflow_schedules
        SET 
            schedule_type = $1,
            cron_expression = $2,
            interval_seconds = $3,
            scheduled_at = $4,
            is_active = $5,
            last_run_at = $6,
            next_run_at = $7,
            run_count = $8,
            timezone = $9,
            metadata = $10,
            updated_at = $11
        WHERE id = $12
    `

	metadataJSON, err := json.Marshal(schedule.Metadata)
	if err != nil {
		return engine.ErrInvalidScheduleConfig().
			WithDetail("reason", "failed to marshal metadata").
			WithCause(err)
	}

	result, err := r.db.ExecContext(ctx, query,
		schedule.ScheduleType,
		schedule.CronExpression,
		schedule.IntervalSeconds,
		schedule.ScheduledAt,
		schedule.IsActive,
		schedule.LastRunAt,
		schedule.NextRunAt,
		schedule.RunCount,
		schedule.Timezone,
		metadataJSON,
		time.Now(),
		schedule.ID,
	)

	if err != nil {
		return engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "update").
			WithDetail("schedule_id", schedule.ID).
			WithCause(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "check_rows_affected").
			WithCause(err)
	}

	if rows == 0 {
		return engine.ErrScheduleNotFound().
			WithDetail("schedule_id", schedule.ID)
	}

	return nil
}

// FindByID finds a schedule by ID
func (r *PostgresScheduleRepository) FindByID(ctx context.Context, id string) (*engine.WorkflowSchedule, error) {
	query := `
        SELECT 
            id, tenant_id, workflow_id,
            schedule_type, cron_expression, interval_seconds, scheduled_at,
            is_active, last_run_at, next_run_at, run_count,
            timezone, metadata,
            created_at, updated_at
        FROM workflow_schedules
        WHERE id = $1
    `

	var schedule engine.WorkflowSchedule
	var metadataJSON []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&schedule.ID,
		&schedule.TenantID,
		&schedule.WorkflowID,
		&schedule.ScheduleType,
		&schedule.CronExpression,
		&schedule.IntervalSeconds,
		&schedule.ScheduledAt,
		&schedule.IsActive,
		&schedule.LastRunAt,
		&schedule.NextRunAt,
		&schedule.RunCount,
		&schedule.Timezone,
		&metadataJSON,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, engine.ErrScheduleNotFound().
			WithDetail("schedule_id", id)
	}

	if err != nil {
		return nil, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "find_by_id").
			WithDetail("schedule_id", id).
			WithCause(err)
	}

	// Unmarshal metadata
	if len(metadataJSON) > 0 && string(metadataJSON) != "null" {
		if err := json.Unmarshal(metadataJSON, &schedule.Metadata); err != nil {
			return nil, engine.ErrInvalidScheduleConfig().
				WithDetail("reason", "failed to unmarshal metadata").
				WithCause(err)
		}
	}

	return &schedule, nil
}

// Delete deletes a schedule
func (r *PostgresScheduleRepository) Delete(ctx context.Context, id string) error {
	query := `DELETE FROM workflow_schedules WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "delete").
			WithDetail("schedule_id", id).
			WithCause(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "check_rows_affected").
			WithCause(err)
	}

	if rows == 0 {
		return engine.ErrScheduleNotFound().
			WithDetail("schedule_id", id)
	}

	return nil
}

// ============================================================================
// Query Operations
// ============================================================================

// FindByWorkflow finds all schedules for a workflow
func (r *PostgresScheduleRepository) FindByWorkflow(
	ctx context.Context,
	workflowID kernel.WorkflowID,
) ([]*engine.WorkflowSchedule, error) {
	query := `
        SELECT 
            id, tenant_id, workflow_id,
            schedule_type, cron_expression, interval_seconds, scheduled_at,
            is_active, last_run_at, next_run_at, run_count,
            timezone, metadata,
            created_at, updated_at
        FROM workflow_schedules
        WHERE workflow_id = $1
        ORDER BY created_at DESC
    `

	rows, err := r.db.QueryContext(ctx, query, workflowID)
	if err != nil {
		return nil, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "find_by_workflow").
			WithDetail("workflow_id", workflowID.String()).
			WithCause(err)
	}
	defer rows.Close()

	schedules := []*engine.WorkflowSchedule{}
	for rows.Next() {
		schedule, err := r.scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "iterate_rows").
			WithCause(err)
	}

	return schedules, nil
}

// FindByTenant finds all schedules for a tenant
func (r *PostgresScheduleRepository) FindByTenant(
	ctx context.Context,
	tenantID kernel.TenantID,
) ([]*engine.WorkflowSchedule, error) {
	query := `
        SELECT 
            s.id, s.tenant_id, s.workflow_id,
            s.schedule_type, s.cron_expression, s.interval_seconds, s.scheduled_at,
            s.is_active, s.last_run_at, s.next_run_at, s.run_count,
            s.timezone, s.metadata,
            s.created_at, s.updated_at
        FROM workflow_schedules s
        WHERE s.tenant_id = $1
        ORDER BY s.created_at DESC
    `

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "find_by_tenant").
			WithDetail("tenant_id", tenantID.String()).
			WithCause(err)
	}
	defer rows.Close()

	schedules := []*engine.WorkflowSchedule{}
	for rows.Next() {
		schedule, err := r.scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "iterate_rows").
			WithCause(err)
	}

	return schedules, nil
}

// FindDue finds all schedules that are due for execution
func (r *PostgresScheduleRepository) FindDue(
	ctx context.Context,
	before time.Time,
) ([]*engine.WorkflowSchedule, error) {
	query := `
        SELECT 
            id, tenant_id, workflow_id,
            schedule_type, cron_expression, interval_seconds, scheduled_at,
            is_active, last_run_at, next_run_at, run_count,
            timezone, metadata,
            created_at, updated_at
        FROM workflow_schedules
        WHERE is_active = true
        AND next_run_at IS NOT NULL
        AND next_run_at <= $1
        ORDER BY next_run_at ASC
        LIMIT 100
    `

	rows, err := r.db.QueryContext(ctx, query, before)
	if err != nil {
		return nil, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "find_due").
			WithCause(err)
	}
	defer rows.Close()

	schedules := []*engine.WorkflowSchedule{}
	for rows.Next() {
		schedule, err := r.scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "iterate_rows").
			WithCause(err)
	}

	return schedules, nil
}

// FindActive finds all active schedules for a tenant
func (r *PostgresScheduleRepository) FindActive(
	ctx context.Context,
	tenantID kernel.TenantID,
) ([]*engine.WorkflowSchedule, error) {
	query := `
        SELECT 
            id, tenant_id, workflow_id,
            schedule_type, cron_expression, interval_seconds, scheduled_at,
            is_active, last_run_at, next_run_at, run_count,
            timezone, metadata,
            created_at, updated_at
        FROM workflow_schedules
        WHERE tenant_id = $1
        AND is_active = true
        ORDER BY next_run_at ASC
    `

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "find_active").
			WithDetail("tenant_id", tenantID.String()).
			WithCause(err)
	}
	defer rows.Close()

	schedules := []*engine.WorkflowSchedule{}
	for rows.Next() {
		schedule, err := r.scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "iterate_rows").
			WithCause(err)
	}

	return schedules, nil
}

// FindByType finds all schedules of a specific type for a tenant
func (r *PostgresScheduleRepository) FindByType(
	ctx context.Context,
	tenantID kernel.TenantID,
	scheduleType engine.ScheduleType,
) ([]*engine.WorkflowSchedule, error) {
	query := `
        SELECT 
            id, tenant_id, workflow_id,
            schedule_type, cron_expression, interval_seconds, scheduled_at,
            is_active, last_run_at, next_run_at, run_count,
            timezone, metadata,
            created_at, updated_at
        FROM workflow_schedules
        WHERE tenant_id = $1
        AND schedule_type = $2
        ORDER BY created_at DESC
    `

	rows, err := r.db.QueryContext(ctx, query, tenantID, scheduleType)
	if err != nil {
		return nil, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "find_by_type").
			WithDetail("tenant_id", tenantID.String()).
			WithDetail("schedule_type", string(scheduleType)).
			WithCause(err)
	}
	defer rows.Close()

	schedules := []*engine.WorkflowSchedule{}
	for rows.Next() {
		schedule, err := r.scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "iterate_rows").
			WithCause(err)
	}

	return schedules, nil
}

// ============================================================================
// Bulk Operations
// ============================================================================

// BulkUpdateStatus updates the active status of multiple schedules
func (r *PostgresScheduleRepository) BulkUpdateStatus(
	ctx context.Context,
	ids []string,
	isActive bool,
) error {
	if len(ids) == 0 {
		return nil
	}

	query := `
        UPDATE workflow_schedules
        SET is_active = $1, updated_at = NOW()
        WHERE id = ANY($2)
    `

	_, err := r.db.ExecContext(ctx, query, isActive, ids)
	if err != nil {
		return engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "bulk_update_status").
			WithDetail("count", len(ids)).
			WithCause(err)
	}

	return nil
}

// BulkDelete deletes multiple schedules
func (r *PostgresScheduleRepository) BulkDelete(
	ctx context.Context,
	ids []string,
) error {
	if len(ids) == 0 {
		return nil
	}

	query := `DELETE FROM workflow_schedules WHERE id = ANY($1)`

	result, err := r.db.ExecContext(ctx, query, ids)
	if err != nil {
		return engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "bulk_delete").
			WithDetail("count", len(ids)).
			WithCause(err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "check_rows_affected").
			WithCause(err)
	}

	if rows == 0 {
		return engine.ErrScheduleNotFound()
	}

	return nil
}

// ============================================================================
// Statistics
// ============================================================================

// CountByTenant counts schedules for a tenant
func (r *PostgresScheduleRepository) CountByTenant(
	ctx context.Context,
	tenantID kernel.TenantID,
) (int, error) {
	query := `SELECT COUNT(*) FROM workflow_schedules WHERE tenant_id = $1`

	var count int
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "count_by_tenant").
			WithDetail("tenant_id", tenantID.String()).
			WithCause(err)
	}

	return count, nil
}

// CountByWorkflow counts schedules for a workflow
func (r *PostgresScheduleRepository) CountByWorkflow(
	ctx context.Context,
	workflowID kernel.WorkflowID,
) (int, error) {
	query := `SELECT COUNT(*) FROM workflow_schedules WHERE workflow_id = $1`

	var count int
	err := r.db.QueryRowContext(ctx, query, workflowID).Scan(&count)
	if err != nil {
		return 0, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "count_by_workflow").
			WithDetail("workflow_id", workflowID.String()).
			WithCause(err)
	}

	return count, nil
}

// CountActive counts active schedules for a tenant
func (r *PostgresScheduleRepository) CountActive(
	ctx context.Context,
	tenantID kernel.TenantID,
) (int, error) {
	query := `
        SELECT COUNT(*) 
        FROM workflow_schedules 
        WHERE tenant_id = $1 AND is_active = true
    `

	var count int
	err := r.db.QueryRowContext(ctx, query, tenantID).Scan(&count)
	if err != nil {
		return 0, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "count_active").
			WithDetail("tenant_id", tenantID.String()).
			WithCause(err)
	}

	return count, nil
}

// GetNextRunTimes gets the next N schedules to run
func (r *PostgresScheduleRepository) GetNextRunTimes(
	ctx context.Context,
	tenantID kernel.TenantID,
	limit int,
) ([]*engine.WorkflowSchedule, error) {
	query := `
        SELECT 
            id, tenant_id, workflow_id,
            schedule_type, cron_expression, interval_seconds, scheduled_at,
            is_active, last_run_at, next_run_at, run_count,
            timezone, metadata,
            created_at, updated_at
        FROM workflow_schedules
        WHERE tenant_id = $1
        AND is_active = true
        AND next_run_at IS NOT NULL
        ORDER BY next_run_at ASC
        LIMIT $2
    `

	rows, err := r.db.QueryContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "get_next_run_times").
			WithDetail("tenant_id", tenantID.String()).
			WithCause(err)
	}
	defer rows.Close()

	schedules := []*engine.WorkflowSchedule{}
	for rows.Next() {
		schedule, err := r.scanSchedule(rows)
		if err != nil {
			return nil, err
		}
		schedules = append(schedules, schedule)
	}

	if err := rows.Err(); err != nil {
		return nil, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "iterate_rows").
			WithCause(err)
	}

	return schedules, nil
}

// ============================================================================
// Helper Functions
// ============================================================================

// scanSchedule scans a single schedule from database rows
func (r *PostgresScheduleRepository) scanSchedule(scanner interface {
	Scan(dest ...interface{}) error
}) (*engine.WorkflowSchedule, error) {
	var schedule engine.WorkflowSchedule
	var metadataJSON []byte

	err := scanner.Scan(
		&schedule.ID,
		&schedule.TenantID,
		&schedule.WorkflowID,
		&schedule.ScheduleType,
		&schedule.CronExpression,
		&schedule.IntervalSeconds,
		&schedule.ScheduledAt,
		&schedule.IsActive,
		&schedule.LastRunAt,
		&schedule.NextRunAt,
		&schedule.RunCount,
		&schedule.Timezone,
		&metadataJSON,
		&schedule.CreatedAt,
		&schedule.UpdatedAt,
	)

	if err != nil {
		return nil, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "scan_schedule").
			WithCause(err)
	}

	// Unmarshal metadata
	if len(metadataJSON) > 0 && string(metadataJSON) != "null" {
		if err := json.Unmarshal(metadataJSON, &schedule.Metadata); err != nil {
			return nil, engine.ErrInvalidScheduleConfig().
				WithDetail("reason", "failed to unmarshal metadata").
				WithCause(err)
		}
	}

	return &schedule, nil
}

// ExistsForWorkflow checks if a schedule exists for a workflow
func (r *PostgresScheduleRepository) ExistsForWorkflow(
	ctx context.Context,
	workflowID kernel.WorkflowID,
	scheduleType engine.ScheduleType,
) (bool, error) {
	query := `
        SELECT EXISTS(
            SELECT 1 FROM workflow_schedules
            WHERE workflow_id = $1 AND schedule_type = $2
        )
    `

	var exists bool
	err := r.db.QueryRowContext(ctx, query, workflowID, scheduleType).Scan(&exists)
	if err != nil {
		return false, engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "exists_for_workflow").
			WithDetail("workflow_id", workflowID.String()).
			WithCause(err)
	}

	return exists, nil
}

// DeleteByWorkflow deletes all schedules for a workflow
func (r *PostgresScheduleRepository) DeleteByWorkflow(
	ctx context.Context,
	workflowID kernel.WorkflowID,
) error {
	query := `DELETE FROM workflow_schedules WHERE workflow_id = $1`

	_, err := r.db.ExecContext(ctx, query, workflowID)
	if err != nil {
		return engine.ErrScheduleExecutionFailed().
			WithDetail("operation", "delete_by_workflow").
			WithDetail("workflow_id", workflowID.String()).
			WithCause(err)
	}

	return nil
}
