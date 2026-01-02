package usecase

import (
	"context"

	"jo3qma.com/yahoo_auctions/internal/domain/model"
	"jo3qma.com/yahoo_auctions/internal/domain/repository"
)

// CategoryUsecase はカテゴリ検索関連のビジネスロジックを担当します
type CategoryUsecase struct {
	repo repository.CategoryItemRepository
}

// NewCategoryUsecase は新しいCategoryUsecaseインスタンスを作成します
func NewCategoryUsecase(repo repository.CategoryItemRepository) *CategoryUsecase {
	return &CategoryUsecase{
		repo: repo,
	}
}

// GetCategoryItems は指定されたカテゴリIDから商品一覧を取得します
func (u *CategoryUsecase) GetCategoryItems(ctx context.Context, categoryID string, page int64) (*model.CategoryItemsPage, error) {
	// ここでバリデーションや追加のビジネスロジックがあれば記述します
	return u.repo.FetchByCategory(ctx, categoryID, page)
}
