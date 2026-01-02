package model

// CategoryItem はカテゴリ一覧で取得される商品のドメインモデルです
// 詳細情報（Item）よりも軽量な情報のみを持ちます
type CategoryItem struct {
	AuctionID      string
	Title          string
	CurrentPrice   int64  // 現在価格（単位：円）
	ImmediatePrice int64  // 即決価格（単位：円）。ない場合は0
	BidCount       int64  // 入札数
	Image          string // 商品画像のURL（一覧用サムネイルなど）
}

// CategoryItemsPage はカテゴリ商品一覧のページネーション結果を表します
type CategoryItemsPage struct {
	Items      []*CategoryItem
	TotalCount int64 // 商品の総数
	HasNext    bool  // 次のページがあるかどうか（簡易判定用）
}
