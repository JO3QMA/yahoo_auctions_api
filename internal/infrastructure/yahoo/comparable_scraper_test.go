package yahoo

import (
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
)

func TestExtractComparablePage(t *testing.T) {
	t.Parallel()

	endedAt := time.Date(2026, 6, 20, 12, 0, 0, 0, time.FixedZone("JST", 9*60*60))
	html := `
<html>
<body>
<script id="__NEXT_DATA__" type="application/json">` + buildClosedSearchNextData(endedAt) + `</script>
</body>
</html>
`

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		t.Fatalf("failed to parse html: %v", err)
	}

	page, err := extractComparablePage(doc)
	if err != nil {
		t.Fatalf("extractComparablePage failed: %v", err)
	}

	if len(page.Items) != 1 {
		t.Fatalf("Items len got %d, want 1", len(page.Items))
	}
	if page.Items[0].AuctionID != "p1111111111" {
		t.Fatalf("AuctionID got %q, want p1111111111", page.Items[0].AuctionID)
	}
	if page.Items[0].WinningPrice != 12000 {
		t.Fatalf("WinningPrice got %d, want 12000", page.Items[0].WinningPrice)
	}
	if !page.Items[0].EndedAt.Equal(endedAt) {
		t.Fatalf("EndedAt got %v, want %v", page.Items[0].EndedAt, endedAt)
	}
	if page.HasNext {
		t.Fatal("HasNext got true, want false")
	}
}

func buildClosedSearchNextData(endedAt time.Time) string {
	return `{
  "props": {
    "pageProps": {
      "initialState": {
        "search": {
          "items": {
            "listing": {
              "items": [
                {
                  "auctionId": "p1111111111",
                  "title": "GTX 1080",
                  "price": 12000,
                  "endTime": "` + endedAt.Format(time.RFC3339) + `",
                  "bidCount": 5
                },
                {
                  "auctionId": "p2222222222",
                  "title": "No bids",
                  "price": 1000,
                  "endTime": "` + endedAt.Format(time.RFC3339) + `",
                  "bidCount": 0
                }
              ]
            }
          }
        }
      }
    }
  }
}`
}
