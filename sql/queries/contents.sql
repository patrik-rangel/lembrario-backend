-- name: CreateContent :one
INSERT INTO contents (id, url, status, type)
VALUES ($1, $2, 'PENDING', $3)
RETURNING *;

-- name: GetContentByID :one
SELECT c.id, c.url, c.status, c.type, c.created_at, c.updated_at,
       m.title, m.description, m.thumbnail_path, m.transcript, m.provider, m.reading_time, m.raw_data,
       n.id as note_id, n.body as note_body, n.created_at as note_created_at, n.updated_at as note_updated_at
FROM contents c
LEFT JOIN metadata m ON c.id = m.content_id
LEFT JOIN notes n ON c.id = n.content_id
WHERE c.id = $1;

-- name: DeleteContent :exec
DELETE FROM contents WHERE id = $1;

-- name: GetContents :many
SELECT c.id, c.url, c.status, c.type, c.created_at, c.updated_at,
       m.title, m.description, m.thumbnail_path, m.transcript, m.provider, m.reading_time, m.raw_data,
       n.id as note_id, n.body as note_body, n.created_at as note_created_at, n.updated_at as note_updated_at
FROM contents c
LEFT JOIN metadata m ON c.id = m.content_id
LEFT JOIN notes n ON c.id = n.content_id
ORDER BY c.created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpsertMetadata :one
INSERT INTO metadata (
    content_id, 
    title, 
    description, 
    thumbnail_path, 
    transcript, 
    provider, 
    reading_time, 
    raw_data,
    updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, NOW()
)
ON CONFLICT (content_id) DO UPDATE SET
    title = EXCLUDED.title,
    description = EXCLUDED.description,
    thumbnail_path = EXCLUDED.thumbnail_path,
    transcript = EXCLUDED.transcript,
    provider = EXCLUDED.provider,
    reading_time = EXCLUDED.reading_time,
    raw_data = EXCLUDED.raw_data,
    updated_at = NOW()
RETURNING *;

-- name: UpdateContentStatus :exec
UPDATE contents
SET status = $2, 
    updated_at = NOW()
WHERE id = $1;

-- name: ListContentsWithMetadata :many
SELECT
    c.id,
    c.url,
    c.type,
    c.created_at,
    c.status,
    m.title,
    m.description,
    m.provider,
    m.thumbnail_path
FROM contents c
INNER JOIN metadata m ON m.content_id = c.id
WHERE c.status = 'COMPLETED'
ORDER BY c.created_at DESC;
