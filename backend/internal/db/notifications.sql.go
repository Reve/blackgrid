// Hand-written queries for notification channels and deliveries.
// Mirrors sqlc style; regenerate from sql/query.sql when sqlc is run.

package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const channelColumns = `id, name, channel_type, enabled, config, created_at, updated_at`

func scanChannel(row pgx.Row, c *NotificationChannel) error {
	return row.Scan(&c.ID, &c.Name, &c.ChannelType, &c.Enabled, &c.Config, &c.CreatedAt, &c.UpdatedAt)
}

const createNotificationChannel = `INSERT INTO notification_channels (name, channel_type, enabled, config)
VALUES ($1, $2, $3, $4)
RETURNING ` + channelColumns

type CreateNotificationChannelParams struct {
	Name        string `json:"name"`
	ChannelType string `json:"channel_type"`
	Enabled     bool   `json:"enabled"`
	Config      []byte `json:"config"`
}

func (q *Queries) CreateNotificationChannel(ctx context.Context, arg CreateNotificationChannelParams) (NotificationChannel, error) {
	row := q.db.QueryRow(ctx, createNotificationChannel, arg.Name, arg.ChannelType, arg.Enabled, arg.Config)
	var c NotificationChannel
	err := scanChannel(row, &c)
	return c, err
}

const getNotificationChannel = `SELECT ` + channelColumns + ` FROM notification_channels WHERE id = $1 LIMIT 1`

func (q *Queries) GetNotificationChannel(ctx context.Context, id pgtype.UUID) (NotificationChannel, error) {
	row := q.db.QueryRow(ctx, getNotificationChannel, id)
	var c NotificationChannel
	err := scanChannel(row, &c)
	return c, err
}

const listNotificationChannels = `SELECT ` + channelColumns + ` FROM notification_channels ORDER BY created_at DESC`

func (q *Queries) ListNotificationChannels(ctx context.Context) ([]NotificationChannel, error) {
	rows, err := q.db.Query(ctx, listNotificationChannels)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []NotificationChannel
	for rows.Next() {
		var c NotificationChannel
		if err := scanChannel(rows, &c); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

const listEnabledNotificationChannels = `SELECT ` + channelColumns + ` FROM notification_channels WHERE enabled = true ORDER BY created_at DESC`

func (q *Queries) ListEnabledNotificationChannels(ctx context.Context) ([]NotificationChannel, error) {
	rows, err := q.db.Query(ctx, listEnabledNotificationChannels)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []NotificationChannel
	for rows.Next() {
		var c NotificationChannel
		if err := scanChannel(rows, &c); err != nil {
			return nil, err
		}
		items = append(items, c)
	}
	return items, rows.Err()
}

const updateNotificationChannel = `UPDATE notification_channels
SET name = $2, channel_type = $3, enabled = $4, config = $5, updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING ` + channelColumns

type UpdateNotificationChannelParams struct {
	ID          pgtype.UUID `json:"id"`
	Name        string      `json:"name"`
	ChannelType string      `json:"channel_type"`
	Enabled     bool        `json:"enabled"`
	Config      []byte      `json:"config"`
}

func (q *Queries) UpdateNotificationChannel(ctx context.Context, arg UpdateNotificationChannelParams) (NotificationChannel, error) {
	row := q.db.QueryRow(ctx, updateNotificationChannel, arg.ID, arg.Name, arg.ChannelType, arg.Enabled, arg.Config)
	var c NotificationChannel
	err := scanChannel(row, &c)
	return c, err
}

const deleteNotificationChannel = `DELETE FROM notification_channels WHERE id = $1`

func (q *Queries) DeleteNotificationChannel(ctx context.Context, id pgtype.UUID) error {
	_, err := q.db.Exec(ctx, deleteNotificationChannel, id)
	return err
}

const deliveryColumns = `id, channel_id, event_type, status, attempts, last_error, payload, created_at, sent_at`

func scanDelivery(row pgx.Row, d *NotificationDelivery) error {
	return row.Scan(&d.ID, &d.ChannelID, &d.EventType, &d.Status, &d.Attempts, &d.LastError, &d.Payload, &d.CreatedAt, &d.SentAt)
}

const createNotificationDelivery = `INSERT INTO notification_deliveries (channel_id, event_type, status, attempts, last_error, payload, sent_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING ` + deliveryColumns

type CreateNotificationDeliveryParams struct {
	ChannelID pgtype.UUID        `json:"channel_id"`
	EventType string             `json:"event_type"`
	Status    string             `json:"status"`
	Attempts  int32              `json:"attempts"`
	LastError pgtype.Text        `json:"last_error"`
	Payload   []byte             `json:"payload"`
	SentAt    pgtype.Timestamptz `json:"sent_at"`
}

func (q *Queries) CreateNotificationDelivery(ctx context.Context, arg CreateNotificationDeliveryParams) (NotificationDelivery, error) {
	row := q.db.QueryRow(ctx, createNotificationDelivery,
		arg.ChannelID, arg.EventType, arg.Status, arg.Attempts, arg.LastError, arg.Payload, arg.SentAt)
	var d NotificationDelivery
	err := scanDelivery(row, &d)
	return d, err
}

const listDeliveriesForChannel = `SELECT ` + deliveryColumns + ` FROM notification_deliveries WHERE channel_id = $1 ORDER BY created_at DESC LIMIT $2`

type ListDeliveriesForChannelParams struct {
	ChannelID pgtype.UUID `json:"channel_id"`
	Limit     int32       `json:"limit"`
}

func (q *Queries) ListDeliveriesForChannel(ctx context.Context, arg ListDeliveriesForChannelParams) ([]NotificationDelivery, error) {
	rows, err := q.db.Query(ctx, listDeliveriesForChannel, arg.ChannelID, arg.Limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []NotificationDelivery
	for rows.Next() {
		var d NotificationDelivery
		if err := scanDelivery(rows, &d); err != nil {
			return nil, err
		}
		items = append(items, d)
	}
	return items, rows.Err()
}
