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

	h := NewAuctionHandler(fakeAuctionGetter{item: item})

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

	h := NewAuctionHandler(fakeAuctionGetter{err: errors.New("not found")})
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
