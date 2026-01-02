package yahoo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/PuerkitoBio/goquery"
	"jo3qma.com/yahoo_auctions/internal/domain/model"
	"jo3qma.com/yahoo_auctions/internal/domain/repository"
)

// yahooScraper はヤフオクのHTMLをスクレイピングして商品情報を取得する実装です
// 腐敗防止層（Anti-Corruption Layer）として、外部システムの不安定な構造を
// ドメインモデルに変換する責務を持ちます
type yahooScraper struct {
	client  *http.Client
	baseURL string
}

// NewYahooScraper は新しいYahooScraperインスタンスを作成します
func NewYahooScraper() repository.ItemRepository {
	return newYahooScraper(
		&http.Client{Timeout: 30 * time.Second},
		"https://page.auctions.yahoo.co.jp",
	)
}

// newYahooScraper はテスト容易性のための内部コンストラクタです。
// 本番コードは NewYahooScraper を利用し、テストでは http.Client/baseURL を注入します。
func newYahooScraper(client *http.Client, baseURL string) repository.ItemRepository {
	return &yahooScraper{
		client:  client,
		baseURL: baseURL,
	}
}

// FetchByID は指定されたオークションIDから商品情報を取得します
func (s *yahooScraper) FetchByID(ctx context.Context, auctionID string) (*model.Item, error) {
	// オークションIDからURLを構築
	url := fmt.Sprintf("%s/jp/auction/%s", s.baseURL, auctionID)

	// 共通関数でHTML取得
	doc, err := fetchHTML(ctx, s.client, url)
	if err != nil {
		return nil, err
	}

	// HTMLから商品情報を抽出
	item, err := s.extractItemInfo(doc, auctionID)
	if err != nil {
		return nil, fmt.Errorf("failed to extract item info: %w", err)
	}

	return item, nil
}

// extractItemInfo はHTMLドキュメントから商品情報を抽出します
// Next.jsのJSONデータを優先して使用し、取得できない場合はエラーを返します
func (s *yahooScraper) extractItemInfo(doc *goquery.Document, auctionID string) (*model.Item, error) {
	// JSONデータをパース
	nextData, err := s.parseNextData(doc)
	if err != nil {
		return nil, fmt.Errorf("failed to parse next data: %w", err)
	}

	// JSONからモデルへのマッピング
	item := s.extractItemFromJSON(nextData, auctionID)
	return item, nil
}

// NextData はNext.jsのJSON構造体です
type NextData struct {
	Props struct {
		PageProps struct {
			InitialState struct {
				Item struct {
					Detail struct {
						Item struct {
							Title                string `json:"title"`
							Price                int64  `json:"price"`
							TaxinPrice           int64  `json:"taxinPrice"`
							Status               string `json:"status"`
							DescriptionHtml      string `json:"descriptionHtml"`
							InitPrice            int64  `json:"initPrice"`
							TaxinStartPrice      int64  `json:"taxinStartPrice"`
							StartTime            string `json:"startTime"` // ISO 8601
							EndTime              string `json:"endTime"`   // ISO 8601
							IsEarlyClosing       bool   `json:"isEarlyClosing"`
							IsAutomaticExtension bool   `json:"isAutomaticExtension"`
							ItemReturnable       struct {
								Allowed bool   `json:"allowed"`
								Comment string `json:"comment"`
							} `json:"itemReturnable"`
							Img []struct {
								Image  string `json:"image"`
								Width  int    `json:"width"`
								Height int    `json:"height"`
							} `json:"img"`
						} `json:"item"`
					} `json:"detail"`
				} `json:"item"`
			} `json:"initialState"`
		} `json:"pageProps"`
	} `json:"props"`
}

// parseNextData はHTMLからNext.jsのJSONデータを抽出・パースします
func (s *yahooScraper) parseNextData(doc *goquery.Document) (*NextData, error) {
	scriptContent := doc.Find("script#__NEXT_DATA__").Text()
	if scriptContent == "" {
		return nil, fmt.Errorf("next data script not found")
	}

	var data NextData
	if err := json.Unmarshal([]byte(scriptContent), &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal next data: %w", err)
	}

	return &data, nil
}

// extractItemFromJSON はNextDataからドメインモデルのItemを構築します
func (s *yahooScraper) extractItemFromJSON(data *NextData, auctionID string) *model.Item {
	itemData := data.Props.PageProps.InitialState.Item.Detail.Item

	item := &model.Item{
		AuctionID:   auctionID,
		Title:       itemData.Title,
		Description: itemData.DescriptionHtml,
		Images:      make([]string, 0, len(itemData.Img)),
	}

	// 価格
	if itemData.TaxinPrice > 0 {
		item.CurrentPrice = itemData.TaxinPrice
	} else {
		item.CurrentPrice = itemData.Price
	}

	// 画像
	seenURLs := make(map[string]bool)
	for _, img := range itemData.Img {
		if !seenURLs[img.Image] {
			item.Images = append(item.Images, img.Image)
			seenURLs[img.Image] = true
		}
	}

	// ステータス
	switch itemData.Status {
	case "open":
		item.Status = model.StatusActive
	case "closed":
		item.Status = model.StatusFinished
	case "cancel", "canceled":
		item.Status = model.StatusCanceled
	default:
		// 終了済みとみなされるケースを確認
		// 現在時刻と比較して終了していればFinishedとするなどのロジックも検討可能だが
		// JSONのstatusを信頼する
		item.Status = model.StatusUnspecified
	}

	// オークション情報
	info := &model.AuctionInformation{
		AuctionID:        auctionID,
		EarlyEnd:         itemData.IsEarlyClosing,
		AutoExtension:    itemData.IsAutomaticExtension,
		Returnable:       itemData.ItemReturnable.Allowed,
		ReturnableDetail: itemData.ItemReturnable.Comment,
	}

	// 開始価格
	if itemData.TaxinStartPrice > 0 {
		info.StartPrice = itemData.TaxinStartPrice
	} else {
		info.StartPrice = itemData.InitPrice
	}

	// 時間パース (ISO 8601形式: "2025-12-29T16:00:10+09:00")
	if t, err := time.Parse(time.RFC3339, itemData.StartTime); err == nil {
		info.StartTime = t
	}
	if t, err := time.Parse(time.RFC3339, itemData.EndTime); err == nil {
		info.EndTime = t
	}

	item.AuctionInfo = info
	return item
}
