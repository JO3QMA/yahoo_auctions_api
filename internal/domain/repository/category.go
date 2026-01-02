package repository

import (
	"context"

	"jo3qma.com/yahoo_auctions/internal/domain/model"
)

// CategoryItemRepository はカテゴリ商品の取得方法を抽象化します。
type CategoryItemRepository interface {
	// FetchByCategory は指定されたカテゴリIDから商品一覧を取得します
	// page は 0 始まりのページ番号です
	FetchByCategory(ctx context.Context, categoryID string, page int64) (*model.CategoryItemsPage, error)
}
