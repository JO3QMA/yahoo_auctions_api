package yahoo

import (
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
)

func TestYahooCategoryScraper_extractCategoryItems(t *testing.T) {
	// テスト用HTMLスニペット
	html := `
<html>
<body>
	<div class="Result__header">
		<div class="SearchMode">
			<div class="Tab">
				<ul>
					<li class="Tab__item Tab__item--current">
						<div>
							<span class="Tab__subText">1,234件</span>
						</div>
					</li>
				</ul>
			</div>
		</div>
	</div>

	<div class="Products__list">
		<ul class="Products__items">
			<li class="Product">
				<div class="Product__detail">
					<h3 class="Product__title">
						<a href="#" class="Product__titleLink" data-auction-id="a123456789">Test Item 1</a>
					</h3>
				</div>
				<div class="Product__priceInfo">
					<span class="Product__price">
						<span class="Product__priceValue">1,000円</span>
					</span>
					<span class="Product__price">
						<span class="Product__priceValue">2,000円</span>
					</span>
				</div>
				<dd class="Product__bid">5</dd>
				<img class="Product__imageData" src="http://example.com/img1.jpg">
			</li>
			<li class="Product">
				<div class="Product__detail">
					<h3 class="Product__title">
						<a href="#" class="Product__titleLink" data-auction-id="b987654321">Test Item 2</a>
					</h3>
				</div>
				<div class="Product__priceInfo">
					<span class="Product__price">
						<span class="Product__priceValue">500円</span>
					</span>
					<!-- 即決なし -->
				</div>
				<dd class="Product__bid">0</dd>
				<img class="Product__imageData" data-src="http://example.com/img2.jpg">
			</li>
		</ul>
	</div>
</body>
</html>
`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("failed to parse html: %v", err)
	}

	scraper := &yahooCategoryScraper{}
	page, err := scraper.extractCategoryItems(doc)
	if err != nil {
		t.Fatalf("extractCategoryItems failed: %v", err)
	}

	// TotalCount チェック
	if page.TotalCount != 1234 {
		t.Errorf("TotalCount got %d, want 1234", page.TotalCount)
	}

	// Items チェック
	if len(page.Items) != 2 {
		t.Fatalf("Items len got %d, want 2", len(page.Items))
	}

	// Item 1
	item1 := page.Items[0]
	if item1.AuctionID != "a123456789" {
		t.Errorf("Item1 AuctionID got %s, want a123456789", item1.AuctionID)
	}
	if item1.Title != "Test Item 1" {
		t.Errorf("Item1 Title got %s, want Test Item 1", item1.Title)
	}
	if item1.CurrentPrice != 1000 {
		t.Errorf("Item1 CurrentPrice got %d, want 1000", item1.CurrentPrice)
	}
	if item1.ImmediatePrice != 2000 {
		t.Errorf("Item1 ImmediatePrice got %d, want 2000", item1.ImmediatePrice)
	}
	if item1.BidCount != 5 {
		t.Errorf("Item1 BidCount got %d, want 5", item1.BidCount)
	}
	if item1.Image != "http://example.com/img1.jpg" {
		t.Errorf("Item1 Image got %s, want http://example.com/img1.jpg", item1.Image)
	}

	// Item 2
	item2 := page.Items[1]
	if item2.AuctionID != "b987654321" {
		t.Errorf("Item2 AuctionID got %s, want b987654321", item2.AuctionID)
	}
	if item2.CurrentPrice != 500 {
		t.Errorf("Item2 CurrentPrice got %d, want 500", item2.CurrentPrice)
	}
	if item2.ImmediatePrice != 0 {
		t.Errorf("Item2 ImmediatePrice got %d, want 0", item2.ImmediatePrice)
	}
	if item2.Image != "http://example.com/img2.jpg" {
		t.Errorf("Item2 Image got %s, want http://example.com/img2.jpg", item2.Image)
	}
}

func TestYahooCategoryScraper_URLConstruction(t *testing.T) {
	// 実際にリクエストは飛ばさず、ロジックだけ確認したいが、
	// FetchByCategoryはfetchHTMLを呼ぶためユニットテストしづらい。
	// ここでは省略するが、必要ならFetchHTMLをinterface経由にするなどのリファクタリングが必要。
}
