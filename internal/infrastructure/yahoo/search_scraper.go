package yahoo

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"jo3qma.com/yahoo_auctions/internal/domain/model"
	"jo3qma.com/yahoo_auctions/internal/domain/repository"
)

type yahooSearchScraper struct {
	client  *http.Client
	baseURL string
}

// NewYahooSearchScraper は新しいSearchItemRepositoryの実装を作成します
func NewYahooSearchScraper() repository.SearchItemRepository {
	return &yahooSearchScraper{
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: "https://auctions.yahoo.co.jp",
	}
}

func (s *yahooSearchScraper) Search(ctx context.Context, query string, page int64) (*model.CategoryItemsPage, error) {
	// URL構築: https://auctions.yahoo.co.jp/search/search?p={query}&b={offset}&n=50&s1=new&o1=d
	const itemsPerPage = 50
	offset := (itemsPerPage * page) + 1

	u, err := url.Parse(fmt.Sprintf("%s/search/search", s.baseURL))
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}

	q := u.Query()
	q.Set("p", query)
	q.Set("b", strconv.FormatInt(offset, 10))
	q.Set("n", strconv.FormatInt(int64(itemsPerPage), 10))
	q.Set("s1", "new") // 新着
	q.Set("o1", "d")   // 降順

	// ヤフオクはスペースを + で受け付ける。url.Values.Encode() は %20 になるため置換する。
	u.RawQuery = strings.ReplaceAll(q.Encode(), "%20", "+")
	targetURL := u.String()

	doc, err := fetchHTML(ctx, s.client, targetURL)
	if err != nil {
		return nil, err
	}

	return ExtractProductList(doc)
}
