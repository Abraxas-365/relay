package engineinfra

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/craftable/storex"
	"github.com/Abraxas-365/relay/engine"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type PostgresWorkflowRepository struct {
	db *sqlx.DB
}

var _ engine.WorkflowRepository = (*PostgresWorkflowRepository)(nil)

func NewPostgresWorkflowRepository(db *sqlx.DB) *PostgresWorkflowRepository {
	return &PostgresWorkflowRepository{db: db}
}

// dbWorkflow is an intermediate struct for database operations
type dbWorkflow struct {
	ID          string          `db:"id"`
	TenantID    string          `db:"tenant_id"`
	Name        string          `db:"name"`
	Description string          `db:"description"`
	Trigger     json.RawMessage `db:"trigger"`
	Steps       json.RawMessage `db:"steps"`
	IsActive    bool            `db:"is_active"`
	CreatedAt   string          `db:"created_at"`
	UpdatedAt   string          `db:"updated_at"`
}

// toDBWorkflow converts domain Workflow to dbWorkflow
func toDBWorkflow(wf engine.Workflow) (*dbWorkflow, error) {
	triggerJSON, err := json.Marshal(wf.Trigger)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal trigger: %w", err)
	}

	stepsJSON := []byte("[]")
	if wf.Node != nil && len(wf.Node) > 0 {
		stepsJSON, err = json.Marshal(wf.Node)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal steps: %w", err)
		}
	}

	return &dbWorkflow{
		ID:          wf.ID.String(),
		TenantID:    wf.TenantID.String(),
		Name:        wf.Name,
		Description: wf.Description,
		Trigger:     triggerJSON,
		Steps:       stepsJSON,
		IsActive:    wf.IsActive,
		CreatedAt:   wf.CreatedAt.Format("2006-01-02 15:04:05.999999"),
		UpdatedAt:   wf.UpdatedAt.Format("2006-01-02 15:04:05.999999"),
	}, nil
}

// toDomainWorkflow converts dbWorkflow to domain Workflow
func toDomainWorkflow(dbWf *dbWorkflow) (*engine.Workflow, error) {
	var trigger engine.WorkflowTrigger
	if err := json.Unmarshal(dbWf.Trigger, &trigger); err != nil {
		return nil, fmt.Errorf("failed to unmarshal trigger: %w", err)
	}

	var steps []engine.WorkflowNode
	if len(dbWf.Steps) > 0 && string(dbWf.Steps) != "null" {
		if err := json.Unmarshal(dbWf.Steps, &steps); err != nil {
			return nil, fmt.Errorf("failed to unmarshal steps: %w", err)
		}
	}

	wf := &engine.Workflow{
		ID:          kernel.WorkflowID(dbWf.ID),
		TenantID:    kernel.TenantID(dbWf.TenantID),
		Name:        dbWf.Name,
		Description: dbWf.Description,
		Trigger:     trigger,
		Node:        steps,
		IsActive:    dbWf.IsActive,
	}

	return wf, nil
}

func (r *PostgresWorkflowRepository) Save(ctx context.Context, wf engine.Workflow) error {
	exists, err := r.workflowExists(ctx, wf.ID)
	if err != nil {
		return errx.Wrap(err, "failed to check workflow existence", errx.TypeInternal)
	}

	if exists {
		return r.update(ctx, wf)
	}
	return r.create(ctx, wf)
}

func (r *PostgresWorkflowRepository) create(ctx context.Context, wf engine.Workflow) error {
	dbWf, err := toDBWorkflow(wf)
	if err != nil {
		return errx.Wrap(err, "failed to convert workflow", errx.TypeInternal).
			WithDetail("workflow_id", wf.ID.String())
	}

	query := `
		INSERT INTO workflows (
			id, tenant_id, name, description, trigger, steps,
			is_active, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :name, :description, :trigger, :steps,
			:is_active, :created_at, :updated_at
		)`

	_, err = r.db.NamedExecContext(ctx, query, dbWf)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" && pqErr.Constraint == "workflows_name_tenant_id_key" {
				return engine.ErrWorkflowAlreadyExists().
					WithDetail("name", wf.Name).
					WithDetail("tenant_id", wf.TenantID.String())
			}
		}
		return errx.Wrap(err, "failed to create workflow", errx.TypeInternal).
			WithDetail("workflow_id", wf.ID.String())
	}

	return nil
}

func (r *PostgresWorkflowRepository) update(ctx context.Context, wf engine.Workflow) error {
	dbWf, err := toDBWorkflow(wf)
	if err != nil {
		return errx.Wrap(err, "failed to convert workflow", errx.TypeInternal).
			WithDetail("workflow_id", wf.ID.String())
	}

	query := `
		UPDATE workflows SET
			name = :name,
			description = :description,
			trigger = :trigger,
			steps = :steps,
			is_active = :is_active,
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id`

	result, err := r.db.NamedExecContext(ctx, query, dbWf)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return engine.ErrWorkflowAlreadyExists().WithDetail("name", wf.Name)
			}
		}
		return errx.Wrap(err, "failed to update workflow", errx.TypeInternal).
			WithDetail("workflow_id", wf.ID.String())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return engine.ErrWorkflowNotFound().WithDetail("workflow_id", wf.ID.String())
	}

	return nil
}

func (r *PostgresWorkflowRepository) FindByID(ctx context.Context, id kernel.WorkflowID) (*engine.Workflow, error) {
	query := `
		SELECT 
			id, tenant_id, name, description, trigger, steps,
			is_active, created_at, updated_at
		FROM workflows
		WHERE id = $1`

	var dbWf dbWorkflow
	err := r.db.GetContext(ctx, &dbWf, query, id.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, engine.ErrWorkflowNotFound().WithDetail("workflow_id", id.String())
		}
		return nil, errx.Wrap(err, "failed to find workflow by id", errx.TypeInternal).
			WithDetail("workflow_id", id.String())
	}

	return toDomainWorkflow(&dbWf)
}

func (r *PostgresWorkflowRepository) FindByName(ctx context.Context, name string, tenantID kernel.TenantID) (*engine.Workflow, error) {
	query := `
		SELECT 
			id, tenant_id, name, description, trigger, steps,
			is_active, created_at, updated_at
		FROM workflows
		WHERE name = $1 AND tenant_id = $2`

	var dbWf dbWorkflow
	err := r.db.GetContext(ctx, &dbWf, query, name, tenantID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, engine.ErrWorkflowNotFound().WithDetail("name", name)
		}
		return nil, errx.Wrap(err, "failed to find workflow by name", errx.TypeInternal).
			WithDetail("name", name)
	}

	return toDomainWorkflow(&dbWf)
}

func (r *PostgresWorkflowRepository) Delete(ctx context.Context, id kernel.WorkflowID, tenantID kernel.TenantID) error {
	query := `DELETE FROM workflows WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.ExecContext(ctx, query, id.String(), tenantID.String())
	if err != nil {
		return errx.Wrap(err, "failed to delete workflow", errx.TypeInternal).
			WithDetail("workflow_id", id.String())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return engine.ErrWorkflowNotFound().WithDetail("workflow_id", id.String())
	}

	return nil
}

func (r *PostgresWorkflowRepository) ExistsByName(ctx context.Context, name string, tenantID kernel.TenantID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM workflows WHERE name = $1 AND tenant_id = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, name, tenantID.String())
	if err != nil {
		return false, errx.Wrap(err, "failed to check workflow existence by name", errx.TypeInternal).
			WithDetail("name", name)
	}

	return exists, nil
}

func (r *PostgresWorkflowRepository) FindByTenant(ctx context.Context, tenantID kernel.TenantID) ([]*engine.Workflow, error) {
	query := `
		SELECT 
			id, tenant_id, name, description, trigger, steps,
			is_active, created_at, updated_at
		FROM workflows
		WHERE tenant_id = $1
		ORDER BY name ASC`

	var dbWorkflows []dbWorkflow
	err := r.db.SelectContext(ctx, &dbWorkflows, query, tenantID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find workflows by tenant", errx.TypeInternal).
			WithDetail("tenant_id", tenantID.String())
	}

	result := make([]*engine.Workflow, 0, len(dbWorkflows))
	for i := range dbWorkflows {
		wf, err := toDomainWorkflow(&dbWorkflows[i])
		if err != nil {
			return nil, errx.Wrap(err, "failed to convert workflow", errx.TypeInternal)
		}
		result = append(result, wf)
	}

	return result, nil
}

func (r *PostgresWorkflowRepository) FindActive(ctx context.Context, tenantID kernel.TenantID) ([]*engine.Workflow, error) {
	query := `
		SELECT 
			id, tenant_id, name, description, trigger, steps,
			is_active, created_at, updated_at
		FROM workflows
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY name ASC`

	var dbWorkflows []dbWorkflow
	err := r.db.SelectContext(ctx, &dbWorkflows, query, tenantID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find active workflows", errx.TypeInternal)
	}

	result := make([]*engine.Workflow, 0, len(dbWorkflows))
	for i := range dbWorkflows {
		wf, err := toDomainWorkflow(&dbWorkflows[i])
		if err != nil {
			return nil, errx.Wrap(err, "failed to convert workflow", errx.TypeInternal)
		}
		result = append(result, wf)
	}

	return result, nil
}

func (r *PostgresWorkflowRepository) FindByTriggerType(ctx context.Context, triggerType engine.TriggerType, tenantID kernel.TenantID) ([]*engine.Workflow, error) {
	query := `
		SELECT 
			id, tenant_id, name, description, trigger, steps,
			is_active, created_at, updated_at
		FROM workflows
		WHERE tenant_id = $1 AND trigger->>'type' = $2
		ORDER BY name ASC`

	var dbWorkflows []dbWorkflow
	err := r.db.SelectContext(ctx, &dbWorkflows, query, tenantID.String(), string(triggerType))
	if err != nil {
		return nil, errx.Wrap(err, "failed to find workflows by trigger type", errx.TypeInternal).
			WithDetail("trigger_type", string(triggerType))
	}

	result := make([]*engine.Workflow, 0, len(dbWorkflows))
	for i := range dbWorkflows {
		wf, err := toDomainWorkflow(&dbWorkflows[i])
		if err != nil {
			return nil, errx.Wrap(err, "failed to convert workflow", errx.TypeInternal)
		}
		result = append(result, wf)
	}

	return result, nil
}

func (r *PostgresWorkflowRepository) FindActiveByTrigger(ctx context.Context, trigger engine.WorkflowTrigger, tenantID kernel.TenantID) ([]*engine.Workflow, error) {
	query := `
		SELECT 
			id, tenant_id, name, description, trigger, steps,
			is_active, created_at, updated_at
		FROM workflows
		WHERE tenant_id = $1 
			AND is_active = true 
			AND trigger->>'type' = $2
		ORDER BY name ASC`

	var dbWorkflows []dbWorkflow
	err := r.db.SelectContext(ctx, &dbWorkflows, query, tenantID.String(), string(trigger.Type))
	if err != nil {
		return nil, errx.Wrap(err, "failed to find active workflows by trigger", errx.TypeInternal).
			WithDetail("trigger_type", string(trigger.Type))
	}

	result := make([]*engine.Workflow, 0, len(dbWorkflows))
	for i := range dbWorkflows {
		wf, err := toDomainWorkflow(&dbWorkflows[i])
		if err != nil {
			return nil, errx.Wrap(err, "failed to convert workflow", errx.TypeInternal)
		}
		result = append(result, wf)
	}

	return result, nil
}

func (r *PostgresWorkflowRepository) List(ctx context.Context, req engine.WorkflowListRequest) (engine.WorkflowListResponse, error) {
	var conditions []string
	var args []any
	argPos := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argPos))
	args = append(args, req.TenantID.String())
	argPos++

	if req.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argPos))
		args = append(args, *req.IsActive)
		argPos++
	}

	if req.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argPos, argPos+1))
		searchPattern := "%" + req.Search + "%"
		args = append(args, searchPattern, searchPattern)
		argPos += 2
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM workflows WHERE %s", whereClause)
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return engine.WorkflowListResponse{}, errx.Wrap(err, "failed to count workflows", errx.TypeInternal)
	}

	// Data query
	dataQuery := fmt.Sprintf(`
		SELECT 
			id, tenant_id, name, description, trigger, steps,
			is_active, created_at, updated_at
		FROM workflows
		WHERE %s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d`,
		whereClause, argPos, argPos+1)

	args = append(args, req.PageSize, req.GetOffset())

	var dbWorkflows []dbWorkflow
	err = r.db.SelectContext(ctx, &dbWorkflows, dataQuery, args...)
	if err != nil {
		return engine.WorkflowListResponse{}, errx.Wrap(err, "failed to list workflows", errx.TypeInternal)
	}

	workflows := make([]engine.Workflow, 0, len(dbWorkflows))
	for i := range dbWorkflows {
		wf, err := toDomainWorkflow(&dbWorkflows[i])
		if err != nil {
			return engine.WorkflowListResponse{}, errx.Wrap(err, "failed to convert workflow", errx.TypeInternal)
		}
		workflows = append(workflows, *wf)
	}

	return storex.NewPaginated(workflows, total, req.Page, req.PageSize), nil
}

func (r *PostgresWorkflowRepository) BulkUpdateStatus(ctx context.Context, ids []kernel.WorkflowID, tenantID kernel.TenantID, isActive bool) error {
	if len(ids) == 0 {
		return nil
	}

	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = id.String()
	}

	query := `
		UPDATE workflows 
		SET is_active = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND id = ANY($3)`

	_, err := r.db.ExecContext(ctx, query, isActive, tenantID.String(), pq.Array(idStrings))
	if err != nil {
		return errx.Wrap(err, "failed to bulk update workflow status", errx.TypeInternal)
	}

	return nil
}

func (r *PostgresWorkflowRepository) workflowExists(ctx context.Context, id kernel.WorkflowID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM workflows WHERE id = $1)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, id.String())
	if err != nil {
		return false, errx.Wrap(err, "failed to check workflow existence", errx.TypeInternal)
	}

	return exists, nil
}
