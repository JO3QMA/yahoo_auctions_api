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
	// 複数のセレクタを試行
	priceText := strings.TrimSpace(doc.Find(".Price__value, .yaPrice, .price, [class*='Price'], [class*='price']").First().Text())

	// フォールバック: ページ全体から正規表現で「現在」の後に続く価格を直接抽出
	if priceText == "" {
		pageText := doc.Text()
		// 「現在」の後に続く価格パターンを抽出（例: "現在 22,000円（税込）"）
		pricePattern := regexp.MustCompile(`現在\s*([0-9,]+)\s*円`)
		matches := pricePattern.FindStringSubmatch(pageText)
		if len(matches) > 1 {
			priceText = matches[0] // マッチした全体の文字列を使用
		}
	}

	// さらにフォールバック: 「現在」と「円」を含むテキストを検索
	if priceText == "" {
		var foundPriceText string
		doc.Find("*").Each(func(i int, s *goquery.Selection) {
			if foundPriceText != "" {
				return // 既に見つかった場合は終了
			}
			text := strings.TrimSpace(s.Text())
			// 「現在」と「円」を含み、数字も含むテキストを探す
			if strings.Contains(text, "現在") && strings.Contains(text, "円") {
				// 数字が含まれているか確認
				if regexp.MustCompile(`\d`).MatchString(text) {
					foundPriceText = text
				}
			}
		})
		if foundPriceText != "" {
			priceText = foundPriceText
		}
	}

	// 最後のフォールバック: ページ全体から「現在」と「円」を含む行を検索
	if priceText == "" {
		pageText := doc.Text()
		lines := strings.Split(pageText, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.Contains(line, "現在") && strings.Contains(line, "円") {
				if regexp.MustCompile(`\d`).MatchString(line) {
					priceText = line
					break
				}
			}
		}
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

	// 商品画像URL取得
	item.Images = s.extractImageURLs(doc)

	// 商品説明取得
	// まずNext.jsのデータ（JSON）から取得を試みる
	description := s.extractDescriptionFromJSON(doc)
	if description == "" {
		// フォールバック: div#description配下のsection直下のdivの中身を取得
		var htmlContent string
		htmlContent, _ = doc.Find("#description section > div").Html()
		description = strings.TrimSpace(htmlContent)
	}
	item.Description = description

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

	// オークション情報の抽出
	item.AuctionInfo = s.extractAuctionInformation(doc, auctionID)

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

// isThumbnailImage はURLがサムネイル画像かどうかを判定します
func (s *yahooScraper) isThumbnailImage(url string) bool {
	lowerURL := strings.ToLower(url)
	// サムネイル画像を示すキーワードをチェック
	thumbnailKeywords := []string{
		"thumb",
		"thumbnail",
		"_s",      // 小さいサイズ（例: s128x128）
		"_m",      // 中サイズ（例: m128x128）
		"_xs",     // 超小サイズ
		"/s/",     // サムネイルパス
		"/thumb/", // サムネイルディレクトリ
		"size=s",  // サイズパラメータ
		"size=m",  // サイズパラメータ
	}
	for _, keyword := range thumbnailKeywords {
		if strings.Contains(lowerURL, keyword) {
			return true
		}
	}
	// サイズ指定パターンをチェック（例: s128x128, m256x256など）
	sizePattern := regexp.MustCompile(`[sm]\d+x\d+`)
	if sizePattern.MatchString(lowerURL) {
		return true
	}
	return false
}

// extractImageURLs はHTMLドキュメントから商品画像のURLを抽出します
func (s *yahooScraper) extractImageURLs(doc *goquery.Document) []string {
	var imageURLs []string
	seenURLs := make(map[string]bool)

	// 商品画像の一般的なセレクタを試行
	selectors := []string{
		".ProductImage img",
		".ProductImage__main img",
		".yaProductImage img",
		".productImage img",
		"[class*='ProductImage'] img",
		"[class*='productImage'] img",
		"[class*='Image'] img",
		"img[src*='auctions.c.yimg.jp']",
		"img[data-src*='auctions.c.yimg.jp']",
	}

	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, sel *goquery.Selection) {
			// src属性を優先、なければdata-src属性を試行
			url := sel.AttrOr("src", "")
			if url == "" {
				url = sel.AttrOr("data-src", "")
			}
			if url == "" {
				url = sel.AttrOr("data-lazy-src", "")
			}

			// URLを正規化（相対URLを絶対URLに変換）
			if url != "" {
				url = strings.TrimSpace(url)
				// 相対URLの場合は絶対URLに変換
				if strings.HasPrefix(url, "//") {
					url = "https:" + url
				} else if strings.HasPrefix(url, "/") {
					url = "https://auctions.yahoo.co.jp" + url
				}
				// サムネイル画像を除外
				if s.isThumbnailImage(url) {
					return
				}
				// 重複を避ける
				if !seenURLs[url] && url != "" {
					seenURLs[url] = true
					imageURLs = append(imageURLs, url)
				}
			}
		})
		if len(imageURLs) > 0 {
			break // 見つかったら終了
		}
	}

	// フォールバック: すべてのimgタグから商品画像らしいものを探す
	if len(imageURLs) == 0 {
		doc.Find("img").Each(func(i int, sel *goquery.Selection) {
			url := sel.AttrOr("src", "")
			if url == "" {
				url = sel.AttrOr("data-src", "")
			}
			if url == "" {
				url = sel.AttrOr("data-lazy-src", "")
			}

			// Yahooオークションの画像URLかどうかを確認
			if url != "" && (strings.Contains(url, "auctions.c.yimg.jp") || strings.Contains(url, "yahoo.co.jp")) {
				url = strings.TrimSpace(url)
				// 相対URLの場合は絶対URLに変換
				if strings.HasPrefix(url, "//") {
					url = "https:" + url
				} else if strings.HasPrefix(url, "/") {
					url = "https://auctions.yahoo.co.jp" + url
				}
				// ロゴやアイコンなどの不要な画像を除外
				if !strings.Contains(url, "logo") && !strings.Contains(url, "icon") && !strings.Contains(url, "avatar") {
					// サムネイル画像を除外
					if s.isThumbnailImage(url) {
						return
					}
					if !seenURLs[url] && url != "" {
						seenURLs[url] = true
						imageURLs = append(imageURLs, url)
					}
				}
			}
		})
	}

	return imageURLs
}

// extractAuctionInformation はHTMLドキュメントからオークション情報を抽出します
func (s *yahooScraper) extractAuctionInformation(doc *goquery.Document, auctionID string) *model.AuctionInformation {
	info := &model.AuctionInformation{
		AuctionID: auctionID,
	}

	// div#otherInfoの配下から情報を抽出
	otherInfo := doc.Find("#otherInfo")
	if otherInfo.Length() == 0 {
		// フォールバック: otherInfoというクラス名やid属性を持つ要素を探す
		otherInfo = doc.Find("[id*='otherInfo'], [class*='otherInfo'], [id*='OtherInfo'], [class*='OtherInfo']")
	}

	// 「その他の情報」テーブルから情報を抽出
	// div#otherInfo配下のテーブルの各行（tr）を走査して、ラベル（th）と値（td）を取得
	otherInfo.Find("table tr").Each(func(i int, tr *goquery.Selection) {
		th := tr.Find("th")
		td := tr.Find("td")

		if th.Length() == 0 || td.Length() == 0 {
			return
		}

		label := strings.TrimSpace(th.First().Text())
		value := strings.TrimSpace(td.First().Text())

		if label == "" || value == "" {
			return
		}

		// 開始時の価格（「開始時の価格」から取得）
		if strings.Contains(label, "開始時の価格") {
			info.StartPrice = parsePrice(value)
		}

		// 開始日時
		if strings.Contains(label, "開始日時") {
			if t, err := parseDateTime(value); err == nil {
				info.StartTime = t
			}
		}

		// 終了日時
		if strings.Contains(label, "終了日時") {
			if t, err := parseDateTime(value); err == nil {
				info.EndTime = t
			}
		}

		// 早期終了
		if strings.Contains(label, "早期終了") {
			info.EarlyEnd = strings.Contains(value, "あり")
		}

		// 自動延長
		if strings.Contains(label, "自動延長") {
			info.AutoExtension = strings.Contains(value, "あり")
		}

		// 返品の可否
		if strings.Contains(label, "返品の可否") {
			// returnableは「返品可」または「返品可能」が含まれているかで判定
			info.Returnable = strings.Contains(value, "返品可") || strings.Contains(value, "返品可能")
			// returnable_detailには文字列情報全体を代入
			info.ReturnableDetail = value
		}
	})

	// フォールバック: div#otherInfo配下から正規表現で抽出
	otherInfoText := otherInfo.Text()
	if info.StartPrice == 0 {
		pricePattern := regexp.MustCompile(`開始時の価格\s*([0-9,]+)\s*円`)
		matches := pricePattern.FindStringSubmatch(otherInfoText)
		if len(matches) > 1 {
			info.StartPrice = parsePrice(matches[0])
		}
	}

	if info.StartTime.IsZero() {
		startPattern := regexp.MustCompile(`開始日時\s*(\d{4}年\d{1,2}月\d{1,2}日[^終了]*\d{1,2}時\d{1,2}分)`)
		matches := startPattern.FindStringSubmatch(otherInfoText)
		if len(matches) > 1 {
			if t, err := parseDateTime(matches[1]); err == nil {
				info.StartTime = t
			}
		}
	}

	if info.EndTime.IsZero() {
		endPattern := regexp.MustCompile(`終了日時\s*(\d{4}年\d{1,2}月\d{1,2}日[^終了]*\d{1,2}時\d{1,2}分)`)
		matches := endPattern.FindStringSubmatch(otherInfoText)
		if len(matches) > 1 {
			if t, err := parseDateTime(matches[1]); err == nil {
				info.EndTime = t
			}
		}
	}

	if !info.EarlyEnd {
		info.EarlyEnd = regexp.MustCompile(`早期終了\s*あり`).MatchString(otherInfoText)
	}

	if !info.AutoExtension {
		info.AutoExtension = regexp.MustCompile(`自動延長\s*あり`).MatchString(otherInfoText)
	}

	if info.ReturnableDetail == "" {
		returnPattern := regexp.MustCompile(`返品の可否\s*([^\n]+)`)
		matches := returnPattern.FindStringSubmatch(otherInfoText)
		if len(matches) > 1 {
			value := strings.TrimSpace(matches[1])
			// returnableは「返品可」または「返品可能」が含まれているかで判定
			info.Returnable = strings.Contains(value, "返品可") || strings.Contains(value, "返品可能")
			// returnable_detailには文字列情報全体を代入
			info.ReturnableDetail = value
		}
	}

	return info
}

// extractDescriptionFromJSON はNext.jsのJSONデータから商品説明を抽出します
func (s *yahooScraper) extractDescriptionFromJSON(doc *goquery.Document) string {
	scriptContent := doc.Find("script#__NEXT_DATA__").Text()
	if scriptContent == "" {
		return ""
	}

	var data struct {
		Props struct {
			PageProps struct {
				InitialState struct {
					Item struct {
						Detail struct {
							Item struct {
								DescriptionHtml string `json:"descriptionHtml"`
							} `json:"item"`
						} `json:"detail"`
					} `json:"item"`
				} `json:"initialState"`
			} `json:"pageProps"`
		} `json:"props"`
	}

	if err := json.Unmarshal([]byte(scriptContent), &data); err != nil {
		return ""
	}

	return data.Props.PageProps.InitialState.Item.Detail.Item.DescriptionHtml
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
