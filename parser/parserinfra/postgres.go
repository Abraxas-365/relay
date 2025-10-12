package parserinfra

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/craftable/storex"
	"github.com/Abraxas-365/relay/parser"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/jmoiron/sqlx"
)

type PostgresParserRepository struct {
	db *sqlx.DB
}

var _ parser.ParserRepository = (*PostgresParserRepository)(nil)

func NewPostgresParserRepository(db *sqlx.DB) *PostgresParserRepository {
	return &PostgresParserRepository{db: db}
}

// dbParser is an intermediate struct for database operations
type dbParser struct {
	ID          string          `db:"id"`
	TenantID    string          `db:"tenant_id"`
	Name        string          `db:"name"`
	Description string          `db:"description"`
	Type        string          `db:"type"`
	Config      json.RawMessage `db:"config"`
	Priority    int             `db:"priority"`
	IsActive    bool            `db:"is_active"`
	CreatedAt   time.Time       `db:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at"`
}

// toDBParser converts domain Parser to dbParser
func toDBParser(p parser.Parser) *dbParser {
	return &dbParser{
		ID:          p.ID.String(),
		TenantID:    p.TenantID.String(),
		Name:        p.Name,
		Description: p.Description,
		Type:        string(p.Type),
		Config:      p.Config,
		Priority:    p.Priority,
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

// toDomainParser converts dbParser to domain Parser
func toDomainParser(db *dbParser) *parser.Parser {
	return &parser.Parser{
		ID:          kernel.ParserID(db.ID),
		TenantID:    kernel.TenantID(db.TenantID),
		Name:        db.Name,
		Description: db.Description,
		Type:        parser.ParserType(db.Type),
		Config:      db.Config,
		Priority:    db.Priority,
		IsActive:    db.IsActive,
		CreatedAt:   db.CreatedAt,
		UpdatedAt:   db.UpdatedAt,
	}
}

// Save inserts or updates a parser
func (r *PostgresParserRepository) Save(ctx context.Context, p parser.Parser) error {
	dbP := toDBParser(p)

	query := `
		INSERT INTO parsers (
			id, tenant_id, name, description, type, config, priority, is_active, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :name, :description, :type, :config, :priority, :is_active, :created_at, :updated_at
		)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			type = EXCLUDED.type,
			config = EXCLUDED.config,
			priority = EXCLUDED.priority,
			is_active = EXCLUDED.is_active,
			updated_at = EXCLUDED.updated_at
	`

	_, err := r.db.NamedExecContext(ctx, query, dbP)
	if err != nil {
		return errx.Wrap(err, "failed to save parser", errx.TypeInternal).
			WithDetail("parser_id", p.ID.String())
	}

	return nil
}

// FindByID finds a parser by ID and tenant ID
func (r *PostgresParserRepository) FindByID(ctx context.Context, id kernel.ParserID, tenantID kernel.TenantID) (*parser.Parser, error) {
	query := `
		SELECT id, tenant_id, name, description, type, config, priority, is_active, created_at, updated_at
		FROM parsers
		WHERE id = $1 AND tenant_id = $2
	`

	var dbP dbParser
	err := r.db.GetContext(ctx, &dbP, query, id.String(), tenantID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, parser.ErrParserNotFound().
				WithDetail("parser_id", id.String()).
				WithDetail("tenant_id", tenantID.String())
		}
		return nil, errx.Wrap(err, "failed to find parser by id", errx.TypeInternal)
	}

	return toDomainParser(&dbP), nil
}

// FindByName finds a parser by name and tenant ID
func (r *PostgresParserRepository) FindByName(ctx context.Context, name string, tenantID kernel.TenantID) (*parser.Parser, error) {
	query := `
		SELECT id, tenant_id, name, description, type, config, priority, is_active, created_at, updated_at
		FROM parsers
		WHERE name = $1 AND tenant_id = $2
	`

	var dbP dbParser
	err := r.db.GetContext(ctx, &dbP, query, name, tenantID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, parser.ErrParserNotFound().
				WithDetail("name", name).
				WithDetail("tenant_id", tenantID.String())
		}
		return nil, errx.Wrap(err, "failed to find parser by name", errx.TypeInternal)
	}

	return toDomainParser(&dbP), nil
}

// Delete deletes a parser
func (r *PostgresParserRepository) Delete(ctx context.Context, id kernel.ParserID, tenantID kernel.TenantID) error {
	query := `DELETE FROM parsers WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.ExecContext(ctx, query, id.String(), tenantID.String())
	if err != nil {
		return errx.Wrap(err, "failed to delete parser", errx.TypeInternal).
			WithDetail("parser_id", id.String())
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return parser.ErrParserNotFound().
			WithDetail("parser_id", id.String()).
			WithDetail("tenant_id", tenantID.String())
	}

	return nil
}

// ExistsByName checks if a parser with the given name exists
func (r *PostgresParserRepository) ExistsByName(ctx context.Context, name string, tenantID kernel.TenantID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM parsers WHERE name = $1 AND tenant_id = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, name, tenantID.String())
	if err != nil {
		return false, errx.Wrap(err, "failed to check parser existence", errx.TypeInternal)
	}

	return exists, nil
}

// FindByTenant finds all parsers for a tenant
func (r *PostgresParserRepository) FindByTenant(ctx context.Context, tenantID kernel.TenantID) ([]*parser.Parser, error) {
	query := `
		SELECT id, tenant_id, name, description, type, config, priority, is_active, created_at, updated_at
		FROM parsers
		WHERE tenant_id = $1
		ORDER BY priority DESC, name ASC
	`

	var dbParsers []dbParser
	err := r.db.SelectContext(ctx, &dbParsers, query, tenantID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find parsers by tenant", errx.TypeInternal)
	}

	parsers := make([]*parser.Parser, len(dbParsers))
	for i, dbP := range dbParsers {
		parsers[i] = toDomainParser(&dbP)
	}

	return parsers, nil
}

// FindByType finds all parsers of a specific type for a tenant
func (r *PostgresParserRepository) FindByType(ctx context.Context, parserType parser.ParserType, tenantID kernel.TenantID) ([]*parser.Parser, error) {
	query := `
		SELECT id, tenant_id, name, description, type, config, priority, is_active, created_at, updated_at
		FROM parsers
		WHERE type = $1 AND tenant_id = $2
		ORDER BY priority DESC, name ASC
	`

	var dbParsers []dbParser
	err := r.db.SelectContext(ctx, &dbParsers, query, string(parserType), tenantID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find parsers by type", errx.TypeInternal)
	}

	parsers := make([]*parser.Parser, len(dbParsers))
	for i, dbP := range dbParsers {
		parsers[i] = toDomainParser(&dbP)
	}

	return parsers, nil
}

// FindActive finds all active parsers for a tenant
func (r *PostgresParserRepository) FindActive(ctx context.Context, tenantID kernel.TenantID) ([]*parser.Parser, error) {
	query := `
		SELECT id, tenant_id, name, description, type, config, priority, is_active, created_at, updated_at
		FROM parsers
		WHERE is_active = true AND tenant_id = $1
		ORDER BY priority DESC, name ASC
	`

	var dbParsers []dbParser
	err := r.db.SelectContext(ctx, &dbParsers, query, tenantID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find active parsers", errx.TypeInternal)
	}

	parsers := make([]*parser.Parser, len(dbParsers))
	for i, dbP := range dbParsers {
		parsers[i] = toDomainParser(&dbP)
	}

	return parsers, nil
}

// FindByPriority finds all parsers ordered by priority (descending)
func (r *PostgresParserRepository) FindByPriority(ctx context.Context, tenantID kernel.TenantID) ([]*parser.Parser, error) {
	query := `
		SELECT id, tenant_id, name, description, type, config, priority, is_active, created_at, updated_at
		FROM parsers
		WHERE tenant_id = $1
		ORDER BY priority DESC, name ASC
	`

	var dbParsers []dbParser
	err := r.db.SelectContext(ctx, &dbParsers, query, tenantID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find parsers by priority", errx.TypeInternal)
	}

	parsers := make([]*parser.Parser, len(dbParsers))
	for i, dbP := range dbParsers {
		parsers[i] = toDomainParser(&dbP)
	}

	return parsers, nil
}

// List finds parsers with pagination
func (r *PostgresParserRepository) List(ctx context.Context, req parser.ListParsersRequest) (parser.ParserListResponse, error) {

	// Build query with filters
	conditions := []string{"tenant_id = $1"}
	args := []interface{}{req.TenantID.String()}
	argPos := 2

	if req.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argPos))
		args = append(args, string(*req.Type))
		argPos++
	}

	if req.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argPos))
		args = append(args, *req.IsActive)
		argPos++
	}

	if req.Search != "" {
		conditions = append(conditions, fmt.Sprintf("(name ILIKE $%d OR description ILIKE $%d)", argPos, argPos))
		args = append(args, "%"+req.Search+"%")
		argPos++
	}

	// Build WHERE clause
	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Count total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM parsers %s", whereClause)
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return parser.ParserListResponse{}, errx.Wrap(err, "failed to count parsers", errx.TypeInternal)
	}

	// Get parsers with pagination
	query := fmt.Sprintf(`
		SELECT id, tenant_id, name, description, type, config, priority, is_active, created_at, updated_at
		FROM parsers
		%s
		ORDER BY priority DESC, name ASC
		LIMIT $%d OFFSET $%d
	`, whereClause, argPos, argPos+1)

	args = append(args, req.PageSize, req.GetOffset())

	var dbParsers []dbParser
	err = r.db.SelectContext(ctx, &dbParsers, query, args...)
	if err != nil {
		return parser.ParserListResponse{}, errx.Wrap(err, "failed to list parsers", errx.TypeInternal)
	}

	// Convert to domain parsers
	parsers := make([]parser.Parser, len(dbParsers))
	for i, dbP := range dbParsers {
		parsers[i] = *toDomainParser(&dbP)
	}

	// Return paginated response
	return storex.NewPaginated(parsers, req.Page, req.PageSize, total), nil
}

// BulkUpdateStatus updates the status of multiple parsers
func (r *PostgresParserRepository) BulkUpdateStatus(ctx context.Context, ids []kernel.ParserID, tenantID kernel.TenantID, isActive bool) error {
	if len(ids) == 0 {
		return nil
	}

	// Convert IDs to strings
	idStrs := make([]string, len(ids))
	for i, id := range ids {
		idStrs[i] = id.String()
	}

	query := `
		UPDATE parsers
		SET is_active = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND id = ANY($3)
	`

	_, err := r.db.ExecContext(ctx, query, isActive, tenantID.String(), idStrs)
	if err != nil {
		return errx.Wrap(err, "failed to bulk update parser status", errx.TypeInternal)
	}

	return nil
}
