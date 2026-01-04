package yahoo

import (
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"jo3qma.com/yahoo_auctions/internal/domain/model"
)

func TestYahooScraper_parseNextData_returnsErrorWhenScriptMissing(t *testing.T) {
	t.Parallel()

	s := &yahooScraper{}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader("<html><head></head><body></body></html>"))
	if err != nil {
		t.Fatalf("failed to build doc: %v", err)
	}

	_, err = s.parseNextData(doc)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestYahooScraper_parseNextData_returnsErrorWhenJSONInvalid(t *testing.T) {
	t.Parallel()

	s := &yahooScraper{}
	html := `<html><head><script id="__NEXT_DATA__">{invalid json}</script></head><body></body></html>`
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("failed to build doc: %v", err)
	}

	_, err = s.parseNextData(doc)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestYahooScraper_extractItemFromJSON_mapsFields(t *testing.T) {
	t.Parallel()

	s := &yahooScraper{}
	auctionID := "x1234567890"

	data := &NextData{}
	item := &data.Props.PageProps.InitialState.Item.Detail.Item
	item.Title = "title"
	item.Price = 100
	item.TaxinPrice = 1234
	item.Status = "open"
	item.DescriptionHtml = "<p>desc</p>"
	item.InitPrice = 1
	item.TaxinStartPrice = 200
	item.StartTime = "2025-12-29T16:00:10+09:00"
	item.EndTime = "2025-12-30T16:00:10+09:00"
	item.IsEarlyClosing = true
	item.IsAutomaticExtension = false
	item.ItemReturnable.Allowed = true
	item.ItemReturnable.Comment = "detail"
	item.Img = []struct {
		Image  string `json:"image"`
		Width  int    `json:"width"`
		Height int    `json:"height"`
	}{
		{Image: "https://example.com/1.jpg", Width: 1, Height: 1},
		{Image: "https://example.com/1.jpg", Width: 1, Height: 1}, // duplicate
		{Image: "https://example.com/2.jpg", Width: 1, Height: 1},
	}

	got := s.extractItemFromJSON(data, auctionID)
	if got.AuctionID != auctionID {
		t.Fatalf("AuctionID got %q, want %q", got.AuctionID, auctionID)
	}
	if got.Title != "title" {
		t.Fatalf("Title got %q, want %q", got.Title, "title")
	}
	if got.Description != "<p>desc</p>" {
		t.Fatalf("Description got %q, want %q", got.Description, "<p>desc</p>")
	}
	if got.CurrentPrice != 1234 {
		t.Fatalf("CurrentPrice got %d, want %d", got.CurrentPrice, 1234)
	}
	if got.Status != model.StatusActive {
		t.Fatalf("Status got %v, want %v", got.Status, model.StatusActive)
	}
	if len(got.Images) != 2 {
		t.Fatalf("Images len got %d, want %d", len(got.Images), 2)
	}
	if got.Images[0] != "https://example.com/1.jpg" || got.Images[1] != "https://example.com/2.jpg" {
		t.Fatalf("Images got %#v, want [%q %q]", got.Images, "https://example.com/1.jpg", "https://example.com/2.jpg")
	}

	if got.AuctionInfo == nil {
		t.Fatalf("AuctionInfo is nil")
	}
	if got.AuctionInfo.StartPrice != 200 {
		t.Fatalf("AuctionInfo.StartPrice got %d, want %d", got.AuctionInfo.StartPrice, 200)
	}
	if got.AuctionInfo.EarlyEnd != true {
		t.Fatalf("AuctionInfo.EarlyEnd got %v, want %v", got.AuctionInfo.EarlyEnd, true)
	}
	if got.AuctionInfo.AutoExtension != false {
		t.Fatalf("AuctionInfo.AutoExtension got %v, want %v", got.AuctionInfo.AutoExtension, false)
	}
	if got.AuctionInfo.Returnable != true {
		t.Fatalf("AuctionInfo.Returnable got %v, want %v", got.AuctionInfo.Returnable, true)
	}
	if got.AuctionInfo.ReturnableDetail != "detail" {
		t.Fatalf("AuctionInfo.ReturnableDetail got %q, want %q", got.AuctionInfo.ReturnableDetail, "detail")
	}

	wantStart, err := time.Parse(time.RFC3339, "2025-12-29T16:00:10+09:00")
	if err != nil {
		t.Fatalf("failed to parse start time: %v", err)
	}
	wantEnd, err := time.Parse(time.RFC3339, "2025-12-30T16:00:10+09:00")
	if err != nil {
		t.Fatalf("failed to parse end time: %v", err)
	}
	if got.AuctionInfo.StartTime != wantStart {
		t.Fatalf("AuctionInfo.StartTime got %v, want %v", got.AuctionInfo.StartTime, wantStart)
	}
	if got.AuctionInfo.EndTime != wantEnd {
		t.Fatalf("AuctionInfo.EndTime got %v, want %v", got.AuctionInfo.EndTime, wantEnd)
	}
}

func TestYahooScraper_extractItemFromJSON_statusMapping(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		status string
		want   model.Status
	}{
		{name: "open", status: "open", want: model.StatusActive},
		{name: "closed", status: "closed", want: model.StatusFinished},
		{name: "cancel", status: "cancel", want: model.StatusCanceled},
		{name: "canceled", status: "canceled", want: model.StatusCanceled},
		{name: "unknown", status: "???", want: model.StatusUnspecified},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := &yahooScraper{}
			data := &NextData{}
			data.Props.PageProps.InitialState.Item.Detail.Item.Status = tc.status

			got := s.extractItemFromJSON(data, "x1234567890")
			if got.Status != tc.want {
				t.Fatalf("Status got %v, want %v", got.Status, tc.want)
			}
		})
	}
}

func TestYahooScraper_extractItemFromJSON_priceFallback(t *testing.T) {
	t.Parallel()

	s := &yahooScraper{}
	data := &NextData{}
	item := &data.Props.PageProps.InitialState.Item.Detail.Item
	item.Price = 999
	item.TaxinPrice = 0

	got := s.extractItemFromJSON(data, "x1234567890")
	if got.CurrentPrice != 999 {
		t.Fatalf("CurrentPrice got %d, want %d", got.CurrentPrice, 999)
	}
}

func TestYahooScraper_extractItemFromJSON_startPriceFallback(t *testing.T) {
	t.Parallel()

	s := &yahooScraper{}
	data := &NextData{}
	item := &data.Props.PageProps.InitialState.Item.Detail.Item
	item.InitPrice = 111
	item.TaxinStartPrice = 0

	got := s.extractItemFromJSON(data, "x1234567890")
	if got.AuctionInfo == nil {
		t.Fatalf("AuctionInfo is nil")
	}
	if got.AuctionInfo.StartPrice != 111 {
		t.Fatalf("AuctionInfo.StartPrice got %d, want %d", got.AuctionInfo.StartPrice, 111)
	}
}

func TestYahooScraper_extractItemFromJSON_timeParseFailureLeavesZero(t *testing.T) {
	t.Parallel()

	s := &yahooScraper{}
	data := &NextData{}
	item := &data.Props.PageProps.InitialState.Item.Detail.Item
	item.StartTime = "not-a-time"
	item.EndTime = "also-not-a-time"

	got := s.extractItemFromJSON(data, "x1234567890")
	if got.AuctionInfo == nil {
		t.Fatalf("AuctionInfo is nil")
	}
	if !got.AuctionInfo.StartTime.IsZero() {
		t.Fatalf("StartTime got %v, want zero", got.AuctionInfo.StartTime)
	}
	if !got.AuctionInfo.EndTime.IsZero() {
		t.Fatalf("EndTime got %v, want zero", got.AuctionInfo.EndTime)
	}
}
