package usecase

import (
	"context"

	"jo3qma.com/yahoo_auctions/internal/domain/model"
	"jo3qma.com/yahoo_auctions/internal/domain/repository"
)

// AuctionUsecase はオークション関連のビジネスロジックを担当します
// 単一責任の原則に従い、オークション取得のユースケースのみを扱います
type AuctionUsecase struct {
	repo repository.ItemRepository
}

// NewAuctionUsecase は新しいAuctionUsecaseインスタンスを作成します
func NewAuctionUsecase(repo repository.ItemRepository) *AuctionUsecase {
	return &AuctionUsecase{
		repo: repo,
	}
}

// GetAuction は指定されたオークションIDから商品情報を取得します
// 将来的に「取得したデータを加工する」などのロジックが入る場所です
func (u *AuctionUsecase) GetAuction(ctx context.Context, auctionID string) (*model.Item, error) {
	return u.repo.FetchByID(ctx, auctionID)
}
