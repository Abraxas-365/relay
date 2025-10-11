package tenantinfra

import (
	"context"
	"database/sql"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/relay/iam/tenant"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// PostgresTenantRepository implementación de PostgreSQL para TenantRepository
type PostgresTenantRepository struct {
	db *sqlx.DB
}

// NewPostgresTenantRepository crea una nueva instancia del repositorio de tenants
func NewPostgresTenantRepository(db *sqlx.DB) tenant.TenantRepository {
	return &PostgresTenantRepository{
		db: db,
	}
}

// FindByID busca un tenant por ID
func (r *PostgresTenantRepository) FindByID(ctx context.Context, id kernel.TenantID) (*tenant.Tenant, error) {
	query := `
		SELECT 
			id, company_name, ruc, status, subscription_plan, max_users, current_users,
			trial_expires_at, subscription_expires_at,
			sire_client_id, sire_client_secret, sire_username, sire_password,
			created_at, updated_at
		FROM tenants 
		WHERE id = $1`

	var t tenant.Tenant
	err := r.db.GetContext(ctx, &t, query, id.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, tenant.ErrTenantNotFound().WithDetail("tenant_id", id.String())
		}
		return nil, errx.Wrap(err, "failed to find tenant by id", errx.TypeInternal).
			WithDetail("tenant_id", id.String())
	}

	return &t, nil
}

// FindByRUC busca un tenant por RUC
func (r *PostgresTenantRepository) FindByRUC(ctx context.Context, ruc string) (*tenant.Tenant, error) {
	query := `
		SELECT 
			id, company_name, ruc, status, subscription_plan, max_users, current_users,
			trial_expires_at, subscription_expires_at,
			sire_client_id, sire_client_secret, sire_username, sire_password,
			created_at, updated_at
		FROM tenants 
		WHERE ruc = $1`

	var t tenant.Tenant
	err := r.db.GetContext(ctx, &t, query, ruc)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, tenant.ErrTenantNotFound().WithDetail("ruc", ruc)
		}
		return nil, errx.Wrap(err, "failed to find tenant by ruc", errx.TypeInternal).
			WithDetail("ruc", ruc)
	}

	return &t, nil
}

// FindAll busca todos los tenants
func (r *PostgresTenantRepository) FindAll(ctx context.Context) ([]*tenant.Tenant, error) {
	query := `
		SELECT 
			id, company_name, ruc, status, subscription_plan, max_users, current_users,
			trial_expires_at, subscription_expires_at,
			sire_client_id, sire_client_secret, sire_username, sire_password,
			created_at, updated_at
		FROM tenants 
		ORDER BY company_name ASC`

	var tenants []tenant.Tenant
	err := r.db.SelectContext(ctx, &tenants, query)
	if err != nil {
		return nil, errx.Wrap(err, "failed to find all tenants", errx.TypeInternal)
	}

	// Convertir a slice de punteros
	result := make([]*tenant.Tenant, len(tenants))
	for i := range tenants {
		result[i] = &tenants[i]
	}

	return result, nil
}

// FindActive busca todos los tenants activos
func (r *PostgresTenantRepository) FindActive(ctx context.Context) ([]*tenant.Tenant, error) {
	query := `
		SELECT 
			id, company_name, ruc, status, subscription_plan, max_users, current_users,
			trial_expires_at, subscription_expires_at,
			sire_client_id, sire_client_secret, sire_username, sire_password,
			created_at, updated_at
		FROM tenants 
		WHERE status = $1
		ORDER BY company_name ASC`

	var tenants []tenant.Tenant
	err := r.db.SelectContext(ctx, &tenants, query, tenant.TenantStatusActive)
	if err != nil {
		return nil, errx.Wrap(err, "failed to find active tenants", errx.TypeInternal)
	}

	// Convertir a slice de punteros
	result := make([]*tenant.Tenant, len(tenants))
	for i := range tenants {
		result[i] = &tenants[i]
	}

	return result, nil
}

// Save guarda o actualiza un tenant
func (r *PostgresTenantRepository) Save(ctx context.Context, t tenant.Tenant) error {
	// Verificar si el tenant ya existe
	exists, err := r.tenantExists(ctx, t.ID)
	if err != nil {
		return errx.Wrap(err, "failed to check tenant existence", errx.TypeInternal)
	}

	if exists {
		return r.update(ctx, t)
	}
	return r.create(ctx, t)
}

// create crea un nuevo tenant
func (r *PostgresTenantRepository) create(ctx context.Context, t tenant.Tenant) error {
	query := `
		INSERT INTO tenants (
			id, company_name, ruc, status, subscription_plan, max_users, current_users,
			trial_expires_at, subscription_expires_at,
			sire_client_id, sire_client_secret, sire_username, sire_password,
			created_at, updated_at
		) VALUES (
			:id, :company_name, :ruc, :status, :subscription_plan, :max_users, :current_users,
			:trial_expires_at, :subscription_expires_at,
			:sire_client_id, :sire_client_secret, :sire_username, :sire_password,
			:created_at, :updated_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, t)
	if err != nil {
		// Verificar violación de constraint de RUC único
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" && pqErr.Constraint == "tenants_ruc_key" {
				return tenant.ErrTenantAlreadyExists().
					WithDetail("ruc", t.RUC)
			}
		}
		return errx.Wrap(err, "failed to create tenant", errx.TypeInternal).
			WithDetail("tenant_id", t.ID.String()).
			WithDetail("ruc", t.RUC)
	}

	return nil
}

// update actualiza un tenant existente
func (r *PostgresTenantRepository) update(ctx context.Context, t tenant.Tenant) error {
	query := `
		UPDATE tenants SET
			company_name = :company_name,
			ruc = :ruc,
			status = :status,
			subscription_plan = :subscription_plan,
			max_users = :max_users,
			current_users = :current_users,
			trial_expires_at = :trial_expires_at,
			subscription_expires_at = :subscription_expires_at,
			sire_client_id = :sire_client_id,
			sire_client_secret = :sire_client_secret,
			sire_username = :sire_username,
			sire_password = :sire_password,
			updated_at = :updated_at
		WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, t)
	if err != nil {
		// Verificar violación de constraint de RUC único
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" && pqErr.Constraint == "tenants_ruc_key" {
				return tenant.ErrTenantAlreadyExists().
					WithDetail("ruc", t.RUC)
			}
		}
		return errx.Wrap(err, "failed to update tenant", errx.TypeInternal).
			WithDetail("tenant_id", t.ID.String())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return tenant.ErrTenantNotFound().WithDetail("tenant_id", t.ID.String())
	}

	return nil
}

// Delete elimina un tenant
func (r *PostgresTenantRepository) Delete(ctx context.Context, id kernel.TenantID) error {
	query := `DELETE FROM tenants WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id.String())
	if err != nil {
		return errx.Wrap(err, "failed to delete tenant", errx.TypeInternal).
			WithDetail("tenant_id", id.String())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return tenant.ErrTenantNotFound().WithDetail("tenant_id", id.String())
	}

	return nil
}

// ExistsByRUC verifica si existe un tenant con el RUC dado
func (r *PostgresTenantRepository) ExistsByRUC(ctx context.Context, ruc string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM tenants WHERE ruc = $1)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, ruc)
	if err != nil {
		return false, errx.Wrap(err, "failed to check tenant existence by ruc", errx.TypeInternal).
			WithDetail("ruc", ruc)
	}

	return exists, nil
}

// tenantExists verifica si un tenant existe por ID
func (r *PostgresTenantRepository) tenantExists(ctx context.Context, id kernel.TenantID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM tenants WHERE id = $1)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, id.String())
	if err != nil {
		return false, errx.Wrap(err, "failed to check tenant existence", errx.TypeInternal).
			WithDetail("tenant_id", id.String())
	}

	return exists, nil
}

// FindByStatus busca tenants por estado
func (r *PostgresTenantRepository) FindByStatus(ctx context.Context, status tenant.TenantStatus) ([]*tenant.Tenant, error) {
	query := `
		SELECT 
			id, company_name, ruc, status, subscription_plan, max_users, current_users,
			trial_expires_at, subscription_expires_at,
			sire_client_id, sire_client_secret, sire_username, sire_password,
			created_at, updated_at
		FROM tenants 
		WHERE status = $1
		ORDER BY company_name ASC`

	var tenants []tenant.Tenant
	err := r.db.SelectContext(ctx, &tenants, query, status)
	if err != nil {
		return nil, errx.Wrap(err, "failed to find tenants by status", errx.TypeInternal).
			WithDetail("status", string(status))
	}

	// Convertir a slice de punteros
	result := make([]*tenant.Tenant, len(tenants))
	for i := range tenants {
		result[i] = &tenants[i]
	}

	return result, nil
}

// FindBySubscriptionPlan busca tenants por plan de suscripción
func (r *PostgresTenantRepository) FindBySubscriptionPlan(ctx context.Context, plan tenant.SubscriptionPlan) ([]*tenant.Tenant, error) {
	query := `
		SELECT 
			id, company_name, ruc, status, subscription_plan, max_users, current_users,
			trial_expires_at, subscription_expires_at,
			sire_client_id, sire_client_secret, sire_username, sire_password,
			created_at, updated_at
		FROM tenants 
		WHERE subscription_plan = $1
		ORDER BY company_name ASC`

	var tenants []tenant.Tenant
	err := r.db.SelectContext(ctx, &tenants, query, plan)
	if err != nil {
		return nil, errx.Wrap(err, "failed to find tenants by subscription plan", errx.TypeInternal).
			WithDetail("plan", string(plan))
	}

	// Convertir a slice de punteros
	result := make([]*tenant.Tenant, len(tenants))
	for i := range tenants {
		result[i] = &tenants[i]
	}

	return result, nil
}

// FindTenantsWithSireCredentials busca tenants que tienen credenciales SIRE configuradas
func (r *PostgresTenantRepository) FindTenantsWithSireCredentials(ctx context.Context) ([]*tenant.Tenant, error) {
	query := `
		SELECT 
			id, company_name, ruc, status, subscription_plan, max_users, current_users,
			trial_expires_at, subscription_expires_at,
			sire_client_id, sire_client_secret, sire_username, sire_password,
			created_at, updated_at
		FROM tenants 
		WHERE sire_client_id IS NOT NULL 
			AND sire_client_secret IS NOT NULL
			AND sire_username IS NOT NULL
			AND sire_password IS NOT NULL
		ORDER BY company_name ASC`

	var tenants []tenant.Tenant
	err := r.db.SelectContext(ctx, &tenants, query)
	if err != nil {
		return nil, errx.Wrap(err, "failed to find tenants with sire credentials", errx.TypeInternal)
	}

	// Convertir a slice de punteros
	result := make([]*tenant.Tenant, len(tenants))
	for i := range tenants {
		result[i] = &tenants[i]
	}

	return result, nil
}

// CountAll cuenta todos los tenants
func (r *PostgresTenantRepository) CountAll(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM tenants`

	var count int
	err := r.db.GetContext(ctx, &count, query)
	if err != nil {
		return 0, errx.Wrap(err, "failed to count all tenants", errx.TypeInternal)
	}

	return count, nil
}

// CountByStatus cuenta tenants por estado
func (r *PostgresTenantRepository) CountByStatus(ctx context.Context, status tenant.TenantStatus) (int, error) {
	query := `SELECT COUNT(*) FROM tenants WHERE status = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, status)
	if err != nil {
		return 0, errx.Wrap(err, "failed to count tenants by status", errx.TypeInternal).
			WithDetail("status", string(status))
	}

	return count, nil
}

// =============================================================================
// TenantConfigRepository Implementation
// =============================================================================

// PostgresTenantConfigRepository implementación de PostgreSQL para TenantConfigRepository
type PostgresTenantConfigRepository struct {
	db *sqlx.DB
}

// NewPostgresTenantConfigRepository crea una nueva instancia del repositorio de configuración de tenants
func NewPostgresTenantConfigRepository(db *sqlx.DB) tenant.TenantConfigRepository {
	return &PostgresTenantConfigRepository{
		db: db,
	}
}

// FindByTenant busca todas las configuraciones de un tenant
func (r *PostgresTenantConfigRepository) FindByTenant(ctx context.Context, tenantID kernel.TenantID) (map[string]string, error) {
	query := `
		SELECT key, value 
		FROM tenant_config 
		WHERE tenant_id = $1
		ORDER BY key ASC`

	rows, err := r.db.QueryContext(ctx, query, tenantID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find tenant config", errx.TypeInternal).
			WithDetail("tenant_id", tenantID.String())
	}
	defer rows.Close()

	config := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, errx.Wrap(err, "failed to scan tenant config row", errx.TypeInternal)
		}
		config[key] = value
	}

	if err := rows.Err(); err != nil {
		return nil, errx.Wrap(err, "error iterating tenant config rows", errx.TypeInternal)
	}

	return config, nil
}

// SaveSetting guarda o actualiza una configuración específica
func (r *PostgresTenantConfigRepository) SaveSetting(ctx context.Context, tenantID kernel.TenantID, key, value string) error {
	query := `
		INSERT INTO tenant_config (tenant_id, key, value, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (tenant_id, key) 
		DO UPDATE SET 
			value = EXCLUDED.value,
			updated_at = NOW()`

	_, err := r.db.ExecContext(ctx, query, tenantID.String(), key, value)
	if err != nil {
		return errx.Wrap(err, "failed to save tenant config setting", errx.TypeInternal).
			WithDetail("tenant_id", tenantID.String()).
			WithDetail("key", key)
	}

	return nil
}

// DeleteSetting elimina una configuración específica
func (r *PostgresTenantConfigRepository) DeleteSetting(ctx context.Context, tenantID kernel.TenantID, key string) error {
	query := `DELETE FROM tenant_config WHERE tenant_id = $1 AND key = $2`

	result, err := r.db.ExecContext(ctx, query, tenantID.String(), key)
	if err != nil {
		return errx.Wrap(err, "failed to delete tenant config setting", errx.TypeInternal).
			WithDetail("tenant_id", tenantID.String()).
			WithDetail("key", key)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return errx.New("tenant config setting not found", errx.TypeNotFound).
			WithDetail("tenant_id", tenantID.String()).
			WithDetail("key", key)
	}

	return nil
}

// GetSetting obtiene una configuración específica
func (r *PostgresTenantConfigRepository) GetSetting(ctx context.Context, tenantID kernel.TenantID, key string) (string, error) {
	query := `SELECT value FROM tenant_config WHERE tenant_id = $1 AND key = $2`

	var value string
	err := r.db.GetContext(ctx, &value, query, tenantID.String(), key)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", errx.New("tenant config setting not found", errx.TypeNotFound).
				WithDetail("tenant_id", tenantID.String()).
				WithDetail("key", key)
		}
		return "", errx.Wrap(err, "failed to get tenant config setting", errx.TypeInternal).
			WithDetail("tenant_id", tenantID.String()).
			WithDetail("key", key)
	}

	return value, nil
}

// HasSetting verifica si existe una configuración específica
func (r *PostgresTenantConfigRepository) HasSetting(ctx context.Context, tenantID kernel.TenantID, key string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM tenant_config WHERE tenant_id = $1 AND key = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, tenantID.String(), key)
	if err != nil {
		return false, errx.Wrap(err, "failed to check tenant config setting existence", errx.TypeInternal).
			WithDetail("tenant_id", tenantID.String()).
			WithDetail("key", key)
	}

	return exists, nil
}

// DeleteAllSettings elimina todas las configuraciones de un tenant
func (r *PostgresTenantConfigRepository) DeleteAllSettings(ctx context.Context, tenantID kernel.TenantID) error {
	query := `DELETE FROM tenant_config WHERE tenant_id = $1`

	_, err := r.db.ExecContext(ctx, query, tenantID.String())
	if err != nil {
		return errx.Wrap(err, "failed to delete all tenant config settings", errx.TypeInternal).
			WithDetail("tenant_id", tenantID.String())
	}

	return nil
}

// CountSettings cuenta las configuraciones de un tenant
func (r *PostgresTenantConfigRepository) CountSettings(ctx context.Context, tenantID kernel.TenantID) (int, error) {
	query := `SELECT COUNT(*) FROM tenant_config WHERE tenant_id = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, tenantID.String())
	if err != nil {
		return 0, errx.Wrap(err, "failed to count tenant config settings", errx.TypeInternal).
			WithDetail("tenant_id", tenantID.String())
	}

	return count, nil
}

// BatchSaveSettings guarda múltiples configuraciones de una vez
func (r *PostgresTenantConfigRepository) BatchSaveSettings(ctx context.Context, tenantID kernel.TenantID, settings map[string]string) error {
	if len(settings) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return errx.Wrap(err, "failed to begin transaction", errx.TypeInternal)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO tenant_config (tenant_id, key, value, created_at, updated_at)
		VALUES ($1, $2, $3, NOW(), NOW())
		ON CONFLICT (tenant_id, key) 
		DO UPDATE SET 
			value = EXCLUDED.value,
			updated_at = NOW()`

	for key, value := range settings {
		_, err := tx.ExecContext(ctx, query, tenantID.String(), key, value)
		if err != nil {
			return errx.Wrap(err, "failed to save tenant config setting in batch", errx.TypeInternal).
				WithDetail("tenant_id", tenantID.String()).
				WithDetail("key", key)
		}
	}

	if err := tx.Commit(); err != nil {
		return errx.Wrap(err, "failed to commit tenant config batch transaction", errx.TypeInternal)
	}

	return nil
}
