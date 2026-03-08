package repository

import (
	"context"

	"jo3qma.com/yahoo_auctions/internal/domain/model"
)

// SearchItemRepository はキーワード検索による商品一覧の取得方法を抽象化します。
type SearchItemRepository interface {
	// Search は指定されたキーワードで商品一覧を取得します（新着順）。
	// page は 0 始まりのページ番号です。
	Search(ctx context.Context, query string, page int64) (*model.CategoryItemsPage, error)
}
