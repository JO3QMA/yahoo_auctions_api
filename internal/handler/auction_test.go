package handler

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	yahoo_auctionv1 "github.com/jo3qma/protobuf/gen/go/yahoo_auction/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
	"jo3qma.com/yahoo_auctions/internal/domain/model"
)

type fakeAuctionGetter struct {
	item *model.Item
	err  error
}

func (f fakeAuctionGetter) GetAuction(ctx context.Context, auctionID string) (*model.Item, error) {
	return f.item, f.err
}

type fakeCategoryGetter struct {
	page *model.CategoryItemsPage
	err  error
}

func (f fakeCategoryGetter) GetCategoryItems(ctx context.Context, categoryID string, page int64) (*model.CategoryItemsPage, error) {
	return f.page, f.err
}

func TestAuctionHandler_GetAuction_mapsDomainToProto(t *testing.T) {
	t.Parallel()

	start := time.Date(2025, 12, 29, 16, 0, 10, 0, time.FixedZone("JST", 9*60*60))
	end := time.Date(2025, 12, 30, 16, 0, 10, 0, time.FixedZone("JST", 9*60*60))

	item := &model.Item{
		AuctionID:    "x1234567890",
		Title:        "title",
		CurrentPrice: 1234,
		Status:       model.StatusActive,
		Images:       []string{"https://example.com/1.jpg", "https://example.com/2.jpg"},
		Description:  "<p>desc</p>",
		AuctionInfo: &model.AuctionInformation{
			AuctionID:        "x1234567890",
			StartPrice:       100,
			StartTime:        start,
			EndTime:          end,
			EarlyEnd:         true,
			AutoExtension:    false,
			Returnable:       true,
			ReturnableDetail: "detail",
		},
	}

	h := NewAuctionHandler(fakeAuctionGetter{item: item}, nil)

	req := connect.NewRequest(&yahoo_auctionv1.GetAuctionRequest{AuctionId: item.AuctionID})
	resp, err := h.GetAuction(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Msg.AuctionId != item.AuctionID {
		t.Fatalf("AuctionId got %q, want %q", resp.Msg.AuctionId, item.AuctionID)
	}
	if resp.Msg.Title != item.Title {
		t.Fatalf("Title got %q, want %q", resp.Msg.Title, item.Title)
	}
	if resp.Msg.CurrentPrice != item.CurrentPrice {
		t.Fatalf("CurrentPrice got %d, want %d", resp.Msg.CurrentPrice, item.CurrentPrice)
	}
	if resp.Msg.Status != yahoo_auctionv1.AuctionStatus(item.Status) {
		t.Fatalf("Status got %v, want %v", resp.Msg.Status, yahoo_auctionv1.AuctionStatus(item.Status))
	}
	if len(resp.Msg.Images) != len(item.Images) {
		t.Fatalf("Images len got %d, want %d", len(resp.Msg.Images), len(item.Images))
	}
	for i := range item.Images {
		if resp.Msg.Images[i] != item.Images[i] {
			t.Fatalf("Images[%d] got %q, want %q", i, resp.Msg.Images[i], item.Images[i])
		}
	}
	if resp.Msg.Description != item.Description {
		t.Fatalf("Description got %q, want %q", resp.Msg.Description, item.Description)
	}

	if resp.Msg.AuctionInformation == nil {
		t.Fatalf("AuctionInformation is nil")
	}
	if resp.Msg.AuctionInformation.AuctionId != item.AuctionInfo.AuctionID {
		t.Fatalf("AuctionInformation.AuctionId got %q, want %q", resp.Msg.AuctionInformation.AuctionId, item.AuctionInfo.AuctionID)
	}
	if resp.Msg.AuctionInformation.StartPrice != item.AuctionInfo.StartPrice {
		t.Fatalf("AuctionInformation.StartPrice got %d, want %d", resp.Msg.AuctionInformation.StartPrice, item.AuctionInfo.StartPrice)
	}
	if resp.Msg.AuctionInformation.EarlyEnd != item.AuctionInfo.EarlyEnd {
		t.Fatalf("AuctionInformation.EarlyEnd got %v, want %v", resp.Msg.AuctionInformation.EarlyEnd, item.AuctionInfo.EarlyEnd)
	}
	if resp.Msg.AuctionInformation.AutoExtension != item.AuctionInfo.AutoExtension {
		t.Fatalf("AuctionInformation.AutoExtension got %v, want %v", resp.Msg.AuctionInformation.AutoExtension, item.AuctionInfo.AutoExtension)
	}
	if resp.Msg.AuctionInformation.Returnable != item.AuctionInfo.Returnable {
		t.Fatalf("AuctionInformation.Returnable got %v, want %v", resp.Msg.AuctionInformation.Returnable, item.AuctionInfo.Returnable)
	}
	if resp.Msg.AuctionInformation.ReturnableDetail != item.AuctionInfo.ReturnableDetail {
		t.Fatalf("AuctionInformation.ReturnableDetail got %q, want %q", resp.Msg.AuctionInformation.ReturnableDetail, item.AuctionInfo.ReturnableDetail)
	}

	wantStart := timestamppb.New(start)
	wantEnd := timestamppb.New(end)

	if resp.Msg.AuctionInformation.StartTime == nil || resp.Msg.AuctionInformation.StartTime.AsTime() != wantStart.AsTime() {
		t.Fatalf("AuctionInformation.StartTime got %v, want %v", resp.Msg.AuctionInformation.StartTime, wantStart)
	}
	if resp.Msg.AuctionInformation.EndTime == nil || resp.Msg.AuctionInformation.EndTime.AsTime() != wantEnd.AsTime() {
		t.Fatalf("AuctionInformation.EndTime got %v, want %v", resp.Msg.AuctionInformation.EndTime, wantEnd)
	}
}

func TestAuctionHandler_GetAuction_returnsNotFoundOnUsecaseError(t *testing.T) {
	t.Parallel()

	h := NewAuctionHandler(fakeAuctionGetter{err: errors.New("not found")}, nil)
	req := connect.NewRequest(&yahoo_auctionv1.GetAuctionRequest{AuctionId: "x1234567890"})
	_, err := h.GetAuction(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error")
	}

	var ce *connect.Error
	if !errors.As(err, &ce) {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if ce.Code() != connect.CodeNotFound {
		t.Fatalf("code got %v, want %v", ce.Code(), connect.CodeNotFound)
	}
}

func TestAuctionHandler_GetCategoryItems_mapsDomainToProto(t *testing.T) {
	t.Parallel()

	itemsPage := &model.CategoryItemsPage{
		Items: []*model.CategoryItem{
			{
				AuctionID:      "cat123",
				Title:          "Category Item 1",
				CurrentPrice:   1000,
				ImmediatePrice: 2000,
				BidCount:       5,
				Image:          "https://example.com/cat1.jpg",
			},
			{
				AuctionID:    "cat456",
				Title:        "Category Item 2",
				CurrentPrice: 500,
				BidCount:     0,
				Image:        "https://example.com/cat2.jpg",
			},
		},
		TotalCount: 100,
		HasNext:    true,
	}

	h := NewAuctionHandler(nil, fakeCategoryGetter{page: itemsPage})

	req := connect.NewRequest(&yahoo_auctionv1.GetCategoryItemsRequest{
		CategoryId: "2084261685",
		Page:       0,
	})

	resp, err := h.GetCategoryItems(context.Background(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.Msg.TotalCount != itemsPage.TotalCount {
		t.Fatalf("TotalCount got %d, want %d", resp.Msg.TotalCount, itemsPage.TotalCount)
	}

	if len(resp.Msg.Items) != len(itemsPage.Items) {
		t.Fatalf("Items len got %d, want %d", len(resp.Msg.Items), len(itemsPage.Items))
	}

	// Item 1
	if resp.Msg.Items[0].AuctionId != itemsPage.Items[0].AuctionID {
		t.Errorf("Item[0].AuctionId got %q, want %q", resp.Msg.Items[0].AuctionId, itemsPage.Items[0].AuctionID)
	}
	if resp.Msg.Items[0].Title != itemsPage.Items[0].Title {
		t.Errorf("Item[0].Title got %q, want %q", resp.Msg.Items[0].Title, itemsPage.Items[0].Title)
	}
	if resp.Msg.Items[0].CurrentPrice != itemsPage.Items[0].CurrentPrice {
		t.Errorf("Item[0].CurrentPrice got %d, want %d", resp.Msg.Items[0].CurrentPrice, itemsPage.Items[0].CurrentPrice)
	}
	if resp.Msg.Items[0].ImmediatePrice != itemsPage.Items[0].ImmediatePrice {
		t.Errorf("Item[0].ImmediatePrice got %d, want %d", resp.Msg.Items[0].ImmediatePrice, itemsPage.Items[0].ImmediatePrice)
	}
	if resp.Msg.Items[0].BidCount != itemsPage.Items[0].BidCount {
		t.Errorf("Item[0].BidCount got %d, want %d", resp.Msg.Items[0].BidCount, itemsPage.Items[0].BidCount)
	}

	// Item 2 (ImmediatePrice 0 check)
	if resp.Msg.Items[1].ImmediatePrice != 0 {
		t.Errorf("Item[1].ImmediatePrice got %d, want 0", resp.Msg.Items[1].ImmediatePrice)
	}
	if resp.Msg.Items[1].Image != itemsPage.Items[1].Image {
		t.Errorf("Item[1].Image got %q, want %q", resp.Msg.Items[1].Image, itemsPage.Items[1].Image)
	}
}

func TestAuctionHandler_GetCategoryItems_returnsErrorOnUsecaseError(t *testing.T) {
	t.Parallel()

	h := NewAuctionHandler(nil, fakeCategoryGetter{err: errors.New("internal error")})

	req := connect.NewRequest(&yahoo_auctionv1.GetCategoryItemsRequest{
		CategoryId: "2084261685",
		Page:       0,
	})

	_, err := h.GetCategoryItems(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error")
	}

	var ce *connect.Error
	if !errors.As(err, &ce) {
		t.Fatalf("expected *connect.Error, got %T: %v", err, err)
	}
	if ce.Code() != connect.CodeInternal {
		t.Fatalf("code got %v, want %v", ce.Code(), connect.CodeInternal)
	}
}
