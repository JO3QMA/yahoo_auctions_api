package yahoo

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"jo3qma.com/yahoo_auctions/internal/domain/model"
	"jo3qma.com/yahoo_auctions/internal/domain/repository"
)

// yahooScraper はヤフオクのHTMLをスクレイピングして商品情報を取得する実装です
// 腐敗防止層（Anti-Corruption Layer）として、外部システムの不安定な構造を
// ドメインモデルに変換する責務を持ちます
type yahooScraper struct {
	client *http.Client
}

// NewYahooScraper は新しいYahooScraperインスタンスを作成します
func NewYahooScraper() repository.ItemRepository {
	return &yahooScraper{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// FetchByID は指定されたオークションIDから商品情報を取得します
func (s *yahooScraper) FetchByID(ctx context.Context, auctionID string) (*model.Item, error) {
	// オークションIDからURLを構築
	url := fmt.Sprintf("https://page.auctions.yahoo.co.jp/jp/auction/%s", auctionID)

	// HTTPリクエストの作成
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 一般的なブラウザに見せかけるUser-Agent（ブロッキング回避のため）
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ja,en-US;q=0.9,en;q=0.8")

	// HTTPリクエストの実行
	res, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch page: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch page: status %d", res.StatusCode)
	}

	// goqueryでHTMLをパース
	doc, err := goquery.NewDocumentFromReader(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
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

// parseDateTime は日時文字列をtime.Timeに変換します
// 例: "2025年12月29日（月）16時0分" -> time.Time
func parseDateTime(text string) (time.Time, error) {
	if text == "" {
		return time.Time{}, fmt.Errorf("empty date string")
	}

	// 「（月）」などの曜日表記を除去
	text = regexp.MustCompile(`\s*\([^)]+\)\s*`).ReplaceAllString(text, " ")

	// 複数の日時形式を試行
	formats := []string{
		"2006年1月2日 15時4分",
		"2006年01月02日 15時04分",
		"2006年1月2日15時4分",
		"2006年01月02日15時04分",
		"2006年1月2日 15:04",
		"2006年01月02日 15:04",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, text); err == nil {
			return t, nil
		}
	}

	// 正規表現で抽出してからパース
	datePattern := regexp.MustCompile(`(\d{4})年(\d{1,2})月(\d{1,2})日[^\d]*(\d{1,2})時(\d{1,2})分`)
	matches := datePattern.FindStringSubmatch(text)
	if len(matches) == 6 {
		year := matches[1]
		month := matches[2]
		day := matches[3]
		hour := matches[4]
		minute := matches[5]

		// ゼロパディング
		if len(month) == 1 {
			month = "0" + month
		}
		if len(day) == 1 {
			day = "0" + day
		}
		if len(hour) == 1 {
			hour = "0" + hour
		}
		if len(minute) == 1 {
			minute = "0" + minute
		}

		dateStr := fmt.Sprintf("%s-%s-%s %s:%s:00", year, month, day, hour, minute)
		if t, err := time.Parse("2006-01-02 15:04:05", dateStr); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("failed to parse date: %s", text)
}

// parsePrice は価格文字列から数値を抽出します
// 例: "1,000円" -> 1000, "送料無料" -> 0
func parsePrice(text string) int64 {
	if text == "" {
		return 0
	}

	// 「無料」「送料無料」などの文字列をチェック
	lowerText := strings.ToLower(text)
	if strings.Contains(lowerText, "無料") || strings.Contains(lowerText, "free") {
		return 0
	}

	// 数字とカンマのみを抽出
	re := regexp.MustCompile(`([0-9,]+)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		// カンマを除去
		priceStr := strings.ReplaceAll(matches[1], ",", "")
		price, err := strconv.ParseInt(priceStr, 10, 64)
		if err == nil {
			return price
		}
	}

	return 0
}
