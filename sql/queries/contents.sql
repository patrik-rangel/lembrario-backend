-- name: CreateContent :one
INSERT INTO contents (id, url, status, type)
VALUES ($1, $2, 'PENDING', $3)
RETURNING *;

-- name: GetContentByID :one
SELECT c.id, c.url, c.status, c.type, c.created_at, c.updated_at,
       m.title, m.description, m.thumbnail_path, m.transcript, m.provider, m.reading_time, m.raw_data
FROM contents c
LEFT JOIN metadata m ON c.id = m.content_id
WHERE c.id = $1;
