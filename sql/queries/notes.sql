-- name: UpsertNote :one
INSERT INTO notes (id, content_id, body, created_at, updated_at)
VALUES ($1, $2, $3, NOW(), NOW())
ON CONFLICT (content_id) 
DO UPDATE SET 
    body = EXCLUDED.body,
    updated_at = NOW()
RETURNING *;

-- name: GetNoteByContentID :one
SELECT * FROM notes WHERE content_id = $1;

-- name: DeleteNoteByContentID :exec
DELETE FROM notes WHERE content_id = $1;
