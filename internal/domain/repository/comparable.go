package repository

import (
	"context"

	"jo3qma.com/yahoo_auctions/internal/domain/model"
)

// ComparableRepository は Sold search から Comparable を取得します。
type ComparableRepository interface {
	SearchSold(ctx context.Context, categoryID, keyword string, page int64) (*model.ComparablePage, error)
}
