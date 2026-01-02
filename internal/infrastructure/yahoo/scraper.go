package yahoo

import (
	"context"
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
func (s *yahooScraper) extractItemInfo(doc *goquery.Document, auctionID string) (*model.Item, error) {
	item := &model.Item{
		AuctionID: auctionID,
		Status:    model.StatusUnspecified,
	}

	// タイトル取得
	// ヤフオクの商品ページのタイトルセレクタ（実際のHTML構造に合わせて調整が必要）
	title := strings.TrimSpace(doc.Find("h1.ProductTitle__text, h1.yaProductTitle, h1").First().Text())
	if title == "" {
		// フォールバック: titleタグから取得
		title = doc.Find("title").Text()
		// " - ヤフオク!"などの接尾辞を除去
		title = strings.TrimSuffix(title, " - ヤフオク!")
		title = strings.TrimSpace(title)
	}
	item.Title = title

	if item.Title == "" {
		return nil, fmt.Errorf("item title not found - page structure may have changed")
	}

	// 現在価格取得
	priceText := strings.TrimSpace(doc.Find(".Price__value, .yaPrice, .price").First().Text())
	if priceText == "" {
		// フォールバック: 価格を含む可能性のあるテキストを検索
		doc.Find("*").Each(func(i int, s *goquery.Selection) {
			text := s.Text()
			if strings.Contains(text, "円") && strings.Contains(text, "現在価格") {
				priceText = text
			}
		})
	}
	item.CurrentPrice = parsePrice(priceText)

	// 送料取得
	shippingText := strings.TrimSpace(doc.Find(".ShippingFee__value, .shippingFee, .shipping").First().Text())
	if shippingText == "" {
		// フォールバック: 送料を含む可能性のあるテキストを検索
		doc.Find("*").Each(func(i int, s *goquery.Selection) {
			text := s.Text()
			if strings.Contains(text, "送料") || strings.Contains(text, "配送料") {
				shippingText = text
			}
		})
	}
	item.ShippingFee = parsePrice(shippingText)

	// オークション状態の判定
	// より正確な判定のため、特定のキーワードとHTML要素を確認
	pageText := doc.Text()
	lowerText := strings.ToLower(pageText)

	// キャンセル状態の判定（最も明確なキーワードから判定）
	if strings.Contains(lowerText, "キャンセル") || strings.Contains(lowerText, "取り消し") {
		item.Status = model.StatusCanceled
	} else if s.isAuctionFinished(doc, lowerText) {
		// 終了済みの判定（「終了済み」「落札済み」などの特定キーワードを確認）
		item.Status = model.StatusFinished
	} else {
		// デフォルトは出品中（「入札する」ボタンが存在する場合は出品中）
		item.Status = model.StatusActive
	}

	return item, nil
}

// isAuctionFinished はオークションが終了済みかどうかを判定します
// 「終了日時」や「終了予定」などの文字列は除外し、実際に終了したことを示す
// キーワードのみを確認します
func (s *yahooScraper) isAuctionFinished(doc *goquery.Document, lowerText string) bool {
	// 「入札する」ボタンが存在する場合は出品中
	if doc.Find("button, a").FilterFunction(func(i int, s *goquery.Selection) bool {
		text := strings.ToLower(s.Text())
		return strings.Contains(text, "入札する") || strings.Contains(text, "入札")
	}).Length() > 0 {
		return false
	}

	// 終了済みを示す特定のキーワードを確認
	// 「終了日時」「終了予定」は除外し、「終了済み」「落札済み」などのみを確認
	finishedKeywords := []string{
		"終了済み",
		"落札済み",
		"このオークションは終了しました",
		"オークションは終了しました",
		"終了しました",
		"落札されました",
		"時間切れ",
	}

	for _, keyword := range finishedKeywords {
		if strings.Contains(lowerText, strings.ToLower(keyword)) {
			return true
		}
	}

	return false
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
