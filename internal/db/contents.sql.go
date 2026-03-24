package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createContent = `-- name: CreateContent :one
INSERT INTO contents (id, url, status, type)
VALUES ($1, $2, 'PENDING', $3)
RETURNING id, url, status, type, created_at, updated_at
`

type CreateContentParams struct {
	ID   string
	Url  string
	Type pgtype.Text
}

func (q *Queries) CreateContent(ctx context.Context, arg CreateContentParams) (Content, error) {
	row := q.db.QueryRow(ctx, createContent, arg.ID, arg.Url, arg.Type)
	var i Content
	err := row.Scan(
		&i.ID,
		&i.Url,
		&i.Status,
		&i.Type,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getContentByID = `-- name: GetContentByID :one
SELECT c.id, c.url, c.status, c.type, c.created_at, c.updated_at,
       m.title, m.description, m.thumbnail_path, m.transcript, m.provider, m.reading_time, m.raw_data
FROM contents c
LEFT JOIN metadata m ON c.id = m.content_id
WHERE c.id = $1
`

type GetContentByIDRow struct {
	ID            string
	Url           string
	Status        string
	Type          pgtype.Text
	CreatedAt     pgtype.Timestamptz
	UpdatedAt     pgtype.Timestamptz
	Title         pgtype.Text
	Description   pgtype.Text
	ThumbnailPath pgtype.Text
	Transcript    pgtype.Text
	Provider      pgtype.Text
	ReadingTime   pgtype.Int4
	RawData       []byte
}

func (q *Queries) GetContentByID(ctx context.Context, id string) (GetContentByIDRow, error) {
	row := q.db.QueryRow(ctx, getContentByID, id)
	var i GetContentByIDRow
	err := row.Scan(
		&i.ID,
		&i.Url,
		&i.Status,
		&i.Type,
		&i.CreatedAt,
		&i.UpdatedAt,
		&i.Title,
		&i.Description,
		&i.ThumbnailPath,
		&i.Transcript,
		&i.Provider,
		&i.ReadingTime,
		&i.RawData,
	)
	return i, err
}
