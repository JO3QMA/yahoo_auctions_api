package handler

import (
	"context"

	"connectrpc.com/connect"
	yahoo_auctionv1 "github.com/jo3qma/protobuf/gen/go/yahoo_auction/v1"
	"jo3qma.com/yahoo_auctions/internal/usecase"
)

// AuctionHandler はgRPC/Connectのハンドラー実装です
// プロトコル層（protobuf）とドメイン層（usecase）を橋渡しします
type AuctionHandler struct {
	uc *usecase.AuctionUsecase
}

// NewAuctionHandler は新しいAuctionHandlerインスタンスを作成します
func NewAuctionHandler(uc *usecase.AuctionUsecase) *AuctionHandler {
	return &AuctionHandler{
		uc: uc,
	}
}

// GetAuction はオークション商品情報を取得するRPCハンドラーです
func (h *AuctionHandler) GetAuction(
	ctx context.Context,
	req *connect.Request[yahoo_auctionv1.GetAuctionRequest],
) (*connect.Response[yahoo_auctionv1.GetAuctionResponse], error) {
	// ユースケースを呼び出して商品情報を取得
	item, err := h.uc.GetAuction(ctx, req.Msg.AuctionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}

	// ドメインモデルをprotobufのレスポンスに変換
	return connect.NewResponse(&yahoo_auctionv1.GetAuctionResponse{
		AuctionId:    item.AuctionID,
		Title:        item.Title,
		CurrentPrice: item.CurrentPrice,
		ShippingFee:  item.ShippingFee,
		Status:       yahoo_auctionv1.AuctionStatus(item.Status),
	}), nil
}
