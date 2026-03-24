package db

import (
	"context"
)

type Querier interface {
	CreateContent(ctx context.Context, arg CreateContentParams) (Content, error)
	GetContentByID(ctx context.Context, id string) (GetContentByIDRow, error)
}

var _ Querier = (*Queries)(nil)
