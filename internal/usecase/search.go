package usecase

import (
	"context"

	"jo3qma.com/yahoo_auctions/internal/domain/model"
	"jo3qma.com/yahoo_auctions/internal/domain/repository"
)

// SearchUsecase はキーワード検索関連のビジネスロジックを担当します
type SearchUsecase struct {
	repo repository.SearchItemRepository
}

// NewSearchUsecase は新しいSearchUsecaseインスタンスを作成します
func NewSearchUsecase(repo repository.SearchItemRepository) *SearchUsecase {
	return &SearchUsecase{
		repo: repo,
	}
}

// Search は指定されたキーワードで商品一覧を取得します（新着順）
func (u *SearchUsecase) Search(ctx context.Context, query string, page int64) (*model.CategoryItemsPage, error) {
	return u.repo.Search(ctx, query, page)
}
