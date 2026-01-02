package yahoo

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
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
		return nil, fmt.Errorf("failed to fetch page: status %d", res.StatusCode)
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
	val, _ := strconv.ParseInt(valStr, 10, 64)
	return val
}

// parseCount は "1,000件" などの文字列から数値を抽出します
func parseCount(s string) int64 {
	return parsePrice(s) // 実装は同じでOK
}
