package yahoo

import (
	"context"
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

	// パース
	return s.extractCategoryItems(doc)
}

func (s *yahooCategoryScraper) extractCategoryItems(doc *goquery.Document) (*model.CategoryItemsPage, error) {
	var items []*model.CategoryItem

	// 商品一覧: div.Products__list ul.Products__items li.Product
	doc.Find("div.Products__list ul.Products__items li.Product").Each(func(i int, s *goquery.Selection) {
		item := &model.CategoryItem{}

		// タイトル: h3.Product__title a.Product__titleLink
		titleLink := s.Find("h3.Product__title a.Product__titleLink")
		item.Title = strings.TrimSpace(titleLink.Text())

		// オークションID: a.Product__titleLink (data-auction-id)
		if id, exists := titleLink.Attr("data-auction-id"); exists {
			item.AuctionID = id
		}

		// 画像: div.Products__list ul.Products__items li.Product img.Product__imageData
		// src属性を取得。遅延ロードなどで src がダミーの場合、data-src 等を見る必要があるかもしれないが、
		// @Untitled-1 の指定通りまずは普通に取得する。
		// img.Product__imageData
		img := s.Find("img.Product__imageData")
		if src, exists := img.Attr("src"); exists {
			item.Image = src
		} else if src, exists := img.Attr("data-src"); exists {
			// fallback
			item.Image = src
		}

		// 価格情報: div.Product__priceInfo
		priceInfo := s.Find("div.Product__priceInfo")

		// 現在の価格: span.Product__price (1つ目)
		currentPriceEl := priceInfo.Find("span.Product__price").First().Find("span.Product__priceValue")
		item.CurrentPrice = parsePrice(currentPriceEl.Text())

		// 即決価格: span.Product__price (2つ目)
		// 存在しない場合もある
		prices := priceInfo.Find("span.Product__price")
		if prices.Length() > 1 {
			immediatePriceEl := prices.Eq(1).Find("span.Product__priceValue")
			item.ImmediatePrice = parsePrice(immediatePriceEl.Text())
		}

		// 入札数: dd.Product__bid
		bidEl := s.Find("dd.Product__bid")
		item.BidCount = parseCount(bidEl.Text())

		items = append(items, item)
	})

	// 商品の総数: div.Result__header > div.SearchMode > div.Tab > ul > li.Tab__item.Tab__item--current > div > span.Tab__subText
	totalCountStr := doc.Find("div.Result__header div.SearchMode div.Tab ul li.Tab__item--current div span.Tab__subText").Text()
	totalCount := parseCount(totalCountStr)

	return &model.CategoryItemsPage{
		Items:      items,
		TotalCount: totalCount,
		HasNext:    len(items) >= 50, // 簡易判定
	}, nil
}
