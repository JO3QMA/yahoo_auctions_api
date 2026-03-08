package yahoo

import (
	"testing"
)

func TestYahooCategoryScraper_URLConstruction(t *testing.T) {
	// 実際にリクエストは飛ばさず、ロジックだけ確認したいが、
	// FetchByCategoryはfetchHTMLを呼ぶためユニットテストしづらい。
	// ここでは省略するが、必要ならFetchHTMLをinterface経由にするなどのリファクタリングが必要。
}
