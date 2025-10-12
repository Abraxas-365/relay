package channelsinfra

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/Abraxas-365/craftable/errx"
	"github.com/Abraxas-365/craftable/storex"
	"github.com/Abraxas-365/relay/channels"
	"github.com/Abraxas-365/relay/pkg/kernel"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

type PostgresChannelRepository struct {
	db *sqlx.DB
}

var _ channels.ChannelRepository = (*PostgresChannelRepository)(nil)

func NewPostgresChannelRepository(db *sqlx.DB) *PostgresChannelRepository {
	return &PostgresChannelRepository{db: db}
}

func (r *PostgresChannelRepository) Save(ctx context.Context, channel channels.Channel) error {
	exists, err := r.channelExists(ctx, channel.ID, channel.TenantID)
	if err != nil {
		return errx.Wrap(err, "failed to check channel existence", errx.TypeInternal)
	}

	if exists {
		return r.update(ctx, channel)
	}
	return r.create(ctx, channel)
}

func (r *PostgresChannelRepository) create(ctx context.Context, channel channels.Channel) error {
	query := `
		INSERT INTO channels (
			id, tenant_id, type, name, description, config, 
			is_active, webhook_url, created_at, updated_at
		) VALUES (
			:id, :tenant_id, :type, :name, :description, :config,
			:is_active, :webhook_url, :created_at, :updated_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, channel)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" && pqErr.Constraint == "channels_name_tenant_id_key" {
				return channels.ErrChannelAlreadyExists().
					WithDetail("name", channel.Name).
					WithDetail("tenant_id", channel.TenantID.String())
			}
		}
		return errx.Wrap(err, "failed to create channel", errx.TypeInternal).
			WithDetail("channel_id", channel.ID.String())
	}

	return nil
}

func (r *PostgresChannelRepository) update(ctx context.Context, channel channels.Channel) error {
	query := `
		UPDATE channels SET
			type = :type,
			name = :name,
			description = :description,
			config = :config,
			is_active = :is_active,
			webhook_url = :webhook_url,
			updated_at = :updated_at
		WHERE id = :id AND tenant_id = :tenant_id`

	result, err := r.db.NamedExecContext(ctx, query, channel)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" {
				return channels.ErrChannelAlreadyExists().WithDetail("name", channel.Name)
			}
		}
		return errx.Wrap(err, "failed to update channel", errx.TypeInternal).
			WithDetail("channel_id", channel.ID.String())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return channels.ErrChannelNotFound().WithDetail("channel_id", channel.ID.String())
	}

	return nil
}

func (r *PostgresChannelRepository) FindByID(ctx context.Context, id kernel.ChannelID, tenantID kernel.TenantID) (*channels.Channel, error) {
	query := `
		SELECT 
			id, tenant_id, type, name, description, config,
			is_active, webhook_url, created_at, updated_at
		FROM channels
		WHERE id = $1 AND tenant_id = $2`

	var channel channels.Channel
	err := r.db.GetContext(ctx, &channel, query, id.String(), tenantID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, channels.ErrChannelNotFound().WithDetail("channel_id", id.String())
		}
		return nil, errx.Wrap(err, "failed to find channel by id", errx.TypeInternal).
			WithDetail("channel_id", id.String())
	}

	return &channel, nil
}

func (r *PostgresChannelRepository) FindByName(ctx context.Context, name string, tenantID kernel.TenantID) (*channels.Channel, error) {
	query := `
		SELECT 
			id, tenant_id, type, name, description, config,
			is_active, webhook_url, created_at, updated_at
		FROM channels
		WHERE name = $1 AND tenant_id = $2`

	var channel channels.Channel
	err := r.db.GetContext(ctx, &channel, query, name, tenantID.String())
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, channels.ErrChannelNotFound().WithDetail("name", name)
		}
		return nil, errx.Wrap(err, "failed to find channel by name", errx.TypeInternal).
			WithDetail("name", name)
	}

	return &channel, nil
}

func (r *PostgresChannelRepository) Delete(ctx context.Context, id kernel.ChannelID, tenantID kernel.TenantID) error {
	query := `DELETE FROM channels WHERE id = $1 AND tenant_id = $2`

	result, err := r.db.ExecContext(ctx, query, id.String(), tenantID.String())
	if err != nil {
		return errx.Wrap(err, "failed to delete channel", errx.TypeInternal).
			WithDetail("channel_id", id.String())
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errx.Wrap(err, "failed to get rows affected", errx.TypeInternal)
	}

	if rowsAffected == 0 {
		return channels.ErrChannelNotFound().WithDetail("channel_id", id.String())
	}

	return nil
}

func (r *PostgresChannelRepository) ExistsByName(ctx context.Context, name string, tenantID kernel.TenantID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM channels WHERE name = $1 AND tenant_id = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, name, tenantID.String())
	if err != nil {
		return false, errx.Wrap(err, "failed to check channel existence by name", errx.TypeInternal).
			WithDetail("name", name)
	}

	return exists, nil
}

func (r *PostgresChannelRepository) FindByTenant(ctx context.Context, tenantID kernel.TenantID) ([]*channels.Channel, error) {
	query := `
		SELECT 
			id, tenant_id, type, name, description, config,
			is_active, webhook_url, created_at, updated_at
		FROM channels
		WHERE tenant_id = $1
		ORDER BY name ASC`

	var channelList []channels.Channel
	err := r.db.SelectContext(ctx, &channelList, query, tenantID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find channels by tenant", errx.TypeInternal).
			WithDetail("tenant_id", tenantID.String())
	}

	result := make([]*channels.Channel, len(channelList))
	for i := range channelList {
		result[i] = &channelList[i]
	}

	return result, nil
}

func (r *PostgresChannelRepository) FindByType(ctx context.Context, channelType channels.ChannelType, tenantID kernel.TenantID) ([]*channels.Channel, error) {
	query := `
		SELECT 
			id, tenant_id, type, name, description, config,
			is_active, webhook_url, created_at, updated_at
		FROM channels
		WHERE type = $1 AND tenant_id = $2
		ORDER BY name ASC`

	var channelList []channels.Channel
	err := r.db.SelectContext(ctx, &channelList, query, channelType, tenantID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find channels by type", errx.TypeInternal).
			WithDetail("type", string(channelType))
	}

	result := make([]*channels.Channel, len(channelList))
	for i := range channelList {
		result[i] = &channelList[i]
	}

	return result, nil
}

func (r *PostgresChannelRepository) FindActive(ctx context.Context, tenantID kernel.TenantID) ([]*channels.Channel, error) {
	query := `
		SELECT 
			id, tenant_id, type, name, description, config,
			is_active, webhook_url, created_at, updated_at
		FROM channels
		WHERE tenant_id = $1 AND is_active = true
		ORDER BY name ASC`

	var channelList []channels.Channel
	err := r.db.SelectContext(ctx, &channelList, query, tenantID.String())
	if err != nil {
		return nil, errx.Wrap(err, "failed to find active channels", errx.TypeInternal)
	}

	result := make([]*channels.Channel, len(channelList))
	for i := range channelList {
		result[i] = &channelList[i]
	}

	return result, nil
}

func (r *PostgresChannelRepository) FindByProvider(ctx context.Context, provider string, tenantID kernel.TenantID) ([]*channels.Channel, error) {
	query := `
		SELECT 
			id, tenant_id, type, name, description, config,
			is_active, webhook_url, created_at, updated_at
		FROM channels
		WHERE tenant_id = $1 AND config->>'provider' = $2
		ORDER BY name ASC`

	var channelList []channels.Channel
	err := r.db.SelectContext(ctx, &channelList, query, tenantID.String(), provider)
	if err != nil {
		return nil, errx.Wrap(err, "failed to find channels by provider", errx.TypeInternal).
			WithDetail("provider", provider)
	}

	result := make([]*channels.Channel, len(channelList))
	for i := range channelList {
		result[i] = &channelList[i]
	}

	return result, nil
}

func (r *PostgresChannelRepository) List(ctx context.Context, req channels.ListChannelsRequest) (channels.ChannelListResponse, error) {
	// Build WHERE conditions
	var conditions []string
	var args []any
	argPos := 1

	conditions = append(conditions, fmt.Sprintf("tenant_id = $%d", argPos))
	args = append(args, req.TenantID.String())
	argPos++

	if req.Type != nil {
		conditions = append(conditions, fmt.Sprintf("type = $%d", argPos))
		args = append(args, *req.Type)
		argPos++
	}

	if req.IsActive != nil {
		conditions = append(conditions, fmt.Sprintf("is_active = $%d", argPos))
		args = append(args, *req.IsActive)
		argPos++
	}

	if req.Provider != nil {
		conditions = append(conditions, fmt.Sprintf("config->>'provider' = $%d", argPos))
		args = append(args, *req.Provider)
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
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM channels WHERE %s", whereClause)
	var total int
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return channels.ChannelListResponse{}, errx.Wrap(err, "failed to count channels", errx.TypeInternal)
	}

	// Data query with pagination
	dataQuery := fmt.Sprintf(`
		SELECT 
			id, tenant_id, type, name, description, config,
			is_active, webhook_url, created_at, updated_at
		FROM channels
		WHERE %s
		ORDER BY name ASC
		LIMIT $%d OFFSET $%d`,
		whereClause, argPos, argPos+1)

	args = append(args, req.PageSize, req.GetOffset())

	var channelList []channels.Channel
	err = r.db.SelectContext(ctx, &channelList, dataQuery, args...)
	if err != nil {
		return channels.ChannelListResponse{}, errx.Wrap(err, "failed to list channels", errx.TypeInternal)
	}

	return storex.NewPaginated(channelList, total, req.Page, req.PageSize), nil
}

func (r *PostgresChannelRepository) BulkUpdateStatus(ctx context.Context, ids []kernel.ChannelID, tenantID kernel.TenantID, isActive bool) error {
	if len(ids) == 0 {
		return nil
	}

	idStrings := make([]string, len(ids))
	for i, id := range ids {
		idStrings[i] = id.String()
	}

	query := `
		UPDATE channels 
		SET is_active = $1, updated_at = NOW()
		WHERE tenant_id = $2 AND id = ANY($3)`

	_, err := r.db.ExecContext(ctx, query, isActive, tenantID.String(), pq.Array(idStrings))
	if err != nil {
		return errx.Wrap(err, "failed to bulk update channel status", errx.TypeInternal)
	}

	return nil
}

func (r *PostgresChannelRepository) CountByType(ctx context.Context, channelType channels.ChannelType, tenantID kernel.TenantID) (int, error) {
	query := `SELECT COUNT(*) FROM channels WHERE type = $1 AND tenant_id = $2`

	var count int
	err := r.db.GetContext(ctx, &count, query, channelType, tenantID.String())
	if err != nil {
		return 0, errx.Wrap(err, "failed to count channels by type", errx.TypeInternal)
	}

	return count, nil
}

func (r *PostgresChannelRepository) CountByTenant(ctx context.Context, tenantID kernel.TenantID) (int, error) {
	query := `SELECT COUNT(*) FROM channels WHERE tenant_id = $1`

	var count int
	err := r.db.GetContext(ctx, &count, query, tenantID.String())
	if err != nil {
		return 0, errx.Wrap(err, "failed to count channels by tenant", errx.TypeInternal)
	}

	return count, nil
}

func (r *PostgresChannelRepository) channelExists(ctx context.Context, id kernel.ChannelID, tenantID kernel.TenantID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM channels WHERE id = $1 AND tenant_id = $2)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, id.String(), tenantID.String())
	if err != nil {
		return false, errx.Wrap(err, "failed to check channel existence", errx.TypeInternal)
	}

	return exists, nil
}
