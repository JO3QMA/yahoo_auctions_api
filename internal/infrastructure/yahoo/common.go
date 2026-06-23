package yahoo

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"jo3qma.com/yahoo_auctions/internal/domain/model"
)

// fetchHTML は指定されたURLからHTMLを取得してgoquery.Documentを返します
// 共通のUser-Agent設定やエラーハンドリングを行います
func fetchHTML(ctx context.Context, client *http.Client, url string) (*goquery.Document, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 一般的なブラウザに見せかけるUser-Agent
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer func() {
		if closeErr := res.Body.Close(); closeErr != nil {
			fmt.Printf("warning: failed to close response body: %v\n", closeErr)
		}
	}()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch page: status %d: %w", res.StatusCode, &UpstreamError{StatusCode: res.StatusCode})
	}

	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	return doc, nil
}

// parsePrice は "1,000円" などの文字列から数値を抽出します
func parsePrice(s string) int64 {
	// 数字のみ抽出
	re := regexp.MustCompile(`[0-9]+`)
	matches := re.FindAllString(s, -1)
	if len(matches) == 0 {
		return 0
	}
	// 結合してパース
	valStr := strings.Join(matches, "")
	val, err := strconv.ParseInt(valStr, 10, 64)
	if err != nil {
		return 0
	}
	return val
}

// parseCount は "1,000件" などの文字列から数値を抽出します
func parseCount(s string) int64 {
	return parsePrice(s) // 実装は同じでOK
}

// ExtractProductList は商品一覧ページのHTMLから商品リストを抽出します。
// カテゴリ一覧・検索結果の両方で同じDOM構造（div.Products__list ul.Products__items li.Product）を前提とします。
func ExtractProductList(doc *goquery.Document) (*model.CategoryItemsPage, error) {
	var items []*model.CategoryItem

	// 商品一覧: div.Products__list ul.Products__items li.Product
	doc.Find("div.Products__list ul.Products__items li.Product").Each(func(i int, sel *goquery.Selection) {
		item := &model.CategoryItem{}

		// タイトル: h3.Product__title a.Product__titleLink
		titleLink := sel.Find("h3.Product__title a.Product__titleLink")
		item.Title = strings.TrimSpace(titleLink.Text())

		// オークションID: a.Product__titleLink (data-auction-id)
		if id, exists := titleLink.Attr("data-auction-id"); exists {
			item.AuctionID = id
		}

		// 画像: img.Product__imageData
		img := sel.Find("img.Product__imageData")
		if src, exists := img.Attr("src"); exists {
			item.Image = src
		} else if src, exists := img.Attr("data-src"); exists {
			item.Image = src
		}

		// 価格情報: div.Product__priceInfo
		priceInfo := sel.Find("div.Product__priceInfo")
		currentPriceEl := priceInfo.Find("span.Product__price").First().Find("span.Product__priceValue")
		item.CurrentPrice = parsePrice(currentPriceEl.Text())

		// 即決価格: span.Product__price (2つ目)
		prices := priceInfo.Find("span.Product__price")
		if prices.Length() > 1 {
			immediatePriceEl := prices.Eq(1).Find("span.Product__priceValue")
			item.ImmediatePrice = parsePrice(immediatePriceEl.Text())
		}

		// 入札数: dd.Product__bid
		bidEl := sel.Find("dd.Product__bid")
		item.BidCount = parseCount(bidEl.Text())

		items = append(items, item)
	})

	// 商品の総数: div.Result__header div.SearchMode div.Tab ul li.Tab__item--current div span.Tab__subText
	totalCountStr := doc.Find("div.Result__header div.SearchMode div.Tab ul li.Tab__item--current div span.Tab__subText").Text()
	totalCount := parseCount(totalCountStr)

	return &model.CategoryItemsPage{
		Items:      items,
		TotalCount: totalCount,
		HasNext:    len(items) >= 50,
	}, nil
}

// extractNextDataJSON は __NEXT_DATA__ スクリプトタグの JSON 本文を返します。
func extractNextDataJSON(doc *goquery.Document) ([]byte, error) {
	raw := strings.TrimSpace(doc.Find(`script#__NEXT_DATA__`).First().Text())
	if raw == "" {
		return nil, fmt.Errorf("failed to parse __NEXT_DATA__: not found")
	}
	return []byte(raw), nil
}
