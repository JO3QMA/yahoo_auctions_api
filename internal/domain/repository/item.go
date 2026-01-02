package repository

import (
	"context"

	"jo3qma.com/yahoo_auctions/internal/domain/model"
)

// ItemRepository は商品の取得方法を抽象化します。
// 実装がDBなのか、外部APIなのか、スクレイピングなのかはドメイン層は知りません。
// これにより、腐敗防止層（Anti-Corruption Layer）のパターンを実現します。
type ItemRepository interface {
	// FetchByID は指定されたオークションIDから商品情報を取得します
	FetchByID(ctx context.Context, auctionID string) (*model.Item, error)
}
