package service

import (
	"context"
	"lembrario-backend/internal/db"
)

type ContentServiceInterface interface {
	Create(ctx context.Context, params CreateContentParams) (db.Content, error)
	GetByID(ctx context.Context, contentID string) (db.GetContentByIDRow, error)
	GetContents(ctx context.Context, params GetContentsParams) ([]db.GetContentsRow, error)
	UpsertNote(ctx context.Context, params UpdateNoteParams) (db.Note, error)
	Delete(ctx context.Context, contentID string) error
}
