package handler

import (
	"context"

	"connectrpc.com/connect"
	yahoo_auctionv1 "github.com/jo3qma/protobuf/gen/go/yahoo_auction/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"jo3qma.com/yahoo_auctions/internal/domain/model"
)

// AuctionGetter はオークション取得ユースケースの最小インターフェースです。
// handler層は具象（usecase.AuctionUsecase）に依存せず、境界変換に集中します。
type AuctionGetter interface {
	GetAuction(ctx context.Context, auctionID string) (*model.Item, error)
}

// CategoryGetter はカテゴリ商品取得ユースケースの最小インターフェースです。
type CategoryGetter interface {
	GetCategoryItems(ctx context.Context, categoryID string, page int64) (*model.CategoryItemsPage, error)
}

// AuctionHandler はgRPC/Connectのハンドラー実装です
// プロトコル層（protobuf）とドメイン層（usecase）を橋渡しします
type AuctionHandler struct {
	uc    AuctionGetter
	catUC CategoryGetter
}

// NewAuctionHandler は新しいAuctionHandlerインスタンスを作成します
func NewAuctionHandler(uc AuctionGetter, catUC CategoryGetter) *AuctionHandler {
	return &AuctionHandler{
		uc:    uc,
		catUC: catUC,
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

// GetCategoryItems はカテゴリの商品一覧を取得するRPCハンドラーです
func (h *AuctionHandler) GetCategoryItems(
	ctx context.Context,
	req *connect.Request[yahoo_auctionv1.GetCategoryItemsRequest],
) (*connect.Response[yahoo_auctionv1.GetCategoryItemsResponse], error) {
	// ユースケースを呼び出して一覧を取得
	pageResult, err := h.catUC.GetCategoryItems(ctx, req.Msg.CategoryId, req.Msg.Page)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// protoへの変換
	items := make([]*yahoo_auctionv1.GetCategoryItemsResponse_Item, 0, len(pageResult.Items))
	for _, item := range pageResult.Items {
		items = append(items, &yahoo_auctionv1.GetCategoryItemsResponse_Item{
			AuctionId:      item.AuctionID,
			Title:          item.Title,
			CurrentPrice:   item.CurrentPrice,
			ImmediatePrice: item.ImmediatePrice,
			BidCount:       item.BidCount,
			// Image は proto 定義にないかもしれないので確認が必要だが、
			// 既存の proto を見る限り GetCategoryItemsResponse_Item に Image フィールドは定義されていない。
			// GetAuctionResponse には Images があるが、CategoryItem には Image (single) がある。
			// proto定義を確認すると:
			/*
			  message Item {
			    string auction_id = 1;
			    string title = 2;
			    int64 current_price = 3;
			    int64 immediate_price = 4;
			    int64 bid_count = 6;
			  }
			*/
			// なので Image は含めない（またはproto修正が必要だが、今回はプランに含まれていないので従う）
		})
	}

	resp := &yahoo_auctionv1.GetCategoryItemsResponse{
		Items:      items,
		TotalCount: pageResult.TotalCount,
	}

	return connect.NewResponse(resp), nil
}
