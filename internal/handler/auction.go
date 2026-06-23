package handler

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	yahoo_auctionv1 "github.com/jo3qma/protobuf/gen/go/yahoo_auction/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"jo3qma.com/yahoo_auctions/internal/domain/model"
	"jo3qma.com/yahoo_auctions/internal/infrastructure/yahoo"
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

// SearchGetter は検索ユースケースの最小インターフェースです。
type SearchGetter interface {
	Search(ctx context.Context, query string, page int64) (*model.CategoryItemsPage, error)
}

// ComparableGetter は Comparable 検索ユースケースの最小インターフェースです。
type ComparableGetter interface {
	SearchComparables(ctx context.Context, categoryID, keyword string, lookbackDays *int32) (*model.ComparableSearchResult, error)
}

// AuctionHandler はgRPC/Connectのハンドラー実装です
// プロトコル層（protobuf）とドメイン層（usecase）を橋渡しします
type AuctionHandler struct {
	uc           AuctionGetter
	catUC        CategoryGetter
	searchUC     SearchGetter
	comparableUC ComparableGetter
}

// NewAuctionHandler は新しいAuctionHandlerインスタンスを作成します
func NewAuctionHandler(uc AuctionGetter, catUC CategoryGetter, searchUC SearchGetter, comparableUC ComparableGetter) *AuctionHandler {
	return &AuctionHandler{
		uc:           uc,
		catUC:        catUC,
		searchUC:     searchUC,
		comparableUC: comparableUC,
	}
}

// upstreamErrorToConnectCode はアップストリームのHTTPステータスに応じてConnectのエラーコードを返します。
func upstreamErrorToConnectCode(err error) connect.Code {
	var ue *yahoo.UpstreamError
	if errors.As(err, &ue) && ue.StatusCode == 404 {
		return connect.CodeNotFound
	}
	return connect.CodeInternal
}

// GetAuction はオークション商品情報を取得するRPCハンドラーです
func (h *AuctionHandler) GetAuction(
	ctx context.Context,
	req *connect.Request[yahoo_auctionv1.GetAuctionRequest],
) (*connect.Response[yahoo_auctionv1.GetAuctionResponse], error) {
	// ユースケースを呼び出して商品情報を取得
	item, err := h.uc.GetAuction(ctx, req.Msg.AuctionId)
	if err != nil {
		return nil, connect.NewError(upstreamErrorToConnectCode(err), err)
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
		return nil, connect.NewError(upstreamErrorToConnectCode(err), err)
	}

	// protoへの変換
	items := make([]*yahoo_auctionv1.GetCategoryItemsResponse_Item, 0, len(pageResult.Items))
	for _, item := range pageResult.Items {
		items = append(items, &yahoo_auctionv1.GetCategoryItemsResponse_Item{
			AuctionId:      item.AuctionID,
			Title:          item.Title,
			CurrentPrice:   item.CurrentPrice,
			ImmediatePrice: item.ImmediatePrice,
			Image:          item.Image,
			BidCount:       item.BidCount,
		})
	}

	resp := &yahoo_auctionv1.GetCategoryItemsResponse{
		Items:      items,
		TotalCount: pageResult.TotalCount,
	}

	return connect.NewResponse(resp), nil
}

// SearchAuctions はキーワード検索で商品一覧を取得するRPCハンドラーです（新着順）
func (h *AuctionHandler) SearchAuctions(
	ctx context.Context,
	req *connect.Request[yahoo_auctionv1.SearchAuctionsRequest],
) (*connect.Response[yahoo_auctionv1.SearchAuctionsResponse], error) {
	pageResult, err := h.searchUC.Search(ctx, req.Msg.Query, req.Msg.Page)
	if err != nil {
		var ue *yahoo.UpstreamError
		if errors.As(err, &ue) && ue.StatusCode == 404 {
			// ヤフオクは「一致する商品はありません」のときに404を返す。検索0件として200で空結果を返す。
			return connect.NewResponse(&yahoo_auctionv1.SearchAuctionsResponse{
				Items:      []*yahoo_auctionv1.SearchAuctionsResponse_Item{},
				TotalCount: 0,
			}), nil
		}
		return nil, connect.NewError(upstreamErrorToConnectCode(err), err)
	}

	items := make([]*yahoo_auctionv1.SearchAuctionsResponse_Item, 0, len(pageResult.Items))
	for _, item := range pageResult.Items {
		items = append(items, &yahoo_auctionv1.SearchAuctionsResponse_Item{
			AuctionId:      item.AuctionID,
			Title:          item.Title,
			CurrentPrice:   item.CurrentPrice,
			ImmediatePrice: item.ImmediatePrice,
			Image:          item.Image,
			BidCount:       item.BidCount,
		})
	}

	resp := &yahoo_auctionv1.SearchAuctionsResponse{
		Items:      items,
		TotalCount: pageResult.TotalCount,
	}

	return connect.NewResponse(resp), nil
}

// SearchComparables は Sold search から Comparable を検索する RPC ハンドラーです。
func (h *AuctionHandler) SearchComparables(
	ctx context.Context,
	req *connect.Request[yahoo_auctionv1.SearchComparablesRequest],
) (*connect.Response[yahoo_auctionv1.SearchComparablesResponse], error) {
	result, err := h.comparableUC.SearchComparables(
		ctx,
		req.Msg.CategoryId,
		req.Msg.IdentityFieldValue,
		req.Msg.LookbackDays,
	)
	if err != nil {
		return nil, connect.NewError(upstreamErrorToConnectCode(err), err)
	}

	comparables := make([]*yahoo_auctionv1.Comparable, 0, len(result.Comparables))
	for _, c := range result.Comparables {
		row := &yahoo_auctionv1.Comparable{
			AuctionId:    c.AuctionID,
			Title:        c.Title,
			WinningPrice: c.WinningPrice,
		}
		if !c.EndedAt.IsZero() {
			row.EndedAt = timestamppb.New(c.EndedAt)
		}
		comparables = append(comparables, row)
	}

	resp := &yahoo_auctionv1.SearchComparablesResponse{
		Comparables: comparables,
		Count:       result.Count,
	}

	return connect.NewResponse(resp), nil
}
