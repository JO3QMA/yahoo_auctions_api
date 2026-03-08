package yahoo

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"jo3qma.com/yahoo_auctions/internal/domain/model"
	"jo3qma.com/yahoo_auctions/internal/domain/repository"
)

type yahooCategoryScraper struct {
	client  *http.Client
	baseURL string
}

// NewYahooCategoryScraper は新しいCategoryItemRepositoryの実装を作成します
func NewYahooCategoryScraper() repository.CategoryItemRepository {
	return &yahooCategoryScraper{
		client:  &http.Client{Timeout: 30 * time.Second},
		baseURL: "https://auctions.yahoo.co.jp",
	}
}

func (s *yahooCategoryScraper) FetchByCategory(ctx context.Context, categoryID string, page int64) (*model.CategoryItemsPage, error) {
	// URL構築
	// 例: https://auctions.yahoo.co.jp/category/list/{categoryID}/?p=&auccat={categoryID}&is_postage_mode=1&dest_pref_code=27&b={offset}&n=50&s1=new&o1=d

	// b (offset) の計算: (1ページあたりの商品数 * (ページ番号)) + 1
	// pageは0始まりとする仕様なので、0ページ目は 1, 1ページ目は 51
	const itemsPerPage = 50
	offset := (itemsPerPage * page) + 1

	u, err := url.Parse(fmt.Sprintf("%s/category/list/%s/", s.baseURL, categoryID))
	if err != nil {
		return nil, fmt.Errorf("invalid base url: %w", err)
	}

	q := u.Query()
	q.Set("auccat", categoryID)
	q.Set("is_postage_mode", "1")
	q.Set("dest_pref_code", "27")
	q.Set("b", strconv.FormatInt(offset, 10))
	q.Set("n", strconv.FormatInt(int64(itemsPerPage), 10))
	q.Set("s1", "new")
	q.Set("o1", "d")
	// p (検索ワード) は指定しない

	u.RawQuery = q.Encode()
	targetURL := u.String()

	// 共通関数でHTML取得
	doc, err := fetchHTML(ctx, s.client, targetURL)
	if err != nil {
		return nil, err
	}

	// パース（共通のExtractProductListを使用）
	return ExtractProductList(doc)
}
