package yahoo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"jo3qma.com/yahoo_auctions/internal/domain/model"
	"jo3qma.com/yahoo_auctions/internal/domain/repository"
)

const (
	comparableItemsPerPage = 50
	comparableSearchBase   = "https://auctions.yahoo.co.jp/closedsearch/closedsearch"
)

type yahooComparableScraper struct {
	client *http.Client
}

// NewYahooComparableScraper は ComparableRepository の実装を作成します。
func NewYahooComparableScraper() repository.ComparableRepository {
	return &yahooComparableScraper{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *yahooComparableScraper) SearchSold(ctx context.Context, categoryID, keyword string, page int64) (*model.ComparablePage, error) {
	offset := (comparableItemsPerPage * page) + 1

	u, err := url.Parse(comparableSearchBase)
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}

	q := u.Query()
	q.Set("p", keyword)
	if categoryID != "" && categoryID != "0" {
		q.Set("auccat", categoryID)
	}
	q.Set("b", strconv.FormatInt(offset, 10))
	q.Set("n", strconv.FormatInt(int64(comparableItemsPerPage), 10))
	u.RawQuery = strings.ReplaceAll(q.Encode(), "%20", "+")

	doc, err := fetchHTML(ctx, s.client, u.String())
	if err != nil {
		return nil, err
	}

	return extractComparablePage(doc)
}

type nextData struct {
	Props struct {
		PageProps struct {
			InitialState struct {
				Search struct {
					Items struct {
						Listing struct {
							Items []closedSearchListingItem `json:"items"`
						} `json:"listing"`
					} `json:"items"`
				} `json:"search"`
			} `json:"initialState"`
		} `json:"pageProps"`
	} `json:"props"`
}

type closedSearchListingItem struct {
	AuctionID string `json:"auctionId"`
	Title     string `json:"title"`
	Price     int64  `json:"price"`
	EndTime   string `json:"endTime"`
	BidCount  int64  `json:"bidCount"`
}

func extractComparablePage(doc *goquery.Document) (*model.ComparablePage, error) {
	raw, err := extractNextDataJSON(doc)
	if err != nil {
		return nil, err
	}

	var data nextData
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, fmt.Errorf("failed to parse __NEXT_DATA__: %w", err)
	}

	listingItems := data.Props.PageProps.InitialState.Search.Items.Listing.Items
	items := make([]*model.Comparable, 0, len(listingItems))
	for _, row := range listingItems {
		if row.AuctionID == "" || row.BidCount <= 0 || row.Price <= 0 {
			continue
		}

		endedAt, err := time.Parse(time.RFC3339, row.EndTime)
		if err != nil {
			continue
		}

		items = append(items, &model.Comparable{
			AuctionID:    row.AuctionID,
			Title:        row.Title,
			WinningPrice: row.Price,
			EndedAt:      endedAt,
		})
	}

	return &model.ComparablePage{
		Items:   items,
		HasNext: len(listingItems) >= comparableItemsPerPage,
	}, nil
}
