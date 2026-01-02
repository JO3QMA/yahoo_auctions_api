package handler

import (
	"context"

	"connectrpc.com/connect"
	yahoo_auctionv1 "github.com/jo3qma/protobuf/gen/go/yahoo_auction/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	resp := &yahoo_auctionv1.GetAuctionResponse{
		AuctionId:    item.AuctionID,
		Title:        item.Title,
		CurrentPrice: item.CurrentPrice,
		Status:       yahoo_auctionv1.AuctionStatus(item.Status),
		Images:       item.Images,
		Description:  item.Description,
	}

	// オークション情報を変換
	if item.AuctionInfo != nil {
		resp.AuctionInformation = &yahoo_auctionv1.AuctionInformation{
			AuctionId:        item.AuctionInfo.AuctionID,
			StartPrice:       item.AuctionInfo.StartPrice,
			EarlyEnd:         item.AuctionInfo.EarlyEnd,
			AutoExtension:    item.AuctionInfo.AutoExtension,
			Returnable:       item.AuctionInfo.Returnable,
			ReturnableDetail: item.AuctionInfo.ReturnableDetail,
		}

		// 開始日時を変換
		if !item.AuctionInfo.StartTime.IsZero() {
			resp.AuctionInformation.StartTime = timestamppb.New(item.AuctionInfo.StartTime)
		}

		// 終了日時を変換
		if !item.AuctionInfo.EndTime.IsZero() {
			resp.AuctionInformation.EndTime = timestamppb.New(item.AuctionInfo.EndTime)
		}
	}

	return connect.NewResponse(resp), nil
}
